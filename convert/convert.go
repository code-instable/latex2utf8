// Package convert turns LaTeX-style input into Unicode text.
//
// The input is scanned once, left to right. What a piece of text means
// depends on how it is written:
//
//	gamma, \gamma      a symbol name; the backslash is optional
//	\L, \o             one letter names need the backslash
//	x_1  x^{2n}  _n    subscripts and superscripts
//	frac{1}{2}         argument commands: frac, sqrt, hat, bar, mathbb, text, ...
//	\"{o}  \'e         the LaTeX text accents
//	alpha{}x           an empty {} after a name ends it: αx, not "α x"
//
// Anything that is not recognised is copied through unchanged, so the same
// input always gives the same output.
package convert

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/code-instable/latex2utf8/symbols"
)

// Mode selects how much of the input is treated as symbol names.
type Mode int

const (
	// Symbols treats every bare word as a possible symbol name: the input is
	// mostly names, as when typing into an editor. This is the default.
	Symbols Mode = iota
	// Text converts only explicit syntax (\name, \name{...}, braced scripts
	// and ^digits) and leaves ordinary words alone, so prose can be passed through.
	Text
)

// Options configure Convert. The zero value is the default behaviour with
// the embedded symbol set, except that Fuzzy must be switched on explicitly.
type Options struct {
	Mode Mode
	// Fuzzy corrects typos in bare words that match no name (see Set.Fuzzy).
	Fuzzy bool
	// Strict accepts exact names only: no case folding, splitting or typo correction.
	Strict bool
	// Join drops all whitespace from the output. By default whitespace is
	// dropped only between two converted pieces, so "alpha beta" gives "αβ"
	// while "x = alpha" keeps its spaces.
	Join bool
	// Set is the symbol database; nil means symbols.Default().
	Set *symbols.Set
}

// Problem is a name that could not be resolved.
type Problem struct {
	Name      string
	Backslash bool // written as \name, so it was meant as a command
}

// Result is the outcome of a conversion.
type Result struct {
	Output   string
	Problems []Problem // unresolved names, in order of first appearance
}

// maxDepth bounds the nesting of argument commands, which are converted recursively.
const maxDepth = 16

// Convert converts input according to opt.
func Convert(input string, opt Options) Result {
	if opt.Set == nil {
		opt.Set = symbols.Default()
	}
	c := &converter{opt: opt, set: opt.Set, probs: &problems{seen: map[Problem]bool{}}}
	c.run(input)
	c.flushSpace()
	return Result{Output: c.out.String(), Problems: c.probs.list}
}

type problems struct {
	list []Problem
	seen map[Problem]bool
}

type converter struct {
	opt   Options
	set   *symbols.Set
	out   strings.Builder
	space string // whitespace seen since the last piece, not written yet
	conv  bool   // whether the last piece written was converted
	depth int
	probs *problems
}

func (c *converter) run(s string) {
	for i := 0; i < len(s); {
		switch ch := s[i]; {
		case isSpace(ch):
			j := i + 1
			for j < len(s) && isSpace(s[j]) {
				j++
			}
			c.space += s[i:j]
			i = j
		case ch == '\\':
			i = c.backslash(s, i)
		case ch == '^' || ch == '_':
			i = c.script(s, i)
		case isLetter(ch) || startsWord(s, i):
			i = c.word(s, i)
		case ch == '{' && strings.HasPrefix(s[i:], "{}") && c.conv && c.space == "":
			// An empty group straight after a converted piece ends it and writes nothing,
			// as in LaTeX: alpha{}x is αx, where "alpha x" would keep the space.
			i += 2
		default:
			j := i + 1
			for j < len(s) && !isSpace(s[j]) && !isLetter(s[j]) && s[j] != '\\' && s[j] != '^' && s[j] != '_' && !startsWord(s, j) {
				j++
			}
			c.emit(s[i:j], false)
			i = j
		}
	}
}

// emit writes a piece of output. Whitespace waiting before it is written too,
// unless it separates two converted pieces on the same line.
func (c *converter) emit(text string, converted bool) {
	if text == "" {
		return
	}
	c.writeSpace(converted)
	c.out.WriteString(text)
	c.conv = converted
}

func (c *converter) writeSpace(next bool) {
	if c.space == "" {
		return
	}
	if !c.opt.Join && (!c.conv || !next || strings.ContainsAny(c.space, "\r\n")) {
		c.out.WriteString(c.space)
	}
	c.space = ""
}

func (c *converter) flushSpace() { c.writeSpace(false) }

func (c *converter) problem(name string, backslash bool) {
	p := Problem{Name: name, Backslash: backslash}
	if !c.probs.seen[p] {
		c.probs.seen[p] = true
		c.probs.list = append(c.probs.list, p)
	}
}

// fragment converts the content of a {...} argument. Inside braces the input
// is always read as symbol names, whatever the mode.
func (c *converter) fragment(s string) string {
	sub := &converter{opt: c.opt, set: c.set, depth: c.depth + 1, probs: c.probs}
	sub.opt.Mode = Symbols
	sub.run(strings.TrimSpace(s))
	return sub.out.String()
}

