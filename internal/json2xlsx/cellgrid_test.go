package json2xlsx

import "testing"

func TestBuildCellGrid_Empty(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{})
	if ok {
		t.Fatal("expected false for empty sheet")
	}
}

func TestBuildCellGrid_EmptyCells(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{}})
	if ok {
		t.Fatal("expected false for sheet with empty cells map")
	}
}

func TestBuildCellGrid_SingleCell(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{"A1": {V: float64(42)}}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 1 || cg.MaxRow != 1 {
		t.Fatalf("expected 1x1, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.ColNames[1] != "A" {
		t.Fatalf("expected colName[1]=A, got %q", cg.ColNames[1])
	}
	if cg.Rows[1][1].V != float64(42) {
		t.Fatalf("expected cell A1=42, got %v", cg.Rows[1][1].V)
	}
}

func TestBuildCellGrid_MultipleCells(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1": {V: "x"},
		"C3": {V: "y"},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 3 || cg.MaxRow != 3 {
		t.Fatalf("expected 3x3, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.Rows[1][1].V != "x" {
		t.Fatalf("expected A1=x, got %v", cg.Rows[1][1].V)
	}
	if cg.Rows[3][3].V != "y" {
		t.Fatalf("expected C3=y, got %v", cg.Rows[3][3].V)
	}
}

func TestBuildCellGrid_SparseCells_EmptyIntermediate(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1": {V: "first"},
		"Z1": {V: "last"},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 26 || cg.MaxRow != 1 {
		t.Fatalf("expected 26x1, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.ColNames[26] != "Z" {
		t.Fatalf("expected colNames[26]=Z, got %q", cg.ColNames[26])
	}
	var emptyCell Cell
	if cg.Rows[1][2] != emptyCell {
		t.Fatal("expected intermediate cell B1 to be zero-value")
	}
}

func TestBuildCellGrid_RowsOnly(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{
		Rows: [][]interface{}{
			{"a", "b", "c"},
			{1, 2},
		},
	})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.MaxCol != 3 || cg.MaxRow != 2 {
		t.Fatalf("expected 3x2, got %dx%d", cg.MaxCol, cg.MaxRow)
	}
	if cg.Rows[1][1].V != "a" || cg.Rows[1][2].V != "b" || cg.Rows[1][3].V != "c" {
		t.Fatalf("first row mismatch: got %v %v %v", cg.Rows[1][1].V, cg.Rows[1][2].V, cg.Rows[1][3].V)
	}
	if cg.Rows[2][1].V != 1 || cg.Rows[2][2].V != 2 {
		t.Fatalf("second row mismatch: got %v %v", cg.Rows[2][1].V, cg.Rows[2][2].V)
	}
	var emptyCell Cell
	if cg.Rows[2][3] != emptyCell {
		t.Fatal("expected Row2[3] to be zero-value")
	}
}

func TestBuildCellGrid_InvalidAxis(t *testing.T) {
	_, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"B2":       {V: float64(1)},
		"INVALID":  {V: float64(2)},
		"":         {V: float64(3)},
		"1A":       {V: float64(4)},
	}})
	if !ok {
		t.Fatal("expected true despite invalid axes")
	}
}

func TestBuildCellGrid_ColNames(t *testing.T) {
	cg, ok := BuildCellGrid(Sheet{Cells: map[string]Cell{
		"A1":  {},
		"AA1": {},
	}})
	if !ok {
		t.Fatal("expected true")
	}
	if cg.ColNames[1] != "A" {
		t.Fatalf("col 1 expected A, got %q", cg.ColNames[1])
	}
	if cg.ColNames[27] != "AA" {
		t.Fatalf("col 27 expected AA, got %q", cg.ColNames[27])
	}
}
