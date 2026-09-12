package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type writeRetryTransport func(*http.Request) (*http.Response, error)

func (f writeRetryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWriteFailureDoesNotReplayAcrossSources(t *testing.T) {
	t.Setenv("PRINTING_PRESS_VERIFY", "1")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "1")
	for _, tc := range []struct {
		name                     string
		status                   int
		readOnly, transportError bool
		calls                    int
		wantError                bool
	}{
		{"write_server", 503, false, false, 1, true},
		{"write_transport", 0, false, true, 1, true},
		{"read_only_post_failover", 503, true, false, 2, false},
		{"write_rate_limit", 429, false, false, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Zero transient retries exercises cross-source replay separately.
			// Rate-limit recovery uses the normal bounded retry budget.
			t.Setenv("PRINTING_PRESS_DOGFOOD", "1")
			if tc.status == 429 {
				t.Setenv("PRINTING_PRESS_DOGFOOD", "")
				t.Setenv("PRINTING_PRESS_VERIFY", "")
			}
			c := newFailoverClient("http://primary.invalid", "http://secondary.invalid")
			c.cacheDir = t.TempDir()
			calls := 0
			c.HTTPClient.Transport = writeRetryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.status == 429 && r.URL.Host != "primary.invalid" {
					t.Errorf("429 failed over to %s", r.URL.Host)
				}
				if calls == 1 && tc.transportError {
					return nil, errors.New("synthetic connection reset")
				}
				status := 200
				if calls == 1 {
					status = tc.status
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}, "Retry-After": []string{"1"}}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Request: r}, nil
			})
			_, _, err := c.doInternal(context.Background(), "POST", "/resource", nil, nil, nil, tc.readOnly)
			if calls != tc.calls || (err != nil) != tc.wantError {
				t.Fatalf("calls=%d err=%v, want calls=%d error=%v", calls, err, tc.calls, tc.wantError)
			}
		})
	}
}
