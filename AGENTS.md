# AGENTS.md

Guidance for AI agents and human contributors working in this repository.

## Project Overview

**The app is not in production with no active users so breaking changes are not a thing.**

Craftsky is a crafting-focused social platform built on the [AT Protocol](https://atproto.com). It is a federated, user-owned alternative to Instagram/Pinterest for the crafting community.

Read [atproto-craft-social-app-reference.md](atproto-craft-social-app-reference.md) before making architectural decisions — it captures the full design rationale (PDS vs App View split, auth flow, lexicon design, tech stack choices).

For the product-level "why" and the community-facing feature intent, read the vision doc: [Building a Social Network for Textile Crafters](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing). It's the source of truth for product principles (no ads ever, data portability, transparent business accounts, chronological feed, no algorithmic ranking) and for the project-post field set. If a technical decision conflicts with this doc, flag it — don't silently diverge.

## Dev Workflow

`just dev` (from the repo root) starts the full compose stack — `postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`. The appview runs only inside Docker in dev; there is no `go run ./cmd/appview` path. Go tests run on the host with `just test` against the compose Postgres (the appview image has no Go toolchain). See `justfile` for the full recipe list.

## Pull Requests

Use [`.github/pull_request_template.md`](.github/pull_request_template.md) for every PR. Keep the summary short, list the tests run, call out API or lexicon impact explicitly, and include screenshots or recordings for UI changes.

## Repository Layout

- `app/` — Flutter client. Uses [`atproto.dart`](https://github.com/myConsciousness/atproto.dart). Talks to `appview/` via HTTPS + session token, never to the PDS directly.
- `appview/` — Go App View. Consumes the atproto firehose via the Tap sidecar (WebSocket-with-acks), indexes Craftsky records into Postgres, serves a JSON/HTTP API to the app, mediates OAuth with the PDS (Token Mediating Backend).
- `lexicon/` — atproto lexicon JSON schemas under the `social.craftsky.*` namespace (matches the `craftsky.social` domain). Treat these as load-bearing: once records are written to real PDSes with a given schema, migrating is painful.
- `docker-compose.yml` — full local-dev stack: `postgres` + `migrate` + `tap` + `tap-bootstrap` + `appview`.
- `justfile` — task-runner recipes (`just dev`, `just test`, `just psql`, `just tap-status`, ...).

## Architectural Rules

1. **Writes go through the PDS, reads come from the App View.** Never have the Flutter app read craft data directly from a PDS in the happy path.
2. **The Flutter app never holds PDS tokens.** OAuth access/refresh tokens live in the App View's `sessions` table. The app only holds a Craftsky session token.
   > **Note on the TMB upgrade path:** the current BFF design is consistent with this rule as written. Upgrading to the Token-Mediating Backend pattern (tracked as future work in the OAuth spec) will require amending this rule to distinguish *refresh* tokens (server-only) from short-lived access tokens + DPoP keys (may be handed down to clients).
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
- Don't store PDS tokens on the device.
- Don't put private-by-intent data (drafts, mutes, wishlists) on the PDS — it's all public right now.
- Don't try to delete data from a user's PDS. The App View controls what's surfaced; the PDS is the user's.
- Don't bypass the lexicon. Records written to a PDS must validate against the lexicon schema or the App View will drop them on index.
