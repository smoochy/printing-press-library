package mcp

import (
	"context"
	"encoding/json"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChildrenInfoReadPost(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/json/GetChildrenUrlInfo" {
			t.Errorf("wrong request: %s %s", r.Method, r.URL.Path)
		}
		var b map[string]any
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Error(err)
		}
		if b["siteUrl"] != "https://example.org/" || b["page"] != float64(0) || b["filterProperties"] == nil {
			t.Errorf("wrong body: %#v", b)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":[]}`))
	}))
	defer srv.Close()
	t.Setenv("BING_WEBMASTER_HOME", t.TempDir())
	t.Setenv("BING_WEBMASTER_API_KEY", "test-placeholder")
	t.Setenv("BING_WEBMASTER_BASE_URL", srv.URL)
	result, err := handleCodeOrchExecute(context.Background(), mcplib.CallToolRequest{Params: mcplib.CallToolParams{Arguments: map[string]any{"endpoint_id": "crawl.children-info", "params": map[string]any{"site": "https://example.org/", "url": "example.org"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("MCP returned error: %+v", result)
	}
	if calls != 1 {
		t.Fatalf("expected one read POST, got %d", calls)
	}
}
