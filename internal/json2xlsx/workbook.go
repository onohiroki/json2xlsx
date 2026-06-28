package json2xlsx

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
			Name:    wb.Name,
			Cells:   wb.Cells,
			Rows:    wb.Rows,
			Cols:    wb.Cols,
			RowDims: wb.RowDims,
			Merges:  wb.Merges,
		}}
	}
	return
}
