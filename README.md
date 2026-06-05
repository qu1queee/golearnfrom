# glean

Learn Go by studying how experienced developers write it.

## What

`glean` fetches a developer's merged pull requests from GitHub, detects coding patterns in their Go code, and surfaces them with real examples. It understands four kinds of patterns:

- **Algorithms** — binary search, BFS, DFS, dynamic programming, two pointers, sliding window, backtracking, prefix sum
- **Data structures** — linked list, stack, heap, trie, union-find, graph
- **Concurrency** — worker pool, errgroup, pipeline, context cancellation, sync.Once, done channel
- **Go idioms** — functional options, interface satisfaction checks, error wrapping, sentinel errors, table-driven tests, defer for cleanup, small interfaces, context as first param

## Why

Reading documentation teaches you the language. Reading real production code teaches you how to use it. `glean` makes that systematic — pick a developer whose code you respect, run it against their PRs, and get structured output you can study or pipe to an LLM for explanation.

## How

```bash
export GITHUB_TOKEN=ghp_...   # github.com/settings/tokens — public_repo scope

make build

# read the output
./glean user <username> --repo <owner/repo>

# pipe to claude for tutoring
./glean user <username> --repo <owner/repo> --prompt | claude
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `--repo` | required | repository to search, e.g. `golang/go` |
| `--limit` | 30 | max PRs with Go files to analyse |
| `--snippets` | 2 | code examples shown per pattern |
| `--prompt` | false | output markdown prompt for piping to an LLM |
| `--json` | false | output raw JSON |
| `--verbose` | false | show per-PR and per-file progress |
