# Craftsky

A crafting-focused social platform built on the [AT Protocol](https://atproto.com).

Craftsky lets makers — sewers, knitters, quilters, woodworkers, and others — share projects, techniques, and works-in-progress on a federated, user-owned protocol.

Domain: **craftsky.social** (app at **app.craftsky.social**). atproto namespace: `social.craftsky.*`.

## Contributing

Craftsky isn't ready for outside code contributions yet — the architecture is still settling and I'm not set up to review PRs. Please don't open one for now; it'll likely sit unmerged.

That said, **discussion is very welcome**. If you have ideas, questions, feedback on the [product vision doc](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing), or want to flag something you'd like to see, please open a [GitHub Discussion](../../discussions) or an issue. I'll update this section once the project is ready for code contributions.

## Architecture

Craftsky follows the standard atproto App View pattern:

```
User's PDS  →  Relay (bsky.network)  →  Craftsky App View  →  Flutter client
   (writes)        (firehose)              (Go + Postgres)       (reads)
```

- **Writes** go through the user's PDS using the Craftsky lexicon.
- **Reads** come from the Craftsky App View, which subscribes to the firehose and indexes relevant records.
- Identity, auth, and data portability are handled by the atproto layer.

See [atproto-craft-social-app-reference.md](atproto-craft-social-app-reference.md) for the full architectural reference.

## Product Vision

The community-facing vision doc — what we're building, what we believe in, and the open questions we want crafters to weigh in on — lives here:

**[Building a Social Network for Textile Crafters](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing)**

It's an open, commentable doc covering the "why", the core post types (simple + structured project posts), principles (no ads, data portability, transparent business accounts), and specific questions the community is helping shape — including which project fields to capture and how monetisation should work.

## Repository Layout

| Directory | Purpose |
|---|---|
| [`app/`](app/) | Flutter client (Dart + `atproto.dart` SDK) |
| [`appview/`](appview/) | Go App View — firehose indexer, HTTP API, OAuth TMB |
| [`lexicon/`](lexicon/) | Craftsky atproto lexicon schemas (`social.craftsky.*`) |

## Tech Stack

- **Client:** Flutter + Dart, using [atproto.dart](https://atprotodart.com)
- **App View:** Go, using [indigo](https://github.com/bluesky-social/indigo), `pgx`, `sqlc`, `chi`
- **Database:** Postgres 16
- **Infrastructure:** Hetzner VPS + Docker Compose + Caddy
- **Push:** FCM

## Getting started

Prerequisites:

- Docker (with Docker Compose v2)
- [`just`](https://just.systems) (`brew install just` on macOS)

Clone and run:

```
git clone <repo>
cd craftsky
just dev
```

This brings up the full compose stack — `postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`. On a cold start allow ~60s for the tap sidecar to finish replaying. Then verify:

```
curl localhost:8080/healthz   # expect {"status":"ok",...} with tap.connected: true
just tap-status               # prints tap connection state from the CLI
just psql                     # psql shell; try: SELECT count(*) FROM bluesky_posts_sample;
```

See [`appview/README.md`](appview/README.md) for the full list of `just` recipes and the host-side test workflow.

## Worktrees

Use the repository helper to create local worktrees in the standard ignored location:

```
just worktree feature/my-change
cd .worktrees/feature-my-change
```

The helper creates a new local branch from `main` by default, fetches `origin/main` first, and refuses to overwrite an existing branch or path. Pass extra options through `just` when needed:

```
just worktree fix/api-cursor --base origin/main
just worktree threads-ux --path .worktrees/threads-ux
```

Clean up stale worktree metadata and empty directories with:

```
just worktree-cleanup
just worktree-cleanup --dry-run
```

Each subdirectory also has its own README:

- [`app/README.md`](app/README.md) — Flutter app setup
- [`appview/README.md`](appview/README.md) — Go App View setup
- [`lexicon/README.md`](lexicon/README.md) — Lexicon schema authoring

## Status

Early scaffolding. See [AGENTS.md](AGENTS.md) for agent/contributor guidance.

## License

Craftsky is licensed under the [GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0).

Copyright (C) 2026 Doug Todd.

In short: you're free to use, study, modify, and redistribute the code, including running it as a service. If you run a modified version as a network service, AGPL requires you to offer the source of your modifications to your users. See the [LICENSE](LICENSE) file for the full terms.

If/when external code contributions open up, contributors will be asked to sign a Contributor License Agreement — see [CLA.md](CLA.md) for the current draft.
