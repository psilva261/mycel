package style

import (
	"9fans.net/go/draw"
	"bytes"
	"fmt"
	"github.com/andybalholm/cascadia"
	"github.com/mjl-/duit"
	"github.com/psilva261/mycel/logger"
	"golang.org/x/image/colornames"
	"golang.org/x/net/html"
	"image"
	"math"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var fontCache = make(map[string]*draw.Font)

var dui *duit.DUI
var availableFontNames []string

const FontBaseSize = 11.0

var WindowWidth = 1280
var WindowHeight = 1080

var MediaValues = map[string]string{
	"type": "screen",
	"width": fmt.Sprintf("%vpx", WindowWidth),
	"orientation": "landscape",
	"prefers-color-scheme": "dark",
}

const AddOnCSS = `
/* https://developer.mozilla.org/en-US/docs/Web/HTML/Inline_elements */
a, abbr, acronym, audio, b, bdi, bdo, big, br, button, canvas, cite, code, data, datalist, del, dfn, em, embed, i, iframe, img, input, ins, kbd, label, map, mark, meter, noscript, object, output, picture, progress, q, ruby, s, samp, script, select, slot, small, span, strong, sub, sup, svg, template, textarea, time, u, tt, var, video, wbr {
  display: inline;
}

/* non-HTML5 elements: https://www.w3schools.com/tags/ref_byfunc.asp */
font, strike, tt {
  display: inline;
}

button, textarea, input, select {
  display: inline-block;
}

/* https://developer.mozilla.org/en-US/docs/Web/HTML/Block-level_elements */
address, article, aside, blockquote, details, dialog, dd, div, dl, dt, fieldset, figcaption, figure, footer, form, h1, h2, h3, h4, h5, h6, header, hgroup, hr, li, main, nav, ol, p, pre, section, table, ul {
  display: block;
}

*[href] {
  color: blue;
  margin-right: 2px;
}
`

func Init(d *duit.DUI) {
	dui = d

	initFontserver()
}

func MergeNodeMaps(m, addOn map[*html.Node]Map) {
	for n, mp := range addOn {
		// "zero" valued Map if it doesn't exist yet
		initial := m[n]

		m[n] = initial.ApplyChildStyle(mp, true)
	}
}

func FetchNodeMap(doc *html.Node, cssText string) (m map[*html.Node]Map, err error) {
	mr, rv, err := FetchNodeRules(doc, cssText)
	if err != nil {
		return nil, fmt.Errorf("fetch rules: %w", err)
	}
	m = make(map[*html.Node]Map)
	for n, rs := range mr {
		ds := make(map[string]Declaration)
		for _, r := range rs {
			for _, d := range r.Declarations {
				if exist, ok := ds[d.Prop]; ok && smaller(d, exist) {
					continue
				}
				if strings.HasPrefix(d.Val, "var(") {
					v := strings.TrimPrefix(d.Val, "var(")
					v = strings.TrimSuffix(v, ")")
					if vv, ok := rv[v]; ok {
						d.Val = vv
					}
				}
				ds[d.Prop] = d
			}
		}
		m[n] = Map{Declarations: ds}
	}
	return
}

func smaller(d, dd Declaration) bool {
	if dd.Important {
		return true
	} else if d.Important {
		return false
	} else {
		return d.Specificity.Less(dd.Specificity)
	}
}

func compile(v string) (cs cascadia.SelectorGroup, err error) {
	return cascadia.ParseGroup(v)
}

func FetchNodeRules(doc *html.Node, cssText string) (m map[*html.Node][]Rule, rVars map[string]string, err error) {
	m = make(map[*html.Node][]Rule)
	rVars = make(map[string]string)
	s, err := Parse(cssText, false)
	if err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	processRule := func(m map[*html.Node][]Rule, r Rule) (err error) {
		for i, sel := range r.Selectors {
			if sel.Val == ":root" {
				for _, d := range r.Declarations {
					rVars[d.Prop] = d.Val
				}
			}
			csg, err := compile(sel.Val)
			if err != nil {
				log.Printf("cssSel compile %v: %v", sel.Val, err)
				continue
			}
			var cs cascadia.Sel
			if n := len(csg); n == 1 {
				cs = csg[0]
			} else {
				log.Errorf("csg len %v", n)
			}
			for _, el := range cascadia.QueryAll(doc, cs) {
				existing, ok := m[el]
				if !ok {
					existing = make([]Rule, 0, 3)
				}
				var sr Rule
				sr = r
				sr.Selectors = []Selector{r.Selectors[i]}
				for j := range sr.Declarations {
					sr.Declarations[j].Specificity[0] = cs.Specificity()[0]
					sr.Declarations[j].Specificity[1] = cs.Specificity()[1]
					sr.Declarations[j].Specificity[2] = cs.Specificity()[2]
				}
				existing = append(existing, sr)
				m[el] = existing
			}
		}
		return
	}
	for _, r := range s.Rules {
		if err := processRule(m, r); err != nil {
			return nil, nil, fmt.Errorf("process rule: %w", err)
		}

		// for media queries
		if strings.HasPrefix(r.Prelude, "@media") {
			p := strings.TrimPrefix(r.Prelude, "@media")
			p = strings.TrimSpace(p)
			yes, err := MatchQuery(p, MediaValues)
			if err != nil {
				log.Errorf("match query %v: %v", r.Prelude, err)
			} else if !yes {
				continue
			}
		}
		for _, rr := range r.Rules {
			if err := processRule(m, rr); err != nil {
				return nil, nil, fmt.Errorf("process embedded rule: %w", err)
			}
		}
	}
	return
}

type DomTree interface {
	Rect() image.Rectangle
	Parent() (p DomTree, ok bool)
	Style() Map
}

type Map struct {
	Declarations map[string]Declaration
	DomTree      `json:"-"`
}

func NewMap(n *html.Node) Map {
	s := Map{
		Declarations: make(map[string]Declaration),
	}

	for _, a := range n.Attr {
		if a.Key == "style" {
			v := strings.TrimSpace(a.Val)
			if !strings.HasSuffix(v, ";") {
				v += ";"
			}
			st, err := Parse(v, true)

			var decls []Declaration
			if len(st.Rules) > 0 {
				decls = st.Rules[0].Declarations
			}

			if err != nil {
				log.Printf("could not parse '%v'", a.Val)
				break
			}

			for _, d := range decls {
				s.Declarations[d.Prop] = d
			}
		} else if a.Key == "height" || a.Key == "width" {
			v := a.Val

			if !strings.HasSuffix(v, "%") && !strings.HasSuffix(v, "px") {
				v += "px"
			}

			s.Declarations[a.Key] = Declaration{
				Prop: a.Key,
				Val:  v,
			}
		} else if a.Key == "bgcolor" {
			s.Declarations["background-color"] = Declaration{
				Prop: "background-color",
				Val:  a.Val,
			}
		}
	}

	return s
}

func (cs Map) ApplyChildStyle(ccs Map, copyAll bool) (res Map) {
	res.Declarations = make(map[string]Declaration)

	for k, v := range cs.Declarations {
		switch k {
		// https://www.w3.org/TR/CSS21/propidx.html
		case "azimuth", "border-collapse", "border-spacing", "caption-side", "color", "cursor", "direction", "elevation", "empty-cells", "font-family", "font-size", "font-style", "font-variant", "font-weight", "font", "letter-spacing", "line-height", "list-style-image", "list-style-position", "list-style-type", "list-style", "orphans", "pitch-range", "pitch", "quotes", "richness", "speak-header", "speak-numeral", "speak-punctuation", "speak", "speech-rate", "stress", "text-align", "text-indent", "text-transform", "visibility", "voice-family", "volume", "white-space", "widows", "word-spacing":
		default:
			if !copyAll {
				continue
			}
		}
		res.Declarations[k] = v
	}
	// overwrite with higher prio child props
	for k, d := range ccs.Declarations {
		if d.Val == "inherit" {
			continue
		}
		if exist, ok := res.Declarations[k]; ok && smaller(d, exist) {
			continue
		}
		res.Declarations[k] = d
	}

	return
}

func (cs Map) Font() *draw.Font {
	fn, ok := cs.FontFilename()
	if !ok || dui == nil {
		return nil
	}
	if runtime.GOOS == "plan9" && dui.Display.HiDPI() {
		// TODO: proper hidpi handling
		return dui.Font(nil)
	}
	font, ok := fontCache[fn]
	if ok {
		return font
	}
	log.Infof("call dui.Display.OpenFont(%v)", fn)
	font, err := dui.Display.OpenFont(fn)
	if err != nil {
		log.Errorf("%v is not avail", fn)
		font = dui.Display.Font
	}
	fontCache[fn] = font

	return font
}

func (cs Map) preferedFontName(preferences []string) string {
	avails := availableFontNames

	if len(avails) == 0 {
		return preferences[0]
	}

	for len(preferences) > 0 {
		var pref string
		pref, preferences = preferences[0], preferences[1:]

		for _, avail := range avails {
			if pref == strings.TrimSuffix(avail, "/") {
				return avail
			}
		}
	}

	return avails[0]
}

func matchClosestFontSize(desired float64, available []int) (closest int) {
	for _, a := range available {
		if closest == 0 || math.Abs(float64(a)-desired) < math.Abs(float64(closest)-desired) {
			closest = a
		}
	}
	return
}

func (cs Map) FontSize() float64 {
	fs, ok := cs.Declarations["font-size"]
	if !ok || fs.Val == "" {
		return FontBaseSize
	}

	if len(fs.Val) <= 2 {
		log.Printf("error parsing font size %v", fs.Val)
		return FontBaseSize
	}
	numStr := fs.Val[0 : len(fs.Val)-2]
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		log.Printf("error parsing font size %v", fs.Val)
		return FontBaseSize
	}
	if strings.HasSuffix(fs.Val, "em") {
		f *= FontBaseSize
	}
	return f
}

