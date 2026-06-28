package json2xlsx

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestChartTypeToString_AllTypes(t *testing.T) {
	cases := []struct {
		ct   excelize.ChartType
		want string
	}{
		{excelize.Col, "col"},
		{excelize.Bar, "bar"},
		{excelize.Line, "line"},
		{excelize.Area, "area"},
		{excelize.Pie, "pie"},
		{excelize.Doughnut, "doughnut"},
		{excelize.Scatter, "scatter"},
		{excelize.Radar, "radar"},
	}
	for _, c := range cases {
		got, err := chartTypeToString(c.ct)
		if err != nil {
			t.Errorf("chartTypeToString(%d): unexpected error: %v", c.ct, err)
			continue
		}
		if got != c.want {
			t.Errorf("chartTypeToString(%d) = %q, want %q", c.ct, got, c.want)
		}
	}
}

func TestChartTypeToString_Unsupported(t *testing.T) {
	_, err := chartTypeToString(excelize.ChartType(255))
	if err == nil {
		t.Fatal("expected error for unsupported chart type")
	}
}

func TestBarDirToString_Col(t *testing.T) {
	v := "col"
	b := &chartAttrStr{Val: &v}
	if got := barDirToString(b); got != "col" {
		t.Errorf("barDirToString(col) = %q, want col", got)
	}
}

func TestBarDirToString_Bar(t *testing.T) {
	v := "bar"
	b := &chartAttrStr{Val: &v}
	if got := barDirToString(b); got != "bar" {
		t.Errorf("barDirToString(bar) = %q, want bar", got)
	}
}

func TestBarDirToString_NilVal(t *testing.T) {
	b := &chartAttrStr{Val: nil}
	if got := barDirToString(b); got != "bar" {
		t.Errorf("barDirToString(nil val) = %q, want bar", got)
	}
}

func TestBarDirToString_Nil(t *testing.T) {
	if got := barDirToString(nil); got != "bar" {
		t.Errorf("barDirToString(nil) = %q, want bar", got)
	}
}

