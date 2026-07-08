package json2xlsx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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

	if sh.Freeze != nil && (sh.Freeze.Row > 0 || sh.Freeze.Col > 0) {
		topLeftCell, err := excelize.CoordinatesToCellName(sh.Freeze.Col+1, sh.Freeze.Row+1)
		if err != nil {
			return err
		}
		panes := excelize.Panes{
			Freeze:      true,
			XSplit:      sh.Freeze.Col,
			YSplit:      sh.Freeze.Row,
			TopLeftCell: topLeftCell,
			ActivePane:  "bottomRight",
		}
		if err := f.SetPanes(name, &panes); err != nil {
			return err
		}
	}

	return nil
}

var dotFuncRe = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_.]*(?:\.[A-Za-z][A-Za-z0-9_]*)+)\s*\(`)

var plainFutureFuncRe = regexp.MustCompile(`\b(IFS|SWITCH|MINIFS|MAXIFS|DAYS|CONCAT|TEXTJOIN|DATEDIF|XLOOKUP)\s*\(`)

// addFutureFuncPrefix adds the _xlfn. prefix to future functions in a formula.
// Functions with a dot in their name (e.g. QUARTILE.INC) are matched by regex;
// dotless newer functions (e.g. IFS, DAYS) are matched by name.
// Without this prefix, Excel 2010-2016 returns #NAME? for these functions.
func addFutureFuncPrefix(formula string) string {
	result := dotFuncRe.ReplaceAllString(formula, `_xlfn.$1(`)
	result = plainFutureFuncRe.ReplaceAllString(result, `_xlfn.$1(`)
	return result
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
		formula := addFutureFuncPrefix(c.F)
		if err := f.SetCellFormula(sheet, axis, formula); err != nil {
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
