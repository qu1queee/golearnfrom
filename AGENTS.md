# glean — agent context

## What this is

A CLI tool that fetches merged GitHub PRs, detects Go coding patterns via AST analysis,
and surfaces real code examples for learning. The primary output is a markdown prompt
designed to be piped to an LLM (`--prompt | claude`).

## Build & run

```bash
make build          # produces ./glean
./glean user <username> --repo <owner/repo> --limit 30
./glean user <username> --repo <owner/repo> --prompt | claude
```

Requires `GITHUB_TOKEN` env var (GitHub personal token, `public_repo` scope).

## Architecture

```
cmd/glean/         CLI entry point (cobra)
internal/
  catalog/         Pattern definitions and AST detectors
    pattern.go     Pattern and Match types, All() / ByKind() accessors
    detect.go      Shared helpers: snippet(), posOf(), hasIdent()
    algorithms.go  Algorithm patterns (binary search, BFS, DFS, DP, ...)
    datastructures.go  Data structure patterns (trie, heap, union-find, ...)
    concurrency.go Concurrency patterns (worker pool, errgroup, pipeline, ...)
    idioms.go      Go idiom patterns (functional options, error wrapping, ...)
  analyzer/        Parses Go source, runs all detectors, returns PatternMatch list
  fetch/           Fetches PRs page by page, deduplicates blobs by SHA
  github/          Thin GitHub API wrapper (search, PR files, blob content)
  profile/         Aggregates PatternHits per author
  report/          Three output modes: terminal (PrintProfile), JSON, markdown prompt (PrintPrompt)
```

## Key design decisions

- **Pattern catalog is the core** — each pattern has a `Detect func(*ast.File, *token.FileSet, []byte) []Match`. Adding a pattern means adding one entry to the relevant `var patterns` slice.
- **Detectors use source strings, not pure AST** — for patterns where the signal is in the text (e.g. `left < right`, `for len(queue)`), `posOf()` finds the exact byte offset so snippets center on the match line, not the function declaration.
- **SHA deduplication** — `processPR` skips blobs already seen. The same file modified across 10 PRs is analysed once.
- **Limit counts Go-file PRs only** — PRs with no `.go` files don't consume the `--limit` budget.
- **No repo subcommand** — user-centric only. One author at a time.

## Adding a new pattern

1. Pick the right file (`algorithms.go`, `datastructures.go`, `concurrency.go`, `idioms.go`).
2. Add a `*Pattern` entry to the `var` slice with `ID`, `Name`, `Kind`, `Description`, `WhyItMatters`, and `Detect`.
3. Write the `Detect` func — use `posOf()` to center the snippet on the actual match, not `fn.Pos()`.
4. Prefer source-string checks over loose AST identifier counting to avoid false positives.
