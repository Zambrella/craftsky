# Web client parity review

Source review: 2 November 2026. This is a behavior inventory and validation
handoff, not evidence of a production deployment or third-party delivery.

The reference is the Flutter routes in `app/lib/router/route_locations.dart`,
their feature pages and their AppView API clients. The web uses the same AppView
contracts through an account-bound, same-origin browser BFF. Existing public
profile/post reads remain separate and are preserved.

| Mobile domain | Web implementation | Verification and boundaries |
|---|---|---|
| Sign-in, create account, multiaccount, expired sessions | `web-client/src/routes/sign-in`, `routes/auth/browser`, `routes/api/auth`, `lib/features/shell` | OAuth registration delegates account creation to the configured provider, as mobile does. Passwords and PDS credentials stay with the provider/AppView. Browser BFF unit tests cover receipt/callback/state/account binding and additive identity enrichment. Existing-account sign-in was verified through the real provider and local AppView, reaching `/app/home` with an active browser session; live provider signup and recovery were not exercised. |
| Identity/crafts/policy onboarding | `lib/features/social/Onboarding.svelte`, `ProfileEditor.svelte` | Three steps, mandatory age 16 declaration and current policy acceptance. Optional diagnostic consent is independent and hidden without configuration. Browser audit regression covers age/policy completion when no DSN exists. |
| Following and craft feeds, projects, refresh, filters, search and hashtags | `lib/features/social/Feed.svelte`, `Search.svelte`, `Filters.svelte`, `Tag.svelte` | Main content reads and cursor pagination are implemented. Browser regression tests cover actual filter wire values, protected content and stale completion handling; this document does not claim exhaustive data/large-account runtime coverage. |
| Posts/projects/comments/replies/quotes, reactions, saves, reporting, pins | `lib/features/social/MemberPost.svelte`, `Thread.svelte`, `lib/features/composer` | Account-bound writes retain operation keys and CID preconditions. Actual browser tests cover publication, editing, conflict recovery and ambiguous-write retry; report reason codes must remain aligned with the Flutter catalog. |
| Project fields | `lib/features/composer/ProjectEditor.svelte` | Fields match `app/lib/projects/composer/project_composer_fields.dart`: common/pattern/material/design fields plus sewing, knitting, crochet and quilting details. Embroidery has no extra structured detail form in the mobile reference. |
| Media, captions, crop/rotate and accessibility descriptions | `lib/features/composer/MediaEditor.svelte`, `media.ts`, `social/VideoPlayer.svelte` | Browser APIs provide selection/editing and HLS/native playback. Unit tests and local fake upload paths do not prove real PDS/video-service processing, provider quotas or every device codec. |
| Private drafts and schedules | `lib/features/drafts`, `lib/features/composer` | Account-scoped IndexedDB drafts preserve blobs and pending commands; scheduled payloads/media stay in private AppView storage. Local tests cover stale accounts, cleanup rollback, accepted-write recovery, scheduling and publish-now. Local drafts intentionally survive ordinary sign-out. |
| Saved items and folders | `lib/features/social/Saved.svelte`, `MemberPost.svelte` | Save/unfile/move, sort, create/rename/delete folders and the native delete-saves option are implemented. Main integration owns complete folder enumeration and per-folder cursor behavior. |
| Profiles, graph and customisation | `lib/features/social/Profile.svelte`, `ProfileEditor.svelte`; `lib/features/settings/Connections.svelte`, `Preferences.svelte` | Graph pages paginate and mutate the correct relationship. Growth preserves unavailable snapshots as gaps. Customisation uses the mobile palette/background assets. Native intentionally hides popularity counts in the main profile header; those counts live in Settings. |
| Business declarations/products/events | `lib/features/settings/Business.svelte`, `lib/features/social/Event.svelte`, `BusinessSummary.svelte` | Typed forms, full business snapshots, preservation flags, product ordering, UTC/IANA event times and CID preconditions. Images can be reviewed, cropped/rotated, described and recompressed before explicit upload. The profile summary renders offerings, location and primary contact/action fields. Local browser tests cover declaration CAS/product preservation, the real browser image transform and event create/edit/delete. |
| Notifications and messaging | `lib/features/social/Activity.svelte`, `lib/features/messages`; `settings/Preferences.svelte` | Activity/seen/category preferences and direct-message authorization, requests, drafts, receipts, report/delete/mute/leave flows exist. Messaging uses the shared Bluesky service, matching native. The Instagram-match activity destination opens Instagram discovery. Actual Bluesky/FCM delivery was not exercised. |
| Restricted accounts, moderation and deletion | `lib/features/settings/Account.svelte`, `SettingsWorkspace.svelte`; browser deletion BFF | Restricted users retain account/standing/muted/blocked/about navigation. Their account page skips member-only profile reads. Deletion uses fresh browser reauthentication and exact DID confirmation; proof is HttpOnly, not browser storage. Accepted deletion clears only that account's local composer/chat/push state; device cleanup failures cannot retry an accepted deletion. Case history, affected-content references with validated destinations, effects, email appeal and case-reference copying are implemented. |
| Languages and appearance | `lib/features/settings/Preferences.svelte` | Uses the exact pinned AppView base-language catalog; interface remains English as in mobile. System/light/dark choices persist per browser. Browser tests exercise theme reload and a full language preference replacement. |
| Subscriptions and licenses | `lib/features/settings/Billing.svelte` | Existing access, account/reconciliation, assignment/reassignment and unassignment are implemented. Existing subscriptions link to the official App Store or Google Play management page. Purchases and restore remain in mobile, per the user's explicit direction. Local browser test proves assignment targets the entered DID. |
| Instagram discovery | `lib/features/settings/Instagram.svelte`, `archive.ts`, `bzip-stream.ts` | Verification challenge, status, explicit candidate confirmation, discoverability, revoke/reactivate, reviewed following import, import management and follow/dismiss suggestions are implemented. Manual/JSON and stored/Deflate/Deflate64/BZip2/ZIP64 archives are supported with 20 MB output, 100k entries, 64 MB directory bounds, CRC and strict header checks. BZip2 runs in a dedicated cancellable Worker with a 30-second deadline and 40 MB compressed-input cap; both BZip2 and ZIP checksums are verified. Tests do not prove Meta webhook delivery or production verification. |
| Browser background push | `lib/features/push`, `src/service-worker`, migration `000113_browser_push` | Optional public Firebase/VAPID configuration, explicit permission, per-account bindings, token renewal and owner-scoped disable/logout cleanup. Go admission/sender tests and the real PostgreSQL migration test pass. Browser worker activation passes locally; actual configured Firebase/OS delivery remains unverified. Browser clicks select the owning account and open Activity or the authenticated chat reference. |
| Optional diagnostics | `lib/features/diagnostics` | Official Sentry SDK loads only after consent. Current SDK granular data collection is disabled; native error type/cause/stack coordinates survive a strict event/envelope allowlist. Actual SDK serialized-canary tests and consent lifecycle tests pass. No production DSN, ingestion, source-map upload or release symbolication was configured or verified. |
| About/legal/cache controls | `lib/features/settings/Account.svelte` | Terms/privacy/guidelines/support links are present. Native image-cache management is delegated to browser cache/site controls; the web worker does not cache private pages, messages or API responses. |