// FontHeight in lowDPI pixels.
func (cs Map) FontHeight() float64 {
	return float64(cs.Font().Height) / float64(dui.Scale(1))
}

func (cs Map) Color() draw.Color {
	if d, ok := cs.Declarations["color"]; ok {
		if h, ok := colorHex(d.Val); ok {
			c := draw.Color(h)
			return c
		}
	}
	return draw.Black
}

func colorHex(propVal string) (c draw.Color, ok bool) {
	var r, g, b uint32
	var x uint32
	if strings.HasPrefix(propVal, "rgb") {
		val := propVal[3:]
		val = strings.TrimPrefix(val, "a")
		val = strings.TrimPrefix(val, "(")
		val = strings.TrimSuffix(val, ")")
		vals := strings.Split(val, ",")
		if len(vals) < 3 {
			log.Errorf("vals=%+v", vals)
			return
		}
		rr, err := strconv.ParseInt(vals[0], 10, 32)
		if err != nil {
			goto default_value
		}
		gg, err := strconv.ParseInt(vals[1], 10, 32)
		if err != nil {
			goto default_value
		}
		bb, err := strconv.ParseInt(vals[2], 10, 32)
		if err != nil {
			goto default_value
		}
		r = uint32(rr) * 256
		g = uint32(gg) * 256
		b = uint32(bb) * 256
	} else if strings.HasPrefix(propVal, "#") {
		hexColor := propVal[1:]

		if len(hexColor) == 3 {
			rr, err := strconv.ParseInt(hexColor[0:1], 16, 32)
			if err != nil {
				goto default_value
			}
			gg, err := strconv.ParseInt(hexColor[1:2], 16, 32)
			if err != nil {
				goto default_value
			}
			bb, err := strconv.ParseInt(hexColor[2:3], 16, 32)
			if err != nil {
				goto default_value
			}
			r = uint32(rr) * 256 * 0x11
			g = uint32(gg) * 256 * 0x11
			b = uint32(bb) * 256 * 0x11
		} else if len(hexColor) == 6 {
			rr, err := strconv.ParseInt(hexColor[0:2], 16, 32)
			if err != nil {
				goto default_value
			}
			gg, err := strconv.ParseInt(hexColor[2:4], 16, 32)
			if err != nil {
				goto default_value
			}
			bb, err := strconv.ParseInt(hexColor[4:6], 16, 32)
			if err != nil {
				goto default_value
			}
			r = uint32(rr) * 256
			g = uint32(gg) * 256
			b = uint32(bb) * 256
		} else {
			goto default_value
		}
	} else if propVal == "inherit" {
		// TODO: handle properly
		goto default_value
	} else {
		colorRGBA, ok := colornames.Map[propVal]
		if !ok {
			goto default_value
		}
		r, g, b, _ = colorRGBA.RGBA()
	}

	x = (r / 256) << 24
	x = x | ((g / 256) << 16)
	x = x | ((b / 256) << 8)
	x = x | 0x000000ff

	return draw.Color(uint32(x)), true
default_value:
	log.Printf("could not interpret %v", propVal)
	return 0, false
}

