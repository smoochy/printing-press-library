# Regen-merge final decisions (beehiiv reprint 2026-09-04)

Path B applied, then completed to a full fresh-tree sync because v4.2.2-era preserved
templates (helpers.go, root.go, sync.go, client.go, tools.go, shellout.go, store.go,
upsert_batch_test.go) were incompatible with v4.31.1 emissions. Upstream absorbed the
functional substance of most patches:

- escaped-path-params: upstream cliutil.EscapePathParam absorbed; re-tightened in
  helpers.go replacePathParam ('+'/'@' strict encoding) because url.PathEscape leaves
  them and beehiiv by_email paths misread them. Path: internal/cli/helpers.go.
- private-response-cache: upstream hardenSQLiteFiles (0600 files/journal) absorbed.
- nested-dependent-sync-paths: upstream dependent sync resolution absorbed.
- workflow-archive-endpoints: fresh archive calls only /publications (no invalid
  collection calls); identity-enrichment calls dropped upstream. Test updated to the
  fresh contract (channel_workflow_test.go).
- fresh-jwt-token-reads: re-applied in subscriptions_jwt-token_subscriptions-get.go
  via GetWithHeadersNoCache (short-lived JWTs never served from cache).
- catalog Go floor bumps: absorbed by fresh go.mod.
- review-polish: N/A (bearer-only CLI, no refresh-token paths).
- beehiiv-insights (6 live-API insight commands): superseded by 8 store-computed
  insights commands (see insights-go-supersession-note.md).

Dropped stale tests referencing replaced APIs: cache_policy_test.go (GetFreshWithHeaders
-> GetWithHeadersNoCache), sync_dependent_test.go (ScopeTable field).
Final state: full fresh v4.31.1 tree + jwt no-cache patch + strict escape patch +
2 adapted patch tests. go build/test/vet all green.
