package json2xlsx

import (
	"math"
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

func TestEval_Trunc(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"TRUNC(3.14)", 3},
		{"TRUNC(-3.14)", -3},
		{"TRUNC(123.456,1)", 123.4},
		{"TRUNC(123.456,-1)", 120},
		{"TRUNC(5)", 5},
		{"TRUNC(-5)", -5},
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

func TestEval_TruncErrors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"TRUNC()", "requires 1 or 2"},
		{"TRUNC(1,2,3)", "requires 1 or 2"},
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

func TestEval_Sign(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"SIGN(5)", 1},
		{"SIGN(0)", 0},
		{"SIGN(-5)", -1},
		{"SIGN(3.14)", 1},
		{"SIGN(-3.14)", -1},
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

func TestEval_SignErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SIGN()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Pi(t *testing.T) {
	got := evalFormula(t, nil, "PI()")
	if got != math.Pi {
		t.Errorf("PI() = %v, want %v", got, math.Pi)
	}
}

func TestEval_PiErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "PI(1)")
	if !strings.Contains(errMsg, "requires no arguments") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Rand(t *testing.T) {
	got := evalFormula(t, nil, "RAND()")
	if got < 0 || got >= 1 {
		t.Errorf("RAND() = %v, want in [0,1)", got)
	}
}

func TestEval_RandErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "RAND(1)")
	if !strings.Contains(errMsg, "requires no arguments") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Sin(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"SIN(0)", 0},
		{"SIN(3.141592653589793/2)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_SinErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SIN()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Cos(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"COS(0)", 1},
		{"COS(3.141592653589793)", -1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_CosErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "COS(1,2)")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Tan(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"TAN(0)", 0},
		{"TAN(0.7853981633974483)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_TanErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "TAN()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Ln(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"LN(1)", 0},
		{"LN(2.718281828459045)", 1},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_LnErrors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"LN(0)", "#NUM!"},
		{"LN(-1)", "#NUM!"},
		{"LN()", "exactly 1"},
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

func TestEval_Log10(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"LOG10(1)", 0},
		{"LOG10(10)", 1},
		{"LOG10(100)", 2},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_Log10Errors(t *testing.T) {
	tests := []struct {
		formula string
		contain string
	}{
		{"LOG10(0)", "#NUM!"},
		{"LOG10(-1)", "#NUM!"},
		{"LOG10()", "exactly 1"},
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

func TestEval_Exp(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"EXP(0)", 1},
		{"EXP(1)", 2.718281828459045},
	}
	for _, tt := range tests {
		t.Run(tt.formula, func(t *testing.T) {
			got := evalFormula(t, nil, tt.formula)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("eval %q = %v, want %v", tt.formula, got, tt.want)
			}
		})
	}
}

func TestEval_ExpErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "EXP(1,2)")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}
