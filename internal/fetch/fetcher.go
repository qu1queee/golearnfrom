package fetch

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/qu1queee/golearnfrom/internal/analyzer"
	ghclient "github.com/qu1queee/golearnfrom/internal/github"
	"github.com/qu1queee/golearnfrom/internal/profile"
)

// Options controls fetch behaviour.
type Options struct {
	Limit   int    // max PRs to fetch
	Repo    string // owner/repo to scope user PR search (required for user mode)
	Verbose bool
}

func defaultOpts(o *Options) *Options {
	if o == nil {
		o = &Options{}
	}
	if o.Limit <= 0 {
		o.Limit = 50
	}
	return o
}

// ForUser fetches PRs authored by username and returns their profile.
// The limit counts only PRs that contain at least one Go file; PRs with no
// Go files are skipped and do not consume the budget.
func ForUser(ctx context.Context, client *ghclient.Client, username string, opts *Options) (*profile.Profile, error) {
	opts = defaultOpts(opts)
	p := profile.New(username)

	seenSHAs := make(map[string]bool)
	page := 1
	for p.PRCount < opts.Limit {
		prs, hasMore, err := client.SearchUserPRsPage(ctx, username, opts.Repo, page)
		if err != nil {
			return p, err
		}
		if opts.Verbose {
			log.Printf("[page %d] fetched %d PRs — %d/%d go-file PRs collected", page, len(prs), p.PRCount, opts.Limit)
		}
		for _, pr := range prs {
			if p.PRCount >= opts.Limit {
				break
			}
			had, err := processPR(ctx, client, pr, p, seenSHAs, opts.Verbose)
			if err != nil {
				log.Printf("[warn] PR #%d %q: %v", pr.Number, pr.Title, err)
				continue
			}
			if had {
				p.PRCount++
			}
		}
		if !hasMore {
			break
		}
		page++
	}

	if p.PRCount == 0 {
		log.Printf("[done] no Go files found in any PR for %s in %s", username, opts.Repo)
	}
	return p, nil
}


// processPR downloads and analyses Go files in a PR.
// seenSHAs deduplicates blobs across PRs — the same file content is never analysed twice.
// Returns true if at least one new Go file was found and analysed.
func processPR(ctx context.Context, client *ghclient.Client, pr *ghclient.PullRequest, p *profile.Profile, seenSHAs map[string]bool, verbose bool) (bool, error) {
	files, err := client.ListPRFiles(ctx, pr.Owner, pr.Repo, pr.Number)
	if err != nil {
		return false, fmt.Errorf("list files: %w", err)
	}

	goFiles := 0
	skipped := 0
	for _, f := range files {
		if strings.HasSuffix(f.Filename, ".go") {
			goFiles++
		}
	}

	if verbose {
		log.Printf("  PR #%d %q — %d go file(s) in diff", pr.Number, pr.Title, goFiles)
	}

	found := false
	newPatterns := 0
	for _, f := range files {
		if seenSHAs[f.SHA] {
			skipped++
			if verbose {
				log.Printf("    skip %s (sha %s already seen)", f.Filename, f.SHA[:8])
			}
			continue
		}
		src, err := client.GetFileContent(ctx, pr.Owner, pr.Repo, f.SHA)
		if err != nil {
			log.Printf("    [warn] get %s: %v", f.Filename, err)
			continue
		}
		seenSHAs[f.SHA] = true
		results, err := analyzer.AnalyzeFile(f.Filename, src)
		if err != nil {
			log.Printf("    [warn] parse %s: %v", f.Filename, err)
			continue
		}
		p.FileCount++
		found = true
		newPatterns += len(results)
		if verbose {
			if len(results) > 0 {
				names := patternNames(results)
				log.Printf("    ✓ %s — %d pattern(s): %s", f.Filename, len(results), names)
			} else {
				log.Printf("    · %s — no patterns", f.Filename)
			}
		}
		analyzer.MergeIntoProfile(p, pr.URL, results)
	}

	if verbose && skipped > 0 {
		log.Printf("    (%d file(s) skipped — duplicate blobs)", skipped)
	}

	return found, nil
}

func patternNames(results []analyzer.PatternMatch) string {
	seen := make(map[string]bool)
	var names []string
	for _, r := range results {
		if !seen[r.Pattern.Name] {
			seen[r.Pattern.Name] = true
			names = append(names, r.Pattern.Name)
		}
	}
	return strings.Join(names, ", ")
}
