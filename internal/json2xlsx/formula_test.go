package json2xlsx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// evalFormula is a test helper: evaluates a single formula string
// against a cells map and returns the result.
func evalFormula(t *testing.T, cells map[string]Cell, formula string) float64 {
	t.Helper()
	ctx := newEvalContext(cells)
	val, err := ctx.evaluate("_test", formula)
	if err != nil {
		t.Fatalf("eval %q: %v", formula, err)
	}
	return val
}

// evalFormulaErr is a test helper: evaluates a formula that is expected to fail.
func evalFormulaErr(t *testing.T, cells map[string]Cell, formula string) string {
	t.Helper()
	ctx := newEvalContext(cells)
	_, err := ctx.evaluate("_test", formula)
	if err == nil {
		t.Fatalf("eval %q: expected error", formula)
	}
	return err.Error()
}

func TestEval_Number(t *testing.T) {
	got := evalFormula(t, nil, "42")
	if got != 42 {
		t.Errorf("got %v, want 42", got)
	}
}

func TestEval_Decimal(t *testing.T) {
	got := evalFormula(t, nil, "3.14")
	if got != 3.14 {
		t.Errorf("got %v, want 3.14", got)
	}
}

func TestEval_BasicArithmetic(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"1+2", 3},
		{"3-1", 2},
		{"2*3", 6},
		{"10/2", 5},
		{"1+2*3", 7},
		{"(1+2)*3", 9},
		{"10-2*3", 4},
		{"(10-2)*3", 24},
		{"-5", -5},
		{"-5+3", -2},
		{"-(5+3)", -8},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_DivisionByZero(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "1/0")
	if !strings.Contains(errMsg, "division by zero") {
		t.Errorf("expected 'division by zero', got %q", errMsg)
	}
}

func TestEval_CellReference(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
	}
	got := evalFormula(t, cells, "A1+A2")
	if got != 30 {
		t.Errorf("got %v, want 30", got)
	}
}

func TestEval_CellReferenceMissing(t *testing.T) {
	got := evalFormula(t, nil, "Z999+1")
	if got != 1 {
		t.Errorf("got %v, want 1 (missing cell treated as 0)", got)
	}
}

func TestEval_CellReferenceCaseInsensitive(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
	}
	got := evalFormula(t, cells, "a1*2")
	if got != 10 {
		t.Errorf("got %v, want 10", got)
	}
}

func TestEval_AbsoluteRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
	}
	got := evalFormula(t, cells, "$A$1*2")
	if got != 10 {
		t.Errorf("$A$1: got %v, want 10", got)
	}
}

func TestEval_MixedRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 3.0},
		"B1": {T: "n", V: 4.0},
	}
	tests := []struct {
		ref  string
		want float64
	}{
		{"$A1+B1", 7},
		{"A$1+$B$1", 7},
		{"$A$1+$B1", 7},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := evalFormula(t, cells, tt.ref)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_Whitespace(t *testing.T) {
	got := evalFormula(t, nil, "  1  +  2  ")
	if got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}

func TestEval_UnaryMinusPrecedence(t *testing.T) {
	got := evalFormula(t, nil, "-2*3")
	if got != -6 {
		t.Errorf("got %v, want -6", got)
	}
}

// ---------------------------------------------------------------------------
// Function tests
// ---------------------------------------------------------------------------

func TestEval_Sum(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0},
		"B1": {T: "n", V: 10.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"SUM(A1:A3)", 6},
		{"SUM(A1:A3,B1)", 16},
		{"SUM(A1,B1)", 11},
		{"SUM(100)", 100},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_SumWithExpression(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
		"A2": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "SUM(A1:A2)*2")
	if got != 30 {
		t.Errorf("got %v, want 30", got)
	}
}

func TestEval_Average(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"A4": {T: "n", V: 40.0},
	}
	got := evalFormula(t, cells, "AVERAGE(A1:A4)")
	if got != 25 {
		t.Errorf("got %v, want 25", got)
	}
}

