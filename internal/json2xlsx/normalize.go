package json2xlsx

import (
	"bytes"
	"fmt"
	"strings"
)

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func ValidateMode(mode MarkdownMode) error {
	switch mode {
	case MarkdownModeFormula, MarkdownModeValue, MarkdownModeBoth:
		return nil
	default:
		return fmt.Errorf("invalid mode: %q (expected f|v|both)", mode)
	}
}

func trimBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func checkJSONArrayWarning(res *ReadWorkbookResult, explicitMode bool, mode MarkdownMode) []string {
	var warnings []string
	if !res.IsXLSX && len(res.RawData) > 0 && bytes.TrimSpace(res.RawData)[0] == '[' {
		if explicitMode && (mode == MarkdownModeFormula || mode == MarkdownModeBoth) {
			warnings = append(warnings, fmt.Sprintf("Warning: --mode=%s is ignored for JSON array input (formulas not supported in this format).", mode))
		}
	}
	return warnings
}
