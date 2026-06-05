package analyzer

import (
	"go/parser"
	"go/token"

	"github.com/qu1queee/golearnfrom/internal/catalog"
	"github.com/qu1queee/golearnfrom/internal/profile"
)

// FileResult holds pattern matches found in a single file.
type FileResult struct {
	Filename string
	Matches  []PatternMatch
}

type PatternMatch struct {
	Pattern *catalog.Pattern
	Match   catalog.Match
}

// AnalyzeFile parses src as Go source and runs all catalog patterns against it.
func AnalyzeFile(filename string, src []byte) ([]PatternMatch, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		// Partial parse — still try the patterns.
		if f == nil {
			return nil, err
		}
	}

	var results []PatternMatch
	for _, pat := range catalog.All() {
		matches := pat.Detect(f, fset, src)
		for _, m := range matches {
			if m.File == "" {
				m.File = filename
			}
			results = append(results, PatternMatch{Pattern: pat, Match: m})
		}
	}
	return results, nil
}

// MergeIntoProfile adds analysis results for a PR into the author's profile.
func MergeIntoProfile(p *profile.Profile, prURL string, results []PatternMatch) {
	for _, r := range results {
		p.Add(r.Pattern, prURL, r.Match)
	}
}
