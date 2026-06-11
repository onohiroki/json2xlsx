package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"json2xlsx/internal/json2xlsx"
)

func usage() {
	fmt.Fprintln(os.Stderr, `json2xlsx - XLSX <-> JSON 相互変換 CLI

Usage:
  json2xlsx to-json [-i input.xlsx] [-o output.json] [--date-display|--date-rfc3339|--date-serial]
  json2xlsx to-xlsx [-i input.json] [-o output.xlsx] [--sheet name]
  json2xlsx to-md   [-i input.(json|xlsx)] [-o output.md] [--mode f|v|both] [--first-row-header]
  json2xlsx to-html [-i input.(json|xlsx)] [-o output.html] [--mode f|v|both] [--grid]  # JSON / XLSX → HTML <table>
  json2xlsx to-csv  [-i input.json] [-o output.csv]   # csvtk / xlsx-cli の JSON を CSV に変換
  json2xlsx        [-i input.json] [-o output.xlsx] [--sheet name]   # to-xlsx として動作

オプション:
  -i           入力ファイル (省略時 stdin)
  -o           出力ファイル (省略時 stdout)
  --sheet      to-xlsx でシート名未指定時のデフォルト
  --date-serial    to-json で日時セルを Excel シリアル値で出力する (既定)
  --date-display   to-json で日時セルを表示文字列で出力する
  --date-rfc3339   to-json で日時セルを RFC3339 (UTC) に再解釈して出力する
  --mode             セル表示モード (f=数式優先, v=値優先, both=併記). to-md デフォルト f, to-html デフォルト v
  --first-row-header to-md で最初の行をテーブルヘッダとして扱う (A/B/C 列名 + 行番号を抑制)
  --grid             to-html で枠線未指定セルにグレーの細枠線を表示する

ロングオプションは --name 形式、短いオプションは -i / -o 形式で指定します
(-name / --i のような表記も受け付けますが、ドキュメントでは上記表記に統一しています)。
to-md / to-html は入力の magic byte (PK\x03\x04) で XLSX か JSON を自動判定する。
to-csv は csvtk csv2json または xlsx-cli -j の JSON を CSV に戻す。`)
}

func main() {
	args := os.Args[1:]
	sub := "to-xlsx"
	if len(args) > 0 {
		switch args[0] {
		case "to-json", "to-xlsx", "to-md", "to-html", "to-csv":
			sub = args[0]
			args = args[1:]
		case "-h", "--help", "help":
			usage()
			return
		}
	}

	switch sub {
	case "to-json":
		runToJSON(args)
	case "to-xlsx":
		runToXLSX(args)
	case "to-md":
		runToMD(args)
	case "to-html":
		runToHTML(args)
	case "to-csv":
		runToCSV(args)
	default:
		usage()
		os.Exit(2)
	}
}

func runToJSON(args []string) {
	fs := flag.NewFlagSet("to-json", flag.ExitOnError)
	fs.Usage = usage
	var input, output string
	var dateDisplay, dateRFC3339, dateSerial bool
	fs.StringVar(&input, "i", "", "input XLSX file (default: stdin)")
	fs.StringVar(&output, "o", "", "output JSON file (default: stdout)")
	fs.BoolVar(&dateDisplay, "date-display", false, "emit date cells as display strings")
	fs.BoolVar(&dateRFC3339, "date-rfc3339", false, "reinterpret date/time serial values as RFC3339 (UTC)")
	fs.BoolVar(&dateSerial, "date-serial", false, "emit date cells as Excel serial values")
	_ = fs.Parse(args)

	dateMode, err := resolveDateMode(dateDisplay, dateRFC3339, dateSerial)
	if err != nil {
		fmt.Fprintf(os.Stderr, "to-json: %v\n", err)
		os.Exit(2)
	}

	r, closeR, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer closeR()

	w, closeW, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer closeW()

	opts := json2xlsx.ToJSONOptions{DateMode: dateMode, WrapWithBook: true}
	if err := json2xlsx.ToJSONWithOptions(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-json: %v\n", err)
		os.Exit(1)
	}
}

func resolveDateMode(dateDisplay, dateRFC3339, dateSerial bool) (json2xlsx.DateMode, error) {
	count := 0
	mode := json2xlsx.DateModeSerial
	if dateDisplay {
		count++
		mode = json2xlsx.DateModeDisplay
	}
	if dateRFC3339 {
		count++
		mode = json2xlsx.DateModeRFC3339
	}
	if dateSerial {
		count++
		mode = json2xlsx.DateModeSerial
	}
	if count > 1 {
		return "", fmt.Errorf("choose only one of --date-display, --date-rfc3339, --date-serial")
	}
	return mode, nil
}

