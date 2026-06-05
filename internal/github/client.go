package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	gogithub "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API.
type Client struct {
	gh *gogithub.Client
}

// New creates an authenticated client using GITHUB_TOKEN env var.
// Falls back to unauthenticated (60 req/h) if the token is absent.
func New() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}
	return &Client{gh: gogithub.NewClient(httpClient)}
}

// PullRequest is a minimal PR descriptor.
type PullRequest struct {
	Number int
	Title  string
	URL    string
	Author string
	Owner  string
	Repo   string
}

// PRFile is a file changed in a PR.
type PRFile struct {
	Filename string
	SHA      string
	Status   string // added, modified, removed, renamed
}

// SearchUserPRsPage returns one page of merged PRs authored by username.
// If repo is non-empty (owner/repo) the search is scoped to that repository.
// Returns the PRs, whether there are more pages, and any error.
func (c *Client) SearchUserPRsPage(ctx context.Context, username, repo string, page int) ([]*PullRequest, bool, error) {
	query := fmt.Sprintf("author:%s type:pr is:merged", username)
	if repo != "" {
		query += fmt.Sprintf(" repo:%s", repo)
	}
	const perPage = 30
	result, _, err := c.gh.Search.Issues(ctx, query, &gogithub.SearchOptions{
		ListOptions: gogithub.ListOptions{Page: page, PerPage: perPage},
	})
	if err != nil {
		return nil, false, fmt.Errorf("search PRs for %s: %w", username, err)
	}
	var prs []*PullRequest
	for _, issue := range result.Issues {
		owner, r, ok := repoFromURL(issue.GetRepositoryURL())
		if !ok {
			continue
		}
		prs = append(prs, &PullRequest{
			Number: issue.GetNumber(),
			Title:  issue.GetTitle(),
			URL:    issue.GetHTMLURL(),
			Author: username,
			Owner:  owner,
			Repo:   r,
		})
	}
	fetched := page*perPage
	hasMore := len(result.Issues) == perPage && fetched < result.GetTotal()
	return prs, hasMore, nil
}


// ListPRFiles returns the Go files changed in a PR (excludes deleted files).
func (c *Client) ListPRFiles(ctx context.Context, owner, repo string, number int) ([]*PRFile, error) {
	files, _, err := c.gh.PullRequests.ListFiles(ctx, owner, repo, number, &gogithub.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("list files for PR %d: %w", number, err)
	}
	var out []*PRFile
	for _, f := range files {
		name := f.GetFilename()
		if !strings.HasSuffix(name, ".go") || f.GetStatus() == "removed" {
			continue
		}
		out = append(out, &PRFile{
			Filename: name,
			SHA:      f.GetSHA(),
			Status:   f.GetStatus(),
		})
	}
	return out, nil
}

// GetFileContent fetches the raw bytes of a file by its blob SHA.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, sha string) ([]byte, error) {
	blob, _, err := c.gh.Git.GetBlob(ctx, owner, repo, sha)
	if err != nil {
		return nil, fmt.Errorf("get blob %s: %w", sha, err)
	}
	switch blob.GetEncoding() {
	case "base64":
		cleaned := strings.ReplaceAll(blob.GetContent(), "\n", "")
		data, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return nil, fmt.Errorf("decode blob %s: %w", sha, err)
		}
		return data, nil
	case "utf-8", "":
		return []byte(blob.GetContent()), nil
	default:
		return nil, fmt.Errorf("unsupported blob encoding: %s", blob.GetEncoding())
	}
}

// repoFromURL parses owner and repo from a GitHub repository API URL.
// e.g. https://api.github.com/repos/golang/go → "golang", "go"
func repoFromURL(u string) (owner, repo string, ok bool) {
	const prefix = "https://api.github.com/repos/"
	u = strings.TrimPrefix(u, prefix)
	parts := strings.SplitN(u, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
