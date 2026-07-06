package json2xlsx

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestTimeToExcelSerial(t *testing.T) {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	got := timeToExcelSerial(epoch)
	if got != 0 {
		t.Errorf("timeToExcelSerial(1899-12-30) = %v, want 0", got)
	}

	day1 := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	got = timeToExcelSerial(day1)
	if got != 2 {
		t.Errorf("timeToExcelSerial(1900-01-01) = %v, want 2", got)
	}
}

func TestEval_Today(t *testing.T) {
	got := evalFormula(t, nil, "TODAY()")
	if got == 0 {
		t.Errorf("TODAY() = 0, expected non-zero")
	}
	_, frac := math.Modf(got)
	if frac != 0 {
		t.Errorf("TODAY() = %v, expected integer (frac=%v)", got, frac)
	}
}

func TestEval_TodayWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "TODAY(1)")
	if !strings.Contains(errMsg, "no arguments") {
		t.Errorf("expected no arguments error, got %q", errMsg)
	}
}

func TestEval_Now(t *testing.T) {
	got := evalFormula(t, nil, "NOW()")
	if got == 0 {
		t.Errorf("NOW() = 0, expected non-zero")
	}
}

func TestEval_NowWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "NOW(1)")
	if !strings.Contains(errMsg, "no arguments") {
		t.Errorf("expected no arguments error, got %q", errMsg)
	}
}
