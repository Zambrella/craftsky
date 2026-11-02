# CraftSky public web client

Server-rendered SvelteKit 3 / Svelte 5 / TypeScript pages for
`https://app.craftsky.social`. The marketing site stays in `web/`. The mobile app
continues to use Flutter. See [ADR 024](../adr/024-public-server-rendered-web-client.md).

## Local development and checks

Use Node 24 (minimum 22.17) and Docker. From the repository root:

```sh
just dev-d
just public-app-dev
```

Start the backend separately with `just dev-d`, as for `just app-run`.
`just public-app-dev` resolves the running worktree's published AppView port
through `scripts/compose-dev`, installs
frontend dependencies if absent, and starts Vite on a stable port derived from the
checkout path. The command prints both URLs. No frontend `.env` file is needed,
and backend secrets are not sourced into the frontend process. An inherited or
frontend `.env` `APPVIEW_ORIGIN` cannot override the selected worktree backend.

Use `just public-app-dev --open` to also open the browser. Set
`CRAFTSKY_PUBLIC_APP_PORT` in the shell or the root `.env.local` if the web port is
already occupied. `--strictPort` prevents silently moving to another worktree's
port. Ctrl-C stops the web client; `just down` stops the worktree's Docker stack.

`just public-app-setup` installs frontend and browser-test dependencies;
`just public-app-check` runs the frontend checks. For standalone frontend work,
`npm run dev` from this directory still accepts an explicit `APPVIEW_ORIGIN`.

From `web-client/`:

```sh
npm run check
npm test
npm run build
npx playwright install chromium
npm run test:e2e
WEB_CLIENT_WORKER=1 npm run test:e2e
```

Browser tests run both the production SSR build and, with `WEB_CLIENT_WORKER=1`,
the actual Cloudflare Worker artifact against a separate local fixture
AppView. They check metadata in initial HTML, escaping, unsafe URL removal,
profile/post navigation, pagination with JavaScript disabled, mobile overflow,
keyboard navigation, automated accessibility, error statuses, and byte-for-byte
preservation of callback and association documents. They do not prove live
AppView integration, actual search indexing, external share-card caches, real
video playback, or device app-link resolution.

Backend tests live in `appview/internal/api/public_pages_test.go`. They exercise
visibility/pagination against PostgreSQL, including the complete production
migration history. With this worktree's PostgreSQL running, use the repository's
focused test wrapper (it also expects the local MinIO service):

```sh
../scripts/appview-test focused ./internal/api -run TestPublicPage
```

Alternatively set `TEST_DATABASE_URL` explicitly to a development/test database
and run `GOTOOLCHAIN=go1.27.3 go -C ../appview test ./internal/api -run TestPublicPage`.
Never point tests at production. Unset database configuration skips integration
tests; that is not database validation.

## Visual reference

The web theme mirrors Flutter's `app/lib/theme/app_theme.dart`,
`brand_colors.dart`, and `theme_extensions.dart`: Outfit typography, DM Serif
project-summary titles, cream/charcoal paper, cobalt accents, 14px card corners,
1.5px ink rules, and 3px hard shadows. Fonts and craft glyphs are copied from
bundled app assets with their font licences, served locally with no third-party
font requests. Profile headers use the default cobalt theme, overlapping 96px
avatars, centred identity and craft chips; published banners are retained.
Per-profile custom colour/texture settings are not yet in the anonymous API.

Browser system colour preferences select the Flutter light/dark palettes.
Motion honours `prefers-reduced-motion`. Native scrolling and focusable image
links provide a photo carousel without client-side JavaScript. Post timestamps
follow Flutter's seven-day compact display, with full UTC timestamps available
to assistive technology and native tooltips. Profile navigation remains the
existing two-tab public surface; no unavailable mobile actions are added.

## Public contract

- `GET /v1/public/profiles/{handleOrDid}`: public profile, links and crafts.
- `GET /v1/public/profiles/{handleOrDid}/posts?cursor=...`: root authored posts.
- `GET /v1/public/profiles/{handleOrDid}/projects?cursor=...`: root authored projects.
- `GET /v1/public/posts/{did}/{rkey}`: root post/project detail.

