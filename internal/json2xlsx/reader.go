package json2xlsx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ReadWorkbookResult は ReadWorkbook の戻り値．
type ReadWorkbookResult struct {
	Workbook *Workbook
	IsXLSX   bool
	// RawData は入力が JSON だった場合の生バイト列．
	// JSON array かどうかの判定などに利用する．XLSX の場合は nil．
	RawData []byte
}

// ReadWorkbook は入力ストリームを magic byte (PK\x03\x04) で判定し，
// XLSX または JSON Workbook としてパースする．
// XLSX の場合は extractWorkbookWithOptions (DateModeDisplay) を，
// JSON の場合は UnmarshalWorkbook → normalizeDateCells を適用する．
func ReadWorkbook(r io.Reader, dataJSON bool) (*ReadWorkbookResult, error) {
	return ReadWorkbookWithFormat(r, dataJSON, false)
}

// inputIsXLSX は先頭 4 バイトが XLSX の magic byte かどうかを判定する．
// 入力が 4 バイト未満の場合は JSON として扱う（io.EOF はエラーにしない）．
func inputIsXLSX(br *bufio.Reader) (bool, error) {
	head, err := br.Peek(4)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bytes.Equal(head, []byte{'P', 'K', 0x03, 0x04}), nil
}

// ReadWorkbookWithFormat は dataJSON / tableJSON を指定して入力をパースする．
func ReadWorkbookWithFormat(r io.Reader, dataJSON, tableJSON bool) (*ReadWorkbookResult, error) {
	if dataJSON && tableJSON {
		return nil, fmt.Errorf("--data-json and --table-json cannot be used together")
	}
	br := bufio.NewReader(r)
	isXLSX, err := inputIsXLSX(br)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	if isXLSX {
		data, err := io.ReadAll(br)
		if err != nil {
			return nil, fmt.Errorf("read input: %w", err)
		}
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("open xlsx: %w", err)
		}
		defer f.Close()
		tmp, err := extractWorkbookWithOptions(f, ToJSONOptions{DateMode: DateModeDisplay})
		if err != nil {
			return nil, err
		}
		normalizeWorkbook(&tmp)
		return &ReadWorkbookResult{Workbook: &tmp, IsXLSX: true}, nil
	}

	data, err := io.ReadAll(br)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	// UTF-8 BOM を除去
	data = trimBOM(data)
	var wb *Workbook
	if tableJSON {
		wb, err = UnmarshalTableJSON(data)
	} else {
		wb, err = UnmarshalWorkbook(data, dataJSON)
	}
	if err != nil {
		return nil, err
	}
	normalizeDateCells(wb)
	return &ReadWorkbookResult{Workbook: wb, RawData: data}, nil
}
