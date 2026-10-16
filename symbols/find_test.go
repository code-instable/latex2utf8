package symbols

import (
	"slices"
	"strings"
	"testing"
)

func findNames(s *Set, limit int, queries ...string) []string {
	hits, _ := s.Find(queries, limit)
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.Name
	}
	return names
}

// The point of Find: one mistake does not hide what was meant.
func TestFindToleratesMistakes(t *testing.T) {
	s := Default()
	tests := []struct {
		query string
		want  []string // the best hits, in this order
	}{
		{"Gamme", []string{"Gamma", "Game", "gamma"}}, // a wrong letter, the case kept
		{"gamme", []string{"gamma", "Gamma", "Game"}}, // ... and the other case
		{"gammma", []string{"gamma", "Gamma"}},        // a letter too many
		{"gama", []string{"gamma"}},                   // a letter missing
		{"alhpa", []string{"alpha", "Alpha"}},         // two letters swapped
		{"nabal", []string{"nabla"}},                  // ... again
		{"epsilom", []string{"epsilon", "Epsilon"}},   // a wrong letter late in a longer word
		{"lamda", []string{"lambda", "Lambda"}},       // a missing letter in the middle
		{"rightarow", []string{"rightarrow", "Rightarrow"}},
		{"GAMMA", []string{"Gamma", "gamma"}}, // case only: Gamma shares the G
		{"gamma", []string{"gamma", "Gamma"}}, // nothing wrong: the name itself first, then the other case
	}
	for _, tt := range tests {
		got := findNames(s, len(tt.want), tt.query)
		if !slices.Equal(got, tt.want) {
			t.Errorf("Find(%q) starts with %v, want %v", tt.query, got, tt.want)
		}
	}
}

// What was typed exactly comes before anything that merely resembles it.
func TestFindRanksTheQueryItselfFirst(t *testing.T) {
	s := Default()
	for _, name := range []string{"gamma", "Gamma", "alpha", "pi", "Pi", "rightarrow", "mbfGamma", "BbbR", "ldots"} {
		hits, _ := s.Find([]string{name}, 5)
		if len(hits) == 0 || hits[0].Name != name {
			t.Errorf("Find(%q) = %v, want %q first", name, hits, name)
		}
	}
}

// Names that start with the query come right after the query itself, before the names that are only near it.
func TestFindPrefersNamesStartingWithTheQuery(t *testing.T) {
	s := Default()
	got := findNames(s, 0, "pi")
	if len(got) < 3 || got[0] != "pi" || got[1] != "Pi" {
		t.Fatalf("Find(pi) = %v", got)
	}
	// pisces starts with "pi"; upin only contains it
	if i, j := slices.Index(got, "pisces"), slices.Index(got, "upin"); i < 0 || j < 0 || i > j {
		t.Errorf("pisces (%d) should come before upin (%d)", i, j)
	}
}

// A short query has no room for a mistake: one wrong letter changes the whole name.
func TestFindShortQueriesAreExact(t *testing.T) {
	s := Default()
	got := findNames(s, 0, "pi")
	for _, n := range got {
		if !strings.Contains(strings.ToLower(n), "pi") {
			t.Errorf("Find(pi) returned %q, which does not contain pi", n)
		}
	}
	if got := findNames(s, 0, "zq"); len(got) != 0 {
		t.Errorf("Find(zq) = %v, want nothing", got)
	}
	// ... while a longer one has some
	if got := findNames(s, 1, "alhpa"); len(got) != 1 {
		t.Errorf("Find(alhpa) = %v", got)
	}
}

func TestFindToleranceGrowsWithTheQuery(t *testing.T) {
	prev := -1
	for n := 0; n < 20; n++ {
		got := FindEdits(n)
		if got < prev {
			t.Errorf("FindEdits(%d) = %d, less than FindEdits(%d) = %d", n, got, n-1, prev)
		}
		prev = got
	}
	if FindEdits(1) != 0 || FindEdits(2) != 0 || FindEdits(5) != 2 || FindEdits(12) != 3 {
		t.Errorf("FindEdits: 1→%d 2→%d 5→%d 12→%d", FindEdits(1), FindEdits(2), FindEdits(5), FindEdits(12))
	}
}

