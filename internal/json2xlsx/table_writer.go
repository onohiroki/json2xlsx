package json2xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// parseAutoFilter は autoFilter フィールドをパースして範囲文字列を返す．
// 文字列形式 ("A1:C10") とオブジェクト形式 ({"ref": "A1:C10"}) の両方をサポートする．
func parseAutoFilter(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}
	switch a := v.(type) {
	case string:
		return a, nil
	case map[string]interface{}:
		ref, ok := a["ref"].(string)
		if !ok || ref == "" {
			return "", fmt.Errorf("autoFilter object must have a non-empty \"ref\" field")
		}
		return ref, nil
	default:
		return "", fmt.Errorf("autoFilter must be a string (e.g. \"A1:C10\") or an object with \"ref\" field")
	}
}

// toExcelizeTable は Table を excelize.Table に変換する．
func toExcelizeTable(t Table) *excelize.Table {
	xlTable := &excelize.Table{
		Name:              t.Name,
		StyleName:         t.StyleName,
		Range:             t.Range,
		ShowColumnStripes: t.BandedColumns,
		ShowFirstColumn:   t.FirstColumn,
		ShowLastColumn:    t.LastColumn,
	}
	if t.BandedRows != nil {
		xlTable.ShowRowStripes = t.BandedRows
	}
	if t.HeaderRow != nil {
		xlTable.ShowHeaderRow = t.HeaderRow
	}
	return xlTable
}

// isCoveredByTable は ref で指定された範囲が tables のいずれかのテーブル範囲に
// 包含されるかを判定する．同一範囲の場合に真を返す．
func isCoveredByTable(ref string, tables []Table) bool {
	for _, t := range tables {
		if t.Range == ref {
			return true
		}
	}
	return false
}
