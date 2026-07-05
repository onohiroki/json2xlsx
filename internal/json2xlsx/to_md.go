package json2xlsx

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
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
	Mode           MarkdownMode
	RowIndex       bool
	FirstRowHeader bool
	ExplicitMode   bool
	DataJSON       bool
	EvalFormulas   bool
}

// ToMarkdown は入力 (JSON Workbook または XLSX) を Markdown テーブルに変換して書き出す。
// 入力種別は先頭 4 バイトの magic byte (PK\x03\x04 → XLSX) で判定する。
func ToMarkdown(r io.Reader, w io.Writer, opts MarkdownOptions) error {
	if opts.Mode == "" {
		opts.Mode = MarkdownModeFormula
	}
	if err := ValidateMode(opts.Mode); err != nil {
		return err
	}

	res, err := ReadWorkbook(r, opts.DataJSON)
	if err != nil {
		return err
	}
	wb := res.Workbook
	var formulaWarnings []string
	if opts.EvalFormulas {
		formulaWarnings = EvalWorkbookFormulas(wb)
	}

	pendingWarnings := checkJSONArrayWarning(res, opts.ExplicitMode, opts.Mode)

	out, hasWarning := renderMarkdown(*wb, opts)
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, msg := range formulaWarnings {
		fmt.Fprintln(os.Stderr, msg)
	}
	for _, msg := range pendingWarnings {
		fmt.Fprintln(os.Stderr, msg)
	}
	if hasWarning {
		fmt.Fprintln(os.Stderr, warnMissingFormulaValue(opts.Mode))
	}
	return nil
}

// displayWidth は文字列の表示幅を返す（全角2、半角1）。
func displayWidth(s string) int {
	return runewidth.StringWidth(s)
}

// padCell は文字列 s を表示幅 width になるまでスペースでパディングする。
// align は "---"（左寄せ）, "---:"（右寄せ）, ":---:"（中央寄せ）のいずれか。
func padCell(s string, width int, align string) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	switch align {
	case "---:":
		return strings.Repeat(" ", width-w) + s
	case ":---:":
		left := (width - w) / 2
		right := width - w - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // "---"
		return s + strings.Repeat(" ", width-w)
	}
}

// formatSeparator は区切り行のセパレータ文字列を生成する。
// align は "---"（左寄せ）, "---:"（右寄せ）, ":---:"（中央寄せ）のいずれか。
func formatSeparator(align string, width int) string {
	switch align {
	case "---:":
		if width < 2 {
			return strings.Repeat("-", width)
		}
		return strings.Repeat("-", width-1) + ":"
	case ":---:":
		if width < 3 {
			return strings.Repeat("-", width)
		}
		return ":" + strings.Repeat("-", width-2) + ":"
	default:
		return strings.Repeat("-", width)
	}
}

// renderMarkdown は Workbook を Markdown 文字列にレンダリングする。
func renderMarkdown(wb Workbook, opts MarkdownOptions) (string, bool) {
	sheets := wb.Sheets

	var b strings.Builder
	var hasWarning bool
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
		out, w := renderSheet(sh, opts)
		b.WriteString(out)
		if w {
			hasWarning = true
		}
	}
	return b.String(), hasWarning
}

