# json2xlsx

[ English | [日本語](README.ja.md) ]

`json2xlsx` is a Go-based CLI tool that generates Excel `.xlsx` files from SheetJS-style JSON. The Go side is responsible only for the JSON → XLSX conversion and does not include any AI calls.[1][2]

It is intended to be used together with the `sheetjs-json-writer` skill to make AI reliably output SheetJS-style JSON.[3][4]

## Purpose

The goal of this project is to separate the following two stages:

1. The AI emits tables and aggregate results as SheetJS-style JSON.
2. `json2xlsx` reads that JSON and converts it to `.xlsx`.[5][6]

This separation keeps the Go tool lightweight, easy to test, and suitable for OSS distribution.[7][8]

### Why use `json2xlsx` — eliminate nondeterminism from AI-generated code

You could ask an AI coding tool to write code that generates XLSX on the spot, but that approach has several issues:

- **Nondeterminism** — the same prompt may produce different code and different output each time; reproducible quality is not guaranteed.
- **Execution risk** — generated code must be executed, which can require library installation, cause version conflicts, or introduce security concerns.
- **Hard to debug** — when output is wrong, a human must trace which part of generated code is incorrect.

`json2xlsx` solves this by making the AI output only JSON:

- **JSON is deterministic** — the same JSON always yields the same XLSX. AI variance is confined to the JSON generation step, while the conversion pipeline remains stable.
- **No code execution required** — the AI only needs to produce JSON; running arbitrary generated code is unnecessary (JSON validation is lightweight).
- **Human-friendly debugging** — JSON can be inspected and edited by humans; fix the JSON and re-convert.
- **The tool is independently testable** — once `json2xlsx` correctness is established, it can be reused without re-testing every time.

In short: the design philosophy is to "have the AI produce data, not code" — inserting a JSON layer absorbs LLM nondeterminism.

## Features

- Lightweight CLI that runs on Go 1.22+.[2]
- Main dependencies are `excelize` (XLSX read/write) and `jsonschema/v6` (JSON validation).[1]
- Accepts JSON that is aware of SheetJS-style Cell Objects.[4][3]
- Can progressively support basic tables, formulas, newlines, borders, colors, number formats, links, etc.[6][9][10]
- Because the AI generation part is separated, it can be combined with any LLM.[11][12]

## Installation

In the initial phase, the tool is intended to be built and used locally.

```bash
git clone git@github.com:onohiroki/json2xlsx.git
cd json2xlsx
go build -o json2xlsx ./cmd/json2xlsx
```

Support for `go install` is planned in the future.

```bash
go install json2xlsx/cmd/json2xlsx@latest
```

## Usage

`json2xlsx` is a single-binary CLI that converts between XLSX and JSON. Subcommands are `to-json` (XLSX → JSON), `to-xlsx` (JSON → XLSX), `to-md` (JSON / XLSX → Markdown table), `to-html` (JSON / XLSX → HTML `<table>`), and `to-csv` (csvtk csv2json reverse). If the subcommand is omitted, it behaves as `to-xlsx` by default.

### `to-json` — XLSX → JSON

Read an XLSX and output JSON in a format acceptable to `json2xlsx` (cell map format).

```bash
json2xlsx to-json -i input.xlsx -o output.json
json2xlsx to-json -i input.xlsx -o output.json --date-serial
json2xlsx to-json -i input.xlsx -o output.json --date-display
json2xlsx to-json -i input.xlsx -o output.json --date-rfc3339
```

If `-i` is omitted, standard input is used; if `-o` is omitted, standard output is used.

```bash
cat input.xlsx | json2xlsx to-json > output.json
```

Date/time cells (`t: "d"`) default to outputting Excel's internal serial value in `v`. Only with `--date-display` will the display string be used in `v`. Only with `--date-rfc3339` will the serial be reinterpreted to RFC3339 (UTC). `--date-serial` outputs the Excel serial as a number. Time-only values (`9:05`) are treated as time without a date.

### `to-xlsx` — JSON → XLSX (default)

