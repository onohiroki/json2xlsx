package json2xlsx

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EvalOptions は単一数式の評価オプション．
type EvalOptions struct {
	// Sheet は評価文脈に使うシート名 (空なら先頭シート)．
	Sheet string
	// Cell は評価原点セル (空なら "A1")．循環参照検出とキャッシュの鍵に使う．
	Cell string
}

// EvalResult は数式評価の結果．成功時は Value，失敗時は Code/Error が入る．
type EvalResult struct {
	Value any    // float64 | string
	Code  string // "#DIV/0!" など．成功時は空
	Error string // 元のエラーメッセージ．成功時は空
}

// MarshalJSON は eval --json 用の形状でエンコードする．
// 成功: {"value": <v>}，失敗: {"error": {"code": "...", "message": "..."}}
func (r EvalResult) MarshalJSON() ([]byte, error) {
	if r.Code != "" || r.Error != "" {
		return json.Marshal(struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}{Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: r.Code, Message: r.Error}})
	}
	return json.Marshal(struct {
		Value any `json:"value"`
	}{Value: r.Value})
}

// EvalFormula は単一の数式文字列を評価する．
//
// wb が nil の場合は空の文脈で評価する (セル参照はすべて 0)．
// wb がある場合は Sheet/Cell で指定した文脈で評価する．
// 先頭の "=" は 1 つだけ取り除く．数式エラーは err ではなく EvalResult に格納して返す．
func EvalFormula(formula string, wb *Workbook, opts EvalOptions) (EvalResult, error) {
	f := strings.TrimPrefix(strings.TrimSpace(formula), "=")
	if strings.TrimSpace(f) == "" {
		return EvalResult{}, fmt.Errorf("empty formula")
	}

	origin := opts.Cell
	if origin == "" {
		origin = "A1"
	}
	origin = normalizeCellRef(origin)

	cells := map[string]Cell{}
	if wb != nil {
		normalizeWorkbook(wb)
		sh, err := resolveSheet(*wb, opts.Sheet, 0)
		if err != nil {
			return EvalResult{}, err
		}
		if sh.Cells != nil {
			cells = sh.Cells
		}
	}

	ctx := newEvalContext(cells)
	fv, err := ctx.evaluate(origin, f)
	if err != nil {
		return EvalResult{Code: errorToExcelCode(err), Error: err.Error()}, nil
	}
	if fv.kind == valueString {
		return EvalResult{Value: fv.str}, nil
	}
	return EvalResult{Value: fv.num}, nil
}

// errorToExcelCode はエラーメッセージから Excel のエラーコードを推定する．
// 見つからなければ #VALUE! を返す．
func errorToExcelCode(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// エンジンが返す説明的なメッセージを Excel コードへ対応付ける．
	switch {
	case strings.Contains(msg, "division by zero"):
		return "#DIV/0!"
	case strings.Contains(msg, "circular reference"):
		return "#CIRC!"
	}
	for _, code := range []string{
		"#DIV/0!", "#N/A", "#NUM!", "#NAME?", "#REF!", "#NULL!",
		"#SPILL!", "#CALC!", "#CIRC!", "#VALUE!",
	} {
		if strings.Contains(msg, code) {
			return code
		}
	}
	return "#VALUE!"
}
