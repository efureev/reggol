package log_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestFacadeParity keeps this package in step with reggol.Logger.
//
// The facade is written by hand, and a missing wrapper is invisible: nothing
// fails to build, users of the facade simply do not get the method. v0 had
// drifted — GetLevel, Println and Write existed on Logger and were absent here.
// This test turns that invariant into something the compiler cannot skip.
func TestFacadeParity(t *testing.T) {
	loggerMethods := exportedMethodsOf(t, filepath.Join("..", "logger.go"), "Logger")
	facadeFuncs := exportedFuncsOf(t, "log.go")

	// Methods that intentionally have no package-level counterpart.
	exempt := map[string]bool{
		// Constructors and per-instance configuration return a Logger; the
		// package-level equivalents are L() and SetLogger().
		"Encoder": true,
	}

	missing := make([]string, 0, len(loggerMethods))

	for _, m := range loggerMethods {
		if exempt[m] || facadeFuncs[m] {
			continue
		}

		missing = append(missing, m)
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("reggol.Logger methods without a facade wrapper: %s\n"+
			"add them to log/log.go, or list them in the exempt set with a reason",
			strings.Join(missing, ", "))
	}
}

// exportedMethodsOf returns the exported method names declared on recv.
func exportedMethodsOf(t *testing.T, path, recv string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	out := make([]string, 0, len(file.Decls))

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}

		if !fn.Name.IsExported() || receiverName(fn.Recv.List[0].Type) != recv {
			continue
		}

		out = append(out, fn.Name.Name)
	}

	return out
}

// exportedFuncsOf returns the exported package-level function names in path.
func exportedFuncsOf(t *testing.T, path string) map[string]bool {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	out := map[string]bool{}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !fn.Name.IsExported() {
			continue
		}

		out[fn.Name.Name] = true
	}

	return out
}

// receiverName unwraps a receiver type to its bare identifier.
func receiverName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}

	return ""
}