// word handles a run of letters at s[i:], not preceded by a backslash. A run
// containing a letter that is not ASCII (naïve, αbeta) is a word of some other
// language and is copied through, so that "ve" in "naïve" is not read as a name.
func (c *converter) word(s string, i int) int {
	j, ascii := i, true
	for j < len(s) {
		if isLetter(s[j]) {
			j++
		} else if startsWord(s, j) {
			_, size := utf8.DecodeRuneInString(s[j:])
			j += size
			ascii = false
		} else {
			break
		}
	}
	if c.opt.Mode == Text || !ascii {
		c.emit(s[i:j], false)
		return j
	}
	// Some names contain digits (note8th); take them if the whole thing is a name.
	if k := alnumEnd(s, j); k > j {
		if v, ok := c.set.Lookup(s[i:k]); ok {
			c.emit(v, true)
			return k
		}
	}
	if j < len(s) && (s[j] == '{' || s[j] == '[') && j-i >= 2 {
		if end, ok := c.command(s[i:j], s, j); ok {
			return end
		}
	}
	c.bare(s[i:j])
	return j
}

// bare resolves a word written without a backslash.
func (c *converter) bare(w string) {
	if len(w) < 2 { // one letter names are too common in text; they need a backslash
		c.emit(w, false)
		return
	}
	if v, ok := c.set.Lookup(w); ok {
		c.emit(v, true)
		return
	}
	if !c.opt.Strict {
		// A slip of case (bbbR, GAMMA) is repaired, but a word typed entirely in
		// lower case must match exactly: "and" is a word, not the symbol And.
		if len(w) >= 3 && hasUpper(w) {
			if _, v, ok := c.set.Fold(w); ok {
				c.emit(v, true)
				return
			}
		}
		if v, ok := c.set.Segment(w); ok {
			c.emit(v, true)
			return
		}
		if c.opt.Fuzzy {
			if n := maxEdits(len(w)); n > 0 {
				if m, _, ok := c.set.Fuzzy(w, n); ok {
					c.emit(m.Value, true)
					return
				}
			}
		}
	}
	c.problem(w, false)
	c.emit(w, false)
}

// maxEdits is how many typos are tolerated in a word of n letters.
func maxEdits(n int) int {
	switch {
	case n < 4:
		return 0
	case n < 8:
		return 1
	}
	return 2
}

// textAccents are the LaTeX accent commands made of a single punctuation mark.
var textAccents = map[byte]rune{
	'`': 0x300, '\'': 0x301, '^': 0x302, '~': 0x303, '=': 0x304, '.': 0x307, '"': 0x308,
}

// backslash handles a command starting at s[i] == '\\'.
func (c *converter) backslash(s string, i int) int {
	j := i + 1
	if j >= len(s) {
		c.emit(`\`, false)
		return j
	}
	switch ch := s[j]; {
	case isLetter(ch):
		k := j
		for k < len(s) && isLetter(s[k]) {
			k++
		}
		if e := alnumEnd(s, k); e > k {
			if v, ok := c.set.Lookup(s[j:e]); ok {
				c.emit(v, true)
				return e
			}
		}
		if k < len(s) && (s[k] == '{' || s[k] == '[') {
			if end, ok := c.command(s[j:k], s, k); ok {
				return end
			}
		}
		if v, ok := c.set.Lookup(s[j:k]); ok {
			c.emit(v, true)
			return k
		}
		c.problem(s[j:k], true)
		c.emit(s[i:k], false)
		return k
	case ch == '\\':
		c.emit(`\\`, false)
		return j + 1
	case textAccents[ch] != 0:
		return c.textAccent(s, i)
	case strings.IndexByte("{}$%&#_", ch) >= 0: // escaped special characters
		c.emit(s[j:j+1], false)
		return j + 1
	}
	c.emit(`\`, false)
	return j
}

// textAccent handles \" \' \^ \~ \= \. and \` followed by {x} or a letter.
func (c *converter) textAccent(s string, i int) int {
	mark := textAccents[s[i+1]]
	j := i + 2
	var arg string
	var end int
	switch {
	case j < len(s) && s[j] == '{':
		content, e, ok := braceGroup(s, j)
		if !ok {
			c.emit(`\`, false)
			return i + 1
		}
		arg, end = c.fragment(content), e
	case j < len(s) && isLetter(s[j]):
		arg, end = s[j:j+1], j+1
	default:
		c.emit(`\`, false)
		return i + 1
	}
	c.emit(c.accent(arg, mark, false), true)
	return end
}

// startsWord reports whether s[i:] begins with a non-ASCII letter or combining mark.
func startsWord(s string, i int) bool {
	if s[i] < utf8.RuneSelf {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return unicode.IsLetter(r) || unicode.IsMark(r)
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func hasUpper(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}
	return false
}

// alnumEnd returns the end of the run of ASCII letters and digits starting at s[i:].
func alnumEnd(s string, i int) int {
	for i < len(s) && (isLetter(s[i]) || isDigit(s[i])) {
		i++
	}
	return i
}

func runeCount(s string) int { return utf8.RuneCountInString(s) }
