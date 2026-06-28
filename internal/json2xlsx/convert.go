package json2xlsx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// WarningError は非 fatal な警告を表す。
// 処理は継続され、XLSX 出力は行われるが、exit code は非零になる。
type WarningError struct {
	Err error
}

func (w *WarningError) Error() string { return w.Err.Error() }
func (w *WarningError) Unwrap() error { return w.Err }

// ConvertOptions は Convert の動作オプション。
type ConvertOptions struct {
	// DataJSON が true の場合、入力を「データ JSON」として扱い、
	// 二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する。
	// false (デフォルト) の場合は SheetJS 形式のみを受け付け、失敗したらエラーを返す。
	DataJSON bool
}

// Convert は JSON を読み込み、XLSX を out に書き出す。
func Convert(r io.Reader, out io.Writer, opts ConvertOptions) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var wb *Workbook
	if opts.DataJSON {
		wb, err = unmarshalDataJSON(data)
	} else {
		wb, err = unmarshalSheetJS(data)
	}
	if err != nil {
		return err
	}

	if err := convertWorkbook(wb, out); err != nil {
		if !opts.DataJSON {
			if schemaErr := ValidateJSON(data); schemaErr != nil {
				return fmt.Errorf("%v\n\n%v", err, schemaErr)
			}
		}
		return err
	}
	return nil
}

// UnmarshalWorkbook は JSON データを Workbook 構造体にパースする。
// DataJSON=false の場合は SheetJS 形式のみ、DataJSON=true の場合は
// 二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する。
func UnmarshalWorkbook(data []byte, dataJSON bool) (*Workbook, error) {
	if dataJSON {
		return unmarshalDataJSON(data)
	}
	return unmarshalSheetJS(data)
}

// unmarshalSheetJS は SheetJS 形式のみを受け付ける。フォールバックなし。
func unmarshalSheetJS(data []byte) (*Workbook, error) {
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

// unmarshalDataJSON は二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する。
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
// 1行目がキーヘッダ、2行目以降が値の行データに変換する。
// 入力は生の JSON バイト列であり、各オブジェクトのキー順は JSON 宣言順を維持する。
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
// 1行目がキーヘッダ、2行目以降が値の行データに変換する。
// 値がすべて配列でない場合は false を返す。
func tryMapOfArrays(data []byte) (*Workbook, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false
	}

	// JSON での出現順を維持するため、Decoder で順次キーを取り出す。
	keys, err := orderedJSONObjectKeys(data)
	if err != nil || len(keys) != len(raw) {
		// フォールバック: 順序が取れない場合は map のイテレーション順を使う。
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

// decodeOrderedObject は 1 つの JSON オブジェクトをパースし、宣言順のキー配列と
// 対応する値のマップを返す。値の型は json.Unmarshal と同等
// (float64 / string / bool / nil / []any / map[string]any)。
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

// orderedJSONObjectKeys は JSON オブジェクトの最上位キーを出現順で返す。
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
		// 対応する値をスキップ
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func convertWorkbook(wb *Workbook, out io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	sheets, styles := flattenWorkbook(wb)

	if err := validateWorkbook(sheets, wb); err != nil {
		return err
	}

	styleMap, err := buildStyles(f, styles)
	if err != nil {
		return fmt.Errorf("build styles: %w", err)
	}

	var warnings int
	if err := createSheets(f, sheets, styleMap, styles, &warnings); err != nil {
		return err
	}

	if err := addChartsToFile(f, wb); err != nil {
		return err
	}

	if err := f.Write(out); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}

	if warnings > 0 {
		return fmt.Errorf("conversion completed with %d warning(s)", warnings)
	}
	return nil
}

func validateWorkbook(sheets []Sheet, wb *Workbook) error {
	if len(sheets) == 0 {
		hasCharts := wb.Book != nil && len(wb.Book.Charts) > 0
		if !hasCharts {
			return fmt.Errorf("no sheets found in JSON input: expected a \"sheets\" array, \"cells\" object, or a \"book\" wrapper with \"sheets\"")
		}
		return nil
	}
	hasData := false
	for _, sh := range sheets {
		if len(sh.Cells) > 0 || len(sh.Rows) > 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		hasCharts := wb.Book != nil && len(wb.Book.Charts) > 0
		if !hasCharts {
			return fmt.Errorf("no valid cell data found in JSON input: each sheet must contain a \"cells\" object (e.g. \"A1\": {...}) or a \"rows\" array")
		}
	}
	return nil
}

func createSheets(f *excelize.File, sheets []Sheet, styleMap map[int]int, styles []Style, warnings *int) error {
	defaultName := f.GetSheetName(0)
	firstAssigned := false

	for i, sh := range sheets {
		name := sh.Name
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}

		if !firstAssigned {
			if name != defaultName {
				if err := f.SetSheetName(defaultName, name); err != nil {
					return fmt.Errorf("rename sheet: %w", err)
				}
			}
			firstAssigned = true
		} else {
			if _, err := f.NewSheet(name); err != nil {
				return fmt.Errorf("new sheet %q: %w", name, err)
			}
		}

		if err := writeSheet(f, name, sh, styleMap, styles, warnings); err != nil {
			return fmt.Errorf("write sheet %q: %w", name, err)
		}
	}
	return nil
}

