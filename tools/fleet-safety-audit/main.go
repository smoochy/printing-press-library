package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var write = flag.Bool("write", false, "apply safe mechanical retrofits and write reprint guards")
var verbose = flag.Bool("verbose", false, "list every affected file")

type patchRecord struct {
	SchemaVersion            int      `json:"schema_version"`
	ID                       string   `json:"id"`
	AppliedAt                string   `json:"applied_at"`
	BaseRunID                string   `json:"base_run_id,omitempty"`
	BasePrintingPressVersion string   `json:"base_printing_press_version,omitempty"`
	Summary                  string   `json:"summary"`
	Reason                   string   `json:"reason"`
	Files                    []string `json:"files"`
	ValidatedOutcome         string   `json:"validated_outcome"`
}

type cluster struct {
	id, summary, reason, outcome string
	files                        map[string]bool
}

var trueParamGuard = regexp.MustCompile(`(?m)^([\t ]*)if true \{\n[\t ]*(params\[[^\n]+\n)[\t ]*\}`)
var splitParamAssignments = regexp.MustCompile(`(?m)(^[\t ]+params\[[^\n]+\n)\n([\t ]+params\[)`)

func main() {
	flag.Parse()
	root := "library"
	if flag.NArg() == 1 {
		root = flag.Arg(0)
	}
	clusters := []*cluster{
		{id: "ambiguous-write-retries-disabled", summary: "Unprotected writes are not replayed after ambiguous transport or server failures; bounded authentication and rate-limit recovery remain available.", reason: "A failed write may have committed remotely before the client observed the failure. Authentication rejection and rate limiting are distinct recovery paths and must not lose their retry budget.", outcome: "Ambiguous-failure branches are guarded without disabling the shared retry budget.", files: map[string]bool{}},
		{id: "path-parameters-percent-encoded", summary: "Path parameters are percent-encoded as single URL segments, including dot segments.", reason: "Raw reserved characters can change routing semantics, while literal dot segments can be normalized away by URL handling.", outcome: "Every generated path substitution percent-encodes reserved characters and explicitly encodes dot-only values.", files: map[string]bool{}},
		{id: "defaulted-numeric-parameters-emitted", summary: "Defaulted numeric and boolean parameters are emitted without constant-true guards.", reason: "Generated constant conditions obscure parameter presence and are rejected by the fleet parameter-construction audit.", outcome: "The fleet scan finds no constant-true parameter guard.", files: map[string]bool{}},
		{id: "batch-rollback-reports-committed-count", summary: "Batch upserts report zero stored rows whenever the enclosing transaction rolls back.", reason: "The stored count must describe committed rows, not in-memory loop progress; a fatal batch error rolls back earlier inserts.", outcome: "The fleet scan finds no fatal batch-transaction return that reports the running stored count.", files: map[string]bool{}},
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := data
		switch {
		case strings.HasSuffix(path, "/internal/client/client.go"):
			updated = retrofitRetry(updated, clusters[0], path)
			validateAmbiguousRetryPolicies(updated, path)
		case strings.HasSuffix(path, "/internal/cli/helpers.go"):
			updated = retrofitPathEncoding(updated, clusters[1], path)
		case strings.Contains(path, "/internal/cli/"):
			updated = retrofitTrueParamGuards(updated, clusters[2], path)
		case strings.HasSuffix(path, "/internal/store/store.go"):
			updated = retrofitRollbackCount(updated, clusters[3], path)
		}
		if !bytes.Equal(updated, data) && *write {
			formatted, err := format.Source(updated)
			if err != nil {
				return fmt.Errorf("refusing invalid Go rewrite %s: %w", path, err)
			}
			updated = formatted
			if err := os.WriteFile(path, updated, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, c := range clusters {
		paths := sortedKeys(c.files)
		fmt.Printf("%s: %d files\n", c.id, len(paths))
		if *verbose {
			for _, path := range paths {
				fmt.Printf("  %s\n", path)
			}
		}
		if *write {
			if err := writePatchRecords(c, paths); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	if !*write {
		for _, c := range clusters {
			if len(c.files) > 0 {
				os.Exit(1)
			}
		}
	}
}

func retrofitRetry(data []byte, c *cluster, path string) []byte {
	data = retrofitPlatformRetryPolicy(data, c, path)
	if strings.HasSuffix(path, "/marketing/dataforseo/internal/client/client.go") {
		original := data
		old := []byte("return req.Header.Get(\"Idempotency-Key\") != \"\"")
		if bytes.Contains(data, old) {
			data = bytes.Replace(data, old, []byte("// A supplied header alone does not establish provider-side deduplication.\n\treturn false"), 1)
			c.files[path] = true
		}
		data = bytes.Replace(data, []byte("// PATCH: Mutations retry rate limits only when an idempotency key makes replay explicit."), []byte("// PATCH: Preserve the existing keyed-write recovery budget for explicit rate-limit rejection."), 1)
		data = bytes.Replace(data, []byte("if attempt < maxRetries && requestCanRetry(req) {"), []byte("if attempt < maxRetries && (requestCanRetry(req) || req.Header.Get(\"Idempotency-Key\") != \"\") {"), 1)
		if !bytes.Equal(original, data) {
			c.files[path] = true
		}
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, 0)
	if err != nil {
		return data
	}
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	source := func(n ast.Node) string {
		return string(data[fset.Position(n.Pos()).Offset:fset.Position(n.End()).Offset])
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.Contains(source(fn), "maxRetries") {
			continue
		}
		methodParameter := false
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				if name.Name == "method" {
					methodParameter = true
				}
			}
		}
		if !methodParameter || fn.Type.Results == nil || len(fn.Type.Results.List) != 3 {
			continue
		}
		// A transport that immediately refuses writes is already read-only.
		// Do not confuse later cache invalidation or idempotency branches with it.
		if len(fn.Body.List) > 0 {
			if guard, ok := fn.Body.List[0].(*ast.IfStmt); ok && source(guard.Cond) == "method != http.MethodGet" && len(guard.Body.List) == 1 {
				if _, returns := guard.Body.List[0].(*ast.ReturnStmt); returns {
					continue
				}
			}
		}
		// Cross-source failover has its own retry boundary and must be reviewed
		// separately; never guess how to thread intent through another method.
		if strings.Contains(source(fn), "requestBaseURLs()") || fn.Name.Name == "attempt" {
			continue
		}
		var local []edit
		approvedGuard := reviewedRetryGuard(path, fn.Name.Name, data)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			branch, ok := n.(*ast.IfStmt)
			if !ok {
				return true
			}
			if hasRetryNode(branch.Cond, "resp.StatusCode >= 500", source) && hasRetryNode(branch.Cond, "attempt < maxRetries", source) && !hasRetryConjunct(branch.Cond, "canRetryAmbiguousFailure", source) && (approvedGuard == "" || !hasRetryConjunct(branch.Cond, approvedGuard, source)) {
				pos := fset.Position(branch.Cond.End()).Offset
				if hasRetryConjunct(branch.Cond, "resp.StatusCode >= 500", source) && hasRetryConjunct(branch.Cond, "attempt < maxRetries", source) {
					local = append(local, edit{pos, pos, " && canRetryAmbiguousFailure"})
				} else {
					local = append(local, edit{fset.Position(branch.Cond.Pos()).Offset, pos, "(" + source(branch.Cond) + ") && canRetryAmbiguousFailure"})
				}
			}
			return true
		})
		// Find the error block immediately following HTTPClient.Do, rather than
		// changing unrelated retries such as token refresh or rate-limit recovery.
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i, stmt := range block.List {
				if _, ok := stmt.(*ast.AssignStmt); !ok {
					continue
				}
				if i+1 >= len(block.List) || !strings.Contains(source(stmt), ":= c.HTTPClient.Do(req)") {
					continue
				}
				branch, ok := block.List[i+1].(*ast.IfStmt)
				if !ok || source(branch.Cond) != "err != nil" {
					continue
				}
				if transportStopsUnsafeWrite(branch.Body, approvedGuard, source) {
					continue
				}
				for _, stmt := range branch.Body.List {
					assignment, ok := stmt.(*ast.AssignStmt)
					if !ok || len(assignment.Lhs) != 1 || source(assignment.Lhs[0]) != "lastErr" {
						continue
					}
					pos := fset.Position(stmt.End()).Offset
					local = append(local, edit{pos, pos, "\n\t\t\tif !canRetryAmbiguousFailure {\n\t\t\t\treturn nil, 0, lastErr\n\t\t\t}"})
					break
				}
			}
			return true
		})
		if len(local) == 0 {
			continue
		}
		prefix := ""
		if strings.Contains(source(fn.Type), "readOnlyIntent bool") {
			prefix = "readOnlyIntent || "
		}
		if !strings.Contains(source(fn.Body), "canRetryAmbiguousFailure :=") {
			pos := fset.Position(fn.Body.Lbrace).Offset + 1
			local = append(local, edit{pos, pos, "\n\t// Keep authentication and rate-limit recovery available; only ambiguous\n\t// transport/server failures must not replay an unprotected write.\n\tcanRetryAmbiguousFailure := " + prefix + "method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions\n"})
		}
		edits = append(edits, local...)
	}
	if len(edits) == 0 {
		return data
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	for _, e := range edits {
		data = append(append(append([]byte{}, data[:e.start]...), []byte(e.text)...), data[e.end:]...)
	}
	c.files[path] = true
	return data
}

