package sheet2xlsx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ToJSONOptions は to-json の出力オプション。
type ToJSONOptions struct {
	// DateMode は日時セル (t=d) の出力モード。未指定時は DateModeSerial。
	DateMode DateMode
	// WrapWithBook は book ラッパー形式 (version + book) で出力する。
	// true の場合、chartsheet のグラフ情報も charts 配列に含める。
	WrapWithBook bool
}

// ToJSON は XLSX を読み込み、sheet2xlsx 互換 JSON (セルマップ形式) を out に書き出す。
func ToJSON(r io.Reader, out io.Writer) error {
	return ToJSONWithOptions(r, out, ToJSONOptions{DateMode: DateModeSerial})
}

// ToJSONWithOptions はオプション付きで XLSX を sheet2xlsx 互換 JSON に変換する。
func ToJSONWithOptions(r io.Reader, out io.Writer, opts ToJSONOptions) error {
	if opts.DateMode == "" {
		opts.DateMode = DateModeSerial
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = f.Close() }()

	wb, err := extractWorkbookWithOptions(f, opts)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&wb); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// sheetJSON は単一シート出力用の補助構造体 (ref を含む)。
// Sheet 型に ref フィールドを追加してもよいが、JSON 出力を簡潔に保つために
// Cells map のキー集合から導出する方針も可能。ここでは Sheet をそのまま使う。

// extractWorkbook は excelize で開いた XLSX から Workbook 構造を抽出する。
// ToJSON と ToMarkdown (XLSX 経路) の両方で再利用される。
func extractWorkbook(f *excelize.File) (Workbook, error) {
	return extractWorkbookWithOptions(f, ToJSONOptions{DateMode: DateModeSerial})
}

func extractWorkbookWithOptions(f *excelize.File, opts ToJSONOptions) (Workbook, error) {
	sc := newStyleCollector()

	// chartsheet 名を事前検出
	chartSheetNames, _ := listChartsheetNames(f)

	sheetNames := f.GetSheetList()
	sheets := make([]Sheet, 0, len(sheetNames))
	for _, name := range sheetNames {
		if chartSheetNames[name] {
			continue // chartsheet はスキップ（セルデータなし）
		}
		sh, err := extractSheet(f, name, sc, opts)
		if err != nil {
			return Workbook{}, fmt.Errorf("extract sheet %q: %w", name, err)
		}
		sheets = append(sheets, sh)
	}

	var wb Workbook
	wb.Styles = sc.styles

	if opts.WrapWithBook {
		// book ラッパー形式で出力
		book := &Book{
			Sheets: make(map[string]Sheet, len(sheets)),
			Styles: sc.styles,
		}
		for _, sh := range sheets {
			book.Sheets[sh.Name] = sh
		}
		// chartsheet からグラフを抽出
		charts, err := extractChartsheets(f)
		if err != nil {
			return Workbook{}, fmt.Errorf("extract chartsheets: %w", err)
		}

		// 埋め込みグラフを抽出
		embeddedCharts, err := extractEmbeddedCharts(f)
		if err != nil {
			return Workbook{}, fmt.Errorf("extract embedded charts: %w", err)
		}
		charts = append(charts, embeddedCharts...)

		if len(charts) > 0 {
			book.Charts = charts
		}
		wb.Version = "0.2"
		wb.Book = book
		return wb, nil
	}

	if len(sheets) == 1 {
		sh := sheets[0]
		wb.Name = sh.Name
		wb.Cells = sh.Cells
		wb.Cols = sh.Cols
		wb.RowDims = sh.RowDims
		wb.Merges = sh.Merges
	} else {
		wb.Sheets = sheets
	}
	return wb, nil
}

