package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xuri/excelize/v2"
)

func newTestSession(t *testing.T, dir string) (*mcp.ClientSession, func()) {
	t.Helper()
	return newTestSessionWithOptions(t, Options{
		Version:   "test",
		AllowDirs: []string{dir},
	})
}

func newTestSessionWithOptions(t *testing.T, opts Options) (*mcp.ClientSession, func()) {
	t.Helper()
	if opts.Version == "" {
		opts.Version = "test"
	}
	srv, err := NewServer(opts)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	cleanup := func() {
		_ = cs.Close()
		_ = ss.Close()
	}
	return cs, cleanup
}

func TestListToolsAndResources(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want := map[string]bool{
		"json2xlsx_convert":   false,
		"json2xlsx_validate":  false,
		"json2xlsx_preview":   false,
		"json2xlsx_inspect":   false,
		"json2xlsx_to_json":   false,
		"json2xlsx_functions": false,
	}
	for _, tool := range tools.Tools {
		want[tool.Name] = true
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not listed", name)
		}
	}

	resources, err := cs.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	uris := map[string]bool{}
	for _, r := range resources.Resources {
		uris[r.URI] = true
	}
	for _, uri := range []string{"json2xlsx://schema", "json2xlsx://schema/table", "json2xlsx://skill/table", "json2xlsx://samples/sales"} {
		if !uris[uri] {
			t.Errorf("resource %q not listed", uri)
		}
	}
}

func TestConvertTable(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	outPath := filepath.Join(dir, "out.xlsx")
	args := map[string]any{
		"table": map[string]any{
			"tables": []any{
				map[string]any{
					"name":   "Sales",
					"header": []any{"item", "qty", "price", "amount"},
					"rows": []any{
						[]any{"A", 2, 10},
						[]any{"B", 3, 20},
					},
					"columns": []any{
						map[string]any{"width": 18},
						map[string]any{"format": "#,##0"},
						map[string]any{"format": "#,##0"},
						map[string]any{"format": "#,##0", "formula": "=B{r}*C{r}"},
					},
				},
			},
		},
		"output_path": outPath,
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "json2xlsx_convert", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	realOut, _ := filepath.EvalSymlinks(outPath)
	if structured["path"] != realOut {
		t.Fatalf("structured path = %v, want %v", structured["path"], realOut)
	}
	f, err := excelize.OpenFile(outPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer f.Close()
	if got, _ := f.GetCellValue("Sales", "A1"); got != "item" {
		t.Fatalf("A1 = %q", got)
	}
	if got, _ := f.GetCellFormula("Sales", "D2"); got != "B2*C2" {
		t.Fatalf("D2 formula = %q", got)
	}
}

func TestValidateInvalidTable(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_validate",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{
						"rows":        []any{[]any{1}},
						"headerStyle": "missing",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	if valid, _ := structured["valid"].(bool); valid {
		t.Fatalf("valid = true, want false")
	}
	issues, _ := structured["issues"].([]any)
	if len(issues) == 0 {
		t.Fatalf("issues empty")
	}
}

func TestPreview(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_preview",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{
						"header": []any{"a", "b"},
						"rows":   []any{[]any{1, 2}},
					},
				},
			},
			"first_row_header": true,
			"compact":          true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	md, _ := structured["markdown"].(string)
	if !strings.Contains(md, "| a | b |") {
		t.Fatalf("markdown = %q", md)
	}
}

func TestConvertRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	outPath := filepath.Join(dir, "exists.xlsx")
	if err := os.WriteFile(outPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table":       map[string]any{"tables": []any{map[string]any{"rows": []any{[]any{1}}}}},
			"output_path": outPath,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error")
	}
	if !strings.Contains(contentText(res.Content), "OUTPUT_EXISTS") {
		t.Fatalf("error text = %q", contentText(res.Content))
	}
}

func TestPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table":       map[string]any{"tables": []any{map[string]any{"rows": []any{[]any{1}}}}},
			"output_path": filepath.Join(outside, "escape.xlsx"),
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error")
	}
	if !strings.Contains(contentText(res.Content), "PATH_NOT_ALLOWED") {
		t.Fatalf("error text = %q", contentText(res.Content))
	}
}

func TestReadSchemaResource(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "json2xlsx://schema/table"})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(res.Contents) != 1 || !strings.Contains(res.Contents[0].Text, "table JSON") {
		t.Fatalf("unexpected resource: %+v", res.Contents)
	}

	skill, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "json2xlsx://skill/table"})
	if err != nil {
		t.Fatalf("ReadResource skill: %v", err)
	}
	if len(skill.Contents) != 1 || !strings.Contains(skill.Contents[0].Text, "table-json-writer") {
		t.Fatalf("unexpected skill resource: %+v", skill.Contents)
	}
}

func TestFunctionList(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "json2xlsx_functions",
		Arguments: map[string]any{"search": "sum"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	funcs, _ := structured["functions"].([]any)
	if len(funcs) == 0 {
		t.Fatalf("no functions found")
	}
}

func TestToJSON(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	xlsxPath := filepath.Join(dir, "in.xlsx")
	f := excelize.NewFile()
	if err := f.SetCellValue("Sheet1", "A1", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	jsonPath := filepath.Join(dir, "out.json")
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_to_json",
		Arguments: map[string]any{
			"input_path":  xlsxPath,
			"output_path": jsonPath,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
}

func contentText(content []mcp.Content) string {
	var b strings.Builder
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestValidateInvalidSheetJS(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_validate",
		Arguments: map[string]any{
			"sheetjs": map[string]any{
				"cells": map[string]any{
					"A1": map[string]any{"t": "unknown", "v": 1},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	if valid, _ := structured["valid"].(bool); valid {
		t.Fatalf("valid = true, want false")
	}
}

func TestConvertSheetJSAndData(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	cases := []struct {
		name string
		arg  map[string]any
	}{
		{
			name: "sheetjs",
			arg: map[string]any{
				"sheetjs": map[string]any{"cells": map[string]any{"A1": map[string]any{"t": "s", "v": "hi"}}},
			},
		},
		{
			name: "data",
			arg: map[string]any{
				"data": []any{[]any{"a", "b"}, []any{1, 2}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(dir, tc.name+".xlsx")
			args := map[string]any{"output_path": outPath}
			for k, v := range tc.arg {
				args[k] = v
			}
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "json2xlsx_convert", Arguments: args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			if res.IsError {
				t.Fatalf("tool error: %v", res.Content)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("output: %v", err)
			}
		})
	}
}

func TestPreviewComputeTableFormula(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_preview",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{
						"header": []any{"qty", "price", "amount"},
						"rows":   []any{[]any{2, 10}},
						"columns": []any{
							map[string]any{},
							map[string]any{},
							map[string]any{"formula": "=A{r}*B{r}"},
						},
					},
				},
			},
			"compute":          true,
			"compact":          true,
			"first_row_header": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	structured, _ := res.StructuredContent.(map[string]any)
	md, _ := structured["markdown"].(string)
	if !strings.Contains(md, "20") {
		t.Fatalf("markdown = %q, want computed value 20", md)
	}
}

func TestRelativeOutputPathUsesAllowDir(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table":       map[string]any{"tables": []any{map[string]any{"rows": []any{[]any{1}}}}},
			"output_path": "relative.xlsx",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	if _, err := os.Stat(filepath.Join(dir, "relative.xlsx")); err != nil {
		t.Fatalf("expected output under allow-dir: %v", err)
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.json")
	if err := os.WriteFile(secret, []byte(`{"cells":{"A1":{"t":"s","v":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "json2xlsx_preview",
		Arguments: map[string]any{"input_path": "link.json"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(contentText(res.Content), "PATH_NOT_ALLOWED") {
		t.Fatalf("expected PATH_NOT_ALLOWED, got %v", res.Content)
	}
}

func TestImagePathOutsideRejected(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	img := filepath.Join(outside, "image.png")
	if err := os.WriteFile(img, []byte("not really a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"sheetjs": map[string]any{
				"cells":    map[string]any{"A1": map[string]any{"t": "s", "v": "x"}},
				"pictures": []any{map[string]any{"cell": "A1", "path": img}},
			},
			"output_path": "out.xlsx",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(contentText(res.Content), "PATH_NOT_ALLOWED") {
		t.Fatalf("expected PATH_NOT_ALLOWED, got %v", res.Content)
	}
}

func TestInlineInputSizeLimit(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSessionWithOptions(t, Options{
		Version:       "test",
		AllowDirs:     []string{dir},
		MaxInputBytes: 32,
	})
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{"rows": []any{[]any{"a long value that exceeds the limit"}}},
				},
			},
			"output_path": "out.xlsx",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(contentText(res.Content), "INPUT_TOO_LARGE") {
		t.Fatalf("expected INPUT_TOO_LARGE, got %q", contentText(res.Content))
	}
}

func TestCellLimit(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSessionWithOptions(t, Options{
		Version:   "test",
		AllowDirs: []string{dir},
		MaxCells:  2,
	})
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{"rows": []any{[]any{1, 2, 3}}},
				},
			},
			"output_path": "out.xlsx",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(contentText(res.Content), "LIMIT_EXCEEDED") {
		t.Fatalf("expected LIMIT_EXCEEDED, got %v", res.Content)
	}
}

func TestExactlyOneInput(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table":       map[string]any{"tables": []any{map[string]any{"rows": []any{[]any{1}}}}},
			"data":        []any{[]any{1}},
			"output_path": "out.xlsx",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError || !strings.Contains(contentText(res.Content), "exactly one") {
		t.Fatalf("expected exactly-one error, got %v", res.Content)
	}
}

func TestToolAnnotations(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		switch tool.Name {
		case "json2xlsx_convert":
			if tool.Annotations == nil || tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
				t.Errorf("convert annotations = %+v", tool.Annotations)
			}
		case "json2xlsx_validate", "json2xlsx_preview", "json2xlsx_inspect", "json2xlsx_functions":
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Errorf("%s annotations = %+v", tool.Name, tool.Annotations)
			}
		}
	}
}

func TestRelativeInputPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.json"), []byte(`{"cells":{"A1":{"t":"s","v":"relative"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "json2xlsx_preview",
		Arguments: map[string]any{"input_path": "in.json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
}

func TestConcurrentPreview(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	args := map[string]any{
		"table": map[string]any{
			"tables": []any{
				map[string]any{"header": []any{"a"}, "rows": []any{[]any{1}}},
			},
		},
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "json2xlsx_preview", Arguments: args})
			if err != nil {
				errs <- err
				return
			}
			if res.IsError {
				errs <- fmt.Errorf("tool error: %v", res.Content)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent call: %v", err)
	}
}

func TestToJSONImageFile(t *testing.T) {
	dir := t.TempDir()
	cs, cleanup := newTestSession(t, dir)
	defer cleanup()
	ctx := context.Background()

	imgData, err := os.ReadFile("../../samples/sample_image.png")
	if err != nil {
		t.Fatalf("read sample image: %v", err)
	}
	xlsxPath := filepath.Join(dir, "image.xlsx")
	f := excelize.NewFile()
	if err := f.AddPictureFromBytes("Sheet1", "A1", &excelize.Picture{Extension: ".png", File: imgData}); err != nil {
		t.Fatalf("add picture: %v", err)
	}
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	jsonPath := filepath.Join(dir, "out.json")
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_to_json",
		Arguments: map[string]any{
			"input_path":  xlsxPath,
			"output_path": jsonPath,
			"image_mode":  "file",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "Sheet1_*.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected an extracted image file in %s", dir)
	}
}
