package json2xlsx

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Step 4: 基本文字列関数
// ---------------------------------------------------------------------------

func TestEval_Concat(t *testing.T) {
	got := evalFormulaStr(t, nil, `CONCAT("Hello"," ","World")`)
	if got != "Hello World" {
		t.Errorf("CONCAT = %q, want %q", got, "Hello World")
	}
}

func TestEval_ConcatEmpty(t *testing.T) {
	got := evalFormulaStr(t, nil, `CONCAT("")`)
	if got != "" {
		t.Errorf("CONCAT empty = %q, want empty", got)
	}
}

func TestEval_ConcatNoArgs(t *testing.T) {
	got := evalFormulaStr(t, nil, `CONCAT()`)
	if got != "" {
		t.Errorf("CONCAT no args = %q, want empty", got)
	}
}

func TestEval_ConcatWithNumbers(t *testing.T) {
	got := evalFormulaStr(t, nil, `CONCAT(100,"円")`)
	if got != "100円" {
		t.Errorf("CONCAT with number = %q, want %q", got, "100円")
	}
}

func TestEval_Concatenate(t *testing.T) {
	got := evalFormulaStr(t, nil, `CONCATENATE("A","B","C")`)
	if got != "ABC" {
		t.Errorf("CONCATENATE = %q, want %q", got, "ABC")
	}
}

func TestEval_Left(t *testing.T) {
	got := evalFormulaStr(t, nil, `LEFT("Hello",2)`)
	if got != "He" {
		t.Errorf("LEFT = %q, want %q", got, "He")
	}
}

func TestEval_LeftJapanese(t *testing.T) {
	got := evalFormulaStr(t, nil, `LEFT("あいうえお",3)`)
	if got != "あいう" {
		t.Errorf("LEFT Japanese = %q, want %q", got, "あいう")
	}
}

func TestEval_LeftAll(t *testing.T) {
	got := evalFormulaStr(t, nil, `LEFT("Hello",10)`)
	if got != "Hello" {
		t.Errorf("LEFT all = %q, want %q", got, "Hello")
	}
}

func TestEval_LeftZero(t *testing.T) {
	got := evalFormulaStr(t, nil, `LEFT("Hello",0)`)
	if got != "" {
		t.Errorf("LEFT zero = %q, want empty", got)
	}
}

func TestEval_LeftWrongArgCount(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, `LEFT("Hello")`)
	if !strings.Contains(errMsg, "requires 2") {
		t.Errorf("expected arg count error, got %q", errMsg)
	}
}

func TestEval_Right(t *testing.T) {
	got := evalFormulaStr(t, nil, `RIGHT("Hello",2)`)
	if got != "lo" {
		t.Errorf("RIGHT = %q, want %q", got, "lo")
	}
}

func TestEval_RightJapanese(t *testing.T) {
	got := evalFormulaStr(t, nil, `RIGHT("あいうえお",2)`)
	if got != "えお" {
		t.Errorf("RIGHT Japanese = %q, want %q", got, "えお")
	}
}

func TestEval_RightAll(t *testing.T) {
	got := evalFormulaStr(t, nil, `RIGHT("Hello",10)`)
	if got != "Hello" {
		t.Errorf("RIGHT all = %q, want %q", got, "Hello")
	}
}

func TestEval_RightZero(t *testing.T) {
	got := evalFormulaStr(t, nil, `RIGHT("Hello",0)`)
	if got != "" {
		t.Errorf("RIGHT zero = %q, want empty", got)
	}
}

func TestEval_Mid(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("Hello",2,3)`)
	if got != "ell" {
		t.Errorf("MID = %q, want %q", got, "ell")
	}
}

func TestEval_MidJapanese(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("あいうえお",2,3)`)
	if got != "いうえ" {
		t.Errorf("MID Japanese = %q, want %q", got, "いうえ")
	}
}

func TestEval_MidStartAtOne(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("Hello",1,3)`)
	if got != "Hel" {
		t.Errorf("MID start=1 = %q, want %q", got, "Hel")
	}
}

func TestEval_MidBeyondString(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("Hello",4,10)`)
	if got != "lo" {
		t.Errorf("MID beyond = %q, want %q", got, "lo")
	}
}

