package json2xlsx

import (
	"encoding/json"
	"fmt"
	"strings"
)

func evalFuncIsnumber(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISNUMBER requires exactly 1 argument")
	}
	if ref, ok := args[0].(*cellRefExpr); ok {
		cell, exists := ctx.cells[ref.ref]
		if !exists || cell.V == nil {
			return 0, nil
		}
		switch cell.V.(type) {
		case float64, float32, int, int64, json.Number:
			return 1, nil
		default:
			return 0, nil
		}
	}
	_, err := args[0].eval(ctx)
	if err != nil {
		return 0, nil
	}
	return 1, nil
}

func evalFuncIsblank(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISBLANK requires exactly 1 argument")
	}
	if ref, ok := args[0].(*cellRefExpr); ok {
		cell, exists := ctx.cells[ref.ref]
		if !exists || cell.V == nil {
			return 1, nil
		}
		return 0, nil
	}
	return 0, nil
}

func evalFuncIstext(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISTEXT requires exactly 1 argument")
	}
	if ref, ok := args[0].(*cellRefExpr); ok {
		cell, exists := ctx.cells[ref.ref]
		if !exists || cell.V == nil {
			return 0, nil
		}
		_, isStr := cell.V.(string)
		if isStr {
			return 1, nil
		}
		return 0, nil
	}
	return 0, nil
}

func evalFuncIsnontext(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISNONTEXT requires exactly 1 argument")
	}
	if ref, ok := args[0].(*cellRefExpr); ok {
		cell, exists := ctx.cells[ref.ref]
		if !exists || cell.V == nil {
			return 1, nil
		}
		_, isStr := cell.V.(string)
		if isStr {
			return 0, nil
		}
		return 1, nil
	}
	return 1, nil
}

func evalFuncIserror(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISERROR requires exactly 1 argument")
	}
	_, err := args[0].eval(ctx)
	if err != nil {
		return 1, nil
	}
	return 0, nil
}

func evalFuncIsna(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ISNA requires exactly 1 argument")
	}
	_, err := args[0].eval(ctx)
	if err != nil && strings.Contains(err.Error(), "#N/A") {
		return 1, nil
	}
	return 0, nil
}
