package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yourname/sheet2xlsx/internal/sheet2xlsx"
)

func usage() {
	fmt.Fprintln(os.Stderr, `sheet2xlsx - XLSX <-> JSON 相互変換 CLI

Usage:
  sheet2xlsx to-json [-i input.xlsx] [-o output.json]
  sheet2xlsx to-xlsx [-i input.json] [-o output.xlsx] [--sheet name]
  sheet2xlsx to-md   [-i input.(json|xlsx)] [-o output.md] [--mode f|v|both] [--row-index]
  sheet2xlsx to-csv  [-i input.json] [-o output.csv]   # csvtk csv2json の逆変換
  sheet2xlsx        [-i input.json] [-o output.xlsx] [--sheet name]   # to-xlsx として動作

オプション:
  -i           入力ファイル (省略時 stdin)
  -o           出力ファイル (省略時 stdout)
  --sheet      to-xlsx でシート名未指定時のデフォルト
  --mode       to-md のセル表示モード (f=数式優先, v=値優先, both=併記). デフォルト f
  --row-index  to-md で行番号列を先頭に出力する

ロングオプションは --name 形式、短いオプションは -i / -o 形式で指定します
(-name / --i のような表記も受け付けますが、ドキュメントでは上記表記に統一しています)。
to-md は入力の magic byte (PK\x03\x04) で XLSX か JSON を自動判定する。
to-csv は csvtk csv2json の出力する JSON を CSV に戻す。`) 
}

func main() {
	args := os.Args[1:]
	sub := "to-xlsx"
	if len(args) > 0 {
		switch args[0] {
		case "to-json", "to-xlsx", "to-md", "to-csv":
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
	fs.StringVar(&input, "i", "", "input XLSX file (default: stdin)")
	fs.StringVar(&output, "o", "", "output JSON file (default: stdout)")
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

	if err := sheet2xlsx.ToJSON(r, w); err != nil {
		fmt.Fprintf(os.Stderr, "to-json: %v\n", err)
		os.Exit(1)
	}
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

	if err := sheet2xlsx.Convert(r, w, sheet); err != nil {
		fmt.Fprintf(os.Stderr, "to-xlsx: %v\n", err)
		os.Exit(1)
	}
}

func runToMD(args []string) {
	fs := flag.NewFlagSet("to-md", flag.ExitOnError)
	fs.Usage = usage
	var input, output, mode string
	var rowIndex bool
	fs.StringVar(&input, "i", "", "input file: JSON Workbook or XLSX (default: stdin)")
	fs.StringVar(&output, "o", "", "output Markdown file (default: stdout)")
	fs.StringVar(&mode, "mode", "f", "cell display mode: f|v|both")
	fs.BoolVar(&rowIndex, "row-index", false, "prepend row number column")
	_ = fs.Parse(args)

	switch sheet2xlsx.MarkdownMode(mode) {
	case sheet2xlsx.MarkdownModeFormula, sheet2xlsx.MarkdownModeValue, sheet2xlsx.MarkdownModeBoth:
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

	opts := sheet2xlsx.MarkdownOptions{
		Mode:     sheet2xlsx.MarkdownMode(mode),
		RowIndex: rowIndex,
	}
	if err := sheet2xlsx.ToMarkdown(r, w, opts); err != nil {
		fmt.Fprintf(os.Stderr, "to-md: %v\n", err)
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

	if err := sheet2xlsx.ToCSV(r, w); err != nil {
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
