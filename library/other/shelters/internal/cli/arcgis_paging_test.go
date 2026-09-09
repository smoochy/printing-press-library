// Copyright 2026 Abe Diaz (@abe238) and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

// TestRedCrossHiddenFetchPaginates verifies the safety-critical hidden-set
// fetch follows exceededTransferLimit and uses stable OID ordering.
func TestRedCrossHiddenFetchPaginates(t *testing.T) {
	previous := getArcGISPage
	t.Cleanup(func() { getArcGISPage = previous })
	calls := 0
	getArcGISPage = func(_ context.Context, rawURL, referer string) ([]byte, error) {
		calls++
		parsed, err := url.Parse(rawURL)
		if err != nil {
			t.Fatalf("parse request URL: %v", err)
		}
		if referer != redCrossReferer {
			t.Errorf("referer = %q, want %q", referer, redCrossReferer)
		}
		if got := parsed.Query().Get("orderByFields"); got != "ObjectID ASC" {
			t.Errorf("orderByFields = %q, want ObjectID ASC", got)
		}
		switch parsed.Query().Get("resultOffset") {
		case "0":
			return []byte(`{"features":[{"attributes":{"active_site_name":"Hidden One","active_state":"TX","active_zip":"75001"}}],"exceededTransferLimit":true}`), nil
		case "1":
			return []byte(`{"features":[{"attributes":{"active_site_name":"Hidden Two","active_state":"LA","active_zip":"70001"}}]}`), nil
		default:
			t.Fatalf("unexpected resultOffset in %s", rawURL)
			return nil, nil
		}
	}

	hidden, err := redCrossHiddenFetch(context.Background())
	if err != nil {
		t.Fatalf("redCrossHiddenFetch: %v", err)
	}
	if calls != 2 || len(hidden) != 2 {
		t.Fatalf("calls=%d hidden=%d, want 2 / 2", calls, len(hidden))
	}
}

func TestArcGISFetchReportsUnsupportedPaginationTruncation(t *testing.T) {
	previous := getArcGISPage
	t.Cleanup(func() { getArcGISPage = previous })
	getArcGISPage = func(context.Context, string, string) ([]byte, error) {
		return []byte(`{"features":[{"attributes":{"id":1}}],"exceededTransferLimit":true}`), nil
	}

	_, err := fetchArcGISPages(context.Background(), arcGISQuery{
		URL:      "https://example.test/query",
		OIDField: "id",
	}, url.Values{"f": {"json"}})
	if err == nil || !strings.Contains(err.Error(), "does not support pagination") {
		t.Fatalf("error = %v, want explicit truncation report", err)
	}
}

// TestHiddenFetchFailsClosedOnServiceError covers the classification the
// fail-closed path depends on: an ArcGIS response can be HTTP 200 and valid
// JSON while carrying a service error or no features at all. Either shape must
// reach the caller as an error, so the visibility check cannot silently read as
// "no hidden shelters" and let a hidden site through.
func TestHiddenFetchFailsClosedOnServiceError(t *testing.T) {
	for name, body := range map[string]string{
		"service error": `{"error":{"code":400,"message":"Unable to complete operation."}}`,
		"no features":   `{"count":0}`,
	} {
		t.Run(name, func(t *testing.T) {
			previous := getArcGISPage
			t.Cleanup(func() { getArcGISPage = previous })
			getArcGISPage = func(context.Context, string, string) ([]byte, error) {
				return []byte(body), nil
			}
			if _, err := redCrossHiddenFetch(context.Background()); err == nil {
				t.Fatal("redCrossHiddenFetch = nil error, want a failure so suppression fails closed")
			}
		})
	}
}
