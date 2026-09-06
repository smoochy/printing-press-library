# Beehiiv CLI

**Sync your Beehiiv audience to a local database and answer growth questions offline in one command.**

Beehiiv-pp-cli mirrors publications, subscribers, segments, posts, podcasts, and more into SQLite. Insights commands compute source attribution, churn sources, send-time performance, and cross-publication comparisons with zero API calls. The full v2 surface, including 2026-09 additions like podcasts, exports, and complimentary access, ships as typed commands with dry-run and agent output.

## Install

The recommended path installs both the `beehiiv-pp-cli` binary and the `pp-beehiiv` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install beehiiv
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install beehiiv --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install beehiiv --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install beehiiv --agent claude-code
npx -y @mvanhorn/printing-press-library install beehiiv --agent claude-code --agent codex
```

### Without Node

The generated install path is category-agnostic until this CLI is published. If `npx` is not available before publish, install Node or use the category-specific Go fallback from the public-library entry after publish.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/beehiiv-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install beehiiv --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-beehiiv --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-beehiiv --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install beehiiv --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/beehiiv-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `BEEHIIV_API_KEY` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


Install the MCP binary from this CLI's published public-library entry or pre-built release.

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "beehiiv": {
      "command": "beehiiv-pp-mcp",
      "env": {
        "BEEHIIV_API_KEY": "<your-key>"
      }
    }
  }
}
```

</details>

## Authentication

Create an API key at app.beehiiv.com (Settings > API Keys) and export BEEHIIV_API_KEY. The key is a bearer token scoped to your organization; 180 requests per minute are shared per org.

## Quick Start

```bash
# Verify the binary and auth wiring without calling the API
beehiiv-pp-cli doctor --dry-run

# Mirror the growth-critical entities into local SQLite
beehiiv-pp-cli sync --resources publications,subscriptions,posts --max-pages 50

# Full-text search across synced posts offline
beehiiv-pp-cli search "welcome" --type posts --limit 5

# One-command health snapshot computed from the local store
beehiiv-pp-cli insights growth-summary pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Growth answers from the local store
- **`insights subscriber-sources`** — See exactly where new subscribers come from: UTM, channel, and referring site, grouped in one call.

  _Reach for this when a growth question needs source attribution without paging the full subscriber list through the API._

  ```bash
  beehiiv-pp-cli insights subscriber-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20 --agent
  ```
- **`insights post-performance`** — Review recent sends with status, timing, and expanded stats in one compact table.

  _Reach for this after a send to review performance without burning per-post API calls._

  ```bash
  beehiiv-pp-cli insights post-performance pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 10 --agent
  ```
- **`insights referral-health`** — Check referral-program config and how many subscribers actually carry referral codes.

  _Reach for this when tuning referral loops to see configuration versus real coverage._

  ```bash
  beehiiv-pp-cli insights referral-health pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent
  ```
- **`insights subscriber-lookup`** — Find one subscriber by email or subscription ID and get a compact record instantly.

  _Reach for this for support questions about a single subscriber when offline speed matters._

  ```bash
  beehiiv-pp-cli insights subscriber-lookup pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 reader@example.com --agent --select subscription.email,subscription.status
  ```
- **`insights churn-sources`** — See which sources, channels, and campaigns drive unsubscribes.

  _Reach for this when unsubscribes spike and you need the offending channel fast._

  ```bash
  beehiiv-pp-cli insights churn-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20 --agent
  ```
- **`insights send-times`** — Find your best send slot: open rate by weekday and hour from your own history.

  _Reach for this when scheduling the next send and you want evidence over habit._

  ```bash
  beehiiv-pp-cli insights send-times pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent
  ```
- **`insights compare-publications`** — Side-by-side growth and engagement across every synced publication.

  _Reach for this when managing several publications and a client report needs one comparison table._

  ```bash
  beehiiv-pp-cli insights compare-publications --agent --select publications.name,publications.net_growth
  ```

## Recipes

### Mirror the audience

```bash
beehiiv-pp-cli sync --resources publications,subscriptions,segments,posts --max-pages 100
```

Cursor-paginated sync of the four growth-critical entities into SQLite.

### Agent-ready growth snapshot

```bash
beehiiv-pp-cli insights growth-summary pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent
```

Single read-only health summary computed from the local store.

### Narrow a deep lookup

```bash
beehiiv-pp-cli insights subscriber-lookup pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 reader@example.com --agent --select subscription.email,subscription.status
```

Pair --agent with --select dotted paths to return only the fields an agent needs.

### Attribute a churn spike

```bash
beehiiv-pp-cli insights churn-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20
```

Group unsubscribes by source, channel, UTM, and referrer offline.

### Ship a subscriber CSV

```bash
beehiiv-pp-cli search "@example.com" --type subscriptions --limit 1000 --csv > subscribers.csv
```

Every list and search command emits CSV for spreadsheets.

## Usage

Run `beehiiv-pp-cli --help` for the full command reference and flag list.

## Paths & environment variables

This CLI separates local files into four path kinds:

| Kind | Contents |
|------|----------|
| `config` | User-editable settings such as `config.toml` and saved profiles |
| `data` | Durable local data: `credentials.toml`, `data.db`, cookies, browser-session proof files, and other auth sidecars |
| `state` | Runtime state such as persisted queries, jobs, and `teach.log` |
| `cache` | Regenerable HTTP/cache files |

Each kind resolves independently. The ladder is:

1. Per-kind env var: `BEEHIIV_CONFIG_DIR`, `BEEHIIV_DATA_DIR`, `BEEHIIV_STATE_DIR`, or `BEEHIIV_CACHE_DIR`
2. `--home <dir>` for this invocation
3. `BEEHIIV_HOME` for a flat relocated root
4. XDG env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_CACHE_HOME`
5. Platform defaults matching existing installs

