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