func retrofitPathEncoding(data []byte, c *cluster, path string) []byte {
	if bytes.Contains(data, []byte("cliutil.EscapePathParam(value)")) {
		return data
	}
	beehiivOld := []byte("func escapePathParam(value string) string {\n\treturn strings.ReplaceAll(url.QueryEscape(value), \"+\", \"%20\")\n}")
	if bytes.Contains(data, beehiivOld) {
		c.files[path] = true
		beehiivNew := []byte("func escapePathParam(value string) string {\n\tif value == \".\" || value == \"..\" {\n\t\treturn strings.Repeat(\"%2E\", len(value))\n\t}\n\treturn strings.ReplaceAll(url.QueryEscape(value), \"+\", \"%20\")\n}")
		return bytes.Replace(data, beehiivOld, beehiivNew, 1)
	}
	squarespaceOld := []byte("func escapePathSegment(value string) string {\n\tif !strings.Contains(value, \",\") {\n\t\treturn url.PathEscape(value)\n\t}\n\tparts := strings.Split(value, \",\")\n\tfor i, p := range parts {\n\t\tparts[i] = url.PathEscape(p)\n\t}\n\treturn strings.Join(parts, \",\")\n}")
	if bytes.Contains(data, squarespaceOld) {
		c.files[path] = true
		squarespaceNew := []byte("func escapePathSegment(value string) string {\n\tif !strings.Contains(value, \",\") {\n\t\treturn escapeSinglePathSegment(value)\n\t}\n\tparts := strings.Split(value, \",\")\n\tfor i, p := range parts {\n\t\tparts[i] = escapeSinglePathSegment(p)\n\t}\n\treturn strings.Join(parts, \",\")\n}\n\nfunc escapeSinglePathSegment(value string) string {\n\tif value == \".\" || value == \"..\" {\n\t\treturn strings.Repeat(\"%2E\", len(value))\n\t}\n\treturn url.PathEscape(value)\n}")
		return bytes.Replace(data, squarespaceOld, squarespaceNew, 1)
	}
	oldRaw := []byte("func replacePathParam(path, name, value string) string {\n\treturn strings.ReplaceAll(path, \"{\"+name+\"}\", value)\n}")
	oldEscaped := []byte("func replacePathParam(path, name, value string) string {\n\treturn strings.ReplaceAll(path, \"{\"+name+\"}\", url.PathEscape(value))\n}")
	if !bytes.Contains(data, oldRaw) && !bytes.Contains(data, oldEscaped) {
		return data
	}
	c.files[path] = true
	newBody := []byte("func replacePathParam(path, name, value string) string {\n\tencoded := url.PathEscape(value)\n\tif value == \".\" || value == \"..\" {\n\t\tencoded = strings.Repeat(\"%2E\", len(value))\n\t}\n\treturn strings.ReplaceAll(path, \"{\"+name+\"}\", encoded)\n}")
	if bytes.Contains(data, oldRaw) {
		data = bytes.Replace(data, oldRaw, newBody, 1)
		if !bytes.Contains(data, []byte("\"net/url\"")) {
			data = bytes.Replace(data, []byte("import (\n"), []byte("import (\n\t\"net/url\"\n"), 1)
		}
		return data
	}
	return bytes.Replace(data, oldEscaped, newBody, 1)
}

