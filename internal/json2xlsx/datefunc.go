package json2xlsx

import (
	"fmt"
	"math"
	"time"
)

func timeToExcelSerial(t time.Time) float64 {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := t.Sub(epoch).Hours() / 24
	return days
}

func excelSerialToTime(serial float64) time.Time {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := math.Floor(serial)
	return epoch.Add(time.Duration(days) * 24 * time.Hour)
}

// excelSerialToDateTime はシリアル値から time.Time への変換．小数部（時刻）も保持する．
func excelSerialToDateTime(serial float64) time.Time {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := math.Floor(serial)
	frac := serial - days
	return epoch.Add(time.Duration(days)*24*time.Hour + time.Duration(frac*86400*1e9))
}

func evalFuncToday(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 0 {
		return 0, fmt.Errorf("TODAY requires no arguments")
	}
	return math.Floor(timeToExcelSerial(time.Now())), nil
}

func evalFuncNow(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 0 {
		return 0, fmt.Errorf("NOW requires no arguments")
	}
	return timeToExcelSerial(time.Now()), nil
}

func evalFuncYear(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("YEAR requires exactly 1 argument")
	}
	v, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	t := excelSerialToTime(v)
	return float64(t.Year()), nil
}

func evalFuncMonth(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("MONTH requires exactly 1 argument")
	}
	v, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	t := excelSerialToTime(v)
	return float64(t.Month()), nil
}

func evalFuncDay(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("DAY requires exactly 1 argument")
	}
	v, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	t := excelSerialToTime(v)
	return float64(t.Day()), nil
}

func evalFuncWeekday(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 1 || len(args) > 2 {
		return 0, fmt.Errorf("WEEKDAY requires 1 or 2 arguments")
	}
	v, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	returnType := 1.0
	if len(args) == 2 {
		returnType, err = ctx.evalArgNum(args[1])
		if err != nil {
			return 0, err
		}
	}
	t := excelSerialToTime(v)
	dow := t.Weekday()
	switch int(returnType) {
	case 1:
		return float64(dow + 1), nil
	case 2:
		return float64((int(dow)+6)%7 + 1), nil
	case 3:
		return float64((int(dow)+6)%7), nil
	default:
		return 0, fmt.Errorf("WEEKDAY unsupported return_type: %v", returnType)
	}
}

func evalFuncDate(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 3 {
		return 0, fmt.Errorf("DATE requires exactly 3 arguments")
	}
	year, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	month, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	day, err := ctx.evalArgNum(args[2])
	if err != nil {
		return 0, err
	}
	t := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
	return timeToExcelSerial(t), nil
}

func evalFuncEdate(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("EDATE requires exactly 2 arguments")
	}
	start, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	months, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	t := excelSerialToTime(start)
	y, m, d := t.Year(), int(t.Month()), t.Day()
	total := m + int(months)
	lastDay := time.Date(y, time.Month(total+1), 0, 0, 0, 0, 0, time.UTC).Day()
	if d > lastDay {
		d = lastDay
	}
	end := time.Date(y, time.Month(total), d, 0, 0, 0, 0, time.UTC)
	return timeToExcelSerial(end), nil
}

func evalFuncEomonth(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("EOMONTH requires exactly 2 arguments")
	}
	start, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	months, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	t := excelSerialToTime(start)
	y, m := t.Year(), t.Month()
	total := int(m) + int(months)
	end := time.Date(y, time.Month(total+1), 0, 0, 0, 0, 0, time.UTC)
	return timeToExcelSerial(end), nil
}

func isWeekend(serial float64) bool {
	t := excelSerialToTime(serial)
	dow := t.Weekday()
	return dow == time.Saturday || dow == time.Sunday
}

func nextWorkday(cur float64, dir float64) float64 {
	for {
		cur += dir
		if !isWeekend(cur) {
			return cur
		}
	}
}

func collectHolidays(ctx *evalContext, args []expr) ([]float64, error) {
	var holidays []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return nil, err
		}
		holidays = append(holidays, vals...)
	}
	return holidays, nil
}

func isHoliday(serial float64, holidays []float64) bool {
	for _, h := range holidays {
		if math.Floor(serial) == math.Floor(h) {
			return true
		}
	}
	return false
}

func evalFuncDays(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("DAYS requires exactly 2 arguments")
	}
	end, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	start, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	return math.Floor(end) - math.Floor(start), nil
}

func evalFuncNetworkdays(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("NETWORKDAYS requires at least 2 arguments")
	}
	start, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	end, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	var holidays []float64
	if len(args) > 2 {
		holidays, err = collectHolidays(ctx, args[2:])
		if err != nil {
			return 0, err
		}
	}
	startDay := math.Floor(start)
	endDay := math.Floor(end)
	dir := 1.0
	if startDay > endDay {
		dir = -1
	}
	var count float64
	for d := startDay; ; d += dir {
		if !isWeekend(d) && !isHoliday(d, holidays) {
			count++
		}
		if d == endDay {
			break
		}
	}
	return count, nil
}

func evalFuncWorkday(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("WORKDAY requires at least 2 arguments")
	}
	start, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	days, err := ctx.evalArgNum(args[1])
	if err != nil {
		return 0, err
	}
	var holidays []float64
	if len(args) > 2 {
		holidays, err = collectHolidays(ctx, args[2:])
		if err != nil {
			return 0, err
		}
	}
	cur := math.Floor(start)
	dir := 1.0
	if days < 0 {
		dir = -1
		days = -days
	}
	for days > 0 {
		cur += dir
		if !isWeekend(cur) && !isHoliday(cur, holidays) {
			days--
		}
	}
	return cur, nil
}

func evalFuncWeeknum(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 1 || len(args) > 2 {
		return 0, fmt.Errorf("WEEKNUM requires 1 or 2 arguments")
	}
	v, err := ctx.evalArgNum(args[0])
	if err != nil {
		return 0, err
	}
	returnType := 1.0
	if len(args) == 2 {
		returnType, err = ctx.evalArgNum(args[1])
		if err != nil {
			return 0, err
		}
	}
	t := excelSerialToTime(v)
	switch int(returnType) {
	case 1:
		_, week := t.ISOWeek()
		return float64(week), nil
	case 2:
		_, week := t.ISOWeek()
		return float64(week), nil
	default:
		return 0, fmt.Errorf("WEEKNUM unsupported return_type: %v", returnType)
	}
}
