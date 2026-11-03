package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/onohiroki/json2xlsx/internal/json2xlsx"
)

// InputSpec は convert / validate / preview で共通の入力指定．
// table / sheetjs / data / input_path のうち exactly one を指定する．
type InputSpec struct {
	Table     *json2xlsx.TableSpec `json:"table,omitempty" jsonschema:"high-level table JSON; use this for simple tables"`
	SheetJS   any                  `json:"sheetjs,omitempty" jsonschema:"SheetJS-style workbook JSON object (pass an object, not a JSON string); accepts {name,cells}, {sheets:[...]}, or {SheetNames,Sheets}"`
	Data      any                  `json:"data,omitempty" jsonschema:"data JSON array or object (pass an array/object, not a JSON string): 2D array, array of objects, or map-of-arrays"`
	InputPath string               `json:"input_path,omitempty" jsonschema:"path to an existing JSON or XLSX file (must be under an allowed directory; relative paths resolve against the first allowed directory)"`
	Format    string               `json:"format,omitempty" jsonschema:"input format for input_path: auto (default), sheetjs, data, table"`
	BaseDir   string               `json:"base_dir,omitempty" jsonschema:"base directory for resolving relative image paths; defaults to the first allowed directory or the input file directory; relative paths resolve against the first allowed directory"`
}

type convertInput struct {
	InputSpec
	OutputPath string `json:"output_path" jsonschema:"required destination XLSX file path (must be under an allowed directory; relative paths resolve against the first allowed directory; the parent directory must exist)"`
	Overwrite  bool   `json:"overwrite,omitempty" jsonschema:"replace the destination if it already exists"`
	Compute    bool   `json:"compute,omitempty" jsonschema:"evaluate formulas without cached values"`
	AutoFit    bool   `json:"autofit,omitempty" jsonschema:"auto-adjust column widths and apply wrap text"`
}

type convertOutput struct {
	Path      string   `json:"path"`
	Bytes     int64    `json:"bytes"`
	Sheets    []string `json:"sheets"`
	Warnings  []string `json:"warnings,omitempty"`
	ElapsedMS int64    `json:"elapsed_ms"`
}

type validateInput struct {
	InputSpec
}

type validateOutput struct {
	Valid   bool                  `json:"valid"`
	Issues  []json2xlsx.JSONIssue `json:"issues"`
	Summary string                `json:"summary"`
}

type previewInput struct {
	InputSpec
	Mode           string `json:"mode,omitempty" jsonschema:"cell display mode: v (default), f, or both"`
	Range          string `json:"range,omitempty" jsonschema:"rectangular range such as A1:F20"`
	Compact        bool   `json:"compact,omitempty" jsonschema:"suppress column alignment padding"`
	FirstRowHeader bool   `json:"first_row_header,omitempty" jsonschema:"treat the first row as a table header"`
	Compute        bool   `json:"compute,omitempty" jsonschema:"evaluate formulas without cached values"`
	MaxBytes       int    `json:"max_bytes,omitempty" jsonschema:"maximum Markdown bytes to return; capped by the server limit"`
}

type previewOutput struct {
	Markdown  string   `json:"markdown"`
	Truncated bool     `json:"truncated"`
	Bytes     int      `json:"bytes"`
	Warnings  []string `json:"warnings,omitempty"`
}

type inspectInput struct {
	InputPath string `json:"input_path" jsonschema:"required path to an XLSX or workbook JSON file (must be under an allowed directory; relative paths resolve against the first allowed directory)"`
	Sheet     string `json:"sheet,omitempty" jsonschema:"restrict the result to one sheet name"`
}

type inspectOutput struct {
	Info json2xlsx.BookInfo `json:"info"`
}

type toJSONInput struct {
	InputPath  string `json:"input_path" jsonschema:"required path to the source XLSX file (must be under an allowed directory; relative paths resolve against the first allowed directory)"`
	OutputPath string `json:"output_path" jsonschema:"required destination JSON file path (must be under an allowed directory; relative paths resolve against the first allowed directory; the parent directory must exist)"`
	Overwrite  bool   `json:"overwrite,omitempty" jsonschema:"replace the destination if it already exists"`
	DateMode   string `json:"date_mode,omitempty" jsonschema:"date output mode: serial (default), rfc3339, or display"`
	ImageMode  string `json:"image_mode,omitempty" jsonschema:"image output mode: base64 (default) or file"`
}

