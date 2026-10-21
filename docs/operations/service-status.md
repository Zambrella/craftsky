# App status controls

The app reads `https://status.craftsky.social/app.json` anonymously, independently of AppView, accounts and device identity. A separate Cloudflare Worker reads the private `craftsky-service-status` R2 bucket's `app.json` object. Preview uses `craftsky-status-preview`, `craftsky-service-status-preview` and `https://preview-status.craftsky.social/app.json`.

These are intended resources. Local implementation and dry-run validation do not establish that they exist remotely. Initial provisioning, DNS, Worker deployment and live publication require separate authorization. No infrastructure or live status was changed during implementation.

## Validator environment

Set up the offline validator explicitly before using any publication command:

```sh
python3 -m venv service-status/.venv
service-status/.venv/bin/python -m pip install -r service-status/requirements.txt
```

The CLI, local helper and `just service-status-test` use that interpreter, or `CRAFTSKY_STATUS_PYTHON` when explicitly supplied. Commands never install validation dependencies or retrieve remote schemas. Drafts and final encoded payloads are checked against the local Draft 2020-12 `service-status/app.schema.json` and semantic rules. Missing dependencies (including URI/date-time format checkers), missing/invalid schema, broken/nonlocal references or validator failures prevent upload. The formal schema accepts additive unknown fields; canonical publication strips them.

## Local development

With the normal dev backend running, use these commands from the repo root:

```sh
just service-status-dev-setup
just service-status-dev
# In another terminal:
just app-run-status -d chrome  # or your iOS or Android device ID
```

Setup installs the pinned web tooling if needed, creates an editable draft at `service-status/.wrangler/dev/app.json` without overwriting an existing draft, and loads it into local simulated R2. The initial example includes all three controls; maintenance has priority. The server uses the real Worker on `http://127.0.0.1:8789/app.json`; starting/restarting it preserves the current local object. App launch supplies the status define for this run and reverses the status port for connected Android devices. It does not edit app config files. Run Flutter in debug mode for the loopback override.

Edit the draft, then update the running endpoint or clear it:

```sh
just service-status-dev-publish
just service-status-dev-publish /path/to/another-status.json
just service-status-dev-clear
```

Every update validates the document and assigns a fresh document revision while preserving the supplied message revision. Publishing/clearing does not rewrite the draft. Rerunning setup explicitly reloads the draft; clear followed by a server restart keeps all controls disabled. The app observes changes on its next foreground refresh (every 60 seconds), foreground resume or maintenance Try Again. Stop the server to test unavailable status and the five-minute maintenance expiry.

All object operations use `--local` and the preview bucket name; the server uses `--local` with remote bindings disabled. State and drafts live under the ignored `service-status/.wrangler/` directory. These commands do not provision resources, publish remotely or require a Cloudflare deployment. For another worktree, set `CRAFTSKY_STATUS_DEV_PORT` to an available port in both server and app terminals. Hosted preview checks remain separate.

## Initial setup

Use the existing Cloudflare account from `web/wrangler.json`; verify account membership and the intended resources before any mutation. Install the checked-in tool lockfile with `npm ci --prefix web`. Use the pinned `node web/node_modules/wrangler/bin/wrangler.js` CLI, authenticated through its authorized login or an operator environment token. Do not put credentials in JSON documents, source, app defines, receipts or logs.

After authorization, provision the separately named private buckets, disable public `r2.dev` and public bucket domains, and deploy `service-status/wrangler.json` for the explicitly selected environment. Set `CLOUDFLARE_ACCOUNT_ID` explicitly to the verified account for Worker setup. Inspect pinned CLI help before provisioning. `--env preview` selects only the preview Worker and binding; omitting it selects production. The publisher does not provision or deploy anything.

Locally review each environment first:

```sh
node web/node_modules/wrangler/bin/wrangler.js deploy --config service-status/wrangler.json --env preview --dry-run --outdir /tmp/craftsky-status-preview
node web/node_modules/wrangler/bin/wrangler.js deploy --config service-status/wrangler.json --dry-run --outdir /tmp/craftsky-status-production
```

Initialize the selected bucket with a confirmed `clear` command once its Worker/domain is available. A missing object is unknown status, not maintenance or an invented normal response.

## Publish or clear

Copy `service-status/example.json` into a local incident draft. All titles, descriptions, URLs and revisions are **public**; include no private account/editor information, credentials or operational secrets. Use plain English text. Optional `maintenance.estimatedRecoveryAt` is a zoned RFC3339 estimate; it never schedules maintenance or automatically clears it.

```sh
scripts/service-status validate /path/to/incident.json
scripts/service-status publish /path/to/incident.json --environment preview
scripts/service-status clear --environment preview
```

