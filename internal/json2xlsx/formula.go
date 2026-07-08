package json2xlsx

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// EvalWorkbookFormulas evaluates all formulas in the workbook that lack
// a cached value (v). Cells with t="f" and no v are evaluated and their
// results are written into v. Cells that fail evaluation are skipped.
// Warning messages for failed evaluations are returned.
func EvalWorkbookFormulas(wb *Workbook) []string {
	var warnings []string
	for si := range wb.Sheets {
		sh := &wb.Sheets[si]
		if len(sh.Cells) == 0 {
			continue
		}
		ctx := newEvalContext(sh.Cells)
		for axis, cell := range sh.Cells {
			if cell.T == "f" && cell.V == nil && cell.F != "" {
				val, err := ctx.evaluate(axis, cell.F)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("warning: %s=%s: %v", axis, cell.F, err))
					continue
				}
				c := sh.Cells[axis]
				c.V = val
				c.T = "n"
				c.F = ""
				sh.Cells[axis] = c
			}
		}
	}
	return warnings
}

// ---------------------------------------------------------------------------
// Tokenizer
// ---------------------------------------------------------------------------

type tokenType int

const (
	tokenNumber  tokenType = iota
	tokenCellRef
	tokenColon
	tokenFunc
	tokenPlus
	tokenMinus
	tokenStar
	tokenSlash
	tokenLParen
	tokenRParen
	tokenComma
	tokenEQ
	tokenNE
	tokenLT
	tokenGT
	tokenLE
	tokenGE
	tokenEOF
	tokenIllegal
)

type token struct {
	typ tokenType
	lit string
}

type tokenizer struct {
	input string
	pos   int
}

func newTokenizer(input string) *tokenizer {
	return &tokenizer{input: input, pos: 0}
}

var knownFuncs = map[string]bool{
	"SUM": true, "AVERAGE": true, "COUNT": true,
	"MIN": true, "MAX": true, "ABS": true, "ROUND": true,
	"IF": true, "AND": true, "OR": true, "NOT": true,
	"PRODUCT": true, "ROUNDUP": true, "ROUNDDOWN": true, "SUMPRODUCT": true,
	"MEDIAN": true, "QUARTILE": true, "QUARTILE.INC": true, "PERCENTILE": true, "PERCENTILE.INC": true,
	"STDEV": true, "STDEV.S": true, "STDEV.P": true,
	"SUMIF": true, "COUNTIF": true,
	"FLOOR": true, "CEILING": true, "MOD": true, "POWER": true, "SQRT": true, "INT": true,
	"COUNTA": true,
	"VAR": true, "VAR.S": true, "VAR.P": true,
	"GEOMEAN": true, "HARMEAN": true, "TRIMMEAN": true,
	"RANK": true, "RANK.EQ": true, "LARGE": true, "SMALL": true,
	"TODAY": true, "NOW": true,
	"AVERAGEIF": true, "SUMIFS": true, "COUNTIFS": true, "AVERAGEIFS": true,
	"MINIFS": true, "MAXIFS": true,
	"IFERROR": true,
	"IFS": true, "SWITCH": true,
	"YEAR": true, "MONTH": true, "DAY": true, "DAYS": true, "DATE": true, "EDATE": true, "EOMONTH": true, "WEEKDAY": true, "WEEKNUM": true,
	"NETWORKDAYS": true, "WORKDAY": true,
	"VLOOKUP": true, "XLOOKUP": true, "INDEX": true, "MATCH": true, "CHOOSE": true,
	"TRUNC": true, "SIGN": true, "PI": true, "RAND": true,
	"SIN": true, "COS": true, "TAN": true, "LN": true, "LOG10": true, "EXP": true,
	"ASIN": true, "ACOS": true, "ATAN": true, "DEGREES": true, "RADIANS": true,
	"ATAN2": true, "SINH": true, "COSH": true, "TANH": true,
	"ASINH": true, "ACOSH": true, "ATANH": true,
	"LOG": true, "FACT": true,
	"SUMSQ": true, "EVEN": true, "ODD": true, "MROUND": true, "DELTA": true, "GESTEP": true,
	"HLOOKUP": true,
	"MODE": true, "MODE.SNGL": true, "SUBTOTAL": true,
	"ISNUMBER": true, "ISBLANK": true, "ISTEXT": true, "ISNONTEXT": true, "ISERROR": true, "ISNA": true,
	"ROW": true, "COLUMN": true,
}

