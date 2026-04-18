# AGENTS.md

Guidance for AI agents and human contributors working in this repository.

## Project Overview

Craftsky is a crafting-focused social platform built on the [AT Protocol](https://atproto.com). It is a federated, user-owned alternative to Instagram/Pinterest for the crafting community.

Read [atproto-craft-social-app-reference.md](atproto-craft-social-app-reference.md) before making architectural decisions — it captures the full design rationale (PDS vs App View split, auth flow, lexicon design, tech stack choices).

For the product-level "why" and the community-facing feature intent, read the vision doc: [Building a Social Network for Textile Crafters](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing). It's the source of truth for product principles (no ads ever, data portability, transparent business accounts, chronological feed, no algorithmic ranking) and for the project-post field set. If a technical decision conflicts with this doc, flag it — don't silently diverge.

## Dev Workflow

`just dev` (from the repo root) starts the full compose stack — `postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`. The appview runs only inside Docker in dev; there is no `go run ./cmd/appview` path. Go tests run on the host with `just test` against the compose Postgres (the appview image has no Go toolchain). See `justfile` for the full recipe list.

## Repository Layout

- `app/` — Flutter client. Uses [`atproto.dart`](https://github.com/myConsciousness/atproto.dart). Talks to `appview/` via HTTPS + session token, never to the PDS directly.
- `appview/` — Go App View. Consumes the atproto firehose via the Tap sidecar (WebSocket-with-acks), indexes Craftsky records into Postgres, serves a JSON/HTTP API to the app, mediates OAuth with the PDS (Token Mediating Backend).
- `lexicon/` — atproto lexicon JSON schemas under the `social.craftsky.*` namespace (matches the `craftsky.social` domain). Treat these as load-bearing: once records are written to real PDSes with a given schema, migrating is painful.
- `docker-compose.yml` — full local-dev stack: `postgres` + `migrate` + `tap` + `tap-bootstrap` + `appview`.
- `justfile` — task-runner recipes (`just dev`, `just test`, `just psql`, `just tap-status`, ...).

## Architectural Rules

1. **Writes go through the PDS, reads come from the App View.** Never have the Flutter app read craft data directly from a PDS in the happy path.
2. **The Flutter app never holds PDS tokens.** OAuth access/refresh tokens live in the App View's `sessions` table. The app only holds a Craftsky session token.
3. **Public data on PDS, private data in Postgres.** Posts, follows, blocks, likes → PDS records. Drafts, mutes, push tokens, moderation state → App View Postgres. See the reference doc's "Data Visibility & Privacy" section.
4. **Lexicon changes need an ADR.** Before modifying anything in `lexicon/`, invoke the project-level `atproto-lexicon` skill (at [`.claude/skills/atproto-lexicon/`](.claude/skills/atproto-lexicon/SKILL.md)) for NSID/type/style/evolution rules, then use `writing-architecture-decision-records` for the decision record itself.
5. **No generic OAuth libraries.** atproto OAuth requires DPoP, dynamic server discovery, and client metadata. Use `indigo/atproto/auth/oauth` or `haileyok/atproto-oauth-golang`.

## Coding Conventions

- **Go:** standard `gofmt`, `slog` for logging, `sqlc` for queries (write SQL, not ORMs), stdlib `net/http` for routing (Go 1.22+ method/path routing is enough), `pgx` for Postgres.
- **Dart/Flutter:** `dart format`, follow `flutter_lints`. Prefer the `atproto.dart` SDK over hand-rolled XRPC calls. Additional rules that apply to **all `**/*.dart` files** live in [`.claude/rules/flutter.md`](.claude/rules/flutter.md) and [`.claude/rules/riverpod.md`](.claude/rules/riverpod.md) — read and follow both before writing Dart.
- **SQL:** migrations in `appview/migrations/` via `golang-migrate/v4` (wrapped by `appview/cmd/cli migrate`). Queries in `appview/queries/` consumed by `sqlc`.
- **Commits:** conventional commits style is fine but not enforced. Keep them focused.

## Project Skills (`.claude/skills/`)

Project-scoped skills that Claude Code auto-discovers:

| Skill | When to use |
|---|---|
| [`atproto-lexicon`](.claude/skills/atproto-lexicon/SKILL.md) | Before designing, writing, reviewing, or changing anything under `lexicon/`. Distils the three canonical atproto lexicon docs (guide, style guide, spec) into NSID conventions, the type system, string constraints, evolution rules, and a checklist. |

## Language-Specific Rules (`.claude/rules/`)

The [`.claude/rules/`](.claude/rules/) directory holds rule files that apply to specific file globs. `CLAUDE.md` imports them with `@` so Claude Code auto-loads them for every session in this repo. Other agents and human contributors should follow them when working on matching files:

| File | Applies to | Summary |
|---|---|---|
| [`.claude/rules/flutter.md`](.claude/rules/flutter.md) | `**/*.dart` | Widget architecture (one class per widget, no `_build*` helpers), theming (no `.withOpacity`), immutable data via `freezed`/`dart_mappable`, modern Dart syntax, logging package over `print`. |
| [`.claude/rules/riverpod.md`](.claude/rules/riverpod.md) | `**/*.dart` | Riverpod 3.x with `@riverpod` code generation. `ref.watch` in build / `ref.read` in callbacks, `FutureOr` for idle providers, `AsyncValue.guard`, switch-based pattern matching (no `.when()`), `ref.listen` for side effects, `ref.mounted` after awaits. |

These are binding for Dart code. Each file carries a YAML `paths:` frontmatter so rule-aware tools can auto-scope them.

## What NOT to Do

- Don't add a second serialization format. The API is JSON/HTTP to match atproto's XRPC. No protobuf/gRPC unless the whole project pivots.
- Don't store PDS tokens on the device.
- Don't put private-by-intent data (drafts, mutes, wishlists) on the PDS — it's all public right now.
- Don't try to delete data from a user's PDS. The App View controls what's surfaced; the PDS is the user's.
- Don't bypass the lexicon. Records written to a PDS must validate against the lexicon schema or the App View will drop them on index.