func TestLegendPosReverse(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"r", "right"},
		{"l", "left"},
		{"t", "top"},
		{"b", "bottom"},
		{"tr", "topRight"},
		{"right", "right"},
		{"custom", "custom"},
		{"", ""},
	}
	for _, c := range cases {
		got := legendPosReverse(c.in)
		if got != c.want {
			t.Errorf("legendPosReverse(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractTitleXML_StrRef(t *testing.T) {
	title := &chartTitleXML{
		Tx: chartTxXML{
			StrRef: &chartStrRefXML{F: "Sheet1!$A$1"},
		},
	}
	got := extractTitleXML(title)
	if got != "Sheet1!$A$1" {
		t.Errorf("extractTitleXML(strRef) = %q, want Sheet1!$A$1", got)
	}
}

func TestExtractTitleXML_Rich(t *testing.T) {
	title := &chartTitleXML{
		Tx: chartTxXML{
			Rich: &chartRichXML{
				P: []chartPXML{
					{R: &chartRXML{Text: "Chart Title"}},
				},
			},
		},
	}
	got := extractTitleXML(title)
	if got != "Chart Title" {
		t.Errorf("extractTitleXML(rich) = %q, want Chart Title", got)
	}
}

func TestExtractTitleXML_RichMultiParagraph(t *testing.T) {
	title := &chartTitleXML{
		Tx: chartTxXML{
			Rich: &chartRichXML{
				P: []chartPXML{
					{R: nil},
					{R: &chartRXML{Text: "Second"}},
				},
			},
		},
	}
	got := extractTitleXML(title)
	if got != "Second" {
		t.Errorf("extractTitleXML(multi) = %q, want Second", got)
	}
}

func TestExtractTitleXML_Empty(t *testing.T) {
	title := &chartTitleXML{Tx: chartTxXML{}}
	got := extractTitleXML(title)
	if got != "" {
		t.Errorf("extractTitleXML(empty) = %q, want empty", got)
	}
}

func TestExtractLegendXML_WithPos(t *testing.T) {
	v := "b"
	legend := &chartLegendXML{
		LegendPos: &chartAttrStr{Val: &v},
	}
	cl := extractLegendXML(legend)
	if cl == nil {
		t.Fatal("extractLegendXML returned nil")
	}
	if !cl.Show {
		t.Error("expected Show=true")
	}
	if cl.Pos != "bottom" {
		t.Errorf("Pos = %q, want bottom", cl.Pos)
	}
}

func TestExtractLegendXML_NilPos(t *testing.T) {
	legend := &chartLegendXML{}
	cl := extractLegendXML(legend)
	if cl == nil {
		t.Fatal("extractLegendXML returned nil")
	}
	if cl.Pos != "" {
		t.Errorf("Pos = %q, want empty", cl.Pos)
	}
}

func TestDetectChartTypeXML_AllTypes(t *testing.T) {
	pa := &chartPlotAreaXML{AreaChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "area" {
		t.Errorf("AreaChart = %q, want area", ct)
	}

	pa = &chartPlotAreaXML{BarChart: &chartBarXML{BarDir: &chartAttrStr{Val: strPtr("col")}}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "col" {
		t.Errorf("BarChart(col) = %q, want col", ct)
	}

	pa = &chartPlotAreaXML{BubbleChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "bubble" {
		t.Errorf("BubbleChart = %q, want bubble", ct)
	}

	pa = &chartPlotAreaXML{DoughnutChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "doughnut" {
		t.Errorf("DoughnutChart = %q, want doughnut", ct)
	}

	pa = &chartPlotAreaXML{LineChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "line" {
		t.Errorf("LineChart = %q, want line", ct)
	}

	pa = &chartPlotAreaXML{PieChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "pie" {
		t.Errorf("PieChart = %q, want pie", ct)
	}

	pa = &chartPlotAreaXML{RadarChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "radar" {
		t.Errorf("RadarChart = %q, want radar", ct)
	}

	pa = &chartPlotAreaXML{ScatterChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "scatter" {
		t.Errorf("ScatterChart = %q, want scatter", ct)
	}
}

func TestDetectChartTypeXML_3DVariants(t *testing.T) {
	pa := &chartPlotAreaXML{Area3DChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "area" {
		t.Errorf("Area3DChart = %q, want area", ct)
	}

	v := "col"
	pa = &chartPlotAreaXML{Bar3DChart: &chartBarXML{BarDir: &chartAttrStr{Val: &v}}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "col" {
		t.Errorf("Bar3DChart(col) = %q, want col", ct)
	}

	pa = &chartPlotAreaXML{Line3DChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "line" {
		t.Errorf("Line3DChart = %q, want line", ct)
	}

	pa = &chartPlotAreaXML{Pie3DChart: &chartChartsXML{}}
	if ct, ok := detectChartTypeXML(pa); !ok || ct != "pie" {
		t.Errorf("Pie3DChart = %q, want pie", ct)
	}
}

func TestDetectChartTypeXML_None(t *testing.T) {
	pa := &chartPlotAreaXML{}
	_, ok := detectChartTypeXML(pa)
	if ok {
		t.Fatal("expected false for empty plot area")
	}
}

func TestExtractMarkerXML_Full(t *testing.T) {
	sym := "circle"
	size := 6
	m := &chartMarkerXML{
		Symbol: &chartAttrStr{Val: &sym},
		Size:   &chartAttrInt{Val: &size},
	}
	cm := extractMarkerXML(m)
	if cm == nil {
		t.Fatal("extractMarkerXML returned nil")
	}
	if cm.Symbol != "circle" {
		t.Errorf("Symbol = %q, want circle", cm.Symbol)
	}
	if cm.Size != 6 {
		t.Errorf("Size = %v, want 6", cm.Size)
	}
}

func TestExtractMarkerXML_NilFields(t *testing.T) {
	m := &chartMarkerXML{}
	cm := extractMarkerXML(m)
	if cm == nil {
		t.Fatal("extractMarkerXML returned nil")
	}
	if cm.Symbol != "" {
		t.Errorf("expected empty symbol, got %q", cm.Symbol)
	}
}

func TestExtractLineXML_Valid(t *testing.T) {
	spPr := &chartSpPrXML{
		Ln: &chartLnXML{W: 19050},
	}
	cl := extractLineXML(spPr)
	if cl == nil {
		t.Fatal("extractLineXML returned nil")
	}
	if cl.Width != 1.5 {
		t.Errorf("Width = %v, want 1.5", cl.Width)
	}
}

func TestExtractLineXML_NilLn(t *testing.T) {
	spPr := &chartSpPrXML{}
	cl := extractLineXML(spPr)
	if cl != nil {
		t.Fatal("expected nil for nil Ln")
	}
}

func TestExtractLineXML_NoFill(t *testing.T) {
	noFill := "a"
	spPr := &chartSpPrXML{
		Ln: &chartLnXML{NoFill: &noFill, W: 12700},
	}
	cl := extractLineXML(spPr)
	if cl != nil {
		t.Fatal("expected nil for NoFill")
	}
}

func TestExtractLineXML_ZeroWidth(t *testing.T) {
	spPr := &chartSpPrXML{
		Ln: &chartLnXML{W: 0},
	}
	cl := extractLineXML(spPr)
	if cl != nil {
		t.Fatal("expected nil for zero width")
	}
}

func TestExtractFillXML_Valid(t *testing.T) {
	v := "FF4472C4"
	spPr := &chartSpPrXML{
		SolidFill: &chartSolidFillXML{
			SrgbClr: &chartAttrStr{Val: &v},
		},
	}
	cf := extractFillXML(spPr)
	if cf == nil {
		t.Fatal("extractFillXML returned nil")
	}
	if cf.Color != "#4472C4" {
		t.Errorf("Color = %q, want #4472C4", cf.Color)
	}
}

func TestExtractFillXML_8Digit(t *testing.T) {
	v := "FF4472C4"
	spPr := &chartSpPrXML{
		SolidFill: &chartSolidFillXML{
			SrgbClr: &chartAttrStr{Val: &v},
		},
	}
	cf := extractFillXML(spPr)
	if cf == nil {
		t.Fatal("extractFillXML returned nil")
	}
	if cf.Color != "#4472C4" {
		t.Errorf("Color = %q, want #4472C4 (8-digit trimmed)", cf.Color)
	}
}

func TestExtractFillXML_NilSolidFill(t *testing.T) {
	spPr := &chartSpPrXML{}
	cf := extractFillXML(spPr)
	if cf != nil {
		t.Fatal("expected nil for nil SolidFill")
	}
}

func TestExtractFillXML_NilSrgbClr(t *testing.T) {
	spPr := &chartSpPrXML{
		SolidFill: &chartSolidFillXML{},
	}
	cf := extractFillXML(spPr)
	if cf != nil {
		t.Fatal("expected nil for nil SrgbClr")
	}
}

func TestExtractDLblsXML_AllTrue(t *testing.T) {
	d := &chartDLblsXML{
		ShowVal:         &chartBoolAttr{Val: true},
		ShowCatName:     &chartBoolAttr{Val: true},
		ShowSerName:     &chartBoolAttr{Val: true},
		ShowPercent:     &chartBoolAttr{Val: true},
		ShowLeaderLines: &chartBoolAttr{Val: true},
	}
	dl := extractDLblsXML(d)
	if dl == nil {
		t.Fatal("extractDLblsXML returned nil")
	}
	if !dl.ShowVal || !dl.ShowCatName || !dl.ShowSerName || !dl.ShowPercent || !dl.ShowLeaderLn {
		t.Error("expected all true")
	}
}

func TestExtractDLblsXML_AllNil(t *testing.T) {
	d := &chartDLblsXML{}
	dl := extractDLblsXML(d)
	if dl == nil {
		t.Fatal("extractDLblsXML returned nil")
	}
	if dl.ShowVal || dl.ShowCatName || dl.ShowSerName || dl.ShowPercent || dl.ShowLeaderLn {
		t.Error("expected all false")
	}
}

func TestExtractAxisXML_Nil(t *testing.T) {
	if a := extractAxisXML(nil); a != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestExtractAxisXML_Full(t *testing.T) {
	v := "My Title"
	ax := &chartAxsXML{
		Title:          &chartTitleXML{Tx: chartTxXML{Rich: &chartRichXML{P: []chartPXML{{R: &chartRXML{Text: v}}}}}},
		MajorGridlines: &chartGridlinesXML{},
		MinorGridlines: &chartGridlinesXML{},
		Scaling: &chartScalingXML{
			Min: &chartAttrFloat{Val: float64Ptr(0)},
			Max: &chartAttrFloat{Val: float64Ptr(100)},
		},
		NumFmt: &chartNumFmtXML{FormatCode: "#,##0"},
	}
	a := extractAxisXML(ax)
	if a == nil {
		t.Fatal("extractAxisXML returned nil")
	}
	if a.Title != "My Title" {
		t.Errorf("Title = %q, want My Title", a.Title)
	}
	if !a.MajorGridLines {
		t.Error("expected MajorGridLines=true")
	}
	if !a.MinorGridLines {
		t.Error("expected MinorGridLines=true")
	}
	if a.Minimum == nil || *a.Minimum != 0 {
		t.Errorf("Minimum = %v, want 0", a.Minimum)
	}
	if a.Maximum == nil || *a.Maximum != 100 {
		t.Errorf("Maximum = %v, want 100", a.Maximum)
	}
	if a.NumFmt != "#,##0" {
		t.Errorf("NumFmt = %q, want #,##0", a.NumFmt)
	}
}

func TestExtractAxisXML_Empty(t *testing.T) {
	ax := &chartAxsXML{}
	a := extractAxisXML(ax)
	if a == nil {
		t.Fatal("extractAxisXML returned nil")
	}
	if a.Title != "" {
		t.Errorf("expected empty title, got %q", a.Title)
	}
}

func strPtr(s string) *string { return &s }
func float64Ptr(f float64) *float64 { return &f }