For containers and agent sandboxes, prefer a single relocated root:

```bash
export BEEHIIV_HOME=/srv/beehiiv
beehiiv-pp-cli doctor
```

Under `BEEHIIV_HOME=/srv/beehiiv`, the four dirs resolve to `/srv/beehiiv/config`, `/srv/beehiiv/data`, `/srv/beehiiv/state`, and `/srv/beehiiv/cache`.

MCP servers do not receive CLI flags from the host. Put relocation in the host `env` block:

```json
{
  "mcpServers": {
    "beehiiv": {
      "command": "beehiiv-pp-mcp",
      "env": {
        "BEEHIIV_HOME": "/srv/beehiiv"
      }
    }
  }
}
```

Precedence matters in fleets: an ambient per-kind variable such as `BEEHIIV_DATA_DIR` overrides an explicit `--home` for that kind. Use `BEEHIIV_HOME` or the per-kind variables for durable fleet relocation; treat `--home` as the weaker per-invocation lever.

Relocation is one-way. Unsetting `BEEHIIV_HOME` does not move files back to platform defaults, and `doctor` cannot find credentials left under a former root. Move the files manually before unsetting relocation variables.

Existing installs keep working because the platform-default rung matches the legacy layout. On the first auth write, stored secrets leave `config.toml` and are consolidated into `credentials.toml` under the data directory. Run `beehiiv-pp-cli doctor --fail-on warn` to check path and credential-location warnings in automation.

## Commands

### advertisement-opportunities

Manage advertisement opportunities

- **`beehiiv-pp-cli advertisement-opportunities <publicationId>`** - Get advertisement opportunities <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>

### authors

Manage authors

- **`beehiiv-pp-cli authors index`** - Retrieve a list of authors available for the publication.
- **`beehiiv-pp-cli authors show`** - Retrieve a single author from a publication.

### automations

Manage automations

- **`beehiiv-pp-cli automations index`** - List automations <Badge intent="info" minimal outlined>OAuth Scope: automations:read</Badge>
- **`beehiiv-pp-cli automations show`** - Get automation <Badge intent="info" minimal outlined>OAuth Scope: automations:read</Badge>

### bulk-subscription-updates

Manage bulk subscription updates

- **`beehiiv-pp-cli bulk-subscription-updates index`** - List subscription updates <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:read</Badge>
- **`beehiiv-pp-cli bulk-subscription-updates show`** - Get subscription update <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:read</Badge>

### bulk-subscriptions

Manage bulk subscriptions

- **`beehiiv-pp-cli bulk-subscriptions <publicationId>`** - Bulk create subscription <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>

### complimentary-access

Manage complimentary access

- **`beehiiv-pp-cli complimentary-access index`** - Retrieve complimentary access objects for the publication.
- **`beehiiv-pp-cli complimentary-access show`** - Retrieve a single complimentary access object.

### condition-sets

Manage condition sets