func addChartsToFile(f *excelize.File, wb *Workbook) error {
	if wb.Book == nil {
		return nil
	}

	helperSheet := "_xlsxchart_helper"
	helperRow := 1
	helperCreated := false
	for _, ch := range wb.Book.Charts {
		for i := range ch.Ser {
			name := ch.Ser[i].Name
			if name != "" && !strings.Contains(name, "!") {
				if !helperCreated {
					if _, err := f.NewSheet(helperSheet); err != nil {
						return fmt.Errorf("create helper sheet: %w", err)
					}
					if err := f.SetSheetVisible(helperSheet, false); err != nil {
						return fmt.Errorf("hide helper sheet: %w", err)
					}
					helperCreated = true
				}
				cell, _ := excelize.CoordinatesToCellName(1, helperRow)
				if err := f.SetCellValue(helperSheet, cell, name); err != nil {
					return fmt.Errorf("write series name to helper sheet: %w", err)
				}
				ch.Ser[i].Name = fmt.Sprintf("'%s'!%s", helperSheet, cell)
				helperRow++
			}
		}
	}

	for _, ch := range wb.Book.Charts {
		ct, err := chartTypeFromString(ch.Ct)
		if err != nil {
			return fmt.Errorf("chart %q: %w", ch.ID, err)
		}
		ec := excelize.Chart{
			Type:   ct,
			Series: toExcelizeSeriesList(ch.Ser),
		}
		if ch.Title != nil && ch.Title.Tx != "" {
			ec.Title = []excelize.RichTextRun{{Text: ch.Title.Tx}}
		}
		if ch.Legend != nil {
			ec.Legend = excelize.ChartLegend{
				Position: ch.Legend.Pos,
			}
		}
		if ch.XAxis != nil {
			ec.XAxis = toExcelizeAxis(*ch.XAxis)
		}
		if ch.YAxis != nil {
			ec.YAxis = toExcelizeAxis(*ch.YAxis)
		}
		if ch.Dim != nil {
			ec.Dimension = excelize.ChartDimension{
				Width:  uint(ch.Dim.W),
				Height: uint(ch.Dim.H),
			}
		}
		if ch.Plot != nil {
			ec.VaryColors = &ch.Plot.VaryColors
			ec.ShowBlanksAs = ch.Plot.ShowBlanksAs
		}
		for _, s := range ch.Ser {
			if s.DLbls != nil {
				ec.PlotArea = excelize.ChartPlotArea{
					ShowVal:         s.DLbls.ShowVal,
					ShowCatName:     s.DLbls.ShowCatName,
					ShowSerName:     s.DLbls.ShowSerName,
					ShowPercent:     s.DLbls.ShowPercent,
					ShowLeaderLines: s.DLbls.ShowLeaderLn,
				}
				break
			}
		}
		switch ch.Mode {
		case "", "embedded":
			ec.Format = chartGraphicOptions(ch.Dim)
			if err := f.AddChart(ch.Sheet, ch.Anchor, &ec); err != nil {
				return fmt.Errorf("chart %q: add chart: %w", ch.ID, err)
			}
		case "chartSheet":
			if err := f.AddChartSheet(ch.Sheet, &ec); err != nil {
				return fmt.Errorf("chart %q: add chart sheet: %w", ch.ID, err)
			}
		default:
			return fmt.Errorf("chart %q: unknown mode %q", ch.ID, ch.Mode)
		}
	}
	return nil
}

