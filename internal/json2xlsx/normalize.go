package json2xlsx

import (
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

func checkDataJSONModeWarning(res *ReadWorkbookResult, dataJSON, explicitMode bool, mode MarkdownMode) []string {
	if !dataJSON || res.IsXLSX || !explicitMode {
		return nil
	}
	if mode != MarkdownModeFormula && mode != MarkdownModeBoth {
		return nil
	}
	return []string{fmt.Sprintf("Warning: --mode=%s has no effect for data JSON input (formulas are not supported in this format).", mode)}
}
