package json2xlsx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// HTMLOptions は HTML レンダリング設定。
type HTMLOptions struct {
	Mode         MarkdownMode
	GridLines    bool // セル間の隙間をなくし、枠線未指定セルにグレーの細枠線を表示する
	ExplicitMode bool
	DataJSON     bool
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

	res, err := ReadWorkbook(r, opts.DataJSON)
	if err != nil {
		return err
	}
	wb := res.Workbook

	var pendingWarnings []string
	if !res.IsXLSX && len(res.RawData) > 0 && bytes.TrimSpace(res.RawData)[0] == '[' {
		if opts.ExplicitMode && (opts.Mode == MarkdownModeFormula || opts.Mode == MarkdownModeBoth) {
			pendingWarnings = append(pendingWarnings, fmt.Sprintf("Warning: --mode=%s is ignored for JSON array input (formulas not supported in this format).", opts.Mode))
		}
	}

	out, hasWarning := renderHTML(*wb, opts)
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, msg := range pendingWarnings {
		fmt.Fprintln(os.Stderr, msg)
	}
	if hasWarning {
		if opts.Mode == MarkdownModeBoth {
			fmt.Fprintln(os.Stderr, "Warning: Missing values for some cells; showing only formulas.")
		} else {
			fmt.Fprintln(os.Stderr, "Warning: Missing values for some cells; showing formulas instead.")
		}
	}
	return nil
}

// renderHTML は Workbook を HTML 文字列にレンダリングする。
func renderHTML(wb Workbook, opts HTMLOptions) (string, bool) {
	sheets, styles := flattenWorkbook(&wb)

	stylesByID := buildStyleMap(styles)

	var b strings.Builder
	var hasWarning bool
	for _, sh := range sheets {
		s, warn := renderSheetHTML(sh, stylesByID, opts)
		b.WriteString(s)
		if warn {
			hasWarning = true
		}
	}
	return b.String(), hasWarning
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
	"thin":             "1px",
	"medium":           "2px",
	"thick":            "3px",
	"hair":             "1px",
	"dashed":           "",
	"dotted":           "",
	"double":           "",
	"mediumDashed":     "2px",
	"dashDot":          "",
	"mediumDashDot":    "2px",
	"dashDotDot":       "",
	"mediumDashDotDot": "2px",
	"slantDashDot":     "",
}

// borderCSSStyle は border style 名を CSS のスタイル表現に変換する。
var borderCSSStyle = map[string]string{
	"thin":             "solid",
	"medium":           "solid",
	"thick":            "solid",
	"hair":             "solid",
	"dashed":           "dashed",
	"dotted":           "dotted",
	"double":           "double",
	"mediumDashed":     "dashed",
	"dashDot":          "dashed",
	"mediumDashDot":    "dashed",
	"dashDotDot":       "dashed",
	"mediumDashDotDot": "dashed",
	"slantDashDot":     "dashed",
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
func renderSheetHTML(sh Sheet, stylesByID map[int]Style, opts HTMLOptions) (string, bool) {
	if len(sh.Cells) == 0 && len(sh.Rows) == 0 {
		return "", false
	}
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
		return "", false
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

	hidden, anchors := buildMergeMap(sh.Merges)

	const thStyle = `style="font-weight:bold;border:1px solid #000"`

	var b strings.Builder
	var hasWarning bool
	withHeader := opts.Mode != MarkdownModeValue

	b.WriteString(`<table style="border-collapse:collapse">` + "\n")
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
			cell := rows[r][c]
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
			var cellStyles []string
			if opts.GridLines {
				cellStyles = append(cellStyles, "border:1px solid #d0d0d0")
			}
			var hasExplicitAlign bool
			if cell.S > 0 {
				if st, found := stylesByID[cell.S]; found {
					if st.Alignment != nil && st.Alignment.Horizontal != "" {
						hasExplicitAlign = true
					}
					if css := styleToCSS(st); css != "" {
						cellStyles = append(cellStyles, css)
					}
				}
			}
			if !hasExplicitAlign {
				switch cell.T {
				case "n", "d":
					cellStyles = append(cellStyles, "text-align:right")
				case "b":
					cellStyles = append(cellStyles, "text-align:center")
				case "f":
					if _, isNum := cell.V.(float64); isNum {
						cellStyles = append(cellStyles, "text-align:right")
					}
				case "":
					if cell.V != nil {
						switch cell.V.(type) {
						case float64, int, int64:
							cellStyles = append(cellStyles, "text-align:right")
						case bool:
							cellStyles = append(cellStyles, "text-align:center")
						}
					}
				}
			}
			if len(cellStyles) > 0 {
				b.WriteString(` style="`)
				b.WriteString(htmlEsc(strings.Join(cellStyles, ";")))
				b.WriteString(`"`)
			}
			b.WriteString(">")
			if withHeader {
				b.WriteString(formatCellHTMLMode(cell, opts.Mode, &hasWarning))
			} else {
				b.WriteString(formatCellHTML(cell, &hasWarning))
			}
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String(), hasWarning
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
func formatCellHTML(cell Cell, hasWarning *bool) string {
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
			*hasWarning = true
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
			*hasWarning = true
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
func formatCellHTMLMode(cell Cell, mode MarkdownMode, hasWarning *bool) string {
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
