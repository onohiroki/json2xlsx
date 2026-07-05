package json2xlsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// roundTrip は JSON 文字列を Convert → ToJSON し、結果の Workbook を返す。
func roundTrip(t *testing.T, jsonStr string) Workbook {
	t.Helper()
	return roundTripWithOptions(t, jsonStr, ToJSONOptions{DateMode: DateModeSerial})
}

func roundTripWithOptions(t *testing.T, jsonStr string, opts ToJSONOptions) Workbook {
	t.Helper()
	var xlsx bytes.Buffer
	if err := Convert(strings.NewReader(jsonStr), &xlsx, ConvertOptions{}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var out bytes.Buffer
	if err := ToJSONWithOptions(bytes.NewReader(xlsx.Bytes()), &out, opts); err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var wb Workbook
	if err := json.Unmarshal(out.Bytes(), &wb); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, out.String())
	}
	return wb
}

func cellText(c Cell) string {
	return fmt.Sprint(c.V)
}

func TestToJSON_BasicTypes(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "hello"},
			"B1": {"t": "n", "v": 42},
			"C1": {"t": "b", "v": true},
			"D1": {"t": "f", "f": "B1*2", "v": 84}
		}
	}`
	wb := roundTrip(t, js)
	if wb.Cells == nil {
		t.Fatalf("Cells nil; wb=%+v", wb)
	}
	if got := wb.Cells["A1"]; got.T != "s" || got.V != "hello" {
		t.Errorf("A1=%+v", got)
	}
	if got := wb.Cells["B1"]; got.T != "n" {
		t.Errorf("B1=%+v", got)
	}
	if got := wb.Cells["C1"]; got.T != "b" {
		t.Errorf("C1=%+v", got)
	}
	if got := wb.Cells["D1"]; got.T != "f" || got.F != "B1*2" {
		t.Errorf("D1=%+v", got)
	}
}

func TestToJSON_EmptyCellsSkipped(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "x"},
			"C3": {"t": "s", "v": "y"}
		}
	}`
	wb := roundTrip(t, js)
	if _, ok := wb.Cells["B1"]; ok {
		t.Errorf("expected B1 absent")
	}
	if _, ok := wb.Cells["A2"]; ok {
		t.Errorf("expected A2 absent")
	}
	if _, ok := wb.Cells["A1"]; !ok {
		t.Errorf("expected A1 present")
	}
	if _, ok := wb.Cells["C3"]; !ok {
		t.Errorf("expected C3 present")
	}
}

func TestToJSON_NewlineNormalized(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "line1\r\nline2\rline3"}
		}
	}`
	wb := roundTrip(t, js)
	v, _ := wb.Cells["A1"].V.(string)
	if strings.Contains(v, "\r") {
		t.Errorf("expected CR removed: %q", v)
	}
	if !strings.Contains(v, "line1\nline2\nline3") {
		t.Errorf("expected LF normalized: %q", v)
	}
}

func TestToJSON_MergesAndCols(t *testing.T) {
	js := `{
		"cells": {"A1": {"t":"s","v":"merged"}},
		"merges": [{"range":"A1:B1"}],
		"cols": [{"col":"A","width":25}],
		"rowDims": [{"row":1,"height":40}]
	}`
	wb := roundTrip(t, js)
	if len(wb.Merges) != 1 || wb.Merges[0].Range != "A1:B1" {
		t.Errorf("merges=%+v", wb.Merges)
	}
	foundCol := false
	for _, c := range wb.Cols {
		if c.Col == "A" && c.Width > 20 {
			foundCol = true
		}
	}
	if !foundCol {
		t.Errorf("col A width missing: %+v", wb.Cols)
	}
	foundRow := false
	for _, r := range wb.RowDims {
		if r.Row == 1 && r.Height > 30 {
			foundRow = true
		}
	}
	if !foundRow {
		t.Errorf("row 1 height missing: %+v", wb.RowDims)
	}
}

func TestToJSON_StylesSharedAndNumFmt(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t":"n","v":1234567,"s":1},
			"A2": {"t":"n","v":2345678,"s":1}
		},
		"styles": [
			{"id":1,"numFmt":"#,##0","fill":{"type":"pattern","pattern":1,"color":["#FFFF00"]}}
		]
	}`
	wb := roundTrip(t, js)
	if len(wb.Styles) != 1 {
		t.Fatalf("expected 1 style, got %d: %+v", len(wb.Styles), wb.Styles)
	}
	if wb.Cells["A1"].S == 0 || wb.Cells["A1"].S != wb.Cells["A2"].S {
		t.Errorf("expected shared style id; A1.S=%d A2.S=%d", wb.Cells["A1"].S, wb.Cells["A2"].S)
	}
	s := wb.Styles[0]
	if s.NumFmt != "#,##0" {
		t.Errorf("numFmt=%q", s.NumFmt)
	}
	if s.Fill == nil || len(s.Fill.Color) == 0 || !strings.EqualFold(s.Fill.Color[0], "#FFFF00") {
		t.Errorf("fill=%+v", s.Fill)
	}
}