Use `--environment production` only for the intended production change. The environment is mandatory. No Git-clean/main-branch check, website build, AppView probe or release is involved. Validation is offline and reads at most 16,385 bytes. Publication validates first, uses the configured account and selected bucket, replaces the draft revision with a fresh UUID, and displays the target and exact intended public document. Type `publish` to confirm; decline or EOF performs no upload. The input file is never rewritten. Every publication and clear receives a new root revision. `message.revision` stays exactly as supplied: unrelated maintenance/minimum edits do not resurface the same dismissed message. Omit only `maintenance` in a complete draft to remove maintenance while preserving other controls. The `clear` preview contains only schema version and root revision and explicitly removes **all** controls.

Use fresh message revisions for each new campaign or intended resurfacing. Storage deliberately retains only the most recently dismissed revision: A dismissed → B dismissed → reused A can resurface. Do not reuse old campaign identities after later campaigns. Schema-2 messages use a new preferences key; legacy announcement markers are not migrated.

The command writes one temporary document to the fixed R2 object with JSON/no-store metadata. It then verifies the fixed public endpoint anonymously: no redirects/cookies/auth, bounded JSON, expected JSON/no-store/CORS/nosniff headers, and exact recognized fields/revision. Verification makes one direct GET with a three-second socket timeout. This is not an overall wall-clock deadline; the app independently keeps its three-second overall request deadline. Only verified public success returns zero. An upload attempt followed by any failure reports **unverified; live state may have changed**. There is no automatic rollback or write retry. Check the public state; restore it using a new confirmed publication if needed. Concurrent operators use last-write-wins: a later revision causes an earlier command's verification to fail honestly.

The confirmation intentionally shows public prose. Diagnostic errors never echo input paths, raw dependency output, response bodies or credentials. Receipts contain only environment/operation/revision and verified outcome. Protect terminal recordings appropriately.

## App behavior and contract

Schema 2 requires `schemaVersion: 2` and an ASCII `[A-Za-z0-9_-]{1,64}` document `revision`. Optional sections are absent when disabled; supplied null fields/sections are invalid. The full UTF-8 body is limited to 16,384 bytes. The formal schema and shared corpus are checked in under `service-status/`.

- `maintenance`: nonblank `title` (160 Unicode scalars), `description` (2,400), optional real zoned RFC3339 `estimatedRecoveryAt` (seconds, at most six fractional digits).
- `minimumVersions`: optional `ios`, `android`, `web` strict semantic versions; absent platform means no requirement. Build metadata has no effect. Core and all-numeric prerelease components are limited to 15 digits for consistent native/web comparison; this does not limit ignored build identifiers. Users below the platform minimum see a blocking Update/Reload page.
- `message`: required independent `revision`, `minVersion`, `title`, `description`; optional inclusive `maxVersion`, public HTTPS `imageUrl`, and `cta`. Reversed ranges fail validation. CTA requires nonblank `text` (160 scalars) and exactly one HTTPS `url` or named `destination`: `feed`, `projects`, `search`, `notifications`, `profile`, `settings`. No arbitrary routes or custom schemes. URL userinfo, controls and malformed syntax are rejected.

All prose renders as plain text. Unknown fields cannot configure API/auth destinations or executable content. Image URLs must point to a trusted public HTTPS host compatible with Flutter web CORS. Images use standard `Image.network` loading and caching without adding AppView authorization or account/device headers. Flutter/platform defaults govern cookies, referrer, redirects, request lifetime and resource use; there is no custom byte limit, deadline or transport cancellation. The display is bounded to 200 logical pixels high with `BoxFit.contain`. Images are decorative: essential information belongs in the description. Loading or failure leaves text/actions available.

Status starts alongside account initialization, checks on launch/resume/manual maintenance retry and every 60 foreground seconds, coalesces requests and expires requests after three seconds. Valid snapshots replace all controls atomically. A successful maintenance read authorizes a cover for five minutes; unchanged success renews it and failed reads do not. Estimates never schedule removal. Failed/invalid/timed-out refreshes release update enforcement immediately but retain only still-fresh maintenance; maintenance expiry cannot revive an invalidated update policy. Later valid snapshots can enforce again. Startup failure, unknown runtime version and unsupported test-host platform never invent update gates; unknown versions also suppress targeted messages. No policy is persisted offline.

Only the highest-priority applicable flow displays: fresh maintenance → trusted required update → eligible undismissed message. Suppression does not dismiss. Update/maintenance cover the retained application in ready/loading/error hosts and consume back; they retain routes, editors, media selections, sessions and ordinary account/policy gates, and never replay/cancel accepted writes or sign out users. Covered controls are removed from pointer/focus/semantics interaction. Valid retained focus is restored on uncovering.

