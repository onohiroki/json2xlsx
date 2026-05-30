package sheet2xlsx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// chartTypeToString は Excelize の ChartType を chart-json-spec.md の ct 文字列に変換する。
func chartTypeToString(ct excelize.ChartType) (string, error) {
	switch ct {
	case excelize.Col:
		return "col", nil
	case excelize.Bar:
		return "bar", nil
	case excelize.Line:
		return "line", nil
	case excelize.Area:
		return "area", nil
	case excelize.Pie:
		return "pie", nil
	case excelize.Doughnut:
		return "doughnut", nil
	case excelize.Scatter:
		return "scatter", nil
	case excelize.Radar:
		return "radar", nil
	default:
		return "", fmt.Errorf("unsupported chart type %d", ct)
	}
}

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
	Title   *chartTitleXML   `xml:"title"`
	Legend  *chartLegendXML  `xml:"legend"`
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
	AreaChart     *chartChartsXML   `xml:"areaChart"`
	Area3DChart   *chartChartsXML   `xml:"area3DChart"`
	BarChart      *chartBarXML      `xml:"barChart"`
	Bar3DChart    *chartBarXML      `xml:"bar3DChart"`
	BubbleChart   *chartChartsXML   `xml:"bubbleChart"`
	DoughnutChart *chartChartsXML   `xml:"doughnutChart"`
	LineChart     *chartChartsXML   `xml:"lineChart"`
	Line3DChart   *chartChartsXML   `xml:"line3DChart"`
	PieChart      *chartChartsXML   `xml:"pieChart"`
	Pie3DChart    *chartChartsXML   `xml:"pie3DChart"`
	RadarChart    *chartChartsXML   `xml:"radarChart"`
	ScatterChart  *chartChartsXML   `xml:"scatterChart"`
	CatAx         []chartAxsXML     `xml:"catAx"`
	ValAx         []chartAxsXML     `xml:"valAx"`
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
	Title          *chartTitleXML   `xml:"title"`
	MajorGridlines *chartGridlinesXML `xml:"majorGridlines"`
	MinorGridlines *chartGridlinesXML `xml:"minorGridlines"`
	Scaling        *chartScalingXML   `xml:"scaling"`
	NumFmt         *chartNumFmtXML    `xml:"numFmt"`
}

type chartGridlinesXML struct {
}

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

// --- ここからチャート抽出ロジック ---

// workbook 内のシート情報をパースするための最小限の型（excelize の xlsxWorkbook が非公開のため）
type wbSheetXML struct {
	Sheet []wbSheetEntryXML `xml:"sheet"`
}

