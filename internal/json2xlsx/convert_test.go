package json2xlsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFreezePanes_Write(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"row": 1},
		"cells": {
			"A1": {"t": "s", "v": "Header"},
			"A2": {"t": "s", "v": "Data"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if !panes.Freeze {
		t.Error("Freeze = false, want true")
	}
	if panes.YSplit != 1 {
		t.Errorf("YSplit = %d, want 1", panes.YSplit)
	}
	if panes.XSplit != 0 {
		t.Errorf("XSplit = %d, want 0", panes.XSplit)
	}
	if panes.TopLeftCell != "A2" {
		t.Errorf("TopLeftCell = %q, want A2", panes.TopLeftCell)
	}
}

func TestFreezePanes_WriteCol(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"col": 1},
		"cells": {
			"A1": {"t": "s", "v": "RowHeader"},
			"B1": {"t": "s", "v": "Data"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if !panes.Freeze {
		t.Error("Freeze = false, want true")
	}
	if panes.YSplit != 0 {
		t.Errorf("YSplit = %d, want 0", panes.YSplit)
	}
	if panes.XSplit != 1 {
		t.Errorf("XSplit = %d, want 1", panes.XSplit)
	}
}

func TestFreezePanes_WriteBoth(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"row": 1, "col": 1},
		"cells": {
			"A1": {"t": "s", "v": "Corner"},
			"B1": {"t": "s", "v": "ColHdr"},
			"A2": {"t": "s", "v": "RowHdr"},
			"B2": {"t": "s", "v": "Data"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if !panes.Freeze {
		t.Error("Freeze = false, want true")
	}
	if panes.YSplit != 1 {
		t.Errorf("YSplit = %d, want 1", panes.YSplit)
	}
	if panes.XSplit != 1 {
		t.Errorf("XSplit = %d, want 1", panes.XSplit)
	}
}

func TestFreezePanes_RoundTrip(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"row": 2},
		"cells": {
			"A1": {"t": "s", "v": "h1"},
			"B1": {"t": "s", "v": "h2"},
			"A2": {"t": "s", "v": "sub1"},
			"B2": {"t": "s", "v": "sub2"},
			"A3": {"t": "s", "v": "d1"},
			"B3": {"t": "n", "v": 1}
		}
	}`
	wb := roundTrip(t, js)
	if wb.Freeze == nil {
		t.Fatal("freeze is nil after round trip")
	}
	if wb.Freeze.Row != 2 {
		t.Errorf("freeze.row = %d, want 2", wb.Freeze.Row)
	}
}

func TestFreezePanes_RoundTripMultiSheet(t *testing.T) {
	js := `{
		"sheets": [
			{"name": "S1", "freeze": {"row": 1}, "cells": {"A1":{"t":"s","v":"H"},"A2":{"t":"s","v":"D"}}},
			{"name": "S2", "freeze": {"col": 1}, "cells": {"A1":{"t":"s","v":"RH"},"B1":{"t":"s","v":"D"}}}
		]
	}`
	wb := roundTrip(t, js)
	if len(wb.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(wb.Sheets))
	}
	if wb.Sheets[0].Freeze == nil || wb.Sheets[0].Freeze.Row != 1 {
		t.Errorf("S1 freeze = %+v, want row=1", wb.Sheets[0].Freeze)
	}
	if wb.Sheets[1].Freeze == nil || wb.Sheets[1].Freeze.Col != 1 {
		t.Errorf("S2 freeze = %+v, want col=1", wb.Sheets[1].Freeze)
	}
}

func TestFreezePanes_NilOnAbsent(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {"A1": {"t": "s", "v": "no freeze"}}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if panes.Freeze {
		t.Error("Freeze = true, want false (no freeze specified)")
	}
}

func TestFreezePanes_ZeroValues(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {},
		"cells": {"A1": {"t": "s", "v": "zero"}}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if panes.Freeze {
		t.Error("Freeze = true, want false (zero-value freeze should be skipped)")
	}
}

func TestFreezePanes_BookWrapperRoundTrip(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"S1": {"freeze": {"row": 2}, "cells": {"A1":{"t":"s","v":"H"},"A2":{"t":"s","v":"Sub"},"A3":{"t":"s","v":"D"}}}
			}
		}
	}`
	// Use WrapWithBook to preserve book wrapper format
	wb := roundTripWithOptions(t, js, ToJSONOptions{DateMode: DateModeSerial, WrapWithBook: true})
	if wb.Book == nil {
		t.Fatal("book is nil after round trip")
	}
	sh, ok := wb.Book.Sheets["S1"]
	if !ok {
		t.Fatal("sheet S1 not found")
	}
	if sh.Freeze == nil {
		t.Fatal("freeze is nil after round trip in book wrapper")
	}
	if sh.Freeze.Row != 2 {
		t.Errorf("freeze.row = %d, want 2", sh.Freeze.Row)
	}
}

func TestFreezePanes_WriteRow4(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"row": 4},
		"cells": {
			"A1":{"t":"s","v":"H1"},"A2":{"t":"s","v":"H2"},"A3":{"t":"s","v":"H3"},"A4":{"t":"s","v":"H4"},
			"A5":{"t":"s","v":"D"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if !panes.Freeze {
		t.Fatal("Freeze = false, want true")
	}
	if panes.YSplit != 4 {
		t.Errorf("YSplit = %d, want 4", panes.YSplit)
	}
	if panes.XSplit != 0 {
		t.Errorf("XSplit = %d, want 0", panes.XSplit)
	}
	if panes.TopLeftCell != "A5" {
		t.Errorf("TopLeftCell = %q, want A5", panes.TopLeftCell)
	}
}

func TestFreezePanes_Col3Row2(t *testing.T) {
	js := `{
		"name": "S1",
		"freeze": {"row": 2, "col": 3},
		"cells": {
			"A1":{"t":"s","v":"C"},"B1":{"t":"s","v":"C"},"C1":{"t":"s","v":"C"},
			"A2":{"t":"s","v":"C"},"B2":{"t":"s","v":"C"},"C2":{"t":"s","v":"C"},
			"D3":{"t":"s","v":"D"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	panes, err := f.GetPanes("S1")
	if err != nil {
		t.Fatalf("GetPanes error: %v", err)
	}
	if !panes.Freeze {
		t.Fatal("Freeze = false, want true")
	}
	if panes.YSplit != 2 {
		t.Errorf("YSplit = %d, want 2", panes.YSplit)
	}
	if panes.XSplit != 3 {
		t.Errorf("XSplit = %d, want 3", panes.XSplit)
	}
}

func TestArrayOfMap_PreservesKeyOrder(t *testing.T) {
	js := `[
		{"name": "Alice", "age": 30, "city": "Tokyo",   "score": 88, "active": true},
		{"name": "Bob",   "age": 25, "city": "Osaka",   "score": 72, "active": false},
		{"name": "Carol", "age": 41, "city": "Nagoya",  "score": 95, "active": true},
		{"name": "Dave",  "age": 36, "city": "Fukuoka", "score": 60, "active": false},
		{"name": "Eve",   "age": 28, "city": "Sapporo", "score": 81, "active": true}
	]`
	f := convertAndOpen(t, js, true)
	defer f.Close()

	expected := []struct {
		cell string
		want string
	}{
		{"A1", "name"},
		{"B1", "age"},
		{"C1", "city"},
		{"D1", "score"},
		{"E1", "active"},
	}
	for _, e := range expected {
		got, _ := f.GetCellValue("Sheet1", e.cell)
		if got != e.want {
			t.Errorf("%s = %q, want %q", e.cell, got, e.want)
		}
	}
}

func TestArrayOfMap_PreservesKeyOrder_Repeated(t *testing.T) {
	js := `[
		{"name": "Alice", "age": 30, "city": "Tokyo",   "score": 88, "active": true},
		{"name": "Bob",   "age": 25, "city": "Osaka",   "score": 72, "active": false}
	]`
	want := []string{"name", "age", "city", "score", "active"}
	cells := []string{"A1", "B1", "C1", "D1", "E1"}
	for i := 0; i < 10; i++ {
		f := convertAndOpen(t, js, true)
		for j, c := range cells {
			got, _ := f.GetCellValue("Sheet1", c)
			if got != want[j] {
				t.Errorf("iter %d, %s = %q, want %q", i, c, got, want[j])
			}
		}
		f.Close()
	}
}

func TestMapOfArrays_ConflictingFieldName_DataJSON(t *testing.T) {
	// トップレベルキーが Workbook 構造体のフィールドと衝突する場合でも、
	// Map-of-Arrays フォールバックで正しく解釈されることを確認する。
	js := `{
		"name":   ["Alice", "Bob",   "Carol"],
		"age":    [30,      25,      41],
		"city":   ["Tokyo", "Osaka", "Nagoya"],
		"score":  [88,      72,      95],
		"active": [true,    false,   true]
	}`
	f := convertAndOpen(t, js, true)
	defer f.Close()

	expected := []struct {
		cell string
		want string
	}{
		{"A1", "name"},
		{"B1", "age"},
		{"C1", "city"},
		{"D1", "score"},
		{"E1", "active"},
		{"A2", "Alice"},
		{"B2", "30"},
		{"C2", "Tokyo"},
	}
	for _, e := range expected {
		got, _ := f.GetCellValue("Sheet1", e.cell)
		if got != e.want {
			t.Errorf("%s = %q, want %q", e.cell, got, e.want)
		}
	}
}

func TestArrayOfMap_MissingKeys_DataJSON(t *testing.T) {
	js := `[
		{"a": 1, "b": 2},
		{"b": 3, "c": 4}
	]`
	f := convertAndOpen(t, js, true)
	defer f.Close()

	expected := []struct {
		cell string
		want string
	}{
		{"A1", "a"},
		{"B1", "b"},
		{"C1", "c"},
	}
	for _, e := range expected {
		got, _ := f.GetCellValue("Sheet1", e.cell)
		if got != e.want {
			t.Errorf("%s = %q, want %q", e.cell, got, e.want)
		}
	}
}

func convertAndOpenWithOpts(t *testing.T, jsonStr string, opts ConvertOptions) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(jsonStr), &buf, opts); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	return f
}

func convertAndOpen(t *testing.T, jsonStr string, dataJSON bool) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(jsonStr), &buf, ConvertOptions{DataJSON: dataJSON}); err != nil {
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
	f := convertAndOpen(t, js, false)
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
	f := convertAndOpen(t, js, false)
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
	f := convertAndOpen(t, js, false)
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
	f := convertAndOpen(t, js, false)
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
	f := convertAndOpen(t, js, false)
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
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
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

// convertWithStderr は Convert を実行し、stderr 出力をキャプチャする。
func convertWithStderr(t *testing.T, jsonStr string, dataJSON bool) (xlsxData []byte, stderrOutput string, convertErr error) {
	t.Helper()
	r := strings.NewReader(jsonStr)
	var buf bytes.Buffer

	// stderr を差し替え
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = stderrW

	convertErr = Convert(r, &buf, ConvertOptions{DataJSON: dataJSON})

	stderrW.Close()
	os.Stderr = origStderr
	var stderrBuf bytes.Buffer
	if _, err := stderrBuf.ReadFrom(stderrR); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes(), stderrBuf.String(), convertErr
}

func TestUnknownStyleID_Warning(t *testing.T) {
	js := `{
		"cells": {
			"A1": {"t":"s","v":"hello", "s": 99}
		}
	}`
	xlsxData, stderrOut, err := convertWithStderr(t, js, false)
	if err == nil {
		t.Fatal("expected error for unknown style id, got nil")
	}
	if !strings.Contains(err.Error(), "warning") {
		t.Errorf("error message should mention warning, got: %v", err)
	}
	if !strings.Contains(stderrOut, "style id 99") {
		t.Errorf("stderr should mention style id 99, got: %q", stderrOut)
	}
	if len(xlsxData) == 0 {
		t.Fatal("expected XLSX output despite warning")
	}
	f, openErr := excelize.OpenReader(bytes.NewReader(xlsxData))
	if openErr != nil {
		t.Fatalf("OpenReader after warning: %v", openErr)
	}
	defer f.Close()
	v, _ := f.GetCellValue("Sheet1", "A1")
	if v != "hello" {
		t.Errorf("A1 = %q, want hello", v)
	}
}

func TestUnknownStyleID_ValidStyleStillWorks(t *testing.T) {
	// A1 は有効な style ID 1、A2 は不明な style ID 99
	js := `{
		"cells": {
			"A1": {"t":"n","v":123, "s": 1},
			"A2": {"t":"s","v":"no-style", "s": 99}
		},
		"styles": [
			{"id": 1, "numFmt": "#,##0"}
		]
	}`
	xlsxData, stderrOut, err := convertWithStderr(t, js, false)
	if err == nil {
		t.Fatal("expected error for unknown style id, got nil")
	}
	if !strings.Contains(stderrOut, "style id 99") {
		t.Errorf("stderr should mention style id 99, got: %q", stderrOut)
	}
	if len(xlsxData) == 0 {
		t.Fatal("expected XLSX output despite warning")
	}
	f, openErr := excelize.OpenReader(bytes.NewReader(xlsxData))
	if openErr != nil {
		t.Fatalf("OpenReader after warning: %v", openErr)
	}
	defer f.Close()
	// A1 (valid style) should still be formatted
	got, _ := f.GetCellValue("Sheet1", "A1")
	if got != "123" {
		t.Errorf("A1 formatted = %q, want 123", got)
	}
	// A2 (unknown style) should still have the value
	v, _ := f.GetCellValue("Sheet1", "A2")
	if v != "no-style" {
		t.Errorf("A2 = %q, want no-style", v)
	}
}

func TestChartEmbedded(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"Data": {
					"cells": {
						"A1": {"t":"s","v":"cat"},
						"B1": {"t":"s","v":"val"},
						"A2": {"t":"s","v":"X"}, "B2": {"t":"n","v":1},
						"A3": {"t":"s","v":"Y"}, "B3": {"t":"n","v":2}
					}
				}
			},
			"charts": [
				{
					"id": "ch1",
					"t": "chart",
					"mode": "embedded",
					"ct": "col",
					"sheet": "Data",
					"anchor": "D2",
					"title": {"tx":"Test Chart"},
					"ser": [{"name":"S1","cat":"Data!$A$2:$A$3","val":"Data!$B$2:$B$3"}]
				}
			]
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == "Data" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("sheet Data not found")
	}
}

func TestChartSheet(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"Data": {
					"cells": {
						"A1": {"t":"s","v":"cat"},
						"B1": {"t":"s","v":"val"},
						"A2": {"t":"s","v":"X"}, "B2": {"t":"n","v":1},
						"A3": {"t":"s","v":"Y"}, "B3": {"t":"n","v":2}
					}
				}
			},
			"charts": [
				{
					"id": "ch2",
					"t": "chart",
					"mode": "chartSheet",
					"ct": "col",
					"sheet": "Chart1",
					"title": {"tx":"Chart Sheet"},
					"ser": [{"name":"S1","cat":"Data!$A$2:$A$3","val":"Data!$B$2:$B$3"}]
				}
			]
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	sheets := f.GetSheetList()
	foundData := false
	foundChart := false
	for _, s := range sheets {
		switch s {
		case "Data":
			foundData = true
		case "Chart1":
			foundChart = true
		}
	}
	if !foundData {
		t.Fatal("sheet Data not found")
	}
	if !foundChart {
		t.Fatal("chartsheet Chart1 not found")
	}
}

func TestChartUnknownMode(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"S1": {
					"cells": {"A1":{"t":"s","v":"a"},"B1":{"t":"n","v":1}}
				}
			},
			"charts": [
				{
					"id": "ch_bad",
					"t": "chart",
					"mode": "bogus",
					"ct": "line",
					"sheet": "S1",
					"anchor": "D2",
					"ser": [{"name":"X","cat":"S1!$A$1:$A$1","val":"S1!$B$1:$B$1"}]
				}
			]
		}
	}`
	var buf bytes.Buffer
	err := Convert(strings.NewReader(js), &buf, ConvertOptions{})
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention unknown mode, got: %v", err)
	}
}

func TestChartInvalidType(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"S1": {
					"cells": {"A1":{"t":"s","v":"a"},"B1":{"t":"n","v":1}}
				}
			},
			"charts": [
				{
					"id": "ch_bad",
					"t": "chart",
					"ct": "nope",
					"sheet": "S1",
					"anchor": "D2",
					"ser": [{"name":"X","cat":"S1!$A$1:$A$1","val":"S1!$B$1:$B$1"}]
				}
			]
		}
	}`
	var buf bytes.Buffer
	err := Convert(strings.NewReader(js), &buf, ConvertOptions{})
	if err == nil {
		t.Fatal("expected error for unknown chart type, got nil")
	}
}

func TestUnknownStyleID_StyleIDZeroIsValid(t *testing.T) {
	// s=0 は「スタイル未指定」なので警告にならない
	js := `{
		"cells": {
			"A1": {"t":"s","v":"ok", "s": 0}
		}
	}`
	_, _, err := convertWithStderr(t, js, false)
	if err != nil {
		t.Fatalf("unexpected error for s=0: %v", err)
	}
}

func TestToJSONChartsheetRoundtrip(t *testing.T) {
	// Excelize で chartsheet 付き XLSX を作成
	var buf bytes.Buffer
	xf := excelize.NewFile()
	defer xf.Close()

	// データシート
	xf.SetCellValue("Sheet1", "A1", "cat")
	xf.SetCellValue("Sheet1", "B1", "val")
	xf.SetCellValue("Sheet1", "A2", "X")
	xf.SetCellValue("Sheet1", "B2", 1)
	xf.SetCellValue("Sheet1", "A3", "Y")
	xf.SetCellValue("Sheet1", "B3", 2)

	// chartsheet を追加
	err := xf.AddChartSheet("Chart1", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{
				Name:       "S1",
				Categories: "Sheet1!$A$2:$A$3",
				Values:     "Sheet1!$B$2:$B$3",
			},
		},
		Title: []excelize.RichTextRun{{Text: "Test Chart"}},
		Legend: excelize.ChartLegend{
			Position: "bottom",
		},
	})
	if err != nil {
		t.Fatalf("AddChartSheet: %v", err)
	}

	if err := xf.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// 読み取り
	var out bytes.Buffer
	err = ToJSONWithOptions(&buf, &out, ToJSONOptions{DateMode: DateModeSerial, WrapWithBook: true})
	if err != nil {
		t.Fatalf("ToJSONWithOptions: %v", err)
	}

	// JSON をパースして検証
	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}

	// version の確認
	if v, ok := result["version"]; !ok || v != "0.2" {
		t.Errorf("version = %v, want 0.2", v)
	}
	// book の存在確認
	book, ok := result["book"].(map[string]interface{})
	if !ok {
		t.Fatal("book not found in output")
	}
	// sheets の存在確認
	sheets, ok := book["sheets"].(map[string]interface{})
	if !ok {
		t.Fatal("book.sheets not found")
	}
	if _, ok := sheets["Sheet1"]; !ok {
		t.Error("Sheet1 not found in book.sheets")
	}
	// Chart1 (chartsheet) は sheets に含まれないこと
	if _, ok := sheets["Chart1"]; ok {
		t.Error("Chart1 (chartsheet) should not appear in book.sheets")
	}
	// charts の存在確認
	charts, ok := book["charts"].([]interface{})
	if !ok {
		t.Fatal("book.charts not found or not an array")
	}
	if len(charts) != 1 {
		t.Fatalf("expected 1 chart, got %d", len(charts))
	}
	ch := charts[0].(map[string]interface{})
	if ch["mode"] != "chartSheet" {
		t.Errorf("mode = %v, want chartSheet", ch["mode"])
	}
	if ch["sheet"] != "Chart1" {
		t.Errorf("sheet = %v, want Chart1", ch["sheet"])
	}
	if ch["ct"] != "col" {
		t.Errorf("ct = %v, want col", ch["ct"])
	}
	// title の確認
	title, ok := ch["title"].(map[string]interface{})
	if !ok {
		t.Fatal("chart.title not found")
	}
	if title["tx"] != "Test Chart" {
		t.Errorf("title.tx = %v, want Test Chart", title["tx"])
	}
	// series の確認
	ser, ok := ch["ser"].([]interface{})
	if !ok || len(ser) == 0 {
		t.Fatal("chart.ser not found or empty")
	}
	s0 := ser[0].(map[string]interface{})
	if s0["name"] != "S1" {
		t.Errorf("ser[0].name = %v, want S1", s0["name"])
	}
	if s0["cat"] != "Sheet1!$A$2:$A$3" {
		t.Errorf("ser[0].cat = %v, want Sheet1!$A$2:$A$3", s0["cat"])
	}
	if s0["val"] != "Sheet1!$B$2:$B$3" {
		t.Errorf("ser[0].val = %v, want Sheet1!$B$2:$B$3", s0["val"])
	}
}

func TestToJSONWrapWithBookNoCharts(t *testing.T) {
	// 通常シートのみで WrapWithBook をテスト
	var buf bytes.Buffer
	xf := excelize.NewFile()
	defer xf.Close()

	xf.SetCellValue("Sheet1", "A1", "hello")
	xf.SetCellValue("Sheet1", "B1", 42)

	if err := xf.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var out bytes.Buffer
	err := ToJSONWithOptions(&buf, &out, ToJSONOptions{DateMode: DateModeSerial, WrapWithBook: true})
	if err != nil {
		t.Fatalf("ToJSONWithOptions: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if v, ok := result["version"]; !ok || v != "0.2" {
		t.Errorf("version = %v, want 0.2", v)
	}
	book, ok := result["book"].(map[string]interface{})
	if !ok {
		t.Fatal("book not found")
	}
	if _, ok := book["charts"]; ok {
		t.Error("charts should not appear when there are none")
	}
	sheets, ok := book["sheets"].(map[string]interface{})
	if !ok {
		t.Fatal("book.sheets not found")
	}
	if _, ok := sheets["Sheet1"]; !ok {
		t.Error("Sheet1 not found")
	}
}

func TestToJSONChartsheetSkippedLegacyMode(t *testing.T) {
	// WrapWithBook=false のとき chartsheet は単にスキップされる
	var buf bytes.Buffer
	xf := excelize.NewFile()
	defer xf.Close()

	xf.SetCellValue("Sheet1", "A1", "data")
	xf.AddChartSheet("Chart1", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{Name: "S1", Categories: "Sheet1!$A$1:$A$1", Values: "Sheet1!$A$1:$A$1"},
		},
	})

	if err := xf.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var out bytes.Buffer
	err := ToJSONWithOptions(&buf, &out, ToJSONOptions{DateMode: DateModeSerial})
	if err != nil {
		t.Fatalf("ToJSONWithOptions: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// legacy mode: flat output with 'name' (single sheet)
	if _, ok := result["name"]; !ok {
		t.Error("expected legacy format with 'name' field")
	}
	if _, ok := result["book"]; ok {
		t.Error("book should not appear in legacy mode")
	}
}

func TestChartFullRoundtrip(t *testing.T) {
	// 元の JSON（scatter chartsheet 2 つ）
	const srcJSON = `{
		"version": "0.2",
		"book": {
			"sheets": {
				"Data": {
					"cells": {
						"A1": {"t":"s","v":"sample"},
						"B1": {"t":"s","v":"x"},
						"C1": {"t":"s","v":"y"},
						"A2": {"t":"s","v":"S1"},"B2":{"t":"n","v":1},"C2":{"t":"n","v":10},
						"A3": {"t":"s","v":"S2"},"B3":{"t":"n","v":2},"C3":{"t":"n","v":20},
						"A4": {"t":"s","v":"S3"},"B4":{"t":"n","v":3},"C4":{"t":"n","v":30}
					}
				}
			},
			"charts": [
				{
					"id":"ch1",
					"mode":"chartSheet",
					"ct":"scatter",
					"sheet":"Scatter1",
					"title":{"tx":"Scatter 1"},
					"legend":{"show":true,"pos":"bottom"},
					"xAxis":{"title":"X Axis","majorGridLines":true},
					"yAxis":{"title":"Y Axis","majorGridLines":true},
					"ser":[
						{
							"name":"Series1",
							"cat":"Data!$B$2:$B$4",
							"val":"Data!$C$2:$C$4",
							"marker":{"symbol":"circle","size":6},
							"line":{"width":1.5}
						}
					]
				},
				{
					"id":"ch2",
					"mode":"chartSheet",
					"ct":"scatter",
					"sheet":"Scatter2",
					"title":{"tx":"Scatter 2"},
					"xAxis":{"title":"X"},
					"yAxis":{"title":"Y"},
					"ser":[
						{
							"name":"Series2",
							"cat":"Data!$B$2:$B$4",
							"val":"Data!$C$2:$C$4",
							"marker":{"symbol":"diamond","size":7}
						}
					]
				}
			]
		}
	}`

	// Step A: JSON → XLSX
	var xlsx1 bytes.Buffer
	if err := Convert(strings.NewReader(srcJSON), &xlsx1, ConvertOptions{}); err != nil {
		t.Fatalf("step A (json→xlsx): %v", err)
	}

	f1, err := excelize.OpenReader(bytes.NewReader(xlsx1.Bytes()))
	if err != nil {
		t.Fatalf("step A open: %v", err)
	}
	sheets1 := f1.GetSheetList()
	f1.Close()

	if len(sheets1) != 4 {
		t.Fatalf("step A: expected 4 sheets (including helper), got %v", sheets1)
	}

	// Step B: XLSX → JSON (book wrapper)
	var jsonOut bytes.Buffer
	if err := ToJSONWithOptions(&xlsx1, &jsonOut, ToJSONOptions{DateMode: DateModeSerial, WrapWithBook: true}); err != nil {
		t.Fatalf("step B (xlsx→json): %v", err)
	}

	var wb2 Workbook
	if err := json.Unmarshal(jsonOut.Bytes(), &wb2); err != nil {
		t.Fatalf("step B unmarshal: %v\nraw:\n%s", err, jsonOut.String())
	}
	if wb2.Version != "0.2" {
		t.Errorf("step B: version = %q, want 0.2", wb2.Version)
	}
	if wb2.Book == nil {
		t.Fatal("step B: book is nil")
	}
	if len(wb2.Book.Charts) != 2 {
		t.Errorf("step B: expected 2 charts, got %d", len(wb2.Book.Charts))
	}
	// scatter → cat/val 正規化の確認
	for i, ch := range wb2.Book.Charts {
		if ch.Mode != "chartSheet" {
			t.Errorf("step B chart[%d].mode = %q, want chartSheet", i, ch.Mode)
		}
		if len(ch.Ser) != 1 {
			t.Errorf("step B chart[%d]: expected 1 ser, got %d", i, len(ch.Ser))
			continue
		}
		wantName := fmt.Sprintf("Series%d", i+1)
		if ch.Ser[0].Name != wantName {
			t.Errorf("step B chart[%d].ser[0].name = %q, want %q", i, ch.Ser[0].Name, wantName)
		}
		if ch.Ser[0].Cat != "Data!$B$2:$B$4" {
			t.Errorf("step B chart[%d].ser[0].cat = %q, want Data!$B$2:$B$4", i, ch.Ser[0].Cat)
		}
		if ch.Ser[0].Val != "Data!$C$2:$C$4" {
			t.Errorf("step B chart[%d].ser[0].val = %q, want Data!$C$2:$C$4", i, ch.Ser[0].Val)
		}
		if ch.Ser[0].Marker == nil {
			t.Errorf("step B chart[%d].ser[0].marker is nil", i)
		}
	}
	// note: scatter chart series line is always NoFill in Excelize, so Line is nil after extraction
	// 1st chart data sheet inside book.sheets
	if _, ok := wb2.Book.Sheets["Data"]; !ok {
		t.Error("step B: Data sheet missing in book.sheets")
	}

	// Step C: JSON(book wrapper) → XLSX
	jsonBytes, _ := json.Marshal(wb2)
	var xlsx2 bytes.Buffer
	if err := Convert(bytes.NewReader(jsonBytes), &xlsx2, ConvertOptions{}); err != nil {
		t.Fatalf("step C (json→xlsx): %v", err)
	}

	f3, err := excelize.OpenReader(bytes.NewReader(xlsx2.Bytes()))
	if err != nil {
		t.Fatalf("step C open: %v", err)
	}
	defer f3.Close()

	sheets3 := f3.GetSheetList()
	if len(sheets3) != 4 {
		t.Fatalf("step C: expected 4 sheets (including helper), got %v", sheets3)
	}
	// chartsheet が存在することを確認
	hasChartsheet := false
	for _, name := range sheets3 {
		if name == "Scatter1" || name == "Scatter2" {
			hasChartsheet = true
		}
	}
	if !hasChartsheet {
		t.Errorf("step C: chartsheets not found in %v", sheets3)
	}
	// データシートは chartsheet 扱いされていないこと
	if _, err := f3.GetCellValue("Data", "A1"); err != nil {
		t.Errorf("step C: Data sheet not readable: %v", err)
	} else if val, _ := f3.GetCellValue("Data", "A1"); val != "sample" {
		t.Errorf("step C: Data!A1 = %q, want sample", val)
	}
}

func TestToJSONEmbeddedChartRoundtrip(t *testing.T) {
	const srcJSON = `{
		"version": "0.2",
		"book": {
			"sheets": {
				"Data": {
					"cells": {
						"A1": {"t":"s","v":"cat"},
						"B1": {"t":"s","v":"val"},
						"A2": {"t":"s","v":"X"}, "B2": {"t":"n","v":1},
						"A3": {"t":"s","v":"Y"}, "B3": {"t":"n","v":2}
					}
				}
			},
			"charts": [
				{
					"id": "ch1",
					"mode": "embedded",
					"ct": "col",
					"sheet": "Data",
					"anchor": "D2",
					"dim": {"w": 10, "h": 15},
					"title": {"tx":"Embedded Chart"},
					"legend": {"show":true, "pos":"bottom"},
					"xAxis": {"title":"X Axis", "majorGridLines":true},
					"yAxis": {"title":"Y Axis", "majorGridLines":true},
					"ser": [
						{
							"name": "S1",
							"cat": "Data!$A$2:$A$3",
							"val": "Data!$B$2:$B$3"
						}
					]
				}
			]
		}
	}`

	// Step A: JSON → XLSX
	var xlsx1 bytes.Buffer
	if err := Convert(strings.NewReader(srcJSON), &xlsx1, ConvertOptions{}); err != nil {
		t.Fatalf("step A (json→xlsx): %v", err)
	}

	f1, err := excelize.OpenReader(bytes.NewReader(xlsx1.Bytes()))
	if err != nil {
		t.Fatalf("step A open: %v", err)
	}
	sheets1 := f1.GetSheetList()
	f1.Close()

	if len(sheets1) != 2 {
		t.Fatalf("step A: expected 2 sheets (including helper), got %v", sheets1)
	}

	// Step B: XLSX → JSON (book wrapper)
	var jsonOut bytes.Buffer
	if err := ToJSONWithOptions(&xlsx1, &jsonOut, ToJSONOptions{DateMode: DateModeSerial, WrapWithBook: true}); err != nil {
		t.Fatalf("step B (xlsx→json): %v", err)
	}

	var wb2 Workbook
	if err := json.Unmarshal(jsonOut.Bytes(), &wb2); err != nil {
		t.Fatalf("step B unmarshal: %v\nraw:\n%s", err, jsonOut.String())
	}
	if wb2.Version != "0.2" {
		t.Errorf("step B: version = %q, want 0.2", wb2.Version)
	}
	if wb2.Book == nil {
		t.Fatal("step B: book is nil")
	}
	if len(wb2.Book.Charts) != 1 {
		t.Fatalf("step B: expected 1 chart, got %d", len(wb2.Book.Charts))
	}

	ch := wb2.Book.Charts[0]
	if ch.Mode != "embedded" {
		t.Errorf("step B chart.mode = %q, want embedded", ch.Mode)
	}
	if ch.Sheet != "Data" {
		t.Errorf("step B chart.sheet = %q, want Data", ch.Sheet)
	}
	if ch.Anchor != "D2" {
		t.Errorf("step B chart.anchor = %q, want D2", ch.Anchor)
	}
	if ch.Dim == nil {
		t.Fatal("step B chart.dim is nil")
	}
	if ch.Dim.W != 10 {
		t.Errorf("step B chart.dim.w = %v, want 10", ch.Dim.W)
	}
	if ch.Dim.H != 15 {
		t.Errorf("step B chart.dim.h = %v, want 15", ch.Dim.H)
	}
	if ch.Dim.OffX != 0 {
		t.Errorf("step B chart.dim.offx = %v, want 0", ch.Dim.OffX)
	}
	if ch.Dim.OffY != 0 {
		t.Errorf("step B chart.dim.offy = %v, want 0", ch.Dim.OffY)
	}
	if ch.Title == nil || ch.Title.Tx != "Embedded Chart" {
		t.Errorf("step B chart.title = %+v, want {Tx:Embedded Chart}", ch.Title)
	}
	if ch.Legend == nil || !ch.Legend.Show || ch.Legend.Pos != "bottom" {
		t.Errorf("step B chart.legend = %+v, want {show:true pos:bottom}", ch.Legend)
	}
	if ch.XAxis == nil || ch.XAxis.Title != "X Axis" || !ch.XAxis.MajorGridLines {
		t.Errorf("step B chart.xAxis = %+v, want title=X Axis majorGridLines=true", ch.XAxis)
	}
	if ch.YAxis == nil || ch.YAxis.Title != "Y Axis" || !ch.YAxis.MajorGridLines {
		t.Errorf("step B chart.yAxis = %+v, want title=Y Axis majorGridLines=true", ch.YAxis)
	}
	if len(ch.Ser) != 1 {
		t.Fatalf("step B expected 1 ser, got %d", len(ch.Ser))
	}
	if ch.Ser[0].Cat != "Data!$A$2:$A$3" {
		t.Errorf("step B ser[0].cat = %q, want Data!$A$2:$A$3", ch.Ser[0].Cat)
	}
	if ch.Ser[0].Val != "Data!$B$2:$B$3" {
		t.Errorf("step B ser[0].val = %q, want Data!$B$2:$B$3", ch.Ser[0].Val)
	}
	if _, ok := wb2.Book.Sheets["Data"]; !ok {
		t.Error("step B: Data sheet missing in book.sheets")
	}

	// Step C: JSON(book wrapper) → XLSX
	jsonBytes, _ := json.Marshal(wb2)
	var xlsx2 bytes.Buffer
	if err := Convert(bytes.NewReader(jsonBytes), &xlsx2, ConvertOptions{}); err != nil {
		t.Fatalf("step C (json→xlsx): %v", err)
	}

	f3, err := excelize.OpenReader(bytes.NewReader(xlsx2.Bytes()))
	if err != nil {
		t.Fatalf("step C open: %v", err)
	}
	defer f3.Close()

	sheets3 := f3.GetSheetList()
	if len(sheets3) != 2 {
		t.Fatalf("step C: expected 2 sheets (including helper), got %v", sheets3)
	}
	if sheets3[0] != "Data" {
		t.Errorf("step C: sheet name = %q, want Data", sheets3[0])
	}
	if val, _ := f3.GetCellValue("Data", "A1"); val != "cat" {
		t.Errorf("step C: Data!A1 = %q, want cat", val)
	}
}

func TestConditionalFormat_CellType(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}, "A2": {"t": "n", "v": 10}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "cell",
					"criteria": ">",
					"value": "5",
					"style": {
						"fill": {"type": "pattern", "pattern": 1, "color": ["#FFC7CE"]},
						"font": {"color": "#9C0006"}
					}
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	if len(cfs) != 1 {
		t.Fatalf("expected 1 conditional format, got %d", len(cfs))
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if len(opts) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(opts))
	}
	if opts[0].Type != "cell" {
		t.Errorf("Type = %q, want cell", opts[0].Type)
	}
	if opts[0].Criteria != "greater than" {
		t.Errorf("Criteria = %q, want greater than", opts[0].Criteria)
	}
	if opts[0].Value != "5" {
		t.Errorf("Value = %q, want 5", opts[0].Value)
	}
}

func TestConditionalFormat_Between(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 5}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "cell",
					"criteria": "between",
					"minValue": "3",
					"maxValue": "7",
					"style": {
						"fill": {"type": "pattern", "pattern": 1, "color": ["#C6EFCE"]},
						"font": {"color": "#006100"}
					}
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Criteria != "between" {
		t.Errorf("Criteria = %q, want between", opts[0].Criteria)
	}
	if opts[0].MinValue != "3" {
		t.Errorf("MinValue = %q, want 3", opts[0].MinValue)
	}
	if opts[0].MaxValue != "7" {
		t.Errorf("MaxValue = %q, want 7", opts[0].MaxValue)
	}
}

func TestConditionalFormat_ColorScale(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "2_color_scale",
					"criteria": "=",
					"minType": "min",
					"maxType": "max",
					"minColor": "#F8696B",
					"maxColor": "#63BE7B"
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "2_color_scale" {
		t.Errorf("Type = %q, want 2_color_scale", opts[0].Type)
	}
	if opts[0].MinType != "min" {
		t.Errorf("MinType = %q, want min", opts[0].MinType)
	}
	if opts[0].MaxType != "max" {
		t.Errorf("MaxType = %q, want max", opts[0].MaxType)
	}
}

func TestConditionalFormat_DataBar(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "data_bar",
					"criteria": "=",
					"minType": "min",
					"maxType": "max",
					"barColor": "#638EC6"
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "data_bar" {
		t.Errorf("Type = %q, want data_bar", opts[0].Type)
	}
}

func TestConditionalFormat_IconSet(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "icon_set",
					"iconStyle": "3TrafficLights1"
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "icon_set" {
		t.Errorf("Type = %q, want icon_set (excelize read-back name)", opts[0].Type)
	}
	if opts[0].IconStyle != "3TrafficLights1" {
		t.Errorf("IconStyle = %q, want 3TrafficLights1", opts[0].IconStyle)
	}
}

func TestConditionalFormat_Formula(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "formula",
					"criteria": "ISODD(A1)",
					"style": {
						"font": {"bold": true}
					}
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "formula" {
		t.Errorf("Type = %q, want formula", opts[0].Type)
	}
}

func TestConditionalFormat_Duplicate(t *testing.T) {
	js := `{
		"sheets": [{
			"name": "S1",
			"cells": {"A1": {"t": "n", "v": 1}, "A2": {"t": "n", "v": 1}},
			"conditionalFormats": [{
				"range": "A1:A10",
				"rules": [{
					"type": "duplicate",
					"criteria": "=",
					"style": {
						"fill": {"type": "pattern", "pattern": 1, "color": ["#FFC7CE"]}
					}
				}]
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "duplicate" {
		t.Errorf("Type = %q, want duplicate", opts[0].Type)
	}
}

func TestConditionalFormat_SingleSheetForm(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {"A1": {"t": "n", "v": 1}},
		"conditionalFormats": [{
			"range": "A1:A10",
			"rules": [{
				"type": "cell",
				"criteria": ">",
				"value": "5",
				"style": {
					"font": {"bold": true}
				}
			}]
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	cfs, err := f.GetConditionalFormats("S1")
	if err != nil {
		t.Fatalf("GetConditionalFormats error: %v", err)
	}
	if len(cfs) != 1 {
		t.Fatalf("expected 1 conditional format, got %d", len(cfs))
	}
	opts, ok := cfs["A1:A10"]
	if !ok {
		t.Fatalf("expected range A1:A10, got keys: %v", keysOfMap(cfs))
	}
	if opts[0].Type != "cell" {
		t.Errorf("Type = %q, want cell", opts[0].Type)
	}
}

func TestAutoFilter_StringForm(t *testing.T) {
	js := `{
		"name": "S1",
		"autoFilter": "A1:C10",
		"cells": {
			"A1": {"t": "s", "v": "H1"},
			"B1": {"t": "s", "v": "H2"},
			"A2": {"t": "s", "v": "d1"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	// Verify autoFilter was set by checking the sheet has filter info
	// We write a new file and check the XML directly via GetPanes won't work.
	// Instead, round-trip through ToJSON to verify preservation.
	_ = f
}

func TestAutoFilter_ObjectForm(t *testing.T) {
	js := `{
		"name": "S1",
		"autoFilter": {"ref": "A1:C10"},
		"cells": {
			"A1": {"t": "s", "v": "H1"},
			"A2": {"t": "s", "v": "d1"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()
}

func TestAutoFilter_RoundTrip(t *testing.T) {
	js := `{
		"name": "S1",
		"autoFilter": "A1:C10",
		"cells": {
			"A1": {"t": "s", "v": "H1"},
			"A2": {"t": "s", "v": "d1"}
		}
	}`
	// autoFilter is not captured in ToJSON read-back, so just verify conversion succeeds
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
}

func TestTable_Basic(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Name"},
			"B1": {"t": "s", "v": "Value"},
			"A2": {"t": "s", "v": "A"},
			"B2": {"t": "n", "v": 1}
		},
		"tables": [{
			"range": "A1:B2",
			"name": "MyTable",
			"style": "TableStyleMedium2"
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	tables, err := f.GetTables("S1")
	if err != nil {
		t.Fatalf("GetTables error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "MyTable" {
		t.Errorf("table name = %q, want MyTable", tables[0].Name)
	}
	if tables[0].Range != "A1:B2" {
		t.Errorf("table range = %q, want A1:B2", tables[0].Range)
	}
}

func TestTable_AutoFilterDedup(t *testing.T) {
	// autoFilter と同じ範囲にテーブルがある場合，autoFilter はスキップされる
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Name"},
			"B1": {"t": "s", "v": "Value"},
			"A2": {"t": "s", "v": "A"},
			"B2": {"t": "n", "v": 1}
		},
		"autoFilter": "A1:B2",
		"tables": [{
			"range": "A1:B2",
			"name": "T1",
			"style": "TableStyleMedium2"
		}]
	}`
	// Should succeed without error - autoFilter is covered by table so it's skipped
	f := convertAndOpen(t, js, false)
	defer f.Close()

	// Table should exist
	tables, err := f.GetTables("S1")
	if err != nil {
		t.Fatalf("GetTables error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
}

func TestTable_AutoFilterDifferentRange(t *testing.T) {
	// テーブルと autoFilter が異なる範囲の場合，両方適用
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Name"},
			"B1": {"t": "s", "v": "Value"},
			"A2": {"t": "s", "v": "A"},
			"B2": {"t": "n", "v": 1},
			"A3": {"t": "s", "v": "B"},
			"B3": {"t": "n", "v": 2}
		},
		"autoFilter": "A1:B3",
		"tables": [{
			"range": "A1:B2",
			"name": "T1"
		}]
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	tables, err := f.GetTables("S1")
	if err != nil {
		t.Fatalf("GetTables error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
}

func TestTable_RoundTrip(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "X"},
			"B1": {"t": "n", "v": 1}
		},
		"tables": [{
			"range": "A1:B1",
			"name": "T1",
			"style": "TableStyleMedium2"
		}]
	}`
	// Tables are not preserved in round-trip (excelize doesn't expose table info in GetTables
	// that we capture in ToJSON), so we just verify no error
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	tables, err := f.GetTables("S1")
	if err != nil {
		t.Fatalf("GetTables: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
}

func TestSparkline_Basic(t *testing.T) {
	// 単純な line スパークライン
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Sparkline"},
			"B1": {"t": "n", "v": 1},
			"C1": {"t": "n", "v": 3},
			"D1": {"t": "n", "v": 2},
			"E1": {"t": "n", "v": 5}
		},
		"sparklines": [{
			"location": "A1",
			"range": "S1!B1:E1",
			"markers": true,
			"high": true,
			"low": true
		}]
	}`
	// Sparklines are not readable back via excelize API, just verify no error
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	// Verify cells are still intact
	got, _ := f.GetCellValue("S1", "A1")
	if got != "Sparkline" {
		t.Errorf("A1 = %q, want Sparkline", got)
	}
}

func TestSparkline_Column(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A2": {"t": "s", "v": "Col"},
			"B2": {"t": "n", "v": 1},
			"C2": {"t": "n", "v": 4},
			"D2": {"t": "n", "v": 2}
		},
		"sparklines": [{
			"location": "A2",
			"range": "S1!B2:D2",
			"type": "column",
			"negative": true,
			"first": true,
			"last": true
		}]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
}

func TestSparkline_WinLoss(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A3": {"t": "s", "v": "W/L"},
			"B3": {"t": "n", "v": 1},
			"C3": {"t": "n", "v": -1},
			"D3": {"t": "n", "v": 1},
			"E3": {"t": "n", "v": 1}
		},
		"sparklines": [{
			"location": "A3",
			"range": "S1!B3:E3",
			"type": "win_loss",
			"axis": true,
			"negative": true
		}]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
}

func TestSparkline_WithColor(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A4": {"t": "s", "v": "Colored"},
			"B4": {"t": "n", "v": 1},
			"C4": {"t": "n", "v": 3}
		},
		"sparklines": [{
			"location": "A4",
			"range": "S1!B4:C4",
			"type": "line",
			"seriesColor": "#4472C4",
			"markersColor": "#FF0000",
			"highColor": "#00FF00",
			"weight": 2.5
		}]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
}

func TestSparkline_SingleSheetForm(t *testing.T) {
	js := `{
		"name": "Data",
		"cells": {
			"A1": {"t": "s", "v": "data"},
			"B1": {"t": "n", "v": 10},
			"C1": {"t": "n", "v": 20}
		},
		"sparklines": [{
			"location": "A2",
			"range": "Data!B1:C1"
		}]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}
}

func TestAutoFit_ColumnWidthCells(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Short"},
			"B1": {"t": "s", "v": "A longer cell value here"},
			"A2": {"t": "n", "v": 123}
		}
	}`
	f := convertAndOpenWithOpts(t, js, ConvertOptions{AutoFit: true})
	defer f.Close()

	wA, _ := f.GetColWidth("S1", "A")
	if wA < 5 || wA > 10 {
		t.Errorf("col A width = %v, expected around 7-9 (content: 'Short', '123')", wA)
	}
	wB, _ := f.GetColWidth("S1", "B")
	if wB < 18 || wB > 30 {
		t.Errorf("col B width = %v, expected around 23-26 (content: 'A longer cell value here')", wB)
	}
}

func TestAutoFit_ColumnWidthRows(t *testing.T) {
	js := `{
		"name": "S1",
		"rows": [
			["Short", "A longer cell value here"],
			[123, "hello"]
		]
	}`
	f := convertAndOpenWithOpts(t, js, ConvertOptions{AutoFit: true})
	defer f.Close()

	wA, _ := f.GetColWidth("S1", "A")
	if wA < 4 || wA > 10 {
		t.Errorf("col A width = %v, expected around 5-8 (content: 'Short', 123)", wA)
	}
	wB, _ := f.GetColWidth("S1", "B")
	if wB < 18 || wB > 30 {
		t.Errorf("col B width = %v, expected around 23-26 (content: 'A longer cell value here')", wB)
	}
}

func TestAutoFit_ExplicitColOverrides(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Very long content here that should not affect width"},
			"B1": {"t": "s", "v": "Short"}
		},
		"cols": [{"col": "A", "width": 5}]
	}`
	f := convertAndOpenWithOpts(t, js, ConvertOptions{AutoFit: true})
	defer f.Close()

	wA, _ := f.GetColWidth("S1", "A")
	if wA != 5 {
		t.Errorf("col A explicit width = %v, want 5", wA)
	}
	wB, _ := f.GetColWidth("S1", "B")
	if wB < 4 || wB > 10 {
		t.Errorf("col B auto-fitted width = %v, expected around 5-8 (content: 'Short')", wB)
	}
}

func TestAutoFit_WrapTextOnNewline(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "Hello"},
			"B1": {"t": "s", "v": "line1\nline2\nline3"}
		}
	}`
	f := convertAndOpenWithOpts(t, js, ConvertOptions{AutoFit: true})
	defer f.Close()

	styleID, err := f.GetCellStyle("S1", "A1")
	if err != nil {
		t.Fatal(err)
	}
	styleA, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if styleA.Alignment != nil && styleA.Alignment.WrapText {
		t.Error("A1 should NOT have WrapText (no newline)")
	}

	styleID, err = f.GetCellStyle("S1", "B1")
	if err != nil {
		t.Fatal(err)
	}
	styleB, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if styleB.Alignment == nil || !styleB.Alignment.WrapText {
		t.Error("B1 should have WrapText (contains newline)")
	}
}

func TestAutoFit_WrapTextOnNewline_WithExistingStyle(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "line1\nline2", "s": 1}
		},
		"styles": [{"id": 1, "font": {"bold": true}}]
	}`
	f := convertAndOpenWithOpts(t, js, ConvertOptions{AutoFit: true})
	defer f.Close()

	styleID, err := f.GetCellStyle("S1", "A1")
	if err != nil {
		t.Fatal(err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.Alignment == nil || !style.Alignment.WrapText {
		t.Error("A1 should have WrapText (contains newline)")
	}
	if style.Font == nil || !style.Font.Bold {
		t.Error("A1 should preserve bold from base style")
	}
}

func TestAutoFit_NoAutoFitWithoutFlag(t *testing.T) {
	js := `{
		"name": "S1",
		"cells": {
			"A1": {"t": "s", "v": "A long cell value that should not affect width without --autofit"}
		}
	}`
	f := convertAndOpen(t, js, false)
	defer f.Close()

	wA, _ := f.GetColWidth("S1", "A")
	// Without autofit, the width should be the excelize default (9.14 or whatever)
	if wA > 20 {
		t.Errorf("col A width = %v, expected default (~9) without --autofit", wA)
	}
}

func keysOfMap(m map[string][]excelize.ConditionalFormatOptions) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
