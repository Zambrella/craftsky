# CraftSky web client

SvelteKit 3 / Svelte 5 / TypeScript client and server-rendered public pages for
`https://app.craftsky.social`. The marketing site stays in `web/`. The mobile app
continues to use Flutter. The signed-in application lives at `/feed`.

Browser routes use the mobile router's names: `/feed`, `/search`,
`/search/tags?tag=…`, `/notifications`, `/notifications/settings`, `/messages`,
`/messages/conversations/:conversationId`, `/profile`, `/profiles/:did` (or
`/profiles/@handle`), `/posts/:did/:rkey`, `/events/:did/:rkey`, `/saved`,
`/saved/folder?folder=…`, `/scheduled`, `/drafts`, and `/settings/*`.
`/projects` redirects to `/feed`, preserving query parameters, as on mobile.
`/` opens `/sign-in` when signed out and `/feed` when signed in. Public post and
profile URLs render the public view when signed out and the app view when signed
in. Browser pages for mobile modals retain `/compose/*`, `/profile/edit`, and
`/feed/filters`; they also have no `/app` prefix.

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
npm run format        # Apply Prettier to maintained source/configuration
npm run format:check
npm run lint
npm run check
npm test
npx playwright install chromium
npm run test:release  # Complete local fixture gate, also used by CI
```

`test:release` builds with a loopback-only diagnostic DSN, executes consent/privacy
envelope tests, and runs both enabled and disabled subscription/video cases.
After successful fixture checks, the gate rebuilds using the original environment,
so the deployment script receives the normal artifact. A failed gate prevents upload.
ESLint covers JavaScript, TypeScript, and Svelte, including async promise-executor
mistakes, unused variables, and Svelte correctness rules. Type analysis remains
owned by `svelte-check`. Generated output and third-party static assets are excluded.
CI retains failure traces/screenshots only for these synthetic fixture tests.

Draft changes stay in memory until **Save draft** is chosen. Closing/discarding
never rewrites another tab's saved data. Explicit saves and deletes check the
stored revision atomically. Submitted post/message commands retain only the
recovery needed to check their original outcome; this is independent of draft
autosave. Chat saves use Web Locks plus a stored-value comparison. A conflicting
tab retains its text in the editor and asks the user to reopen the latest version.

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

## Live smoke test (Flutter Patrol parity)

This separate, opt-in Playwright suite mirrors
[`app/patrol_test/live_smoke_test.dart`](../app/patrol_test/live_smoke_test.dart).
It uses real browser OAuth, this checkout's Docker AppView and the same coral/teal
PNG fixtures. No provider, API, upload or persistence operation is mocked.

Put `E2E_HANDLE` and `E2E_PASSWORD` in ignored `web-client/.env.local` using a
**dedicated disposable Bluesky-hosted account** and its full account password.
The runner preserves `#` in an unquoted password, matching Patrol; quoted dotenv
values also work. Do not add inline comments to the password line. App passwords
do not work for OAuth.

```sh
# Repository root: start this checkout's backend.
just dev-d
# Stop just web-run first: the test owns the configured local web callback port.
cd web-client
npm run test:e2e:live -- --list  # Discovery only; no account/database changes.
npm run test:e2e:live
# Optional visible Chromium:
npm run test:e2e:live -- --headed
```

The runner discovers the worktree's AppView port and reads its `WEB_APP_ORIGIN`.
Both must be explicit HTTP loopback addresses; there is no remote-backend override.
It checks health, resolves the account DID and resets **only that DID's local
onboarding completion**, as the Patrol wrapper does. The backend must already
support real OAuth and allow its configured browser callback. The web port must
be free; the suite never reuses an unrelated running server.

The journey changes the profile bio, completes all three onboarding steps,
selects the two fixtures, rotates the first image through the editor, publishes,
likes, edits the text and removes the second image, then deletes its own post
through the UI. Fresh account-bound AppView reads check bio persistence, image
count, like state, changed record CID/edit timestamp, retained first image CID
and eventual deletion/404. Image downloads use a separate unauthenticated
request context; decoded dimensions must match the accepted crop output and the
untouched 640×480 second image. Local previews verify fixture colors and that
only the first image changes. Web uses the browser file input instead of the
native OS photo picker, and the web Post action publishes without Patrol's
native confirmation dialog.

