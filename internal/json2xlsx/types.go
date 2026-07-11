package json2xlsx

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

// ConditionalRule は1つの条件付き書式ルール。
type ConditionalRule struct {
	Type           string  `json:"type"`
	Criteria       string  `json:"criteria,omitempty"`
	Value          string  `json:"value,omitempty"`
	MinValue       string  `json:"minValue,omitempty"`
	MidValue       string  `json:"midValue,omitempty"`
	MaxValue       string  `json:"maxValue,omitempty"`
	Style          *Style  `json:"style,omitempty"`
	MinType        string  `json:"minType,omitempty"`
	MidType        string  `json:"midType,omitempty"`
	MaxType        string  `json:"maxType,omitempty"`
	MinColor       string  `json:"minColor,omitempty"`
	MidColor       string  `json:"midColor,omitempty"`
	MaxColor       string  `json:"maxColor,omitempty"`
	BarColor       string  `json:"barColor,omitempty"`
	BarBorderColor string  `json:"barBorderColor,omitempty"`
	BarDirection   string  `json:"barDirection,omitempty"`
	BarOnly        bool    `json:"barOnly,omitempty"`
	BarSolid       bool    `json:"barSolid,omitempty"`
	IconStyle      string  `json:"iconStyle,omitempty"`
	ReverseIcons   bool    `json:"reverseIcons,omitempty"`
	IconsOnly      bool    `json:"iconsOnly,omitempty"`
	AboveAverage   *bool   `json:"aboveAverage,omitempty"`
	Percent        bool    `json:"percent,omitempty"`
	StopIfTrue     bool    `json:"stopIfTrue,omitempty"`
}

// ConditionalFormat は1つの条件付き書式グループ（範囲＋ルール群）．
type ConditionalFormat struct {
	Range string            `json:"range"`
	Rules []ConditionalRule `json:"rules"`
}

