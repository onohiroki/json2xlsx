package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_Floor(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"FLOOR(3.7,2)", 2},
		{"FLOOR(-5.4,1)", -6},
		{"FLOOR(5.4,1)", 5},
		{"FLOOR(5.4,2)", 4},
		{"FLOOR(-5.4,-1)", -5},
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

func TestEval_FloorErrors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"FLOOR(5,0)", "#DIV/0"},
		{"FLOOR(5,-1)", "#NUM"},
		{"FLOOR(1)", "exactly 2"},
		{"FLOOR(1,2,3)", "exactly 2"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errMsg := evalFormulaErr(t, nil, tt.formula)
			if !strings.Contains(errMsg, tt.contain) {
				t.Errorf("eval %q error = %q, want containing %q", tt.formula, errMsg, tt.contain)
			}
		})
	}
}

func TestEval_Ceiling(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"CEILING(3.7,2)", 4},
		{"CEILING(-2.5,2)", -2},
		{"CEILING(-2.5,-2)", -4},
		{"CEILING(5.4,1)", 6},
		{"CEILING(5.4,2)", 6},
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

func TestEval_CeilingErrors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"CEILING(5,0)", "#DIV/0"},
		{"CEILING(5,-1)", "#NUM"},
		{"CEILING(1)", "exactly 2"},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			errMsg := evalFormulaErr(t, nil, tt.formula)
			if !strings.Contains(errMsg, tt.contain) {
				t.Errorf("eval %q error = %q, want containing %q", tt.formula, errMsg, tt.contain)
			}
		})
	}
}

func TestEval_Mod(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"MOD(10,3)", 1},
		{"MOD(10,2)", 0},
		{"MOD(-3,2)", 1},
		{"MOD(3,-2)", -1},
		{"MOD(-3,-2)", -1},
		{"MOD(5.5,2)", 1.5},
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

func TestEval_ModDivideByZero(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "MOD(5,0)")
	if !strings.Contains(errMsg, "#DIV/0") {
		t.Errorf("expected #DIV/0 error, got %q", errMsg)
	}
}

func TestEval_ModWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "MOD(1)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Power(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"POWER(2,3)", 8},
		{"POWER(9,0.5)", 3},
		{"POWER(0,0)", 1},
		{"POWER(2,-1)", 0.5},
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

func TestEval_PowerWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "POWER(2)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Sqrt(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"SQRT(9)", 3},
		{"SQRT(0)", 0},
		{"SQRT(2.25)", 1.5},
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

func TestEval_SqrtNegative(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SQRT(-1)")
	if !strings.Contains(errMsg, "#NUM") {
		t.Errorf("expected #NUM error, got %q", errMsg)
	}
}

func TestEval_Int(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"INT(3.7)", 3},
		{"INT(-5.4)", -6},
		{"INT(0)", 0},
		{"INT(5)", 5},
		{"INT(-1)", -1},
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

func TestEval_IntWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "INT(1,2)")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Counta(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "s", V: "hello"},
		"A4": {T: "n", V: 30.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"COUNTA(A1:A4)", 3},
		{"COUNTA(A1)", 1},
		{"COUNTA(A3)", 0},
		{"COUNTA(42)", 1},
		{"COUNTA()", 0},
		{"COUNTA(A1,42)", 2},
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
