# json2xlsx

[ English | [日本語](README.ja.md) ]

`json2xlsx` is a Go-based CLI tool that generates Excel `.xlsx` files from SheetJS-style JSON or 2D array JSON. The Go side is responsible only for the JSON → XLSX conversion and does not include any AI calls.[1][2]

It is intended to be used together with the `sheetjs-json-writer` skill (included in the `skills/` directory) to make AI reliably output SheetJS-style JSON.[3][4]

## Purpose

The goal of this project is to separate the following two stages:

1. The AI emits tables and aggregate results as SheetJS-style JSON.
2. `json2xlsx` reads that JSON and converts it to `.xlsx`.[5]

While the AI primarily outputs SheetJS-style JSON, `json2xlsx` also accepts 2D array JSON input for convenience when interacting with other tools.

This separation keeps the Go tool lightweight, easy to test, and suitable for OSS distribution.

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
- Accepts JSON in both SheetJS-style Cell Objects and simple 2D array formats.[4][3]
- Supports formulas, newlines, borders, colors, number formats, links, freeze panes, etc. (Note: 2D array format does not support formulas or styles).
- Because the AI generation part is separated, it can be combined with any LLM.
- **Chart generation** — bar, column, line, area, pie, doughnut, scatter, radar charts with customizable titles, legends, axes, and data labels.
- **Picture / image support** — embed images in cells from file paths or base64 data; includes sheet background images.
- **Japanese-friendly** — series names containing Japanese characters (e.g., `予算`, `実績`) are automatically preserved in Excel legends.

## Installation

### `go install` (recommended)

```bash
go install github.com/onohiroki/json2xlsx/cmd/json2xlsx@latest
```

### Build from source

```bash
git clone git@github.com:onohiroki/json2xlsx.git
cd json2xlsx
go build -o json2xlsx ./cmd/json2xlsx
```

## Usage

`json2xlsx` is a single-binary CLI that converts between XLSX and JSON. Subcommands are `to-json` (XLSX → JSON), `to-xlsx` (JSON → XLSX), `to-md` (JSON / XLSX → Markdown table), `to-html` (JSON / XLSX → HTML `<table>`), and `to-csv` (JSON / XLSX → CSV). If the subcommand is omitted, it behaves as `to-xlsx` by default.

### `to-json` — XLSX → JSON

Read an XLSX and output JSON in a format acceptable to `json2xlsx` (cell map format).

```bash
json2xlsx to-json -i input.xlsx -o output.json
json2xlsx to-json -i input.xlsx -o output.json --date-display
json2xlsx to-json -i input.xlsx -o output.json --date-rfc3339
json2xlsx to-json -i input.xlsx -o output.json --image-mode file
```

If `-i` is omitted, standard input is used; if `-o` is omitted, standard output is used.

```bash
cat input.xlsx | json2xlsx to-json > output.json
```

Date/time cells (`t: "d"`) default to outputting Excel's internal serial value as a number in `v`. Use `--date-display` to output the display string instead, or `--date-rfc3339` to output RFC3339 (UTC) strings. The `--date-serial` flag is a compatibility alias that behaves identically to the default. Time-only values (`9:05`) are treated as time without a date.

**Image output mode** — `--image-mode` controls how embedded images in the XLSX are exported:
- `base64` (default): images are embedded directly in the JSON output as base64-encoded strings in the `data` field.
- `file`: images are written to separate files on disk (named `image_1.png`, `image_2.jpg`, etc.) alongside the JSON output. The `path` field in the JSON will reference the file.

### `to-xlsx` — JSON → XLSX (default)