func TestEval_MidStartBeyondString(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("Hello",10,3)`)
	if got != "" {
		t.Errorf("MID start beyond = %q, want empty", got)
	}
}

func TestEval_MidNegativeStart(t *testing.T) {
	got := evalFormulaStr(t, nil, `MID("Hello",-1,3)`)
	if got != "Hel" {
		t.Errorf("MID negative start = %q, want %q", got, "Hel")
	}
}

func TestEval_Len(t *testing.T) {
	got := evalFormula(t, nil, `LEN("Hello")`)
	if got != 5 {
		t.Errorf("LEN = %v, want 5", got)
	}
}

func TestEval_LenJapanese(t *testing.T) {
	got := evalFormula(t, nil, `LEN("あいうえお")`)
	if got != 5 {
		t.Errorf("LEN Japanese = %v, want 5", got)
	}
}

func TestEval_LenEmpty(t *testing.T) {
	got := evalFormula(t, nil, `LEN("")`)
	if got != 0 {
		t.Errorf("LEN empty = %v, want 0", got)
	}
}

func TestEval_LenNumber(t *testing.T) {
	got := evalFormula(t, nil, `LEN(12345)`)
	if got != 5 {
		t.Errorf("LEN number = %v, want 5", got)
	}
}

func TestEval_Upper(t *testing.T) {
	got := evalFormulaStr(t, nil, `UPPER("Hello World")`)
	if got != "HELLO WORLD" {
		t.Errorf("UPPER = %q, want %q", got, "HELLO WORLD")
	}
}

func TestEval_UpperJapanese(t *testing.T) {
	// Japanese characters are not affected by case conversion
	got := evalFormulaStr(t, nil, `UPPER("hello")`)
	if got != "HELLO" {
		t.Errorf("UPPER = %q, want %q", got, "HELLO")
	}
}

func TestEval_Lower(t *testing.T) {
	got := evalFormulaStr(t, nil, `LOWER("Hello World")`)
	if got != "hello world" {
		t.Errorf("LOWER = %q, want %q", got, "hello world")
	}
}

func TestEval_LowerAlreadyLower(t *testing.T) {
	got := evalFormulaStr(t, nil, `LOWER("already")`)
	if got != "already" {
		t.Errorf("LOWER = %q, want %q", got, "already")
	}
}

func TestEval_Trim(t *testing.T) {
	got := evalFormulaStr(t, nil, `TRIM("  Hello   World  ")`)
	if got != "Hello World" {
		t.Errorf("TRIM = %q, want %q", got, "Hello World")
	}
}

func TestEval_TrimNoChange(t *testing.T) {
	got := evalFormulaStr(t, nil, `TRIM("Hello World")`)
	if got != "Hello World" {
		t.Errorf("TRIM no change = %q, want %q", got, "Hello World")
	}
}

func TestEval_TrimEmpty(t *testing.T) {
	got := evalFormulaStr(t, nil, `TRIM("")`)
	if got != "" {
		t.Errorf("TRIM empty = %q, want empty", got)
	}
}

func TestEval_TrimTabsAndNewlines(t *testing.T) {
	got := evalFormulaStr(t, nil, "TRIM(\"\tHello\n  World \")")
	if got != "Hello World" {
		t.Errorf("TRIM tabs = %q, want %q", got, "Hello World")
	}
}

func TestEval_ConcatWithCellRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "Hello"},
		"A2": {T: "s", V: "World"},
	}
	got := evalFormulaStr(t, cells, `CONCAT(A1," ",A2)`)
	if got != "Hello World" {
		t.Errorf("CONCAT with cell ref = %q, want %q", got, "Hello World")
	}
}

func TestEval_LenWithCellRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "あいう"},
	}
	got := evalFormula(t, cells, `LEN(A1)`)
	if got != 3 {
		t.Errorf("LEN cell ref = %v, want 3", got)
	}
}

func TestEval_LeftWithCellRef(t *testing.T) {
	cells := map[string]Cell{
		"A1": {T: "s", V: "Hello World"},
	}
	got := evalFormulaStr(t, cells, `LEFT(A1,5)`)
	if got != "Hello" {
		t.Errorf("LEFT cell ref = %q, want %q", got, "Hello")
	}
}

// ---------------------------------------------------------------------------
// FIND
// ---------------------------------------------------------------------------

func TestEval_FindBasic(t *testing.T) {
	got := evalFormula(t, nil, `FIND("is","this is a test")`)
	if got != 3 {
		t.Errorf("FIND basic = %v, want 3", got)
	}
}

func TestEval_FindWithStartNum(t *testing.T) {
	got := evalFormula(t, nil, `FIND("is","this is a test",5)`)
	if got != 6 {
		t.Errorf("FIND start_num = %v, want 6", got)
	}
}

func TestEval_FindJapanese(t *testing.T) {
	got := evalFormula(t, nil, `FIND("い","あいうえお")`)
	if got != 2 {
		t.Errorf("FIND Japanese = %v, want 2", got)
	}
}

func TestEval_FindCaseSensitive(t *testing.T) {
	got1 := evalFormula(t, nil, `FIND("A","ABCabc")`)
	if got1 != 1 {
		t.Errorf("FIND 'A' = %v, want 1", got1)
	}
	got2 := evalFormula(t, nil, `FIND("a","ABCabc")`)
	if got2 != 4 {
		t.Errorf("FIND 'a' = %v, want 4", got2)
	}
}

func TestEval_FindNotFound(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, `FIND("x","abc")`)
	if errMsg == "" || !strings.Contains(errMsg, "not found") {
		t.Errorf("FIND not found expected error, got %q", errMsg)
	}
}

func TestEval_FindStartNumOutOfRange(t *testing.T) {
	errMsg := evalFormulaErr(t, nil, `FIND("a","abc",10)`)
	if errMsg == "" || !strings.Contains(errMsg, "out of range") {
		t.Errorf("FIND start_num out of range expected, got %q", errMsg)
	}
}

// ---------------------------------------------------------------------------
// TEXT (integration — detailed format tests in textformat_test.go)
// ---------------------------------------------------------------------------

func TestEval_TextNumber(t *testing.T) {
	got := evalFormulaStr(t, nil, `TEXT(1234.567,"#,##0.00")`)
	want := "1,234.57"
	if got != want {
		t.Errorf("TEXT number = %q, want %q", got, want)
	}
}

func TestEval_TextPercent(t *testing.T) {
	got := evalFormulaStr(t, nil, `TEXT(0.25,"0.00%")`)
	want := "25.00%"
	if got != want {
		t.Errorf("TEXT percent = %q, want %q", got, want)
	}
}

func TestEval_TextDate(t *testing.T) {
	// 45658 = 2025-01-01
	got := evalFormulaStr(t, nil, `TEXT(45658,"yyyy/mm/dd")`)
	want := "2025/01/01"
	if got != want {
		t.Errorf("TEXT date = %q, want %q", got, want)
	}
}

func TestEval_TextDateTime(t *testing.T) {
	// 45658.5 = 2025-01-01 12:00:00
	got := evalFormulaStr(t, nil, `TEXT(45658.5,"yyyy/mm/dd hh:mm:ss")`)
	want := "2025/01/01 12:00:00"
	if got != want {
		t.Errorf("TEXT datetime = %q, want %q", got, want)
	}
}

func TestEval_TextAt(t *testing.T) {
	got := evalFormulaStr(t, nil, `TEXT("hello","@")`)
	want := "hello"
	if got != want {
		t.Errorf("TEXT @ = %q, want %q", got, want)
	}
}

func TestEval_TextGeneral(t *testing.T) {
	got := evalFormulaStr(t, nil, `TEXT(123,"General")`)
	want := "123"
	if got != want {
		t.Errorf("TEXT General = %q, want %q", got, want)
	}
}

func TestEval_TextEmpty(t *testing.T) {
	got := evalFormulaStr(t, nil, `TEXT(42,"")`)
	want := "42"
	if got != want {
		t.Errorf("TEXT empty format = %q, want %q", got, want)
	}
}