func TestEval_AverageEmpty(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "AVERAGE(Z1:Z999)")
	if !strings.Contains(errMsg, "AVERAGE of empty range") {
		t.Errorf("expected 'AVERAGE of empty range', got %q", errMsg)
	}
}

func TestEval_Count(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"A3": {T: "s", V: "hello"},
		"A4": {T: "n", V: 4.0},
	}
	got := evalFormula(t, cells, "COUNT(A1:A4)")
	if got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}

func TestEval_CountEmptyRange(t *testing.T) {
	got := evalFormula(t, nil, "COUNT(Z1:Z999)")
	if got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestEval_Min(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 3.0},
		"A3": {T: "n", V: 7.0},
	}
	got := evalFormula(t, cells, "MIN(A1:A3)")
	if got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}

func TestEval_Max(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 3.0},
		"A3": {T: "n", V: 7.0},
	}
	got := evalFormula(t, cells, "MAX(A1:A3)")
	if got != 10 {
		t.Errorf("got %v, want 10", got)
	}
}

func TestEval_Abs(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"ABS(-5)", 5},
		{"ABS(5)", 5},
		{"ABS(0)", 0},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_AbsWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "ABS(1,2)")
	if !strings.Contains(errMsg, "requires exactly 1 argument") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Round(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"ROUND(3.1415,2)", 3.14},
		{"ROUND(3.1415,0)", 3},
		{"ROUND(3.1415,1)", 3.1},
		{"ROUND(5.6789,2)", 5.68},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEval_RoundWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "ROUND(1)")
	if !strings.Contains(errMsg, "requires exactly 2 arguments") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_NestedFunc(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0},
		"B1": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
		"B3": {T: "n", V: 30.0},
	}
	got := evalFormula(t, cells, "SUM(A1:A3)*AVERAGE(B1:B3)")
	if got != 6*20 { // SUM=6, AVERAGE=20
		t.Errorf("got %v, want 120", got)
	}
}

func TestEval_FuncCaseInsensitive(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
	}
	tests := []string{"sum(A1:A2)", "Sum(A1:A2)", "SUM(A1:A2)"}
	for _, formula := range tests {
		t.Run(formula, func(t *testing.T) {
			got := evalFormula(t, cells, formula)
			if got != 3 {
				t.Errorf("got %v, want 3", got)
			}
		})
	}
}

func TestEval_AllFunctions_MultipleArgs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"B1": {T: "n", V: 5.0},
	}
	got := evalFormula(t, cells, "SUM(A1:A2,B1)")
	if got != 35 {
		t.Errorf("got %v, want 35", got)
	}
}

func TestEval_MinMax_MultipleArgs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"B1": {T: "n", V: 5.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"MIN(A1:A2,B1)", 5},
		{"MAX(A1:A2,B1)", 20},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error tests
// ---------------------------------------------------------------------------

func TestEval_CircularRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "f", F: "B1+1"},
		"B1": {T: "f", F: "A1+1"},
	}
	ctx := newEvalContext(cells)
	_, err := ctx.evaluate("A1", cells["A1"].F)
	if err == nil {
		t.Fatal("expected circular reference error")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("expected 'circular reference', got %q", err.Error())
	}
}

func TestEval_RangeOutsideFunction(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "A1:A3")
	if !strings.Contains(errMsg, "cannot be used outside a function") {
		t.Errorf("expected range error, got %q", errMsg)
	}
}

func TestEval_UnknownFunction(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "FOO(1)")
	if !strings.Contains(errMsg, "unexpected token") {
		t.Errorf("expected 'unexpected token', got %q", errMsg)
	}
}

func TestEval_IllegalToken(t *testing.T) {
	_, err := newParser("@bad").parse()
	if err == nil {
		t.Error("expected parse error for illegal token")
	}
}

func TestEval_MissingParen(t *testing.T) {
	_, err := newParser("SUM(A1").parse()
	if err == nil {
		t.Error("expected parse error for missing paren")
	}
}

