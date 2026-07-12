package json2xlsx

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// formatValue formats a formulaValue according to an Excel-style format string.
func formatValue(v formulaValue, formatStr string) string {
	formatStr = strings.TrimSpace(formatStr)
	if formatStr == "" || strings.EqualFold(formatStr, "General") {
		return v.asString()
	}

	sections := splitFormatSections(formatStr)
	selected := selectSection(sections, v)

	if looksLikeDateFormat(selected) {
		return formatDate(v, selected)
	}
	if looksLikeNumberFormat(selected) {
		return formatNumber(v, selected)
	}
	if selected == "@" {
		return v.asString()
	}

	// Fallback: strip quotes and escapes
	return stripLiterals(selected)
}

// ---------------------------------------------------------------------------
// Section splitting
// ---------------------------------------------------------------------------

func splitFormatSections(fmtStr string) []string {
	var sections []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(fmtStr); i++ {
		ch := fmtStr[i]
		if ch == '"' {
			inQuote = !inQuote
		}
		if ch == ';' && !inQuote {
			sections = append(sections, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(ch)
	}
	sections = append(sections, cur.String())
	return sections
}

func selectSection(sections []string, v formulaValue) string {
	switch len(sections) {
	case 1:
		return sections[0]
	case 2:
		if v.kind == valueNumber && v.num < 0 {
			return sections[1]
		}
		return sections[0]
	case 3:
		if v.kind == valueNumber && v.num > 0 {
			return sections[0]
		} else if v.kind == valueNumber && v.num < 0 {
			return sections[1]
		}
		return sections[2]
	case 4:
		if v.kind == valueString {
			return sections[3]
		}
		if v.num > 0 {
			return sections[0]
		} else if v.num < 0 {
			return sections[1]
		}
		return sections[2]
	}
	return sections[0]
}

// ---------------------------------------------------------------------------
// Format type detection
// ---------------------------------------------------------------------------

func looksLikeDateFormat(fmtStr string) bool {
	s := stripQuotedRunes(fmtStr)
	hasDateToken := false
	for i := 0; i < len(s); {
		switch {
		case matchToken(s, i, "mmmm"): i += 4; hasDateToken = true
		case matchToken(s, i, "mmm"):  i += 3; hasDateToken = true
		case matchToken(s, i, "dddd"): i += 4; hasDateToken = true
		case matchToken(s, i, "ddd"):  i += 3; hasDateToken = true
		case matchToken(s, i, "yyyy"): i += 4; hasDateToken = true
		case matchToken(s, i, "yy"):   i += 2; hasDateToken = true
		case matchToken(s, i, "hh"):   i += 2; hasDateToken = true
		case matchToken(s, i, "h"):    i += 1; hasDateToken = true
		case matchToken(s, i, "ss"):   i += 2; hasDateToken = true
		case matchToken(s, i, "s"):    i += 1; hasDateToken = true
		case matchToken(s, i, "dd"):   i += 2; hasDateToken = true
		case matchToken(s, i, "d"):    i += 1; hasDateToken = true
		case matchToken(s, i, "mm"):   i += 2; hasDateToken = true
		case matchToken(s, i, "m"):    i += 1; hasDateToken = true
		default:
			r, size := utf8DecodeRune(s, i)
			if r == -1 {
				i++
			} else {
				i += size
			}
		}
	}
	return hasDateToken
}

func looksLikeNumberFormat(fmtStr string) bool {
	s := stripQuotedRunes(fmtStr)
	return strings.ContainsAny(s, "0#") || strings.Contains(s, "%")
}

func stripQuotedRunes(s string) string {
	var out strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		if ch == '\\' && i+1 < len(s) {
			i++
			continue
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func matchToken(s string, pos int, tok string) bool {
	if pos+len(tok) > len(s) {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if s[pos+i] != tok[i] {
			return false
		}
	}
	// check next char is not same letter (avoid partial match like "y" matching "yy")
	if pos+len(tok) < len(s) && s[pos+len(tok)] == tok[0] {
		return false
	}
	return true
}

func utf8DecodeRune(s string, pos int) (rune, int) {
	if pos >= len(s) {
		return -1, 0
	}
	r, size := rune(s[pos]), 1
	if r > unicode.MaxASCII {
		r, size = utf8.DecodeRuneInString(s[pos:])
	}
	return r, size
}

// ---------------------------------------------------------------------------
// Date formatting
// ---------------------------------------------------------------------------

func formatDate(v formulaValue, fmtStr string) string {
	var t time.Time
	if v.kind == valueNumber {
		t = excelSerialToDateTime(v.num)
	} else {
		// try parse as serial number
		if f, err := strconv.ParseFloat(v.str, 64); err == nil {
			t = excelSerialToDateTime(f)
		} else {
			return v.str
		}
	}
	return formatTimeWithPattern(t, fmtStr)
}

func formatTimeWithPattern(t time.Time, fmtStr string) string {
	// First pass: tokenize the format string, disambiguating m/mm
	type token struct {
		kind string // "yyyy","yy","mmmm","mmm","dddd","ddd","dd","d","hh","h","mm","m","ss","s","lit"
		lit  string
	}
	var toks []token
	i := 0
	for i < len(fmtStr) {
		ch := fmtStr[i]
		if ch == '"' {
			i++
			var lit strings.Builder
			for i < len(fmtStr) && fmtStr[i] != '"' {
				lit.WriteByte(fmtStr[i])
				i++
			}
			i++ // skip closing quote
			toks = append(toks, token{"lit", lit.String()})
			continue
		}
		if ch == '\\' && i+1 < len(fmtStr) {
			toks = append(toks, token{"lit", string(fmtStr[i+1])})
			i += 2
			continue
		}
		switch {
		case matchToken(fmtStr, i, "yyyy"):
			toks = append(toks, token{"yyyy", ""}); i += 4
		case matchToken(fmtStr, i, "yy"):
			toks = append(toks, token{"yy", ""}); i += 2
		case matchToken(fmtStr, i, "mmmm"):
			toks = append(toks, token{"mmmm", ""}); i += 4
		case matchToken(fmtStr, i, "mmm"):
			toks = append(toks, token{"mmm", ""}); i += 3
		case matchToken(fmtStr, i, "dddd"):
			toks = append(toks, token{"dddd", ""}); i += 4
		case matchToken(fmtStr, i, "ddd"):
			toks = append(toks, token{"ddd", ""}); i += 3
		case matchToken(fmtStr, i, "dd"):
			toks = append(toks, token{"dd", ""}); i += 2
		case matchToken(fmtStr, i, "d"):
			toks = append(toks, token{"d", ""}); i += 1
		case matchToken(fmtStr, i, "hh"):
			toks = append(toks, token{"hh", ""}); i += 2
		case matchToken(fmtStr, i, "h"):
			toks = append(toks, token{"h", ""}); i += 1
		case matchToken(fmtStr, i, "ss"):
			toks = append(toks, token{"ss", ""}); i += 2
		case matchToken(fmtStr, i, "s"):
			toks = append(toks, token{"s", ""}); i += 1
		case matchToken(fmtStr, i, "mm"):
			toks = append(toks, token{"_mm", ""}); i += 2 // defer decision
		case matchToken(fmtStr, i, "m"):
			toks = append(toks, token{"_m", ""}); i += 1
		default:
			toks = append(toks, token{"lit", string(ch)}); i++
		}
	}

	// Second pass: disambiguate _mm/_m based on context
	// m after h or : is minute; otherwise month
	resolveM := func(idx int) string {
		// look back for h or : within last few tokens
		for j := idx - 1; j >= 0 && j >= idx-3; j-- {
			if toks[j].kind == "lit" && toks[j].lit == ":" {
				return "mm"
			}
			if toks[j].kind == "hh" || toks[j].kind == "h" {
				return "mm"
			}
			if toks[j].kind != "lit" || toks[j].lit == " " {
				break
			}
		}
		return "mon"
	}
	for j := range toks {
		if toks[j].kind == "_mm" || toks[j].kind == "_m" {
			if resolveM(j) == "mm" {
				toks[j].kind = strings.TrimPrefix(toks[j].kind, "_")
			} else {
				if toks[j].kind == "_mm" {
					toks[j].kind = "mon2"
				} else {
					toks[j].kind = "mon1"
				}
			}
		}
	}

	var out strings.Builder
	for _, tok := range toks {
		switch tok.kind {
		case "yyyy":
			fmt.Fprintf(&out, "%04d", t.Year())
		case "yy":
			fmt.Fprintf(&out, "%02d", t.Year()%100)
		case "mmmm":
			out.WriteString(t.Month().String())
		case "mmm":
			out.WriteString(t.Month().String()[:3])
		case "dddd":
			out.WriteString(t.Weekday().String())
		case "ddd":
			out.WriteString(t.Weekday().String()[:3])
		case "dd":
			fmt.Fprintf(&out, "%02d", t.Day())
		case "d":
			fmt.Fprintf(&out, "%d", t.Day())
		case "hh":
			fmt.Fprintf(&out, "%02d", t.Hour())
		case "h":
			fmt.Fprintf(&out, "%d", t.Hour())
		case "mm":
			fmt.Fprintf(&out, "%02d", t.Minute())
		case "m":
			fmt.Fprintf(&out, "%d", t.Minute())
		case "mon2":
			fmt.Fprintf(&out, "%02d", t.Month())
		case "mon1":
			fmt.Fprintf(&out, "%d", t.Month())
		case "ss":
			fmt.Fprintf(&out, "%02d", t.Second())
		case "s":
			fmt.Fprintf(&out, "%d", t.Second())
		default:
			out.WriteString(tok.lit)
		}
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Number formatting
// ---------------------------------------------------------------------------

type numFormat struct {
	intDigits   int    // mandatory integer digits (0)
	intOptional int    // optional integer digits (#)
	thousandsSep bool
	decimal     bool
	decDigits   int    // mandatory decimal digits (0)
	decOptional int    // optional decimal digits (#)
	isPercent   bool
}

func parseNumberFormat(section string) numFormat {
	s := section
	var nf numFormat

	if strings.Contains(s, "%") {
		nf.isPercent = true
		s = strings.ReplaceAll(s, "%", "")
	}

	if strings.Contains(s, ",") {
		nf.thousandsSep = true
		s = strings.ReplaceAll(s, ",", "")
	}

	dotIdx := strings.Index(s, ".")
	if dotIdx >= 0 {
		nf.decimal = true
		intPart := s[:dotIdx]
		decPart := s[dotIdx+1:]
		for _, ch := range intPart {
			if ch == '0' {
				nf.intDigits++
			} else if ch == '#' {
				nf.intOptional++
			}
		}
		for _, ch := range decPart {
			if ch == '0' {
				nf.decDigits++
			} else if ch == '#' {
				nf.decOptional++
			}
		}
	} else {
		for _, ch := range s {
			if ch == '0' {
				nf.intDigits++
			} else if ch == '#' {
				nf.intOptional++
			}
		}
	}
	return nf
}

func formatNumber(v formulaValue, section string) string {
	if v.kind == valueString {
		return v.str
	}
	num := v.num

	// Extract prefix, pattern, and suffix from the section
	prefix, pattern, suffix := splitNumberSection(section)
	nf := parseNumberFormat(pattern)

	if nf.isPercent {
		num *= 100
	}

	absNum := math.Abs(num)
	intPart := int64(absNum)

	intStr := strconv.FormatInt(intPart, 10)
	minDigits := nf.intDigits
	if nf.intDigits == 0 && nf.intOptional == 0 {
		minDigits = 1
	}
	if len(intStr) < minDigits {
		intStr = strings.Repeat("0", minDigits-len(intStr)) + intStr
	}

	if nf.thousandsSep && len(intStr) > 3 {
		var b strings.Builder
		for j, c := range intStr {
			if j > 0 && (len(intStr)-j)%3 == 0 {
				b.WriteByte(',')
			}
			b.WriteRune(c)
		}
		intStr = b.String()
	}

	var decStr string
	totalDec := nf.decDigits + nf.decOptional
	if totalDec > 0 {
		format := "%." + strconv.Itoa(totalDec) + "f"
		fracPart := fmt.Sprintf(format, absNum-float64(intPart))
		decStr = fracPart[2:] // skip "0."
		// trim trailing zeros from optional part
		if nf.decOptional > 0 {
			keep := nf.decDigits
			decStr = trimTrailingZeros(decStr, keep)
		}
	}

	numberPart := intStr
	if totalDec > 0 {
		if decStr == "" {
			decStr = strings.Repeat("0", nf.decDigits)
		}
		numberPart += "." + decStr
	}
	if nf.isPercent {
		numberPart += "%"
	}

	// For negative values whose section does not already handle the sign,
	// prepend "-".
	if num < 0 && !strings.ContainsAny(prefix, "-(") && !strings.ContainsAny(suffix, ")") {
		prefix = "-" + prefix
	}

	return prefix + numberPart + suffix
}

func splitNumberSection(section string) (prefix, pattern, suffix string) {
	start, end := -1, -1
	inQuote := false
	for i := 0; i < len(section); i++ {
		ch := section[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == '\\' && i+1 < len(section) {
			i++
			continue
		}
		if inQuote {
			continue
		}
		if ch == '0' || ch == '#' || ch == '.' || ch == ',' || ch == '%' {
			if start == -1 {
				start = i
			}
			end = i + 1
		}
	}
	if start == -1 {
		return section, "", ""
	}
	// Include quoted/escaped parts within the pattern region
	return section[:start], section[start:end], section[end:]
}

func trimTrailingZeros(s string, keep int) string {
	if keep >= len(s) {
		return s
	}
	i := len(s)
	for i > keep && s[i-1] == '0' {
		i--
	}
	return s[:i]
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stripLiterals(s string) string {
	var out strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == '\\' && i+1 < len(s) {
			out.WriteByte(s[i+1])
			i++
			continue
		}
		if inQuote {
			out.WriteByte(ch)
		}
	}
	return out.String()
}
