package json2xlsx

import (
	"encoding/json"
	"testing"
)

func TestScalarToString_Nil(t *testing.T) {
	if got := scalarToString(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestScalarToString_String(t *testing.T) {
	if got := scalarToString("hello"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestScalarToString_Bool(t *testing.T) {
	if got := scalarToString(true); got != "true" {
		t.Fatalf("expected true, got %q", got)
	}
	if got := scalarToString(false); got != "false" {
		t.Fatalf("expected false, got %q", got)
	}
}

func TestScalarToString_Float64(t *testing.T) {
	if got := scalarToString(float64(42)); got != "42" {
		t.Fatalf("expected 42, got %q", got)
	}
	if got := scalarToString(float64(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %q", got)
	}
}

func TestScalarToString_IntTypes(t *testing.T) {
	if got := scalarToString(42); got != "42" {
		t.Fatalf("expected 42, got %q", got)
	}
	if got := scalarToString(int64(99)); got != "99" {
		t.Fatalf("expected 99, got %q", got)
	}
	if got := scalarToString(float32(1.5)); got != "1.5" {
		t.Fatalf("expected 1.5, got %q", got)
	}
}

func TestScalarToString_JSONNumber(t *testing.T) {
	jn := json.Number("12345")
	if got := scalarToString(jn); got != "12345" {
		t.Fatalf("expected 12345, got %q", got)
	}
}

func TestScalarToString_UnknownType(t *testing.T) {
	type custom struct{ X int }
	got := scalarToString(custom{X: 5})
	if got != "{5}" {
		t.Fatalf("expected {5}, got %q", got)
	}
}

func TestToFloat64_Nil(t *testing.T) {
	if got := toFloat64(nil); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestToFloat64_NumericTypes(t *testing.T) {
	if got := toFloat64(float64(3.14)); got != 3.14 {
		t.Fatalf("expected 3.14, got %f", got)
	}
	if got := toFloat64(float32(2.5)); got != 2.5 {
		t.Fatalf("expected 2.5, got %f", got)
	}
	if got := toFloat64(42); got != 42 {
		t.Fatalf("expected 42, got %f", got)
	}
	if got := toFloat64(int64(99)); got != 99 {
		t.Fatalf("expected 99, got %f", got)
	}
	if got := toFloat64("123.45"); got != 123.45 {
		t.Fatalf("expected 123.45, got %f", got)
	}
}

func TestToFloat64_JSONNumber(t *testing.T) {
	jn := json.Number("456.78")
	if got := toFloat64(jn); got != 456.78 {
		t.Fatalf("expected 456.78, got %f", got)
	}
}

func TestToFloat64_UnparseableString(t *testing.T) {
	if got := toFloat64("not-a-number"); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}

func TestToFloat64_UnknownType(t *testing.T) {
	if got := toFloat64(struct{}{}); got != 0 {
		t.Fatalf("expected 0, got %f", got)
	}
}
