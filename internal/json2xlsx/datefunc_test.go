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

func TestExcelSerialToTime(t *testing.T) {
	got := excelSerialToTime(0)
	expected := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("excelSerialToTime(0) = %v, want %v", got, expected)
	}
	got = excelSerialToTime(43831)
	expected = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("excelSerialToTime(43831) = %v, want %v", got, expected)
	}
}

func TestEval_Year(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"YEAR(43831)", 2020},
		{"YEAR(43832)", 2020},
		{"YEAR(0)", 1899},
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

func TestEval_YearWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "YEAR()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Month(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"MONTH(43831)", 1},
		{"MONTH(43832)", 1},
		{"MONTH(43862)", 2},
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

func TestEval_MonthWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "MONTH()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Day(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"DAY(43831)", 1},
		{"DAY(43832)", 2},
		{"DAY(43862)", 1},
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

func TestEval_DayWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "DAY()")
	if !strings.Contains(errMsg, "exactly 1") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Weekday(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		// 43831 = 2020-01-01 (Wednesday)
		{"WEEKDAY(43831)", 4},     // return_type=1: Sun=1..Sat=7 → Wed=4
		{"WEEKDAY(43831,1)", 4},
		{"WEEKDAY(43831,2)", 3},   // return_type=2: Mon=1..Sun=7 → Wed=3
		{"WEEKDAY(43831,3)", 2},   // return_type=3: Mon=0..Sun=6 → Wed=2
		// 43832 = 2020-01-02 (Thursday)
		{"WEEKDAY(43832,1)", 5},
		{"WEEKDAY(43832,2)", 4},
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

func TestEval_WeekdayWrongArgs(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "WEEKDAY()")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_WeekdayUnsupportedReturnType(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "WEEKDAY(43831,99)")
	if !strings.Contains(errMsg, "unsupported return_type") {
		t.Errorf("expected unsupported return_type error, got %q", errMsg)
	}
}

func TestEval_Weeknum(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		// 43831 = 2020-01-01 → ISO week 1
		{"WEEKNUM(43831)", 1},
		{"WEEKNUM(43831,1)", 1},
		{"WEEKNUM(43831,2)", 1},
		// 43833 = 2020-01-03 (Friday) → still ISO week 1
		{"WEEKNUM(43833)", 1},
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

func TestEval_WeeknumWrongArgs(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "WEEKNUM()")
	if !strings.Contains(errMsg, "requires") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Date(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"DATE(2020,1,1)", 43831},
		{"DATE(2020,2,1)", 43862},
		{"DATE(2020,1,31)", 43861},
		{"DATE(2020,2,29)", 43890},
		{"DATE(1899,12,30)", 0},
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

func TestEval_DateWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "DATE(2020,1)")
	if !strings.Contains(errMsg, "exactly 3") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Edate(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"EDATE(43831,1)", 43862},
		{"EDATE(43831,-1)", 43800},
		{"EDATE(43831,0)", 43831},
		{"EDATE(43861,1)", 43890},
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

func TestEval_EdateWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "EDATE(43831)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Eomonth(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"EOMONTH(43831,0)", 43861},
		{"EOMONTH(43831,1)", 43890},
		{"EOMONTH(43831,-1)", 43830},
		{"EOMONTH(43862,1)", 43921},
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

func TestEval_EomonthWrongArg(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "EOMONTH(43831)")
	if !strings.Contains(errMsg, "exactly 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}
