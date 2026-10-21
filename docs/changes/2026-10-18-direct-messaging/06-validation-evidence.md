# Direct Messaging Validation Evidence

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
