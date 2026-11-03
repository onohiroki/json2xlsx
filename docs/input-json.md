# Input JSON comparison

[ English | [日本語](input-json.ja.md) ]

`json2xlsx` accepts three families of input JSON, each designed for a different purpose. This document explains how they differ and when to use each one, based on the actual behavior of the implementation.

- [Back to README](../README.md)
- [Japanese version](input-json.ja.md)

## 1. Overview of the three formats

| Format | Flag | Input shape | Typical use |
|---|---|---|---|
| **SheetJS format** | none | `cells` cell map, `sheets` array, or `book` wrapper | Detailed workbooks, charts, images, XLSX round-trips |
| **Data JSON** | `--data-json` | 2D array, array of objects, or map-of-arrays | Raw data from CSV / SQL / APIs |
| **table JSON** | `--table-json` | `tables` / `header` / `rows` / `columns` | Ordinary tables, reports, summary sheets |

`--data-json` and `--table-json` are mutually exclusive. Passing both is an error.

## 2. Which format should I use?

1. **Do you need charts, images, conditional formatting, sparklines, or fine-grained cell placement?**  
   → Use **SheetJS format**. This is also the format to use when you want to round-trip an XLSX through `to-json`.

2. **Do you need headers, column widths, number formats, and formula templates for an ordinary table?**  
   → Use **table JSON**. It is also the easiest format to generate with an LLM because you do not have to count cell coordinates.

3. **Do you only need values, with no styling or formulas?**  
   → Use **data JSON**. This is the natural choice for CSV output, SQL results, and API responses.

A simple decision flow:

```
Do you need charts, images, conditional formatting, or sparklines?
  └─ yes → SheetJS format
       └─ no → Do you need headers, column formats, or formula templates?
            └─ yes → table JSON
                 └─ no → data JSON
```

## 3. The same table in all three formats

Consider this table:

| Product | Qty | Unit price | Amount |
|---|---:|---:|---:|
| A | 10 | 100 | formula |
| B | 5 | 200 | formula |

### SheetJS format

```json
{
  "cells": {
    "A1": {"t": "s", "v": "Product"},
    "B1": {"t": "s", "v": "Qty"},
    "C1": {"t": "s", "v": "Unit price"},
    "D1": {"t": "s", "v": "Amount"},
    "A2": {"t": "s", "v": "A"},
    "B2": {"t": "n", "v": 10},
    "C2": {"t": "n", "v": 100},
    "D2": {"t": "f", "f": "B2*C2"},
    "A3": {"t": "s", "v": "B"},
    "B3": {"t": "n", "v": 5},
    "C3": {"t": "n", "v": 200},
    "D3": {"t": "f", "f": "B3*C3"}
  }
}
```

### table JSON

```json
{
  "tables": [
    {
      "columns": [
        {"key": "product", "header": "Product"},
        {"key": "qty", "header": "Qty"},
        {"key": "price", "header": "Unit price"},
        {"key": "amount", "header": "Amount", "formula": "={qty}*{price}"}
      ],
      "rows": [
        {"product": "A", "qty": 10, "price": 100},
        {"product": "B", "qty": 5, "price": 200}
      ]
    }
  ]
}
```

### Data JSON (2D array)

```json
[["Product", "Qty", "Unit price"], ["A", 10, 100], ["B", 5, 200]]
```

Data JSON is value-only, so it cannot express a formula column such as “Amount”. Use table JSON or SheetJS format when you need formulas or styling.

### Output comparison

SheetJS format and table JSON produce the same result with `to-md --first-row-header`:

```markdown
| Product | Qty | Unit price | Amount |
| ------- | --: | ---------: | ------ |
| A       |  10 |        100 | =B2*C2 |
| B       |   5 |        200 | =B3*C3 |
```

## 4. Conversion pipeline

The three formats differ only in input parsing. They all converge on the same `Workbook` representation.

```
JSON + flags
├─ SheetJS → UnmarshalWorkbook(false)
│             ├─ normalize SheetNames / Sheets form
│             ├─ strip internal keys such as !ref
│             └─ json.Unmarshal
├─ Data JSON → UnmarshalWorkbook(true)
│             └─ auto-detect 2D array / array of objects / map-of-arrays
└─ table JSON → UnmarshalTableJSON
              └─ field validation and semantic validation

                ↓ normalizeWorkbook
        Workbook{Sheets, Styles}
                ↓
   writeSheet / BuildCellGrid → excelize
```

Command entry points:

| Command | Entry point | Notes |
|---|---|---|
| `to-xlsx` | `ParseWorkbook` | Supports all three formats |
| `to-md` / `to-html` | `ReadWorkbookWithFormat` | Supports all three formats |
| `to-csv` | `ToCSVWithFormat` | SheetJS path now uses `UnmarshalWorkbook` |
| `info` | `ReadBookInfoSourceWithFormat` | Auto-detects JSON format |
| MCP | `loadWorkbook` | `input_path` is auto-detected with `DetectJSONFormat` |

