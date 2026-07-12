package main

import (
    "flag"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "runtime/debug"
    "strings"

    "github.com/onohiroki/json2xlsx/internal/json2xlsx"
)

// inputBaseDir returns the directory component of an input file path.
// It is used to resolve relative image paths when converting JSON → XLSX.
// If the caller passes an empty string (e.g., stdin), the returned
// value is also an empty string so that callers can treat it as “no base”.
func inputBaseDir(input string) string {
    if input == "" {
        return ""
    }
    return filepath.Dir(input)
}

var version = "dev"

func getVersion() string {
    if version != "dev" {
        return version
    }
    info, ok := debug.ReadBuildInfo()
    if !ok {
        return "dev"
    }
    if info.Main.Version != "" && info.Main.Version != "(devel)" {
        return info.Main.Version
    }
    return "dev"
}

// Note: Japanese text in this file uses "，" and "．" as punctuation
// (not "、" and "。"). Keep this consistent when editing help messages.

func usage() {
	if isJapanese() {
		usageJa()
	} else {
		usageEn()
	}
}

func isJapanese() bool {
	lang := os.Getenv("LANG")
	return lang == "ja" || strings.HasPrefix(lang, "ja_") || strings.HasPrefix(lang, "ja.") || strings.HasPrefix(lang, "ja-")
}

func usageJa() {
	fmt.Fprintln(os.Stderr, `json2xlsx - XLSX <-> JSON 相互変換 CLI

Usage:
  json2xlsx to-json [-i input.xlsx] [-o output.json] [--date-display|--date-rfc3339|--date-serial] [--image-mode base64|file]
  json2xlsx to-xlsx [-i input.json] [-o output.xlsx] [--data-json] [--compute]
  json2xlsx to-md   [-i input.(json|xlsx)] [-o output.md] [--mode f|v|both] [--first-row-header] [--data-json] [--compute]
  json2xlsx to-html [-i input.(json|xlsx)] [-o output.html] [--mode f|v|both] [--grid] [--data-json] [--compute]
  json2xlsx to-csv  [-i input.(json|xlsx)] [-o output.csv] [--sheet name] [--sheet-index n] [--data-json] [--compute]
  json2xlsx         [-i input.json] [-o output.xlsx]

オプション:
  -i                   入力ファイル (省略時 stdin)．JSON (SheetJS 形式 / データ JSON) または XLSX
  -o                   出力ファイル (省略時 stdout)
  --data-json          入力をデータ JSON (二次元配列 / オブジェクト配列 / Map-of-Arrays) として扱う
  --compute            数式 (t=f) を評価し v を補完する
  --sheet              to-csv で入力 XLSX または JSON から抽出するシート名
  --sheet-index        to-csv で入力 XLSX または JSON から抽出するシート番号 (1から開始)
  --date-serial        to-json で日時セルを Excel シリアル値で出力する (既定)
  --date-display       to-json で日時セルを表示文字列で出力する (非推奨，--date-rfc3339 を推奨)
  --date-rfc3339       to-json で日時セルを RFC3339 (UTC) に再解釈して出力する
  --image-mode         to-json の画像出力モード: base64 (JSON に埋め込み) または file (外部ファイル)
  --mode               セル表示モード (f=数式優先, v=値優先, both=併記)．to-md デフォルト f, to-html デフォルト v
  --first-row-header   to-md で最初の行をテーブルヘッダとして扱う (A/B/C 列名 + 行番号を抑制)
  --grid               to-html で枠線未指定セルにグレーの細枠線を表示する

ロングオプションは --name 形式，短いオプションは -i / -o 形式で指定します
(-name / --i のような表記も受け付けますが，ドキュメントでは上記表記に統一しています)．
to-md / to-html / to-xlsx は入力の magic byte (PK\x03\x04) で XLSX か JSON を自動判定する．
JSON の場合は SheetJS Cell Object 形式を基本とするが，--data-json を指定すると二次元配列・オブジェクト配列・Map-of-Arrays の 3 形式を受け付ける．
to-csv は JSON (SheetJS / 二次元配列 / csvtk / xlsx-cli) または XLSX を CSV に戻す．
サブコマンドを省略した場合は to-xlsx として動作する．`)
}

