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

func evalFuncSin(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("SIN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Sin(n), nil
}

func evalFuncCos(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("COS requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Cos(n), nil
}

func evalFuncTan(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("TAN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Tan(n), nil
}

func evalFuncLn(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("LN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("LN #NUM!")
	}
	return math.Log(n), nil
}

func evalFuncLog10(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("LOG10 requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("LOG10 #NUM!")
	}
	return math.Log10(n), nil
}

func evalFuncExp(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("EXP requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Exp(n), nil
}

func evalFuncAsin(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ASIN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n < -1 || n > 1 {
		return 0, fmt.Errorf("ASIN #NUM!")
	}
	return math.Asin(n), nil
}

func evalFuncAcos(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ACOS requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n < -1 || n > 1 {
		return 0, fmt.Errorf("ACOS #NUM!")
	}
	return math.Acos(n), nil
}

func evalFuncAtan(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ATAN requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Atan(n), nil
}

func evalFuncDegrees(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("DEGREES requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return n * 180 / math.Pi, nil
}

func evalFuncRadians(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("RADIANS requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return n * math.Pi / 180, nil
}

func evalFuncAtan2(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("ATAN2 requires exactly 2 arguments")
	}
	x, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	y, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Atan2(y, x), nil
}

func evalFuncSinh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("SINH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Sinh(n), nil
}

func evalFuncCosh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("COSH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Cosh(n), nil
}

func evalFuncTanh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("TANH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Tanh(n), nil
}

func evalFuncAsinh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ASINH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Asinh(n), nil
}

func evalFuncAcosh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ACOSH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, fmt.Errorf("ACOSH #NUM!")
	}
	return math.Acosh(n), nil
}

func evalFuncAtanh(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ATANH requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n <= -1 || n >= 1 {
		return 0, fmt.Errorf("ATANH #NUM!")
	}
	return math.Atanh(n), nil
}

func evalFuncLog(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 1 || len(args) > 2 {
		return 0, fmt.Errorf("LOG requires 1 or 2 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("LOG #NUM!")
	}
	if len(args) == 1 {
		return math.Log(n), nil
	}
	base, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	if base <= 0 || base == 1 {
		return 0, fmt.Errorf("LOG #NUM!")
	}
	return math.Log(n) / math.Log(base), nil
}

func evalFuncFact(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("FACT requires exactly 1 argument")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	m := int(n)
	if float64(m) != n {
		m = int(math.Trunc(n))
	}
	if m < 0 {
		return 0, fmt.Errorf("FACT #NUM!")
	}
	if m == 0 {
		return 1, nil
	}
	result := 1.0
	for i := 2; i <= m; i++ {
		result *= float64(i)
	}
	return result, nil
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
