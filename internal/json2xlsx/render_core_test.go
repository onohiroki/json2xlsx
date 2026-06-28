package json2xlsx

import (
	"encoding/json"
	"testing"
)

func TestBuildCellGrid_Empty(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{})
	if ok {
		t.Fatal("expected false for empty sheet")
	}
}

func TestBuildCellGrid_EmptyCells(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{}})
	if ok {
		t.Fatal("expected false for sheet with empty cells map")
	}
}

func TestBuildCellGrid_SingleCell(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{"A1": {V: float64(42)}}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 1 || cg.MaxRow != 1 {
		t.Fatalf("expected 1x1, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.ColNames[1] != "A" {
		t.Fatalf("expected colName[1]=A, got %q", cg.ColNames[1])
	}
	if cg.Rows[1][1].V != float64(42) {
		t.Fatalf("expected cell A1=42, got %v", cg.Rows[1][1].V)
	}
}

func TestBuildCellGrid_MultipleCells(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1": {V: "x"},
		"C3": {V: "y"},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 3 || cg.MaxRow != 3 {
		t.Fatalf("expected 3x3, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.Rows[1][1].V != "x" {
		t.Fatalf("expected A1=x, got %v", cg.Rows[1][1].V)
	}
	if cg.Rows[3][3].V != "y" {
		t.Fatalf("expected C3=y, got %v", cg.Rows[3][3].V)
	}
}

func TestBuildCellGrid_SparseCells_EmptyIntermediate(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1": {V: "first"},
		"Z1": {V: "last"},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 26 || cg.MaxRow != 1 {
		t.Fatalf("expected 26x1, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.ColNames[26] != "Z" {
		t.Fatalf("expected colNames[26]=Z, got %q", cg.ColNames[26])
	}
	// 中間セルは空
	var emptyCell Cell
	if cg.Rows[1][2] != emptyCell {
		t.Fatal("expected intermediate cell B1 to be zero-value")
	}
}

func TestBuildCellGrid_RowsOnly(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{
		Rows: [][]interface{}{
			{"a", "b", "c"},
			{1, 2},
		},
	})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 3 || cg.MaxRow != 2 {
		t.Fatalf("expected 3x2, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.Rows[1][1].V != "a" || cg.Rows[1][2].V != "b" || cg.Rows[1][3].V != "c" {
		t.Fatalf("first row mismatch: got %v %v %v", cg.Rows[1][1].V, cg.Rows[1][2].V, cg.Rows[1][3].V)
	}
	if cg.Rows[2][1].V != 1 || cg.Rows[2][2].V != 2 {
		t.Fatalf("second row mismatch: got %v %v", cg.Rows[2][1].V, cg.Rows[2][2].V)
	}
	// Rows[2][3] should be zero-value
	var emptyCell Cell
	if cg.Rows[2][3] != emptyCell {
		t.Fatal("expected Row2[3] to be zero-value")
	}
}

func TestBuildCellGrid_InvalidAxis(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"B2":       {V: float64(1)},
		"INVALID":  {V: float64(2)},
		"":         {V: float64(3)},
		"1A":       {V: float64(4)},
	}})
	if !ok {
		t.Fatal("expected true despite invalid axes")
	}
}

func TestBuildCellGrid_ColNames(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1": {},
		"AA1": {},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.ColNames[1] != "A" {
		t.Fatalf("col 1 expected A, got %q", cg.ColNames[1])
	}
	if cg.ColNames[27] != "AA" {
		t.Fatalf("col 27 expected AA, got %q", cg.ColNames[27])
	}
}

func TestCellDisplayValue_FormulaModeFormula(t *testing.T) {
	var w bool
	cases := []struct {
		cell Cell
		want string
	}{
		{Cell{T: "f", F: "1+2", V: float64(3)}, "=1+2"},
		{Cell{T: "f", F: "A1+B1"}, "=A1+B1"},
		{Cell{T: "n", V: float64(42)}, "42"},
		{Cell{T: "s", V: "hello"}, "hello"},
		{Cell{T: "b", V: true}, "true"},
		{Cell{T: "b", V: false}, "false"},
		{Cell{V: nil}, ""},
	}
	for _, c := range cases {
		got := CellDisplayValue(c.cell, MarkdownModeFormula, &w)
		if got != c.want {
			t.Errorf("CellDisplayValue(%+v, formula) = %q, want %q", c.cell, got, c.want)
		}
	}
}

func TestCellDisplayValue_FormulaModeValue(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "1+2", V: float64(3)}, MarkdownModeValue, &w)
	if got != "3" {
		t.Fatalf("expected value 3, got %q", got)
	}
	if w {
		t.Fatal("expected no warning when value present")
	}
}

