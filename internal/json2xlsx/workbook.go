package json2xlsx

// normalizeWorkbook は Workbook を内部正規化形式に変換し，wb.Sheets と wb.Styles を常に正しい状態にする．
// Book ラッパー形式，単一シート形式，複数シート形式のいずれからでも統一的な Sheets + Styles 表現を得る．
// Book フィールドは Charts 参照のために保持される．
func normalizeWorkbook(wb *Workbook) {
	sheets, styles := flattenWorkbook(wb)
	wb.Sheets = sheets
	wb.Styles = styles
}

// forEachCell は Workbook の全シートの全セルに対して fn を適用する．
// normalizeWorkbook 後に呼び出されることを前提とする．
func forEachCell(wb *Workbook, fn func(axis string, cell Cell) Cell) {
	for i := range wb.Sheets {
		for axis, cell := range wb.Sheets[i].Cells {
			wb.Sheets[i].Cells[axis] = fn(axis, cell)
		}
	}
}

// flattenWorkbook は Workbook の 3 形式（単一シート / 複数シート / book ラッパー）を
// 正規化し、シート一覧とスタイル一覧を返す．
func flattenWorkbook(wb *Workbook) (sheets []Sheet, styles []Style) {
	styles = wb.Styles
	switch {
	case wb.Book != nil:
		for name, sh := range wb.Book.Sheets {
			sh.Name = name
			sheets = append(sheets, sh)
		}
		if len(wb.Book.Styles) > 0 {
			styles = wb.Book.Styles
		}
	case len(wb.Sheets) > 0:
		sheets = wb.Sheets
	default:
		sheets = []Sheet{{
			Name:               wb.Name,
			Cells:              wb.Cells,
			Rows:               wb.Rows,
			Cols:               wb.Cols,
			RowDims:            wb.RowDims,
			Merges:             wb.Merges,
			Freeze:             wb.Freeze,
			AutoFilter:         wb.AutoFilter,
			Tables:             wb.Tables,
			Sparklines:         wb.Sparklines,
			ConditionalFormats: wb.ConditionalFormats,
		}}
	}
	return
}
