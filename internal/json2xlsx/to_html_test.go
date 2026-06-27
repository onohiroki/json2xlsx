package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func runToHTML(t *testing.T, input string, opts ...HTMLOptions) string {
	t.Helper()
	var o HTMLOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	var out bytes.Buffer
	if err := ToHTML(strings.NewReader(input), &out, o); err != nil {
		t.Fatalf("ToHTML: %v", err)
	}
	return out.String()
}

func TestToHTML_Basic(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"hello"},
	    "B1": {"t":"n","v":42},
	    "B2": {"t":"n","v":3.14}
	  }
	}`
	got := runToHTML(t, in)
	want := "<table style=\"border-collapse:collapse\">\n<tr><td>hello</td><td style=\"text-align:right\">42</td></tr>\n<tr><td></td><td style=\"text-align:right\">3.14</td></tr>\n</table>\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToHTML_Formula(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"f","f":"1+2","v":3},
	    "B1": {"t":"f","f":"X"}
	  }
	}`
	got := runToHTML(t, in)
	want := "<table style=\"border-collapse:collapse\">\n<tr><td style=\"text-align:right\">3</td><td>=X</td></tr>\n</table>\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToHTML_DateSerial(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"n","v":45678,"z":"yyyy-mm-dd"},
	    "B1": {"t":"n","v":0.04623843,"z":"h:mm:ss"},
	    "C1": {"t":"s","v":"hello","z":"yyyy-mm-dd"},
	    "D1": {"t":"n","v":42,"z":"#,##0"}
	  }
	}`
	got := runToHTML(t, in)
	if !strings.Contains(got, "2025-01-21T00:00:00") {
		t.Fatalf("date serial 45678 should become RFC3339. got:\n%s", got)
	}
	if !strings.Contains(got, "1:06:35") {
		t.Fatalf("time serial 0.04623843 should become h:mm:ss. got:\n%s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("string value should stay as-is. got:\n%s", got)
	}
	if !strings.Contains(got, "42") {
		t.Fatalf("non-date format #,##0 should remain numeric. got:\n%s", got)
	}
}

func TestToHTML_MergeCells(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"header"},
	    "C1": {"t":"s","v":"C1"},
	    "A2": {"t":"s","v":"A2"},
	    "B2": {"t":"n","v":100},
	    "C2": {"t":"n","v":200},
	    "A3": {"t":"n","v":300},
	    "C3": {"t":"n","v":500}
	  },
	  "merges": [
	    {"range":"A1:B1"},
	    {"range":"A2:A3"}
	  ]
	}`
	got := runToHTML(t, in)
	if !strings.Contains(got, `colspan="2"`) {
		t.Fatalf("expected colspan=\"2\" for A1:B1. got:\n%s", got)
	}
	if !strings.Contains(got, `rowspan="2"`) {
		t.Fatalf("expected rowspan=\"2\" for A2:A3. got:\n%s", got)
	}
	if strings.Contains(got, "<td></td><td></td>") {
		t.Fatalf("merged cells should not produce empty cells.\n got:\n%s", got)
	}
}

func TestToHTML_Styles(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"Header","s":1},
	    "A2": {"t":"n","v":42,"s":2}
	  },
	  "styles": [
	    {"id":1,"fill":{"type":"pattern","pattern":1,"color":["#E0EBF5"]},"font":{"bold":true},"alignment":{"horizontal":"center"}},
	    {"id":2,"border":[{"style":"thin","color":"#000000"}],"numFmt":"#,##0"}
	  ]
	}`
	got := runToHTML(t, in)
	if !strings.Contains(got, "background-color:#E0EBF5") {
		t.Fatalf("expected background-color style. got:\n%s", got)
	}
	if !strings.Contains(got, "font-weight:bold") {
		t.Fatalf("expected font-weight:bold. got:\n%s", got)
	}
	if !strings.Contains(got, "text-align:center") {
		t.Fatalf("expected text-align:center. got:\n%s", got)
	}
	if !strings.Contains(got, "border:1px solid #000000") {
		t.Fatalf("expected border style. got:\n%s", got)
	}
}

func TestToHTML_Escaping(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"a&b"},
	    "B1": {"t":"s","v":"<tag>"},
	    "C1": {"t":"s","v":"line1\nline2"}
	  }
	}`
	got := runToHTML(t, in)
	if !strings.Contains(got, "a&amp;b") {
		t.Fatalf("expected &amp; for &. got:\n%s", got)
	}
	if !strings.Contains(got, "&lt;tag&gt;") {
		t.Fatalf("expected &lt;&gt; escaping. got:\n%s", got)
	}
	if !strings.Contains(got, "<br />") {
		t.Fatalf("expected <br /> for newline. got:\n%s", got)
	}
}