`to-csv` still has a legacy path for arrays without a format flag (the `xlsx-cli` / `csvtk` compatible path). Use `--data-json` explicitly to treat an array as data JSON.

## 5. Command support matrix

`✓` = supported, `✗` = not supported, `△` = conditional.

| Feature | `to-xlsx` | `to-md` | `to-html` | `to-csv` | `info` | MCP |
|---|---|---|---|---|---|---|
| SheetJS format | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Data JSON | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| table JSON | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Auto-detect JSON format | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ |
| `--range` | ✗ | ✓ | ✓ | ✓ | ✗ | preview only |
| Sheet selection | ✗ | ✗ | ✗ | ✓ | `--sheet` | inspect only |
| `--compute` | ✓ | ✓ | ✓ | ✓ | ✗ | ✓ |
| `--autofit` | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |

Notes:

- `to-xlsx` converts the whole workbook, so it has no range or sheet selection.
- `to-md` / `to-html` render every sheet. Only `to-md` prints sheet headings.
- `to-csv` selects one sheet with `--sheet` / `--sheet-index`. For data JSON, the effective name `Sheet1` also works.
- `info` auto-detects the JSON format and accepts `--data-json` / `--table-json` overrides.
- MCP `json2xlsx_preview` accepts `range`, and `json2xlsx_inspect` accepts `sheet`.

## 6. Behavioral differences

### 6.1 Sheet order and names

- `book.sheets` is an object, so its key order is not guaranteed. Use `book.sheetOrder` to control the order.
- Without `book.sheetOrder`, sheets are sorted alphabetically by name.
- `to-json` writes the XLSX sheet order as `book.sheetOrder`.
- Duplicate sheet names are rejected case-insensitively.
- Empty sheet names receive effective names such as `Sheet1` and `Sheet2`. `info` and `to-md` use these effective names.

### 6.2 Headers and columns

| Format | Header | Column count | Missing values |
|---|---|---|---|
| SheetJS | None (place values at A1, B1, ...) | Max coordinate in `cells` / max row length in `rows` | Empty cell |
| Data JSON (2D array) | None (the first row is data too) | Max row length | Empty cell |
| Data JSON (array of objects) | Generated from first-seen key order | Union of all keys | `null` |
| Data JSON (map-of-arrays) | Key declaration order | Number of keys | `null` |
| table JSON | `header` or `columns[].header` | Max of `header`, `columns`, and longest row | Empty cell |

Additional notes:

- For data JSON arrays of objects, column order follows the order in which keys first appear.
- For table JSON object rows, `columns` order wins.
- If a table JSON array row is longer than the header, extra columns are created.
- Unknown keys in table JSON object rows are errors. Extra values in array rows are allowed.
- `freezeHeader` and `autoFilter` require a header.

### 6.3 Formulas

**SheetJS format**

- Use `t: "f"` and `f`. Do not include `=` in `f`.
- If `f` starts with `=`, the preview shows `==1+1`.
- If `t` is omitted but `f` is present, the cell is treated as a formula.
- You may provide a cached value in `v`.

**table JSON**

- Strings starting with `=` are formulas.
- `columns[].formula` defines a formula template.
- `{key}` expands to the current row cell, `{key:col}` to the column letter, and `{r}` to the row number.
- Unknown placeholders are validation errors.
- To write a literal string starting with `=`, use `{"v": "=text"}`.
- Cross-sheet references such as `=SUM(Sales!D2:D5)` are supported.

**Data JSON**

- There are no formulas. `=SUM(1,2)` is written as plain text.
- `to-md` / `to-html` warn when `--mode f` or `--mode both` is explicitly set.

**`--compute` semantics**

- Only formula cells without `v` are evaluated. Existing cached values are trusted.
- Numeric results keep `f` and gain a computed `v`.
- String results change to `t: "s"` and lose `f`.
- `to-csv` cannot display formulas. Without `--compute`, formula cells become empty and a warning is printed.

### 6.4 Styles and number formats

| Format | Style reference | Validation |
|---|---|---|
| SheetJS | `styles[].id` via `s` | Undefined IDs warn; with `z`, only the number format is applied |
| table JSON | Named styles in `styles` | Undefined names, invalid colors, and invalid border names are errors |
| Data JSON | None | — |

table JSON style precedence is `bodyStyle` / `headerStyle` < `columns[].style` < cell `style`. `format` and `align` are applied on top.

Styles and number formats only affect **XLSX output**:

- `to-md` / `to-csv` display raw values.
- `to-html` applies alignment, fonts, fills, and borders, but not number formats.

