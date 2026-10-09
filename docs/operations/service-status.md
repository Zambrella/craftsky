# Service status and maintenance

The app reads `https://status.craftsky.social/app.json` anonymously, independently of AppView, accounts and device identity. A separate Cloudflare Worker reads the private `craftsky-service-status` R2 bucket's `app.json` object. Preview uses `craftsky-status-preview`, `craftsky-service-status-preview` and `https://preview-status.craftsky.social/app.json`.

These are intended resources. Local implementation and dry-run validation do not establish that they exist remotely. Initial provisioning, DNS, Worker deployment and live publication require separate authorization. No infrastructure or live status was changed during implementation.

## Local development

With the normal dev backend running, use these commands from the repo root:

```sh
just service-status-dev-setup
just service-status-dev
# In another terminal:
just app-run-status -d chrome  # or -d macos / your iOS or Android device ID
```

Setup installs the pinned web tooling if needed, creates an editable draft at `service-status/.wrangler/dev/app.json` without overwriting an existing draft, and loads it into local simulated R2. The initial example is maintenance. The server uses the real Worker on `http://127.0.0.1:8789/app.json`; starting/restarting it preserves the current local object. App launch supplies the status define for this run and reverses the status port for connected Android devices. It does not edit app config files. Run Flutter in debug mode for the loopback override.

Edit the draft, then update the running endpoint or clear it:

```sh
just service-status-dev-publish
just service-status-dev-publish /path/to/another-status.json
just service-status-dev-clear
```

Every update validates the document and assigns a fresh revision, so an announcement can reappear after dismissal. Publishing/clearing does not rewrite the draft. Rerunning setup explicitly reloads the draft; clear followed by a server restart stays normal. The app observes changes on its next foreground refresh (every 60 seconds), foreground resume or maintenance Try Again. Stop the server to test unavailable status and the five-minute maintenance expiry.

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

Copy `service-status/example.json` into a local incident draft. Title/message are **public**; include no private account/editor information, credentials or operational secrets. Use plain English text. Optional `estimatedRecoveryAt` is a zoned RFC3339 estimate; it never schedules maintenance or automatically clears it.

```sh
scripts/service-status validate /path/to/incident.json
scripts/service-status publish /path/to/incident.json --environment preview
scripts/service-status clear --environment preview
```

Use `--environment production` only for the intended production change. The environment is mandatory. No Git-clean/main-branch check, website build, AppView probe or release is involved. Validation is offline and reads at most 16,385 bytes. Publication validates first, uses the configured account and selected bucket, replaces the draft revision with a fresh UUID, and displays the target and exact intended public document. Type `publish` to confirm; decline or EOF performs no upload. The input file is never rewritten. Every publication and clear receives a new revision, so republishing an announcement makes it visible again after dismissal.

The command writes one temporary document to the fixed R2 object with JSON/no-store metadata. It then verifies the fixed public endpoint anonymously: no redirects/cookies/auth, bounded JSON, expected JSON/no-store/CORS/nosniff headers, and exact recognized fields/revision. Verification makes one direct GET with a three-second socket timeout. This is not an overall wall-clock deadline; the app independently keeps its three-second overall request deadline. Only verified public success returns zero. An upload attempt followed by any failure reports **unverified; live state may have changed**. There is no automatic rollback or write retry. Check the public state; restore it using a new confirmed publication if needed. Concurrent operators use last-write-wins: a later revision causes an earlier command's verification to fail honestly.

The confirmation intentionally shows public prose. Diagnostic errors never echo input paths, raw dependency output, response bodies or credentials. Receipts contain only environment/operation/revision and verified outcome. Protect terminal recordings appropriately.

## App behavior and contract

Schema 1 has exact modes `normal`, `announcement`, `maintenance`; ASCII revision tokens of 1–64 characters; notice titles up to 160 Unicode scalars and messages up to 2,400. Notices require nonblank title/message. The UTF-8 body is at most 16,384 bytes. Unknown fields are ignored and cannot configure links, API destinations, auth, features or schedules. Estimates require real zoned dates with seconds and at most six fractional digits; null and unzoned/impossible values are rejected. `normal` can omit notice text.

Status starts alongside app initialization, refreshes on foreground resume/manual retry and every 60 foreground seconds, coalesces overlapping requests, and aborts each request after three seconds. A successful maintenance read authorizes the cover for at most five minutes; unchanged success renews trust, failed reads do not. Expiry permits ordinary attempts without claiming recovery. The estimate is informational. Announcements use the existing CraftSky-themed dialog over the retained screen, with button/barrier/Escape/back dismissal. Only dismissed announcement revision is persisted; maintenance is never restored on a new process.

The cover retains routes, editors, media selections, session and current account/policy gates. Accepted work can complete under the cover; the status layer never replays a write or signs out an account. Root back is consumed during maintenance. Accessibility hides covered controls, exposes labelled status actions and restores a retained keyboard focus when it still exists. Real reader/platform-back checks remain separate from widget tests.

## Verification and release evidence

```sh
just service-status-test
just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart
just app-test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart
just app-analyze
```

Hosted verification is a manual release check after separately authorized setup. Read the preview endpoint with `curl --fail --max-time 3 -i https://preview-status.craftsky.social/app.json`; confirm the JSON, revision, no-store and CORS headers. From the supported web app origin, use browser developer tools to fetch the same endpoint with `credentials: 'omit'`, `redirect: 'error'`, `cache: 'no-store'` and `referrerPolicy: 'no-referrer'`. Inspect the request for absence of cookies/auth/device headers and confirm the body matches. These manual steps replace the standalone hosted browser harness.

For freshness evidence, authorize preview-only publication of maintenance A, announcement B and normal C. Use the same native and browser clients across changes; after each verified publication, the next successful foreground refresh must observe the replacement within the app's three-second request deadline. Confirm a disposable landing-site deploy/rollback leaves the status revision unchanged. Capture safe results only. No hosted publication or cache/CORS verification has been performed locally.

Record real VoiceOver, TalkBack and supported web-reader results for long text, both themes, supported large text scale, retry/dismiss controls, hidden editor semantics, focus restoration and browser/Android predictive back. Local semantics and fake-R2 tests do not satisfy those checks. See the implementation execution notes in `docs/changes/2026-10-08-service-status-maintenance/05-implementation-plan.md` for actual checks and remaining release gaps.
