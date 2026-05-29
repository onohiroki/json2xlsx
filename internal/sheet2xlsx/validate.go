package sheet2xlsx

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var errPrinter = message.NewPrinter(language.English)

//go:embed schema.json
// NOTE: `schema.json` is a copy of `schemas/sheet2xlsx.schema.json`.
// Keep them in sync when updating the schema.
var schemaData string

var compiledSchema *jsonschema.Schema

func init() {
	var schemaObj any
	if err := json.Unmarshal([]byte(schemaData), &schemaObj); err != nil {
		panic("schema: " + err.Error())
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaObj); err != nil {
		panic("schema: " + err.Error())
	}
	var err error
	compiledSchema, err = c.Compile("schema.json")
	if err != nil {
		panic("schema: " + err.Error())
	}
}

const maxSchemaErrors = 10

func ValidateJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	if err := compiledSchema.Validate(v); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return formatSchemaError(ve)
		}
	}
	return nil
}

type schemaIssue struct {
	path    string
	message string
}

func formatSchemaError(ve *jsonschema.ValidationError) error {
	issues := collectIssues(ve)

	if len(issues) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("JSON structure issues")
	if len(issues) > maxSchemaErrors {
		fmt.Fprintf(&b, " (%d shown, %d more)", maxSchemaErrors, len(issues)-maxSchemaErrors)
	}
	b.WriteString(":\n")

	n := len(issues)
	if n > maxSchemaErrors {
		n = maxSchemaErrors
	}
	for _, iss := range issues[:n] {
		msg := strings.TrimRight(iss.message, "\n")
		lines := strings.Split(msg, "\n")
		b.WriteString("  ")
		b.WriteString(iss.path)
		b.WriteString(": ")
		b.WriteString(lines[0])
		b.WriteString("\n")
		for _, line := range lines[1:] {
			b.WriteString("    ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

func collectIssues(ve *jsonschema.ValidationError) []schemaIssue {
	if isRootOneOf(ve) {
		var issues []schemaIssue
		issues = append(issues, schemaIssue{
			path:    instPathToJQ(ve.InstanceLocation),
			message: `must be either a single sheet with "name" and "cells", a "sheets" array, or a "book" wrapper with version/sheets/charts (mode: embedded|chartSheet)`,
		})
		for _, cause := range ve.Causes {
			issues = append(issues, collectIssues(cause)...)
		}
		return issues
	}

	if len(ve.Causes) == 0 {
		return []schemaIssue{{
			path:    instPathToJQ(ve.InstanceLocation),
			message: ve.ErrorKind.LocalizedString(errPrinter),
		}}
	}

	var issues []schemaIssue
	for _, cause := range ve.Causes {
		issues = append(issues, collectIssues(cause)...)
	}
	return issues
}

func isRootOneOf(ve *jsonschema.ValidationError) bool {
	kp := ve.ErrorKind.KeywordPath()
	return len(kp) == 1 && kp[0] == "oneOf"
}

func instPathToJQ(segments []string) string {
	if len(segments) == 0 {
		return ".input"
	}
	var b strings.Builder
	for _, seg := range segments {
		if isAllDigits(seg) {
			fmt.Fprintf(&b, "[%s]", seg)
		} else {
			b.WriteString(".")
			b.WriteString(seg)
		}
	}
	return b.String()
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
