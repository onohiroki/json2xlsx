---
name: sheetjs-json-writer
description: Instruct the AI to output only SheetJS-style JSON that can be converted to XLSX with `json2xlsx`.
---

# sheetjs-json-writer

## Purpose
Receive natural-language table instructions from the user and output **only** SheetJS-style JSON that `json2xlsx` can interpret.

## Absolute rules
- Output must be JSON only. Do not include any explanatory text, preface, or postface.
- Do not wrap the output in a Markdown code fence (triple backticks); the first character must be `{`.
- Do not include any non-JSON characters at all.
- **When instructed to perform calculations, aggregations, or analysis (e.g., averages, totals, summaries), prioritize writing Excel formulas over pre-computed values. Use cell references in formula objects (`t: "f"`, `f: "SUM(A1:A3)"`, etc.) so the resulting XLSX is recalculatable on the sheet. You may include the computed result in `v` alongside the formula, but the formula (`f`) is required.**

## Top-level structures

Three formats are supported:

**Flat (single sheet):**
```json
{"name":"Sheet1","cells":{...},"rows":[...],"cols":[...],"rowDims":[...],"merges":[...],"styles":[...]}
```

**Flat (multiple sheets):**
```json
{"sheets":[{"name":"...","cells":{...}},...],"styles":[...]}
```

**Book wrapper (with charts):**
```json
{
  "version": "0.2",
  "book": {
    "props": {},
    "sheets": {"Sheet1":{"cells":{...}}, "Sheet2":{"cells":{...}}},
    "charts": [],
    "styles": []
  }
}
```

`book.sheets` is an object whose keys are sheet names. Chart definitions may be placed in `book.charts`.

## Cell objects (`cells`)
Keys are cell addresses like `A1`. Each value may contain the following fields:

| Key | Meaning | Example |
|-----|---------|--------|
| `t` | Cell type: `s` string / `n` number / `b` boolean / `f` formula / `d` date | `"n"` |
| `v` | Value | `100` |
| `f` | Formula (used when `t` is `"f"`) | `"B2*C2"` |
| `z` | Number format code | `"#,##0"` |
| `s` | Reference to `styles[].id` | `1` |
| `l` | Hyperlink (string or `{target, tooltip}`) | `"https://..."` |

## Notation rules
- Formulas: no `=` prefix. Line breaks: use `\n` (requires `wrapText: true` style). Colors: `#RRGGBB` only. Reference styles via `s` field.

## Style definitions (`styles`)
```json
{
  "id": 1,
  "fill":   {"type": "pattern", "pattern": 1, "color": ["#E0EBF5"]},
  "border": [{"style": "thin", "color": "#000000"}],
  "font":   {"name": "Calibri", "size": 11, "bold": true, "color": "#000000"},
  "alignment": {"horizontal": "center", "vertical": "center", "wrapText": true},
  "numFmt": "#,##0"
}
```
- If `border[].side` is omitted, the border applies to all four sides. Use `"left"|"right"|"top"|"bottom"` to specify a side.
- Border styles: `thin`, `medium`, `thick`, `dashed`, `dotted`, `double`, etc.

## Other
- Column width: `"cols": [{"col": "A", "width": 18}]`
- Row height: `"rowDims": [{"row": 1, "height": 24}]`
- Merge ranges: `"merges": [{"range": "A1:B1"}]`
- Freeze panes: `"freeze": {"row": 1}` (freeze header rows) or `{"col": 1}` (freeze left columns)

## Example: sales table (single sheet)
```json
{
  "name": "Sales",
  "cells": {
    "A1":{"t":"s","v":"Product","s":2},"B1":{"t":"s","v":"Quantity","s":2},
    "C1":{"t":"s","v":"Unit Price","s":2},"D1":{"t":"s","v":"Total","s":2},
    "A2":{"t":"s","v":"Product A\nSpecial"},"B2":{"t":"n","v":100},
    "C2":{"t":"n","v":5000,"s":1},"D2":{"t":"f","f":"B2*C2","s":1}
  },
  "styles":[
    {"id":1,"numFmt":"#,##0","border":[{"style":"thin","color":"#000000"}]},
    {"id":2,"fill":{"type":"pattern","pattern":1,"color":["#E0EBF5"]},"border":[{"style":"thin","color":"#000000"}],"font":{"bold":true},"alignment":{"horizontal":"center","wrapText":true}}
  ]
}
```