func runToXLSX(args []string) {
	fs := flag.NewFlagSet("to-xlsx", flag.ExitOnError)
	fs.Usage = usage
	var input, output, sheet string
	fs.StringVar(&input, "i", "", "input JSON file (default: stdin)")
	fs.StringVar(&output, "o", "", "output XLSX file (default: stdout)")
	fs.StringVar(&sheet, "sheet", "", "default sheet name")
	_ = fs.Parse(args)

	r, closeR, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer closeR()

	w, closeW, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer closeW()

	if err := json2xlsx.Convert(r, w, sheet); err != nil {
		fmt.Fprintf(os.Stderr, "to-xlsx: %v\n", err)
		os.Exit(1)
	}
}

func runToMD(args []string) {
	fs := flag.NewFlagSet("to-md", flag.ExitOnError)
	fs.Usage = usage
	var input, output, mode string
	var firstRowHeader bool
	fs.StringVar(&input, "i", "", "input file: JSON Workbook or XLSX (default: stdin)")
	fs.StringVar(&output, "o", "", "output Markdown file (default: stdout)")
	fs.StringVar(&mode, "mode", "f", "cell display mode: f|v|both")
	fs.BoolVar(&firstRowHeader, "first-row-header", false, "use first row as table header (suppress A/B/C column headers and row numbers)")
	_ = fs.Parse(args)

	switch json2xlsx.MarkdownMode(mode) {
	case json2xlsx.MarkdownModeFormula, json2xlsx.MarkdownModeValue, json2xlsx.MarkdownModeBoth:
	default:
		fmt.Fprintf(os.Stderr, "to-md: invalid -mode %q (expected f|v|both)\n", mode)
		os.Exit(2)
	}

	r, closeR, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer closeR()

	w, closeW, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer closeW()

	opts := json2xlsx.MarkdownOptions{
		Mode:            json2xlsx.MarkdownMode(mode),
		FirstRowHeader:  firstRowHeader,
		RowIndex:        !firstRowHeader,
	}
	if err := json2xlsx.ToMarkdown(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-md: %v\n", err)
		os.Exit(1)
	}
}

func runToHTML(args []string) {
	fs := flag.NewFlagSet("to-html", flag.ExitOnError)
	fs.Usage = usage
	var input, output, mode string
	var grid bool
	fs.StringVar(&input, "i", "", "input file: JSON Workbook or XLSX (default: stdin)")
	fs.StringVar(&output, "o", "", "output HTML file (default: stdout)")
	fs.StringVar(&mode, "mode", "v", "cell display mode: f|v|both (default: v)")
	fs.BoolVar(&grid, "grid", false, "collapse cellspacing and show thin gray borders on all cells")
	_ = fs.Parse(args)

	switch json2xlsx.MarkdownMode(mode) {
	case json2xlsx.MarkdownModeFormula, json2xlsx.MarkdownModeValue, json2xlsx.MarkdownModeBoth:
	default:
		fmt.Fprintf(os.Stderr, "to-html: invalid --mode %q (expected f|v|both)\n", mode)
		os.Exit(2)
	}

	r, closeR, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer closeR()

	w, closeW, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer closeW()

	opts := json2xlsx.HTMLOptions{Mode: json2xlsx.MarkdownMode(mode), GridLines: grid}
	if err := json2xlsx.ToHTML(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-html: %v\n", err)
		os.Exit(1)
	}
}

func runToCSV(args []string) {
	fs := flag.NewFlagSet("to-csv", flag.ExitOnError)
	fs.Usage = usage
	var input, output string
	fs.StringVar(&input, "i", "", "input CSV JSON file (default: stdin)")
	fs.StringVar(&output, "o", "", "output CSV file (default: stdout)")
	_ = fs.Parse(args)

	r, closeR, err := openInput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open input: %v\n", err)
		os.Exit(1)
	}
	defer closeR()

	w, closeW, err := openOutput(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer closeW()

	if err := json2xlsx.ToCSV(r, w); err != nil {
		fmt.Fprintf(os.Stderr, "to-csv: %v\n", err)
		os.Exit(1)
	}
}

func openInput(path string) (io.Reader, func(), error) {
	if path == "" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func openOutput(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}
