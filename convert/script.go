package convert

import (
	"strings"
	"unicode/utf8"
)

// script handles ^ and _ at s[i]. The argument is a braced group, a name
// (_beta, ^uparrow), a single letter, a number with an optional sign (^-12),
// or one of the characters ( ) = + - ! < >. Unlike LaTeX, an unbraced number
// is taken whole: _12 is ₁₂, not ₁ followed by 2. If any character has no
// Unicode script form the ^ or _ is left as typed.
//
// In Text mode only braced groups and (for ^) numbers are converted, so that
// snake_case and file_1.txt survive.
func (c *converter) script(s string, i int) int {
	kind := s[i]
	keep := func() int {
		c.emit(s[i:i+1], false)
		return i + 1
	}
	j := i + 1
	if j >= len(s) {
		return keep()
	}
	text := c.opt.Mode == Text
	ch := s[j]
	switch {
	case ch == '{':
		if content, end, ok := braceGroup(s, j); ok {
			if v, ok := c.scriptText(kind, content); ok {
				c.emit(v, true)
				return end
			}
		}
		return keep()
	case text && kind == '_':
		return keep()
	case isLetter(ch):
		if text {
			return keep()
		}
		k := j
		for k < len(s) && isLetter(s[k]) {
			k++
		}
		if v, ok := c.set.Lookup(string(kind) + s[j:k]); ok { // ^beta, _gamma
			c.emit(v, true)
			return k
		}
		if v, ok := c.set.Lookup(string(kind) + s[j:j+1]); ok { // the first letter only
			c.emit(v, true)
			return j + 1
		}
		return keep()
	}
	// a number, signed or not
	k := j
	if (ch == '+' || ch == '-') && k+1 < len(s) && isDigit(s[k+1]) {
		k++
	}
	if isDigitAt(s, k) {
		e := k
		for isDigitAt(s, e) {
			e++
		}
		if v, ok := c.scriptText(kind, s[j:e]); ok {
			c.emit(v, true)
			return e
		}
		return keep()
	}
	if text {
		return keep()
	}
	// a single bracket, sign or comparison character
	if v, ok := c.set.Lookup(string(kind) + s[j:j+1]); ok {
		c.emit(v, true)
		return j + 1
	}
	return keep()
}

func isDigitAt(s string, i int) bool { return i < len(s) && isDigit(s[i]) }

// scriptText converts every character of text to its superscript (kind '^')
// or subscript (kind '_') form. Names such as beta may be written plainly or
// with a backslash, and spaces are ignored. It fails, writing nothing, unless
// every character can be converted.
func (c *converter) scriptText(kind byte, text string) (string, bool) {
	prefix := string(kind)
	var b strings.Builder
	for i := 0; i < len(text); {
		ch := text[i]
		switch {
		case isSpace(ch):
			i++
		case ch == '\\' || isLetter(ch):
			if ch == '\\' {
				i++
			}
			j := i
			for j < len(text) && isLetter(text[j]) {
				j++
			}
			if j == i {
				return "", false
			}
			if v, ok := c.set.Lookup(prefix + text[i:j]); ok { // a name: ^{beta}
				b.WriteString(v)
			} else {
				for _, l := range text[i:j] { // otherwise letter by letter: ^{xyz}
					v, ok := c.set.Lookup(prefix + string(l))
					if !ok {
						return "", false
					}
					b.WriteString(v)
				}
			}
			i = j
		default:
			_, size := utf8.DecodeRuneInString(text[i:])
			v, ok := c.set.Lookup(prefix + text[i:i+size])
			if !ok {
				return "", false
			}
			b.WriteString(v)
			i += size
		}
	}
	return b.String(), b.Len() > 0
}
