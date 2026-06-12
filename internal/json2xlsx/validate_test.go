package json2xlsx

import (
	"os"
	"strings"
	"testing"
)

func TestValidateJSON_ValidSingleSheet(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "s", "v": "hello"},
			"B1": {"t": "n", "v": 100}
		}
	}`
	if err := ValidateJSON([]byte(data)); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestValidateJSON_ValidMultiSheet(t *testing.T) {
	data := `{
		"sheets": [
			{"name": "S1", "cells": {"A1": {"t": "s", "v": "a"}}},
			{"name": "S2", "cells": {"A1": {"t": "n", "v": 1}}}
		],
		"styles": [{"id": 1, "font": {"bold": true}}]
	}`
	if err := ValidateJSON([]byte(data)); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestValidateJSON_ValidWithAllCellTypes(t *testing.T) {
	data := `{
		"name": "Types",
		"cells": {
			"A1": {"t": "s", "v": "str"},
			"A2": {"t": "n", "v": 42},
			"A3": {"t": "b", "v": true},
			"A4": {"t": "f", "f": "SUM(A1:A3)"},
			"A5": {"t": "d", "v": "2026-05-24T00:00:00+09:00"}
		}
	}`
	if err := ValidateJSON([]byte(data)); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestValidateJSON_InvalidCellType(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "x", "v": "bad"}
		}
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), ".cells.A1.t") {
		t.Errorf("should mention path .cells.A1.t, got: %v", err)
	}
}

func TestValidateJSON_MissingRequiredField(t *testing.T) {
	data := `{"name": "Sheet1"}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cells") && !strings.Contains(err.Error(), "required") {
		t.Errorf("should mention missing 'cells', got: %v", err)
	}
}

func TestValidateJSON_UnknownAdditionalProperty(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {"A1": {"t": "s", "v": "x"}},
		"unknown_field": true
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("should mention unknown_field, got: %v", err)
	}
}

func TestValidateJSON_InvalidCellRef(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {
			"1A": {"t": "s", "v": "bad-ref"}
		}
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "1A") {
		t.Errorf("should mention '1A', got: %v", err)
	}
}

func TestValidateJSON_OneOfSimplified(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"sheets": [{"name": "S1", "cells": {"A1": {"t": "s", "v": "a"}}}]
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "single sheet") || !strings.Contains(err.Error(), "sheets") {
		t.Errorf("should mention both formats, got: %v", err)
	}
}

func TestValidateJSON_InvalidHyperlink(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "s", "v": "link", "l": 123}
		}
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateJSON_JSONSyntaxError(t *testing.T) {
	data := `{invalid json}`
	if err := ValidateJSON([]byte(data)); err != nil {
		t.Error("expected nil for JSON syntax errors, got:", err)
	}
}

func TestValidateJSON_MultipleErrors(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {
			"A1": {"t": "x", "v": 1},
			"B1": {"t": "y", "v": 2}
		}
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), ".cells.A1.t") && !strings.Contains(err.Error(), ".cells.B1.t") {
		t.Errorf("should mention cell-level errors, got: %v", err)
	}
}

func TestValidate_ValidStyleWithAllBorderTypes(t *testing.T) {
	for _, style := range []string{"thin", "medium", "thick", "dashed", "dotted", "double", "hair", "mediumDashed", "dashDot", "mediumDashDot", "dashDotDot", "mediumDashDotDot", "slantDashDot"} {
		data := `{
			"name": "Sheet1",
			"cells": {"A1": {"t": "s", "v": "x"}},
			"styles": [{"id": 1, "border": [{"style": "` + style + `", "color": "#000000"}]}]
		}`
		if err := ValidateJSON([]byte(data)); err != nil {
			t.Errorf("border style %q should be valid, got: %v", style, err)
		}
	}
}

func TestValidate_ExampleFiles(t *testing.T) {
	for _, name := range []string{"sales.json", "styles.json", "merge.json", "time_diff.json", "time_test.json"} {
		data, err := os.ReadFile("../../samples/" + name)
		if err != nil {
			t.Skipf("skip %s: %v", name, err)
		}
		if err := ValidateJSON(data); err != nil {
			t.Errorf("example %s should be valid, got: %v", name, err)
		}
	}
}

func TestValidate_EmptyCells(t *testing.T) {
	data := `{"name":"S","cells":{}}`
	if err := ValidateJSON([]byte(data)); err != nil {
		t.Errorf("empty cells should be valid, got: %v", err)
	}
}

func TestValidate_FillMissingColor(t *testing.T) {
	data := `{
		"name": "Sheet1",
		"cells": {"A1": {"t": "s", "v": "x"}},
		"styles": [{"id": 1, "fill": {"type": "pattern", "pattern": 1, "color": []}}]
	}`
	err := ValidateJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidate_InstPathToJQ(t *testing.T) {
	tests := []struct {
		in  []string
		out string
	}{
		{nil, ".input"},
		{[]string{}, ".input"},
		{[]string{"name"}, ".name"},
		{[]string{"sheets", "0", "name"}, ".sheets[0].name"},
		{[]string{"cells", "A1", "t"}, ".cells.A1.t"},
		{[]string{"styles", "2", "border", "0", "style"}, ".styles[2].border[0].style"},
	}
	for _, tt := range tests {
		got := instPathToJQ(tt.in)
		if got != tt.out {
			t.Errorf("instPathToJQ(%v) = %q, want %q", tt.in, got, tt.out)
		}
	}
}

func TestValidate_IsAllDigits(t *testing.T) {
	if !isAllDigits("0") {
		t.Error("expected true for '0'")
	}
	if !isAllDigits("123") {
		t.Error("expected true for '123'")
	}
	if isAllDigits("A1") {
		t.Error("expected false for 'A1'")
	}
	if isAllDigits("") {
		t.Error("expected false for ''")
	}
}