### 6.5 Numbers, null, and types

| Format | Number decoding | Large integers |
|---|---|---|
| SheetJS | `encoding/json` → `float64` | Precision is lost above 2^53 |
| Data JSON | `json.Number` → `int64` / `float64` | Preserved at JSON parse time |
| table JSON | `json.Number` → `int64` / `float64` | Preserved at JSON parse time |

Excel itself stores roughly 15 significant digits. Even when the JSON parser preserves a large integer, Excel may round it for display or recalculation.

Null and nested objects:

- Data JSON `null` becomes an empty cell.
- Nested objects in data JSON are displayed as `map[...]` in previews and CSV.
- Putting an object into a table JSON cell is treated as a cell object and an unknown field causes an error.
- `to-csv --data-json` pads rows to the maximum column count. `null` becomes an empty field.

### 6.6 Validation

| Format | Validation strength |
|---|---|
| SheetJS | Schema is used on parse failure and by MCP validate. The CLI may ignore unknown fields |
| Data JSON | Structural detection only. Almost no semantic validation |
| table JSON | Rejects unknown fields; validates sheet names, key duplication, style references, and formula placeholders |

The schemas are relaxed to match runtime behavior:

- SheetJS accepts either `cells` or `rows`.
- `cell.t` is optional. A cell with `f` is treated as a formula.
- table JSON accepts `rows`, `header`, or `columns[].header`.

### 6.7 Round-trips

`to-json` always emits a **book wrapper with a cell map**:

- `rows` are expanded into `cells`.
- Formulas, merges, freeze panes, autofilter, and column widths are preserved.
- Sheet order is preserved through `book.sheetOrder`.
- Styles are written both as `book.styles` and as top-level `styles`.
- Empty sheets may be emitted as `{"name": "Sheet1"}` without `cells`.

To preserve sheet order, feed the `to-json` output directly back into `to-xlsx`. If you remove `book.sheetOrder`, sheets fall back to alphabetical order.

### 6.8 Input edge cases

- UTF-8 BOMs are stripped by every entry point.
- JSON shorter than 4 bytes can be read by `to-md`, `to-html`, and `info`.
- `[]` is an empty workbook as data JSON. `to-xlsx` and `to-csv --data-json` report that there is no data.
- Arrays passed to `to-csv` without a flag use the legacy array-of-objects path, not data JSON. Arrays containing booleans are rejected there.

## 7. Pitfalls

1. **`to-xlsx` / `to-md` / `to-html` / `to-csv` do not auto-detect JSON format**  
   Use `--table-json` for table JSON and `--data-json` for data JSON. Only `info` and MCP auto-detect.

2. **The wrong flag can silently reinterpret the input**  
   For example, passing table JSON with `--data-json` treats `tables` as a one-column array. This may not produce an error.

3. **`info` auto-detection is not perfect**  
   If a map-of-arrays has a column named `rows` or `styles`, pass `--data-json` / `--table-json` explicitly.

4. **Arrays without a flag in `to-csv` use the legacy path**  
   Add `--data-json` to use the data JSON pipeline.

5. **Preview commands do not apply number formats or styles**  
   Open the XLSX to verify formatting.

6. **Duplicate sheet names are now errors**  
   They are no longer silently overwritten.

7. **Sheet order comes from `book.sheetOrder`**  
   `book.sheets` object order is not guaranteed.

8. **Warning exit codes differ by command**  
   `to-xlsx` exits 1 when conversion warnings occur; `to-md` / `to-html` / `to-csv` exit 0.

9. **table JSON formulas with cached values are not recomputed**  
   `--compute` only evaluates cells that lack `v`.

## 8. Recommended workflow

1. **Ordinary tables**  
   Write table JSON, validate it with MCP `json2xlsx_validate`, preview it with `json2xlsx_preview`, then convert with `to-xlsx`.

2. **Raw data**  
   Pass CSV / SQL / API output as data JSON and narrow it with `--range` when needed. Move to table JSON or SheetJS format when styling becomes necessary.

3. **Advanced workbooks and round-trips**  
   Use SheetJS format. Export an existing XLSX with `to-json`, edit it while preserving `book.sheetOrder`, then convert it back.

4. **Checking a file**  
   Use `info` to inspect sheet count, cell count, and formula count. It auto-detects table JSON and data JSON as well.

## 9. Related links

- [README (English)](../README.md)
- [JSON Schema (SheetJS)](../schemas/json2xlsx.schema.json)
- [JSON Schema (table)](../schemas/table.schema.json)
- [sheetjs-json-writer](../skills/sheetjs-json-writer/SKILL.md)
- [table-json-writer](../skills/table-json-writer/SKILL.md)
- [Samples](../samples/)
