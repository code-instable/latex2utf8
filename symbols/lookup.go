package symbols

import (
	"math/bits"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/code-instable/latex2utf8/fuzzy"
)

// MinSegment is the shortest name Segment will use as a piece of a longer
// word. Shorter names (mu, pi, in, to, ...) are common letter sequences and
// would split ordinary words.
const MinSegment = 3

// MinFuzzyName is the shortest name Fuzzy proposes as a correction.
const MinFuzzyName = 3

type candidate struct {
	name  string
	runes []rune
	sig   uint32 // letters present, see signature
}

// signature is a bit set of the ASCII letters of w, ignoring case. One edit
// changes at most two bits of it, so two words within n edits of each other
// have signatures differing in at most 2n bits; Fuzzy uses that to skip most
// candidates without computing a distance.
func signature(w []rune) uint32 {
	var sig uint32
	for _, r := range w {
		switch {
		case r >= 'a' && r <= 'z':
			sig |= 1 << (r - 'a')
		case r >= 'A' && r <= 'Z':
			sig |= 1 << (r - 'A')
		}
	}
	return sig
}

// Fold finds the name equal to word up to letter case. When several names
// qualify it keeps the one differing from word in the fewest letters, and
// gives up if that is not unique (or the candidates mean different symbols).
func (s *Set) Fold(word string) (name, value string, ok bool) {
	s.foldOnce.Do(func() {
		s.fold = make(map[string][]string)
		for n := range s.values {
			if allLetters(n) {
				l := strings.ToLower(n)
				s.fold[l] = append(s.fold[l], n)
			}
		}
		for _, names := range s.fold {
			sort.Strings(names)
		}
	})
	cands := s.fold[strings.ToLower(word)]
	if len(cands) == 0 {
		return "", "", false
	}
	best, bestDiff := []string(nil), len(word)+1
	for _, c := range cands {
		d := caseDiff(c, word)
		switch {
		case d < bestDiff:
			best, bestDiff = []string{c}, d
		case d == bestDiff:
			best = append(best, c)
		}
	}
	v := s.values[best[0]]
	for _, c := range best[1:] {
		if s.values[c] != v {
			return "", "", false
		}
	}
	return best[0], v, true
}

