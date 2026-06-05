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

// PrintProfile writes a human-readable profile to w.
func PrintProfile(w io.Writer, p *profile.Profile, snippetLimit int) {
	fmt.Fprintf(w, "\n%s\n", divider)
	fmt.Fprintf(w, "  Learning from: %s   (%d PRs · %d Go files)\n", p.Username, p.PRCount, p.FileCount)
	fmt.Fprintf(w, "%s\n\n", divider)

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

		fmt.Fprintf(w, "%s\n\n", kindHeader(kind))

		for _, pat := range found {
			count := p.PatternCount[pat.ID]
			fmt.Fprintf(w, "  ◆ %s   (%d occurrence(s))\n", pat.Name, count)
			fmt.Fprintf(w, "    %s\n", pat.Description)
			fmt.Fprintf(w, "    → %s\n\n", pat.WhyItMatters)

			hits := p.Hits[pat.ID]
			shown := 0
			for _, hit := range hits {
				if shown >= snippetLimit {
					break
				}
				if hit.Match.Snippet == "" {
					continue
				}
				fmt.Fprintf(w, "    From: %s  (line %d)\n", hit.PR, hit.Match.Line)
				for _, line := range strings.Split(hit.Match.Snippet, "\n") {
					fmt.Fprintf(w, "      %s\n", line)
				}
				fmt.Fprintln(w)
				shown++
			}
		}
	}
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
		Username     string             `json:"username"`
		PRCount      int                `json:"pr_count"`
		FileCount    int                `json:"file_count"`
		PatternCount map[string]int     `json:"pattern_counts"`
		Hits         map[string][]jsonHit `json:"hits"`
	}

	var out []jsonProfile
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
func PrintPrompt(w io.Writer, p *profile.Profile, snippetLimit int) {
	fmt.Fprintf(w, "# Go patterns from %s — %s (%d PRs, %d files)\n\n",
		p.Username, p.CreatedAt.Format("2006-01-02"), p.PRCount, p.FileCount)

	fmt.Fprintln(w, "You are a Go tutor. Below are coding patterns detected in this developer's")
	fmt.Fprintln(w, "merged pull requests. For each pattern:")
	fmt.Fprintln(w, "1. Explain clearly what it is and how it works in Go.")
	fmt.Fprintln(w, "2. Explain why this codebase uses it consistently — what problem it solves.")
	fmt.Fprintln(w, "3. State the key insight: the one thing to remember about this pattern.")
	fmt.Fprintln(w, "4. Give a minimal, self-contained \"try it yourself\" Go snippet I can run.")
	fmt.Fprintln(w, "5. Mention any common mistakes or anti-patterns to avoid.")
	fmt.Fprintln(w)

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

		fmt.Fprintf(w, "---\n\n## %s\n\n", strings.ToUpper(string(kind)))

		for _, pat := range found {
			count := p.PatternCount[pat.ID]
			fmt.Fprintf(w, "### %s (%d occurrence(s))\n\n", pat.Name, count)
			fmt.Fprintf(w, "_%s_\n\n", pat.Description)
			fmt.Fprintf(w, "**Why it matters:** %s\n\n", pat.WhyItMatters)

			hits := p.Hits[pat.ID]
			shown := 0
			for _, hit := range hits {
				if shown >= snippetLimit {
					break
				}
				if hit.Match.Snippet == "" {
					continue
				}
				fmt.Fprintf(w, "**Real example** (from %s, line %d):\n", hit.PR, hit.Match.Line)
				fmt.Fprintf(w, "```go\n%s\n```\n\n", hit.Match.Snippet)
				shown++
			}
		}
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Work through each pattern above. After explaining all of them, give me a")
	fmt.Fprintln(w, "prioritised list of which ones I should focus on learning first if I want")
	fmt.Fprintln(w, "to write idiomatic, production-quality Go.")
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
