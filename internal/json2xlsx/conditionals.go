package json2xlsx

import "fmt"

func evalFuncAverageif(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args) > 3 {
		return 0, fmt.Errorf("AVERAGEIF requires 2 or 3 arguments")
	}
	checkRange, err := rangeOrCellRefs(ctx, args[0])
	if err != nil {
		return 0, fmt.Errorf("AVERAGEIF first argument: %w", err)
	}
	criteriaVal, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	var avgRefs []string
	if len(args) == 3 {
		avgRefs, err = rangeOrCellRefs(ctx, args[2])
		if err != nil {
			return 0, fmt.Errorf("AVERAGEIF third argument: %w", err)
		}
	} else {
		avgRefs = checkRange
	}
	limit := len(checkRange)
	if len(avgRefs) < limit {
		limit = len(avgRefs)
	}
	var total float64
	var count float64
	for i := 0; i < limit; i++ {
		cellVal, err := ctx.getCellValue(checkRange[i])
		if err != nil {
			continue
		}
		if cellVal == criteriaVal {
			avgVal, err := ctx.getCellValue(avgRefs[i])
			if err == nil {
				total += avgVal
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("AVERAGEIF #DIV/0!")
	}
	return total / count, nil
}

func evalFuncSumifs(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 3 || len(args)%2 == 0 {
		return 0, fmt.Errorf("SUMIFS requires sum_range + criteria_range,criteria pairs")
	}
	sumRefs, err := rangeOrCellRefs(ctx, args[0])
	if err != nil {
		return 0, fmt.Errorf("SUMIFS sum_range: %w", err)
	}
	numPairs := (len(args) - 1) / 2
	type criteria struct {
		refs []string
		val  float64
	}
	criteriaList := make([]criteria, numPairs)
	for i := 0; i < numPairs; i++ {
		crIdx := 1 + i*2
		cvIdx := 1 + i*2 + 1
		refs, err := rangeOrCellRefs(ctx, args[crIdx])
		if err != nil {
			return 0, fmt.Errorf("SUMIFS criteria_range %d: %w", i+1, err)
		}
		val, err := args[cvIdx].eval(ctx)
		if err != nil {
			return 0, err
		}
		criteriaList[i] = criteria{refs, val}
	}
	limit := len(sumRefs)
	for _, c := range criteriaList {
		if len(c.refs) < limit {
			limit = len(c.refs)
		}
	}
	var total float64
	for row := 0; row < limit; row++ {
		match := true
		for _, c := range criteriaList {
			cellVal, err := ctx.getCellValue(c.refs[row])
			if err != nil || cellVal != c.val {
				match = false
				break
			}
		}
		if match {
			sumVal, err := ctx.getCellValue(sumRefs[row])
			if err == nil {
				total += sumVal
			}
		}
	}
	return total, nil
}

func evalFuncCountifs(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return 0, fmt.Errorf("COUNTIFS requires criteria_range,criteria pairs")
	}
	numPairs := len(args) / 2
	type criteria struct {
		refs []string
		val  float64
	}
	criteriaList := make([]criteria, numPairs)
	for i := 0; i < numPairs; i++ {
		crIdx := i * 2
		cvIdx := i*2 + 1
		refs, err := rangeOrCellRefs(ctx, args[crIdx])
		if err != nil {
			return 0, fmt.Errorf("COUNTIFS criteria_range %d: %w", i+1, err)
		}
		val, err := args[cvIdx].eval(ctx)
		if err != nil {
			return 0, err
		}
		criteriaList[i] = criteria{refs, val}
	}
	limit := len(criteriaList[0].refs)
	for _, c := range criteriaList[1:] {
		if len(c.refs) < limit {
			limit = len(c.refs)
		}
	}
	var count float64
	for row := 0; row < limit; row++ {
		match := true
		for _, c := range criteriaList {
			cellVal, err := ctx.getCellValue(c.refs[row])
			if err != nil || cellVal != c.val {
				match = false
				break
			}
		}
		if match {
			count++
		}
	}
	return count, nil
}

func evalFuncAverageifs(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 3 || len(args)%2 == 0 {
		return 0, fmt.Errorf("AVERAGEIFS requires avg_range + criteria_range,criteria pairs")
	}
	avgRefs, err := rangeOrCellRefs(ctx, args[0])
	if err != nil {
		return 0, fmt.Errorf("AVERAGEIFS avg_range: %w", err)
	}
	numPairs := (len(args) - 1) / 2
	type criteria struct {
		refs []string
		val  float64
	}
	criteriaList := make([]criteria, numPairs)
	for i := 0; i < numPairs; i++ {
		crIdx := 1 + i*2
		cvIdx := 1 + i*2 + 1
		refs, err := rangeOrCellRefs(ctx, args[crIdx])
		if err != nil {
			return 0, fmt.Errorf("AVERAGEIFS criteria_range %d: %w", i+1, err)
		}
		val, err := args[cvIdx].eval(ctx)
		if err != nil {
			return 0, err
		}
		criteriaList[i] = criteria{refs, val}
	}
	limit := len(avgRefs)
	for _, c := range criteriaList {
		if len(c.refs) < limit {
			limit = len(c.refs)
		}
	}
	var total float64
	var count float64
	for row := 0; row < limit; row++ {
		match := true
		for _, c := range criteriaList {
			cellVal, err := ctx.getCellValue(c.refs[row])
			if err != nil || cellVal != c.val {
				match = false
				break
			}
		}
		if match {
			avgVal, err := ctx.getCellValue(avgRefs[row])
			if err == nil {
				total += avgVal
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("AVERAGEIFS #DIV/0!")
	}
	return total / count, nil
}
