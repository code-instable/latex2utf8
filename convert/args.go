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
		if !ok {
			return 0, false
		}
		c.emit(c.sqrt(index, arg), true)
		return end, true
	}
	arg, end, ok := braceGroup(s, k)
	if !ok {
		return 0, false
	}
	switch cmd.Kind {
	case symbols.Accent:
		c.emit(c.accent(c.fragment(arg), cmd.Mark, false), true)
	case symbols.WideAccent:
		c.emit(c.accent(c.fragment(arg), cmd.Mark, true), true)
	case symbols.Font:
		c.emit(c.font(cmd.Prefix, arg), true)
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

func skipSpace(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

// accent puts the combining mark on the text: on its last rune, or on every
// rune if wide. Characters that exist precomposed (â, ñ, ∉) are composed.
func (c *converter) accent(text string, mark rune, wide bool) string {
	if text == "" {
		return string(mark)
	}
	apply := func(r rune) string {
		if composed, ok := c.set.Compose(r, mark); ok {
			return string(composed)
		}
		return string(r) + string(mark)
	}
	runes := []rune(text)
	var b strings.Builder
	if wide {
		for _, r := range runes {
			if unicode.IsSpace(r) {
				b.WriteRune(r)
			} else {
				b.WriteString(apply(r))
			}
		}
		return b.String()
	}
	last := len(runes) - 1
	b.WriteString(string(runes[:last]))
	b.WriteString(apply(runes[last]))
	return b.String()
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
