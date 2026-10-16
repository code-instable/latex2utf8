package convert

import (
	"strings"
	"unicode"
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
// of Convert: whenever Convert leaves a text unchanged, Convert(Reverse(x)) is x
// again. (This is checked on every call, not just hoped for.) Single ASCII
// characters are never replaced (< stays <, not lt) and unknown characters are kept.
//
// Convert reads a run of letters, of any script, as one word, and ^ _ \ { [
// as syntax. So where a name would be misread next to the text around it,
// Reverse fences it in:
//
//   - a name right after a letter (of any script), a digit, ^ or _ gets a
//     backslash: x\gamma, ^\gamma
//   - a name right before a letter or digit, or before a { or [ that would be
//     taken as its argument, is closed with an empty group: del{}x, sqrt{}{x}
//   - a literal {} after a name is escaped, so that it is not mistaken for the
//     closing group: gamma{}\{\}
//   - a symbol right after a literal backslash stays as it is, because no name
//     can follow a lone backslash
//   - letters that form a word with a symbol, and that would turn into a name
//     once the symbol is replaced (the alpha of "𝚪alpha"), are written
//     text{alpha}
func Reverse(input string, opt ReverseOptions) string {
	set := opt.Set
	if set == nil {
		set = symbols.Default()
	}
	out, cuts := reverse(input, set, opt, false)
	if out == input {
		return out
	}
	// The fences above cover the cases known to go wrong; this makes the promise hold for the
	// rest. If Convert would leave the input alone, the output must read back as the input: if it
	// does not, leave the words that mix scripts as they are, and failing that change nothing.
	// The one thing it cannot give back is whitespace between two symbols, which Convert drops
	// (that is what makes "alpha beta" be αβ), so that is not expected to come back.
	check := Options{Strict: true, Set: set}
	if Convert(input, check).Output != input {
		return out // lutf itself would change this text: there is nothing to preserve
	}
	if Convert(out, check).Output == without(input, cuts) {
		return out
	}
	if out, cuts = reverse(input, set, opt, true); Convert(out, check).Output == without(input, cuts) {
		return out
	}
	return input
}

// without returns s with the given [start, end) ranges removed.
func without(s string, cuts [][2]int) string {
	if len(cuts) == 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, c := range cuts {
		b.WriteString(s[last:c[0]])
		last = c[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// reverse does the work of Reverse. With keepMixed, a word that mixes ASCII letters or digits with
// letters of another script is copied as it is, symbols and all.
// It also returns the ranges of input holding whitespace between two names, which Convert will drop.
func reverse(input string, set *symbols.Set, opt ReverseOptions, keepMixed bool) (string, [][2]int) {
	var b strings.Builder
	var dropped [][2]int
	var prev, prev2 byte // the last two ASCII characters copied as they were (0: not that, or a name)
	// Convert reads \\ as one escaped backslash, so only an odd run of backslashes ends in a lone one that
	// starts a command. run is the length of the run of backslashes just copied, before the length of the
	// run that ended before the last character.
	run, runBefore := 0, 0
	prevLetter := false // the last character copied as it was is a non-ASCII letter or mark
	inLetter := false   // the last character of the input read is a non-ASCII letter or mark
	afterName := false  // the last thing written was a name
	for i := 0; i < len(input); {
		loneBackslash := prev == '\\' && run%2 == 1 // the text so far ends in a backslash that starts a command
		if keepMixed && (isAlnumASCII(input[i]) || startsWord(input, i)) {
			j, ascii, other := i, false, false
			for j < len(input) {
				if isAlnumASCII(input[j]) {
					ascii = true
					j++
				} else if startsWord(input, j) {
					other = true
					_, size := utf8.DecodeRuneInString(input[j:])
					j += size
				} else {
					break
				}
			}
			if ascii && other {
				b.WriteString(input[i:j])
				last, _ := utf8.DecodeLastRuneInString(input[i:j])
				prev, prev2, prevLetter, inLetter, afterName, run, runBefore = 0, 0, false, false, false, 0, 0
				if last < utf8.RuneSelf {
					prev = byte(last)
				} else {
					prevLetter, inLetter = true, true
				}
				i = j
				continue
			}
		}
		if afterName && isSpace(input[i]) {
			j := i
			for j < len(input) && isSpace(input[j]) {
				j++
			}
			if j < len(input) && !strings.ContainsAny(input[i:j], "\r\n") {
				if next, _ := longestName(set, input[j:], false); next != nil {
					b.WriteString(input[i:j]) // between two names: Convert drops it, and it replaces the separator
					dropped = append(dropped, [2]int{i, j})
					prev, prev2, prevLetter, inLetter, afterName, run, runBefore = ' ', 0, false, false, false, 0, 0
					i = j
					continue
				}
			}
		}
		if !loneBackslash { // no name can follow a lone backslash
			if names, size := longestName(set, input[i:], false); names != nil {
				text := names[0]
				bare := text
				// script keys (^2, _n) are written as they are; the backslash would turn them into a text accent
				script := text[0] == '^' || text[0] == '_'
				afterTextAccent := prev2 == '\\' && runBefore%2 == 1 && textAccents[prev] != 0 // \' + n would be the accent \'n
				slash := !script && (opt.Backslash || len(text) < 2 || isAlnumASCII(prev) || prevLetter || prev == '^' || prev == '_' || afterTextAccent)
				if slash {
					text = `\` + text
				} else if afterName && !script {
					b.WriteByte(' ') // two bare names would run together; Convert drops this space
				}
				b.WriteString(text)
				last, _ := utf8.DecodeLastRuneInString(input[i : i+size])
				inLetter = last >= utf8.RuneSelf && isLetterOrMark(last)
				i += size
				rest := input[i:]
				prev, prev2, prevLetter, afterName, run, runBefore = 0, 0, false, true, 0, 0
				switch {
				case strings.HasPrefix(rest, "{}"):
					b.WriteString(`{}\{\}`)
					i += 2
					prev, afterName, inLetter = '}', false, false
				case rest != "" && isAlnumASCII(rest[0]):
					b.WriteString("{}")
				case rest != "" && (rest[0] == '{' || rest[0] == '[') && isCommand(set, bare):
					b.WriteString("{}")
				case rest != "" && startsWord(rest, 0):
					// a letter that is not replaced either would join the name into one word
					if again, _ := longestName(set, rest, false); again == nil {
						b.WriteString("{}")
					}
				}
				continue
			}
		}
		if isLetter(input[i]) {
			j := i
			for j < len(input) && isLetter(input[j]) {
				j++
			}
			seg := input[i:j]
			// ASCII letters sharing a word with a symbol: Convert leaves the word alone, but once the
			// symbol becomes a name these letters stand by themselves and may turn into one too. That
			// happens only if every neighbouring letter of another script is replaced; one that stays
			// keeps the word together.
			if !prevLetter && !loneBackslash { // after a lone backslash the letters are a command name, not a word
				rightReplaced, rightKept := false, false
				if startsWord(input, j) {
					names, _ := longestName(set, input[j:], false)
					rightReplaced, rightKept = names != nil, names == nil
				}
				if (afterName && inLetter || rightReplaced) && !rightKept && Convert(seg, Options{Fuzzy: true}).Output != seg {
					b.WriteString("text{" + seg + "}")
					prev, prev2, prevLetter, inLetter, afterName, run, runBefore = '}', 0, false, false, false, 0, 0
					i = j
					continue
				}
			}
			b.WriteString(seg)
			prev, prev2, prevLetter, inLetter, afterName, run, runBefore = seg[len(seg)-1], 0, false, false, false, 0, 0
			if len(seg) > 1 {
				prev2 = seg[len(seg)-2]
			}
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(input[i:])
		b.WriteString(input[i : i+size])
		prev2 = prev
		prev, prevLetter = 0, false
		if r == '\\' {
			run++
		} else {
			runBefore, run = run, 0
		}
		if r < utf8.RuneSelf {
			prev = byte(r)
		} else {
			prev2 = 0
			prevLetter = isLetterOrMark(r)
		}
		inLetter = r >= utf8.RuneSelf && isLetterOrMark(r)
		afterName = false
		i += size
	}
	return b.String(), dropped
}

func isLetterOrMark(r rune) bool { return unicode.IsLetter(r) || unicode.IsMark(r) }

func isCommand(set *symbols.Set, name string) bool {
	_, ok := set.Command(name)
	return ok
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
