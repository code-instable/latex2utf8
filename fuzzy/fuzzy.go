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
	if len(a) == 0 || len(b) == 0 {
		return capped((len(a)+len(b))*Edit, limit)
	}
	// Each length difference needs at least one insertion or deletion.
	if d := abs(len(a)-len(b)) * Edit; d > limit {
		return limit + 1
	}
	width := len(b) + 1
	if cap(s.prev) < width {
		s.prev2 = make([]int, width)
		s.prev = make([]int, width)
		s.cur = make([]int, width)
	}
	prev2, prev, cur := s.prev2[:width], s.prev[:width], s.cur[:width]
	for j := range prev {
		prev[j] = j * Edit
	}
	prevMin := 0 // smallest value of the previous row
	for i := 1; i <= len(a); i++ {
		cur[0] = i * Edit
		rowMin := cur[0]
		for j := 1; j <= len(b); j++ {
			sub := Edit
			switch {
			case a[i-1] == b[j-1]:
				sub = 0
			case sameLetter(a[i-1], b[j-1]):
				sub = CaseEdit
			}
			v := min(prev[j]+Edit, cur[j-1]+Edit, prev[j-1]+sub)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
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
	return capped(prev[len(b)], limit)
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
