package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	assets "github.com/onohiroki/json2xlsx"
)

type staticResource struct {
	URI         string
	Name        string
	Description string
	File        string
	MIMEType    string
}

var schemaResources = []staticResource{
	{
		URI:         "json2xlsx://schema",
		Name:        "SheetJS workbook JSON schema",
		Description: "Full JSON Schema for the SheetJS-style workbook format.",
		File:        "schemas/json2xlsx.schema.json",
		MIMEType:    "application/schema+json",
	},
	{
		URI:         "json2xlsx://schema/table",
		Name:        "table JSON schema",
		Description: "JSON Schema for the high-level table format used by --table-json.",
		File:        "schemas/table.schema.json",
		MIMEType:    "application/schema+json",
	},
}

var skillResources = []staticResource{
	{
		URI:         "json2xlsx://skill/sheetjs",
		Name:        "sheetjs-json-writer skill",
		Description: "Instructions for producing SheetJS-style JSON.",
		File:        "skills/sheetjs-json-writer/SKILL.md",
		MIMEType:    "text/markdown",
	},
	{
		URI:         "json2xlsx://skill/sheetjs/ja",
		Name:        "sheetjs-json-writer skill (Japanese)",
		Description: "SheetJS 形式 JSON を生成するための日本語スキル．",
		File:        "skills/sheetjs-json-writer/SKILL.ja.md",
		MIMEType:    "text/markdown",
	},
	{
		URI:         "json2xlsx://skill/table",
		Name:        "table-json-writer skill",
		Description: "Instructions for producing high-level table JSON.",
		File:        "skills/table-json-writer/SKILL.md",
		MIMEType:    "text/markdown",
	},
	{
		URI:         "json2xlsx://skill/table/ja",
		Name:        "table-json-writer skill (Japanese)",
		Description: "高レベルな table JSON を生成するための日本語スキル．",
		File:        "skills/table-json-writer/SKILL.ja.md",
		MIMEType:    "text/markdown",
	},
}

var sampleResources = []staticResource{
	{URI: "json2xlsx://samples/sales", Name: "sample: sales", Description: "Basic sales table with formulas and styles.", File: "samples/sales.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/styles", Name: "sample: styles", Description: "Style definitions.", File: "samples/styles.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/table_sheetjs", Name: "sample: table (SheetJS)", Description: "Table with autofilter in SheetJS format.", File: "samples/table_sheetjs.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/table_map_of_array", Name: "sample: table (map of arrays)", Description: "Map-of-arrays data JSON.", File: "samples/table_map_of_array.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/table_array_of_map", Name: "sample: table (array of objects)", Description: "Array-of-objects data JSON.", File: "samples/table_array_of_map.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/compute_formulas", Name: "sample: formulas", Description: "Formula evaluation example.", File: "samples/compute_formulas.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/conditional_format", Name: "sample: conditional formatting", Description: "Conditional formatting example.", File: "samples/conditional_format.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/chart_bar", Name: "sample: bar chart", Description: "Bar chart example.", File: "samples/chart_bar.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/chart_scatter", Name: "sample: scatter chart", Description: "Scatter chart example.", File: "samples/chart_scatter.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/freeze", Name: "sample: freeze panes", Description: "Freeze panes example.", File: "samples/freeze.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/merge", Name: "sample: merged cells", Description: "Merged cells example.", File: "samples/merge.json", MIMEType: "application/json"},
	{URI: "json2xlsx://samples/sparklines", Name: "sample: sparklines", Description: "Sparkline example.", File: "samples/sparklines.json", MIMEType: "application/json"},
}

func (s *server) addResources(srv *mcp.Server) {
	for _, r := range schemaResources {
		s.addResource(srv, r)
	}
	for _, r := range skillResources {
		s.addResource(srv, r)
	}
	for _, r := range sampleResources {
		s.addResource(srv, r)
	}
}

func (s *server) addResource(srv *mcp.Server, r staticResource) {
	data, err := assets.FS.ReadFile(r.File)
	if err != nil {
		return
	}
	srv.AddResource(&mcp.Resource{
		URI:         r.URI,
		Name:        r.Name,
		Description: r.Description,
		MIMEType:    r.MIMEType,
		Size:        int64(len(data)),
	}, resourceHandler(r.File, r.MIMEType))
}

func resourceHandler(file, mime string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		data, err := assets.FS.ReadFile(file)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: mime,
				Text:     string(data),
			}},
		}, nil
	}
}
