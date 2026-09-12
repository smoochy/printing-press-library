package main

import (
	"bytes"
	"testing"
)

func TestRetryRetrofitDisablesAmbiguousWriteReplay(t *testing.T) {
	in := []byte(retryFixture)
	c := &cluster{files: map[string]bool{}}
	got := retrofitRetry(in, c, "library/example/internal/client/client.go")
	if bytes.Contains(got, []byte("readOnlyIntent")) || !bytes.Contains(got, []byte("canRetryAmbiguousFailure := method")) || bytes.Contains(got, []byte("maxRetries = 0")) {
		t.Fatalf("retry retrofit missing safety gate:\n%s", got)
	}
	for _, want := range []string{"if !canRetryAmbiguousFailure", "resp.StatusCode >= 500 && attempt < maxRetries && canRetryAmbiguousFailure", "resp.StatusCode == 401 && attempt < maxRetries", "resp.StatusCode == 429 && attempt < maxRetries"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
	if again := retrofitRetry(got, &cluster{files: map[string]bool{}}, "example"); !bytes.Equal(again, got) {
		t.Fatal("retry retrofit is not idempotent")
	}
}

func TestRetryRetrofitPreservesExplicitReadOnlyIntent(t *testing.T) {
	in := bytes.Replace([]byte(retryFixture), []byte("do(method string)"), []byte("do(method string, readOnlyIntent bool)"), 1)
	got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "library/example/internal/client/client.go")
	if !bytes.Contains(got, []byte("canRetryAmbiguousFailure := readOnlyIntent || method")) {
		t.Fatalf("retry retrofit dropped explicit read-only intent:\n%s", got)
	}
}

const retryFixture = `package client
func (c *Client) do(method string) ([]byte, int, error) {
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 401 && attempt < maxRetries { continue }
		if resp.StatusCode == 429 && attempt < maxRetries { continue }
		if resp.StatusCode >= 500 && attempt < maxRetries { continue }
		return nil, resp.StatusCode, nil
	}
	return nil, 0, lastErr
}
`

func TestRetryRetrofitDoesNotTrustAnUnreviewedHelperName(t *testing.T) {
	in := bytes.Replace([]byte(retryFixture), []byte("lastErr = err"), []byte("lastErr = err\nif !requestCanRetry(method, false) { return nil, 0, lastErr }"), 1)
	in = bytes.Replace(in, []byte("resp.StatusCode >= 500 && attempt < maxRetries"), []byte("resp.StatusCode >= 500 && attempt < maxRetries && requestCanRetry(method, false)"), 1)
	got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "client.go")
	if !bytes.Contains(got, []byte("&& canRetryAmbiguousFailure")) || !bytes.Contains(got, []byte("if !canRetryAmbiguousFailure {")) {
		t.Fatalf("unreviewed helper bypassed conservative guards:\n%s", got)
	}
}

func TestRetryRetrofitRepairsPartiallyGuardedCustomPredicates(t *testing.T) {
	for _, predicate := range []string{"!nonIdempotent", "method != \"POST\" && method != \"PATCH\"", "retryOnServerError(method)", "isIdempotentMethod(method)"} {
		t.Run(predicate, func(t *testing.T) {
			in := bytes.Replace([]byte(retryFixture), []byte("const maxRetries = 3"), []byte("canRetryAmbiguousFailure := method == http.MethodGet\nconst maxRetries = 3"), 1)
			in = bytes.Replace(in, []byte("lastErr = err"), []byte("lastErr = err\nif !canRetryAmbiguousFailure { return nil, 0, lastErr }"), 1)
			in = bytes.Replace(in, []byte("resp.StatusCode >= 500 && attempt < maxRetries"), []byte("resp.StatusCode >= 500 && attempt < maxRetries && "+predicate), 1)
			got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "client.go")
			if !bytes.Contains(got, []byte(predicate+" && canRetryAmbiguousFailure")) {
				t.Fatalf("missed partial 5xx policy:\n%s", got)
			}
			if bytes.Count(got, []byte("canRetryAmbiguousFailure :=")) != 1 {
				t.Fatal("duplicated local guard")
			}
			if again := retrofitRetry(got, &cluster{files: map[string]bool{}}, "client.go"); !bytes.Equal(again, got) {
				t.Fatal("not idempotent")
			}
		})
	}
}

