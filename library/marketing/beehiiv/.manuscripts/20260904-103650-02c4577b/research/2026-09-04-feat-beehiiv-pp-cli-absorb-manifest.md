# Beehiiv CLI Absorb Manifest (reprint 2026-09-04)

## Absorbed (match or beat everything that exists)
| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-----------|-------------------|-------------|
| 1 | Full v2 API surface: 70 paths / 98 ops (16 new documented ops added) | prior CLI + raw 2026-09 docs | (generated endpoint) typed resource commands from merged spec | typed flags, --dry-run, typed exits |
| 2 | Auto pagination aggregation | deldrid1/beehiiv-cli --all | (behavior in beehiiv-pp-cli sync --max-pages) cursor-paginated full sync | full local mirror, no manual paging |
| 3 | CSV export everywhere | deldrid1 CLI; official MCP gap | (behavior in beehiiv-pp-cli search --csv) | all list/search output modes |
| 4 | Default publication ID | mcp-beehiiv BEEHIIV_PUBLICATION_ID | (behavior in beehiiv-pp-cli publications list) path-param default from BEEHIIV_PUBLICATION_ID when flag omitted | single-publication convenience |
| 5 | Rate-limit + retry with typed throttle signal | deldrid1 CLI | (behavior in beehiiv-pp-cli publications list) AdaptiveLimiter + RateLimitError | throttle never reads as "no data" |
| 6 | Canonical auth env var | beehiiv docs / SDK convention | (behavior in beehiiv-pp-cli doctor) BEEHIIV_API_KEY checked and reported | replaces slug-derived BEEHIIV_BEARER_AUTH |
| 7 | Growth summary | prior CLI (public patch beehiiv-insights) | beehiiv-pp-cli insights growth-summary | rebuilt on enlarged store (podcasts, exports) |
| 8 | Subscriber sources | prior CLI (public patch beehiiv-insights) | beehiiv-pp-cli insights subscriber-sources | UTM/channel/referrer grouping |
| 9 | Post performance | prior CLI (public patch beehiiv-insights) | beehiiv-pp-cli insights post-performance | reframed: carries aggregate-stats/deliverability fields |
| 10 | Referral health | prior CLI (public patch beehiiv-insights) | beehiiv-pp-cli insights referral-health | config + code coverage |
| 11 | Subscriber lookup | prior CLI (public patch beehiiv-insights) | beehiiv-pp-cli insights subscriber-lookup | email or ID, compact record |
| 12 | MCP server covering the API | mcp-beehiiv 78 raw tools | (behavior in beehiiv-pp-cli doctor --json) stdio+http MCP; >50 endpoints -> thin search+execute orchestration | token-efficient agent surface |
| 13 | 9 public patches carried as watch-list | /printing-press-amend records | (behavior in beehiiv-pp-cli doctor) regressions guarded: path escaping, JWT cache bypass, nested sync paths, private cache perms, archive routing | no silent regression of API truth |

### Reprint verdict notes
- `insights field-coverage` DROPPED per subagent reprint verdict (monthly use). User may override at the Phase Gate 1.5.
- OAuth keychain login: intentionally out of scope (API-key CLI). Prior patch review-polish kept: unsupported refresh-token paths fail explicitly.

## Transcendence (only possible with our approach)
| # | Feature | Command | Buildability | Why Only We Can Do This | Long Description |
|---|---------|---------|--------------|------------------------|------------------|
| 1 | Subscriber Sources | insights subscriber-sources | hand-code | Groups subscriptions by UTM/channel/referring-site in local SQLite; zero API calls at query time | Use this command for total audience acquisition by source. Do NOT use it for unsubscribe attribution; use 'insights churn-sources' instead. |
| 2 | Post Performance | insights post-performance | hand-code | Compact per-send review from posts table incl. aggregate-stats fields; no per-post fan-out | Use this command for per-post detail. Do NOT use it for the account-level snapshot; use 'insights growth-summary' instead. |
| 3 | Referral Health | insights referral-health | hand-code | Joins publication referral config with subscriber code coverage locally | none |
| 4 | Subscriber Lookup | insights subscriber-lookup | hand-code | Offline single-subscriber resolution; no rate-limit cost | Use this command for one subscriber by email or ID. Do NOT use it for source-attribution counts; use 'insights subscriber-sources' instead. |
| 5 | Churn Sources | insights churn-sources | hand-code | Unsubscribes grouped by source/channel/UTM/referrer from the local store | Use this command for unsubscribe attribution. Do NOT use it for total acquisition by source; use 'insights subscriber-sources' instead. |
| 6 | Send-Times | insights send-times | hand-code | Open rate by send weekday+hour from post timestamps joined to aggregate stats | none |
| 7 | Compare Publications | insights compare-publications | hand-code | Cross-publication rollups over synced publications/subscriptions/posts | Use this command for cross-publication side-by-side comparison. Do NOT use it for single-publication health; use 'insights growth-summary' instead. |

Hand-code commitment: 7 novel features, all hand-code (~50-150 LoC each + wiring). Growth Summary rebuild ships as absorbed row 7 (also hand-code).
