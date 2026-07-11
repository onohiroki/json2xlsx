package json2xlsx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// extractAutoFilterFromSheet はワークシート XML から autoFilter の範囲を抽出する．
func extractAutoFilterFromSheet(f *excelize.File, sheetName string) (string, error) {
	sheetIdx, err := f.GetSheetIndex(sheetName)
	if err != nil || sheetIdx < 0 {
		return "", nil
	}
	wsPath := fmt.Sprintf("xl/worksheets/sheet%d.xml", sheetIdx+1)
	raw, ok := f.Pkg.Load(wsPath)
	if !ok {
		return "", nil
	}
	var ws struct {
		AutoFilter *struct {
			Ref string `xml:"ref,attr"`
		} `xml:"http://schemas.openxmlformats.org/spreadsheetml/2006/main autoFilter"`
	}
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&ws); err != nil {
		return "", nil
	}
	if ws.AutoFilter == nil {
		return "", nil
	}
	return ws.AutoFilter.Ref, nil
}

// condFmtOptToRule は excelize.ConditionalFormatOptions を ConditionalRule に変換する．
func condFmtOptToRule(opt excelize.ConditionalFormatOptions) ConditionalRule {
	return ConditionalRule{
		Type:           opt.Type,
		Criteria:       opt.Criteria,
		Value:          opt.Value,
		MinValue:       opt.MinValue,
		MidValue:       opt.MidValue,
		MaxValue:       opt.MaxValue,
		MinType:        opt.MinType,
		MidType:        opt.MidType,
		MaxType:        opt.MaxType,
		MinColor:       addHash(opt.MinColor),
		MidColor:       addHash(opt.MidColor),
		MaxColor:       addHash(opt.MaxColor),
		BarColor:       addHash(opt.BarColor),
		BarBorderColor: addHash(opt.BarBorderColor),
		BarDirection:   opt.BarDirection,
		BarOnly:        opt.BarOnly,
		BarSolid:       opt.BarSolid,
		IconStyle:      opt.IconStyle,
		ReverseIcons:   opt.ReverseIcons,
		IconsOnly:      opt.IconsOnly,
		AboveAverage:   boolPtr(opt.AboveAverage),
		Percent:        opt.Percent,
		StopIfTrue:     opt.StopIfTrue,
	}
}

// boolPtr は bool のポインタを返す．
func boolPtr(b bool) *bool { return &b }

// parseDxfs は styles.xml から微分書式 (dxf) の一覧を抽出する．
func parseDxfs(f *excelize.File) ([]dxfStyleXML, error) {
	raw, ok := f.Pkg.Load("xl/styles.xml")
	if !ok {
		return nil, nil
	}
	var styles struct {
		Dxfs *struct {
			Dxfs []dxfStyleXML `xml:"dxf"`
		} `xml:"dxfs"`
	}
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&styles); err != nil {
		return nil, nil
	}
	if styles.Dxfs == nil {
		return nil, nil
	}
	return styles.Dxfs.Dxfs, nil
}

// dxfToStyle は dxf 要素を Style に変換する．
// 書式情報がない空の dxf の場合は nil を返す．
func dxfToStyle(dxf *dxfStyleXML) *Style {
	if dxf == nil {
		return nil
	}
	if dxfIsEmpty(dxf) {
		return nil
	}
	var s Style

	if dxf.Font != nil {
		f := &Font{}
		if dxf.Font.Bold != nil {
			f.Bold = dxf.Font.Bold.Val
		}
		if dxf.Font.Italic != nil {
			f.Italic = dxf.Font.Italic.Val
		}
		if dxf.Font.Size != nil {
			f.Size = dxf.Font.Size.Val
		}
		if dxf.Font.Name != nil {
			f.Name = dxf.Font.Name.Val
		}
		if dxf.Font.Color != nil {
			f.Color = dxfColorToHash(dxf.Font.Color)
		}
		s.Font = f
	}

	if dxf.NumFmt != nil {
		s.NumFmt = dxf.NumFmt.FormatCode
	}

	if dxf.Alignment != nil {
		s.Alignment = &Alignment{
			Horizontal: dxf.Alignment.Horizontal,
			Vertical:   dxf.Alignment.Vertical,
			WrapText:   dxf.Alignment.WrapText,
		}
	}

	if dxf.Fill != nil && dxf.Fill.PatternFill != nil {
		pf := dxf.Fill.PatternFill
		fill := &Fill{
			Type:    pf.PatternType,
			Pattern: 1,
		}
		// 条件付き書式の fill は bgColor に色が設定される
		switch {
		case pf.FgColor != nil && pf.FgColor.RGB != "":
			fill.Color = []string{dxfColorToHash(pf.FgColor)}
		case pf.BgColor != nil && pf.BgColor.RGB != "":
			fill.Color = []string{dxfColorToHash(pf.BgColor)}
		}
		if len(fill.Color) > 0 {
			s.Fill = fill
		}
	}

	if dxf.Border != nil {
		addBorder := func(edge *dxfBorderEdgeXML, side string) {
			if edge == nil || edge.Style == "" {
				return
			}
			b := Border{Style: edge.Style, Side: side}
			if edge.Color != nil {
				b.Color = dxfColorToHash(edge.Color)
			}
			s.Border = append(s.Border, b)
		}
		addBorder(dxf.Border.Left, "left")
		addBorder(dxf.Border.Right, "right")
		addBorder(dxf.Border.Top, "top")
		addBorder(dxf.Border.Bottom, "bottom")
	}

	return &s
}

