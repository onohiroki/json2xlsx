package json2xlsx

import "fmt"

func evalFuncVar(ctx *evalContext, args []expr, population bool) (float64, error) {
	var all []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	n := len(all)
	if n == 0 {
		return 0, fmt.Errorf("VAR of empty set")
	}
	if !population && n < 2 {
		return 0, fmt.Errorf("VAR.S requires at least 2 values")
	}
	var sum float64
	for _, v := range all {
		sum += v
	}
	mean := sum / float64(n)
	var sqDiff float64
	for _, v := range all {
		d := v - mean
		sqDiff += d * d
	}
	divisor := float64(n)
	if !population {
		divisor = float64(n - 1)
	}
	return sqDiff / divisor, nil
}

func evalFuncVarS(ctx *evalContext, args []expr) (float64, error) {
	return evalFuncVar(ctx, args, false)
}

func evalFuncVarP(ctx *evalContext, args []expr) (float64, error) {
	return evalFuncVar(ctx, args, true)
}