- **`beehiiv-pp-cli condition-sets index`** - Retrieve all active condition sets for a publication. Condition sets define reusable audience segments for targeting content to specific subscribers. Use the `purpose` parameter to filter by a specific use case.
- **`beehiiv-pp-cli condition-sets show`** - Retrieve a single active dynamic content condition set for a publication. Use `expand[]=stats` to calculate and return the active subscriber count synchronously.

### custom-fields

Manage custom fields

- **`beehiiv-pp-cli custom-fields create`** - Create custom field <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:write</Badge>
- **`beehiiv-pp-cli custom-fields delete`** - Delete custom field <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:write</Badge>
- **`beehiiv-pp-cli custom-fields index`** - List custom fields <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:read</Badge>
- **`beehiiv-pp-cli custom-fields patch`** - Update custom field <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:write</Badge>
- **`beehiiv-pp-cli custom-fields put`** - Update custom field <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:write</Badge>
- **`beehiiv-pp-cli custom-fields show`** - Get custom field <Badge intent="info" minimal outlined>OAuth Scope: custom_fields:read</Badge>

### data-privacy

Manage data privacy

- **`beehiiv-pp-cli data-privacy data-deletion-create`** - <Warning>This is a gated feature that requires enablement. Contact support to enable Data Deletion API access for your organization.</Warning>

Creates a data deletion request for a subscriber within your organization. The subscriber's data will be redacted from all publications in the organization after a 14-day safety delay. This action cannot be undone once processing begins.
- **`beehiiv-pp-cli data-privacy data-deletion-index`** - <Warning>This is a gated feature that requires enablement. Contact support to enable Data Deletion API access for your organization.</Warning>

List all data deletion requests for your organization.
- **`beehiiv-pp-cli data-privacy data-deletion-show`** - <Warning>This is a gated feature that requires enablement. Contact support to enable Data Deletion API access for your organization.</Warning>

Retrieve the details and current status of a specific data deletion request.

### email-blasts

Manage email blasts

- **`beehiiv-pp-cli email-blasts index`** - List email blasts <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>
- **`beehiiv-pp-cli email-blasts show`** - Get email blast <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>

### engagements

Manage engagements

- **`beehiiv-pp-cli engagements <publicationId>`** - Retrieve email engagement metrics for a specific publication over a defined date range and granularity.<br><br> By default, the endpoint returns metrics for the past day, aggregated daily. The max number of days allowed is 31. All dates and times are in UTC.

### exports

Manage exports

- **`beehiiv-pp-cli exports subscription-create`** - Start a subscription export. Returns an existing in-progress export instead of starting a duplicate.
- **`beehiiv-pp-cli exports subscription-index`** - List subscription exports for the publication, newest first.
- **`beehiiv-pp-cli exports subscription-show`** - Get a subscription export. Poll until status is completed, then read download_url. Gated feature requiring enablement.

### newsletter-lists

Manage newsletter lists

- **`beehiiv-pp-cli newsletter-lists index`** - <Note title="Currently in beta" icon="b">
  Newsletter Lists is currently in beta, the API is subject to change.
</Note>
List all newsletter lists for a publication.
- **`beehiiv-pp-cli newsletter-lists show`** - <Note title="Currently in beta" icon="b">
  Newsletter Lists is currently in beta, the API is subject to change.
</Note>
Retrieve a single newsletter list belonging to a specific publication.

### podcasts

Manage podcasts

- **`beehiiv-pp-cli podcasts index`** - List podcasts for the publication.
- **`beehiiv-pp-cli podcasts show`** - Retrieve a single podcast.

### polls

Manage polls

- **`beehiiv-pp-cli polls index`** - Retrieve all polls belonging to a specific publication. Poll choices are always included. Use `expand[]=stats` to include aggregate vote counts per choice.
- **`beehiiv-pp-cli polls show`** - Retrieve detailed information about a specific poll belonging to a publication. Use `expand[]=stats` for aggregate vote counts, or `expand[]=poll_responses` for individual subscriber responses.

### post-templates

Manage post templates

- **`beehiiv-pp-cli post-templates <publicationId>`** - Retrieve a list of post templates available for the publication.

### posts

Manage posts

- **`beehiiv-pp-cli posts aggregate-stats`** - Get aggregate stats <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>
- **`beehiiv-pp-cli posts create`** - <Note title="Currently in beta" icon="b">
  This feature is currently in beta, the API is subject to change, and available only to Enterprise users.<br/><br/>To inquire about Enterprise pricing,
  please visit our <a href="https://www.beehiiv.com/enterprise">Enterprise page</a>.
