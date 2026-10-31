# Direct Messaging Validation Evidence

## Conversation Block account: 2026-10-31

FR-009 / NFR-006 / AC-009/023, UT-013: destructive Block account opens the
shared CraftSky public-block confirmation and dispatches the existing profile
mutation only after explicit confirmation. Peer DID and operation key verified
through the repository boundary. Success uses the themed messenger, retains the
draft/conversation and prevents replies through the accepted block overlay even
when the fetched conversation summary is stale. No automatic leave or report.

Held mutations disable duplicate action/send/accept. Request Ignore stays usable
after block acceptance (red exposed an overbroad disabled-state guard). Failures
restore controls and show shared error feedback. Cancellation, changed account,
same-account reactivation and connection loss before confirmation dispatch nothing;
late completion after account change/navigation/disposal publishes no stale success.

Logs: /private/tmp/craftsky-dm-block-{red,green,guards-red,guards-green,flow}.log.
Focused conversation suite: **26 passed** after the last subscription adjustment.
Messaging, profile block provider/actions/page, settings/modal navigation suites:
**286 passed**, 5 existing conditional skips, exit 0, 24s. Log:
/private/tmp/craftsky-dm-block-regressions.log.
Whole-app analysis clean: /private/tmp/craftsky-dm-block-analyze.log.
Formatting/git diff checks pass. Simulator hot reload succeeds; runtime errors
empty. No real block/message/report or backend/permission/migration/production
change. These tests use a fake profile repository; no live block is claimed.

## Conversation reporting: 2026-10-31

FR-010 / BR-003 / NFR-002/003/006 / AC-010/023, IT-004 / UT-005 / UT-013 /
IT-013: red Report conversation action uses the existing full-screen report
route, empty reason, optional explanation and app-bar Submit. Current capability
gates the action. Opening does not submit; explicit mocked submission returns
without leaving the conversation or losing the draft. Existing message-report,
profile-return and iOS-swipe tests pass.

Backend derives convoRef and peer DID from current authorized direct membership,
requires the existing report scope/exact child/CraftSky profiles, and submits to
the fixed Bluesky audience. Tests capture the wire body: no transcript/history
read, caller DID or CraftSky moderation row. Same operation submits once;
message/conversation target reuse conflicts and uncertain requests never replay.
Client request serialization and controller receipt-only recovery also pass.

Red/green artifacts: /private/tmp/craftsky-convo-report-{api,service,client,ui}-{red,green}.log.
Final real-Postgres race run: ./scripts/appview-test focused -race -p=1
 ./internal/chat ./internal/api — both pass, 72s and 149s.
Log: /private/tmp/craftsky-convo-report-backend-race.log.
Final Flutter messaging/settings/saved/modal suites: **231 passed**, 5 existing
conditional skips, exit 0, 19s. Log:
/private/tmp/craftsky-convo-report-regressions.log.
Whole-app analysis and formatting clean; git diff --check passes.
Simulator hot reload succeeds; no runtime errors observed.

Local development AppView rebuilt/recreated (appview and cli compile in Docker).
Local /healthz responds with DB ok/Tap connected; status remains degraded while
Tap telemetry is unknown and optional image safety is not ready. No production
mutation, OAuth widening, migration, live report/message, commit or push.
Mocked upstream/real local database checks do not prove Bluesky acceptance or
moderation outcomes. Live report testing remains excluded at the owner's request.

## Messages drawer and native conversation Back: 2026-10-31

UT-010 / IT-013, FR-003/016, NFR-006, AC-004/014/023: Messages root replaces
Back with the shell drawer in ready/setup states, sharing leading selection with
loading/error scaffolds. The real production shell test opens the drawer with
Messages selected. Children keep their existing routes/Back. iOS details edge
swipe now returns to Messages; a fresh edit is flushed and retained on reopening.
Explicit/system Back, peer-profile/report return paths and known save-error guard
remain. No navigation schema/key/branch changes were required.

Red/green logs: /private/tmp/craftsky-dm-navigation-{drawer,swipe}-{red,green}.log.
Final messaging/settings/saved/modal navigation suites: **228 passed**, 5 existing
conditional skips, exit 0, 17 seconds. Log:
/private/tmp/craftsky-dm-navigation-regressions.log.
Whole-app analysis clean: /private/tmp/craftsky-dm-navigation-analyze.log.
Formatting/diff checks pass. Hot reload succeeds; runtime errors empty.
Physical-device swipe acceptance remains owner-run; no live message/report,
production mutation, backend, dependency or persistence format change.

## Time sections and compact grouping: 2026-10-31