type wbSheetEntryXML struct {
	Name    string `xml:"name,attr"`
	SheetID int    `xml:"sheetId,attr"`
	ID      string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

// extractChartsheets は XLSX 内の chartsheet からグラフ情報を抽出する。
func extractChartsheets(f *excelize.File) ([]Chart, error) {
	rels, err := loadChartRels(f, "xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, fmt.Errorf("load workbook rels: %w", err)
	}
	if rels == nil {
		return nil, nil
	}

	// workbook.xml からシート一覧を読み取る（f.WorkBook の型が非公開のため直接 Pkg から）
	sheetEntries, err := loadSheetEntries(f)
	if err != nil {
		return nil, fmt.Errorf("load sheet entries: %w", err)
	}
	if len(sheetEntries) == 0 {
		return nil, nil
	}

	// rId → relationship のマップ
	sheetRels := make(map[string]*chartRel)
	for i := range rels.Relationships {
		sheetRels[rels.Relationships[i].ID] = &rels.Relationships[i]
	}

	const chartsheetRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet"

	var charts []Chart
	for _, sh := range sheetEntries {
		rel, ok := sheetRels[sh.ID]
		if !ok {
			continue
		}
		if rel.Type != chartsheetRelType {
			continue
		}

		csPath := resolveRelPath("xl/workbook.xml", rel.Target)
		ch, ok, err := extractChartFromChartsheet(f, csPath, sh.Name)
		if err != nil {
			return nil, fmt.Errorf("extract chartsheet %q: %w", sh.Name, err)
		}
		if ok {
			charts = append(charts, *ch)
		}
	}
	return charts, nil
}

// loadSheetEntries は xl/workbook.xml からシートエントリ一覧を読み込む。
func loadSheetEntries(f *excelize.File) ([]wbSheetEntryXML, error) {
	raw, ok := f.Pkg.Load("xl/workbook.xml")
	if !ok {
		return nil, nil
	}
	var wb struct {
		Sheets wbSheetXML `xml:"sheets"`
	}
	if err := xml.Unmarshal(raw.([]byte), &wb); err != nil {
		return nil, err
	}
	return wb.Sheets.Sheet, nil
}

func extractChartFromChartsheet(f *excelize.File, csPath, sheetName string) (*Chart, bool, error) {
	rawCS, ok := f.Pkg.Load(csPath)
	if !ok {
		return nil, false, nil
	}

	var cs chartSheetXML
	if err := xml.Unmarshal(rawCS.([]byte), &cs); err != nil {
		return nil, false, fmt.Errorf("unmarshal chartsheet: %w", err)
	}
	if cs.Drawing == nil || cs.Drawing.ID == "" {
		return nil, false, nil
	}

	// 1. chartsheet の rels から drawing への関係を取得
	csRelsPath := chartsheetRelsPath(csPath)
	csRels, err := loadChartRels(f, csRelsPath)
	if err != nil {
		return nil, false, fmt.Errorf("load chartsheet rels: %w", err)
	}
	if csRels == nil {
		return nil, false, nil
	}

	const drawingRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing"
	var drawingTarget string
	for _, r := range csRels.Relationships {
		if r.ID == cs.Drawing.ID && r.Type == drawingRelType {
			drawingTarget = r.Target
			break
		}
	}
	if drawingTarget == "" {
		return nil, false, nil
	}

	drawingPath := resolveRelPath(csPath, drawingTarget)

	// 2. drawing の rels から chart への関係を取得
	drawingDir := filepath.Dir(drawingPath)
	drawingRelsPath := drawingDir + "/_rels/" + filepath.Base(drawingPath) + ".rels"
	drawingRels, err := loadChartRels(f, drawingRelsPath)
	if err != nil {
		return nil, false, fmt.Errorf("load drawing rels: %w", err)
	}
	if drawingRels == nil {
		return nil, false, nil
	}

	const chartRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	var chartPath string
	for _, r := range drawingRels.Relationships {
		if r.Type == chartRelType {
			chartPath = resolveRelPath(drawingPath, r.Target)
			break
		}
	}
	if chartPath == "" {
		return nil, false, nil
	}

	return parseChartXML(f, chartPath, sheetName)
}

func chartsheetRelsPath(csPath string) string {
	csDir := filepath.Dir(csPath)
	return csDir + "/_rels/" + filepath.Base(csPath) + ".rels"
}

func parseChartXML(f *excelize.File, chartPath, sheetName string) (*Chart, bool, error) {
	rawChart, ok := f.Pkg.Load(chartPath)
	if !ok {
		return nil, false, nil
	}

	var cs chartSpaceXML
	if err := xml.Unmarshal(rawChart.([]byte), &cs); err != nil {
		return nil, false, fmt.Errorf("unmarshal chart: %w", err)
	}

	if cs.Chart.PlotArea == nil {
		return nil, false, nil
	}

	ch := &Chart{
		Mode:  "chartSheet",
		Sheet: sheetName,
	}

	if ct, ok := detectChartTypeXML(cs.Chart.PlotArea); ok {
		ch.Ct = ct
	}

	if cs.Chart.Title != nil {
		if t := extractTitleXML(cs.Chart.Title); t != "" {
			ch.Title = &ChartTitle{Tx: t}
		}
	}

	if cs.Chart.Legend != nil {
		ch.Legend = extractLegendXML(cs.Chart.Legend)
	}

	ch.Ser = extractSeriesXML(cs.Chart.PlotArea)

	if cs.Chart.PlotArea.CatAx != nil {
		for _, ax := range cs.Chart.PlotArea.CatAx {
			if a := extractAxisXML(&ax); a != nil {
				ch.XAxis = a
			}
		}
	}
	if cs.Chart.PlotArea.ValAx != nil {
		for i, ax := range cs.Chart.PlotArea.ValAx {
			if a := extractAxisXML(&ax); a != nil {
				if i == 0 {
					ch.YAxis = a
				}
			}
		}
	}

	return ch, true, nil
}

func detectChartTypeXML(pa *chartPlotAreaXML) (string, bool) {
	switch {
	case pa.AreaChart != nil || pa.Area3DChart != nil:
		return "area", true
	case pa.BarChart != nil:
		return barDirToString(pa.BarChart.BarDir), true
	case pa.Bar3DChart != nil:
		return barDirToString(pa.Bar3DChart.BarDir), true
	case pa.BubbleChart != nil:
		return "bubble", true
	case pa.DoughnutChart != nil:
		return "doughnut", true
	case pa.LineChart != nil || pa.Line3DChart != nil:
		return "line", true
	case pa.PieChart != nil || pa.Pie3DChart != nil:
		return "pie", true
	case pa.RadarChart != nil:
		return "radar", true
	case pa.ScatterChart != nil:
		return "scatter", true
	default:
		return "", false
	}
}

func barDirToString(b *chartAttrStr) string {
	if b != nil && b.Val != nil && *b.Val == "col" {
		return "col"
	}
	return "bar"
}

func extractTitleXML(title *chartTitleXML) string {
	if title.Tx.StrRef != nil && title.Tx.StrRef.F != "" {
		return title.Tx.StrRef.F
	}
	if title.Tx.Rich != nil {
		for _, p := range title.Tx.Rich.P {
			if p.R != nil && p.R.Text != "" {
				return p.R.Text
			}
		}
	}
	return ""
}

func extractLegendXML(legend *chartLegendXML) *ChartLegend {
	cl := &ChartLegend{Show: true}
	if legend.LegendPos != nil && legend.LegendPos.Val != nil {
		// legendPosReverse で内部略称 → フル名に変換（XML では "b"/"r"/"l"/"t"/"tr"）
		cl.Pos = legendPosReverse(*legend.LegendPos.Val)
	}
	return cl
}

// legendPosReverse は XML 内部の legendPos 略称を chart-json-spec のフル名に変換する。
// 既にフル名の場合はそのまま返す。
func legendPosReverse(s string) string {
	switch s {
	case "r":
		return "right"
	case "l":
		return "left"
	case "t":
		return "top"
	case "b":
		return "bottom"
	case "tr":
		return "topRight"
	default:
		return s
	}
}

func extractSeriesXML(pa *chartPlotAreaXML) []ChartSeries {
	allSer := collectAllSeriesXML(pa)
	if len(allSer) == 0 {
		return nil
	}
	result := make([]ChartSeries, 0, len(allSer))
	for _, s := range allSer {
		result = append(result, extractSingleSeriesXML(s))
	}
	return result
}

func collectAllSeriesXML(pa *chartPlotAreaXML) []chartSerXML {
	for _, area := range []*chartChartsXML{pa.AreaChart, pa.Area3DChart, pa.BubbleChart, pa.DoughnutChart, pa.LineChart, pa.Line3DChart, pa.PieChart, pa.Pie3DChart, pa.RadarChart, pa.ScatterChart} {
		if area != nil && area.Ser != nil {
			return area.Ser
		}
	}
	if pa.BarChart != nil && pa.BarChart.Ser != nil {
		return pa.BarChart.Ser
	}
	if pa.Bar3DChart != nil && pa.Bar3DChart.Ser != nil {
		return pa.Bar3DChart.Ser
	}
	return nil
}

func extractSingleSeriesXML(s chartSerXML) ChartSeries {
	cs := ChartSeries{}
	if s.Tx != nil && s.Tx.StrRef != nil && s.Tx.StrRef.F != "" {
		cs.Name = s.Tx.StrRef.F
	}
	if s.Cat != nil && s.Cat.StrRef != nil {
		cs.Cat = s.Cat.StrRef.F
	}
	if s.Val != nil && s.Val.NumRef != nil {
		cs.Val = s.Val.NumRef.F
	}
	if s.XVal != nil && s.XVal.StrRef != nil {
		cs.XVal = &s.XVal.StrRef.F
		// scatter: xVal → cat (JSON 仕様の cat/val 統一のため)
		if cs.Cat == "" {
			cs.Cat = s.XVal.StrRef.F
		}
	}
	if s.YVal != nil && s.YVal.NumRef != nil {
		cs.YVal = &s.YVal.NumRef.F
		// scatter: yVal → val (JSON 仕様の cat/val 統一のため)
		if cs.Val == "" {
			cs.Val = s.YVal.NumRef.F
		}
	}
	if s.Marker != nil {
		cs.Marker = extractMarkerXML(s.Marker)
	}
	// spPr → line, fill
	if s.SpPr != nil {
		cs.Line = extractLineXML(s.SpPr)
		cs.Fill = extractFillXML(s.SpPr)
	}
	// dLbls
	if s.DLbls != nil {
		cs.DLbls = extractDLblsXML(s.DLbls)
	}
	return cs
}

func extractMarkerXML(m *chartMarkerXML) *ChartMarker {
	cm := &ChartMarker{}
	if m.Symbol != nil && m.Symbol.Val != nil {
		cm.Symbol = *m.Symbol.Val
	}
	if m.Size != nil && m.Size.Val != nil {
		cm.Size = float64(*m.Size.Val)
	}
	return cm
}

func extractLineXML(spPr *chartSpPrXML) *ChartLine {
	if spPr.Ln == nil || spPr.Ln.NoFill != nil || spPr.Ln.W == 0 {
		return nil
	}
	// EMU → pt (1pt = 12700 EMUs)
	return &ChartLine{Width: float64(spPr.Ln.W) / 12700}
}

func extractFillXML(spPr *chartSpPrXML) *ChartFill {
	if spPr.SolidFill == nil || spPr.SolidFill.SrgbClr == nil || spPr.SolidFill.SrgbClr.Val == nil {
		return nil
	}
	return &ChartFill{Color: "#" + *spPr.SolidFill.SrgbClr.Val}
}

func extractDLblsXML(d *chartDLblsXML) *ChartDLbls {
	dl := &ChartDLbls{}
	if d.ShowVal != nil {
		dl.ShowVal = d.ShowVal.Val
	}
	if d.ShowCatName != nil {
		dl.ShowCatName = d.ShowCatName.Val
	}
	if d.ShowSerName != nil {
		dl.ShowSerName = d.ShowSerName.Val
	}
	if d.ShowPercent != nil {
		dl.ShowPercent = d.ShowPercent.Val
	}
	if d.ShowLeaderLines != nil {
		dl.ShowLeaderLn = d.ShowLeaderLines.Val
	}
	return dl
}

func extractAxisXML(ax *chartAxsXML) *ChartAxis {
	if ax == nil {
		return nil
	}
	a := &ChartAxis{}
	if ax.Title != nil {
		if t := extractTitleXML(ax.Title); t != "" {
			a.Title = t
		}
	}
	if ax.MajorGridlines != nil {
		a.MajorGridLines = true
	}
	if ax.MinorGridlines != nil {
		a.MinorGridLines = true
	}
	if ax.Scaling != nil {
		if ax.Scaling.Min != nil && ax.Scaling.Min.Val != nil {
			a.Minimum = ax.Scaling.Min.Val
		}
		if ax.Scaling.Max != nil && ax.Scaling.Max.Val != nil {
			a.Maximum = ax.Scaling.Max.Val
		}
	}
	if ax.NumFmt != nil {
		a.NumFmt = ax.NumFmt.FormatCode
	}
	return a
}

func loadChartRels(f *excelize.File, path string) (*chartRels, error) {
	raw, ok := f.Pkg.Load(path)
	if !ok {
		return nil, nil
	}
	var rels chartRels
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&rels); err != nil && err != io.EOF {
		return nil, err
	}
	return &rels, nil
}

func resolveRelPath(basePath, relTarget string) string {
	if strings.HasPrefix(relTarget, "/") {
		return strings.TrimPrefix(relTarget, "/")
	}
	baseDir := filepath.Dir(basePath)
	cleaned := filepath.Clean(baseDir + "/" + relTarget)
	return cleaned
}

// listChartsheetNames は chartsheet のシート名一覧を返す。
func listChartsheetNames(f *excelize.File) (map[string]bool, error) {
	rels, err := loadChartRels(f, "xl/_rels/workbook.xml.rels")
	if err != nil || rels == nil {
		return nil, err
	}

	sheetEntries, err := loadSheetEntries(f)
	if err != nil || len(sheetEntries) == 0 {
		return nil, err
	}

	sheetRels := make(map[string]*chartRel)
	for i := range rels.Relationships {
		sheetRels[rels.Relationships[i].ID] = &rels.Relationships[i]
	}

	const chartsheetRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet"
	names := make(map[string]bool)
	for _, sh := range sheetEntries {
		if rel, ok := sheetRels[sh.ID]; ok && rel.Type == chartsheetRelType {
			names[sh.Name] = true
		}
	}
	return names, nil
}
