package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIdempotencyHeaderDoesNotAuthorizeAmbiguousReplay(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, status := range []int{500, 503} {
			c := testClient("http://fixture.invalid")
			c.cacheDir = t.TempDir()
			calls := 0
			c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"ambiguous failure"}`)), Request: req}, nil
			})
			_, _, err := c.do(method, "/v3/billable/task_post", nil, map[string]any{"keyword": "fixture"}, map[string]string{"Idempotency-Key": "not-provider-proof"})
			if err == nil || calls != 1 {
				t.Fatalf("%s status=%d error=%v calls=%d; want error and one call", method, status, err, calls)
			}
		}
	}
}

func TestKeyedWriteRateLimitRecoveryRemainsBounded(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		c := testClient("http://fixture.invalid")
		c.cacheDir = t.TempDir()
		calls := 0
		c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("Idempotency-Key") != "stable-key" {
				t.Error("retry changed caller key")
			}
			status := 200
			if calls == 1 || exhausted {
				status = 429
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: req}, nil
		})
		_, _, err := c.do(http.MethodPost, "/v3/billable/task_post", nil, map[string]any{"keyword": "fixture"}, map[string]string{"Idempotency-Key": "stable-key"})
		want := 2
		if exhausted {
			want = 4
		}
		if (err != nil) != exhausted || calls != want {
			t.Fatalf("exhausted=%v error=%v calls=%d want=%d", exhausted, err, calls, want)
		}
	}
}
