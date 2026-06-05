package catalog

import (
	"go/ast"
	"go/token"
	"strings"
)

var algorithms = []*Pattern{
	{
		ID:           "binary-search",
		Name:         "Binary Search",
		Kind:         KindAlgorithm,
		Description:  "Divides a sorted search space in half each iteration using lo/hi/mid indices.",
		WhyItMatters: "Core pattern for O(log n) search; appears in almost every algorithm interview.",
		Detect:       detectBinarySearch,
	},
	{
		ID:           "two-pointers",
		Name:         "Two Pointers",
		Kind:         KindAlgorithm,
		Description:  "Uses two indices moving toward or away from each other over a slice.",
		WhyItMatters: "Reduces O(n²) brute-force to O(n) for many array/string problems.",
		Detect:       detectTwoPointers,
	},
	{
		ID:           "sliding-window",
		Name:         "Sliding Window",
		Kind:         KindAlgorithm,
		Description:  "Maintains a dynamic window over a sequence, expanding or shrinking based on a condition.",
		WhyItMatters: "Solves subarray/substring optimisation problems in O(n).",
		Detect:       detectSlidingWindow,
	},
	{
		ID:           "bfs",
		Name:         "Breadth-First Search",
		Kind:         KindAlgorithm,
		Description:  "Uses a queue and visited set to explore a graph level by level.",
		WhyItMatters: "Shortest-path and level-order traversal in trees and graphs.",
		Detect:       detectBFS,
	},
	{
		ID:           "dfs",
		Name:         "Depth-First Search",
		Kind:         KindAlgorithm,
		Description:  "Recursive or stack-based graph traversal that goes deep before backtracking.",
		WhyItMatters: "Connectivity, cycle detection, topological sort — ubiquitous in interviews.",
		Detect:       detectDFS,
	},
	{
		ID:           "dynamic-programming",
		Name:         "Dynamic Programming",
		Kind:         KindAlgorithm,
		Description:  "Memoisation or tabulation to avoid recomputing overlapping subproblems.",
		WhyItMatters: "Essential for optimisation problems; signals strong algorithmic thinking.",
		Detect:       detectDP,
	},
	{
		ID:           "backtracking",
		Name:         "Backtracking",
		Kind:         KindAlgorithm,
		Description:  "Recursive search that undoes a choice before exploring the next branch.",
		WhyItMatters: "Permutations, combinations, constraint satisfaction — classic interview territory.",
		Detect:       detectBacktracking,
	},
	{
		ID:           "prefix-sum",
		Name:         "Prefix Sum",
		Kind:         KindAlgorithm,
		Description:  "Precomputes cumulative sums to answer range queries in O(1).",
		WhyItMatters: "Turns repeated range-sum queries from O(n) each into O(1) after O(n) setup.",
		Detect:       detectPrefixSum,
	},
}

func detectBinarySearch(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
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
		// mid must be computed from lo/hi bounds — the defining binary search arithmetic
		hasMidCalc := strings.Contains(fnSrc, "lo+(hi-lo)/2") ||
			strings.Contains(fnSrc, "lo + (hi-lo)/2") ||
			strings.Contains(fnSrc, "(lo+hi)/2") ||
			strings.Contains(fnSrc, "(lo + hi) / 2") ||
			strings.Contains(fnSrc, "(left+right)/2") ||
			strings.Contains(fnSrc, "(left + right) / 2") ||
			strings.Contains(fnSrc, "left+(right-left)/2") ||
			strings.Contains(fnSrc, "left + (right-left)/2")
		if !hasMidCalc {
			return true
		}
		// mid must be used as a slice index: arr[mid]
		hasMidIndex := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			idx, ok := inner.(*ast.IndexExpr)
			if !ok {
				return true
			}
			if id, ok := idx.Index.(*ast.Ident); ok && id.Name == "mid" {
				hasMidIndex = true
			}
			return true
		})
		if hasMidIndex {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 3),
			})
		}
		return true
	})
	return matches
}

