package json2xlsx

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// condFmtTypeMap は JSON の type 名を excelize 内部の type 名に変換する．
var condFmtTypeMap = map[string]string{
	"cell":        "cell",
	"formula":     "formula",
	"2_color_scale": "2_color_scale",
	"3_color_scale": "3_color_scale",
	"data_bar":    "data_bar",
	"iconSet":     "icon_set",
	"top":         "top",
	"bottom":      "bottom",
	"average":     "average",
	"duplicate":   "duplicate",
	"unique":      "unique",
	"blanks":      "blanks",
	"no_blanks":   "no_blanks",
	"errors":      "errors",
	"no_errors":   "no_errors",
	"text":        "text",
	"time_period": "time_period",
}

// buildConditionalFormatOpts は ConditionalRule のスライスを
// excelize.ConditionalFormatOptions のスライスに変換する．
func buildConditionalFormatOpts(f *excelize.File, rules []ConditionalRule) ([]excelize.ConditionalFormatOptions, error) {
	opts := make([]excelize.ConditionalFormatOptions, len(rules))
	for i, rule := range rules {
		xlType := rule.Type
		if mapped, ok := condFmtTypeMap[rule.Type]; ok {
			xlType = mapped
		}
		opt := excelize.ConditionalFormatOptions{
			Type:           xlType,
			Criteria:       rule.Criteria,
			Value:          rule.Value,
			MinValue:       rule.MinValue,
			MidValue:       rule.MidValue,
			MaxValue:       rule.MaxValue,
			MinType:        rule.MinType,
			MidType:        rule.MidType,
			MaxType:        rule.MaxType,
			MinColor:       stripHash(rule.MinColor),
			MidColor:       stripHash(rule.MidColor),
			MaxColor:       stripHash(rule.MaxColor),
			BarColor:       stripHash(rule.BarColor),
			BarBorderColor: stripHash(rule.BarBorderColor),
			BarDirection:   rule.BarDirection,
			BarOnly:        rule.BarOnly,
			BarSolid:       rule.BarSolid,
			IconStyle:      rule.IconStyle,
			ReverseIcons:   rule.ReverseIcons,
			IconsOnly:      rule.IconsOnly,
			Percent:        rule.Percent,
			StopIfTrue:     rule.StopIfTrue,
		}
		if rule.AboveAverage != nil {
			opt.AboveAverage = *rule.AboveAverage
		}
		if rule.Style != nil {
			es, err := toExcelizeStyle(*rule.Style)
			if err != nil {
				return nil, fmt.Errorf("rule %d: %w", i, err)
			}
			idx, err := f.NewConditionalStyle(es)
			if err != nil {
				return nil, fmt.Errorf("rule %d: %w", i, err)
			}
			opt.Format = idx
		}
		opts[i] = opt
	}
	return opts, nil
}