FR-003/005, NFR-006/008, AC-004/006/023, UT-010: real package headers cover the
first loaded authoritative message, local day changes and gaps >=5 minutes;
sender changes alone do not add dates. Headers stay centred across both bubble
alignments. Same-sender messages within five minutes use compact spacing. Normal
bubble timestamps are hidden while pending/error indicators and previews remain.
Date formatting covers Today/Yesterday, weekday/date, other years, local calendar
boundaries, 12/24h preference and large text. Prepending/removing history moves
headers correctly; existing anchor and visible-only read checks pass.

Red: /private/tmp/craftsky-dm-time-sections-red.log. Final command from app/:
fvm flutter test --no-pub test/chat — **184 passed**, exit 0, 20 seconds.
Log: /private/tmp/craftsky-dm-time-sections-regressions.log.
Whole-app analysis clean: /private/tmp/craftsky-dm-time-sections-analyze.log.
Formatting/diff checks pass. Hot reload succeeds; runtime errors empty. No real
send/report, production, dependency, API/backend or persistence change.

## Immediate composer clearing: 2026-10-31

FR-005/006, AC-006/007, UT-010/009: held-send UI test first fails because the input
still contains submitted text. Composer projection now renders that unchanged
pending revision empty while retaining durable recovery text. A newly typed
identical message first failed as an unchanged revision; controller edits now
start from the visible input. Different, identical and empty subsequent drafts
survive confirmation and local persistence. Existing failure/uncertainty recovery,
local-save rollback and pending-bubble tests all continue to pass.

Final command from app/: fvm flutter test --no-pub test/chat.
**176 passed**, exit 0, 21 seconds. Log:
/private/tmp/craftsky-dm-clear-composer-regressions.log.
Whole-app analysis clean: /private/tmp/craftsky-dm-clear-composer-analyze.log.
Formatting/diff checks pass. Hot reload succeeds; runtime errors empty.
No live sends/reports, new persistence format, API/backend or production change.

## Immediate outgoing bubbles: 2026-10-31

FR-005/006, NFR-006/008, AC-006/007, UT-010/009: pending bubbles use the standard
package renderer and spinner, suppressing time until the confirmed server message
arrives. In-memory operation/receipt associations preserve a stable local ID;
confirmed history replaces the placeholder once, without text-based inference.
Failures/uncertainty retain existing themed recovery, later edits and no replay.

Meaningful red: no bubble on held Send
(/private/tmp/craftsky-dm-pending-bubble-red.log); content/receipt ordering tests
also reproduced the need for package updateMessage operations. Access-loss red:
/private/tmp/craftsky-dm-pending-access-red.log. Recovery/ordering focused green:
36 passed (/private/tmp/craftsky-dm-pending-recovery-green.log), followed by
access fencing and the final whole messaging run: **174 passed**, exit 0,
14 seconds (/private/tmp/craftsky-dm-pending-regressions.log).
Whole-app analysis: clean (/private/tmp/craftsky-dm-pending-analyze-final.log).
Format/diff checks pass. Simulator hot reload succeeds; no runtime errors.
No real send/report, production, dependency or persistence change was performed.

## Package-owned conversation rendering: 2026-10-30

- UT-010 / NFR-008: standard SimpleTextMessage/ChatMessage wrappers, timestamps,
  grouped spacing and CraftSky ChatTheme replace custom bubbles/text rendering.
  No outlines or fabricated delivery/seen status. Existing composer, opposite
  insets, previews, visible-only reads and older-page scroll anchor remain.
- UT-013 / UT-008: long press, large-screen secondary click and keyboard actions
  open shared menus. Received text offers Report; owned/deleted messages do not.
  Copy/Open link require explicit selection and retain exact original content.
  Stale owner/history/revision/disposal actions are discarded. Compact menu root
  Navigator capture fixes the reproduced deactivated-source exception.
- Red artifacts: `/private/tmp/craftsky-dm-package-render-red.log`,
  `/private/tmp/craftsky-dm-package-menu-red.log`. Focused greens:
  `/private/tmp/craftsky-dm-package-render-green.log`,
  `/private/tmp/craftsky-dm-package-menu-final.log`.
- Final regressions: **320 passed**, exit 0, 29 seconds. Command from app/:
  `flutter test --no-pub test/chat test/profile/widgets test/router/full_profile_navigation_test.dart test/theme/craftsky_context_menu_test.dart test/shared/messaging`.
  `/private/tmp/craftsky-dm-package-regressions-final.log`.
  Whole-app analysis: no issues; `/private/tmp/craftsky-dm-package-analyze-final2.log`.
- Hot reload succeeds; runtime errors are empty. No real sends/reports, production,
  dependency change, commit or push. Device/provider acceptance remains owner-run.

