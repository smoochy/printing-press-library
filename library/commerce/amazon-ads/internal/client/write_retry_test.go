package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/commerce/amazon-ads/internal/config"
)

func TestWriteRefreshesAndRetriesAfterUnauthorized(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "1")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "1")
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			var apiCalls, tokenCalls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/token" {
					tokenCalls++
					_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "recovered-access", "expires_in": 3600})
					return
				}
				apiCalls++
				if r.Method != method {
					t.Errorf("method = %s, want %s", r.Method, method)
				}
				if r.Header.Get("Authorization") != "Bearer recovered-access" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_, _ = io.WriteString(w, `{"ok":true}`)
			}))
			defer srv.Close()
			cfg := &config.Config{BaseURL: srv.URL, TokenURL: srv.URL + "/token", Path: filepath.Join(t.TempDir(), "config.toml"), ClientID: "client-id", ClientSecret: "client-secret", AccessToken: "stale-access", RefreshToken: "stable-refresh"}
			c := New(cfg, time.Second, 0)
			c.cacheDir = t.TempDir()
			_, status, err := c.doInternal(context.Background(), method, "/resource", nil, nil, nil, false)
			if err != nil || status != 200 || apiCalls != 2 || tokenCalls != 1 {
				t.Fatalf("status=%d err=%v api=%d token=%d", status, err, apiCalls, tokenCalls)
			}
		})
	}
}

type retryTestTransport func(*http.Request) (*http.Response, error)

func (f retryTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWriteRetryFailureCategories(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "1")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "1")
	for _, tc := range []struct {
		name                     string
		status                   int
		transportError, readOnly bool
		wantCalls                int
		wantError                bool
	}{
		{"transport", 0, true, false, 1, true},
		{"server", 503, false, false, 1, true},
		{"rate_limit", 429, false, false, 2, false},
		{"rate_limit_exhausted", 429, false, false, 4, true},
		{"read_only_post", 503, false, true, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(&config.Config{BaseURL: "http://fixture.invalid", AccessToken: "test-access"}, time.Second, 0)
			c.cacheDir = t.TempDir()
			calls := 0
			c.HTTPClient.Transport = retryTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.transportError {
					return nil, errors.New("synthetic connection reset after write")
				}
				status := 200
				if calls == 1 || tc.name == "rate_limit_exhausted" {
					status = tc.status
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}, "Retry-After": []string{"1"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: r}, nil
			})
			_, _, err := c.doInternal(context.Background(), "POST", "/resource", nil, nil, nil, tc.readOnly)
			if (err != nil) != tc.wantError || calls != tc.wantCalls {
				t.Fatalf("err=%v calls=%d, want error=%v calls=%d", err, calls, tc.wantError, tc.wantCalls)
			}
		})
	}
}
