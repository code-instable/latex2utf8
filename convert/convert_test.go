package convert

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/code-instable/latex2utf8/symbols"
)

var fuzzyOn = Options{Fuzzy: true}

func conv(t *testing.T, opt Options, in string) string {
	t.Helper()
	return Convert(in, opt).Output
}

type tc struct{ in, want string }

func run(t *testing.T, opt Options, tests []tc) {
	t.Helper()
	for _, tt := range tests {
		if got := conv(t, opt, tt.in); got != tt.want {
			t.Errorf("Convert(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// The examples of the README and of the --help text.
func TestDocumentedExamples(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"mbfGamma", "𝚪"},
		{"mitX_1", "𝑋₁"},
		{"mitX _1", "𝑋₁"},
		{"_n", "ₙ"},
		{"varepsilon", "ɛ"},
		{"BbbR", "ℝ"},
		{"bbbR", "ℝ"},
		{"gammma", "γ"},
		{"mbfGamma mitX neq mitu _0", "𝚪𝑋≠𝑢₀"},
		{"mbfGamma mitX _2", "𝚪𝑋₂"},
	})
}

func TestOptionalBackslash(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"gamma", "γ"}, {`\gamma`, "γ"},
		{"Gamma", "Γ"}, {`\Gamma`, "Γ"},
		{`\alpha+\beta`, "α+β"},
		{`\alpha \beta`, "αβ"},
		{`\mathbb{R}`, "ℝ"}, {"mathbb{R}", "ℝ"},
		{`\frac{1}{2}`, "½"}, {"frac{1}{2}", "½"},
		{`\L`, "Ł"}, {`\o`, "ø"}, {`\i`, "й"},
		{`\note8th`, "\U0001D160"}, {"note8th", "\U0001D160"},
	})
}

// A one letter name is far too common in running text to be read without a backslash.
func TestOneLetterNamesNeedABackslash(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"I", "I"}, {"i", "i"}, {"L", "L"}, {"a b o", "a b o"},
		{`\I`, "Й"}, {`\L`, "Ł"},
	})
}

