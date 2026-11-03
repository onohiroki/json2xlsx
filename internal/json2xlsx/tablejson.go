package json2xlsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// TableSpec は高レベルな table JSON のトップレベル構造．
type TableSpec struct {
	Tables []TableInput `json:"tables" jsonschema:"one or more tables; each table becomes one worksheet"`
}

// TableInput は 1 シート分の高レベルな表定義．
type TableInput struct {
	Name         string                `json:"name,omitempty" jsonschema:"worksheet name (default: Sheet1, Sheet2, ...)"`
	Header       []any                 `json:"header,omitempty" jsonschema:"optional header row; data starts on row 2 when present"`
	Rows         [][]any               `json:"rows" jsonschema:"data rows; each row is an array of cell values"`
	Columns      []TableColumn         `json:"columns,omitempty" jsonschema:"optional per-column settings; index 0 is column A"`
	Styles       map[string]TableStyle `json:"styles,omitempty" jsonschema:"named styles referenced by headerStyle, bodyStyle, columns[].style, or cell style"`
	HeaderStyle  string                `json:"headerStyle,omitempty" jsonschema:"name of the style applied to the header row"`
	BodyStyle    string                `json:"bodyStyle,omitempty" jsonschema:"name of the style applied to data cells"`
	FreezeHeader bool                  `json:"freezeHeader,omitempty" jsonschema:"freeze the header row when true"`
	AutoFilter   bool                  `json:"autoFilter,omitempty" jsonschema:"enable autofilter on the header and data range when true"`
	Merges       []string              `json:"merges,omitempty" jsonschema:"merged ranges such as A1:D1"`
}

// TableColumn は列単位の設定．
type TableColumn struct {
	Width   *float64 `json:"width,omitempty" jsonschema:"column width (0 < width <= 255)"`
	Format  string   `json:"format,omitempty" jsonschema:"number format code applied to this column"`
	Align   string   `json:"align,omitempty" jsonschema:"horizontal alignment: left, center, or right"`
	Style   string   `json:"style,omitempty" jsonschema:"name of a style in styles"`
	Formula string   `json:"formula,omitempty" jsonschema:"formula template applied to every data row; {r} is replaced by the row number"`
}

// TableStyle は名前付きスタイル．SheetJS の Style から id を除いた形．
type TableStyle struct {
	Fill      *Fill      `json:"fill,omitempty" jsonschema:"cell fill"`
	Border    []Border   `json:"border,omitempty" jsonschema:"cell borders; omit side to apply to all four sides"`
	Font      *Font      `json:"font,omitempty" jsonschema:"font settings"`
	Alignment *Alignment `json:"alignment,omitempty" jsonschema:"alignment settings"`
	NumFmt    string     `json:"numFmt,omitempty" jsonschema:"number format code"`
}

func (s TableStyle) toStyle() Style {
	return Style{
		Fill:      cloneFill(s.Fill),
		Border:    append([]Border(nil), s.Border...),
		Font:      cloneFont(s.Font),
		Alignment: cloneAlignment(s.Alignment),
		NumFmt:    s.NumFmt,
	}
}

func cloneFill(f *Fill) *Fill {
	if f == nil {
		return nil
	}
	c := *f
	c.Color = append([]string(nil), f.Color...)
	return &c
}

func cloneFont(f *Font) *Font {
	if f == nil {
		return nil
	}
	c := *f
	return &c
}

func cloneAlignment(a *Alignment) *Alignment {
	if a == nil {
		return nil
	}
	c := *a
	return &c
}

// TableValidationError は table JSON の意味検証で見つかった問題を表す．
type TableValidationError struct {
	Issues []JSONIssue
}

func (e *TableValidationError) Error() string {
	if err := formatIssues(e.Issues); err != nil {
		return err.Error()
	}
	return "invalid table JSON"
}

type tableCell struct {
	V      any
	F      string
	Format string
	Style  string
	Align  string
	T      string
}

type tableBuilder struct {
	registry *tableStyleRegistry
}

