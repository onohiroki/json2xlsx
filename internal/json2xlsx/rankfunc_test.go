package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_RankDescending(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"RANK(30,A1:A3)", 1},
		{"RANK(20,A1:A3)", 2},
		{"RANK(10,A1:A3)", 3},
		{"RANK(30,A1:A3,0)", 1},
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

func TestEval_RankAscending(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"RANK(10,A1:A3,1)", 1},
		{"RANK(20,A1:A3,1)", 2},
		{"RANK(30,A1:A3,1)", 3},
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

func TestEval_RankTies(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 10.0},
		"A3": {T: "n", V: 8.0},
	}
	got := evalFormula(t, cells, "RANK(10,A1:A3)")
	if got != 1 {
		t.Errorf("RANK(10,A1:A3) = %v, want 1", got)
	}
	got = evalFormula(t, cells, "RANK(8,A1:A3)")
	if got != 3 {
		t.Errorf("RANK(8,A1:A3) = %v, want 3", got)
	}
}

func TestEval_RankEq(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
	}
	got := evalFormula(t, cells, "RANK.EQ(20,A1:A3)")
	if got != 2 {
		t.Errorf("RANK.EQ(20,A1:A3) = %v, want 2", got)
	}
}

func TestEval_RankWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "RANK()")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_RankEmptyRef(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "RANK(1,Z1:Z999)")
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("expected not found error, got %q", errMsg)
	}
}

func TestEval_RankNotFound(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
	}
	errMsg := evalFormulaErr(t, cells, "RANK(99,A1:A2)")
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("expected not found error, got %q", errMsg)
	}
}

func TestEval_Large(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
		"A2": {T: "n", V: 3.0},
		"A3": {T: "n", V: 7.0},
		"A4": {T: "n", V: 1.0},
		"A5": {T: "n", V: 9.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"LARGE(A1:A5,1)", 9},
		{"LARGE(A1:A5,2)", 7},
		{"LARGE(A1:A5,3)", 5},
		{"LARGE(A1:A5,4)", 3},
		{"LARGE(A1:A5,5)", 1},
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

func TestEval_LargeErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "LARGE(1)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "LARGE(1,2,3)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_LargeEmptySet(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "LARGE(Z1:Z999,1)")
	if !strings.Contains(errMsg, "empty") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
}

func TestEval_LargeKOutOfRange(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "LARGE(1,0)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "LARGE(1,2)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
}

func TestEval_Small(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 5.0},
		"A2": {T: "n", V: 3.0},
		"A3": {T: "n", V: 7.0},
		"A4": {T: "n", V: 1.0},
		"A5": {T: "n", V: 9.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"SMALL(A1:A5,1)", 1},
		{"SMALL(A1:A5,2)", 3},
		{"SMALL(A1:A5,3)", 5},
		{"SMALL(A1:A5,4)", 7},
		{"SMALL(A1:A5,5)", 9},
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

func TestEval_SmallWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SMALL(1)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "SMALL(1,2,3)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_SmallEmptySet(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SMALL(Z1:Z999,1)")
	if !strings.Contains(errMsg, "empty") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
}

func TestEval_SmallKOutOfRange(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SMALL(1,0)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "SMALL(1,2)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
}
