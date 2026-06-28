package json2xlsx

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// extractChartsheets は XLSX 内の chartsheet からグラフ情報を抽出する。
func extractChartsheets(f *excelize.File) ([]Chart, error) {
	rels, err := loadChartRels(f, "xl/_rels/workbook.xml.rels")
	if err != nil {
		return nil, fmt.Errorf("load workbook rels: %w", err)
	}
	if rels == nil {
		return nil, nil
	}

	sheetEntries, err := loadSheetEntries(f)
	if err != nil {
		return nil, fmt.Errorf("load sheet entries: %w", err)
	}
	if len(sheetEntries) == 0 {
		return nil, nil
	}

	sheetRels := make(map[string]*chartRel)
	for i := range rels.Relationships {
		sheetRels[rels.Relationships[i].ID] = &rels.Relationships[i]
	}

	const chartsheetRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet"

	var charts []Chart
	for _, sh := range sheetEntries {
		rel, ok := sheetRels[sh.ID]
		if !ok {
			continue
		}
		if rel.Type != chartsheetRelType {
			continue
		}

		csPath := resolveRelPath("xl/workbook.xml", rel.Target)
		ch, ok, err := extractChartFromChartsheet(f, csPath, sh.Name)
		if err != nil {
			return nil, fmt.Errorf("extract chartsheet %q: %w", sh.Name, err)
		}
		if ok {
			charts = append(charts, *ch)
		}
	}
	return charts, nil
}

// loadSheetEntries は xl/workbook.xml からシートエントリ一覧を読み込む。
func loadSheetEntries(f *excelize.File) ([]wbSheetEntryXML, error) {
	raw, ok := f.Pkg.Load("xl/workbook.xml")
	if !ok {
		return nil, nil
	}
	var wb struct {
		Sheets wbSheetXML `xml:"sheets"`
	}
	if err := xml.Unmarshal(raw.([]byte), &wb); err != nil {
		return nil, err
	}
	return wb.Sheets.Sheet, nil
}

// extractChartFromChartsheet は chartsheet XML から 1 つのグラフを抽出する。
func extractChartFromChartsheet(f *excelize.File, csPath, sheetName string) (*Chart, bool, error) {
	rawCS, ok := f.Pkg.Load(csPath)
	if !ok {
		return nil, false, nil
	}

	var cs chartSheetXML
	if err := xml.Unmarshal(rawCS.([]byte), &cs); err != nil {
		return nil, false, fmt.Errorf("unmarshal chartsheet: %w", err)
	}
	if cs.Drawing == nil || cs.Drawing.ID == "" {
		return nil, false, nil
	}

	csRelsPath := chartsheetRelsPath(csPath)
	csRels, err := loadChartRels(f, csRelsPath)
	if err != nil {
		return nil, false, fmt.Errorf("load chartsheet rels: %w", err)
	}
	if csRels == nil {
		return nil, false, nil
	}

	const drawingRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing"
	var drawingTarget string
	for _, r := range csRels.Relationships {
		if r.ID == cs.Drawing.ID && r.Type == drawingRelType {
			drawingTarget = r.Target
			break
		}
	}
	if drawingTarget == "" {
		return nil, false, nil
	}

	drawingPath := resolveRelPath(csPath, drawingTarget)

	drawingDir := filepath.Dir(drawingPath)
	drawingRelsPath := drawingDir + "/_rels/" + filepath.Base(drawingPath) + ".rels"
	drawingRels, err := loadChartRels(f, drawingRelsPath)
	if err != nil {
		return nil, false, fmt.Errorf("load drawing rels: %w", err)
	}
	if drawingRels == nil {
		return nil, false, nil
	}

	const chartRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	var chartPath string
	for _, r := range drawingRels.Relationships {
		if r.Type == chartRelType {
			chartPath = resolveRelPath(drawingPath, r.Target)
			break
		}
	}
	if chartPath == "" {
		return nil, false, nil
	}

	return parseChartXML(f, chartPath, sheetName)
}

// chartsheetRelsPath は chartsheet XML に対応する rels ファイルのパスを返す。
func chartsheetRelsPath(csPath string) string {
	csDir := filepath.Dir(csPath)
	return csDir + "/_rels/" + filepath.Base(csPath) + ".rels"
}

// loadChartRels は指定されたパスから XML の relationships を読み込む。
func loadChartRels(f *excelize.File, path string) (*chartRels, error) {
	raw, ok := f.Pkg.Load(path)
	if !ok {
		return nil, nil
	}
	var rels chartRels
	dec := xml.NewDecoder(bytes.NewReader(raw.([]byte)))
	if err := dec.Decode(&rels); err != nil && err != io.EOF {
		return nil, err
	}
	return &rels, nil
}