func usageEn() {
	fmt.Fprintln(os.Stderr, `json2xlsx - XLSX <-> JSON CLI conversion tool

Usage:
  json2xlsx to-json [-i input.xlsx] [-o output.json] [--date-display|--date-rfc3339|--date-serial] [--image-mode base64|file]
  json2xlsx to-xlsx [-i input.json] [-o output.xlsx] [--data-json] [--compute]
  json2xlsx to-md   [-i input.(json|xlsx)] [-o output.md] [--mode f|v|both] [--first-row-header] [--data-json] [--compute]
  json2xlsx to-html [-i input.(json|xlsx)] [-o output.html] [--mode f|v|both] [--grid] [--data-json] [--compute]
  json2xlsx to-csv  [-i input.(json|xlsx)] [-o output.csv] [--sheet name] [--sheet-index n] [--data-json] [--compute]
  json2xlsx         [-i input.json] [-o output.xlsx]

Options:
  -i                   Input file (default: stdin). JSON (SheetJS / data JSON) or XLSX
  -o                   Output file (default: stdout)
  --data-json          Treat input as data JSON (2D array / array of objects / map-of-arrays)
  --compute            Evaluate formulas (t=f) and fill cached values
  --sheet              Sheet name for to-csv
  --sheet-index        Sheet index for to-csv (1-based)
  --date-serial        Emit date cells as Excel serial values (default)
  --date-display       Emit date cells as display strings
  --date-rfc3339       Reinterpret date/time serial as RFC3339 (UTC)
  --image-mode         Image output mode: base64 (embed in JSON) or file (write to disk)
  --mode               Cell display mode (f=formula, v=value, both=both). Default to-md=f, to-html=v
  --first-row-header   Treat first row as table header (suppress A/B/C and row numbers)
  --grid               Show light gray gridlines for empty cells in to-html

Long options use --name, short options use -i / -o.
When subcommand is omitted, defaults to to-xlsx.
to-md / to-html / to-xlsx auto-detects XLSX vs JSON using magic bytes (PK\x03\x04).
Without --data-json, JSON input must be SheetJS Cell Object format.
With --data-json, accepts 2D array / array of objects / map-of-arrays.
to-csv converts JSON or XLSX back to CSV.`)
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
			fmt.Fprintln(os.Stderr, "json2xlsx", getVersion())
			usage()
			return
		case "-version", "--version", "version":
			fmt.Fprintln(os.Stderr, "json2xlsx", getVersion())
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
	var imageMode string
	fs.StringVar(&input, "i", "", "input XLSX file (default: stdin)")
	fs.StringVar(&output, "o", "", "output JSON file (default: stdout)")
	fs.BoolVar(&dateDisplay, "date-display", false, "emit date cells as display strings")
	fs.BoolVar(&dateRFC3339, "date-rfc3339", false, "reinterpret date/time serial values as RFC3339 (UTC)")
	fs.BoolVar(&dateSerial, "date-serial", false, "emit date cells as Excel serial values")
	fs.StringVar(&imageMode, "image-mode", "base64", "image output mode: base64 (embed in JSON) or file (write to disk)")
	_ = fs.Parse(args)

	dateMode, err := resolveDateMode(dateDisplay, dateRFC3339, dateSerial)
	if err != nil {
		fmt.Fprintf(os.Stderr, "to-json: %v\n", err)
		os.Exit(2)
	}

	imageModeVal := json2xlsx.ImageMode(imageMode)
	if imageModeVal != json2xlsx.ImageModeBase64 && imageModeVal != json2xlsx.ImageModeFile {
		fmt.Fprintf(os.Stderr, "to-json: --image-mode must be 'base64' or 'file'\n")
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

	opts := json2xlsx.ToJSONOptions{
		DateMode:    dateMode,
		WrapWithBook: true,
		ImageMode:   imageModeVal,
		BaseDir:     outputBaseDir(output),
	}
	if err := json2xlsx.ToJSONWithOptions(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-json: %v\n", err)
		os.Exit(1)
	}
}

func outputBaseDir(output string) string {
	if output == "" {
		cwd, _ := os.Getwd()
		return cwd
	}
	return filepath.Dir(output)
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
	var input, output string
	var dataJSON, compute bool
	fs.StringVar(&input, "i", "", "input JSON file (default: stdin)")
	fs.StringVar(&output, "o", "", "output XLSX file (default: stdout)")
	fs.BoolVar(&dataJSON, "data-json", false, "accept 2D array, array of objects, or map-of-arrays JSON")
	fs.BoolVar(&compute, "compute", false, "evaluate simple formulas and fill cached values")
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

	baseDir := inputBaseDir(input)
	if err := json2xlsx.Convert(r, w, json2xlsx.ConvertOptions{DataJSON: dataJSON, EvalFormulas: compute, BaseDir: baseDir}); err != nil {
		fmt.Fprintf(os.Stderr, "to-xlsx: %v\n", err)
		os.Exit(1)
	}
}

func runToMD(args []string) {
	fs := flag.NewFlagSet("to-md", flag.ExitOnError)
	fs.Usage = usage
	var input, output, mode string
	var firstRowHeader, dataJSON, compute bool
	fs.StringVar(&input, "i", "", "input file: JSON Workbook or XLSX (default: stdin)")
	fs.StringVar(&output, "o", "", "output Markdown file (default: stdout)")
	fs.StringVar(&mode, "mode", "f", "cell display mode: f|v|both")
	fs.BoolVar(&firstRowHeader, "first-row-header", false, "use first row as table header (suppress A/B/C column headers and row numbers)")
	fs.BoolVar(&dataJSON, "data-json", false, "accept 2D array, array of objects, or map-of-arrays JSON")
	fs.BoolVar(&compute, "compute", false, "evaluate formulas (t=f) and fill cached values")
	_ = fs.Parse(args)

	if err := json2xlsx.ValidateMode(json2xlsx.MarkdownMode(mode)); err != nil {
		fmt.Fprintf(os.Stderr, "to-md: %v\n", err)
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

	explicitMode := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "mode" {
			explicitMode = true
		}
	})

	opts := json2xlsx.MarkdownOptions{
		Mode:           json2xlsx.MarkdownMode(mode),
		FirstRowHeader: firstRowHeader,
		RowIndex:       !firstRowHeader,
		ExplicitMode:   explicitMode,
		DataJSON:       dataJSON,
		EvalFormulas:   compute,
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
	var grid, dataJSON, compute bool
	fs.StringVar(&input, "i", "", "input file: JSON Workbook or XLSX (default: stdin)")
	fs.StringVar(&output, "o", "", "output HTML file (default: stdout)")
	fs.StringVar(&mode, "mode", "v", "cell display mode: f|v|both (default: v)")
	fs.BoolVar(&grid, "grid", false, "collapse cellspacing and show thin gray borders on all cells")
	fs.BoolVar(&dataJSON, "data-json", false, "accept 2D array, array of objects, or map-of-arrays JSON")
	fs.BoolVar(&compute, "compute", false, "evaluate formulas (t=f) and fill cached values")
	_ = fs.Parse(args)

	if err := json2xlsx.ValidateMode(json2xlsx.MarkdownMode(mode)); err != nil {
		fmt.Fprintf(os.Stderr, "to-html: %v\n", err)
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

	explicitMode := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "mode" {
			explicitMode = true
		}
	})

	opts := json2xlsx.HTMLOptions{
		Mode:         json2xlsx.MarkdownMode(mode),
		GridLines:    grid,
		ExplicitMode: explicitMode,
		DataJSON:     dataJSON,
		EvalFormulas: compute,
	}
	if err := json2xlsx.ToHTML(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-html: %v\n", err)
		os.Exit(1)
	}
}

func runToCSV(args []string) {
	fs := flag.NewFlagSet("to-csv", flag.ExitOnError)
	fs.Usage = usage
	var input, output, sheet string
	var sheetIndex int
	var dataJSON, compute bool
	fs.StringVar(&input, "i", "", "input CSV JSON or XLSX file (default: stdin)")
	fs.StringVar(&output, "o", "", "output CSV file (default: stdout)")
	fs.StringVar(&sheet, "sheet", "", "sheet name for XLSX or Workbook JSON")
	fs.IntVar(&sheetIndex, "sheet-index", 0, "sheet index (1-based) for XLSX or Workbook JSON")
	fs.BoolVar(&dataJSON, "data-json", false, "accept 2D array, array of objects, or map-of-arrays JSON")
	fs.BoolVar(&compute, "compute", false, "evaluate formulas (t=f) and fill cached values")
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

	if err := json2xlsx.ToCSV(r, w, sheet, sheetIndex, dataJSON, compute); err != nil {
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