func TestToJSON_MultipleSheets(t *testing.T) {
	js := `{
		"sheets": [
			{"name": "First", "cells": {"A1": {"t":"s","v":"one"}}},
			{"name": "Second", "cells": {"A1": {"t":"s","v":"two"}}}
		]
	}`
	wb := roundTrip(t, js)
	if len(wb.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d (wb=%+v)", len(wb.Sheets), wb)
	}
	if wb.Sheets[0].Name != "First" || wb.Sheets[1].Name != "Second" {
		t.Errorf("sheet names: %q %q", wb.Sheets[0].Name, wb.Sheets[1].Name)
	}
	if wb.Sheets[0].Cells["A1"].V != "one" {
		t.Errorf("First!A1=%+v", wb.Sheets[0].Cells["A1"])
	}
}

func TestToJSON_DateCells_DefaultSerial(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t": "n", "v": 0.3784722222222222, "s": 1},
			"B1": {"t": "n", "v": 0.7847222222222222, "s": 1}
		},
		"styles": [
			{"id": 1, "numFmt": "h:mm"}
		]
	}`
	wb := roundTrip(t, js)
	if got := cellText(wb.Cells["A1"]); got != "0.3784722222222222" {
		t.Fatalf("A1=%q, want serial", got)
	}
	if got := cellText(wb.Cells["B1"]); got != "0.7847222222222222" {
		t.Fatalf("B1=%q, want serial", got)
	}
}

func TestToJSON_DateCells_DisplayOption(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t": "n", "v": 0.3784722222222222, "s": 1},
			"B1": {"t": "n", "v": 0.7847222222222222, "s": 1}
		},
		"styles": [
			{"id": 1, "numFmt": "h:mm"}
		]
	}`
	wb := roundTripWithOptions(t, js, ToJSONOptions{DateMode: DateModeDisplay})
	if got := cellText(wb.Cells["A1"]); got != "9:05" && got != "09:05" {
		t.Fatalf("A1=%q, want display time", got)
	}
	if got := cellText(wb.Cells["B1"]); got != "18:50" {
		t.Fatalf("B1=%q, want display time", got)
	}
}

func TestToJSON_DateCells_RFC3339Option(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t": "n", "v": 0.3784722222222222, "s": 1},
			"B1": {"t": "n", "v": 0.7847222222222222, "s": 1}
		},
		"styles": [
			{"id": 1, "numFmt": "h:mm"}
		]
	}`
	wb := roundTripWithOptions(t, js, ToJSONOptions{DateMode: DateModeRFC3339})
	if got := cellText(wb.Cells["A1"]); got != "1899-12-30T09:05:00Z" {
		t.Fatalf("A1=%q, want RFC3339", got)
	}
	if got := cellText(wb.Cells["B1"]); got != "1899-12-30T18:50:00Z" {
		t.Fatalf("B1=%q, want RFC3339", got)
	}
}

func TestToJSON_FreezePanes_ReadDirect(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetCellValue("Sheet1", "A1", "Name")
	f.SetCellValue("Sheet1", "B1", "Score")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", 95)

	panes := excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomRight",
	}
	if err := f.SetPanes("Sheet1", &panes); err != nil {
		t.Fatalf("SetPanes: %v", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var out bytes.Buffer
	if err := ToJSONWithOptions(bytes.NewReader(buf.Bytes()), &out, ToJSONOptions{DateMode: DateModeSerial}); err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var wb Workbook
	if err := json.Unmarshal(out.Bytes(), &wb); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if wb.Freeze == nil {
		t.Fatal("freeze is nil, expected row=1")
	}
	if wb.Freeze.Row != 1 {
		t.Errorf("freeze.row = %d, want 1", wb.Freeze.Row)
	}
	if wb.Freeze.Col != 0 {
		t.Errorf("freeze.col = %d, want 0", wb.Freeze.Col)
	}
}
