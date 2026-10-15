// Package symbols holds the symbol database of lutf and the lookups on it.
//
// The data lives in tab separated files under data/, embedded in the binary:
//
//	symbols.tsv    name -> value             the canonical symbols
//	aliases.tsv    alias -> existing name    extra spellings of a symbol
//	commands.tsv   name -> kind, parameter   commands taking a {...} argument
//	fractions.tsv  n/d -> character          vulgar fractions
//	compose.tsv    base, mark -> character   precomposed accented characters
//
// Names are pure ASCII letters, except script keys, which start with ^ or _
// (^2, _n, ^beta) and give the superscript or subscript form of a character.
package symbols

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

//go:embed data/*.tsv
var dataFS embed.FS

// Kind tells what a Command does with its argument.
type Kind uint8

const (
	// Accent puts a combining mark on the last rune of the argument.
	Accent Kind = iota + 1
	// WideAccent puts a combining mark on every rune of the argument.
	WideAccent
	// Font maps the letters and digits of the argument to a math alphabet.
	Font
	// Frac takes two arguments and builds a fraction.
	Frac
	// Sqrt builds a (possibly indexed) square root.
	Sqrt
	// Text copies its argument verbatim.
	Text
)

var kindNames = map[string]Kind{
	"accent": Accent, "wideaccent": WideAccent, "font": Font,
	"frac": Frac, "sqrt": Sqrt, "text": Text,
}

// Command is a name that, followed by a {...} argument, does more than a
// plain symbol lookup.
type Command struct {
	Name   string
	Kind   Kind
	Mark   rune   // Accent, WideAccent: the combining mark
	Prefix string // Font: prefix of the math alphabet names (Bbb, mbf, ...)
}

// Entry is one resolvable name, as listed by Set.Entries.
type Entry struct {
	Name  string
	Value string
	Alias bool // not the canonical name of its symbol
}

// Sources are the readers Parse builds a Set from. Nil readers stand for empty files.
type Sources struct {
	Symbols, Aliases, Commands, Fractions, Compose io.Reader
}

// Set is an immutable symbol database, safe for concurrent use.
type Set struct {
	values    map[string]string // every resolvable plain or script name
	canonical map[string]bool   // names that are not aliases
	cmds      map[string]Command
	compose   map[[2]rune]rune
	fracs     map[string]string
	maxLen    int // longest letters-only name, in bytes

	foldOnce sync.Once
	fold     map[string][]string // lower-cased letters-only name -> names, sorted

	fuzzyOnce sync.Once
	cands     []candidate
	byLen     map[int][]int // rune length -> indexes into cands

	revOnce sync.Once
	rev     map[string][]string
	maxVal  int // longest value, in runes
}

var (
	defaultOnce sync.Once
	defaultSet  *Set
	defaultErr  error
)

// Default returns the database embedded in the binary. The embedded data is
// checked by the tests, so a failure to load it is a programming error and panics.
func Default() *Set {
	defaultOnce.Do(func() { defaultSet, defaultErr = loadEmbedded() })
	if defaultErr != nil {
		panic("symbols: embedded data is invalid: " + defaultErr.Error())
	}
	return defaultSet
}

func loadEmbedded() (*Set, error) {
	var openErr error
	open := func(name string) io.Reader {
		f, err := dataFS.Open("data/" + name)
		if err != nil {
			openErr = err
			return nil
		}
		return f
	}
	src := Sources{
		Symbols: open("symbols.tsv"), Aliases: open("aliases.tsv"), Commands: open("commands.tsv"),
		Fractions: open("fractions.tsv"), Compose: open("compose.tsv"),
	}
	if openErr != nil {
		return nil, openErr
	}
	return Parse(src)
}