## First-enable inbox handoff: 2026-10-30

- SETUP-001 / ROUTE-001: regression test first fails because the confirmed
  enablement leaves setup visible. Red: `/private/tmp/craftsky-first-enable-red.log`.
- The existing content gate now follows current status without retaining setup.
  The Messages route shows the inbox directly after confirmation. Explicit
  settings-toggle behavior and permission/account failure handling remain.
- Messaging suite: **158 passed**; final route/settings suites: **23 passed**.
  Logs: `/private/tmp/craftsky-first-enable-green.log`,
  `/private/tmp/craftsky-first-enable-final.log`.
- Whole-app analysis: no issues; `/private/tmp/craftsky-first-enable-analyze-final.log`.
  Hot reload succeeds on the connected simulator; runtime-error query is empty.
  Actual live onboarding is not replayed; no message, report or production action.

## Combined initial conversation load: 2026-10-21

- FR-003/FR-019, NFR-001/002/003/004, IT-002/IT-019: the new fixed private
  `GET /v1/chat/conversations/{id}/details` returns metadata and first history
  page under one existing capability boundary. Tests assert exactly one status,
  conversation and history RPC, one capability invocation, no mutations, opaque
  cursor serialization and continuation through the standalone history route.
- Permission, revoked grant, account enablement, membership, CraftSky profile,
  block and upstream failure tests pass. Query/body overrides are rejected;
  standard private-cache and route authentication/device policies apply.
- UT-009/IT-015/IT-017: held network and local-storage futures prove overlapping
  draft restoration and combined fetch, atomic publication, stale owner-result
  discard, preservation of the original typed failure and handling of a later
  secondary failure without duplicate/unhandled reports.
- Red artifacts: `/private/tmp/craftsky-dm-details-contract-red.log` (400 instead
  of 200) and `/private/tmp/craftsky-dm-details-concurrency-red.log` (details not
  called). Final focused green:
  `/private/tmp/craftsky-dm-details-concurrency-green.log`,
  `/private/tmp/craftsky-dm-details-final-boundaries.log`.
- Full affected backend race suites pass:
  `./scripts/appview-test focused -race ./internal/chat ./internal/api ./internal/routes`.
  `/private/tmp/craftsky-dm-details-backend-regressions.log` (exit 0).
- Flutter regression command from `app/`:
  `flutter test --no-pub test/chat test/auth test/router test/profile test/notifications test/shared/messaging`.
  **1,038 passed, 7 existing configured skips**, exit 0, 63 seconds:
  `/private/tmp/craftsky-dm-details-flutter-final.log`.
  Whole-app analysis reports **no issues found**, exit 0:
  `/private/tmp/craftsky-dm-details-analyze-final2.log`.
- Full `GOFLAGS='-p=1' ./scripts/appview-check` stops at govulncheck (exit 3):
  **GO-2026-6809**, `crypto/x509@go1.27.2`, fixed in `go1.27.3`.
  No dependencies/toolchain were changed; remaining full-gate phases did not
  run. Log `/private/tmp/craftsky-dm-details-appview-check.log`; artifacts
  `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.XKmI5ImWQp`.
- Local `./scripts/compose-dev up -d --no-deps --build appview` succeeds.
  `/healthz` is 200, database OK, Tap connected. An unauthenticated synthetic
  conversation details URL returns 401 with the standard error envelope.
  Build log: `/private/tmp/craftsky-dm-details-local-rebuild.log`.
- DTD discovery found no running debug session; reconnecting the prior URI
  failed with connection refused. Hot reload and new runtime-error inspection
  could not run. No authenticated message content was read, no send/report was
  performed, and there is no measured device speedup or production validation.

## Authenticated messaging runtime and bounded cache: 2026-10-21

- UT-006 / IT-013: session-owned repository/status, eligible unread counts and
  coordinator persist across route changes; disabled/signed-out states make no
  private content reads. App resume refreshes only lightweight state; owning the
  coordinator does not register a visible view or start hidden-page polling.
- UT-010: successful accepted/request lists reopen as the same controller with
  no extra initial fetch. At most three idle conversations retain history for
  two minutes; oldest idle history is evicted first and active reuse cancels
  expiry. Expiry clears private in-memory content while saved unsent drafts
  remain available under the existing draft lease.
- IT-017 / NFR-001 / NFR-002: synthetic private canaries clear on account switch,
  same-account reauthentication, logout, disablement, permission loss, generation
  change and authorization failure. No hidden history/list refetch follows.
  Recipient search and report forms still dispose on page exit; no report is
  submitted by these tests.