func caseDiff(a, b string) int {
	n := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

// Segment splits a letters-only word into names of at least MinSegment
// letters and returns their values concatenated. It uses the fewest pieces,
// preferring longer pieces first, and fails unless the whole word is covered.
func (s *Set) Segment(word string) (string, bool) {
	n := len(word)
	if n < MinSegment {
		return "", false
	}
	const inf = 1 << 30
	best := make([]int, n+1)
	next := make([]int, n+1)
	for i := n - 1; i >= 0; i-- {
		best[i] = inf
		for l := min(s.maxLen, n-i); l >= MinSegment; l-- {
			if best[i+l] >= inf {
				continue
			}
			if _, ok := s.values[word[i:i+l]]; !ok {
				continue
			}
			if c := best[i+l] + 1; c < best[i] {
				best[i], next[i] = c, l
			}
		}
	}
	if best[0] >= inf {
		return "", false
	}
	var b strings.Builder
	for i := 0; i < n; i += next[i] {
		b.WriteString(s.values[word[i:i+next[i]]])
	}
	return b.String(), true
}

func (s *Set) buildFuzzy() {
	s.fuzzyOnce.Do(func() {
		s.byLen = make(map[int][]int)
		for n := range s.values {
			if allLetters(n) && len(n) >= MinFuzzyName {
				r := []rune(n)
				s.cands = append(s.cands, candidate{name: n, runes: r, sig: signature(r)})
			}
		}
		sort.Slice(s.cands, func(i, j int) bool { return s.cands[i].name < s.cands[j].name })
		for i, c := range s.cands {
			s.byLen[len(c.runes)] = append(s.byLen[len(c.runes)], i)
		}
	})
}

// Match is a name together with its value.
type Match struct{ Name, Value string }

// Fuzzy looks for the name closest to word within maxEdits edits (see package
// fuzzy). Typos rarely hit the first letter, so a name must start with the same
// letter as word (ignoring case) to be considered; this keeps ordinary words
// such as "brown" from being "corrected" to "frown". It succeeds only if the
// closest names all mean the same symbol; if they differ, ok is false and tied
// lists them so the caller can suggest them.
func (s *Set) Fuzzy(word string, maxEdits int) (m Match, tied []string, ok bool) {
	s.buildFuzzy()
	w := []rune(word)
	if len(w) == 0 {
		return Match{}, nil, false
	}
	limit := maxEdits * fuzzy.Edit
	var scratch fuzzy.Scratch
	sig := signature(w)
	initial := unicode.ToLower(w[0])
	bestDist := limit + 1
	var best []int
	for l := len(w) - maxEdits; l <= len(w)+maxEdits; l++ {
		for _, ci := range s.byLen[l] {
			cand := &s.cands[ci]
			if unicode.ToLower(cand.runes[0]) != initial || bits.OnesCount32(sig^cand.sig) > 2*maxEdits {
				continue
			}
			d := scratch.Distance(w, cand.runes, bestDist)
			switch {
			case d < bestDist:
				bestDist, best = d, []int{ci}
			case d == bestDist && d <= limit:
				best = append(best, ci)
			}
		}
	}
	if len(best) == 0 {
		return Match{}, nil, false
	}
	sort.Ints(best) // cands is sorted by name, so this makes the result deterministic
	first := s.cands[best[0]].name
	value := s.values[first]
	same := true
	for _, ci := range best {
		tied = append(tied, s.cands[ci].name)
		if s.values[s.cands[ci].name] != value {
			same = false
		}
	}
	if !same {
		return Match{}, tied, false
	}
	return Match{Name: first, Value: value}, nil, true
}

// Suggest returns up to n names resembling word, closest first, for use in
// "did you mean" messages. It is deliberately more generous than Fuzzy.
func (s *Set) Suggest(word string, n int) []string {
	s.buildFuzzy()
	w := []rune(word)
	if len(w) == 0 || n <= 0 {
		return nil
	}
	limit := 2 * fuzzy.Edit
	if len(w) <= 3 {
		limit = fuzzy.Edit
	}
	type scored struct {
		name string
		d    int
	}
	var found []scored
	var scratch fuzzy.Scratch
	for l := len(w) - 2; l <= len(w)+2; l++ {
		for _, ci := range s.byLen[l] {
			if d := scratch.Distance(w, s.cands[ci].runes, limit); d <= limit {
				found = append(found, scored{s.cands[ci].name, d})
			}
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].d != found[j].d {
			return found[i].d < found[j].d
		}
		return found[i].name < found[j].name
	})
	var out []string
	seen := map[string]bool{}
	for _, f := range found {
		// one suggestion per symbol: aliases of an already suggested symbol add nothing
		if v := s.values[f.name]; !seen[v] {
			seen[v] = true
			out = append(out, f.name)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

// Entries returns every resolvable name with its value, sorted by name.
func (s *Set) Entries() []Entry {
	s.buildFind()
	return slices.Clone(s.ents)
}

func (s *Set) buildReverse() {
	s.revOnce.Do(func() {
		s.rev = make(map[string][]string)
		for n, v := range s.values {
			s.rev[v] = append(s.rev[v], n)
			if r := len([]rune(v)); r > s.maxVal {
				s.maxVal = r
			}
		}
		for _, names := range s.rev {
			sort.Slice(names, func(i, j int) bool { return s.nameLess(names[i], names[j]) })
		}
	})
}

// nameLess orders the names of one symbol from most to least preferable: names
// usable without a backslash first, then canonical names, then shorter ones.
func (s *Set) nameLess(a, b string) bool {
	if sa, sb := len(a) < 2, len(b) < 2; sa != sb {
		return !sa
	}
	if ca, cb := s.canonical[a], s.canonical[b]; ca != cb {
		return ca
	}
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// Reverse returns the names whose value is exactly value, best name first.
// The slice is a copy: a Set is immutable and callers may do what they like with it.
func (s *Set) Reverse(value string) []string {
	s.buildReverse()
	return slices.Clone(s.rev[value])
}

// MaxValueRunes is the length in runes of the longest value, which bounds how
// far ahead a reverse lookup has to look.
func (s *Set) MaxValueRunes() int {
	s.buildReverse()
	return s.maxVal
}
