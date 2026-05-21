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
type MarkdownOptions struct {
	Mode     MarkdownMode
	RowIndex bool
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
		wb, err = extractWorkbook(f)
		if err != nil {
			return err
		}
	} else {
		dec := json.NewDecoder(br)
		if err := dec.Decode(&wb); err != nil {
			return fmt.Errorf("unsupported input: expected JSON Workbook or XLSX: %w", err)
		}
	}

	out := renderMarkdown(wb, opts)
	if _, err := io.WriteString(w, out); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

// renderMarkdown は Workbook を Markdown 文字列にレンダリングする。
func renderMarkdown(wb Workbook, opts MarkdownOptions) string {
	// 単一シート形式を Sheets に正規化。
	sheets := wb.Sheets
	if len(sheets) == 0 && (wb.Cells != nil || wb.Name != "" || wb.Merges != nil) {
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

	// ヘッダ行
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

	// 本文
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
	default:
		// t = s/n/b/d/空 はすべて v を文字列化。
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