</Note>
Create a post for a specific publication. For a detailed walkthrough including setup, testing workflows, and working with custom HTML and templates, see the <a href="https://www.beehiiv.com/support/article/36759164012439-using-the-send-api-and-create-post-endpoint">Using the Send API and Create Post Endpoint</a> guide.

## Content methods

There are three ways to provide content for a post. You must provide either `blocks` or `body_content`, but not both.

### 1. Blocks

Use the `blocks` field to build your post with structured content blocks such as paragraphs, images, headings, buttons, tables, and more. Each block has a `type` and its own set of properties. This method gives you fine-grained control over individual content elements and supports features like visual settings, visibility settings, and dynamic content targeting.

### 2. Raw HTML (`body_content`)

Use the `body_content` field to provide a single string of raw HTML. The HTML is wrapped in an `htmlSnippet` block internally. This is useful when you have pre-built HTML content or are migrating from another platform.

### 3. HTML blocks within blocks

Use `type: html` blocks inside the `blocks` array to embed raw HTML snippets alongside other structured blocks. This lets you mix structured content (paragraphs, images, etc.) with custom HTML where needed.

## CSS and styling guardrails

beehiiv processes all HTML content through a sanitization pipeline. When using `body_content` or `html` blocks, be aware of the following:

- **`<style>` tags are removed.** All `<style>` block elements are stripped during sanitization. Do not rely on embedded stylesheets.
- **`<link>` tags are removed.** External stylesheet references are not allowed.
- **Inline styles are preserved.** Styles applied directly to elements via the `style` attribute (e.g., `<div style="color: red;">`) are kept intact.
- **CSS classes have no effect.** While class attributes are not stripped, no corresponding stylesheets are loaded to apply them.
- **beehiiv's email template wraps your content.** Your HTML is rendered inside beehiiv's email table structure, which applies its own layout and spacing. This may affect the appearance of your content.
- **Use inline styles for all visual styling.** Since `<style>` and `<link>` tags are removed, inline styles on individual elements are the only reliable way to control appearance.
- **`beehiiv-pp-cli posts delete`** - Delete or Archive a post. Any post that has been confirmed will have it's status changed to `archived`. Posts in the `draft` status will be permanently deleted.
- **`beehiiv-pp-cli posts index`** - List posts <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>
- **`beehiiv-pp-cli posts show`** - Get post <Badge intent="info" minimal outlined>OAuth Scope: posts:read</Badge>
- **`beehiiv-pp-cli posts update`** - <Note title="Currently in beta" icon="b">
  This feature is currently in beta, the API is subject to change, and available only to Enterprise users.<br/><br/>To inquire about Enterprise pricing,
  please visit our <a href="https://www.beehiiv.com/enterprise">Enterprise page</a>.
</Note>
Update an existing post for a specific publication. Only the fields provided in the request body will be updated — all other fields remain unchanged. For a detailed walkthrough of content methods and working with custom HTML, see the <a href="https://www.beehiiv.com/support/article/36759164012439-using-the-send-api-and-create-post-endpoint">Using the Send API and Create Post Endpoint</a> guide.

To update post content, provide either `blocks` or `body_content` (not both). If neither is provided, the existing content is preserved. The same content methods and CSS guardrails described in the create endpoint apply here.

### publications

Manage publications

- **`beehiiv-pp-cli publications index`** - List publications <Badge intent="info" minimal outlined>OAuth Scope: publications:read</Badge>
- **`beehiiv-pp-cli publications show`** - Get publication <Badge intent="info" minimal outlined>OAuth Scope: publications:read</Badge>

### referral-program

Manage referral program

- **`beehiiv-pp-cli referral-program <publicationId>`** - Get referral program <Badge intent="info" minimal outlined>OAuth Scope: referral_program:read</Badge>

### segments

Manage segments

