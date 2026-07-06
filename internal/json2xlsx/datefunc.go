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
