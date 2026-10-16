package fuzzy

import (
	"math/rand"
	"testing"
	"unicode"
)

func TestDistance(t *testing.T) {
	const big = 1000
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"gamma", "gamma", 0},
		{"gamma", "gammma", Edit},      // insertion
		{"gammma", "gamma", Edit},      // deletion
		{"gamma", "gamna", Edit},       // substitution
		{"gamma", "gmama", Edit},       // adjacent transposition counts once
		{"alpha", "alhpa", Edit},       // transposition
		{"bbbR", "BbbR", CaseEdit},     // case only
		{"bbbR", "Bbbr", 2 * CaseEdit}, // case only, twice
		{"abc", "", 3 * Edit},
		{"", "abc", 3 * Edit},
		{"kitten", "sitting", 3 * Edit},
		{"αβγ", "αβδ", Edit}, // rune based, not byte based
		{"é", "e", Edit},     // one rune each; not two bytes against one
	}
	for _, tt := range tests {
		if got := Distance(tt.a, tt.b, big); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		if got := Distance(tt.b, tt.a, big); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, want %d (symmetry)", tt.b, tt.a, got, tt.want)
		}
	}
}

func TestDistanceLimit(t *testing.T) {
	// beyond the limit the result is exactly limit+1, never anything else
	if got := Distance("alpha", "omega", Edit); got != Edit+1 {
		t.Errorf("over limit: got %d, want %d", got, Edit+1)
	}
	// at the limit the exact value is returned
	if got := Distance("alpha", "alpa", Edit); got != Edit {
		t.Errorf("at limit: got %d, want %d", got, Edit)
	}
	// length difference alone can exceed the limit
	if got := Distance("a", "abcdef", 3*Edit); got != 3*Edit+1 {
		t.Errorf("length gap: got %d, want %d", got, 3*Edit+1)
	}
	// empty operands honour the limit too
	if got := Distance("", "abcd", Edit); got != Edit+1 {
		t.Errorf("empty: got %d, want %d", got, Edit+1)
	}
}

// reference is a straightforward full-matrix implementation of the same metric.
func reference(a, b []rune) int {
	d := make([][]int, len(a)+1)
	for i := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i * Edit
	}
	for j := range d[0] {
		d[0][j] = j * Edit
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			sub := Edit
			if a[i-1] == b[j-1] {
				sub = 0
			} else if unicode.ToLower(a[i-1]) == unicode.ToLower(b[j-1]) {
				sub = CaseEdit
			}
			v := min(d[i-1][j]+Edit, d[i][j-1]+Edit, d[i-1][j-1]+sub)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				v = min(v, d[i-2][j-2]+Edit)
			}
			d[i][j] = v
		}
	}
	return d[len(a)][len(b)]
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	alphabet := []rune("abAB")
	word := func() []rune {
		w := make([]rune, rng.Intn(8))
		for i := range w {
			w[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return w
	}
	var s Scratch // deliberately reused, to catch stale-row bugs
	for n := 0; n < 20000; n++ {
		a, b := word(), word()
		full := reference(a, b)
		for _, limit := range []int{0, 1, 2, 3, 4, 6, 100} {
			want := full
			if want > limit {
				want = limit + 1
			}
			if got := s.Distance(a, b, limit); got != want {
				t.Fatalf("Distance(%q, %q, %d) = %d, want %d (full %d)", string(a), string(b), limit, got, want, full)
			}
		}
	}
}

func within(query, text string, limit int, fold bool) int {
	var s Scratch
	return s.Within([]rune(query), []rune(text), limit, fold)
}

func TestWithin(t *testing.T) {
	const big = 1000
	tests := []struct {
		query, text string
		fold        bool
		want        int
	}{
		{"gamma", "gamma", false, 0},
		{"gamma", "gammadot", false, 0},     // a part of the text is enough
		{"gamma", "bigamma", false, 0},      // ... wherever it is
		{"gamme", "gamma", false, Edit},     // one mistake
		{"gamme", "gammadot", false, Edit},  // ... in a longer text
		{"alhpa", "alpha", false, Edit},     // a transposition is one mistake
		{"Gamma", "gamma", false, CaseEdit}, // case alone is half a mistake
		{"Gamma", "gamma", true, 0},         // ... or nothing at all
		{"GAMMA", "bigamma", true, 0},
		{"gamma", "", false, 5 * Edit}, // nothing to match in
		{"", "anything", false, 0},     // nothing to look for
		{"nabal", "nabla", false, Edit},
		{"xyz", "gamma", false, 3 * Edit}, // no letter in common
		{"αβγ", "xαβγy", false, 0},        // runes, not bytes
	}
	for _, tt := range tests {
		if got := within(tt.query, tt.text, big, tt.fold); got != tt.want {
			t.Errorf("Within(%q, %q, fold=%v) = %d, want %d", tt.query, tt.text, tt.fold, got, tt.want)
		}
	}
	// the limit works like Distance's
	if got := within("gamma", "omega", Edit, false); got != Edit+1 {
		t.Errorf("over limit: got %d, want %d", got, Edit+1)
	}
}

// Within is the smallest Distance to any run of consecutive runes of the text.
func TestWithinAgainstBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	alphabet := []rune("abAB")
	word := func(max int) []rune {
		w := make([]rune, rng.Intn(max))
		for i := range w {
			w[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return w
	}
	var s Scratch // reused on purpose
	for n := 0; n < 20000; n++ {
		q, text := word(6), word(9)
		if len(q) == 0 {
			continue
		}
		want := len(q) * Edit // the empty run costs one insertion per rune
		for i := 0; i <= len(text); i++ {
			for j := i; j <= len(text); j++ {
				want = min(want, reference(q, text[i:j]))
			}
		}
		for _, limit := range []int{0, 2, 4, 100} {
			w := want
			if w > limit {
				w = limit + 1
			}
			if got := s.Within(q, text, limit, false); got != w {
				t.Fatalf("Within(%q, %q, %d) = %d, want %d (full %d)", string(q), string(text), limit, got, w, want)
			}
		}
	}
}

func BenchmarkDistance(b *testing.B) {
	x, y := []rune("mbfitsansGamma"), []rune("mbfitsansGammma")
	var s Scratch
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Distance(x, y, 2*Edit)
	}
}
