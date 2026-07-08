package json2xlsx

import "fmt"

func evalFuncIf(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 3 {
		return 0, fmt.Errorf("IF requires exactly 3 arguments")
	}
	cond, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if cond != 0 {
		return args[1].eval(ctx)
	}
	return args[2].eval(ctx)
}

func evalFuncAnd(ctx *evalContext, args []expr) (float64, error) {
	for _, arg := range args {
		v, err := arg.eval(ctx)
		if err != nil {
			return 0, err
		}
		if v == 0 {
			return 0, nil
		}
	}
	return 1, nil
}

func evalFuncOr(ctx *evalContext, args []expr) (float64, error) {
	for _, arg := range args {
		v, err := arg.eval(ctx)
		if err != nil {
			return 0, err
		}
		if v != 0 {
			return 1, nil
		}
	}
	return 0, nil
}

func evalFuncNot(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("NOT requires exactly 1 argument")
	}
	v, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return 1, nil
	}
	return 0, nil
}

func evalFuncSwitch(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 3 {
		return 0, fmt.Errorf("SWITCH requires at least 3 arguments")
	}
	expr, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	for i := 1; i < len(args)-1; i += 2 {
		val, err := args[i].eval(ctx)
		if err != nil {
			return 0, err
		}
		if val == expr {
			return args[i+1].eval(ctx)
		}
	}
	if len(args)%2 == 0 {
		return args[len(args)-1].eval(ctx)
	}
	return 0, fmt.Errorf("SWITCH #N/A")
}

func evalFuncIfs(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return 0, fmt.Errorf("IFS requires an even number of arguments (condition, value pairs)")
	}
	for i := 0; i < len(args); i += 2 {
		cond, err := args[i].eval(ctx)
		if err != nil {
			return 0, err
		}
		if cond != 0 {
			return args[i+1].eval(ctx)
		}
	}
	return 0, fmt.Errorf("IFS #N/A")
}
