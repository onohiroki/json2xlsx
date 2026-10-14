package json2xlsx

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvalFormulaNoWorkbook(t *testing.T) {
	cases := []struct {
		formula string
		want    any
	}{
		{"=1+2*3", float64(7)},
		{"SUM(1,2,3)", float64(6)},
		{"=SUM(1,2,3)", float64(6)}, // leading '=' optional
		{`="a"&"b"`, "ab"},
		{"=CONCAT(\"x\",\"y\")", "xy"},
	}
	for _, tc := range cases {
		res, err := EvalFormula(tc.formula, nil, EvalOptions{})
		if err != nil {
			t.Errorf("EvalFormula(%q) error: %v", tc.formula, err)
			continue
		}
		if res.Code != "" || res.Error != "" {
			t.Errorf("EvalFormula(%q) unexpected error result: %+v", tc.formula, res)
			continue
		}
		if res.Value != tc.want {
			t.Errorf("EvalFormula(%q) = %v (%T), want %v (%T)", tc.formula, res.Value, res.Value, tc.want, tc.want)
		}
	}
}

func TestEvalFormulaWithWorkbook(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{"name": "S1", "cells": {
		"A1": {"t": "n", "v": 10},
		"A2": {"t": "n", "v": 20}
	}}`)

	res, err := EvalFormula("=SUM(A1:A2)", wb, EvalOptions{})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(30) {
		t.Errorf("value = %v, want 30", res.Value)
	}

	res, err = EvalFormula("=A1*2", wb, EvalOptions{})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(20) {
		t.Errorf("value = %v, want 20", res.Value)
	}
}

func TestEvalFormulaSheetSelection(t *testing.T) {
	wb := mustUnmarshalWorkbook(t, `{"sheets": [
		{"name": "One", "cells": {"A1": {"t": "n", "v": 1}}},
		{"name": "Two", "cells": {"A1": {"t": "n", "v": 2}}}
	]}`)

	res, err := EvalFormula("=A1", wb, EvalOptions{Sheet: "Two"})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(2) {
		t.Errorf("value = %v, want 2", res.Value)
	}

	// 先頭シートは既定．
	res, err = EvalFormula("=A1", wb, EvalOptions{})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Value != float64(1) {
		t.Errorf("value = %v, want 1", res.Value)
	}

	if _, err := EvalFormula("=A1", wb, EvalOptions{Sheet: "Missing"}); err == nil {
		t.Error("expected error for missing sheet")
	}
}

func TestEvalFormulaErrors(t *testing.T) {
	res, err := EvalFormula("=1/0", nil, EvalOptions{})
	if err != nil {
		t.Fatalf("EvalFormula returned err for a formula error: %v", err)
	}
	if res.Code != "#DIV/0!" {
		t.Errorf("code = %q, want #DIV/0!", res.Code)
	}
	if res.Error == "" {
		t.Error("expected error message to be preserved")
	}

	res, err = EvalFormula("=MOD(5,0)", nil, EvalOptions{})
	if err != nil {
		t.Fatalf("EvalFormula: %v", err)
	}
	if res.Code != "#DIV/0!" {
		t.Errorf("code = %q, want #DIV/0!", res.Code)
	}
}

func TestEvalFormulaEmpty(t *testing.T) {
	if _, err := EvalFormula("", nil, EvalOptions{}); err == nil {
		t.Error("expected error for empty formula")
	}
	if _, err := EvalFormula("=", nil, EvalOptions{}); err == nil {
		t.Error("expected error for '=' only")
	}
}

func TestEvalResultMarshalJSON(t *testing.T) {
	b, err := json.Marshal(EvalResult{Value: float64(3)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"value":3`) {
		t.Errorf("value JSON = %s, want value:3", b)
	}

	b, err = json.Marshal(EvalResult{Code: "#DIV/0!", Error: "division by zero"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"code":"#DIV/0!"`) || !strings.Contains(s, `"message":"division by zero"`) {
		t.Errorf("error JSON = %s, want code and message", s)
	}
}
