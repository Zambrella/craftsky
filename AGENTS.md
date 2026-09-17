# AGENTS.md

Guidance for AI agents and human contributors working in this repository.

## Project Overview

**Production infrastructure is deployed, but the app has no active users, so backward compatibility for shipped app behavior is not yet required. Production data, credentials, and infrastructure still require normal change controls.**

Craftsky is a crafting-focused social platform built on the [AT Protocol](https://atproto.com). It is a federated, user-owned alternative to Instagram/Pinterest for the crafting community.

Read [atproto-craft-social-app-reference.md](atproto-craft-social-app-reference.md) before making architectural decisions — it captures the full design rationale (PDS vs App View split, auth flow, lexicon design, tech stack choices).

For the product-level "why" and the community-facing feature intent, read the vision doc: [Building a Social Network for Textile Crafters](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing). It's the source of truth for product principles (no ads ever, data portability, transparent business accounts, chronological feed, no algorithmic ranking) and for the project-post field set. If a technical decision conflicts with this doc, flag it — don't silently diverge.

## Dev Workflow

`just dev` (from the repo root) starts the full compose stack — `postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`. The appview runs only inside Docker in dev; there is no `go run ./cmd/appview` path. Go tests run on the host with `just test` against the compose Postgres (the appview image has no Go toolchain). See `justfile` for the full recipe list.

## Production Hosting (Render)

The backend is hosted on Render. [`render.yaml`](render.yaml) is the source of truth for infrastructure configuration, [ADR 016](adr/016-render-managed-production-infrastructure.md) records the architecture decision, and [`docs/operations/production-render.md`](docs/operations/production-render.md) is the deployment and recovery runbook.

Basic production configuration:

| Resource | Render name | Configuration |
|---|---|---|
| Blueprint project | `craftsky` | Environment `production`; private-network isolation and protection enabled |
| AppView | `craftsky-appview` | Docker web service from this repository, branch `main`, root directory `appview`, Frankfurt, one `0.5c-512mb` instance |
| Tap | `craftsky-tap` | Private image service, Frankfurt, one `0.5c-512mb` instance, 1 GB disk mounted at `/data` |
| PostgreSQL | `craftsky-prod-db` | PostgreSQL 16, Frankfurt, database/user `craftsky`, Basic 1 GB plan, 5 GB autoscaling storage, no public IP allowlist |
| Environment group | `craftsky-appview-prod-config` | Non-secret AppView production configuration |

- Public AppView origin: `https://appview.craftsky.social`.
- Shallow Render restart probe: `GET /health`. Operator/deployment health: `GET /healthz`.
- Tap is private and AppView connects to it over Render's private network.
- Scheduled private media is stored in AWS S3 in `eu-central-1`, not on Render disks.
- AppView uses the direct internal PostgreSQL URL. Do not switch it to transaction-mode PgBouncer; owner lifecycle fencing relies on session advisory locks.
- AppView's 1 GB disk at `/var/lib/craftsky-deploy-serialization` must remain empty. It exists only to prevent overlapping singleton deployments.
- Service auto-deploy and Blueprint Auto Sync are disabled. Infrastructure changes require a reviewed manual Blueprint sync.
- Routine application releases are created and deployed from the maintainer's local machine. `scripts/release` creates the version/changelog commit and immutable `prod-vX.Y.Z` tag; `scripts/appview-deploy` deploys that exact pushed tag and verifies public health. The tag must equal the semantic version embedded from `appview/VERSION`.
- The pre-deploy command validates production dependencies before applying migrations: `/app/cli --env prod ping && /app/cli --env prod migrate up`.

When using the Render MCP tools:

1. Call `list_workspaces` first and ask the user to confirm the intended workspace. Pass that confirmed `workspaceId` explicitly on every later call; do not rely on session-selected workspace state.
2. Discover resource IDs with `list_services` and `list_postgres_instances`; match the checked-in names above. Do not commit Render resource IDs, API keys, connection strings, or secrets.
3. Prefer read-only inspection first: `get_service`, `list_deploys`, `get_deploy`, `list_logs`, `get_metrics`, and read-only `query_render_postgres`.
4. Treat environment updates, deploy triggers, and resource creation as production mutations. Perform them only when explicitly requested and after confirming the resource and workspace.
5. Do not trigger a routine Render deploy through MCP after pushing a production tag; use the checked-in local `scripts/appview-deploy` path. Use `trigger_deploy` only for an explicitly requested manual redeploy or rollback when the normal exact-tag path is unsuitable.

For the Render CLI, run commands from the repository root. `render blueprints validate` must report `valid: true` before a reviewed Blueprint sync. Use CLI discovery (`render --help` and subcommand help) rather than assuming command syntax, and never place Render credentials in repository files or command output. See the production runbook for secrets, DNS, post-deploy checks, rollback, PostgreSQL recovery, and Tap recovery.

## Pull Requests

Use [`.github/pull_request_template.md`](.github/pull_request_template.md) for every PR. Keep the summary short, list the tests run, call out API or lexicon impact explicitly, and include screenshots or recordings for UI changes. GitHub Actions runs path-aware tests only for pull requests. Releases, builds, tags, and production deployments are local maintainer operations documented in [`docs/operations/releases.md`](docs/operations/releases.md).

## Repository Layout

- `app/` — Flutter client. Uses [`atproto.dart`](https://github.com/myConsciousness/atproto.dart). Reads and ordinary writes use `appview/` via HTTPS + session token; Flutter never contacts a PDS directly. The sole approved external write is direct Bluesky video upload using the ephemeral service JWT constrained by ADR 012.
- `appview/` — Go App View. Consumes the atproto firehose via the Tap sidecar (WebSocket-with-acks), indexes Craftsky records into Postgres, serves a JSON/HTTP API to the app, mediates OAuth with the PDS (Token Mediating Backend).
- `lexicon/` — atproto lexicon JSON schemas under the `social.craftsky.*` namespace (matches the `craftsky.social` domain). Treat these as load-bearing: once records are written to real PDSes with a given schema, migrating is painful.
- `docker-compose.yml` — full local-dev stack: `postgres` + `migrate` + `tap` + `tap-bootstrap` + `appview`.
- `justfile` — task-runner recipes (`just dev`, `just test`, `just psql`, `just tap-status`, ...).

## Architectural Rules

1. **Record writes go through the PDS; reads come from the App View.** Never have Flutter read craft data directly from a PDS. Ordinary record/blob writes remain AppView-mediated. ADR 012 narrowly permits Flutter to upload a video directly to the configured Bluesky video service, which stores the processed blob on the user's PDS; final post creation and verification still go through AppView.
2. **OAuth credentials stay server-side.** OAuth access/refresh tokens, DPoP keys, and generic PDS credentials live only in AppView. ADR 012 permits Flutter to hold one short-lived PDS-signed service JWT in memory solely for the exact Bluesky video-upload origin/path. It must never be persisted, logged, forwarded on redirect, or reused for another operation. Any broader TMB handoff requires a separate architecture decision.
3. **Public data on PDS, private data in Postgres.** Posts, follows, blocks, likes → PDS records. Drafts, mutes, push tokens, moderation state → App View Postgres. See the reference doc's "Data Visibility & Privacy" section.
4. **Lexicon changes need an ADR.** Before modifying anything in `lexicon/`, invoke the project-level `atproto-lexicon` skill (at [`.claude/skills/atproto-lexicon/`](.claude/skills/atproto-lexicon/SKILL.md)) for NSID/type/style/evolution rules, then use `writing-architecture-decision-records` for the decision record itself.
   > **New firehose indexers** register via `dispatcher.Register(nsid, idx)` in `appview/internal/app/deps.go`. One indexer per NSID; `Handle` must be idempotent on `(URI, CID)`.
5. **No generic OAuth libraries.** atproto OAuth requires DPoP, dynamic server discovery, and client metadata. Use `indigo/atproto/auth/oauth` or `haileyok/atproto-oauth-golang`.

## Coding Conventions

- **Go:** standard `gofmt`, `slog` for logging, `sqlc` for queries (write SQL, not ORMs), stdlib `net/http` for routing (Go 1.22+ method/path routing is enough), `pgx` for Postgres.
- **atproto identifiers:** strongly prefer indigo's typed wrappers from [`github.com/bluesky-social/indigo/atproto/syntax`](https://github.com/bluesky-social/indigo/tree/main/atproto/syntax) — `syntax.DID`, `syntax.Handle`, `syntax.NSID`, `syntax.ATURI`, `syntax.RecordKey`, `syntax.CID`, `syntax.TID`, `syntax.AtIdentifier` — over plain `string` for any field, parameter, or return value that semantically carries an atproto identifier. They are zero-cost (`type X string`), implement `MarshalText`/`UnmarshalText` so JSON round-trips through `Parse*()` automatically, and pgx v5 accepts them as query parameters via its reflective fallback. **Parse at the boundary, trust internally:** the WS-decode site (`tap.Event` construction), the auth middleware (`X-Dev-DID` header), and HTTP request bodies (e.g. `loginRequest.Handle`) call `syntax.Parse*` once and hand typed values downstream — handlers, indexers, and storage helpers should not re-validate. `syntax.CID` is the one exception: indigo documents it as an "informal helper" — direct-cast it through (the real validator is `ipfs/go-cid`, applied where canonical CID handling actually matters).
- **SQL:** migrations in `appview/migrations/` via `golang-migrate/v4` (wrapped by `appview/cmd/cli migrate`). Queries in `appview/queries/` consumed by `sqlc`.
- **Commits:** conventional commits style is fine but not enforced. Keep them focused.
- **API:** The HTTP surface between the Flutter app and the AppView is governed by the API architecture spec ([`docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`](docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md)). Before adding or changing any route, read it — it fixes the `/v1/` prefix, auth headers, error envelope (`{error, message, requestId}`), opaque-cursor pagination, and URL conventions.
- **JSON casing:** Every `/v1/*` JSON body uses camelCase keys — requests, responses, and error envelopes alike. `/oauth/*` keeps whatever the atproto OAuth spec dictates; SQL column names stay snake_case (wire JSON is a separate concern from storage). See [`docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`](docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md) §1.
- **Lexicon-derived Go types:** Generated by `just lexgen` into `appview/internal/lexicon/craftsky/`. Indexers and storage code consume these as `json.Unmarshal` targets — do not hand-roll narrow record structs in indexer files. After editing anything under `lexicon/`, run `just lexgen` and commit the regenerated package alongside the schema change. See [`docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md`](docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md).

## Project Skills (`.claude/skills/`)

Project-scoped skills that Claude Code auto-discovers:

| Skill | When to use |
|---|---|
| [`atproto-lexicon`](.claude/skills/atproto-lexicon/SKILL.md) | Before designing, writing, reviewing, or changing anything under `lexicon/`. Distils the three canonical atproto lexicon docs (guide, style guide, spec) into NSID conventions, the type system, string constraints, evolution rules, and a checklist. |



## What NOT to Do

- Don't add a second serialization format. The API is JSON/HTTP to match atproto's XRPC. No protobuf/gRPC unless the whole project pivots.
- Don't persist PDS-issued tokens on the device. The ADR 012 video service JWT exception is memory-only and purpose-bound; OAuth access/refresh tokens and DPoP keys never enter Flutter.
- Don't put private-by-intent data (drafts, mutes, wishlists) on the PDS — it's all public right now.
- Don't delete data from a user's PDS for moderation, retention, cleanup, or ordinary product behavior. The App View controls what's surfaced; the PDS is the user's.
  - The sole exception is the explicit permanent CraftSky account-deletion flow approved by the authenticated owner after fresh PDS OAuth reauthentication and exact-handle confirmation. Its deletion worker may list and delete only that owner's records in registered `social.craftsky.*` record collections.
  - The exception does not permit deleting/deactivating the DID or PDS account, deleting another namespace, directly deleting blobs, exposing PDS credentials to Flutter, or adding a reusable generic PDS-delete facility. Public AppView removal still converges through the existing Tap/indexer delete handlers.
- Don't bypass the lexicon. Records written to a PDS must validate against the lexicon schema or the App View will drop them on index.
