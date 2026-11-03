package json2xlsx

import (
	"fmt"
	"strings"
	"testing"
)

func crossSheetWorkbook() *Workbook {
	return &Workbook{Sheets: []Sheet{
		{
			Name: "Sales",
			Cells: map[string]Cell{
				"A2": {T: "s", V: "A"},
				"B2": {T: "n", V: float64(10)},
				"C2": {T: "n", V: float64(100)},
				"D2": {T: "f", F: "B2*C2"},
				"A3": {T: "s", V: "B"},
				"B3": {T: "n", V: float64(5)},
				"C3": {T: "n", V: float64(200)},
				"D3": {T: "f", F: "B3*C3"},
			},
		},
		{
			Name: "Summary",
			Cells: map[string]Cell{
				"B1": {T: "f", F: "SUM(Sales!D2:D3)"},
				"B2": {T: "f", F: "Sales!D2*2"},
				"B3": {T: "f", F: "sales!D3"},
			},
		},
	}}
}

func findSheet(t *testing.T, wb *Workbook, name string) *Sheet {
	t.Helper()
	for i := range wb.Sheets {
		if wb.Sheets[i].Name == name {
			return &wb.Sheets[i]
		}
	}
	t.Fatalf("sheet %q not found", name)
	return nil
}

func requireCellValue(t *testing.T, wb *Workbook, sheet, axis string, want any) {
	t.Helper()
	sh := findSheet(t, wb, sheet)
	cell, ok := sh.Cells[axis]
	if !ok {
		t.Fatalf("%s!%s not found", sheet, axis)
	}
	if fmt.Sprint(cell.V) != fmt.Sprint(want) {
		t.Fatalf("%s!%s = %v, want %v", sheet, axis, cell.V, want)
	}
}

func TestEvalWorkbookFormulasCrossSheet(t *testing.T) {
	wb := crossSheetWorkbook()
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	requireCellValue(t, wb, "Summary", "B1", float64(2000))
	requireCellValue(t, wb, "Summary", "B2", float64(2000))
	requireCellValue(t, wb, "Summary", "B3", float64(1000))
}

func TestEvalWorkbookFormulasQuotedSheetAndLookups(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{
			Name: "Sales 2026",
			Cells: map[string]Cell{
				"A2": {T: "s", V: "A"},
				"B2": {T: "n", V: float64(10)},
				"C2": {T: "n", V: float64(100)},
				"D2": {T: "f", F: "B2*C2"},
				"A3": {T: "s", V: "B"},
				"B3": {T: "n", V: float64(5)},
				"C3": {T: "n", V: float64(200)},
				"D3": {T: "f", F: "B3*C3"},
			},
		},
		{
			Name: "Summary",
			Cells: map[string]Cell{
				"B1":  {T: "f", F: `SUM('Sales 2026'!D2:D3)`},
				"B2":  {T: "f", F: `COUNTIF('Sales 2026'!A2:A3,"A")`},
				"B3":  {T: "f", F: `VLOOKUP("A",'Sales 2026'!A2:D3,4,0)`},
				"B4":  {T: "f", F: `INDEX('Sales 2026'!D2:D3,1)`},
				"B5":  {T: "f", F: `MATCH("B",'Sales 2026'!A2:A3,0)`},
				"B6":  {T: "f", F: `XLOOKUP("A",'Sales 2026'!A2:A3,'Sales 2026'!D2:D3)`},
				"B7":  {T: "f", F: `RANK('Sales 2026'!D2,'Sales 2026'!D2:D3)`},
				"B8":  {T: "f", F: `SUBTOTAL(9,'Sales 2026'!D2:D3)`},
				"B9":  {T: "f", F: `COUNTA('Sales 2026'!A2:A3)`},
				"B10": {T: "f", F: `ISNUMBER('Sales 2026'!B2)`},
				"B11": {T: "f", F: `SUMIF('Sales 2026'!A2:A3,"A",'Sales 2026'!D2:D3)`},
				"B12": {T: "f", F: `ROW('Sales 2026'!D3)`},
				"B13": {T: "f", F: `COLUMN('Sales 2026'!D3)`},
				"B14": {T: "f", F: `HLOOKUP("A",'Sales 2026'!A2:D3,1,0)`},
				"B15": {T: "f", F: `AVERAGEIF('Sales 2026'!A2:A3,"A",'Sales 2026'!D2:D3)`},
				"B16": {T: "f", F: `SUMIFS('Sales 2026'!D2:D3,'Sales 2026'!A2:A3,"A")`},
				"B17": {T: "f", F: `COUNTIFS('Sales 2026'!A2:A3,"A")`},
				"B18": {T: "f", F: `MINIFS('Sales 2026'!D2:D3,'Sales 2026'!A2:A3,"A")`},
				"B19": {T: "f", F: `MAXIFS('Sales 2026'!D2:D3,'Sales 2026'!A2:A3,"A")`},
				"B20": {T: "f", F: `AVERAGEIFS('Sales 2026'!D2:D3,'Sales 2026'!A2:A3,"A")`},
				"B21": {T: "f", F: `COUNT('Sales 2026'!B2:B3)`},
			},
		},
	}}
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	checks := map[string]float64{
		"B1": 2000, "B2": 1, "B3": 1000, "B4": 1000, "B5": 2, "B6": 1000,
		"B7": 1, "B8": 2000, "B9": 2, "B10": 1, "B11": 1000, "B12": 3, "B13": 4,
		"B15": 1000, "B16": 1000, "B17": 1, "B18": 1000, "B19": 1000, "B20": 1000,
		"B21": 2,
	}
	for axis, want := range checks {
		requireCellValue(t, wb, "Summary", axis, want)
	}
}

