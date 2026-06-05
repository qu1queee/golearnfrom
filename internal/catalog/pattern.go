package catalog

import (
	"go/ast"
	"go/token"
)

type Kind string

const (
	KindAlgorithm    Kind = "algorithm"
	KindDataStructure Kind = "datastructure"
	KindConcurrency  Kind = "concurrency"
	KindIdiom        Kind = "idiom"
)

// Match is a single occurrence of a pattern found in a file.
type Match struct {
	File    string
	Line    int
	Snippet string
}

// Pattern is a named, detectable coding pattern.
type Pattern struct {
	ID          string
	Name        string
	Kind        Kind
	Description string
	WhyItMatters string
	Detect      func(file *ast.File, fset *token.FileSet, src []byte) []Match
}

// All returns every pattern in the catalog.
func All() []*Pattern {
	var out []*Pattern
	out = append(out, algorithms...)
	out = append(out, datastructures...)
	out = append(out, concurrency...)
	out = append(out, idioms...)
	return out
}

// ByKind returns patterns filtered by kind.
func ByKind(k Kind) []*Pattern {
	var out []*Pattern
	for _, p := range All() {
		if p.Kind == k {
			out = append(out, p)
		}
	}
	return out
}
