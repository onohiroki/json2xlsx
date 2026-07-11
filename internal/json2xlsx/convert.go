package json2xlsx

import (
	"fmt"
	"io"
	"os"
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
	// EvalFormulas が true の場合，t="f" かつ v のないセルの数式を評価し，
	// 計算結果を v に補完する．
	EvalFormulas bool
	// BaseDir は画像パス解決の基準ディレクトリ．空の場合は CWD を使用．
	BaseDir string
}

// Convert は JSON を読み込み，XLSX を out に書き出す．
func Convert(r io.Reader, out io.Writer, opts ConvertOptions) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	wb, err := UnmarshalWorkbook(data, opts.DataJSON)
	if err != nil {
		return err
	}

	var formulaWarnings []string
	if opts.EvalFormulas {
		formulaWarnings = EvalWorkbookFormulas(wb)
	}

	if err := convertWorkbook(wb, out, opts.BaseDir); err != nil {
		if !opts.DataJSON {
			if schemaErr := ValidateJSON(data); schemaErr != nil {
				return fmt.Errorf("%v\n\n%v", err, schemaErr)
			}
		}
		return err
	}
	for _, msg := range formulaWarnings {
		fmt.Fprintln(os.Stderr, msg)
	}
	return nil
}
