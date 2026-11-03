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
