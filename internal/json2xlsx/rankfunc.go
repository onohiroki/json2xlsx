package json2xlsx

import (
	"fmt"
	"sort"
)

func evalFuncRank(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args) > 3 {
		return 0, fmt.Errorf("RANK requires 2 or 3 arguments")
	}
	n, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	refs, err := rangeOrCellRefs(ctx, args[1])
	if err != nil {
		return 0, fmt.Errorf("RANK second argument: %w", err)
	}
	if len(refs) == 0 {
		return 0, fmt.Errorf("RANK empty reference")
	}
	var values []float64
	for _, ref := range refs {
		v, err := ctx.getCellValue(ref)
		if err != nil {
			continue
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("RANK no values in reference")
	}
	order := 0.0
	if len(args) == 3 {
		order, err = args[2].eval(ctx)
		if err != nil {
			return 0, err
		}
	}
	if order == 0 {
		sort.Sort(sort.Reverse(sort.Float64Slice(values)))
	} else {
		sort.Float64s(values)
	}
	rank := 1
	for i, v := range values {
		if v == n {
			rank = i + 1
			break
		}
	}
	for _, v := range values {
		if v == n {
			return float64(rank), nil
		}
	}
	return 0, fmt.Errorf("RANK value not found in reference")
}

func evalFuncLarge(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("LARGE requires exactly 2 arguments")
	}
	var all []float64
	vals, err := ctx.evalArg(args[0])
	if err != nil {
		return 0, err
	}
	all = append(all, vals...)
	if len(all) == 0 {
		return 0, fmt.Errorf("LARGE of empty set")
	}
	kRaw, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	k := int(kRaw)
	if k < 1 || k > len(all) {
		return 0, fmt.Errorf("LARGE k out of range")
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(all)))
	return all[k-1], nil
}

func evalFuncSmall(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("SMALL requires exactly 2 arguments")
	}
	var all []float64
	vals, err := ctx.evalArg(args[0])
	if err != nil {
		return 0, err
	}
	all = append(all, vals...)
	if len(all) == 0 {
		return 0, fmt.Errorf("SMALL of empty set")
	}
	kRaw, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	k := int(kRaw)
	if k < 1 || k > len(all) {
		return 0, fmt.Errorf("SMALL k out of range")
	}
	sort.Float64s(all)
	return all[k-1], nil
}
