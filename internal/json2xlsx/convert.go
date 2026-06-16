package json2xlsx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// WarningError は非 fatal な警告を表す。
// 処理は継続され、XLSX 出力は行われるが、exit code は非零になる。
type WarningError struct {
	Err error
}

func (w *WarningError) Error() string { return w.Err.Error() }
func (w *WarningError) Unwrap() error { return w.Err }

// Convert は JSON を読み込み、XLSX を out に書き出す。
// defaultSheetName が空でない場合、シート名未指定時のデフォルトとして使う。
func Convert(r io.Reader, out io.Writer, defaultSheetName string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var wb Workbook
	if err := json.Unmarshal(data, &wb); err != nil {
		if schemaErr := ValidateJSON(data); schemaErr != nil {
			return fmt.Errorf("%v\n\n%v", err, schemaErr)
		}
		return fmt.Errorf("parse json: %w", err)
	}

	if err := convertWorkbook(&wb, out, defaultSheetName); err != nil {
		if schemaErr := ValidateJSON(data); schemaErr != nil {
			return fmt.Errorf("%v\n\n%v", err, schemaErr)
		}
		return err
	}
	return nil
}

func convertWorkbook(wb *Workbook, out io.Writer, defaultSheetName string) error {
	f := excelize.NewFile()
	defer f.Close()

	// シート一覧を組み立てる (book ラッパー → 配列形式 → 単一シート の優先順)
	var sheets []Sheet
	styles := wb.Styles

	if wb.Book != nil {
		// book ラッパー形式: map を配列に展開
		for name, sh := range wb.Book.Sheets {
			sh.Name = name
			sheets = append(sheets, sh)
		}
		if len(wb.Book.Styles) > 0 {
			styles = wb.Book.Styles
		}
		// book.Charts は後で処理
	} else if len(wb.Sheets) > 0 {
		sheets = wb.Sheets
	} else {
		// 単一シート形式
		sheets = []Sheet{{
			Name:    wb.Name,
			Cells:   wb.Cells,
			Rows:    wb.Rows,
			Cols:    wb.Cols,
			RowDims: wb.RowDims,
			Merges:  wb.Merges,
		}}
	}

	// スタイル ID -> excelize スタイル ID マッピング
	styleMap, err := buildStyles(f, styles)
	if err != nil {
		return fmt.Errorf("build styles: %w", err)
	}

	// 既定シート名 ("Sheet1") を最初のシートに割り当て
	defaultName := f.GetSheetName(0)
	firstAssigned := false

	var warnings int

	for i, sh := range sheets {
		name := sh.Name
		if name == "" {
			if i == 0 && defaultSheetName != "" {
				name = defaultSheetName
			} else {
				name = fmt.Sprintf("Sheet%d", i+1)
			}
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

		if err := writeSheet(f, name, sh, styleMap, styles, &warnings); err != nil {
			return fmt.Errorf("write sheet %q: %w", name, err)
		}
	}

	// チャート変換 (book ラッパー形式のみ)
	if wb.Book != nil {
		// すべての系列名のうち、リテラル文字列（!を含まない）を補助シートのセルに書き込み、
		// セル参照に置き換える。これにより excelize が有効な strRef を出力する。
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
			// 各系列の dLbls（最初の系列）→ PlotArea
			for _, s := range ch.Ser {
				if s.DLbls != nil {
					ec.PlotArea = excelize.ChartPlotArea{
						ShowVal:      s.DLbls.ShowVal,
						ShowCatName:  s.DLbls.ShowCatName,
						ShowSerName:  s.DLbls.ShowSerName,
						ShowPercent:  s.DLbls.ShowPercent,
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
				// AddChartSheet は anchor/offset を受け付けず、シート名のみ
				if err := f.AddChartSheet(ch.Sheet, &ec); err != nil {
					return fmt.Errorf("chart %q: add chart sheet: %w", ch.ID, err)
				}
			default:
				return fmt.Errorf("chart %q: unknown mode %q", ch.ID, ch.Mode)
			}
		}
	}

	// 警告があっても XLSX 出力は常に行う
	if err := f.Write(out); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}

	if warnings > 0 {
		return fmt.Errorf("conversion completed with %d warning(s)", warnings)
	}
	return nil
}

func writeSheet(f *excelize.File, name string, sh Sheet, styleMap map[int]int, styles []Style, warnings *int) error {
	// AoA 形式 (rows) の展開: 1 行目 = 1 行目に配置
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

	// Cell Object 形式 (WarningError は非 fatal として継続)
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

	// 列幅 (Excel 制限: 0 < width <= 255)
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

	// 行高 (Excel 制限: 0 < height <= 409)
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

	// マージ
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

	// ハイパーリンク
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

	// スタイル適用 (z 単独指定にも対応)
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

// chartTypeFromString は chart-json-spec.md の ct 文字列を Excelize の ChartType に変換する。
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

// chartGraphicOptions は ChartDim から GraphicOptions を生成する。
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

// toExcelizeSeriesList は ChartSeries のスライスを Excelize の ChartSeries スライスに変換する。
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

// toExcelizeAxis は ChartAxis を Excelize の ChartAxis に変換する。
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

