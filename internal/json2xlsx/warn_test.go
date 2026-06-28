package json2xlsx

import "testing"

func TestWarnMissingFormulaValue_Both(t *testing.T) {
	got := warnMissingFormulaValue(MarkdownModeBoth)
	want := "Warning: Missing values for some cells; showing only formulas."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWarnMissingFormulaValue_Others(t *testing.T) {
	for _, mode := range []MarkdownMode{MarkdownModeFormula, MarkdownModeValue} {
		got := warnMissingFormulaValue(mode)
		want := "Warning: Missing values for some cells; showing formulas instead."
		if got != want {
			t.Fatalf("warnMissingFormulaValue(%q) = %q, want %q", mode, got, want)
		}
	}
}

func TestWarnFormulaOnlyCSV(t *testing.T) {
	got := warnFormulaOnlyCSV()
	want := "Warning: Some cells have formulas but no values; treating them as empty."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
