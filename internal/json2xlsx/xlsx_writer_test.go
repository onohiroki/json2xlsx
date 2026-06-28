package json2xlsx

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseLink_String(t *testing.T) {
	target, tooltip := parseLink("https://example.com")
	if target != "https://example.com" {
		t.Errorf("target = %q, want %q", target, "https://example.com")
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestParseLink_Map(t *testing.T) {
	l := map[string]interface{}{
		"target":  "https://example.com",
		"tooltip": "Click me",
	}
	target, tooltip := parseLink(l)
	if target != "https://example.com" {
		t.Errorf("target = %q, want %q", target, "https://example.com")
	}
	if tooltip != "Click me" {
		t.Errorf("tooltip = %q, want %q", tooltip, "Click me")
	}
}

func TestParseLink_MapNoTooltip(t *testing.T) {
	l := map[string]interface{}{
		"target": "https://example.com",
	}
	target, tooltip := parseLink(l)
	if target != "https://example.com" {
		t.Errorf("target = %q, want %q", target, "https://example.com")
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestParseLink_MapNoTarget(t *testing.T) {
	l := map[string]interface{}{
		"tooltip": "Click me",
	}
	target, tooltip := parseLink(l)
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if tooltip != "Click me" {
		t.Errorf("tooltip = %q, want %q", tooltip, "Click me")
	}
}

func TestParseLink_MapEmpty(t *testing.T) {
	target, tooltip := parseLink(map[string]interface{}{})
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestParseLink_Nil(t *testing.T) {
	target, tooltip := parseLink(nil)
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestParseLink_UnsupportedType(t *testing.T) {
	target, tooltip := parseLink(42)
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestParseLink_MapWrongTypes(t *testing.T) {
	l := map[string]interface{}{
		"target":  123,
		"tooltip": true,
	}
	target, tooltip := parseLink(l)
	if target != "" {
		t.Errorf("target = %q, want empty", target)
	}
	if tooltip != "" {
		t.Errorf("tooltip = %q, want empty", tooltip)
	}
}

func TestMergeStyleWithNumFmt_OverridesNumFmt(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	styles := []Style{
		{ID: 1, NumFmt: "#,##0"},
	}
	styleMap, err := buildStyles(f, styles)
	if err != nil {
		t.Fatalf("buildStyles: %v", err)
	}
	origID := styleMap[1]

	newID, err := mergeStyleWithNumFmt(f, styles, 1, "0.00")
	if err != nil {
		t.Fatalf("mergeStyleWithNumFmt: %v", err)
	}
	if newID == 0 {
		t.Fatal("mergeStyleWithNumFmt returned 0")
	}
	if newID == origID {
		t.Error("mergeStyleWithNumFmt returned same style ID as original (expected new style)")
	}
}

func TestMergeStyleWithNumFmt_FallbackWhenStyleNotFound(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	id, err := mergeStyleWithNumFmt(f, []Style{}, 999, "0.00")
	if err != nil {
		t.Fatalf("mergeStyleWithNumFmt: %v", err)
	}
	if id == 0 {
		t.Fatal("mergeStyleWithNumFmt returned 0 on fallback")
	}
}
