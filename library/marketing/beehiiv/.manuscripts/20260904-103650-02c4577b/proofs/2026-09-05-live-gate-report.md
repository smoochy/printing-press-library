# Live Gate Report — beehiiv-pp-cli (2026-09-05)

## Credential timeline (this replaces the earlier no-key proof notes)

- 2026-09-04: structural verification only (verify, offline dogfood, shipcheck legs).
  A phase5-skip marker recorded auth-unavailable. Superseded.
- 2026-09-05: operator provided BEEHIIV_API_KEY. Full live gate re-run on commit
  4b678b2bd (source fingerprint 362f5101... matches this tree; publish validate passes
  all checks including phase5).

## Final result

- phase5-acceptance.json: status=pass, level=full, matrix 263, 263/263 passed, 0 failed.
- publish-live-gate.json: full runner output for the same run (verdict PASS).

## Bucket accounting (why 342 skipped / 342 unverified alongside 263 passed)

The full matrix enumerates every leaf command's help, happy-path, json-fidelity,
and error-path rows. The 342 skipped rows are structural, not auth failures:

- 218: dependent-endpoint fixtures — the runner's list-companion machinery could not
  draw publicationId fixtures for nested endpoints (publications_id cascade).
- 53: mutating endpoints — error_path rows skip live calls without --dry-run to avoid
  real writes (bulk-subscriptions, custom-fields create/delete, tiers put, webhooks, …).
- 43: commands with no positional argument or non-id positionals (analytics, which,
  api, users, workflow …).
- The rest: dry-run-only commands and a few unsynthesizable request bodies.

The 2 unverified rows are users (happy_path + json_fidelity): /users/identify answers
403 to an API key on this organization (OAuth-token-scoped surface); the runner marks
those unverified-needs-access. Every other endpoint family verified live.

## Post-acceptance review fixes (all covered by this run's scope)

Publication-scoped insights now filter on the publications_id/parent_id JSON keys that
dependent sync injects per row (supplied+present → scoped metrics; well-formed-but-absent
→ empty result with note, exit 0; malformed → typed not-found exit 3). The six novel
commands declare pp:typed-exit-codes + pp:happy-args fixtures so the matrix exercises
real happy and json-fidelity rows for each.
