package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/onohiroki/json2xlsx/internal/json2xlsx"
	"github.com/xuri/excelize/v2"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "json2xlsx-cli-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp dir: %v\n", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmpDir, "json2xlsx")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %s\n", out)
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestResolveDateMode_Default(t *testing.T) {
	mode, err := json2xlsx.ResolveDateMode(false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeSerial {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeSerial)
	}
}

func TestResolveDateMode_Display(t *testing.T) {
	mode, err := json2xlsx.ResolveDateMode(true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeDisplay {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeDisplay)
	}
}

func TestResolveDateMode_RFC3339(t *testing.T) {
	mode, err := json2xlsx.ResolveDateMode(false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeRFC3339 {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeRFC3339)
	}
}

func TestResolveDateMode_Serial(t *testing.T) {
	mode, err := json2xlsx.ResolveDateMode(false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeSerial {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeSerial)
	}
}

func TestResolveDateMode_Conflict(t *testing.T) {
	_, err := json2xlsx.ResolveDateMode(true, true, false)
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
}

func TestResolveDateMode_TripleConflict(t *testing.T) {
	_, err := json2xlsx.ResolveDateMode(true, true, true)
	if err == nil {
		t.Fatal("expected error for triple conflicting flags")
	}
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run %v %v: %v", binPath, args, err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func runCLIWithStdin(t *testing.T, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run %v %v: %v", binPath, args, err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestCLI_Help(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		t.Run(arg, func(t *testing.T) {
			stdout, stderr, code := runCLI(t, arg)
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
			if stdout != "" {
				t.Errorf("expected stderr output, got stdout")
			}
			if !strings.Contains(stderr, "json2xlsx") {
				t.Errorf("help output should contain 'json2xlsx', got: %s", stderr)
			}
		})
	}
}

func TestCLI_ToXLSX_ValidJSON(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "hello"}}}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-xlsx")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if len(stdout) == 0 {
		t.Fatal("expected non-empty XLSX output")
	}
	if !bytes.HasPrefix([]byte(stdout), []byte{0x50, 0x4B, 0x03, 0x04}) {
		t.Error("output does not start with XLSX magic bytes (PK\\x03\\x04)")
	}
}

func TestCLI_ToXLSX_ValidJSON_NoSubcommand(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "hello"}}}`
	stdout, stderr, code := runCLIWithStdin(t, js)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if len(stdout) == 0 {
		t.Fatal("expected non-empty XLSX output")
	}
}

func TestCLI_ToJSON_ValidXLSX(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "hello"}}}`
	xlsxOut, _, code := runCLIWithStdin(t, js, "to-xlsx")
	if code != 0 {
		t.Fatalf("to-xlsx failed: %d", code)
	}

	stdout, stderr, code := runCLIWithStdin(t, xlsxOut, "to-json")
	if code != 0 {
		t.Fatalf("to-json exit code = %d, stderr: %s", code, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatal("to-json output is not valid JSON")
	}
}

func TestCLI_ToMD_ValidJSON(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "hello"}, "B1": {"t": "n", "v": 42}}}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-md")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("expected 'hello' in markdown output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "42") {
		t.Errorf("expected '42' in markdown output, got: %s", stdout)
	}
}

func TestCLI_ToMD_FirstRowHeader(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "Name"}, "A2": {"t": "s", "v": "Alice"}}}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-md", "--first-row-header")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "|   | A |") {
		t.Error("expected no A/B/C column headers with --first-row-header")
	}
}

func TestCLI_ToHTML_ValidJSON(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "hello"}}}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-html")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("expected 'hello' in HTML output, got: %s", stdout)
	}
}

func TestCLI_ToCSV_ValidJSON(t *testing.T) {
	js := `{"cells": {"A1": {"t": "s", "v": "a"}, "B1": {"t": "n", "v": 1}}}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-csv")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "a,1") {
		t.Errorf("expected 'a,1' in CSV output, got: %s", stdout)
	}
}

func TestCLI_ToXLSX_DataJSON_Array(t *testing.T) {
	js := `[["a", 1], ["b", 2]]`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-xlsx", "--data-json")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if len(stdout) == 0 {
		t.Fatal("expected non-empty XLSX output")
	}
}

func TestCLI_ToXLSX_InvalidJSON(t *testing.T) {
	_, stderr, code := runCLIWithStdin(t, "not json", "to-xlsx")
	if code == 0 {
		t.Fatal("expected non-zero exit code for invalid JSON")
	}
	if stderr == "" {
		t.Error("expected error message on stderr")
	}
}

func TestCLI_ToCSV_WithSheetIndex(t *testing.T) {
	js := `{"sheets": [{"name": "S1", "cells": {"A1": {"t": "s", "v": "a"}}}, {"name": "S2", "cells": {"A1": {"t": "s", "v": "b"}}}]}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-csv", "--sheet-index", "2")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "b") {
		t.Errorf("expected 'b' from sheet 2, got: %s", stdout)
	}
}

