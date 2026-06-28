package json2xlsx

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func addChartsToFile(f *excelize.File, wb *Workbook) error {
	if wb.Book == nil {
		return nil
	}

	helperSheet := "_xlsxchart_helper"
	helperRow := 1
	helperCreated := false
	for _, ch := range wb.Book.Charts {
		for i := range ch.Ser {
			name := ch.Ser[i].Name
			if name != "" && !strings.Contains(name, "!") {
				if !helperCreated {
					if _, err := f.NewSheet(helperSheet); err != nil {
						return fmt.Errorf("create helper sheet: %w", err)
					}
					if err := f.SetSheetVisible(helperSheet, false); err != nil {
						return fmt.Errorf("hide helper sheet: %w", err)
					}
					helperCreated = true
				}
				cell, _ := excelize.CoordinatesToCellName(1, helperRow)
				if err := f.SetCellValue(helperSheet, cell, name); err != nil {
					return fmt.Errorf("write series name to helper sheet: %w", err)
				}
				ch.Ser[i].Name = fmt.Sprintf("'%s'!%s", helperSheet, cell)
				helperRow++
			}
		}
	}

	for _, ch := range wb.Book.Charts {
		ct, err := chartTypeFromString(ch.Ct)
		if err != nil {
			return fmt.Errorf("chart %q: %w", ch.ID, err)
		}
		ec := excelize.Chart{
			Type:   ct,
			Series: toExcelizeSeriesList(ch.Ser),
		}
		if ch.Title != nil && ch.Title.Tx != "" {
			ec.Title = []excelize.RichTextRun{{Text: ch.Title.Tx}}
		}
		if ch.Legend != nil {
			ec.Legend = excelize.ChartLegend{
				Position: ch.Legend.Pos,
			}
		}
		if ch.XAxis != nil {
			ec.XAxis = toExcelizeAxis(*ch.XAxis)
		}
		if ch.YAxis != nil {
			ec.YAxis = toExcelizeAxis(*ch.YAxis)
		}
		if ch.Dim != nil {
			ec.Dimension = excelize.ChartDimension{
				Width:  uint(ch.Dim.W),
				Height: uint(ch.Dim.H),
			}
		}
		if ch.Plot != nil {
			ec.VaryColors = &ch.Plot.VaryColors
			ec.ShowBlanksAs = ch.Plot.ShowBlanksAs
		}
		for _, s := range ch.Ser {
			if s.DLbls != nil {
				ec.PlotArea = excelize.ChartPlotArea{
					ShowVal:         s.DLbls.ShowVal,
					ShowCatName:     s.DLbls.ShowCatName,
					ShowSerName:     s.DLbls.ShowSerName,
					ShowPercent:     s.DLbls.ShowPercent,
					ShowLeaderLines: s.DLbls.ShowLeaderLn,
				}
				break
			}
		}
		switch ch.Mode {
		case "", "embedded":
			ec.Format = chartGraphicOptions(ch.Dim)
			if err := f.AddChart(ch.Sheet, ch.Anchor, &ec); err != nil {
				return fmt.Errorf("chart %q: add chart: %w", ch.ID, err)
			}
		case "chartSheet":
			if err := f.AddChartSheet(ch.Sheet, &ec); err != nil {
				return fmt.Errorf("chart %q: add chart sheet: %w", ch.ID, err)
			}
		default:
			return fmt.Errorf("chart %q: unknown mode %q", ch.ID, ch.Mode)
		}
	}
	return nil
}

func chartTypeFromString(ct string) (excelize.ChartType, error) {
	switch ct {
	case "col":
		return excelize.Col, nil
	case "bar":
		return excelize.Bar, nil
	case "line":
		return excelize.Line, nil
	case "area":
		return excelize.Area, nil
	case "pie":
		return excelize.Pie, nil
	case "doughnut":
		return excelize.Doughnut, nil
	case "scatter":
		return excelize.Scatter, nil
	case "radar":
		return excelize.Radar, nil
	default:
		return 0, fmt.Errorf("unsupported chart type %q", ct)
	}
}

func chartGraphicOptions(dim *ChartDim) excelize.GraphicOptions {
	opts := excelize.GraphicOptions{ScaleX: 1.0, ScaleY: 1.0}
	if dim != nil {
		if dim.Sx > 0 {
			opts.ScaleX = dim.Sx
		}
		if dim.Sy > 0 {
			opts.ScaleY = dim.Sy
		}
		opts.OffsetX = int(dim.OffX)
		opts.OffsetY = int(dim.OffY)
	}
	return opts
}

func toExcelizeSeriesList(series []ChartSeries) []excelize.ChartSeries {
	result := make([]excelize.ChartSeries, len(series))
	for i, s := range series {
		es := excelize.ChartSeries{
			Name:       s.Name,
			Categories: s.Cat,
			Values:     s.Val,
		}
		if s.Line != nil {
			es.Line = excelize.ChartLine{Width: s.Line.Width}
		}
		if s.Fill != nil && s.Fill.Color != "" {
			es.Fill = excelize.Fill{Color: []string{s.Fill.Color}}
		}
		if s.Marker != nil {
			es.Marker = excelize.ChartMarker{
				Symbol: s.Marker.Symbol,
				Size:   int(s.Marker.Size),
			}
		}
		result[i] = es
	}
	return result
}

func toExcelizeAxis(axis ChartAxis) excelize.ChartAxis {
	ea := excelize.ChartAxis{}
	if axis.Title != "" {
		ea.Title = []excelize.RichTextRun{{Text: axis.Title}}
	}
	ea.ReverseOrder = axis.ReverseOrder
	ea.MajorGridLines = axis.MajorGridLines
	ea.MinorGridLines = axis.MinorGridLines
	if axis.NumFmt != "" {
		ea.NumFmt = excelize.ChartNumFmt{CustomNumFmt: axis.NumFmt}
	}
	if axis.Minimum != nil {
		ea.Minimum = axis.Minimum
	}
	if axis.Maximum != nil {
		ea.Maximum = axis.Maximum
	}
	if axis.MajorUnit != nil {
		ea.MajorUnit = *axis.MajorUnit
	}
	return ea
}