func detectTwoPointers(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
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
		// Require opposing movement: one pointer increments, the other decrements.
		hasInc := strings.Contains(fnSrc, "left++") || strings.Contains(fnSrc, "lo++")
		hasDec := strings.Contains(fnSrc, "right--") || strings.Contains(fnSrc, "hi--")
		if !hasInc || !hasDec {
			return true
		}
		// Require a converging loop condition: left < right (or equivalent).
		hasConverge := strings.Contains(fnSrc, "left < right") ||
			strings.Contains(fnSrc, "left <= right") ||
			strings.Contains(fnSrc, "lo < hi") ||
			strings.Contains(fnSrc, "lo <= hi")
		if hasConverge {
			matchPos := posOf(fset, file, src, start, "left++")
			if !matchPos.IsValid() {
				matchPos = posOf(fset, file, src, start, "lo++")
			}
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

func detectSlidingWindow(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
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
		// Both pointers advance in the same direction (right expands, left shrinks window).
		hasBothForward := strings.Contains(fnSrc, "left++") && strings.Contains(fnSrc, "right++")
		if !hasBothForward {
			return true
		}
		// Must also have a shrink/constraint condition: left advances conditionally, not just always.
		// Detect: "left++" inside an if-block or after a condition check (common shrink signal).
		hasShrink := strings.Contains(fnSrc, "right-left") ||
			strings.Contains(fnSrc, "right - left") ||
			strings.Contains(fnSrc, "windowSize") ||
			strings.Contains(fnSrc, "maxLen") ||
			strings.Contains(fnSrc, "minLen")
		if hasShrink {
			matchPos := posOf(fset, file, src, start, "right++")
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

func detectBFS(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
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
		// Require the canonical BFS loop: consuming the queue until empty.
		hasQueueLoop := strings.Contains(fnSrc, "for len(queue)") ||
			strings.Contains(fnSrc, "for len(bfsQueue)")
		if !hasQueueLoop {
			return true
		}
		// Require a visited/seen guard to avoid re-processing nodes.
		hasVisited := strings.Contains(fnSrc, "visited") || strings.Contains(fnSrc, "seen")
		if hasVisited {
			needle := "for len(queue)"
			if strings.Contains(fnSrc, "for len(bfsQueue)") {
				needle = "for len(bfsQueue)"
			}
			matchPos := posOf(fset, file, src, start, needle)
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

func detectDFS(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		// DFS: recursive function calling itself with visited map or stack usage
		fnName := fn.Name.Name
		if !strings.Contains(strings.ToLower(fnName), "dfs") &&
			!strings.Contains(strings.ToLower(fnName), "depth") {
			// check if it's a recursive fn with visited
			hasSelfCall := false
			hasVisited := false
			ast.Inspect(fn.Body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == fnName {
						hasSelfCall = true
					}
				}
				if id, ok := inner.(*ast.Ident); ok {
					if id.Name == "visited" || id.Name == "seen" || id.Name == "stack" {
						hasVisited = true
					}
				}
				return true
			})
			if !hasSelfCall || !hasVisited {
				return true
			}
		}
		pos := fset.Position(fn.Pos())
		matches = append(matches, Match{
			File:    pos.Filename,
			Line:    pos.Line,
			Snippet: snippet(fset, fn.Pos(), src, 4),
		})
		return true
	})
	return matches
}

func detectDP(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		hasDPTable := false
		hasMemo := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if id, ok := inner.(*ast.Ident); ok {
				switch id.Name {
				case "dp", "memo", "cache", "table", "mem":
					hasDPTable = true
				}
			}
			// detect make([][]int, ...) — 2D table
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "make" {
				if len(call.Args) > 0 {
					ast.Inspect(call.Args[0], func(t ast.Node) bool {
						if _, ok := t.(*ast.ArrayType); ok {
							hasMemo = true
						}
						return true
					})
				}
			}
			return true
		})
		if hasDPTable && hasMemo {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 4),
			})
		}
		return true
	})
	return matches
}

func detectBacktracking(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		fnName := fn.Name.Name
		// look for recursive function + append + slice trimming (undo step)
		hasSelfCall := false
		hasAppend := false
		hasUndo := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok {
				if id.Name == fnName {
					hasSelfCall = true
				}
				if id.Name == "append" {
					hasAppend = true
				}
			}
			return true
		})
		// undo step: path = path[:len(path)-1]
		srcStr := string(src)
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if end > start && end <= len(src) {
			fnSrc := srcStr[start:end]
			if strings.Contains(fnSrc, "[:len(") || strings.Contains(fnSrc, "= path[:") || strings.Contains(fnSrc, "= curr[:") {
				hasUndo = true
			}
		}
		if hasSelfCall && hasAppend && hasUndo {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 4),
			})
		}
		return true
	})
	return matches
}

func detectPrefixSum(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
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
		// Must be a numeric slice, not a string variable named "prefix".
		// Require make([]int/[]float allocation for the prefix array.
		hasPrefixSlice := (strings.Contains(fnSrc, "prefix") || strings.Contains(fnSrc, "prefixSum") || strings.Contains(fnSrc, "cumSum")) &&
			(strings.Contains(fnSrc, "make([]int") || strings.Contains(fnSrc, "make([]float"))
		if !hasPrefixSlice {
			return true
		}
		// Require the accumulation pattern: prefix[i] = prefix[i-1] + ...
		hasBuild := strings.Contains(fnSrc, "prefix[i]") ||
			strings.Contains(fnSrc, "prefix[i-1]") ||
			strings.Contains(fnSrc, "prefixSum[i]") ||
			strings.Contains(fnSrc, "cumSum[i]")
		if hasBuild {
			needle := "prefix[i]"
			if strings.Contains(fnSrc, "prefixSum[i]") {
				needle = "prefixSum[i]"
			} else if strings.Contains(fnSrc, "cumSum[i]") {
				needle = "cumSum[i]"
			}
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