- **`beehiiv-pp-cli segments create`** - Create a new segment.<br><br> **Manual segments** — Use `subscriptions` or `emails` input to create a segment from an explicit list of subscription IDs or email addresses. The segment is processed synchronously and returns with `status: completed`. Net new email addresses will be ignored; create subscriptions using the `Create Subscription` endpoint.<br><br> **Dynamic segments** — Use `custom_fields` input to create a segment that filters subscribers by custom field values. The segment is processed asynchronously and returns with `status: pending`. Results will be available in the `List Segment Subscribers` endpoint after processing is complete.
- **`beehiiv-pp-cli segments delete`** - Delete a segment. Deleting the segment does not effect the subscriptions in the segment.
- **`beehiiv-pp-cli segments index`** - List segments <Badge intent="info" minimal outlined>OAuth Scope: segments:read</Badge>
- **`beehiiv-pp-cli segments show`** - Get segment <Badge intent="info" minimal outlined>OAuth Scope: segments:read</Badge>

### subscriptions

Manage subscriptions

- **`beehiiv-pp-cli subscriptions bulk-updates-patch`** - Update subscriptions <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions bulk-updates-patch-status`** - Update subscriptions' status <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions bulk-updates-put`** - Update subscriptions <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions bulk-updates-put-status`** - Update subscriptions' status <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions create`** - Create subscription <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions delete`** - <Warning>This cannot be undone. All data associated with the subscription will also be deleted. We recommend unsubscribing when possible instead of deleting. If a premium subscription is deleted they will no longer be billed.</Warning> Deletes a subscription.
- **`beehiiv-pp-cli subscriptions get-by-email`** - <Info>Please note that this endpoint requires the email to be URL encoded. Please reference your language's documentation for the correct method of encoding.</Info> Retrieve a single subscription belonging to a specific email address in a specific publication.
- **`beehiiv-pp-cli subscriptions get-by-id`** - <Info>In previous versions of the API, another endpoint existed to retrieve a subscription by the subscriber ID. This endpoint is now deprecated and will be removed in a future version of the API. Please use this endpoint instead. The subscription ID can be found by exporting a list of subscriptions either via the `Settings > Publications > Export Data` or by exporting a CSV in a segment.</Info> Retrieve a single subscription belonging to a specific publication.
- **`beehiiv-pp-cli subscriptions get-by-subscriber-id`** - Get subscription by subscriber ID <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:read</Badge>
- **`beehiiv-pp-cli subscriptions index`** - Retrieve all subscriptions belonging to a specific publication.

<Info> **New**: This endpoint now supports cursor-based pagination for better performance and consistency. Use the `cursor` parameter instead of `page` for new integrations. </Info>
<Warning> **Deprecation Notice**: Offset-based pagination (using `page` parameter) is deprecated and limited to 100 pages maximum. Please migrate to cursor-based pagination. See our [Pagination Guide](/welcome/pagination) for details. </Warning>
- **`beehiiv-pp-cli subscriptions patch`** - Update subscription by ID <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions put`** - Update subscription by ID <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>
- **`beehiiv-pp-cli subscriptions update-by-email`** - Update subscription by email <Badge intent="info" minimal outlined>OAuth Scope: subscriptions:write</Badge>

### tiers

Manage tiers

- **`beehiiv-pp-cli tiers create`** - Create a tier <Badge intent="info" minimal outlined>OAuth Scope: tiers:write</Badge>
- **`beehiiv-pp-cli tiers index`** - List tiers <Badge intent="info" minimal outlined>OAuth Scope: tiers:read</Badge>
- **`beehiiv-pp-cli tiers patch`** - Update a tier <Badge intent="info" minimal outlined>OAuth Scope: tiers:write</Badge>
- **`beehiiv-pp-cli tiers put`** - Update a tier <Badge intent="info" minimal outlined>OAuth Scope: tiers:write</Badge>
- **`beehiiv-pp-cli tiers show`** - Get tier <Badge intent="info" minimal outlined>OAuth Scope: tiers:read</Badge>

### users

Manage users

- **`beehiiv-pp-cli users`** - Identify user <Badge intent="info" minimal outlined>OAuth Scope: identify:read</Badge>

### webhooks

Manage webhooks

- **`beehiiv-pp-cli webhooks create`** - Create a webhook <Badge intent="info" minimal outlined>OAuth Scope: webhooks:write</Badge>
- **`beehiiv-pp-cli webhooks delete`** - Delete a webhook <Badge intent="info" minimal outlined>OAuth Scope: webhooks:write</Badge>
- **`beehiiv-pp-cli webhooks index`** - List webhooks <Badge intent="info" minimal outlined>OAuth Scope: webhooks:read</Badge>
- **`beehiiv-pp-cli webhooks show`** - Get webhook <Badge intent="info" minimal outlined>OAuth Scope: webhooks:read</Badge>
- **`beehiiv-pp-cli webhooks update`** - Update webhook <Badge intent="info" minimal outlined>OAuth Scope: webhooks:write</Badge>

