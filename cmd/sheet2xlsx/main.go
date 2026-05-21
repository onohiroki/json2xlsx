package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yourname/sheet2xlsx/internal/sheet2xlsx"
)

func main() {
	var (
		input  string
		output string
		sheet  string
	)
	flag.StringVar(&input, "i", "", "input JSON file (default: stdin)")
	flag.StringVar(&output, "o", "", "output XLSX file (required)")
	flag.StringVar(&sheet, "sheet", "", "default sheet name when not specified in JSON")
	flag.Parse()

	if output == "" {
		fmt.Fprintln(os.Stderr, "error: -o is required")
		flag.Usage()
		os.Exit(2)
	}

	var r io.Reader = os.Stdin
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open input: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	out, err := os.Create(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if err := sheet2xlsx.Convert(r, out, sheet); err != nil {
		fmt.Fprintf(os.Stderr, "convert: %v\n", err)
		os.Exit(1)
	}
}
