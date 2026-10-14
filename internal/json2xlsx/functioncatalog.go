package json2xlsx

import (
	"sort"
	"strings"
)

// 数式エンジンが対応する関数のメタデータ．
//
// knownFuncs (formula.go) がトークナイザの真実源であり，functionCatalog は
// それを説明するための情報 (カテゴリ) を持つ．両者の不一致は init() で検出する．
//
// カテゴリ文字列は CLI の --category でも使うため小文字で固定する．
const (
	CategoryMath        = "math"
	CategoryLogical     = "logical"
	CategoryStatistical = "statistical"
	CategoryConditional = "conditional"
	CategoryLookup      = "lookup"
	CategoryText        = "text"
	CategoryDate        = "date"
	CategoryInfo        = "info"
	CategoryTrig        = "trig"
)

// FunctionInfo は 1 関数のメタデータ．
type FunctionInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

// functionCategories は display 順を固定するためのカテゴリ一覧．
var functionCategories = []string{
	CategoryMath,
	CategoryLogical,
	CategoryStatistical,
	CategoryConditional,
	CategoryLookup,
	CategoryText,
	CategoryDate,
	CategoryInfo,
	CategoryTrig,
}

// functionCatalog は対応関数とカテゴリの一覧．knownFuncs と 1:1 で対応する．
var functionCatalog = []FunctionInfo{
	// math
	{Name: "SUM", Category: CategoryMath},
	{Name: "AVERAGE", Category: CategoryMath},
	{Name: "COUNT", Category: CategoryMath},
	{Name: "COUNTA", Category: CategoryMath},
	{Name: "MIN", Category: CategoryMath},
	{Name: "MAX", Category: CategoryMath},
	{Name: "ABS", Category: CategoryMath},
	{Name: "ROUND", Category: CategoryMath},
	{Name: "ROUNDUP", Category: CategoryMath},
	{Name: "ROUNDDOWN", Category: CategoryMath},
	{Name: "PRODUCT", Category: CategoryMath},
	{Name: "SUMPRODUCT", Category: CategoryMath},
	{Name: "POWER", Category: CategoryMath},
	{Name: "SQRT", Category: CategoryMath},
	{Name: "INT", Category: CategoryMath},
	{Name: "MOD", Category: CategoryMath},
	{Name: "FLOOR", Category: CategoryMath},
	{Name: "CEILING", Category: CategoryMath},
	{Name: "TRUNC", Category: CategoryMath},
	{Name: "SIGN", Category: CategoryMath},
	{Name: "PI", Category: CategoryMath},
	{Name: "RAND", Category: CategoryMath},
	{Name: "SUMSQ", Category: CategoryMath},
	{Name: "EVEN", Category: CategoryMath},
	{Name: "ODD", Category: CategoryMath},
	{Name: "MROUND", Category: CategoryMath},
	{Name: "DELTA", Category: CategoryMath},
	{Name: "GESTEP", Category: CategoryMath},
	{Name: "FACT", Category: CategoryMath},

	// logical
	{Name: "IF", Category: CategoryLogical},
	{Name: "AND", Category: CategoryLogical},
	{Name: "OR", Category: CategoryLogical},
	{Name: "NOT", Category: CategoryLogical},
	{Name: "IFERROR", Category: CategoryLogical},
	{Name: "IFS", Category: CategoryLogical},
	{Name: "SWITCH", Category: CategoryLogical},

	// statistical
	{Name: "MEDIAN", Category: CategoryStatistical},
	{Name: "QUARTILE", Category: CategoryStatistical},
	{Name: "QUARTILE.INC", Category: CategoryStatistical},
	{Name: "PERCENTILE", Category: CategoryStatistical},
	{Name: "PERCENTILE.INC", Category: CategoryStatistical},
	{Name: "STDEV", Category: CategoryStatistical},
	{Name: "STDEV.S", Category: CategoryStatistical},
	{Name: "STDEV.P", Category: CategoryStatistical},
	{Name: "VAR", Category: CategoryStatistical},
	{Name: "VAR.S", Category: CategoryStatistical},
	{Name: "VAR.P", Category: CategoryStatistical},
	{Name: "GEOMEAN", Category: CategoryStatistical},
	{Name: "HARMEAN", Category: CategoryStatistical},
	{Name: "TRIMMEAN", Category: CategoryStatistical},
	{Name: "RANK", Category: CategoryStatistical},
	{Name: "RANK.EQ", Category: CategoryStatistical},
	{Name: "LARGE", Category: CategoryStatistical},
	{Name: "SMALL", Category: CategoryStatistical},
	{Name: "MODE", Category: CategoryStatistical},
	{Name: "MODE.SNGL", Category: CategoryStatistical},
	{Name: "SUBTOTAL", Category: CategoryStatistical},

	// conditional
	{Name: "SUMIF", Category: CategoryConditional},
	{Name: "COUNTIF", Category: CategoryConditional},
	{Name: "SUMIFS", Category: CategoryConditional},
	{Name: "COUNTIFS", Category: CategoryConditional},
	{Name: "AVERAGEIF", Category: CategoryConditional},
	{Name: "AVERAGEIFS", Category: CategoryConditional},
	{Name: "MINIFS", Category: CategoryConditional},
	{Name: "MAXIFS", Category: CategoryConditional},

	// lookup
	{Name: "VLOOKUP", Category: CategoryLookup},
	{Name: "HLOOKUP", Category: CategoryLookup},
	{Name: "XLOOKUP", Category: CategoryLookup},
	{Name: "INDEX", Category: CategoryLookup},
	{Name: "MATCH", Category: CategoryLookup},
	{Name: "CHOOSE", Category: CategoryLookup},

	// text
	{Name: "CONCAT", Category: CategoryText},
	{Name: "CONCATENATE", Category: CategoryText},
	{Name: "LEFT", Category: CategoryText},
	{Name: "RIGHT", Category: CategoryText},
	{Name: "MID", Category: CategoryText},
	{Name: "LEN", Category: CategoryText},
	{Name: "UPPER", Category: CategoryText},
	{Name: "LOWER", Category: CategoryText},
	{Name: "TRIM", Category: CategoryText},
	{Name: "FIND", Category: CategoryText},
	{Name: "TEXT", Category: CategoryText},

	// date
	{Name: "TODAY", Category: CategoryDate},
	{Name: "NOW", Category: CategoryDate},
	{Name: "YEAR", Category: CategoryDate},
	{Name: "MONTH", Category: CategoryDate},
	{Name: "DAY", Category: CategoryDate},
	{Name: "DAYS", Category: CategoryDate},
	{Name: "DATE", Category: CategoryDate},
	{Name: "EDATE", Category: CategoryDate},
	{Name: "EOMONTH", Category: CategoryDate},
	{Name: "WEEKDAY", Category: CategoryDate},
	{Name: "WEEKNUM", Category: CategoryDate},
	{Name: "NETWORKDAYS", Category: CategoryDate},
	{Name: "WORKDAY", Category: CategoryDate},

	// info
	{Name: "ISNUMBER", Category: CategoryInfo},
	{Name: "ISBLANK", Category: CategoryInfo},
	{Name: "ISTEXT", Category: CategoryInfo},
	{Name: "ISNONTEXT", Category: CategoryInfo},
	{Name: "ISERROR", Category: CategoryInfo},
	{Name: "ISNA", Category: CategoryInfo},
	{Name: "ROW", Category: CategoryInfo},
	{Name: "COLUMN", Category: CategoryInfo},

	// trig
	{Name: "SIN", Category: CategoryTrig},
	{Name: "COS", Category: CategoryTrig},
	{Name: "TAN", Category: CategoryTrig},
	{Name: "ASIN", Category: CategoryTrig},
	{Name: "ACOS", Category: CategoryTrig},
	{Name: "ATAN", Category: CategoryTrig},
	{Name: "ATAN2", Category: CategoryTrig},
	{Name: "SINH", Category: CategoryTrig},
	{Name: "COSH", Category: CategoryTrig},
	{Name: "TANH", Category: CategoryTrig},
	{Name: "ASINH", Category: CategoryTrig},
	{Name: "ACOSH", Category: CategoryTrig},
	{Name: "ATANH", Category: CategoryTrig},
	{Name: "DEGREES", Category: CategoryTrig},
	{Name: "RADIANS", Category: CategoryTrig},
	{Name: "LN", Category: CategoryTrig},
	{Name: "LOG", Category: CategoryTrig},
	{Name: "LOG10", Category: CategoryTrig},
	{Name: "EXP", Category: CategoryTrig},
}