// Every name in the database converts to its value, written bare or with a backslash.
func TestEveryNameConverts(t *testing.T) {
	set := symbols.Default()
	for _, e := range set.Entries() {
		in := e.Name
		if len(in) < 2 && !strings.ContainsAny(in[:1], "^_") {
			in = `\` + in
		}
		if got := Convert(in, Options{Strict: true}).Output; got != e.Value {
			t.Errorf("Convert(%q) = %q, want %q", in, got, e.Value)
		}
		if e.Name[0] != '^' && e.Name[0] != '_' {
			if got := Convert(`\`+e.Name, Options{Strict: true}).Output; got != e.Value {
				t.Errorf("Convert(%q) = %q, want %q", `\`+e.Name, got, e.Value)
			}
		}
	}
}

// Text that is not made of symbol names is left alone. This used to be
// replaced piecemeal ("hello world" became a single "𝒹").
func TestPassThrough(t *testing.T) {
	for _, in := range []string{
		"hello world", "foo", "xyzzy", "123 + 456 = 579", "(a, b) [c] {d}", "naïve café", "日本語 text",
		"The quick brown fox", "x", "", "   ", "\n\n", "a  b", "tab\tseparated",
	} {
		if got := conv(t, fuzzyOn, in); got != in {
			t.Errorf("Convert(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestSubstringsOfWordsAreNotReplaced(t *testing.T) {
	// "the", "value", "of" contain the names "he", "val", "of", "alp"... as substrings
	in := "the value of everything is large"
	if got := conv(t, fuzzyOn, in); got != in {
		t.Errorf("Convert(%q) = %q", in, got)
	}
	run(t, fuzzyOn, []tc{{"the value of alpha is large", "the value of α is large"}})
}

func TestWhitespace(t *testing.T) {
	run(t, Options{}, []tc{
		{"alpha beta", "αβ"},              // between two converted pieces: dropped
		{"alpha    \t beta", "αβ"},        // any amount of it
		{"x = alpha + beta", "x = α + β"}, // next to text that stays: kept
		{"neq 0", "≠ 0"},
		{"alpha\nbeta", "α\nβ"}, // line breaks are structure, never dropped
		{"  alpha  ", "  α  "},  // leading and trailing space is kept
		{"alpha\r\n\r\nbeta", "α\r\n\r\nβ"},
	})
	run(t, Options{Join: true}, []tc{
		{"x = alpha + beta", "x=α+β"},
		{"alpha\nbeta", "αβ"},
	})
}

// Whitespace is Unicode whitespace, not only ASCII: text pasted from a web page is full of
// non-breaking spaces, and the first version of lutf (strings.Fields) split on them.
func TestUnicodeWhitespace(t *testing.T) {
	run(t, Options{}, []tc{
		{"alpha\u00a0beta", "αβ"}, // dropped between two converted names, like a space
		{"alpha\u2003beta", "αβ"},
		{"alpha\u3000beta", "αβ"},
		{"alpha\u202fbeta", "αβ"},
		{"x\u00a0=\u00a0alpha", "x\u00a0=\u00a0α"}, // next to anything else: kept
		{"alpha\u2028beta", "α\u2028β"},            // line and paragraph separators are line breaks:
		{"alpha\u2029beta", "α\u2029β"},            // never dropped
		{"alpha\u0085beta", "α\u0085β"},
		{"frac{1}\u00a0{2}", "½"}, // between the two arguments
		{"x^{a\u00a0b}", "xᵃᵇ"},   // inside a script
		{"naïve\u00a0café", "naïve\u00a0café"},
		{"\u00a0alpha\u00a0", "\u00a0α\u00a0"}, // at the ends
	})
	run(t, Options{Join: true}, []tc{
		{"x\u00a0=\u00a0alpha\u2003beta", "x=αβ"}, // Join removes all of it
		{"a\u2028b", "ab"},
	})
}

func TestScripts(t *testing.T) {
	run(t, Options{}, []tc{
		{"x^2", "x²"}, {"x_1", "x₁"}, {"x^{12}", "x¹²"}, {"x_{12}", "x₁₂"},
		{"x^12", "x¹²"}, {"_12", "₁₂"}, // an unbraced number is taken whole
		{"x^-1", "x⁻¹"}, {"x^+2", "x⁺²"}, {"x^{-1}", "x⁻¹"}, {"x^{n+1}", "xⁿ⁺¹"},
		{"x^{ab}", "xᵃᵇ"}, {"x_{ij}", "xᵢⱼ"}, {"x^{ a b }", "xᵃᵇ"},
		{"x^n", "xⁿ"}, {"x_n", "xₙ"}, {"_n", "ₙ"}, {"^T", "ᵀ"},
		{"x^{(2)}", "x⁽²⁾"}, {"x^(", "x⁽"}, {"a_=", "a₌"},
		{"x^{beta}", "x\u1d5d"}, {"x^beta", "x\u1d5d"}, {`x_{\gamma}`, "x\u1d67"},
		{"mitX_1", "𝑋₁"}, {"alpha_1^2", "α₁²"},
	})
}

func TestScriptsWithoutAUnicodeForm(t *testing.T) {
	// subscript c, z, q do not exist in Unicode; the marker stays as typed
	run(t, Options{}, []tc{
		{"x_c", "x_c"}, {"x_{cz}", "x_{cz}"}, {"x^{S}", "x^{S}"}, {"x^", "x^"}, {"x_", "x_"}, {"x^!", "x\ua71d"},
		{"x^,", "x^,"}, {"2^", "2^"},
	})
}

func TestArgumentCommands(t *testing.T) {
	run(t, fuzzyOn, []tc{
		// fractions
		{"frac{1}{2}", "½"}, {"dfrac{3}{4}", "¾"}, {"frac12", "½"}, {"frac{ 1 }{ 2 }", "½"},
		{"frac{12}{35}", "¹²\u2044₃₅"}, {"frac{n+1}{2}", "ⁿ⁺¹\u2044₂"}, {"frac{alpha}{beta}", "α/β"},
		{"frac{a+b}{c}", "(a+b)/c"}, {"frac{x}", "frac{x}"}, // second argument missing: not a command
		// roots
		{"sqrt{2}", "√2"}, {"sqrt{25}", "√25"}, {"sqrt{x}", "√x"}, {"sqrt{x+1}", "√(x+1)"},
		{"sqrt[3]{8}", "∛8"}, {"sqrt[4]{x}", "∜x"}, {"sqrt[n]{x}", "ⁿ√x"}, {"sqrt[2]{x}", "√x"},
		// math alphabets
		{"mathbb{R}", "ℝ"}, {"mathbb{RQ}", "ℝℚ"}, {"mathbb{1}", "𝟙"}, {"mathcal{A}", "𝒜"}, {"mathfrak{g}", "𝔤"},
		{"mathbf{x}", "𝐱"}, {"mathbf{Gamma}", "𝚪"}, {`mathbf{\alpha}`, "𝛂"}, {"Bbb{R}", "ℝ"}, {"mbf{Mu}", "𝚳"},
		{"mathbb{x y}", "𝕩 𝕪"}, {"mathbb{R+}", "ℝ+"},
		// verbatim text
		{"text{ and }", " and "}, {"x text{ and } y", "x  and  y"}, {"text{alpha}", "alpha"}, {"mathrm{d}x", "dx"},
	})
}

func TestAccents(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"hat{a}", "\u00e2"}, {"hat{x}", "x\u0302"}, {"bar{x}", "x\u0304"}, {"vec{v}", "v\u20d7"}, {"dot{x}", "\u1e8b"},
		{"tilde{n}", "\u00f1"}, {"acute{e}", "\u00e9"}, {"grave{a}", "\u00e0"}, {"ddot{o}", "\u00f6"}, {"check{c}", "\u010d"},
		{"breve{a}", "\u0103"}, {"hat{alpha}", "\u03b1\u0302"}, {"acute{alpha}", "\u03ac"},
		{"not{=}", "\u2260"}, {"not{in}", "\u2209"}, {"not{x}", "x\u0338"},
		{"widehat{xy}", "x\u0302\u0177"}, {"overline{xy}", "x\u0305y\u0305"}, {"underline{a}", "a\u0332"},
		{"hat{xy}", "x\u0177"},                                        // narrow accents sit on the last character
		{"hat{}", "hat{}"}, {`\u{}`, "˘"}, {"mathbb{ }", "mathbb{ }"}, // a blank argument is no accent or alphabet
		{`\c{c}`, "\u00e7"}, {`\v{s}`, "\u0161"}, {`\u{a}`, "\u0103"}, {`\H{o}`, "\u0151"}, {`\k{a}`, "\u0105"},
		// the LaTeX text accents
		{`\"{o}`, "\u00f6"}, {`\"o`, "\u00f6"}, {`\'e`, "\u00e9"}, {`\'{e}`, "\u00e9"}, {"\\`a", "\u00e0"}, {`\^o`, "\u00f4"},
		{`\~n`, "\u00f1"}, {`\=a`, "\u0101"}, {`\.z`, "\u017c"}, {`\^{alpha}`, "\u03b1\u0302"},
		// nested
		{"hat{tilde{a}}", "\u00e3\u0302"}, {"frac{hat{a}}{2}", "\u00e2/2"},
	})
}

// A wide accent goes once on each character, after the combining marks it already has, and
// never on a combining mark itself.
func TestAccentsKeepExistingCombiningMarks(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"overline{x\u0302}", "x\u0302\u0305"},         // not x\u0305\u0302\u0305
		{"widehat{acute{x}}", "x\u0301\u0302"},         // not an extra circumflex on the acute
		{"overline{x\u0302y}", "x\u0302\u0305y\u0305"}, // each character once, marks stay with their character
		{"underline{a b}", "a\u0332 b\u0332"},          // spaces still separate
		{"hat{acute{x}}", "x\u0301\u0302"},             // narrow: the last character, with its marks
		{"hat{xy\u0301}", "xy\u0301\u0302"},
		{"overline{\u0301}", "\u0301\u0305"},  // a mark with nothing to sit on is its own character
		{"hat{acute{alpha}}", "\u03ac\u0302"}, // precomposed letters are still accented
		{"hat{a}", "\u00e2"},                  // ... and still composed when nothing is there yet
	})
}

// Argument commands are only recognised when their argument follows directly,
// so the same words keep working as symbols and as ordinary words.
func TestCommandsWithoutArguments(t *testing.T) {
	run(t, fuzzyOn, []tc{
		{"bar", "bar"}, {"hat", "hat"}, {"frac", "frac"}, {"text", "text"},
		{"bar {x}", "bar {x}"}, // a space separates the argument
		{"hat{", "hat{"}, {"frac{1", "frac{1"}, {"hat{x", "hat{x"},
		{"tilde", "\u0303"}, // still a symbol: the combining mark
		{"sqrt", "\u221a"},  // ... and the radical sign
		{"frac{1}x", "frac{1}x"},
	})
}

func TestEscapes(t *testing.T) {
	run(t, Options{}, []tc{
		{`\{x\}`, "{x}"}, {`50\%`, "50%"}, {`\$5`, "$5"}, {`a\_b`, "a_b"}, {`\#1`, "#1"}, {`a \& b`, "a & b"},
		{`\\`, `\\`}, {`end\`, `end\`}, {`\ `, `\ `}, {`\1`, `\1`},
	})
}

func TestUnknownNamesAreKeptAndReported(t *testing.T) {
	r := Convert(`\foo \qux{x} baz \foo`, Options{})
	if r.Output != `\foo \qux{x} baz \foo` {
		t.Errorf("Output = %q", r.Output)
	}
	want := []Problem{{"foo", true}, {"qux", true}, {"baz", false}}
	if !reflect.DeepEqual(r.Problems, want) {
		t.Errorf("Problems = %v, want %v", r.Problems, want)
	}
	if r := Convert("alpha beta", Options{}); len(r.Problems) != 0 {
		t.Errorf("unexpected problems %v", r.Problems)
	}
}

func TestTextMode(t *testing.T) {
	text := Options{Mode: Text, Fuzzy: true}
	// words are never converted, however much they look like symbol names
	for _, in := range []string{
		"the value of alpha is large", "to be or not to be", "a star in the sum of times", "snake_case and file_1.txt",
		"gammma", "bbbR", "x_n", "a_b_c", "frac{1}{2}", "hat{x}", "mathbb{R}", "I am here", "alpha_1",
	} {
		if got := conv(t, text, in); got != in {
			t.Errorf("text mode changed %q into %q", in, got)
		}
	}
	// explicit syntax is
	run(t, text, []tc{
		{`the value of \alpha is large`, "the value of α is large"},
		{`\alpha and \beta`, "α and β"}, // a gap between two converted pieces is closed only if nothing separates them
		{`\mathbb{R}`, "ℝ"}, {`\frac{1}{2}`, "½"}, {`\hat{x}`, "x\u0302"}, {`\"{o}`, "\u00f6"},
		{"x^2", "x²"}, {"x^{n}", "xⁿ"}, {"x_{12}", "x₁₂"}, {"x^-1", "x⁻¹"},
		{`caf\'e au lait`, "caf\u00e9 au lait"},
		{"C:\\Users\\foo", "C:\\Users\\foo"}, // unknown commands are kept
	})
}

func TestStrictMode(t *testing.T) {
	strict := Options{Strict: true, Fuzzy: true}
	run(t, strict, []tc{{"gamma", "γ"}, {`\gamma`, "γ"}, {"alphabeta", "alphabeta"}, {"bbbR", "bbbR"}, {"gammma", "gammma"}})
	r := Convert("gammma bbbR", strict)
	want := []Problem{{"gammma", false}, {"bbbR", false}}
	if !reflect.DeepEqual(r.Problems, want) {
		t.Errorf("Problems = %v, want %v", r.Problems, want)
	}
}

func TestFuzzyIsOptIn(t *testing.T) {
	if got := conv(t, Options{}, "gammma"); got != "gammma" {
		t.Errorf("without Fuzzy, got %q", got)
	}
	run(t, fuzzyOn, []tc{
		{"gammma", "γ"}, {"alhpa", "α"}, {"thetaa", "θ"},
		{"foo", "foo"}, // too short to correct
		{"zzzzzzzz", "zzzzzzzz"},
	})
	// a slip of case is repaired without Fuzzy: it is not a typo
	run(t, Options{}, []tc{{"bbbR", "ℝ"}, {"GAMMA", "Γ"}, {"bbbr", "bbbr"}})
	// ... but a word in lower case only has to be exact: these are words, not the symbols And, Or, Top, Times
	run(t, Options{}, []tc{{"and", "and"}, {"or", "or"}, {"times", "×"}, {"top", "⊤"}, {"this and that", "this and that"}})
	// and so is gluing names together
	run(t, Options{}, []tc{{"alphabeta", "αβ"}, {"mbfGammamitX", "𝚪𝑋"}})
}

func TestDigitsDoNotGetEaten(t *testing.T) {
	// an unknown word followed by digits must never be "corrected" into a symbol
	// that drops them
	run(t, fuzzyOn, []tc{{"alpha2", "α2"}, {"2alpha", "2α"}, {"H2O", "H2O"}, {"beta10", "β10"}, {"alpha_2", "α₂"}})
}

func TestDeterministic(t *testing.T) {
	// the first implementation depended on map iteration order
	in := "foo xyzzy hello gammma thetaa qwerty abcde ab alpha beta neq _0 x^2 frac{1}{3} hat{a}"
	first := conv(t, fuzzyOn, in)
	for i := 0; i < 300; i++ {
		if got := conv(t, fuzzyOn, in); got != first {
			t.Fatalf("run %d: %q differs from %q", i, got, first)
		}
	}
}

func TestDepthLimit(t *testing.T) {
	in := strings.Repeat("hat{", 5000) + "x" + strings.Repeat("}", 5000)
	_ = conv(t, Options{}, in) // must neither overflow the stack nor take forever
}

func TestUnbalancedAndOddInput(t *testing.T) {
	for _, in := range []string{
		"{", "}", "{{", "}}", "hat{", "frac{1}{", "sqrt[", "sqrt[3", "sqrt[3]", `\`, `\\\`, "^", "_", "^{", "_{", "^{}", "x^{",
		"\x00", "\u00a0", "\u200b", `\"`, `\"{`, "\\'", "sqrt[]{x}", "frac{}{}", "mathbb{}", "text{}", "𝚪", "\t\n\r",
	} {
		out := conv(t, fuzzyOn, in)
		if !utf8.ValidString(out) {
			t.Errorf("Convert(%q) produced invalid UTF-8", in)
		}
	}
	if got := conv(t, Options{}, "text{}"); got != "" {
		t.Errorf(`text{} = %q`, got)
	}
}

func FuzzConvert(f *testing.F) {
	for _, s := range []string{
		"mbfGamma mitX_1", `\alpha+\beta`, "frac{1}{2} sqrt[3]{x}", "hat{a} \\\"{o}", "x^{12} y_{ij}", "hello world", "{{}}", "\\\\{",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !utf8.ValidString(s) {
			t.Skip()
		}
		for _, opt := range []Options{{Fuzzy: true}, {Mode: Text}, {Strict: true}, {Join: true}} {
			a := Convert(s, opt)
			if !utf8.ValidString(a.Output) {
				t.Fatalf("invalid UTF-8 output %q for %q", a.Output, s)
			}
			if b := Convert(s, opt); b.Output != a.Output {
				t.Fatalf("not deterministic for %q: %q then %q", s, a.Output, b.Output)
			}
		}
	})
}