// ---------------------------------------------------------------------------
// EvalWorkbookFormulas integration tests
// ---------------------------------------------------------------------------

func TestEvalWorkbookFormulas_Basic(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "Sheet1",
			Cells: map[string]Cell{
				"A1": {T: "n", V: 10.0},
				"A2": {T: "n", V: 20.0},
				"A3": {T: "f", F: "SUM(A1:A2)"},
			},
		}},
	}
	EvalWorkbookFormulas(wb)
	cell := wb.Sheets[0].Cells["A3"]
	if cell.V == nil {
		t.Fatal("A3.V is nil after evaluation")
	}
	v, ok := cell.V.(float64)
	if !ok {
		t.Fatalf("A3.V type = %T, want float64", cell.V)
	}
	if v != 30 {
		t.Errorf("A3.V = %v, want 30", v)
	}
}

func TestEvalWorkbookFormulas_MultipleSheets(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{
				Name: "Sheet1",
				Cells: map[string]Cell{
					"A1": {T: "n", V: 1.0},
					"A2": {T: "f", F: "A1+1"},
				},
			},
			{
				Name: "Sheet2",
				Cells: map[string]Cell{
					"B1": {T: "n", V: 100.0},
					"B2": {T: "f", F: "B1*2"},
				},
			},
		},
	}
	EvalWorkbookFormulas(wb)

	// Sheet1.A2
	cell1 := wb.Sheets[0].Cells["A2"]
	if cell1.V == nil {
		t.Fatal("Sheet1 A2.V is nil")
	}
	v1, _ := cell1.V.(float64)
	if v1 != 2 {
		t.Errorf("Sheet1 A2 = %v, want 2", v1)
	}

	// Sheet2.B2
	cell2 := wb.Sheets[1].Cells["B2"]
	if cell2.V == nil {
		t.Fatal("Sheet2 B2.V is nil")
	}
	v2, _ := cell2.V.(float64)
	if v2 != 200 {
		t.Errorf("Sheet2 B2 = %v, want 200", v2)
	}
}

func TestEvalWorkbookFormulas_SkipAlreadyComputed(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "Sheet1",
			Cells: map[string]Cell{
				"A1": {T: "n", V: 5.0},
				"A2": {T: "f", F: "A1+1", V: 6.0}, // already has v
			},
		}},
	}
	EvalWorkbookFormulas(wb)
	cell := wb.Sheets[0].Cells["A2"]
	v, _ := cell.V.(float64)
	if v != 6 {
		t.Errorf("A2.V should remain 6, got %v", v)
	}
}

func TestEvalWorkbookFormulas_FormulaCellWithValue(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "Sheet1",
			Cells: map[string]Cell{
				"A1": {T: "n", V: 5.0},
				"A2": {T: "f", F: "A1+1", V: 999.0}, // existing cached value
			},
		}},
	}
	EvalWorkbookFormulas(wb)
	cell := wb.Sheets[0].Cells["A2"]
	v, _ := cell.V.(float64)
	if v != 999 {
		t.Errorf("A2.V should keep existing 999, got %v", v)
	}
}

func TestEvalWorkbookFormulas_ChainedFormulas(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "Sheet1",
			Cells: map[string]Cell{
				"A1": {T: "n", V: 10.0},
				"A2": {T: "f", F: "A1*2"},
				"A3": {T: "f", F: "A2+5"},
			},
		}},
	}
	EvalWorkbookFormulas(wb)

	a2 := wb.Sheets[0].Cells["A2"]
	v2, _ := a2.V.(float64)
	if v2 != 20 {
		t.Errorf("A2 = %v, want 20", v2)
	}

	a3 := wb.Sheets[0].Cells["A3"]
	v3, _ := a3.V.(float64)
	if v3 != 25 {
		t.Errorf("A3 = %v, want 25", v3)
	}
}

