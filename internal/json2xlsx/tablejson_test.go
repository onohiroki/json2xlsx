package json2xlsx

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func convertTableJSON(t *testing.T, input string) *excelize.File {
	t.Helper()
	var buf bytes.Buffer
	err := Convert(strings.NewReader(input), &buf, ConvertOptions{TableJSON: true})
	if err != nil {
		t.Fatalf("Convert(table): %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestTableJSONBasic(t *testing.T) {
	const input = `{
	  "tables": [{
	    "name": "Sales",
	    "header": ["商品", "数量", "単価", "金額"],
	    "rows": [
	      ["A", 10, 100, "=B2*C2"],
	      ["B", 5, 200, "=B3*C3"]
	    ],
	    "columns": [
	      {"width": 18},
	      {"format": "#,##0", "align": "right"},
	      {"format": "#,##0", "align": "right"},
	      {"format": "#,##0", "align": "right"}
	    ],
	    "styles": {
	      "header": {
	        "font": {"bold": true},
	        "fill": {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]}
	      }
	    },
	    "headerStyle": "header",
	    "freezeHeader": true,
	    "autoFilter": true
	  }]
	}`

	f := convertTableJSON(t, input)
	if got := f.GetSheetName(0); got != "Sales" {
		t.Fatalf("sheet name = %q, want Sales", got)
	}
	if got, _ := f.GetCellValue("Sales", "A1"); got != "商品" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellValue("Sales", "B2"); got != "10" {
		t.Fatalf("B2 = %q", got)
	}
	if got, _ := f.GetCellFormula("Sales", "D2"); got != "B2*C2" {
		t.Fatalf("D2 formula = %q", got)
	}
	if got, _ := f.GetCellStyle("Sales", "A1"); got == 0 {
		t.Fatalf("A1 has no style")
	}
	if got, _ := f.GetColWidth("Sales", "A"); got <= 0 {
		t.Fatalf("column A width = %v", got)
	}
}

func TestTableJSONFormulaTemplate(t *testing.T) {
	const input = `{
	  "tables": [{
	    "header": ["qty", "price", "amount"],
	    "rows": [[2, 10], [3, 20]],
	    "columns": [
	      {},
	      {},
	      {"formula": "=A{r}*B{r}"}
	    ]
	  }]
	}`

	f := convertTableJSON(t, input)
	if got, _ := f.GetCellFormula("Sheet1", "C2"); got != "A2*B2" {
		t.Fatalf("C2 formula = %q, want A2*B2", got)
	}
	if got, _ := f.GetCellFormula("Sheet1", "C3"); got != "A3*B3" {
		t.Fatalf("C3 formula = %q, want A3*B3", got)
	}
}

func TestTableJSONCellObject(t *testing.T) {
	const input = `{
	  "tables": [{
	    "rows": [
	      [{"v": "literal = text"}, {"v": 5, "format": "0.00"}, {"v": 1, "f": "2+2"}]
	    ]
	  }]
	}`

	f := convertTableJSON(t, input)
	if got, _ := f.GetCellValue("Sheet1", "A1"); got != "literal = text" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "B1"); got != "5.00" {
		t.Fatalf("B1 = %q", got)
	}
	if got, _ := f.GetCellFormula("Sheet1", "C1"); got != "2+2" {
		t.Fatalf("C1 formula = %q", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "C1"); got != "1" {
		t.Fatalf("C1 cached value = %q", got)
	}
}

func TestTableJSONValidation(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "missing style",
			in:   `{"tables":[{"rows":[[1]],"headerStyle":"missing"}]}`,
			want: `.input.tables[0].headerStyle`,
		},
		{
			name: "freeze without header",
			in:   `{"tables":[{"rows":[[1]],"freezeHeader":true}]}`,
			want: `.input.tables[0].freezeHeader`,
		},
		{
			name: "bad merge",
			in:   `{"tables":[{"rows":[[1]],"merges":["A1"]}]}`,
			want: `.input.tables[0].merges[0]`,
		},
		{
			name: "unknown cell field",
			in:   `{"tables":[{"rows":[[{"value":1}]]}]}`,
			want: `.input.tables[0].rows[0][0]`,
		},
		{
			name: "duplicate sheet",
			in:   `{"tables":[{"name":"S","rows":[[1]]},{"name":"s","rows":[[2]]}]}`,
			want: `.input.tables[1].name`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := ValidateTableJSON([]byte(tc.in))
			if err != nil {
				t.Fatalf("ValidateTableJSON: %v", err)
			}
			if len(issues) == 0 {
				t.Fatalf("expected issues")
			}
			found := false
			for _, iss := range issues {
				if strings.Contains(iss.Path, tc.want) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("issues = %+v, want path containing %q", issues, tc.want)
			}
		})
	}
}

