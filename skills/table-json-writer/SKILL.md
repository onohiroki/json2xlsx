---
name: table-json-writer
description: Instruct the AI to output high-level table JSON that `json2xlsx --table-json` converts to XLSX.
---

# table-json-writer

Use this skill when the user asks for a simple table, report, or summary sheet. Output **table JSON only**. For charts, pictures, conditional formatting, sparklines, or other advanced workbook features, use the `sheetjs-json-writer` skill instead.

## Absolute rules
- Output the table object itself. In a plain chat, output JSON only and do not wrap it in a Markdown code fence. When using MCP, pass the object as the `table` argument of `json2xlsx_convert`.
- Use `tables` as the top-level key.
- Prefer `header` + `rows` for plain data.
- Use `columns[].formula` with `{r}` instead of repeating the same formula in every row.
- After generating JSON, call `json2xlsx_validate` and fix every issue. Then call `json2xlsx_preview` if available.

## Minimal shape
```json
{
  "tables": [
    {
      "name": "Sales",
      "header": ["商品", "数量", "単価", "金額"],
      "rows": [
        ["A", 10, 100],
        ["B", 5, 200]
      ],
      "columns": [
        { "width": 18 },
        { "format": "#,##0" },
        { "format": "#,##0" },
        { "format": "#,##0", "formula": "=B{r}*C{r}" }
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
| `header` | Optional header row at row 1. Data starts at row 2. |
| `rows` | Required array of rows. Each row is an array of cell values. |
| `columns` | Optional array of column settings. Index 0 = column A. |
| `columns[].width` | Column width, `0 < width <= 255`. |
| `columns[].format` | Number format such as `#,##0` or `0.00%`. |
| `columns[].align` | `left`, `center`, or `right`. |
| `columns[].style` | Name in `styles`. |
| `columns[].formula` | Formula template for every data row. `{r}` becomes the row number. |
| `styles` | Named styles (see below). |
| `headerStyle` / `bodyStyle` | Style names for the header and data cells. |
| `freezeHeader` | Freeze the header row. Requires `header`. |
| `autoFilter` | Add autofilter to header + data. Requires `header`. |
| `merges` | Array of ranges such as `A1:D1`. |

## Cell values
- Numbers, strings, booleans, and `null` are written as-is.
- A string starting with `=` is a formula.
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
- Sheet names must be at most 31 characters and must not contain `[]:*?/\`.