type tableStyleRegistry struct {
	byKey  map[string]int
	styles []Style
}

func newTableStyleRegistry() *tableStyleRegistry {
	return &tableStyleRegistry{byKey: map[string]int{}}
}

func (r *tableStyleRegistry) register(s Style) int {
	s.ID = 0
	keyBytes, _ := json.Marshal(s)
	key := string(keyBytes)
	if id, ok := r.byKey[key]; ok {
		return id
	}
	id := len(r.styles) + 1
	s.ID = id
	r.byKey[key] = id
	r.styles = append(r.styles, s)
	return id
}

// ParseTableSpec は table JSON をパースし，意味検証の問題も返す．
// JSON として壊れている場合は err を返す．
func ParseTableSpec(data []byte) (*TableSpec, []JSONIssue, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	dec.DisallowUnknownFields()
	var spec TableSpec
	if err := dec.Decode(&spec); err != nil {
		return nil, nil, fmt.Errorf("parse table JSON: %w", err)
	}
	issues := validateTableSpec(&spec)
	return &spec, issues, nil
}

// ValidateTableJSON は table JSON を検証し，構造化された問題一覧を返す．
func ValidateTableJSON(data []byte) ([]JSONIssue, error) {
	_, issues, err := ParseTableSpec(data)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// UnmarshalTableJSON は table JSON を Workbook に変換する．
func UnmarshalTableJSON(data []byte) (*Workbook, error) {
	spec, issues, err := ParseTableSpec(data)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return nil, &TableValidationError{Issues: issues}
	}
	return spec.Workbook()
}

// Validate は TableSpec の意味検証を行い，問題一覧を返す．
func (s *TableSpec) Validate() []JSONIssue {
	return validateTableSpec(s)
}

// Workbook は TableSpec を Workbook に変換する．
func (s *TableSpec) Workbook() (*Workbook, error) {
	return tableSpecToWorkbook(s)
}

func tableSpecToWorkbook(spec *TableSpec) (*Workbook, error) {
	b := &tableBuilder{registry: newTableStyleRegistry()}
	sheets := make([]Sheet, 0, len(spec.Tables))
	for i, t := range spec.Tables {
		sh, err := b.buildSheet(t, i)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sh)
	}
	wb := &Workbook{Sheets: sheets, Styles: b.registry.styles}
	normalizeWorkbook(wb)
	return wb, nil
}

