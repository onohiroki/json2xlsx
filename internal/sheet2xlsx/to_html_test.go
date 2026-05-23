package sheet2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func runToHTML(t *testing.T, input string) string {
	t.Helper()
	var out bytes.Buffer
	if err := ToHTML(strings.NewReader(input), &out, HTMLOptions{}); err != nil {
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
	want := "<table>\n<tr><td>hello</td><td>42</td></tr>\n<tr><td></td><td>3.14</td></tr>\n</table>\n"
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
	want := "<table>\n<tr><td>3</td><td>=X</td></tr>\n</table>\n"
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

func TestToHTML_MultiSheet(t *testing.T) {
	in := `{
	  "sheets": [
	    {"name":"First","cells":{"A1":{"t":"s","v":"x"}}},
	    {"name":"Second","cells":{"A1":{"t":"n","v":1}}}
	  ]
	}`
	got := runToHTML(t, in)
	want := "<table>\n<tr><td>x</td></tr>\n</table>\n<table>\n<tr><td>1</td></tr>\n</table>\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToHTML_EmptyCell(t *testing.T) {
	in := `{"cells":{"A1":{"t":"s","v":"x"},"B2":{"t":"n","v":1}}}`
	got := runToHTML(t, in)
	want := "<table>\n<tr><td>x</td><td></td></tr>\n<tr><td></td><td>1</td></tr>\n</table>\n"
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

func TestToHTML_XLSXPath(t *testing.T) {
	jsonIn := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"s","v":"hello"},
	    "B1": {"t":"n","v":3}
	  }
	}`
	var xlsxBuf bytes.Buffer
	if err := Convert(strings.NewReader(jsonIn), &xlsxBuf, ""); err != nil {
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