Read JSON and output `.xlsx`. Input is expected in SheetJS-style Workbook JSON by default. With `--data-json`, accepts 2D array (e.g. `[["A", 1], ["B", 2]]`), array of objects, or map-of-arrays JSON. Note that data JSON formats (`--data-json`) do not support formulas or styles.

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx
json2xlsx to-xlsx -i data.json -o output.xlsx --data-json
```

Use `--compute` to evaluate formulas (`t: "f"`) that lack a cached value (`v`). Cells that fail evaluation are skipped with a warning on stderr.

```bash
json2xlsx to-xlsx -i input.json -o output.xlsx --compute
```

### Formula engine

The built-in formula engine evaluates formulas cell-by-cell. Below is the full list of supported features.

**Arithmetic operators:** `+`, `-`, `*`, `/`

**Comparison operators:** `<`, `>`, `=`, `<=`, `>=`, `<>` (not equal). Comparison results are `1` (true) or `0` (false).

**Supported functions:**

| Function | Description |
|----------|-------------|
| `SUM(n1, n2, ...)` | Sum of numbers |
| `AVERAGE(n1, n2, ...)` | Arithmetic mean |
| `COUNT(n1, n2, ...)` | Count of numeric cells |
| `COUNTA(n1, n2, ...)` | Count of non-empty cells |
| `MIN(n1, n2, ...)` | Minimum value |
| `MAX(n1, n2, ...)` | Maximum value |
| `ABS(x)` | Absolute value |
| `ROUND(x, digits)` | Round to specified digits |
| `ROUNDUP(x, digits)` | Round away from zero |
| `ROUNDDOWN(x, digits)` | Round toward zero |
| `INT(x)` | Integer portion |
| `TRUNC(x, digits)` | Truncate toward zero |
| `SIGN(x)` | Sign of x (-1, 0, or 1) |
| `PI()` | π (3.14159...) |
| `RAND()` | Random number in [0, 1) |
| `PRODUCT(n1, n2, ...)` | Multiply numbers |
| `SUMPRODUCT(a1, a2, ...)` | Sum of element-wise products |
| `POWER(x, y)` | x raised to the power y |
| `SQRT(x)` | Square root |
| `LN(x)` | Natural logarithm |
| `LOG(x, [base])` | Logarithm (default: natural) |
| `LOG10(x)` | Base-10 logarithm |
| `EXP(x)` | e raised to the power x |
| `MOD(x, y)` | Remainder of x / y |
| `FLOOR(x, significance)` | Round down to nearest multiple |
| `CEILING(x, significance)` | Round up to nearest multiple |
| `SIN(x)` | Sine (radians) |
| `COS(x)` | Cosine (radians) |
| `TAN(x)` | Tangent (radians) |
| `ASIN(x)` | Arc sine |
| `ACOS(x)` | Arc cosine |
| `ATAN(x)` | Arc tangent |
| `ATAN2(x, y)` | Arc tangent of y/x |
| `DEGREES(x)` | Convert radians to degrees |
| `RADIANS(x)` | Convert degrees to radians |
| `SINH(x)` | Hyperbolic sine |
| `COSH(x)` | Hyperbolic cosine |
| `TANH(x)` | Hyperbolic tangent |
| `ASINH(x)` | Inverse hyperbolic sine |
| `ACOSH(x)` | Inverse hyperbolic cosine |
| `ATANH(x)` | Inverse hyperbolic tangent |
| `FACT(x)` | Factorial |
| `MEDIAN(n1, n2, ...)` | Median value |
| `STDEV.S(n1, n2, ...)` | Sample standard deviation |
| `STDEV.P(n1, n2, ...)` | Population standard deviation |
| `VAR.S(n1, n2, ...)` | Sample variance |
| `VAR.P(n1, n2, ...)` | Population variance |
| `RANK(x, range, [order])` | Rank of x in range (0=desc, non-zero=asc) |
| `RANK.EQ(x, range, [order])` | Alias for RANK |
| `LARGE(range, k)` | k-th largest value |
| `SMALL(range, k)` | k-th smallest value |
| `IF(cond, t_val, f_val)` | Conditional (cond != 0 → t_val, else f_val) |
| `IFERROR(expr, fallback)` | Return fallback if expr errors |
| `AND(n1, n2, ...)` | Logical AND (1 if all non-zero) |
| `OR(n1, n2, ...)` | Logical OR (1 if any non-zero) |
| `NOT(x)` | Logical NOT (1 if zero) |
| `SUMIF(check_range, criteria, [sum_range])` | Sum cells matching criteria |
| `COUNTIF(range, criteria)` | Count cells matching criteria |
| `AVERAGEIF(check_range, criteria, [avg_range])` | Average cells matching criteria |
| `SUMIFS(sum_range, crit_range1, crit1, ...)` | Sum with multiple criteria |
| `COUNTIFS(crit_range1, crit1, ...)` | Count with multiple criteria |
| `AVERAGEIFS(avg_range, crit_range1, crit1, ...)` | Average with multiple criteria |
| `VLOOKUP(value, table, col_index)` | Vertical lookup (exact match) |
| `XLOOKUP(value, lookup_arr, return_arr, [not_found])` | Modern lookup with optional default |
| `INDEX(range, row, [col])` | Value at given row/col |
| `MATCH(value, range, match_type)` | Position of value in range |
| `CHOOSE(index, val1, val2, ...)` | Select by 1-based index |
| `TODAY()` | Current date as serial number |
| `NOW()` | Current date and time as serial number |

**Limitations:**

- Only numeric values are supported. Text functions (e.g. `CONCAT`, `LEFT`, `FIND`) and string comparisons in criteria are **not** available.
- Range references (e.g. `A1:A10`) are valid only inside function arguments; standalone ranges produce an error.
- Cell references use A1-style only (no R1C1). Column letters are limited to 3 characters (`A`–`ZZZ`).
- Cross-sheet references are **not** supported.
- Array formulas, volatile flags, and iterative calculation are **not** supported.
- Circular references are detected and reported as warnings. All other evaluation errors cause the cell to be skipped.

Omitting the subcommand has the same behavior.

```bash
json2xlsx -i input.json -o output.xlsx
```

Standard input is also supported.

```bash
cat input.json | json2xlsx to-xlsx -o output.xlsx
```

### `to-md` — JSON / XLSX → Markdown

Convert a Workbook to Markdown tables. Input accepts both JSON (json2xlsx-compatible Workbook) and XLSX; auto-detection is done via the first 4 magic bytes (`PK\x03\x04`). Useful as an intermediate representation to show to AI or to inspect with `cat`. Use `--compute` to evaluate formulas before output.

```bash
json2xlsx to-md -i input.json -o output.md
json2xlsx to-md -i input.xlsx -o output.md
json2xlsx to-md -i input.json -o output.md --compute
cat input.xlsx | json2xlsx to-md > output.md
```

#### Options

- `--mode f` (default): show formula if present (`=B1*2`), otherwise show `v`.
- `--mode v`: prefer `v`. If `v` is missing, fall back to the formula `=B1*2`.
- `--mode both`: when both `v` and a formula exist, display both like `84<br />=B1*2`.
- `--first-row-header`: treat the first row as the table header. Suppress A/B/C column names and row numbers.
- `--data-json`: treat JSON input as data JSON (2D array, array of objects, or map-of-arrays).

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
json2xlsx to-html -i input.json -o output.html --grid
cat input.xlsx | json2xlsx to-html > output.html
```

