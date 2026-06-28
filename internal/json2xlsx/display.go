package json2xlsx

// CellDisplayValue は cell の表示文字列 (エスケープ前) を mode に従って返す．
func CellDisplayValue(cell Cell, mode MarkdownMode, hasWarning *bool) string {
	vStr := scalarToString(cell.V)
	hasV := cell.V != nil && vStr != ""
	hasF := cell.F != ""

	var raw string
	switch cell.T {
	case "f":
		switch mode {
		case MarkdownModeValue:
			if hasV {
				raw = vStr
			} else if hasF {
				raw = "=" + cell.F
				*hasWarning = true
			}
		case MarkdownModeBoth:
			if hasV && hasF {
				raw = vStr + "<br />=" + cell.F
			} else if hasF {
				raw = "=" + cell.F
				*hasWarning = true
			} else if hasV {
				raw = vStr
			}
		default: // MarkdownModeFormula
			if hasF {
				raw = "=" + cell.F
			} else if hasV {
				raw = vStr
			}
		}
	case "d":
		if hasV {
			if cell.Z != "" && isTimeOnlyFormat(cell.Z) {
				raw = formatTimeOnly(toFloat64(cell.V), cell.Z)
			} else {
				raw = dateCellToString(cell.V)
			}
		}
	default:
		if hasV {
			raw = vStr
		} else if hasF {
			if mode == MarkdownModeValue || mode == MarkdownModeBoth {
				raw = "=" + cell.F
				*hasWarning = true
			} else {
				raw = "=" + cell.F
			}
		}
	}

	return raw
}