func writeSheet(f *excelize.File, name string, sh Sheet, styleMap map[int]int, styles []Style, warnings *int) error {
	// AoA 形式 (rows) の展開: 1 行目 = 1 行目に配置
	for r, row := range sh.Rows {
		for c, v := range row {
			axis, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(name, axis, v); err != nil {
				return err
			}
		}
	}

	// Cell Object 形式 (WarningError は非 fatal として継続)
	for axis, cell := range sh.Cells {
		if err := setCell(f, name, axis, cell, styleMap, styles); err != nil {
			var we *WarningError
			if errors.As(err, &we) {
				fmt.Fprintf(os.Stderr, "warning: %v\n", we.Err)
				*warnings++
			} else {
				return fmt.Errorf("set cell %s: %w", axis, err)
			}
		}
	}

	// 列幅 (Excel 制限: 0 < width <= 255)
	for _, c := range sh.Cols {
		if c.Col == "" || c.Width <= 0 {
			continue
		}
		w := c.Width
		if w > 255 {
			w = 255
		}
		if err := f.SetColWidth(name, c.Col, c.Col, w); err != nil {
			return err
		}
	}

	// 行高 (Excel 制限: 0 < height <= 409)
	for _, rd := range sh.RowDims {
		if rd.Row <= 0 || rd.Height <= 0 {
			continue
		}
		h := rd.Height
		if h > 409 {
			h = 409
		}
		if err := f.SetRowHeight(name, rd.Row, h); err != nil {
			return err
		}
	}

	// マージ
	for _, m := range sh.Merges {
		parts := strings.Split(m.Range, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid merge range: %q", m.Range)
		}
		if err := f.MergeCell(name, parts[0], parts[1]); err != nil {
			return err
		}
	}

	return nil
}

func setCell(f *excelize.File, sheet, axis string, c Cell, styleMap map[int]int, styles []Style) error {
	switch c.T {
	case "f":
		if c.F == "" {
			return fmt.Errorf("cell %s: type=f but formula empty", axis)
		}
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
		}
		if err := f.SetCellFormula(sheet, axis, c.F); err != nil {
			return err
		}
	case "b":
		bv, ok := c.V.(bool)
		if !ok {
			return fmt.Errorf("cell %s: type=b but value not bool", axis)
		}
		if err := f.SetCellBool(sheet, axis, bv); err != nil {
			return err
		}
	case "n":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	case "s", "":
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
		}
	case "d":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cell %s: unknown type %q", axis, c.T)
	}

	// ハイパーリンク
	if c.L != nil {
		target, tooltip := parseLink(c.L)
		if target != "" {
			opts := []excelize.HyperlinkOpts{}
			if tooltip != "" {
				opts = append(opts, excelize.HyperlinkOpts{Tooltip: &tooltip})
			}
			if err := f.SetCellHyperLink(sheet, axis, target, "External", opts...); err != nil {
				return err
			}
		}
	}

	// スタイル適用 (z 単独指定にも対応)
	if c.S != 0 {
		if baseIdx, ok := styleMap[c.S]; ok {
			styleID := baseIdx
			if c.Z != "" {
				var mergeErr error
				styleID, mergeErr = mergeStyleWithNumFmt(f, styles, c.S, c.Z)
				if mergeErr != nil {
					return mergeErr
				}
			}
			if err := f.SetCellStyle(sheet, axis, axis, styleID); err != nil {
				return err
			}
		} else if c.Z != "" {
			id, err := f.NewStyle(&excelize.Style{CustomNumFmt: &c.Z})
			if err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, axis, axis, id); err != nil {
				return err
			}
		} else {
			return &WarningError{Err: fmt.Errorf("cell %s: style id %d not defined in styles", axis, c.S)}
		}
	} else if c.Z != "" {
		id, err := f.NewStyle(&excelize.Style{CustomNumFmt: &c.Z})
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, axis, axis, id); err != nil {
			return err
		}
	}

	return nil
}