#### Options

- `--mode v` (default): formula cells display their computed value (`v`). No header row.
- `--mode f`: show formulas (`=B2*C2`). Column names A/B/C and row numbers are shown in `<th>`.
- `--mode both`: show both `v` and formula like `v<br />=f`.
- `--data-json`: treat JSON input as data JSON (2D array, array of objects, or map-of-arrays).
- `--grid`: show thin gray borders on all cells, including empty ones (collapses cellspacing).
- `--compute`: evaluate formulas before output.

#### Style mapping

- Fill → `background-color`
- Font.Bold → `font-weight: bold`
- Font.Color → `color`
- Font.Italic → `font-style: italic`
- Font.Size → `font-size`
- Alignment → `text-align`, `vertical-align`, `white-space`
- Border → `border`

### `to-csv` — JSON / XLSX -> CSV

Convert JSON or XLSX to CSV. Supports json2xlsx Workbook JSON, `csvtk csv2json` output, and `xlsx-cli -j` output. With `--data-json`, accepts 2D array, array of objects, or map-of-arrays JSON.

```bash
json2xlsx to-csv -i input.json -o output.csv
json2xlsx to-csv -i data.json -o output.csv --data-json
json2xlsx to-csv -i input.xlsx -o output.csv --sheet "Sheet1"
json2xlsx to-csv -i input.xlsx -o output.csv --sheet-index 1
cat input.json | json2xlsx to-csv > output.csv
```

