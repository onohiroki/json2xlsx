package json2xlsx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func convertWorkbook(wb *Workbook, out io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	if err := validateWorkbook(wb.Sheets, wb); err != nil {
		return err
	}

	styleMap, err := buildStyles(f, wb.Styles)
	if err != nil {
		return fmt.Errorf("build styles: %w", err)
	}

	var warnings int
	if err := createSheets(f, wb.Sheets, styleMap, wb.Styles, &warnings); err != nil {
		return err
	}

	if err := addChartsToFile(f, wb); err != nil {
		return err
	}

	if err := f.Write(out); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}

	if warnings > 0 {
		return fmt.Errorf("conversion completed with %d warning(s)", warnings)
	}
	return nil
}

func validateWorkbook(sheets []Sheet, wb *Workbook) error {
	if len(sheets) == 0 {
		hasCharts := wb.Book != nil && len(wb.Book.Charts) > 0
		if !hasCharts {
			return fmt.Errorf("no sheets found in JSON input: expected a \"sheets\" array, \"cells\" object, or a \"book\" wrapper with \"sheets\"")
		}
		return nil
	}
	hasData := false
	for _, sh := range sheets {
		if len(sh.Cells) > 0 || len(sh.Rows) > 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		hasCharts := wb.Book != nil && len(wb.Book.Charts) > 0
		if !hasCharts {
			return fmt.Errorf("no valid cell data found in JSON input: each sheet must contain a \"cells\" object (e.g. \"A1\": {...}) or a \"rows\" array")
		}
	}
	return nil
}

func createSheets(f *excelize.File, sheets []Sheet, styleMap map[int]int, styles []Style, warnings *int) error {
	defaultName := f.GetSheetName(0)
	firstAssigned := false

	for i, sh := range sheets {
		name := sh.Name
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}

		if !firstAssigned {
			if name != defaultName {
				if err := f.SetSheetName(defaultName, name); err != nil {
					return fmt.Errorf("rename sheet: %w", err)
				}
			}
			firstAssigned = true
		} else {
			if _, err := f.NewSheet(name); err != nil {
				return fmt.Errorf("new sheet %q: %w", name, err)
			}
		}

		if err := writeSheet(f, name, sh, styleMap, styles, warnings); err != nil {
			return fmt.Errorf("write sheet %q: %w", name, err)
		}
	}
	return nil
}

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

func writeSheet(f *excelize.File, name string, sh Sheet, styleMap map[int]int, styles []Style, warnings *int) error {
	for r, row := range sh.Rows {
		for c, v := range row {
			axis, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(name, axis, v); err != nil {
				return err
			}
		}
	}

	for axis, cell := range sh.Cells {
		if err := setCell(f, name, axis, cell, styleMap, styles); err != nil {
			var we *WarningError
			if errors.As(err, &we) {
				fmt.Fprintf(os.Stderr, "warning: %v\n", we.Err)
				*warnings++
			} else {
				return fmt.Errorf("set cell %s: %w", axis, err)
			}
		}
	}

	for _, c := range sh.Cols {
		if c.Col == "" || c.Width <= 0 {
			continue
		}
		w := c.Width
		if w > 255 {
			w = 255
		}
		if err := f.SetColWidth(name, c.Col, c.Col, w); err != nil {
			return err
		}
	}

	for _, rd := range sh.RowDims {
		if rd.Row <= 0 || rd.Height <= 0 {
			continue
		}
		h := rd.Height
		if h > 409 {
			h = 409
		}
		if err := f.SetRowHeight(name, rd.Row, h); err != nil {
			return err
		}
	}

	for _, m := range sh.Merges {
		parts := strings.Split(m.Range, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid merge range: %q", m.Range)
		}
		if err := f.MergeCell(name, parts[0], parts[1]); err != nil {
			return err
		}
	}

	return nil
}

func setCell(f *excelize.File, sheet, axis string, c Cell, styleMap map[int]int, styles []Style) error {
	switch c.T {
	case "f":
		if c.F == "" {
			return fmt.Errorf("cell %s: type=f but formula empty", axis)
		}
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
		}
		if err := f.SetCellFormula(sheet, axis, c.F); err != nil {
			return err
		}
	case "b":
		bv, ok := c.V.(bool)
		if !ok {
			return fmt.Errorf("cell %s: type=b but value not bool", axis)
		}
		if err := f.SetCellBool(sheet, axis, bv); err != nil {
			return err
		}
	case "n":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	case "s", "":
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
		}
	case "d":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cell %s: unknown type %q", axis, c.T)
	}

	if c.L != nil {
		target, tooltip := parseLink(c.L)
		if target != "" {
			opts := []excelize.HyperlinkOpts{}
			if tooltip != "" {
				opts = append(opts, excelize.HyperlinkOpts{Tooltip: &tooltip})
			}
			if err := f.SetCellHyperLink(sheet, axis, target, "External", opts...); err != nil {
				return err
			}
		}
	}

	if c.S != 0 {
		if baseIdx, ok := styleMap[c.S]; ok {
			styleID := baseIdx
			if c.Z != "" {
				var mergeErr error
				styleID, mergeErr = mergeStyleWithNumFmt(f, styles, c.S, c.Z)
				if mergeErr != nil {
					return mergeErr
				}
			}
			if err := f.SetCellStyle(sheet, axis, axis, styleID); err != nil {
				return err
			}
		} else if c.Z != "" {
			id, err := f.NewStyle(&excelize.Style{CustomNumFmt: &c.Z})
			if err != nil {
				return err
			}
			if err := f.SetCellStyle(sheet, axis, axis, id); err != nil {
				return err
			}
		} else {
			return &WarningError{Err: fmt.Errorf("cell %s: style id %d not defined in styles", axis, c.S)}
		}
	} else if c.Z != "" {
		id, err := f.NewStyle(&excelize.Style{CustomNumFmt: &c.Z})
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, axis, axis, id); err != nil {
			return err
		}
	}

	return nil
}

func mergeStyleWithNumFmt(f *excelize.File, styles []Style, styleID int, numFmt string) (int, error) {
	for i := range styles {
		if styles[i].ID == styleID {
			es, err := toExcelizeStyle(styles[i])
			if err != nil {
				return 0, err
			}
			es.CustomNumFmt = &numFmt
			return f.NewStyle(es)
		}
	}
	return f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
}

func parseLink(l interface{}) (target, tooltip string) {
	switch v := l.(type) {
	case string:
		return v, ""
	case map[string]interface{}:
		if t, ok := v["target"].(string); ok {
			target = t
		}
		if t, ok := v["tooltip"].(string); ok {
			tooltip = t
		}
	}
	return
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