Read JSON and output `.xlsx`. Use `--sheet` to set the default sheet name when none is specified.

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx --sheet Sheet1
```

Omitting the subcommand has the same behavior.

```bash
json2xlsx -i input.json -o output.xlsx
```

Standard input is also supported.

```bash
cat input.json | json2xlsx to-xlsx -o output.xlsx
```

### `to-md` — JSON / XLSX → Markdown

Convert a Workbook to Markdown tables. Input accepts both JSON (json2xlsx-compatible Workbook) and XLSX; auto-detection is done via the first 4 magic bytes (`PK\x03\x04`). Useful as an intermediate representation to show to AI or to inspect with `cat`.

```bash
json2xlsx to-md -i input.json -o output.md
json2xlsx to-md -i input.xlsx -o output.md
cat input.xlsx | json2xlsx to-md > output.md
```

#### Options

- `--mode f` (default): show formula if present (`=B1*2`), otherwise show `v`.
- `--mode v`: prefer `v`. If `v` is missing, fall back to the formula `=B1*2`.
- `--mode both`: when both `v` and a formula exist, display both like `84<br />=B1*2`.
- `--first-row-header`: treat the first row as the table header. Suppress A/B/C column names and row numbers.

Long options are shown in `--name` form. Short `-i` / `-o` remain single-dash. Single-dash variants (e.g. `-mode`) may also be accepted, but the docs use `--` for consistency.

#### Output example

`--mode f` (default):

```text
|   | A | B | C | D |
| --- | --- | --- | --- | --- |
| 1 | 製品 | 数量 | 単価 | 合計 |
| 2 | 商品A | 100 | 5000 | =B2*C2 |
```

`--first-row-header` (treat first row as header):

```text
| 製品 | 数量 | 単価 | 合計 |
| --- | --- | --- | --- |
| 商品A | 100 | 5000 | =B2*C2 |
```

#### Multiple sheets

When passing a multi-sheet Workbook, tables are concatenated with `## <sheet name>` headings per sheet. For single-sheet input, the heading is omitted.

#### Notes

- Characters inside cells such as `|`, `\`, and newlines are escaped for GFM table safety (`\|`, `\\`, `<br />`).
- Styles (color, borders, fonts), column widths, and row heights are not reflected in Markdown.
- Merged cells output only the value of the top-left cell; other cells are empty.

### `to-html` — JSON / XLSX → HTML `<table>`

Convert a Workbook to an HTML `<table>` fragment. Input detection works like `to-md` using magic bytes.

```bash
json2xlsx to-html -i input.json -o output.html
json2xlsx to-html -i input.xlsx -o output.html
cat input.xlsx | json2xlsx to-html > output.html
```

#### Options

- `--mode v` (default): formula cells display their computed value (`v`). No header row.
- `--mode f`: show formulas (`=B2*C2`). Column names A/B/C and row numbers are shown in `<th>`.
- `--mode both`: show both `v` and formula like `v<br />=f`.

#### Style mapping

- Fill → `background-color`
- Font.Bold → `font-weight: bold`
- Font.Color → `color`
- Font.Italic → `font-style: italic`
- Font.Size → `font-size`
- Alignment → `text-align`, `vertical-align`, `white-space`
- Border → `border`

### `to-csv` — csvtk / xlsx-cli JSON -> CSV

Convert JSON output by `csvtk csv2json` (array of objects) and JSON output by `xlsx-cli -j` to CSV. The `xlsx-cli -j` format processes only the first sheet and ignores the sheet-name row; it errors if there is no array after the sheet-name row. It does not accept json2xlsx's Workbook format (the output of `to-json`) and will exit with an error.

```bash
json2xlsx to-csv -i input.json -o output.csv
cat input.json | json2xlsx to-csv > output.csv
```

## Input JSON concepts

`json2xlsx` accepts JSON inspired by SheetJS compatibility and converts it to `.xlsx` on a per-cell basis.[3][4]

The expected input representations are three kinds:

- Array-of-objects form
- Cell reference form (`A1`, `B2`, etc.)
- Cell Object form

### Example: Cell Object form

```json
{
  "cells": {
    "A1": {"t": "s", "v": "製品"},
    "B1": {"t": "s", "v": "数量"},
    "C1": {"t": "s", "v": "単価"},
    "D1": {"t": "s", "v": "合計", "s": 1},
    "A2": {"t": "s", "v": "商品A\n特価"},
    "B2": {"t": "n", "v": 100},
    "C2": {"t": "n", "v": 5000},
    "D2": {"t": "f", "f": "B2*C2", "v": 500000, "s": 1}
  },
  "styles": [
    {
      "id": 1,
      "fill": {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]},
      "border": [{"style": "thin", "color": "#000000"}],
      "numFmt": "#,##0"
    }
  ]
}
```

## Planned features

| Feature | Initial support | Notes |
|------|----------|------|
| Basic table generation | Yes | placement of strings and numbers [1] |
| Formulas | Yes | assumes cell-reference formulas [6] |
| Newlines in cells | Yes | treat `\n` as newline [10][13] |
| Borders | Yes | thin / medium, etc. [9] |
| Background color | Yes | use Fill [14] |
| Number formats | Yes | equivalent to `z` / `numFmt` [15] |
| Hyperlinks | Yes | specified via `L` field |
| Merged cells | Yes | specified with `merges` array |
| Rich text | No | out of initial scope [4] |

## Go dependencies

Current dependencies are:

```go
require (
    github.com/xuri/excelize/v2 v2.8.1
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
)
```

`excelize` is a widely used OSS library for reading and writing Excel files in Go, and `jsonschema` is used for JSON Schema validation.[2][1]

## Proposed Go data structures

```go
type Cell struct {
    T string      `json:"t,omitempty"`
    V interface{} `json:"v,omitempty"`
    F string      `json:"f,omitempty"`
    Z string      `json:"z,omitempty"`
    S int         `json:"s,omitempty"`
    L interface{} `json:"l,omitempty"`
}

