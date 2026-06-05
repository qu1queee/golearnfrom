package catalog

import (
	"go/ast"
	"go/token"
	"strings"
)

var datastructures = []*Pattern{
	{
		ID:           "linked-list",
		Name:         "Linked List",
		Kind:         KindDataStructure,
		Description:  "Custom node-based linked list implementation with Next pointer.",
		WhyItMatters: "Pointer manipulation, sentinel nodes, and in-place reversal — common interview basics.",
		Detect:       detectLinkedList,
	},
	{
		ID:           "stack",
		Name:         "Stack",
		Kind:         KindDataStructure,
		Description:  "Slice-backed LIFO stack using append/pop idiom.",
		WhyItMatters: "Expression evaluation, DFS, monotonic stack problems.",
		Detect:       detectStack,
	},
	{
		ID:           "heap",
		Name:         "Heap / Priority Queue",
		Kind:         KindDataStructure,
		Description:  "container/heap implementation or manual heap with Push/Pop methods.",
		WhyItMatters: "Top-K problems, Dijkstra, scheduling — priority queues appear constantly.",
		Detect:       detectHeap,
	},
	{
		ID:           "trie",
		Name:         "Trie",
		Kind:         KindDataStructure,
		Description:  "Prefix tree with fixed-size or map-based children.",
		WhyItMatters: "String prefix search, autocomplete, word dictionaries.",
		Detect:       detectTrie,
	},
	{
		ID:           "union-find",
		Name:         "Union-Find (Disjoint Set)",
		Kind:         KindDataStructure,
		Description:  "parent/rank arrays with find (path compression) and union by rank.",
		WhyItMatters: "Connected components, cycle detection in undirected graphs, Kruskal's MST.",
		Detect:       detectUnionFind,
	},
	{
		ID:           "graph-adjacency",
		Name:         "Graph (Adjacency List)",
		Kind:         KindDataStructure,
		Description:  "Graph represented as map or slice of slices for neighbor lists.",
		WhyItMatters: "Foundation for BFS/DFS/shortest-path — nearly every graph problem needs it.",
		Detect:       detectGraph,
	},
}

func detectLinkedList(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				starExpr, ok := field.Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				if id, ok := starExpr.X.(*ast.Ident); ok && id.Name == ts.Name.Name {
					pos := fset.Position(ts.Pos())
					matches = append(matches, Match{
						File:    pos.Filename,
						Line:    pos.Line,
						Snippet: snippet(fset, ts.Pos(), src, 4),
					})
					break
				}
			}
		}
	}
	return matches
}

func detectStack(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		hasStack := false
		hasPop := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if id, ok := inner.(*ast.Ident); ok && (id.Name == "stack" || id.Name == "stk") {
				hasStack = true
			}
			return true
		})
		// pop pattern: stack = stack[:len(stack)-1]
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if end > start && end <= len(src) {
			fnSrc := string(src[start:end])
			if strings.Contains(fnSrc, "[:len(stack)-1]") || strings.Contains(fnSrc, "[:len(stk)-1]") {
				hasPop = true
			}
		}
		if hasStack && hasPop {
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

func detectHeap(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	if !importsPackage(file, "container/heap") {
		return matches
	}
	// find the type that implements heap.Interface
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fn.Name.Name == "Push" || fn.Name.Name == "Pop" || fn.Name.Name == "Less" {
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

func detectTrie(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			// The defining signal of a Trie node is a field specifically named
			// "children" or "Children" whose element type is a pointer to a struct.
			// This excludes generic domain Node types (k8s, graphs, trees).
			for _, field := range st.Fields.List {
				isChildren := false
				for _, name := range field.Names {
					if strings.EqualFold(name.Name, "children") {
						isChildren = true
						break
					}
				}
				if !isChildren {
					continue
				}
				switch ft := field.Type.(type) {
				case *ast.ArrayType:
					if _, ok := ft.Elt.(*ast.StarExpr); ok {
						pos := fset.Position(ts.Pos())
						matches = append(matches, Match{
							File:    pos.Filename,
							Line:    pos.Line,
							Snippet: snippet(fset, ts.Pos(), src, 6),
						})
					}
				case *ast.MapType:
					if _, ok := ft.Value.(*ast.StarExpr); ok {
						pos := fset.Position(ts.Pos())
						matches = append(matches, Match{
							File:    pos.Filename,
							Line:    pos.Line,
							Snippet: snippet(fset, ts.Pos(), src, 6),
						})
					}
				}
			}
		}
	}
	return matches
}

func detectUnionFind(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		name := strings.ToLower(fn.Name.Name)
		if name != "find" {
			return true
		}
		start := fset.Position(fn.Body.Lbrace).Offset
		end := fset.Position(fn.Body.Rbrace).Offset
		if end <= start || end > len(src) {
			return true
		}
		fnSrc := string(src[start:end])
		// Path compression is the definitive signal: parent[x] = find(parent[x]).
		// This self-referential assignment distinguishes UF from any function named find.
		hasPathCompression := strings.Contains(fnSrc, "parent[") &&
			strings.Contains(fnSrc, "find(parent[")
		if hasPathCompression {
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

func detectGraph(file *ast.File, fset *token.FileSet, src []byte) []Match {
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
		// Require the adjacency list to be allocated as a slice-of-slices or map-of-slices.
		// This rules out domain types that happen to use the word "graph".
		hasAdjAlloc := strings.Contains(fnSrc, "make([][]") ||
			strings.Contains(fnSrc, "make(map[int][]") ||
			strings.Contains(fnSrc, "make(map[string][]")
		if !hasAdjAlloc {
			return true
		}
		hasAdjName := strings.Contains(fnSrc, "graph") ||
			strings.Contains(fnSrc, "adj") ||
			strings.Contains(fnSrc, "neighbors")
		if hasAdjName {
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