func TestCLI_ToJSON_DateDisplay(t *testing.T) {
	js := `{"cells": {"A1": {"t": "d", "v": 45000}}}`
	xlsxOut, _, code := runCLIWithStdin(t, js, "to-xlsx")
	if code != 0 {
		t.Fatalf("to-xlsx failed: %d", code)
	}

	stdout, stderr, code := runCLIWithStdin(t, xlsxOut, "to-json", "--date-display")
	if code != 0 {
		t.Fatalf("to-json --date-display exit code = %d, stderr: %s", code, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatal("to-json --date-display output is not valid JSON")
	}
}

func TestCLI_ToJSON_DateRFC3339(t *testing.T) {
	js := `{"cells": {"A1": {"t": "d", "v": 45000}}}`
	xlsxOut, _, code := runCLIWithStdin(t, js, "to-xlsx")
	if code != 0 {
		t.Fatalf("to-xlsx failed: %d", code)
	}

	stdout, stderr, code := runCLIWithStdin(t, xlsxOut, "to-json", "--date-rfc3339")
	if code != 0 {
		t.Fatalf("to-json --date-rfc3339 exit code = %d, stderr: %s", code, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatal("to-json --date-rfc3339 output is not valid JSON")
	}
}

func TestCLI_ToXLSX_Compute(t *testing.T) {
	js := `{"name":"S1","cells":{"A1":{"t":"n","v":3},"B1":{"t":"n","v":7},"C1":{"t":"f","f":"SUM(A1:B1)"}}}`
	xlsxOut, stderr, code := runCLIWithStdin(t, js, "to-xlsx", "--compute")
	if code != 0 {
		t.Fatalf("to-xlsx --compute exit code = %d, stderr: %s", code, stderr)
	}

	// Verify the computed value is present by converting back to JSON
	jsonOut, stderr, code := runCLIWithStdin(t, xlsxOut, "to-json")
	if code != 0 {
		t.Fatalf("to-json exit code = %d, stderr: %s", code, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// Navigate to C1
	book := result["book"].(map[string]interface{})
	sheets := book["sheets"].(map[string]interface{})
	s1 := sheets["S1"].(map[string]interface{})
	cells := s1["cells"].(map[string]interface{})
	c1 := cells["C1"].(map[string]interface{})

	v, ok := c1["v"]
	if !ok {
		t.Fatal("C1.v is missing -- formula was not computed")
	}
	if v.(float64) != 10 {
		t.Errorf("C1.v = %v, want 10", v)
	}

	// Formula should still be present
	f, ok := c1["f"]
	if !ok || f != "SUM(A1:B1)" {
		t.Errorf("C1.f = %v, want SUM(A1:B1)", f)
	}
}

const formulaJSON = `{"name":"S1","cells":{"A1":{"t":"n","v":3},"B1":{"t":"n","v":7},"C1":{"t":"f","f":"SUM(A1:B1)"}}}`

func TestCLI_ToMD_Compute(t *testing.T) {
	t.Run("with --compute", func(t *testing.T) {
		stdout, stderr, code := runCLIWithStdin(t, formulaJSON, "to-md", "--mode", "v", "--compute")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "10") {
			t.Errorf("expected computed value '10' in output, got: %s", stdout)
		}
		if strings.Contains(stderr, "Warning") {
			t.Errorf("no warning expected with --compute, got: %s", stderr)
		}
	})

	t.Run("without --compute", func(t *testing.T) {
		_, stderr, code := runCLIWithStdin(t, formulaJSON, "to-md", "--mode", "v")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stderr, "Warning") {
			t.Errorf("expected warning about missing values, got: %s", stderr)
		}
	})
}

func TestCLI_ToHTML_Compute(t *testing.T) {
	t.Run("with --compute", func(t *testing.T) {
		stdout, stderr, code := runCLIWithStdin(t, formulaJSON, "to-html", "--mode", "v", "--compute")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "10") {
			t.Errorf("expected computed value '10' in output, got: %s", stdout)
		}
		if strings.Contains(stderr, "Warning") {
			t.Errorf("no warning expected with --compute, got: %s", stderr)
		}
	})

	t.Run("without --compute", func(t *testing.T) {
		_, stderr, code := runCLIWithStdin(t, formulaJSON, "to-html", "--mode", "v")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stderr, "Warning") {
			t.Errorf("expected warning about missing values, got: %s", stderr)
		}
	})
}

