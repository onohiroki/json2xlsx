package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_Averageif(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 10.0},
		"A4": {T: "n", V: 30.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
		"B3": {T: "n", V: 300.0},
		"B4": {T: "n", V: 400.0},
	}
	got := evalFormula(t, cells, "AVERAGEIF(A1:A4,10,B1:B4)")
	// Rows 1 and 3 match: values 100 and 300 → average 200
	if got != 200 {
		t.Errorf("AVERAGEIF = %v, want 200", got)
	}
}

func TestEval_AverageifNoAvgRange(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "AVERAGEIF(A1:A3,10)")
	// Rows 1 and 3 match: 10 and 10 → average 10
	if got != 10 {
		t.Errorf("AVERAGEIF(A1:A3,10) = %v, want 10", got)
	}
}

func TestEval_AverageifWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "AVERAGEIF(A1:A2)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_AverageifNoMatch(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
	}
	errMsg := evalFormulaErr(t, cells, "AVERAGEIF(A1:A2,99)")
	if !strings.Contains(errMsg, "#DIV/0") {
		t.Errorf("expected #DIV/0 error, got %q", errMsg)
	}
}

func TestEval_Sumifs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 100.0},
		"A2": {T: "n", V: 200.0},
		"A3": {T: "n", V: 300.0},
		"B1": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
		"B3": {T: "n", V: 10.0},
		"C1": {T: "n", V: 1.0},
		"C2": {T: "n", V: 1.0},
		"C3": {T: "n", V: 2.0},
	}
	// SUMIFS(A1:A3, B1:B3, 10, C1:C3, 1)
	// Row 1: B=10, C=1 → match → sum 100
	// Row 2: B=20 → no match
	// Row 3: B=10, C=2 → no match (C≠1)
	got := evalFormula(t, cells, "SUMIFS(A1:A3,B1:B3,10,C1:C3,1)")
	if got != 100 {
		t.Errorf("SUMIFS = %v, want 100", got)
	}
}

func TestEval_SumifsNoMatch(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 100.0},
		"B1": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "SUMIFS(A1,B1,99)")
	if got != 0 {
		t.Errorf("SUMIFS no match = %v, want 0", got)
	}
}

func TestEval_SumifsSinglePair(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 100.0},
		"A2": {T: "n", V: 200.0},
		"B1": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
	}
	got := evalFormula(t, cells, "SUMIFS(A1:A2,B1:B2,10)")
	if got != 100 {
		t.Errorf("SUMIFS single pair = %v, want 100", got)
	}
}

func TestEval_SumifsWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SUMIFS(A1:A2)")
	if !strings.Contains(errMsg, "pairs") {
		t.Errorf("expected pairs error, got %q", errMsg)
	}
}

func TestEval_Countifs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 10.0},
		"A4": {T: "n", V: 30.0},
		"B1": {T: "n", V: 1.0},
		"B2": {T: "n", V: 2.0},
		"B3": {T: "n", V: 1.0},
		"B4": {T: "n", V: 3.0},
	}
	got := evalFormula(t, cells, "COUNTIFS(A1:A4,10,B1:B4,1)")
	// Rows 1 and 3 match: A=10 and B=1 → count 2
	if got != 2 {
		t.Errorf("COUNTIFS = %v, want 2", got)
	}
}

func TestEval_CountifsNoMatch(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
	}
	got := evalFormula(t, cells, "COUNTIFS(A1,99)")
	if got != 0 {
		t.Errorf("COUNTIFS no match = %v, want 0", got)
	}
}

func TestEval_CountifsWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "COUNTIFS(1)")
	if !strings.Contains(errMsg, "pairs") {
		t.Errorf("expected pairs error, got %q", errMsg)
	}
}

func TestEval_Averageifs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 100.0},
		"A2": {T: "n", V: 200.0},
		"A3": {T: "n", V: 300.0},
		"B1": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
		"B3": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "AVERAGEIFS(A1:A3,B1:B3,10)")
	// Rows 1 and 3 match: 100 and 300 → average 200
	if got != 200 {
		t.Errorf("AVERAGEIFS = %v, want 200", got)
	}
}

func TestEval_AverageifsWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "AVERAGEIFS(A1:A2)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_AverageifsNoMatch(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 100.0},
		"B1": {T: "n", V: 10.0},
	}
	errMsg := evalFormulaErr(t, cells, "AVERAGEIFS(A1,B1,99)")
	if !strings.Contains(errMsg, "#DIV/0") {
		t.Errorf("expected #DIV/0 error, got %q", errMsg)
	}
}
