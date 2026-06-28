package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func runToMD(t *testing.T, input string, opts MarkdownOptions) string {
	t.Helper()
	var out bytes.Buffer
	if err := ToMarkdown(strings.NewReader(input), &out, opts); err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	return out.String()
}

func TestToMarkdown_JSON_ModeFormula(t *testing.T) {
	in := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"n","v":42},
	    "B1": {"t":"n","v":7},
	    "C1": {"t":"f","f":"A1*B1","v":294}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula})
	want := "| A   | B   | C      |\n" +
		"| --- | --- | ------ |\n" +
		"| 42  | 7   | =A1*B1 |\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToMarkdown_JSON_ModeValue(t *testing.T) {
	in := `{
	  "name": "S",
	  "cells": {
	    "A1": {"t":"f","f":"1+2","v":3},
	    "B1": {"t":"f","f":"X"}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeValue})
	want := "| A   | B   |\n" +
		"| --- | --- |\n" +
		"| 3   | =X  |\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToMarkdown_JSON_ModeBoth(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"f","f":"B1*2","v":84},
	    "B1": {"t":"n","v":42}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeBoth})
	want := "| A             | B   |\n" +
		"| ------------- | --- |\n" +
		"| 84<br />=B1*2 | 42  |\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToMarkdown_Default(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"a"},
	    "B1": {"t":"s","v":"b"},
	    "A2": {"t":"n","v":1},
	    "B2": {"t":"n","v":2}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula, FirstRowHeader: false, RowIndex: true})
	want := "|     | A   | B   |\n" +
		"| --: | --- | --- |\n" +
		"|   1 | a   | b   |\n" +
		"|   2 | 1   | 2   |\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToMarkdown_MultiSheet(t *testing.T) {
	in := `{
	  "sheets": [
	    {"name":"First","cells":{"A1":{"t":"s","v":"x"}}},
	    {"name":"Second","cells":{"A1":{"t":"n","v":1}}}
	  ]
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula})
	want := "## First\n\n" +
		"| A   |\n| --- |\n| x   |\n" +
		"\n## Second\n\n" +
		"| A   |\n| --- |\n| 1   |\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToMarkdown_Escaping(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"a|b"},
	    "B1": {"t":"s","v":"line1\nline2"},
	    "C1": {"t":"s","v":"back\\slash"}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula})
	want := "| A    | B                | C           |\n" +
		"| ---- | ---------------- | ----------- |\n" +
		`| a\|b | line1<br />line2 | back\\slash |` + "\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToMarkdown_JSON_DateSerial(t *testing.T) {
	in := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"n","v":45678,"z":"yyyy-mm-dd"},
	    "B1": {"t":"n","v":0.04623843,"z":"h:mm:ss"},
	    "C1": {"t":"s","v":"hello","z":"yyyy-mm-dd"},
	    "D1": {"t":"n","v":42,"z":"#,##0"}
	  }
	}`
	got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula})
	if !strings.Contains(got, "2025-01-21T00:00:00") {
		t.Fatalf("date serial 45678 should become RFC3339. got:\n%s", got)
	}
	if !strings.Contains(got, "1:06:35") {
		t.Fatalf("time serial 0.04623843 should become h:mm:ss format. got:\n%s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("string value with date z should stay as-is. got:\n%s", got)
	}
	if !strings.Contains(got, "42") {
		t.Fatalf("non-date format #,##0 should remain numeric. got:\n%s", got)
	}
}

func TestToMarkdown_ModeValue_FallbackWarning(t *testing.T) {
	// mode=v でセルに f はあるが v がない場合、hasWarning=true が返り、式が表示される。
	wb := Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "s", V: "製品"},
				"B1": {T: "f", F: "B2*C2"}, // v なし
			},
		}},
	}
	out, hasWarning := renderMarkdown(wb, MarkdownOptions{Mode: MarkdownModeValue})
	if !hasWarning {
		t.Fatalf("expected hasWarning=true when cell has formula but no value in mode=v")
	}
	if !strings.Contains(out, "=B2*C2") {
		t.Fatalf("expected formula fallback in output. got:\n%s", out)
	}
}

func TestToMarkdown_ModeValue_NoWarningWhenValuePresent(t *testing.T) {
	// mode=v でセルに v がある場合、hasWarning=false が返る。
	wb := Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "f", F: "1+2", V: float64(3)},
			},
		}},
	}
	_, hasWarning := renderMarkdown(wb, MarkdownOptions{Mode: MarkdownModeValue})
	if hasWarning {
		t.Fatalf("expected hasWarning=false when cell has both formula and value in mode=v")
	}
}

func TestToMarkdown_InvalidMode(t *testing.T) {
	var out bytes.Buffer
	err := ToMarkdown(strings.NewReader(`{"cells":{}}`), &out, MarkdownOptions{Mode: MarkdownMode("xxx")})
	if err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}

func TestToMarkdown_InvalidInput(t *testing.T) {
	var out bytes.Buffer
	err := ToMarkdown(strings.NewReader("\x00\x01"), &out, MarkdownOptions{})
	if err == nil {
		t.Fatalf("expected error for too-short / unknown input")
	}
}

func TestToMarkdown_UnknownBinary(t *testing.T) {
	var out bytes.Buffer
	err := ToMarkdown(strings.NewReader("\xFF\xFE\xFD\xFCxxxx"), &out, MarkdownOptions{})
	if err == nil {
		t.Fatalf("expected error for binary that's neither JSON nor XLSX")
	}
}

func TestToMarkdown_NoHeader(t *testing.T) {
	in := `{
	  "cells": {
	    "A1": {"t":"s","v":"Name"},
	    "B1": {"t":"s","v":"Value"},
	    "A2": {"t":"s","v":"Alice"},
	    "B2": {"t":"n","v":100},
	    "A3": {"t":"s","v":"Bob"},
	    "B3": {"t":"n","v":200}
	  }
	}`
	t.Run("basic", func(t *testing.T) {
		got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula, FirstRowHeader: true})
		want := "| Name  | Value |\n" +
			"| ----- | ----: |\n" +
			"| Alice |   100 |\n" +
			"| Bob   |   200 |\n"
		if got != want {
			t.Fatalf("mismatch.\n got:\n%s\nwant:\n%s", got, want)
		}
	})
	t.Run("row_index_ignored", func(t *testing.T) {
		got := runToMD(t, in, MarkdownOptions{Mode: MarkdownModeFormula, RowIndex: true, FirstRowHeader: true})
		want := "| Name  | Value |\n" +
			"| ----- | ----: |\n" +
			"| Alice |   100 |\n" +
			"| Bob   |   200 |\n"
		if got != want {
			t.Fatalf("RowIndex should be ignored with FirstRowHeader.\n got:\n%s\nwant:\n%s", got, want)
		}
	})
	t.Run("single_row", func(t *testing.T) {
		in := `{"cells":{"A1":{"t":"s","v":"only"}}}`
		got := runToMD(t, in, MarkdownOptions{FirstRowHeader: true})
		want := "| only |\n| ---- |\n"
		if got != want {
			t.Fatalf("mismatch for single row.\n got:\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestToMarkdown_XLSXPath(t *testing.T) {
	// JSON -> XLSX (Convert) -> Markdown と、JSON 直接 -> Markdown が同等。
	jsonIn := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"s","v":"hello"},
	    "B1": {"t":"n","v":3}
	  }
	}`
	var xlsxBuf bytes.Buffer
	if err := Convert(strings.NewReader(jsonIn), &xlsxBuf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	var mdFromXLSX bytes.Buffer
	if err := ToMarkdown(bytes.NewReader(xlsxBuf.Bytes()), &mdFromXLSX, MarkdownOptions{Mode: MarkdownModeFormula}); err != nil {
		t.Fatalf("ToMarkdown(xlsx): %v", err)
	}
	got := mdFromXLSX.String()
	// XLSX 経路では Sheet 名は Convert 既定 (Sheet1) になるはず。ヘッダ A B と値が出ること。
	if !strings.Contains(got, "| A     | B   |") {
		t.Fatalf("missing header. got:\n%s", got)
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "3") {
		t.Fatalf("missing values. got:\n%s", got)
	}
}