func (cs Map) IsInline() bool {
	propVal, ok := cs.Declarations["float"]
	if ok && propVal.Val == "left" {
		return true
	}
	propVal, ok = cs.Declarations["display"]
	if ok {
		return propVal.Val == "inline" ||
			propVal.Val == "inline-block"
	}
	return false
}

func (cs Map) IsDisplayNone() bool {
	propVal, ok := cs.Declarations["display"]
	if ok && propVal.Val == "none" {
		return true
	}
	/*propVal, ok = cs.Declarations["position"]
	if ok && propVal.Val == "fixed" {
		return true
	}*/
	propVal, ok = cs.Declarations["clip"]
	if ok && strings.ReplaceAll(propVal.Val, " ", "") == "rect(1px,1px,1px,1px)" {
		return true
	}
	propVal, ok = cs.Declarations["width"]
	if ok && propVal.Val == "1px" {
		propVal, ok = cs.Declarations["height"]
		if ok && propVal.Val == "1px" {
			return true
		}
	}
	return false
}

func (cs Map) IsFlex() bool {
	propVal, ok := cs.Declarations["display"]
	if ok {
		return propVal.Val == "flex"
	}
	return false
}

func (cs Map) IsFlexDirectionRow() bool {
	propVal, ok := cs.Declarations["flex-direction"]
	if ok {
		switch propVal.Val {
		case "row":
			return true
		case "column":
			return false
		}
	}
	return true // TODO: be more specific
}

