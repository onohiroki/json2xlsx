package json2xlsx

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/xuri/excelize/v2"
)

type csvKeyValue struct {
	Key   string
	Value *string
}

func ToCSV(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	isWorkbook, trimmed, err := normalizeCSVInput(data)
	if err != nil {
		return err
	}

	if isWorkbook {
		return convertWorkbookToCSV(trimmed, w)
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()

	t, err := dec.Token()
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	delim, ok := t.(json.Delim)
	if !ok || delim != '[' {
		return fmt.Errorf("expected JSON array, got %v", t)
	}

	var allKeys []string
	keySet := make(map[string]bool)
	var rows [][]csvKeyValue

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return fmt.Errorf("parse object: %w", err)
		}
		delim, ok := t.(json.Delim)
		if !ok || delim != '{' {
			return fmt.Errorf("expected JSON object, got %v", t)
		}

		var row []csvKeyValue
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return fmt.Errorf("parse key: %w", err)
			}
			k, ok := key.(string)
			if !ok {
				return fmt.Errorf("expected string key, got %v", key)
			}

			var raw interface{}
			if err := dec.Decode(&raw); err != nil {
				return fmt.Errorf("parse value for key %q: %w", k, err)
			}

			val, err := normalizeCSVValue(k, raw)
			if err != nil {
				return err
			}

			row = append(row, csvKeyValue{Key: k, Value: val})
		}

		t, err = dec.Token()
		if err != nil {
			return fmt.Errorf("parse object end: %w", err)
		}
		delim, ok = t.(json.Delim)
		if !ok || delim != '}' {
			return fmt.Errorf("expected end of JSON object, got %v", t)
		}

		if len(row) == 0 {
			continue
		}

		if allKeys == nil {
			for _, kv := range row {
				allKeys = append(allKeys, kv.Key)
				keySet[kv.Key] = true
			}
		} else {
			for _, kv := range row {
				if !keySet[kv.Key] {
					allKeys = append(allKeys, kv.Key)
					keySet[kv.Key] = true
				}
			}
		}
		rows = append(rows, row)
	}

	t, err = dec.Token()
	if err != nil {
		return fmt.Errorf("parse json: %w", err)
	}
	delim, ok = t.(json.Delim)
	if !ok || delim != ']' {
		return fmt.Errorf("expected end of array, got %v", t)
	}

	if len(rows) == 0 {
		return errors.New("empty input: no data rows found")
	}
	if len(allKeys) == 0 {
		return errors.New("empty input: no columns found")
	}

	csvw := csv.NewWriter(w)

	if err := csvw.Write(allKeys); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(allKeys))
		for i, key := range allKeys {
			for _, kv := range row {
				if kv.Key == key {
					if kv.Value != nil {
						record[i] = *kv.Value
					}
					break
				}
			}
		}
		if err := csvw.Write(record); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	csvw.Flush()
	return csvw.Error()
}

func normalizeCSVInput(data []byte) (isWorkbook bool, trimmed []byte, err error) {
	trimmed = bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return false, nil, errors.New("empty input")
	}

	if len(trimmed) >= 3 && trimmed[0] == 0xEF && trimmed[1] == 0xBB && trimmed[2] == 0xBF {
		trimmed = bytes.TrimLeft(trimmed[3:], " \t\r\n")
	}
	if len(trimmed) == 0 {
		return false, nil, errors.New("empty input")
	}

	switch trimmed[0] {
	case '[':
		return false, trimmed, nil
	case '{':
		return true, trimmed, nil
	default:
		lineEnd := bytes.IndexAny(trimmed, "\r\n")
		if lineEnd < 0 {
			return false, nil, errors.New("unsupported input: expected JSON array starting with '[' or xlsx-cli sheet name followed by JSON array")
		}
		rest := bytes.TrimLeft(trimmed[lineEnd+1:], " \t\r\n")
		if len(rest) == 0 {
			return false, nil, errors.New("unsupported input: expected xlsx-cli JSON array after sheet name")
		}
		if rest[0] == '{' {
			return true, rest, nil
		}
		if rest[0] != '[' {
			return false, nil, errors.New("unsupported input: expected xlsx-cli JSON array after sheet name")
		}
		return false, rest, nil
	}
}

func convertWorkbookToCSV(data []byte, w io.Writer) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	var wb Workbook
	json.Unmarshal(data, &wb)

	var cells map[string]Cell
	if wb.Cells != nil {
		cells = wb.Cells
	} else if len(wb.Sheets) > 0 {
		cells = wb.Sheets[0].Cells
	} else if wb.Book != nil && len(wb.Book.Sheets) > 0 {
		for _, s := range wb.Book.Sheets {
			cells = s.Cells
			break
		}
	} else {
		// Try to find any sheet-like object if it's a simple map of sheet names to cells
		for _, v := range m {
			if sheet, ok := v.(map[string]interface{}); ok {
				// Check if it looks like a cells map (keys are cell addresses)
				isCells := true
				for k := range sheet {
					if _, _, err := excelize.CellNameToCoordinates(k); err != nil {
						isCells = false
						break
					}
				}
				if isCells && len(sheet) > 0 {
					// Re-unmarshal this part as cells
					cellData, _ := json.Marshal(sheet)
					json.Unmarshal(cellData, &cells)
					break
				}
			}
		}
	}

	if len(cells) == 0 {
		return errors.New("empty input: no cells found")
	}

	type cellInfo struct {
		r, c int
		val  string
	}
	var cellList []cellInfo
	maxR, maxC := 0, 0
	var hasWarning bool

	for addr, cell := range cells {
		col, row, err := excelize.CellNameToCoordinates(addr)
		if err != nil {
			continue
		}
		val := ""
		if cell.V != nil {
			val = fmt.Sprint(cell.V)
		} else if cell.F != "" {
			hasWarning = true
		}

		cellList = append(cellList, cellInfo{row, col, val})
		maxR = max(maxR, row)
		maxC = max(maxC, col)
	}

	if len(cellList) == 0 {
		return errors.New("empty input: no valid cells found")
	}

	// 1-based to 0-based for matrix, but maxR/maxC are 1-based sizes
	grid := make([][]string, maxR)
	for i := range maxR {
		grid[i] = make([]string, maxC)
	}

	for _, ci := range cellList {
		grid[ci.r-1][ci.c-1] = ci.val
	}

	csvw := csv.NewWriter(w)
	for _, row := range grid {
		if err := csvw.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	csvw.Flush()
	if hasWarning {
		fmt.Fprintln(os.Stderr, "Warning: Some cells have formulas but no values; treating them as empty.")
	}
	return csvw.Error()
}

func normalizeCSVValue(key string, raw interface{}) (*string, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		vv := v
		return &vv, nil
	case json.Number:
		vv := v.String()
		return &vv, nil
	default:
		return nil, fmt.Errorf("unsupported value for key %q: expected string, number, or null, got %T", key, raw)
	}
}
