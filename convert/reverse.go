package convert

import (
	"strings"
	"unicode/utf8"

	"github.com/code-instable/latex2utf8/symbols"
)

// ReverseOptions configure Reverse.
type ReverseOptions struct {
	// Backslash writes \name instead of name. Names of a single letter always
	// get the backslash, because Convert would not read them otherwise.
	Backslash bool
	// Set is the symbol database; nil means symbols.Default().
	Set *symbols.Set
}

// Reverse replaces the Unicode symbols in input by their names, the inverse
// of Convert: Convert(Reverse(x)) is x again. Single ASCII characters are
// never replaced (< stays <, not lt) and unknown characters are kept. Where a
// name would run into the letters around it, the text is made unambiguous:
// a name right after a letter or digit gets a backslash (x\gamma), and a name
// right before one is closed with an empty group (del{}x).
func Reverse(input string, opt ReverseOptions) string {
	set := opt.Set
	if set == nil {
		set = symbols.Default()
	}
	var b strings.Builder
	afterAlnum, afterName := false, false // what was written last: an ASCII letter or digit, a name
	for i := 0; i < len(input); {
		if names, size := longestName(set, input[i:], false); names != nil {
			text := names[0]
			// script keys (^2, _n) are written as they are; the backslash would turn them into a text accent
			script := text[0] == '^' || text[0] == '_'
			slash := !script && (opt.Backslash || len(text) < 2 || afterAlnum)
			if slash {
				text = `\` + text
			} else if afterName && !script {
				b.WriteByte(' ') // two bare names would run together; Convert drops this space
			}
			b.WriteString(text)
			i += size
			if i < len(input) && isAlnumASCII(input[i]) {
				b.WriteString("{}")
			}
			afterAlnum, afterName = false, true
			continue
		}
		r, size := utf8.DecodeRuneInString(input[i:])
		b.WriteString(input[i : i+size])
		afterAlnum = r < utf8.RuneSelf && isAlnumASCII(byte(r))
		afterName = false
		i += size
	}
	return b.String()
}

func isAlnumASCII(c byte) bool { return isLetter(c) || isDigit(c) }

// longestName finds the longest prefix of s that is the value of some symbol
// and returns its names (best first) and the length of the prefix in bytes.
// Lone ASCII characters match only if ascii is true.
func longestName(set *symbols.Set, s string, ascii bool) (names []string, size int) {
	ends := make([]int, 0, 4)
	for off := 0; off < len(s) && len(ends) < set.MaxValueRunes(); {
		_, w := utf8.DecodeRuneInString(s[off:])
		off += w
		ends = append(ends, off)
	}
	for n := len(ends); n >= 1; n-- {
		prefix := s[:ends[n-1]]
		if n == 1 && prefix[0] < utf8.RuneSelf && !ascii {
			continue
		}
		if names := set.Reverse(prefix); len(names) > 0 {
			return names, len(prefix)
		}
	}
	return nil, 0
}

// Info describes one character (or multi character symbol) of the input.
type Info struct {
	Text  string
	Runes []rune
	Bytes []byte // the UTF-8 encoding
	Names []string
}

// Describe lists the non-blank characters of input with their code points,
// UTF-8 bytes and every name they are known by.
func Describe(input string, set *symbols.Set) []Info {
	if set == nil {
		set = symbols.Default()
	}
	var out []Info
	for i := 0; i < len(input); {
		names, size := longestName(set, input[i:], true)
		if size == 0 {
			_, size = utf8.DecodeRuneInString(input[i:])
		}
		text := input[i : i+size]
		i += size
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, Info{Text: text, Runes: []rune(text), Bytes: []byte(text), Names: names})
	}
	return out
}
