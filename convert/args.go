package convert

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/code-instable/latex2utf8/symbols"
)

// command converts name{...} (or sqrt[n]{...}) where s[k] is the character
// following the name. It reports the index after the command, or false if
// name is not a command or its arguments are missing, in which case nothing
// has been written and the caller treats name as an ordinary symbol.
func (c *converter) command(name, s string, k int) (end int, ok bool) {
	cmd, found := c.set.Command(name)
	if !found || c.depth >= maxDepth {
		return 0, false
	}
	switch cmd.Kind {
	case symbols.Frac:
		num, p, ok := braceGroup(s, k)
		if !ok {
			return 0, false
		}
		den, end, ok := braceGroup(s, skipSpace(s, p))
		if !ok {
			return 0, false
		}
		c.emit(c.frac(num, den), true)
		return end, true
	case symbols.Sqrt:
		index, p := "", k
		if s[p] == '[' {
			n := strings.IndexByte(s[p:], ']')
			if n < 0 {
				return 0, false
			}
			index, p = s[p+1:p+n], p+n+1
		}
		arg, end, ok := braceGroup(s, p)
		if !ok || blank(arg) {
			return 0, false
		}
		c.emit(c.sqrt(index, arg), true)
		return end, true
	}
	arg, end, ok := braceGroup(s, k)
	// A blank argument makes no accent, alphabet or root, so the name is just a symbol
	// followed by an empty group. That matters for names that are both: u is the
	// symbol ˘ as well as the breve accent \u{a}, and u{} must stay ˘.
	if !ok || (blank(arg) && cmd.Kind != symbols.Text) {
		return 0, false
	}
	switch cmd.Kind {
	case symbols.Accent:
		c.emit(c.accent(c.fragment(arg), cmd.Mark, false), true)
	case symbols.WideAccent:
		c.emit(c.accent(c.fragment(arg), cmd.Mark, true), true)
	case symbols.Font:
		out := c.font(cmd.Prefix, arg)
		if c.opt.Join {
			out = strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) {
					return -1
				}
				return r
			}, out)
		}
		c.emit(out, true)
	case symbols.Text:
		c.emit(arg, false)
	}
	return end, true
}

// braceGroup returns the content of the {...} group starting at s[k] and the
// index after it. Groups nest, and \{ \} do not count.
func braceGroup(s string, k int) (content string, end int, ok bool) {
	if k >= len(s) || s[k] != '{' {
		return "", 0, false
	}
	depth := 0
	for i := k; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return s[k+1 : i], i + 1, true
			}
		}
	}
	return "", 0, false
}

func blank(s string) bool { return strings.TrimSpace(s) == "" }

func skipSpace(s string, i int) int {
	for i < len(s) {
		n := spaceLen(s, i)
		if n == 0 {
			break
		}
		i += n
	}
	return i
}

// accent puts the combining mark on the text: on its last character, or on every
// character if wide. A character is a letter with the combining marks it already
// has, so the new mark goes after those and never on one of them. Characters
// that exist precomposed (â, ñ, ∉) are composed.
func (c *converter) accent(text string, mark rune, wide bool) string {
	if text == "" {
		return string(mark)
	}
	apply := func(cluster []rune) string {
		if len(cluster) == 1 { // only a bare letter can have a precomposed form
			if composed, ok := c.set.Compose(cluster[0], mark); ok {
				return string(composed)
			}
		}
		return string(cluster) + string(mark)
	}
	clusters := characters(text)
	var b strings.Builder
	if wide {
		for _, cl := range clusters {
			if len(cl) == 1 && unicode.IsSpace(cl[0]) {
				b.WriteString(string(cl))
			} else {
				b.WriteString(apply(cl))
			}
		}
		return b.String()
	}
	last := len(clusters) - 1
	for _, cl := range clusters[:last] {
		b.WriteString(string(cl))
	}
	b.WriteString(apply(clusters[last]))
	return b.String()
}

// characters splits text into runes with the combining marks that follow each of them.
func characters(text string) [][]rune {
	var out [][]rune
	for _, r := range text {
		if n := len(out); n > 0 && unicode.IsMark(r) {
			out[n-1] = append(out[n-1], r)
		} else {
			out = append(out, []rune{r})
		}
	}
	return out
}

var digitNames = [...]string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}

// font maps the letters, digits and names (alpha, \Gamma) of text to the math
// alphabet with the given name prefix. Characters without a styled form are kept.
func (c *converter) font(prefix, text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		ch := text[i]
		if ch == '\\' && i+1 < len(text) && isLetter(text[i+1]) {
			i++
			ch = text[i]
		}
		switch {
		case isLetter(ch):
			j := i
			for j < len(text) && isLetter(text[j]) {
				j++
			}
			if v, ok := c.set.Lookup(prefix + text[i:j]); ok {
				b.WriteString(v)
			} else {
				for _, l := range text[i:j] {
					if v, ok := c.set.Lookup(prefix + string(l)); ok {
						b.WriteString(v)
					} else {
						b.WriteRune(l)
					}
				}
			}
			i = j
		case isDigit(ch):
			if v, ok := c.set.Lookup(prefix + digitNames[ch-'0']); ok {
				b.WriteString(v)
			} else {
				b.WriteByte(ch)
			}
			i++
		default:
			_, size := utf8.DecodeRuneInString(text[i:])
			b.WriteString(text[i : i+size])
			i += size
		}
	}
	return b.String()
}

const fractionSlash = "⁄"

// frac builds a fraction: a vulgar fraction (½) when there is one, otherwise a
// superscript numerator and subscript denominator around the fraction slash
// (¹²⁄₃₅), otherwise plain num/den.
func (c *converter) frac(num, den string) string {
	num, den = strings.TrimSpace(num), strings.TrimSpace(den)
	if v, ok := c.set.Fraction(num, den); ok {
		return v
	}
	n, d := c.fragment(num), c.fragment(den)
	if sup, ok := c.scriptText('^', n); ok {
		if sub, ok := c.scriptText('_', d); ok {
			return sup + fractionSlash + sub
		}
	}
	return group(n) + "/" + group(d)
}

// sqrt builds a root: √x, ∛x, ∜x or ⁿ√x.
func (c *converter) sqrt(index, arg string) string {
	inner := c.fragment(arg)
	if runeCount(inner) > 1 && !allDigits(inner) {
		inner = "(" + inner + ")"
	}
	root := "√"
	switch idx := strings.TrimSpace(index); idx {
	case "", "2":
	case "3":
		root = "∛"
	case "4":
		root = "∜"
	default:
		if sup, ok := c.scriptText('^', c.fragment(idx)); ok {
			root = sup + "√"
		} else {
			root = "(" + c.fragment(idx) + ")√"
		}
	}
	return root + inner
}

// group wraps multi rune text in parentheses.
func group(s string) string {
	if runeCount(s) > 1 {
		return "(" + s + ")"
	}
	return s
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return s != ""
}
