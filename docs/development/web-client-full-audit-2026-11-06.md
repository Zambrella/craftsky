# CraftSky web client audit — 6 November 2026

Audited commit: `45b3f42e55e4` (`fix(web-client): handle errors consistently across features`).
Scope: `web-client/`, its CI/release entry points, and the relevant AppView trust
boundaries. This is an audit and remediation backlog; application code was not
changed. The earlier error-handling review is incorporated, not treated as proof
that every failure path or browser test is covered.

The core design is sound: server-held credentials, account-bound requests,
escaped content, explicit proxy routes, durable publication attempts, and
privacy-filtered diagnostics. The main weaknesses are concurrency across browser
tabs, resource bounds at the proxy, incomplete recovery paths, and gaps between
the implemented behavior and the release checks. No cross-account authorization
bypass or executable-content injection was demonstrated in the reviewed paths.
That is a scoped result, not a penetration-test certification.

Implementation follow-up: [selected remediation and validation](web-client-audit-remediation-2026-11-06.md).
The findings below describe the audited commit before that remediation.

## Priority and evidence

- **P1:** address before treating this branch as release-ready: data/recovery
  integrity, publicly reachable resource amplification, or a failing release gate.
- **P2:** address in the next reliability/security pass, or before enabling the
  affected feature.
- **P3:** planned quality or accessibility improvement.
- **Reproduced:** a local test or isolated probe exercised the behavior.
- **Source-confirmed:** the relevant control flow/configuration was inspected;
  the end-to-end failure was not induced.
- **Risk/opportunity:** an improvement with a concrete source basis, without a
  claim of observed production impact.

## Findings

### F01 — P1: another tab can destroy unresolved write recovery

**Evidence: reproduced.**
[draft storage](../../web-client/src/lib/features/drafts/store.ts), lines 180–195;
[composer recovery](../../web-client/src/lib/features/composer/recovery.ts), lines
70–123; [message storage](../../web-client/src/lib/features/messages/model.ts),
lines 49–96.

Each composer activation serializes its own saves, but two activations of the
same draft have independent queues. `save` and `restore` unconditionally replace
the IndexedDB row; `remove` unconditionally deletes it. A stale editor can erase
another tab's frozen pending request, restore an older version on Discard, or
recreate the row after the other tab publishes it. The draft-list Delete button
only checks its previously loaded snapshot. The same pattern exists in chat:
two `MessageDraftStore` instances share a localStorage key without a revision
check or ownership protocol.

The probes demonstrated both a composer pending key being overwritten and a
completed row being recreated. A separate probe demonstrated a stale chat save
erasing another tab's frozen send. This weakens the guarantee that an uncertain
write can only be retried with its original key and payload.

**Fix:** add atomic revision/ownership checks to persistence, including delete,
discard, publication completion, and pending transitions. A tab notification
channel can explain the conflict, but is not a substitute for atomic checks.
For chat, use a transactional store or a supported cross-tab lock protocol.
Preserve conflicting text as a separate recoverable copy. Never silently replace
an unresolved attempt.

**Acceptance:** two tabs editing, sending, publishing, discarding, and deleting
the same item cannot erase pending state or resurrect a completed item. Test
crashed tabs and expired ownership as well as normal handoff.

### F02 — P1: the proxy buffers large bodies before authenticating the bearer

**Evidence: buffering path reproduced; production exhaustion not attempted.**
[proxy](../../web-client/src/routes/api/appview/[...path]/+server.ts), lines 31–88;
[credential parsing](../../web-client/src/lib/server/auth/session.ts), lines 265–285.

The proxy accepts a structurally valid account cookie, retains up to 32 MiB in
chunks, and allocates another full-size byte array before AppView validates the
opaque bearer. Cookie metadata is intentionally unsigned, so a direct HTTP
caller can construct a syntactically valid cookie containing an invalid token.
Origin checks protect browsers against CSRF; they do not authenticate a direct
HTTP client. The probe confirmed full body consumption before the upstream 401.

