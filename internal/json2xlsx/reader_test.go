package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadWorkbook_XLSX(t *testing.T) {
	jsonIn := `{
	  "name": "Sheet1",
	  "cells": {
	    "A1": {"t":"s","v":"hello"},
	    "B1": {"t":"n","v":42}
	  }
	}`
	var xlsxBuf bytes.Buffer
	if err := Convert(strings.NewReader(jsonIn), &xlsxBuf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	res, err := ReadWorkbook(bytes.NewReader(xlsxBuf.Bytes()), false)
	if err != nil {
		t.Fatalf("ReadWorkbook(xlsx): %v", err)
	}
	if !res.IsXLSX {
		t.Fatal("IsXLSX = false, want true")
	}
	if res.RawData != nil {
		t.Fatal("RawData should be nil for XLSX input")
	}
	wb := res.Workbook
	if wb == nil {
		t.Fatal("Workbook is nil")
	}
	if len(wb.Sheets) == 0 && len(wb.Cells) == 0 {
		t.Fatal("workbook has no sheets or cells")
	}
	// Convert では単一シート形式の Cells が使われる
	if c, ok := wb.Cells["A1"]; !ok {
		t.Error("missing cell A1")
	} else if c.V != "hello" {
		t.Errorf("A1.V = %v, want hello", c.V)
	}
}

func TestReadWorkbook_JSON_SheetJS(t *testing.T) {
	js := `{
	  "name": "Test",
	  "cells": {
	    "A1": {"t":"s","v":"foo"},
	    "B1": {"t":"n","v":99}
	  }
	}`
	res, err := ReadWorkbook(strings.NewReader(js), false)
	if err != nil {
		t.Fatalf("ReadWorkbook: %v", err)
	}
	if res.IsXLSX {
		t.Fatal("IsXLSX = true, want false")
	}
	if res.RawData == nil {
		t.Fatal("RawData should not be nil for JSON input")
	}
	wb := res.Workbook
	if wb.Name != "Test" {
		t.Errorf("Name = %q, want Test", wb.Name)
	}
	if c, ok := wb.Cells["A1"]; !ok {
		t.Error("missing cell A1")
	} else if c.V != "foo" {
		t.Errorf("A1.V = %v, want foo", c.V)
	}
}

func TestReadWorkbook_JSON_2DArray(t *testing.T) {
	js := `[[1,2],[3,4]]`
	res, err := ReadWorkbook(strings.NewReader(js), true)
	if err != nil {
		t.Fatalf("ReadWorkbook(dataJSON): %v", err)
	}
	if res.IsXLSX {
		t.Fatal("IsXLSX = true, want false")
	}
	wb := res.Workbook
	if len(wb.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(wb.Rows))
	}
}

func TestReadWorkbook_JSON_ArrayOfObjects(t *testing.T) {
	js := `[
	  {"name": "Alice", "age": 30},
	  {"name": "Bob",   "age": 25}
	]`
	res, err := ReadWorkbook(strings.NewReader(js), true)
	if err != nil {
		t.Fatalf("ReadWorkbook(dataJSON): %v", err)
	}
	if res.IsXLSX {
		t.Fatal("IsXLSX = true, want false")
	}
	wb := res.Workbook
	if len(wb.Rows) < 2 {
		t.Fatalf("expected at least 2 rows (header + data), got %d", len(wb.Rows))
	}
}

func TestReadWorkbook_JSON_MapOfArrays(t *testing.T) {
	js := `{
	  "x": [1, 2, 3],
	  "y": ["a", "b", "c"]
	}`
	res, err := ReadWorkbook(strings.NewReader(js), true)
	if err != nil {
		t.Fatalf("ReadWorkbook(dataJSON): %v", err)
	}
	if res.IsXLSX {
		t.Fatal("IsXLSX = true, want false")
	}
	wb := res.Workbook
	if len(wb.Rows) != 4 {
		t.Fatalf("expected 4 rows (header + 3 data), got %d", len(wb.Rows))
	}
}

func TestReadWorkbook_BOM(t *testing.T) {
	// BOM 付き SheetJS JSON が正しくパースされることを確認
	bom := []byte{0xEF, 0xBB, 0xBF}
	js := `{"name":"S","cells":{"A1":{"t":"s","v":"bom"}}}`
	data := append(bom, []byte(js)...)

	res, err := ReadWorkbook(bytes.NewReader(data), false)
	if err != nil {
		t.Fatalf("ReadWorkbook(BOM JSON): %v", err)
	}
	if res.IsXLSX {
		t.Fatal("IsXLSX = true, want false")
	}
	wb := res.Workbook
	if c, ok := wb.Cells["A1"]; !ok {
		t.Error("missing cell A1")
	} else if c.V != "bom" {
		t.Errorf("A1.V = %v, want bom", c.V)
	}
}

func TestReadWorkbook_EmptyInput(t *testing.T) {
	_, err := ReadWorkbook(strings.NewReader(""), false)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestReadWorkbook_CorruptedXLSX(t *testing.T) {
	// PK magic byte を持つが中身は不正なバイナリ
	data := []byte{'P', 'K', 0x03, 0x04, 0x00, 0x00, 0x00, 0x00}
	_, err := ReadWorkbook(bytes.NewReader(data), false)
	if err == nil {
		t.Fatal("expected error for corrupted XLSX")
	}
}

func TestReadWorkbook_InvalidJSON(t *testing.T) {
	// XLSX でも JSON でもないバイナリ
	_, err := ReadWorkbook(strings.NewReader("\xFF\xFE\xFD\xFCxxxx"), false)
	if err == nil {
		t.Fatal("expected error for unknown binary input")
	}
}

func TestReadWorkbook_NormalizeDateCells(t *testing.T) {
	// z に日付書式コードを持つセルが t=d に正規化されることを確認
	js := `{
	  "cells": {
	    "A1": {"t":"n","v":45000,"z":"yyyy-mm-dd"}
	  }
	}`
	res, err := ReadWorkbook(strings.NewReader(js), false)
	if err != nil {
		t.Fatalf("ReadWorkbook: %v", err)
	}
	c := res.Workbook.Cells["A1"]
	if c.T != "d" {
		t.Errorf("A1.T = %q, want d (normalizeDateCells should convert)", c.T)
	}
}

func TestReadWorkbook_IsXLSX_WithRawData(t *testing.T) {
	// JSON 入力時に RawData が正しく設定されることを確認
	js := `{"cells":{"A1":{"t":"s","v":"x"}}}`
	res, err := ReadWorkbook(strings.NewReader(js), false)
	if err != nil {
		t.Fatalf("ReadWorkbook: %v", err)
	}
	if res.RawData == nil {
		t.Fatal("RawData is nil for JSON input")
	}
	if !bytes.Contains(res.RawData, []byte(`"A1"`)) {
		t.Fatal("RawData should contain the original JSON content")
	}
	// XLSX 入力時は RawData が nil であることを確認
	var xlsxBuf bytes.Buffer
	if err := Convert(strings.NewReader(js), &xlsxBuf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	res2, err := ReadWorkbook(bytes.NewReader(xlsxBuf.Bytes()), false)
	if err != nil {
		t.Fatalf("ReadWorkbook(xlsx): %v", err)
	}
	if res2.RawData != nil {
		t.Fatal("RawData should be nil for XLSX input")
	}
}