func TestCLI_ToCSV_Compute(t *testing.T) {
	t.Run("with --compute", func(t *testing.T) {
		stdout, stderr, code := runCLIWithStdin(t, formulaJSON, "to-csv", "--compute")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stdout, "10") {
			t.Errorf("expected computed value '10' in output, got: %s", stdout)
		}
		if strings.Contains(stderr, "Warning") {
			t.Errorf("no warning expected with --compute, got: %s", stderr)
		}
	})

	t.Run("without --compute", func(t *testing.T) {
		_, stderr, code := runCLIWithStdin(t, formulaJSON, "to-csv")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr)
		}
		if !strings.Contains(stderr, "Warning") {
			t.Errorf("expected warning about missing values, got: %s", stderr)
		}
	})
}

func TestExtractFormulaArg(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		formula string
		rest    []string
	}{
		{"formula only", []string{"=1+2"}, "=1+2", []string{}},
		{"flag before formula", []string{"-i", "book.xlsx", "=SUM(A1:A2)"}, "=SUM(A1:A2)", []string{"-i", "book.xlsx"}},
		{"formula before flag", []string{"=SUM(A1:A2)", "-i", "book.xlsx"}, "=SUM(A1:A2)", []string{"-i", "book.xlsx"}},
		{"json flag", []string{"--json", "=1"}, "=1", []string{"--json"}},
		{"cell flag", []string{"--cell", "B2", "=A1"}, "=A1", []string{"--cell", "B2"}},
		{"sheet with equals", []string{"--sheet=Data", "=A1"}, "=A1", []string{"--sheet=Data"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			formula, rest, err := extractFormulaArg(tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if formula != tc.formula {
				t.Errorf("formula = %q, want %q", formula, tc.formula)
			}
			if strings.Join(rest, " ") != strings.Join(tc.rest, " ") {
				t.Errorf("rest = %v, want %v", rest, tc.rest)
			}
		})
	}

	if _, _, err := extractFormulaArg([]string{"=1", "=2"}); err == nil {
		t.Error("expected error for two positional formulas")
	}
}

func TestCLI_Functions(t *testing.T) {
	stdout, stderr, code := runCLI(t, "functions", "--search", "lookup")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	for _, name := range []string{"VLOOKUP", "HLOOKUP", "XLOOKUP"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %s in output, got: %s", name, stdout)
		}
	}

	stdout, stderr, code = runCLI(t, "functions", "--category", "date", "--json")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("functions --json output is not valid JSON: %s", stdout)
	}
	if !strings.Contains(stdout, `"name": "TODAY"`) && !strings.Contains(stdout, `"name":"TODAY"`) {
		t.Errorf("expected TODAY in JSON output, got: %s", stdout)
	}

	_, stderr, code = runCLI(t, "functions", "--category", "nope")
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for unknown category (stderr: %s)", code, stderr)
	}
}

func TestCLI_Info(t *testing.T) {
	book := `{"name":"S1","cells":{"A1":{"t":"s","v":"x"},"B2":{"t":"f","f":"1+1"}}}`
	stdout, stderr, code := runCLIWithStdin(t, book, "info")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "S1") || !strings.Contains(stdout, "A1:B2") {
		t.Errorf("expected sheet name and used range, got: %s", stdout)
	}

	stdout, stderr, code = runCLIWithStdin(t, book, "info", "--json")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("info --json output is not valid JSON: %s", stdout)
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info["schema"] != "json2xlsx-info/1" {
		t.Errorf("schema = %v, want json2xlsx-info/1", info["schema"])
	}
}

func TestCLI_Eval(t *testing.T) {
	stdout, stderr, code := runCLI(t, "eval", "=SUM(1,2,3)")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "6" {
		t.Errorf("stdout = %q, want 6", stdout)
	}

	// 数式エラーは exit 0 で出力される
	stdout, stderr, code = runCLI(t, "eval", "=1/0")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if strings.TrimSpace(stdout) != "#DIV/0!" {
		t.Errorf("stdout = %q, want #DIV/0!", stdout)
	}

	stdout, stderr, code = runCLI(t, "eval", "--json", "=1/0")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !json.Valid([]byte(stdout)) || !strings.Contains(stdout, "#DIV/0!") {
		t.Errorf("expected JSON error with code, got: %s", stdout)
	}
}

func TestCLI_EvalWithStdinWorkbook(t *testing.T) {
	book := `{"name":"S1","cells":{"A1":{"t":"n","v":10},"A2":{"t":"n","v":20}}}`
	stdout, stderr, code := runCLIWithStdin(t, book, "eval", "=SUM(A1:A2)")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "30" {
		t.Errorf("stdout = %q, want 30", stdout)
	}
}