Options:

- `--sheet`: extract a specific sheet by name (for multi-sheet XLSX or Workbook JSON).
- `--sheet-index`: extract a sheet by 1-based index (for multi-sheet XLSX or Workbook JSON).
- `--compute`: evaluate formulas before output.

## Input JSON concepts

`json2xlsx` accepts JSON inspired by SheetJS compatibility and converts it to `.xlsx` on a per-cell basis.[3][4]

The expected input representations come in two modes:

**Default (no `--data-json`):** expects SheetJS-style Workbook / Sheet JSON (cell reference or Cell Object form).

**With `--data-json`:** accepts the following data-oriented formats:
- 2D array JSON (e.g. `[["Header1", "Header2"], ["val1", 123]]`)
- Array-of-objects form (e.g. `[{"Name": "Alice", "Age": 30}, ...]`)
- Map-of-arrays form (e.g. `{"Name": ["Alice", "Bob"], "Age": [30, 25]}`)

Note: data JSON formats are for pure data and do not support formulas or styles. For formulas and styles, use the SheetJS-style Cell Object form.

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

### Example: Map-of-arrays form

Keys preserve their JSON declaration order to become the header row; each array becomes a column. Arrays of different lengths are padded with `null`.

```json
{
  "name": ["Alice", "Bob", "Carol"],
  "age":  [30,      25,    41],
  "city": ["Tokyo", "Osaka", "Nagoya"]
}
```

This produces:

| name | age | city |
|------|-----|------|
| Alice | 30 | Tokyo |
| Bob | 25 | Osaka |
| Carol | 41 | Nagoya |

See `samples/table_map_of_array.json` for a full example.

## Charts

Charts are supported in the `book` wrapper format via the `charts` array. Each chart object can be embedded in a sheet or placed on its own chart sheet.

### Chart object fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Chart identifier |
| `t` | string | Always `"chart"` |
| `mode` | string | `"embedded"` (default, anchored in a sheet) or `"chartSheet"` (standalone chart sheet) |
| `ct` | string | Chart type: `"col"`, `"bar"`, `"line"`, `"area"`, `"pie"`, `"doughnut"`, `"scatter"`, `"radar"` |
| `sheet` | string | Sheet name for embedded charts |
| `anchor` | string | Anchor cell (e.g. `"E2"`) for embedded charts |
| `dim` | object | `{w, h, offx, offy, sx, sy}` — width/height in pixels, offsets in EMU, scale factors |
| `title` | object | `{tx, overlay}` — chart title text and overlay flag |
| `legend` | object | `{show, pos}` — `pos`: `"top"`, `"bottom"`, `"left"`, `"right"`, `"topRight"` |
| `xAxis` | object | `{title, minimum, maximum, majorUnit, minorUnit, reverseOrder, majorGridLines, minorGridLines, numFmt}` |
| `yAxis` | object | Same as `xAxis` |
| `plot` | object | `{varyColors, showBlanksAs}` — plot area options |
| `ser` | array | Array of series objects (see below) |

