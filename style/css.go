package style

import (
	"bytes"
	"fmt"
	"github.com/andybalholm/cascadia"
	"github.com/psilva261/opossum"
	"github.com/psilva261/opossum/logger"
	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
	"io"
	"strings"
)

// Sheet represents a stylesheet with rules.
//
// structs inspired by now discontinued github.com/aymerick/douceur
type Sheet struct {
	Rules []Rule
}

type Rule struct {
	Prelude      string
	Selectors    []Selector
	Declarations []Declaration

	Rules []Rule
}

type Selector struct {
	Val string
}

type Declaration struct {
	Important   bool
	Specificity cascadia.Specificity
	Prop        string
	Val         string
}

func Preprocess(s string) (bs []byte, ct opossum.ContentType, imports []string, err error) {
	buf := bytes.NewBufferString("")
	l := css.NewLexer(parse.NewInputString(s))
	ct.MediaType = "text/css"
	ct.Params = make(map[string]string)
	at := ""
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			if err != io.EOF {
				err = l.Err()
			}
			break
		}
		if d := string(data); tt == css.AtKeywordToken && (d == "@charset" || d == "@import") {
			at = d
		} else if tt == css.SemicolonToken {
			at = ""
		}
		switch at {
		case "@charset":
			if tt == css.StringToken {
				ct.Params["charset"] = string(data)
			}
		case "@import":
			if tt == css.StringToken || tt == css.URLToken {
				imports = append(imports, parseUrl(string(data)))
			}
		default:
			buf.Write(data)
		}
	}
	return buf.Bytes(), ct, imports, nil
}

func parseUrl(u string) string {
	u = strings.TrimPrefix(u, "url(")
	u = strings.TrimSuffix(u, ")")
	u = strings.ReplaceAll(u, `'`, ``)
	u = strings.ReplaceAll(u, `"`, ``)
	return u
}

func Parse(str string, inline bool) (s Sheet, err error) {
	s.Rules = make([]Rule, 0, 1000)
	stack := make([]Rule, 0, 2)
	selectors := make([]Selector, 0, 1)
	bs, ct, imports, err := Preprocess(str)
	if err != nil {
		return s, fmt.Errorf("preprocess: %v", err)
	}
	for _, imp := range imports {
		log.Infof("skipping import %v", imp)
	}
	p := css.NewParser(parse.NewInputString(ct.Utf8(bs)), inline)
	if inline {
		stack = append(stack, Rule{})
		defer func() {
			s.Rules = append(s.Rules, stack[0])
		}()
	}
	for {
		gt, _, data := p.Next()
		switch gt {
		case css.ErrorGrammar:
			if err := p.Err(); err == io.EOF {
				return s, nil
			} else {
				return s, fmt.Errorf("next: %v", err)
			}
		case css.QualifiedRuleGrammar:
			sel := Selector{}
			for _, val := range p.Values() {
				sel.Val += string(val.Data)
			}
			selectors = append(selectors, sel)
		case css.AtRuleGrammar, css.BeginAtRuleGrammar, css.BeginRulesetGrammar, css.DeclarationGrammar, css.CustomPropertyGrammar:
			var d Declaration
			if gt == css.BeginRulesetGrammar || gt == css.BeginAtRuleGrammar || gt == css.AtRuleGrammar {
				// TODO: why also gt == css.AtRuleGrammar? some sites crash otherwise
				stack = append(stack, Rule{})
			}
			r := &(stack[len(stack)-1])
			if gt == css.DeclarationGrammar || gt == css.CustomPropertyGrammar {
				d.Prop = string(data)
			}
			if gt == css.BeginAtRuleGrammar {
				r.Prelude = string(data)
			}
			vals := p.Values()
			for i, val := range vals {
				if gt == css.DeclarationGrammar || gt == css.CustomPropertyGrammar {
					if string(val.Data) == "!" && len(vals) == i+2 && string(vals[i+1].Data) == "important" {
						d.Important = true
						break
					} else {
						d.Val += string(val.Data)
					}
				} else if gt == css.BeginRulesetGrammar {
					if len(selectors) == 0 {
						sel := Selector{
							Val: string(val.Data),
						}
						selectors = append(selectors, sel)
					} else {
						selectors[len(selectors)-1].Val += string(val.Data)
					}
				} else if gt == css.BeginAtRuleGrammar {
					r.Prelude += string(val.Data)
				} else {
				}
			}
			if gt == css.DeclarationGrammar || gt == css.CustomPropertyGrammar {
				d.Val = strings.TrimSpace(d.Val)
				r.Declarations = append(r.Declarations, d)
			}
		case css.EndRulesetGrammar, css.EndAtRuleGrammar:
			var r Rule
			if len(stack) == 1 {
				r, stack = stack[len(stack)-1], stack[:len(stack)-1]
				r.Selectors = append([]Selector{}, selectors...)
				s.Rules = append(s.Rules, r)
			} else {
				p := &(stack[len(stack)-2])
				r, stack = stack[len(stack)-1], stack[:len(stack)-1]
				r.Selectors = append([]Selector{}, selectors...)
				p.Rules = append(p.Rules, r)
			}

			selectors = make([]Selector, 0, 1)
		case css.CommentGrammar:
		default:
			log.Errorf("unknown token type %+v", gt)
		}
	}
	return
}
