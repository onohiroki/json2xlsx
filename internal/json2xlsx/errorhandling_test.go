package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_IferrorNoError(t *testing.T) {
	got := evalFormula(t, nil, "IFERROR(100,999)")
	if got != 100 {
		t.Errorf("IFERROR(100,999) = %v, want 100", got)
	}
}

func TestEval_IferrorCatchesError(t *testing.T) {
	got := evalFormula(t, nil, "IFERROR(1/0,999)")
	if got != 999 {
		t.Errorf("IFERROR(1/0,999) = %v, want 999", got)
	}
}

func TestEval_IferrorCatchesSqrtNegative(t *testing.T) {
	got := evalFormula(t, nil, "IFERROR(SQRT(-1),-1)")
	if got != -1 {
		t.Errorf("IFERROR(SQRT(-1),-1) = %v, want -1", got)
	}
}

func TestEval_IferrorNested(t *testing.T) {
	got := evalFormula(t, nil, "IFERROR(IFERROR(1/0,200),300)")
	if got != 200 {
		t.Errorf("IFERROR(IFERROR(1/0,200),300) = %v, want 200", got)
	}
}

func TestEval_IferrorErrorInErrorArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "IFERROR(1/0,2/0)")
	if !strings.Contains(errMsg, "division by zero") {
		t.Errorf("expected division by zero error, got %q", errMsg)
	}
}

func TestEval_IferrorWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "IFERROR(1)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected exactly 2 error, got %q", errMsg)
	}
}

func TestEval_IferrorWithCellRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 42.0},
	}
	got := evalFormula(t, cells, "IFERROR(A1,999)")
	if got != 42 {
		t.Errorf("IFERROR(A1,999) = %v, want 42", got)
	}

	// A2 doesn't exist → getCellValue returns 0, nil → no error
	got = evalFormula(t, cells, "IFERROR(A2,999)")
	if got != 0 {
		t.Errorf("IFERROR(A2,999) = %v, want 0", got)
	}
}
