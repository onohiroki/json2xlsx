package json2xlsx

import (
	"testing"
)

func TestNormalizeNewlines(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello\r\nworld", "hello\nworld"},
		{"hello\rworld", "hello\nworld"},
		{"hello\nworld", "hello\nworld"},
		{"hello\r\n\r\nworld", "hello\n\nworld"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeNewlines(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeNewlines(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestValidateMode(t *testing.T) {
	if err := ValidateMode(MarkdownModeFormula); err != nil {
		t.Errorf("expected nil for formula mode, got %v", err)
	}
	if err := ValidateMode(MarkdownModeValue); err != nil {
		t.Errorf("expected nil for value mode, got %v", err)
	}
	if err := ValidateMode(MarkdownModeBoth); err != nil {
		t.Errorf("expected nil for both mode, got %v", err)
	}
	if err := ValidateMode(MarkdownMode("")); err == nil {
		t.Errorf("expected error for empty mode")
	}
	if err := ValidateMode(MarkdownMode("xxx")); err == nil {
		t.Errorf("expected error for unknown mode")
	}
}

func TestTrimBOM(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte{0xEF, 0xBB, 0xBF, 'a', 'b', 'c'}, []byte{'a', 'b', 'c'}},
		{[]byte{'a', 'b', 'c'}, []byte{'a', 'b', 'c'}},
		{[]byte{0xEF, 0xBB}, []byte{0xEF, 0xBB}},
		{nil, nil},
	}
	for _, tt := range tests {
		got := trimBOM(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("trimBOM(%v) = %v; want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("trimBOM(%v) = %v; want %v", tt.input, got, tt.expected)
				break
			}
		}
	}
}

func TestCheckJSONArrayWarning_NoWarningForXLSX(t *testing.T) {
	res := &ReadWorkbookResult{IsXLSX: true}
	w := checkJSONArrayWarning(res, true, MarkdownModeFormula)
	if len(w) != 0 {
		t.Errorf("expected no warnings for XLSX input, got %v", w)
	}
}

func TestCheckJSONArrayWarning_NoWarningForDefaultMode(t *testing.T) {
	res := &ReadWorkbookResult{RawData: []byte(`[{"A1":1}]`)}
	w := checkJSONArrayWarning(res, false, MarkdownModeFormula)
	if len(w) != 0 {
		t.Errorf("expected no warnings when mode is not explicitly set, got %v", w)
	}
}

func TestCheckJSONArrayWarning_WarningForFormulaMode(t *testing.T) {
	res := &ReadWorkbookResult{RawData: []byte(`[{"A1":1}]`)}
	w := checkJSONArrayWarning(res, true, MarkdownModeFormula)
	if len(w) != 1 {
		t.Fatalf("expected 1 warning for formula mode with JSON array, got %v", w)
	}
	if w[0] != "Warning: --mode=f is ignored for JSON array input (formulas not supported in this format)." {
		t.Errorf("unexpected warning message: %q", w[0])
	}
}

func TestCheckJSONArrayWarning_WarningForBothMode(t *testing.T) {
	res := &ReadWorkbookResult{RawData: []byte(`[{"A1":1}]`)}
	w := checkJSONArrayWarning(res, true, MarkdownModeBoth)
	if len(w) != 1 {
		t.Fatalf("expected 1 warning for both mode with JSON array, got %v", w)
	}
}

func TestCheckJSONArrayWarning_NoWarningForValueMode(t *testing.T) {
	res := &ReadWorkbookResult{RawData: []byte(`[{"A1":1}]`)}
	w := checkJSONArrayWarning(res, true, MarkdownModeValue)
	if len(w) != 0 {
		t.Errorf("expected no warnings for value mode, got %v", w)
	}
}