- Meaningful first reds and focused greens are recorded in
  `/private/tmp/craftsky-dm-runtime-{red,green}.log`,
  `/private/tmp/craftsky-dm-list-cache-{red,green}.log` and
  `/private/tmp/craftsky-dm-conversation-cache-{red,green}.log`.
  Initial broad verification exposed one OAuth routing fixture making an
  unmocked status request; background I/O is isolated in that navigation test.
- Final affected regression command (from `app/`):
  `flutter test --no-pub test/chat test/auth test/router test/profile test/notifications test/shared/messaging`.
  **1,017 passed, 7 existing configured skips**, exit 0, 59 seconds.
  Log: `/private/tmp/craftsky-dm-runtime-regressions-final.log`.
- Whole-app `dart analyze --fatal-infos`: **no issues found**, exit 0.
  Log: `/private/tmp/craftsky-dm-runtime-analyze-final.log`.
  Full build_runner generation succeeds; only changed provider body hashes are
  retained, preserving prior generated edits and avoiding unrelated formatting.
- Simulator hot restart succeeds; subsequent runtime-error query is empty.
  Live validation remains owner-run. No remote/production mutation, credential
  inspection, real message/report, dependency or backend/API change occurred.

## Onboarding Overlay and stuck spinner correction: 2026-10-21

- Reproduced the exact `RawTooltip` missing-Overlay assertion with the real
  themed `ScaffoldMessengerImpl`: queue a validation snackbar, activate a
  synthetic account, and hold root account initialization. The fallback
  Scaffold was outside the Router's Navigator. Red artifact:
  `/private/tmp/craftsky-dm-overlay-red.log`.
- Loading/error fallback screens now have a Navigator/Overlay; account-recovery
  modals use that Navigator. Tests cover queued feedback through loading,
  failure and recovery, plus account switching with the gate mounted in
  `MaterialApp.builder`, matching the app. Shared CraftSky feedback is retained.
- UT-006: held success/failure responses both reproduce listeners receiving
  completed sync while `busy` is still true, with no idle notification:
  `/private/tmp/craftsky-dm-spinner-listener-red.log`. The coordinator now
  publishes idle state in `finally`, fenced to the current account operation.
  Widget coverage also checks the title spinner disappears on syncing → ready.
- Final focused tests pass (23):
  `/private/tmp/craftsky-dm-onboarding-green-final.log`. Chat/auth/router/shared
  messaging regressions pass (535, 7 configured skips, 33s):
  `/private/tmp/craftsky-dm-onboarding-regressions.log`. Whole-app analysis has
  no issues: `/private/tmp/craftsky-dm-onboarding-analyze-final.log`.
  Five initial test-style diagnostics were corrected before final verification.
- Hot reload succeeded on the connected iPhone 18 Pro simulator; subsequent
  runtime-error inspection returned no errors. Actual OAuth/onboarding was not
  repeated. An aggregate read-only local sync-state query showed one ready row
  with zero failures; no account identifiers or message content were inspected.
- `InvalidHandle` comes from login validation; messaging enablement does not
  call `AuthController.signIn`. The upstream reason for that earlier validation
  response is unverified. Later layout/Hero errors are consistent with a cascade
  from the reproduced first rendering failure.
- Flutter presentation/coordinator changes only. Backend, API, grant policy,
  storage, dependencies and diagnostics are unchanged. No live message/report,
  commit, push, deployment or production mutation. Provider/device acceptance
  remains owner-run.

## Existing AppMessenger correction: 2026-10-21

UT-017 / NFR-006, FR-015 / AC-017, AC-023: replace custom inline alert banners with the existing AppMessenger. Routine sync has only a title spinner; initial loaders remain centered. State explanations and explicit uncertain-operation controls remain in the page. No alert wrapper adds layout above Scaffold.

- Red: `/private/tmp/craftsky-chat-app-messenger-red.log` finds the old custom banner where a messenger dispatch is required.
- Final Flutter: `flutter test --no-pub test/chat test/shared/messaging` passes **122 tests**, **no skips**, **9 seconds**, exit 0: `/private/tmp/craftsky-chat-messenger-tests-final.log`.
- Actual `ScaffoldMessengerImpl` tests prove the standard floating snackbar and unchanged AppBar origin. Other regressions cover retry/dismiss semantics, deduplication, covered/disposed routes, routine sync, initial/background loaders, retained editor/draft/previews and report uncertainty without replay.
- Whole-app analysis exits 0 with zero diagnostics (empty file): `/private/tmp/craftsky-chat-messenger-analyze-final.log`. Formatting and diff checks pass.
- No backend/auth/API/schema/dependency changes or remote actions in this pass; no container rebuild, commit, push, production action, live message/report or device validation. Earlier backend evidence is retained unchanged.

