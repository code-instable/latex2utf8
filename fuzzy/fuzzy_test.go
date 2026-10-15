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

func BenchmarkDistance(b *testing.B) {
	x, y := []rune("mbfitsansGamma"), []rune("mbfitsansGammma")
	var s Scratch
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Distance(x, y, 2*Edit)
	}
}
