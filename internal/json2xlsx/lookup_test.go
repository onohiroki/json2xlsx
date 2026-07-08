package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_Choose(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"CHOOSE(1,10)", 10},
		{"CHOOSE(1,10,20,30)", 10},
		{"CHOOSE(2,10,20,30)", 20},
		{"CHOOSE(3,10,20,30)", 30},
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

func TestEval_ChooseErrors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"CHOOSE()", "at least 2 arguments"},
		{"CHOOSE(1)", "at least 2 arguments"},
		{"CHOOSE(0,10,20)", "out of range"},
		{"CHOOSE(3,10,20)", "out of range"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, nil, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}

func TestEval_Vlookup(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
		"B3": {T: "n", V: 300.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"VLOOKUP(10,A1:B3,2)", 100},
		{"VLOOKUP(20,A1:B3,2)", 200},
		{"VLOOKUP(30,A1:B3,2)", 300},
		{"VLOOKUP(10,A1:B3,1)", 10},
		{"VLOOKUP(20,A1:B3,1)", 20},
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

func TestEval_VlookupWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "VLOOKUP(10,A1:B2)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_VlookupErrors(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
	}
	tests := []struct {
		formula string
		contain string
	}{
		{"VLOOKUP(99,A1:B2,2)", "not found"},
		{"VLOOKUP(10,A1:B2,3)", "column index out of range"},
		{"VLOOKUP(10,A1:B2,0)", "column index out of range"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, cells, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}

func TestEval_Match(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"A4": {T: "n", V: 20.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"MATCH(10,A1:A4,0)", 1},
		{"MATCH(20,A1:A4,0)", 2},
		{"MATCH(30,A1:A4,0)", 3},
		{"MATCH(20,A1:A3,0)", 2},
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

func TestEval_MatchWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "MATCH(10)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_MatchErrors(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
	}
	tests := []struct {
		formula string
		contain string
	}{
		{"MATCH(99,A1:A2,0)", "not found"},
		{"MATCH(10,A1:A2,1)", "not yet supported"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, cells, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}

func TestEval_Index(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
		"B3": {T: "n", V: 300.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"INDEX(A1:A3,1)", 10},
		{"INDEX(A1:A3,2)", 20},
		{"INDEX(A1:A3,3)", 30},
		{"INDEX(A1:B3,1,1)", 10},
		{"INDEX(A1:B3,2,2)", 200},
		{"INDEX(A1:B3,3,2)", 300},
		{"INDEX(A1:B3,1,2)", 100},
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

func TestEval_IndexWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "INDEX()")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "INDEX(A1:B3,1,2,3)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_IndexColumnOutOfRange(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"B1": {T: "n", V: 100.0},
	}
	errMsg := evalFormulaErr(t, cells, "INDEX(A1:B1,1,3)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, cells, "INDEX(A1:B1,1,0)")
	if !strings.Contains(errMsg, "out of range") {
		t.Errorf("expected out of range error, got %q", errMsg)
	}
}

func TestEval_IndexErrors(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
	}
	tests := []struct {
		formula string
		contain string
	}{
		{"INDEX(A1:A2,0)", "out of range"},
		{"INDEX(A1:A2,3)", "out of range"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, cells, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}

func TestEval_Xlookup(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
		"B3": {T: "n", V: 300.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"XLOOKUP(10,A1:A3,B1:B3)", 100},
		{"XLOOKUP(20,A1:A3,B1:B3)", 200},
		{"XLOOKUP(30,A1:A3,B1:B3)", 300},
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

func TestEval_XlookupIfNotFound(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"B1": {T: "n", V: 100.0},
		"B2": {T: "n", V: 200.0},
	}
	got := evalFormula(t, cells, "XLOOKUP(99,A1:A2,B1:B2,-1)")
	if got != -1 {
		t.Errorf("XLOOKUP with not found = %v, want -1", got)
	}
}

func TestEval_XlookupWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "XLOOKUP(10,A1:A2)")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_XlookupErrors(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"B1": {T: "n", V: 100.0},
	}
	tests := []struct {
		formula string
		contain string
	}{
		{"XLOOKUP(99,A1:A2,B1:B1)", "same size"},
		{"XLOOKUP(99,A1:A2,B1:B2)", "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, cells, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}

func TestEval_Hlookup(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"B1": {T: "n", V: 2.0},
		"C1": {T: "n", V: 3.0},
		"A2": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
		"C2": {T: "n", V: 30.0},
		"A3": {T: "n", V: 100.0},
		"B3": {T: "n", V: 200.0},
		"C3": {T: "n", V: 300.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"HLOOKUP(2,A1:C3,2)", 20},
		{"HLOOKUP(3,A1:C3,3)", 300},
		{"HLOOKUP(1,A1:C3,1)", 1},
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

func TestEval_HlookupErrors(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"B1": {T: "n", V: 2.0},
		"A2": {T: "n", V: 10.0},
		"B2": {T: "n", V: 20.0},
	}
	tests := []struct {
		formula string
		contain string
	}{
		{"HLOOKUP(5,A1:B2,2)", "not found"},
		{"HLOOKUP(1,A1:B2,3)", "index out of range"},
		{"HLOOKUP(1,A1:B2,0)", "index out of range"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errStr := evalFormulaErr(t, cells, tt.formula)
			if !strings.Contains(errStr, tt.contain) {
				t.Errorf("eval %q error = %q, want contain %q", tt.formula, errStr, tt.contain)
			}
		})
	}
}