func (t *tokenizer) next() token {
	t.skipWhitespace()
	if t.pos >= len(t.input) {
		return token{typ: tokenEOF}
	}

	ch := t.input[t.pos]

	switch {
	case ch == '+':
		t.pos++
		return token{typ: tokenPlus, lit: "+"}
	case ch == '-':
		t.pos++
		return token{typ: tokenMinus, lit: "-"}
	case ch == '*':
		t.pos++
		return token{typ: tokenStar, lit: "*"}
	case ch == '/':
		t.pos++
		return token{typ: tokenSlash, lit: "/"}
	case ch == '(':
		t.pos++
		return token{typ: tokenLParen, lit: "("}
	case ch == ')':
		t.pos++
		return token{typ: tokenRParen, lit: ")"}
	case ch == ',':
		t.pos++
		return token{typ: tokenComma, lit: ","}
	case ch == ':':
		t.pos++
		return token{typ: tokenColon, lit: ":"}
	case ch == '<':
		t.pos++
		if t.pos < len(t.input) && t.input[t.pos] == '=' {
			t.pos++
			return token{typ: tokenLE, lit: "<="}
		} else if t.pos < len(t.input) && t.input[t.pos] == '>' {
			t.pos++
			return token{typ: tokenNE, lit: "<>"}
		}
		return token{typ: tokenLT, lit: "<"}
	case ch == '>':
		t.pos++
		if t.pos < len(t.input) && t.input[t.pos] == '=' {
			t.pos++
			return token{typ: tokenGE, lit: ">="}
		}
		return token{typ: tokenGT, lit: ">"}
	case ch == '=':
		t.pos++
		return token{typ: tokenEQ, lit: "="}
	case ch == '.' || (ch >= '0' && ch <= '9'):
		return t.readNumber()
	default:
		return t.readIdent()
	}
}

func (t *tokenizer) skipWhitespace() {
	for t.pos < len(t.input) && (t.input[t.pos] == ' ' || t.input[t.pos] == '\t') {
		t.pos++
	}
}

func (t *tokenizer) readNumber() token {
	start := t.pos
	isFloat := false
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch >= '0' && ch <= '9' {
			t.pos++
		} else if ch == '.' && !isFloat {
			isFloat = true
			t.pos++
		} else {
			break
		}
	}
	return token{typ: tokenNumber, lit: t.input[start:t.pos]}
}

func (t *tokenizer) readIdent() token {
	start := t.pos
	for t.pos < len(t.input) {
		ch := t.input[t.pos]
		if ch == '$' || ch == '.' || ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			t.pos++
		} else {
			break
		}
	}
	raw := t.input[start:t.pos]
	cleaned := strings.ReplaceAll(raw, "$", "")
	upper := strings.ToUpper(cleaned)
	// Strip _xlfn. prefix added by Excel for future/unsupported functions.
	upper = strings.TrimPrefix(upper, "_XLFN.")

	if knownFuncs[upper] {
		return token{typ: tokenFunc, lit: upper}
	}

	if looksLikeCellRef(cleaned) {
		return token{typ: tokenCellRef, lit: strings.ToUpper(cleaned)}
	}

	return token{typ: tokenIllegal, lit: raw}
}

