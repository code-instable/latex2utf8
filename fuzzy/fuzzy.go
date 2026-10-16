// Package fuzzy implements the bounded edit distance used to correct typos in
// symbol names.
//
// The metric is the optimal string alignment distance (insertion, deletion,
// substitution and adjacent transposition) computed over runes. Costs are
// expressed in half edits so that a difference of letter case alone, which is
// a much weaker signal than a different letter, can be priced at half an edit:
// "bbbR" is then closer to "BbbR" than to "Bbbr".
package fuzzy

import "unicode"

const (
	// Edit is the cost of one insertion, deletion, substitution or adjacent
	// transposition.
	Edit = 2
	// CaseEdit is the cost of replacing a rune by the same letter in the other case.
	CaseEdit = 1
)

// Scratch holds the working rows of the distance computation so that many
// distances can be computed without allocating. The zero value is ready to use.
// A Scratch must not be used by several goroutines at once.
type Scratch struct {
	prev2, prev, cur []int
}

// Distance returns the weighted distance between a and b, or limit+1 as soon as
// it is certain that the distance exceeds limit. Both limit and the result are
// in the units of Edit and CaseEdit.
func (s *Scratch) Distance(a, b []rune, limit int) int {
	return s.run(a, b, limit, false, CaseEdit)
}

// Within returns the distance between query and the closest part of text: the
// smallest Distance from query to any run of consecutive runes of text. It is
// what makes "gamme" close to "gammadot" as well as to "gamma". With foldCase
// a difference of letter case costs nothing. Like Distance it returns limit+1
// once the result is certain to exceed limit.
func (s *Scratch) Within(query, text []rune, limit int, foldCase bool) int {
	caseCost := CaseEdit
	if foldCase {
		caseCost = 0
	}
	return s.run(query, text, limit, true, caseCost)
}

// run computes the optimal string alignment distance. With free, the match may
// start and end anywhere in b (approximate substring matching).
func (s *Scratch) run(a, b []rune, limit int, free bool, caseCost int) int {
	if len(a) == 0 {
		if free {
			return 0
		}
		return capped(len(b)*Edit, limit)
	}
	if len(b) == 0 {
		return capped(len(a)*Edit, limit)
	}
	// Each length difference needs at least one insertion or deletion.
	if !free {
		if d := abs(len(a)-len(b)) * Edit; d > limit {
			return limit + 1
		}
	}
	width := len(b) + 1
	if cap(s.prev) < width {
		s.prev2 = make([]int, width)
		s.prev = make([]int, width)
		s.cur = make([]int, width)
	}
	prev2, prev, cur := s.prev2[:width], s.prev[:width], s.cur[:width]
	for j := range prev {
		if free {
			prev[j] = 0
		} else {
			prev[j] = j * Edit
		}
	}
	cost := func(x, y rune) int {
		switch {
		case x == y:
			return 0
		case sameLetter(x, y):
			return caseCost
		}
		return Edit
	}
	prevMin := 0 // smallest value of the previous row
	for i := 1; i <= len(a); i++ {
		cur[0] = i * Edit
		rowMin := cur[0]
		for j := 1; j <= len(b); j++ {
			v := min(prev[j]+Edit, cur[j-1]+Edit, prev[j-1]+cost(a[i-1], b[j-1]))
			if i > 1 && j > 1 && cost(a[i-1], b[j-2]) == 0 && cost(a[i-2], b[j-1]) == 0 && a[i-1] != a[i-2] {
				v = min(v, prev2[j-2]+Edit)
			}
			cur[j] = v
			rowMin = min(rowMin, v)
		}
		// Later rows are bounded below by the two latest rows (transposition
		// reaches back two rows), so the distance can no longer drop to limit.
		if i > 1 && rowMin > limit && prevMin > limit {
			return limit + 1
		}
		prevMin = rowMin
		prev2, prev, cur = prev, cur, prev2
	}
	if !free {
		return capped(prev[len(b)], limit)
	}
	best := prev[0]
	for _, v := range prev[1:] {
		best = min(best, v)
	}
	return capped(best, limit)
}

// Distance is a convenience wrapper that allocates its own Scratch.
func Distance(a, b string, limit int) int {
	var s Scratch
	return s.Distance([]rune(a), []rune(b), limit)
}

func sameLetter(a, b rune) bool {
	return a != b && unicode.ToLower(a) == unicode.ToLower(b)
}

func capped(d, limit int) int {
	if d > limit {
		return limit + 1
	}
	return d
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