func (b *tableBuilder) buildSheet(t TableInput, index int) (Sheet, error) {
	name := t.Name
	if name == "" {
		name = fmt.Sprintf("Sheet%d", index+1)
	}

	headerRows := 0
	if len(t.Header) > 0 {
		headerRows = 1
	}
	totalRows := headerRows + len(t.Rows)
	ncols := len(t.Header)
	if len(t.Columns) > ncols {
		ncols = len(t.Columns)
	}
	for _, row := range t.Rows {
		if len(row) > ncols {
			ncols = len(row)
		}
	}
	if ncols == 0 {
		return Sheet{}, fmt.Errorf("table %q has no columns", name)
	}
	if totalRows == 0 {
		return Sheet{}, fmt.Errorf("table %q has no header or rows", name)
	}

	rows := make([][]any, totalRows)
	for i := range rows {
		rows[i] = make([]any, ncols)
	}
	cells := map[string]Cell{}
	cols := make([]ColInfo, 0, ncols)

	for j := range t.Columns {
		if t.Columns[j].Width == nil {
			continue
		}
		width := *t.Columns[j].Width
		if width <= 0 {
			continue
		}
		colName, err := columnName(j)
		if err != nil {
			return Sheet{}, err
		}
		cols = append(cols, ColInfo{Col: colName, Width: width})
	}

	// ヘッダ行
	if headerRows == 1 {
		for j, v := range t.Header {
			if v == nil {
				continue
			}
			col := columnAt(t.Columns, j)
			styleID := b.effectiveStyle(t, true, col, tableCell{})
			axis, err := cellName(j, 0)
			if err != nil {
				return Sheet{}, err
			}
			value := normalizeTableValue(v)
			if styleID == 0 {
				rows[0][j] = value
				continue
			}
			cells[axis] = Cell{T: inferCellType(value), V: value, S: styleID}
		}
	}

	// データ行
	for i, row := range t.Rows {
		sheetRow := headerRows + i
		for j := 0; j < ncols; j++ {
			var raw any
			if j < len(row) {
				raw = row[j]
			}
			col := columnAt(t.Columns, j)
			cellObj, isObj, err := parseTableCellObject(raw)
			if err != nil {
				return Sheet{}, fmt.Errorf("table %q row %d column %s: %w", name, i+1, mustColumnName(j), err)
			}

			formula := ""
			switch {
			case isObj && cellObj.F != "":
				formula = cellObj.F
			case col.Formula != "":
				formula = expandFormula(col.Formula, sheetRow+1)
			}
			if formula == "" && !isObj {
				if s, ok := raw.(string); ok && strings.HasPrefix(s, "=") {
					formula = strings.TrimPrefix(s, "=")
				}
			}

			value := raw
			if isObj {
				value = cellObj.V
				if formula != "" && cellObj.V == nil {
					value = nil
				}
			} else if formula != "" {
				// 列テンプレートの場合，行の値はキャッシュ値として使える．
				if isNumericValue(raw) {
					value = normalizeTableValue(raw)
				} else {
					value = nil
				}
			}
			value = normalizeTableValue(value)

			align := col.Align
			if isObj && cellObj.Align != "" {
				align = cellObj.Align
			}
			format := col.Format
			if isObj && cellObj.Format != "" {
				format = cellObj.Format
			}
			cellForStyle := tableCell{Format: format, Align: align}
			if isObj && cellObj.Style != "" {
				cellForStyle.Style = cellObj.Style
			}
			styleID := b.effectiveStyle(t, false, col, cellForStyle)

			axis, err := cellName(j, sheetRow)
			if err != nil {
				return Sheet{}, err
			}

			if formula != "" {
				c := Cell{T: "f", F: formula, V: value, S: styleID}
				cells[axis] = c
				rows[sheetRow][j] = nil
				continue
			}

			t := inferCellType(value)
			if isObj && cellObj.T != "" {
				t = cellObj.T
			}
			if styleID == 0 {
				rows[sheetRow][j] = value
				continue
			}
			cells[axis] = Cell{T: t, V: value, S: styleID}
			rows[sheetRow][j] = nil
		}
	}

	sh := Sheet{
		Name:  name,
		Rows:  rows,
		Cells: cells,
		Cols:  cols,
	}
	for _, m := range t.Merges {
		sh.Merges = append(sh.Merges, Merge{Range: strings.ToUpper(m)})
	}
	if t.FreezeHeader && headerRows == 1 {
		sh.Freeze = &FreezePane{Row: 1}
	}
	if t.AutoFilter && headerRows == 1 {
		lastCol, _ := columnName(ncols - 1)
		sh.AutoFilter = fmt.Sprintf("A1:%s%d", lastCol, totalRows)
	}
	return sh, nil
}

func (b *tableBuilder) effectiveStyle(t TableInput, header bool, col TableColumn, cell tableCell) int {
	var base *TableStyle
	styleName := t.BodyStyle
	if header {
		styleName = t.HeaderStyle
	}
	if styleName != "" {
		if s, ok := t.Styles[styleName]; ok {
			base = &s
		}
	}
	if col.Style != "" {
		if s, ok := t.Styles[col.Style]; ok {
			base = &s
		}
	}
	if cell.Style != "" {
		if s, ok := t.Styles[cell.Style]; ok {
			base = &s
		}
	}

	format := col.Format
	if cell.Format != "" {
		format = cell.Format
	}
	align := col.Align
	if cell.Align != "" {
		align = cell.Align
	}
	if base == nil && format == "" && align == "" {
		return 0
	}

	var style Style
	if base != nil {
		style = base.toStyle()
	}
	if format != "" {
		style.NumFmt = format
	}
	if align != "" {
		if style.Alignment == nil {
			style.Alignment = &Alignment{}
		} else {
			style.Alignment = cloneAlignment(style.Alignment)
		}
		style.Alignment.Horizontal = align
	}
	return b.registry.register(style)
}