// Style はスタイル定義。id によりセルから参照される。
type Style struct {
	ID        int        `json:"id,omitempty"`
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

// FreezePane は固定ペイン（フリーズパン）設定。
type FreezePane struct {
	Row int `json:"row,omitempty"` // 固定する行数
	Col int `json:"col,omitempty"` // 固定する列数
}

// Sparkline はスパークライン設定．
type Sparkline struct {
	Location      string  `json:"location"`
	Range         string  `json:"range"`
	Type          string  `json:"type,omitempty"`
	Weight        float64 `json:"weight,omitempty"`
	DateAxis      bool    `json:"dateAxis,omitempty"`
	Markers       bool    `json:"markers,omitempty"`
	High          bool    `json:"high,omitempty"`
	Low           bool    `json:"low,omitempty"`
	First         bool    `json:"first,omitempty"`
	Last          bool    `json:"last,omitempty"`
	Negative      bool    `json:"negative,omitempty"`
	Axis          bool    `json:"axis,omitempty"`
	Hidden        bool    `json:"hidden,omitempty"`
	Reverse       bool    `json:"reverse,omitempty"`
	Style         int     `json:"style,omitempty"`
	SeriesColor   string  `json:"seriesColor,omitempty"`
	NegativeColor string  `json:"negativeColor,omitempty"`
	MarkersColor  string  `json:"markersColor,omitempty"`
	FirstColor    string  `json:"firstColor,omitempty"`
	LastColor     string  `json:"lastColor,omitempty"`
	HighColor     string  `json:"highColor,omitempty"`
	LowColor      string  `json:"lowColor,omitempty"`
	EmptyCells    string  `json:"emptyCells,omitempty"`
	Max           int     `json:"max,omitempty"`
	CustMax       int     `json:"customMax,omitempty"`
	Min           int     `json:"min,omitempty"`
	CustMin       int     `json:"customMin,omitempty"`
}

// AutoFilter はオートフィルタ設定（SheetJS 互換）．
type AutoFilter struct {
	Ref string `json:"ref"`
}

// Table は構造化テーブル設定．
type Table struct {
	Range          string `json:"range"`
	Name           string `json:"name,omitempty"`
	StyleName      string `json:"style,omitempty"`
	BandedRows     *bool  `json:"bandedRows,omitempty"`
	BandedColumns  bool   `json:"bandedColumns,omitempty"`
	FirstColumn    bool   `json:"firstColumn,omitempty"`
	LastColumn     bool   `json:"lastColumn,omitempty"`
	HeaderRow      *bool  `json:"headerRow,omitempty"`
}

// Picture はワークシートに配置する画像。
type Picture struct {
	Cell           string  `json:"cell"`
	Path           string  `json:"path,omitempty"`
	Data           string  `json:"data,omitempty"`     // base64 エンコードされた画像データ
	Extension      string  `json:"extension,omitempty"` // 拡張子 (png, jpg, gif...)
	AltText        string  `json:"altText,omitempty"`
	PrintObject    *bool   `json:"printObject,omitempty"`
	Locked         *bool   `json:"locked,omitempty"`
	LockAspectRatio *bool `json:"lockAspectRatio,omitempty"`
	OffsetX        int     `json:"offsetX,omitempty"`
	OffsetY        int     `json:"offsetY,omitempty"`
	ScaleX         float64 `json:"scaleX,omitempty"`
	ScaleY         float64 `json:"scaleY,omitempty"`
	Hyperlink      string  `json:"hyperlink,omitempty"`
	Positioning    string  `json:"positioning,omitempty"`
}

// SheetBackground はワークシートの背景画像。
type SheetBackground struct {
	Path      string `json:"path,omitempty"`
	Data      string `json:"data,omitempty"`
	Extension string `json:"extension,omitempty"`
}

// Sheet は 1 シート分の定義。
type Sheet struct {
	Name               string              `json:"name,omitempty"`
	Cells              map[string]Cell     `json:"cells,omitempty"`
	Rows               [][]interface{}     `json:"rows,omitempty"`   // AoA 形式
	Cols               []ColInfo           `json:"cols,omitempty"`
	RowDims            []RowInfo           `json:"rowDims,omitempty"`
	Merges             []Merge             `json:"merges,omitempty"`
	Freeze             *FreezePane         `json:"freeze,omitempty"`
	AutoFilter         interface{}         `json:"autoFilter,omitempty"`
	Tables             []Table             `json:"tables,omitempty"`
	Sparklines         []Sparkline         `json:"sparklines,omitempty"`
	ConditionalFormats []ConditionalFormat `json:"conditionalFormats,omitempty"`
	Pictures           []Picture           `json:"pictures,omitempty"`
	Background         *SheetBackground    `json:"background,omitempty"`
}

// Chart はグラフオブジェクト。chart-json-spec.md の ChartObject に対応。
type Chart struct {
	ID     string        `json:"id,omitempty"`
	T      string        `json:"t,omitempty"`
	Mode   string        `json:"mode,omitempty"` // "embedded"(default) | "chartSheet"
	Ct     string        `json:"ct,omitempty"`
	Sheet  string        `json:"sheet,omitempty"`
	Anchor string        `json:"anchor,omitempty"`
	Dim    *ChartDim     `json:"dim,omitempty"`
	Title  *ChartTitle   `json:"title,omitempty"`
	Legend *ChartLegend  `json:"legend,omitempty"`
	Plot   *ChartPlot    `json:"plot,omitempty"`
	XAxis  *ChartAxis    `json:"xAxis,omitempty"`
	YAxis  *ChartAxis    `json:"yAxis,omitempty"`
	Ser    []ChartSeries `json:"ser,omitempty"`
	Style  interface{}   `json:"style,omitempty"`
	Meta   interface{}   `json:"meta,omitempty"`
}

type ChartDim struct {
	W    float64 `json:"w,omitempty"`
	H    float64 `json:"h,omitempty"`
	OffX float64 `json:"offx,omitempty"`
	OffY float64 `json:"offy,omitempty"`
	Sx   float64 `json:"sx,omitempty"`
	Sy   float64 `json:"sy,omitempty"`
}

type ChartTitle struct {
	Tx      string `json:"tx,omitempty"`
	Overlay bool   `json:"overlay,omitempty"`
}

type ChartLegend struct {
	Show bool   `json:"show,omitempty"`
	Pos  string `json:"pos,omitempty"`
}

type ChartPlot struct {
	VaryColors   bool   `json:"varyColors,omitempty"`
	ShowBlanksAs string `json:"showBlanksAs,omitempty"`
}

type ChartAxis struct {
	Title          string   `json:"title,omitempty"`
	Minimum        *float64 `json:"minimum,omitempty"`
	Maximum        *float64 `json:"maximum,omitempty"`
	MajorUnit      *float64 `json:"majorUnit,omitempty"`
	MinorUnit      *float64 `json:"minorUnit,omitempty"`
	ReverseOrder   bool     `json:"reverseOrder,omitempty"`
	MajorGridLines bool     `json:"majorGridLines,omitempty"`
	MinorGridLines bool     `json:"minorGridLines,omitempty"`
	NumFmt         string   `json:"numFmt,omitempty"`
}

type ChartSeries struct {
	Name   string         `json:"name,omitempty"`
	Cat    string         `json:"cat,omitempty"`
	Val    string         `json:"val,omitempty"`
	XVal   *string        `json:"xVal,omitempty"`
	YVal   *string        `json:"yVal,omitempty"`
	Bubble *string        `json:"bubble,omitempty"`
	Line   *ChartLine     `json:"line,omitempty"`
	Fill   *ChartFill     `json:"fill,omitempty"`
	Marker *ChartMarker   `json:"marker,omitempty"`
	DLbls  *ChartDLbls    `json:"dLbls,omitempty"`
}

type ChartLine struct {
	Width float64 `json:"width,omitempty"`
}

type ChartFill struct {
	Color string `json:"color,omitempty"`
}

type ChartMarker struct {
	Symbol string  `json:"symbol,omitempty"`
	Size   float64 `json:"size,omitempty"`
}

type ChartDLbls struct {
	ShowVal      bool `json:"showVal,omitempty"`
	ShowCatName  bool `json:"showCatName,omitempty"`
	ShowSerName  bool `json:"showSerName,omitempty"`
	ShowPercent  bool `json:"showPercent,omitempty"`
	ShowLeaderLn bool `json:"showLeaderLn,omitempty"`
}

// Book は book ラッパー形式の内部構造。
type Book struct {
	Props  interface{}      `json:"props,omitempty"`
	Sheets map[string]Sheet `json:"sheets,omitempty"`
	Charts []Chart          `json:"charts,omitempty"`
	Styles []Style          `json:"styles,omitempty"`
}

// Workbook はトップレベルの JSON 構造。
//
// 単一シート形式 (cells を直接持つ), 複数シート形式 (sheets 配列),
// book ラッパー形式 (version + book) のすべてに対応する。
type Workbook struct {
	// book ラッパー形式
	Version string `json:"version,omitempty"`
	Book    *Book  `json:"book,omitempty"`

	// 複数シート
	Sheets []Sheet `json:"sheets,omitempty"`

	// 単一シート (Sheet と同じフィールド)
	Name               string              `json:"name,omitempty"`
	Cells              map[string]Cell     `json:"cells,omitempty"`
	Rows               [][]interface{}     `json:"rows,omitempty"`
	Cols               []ColInfo           `json:"cols,omitempty"`
	RowDims            []RowInfo           `json:"rowDims,omitempty"`
	Merges             []Merge             `json:"merges,omitempty"`
	Freeze             *FreezePane         `json:"freeze,omitempty"`
	AutoFilter         interface{}         `json:"autoFilter,omitempty"`
	Tables             []Table             `json:"tables,omitempty"`
	Sparklines         []Sparkline         `json:"sparklines,omitempty"`
	ConditionalFormats []ConditionalFormat `json:"conditionalFormats,omitempty"`
	Pictures           []Picture           `json:"pictures,omitempty"`
	Background         *SheetBackground    `json:"background,omitempty"`

	Styles []Style `json:"styles,omitempty"`
}

// Link はハイパーリンク表現。
type Link struct {
	Target  string `json:"target"`
	Tooltip string `json:"tooltip,omitempty"`
}

// ImageMode は画像出力モード（XLSX→JSON 変換時の画像データの表現方法）。
type ImageMode string

const (
	ImageModeBase64 ImageMode = "base64"
	ImageModeFile   ImageMode = "file"
)
