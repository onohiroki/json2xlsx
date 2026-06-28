package json2xlsx

// warnMissingFormulaValue はセルに数式のみあり値がない場合の警告メッセージを返す．
func warnMissingFormulaValue(mode MarkdownMode) string {
	if mode == MarkdownModeBoth {
		return "Warning: Missing values for some cells; showing only formulas."
	}
	return "Warning: Missing values for some cells; showing formulas instead."
}

// warnFormulaOnlyCSV は CSV 出力時に値のない数式セルがあった場合の警告メッセージを返す．
func warnFormulaOnlyCSV() string {
	return "Warning: Some cells have formulas but no values; treating them as empty."
}
