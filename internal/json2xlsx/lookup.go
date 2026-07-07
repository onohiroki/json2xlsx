package json2xlsx

import "fmt"

func evalFuncChoose(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("CHOOSE requires at least 2 arguments")
	}
	idxVal, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	idx := int(idxVal)
	if idx < 1 || idx >= len(args) {
		return 0, fmt.Errorf("CHOOSE index out of range")
	}
	return args[idx].eval(ctx)
}

func evalFuncVlookup(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 3 || len(args) > 4 {
		return 0, fmt.Errorf("VLOOKUP requires 3 or 4 arguments")
	}
	lookupVal, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	var start, end string
	switch a := args[1].(type) {
	case *rangeExpr:
		start, end = a.start, a.end
	case *cellRefExpr:
		start, end = a.ref, a.ref
	default:
		return 0, fmt.Errorf("VLOOKUP second argument must be a cell reference or range")
	}
	c1, r1 := parseCellRef(start)
	c2, r2 := parseCellRef(end)
	minCol, maxCol := c1, c2
	if c1 > c2 {
		minCol, maxCol = c2, c1
	}
	minRow, maxRow := r1, r2
	if r1 > r2 {
		minRow, maxRow = r2, r1
	}
	numCols := maxCol - minCol + 1
	numRows := maxRow - minRow + 1
	colIdxVal, err := args[2].eval(ctx)
	if err != nil {
		return 0, err
	}
	colIdx := int(colIdxVal)
	if colIdx < 1 || colIdx > numCols {
		return 0, fmt.Errorf("VLOOKUP column index out of range")
	}
	if len(args) == 4 {
		approx, err := args[3].eval(ctx)
		if err != nil {
			return 0, err
		}
		if approx != 0 {
			return 0, fmt.Errorf("VLOOKUP approximate match not yet supported")
		}
	}
	refs := expandRange(start, end)
	for r := 0; r < numRows; r++ {
		cellVal, err := ctx.getCellValue(refs[r])
		if err != nil {
			continue
		}
		if cellVal == lookupVal {
			return ctx.getCellValue(refs[(colIdx-1)*numRows+r])
		}
	}
	return 0, fmt.Errorf("VLOOKUP value not found")
}

func evalFuncMatch(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args) > 3 {
		return 0, fmt.Errorf("MATCH requires 2 or 3 arguments")
	}
	lookupVal, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	refs, err := rangeOrCellRefs(ctx, args[1])
	if err != nil {
		return 0, fmt.Errorf("MATCH second argument: %w", err)
	}
	matchType := 0
	if len(args) == 3 {
		matchTypeVal, err := args[2].eval(ctx)
		if err != nil {
			return 0, err
		}
		matchType = int(matchTypeVal)
	}
	switch matchType {
	case 0:
		for i, ref := range refs {
			cellVal, err := ctx.getCellValue(ref)
			if err != nil {
				continue
			}
			if cellVal == lookupVal {
				return float64(i + 1), nil
			}
		}
		return 0, fmt.Errorf("MATCH value not found")
	case 1, -1:
		return 0, fmt.Errorf("MATCH approximate match not yet supported")
	default:
		return 0, fmt.Errorf("MATCH invalid match_type")
	}
}

func evalFuncIndex(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args) > 3 {
		return 0, fmt.Errorf("INDEX requires 2 or 3 arguments")
	}
	var start, end string
	switch a := args[0].(type) {
	case *rangeExpr:
		start, end = a.start, a.end
	case *cellRefExpr:
		start, end = a.ref, a.ref
	default:
		return 0, fmt.Errorf("INDEX first argument must be a cell reference or range")
	}
	rowNumVal, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	rowNum := int(rowNumVal)
	colNum := 1
	if len(args) == 3 {
		colNumVal, err := args[2].eval(ctx)
		if err != nil {
			return 0, err
		}
		colNum = int(colNumVal)
	}
	c1, r1 := parseCellRef(start)
	c2, r2 := parseCellRef(end)
	minCol, maxCol := c1, c2
	if c1 > c2 {
		minCol, maxCol = c2, c1
	}
	minRow, maxRow := r1, r2
	if r1 > r2 {
		minRow, maxRow = r2, r1
	}
	numCols := maxCol - minCol + 1
	numRows := maxRow - minRow + 1
	if rowNum < 1 || rowNum > numRows {
		return 0, fmt.Errorf("INDEX row out of range")
	}
	if colNum < 1 || colNum > numCols {
		return 0, fmt.Errorf("INDEX column out of range")
	}
	refs := expandRange(start, end)
	return ctx.getCellValue(refs[(colNum-1)*numRows+(rowNum-1)])
}

func evalFuncXlookup(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 3 || len(args) > 6 {
		return 0, fmt.Errorf("XLOOKUP requires 3 to 6 arguments")
	}
	lookupVal, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	lookupRefs, err := rangeOrCellRefs(ctx, args[1])
	if err != nil {
		return 0, fmt.Errorf("XLOOKUP second argument: %w", err)
	}
	returnRefs, err := rangeOrCellRefs(ctx, args[2])
	if err != nil {
		return 0, fmt.Errorf("XLOOKUP third argument: %w", err)
	}
	if len(lookupRefs) != len(returnRefs) {
		return 0, fmt.Errorf("XLOOKUP lookup and return arrays must be the same size")
	}
	for i, ref := range lookupRefs {
		cellVal, err := ctx.getCellValue(ref)
		if err != nil {
			continue
		}
		if cellVal == lookupVal {
			return ctx.getCellValue(returnRefs[i])
		}
	}
	if len(args) >= 4 {
		return args[3].eval(ctx)
	}
	return 0, fmt.Errorf("XLOOKUP value not found")
}
