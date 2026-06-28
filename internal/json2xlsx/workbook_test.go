package json2xlsx

import "testing"

func TestForEachCell_SingleSheet(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{{
			Cells: map[string]Cell{
				"A1": {T: "n", V: float64(1)},
				"B1": {T: "s", V: "x"},
			},
		}},
	}
	var visited []string
	forEachCell(wb, func(axis string, cell Cell) Cell {
		visited = append(visited, axis)
		return cell
	})
	if len(visited) != 2 {
		t.Fatalf("expected 2 cells visited, got %d", len(visited))
	}
}

func TestForEachCell_MultiSheet(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Cells: map[string]Cell{"A1": {T: "n", V: float64(1)}}},
			{Cells: map[string]Cell{"B1": {T: "s", V: "y"}}},
		},
	}
	var visited []string
	forEachCell(wb, func(axis string, cell Cell) Cell {
		visited = append(visited, axis)
		return cell
	})
	if len(visited) != 2 {
		t.Fatalf("expected 2 cells visited, got %d", len(visited))
	}
}

func TestForEachCell_BookWrapper(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Name: "S1", Cells: map[string]Cell{"A1": {T: "n", V: float64(1)}}},
			{Name: "S2", Cells: map[string]Cell{"B1": {T: "s", V: "y"}}},
		},
	}
	var visited []string
	forEachCell(wb, func(axis string, cell Cell) Cell {
		visited = append(visited, axis)
		return cell
	})
	if len(visited) != 2 {
		t.Fatalf("expected 2 cells visited, got %d", len(visited))
	}
}

func TestForEachCell_Mutation(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Cells: map[string]Cell{"A1": {T: "n", V: float64(1)}}},
			{Cells: map[string]Cell{"B1": {T: "s", V: "x"}}},
			{Name: "S1", Cells: map[string]Cell{"C1": {T: "b", V: false}}},
		},
	}
	forEachCell(wb, func(axis string, cell Cell) Cell {
		cell.T = "s"
		cell.V = "mutated"
		return cell
	})
	if wb.Sheets[0].Cells["A1"].V != "mutated" {
		t.Errorf("sheet0 A1.V = %v, want mutated", wb.Sheets[0].Cells["A1"].V)
	}
	if wb.Sheets[1].Cells["B1"].V != "mutated" {
		t.Errorf("sheet1 B1.V = %v, want mutated", wb.Sheets[1].Cells["B1"].V)
	}
	if wb.Sheets[2].Cells["C1"].V != "mutated" {
		t.Errorf("sheet2 C1.V = %v, want mutated", wb.Sheets[2].Cells["C1"].V)
	}
}

func TestForEachCell_NilMaps(t *testing.T) {
	wb := &Workbook{}
	// nil Cells, nil Sheets, nil Book でも panic しないことを確認
	var count int
	forEachCell(wb, func(axis string, cell Cell) Cell {
		count++
		return cell
	})
	if count != 0 {
		t.Fatalf("expected 0 calls, got %d", count)
	}
}

func TestForEachCell_MixedFormats(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Cells: map[string]Cell{"A1": {T: "n", V: float64(1)}}},
			{Cells: map[string]Cell{"B1": {T: "s", V: "x"}}},
			{Name: "S1", Cells: map[string]Cell{"C1": {T: "b", V: false}}},
		},
	}
	var visited []string
	forEachCell(wb, func(axis string, cell Cell) Cell {
		visited = append(visited, axis)
		return cell
	})
	if len(visited) != 3 {
		t.Fatalf("expected 3 cells visited, got %d: %v", len(visited), visited)
	}
}

func TestFlattenWorkbook_BookWrapper(t *testing.T) {
	wb := &Workbook{
		Book: &Book{
			Sheets: map[string]Sheet{
				"S1": {Cells: map[string]Cell{"A1": {T: "s", V: "x"}}},
				"S2": {Cells: map[string]Cell{"A1": {T: "n", V: 1}}},
			},
			Styles: []Style{{ID: 1, Font: &Font{Bold: true}}},
		},
	}
	sheets, styles := flattenWorkbook(wb)
	if len(sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(sheets))
	}
	// map のイテレーション順は非決定論的．両方の名前が存在すれば OK．
	names := map[string]bool{}
	for _, sh := range sheets {
		names[sh.Name] = true
	}
	if !names["S1"] || !names["S2"] {
		t.Errorf("expected sheets S1 and S2, got %v", sheets)
	}
	if len(styles) != 1 || styles[0].ID != 1 {
		t.Errorf("expected Book.Styles to override, got %+v", styles)
	}
}

