package json2xlsx

import (
	"encoding/xml"
	"fmt"
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

// parseChartXML は chart XML をパースして Chart 構造体を返す。
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

	ch.Ser = extractSeriesXML(cs.Chart.PlotArea, f)

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

// detectChartTypeXML は PlotArea 内のどのチャート要素が存在するかで種類を判定する。
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

// barDirToString は barDir 属性から "col" / "bar" を返す。
func barDirToString(b *chartAttrStr) string {
	if b != nil && b.Val != nil && *b.Val == "col" {
		return "col"
	}
	return "bar"
}

// extractTitleXML は chartTitleXML からタイトル文字列を抽出する。
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

// extractLegendXML は chartLegendXML から ChartLegend を生成する。
func extractLegendXML(legend *chartLegendXML) *ChartLegend {
	cl := &ChartLegend{Show: true}
	if legend.LegendPos != nil && legend.LegendPos.Val != nil {
		cl.Pos = legendPosReverse(*legend.LegendPos.Val)
	}
	return cl
}

// legendPosReverse は XML 内部の legendPos 略称を chart-json-spec のフル名に変換する。
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

// extractSeriesXML は PlotArea 内の全系列を抽出する。
func extractSeriesXML(pa *chartPlotAreaXML, f *excelize.File) []ChartSeries {
	allSer := collectAllSeriesXML(pa)
	if len(allSer) == 0 {
		return nil
	}
	result := make([]ChartSeries, 0, len(allSer))
	for _, s := range allSer {
		result = append(result, extractSingleSeriesXML(s, f))
	}
	return result
}

// collectAllSeriesXML は PlotArea 内の全系列 XML を収集する。
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

// resolveHelperSheetRef は補助シートを参照しているセル参照を解決し、実際の値を返す。
func resolveHelperSheetRef(f *excelize.File, ref string) string {
	if !strings.HasPrefix(ref, "'_xlsx") {
		return ref
	}
	closeQuote := strings.IndexByte(ref[1:], '\'')
	if closeQuote < 0 {
		return ref
	}
	sheetName := ref[1 : closeQuote+1]
	cell := ref[closeQuote+3:]
	val, err := f.GetCellValue(sheetName, cell)
	if err != nil || val == "" {
		return ref
	}
	return val
}

// extractSingleSeriesXML は 1 系列の XML を ChartSeries に変換する。
func extractSingleSeriesXML(s chartSerXML, f *excelize.File) ChartSeries {
	cs := ChartSeries{}
	if s.Tx != nil {
		if s.Tx.StrRef != nil && s.Tx.StrRef.F != "" {
			cs.Name = resolveHelperSheetRef(f, s.Tx.StrRef.F)
		} else if s.Tx.Rich != nil {
			for _, p := range s.Tx.Rich.P {
				if p.R != nil && p.R.Text != "" {
					cs.Name = p.R.Text
					break
				}
			}
		}
	}
	if s.Cat != nil && s.Cat.StrRef != nil {
		cs.Cat = s.Cat.StrRef.F
	}
	if s.Val != nil && s.Val.NumRef != nil {
		cs.Val = s.Val.NumRef.F
	}
	if s.XVal != nil && s.XVal.StrRef != nil {
		cs.XVal = &s.XVal.StrRef.F
		if cs.Cat == "" {
			cs.Cat = s.XVal.StrRef.F
		}
	}
	if s.YVal != nil && s.YVal.NumRef != nil {
		cs.YVal = &s.YVal.NumRef.F
		if cs.Val == "" {
			cs.Val = s.YVal.NumRef.F
		}
	}
	if s.Marker != nil {
		cs.Marker = extractMarkerXML(s.Marker)
	}
	if s.SpPr != nil {
		cs.Line = extractLineXML(s.SpPr)
		cs.Fill = extractFillXML(s.SpPr)
	}
	if s.DLbls != nil {
		cs.DLbls = extractDLblsXML(s.DLbls)
	}
	return cs
}

// extractMarkerXML は chartMarkerXML から ChartMarker を生成する。
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

// extractLineXML は spPr から線幅を抽出する。
func extractLineXML(spPr *chartSpPrXML) *ChartLine {
	if spPr.Ln == nil || spPr.Ln.NoFill != nil || spPr.Ln.W == 0 {
		return nil
	}
	return &ChartLine{Width: float64(spPr.Ln.W) / 12700}
}

// extractFillXML は spPr から塗りつぶし色を抽出する。
func extractFillXML(spPr *chartSpPrXML) *ChartFill {
	if spPr.SolidFill == nil || spPr.SolidFill.SrgbClr == nil || spPr.SolidFill.SrgbClr.Val == nil {
		return nil
	}
	val := *spPr.SolidFill.SrgbClr.Val
	if len(val) == 8 {
		val = val[2:]
	}
	return &ChartFill{Color: "#" + val}
}

// extractDLblsXML は chartDLblsXML から ChartDLbls を生成する。
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

// extractAxisXML は chartAxsXML から ChartAxis を生成する。
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