// looksLikeCellRef checks if s matches [A-Za-z]{1,3}[0-9]+.
func looksLikeCellRef(s string) bool {
	if len(s) == 0 {
		return false
	}
	i := 0
	for i < len(s) && ((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
		i++
	}
	if i == 0 || i > 3 {
		return false
	}
	for j := i; j < len(s); j++ {
		if s[j] < '0' || s[j] > '9' {
			return false
		}
	}
	return i < len(s)
}

// ---------------------------------------------------------------------------
// AST
// ---------------------------------------------------------------------------

type expr interface {
	eval(ctx *evalContext) (float64, error)
}

type numberExpr struct {
	val float64
}

func (e *numberExpr) eval(ctx *evalContext) (float64, error) {
	return e.val, nil
}

type cellRefExpr struct {
	ref string
}

func (e *cellRefExpr) eval(ctx *evalContext) (float64, error) {
	return ctx.getCellValue(normalizeCellRef(e.ref))
}

type rangeExpr struct {
	start, end string
}

func (e *rangeExpr) eval(ctx *evalContext) (float64, error) {
	return 0, fmt.Errorf("range %s:%s cannot be used outside a function", e.start, e.end)
}

type binaryExpr struct {
	left, right expr
	op          tokenType
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (e *binaryExpr) eval(ctx *evalContext) (float64, error) {
	left, err := e.left.eval(ctx)
	if err != nil {
		return 0, err
	}
	right, err := e.right.eval(ctx)
	if err != nil {
		return 0, err
	}
	switch e.op {
	case tokenPlus:
		return left + right, nil
	case tokenMinus:
		return left - right, nil
	case tokenStar:
		return left * right, nil
	case tokenSlash:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case tokenEQ:
		return boolToFloat(left == right), nil
	case tokenNE:
		return boolToFloat(left != right), nil
	case tokenLT:
		return boolToFloat(left < right), nil
	case tokenGT:
		return boolToFloat(left > right), nil
	case tokenLE:
		return boolToFloat(left <= right), nil
	case tokenGE:
		return boolToFloat(left >= right), nil
	}
	return 0, fmt.Errorf("internal: unknown binary operator %d", e.op)
}

type unaryExpr struct {
	operand expr
	op      tokenType
}

func (e *unaryExpr) eval(ctx *evalContext) (float64, error) {
	val, err := e.operand.eval(ctx)
	if err != nil {
		return 0, err
	}
	if e.op == tokenMinus {
		return -val, nil
	}
	return val, nil
}

type funcCallExpr struct {
	name string
	args []expr
}

func (e *funcCallExpr) eval(ctx *evalContext) (float64, error) {
	switch e.name {
	case "SUM":
		return evalFuncSum(ctx, e.args)
	case "AVERAGE":
		return evalFuncAverage(ctx, e.args)
	case "COUNT":
		return evalFuncCount(ctx, e.args)
	case "MIN":
		return evalFuncMin(ctx, e.args)
	case "MAX":
		return evalFuncMax(ctx, e.args)
	case "ABS":
		return evalFuncAbs(ctx, e.args)
	case "ROUND":
		return evalFuncRound(ctx, e.args)
	case "IF":
		return evalFuncIf(ctx, e.args)
	case "AND":
		return evalFuncAnd(ctx, e.args)
	case "OR":
		return evalFuncOr(ctx, e.args)
	case "NOT":
		return evalFuncNot(ctx, e.args)
	case "PRODUCT":
		return evalFuncProduct(ctx, e.args)
	case "ROUNDUP":
		return evalFuncRoundup(ctx, e.args)
	case "ROUNDDOWN":
		return evalFuncRounddown(ctx, e.args)
	case "SUMPRODUCT":
		return evalFuncSumproduct(ctx, e.args)
	case "MEDIAN":
		return evalFuncMedian(ctx, e.args)
	case "QUARTILE", "QUARTILE.INC":
		return evalFuncQuartileInc(ctx, e.args)
	case "PERCENTILE", "PERCENTILE.INC":
		return evalFuncPercentileInc(ctx, e.args)
	case "STDEV", "STDEV.S":
		return evalFuncStdevS(ctx, e.args)
	case "STDEV.P":
		return evalFuncStdevP(ctx, e.args)
	case "SUMIF":
		return evalFuncSumif(ctx, e.args)
	case "COUNTIF":
		return evalFuncCountif(ctx, e.args)
	case "FLOOR":
		return evalFuncFloor(ctx, e.args)
	case "CEILING":
		return evalFuncCeiling(ctx, e.args)
	case "MOD":
		return evalFuncMod(ctx, e.args)
	case "POWER":
		return evalFuncPower(ctx, e.args)
	case "SQRT":
		return evalFuncSqrt(ctx, e.args)
	case "INT":
		return evalFuncInt(ctx, e.args)
	case "COUNTA":
		return evalFuncCounta(ctx, e.args)
	case "VAR", "VAR.S":
		return evalFuncVarS(ctx, e.args)
	case "VAR.P":
		return evalFuncVarP(ctx, e.args)
	case "GEOMEAN":
		return evalFuncGeomean(ctx, e.args)
	case "HARMEAN":
		return evalFuncHarmean(ctx, e.args)
	case "TRIMMEAN":
		return evalFuncTrimmean(ctx, e.args)
	case "RANK", "RANK.EQ":
		return evalFuncRank(ctx, e.args)
	case "LARGE":
		return evalFuncLarge(ctx, e.args)
	case "SMALL":
		return evalFuncSmall(ctx, e.args)
	case "TODAY":
		return evalFuncToday(ctx, e.args)
	case "NOW":
		return evalFuncNow(ctx, e.args)
	case "YEAR":
		return evalFuncYear(ctx, e.args)
	case "MONTH":
		return evalFuncMonth(ctx, e.args)
	case "DAY":
		return evalFuncDay(ctx, e.args)
	case "DAYS":
		return evalFuncDays(ctx, e.args)
	case "DATE":
		return evalFuncDate(ctx, e.args)
	case "EDATE":
		return evalFuncEdate(ctx, e.args)
	case "EOMONTH":
		return evalFuncEomonth(ctx, e.args)
	case "WEEKDAY":
		return evalFuncWeekday(ctx, e.args)
	case "WEEKNUM":
		return evalFuncWeeknum(ctx, e.args)
	case "NETWORKDAYS":
		return evalFuncNetworkdays(ctx, e.args)
	case "WORKDAY":
		return evalFuncWorkday(ctx, e.args)
	case "AVERAGEIF":
		return evalFuncAverageif(ctx, e.args)
	case "SUMIFS":
		return evalFuncSumifs(ctx, e.args)
	case "COUNTIFS":
		return evalFuncCountifs(ctx, e.args)
	case "AVERAGEIFS":
		return evalFuncAverageifs(ctx, e.args)
	case "MINIFS":
		return evalFuncMinifs(ctx, e.args)
	case "MAXIFS":
		return evalFuncMaxifs(ctx, e.args)
	case "IFS":
		return evalFuncIfs(ctx, e.args)
	case "SWITCH":
		return evalFuncSwitch(ctx, e.args)
	case "IFERROR":
		return evalFuncIferror(ctx, e.args)
	case "VLOOKUP":
		return evalFuncVlookup(ctx, e.args)
	case "XLOOKUP":
		return evalFuncXlookup(ctx, e.args)
	case "INDEX":
		return evalFuncIndex(ctx, e.args)
	case "MATCH":
		return evalFuncMatch(ctx, e.args)
	case "CHOOSE":
		return evalFuncChoose(ctx, e.args)
	case "TRUNC":
		return evalFuncTrunc(ctx, e.args)
	case "SIGN":
		return evalFuncSign(ctx, e.args)
	case "PI":
		return evalFuncPi(ctx, e.args)
	case "RAND":
		return evalFuncRand(ctx, e.args)
	case "SIN":
		return evalFuncSin(ctx, e.args)
	case "COS":
		return evalFuncCos(ctx, e.args)
	case "TAN":
		return evalFuncTan(ctx, e.args)
	case "LN":
		return evalFuncLn(ctx, e.args)
	case "LOG10":
		return evalFuncLog10(ctx, e.args)
	case "EXP":
		return evalFuncExp(ctx, e.args)
	case "ASIN":
		return evalFuncAsin(ctx, e.args)
	case "ACOS":
		return evalFuncAcos(ctx, e.args)
	case "ATAN":
		return evalFuncAtan(ctx, e.args)
	case "DEGREES":
		return evalFuncDegrees(ctx, e.args)
	case "RADIANS":
		return evalFuncRadians(ctx, e.args)
	case "ATAN2":
		return evalFuncAtan2(ctx, e.args)
	case "SINH":
		return evalFuncSinh(ctx, e.args)
	case "COSH":
		return evalFuncCosh(ctx, e.args)
	case "TANH":
		return evalFuncTanh(ctx, e.args)
	case "ASINH":
		return evalFuncAsinh(ctx, e.args)
	case "ACOSH":
		return evalFuncAcosh(ctx, e.args)
	case "ATANH":
		return evalFuncAtanh(ctx, e.args)
	case "LOG":
		return evalFuncLog(ctx, e.args)
	case "FACT":
		return evalFuncFact(ctx, e.args)
	case "SUMSQ":
		return evalFuncSumsq(ctx, e.args)
	case "EVEN":
		return evalFuncEven(ctx, e.args)
	case "ODD":
		return evalFuncOdd(ctx, e.args)
	case "MROUND":
		return evalFuncMround(ctx, e.args)
	case "DELTA":
		return evalFuncDelta(ctx, e.args)
	case "GESTEP":
		return evalFuncGestep(ctx, e.args)
	case "HLOOKUP":
		return evalFuncHlookup(ctx, e.args)
	case "MODE", "MODE.SNGL":
		return evalFuncMode(ctx, e.args)
	case "SUBTOTAL":
		return evalFuncSubtotal(ctx, e.args)
	case "ISNUMBER":
		return evalFuncIsnumber(ctx, e.args)
	case "ISBLANK":
		return evalFuncIsblank(ctx, e.args)
	case "ISTEXT":
		return evalFuncIstext(ctx, e.args)
	case "ISNONTEXT":
		return evalFuncIsnontext(ctx, e.args)
	case "ISERROR":
		return evalFuncIserror(ctx, e.args)
	case "ISNA":
		return evalFuncIsna(ctx, e.args)
	case "ROW":
		return evalFuncRow(ctx, e.args)
	case "COLUMN":
		return evalFuncColumn(ctx, e.args)
	}
	return 0, fmt.Errorf("unknown function: %s", e.name)
}

// ---------------------------------------------------------------------------
// Parser (recursive descent)
// ---------------------------------------------------------------------------
//
// Grammar:
//
//	expr         → comparison
//	comparison   → addition (('<' | '>' | '=' | '<=' | '>=' | '<>') addition)*
//	addition     → term (('+' | '-') term)*
//	term         → factor (('*' | '/') factor)*
//	factor       → primary (':' primary)?
//	primary      → NUMBER | CELL_REF | '(' expr ')' | FUNC '(' args ')' | '-' primary
//	args         → expr (',' expr)*

type parser struct {
	input  string
	tokens []token
	pos    int
	err    error
}

func newParser(input string) *parser {
	t := newTokenizer(input)
	p := &parser{input: input}
	for {
		tok := t.next()
		p.tokens = append(p.tokens, tok)
		if tok.typ == tokenEOF || tok.typ == tokenIllegal {
			break
		}
	}
	return p
}

func (p *parser) parse() (expr, error) {
	if p.err != nil {
		return nil, p.err
	}
	result := p.parseExpr()
	if p.err != nil {
		return nil, p.err
	}
	if p.peek().typ != tokenEOF {
		return nil, fmt.Errorf("unexpected token after expression: %q", p.peek().lit)
	}
	return result, nil
}

func (p *parser) parseExpr() expr {
	return p.parseComparison()
}

func (p *parser) parseComparison() expr {
	left := p.parseAddition()
	for p.peek().typ == tokenLT || p.peek().typ == tokenGT || p.peek().typ == tokenEQ ||
		p.peek().typ == tokenLE || p.peek().typ == tokenGE || p.peek().typ == tokenNE {
		op := p.next().typ
		right := p.parseAddition()
		left = &binaryExpr{left: left, right: right, op: op}
	}
	return left
}

func (p *parser) parseAddition() expr {
	left := p.parseTerm()
	for p.peek().typ == tokenPlus || p.peek().typ == tokenMinus {
		op := p.next().typ
		right := p.parseTerm()
		left = &binaryExpr{left: left, right: right, op: op}
	}
	return left
}

func (p *parser) parseTerm() expr {
	left := p.parseFactor()
	for p.peek().typ == tokenStar || p.peek().typ == tokenSlash {
		op := p.next().typ
		right := p.parseFactor()
		left = &binaryExpr{left: left, right: right, op: op}
	}
	return left
}

func (p *parser) parseFactor() expr {
	primary := p.parsePrimary()
	if p.peek().typ == tokenColon {
		p.next()
		right := p.parsePrimary()
		start, ok1 := primary.(*cellRefExpr)
		end, ok2 := right.(*cellRefExpr)
		if !ok1 || !ok2 {
			p.error("invalid range: expected cell references on both sides of ':'")
			return primary
		}
		return &rangeExpr{start: start.ref, end: end.ref}
	}
	return primary
}

func (p *parser) parsePrimary() expr {
	tok := p.next()
	switch tok.typ {
	case tokenNumber:
		val, err := strconv.ParseFloat(tok.lit, 64)
		if err != nil {
			p.error("invalid number: %s", tok.lit)
			return &numberExpr{val: 0}
		}
		return &numberExpr{val: val}

	case tokenCellRef:
		return &cellRefExpr{ref: tok.lit}

	case tokenFunc:
		return p.parseFuncCall(tok)

	case tokenLParen:
		e := p.parseExpr()
		if p.next().typ != tokenRParen {
			p.error("expected ')'")
		}
		return e

	case tokenMinus:
		return &unaryExpr{op: tokenMinus, operand: p.parsePrimary()}

	default:
		p.error("unexpected token: %q", tok.lit)
		return &numberExpr{val: 0}
	}
}

func (p *parser) parseFuncCall(fn token) expr {
	if p.next().typ != tokenLParen {
		p.error("expected '(' after function %s", fn.lit)
		return &funcCallExpr{name: fn.lit}
	}
	args := p.parseArgs()
	if p.next().typ != tokenRParen {
		p.error("expected ')' after arguments of %s", fn.lit)
	}
	return &funcCallExpr{name: fn.lit, args: args}
}

func (p *parser) parseArgs() []expr {
	var args []expr
	if p.peek().typ == tokenRParen {
		return args
	}
	args = append(args, p.parseExpr())
	for p.peek().typ == tokenComma {
		p.next()
		args = append(args, p.parseExpr())
	}
	return args
}

func (p *parser) next() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{typ: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) error(format string, args ...interface{}) {
	if p.err == nil {
		p.err = fmt.Errorf(format, args...)
	}
}

// ---------------------------------------------------------------------------
// Evaluator context
// ---------------------------------------------------------------------------

type evalContext struct {
	cells      map[string]Cell
	visiting   map[string]bool
	cache      map[string]float64
	formulaRef string
}

func newEvalContext(cells map[string]Cell) *evalContext {
	return &evalContext{
		cells:    cells,
		visiting: make(map[string]bool),
		cache:    make(map[string]float64),
	}
}

func (ctx *evalContext) evaluate(originAxis, formula string) (float64, error) {
	if cached, ok := ctx.cache[originAxis]; ok {
		return cached, nil
	}
	if ctx.visiting[originAxis] {
		return 0, fmt.Errorf("circular reference detected")
	}
	ctx.visiting[originAxis] = true
	defer delete(ctx.visiting, originAxis)

	ctx.formulaRef = originAxis
	p := newParser(formula)
	ast, err := p.parse()
	if err != nil {
		return 0, fmt.Errorf("parse error: %w", err)
	}
	val, err := ast.eval(ctx)
	if err != nil {
		return 0, err
	}
	ctx.cache[originAxis] = val
	return val, nil
}

// getCellValue returns the numeric value of a cell by reference.
// If the cell has a formula without a cached value, it is evaluated recursively.
func (ctx *evalContext) getCellValue(ref string) (float64, error) {
	cell, ok := ctx.cells[ref]
	if !ok {
		return 0, nil
	}
	if cell.V != nil {
		return toFloat64(cell.V), nil
	}
	if cell.T == "f" && cell.F != "" {
		if cached, ok := ctx.cache[ref]; ok {
			return cached, nil
		}
		return ctx.evaluate(ref, cell.F)
	}
	return 0, nil
}

func (ctx *evalContext) collectRange(start, end string) []float64 {
	refs := expandRange(start, end)
	var vals []float64
	for _, ref := range refs {
		cell, ok := ctx.cells[ref]
		if !ok {
			continue
		}
		if cell.V != nil {
			if _, isStr := cell.V.(string); isStr {
				continue
			}
		}
		v, err := ctx.getCellValue(ref)
		if err == nil {
			vals = append(vals, v)
		}
	}
	return vals
}

func (ctx *evalContext) evalArg(arg expr) ([]float64, error) {
	if r, ok := arg.(*rangeExpr); ok {
		return ctx.collectRange(r.start, r.end), nil
	}
	val, err := arg.eval(ctx)
	if err != nil {
		return nil, err
	}
	return []float64{val}, nil
}

// ---------------------------------------------------------------------------
// Function implementations
// ---------------------------------------------------------------------------

func evalFuncSum(ctx *evalContext, args []expr) (float64, error) {
	var total float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		for _, v := range vals {
			total += v
		}
	}
	return total, nil
}

func evalFuncAverage(ctx *evalContext, args []expr) (float64, error) {
	var total float64
	var count float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		for _, v := range vals {
			total += v
			count++
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("AVERAGE of empty range")
	}
	return total / count, nil
}

// cellHasNumericValue checks whether a cell contains a numeric value or a
// formula that can be evaluated to a number. String-valued cells are excluded.
func (ctx *evalContext) cellHasNumericValue(ref string) bool {
	cell, ok := ctx.cells[normalizeCellRef(ref)]
	if !ok {
		return false
	}
	if cell.V != nil {
		if _, isStr := cell.V.(string); isStr {
			return false
		}
	}
	if _, err := ctx.getCellValue(ref); err == nil {
		return true
	}
	return false
}

func evalFuncCount(ctx *evalContext, args []expr) (float64, error) {
	var count float64
	for _, arg := range args {
		switch a := arg.(type) {
		case *rangeExpr:
			refs := expandRange(a.start, a.end)
			for _, ref := range refs {
				if ctx.cellHasNumericValue(ref) {
					count++
				}
			}
		case *cellRefExpr:
			if ctx.cellHasNumericValue(a.ref) {
				count++
			}
		default:
			if _, err := arg.eval(ctx); err == nil {
				count++
			}
		}
	}
	return count, nil
}

func evalFuncMin(ctx *evalContext, args []expr) (float64, error) {
	var minVal *float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		for _, v := range vals {
			if minVal == nil || v < *minVal {
				minVal = &v
			}
		}
	}
	if minVal == nil {
		return 0, nil
	}
	return *minVal, nil
}

