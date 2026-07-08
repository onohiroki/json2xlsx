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
	v, err := args[0].eval(ctx)
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
	v, err := args[0].eval(ctx)
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
	v, err := args[0].eval(ctx)
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
	v, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	returnType := 1.0
	if len(args) == 2 {
		returnType, err = args[1].eval(ctx)
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

func evalFuncWeeknum(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 1 || len(args) > 2 {
		return 0, fmt.Errorf("WEEKNUM requires 1 or 2 arguments")
	}
	v, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	returnType := 1.0
	if len(args) == 2 {
		returnType, err = args[1].eval(ctx)
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
