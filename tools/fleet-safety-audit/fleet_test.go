package main

import (
	"bytes"
	"encoding/json"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Opt-in because a checkout's review base is not a stable unit-test input.
// This independently replays every mechanical source change from its current
// upstream blob and verifies patch-record coverage of the actual Git diff.
func TestFleetDiffAgainstBase(t *testing.T) {
	base := os.Getenv("FLEET_AUDIT_BASE")
	if base == "" {
		t.Skip("set FLEET_AUDIT_BASE to audit the full candidate diff")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) []byte {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return out
	}
	files := strings.Fields(string(git("diff", "--name-only", base, "--", "library")))
	tracked := map[string]bool{}
	for _, path := range strings.Fields(string(git("ls-files", "--", "library"))) {
		tracked[path] = true
	}
	checked := 0
	for _, path := range files {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		actual, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, actual, parser.AllErrors); err != nil {
			t.Fatalf("invalid Go %s: %v", path, err)
		}
		cliRoot, _, ok := strings.Cut(path, "/internal/")
		if !ok {
			t.Fatalf("unexpected source path %s", path)
		}
		records, err := filepath.Glob(filepath.Join(root, cliRoot, ".printing-press-patches", "*.json"))
		if err != nil {
			t.Fatal(err)
		}
		covered := false
		for _, record := range records {
			rel, err := filepath.Rel(root, record)
			if err != nil {
				t.Fatal(err)
			}
			if !tracked[rel] {
				continue
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			var patch patchRecord
			if err := json.Unmarshal(data, &patch); err != nil {
				t.Fatalf("%s: %v", record, err)
			}
			for _, file := range patch.Files {
				if cliRoot+"/"+file == path {
					covered = true
				}
			}
		}
		if !covered {
			t.Errorf("no tracked reprint guard covers %s (stage new records, including ignored paths)", path)
		}
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		// This bespoke cross-source boundary is covered by executable tests.
		if path == "library/media-and-entertainment/soccer-goat/internal/client/client.go" {
			continue
		}
		original := git("show", base+":"+path)
		c := &cluster{files: map[string]bool{}}
		var expected []byte
		switch {
		case strings.HasSuffix(path, "/internal/client/client.go"):
			expected = retrofitRetry(original, c, path)
		case strings.HasSuffix(path, "/internal/cli/helpers.go"):
			expected = retrofitPathEncoding(original, c, path)
		case strings.HasSuffix(path, "/internal/store/store.go"):
			expected = retrofitRollbackCount(original, c, path)
		default:
			expected = retrofitTrueParamGuards(original, c, path)
		}
		if bytes.Equal(expected, original) {
			t.Errorf("unnecessary source change: %s", path)
			continue
		}
		expected, err = format.Source(expected)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("source differs from current-base mechanical replay: %s", path)
		}
		checked++
	}
	t.Logf("checked %d mechanical source files against %s", checked, base)
}