// ---- reverse -----------------------------------------------------------------------

func TestReverse(t *testing.T) {
	del := symbols.Default().Reverse("∇")[0]
	sqrtName := symbols.Default().Reverse("√")[0]
	tests := []struct {
		in, want string
		bs       bool
	}{
		{"γ", "gamma", false},
		{"𝚪𝑋", "mbfGamma mitX", false},
		{"x < y", "x < y", false}, // ASCII is never replaced
		{"αβ", `\alpha\beta`, true},
		{"unknown \ue000?", "unknown \ue000?", false}, // private use: no name
		{"", "", false},
		// names next to letters and digits are kept apart from them
		{"∇f", del + "{}f", false}, // closed with an empty group ...
		{"fγ", `f\gamma`, false},   // ... or opened with a backslash
		{"2γ", `2\gamma`, false},
		{"γ2", `gamma{}2`, false},
		{"²3", `^2{}3`, false},
		{"Ł", `\L`, false}, // one letter names always need the backslash
		{"ŁŁ", `\L\L`, false},
		{"Łx", `\L{}x`, false},
		{"x Ł", `x \L`, false},
		// names that are also commands, next to literal braces
		{"√{x}", sqrtName + `{}{x}`, false},
		{"√[3]{x}", sqrtName + `{}[3]{x}`, false},
		{"γ{}", `gamma{}\{\}`, false}, // a literal empty group must not be swallowed
		{"γ{x}", `gamma{x}`, false},   // gamma is no command: nothing to guard against
		// symbols after ^ _ \ keep them literal
		{"^γ", `^\gamma`, false},
		{"_γ", `_\gamma`, false},
		{"x^γ", `x^\gamma`, false},
		{`\∇`, `\∇`, false}, // no name can follow a lone backslash: the symbol stays
	}
	for _, tt := range tests {
		if got := Reverse(tt.in, ReverseOptions{Backslash: tt.bs}); got != tt.want {
			t.Errorf("Reverse(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	// the name of ∈ is whatever the database prefers; only check the shape
	if got := Reverse("a ∈ ℝ", ReverseOptions{}); !strings.HasSuffix(got, "BbbR") || strings.ContainsAny(got, "∈ℝ") {
		t.Errorf(`Reverse("a ∈ ℝ") = %q`, got)
	}
}

// Convert(Reverse(x)) is x again, exactly: every symbol on its own and between letters, digits
// and other text. (An earlier version of this test ignored spaces, which hid that "∇x" came back
// as "∇ x".)
func TestReverseRoundTrip(t *testing.T) {
	contexts := [][2]string{
		{"", ""}, {"x", ""}, {"", "x"}, {"x", "y"}, {"2", ""}, {"", "3"}, {"2", "3"}, {"(", ")"}, {"x ", " y"},
		{"", "{x}"}, {"", "{}"}, {"", "[3]{x}"}, {"", "{}{x}"}, {"^", ""}, {"_", ""}, {`\`, ""}, {"x^", "y"}, {"{", "}"},
	}
	seen := map[string]bool{}
	failures := 0
	for _, e := range symbols.Default().Entries() {
		v := e.Value
		if seen[v] || (len(v) == 1 && v[0] < utf8.RuneSelf) {
			continue
		}
		seen[v] = true
		for _, ctx := range contexts {
			in := ctx[0] + v + ctx[1]
			if Convert(in, Options{Strict: true}).Output != in {
				continue // text that lutf itself would convert cannot come back unchanged
			}
			for _, bs := range []bool{false, true} {
				rev := Reverse(in, ReverseOptions{Backslash: bs})
				if got := Convert(rev, Options{Strict: true}).Output; got != in {
					if failures++; failures <= 20 {
						t.Errorf("Reverse(%q, backslash=%v) = %q, which converts to %q", in, bs, rev, got)
					}
				}
			}
		}
	}
	if failures > 20 {
		t.Errorf("... and %d more", failures-20)
	}
	// longer texts
	for _, in := range []string{"x∈ℝ and γ", "αβγ", "fγ(x)", "ŁÓDŹ", "∇f·∇g", "α²+β²=γ²", "x₁₂≤y³", "x ≤ y"} {
		for _, bs := range []bool{false, true} {
			rev := Reverse(in, ReverseOptions{Backslash: bs})
			if got := Convert(rev, Options{Strict: true}).Output; got != in {
				t.Errorf("round trip of %q via %q gave %q", in, rev, got)
			}
		}
	}
}

// Random texts made of symbols and of the characters Convert treats specially. For every one that
// Convert leaves unchanged, Reverse followed by Convert must give it back. (Whitespace is left out:
// a space between two symbols is the documented exception, and spaces are covered above.)
func TestReverseRoundTripRandom(t *testing.T) {
	pieces := []string{
		"γ", "∇", "√", "²", "ₙ", "Ł", "\u0301", "˘", "ℝ", "𝚪", "½", "≠", "∉", "ᵞ", "…", "α", "ⁿ",
		"x", "y", "2", "3", "a", "e", "^", "_", `\`, "{", "}", "{}", "[", "]", "(", ")", "'", `"`, "=", "+", "-", "alpha", "hat", "sqrt", "u",
	}
	rng := rand.New(rand.NewSource(7))
	checked, failures := 0, 0
	for n := 0; n < 60000; n++ {
		var b strings.Builder
		for k := 1 + rng.Intn(6); k > 0; k-- {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		in := b.String()
		if Convert(in, Options{Strict: true}).Output != in {
			continue
		}
		checked++
		for _, bs := range []bool{false, true} {
			rev := Reverse(in, ReverseOptions{Backslash: bs})
			if got := Convert(rev, Options{Strict: true}).Output; got != in {
				if failures++; failures <= 25 {
					t.Errorf("Reverse(%q, backslash=%v) = %q, which converts to %q", in, bs, rev, got)
				}
			}
		}
	}
	if failures > 25 {
		t.Errorf("... and %d more", failures-25)
	}
	if checked < 5000 {
		t.Errorf("only %d of the random texts were fixed points; the generator no longer exercises the property", checked)
	}
}

// Realistic texts: every symbol gets a name, and the result converts back to the text.
func TestReverseRealisticTexts(t *testing.T) {
	named := []string{"αβγ", "fγ(x)", "∇f·∇g", "α²+β²=γ²", "a≠b", "A∩B⊆A∪B", "f:X→Y", "∀x∈ℝ: x²≥0", "cosθ+sinθ", "ℝ→ℝ", "x₁₂≤y³", "ΔxΔp≥ℏ/2", "√2≈1.414", "𝚪(n)=(n-1)!"}
	for _, in := range named {
		out := Reverse(in, ReverseOptions{})
		for _, r := range out {
			if r >= utf8.RuneSelf {
				t.Errorf("Reverse(%q) = %q still has the symbol %q", in, out, r)
				break
			}
		}
		if got := Convert(out, Options{Strict: true}).Output; got != in {
			t.Errorf("Reverse(%q) = %q, which converts to %q", in, out, got)
		}
	}
	// mixed with plain letters the names stay out of the way of the words
	if got := Reverse("cosθ+sinθ", ReverseOptions{}); got != `cos\theta+sin\theta` {
		t.Errorf("got %q", got)
	}
	if got := Reverse("fγ(x)", ReverseOptions{}); got != `f\gamma(x)` {
		t.Errorf("got %q", got)
	}
}

// Texts for which the fences are not enough. lutf leaves them alone, so Reverse must give them back
// as they were: it falls back to leaving the words that mix scripts as they are, or, failing that,
// to changing nothing, rather than writing names that would be read differently.
func TestReverseSafetyNet(t *testing.T) {
	for _, in := range []string{
		"2ₙtext{}", "ⁿtext{}", "²ⁿtext{}", // text{} would become the command text
		"2αnote8thétext", "γ∇γnote8thé", // note8th is a name once α or γ stops gluing it to a word
		"^{a∇}", "∇ₙtext{}", "e^{iπ}+1=0", // names inside a script
	} {
		if Convert(in, Options{Strict: true}).Output != in {
			t.Fatalf("%q is not a text lutf leaves alone; the test is wrong", in)
		}
		for _, bs := range []bool{false, true} {
			rev := Reverse(in, ReverseOptions{Backslash: bs})
			if got := Convert(rev, Options{Strict: true}).Output; got != in {
				t.Errorf("Reverse(%q, backslash=%v) = %q, which converts to %q", in, bs, rev, got)
			}
		}
	}
	// the last resort changes nothing at all
	if got := Reverse("^{a∇}", ReverseOptions{}); got != "^{a∇}" {
		t.Errorf("got %q", got)
	}
	// text that lutf would convert by itself is none of the net's business
	if got := Reverse("alpha ∇", ReverseOptions{}); !strings.HasPrefix(got, "alpha ") || strings.Contains(got, "∇") {
		t.Errorf("got %q", got)
	}
}

// Reverse works with the symbol set it is given, not only the built-in one, also when it has to
// decide whether letters next to a symbol would turn into a name.
func TestReverseUsesTheGivenSet(t *testing.T) {
	set, err := symbols.Parse(symbols.Sources{
		Symbols:  strings.NewReader("foo\tγ\nquux\tQ\n"),
		Commands: strings.NewReader("text\ttext\t\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// quux is a name only in this set: written bare after γ it would be read as the symbol Q,
	// so it has to be fenced with text{}
	in := "γquux"
	rev := Reverse(in, ReverseOptions{Set: set})
	if rev != "foo{}text{quux}" {
		t.Errorf("Reverse(%q) = %q, want foo{}text{quux}", in, rev)
	}
	if got := Convert(rev, Options{Set: set, Strict: true}).Output; got != in {
		t.Errorf("%q converts to %q", rev, got)
	}
}

// Unicode spaces that are symbols (quad, thinspace, ...) are named and come back exactly; other
// Unicode whitespace between two symbols is dropped by Convert, like an ASCII space.
func TestReverseWithUnicodeWhitespace(t *testing.T) {
	for _, in := range []string{"α\u2003β", "x\u2009y", "\u2003x", "x\u2003", "α\u2003\u2003β", "x\u00a0y", "\u00a0x", "α\u00a0x", "x\u00a0β"} {
		if Convert(in, Options{Strict: true}).Output != in {
			t.Fatalf("%q is not left alone by Convert; the test is wrong", in)
		}
		rev := Reverse(in, ReverseOptions{})
		if got := Convert(rev, Options{Strict: true}).Output; got != in {
			t.Errorf("Reverse(%q) = %q, which converts to %q", in, rev, got)
		}
	}
	// the symbols are named, not copied
	if got := Reverse("α\u2003β", ReverseOptions{}); strings.ContainsRune(got, '\u2003') {
		t.Errorf("quad was not named: %q", got)
	}
	// a space that is no symbol, between two symbols: both are named, the space cannot come back
	rev := Reverse("α\u00a0β", ReverseOptions{})
	if strings.ContainsAny(rev, "αβ") {
		t.Errorf("symbols not named: %q", rev)
	}
	if got := Convert(rev, Options{Strict: true}).Output; got != "αβ" {
		t.Errorf("%q converts to %q", rev, got)
	}
}

// The one thing a round trip cannot keep: lutf drops whitespace between two converted
// symbols (that is how "mbfGamma mitX _2" becomes 𝚪𝑋₂), so a space between two symbols
// in the original is gone after Reverse and Convert.
func TestReverseRoundTripLosesASpaceBetweenSymbols(t *testing.T) {
	// the symbols are still named: only the space cannot come back
	rev := Reverse("α β", ReverseOptions{})
	if rev != "alpha beta" {
		t.Errorf("Reverse = %q, want alpha beta", rev)
	}
	if got := Convert(rev, Options{Strict: true}).Output; got != "αβ" {
		t.Errorf("%q converts to %q", rev, got)
	}
	// a space next to anything else, and a line break, are kept
	for _, in := range []string{"α x", "x β", "α\nβ", "α\r\nβ"} {
		if got := Convert(Reverse(in, ReverseOptions{}), Options{Strict: true}).Output; got != in {
			t.Errorf("round trip of %q gave %q", in, got)
		}
	}
}

// An empty group straight after a converted piece ends it and disappears.
func TestEmptyGroupEndsAName(t *testing.T) {
	run(t, Options{}, []tc{
		{"alpha{}x", "αx"}, {`\alpha{}x`, "αx"}, {"alpha{}2", "α2"}, {"alpha{} beta", "αβ"},
		{"^2{}3", "²3"}, {"x_1{}2", "x₁2"}, {"frac{1}{2}{}x", "½x"}, {"alpha{}{}beta", "αβ"},
		// nothing to end: the braces stay
		{"x{}", "x{}"}, {"{}", "{}"}, {"hello{}", "hello{}"}, {"alpha {}", "α {}"}, {"alpha{x}", "α{x}"},
	})
	run(t, Options{Mode: Text}, []tc{{`\alpha{}x`, "αx"}, {"word{}", "word{}"}})
}

func TestDescribe(t *testing.T) {
	infos := Describe("γ x", nil)
	if len(infos) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(infos), infos)
	}
	g := infos[0]
	if g.Text != "γ" || !reflect.DeepEqual(g.Runes, []rune{0x3b3}) || !reflect.DeepEqual(g.Bytes, []byte{0xce, 0xb3}) {
		t.Errorf("bad description of γ: %+v", g)
	}
	if len(g.Names) == 0 || g.Names[0] != "gamma" {
		t.Errorf("names of γ: %v", g.Names)
	}
	if len(infos[1].Names) != 0 {
		t.Errorf("x should have no names, got %v", infos[1].Names)
	}
}

// ---- benchmarks ----------------------------------------------------------------------

func benchInput(words int) string {
	base := []string{"alpha", "beta", "mbfGamma", "mitX_1", "neq", "unknownword", "x^2", "frac{1}{2}", "hello", "gammma"}
	var b strings.Builder
	for i := 0; i < words; i++ {
		b.WriteString(base[i%len(base)])
		b.WriteByte(' ')
	}
	return b.String()
}

func BenchmarkConvert(b *testing.B) {
	for _, n := range []int{1, 100, 1000} {
		in := benchInput(n)
		b.Run(fmt.Sprintf("%dwords", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				Convert(in, fuzzyOn)
			}
		})
	}
}
