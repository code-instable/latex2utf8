package convert

import (
	"fmt"
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
		{"hat{xy}", "x\u0177"}, // narrow accents sit on the last character
		{"hat{}", "\u0302"},
		{`\c{c}`, "\u00e7"}, {`\v{s}`, "\u0161"}, {`\u{a}`, "\u0103"}, {`\H{o}`, "\u0151"}, {`\k{a}`, "\u0105"},
		// the LaTeX text accents
		{`\"{o}`, "\u00f6"}, {`\"o`, "\u00f6"}, {`\'e`, "\u00e9"}, {`\'{e}`, "\u00e9"}, {"\\`a", "\u00e0"}, {`\^o`, "\u00f4"},
		{`\~n`, "\u00f1"}, {`\=a`, "\u0101"}, {`\.z`, "\u017c"}, {`\^{alpha}`, "\u03b1\u0302"},
		// nested
		{"hat{tilde{a}}", "\u00e3\u0302"}, {"frac{hat{a}}{2}", "\u00e2/2"},
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
	tests := []struct {
		in, want string
		bs       bool
	}{
		{"γ", "gamma", false},
		{"𝚪𝑋", "mbfGamma mitX", false},
		{"∇f", symbols.Default().Reverse("∇")[0] + " f", false},
		{"fγ", "f gamma", false},
		{"x < y", "x < y", false}, // ASCII is never replaced
		{"αβ", `\alpha\beta`, true},
		{"Ł", `\L`, false}, // one letter names always need the backslash
		{"ŁŁ", `\L\L`, false},
		{"Łx", `\L x`, false},
		{"a ∈ ℝ", "a in BbbR", false},
		{"unknown \ue000?", "unknown \ue000?", false}, // private use: no name
		{"", "", false},
	}
	for _, tt := range tests {
		got := Reverse(tt.in, ReverseOptions{Backslash: tt.bs})
		if tt.in == "a ∈ ℝ" {
			// the preferred name of ∈ is the shortest one; only check the shape
			if !strings.HasSuffix(got, "BbbR") || strings.ContainsAny(got, "∈ℝ") {
				t.Errorf("Reverse(%q) = %q", tt.in, got)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("Reverse(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Reverse followed by Convert gives back what it started with.
func TestReverseRoundTrip(t *testing.T) {
	set := symbols.Default()
	seen := map[string]bool{}
	for _, e := range set.Entries() {
		v := e.Value
		if seen[v] || (len(v) == 1 && v[0] < utf8.RuneSelf) {
			continue
		}
		seen[v] = true
		for _, bs := range []bool{false, true} {
			name := Reverse(v, ReverseOptions{Backslash: bs})
			if got := Convert(name, Options{Strict: true}).Output; got != v {
				t.Errorf("Reverse(%q) = %q, which converts to %q", v, name, got)
			}
		}
	}
	// and in context
	for _, in := range []string{"x∈ℝ and γ", "αβγ", "fγ(x)", "ŁÓDŹ"} {
		for _, bs := range []bool{false, true} {
			rev := Reverse(in, ReverseOptions{Backslash: bs})
			if got := Convert(rev, Options{Strict: true}).Output; strings.ReplaceAll(got, " ", "") != strings.ReplaceAll(in, " ", "") {
				t.Errorf("round trip of %q via %q gave %q", in, rev, got)
			}
		}
	}
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