func evalFuncMax(ctx *evalContext, args []expr) (float64, error) {
	var maxVal *float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		for _, v := range vals {
			if maxVal == nil || v > *maxVal {
				maxVal = &v
			}
		}
	}
	if maxVal == nil {
		return 0, nil
	}
	return *maxVal, nil
}

func evalFuncAbs(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("ABS requires exactly 1 argument")
	}
	val, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	return math.Abs(val), nil
}

func evalFuncRound(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("ROUND requires exactly 2 arguments")
	}
	val, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	digitsRaw, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	digits := int(digitsRaw)
	pow := math.Pow(10, float64(digits))
	return math.Round(val*pow) / pow, nil
}

func evalFuncProduct(ctx *evalContext, args []expr) (float64, error) {
	if len(args) == 0 {
		return 0, nil
	}
	total := 1.0
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		for _, v := range vals {
			total *= v
		}
	}
	return total, nil
}

func evalFuncRoundup(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("ROUNDUP requires exactly 2 arguments")
	}
	val, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	digitsRaw, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	digits := int(digitsRaw)
	pow := math.Pow(10, float64(digits))
	if val >= 0 {
		return math.Ceil(val*pow) / pow, nil
	}
	return math.Floor(val*pow) / pow, nil
}

func evalFuncRounddown(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("ROUNDDOWN requires exactly 2 arguments")
	}
	val, err := args[0].eval(ctx)
	if err != nil {
		return 0, err
	}
	digitsRaw, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	digits := int(digitsRaw)
	pow := math.Pow(10, float64(digits))
	if val >= 0 {
		return math.Floor(val*pow) / pow, nil
	}
	return math.Ceil(val*pow) / pow, nil
}

