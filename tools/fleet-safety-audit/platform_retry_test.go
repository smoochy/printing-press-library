package main

import (
	"bytes"
	"testing"
)

func TestPlatformRetrySeparatesAmbiguityFromRejection(t *testing.T) {
	for _, mutationIntent := range []bool{false, true} {
		policy := "readOnlyIntent || platform.CanRetryRequest(method, requestIdempotencyKey(c.Config, headerOverrides))"
		if mutationIntent {
			policy = "readOnlyIntent || (!mutationIntent && platform.CanRetryRequest(method, requestIdempotencyKey(c.Config, headerOverrides)))"
		}
		in := bytes.Replace([]byte(retryFixture), []byte("const maxRetries = 3"), []byte("canRetryAmbiguousFailure := "+policy+"\nconst maxRetries = 3"), 1)
		in = bytes.Replace(in, []byte("resp.StatusCode == 429 && attempt < maxRetries"), []byte("resp.StatusCode == 429 && attempt < maxRetries && canRetryAmbiguousFailure"), 1)
		got := retrofitRetry(in, &cluster{files: map[string]bool{}}, "client.go")
		if !bytes.Contains(got, []byte("canRetryRejectedRequest := "+policy)) || !bytes.Contains(got, []byte("resp.StatusCode == 429 && attempt < maxRetries && canRetryRejectedRequest")) {
			t.Fatalf("lost rejection policy:\n%s", got)
		}
		if bytes.Contains(got, []byte("canRetryAmbiguousFailure := "+policy)) {
			t.Fatal("kept unproven platform policy for ambiguous failures")
		}
		if again := retrofitRetry(got, &cluster{files: map[string]bool{}}, "client.go"); !bytes.Equal(got, again) {
			t.Fatal("platform retrofit not idempotent")
		}
	}
}

func TestWidenedNamedGuardFailsClosed(t *testing.T) {
	for _, expression := range []string{"true", "readOnlyIntent || method == http.MethodPut", "platform.CanRetryRequest(method, key)", "providerPolicy(method)"} {
		t.Run(expression, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("widened named guard escaped audit")
				}
			}()
			validateAmbiguousRetryPolicies([]byte("package client\nfunc do(method string){canRetryAmbiguousFailure := "+expression+"}"), "client.go")
		})
	}
}
