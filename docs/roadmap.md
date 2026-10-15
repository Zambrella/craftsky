# Roadmap

This is a hand-maintained overview of implementation history and future ideas. It is not a release schedule or an assignment list. See [CONTRIBUTING.md](../CONTRIBUTING.md) to agree on scope before starting substantial work.

## Current status

The Flutter client, AppView, public record schemas, and production infrastructure are implemented for private-beta development. Code and documentation contributions are open under AGPL-3.0; the CLA remains an inactive draft.

A checked item records implementation or preparation, not independent verification of deployment, store availability, policy approval, or full public-launch readiness. The [release runbook](operations/releases.md#private-beta-and-full-public-launch) and [online-safety gap register](../online-safety/registers/compliance-gap-register.md) track the remaining launch controls. In particular, subscription and video availability depend on configuration, and draft policies still require their recorded approvals.

For a first contribution, use an issue whose scope has been accepted by a maintainer rather than selecting an unchecked roadmap idea. See the [triage process](maintainers/triage.md) for how tasks become contributor-ready.

## v1 implementation record

The original v1 scope is retained below for context. Linked designs explain the historical approach; later ADRs and current component guides capture subsequent changes.

### AppView / API

- [x] Server scaffold — [`2026-04-16-appview-server-scaffold-design.md`](superpowers/specs/2026-04-16-appview-server-scaffold-design.md)
- [x] Tap firehose integration — [`2026-04-17-tap-integration-design.md`](superpowers/specs/2026-04-17-tap-integration-design.md)
- [x] OAuth BFF (client auth, session storage) — [`2026-04-18-appview-oauth-bff-design.md`](superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [x] API architecture (REST, `/v1/` prefix, auth headers, error envelope, pagination) — [`2026-04-21-appview-api-architecture-design.md`](superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [x] Feed: `GET /v1/feed/timeline`
- [x] Profile reads, edits, and follow/unfollow; record writes are mediated by AppView.
- [x] Post and thread reads, creation, deletion, likes, and unlikes.
- [x] Notifications: `GET /v1/notifications`
- [x] Image upload — [ADR 017](../adr/017-direct-image-upload-service-token-handoff.md) defines the narrow direct-upload handoff.
- [x] Tap-backed indexing of CraftSky and shared atproto records.
- [x] Request body size limits — [`2026-06-28-appview-architecture-hardening`](changes/2026-06-28-appview-architecture-hardening/)
- [x] Cross-cutting envelope helpers + device-id middleware — [`2026-06-28-appview-architecture-hardening`](changes/2026-06-28-appview-architecture-hardening/)
- [x] Rate limiting — per-token and per-device-id — [`2026-06-28-appview-architecture-hardening`](changes/2026-06-28-appview-architecture-hardening/)
- [x] CORS policy — [`2026-06-28-appview-architecture-hardening`](changes/2026-06-28-appview-architecture-hardening/)
- [x] Success response envelope — bare v1 success bodies retained; errors stay enveloped — [`2026-06-28-appview-architecture-hardening`](changes/2026-06-28-appview-architecture-hardening/)
- [x] Observability — request logging format, request-ID propagation into downstream calls, metrics, tracing
- [x] Search for posts, projects, profiles, and hashtags.
- [x] Reposts and quote posts.
- [x] Reports and moderation endpoints.
- [x] Push notification registration — `POST /v1/notifications/devices` etc.
- [x] Durable revoked-session cleanup — [processor](../appview/internal/auth/session_cleanup_processor.go).
- [x] Blocking, muting
- [x] Save posts
- [x] Instagram follow import
- [x] Instagram post import
- [x] How to handle different languages
- [x] Schedule posts
- [x] Pinned posts
- [x] Growth metrics
- [x] Business profiles and event records.
  - [x] Featured products
  - [x] CTA (e.g. to shop)
  - [x] Business info
  - [x] Upcoming events
- [x] Make sure to handle PDS migration
- [x] Video posts (including Instagram post importer)
- [x] Pinned Indigo and Tap dependency refresh; future updates remain ongoing maintenance.
- [x] Pro and Business subscriptions (RevenueCat)
  - [x] Webhooks
  - [x] Backend tracking against accounts
- [x] Moderation warnings, standing/history, and appeals.

### Flutter app

- [x] App scaffold — [`2026-04-19-flutter-app-scaffold-design.md`](superpowers/specs/2026-04-19-flutter-app-scaffold-design.md)
- [x] i18n scaffold — [`2026-04-19-flutter-i18n-scaffold-design.md`](superpowers/specs/2026-04-19-flutter-i18n-scaffold-design.md)
- [x] Navigation scaffold — [`2026-04-19-flutter-navigation-scaffolding-design.md`](superpowers/specs/2026-04-19-flutter-navigation-scaffolding-design.md)
- [x] App initialisation tests — [`2026-04-19-app-initialisation-tests-design.md`](superpowers/specs/2026-04-19-app-initialisation-tests-design.md)
- [x] OAuth login flow (handle entry, browser handoff, deep-link return, session persistence).
- [x] Device-id generation and persistence (for `X-Craftsky-Device-Id` header)
- [x] API client layer (thin wrapper that injects auth headers, handles errors, decodes envelopes)
- [x] Feed screen (timeline consumption + pagination)
- [x] Profile screen (view + edit)
- [x] Post composer (text-only first; image attach lands with blob upload)
- [x] Post detail / thread view
- [x] Follow / unfollow interactions
- [x] Notifications screen
- [x] Error-handling UX for API failures.
- [x] Rich text — facets, mentions, links, hashtags
- [x] Quote posts
- [x] Multi-account support
- [x] Drafts
- [x] Drawer
- [x] Profile settings screen — handle change, avatar upload, privacy controls
- [x] Profile customisation
- [x] Link previews
- [x] YouTube video embeds
- [x] Tablet support
- [x] Dark theme
- [x] Registration
- [x] Welcome page
- [x] Onboarding
- [x] Limit/compress images to 2MB (Bluesky max CDN size)
- [x] Update create project flow to be more streamlined
- [x] Iconography
- [x] Be able to see who liked, reposted and quote reposted a post
- [x] App icon and splash screen
- [x] Pro and Business subscriptions (RevenueCat)
  - [x] Front-end logic
  - [x] Paywall
  - [x] Locking content
- [x] Add the correct OAuth scopes
- [x] Make sure OAuth and lexicons are broadcast to the network

### Lexicons

- [x] `social.craftsky.feed.post`, `.feed.like`, `.feed.repost`, `.actor.profile` defined
- [x] Align Lexicons with Flutter composer requirements.
- [x] Initial project-post field set; continue gathering feedback from crafters.
- [x] Initial project taxonomy and craft-specific metadata.

### Web / marketing

- [x] Public website at craftsky.social (Cloudflare Worker with static assets; see [current setup](../web/README.md)) — [`2026-04-22-landing-page-design.md`](superpowers/specs/2026-04-22-landing-page-design.md)

### Ops / infra

- [x] Production deploy (Render AppView + Tap + managed Postgres, external managed S3-compatible storage) — [ADR 016](../adr/016-render-managed-production-infrastructure.md)
- [x] Backup strategy
- [x] Monitoring / alerting for Tap connection health, firehose lag, indexer errors
- [x] App store distribution — Apple Developer + Google Play accounts, bundle/app IDs, signing keys, TestFlight + Play internal track, store listings, review submission
- [x] Secrets management for local dev — `.env` handling conventions, `.env.example`, what's safe to commit
- [x] Correct Sentry config for production AppView
- [x] Correct Sentry config for App (including automated debug symbol uploads)

### Product / community

- [x] Handle and identity approach.
- [x] Initial reporting and moderation implementation.
- [ ] Ozone moderation backend — self-hosted Ozone instance for triaging reports, labelling, and takedowns (see https://atproto.com/guides/using-ozone)
- [x] Initial online-safety assessment documents; outstanding approvals and evidence remain in the gap register.
- [x] Terms and privacy policy drafts linked from app and website; publication approval is tracked separately.

### Governance

- [x] Open-source [contribution guide](../CONTRIBUTING.md), [issue forms](../.github/ISSUE_TEMPLATE/), and [triage process](maintainers/triage.md).

---

## After v1

Ideas for future scoping, without delivery dates or a committed order. Discuss the user need and current implementation before selecting an approach.

### AppView / API

- [ ] **Active-sessions UI endpoints** — `GET /v1/auth/sessions`, `DELETE /v1/auth/sessions/{id}`. Uses `last_device_id` / `device_label` already being captured.
- [ ] **OpenAPI document + typed Dart client generation** — likely worthwhile once the API has stabilised.
- [ ] **XRPC interop surface** — expose `/xrpc/social.craftsky.*` for third-party atproto clients. Additive to the REST API.
- [ ] **Broader TMB handoff evaluation** — reassess only if a new use case requires it. Immediate image and video uploads already have narrowly scoped exceptions under ADRs 017 and 012; broader client-side credential authority needs a separate architecture decision.
- [ ] **Pre-computed feed tables (fan-out-on-write)** — only if the join-based feed hits performance limits. Reference doc says the basic approach handles thousands of users.
- [ ] **Labeller integration** — subscribe to atproto labellers for content categorisation.
- [ ] **App-layer encryption** of `oauth_sessions.data` — OAuth BFF §6.
- [ ] **Client-key rotation** that survives user sessions — OAuth BFF §6.
- [ ] **standard.site** - Add support for https://standard.site/
- [ ] **Error tracing** - Link together Flutter and AppView error reporting.
- [ ] **Editing posts** - possible? Only in first 15 mins? Upgrade a post to a project?

### Flutter app

- [ ] **Image composition** — cropping, multi-image layouts
- [ ] **Accessibility audit** — screen reader, dynamic type, contrast.

### Product

- [ ] **Web version**
- [ ] **Stash Hub integration**
- [ ] **Custom feeds**
- [ ] **Polls**
- [ ] **Visibility and interaction rules** - See Bluesky's "Post interaction settings" for an example

### Ops / infra

- [ ] **Expand PR check coverage** — path-aware AppView, Flutter, service-status, and workflow checks already exist in [GitHub Actions](../.github/workflows/backend-ci.yml). Identify remaining package and device gaps; releases and deployment remain local maintainer operations.
- [ ] **Secondary relay / fallback** — what do we do when bsky.network goes down?
- [ ] **Horizontal scaling story** — do we ever need it? If so, what does it look like?
- [ ] **Data export for users** — "download everything you've posted" — worth offering even though the data is already portable via atproto.
- [ ] **CraftSky owned CDN** - For both images and video

### Governance

- [ ] **ADR archive pruning policy** — how do we keep the `docs/` folder navigable as it grows?
