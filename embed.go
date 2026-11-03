// Package assets embeds schemas, skills, and sample JSON so that the MCP
// server can expose them as resources without extra files at runtime.
package assets

import "embed"

// FS contains the JSON schemas, skill documents, and selected sample inputs.
//
//go:embed schemas/json2xlsx.schema.json schemas/table.schema.json
//go:embed skills/sheetjs-json-writer/SKILL.md skills/sheetjs-json-writer/SKILL.ja.md
//go:embed skills/table-json-writer/SKILL.md skills/table-json-writer/SKILL.ja.md
//go:embed samples/sales.json samples/styles.json samples/table_sheetjs.json
//go:embed samples/table_map_of_array.json samples/table_array_of_map.json
//go:embed samples/compute_formulas.json samples/conditional_format.json
//go:embed samples/chart_bar.json samples/chart_scatter.json
//go:embed samples/freeze.json samples/merge.json samples/sparklines.json
var FS embed.FS