func TestEvalWorkbookFormulas_CircularRef(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Name: "Sheet1",
			Cells: map[string]Cell{
				"A1": {T: "f", F: "B1+1"},
				"B1": {T: "f", F: "A1+1"},
			},
		}},
	}
	EvalWorkbookFormulas(wb)
	// Both should remain nil (evaluation skipped due to circular reference)
	if wb.Sheets[0].Cells["A1"].V != nil {
		t.Error("A1.V should be nil (circular reference)")
	}
	if wb.Sheets[0].Cells["B1"].V != nil {
		t.Error("B1.V should be nil (circular reference)")
	}
}

// ---------------------------------------------------------------------------
// Integration test: Convert with EvalFormulas
// ---------------------------------------------------------------------------

func TestComputeFlag_Integration(t *testing.T) {
	js := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "n", "v": 10},
			"A2": {"t": "n", "v": 20},
			"A3": {"t": "f", "f": "SUM(A1:A2)"}
		}
	}`
	f := convertJSONtoXLSXWithOpts(t, js, ConvertOptions{EvalFormulas: true})
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue A3: %v", err)
	}
	if val != "30" {
		t.Errorf("A3 = %q, want \"30\"", val)
	}
}

func TestComputeFlag_WithoutFlag(t *testing.T) {
	js := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "n", "v": 10},
			"A2": {"t": "n", "v": 20},
			"A3": {"t": "f", "f": "SUM(A1:A2)"}
		}
	}`
	f := convertJSONtoXLSXWithOpts(t, js, ConvertOptions{})
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue A3: %v", err)
	}
	// Without --compute, formula cell has no cached value,
	// so excelize returns empty string.
	if val != "" {
		t.Errorf("A3 = %q, want empty string (no compute)", val)
	}
}

