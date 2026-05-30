package sheet2xlsx

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// styleCollector は excelize style ID から JSON Style.ID へのキャッシュ。
type styleCollector struct {
	byExcelizeID map[int]int
	dateByID     map[int]bool
	styles       []Style
	nextID       int
}

func newStyleCollector() *styleCollector {
	return &styleCollector{
		byExcelizeID: map[int]int{},
		dateByID:     map[int]bool{},
		nextID:       1,
	}
}

// collect は excelize の style ID を JSON Style.ID に変換 (キャッシュ) し、
// 当該スタイルが日付書式かどうかも返す。
func (sc *styleCollector) collect(f *excelize.File, excelizeID int) (int, bool, error) {
	if id, ok := sc.byExcelizeID[excelizeID]; ok {
		return id, sc.dateByID[id], nil
	}
	es, err := f.GetStyle(excelizeID)
	if err != nil {
		return 0, false, err
	}

	js := Style{ID: sc.nextID}
	sc.nextID++

	// Fill
	if es.Fill.Type != "" || len(es.Fill.Color) > 0 || es.Fill.Pattern != 0 {
		colors := make([]string, 0, len(es.Fill.Color))
		for _, c := range es.Fill.Color {
			colors = append(colors, addHash(c))
		}
		js.Fill = &Fill{
			Type:    es.Fill.Type,
			Pattern: es.Fill.Pattern,
			Color:   colors,
		}
	}

	// Border
	for _, b := range es.Border {
		if b.Style == 0 {
			continue
		}
		js.Border = append(js.Border, Border{
			Style: borderStyleName(b.Style),
			Color: addHash(b.Color),
			Side:  b.Type,
		})
	}

	// Font
	if es.Font != nil && (es.Font.Family != "" || es.Font.Size != 0 || es.Font.Bold || es.Font.Italic || es.Font.Color != "") {
		js.Font = &Font{
			Name:   es.Font.Family,
			Size:   es.Font.Size,
			Bold:   es.Font.Bold,
			Italic: es.Font.Italic,
			Color:  addHash(es.Font.Color),
		}
	}

	// Alignment
	if es.Alignment != nil && (es.Alignment.Horizontal != "" || es.Alignment.Vertical != "" || es.Alignment.WrapText) {
		js.Alignment = &Alignment{
			Horizontal: es.Alignment.Horizontal,
			Vertical:   es.Alignment.Vertical,
			WrapText:   es.Alignment.WrapText,
		}
	}

	// NumFmt
	numFmt := ""
	if es.CustomNumFmt != nil && *es.CustomNumFmt != "" {
		numFmt = *es.CustomNumFmt
	} else if code, ok := builtInNumFmtCode[es.NumFmt]; ok {
		numFmt = code
	}
	js.NumFmt = numFmt

	isDate := isDateFormat(numFmt, es.NumFmt)

	sc.styles = append(sc.styles, js)
	sc.byExcelizeID[excelizeID] = js.ID
	sc.dateByID[js.ID] = isDate
	return js.ID, isDate, nil
}

// borderStyleName は excelize の数値 style を文字列名に逆引きする。
func borderStyleName(n int) string {
	for name, v := range borderStyleMap {
		if v == n {
			return name
		}
	}
	return ""
}

func addHash(c string) string {
	if c == "" {
		return ""
	}
	if strings.HasPrefix(c, "#") {
		return c
	}
	// Strip AA prefix from 8-digit AARRGGBB to keep only 6-digit RRGGBB.
	// Excelize の getPaletteColor は常に FF を先頭に付加するため、
	// 8桁の色をそのまま渡すと FF004472C4 のように10桁化して黒になる。
	if len(c) == 8 {
		c = c[2:]
	}
	return "#" + c
}

// builtInNumFmtCode は Excel ビルトイン書式 ID から書式コードへの最小マッピング。
// 日付/時刻判定に必要なものを中心に網羅する。
var builtInNumFmtCode = map[int]string{
	0:  "General",
	1:  "0",
	2:  "0.00",
	3:  "#,##0",
	4:  "#,##0.00",
	9:  "0%",
	10: "0.00%",
	11: "0.00E+00",
	12: "# ?/?",
	13: "# ??/??",
	14: "m/d/yy",
	15: "d-mmm-yy",
	16: "d-mmm",
	17: "mmm-yy",
	18: "h:mm AM/PM",
	19: "h:mm:ss AM/PM",
	20: "h:mm",
	21: "h:mm:ss",
	22: "m/d/yy h:mm",
	37: "#,##0 ;(#,##0)",
	38: "#,##0 ;[Red](#,##0)",
	39: "#,##0.00;(#,##0.00)",
	40: "#,##0.00;[Red](#,##0.00)",
	45: "mm:ss",
	46: "[h]:mm:ss",
	47: "mmss.0",
	48: "##0.0E+0",
	49: "@",
}

// isDateFormat は numFmt コードまたはビルトイン ID から日付書式かを判定する。
func isDateFormat(code string, numFmtID int) bool {
	switch numFmtID {
	case 14, 15, 16, 17, 18, 19, 20, 21, 22, 45, 46, 47:
		return true
	}
	if code == "" {
		return false
	}
	// "General" や "@" は除外
	if code == "General" || code == "@" {
		return false
	}
	// y/m/d/h いずれかを含めば日付/時刻書式
	lc := strings.ToLower(code)
	// 単純な検出 (ダブルクォート内の文字も含むが、誤判定は許容)
	for _, ch := range []string{"y", "d", "h"} {
		if strings.Contains(lc, ch) {
			return true
		}
	}
	// "m" は #,##0 などの中には現れないので含めば日付と判定 (ただし誤判定可能性あり)
	if strings.Contains(lc, "mm") {
		return true
	}
	return false
}