// renderSheet は単一シートをテーブルとしてレンダリングする。空シートは空文字を返す。
// FirstRowHeader=true のとき 1 行目をテーブルヘッダとして扱う（--first-row-header）。
// それ以外は A/B/C 列名ヘッダ + 行番号を表示する（デフォルト）。
func renderSheet(sh Sheet, opts MarkdownOptions) (string, bool) {
	cg, ok := BuildCellGrid(sh)
	if !ok {
		return "", false
	}

	colAlign := make([]string, cg.MaxCol+1)
	for c := 1; c <= cg.MaxCol; c++ {
		colAlign[c] = "---"
	}
	startRow := 3
	if opts.FirstRowHeader {
		startRow = 2
	}
	sampleEnd := startRow + 4
	if sampleEnd > cg.MaxRow {
		sampleEnd = cg.MaxRow
	}
	typeCount := make([]map[string]int, cg.MaxCol+1)
	for c := 1; c <= cg.MaxCol; c++ {
		typeCount[c] = map[string]int{}
	}
	for r := startRow; r <= sampleEnd; r++ {
		for c := 1; c <= cg.MaxCol; c++ {
			cell := cg.Rows[r][c]
			t := cell.T
			if t == "f" {
				if _, isNum := cell.V.(float64); isNum {
					t = "n"
				} else {
					t = "s"
				}
			}
			if t != "" {
				typeCount[c][t]++
			} else if cell.V != nil {
				switch cell.V.(type) {
				case float64, int, int64:
					typeCount[c]["n"]++
				case bool:
					typeCount[c]["b"]++
				default:
					typeCount[c]["s"]++
				}
			}
		}
	}
	for c := 1; c <= cg.MaxCol; c++ {
		total := 0
		for _, v := range typeCount[c] {
			total += v
		}
		if total == 0 {
			continue
		}
		nCount := typeCount[c]["n"] + typeCount[c]["d"]
		bCount := typeCount[c]["b"]
		switch {
		case nCount > total/2:
			colAlign[c] = "---:"
		case bCount > total/2:
			colAlign[c] = ":---:"
		}
	}

	colWidths := make([]int, cg.MaxCol+1)
	for c := 1; c <= cg.MaxCol; c++ {
		colWidths[c] = displayWidth(cg.ColNames[c])
	}

	rowIndexWidth := 0
	if opts.RowIndex && !opts.FirstRowHeader {
		rowIndexWidth = len(strconv.Itoa(cg.MaxRow))
	}

	var hasWarning bool
	if opts.FirstRowHeader {
		for c := 1; c <= cg.MaxCol; c++ {
			cell := cg.Rows[1][c]
			if w := displayWidth(formatCell(cell, opts.Mode, &hasWarning)); w > colWidths[c] {
				colWidths[c] = w
			}
		}
	}

	dataStart := 1
	if opts.FirstRowHeader {
		dataStart = 2
	}
	for r := dataStart; r <= cg.MaxRow; r++ {
		for c := 1; c <= cg.MaxCol; c++ {
			cell := cg.Rows[r][c]
			if w := displayWidth(formatCell(cell, opts.Mode, &hasWarning)); w > colWidths[c] {
				colWidths[c] = w
			}
		}
	}

	const minSepWidth = 3
	for c := 1; c <= cg.MaxCol; c++ {
		if colWidths[c] < minSepWidth {
			colWidths[c] = minSepWidth
		}
	}
	if rowIndexWidth < minSepWidth {
		rowIndexWidth = minSepWidth
	}

	var b strings.Builder

	if opts.FirstRowHeader {
		b.WriteString("|")
		for c := 1; c <= cg.MaxCol; c++ {
			cell := cg.Rows[1][c]
			val := formatCell(cell, opts.Mode, &hasWarning)
			b.WriteString(" ")
			b.WriteString(padCell(val, colWidths[c], "---"))
			b.WriteString(" |")
		}
		b.WriteString("\n")

		b.WriteString("|")
		for c := 1; c <= cg.MaxCol; c++ {
			b.WriteString(" ")
			b.WriteString(formatSeparator(colAlign[c], colWidths[c]))
			b.WriteString(" |")
		}
		b.WriteString("\n")

		for r := 2; r <= cg.MaxRow; r++ {
			b.WriteString("|")
			for c := 1; c <= cg.MaxCol; c++ {
				cell := cg.Rows[r][c]
				val := formatCell(cell, opts.Mode, &hasWarning)
				b.WriteString(" ")
				b.WriteString(padCell(val, colWidths[c], colAlign[c]))
				b.WriteString(" |")
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("|")
		if opts.RowIndex {
			b.WriteString(" ")
			b.WriteString(padCell("", rowIndexWidth, "---:"))
			b.WriteString(" |")
		}
		for c := 1; c <= cg.MaxCol; c++ {
			b.WriteString(" ")
			b.WriteString(padCell(cg.ColNames[c], colWidths[c], "---"))
			b.WriteString(" |")
		}
		b.WriteString("\n")

		b.WriteString("|")
		if opts.RowIndex {
			b.WriteString(" ")
			b.WriteString(formatSeparator("---:", rowIndexWidth))
			b.WriteString(" |")
		}
		for c := 1; c <= cg.MaxCol; c++ {
			b.WriteString(" ")
			b.WriteString(formatSeparator(colAlign[c], colWidths[c]))
			b.WriteString(" |")
		}
		b.WriteString("\n")

		for r := 1; r <= cg.MaxRow; r++ {
			b.WriteString("|")
			if opts.RowIndex {
				b.WriteString(" ")
				b.WriteString(padCell(strconv.Itoa(r), rowIndexWidth, "---:"))
				b.WriteString(" |")
			}
			for c := 1; c <= cg.MaxCol; c++ {
				cell := cg.Rows[r][c]
				val := formatCell(cell, opts.Mode, &hasWarning)
				b.WriteString(" ")
				b.WriteString(padCell(val, colWidths[c], colAlign[c]))
				b.WriteString(" |")
			}
			b.WriteString("\n")
		}
	}

	return b.String(), hasWarning
}

// formatCell は 1 セルの Markdown 表現を返す。
func formatCell(cell Cell, mode MarkdownMode, hasWarning *bool) string {
	return escapeMarkdownCell(CellDisplayValue(cell, mode, hasWarning))
}

// escapeMarkdownCell は Markdown テーブルセル向けのエスケープを行う。
func escapeMarkdownCell(s string) string {
	if s == "" {
		return ""
	}
	// バックスラッシュを先に処理。
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", `\|`)
	s = normalizeNewlines(s)
	s = strings.ReplaceAll(s, "\n", "<br />")
	return s
}