### Series object fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Series name (literal string like `"予算"` or cell reference like `"Sheet1!$A$1"`) |
| `cat` | string | Categories range (e.g. `"部門予算!$A$2:$A$7"`) |
| `val` | string | Values range (e.g. `"部門予算!$B$2:$B$7"`) |
| `xVal` | string | X-values range (scatter only) |
| `yVal` | string | Y-values range (scatter only) |
| `bubble` | string | Bubble size range (bubble chart only) |
| `line` | object | `{width}` — line width in pt |
| `fill` | object | `{color}` — fill color (e.g. `"#FF0000"`) |
| `marker` | object | `{symbol, size}` — marker symbol and size |
| `dLbls` | object | `{showVal, showCatName, showSerName, showPercent, showLeaderLn}` — data labels |

### Example

```json
{
  "version": "0.2",
  "book": {
    "sheets": {
      "部門予算": {
        "cells": {
          "A1": { "t": "s", "v": "部門" },
          "B1": { "t": "s", "v": "予算(百万円)" },
          "A2": { "t": "s", "v": "営業" }, "B2": { "t": "n", "v": 120 },
          "A3": { "t": "s", "v": "開発" }, "B3": { "t": "n", "v": 200 }
        }
      }
    },
    "charts": [
      {
        "id": "chart1",
        "t": "chart",
        "mode": "embedded",
        "ct": "col",
        "sheet": "部門予算",
        "anchor": "E2",
        "dim": { "w": 640, "h": 360 },
        "title": { "tx": "部門別予算と実績" },
        "legend": { "pos": "bottom" },
        "ser": [
          { "name": "予算", "cat": "部門予算!$A$2:$A$7", "val": "部門予算!$B$2:$B$7" },
          { "name": "実績", "cat": "部門予算!$A$2:$A$7", "val": "部門予算!$C$2:$C$7" }
        ]
      }
    ]
  }
}
```

See `samples/chart_bar.json`, `samples/chart_scatter.json`, `samples/chart_timeseries.json` for more examples.

## Pictures

Images can be embedded in sheets via the `pictures` array in a Sheet object. Each picture can reference a file on disk or be provided as base64-encoded data. A sheet-level background image is set via the `background` field.

### Picture object fields

| Field | Type | Description |
|-------|------|-------------|
| `cell` | string | Anchor cell (e.g. `"B1"`) |
| `path` | string | Image file path (relative or absolute) |
| `data` | string | Base64-encoded image data (alternative to `path`) |
| `extension` | string | File extension (`png`, `jpg`, `gif`) — required when using `data` |
| `altText` | string | Alternative text |
| `offsetX` | int | Horizontal offset from the anchor cell in EMU |
| `offsetY` | int | Vertical offset from the anchor cell in EMU |
| `scaleX` | float | Horizontal scale factor |
| `scaleY` | float | Vertical scale factor |
| `positioning` | string | `"oneCell"` (move and size with cells) or `"absolute"` |
| `hyperlink` | string | Hyperlink target URL |
| `printObject` | bool | Whether to print the picture |
| `locked` | bool | Whether the picture is locked |
| `lockAspectRatio` | bool | Whether to lock the aspect ratio |

### Background object fields

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Image file path |
| `data` | string | Base64-encoded image data (alternative to `path`) |
| `extension` | string | File extension — required when using `data` |

### Example

```json
{
  "sheets": [
    {
      "name": "Sheet1",
      "cells": {
        "A1": { "t": "s", "v": "Image from path" },
        "A2": { "t": "s", "v": "Image from base64" }
      },
      "pictures": [
        {
          "cell": "B1",
          "path": "images/sample.png",
          "altText": "Sample image",
          "scaleX": 0.5,
          "scaleY": 0.5
        },
        {
          "cell": "B2",
          "data": "iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAF0lEQVR4nGL5z4APMOGVHbHSgAAAAP//RM4BFjLZ0j4AAAAASUVORK5CYII=",
          "extension": "png",
          "altText": "Image from base64"
        }
      ],
      "background": {
        "path": "images/background.png"
      }
    }
  ]
}
```

See `samples/pictures.json` and `samples/pictures_new.json` for complete examples.

## Planned features