// resolveRelPath はベースパスからの相対パスを絶対パスに解決する。
func resolveRelPath(basePath, relTarget string) string {
	if strings.HasPrefix(relTarget, "/") {
		return strings.TrimPrefix(relTarget, "/")
	}
	baseDir := filepath.Dir(basePath)
	cleaned := filepath.Clean(baseDir + "/" + relTarget)
	return cleaned
}

// extractEmbeddedCharts は全 worksheet から埋め込みグラフを抽出する。
func extractEmbeddedCharts(f *excelize.File) ([]Chart, error) {
	rels, err := loadChartRels(f, "xl/_rels/workbook.xml.rels")
	if err != nil || rels == nil {
		return nil, err
	}

	sheetEntries, err := loadSheetEntries(f)
	if err != nil || len(sheetEntries) == 0 {
		return nil, err
	}

	sheetRels := make(map[string]*chartRel)
	for i := range rels.Relationships {
		sheetRels[rels.Relationships[i].ID] = &rels.Relationships[i]
	}

	const worksheetRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet"

	var charts []Chart
	for _, sh := range sheetEntries {
		rel, ok := sheetRels[sh.ID]
		if !ok || rel.Type != worksheetRelType {
			continue
		}

		wsPath := resolveRelPath("xl/workbook.xml", rel.Target)
		wsCharts, err := extractChartsFromWorksheet(f, wsPath, sh.Name)
		if err != nil {
			return nil, fmt.Errorf("extract worksheet %q: %w", sh.Name, err)
		}
		charts = append(charts, wsCharts...)
	}

	return charts, nil
}