type toJSONOutput struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type functionsInput struct {
	Search   string `json:"search,omitempty" jsonschema:"case-insensitive substring to search for in function names"`
	Category string `json:"category,omitempty" jsonschema:"exact category name to filter by"`
}

type functionsOutput struct {
	Functions  []json2xlsx.FunctionInfo `json:"functions"`
	Categories []string                 `json:"categories"`
}

func (s *server) addTools(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:  "json2xlsx_convert",
		Title: "Convert JSON to XLSX",
		Description: `Convert SheetJS-style JSON or high-level table JSON into an XLSX file.
Specify exactly one of table, sheetjs, data, or input_path. output_path is required, must be under an allowed directory, and its parent directory must exist.
For simple tables, use table with header/rows/columns. Use json2xlsx_validate first to catch structural issues.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
			Title:           "Convert JSON to XLSX",
		},
	}, s.handleConvert)

	mcp.AddTool(srv, &mcp.Tool{
		Name:  "json2xlsx_validate",
		Title: "Validate json2xlsx input",
		Description: `Validate table JSON, SheetJS JSON, or data JSON without writing any files.
Returns valid=false and an issues array with paths such as .input.tables[0].rows[3]. Fix the issues before calling json2xlsx_convert.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
			Title:         "Validate json2xlsx input",
		},
	}, s.handleValidate)

	mcp.AddTool(srv, &mcp.Tool{
		Name:  "json2xlsx_preview",
		Title: "Preview as Markdown",
		Description: `Render the input as a Markdown table without writing an XLSX file.
Accepts the same inputs as json2xlsx_convert, plus an XLSX path. Use this to check values, formulas, and layout before or after conversion.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
			Title:         "Preview as Markdown",
		},
	}, s.handlePreview)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "json2xlsx_inspect",
		Title:       "Inspect workbook",
		Description: `Summarize an XLSX or JSON file (SheetJS / data / table, auto-detected): sheets, used ranges, cell counts, formulas, tables, charts, pictures, and more.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
			Title:         "Inspect workbook",
		},
	}, s.handleInspect)

	mcp.AddTool(srv, &mcp.Tool{
		Name:  "json2xlsx_to_json",
		Title: "Convert XLSX to JSON",
		Description: `Convert an XLSX file to json2xlsx-compatible JSON on disk.
output_path is required and its parent directory must exist. image_mode defaults to base64 so that no extra image files are written; use file to write images next to the JSON. Existing image files cause IMAGE_EXISTS unless overwrite=true. Images are staged in a temporary directory and only moved into place after the JSON is written.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
			Title:           "Convert XLSX to JSON",
		},
	}, s.handleToJSON)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "json2xlsx_functions",
		Title:       "List supported formula functions",
		Description: `List the formula functions supported by the built-in evaluator. Filter by name substring or exact category.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
			Title:         "List supported formula functions",
		},
	}, s.handleFunctions)
}

func boolPtr(b bool) *bool { return &b }

