package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/commerce/square/internal/config"
)

type writeSafetyTransport func(*http.Request) (*http.Response, error)

func (f writeSafetyTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestAmbiguousWritesNeverReplay(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "")
	paths := []string{"/write-fixture"}
	for _, path := range paths {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			for _, status := range []int{0, 500, 503} {
				t.Run(path+"/"+method+"/"+http.StatusText(status), func(t *testing.T) {
					c := New(&config.Config{BaseURL: "http://fixture.invalid", AuthHeaderVal: "fixture-token"}, time.Second, 0)
					c.cacheDir = t.TempDir()
					calls := 0
					c.HTTPClient.Transport = writeSafetyTransport(func(req *http.Request) (*http.Response, error) {
						calls++
						if status == 0 {
							return nil, errors.New("synthetic reset after possible commit")
						}
						return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"synthetic failure"}`)), Request: req}, nil
					})
					_, _, err := c.do(context.Background(), method, path, nil, map[string]any{"value": "once"}, map[string]string{"Authorization": "fixture-token", "Idempotency-Key": "not-proof-of-provider-deduplication"})
					if err == nil || calls != 1 {
						t.Fatalf("error=%v calls=%d; want error and exactly one attempt", err, calls)
					}
				})
			}
		}
	}
}

func TestWriteRateLimitRecoveryRemainsBounded(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "")
	for _, exhausted := range []bool{false, true} {
		c := New(&config.Config{BaseURL: "http://fixture.invalid", AuthHeaderVal: "fixture-token"}, time.Second, 0)
		c.cacheDir = t.TempDir()
		calls := 0
		firstRequestID := ""
		c.HTTPClient.Transport = writeSafetyTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				firstRequestID = req.Header.Get("__requestid")
			} else if req.Header.Get("__requestid") != firstRequestID {
				t.Error("retry changed request identity")
			}
			status := http.StatusOK
			if calls == 1 || exhausted {
				status = http.StatusTooManyRequests
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
		})
		_, _, err := c.do(context.Background(), http.MethodPost, "/write-fixture", nil, map[string]any{"value": "once"}, map[string]string{"Authorization": "fixture-token", "Idempotency-Key": "stable-key"})
		wantCalls := 2
		if exhausted {
			wantCalls = 4
		}
		if (err != nil) != exhausted || calls != wantCalls {
			t.Fatalf("exhausted=%v err=%v calls=%d want=%d", exhausted, err, calls, wantCalls)
		}
	}
}

func TestExplicitReadOnlyIntentStillRetriesServerFailure(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "")
	c := New(&config.Config{BaseURL: "http://fixture.invalid", AuthHeaderVal: "fixture-token"}, time.Second, 0)
	c.cacheDir = t.TempDir()
	calls := 0
	c.HTTPClient.Transport = writeSafetyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		status := 200
		if calls == 1 {
			status = 503
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
	})
	_, _, err := c.doRead(context.Background(), http.MethodPost, "/read-fixture", nil, map[string]any{"query": "fixture"}, nil)
	if err != nil || calls != 2 {
		t.Fatalf("error=%v calls=%d; want safe read recovery", err, calls)
	}
}