## Combined OAuth and setup revision: 2026-10-21

- Approved scope: request narrow messaging permissions in login/registration while base grant validation remains independent; explicit account-wide CraftSky enablement; conditional push/audience settings; safe permission recovery. Tests use synthetic identities/transports and disposable local database schemas.
- Flutter final: `flutter test --no-pub test/chat test/auth test/router test/settings test/profile` exits 0: **929 passed**, **11 configured subscription skips**, **66 seconds**. No chat test is skipped. Artifact: `/private/tmp/craftsky-combined-flutter-final.log`.
- Whole-app `dart analyze --format=machine` exits 0 with **zero diagnostics** (empty file): `/private/tmp/craftsky-combined-analyze-final.log`. Localization generation, scoped formatting and diff checks pass.
- Backend initial full auth/chat/API run: auth and chat pass under race (51.903s/71.889s); API exposes an older profile-eligibility fixture mismatch. After seeding required synthetic CraftSky profiles, its focused chat contracts pass (4.024s), then the full API suite passes under race (142.542s): `/private/tmp/craftsky-combined-api-final.log`.
- Final-source full auth race suite passes (37.002s), including combined requests, base-only and chat-capable handoffs, already-enabled-only sync recovery and stale initiating-session/generation rejection: `/private/tmp/craftsky-combined-auth-final.log`.
- Full handoff tests prove ordinary session access for both base-only and chat-capable grants, with messaging remaining disabled (3.335s under race): `/private/tmp/craftsky-combined-handoff-green.log`.
- Red/green paths, permission-loss/disposal failures, fixture corrections and the final already-enabled recovery extension are recorded in `05-implementation-plan.md`. The first broad Flutter run's five failures are retained in `/private/tmp/craftsky-combined-flutter-regressions.log`; all are resolved in the final run.
- These checks do not prove current provider consent screens, device interactions or live notification/report acceptance. No local runtime container rebuild, commit, push, production mutation or live message/report occurred. Existing owner-run manual validation remains separate.

## Boundaries

The owner approved local auth, permission, private-data and migration implementation
on 2026-10-19. No production migration, deployment, live message or report was
performed. Local tests use synthetic accounts and isolated PostgreSQL schemas.

The owner will perform MAN-001–MAN-004 personally. There is no approved Bluesky
report-testing procedure: live report acceptance is **assumed, not verified**.
Deterministic tests verify the fixed moderation destination, authorized message
reference, explanation limits, single physical attempt and uncertain receipt.
They do not establish actual Bluesky acceptance.

## Original implementation verification: 2026-10-19

| Check | Result |
|---|---|
| Focused red/green tests | Passing; chronological evidence in `05-implementation-plan.md` |
| Exact migration 000102 and down constraints | Passing isolated PostgreSQL 16 tests; no duplicate migration prefix |
| Pinned sqlc generation | Passed `bash scripts/chat-db-generate --check` |
| Standalone backend race suite | Affected packages passed; superseded by isolated gate, not completed all-package |
| Isolated `scripts/appview-check` | Passed complete controlled retry (exit 0): 49 packages / 6,498 test-subtest passes in each ordinary and race suite; zero skips/failures; image/binary/security/migration/media/health/Tap-handshake gates passed |
| Flutter analyzer | Passed whole-app `dart analyze --format=machine`: zero diagnostics after final authority fixes and generation |
| Flutter full suite | Final complete repeat passed: `flutter test --no-pub --concurrency=4 --timeout=2m`; 3,166 passed, 38 configured skips (274 seconds), after final access-loss and remote-authority races |
| Web build | Final source passed (63.1s): `flutter build web --no-pub --dart-define=CRAFTSKY_API_BASE_URL=https://appview.test` |
| Built license artifact | Final web, Android APK and iOS simulator license bundles include flutter_chat_ui, flutter_chat_core, cross_cache and Apache license text |
| iOS simulator build | Final-source simulator rebuild passed (43.5s); exported license bundle checked; no identity signing/install/run |
| Android build | Final-source debug APK rebuild passed (17.2s); exported license bundle checked |

Selected dependencies remain flutter_chat_ui 2.12.0, flutter_chat_core 2.9.1 and
cross_cache 1.1.1. Package media fetching/caching and automatic link previews are
disabled. CraftSky owns transport, draft, request, reporting and preview rules.
Host widget tests exercise narrow layouts, 2x text, keyboard/composer recovery,
visible-message watermarks, older-page scroll anchoring and route lifecycle.
These checks do not replace VoiceOver, TalkBack or browser interaction.