func evalFuncSumproduct(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 {
		return 0, fmt.Errorf("SUMPRODUCT requires at least 2 arguments")
	}
	var arrays [][]float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		arrays = append(arrays, vals)
	}
	minLen := len(arrays[0])
	for _, a := range arrays[1:] {
		if len(a) < minLen {
			minLen = len(a)
		}
	}
	var total float64
	for i := 0; i < minLen; i++ {
		prod := 1.0
		for _, a := range arrays {
			prod *= a[i]
		}
		total += prod
	}
	return total, nil
}

func evalFuncMedian(ctx *evalContext, args []expr) (float64, error) {
	var all []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	if len(all) == 0 {
		return 0, fmt.Errorf("MEDIAN of empty set")
	}
	sort.Float64s(all)
	n := len(all)
	if n%2 == 1 {
		return all[n/2], nil
	}
	return (all[n/2-1] + all[n/2]) / 2, nil
}

func evalFuncStdev(ctx *evalContext, args []expr, population bool) (float64, error) {
	var all []float64
	for _, arg := range args {
		vals, err := ctx.evalArg(arg)
		if err != nil {
			return 0, err
		}
		all = append(all, vals...)
	}
	n := len(all)
	if n == 0 {
		return 0, fmt.Errorf("STDEV of empty set")
	}
	if !population && n < 2 {
		return 0, fmt.Errorf("STDEV.S requires at least 2 values")
	}
	var sum float64
	for _, v := range all {
		sum += v
	}
	mean := sum / float64(n)
	var sqDiff float64
	for _, v := range all {
		d := v - mean
		sqDiff += d * d
	}
	divisor := float64(n)
	if !population {
		divisor = float64(n - 1)
	}
	return math.Sqrt(sqDiff / divisor), nil
}

