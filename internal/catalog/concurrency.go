package catalog

import (
	"go/ast"
	"go/token"
	"strings"
)

var concurrency = []*Pattern{
	{
		ID:           "worker-pool",
		Name:         "Worker Pool",
		Kind:         KindConcurrency,
		Description:  "Fixed number of goroutines consuming from a shared jobs channel.",
		WhyItMatters: "Bounded parallelism — prevents goroutine explosion under load.",
		Detect:       detectWorkerPool,
	},
	{
		ID:           "errgroup",
		Name:         "errgroup Fan-out",
		Kind:         KindConcurrency,
		Description:  "golang.org/x/sync/errgroup to launch goroutines and collect the first error.",
		WhyItMatters: "Idiomatic Go for concurrent work with error propagation.",
		Detect:       detectErrgroup,
	},
	{
		ID:           "pipeline",
		Name:         "Pipeline (channel chaining)",
		Kind:         KindConcurrency,
		Description:  "Stages connected by channels; each stage reads from upstream and writes downstream.",
		WhyItMatters: "Composable streaming data processing — a Go concurrency classic.",
		Detect:       detectPipeline,
	},
	{
		ID:           "context-cancellation",
		Name:         "Context Cancellation",
		Kind:         KindConcurrency,
		Description:  "Uses context.WithCancel or context.WithTimeout to propagate cancellation.",
		WhyItMatters: "Correct goroutine lifecycle management; required in production Go.",
		Detect:       detectContextCancel,
	},
	{
		ID:           "sync-once",
		Name:         "sync.Once (Lazy Init)",
		Kind:         KindConcurrency,
		Description:  "sync.Once to initialise a value exactly once across goroutines.",
		WhyItMatters: "Safe singleton or lazy initialisation without a mutex guard.",
		Detect:       detectSyncOnce,
	},
	{
		ID:           "done-channel",
		Name:         "Done Channel",
		Kind:         KindConcurrency,
		Description:  "Closing a chan struct{} to broadcast a stop signal to multiple goroutines.",
		WhyItMatters: "Simple, zero-allocation fan-out cancellation before context was idiomatic.",
		Detect:       detectDoneChan,
	},
}

func detectWorkerPool(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		hasJobs := false
		hasGoRoutineLoop := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if id, ok := inner.(*ast.Ident); ok {
				if id.Name == "jobs" || id.Name == "work" || id.Name == "tasks" || id.Name == "workers" {
					hasJobs = true
				}
			}
			// for range loop launching goroutines
			rangeStmt, ok := inner.(*ast.RangeStmt)
			if !ok {
				return true
			}
			ast.Inspect(rangeStmt, func(r ast.Node) bool {
				if _, ok := r.(*ast.GoStmt); ok {
					hasGoRoutineLoop = true
				}
				return true
			})
			return true
		})
		if hasJobs && hasGoRoutineLoop {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 5),
			})
		}
		return true
	})
	return matches
}

func detectErrgroup(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	if !importsPackage(file, "golang.org/x/sync/errgroup") {
		return matches
	}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		if hasIdent(fn.Body, "errgroup") || hasIdent(fn.Body, "eg") || hasIdent(fn.Body, "g") {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 5),
			})
		}
		return true
	})
	return matches
}

func detectPipeline(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		// pipeline: function returns a channel and ranges over input channel
		returnsChain := false
		if fn.Type.Results != nil {
			for _, r := range fn.Type.Results.List {
				if _, ok := r.Type.(*ast.ChanType); ok {
					returnsChain = true
				}
			}
		}
		if !returnsChain {
			return true
		}
		hasRangeOnChan := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			rs, ok := inner.(*ast.RangeStmt)
			if !ok {
				return true
			}
			ast.Inspect(rs.X, func(x ast.Node) bool {
				if _, ok := x.(*ast.Ident); ok {
					hasRangeOnChan = true
				}
				return true
			})
			return true
		})
		if hasRangeOnChan {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 5),
			})
		}
		return true
	})
	return matches
}

func detectContextCancel(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	if !importsPackage(file, "context") {
		return matches
	}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if end <= start || end > len(src) {
			return true
		}
		fnSrc := string(src[start:end])
		needle := ""
		switch {
		case strings.Contains(fnSrc, "context.WithCancel"):
			needle = "context.WithCancel"
		case strings.Contains(fnSrc, "context.WithTimeout"):
			needle = "context.WithTimeout"
		case strings.Contains(fnSrc, "context.WithDeadline"):
			needle = "context.WithDeadline"
		}
		if needle != "" {
			matchPos := posOf(fset, file, src, start, needle)
			if !matchPos.IsValid() {
				matchPos = fn.Pos()
			}
			pos := fset.Position(matchPos)
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, matchPos, src, 3),
			})
		}
		return true
	})
	return matches
}

func detectSyncOnce(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	if !importsPackage(file, "sync") {
		return matches
	}
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "sync" && sel.Sel.Name == "Once" {
				pos := fset.Position(sel.Pos())
				matches = append(matches, Match{
					File:    pos.Filename,
					Line:    pos.Line,
					Snippet: snippet(fset, sel.Pos(), src, 4),
				})
			}
		}
		return true
	})
	return matches
}

func detectDoneChan(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		hasDone := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if id, ok := inner.(*ast.Ident); ok && (id.Name == "done" || id.Name == "quit" || id.Name == "stop") {
				hasDone = true
			}
			return true
		})
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if end <= start || end > len(src) {
			return true
		}
		fnSrc := string(src[start:end])
		if hasDone && strings.Contains(fnSrc, "struct{}") && strings.Contains(fnSrc, "close(") {
			matchPos := posOf(fset, file, src, start, "close(")
			if !matchPos.IsValid() {
				matchPos = fn.Pos()
			}
			pos := fset.Position(matchPos)
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, matchPos, src, 4),
			})
		}
		return true
	})
	return matches
}