func TestToHTML_DefaultAlignment(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"text"},
	    "B1": {"t":"n","v":123},
	    "C1": {"t":"d","v":"2024-01-15T00:00:00Z"},
	    "D1": {"t":"b","v":true},
	    "E1": {"t":"f","f":"1+2","v":3},
	    "F1": {"t":"f","f":"hello"},
	    "G1": {"t":"s","v":"labeled","s":1}
	  },
	  "styles": [
	    {"id":1,"alignment":{"horizontal":"center"}}
	  ]
	}`
	got := runToHTML(t, in)
	if !strings.Contains(got, "<td>text</td>") {
		t.Fatalf("string should be left-aligned (no style). got:\n%s", got)
	}
	if !strings.Contains(got, "text-align:right") {
		t.Fatalf("number/date/formula-with-value should be right-aligned. got:\n%s", got)
	}
	if !strings.Contains(got, "text-align:center") {
		t.Fatalf("boolean should be center-aligned. got:\n%s", got)
	}
	if strings.Count(got, "text-align:right") != 3 {
		t.Fatalf("expected exactly 3 right-aligned cells (n, d, f with value). got:\n%s", got)
	}
	if !strings.Contains(got, `text-align:center">true`) {
		t.Fatalf("boolean cell should have center. got:\n%s", got)
	}
}

func TestToHTML_MultiSheet(t *testing.T) {
	in := `{
	  "sheets": [
	    {"name":"First","cells":{"A1":{"t":"s","v":"x"}}},
	    {"name":"Second","cells":{"A1":{"t":"n","v":1}}}
	  ]
	}`
	got := runToHTML(t, in)
	want := "<table style=\"border-collapse:collapse\">\n<tr><td>x</td></tr>\n</table>\n<table style=\"border-collapse:collapse\">\n<tr><td style=\"text-align:right\">1</td></tr>\n</table>\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToHTML_EmptyCell(t *testing.T) {
	in := `{"cells":{"A1":{"t":"s","v":"x"},"B2":{"t":"n","v":1}}}`
	got := runToHTML(t, in)
	want := "<table style=\"border-collapse:collapse\">\n<tr><td>x</td><td></td></tr>\n<tr><td></td><td style=\"text-align:right\">1</td></tr>\n</table>\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToHTML_InvalidInput(t *testing.T) {
	var out bytes.Buffer
	err := ToHTML(strings.NewReader("\x00\x01"), &out, HTMLOptions{})
	if err == nil {
		t.Fatalf("expected error for too-short input")
	}
}

func TestToHTML_UnknownBinary(t *testing.T) {
	var out bytes.Buffer
	err := ToHTML(strings.NewReader("\xFF\xFE\xFD\xFCxxxx"), &out, HTMLOptions{})
	if err == nil {
		t.Fatalf("expected error for binary that's neither JSON nor XLSX")
	}
}

func TestToHTML_EmptyWorkbook(t *testing.T) {
	got := runToHTML(t, `{"cells":{}}`)
	if got != "" {
		t.Fatalf("expected empty output for empty workbook. got:\n%q", got)
	}
}

func TestToHTML_ModeFormula(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"f","f":"SUM(B1:C1)","v":300},
	    "B1": {"t":"n","v":100},
	    "C1": {"t":"n","v":200},
	    "A2": {"t":"f","f":"B2*C2"},
	    "B2": {"t":"n","v":5},
	    "C2": {"t":"n","v":10}
	  }
	}`
	got := runToHTML(t, in, HTMLOptions{Mode: MarkdownModeFormula})
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">A</th>`) {
		t.Fatalf("expected column header 'A'. got:\n%s", got)
	}
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">B</th>`) {
		t.Fatalf("expected column header 'B'. got:\n%s", got)
	}
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">C</th>`) {
		t.Fatalf("expected column header 'C'. got:\n%s", got)
	}
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">1</th>`) {
		t.Fatalf("expected row number '1'. got:\n%s", got)
	}
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">2</th>`) {
		t.Fatalf("expected row number '2'. got:\n%s", got)
	}
	if !strings.Contains(got, "=SUM(B1:C1)") {
		t.Fatalf("expected formula =SUM(B1:C1). got:\n%s", got)
	}
	if !strings.Contains(got, "=B2*C2") {
		t.Fatalf("expected formula =B2*C2. got:\n%s", got)
	}
}

func TestToHTML_ModeBoth(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"f","f":"SUM(B1:C1)","v":300},
	    "B1": {"t":"n","v":100},
	    "C1": {"t":"n","v":200}
	  }
	}`
	got := runToHTML(t, in, HTMLOptions{Mode: MarkdownModeBoth})
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">A</th>`) {
		t.Fatalf("expected column header 'A'. got:\n%s", got)
	}
	if !strings.Contains(got, `<th style="font-weight:bold;border:1px solid #000">1</th>`) {
		t.Fatalf("expected row number '1'. got:\n%s", got)
	}
	if !strings.Contains(got, "300<br />=SUM(B1:C1)") {
		t.Fatalf("expected value+formula. got:\n%s", got)
	}
}

func TestToHTML_XLSXPath(t *testing.T) {
	jsonIn := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"s","v":"hello"},
	    "B1": {"t":"n","v":3}
	  }
	}`
	var xlsxBuf bytes.Buffer
	if err := Convert(strings.NewReader(jsonIn), &xlsxBuf); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	var htmlBuf bytes.Buffer
	if err := ToHTML(bytes.NewReader(xlsxBuf.Bytes()), &htmlBuf, HTMLOptions{}); err != nil {
		t.Fatalf("ToHTML(xlsx): %v", err)
	}
	got := htmlBuf.String()
	if !strings.Contains(got, "hello") || !strings.Contains(got, "3") {
		t.Fatalf("missing values. got:\n%s", got)
	}
}
