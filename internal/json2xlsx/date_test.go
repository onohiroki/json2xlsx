package json2xlsx

import (
	"encoding/json"
	"testing"
)

func TestDateCellToString_Nil(t *testing.T) {
	if got := dateCellToString(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestDateCellToString_StringInput(t *testing.T) {
	if got := dateCellToString("2025-01-21"); got != "2025-01-21" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestDateCellToString_Serial(t *testing.T) {
	got := dateCellToString(float64(45678))
	if got != "2025-01-21T00:00:00" {
		t.Fatalf("expected RFC3339, got %q", got)
	}
}

func TestDateCellToString_JSONNumberSerial(t *testing.T) {
	jn := json.Number("45678")
	got := dateCellToString(jn)
	if got != "2025-01-21T00:00:00" {
		t.Fatalf("expected RFC3339, got %q", got)
	}
}

func TestDateCellToString_InvalidJSONNumber(t *testing.T) {
	jn := json.Number("not-a-number")
	got := dateCellToString(jn)
	if got != "not-a-number" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestDateCellToString_UnknownType(t *testing.T) {
	got := dateCellToString(struct{ X int }{X: 5})
	if got != "{5}" {
		t.Fatalf("expected {5}, got %q", got)
	}
}

func TestIsTimeOnlyFormat_Empty(t *testing.T) {
	if isTimeOnlyFormat("") {
		t.Fatal("expected false for empty string")
	}
}

func TestIsTimeOnlyFormat_TimeOnly(t *testing.T) {
	cases := []string{"h:mm", "h:mm:ss", "hh:mm", "mm:ss", "[h]:mm:ss"}
	for _, c := range cases {
		if !isTimeOnlyFormat(c) {
			t.Errorf("expected true for %q", c)
		}
	}
}

func TestIsTimeOnlyFormat_DateTime(t *testing.T) {
	if isTimeOnlyFormat("yyyy-mm-dd h:mm") {
		t.Fatal("expected false for datetime format")
	}
}

func TestIsTimeOnlyFormat_DateOnly(t *testing.T) {
	if isTimeOnlyFormat("yyyy-mm-dd") {
		t.Fatal("expected false for date-only format")
	}
}

func TestIsTimeOnlyFormat_NumberFormat(t *testing.T) {
	if isTimeOnlyFormat("#,##0") {
		t.Fatal("expected false for number format")
	}
}

func TestFormatTimeOnly_Positive(t *testing.T) {
	got := formatTimeOnly(0.04623843, "h:mm:ss")
	if got != "1:06:35" {
		t.Fatalf("expected 1:06:35, got %q", got)
	}
}

func TestFormatTimeOnly_Negative(t *testing.T) {
	got := formatTimeOnly(-0.04623843, "h:mm:ss")
	if got != "-1:06:35" {
		t.Fatalf("expected -1:06:35, got %q", got)
	}
}

func TestFormatTimeOnly_HHLeadZero(t *testing.T) {
	got := formatTimeOnly(0.04623843, "hh:mm:ss")
	if got != "01:06:35" {
		t.Fatalf("expected 01:06:35, got %q", got)
	}
}

func TestFormatTimeOnly_HoursMinutes(t *testing.T) {
	got := formatTimeOnly(0.5, "h:mm")
	if got != "12:00" {
		t.Fatalf("expected 12:00, got %q", got)
	}
}

func TestFormatTimeOnly_HoursMinutesLeadZero(t *testing.T) {
	got := formatTimeOnly(0.5, "hh:mm")
	if got != "12:00" {
		t.Fatalf("expected 12:00, got %q", got)
	}
}

func TestFormatTimeOnly_MinutesSeconds(t *testing.T) {
	got := formatTimeOnly(3661.0/86400, "mm:ss")
	if got != "61:01" {
		t.Fatalf("expected 61:01, got %q", got)
	}
}

func TestFormatTimeOnly_MinutesSecondsNoHours(t *testing.T) {
	got := formatTimeOnly(150.0/86400, "mm:ss")
	if got != "02:30" {
		t.Fatalf("expected 02:30, got %q", got)
	}
}

func TestFormatTimeOnly_HourBracket(t *testing.T) {
	got := formatTimeOnly(90000.0/86400, "[h]:mm:ss")
	if got != "25:00:00" {
		t.Fatalf("expected 25:00:00, got %q", got)
	}
}

func TestNormalizeDateCells_SingleSheet(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "n", V: float64(45000), Z: "yyyy-mm-dd"},
				"B1": {T: "s", V: "hello"},
			},
		}},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d", wb.Sheets[0].Cells["A1"].T)
	}
	if wb.Sheets[0].Cells["B1"].T != "s" {
		t.Errorf("B1.T = %q, want s (unchanged)", wb.Sheets[0].Cells["B1"].T)
	}
}

func TestNormalizeDateCells_MultiSheet(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Cells: map[string]Cell{"A1": {T: "n", V: float64(45000), Z: "yyyy-mm-dd"}}},
			{Cells: map[string]Cell{"B1": {T: "n", V: float64(100), Z: "m/d/yy"}}},
		},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "d" {
		t.Errorf("Sheet0 A1.T = %q, want d", wb.Sheets[0].Cells["A1"].T)
	}
	if wb.Sheets[1].Cells["B1"].T != "d" {
		t.Errorf("Sheet1 B1.T = %q, want d", wb.Sheets[1].Cells["B1"].T)
	}
}

func TestNormalizeDateCells_BookWrapper(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "S1",
			Cells: map[string]Cell{"A1": {T: "n", V: float64(45000), Z: "yyyy-mm-dd"}},
		}},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d", wb.Sheets[0].Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenAlreadyDate(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "d", V: float64(45000), Z: "yyyy-mm-dd"},
			},
		}},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d (should remain d)", wb.Sheets[0].Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenFormula(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "f", F: "TODAY()", Z: "yyyy-mm-dd"},
			},
		}},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "f" {
		t.Errorf("A1.T = %q, want f (formula should not change)", wb.Sheets[0].Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenNoFormat(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "n", V: float64(42)},
			},
		}},
	}
	normalizeDateCells(wb)
	if wb.Sheets[0].Cells["A1"].T != "n" {
		t.Errorf("A1.T = %q, want n (no format code)", wb.Sheets[0].Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NilCells(t *testing.T) {
	wb := &Workbook{}
	normalizeDateCells(wb)
}
