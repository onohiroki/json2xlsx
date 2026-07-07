package json2xlsx

import (
	"fmt"
	"math"
	"math/rand"
)

func evalFuncFloor(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("FLOOR requires exactly 2 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	sig, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if sig == 0 {
		return 0, fmt.Errorf("FLOOR #DIV/0!")
	}
	if n > 0 && sig < 0 {
		return 0, fmt.Errorf("FLOOR #NUM!")
	}
	return math.Floor(n/sig) * sig, nil
}

func evalFuncCeiling(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("CEILING requires exactly 2 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	sig, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if sig == 0 {
		return 0, fmt.Errorf("CEILING #DIV/0!")
	}
	if n > 0 && sig < 0 {
		return 0, fmt.Errorf("CEILING #NUM!")
	}
	return math.Ceil(n/sig) * sig, nil
}

func evalFuncMod(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("MOD requires exactly 2 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	d, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if d == 0 {
		return 0, fmt.Errorf("MOD #DIV/0!")
	}
	return n - d*math.Floor(n/d), nil
}

func evalFuncPower(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("POWER requires exactly 2 arguments")
	}
	b, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	e, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Pow(b, e), nil
}

func evalFuncSqrt(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("SQRT requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("SQRT #NUM!")
	}
	return math.Sqrt(n), nil
}

func evalFuncInt(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("INT requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Floor(n), nil
}

func evalFuncTrunc(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 1 || len(args) > 2 {
		return 0, fmt.Errorf("TRUNC requires 1 or 2 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	digits := 0
	if len(args) == 2 {
		d, err := args[1].eval(ctx)
		if err != nil {
			return 0, err
		}
		digits = int(d)
	}
	scale := math.Pow(10, float64(digits))
	return math.Trunc(n*scale) / scale, nil
}

func evalFuncSign(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("SIGN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 1, nil
	}
	if n < 0 {
		return -1, nil
	}
	return 0, nil
}

func evalFuncPi(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 0 {
		return 0, fmt.Errorf("PI requires no arguments")
	}
	return math.Pi, nil
}

func evalFuncRand(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 0 {
		return 0, fmt.Errorf("RAND requires no arguments")
	}
	return rand.Float64(), nil
}

func evalFuncCounta(ctx *evalContext, args []expr) (float64, error) {
	var count float64
	for _, arg := range args {
		switch a := arg.(type) {
		case *rangeExpr:
			refs := expandRange(a.start, a.end)
			for _, ref := range refs {
				if _, ok := ctx.cells[ref]; ok {
					count++
				}
			}
		case *cellRefExpr:
			if _, ok := ctx.cells[a.ref]; ok {
				count++
			}
		default:
			_, err := arg.eval(ctx)
			if err == nil {
				count++
			}
		}
	}
	return count, nil
}
