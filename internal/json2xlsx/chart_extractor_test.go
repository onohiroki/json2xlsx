package json2xlsx

import (
	"testing"
)

func TestResolveRelPath_Absolute(t *testing.T) {
	got := resolveRelPath("xl/workbook.xml", "/xl/charts/chart1.xml")
	if got != "xl/charts/chart1.xml" {
		t.Errorf("absolute: got %q, want xl/charts/chart1.xml", got)
	}
}

func TestResolveRelPath_Relative(t *testing.T) {
	got := resolveRelPath("xl/workbook.xml", "charts/chart1.xml")
	if got != "xl/charts/chart1.xml" {
		t.Errorf("relative: got %q, want xl/charts/chart1.xml", got)
	}
}

func TestResolveRelPath_ParentDir(t *testing.T) {
	got := resolveRelPath("xl/worksheets/sheet1.xml", "../charts/chart1.xml")
	if got != "xl/charts/chart1.xml" {
		t.Errorf("parent: got %q, want xl/charts/chart1.xml", got)
	}
}

func TestResolveRelPath_SameDir(t *testing.T) {
	got := resolveRelPath("xl/drawings/drawing1.xml", "image1.png")
	if got != "xl/drawings/image1.png" {
		t.Errorf("same dir: got %q, want xl/drawings/image1.png", got)
	}
}

func TestResolveRelPath_DeepNesting(t *testing.T) {
	got := resolveRelPath("a/b/c/d.xml", "../../x/y.xml")
	if got != "a/x/y.xml" {
		t.Errorf("deep: got %q, want a/x/y.xml", got)
	}
}

func TestChartsheetRelsPath(t *testing.T) {
	got := chartsheetRelsPath("xl/chartsheets/sheet1.xml")
	want := "xl/chartsheets/_rels/sheet1.xml.rels"
	if got != want {
		t.Errorf("chartsheetRelsPath = %q, want %q", got, want)
	}
}

func TestChartsheetRelsPath_Deep(t *testing.T) {
	got := chartsheetRelsPath("xl/foo/bar/baz.xml")
	want := "xl/foo/bar/_rels/baz.xml.rels"
	if got != want {
		t.Errorf("chartsheetRelsPath = %q, want %q", got, want)
	}
}

func TestChartRefInGraphicFrame_Nil(t *testing.T) {
	_, ok := chartRefInGraphicFrame(nil)
	if ok {
		t.Fatal("expected false for nil")
	}
}

func TestChartRefInGraphicFrame_NoGraphic(t *testing.T) {
	gf := &xdrGraphicFrame{}
	_, ok := chartRefInGraphicFrame(gf)
	if ok {
		t.Fatal("expected false for no Graphic")
	}
}

func TestChartRefInGraphicFrame_NoGraphicData(t *testing.T) {
	gf := &xdrGraphicFrame{
		Graphic: &xdrGraphic{},
	}
	_, ok := chartRefInGraphicFrame(gf)
	if ok {
		t.Fatal("expected false for no GraphicData")
	}
}

func TestChartRefInGraphicFrame_WrongURI(t *testing.T) {
	gf := &xdrGraphicFrame{
		Graphic: &xdrGraphic{
			GraphicData: &xdrGraphicData{
				URI: "http://example.com/other",
			},
		},
	}
	_, ok := chartRefInGraphicFrame(gf)
	if ok {
		t.Fatal("expected false for wrong URI")
	}
}

func TestChartRefInGraphicFrame_NoChart(t *testing.T) {
	gf := &xdrGraphicFrame{
		Graphic: &xdrGraphic{
			GraphicData: &xdrGraphicData{
				URI: "http://schemas.openxmlformats.org/drawingml/2006/chart",
			},
		},
	}
	_, ok := chartRefInGraphicFrame(gf)
	if ok {
		t.Fatal("expected false for no Chart ref")
	}
}

func TestChartRefInGraphicFrame_Valid(t *testing.T) {
	gf := &xdrGraphicFrame{
		Graphic: &xdrGraphic{
			GraphicData: &xdrGraphicData{
				URI:   "http://schemas.openxmlformats.org/drawingml/2006/chart",
				Chart: &xdrChartRef{ID: "rId1"},
			},
		},
	}
	id, ok := chartRefInGraphicFrame(gf)
	if !ok {
		t.Fatal("expected true for valid frame")
	}
	if id != "rId1" {
		t.Errorf("id = %q, want rId1", id)
	}
}

func TestCollectAllSeriesXML_FromChartsXML(t *testing.T) {
	pa := &chartPlotAreaXML{
		LineChart: &chartChartsXML{
			Ser: []chartSerXML{
				{Tx: &chartTxXML{StrRef: &chartStrRefXML{F: "S1"}}},
			},
		},
	}
	ser := collectAllSeriesXML(pa)
	if len(ser) != 1 {
		t.Fatalf("expected 1 series, got %d", len(ser))
	}
}

func TestCollectAllSeriesXML_FromBar(t *testing.T) {
	pa := &chartPlotAreaXML{
		BarChart: &chartBarXML{
			Ser: []chartSerXML{
				{Tx: &chartTxXML{StrRef: &chartStrRefXML{F: "B1"}}},
			},
		},
	}
	ser := collectAllSeriesXML(pa)
	if len(ser) != 1 {
		t.Fatalf("expected 1 series, got %d", len(ser))
	}
}

func TestCollectAllSeriesXML_None(t *testing.T) {
	pa := &chartPlotAreaXML{}
	ser := collectAllSeriesXML(pa)
	if ser != nil {
		t.Fatal("expected nil for empty plot area")
	}
}