The isolated gate failure was a 30-second setup-context timeout applying existing
migration 000073 in one search fixture, before assertions. The exact unchanged
case passed on focused retry (21.781s). Full race/container stages of that first
gate were not reached. The fresh isolated gate ran alone with
`GOMAXPROCS=2` and passed completely. Its artifacts are in
`/private/tmp/craftsky-dm-appview-check-final`. Earlier affected-package race
results and the focused retry remain separate evidence. The pinned Tap handshake
passes; deterministic live event injection is unavailable in that Tap version.

## Manual validation retained by the owner

| Check | Outstanding evidence |
|---|---|
| MAN-001 / AT-024 | Actual granular OAuth grants, eligibility, external conversation/request history, explicit creation/send/read/accept/mute/leave, shared incoming policy, sync/reconnect; report acceptance remains an assumption |
| MAN-002 | Signed iOS/Android APNs/FCM delivery and real OS taps across foreground/background/cold start, owner switch/logout/disconnect and lost membership |
| MAN-003 | VoiceOver, TalkBack, web keyboard/screen reader, real link/card navigation and resume/reconnect |
| MAN-004 | Actual iOS/Android/web execution and selected-package interaction; compilation alone is recorded separately above |

Use consenting test accounts and the intended test AppView origin. Never paste
credentials, message bodies, recipient IDs or full provider payloads into evidence.
Record outcomes and static failure codes. A successful HTTP response, mocked
provider or injected notification callback does not establish live behavior.

## Local configuration prerequisite

Messaging intentionally remains unavailable without a dedicated operation-digest
key. `.env.local.example` declares `CHAT_OPERATION_HMAC_KEY` and
`CHAT_OPERATION_HMAC_KEY_VERSION`; `.env.local` is developer-owned and gitignored.
Compose already loads that file. Provision no key in tracked source, Flutter or
diagnostic output. The key is an unpadded base64url encoding of 32 random bytes.

To provision a **local development** key without displaying it, run this from the
repository root after checking the existing `.env.local` file:

```sh
python3 - <<'PY'
from pathlib import Path
import os, secrets
p = Path('.env.local')
lines = p.read_text().splitlines() if p.exists() else []
if any(line.startswith('CHAT_OPERATION_HMAC_KEY=') and line.partition('=')[2].strip() for line in lines):
    raise SystemExit('Existing messaging key retained; no change made')
lines = [line for line in lines if not line.startswith(('CHAT_OPERATION_HMAC_KEY=', 'CHAT_OPERATION_HMAC_KEY_VERSION='))]
lines += ['CHAT_OPERATION_HMAC_KEY=' + secrets.token_urlsafe(32), 'CHAT_OPERATION_HMAC_KEY_VERSION=1']
p.write_text('\n'.join(lines) + '\n')
os.chmod(p, 0o600)
PY
```

Restart the local Compose AppView through the normal development workflow after
configuration. This document does not authorize live messages/reports or change
production configuration. Production provisioning/release remains separate.
Keep the key stable across local restarts. On version rotation, a receipt whose
old verification key is unavailable remains protected and must never be replayed.

## Storage and cleanup

The server stores operational IDs, keyed request digests, cursors, request
generations, deduplication and delivery metadata, with no transcript or report
explanation archive. Lifetime minimal tombstones/receipts preserve request
suppression and uncertain-operation safety. Pending alerts expire after 24 hours;
terminal routing detail is pruned after 30 days in bounded batches. Disconnect
cancels work but retains these safety records. Account deletion purges CraftSky
private state and does not delete upstream chat.

Device drafts are owner/conversation-scoped, local only, serialized and fenced
against delayed saves or initialization after logout. Disconnect retains them
hidden; logout removes only that owner's drafts. They never auto-send on reconnect.

## Required-change verification: 2026-10-20

The implementation review's five required findings are addressed in the local
source. Each was reproduced with a failing boundary regression before its fix;
red/green details are in `05-implementation-plan.md`. The original review verdict
is retained as historical evidence, pending a fresh implementation review.

- IR-001: real upstream access denials traverse migrated service and registered
  handlers as safe consumer denial codes; the repository/controller test uses
  that wire response and fences held private completions while retaining drafts.
- IR-002: selection/discovery now run inside a claimed, bounded poll, with durable
  failure backoff, claim-token finalization and scheduler fairness. A real-DB
  test proves two early-failing owners cannot starve a healthy third owner.
- IR-003: upstream Retry-After survives the consumer header/repository mapping;
  a 120-second deadline holds through manual retry and background/resume.
- IR-004: failed send receipts survive HTTP on first and duplicate submission,
  with exactly one physical upstream attempt and a locally retained failed draft.
- IR-005: uncertain sends reconcile the original receipt and authorized history
  automatically and before a fresh UUID. Confirmed originals prevent another
  send; matching history never establishes delivery and later edits remain local.

