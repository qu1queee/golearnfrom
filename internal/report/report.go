package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/qu1queee/golearnfrom/internal/catalog"
	"github.com/qu1queee/golearnfrom/internal/profile"
)

const (
	divider = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) println(args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, args...)
}

// PrintProfile writes a human-readable profile to w.
func PrintProfile(w io.Writer, p *profile.Profile, snippetLimit int) error {
	ew := &errWriter{w: w}
	ew.printf("\n%s\n", divider)
	ew.printf("  Learning from: %s   (%d PRs · %d Go files)\n", p.Username, p.PRCount, p.FileCount)
	ew.printf("%s\n\n", divider)

	kinds := []catalog.Kind{
		catalog.KindAlgorithm,
		catalog.KindDataStructure,
		catalog.KindConcurrency,
		catalog.KindIdiom,
	}

	for _, kind := range kinds {
		patterns := catalog.ByKind(kind)
		var found []*catalog.Pattern
		for _, pat := range patterns {
			if p.PatternCount[pat.ID] > 0 {
				found = append(found, pat)
			}
		}
		if len(found) == 0 {
			continue
		}

		ew.printf("%s\n\n", kindHeader(kind))

		for _, pat := range found {
			count := p.PatternCount[pat.ID]
			ew.printf("  ◆ %s   (%d occurrence(s))\n", pat.Name, count)
			ew.printf("    %s\n", pat.Description)
			ew.printf("    → %s\n\n", pat.WhyItMatters)

			hits := p.Hits[pat.ID]
			shown := 0
			for _, hit := range hits {
				if shown >= snippetLimit {
					break
				}
				if hit.Match.Snippet == "" {
					continue
				}
				ew.printf("    From: %s  (line %d)\n", hit.PR, hit.Match.Line)
				for line := range strings.SplitSeq(hit.Match.Snippet, "\n") {
					ew.printf("      %s\n", line)
				}
				ew.println()
				shown++
			}
		}
	}
	return ew.err
}

// PrintJSON writes a JSON representation of profiles to w.
func PrintJSON(w io.Writer, profiles ...*profile.Profile) error {
	type jsonHit struct {
		PatternID   string `json:"pattern_id"`
		PatternName string `json:"pattern_name"`
		Kind        string `json:"kind"`
		PR          string `json:"pr"`
		File        string `json:"file"`
		Line        int    `json:"line"`
		Snippet     string `json:"snippet"`
	}
	type jsonProfile struct {
		Username     string               `json:"username"`
		PRCount      int                  `json:"pr_count"`
		FileCount    int                  `json:"file_count"`
		PatternCount map[string]int       `json:"pattern_counts"`
		Hits         map[string][]jsonHit `json:"hits"`
	}

	out := make([]jsonProfile, 0, len(profiles))
	for _, p := range profiles {
		jp := jsonProfile{
			Username:     p.Username,
			PRCount:      p.PRCount,
			FileCount:    p.FileCount,
			PatternCount: p.PatternCount,
			Hits:         make(map[string][]jsonHit),
		}
		for id, hits := range p.Hits {
			for _, h := range hits {
				jp.Hits[id] = append(jp.Hits[id], jsonHit{
					PatternID:   h.Pattern.ID,
					PatternName: h.Pattern.Name,
					Kind:        string(h.Pattern.Kind),
					PR:          h.PR,
					File:        h.Match.File,
					Line:        h.Match.Line,
					Snippet:     h.Match.Snippet,
				})
			}
		}
		out = append(out, jp)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// PrintPrompt writes a markdown prompt designed to be piped to an LLM CLI (e.g. claude).
// It includes a tutor instruction, every detected pattern with description and snippets,
// and closes with a learning ask.
func PrintPrompt(w io.Writer, p *profile.Profile, snippetLimit int) error {
	ew := &errWriter{w: w}
	ew.printf("# Go patterns from %s — %s (%d PRs, %d files)\n\n",
		p.Username, p.CreatedAt.Format("2006-01-02"), p.PRCount, p.FileCount)

	ew.println("You are a Go tutor. Below are coding patterns detected in this developer's")
	ew.println("merged pull requests. For each pattern:")
	ew.println("1. Explain clearly what it is and how it works in Go.")
	ew.println("2. Explain why this codebase uses it consistently — what problem it solves.")
	ew.println("3. State the key insight: the one thing to remember about this pattern.")
	ew.println(`4. Give a minimal, self-contained "try it yourself" Go snippet I can run.`)
	ew.println("5. Mention any common mistakes or anti-patterns to avoid.")
	ew.println()

	kinds := []catalog.Kind{
		catalog.KindAlgorithm,
		catalog.KindDataStructure,
		catalog.KindConcurrency,
		catalog.KindIdiom,
	}

	for _, kind := range kinds {
		var found []*catalog.Pattern
		for _, pat := range catalog.ByKind(kind) {
			if p.PatternCount[pat.ID] > 0 {
				found = append(found, pat)
			}
		}
		if len(found) == 0 {
			continue
		}

		ew.printf("---\n\n## %s\n\n", strings.ToUpper(string(kind)))

		for _, pat := range found {
			count := p.PatternCount[pat.ID]
			ew.printf("### %s (%d occurrence(s))\n\n", pat.Name, count)
			ew.printf("_%s_\n\n", pat.Description)
			ew.printf("**Why it matters:** %s\n\n", pat.WhyItMatters)

			hits := p.Hits[pat.ID]
			shown := 0
			for _, hit := range hits {
				if shown >= snippetLimit {
					break
				}
				if hit.Match.Snippet == "" {
					continue
				}
				ew.printf("**Real example** (from %s, line %d):\n", hit.PR, hit.Match.Line)
				ew.printf("```go\n%s\n```\n\n", hit.Match.Snippet)
				shown++
			}
		}
	}

	ew.println("---")
	ew.println()
	ew.println("Work through each pattern above. After explaining all of them, give me a")
	ew.println("prioritised list of which ones I should focus on learning first if I want")
	ew.println("to write idiomatic, production-quality Go.")
	return ew.err
}

func kindHeader(k catalog.Kind) string {
	switch k {
	case catalog.KindAlgorithm:
		return "ALGORITHMS"
	case catalog.KindDataStructure:
		return "DATA STRUCTURES"
	case catalog.KindConcurrency:
		return "CONCURRENCY"
	case catalog.KindIdiom:
		return "GO IDIOMS"
	default:
		return strings.ToUpper(string(k))
	}
}