func TestFlattenWorkbook_BookWrapper_FallbackStyles(t *testing.T) {
	wb := &Workbook{
		Styles: []Style{{ID: 99, Font: &Font{Italic: true}}},
		Book: &Book{
			Sheets: map[string]Sheet{"X": {Cells: map[string]Cell{"A1": {T: "s", V: "a"}}}},
		},
	}
	_, styles := flattenWorkbook(wb)
	if len(styles) != 1 || styles[0].ID != 99 {
		t.Errorf("expected top-level Styles as fallback, got %+v", styles)
	}
}

func TestFlattenWorkbook_MultiSheet(t *testing.T) {
	wb := &Workbook{
		Styles: []Style{{ID: 10}},
		Sheets: []Sheet{
			{Name: "A", Cells: map[string]Cell{"A1": {T: "s", V: "a"}}},
			{Name: "B", Cells: map[string]Cell{"A1": {T: "n", V: 2}}},
		},
	}
	sheets, styles := flattenWorkbook(wb)
	if len(sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(sheets))
	}
	if sheets[0].Name != "A" || sheets[1].Name != "B" {
		t.Errorf("unexpected sheet order: %q, %q", sheets[0].Name, sheets[1].Name)
	}
	if len(styles) != 1 || styles[0].ID != 10 {
		t.Errorf("expected wb.Styles, got %+v", styles)
	}
}

func TestFlattenWorkbook_SingleSheet(t *testing.T) {
	wb := &Workbook{
		Name:    "MySheet",
		Cells:   map[string]Cell{"B2": {T: "n", V: 42}},
		Cols:    []ColInfo{{Col: "A", Width: 10}},
		RowDims: []RowInfo{{Row: 1, Height: 20}},
		Merges:  []Merge{{Range: "A1:B2"}},
		Styles:  []Style{{ID: 2}},
	}
	sheets, styles := flattenWorkbook(wb)
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	if sheets[0].Name != "MySheet" {
		t.Errorf("expected Name=MySheet, got %q", sheets[0].Name)
	}
	if sheets[0].Cells["B2"].V != 42 {
		t.Errorf("unexpected cell value: %+v", sheets[0].Cells["B2"])
	}
	if len(sheets[0].Cols) != 1 || sheets[0].Cols[0].Col != "A" {
		t.Errorf("cols not carried over: %+v", sheets[0].Cols)
	}
	if len(sheets[0].RowDims) != 1 || sheets[0].RowDims[0].Row != 1 {
		t.Errorf("rowDims not carried over: %+v", sheets[0].RowDims)
	}
	if len(sheets[0].Merges) != 1 || sheets[0].Merges[0].Range != "A1:B2" {
		t.Errorf("merges not carried over: %+v", sheets[0].Merges)
	}
	if len(styles) != 1 || styles[0].ID != 2 {
		t.Errorf("expected wb.Styles, got %+v", styles)
	}
}

func TestFlattenWorkbook_SingleSheet_RowsOnly(t *testing.T) {
	wb := &Workbook{
		Rows: [][]interface{}{{"a", 1}, {"b", 2}},
	}
	sheets, _ := flattenWorkbook(wb)
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	if len(sheets[0].Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(sheets[0].Rows))
	}
}

func TestFlattenWorkbook_Empty(t *testing.T) {
	wb := &Workbook{}
	sheets, styles := flattenWorkbook(wb)
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet (empty default), got %d", len(sheets))
	}
	if sheets[0].Cells != nil || sheets[0].Rows != nil {
		t.Errorf("expected empty sheet, got %+v", sheets[0])
	}
	if styles != nil {
		t.Errorf("expected nil styles, got %+v", styles)
	}
}

func TestFlattenWorkbook_BookWrapper_PreservesCellData(t *testing.T) {
	wb := &Workbook{
		Book: &Book{
			Sheets: map[string]Sheet{
				"Data": {Cells: map[string]Cell{
					"A1": {T: "s", V: "hello"},
					"B1": {T: "n", V: 3.14},
				}},
			},
		},
	}
	sheets, _ := flattenWorkbook(wb)
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	if sheets[0].Cells["A1"].V != "hello" {
		t.Errorf("A1 value = %v, want hello", sheets[0].Cells["A1"].V)
	}
	if sheets[0].Cells["B1"].V != 3.14 {
		t.Errorf("B1 value = %v, want 3.14", sheets[0].Cells["B1"].V)
	}
}

func TestFlattenWorkbook_Priority_BookOverSheets(t *testing.T) {
	wb := &Workbook{
		Book: &Book{
			Sheets: map[string]Sheet{"BS": {Cells: map[string]Cell{"A1": {T: "s", V: "book"}}}},
		},
		Sheets: []Sheet{{Name: "AS", Cells: map[string]Cell{"A1": {T: "s", V: "arr"}}}},
		Cells:  map[string]Cell{"A1": {T: "s", V: "single"}},
	}
	sheets, _ := flattenWorkbook(wb)
	if len(sheets) != 1 || sheets[0].Name != "BS" {
		t.Errorf("expected Book wrapper to take priority, got %+v", sheets)
	}
}

