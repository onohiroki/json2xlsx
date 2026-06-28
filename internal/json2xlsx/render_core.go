package json2xlsx

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// CellGrid はシートのセルを行列として保持する中間表現。
type CellGrid struct {
	Rows     [][]Cell // 1-indexed: Rows[r][c]
	MaxCol   int
	MaxRow   int
	ColNames []string // 1-indexed
}

// BuildCellGrid は Sheet の cells/rows から二次元行列を構築する。
// 空シートの場合は第二戻り値が false になる。
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

// CellDisplayValue は cell の表示文字列 (エスケープ前) を mode に従って返す。
func CellDisplayValue(cell Cell, mode MarkdownMode, hasWarning *bool) string {
	vStr := scalarToString(cell.V)
	hasV := cell.V != nil && vStr != ""
	hasF := cell.F != ""

	var raw string
	switch cell.T {
	case "f":
		switch mode {
		case MarkdownModeValue:
			if hasV {
				raw = vStr
			} else if hasF {
				raw = "=" + cell.F
				*hasWarning = true
			}
		case MarkdownModeBoth:
			if hasV && hasF {
				raw = vStr + "<br />=" + cell.F
			} else if hasF {
				raw = "=" + cell.F
				*hasWarning = true
			} else if hasV {
				raw = vStr
			}
		default: // MarkdownModeFormula
			if hasF {
				raw = "=" + cell.F
			} else if hasV {
				raw = vStr
			}
		}
	case "d":
		if hasV {
			if cell.Z != "" && isTimeOnlyFormat(cell.Z) {
				raw = formatTimeOnly(toFloat64(cell.V), cell.Z)
			} else {
				raw = dateCellToString(cell.V)
			}
		}
	default:
		if hasV {
			raw = vStr
		} else if hasF {
			if mode == MarkdownModeValue || mode == MarkdownModeBoth {
				raw = "=" + cell.F
				*hasWarning = true
			} else {
				raw = "=" + cell.F
			}
		}
	}

	return raw
}

// scalarToString は Cell.V (interface{}) を文字列化する。
func scalarToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case float32:
		return scalarToString(float64(x))
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case json.Number:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

// toFloat64 は interface{} から float64 を抽出する。失敗時は 0 を返す。
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, err := x.Float64()
		if err == nil {
			return f
		}
		return 0
	default:
		if s, ok := v.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
		return 0
	}
}

// dateCellToString は日付セルの V を文字列化する。
// 数値（シリアル値）の場合は RFC3339、文字列の場合はそのまま返す。
func dateCellToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	var serial float64
	switch x := v.(type) {
	case float64:
		serial = x
	case float32:
		serial = float64(x)
	case int:
		serial = float64(x)
	case int64:
		serial = float64(x)
	case json.Number:
		if f, err := x.Float64(); err == nil {
			serial = f
		} else {
			return x.String()
		}
	default:
		return fmt.Sprint(v)
	}
	if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
		return t.UTC().Format("2006-01-02T15:04:05")
	}
	if serial == float64(int64(serial)) {
		return strconv.FormatInt(int64(serial), 10)
	}
	return strconv.FormatFloat(serial, 'g', -1, 64)
}

// isTimeOnlyFormat は書式コードが時刻のみ（日付コンポーネントなし）かどうかを判定する。
func isTimeOnlyFormat(code string) bool {
	if code == "" {
		return false
	}
	lc := strings.ToLower(code)
	hasTime := strings.Contains(lc, "h") || strings.Contains(lc, "mm") || strings.Contains(lc, "ss")
	hasDate := strings.Contains(lc, "y") || strings.Contains(lc, "d")
	return hasTime && !hasDate
}

// formatTimeOnly は時刻シリアル値を書式コードに従って文字列化する。
func formatTimeOnly(serial float64, z string) string {
	totalSec := int(math.Round(serial * 86400))
	abs := totalSec
	sign := ""
	if abs < 0 {
		sign = "-"
		abs = -abs
	}

	h := abs / 3600
	m := (abs % 3600) / 60
	s := abs % 60

	lc := strings.ToLower(z)
	hasSeconds := strings.Contains(lc, "ss")
	hourLeadZero := strings.Contains(lc, "hh")
	hasHours := strings.Contains(lc, "h") || strings.Contains(lc, "[h]")

	if hasHours {
		if hasSeconds {
			if hourLeadZero {
				return fmt.Sprintf("%s%02d:%02d:%02d", sign, h, m, s)
			}
			return fmt.Sprintf("%s%01d:%02d:%02d", sign, h, m, s)
		}
		if hourLeadZero {
			return fmt.Sprintf("%s%02d:%02d", sign, h, m)
		}
		return fmt.Sprintf("%s%01d:%02d", sign, h, m)
	}

	totalMin := abs / 60
	sec := abs % 60
	if hasSeconds {
		return fmt.Sprintf("%02d:%02d", totalMin, sec)
	}
	return fmt.Sprintf("%02d:%02d", totalMin, sec)
}

// normalizeDateCells は z に日付/時刻書式コードを持つセルの T を "d" に書き換える。
func normalizeDateCells(wb *Workbook) {
	forEachCell(wb, func(axis string, cell Cell) Cell {
		if cell.Z != "" && cell.T != "d" && cell.T != "f" {
			if isDateFormat(cell.Z, 0) {
				cell.T = "d"
			}
		}
		return cell
	})
}
