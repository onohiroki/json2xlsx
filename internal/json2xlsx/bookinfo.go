package json2xlsx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// InfoSchema は info --json のスキーマ識別子．
const InfoSchema = "json2xlsx-info/1"

// SheetInfo は 1 シートのサマリ．
type SheetInfo struct {
	Name               string      `json:"name"`
	UsedRange          string      `json:"usedRange"`
	Rows               int         `json:"rows"`
	Cols               int         `json:"cols"`
	Cells              int         `json:"cells"`
	Formulas           int         `json:"formulas"`
	Merges             int         `json:"merges"`
	Tables             []Table     `json:"tables,omitempty"`
	Charts             int         `json:"charts"`
	ConditionalFormats int         `json:"conditionalFormats"`
	Pictures           int         `json:"pictures"`
	Sparklines         int         `json:"sparklines"`
	Freeze             *FreezePane `json:"freeze,omitempty"`
}

// BookTotals は全シート合計．
type BookTotals struct {
	Sheets   int `json:"sheets"`
	Cells    int `json:"cells"`
	Formulas int `json:"formulas"`
	Merges   int `json:"merges"`
	Tables   int `json:"tables"`
	Charts   int `json:"charts"`
	Pictures int `json:"pictures"`
}

// BookInfo は info の結果全体．
type BookInfo struct {
	Schema string      `json:"schema"`
	Sheets []SheetInfo `json:"sheets"`
	Totals BookTotals  `json:"totals"`
}

// BuildBookInfo は正規化前の Workbook からサマリを作る．
// グラフ情報は wb.Book があればそこから取得する．
func BuildBookInfo(wb *Workbook) BookInfo {
	return BuildBookInfoWithCharts(wb, nil)
}

// BuildBookInfoWithCharts はグラフ一覧を明示的に受け取ってサマリを作る．
// charts が nil かつ wb.Book がある場合は wb.Book.Charts を使う．
// wb は破壊的に正規化される (Sheets が確定する)．
func BuildBookInfoWithCharts(wb *Workbook, charts []Chart) BookInfo {
	if charts == nil && wb.Book != nil {
		charts = wb.Book.Charts
	}
	normalizeWorkbook(wb)

	info := BookInfo{Schema: InfoSchema, Sheets: []SheetInfo{}}
	for i := range wb.Sheets {
		sh := &wb.Sheets[i]
		si := SheetInfo{Name: sh.Name}

		minC, minR, maxC, maxR, ok := sheetUsedBounds(sh)
		if ok {
			si.UsedRange = formatUsedRange(minC, minR, maxC, maxR)
			si.Rows = maxR - minR + 1
			si.Cols = maxC - minC + 1
		}
		si.Cells = sheetCellCount(sh)
		si.Formulas = sheetFormulaCount(sh)
		si.Merges = len(sh.Merges)
		si.Tables = sh.Tables
		si.ConditionalFormats = len(sh.ConditionalFormats)
		si.Pictures = len(sh.Pictures)
		si.Sparklines = len(sh.Sparklines)
		si.Freeze = sh.Freeze

		for _, c := range charts {
			if c.Sheet == sh.Name {
				si.Charts++
			}
		}

		info.Sheets = append(info.Sheets, si)
	}
	info.Totals = TotalsForSheets(info.Sheets)
	return info
}

// TotalsForSheets はシート一覧から合計を計算する．
func TotalsForSheets(sheets []SheetInfo) BookTotals {
	var t BookTotals
	for _, s := range sheets {
		t.Sheets++
		t.Cells += s.Cells
		t.Formulas += s.Formulas
		t.Merges += s.Merges
		t.Tables += len(s.Tables)
		t.Charts += s.Charts
		t.Pictures += s.Pictures
	}
	return t
}

// sheetUsedBounds はシートの使用範囲 (最小・最大の行/列) を返す．
// 座標は 1 始まり．データがなければ ok=false．
func sheetUsedBounds(sh *Sheet) (minC, minR, maxC, maxR int, ok bool) {
	if len(sh.Cells) > 0 {
		first := true
		for axis := range sh.Cells {
			c, r, err := excelize.CellNameToCoordinates(axis)
			if err != nil {
				continue
			}
			if first {
				minC, minR, maxC, maxR = c, r, c, r
				first = false
				continue
			}
			if c < minC {
				minC = c
			}
			if c > maxC {
				maxC = c
			}
			if r < minR {
				minR = r
			}
			if r > maxR {
				maxR = r
			}
		}
		return minC, minR, maxC, maxR, !first
	}

	if len(sh.Rows) > 0 {
		maxCols := 0
		for _, row := range sh.Rows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}
		if maxCols == 0 {
			return 0, 0, 0, 0, false
		}
		return 1, 1, maxCols, len(sh.Rows), true
	}
	return 0, 0, 0, 0, false
}

func formatUsedRange(minC, minR, maxC, maxR int) string {
	start, err1 := excelize.CoordinatesToCellName(minC, minR)
	end, err2 := excelize.CoordinatesToCellName(maxC, maxR)
	if err1 != nil || err2 != nil {
		return ""
	}
	if start == end {
		return start
	}
	return start + ":" + end
}

func sheetCellCount(sh *Sheet) int {
	if len(sh.Cells) > 0 {
		return len(sh.Cells)
	}
	n := 0
	for _, row := range sh.Rows {
		n += len(row)
	}
	return n
}

func sheetFormulaCount(sh *Sheet) int {
	n := 0
	if len(sh.Cells) > 0 {
		for _, cell := range sh.Cells {
			if cell.T == "f" || cell.F != "" {
				n++
			}
		}
	}
	return n
}

// ReadBookInfoSource は入力 (XLSX または Workbook JSON) を読み込み，
// サマリ計算に必要な Workbook とグラフ一覧を返す．
// XLSX の場合はシート順を保持するため book ラッパーを使わずに抽出する．
func ReadBookInfoSource(r io.Reader) (*Workbook, []Chart, error) {
	br := bufio.NewReader(r)
	head, err := br.Peek(4)
	if err != nil {
		return nil, nil, fmt.Errorf("read input: %w", err)
	}

	if bytes.Equal(head, []byte{'P', 'K', 0x03, 0x04}) {
		data, err := io.ReadAll(br)
		if err != nil {
			return nil, nil, fmt.Errorf("read input: %w", err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			return nil, nil, fmt.Errorf("open xlsx: %w", err)
		}
		defer func() { _ = f.Close() }()

		wb, err := extractWorkbookWithOptions(f, ToJSONOptions{DateMode: DateModeDisplay})
		if err != nil {
			return nil, nil, err
		}
		var charts []Chart
		if cs, err := extractChartsheets(f); err == nil {
			charts = append(charts, cs...)
		}
		if cs, err := extractEmbeddedCharts(f); err == nil {
			charts = append(charts, cs...)
		}
		normalizeWorkbook(&wb)
		return &wb, charts, nil
	}

	data, err := io.ReadAll(br)
	if err != nil {
		return nil, nil, fmt.Errorf("read input: %w", err)
	}
	data = trimBOM(data)
	wb, err := UnmarshalWorkbook(data, false)
	if err != nil {
		return nil, nil, err
	}
	normalizeDateCells(wb)
	normalizeWorkbook(wb)
	return wb, nil, nil
}
