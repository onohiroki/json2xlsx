package sheet2xlsx

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Convert は JSON を読み込み、XLSX を out に書き出す。
// defaultSheetName が空でない場合、シート名未指定時のデフォルトとして使う。
func Convert(r io.Reader, out io.Writer, defaultSheetName string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var wb Workbook
	if err := json.Unmarshal(data, &wb); err != nil {
		if schemaErr := ValidateJSON(data); schemaErr != nil {
			return fmt.Errorf("%v\n\n%v", err, schemaErr)
		}
		return fmt.Errorf("parse json: %w", err)
	}

	if err := convertWorkbook(data, &wb, out, defaultSheetName); err != nil {
		if schemaErr := ValidateJSON(data); schemaErr != nil {
			return fmt.Errorf("%v\n\n%v", err, schemaErr)
		}
		return err
	}
	return nil
}

func convertWorkbook(data []byte, wb *Workbook, out io.Writer, defaultSheetName string) error {
	f := excelize.NewFile()
	defer f.Close()

	// シート一覧を組み立てる
	sheets := wb.Sheets
	if len(sheets) == 0 {
		// 単一シート形式
		sheets = []Sheet{{
			Name:    wb.Name,
			Cells:   wb.Cells,
			Rows:    wb.Rows,
			Cols:    wb.Cols,
			RowDims: wb.RowDims,
			Merges:  wb.Merges,
		}}
	}

	// スタイル ID -> excelize スタイル ID マッピング
	styleMap, err := buildStyles(f, wb.Styles)
	if err != nil {
		return fmt.Errorf("build styles: %w", err)
	}

	// 既定シート名 ("Sheet1") を最初のシートに割り当て
	defaultName := f.GetSheetName(0)
	firstAssigned := false

	for i, sh := range sheets {
		name := sh.Name
		if name == "" {
			if i == 0 && defaultSheetName != "" {
				name = defaultSheetName
			} else {
				name = fmt.Sprintf("Sheet%d", i+1)
			}
		}

		if !firstAssigned {
			if name != defaultName {
				if err := f.SetSheetName(defaultName, name); err != nil {
					return fmt.Errorf("rename sheet: %w", err)
				}
			}
			firstAssigned = true
		} else {
			if _, err := f.NewSheet(name); err != nil {
				return fmt.Errorf("new sheet %q: %w", name, err)
			}
		}

		if err := writeSheet(f, name, sh, styleMap); err != nil {
			return fmt.Errorf("write sheet %q: %w", name, err)
		}
	}

	if err := f.Write(out); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}
	return nil
}

func writeSheet(f *excelize.File, name string, sh Sheet, styleMap map[int]int) error {
	// AoA 形式 (rows) の展開: 1 行目 = 1 行目に配置
	for r, row := range sh.Rows {
		for c, v := range row {
			axis, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(name, axis, v); err != nil {
				return err
			}
		}
	}

	// Cell Object 形式
	for axis, cell := range sh.Cells {
		if err := setCell(f, name, axis, cell, styleMap); err != nil {
			return fmt.Errorf("set cell %s: %w", axis, err)
		}
	}

	// 列幅 (Excel 制限: 0 < width <= 255)
	for _, c := range sh.Cols {
		if c.Col == "" || c.Width <= 0 {
			continue
		}
		w := c.Width
		if w > 255 {
			w = 255
		}
		if err := f.SetColWidth(name, c.Col, c.Col, w); err != nil {
			return err
		}
	}

	// 行高 (Excel 制限: 0 < height <= 409)
	for _, rd := range sh.RowDims {
		if rd.Row <= 0 || rd.Height <= 0 {
			continue
		}
		h := rd.Height
		if h > 409 {
			h = 409
		}
		if err := f.SetRowHeight(name, rd.Row, h); err != nil {
			return err
		}
	}

	// マージ
	for _, m := range sh.Merges {
		parts := strings.Split(m.Range, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid merge range: %q", m.Range)
		}
		if err := f.MergeCell(name, parts[0], parts[1]); err != nil {
			return err
		}
	}

	return nil
}

func setCell(f *excelize.File, sheet, axis string, c Cell, styleMap map[int]int) error {
	switch c.T {
	case "f":
		if c.F == "" {
			return fmt.Errorf("cell %s: type=f but formula empty", axis)
		}
		if err := f.SetCellFormula(sheet, axis, c.F); err != nil {
			return err
		}
		// 計算済み値があれば設定 (excelize は値を自動計算しないため任意)
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
			// SetCellValue は数式を上書きしてしまうため再設定
			if err := f.SetCellFormula(sheet, axis, c.F); err != nil {
				return err
			}
		}
	case "b":
		bv, ok := c.V.(bool)
		if !ok {
			return fmt.Errorf("cell %s: type=b but value not bool", axis)
		}
		if err := f.SetCellBool(sheet, axis, bv); err != nil {
			return err
		}
	case "n":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	case "s", "":
		if c.V != nil {
			if err := f.SetCellValue(sheet, axis, c.V); err != nil {
				return err
			}
		}
	case "d":
		if err := f.SetCellValue(sheet, axis, c.V); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cell %s: unknown type %q", axis, c.T)
	}

	// ハイパーリンク
	if c.L != nil {
		target, tooltip := parseLink(c.L)
		if target != "" {
			opts := []excelize.HyperlinkOpts{}
			if tooltip != "" {
				opts = append(opts, excelize.HyperlinkOpts{Tooltip: &tooltip})
			}
			if err := f.SetCellHyperLink(sheet, axis, target, "External", opts...); err != nil {
				return err
			}
		}
	}

	// スタイル適用 (z 単独指定にも対応)
	styleID, ok := styleMap[c.S]
	if c.S != 0 && ok {
		if c.Z != "" {
			// z を一時的に追加したスタイルを作る
			id, err := mergeNumFmt(f, styleID, c.Z)
			if err != nil {
				return err
			}
			styleID = id
		}
		if err := f.SetCellStyle(sheet, axis, axis, styleID); err != nil {
			return err
		}
	} else if c.Z != "" {
		id, err := f.NewStyle(&excelize.Style{NumFmt: 0, CustomNumFmt: &c.Z})
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, axis, axis, id); err != nil {
			return err
		}
	}

	return nil
}

func parseLink(l interface{}) (target, tooltip string) {
	switch v := l.(type) {
	case string:
		return v, ""
	case map[string]interface{}:
		if t, ok := v["target"].(string); ok {
			target = t
		}
		if t, ok := v["tooltip"].(string); ok {
			tooltip = t
		}
	}
	return
}

func mergeNumFmt(f *excelize.File, baseID int, numFmt string) (int, error) {
	// シンプル化: 既存スタイルをコピーして numFmt を上書きするのは API 上難しいため
	// 新規スタイルを作成して numFmt のみ追加する。
	// (見た目の優先度として z はセル単位の数値書式上書きとして扱う)
	return f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
}