// tlbr parses 4-tuple of top-right-bottom-left like in margin,
// margin-top, ...-right, ...-bottom, ...-left.
func (cs *Map) Tlbr(key string) (s duit.Space, err error) {
	if all, ok := cs.Declarations[key]; ok {
		parts := strings.Split(all.Val, " ")
		nums := make([]int, len(parts))
		for i, p := range parts {
			if f, _, err := length(cs, p); err == nil {
				nums[i] = int(f)
			} else {
				return s, fmt.Errorf("length: %w", err)
			}
		}
		s.Top = nums[0]
		s.Right = s.Top
		s.Bottom = s.Top
		s.Left = s.Top
		if len(nums) >= 2 {
			s.Right = nums[1]
			s.Left = s.Right
		}
		if len(nums) >= 3 {
			s.Bottom = nums[2]
		}
		if len(nums) == 4 {
			s.Left = nums[3]
		}
	}

	if t, err := cs.CssPx(key + "-top"); err == nil {
		s.Top = t
	}
	if r, err := cs.CssPx(key + "-right"); err == nil {
		s.Right = r
	}
	if b, err := cs.CssPx(key + "-bottom"); err == nil {
		s.Bottom = b
	}
	if l, err := cs.CssPx(key + "-left"); err == nil {
		s.Left = l
	}

	// Limit very rudimentary relative unit calculation
	if s.Top > 100 {
		s.Top = 0
	}
	if s.Bottom > 100 {
		s.Bottom = 0
	}

	return
}

var reBcInput = regexp.MustCompile(`^[0-9\+\*\-\/\(\)\\.\s]+$`)

func calc(cs *Map, l string) (f float64, unit string, err error) {
	if !strings.HasPrefix(l, "calc(") && !strings.HasSuffix(l, ")") {
		return 0, "", fmt.Errorf("wrong format")
	}
	l = strings.TrimPrefix(l, "calc(")
	l = strings.TrimSuffix(l, ")")
	if len(l) > 50 {
		return 0, "", fmt.Errorf("parse expression: %v", l)
	}
	l = strings.ReplaceAll(l, "px", "")
	for _, u := range []string{"rem", "em", "%", "vw", "vh"} {
		if !strings.Contains(l, u) {
			continue
		}
		r, _, err := length(cs, "1"+u)
		if err != nil {
			return 0, "", fmt.Errorf("u %v: %v", u, err)
		}
		l = strings.ReplaceAll(l, u, fmt.Sprintf("*%v", r))
	}
	if !reBcInput.MatchString(l) {
		return 0, "", fmt.Errorf("parse expression: %v", l)
	}
	cmd := exec.Command("bc")
	cmd.Stdin = strings.NewReader(l + "\n")
	var out bytes.Buffer
	var er bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &er
	if e := er.String(); e != "" {
		log.Errorf("bc: %v", e)
	}
	if err = cmd.Run(); err != nil {
		return
	}
	f, err = strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	return
}

