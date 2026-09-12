# Fleet safety retrofit

This tool audits recognized generated-source patterns without printing CLIs or
changing catalog indexes. Run it from the repository root:

```sh
GO111MODULE=off go run ./tools/fleet-safety-audit
GO111MODULE=off go test ./tools/fleet-safety-audit
```

The audit exits nonzero when it finds a candidate. `-verbose` lists candidates;
`-write` applies the mechanical repairs and creates per-CLI reprint guards.
Inspect the candidate diff before committing. A zero result covers the patterns
this tool recognizes, not every possible bespoke implementation.

The retry repair preserves the existing retry budget for authentication refresh
and rate-limit responses. Only transport errors and server errors are restricted
to GET, HEAD, OPTIONS, or explicit read-only intent. Each transport and server
branch is checked separately; a guarded branch does not exempt its file. Both
positive guarded-continue and negative early-return styles are recognized.
Fixed-method helpers are left alone. Soccer Goat's separate
cross-source failover boundary requires a manual repair and executable tests.

The other repairs encode path parameters (including dot-only segments), remove
constant-true parameter guards, and report zero stored rows when a batch rolls
back. They do not advance sync watermarks or change MCP tool registration.

For a staged candidate, independently replay every mechanical source edit from
the review base and require a tracked reprint guard for every changed source:

```sh
FLEET_AUDIT_BASE=upstream/main GO111MODULE=off go test ./tools/fleet-safety-audit -count=1 -v
git diff --cached --check
```

## Issue 1977 integration

The retrofit was applied directly to upstream/main
`ce7f84011f58333dc2a89ba3a9b382dcf13ef696`. It does not carry the old branch's
library diff. This base produced 441 mechanical retry-client changes, 265 path
helpers, 27 parameter files, and 346 stores. Soccer Goat adds one manually
reviewed client change. FedEx and Paperclip's already-safe retry policies remain
unchanged. Paperclip's three requested guard records are tracked explicitly.

Representative checks use local fixtures and temporary configuration directories,
not production credentials or provider requests. Amazon Ads tests exercise write
401 refresh, bounded 429 recovery, ambiguous write failures, read-only POST
recovery, and rollback counts. Soccer Goat tests cover cross-source replay.
The full fleet replay complements representative execution; it does not claim
that every CLI's test suite was executed.

## Reviewed retry policies

OrderToGo's endpoint-specific exclusion did not protect payment checkout or
coupon mark-used. Conduyt CRM, GitHub, and Lunch Money's method filters allowed
PUT or DELETE without establishing operation-level replay safety. Their 5xx
conditions now also require the existing conservative safety gate. Tests cover
POST, PUT, PATCH, and DELETE transport/500/503 failures and bounded 429 recovery.

The audit recognizes these bounded exceptions by client, function, and a pinned
fingerprint of the helper contract. A changed fingerprint fails the audit and
requires another manual review:

| Client | Preserved contract and evidence |
| --- | --- |
| FedEx | Exact read-only POST path allowlist, including rate quotes and tracking; existing client tests cover allowed reads, denied shipment/pickup writes, and unknown future paths. |
| Paperclip | GET/HEAD/OPTIONS or explicit read-only intent; existing client tests cover its mutation boundary. |
| DataForSEO | GET/HEAD only for ambiguous failures. A caller-supplied Idempotency-Key header does not establish provider deduplication. The pre-existing bounded keyed-write 429 recovery remains separate and is tested. |

Jobber's GraphQL helper rejects non-GET intent before making a request. USPTO's
GetJSON helper constructs GET requests only. Neither is a general write
transport. Soccer Goat's separately reviewed failover boundary remains covered
by its existing write-replay tests.

Another 79 clients delegated their guard to platform.CanRetryRequest, which
treated PUT/DELETE or a supplied key as safe without an operation-level provider
contract. Their ambiguous-failure predicate now accepts only read-only methods
or explicit read-only intent. Existing mutationIntent exclusions remain in
place, including for wire-level GET mutations. Their original platform policy
is preserved separately for bounded 429 rejection recovery. Seven representative
clients have regression tests, covering all six variants without mutationIntent
(Crestron, Cosmos, D&D Beyond, Square, AMC Theatres, and Judge.me) plus Exa.
The audit validates guard declarations as well as their use, so replacing the
predicate with a wider method/key policy fails validation.
