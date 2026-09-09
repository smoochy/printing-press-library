package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishPlanReadOnlyAndQuota(t *testing.T) {
	for _, tc := range []struct {
		name, quota string
		want        int
		fail        bool
	}{{"normal", `{"d":{"DailyQuota":1}}`, 1, false}, {"zero", `{"d":{"DailyQuota":0}}`, 0, false}, {"missing", `{"d":{}}`, 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/json/GetUrlSubmissionQuota" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.quota))
			}))
			defer srv.Close()
			t.Setenv("BING_WEBMASTER_API_KEY", "test-placeholder")
			t.Setenv("BING_WEBMASTER_BASE_URL", srv.URL)
			t.Setenv("PRINTING_PRESS_VERIFY", "")
			file := filepath.Join(t.TempDir(), "urls.txt")
			if err := os.WriteFile(file, []byte("https://example.org/a\nhttps://example.org/b\n"), 0600); err != nil {
				t.Fatal(err)
			}
			flags := &rootFlags{asJSON: true}
			cmd := newPublishCommand(flags, true)
			if cmd.Flags().Lookup("confirm") != nil || cmd.Annotations["mcp:read-only"] != "true" {
				t.Fatal("unsafe plan surface")
			}
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{"--site", "https://example.org/", "--file", file})
			err := cmd.Execute()
			if tc.fail {
				if err == nil {
					t.Fatal("missing quota accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var plan bPublishPlan
			if err = json.Unmarshal(out.Bytes(), &plan); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || plan.WouldSubmit != tc.want || plan.Submitted != 0 || plan.Confirmed {
				t.Fatalf("unsafe result %+v calls=%d", plan, calls)
			}
		})
	}
}

func TestPublishPlanRejectsConfirm(t *testing.T) {
	cmd := newPublishCmd(&rootFlags{})
	cmd.SetArgs([]string{"plan", "--confirm"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("plan accepted confirmation")
	}
	if cmd.Annotations["mcp:read-only"] == "true" {
		t.Fatal("mutating parent marked read-only")
	}
}