func extractSheet(f *excelize.File, name string, sc *styleCollector, opts ToJSONOptions) (Sheet, error) {
	sh := Sheet{Name: name, Cells: map[string]Cell{}}

	// GetCols でシート全体の最大列数を取得する（グラフシートなどでは失敗するためエラーは無視）。
	// Rows().Columns() は値のあるセルまでしか返さないため、スタイルのみの空セルを見逃さないために必要。
	var maxCol int
	if allCols, err := f.GetCols(name); err == nil {
		maxCol = len(allCols)
	}

	rows, err := f.Rows(name)
	if err != nil {
		return sh, err
	}
	defer func() { _ = rows.Close() }()

	rowIdx := 0
	for rows.Next() {
		rowIdx++
		cols, err := rows.Columns()
		if err != nil {
			return sh, err
		}
		for colIdx := 1; colIdx <= len(cols); colIdx++ {
			axis, err := excelize.CoordinatesToCellName(colIdx, rowIdx)
			if err != nil {
				return sh, err
			}
			cell, ok, err := extractCell(f, name, axis, sc, opts)
			if err != nil {
				return sh, err
			}
			if ok {
				sh.Cells[axis] = cell
			}
		}
		// Rows().Columns() で取得できなかったスタイルのみの空セルを補完する。
		for colIdx := len(cols) + 1; colIdx <= maxCol; colIdx++ {
			axis, err := excelize.CoordinatesToCellName(colIdx, rowIdx)
			if err != nil {
				return sh, err
			}
			cell, ok, err := extractCell(f, name, axis, sc, opts)
			if err != nil {
				return sh, err
			}
			if ok {
				sh.Cells[axis] = cell
			}
		}
	}

	// merges
	if mcs, err := f.GetMergeCells(name); err == nil {
		for _, m := range mcs {
			sh.Merges = append(sh.Merges, Merge{Range: m.GetStartAxis() + ":" + m.GetEndAxis()})
		}
	}

	// cols (デフォルト幅と異なるものだけ抽出)
	// 出現したセルから列集合を導出
	colSet := map[int]struct{}{}
	for axis := range sh.Cells {
		c, _, err := excelize.CellNameToCoordinates(axis)
		if err == nil {
			colSet[c] = struct{}{}
		}
	}
	const defaultColWidth = 9.140625
	for c := range colSet {
		colName, err := excelize.ColumnNumberToName(c)
		if err != nil {
			continue
		}
		w, err := f.GetColWidth(name, colName)
		if err != nil {
			continue
		}
		if w > 0 && w != defaultColWidth {
			sh.Cols = append(sh.Cols, ColInfo{Col: colName, Width: w})
		}
	}

	// rowDims
	rowSet := map[int]struct{}{}
	for axis := range sh.Cells {
		_, rIdx, err := excelize.CellNameToCoordinates(axis)
		if err == nil {
			rowSet[rIdx] = struct{}{}
		}
	}
	for r := range rowSet {
		h, err := f.GetRowHeight(name, r)
		if err != nil {
			continue
		}
		// excelize はデフォルト行高を返すため、明示設定されたかの判定は厳密にはできない。
		// ここではデフォルト 15.0 と一致する場合スキップする簡易判定。
		if h > 0 && h != 15.0 {
			sh.RowDims = append(sh.RowDims, RowInfo{Row: r, Height: h})
		}
	}

	return sh, nil
}

func extractCell(f *excelize.File, sheet, axis string, sc *styleCollector, opts ToJSONOptions) (Cell, bool, error) {
	formula, err := f.GetCellFormula(sheet, axis)
	if err != nil {
		return Cell{}, false, err
	}
	ct, err := f.GetCellType(sheet, axis)
	if err != nil {
		return Cell{}, false, err
	}
	rawVal, err := f.GetCellValue(sheet, axis, excelize.Options{RawCellValue: true})
	if err != nil {
		return Cell{}, false, err
	}

	// 空セル判定: 数式も値もスタイルもなし → スキップ
	excelizeStyleID, _ := f.GetCellStyle(sheet, axis)
	if formula == "" && rawVal == "" && excelizeStyleID == 0 {
		return Cell{}, false, nil
	}

	cell := Cell{}

	// スタイル
	var isDateFmt bool
	if excelizeStyleID != 0 {
		jsonID, dateFmt, err := sc.collect(f, excelizeStyleID)
		if err != nil {
			return Cell{}, false, err
		}
		cell.S = jsonID
		isDateFmt = dateFmt
	}

	// 型と値
	switch {
	case formula != "":
		cell.T = "f"
		cell.F = formula
		if rawVal != "" {
			cell.V = parseScalar(rawVal)
		}
	case ct == excelize.CellTypeBool:
		cell.T = "b"
		cell.V = rawVal == "1" || strings.EqualFold(rawVal, "true")
	case ct == excelize.CellTypeNumber, ct == excelize.CellTypeUnset:
		if rawVal == "" {
			// スタイルだけのセル
			if cell.S == 0 {
				return Cell{}, false, nil
			}
		} else if isDateFmt {
			cell.T = "d"
			cell.V = resolveDateCellValue(f, sheet, axis, rawVal, opts)
		} else {
			cell.T = "n"
			cell.V = parseScalar(rawVal)
		}
	case ct == excelize.CellTypeDate:
		cell.T = "d"
		cell.V = resolveDateCellValue(f, sheet, axis, rawVal, opts)
	default: // 文字列系
		cell.T = "s"
		cell.V = normalizeNewlines(rawVal)
	}

	return cell, true, nil
}

func resolveDateCellValue(f *excelize.File, sheet, axis, rawVal string, opts ToJSONOptions) interface{} {
	switch opts.DateMode {
	case DateModeDisplay:
		if displayed, err := f.GetCellValue(sheet, axis); err == nil && displayed != "" {
			return normalizeNewlines(displayed)
		}
		return rawVal
	case DateModeRFC3339:
		if serial, err := strconv.ParseFloat(rawVal, 64); err == nil {
			if t, err := excelize.ExcelDateToTime(serial, false); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
		return rawVal
	case DateModeSerial:
		if serial, err := strconv.ParseFloat(rawVal, 64); err == nil {
			return serial
		}
		return rawVal
	default:
		return rawVal
	}
}

// parseScalar は数値文字列なら数値、真偽値文字列なら bool、それ以外は文字列として返す。
func parseScalar(s string) interface{} {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if fv, err := strconv.ParseFloat(s, 64); err == nil {
		return fv
	}
	return s
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
