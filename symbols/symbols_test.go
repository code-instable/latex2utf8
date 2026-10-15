package symbols

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// ---- the embedded data -------------------------------------------------------

func TestDefaultLoads(t *testing.T) {
	s := Default()
	if n := s.Len(); n < 2800 {
		t.Fatalf("only %d names loaded, the table has shrunk", n)
	}
	if Default() != s {
		t.Error("Default is not cached")
	}
}

func TestKnownValues(t *testing.T) {
	s := Default()
	tests := map[string]string{
		"gamma": "γ", "Gamma": "Γ", "mbfGamma": "𝚪", "mitX": "𝑋", "BbbR": "ℝ", "nabla": "∇",
		"neq": "≠", "varepsilon": "ɛ", "L": "Ł", "o": "ø", "_1": "₁", "^2": "²",
		"quad":   " ",  // whitespace values must survive unchanged
		"neqsim": "≂̸", // multi rune values
		// additions
		"_n": "ₙ", "_a": "ₐ", "^q": "𐞥", "omicron": "ο", "Mu": "Μ", "Nu": "Ν", "Omicron": "Ο",
		"ldots": "…", "land": "∧", "lor": "∨", "lnot": "¬", "impliedby": "⟸", "lbrace": "{", "note8th": "\U0001D160",
		"mith": "ℎ", "mfrakI": "ℑ", "mfrakR": "ℜ", "frac12": "½", "frac34": "¾",
		"cyrBe": "Б", "cyri": "й", "clefg": "𝄞",
	}
	for name, want := range tests {
		got, ok := s.Lookup(name)
		if !ok || got != want {
			t.Errorf("Lookup(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
	if _, ok := s.Lookup("definitelynotasymbol"); ok {
		t.Error("unknown name resolved")
	}
}

func TestNamesAndValues(t *testing.T) {
	for _, e := range Default().Entries() {
		if !validName(e.Name) {
			t.Errorf("invalid name %q", e.Name)
		}
		if e.Value == "" || !utf8.ValidString(e.Value) {
			t.Errorf("%q has invalid value %q", e.Name, e.Value)
		}
	}
}

func TestEveryFontPrefixCoversTheLatinAlphabet(t *testing.T) {
	s := Default()
	seen := map[string]bool{}
	for _, e := range s.cmds {
		if e.Kind != Font || seen[e.Prefix] {
			continue
		}
		seen[e.Prefix] = true
		for c := 'a'; c <= 'z'; c++ {
			for _, r := range []rune{c, unicode.ToUpper(c)} {
				if _, ok := s.Lookup(e.Prefix + string(r)); !ok {
					t.Errorf("math alphabet %s has no %c", e.Prefix, r)
				}
			}
		}
	}
	if len(seen) < 13 {
		t.Errorf("only %d math alphabets found", len(seen))
	}
}

func TestCommandsAreWellFormed(t *testing.T) {
	s := Default()
	for name, c := range s.cmds {
		switch c.Kind {
		case Accent, WideAccent:
			if !unicode.In(c.Mark, unicode.Mn, unicode.Me) {
				t.Errorf("%s: U+%04X is not a combining mark", name, c.Mark)
			}
		case Font:
			if _, ok := s.Lookup(c.Prefix + "A"); !ok {
				t.Errorf("%s: unknown alphabet prefix %q", name, c.Prefix)
			}
		}
	}
	for _, name := range []string{"hat", "bar", "vec", "tilde", "acute", "mathbb", "Bbb", "frac", "sqrt", "text", "overline"} {
		if _, ok := s.Command(name); !ok {
			t.Errorf("missing command %q", name)
		}
	}
}

func TestComposeAndFractions(t *testing.T) {
	s := Default()
	compose := []struct {
		base, mark, want rune
	}{
		{'a', 0x302, 'â'}, {'e', 0x301, 'é'}, {'n', 0x303, 'ñ'}, {'=', 0x338, '≠'}, {'∈', 0x338, '∉'}, {'и', 0x306, 'й'},
	}
	for _, c := range compose {
		if got, ok := s.Compose(c.base, c.mark); !ok || got != c.want {
			t.Errorf("Compose(%q, U+%04X) = %q, %v; want %q", c.base, c.mark, got, ok, c.want)
		}
	}
	if _, ok := s.Compose('x', 0x302); ok {
		t.Error("x + circumflex has no precomposed form")
	}
	if v, ok := s.Fraction("1", "2"); !ok || v != "½" {
		t.Errorf("Fraction(1,2) = %q, %v", v, ok)
	}
	if _, ok := s.Fraction("7", "9"); ok {
		t.Error("7/9 is not a vulgar fraction")
	}
}

// ---- lookups -------------------------------------------------------------------

func TestFold(t *testing.T) {
	s := Default()
	tests := []struct {
		in, name string
		ok       bool
	}{
		{"bbbR", "BbbR", true}, // two candidates (BbbR, Bbbr): the closer one wins
		{"bbbr", "Bbbr", true},
		{"GAMMA", "Gamma", true}, // all upper case is closer to Gamma than to gamma
		{"mbfgamma", "mbfgamma", true},
		{"xyzzy", "", false},
	}
	for _, tt := range tests {
		name, _, ok := s.Fold(tt.in)
		if ok != tt.ok || (ok && name != tt.name) {
			t.Errorf("Fold(%q) = %q, %v; want %q, %v", tt.in, name, ok, tt.name, tt.ok)
		}
	}
}

func TestSegment(t *testing.T) {
	s := Default()
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"alphabeta", "αβ", true},
		{"mbfGammamitX", "𝚪𝑋", true},
		{"hello", "", false},
		{"alphax", "", false}, // 'x' alone is not a piece
		{"pimu", "", false},   // pieces shorter than MinSegment are never used
		{"al", "", false},
	}
	for _, tt := range tests {
		got, ok := s.Segment(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Segment(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestFuzzy(t *testing.T) {
	s := Default()
	m, _, ok := s.Fuzzy("gammma", 1)
	if !ok || m.Name != "gamma" || m.Value != "γ" {
		t.Errorf(`Fuzzy("gammma") = %+v, %v`, m, ok)
	}
	if _, _, ok := s.Fuzzy("zzzzzzzz", 2); ok {
		t.Error("nonsense should not match")
	}
	if _, _, ok := s.Fuzzy("gammma", 0); ok {
		t.Error("zero edits allowed must not match")
	}
}

func TestFuzzyTiesAreReportedNotGuessed(t *testing.T) {
	s, err := Parse(Sources{Symbols: strings.NewReader("alpha\tA\nalphb\tB\nalphc\tA\n")})
	if err != nil {
		t.Fatal(err)
	}
	// "alphx" is one substitution away from all three
	m, tied, ok := s.Fuzzy("alphx", 1)
	if ok {
		t.Fatalf("ambiguous input resolved to %+v", m)
	}
	if want := []string{"alpha", "alphb", "alphc"}; !reflect.DeepEqual(tied, want) {
		t.Errorf("tied = %v, want %v", tied, want)
	}
	// candidates with the same value are not ambiguous
	s, _ = Parse(Sources{Symbols: strings.NewReader("alpha\tA\nalphb\tA\n")})
	if m, _, ok := s.Fuzzy("alphx", 1); !ok || m.Name != "alpha" {
		t.Errorf("aliases of one symbol should resolve to the first, got %+v, %v", m, ok)
	}
}

func TestLookupsAreDeterministic(t *testing.T) {
	// map iteration order used to leak into results; run many times to catch a regression
	var first [4]string
	for i := 0; i < 100; i++ {
		s := Default()
		var got [4]string
		got[0], _ = s.Segment("alphabetagamma")
		m, _, _ := s.Fuzzy("thetaa", 1)
		got[1] = m.Name
		_, got[2], _ = s.Fold("bbbR")
		got[3] = strings.Join(s.Suggest("gamm", 5), ",")
		if i == 0 {
			first = got
		} else if got != first {
			t.Fatalf("run %d: %v differs from first run %v", i, got, first)
		}
	}
}

func TestSuggest(t *testing.T) {
	s := Default()
	got := s.Suggest("gamm", 3)
	if len(got) == 0 || len(got) > 3 {
		t.Fatalf("Suggest returned %v", got)
	}
	found := false
	for _, g := range got {
		if g == "gamma" {
			found = true
		}
	}
	if !found {
		t.Errorf("gamma not suggested for gamm: %v", got)
	}
	if got := s.Suggest("", 3); got != nil {
		t.Errorf("empty word: %v", got)
	}
}

func TestReverse(t *testing.T) {
	s := Default()
	if names := s.Reverse("γ"); len(names) == 0 || names[0] != "gamma" {
		t.Errorf("Reverse(γ) = %v, want gamma first", names)
	}
	if names := s.Reverse("Ł"); !reflect.DeepEqual(names, []string{"L"}) {
		t.Errorf("Reverse(Ł) = %v", names)
	}
	if s.MaxValueRunes() < 2 {
		t.Error("multi rune values exist, MaxValueRunes must be at least 2")
	}
	// names usable without a backslash always come before one letter names
	for _, e := range s.Entries() {
		names := s.Reverse(e.Value)
		for i := 1; i < len(names); i++ {
			if len(names[i-1]) < 2 && len(names[i]) >= 2 {
				t.Errorf("%q: %v lists a one letter name before %q", e.Value, names, names[i])
			}
		}
	}
}

func TestEntriesSortedAndAliasFlag(t *testing.T) {
	es := Default().Entries()
	for i := 1; i < len(es); i++ {
		if es[i-1].Name >= es[i].Name {
			t.Fatalf("not strictly sorted at %q, %q", es[i-1].Name, es[i].Name)
		}
	}
	flags := map[string]bool{}
	for _, e := range es {
		flags[e.Name] = e.Alias
	}
	if flags["gamma"] || !flags["cyrBe"] || !flags["ldots"] {
		t.Errorf("alias flags wrong: gamma=%v cyrBe=%v ldots=%v", flags["gamma"], flags["cyrBe"], flags["ldots"])
	}
}

// ---- the loader ----------------------------------------------------------------

func TestParseErrors(t *testing.T) {
	sym := func(s string) Sources { return Sources{Symbols: strings.NewReader(s)} }
	tests := []struct {
		name string
		src  Sources
		want string // substring of the error
	}{
		{"leading digit", sym("1ab\tx\n"), "invalid name"},
		{"punctuation in name", sym("a-b\tx\n"), "invalid name"},
		{"duplicate name", sym("abc\tx\nabc\ty\n"), "duplicate name"},
		{"invalid name", sym("a b\tx\n"), "invalid name"},
		{"bad script", sym("^ab1\tx\n"), "invalid name"},
		{"empty value", sym("abc\t\n"), "empty value"},
		{"missing field", sym("abc\n"), "symbols.tsv:1"},
		{"control char", sym("abc\tx\x01\n"), "control character"},
		{"bad utf8", sym("abc\t\xff\n"), "not valid UTF-8"},
		{"alias target", Sources{Symbols: strings.NewReader("abc\tx\n"), Aliases: strings.NewReader("def\tnope\n")}, "unknown target"},
		{"alias clash", Sources{Symbols: strings.NewReader("abc\tx\n"), Aliases: strings.NewReader("abc\tabc\n")}, "already defined"},
		{"command kind", Sources{Commands: strings.NewReader("hat\tspin\t0302\n")}, "unknown kind"},
		{"command mark", Sources{Commands: strings.NewReader("hat\taccent\tzz\n")}, "invalid code point"},
		{"command dup", Sources{Commands: strings.NewReader("hat\taccent\t0302\nhat\taccent\t0302\n")}, "duplicate command"},
		{"font prefix", Sources{Commands: strings.NewReader("mb\tfont\t12\n")}, "invalid font prefix"},
		{"compose", Sources{Compose: strings.NewReader("0061\t0302\n")}, "compose.tsv:1"},
		{"fraction", Sources{Fractions: strings.NewReader("a/b\tx\n")}, "invalid fraction"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.src)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want it to contain %q", tt.name, err, tt.want)
		}
	}
}

func TestParseFormat(t *testing.T) {
	src := "# comment\n\nabc\tx\tinline comment\r\nde\t y \n"
	s, err := Parse(Sources{Symbols: strings.NewReader(src)})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Lookup("abc"); v != "x" {
		t.Errorf("CRLF or comment column broke the value: %q", v)
	}
	if v, _ := s.Lookup("de"); v != " y " {
		t.Errorf("values must not be trimmed, got %q", v)
	}
	if s.Len() != 2 {
		t.Errorf("Len = %d", s.Len())
	}
}

func BenchmarkLoadEmbedded(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := loadEmbedded(); err != nil {
			b.Fatal(err)
		}
	}
}
