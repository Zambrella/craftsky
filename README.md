# Craftsky

A crafting-focused social platform built on the [AT Protocol](https://atproto.com).

Craftsky lets makers — sewers, knitters, quilters, woodworkers, and others — share projects, techniques, and works-in-progress on a federated, user-owned protocol.

Domain: **craftsky.social** (app at **app.craftsky.social**). atproto namespace: `social.craftsky.*`.

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

## Getting Started

Each subdirectory has its own README with setup instructions:

- [`app/README.md`](app/README.md) — Flutter app setup
- [`appview/README.md`](appview/README.md) — Go App View setup
- [`lexicon/README.md`](lexicon/README.md) — Lexicon schema authoring

## Status

Early scaffolding. See [AGENTS.md](AGENTS.md) for agent/contributor guidance.