## Example: sales management (multiple sheets)
```json
{
  "sheets": [
    {
      "name": "Sales",
      "cells": {
        "A1":{"t":"s","v":"Product","s":2},"B1":{"t":"s","v":"Quantity","s":2},
        "C1":{"t":"s","v":"Unit Price","s":2},"D1":{"t":"s","v":"Total","s":2},
        "A2":{"t":"s","v":"Product A"},"B2":{"t":"n","v":100},
        "C2":{"t":"n","v":5000,"s":1},"D2":{"t":"f","f":"B2*C2","s":1}
      },
      "merges":[],
      "rowDims":[],
      "cols":[{"col":"A","width":14},{"col":"B","width":10},{"col":"C","width":10},{"col":"D","width":12}]
    },
    {
      "name": "Product Master",
      "cells": {
        "A1":{"t":"s","v":"Product Code","s":2},"B1":{"t":"s","v":"Product Name","s":2},
        "C1":{"t":"s","v":"Unit Price","s":2},
        "A2":{"t":"s","v":"A-001"},"B2":{"t":"s","v":"Product A"},
        "C2":{"t":"n","v":5000,"s":1}
      }
    }
  ],
  "styles":[
    {"id":1,"numFmt":"#,##0","border":[{"style":"thin","color":"#000000"}]},
    {"id":2,"fill":{"type":"pattern","pattern":1,"color":["#E0EBF5"]},"border":[{"style":"thin","color":"#000000"}],"font":{"bold":true},"alignment":{"horizontal":"center","wrapText":true}}
  ]
}
```

## Charts

In the book wrapper format, `book.charts` may contain an array of chart objects.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id`    | string | optional | Unique chart identifier |
| `t`     | string | optional | Use `"chart"` for charts |
| `mode`  | string | optional | `"embedded"` (in-sheet) or `"chartSheet"` (separate sheet) |
| `ct`    | string | required | `col`, `bar`, `line`, `area`, `pie`, `doughnut`, `scatter`, `radar`, `combo` |
| `sheet` | string | required | Target sheet name |
| `anchor`| string | embedded only | Top-left cell address (A1) |
| `ser`   | array  | required | Array of series (one or more) |
| `dim`   | object | optional | Size and offsets (`w`, `h`, `offx`, `offy`) |
| `title` | object | optional | `{ "tx": "Title" }` |
| `legend`| object | optional | `{ "show": true, "pos": "bottom" }` |
| `plot`  | object | optional | Plot area options |
| `xAxis` | object | optional | X-axis settings |
| `yAxis` | object | optional | Y-axis settings |

Series (`ser[]`) fields:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Series name |
| `cat`  | string | Category range (A1, e.g. `"Sheet1!$A$2:$A$13"`) |
| `val`  | string | Value range (A1) |
| `line` | object | Line style (`{ "width": 1.5 }`) |
| `fill` | object | Fill style (`{ "color": "#4472C4" }`) |

Example (embedded):
```json
{
  "version": "0.2",
  "book": {
    "sheets": {
      "Sheet1": { "cells": { "A1": {"t":"s","v":"Month"}, "B1": {"t":"s","v":"Sales"} } }
    },
    "charts": [
      {
        "id": "ch1",
        "t": "chart",
        "mode": "embedded",
        "ct": "col",
        "sheet": "Sheet1",
        "anchor": "D2",
        "title": { "tx": "Monthly Sales" },
        "ser": [
          { "name": "Sales", "cat": "Sheet1!$A$2:$A$13", "val": "Sheet1!$B$2:$B$13" }
        ]
      }
    ]
  }
}
```

For chartSheet mode, set `mode` to `"chartSheet"`, omit `anchor`, and use a new sheet name in `sheet`.

## Validation with json2xlsx
- Validate before final output:
  ```
  json2xlsx -i output.json > /dev/null      # Linux / macOS
  json2xlsx -i output.json                  # Windows PowerShell
  ```
- If an error is reported, fix the JSON and re-test until exit code 0.
- Common issues: **invalid cell name** (use A-Z, AA-AZ), **additional properties** (wrong top-level structure), **missing property** (`cells` key required), **invalid enum** (use allowed values, e.g. `"bottom"` not `"b"`).

## Self-healing on errors
- If a JSON syntax error, unknown schema field, or undefined style ID is detected, output **only** the corrected JSON (no explanatory text).
