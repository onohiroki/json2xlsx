package json2xlsx

import "github.com/xuri/excelize/v2"

// CellGrid はシートのセルを行列として保持する中間表現．
type CellGrid struct {
	Rows     [][]Cell // 1-indexed: Rows[r][c]
	MaxCol   int
	MaxRow   int
	ColNames []string // 1-indexed
	// RowOffset / ColOffset は --range で切り出したときの原点補正．
	// 切り出しなしの場合は 0．絶対座標は RowOffset+r / ColOffset+c．
	RowOffset int
	ColOffset int
}

// BuildCellGrid は Sheet の cells/rows から二次元行列を構築する．
// cells と rows の両方がある場合は rows を先に書き，cells で上書きする．
// 空シートの場合は第二戻り値が false になる．
func BuildCellGrid(sh Sheet) (CellGrid, bool) {
	maxCol, maxRow := 0, 0
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
	if len(sh.Rows) > maxRow {
		maxRow = len(sh.Rows)
	}
	for _, row := range sh.Rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}
	if maxCol == 0 || maxRow == 0 {
		return CellGrid{}, false
	}

	rows := make([][]Cell, maxRow+1)
	for r := 1; r <= maxRow; r++ {
		rows[r] = make([]Cell, maxCol+1)
	}

	for r, row := range sh.Rows {
		for c, val := range row {
			if c+1 > maxCol {
				continue
			}
			rows[r+1][c+1] = Cell{V: val}
		}
	}
	for axis, cell := range sh.Cells {
		c, r, err := excelize.CellNameToCoordinates(axis)
		if err == nil && c <= maxCol && r <= maxRow {
			rows[r][c] = cell
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

// mergedSheetCells は rows と cells を 1 つのセルマップにまとめる．
// cells が優先される．数式エンジンの入力ように使う．
func mergedSheetCells(sh Sheet) map[string]Cell {
	cells := make(map[string]Cell, len(sh.Cells))
	for r, row := range sh.Rows {
		for c, val := range row {
			if val == nil {
				continue
			}
			axis, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				continue
			}
			cells[axis] = Cell{V: val}
		}
	}
	for axis, cell := range sh.Cells {
		cells[axis] = cell
	}
	return cells
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
