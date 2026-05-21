package sheet2xlsx

// Cell は SheetJS 風のセルオブジェクトを表す。
//
//	t: セル型 ("s" 文字列, "n" 数値, "b" 真偽値, "f" 数式, "d" 日付)
//	v: セルの値
//	f: 数式 (t="f" のとき必須)
//	z: 数値書式コード (例: "#,##0")
//	s: styles 配列内の id への参照
//	l: ハイパーリンク (string または {target, tooltip})
type Cell struct {
	T string      `json:"t,omitempty"`
	V interface{} `json:"v,omitempty"`
	F string      `json:"f,omitempty"`
	Z string      `json:"z,omitempty"`
	S int         `json:"s,omitempty"`
	L interface{} `json:"l,omitempty"`
}

// Fill は塗りつぶし設定。
type Fill struct {
	Type    string   `json:"type"`
	Pattern int      `json:"pattern"`
	Color   []string `json:"color"`
}

// Border は罫線設定。
//
//	style: thin, medium, thick, dashed, dotted, double
//	side : left, right, top, bottom (省略時は全辺)
type Border struct {
	Style string `json:"style"`
	Color string `json:"color"`
	Side  string `json:"side,omitempty"`
}

// Font は文字スタイル。
type Font struct {
	Name   string  `json:"name,omitempty"`
	Size   float64 `json:"size,omitempty"`
	Bold   bool    `json:"bold,omitempty"`
	Italic bool    `json:"italic,omitempty"`
	Color  string  `json:"color,omitempty"`
}

// Alignment はセル内配置。
type Alignment struct {
	Horizontal string `json:"horizontal,omitempty"` // left, center, right
	Vertical   string `json:"vertical,omitempty"`   // top, center, bottom
	WrapText   bool   `json:"wrapText,omitempty"`
}

// Style はスタイル定義。id によりセルから参照される。
type Style struct {
	ID        int        `json:"id"`
	Fill      *Fill      `json:"fill,omitempty"`
	Border    []Border   `json:"border,omitempty"`
	Font      *Font      `json:"font,omitempty"`
	Alignment *Alignment `json:"alignment,omitempty"`
	NumFmt    string     `json:"numFmt,omitempty"`
}

// ColInfo は列幅指定。
type ColInfo struct {
	Col   string  `json:"col"`             // "A" など
	Width float64 `json:"width"`
}

// RowInfo は行高指定。
type RowInfo struct {
	Row    int     `json:"row"`            // 1 始まり
	Height float64 `json:"height"`
}

// Merge はマージセル指定。
type Merge struct {
	Range string `json:"range"` // 例 "A1:B2"
}

// Sheet は 1 シート分の定義。
type Sheet struct {
	Name    string           `json:"name,omitempty"`
	Cells   map[string]Cell  `json:"cells,omitempty"`
	Rows    [][]interface{}  `json:"rows,omitempty"`   // AoA 形式
	Cols    []ColInfo        `json:"cols,omitempty"`
	RowDims []RowInfo        `json:"rowDims,omitempty"`
	Merges  []Merge          `json:"merges,omitempty"`
}

// Workbook はトップレベルの JSON 構造。
//
// 単一シート形式 (cells を直接持つ) と複数シート形式 (sheets) の両方に対応する。
type Workbook struct {
	// 複数シート
	Sheets []Sheet `json:"sheets,omitempty"`

	// 単一シート (Sheet と同じフィールド)
	Name    string          `json:"name,omitempty"`
	Cells   map[string]Cell `json:"cells,omitempty"`
	Rows    [][]interface{} `json:"rows,omitempty"`
	Cols    []ColInfo       `json:"cols,omitempty"`
	RowDims []RowInfo       `json:"rowDims,omitempty"`
	Merges  []Merge         `json:"merges,omitempty"`

	Styles []Style `json:"styles,omitempty"`
}

// Link はハイパーリンク表現。
type Link struct {
	Target  string `json:"target"`
	Tooltip string `json:"tooltip,omitempty"`
}
