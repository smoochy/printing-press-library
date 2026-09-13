// Copyright 2026 Felix Banuchi and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/health/peloton/internal/cliutil"
)

func intPtr(v int) *int { return &v }

// TestSyncCompleteEventJSONFieldsAndOmitEmpty guards the pure event-shape
// contract syncCompleteEventJSON exists to provide: "total" (this call's
// count) is unchanged from the prior raw fmt.Sprintf shape so an existing
// consumer sees no behavior change, "store_total" is present when known,
// and "resume_cursor" is present only when non-empty (a natural-completion
// call has nothing to resume, so the field should disappear rather than
// serialize as an empty string every time).
func TestSyncCompleteEventJSONFieldsAndOmitEmpty(t *testing.T) {
	naturalCompletion := syncCompleteEventJSON("classes", 100, intPtr(500), "", 1234)
	var obj map[string]any
	if err := json.Unmarshal([]byte(naturalCompletion), &obj); err != nil {
		t.Fatalf("event is not valid JSON: %v\n%s", err, naturalCompletion)
	}
	if obj["event"] != "sync_complete" || obj["resource"] != "classes" {
		t.Fatalf("event/resource = %#v/%#v, want sync_complete/classes", obj["event"], obj["resource"])
	}
	if total, _ := obj["total"].(float64); total != 100 {
		t.Fatalf("total = %v, want 100", obj["total"])
	}
	if storeTotal, _ := obj["store_total"].(float64); storeTotal != 500 {
		t.Fatalf("store_total = %v, want 500", obj["store_total"])
	}
	if _, ok := obj["resume_cursor"]; ok {
		t.Fatalf("resume_cursor present on a natural-completion event, want omitted: %s", naturalCompletion)
	}

	capped := syncCompleteEventJSON("classes", 100, intPtr(100), "opaque-cursor-value", 1234)
	if !strings.Contains(capped, `"resume_cursor":"opaque-cursor-value"`) {
		t.Fatalf("resume_cursor missing or wrong on a capped completion event: %s", capped)
	}
}

// TestSyncCompleteEventJSONOmitsStoreTotalWhenNil guards a review finding:
// a failed db.Count(resource) (e.g. another sync worker briefly locking the
// store) used to be discarded and reported as store_total 0 -- reading as
// "the store just lost everything" to a caller comparing store_total across
// calls, a worse outcome than not having the cumulative count at all. nil
// must omit the field, not serialize a false zero.
func TestSyncCompleteEventJSONOmitsStoreTotalWhenNil(t *testing.T) {
	event := syncCompleteEventJSON("classes", 100, nil, "", 1234)
	var obj map[string]any
	if err := json.Unmarshal([]byte(event), &obj); err != nil {
		t.Fatalf("event is not valid JSON: %v\n%s", err, event)
	}
	if _, ok := obj["store_total"]; ok {
		t.Fatalf("store_total present with a nil pointer, want omitted so it can't be mistaken for a real zero: %s", event)
	}
}

// TestSyncCompleteReportsStoreTotalDistinctFromPerCallTotal guards the
// live-tested bug this exists to fix: two consecutive resumed sync calls
// against a large account both logged the same {"total":500}, which reads
// as a stuck cursor even though resumption is genuinely advancing (each
// call fetched a different slice, confirmed by distinct row IDs). This
// drives a single real `sync --resources classes` call against a small
// fixture (so pagination completes naturally in one page) and asserts
// sync_complete's store_total reflects the resource's true row count in the
// local store, with no resume_cursor (nothing to resume after natural
// completion).
func TestSyncCompleteReportsStoreTotalOnNaturalCompletion(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"class-1","title":"Test Ride","duration":1800}]`))
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--resources", "classes", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --resources classes: %v\noutput: %s", err, out.String())
	}

	line := findEventLine(t, out.String(), "sync_complete", "classes")
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("sync_complete line is not valid JSON: %v\n%s", err, line)
	}
	if storeTotal, _ := event["store_total"].(float64); storeTotal != 1 {
		t.Fatalf("store_total = %v, want 1 (one class row in the store): %s", event["store_total"], line)
	}
	if _, ok := event["resume_cursor"]; ok {
		t.Fatalf("resume_cursor present after natural completion, want omitted: %s", line)
	}
}

// TestSyncCompleteReportsResumeCursorWhenCapped guards the other half of the
// same fix: a --max-pages-capped run leaves a real resume position, and
// sync_complete must surface it so a caller can tell "capped, more to
// fetch" from "done" without cross-referencing a separate sync_warning
// event. Reuses the same never-terminating classes-list fixture
// TestWorkflowArchiveMaxPagesBoundsFlatPhase drives.
func TestSyncCompleteReportsResumeCursorWhenCapped(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	fullPage := make([]byte, 0, 4096)
	fullPage = append(fullPage, '[')
	for i := 0; i < 100; i++ {
		if i > 0 {
			fullPage = append(fullPage, ',')
		}
		fullPage = append(fullPage, []byte(`{"id":"class-`+strconv.Itoa(i)+`","title":"Ride","duration":1800}`)...)
	}
	fullPage = append(fullPage, ']')

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fullPage)
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--resources", "classes", "--db", dbPath, "--home", home, "--json", "--max-pages", "1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --resources classes --max-pages 1: %v\noutput: %s", err, out.String())
	}

	line := findEventLine(t, out.String(), "sync_complete", "classes")
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("sync_complete line is not valid JSON: %v\n%s", err, line)
	}
	if storeTotal, _ := event["store_total"].(float64); storeTotal != 100 {
		t.Fatalf("store_total = %v, want 100: %s", event["store_total"], line)
	}
	resumeCursor, _ := event["resume_cursor"].(string)
	if resumeCursor == "" {
		t.Fatalf("resume_cursor missing or empty after a --max-pages cap hit: %s", line)
	}
}

// findEventLine returns the single NDJSON line in output whose "event" and
// "resource" fields match, failing the test if there isn't exactly one.
func findEventLine(t *testing.T, output, event, resource string) string {
	t.Helper()
	var match string
	var count int
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) != nil {
			continue
		}
		if obj["event"] == event && obj["resource"] == resource {
			match = line
			count++
		}
	}
	if count != 1 {
		t.Fatalf("found %d %q event line(s) for resource %q, want exactly 1:\n%s", count, event, resource, output)
	}
	return match
}