Lists return `{items, cursor?}`, fixed at 20 items. Cursors are opaque. API errors
use the standard `{error, message, requestId}` envelope. Unknown/ineligible content
returns 404; invalid identifiers/cursors return 400; exhausted admission budgets
return 429/503 with Retry-After; unavailable dependencies return 503. No device ID
or authentication is required. No authenticated endpoint was made anonymous.

The frontend forwards neither incoming credentials nor arbitrary headers. It
contacts only the configured AppView origin, disallows redirects, bounds upstream
responses to 2 MiB, and times requests out. Public response DTOs never contain
viewer state, editable links, drafts, schedules, tokens or subscription details.
There is no direct PDS access. Text is escaped, external links accept HTTPS only,
and outbound user links are marked `ugc nofollow noreferrer`.

HTML and JSON are `no-store`; page metadata uses eligible content only. The
Cloudflare configuration must not override these with a cache-everything rule.
Search engines and social networks control their own copies of public content.
This first version has no content directory/sitemap; crawlers discover pages
through shared links and profile pagination. Cursor pages are `noindex,follow`.

Images retain authored alt text and dimensions where provided. Video uses the
existing public HLS playlist with native browser controls and a direct playlist
link. Browsers without native HLS need a compatible player; captions and expanded
quote/comment rendering are not part of this slice.

## Dependency audit

The locked dependency audit on 2026-11-02 reported four low-severity entries
through `deepmerge` 4.3.1, used by the Vite Svelte plugin for build configuration.
The registry had no newer stable `deepmerge` release. Public request data is not
passed through that build-time merge. Recheck the advisory when upgrading; do
not apply the audit tool's proposed downgrade to pre-release-era SvelteKit.

## Production deployment and rollback

This replaces the static verified-link host, **not** the marketing Worker or an
AppView Render service. No production settings have been changed here.

1. Review and release AppView through the existing exact-tag workflow in
   [releases](../docs/operations/releases.md). Confirm the four public reads work
   anonymously, and hidden/deleted/unsafe fixtures are not returned.
2. Identify the existing Cloudflare Pages project owning `app.craftsky.social`.
   Record its current deployment ID and settings for rollback. Do not create a
   second project, move the custom domain, or alter DNS as an incidental step.
3. Change that project's root directory from `verified-links` to `web-client`,
   build command to `npm ci && npm run build`, and output directory to
   `.svelte-kit/cloudflare`. Build from the reviewed repository commit with Node 24.
   Set runtime `APPVIEW_ORIGIN=https://appview.craftsky.social` in production and
   to an explicitly reviewed nonproduction backend for previews. The Wrangler
   name is a local artifact label; confirm the existing project name before any
   CLI publication and pass it explicitly with `--project-name`.
4. Upload a preview deployment for review before promoting it to the existing
   custom domain. Preview hosts get `X-Robots-Tag: noindex, nofollow`. Verify the
   Worker artifact, not just the Vite development server. Keep source maps private.
5. Confirm both association files return JSON directly with HTTP 200, no redirect;
   callbacks remain no-store/no-referrer with restrictive CSP and no reflected
   query values. Test ordinary and project posts, profile tabs, pagination, 404s,
   a hidden record and a removed account. Ensure error pages are not cached.
6. Verify shared links on release-signed Android/iOS devices with and without the
   app installed. iOS/browser same-site navigation rules can keep a link in the
   browser; association files alone cannot prove app opening. Check real social
   preview fetchers and search tooling separately.
7. Monitor backend latency and 429/503 rates. Anonymous admission limits are shared
   across the singleton AppView, independent of optional client IDs. Add reviewed
   per-visitor controls at the edge if needed; do not weaken backend bounds or
   blindly trust forwarded IP headers to accommodate traffic.

Rollback: restore the previous Pages deployment and its static-root build settings.
The additive AppView endpoints can remain deployed. Recheck association files and
both callbacks after rollback. Keep the existing marketing deployment untouched.