// Parse builds a Set from the given sources, validating every record.
func Parse(src Sources) (*Set, error) {
	s := &Set{
		values:    map[string]string{},
		canonical: map[string]bool{},
		cmds:      map[string]Command{},
		compose:   map[[2]rune]rune{},
		fracs:     map[string]string{},
	}
	err := records(src.Symbols, "symbols", 2, func(f []string) error {
		name, value := f[0], f[1]
		if !validName(name) {
			return fmt.Errorf("invalid name %q", name)
		}
		if err := validValue(value); err != nil {
			return err
		}
		if _, dup := s.values[name]; dup {
			return fmt.Errorf("duplicate name %q", name)
		}
		s.add(name, value, true)
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = records(src.Fractions, "fractions", 2, func(f []string) error {
		num, den, ok := strings.Cut(f[0], "/")
		if !ok || !allDigits(num) || !allDigits(den) {
			return fmt.Errorf("invalid fraction %q", f[0])
		}
		if err := validValue(f[1]); err != nil {
			return err
		}
		s.fracs[f[0]] = f[1]
		// \frac12 style names for single digit fractions, unless already defined
		if len(num) == 1 && len(den) == 1 {
			if name := "frac" + num + den; s.values[name] == "" {
				s.add(name, f[1], true)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = records(src.Aliases, "aliases", 2, func(f []string) error {
		alias, target := f[0], f[1]
		if !validName(alias) {
			return fmt.Errorf("invalid alias %q", alias)
		}
		value, ok := s.values[target]
		if !ok {
			return fmt.Errorf("alias %q: unknown target %q", alias, target)
		}
		if _, dup := s.values[alias]; dup {
			return fmt.Errorf("alias %q is already defined", alias)
		}
		s.add(alias, value, false)
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = records(src.Commands, "commands", 3, func(f []string) error {
		name := f[0]
		if !allLetters(name) {
			return fmt.Errorf("invalid command name %q", name)
		}
		kind, ok := kindNames[f[1]]
		if !ok {
			return fmt.Errorf("command %q: unknown kind %q", name, f[1])
		}
		if _, dup := s.cmds[name]; dup {
			return fmt.Errorf("duplicate command %q", name)
		}
		c := Command{Name: name, Kind: kind}
		switch kind {
		case Accent, WideAccent:
			cp, err := strconv.ParseUint(f[2], 16, 32)
			if err != nil || !utf8.ValidRune(rune(cp)) || cp == 0 {
				return fmt.Errorf("command %q: invalid code point %q", name, f[2])
			}
			c.Mark = rune(cp)
		case Font:
			if !allLetters(f[2]) {
				return fmt.Errorf("command %q: invalid font prefix %q", name, f[2])
			}
			c.Prefix = f[2]
		}
		s.cmds[name] = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = records(src.Compose, "compose", 3, func(f []string) error {
		var cp [3]rune
		for i := range cp {
			v, err := strconv.ParseUint(f[i], 16, 32)
			if err != nil || !utf8.ValidRune(rune(v)) || v == 0 {
				return fmt.Errorf("invalid code point %q", f[i])
			}
			cp[i] = rune(v)
		}
		s.compose[[2]rune{cp[0], cp[1]}] = cp[2]
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Set) add(name, value string, canonical bool) {
	s.values[name] = value
	if canonical {
		s.canonical[name] = true
	}
	if allLetters(name) && len(name) > s.maxLen {
		s.maxLen = len(name)
	}
}

// records calls fn for the tab separated fields of every record of r. Blank
// lines and lines starting with # are skipped; fields beyond n (comments) are
// dropped. Errors are prefixed with the file name and line number.
func records(r io.Reader, file string, n int, fn func(fields []string) error) error {
	if r == nil {
		return nil
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSuffix(sc.Text(), "\r")
		if text == "" || text[0] == '#' {
			continue
		}
		f := strings.SplitN(text, "\t", n+1)
		if len(f) < n {
			return fmt.Errorf("%s.tsv:%d: want %d tab separated fields, got %d", file, line, n, len(f))
		}
		if err := fn(f[:n]); err != nil {
			return fmt.Errorf("%s.tsv:%d: %w", file, line, err)
		}
	}
	return sc.Err()
}

func validValue(v string) error {
	if v == "" {
		return fmt.Errorf("empty value")
	}
	if !utf8.ValidString(v) {
		return fmt.Errorf("value %q is not valid UTF-8", v)
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f || r == utf8.RuneError {
			return fmt.Errorf("value %q contains control character U+%04X", v, r)
		}
	}
	return nil
}

// validName reports whether n is an identifier (a letter followed by letters
// and digits, like note8th) or a script key (^x, _x, ^name, _name, where x is
// one of the characters of scriptPunct).
func validName(n string) bool {
	if n == "" {
		return false
	}
	if n[0] == '^' || n[0] == '_' {
		rest := n[1:]
		if allLetters(rest) {
			return true
		}
		return len(rest) == 1 && strings.IndexByte(scriptPunct, rest[0]) >= 0
	}
	return allLetters(n[:1]) && allAlnum(n)
}

func allAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; !isLetterByte(c) && !isDigitByte(c) {
			return false
		}
	}
	return s != ""
}

// scriptPunct are the non-letters allowed after ^ or _ in a script key.
const scriptPunct = "0123456789+-=()!<>"

func allLetters(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isLetterByte(s[i]) {
			return false
		}
	}
	return true
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigitByte(s[i]) {
			return false
		}
	}
	return true
}

func isLetterByte(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

// Lookup returns the value of an exact name.
func (s *Set) Lookup(name string) (string, bool) {
	v, ok := s.values[name]
	return v, ok
}

// Command returns the argument command of that name.
func (s *Set) Command(name string) (Command, bool) {
	c, ok := s.cmds[name]
	return c, ok
}

// Compose returns the precomposed character for base followed by a combining mark.
func (s *Set) Compose(base, mark rune) (rune, bool) {
	r, ok := s.compose[[2]rune{base, mark}]
	return r, ok
}

// Fraction returns the vulgar fraction character for num/den.
func (s *Set) Fraction(num, den string) (string, bool) {
	v, ok := s.fracs[num+"/"+den]
	return v, ok
}

// Len returns the number of resolvable names.
func (s *Set) Len() int { return len(s.values) }