func evalFuncStdevS(ctx *evalContext, args []expr) (float64, error) {
	return evalFuncStdev(ctx, args, false)
}

func evalFuncStdevP(ctx *evalContext, args []expr) (float64, error) {
	return evalFuncStdev(ctx, args, true)
}

func evalFuncSumif(ctx *evalContext, args []expr) (float64, error) {
	if len(args) < 2 || len(args) > 3 {
		return 0, fmt.Errorf("SUMIF requires 2 or 3 arguments")
	}
	checkRange, err := rangeOrCellRefs(ctx, args[0])
	if err != nil {
		return 0, fmt.Errorf("SUMIF first argument: %w", err)
	}
	criteriaVal, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	var sumRefs []string
	if len(args) == 3 {
		sumRefs, err = rangeOrCellRefs(ctx, args[2])
		if err != nil {
			return 0, fmt.Errorf("SUMIF third argument: %w", err)
		}
	} else {
		sumRefs = checkRange
	}
	limit := len(checkRange)
	if len(sumRefs) < limit {
		limit = len(sumRefs)
	}
	var total float64
	for i := 0; i < limit; i++ {
		cellVal, err := ctx.getCellValue(checkRange[i])
		if err != nil {
			continue
		}
		if cellVal == criteriaVal {
			sumVal, err := ctx.getCellValue(sumRefs[i])
			if err == nil {
				total += sumVal
			}
		}
	}
	return total, nil
}

