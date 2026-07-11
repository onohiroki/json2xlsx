package json2xlsx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

const sparklineExtURI = "{05C60535-1F16-4fd2-B633-F4F36F0B64E0}"

const sparklineNS = "http://schemas.microsoft.com/office/spreadsheetml/2009/9/main"
const mainNS = "http://schemas.microsoft.com/office/excel/2006/main"

type extLstRawXML struct {
	Ext []extRawXML `xml:"ext"`
}

type extRawXML struct {
	URI     string `xml:"uri,attr"`
	Content string `xml:",innerxml"`
}

type sparklineGroupsXML struct {
	XMLName xml.Name            `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main sparklineGroups"`
	Groups  []sparklineGroupXML `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main sparklineGroup"`
}

type sparklineGroupXML struct {
	Type                string             `xml:"type,attr,omitempty"`
	ManualMax           int                `xml:"manualMax,attr,omitempty"`
	ManualMin           int                `xml:"manualMin,attr,omitempty"`
	LineWeight          float64            `xml:"lineWeight,attr,omitempty"`
	DateAxis            bool               `xml:"dateAxis,attr,omitempty"`
	DisplayEmptyCellsAs string             `xml:"displayEmptyCellsAs,attr,omitempty"`
	Markers             bool               `xml:"markers,attr,omitempty"`
	High                bool               `xml:"high,attr,omitempty"`
	Low                 bool               `xml:"low,attr,omitempty"`
	First               bool               `xml:"first,attr,omitempty"`
	Last                bool               `xml:"last,attr,omitempty"`
	Negative            bool               `xml:"negative,attr,omitempty"`
	DisplayXAxis        bool               `xml:"displayXAxis,attr,omitempty"`
	DisplayHidden       bool               `xml:"displayHidden,attr,omitempty"`
	MinAxisType         string             `xml:"minAxisType,attr,omitempty"`
	MaxAxisType         string             `xml:"maxAxisType,attr,omitempty"`
	RightToLeft         bool               `xml:"rightToLeft,attr,omitempty"`
	ColorSeries         *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorSeries"`
	ColorNegative       *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorNegative"`
	ColorAxis           *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorAxis"`
	ColorMarkers        *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorMarkers"`
	ColorFirst          *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorFirst"`
	ColorLast           *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorLast"`
	ColorHigh           *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorHigh"`
	ColorLow            *colorRawXML       `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main colorLow"`
	Sparklines          sparklinesRawXML   `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main sparklines"`
}

type colorRawXML struct {
	Auto    bool    `xml:"auto,attr"`
	RGB     string  `xml:"rgb,attr"`
	Indexed int     `xml:"indexed,attr"`
	Theme   *int    `xml:"theme,attr"`
	Tint    float64 `xml:"tint,attr"`
}

type sparklinesRawXML struct {
	Sparkline []sparklineRawXML `xml:"http://schemas.microsoft.com/office/spreadsheetml/2009/9/main sparkline"`
}

type sparklineRawXML struct {
	F     string `xml:"http://schemas.microsoft.com/office/excel/2006/main f"`
	Sqref string `xml:"http://schemas.microsoft.com/office/excel/2006/main sqref"`
}

// extractSparklinesFromSheet はシートXMLからスパークラインを抽出する．
func extractSparklinesFromSheet(f *excelize.File, sheetName string) ([]Sparkline, error) {
	sheetIdx, err := f.GetSheetIndex(sheetName)
	if err != nil || sheetIdx < 0 {
		return nil, nil
	}
	wsPath := fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetIdx+1)
	raw, ok := f.Pkg.Load(wsPath)
	if !ok {
		return nil, nil
	}

	var ws struct {
		ExtLst *extLstRawXML `xml:"http://schemas.openxmlformats.org/spreadsheetml/2006/main extLst"`
	}
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&ws); err != nil {
		return nil, nil
	}
	if ws.ExtLst == nil {
		return nil, nil
	}

	var extContent string
	for _, e := range ws.ExtLst.Ext {
		if e.URI == sparklineExtURI {
			extContent = e.Content
			break
		}
	}
	if extContent == "" {
		return nil, nil
	}

	// extContent は <x14:sparklineGroups xmlns:xm="...">...</x14:sparklineGroups>
	// x14 名前空間宣言を追加してからデコードする．
	fixedContent := strings.Replace(extContent, "<x14:sparklineGroups", `<x14:sparklineGroups xmlns:x14="`+sparklineNS+`"`, 1)
	var groups sparklineGroupsXML
	dec2 := xml.NewDecoder(strings.NewReader(fixedContent))
	if err := dec2.Decode(&groups); err != nil {
		return nil, fmt.Errorf("decode sparkline groups: %w", err)
	}

	var result []Sparkline
	for _, g := range groups.Groups {
		base := Sparkline{
			Type:          g.Type,
			Weight:        g.LineWeight,
			DateAxis:      g.DateAxis,
			Markers:       g.Markers,
			High:          g.High,
			Low:           g.Low,
			First:         g.First,
			Last:          g.Last,
			Negative:      g.Negative,
			Hidden:        g.DisplayHidden,
			Reverse:       g.RightToLeft,
			Axis:          g.DisplayXAxis,
			EmptyCells:    g.DisplayEmptyCellsAs,
			Max:           g.ManualMax,
			Min:           g.ManualMin,
			SeriesColor:   colorRGB(g.ColorSeries),
			NegativeColor: colorRGB(g.ColorNegative),
			MarkersColor:  colorRGB(g.ColorMarkers),
			FirstColor:    colorRGB(g.ColorFirst),
			LastColor:     colorRGB(g.ColorLast),
			HighColor:     colorRGB(g.ColorHigh),
			LowColor:      colorRGB(g.ColorLow),
		}
		for _, s := range g.Sparklines.Sparkline {
			sl := base
			sl.Location = s.Sqref
			sl.Range = s.F
			result = append(result, sl)
		}
	}
	return result, nil
}

func colorRGB(c *colorRawXML) string {
	if c == nil || c.RGB == "" {
		return ""
	}
	s := c.RGB
	// excelize は RGB 属性を "FF" + hex_color (AARRGGBB) の形式で保存する．
	// alpha が FF の場合は取り除いて #RRGGBB 形式にする．
	if len(s) == 8 && strings.ToUpper(s[:2]) == "FF" {
		s = s[2:]
	}
	return "#" + s
}