func TestEvalWorkbookFormulasJapaneseSheetQuoted(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "売上", Cells: map[string]Cell{"A1": {T: "n", V: float64(42)}}},
		{Name: "集計", Cells: map[string]Cell{"A1": {T: "f", F: `SUM('売上'!A1:A1)`}}},
	}}
	if warnings := EvalWorkbookFormulas(wb); len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	requireCellValue(t, wb, "集計", "A1", float64(42))
}

func TestEvalWorkbookFormulasUnquotedJapaneseSheetFails(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "売上", Cells: map[string]Cell{"A1": {T: "n", V: float64(42)}}},
		{Name: "集計", Cells: map[string]Cell{"A1": {T: "f", F: `売上!A1`}}},
	}}
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) == 0 || !strings.Contains(warnings[0], "parse error") {
		t.Fatalf("warnings = %v, want a parse error", warnings)
	}
}

func TestEvalWorkbookFormulasQuotedApostrophe(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "Bob's", Cells: map[string]Cell{"A1": {T: "n", V: float64(7)}}},
		{Name: "Summary", Cells: map[string]Cell{"A1": {T: "f", F: `'Bob''s'!A1`}}},
	}}
	if warnings := EvalWorkbookFormulas(wb); len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	requireCellValue(t, wb, "Summary", "A1", float64(7))
}

func TestEvalWorkbookFormulasUnknownSheet(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "Summary", Cells: map[string]Cell{"A1": {T: "f", F: `Missing!A1`}}},
	}}
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) == 0 || !strings.Contains(warnings[0], "unknown sheet") {
		t.Fatalf("warnings = %v, want unknown sheet", warnings)
	}
}

func TestEvalWorkbookFormulasCrossSheetCircular(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "One", Cells: map[string]Cell{"A1": {T: "f", F: `Two!A1`}}},
		{Name: "Two", Cells: map[string]Cell{"A1": {T: "f", F: `One!A1`}}},
	}}
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) == 0 || !strings.Contains(warnings[0], "circular reference") {
		t.Fatalf("warnings = %v, want circular reference", warnings)
	}
}

func TestEvalWorkbookFormulasRangeCannotSpanSheets(t *testing.T) {
	wb := &Workbook{Sheets: []Sheet{
		{Name: "One", Cells: map[string]Cell{"A1": {T: "n", V: float64(1)}}},
		{Name: "Two", Cells: map[string]Cell{"A1": {T: "n", V: float64(2)}}},
		{Name: "Summary", Cells: map[string]Cell{"A1": {T: "f", F: `SUM(One!A1:Two!A1)`}}},
	}}
	warnings := EvalWorkbookFormulas(wb)
	if len(warnings) == 0 || !strings.Contains(warnings[0], "range cannot span sheets") {
		t.Fatalf("warnings = %v, want range cannot span sheets", warnings)
	}
}

func TestEvalFormulaCrossSheet(t *testing.T) {
	wb := crossSheetWorkbook()
	res, err := EvalFormula("=SUM(Sales!D2:D3)", wb, EvalOptions{Sheet: "Summary"})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(2000) {
		t.Fatalf("value = %v, want 2000", res.Value)
	}

	res, err = EvalFormula("=Sales!D2*2", wb, EvalOptions{Sheet: "Summary"})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(2000) {
		t.Fatalf("value = %v, want 2000", res.Value)
	}
}
