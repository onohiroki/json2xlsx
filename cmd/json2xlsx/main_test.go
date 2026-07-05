package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onohiroki/json2xlsx/internal/json2xlsx"
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
	mode, err := resolveDateMode(false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeSerial {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeSerial)
	}
}

func TestResolveDateMode_Display(t *testing.T) {
	mode, err := resolveDateMode(true, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeDisplay {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeDisplay)
	}
}

func TestResolveDateMode_RFC3339(t *testing.T) {
	mode, err := resolveDateMode(false, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeRFC3339 {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeRFC3339)
	}
}

func TestResolveDateMode_Serial(t *testing.T) {
	mode, err := resolveDateMode(false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != json2xlsx.DateModeSerial {
		t.Errorf("got %q, want %q", mode, json2xlsx.DateModeSerial)
	}
}

func TestResolveDateMode_Conflict(t *testing.T) {
	_, err := resolveDateMode(true, true, false)
	if err == nil {
		t.Fatal("expected error for conflicting flags")
	}
}

func TestResolveDateMode_TripleConflict(t *testing.T) {
	_, err := resolveDateMode(true, true, true)
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
