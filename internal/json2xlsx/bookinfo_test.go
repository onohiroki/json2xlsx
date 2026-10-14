package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func mustUnmarshalWorkbook(t *testing.T, jsonStr string) *Workbook {
	t.Helper()
	wb, err := UnmarshalWorkbook([]byte(jsonStr), false)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	normalizeWorkbook(wb)
	return wb
}

func TestBuildBookInfoBasic(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "x"},
			"B2": {"t": "n", "v": 1},
			"C3": {"t": "f", "f": "SUM(B2:B2)"}
		}
	}`)

	info := BuildBookInfo(wb)
	if info.Schema != InfoSchema {
		t.Errorf("schema = %q, want %q", info.Schema, InfoSchema)
	}
	if len(info.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(info.Sheets))
	}
	sh := info.Sheets[0]
	if sh.Name != "S1" {
		t.Errorf("name = %q, want S1", sh.Name)
	}
	if sh.UsedRange != "A1:C3" {
		t.Errorf("usedRange = %q, want A1:C3", sh.UsedRange)
	}
	if sh.Rows != 3 || sh.Cols != 3 {
		t.Errorf("rows/cols = %d/%d, want 3/3", sh.Rows, sh.Cols)
	}
	if sh.Cells != 3 {
		t.Errorf("cells = %d, want 3", sh.Cells)
	}
	if sh.Formulas != 1 {
		t.Errorf("formulas = %d, want 1", sh.Formulas)
	}
	if info.Totals.Sheets != 1 || info.Totals.Cells != 3 || info.Totals.Formulas != 1 {
		t.Errorf("unexpected totals: %+v", info.Totals)
	}
}

func TestBuildBookInfoEmptySheet(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{"name": "Empty", "cells": {}}`)
	info := BuildBookInfo(wb)
	if len(info.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(info.Sheets))
	}
	sh := info.Sheets[0]
	if sh.UsedRange != "" {
		t.Errorf("usedRange = %q, want empty", sh.UsedRange)
	}
	if sh.Cells != 0 {
		t.Errorf("cells = %d, want 0", sh.Cells)
	}
}

func TestBuildBookInfoTablesAndCharts(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{
		"sheets": [
			{"name": "Data", "cells": {"A1": {"t": "s", "v": "h"}},
			 "tables": [{"range": "A1:B2", "name": "T1"}]}
		],
		"book": {
			"sheets": {"Data": {"cells": {"A1": {"t": "s", "v": "h"}},
			                    "tables": [{"range": "A1:B2", "name": "T1"}]}},
			"charts": [{"id": "c1", "t": "chart", "ct": "col", "sheet": "Data", "anchor": "E2"}]
		}
	}`)
	// book ラッパーがあるので charts は wb.Book から取得される．
	info := BuildBookInfo(wb)
	if info.Totals.Charts != 1 {
		t.Errorf("charts = %d, want 1", info.Totals.Charts)
	}
	if info.Totals.Tables != 1 {
		t.Errorf("tables = %d, want 1", info.Totals.Tables)
	}
	if len(info.Sheets) != 1 || info.Sheets[0].Charts != 1 {
		t.Errorf("per-sheet charts = %+v, want 1", info.Sheets)
	}
}

func TestBuildBookInfoWithChartsExplicit(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{"name": "Data", "cells": {"A1": {"t": "s", "v": "h"}}}`)
	info := BuildBookInfoWithCharts(wb, []Chart{{ID: "c1", Sheet: "Data"}})
	if info.Totals.Charts != 1 {
		t.Errorf("charts = %d, want 1", info.Totals.Charts)
	}
}

func TestTotalsForSheets(t *testing.T) {
	totals := TotalsForSheets([]SheetInfo{
		{Name: "a", Cells: 3, Formulas: 1, Merges: 2, Tables: []Table{{Name: "t1"}, {Name: "t2"}}, Charts: 1, Pictures: 1},
		{Name: "b", Cells: 2, Formulas: 0},
	})
	if totals.Sheets != 2 || totals.Cells != 5 || totals.Formulas != 1 || totals.Merges != 2 ||
		totals.Tables != 2 || totals.Charts != 1 || totals.Pictures != 1 {
		t.Errorf("unexpected totals: %+v", totals)
	}
}

func TestReadBookInfoSourceJSON(t *testing.T) {
	r := strings.NewReader(`{"name": "S1", "cells": {"A1": {"t": "n", "v": 5}}}`)
	wb, charts, err := ReadBookInfoSource(r)
	if err != nil {
		t.Fatalf("ReadBookInfoSource: %v", err)
	}
	if charts != nil {
		t.Errorf("expected no charts for JSON input, got %d", len(charts))
	}
	info := BuildBookInfo(wb)
	if info.Totals.Cells != 1 {
		t.Errorf("cells = %d, want 1", info.Totals.Cells)
	}
}

func TestReadBookInfoSourceXLSX(t *testing.T) {
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(`{"name": "S1", "cells": {"A1": {"t": "s", "v": "hi"}, "A2": {"t": "n", "v": 7}}}`), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	wb, _, err := ReadBookInfoSource(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ReadBookInfoSource: %v", err)
	}
	info := BuildBookInfo(wb)
	if len(info.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(info.Sheets))
	}
	if info.Sheets[0].Name != "S1" {
		t.Errorf("name = %q, want S1", info.Sheets[0].Name)
	}
	if info.Sheets[0].UsedRange != "A1:A2" {
		t.Errorf("usedRange = %q, want A1:A2", info.Sheets[0].UsedRange)
	}
}