| Feature | Initial support | Notes |
|------|----------|------|
| Basic table generation | Yes | placement of strings and numbers [1] |
| Formulas | Yes | assumes cell-reference formulas |
| Newlines in cells | Yes | treat `\n` as newline |
| Borders | Yes | thin / medium, etc. |
| Background color | Yes | use Fill |
| Number formats | Yes | equivalent to `z` / `numFmt` [6] |
| Hyperlinks | Yes | specified via `L` field |
| Merged cells | Yes | specified with `merges` array |
| Charts | Yes | bar, column, line, area, pie, doughnut, scatter, radar |
| Pictures / images | Yes | file path or base64, sheet background |
| Freeze panes | Yes | freeze rows/columns via `freeze` field |
| Rich text | No | out of initial scope [4] |

## Go dependencies

Current dependencies are:

```go
require (
    github.com/xuri/excelize/v2 v2.8.1
    github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
    github.com/mattn/go-runewidth v0.0.24
    golang.org/x/text v0.14.0
)
```

`excelize` is a widely used OSS library for reading and writing Excel files in Go, `jsonschema` is used for JSON Schema validation, `go-runewidth` is used to calculate text width for Markdown output, and `golang.org/x/text` is used for locale-aware error messages.[2][1]

## Go data structures

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

type ColInfo struct {
    Col   string  `json:"col"`
    Width float64 `json:"width"`
}

type RowInfo struct {
    Row    int     `json:"row"`
    Height float64 `json:"height"`
}

type FreezePane struct {
    Row int `json:"row,omitempty"`
    Col int `json:"col,omitempty"`
}

type Picture struct {
    Cell            string  `json:"cell"`
    Path            string  `json:"path,omitempty"`
    Data            string  `json:"data,omitempty"`
    Extension       string  `json:"extension,omitempty"`
    AltText         string  `json:"altText,omitempty"`
    PrintObject     *bool   `json:"printObject,omitempty"`
    Locked          *bool   `json:"locked,omitempty"`
    LockAspectRatio *bool   `json:"lockAspectRatio,omitempty"`
    OffsetX         int     `json:"offsetX,omitempty"`
    OffsetY         int     `json:"offsetY,omitempty"`
    ScaleX          float64 `json:"scaleX,omitempty"`
    ScaleY          float64 `json:"scaleY,omitempty"`
    Hyperlink       string  `json:"hyperlink,omitempty"`
    Positioning     string  `json:"positioning,omitempty"`
}

type SheetBackground struct {
    Path      string `json:"path,omitempty"`
    Data      string `json:"data,omitempty"`
    Extension string `json:"extension,omitempty"`
}
```

## Relationship with `sheetjs-json-writer`

The `sheetjs-json-writer` skill included in the `skills/` directory is a SKILL.md intended to impose the following constraints on AI:

- Output only JSON.
- Do not add Markdown explanations.
- Use fields like `t`, `v`, `f`, `s` correctly.
- Emit formulas, newlines, and styles in a defined format.[4][3]

This allows `json2xlsx` to remain simple under the assumption that "correctly formatted JSON will be provided".

A reimplementation referencing SheetJS-compatible specs can also be organized as a compatible implementation.[5][4]

## Licensing

This tool is licensed under the **MIT License**.

The licenses of major dependencies are as follows:
- `excelize`: BSD 3-Clause
- `jsonschema/v6`: MIT
- `go-runewidth`: MIT

## References

- Excelize documentation / package reference[1]
- Excelize repository[2]
- SheetJS API reference[3]
- SheetJS Cell Objects[4]
- SheetJS license[5]
- SheetJS Number Formats[6]

Sources
[1] excelize package - github.com/xuri/excelize/v2 https://pkg.go.dev/github.com/xuri/excelize/v2
[2] qax-os/excelize: Go language library for reading and ... https://github.com/qax-os/excelize
[3] API Reference https://docs.sheetjs.com/docs/api/
[4] Cell Objects https://docs.sheetjs.com/docs/csf/cell/
[5] License https://docs.sheetjs.com/docs/miscellany/license/
[6] Number Formats https://docs.sheetjs.com/docs/csf/features/nf/
