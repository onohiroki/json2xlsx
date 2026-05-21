package sheet2xlsx

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// buildStyles は JSON のスタイル定義群を excelize のスタイル ID に変換する。
// 戻り値は user id -> excelize style id のマップ。
func buildStyles(f *excelize.File, styles []Style) (map[int]int, error) {
	out := make(map[int]int, len(styles))
	for _, s := range styles {
		es, err := toExcelizeStyle(s)
		if err != nil {
			return nil, fmt.Errorf("style id=%d: %w", s.ID, err)
		}
		id, err := f.NewStyle(es)
		if err != nil {
			return nil, fmt.Errorf("style id=%d: %w", s.ID, err)
		}
		out[s.ID] = id
	}
	return out, nil
}

func toExcelizeStyle(s Style) (*excelize.Style, error) {
	es := &excelize.Style{}

	if s.Fill != nil {
		es.Fill = excelize.Fill{
			Type:    s.Fill.Type,
			Pattern: s.Fill.Pattern,
			Color:   stripHashes(s.Fill.Color),
		}
		if es.Fill.Type == "" {
			es.Fill.Type = "pattern"
		}
		if es.Fill.Pattern == 0 && len(es.Fill.Color) > 0 {
			es.Fill.Pattern = 1
		}
	}

	if len(s.Border) > 0 {
		bs, err := toBorders(s.Border)
		if err != nil {
			return nil, err
		}
		es.Border = bs
	}

	if s.Font != nil {
		es.Font = &excelize.Font{
			Family: s.Font.Name,
			Size:   s.Font.Size,
			Bold:   s.Font.Bold,
			Italic: s.Font.Italic,
			Color:  stripHash(s.Font.Color),
		}
	}

	if s.Alignment != nil {
		es.Alignment = &excelize.Alignment{
			Horizontal: s.Alignment.Horizontal,
			Vertical:   s.Alignment.Vertical,
			WrapText:   s.Alignment.WrapText,
		}
	}

	if s.NumFmt != "" {
		nf := s.NumFmt
		es.CustomNumFmt = &nf
	}

	return es, nil
}

var borderStyleMap = map[string]int{
	"thin":             1,
	"medium":           2,
	"dashed":           3,
	"dotted":           4,
	"thick":            5,
	"double":           6,
	"hair":             7,
	"mediumDashed":     8,
	"dashDot":          9,
	"mediumDashDot":    10,
	"dashDotDot":       11,
	"mediumDashDotDot": 12,
	"slantDashDot":     13,
}

func toBorders(in []Border) ([]excelize.Border, error) {
	out := make([]excelize.Border, 0, len(in)*4)
	sides := []string{"left", "right", "top", "bottom"}
	for _, b := range in {
		styleNum, ok := borderStyleMap[b.Style]
		if !ok {
			return nil, fmt.Errorf("unknown border style: %q", b.Style)
		}
		color := stripHash(b.Color)
		if b.Side == "" {
			for _, side := range sides {
				out = append(out, excelize.Border{Type: side, Color: color, Style: styleNum})
			}
		} else {
			out = append(out, excelize.Border{Type: b.Side, Color: color, Style: styleNum})
		}
	}
	return out, nil
}

func stripHash(c string) string {
	return strings.TrimPrefix(c, "#")
}

func stripHashes(cs []string) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = stripHash(c)
	}
	return out
}