Focused chat/notification tests: 210 passes, exit 0, 16 seconds;
`/private/tmp/craftsky-dm-corrections-focused.log`. The initial whole-app analysis exited 0 but retained three informational const
lints, corrected during final artifact readback; the final analyzer result is
recorded below. Initial artifact: `/private/tmp/craftsky-dm-corrections-analyze.log`.
Full backend, Flutter and supported-platform verification completed successfully;
final results appear below.
The original verification table above does not claim those older runs cover
the corrected source. Manual and live boundaries remain unchanged.

### Correction artifacts

| Finding | Behavioral red | Focused green |
|---|---|---|
| IR-001 | `/private/tmp/craftsky-dm-ir001-red.log` | `/private/tmp/craftsky-dm-ir001-green.log`, `/private/tmp/craftsky-dm-ir001-flutter.log` |
| IR-002 | `/private/tmp/craftsky-dm-ir002-red.log` | `/private/tmp/craftsky-dm-ir002-final.log` |
| IR-003 | `/private/tmp/craftsky-dm-ir003-red.log` | `/private/tmp/craftsky-dm-ir003-green.log`, `/private/tmp/craftsky-dm-ir003-flutter.log` |
| IR-004 | `/private/tmp/craftsky-dm-ir004-red.log` | `/private/tmp/craftsky-dm-ir004-green.log`, `/private/tmp/craftsky-dm-ir004-flutter.log` |
| IR-005 | `/private/tmp/craftsky-dm-ir005-red.log` | `/private/tmp/craftsky-dm-ir005-green.log`, `/private/tmp/craftsky-dm-corrections-focused.log` |

These are local test artifacts. The earlier green controller run predates the two
additional IR-005 boundary assertions; the complete focused pass includes them.

The first correction gate passed the complete ordinary suite (49 packages,
6,503 test/subtest passes, zero failed/skipped tests), but its race run caught an
ordering error in the new fairness fixture: cancellation could precede one early
failure callback (observed calls 0). That attempt has 48 passing race packages,
6,502 passing test/subtest results and one failed fixture; it did not reach later
container stages. Its artifacts remain in
`/private/tmp/craftsky-dm-corrections-check`. After synchronizing both callbacks
before cancellation, all original assertions remain and ten consecutive race
repetitions pass (8.438s): `/private/tmp/craftsky-dm-ir002-race-repeat.log`.
A fresh complete gate uses `/private/tmp/craftsky-dm-corrections-check-final`.

### Final correction-source verification

| Check | Result / local artifact |
|---|---|
| Complete isolated backend gate | Exit 0, all release gates passed; `/private/tmp/craftsky-dm-corrections-check-final` and `/private/tmp/craftsky-dm-corrections-check-final.log` |
| Backend ordinary and race suites | Each: 49 passing packages, 6,503 test/subtest passes, zero failed or skipped tests; `tests.json` / `race-tests.json` in the final gate directory |
| Complete Flutter suite | Exit 0: 3,171 passes, 38 configured skips, 285s; `/private/tmp/craftsky-dm-corrections-flutter-full.log` |
| Final controller regressions | 15 passes after test-only const cleanup; `/private/tmp/craftsky-dm-corrections-controller-final.log` |
| Final whole-app analysis | Exit 0, empty output, zero diagnostics; `/private/tmp/craftsky-dm-corrections-analyze-final.log` |
| Android debug APK | Passed, 17.3s; `/private/tmp/craftsky-dm-corrections-apk.log` |
| iOS simulator without signing | Passed, 52.6s; `/private/tmp/craftsky-dm-corrections-ios.log` |
| Web | Passed, 63.1s; `/private/tmp/craftsky-dm-corrections-web.log` |
| License artifacts | Final web/APK/iOS bundles include selected chat packages and Apache license text |
| Formatting / diff | Go formatting and `git diff --check` pass |

Builds use `CRAFTSKY_API_BASE_URL=https://appview.test` and were not installed or
run against a live service. The final backend gate includes generation/module,
static/security, binary/audit/image, migration failure, bounded media, packaged
console, migrated database/health and pinned Tap handshake checks. Tap provides
no deterministic live event injection API; no live ingestion is established.
The first correction gate failure above remains disclosed rather than replaced.
No Dart annotation, SQL/schema or dependency changed in the correction pass.

Final artifact readback found the initial analyzer's exit 0 still included three
informational const lints in a new test fixture. After making its immutable map
const, fresh analysis is empty/exit 0 and all 15 controller cases pass. No
production source changed after the complete Flutter suite or supported builds;
those checks were not repeated for the test-only const cleanup.

