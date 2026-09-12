package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Platform method/key conventions are useful for explicit rejection recovery,
// but do not establish that a provider deduplicates an ambiguous write.
func retrofitPlatformRetryPolicy(data []byte, c *cluster, path string) []byte {
	if !bytes.Contains(data, []byte("platform.CanRetryRequest(")) {
		return data
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, 0)
	if err != nil {
		panic(err)
	}
	source := func(n ast.Node) string {
		return string(data[fset.Position(n.Pos()).Offset:fset.Position(n.End()).Offset])
	}
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assignment, ok := n.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || source(assignment.Lhs[0]) != "canRetryAmbiguousFailure" {
				return true
			}
			original := source(assignment.Rhs[0])
			if !strings.Contains(original, "platform.CanRetryRequest(") {
				return true
			}
			const platformPolicy = "platform.CanRetryRequest(method, requestIdempotencyKey(c.Config, headerOverrides))"
			const safeMethods = "method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions"
			var replacement string
			switch original {
			case "readOnlyIntent || " + platformPolicy:
				replacement = "readOnlyIntent || " + safeMethods
			case "readOnlyIntent || (!mutationIntent && " + platformPolicy + ")":
				replacement = "readOnlyIntent || (!mutationIntent && (" + safeMethods + "))"
			default:
				panic(path + ": unreviewed platform ambiguous-retry policy; manual safety review required")
			}
			var rateEdits []edit
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				branch, ok := n.(*ast.IfStmt)
				if !ok || (!hasRetryNode(branch.Cond, "resp.StatusCode == 429", source) && !hasRetryNode(branch.Cond, "resp.StatusCode == http.StatusTooManyRequests", source)) {
					return true
				}
				ast.Inspect(branch, func(n ast.Node) bool {
					if name, ok := n.(*ast.Ident); ok && name.Name == "canRetryAmbiguousFailure" {
						rateEdits = append(rateEdits, edit{fset.Position(name.Pos()).Offset, fset.Position(name.End()).Offset, "canRetryRejectedRequest"})
					}
					return true
				})
				return false
			})
			if len(rateEdits) > 0 && !strings.Contains(source(fn.Body), "canRetryRejectedRequest :=") {
				replacement += "\n\t// Preserve bounded recovery from explicit rate-limit rejection separately.\n\tcanRetryRejectedRequest := " + original
			}
			edits = append(edits, edit{fset.Position(assignment.Rhs[0].Pos()).Offset, fset.Position(assignment.Rhs[0].End()).Offset, replacement})
			edits = append(edits, rateEdits...)
			return true
		})
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, e := range edits {
		data = append(append(append([]byte{}, data[:e.start]...), []byte(e.text)...), data[e.end:]...)
	}
	if len(edits) > 0 {
		c.files[path] = true
	}
	return data
}

// The identifier is not proof of safety. Reject a widened or unfamiliar
// declaration even when every retry branch still mentions the same guard.
func validateAmbiguousRetryPolicies(data []byte, path string) {
	if !bytes.Contains(data, []byte("canRetryAmbiguousFailure")) {
		return
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, 0)
	if err != nil {
		panic(err)
	}
	normalize := func(s string) string { return strings.Join(strings.Fields(s), "") }
	const safeMethods = "method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions"
	allowed := map[string]bool{
		normalize(safeMethods):                                                    true,
		normalize("readOnlyIntent || " + safeMethods):                             true,
		normalize("readOnlyIntent || (!mutationIntent && (" + safeMethods + "))"): true,
	}
	check := func(name string, value ast.Expr) {
		if name != "canRetryAmbiguousFailure" {
			return
		}
		source := string(data[fset.Position(value.Pos()).Offset:fset.Position(value.End()).Offset])
		if !allowed[normalize(source)] {
			panic(path + ": widened or unreviewed ambiguous-retry declaration: " + source)
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, name := range n.Lhs {
				if ident, ok := name.(*ast.Ident); ok && len(n.Rhs) == len(n.Lhs) {
					check(ident.Name, n.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if len(n.Values) == len(n.Names) {
					check(name.Name, n.Values[i])
				}
			}
		}
		return true
	})
}