// catalogByName は functionCatalog を名前で引くための索引．
var catalogByName = map[string]string{}

func init() {
	for _, fi := range functionCatalog {
		if _, dup := catalogByName[fi.Name]; dup {
			panic("json2xlsx: duplicate function in catalog: " + fi.Name)
		}
		catalogByName[fi.Name] = fi.Category
		if !knownFuncs[fi.Name] {
			panic("json2xlsx: catalog function not in knownFuncs: " + fi.Name)
		}
	}
	for name := range knownFuncs {
		if _, ok := catalogByName[name]; !ok {
			panic("json2xlsx: knownFuncs function missing from catalog: " + name)
		}
	}
}

// KnownFunctions は対応する全関数を名前昇順で返す．
func KnownFunctions() []FunctionInfo {
	out := make([]FunctionInfo, len(functionCatalog))
	copy(out, functionCatalog)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// FunctionCategories は表示順を固定したカテゴリ一覧を返す．
func FunctionCategories() []string {
	out := make([]string, len(functionCategories))
	copy(out, functionCategories)
	return out
}

// FilterFunctions は名前の部分一致 (大文字小文字無視) とカテゴリの完全一致で
// 絞り込んだ関数一覧を名前昇順で返す．search が空なら名前で絞り込まない．
func FilterFunctions(search, category string) []FunctionInfo {
	search = strings.ToUpper(strings.TrimSpace(search))
	category = strings.ToLower(strings.TrimSpace(category))
	out := []FunctionInfo{}
	for _, fi := range KnownFunctions() {
		if search != "" && !strings.Contains(fi.Name, search) {
			continue
		}
		if category != "" && fi.Category != category {
			continue
		}
		out = append(out, fi)
	}
	return out
}
