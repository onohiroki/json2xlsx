package json2xlsx

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func runToCSV(t *testing.T, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := ToCSV(strings.NewReader(input), &out, "", 0, false, false)
	return out.String(), err
}

func TestToCSV_Basic(t *testing.T) {
	in := `[
  {
    "製品": "商品A\n特価",
    "数量": "100",
    "単価": "5,000",
    "合計": "500,000",
    "": null
  },
  {
    "製品": "商品B",
    "数量": "50",
    "単価": "8,000",
    "合計": "400,000",
    "": null
  },
  {
    "製品": "合計",
    "数量": null,
    "単価": null,
    "合計": "900,000",
    "": null
  }
]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "製品,数量,単価,合計,\n" +
		"\"商品A\n特価\",100,\"5,000\",\"500,000\",\n" +
		"商品B,50,\"8,000\",\"400,000\",\n" +
		"合計,,,\"900,000\",\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLIInput(t *testing.T) {
	in := "売上\n" + `[
  {
    "製品": "商品A\n特価",
    "数量": 100,
    "単価": 5000,
    "合計": ""
  },
  {
    "製品": "商品B",
    "数量": 50,
    "単価": 8000,
    "合計": ""
  },
  {
    "製品": "合計",
    "合計": ""
  }
]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "製品,数量,単価,合計\n" +
		"\"商品A\n特価\",100,5000,\n" +
		"商品B,50,8000,\n" +
		"合計,,,\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLIMultiSheet_FirstOnly(t *testing.T) {
	in := "Sheet1\n[{\"a\":1}]\nSheet2\n[{\"a\":2}]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n1\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLISheetWithoutArray(t *testing.T) {
	_, err := runToCSV(t, "売上\n")
	if err == nil {
		t.Fatal("expected error for sheet name without array")
	}
}

func TestToCSV_EmptyArray(t *testing.T) {
	_, err := runToCSV(t, `[]`)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestToCSV_WorkbookInput(t *testing.T) {
	in := `{"name":"S","cells":{"A1":{"v":"val"}}}`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "val\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_SheetJSStyle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCSV  string
		wantWarn string
	}{
		{
			name: "Basic SheetJS style",
			input: `{
				"Sheet1": {
					"A1": {"v": "Header1"},
					"B1": {"v": "Header2"},
					"A2": {"v": "Val1"},
					"B2": {"v": 100}
				}
			}`,
			wantCSV: "Header1,Header2\nVal1,100\n",
		},
		{
			name: "Formula without value",
			input: `{
				"Sheet1": {
					"A1": {"v": "Header1"},
					"A2": {"f": "SUM(B1:B10)"}
				}
			}`,
			wantCSV:  "Header1\n\n",
			wantWarn: "Warning: Some cells have formulas but no values; treating them as empty.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer

			// Capture stderr
			oldStderr := os.Stderr
			r, w_err, _ := os.Pipe()
			os.Stderr = w_err

			err := ToCSV(strings.NewReader(tt.input), &w, "", 0, false, false)

			// Restore stderr
			w_err.Close()
			os.Stderr = oldStderr
			var stderrBuf bytes.Buffer
			io.Copy(&stderrBuf, r)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := w.String()
			if got != tt.wantCSV {
				t.Errorf("got CSV:\n%s\nwant CSV:\n%s", got, tt.wantCSV)
			}

			if tt.wantWarn != "" {
				if !strings.Contains(stderrBuf.String(), tt.wantWarn) {
					t.Errorf("expected warning %q, got %q", tt.wantWarn, stderrBuf.String())
				}
			} else if stderrBuf.Len() > 0 {
				t.Errorf("unexpected warning: %q", stderrBuf.String())
			}
		})
	}
}

func TestToCSV_NonJSON(t *testing.T) {
	_, err := runToCSV(t, `hello`)
	if err == nil {
		t.Fatal("expected error for non-JSON")
	}
}

