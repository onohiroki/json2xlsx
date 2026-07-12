package json2xlsx

import "github.com/xuri/excelize/v2"

// CellGrid はシートのセルを行列として保持する中間表現．
type CellGrid struct {
	Rows     [][]Cell // 1-indexed: Rows[r][c]
	MaxCol   int
	MaxRow   int
	ColNames []string // 1-indexed
}

// BuildCellGrid は Sheet の cells/rows から二次元行列を構築する．
// 空シートの場合は第二戻り値が false になる．
func BuildCellGrid(sh Sheet) (CellGrid, bool) {
	maxCol, maxRow := 0, 0
	if len(sh.Cells) > 0 {
		for axis := range sh.Cells {
			c, r, err := excelize.CellNameToCoordinates(axis)
			if err != nil {
				continue
			}
			if c > maxCol {
				maxCol = c
			}
			if r > maxRow {
				maxRow = r
			}
		}
	} else {
		maxRow = len(sh.Rows)
		for _, row := range sh.Rows {
			if len(row) > maxCol {
				maxCol = len(row)
			}
		}
	}
	if maxCol == 0 || maxRow == 0 {
		return CellGrid{}, false
	}

	rows := make([][]Cell, maxRow+1)
	for r := 1; r <= maxRow; r++ {
		rows[r] = make([]Cell, maxCol+1)
	}

	if len(sh.Cells) > 0 {
		for axis, cell := range sh.Cells {
			c, r, err := excelize.CellNameToCoordinates(axis)
			if err == nil && c <= maxCol && r <= maxRow {
				rows[r][c] = cell
			}
		}
	} else {
		for r, row := range sh.Rows {
			for c, val := range row {
				rows[r+1][c+1] = Cell{V: val}
			}
		}
	}

	colNames := make([]string, maxCol+1)
	for c := 1; c <= maxCol; c++ {
		name, _ := excelize.ColumnNumberToName(c)
		colNames[c] = name
	}

	return CellGrid{
		Rows:     rows,
		MaxCol:   maxCol,
		MaxRow:   maxRow,
		ColNames: colNames,
	}, true
}

// cellGridToCSVRows は CellGrid を CSV 出力用の [][]string に変換する．
// 値を持たず数式のみのセルがあった場合 hasWarning が true になる．
func cellGridToCSVRows(cg CellGrid, hasWarning *bool) [][]string {
	if cg.MaxRow == 0 || cg.MaxCol == 0 {
		return nil
	}
	grid := make([][]string, cg.MaxRow)
	for r := 1; r <= cg.MaxRow; r++ {
		row := make([]string, cg.MaxCol)
		for c := 1; c <= cg.MaxCol; c++ {
			cell := cg.Rows[r][c]
			if cell.V != nil {
				if cell.T == "d" {
					var w bool
					row[c-1] = CellDisplayValue(cell, MarkdownModeFormula, &w)
				} else {
					row[c-1] = scalarToString(cell.V)
				}
			} else if cell.F != "" {
				if hasWarning != nil {
					*hasWarning = true
				}
			}
		}
		grid[r-1] = row
	}
	return grid
}
