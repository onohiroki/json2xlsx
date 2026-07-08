package json2xlsx

import (
	"strings"
	"testing"
)

func TestEval_If(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(1,10,20)", 10},
		{"IF(0,10,20)", 20},
		{"IF(5,10,20)", 10},
		{"IF(-1,10,20)", 10},
		{"IF(5-5,10,20)", 20},
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

func TestEval_IfWithComparison(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 15.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(A1>10,100,200)", 100},
		{"IF(A1>20,100,200)", 200},
		{"IF(A1=15,1,0)", 1},
		{"IF(A1<>15,1,0)", 0},
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

func TestEval_IfShortCircuit(t *testing.T) {
	got := evalFormula(t, nil, "IF(0,1/0,999)")
	if got != 999 {
		t.Errorf("IF(0,1/0,999) = %v, want 999", got)
	}
	got = evalFormula(t, nil, "IF(1,999,1/0)")
	if got != 999 {
		t.Errorf("IF(1,999,1/0) = %v, want 999", got)
	}
}

func TestEval_IfNested(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 85.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(A1>=90,1,IF(A1>=80,2,3))", 2},
		{"IF(A1>=90,1,IF(A1>=80,2,IF(A1>=60,3,4)))", 2},
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

func TestEval_IfWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "IF(1)")
	if !strings.Contains(errMsg, "requires exactly 3 arguments") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_And(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"AND(1,1)", 1},
		{"AND(1,0)", 0},
		{"AND(0,1)", 0},
		{"AND(0,0)", 0},
		{"AND(1,1,1)", 1},
		{"AND(1,2,3)", 1},
		{"AND(1,2,0)", 0},
		{"AND()", 1},
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

func TestEval_AndShortCircuit(t *testing.T) {
	got := evalFormula(t, nil, "AND(0,1/0)")
	if got != 0 {
		t.Errorf("AND(0,1/0) = %v, want 0", got)
	}
}

func TestEval_Or(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"OR(0,1)", 1},
		{"OR(0,0)", 0},
		{"OR(1,0)", 1},
		{"OR(1,1)", 1},
		{"OR(0,0,0)", 0},
		{"OR(0,0,1)", 1},
		{"OR()", 0},
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

func TestEval_OrShortCircuit(t *testing.T) {
	got := evalFormula(t, nil, "OR(1,1/0)")
	if got != 1 {
		t.Errorf("OR(1,1/0) = %v, want 1", got)
	}
}

func TestEval_Not(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"NOT(0)", 1},
		{"NOT(1)", 0},
		{"NOT(100)", 0},
		{"NOT(-1)", 0},
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

func TestEval_NotWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "NOT(1,2)")
	if !strings.Contains(errMsg, "requires exactly 1 argument") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_CombinedLogical(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 15.0},
		"A2": {T: "n", V: 5.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IF(AND(A1>10,A2>0),100,200)", 100},
		{"IF(AND(A1>10,A2>10),100,200)", 200},
		{"IF(OR(A1>10,A2>10),100,200)", 100},
		{"IF(OR(A1>20,A2>20),100,200)", 200},
		{"IF(NOT(A1>20),100,200)", 100},
		{"IF(AND(A1>10,OR(A2>0,1=0)),1,0)", 1},
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

func TestEval_Ifs(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 85.0},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"IFS(A1>=90,1,A1>=80,2,A1>=60,3)", 2},
		{"IFS(A1>=90,1,A1>=70,2,A1>=60,3)", 2},
		{"IFS(A1=85,100,A1=86,200)", 100},
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

func TestEval_IfsNoMatch(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 50.0},
	}
	errMsg := evalFormulaErr(t, cells, "IFS(A1>=90,1,A1>=80,2)")
	if !strings.Contains(errMsg, "#N/A") {
		t.Errorf("expected #N/A error, got %q", errMsg)
	}
}

func TestEval_IfsWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "IFS(1)")
	if !strings.Contains(errMsg, "even number") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Switch(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"SWITCH(1,1,10,2,20)", 10},
		{"SWITCH(2,1,10,2,20)", 20},
		{"SWITCH(3,1,10,2,20,99)", 99},
		{"SWITCH(1+1,1,10,2,20)", 20},
		{"SWITCH(5,5,100)", 100},
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

func TestEval_SwitchNoMatchNoDefault(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SWITCH(3,1,10,2,20)")
	if !strings.Contains(errMsg, "#N/A") {
		t.Errorf("expected #N/A error, got %q", errMsg)
	}
}

func TestEval_SwitchWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, "SWITCH(1)")
	if !strings.Contains(errMsg, "at least 3") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_IfWithNestedFunc(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 10.0},
		"A2": {T: "n", V: 20.0},
		"A3": {T: "n", V: 30.0},
		"B1": {T: "n", V: 5.0},
	}
	got := evalFormula(t, cells, "IF(SUM(A1:A3)>50,SUM(A1:A3)*2,0)")
	if got != 120 {
		t.Errorf("got %v, want 120", got)
	}

	got = evalFormula(t, cells, "IF(SUM(A1:A3)<50,SUM(A1:A3)*2,999)")
	if got != 999 {
		t.Errorf("got %v, want 999", got)
	}
}
