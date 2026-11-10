# ADR 024: Public server-rendered CraftSky web pages

Date: 2026-11-02
Status: Accepted (maintainer confirmed scope and host)

## Context

Shared CraftSky projects, posts and profiles need useful initial HTML, search
indexing and page-specific link previews. A client-rendered bootstrap cannot provide
that document experience. The maintainer selected `app.craftsky.social` so the
same links can open installed mobile apps. This supersedes earlier proposals
for `share.craftsky.social` and the vision document's original client stack
choice; the product principles remain unchanged.

## Decision

Add a SvelteKit/TypeScript client in `web-client/`, deployed to the existing
Cloudflare Pages host that currently serves `verified-links/`. Keep the marketing
site separate. Render this read-only slice entirely on the server with no
hydration JavaScript. Use normal links for navigation and pagination.

The Go AppView retains all data access and policy. Four separate anonymous GET
endpoints under `/v1/public/` serve explicit public DTOs. Existing member routes
retain their authorization. Anonymous responses ignore cookies and bearer tokens,
exclude all viewer/private state, require active CraftSky membership, and exclude
suspended, hidden, taken-down and warning-labelled accounts/content. Warning
content is withheld because this slice has no informed-reveal flow. Safety-cleared
post media use the existing media presentation; unsafe profile images are omitted.
Business and sponsored labels remain visible without exposing subscription data.

Author feeds contain chronological root posts or projects, with bounded keyset
pagination and filtering before the limit. Replies, reactions, repost feeds,
quote expansion, search, sign-in and mutations are outside this slice.

Preserve the exact OAuth fallback documents and Android association document;
extend the iOS association document to `/posts/*` and `/profiles/*`. Authentication
capabilities are never reflected into HTML, metadata, logs or referrers.

Do not cache anonymous JSON or HTML. Fresh requests re-evaluate current policy.
Third-party link-preview caches cannot be invalidated by these response headers.
Canonical URLs always use the controlled production host and stable DID/rkey
identifiers. Preview hosts and error responses are marked non-indexable.

Anonymous work has independent aggregate admission limits (20 requests/second,
burst 40, maximum eight in flight) because the member limiter's device/session
keys are absent. Database work has an eight-second request deadline. Per-visitor
edge abuse controls can be configured separately without trusting spoofable IDs.

## Consequences

Mobile UI remains Flutter. Web gains a separate presentation implementation and
Node dependency/test lifecycle, while business rules and credentials stay in Go.
The initial client needs no schema migration or lexicon change. Expanding into a
signed-in client requires an explicit browser-session design, not copying mobile
credentials into this frontend. Shared caches require a future invalidation design.

Deployment needs a normal AppView release followed by updating the existing
verified-link Pages project's build settings. No deployment or DNS changes are
performed by this implementation. Device association and third-party share-card
checks remain release checks, not conclusions drawn from local tests.
