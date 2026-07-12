package json2xlsx

import (
	"testing"
)

func TestFormatValue_NumberInteger(t *testing.T) {
	got := formatValue(numVal(42), "0")
	if got != "42" {
		t.Errorf("format 0 = %q, want %q", got, "42")
	}
}

func TestFormatValue_NumberDecimal(t *testing.T) {
	got := formatValue(numVal(3.14159), "0.00")
	if got != "3.14" {
		t.Errorf("format 0.00 = %q, want %q", got, "3.14")
	}
}

func TestFormatValue_NumberThousands(t *testing.T) {
	got := formatValue(numVal(1234567), "#,##0")
	if got != "1,234,567" {
		t.Errorf("format #,##0 = %q, want %q", got, "1,234,567")
	}
}

func TestFormatValue_NumberThousandsDec(t *testing.T) {
	got := formatValue(numVal(1234567.89), "#,##0.00")
	if got != "1,234,567.89" {
		t.Errorf("format #,##0.00 = %q, want %q", got, "1,234,567.89")
	}
}

func TestFormatValue_NumberPercent(t *testing.T) {
	got := formatValue(numVal(0.25), "0.00%")
	if got != "25.00%" {
		t.Errorf("format 0.00%% = %q, want %q", got, "25.00%")
	}
}

func TestFormatValue_NumberPercentInt(t *testing.T) {
	got := formatValue(numVal(0.5), "0%")
	if got != "50%" {
		t.Errorf("format 0%% = %q, want %q", got, "50%")
	}
}

func TestFormatValue_NumberNegative(t *testing.T) {
	got := formatValue(numVal(-42), "0")
	if got != "-42" {
		t.Errorf("format negative = %q, want %q", got, "-42")
	}
}

func TestFormatValue_NumberZeroPadding(t *testing.T) {
	got := formatValue(numVal(5), "000")
	if got != "005" {
		t.Errorf("format zero pad = %q, want %q", got, "005")
	}
}

func TestFormatValue_NumberSectionPositive(t *testing.T) {
	got := formatValue(numVal(5), `0.00;(0.00)`)
	if got != "5.00" {
		t.Errorf("format section positive = %q, want %q", got, "5.00")
	}
}

func TestFormatValue_NumberSectionNegative(t *testing.T) {
	got := formatValue(numVal(-5), `0.00;(0.00)`)
	if got != "(5.00)" {
		t.Errorf("format section negative = %q, want %q", got, "(5.00)")
	}
}

func TestFormatValue_DateYMD(t *testing.T) {
	got := formatValue(numVal(45658), "yyyy/mm/dd")
	if got != "2025/01/01" {
		t.Errorf("format yyyy/mm/dd = %q, want %q", got, "2025/01/01")
	}
}

func TestFormatValue_DateYYMD(t *testing.T) {
	got := formatValue(numVal(45658), "yy/mm/dd")
	if got != "25/01/01" {
		t.Errorf("format yy/mm/dd = %q, want %q", got, "25/01/01")
	}
}

func TestFormatValue_DateMDY(t *testing.T) {
	got := formatValue(numVal(45658), "m/d/yyyy")
	if got != "1/1/2025" {
		t.Errorf("format m/d/yyyy = %q, want %q", got, "1/1/2025")
	}
}

func TestFormatValue_DateHyphen(t *testing.T) {
	got := formatValue(numVal(45658), "yyyy-mm-dd")
	if got != "2025-01-01" {
		t.Errorf("format yyyy-mm-dd = %q, want %q", got, "2025-01-01")
	}
}

func TestFormatValue_DateTime(t *testing.T) {
	got := formatValue(numVal(45658.5), "yyyy/mm/dd hh:mm:ss")
	if got != "2025/01/01 12:00:00" {
		t.Errorf("format datetime = %q, want %q", got, "2025/01/01 12:00:00")
	}
}

func TestFormatValue_DateMonthNames(t *testing.T) {
	got := formatValue(numVal(45658), "mmmm d, yyyy")
	if got != "January 1, 2025" {
		t.Errorf("format mmmm = %q, want %q", got, "January 1, 2025")
	}
}

func TestFormatValue_DateWeekday(t *testing.T) {
	got := formatValue(numVal(45658), "dddd")
	if got != "Wednesday" {
		t.Errorf("format dddd = %q, want %q", got, "Wednesday")
	}
}

func TestFormatValue_TextAt(t *testing.T) {
	got := formatValue(strVal("hello"), "@")
	if got != "hello" {
		t.Errorf("format @ = %q, want %q", got, "hello")
	}
}

func TestFormatValue_General(t *testing.T) {
	got := formatValue(numVal(42), "General")
	if got != "42" {
		t.Errorf("format General = %q, want %q", got, "42")
	}
}

func TestFormatValue_EmptyFormat(t *testing.T) {
	got := formatValue(numVal(42), "")
	if got != "42" {
		t.Errorf("format empty = %q, want %q", got, "42")
	}
}

func TestFormatValue_StringNumber(t *testing.T) {
	got := formatValue(strVal("hello"), "0.00")
	if got != "hello" {
		t.Errorf("format string as number = %q, want %q", got, "hello")
	}
}