func columnAt(cols []TableColumn, i int) TableColumn {
	if i < len(cols) {
		return cols[i]
	}
	return TableColumn{}
}

func columnName(index int) (string, error) {
	name, err := excelize.ColumnNumberToName(index + 1)
	if err != nil {
		return "", fmt.Errorf("column index %d: %w", index, err)
	}
	return name, nil
}

func mustColumnName(index int) string {
	name, err := columnName(index)
	if err != nil {
		return "?"
	}
	return name
}

func cellName(col, row int) (string, error) {
	name, err := excelize.CoordinatesToCellName(col+1, row+1)
	if err != nil {
		return "", fmt.Errorf("cell at row %d column %d: %w", row+1, col+1, err)
	}
	return name, nil
}

func expandFormula(tmpl string, row int) string {
	f := strings.TrimPrefix(tmpl, "=")
	return strings.ReplaceAll(f, "{r}", strconv.Itoa(row))
}

func inferCellType(v any) string {
	switch v.(type) {
	case nil:
		return ""
	case bool:
		return "b"
	case string:
		return "s"
	case int, int64, float64:
		return "n"
	default:
		return ""
	}
}

func isNumericValue(v any) bool {
	switch v.(type) {
	case int, int64, float64, json.Number:
		return true
	default:
		return false
	}
}

func normalizeTableValue(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	if f, err := n.Float64(); err == nil {
		return f
	}
	return n.String()
}

func parseTableCellObject(v any) (tableCell, bool, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return tableCell{}, false, nil
	}
	allowed := map[string]bool{
		"v": true, "f": true, "format": true, "style": true, "align": true, "t": true,
	}
	for k := range m {
		if !allowed[k] {
			return tableCell{}, true, fmt.Errorf("unknown cell field %q", k)
		}
	}
	var c tableCell
	c.V = m["v"]
	if c.V != nil && !isScalarValue(c.V) {
		return tableCell{}, true, fmt.Errorf("cell field \"v\" must be a scalar")
	}
	for key, dst := range map[string]*string{
		"f": &c.F, "format": &c.Format, "style": &c.Style, "align": &c.Align, "t": &c.T,
	} {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			return tableCell{}, true, fmt.Errorf("cell field %q must be a string", key)
		}
		*dst = s
	}
	if c.T != "" {
		switch c.T {
		case "s", "n", "b", "f", "d":
		default:
			return tableCell{}, true, fmt.Errorf("unknown cell type %q", c.T)
		}
	}
	if c.T == "f" && c.F == "" {
		return tableCell{}, true, fmt.Errorf("cell type \"f\" requires a formula in \"f\"")
	}
	if c.F != "" && c.T != "" && c.T != "f" {
		return tableCell{}, true, fmt.Errorf("cell formula requires type \"f\" or an empty type")
	}
	return c, true, nil
}

func isScalarValue(v any) bool {
	switch v.(type) {
	case nil, string, bool, int, int64, float64, json.Number:
		return true
	default:
		return false
	}
}

var mergeRangeRe = regexp.MustCompile(`^[A-Za-z]{1,3}[1-9][0-9]*:[A-Za-z]{1,3}[1-9][0-9]*$`)