Mobile Update opens the fixed CraftSky App Store/Google Play listing. Merely opening or returning from a store does not clear the gate. Web Reload explicitly reloads the current URL once; it may discard in-memory work. There is no automatic reload loop. Launch failure leaves retryable inline feedback. Messages retain button/barrier/Escape/back dismissal per installation/browser storage scope. Successful CTA activation suppresses and saves its captured message revision. Activation failure leaves retryable feedback; save failure after activation hides that revision only for the session and does not replay the action. A replacement message or new blocker remains authoritative during late completion. Named actions use the normal router and account/policy gates; initialization hosts without a ready router report unavailable action feedback.

**Rollout:** schema 1 is rejected by the new parser/publisher, with status failures handled as above. Old installed clients do not understand schema 2 and cannot acquire mandatory-update support from publication alone. Release a compatible app and publish configuration separately under normal release controls. No live publication/deployment is authorized by this runbook update.

## Verification and release evidence

```sh
just service-status-test
just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart
just app-test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart
just app-analyze
```

Hosted verification is a manual release check after separately authorized setup. Read the preview endpoint with `curl --fail --max-time 3 -i https://preview-status.craftsky.social/app.json`; confirm the JSON, revision, no-store and CORS headers. From the supported web app origin, use browser developer tools to fetch the same endpoint with `credentials: 'omit'`, `redirect: 'error'`, `cache: 'no-store'` and `referrerPolicy: 'no-referrer'`. Inspect the request for absence of cookies/auth/device headers and confirm the body matches. These manual steps replace the standalone hosted browser harness.

For freshness evidence, separately authorize preview-only publication of a combined snapshot, removal of maintenance, removal of minima, then full clear. Use the same native and browser clients across changes; after each verified publication, the next successful foreground refresh must observe the replacement within the app's three-second request deadline. Confirm a disposable landing-site deploy/rollback leaves the status revision unchanged. Capture safe results only. No hosted publication or cache/CORS verification has been performed locally.

Record real VoiceOver, TalkBack and supported web-reader results for long text, both themes, supported large text scale, retry/dismiss controls, hidden editor semantics, focus restoration and browser/Android predictive back. Local semantics and fake-R2 tests do not satisfy those checks. See the implementation execution notes in `docs/changes/2026-10-08-service-status-maintenance/05-implementation-plan.md` for actual checks and remaining release gaps.


For this change, MAN-001 verifies official listing availability/handoff on physical iOS/Android. MAN-002 verifies an already-open version A tab can obtain compiled version B at the same path/query under actual authorized-preview CDN/browser/bootstrap caches. Do not clear all browser data or unregister workers to manufacture recovery. The local compiled reload harness proves only the adapter/local build transition. `scripts/web-deploy` deploys the public website and is not evidence of Flutter-app hosting; identify the actual Flutter hosting/cache path before separately authorized release work. MAN-003 covers real readers, predictive/platform/browser back and focus. See `docs/changes/2026-10-20-app-status-flows/05-implementation-plan.md` for executed evidence and explicit unrun checks.

## Local compiled web verification

Use the existing pinned Playwright installation from `web/package-lock.json`; do not install/upgrade tools implicitly from a validation command. Build two disposable artifacts from `app/` with the production runtime provider and reload adapter:

```sh
flutter build web --debug --no-pub --no-web-resources-cdn \
  --target test/service_status/support/status_reload_harness.dart \
  --build-name 1.8.0 --build-number 1 \
  --dart-define=STATUS_HARNESS_COMPILED_BUILD=1.8.0+1 \
  --dart-define=CRAFTSKY_STATUS_URL=http://127.0.0.1:4176/app.json \
  --output /tmp/craftsky-status-build-a
flutter build web --debug --no-pub --no-web-resources-cdn \
  --target test/service_status/support/status_reload_harness.dart \
  --build-name 1.9.0 --build-number 2 \
  --dart-define=STATUS_HARNESS_COMPILED_BUILD=1.9.0+2 \
  --dart-define=CRAFTSKY_STATUS_URL=http://127.0.0.1:4176/app.json \
  --output /tmp/craftsky-status-build-b
```

Then from the repository root:

```sh
CRAFTSKY_STATUS_BUILD_A=/tmp/craftsky-status-build-a \
CRAFTSKY_STATUS_BUILD_B=/tmp/craftsky-status-build-b \
  npm exec --prefix web -- playwright test --config web/playwright.status.config.js
```

The dedicated runner starts only disposable loopback app/status servers on ports 4175/4176. It checks a compiled marker plus real package-info metadata, recovery to B, still-old gating, unavailable-status fail-open, and actual browser message dismissal across document reload. It counts document requests rather than router history updates. The current URL including an added fragment is preserved by the browser's committed reload; the existing app path strategy may normalize fragments again during startup. This harness does not change that routing policy or establish deployed cache behavior.

The anonymous status RequestInit tests remain runnable without that server in `service_status_repository_web.browser.dart`. Trusted-host images use Flutter network loading; the former custom image transport probes were removed.