func evalFuncCountif(ctx *evalContext, args []expr) (float64, error) {
	if len(args) != 2 {
		return 0, fmt.Errorf("COUNTIF requires exactly 2 arguments")
	}
	refs, err := rangeOrCellRefs(ctx, args[0])
	if err != nil {
		return 0, fmt.Errorf("COUNTIF first argument: %w", err)
	}
	criteriaVal, err := args[1].eval(ctx)
	if err != nil {
		return 0, err
	}
	var count float64
	for _, ref := range refs {
		cellVal, err := ctx.getCellValue(ref)
		if err != nil {
			continue
		}
		if cellVal == criteriaVal {
			count++
		}
	}
	return count, nil
}

// rangeOrCellRefs extracts a list of cell references from an expression.
// Accepts *rangeExpr (expanded) or *cellRefExpr (single element).
func rangeOrCellRefs(ctx *evalContext, arg expr) ([]string, error) {
	switch a := arg.(type) {
	case *rangeExpr:
		return expandRange(a.start, a.end), nil
	case *cellRefExpr:
		return []string{a.ref}, nil
	default:
		return nil, fmt.Errorf("expected a cell reference or range")
	}
}

func evalFuncRow(ctx *evalContext, args []expr) (float64, error) {
	if len(args) > 1 {
		return 0, fmt.Errorf("ROW requires 0 or 1 argument")
	}
	if len(args) == 0 {
		if ctx.formulaRef == "" {
			return 0, nil
		}
		_, r := parseCellRef(ctx.formulaRef)
		return float64(r), nil
	}
	ref, ok := args[0].(*cellRefExpr)
	if !ok {
		return 0, fmt.Errorf("ROW argument must be a cell reference")
	}
	_, r := parseCellRef(ref.ref)
	return float64(r), nil
}

