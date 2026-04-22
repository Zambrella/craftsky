# Roadmap

High-level, whole-project view of what's done, what's in flight, and what's known-but-not-yet-scoped. Items marked "→ own spec" need a design spec of their own before implementation; links go to specs that exist.

This doc is hand-maintained. When you finish something, move it under "Done." When you think of a new thing, drop it under the relevant heading. Don't wait until it's fully scoped — the point is to capture the universe of open items so nothing rots silently.

---

## v1

The minimum we need to ship a first usable Craftsky Flutter app with a real AppView backing it.

### AppView / API

- [x] Server scaffold — [`2026-04-16-appview-server-scaffold-design.md`](superpowers/specs/2026-04-16-appview-server-scaffold-design.md)
- [x] Tap firehose integration — [`2026-04-17-tap-integration-design.md`](superpowers/specs/2026-04-17-tap-integration-design.md)
- [x] OAuth BFF (client auth, session storage) — [`2026-04-18-appview-oauth-bff-design.md`](superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [x] API architecture (REST, `/v1/` prefix, auth headers, error envelope, pagination) — [`2026-04-21-appview-api-architecture-design.md`](superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [ ] Feed: `GET /v1/feed/timeline` → own spec
- [ ] Profiles: `GET /v1/profiles/@{handleOrDid}`, `GET /v1/profiles/@{handleOrDid}/posts`, `PATCH /v1/profiles/me`, `PUT /v1/profiles/me`, follow/unfollow → own spec
- [ ] Posts: `GET /v1/posts/{did}/{rkey}`, thread, create, delete, like, unlike → own spec (splits into reads / writes if it gets big)
- [ ] Notifications: `GET /v1/notifications` → own spec
- [ ] Write proxy: DPoP-signed path from authenticated handler to user's PDS, shared by all write endpoints → own spec (blocker for any write endpoint)
- [ ] Blob upload (required for images on posts) → own spec; may trigger TMB upgrade per OAuth BFF §6
- [ ] Indexer: consume Tap → populate `posts`, `follows`, `likes`, `blocks` tables for Craftsky lexicons → own spec (replaces the `000001_bluesky_posts_sample` placeholder)
- [ ] Request body size limits → own spec (small, could be folded into another)
- [ ] Cross-cutting envelope helpers + device-id middleware (part of API architecture implementation, not its own spec)

### Flutter app

- [x] App scaffold — [`2026-04-19-flutter-app-scaffold-design.md`](superpowers/specs/2026-04-19-flutter-app-scaffold-design.md)
- [x] i18n scaffold — [`2026-04-19-flutter-i18n-scaffold-design.md`](superpowers/specs/2026-04-19-flutter-i18n-scaffold-design.md)
- [x] Navigation scaffold — [`2026-04-19-flutter-navigation-scaffolding-design.md`](superpowers/specs/2026-04-19-flutter-navigation-scaffolding-design.md)
- [x] App initialisation tests — [`2026-04-19-app-initialisation-tests-design.md`](superpowers/specs/2026-04-19-app-initialisation-tests-design.md)
- [ ] OAuth login flow (handle entry, browser handoff, deep-link return, session persistence) → own spec
- [ ] Device-id generation and persistence (for `X-Craftsky-Device-Id` header)
- [ ] API client layer (thin wrapper that injects auth headers, handles errors, decodes envelopes)
- [ ] Feed screen (timeline consumption + pagination)
- [ ] Profile screen (view + edit)
- [ ] Post composer (text-only first; image attach lands with blob upload)
- [ ] Post detail / thread view
- [ ] Follow / unfollow interactions
- [ ] Notifications screen
- [ ] Error-handling UX (how do we surface `error` codes from the envelope to users?)

### Lexicons

- [x] `social.craftsky.feed.post`, `.feed.like`, `.feed.repost`, `.actor.profile` defined
- [ ] Verify lexicons against real Flutter composer requirements before first real PDS writes happen — any missing/wrong field now is painful to fix later

### Web / marketing

- [ ] Landing page at craftsky.social (hero + 8 sections, Cloudflare Pages, anonymous PostHog) — [`2026-04-22-landing-page-design.md`](superpowers/specs/2026-04-22-landing-page-design.md)

### Ops / infra

- [ ] Production deploy (Hetzner VPS + Docker Compose + Caddy) → own spec
- [ ] Client private key management for OAuth in prod (env var vs file vs KMS) — OAuth BFF §5.1 open question
- [ ] Backup strategy for Postgres → own spec
- [ ] Monitoring / alerting for Tap connection health, firehose lag, indexer errors

### Product / community

- [ ] Handle suffix decision (users get `<name>.craftsky.social`? Some other domain? Use bsky.social handles?)
- [ ] Initial moderation plan — even MVP needs a "report a post" path, even if the backend is just "email an inbox"
- [ ] Legal read (UK Online Safety Act implications per reference doc)

---

## After v1

Scoped but not urgent. Ordered roughly by expected sequence, not strictly prioritised.

### AppView / API

- [ ] **Rate limiting** — per-token and per-device-id. Needed before public launch.
- [ ] **CORS policy** — only becomes relevant if/when a web client is scoped.
- [ ] **Success response envelope** — decide whether to wrap successful responses in `{"data": ...}`. Bumps to `/v2/` if we change it.
- [ ] **Observability** — request logging format, request-ID propagation into downstream calls, metrics, tracing.
- [ ] **Search** — posts by text, tag, craft type, materials. Separate service or Postgres FTS? Separate spec either way.
- [ ] **Reposts** — `POST /v1/posts/{did}/{rkey}/reposts` etc. Lexicon already defined.
- [ ] **Blocks, mutes, reports** — moderation endpoints.
- [ ] **Push notification registration** — `POST /v1/notifications/devices` etc.
- [ ] **Active-sessions UI endpoints** — `GET /v1/auth/sessions`, `DELETE /v1/auth/sessions/{id}`. Uses `last_device_id` / `device_label` already being captured.
- [ ] **OpenAPI document + typed Dart client generation** — likely worthwhile once the API has stabilised.
- [ ] **XRPC interop surface** — expose `/xrpc/social.craftsky.*` for third-party atproto clients. Additive to the REST API.
- [ ] **TMB upgrade** — `/auth/session/exchange` + `/auth/session/refresh` so the client can make DPoP-signed calls directly. Primary motivator: avoid proxying blob uploads. OAuth BFF §6.
- [ ] **Pre-computed feed tables (fan-out-on-write)** — only if the join-based feed hits performance limits. Reference doc says the basic approach handles thousands of users.
- [ ] **Labeller integration** — subscribe to atproto labellers for content categorisation.
- [ ] **Sweeper process** for revoked Craftsky sessions — replaces lazy cleanup at scale.
- [ ] **App-layer encryption** of `oauth_sessions.data` — OAuth BFF §6.
- [ ] **Client-key rotation** that survives user sessions — OAuth BFF §6.

### Flutter app

- [ ] **Image composition** — cropping, multi-image layouts, camera roll picker.
- [ ] **Drafts** — private, server-side (per AGENTS.md rule #3).
- [ ] **Rich text** — facets, mentions, links, hashtags.
- [ ] **Quote posts**
- [ ] **Profile settings screen** — handle change, avatar upload, privacy controls.
- [ ] **Accessibility audit** — screen reader, dynamic type, contrast.
- [ ] **Offline / retry behaviour** — optimistic writes that reconcile when the firehose catches up.

### Lexicons

- [ ] **Project-post field set validation with real crafters** — the reference doc flags this as the most important early decision; first real users will tell us what's missing.
- [ ] **Pattern/material/technique taxonomy** — controlled vocabularies or free-form? If controlled, who maintains them?

### Product / community

- [ ] **Web viewer for SEO / discoverability** — reference doc calls out "how to sew a French seam" as a growth channel.
- [ ] **Stash Hub integration** — existing user base seed.
- [ ] **Business accounts** — transparent-business-account requirement from vision doc.
- [ ] **Monetisation model** — premium features, Stash Hub tie-in.

### Ops / infra

- [ ] **Secondary relay / fallback** — what do we do when bsky.network goes down?
- [ ] **Horizontal scaling story** — do we ever need it? If so, what does it look like?
- [ ] **Data export for users** — "download everything you've posted" — worth offering even though the data is already portable via atproto.

### Governance

- [ ] **Open-source contribution guide** — CONTRIBUTING.md, issue templates, triage process.
- [ ] **ADR archive pruning policy** — how do we keep the `docs/` folder navigable as it grows?