All five required findings are addressed locally. Fresh implementation review is
pending the owner's stage choice; the prior review is not relabeled approved.
Manual/live boundaries and operation-key provisioning remain as documented.
No commit, push, production change, deployment, live message or report occurred.

## IR-006 correction verification: 2026-10-20

The renewed review's remaining in-flight rate-limit finding is addressed locally.
The first fake-clock test observed 32 heartbeat calls where two were allowed
before a 120-second retry deadline. Scheduling metadata is now retained under
current account/generation ownership before applying the existing view-specific
publication guard. Background/view transitions no longer discard backoff; stale
view results and disposed/replaced-owner work remain fenced.

- Red: `/private/tmp/craftsky-dm-ir006-red.log`.
- First focused green: `/private/tmp/craftsky-dm-ir006-green.log` (five tests).
- Lifecycle/view/HTTP expansion: `/private/tmp/craftsky-dm-ir006-boundaries.log`
  (eight tests).
- Final coordinator coverage: ten passes, including disposal and owner changes;
  `/private/tmp/craftsky-dm-ir006-coordinator-final.log`.
- Chat/notification regression coverage: 216 passes, exit 0, 24 seconds;
  `/private/tmp/craftsky-dm-ir006-focused.log`.
- Whole-app analysis: final exit 0 and empty output, zero diagnostics;
  `/private/tmp/craftsky-dm-ir006-analyze-final.log`. Its first run had one
  informational fixture-style lint, fixed before final verification.

Full Flutter/platform verification completed successfully, as recorded below.
Backend, API/auth, permissions,
SQL/schema, dependencies and generated files are unchanged in this pass. The
recent complete backend gate above remains unchanged-source evidence and was not
rerun. The current review artifact is retained as historical correction input;
a new approval/review is not performed automatically. Manual/live scope remains
owner-run, and live Bluesky report acceptance remains assumed/unverified.

### Final IR-006 verification

| Check | Result / local artifact |
|---|---|
| Complete Flutter suite | Exit 0, 3,177 passes, 38 configured skips, 285s; `/private/tmp/craftsky-dm-ir006-flutter-full.log` |
| Whole-app analysis | Exit 0, empty output, zero diagnostics; `/private/tmp/craftsky-dm-ir006-analyze-final.log` |
| Android debug APK | Passed, 16.6s; `/private/tmp/craftsky-dm-ir006-apk.log` |
| iOS simulator without signing | Passed, 44.4s; `/private/tmp/craftsky-dm-ir006-ios.log` |
| Web | Passed, 63.8s; `/private/tmp/craftsky-dm-ir006-web.log` |
| License artifacts | Selected chat packages and Apache text verified in all three builds |
| Diff check | `git diff --check` passes |

No chat test is skipped. The complete suite and supported builds use the final
coordinator/test source after the fixture-style cleanup. Backend, auth, API,
migrations, dependencies and generated files were unchanged, so the earlier
complete backend gate is retained as existing evidence and was not rerun.
Builds used `CRAFTSKY_API_BASE_URL=https://appview.test` without installation,
runtime/device execution or live messaging. IR-006 is addressed locally; review
approval is pending a fresh review rather than inferred from passing tests.
Manual/live validation, report acceptance and key-provisioning boundaries remain
unchanged. No commit, push, deployment or production mutation was performed.

## Local manual-test configuration repair: 2026-10-20

The owner reported the enable-messaging “Continue with Bluesky” action failing
in the local development app. Read-only runtime inspection confirmed four POST
`/v1/chat/authorization` responses with HTTP 503 around 15:32 UTC. This worktree's
AppView had no CHAT_OPERATION_HMAC_KEY, and no .env.local existed. The route
composition deliberately omits its authorization starter without that key.

Provisioned a fresh dedicated 32-byte key with version 1 in ignored .env.local
(mode 0600), without printing the key. No previous key was replaced. A count-only
local DB check found zero enabled messaging accounts before the restart. Recreated
only this worktree's AppView using `scripts/compose-dev up -d --no-deps appview`;
other services/worktrees, production and application source were untouched.
Runtime inspection confirms the key is loaded and decodes to 32 bytes. The DB
health check passes and Tap is connected; initial aggregate health was degraded
while Tap telemetry had no cursor-advance sample after the restart. A subsequent
operator health check returns status ok, db ok and Tap telemetry progressing.

The confirmed local configuration blocker is removed. OAuth consent/return and
actual messaging remain owner-run validation; no account credentials, live chat
messages, reports or push deliveries were used to test this repair. Preserve the
local key across future restarts. Earlier statements that no key had been
provisioned describe prior implementation passes, before this repair.
