package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnmarshalWorkbook_SheetJSLibraryShape(t *testing.T) {
	input := `{
	  "SheetNames": ["売上"],
	  "Sheets": {
	    "売上": {
	      "A1": {"t": "s", "v": "商品"},
	      "B1": {"t": "n", "v": 10},
	      "!ref": "A1:B1"
	    }
	  }
	}`
	wb, err := UnmarshalWorkbook([]byte(input), false)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	if len(wb.Sheets) != 1 {
		t.Fatalf("sheets = %d, want 1", len(wb.Sheets))
	}
	if wb.Sheets[0].Name != "売上" {
		t.Fatalf("sheet name = %q", wb.Sheets[0].Name)
	}
	if _, ok := wb.Sheets[0].Cells["!ref"]; ok {
		t.Fatal("!ref should be ignored")
	}
	if got := wb.Sheets[0].Cells["A1"].V; got != "商品" {
		t.Fatalf("A1 = %v", got)
	}
}

func TestNormalizeSheetJSLibraryShape_WithCellsWrapper(t *testing.T) {
	input := `{
	  "SheetNames": ["S1", "S2"],
	  "Sheets": {
	    "S1": {"cells": {"A1": {"t": "s", "v": "a"}}},
	    "S2": {"A1": {"t": "s", "v": "b"}}
	  }
	}`
	normalized, ok := NormalizeSheetJSLibraryShape([]byte(input))
	if !ok {
		t.Fatal("expected normalization")
	}
	if !strings.Contains(string(normalized), `"name":"S1"`) || !strings.Contains(string(normalized), `"name":"S2"`) {
		t.Fatalf("normalized = %s", normalized)
	}
}

func TestUnmarshalWorkbook_StripsInternalKeys(t *testing.T) {
	input := `{"name":"S","cells":{"!ref":"A1:B1","A1":{"t":"s","v":"a"}}}`
	wb, err := UnmarshalWorkbook([]byte(input), false)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	if _, ok := wb.Sheets[0].Cells["!ref"]; ok {
		t.Fatal("!ref should be removed")
	}
	if wb.Sheets[0].Cells["A1"].V != "a" {
		t.Fatalf("A1 = %v", wb.Sheets[0].Cells["A1"].V)
	}
}

func TestUnmarshalWorkbook_DataJSON_PreservesIntegerPrecision(t *testing.T) {
	wb, err := UnmarshalWorkbook([]byte(`[[9007199254740993]]`), true)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	got := wb.Rows[0][0]
	if got != int64(9007199254740993) {
		t.Fatalf("value = %v (%T), want int64", got, got)
	}
}

func TestUnmarshalWorkbook_DataJSON_ObjectArrayPreservesIntegerPrecision(t *testing.T) {
	wb, err := UnmarshalWorkbook([]byte(`[{"n":9007199254740993}]`), true)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	got := wb.Rows[1][0]
	if got != int64(9007199254740993) {
		t.Fatalf("value = %v (%T), want int64", got, got)
	}
}

func TestUnmarshalWorkbook_DataJSON_MapOfArraysPreservesIntegerPrecision(t *testing.T) {
	wb, err := UnmarshalWorkbook([]byte(`{"n":[9007199254740993]}`), true)
	if err != nil {
		t.Fatalf("UnmarshalWorkbook: %v", err)
	}
	got := wb.Rows[1][0]
	if got != int64(9007199254740993) {
		t.Fatalf("value = %v (%T), want int64", got, got)
	}
}

func TestUnmarshalWorkbook_BOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	if _, err := UnmarshalWorkbook(append(bom, []byte(`{"cells":{"A1":{"t":"s","v":"x"}}}`)...), false); err != nil {
		t.Fatalf("sheetjs BOM: %v", err)
	}
	if _, err := UnmarshalWorkbook(append(bom, []byte(`[["a"]]`)...), true); err != nil {
		t.Fatalf("data BOM: %v", err)
	}
}

func TestParseTableSpec_BOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	if _, _, err := ParseTableSpec(append(bom, []byte(`{"tables":[{"rows":[[1]]}]}`)...)); err != nil {
		t.Fatalf("table BOM: %v", err)
	}
}

func TestReadWorkbookWithFormat_ShortJSON(t *testing.T) {
	res, err := ReadWorkbookWithFormat(bytes.NewReader([]byte(`[]`)), true, false)
	if err != nil {
		t.Fatalf("ReadWorkbookWithFormat([]): %v", err)
	}
	if res.Workbook == nil {
		t.Fatal("workbook is nil")
	}
	res, err = ReadWorkbookWithFormat(bytes.NewReader([]byte(`{}`)), false, false)
	if err != nil {
		t.Fatalf("ReadWorkbookWithFormat({}): %v", err)
	}
	if res.Workbook == nil {
		t.Fatal("workbook is nil")
	}
}
