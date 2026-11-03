package json2xlsx

import "testing"

func TestDetectJSONFormat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"2D array", `[["a",1]]`, "data"},
		{"object array", `[{"a":1}]`, "data"},
		{"table JSON", `{"tables":[{"rows":[[1]]}]}`, "table"},
		{"table JSON with header", `{"tables":[{"header":["A"]}]}`, "table"},
		{"table JSON with column header", `{"tables":[{"columns":[{"header":"A"}]}]}`, "table"},
		{"empty tables", `{"tables":[]}`, "sheetjs"},
		{"structured table", `{"tables":[{"range":"A1:B2"}]}`, "sheetjs"},
		{"sheetjs cells", `{"cells":{"A1":{"t":"s","v":"x"}}}`, "sheetjs"},
		{"sheetjs cells with tables", `{"cells":{"A1":{"t":"s","v":"x"}},"tables":[{"range":"A1:B2"}]}`, "sheetjs"},
		{"sheetjs sheets", `{"sheets":[{"cells":{}}]}`, "sheetjs"},
		{"sheetjs book", `{"book":{"sheets":{"S":{"cells":{}}}}}`, "sheetjs"},
		{"sheetjs library shape", `{"SheetNames":["S"],"Sheets":{"S":{"A1":{"v":"x"}}}}`, "sheetjs"},
		{"sheetjs rows only", `{"name":"S","rows":[["a",1]]}`, "sheetjs"},
		{"map of arrays", `{"a":[1,2],"b":[3,4]}`, "data"},
		{"map of arrays with name column", `{"name":["a","b"],"age":[1,2]}`, "data"},
		{"empty object", `{}`, "sheetjs"},
		{"BOM data", "\xef\xbb\xbf[[1]]", "data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectJSONFormat([]byte(tt.in)); got != tt.want {
				t.Fatalf("DetectJSONFormat(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
