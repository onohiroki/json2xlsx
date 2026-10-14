package json2xlsx

import (
	"strings"
	"testing"
)

func TestParseCellRangeValid(t *testing.T) {
	cases := []struct {
		in                     string
		minC, minR, maxC, maxR int
	}{
		{"A1:F20", 1, 1, 6, 20},
		{"A1", 1, 1, 1, 1},
		{"b2", 2, 2, 2, 2},
		{"B2:C3", 2, 2, 3, 3},
		{"F20:A1", 1, 1, 6, 20}, // 逆順は正規化
		{"  C3:B2 ", 2, 2, 3, 3},
	}
	for _, tc := range cases {
		got, err := ParseCellRange(tc.in)
		if err != nil {
			t.Errorf("ParseCellRange(%q) error: %v", tc.in, err)
			continue
		}
		if got.MinCol != tc.minC || got.MinRow != tc.minR || got.MaxCol != tc.maxC || got.MaxRow != tc.maxR {
			t.Errorf("ParseCellRange(%q) = %+v, want %d,%d,%d,%d", tc.in, got, tc.minC, tc.minR, tc.maxC, tc.maxR)
		}
	}
}

func TestParseCellRangeInvalid(t *testing.T) {
	for _, in := range []string{"", "A", "1", "A1:", ":B2", "A1:B2:C3", "AB:CD"} {
		if _, err := ParseCellRange(in); err == nil {
			t.Errorf("ParseCellRange(%q) expected error", in)
		}
	}
}

func TestBuildCellGridRangeNilMatchesFull(t *testing.T) {
	sh := Sheet{Cells: map[string]Cell{
		"A1": {V: "a"},
		"C3": {V: float64(3)},
	}}
	full, ok := BuildCellGrid(sh)
	if !ok {
		t.Fatal("BuildCellGrid returned not ok")
	}
	got, ok, err := BuildCellGridRange(sh, nil)
	if err != nil || !ok {
		t.Fatalf("BuildCellGridRange(nil) ok=%v err=%v", ok, err)
	}
	if got.MaxRow != full.MaxRow || got.MaxCol != full.MaxCol || got.RowOffset != 0 || got.ColOffset != 0 {
		t.Errorf("range(nil) = %+v, want same shape as full %+v", got, full)
	}
}

func TestBuildCellGridRangeCrop(t *testing.T) {
	sh := Sheet{Cells: map[string]Cell{
		"A1": {V: "h1"}, "B1": {V: "h2"}, "C1": {V: "h3"},
		"A2": {V: float64(1)}, "B2": {V: float64(2)}, "C2": {V: float64(3)},
		"A3": {V: float64(4)}, "B3": {V: float64(5)}, "C3": {V: float64(6)},
	}}

	rng, err := ParseCellRange("B2:C3")
	if err != nil {
		t.Fatalf("ParseCellRange: %v", err)
	}
	cg, ok, err := BuildCellGridRange(sh, &rng)
	if err != nil || !ok {
		t.Fatalf("BuildCellGridRange ok=%v err=%v", ok, err)
	}
	if cg.MaxRow != 2 || cg.MaxCol != 2 {
		t.Fatalf("shape = %dx%d, want 2x2", cg.MaxRow, cg.MaxCol)
	}
	if cg.RowOffset != 1 || cg.ColOffset != 1 {
		t.Errorf("offset = (%d,%d), want (1,1)", cg.RowOffset, cg.ColOffset)
	}
	if cg.ColNames[1] != "B" || cg.ColNames[2] != "C" {
		t.Errorf("colNames = %v, want B,C", cg.ColNames)
	}
	if cg.Rows[1][1].V != float64(2) {
		t.Errorf("B2 = %v, want 2", cg.Rows[1][1].V)
	}
	if cg.Rows[2][2].V != float64(6) {
		t.Errorf("C3 = %v, want 6", cg.Rows[2][2].V)
	}
}

func TestBuildCellGridRangeClamp(t *testing.T) {
	sh := Sheet{Cells: map[string]Cell{
		"A1": {V: float64(1)},
		"B2": {V: float64(2)},
	}}
	rng, _ := ParseCellRange("A1:Z100")
	cg, ok, err := BuildCellGridRange(sh, &rng)
	if err != nil || !ok {
		t.Fatalf("BuildCellGridRange ok=%v err=%v", ok, err)
	}
	if cg.MaxRow != 2 || cg.MaxCol != 2 {
		t.Errorf("shape = %dx%d, want 2x2 (clamped)", cg.MaxRow, cg.MaxCol)
	}
}

func TestBuildCellGridRangeOutOfRange(t *testing.T) {
	sh := Sheet{Cells: map[string]Cell{"A1": {V: float64(1)}}}
	rng, _ := ParseCellRange("Z9:Z10")
	if _, _, err := BuildCellGridRange(sh, &rng); err == nil {
		t.Error("expected error for range outside data")
	}
}

func TestToMarkdownRangeAbsoluteLabels(t *testing.T) {
	sh := Sheet{Name: "S", Cells: map[string]Cell{
		"A1": {T: "s", V: "h1"}, "B1": {T: "s", V: "h2"}, "C1": {T: "s", V: "h3"},
		"A2": {T: "n", V: float64(1)}, "B2": {T: "n", V: float64(2)}, "C2": {T: "n", V: float64(3)},
	}}
	wb := Workbook{Sheets: []Sheet{sh}}
	out, hasWarning, err := renderMarkdownRange(wb, MarkdownOptions{Mode: MarkdownModeValue, RowIndex: true}, mustRange(t, "B1:C2"))
	if err != nil {
		t.Fatalf("renderMarkdownRange: %v", err)
	}
	if hasWarning {
		t.Error("unexpected warning")
	}
	if !containsAll(out, "| B", "| C", "|   1", "|   2") {
		t.Errorf("expected absolute column/row labels in output:\n%s", out)
	}
	if strings.Contains(out, "| A") {
		t.Errorf("column A should not appear in B1:C2 output:\n%s", out)
	}
}

func TestToMarkdownRangeOutOfRangeError(t *testing.T) {
	wb := Workbook{Sheets: []Sheet{{Name: "S", Cells: map[string]Cell{"A1": {T: "s", V: "x"}}}}}
	_, _, err := renderMarkdownRange(wb, MarkdownOptions{Mode: MarkdownModeValue}, mustRange(t, "Z9:Z10"))
	if err == nil {
		t.Error("expected error for out-of-range")
	}
}

func mustRange(t *testing.T, s string) *CellRange {
	t.Helper()
	r, err := ParseCellRange(s)
	if err != nil {
		t.Fatalf("ParseCellRange(%q): %v", s, err)
	}
	return &r
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
