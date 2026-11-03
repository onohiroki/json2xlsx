package json2xlsx

import (
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
