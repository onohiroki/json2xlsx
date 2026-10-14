package json2xlsx

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// CellRange は A1:F20 形式の矩形範囲 (座標は 1 始まり)．
type CellRange struct {
	MinCol int
	MinRow int
	MaxCol int
	MaxRow int
}

// ParseCellRange は "A1:F20" または単一セル "A1" を解析する．
// 小文字も受け付ける．順序が逆 (F20:A1) の場合は正規化する．
func ParseCellRange(s string) (CellRange, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return CellRange{}, fmt.Errorf("empty range")
	}
	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		return CellRange{}, fmt.Errorf("invalid range %q: expected A1:F20 or A1", raw)
	}

	minC, minR, err := excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return CellRange{}, fmt.Errorf("invalid range %q: %w", raw, err)
	}
	maxC, maxR := minC, minR
	if len(parts) == 2 {
		maxC, maxR, err = excelize.CellNameToCoordinates(parts[1])
		if err != nil {
			return CellRange{}, fmt.Errorf("invalid range %q: %w", raw, err)
		}
	}

	if minC > maxC {
		minC, maxC = maxC, minC
	}
	if minR > maxR {
		minR, maxR = maxR, minR
	}
	return CellRange{MinCol: minC, MinRow: minR, MaxCol: maxC, MaxRow: maxR}, nil
}

// BuildCellGridRange は Sheet をグリッド化し，rng があれば切り出す．
// rng が nil の場合は BuildCellGrid と同じ結果を返す．
// 範囲が実データと交差しない場合は ok=false とエラーを返す．
func BuildCellGridRange(sh Sheet, rng *CellRange) (CellGrid, bool, error) {
	full, ok := BuildCellGrid(sh)
	if !ok {
		return CellGrid{}, false, nil
	}
	if rng == nil {
		return full, true, nil
	}

	minC := rng.MinCol
	if minC < 1 {
		minC = 1
	}
	minR := rng.MinRow
	if minR < 1 {
		minR = 1
	}
	maxC := rng.MaxCol
	if maxC > full.MaxCol {
		maxC = full.MaxCol
	}
	maxR := rng.MaxRow
	if maxR > full.MaxRow {
		maxR = full.MaxRow
	}
	if minC > maxC || minR > maxR {
		return CellGrid{}, false, fmt.Errorf("range does not intersect the sheet data (used range is %dx%d)", full.MaxRow, full.MaxCol)
	}

	rows := make([][]Cell, maxR-minR+2)
	for r := minR; r <= maxR; r++ {
		row := make([]Cell, maxC-minC+2)
		for c := minC; c <= maxC; c++ {
			row[c-minC+1] = full.Rows[r][c]
		}
		rows[r-minR+1] = row
	}

	colNames := make([]string, maxC-minC+2)
	for c := minC; c <= maxC; c++ {
		name, err := excelize.ColumnNumberToName(c)
		if err != nil {
			return CellGrid{}, false, err
		}
		colNames[c-minC+1] = name
	}

	return CellGrid{
		Rows:      rows,
		MaxCol:    maxC - minC + 1,
		MaxRow:    maxR - minR + 1,
		ColNames:  colNames,
		RowOffset: minR - 1,
		ColOffset: minC - 1,
	}, true, nil
}