func mergeStyleWithNumFmt(f *excelize.File, styles []Style, styleID int, numFmt string) (int, error) {
	for i := range styles {
		if styles[i].ID == styleID {
			es, err := toExcelizeStyle(styles[i])
			if err != nil {
				return 0, err
			}
			es.CustomNumFmt = &numFmt
			return f.NewStyle(es)
		}
	}
	return f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
}

func parseLink(l interface{}) (target, tooltip string) {
	switch v := l.(type) {
	case string:
		return v, ""
	case map[string]interface{}:
		if t, ok := v["target"].(string); ok {
			target = t
		}
		if t, ok := v["tooltip"].(string); ok {
			tooltip = t
		}
	}
	return
}

// chartTypeFromString は chart-json-spec.md の ct 文字列を Excelize の ChartType に変換する。
func chartTypeFromString(ct string) (excelize.ChartType, error) {
	switch ct {
	case "col":
		return excelize.Col, nil
	case "bar":
		return excelize.Bar, nil
	case "line":
		return excelize.Line, nil
	case "area":
		return excelize.Area, nil
	case "pie":
		return excelize.Pie, nil
	case "doughnut":
		return excelize.Doughnut, nil
	case "scatter":
		return excelize.Scatter, nil
	case "radar":
		return excelize.Radar, nil
	default:
		return 0, fmt.Errorf("unsupported chart type %q", ct)
	}
}

// chartGraphicOptions は ChartDim から GraphicOptions を生成する。
func chartGraphicOptions(dim *ChartDim) excelize.GraphicOptions {
	opts := excelize.GraphicOptions{ScaleX: 1.0, ScaleY: 1.0}
	if dim != nil {
		if dim.Sx > 0 {
			opts.ScaleX = dim.Sx
		}
		if dim.Sy > 0 {
			opts.ScaleY = dim.Sy
		}
		opts.OffsetX = int(dim.OffX)
		opts.OffsetY = int(dim.OffY)
	}
	return opts
}

// toExcelizeSeriesList は ChartSeries のスライスを Excelize の ChartSeries スライスに変換する。
func toExcelizeSeriesList(series []ChartSeries) []excelize.ChartSeries {
	result := make([]excelize.ChartSeries, len(series))
	for i, s := range series {
		es := excelize.ChartSeries{
			Name:       s.Name,
			Categories: s.Cat,
			Values:     s.Val,
		}
		if s.Line != nil {
			es.Line = excelize.ChartLine{Width: s.Line.Width}
		}
		if s.Fill != nil && s.Fill.Color != "" {
			es.Fill = excelize.Fill{Color: []string{s.Fill.Color}}
		}
		if s.Marker != nil {
			es.Marker = excelize.ChartMarker{
				Symbol: s.Marker.Symbol,
				Size:   int(s.Marker.Size),
			}
		}
		result[i] = es
	}
	return result
}

// toExcelizeAxis は ChartAxis を Excelize の ChartAxis に変換する。
func toExcelizeAxis(axis ChartAxis) excelize.ChartAxis {
	ea := excelize.ChartAxis{}
	if axis.Title != "" {
		ea.Title = []excelize.RichTextRun{{Text: axis.Title}}
	}
	ea.ReverseOrder = axis.ReverseOrder
	ea.MajorGridLines = axis.MajorGridLines
	ea.MinorGridLines = axis.MinorGridLines
	if axis.NumFmt != "" {
		ea.NumFmt = excelize.ChartNumFmt{CustomNumFmt: axis.NumFmt}
	}
	if axis.Minimum != nil {
		ea.Minimum = axis.Minimum
	}
	if axis.Maximum != nil {
		ea.Maximum = axis.Maximum
	}
	if axis.MajorUnit != nil {
		ea.MajorUnit = *axis.MajorUnit
	}
	return ea
}
