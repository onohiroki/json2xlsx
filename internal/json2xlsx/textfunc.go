package json2xlsx

import (
	"fmt"
	"strings"
	"unicode"
)

func evalFuncConcat(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) == 0 {
		return strVal(""), nil
	}
	var b strings.Builder
	for _, arg := range args {
		fv, err := arg.eval(ctx)
		if err != nil {
			return formulaValue{}, err
		}
		b.WriteString(fv.asString())
	}
	return strVal(b.String()), nil
}

func evalFuncLeft(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 2 {
		return formulaValue{}, fmt.Errorf("LEFT requires 2 arguments")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	n, err := ctx.evalArgNum(args[1])
	if err != nil {
		return formulaValue{}, err
	}
	runes := []rune(str.asString())
	if n < 0 {
		n = 0
	}
	if int(n) > len(runes) {
		n = float64(len(runes))
	}
	return strVal(string(runes[:int(n)])), nil
}

func evalFuncRight(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 2 {
		return formulaValue{}, fmt.Errorf("RIGHT requires 2 arguments")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	n, err := ctx.evalArgNum(args[1])
	if err != nil {
		return formulaValue{}, err
	}
	runes := []rune(str.asString())
	if n < 0 {
		n = 0
	}
	if int(n) > len(runes) {
		n = float64(len(runes))
	}
	return strVal(string(runes[len(runes)-int(n):])), nil
}

func evalFuncMid(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 3 {
		return formulaValue{}, fmt.Errorf("MID requires 3 arguments")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	start, err := ctx.evalArgNum(args[1])
	if err != nil {
		return formulaValue{}, err
	}
	n, err := ctx.evalArgNum(args[2])
	if err != nil {
		return formulaValue{}, err
	}
	runes := []rune(str.asString())
	// Excel: start is 1-based
	idx := int(start) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(runes) {
		return strVal(""), nil
	}
	if n < 0 {
		n = 0
	}
	end := idx + int(n)
	if end > len(runes) {
		end = len(runes)
	}
	return strVal(string(runes[idx:end])), nil
}

func evalFuncLen(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 1 {
		return formulaValue{}, fmt.Errorf("LEN requires 1 argument")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	return numVal(float64(len([]rune(str.asString())))), nil
}

func evalFuncUpper(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 1 {
		return formulaValue{}, fmt.Errorf("UPPER requires 1 argument")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	return strVal(strings.ToUpper(str.asString())), nil
}

func evalFuncLower(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 1 {
		return formulaValue{}, fmt.Errorf("LOWER requires 1 argument")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	return strVal(strings.ToLower(str.asString())), nil
}

func evalFuncTrim(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 1 {
		return formulaValue{}, fmt.Errorf("TRIM requires 1 argument")
	}
	str, err := args[0].eval(ctx)
	if err != nil {
		return formulaValue{}, err
	}
	// Excel TRIM: removes all leading/trailing spaces, and replaces
	// multiple internal spaces with a single space.
	s := str.asString()
	s = strings.TrimSpace(s)
	// Collapse multiple spaces
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strVal(b.String()), nil
}