// A backslash, or blanks around the query, change nothing.
func TestFindIgnoresBackslashAndBlanks(t *testing.T) {
	s := Default()
	want := findNames(s, 10, "gamme")
	for _, q := range []string{`\gamme`, " gamme ", ` \gamme`, "\tgamme\n"} {
		if got := findNames(s, 10, q); !slices.Equal(got, want) {
			t.Errorf("Find(%q) = %v, want %v", q, got, want)
		}
	}
	if hits, total := s.Find([]string{"", `\`, "  "}, 0); hits != nil || total != 0 {
		t.Errorf("empty queries found %v (%d)", hits, total)
	}
	if hits, total := s.Find(nil, 0); hits != nil || total != 0 {
		t.Errorf("no queries found %v (%d)", hits, total)
	}
}

// Typing the symbol finds its names, every one of them.
func TestFindBySymbol(t *testing.T) {
	s := Default()
	hits, _ := s.Find([]string{"∇"}, 0)
	var names []string
	for _, h := range hits {
		if h.Score != -1 {
			t.Errorf("%q scores %d for its own symbol, want -1", h.Name, h.Score)
		}
		names = append(names, h.Name)
	}
	if !slices.Equal(names, []string{"del", "nabla"}) {
		t.Errorf("Find(∇) = %v", names)
	}
}

func TestFindLimitAndTotal(t *testing.T) {
	s := Default()
	all, total := s.Find([]string{"arrow"}, 0)
	if total != len(all) || total < 50 {
		t.Fatalf("limit 0 should return everything: %d hits, total %d", len(all), total)
	}
	few, total2 := s.Find([]string{"arrow"}, 3)
	if len(few) != 3 || total2 != total {
		t.Errorf("limit 3: %d hits, total %d (want 3, %d)", len(few), total2, total)
	}
	for i := range few {
		if few[i] != all[i] {
			t.Errorf("limit changed the order at %d: %v vs %v", i, few[i], all[i])
		}
	}
}

// With several queries a name counts for the closest of them.
func TestFindSeveralQueries(t *testing.T) {
	s := Default()
	got := findNames(s, 0, "Gamme", "alhpa")
	for _, want := range []string{"Gamma", "alpha"} {
		if !slices.Contains(got, want) {
			t.Errorf("Find(Gamme, alhpa) lacks %q", want)
		}
	}
	// no name is listed twice
	seen := map[string]bool{}
	for _, n := range got {
		if seen[n] {
			t.Errorf("%q listed twice", n)
		}
		seen[n] = true
	}
	// a name found by both queries is no worse than by the better of them
	single := findNames(s, 3, "gamme")
	both := findNames(s, 3, "gamme", "zzzzzzzz")
	if !slices.Equal(single, both) {
		t.Errorf("a second query that finds nothing changed the order: %v vs %v", single, both)
	}
}

func TestFindNothing(t *testing.T) {
	s := Default()
	if hits, total := s.Find([]string{"xyzzyqwv"}, 0); len(hits) != 0 || total != 0 {
		t.Errorf("found %v (%d) for gibberish", hits, total)
	}
}

// The order is a function of the query: no map order, no ties left to chance.
func TestFindIsDeterministic(t *testing.T) {
	for _, q := range []string{"Gamme", "ar", "left", "bb", "fraktur"} {
		want := findNames(Default(), 0, q)
		for i := 0; i < 5; i++ {
			if got := findNames(Default(), 0, q); !slices.Equal(got, want) {
				t.Fatalf("Find(%q) is not stable", q)
			}
		}
	}
}

// A name finds itself, and nothing that merely resembles it comes first. Each Find looks at every name, so
// only one name in twenty is tried.
func TestFindEveryNameFindsItself(t *testing.T) {
	s := Default()
	for i, e := range s.Entries() {
		if i%20 != 0 || !allLetters(e.Name) {
			continue // script keys carry the ^ or _ along
		}
		hits, _ := s.Find([]string{e.Name}, 0)
		found := false
		for _, h := range hits {
			if h.Name == e.Name {
				found = true
				if h.Value != e.Value {
					t.Errorf("%q found with value %q, want %q", e.Name, h.Value, e.Value)
				}
			}
		}
		if !found {
			t.Fatalf("Find(%q) does not find it", e.Name)
		}
		// the top hit is the name itself, or another one that is the same up to case (Gamma for gamma)
		if len(hits) > 0 && !strings.EqualFold(hits[0].Name, e.Name) && hits[0].Value != e.Value {
			t.Errorf("Find(%q) starts with %q", e.Name, hits[0].Name)
		}
	}
}

// Find does not change what was loaded, and is safe to call from several goroutines.
func TestFindConcurrently(t *testing.T) {
	s := Default()
	want := findNames(s, 5, "Gamme")
	done := make(chan []string)
	for i := 0; i < 8; i++ {
		go func() { done <- findNames(s, 5, "Gamme") }()
	}
	for i := 0; i < 8; i++ {
		if got := <-done; !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func BenchmarkFind(b *testing.B) {
	s := Default()
	s.Find([]string{"warm"}, 1) // build the tables first
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Find([]string{"Gamme"}, 20)
	}
}
