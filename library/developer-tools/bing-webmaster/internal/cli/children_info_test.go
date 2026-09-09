package cli

import (
	"bytes"
	"encoding/json"
	"github.com/mvanhorn/printing-press-library/library/developer-tools/bing-webmaster/internal/cliutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChildrenInfoUsesReadOnlyPost(t *testing.T) {
	restore, err := cliutil.SetHomeOverride("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(restore)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/json/GetChildrenUrlInfo" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["siteUrl"] != "https://example.org/" || body["url"] != "example.org" || body["page"] != float64(0) || body["filterProperties"] == nil {
			t.Errorf("wrong wrapped body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":[]}`))
	}))
	defer server.Close()
	t.Setenv("BING_WEBMASTER_BASE_URL", server.URL)
	t.Setenv("BING_WEBMASTER_API_KEY", "test-placeholder")
	t.Setenv("BING_WEBMASTER_TEST_SITE", "")
	for _, dry := range []bool{false, true} {
		root := RootCmd()
		var output bytes.Buffer
		root.SetOut(&output)
		root.SetErr(&output)
		args := []string{"crawl", "children-info", "--site", "https://example.org/", "--url", "example.org", "--json", "--home", t.TempDir(), "--no-cache"}
		if dry {
			args = append(args, "--dry-run")
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("expected one POST and no dry-run HTTP, got %d", calls)
	}
}
