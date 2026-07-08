package json2xlsx

import (
	"testing"
)

func TestEval_Isnumber(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 42.0},
		"A2": {T: "s", V: "hello"},
		"A3": {T: "b", V: true},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISNUMBER(A1)", 1},
		{"ISNUMBER(A2)", 0},
		{"ISNUMBER(A3)", 0},
		{"ISNUMBER(99)", 1},
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

func TestEval_Isblank(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "n", V: 42.0},
		"A2": {T: "s", V: ""},
		"A4": {T: "s", V: "x"},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISBLANK(A1)", 0},
		{"ISBLANK(A2)", 0},
		{"ISBLANK(A3)", 1},
		{"ISBLANK(42)", 0},
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

func TestEval_Istext(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "hello"},
		"A2": {T: "n", V: 42.0},
		"A3": {T: "b", V: true},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISTEXT(A1)", 1},
		{"ISTEXT(A2)", 0},
		{"ISTEXT(A3)", 0},
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

func TestEval_Isnontext(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "hello"},
		"A2": {T: "n", V: 42.0},
		"A3": {T: "b", V: true},
	}
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISNONTEXT(A1)", 0},
		{"ISNONTEXT(A2)", 1},
		{"ISNONTEXT(A3)", 1},
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

func TestEval_Iserror(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISERROR(1/0)", 1},
		{"ISERROR(42)", 0},
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

func TestEval_Isna(t *testing.T) {
	tests := []struct {
		formula string
		want    float64
	}{
		{"ISNA(1/0)", 0},
		{"ISNA(42)", 0},
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

func TestEval_IsnaCatchesModeError(t *testing.T) {
	got := evalFormula(t, nil, "ISNA(MODE(1,2,3))")
	if got != 1 {
		t.Errorf("ISNA(MODE(1,2,3)) = %v, want 1", got)
	}
}
