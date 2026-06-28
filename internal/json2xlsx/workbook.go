package json2xlsx

// forEachCell は Workbook の 3 形式（単一シート / 複数シート / book ラッパー）を透過的に
// イテレーションし，全セルに対して fn を適用する．fn は axis と cell を受け取り，
// 変更後の cell を返す（変更がない場合はそのまま返す）．
func forEachCell(wb *Workbook, fn func(axis string, cell Cell) Cell) {
	for axis, cell := range wb.Cells {
		wb.Cells[axis] = fn(axis, cell)
	}
	for i := range wb.Sheets {
		for axis, cell := range wb.Sheets[i].Cells {
			wb.Sheets[i].Cells[axis] = fn(axis, cell)
		}
	}
	if wb.Book != nil {
		for name, sh := range wb.Book.Sheets {
			for axis, cell := range sh.Cells {
				sh.Cells[axis] = fn(axis, cell)
			}
			wb.Book.Sheets[name] = sh
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
