package json2xlsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// UnmarshalWorkbook は JSON データを Workbook 構造体にパースする．
// DataJSON=false の場合は SheetJS 形式のみ，DataJSON=true の場合は
// 二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する．
// 戻り値の Workbook は常に正規化済み（wb.Sheets / wb.Styles が populated）である．
func UnmarshalWorkbook(data []byte, dataJSON bool) (*Workbook, error) {
	var wb *Workbook
	var err error
	if dataJSON {
		wb, err = unmarshalDataJSON(data)
	} else {
		wb, err = unmarshalSheetJS(data)
	}
	if err != nil {
		return nil, err
	}
	normalizeWorkbook(wb)
	return wb, nil
}

// unmarshalSheetJS は SheetJS 形式のみを受け付ける．フォールバックなし．
func unmarshalSheetJS(data []byte) (*Workbook, error) {
	if normalized, ok := NormalizeSheetJSLibraryShape(data); ok {
		data = normalized
	}
	data = StripSheetJSInternalKeys(data)
	var wb Workbook
	if err := json.Unmarshal(data, &wb); err != nil {
		hint := "Hint: Use --data-json for 2D array, array of objects, or map-of-arrays format."
		if schemaErr := ValidateJSON(data); schemaErr != nil {
			return nil, fmt.Errorf("ERROR: %v\n\n%s\n\n%v", err, hint, schemaErr)
		}
		return nil, fmt.Errorf("ERROR: %v\n\n%s", err, hint)
	}
	return &wb, nil
}

// stripSheetJSInternalKeys は SheetJS の内部キー (!ref, !cols など) を再帰的に除去する．
func StripSheetJSInternalKeys(data []byte) []byte {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return data
	}
	out, err := json.Marshal(stripBangKeys(v))
	if err != nil {
		return data
	}
	return out
}

func stripBangKeys(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if strings.HasPrefix(k, "!") {
				continue
			}
			out[k] = stripBangKeys(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = stripBangKeys(val)
		}
		return out
	default:
		return v
	}
}

// normalizeSheetJSLibraryShape は SheetJS ライブラリの Workbook 形式
// ({"SheetNames": [...], "Sheets": {name: cells}}) を json2xlsx の
// sheets 配列形式に変換する．変換できない場合は元のデータを返す．
func NormalizeSheetJSLibraryShape(data []byte) ([]byte, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return data, false
	}
	sheetsRaw, ok := raw["Sheets"]
	if !ok {
		return data, false
	}
	var sheets map[string]json.RawMessage
	if err := json.Unmarshal(sheetsRaw, &sheets); err != nil {
		return data, false
	}

	var names []string
	if namesRaw, ok := raw["SheetNames"]; ok {
		if err := json.Unmarshal(namesRaw, &names); err != nil {
			return data, false
		}
	}

	type sheetOut struct {
		Name  string                     `json:"name"`
		Cells map[string]json.RawMessage `json:"cells"`
	}
	out := make([]sheetOut, 0, len(sheets))
	seen := map[string]bool{}
	add := func(name string, sheetRaw json.RawMessage) {
		if seen[name] {
			return
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(sheetRaw, &m); err != nil {
			return
		}
		var cells map[string]json.RawMessage
		if inner, ok := m["cells"]; ok {
			if err := json.Unmarshal(inner, &cells); err != nil {
				return
			}
		} else {
			cells = m
		}
		filtered := make(map[string]json.RawMessage, len(cells))
		for k, v := range cells {
			if strings.HasPrefix(k, "!") {
				continue
			}
			filtered[k] = v
		}
		out = append(out, sheetOut{Name: name, Cells: filtered})
		seen[name] = true
	}

	for _, name := range names {
		if sheetRaw, ok := sheets[name]; ok {
			add(name, sheetRaw)
		}
	}
	rest := make([]string, 0, len(sheets))
	for name := range sheets {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	for _, name := range rest {
		add(name, sheets[name])
	}
	if len(out) == 0 {
		return data, false
	}
	normalized, err := json.Marshal(map[string]any{"sheets": out})
	if err != nil {
		return data, false
	}
	return normalized, true
}

// unmarshalDataJSON は二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する．
func unmarshalDataJSON(data []byte) (*Workbook, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		// 1) 二次元配列形式 [[...], ...]
		var rows [][]any
		if err := json.Unmarshal(trimmed, &rows); err == nil {
			return &Workbook{Rows: rows}, nil
		}

		// 2) オブジェクト配列形式 [{...}, ...]
		var raws []json.RawMessage
		if err := json.Unmarshal(trimmed, &raws); err == nil && len(raws) > 0 {
			if wb, err := objectArrayToWorkbook(trimmed); err == nil {
				return wb, nil
			}
		}

		return nil, fmt.Errorf(
			"--data-json: input is a JSON array but does not appear to be a valid 2D data array: " +
				"expected [[...], ...] (array of arrays) or [{...}, ...] (array of objects)")
	}

	// 3) Map-of-Arrays 形式 {key: [...], ...}
	if wb, ok := tryMapOfArrays(data); ok {
		return wb, nil
	}

	return nil, fmt.Errorf("--data-json: expected array or map-of-arrays JSON")
}