type Fill struct {
    Type    string   `json:"type"`
    Pattern int      `json:"pattern"`
    Color   []string `json:"color"`
}

type Border struct {
    Style string `json:"style"`
    Color string `json:"color"`
    Side  string `json:"side,omitempty"`
}

type Font struct {
    Name   string  `json:"name,omitempty"`
    Size   float64 `json:"size,omitempty"`
    Bold   bool    `json:"bold,omitempty"`
    Italic bool    `json:"italic,omitempty"`
    Color  string  `json:"color,omitempty"`
}

type Alignment struct {
    Horizontal string `json:"horizontal,omitempty"`
    Vertical   string `json:"vertical,omitempty"`
    WrapText   bool   `json:"wrapText,omitempty"`
}

type Style struct {
    ID        int        `json:"id"`
    Fill      *Fill      `json:"fill,omitempty"`
    Border    []Border   `json:"border,omitempty"`
    Font      *Font      `json:"font,omitempty"`
    Alignment *Alignment `json:"alignment,omitempty"`
    NumFmt    string     `json:"numFmt,omitempty"`
}
```

## Relationship with `sheetjs-json-writer`

The separate `sheetjs-json-writer` is a SKILL.md intended to impose the following constraints on AI:

- Output only JSON.
- Do not add Markdown explanations.
- Use fields like `t`, `v`, `f`, `s` correctly.
- Emit formulas, newlines, and styles in a defined format.[4][6][3]

This allows `json2xlsx` to remain simple under the assumption that "correctly formatted JSON will be provided".

## Licensing

This arrangement is suitable for OSS publication. `excelize` is BSD 3-Clause and `jsonschema/v6` is Apache 2.0, but this tool itself does not include AI calls.

A reimplementation referencing SheetJS-compatible specs can also be organized as a compatible implementation.[5][4]

## Development status

Implementation progressed in the following order and all items are completed:

1. ✅ JSON reading
2. ✅ Basic table output
3. ✅ Cell Object support
4. ✅ Formula support
5. ✅ Style support
6. ✅ Newlines, column widths, row heights, links support
7. ✅ Test coverage

## Deliverables

This repository contains the following:

- ✅ `README.md`
- ✅ `SKILL.md` (for `sheetjs-json-writer`)
- ✅ Go implementation
- ✅ Sample JSON (under `test_data/`)

## References

- Excelize documentation / package reference[1]
- Excelize repository[2]
- SheetJS API reference[3]
- SheetJS Cell Objects[4]
- SheetJS license[5]

Sources
[1] excelize package - github.com/xuri/excelize/v2 https://pkg.go.dev/github.com/xuri/excelize/v2
[2] qax-os/excelize: Go language library for reading and ... https://github.com/qax-os/excelize
[3] API Reference https://docs.sheetjs.com/docs/api/
[4] Cell Objects https://docs.sheetjs.com/docs/csf/cell/
[5] License https://docs.sheetjs.com/docs/miscellany/license/
[6] How do I save data using `json_to_sheet` to save formulas? https://github.com/SheetJS/sheetjs/issues/2017
[7] はじめに · Excelize ドキュメンテーション - Ri Xu Online https://xuri.me/excelize/ja/
[8] openai-go/LICENSE at main https://github.com/openai/openai-go/blob/main/LICENSE
[9] Cell · Excelize Document - Ri Xu Online https://xuri.me/excelize/en/cell.html
[10] セル · Excelize ドキュメンテーション - Ri Xu Online https://xuri.me/excelize/ja/cell.html
[11] How to write better prompts for GitHub Copilot https://github.blog/developer-skills/github/how-to-write-better-prompts-for-github-copilot/
[12] Best practices for using GitHub Copilot https://docs.github.com/en/copilot/get-started/best-practices
[13] How can i wraptext to the new row in excelize https://stackoverflow.com/questions/74632921/how-can-i-wraptext-to-the-new-row-in-excelize
[14] Style · Excelize Document - Ri Xu Online https://xuri.me/excelize/en/style.html
[15] Number Formats https://docs.sheetjs.com/docs/csf/features/nf/
