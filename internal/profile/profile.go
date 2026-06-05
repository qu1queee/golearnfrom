package profile

import (
	"time"

	"github.com/qu1queee/golearnfrom/internal/catalog"
)

// PatternHit records one occurrence of a pattern in a PR.
type PatternHit struct {
	Pattern *catalog.Pattern
	PR      string // URL or title
	Match   catalog.Match
}

// Profile aggregates pattern findings for a single author.
type Profile struct {
	Username  string
	CreatedAt time.Time

	PRCount   int
	FileCount int

	// Hits keyed by pattern ID.
	Hits map[string][]PatternHit

	// PatternCount is a convenience summary: patternID → count.
	PatternCount map[string]int
}

func New(username string) *Profile {
	return &Profile{
		Username:     username,
		CreatedAt:    time.Now(),
		Hits:         make(map[string][]PatternHit),
		PatternCount: make(map[string]int),
	}
}

func (p *Profile) Add(pat *catalog.Pattern, pr string, m catalog.Match) {
	hit := PatternHit{Pattern: pat, PR: pr, Match: m}
	p.Hits[pat.ID] = append(p.Hits[pat.ID], hit)
	p.PatternCount[pat.ID]++
}

// TopPatterns returns pattern IDs sorted by hit count descending, limited to n.
func (p *Profile) TopPatterns(n int) []string {
	type kv struct {
		id    string
		count int
	}
	var pairs []kv
	for id, c := range p.PatternCount {
		pairs = append(pairs, kv{id, c})
	}
	// simple insertion sort — profile sizes are small
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].count > pairs[j-1].count; j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}
	var out []string
	for i, kv := range pairs {
		if i >= n {
			break
		}
		out = append(out, kv.id)
	}
	return out
}