Near the limit, the implementation retains roughly two body-sized buffers before
other runtime overhead. Concurrent requests can amplify memory use within an
isolate. The 120-second timeout starts only after buffering, so it does not bound
the incoming read. Cloudflare documents a 128 MB isolate memory limit and
recommends streaming large bodies. [Cloudflare limits](https://developers.cloudflare.com/workers/platform/limits/),
[streaming guidance](https://developers.cloudflare.com/workers/best-practices/workers-best-practices/).

**Fix:** enforce route-specific body limits, reject declared oversized bodies
early, and stream through a byte-counting bound where supported. Small JSON
routes should not inherit upload-sized allowances. If buffering remains
necessary, validate authority and apply admission controls before allocating it;
also bound body-read time. Do not trust the cookie shape as authentication or
forward browser-supplied bearer/forwarding headers.

**Acceptance:** invalid credentials and oversized/slow bodies consume bounded
resources; concurrent large uploads remain within Worker limits. Exercise actual
Worker streaming and cancellation locally before any hosted load test.

### F03 — P2: a late 401 can delete a newly refreshed session

**Evidence: reproduced with two request cookie snapshots.**
[session store/remove](../../web-client/src/lib/server/auth/session.ts), lines
297–343; [proxy 401 handling](../../web-client/src/routes/api/appview/[...path]/+server.ts),
line 95.

Signing in again or completing chat authorization for an existing DID reuses its
account ID and cookie name. A request sent with the old token can subsequently
return 401 and emit a deletion for that same cookie. The browser applies that
late deletion to the newly stored token. `remove` also derives active-account
changes from the stale request snapshot. Account binding prevents using another
account's bearer, but does not prevent this response-ordering race.

**Fix:** distinguish account identity from credential activation/generation and
make invalidation target only the rejected activation. A client-side check alone
cannot undo a stale `Set-Cookie` deletion. Review active-account and pending-intent
cookies under the same response-ordering model.

**Acceptance:** delay an old-token request, refresh the same account in another
tab, then release its 401. The new activation and any newer account selection
must remain usable. Include chat reauthorization and logout races.

### F04 — P2: release flags disappear on signed-in public URLs

**Evidence: reproduced using the actual flag helpers and both page-data shapes.**
[release config](../../web-client/src/lib/features/shell/release-config.ts), lines
4–7; [post loader](../../web-client/src/routes/posts/[did]/[rkey]/+page.server.ts),
lines 6–8; [profile loader](../../web-client/src/routes/profiles/[handleOrDid]/+page.server.ts),
lines 7–9.

The helpers read `page.data.subscriptionsEnabled` and `videoUploadsEnabled`.
Signed-in post/profile routes instead return these inside `page.data.authenticated`.
The layout normalizes those two shapes for the shell, but the flag helpers do
not. An enabled feature therefore becomes disabled on these URLs: for example,
paid post actions can show Coming soon on a post detail that was available in the
feed. This is a false-negative feature gate, not an entitlement bypass. The
checked-in production subscriptions flag is currently false, which masks that
part of the defect until enabled.

**Fix:** expose a single typed authenticated context/configuration to all routes.
**Acceptance:** enabled and disabled flags behave identically on direct loads,
client navigation, own-profile routes, other-profile routes, and post details.

### F05 — P2: stalled media requests can trap the publishing screen

**Evidence: simulated stalled upload reproduced; related paths source-confirmed.**
[video upload](../../web-client/src/lib/features/composer/media.ts), lines
266–325; [publishing overlay](../../web-client/src/lib/features/composer/PublishingOverlay.svelte),
lines 26–39; [transport](../../web-client/src/lib/api/transport.ts), lines 159–166;
[remote image editing](../../web-client/src/lib/features/composer/MediaEditor.svelte),
lines 152–164.

Video's 15-minute limit is checked between polling iterations, after the initial
upload and body read. Neither the upload nor an individual status fetch has its
own deadline. A response that never settles never reaches that check. The probe
advanced time by 16 minutes and the upload remained pending until explicitly
aborted. While publishing, internal navigation is canceled and the modal blocks
Escape without offering a cancellation action.

The shared browser transport and remote-image fetch also rely on caller signals
without a default deadline. The BFF's upstream timeout does not cover every
browser connection, direct upload, or incoming body phase. Remote image editing
checks size only after `response.blob()` has downloaded the whole response.

**Fix:** compose lifecycle cancellation with per-stage deadlines, including body
reads, and stream-limit remote media. Provide a safe way to stop waiting. A
timeout after a record write may have been sent must retain the exact pending
operation; it must not be presented as a definite rejection or retried with a
new key. [AbortSignal guidance](https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal).

**Acceptance:** hung upload, hung poll/body, cancellation, and lost mutation
responses all leave a usable screen and preserve the appropriate recovery state.

### F06 — P2: autosave diagnostics still lose the original storage failure

**Evidence: source-confirmed.**
[draft storage](../../web-client/src/lib/features/drafts/store.ts), lines 108–109
and 138–149; [autosave](../../web-client/src/lib/features/composer/ComposerWorkspace.svelte),
lines 270–279 and 306–318.

IndexedDB open/transaction errors are replaced with a `UserFacingError` without
`request.error`/`tx.error` as the cause. Autosave catches discard even that wrapper
and only update a status string; the teardown checkpoint catch is empty. A quota
failure, database shutdown, and other failures therefore lose their typed cause
or never reach the protected reporter. This remains outside the paths fixed in
the previous error-handling pass.

**Fix:** retain available native causes and report once per continuing storage
failure at the consuming boundary, resetting suppression after success. Keep
draft text, media, filenames, and identifiers out of diagnostic context. Show a
persistent, actionable unsaved-state warning.

**Acceptance:** injected quota/open/abort failures retain the native cause in
serialized safe diagnostics, never include private canaries, never claim a save,
and do not generate one issue per keystroke.

### F07 — P2: failed captions cannot be retried in the current player

**Evidence: source-confirmed.**
[video player](../../web-client/src/lib/features/social/VideoPlayer.svelte), lines
24–46 and 96–120.

`captionsStarted` is set before loading and never reset on failure. After one
transient failure, replaying the video cannot reload captions; the only offered
link opens the HLS playlist, which does not recover CraftSky's fetched caption
tracks. Native-HLS media errors also lack the explicit recovery handling used by
the hls.js branch. This matters especially for users relying on captions and
Safari's native playback path.

**Fix:** track success per caption, offer retry for failed tracks, and add native
media-error feedback. Capture unexpected consumed failures with safe static
labels. Avoid duplicate tracks/object URLs and release them on disposal.

**Acceptance:** fail a caption once, retry without navigating away, and verify
that it becomes selectable; exercise native-HLS error/recovery separately.

### F08 — P1 release gate: seven authenticated browser tests currently fail

**Evidence: reproduced against both Node preview and the local Worker.**

Each full diagnostics-enabled run produced **328 passed, 7 failed, 2 skipped**.
All seven observed failures were stale expectations following the error-handling
changes, rather than proof that the underlying actions failed:

| File | Assertion line | Stale expectation |
| --- | ---: | --- |
| `tests/authenticated-like-animation.spec.ts` | 147 | Raw rejected-like server message |
| `tests/authenticated-navigation-parity.spec.ts` | 148 | Raw hashtag-outage server message |
| `tests/authenticated-search.spec.ts` | 280 | Diagnostic operation `workspace` |
| `tests/authenticated-settings-detail-parity.spec.ts` | 117 | Raw messaging-preference message |
| Same file | 191 | Raw notification-preference message |
| Same file | 295 | Raw language-preference message |
| Same file | 658 | Raw Instagram-discovery message |

Six expect server prose that is now deliberately replaced by client-owned copy;
one expects the old operation instead of `social.Search.suggest`. These assertions
also prevent the later rollback/navigation/privacy checks in those tests from
running. The earlier focused 11-test validation did not establish full-suite
success.

**Fix:** update the expected public copy and static operation labels, explicitly
assert that protected server canaries are absent, retain every behavioral
assertion, and run the complete Node and Worker gates. Do not restore raw server
messages or skip tests to make the suite green.

### F09 — P2: CI skips diagnostic privacy and production-disabled feature cases

**Evidence: source-confirmed.**
[CI](../../.github/workflows/web-client-ci.yml), lines 36–44;
[authenticated configuration](../../web-client/tests/playwright-authenticated.config.ts),
lines 4–5; [diagnostic tests](../../web-client/tests/authenticated-diagnostics.spec.ts),
lines 3–7; [search tests](../../web-client/tests/authenticated-search.spec.ts),
lines 233–239.

CI neither builds with a loopback Sentry DSN nor sets `WEB_CLIENT_DIAGNOSTICS=1`,
so the three SDK envelope/consent tests and two diagnostic search cases skip.
The tests also default subscriptions and video uploads to true, while the beta
and video-disabled cases require explicit false values. Those two cases skipped
in this audit's enabled-feature runs too; both passed when run separately with
the flags disabled. The checked-in Worker configuration
ships subscriptions disabled and does not enable video uploads.

**Fix:** add explicit, isolated diagnostic and release-flag test configurations
to CI/local release checks. Keep the diagnostic build fixture separate from
deployment artifacts and intercept all telemetry. Upload failure artifacts from
fixture tests so a developer can inspect failures without reproducing the entire
suite. The current CI has no artifact-upload step.

**Acceptance:** CI shows those cases executed, with no hosted telemetry or live
account credentials, and checks the configuration intended for release.

### F10 — P3: authenticated pages have indistinguishable browser titles

**Evidence: source-confirmed.**
[AppShell](../../web-client/src/lib/features/shell/AppShell.svelte), lines 608–615.

Every authenticated screen uses the title `CraftSky`. Feed, Settings, composer,
and messaging tabs cannot be distinguished from their title, and route changes
do not provide a meaningful new page title for assistive technology. Public
post/profile metadata and Sign in already have descriptive titles.

**Fix:** define route-owned titles such as `Notifications · CraftSky` and
`Privacy settings · CraftSky`. Keep private message contents and sensitive
recipient details out of browser history/title text. W3C explains why descriptive
page titles help navigation and orientation. [Page Titled guidance](https://www.w3.org/WAI/WCAG22/Understanding/page-titled.html).

**Acceptance:** direct and client-side navigation update titles meaningfully;
check title announcements with a screen reader as well as DOM assertions.

### F11 — P2 coverage gap: mobile layouts are tested only in Chromium

**Evidence: source-confirmed.**
[public config](../../web-client/playwright.config.ts),
[authenticated config](../../web-client/tests/playwright-authenticated.config.ts),
and [CI](../../.github/workflows/web-client-ci.yml), line 40.

There are substantial narrow/iPad layout and Axe checks, but no configured
Firefox or WebKit projects and CI installs only Chromium. Viewport emulation
does not exercise Safari's IndexedDB lifecycle, native HLS, Home Screen push,
popover/focus behavior, or browser-specific media handling.

**Fix:** add a focused WebKit and Firefox smoke suite covering authentication,
draft save/reopen, compose/media, key dialogs/selectors, and navigation. Keep
physical iOS push and VoiceOver checks as a separate manual release checklist;
Playwright WebKit is useful coverage, not physical-device proof.
[Playwright browser projects](https://playwright.dev/docs/test-projects).

### F12 — P3: a build-tool advisory patch is now available

**Evidence: current npm advisory and package metadata checks.**
[package manifest](../../web-client/package.json), line 21;
[dependency notes](../../web-client/README.md), “Dependency audit”.

The locked audit reports **4 moderate and 4 low entries**, all through development
tools; `npm audit --omit=dev` reports **zero**. These are dependency-tree entries,
not eight independent remotely exploitable application vulnerabilities.

Unlike the previous README note, npm now offers `@sentry/vite-plugin@11.11.0` as
a non-major remediation for the moderate chain. Its published metadata depends
on `@sentry/bundler-plugins@11.11.0`, which uses `glob ^14.0.3`. The glob advisory
lists 14.0.2 as patched. No upgrade was installed during this audit.
[Glob advisory](https://github.com/advisories/GHSA-3g3m-2fjr-6pw4).

**Fix:** upgrade the build plugin deliberately and rerun the private-map/debug-ID
checks, fixture diagnostic envelope tests, build, and audit. Assess the remaining
low-severity `deepmerge` chain separately; the advisory lists no patched release.
Do not apply npm's suggested SvelteKit/adapter downgrades.
[deepmerge advisory](https://github.com/advisories/GHSA-gcqx-w48m-w2h4).

## Development and performance opportunities

These are improvement work, not additional demonstrated exploits.

1. **Extract controllers around coherent responsibilities.**
   `MessagesWorkspace.svelte` is 1,906 lines, `ComposerWorkspace.svelte` 1,630,
   `Business.svelte` 1,289, `ProfileEditor.svelte` 1,254, and `AppShell.svelte`
   1,091, including markup/styles. The chat controller is already a useful
   precedent. Extract composer publication/recovery, account activation, business
   editing, and page-history orchestration in small behavior-preserving changes.
   Avoid a generic framework rewrite. Resolve F01/F03 with explicit invariants
   before consolidating lifecycle helpers. The two Work implementations currently
   have different retry/read semantics; preserve those differences intentionally.

2. **Validate API and stored-data shapes at the boundary.**
   `APITransport` verifies object/page envelopes, but most reads ultimately cast
   to `T`; `publicAPI` casts parsed JSON directly. Draft records also have no
   schema/version decoder beyond owner checking. Add focused decoders for
   authority-bearing responses, presentation unions, and durable pending records.
   Test malformed rows/items without allowing one bad record to destroy an entire
   feed or draft list. Consider generating DTOs/consumer-route contract checks
   from a shared contract to reduce Flutter/AppView/web drift. This does not
   require a new API serialization format or lexicon change.

3. **Measure startup work before changing bundle splitting.**
   Authenticated server navigation performs `whoami`; shell boot then requests
   eligibility, onboarding, billing, language preferences, and profile, with
   feature screens sometimes repeating reads. Measure latency/request counts and
   introduce account-scoped single-flight/caching only with explicit invalidation.
   Keep authorization server-enforced. `LikeReaction` eagerly initializes Rive on
   mount even for reduced-motion users: the build contains a 2,241,449-byte WASM
   asset and an approximately 344 kB minified runtime chunk. Consider idle/intent
   loading while preserving the immediate CSS fallback and animation behavior.
   The largest JS warning is the approximately 579 kB hls.js chunk, which is
   already dynamically imported on playback; it is not evidence that all users
   download it at startup. Sentry is also consent-loaded. Add route transfer and
   responsiveness budgets, rather than suppressing the warning or blindly
   splitting every component.

4. **Bound decoded-image work and make draft storage health visible.**
   `prepareImage` accepts up to 40 MB compressed input and decodes the original
   bitmap before limiting output to 4,000 pixels per side. That output limit
   does not bound decode memory. Evaluate header/dimension limits and supported
   decode-time resizing, with representative large camera files on mobile
   browsers. No device OOM was reproduced here. Add storage-capacity/persistence
   handling where supported and a clear recovery/export option for device-only
   drafts. Do not promise that browser storage is a durable cross-device backup.

5. **Make routine maintenance repeatable.**
   Prettier is installed/configured, but there is no format-check script or CI
   gate; there is no ESLint dependency/configuration in this package. Add a small
   Svelte/TypeScript rule set for async mistakes and formatting checks, avoiding
   cosmetic mass rewrites. Add dependency-update automation or a scheduled audit
   process with reviewed exceptions, particularly for toolchains that ordinary
   production-only audits omit. Document the supported browser versions and a
   short manual assistive-technology checklist.

## Safeguards verified in the reviewed implementation

- Secure, HttpOnly, host-only `__Host-` cookies on HTTPS; opaque AppView session
  tokens stay out of JavaScript. Cookie metadata is treated as untrusted and
  bearer identity is verified server-side.
- Explicit consumer route/method allowlist, segment encoding/traversal checks,
  same-origin mutation checks, no arbitrary upstream origin, no incoming bearer
  forwarding, and no upstream redirects followed by the BFF.
- OAuth state and pending handoff validation; source-token binding for chat
  authorization; owner-bound fresh-reauthentication account deletion.
- Escaped text rendering and HTTPS-only user link/media helpers. No raw HTML
  rendering sink was found in `src`. CSP blocks objects/framing and arbitrary
  inline script. The broad HTTPS connect/media policies support federation;
  review narrowing where possible, but they are not by themselves an exploit.
- No-store page/API responses and no-referrer headers. The service worker has no
  fetch/cache handler for authenticated content.
- Direct video capability goes only to the fixed video service with omitted
  credentials, no redirects, and no referrer. Record writes remain mediated by
  AppView. Archive imports have output/directory bounds and a terminated BZip2
  worker deadline.
- Consent-gated browser reporting, final serialized privacy filters, safe static
  labels, and private source maps. The built public Worker directory contains
  **zero `.map` files**. Hosted receipt and symbolication were not verified.
- Strong existing local tests for account isolation, frozen retry payloads,
  published-edit conflicts, IndexedDB transaction completion, Unicode/IME,
  keyboard/focus behavior, contrast, reduced motion, and narrow layouts.

## Validation evidence

All checks use this worktree and local fixture data. No real OAuth account,
production API write, deployment, or source-map upload was performed. The
starting working tree was clean. Temporary probes were used to verify existing
behavior and are not proposed regression tests that bless the defects.

- `npm run check`: zero errors and zero warnings.
- `npm test`: 342 passed across 55 files.
- Production build with `PUBLIC_SENTRY_DSN=http://public@127.0.0.1:19846/1`
  and `PUBLIC_SENTRY_ENVIRONMENT=test`: passed; private maps prepared, no upload.
  This is a fixture diagnostic build, not an artifact to promote to production.
- `npm run test:e2e`: 11 passed.
- `WEB_CLIENT_WORKER=1 npm run test:e2e`: 11 passed.
- `WEB_CLIENT_DIAGNOSTICS=1 npm run test:e2e:app`: 328 passed, 7 failed,
  2 configuration-specific skips; failures detailed in F08.
- `WEB_CLIENT_WORKER=1 WEB_CLIENT_DIAGNOSTICS=1 npm run test:e2e:app`:
  328 passed, the same 7 failed, and 2 configuration-specific skips.
- Targeted Worker browser cases with `WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false`
  and `WEB_CLIENT_VIDEO_UPLOADS_ENABLED=false`: both passed.
- Six isolated probes passed by asserting the observed defects: composer
  multi-activation overwrite/resurrection; chat pending-send overwrite;
  old-token 401 cookie deletion after refresh; pre-auth body buffering; nested
  release-flag lookup; stalled video upload after simulated 16 minutes.
- `npm audit --json`: 4 moderate/4 low development-tool entries;
  `npm audit --omit=dev --json`: zero entries. Patch metadata read from npm;
  package/lockfile unchanged.

Loopback server checks required sandbox socket permission. Browser tests ran in
Chromium against both local Node preview and the local Cloudflare runtime.
Physical devices, WebKit/Firefox, screen readers, production edge rules/logs,
live Firebase delivery, real provider/PDS integration, and concurrent hosted
resource limits are outside the executed checks. Earlier live evidence is not
treated as a fresh result for this commit.

## Suggested implementation order

1. Restore the complete release gate (F08) and add the missing configurations
   (F09), retaining all meaningful behavioral/privacy assertions.
2. Fix persistence concurrency and session invalidation (F01, F03), followed by
   Worker resource bounds (F02). These need failure/race tests before refactors.
3. Normalize route configuration and finish recovery/accessibility work
   (F04–F07, F10).
4. Add cross-engine smoke coverage and upgrade the advisory-bearing toolchain
   (F11–F12).
5. Make small measured architecture/performance improvements under the restored
   test gate. Verify hosted configuration and real integrations separately before
   promoting a release.
