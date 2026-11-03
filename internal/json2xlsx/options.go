package json2xlsx

import (
	"fmt"
	"io"
	"os"
)

// ResolveDateMode は排他的に指定された日付モードのフラグを解決する．
func ResolveDateMode(display, rfc3339, serial bool) (DateMode, error) {
	count := 0
	mode := DateModeSerial
	if display {
		count++
		mode = DateModeDisplay
	}
	if rfc3339 {
		count++
		mode = DateModeRFC3339
	}
	if serial {
		count++
		mode = DateModeSerial
	}
	if count > 1 {
		return "", fmt.Errorf("choose only one of --date-display, --date-rfc3339, --date-serial")
	}
	return mode, nil
}

// EmitWarning は警告を出力する．warn が nil の場合は従来通り stderr に出す．
func EmitWarning(warn func(string), msg string) {
	if warn != nil {
		warn(msg)
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

// IsWarningOnly は警告だけで処理が完了したことを示すエラーかどうかを返す．
func IsWarningOnly(err error) bool {
	_, ok := err.(*WarningError)
	return ok
}

// readAllLimited は limit バイトを超える入力をエラーにする．
// limit が 0 以下の場合は制限しない．
func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(r)
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("input exceeds the size limit of %d bytes", limit)
	}
	return data, nil
}