// objectArrayToWorkbook はオブジェクト配列 [{key: val}, ...] を
// 1行目がキーヘッダ，2行目以降が値の行データに変換する．
// 入力は生の JSON バイト列であり，各オブジェクトのキー順は JSON 宣言順を維持する．
func objectArrayToWorkbook(data []byte) (*Workbook, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return nil, err
	}

	type orderedObj struct {
		keys   []string
		values map[string]any
	}
	objects := make([]orderedObj, 0, len(raws))
	keySet := make(map[string]bool)
	var allKeys []string
	for _, r := range raws {
		ks, vs, err := decodeOrderedObject(r)
		if err != nil {
			return nil, err
		}
		objects = append(objects, orderedObj{keys: ks, values: vs})
		for _, k := range ks {
			if !keySet[k] {
				keySet[k] = true
				allKeys = append(allKeys, k)
			}
		}
	}

	rows := make([][]any, 0, len(objects)+1)
	header := make([]any, len(allKeys))
	for i, k := range allKeys {
		header[i] = k
	}
	rows = append(rows, header)

	for _, obj := range objects {
		row := make([]any, len(allKeys))
		for i, k := range allKeys {
			row[i] = obj.values[k]
		}
		rows = append(rows, row)
	}

	return &Workbook{Rows: rows}, nil
}

// tryMapOfArrays は Map-of-Arrays 形式 {key: [val, ...], ...} を
// 1行目がキーヘッダ，2行目以降が値の行データに変換する．
// 値がすべて配列でない場合は false を返す．
func tryMapOfArrays(data []byte) (*Workbook, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}

	// JSON での出現順を維持するため，Decoder で順次キーを取り出す．
	keys, err := orderedJSONObjectKeys(data)
	if err != nil || len(keys) != len(raw) {
		keys = keys[:0]
		for k := range raw {
			keys = append(keys, k)
		}
	}

	arrays := make([][]any, len(keys))
	maxLen := 0
	for i, k := range keys {
		var arr []any
		if err := json.Unmarshal(raw[k], &arr); err != nil {
			return nil, false
		}
		if len(arr) > maxLen {
			maxLen = len(arr)
		}
		arrays[i] = arr
	}

	if maxLen == 0 {
		return nil, false
	}

	rows := make([][]any, 0, maxLen+1)
	header := make([]any, len(keys))
	for i, k := range keys {
		header[i] = k
	}
	rows = append(rows, header)

	for ri := 0; ri < maxLen; ri++ {
		row := make([]any, len(keys))
		for ci, arr := range arrays {
			if ri < len(arr) {
				row[ci] = arr[ri]
			}
		}
		rows = append(rows, row)
	}

	return &Workbook{Rows: rows}, true
}

// decodeOrderedObject は 1 つの JSON オブジェクトをパースし，宣言順のキー配列と
// 対応する値のマップを返す．値の型は json.Unmarshal と同等
// (float64 / string / bool / nil / []any / map[string]any)．
func decodeOrderedObject(data []byte) ([]string, map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, nil, fmt.Errorf("not a JSON object")
	}
	keys := []string{}
	values := map[string]any{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		k, ok := kt.(string)
		if !ok {
			return nil, nil, fmt.Errorf("unexpected key token")
		}
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, nil, err
		}
		keys = append(keys, k)
		values[k] = v
	}
	return keys, values, nil
}

// orderedJSONObjectKeys は JSON オブジェクトの最上位キーを出現順で返す．
func orderedJSONObjectKeys(data []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("not a JSON object")
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		k, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected key token")
		}
		keys = append(keys, k)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}
