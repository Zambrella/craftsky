# CraftSky web client

SvelteKit 3 / Svelte 5 / TypeScript client and server-rendered public pages for
`https://app.craftsky.social`. The marketing site stays in `web/`. The mobile app
continues to use Flutter. The signed-in application lives at `/app/home`.
See [ADR 024](../adr/024-public-server-rendered-web-client.md) and
[ADR 025](../adr/025-browser-consumer-handoff.md).

## Local development and checks

Use Node 24 (minimum 22.17) and Docker. From the repository root:

```sh
just dev-d
just web-run
```

Start the backend separately with `just dev-d`, as for `just app-run`.
`just web-run` (also available as `just public-app-dev`) resolves the running worktree's published AppView port
through `scripts/compose-dev`, installs
frontend dependencies if absent, and starts Vite on a stable port derived from the
checkout path. The command prints both URLs. No frontend `.env` file is needed,
and backend secrets are not sourced into the frontend process. An inherited or
frontend `.env` `APPVIEW_ORIGIN` cannot override the selected worktree backend.

Use `just public-app-dev --open` to also open the browser. Set
`CRAFTSKY_PUBLIC_APP_PORT` in the shell or the root `.env.local` if the web port is
already occupied. `--strictPort` prevents silently moving to another worktree's
port. Ctrl-C stops the web client; `just down` stops the worktree's Docker stack.

`just web-client-setup` installs frontend and browser-test dependencies;
`just web-client-check` runs the frontend checks. The earlier `public-app-*` recipes
remain aliases. For standalone frontend work,
`npm run dev` from this directory still accepts an explicit `APPVIEW_ORIGIN`.

From `web-client/`:

```sh
npm run check
npm test
npm run build
npx playwright install chromium
npm run test:e2e
npm run test:e2e:app
WEB_CLIENT_WORKER=1 npm run test:e2e
WEB_CLIENT_WORKER=1 npm run test:e2e:app
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

See [the feature-parity and final validation notes](../docs/development/web-client-parity.md)
for the implemented scope and executed checks.

## Signed-in application

Open `/sign-in` to sign in with an AT Protocol handle or create an account.
Browser OAuth uses its own callback and leaves the verified mobile app callbacks
intact. AppView retains OAuth tokens and DPoP keys. The BFF keeps only opaque
CraftSky child sessions in HttpOnly host-only cookies; JavaScript receives account
IDs, DIDs and handles. Every authenticated API operation binds to its captured
account ID. Switching accounts remounts the workspace, 401 removes only that
account, and temporary failures retain sessions. Unsafe BFF requests require an
exact same-origin Origin header. The API proxy has an explicit consumer-route
allowlist and never forwards browser-supplied bearer tokens or cookies.

The client includes chronological following and craft feeds; project filters;
people, post, project and hashtag discovery; rich posts and nested comments;
likes, reposts, quotes, reports and saves; profile editing, links and customisation;
image crop/rotate/alt text and HLS video with captions; new posts/projects,
replies, quotes and edits; private local drafts and scheduled posts; conversations
and requests; activity; notification/language/privacy preferences; account
standing and reauthenticated deletion; business profiles/products/events;
existing subscription access and licence assignment; and Instagram verification,
following-file import and community suggestions.

Purchases remain in the iOS/Android apps. The browser does not run a RevenueCat
checkout or expose a purchase action. Backend subscription access remains the
authority for paid features. Feature availability also respects the existing
backend configuration for messaging, billing, Instagram and link previews.

Drafts and media live in browser IndexedDB, partitioned by owner DID. Private
scheduled media uses the existing private AppView storage flow. A pending publish
retains its original operation UUID, exact payload and observed CID across
reloads. Uncertain writes offer an explicit retry of the same action. They do not
automatically issue a new publish. Media service JWTs use only the existing
purpose-bound ADR 012/017 exceptions and are never persisted.

Authenticated browser tests use an isolated stateful AppView double; they never
publish to a real PDS. They exercise writes through the actual cookie BFF, editor
CAS conflicts, ambiguous-write recovery, draft/media reloads and account
partitioning, scheduling, social actions, settings, business tools, licence
assignment, Instagram workflows, narrow layouts and accessibility. The same
suite runs against the production Node preview and Cloudflare Worker artifact.
Local provider mocks do not prove live OAuth consent, external message delivery,
Firebase receipt, production configuration or device app-link behaviour.

## Optional browser error reporting

Set `PUBLIC_SENTRY_DSN` at build time to make optional error reporting available.
It stays off until the device owner opts in during onboarding or Settings →
Privacy. Disabling consent closes the SDK and blocks queued exports. Only native
exception kinds and selected application-code stack coordinates are sent; API,
offline and cancellation failures are excluded. Default request instrumentation,
breadcrumbs, traces, replay, logs, attachments, private prose, URLs and viewer
context are disabled or removed again at the transport boundary.
Serialized-SDK tests verify protected-data canaries. Production delivery and
release symbolication require separate deployment validation.

## Browser notifications

Browser push uses the existing server-side Firebase delivery pipeline. It is
optional and unavailable until all required public configuration below is set at
**build time**. No Firebase service-account key belongs in frontend variables.
The VAPID value is the public web-push key from the same Firebase project.

```dotenv
PUBLIC_FIREBASE_API_KEY=
PUBLIC_FIREBASE_APP_ID=
PUBLIC_FIREBASE_PROJECT_ID=
PUBLIC_FIREBASE_MESSAGING_SENDER_ID=
PUBLIC_FIREBASE_VAPID_KEY=
PUBLIC_FIREBASE_AUTH_DOMAIN=
```

Notifications require HTTPS (or local loopback), browser support, explicit
permission from Settings → Notifications, and a working AppView Firebase sender.
iOS/iPadOS requires adding the site to the Home Screen. The bundled service
worker registers only when configured/enabled, has no fetch handler, and never
caches authenticated pages, messages or API responses. Notification routing stores
only opaque account/subscription identifiers in the browser. Chat opens resolve a
private opaque reference through the authenticated API. Disabling one account's
notifications retains other registered accounts.

No production Firebase configuration, Render change, web publication or external
notification delivery is performed by the implementation or automated tests.

## Visual reference

The web theme mirrors Flutter's `app/lib/theme/app_theme.dart`,
`brand_colors.dart`, and `theme_extensions.dart`: Outfit typography, DM Serif
project-summary titles, cream/charcoal paper, cobalt accents, 14px card corners,
1.5px ink rules, and 3px hard shadows. Fonts and craft glyphs are copied from
bundled app assets with their font licences, served locally with no third-party
font requests. Profile headers use the default cobalt theme, overlapping 96px
avatars, centred identity and craft chips; published banners are retained.
Per-profile custom colour/texture settings are not yet in the anonymous API.

Browser system colour preferences select the Flutter light/dark palettes by default;
Settings can select an explicit light or dark theme. Signed-in profiles also
respect the published colour and texture selection.
Motion honours `prefers-reduced-motion`. Native scrolling and focusable image
links provide a photo carousel without client-side JavaScript. Post timestamps
follow Flutter's seven-day compact display, with full UTC timestamps available
to assistive technology and native tooltips. Anonymous profile navigation retains the two-tab public surface. Authenticated
profiles offer the corresponding mobile actions and content tabs.

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

The anonymous frontend forwards neither incoming credentials nor arbitrary headers. It
contacts only the configured AppView origin, disallows redirects, bounds upstream
responses to 2 MiB, and times requests out. Public response DTOs never contain
viewer state, editable links, drafts, schedules, tokens or subscription details.
Anonymous pages never access a PDS directly. Text is escaped, external links accept HTTPS only,
and outbound user links are marked `ugc nofollow noreferrer`.

HTML and JSON are `no-store`; page metadata uses eligible content only. The
Cloudflare configuration must not override these with a cache-everything rule.
Search engines and social networks control their own copies of public content.
This first version has no content directory/sitemap; crawlers discover pages
through shared links and profile pagination. Cursor pages are `noindex,follow`.

Images retain authored alt text and dimensions where provided. Video uses the
existing public HLS playlist with native browser controls and a direct playlist
link. Browsers without native HLS need a compatible player; captions and expanded
quote/comment rendering remain available in the authenticated application.

## Dependency audit

The locked dependency audit on 2026-11-02 reported four low-severity entries
through `deepmerge` 4.3.1, used by the Vite Svelte plugin for build configuration.
The registry had no newer stable `deepmerge` release. Public request data is not
passed through that build-time merge. Recheck the advisory when upgrading; do
not apply the audit tool's proposed downgrade to pre-release-era SvelteKit.

## Production deployment and rollback

This replaces the static verified-link host, **not** the marketing Worker or an
AppView Render service. No production settings have been changed here.

1. Review and release AppView, including migrations 000112 and 000113, through the existing exact-tag workflow in
   [releases](../docs/operations/releases.md). Confirm the four public reads work
   anonymously, and hidden/deleted/unsafe fixtures are not returned.
2. Identify the existing Cloudflare Pages project owning `app.craftsky.social`.
   Record its current deployment ID and settings for rollback. Do not create a
   second project, move the custom domain, or alter DNS as an incidental step.
3. Change that project's root directory from `verified-links` to `web-client`,
   build command to `npm ci && npm run build`, and output directory to
   `.svelte-kit/cloudflare`. Build from the reviewed repository commit with Node 24.
   Set runtime `APPVIEW_ORIGIN=https://appview.craftsky.social` in production.
   AppView `WEB_APP_ORIGIN` must remain `https://app.craftsky.social` in production.
   Anonymous preview checks use an explicitly reviewed nonproduction backend.
   OAuth redirects deliberately allow only the controlled production app origin
   or explicitly configured development loopback; arbitrary Pages preview hosts
   cannot receive a production login handoff. Set optional Firebase public
   variables at build time if browser notifications are enabled. The Wrangler
   name is a local artifact label; confirm the existing project name before any
   CLI publication and pass it explicitly with `--project-name`.
4. Upload a preview deployment for review before promoting it to the existing
   custom domain. Release the updated Android intent-filter rules before testing
   browser OAuth with the native app installed; the former host-wide filter
   otherwise claims the browser-only callbacks. Preview hosts get `X-Robots-Tag: noindex, nofollow`. Verify the
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