**The bio changes and test records are public on the account's real PDS**, even
with a local AppView. A unique marker identifies each run. Cleanup targets only
that run's owned post and uses the UI; it never bulk-deletes records. A failure
before indexing, a stopped process, or failed deletion can leave the marked post
behind: remove it through CraftSky before retrying. The bio is intentionally not
restored, matching Patrol. Live tests are excluded from ordinary `npm test`,
`test:e2e` and `test:e2e:app` and are not a CI gate.

Use the wrapper rather than invoking Playwright directly. It loads credentials
only at runtime and redacts them from line-buffered output. Trace, video,
screenshots, persisted sessions and detailed error reports are disabled; temporary
runner artifacts are removed. Do not enable Playwright protocol/debug logging or
record OAuth sessions. A redacted stage and single-line failure reporter identifies the failing check.

Validation on 4 November 2026: the complete live Chromium journey passed against
this checkout's local AppView, including real Bluesky OAuth, persisted onboarding
bio, two decoded published images, persisted like, edited text with retained
first-image CID and like, and UI deletion followed by a fresh 404. The standalone
live-test TypeScript check and `git diff --check` also passed. This proves the
local web/AppView journey with the real provider/PDS, not production web hosting
or native OS photo-picker behavior.

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

Set `PUBLIC_SENTRY_DSN` and `PUBLIC_SENTRY_ENVIRONMENT` at build time. Reporting
stays off until explicitly enabled in onboarding or Settings → Privacy. Unexpected
browser/framework failures and consumed workspace/composer failures retain safe
stacks, deployment metadata and AppView correlation; private payloads and expected
cancellation/offline/validation failures are excluded. Revoking consent stops export.

Builds stamp debug IDs and retain source maps privately in `.sentry-artifacts/`,
removing maps from public assets. Run `npm run sentry:upload` with the authenticated
Sentry CLI before deploying that exact build. Ordinary builds do not upload code.
See [diagnostics setup, privacy and verification](src/lib/features/diagnostics/README.md).

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

The 2026-11-06 remediation upgrades `@sentry/vite-plugin` to 11.11.0,
which uses the patched `glob` 14 chain. A `minimatch ^11.0.5` override also
covers ESLint/typescript-eslint's development-only dependency; remove the
override when their declared ranges include the fix. Lint, unit tests, builds,
and private-map checks must pass after dependency updates.

The remaining four low-severity audit entries are the transitive `deepmerge`
4.3.1 chain in the Svelte build tools; its advisory has no patched version.
`npm audit --omit=dev` reports no advisories. Do not apply the audit tool's
suggested pre-release-era SvelteKit/adapter downgrades.

## Production deployment and rollback

The authoritative workflow is [Web client: preview, then release](../docs/operations/releases.md#web-client-preview-then-release).
The client uses the `craftsky-public-app` **Worker with static assets**, not a
Pages production build. Create and push an annotated `web-vX.Y.Z` release with
`just release-create-web` / `just release-push web`, then run
`just web-client-preview web-vX.Y.Z` from that exact release commit on clean
`main`. It checks/builds and uploads without promoting live traffic. Untagged
and dirty uploads are rejected. `VERSION` controls the displayed version and
Sentry release; the release command maintains `CHANGELOG.md`. Review/sign in at
<https://preview-craftsky-public-app.hugetoothstudios.workers.dev>, then promote
the recorded version ID to 100% in Cloudflare's Deployments tab.

Preview sign-in requires releasing AppView with ADR 026's exact preview callback
allowlist. Unique version URLs and other preview aliases cannot receive login
handoffs. Preview uses production accounts/data, with real writes. The initial
setup preserves the existing `craftsky-verified-links` Pages host on
`app.craftsky.social`; move that domain to the Worker only at first public
release, following the release runbook. The marketing Worker remains separate.

Preserve migrations 000112/000113 and the public-read/privacy checks; verify
hidden/deleted/unsafe fixtures are not returned. Monitor backend latency and
429/503 rates. AppView anonymous admission limits remain shared across the
singleton. Edge controls must not weaken backend bounds or trust arbitrary
forwarded IP headers. Device app linking and external share-card/search behavior
remain live release checks, separate from local browser tests.
