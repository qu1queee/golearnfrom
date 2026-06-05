package catalog

import (
	"bytes"
	"go/ast"
	"go/token"
	"strings"
)

// snippet extracts source lines around a position.
func snippet(fset *token.FileSet, pos token.Pos, src []byte, context int) string {
	position := fset.Position(pos)
	lines := bytes.Split(src, []byte("\n"))
	start := position.Line - 1 - context
	end := position.Line + context
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	var sb strings.Builder
	for _, l := range lines[start:end] {
		sb.Write(l)
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}

// hasIdent checks whether an identifier with the given name appears in node.
func hasIdent(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			found = true
		}
		return !found
	})
	return found
}

// importsPackage reports whether the file imports the given path.
func importsPackage(file *ast.File, path string) bool {
	for _, imp := range file.Imports {
		v := strings.Trim(imp.Path.Value, `"`)
		if v == path {
			return true
		}
	}
	return false
}

// posOf returns the token.Pos of the first occurrence of needle in src[offset:].
// Returns token.NoPos if not found. Use this to center snippets on the actual match line
// rather than the surrounding function declaration.
func posOf(fset *token.FileSet, file *ast.File, src []byte, offset int, needle string) token.Pos {
	idx := bytes.Index(src[offset:], []byte(needle))
	if idx < 0 {
		return token.NoPos
	}
	return fset.File(file.Pos()).Pos(offset + idx)
}