// extractChartsFromWorksheet は 1 つの worksheet XML から drawing を辿りグラフを抽出する。
func extractChartsFromWorksheet(f *excelize.File, wsPath, sheetName string) ([]Chart, error) {
	rawWS, ok := f.Pkg.Load(wsPath)
	if !ok {
		return nil, nil
	}

	var ws struct {
		Drawing *struct {
			ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"drawing"`
	}
	if err := xml.Unmarshal(rawWS.([]byte), &ws); err != nil {
		return nil, fmt.Errorf("unmarshal worksheet: %w", err)
	}
	if ws.Drawing == nil || ws.Drawing.ID == "" {
		return nil, nil
	}

	wsRelsPath := filepath.Dir(wsPath) + "/_rels/" + filepath.Base(wsPath) + ".rels"
	wsRels, err := loadChartRels(f, wsRelsPath)
	if err != nil {
		return nil, fmt.Errorf("load worksheet rels: %w", err)
	}
	if wsRels == nil {
		return nil, nil
	}

	const drawingRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing"
	var drawingTarget string
	for _, r := range wsRels.Relationships {
		if r.ID == ws.Drawing.ID && r.Type == drawingRelType {
			drawingTarget = r.Target
			break
		}
	}
	if drawingTarget == "" {
		return nil, nil
	}

	drawingPath := resolveRelPath(wsPath, drawingTarget)
	return extractChartsFromDrawing(f, drawingPath, sheetName)
}

// extractChartsFromDrawing は drawing XML をパースし全ての埋め込みグラフを抽出する。
func extractChartsFromDrawing(f *excelize.File, drawingPath, sheetName string) ([]Chart, error) {
	drawDir := filepath.Dir(drawingPath)
	drawRelsPath := drawDir + "/_rels/" + filepath.Base(drawingPath) + ".rels"
	drawRels, err := loadChartRels(f, drawRelsPath)
	if err != nil {
		return nil, fmt.Errorf("load drawing rels: %w", err)
	}
	if drawRels == nil {
		return nil, nil
	}

	const chartRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart"
	chartRelMap := make(map[string]string)
	for _, r := range drawRels.Relationships {
		if r.Type == chartRelType {
			chartRelMap[r.ID] = resolveRelPath(drawingPath, r.Target)
		}
	}
	if len(chartRelMap) == 0 {
		return nil, nil
	}

	rawDraw, ok := f.Pkg.Load(drawingPath)
	if !ok {
		return nil, nil
	}

	var dr wsDrXML
	if err := xml.Unmarshal(rawDraw.([]byte), &dr); err != nil {
		var dr2 struct {
			XMLName         xml.Name           `xml:"http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing wsDr"`
			TwoCellAnchors  []xdrTwoCellAnchor `xml:"twoCellAnchor"`
			OneCellAnchors  []xdrOneCellAnchor `xml:"oneCellAnchor"`
			AbsoluteAnchors []xdrAbsAnchor     `xml:"absoluteAnchor"`
		}
		if err2 := xml.Unmarshal(rawDraw.([]byte), &dr2); err2 != nil {
			return nil, fmt.Errorf("unmarshal drawing: %w", err2)
		}
		dr.TwoCellAnchors = dr2.TwoCellAnchors
		dr.OneCellAnchors = dr2.OneCellAnchors
		dr.AbsoluteAnchors = dr2.AbsoluteAnchors
	}

	type anchorChart struct {
		anchorCell string
		dim        ChartDim
		chartRID   string
	}

	var items []anchorChart

	for i := range dr.TwoCellAnchors {
		a := &dr.TwoCellAnchors[i]
		rid, ok := chartRefInGraphicFrame(a.GraphicFrame)
		if !ok {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(a.From.Col+1, a.From.Row+1)
		if err != nil {
			cell = "A1"
		}
		const pxPerEMU = 9525.0
		colSpan := a.To.Col - a.From.Col
		rowSpan := a.To.Row - a.From.Row
		const pxPerCol = 64.0
		const pxPerRow = 20.0
		w := float64(a.To.ColOff)/pxPerEMU + float64(colSpan)*pxPerCol
		h := float64(a.To.RowOff)/pxPerEMU + float64(rowSpan)*pxPerRow
		items = append(items, anchorChart{
			anchorCell: cell,
			dim: ChartDim{
				OffX: float64(a.From.ColOff),
				OffY: float64(a.From.RowOff),
				W:    w,
				H:    h,
			},
			chartRID: rid,
		})
	}

	for i := range dr.OneCellAnchors {
		a := &dr.OneCellAnchors[i]
		rid, ok := chartRefInGraphicFrame(a.GraphicFrame)
		if !ok {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(a.From.Col+1, a.From.Row+1)
		if err != nil {
			cell = "A1"
		}
		const pxPerEMU = 9525.0
		items = append(items, anchorChart{
			anchorCell: cell,
			dim: ChartDim{
				OffX: float64(a.From.ColOff),
				OffY: float64(a.From.RowOff),
				W:    float64(a.Ext.Cx) / pxPerEMU,
				H:    float64(a.Ext.Cy) / pxPerEMU,
			},
			chartRID: rid,
		})
	}

	for i := range dr.AbsoluteAnchors {
		a := &dr.AbsoluteAnchors[i]
		rid, ok := chartRefInGraphicFrame(a.GraphicFrame)
		if !ok {
			continue
		}
		const defaultColEMU = 609600.0
		const defaultRowEMU = 190500.0
		col := int(float64(a.Pos.X) / defaultColEMU)
		row := int(float64(a.Pos.Y) / defaultRowEMU)
		cell, err := excelize.CoordinatesToCellName(col+1, row+1)
		if err != nil {
			cell = "A1"
		}
		const pxPerEMU = 9525.0
		items = append(items, anchorChart{
			anchorCell: cell,
			dim: ChartDim{
				OffX: float64(a.Pos.X) - float64(col)*defaultColEMU,
				OffY: float64(a.Pos.Y) - float64(row)*defaultRowEMU,
				W:    float64(a.Ext.Cx) / pxPerEMU,
				H:    float64(a.Ext.Cy) / pxPerEMU,
			},
			chartRID: rid,
		})
	}

	var charts []Chart
	for _, it := range items {
		chartPath, ok := chartRelMap[it.chartRID]
		if !ok {
			continue
		}
		ch, ok, err := parseChartXML(f, chartPath, sheetName)
		if err != nil {
			return nil, fmt.Errorf("parse chart at %q: %w", chartPath, err)
		}
		if ok {
			ch.Mode = "embedded"
			ch.Anchor = it.anchorCell
			ch.Dim = &it.dim
			charts = append(charts, *ch)
		}
	}

	return charts, nil
}

// chartRefInGraphicFrame は GraphicFrame からチャート参照 ID を取得する。
func chartRefInGraphicFrame(gf *xdrGraphicFrame) (string, bool) {
	if gf == nil || gf.Graphic == nil || gf.Graphic.GraphicData == nil {
		return "", false
	}
	gd := gf.Graphic.GraphicData
	if gd.URI != "http://schemas.openxmlformats.org/drawingml/2006/chart" || gd.Chart == nil {
		return "", false
	}
	return gd.Chart.ID, true
}

// listChartsheetNames は chartsheet のシート名一覧を返す。
func listChartsheetNames(f *excelize.File) (map[string]bool, error) {
	rels, err := loadChartRels(f, "xl/_rels/workbook.xml.rels")
	if err != nil || rels == nil {
		return nil, err
	}

	sheetEntries, err := loadSheetEntries(f)
	if err != nil || len(sheetEntries) == 0 {
		return nil, err
	}

	sheetRels := make(map[string]*chartRel)
	for i := range rels.Relationships {
		sheetRels[rels.Relationships[i].ID] = &rels.Relationships[i]
	}

	const chartsheetRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/chartsheet"
	names := make(map[string]bool)
	for _, sh := range sheetEntries {
		if rel, ok := sheetRels[sh.ID]; ok && rel.Type == chartsheetRelType {
			names[sh.Name] = true
		}
	}
	return names, nil
}
