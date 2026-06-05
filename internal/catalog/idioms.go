package catalog

import (
	"go/ast"
	"go/token"
	"strings"
)

var idioms = []*Pattern{
	{
		ID:           "functional-options",
		Name:         "Functional Options",
		Kind:         KindIdiom,
		Description:  "Option type is a function that mutates a config struct, passed as variadic args.",
		WhyItMatters: "Extensible APIs without breaking callers — the idiomatic Go way to handle optional config.",
		Detect:       detectFunctionalOptions,
	},
	{
		ID:           "interface-satisfaction",
		Name:         "Interface Satisfaction Check",
		Kind:         KindIdiom,
		Description:  "var _ InterfaceName = (*ConcreteType)(nil) — compile-time assertion.",
		WhyItMatters: "Catches missing method implementations at compile time, not runtime.",
		Detect:       detectInterfaceSatisfaction,
	},
	{
		ID:           "error-wrapping",
		Name:         "Error Wrapping with %w",
		Kind:         KindIdiom,
		Description:  "fmt.Errorf(\"context: %w\", err) to add context while preserving unwrap chain.",
		WhyItMatters: "Allows errors.Is / errors.As to work through the chain; idiomatic since Go 1.13.",
		Detect:       detectErrorWrapping,
	},
	{
		ID:           "sentinel-errors",
		Name:         "Sentinel Errors",
		Kind:         KindIdiom,
		Description:  "Package-level var ErrFoo = errors.New(...) for expected error conditions.",
		WhyItMatters: "Lets callers use errors.Is(err, ErrFoo) for precise error matching.",
		Detect:       detectSentinelErrors,
	},
	{
		ID:           "table-driven-tests",
		Name:         "Table-Driven Tests",
		Kind:         KindIdiom,
		Description:  "Slice of anonymous structs with input/expected fields, iterated with t.Run.",
		WhyItMatters: "The standard Go testing idiom — reduces boilerplate, easy to extend.",
		Detect:       detectTableDrivenTests,
	},
	{
		ID:           "context-first-param",
		Name:         "Context as First Parameter",
		Kind:         KindIdiom,
		Description:  "Functions that cross API or goroutine boundaries accept ctx context.Context as first arg.",
		WhyItMatters: "Required for cancellation and deadline propagation; Go convention.",
		Detect:       detectContextFirstParam,
	},
	{
		ID:           "defer-cleanup",
		Name:         "Defer for Cleanup",
		Kind:         KindIdiom,
		Description:  "defer resource.Close() immediately after acquiring a resource.",
		WhyItMatters: "Guarantees cleanup even on early returns or panics — the idiomatic resource lifecycle.",
		Detect:       detectDeferCleanup,
	},
	{
		ID:           "small-interfaces",
		Name:         "Small Interfaces (1-2 methods)",
		Kind:         KindIdiom,
		Description:  "Interface definitions with one or two methods, often with -er suffix.",
		WhyItMatters: "Prefer small interfaces — io.Reader, io.Writer are the model. Easier to mock and compose.",
		Detect:       detectSmallInterfaces,
	},
}

func detectFunctionalOptions(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	// Collect option type names: must be a func(*T) whose type name contains "Option".
	// This excludes callbacks, patchers, handlers, and other func(*T) types.
	optionTypes := map[string]bool{}
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
			if !strings.Contains(strings.ToLower(ts.Name.Name), "option") {
				continue
			}
			ft, ok := ts.Type.(*ast.FuncType)
			if !ok {
				continue
			}
			if ft.Params == nil || len(ft.Params.List) != 1 {
				continue
			}
			if _, ok := ft.Params.List[0].Type.(*ast.StarExpr); ok {
				optionTypes[ts.Name.Name] = true
				pos := fset.Position(ts.Pos())
				matches = append(matches, Match{
					File:    pos.Filename,
					Line:    pos.Line,
					Snippet: snippet(fset, ts.Pos(), src, 5),
				})
			}
		}
	}
	// Also detect With* constructor functions that return an option type.
	if len(optionTypes) == 0 {
		return matches
	}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "With") {
			return true
		}
		if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
			return true
		}
		if id, ok := fn.Type.Results.List[0].Type.(*ast.Ident); ok && optionTypes[id.Name] {
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

func detectInterfaceSatisfaction(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// var _ Interface = (*Type)(nil)
			if len(vs.Names) == 1 && vs.Names[0].Name == "_" && vs.Type != nil {
				pos := fset.Position(vs.Pos())
				matches = append(matches, Match{
					File:    pos.Filename,
					Line:    pos.Line,
					Snippet: snippet(fset, vs.Pos(), src, 1),
				})
			}
		}
	}
	return matches
}

func detectErrorWrapping(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "fmt" && sel.Sel.Name == "Errorf" {
			// check for %w
			if len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && strings.Contains(lit.Value, "%w") {
					pos := fset.Position(call.Pos())
					matches = append(matches, Match{
						File:    pos.Filename,
						Line:    pos.Line,
						Snippet: snippet(fset, call.Pos(), src, 1),
					})
				}
			}
		}
		return true
	})
	return matches
}

func detectSentinelErrors(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if strings.HasPrefix(name.Name, "Err") || strings.HasPrefix(name.Name, "err") {
					pos := fset.Position(vs.Pos())
					matches = append(matches, Match{
						File:    pos.Filename,
						Line:    pos.Line,
						Snippet: snippet(fset, vs.Pos(), src, 1),
					})
					break
				}
			}
		}
	}
	return matches
}

func detectTableDrivenTests(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	if !strings.HasSuffix(fset.File(file.Pos()).Name(), "_test.go") {
		return matches
	}
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		if !strings.HasPrefix(fn.Name.Name, "Test") {
			return true
		}
		// look for []struct{ ... } inside the test
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			cl, ok := inner.(*ast.CompositeLit)
			if !ok {
				return true
			}
			at, ok := cl.Type.(*ast.ArrayType)
			if !ok {
				return true
			}
			if _, ok := at.Elt.(*ast.StructType); ok {
				pos := fset.Position(fn.Pos())
				matches = append(matches, Match{
					File:    pos.Filename,
					Line:    pos.Line,
					Snippet: snippet(fset, fn.Pos(), src, 4),
				})
				return false
			}
			return true
		})
		return true
	})
	return matches
}

func detectContextFirstParam(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
			return true
		}
		first := fn.Type.Params.List[0]
		sel, ok := first.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "context" && sel.Sel.Name == "Context" {
			pos := fset.Position(fn.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, fn.Pos(), src, 2),
			})
		}
		return true
	})
	return matches
}

func detectDeferCleanup(file *ast.File, fset *token.FileSet, src []byte) []Match {
	var matches []Match
	ast.Inspect(file, func(n ast.Node) bool {
		ds, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}
		call, ok := ds.Call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := call.Sel.Name
		if name == "Close" || name == "Done" || name == "Cancel" || name == "Unlock" || name == "RUnlock" || name == "Stop" {
			pos := fset.Position(ds.Pos())
			matches = append(matches, Match{
				File:    pos.Filename,
				Line:    pos.Line,
				Snippet: snippet(fset, ds.Pos(), src, 1),
			})
		}
		return true
	})
	return matches
}

func detectSmallInterfaces(file *ast.File, fset *token.FileSet, src []byte) []Match {
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
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			if iface.Methods != nil && len(iface.Methods.List) <= 2 {
				pos := fset.Position(ts.Pos())
				matches = append(matches, Match{
					File:    pos.Filename,
					Line:    pos.Line,
					Snippet: snippet(fset, ts.Pos(), src, 5),
				})
			}
		}
	}
	return matches
}