func TestNormalizeWorkbook_BookWrapper(t *testing.T) {
	wb := &Workbook{
		Book: &Book{
			Sheets: map[string]Sheet{
				"S1": {Cells: map[string]Cell{"A1": {T: "s", V: "x"}}},
				"S2": {Cells: map[string]Cell{"A1": {T: "n", V: 1}}},
			},
			Styles: []Style{{ID: 1, Font: &Font{Bold: true}}},
		},
	}
	normalizeWorkbook(wb)
	if len(wb.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(wb.Sheets))
	}
	names := map[string]bool{}
	for _, sh := range wb.Sheets {
		names[sh.Name] = true
	}
	if !names["S1"] || !names["S2"] {
		t.Errorf("expected sheets S1 and S2, got %v", wb.Sheets)
	}
	if len(wb.Styles) != 1 || wb.Styles[0].ID != 1 {
		t.Errorf("expected Book.Styles, got %+v", wb.Styles)
	}
	// Book フィールドは Charts 参照のために保持される
	if wb.Book == nil {
		t.Error("Book field should be preserved after normalizeWorkbook")
	}
}

func TestNormalizeWorkbook_SingleSheet(t *testing.T) {
	wb := &Workbook{
		Name:    "MySheet",
		Cells:   map[string]Cell{"B2": {T: "n", V: 42}},
		Cols:    []ColInfo{{Col: "A", Width: 10}},
		Merges:  []Merge{{Range: "A1:B2"}},
		Styles:  []Style{{ID: 2}},
	}
	normalizeWorkbook(wb)
	if len(wb.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(wb.Sheets))
	}
	if wb.Sheets[0].Name != "MySheet" {
		t.Errorf("expected Name=MySheet, got %q", wb.Sheets[0].Name)
	}
	if wb.Sheets[0].Cells["B2"].V != 42 {
		t.Errorf("unexpected cell value: %+v", wb.Sheets[0].Cells["B2"])
	}
	if len(wb.Sheets[0].Cols) != 1 {
		t.Errorf("cols not carried over: %+v", wb.Sheets[0].Cols)
	}
	if len(wb.Styles) != 1 || wb.Styles[0].ID != 2 {
		t.Errorf("expected wb.Styles, got %+v", wb.Styles)
	}
}

func TestNormalizeWorkbook_RowsOnly(t *testing.T) {
	wb := &Workbook{
		Rows: [][]interface{}{{"a", 1}, {"b", 2}},
	}
	normalizeWorkbook(wb)
	if len(wb.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(wb.Sheets))
	}
	if len(wb.Sheets[0].Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(wb.Sheets[0].Rows))
	}
}

func TestNormalizeWorkbook_AlreadyNormalized(t *testing.T) {
	wb := &Workbook{
		Sheets: []Sheet{
			{Name: "A", Cells: map[string]Cell{"A1": {T: "s", V: "a"}}},
			{Name: "B", Cells: map[string]Cell{"A1": {T: "n", V: 2}}},
		},
		Styles: []Style{{ID: 10}},
	}
	normalizeWorkbook(wb)
	if len(wb.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(wb.Sheets))
	}
	if wb.Sheets[0].Name != "A" || wb.Sheets[1].Name != "B" {
		t.Errorf("unexpected sheet order: %q, %q", wb.Sheets[0].Name, wb.Sheets[1].Name)
	}
}

func TestNormalizeWorkbook_Empty(t *testing.T) {
	wb := &Workbook{}
	normalizeWorkbook(wb)
	if len(wb.Sheets) != 1 {
		t.Fatalf("expected 1 sheet (empty default), got %d", len(wb.Sheets))
	}
	if wb.Sheets[0].Cells != nil || wb.Sheets[0].Rows != nil {
		t.Errorf("expected empty sheet, got %+v", wb.Sheets[0])
	}
	if wb.Styles != nil {
		t.Errorf("expected nil styles, got %+v", wb.Styles)
	}
}

func TestNormalizeWorkbook_BookWrapper_StylesPrecedence(t *testing.T) {
	wb := &Workbook{
		Styles: []Style{{ID: 99, Font: &Font{Italic: true}}},
		Book: &Book{
			Sheets: map[string]Sheet{"X": {Cells: map[string]Cell{"A1": {T: "s", V: "a"}}}},
		},
	}
	normalizeWorkbook(wb)
	if len(wb.Styles) != 1 || wb.Styles[0].ID != 99 {
		t.Errorf("expected top-level Styles as fallback, got %+v", wb.Styles)
	}
}