func evalFuncColumn(ctx *evalContext, args []expr) (float64, error) {
	if len(args) > 1 {
		return 0, fmt.Errorf("COLUMN requires 0 or 1 argument")
	}
	if len(args) == 0 {
		if ctx.formulaRef == "" {
			return 0, nil
		}
		c, _ := parseCellRef(ctx.formulaRef)
		return float64(c), nil
	}
	ref, ok := args[0].(*cellRefExpr)
	if !ok {
		return 0, fmt.Errorf("COLUMN argument must be a cell reference")
	}
	c, _ := parseCellRef(ref.ref)
	return float64(c), nil
}

// ---------------------------------------------------------------------------
// Cell reference utilities
// ---------------------------------------------------------------------------

// normalizeCellRef uppercases the column part of a cell reference.
func normalizeCellRef(ref string) string {
	i := 0
	for i < len(ref) && ((ref[i] >= 'A' && ref[i] <= 'Z') || (ref[i] >= 'a' && ref[i] <= 'z')) {
		i++
	}
	if i == 0 {
		return ref
	}
	return strings.ToUpper(ref[:i]) + ref[i:]
}

// parseCellRef converts "A1" to (1, 1).
func parseCellRef(ref string) (col, row int) {
	i := 0
	for i < len(ref) && ((ref[i] >= 'A' && ref[i] <= 'Z') || (ref[i] >= 'a' && ref[i] <= 'z')) {
		i++
	}
	col = 0
	for _, ch := range strings.ToUpper(ref[:i]) {
		col = col*26 + int(ch-'A'+1)
	}
	row, _ = strconv.Atoi(ref[i:])
	return
}

// formatCellRef converts (1, 1) to "A1".
func formatCellRef(col, row int) string {
	var prefix string
	for col > 0 {
		col--
		prefix = string(rune('A'+col%26)) + prefix
		col /= 26
	}
	return prefix + strconv.Itoa(row)
}

// expandRange returns all cell references between start and end (inclusive).
func expandRange(start, end string) []string {
	c1, r1 := parseCellRef(start)
	c2, r2 := parseCellRef(end)

	minCol := c1
	maxCol := c2
	if c1 > c2 {
		minCol, maxCol = c2, c1
	}
	minRow := r1
	maxRow := r2
	if r1 > r2 {
		minRow, maxRow = r2, r1
	}

	var refs []string
	for c := minCol; c <= maxCol; c++ {
		for r := minRow; r <= maxRow; r++ {
			refs = append(refs, formatCellRef(c, r))
		}
	}
	return refs
}