func validateTableSpec(spec *TableSpec) []JSONIssue {
	var issues []JSONIssue
	add := func(path, msg string) {
		issues = append(issues, JSONIssue{Path: path, Message: msg})
	}

	if len(spec.Tables) == 0 {
		add(".input.tables", "must contain at least one table")
		return issues
	}

	seenNames := map[string]int{}
	for i, t := range spec.Tables {
		path := fmt.Sprintf(".input.tables[%d]", i)
		name := t.Name
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		if err := validateTableSheetName(name); err != nil {
			add(path+".name", err.Error())
		} else if prev, ok := seenNames[strings.ToLower(name)]; ok {
			add(path+".name", fmt.Sprintf("duplicate sheet name %q (already used by tables[%d])", name, prev))
		} else {
			seenNames[strings.ToLower(name)] = i
		}

		if len(t.Header) == 0 && len(t.Rows) == 0 {
			add(path+".rows", "table must have a header or at least one row")
		}
		for j, h := range t.Header {
			if !isScalarValue(h) {
				add(fmt.Sprintf("%s.header[%d]", path, j), "header value must be a scalar")
			}
		}

		for j, col := range t.Columns {
			colPath := fmt.Sprintf("%s.columns[%d]", path, j)
			if col.Width != nil && (*col.Width <= 0 || *col.Width > 255) {
				add(colPath+".width", "width must be greater than 0 and at most 255")
			}
			switch col.Align {
			case "", "left", "center", "right":
			default:
				add(colPath+".align", `align must be "left", "center", or "right"`)
			}
			if col.Style != "" {
				if _, ok := t.Styles[col.Style]; !ok {
					add(colPath+".style", fmt.Sprintf("style %q is not defined in styles", col.Style))
				}
			}
			if strings.ContainsAny(strings.ReplaceAll(col.Formula, "{r}", ""), "{}") {
				add(colPath+".formula", `only the {r} placeholder is supported in formula templates`)
			}
		}

		if t.HeaderStyle != "" {
			if _, ok := t.Styles[t.HeaderStyle]; !ok {
				add(path+".headerStyle", fmt.Sprintf("style %q is not defined in styles", t.HeaderStyle))
			}
		}
		if t.BodyStyle != "" {
			if _, ok := t.Styles[t.BodyStyle]; !ok {
				add(path+".bodyStyle", fmt.Sprintf("style %q is not defined in styles", t.BodyStyle))
			}
		}
		if t.FreezeHeader && len(t.Header) == 0 {
			add(path+".freezeHeader", "freezeHeader requires a header")
		}
		if t.AutoFilter && len(t.Header) == 0 {
			add(path+".autoFilter", "autoFilter requires a header")
		}
		for j, m := range t.Merges {
			if !mergeRangeRe.MatchString(m) {
				add(fmt.Sprintf("%s.merges[%d]", path, j), fmt.Sprintf("invalid merge range %q; expected A1:B2", m))
			}
		}

		for r, row := range t.Rows {
			for c, v := range row {
				cellPath := fmt.Sprintf("%s.rows[%d][%d]", path, r, c)
				if _, isObj, err := parseTableCellObject(v); err != nil {
					add(cellPath, err.Error())
					continue
				} else if !isObj && !isScalarValue(v) {
					add(cellPath, "cell value must be a scalar or a cell object")
				}
				cellObj, isObj, _ := parseTableCellObject(v)
				if isObj && cellObj.Style != "" {
					if _, ok := t.Styles[cellObj.Style]; !ok {
						add(cellPath+".style", fmt.Sprintf("style %q is not defined in styles", cellObj.Style))
					}
				}
				if isObj && cellObj.Align != "" {
					switch cellObj.Align {
					case "left", "center", "right":
					default:
						add(cellPath+".align", `align must be "left", "center", or "right"`)
					}
				}
			}
		}
	}
	return issues
}

func validateTableSheetName(name string) error {
	if name == "" {
		return fmt.Errorf("sheet name must not be empty")
	}
	if utf8.RuneCountInString(name) > 31 {
		return fmt.Errorf("sheet name %q is longer than 31 characters", name)
	}
	if strings.ContainsAny(name, `[]:*?/\`) {
		return fmt.Errorf("sheet name %q contains an invalid character ([]:*?/\\)", name)
	}
	if strings.HasPrefix(name, "'") || strings.HasSuffix(name, "'") {
		return fmt.Errorf("sheet name %q must not start or end with an apostrophe", name)
	}
	if strings.EqualFold(name, "History") {
		return fmt.Errorf("sheet name %q is reserved by Excel", name)
	}
	return nil
}
