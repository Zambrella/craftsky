# Web client audit remediation — 6 November 2026

Implements the selected items from [the audit](web-client-full-audit-2026-11-06.md).
The maintainer requested explicit-only draft saving, fixes for F02–F05, F08–F10
and F12, focused code-quality/data-boundary improvements, and lint/format gates.
F07, F11, startup-performance work, and decoded-image/storage-health work remain
out of scope by request. No deployment, source-map upload, AppView API contract,
lexicon, Flutter, or production infrastructure change is included.

## Behavior and safeguards

| Finding | Implemented behavior |
| --- | --- |
| F01 / F06 | Composer typing, media edits, navigation, teardown, and chat typing no longer autosave. Only **Save draft** stores unsent edits. Discard leaves the last explicit save alone. Submitted commands still persist their exact recovery data before a write and retain it when the outcome is uncertain. |
| F01 residual concurrency | IndexedDB compares revisions inside the write transaction. Explicit save/delete/publication cleanup reject stale editors. Content-free deletion markers prevent old editors from resurrecting completed drafts. Chat uses an origin-wide Web Lock and compares the previously loaded stored value before writing. Conflicts leave the current text in the editor. |
| F02 | The BFF streams request chunks with upstream backpressure instead of accumulating and copying the whole body. JSON routes allow 1 MiB; the two binary media paths retain a 32 MiB cap. Declared oversize is rejected before upstream fetch; actual bytes are counted for chunked requests. The 120-second deadline covers incoming body reads as well as upstream fetch. |
| F03 | Credentials use generation-specific cookie names while retaining the stable account ID. An old request can delete only its own credential generation. Background requests no longer rewrite handle cookies, account selection, or shared authorization intents. Expired selection falls back to an available account. |
| F04 | The shell and release helpers consume one authenticated-page context, including signed-in public post/profile routes. Enabled subscription actions are exercised on the feed, direct posts, and profiles. |
| F05 | API waits have a 120-second bound, including response-body reads. Video has a 15-minute overall limit, a five-minute upload limit, and 30-second polling limits. Remote image reads have a one-minute deadline and count bytes before creating a Blob. **Stop waiting** releases the publishing overlay; an interrupted record write remains uncertain and can only retry its frozen command. |
| F06 remaining storage errors | Explicit save/recovery failures preserve available IndexedDB causes. Confirmed publication stays confirmed if local cleanup fails; the original cleanup failure is reported safely. Serialized SDK tests retain the quota cause/operation and exclude private canaries. |
| F08 | Browser assertions use client-owned error copy and current operation labels, with negative assertions for raw server prose. Autosave-dependent tests now exercise explicit saves and unsaved state. Behavioral rollback and navigation checks remain. |
| F09 | CI and `just web-client-check` share `npm run test:release`: formatting, lint, types, unit tests, diagnostic fixture build, Node/Worker browser suites, and both disabled-feature cases. CI retains failure artifacts from synthetic fixture tests. After a successful gate, the script rebuilds with the original environment so the existing deployment command cannot upload a loopback diagnostic artifact. |
| F10 | Authenticated routes receive descriptive, static browser titles. Titles contain no message text, recipient identifiers, or private query values. |
| F12 | Sentry's build plugin is 11.11.0. Its updated glob dependency resolves the prior moderate advisory chain. A documented minimatch override also removes the moderate advisory introduced through the new lint tools. Four low development-tool audit entries remain in the unpatched deepmerge chain; no runtime dependency advisory was found. |

“Explicit-only drafts” is distinct from uncertain-write recovery: typing creates
no saved draft, while selecting Post/Send freezes the submitted command so a lost
response cannot cause an automatic duplicate write. New text typed while a chat
send is pending is kept only in memory. Returning to an existing scheduled post
uses a fresh local editor ID, separate from the server schedule's operation ID.

## Maintainability improvements

- Extracted publication execution and results from `ComposerWorkspace.svelte`
  into a focused module; navigation, presentation, and feed updates stay in the
  editor. Extracted shared cancellation/deadline and bounded proxy-body helpers.
- Added validation at the durable-draft/recovery boundary, where malformed data
  could otherwise drive a write. Broader API DTO generation was not introduced:
  it would require a separate shared-contract migration and is unnecessary for
  these fixes. Invalid recovery fails closed without deleting its stored data.
- Centralized authenticated page context. Chat's public share-link handoff is
  bounded, temporary memory rather than an implicit saved message draft.
- Added ESLint for JavaScript, TypeScript, and Svelte, plus executable Prettier
  format/check commands. Kept existing intentional nonreactive collections,
  static unkeyed lists, and security regexes; removed unused code/imports exposed
  by lint. `svelte-check` remains the type-analysis gate. Generated outputs and
  static vendor assets are excluded from formatting/linting.

## Validation

- Formatting, ESLint, and `svelte-check` pass with no warnings/errors.
- All 356 unit tests in 58 files pass, including stale-session/recovery races,
  original diagnostic causes, malformed storage, and stalled media requests.
- Node and Worker each pass 341 enabled authenticated cases, 11 public cases,
  and two separately configured disabled-feature cases: **708 browser test
  executions**. The two expected skips in each enabled run are the cases executed
  explicitly with the features disabled. Diagnostic consent/envelope cases run.
- All 20 development/release/deployment helper tests pass against local fixtures.
- `npm run test:release` completes successfully, including the final normal
  build. The resulting artifact has no loopback fixture DSN or public source
  maps; 46 private source maps remain available for the existing upload workflow.
  No source maps were uploaded. `git diff --check` passes.
- The dependency audit reports zero runtime vulnerabilities and four low
  development-tool entries; no moderate, high, or critical entries remain.

The release-gate browser evidence above uses Chromium and local synthetic AppView
fixtures. Hosted Cloudflare limits, physical-device behavior, WebKit/Firefox,
screen-reader announcements, and hosted Sentry symbolication remain separate
checks. The intentionally deferred audit findings remain deferred.

## Live smoke follow-up

`npm run test:e2e:live` subsequently passed using the supplied dedicated-account
credentials, this worktree's Docker AppView, and real Bluesky OAuth/PDS services.
It verified onboarding and the saved bio through fresh reads, two-image upload
with first-image rotation and decoded-image checks, publication, like persistence,
edited text and second-image removal while retaining the first image CID and like,
then deletion of only this run's post through the UI followed by a fresh 404.

The live run exposed a desktop onboarding defect: the “Profile saved” toast covered
the next-step button, and pointer hover paused its dismissal. A synthetic fixture
reproduced the failure. The shared profile editor now leaves confirmation to the
onboarding step transition; ordinary profile saves retain their toast. Three
onboarding cases pass on each of Node and Worker, covering desktop and compact
layouts. Build, lint, type checks, and formatting also pass after the fix.

An intermediate run timed out waiting for the fresh bio read. A subsequent public
PDS/local-index comparison agreed, and the full rerun passed without increasing
timeouts or weakening persistence assertions. Startup indexing lag is plausible,
but the exact timing cause was not established. Live failures now report bounded
control/status flags without profile text or account data. Private browser
artifacts remain disabled. The temporary `.env.local` link was removed; the
supplied `.env` was not changed. No deployment or source-map upload was performed.