func TestRetryRetrofitDoesNotSkipAnotherFunctionInGuardedFile(t *testing.T) {
	first := retrofitRetry([]byte(retryFixture), &cluster{files: map[string]bool{}}, "client.go")
	second := bytes.Replace([]byte(retryFixture), []byte("package client\n"), nil, 1)
	second = bytes.Replace(second, []byte("do(method"), []byte("doOther(method"), 1)
	got := retrofitRetry(append(first, second...), &cluster{files: map[string]bool{}}, "client.go")
	if bytes.Count(got, []byte("canRetryAmbiguousFailure :=")) != 2 {
		t.Fatal("file-wide marker hid second unsafe function")
	}
}

func TestRetryRetrofitDistinguishesReadOnlyGuardFromWriteMaintenance(t *testing.T) {
	for _, tc := range []struct {
		name, statement string
		wantChange      bool
	}{
		{"read-only", "if method != http.MethodGet { return nil, 0, errReadOnly }", false},
		{"cache-invalidation", "if method != http.MethodGet { c.invalidateCache() }", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := bytes.Replace([]byte(retryFixture), []byte("const maxRetries = 3"), []byte(tc.statement+"\nconst maxRetries = 3"), 1)
			got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "client.go")
			if changed := !bytes.Equal(got, in); changed != tc.wantChange {
				t.Fatalf("changed=%v, want %v", changed, tc.wantChange)
			}
		})
	}
}

func TestRetryRetrofitLeavesFixedMethodHelpersAlone(t *testing.T) {
	in := bytes.Replace([]byte(retryFixture), []byte("do(method string)"), []byte("GetJSON(path string)"), 1)
	got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "client.go")
	if !bytes.Equal(got, in) {
		t.Fatal("rewrote helper without a method parameter")
	}
}

func TestPathRetrofitPreservesOneEncodedSegment(t *testing.T) {
	in := []byte("package cli\n\nimport (\n\t\"strings\"\n)\n\nfunc replacePathParam(path, name, value string) string {\n\treturn strings.ReplaceAll(path, \"{\"+name+\"}\", value)\n}\n")
	c := &cluster{files: map[string]bool{}}
	got := retrofitPathEncoding(in, c, "library/example/internal/cli/helpers.go")
	for _, want := range [][]byte{[]byte("\"net/url\""), []byte("url.PathEscape(value)"), []byte("strings.Repeat(\"%2E\", len(value))")} {
		if !bytes.Contains(got, want) {
			t.Fatalf("path retrofit missing %q:\n%s", want, got)
		}
	}
}

func TestRollbackRetrofitReportsCommittedRows(t *testing.T) {
	in := []byte("func (s *Store) UpsertBatch(resourceType string, items []json.RawMessage) (int, int, error) {\n\treturn stored, extractFailures, fmt.Errorf(\"failed\")\n}\n\nfunc next() {}\n")
	c := &cluster{files: map[string]bool{}}
	got := retrofitRollbackCount(in, c, "library/example/internal/store/store.go")
	if bytes.Contains(got, []byte("return stored, extractFailures, fmt.Errorf(")) || !bytes.Contains(got, []byte("return 0, extractFailures, fmt.Errorf(")) {
		t.Fatalf("rollback retrofit kept an uncommitted count:\n%s", got)
	}
}

func TestDefaultedParameterGuardRetrofitIsCompact(t *testing.T) {
	in := []byte("\tif true {\n\t\tparams[\"limit\"] = formatCLIParamValue(limit)\n\t}\n\tif true {\n\t\tparams[\"offset\"] = formatCLIParamValue(offset)\n\t}\n")
	c := &cluster{files: map[string]bool{}}
	got := retrofitTrueParamGuards(in, c, "library/example/internal/cli/list.go")
	if bytes.Contains(got, []byte("if true")) || bytes.Contains(bytes.TrimSpace(got), []byte("\n\n")) {
		t.Fatalf("parameter retrofit left a constant guard or blank separator:\n%s", got)
	}
}