func length(cs *Map, l string) (f float64, unit string, err error) {
	var s string

	if l == "auto" || l == "inherit" || l == "initial" || l == "0" {
		return 0, "px", nil
	}

	if strings.Contains(l, "calc") {
		return calc(cs, l)
	}

	for _, suffix := range []string{"px", "%", "rem", "em", "ex", "vw", "vh", "mm"} {
		if strings.HasSuffix(l, suffix) {
			if s = strings.TrimSuffix(l, suffix); s != "" {
				f, err = strconv.ParseFloat(s, 64)
				if err != nil {
					return 0, "", fmt.Errorf("error parsing '%v': %w", l, err)
				}
			}
			unit = suffix
			break
		}
	}

	switch unit {
	case "px":
	case "rem":
		// TODO: use font size from root element
		f *= FontBaseSize
	case "em", "ex":
		// TODO: distinguish between em and ex
		if cs == nil {
			f *= FontBaseSize
		} else {
			f *= cs.FontHeight()
		}
	case "vw":
		f *= float64(WindowWidth) / 100.0
	case "vh":
		f *= float64(WindowHeight) / 100.0
	case "%":
		if cs == nil {
			return 0.0, "%", nil
		}
		var wp int
		if p, ok := cs.DomTree.Parent(); ok {
			wp = p.Style().baseWidth()
		} else {
			log.Printf("%% unit used in root element")
		}
		f *= 0.01 * float64(wp)
	case "mm":
		dpi := 100
		if dui != nil && dui.Display != nil && dui.Display.DPI != 0 {
			dpi = dui.Display.DPI
		}
		f *= float64(dpi) / 25.4
	default:
		return f, unit, fmt.Errorf("unknown suffix: %v", l)
	}

	return
}

func (cs *Map) Height() int {
	d, ok := cs.Declarations["height"]
	if ok {
		f, _, err := length(cs, d.Val)
		if err != nil {
			log.Errorf("cannot parse height: %v", err)
		}
		return int(f)
	}
	return 0
}

func (cs Map) Width() int {
	w := cs.width()
	if w > 0 {
		if d, ok := cs.Declarations["max-width"]; ok {
			f, _, err := length(&cs, d.Val)
			if err != nil {
				log.Errorf("cannot parse width: %v", err)
			}
			if mw := int(f); 0 < mw && mw < w {
				return int(mw)
			}
		}
	}
	return w
}

func (cs Map) width() int {
	d, ok := cs.Declarations["width"]
	if ok {
		f, _, err := length(&cs, d.Val)
		if err != nil {
			log.Errorf("cannot parse width: %v", err)
		}
		if f > 0 {
			return int(f)
		}
	}
	if _, ok := cs.DomTree.Parent(); !ok {
		return WindowWidth
	}
	return 0
}

// baseWidth to calculate relative widths
func (cs Map) baseWidth() int {
	if w := cs.Width(); w != 0 {
		return w
	}
	if p, ok := cs.DomTree.Parent(); !ok {
		return WindowWidth
	} else {
		return p.Style().baseWidth()
	}
}

func (cs Map) Css(propName string) string {
	d, ok := cs.Declarations[propName]
	if !ok {
		return ""
	}
	return d.Val
}

func (cs *Map) CssPx(propName string) (l int, err error) {
	d, ok := cs.Declarations[propName]
	if !ok {
		return 0, fmt.Errorf("property doesn't exist")
	}
	f, _, err := length(cs, d.Val)
	if err != nil {
		return 0, err
	}
	l = int(f)
	return
}

func (cs Map) SetCss(k, v string) {
	cs.Declarations[k] = Declaration{
		Prop: k,
		Val:  v,
	}
}