func TestComputeFlag_FormulaCellWithExistingValue(t *testing.T) {
	js := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "n", "v": 10},
			"A2": {"t": "n", "v": 20},
			"A3": {"t": "f", "f": "SUM(A1:A2)", "v": 999}
		}
	}`
	f := convertJSONtoXLSXWithOpts(t, js, ConvertOptions{EvalFormulas: true})
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue A3: %v", err)
	}
	if val != "999" {
		t.Errorf("A3 = %q, want \"999\" (existing value should be preserved)", val)
	}
}

// TestComputeFlag_DataJSON_Rows verifies that Cells in Rows format
// are not evaluated (formulas not supported in that format).
func TestComputeFlag_DataJSON_Rows(t *testing.T) {
	js := `[[1, 2, 3]]`
	f := convertJSONtoXLSXWithOpts(t, js, ConvertOptions{DataJSON: true, EvalFormulas: true})
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "C1")
	if err != nil {
		t.Fatalf("GetCellValue C1: %v", err)
	}
	if val != "3" {
		t.Errorf("C1 = %q, want \"3\"", val)
	}
}

// ---------------------------------------------------------------------------
// Tokenizer unit tests
// ---------------------------------------------------------------------------

func TestTokenizer_Numbers(t *testing.T) {
	tok := newTokenizer("123")
	tk := tok.next()
	if tk.typ != tokenNumber || tk.lit != "123" {
		t.Errorf("got %v %q, want tokenNumber 123", tk.typ, tk.lit)
	}
}

func TestTokenizer_Decimal(t *testing.T) {
	tok := newTokenizer("3.14")
	tk := tok.next()
	if tk.typ != tokenNumber || tk.lit != "3.14" {
		t.Errorf("got %v %q, want tokenNumber 3.14", tk.typ, tk.lit)
	}
}

func TestTokenizer_CellRef(t *testing.T) {
	tok := newTokenizer("A1")
	tk := tok.next()
	if tk.typ != tokenCellRef || tk.lit != "A1" {
		t.Errorf("got %v %q, want tokenCellRef A1", tk.typ, tk.lit)
	}
}

func TestTokenizer_AbsoluteRef(t *testing.T) {
	tok := newTokenizer("$A$1")
	tk := tok.next()
	if tk.typ != tokenCellRef || tk.lit != "A1" {
		t.Errorf("got %v %q, want tokenCellRef A1", tk.typ, tk.lit)
	}
}

func TestTokenizer_MixedRef(t *testing.T) {
	tok := newTokenizer("$A1")
	tk := tok.next()
	if tk.typ != tokenCellRef || tk.lit != "A1" {
		t.Errorf("got %v %q, want tokenCellRef A1", tk.typ, tk.lit)
	}
}

func TestTokenizer_FuncName(t *testing.T) {
	tok := newTokenizer("SUM")
	tk := tok.next()
	if tk.typ != tokenFunc || tk.lit != "SUM" {
		t.Errorf("got %v %q, want tokenFunc SUM", tk.typ, tk.lit)
	}
}

func TestTokenizer_FuncCaseInsensitive(t *testing.T) {
	tok := newTokenizer("sum")
	tk := tok.next()
	if tk.typ != tokenFunc || tk.lit != "SUM" {
		t.Errorf("got %v %q, want tokenFunc SUM", tk.typ, tk.lit)
	}
}

func TestTokenizer_Operators(t *testing.T) {
	tests := []struct {
		input string
		typ   tokenType
		lit   string
	}{
		{"+", tokenPlus, "+"},
		{"-", tokenMinus, "-"},
		{"*", tokenStar, "*"},
		{"/", tokenSlash, "/"},
		{"(", tokenLParen, "("},
		{")", tokenRParen, ")"},
		{",", tokenComma, ","},
		{":", tokenColon, ":"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := newTokenizer(tt.input)
			tk := tok.next()
			if tk.typ != tt.typ || tk.lit != tt.lit {
				t.Errorf("got %v %q, want %v %q", tk.typ, tk.lit, tt.typ, tt.lit)
			}
		})
	}
}

func TestTokenizer_Whitespace(t *testing.T) {
	tok := newTokenizer("  A1  +  B1  ")
	tks := []tokenType{tokenCellRef, tokenPlus, tokenCellRef, tokenEOF}
	for i, want := range tks {
		tk := tok.next()
		if tk.typ != want {
			t.Errorf("token %d: got %v, want %v", i, tk.typ, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Tokenizer comparison operator tests
// ---------------------------------------------------------------------------

func TestTokenizer_ComparisonOperators(t *testing.T) {
	tests := []struct {
		input string
		typ   tokenType
		lit   string
	}{
		{"<", tokenLT, "<"},
		{">", tokenGT, ">"},
		{"=", tokenEQ, "="},
		{"<=", tokenLE, "<="},
		{">=", tokenGE, ">="},
		{"<>", tokenNE, "<>"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := newTokenizer(tt.input)
			tk := tok.next()
			if tk.typ != tt.typ || tk.lit != tt.lit {
				t.Errorf("got %v %q, want %v %q", tk.typ, tk.lit, tt.typ, tt.lit)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cell reference utility tests
// ---------------------------------------------------------------------------

func TestNormalizeCellRef(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"A1", "A1"},
		{"a1", "A1"},
		{"aa10", "AA10"},
		{"Bc123", "BC123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeCellRef(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseCellRef(t *testing.T) {
	col, row := parseCellRef("A1")
	if col != 1 || row != 1 {
		t.Errorf("A1: col=%d row=%d, want 1 1", col, row)
	}
	col, row = parseCellRef("B2")
	if col != 2 || row != 2 {
		t.Errorf("B2: col=%d row=%d, want 2 2", col, row)
	}
	col, row = parseCellRef("AA10")
	if col != 27 || row != 10 {
		t.Errorf("AA10: col=%d row=%d, want 27 10", col, row)
	}
}

func TestFormatCellRef(t *testing.T) {
	tests := []struct {
		col, row int
		want     string
	}{
		{1, 1, "A1"},
		{2, 2, "B2"},
		{27, 10, "AA10"},
	}
	for _, tt := range tests {
		got := formatCellRef(tt.col, tt.row)
		if got != tt.want {
			t.Errorf("(%d,%d) = %q, want %q", tt.col, tt.row, got, tt.want)
		}
	}
}

func TestExpandRange(t *testing.T) {
	refs := expandRange("A1", "C3")
	want := []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d", len(refs), len(want))
	}
	for i := range want {
		if refs[i] != want[i] {
			t.Errorf("refs[%d] = %q, want %q", i, refs[i], want[i])
		}
	}
}

func TestExpandRange_Reversed(t *testing.T) {
	// Reverse order should produce same result
	refs := expandRange("C3", "A1")
	want := []string{"A1", "A2", "A3", "B1", "B2", "B3", "C1", "C2", "C3"}
	if len(refs) != len(want) {
		t.Fatalf("got %d refs, want %d", len(refs), len(want))
	}
	for i := range want {
		if refs[i] != want[i] {
			t.Errorf("refs[%d] = %q, want %q", i, refs[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestEval_EmptyFormula(t *testing.T) {
	_, err := newParser("").parse()
	if err == nil {
		t.Error("expected error for empty formula")
	}
}

func TestEval_FormulaWithOnlyNumber(t *testing.T) {
	got := evalFormula(t, nil, "42")
	if got != 42 {
		t.Errorf("got %v, want 42", got)
	}
}

func TestEval_BinaryPlusUnary(t *testing.T) {
	got := evalFormula(t, nil, "1+-2")
	if got != -1 {
		t.Errorf("got %v, want -1", got)
	}
}

func TestEval_BinaryMinusUnary(t *testing.T) {
	got := evalFormula(t, nil, "5--3")
	if got != 8 {
		t.Errorf("got %v, want 8", got)
	}
}

func TestEval_MultipleRanges(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"B1": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
	}
	got := evalFormula(t, cells, "SUM(A1:A2, B1:B2)")
	if got != 33 {
		t.Errorf("got %v, want 33", got)
	}
}

func TestEval_SumEmptyRange(t *testing.T) {
	got := evalFormula(t, nil, "SUM(Z1:Z999)")
	if got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestEval_SumMissingCells(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
		"A3": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "SUM(A1:A3)")
	// A2 is missing -- should be skipped (5+10=15)
	if got != 15 {
		t.Errorf("got %v, want 15", got)
	}
}

func TestEval_RangeReversed(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0},
	}
	got := evalFormula(t, cells, "SUM(A3:A1)")
	if got != 6 {
		t.Errorf("got %v, want 6", got)
	}
}

func TestEval_CountIndividualRefs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "s", V: "hello"},
		"A3": {T: "n", V: 3.0},
	}
	got := evalFormula(t, cells, "COUNT(A1, A2, A3)")
	// A2 is string → not counted
	if got != 2 {
		t.Errorf("got %v, want 2", got)
	}
}

func TestEval_CountWithLiterals(t *testing.T) {
	got := evalFormula(t, nil, "COUNT(1, 2, 3)")
	if got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}

func TestEval_AverageSingleValue(t *testing.T) {
	got := evalFormula(t, nil, "AVERAGE(42)")
	if got != 42 {
		t.Errorf("got %v, want 42", got)
	}
}

func TestEval_MinSingleValue(t *testing.T) {
	got := evalFormula(t, nil, "MIN(99)")
	if got != 99 {
		t.Errorf("got %v, want 99", got)
	}
}

func TestEval_MaxSingleValue(t *testing.T) {
	got := evalFormula(t, nil, "MAX(99)")
	if got != 99 {
		t.Errorf("got %v, want 99", got)
	}
}

func TestEval_FormulaCellRefWithString(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "hello"},
	}
	// Referencing a string cell in arithmetic should treat it as 0
	got := evalFormula(t, cells, "A1+5")
	if got != 5 {
		t.Errorf("got %v, want 5", got)
	}
}

func TestEval_BookWrapperWithCompute(t *testing.T) {
	js := `{
		"version": "0.2",
		"book": {
			"sheets": {
				"Sheet1": {
					"cells": {
						"A1": {"t": "n", "v": 10},
						"A2": {"t": "n", "v": 20},
						"A3": {"t": "f", "f": "SUM(A1:A2)"}
					}
				}
			}
		}
	}`

	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{EvalFormulas: true}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	val, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue A3: %v", err)
	}
	if val != "30" {
		t.Errorf("A3 = %q, want \"30\"", val)
	}
}

func TestEval_MultiSheetWithCompute(t *testing.T) {
	js := `{
		"sheets": [
			{
				"name": "S1",
				"cells": {
					"A1": {"t": "n", "v": 5},
					"A2": {"t": "f", "f": "A1*3"}
				}
			},
			{
				"name": "S2",
				"cells": {
					"B1": {"t": "n", "v": 100},
					"B2": {"t": "f", "f": "B1/2"}
				}
			}
		]
	}`

	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{EvalFormulas: true}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	v1, _ := f.GetCellValue("S1", "A2")
	if v1 != "15" {
		t.Errorf("S1 A2 = %q, want \"15\"", v1)
	}

	v2, _ := f.GetCellValue("S2", "B2")
	if v2 != "50" {
		t.Errorf("S2 B2 = %q, want \"50\"", v2)
	}
}

func TestEval_ComputeWithExistingV(t *testing.T) {
	// Cell has both formula and an existing cached value -- should preserve the existing value
	js := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "n", "v": 5},
			"A2": {"t": "f", "f": "A1+1", "v": 999}
		}
	}`

	var buf bytes.Buffer
	if err := Convert(strings.NewReader(js), &buf, ConvertOptions{EvalFormulas: true}); err != nil {
		t.Fatalf("Convert error: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	val, _ := f.GetCellValue("Sheet1", "A2")
	if val != "999" {
		t.Errorf("A2 = %q, want \"999\" (preserve existing cached value)", val)
	}
}

// ---------------------------------------------------------------------------
// Comparison operator tests
// ---------------------------------------------------------------------------

func TestEval_ComparisonOperators(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"1=1", 1},
		{"1=2", 0},
		{"1<>2", 1},
		{"1<>1", 0},
		{"1<2", 1},
		{"2<1", 0},
		{"2>1", 1},
		{"1>2", 0},
		{"1<=2", 1},
		{"1<=1", 1},
		{"3<=2", 0},
		{"2>=1", 1},
		{"2>=2", 1},
		{"1>=2", 0},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_ComparisonPrecedence(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"1+2<3+4", 1},  // 3<7 → TRUE
		{"1+2*3>10", 0}, // 7>10 → FALSE
		{"5-3=2", 1},    // 2=2 → TRUE
		{"10/2<>5", 0},  // 5<>5 → FALSE
		{"2*3>=7", 0},   // 6>=7 → FALSE
		{"2*3<=7", 1},   // 6<=7 → TRUE
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_ChainedComparison(t *testing.T) {
	// Excel semantics: left-associative, 1<2<3 = (1<2)<3 = TRUE<3 = 1<3 = TRUE
	tests := []struct {
		formula string
		want    float64
	}{
		{"1<2<3", 1},     // (1<2)=1, 1<3 → TRUE
		{"3>2>1", 0},     // (3>2)=1, 1>1 → FALSE
		{"1=1=1", 1},     // (1=1)=1, 1=1 → TRUE
		{"1=1=0", 0},     // (1=1)=1, 1=0 → FALSE
		{"1<2=1", 1},     // (1<2)=1, 1=1 → TRUE
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_ComparisonWithCellRefs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 10.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"A1=A3", 1},
		{"A1=A2", 0},
		{"A1<A2", 1},
		{"A2<A1", 0},
		{"A1<>A3", 0},
		{"A1<>A2", 1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IF / AND / OR / NOT tests
// ---------------------------------------------------------------------------

func TestEval_If(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(1,10,20)", 10},
		{"IF(0,10,20)", 20},
		{"IF(5,10,20)", 10},
		{"IF(-1,10,20)", 10},
		{"IF(5-5,10,20)", 20},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_IfWithComparison(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 15.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(A1>10,100,200)", 100},
		{"IF(A1>20,100,200)", 200},
		{"IF(A1=15,1,0)", 1},
		{"IF(A1<>15,1,0)", 0},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_IfShortCircuit(t *testing.T) {
	// Division by zero in non-evaluated branch should NOT cause error
	got := evalFormula(t, nil, "IF(0,1/0,999)")
	if got != 999 {
		t.Errorf("IF(0,1/0,999) = %v, want 999", got)
	}
	got = evalFormula(t, nil, "IF(1,999,1/0)")
	if got != 999 {
		t.Errorf("IF(1,999,1/0) = %v, want 999", got)
	}
}

func TestEval_IfNested(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 85.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(A1>=90,1,IF(A1>=80,2,3))", 2},
		{"IF(A1>=90,1,IF(A1>=80,2,IF(A1>=60,3,4)))", 2},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_IfWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "IF(1)")
	if !strings.Contains(errMsg, "requires exactly 3 arguments") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_And(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"AND(1,1)", 1},
		{"AND(1,0)", 0},
		{"AND(0,1)", 0},
		{"AND(0,0)", 0},
		{"AND(1,1,1)", 1},
		{"AND(1,2,3)", 1},   // all non-zero
		{"AND(1,2,0)", 0},   // last is zero
		{"AND()", 1},         // Excel: AND() = TRUE
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_AndShortCircuit(t *testing.T) {
	// Division by zero in non-evaluated arg should NOT cause error
	got := evalFormula(t, nil, "AND(0,1/0)")
	if got != 0 {
		t.Errorf("AND(0,1/0) = %v, want 0", got)
	}
}

func TestEval_Or(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"OR(0,1)", 1},
		{"OR(0,0)", 0},
		{"OR(1,0)", 1},
		{"OR(1,1)", 1},
		{"OR(0,0,0)", 0},
		{"OR(0,0,1)", 1},
		{"OR()", 0},          // Excel: OR() = FALSE
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_OrShortCircuit(t *testing.T) {
	got := evalFormula(t, nil, "OR(1,1/0)")
	if got != 1 {
		t.Errorf("OR(1,1/0) = %v, want 1", got)
	}
}

func TestEval_Not(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"NOT(0)", 1},
		{"NOT(1)", 0},
		{"NOT(100)", 0},
		{"NOT(-1)", 0},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_NotWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "NOT(1,2)")
	if !strings.Contains(errMsg, "requires exactly 1 argument") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_CombinedLogical(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 15.0},
		"A2": {T: "n", V: 5.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(AND(A1>10,A2>0),100,200)", 100},
		{"IF(AND(A1>10,A2>10),100,200)", 200},
		{"IF(OR(A1>10,A2>10),100,200)", 100},
		{"IF(OR(A1>20,A2>20),100,200)", 200},
		{"IF(NOT(A1>20),100,200)", 100},
		{"IF(AND(A1>10,OR(A2>0,1=0)),1,0)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, cells, tt.formula)
			if got != tt.want {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_IfWithNestedFunc(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"B1": {T: "n", V: 5.0},
	}
	// SUM=60, 60>50 → TRUE, return 60*2=120
	got := evalFormula(t, cells, "IF(SUM(A1:A3)>50,SUM(A1:A3)*2,0)")
	if got != 120 {
		t.Errorf("got %v, want 120", got)
	}

	// SUM=60, 60<50 → FALSE, return 999
	got = evalFormula(t, cells, "IF(SUM(A1:A3)<50,SUM(A1:A3)*2,999)")
	if got != 999 {
		t.Errorf("got %v, want 999", got)
	}
}

// ---------------------------------------------------------------------------
// Helper: convertJSONtoXLSXWithOpts converts JSON with the given options
// and returns an open excelize.File.
// ---------------------------------------------------------------------------

func convertJSONtoXLSXWithOpts(t *testing.T, jsonStr string, opts ConvertOptions) *excelize.File {
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