func TestToCSV_SingleRow(t *testing.T) {
	in := `[{"a":"1","b":"2"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b\n1,2\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_NullValues(t *testing.T) {
	in := `[{"a":null,"b":"x","c":null}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b,c\n,x,\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_RejectBoolValue(t *testing.T) {
	_, err := runToCSV(t, `[{"a":true}]`)
	if err == nil {
		t.Fatal("expected error for bool value")
	}
}

func TestToCSV_EmptyKey(t *testing.T) {
	in := `[{"":"val","b":"x"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := ",b\nval,x\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_EmbeddedQuotes(t *testing.T) {
	in := `[{"a":"he said \"hello\""}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n\"he said \"\"hello\"\"\"\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_EmbeddedNewlines(t *testing.T) {
	in := "[\n  {\"a\":\"line1\\nline2\"}\n]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n\"line1\nline2\"\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_DifferentKeys(t *testing.T) {
	in := `[{"a":"1"},{"b":"2"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b\n1,\n,2\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_NumberLexemePreserved(t *testing.T) {
	in := `[{"n":1e3,"m":1.0,"p":-2E-3}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "n,m,p\n1e3,1.0,-2E-3\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_BOM(t *testing.T) {
	in := "\xef\xbb\xbf[{\"a\":\"1\"}]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n1\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestResolveSheet(t *testing.T) {
	t.Run("single sheet with Cells", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{{Cells: map[string]Cell{"A1": {V: "x"}}}}}
		sh, err := resolveSheet(wb, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["A1"].V != "x" {
			t.Errorf("expected A1=x, got %v", sh.Cells["A1"].V)
		}
	})

	t.Run("Sheets array first", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{
			{Name: "S1", Cells: map[string]Cell{"A1": {V: "s1"}}},
			{Name: "S2", Cells: map[string]Cell{"A1": {V: "s2"}}},
		}}
		sh, err := resolveSheet(wb, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["A1"].V != "s1" {
			t.Errorf("expected s1, got %v", sh.Cells["A1"].V)
		}
	})

	t.Run("sheet name from Sheets array", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{
			{Name: "Sheet1", Cells: map[string]Cell{"B2": {V: "found"}}},
		}}
		sh, err := resolveSheet(wb, "Sheet1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["B2"].V != "found" {
			t.Errorf("expected found, got %v", sh.Cells["B2"].V)
		}
	})

	t.Run("sheet name from Sheets array", func(t *testing.T) {
		wb := Workbook{
			Sheets: []Sheet{
				{Name: "Report", Cells: map[string]Cell{"C3": {V: "book_val"}}},
			},
		}
		sh, err := resolveSheet(wb, "Report", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["C3"].V != "book_val" {
			t.Errorf("expected book_val, got %v", sh.Cells["C3"].V)
		}
	})

	t.Run("sheet name not found", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{{Name: "S1"}}}
		_, err := resolveSheet(wb, "NONEXIST", 0)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("sheet index valid", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{
			{Cells: map[string]Cell{"A1": {V: "idx0"}}},
			{Cells: map[string]Cell{"A1": {V: "idx1"}}},
		}}
		sh, err := resolveSheet(wb, "", 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["A1"].V != "idx1" {
			t.Errorf("expected idx1, got %v", sh.Cells["A1"].V)
		}
	})

	t.Run("sheet index out of range", func(t *testing.T) {
		wb := Workbook{Sheets: []Sheet{{Cells: map[string]Cell{"A1": {V: "x"}}}}}
		_, err := resolveSheet(wb, "", 999)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("sheet index out of range (empty sheets)", func(t *testing.T) {
		wb := Workbook{}
		_, err := resolveSheet(wb, "", 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("first sheet fallback (no name, no index)", func(t *testing.T) {
		wb := Workbook{
			Sheets: []Sheet{
				{Name: "X", Cells: map[string]Cell{"Z99": {V: "fallback"}}},
			},
		}
		sh, err := resolveSheet(wb, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells["Z99"].V != "fallback" {
			t.Errorf("expected fallback, got %v", sh.Cells["Z99"].V)
		}
	})

	t.Run("no cells anywhere", func(t *testing.T) {
		wb := Workbook{}
		sh, err := resolveSheet(wb, "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sh.Cells != nil {
			t.Errorf("expected nil, got %v", sh.Cells)
		}
	})
}

func TestCellGridToCSVRows(t *testing.T) {
	t.Run("basic grid", func(t *testing.T) {
		cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
			"A1": {V: "a1"},
			"B1": {V: "b1"},
			"A2": {V: "a2"},
		}})
		if !ok {
			t.Fatal("expected ok")
		}
		var warn bool
		grid := cellGridToCSVRows(cg, &warn)
		if warn {
			t.Error("expected no warning")
		}
		if len(grid) != 2 || len(grid[0]) != 2 {
			t.Fatalf("expected 2x2 grid, got %dx%d", len(grid), len(grid[0]))
		}
		if grid[0][0] != "a1" || grid[0][1] != "b1" || grid[1][0] != "a2" {
			t.Errorf("unexpected grid content: %v", grid)
		}
	})

	t.Run("formula without value triggers warning", func(t *testing.T) {
		cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
			"A1": {F: "SUM(B:B)"},
		}})
		if !ok {
			t.Fatal("expected ok")
		}
		var warn bool
		grid := cellGridToCSVRows(cg, &warn)
		if !warn {
			t.Error("expected warning")
		}
		if len(grid) != 1 || grid[0][0] != "" {
			t.Errorf("expected empty cell, got %v", grid[0][0])
		}
	})

	t.Run("empty cell grid", func(t *testing.T) {
		var warn bool
		grid := cellGridToCSVRows(CellGrid{}, &warn)
		if warn {
			t.Error("expected no warning for empty")
		}
		if grid != nil {
			t.Error("expected nil grid")
		}
	})

	t.Run("invalid cell addresses are skipped", func(t *testing.T) {
		cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
			"A1": {V: "ok"},
			"??": {V: "bad"},
		}})
		if !ok {
			t.Fatal("expected ok")
		}
		var warn bool
		grid := cellGridToCSVRows(cg, &warn)
		if warn {
			t.Error("expected no warning")
		}
		if len(grid) != 1 || grid[0][0] != "ok" {
			t.Errorf("expected single cell 'ok', got %v", grid)
		}
	})

	t.Run("value with formula (no warning)", func(t *testing.T) {
		cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
			"A1": {V: 42.0, F: "SUM(B1:B10)"},
		}})
		if !ok {
			t.Fatal("expected ok")
		}
		var warn bool
		grid := cellGridToCSVRows(cg, &warn)
		if warn {
			t.Error("expected no warning when value present")
		}
		if grid[0][0] != "42" {
			t.Errorf("expected 42, got %q", grid[0][0])
		}
	})
}