## Final validation

Completed locally on 2 November 2026 against the final application source:

- `npm run check`: **0 errors, 0 warnings**.
- `npm test`: **183 tests across 28 files passed**, covering API contracts, privacy, sessions, media, storage, messaging, filters, archive bounds and recovery. Three added handoff regressions cover legitimate AppView identity enrichment, ignoring unrelated upstream fields, and retaining strict browser-cookie schemas.
- `npm run build`: production SvelteKit/Cloudflare artifact and bundled service worker built successfully. HLS, archive codecs and optional diagnostic SDK code are loaded separately from the initial application.
- `npm run test:e2e:app`: **35 authenticated Chromium scenarios passed** on production preview; **the same 35 passed** with `WEB_CLIENT_WORKER=1` against the built Cloudflare Worker. This includes the real bundled BZip2 worker, compact quote DTOs, project-field annotations, repeated filter parameters, browser OAuth confirmation, restricted accounts, empty-202 deletion acceptance, business image transforms and subscription management links.
- `npm run test:e2e`: **9 public Chromium scenarios passed** on production preview; **the same 9 passed** against the built Worker. Initial HTML, metadata, privacy, native callback/association bytes, accessibility, light/dark themes and narrow layouts remain covered.
- `./scripts/appview-test focused ./internal/auth ./internal/api ./internal/app ./internal/push ./internal/accountdeletion ./internal/routes`: **all six suites passed against this worktree's PostgreSQL and MinIO**, including the complete migration history and preserved public-read privacy rules. The additional browser-push migration regression also passed against PostgreSQL.
- `fvm flutter test test/auth/platform/verified_link_configuration_test.dart`: **6 tests passed**. Android's shared-link/mobile-callback path rules match iOS; browser-only callbacks are excluded.
- `python3 -m unittest discover -s scripts -p test_public_app_dev.py`: **5 launcher tests passed**. `just --list` parses successfully and `just web-run` starts only the frontend after the separate backend startup.
- The local backend was rebuilt with `just dev-d`; `/healthz` returned healthy and the worktree sign-in URL returned HTTP 200.
- The mobile composer screenshot was inspected; automated accessibility and overflow checks pass. No physical-device or remote-CI result is implied.
- Dependency audit: **four existing low-severity build-tool advisories; no moderate, high or critical advisories**. No production credentials, projects or infrastructure were changed.

For the full frontend gate, run `just web-client-check`. Existing `public-app-*`
recipes remain available. Start interactive development with `just dev-d` followed
by `just web-run`; the backend and frontend remain separate operations.

Basic existing-account OAuth sign-in was subsequently verified against the real
provider and this worktree's AppView. It exposed profile decorators adding
`accountType`, `crafts` and `customisation` to the handoff response; the web now
ignores additive upstream fields while selecting and validating only auth data.
The browser fixture includes the same enrichment for ordinary and chat handoffs.
After this fix, all five authentication browser scenarios passed again on both
production preview and the Cloudflare Worker, alongside the 183-test unit suite,
clean type checking and the production build.

Remaining external verification is configuration-dependent: additional PDS
providers, account creation/recovery, PDS image/video processing, Instagram/Meta DM
verification, store entitlement reconciliation, FCM browser/OS notifications and
Sentry ingestion/symbolication. These were not exercised against live accounts.
The updated Android path rules require a mobile release before testing browser
OAuth on `app.craftsky.social` with that native app installed. Production AppView
release, optional public Firebase/Sentry configuration and reviewed web publication
remain deployment tasks; this implementation did not deploy them.
