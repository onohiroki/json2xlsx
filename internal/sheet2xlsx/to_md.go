package sheet2xlsx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// MarkdownMode はセル値の出力方針を表す。
type MarkdownMode string

const (
	// MarkdownModeFormula: 数式があれば数式、なければ v。
	MarkdownModeFormula MarkdownMode = "f"
	// MarkdownModeValue: v 優先、なければ数式。
	MarkdownModeValue MarkdownMode = "v"
	// MarkdownModeBoth: v と数式が両方ある場合 "v<br />=f"。
	MarkdownModeBoth MarkdownMode = "both"
)

// MarkdownOptions は Markdown レンダリング設定。
//
// デフォルト (FirstRowHeader=false, RowIndex=true): A/B/C 列名ヘッダ + 行番号列を表示する。
// --first-row-header 相当 (FirstRowHeader=true, RowIndex=false): 1行目をテーブルヘッダとして扱う。
type MarkdownOptions struct {
	Mode            MarkdownMode
	RowIndex        bool
	FirstRowHeader  bool
}

// ToMarkdown は入力 (JSON Workbook または XLSX) を Markdown テーブルに変換して書き出す。
// 入力種別は先頭 4 バイトの magic byte (PK\x03\x04 → XLSX) で判定する。
func ToMarkdown(r io.Reader, w io.Writer, opts MarkdownOptions) error {
	if opts.Mode == "" {
		opts.Mode = MarkdownModeFormula
	}
	switch opts.Mode {
	case MarkdownModeFormula, MarkdownModeValue, MarkdownModeBoth:
	default:
		return fmt.Errorf("invalid mode: %q (expected f|v|both)", opts.Mode)
	}

	br := bufio.NewReader(r)
	head, err := br.Peek(4)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var wb Workbook
	if bytes.Equal(head, []byte{'P', 'K', 0x03, 0x04}) {
		data, err := io.ReadAll(br)
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("open xlsx: %w", err)
		}
		defer f.Close()
		wb, err = extractWorkbookWithOptions(f, ToJSONOptions{DateMode: DateModeDisplay})
		if err != nil {
			return err
		}
	} else {
		data, err := io.ReadAll(br)
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		if err := json.Unmarshal(data, &wb); err != nil {
			if schemaErr := ValidateJSON(data); schemaErr != nil {
				return fmt.Errorf("%v\n\n%v", err, schemaErr)
			}
			return fmt.Errorf("unsupported input: expected JSON Workbook or XLSX: %w", err)
		}
		normalizeDateCells(&wb)
	}

	out := renderMarkdown(wb, opts)
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// normalizeDateCells は z に日付/時刻書式コードを持つセルの T を "d" に書き換える。
func normalizeDateCells(wb *Workbook) {
	for axis, cell := range wb.Cells {
		if cell.Z != "" && cell.T != "d" && cell.T != "f" {
			if isDateFormat(cell.Z, 0) {
				cell.T = "d"
				wb.Cells[axis] = cell
			}
		}
	}
	for i := range wb.Sheets {
		for axis, cell := range wb.Sheets[i].Cells {
			if cell.Z != "" && cell.T != "d" && cell.T != "f" {
				if isDateFormat(cell.Z, 0) {
					cell.T = "d"
					wb.Sheets[i].Cells[axis] = cell
				}
			}
		}
	}
	if wb.Book != nil {
		for name := range wb.Book.Sheets {
			sh := wb.Book.Sheets[name]
			for axis, cell := range sh.Cells {
				if cell.Z != "" && cell.T != "d" && cell.T != "f" {
					if isDateFormat(cell.Z, 0) {
						cell.T = "d"
						sh.Cells[axis] = cell
					}
				}
			}
			wb.Book.Sheets[name] = sh
		}
	}
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

	// No hours (e.g. "mm:ss") → total minutes:seconds
	totalMin := abs / 60
	sec := abs % 60
	if hasSeconds {
		return fmt.Sprintf("%02d:%02d", totalMin, sec)
	}
	return fmt.Sprintf("%02d:%02d", totalMin, sec)
}