func retrofitTrueParamGuards(data []byte, c *cluster, path string) []byte {
	if !trueParamGuard.Match(data) {
		return data
	}
	updated := trueParamGuard.ReplaceAll(data, []byte("$1$2"))
	for {
		compacted := splitParamAssignments.ReplaceAll(updated, []byte("$1$2"))
		if bytes.Equal(compacted, updated) {
			break
		}
		updated = compacted
	}
	if !bytes.Equal(updated, data) {
		c.files[path] = true
	}
	return updated
}

func retrofitRollbackCount(data []byte, c *cluster, path string) []byte {
	staleComment := []byte("\t\t\t// Return the running stored count rather than zero so callers\n\t\t\t// inspecting partial progress on failure see what already\n\t\t\t// landed in earlier loop iterations.\n")
	fixedComment := []byte("\t\t\t// A non-nil error aborts this transaction through the deferred\n\t\t\t// rollback, so no earlier in-memory progress was committed.\n")
	if bytes.Contains(data, staleComment) {
		c.files[path] = true
		data = bytes.Replace(data, staleComment, fixedComment, 1)
	}
	if bytes.Contains(data, []byte("func (s *Store) upsertBatchTx(")) {
		old := data
		data = bytes.ReplaceAll(data, []byte("return stored, extractFailures, fmt.Errorf("), []byte("return 0, extractFailures, fmt.Errorf("))
		data = bytes.Replace(data, []byte("return stored, extractFailures, err\n\t}"), []byte("return 0, extractFailures, err\n\t}"), 1)
		data = bytes.Replace(data, []byte("return stored, extractFailures, 0, err\n\t}"), []byte("return 0, extractFailures, 0, err\n\t}"), 1)
		data = bytes.Replace(data, []byte("return stored, extractFailures, 0, fmt.Errorf("), []byte("return 0, extractFailures, 0, fmt.Errorf("), 1)
		data = bytes.Replace(data, []byte("return stored, extractFailures, totalCount, fmt.Errorf("), []byte("return 0, extractFailures, 0, fmt.Errorf("), 1)
		if !bytes.Equal(data, old) {
			c.files[path] = true
		}
		return data
	}
	if bytes.Contains(data, []byte("func (s *Store) upsertBatch(")) && bytes.Contains(data, []byte("return stored, extractFailures, fmt.Errorf(")) {
		c.files[path] = true
		return bytes.ReplaceAll(data, []byte("return stored, extractFailures, fmt.Errorf("), []byte("return 0, extractFailures, fmt.Errorf("))
	}
	start := bytes.Index(data, []byte("func (s *Store) UpsertBatchDetailed("))
	if start < 0 {
		start = bytes.Index(data, []byte("func (s *Store) UpsertBatch("))
		if start < 0 {
			return data
		}
	}
	endRel := bytes.Index(data[start:], []byte("\nfunc "))
	if endRel < 0 {
		return data
	}
	end := start + endRel
	body := data[start:end]
	oldDetailed := []byte("return stored, extractFailures, typedFailures, fmt.Errorf(")
	oldSimple := []byte("return stored, extractFailures, fmt.Errorf(")
	if !bytes.Contains(body, oldDetailed) && !bytes.Contains(body, oldSimple) {
		return data
	}
	c.files[path] = true
	body = bytes.ReplaceAll(body, oldDetailed, []byte("return 0, extractFailures, typedFailures, fmt.Errorf("))
	body = bytes.ReplaceAll(body, oldSimple, []byte("return 0, extractFailures, fmt.Errorf("))
	return append(append(append([]byte{}, data[:start]...), body...), data[end:]...)
}

func writePatchRecords(c *cluster, paths []string) error {
	byRoot := map[string][]string{}
	for _, path := range paths {
		marker := "/internal/"
		idx := strings.Index(path, marker)
		if idx < 0 {
			continue
		}
		cliRoot := path[:idx]
		byRoot[cliRoot] = append(byRoot[cliRoot], strings.TrimPrefix(path, cliRoot+"/"))
	}
	for cliRoot, files := range byRoot {
		var manifest map[string]any
		if data, err := os.ReadFile(filepath.Join(cliRoot, ".printing-press.json")); err == nil {
			_ = json.Unmarshal(data, &manifest)
		}
		record := patchRecord{SchemaVersion: 2, ID: c.id, AppliedAt: "2026-09-10", Summary: c.summary, Reason: c.reason, Files: files, ValidatedOutcome: c.outcome}
		if v, ok := manifest["run_id"].(string); ok {
			record.BaseRunID = v
		}
		if v, ok := manifest["printing_press_version"].(string); ok {
			record.BasePrintingPressVersion = v
		}
		sort.Strings(record.Files)
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		patchDir := filepath.Join(cliRoot, ".printing-press-patches")
		if err := os.MkdirAll(patchDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(patchDir, c.id+".json"), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
