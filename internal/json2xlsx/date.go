package json2xlsx

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// DateMode は to-json の日時出力モード．
type DateMode string

const (
	// DateModeDisplay は Excel の表示文字列を出力する．
	DateModeDisplay DateMode = "display"
	// DateModeRFC3339 は Excel シリアル値を RFC3339 (UTC) に再解釈して出力する．
	DateModeRFC3339 DateMode = "rfc3339"
	// DateModeSerial は Excel シリアル値をそのまま数値として出力する．
	DateModeSerial DateMode = "serial"
)

// dateCellToString は日付セルの V を文字列化する．
// 数値（シリアル値）の場合は RFC3339、文字列の場合はそのまま返す．
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

// isTimeOnlyFormat は書式コードが時刻のみ（日付コンポーネントなし）かどうかを判定する．
func isTimeOnlyFormat(code string) bool {
	if code == "" {
		return false
	}
	lc := strings.ToLower(code)
	hasTime := strings.Contains(lc, "h") || strings.Contains(lc, "mm") || strings.Contains(lc, "ss")
	hasDate := strings.Contains(lc, "y") || strings.Contains(lc, "d")
	return hasTime && !hasDate
}

// formatTimeOnly は時刻シリアル値を書式コードに従って文字列化する．
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

// normalizeDateCells は z に日付/時刻書式コードを持つセルの T を "d" に書き換える．
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