func TestCellDisplayValue_FormulaModeValue_Fallback(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "A1+B1"}, MarkdownModeValue, &w)
	if got != "=A1+B1" {
		t.Fatalf("expected formula fallback, got %q", got)
	}
	if !w {
		t.Fatal("expected warning when formula without value in mode=v")
	}
}

func TestCellDisplayValue_FormulaModeBoth(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "1+2", V: float64(3)}, MarkdownModeBoth, &w)
	if got != "3<br />=1+2" {
		t.Fatalf("expected both value+formula, got %q", got)
	}
	if w {
		t.Fatal("expected no warning when both present")
	}
}

func TestCellDisplayValue_FormulaModeBoth_Fallback(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "A1+B1"}, MarkdownModeBoth, &w)
	if got != "=A1+B1" {
		t.Fatalf("expected formula fallback, got %q", got)
	}
	if !w {
		t.Fatal("expected warning when formula without value in mode=both")
	}
}

func TestCellDisplayValue_FormulaModeBoth_ValueOnly(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", V: float64(42)}, MarkdownModeBoth, &w)
	if got != "42" {
		t.Fatalf("expected value 42, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_DefaultWithFormula(t *testing.T) {
	t.Run("mode_value_warning", func(t *testing.T) {
		var w bool
		got := CellDisplayValue(Cell{F: "SUM(A1:A10)"}, MarkdownModeValue, &w)
		if got != "=SUM(A1:A10)" {
			t.Fatalf("expected formula fallback, got %q", got)
		}
		if !w {
			t.Fatal("expected warning for default type with formula in mode=v")
		}
	})
	t.Run("mode_formula_no_warning", func(t *testing.T) {
		var w bool
		got := CellDisplayValue(Cell{F: "SUM(A1:A10)"}, MarkdownModeFormula, &w)
		if got != "=SUM(A1:A10)" {
			t.Fatalf("expected formula, got %q", got)
		}
		if w {
			t.Fatal("expected no warning in mode=f")
		}
	})
}

func TestCellDisplayValue_DateCell(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "d", V: float64(45678)}, MarkdownModeFormula, &w)
	if got != "2025-01-21T00:00:00" {
		t.Fatalf("expected RFC3339 date, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_DateCellString(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "d", V: "2025-01-21"}, MarkdownModeFormula, &w)
	if got != "2025-01-21" {
		t.Fatalf("expected string passthrough, got %q", got)
	}
}

func TestCellDisplayValue_TimeOnly(t *testing.T) {
	var w bool
	// 0.04623843 = 1:06:35 (serial * 86400 ≈ 3996 seconds)
	// T は normalizeDateCells により "d" に設定されている想定
	got := CellDisplayValue(Cell{T: "d", V: float64(0.04623843), Z: "h:mm:ss"}, MarkdownModeFormula, &w)
	if got != "1:06:35" {
		t.Fatalf("expected time 1:06:35, got %q", got)
	}
}

func TestCellDisplayValue_NoValueNoFormula(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "s"}, MarkdownModeFormula, &w)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_HasWarningPreserved(t *testing.T) {
	// Warning should be set by pointer even when called multiple times
	var w bool
	CellDisplayValue(Cell{T: "f", F: "X"}, MarkdownModeValue, &w)
	if !w {
		t.Fatal("expected warning after first call")
	}

	// Reset and call without warning condition
	w = false
	CellDisplayValue(Cell{T: "s", V: "ok"}, MarkdownModeValue, &w)
	if w {
		t.Fatal("expected no warning for normal cell")
	}
}

func TestScalarToString_Nil(t *testing.T) {
	if got := scalarToString(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestScalarToString_String(t *testing.T) {
	if got := scalarToString("hello"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestScalarToString_Bool(t *testing.T) {
	if got := scalarToString(true); got != "true" {
		t.Fatalf("expected true, got %q", got)
	}
	if got := scalarToString(false); got != "false" {
		t.Fatalf("expected false, got %q", got)
	}
}

func TestScalarToString_Float64(t *testing.T) {
	if got := scalarToString(float64(42)); got != "42" {
		t.Fatalf("expected 42, got %q", got)
	}
	if got := scalarToString(float64(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %q", got)
	}
}

func TestScalarToString_IntTypes(t *testing.T) {
	if got := scalarToString(42); got != "42" {
		t.Fatalf("expected 42, got %q", got)
	}
	if got := scalarToString(int64(99)); got != "99" {
		t.Fatalf("expected 99, got %q", got)
	}
	if got := scalarToString(float32(1.5)); got != "1.5" {
		t.Fatalf("expected 1.5, got %q", got)
	}
}

func TestScalarToString_JSONNumber(t *testing.T) {
	jn := json.Number("12345")
	if got := scalarToString(jn); got != "12345" {
		t.Fatalf("expected 12345, got %q", got)
	}
}

func TestScalarToString_UnknownType(t *testing.T) {
	type custom struct{ X int }
	got := scalarToString(custom{X: 5})
	if got != "{5}" {
		t.Fatalf("expected {5}, got %q", got)
	}
}

func TestToFloat64_Nil(t *testing.T) {
	if got := toFloat64(nil); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestToFloat64_NumericTypes(t *testing.T) {
	if got := toFloat64(float64(3.14)); got != 3.14 {
		t.Fatalf("expected 3.14, got %f", got)
	}
	if got := toFloat64(float32(2.5)); got != 2.5 {
		t.Fatalf("expected 2.5, got %f", got)
	}
	if got := toFloat64(42); got != 42 {
		t.Fatalf("expected 42, got %f", got)
	}
	if got := toFloat64(int64(99)); got != 99 {
		t.Fatalf("expected 99, got %f", got)
	}
	if got := toFloat64("123.45"); got != 123.45 {
		t.Fatalf("expected 123.45, got %f", got)
	}
}

func TestToFloat64_JSONNumber(t *testing.T) {
	jn := json.Number("456.78")
	if got := toFloat64(jn); got != 456.78 {
		t.Fatalf("expected 456.78, got %f", got)
	}
}

func TestToFloat64_UnparseableString(t *testing.T) {
	if got := toFloat64("not-a-number"); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestToFloat64_UnknownType(t *testing.T) {
	if got := toFloat64(struct{}{}); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

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
	// 1:06:35 → serial 0.04623843 → 3996 seconds
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
	// 3661 seconds = 61:01 (mm:ss)
	got := formatTimeOnly(3661.0/86400, "mm:ss")
	if got != "61:01" {
		t.Fatalf("expected 61:01, got %q", got)
	}
}

func TestFormatTimeOnly_MinutesSecondsNoHours(t *testing.T) {
	// 150 seconds = 2:30 (mm:ss without hours)
	got := formatTimeOnly(150.0/86400, "mm:ss")
	if got != "02:30" {
		t.Fatalf("expected 02:30, got %q", got)
	}
}

func TestFormatTimeOnly_HourBracket(t *testing.T) {
	// 90000 seconds = 25:00:00 (elapsed hours format [h]:mm:ss)
	got := formatTimeOnly(90000.0/86400, "[h]:mm:ss")
	if got != "25:00:00" {
		t.Fatalf("expected 25:00:00, got %q", got)
	}
}

func TestNormalizeDateCells_SingleSheet(t *testing.T) {
	wb := &Workbook{
		Cells: map[string]Cell{
			"A1": {T: "n", V: float64(45000), Z: "yyyy-mm-dd"},
			"B1": {T: "s", V: "hello"}, // 日付書式なし
		},
	}
	normalizeDateCells(wb)
	if wb.Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d", wb.Cells["A1"].T)
	}
	if wb.Cells["B1"].T != "s" {
		t.Errorf("B1.T = %q, want s (unchanged)", wb.Cells["B1"].T)
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
		Book: &Book{
			Sheets: map[string]Sheet{
				"S1": {Cells: map[string]Cell{"A1": {T: "n", V: float64(45000), Z: "yyyy-mm-dd"}}},
			},
		},
	}
	normalizeDateCells(wb)
	if wb.Book.Sheets["S1"].Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d", wb.Book.Sheets["S1"].Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenAlreadyDate(t *testing.T) {
	wb := &Workbook{
		Cells: map[string]Cell{
			"A1": {T: "d", V: float64(45000), Z: "yyyy-mm-dd"}, // 既に t=d
		},
	}
	normalizeDateCells(wb)
	if wb.Cells["A1"].T != "d" {
		t.Errorf("A1.T = %q, want d (should remain d)", wb.Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenFormula(t *testing.T) {
	wb := &Workbook{
		Cells: map[string]Cell{
			"A1": {T: "f", F: "TODAY()", Z: "yyyy-mm-dd"}, // t=f はスキップ
		},
	}
	normalizeDateCells(wb)
	if wb.Cells["A1"].T != "f" {
		t.Errorf("A1.T = %q, want f (formula should not change)", wb.Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NoChangeWhenNoFormat(t *testing.T) {
	wb := &Workbook{
		Cells: map[string]Cell{
			"A1": {T: "n", V: float64(42)}, // z なし
		},
	}
	normalizeDateCells(wb)
	if wb.Cells["A1"].T != "n" {
		t.Errorf("A1.T = %q, want n (no format code)", wb.Cells["A1"].T)
	}
}

func TestNormalizeDateCells_NilCells(t *testing.T) {
	// nil Cells / nil Sheets / nil Book でも panic しない
	wb := &Workbook{}
	normalizeDateCells(wb)
	// panic しなければ成功
}
