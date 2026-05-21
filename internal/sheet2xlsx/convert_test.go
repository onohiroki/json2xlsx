package sheet2xlsx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func convertAndOpen(t *testing.T, jsonStr string) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(jsonStr), &buf, ""); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	return f
}

func TestBasicCellObject(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "hello"},
			"B1": {"t": "n", "v": 42},
			"C1": {"t": "b", "v": true},
			"D1": {"t": "f", "f": "B1*2"}
		}
	}`
	f := convertAndOpen(t, js)
	defer f.Close()

	if got, _ := f.GetCellValue("S1", "A1"); got != "hello" {
		t.Errorf("A1 = %q, want hello", got)
	}
	if got, _ := f.GetCellValue("S1", "B1"); got != "42" {
		t.Errorf("B1 = %q, want 42", got)
	}
	if got, _ := f.GetCellValue("S1", "C1"); got != "TRUE" {
		t.Errorf("C1 = %q, want TRUE", got)
	}
	if got, _ := f.GetCellFormula("S1", "D1"); got != "B1*2" {
		t.Errorf("D1 formula = %q, want B1*2", got)
	}
}

func TestRowsAoA(t *testing.T) {
	js := `{
		"name": "Data",
		"rows": [
			["name", "qty"],
			["apple", 3],
			["banana", 5]
		]
	}`
	f := convertAndOpen(t, js)
	defer f.Close()

	if got, _ := f.GetCellValue("Data", "A1"); got != "name" {
		t.Errorf("A1 = %q", got)
	}
	if got, _ := f.GetCellValue("Data", "B3"); got != "5" {
		t.Errorf("B3 = %q", got)
	}
}

func TestStylesAndNumFmt(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t": "n", "v": 1234567, "s": 1}
		},
		"styles": [
			{
				"id": 1,
				"numFmt": "#,##0",
				"fill": {"type":"pattern","pattern":1,"color":["#FFFF00"]},
				"border": [{"style":"thin","color":"#000000"}]
			}
		]
	}`
	f := convertAndOpen(t, js)
	defer f.Close()

	// 整形済み表示値を確認
	got, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1,234,567" {
		t.Errorf("A1 formatted = %q, want 1,234,567", got)
	}
}

func TestNewline(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t": "s", "v": "line1\nline2", "s": 1}
		},
		"styles": [
			{"id": 1, "alignment": {"wrapText": true}}
		]
	}`
	f := convertAndOpen(t, js)
	defer f.Close()
	got, _ := f.GetCellValue("Sheet1", "A1")
	if !strings.Contains(got, "\n") {
		t.Errorf("expected newline preserved, got %q", got)
	}
}

func TestMultipleSheets(t *testing.T) {
	js := `{
		"sheets": [
			{"name": "First", "cells": {"A1": {"t":"s","v":"one"}}},
			{"name": "Second", "cells": {"A1": {"t":"s","v":"two"}}}
		]
	}`
	f := convertAndOpen(t, js)
	defer f.Close()
	if v, _ := f.GetCellValue("First", "A1"); v != "one" {
		t.Errorf("First!A1=%q", v)
	}
	if v, _ := f.GetCellValue("Second", "A1"); v != "two" {
		t.Errorf("Second!A1=%q", v)
	}
}

func TestMergeAndDimensions(t *testing.T) {
	js := `{
		"cells": {"A1": {"t":"s","v":"merged"}},
		"merges": [{"range":"A1:B1"}],
		"cols": [{"col":"A","width":20}],
		"rowDims": [{"row":1,"height":40}]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ""); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	merges, err := f.GetMergeCells("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge, got %d", len(merges))
	}

	w, _ := f.GetColWidth("Sheet1", "A")
	if w != 20 {
		t.Errorf("col A width = %v, want 20", w)
	}
	h, _ := f.GetRowHeight("Sheet1", 1)
	if h != 40 {
		t.Errorf("row 1 height = %v, want 40", h)
	}
}
