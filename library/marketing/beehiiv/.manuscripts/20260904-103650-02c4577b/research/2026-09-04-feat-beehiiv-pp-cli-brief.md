# Beehiiv CLI Brief (reprint, redo research 2026-09-04)

## API Identity
- Domain: Newsletter growth platform API (v2 REST). Base: https://api.beehiiv.com/v2
- Users: Newsletter operators, growth hackers, agencies managing many publications.
- Data profile: Publications, subscriptions (PII: emails), segments, posts, automations, tiers, webhooks, podcasts, exports, complimentary access, ad network. Cursor pagination. Timestamps are unix seconds.

## Reachability Risk
- Low. Official documented REST API. 180 req/min per org; 429 with rate headers. Bearer token.
- No token in env; live dogfood will be skipped or gated on user-provided BEEHIIV_API_KEY.
- Probe-safe endpoint used: none declared; base GET returns 401 without key (expected).

## Top Workflows
1. Sync audience to local SQLite; query/filter subscribers offline without burning rate limit.
2. Growth analysis: subscriber sources (UTM/channel), growth summary, post performance.
3. Segment operations: list/expand/recalculate, member counts, stale detection.
4. Post lifecycle: draft/schedule/send state tracking, aggregate stats, preview/test send.
5. Newsletter list management: create/update/delete lists and list subscriptions; bulk updates.
6. Podcast private feed distribution: mint/send private RSS links per subscriber.
7. Data exports: start subscription export, poll to completion, fetch signed CSV link.

## Table Stakes (absorb targets)
- Prior CLI (v4.2.2, 2026-05-10): 82 typed endpoints, 6 insights commands, SQLite sync, offline search, --json/--select/--agent.
- Official docs MCP exists for docs navigation, not the API itself. Zapier/Make automate but no local analytics.

## Data Layer
- Primary entities: publications, subscriptions, segments, posts, automations, tiers, webhooks, newsletter_lists, custom_fields, podcasts, episodes, exports, complimentary_access.
- Sync cursor: cursor-based pagination everywhere; offset `page` param deprecated (page>100 -> 400, warning headers).
- FTS/search: subscribers by email/UTM/status; posts by title/status.

## API Churn Found By Redo (2026-09)
- Cursor pagination is now recommended; offset deprecated (cap page 100). List/sync must prefer cursor.
- New documented endpoints missing from 2026-05 spec (16 ops): complimentary_access index/show, exports create/index/show, podcasts 8 ops (shows, episodes, private feeds by id/email, send feed emails), workspaces/permissions, posts preview/test_send.
- posts/preview returns app.beehiiv.com preview_url (needs app session to view). posts/test_send consumes daily test-send quota (write op).
- Rate limit 180 req/min per organization; 429 + Retry-After style headers.

## User Vision
- Reprint on machine v4.31.1 (prior v4.2.2). Enrichments approved by user: MCP surface (stdio+http, thin orchestration), cache freshness block, canonical auth env var BEEHIIV_API_KEY (prior used slug-derived BEEHIIV_BEARER_AUTH).
- Carry 9 public patches as watch-list (insights commands, path escaping, JWT cache bypass, nested sync paths, private cache perms, refresh-token hard-fail, archive endpoint routing, Go floor bumps).

## Product Thesis
- Name: beehiiv-pp-cli
- Why: The only beehiiv client that syncs audience to a local database and answers growth questions offline in one command, now with current-machine MCP surface and the 2026-09 API additions.

## Build Priorities
1. 16 new documented endpoints as typed commands (podcasts, exports, complimentary access, workspace permissions, post preview/test).
2. Cursor-paginated sync across all list endpoints.
3. Insights family rebuilt on the enlarged store (add podcast/export coverage).
4. Canonical auth env var BEEHIIV_API_KEY.
5. MCP: stdio+http transport; >50 typed endpoints -> auto thin search+execute pattern.
6. Cache freshness on syncable reads; stale_after tuned to newsletter cadence (24h).
