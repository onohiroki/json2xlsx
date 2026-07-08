package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_VarS(t *testing.T) {
	// {2,4,4,4,5,5,7,9}: mean=5, sumSqDiff=32, VAR.S=32/7≈4.5714
	got := evalFormula(t, nil, "VAR.S(2,4,4,4,5,5,7,9)")
	want := 32.0 / 7.0
	if got != want {
		t.Errorf("VAR.S = %v, want %v", got, want)
	}
}

func TestEval_Var(t *testing.T) {
	got := evalFormula(t, nil, "VAR(2,4,4,4,5,5,7,9)")
	want := 32.0 / 7.0
	if got != want {
		t.Errorf("VAR = %v, want %v", got, want)
	}
}

func TestEval_VarP(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0},
		"A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0},
	}
	got := evalFormula(t, cells, "VAR.P(A1:A3)")
	// population variance = ((1-2)^2 + (2-2)^2 + (3-2)^2)/3 = (1+0+1)/3 = 2/3
	if got != 2.0/3.0 {
		t.Errorf("VAR.P(A1:A3) = %v, want %v", got, 2.0/3.0)
	}
}

func TestEval_VarSLiteral(t *testing.T) {
	got := evalFormula(t, nil, "VAR.S(2,4,4,4,5,5,7,9)")
	want := 32.0 / 7.0
	if got != want {
		t.Errorf("VAR.S = %v, want %v", got, want)
	}
}

func TestEval_VarPTooFew(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "VAR.P()")
	if !strings.Contains(errMsg, "empty set") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
}

func TestEval_VarSEmptySet(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "VAR.S(Z1:Z999)")
	if !strings.Contains(errMsg, "empty") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
}

func TestEval_VarSTooFew(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "VAR.S(1)")
	if !strings.Contains(errMsg, "at least 2") {
		t.Errorf("expected at least 2 error, got %q", errMsg)
	}
}

func TestEval_VarPOneValueWorks(t *testing.T) {
	got := evalFormula(t, nil, "VAR.P(5)")
	if got != 0 {
		t.Errorf("VAR.P(5) = %v, want 0", got)
	}
}

func TestEval_QuartileInc(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0}, "A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0}, "A4": {T: "n", V: 4.0},
		"A5": {T: "n", V: 5.0}, "A6": {T: "n", V: 6.0},
		"A7": {T: "n", V: 7.0}, "A8": {T: "n", V: 8.0},
		"A9": {T: "n", V: 9.0}, "A10": {T: "n", V: 10.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"QUARTILE.INC(A1:A10,0)", 1},
		{"QUARTILE.INC(A1:A10,1)", 3.25},
		{"QUARTILE.INC(A1:A10,2)", 5.5},
		{"QUARTILE.INC(A1:A10,3)", 7.75},
		{"QUARTILE.INC(A1:A10,4)", 10},
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

func TestEval_QuartileErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "QUARTILE.INC(1)")
	if !strings.Contains(errMsg, "at least 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "QUARTILE.INC(1,5)")
	if !strings.Contains(errMsg, "must be 0-4") {
		t.Errorf("expected quart range error, got %q", errMsg)
	}
}

func TestEval_PercentileInc(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"PERCENTILE.INC(1,2,3,4,5,0)", 1},
		{"PERCENTILE.INC(1,2,3,4,5,0.25)", 2},
		{"PERCENTILE.INC(1,2,3,4,5,0.5)", 3},
		{"PERCENTILE.INC(1,2,3,4,5,1)", 5},
		{"PERCENTILE.INC(1,2,3,4,5,0.3)", 2.2},
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

func TestEval_PercentileErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "PERCENTILE.INC(1)")
	if !strings.Contains(errMsg, "at least 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "PERCENTILE.INC(1,2,-0.1)")
	if !strings.Contains(errMsg, "between 0 and 1") {
		t.Errorf("expected k range error, got %q", errMsg)
	}
}

