package sheet2xlsx

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// HTMLOptions は HTML レンダリング設定。
type HTMLOptions struct {
	Mode MarkdownMode
}

// ToHTML は入力 (JSON Workbook または XLSX) を HTML <table> に変換して書き出す。
func ToHTML(r io.Reader, w io.Writer, opts HTMLOptions) error {
	if opts.Mode == "" {
		opts.Mode = MarkdownModeValue
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

	out := renderHTML(wb, opts.Mode)
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// renderHTML は Workbook を HTML 文字列にレンダリングする。
func renderHTML(wb Workbook, mode MarkdownMode) string {
	var sheets []Sheet
	styles := wb.Styles
	if wb.Book != nil {
		for name, sh := range wb.Book.Sheets {
			sh.Name = name
			sheets = append(sheets, sh)
		}
		if len(wb.Book.Styles) > 0 {
			styles = wb.Book.Styles
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

	stylesByID := buildStyleMap(styles)

	var b strings.Builder
	for _, sh := range sheets {
		b.WriteString(renderSheetHTML(sh, stylesByID, mode))
	}
	return b.String()
}

// buildStyleMap は Styles 配列を id → Style のマップに変換する。
func buildStyleMap(styles []Style) map[int]Style {
	m := make(map[int]Style, len(styles))
	for _, s := range styles {
		m[s.ID] = s
	}
	return m
}

// borderCSSWidth は border style 名を CSS の太さ表現に変換する。
var borderCSSWidth = map[string]string{
	"thin":            "1px",
	"medium":          "2px",
	"thick":           "3px",
	"hair":            "1px",
	"dashed":          "",
	"dotted":          "",
	"double":          "",
	"mediumDashed":    "2px",
	"dashDot":         "",
	"mediumDashDot":   "2px",
	"dashDotDot":      "",
	"mediumDashDotDot": "2px",
	"slantDashDot":    "",
}

// borderCSSStyle は border style 名を CSS のスタイル表現に変換する。
var borderCSSStyle = map[string]string{
	"thin":            "solid",
	"medium":          "solid",
	"thick":           "solid",
	"hair":            "solid",
	"dashed":          "dashed",
	"dotted":          "dotted",
	"double":          "double",
	"mediumDashed":    "dashed",
	"dashDot":         "dashed",
	"mediumDashDot":   "dashed",
	"dashDotDot":      "dashed",
	"mediumDashDotDot": "dashed",
	"slantDashDot":    "dashed",
}

// borderSideCSS は border side 名を CSS プロパティ名に変換する。
var borderSideCSS = map[string]string{
	"left":   "border-left",
	"right":  "border-right",
	"top":    "border-top",
	"bottom": "border-bottom",
}

// styleToCSS は Style をインライン CSS 文字列に変換する。
func styleToCSS(s Style) string {
	var parts []string

	if s.Fill != nil && len(s.Fill.Color) > 0 {
		parts = append(parts, "background-color:"+s.Fill.Color[0])
	}
	if s.Font != nil {
		if s.Font.Bold {
			parts = append(parts, "font-weight:bold")
		}
		if s.Font.Italic {
			parts = append(parts, "font-style:italic")
		}
		if s.Font.Size > 0 {
			parts = append(parts, fmt.Sprintf("font-size:%.0fpt", s.Font.Size))
		}
		if s.Font.Color != "" {
			parts = append(parts, "color:"+s.Font.Color)
		}
	}
	if s.Alignment != nil {
		if s.Alignment.Horizontal != "" {
			parts = append(parts, "text-align:"+s.Alignment.Horizontal)
		}
		if s.Alignment.Vertical != "" {
			parts = append(parts, "vertical-align:"+s.Alignment.Vertical)
		}
		if s.Alignment.WrapText {
			parts = append(parts, "white-space:pre-wrap")
		}
	}
	for _, b := range s.Border {
		w := borderCSSWidth[b.Style]
		st := borderCSSStyle[b.Style]
		prop, ok := borderSideCSS[b.Side]
		if !ok {
			prop = "border" // Side="" means all sides
		}
		v := ""
		if w != "" {
			v = w + " " + st
		} else {
			v = st
		}
		if b.Color != "" {
			v += " " + b.Color
		}
		parts = append(parts, prop+":"+v)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";")
}

type mergeCell struct {
	Colspan int
	Rowspan int
}

// buildMergeMap は Merges から非表示セル集合とマージ情報マップを構築する。
func buildMergeMap(merges []Merge) (hidden map[string]bool, anchors map[string]mergeCell) {
	hidden = make(map[string]bool)
	anchors = make(map[string]mergeCell)
	for _, m := range merges {
		parts := strings.Split(m.Range, ":")
		if len(parts) != 2 {
			continue
		}
		sc, sr, err1 := excelize.CellNameToCoordinates(parts[0])
		ec, er, err2 := excelize.CellNameToCoordinates(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}
		colspan := ec - sc + 1
		rowspan := er - sr + 1
		anchors[parts[0]] = mergeCell{Colspan: colspan, Rowspan: rowspan}

		for r := sr; r <= er; r++ {
			for c := sc; c <= ec; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				if axis != parts[0] {
					hidden[axis] = true
				}
			}
		}
	}
	return
}

// renderSheetHTML は単一シートを <table> としてレンダリングする。
func renderSheetHTML(sh Sheet, stylesByID map[int]Style, mode MarkdownMode) string {
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

	colNames := make([]string, maxCol+1)
	for c := 1; c <= maxCol; c++ {
		name, _ := excelize.ColumnNumberToName(c)
		colNames[c] = name
	}

	hidden, anchors := buildMergeMap(sh.Merges)

	const thStyle = `style="font-weight:bold;border:1px solid #000"`

	var b strings.Builder
	withHeader := mode != MarkdownModeValue

	b.WriteString("<table>\n")
	if withHeader {
		b.WriteString("<tr>")
		b.WriteString("<th " + thStyle + ">")
		b.WriteString(htmlEsc(""))
		b.WriteString("</th>")
		for c := 1; c <= maxCol; c++ {
			b.WriteString("<th " + thStyle + ">")
			b.WriteString(htmlEsc(colNames[c]))
			b.WriteString("</th>")
		}
		b.WriteString("</tr>\n")
	}
	for r := 1; r <= maxRow; r++ {
		b.WriteString("<tr>")
		if withHeader {
			b.WriteString("<th " + thStyle + ">")
			b.WriteString(strconv.Itoa(r))
			b.WriteString("</th>")
		}
		for c := 1; c <= maxCol; c++ {
			axis := colNames[c] + strconv.Itoa(r)
			if hidden[axis] {
				continue
			}
			cell, ok := sh.Cells[axis]
			mi, isAnchor := anchors[axis]

			b.WriteString("<td")
			if isAnchor {
				if mi.Colspan > 1 {
					b.WriteString(` colspan="`)
					b.WriteString(strconv.Itoa(mi.Colspan))
					b.WriteString(`"`)
				}
				if mi.Rowspan > 1 {
					b.WriteString(` rowspan="`)
					b.WriteString(strconv.Itoa(mi.Rowspan))
					b.WriteString(`"`)
				}
			}
			if ok && cell.S > 0 {
				if st, found := stylesByID[cell.S]; found {
					if css := styleToCSS(st); css != "" {
						b.WriteString(` style="`)
						b.WriteString(htmlEsc(css))
						b.WriteString(`"`)
					}
				}
			}
			b.WriteString(">")
			if ok {
				if withHeader {
					b.WriteString(formatCellHTMLMode(cell, mode))
				} else {
					b.WriteString(formatCellHTML(cell))
				}
			}
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String()
}

// htmlEsc は style 属性値として安全な文字列にエスケープする。
func htmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// formatCellHTML は 1 セルの HTML 表現を返す。
// 値優先 (v) でフォーマットし、HTML エスケープを行う。
func formatCellHTML(cell Cell) string {
	vStr := scalarToString(cell.V)
	hasV := cell.V != nil && vStr != ""
	hasF := cell.F != ""

	var raw string
	switch cell.T {
	case "f":
		if hasV {
			raw = vStr
		} else if hasF {
			raw = "=" + cell.F
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
			raw = "=" + cell.F
		}
	}

	raw = strings.ReplaceAll(raw, "&", "&amp;")
	raw = strings.ReplaceAll(raw, "<", "&lt;")
	raw = strings.ReplaceAll(raw, ">", "&gt;")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	raw = strings.ReplaceAll(raw, "\n", "<br />")
	return raw
}

// formatCellHTMLMode は mode に応じてセル値を HTML 表現に変換する。
func formatCellHTMLMode(cell Cell, mode MarkdownMode) string {
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
		if hasV {
			raw = vStr
		} else if hasF {
			raw = "=" + cell.F
		}
	}

	raw = strings.ReplaceAll(raw, "<br />", "\x00")
	raw = strings.ReplaceAll(raw, "&", "&amp;")
	raw = strings.ReplaceAll(raw, "<", "&lt;")
	raw = strings.ReplaceAll(raw, ">", "&gt;")
	raw = strings.ReplaceAll(raw, "\x00", "<br />")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	raw = strings.ReplaceAll(raw, "\n", "<br />")
	return raw
}
