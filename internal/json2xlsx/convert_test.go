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

func convertAndOpen(t *testing.T, jsonStr string) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(jsonStr), &buf); err != nil {
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
	if err := Convert(strings.NewReader(js), &buf); err != nil {
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
func convertWithStderr(t *testing.T, jsonStr string) (xlsxData []byte, stderrOutput string, convertErr error) {
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

	convertErr = Convert(r, &buf)

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
	xlsxData, stderrOut, err := convertWithStderr(t, js)
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
	xlsxData, stderrOut, err := convertWithStderr(t, js)
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
	f := convertAndOpen(t, js)
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
	f := convertAndOpen(t, js)
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
	err := Convert(strings.NewReader(js), &buf)
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
	err := Convert(strings.NewReader(js), &buf)
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
	_, _, err := convertWithStderr(t, js)
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
	if err := Convert(strings.NewReader(srcJSON), &xlsx1); err != nil {
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
	if err := Convert(bytes.NewReader(jsonBytes), &xlsx2); err != nil {
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
	if err := Convert(strings.NewReader(srcJSON), &xlsx1); err != nil {
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
	if err := Convert(bytes.NewReader(jsonBytes), &xlsx2); err != nil {
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