// renderMarkdown は Workbook を Markdown 文字列にレンダリングする。
func renderMarkdown(wb Workbook, opts MarkdownOptions) string {
	var sheets []Sheet
	if wb.Book != nil {
		for name, sh := range wb.Book.Sheets {
			sh.Name = name
			sheets = append(sheets, sh)
		}
	} else if len(wb.Sheets) > 0 {
		sheets = wb.Sheets
	} else if wb.Cells != nil || wb.Name != "" || wb.Merges != nil {
		sheets = []Sheet{{
			Name:    wb.Name,
			Cells:   wb.Cells,
			Cols:    wb.Cols,
			RowDims: wb.RowDims,
			Merges:  wb.Merges,
		}}
	}

	var b strings.Builder
	multi := len(sheets) > 1
	for i, sh := range sheets {
		if multi {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString("## ")
			b.WriteString(sh.Name)
			b.WriteString("\n\n")
		}
		b.WriteString(renderSheet(sh, opts))
	}
	return b.String()
}

// renderSheet は単一シートをテーブルとしてレンダリングする。空シートは空文字を返す。
// FirstRowHeader=true のとき 1 行目をテーブルヘッダとして扱う（--first-row-header）。
// それ以外は A/B/C 列名ヘッダ + 行番号を表示する（デフォルト）。
func renderSheet(sh Sheet, opts MarkdownOptions) string {
	if len(sh.Cells) == 0 {
		return ""
	}
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
	if maxCol == 0 || maxRow == 0 {
		return ""
	}

	// ヘッダ列名のキャッシュ。
	colNames := make([]string, maxCol+1)
	for c := 1; c <= maxCol; c++ {
		name, _ := excelize.ColumnNumberToName(c)
		colNames[c] = name
	}

	var b strings.Builder

	if opts.FirstRowHeader {
		// 1行目をヘッダとして出力
		b.WriteString("|")
		for c := 1; c <= maxCol; c++ {
			axis := colNames[c] + "1"
			cell, ok := sh.Cells[axis]
			b.WriteString(" ")
			if ok {
				b.WriteString(formatCell(cell, opts.Mode))
			}
			b.WriteString(" |")
		}
		b.WriteString("\n")

		// 区切り行
		b.WriteString("|")
		for c := 1; c <= maxCol; c++ {
			b.WriteString(" --- |")
		}
		b.WriteString("\n")

		// 本文 (r=2..maxRow)
		for r := 2; r <= maxRow; r++ {
			b.WriteString("|")
			for c := 1; c <= maxCol; c++ {
				axis := colNames[c] + strconv.Itoa(r)
				cell, ok := sh.Cells[axis]
				b.WriteString(" ")
				if ok {
					b.WriteString(formatCell(cell, opts.Mode))
				}
				b.WriteString(" |")
			}
			b.WriteString("\n")
		}
	} else {
		// ヘッダ行 (ABC)
		b.WriteString("|")
		if opts.RowIndex {
			b.WriteString("   |")
		}
		for c := 1; c <= maxCol; c++ {
			b.WriteString(" ")
			b.WriteString(colNames[c])
			b.WriteString(" |")
		}
		b.WriteString("\n")

		// 区切り行
		b.WriteString("|")
		if opts.RowIndex {
			b.WriteString(" --- |")
		}
		for c := 1; c <= maxCol; c++ {
			b.WriteString(" --- |")
		}
		b.WriteString("\n")

		// 本文 (r=1..maxRow)
		for r := 1; r <= maxRow; r++ {
			b.WriteString("|")
			if opts.RowIndex {
				b.WriteString(" ")
				b.WriteString(strconv.Itoa(r))
				b.WriteString(" |")
			}
			for c := 1; c <= maxCol; c++ {
				axis := colNames[c] + strconv.Itoa(r)
				cell, ok := sh.Cells[axis]
				b.WriteString(" ")
				if ok {
					b.WriteString(formatCell(cell, opts.Mode))
				}
				b.WriteString(" |")
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatCell は 1 セルの Markdown 表現を返す。
func formatCell(cell Cell, mode MarkdownMode) string {
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
			}
		case MarkdownModeBoth:
			if hasV && hasF {
				raw = vStr + "<br />=" + cell.F
			} else if hasF {
				raw = "=" + cell.F
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
		// t = s/n/b/空 はすべて v を文字列化。
		if hasV {
			raw = vStr
		} else if hasF {
			// 型未指定だが数式がある場合
			if mode == MarkdownModeValue {
				raw = "=" + cell.F
			} else {
				raw = "=" + cell.F
			}
		}
	}

	return escapeMarkdownCell(raw)
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
		// 整数なら小数点なしで表示。
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

// escapeMarkdownCell は Markdown テーブルセル向けのエスケープを行う。
func escapeMarkdownCell(s string) string {
	if s == "" {
		return ""
	}
	// バックスラッシュを先に処理。
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "<br />")
	return s
}