func TestEval_QuartileAlias(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0}, "A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0}, "A4": {T: "n", V: 4.0},
	}
	got := evalFormula(t, cells, "QUARTILE(A1:A4,2)")
	want := evalFormula(t, cells, "QUARTILE.INC(A1:A4,2)")
	if got != want {
		t.Errorf("QUARTILE = %v, QUARTILE.INC = %v, want equal", got, want)
	}
}

func TestEval_PercentileAlias(t *testing.T) {
	got := evalFormula(t, nil, "PERCENTILE(1,2,3,4,5,0.5)")
	want := evalFormula(t, nil, "PERCENTILE.INC(1,2,3,4,5,0.5)")
	if got != want {
		t.Errorf("PERCENTILE = %v, PERCENTILE.INC = %v, want equal", got, want)
	}
}

func TestEval_Geomean(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"GEOMEAN(1,2,4)", 2},
		{"GEOMEAN(2,8)", 4},
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

func TestEval_GeomeanErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "GEOMEAN()")
	if !strings.Contains(errMsg, "empty set") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "GEOMEAN(-1,2)")
	if !strings.Contains(errMsg, "positive") {
		t.Errorf("expected positive error, got %q", errMsg)
	}
}

func TestEval_Harmean(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"HARMEAN(1,2,4)", 12.0 / 7.0},
		{"HARMEAN(2,8)", 3.2},
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

func TestEval_HarmeanErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "HARMEAN()")
	if !strings.Contains(errMsg, "empty set") {
		t.Errorf("expected empty set error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "HARMEAN(0,2)")
	if !strings.Contains(errMsg, "positive") {
		t.Errorf("expected positive error, got %q", errMsg)
	}
}

func TestEval_Trimmean(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0}, "A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0}, "A4": {T: "n", V: 4.0},
		"A5": {T: "n", V: 5.0}, "A6": {T: "n", V: 6.0},
		"A7": {T: "n", V: 7.0}, "A8": {T: "n", V: 8.0},
		"A9": {T: "n", V: 9.0}, "A10": {T: "n", V: 10.0},
	}
	got := evalFormula(t, cells, "TRIMMEAN(A1:A10,0.2)")
	if got != 5.5 {
		t.Errorf("TRIMMEAN = %v, want 5.5", got)
	}
}

func TestEval_TrimmeanLiteral(t *testing.T) {
	got := evalFormula(t, nil, "TRIMMEAN(1,2,3,4,5,0)")
	if got != 3 {
		t.Errorf("TRIMMEAN = %v, want 3", got)
	}
}

func TestEval_TrimmeanErrors(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "TRIMMEAN(1)")
	if !strings.Contains(errMsg, "at least 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
	errMsg = evalFormulaErr(t, nil, "TRIMMEAN(1,2,-0.1)")
	if !strings.Contains(errMsg, "must be between") {
		t.Errorf("expected percent range error, got %q", errMsg)
	}
}

func TestEval_Mode(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"MODE(1,2,2,3)", 2},
		{"MODE.SNGL(1,2,2,3,3)", 2},
		{"MODE(1,1,2,2,3,3,3)", 3},
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

func TestEval_ModeNoMode(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "MODE(1,2,3)")
	if !strings.Contains(errMsg, "#N/A") {
		t.Errorf("expected #N/A error, got %q", errMsg)
	}
}

func TestEval_Subtotal(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 1.0}, "A2": {T: "n", V: 2.0},
		"A3": {T: "n", V: 3.0}, "A4": {T: "n", V: 4.0},
		"B1": {T: "n", V: 10.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"SUBTOTAL(1,A1:A4)", 2.5},
		{"SUBTOTAL(2,A1:A4)", 4},
		{"SUBTOTAL(4,A1:A4)", 4},
		{"SUBTOTAL(5,A1:A4)", 1},
		{"SUBTOTAL(9,A1:A4)", 10},
		{"SUBTOTAL(9,A1:A4,B1)", 20},
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
