package json2xlsx

import (
	"fmt"
	"math"
	"sort"
)

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

func percentileInc(all []float64, k float64) float64 {
	sort.Float64s(all)
	n := len(all)
	if k <= 0 {
		return all[0]
	}
	if k >= 1 {
		return all[n-1]
	}
	pos := k * float64(n-1)
	lower := int(math.Floor(pos))
	frac := pos - float64(lower)
	if lower >= n-1 {
		return all[n-1]
	}
	return all[lower] + frac*(all[lower+1]-all[lower])
}

func evalFuncQuartileInc(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("QUARTILE.INC requires at least 2 arguments")
	}
	var all []float64
	for i := 0; i < len(args)-1; i++ {
		vals, err := ctx.evalArg(args[i])
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("QUARTILE.INC of empty set")
	}
	quart, err := args[len(args)-1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if quart < 0 || quart > 4 {
		return 0, fmt.Errorf("QUARTILE.INC quart must be 0-4")
	}
	return percentileInc(all, quart/4.0), nil
}

func evalFuncPercentileInc(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("PERCENTILE.INC requires at least 2 arguments")
	}
	var all []float64
	for i := 0; i < len(args)-1; i++ {
		vals, err := ctx.evalArg(args[i])
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("PERCENTILE.INC of empty set")
	}
	k, err := args[len(args)-1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if k < 0 || k > 1 {
		return 0, fmt.Errorf("PERCENTILE.INC k must be between 0 and 1")
	}
	return percentileInc(all, k), nil
}

func evalFuncGeomean(ctx *evalContext, args []expr) (float64, error) {
	var all []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("GEOMEAN of empty set")
	}
	var sumLog float64
	for _, v := range all {
		if v <= 0 {
			return 0, fmt.Errorf("GEOMEAN requires positive values")
		}
		sumLog += math.Log(v)
	}
	return math.Exp(sumLog / float64(len(all))), nil
}

func evalFuncHarmean(ctx *evalContext, args []expr) (float64, error) {
	var all []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("HARMEAN of empty set")
	}
	var sumRecip float64
	for _, v := range all {
		if v <= 0 {
			return 0, fmt.Errorf("HARMEAN requires positive values")
		}
		sumRecip += 1.0 / v
	}
	return float64(len(all)) / sumRecip, nil
}

func evalFuncTrimmean(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("TRIMMEAN requires at least 2 arguments")
	}
	var all []float64
	for i := 0; i < len(args)-1; i++ {
		vals, err := ctx.evalArg(args[i])
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("TRIMMEAN of empty set")
	}
	percent, err := args[len(args)-1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if percent < 0 || percent >= 1 {
		return 0, fmt.Errorf("TRIMMEAN percent must be between 0 and 1")
	}
	sort.Float64s(all)
	n := len(all)
	np := float64(n) * percent
	numToExclude := int(math.Floor(np/2.0)) * 2
	start := numToExclude / 2
	end := n - numToExclude/2
	if start >= end {
		return 0, fmt.Errorf("TRIMMEAN too much trimming")
	}
	var total float64
	for i := start; i < end; i++ {
		total += all[i]
	}
	return total / float64(end-start), nil
}
