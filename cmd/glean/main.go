package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qu1queee/golearnfrom/internal/fetch"
	ghclient "github.com/qu1queee/golearnfrom/internal/github"
	"github.com/qu1queee/golearnfrom/internal/report"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "glean",
		Short: "Learn Go patterns from real PR contributions",
		Long: `glean analyses GitHub pull requests and surfaces coding patterns
so you can learn techniques used by experienced Go developers.

Set GITHUB_TOKEN env var for authenticated access (recommended).`,
	}
	root.AddCommand(userCmd())
	return root
}

func userCmd() *cobra.Command {
	var (
		repo         string
		limit        int
		outputJSON   bool
		outputPrompt bool
		snippetLimit int
		verbose      bool
	)

	cmd := &cobra.Command{
		Use:   "user <github-username>",
		Short: "Analyse a user's merged PR contributions in a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			if repo == "" {
				return fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
			}
			if !strings.Contains(repo, "/") {
				return fmt.Errorf("--repo must be in owner/repo format, got %q", repo)
			}

			client := ghclient.New()
			ctx := context.Background()

			fmt.Fprintf(os.Stderr, "Fetching PRs for %s in %s (limit %d)...\n", username, repo, limit)

			p, err := fetch.ForUser(ctx, client, username, &fetch.Options{
				Repo:    repo,
				Limit:   limit,
				Verbose: verbose,
			})
			if err != nil {
				return err
			}

			switch {
			case outputJSON:
				return report.PrintJSON(os.Stdout, p)
			case outputPrompt:
				report.PrintPrompt(os.Stdout, p, snippetLimit)
			default:
				report.PrintProfile(os.Stdout, p, snippetLimit)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "repository to search (owner/repo) — required")
	cmd.Flags().IntVar(&limit, "limit", 30, "max PRs to fetch")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&outputPrompt, "prompt", false, "output as a markdown prompt ready to pipe to an LLM")
	cmd.Flags().IntVar(&snippetLimit, "snippets", 2, "max code snippets shown per pattern")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "print progress to stderr")
	return cmd
}
