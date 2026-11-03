---
name: table-json-writer
description: Instruct the AI to output high-level table JSON that `json2xlsx --table-json` converts to XLSX.
---

# table-json-writer

Use this skill when the user asks for a simple table, report, or summary sheet. Output **table JSON only**. For charts, pictures, conditional formatting, sparklines, or other advanced workbook features, use the `sheetjs-json-writer` skill instead.

## Absolute rules
- Output the table object itself. In a plain chat, output JSON only and do not wrap it in a Markdown code fence. When using MCP, pass the object as the `table` argument of `json2xlsx_convert`.
- Use `tables` as the top-level key.
- Prefer object rows keyed by `columns[].key` for plain data.
- Use `columns[].formula` with `{key}` instead of repeating the same formula in every row.
- After generating JSON, call `json2xlsx_validate` and fix every issue. Then call `json2xlsx_preview` if available.

## Minimal shape
```json
{
  "tables": [
    {
      "name": "Sales",
      "columns": [
        { "key": "商品", "header": "商品", "width": 18 },
        { "key": "数量", "header": "数量", "format": "#,##0" },
        { "key": "単価", "header": "単価", "format": "#,##0" },
        { "key": "金額", "header": "金額", "format": "#,##0", "formula": "={数量}*{単価}" }
      ],
      "rows": [
        { "商品": "A", "数量": 10, "単価": 100 },
        { "商品": "B", "数量": 5,  "単価": 200 }
      ]
    }
  ]
}
```

## Field reference
| Field | Meaning |
|---|---|
| `tables` | Required. One element = one worksheet. |
| `name` | Worksheet name. Default `Sheet1`, `Sheet2`, ... |
| `header` | Optional header row at row 1. Data starts at row 2. `columns[].header` overrides it. |
| `rows` | Required array of rows. Each row is an array of cell values or an object keyed by `columns[].key`. |
| `columns` | Optional array of column settings. Index 0 = column A. |
| `columns[].key` | Logical column name for formulas. Required for object rows. Japanese is allowed. |
| `columns[].header` | Header text for this column. Overrides the top-level `header`. |
| `columns[].width` | Column width, `0 < width <= 255`. |
| `columns[].format` | Number format such as `#,##0` or `0.00%`. |
| `columns[].align` | `left`, `center`, or `right`. |
| `columns[].style` | Name in `styles`. |
| `columns[].formula` | Formula template for every data row. `{key}` is the current row cell, `{key:col}` is the column letter, and `{r}` is the row number. |
| `styles` | Named styles (see below). |
| `headerStyle` / `bodyStyle` | Style names for the header and data cells. |
| `freezeHeader` | Freeze the header row. Requires `header`. |
| `autoFilter` | Add autofilter to header + data. Requires `header`. |
| `merges` | Array of ranges such as `A1:D1`. |

## Cell values
- Numbers, strings, booleans, and `null` are written as-is.
- A string starting with `=` is a formula.
- In formulas, `{key}` becomes the current row cell (`B2`), `{key:col}` becomes the column letter (`B`), and `{r}` becomes the row number (`2`).
- Example: `={数量}*{単価}` becomes `=B2*C2`; `=SUM({金額:col}2:{金額:col}6)` becomes `=SUM(D2:D6)`.
- Unknown placeholders are validation errors. Use SheetJS JSON if you need literal braces in formulas.
- To write a literal string that starts with `=`, use `{ "v": "=text" }`.
- For a formula with a cached value, use `{ "v": 42, "f": "SUM(A1:A3)" }`.
- Cell object fields: `v`, `f`, `format`, `style`, `align`, `t`.
- Do not put objects or arrays in `v`.

## Styles
```json
{
  "styles": {
    "header": {
      "font": { "bold": true, "color": "#000000" },
      "fill": { "type": "pattern", "pattern": 1, "color": ["#E0EBF5"] },
      "border": [{ "style": "thin", "color": "#000000" }],
      "alignment": { "horizontal": "center" }
    },
    "body": {
      "border": [{ "style": "thin", "color": "#000000" }]
    }
  },
  "headerStyle": "header",
  "bodyStyle": "body"
}
```
- Colors use `#RRGGBB`.
- Border styles: `thin`, `medium`, `thick`, `dashed`, `dotted`, `double`, `hair`, and related variants.
- If `border[].side` is omitted, the border applies to all four sides.
- Style precedence: `bodyStyle`/`headerStyle` < `columns[].style` < cell `style`.
- `format` and `align` are applied on top of the selected style.

## Common mistakes
- Do not use `id` in table styles. They are assigned automatically.
- Do not add `=` to the `f` field of a cell object. In a plain string value, `=` marks a formula.
- Do not use a formula column without a header if you also set `freezeHeader` or `autoFilter`.
- Do not mix array rows and object rows in one table.
- Object rows require a `key` for every column. Keys must not contain whitespace, `{`, `}`, or `:`.
- Sheet names must be at most 31 characters and must not contain `[]:*?/\`.
