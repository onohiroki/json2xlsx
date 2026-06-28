package json2xlsx

import (
	"bufio"
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

func ToCSV(r io.Reader, w io.Writer, sheetName string, sheetIndex int, dataJSON bool) error {
	br := bufio.NewReader(r)
	head, err := br.Peek(4)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read input: %w", err)
	}

	if bytes.Equal(head, []byte{'P', 'K', 0x03, 0x04}) {
		res, err := ReadWorkbook(br, dataJSON)
		if err != nil {
			return err
		}
		return convertWorkbookObjectToCSV(*res.Workbook, w, nil, sheetName, sheetIndex)
	}

	data, err := io.ReadAll(br)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	if dataJSON {
		return convertDataJSONToCSV(data, w)
	}

	isWorkbook, trimmed, err := detectCSVInputFormat(data)
	if err != nil {
		return err
	}

	if isWorkbook {
		return convertWorkbookToCSV(trimmed, w, sheetName, sheetIndex)
	}

	return convertArrayOfObjectsToCSV(trimmed, w)
}

func convertArrayOfObjectsToCSV(data []byte, w io.Writer) error {
	dec := json.NewDecoder(bytes.NewReader(data))
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

func convertDataJSONToCSV(data []byte, w io.Writer) error {
	wb, err := UnmarshalWorkbook(data, true)
	if err != nil {
		return err
	}

	csvw := csv.NewWriter(w)
	for _, row := range wb.Rows {
		record := make([]string, len(row))
		for i, v := range row {
			record[i] = fmt.Sprint(v)
		}
		if err := csvw.Write(record); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}
	csvw.Flush()
	return csvw.Error()
}

func detectCSVInputFormat(data []byte) (isWorkbook bool, trimmed []byte, err error) {
	trimmed = bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return false, nil, errors.New("empty input")
	}
	trimmed = trimBOM(trimmed)
	trimmed = bytes.TrimLeft(trimmed, " \t\r\n")
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

func convertWorkbookToCSV(data []byte, w io.Writer, sheetName string, sheetIndex int) error {
	var wb Workbook
	if err := json.Unmarshal(data, &wb); err != nil {
		return fmt.Errorf("parse workbook json: %w", err)
	}
	normalizeWorkbook(&wb)
	return convertWorkbookObjectToCSV(wb, w, data, sheetName, sheetIndex)
}

func convertWorkbookObjectToCSV(wb Workbook, w io.Writer, data []byte, sheetName string, sheetIndex int) error {
	cells, err := resolveSheetCells(wb, sheetName, sheetIndex)
	if err != nil {
		return err
	}
	if len(cells) == 0 && data != nil {
		cells = guessCellMapFromData(data)
	}
	if len(cells) == 0 {
		return errors.New("empty input: no cells found")
	}

	grid, hasWarning := cellMapToGrid(cells)
	if len(grid) == 0 {
		return errors.New("empty input: no valid cells found")
	}

	if err := writeCSVRecords(w, grid); err != nil {
		return fmt.Errorf("write csv row: %w", err)
	}
	if hasWarning {
		fmt.Fprintln(os.Stderr, "Warning: Some cells have formulas but no values; treating them as empty.")
	}
	return nil
}

func writeCSVRecords(w io.Writer, records [][]string) error {
	csvw := csv.NewWriter(w)
	for _, record := range records {
		if err := csvw.Write(record); err != nil {
			return err
		}
	}
	csvw.Flush()
	return csvw.Error()
}

func resolveSheetCells(wb Workbook, sheetName string, sheetIndex int) (map[string]Cell, error) {
	switch {
	case sheetName != "":
		for _, s := range wb.Sheets {
			if s.Name == sheetName {
				return s.Cells, nil
			}
		}
		return nil, fmt.Errorf("sheet %q not found", sheetName)

	case sheetIndex > 0:
		idx := sheetIndex - 1
		if idx < len(wb.Sheets) {
			return wb.Sheets[idx].Cells, nil
		}
		return nil, fmt.Errorf("sheet index %d not found", sheetIndex)

	case len(wb.Sheets) > 0:
		return wb.Sheets[0].Cells, nil

	default:
		return nil, nil
	}
}

func guessCellMapFromData(data []byte) map[string]Cell {
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	for _, v := range m {
		sheet, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		isCells := true
		for k := range sheet {
			if _, _, err := excelize.CellNameToCoordinates(k); err != nil {
				isCells = false
				break
			}
		}
		if isCells && len(sheet) > 0 {
			cellData, _ := json.Marshal(sheet)
			var cells map[string]Cell
			json.Unmarshal(cellData, &cells)
			return cells
		}
	}
	return nil
}

func cellMapToGrid(cells map[string]Cell) ([][]string, bool) {
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
		return nil, hasWarning
	}

	grid := make([][]string, maxR)
	for i := range maxR {
		grid[i] = make([]string, maxC)
	}
	for _, ci := range cellList {
		grid[ci.r-1][ci.c-1] = ci.val
	}
	return grid, hasWarning
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
