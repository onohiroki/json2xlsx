package json2xlsx

import "fmt"

func evalFuncIferror(ctx *evalContext, args []expr) (formulaValue, error) {
	if len(args) != 2 {
		return formulaValue{}, fmt.Errorf("IFERROR requires exactly 2 arguments")
	}
	val, err := args[0].eval(ctx)
	if err != nil {
		return args[1].eval(ctx)
	}
	return val, nil
}