func TestCLI_ToMD_Range(t *testing.T) {
	book := `{"name":"S1","cells":{
		"A1":{"t":"s","v":"a"},"B1":{"t":"s","v":"b"},
		"A2":{"t":"n","v":1},"B2":{"t":"n","v":2}}}`
	stdout, stderr, code := runCLIWithStdin(t, book, "to-md", "--range", "B1:B2")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "| B") {
		t.Errorf("expected column B label, got: %s", stdout)
	}
	if strings.Contains(stdout, "| A") {
		t.Errorf("column A should not appear, got: %s", stdout)
	}
}

func TestCLI_ToCSV_Range(t *testing.T) {
	book := `{"name":"S1","cells":{
		"A1":{"t":"n","v":1},"B1":{"t":"n","v":2},
		"A2":{"t":"n","v":3},"B2":{"t":"n","v":4}}}`
	stdout, stderr, code := runCLIWithStdin(t, book, "to-csv", "--range", "B1:B2")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if strings.TrimSpace(stdout) != "2\n4" {
		t.Errorf("stdout = %q, want \"2\\n4\"", stdout)
	}
}

func TestCLI_ToXLSX_TableJSON(t *testing.T) {
	js := `{"tables":[{"header":["a","b"],"rows":[[1,2]],"columns":[{},{"format":"#,##0"}]}]}`
	stdout, stderr, code := runCLIWithStdin(t, js, "to-xlsx", "--table-json")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	if !strings.HasPrefix(stdout, "PK\x03\x04") {
		t.Fatalf("output does not look like XLSX")
	}
}

func TestCLI_ToXLSX_TableJSONInvalid(t *testing.T) {
	js := `{"tables":[{"rows":[[1]],"headerStyle":"missing"}]}`
	_, stderr, code := runCLIWithStdin(t, js, "to-xlsx", "--table-json")
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !strings.Contains(stderr, "headerStyle") {
		t.Fatalf("stderr = %q, want headerStyle issue", stderr)
	}
}

func TestCLI_MCP_ConvertTable(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "cli-test", Version: "1"}, nil)
	cmd := exec.Command(binPath, "mcp", "--allow-dir", dir)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("no tools listed")
	}

	outPath := filepath.Join(dir, "mcp.xlsx")
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "json2xlsx_convert",
		Arguments: map[string]any{
			"table": map[string]any{
				"tables": []any{
					map[string]any{
						"header": []any{"a", "b"},
						"rows":   []any{[]any{1, 2}},
					},
				},
			},
			"output_path": outPath,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output is empty")
	}
}

func TestCLI_MCP_CreateAllowDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nested", "out")

	_, stderr, code := runCLIWithStdin(t, "", "mcp", "--allow-dir", missing)
	if code == 0 {
		t.Fatalf("expected failure without --create-allow-dir")
	}
	if !strings.Contains(stderr, "allow-dir") {
		t.Fatalf("stderr = %q", stderr)
	}

	_, stderr, code = runCLIWithStdin(t, "", "mcp", "--create-allow-dir", "--allow-dir", missing)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	info, err := os.Stat(missing)
	if err != nil {
		t.Fatalf("stat created allow-dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", missing)
	}
}

func TestCLI_ToJSON_ImageOverwriteGuard(t *testing.T) {
	dir := t.TempDir()
	imgData, err := os.ReadFile("../../samples/sample_image.png")
	if err != nil {
		t.Fatal(err)
	}
	xlsxPath := filepath.Join(dir, "in.xlsx")
	f := excelize.NewFile()
	if err := f.AddPictureFromBytes("Sheet1", "A1", &excelize.Picture{Extension: ".png", File: imgData}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	imagePath := filepath.Join(dir, "Sheet1_1.png")
	if err := os.WriteFile(imagePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(dir, "out.json")

	_, stderr, code := runCLI(t, "to-json", "-i", xlsxPath, "-o", jsonPath, "--image-mode", "file")
	if code == 0 {
		t.Fatalf("expected failure without --overwrite-images")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("stderr = %q", stderr)
	}
	if got, _ := os.ReadFile(imagePath); string(got) != "old" {
		t.Fatalf("image = %q, want old", got)
	}
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("JSON should not have been written: %v", err)
	}

	_, stderr, code = runCLI(t, "to-json", "-i", xlsxPath, "-o", jsonPath, "--image-mode", "file", "--overwrite-images")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr: %s", code, stderr)
	}
	got, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "old" || len(got) == 0 {
		t.Fatalf("image was not replaced: %q", got)
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("JSON output: %v", err)
	}
}
