package style

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Functions MatchQuery and parseQuery ported from
// https://github.com/ericf/css-mediaquery
// originally released as
// Copyright (c) 2014, Yahoo! Inc. All rights reserved.
// Copyrights licensed under the New BSD License.

var (
	reExpressions    = regexp.MustCompile(`\([^\)]+\)`)
	reMediaQuery     = regexp.MustCompile(`^(?:(only|not)?\s*([_a-z][_a-z0-9-]*)|(\([^\)]+\)))(?:\s*and\s*(.*))?$`) // TODO: /i,
	reMqExpression   = regexp.MustCompile(`^\(\s*([_a-z-][_a-z0-9-]*)\s*(?:\:\s*([^\)]+))?\s*\)$`)
	reMqFeature      = regexp.MustCompile(`^(?:(min|max)-)?(.+)`)
	reLengthUnit     = regexp.MustCompile(`(em|rem|px|cm|mm|in|pt|pc)?\s*$`)
	reResolutionUnit = regexp.MustCompile(`(dpi|dpcm|dppx)?\s*$`)
)

type MediaQuery struct {
	inverse bool
	typ     string
	exprs   []MediaExpr
}

type MediaExpr struct {
	modifier string
	feature  string
	value    string
}

func MatchQuery(mediaQuery string, values map[string]string) (yes bool, err error) {
	qs, err := parseQuery(mediaQuery)
	if err != nil {
		return false, fmt.Errorf("parse query: %v", err)
	}
	for _, q := range qs {
		inverse := q.inverse
		typeMatch := q.typ == "all" || values["type"] == q.typ
		if (typeMatch && inverse) || !(typeMatch || inverse) {
			continue
		}

		every := true
		for _, expr := range q.exprs {
			var valueFloat float64
			var expValueFloat float64
			feature := expr.feature
			modifier := expr.modifier
			expValue := expr.value
			value := values[feature]

			if value == "" {
				every = false
				break
			}
			switch feature {
			case "orientation", "scan", "prefers-color-scheme":
				if strings.ToLower(value) != strings.ToLower(expValue) {
					every = false
					break
				}
			case "width", "height", "device-width", "device-height":
				if expValueFloat, err = toPx(expValue); err != nil {
					break
				}
				if valueFloat, err = toPx(value); err != nil {
					break
				}
			case "resolution":
				if expValueFloat, err = toDpi(expValue); err != nil {
					break
				}
				if valueFloat, err = toDpi(value); err != nil {
					break
				}
			case "aspect-ratio", "device-aspect-ratio", /* Deprecated */ "device-pixel-ratio":
				if expValueFloat, err = toDecimal(expValue); err != nil {
					break
				}
				if valueFloat, err = toDecimal(value); err != nil {
					break
				}
			case "grid", "color", "color-index", "monochrome":
				var i int64
				i, err = strconv.ParseInt(expValue, 10, 64)
				if err != nil {
					i = 1
					err = nil
				}
				expValueFloat = float64(i)
				i, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					i = 0
					err = nil
				}
				valueFloat = float64(i)
			}
			switch modifier {
			case "min":
				every = valueFloat >= expValueFloat
			case "max":
				every = valueFloat <= expValueFloat
			default:
				every = valueFloat == expValueFloat
			}
		}
		if (every && !inverse) || (!every && inverse) {
			return true, nil
		}
	}
	return false, nil
}

func parseQuery(mediaQuery string) (tokens []MediaQuery, err error) {
	parts := strings.Split(mediaQuery, ",")
	for _, q := range parts {
		q = strings.TrimSpace(q)
		captures := reMediaQuery.FindStringSubmatch(q)

		if captures == nil {
			return tokens, fmt.Errorf("Invalid CSS media query: %v", q)
		}
		modifier := captures[1]
		parsed := MediaQuery{}
		parsed.inverse = strings.ToLower(modifier) == "not"
		if typ := captures[2]; typ == "" {
			parsed.typ = "all"
		} else {
			parsed.typ = strings.ToLower(typ)
		}

		var exprs string
		if len(captures) >= 4 {
			exprs += captures[3]
		}
		if len(captures) >= 5 {
			exprs += captures[4]
		}
		exprs = strings.TrimSpace(exprs)
		if exprs == "" {
			tokens = append(tokens, parsed)
			continue
		}

		exprsList := reExpressions.FindStringSubmatch(exprs)
		if exprsList == nil {
			return tokens, fmt.Errorf("Invalid CSS media query: %v", q)
		}
		for _, expr := range exprsList {
			var captures = reMqExpression.FindStringSubmatch(expr)

			if captures == nil {
				return tokens, fmt.Errorf("Invalid CSS media query: %v", q)
			}
			feature := reMqFeature.FindStringSubmatch(strings.ToLower(captures[1]))
			parsed.exprs = append(parsed.exprs, MediaExpr{
				modifier: feature[1],
				feature:  feature[2],
				value:    captures[2],
			})
		}
		tokens = append(tokens, parsed)
	}
	return
}

// -- Utilities ----------------------------------------------------------------

var reQuot = regexp.MustCompile(`^(\d+)\s*\/\s*(\d+)$`)

func toDecimal(ratio string) (decimal float64, err error) {
	decimal, err = strconv.ParseFloat(ratio, 64)
	if err != nil {
		numbers := reQuot.FindStringSubmatch(ratio)
		if numbers == nil {
			return 0, fmt.Errorf("cannot parse %v", ratio)
		}
		p, err := strconv.ParseFloat(numbers[0], 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %v", p)
		}
		q, err := strconv.ParseFloat(numbers[1], 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %v", q)
		}
		if q == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		decimal = p / q
	}
	return
}

func toDpi(resolution string) (value float64, err error) {
	if value, err = strconv.ParseFloat(resolution, 64); err != nil {
		return
	}
	units := reResolutionUnit.FindStringSubmatch(resolution)[1]

	switch units {
	case "dpcm":
		value /= 2.54
	case "dppx":
		value *= 96
	}
	return
}

func toPx(length string) (value float64, err error) {
	units := reLengthUnit.FindStringSubmatch(length)[1]
	length = length[:len(length)-len(units)]
	if value, err = strconv.ParseFloat(length, 64); err != nil {
		return
	}

	switch units {
	case "em":
		value *= 16
	case "rem":
		value *= 16
	case "cm":
		value *= 96 / 2.54
	case "mm":
		value *= 96 / 2.54 / 10
	case "in":
		value *= 96
	case "pt":
		value *= 72
	case "pc":
		value *= 72 / 12
	}
	return
}
