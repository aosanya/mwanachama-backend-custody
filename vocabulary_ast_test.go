package mwanachamacustody_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// constantsByType reads the declared constants out of one file and groups
// their string values by the type they are declared with. Reading the source
// rather than a hand-kept list is the point: a constant added to the file
// and to no spec has to be visible to the test that holds the two together.
func constantsByType(t *testing.T, path string) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	out := map[string][]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		typeName := ""
		for _, s := range gen.Specs {
			value, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if id, ok := value.Type.(*ast.Ident); ok {
				typeName = id.Name
			}
			if typeName == "" {
				continue
			}
			for _, v := range value.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				out[typeName] = append(out[typeName], unquoted)
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s declares no typed string constants, so the agreement test proves nothing", path)
	}
	return out
}