// dxfColorToHash は dxfColorXML の RGB を #RRGGBB 形式に変換する．
func dxfColorToHash(c *dxfColorXML) string {
	if c == nil || c.RGB == "" {
		return ""
	}
	s := c.RGB
	if len(s) == 8 && strings.ToUpper(s[:2]) == "FF" {
		s = s[2:]
	}
	return "#" + s
}

// dxfIsEmpty は dxf に書式情報がない場合に true を返す．
func dxfIsEmpty(dxf *dxfStyleXML) bool {
	return dxf.Font == nil && dxf.NumFmt == nil && dxf.Fill == nil && dxf.Alignment == nil && dxf.Border == nil
}

// condFmtTypeUsesDxf は指定された条件付き書式タイプが dxf (差分書式) を使用するかを返す．
// colorScale, dataBar, iconSet は組み込みの色/アイコン設定を使うため dxf を使用しない．
// 型名は json2xlsx の JSON 型名 (cell, formula, top, bottom, aboveAvg, unique, duplicate, date) を想定．
func condFmtTypeUsesDxf(t string) bool {
	switch t {
	case "cell", "formula", "top", "bottom", "aboveAvg", "aboveAverage", "unique", "duplicate", "date":
		return true
	default:
		return false
	}
}

// dxfStyleXML は dxf 要素の XML 構造．
type dxfStyleXML struct {
	Font      *dxfFontXML      `xml:"font"`
	NumFmt    *dxfNumFmtXML    `xml:"numFmt"`
	Fill      *dxfFillXML      `xml:"fill"`
	Alignment *dxfAlignXML     `xml:"alignment"`
	Border    *dxfBorderXML    `xml:"border"`
}

type dxfAttrBool struct {
	Val bool `xml:"val,attr"`
}

type dxfAttrFloat struct {
	Val float64 `xml:"val,attr"`
}

type dxfAttrString struct {
	Val string `xml:"val,attr"`
}

type dxfFontXML struct {
	Bold   *dxfAttrBool   `xml:"b"`
	Italic *dxfAttrBool   `xml:"i"`
	Size   *dxfAttrFloat  `xml:"sz"`
	Color  *dxfColorXML   `xml:"color"`
	Name   *dxfAttrString `xml:"name"`
}

type dxfNumFmtXML struct {
	FormatCode string `xml:"formatCode,attr"`
}

type dxfFillXML struct {
	PatternFill *dxfPatternFillXML `xml:"patternFill"`
}

type dxfPatternFillXML struct {
	PatternType string      `xml:"patternType,attr"`
	FgColor     *dxfColorXML `xml:"fgColor"`
	BgColor     *dxfColorXML `xml:"bgColor"`
}

type dxfAlignXML struct {
	Horizontal string `xml:"horizontal,attr"`
	Vertical   string `xml:"vertical,attr"`
	WrapText   bool   `xml:"wrapText,attr"`
}

type dxfBorderXML struct {
	Left   *dxfBorderEdgeXML `xml:"left"`
	Right  *dxfBorderEdgeXML `xml:"right"`
	Top    *dxfBorderEdgeXML `xml:"top"`
	Bottom *dxfBorderEdgeXML `xml:"bottom"`
}

type dxfBorderEdgeXML struct {
	Style string       `xml:"style,attr"`
	Color *dxfColorXML `xml:"color"`
}

type dxfColorXML struct {
	Auto    bool    `xml:"auto,attr"`
	RGB     string  `xml:"rgb,attr"`
	Indexed int     `xml:"indexed,attr"`
	Theme   *int    `xml:"theme,attr"`
	Tint    float64 `xml:"tint,attr"`
}
