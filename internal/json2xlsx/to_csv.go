package json2xlsx

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

	trimmed, err := normalizeCSVInput(data)
	if err != nil {
		return err
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

func normalizeCSVInput(data []byte) ([]byte, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return nil, errors.New("empty input")
	}

	if len(trimmed) >= 3 && trimmed[0] == 0xEF && trimmed[1] == 0xBB && trimmed[2] == 0xBF {
		trimmed = bytes.TrimLeft(trimmed[3:], " \t\r\n")
	}
	if len(trimmed) == 0 {
		return nil, errors.New("empty input")
	}

	switch trimmed[0] {
	case '[':
		return trimmed, nil
	case '{':
		return nil, errors.New("unsupported input: expected csvtk/xlsx-cli JSON (array of objects), got json2xlsx Workbook JSON (object)")
	default:
		lineEnd := bytes.IndexAny(trimmed, "\r\n")
		if lineEnd < 0 {
			return nil, errors.New("unsupported input: expected JSON array starting with '[' or xlsx-cli sheet name followed by JSON array")
		}
		rest := bytes.TrimLeft(trimmed[lineEnd+1:], " \t\r\n")
		if len(rest) == 0 {
			return nil, errors.New("unsupported input: expected xlsx-cli JSON array after sheet name")
		}
		if rest[0] == '{' {
			return nil, errors.New("unsupported input: expected xlsx-cli JSON array after sheet name, got JSON object")
		}
		if rest[0] != '[' {
			return nil, errors.New("unsupported input: expected xlsx-cli JSON array after sheet name")
		}
		return rest, nil
	}
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

