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
  sheet2xlsx to-xlsx [-i input.json] [-o output.xlsx] [-sheet name]
  sheet2xlsx        [-i input.xlsx] [-o output.json]   # to-json として動作

オプション:
  -i  入力ファイル (省略時 stdin)
  -o  出力ファイル (省略時 stdout)
  -sheet  to-xlsx でシート名未指定時のデフォルト`)
}

func main() {
	args := os.Args[1:]
	sub := "to-json"
	if len(args) > 0 {
		switch args[0] {
		case "to-json", "to-xlsx":
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
