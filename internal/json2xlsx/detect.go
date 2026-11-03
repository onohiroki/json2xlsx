package json2xlsx

import (
	"bytes"
	"encoding/json"
)

// DetectJSONFormat は JSON 入力が SheetJS / data / table のどれかを推定する．
// 形式が曖昧な場合は sheetjs を返す．明示的な指定が必要なケースでは，
// 呼び出し側の --data-json / --table-json が優先される．
func DetectJSONFormat(data []byte) string {
	trimmed := bytes.TrimSpace(trimBOM(data))
	if len(trimmed) == 0 {
		return "sheetjs"
	}
	switch trimmed[0] {
	case '[':
		return "data"
	case '{':
	default:
		return "sheetjs"
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &m); err != nil {
		return "sheetjs"
	}
	if jsonObjectHasAnyKey(m, "cells", "sheets", "book", "SheetNames", "Sheets") {
		return "sheetjs"
	}

	if tables, ok := m["tables"]; ok {
		if len(m) == 1 && isTableJSONTableArray(tables) {
			return "table"
		}
		// トップレベルの tables は SheetJS の構造化テーブルでも使われる．
		return "sheetjs"
	}

	if len(m) > 0 && !jsonObjectHasAnyKey(m, sheetJSArrayKeys...) && jsonObjectValuesAreArrays(m) {
		return "data"
	}
	return "sheetjs"
}

// sheetJSArrayKeys は Map-of-Arrays と紛らわしい SheetJS のトップレベルキー．
var sheetJSArrayKeys = []string{
	"rows",
	"cols",
	"rowDims",
	"merges",
	"freeze",
	"autoFilter",
	"sparklines",
	"conditionalFormats",
	"pictures",
	"background",
	"styles",
}

func jsonObjectHasAnyKey(m map[string]json.RawMessage, keys ...string) bool {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func jsonObjectValuesAreArrays(m map[string]json.RawMessage) bool {
	for _, raw := range m {
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil {
			return false
		}
	}
	return true
}

func isTableJSONTableArray(raw json.RawMessage) bool {
	var tables []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tables); err != nil || len(tables) == 0 {
		return false
	}
	for _, table := range tables {
		if !jsonObjectHasAnyKey(table, "rows", "header", "columns") {
			return false
		}
	}
	return true
}
