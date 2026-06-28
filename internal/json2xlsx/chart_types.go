package json2xlsx

import "encoding/xml"

// --- 内部 XML パース用の最小限の型定義 ---
// Excelize v2.8.1 ではこれらの型が非公開のため、自前で定義する。
// 完全な struct ではなく、必要なフィールドのみを定義する（Go xml.Decoder は lenient）。

type chartRels struct {
	XMLName       xml.Name       `xml:"http://schemas.openxmlformats.org/package/2006/relationships Relationships"`
	Relationships []chartRel     `xml:"Relationship"`
}

type chartRel struct {
	ID         string `xml:"Id,attr"`
	Target     string `xml:"Target,attr"`
	Type       string `xml:"Type,attr"`
	TargetMode string `xml:"TargetMode,attr,omitempty"`
}

type chartSheetXML struct {
	XMLName xml.Name         `xml:"http://schemas.openxmlformats.org/spreadsheetml/2006/main chartsheet"`
	Drawing *chartDrawingXML `xml:"drawing"`
}

type chartDrawingXML struct {
	ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type chartSpaceXML struct {
	XMLName xml.Name      `xml:"http://schemas.openxmlformats.org/drawingml/2006/chart chartSpace"`
	Chart   chartMainXML  `xml:"chart"`
}

type chartMainXML struct {
	Title    *chartTitleXML    `xml:"title"`
	Legend   *chartLegendXML   `xml:"legend"`
	PlotArea *chartPlotAreaXML `xml:"plotArea"`
}

type chartTitleXML struct {
	Tx chartTxXML `xml:"tx"`
}

type chartTxXML struct {
	StrRef *chartStrRefXML `xml:"strRef"`
	Rich   *chartRichXML   `xml:"rich"`
}

type chartStrRefXML struct {
	F string `xml:"f"`
}

type chartRichXML struct {
	P []chartPXML `xml:"http://schemas.openxmlformats.org/drawingml/2006/main p"`
}

type chartPXML struct {
	R *chartRXML `xml:"http://schemas.openxmlformats.org/drawingml/2006/main r"`
}

type chartRXML struct {
	Text string `xml:"http://schemas.openxmlformats.org/drawingml/2006/main t"`
}

type chartLegendXML struct {
	LegendPos *chartAttrStr `xml:"legendPos"`
}

type chartAttrStr struct {
	Val *string `xml:"val,attr"`
}

type chartPlotAreaXML struct {
	AreaChart     *chartChartsXML `xml:"areaChart"`
	Area3DChart   *chartChartsXML `xml:"area3DChart"`
	BarChart      *chartBarXML    `xml:"barChart"`
	Bar3DChart    *chartBarXML    `xml:"bar3DChart"`
	BubbleChart   *chartChartsXML `xml:"bubbleChart"`
	DoughnutChart *chartChartsXML `xml:"doughnutChart"`
	LineChart     *chartChartsXML `xml:"lineChart"`
	Line3DChart   *chartChartsXML `xml:"line3DChart"`
	PieChart      *chartChartsXML `xml:"pieChart"`
	Pie3DChart    *chartChartsXML `xml:"pie3DChart"`
	RadarChart    *chartChartsXML `xml:"radarChart"`
	ScatterChart  *chartChartsXML `xml:"scatterChart"`
	CatAx         []chartAxsXML   `xml:"catAx"`
	ValAx         []chartAxsXML   `xml:"valAx"`
}

type chartBarXML struct {
	BarDir *chartAttrStr `xml:"barDir"`
	Ser    []chartSerXML `xml:"ser"`
}

type chartChartsXML struct {
	Ser []chartSerXML `xml:"ser"`
}

type chartSerXML struct {
	Tx     *chartTxXML      `xml:"tx"`
	Cat    *chartCatXML     `xml:"cat"`
	Val    *chartValXML     `xml:"val"`
	XVal   *chartCatXML     `xml:"xVal"`
	YVal   *chartValXML     `xml:"yVal"`
	Marker *chartMarkerXML  `xml:"marker"`
	SpPr   *chartSpPrXML    `xml:"spPr"`
	DLbls  *chartDLblsXML   `xml:"dLbls"`
}

type chartSpPrXML struct {
	NoFill    *string            `xml:"http://schemas.openxmlformats.org/drawingml/2006/main noFill"`
	SolidFill *chartSolidFillXML `xml:"http://schemas.openxmlformats.org/drawingml/2006/main solidFill"`
	Ln        *chartLnXML        `xml:"http://schemas.openxmlformats.org/drawingml/2006/main ln"`
}

type chartSolidFillXML struct {
	SrgbClr   *chartAttrStr      `xml:"http://schemas.openxmlformats.org/drawingml/2006/main srgbClr"`
	SchemeClr *chartSchemeClrXML `xml:"http://schemas.openxmlformats.org/drawingml/2006/main schemeClr"`
}

type chartSchemeClrXML struct {
	Val string `xml:"val,attr"`
}

type chartLnXML struct {
	W         int                `xml:"w,attr,omitempty"`
	NoFill    *string            `xml:"http://schemas.openxmlformats.org/drawingml/2006/main noFill"`
	SolidFill *chartSolidFillXML `xml:"http://schemas.openxmlformats.org/drawingml/2006/main solidFill"`
}

type chartDLblsXML struct {
	ShowVal         *chartBoolAttr `xml:"showVal"`
	ShowCatName     *chartBoolAttr `xml:"showCatName"`
	ShowSerName     *chartBoolAttr `xml:"showSerName"`
	ShowPercent     *chartBoolAttr `xml:"showPercent"`
	ShowLeaderLines *chartBoolAttr `xml:"showLeaderLines"`
	ShowLegendKey   *chartBoolAttr `xml:"showLegendKey"`
}

type chartBoolAttr struct {
	Val bool `xml:"val,attr"`
}

type chartCatXML struct {
	StrRef *chartStrRefXML `xml:"strRef"`
}

type chartValXML struct {
	NumRef *chartNumRefXML `xml:"numRef"`
}

type chartNumRefXML struct {
	F string `xml:"f"`
}

type chartMarkerXML struct {
	Symbol *chartAttrStr `xml:"symbol"`
	Size   *chartAttrInt `xml:"size"`
}

type chartAttrInt struct {
	Val *int `xml:"val,attr"`
}

type chartAxsXML struct {
	Title          *chartTitleXML     `xml:"title"`
	MajorGridlines *chartGridlinesXML `xml:"majorGridlines"`
	MinorGridlines *chartGridlinesXML `xml:"minorGridlines"`
	Scaling        *chartScalingXML   `xml:"scaling"`
	NumFmt         *chartNumFmtXML    `xml:"numFmt"`
}

type chartGridlinesXML struct{}

type chartScalingXML struct {
	Min *chartAttrFloat `xml:"min"`
	Max *chartAttrFloat `xml:"max"`
}

type chartAttrFloat struct {
	Val *float64 `xml:"val,attr"`
}

type chartNumFmtXML struct {
	FormatCode string `xml:"formatCode,attr"`
}

// workbook 内のシート情報をパースするための最小限の型（excelize の xlsxWorkbook が非公開のため）
type wbSheetXML struct {
	Sheet []wbSheetEntryXML `xml:"sheet"`
}

type wbSheetEntryXML struct {
	Name    string `xml:"name,attr"`
	SheetID int    `xml:"sheetId,attr"`
	ID      string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

// --- Drawing/Worksheet XML types for embedded chart extraction ---

type wsDrXML struct {
	XMLName         xml.Name           `xml:"wsDr"`
	TwoCellAnchors  []xdrTwoCellAnchor `xml:"twoCellAnchor"`
	OneCellAnchors  []xdrOneCellAnchor `xml:"oneCellAnchor"`
	AbsoluteAnchors []xdrAbsAnchor     `xml:"absoluteAnchor"`
}

type xdrTwoCellAnchor struct {
	From         xdrCellPoint     `xml:"from"`
	To           xdrCellPoint     `xml:"to"`
	GraphicFrame *xdrGraphicFrame `xml:"graphicFrame"`
}

type xdrOneCellAnchor struct {
	From         xdrCellPoint     `xml:"from"`
	Ext          xdrExt           `xml:"ext"`
	GraphicFrame *xdrGraphicFrame `xml:"graphicFrame"`
}

type xdrAbsAnchor struct {
	Pos          xdrPos           `xml:"pos"`
	Ext          xdrExt           `xml:"ext"`
	GraphicFrame *xdrGraphicFrame `xml:"graphicFrame"`
}

type xdrCellPoint struct {
	Col    int   `xml:"col"`
	Row    int   `xml:"row"`
	ColOff int64 `xml:"colOff"`
	RowOff int64 `xml:"rowOff"`
}

type xdrPos struct {
	X int64 `xml:"x,attr"`
	Y int64 `xml:"y,attr"`
}

type xdrExt struct {
	Cx int64 `xml:"cx,attr"`
	Cy int64 `xml:"cy,attr"`
}

type xdrGraphicFrame struct {
	Graphic *xdrGraphic `xml:"http://schemas.openxmlformats.org/drawingml/2006/main graphic"`
}

type xdrGraphic struct {
	GraphicData *xdrGraphicData `xml:"http://schemas.openxmlformats.org/drawingml/2006/main graphicData"`
}

type xdrGraphicData struct {
	URI   string       `xml:"uri,attr"`
	Chart *xdrChartRef `xml:"http://schemas.openxmlformats.org/drawingml/2006/chart chart"`
}

type xdrChartRef struct {
	ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}