### workspaces

Manage workspaces

- **`beehiiv-pp-cli workspaces identify`** - Identify workspace <Badge intent="info" minimal outlined>OAuth Scope: identify:read</Badge>
- **`beehiiv-pp-cli workspaces permissions-show`** - Retrieve the permissions granted to the OAuth or API token for this workspace.
- **`beehiiv-pp-cli workspaces publications-by-subscription-email`** - Retrieve all publications in the workspace that have a subscription for the specified email address. The workspace is determined by the provided API key.


### Self-learning loop

This CLI caches per-question discovery so repeat queries skip the walk and structurally similar queries get answered via entity substitution. The loop also self-captures: every invocation is journaled locally, and failed-flag corrections plus fresh teaches surface as candidates on the next `recall` for confirm/reject judgment. Agents call `recall` before discovery and fire `teach &` after answering. See the `## Automatic learning` section in `SKILL.md` for the full protocol.

- **`beehiiv-pp-cli recall <query>`** - Look up cached resources for a query before running discovery
- **`beehiiv-pp-cli teach`** - Record a query -> resource mapping (silent on success, safe to background with `&`)
- **`beehiiv-pp-cli learnings list`** - Inspect taught rows
- **`beehiiv-pp-cli learnings forget <query>`** - Undo a teach
- **`beehiiv-pp-cli learnings candidates`** - List auto-captured candidates awaiting confirm/reject
- **`beehiiv-pp-cli learnings stats`** - Local loop metrics: recall hit rate, teach-to-reuse, playbook resolution, candidate counts
- **`beehiiv-pp-cli teach-pattern`** - Install a query/resource template up front
- **`beehiiv-pp-cli teach-lookup`** - Add an entity mapping (e.g. country code, team alias) for pattern substitution

Pass `--no-learn` or set `BEEHIIV_NO_LEARN=true` to disable the loop for deterministic flows.

The local store's schema version stamp is one-way: once this version of `beehiiv-pp-cli` opens the database, older binaries refuse it with a version error — upgrade the binary rather than downgrading.

## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
beehiiv-pp-cli advertisement-opportunities mock-value

# JSON for scripting and agents
beehiiv-pp-cli advertisement-opportunities mock-value --json
# Filter to specific fields
beehiiv-pp-cli advertisement-opportunities mock-value --json --select advertisement_kind,advertiser_name,id

# Dry run — show the request without sending
beehiiv-pp-cli advertisement-opportunities mock-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
beehiiv-pp-cli advertisement-opportunities mock-value --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select <field>[,<field>...]` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and add `--ignore-missing` to delete retries when a no-op success is acceptable
- **Explicit confirmation** - `--agent` does not imply `--yes`; pass `--yes` separately only after the target, arguments, and side effects are clear
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
beehiiv-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Run `beehiiv-pp-cli doctor` to see the resolved config, data, state, and cache directories. The platform-default config path is `~/.config/beehiiv-pp-cli/config.toml`; `--home`, `BEEHIIV_HOME`, and per-kind env vars can relocate it.

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `BEEHIIV_API_KEY` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `beehiiv-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `beehiiv-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $BEEHIIV_API_KEY`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **401 INVALID_API_KEY on every call** — export BEEHIIV_API_KEY=<key from app.beehiiv.com settings>
- **429 Too Many Requests** — The CLI applies adaptive backoff; wait for the Retry-After window and reduce concurrent jobs
- **List endpoints cap at page 100** — Use cursor pagination; the CLI sync already prefers cursor tokens
- **Export download_url is null** — Poll beehiiv-pp-cli exports subscription-show <publicationId> <id> until status is completed, then read download_url

## Sources & Inspiration

This CLI was built by studying these projects and resources:

- [**deldrid1/beehiiv-cli**](https://github.com/deldrid1/beehiiv-cli) — Go (2 stars)
- [**beehiiv-pp-cli (prior print)**](https://github.com/mvanhorn/printing-press-library/tree/main/library/marketing/beehiiv) — Go
- [**Frozen-Software-LLC/mcp-beehiiv**](https://github.com/Frozen-Software-LLC/mcp-beehiiv) — TypeScript
- [**beehiiv/typescript-sdk**](https://github.com/beehiiv/typescript-sdk) — TypeScript

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
