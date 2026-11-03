package json2xlsx

import (
	"fmt"
	"sort"
	"strings"
)

// normalizeWorkbook は Workbook を内部正規化形式に変換し，wb.Sheets と wb.Styles を常に正しい状態にする．
// Book ラッパー形式，単一シート形式，複数シート形式のいずれからでも統一的な Sheets + Styles 表現を得る．
// Book フィールドは Charts 参照のために保持される．
func normalizeWorkbook(wb *Workbook) {
	sheets, styles := flattenWorkbook(wb)
	wb.Sheets = sheets
	wb.Styles = styles
	// t を省略した数式セルは f を失わないよう f 型として扱う．
	forEachCell(wb, func(_ string, cell Cell) Cell {
		if cell.T == "" && cell.F != "" {
			cell.T = "f"
		}
		return cell
	})
}

// validateSheetNames は実効シート名の重複を大文字小文字を無視して検証する．
// 空のシート名は Sheet1, Sheet2, ... として扱う．
func validateSheetNames(sheets []Sheet) error {
	seen := make(map[string]int, len(sheets))
	for i, sh := range sheets {
		name := effectiveSheetName(sh, i)
		key := strings.ToLower(name)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("duplicate sheet name %q (sheets %d and %d)", name, prev+1, i+1)
		}
		seen[key] = i
	}
	return nil
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
		seen := make(map[string]bool, len(wb.Book.Sheets))
		for _, name := range wb.Book.SheetOrder {
			sh, ok := wb.Book.Sheets[name]
			if !ok || seen[name] {
				continue
			}
			sh.Name = name
			sheets = append(sheets, sh)
			seen[name] = true
		}
		rest := make([]string, 0, len(wb.Book.Sheets))
		for name := range wb.Book.Sheets {
			if !seen[name] {
				rest = append(rest, name)
			}
		}
		sort.Strings(rest)
		for _, name := range rest {
			sh := wb.Book.Sheets[name]
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
			Pictures:           wb.Pictures,
			Background:         wb.Background,
		}}
	}
	return
}
