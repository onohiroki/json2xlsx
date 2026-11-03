package json2xlsx

import (
	"fmt"
	"io"
)

// WarningError は非 fatal な警告を表す．
// 処理は継続され，XLSX 出力は行われるが，exit code は非零になる．
type WarningError struct {
	Err error
}

func (w *WarningError) Error() string { return w.Err.Error() }
func (w *WarningError) Unwrap() error { return w.Err }

// ConvertOptions は Convert の動作オプション．
type ConvertOptions struct {
	// DataJSON が true の場合，入力を「データ JSON」として扱い，
	// 二次元配列 / オブジェクト配列 / Map-of-Arrays の 3 形式を自動判別する．
	// false (デフォルト) の場合は SheetJS 形式のみを受け付け，失敗したらエラーを返す．
	DataJSON bool
	// TableJSON が true の場合，入力を高レベルな table JSON として扱う．
	// DataJSON とは排他．
	TableJSON bool
	// EvalFormulas が true の場合，t="f" かつ v のないセルの数式を評価し，
	// 計算結果を v に補完する．
	EvalFormulas bool
	// AutoFit が true の場合，列幅をセル内容に合わせて自動調整する．
	// 明示的に cols[].width が指定された列はその値が優先される．
	// また，改行 (\n) を含むセルには WrapText が自動適用される（行高は設定しない）．
	AutoFit bool
	// BaseDir は画像パス解決の基準ディレクトリ．空の場合は CWD を使用．
	BaseDir string
	// Warn は警告の出力先．nil の場合は stderr に出力する．
	Warn func(string)
	// MaxInputBytes は入力の最大バイト数．0 以下は無制限．
	MaxInputBytes int64
}

// ParseWorkbook は JSON バイト列を Workbook にパースして正規化する．
func ParseWorkbook(data []byte, opts ConvertOptions) (*Workbook, error) {
	if opts.TableJSON {
		if opts.DataJSON {
			return nil, fmt.Errorf("--data-json and --table-json cannot be used together")
		}
		return UnmarshalTableJSON(data)
	}
	return UnmarshalWorkbook(data, opts.DataJSON)
}

// ParseWorkbookReader は reader から読み込んで ParseWorkbook する．
// 戻り値の data はスキーマ検証のフォールバックに使う．
func ParseWorkbookReader(r io.Reader, opts ConvertOptions) (*Workbook, []byte, error) {
	data, err := readAllLimited(r, opts.MaxInputBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("read input: %w", err)
	}
	wb, err := ParseWorkbook(data, opts)
	if err != nil {
		return nil, data, err
	}
	return wb, data, nil
}

// ConvertWorkbook はパース済みの Workbook を XLSX として out に書き出す．
func ConvertWorkbook(wb *Workbook, out io.Writer, opts ConvertOptions) error {
	var formulaWarnings []string
	if opts.EvalFormulas {
		formulaWarnings = EvalWorkbookFormulas(wb)
	}

	var warnings []string
	if err := convertWorkbook(wb, out, opts.BaseDir, opts.AutoFit, &warnings); err != nil {
		return err
	}

	warnings = append(formulaWarnings, warnings...)
	for _, msg := range warnings {
		EmitWarning(opts.Warn, msg)
	}
	if len(warnings) > 0 {
		return &WarningError{Err: fmt.Errorf("conversion completed with %d warning(s)", len(warnings))}
	}
	return nil
}

// Convert は JSON を読み込み，XLSX を out に書き出す．
func Convert(r io.Reader, out io.Writer, opts ConvertOptions) error {
	data, err := readAllLimited(r, opts.MaxInputBytes)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	wb, err := ParseWorkbook(data, opts)
	if err != nil {
		return err
	}

	if err := ConvertWorkbook(wb, out, opts); err != nil {
		if IsWarningOnly(err) {
			return err
		}
		if !opts.DataJSON && !opts.TableJSON {
			if schemaErr := ValidateJSON(data); schemaErr != nil {
				return fmt.Errorf("%v\n\n%v", err, schemaErr)
			}
		}
		return err
	}
	return nil
}