func TestCollectAllSeriesXML_Priority(t *testing.T) {
	// area が先にマッチし、bar は無視される
	pa := &chartPlotAreaXML{
		AreaChart: &chartChartsXML{
			Ser: []chartSerXML{
				{Tx: &chartTxXML{StrRef: &chartStrRefXML{F: "A1"}}},
			},
		},
		BarChart: &chartBarXML{
			Ser: []chartSerXML{
				{Tx: &chartTxXML{StrRef: &chartStrRefXML{F: "B1"}}},
			},
		},
	}
	ser := collectAllSeriesXML(pa)
	if len(ser) != 1 {
		t.Fatalf("expected 1 series, got %d", len(ser))
	}
	if ser[0].Tx.StrRef.F != "A1" {
		t.Errorf("expected area chart series, got %q", ser[0].Tx.StrRef.F)
	}
}

func TestExtractSingleSeriesXML_Basic(t *testing.T) {
	s := chartSerXML{
		Tx:  &chartTxXML{StrRef: &chartStrRefXML{F: "MyName"}},
		Cat: &chartCatXML{StrRef: &chartStrRefXML{F: "Sheet1!$A$1:$A$2"}},
		Val: &chartValXML{NumRef: &chartNumRefXML{F: "Sheet1!$B$1:$B$2"}},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Name != "MyName" {
		t.Errorf("Name = %q, want MyName", cs.Name)
	}
	if cs.Cat != "Sheet1!$A$1:$A$2" {
		t.Errorf("Cat = %q, want Sheet1!$A$1:$A$2", cs.Cat)
	}
	if cs.Val != "Sheet1!$B$1:$B$2" {
		t.Errorf("Val = %q, want Sheet1!$B$1:$B$2", cs.Val)
	}
}

func TestExtractSingleSeriesXML_Scatter(t *testing.T) {
	xVal := "Sheet1!$A$1:$A$2"
	yVal := "Sheet1!$B$1:$B$2"
	s := chartSerXML{
		XVal: &chartCatXML{StrRef: &chartStrRefXML{F: xVal}},
		YVal: &chartValXML{NumRef: &chartNumRefXML{F: yVal}},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Cat != xVal {
		t.Errorf("Cat = %q, want %q (from xVal)", cs.Cat, xVal)
	}
	if cs.Val != yVal {
		t.Errorf("Val = %q, want %q (from yVal)", cs.Val, yVal)
	}
	if cs.XVal == nil || *cs.XVal != xVal {
		t.Errorf("XVal = %v, want %q", cs.XVal, xVal)
	}
	if cs.YVal == nil || *cs.YVal != yVal {
		t.Errorf("YVal = %v, want %q", cs.YVal, yVal)
	}
}

func TestExtractSingleSeriesXML_ScatterWithCatVal(t *testing.T) {
	// cat と xVal 両方ある場合、cat が優先される
	s := chartSerXML{
		Cat:  &chartCatXML{StrRef: &chartStrRefXML{F: "Sheet1!$C$1:$C$2"}},
		XVal: &chartCatXML{StrRef: &chartStrRefXML{F: "Sheet1!$A$1:$A$2"}},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Cat != "Sheet1!$C$1:$C$2" {
		t.Errorf("Cat should keep original, got %q", cs.Cat)
	}
}

func TestExtractSingleSeriesXML_RichTextName(t *testing.T) {
	s := chartSerXML{
		Tx: &chartTxXML{
			Rich: &chartRichXML{
				P: []chartPXML{
					{R: &chartRXML{Text: "Rich Name"}},
				},
			},
		},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Name != "Rich Name" {
		t.Errorf("Name = %q, want Rich Name", cs.Name)
	}
}

func TestExtractSingleSeriesXML_EmptyTx(t *testing.T) {
	s := chartSerXML{}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Name != "" {
		t.Errorf("expected empty name, got %q", cs.Name)
	}
}

func TestExtractSingleSeriesXML_Marker(t *testing.T) {
	sym := "diamond"
	size := 7
	s := chartSerXML{
		Marker: &chartMarkerXML{
			Symbol: &chartAttrStr{Val: &sym},
			Size:   &chartAttrInt{Val: &size},
		},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Marker == nil {
		t.Fatal("Marker is nil")
	}
	if cs.Marker.Symbol != "diamond" {
		t.Errorf("Marker.Symbol = %q, want diamond", cs.Marker.Symbol)
	}
	if cs.Marker.Size != 7 {
		t.Errorf("Marker.Size = %v, want 7", cs.Marker.Size)
	}
}

func TestExtractSingleSeriesXML_Fill(t *testing.T) {
	v := "FF4472C4"
	s := chartSerXML{
		SpPr: &chartSpPrXML{
			SolidFill: &chartSolidFillXML{
				SrgbClr: &chartAttrStr{Val: &v},
			},
		},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.Fill == nil {
		t.Fatal("Fill is nil")
	}
	if cs.Fill.Color != "#4472C4" {
		t.Errorf("Fill.Color = %q, want #4472C4", cs.Fill.Color)
	}
}

func TestExtractSingleSeriesXML_DLbls(t *testing.T) {
	s := chartSerXML{
		DLbls: &chartDLblsXML{
			ShowVal: &chartBoolAttr{Val: true},
		},
	}
	cs := extractSingleSeriesXML(s, nil)
	if cs.DLbls == nil {
		t.Fatal("DLbls is nil")
	}
	if !cs.DLbls.ShowVal {
		t.Error("expected ShowVal=true")
	}
}
