package sheet2xlsx

// DateMode は to-json の日時出力モード。
type DateMode string

const (
	// DateModeDisplay は Excel の表示文字列を出力する。
	DateModeDisplay DateMode = "display"
	// DateModeRFC3339 は Excel シリアル値を RFC3339 (UTC) に再解釈して出力する。
	DateModeRFC3339 DateMode = "rfc3339"
	// DateModeSerial は Excel シリアル値をそのまま数値として出力する。
	DateModeSerial DateMode = "serial"
)

