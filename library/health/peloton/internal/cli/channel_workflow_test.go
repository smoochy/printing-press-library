// Copyright 2026 Felix Banuchi and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/health/peloton/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/health/peloton/internal/store"
)

// TestWorkflowArchiveActuallyArchivesData guards a round-9 live verification
// finding worse than the report described: newWorkflowArchiveCmd's resource
// list was hardcoded to an empty slice (`resources := []string{}`), so every
// invocation — with or without --full, and regardless of account size —
// silently reported "0 synced" and archived nothing at all. It's also
// reachable as an MCP tool, so an agent calling it saw a clean success
// response for a command that never did anything. Fixed by delegating to a
// real `sync` command instance (`--resources workouts,classes`, which
// cascades to performance/workout_details automatically) rather than a
// second hand-rolled sync loop, so this command inherits sync's actual
// resource list, dependent fan-out, and bounding flags instead of
// duplicating them. This test drives the real `workflow archive` command
// end-to-end against a fake server and asserts the local store actually
// ends up populated, across all four resources sync cascades to.
func TestWorkflowArchiveActuallyArchivesData(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/ride/archived"):
			_, _ = w.Write([]byte(`[{"id":"class-1","title":"Test Ride","duration":1800}]`))
		case strings.Contains(r.URL.Path, "/performance_graph"):
			_, _ = w.Write([]byte(`{"summaries":[]}`))
		case strings.HasSuffix(r.URL.Path, "/workouts"):
			_, _ = w.Write([]byte(`[{"id":"workout-1","ride_id":"class-1","start_time":"2026-01-01T10:00:00Z"}]`))
		case strings.Contains(r.URL.Path, "/workout/"):
			_, _ = w.Write([]byte(`{"id":"workout-1","ride_id":"class-1"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "archive", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow archive: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	// The bug: this used to always be exactly the string
	// {"resources_synced":0,"total_items":0,...} with no way to make it
	// synced anything, on any account. Assert the delegated sync's own
	// event-stream shape instead, and that it actually reports work done.
	if !strings.Contains(out.String(), `"event":"sync_summary"`) {
		t.Fatalf("expected sync's event stream (workflow archive should delegate to sync), got: %s", out.String())
	}
	if strings.Contains(out.String(), `"resources_synced":0`) {
		t.Fatalf("workflow archive still reports the old dead-command shape: %s", out.String())
	}

	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, resource := range []string{"workouts", "classes", "performance", "workout_details"} {
		count, err := db.Count(resource)
		if err != nil {
			t.Fatalf("db.Count(%s): %v", resource, err)
		}
		if count == 0 {
			t.Fatalf("workflow archive did not populate %q at all — resources cascade broken", resource)
		}
	}
}

// TestWorkflowArchiveBoundingFlagsAccepted guards that --max-pages,
// --max-parents, and --stale-before (added so workflow_archive can inherit
// the same bounded, resumable sync `sync` already has) are real, wired
// flags — not just present in --help with no effect. A cobra flag-parse
// failure here would mean a flag was declared but never threaded into the
// delegated sync args.
func TestWorkflowArchiveBoundingFlagsAccepted(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "archive", "--db", dbPath, "--home", home, "--json", "--max-pages", "3", "--max-parents", "10", "--stale-before", "24h"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow archive --max-pages --max-parents --stale-before: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}
}

// TestWorkflowArchiveDefaultsBoundedUnderMCPShelloutEnv guards a round-11
// live verification finding: --max-pages/--max-parents existing as flags
// (TestWorkflowArchiveMaxPagesBoundsFlatPhase) isn't enough by itself -- an
// MCP agent that calls this tool with no arguments (the report's exact
// repro: `workflow_archive --max-parents 25 --json`, still omitting
// --max-pages) still hangs on the flat classes phase's unbounded default.
// When PELOTON_MCP_SHELLOUT is set (as the real MCP server sets it on
// itself, inherited by every companion CLI subprocess it spawns -- see
// cliutil.MCPShelloutEnvVar), an unset --max-pages must default to a real
// bound instead of staying unlimited, using the same never-terminating
// classes-list fixture TestWorkflowArchiveMaxPagesBoundsFlatPhase drives.
func TestWorkflowArchiveDefaultsBoundedUnderMCPShelloutEnv(t *testing.T) {
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
		if strings.Contains(r.URL.Path, "/ride/archived") {
			_, _ = w.Write(fullPage)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	t.Setenv(cliutil.MCPShelloutEnvVar, "1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	// Deliberately no --max-pages/--max-parents: this is the exact call
	// shape an MCP agent makes when it doesn't know about those flags.
	root.SetArgs([]string{"workflow", "archive", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow archive under MCPShelloutEnvVar: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), `"reason":"max_pages_cap_hit"`) {
		t.Fatalf("expected the flat classes phase to hit a default page bound with no --max-pages passed, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "reached --max-pages cap of 5") {
		t.Fatalf("expected the MCP-shellout default of 5 pages, got: %s", out.String())
	}
}

// TestWorkflowArchiveMaxPagesBoundsFlatPhase guards a round-10 live
// verification finding: --max-parents bounds the per-workout dependent
// fan-out (performance/workout_details), but nothing bounded the flat
// classes/workouts phase that runs before it. classes in particular
// declares no incremental filter, so a full pass paginates the account's
// entire archived-class history every single call — confirmed live at
// 48,742 items taking ~4.5 minutes, well past what an MCP tool call's
// timeout tolerates, so no --max-parents value could make a real account's
// archive call complete over MCP. This drives a fake classes list endpoint
// that would paginate across 3 full pages (100 items each, the resource's
// hardcoded page size) and asserts --max-pages 1 stops after page one with
// a max_pages_cap_hit warning, rather than fetching everything.
func TestWorkflowArchiveMaxPagesBoundsFlatPhase(t *testing.T) {
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
		if strings.Contains(r.URL.Path, "/ride/archived") {
			_, _ = w.Write(fullPage)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "archive", "--db", dbPath, "--home", home, "--json", "--max-pages", "1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow archive --max-pages 1: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "max_pages_cap_hit") {
		t.Fatalf("expected a max_pages_cap_hit warning bounding the classes phase at 1 page, got: %s", out.String())
	}

	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	count, err := db.Count("classes")
	if err != nil {
		t.Fatal(err)
	}
	if count != 100 {
		t.Fatalf("classes count = %d, want exactly 100 (one page) — --max-pages did not bound the flat phase", count)
	}
}

// TestWorkflowStatusJSONReturnsStructuredEnvelopeWhenEmpty guards a
// live-tested bug: workflow_status(json=true)/workflow_status(agent=true) on
// an empty store returned a bare "{}", giving the modes agents actually use
// (structured output) no explanation at all, while plain-text mode correctly
// said "No archived data. Run 'workflow archive' to sync." -- search already
// returns {"count":0,"store_status":"empty",...} for the same situation, so
// workflow status must match that shape instead of returning nothing
// actionable to the caller that most needs a machine-readable answer.
func TestWorkflowStatusJSONReturnsStructuredEnvelopeWhenEmpty(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	dbPath := filepath.Join(home, "data", "data.db")
	db, err := store.OpenWithContext(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "status", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow status --json: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	if strings.TrimSpace(out.String()) == "{}" {
		t.Fatalf("workflow status --json on an empty store still returns a bare {}: %s", out.String())
	}

	var envelope struct {
		StoreStatus string         `json:"store_status"`
		Resources   map[string]any `json:"resources"`
		Total       int            `json:"total"`
		NextStep    string         `json:"next_step"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("workflow status --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if envelope.StoreStatus != "empty" {
		t.Fatalf("store_status = %q, want %q", envelope.StoreStatus, "empty")
	}
	if len(envelope.Resources) != 0 {
		t.Fatalf("resources = %#v, want empty on a never-synced store", envelope.Resources)
	}
	if envelope.NextStep == "" {
		t.Fatal("next_step is empty; an agent reading this JSON has no path forward")
	}
}

// TestWorkflowStatusJSONIncludesCountsAndLastSyncedAt guards the rest of the
// same fix: after a real archive, structured mode must report store_status
// "ready", per-resource counts, and a last_synced_at timestamp for each
// resource actually synced -- not just fix the empty case.
func TestWorkflowStatusJSONIncludesCountsAndLastSyncedAt(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/ride/archived"):
			_, _ = w.Write([]byte(`[{"id":"class-1","title":"Test Ride","duration":1800}]`))
		case strings.Contains(r.URL.Path, "/performance_graph"):
			_, _ = w.Write([]byte(`{"summaries":[]}`))
		case strings.HasSuffix(r.URL.Path, "/workouts"):
			_, _ = w.Write([]byte(`[{"id":"workout-1","ride_id":"class-1","start_time":"2026-01-01T10:00:00Z"}]`))
		case strings.Contains(r.URL.Path, "/workout/"):
			_, _ = w.Write([]byte(`{"id":"workout-1","ride_id":"class-1"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var archiveOut, archiveErr bytes.Buffer
	root.SetOut(&archiveOut)
	root.SetErr(&archiveErr)
	root.SetArgs([]string{"workflow", "archive", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow archive: %v\nstdout: %s\nstderr: %s", err, archiveOut.String(), archiveErr.String())
	}

	root = newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "status", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow status --json: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	var envelope struct {
		StoreStatus string `json:"store_status"`
		Resources   map[string]struct {
			Count        int    `json:"count"`
			LastSyncedAt string `json:"last_synced_at"`
		} `json:"resources"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("workflow status --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if envelope.StoreStatus != "ready" {
		t.Fatalf("store_status = %q, want %q after a real archive", envelope.StoreStatus, "ready")
	}
	workouts, ok := envelope.Resources["workouts"]
	if !ok {
		t.Fatalf("resources missing \"workouts\" entry: %#v", envelope.Resources)
	}
	if workouts.Count == 0 {
		t.Fatal("workouts count = 0, want at least one archived workout")
	}
	if workouts.LastSyncedAt == "" {
		t.Fatal("workouts last_synced_at is empty after a real archive")
	}
	if envelope.Total == 0 {
		t.Fatal("total = 0, want the sum of all resource counts")
	}

	// performance is a parent-keyed dependent (one request per already-
	// synced workout, no bulk endpoint): in the default (non---full) sync
	// mode it never gets a plain sync_state row, only resources.synced_at
	// from its own upserts. Guards the review finding that dependent
	// resources' last_synced_at was silently omitted from this envelope.
	performance, ok := envelope.Resources["performance"]
	if !ok {
		t.Fatalf("resources missing \"performance\" entry: %#v", envelope.Resources)
	}
	if performance.LastSyncedAt == "" {
		t.Fatal("performance last_synced_at is empty after a real archive -- dependent-resource timestamps must fall back to resources.synced_at")
	}
}

// TestWorkflowStatusJSONDoesNotMisclassifyAZeroResultSyncAsEmpty guards a
// review finding: Status()'s row-count view can't distinguish "never
// synced" from "a sync completed but the account genuinely had zero items"
// -- both look like zero rows in the resources table. A flat resource sync
// still writes a sync_state row even when it fetches nothing, so
// HasSyncHistory must be consulted alongside Status() before reporting
// store_status "empty", or a real completed sync gets told to archive again.
func TestWorkflowStatusJSONDoesNotMisclassifyAZeroResultSyncAsEmpty(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	t.Setenv("PELOTON_BASE_URL", server.URL)
	t.Setenv("PELOTON_USER_ID", "u1")
	seedValidOAuthBundleForLiveFetchTests(t)

	dbPath := filepath.Join(home, "data", "data.db")
	root := newRootCmd(&rootFlags{})
	var syncOut, syncErr bytes.Buffer
	root.SetOut(&syncOut)
	root.SetErr(&syncErr)
	root.SetArgs([]string{"sync", "--resources", "workouts", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync --resources workouts: %v\nstdout: %s\nstderr: %s", err, syncOut.String(), syncErr.String())
	}

	root = newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "status", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow status --json: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	var envelope struct {
		StoreStatus string         `json:"store_status"`
		Resources   map[string]any `json:"resources"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("workflow status --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if envelope.StoreStatus != "ready" {
		t.Fatalf("store_status = %q, want %q -- a completed zero-result sync is not the same as never having synced", envelope.StoreStatus, "ready")
	}
}

// TestWorkflowStatusJSONReportsEmptyForInterruptedSyncNotJustReset guards a
// review finding on the fix above: --full writes a sync_state row (cursor
// "", count 0) for every named resource BEFORE fetching starts, and the
// flat sync loop checkpoints sync_state after every page, well before a
// resource is actually done -- both are bit-for-bit identical to a genuine
// zero-item completion if HasSyncHistory trusted "any sync_state row
// exists." An account interrupted right after that reset/checkpoint (crash,
// kill -9, container restart) would then have sync_state populated but the
// resources table still empty, and workflow status must not call that
// "ready" -- it has to say "empty" and point back at archiving, the same
// as a store that was never touched at all. This drives SaveSyncState (not
// SaveSyncStateCompleted) directly, matching exactly what --full's reset
// and the mid-pagination checkpoint call, without running a real sync.
func TestWorkflowStatusJSONReportsEmptyForInterruptedSyncNotJustReset(t *testing.T) {
	home := t.TempDir()
	restore, err := cliutil.SetHomeOverride(home)
	if err != nil {
		t.Fatal(err)
	}
	defer restore()

	dbPath := filepath.Join(home, "data", "data.db")
	db, err := store.OpenWithContext(t.Context(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSyncState("workouts", "", 0); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd(&rootFlags{})
	var out, stderr bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&stderr)
	root.SetArgs([]string{"workflow", "status", "--db", dbPath, "--home", home, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("workflow status --json: %v\nstdout: %s\nstderr: %s", err, out.String(), stderr.String())
	}

	var envelope struct {
		StoreStatus string `json:"store_status"`
		NextStep    string `json:"next_step"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("workflow status --json output is not valid JSON: %v\n%s", err, out.String())
	}
	if envelope.StoreStatus != "empty" {
		t.Fatalf("store_status = %q, want %q -- an in-progress/reset sync_state row with no archived resources must not read as a completed sync", envelope.StoreStatus, "empty")
	}
	if envelope.NextStep == "" {
		t.Fatal("next_step is empty; an agent reading this JSON has no path forward")
	}
}
