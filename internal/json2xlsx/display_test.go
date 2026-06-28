package json2xlsx

import "testing"

func TestCellDisplayValue_FormulaModeFormula(t *testing.T) {
	var w bool
	cases := []struct {
		cell Cell
		want string
	}{
		{Cell{T: "f", F: "1+2", V: float64(3)}, "=1+2"},
		{Cell{T: "f", F: "A1+B1"}, "=A1+B1"},
		{Cell{T: "n", V: float64(42)}, "42"},
		{Cell{T: "s", V: "hello"}, "hello"},
		{Cell{T: "b", V: true}, "true"},
		{Cell{T: "b", V: false}, "false"},
		{Cell{V: nil}, ""},
	}
	for _, c := range cases {
		got := CellDisplayValue(c.cell, MarkdownModeFormula, &w)
		if got != c.want {
			t.Errorf("CellDisplayValue(%+v, formula) = %q, want %q", c.cell, got, c.want)
		}
	}
}

func TestCellDisplayValue_FormulaModeValue(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "1+2", V: float64(3)}, MarkdownModeValue, &w)
	if got != "3" {
		t.Fatalf("expected value 3, got %q", got)
	}
	if w {
		t.Fatal("expected no warning when value present")
	}
}

func TestCellDisplayValue_FormulaModeValue_Fallback(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "A1+B1"}, MarkdownModeValue, &w)
	if got != "=A1+B1" {
		t.Fatalf("expected formula fallback, got %q", got)
	}
	if !w {
		t.Fatal("expected warning when formula without value in mode=v")
	}
}

func TestCellDisplayValue_FormulaModeBoth(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "1+2", V: float64(3)}, MarkdownModeBoth, &w)
	if got != "3<br />=1+2" {
		t.Fatalf("expected both value+formula, got %q", got)
	}
	if w {
		t.Fatal("expected no warning when both present")
	}
}

func TestCellDisplayValue_FormulaModeBoth_Fallback(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", F: "A1+B1"}, MarkdownModeBoth, &w)
	if got != "=A1+B1" {
		t.Fatalf("expected formula fallback, got %q", got)
	}
	if !w {
		t.Fatal("expected warning when formula without value in mode=both")
	}
}

func TestCellDisplayValue_FormulaModeBoth_ValueOnly(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "f", V: float64(42)}, MarkdownModeBoth, &w)
	if got != "42" {
		t.Fatalf("expected value 42, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_DefaultWithFormula(t *testing.T) {
	t.Run("mode_value_warning", func(t *testing.T) {
		var w bool
		got := CellDisplayValue(Cell{F: "SUM(A1:A10)"}, MarkdownModeValue, &w)
		if got != "=SUM(A1:A10)" {
			t.Fatalf("expected formula fallback, got %q", got)
		}
		if !w {
			t.Fatal("expected warning for default type with formula in mode=v")
		}
	})
	t.Run("mode_formula_no_warning", func(t *testing.T) {
		var w bool
		got := CellDisplayValue(Cell{F: "SUM(A1:A10)"}, MarkdownModeFormula, &w)
		if got != "=SUM(A1:A10)" {
			t.Fatalf("expected formula, got %q", got)
		}
		if w {
			t.Fatal("expected no warning in mode=f")
		}
	})
}

func TestCellDisplayValue_DateCell(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "d", V: float64(45678)}, MarkdownModeFormula, &w)
	if got != "2025-01-21T00:00:00" {
		t.Fatalf("expected RFC3339 date, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_DateCellString(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "d", V: "2025-01-21"}, MarkdownModeFormula, &w)
	if got != "2025-01-21" {
		t.Fatalf("expected string passthrough, got %q", got)
	}
}

func TestCellDisplayValue_TimeOnly(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "d", V: float64(0.04623843), Z: "h:mm:ss"}, MarkdownModeFormula, &w)
	if got != "1:06:35" {
		t.Fatalf("expected time 1:06:35, got %q", got)
	}
}

func TestCellDisplayValue_NoValueNoFormula(t *testing.T) {
	var w bool
	got := CellDisplayValue(Cell{T: "s"}, MarkdownModeFormula, &w)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if w {
		t.Fatal("expected no warning")
	}
}

func TestCellDisplayValue_HasWarningPreserved(t *testing.T) {
	var w bool
	CellDisplayValue(Cell{T: "f", F: "X"}, MarkdownModeValue, &w)
	if !w {
		t.Fatal("expected warning after first call")
	}

	w = false
	CellDisplayValue(Cell{T: "s", V: "ok"}, MarkdownModeValue, &w)
	if w {
		t.Fatal("expected no warning for normal cell")
	}
}