func fail[T any](err error) (*mcp.CallToolResult, T, error) {
	var zero T
	return nil, zero, err
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func (s *server) handleConvert(ctx context.Context, _ *mcp.CallToolRequest, in convertInput) (res *mcp.CallToolResult, out convertOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[convertOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()
	if err := s.acquire(ctx); err != nil {
		return fail[convertOutput](errf("CANCELLED", "%v", err))
	}
	defer s.release()

	started := time.Now()
	wb, _, baseDir, kind, err := s.loadWorkbook(ctx, in.InputSpec, false)
	if err != nil {
		return fail[convertOutput](err)
	}
	if kind == "xlsx" {
		return fail[convertOutput](errf("INVALID_INPUT", "json2xlsx_convert requires JSON input, not XLSX"))
	}
	if err := s.checkWorkbook(wb); err != nil {
		return fail[convertOutput](err)
	}
	if err := s.checkWorkbookPaths(wb, baseDir); err != nil {
		return fail[convertOutput](err)
	}

	outPath, err := s.checkWritePath(in.OutputPath, in.Overwrite)
	if err != nil {
		return fail[convertOutput](err)
	}

	var warnings []string
	opts := json2xlsx.ConvertOptions{
		EvalFormulas: in.Compute,
		AutoFit:      in.AutoFit,
		BaseDir:      baseDir,
		Warn:         func(msg string) { warnings = append(warnings, msg) },
	}
	var buf bytes.Buffer
	if err := json2xlsx.ConvertWorkbook(wb, &buf, opts); err != nil && !json2xlsx.IsWarningOnly(err) {
		return fail[convertOutput](errf("CONVERSION_FAILED", "%v", err))
	}
	if int64(buf.Len()) > s.opts.MaxOutputBytes {
		return fail[convertOutput](errf("OUTPUT_TOO_LARGE", "generated XLSX is %d bytes; limit is %d bytes", buf.Len(), s.opts.MaxOutputBytes))
	}
	if err := writeFileAtomic(outPath, buf.Bytes(), in.Overwrite); err != nil {
		return fail[convertOutput](err)
	}

	sheets := make([]string, 0, len(wb.Sheets))
	for _, sh := range wb.Sheets {
		sheets = append(sheets, sh.Name)
	}
	out = convertOutput{
		Path:      outPath,
		Bytes:     int64(buf.Len()),
		Sheets:    sheets,
		Warnings:  warnings,
		ElapsedMS: time.Since(started).Milliseconds(),
	}
	text := fmt.Sprintf("Wrote %s (%d bytes, %d sheet(s)).", out.Path, out.Bytes, len(out.Sheets))
	if len(warnings) > 0 {
		text += "\nWarnings:\n- " + strings.Join(warnings, "\n- ")
	}
	return textResult(text), out, nil
}

func (s *server) handleValidate(ctx context.Context, _ *mcp.CallToolRequest, in validateInput) (res *mcp.CallToolResult, out validateOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[validateOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()

	issues := make([]json2xlsx.JSONIssue, 0)
	wb, raw, _, kind, loadErr := s.loadWorkbook(ctx, in.InputSpec, false)
	if loadErr != nil {
		var te *json2xlsx.TableValidationError
		var toolErr *toolError
		switch {
		case errors.As(loadErr, &te):
			issues = append(issues, te.Issues...)
		case errors.As(loadErr, &toolErr):
			if toolErr.Code == "INVALID_INPUT" {
				issues = append(issues, json2xlsx.JSONIssue{Path: ".input", Message: toolErr.Err.Error()})
			} else {
				return fail[validateOutput](loadErr)
			}
		default:
			issues = append(issues, json2xlsx.JSONIssue{Path: ".input", Message: loadErr.Error()})
		}
	}
	if wb != nil {
		if err := s.checkWorkbook(wb); err != nil {
			issues = append(issues, json2xlsx.JSONIssue{Path: ".input", Message: err.Error()})
		}
		if kind == "sheetjs" && len(raw) > 0 {
			if schemaIssues, err := json2xlsx.ValidateJSONIssues(raw); err == nil {
				issues = append(issues, schemaIssues...)
			}
		}
	}
	out = validateOutput{
		Valid:   len(issues) == 0,
		Issues:  issues,
		Summary: fmt.Sprintf("%d issue(s)", len(issues)),
	}
	if out.Valid {
		out.Summary = "valid"
	}
	var text strings.Builder
	text.WriteString(out.Summary)
	for i, iss := range issues {
		if i >= 10 {
			fmt.Fprintf(&text, "\n... and %d more", len(issues)-10)
			break
		}
		fmt.Fprintf(&text, "\n- %s: %s", iss.Path, iss.Message)
	}
	return textResult(text.String()), out, nil
}

func (s *server) handlePreview(ctx context.Context, _ *mcp.CallToolRequest, in previewInput) (res *mcp.CallToolResult, out previewOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[previewOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()
	if err := s.acquire(ctx); err != nil {
		return fail[previewOutput](errf("CANCELLED", "%v", err))
	}
	defer s.release()

	wb, _, _, _, err := s.loadWorkbook(ctx, in.InputSpec, true)
	if err != nil {
		return fail[previewOutput](err)
	}
	if err := s.checkWorkbook(wb); err != nil {
		return fail[previewOutput](err)
	}

	mode := in.Mode
	if mode == "" {
		mode = "v"
	}
	maxBytes := in.MaxBytes
	if maxBytes <= 0 || maxBytes > s.opts.MaxPreviewBytes {
		maxBytes = s.opts.MaxPreviewBytes
	}

	var warnings []string
	opts := json2xlsx.MarkdownOptions{
		Mode:           json2xlsx.MarkdownMode(mode),
		RowIndex:       !in.FirstRowHeader,
		FirstRowHeader: in.FirstRowHeader,
		ExplicitMode:   in.Mode != "",
		EvalFormulas:   in.Compute,
		Compact:        in.Compact,
		Range:          in.Range,
		Warn:           func(msg string) { warnings = append(warnings, msg) },
	}
	var buf bytes.Buffer
	if err := json2xlsx.ToMarkdownWorkbook(wb, &buf, opts); err != nil {
		return fail[previewOutput](errf("PREVIEW_FAILED", "%v", err))
	}
	text, truncated := truncateUTF8(buf.String(), maxBytes)
	if truncated {
		text += "\n\n... (preview truncated)\n"
	}
	out = previewOutput{
		Markdown:  text,
		Truncated: truncated,
		Bytes:     len(text),
		Warnings:  warnings,
	}
	return textResult(out.Markdown), out, nil
}

func (s *server) handleInspect(ctx context.Context, _ *mcp.CallToolRequest, in inspectInput) (res *mcp.CallToolResult, out inspectOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[inspectOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()
	data, _, err := s.readFile(in.InputPath)
	if err != nil {
		return fail[inspectOutput](err)
	}
	var info json2xlsx.BookInfo
	if !isXLSXData(data) {
		format := json2xlsx.DetectJSONFormat(data)
		if format == "table" || format == "data" {
			opts := json2xlsx.ConvertOptions{TableJSON: format == "table", DataJSON: format == "data"}
			wb, err := json2xlsx.ParseWorkbook(data, opts)
			if err != nil {
				return fail[inspectOutput](errf("INSPECT_FAILED", "%v", err))
			}
			info = json2xlsx.BuildBookInfo(wb)
		} else {
			wb, charts, err := json2xlsx.ReadBookInfoSource(bytes.NewReader(data))
			if err != nil {
				return fail[inspectOutput](errf("INSPECT_FAILED", "%v", err))
			}
			info = json2xlsx.BuildBookInfoWithCharts(wb, charts)
		}
	} else {
		wb, charts, err := json2xlsx.ReadBookInfoSource(bytes.NewReader(data))
		if err != nil {
			return fail[inspectOutput](errf("INSPECT_FAILED", "%v", err))
		}
		info = json2xlsx.BuildBookInfoWithCharts(wb, charts)
	}
	if in.Sheet != "" {
		filtered := make([]json2xlsx.SheetInfo, 0, 1)
		for _, sh := range info.Sheets {
			if sh.Name == in.Sheet {
				filtered = append(filtered, sh)
			}
		}
		if len(filtered) == 0 {
			return fail[inspectOutput](errf("SHEET_NOT_FOUND", "sheet %q not found", in.Sheet))
		}
		info.Sheets = filtered
	}
	text := fmt.Sprintf("%d sheet(s), %d cell(s), %d formula(s).", info.Totals.Sheets, info.Totals.Cells, info.Totals.Formulas)
	return textResult(text), inspectOutput{Info: info}, nil
}

func (s *server) handleToJSON(ctx context.Context, _ *mcp.CallToolRequest, in toJSONInput) (res *mcp.CallToolResult, out toJSONOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[toJSONOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()
	if err := s.acquire(ctx); err != nil {
		return fail[toJSONOutput](errf("CANCELLED", "%v", err))
	}
	defer s.release()

	dateMode, err := parseDateMode(in.DateMode)
	if err != nil {
		return fail[toJSONOutput](err)
	}
	imageMode, err := parseImageMode(in.ImageMode)
	if err != nil {
		return fail[toJSONOutput](err)
	}

	data, _, err := s.readFile(in.InputPath)
	if err != nil {
		return fail[toJSONOutput](err)
	}
	outPath, err := s.checkWritePath(in.OutputPath, in.Overwrite)
	if err != nil {
		return fail[toJSONOutput](err)
	}

	var buf bytes.Buffer
	opts := json2xlsx.ToJSONOptions{
		DateMode:        dateMode,
		WrapWithBook:    true,
		ImageMode:       imageMode,
		BaseDir:         filepath.Dir(outPath),
		OverwriteImages: in.Overwrite,
	}
	stager, err := json2xlsx.ToJSONWithOptionsStaged(bytes.NewReader(data), &buf, opts)
	if err != nil {
		var ie *json2xlsx.ImageExistsError
		if errors.As(err, &ie) {
			return fail[toJSONOutput](errf("IMAGE_EXISTS", "%v", err))
		}
		return fail[toJSONOutput](errf("CONVERSION_FAILED", "%v", err))
	}
	if stager != nil {
		defer stager.Cleanup()
	}
	if int64(buf.Len()) > s.opts.MaxOutputBytes {
		return fail[toJSONOutput](errf("OUTPUT_TOO_LARGE", "generated JSON is %d bytes; limit is %d bytes", buf.Len(), s.opts.MaxOutputBytes))
	}
	if stager != nil {
		if err := stager.Commit(); err != nil {
			var ie *json2xlsx.ImageExistsError
			if errors.As(err, &ie) {
				return fail[toJSONOutput](errf("IMAGE_EXISTS", "%v", err))
			}
			return fail[toJSONOutput](errf("IMAGE_WRITE_FAILED", "%v", err))
		}
	}
	if err := writeFileAtomic(outPath, buf.Bytes(), in.Overwrite); err != nil {
		if stager != nil {
			_ = stager.Rollback()
		}
		return fail[toJSONOutput](err)
	}
	if stager != nil {
		if err := stager.Finalize(); err != nil {
			return fail[toJSONOutput](errf("IMAGE_CLEANUP_FAILED", "%v", err))
		}
	}
	out = toJSONOutput{Path: outPath, Bytes: int64(buf.Len())}
	return textResult(fmt.Sprintf("Wrote %s (%d bytes).", out.Path, out.Bytes)), out, nil
}

func (s *server) handleFunctions(ctx context.Context, _ *mcp.CallToolRequest, in functionsInput) (res *mcp.CallToolResult, out functionsOutput, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, out, err = fail[functionsOutput](errf("INTERNAL", "panic: %v", r))
		}
	}()
	categories := json2xlsx.FunctionCategories()
	if in.Category != "" {
		ok := false
		for _, c := range categories {
			if c == in.Category {
				ok = true
				break
			}
		}
		if !ok {
			return fail[functionsOutput](errf("INVALID_INPUT", "unknown category %q; valid categories: %s", in.Category, strings.Join(categories, ", ")))
		}
	}
	out = functionsOutput{
		Functions:  json2xlsx.FilterFunctions(in.Search, in.Category),
		Categories: categories,
	}
	return textResult(fmt.Sprintf("%d function(s).", len(out.Functions))), out, nil
}

func (s *server) loadWorkbook(ctx context.Context, spec InputSpec, allowXLSX bool) (*json2xlsx.Workbook, []byte, string, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, "", "", errf("CANCELLED", "%v", err)
	}
	count := 0
	if spec.Table != nil {
		count++
	}
	if spec.SheetJS != nil {
		count++
	}
	if spec.Data != nil {
		count++
	}
	if spec.InputPath != "" {
		count++
	}
	if count != 1 {
		return nil, nil, "", "", errf("INVALID_INPUT", "provide exactly one of table, sheetjs, data, or input_path")
	}

	switch {
	case spec.Table != nil:
		if data, err := json.Marshal(spec.Table); err == nil && int64(len(data)) > s.opts.MaxInputBytes {
			return nil, nil, "", "table", errf("INPUT_TOO_LARGE", "inline input is %d bytes; limit is %d bytes", len(data), s.opts.MaxInputBytes)
		}
		if issues := spec.Table.Validate(); len(issues) > 0 {
			return nil, nil, "", "table", &json2xlsx.TableValidationError{Issues: issues}
		}
		wb, err := spec.Table.Workbook()
		if err != nil {
			return nil, nil, "", "table", errf("INVALID_INPUT", "%v", err)
		}
		baseDir, err := s.resolveBaseDir(spec.BaseDir)
		if err != nil {
			return nil, nil, "", "table", err
		}
		return wb, nil, baseDir, "table", nil

	case spec.SheetJS != nil:
		value, err := unwrapInlineJSON(spec.SheetJS, "sheetjs")
		if err != nil {
			return nil, nil, "", "sheetjs", err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, nil, "", "sheetjs", errf("INVALID_INPUT", "encode sheetjs input: %v", err)
		}
		if normalized, ok := json2xlsx.NormalizeSheetJSLibraryShape(data); ok {
			data = normalized
		}
		data = json2xlsx.StripSheetJSInternalKeys(data)
		if int64(len(data)) > s.opts.MaxInputBytes {
			return nil, nil, "", "sheetjs", errf("INPUT_TOO_LARGE", "inline input is %d bytes; limit is %d bytes", len(data), s.opts.MaxInputBytes)
		}
		wb, err := json2xlsx.ParseWorkbook(data, json2xlsx.ConvertOptions{})
		if err != nil {
			return nil, data, "", "sheetjs", errf("INVALID_INPUT", "%v", err)
		}
		baseDir, err := s.resolveBaseDir(spec.BaseDir)
		if err != nil {
			return nil, nil, "", "sheetjs", err
		}
		return wb, data, baseDir, "sheetjs", nil

	case spec.Data != nil:
		value, err := unwrapInlineJSON(spec.Data, "data")
		if err != nil {
			return nil, nil, "", "data", err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, nil, "", "data", errf("INVALID_INPUT", "encode data input: %v", err)
		}
		if int64(len(data)) > s.opts.MaxInputBytes {
			return nil, nil, "", "data", errf("INPUT_TOO_LARGE", "inline input is %d bytes; limit is %d bytes", len(data), s.opts.MaxInputBytes)
		}
		wb, err := json2xlsx.ParseWorkbook(data, json2xlsx.ConvertOptions{DataJSON: true})
		if err != nil {
			return nil, data, "", "data", errf("INVALID_INPUT", "%v", err)
		}
		baseDir, err := s.resolveBaseDir(spec.BaseDir)
		if err != nil {
			return nil, nil, "", "data", err
		}
		return wb, data, baseDir, "data", nil

	default:
		data, abs, err := s.readFile(spec.InputPath)
		if err != nil {
			return nil, nil, "", "", err
		}
		baseDir := filepath.Dir(abs)
		if isXLSXData(data) {
			if !allowXLSX {
				return nil, nil, "", "xlsx", errf("INVALID_INPUT", "input_path must be JSON for this tool; use json2xlsx_preview or json2xlsx_inspect for XLSX")
			}
			res, err := json2xlsx.ReadWorkbook(bytes.NewReader(data), false)
			if err != nil {
				return nil, nil, "", "xlsx", errf("INVALID_INPUT", "%v", err)
			}
			return res.Workbook, nil, baseDir, "xlsx", nil
		}

		format := strings.ToLower(strings.TrimSpace(spec.Format))
		if format == "" || format == "auto" {
			format = json2xlsx.DetectJSONFormat(data)
		}
		opts := json2xlsx.ConvertOptions{}
		switch format {
		case "table":
			opts.TableJSON = true
		case "data":
			opts.DataJSON = true
		case "sheetjs":
		default:
			return nil, nil, "", "", errf("INVALID_INPUT", "unknown format %q; use auto, sheetjs, data, or table", spec.Format)
		}
		if format == "sheetjs" {
			if normalized, ok := json2xlsx.NormalizeSheetJSLibraryShape(data); ok {
				data = normalized
			}
			data = json2xlsx.StripSheetJSInternalKeys(data)
		}
		wb, err := json2xlsx.ParseWorkbook(data, opts)
		if err != nil {
			return nil, data, baseDir, format, errf("INVALID_INPUT", "%v", err)
		}
		return wb, data, baseDir, format, nil
	}
}

func (s *server) resolveBaseDir(baseDir string) (string, error) {
	if strings.TrimSpace(baseDir) != "" {
		return s.checkDirPath(baseDir)
	}
	if len(s.roots) == 0 {
		return "", errf("INVALID_INPUT", "no allowed directory is configured")
	}
	return s.roots[0], nil
}

func (s *server) checkWorkbook(wb *json2xlsx.Workbook) error {
	if wb == nil {
		return errf("INVALID_INPUT", "workbook is nil")
	}
	if len(wb.Sheets) > s.opts.MaxSheets {
		return errf("LIMIT_EXCEEDED", "workbook has %d sheets; limit is %d", len(wb.Sheets), s.opts.MaxSheets)
	}
	cells := 0
	for _, sh := range wb.Sheets {
		cells += len(sh.Cells)
		for _, row := range sh.Rows {
			cells += len(row)
			if cells > s.opts.MaxCells {
				return errf("LIMIT_EXCEEDED", "workbook has more than %d cells", s.opts.MaxCells)
			}
		}
	}
	return nil
}

func (s *server) checkWorkbookPaths(wb *json2xlsx.Workbook, baseDir string) error {
	const maxImageBytes = 16 << 20
	const maxTotalImageBytes = 64 << 20
	total := int64(0)
	for _, sh := range wb.Sheets {
		for _, pic := range sh.Pictures {
			if pic.Path != "" {
				resolved := json2xlsx.ResolvePath(pic.Path, baseDir)
				if _, err := s.checkReadPath(resolved); err != nil {
					return errf("PATH_NOT_ALLOWED", "picture path %q: %v", pic.Path, err)
				}
			}
			if pic.Data != "" {
				n := int64(base64.StdEncoding.DecodedLen(len(pic.Data)))
				if n > maxImageBytes {
					return errf("LIMIT_EXCEEDED", "picture data is larger than %d bytes", maxImageBytes)
				}
				total += n
			}
		}
		if sh.Background != nil {
			if sh.Background.Path != "" {
				resolved := json2xlsx.ResolvePath(sh.Background.Path, baseDir)
				if _, err := s.checkReadPath(resolved); err != nil {
					return errf("PATH_NOT_ALLOWED", "background path %q: %v", sh.Background.Path, err)
				}
			}
			if sh.Background.Data != "" {
				n := int64(base64.StdEncoding.DecodedLen(len(sh.Background.Data)))
				if n > maxImageBytes {
					return errf("LIMIT_EXCEEDED", "background image data is larger than %d bytes", maxImageBytes)
				}
				total += n
			}
		}
	}
	if total > maxTotalImageBytes {
		return errf("LIMIT_EXCEEDED", "total embedded image data is larger than %d bytes", maxTotalImageBytes)
	}
	return nil
}

// unwrapInlineJSON は文字列として渡された JSON オブジェクト/配列を受け付ける．
// MCP クライアントによってはネストしたオブジェクトを文字列化して送るため，
// そのような入力でも変換できるようにする．
func unwrapInlineJSON(v any, name string) (any, error) {
	s, ok := v.(string)
	if !ok {
		return v, nil
	}
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, errf("INVALID_INPUT", "%s must not be an empty string", name)
	}
	var out any
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, errf("INVALID_INPUT", "%s must be a JSON object or array, not a plain string: %v", name, err)
	}
	return out, nil
}

func isXLSXData(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], []byte{'P', 'K', 3, 4})
}

func parseDateMode(s string) (json2xlsx.DateMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "serial":
		return json2xlsx.DateModeSerial, nil
	case "rfc3339":
		return json2xlsx.DateModeRFC3339, nil
	case "display":
		return json2xlsx.DateModeDisplay, nil
	default:
		return "", errf("INVALID_INPUT", "date_mode must be serial, rfc3339, or display")
	}
}

func parseImageMode(s string) (json2xlsx.ImageMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "base64":
		return json2xlsx.ImageModeBase64, nil
	case "file":
		return json2xlsx.ImageModeFile, nil
	default:
		return "", errf("INVALID_INPUT", "image_mode must be base64 or file")
	}
}

func truncateUTF8(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s, false
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(s[:cut]) {
		cut--
	}
	return s[:cut], true
}
