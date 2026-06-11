package json2xlsx

import (
	"bytes"
	"strings"
	"testing"
)

func runToCSV(t *testing.T, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := ToCSV(strings.NewReader(input), &out)
	return out.String(), err
}

func TestToCSV_Basic(t *testing.T) {
	in := `[
  {
    "製品": "商品A\n特価",
    "数量": "100",
    "単価": "5,000",
    "合計": "500,000",
    "": null
  },
  {
    "製品": "商品B",
    "数量": "50",
    "単価": "8,000",
    "合計": "400,000",
    "": null
  },
  {
    "製品": "合計",
    "数量": null,
    "単価": null,
    "合計": "900,000",
    "": null
  }
]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "製品,数量,単価,合計,\n" +
		"\"商品A\n特価\",100,\"5,000\",\"500,000\",\n" +
		"商品B,50,\"8,000\",\"400,000\",\n" +
		"合計,,,\"900,000\",\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLIInput(t *testing.T) {
	in := "売上\n" + `[
  {
    "製品": "商品A\n特価",
    "数量": 100,
    "単価": 5000,
    "合計": ""
  },
  {
    "製品": "商品B",
    "数量": 50,
    "単価": 8000,
    "合計": ""
  },
  {
    "製品": "合計",
    "合計": ""
  }
]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "製品,数量,単価,合計\n" +
		"\"商品A\n特価\",100,5000,\n" +
		"商品B,50,8000,\n" +
		"合計,,,\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLIMultiSheet_FirstOnly(t *testing.T) {
	in := "Sheet1\n[{\"a\":1}]\nSheet2\n[{\"a\":2}]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n1\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLISheetWithoutArray(t *testing.T) {
	_, err := runToCSV(t, "売上\n")
	if err == nil {
		t.Fatal("expected error for sheet name without array")
	}
}

func TestToCSV_EmptyArray(t *testing.T) {
	_, err := runToCSV(t, `[]`)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestToCSV_WorkbookInput(t *testing.T) {
	_, err := runToCSV(t, `{"name":"S","cells":{}}`)
	if err == nil {
		t.Fatal("expected error for Workbook JSON")
	}
}

func TestToCSV_NonJSON(t *testing.T) {
	_, err := runToCSV(t, `hello`)
	if err == nil {
		t.Fatal("expected error for non-JSON")
	}
}

func TestToCSV_SingleRow(t *testing.T) {
	in := `[{"a":"1","b":"2"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b\n1,2\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_NullValues(t *testing.T) {
	in := `[{"a":null,"b":"x","c":null}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b,c\n,x,\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_RejectBoolValue(t *testing.T) {
	_, err := runToCSV(t, `[{"a":true}]`)
	if err == nil {
		t.Fatal("expected error for bool value")
	}
}

func TestToCSV_EmptyKey(t *testing.T) {
	in := `[{"":"val","b":"x"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := ",b\nval,x\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_EmbeddedQuotes(t *testing.T) {
	in := `[{"a":"he said \"hello\""}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n\"he said \"\"hello\"\"\"\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_EmbeddedNewlines(t *testing.T) {
	in := "[\n  {\"a\":\"line1\\nline2\"}\n]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n\"line1\nline2\"\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_DifferentKeys(t *testing.T) {
	in := `[{"a":"1"},{"b":"2"}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a,b\n1,\n,2\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_NumberLexemePreserved(t *testing.T) {
	in := `[{"n":1e3,"m":1.0,"p":-2E-3}]`
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "n,m,p\n1e3,1.0,-2E-3\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_BOM(t *testing.T) {
	in := "\xef\xbb\xbf[{\"a\":\"1\"}]"
	got, err := runToCSV(t, in)
	if err != nil {
		t.Fatalf("ToCSV: %v", err)
	}
	want := "a\n1\n"
	if got != want {
		t.Fatalf("mismatch.\n got:\n%q\nwant:\n%q", got, want)
	}
}

func TestToCSV_XLSXCLIAndCSVTKEquivalent(t *testing.T) {
	xlsxCLI := "売上\n" + `[
  {"製品":"商品A\n特価","数量":100,"単価":5000,"合計":""},
  {"製品":"商品B","数量":50,"単価":8000,"合計":""},
  {"製品":"合計","合計":""}
]`
	csvtk := `[
  {"製品":"商品A\n特価","数量":"100","単価":"5000","合計":null},
  {"製品":"商品B","数量":"50","単価":"8000","合計":null},
  {"製品":"合計","数量":null,"単価":null,"合計":null}
]`

	xlsxOut, err := runToCSV(t, xlsxCLI)
	if err != nil {
		t.Fatalf("ToCSV(xlsx-cli): %v", err)
	}
	csvtkOut, err := runToCSV(t, csvtk)
	if err != nil {
		t.Fatalf("ToCSV(csvtk): %v", err)
	}
	if xlsxOut != csvtkOut {
		t.Fatalf("xlsx-cli and csvtk outputs differ.\nxlsx-cli:\n%q\ncsvtk:\n%q", xlsxOut, csvtkOut)
	}
}