func TestWriteCSVRecords(t *testing.T) {
	t.Run("basic write", func(t *testing.T) {
		var buf bytes.Buffer
		records := [][]string{
			{"a", "b"},
			{"1", "2"},
		}
		if err := writeCSVRecords(&buf, records); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "a,b\n1,2\n"
		if got := buf.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty records", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeCSVRecords(&buf, [][]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := buf.String(); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestGuessCellMapFromData(t *testing.T) {
	t.Run("valid cell map", func(t *testing.T) {
		data := []byte(`{"Sheet1":{"A1":{"v":"hello"},"B1":{"v":123}}}`)
		cells := guessCellMapFromData(data)
		if cells == nil || len(cells) != 2 {
			t.Fatalf("expected 2 cells, got %d", len(cells))
		}
		if cells["A1"].V != "hello" || cells["B1"].V != float64(123) {
			t.Errorf("unexpected cell values: %v", cells)
		}
	})

	t.Run("not cell-like map returns nil", func(t *testing.T) {
		data := []byte(`{"config":{"debug":true}}`)
		cells := guessCellMapFromData(data)
		if cells != nil {
			t.Error("expected nil for non-cell data")
		}
	})

	t.Run("non-object map returns nil", func(t *testing.T) {
		data := []byte(`"just a string"`)
		cells := guessCellMapFromData(data)
		if cells != nil {
			t.Error("expected nil for string input")
		}
	})
}

func TestToCSV_WorkbookDateInput(t *testing.T) {
	in := `{
		"name": "S",
		"cells": {
			"A1": {"t": "d", "v": 45678, "z": "yyyy-mm-dd"},
			"A2": {"t": "d", "v": "2025-01-21T00:00:00Z", "z": "yyyy-mm-dd"},
			"B1": {"t": "n", "v": 42}
		}
	}`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "2025-01-21T00:00:00,42\n2025-01-21T00:00:00Z,\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXInput(t *testing.T) {
	// Create a temporary XLSX file
	f := excelize.NewFile()
	sheetName := "TestSheet"
	f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "Header1")
	f.SetCellValue(sheetName, "B1", "Header2")
	f.SetCellValue(sheetName, "A2", "Value1")
	f.SetCellValue(sheetName, "B2", 123)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("failed to write xlsx: %v", err)
	}

	t.Run("Default sheet (first sheet)", func(t *testing.T) {
		// excelize.NewFile() creates "Sheet1" as the first sheet.
		// We'll put some data in Sheet1 to test default behavior.
		f1 := excelize.NewFile()
		f1.SetCellValue("Sheet1", "A1", "S1V1")
		var buf1 bytes.Buffer
		f1.Write(&buf1)

		var out bytes.Buffer
		err := ToCSV(bytes.NewReader(buf1.Bytes()), &out, "", 0, false, false)
		if err != nil {
			t.Fatalf("ToCSV: %v", err)
		}
		got := out.String()
		want := "S1V1\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Specific sheet", func(t *testing.T) {
		var out bytes.Buffer
		err := ToCSV(bytes.NewReader(buf.Bytes()), &out, sheetName, 0, false, false)
		if err != nil {
			t.Fatalf("ToCSV: %v", err)
		}
		got := out.String()
		want := "Header1,Header2\nValue1,123\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Specific sheet by index", func(t *testing.T) {
		var out bytes.Buffer
		// Sheet1 is index 1, TestSheet is index 2
		err := ToCSV(bytes.NewReader(buf.Bytes()), &out, "", 2, false, false)
		if err != nil {
			t.Fatalf("ToCSV: %v", err)
		}
		got := out.String()
		want := "Header1,Header2\nValue1,123\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