func TestTableJSONDataJSONConflict(t *testing.T) {
	err := Convert(strings.NewReader(`{"tables":[]}`), &bytes.Buffer{}, ConvertOptions{TableJSON: true, DataJSON: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTableJSONStylePrecedence(t *testing.T) {
	const input = `{
	  "tables": [{
	    "rows": [[1, 2]],
	    "styles": {
	      "body": {"font": {"bold": true}},
	      "col": {"font": {"italic": true}}
	    },
	    "bodyStyle": "body",
	    "columns": [{}, {"style": "col", "align": "right"}]
	  }]
	}`

	f := convertTableJSON(t, input)
	a, _ := f.GetCellStyle("Sheet1", "A1")
	b, _ := f.GetCellStyle("Sheet1", "B1")
	if a == 0 || b == 0 {
		t.Fatalf("styles not applied: A=%d B=%d", a, b)
	}
	if a == b {
		t.Fatalf("column style should override body style")
	}
}

func TestTableJSONFormulaTemplateCompute(t *testing.T) {
	const input = `{
	  "tables": [{
	    "header": ["qty", "price", "amount"],
	    "rows": [[2, 10], [3, 20]],
	    "columns": [{}, {}, {"formula": "=A{r}*B{r}"}]
	  }]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(input), &buf, ConvertOptions{TableJSON: true, EvalFormulas: true}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, _ := f.GetCellValue("Sheet1", "C2"); got != "20" {
		t.Fatalf("C2 = %q, want 20", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "C3"); got != "60" {
		t.Fatalf("C3 = %q, want 60", got)
	}
}

func TestTableJSONToMarkdown(t *testing.T) {
	const input = `{"tables":[{"header":["a","b"],"rows":[[1,2]]}]}`
	var out bytes.Buffer
	opts := MarkdownOptions{TableJSON: true, FirstRowHeader: true, RowIndex: false, Mode: MarkdownModeValue, Compact: true}
	if err := ToMarkdown(strings.NewReader(input), &out, opts); err != nil {
		t.Fatalf("ToMarkdown: %v", err)
	}
	if !strings.Contains(out.String(), "| a | b |") {
		t.Fatalf("markdown = %q", out.String())
	}
}

func TestTableJSONToCSV(t *testing.T) {
	const input = `{"tables":[{"header":["a","b"],"rows":[[1,2]]}]}`
	var out bytes.Buffer
	if err := ToCSVWithFormat(strings.NewReader(input), &out, "", 0, false, true, false, ""); err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "a,b\n1,2" {
		t.Fatalf("csv = %q", got)
	}
}

func TestTableJSONStyleValidation(t *testing.T) {
	invalid := `{
	  "tables": [{
	    "rows": [[1]],
	    "headerStyle": "header",
	    "styles": {
	      "header": {
	        "fill": {"type": "solid", "pattern": 1, "color": ["FFE0EBF5"]},
	        "border": [{"style": "thin", "color": "FF000000"}]
	      }
	    }
	  }]
	}`
	issues, err := ValidateTableJSON([]byte(invalid))
	if err != nil {
		t.Fatalf("ValidateTableJSON: %v", err)
	}
	if len(issues) < 3 {
		t.Fatalf("issues = %+v, want fill.type, fill.color, border.color issues", issues)
	}

	valid := `{
	  "tables": [{
	    "rows": [[1]],
	    "headerStyle": "header",
	    "styles": {
	      "header": {
	        "fill": {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]},
	        "border": [{"style": "thin", "color": "#000000"}]
	      }
	    }
	  }]
	}`
	issues, err = ValidateTableJSON([]byte(valid))
	if err != nil {
		t.Fatalf("ValidateTableJSON valid: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %+v, want none", issues)
	}
}

func TestTableJSONObjectRowsJapaneseKeys(t *testing.T) {
	const input = `{
	  "tables": [{
	    "columns": [
	      {"key": "商品", "header": "商品名", "width": 18},
	      {"key": "数量", "header": "数量", "format": "#,##0"},
	      {"key": "単価", "header": "単価", "format": "#,##0"},
	      {"key": "金額", "header": "金額", "format": "#,##0", "formula": "={数量}*{単価}"}
	    ],
	    "rows": [
	      {"商品": "A", "数量": 10, "単価": 100},
	      {"商品": "B", "数量": 5, "単価": 200}
	    ]
	  }]
	}`

	f := convertTableJSON(t, input)
	if got, _ := f.GetCellValue("Sheet1", "A1"); got != "商品名" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellFormula("Sheet1", "D2"); got != "B2*C2" {
		t.Fatalf("D2 = %q, want B2*C2", got)
	}
	if got, _ := f.GetCellFormula("Sheet1", "D3"); got != "B3*C3" {
		t.Fatalf("D3 = %q, want B3*C3", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "A2"); got != "A" {
		t.Fatalf("A2 = %q", got)
	}
}

func TestTableJSONObjectRowsColumnRangeFormula(t *testing.T) {
	const input = `{
	  "tables": [{
	    "columns": [
	      {"key": "item"},
	      {"key": "amount"}
	    ],
	    "rows": [
	      {"item": "total", "amount": "=SUM({amount:col}2:{amount:col}3)"}
	    ]
	  }]
	}`
	f := convertTableJSON(t, input)
	if got, _ := f.GetCellFormula("Sheet1", "B1"); got != "SUM(B2:B3)" {
		t.Fatalf("B1 = %q, want SUM(B2:B3)", got)
	}
}

func TestTableJSONArrayRowsWithKeys(t *testing.T) {
	const input = `{
	  "tables": [{
	    "columns": [
	      {"key": "qty"},
	      {"key": "price"},
	      {"key": "amount", "formula": "={qty}*{price}"}
	    ],
	    "rows": [[2, 10]]
	  }]
	}`
	f := convertTableJSON(t, input)
	if got, _ := f.GetCellFormula("Sheet1", "C1"); got != "A1*B1" {
		t.Fatalf("C1 = %q, want A1*B1", got)
	}
}

func TestTableJSONObjectRowsCellObject(t *testing.T) {
	const input = `{
	  "tables": [{
	    "columns": [{"key": "a"}, {"key": "b"}],
	    "rows": [{"a": {"v": 5, "format": "0.00"}, "b": {"v": 10, "f": "={a}*2"}}]
	  }]
	}`
	f := convertTableJSON(t, input)
	if got, _ := f.GetCellFormula("Sheet1", "B1"); got != "A1*2" {
		t.Fatalf("B1 = %q, want A1*2", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "A1"); got != "5.00" {
		t.Fatalf("A1 = %q", got)
	}
}

func TestTableJSONObjectRowsValidation(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "unknown row key",
			in:   `{"tables":[{"columns":[{"key":"a"}],"rows":[{"a":1,"foo":2}]}]}`,
			want: `.rows[0].foo`,
		},
		{
			name: "duplicate key",
			in:   `{"tables":[{"columns":[{"key":"a"},{"key":"a"}],"rows":[{"a":1}]}]}`,
			want: `.columns[1].key`,
		},
		{
			name: "mixed rows",
			in:   `{"tables":[{"columns":[{"key":"a"}],"rows":[[1],{"a":2}]}]}`,
			want: `.rows[1]`,
		},
		{
			name: "missing key for object rows",
			in:   `{"tables":[{"columns":[{}],"rows":[{"a":1}]}]}`,
			want: `.columns[0].key`,
		},
		{
			name: "unknown formula placeholder",
			in:   `{"tables":[{"columns":[{"key":"a"},{"formula":"={missing}"}],"rows":[[1,2]]}]}`,
			want: `.columns[1].formula`,
		},
		{
			name: "reserved key r",
			in:   `{"tables":[{"columns":[{"key":"r"}],"rows":[[1]]}]}`,
			want: `.columns[0].key`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues, err := ValidateTableJSON([]byte(tc.in))
			if err != nil {
				t.Fatalf("ValidateTableJSON: %v", err)
			}
			found := false
			for _, iss := range issues {
				if strings.Contains(iss.Path, tc.want) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("issues = %+v, want path containing %q", issues, tc.want)
			}
		})
	}
}

func TestTableJSONColumnHeaderPrecedence(t *testing.T) {
	const input = `{
	  "tables": [{
	    "header": ["top-a", "top-b"],
	    "columns": [{"key": "a", "header": "col-a"}, {"key": "b"}],
	    "rows": [{"a": 1, "b": 2}]
	  }]
	}`
	f := convertTableJSON(t, input)
	if got, _ := f.GetCellValue("Sheet1", "A1"); got != "col-a" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellValue("Sheet1", "B1"); got != "top-b" {
		t.Fatalf("B1 = %q", got)
	}
}

func TestTableJSONCrossSheetCompute(t *testing.T) {
	const input = `{
	  "tables": [
	    {
	      "name": "Sales",
	      "columns": [
	        {"key": "item", "header": "item"},
	        {"key": "qty", "header": "qty"},
	        {"key": "price", "header": "price"},
	        {"key": "amount", "header": "amount", "formula": "={qty}*{price}"}
	      ],
	      "rows": [
	        {"item": "A", "qty": 10, "price": 100},
	        {"item": "B", "qty": 5, "price": 200}
	      ]
	    },
	    {
	      "name": "Summary",
	      "columns": [{"key": "label", "header": "label"}, {"key": "value", "header": "value"}],
	      "rows": [{"label": "total", "value": "=SUM(Sales!D2:D3)"}]
	    }
	  ]
	}`
	var buf bytes.Buffer
	if err := Convert(strings.NewReader(input), &buf, ConvertOptions{TableJSON: true, EvalFormulas: true}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, _ := f.GetCellValue("Summary", "B2"); got != "2000" {
		t.Fatalf("Summary!B2 = %q, want 2000", got)
	}
}
