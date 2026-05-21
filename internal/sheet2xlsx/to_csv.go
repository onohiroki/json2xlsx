package sheet2xlsx

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return errors.New("empty input")
	}

	if len(trimmed) >= 3 && trimmed[0] == 0xEF && trimmed[1] == 0xBB && trimmed[2] == 0xBF {
		trimmed = trimmed[3:]
	}
	if len(trimmed) == 0 {
		return errors.New("empty input")
	}

	switch trimmed[0] {
	case '{':
		return errors.New("unsupported input: expected csvtk csv2json JSON (array of objects), got sheet2xlsx Workbook JSON (object)")
	case '[':
	default:
		return errors.New("unsupported input: expected csvtk csv2json JSON array starting with '['")
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))

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
		if _, ok := t.(json.Delim); !ok {
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

			var val *string
			if err := dec.Decode(&val); err != nil {
				return fmt.Errorf("parse value for key %q: %w", k, err)
			}

			row = append(row, csvKeyValue{Key: k, Value: val})
		}

		t, err = dec.Token()
		if err != nil {
			return fmt.Errorf("parse object end: %w", err)
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
	if _, ok := t.(json.Delim); !ok {
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
