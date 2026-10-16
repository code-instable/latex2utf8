package symbols

import (
	"sort"
	"strings"
	"unicode"

	"github.com/code-instable/latex2utf8/fuzzy"
)

// Hit is a name found by Find.
type Hit struct {
	Entry
	// Score ranks the name among the others; lower is closer. A symbol that is
	// exactly what was asked for scores -1. The numbers mean nothing otherwise.
	Score int
}

func (s *Set) buildFind() {
	s.findOnce.Do(func() {
		s.ents = make([]Entry, 0, len(s.values))
		for n, v := range s.values {
			s.ents = append(s.ents, Entry{Name: n, Value: v, Alias: !s.canonical[n]})
		}
		sort.Slice(s.ents, func(i, j int) bool { return s.ents[i].Name < s.ents[j].Name })
		s.entRunes = make([][]rune, len(s.ents))
		for i, e := range s.ents {
			s.entRunes[i] = []rune(e.Name)
		}
	})
}

// FindEdits is how many mistakes Find tolerates in a query of n letters: none in a
// very short one, where a single wrong letter changes everything, and more as it
// gets longer.
func FindEdits(n int) int {
	switch {
	case n <= 2:
		return 0
	case n <= 4:
		return 1
	case n <= 7:
		return 2
	}
	return 3
}

// Find looks for the names closest to the queries and ranks them, best first. It is
// loose where converting is strict: it accepts a few mistakes in the query (see
// FindEdits), a different case, and a query that is only a part of the name, so
// that "Gamme" finds Gamma. A query that is a symbol finds the names of that symbol.
// A leading backslash is ignored. With several queries a name counts for the best of them.
//
// The names are ranked in three tiers: the query itself (up to case) comes first, then the
// names that start with it, then every other name that is near enough. Within a tier the
// score of a name is twice the distance from the query to the closest part of the name,
// plus the distance to the whole name: "gamme" is as close to gamma as to gammadot, but
// closer as a whole to gamma. Ties go to the name that shares the longest
// beginning with the query (typos tend to come late in a word, so Gamme means Gamma
// rather than Game), then to the shorter name, a name that is not an alias, and
// finally to the first in alphabetical order, so the order is always the same.
//
// It returns at most limit hits (all of them if limit is 0) and how many names matched.
func (s *Set) Find(queries []string, limit int) (hits []Hit, total int) {
	type query struct {
		raw   string
		runes []rune
		max   int // how far the closest part of a name may be, in fuzzy units
	}
	var qs []query
	for _, q := range queries {
		q = strings.TrimPrefix(strings.TrimSpace(q), `\`)
		if q != "" {
			r := []rune(q)
			qs = append(qs, query{q, r, FindEdits(len(r)) * fuzzy.Edit})
		}
	}
	if len(qs) == 0 {
		return nil, 0
	}
	s.buildFind()

	const (
		unlimited = 1 << 20
		tierStep  = 1 << 14 // more than any distance
	)
	type scored struct{ i, score, prefix int }
	var found []scored
	var scratch fuzzy.Scratch
	for i, e := range s.ents {
		name := s.entRunes[i]
		best := scored{i: i}
		ok := false
		for _, q := range qs {
			score, prefix := -1, 0
			if e.Value != q.raw {
				// case does not count for deciding whether a name is near enough, nor for how near
				// its closest part is ...
				near := scratch.Within(q.runes, name, q.max, true)
				if near > q.max {
					continue
				}
				// ... it counts once, in the distance to the whole name
				whole := scratch.Distance(q.runes, name, unlimited)
				prefix = commonPrefix(q.runes, name)
				tier := 2 // anything near enough
				switch {
				case prefix == len(q.runes) && len(name) == len(q.runes):
					tier = 0 // the query itself, up to case
				case prefix == len(q.runes):
					tier = 1 // a name that starts with the query
				}
				score = tier*tierStep + 2*near + whole
			}
			if !ok || score < best.score || score == best.score && prefix > best.prefix {
				best.score, best.prefix, ok = score, prefix, true
			}
		}
		if ok {
			found = append(found, best)
		}
	}
	sort.Slice(found, func(a, b int) bool {
		x, y := found[a], found[b]
		ex, ey := s.ents[x.i], s.ents[y.i]
		switch {
		case x.score != y.score:
			return x.score < y.score
		case x.prefix != y.prefix:
			return x.prefix > y.prefix
		case len(ex.Name) != len(ey.Name):
			return len(ex.Name) < len(ey.Name)
		case ex.Alias != ey.Alias:
			return !ex.Alias
		}
		return ex.Name < ey.Name
	})
	total = len(found)
	if limit > 0 && len(found) > limit {
		found = found[:limit]
	}
	hits = make([]Hit, len(found))
	for k, f := range found {
		hits[k] = Hit{Entry: s.ents[f.i], Score: f.score}
	}
	return hits, total
}

// commonPrefix is how many runes the two start with in common, ignoring case.
func commonPrefix(a, b []rune) int {
	n := 0
	for n < len(a) && n < len(b) && unicode.ToLower(a[n]) == unicode.ToLower(b[n]) {
		n++
	}
	return n
}
