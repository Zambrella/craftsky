# TDD Implementation Plan: Direct Messaging

## App-wide foreground refresh TDD: 2026-10-31

Order: FG-001 / FR-015 session-level Feed heartbeat plus retained accepted inbox
and unread refresh; FG-002 / FR-007 real drawer/rail badge integration; FG-003 /
FR-016 lifecycle, account and view priority guards. Each uses red/green before
refactor, then messaging/navigation regressions, analysis, format/diff, simulator
reload/runtime check and evidence. Reuse existing badge/native count semantics;
no live sends, backend/permission/schema changes, commit or push. All selected
loops complete:
- FG-001 red observed zero heartbeats on Feed; green retains accepted inbox and
  native unread state and starts one session fallback at the existing 30-second
  inbox cadence. Page changes do not recreate it; resume forces fresh state.
- FG-002 existing shared ChatUnreadIcon already wired to drawer and rail; reuse
  verified through real shell integration while Feed stays selected. Counts
  update to 7, 99+ at native sentinel 100 and hidden at zero, excluding requests.
  No duplicate badge component or locally derived count added.
- FG-003 red exposed retained inbox revision 2 while conversation reached 3;
  green refreshes retained lists for conversation revisions too. The visible
  conversation outranks the session fallback independent of registration order,
  retains 4-second cadence, and returns to 30 seconds on close. Held responses
  after logout/account switch/disabled/revoked/background/dispose do not refresh
  old lists/counts. Existing non-overlap/Retry-After tests remain green.

Session scope observes the eligible coordinator, sets its lifecycle fallback and
closes its subscription at disposal. Runtime retains accepted inbox without
warming individual conversations or marking messages read. A provider retention
regression assumed a single total list fetch when requests loaded; updated to
assert one fetch per bucket because accepted is now eagerly owned by the session.
Search/report forms remain page-scoped. The router fixture now explicitly returns
null for its no-runtime override after the runtime gained a coordinator result.

Final command from app/: fvm flutter test --no-pub test/chat test/router
 test/notifications test/auth test/shared/sharing — **860 passed**, 7 existing
conditional skips, exit 0, 49s. Whole-app analysis clean; formatting and diff check
pass. Logs: /private/tmp/craftsky-foreground-{session-red,session-green,details-red,
focused,guards,badges,provider-final,regressions-final,analyze-final}.log.
Simulator hot reload and subsequent hot restart succeed, runtime errors empty.
No live message sent, automatic read or push proof; physical-device arrival and
background delivery remain unverified. No backend/API/permission/schema change,
commit, push or production action. Implementation review remains optional.

## Shared picker TDD: 2026-10-31

PICKER-001 / FR-021 / AC-031: red/green both shared picker modes, draft unchanged
for new-message mode and link-only prefill. PICKER-002 / FR-016 / AC-014: red/green
canonical child route/shared full-screen page and Close/Back/selection stack.
PICKER-003: migrate duplicate UI tests, retain controller test and verify failure,
pending operations, initial recipient and large-text/keyboard behavior. Then run
affected messaging/modal regressions, analysis, formatting, reload and evidence.
No live sends, backend/scope/schema changes, commit or push. All selected loops
complete:
- PICKER-001: red lacked ChatRecipientPicker; green shares suggestions, search,
  availability and canonical selection. Plain New message preserves the existing
  draft, while Send via chat appends/persists the exact link before opening.
- PICKER-002: route red found no shared picker. Green uses FullscreenModalPage
  around the existing FullscreenModalRoute, keeps /messages/new under Messages,
  and verifies Close/system Back and selection without a lingering picker page.
- PICKER-003: duplicate NewMessageView removed; its relevant interaction coverage
  migrated to both shared-picker modes and controller boundary tests retained.
  Tests cover initial recipient, availability/start failures, held duplicate taps,
  changed owner, dismissal and small-screen large-text/keyboard selection. A cold
  initial selection exposed recipient provider disposal before readiness; retain
  a temporary subscription throughout selection, just as for draft preparation.
  Large-text tests wait for layout after scrolling before tapping the contact.

Final affected Flutter suite: **508 passed**, 17 existing conditional skips,
exit 0, 30s. Command from app/: fvm flutter test --no-pub test/chat
 test/shared/sharing test/router/responsive_modal_navigation_test.dart
 test/router/settings_routes_test.dart test/profile/widgets/profile_actions_test.dart
 test/profile/profile_page_test.dart test/business/pages/event_detail_page_test.dart
 test/business/pages/events_settings_page_test.dart test/feed/widgets/post_card_test.dart
 test/saved_posts. Whole-app analysis clean; formatting and git diff --check pass.
Logs: /private/tmp/craftsky-picker-{regressions-final,analyze-final,large-text}.log.
Simulator hot reload succeeded; no runtime errors found. No live messages sent;
physical-device modal gestures/end-to-end sharing remain owner-run. Implementation
review is a separate optional next stage. No commit, push or production action.

## Send via chat TDD: 2026-10-31

FR-021 / AC-031: SHARE-001 draft prefill red/green, SHARE-002 picker selection/
search/suggestions/owner guards, SHARE-003 menu coverage; then messaging/sharing/
profile/event regressions, analysis, formatting and runtime discovery. No live
send, backend/scope/migration, commit or production action.

All selected loops complete. SHARE-001 red lacked prefillLink; green appends to
the visible composer projection, flushes through the existing draft store and
does not send. Existing/newer draft and pending-attempt rules stay in the shared
controller. SHARE-002/003 red lacked shared picker/menu APIs; green verifies
recent suggestions, CraftSky search without autocorrect, canonical existing/new
conversation selection, exact persisted URL, recipient denial, changed owner,
dismissal during selection and suggestion failure feedback with zero sends.
Selection testing exposed an auto-disposed conversation provider before prefill;
a temporary widget subscription now retains it through load/flush/navigation.

Every current Copy link call site uses shared linkSharingMenuItems: PostCard
(including project posts), saved posts, event details/management. Own/visitor
profile action sets carry matching callbacks for expanded/compact menus. Existing
profile/avatar/identity, skeleton and themed messaging components are reused.
Source scan confirms no other active Copy link context actions were missed.

Broad command from app/: fvm flutter test --no-pub test/chat test/shared/sharing
 test/profile/widgets/profile_actions_test.dart test/profile/profile_page_test.dart
 test/business/pages/event_detail_page_test.dart
 test/business/pages/events_settings_page_test.dart
 test/feed/widgets/post_card_test.dart test/saved_posts — **470 passed**, 12 existing
conditional skips, exit 0, 24s. After adding suggestion-load feedback, final picker
suite **7 passed**; whole-app analysis clean. Dart formatting/diff checks pass.
Logs: /private/tmp/craftsky-share-{draft-red,draft-green,ui-red,focused,regressions,
picker-final,analyze}.log. Hot reload succeeds; no runtime errors observed.
No live messages sent, backend/API/permission/schema change, commit, push or
production action. Physical-device sharing acceptance remains owner-run.

## Delete for me TDD: 2026-10-31

FR-020 / AC-030; explicit user approval includes Delete for me semantics and
normal messaging permission expansion, with no existing-session compatibility.
1. DEL-001 red/green backend permission/own-message delete and strict HTTP contract.
2. DEL-002 red/green client URL, confirmed removal, stale-page suppression and
   failure/owner/access/duplicate guards without losing drafts/outgoing state.
3. DEL-003 red/green owned-text menu, confirmation, themed feedback and stale
   actions; pending/received/deleted messages have no delete action.
4. Affected backend race/auth/routes gates, messaging tests, analysis/formatting,
   local AppView rebuild/health check, simulator reload and evidence.
No live messages deleted; no migration, commit, push or production action.
All selected loops complete:
- DEL-001 HTTP red rejected the new action; green validates identifiers, no
  body/query and authenticated no-store dispatch. Service red returned unsupported;
  green requires delete scope, enabled/current exact child, supported direct
  membership/profiles and upstream sender=owner. Tests cover requests, foreign
  members/conversations, received/missing/deleted/unsupported messages, revoked
  child, disabled access, malformed identifiers/provider output and transport
  failure. Fixed audience/body verified; no automatic replay. An older-page test
  verifies the server-retained page hint beyond the normal bounded scan.
- DEL-002 client red lacked delete APIs; green validates {deleted:true}, removes
  only after success and retains draft/cursor/other messages. Successful IDs
  suppress stale refresh/older-page content and remove just-sent outgoing mirrors.
  Held requests reject duplicate actions. Network failure retains the message;
  access loss, owner change and disposal discard stale completion. Disposal's
  existing privacy cleanup intentionally clears in-memory draft state.
- DEL-003 red lacked adapter onDelete. Green covers red long-press/secondary-click
  own-message action, recipient-retention confirmation, cancel, held request,
  success/error feedback and preserved draft. Received/deleted/unsupported/local
  pending messages cannot delete. Menu changes in owner/history/revision/disposal
  and changed message/account during confirmation discard stale actions.

The changed-message confirmation test exposed preview invalidation during widget
build. Preview refresh now hides stale content and invalidates after the frame,
guarded by current owner/conversation/message identity and lifetime. Existing
profile/post preview tests pass (FR-017 preservation).

Full gates exposed required new-route catalogue bookkeeping: exact suspension
classification (same denial as other messaging writes), route inventory and
private diagnostic metadata. All corrected; serialized local/Sentry diagnostics
verify redaction of conversation/message IDs, query credentials and opaque cause
prose while retaining the operation. No diagnostic target becomes public.

Final Flutter messaging/settings/modal-navigation suite: **242 passed**, 5 existing
conditional skips, exit 0, 19s. Final controller suite including one additional
just-sent-mirror regression: **27 passed**. Whole-app analysis, formatting and
git diff --check pass. Backend race suites pass: chat 75.279s, auth 38.633s;
API rerun 160.210s; routes final rerun 24.878s. Focused private-diagnostic race
checks pass (1.779s). Artifacts use /private/tmp/craftsky-dm-delete- prefix:
api-{red,green}, service-{red,green}, client-{red,green}, ui-{red,green},
backend-race, routes-api-race, routes-final, route-contract-final, diagnostics,
regressions, controller-final, analyze, local-build logs.

Local AppView rebuilt/recreated (appview/cli compile); /healthz reports ok with
DB ok/Tap connected and progressing. Public local OAuth metadata advertises the
new narrow method. No debug app/DTD process exists for hot reload; device UI/live
deletion remain unverified. No real messages deleted, migration, commit, push or
production action. Existing sessions receive no compatibility branch, per user.
Implementation review is the optional next stage.

## Conversation account block TDD: 2026-10-31

FR-009 / NFR-006 / AC-009/023, UT-013. User request authorizes this local
entry point into the existing account-block flow; no live block or backend change.
1. Red/green menu -> explicit public-block confirmation -> shared peer mutation,
   feedback, no leave/report, draft retained and sends disabled on acceptance.
2. Red/green cancellation/failure/held-operation and stale-account safeguards,
   covering accepted and request conversations.
3. Messaging/profile regression suites, analysis, formatting, hot reload/evidence.
All loops complete. First meaningful red found no Block account menu entry;
green uses shared red menu styling, public-block confirmation/copy and existing
toggleBlockProfileProvider for the peer. Confirmed success refreshes conversation
metadata without leaving/reporting or replacing the unsent draft. Shared accepted
block overlays prevent replies while authoritative reads catch up; unblocking
remains available through the profile/settings flow.

Held-response coverage prevents duplicate dispatch, send and accept while busy.
The request-success test exposed Ignore also being disabled by the initial block
guard; its green fix applies the post-block restriction to Accept only. Ignore
remains available after success. Failure restores controls with CraftSky error
feedback. Cancellation, changed account, same-account reactivation and connection
loss before confirmation dispatch nothing. Late completions after account change,
route coverage or disposal publish no stale success or navigation.

Logs: /private/tmp/craftsky-dm-block-{red,green,guards-red,guards-green,flow}.log.
Final focused flow suite: 26 pass after the last provider-subscription adjustment.
Broader command from app/: fvm flutter test --no-pub test/chat
 test/profile/providers/toggle_block_profile_provider_test.dart
 test/profile/widgets/profile_actions_test.dart test/profile/profile_page_test.dart
 test/router/settings_routes_test.dart
 test/router/responsive_modal_navigation_test.dart — **286 passed**, 5 existing
conditional skips, exit 0, 24 seconds. Log:
/private/tmp/craftsky-dm-block-regressions.log.
Whole-app analysis clean, formatting and diff checks pass. Simulator hot reload
succeeds; no runtime errors observed. No live block/send/report, backend, new
scope/migration, production mutation, commit or push. Implementation review is
the optional next stage; physical-device acceptance remains owner-run.

## Conversation reporting TDD: 2026-10-31

FR-010 / BR-003 / NFR-002/003/006 / AC-010/023. User approved local client/backend
extension; no live report tests, migration, OAuth/scope changes, commit or push.
Order:
1. IT-004 API target discrimination and backend reference/access/dedupe tests.
2. UT-005 client request and report-controller receipt recovery for conversation.
3. UT-013 / IT-013 red menu action/shared form and capability/navigation guards.
4. Focused real-Postgres backend race suites, messaging/navigation regressions,
   analysis, formatting, runtime reload and evidence. Release/deploy not requested.
All selected loops complete:
- IT-004 API red rejected the new subjectType; green dispatches conversation
  reports while retaining message requests and rejecting mixed/forged targets.
  Backend red returned unavailable; green derives the peer from current direct
  membership and sends only the typed convoRef through the fixed report audience.
  Real-Postgres tests cover enablement, exact-child revocation, scopes, membership,
  CraftSky profile restrictions, no transcript reads, deduplication, changed-target
  conflicts and uncertain results without replay. No CraftSky report row created.
- UT-005 client red lacked the conversation API/nullable report target. Green
  serializes subjectType without messageId/DID and reuses the existing bounded
  validation, operation identity and receipt-only recovery. 12 focused tests pass.
- UT-013 / IT-013 red had no Report conversation menu action. Green verifies red
  styling, shared full-screen form, conversation disclosure, initially empty
  reason/disabled Submit, explicit submission, no leave and retained unsent draft.
  Capability-disabled action is absent. The submission journey also exposed a
  missing MessengerScope in the test harness; it now supplies the shared service.
  All 16 route tests pass, including existing message report/profile/swipe paths.

Final backend command: ./scripts/appview-test focused -race -p=1
 ./internal/chat ./internal/api — both packages pass against isolated local
Postgres schemas (72s / 149s). Final Flutter command from app/:
fvm flutter test --no-pub test/chat test/router/settings_routes_test.dart
 test/router/saved_posts_route_test.dart
 test/router/responsive_modal_navigation_test.dart — **231 passed**, 5 existing
conditional skips, exit 0. Whole-app analysis, formatting and diff checks pass.
Artifacts: /private/tmp/craftsky-convo-report-{api,service,client,ui}-{red,green}.log;
backend-race, regressions, analyze and format logs share the same prefix.

Local AppView rebuilt/recreated with ./scripts/compose-dev up -d --no-deps
 --build appview; Docker builds both appview and cli. Local /healthz responds,
DB is ok and Tap connected; readiness is degraded with unknown Tap telemetry
and optional image safety not ready. This is not production/deployment proof.
Simulator hot reload succeeds; runtime errors empty. No live reports/messages,
new scope/migration, production action, commit or push. Owner-run live validation
remains excluded; implementation review is the optional next stage.

## Messages navigation correction TDD: 2026-10-31

UT-010 / IT-013, FR-003/016, NFR-006, AC-004/014/023:
1. Red/green Messages drawer button for ready and first-use routes/states.
2. Red/green actual iOS details swipe, draft flush/retention and existing child Back.
3. Real-shell drawer and navigation regressions, messaging suite, analysis, reload.
No route/schema/API/auth/permission change, live messages, commit or push needed.
Drawer loop green: ready/setup root tests first found Back instead of the shared
drawer button. Root now omits its child close callback; loading/error/setup and
loaded app bars use one shared leading helper. No route or shell-branch changes.
iOS swipe loop green: the real edge drag first failed to leave details because
PopScope always vetoed Back. It now allows native pop unless a local save error
is already known, and flushes the retained controller draft on a successful pop.
Explicit Back still awaits the existing guarded flush before closing. The widget
journey types a fresh draft, swipes back, verifies storage, reopens details and
verifies the same text, then checks explicit/system Back paths.

A production-router/shell test opens the actual drawer at /messages and verifies
Messages is selected. Requests/new-message/details remain existing children;
paths, navigator keys, branch retention and root-only drawer-drag policy unchanged.
Shared chat leading helper covers loaded, loading, failure and setup app bars;
child close callbacks continue to choose Back and wide roots keep no drawer.

Red/green artifacts: /private/tmp/craftsky-dm-navigation-drawer-{red,green}.log;
/private/tmp/craftsky-dm-navigation-swipe-{red,green}.log.
Real-shell test: /private/tmp/craftsky-dm-navigation-shell.log.
Final command from app/: fvm flutter test --no-pub test/chat
 test/router/settings_routes_test.dart test/router/saved_posts_route_test.dart
 test/router/responsive_modal_navigation_test.dart.
**228 passed, 5 existing conditional skips**, exit 0, 17 seconds.
Log: /private/tmp/craftsky-dm-navigation-regressions.log.
Whole-app analysis clean: /private/tmp/craftsky-dm-navigation-analyze.log.
Formatting/diff checks pass. Simulator hot reload succeeds; runtime errors empty.
No live sends/reports, backend, schema, auth/permission, dependency, commit, push
or production change. All loops and final verification complete; review optional.

## Time section TDD: 2026-10-31

FR-003/005, NFR-006/008, AC-004/006/023; UT-010.
1. Red/green adapter section boundaries, centring and hidden bubble timestamps.
2. Formatting/calendar cases and prepended-history header update; preserve
   pending/error indicators, embeds, read tracking and older-page anchoring.
3. Messaging regressions, analysis, formatting, runtime reload and evidence.
User approved the proposed first/new-day/five-minute presentation. No backend,
API, auth/permissions, migration, persistence or live sends required.

UT-010 complete: red found no Yesterday/Today section labels. ChatTimeHeader
uses ARB labels, intl dates and Material time formatting with the 12/24h preference.
Derived history metadata flags first/new-local-day/elapsed-five-minute boundaries,
so older-page changes update mounted package rows through updateMessage. Headers
use the package full-width slot; opposite bubble insets move into its padding
override, with normal section and compact same-sender spacing. All ordinary
bubble timestamps are hidden; pending/error status and confirmed previews remain.

Tests cover exact/short gaps, sender changes, local midnight, Today/Yesterday,
older dates/year, 12/24h, larger text, full-width centring and actual grouped gaps.
Prepending/removing history moves the header without duplicates. The pre-existing
pagination test's first long drag stopped 34px before the new actual maximum
extent after the first header was laid out. A final short drag now reaches the
real boundary before asserting callback count and unchanged message anchor;
no pagination production behavior or threshold was changed.

Artifacts: /private/tmp/craftsky-dm-time-sections-{red,green,focused,prepend,regressions,analyze}.log;
/private/tmp/craftsky-dm-time-pagination-{debug,green}.log.
Final fvm flutter test --no-pub test/chat: **184 passed**, exit 0, 20 seconds.
Whole-app analysis clean. Formatting/diff checks pass. Simulator hot reload
succeeds; runtime errors empty. No live messages/reports, dependencies, backend,
API, auth/permissions, migrations, persistence, commit, push or production change.
All planned loops and verification complete; implementation review is optional.

## Immediate composer clearing TDD: 2026-10-31

FR-005/006, AC-006/007, UT-010/009. First extend the held-send widget test to
require an empty input before confirmation, then cover a new draft identical to
the pending message. Implement through projected composer text and retain the
existing durable attempt and recovery flow. Run messaging tests and analysis,
then reload the running app if available. No commits/live sends authorized.
UT-010 green: held-send widget test now observes an empty input before the
response while the bubble retains the submitted text.
UT-009 green: the identical-text case first failed because it was treated as
unchanged submitted text; edits now apply against the visible composer projection.
Different, identical and deliberately empty new drafts survive confirmation and
flush/reload. Recovery tests retain original text without a new edit. The durable
attempt, receipt reducer and persistence format remain unchanged.

Artifacts: /private/tmp/craftsky-dm-clear-composer-{red,green,regressions,analyze}.log
and /private/tmp/craftsky-dm-clear-identical-{red,green}.log.
Final: **176 messaging tests passed**, exit 0, 21 seconds; whole-app analysis clean.
Formatting/diff checks pass. Hot reload succeeds; no runtime errors. No real
messages/reports, production action, backend, schema, dependency, commit or push.
All selected loops and verification are complete; review is the optional next stage.

## Immediate send bubble TDD: 2026-10-31

Approved contract: FR-005/FR-006, NFR-006/008, AC-006/007. Local presentation only.
Test order:
1. UT-010: held send -> immediate standard bubble/spinner/no time -> confirmed
   server message appears once with timestamp and later composer edits retained.
2. UT-009: failure/uncertainty, receipt-before-history and history-before-receipt
   reconciliation, restored attempts and presentation isolation.
3. Messaging regressions, analysis, formatting, runtime reload if available.

Implementation uses existing drafts/receipts plus ephemeral controller snapshots,
standard package bubbles/status, opaque stable package IDs, and no heuristic
success/replay. No backend, API, permissions, migration, dependency or persisted
message-cache change. No commit/push/live messaging requested.
UT-010 green: held response showed no bubble before the change; the existing
request/accept/send test now verifies the immediate package bubble/spinner,
hidden time, stable ID after confirmation, one final bubble and retained later
composer edits. Logs: /private/tmp/craftsky-dm-pending-bubble-{red,green}.log.
UT-009 complete: failed/uncertain attempts stop spinning with themed recovery
controls, later status confirmation retains the original operation association,
receipt-before-history keeps one placeholder without invented time, and history
before receipt never matches by text. Restored attempts remain uncertain and do
not replay. Pending rows cannot preview/report/read; confirmed rows can preview.
Access-loss test first exposed retained outgoing presentation; getter/receipt
association now respect current conversation access, including late completions.

Package update regression: setMessages alone can preserve a mounted row's old
sending content under a stable ID. Explicit updateMessage operations fix this
without replacing the standard renderer or forcing a whole-list remount.
Existing tests awaiting pumpAndSettle during an intentionally pending send now
pump finite frames; no infinite animation is expected to settle.

The local-save characterization initially expected a failed bubble, but source
and test evidence show the existing rollback removes the attempt and retains an
unsent draft. The corrected assertion verifies that behavior; no reducer,
persistence, send/retry or network behavior change was needed.

Final verification: fvm flutter test --no-pub test/chat: **174 passed**, exit 0,
14 seconds (/private/tmp/craftsky-dm-pending-regressions.log). Whole-app analysis
has no issues (/private/tmp/craftsky-dm-pending-analyze-final.log); initial import
ordering lint was corrected. Formatting and git diff --check pass. Simulator
hot reload succeeds; runtime-error query is empty. No live send/report, backend,
API, auth/permission, migration, dependency, new persisted cache, commit or push.

- [x] Approved contract, UT-010 and UT-009 loops, regressions and evidence complete.
- [x] Existing recovery and ownership behavior preserved.
- [x] Documentation updated; implementation review remains the optional next stage.

## Package-owned conversation UI execution: 2026-10-30

User approved themed package bubbles and removal of outlines, preserving
profile/post previews and moving reports into message menus. UI-only scope;
local message/privacy boundaries remain unchanged.

1. UT-010 / NFR-008: red-green standard bubbles/timestamps, package preview slot,
   theme, opposite insets, standard animated/grouped wrappers and history anchors.
2. UT-013 / FR-010, UT-008 / FR-012: red-green guarded long-press/secondary-click
   Report/Copy/Open link menu; verify no automatic fetch/report or stale action.
3. Run affected regressions, analysis and simulator hot reload/error inspection;
   record evidence and keep live acceptance separate.

### Package rendering outcomes

- UT-010 / NFR-008 complete. Meaningful red found no package SimpleTextMessage
  bubbles (`/private/tmp/craftsky-dm-package-render-red.log`). Standard
  SimpleTextMessage and ChatMessage now own bubble layout, timestamp placement,
  sender alignment, grouping spacing and animation wrappers. ChatTheme derives
  from CraftSky typography/colours/radius. Outlines are removed; timestamps stay
  on and unsupported delivery/seen icons stay off. Prior opposite-side insets,
  draft-aware composer and visible-message measurement keys remain.
- The package linkPreviewBuilder hosts the existing guarded preview widget once
  for supported text messages. Profile/post navigation, hero transitions and
  preview suppression for deleted/unknown/request messages retain their paths.
  The original-message/menu extension performs no automatic URL fetch. Removed
  the obsolete custom rich-text renderer and replaced its original-link test
  with menu interaction coverage.
- UT-013 / UT-008 complete. Meaningful red found no Report message menu after
  long press (`/private/tmp/craftsky-dm-package-menu-red.log`). Shared menus now
  provide Copy, original Open link actions and received-supported-text Report.
  Own/deleted messages cannot be reported; opening/dismissing a menu does not
  report. Keyboard Context Menu / Shift+F10 joins long press and secondary click.
  Owner, generation, message revision/content, route and widget lifetime are
  rechecked before actions. Copy occurs only on explicit selection.
- Boundary tests exposed a shared compact-menu source lifetime bug: selecting
  after the row disappears looked up a deactivated context. The menu now captures
  its root Navigator before awaiting. Tests discard stale owner, history,
  revision and disposed-source actions without framework errors. Existing shared
  menu tests pass; no menu permission or remote mutation behavior was widened.
- Focused package rendering: 4 passed; message-menu interactions/lifetimes:
  7 passed. Final messaging/profile/shared-menu/feedback regressions:
  **320 passed**, exit 0, 29 seconds:
  `/private/tmp/craftsky-dm-package-regressions-final.log`.
  Whole-app analysis: **no issues found**, exit 0:
  `/private/tmp/craftsky-dm-package-analyze-final2.log`.
  Initial import/test-style diagnostics and a package timestamp sizing-duplicate
  expectation were corrected before final checks. Formatting/diff checks pass.
- Simulator hot reload succeeds; runtime-error query returns none. No real send,
  report, clipboard read, production mutation, backend/API/schema/dependency
  change, commit or push. Local test callbacks do not prove live provider behavior.
- [x] Approved contract, both red/green loops, final checks and evidence complete.
  Implementation review remains a separate optional stage.

## First-enable inbox handoff: 2026-10-30

User requested automatic entry to the regular conversation list after first
enablement. SETUP-001 / ROUTE-001, BR-003 / FR-001 / FR-016 / AC-002 / AC-014.
Updated the existing route-level test first: red showed Messages absent after
successful enablement (`/private/tmp/craftsky-first-enable-red.log`). Removed
the gate's sticky setup state; confirmed status now selects the inbox directly.
Explicit settings still stay open after toggles; controller failure and account
fences are unchanged. No additional network request or navigation-stack entry.

Messaging regressions: **158 passed**, exit 0
(`/private/tmp/craftsky-first-enable-green.log`). Final minimal implementation:
route/settings suites **23 passed**, exit 0
(`/private/tmp/craftsky-first-enable-final.log`). Whole-app analysis passes with
no issues. Simulator hot reload succeeded; runtime-error query returned none.
No backend, schema, permissions, persistence, commit, push, live message/report
or production change. Owner-run live onboarding remains separate.

## Combined initial-load execution: 2026-10-21

User approved the proposed optimization, including the existing local/private
read boundary. No new permissions, auth-lock design, migration or production
change. FR-003/FR-019, NFR-001/002/003/004, AC-004/016/021/028.

1. IT-002 / IT-019: red-green service combined details and fixed HTTP contract;
   test one upstream sequence, private pagination and denial/failure boundaries.
2. UT-009 / IT-015 / IT-017: red-green Flutter combined API load and concurrent
   draft restore; preserve original errors, ownership and no partial publication.
3. Run affected backend/Flutter regressions and analysis, rebuild local AppView,
   hot-reload/restart client as needed, inspect runtime errors and record evidence.

### Combined initial-load outcomes

- IT-002 / IT-019: added authenticated, no-store, no-activity
  `GET /v1/chat/conversations/{id}/details`, returning `conversation` and the
  first `messages` page (fixed limit 50). A single capability boundary performs
  status, membership/profile lookup and history fetch: three upstream reads
  instead of five. Blocked conversations remain readable with sending disabled.
  No new OAuth scopes, credentials, database storage or lock policy.
- Extracted the existing history-page mapping/cursor implementation for both
  routes. Tests prove that the combined response's opaque continuation resumes
  through the existing history endpoint. Independent history reads retain their
  full authorization/membership checks. Invalid input, missing authority,
  revoked child grants, missing profiles/nonmembers and upstream/malformed
  failures return no successful partial payload.
- UT-009 / IT-015 / IT-017: the Flutter initial load uses the combined response
  and starts local draft restoration concurrently. Publication awaits both;
  owner/epoch guards, drafts and original typed errors/stacks remain intact.
  Secondary late errors are consumed without duplicate reporting. Background
  refresh and pagination retain their existing paths.
- Meaningful reds: the new HTTP action initially returned 400 instead of 200
  (`/private/tmp/craftsky-dm-details-contract-red.log`), and the controller did
  not invoke details (`/private/tmp/craftsky-dm-details-concurrency-red.log`).
  Focused greens pass. The stale-load fixture now holds the combined response,
  matching the new boundary while retaining private-canary assertions.
- Full affected backend packages pass under race:
  `./scripts/appview-test focused -race ./internal/chat ./internal/api ./internal/routes`.
  Log: `/private/tmp/craftsky-dm-details-backend-regressions.log`.
  Final session-count/continuation tests also pass:
  `/private/tmp/craftsky-dm-details-final-boundaries.log`.
- Flutter chat/auth/router/profile/notifications/shared-messaging regressions:
  **1,038 passed, 7 existing configured skips**, exit 0, 63 seconds.
  `/private/tmp/craftsky-dm-details-flutter-final.log`.
  Whole-app analysis: **no issues found**, exit 0:
  `/private/tmp/craftsky-dm-details-analyze-final2.log`.
  Formatting and diff checks pass; no generated-code changes needed in this pass.
- Full `GOFLAGS='-p=1' ./scripts/appview-check` is **blocked**, exit 3 at
  govulncheck: **GO-2026-6809**, pinned `crypto/x509@go1.27.2`, fixed in
  `go1.27.3`. This existing toolchain vulnerability is outside the approved
  optimization; no toolchain upgrade was made. Later full-gate phases did not
  run. Log: `/private/tmp/craftsky-dm-details-appview-check.log`; artifacts:
  `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.XKmI5ImWQp`.
- Rebuilt/restarted only the local development AppView. `/healthz` returns 200
  with database OK and Tap connected; an unauthenticated synthetic details URL
  returns the standard 401 envelope. No real conversation was requested.
  Log: `/private/tmp/craftsky-dm-details-local-rebuild.log`.
- Client hot reload was attempted after the backend rebuild, but DTD discovery
  found no running debug session and the previously connected URI refused the
  connection. The Flutter update therefore has no new simulator runtime-error
  receipt. Actual device timing and live acceptance remain owner-run.
- [x] Approved contract, red/green loops, private-data/draft guards and affected
  regression checks complete; evidence recorded and read back.
- [x] Local backend rebuilt and health/authentication boundary checked.
- [ ] Full AppView release gate: blocked by the existing pinned Go vulnerability.
- [ ] Client hot reload/runtime inspection: no active debug session available.
- No commit, push, production deployment, migration, real message or report.
  Implementation review remains a separate optional workflow stage.

## Conversation header profile action: 2026-10-21

UT-010 / IT-013, FR-003 / AC-023: test avatar and identity taps from real chat
routes, asserting peer DID and compact presentation request, then Back restores
the conversation/draft. Implement using showUserProfileCard and existing account
navigation guards; verify focused chat and compact-profile regressions.

### Header profile action outcomes

Meaningful red: tapping the avatar left the conversation visible instead of
opening the profile route. `/private/tmp/craftsky-dm-header-profile-red.log`.
Green: the avatar and name/handle share a labelled, keyboard-accessible InkWell
that uses the existing showUserProfileCard helper for the peer DID, fenced to
the active account and current route. The background loader stays outside the
target. The routing test verifies the compact presentation request for both
targets and returns to the same unsent draft after each visit.
`/private/tmp/craftsky-dm-header-profile-green.log`.

Final chat and existing compact-profile regressions: **176 passed**, exit 0,
12 seconds; `/private/tmp/craftsky-dm-header-profile-regressions.log`. Scoped
analysis passes with no issues; `/private/tmp/craftsky-dm-header-profile-analyze.log`.
Formatting/diff checks pass. Simulator hot reload succeeds. No new profile
transport, route definition, data/API, permission, dependency, live messaging,
commit, push or deployment action. Implementation review remains optional.

## Request-action execution: 2026-10-21

User explicitly requested direct Ignore-as-leave and removal of the request
explainer. FR-008/FR-009, AC-009/AC-012, AT-003/AT-009/UT-010.

1. Red: exercise Ignore from the real request details view; verify no acceptance,
   send or confirmation and no premature/double close during held leave.
2. Green: add localized responsive Accept/Ignore actions and guarded close.
3. Verify failure and stale view/owner/capability guards, accepted composer,
   narrow large-text layout and nearby messaging regressions; record evidence.

### Request-action outcomes

- Meaningful red: Ignore was absent from the request details view.
  `/private/tmp/craftsky-dm-ignore-red.log`. Focused green passes:
  `/private/tmp/craftsky-dm-ignore-green.log`.
- Removed the explainer from rendering. Added localized Ignore with the existing
  secondary ChunkyButton style; Accept request remains primary. Controls share a
  row normally and stack on narrow/large-text layouts. Ignore alone shows its
  progress while both actions are disabled during leave.
- Ignore directly invokes the existing controller leave operation, without a
  confirmation dialog. Close requires success, the initiating controller, current
  route, mounted view and continuing access. More-menu Leave keeps its dialog.
- Failure preserves the request and themed error feedback; no acceptance or send
  occurs. Tests cover duplicate taps, real account switching, covered/disposed
  routes, capability loss and the existing acceptance-to-composer flow.
  The first owner-switch fixture retained a deliberately stale busy view and
  timed out; the corrected fixture switches the actual registry and exercises
  the production gate that removes that view.
- Final chat/shared-feedback regressions: **173 passed**, exit 0, 11 seconds.
  `/private/tmp/craftsky-dm-ignore-regressions-final.log`.
  Whole-app analysis: **no issues found**, exit 0.
  `/private/tmp/craftsky-dm-ignore-analyze-final.log`.
- Localization generation, formatting and diff checks pass. Simulator hot reload
  succeeded; no live Ignore, message or report was executed. No API, backend,
  permission, migration, dependency, commit, push or deployment change.
- [x] Approved revision, red/green, failure/lifetime checks and evidence complete.
- [x] Implementation review remains a separate optional stage.

## Active-account runtime/cache execution: 2026-10-21

Approved by the maintainer's request to implement the recommended account-level
provider lifetimes. Existing local privacy/lifecycle authorization applies; no
remote or production action is included. Follow the current approved revisions
of requirements, acceptance tests and coding plan.

1. UT-006 / IT-013: red-green authenticated ownership of lightweight providers,
   with no hidden-view polling and no private data work while disabled/signed out.
2. UT-010 / IT-017 / NFR-002: red-green retained lists, bounded idle conversation
   history, resubscription/expiry and account/access/generation cleanup.
3. Verify resume, lazy form disposal, drafts and existing scheduling boundaries;
   run affected Flutter suites and analysis, apply to simulator, record evidence.

### Execution outcomes

- **UT-006 / IT-013 complete:** the initial red exposed missing session ownership
  of lightweight state. `ChatSessionScope` now owns status/repository and eligible
  counts/coordinator independently of messaging routes. Resume refreshes status
  and existing counts. Moving between non-chat pages does not refetch; disabled
  and signed-out states make no private reads, and hidden views do not poll.
  Artifacts: `/private/tmp/craftsky-dm-runtime-{red,green}.log`.
- **UT-010 complete:** red showed chats/requests rebuilding on reopen and loaded
  history disappearing immediately. Successful lists now retain account-owned
  links after first use. Conversations retain at most three idle controllers for
  two minutes; oldest idle entries expire first, resubscription cancels expiry,
  and active listeners stay loaded. Local draft persistence is unchanged.
  Artifacts: `/private/tmp/craftsky-dm-list-cache-{red,green}.log`,
  `/private/tmp/craftsky-dm-conversation-cache-{red,green}.log`.
- **IT-017 / NFR-001 / NFR-002 complete:** private canaries clear on owner switch,
  same-account reauthentication, logout, disablement, permission loss, generation
  change and authorization failure, without hidden content refetch. The active
  cache owner watches the exact lease and generation so paused content providers
  cannot retain private data beyond their authority. Search/report forms dispose
  independently while the session runtime remains alive.
- Fixture corrections: widget lifecycle tests now traverse the real
  inactive/hidden/paused/resumed sequence; fake time advances to run Riverpod's
  scheduled disposal after link release. The OAuth handoff navigation test
  isolates messaging background I/O instead of issuing an unmocked status read.
  These corrections do not weaken production account or capability checks.
- Full code generation succeeded. Preserved existing generated changes and
  discarded incidental generator formatting; only the two changed provider body
  hashes are new. No generated API or routing signature changed in this pass.
- Final verification: **1,017 tests pass, 7 existing configured skips**, exit 0,
  59 seconds across chat, auth, router, profile, notifications and shared
  messaging. Whole-app `dart analyze --fatal-infos` exits 0: no issues found.
  Artifacts: `/private/tmp/craftsky-dm-runtime-regressions-final.log`,
  `/private/tmp/craftsky-dm-runtime-analyze-final.log`.
- Simulator hot restart succeeded to install the new shell wrapper; subsequent
  Dart runtime-error inspection returned no errors. This is local runtime
  evidence, not a live OAuth/message/report or physical-device acceptance result.

- [x] Approved requirements, acceptance cases and design recorded.
- [x] Runtime, retention and eviction red-green loops complete.
- [x] Active lease/access cleanup, lazy forms, draft and polling regressions pass.
- [x] Generation, formatting, affected regressions and whole-app analysis pass.
- [x] Existing worktree edits preserved; no commit, push, deployment, backend/API,
  migration, permission policy change or live message/report action.
- [x] Implementation evidence recorded and read back. Implementation review is
  the next optional workflow stage; it has not been performed for this pass.

## Onboarding runtime correction: 2026-10-21

Reproduced the reported missing Overlay with a queued CraftSky validation
snackbar crossing root account initialization. Added a fallback Navigator for
loading/error screens and routed account-recovery modals through it. Regression
coverage matches the actual `MaterialApp.builder` placement, including failure,
recovery and account switching. Shared feedback rendering is retained.

Separately reproduced UT-006's stale published `busy` state on both heartbeat
success and failure. Notify current listeners after clearing `busy` in `finally`;
retain scheduling, backoff and account fencing. Focused tests pass (23), broader
auth/chat/router/feedback regressions pass (535, 7 configured skips), and whole-app
analysis is clean. Simulator hot reload succeeded and subsequent runtime-error
inspection was empty; actual OAuth/onboarding remains owner-run. Detailed
red/green artifacts and scope boundaries are in `06-validation-evidence.md`.

## AppMessenger correction: 2026-10-21

UIP-028 / NFR-006, FR-015, AC-017, AC-023: the maintainer's screenshot showed the custom sync banner above the page Scaffold, leaving a black region and moving the AppBar. The preceding pass reused snackbar styling in a separate inline renderer; it did not complete the requested migration to the existing message delivery system.

Remove the bespoke CraftskyMessageBanner renderer and its renderer-only tests. Chat alerts now dispatch through the existing AppMessenger/context.showInfo/showWarning/showError flow, using a non-rendering ChatMessageEffect to publish controller state after layout. It deduplicates unchanged messages, ignores covered/disposed routes, guards stale retry actions and never dismisses another feature's message. Authorization failures use the existing guarded async message path directly.

Apply this across synchronization/degraded-state warnings, settings/recipient errors, draft saves, send failures/uncertainty and report outcomes. Request instructions, incoming-policy/permission states and durable send/report results remain normal in-page text with their existing controls. Uncertain operations retain their explicit retry checks; no automatic remote action is added.

Routine synchronization emits no alert. An inherited loading value supplies only the existing 16px title spinner. Initial inbox/conversation loads keep their centered spinner without an extra title loader. No feedback wrapper adds geometry above the Scaffold; the composer and keyboard state remain stable.

Observed red: the synchronization test still found the custom inline banner. Green dispatches one warning through the recording AppMessenger with the existing retry action, preserving the same editor/controller. The real ScaffoldMessengerImpl test verifies the floating themed snackbar and an unchanged AppBar origin. Additional regressions cover routine syncing, covered-route/disposal guards, deduplication, draft retention, private previews, request acceptance and uncertain report status without replay.

Verification: all **122 chat/shared-messenger tests pass**, exit 0, 9 seconds: `/private/tmp/craftsky-chat-messenger-tests.log`. The first runs exposed a duplicate test-builder argument, missing test localizations, obsolete timestamp/copy assertions, and an initial/background spinner overlap; all are corrected in the green suite. Final-source verification passes the same 122 tests again after cleanup: `/private/tmp/craftsky-chat-messenger-tests-final.log`. Whole-app analysis exits 0 with an empty diagnostics file: `/private/tmp/craftsky-chat-messenger-analyze-final.log`. Formatting and `git diff --check` pass. No backend, auth/permission, API, migration, dependency or live action changed in this correction.

## Combined OAuth/setup execution: 2026-10-21

Inputs: approved 2026-10-21 revisions of requirements, acceptance tests and coding plan. Local auth/permission changes explicitly authorized; existing unrelated work preserved. No commit/push/deployment or live message/report.

| Order | Test | Requirements / AC | State |
|---|---|---|---|
| 1 | AUTH-001 | BR-002, RULE-004, NFR-001 / AC-002, AC-019 | Complete |
| 2 | AUTH-002 | FR-001, FR-016, RULE-004 / AC-002, AC-014 | Complete |
| 3 | SETUP-001 | BR-003, FR-001, FR-011, FR-014, FR-016 / AC-002, AC-011, AC-015 | Complete |
| 4 | ROUTE-001 | FR-001, FR-016, NFR-001, NFR-006 / AC-002, AC-014, AC-016, AC-023 | Complete |

First red: real consumer login/registration PAR omits chat scopes. Scope request and base grant validation stay distinct. Then recovery confirmation independence, settings enablement, and route journey. Relevant existing tests provide the ordinary-session, policy-preservation, background/delivery fencing and draft regressions.

AUTH-001 red: both real PAR requests omitted the chat status grant. Green requests the narrow union in login/registration only; Config.Scopes remains the required base. Combined request, semantic capability and base scope tests pass (2.042s). An initial test read lacked its required owner fence; the fixture now reads persisted request JSON directly without weakening production fences. Logs: `/private/tmp/craftsky-combined-auth-{red,green}.log`.

AUTH-002 red: permission recovery changed the connection to enabled/generation 2/push false for both initial choices. Green validates existing same-owner/device/generation authority without changing enablement, generation or push preference; disabled sync remains paused. Recovery and stale authority/handoff regressions pass (9.846s). Logs: `/private/tmp/craftsky-combined-recovery-{red,green}.log`.

SETUP-001 red: disabled Messages rendered the old authorization screen rather than Direct messages setup. Green reuses one settings surface, makes the account switch reversible, hides preferences while off, preserves push/audience choices and keeps setup visible until Open messages. Initial, iOS Material-switch, background-loader and exact-policy regressions pass. The route assertion deliberately checks persisted effects rather than incidental read counts. Logs: `/private/tmp/craftsky-combined-setup-{red,green}.log`, `/private/tmp/craftsky-combined-settings-tests.log`.

ROUTE-001: extended the original setup journey through enable, disable, re-enable and explicit Open messages; cold enabled visits open Messages directly. A separate meaningful red found permission loss during post-enable preference loading republishing a stale ready status. Green publishes the latest classified status. The pending/disposed-owner case also found a preference request starting after disposal; explicit current-lifetime checks now prevent subsequent private fetches. Both guards retain already-submitted remote outcomes without replay or stale publication. Logs: `/private/tmp/craftsky-combined-access-loss-{red,green}.log`, `/private/tmp/craftsky-combined-flutter-regressions.log` (initial disposal failure), `/private/tmp/craftsky-combined-boundary-regressions.log` (green).

Broader regressions exposed prior follow-up fixture drift: API rejected-send fixtures lacked the newly required CraftSky profiles, and route fixtures used old Messages/Settings/Saved labels or indices. Updated only those synthetic profiles/assertions; production eligibility and navigation behavior remain intact. The three API contract tests pass under race (4.024s); the settings/disposal/router group passes 41 tests with 5 configured subscription skips. Initial broad auth/chat packages pass under race (51.903s/71.889s); the corrected full API package passes under race (142.542s).

Final Flutter verification passes **929 tests**, with **11 pre-existing configured subscription skips**, in 66 seconds, exit 0. It covers chat, auth, router, settings and profile suites. Whole-app static analysis exits 0 with a zero-byte diagnostics file. Logs: `/private/tmp/craftsky-combined-flutter-final.log`, `/private/tmp/craftsky-combined-analyze-final.log`. Localization generation, scoped Dart formatting, Go formatting and diff checks pass. No provider/router generation, dependency or schema change was needed.

AUTH-001 complete ordinary handoff coverage also passes for base-only and chat-capable grants (3.335s under race): both get usable CraftSky child sessions and neither enables messaging. AUTH-002's final extension caught a previously enabled account staying paused after permission recovery. Its red is `/private/tmp/craftsky-combined-enabled-recovery-red.log`; the minimum change resumes retained sync only for an already-enabled account. Enablement, generation, push choice and cursor remain preserved, and disabled accounts stay paused. The final full auth race suite passes (37.002s): `/private/tmp/craftsky-combined-auth-final.log`.

- [x] Approved requirements/tests/design revisions recorded before code changes; all four vertical loops executed and notes read back.
- [x] Preferences remain conditional, shared audience is unchanged on toggles, and hidden local drafts retain existing account fencing.
- [x] Initial centered and background title loaders, Material switches and CraftSky themed feedback retained.
- [x] Flutter regression suites and whole-app analysis pass; full chat/API race suites pass.
- [x] Final full auth race suite passes on the final source; stale-generation, missing-purpose, session/permission loss, base-only access and both recovery choices remain covered.
- [x] No commit, push, deployment, runtime AppView rebuild, production change or live message/report. Owner-run device/OAuth/provider validation remains separate; report acceptance remains unverified.
- [x] Implementation review remains a separate next-stage choice, not automatically performed or relabeled approved.

## Profile action follow-up: 2026-10-20

Explicit maintainer direction is recorded in the approved revisions of requirements, acceptance tests and coding plan. No backend, permission, migration or live-message action is in scope.

| Order | Test | Requirements / AC | State |
|---|---|---|---|
| 1 | UT-015 | NFR-006, NFR-008 / AC-023 | Complete: expanded/collapsed shared menu |
| 2 | UT-016 | FR-004, NFR-006, NFR-008 / AC-005, AC-023, AC-027 | Complete: themed denial/failure feedback and allowed navigation |


UT-015 red found standalone mute controls and missing menu entries. Green uses one shared ProfileMoreActionsButton in the action row and collapsed AppBar, preserving state-dependent labels and guards. Profile action tests pass; actual expanded/collapsed mute-and-Unmute page test passes. Logs: `/private/tmp/craftsky-profile-menu-{red,green,collapsed-green}.log`. The first page regression exposed the separate collapsed mute control, which is now removed; the menu is built only when that header control is visible.

UT-016 red showed the denied recipient still had a disabled Message button. Green keeps it tappable and uses the existing guarded context.showInfo call: known denial shows the neutral recipient-unavailable copy, failed/incomplete checks show generic unavailability, and only a confirmed available recipient opens selection. Exact account/consent/busy/block guards remain. The widget test covers denial, provider failure and recovery to available without hidden create/send calls. Provider retry is disabled only in the test harness to make the failure case deterministic. Logs: `/private/tmp/craftsky-profile-message-{red,green}.log`.

Final profile follow-up verification: `flutter test --no-pub test/chat test/profile/widgets/profile_actions_test.dart test/profile/profile_page_test.dart` passes **139 tests**, no failures/skips, 12 seconds. After test-fixture lint cleanup, the two profile suites plus Message button suite pass **49 tests** on the final files. `dart analyze --format=machine lib/chat lib/profile test/chat test/profile` exits 0 with no findings. Formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-profile-actions-{tests-final,focused-final,analyze-final}.log`.

- [x] UT-015 and UT-016 completed with meaningful red/green evidence.
- [x] Shared More menu retains mute/unmute callbacks, busy/block eligibility, share/report/block/copy and responsive presentation; actual page tests include collapsed-header use.
- [x] Message denial and provider failure use the existing AppMessenger informational path without navigation; available recipients retain explicit selection flow.
- [x] Requirements, acceptance, coding/implementation and UI notes updated and read back.
- [x] No backend/API/permission/dependency changes, migration, commit, deployment or live message/report. Native/device/live checks were not repeated for these client-only changes; final implementation review remains separate.

## Follow-up execution: 2026-10-20 CraftSky eligibility and chat UI

Inputs are the explicit maintainer revision now recorded at the top of `01-requirements.md`, `02-acceptance-tests.md` and `04-coding-plan.md`. Earlier completed loops remain historical evidence; external-peer/native-count/tab expectations are superseded. Existing uncommitted UI polish is retained. No commit, deployment, live message/report or migration is requested.

| Order | Test ID | Requirements | AC | State |
|---|---|---|---|---|
| 1 | IT-021 | BR-001, FR-002, FR-003, FR-004, FR-018, NFR-001 | AC-030 | Complete: filtered pagination and profile-removal denial |
| 2 | UT-012 | FR-007, RULE-002, RULE-005, NFR-004 | AC-008, AC-031 | Complete: filtered/capped/bounded counts and narrower permission |
| 3 | IT-022 | FR-012, FR-013 | AC-030 | Complete: sync/delivery/open eligibility and baseline lifetime preservation |
| 4 | UT-013 | FR-002, FR-015, FR-018, NFR-006, NFR-008 | AC-023, AC-031 | Complete: navigation, themed refresh, loading and composer |

IT-021: list-filter red showed both buckets exposing the hidden first-page peer (one RPC instead of continuing). Green uses a batched sqlc profile query and preserves opaque continuation. Known-ID/profile-removal red exposed history, actions, start and send; green centralizes consumer access through `supportedConversation`, checks availability/start, and retains existing owner/capability fencing. Nearby pagination/history/action/start/send/open tests pass (9.733s). Logs: `/private/tmp/craftsky-dm-{eligibility-red,eligibility-green,profile-access-red,profile-access-green}.log`. Existing positive fixtures now seed public CraftSky profiles without creating peer OAuth sessions.

UT-012: red returned global `100/100` with zero list calls where filtered totals are `2/3`. Green enumerates authoritative lists, batches profile eligibility, deduplicates IDs, counts all pending requests (including read/muted), excludes accepted muted/blocked/disabled peers, caps at 100, and rejects cycles, provider failure and 20-page budget exhaustion without publishing partial counts. Overall count handling is bounded to 20 seconds. A further red showed an unused global-count grant preventing readiness; it is now removed from optional permissions and read capability checks, narrowing authority without changing existing grant compatibility. Count/capability/status tests pass (8.248s). Logs: `/private/tmp/craftsky-dm-{counts-red,count-scope-red,counts-green}.log`.

IT-022: red sent both queued notifications despite profile removal at initial/final check. Green routes both delivery checks and incremental sync through the same profile boundary; recovery pages use batched filtering. Hidden events still advance the cursor. Further red proved that dropping hidden baseline state replayed pre-enable history when the peer later joined CraftSky. Final green retains only existing private opaque baseline dedupe IDs/request-lifetime flags (no peer identity or message body) so later membership neither replays accepted history nor resets baseline request suppression, even for a later request message. These tombstones never populate views/counts or authorize push. Focused join/recovery/delivery tests pass under race (6.941s). Notification-open profile denial and either-direction local-block badge tests also pass under race (4.111s). Logs: `/private/tmp/craftsky-dm-profile-{push-red,push-green,join-red,join-green,open-block}.log`.

UT-013: navigation red found the old tab bar and no FAB. Green uses the existing typed request route, pending badge and branded FAB; request return/cold fallback resolves to chats. Composer red lacked the requested hint and title refresh feedback. Green hides the visible field label (retaining its accessible label), shows localized `Message` as hint and centers the send control; controlled refresh retains message/editor content and shows a 16px stitch indicator next to the title. Both accepted/request views pass actual yarn pull gestures, retained-row and title-spinner assertions at 320px / 2x text scale. Remaining loading surfaces use the same title pattern, and history pagination adds no loading geometry. Logs: `/private/tmp/craftsky-dm-{navigation-red,navigation-green,composer-red,composer-green,yarn-refresh}.log`.

Final Flutter verification: `flutter test --no-pub test/chat test/notifications test/router/notification_open_routing_test.dart test/theme/yarn_refresh_scroll_view_test.dart` passes **233 tests, no failures or skips**, 19 seconds. Whole-app `dart analyze --format=machine` exits 0 with no findings. SQL generation check passes. Three synthetic capture tests pass; visually inspected chats, requests, conversation composer and retained-history refresh using bundled fonts; temporary capture harness removed. Images: `/private/tmp/craftsky-dm-followup-{inbox-accepted,inbox-request,conversation,refresh}.png`. No real account or transcript data used.

Final backend verification: complete `./internal/chat` race suite passes on the final source (67.196s); `./internal/auth` (42.801s) and `./internal/routes` (29.306s) race suites pass on their unchanged final source. The first broad run found a duplicate positive-profile fixture after shared fixture seeding; corrected the fixture with `ON CONFLICT DO NOTHING` and reran successfully. Logs: `/private/tmp/craftsky-dm-followup-{chat-final,backend,backend-final}.log`. The optional permission reduction changes no stored OAuth credential or migration and requests no wider authority.

Runtime: rebuilt/restarted only the local development AppView with `scripts/compose-dev up -d --build --no-deps appview`, including the final baseline fix. Final `/healthz` returns HTTP 200, status `ok`, database `ok`, Tap connected. Build log: `/private/tmp/craftsky-dm-followup-local-build-final.log`. No production deployment or live message/report was performed. Reload the app client to load its new widgets.

Completion checklist:

- [x] IT-021 / UT-012 / IT-022 / UT-013 completed with meaningful red/green evidence and regressions.
- [x] Revised user instructions and changed product behavior recorded in requirements, acceptance cases and coding plan.
- [x] 233 focused Flutter tests pass; whole-app analysis has no findings; SQL generation check and `git diff --check` pass.
- [x] Chat/auth/routes race suites pass; synthetic screens inspected; local runtime rebuilt and healthy.
- [x] No commit/push, dependency addition, migration, production action or live messaging/reporting performed.
- [x] Implementation and UI notes updated and read back before handoff.

The local scope approval already given and the explicit behavior instructions authorize this narrower eligibility implementation. Physical-device, assistive-technology and live/provider validation remain maintainer-owned. Full Flutter suite and supported-platform native rebuilds were not repeated for these client presentation changes; no current-source device proof is claimed. Final implementation review remains a separate next-stage choice.

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (Approved with notes)
- Coding plan: `04-coding-plan.md`

## Authorization and Execution State
- Owner invoked implement-tdd on 2026-10-19 and explicitly approved the planned local auth/permission, private-data and migration implementation.
- Current worktree only. No commits or pushes authorized. Production mutations and live messages/reports require separate authorization.
- First loop: UT-001, `TestChatOptionalScopesPreserveBaseSession`, then semantic RPC capability cases; IT-015 follows.
- Test order mirrors coding-plan section 9; repeated IDs are first listed once, with later slices extending their assertions before final completion. Acceptance IDs summarize their owning lower-level suites rather than duplicate umbrella tests.
- Status: local implementation, selected TDD loops, full regression gates, supported-platform compilation and evidence reconciliation are complete. Actual provider/device validation is deferred to the owner; report acceptance is assumed and unverified. Implementation review remains a separate next-stage choice.

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one failing behavior test before its minimum implementation.
- Run the smallest relevant test and confirm a meaningful missing-behavior failure.
- Refactor only while green; keep unrelated changes out of scope.
- Carry privacy and owner-fencing assertions in each owning loop.
- Distinguish unit/mock, migrated PostgreSQL, actual provider, platform and OS-push evidence.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State | Status |
|---|---|---|---|---|---|
| 1 | UT-001 | FR-001, RULE-004, NFR-001 | AC-002, AC-019 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 2 | IT-015 | RULE-004, NFR-001 | AC-019 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 3 | AT-002 | BR-002, BR-003, FR-001 | AC-002 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 4 | AT-019 | RULE-004, NFR-001 | AC-019 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 5 | REG-001 | BR-002, FR-016, RULE-004 | AC-002, AC-014, AC-019 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 6 | IT-005 | BR-002, FR-001, FR-013, FR-016 | AC-002, AC-014 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 7 | IT-013 | FR-014, FR-016, NFR-001 | AC-016, AC-019 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 8 | IT-019 | NFR-003 | AC-021 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 9 | AT-014 | BR-002, FR-001, FR-013, FR-016 | AC-014 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 10 | AT-016 | FR-014, FR-016, NFR-001 | AC-016 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 11 | AT-021 | NFR-003 | AC-021 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 12 | UT-002 | FR-005 | AC-006 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 13 | UT-005 | FR-010 | AC-010 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 14 | UT-011 | BR-003 | AC-002, AC-010 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 15 | IT-004 | BR-003, FR-010 | AC-010 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 16 | AT-006 | FR-005 | AC-006 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 17 | AT-010 | BR-003, FR-010 | AC-010 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 18 | IT-006 | FR-005, FR-006 | AC-006, AC-007 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 19 | AT-007 | FR-006 | AC-007 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 20 | IT-001 | BR-001, FR-002, FR-008 | AC-001, AC-003 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 21 | IT-002 | FR-003 | AC-004 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 22 | UT-003 | FR-003 | AC-004 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 23 | IT-003 | BR-001, FR-004, FR-018 | AC-005, AC-027 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 24 | AT-001 | BR-001, FR-002 | AC-001 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 25 | AT-004 | FR-003 | AC-004 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 26 | AT-005 | BR-001, FR-004 | AC-005 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 27 | AT-027 | FR-018 | AC-027 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 28 | UT-004 | FR-007, RULE-002, RULE-005 | AC-008 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 29 | IT-007 | FR-007, FR-008, FR-009, RULE-002 | AC-003, AC-008, AC-009 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 30 | IT-008 | FR-011, RULE-002 | AC-011 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 31 | IT-014 | RULE-001 | AC-018 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 32 | AT-003 | BR-001, FR-002, FR-008 | AC-003 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 33 | AT-008 | FR-007, RULE-002, RULE-005 | AC-008 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 34 | AT-009 | FR-008, FR-009, RULE-002 | AC-009 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 35 | AT-011 | FR-011, RULE-002 | AC-011 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 36 | AT-018 | RULE-001 | AC-018 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 37 | IT-009 | FR-007, FR-009, FR-012, FR-015 | AC-012, AC-017 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 38 | IT-010 | FR-012, FR-013 | AC-013, AC-029 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 39 | IT-012 | FR-013 | AC-013, AC-029 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 40 | AT-012 | FR-007, FR-009, FR-012, FR-015 | AC-012 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 41 | AT-013 | FR-012, FR-013 | AC-013 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 42 | AT-029 | FR-013 | AC-029 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 43 | UT-007 | FR-013, RULE-003 | AC-015 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 44 | IT-011 | FR-013, FR-014, RULE-001, RULE-003, RULE-005 | AC-015, AC-018 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 45 | AT-015 | FR-013, FR-014, RULE-003, RULE-005 | AC-015 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 46 | REG-002 | FR-013, RULE-005 | AC-013, AC-015 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 47 | UT-006 | FR-001, FR-015, NFR-004, NFR-005 | AC-017, AC-022 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 48 | IT-020 | NFR-004, NFR-005 | AC-017, AC-022 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 49 | AT-017 | FR-001, FR-015, NFR-004 | AC-017 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 50 | AT-022 | NFR-004, NFR-005 | AC-022 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 51 | UT-010 | NFR-006, NFR-008 | AC-023, AC-026 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 52 | AT-023 | NFR-006 | AC-023 | Missing behavior; establish red in owning suite | Local widget coverage; manual/platform evidence separate |
| 53 | AT-026 | NFR-008 | AC-026 | Missing behavior; establish red in owning suite | Local widget coverage; manual/platform evidence separate |
| 54 | MAN-004 | NFR-008, NFR-006 | AC-026, AC-023 | Unverified; requires controlled live/platform evidence | Deferred to owner; report acceptance assumed/unverified |
| 55 | UT-009 | FR-005, FR-016, FR-019, NFR-002 | AC-028 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 56 | IT-017 | FR-005, FR-016, FR-019, NFR-002 | AC-028 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 57 | AT-028 | FR-005, FR-016, FR-019, NFR-002 | AC-028 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 58 | UT-008 | FR-017 | AC-025 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 59 | IT-016 | FR-017 | AC-025 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 60 | AT-025 | FR-017 | AC-025 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 61 | REG-004 | FR-017 | AC-025 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 62 | IT-018 | NFR-002 | AC-020, AC-028 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 63 | AT-020 | NFR-002 | AC-020 | Missing behavior; establish red in owning suite | Complete: owning local suites; see final evidence index |
| 64 | REG-005 | NFR-002 | AC-020 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 65 | REG-006 | FR-016, NFR-002 | AC-014, AC-020 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 66 | REG-003 | FR-014, NFR-001 | AC-016 | Missing behavior; establish red in owning suite | Complete: isolated backend gate and full Flutter regression |
| 67 | AT-024 | NFR-007 | AC-024 | Unverified; requires controlled live/platform evidence | Deferred to owner; report acceptance assumed/unverified |
| 68 | MAN-001 | NFR-007, RULE-004, FR-001, FR-002, FR-004, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013 | AC-024, AC-001, AC-002, AC-005, AC-008, AC-015, AC-019, AC-029 | Unverified; requires controlled live/platform evidence | Deferred to owner; report acceptance assumed/unverified |
| 69 | MAN-002 | NFR-006, FR-014, FR-016, NFR-001 | AC-023, AC-015, AC-016 | Unverified; requires controlled live/platform evidence | Deferred to owner; report acceptance assumed/unverified |
| 70 | MAN-003 | NFR-006, NFR-008, FR-017 | AC-023, AC-025, AC-026 | Unverified; requires controlled live/platform evidence | Deferred to owner; report acceptance assumed/unverified |

## Implementation Steps
### Step 1: UT-001
- Contract: Semantic capability and eligibility evaluation.
- Red: `go test ./internal/auth -run '^TestChatOptionalScopesPreserveBaseSession$' -count=1` failed because discovery metadata omitted optional authority.
- Green: added the explicit 15 chat methods, separately routed report permission and create/update-only declaration permission to consumer discovery. Ordinary config/PAR requirements remain base-only; localhost client ID encodes the advertised union. Operator discovery remains separate.
- Next red: `TestChatCapabilityNormalizesRPCScopes` failed for combined, reordered/encoded and encoded-positional method grants.
- Green: semantic explicit method/audience parser; wrong audiences/fragments, wildcard/unrelated methods, duplicate audience, unknown parameters and malformed scopes do not establish a narrow capability.
- Verification: focused scope/config/operator tests pass; `go test ./internal/auth -count=1` passes with DB tests skipped. Subsequent configured focused run also passes these unit cases.
- Remaining: service eligibility/status and full capability matrix will be extended at the owning chat service boundary. UT-001 is not claimed fully complete.
- Refactor: gofmt only while green.

### Step 2: IT-015
- Contract: Optional OAuth flow and routed authority.
- Red: `TestChatCapabilityDenialPreservesBaseSession` failed against configured PostgreSQL because the coordinator had no capability-aware boundary.
- Green: `WithCapability` checks effective optional grants inside the existing owner/session fence; missing chat grants deny only that operation.
- Red: `TestChatBackgroundSelectsCapableSameOwnerSession` reached the missing selector boundary after correcting test expiry fixtures to obey production constraints.
- Green: `SelectForCapability` skips base-only and expired candidates, remains same-owner, checks effective grants and does not touch child activity. Existing selector regressions pass.
- Red: `TestChatUpstreamPermissionLossPreservesBaseSession` showed upstream optional permission loss revoked the base parent.
- Green: optional operation failures preserve the original typed cause and yield consent-required without marking the base parent terminal; ordinary access remains valid. Static wrapper text does not expose provider prose.
- Verification: `scripts/appview-test focused ./internal/auth -run 'Test(Chat|BackgroundSessionSelector|TranslatePDS)'` passes against the worktree's actual PostgreSQL. Full auth race regression passed against local PostgreSQL (20.087s).
- Added `scripts/appview-test focused` to reuse existing dev-service discovery and fail-closed database configuration for repeated TDD runs; no hardcoded test port.
- Refresh red: `TestChatRefreshRechecksEffectiveOptionalGrant` observed one unauthorized chat dispatch after the SDK discarded refreshed effective scopes.
- Refresh green: one-operation transport tracks the response scope before rotated-credential persistence and gates resource dispatch/retries. No added retry. Focused chat auth tests pass with real PostgreSQL and race detection.
- Remaining: bound additional-consent/PAR/callback/disconnect races and fixed chat/report/declaration transport. These need the generation/persistence seams and remain pending.
- Order clarification: proceed to the planned slice-2 no-activity foundation, then return to consent cases once owner messaging generation exists; do not claim IT-015 complete.

### Step 3: AT-002
- Contract: Optional enablement and honest state.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 4: AT-019
- Contract: Semantic least-authority capabilities.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 5: REG-001
- Contract: Ordinary OAuth/session access, expiry and device-scoped logout.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 6: IT-005
- Contract: Multi-session enablement, expiry and lifecycle fences.
- Red: `TestChatAutomaticAuthenticationDoesNotExtendInactivity` failed because no no-activity store boundary existed.
- Green: `LookupWithoutActivity` shares ordinary lifecycle/epoch/expiry validation and revocation, skipping only the activity UPDATE. Explicit activity retains normal extension semantics.
- Red: `TestChatAutomaticAuthServicesUseNoActivityLookup` failed for both real and stacked development services.
- Green: a server-owned `NoActivityAuthService` contract and both real/development implementations use the no-touch lookup; invalid credentials remain invalid.
- Verification: focused real-PostgreSQL chat automatic-auth and existing CraftskySession/StackedAuth regressions pass. `AuthenticatedWithoutActivity` shares the existing HTTP parsing/error policy and has a passing focused middleware test; actual chat route policy wiring is still pending.
- Red: `TestChatAccountsMigrationEnforcesOwnerAndGeneration` applied all exact migrations and found no durable messaging account state.
- Green: next free migration 000102 adds owner-FK account state, positive generation and default-disabled/default-on-push values. Production-schema PostgreSQL 16 test passes.
- Red: `TestConnectionDisconnectFencesStaleConsent` could not build because the new Store API did not exist; this was the missing feature surface, not a database/setup failure.
- Green: generated sqlc/pgx queries and owner-fenced short transactions implement connection/generation transitions. Disconnect preserves preference, advances generation, and stale consent CAS cannot re-enable; another owner is unaffected. Configured production-schema test passes.
- Generator: pinned sqlc 1.31.1 uses the actual migration directory; `just chat-db-generate` runs the checked-in script. Shell syntax and recipe discovery checked.
- Remaining: actual consent callback/initiating child binding, disconnect delivery/activity cancellation, session/lifecycle races and route wiring; do not claim IT-005 fully complete.

### Step 7: IT-013
- Contract: Owner/membership enforcement and late response fencing.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 8: IT-019
- Contract: Registered-route contract and malicious inputs.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 9: AT-014
- Contract: Device and account revocation fencing.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 10: AT-016
- Contract: Account-safe notification opens and access.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 11: AT-021
- Contract: Bounded fixed API surface.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 12: UT-002
- Contract: Independent Dart/Go text validation (FR-005 / AC-006).
- Go red: `go test ./internal/chat -run '^TestMessageTextValidation$' -count=1` failed at blank, 1001-grapheme/emoji, over-byte and malformed UTF-8 cases with the minimal no-op validator.
- Go green: independent valid-Unicode, nonblank, 1000-grapheme and 10000-byte checks, using existing uniseg. Typed static errors contain no input. Valid input is never modified.
- Flutter setup: `just app-test test/chat/message_validation_test.dart` prepared checkout-local host Rive libraries and resolved existing dependencies. Corrected an initial test package-name typo (`craftsky_app`); that compile failure is not a behavior-red claim.
- Flutter red: after setup correction, `flutter test --no-pub test/chat/message_validation_test.dart` failed meaningfully on blank input with the no-op validator.
- Flutter green: independent characters/UTF-8 checks plus well-formed UTF-16 validation; no formatter/truncation or mutation. Same focused command passes; dart format reports no changes.
- Fixtures in both languages cover 999/1000/1001 graphemes, emoji/combining/ZWJ clusters, and 9999/10000/10001 bytes within the grapheme limit. Malformed Unicode is rejected.
- Follow-up parity case: BOM-only text was accepted by Go but rejected by Dart trim; observed a Go red and aligned blank checking without altering valid input.
- Integration no-send and retained-draft assertions remain IT-006/AT-006/AT-028 responsibilities; passing validators alone do not prove those flows.

### Step 13: UT-005
- Contract: Report validation and evidence construction (FR-010 / AC-010).
- Go red: `TestReportInputValidation` failed at missing/unsupported reasons, 2001 graphemes, 20001 bytes and malformed Unicode with a no-op validator.
- Go green: exact supported consumer report reasons and independent optional explanation limits, with static typed errors and no truncation. Focused message/report validator suite passes.
- Flutter red: `flutter test --no-pub test/chat/report_controller_test.dart` failed at missing reason type with a no-op validator.
- Flutter green: typed report input errors, the same supported reason list and independent Unicode/grapheme/byte checks. `flutter test --no-pub test/chat` passes both suites and targeted Dart analysis passes.
- Fixtures cover nil/empty explanations, 1999/2000/2001 graphemes, 19999/20000/20001 bytes within grapheme limits, and malformed Unicode.
- Green refactor: moved the Flutter text validator to `models/text_validation.dart` before adding report text validation; message tests remained green. Both validators share the well-formed-Unicode helper.
- Remaining: authorized message-view evidence, forged reference denial and fixed moderation routing. UT-005 is partial, not complete; IT-004/durable report handling and UI remain pending.

### Step 14: UT-011
- Contract: Consent and reporting copy semantics.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 15: IT-004
- Contract: Real report adapter boundary under faults.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 16: AT-006
- Contract: Valid text and confirmed send states.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 17: AT-010
- Contract: Bluesky-only message reports.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 18: IT-006
- Contract: Durable send operation and all crash windows.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 19: AT-007
- Contract: Uncertain sends and deliberate recovery.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 20: IT-001
- Contract: Direct inbox/request adapter.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 21: IT-002
- Contract: History pagination and deletion.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 22: UT-003
- Contract: History mapping/merge and view anchor.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 23: IT-003
- Contract: Entry points, availability and canonical start.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 24: AT-001
- Contract: Interoperable paginated inbox.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 25: AT-004
- Contract: Stable safe history.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 26: AT-005
- Contract: Availability before canonical creation.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 27: AT-027
- Contract: New message and profile entry.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 28: UT-004
- Contract: Read watermark and native badge mapping.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 29: IT-007
- Contract: Read/accept/mute/leave authoritative round trips.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 30: IT-008
- Contract: Conditional declaration merge.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 31: IT-014
- Contract: Indexed/newly confirmed block races.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 32: AT-003
- Contract: Explicit request acceptance.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 33: AT-008
- Contract: Native unread counts and presented-message reads.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 34: AT-009
- Contract: Shared actions and truthful leave.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 35: AT-011
- Contract: Conditional shared incoming policy.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 36: AT-018
- Contract: Block enforcement without history deletion.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 37: IT-009
- Contract: Cursor and forward-compatible event decoding.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 38: IT-010
- Contract: Atomic progress, dedupe and outbox.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 39: IT-012
- Contract: Lifetime request and recovery state machine.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 40: AT-012
- Contract: Authoritative event reconciliation.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 41: AT-013
- Contract: Crash-consistent logical notifications.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 42: AT-029
- Contract: Request lifetime and coalesced catch-up.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 43: UT-007
- Contract: Push identity and payload construction.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 44: IT-011
- Contract: Dispatch-time eligibility and payload privacy.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 45: AT-015
- Contract: Private account-wide push.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 46: REG-002
- Contract: Social notification categories, audience/profile-mute filters and durable leases.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 47: UT-006
- Contract: Recovery classification and deterministic scheduling.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 48: IT-020
- Contract: Fair bounded shared scheduler under faults.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 49: AT-017
- Contract: Classified failure and retained owner state.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 50: AT-022
- Contract: Shared bounded polling.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 51: UT-010
- Contract: Conversation adapter and messaging-flow rendering, semantics and package feature controls.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 52: AT-023
- Contract: Supported platform usability.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 53: AT-026
- Contract: Selected chat UI package fit.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 54: MAN-004
- Contract: Planning record and real package/platform fit.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 55: UT-009
- Contract: Draft reducer and send transitions.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 56: IT-017
- Contract: Real local draft storage and process lifecycle.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 57: AT-028
- Contract: Private local drafts without offline replay.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 58: UT-008
- Contract: Preview candidate and presentation policy.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 59: IT-016
- Contract: Preview access, fetch counts and navigation races.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 60: AT-025
- Contract: First supported CraftSky preview only.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 61: REG-004
- Contract: Existing content link navigation and access controls.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 62: IT-018
- Contract: Serialized sink/storage and cleanup inspection.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 63: AT-020
- Contract: Serialized privacy and minimal retention.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 64: REG-005
- Contract: Existing diagnostic sanitization and issue ownership.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 65: REG-006
- Contract: Owner deletion namespace boundaries.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 66: REG-003
- Contract: Account-safe existing notification destinations.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 67: AT-024
- Contract: Actual integrated provider behavior.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 68: MAN-001
- Contract: Actual CraftSky/AppView/PDS/chat/moderation integration during full implementation.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 69: MAN-002
- Contract: APNs/FCM and real OS taps on iOS and Android.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

### Step 70: MAN-003
- Contract: Assistive technology and actual platform interaction.
- Executed through the owning local suites in the final requirement-to-evidence index and chronological TDD log below. Acceptance umbrellas do not claim independent live-provider execution; owner manual gaps remain explicit.

## Manual Validation Ownership
- Owner will perform live/device/manual checks. There is no approved report-testing procedure; owner accepts live report behavior as an assumption, not verified evidence. Do not submit live reports. See `06-validation-evidence.md`.

## Regression Findings During Foundations
- Affected auth/chat/db/routes/app packages passed with real local PostgreSQL/MinIO and race detection. Owner-lifecycle inventory correctly failed when chat_accounts.owner_did was undeclared. Added its delete-row inventory role; focused inventory and full owner-lifecycle race suite then passed.
- REG-006 private cleanup test was extended with two owners messaging state and failed because deletion retained the deleting owners chat row. Added owner-scoped deletion; repeatable cleanup/other-owner preservation test passes with race detection. This coverage was pulled forward with the new table rather than postponing privacy cleanup to slice 13.

## Final Verification
- Focused auth: from `appview/`, `go test ./internal/auth -run 'TestChat(OptionalScopesPreserveBaseSession|CapabilityNormalizesRPCScopes)' -count=1`.
- Backend: `just appview-test-unit`, configured `just test`, and `just appview-check`. Unit skips are not persistence evidence.
- Flutter: `just app-test test/chat`, related notification/router/observability suites, `just app-test`, `just app-analyze`.
- Real service and platform prerequisites/evidence: record in `06-validation-evidence.md`; keep unavailable checks explicitly unverified.

## Remaining validation
- All selected local feature loops are implemented; the chronological entries below preserve the actual red/green history.
- Final isolated backend gate, complete Flutter regression repeat and all three supported-platform rebuilds passed after the final authority-race extension. Results are recorded below and in `06-validation-evidence.md`.
- Owner-run MAN-001–MAN-004 and AT-024 remain deferred. Live report acceptance is assumed/unverified. Local key configuration is documented in `06-validation-evidence.md`; production setup/release is separate.

## Completion Checklist
- [x] All Must requirements covered by local tests or documented owner-run gaps
- [x] All planned local Must tests passing; MAN-001–MAN-004 and AT-024 explicitly deferred to owner
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Implementation review explicitly deferred to the separate next-stage owner choice; no review completion is claimed
- [x] Plan read back before stage handoff

## Resumed execution: optional consent
- IT-015 / RULE-004, NFR-001: `TestChatConsentPurposeIsSeparateFromOnboarding` first failed because chat callback attempts were rejected; passed after a separate chat purpose with consumer handoff permission was added. Ordinary onboarding permission remains unchanged.
- IT-015 / FR-001, RULE-004, NFR-001: `scripts/appview-test focused ./internal/auth -run '^TestChatConsentHandoffEnablesOnlyAfterConfirmation$' -v` failed on missing durable consent-binding columns, then passed after migration/metadata/callback/handoff changes. The test uses the real migrated schema and a single database connection; it proves callback does not run profile reconciliation and connection stays disabled until child confirmation. Confirmation is idempotent and preserves the original child.
- Chat consent captures exact initiating child hash, parent, device, messaging generation, owner generation/auth epoch and optional push choice. Callback and confirmation revalidate this authority under the existing owner fence. Start admission, negative race cases and purpose-client routing remain the active loop.

- IT-015 consent admission: `TestChatConsentStartBindsAuthenticatedChild` failed because the authenticated admission interface was absent. After adding same-owner admission it passed; the synthetic OAuth server proves fixed scopes/login hint and persisted exact-child/generation/push binding, plus rejection of another child. Two fixture-only metadata/import errors were corrected and do not count as behavioral red evidence.
- IT-015 stale callbacks: `TestChatConsentRejectsStaleInitiatingAuthority` exposed missing consent expiry validation (callback finalized); after expiry validation all disconnect, child logout, device change, idle expiry and consent expiry cases pass before token exchange. An idle-expiry fixture check violation was corrected separately.
- IT-015 purpose transport: `TestChatPurposeClientRechecksRefreshedGrant` failed because the purpose client could not share refreshed authority; it now passes using a grant-bound clone retaining the resource client's policy. No resource request occurs after refresh loses optional scopes; base access remains valid.
- Regression: `scripts/appview-test focused ./internal/auth ./internal/chat ./internal/db -race` passed (auth 27.865s, chat 2.042s, db 21.770s), before the last purpose-client helper addition. The focused refresh/purpose tests passed afterward.

- IT-005 local rollback: `TestChatMigrationRollbackRestoresAuthPurposeContract` failed because rollback retained chat consent admission; after restoring the exact preceding constraints/removing binding columns, rollback and reapply pass in an isolated migrated schema.
- IT-015 fixed transport: `TestTransportPinsChatAndReportProxy` first failed on the missing adapter, then exposed an SDK nil-header panic. After cloning/initializing headers it passes: chat/report use their separate fixed PDS proxy audiences and unapproved deletion never dispatches. The adapter enforces a ten-second call timeout/two-MiB response bound and adds no automatic write retries.
- UT-001 capability matrix: `TestCapabilitiesKeepReportAndDeclarationIndependent` passes for base/full and independently omitted report/declaration grants. A fixture initially omitted mandatory atproto; that fixture error was corrected without changing the existing semantic grant validator.
- UT-001 / AT-002 backend status: `TestStatusDistinguishesConsentRestrictionAndFailure` initially failed because the service was absent; migrated-store/synthetic-provider tests now pass for disabled, authorizationNeeded, restricted, unavailable and ready without serializing provider prose.
- IT-019 status route: `TestChatStatusRouteRequiresAuthenticationAndDevice` failed with 404, then passed with normal envelopes, owner/device auth and explicit private diagnostic metadata. Initial AddRoutes argument setup was corrected separately.
- IT-005 status routing: `TestChatStatusUsesAutomaticAuthentication` failed (ordinary lookup recorded activity); now passes with a server-owned NoActivity policy and identical auth parsing, plus fail-closed behavior for auth implementations without the seam.
- Composition now wires chat status through the real OAuth coordinator and hardened no-redirect PDS client. The rest of the consumer surface remains pending; this is not a completed feature.

- IT-006 configuration prerequisite: malformed `CHAT_OPERATION_HMAC_KEY` was silently ignored (red); dedicated redacted 256-bit base64url key and bounded version configuration now pass malformed/positive/redaction tests. No production key was provisioned. This prerequisite was executed before the receipt loop, while the remaining route matrix will grow alongside its owning services.
- IT-006 minimal durable ledger: `TestOperationReceiptDoesNotRepeatAmbiguousSend` first failed on the missing ledger; it now passes against actual migrated PostgreSQL. Same UUID/payload returns a protected receipt without a second call, changed payload conflicts, and SQL contains neither private message text nor provider prose. Only keyed request digests and bounded outcomes are retained.
- IT-006 prepared-crash recovery: `TestExpiredPreparedOperationIsNotSubmittedAndNeverReplayed` failed because the orphan remained uncertain. A 30-second prepared claim now expires as definitely not submitted, and the original attempt CAS also checks that deadline. Another request never takes over the write; focused prepared/ambiguous tests both pass.
- Later regression verification: route suite with race detection passed (24.852s); application suite with race detection passed (43.879s) after its shared synthetic auth schema was brought up to migration 102. The route inventory assertion was extended by exactly the new approved status route. Auth/chat race suites also passed before receipt changes (27.637s/6.141s).

- UT-011 / AT-002 enablement copy: the first Flutter test failed on the absent page; `flutter test --no-pub test/chat/messaging_consent_test.dart` now passes. Shared inbox/actions across apps, Bluesky hosting outside public PDS records, no end-to-end encryption, Bluesky-only reports and unchanged incoming policy appear before an explicit authorization trigger. Push starts on and the trigger receives that choice. ARB localization generation completed; targeted Dart analysis is clean.
- Receipt cleanup regression found an unclassified cascading chat_accounts parent. The parent now waits for its separately bounded chat_operations purge component. Full ownerlifecycle race suite passed (148.430s), and `TestTerminalChatAccountWaitsForBoundedReceiptPurge` proves a three-receipt account is drained one row per batch before parent removal. Chat/db/accountdeletion race suites passed (6.834s/24.030s/9.066s) after the receipt migration, before report-service changes.
- UT-005 evidence: `TestReportEvidenceUsesAccessibleMessageSender` failed on the missing evidence builder and now passes. It derives the messageRef sender from an accessible direct view, rejects unrelated senders/nonmembers/groups and serializes no transcript.
- IT-004 initial report path: `TestReportUsesCurrentMessageReferenceAndSingleAttempt` failed on the missing service and now passes using migrated PostgreSQL and synthetic upstream responses. Current conversation/message reads determine the reference, fixed moderation routing is asserted, duplicate operations call createReport once, forged message IDs do not dispatch, and moderation_reports remains empty. Full retry faults and reports on deeply paginated history remain pending.

- UT-011 / AT-010 report confirmation: `flutter test --no-pub test/chat/report_page_test.dart` failed on the missing page, then passed after the localized report flow. It names Bluesky and message-reference/reason/explanation scope before the explicit trigger; confirmed acceptance alone displays success. A test finder initially matched the select input's search field as well as explanation; the explanation now has its own key and that fixture error is not behavioral red evidence. Failed/uncertain recovery rendering needs its remaining assertions.
- IT-006 send path: `TestSendCommitsAttemptBeforeNetworkAndDoesNotReplay` failed on the missing Send service; it now passes with actual PostgreSQL. The remote recorder observes committed attempting before a simulated 500, duplicate UUID sends once, and invalid blank text dispatches nothing. Participant lifecycle fences, messaging-generation checks and current accepted-state/either-direction block guards surround the effect. Full real SDK challenge/retry faults and confirmed-send UI remain pending.
- IT-005 / IT-013 exact-child authority: `TestStatusRejectsRevokedExactChildDespiteLiveSibling` first exposed a revoked child borrowing a live sibling parent. A server-derived bearer hash now reaches chat consumers and the exact child is checked under the session fence. Extending the test to disconnected messaging exposed a private disabled-status read gap; ConnectionForChild now validates there too. Focused status/send/report cases pass; automatic work must use a separate server-selected active background path when implemented.
- sqlc bindings were regenerated after receipt-generation, current-block and exact-child queries. The receipt stores its messaging generation and preserves old tombstones across reconnect; no message/report payload is persisted.

- IT-019 authenticated authorization: `TestChatAuthorizationCannotSelectOwnerOrArbitraryTarget` passed after adding the bounded strict handler. It rejects owner/audience/return-target controls and unsupported handoffs, uses authenticated owner/parent/child/device, and forwards the explicit push choice. The first handler version exposed an auth/observability import cycle; it was corrected to the existing auth diagnostic bridge before passing. The authorization route first returned 404 in its credential test, then passed after private catalogue/diagnostic/rate/suspension registration. Authorization is unavailable without the dedicated operation key; no production configuration changed.
- IT-015 handoff regression: the expanded consent-confirmation test caught repository onboarding jobs being enqueued. Chat confirmation now skips that unrelated work after atomically confirming credentials/enablement; consent plus ordinary handoff/profile-recovery regressions pass.
- IT-005 connection toggle: `TestConnectionToggleRequiresCurrentChildAndOptionalAuthority` first failed on missing service behavior; all configured variants now pass. Disconnect is local after optional grant loss; re-enable requires full current grant and upstream eligibility, preserving shared policy and push preference.
- IT-019 opaque identifiers: bounded ConversationID/MessageID parsers and redacted formatting pass validation/wire tests. A green refactor carries those types through send/report boundaries with explicit raw conversion only for upstream wire fields. The refactor's one internal string-conversion compile error was corrected before continuing.
- UT-001 malformed eligibility: an empty upstream status object incorrectly became ready (red). Required pointer decoding now treats missing chatDisabled as unavailable throughout status/send/report/connection entry points; focused service regressions pass.
- IT-019 send contract: `TestChatSendContractRejectsArbitraryControls` failed on the missing handler, then passed with bounded UTF-8/strict JSON, exact UUID/text/owner inputs, no-store responses and 202 uncertain receipts without provider prose/body. The route's initial 404 was fixed with normal auth/device/current-member guards and private diagnostic classification. Shared error helpers preserve typed causes privately and emit fixed consumer envelopes.
- Latest broad checks before the authorization handler addition: chat/auth/middleware/routes/app race suites passed (12.077s/33.254s/2.240s/31.574s/51.567s). Four current Flutter chat tests passed. Subsequent targeted Dart analysis found one brace-style lint in report-page copy controls; it was fixed mechanically and awaits the next analysis pass.

- IT-001 / FR-002, FR-008 inbox: `TestInboxContinuesThroughFilteredGroupsAndKeepsCursorPrivate` passed for accepted and request buckets after missing listing behavior was implemented. Five filtered group-only pages preserve a random owner/generation/scope-bound continuation; external direct peers and latest text retain their upstream state.
- IT-002 / FR-003 history: `TestHistoryKeepsSafePlaceholdersAndPrivatePagination` passed after the absent History service was added. It preserves chronological stable IDs, deleted and unknown placeholders, valid cursors on empty pages, and never accepts or marks read. UT-003 still needs Flutter merge/anchor assertions.
- IT-003 / FR-004 start: `TestAvailabilityDoesNotCreateAndExplicitStartIsCanonical` failed on missing service methods, then passed. Availability cannot create; explicit starts use fresh availability and either-direction block checks under participant/session fences, reject self and validate canonical direct membership. This belongs to IT-003; the progress commentary also mentioned UT-003, whose contract is history rather than starting.
- UT-004 / IT-007: `TestReadNamesPresentedMessageAndCountsUseNativeBuckets` failed on missing UpdateRead/UnreadCounts, then passed. The exact required presented ID is forwarded; request state remains request, counts use includeGroupChats=false and retain cap sentinel 100.
- IT-007 / FR-008, FR-009: `TestConversationActionsConfirmAuthoritativeStateWithoutReading` failed on absent actions and now passes: acceptance without rev is confirmed by current accepted state, shared mute/unmute preserve unread state, leave validates the upstream result. Pending delivery cancellation will be extended with the outbox.
- IT-004 / FR-010 deep-history report: `TestReportRevalidatesMessageOnPreviouslyLoadedOlderPage` failed with inaccessible message after page six was loaded. Continuations now attach only up to 100 message IDs for the already bounded 30-minute page handle; reporting refetches that current upstream page. Both report integration tests pass; no text or sender reference is persisted.
- IT-006 protected receipts: expanded `TestSendCommitsAttemptBeforeNetworkAndDoesNotReplay` first failed on absent Receipt, then passed. Owned receipt reads project unresolved attempts as uncertain, reject missing UUIDs, and never repeat the remote effect.

- IT-008 / FR-011: `TestIncomingPolicyMergesLatestDeclarationAfterConflict` failed on missing policy support, then passed with authoritative self-record reads and bounded InvalidSwap refetch/merge. A separate missing IncomingPolicy read failure was observed and fixed. Unrelated group/future fields survive, only own declaration/self is written, no proxy is used for PDS record operations, and readback confirms persistence without promising immediate chat propagation. Official source `bluesky-social/social-app/src/state/queries/messages/actor-declaration.ts` confirms the following fallback when no declaration exists.
- IT-014 / RULE-001: `TestEitherDirectionBlockPreservesReadAndPreventsEffects` failed on absent conversation summary, then passed after adding current summary/canSend. The synthetic aggregate fixture initially lacked required Tap/source FK rows; these were added without changing business behavior. Both directions reject start/send and return readable accessible history/availability false.
- IT-019 HTTP consumer: `TestChatConsumerStrictPaginationContract` first failed on missing handler, then caught invalid cursor returning 503. Fixed consumer mapping now returns 400, rejects unknown/duplicate query keys and out-of-range limits, and keeps responses no-store. Route credential matrix first returned 404; registered the implemented fixed routes with normal owner/device/member guards, automatic no-activity reads, explicit denied suspended mutations and private diagnostic catalogue entries. Targeted API/routes/inventory/suspension tests pass.
- IT-011 preference prerequisite: `TestPushPreferenceIsLocalAccountWideAndPreservedOnDisconnect` failed on missing service, then passed. Preference changes are exact-child validated local SQL, persist through disconnect and dispatch no upstream write. Preference routes first returned 404 and now pass credential checks. Full dispatch policy remains pending.
- Affected package regression: `scripts/appview-test focused ./internal/chat ./internal/auth ./internal/db ./internal/api ./internal/routes ./internal/app ./internal/accountdeletion -race` passed (chat 38.032s, auth 47.010s, db 37.975s, api 141.983s, routes 45.700s, app 56.558s, accountdeletion 14.685s). This run preceded the new synchronization tables.
- IT-009 decoder: `TestLogDecoderPreservesUnknownProgressWithoutPrivatePayload` failed on missing decoder and now passes. Reviewed type/revision/identifier fields survive; unknown arbitrary payload and text do not. Diagnostic formatting is redacted. Official event-agent source confirms resuming from event revisions, preserving cursor after transient failure and cursorless head only for initialization.
- IT-009 / IT-010 initial durable sync: `TestSyncBaselineIsSilentAndProgressCommitsWithNotification` failed on missing synchronizer, then passed using real migrated PostgreSQL. Three bounded baseline phases seed head and enumerate accepted/requests without alerts; normal log progress uses a durable expiring CAS claim. Injected logical-notification insert failure rolls back cursor and dedupe together; retry emits one logical row. No private text persists. This is initial core evidence, not completed request/recovery/scheduler/delivery coverage.

- Sync cleanup regression: migrated inventory first reported the four new sync ownership roles missing. Added indexed bounded components and made chat_accounts wait for them before deletion; inventory/keyset/unique-key and bounded chat-receipt parent tests pass. The later heartbeat role likewise failed inventory before being added.
- IT-012 / FR-013: `TestRequestNotificationLifetimeSurvivesLeaveAndReconnect` caught a forgotten leave state. Durable left state and cancellation now preserve once-per-request eligibility across reconnect/reappearance; positive accepted state then new request increments the request generation. The test passes.
- IT-007 leave extension: `TestConversationActionsConfirmAuthoritativeStateWithoutReading` failed because a pending logical notification survived confirmed leave; it now passes after owner-scoped cancellation/state recording in the existing held session transaction. Subscription-delivery cancellation will extend the same boundary when its table lands.
- IT-009 / IT-012 recovery: `TestRecoveryResumesSavedCursorAndCoalescesAcceptedConversation` failed with two normal alerts after interruption, then passed with a durable recovery ID and one accepted-conversation catch-up key. Valid saved cursor is resumed, not replaced by head.
- IT-009 / IT-012 expired cursor: `TestExpiredCursorReconcilesPagesBeforeInstallingNewHead` initially returned InvalidCursor; it now passes with checkpointed recovery-head, accepted and request phases. All authoritative pages reconcile before installing the replacement cursor; prior baseline message identity remains suppressed. Actual provider cursor error names/expiry behavior remain MAN-001 evidence.
- IT-020 heartbeat: `TestViewHeartbeatExpiresAndDoesNotTouchSessionActivity` failed on absent view activity, then passed. Current conversation view activity lasts twelve seconds, healthy account polling becomes four seconds, expired/inbox activity uses thirty seconds, heartbeat performs no inline getLog, and child lastSeen/idle expiry stay unchanged. Fake times use PostgreSQL microsecond precision.
- IT-019 / IT-005 heartbeat route: new sync route returned 404/missing policy, then passed after private registration with NoActivity=true. Targeted API/routes/inventory/suspension checks pass.

- IT-020 capacity: `TestPollClaimsBoundConcurrentAccountsAcrossWorkerInstances` failed because three accounts polled concurrently. A short transaction-scoped capacity lock plus shared claim-count guard now caps active account claims at two across worker instances; the test and heartbeat cadence pass with race detection. sqlc initially required qualification of the outer owner column after the capacity subquery was added; generation then passed.
- IT-020 fair scheduler: `TestSchedulerServesHealthyOwnerWhileAnotherPollIsHeld` failed on missing Run/pulse seam, then passed with race detection. Two local task slots share SQL claims, completion yields capacity to the next due owner, repeated wakeups do not overlap the held owner, and shutdown joins canceled work. Expected authorization loss becomes a paused account state; private unexpected poll failures use one source-owned observer. Worker startup/shutdown is wired to AppView only when the dedicated operation key is configured.
- UT-006 / IT-020 rate limiting: `TestPollRateLimitKeepsCursorAndHonorsLongRetryAfter` failed because the response lost Retry-After. Transport now preserves a typed RateLimitError with original cause; worker keeps progress and schedules the owner at the longer provider delay (tested 120 seconds). Ordinary failed reads are scheduled with exponential jitter from one second capped at sixty; each tick makes one attempt, with no send/report retry wrapper. Remaining full resource/fault assertions are still open.
- IT-005 disconnect: `TestDisconnectCancelsWorkAndRetainsResumeAndOperationTombstones` failed with live activity/claim/pending work after disconnect. Disconnect now atomically clears continuation/activity, cancels logical notifications, releases the claim and marks recovery while retaining cursor and operation receipts; request lifetime regression also passes.
- IT-005 / IT-020 device logout: heartbeat test extension caught retained activity after child revocation. A schema-bounded one-row-per-child cleanup trigger removes activity on every existing revocation path; heartbeat and migration rollback/reapply tests pass. It does not change session expiry or owner messaging preferences.

- IT-005 / IT-015 reconnect: `TestReconnectResumesPausedCursorWithoutRebaselining` failed with a five-minute authorization pause surviving reconnect, then passed after resumable-state scheduling. The consent handoff test extension likewise caught confirmation failing to resume prior progress; its transaction now resumes the cursor/recovery identity with credential activation, and the focused test passes.
- Backend race regression after sync/heartbeat changes passed: chat 33.601s, db 26.699s, auth 34.287s. This run preceded the reconnect additions and private delivery work. App/main composition compiled and focused chat config tests passed after scheduler wiring; no live process/provider was started.
- UT-007 / RULE-003 provider contract: `TestFirebaseChatPayloadHasOnlyOpaqueRoutingAndSenderCopy` failed on absent private routing support, then passed through the real Firebase serializer for iOS/Android. Data is exactly version/type/opaque subscription/random open reference; no social targets, message identifiers or text. Existing Firebase/payload unit regressions passed. DM is a separate internal request variant; social notification categories were not widened.
- IT-010 / IT-011 fanout: the atomic sync test extension first failed on absent chat_push_deliveries, then passed. One row per active subscription is committed with logical notification/dedupe/cursor; an inactive installation receives none and social notification_events stays empty. Terminal inventory first rejected the new delivery owner role; bounded delivery/notification/subscription purge components were added, and focused inventory/keyset/cascade tests passed.
- IT-011 mute separation: `TestDeliveryIgnoresProfileMuteAndChecksCurrentConversationMute` failed on absent dispatcher, then passed. Server-selected live capability, current direct membership, unread/mute/left/preference/block/generation and subscription state are checked before provider send under participant/session fences. Profile mute is ignored; completed delivery cannot claim again. Further race/fault/retention assertions remain open.

- IT-011 invalid token: `TestDeliveryInvalidTokenCannotDeactivateRotatedToken` failed because a current rejected installation remained active. The final transaction now invalidates only the exact attempted token after token/lease CAS, disables its subscriptions and cancels associated work. A token rotated during the provider call stays active. Both variants and mute separation pass with race detection. A generated timestamp field-name compile mismatch was corrected before the green run.
- IT-005/IT-007/IT-011 cancellation extensions: pending device rows survived disconnect, confirmed leave and push-disable in their first expanded runs. The respective owner/conversation SQL transactions now cancel logical and per-subscription work together. Focused lifetime, actions, disconnect and preference tests pass; no upstream calls occur for the local preference/disconnect paths.
- IT-013 private opens: `TestNotificationOpenResolvesOnlyCurrentAccessibleOwnerConversation` failed on missing service, then passed with owner/reference lookup plus exact child, enabled/generation and fresh membership checks. Unknown references never reach conversation lookup; requests are neither accepted nor marked read. Its route credential test first returned 404 and now passes with private/no-activity registration.
- IT-018 retention: `TestMaintenancePurgesRoutingDetailWithoutRearmingLifetimeState` failed on missing Maintain, then passed. Fixed 100-row statements expire alerts at 24h, clear terminal routing and device-delivery detail at 30d, and clean expired views/continuations. Request generations, message dispositions and operation receipts remain lifetime metadata. Notification uniqueness uses a digest of reviewed operational IDs (never message/report prose), permitting full routing detail removal while retaining the tombstone.

- Worker lifecycle: `TestDeliveryWorkerSendsQueuedJobAndStops` failed on missing Run, then passed with race detection. Private per-attempt diagnostics retain typed causes/stacks at the owning dispatch boundary; expected authorization recovery is not repeatedly reported. AppView shares its existing provider sender across social and private dispatch, and starts credential-independent bounded maintenance with graceful shutdown joins.
- Broad backend regression after delivery/retention: chat, push, routes, ownerlifecycle, accountdeletion and db passed with race detection (51.239s/6.460s/40.947s/36.770s/12.775s/34.391s). App integration found direct constructors violating the repository's composition test and auth fixtures missing migration 21 push tables; moved construction into chat dependency helpers and added that prerequisite to the synthetic schema. Focused regressions pass. Subsequent full app race suite passes (50.580s); auth found one older test manually recreating those now-shared push tables and is being repaired.
- UT-003 / FR-003 Flutter history: missing model/history imports established red. Immutable, redacted identifiers and message/page/history state now preserve chronological IDs, keep equal text under distinct IDs, strip deleted/unsupported bodies and prevent stale-page resurrection. Empty supported pages retain a valid continuation. The focused Flutter test passes and targeted analysis is clean. Incorrect initial spelling of the repository's new named-constructor syntax was corrected before green; it is not behavioral red evidence.
- Auth fixture correction is verified: focused shared-installation logout passed, then the complete auth race suite passed (30.756s).
- IT-019 / FR-002 Flutter inbox contract: absent repository/models established red; the fixed AppView adapter now decodes external peers and preserves empty-page continuations with redacted DTOs. Cross-checking the actual server exposed the initial test's incorrect `bucket` query name; correcting the fixture to the authoritative `status` produced a meaningful request failure, then the adapter was corrected. No user-selected owner/host/RPC is accepted.
- IT-019 Flutter fixed-surface expansion: missing conversation/history/action/receipt methods established red. Contract tests now pass for explicit read watermarks, accept/mute/leave, availability versus explicit start, uncertain sends and metadata-only receipt lookup. Missing status/preferences/sync/report/open/consent methods separately established red and now pass. Native count 100 displays `99+`; optional report capability remains separate. No retry wrapper or upstream SDK was added to Flutter. Controller ownership, transport fault/privacy and routed UI coverage remain open.
- UT-009 draft reducer: absent reducer established red; focused test now passes for pending/uncertain/failed preservation, explicit recovery, exact submitted-revision confirmation, later edits, discard and unchanged over-limit local input. The reducer performs no network operation. Durable restoration/logout races and routed composer integration remain pending.
- IT-017 secure local storage: absent adapter established red. Production flutter_secure_storage adapter (platform channel mocked) passes isolated A/B restoration, pending-to-uncertain restart, opaque hashed storage keys and owner-only purge. Expanded race test exposed initialization failing to admit newly retained accounts; serialized reconciliation now preserves monotonic removal generations. Delayed/queued saves cannot restore a removed owner. Cold logout test then exposed uninitialized stores skipping persisted deletion; removeAccount now loads the manifest before purge and passes. Storage failures retain original causes/stacks in a redacted exception. iOS uses non-synchronizing device-only accessibility; Android uses a dedicated namespace without reset-on-error; browser storage remains origin-local, with real platform/process validation delegated to MAN tests. Android backup exclusion and actual registry/controller integration remain open.
- IT-017 platform storage configuration: native-contract test failed because Android had no draft backup exclusions. Dedicated namespace data, wrapped keys and configuration are now excluded from both cloud backup and device transfer (including pre-Android-12 full-backup rules); the focused test passes. Exact names were checked in the installed flutter_secure_storage 11.2.0 native implementation. This verifies checked-in configuration, not an actual device transfer.
- IT-017 actual account cleanup: missing draft store provider established red; the existing account-session private-state cleaner now awaits owner-only draft removal. Focused cleanup test passes with B preserved.
- UT-009 conversation orchestration: absent controller established red; durable attempt-before-send and confirmed-message/later-edit assertions pass. IT-017 stale-load extension then caught disposal writing an empty, not-yet-loaded draft over retained text. The controller now flushes only after draft restoration; stale A responses publish nothing and preserve retained A text. No conversation load marks read or accepts a request.
- UT-004 / FR-007 controller read boundary: missing presented-message API established red. The controller now updates only a loaded, actually presented watermark while foreground, coalesces an in-flight read and never regresses to an older visible page. Focused tests prove initial load/background/unknown IDs do not updateRead and fetching a newer message does not mark it read. Real package viewport callbacks remain to be connected and tested.
- UT-009 explicit recovery: missing receipt-check/discard controller methods established red. Focused tests now pass for a failed transport becoming uncertain, zero implicit resend, metadata receipt check without replay, confirmed clearing and explicit local discard. History refresh uses authoritative IDs, never text matching.
- IT-015 / IT-019 account provider acquisition: missing providers established red after correcting a fixture import ambiguity. Account-specific Dio acquisition and status completion now use captured activation guards and redacted AccountKey families. Focused switch-during-acquisition test passes with zero HTTP requests. The isolated test initially inherited Riverpod's retry policy and timed out; it now uses the application's existing no-retry policy, and that timeout is fixture evidence rather than a behavioral failure. Code generation completed; unrelated generated hash/format-only drift was removed.
- IT-019 / UT-002 Unicode boundary extension: the HTTP send contract first dispatched an unpaired surrogate after Go silently replaced it. The shared JSON Unicode guard now rejects malformed UTF-8 and unpaired escaped surrogates before decoding all chat consumer bodies. A separate real transport test then failed on upstream silent repair; the adapter now applies the same guard. Focused API/transport tests pass, preserving valid surrogate pairs, literal backslash-u text and intentional replacement characters unchanged.
- UT-001 readiness extension: status was ready after loss of getMessages/acceptConvo or removal of the operation HMAC key (red). It now requires the full core chat RPC set and an available operation ledger, while independently omitted report/declaration grants stay ready with their capabilities false. Focused status/capability regressions pass.
- IT-015/IT-017 same-owner handoff: server exchange lacked source metadata (red); it now returns server-owned chat purpose and the initiating child-token SHA-256 hash, with ordinary exchange bodies unchanged. Real migrated consent confirmation passes. Flutter registry test then caught chat handoff activating another account; pending chat receipts now bind the retained source generation through restart, replace only that owner and preserve another active lease/cached identity. Removed/replaced sources are rejected. Auth-controller test caught unrelated active-account invalidation and now passes without disturbing B. Expanded before/during-confirmation replacement tests initially reported storage failures and dispatched stale confirmation; source-specific cancellation now prevents stale dispatch/publication, discards the stale receipt and preserves newer credentials. Focused model/controller tests pass. Same-owner routing and serialized private diagnostic checks remain pending.
- IT-018 chat handoff diagnostics: source-selected private context now replaces public-record context for chat-only handoff storage failures. The actual console serializer test passes with positive operation account/failure-stage fields and negative bearer/code/receipt/hash canaries. The first assertion looked for the account field at the outer level rather than the serializer's diagnostic map; that fixture error was corrected and is not behavioral red evidence. Backend auth race regression passes after exchange metadata changes (31.068s). A broader Flutter run exposed an older sign-out test lacking the now-required draft storage platform mock; the fixture is updated and broader verification is being repeated.
- UT-010 package adapter: absent selected-package dependency/adapter established red. Exact flutter_chat_ui 2.12.0/core 2.9.1 resolved with nine additions and no existing downgrade/override. The widget test now passes for actual Chat rendering, localized deletion, custom composer and no writable package composer. Its first composer assertion incorrectly counted read-only SelectableText internals; corrected the fixture to reject writable inputs only. Direct cross_cache 1.1.1 is declared solely to implement the disabled media-cache interface: no superclass construction, network, filesystem or IndexedDB initialization. Package presentation uses ephemeral message/author aliases rather than operational IDs; custom builders disable all default link/media fetches. Cached Apache LICENSE files exist and no package NOTICE file is present; final distribution license-bundle checks remain pending. Combined Flutter auth/model/API/chat regression now passes (63 tests).
- UT-004 actual package viewport: absent visibility callback established red. Per-item layout intersection now reports only rendered viewport IDs; an actual scroll test passes and excludes off-screen fetched history. The domain controller retains the foreground/monotonic watermark boundary.
- UT-010 pagination: missing older-page callback established red. Initial package integration lost the current message after prepend. A frame barrier restored the ID but exposed the package loading-sliver/top-padding jump. Reversed package ordering plus a layout-independent, localized loading overlay preserves the current message within two pixels in the actual widget test. Viewport/text tests also pass. The unnecessary frame barrier is removed while green and the same tests are being repeated. This is host widget evidence, not device platform evidence.
- UT-003 controller paging: absent loadOlder/refresh established red. Focused tests pass for coalesced older-page fetches, overlap dedupe, preserving the older continuation during newest refresh and keeping an edited local draft. Neither path marks read. Actual package anchor tests also remain green after removing the frame barrier. Large non-overlapping newest-refresh gaps still need an explicit continuation-recovery case before this umbrella ID is complete.
- IT-007 request/actions controller: absent action/counter-refresh contract established red. A request draft cannot send on load; explicit confirmed acceptance enables the composer only after a fresh summary. Confirmed mute/leave refresh native aggregate counts without read updates or optimistic arithmetic, retain local draft text, and leave clears the private presentation. Focused controller tests pass. Actual preferences/inbox invalidation and routed dialogs remain pending.
- IT-007 provider orchestration: missing account conversation/count families established red. The family now owns guarded controller lifetime and serialized local draft initialization; native aggregate 5 becomes authoritative 2 after accept rather than subtracting one. Focused provider tests pass. A filtered generator invocation unexpectedly removed 171 other generated files; restored those baseline outputs and stopped using filtered generation. A missing select-extension import was corrected separately; neither build-fixture failure counts as behavioral evidence.
- UT-006 stable status dependency: repeated ready status originally rebuilt the open controller and dropped its presentation lifetime (red). Messaging consumers now select the stable enabled/state/read-capability tuple through Riverpod selectAsync. Focused tests pass: unchanged status preserves the controller/draft while confirmed shared actions still refresh native aggregates. Explicit disconnect disposal remains the next case.
- IT-017 disconnect eviction: an expanded provider test caught a disposed controller retaining private history/draft in memory. Disposal now schedules its permitted local draft flush, then clears message, summary, composer and read-watermark state. The disconnect test passes with durable same-owner text retained and hidden; nearby controller/provider tests pass.
- IT-015 fail-closed client admission: the missing-active-account case initially dispatched a status read under the generic guard's isolated-test null allowance (red). Chat providers now require a non-null, matching active account lease; the complete provider test file passes with zero dispatch for missing/stale authority.
- IT-005/IT-017 reconnect generation: missing status generation established backend red; status/connection replies now project the current account generation. The real migrated disconnect/reconnect test passes even when both sampled states are ready. Flutter extension then caught unchanged-ready selection retaining the old controller across a changed generation. Typed status decoding and the selected dependency now include generation; provider/API tests pass, evicting old in-memory content while restoring only the retained device draft.
- IT-001 inbox controller: absent controller established red. Focused tests pass for independent accepted/request buckets (including external peers), preserved empty-page continuation, coalesced pagination and stable-ID replacement without duplicate rows. Reads perform no creation/accept/read operation. A refresh snapshot token discards stale older-page completion; its explicit race assertion and routed UI are still pending.
- UT-011 reconnect push choice: missing initial-choice input established red. Consent now initializes from the existing account preference, while first enablement still defaults on. Both consent tests pass; opening the form performs no mutation.
- IT-001/UT-010 inbox page: absent page established red. The actual provider/widget path now renders independent shared inbox/request tabs, current peer labels and native `99+` aggregates; New message remains an explicit navigation trigger. The widget test opens a typed conversation, changes tabs, then switches active account and proves A's private rows disappear. No start/accept/read call is implemented by the read-only fixture, so accidental mutation would fail the test. Inbox families share the existing account/generation gate and dispose their in-memory rows at the boundary. The page still needs its real router entry/actions and broader accessibility/error-state assertions.
- Green refactor: extracted the shared account/readiness/consent gate and static retry panel for the upcoming private content pages. Inbox and consent widget tests remain green; account, generation and native-read semantics are unchanged.
- AT-003/AT-006 conversation page: absent page established red. The real selected chat widget now composes with an explicit request-accept gate and CraftSky-owned composer. The widget test passes for no composer before accept, one explicit acceptance, durable pending send and a confirmed bubble preserving text edited during the request. A widget/model name collision was resolved by naming the UI MessagingConversationPage; the send test also needed a frame after entering text before tapping the newly enabled button. Those fixture/build corrections are not behavioral red evidence. Foreground viewport reads and draft back/background flushes are connected; report/menu/recovery/navigation/accessibility assertions remain open.
- IT-015 missing-purpose defense: expanded migrated tests exposed exchange and confirmation treating a missing purpose row as ordinary login. Both now require and lock the purpose metadata, accept only consumer purposes and fail closed when it disappears. Normal chat confirmation and the negative cases pass. Full auth regression remains to be repeated after this guard.
- UT-009 uncertain resend reducer/controller: missing repeat and resend entry points established red. Acknowledgment is required for an uncertain operation; explicit repeat receives a new UUID, persists it before dispatch, submits the earlier snapshot and preserves newer composer revisions. Focused tests pass. An initially misplaced reducer method was corrected before green; it is implementation build evidence, not an additional behavioral failure. Shared submission was extracted and existing controller/page tests stayed green.
- UT-009 local resend fault: failed draft persistence initially dropped the original unresolved receipt (red). It now restores that attempt while preserving edits, repairs queued local metadata without network dispatch and keeps recoverable save feedback. Focused draft/controller/page tests pass. Backend auth/chat race suites also passed after the purpose/generation defenses (37.669s/42.185s).

- UT-009 explicit resend UI: the new widget test failed because Send again was absent. The localized duplicate-risk dialog now cancels without dispatch or explicitly repeats the earlier snapshot under a fresh UUID, preserving newer text. Both conversation flow widget tests pass.

- UT-011 / FR-011, FR-014 settings: missing controller/page established red. Focused controller and widget tests pass for read-only initial loading, explicit account push versus shared incoming-policy updates, stale-owner zero dispatch and a cancelable disconnect disclosure retaining hidden local drafts. An incorrect localized title getter was corrected before green. Routed provider acquisition and external-change/failure cases remain open.

- IT-003 / FR-004, FR-018 recipient flow: missing controller/view established red. Focused controller and widget tests pass for CraftSky-only search results excluding self, selection-only availability, unavailable zero-create, explicit canonical start and stale-owner zero dispatch. Existing external conversations remain handled by inbox/history; profile-entry wiring and search race/pagination cases remain open.

- IT-004 client recovery: missing report controller established red; focused tests pass for one uncertain submission and metadata-only receipt confirmation without explanation replay. IT-013 typed route locations first failed on missing route classes; all five approved paths now generate under the authenticated shell without adding a main branch. Routes and nearby provider/inbox/new-message/settings regressions pass (11 tests). Actual cold-render/action/account-boundary route coverage remains open.

- IT-007 / UT-011 conversation actions: actual widget test first failed on absent peer-message report action; now passes for peer text reporting, shared mute, cancelable leave explanation and close only after confirmed leave. Selected-package anchor/viewport/text regressions pass (6 tests). IT-004 report form first failed on absent receipt callback; uncertain form now preserves explanation and checks metadata without replay. Focused report tests pass (4); a finder matching the select widget’s internal text field was narrowed to the explanation key before green.

- REG-003 / IT-013 shell entry: missing Messages tile established red. Compact drawer and large rail navigate once to `/messages` while preserving all five primary branches and existing Settings index; both tests pass. Settings navigation then failed on missing Message settings row; the real production-router widget test now passes, including back stack. Native accepted badge and expanded shell layout regressions remain open.

- UT-004 native badge: missing account-bound badge established red. The shell icon now renders accepted native aggregate `100` as `99+`, excludes request count and hides zero; focused widget plus compact/large Messages navigation tests pass (3). An initial plain-text test filter selected no tests and was corrected to the regex flag; it is not verification evidence.

- UT-004 aggregate refresh: expanded foreground read test failed with zero native refresh callbacks after two authoritative read updates; it now passes with exactly two, never optimistic arithmetic. IT-015 consent return: missing completed activation established controller red; active confirmation now publishes its exact new lease, while another active owner is preserved. AuthComplete previously sent that active chat owner to Feed (widget red); it now opens Messages only if the completion lease still equals the current lease. Focused stale/current-owner and existing auth regressions are being repeated; the synthetic controller must load the registry before completion as the real controller does.

- IT-015 consent-route verification now passes (7 focused auth cases) after the synthetic controller was aligned with real registry initialization. IT-017 live draft flush guard: the expanded widget test found an empty persisted draft when account switching was allowed. Conversation lifetime now registers with the existing unsaved-work guard, flushes locally before activation and unregisters on disposal; all three conversation-flow tests pass. No consent prompt or remote draft queue was added.

- UT-003 disjoint newest-page recovery: the new test exposed the old continuation skipping the unseen interval. In-memory continuation segments now bridge backward to the retained history before resuming its earlier cursor; overlapping IDs dedupe and tombstones remain permanent. Focused history/receipt tests pass (3). The earlier overlapping-refresh fixture now actually includes an overlap. A nested-refresh/pagination race extension remains open. Combined chat/auth regression passed before this addition (86 tests). Focused static analysis found no compile errors; remaining lint cleanup is in progress.

- IT-013 private notification opens: missing chat resolver/destination established red. Opaque chat facts now remain separate from social categories; exact recipient activation precedes private reference resolution, missing binding performs no lookup and a switch during lookup prevents navigation. Runtime then failed on absent chat callbacks; cold-start chat opens now use the private resolver and foreground chat refreshes only chat aggregates/buckets. The combined envelope/coordinator/runtime regressions pass (14 tests). Generation-bound provider and full delayed-navigation delivery assertions remain open.

- UT-006 / FR-015 timer: absent coordinator established red. Focused tests pass for 4s conversation/30s inbox cadence, single in-flight heartbeat, coalescing, background stop and owner loss. Rate-limit extension required clock/deadline support; retained snapshot and 120s Retry-After are honored even on explicit retry. Resume then exposed a canceled deadline with no replacement timer; rescheduling now passes. A test-owned pending timer and invalid paused-to-resumed OS transition were corrected as fixture issues. The real inbox/provider lifecycle path passes with no background heartbeats and immediate resumed refresh (3 combined tests). Stable content/focus during coordinator admission and remaining lifecycle race cases are still open.

- UT-010 stable synchronization wrapper: actual keyboard test caught a new controller and empty text after delayed coordinator admission. The activity wrapper now keeps one child position through admission/status changes; composer identity/text/focus and nearby lifecycle/timer tests pass (4). UT-008 missing canonical-link parser established red; malicious origin/credential/port/identifier inputs are rejected and focused replies preserved. A typed at-primitives error exposed the initial too-narrow catch and is now handled at the pure validation boundary. IT-016 missing resolver then established red; current-viewer post/profile/event compact data, focused-reply canonical root, no request lookup and no failed-first fallback tests pass (2). Bubble/provider integration is being verified.

- IT-016 accepted-bubble integration: missing preview provider established red. The selected chat widget now requests one compact current-viewer card only for accepted text bubbles; actual widget test passes for zero request fetches, one post fetch after accept and zero composer lookup. Nearby policy/resolver/conversation regressions pass (6). UT-008 original links: missing explicit-link renderer established red; a real tap now emits the original URI without any preview lookup, with account-guarded internal navigation or the existing external-link confirmation. Focused link/policy/conversation regressions pass (6).

- IT-003 profile entry: missing button/availability provider established red. The action reads availability without creation, disables unavailable peers and opens explicit recipient selection when allowed; the existing profile action row now supplies it outside block state. Focused profile/new-message/route/action regressions pass (14). A fixture expecting a Material IconButton was corrected to the repository’s ChunkyIconButton. The optional typed recipient query reconstructs selection without a cached profile; its explicit cold-query assertion remains open.

- UT-006 stacked route ownership: actual navigation test found no conversation heartbeat after a covered inbox stopped the shared coordinator. View registrations now independently track route/lifecycle activity and choose the visible registration; a covered route cannot disable the visible conversation. Stack push/pop, stable composer and existing timer/inbox regressions pass (5).

- IT-003 debounced selection: expanded widget test caught the old recipient remaining startable after query editing. The current selection/search epoch is now cleared immediately before debounce, so delayed results cannot reinstate that target; focused recipient tests pass (2). UT-010 cold navigation: ready and disabled Messages initially had no Back action, then revealed the generic fallback reopening Messages. Ready, consent and error gates now carry explicit close callbacks; cold Messages returns to Feed and both route variants pass. Other private leaf pages receive their proper back/fallback callback.

- IT-019 malformed confirmation: the fixed API adapter initially accepted a confirmed send with no message ID. Receipt decoding now fails closed before any draft clearing; API/draft/conversation regressions pass (15). UT-001/IT-015 refreshed status: real migrated test caught pre-refresh report/policy capabilities being projected after eligibility refreshed their grant. Capability projection now follows the eligibility read; the full focused status matrix passes (11.968s). An initial edit landed in the adjacent generic RPC helper and was corrected before green; that compile error is not behavioral evidence.

- IT-006 / IT-004 rate-limit disposition: actual transport plus migrated ledger test failed with uncertain send/report receipts after explicit HTTP 429 rejection. Those responses now persist failed/rejected while repeated operation IDs remain receipt-only; ledger/send regressions pass (3.712s). IT-006/IT-004/IT-015 complete retry-stack evidence: new tests through the pinned Indigo OAuth SDK, purpose resource client and chat adapter already pass for 500/429/connection drop and a valid nonce challenge followed by connection drop. Both send/report have one physical call, or exactly two for that explicit challenge; no ambiguous replay. This adds integration evidence for an existing invariant (8 cases, 1.521s), not a claimed new red/green implementation.

- IT-009 / IT-010 deleted-message checkpoint: expanded migrated sync test caught cursor 0003 committed while the message stayed notified and both device alerts remained pending. The same transaction now keeps a permanent deletion disposition and cancels only matching message alerts/leases. Cursor/outbox, request lifetime and delivery race regressions pass (6.551s); sqlc generation succeeded. No message content or public record was added to storage.

- IT-018 transport provenance: the shared API unwrap fallback first exposed provider prose and dropped the typed original cause/stack. It now uses a static message and preserves original cause, stack and reviewed HTTP metadata; source/API mapping/chat contract regressions pass (21 tests). IT-017 physical account cancellation: a held real Dio adapter initially survived owner activation; the account chat repository now owns cancellation tokens and cancels on provider disposal without closing shared Dio. Focused held-request test passes. Messaging-generation/content cancellation is the next case.

- IT-017 generation cancellation: missing scoped content client established red. Production chat/content/search/preview requests now share an account-and-messaging-generation cancellation lifetime while local status/settings retain their base access. Held Dio requests cancel on generation replacement, the shared adapter stays open, and surrounding provider/inbox/conversation regressions pass (12). Injected repositories remain supported by the existing fixture seams.

- IT-018 recipient diagnostic redaction: a new test found the raw recipient DID in generated Riverpod provider formatting. Availability families now use an immutable typed recipient key whose diagnostic string is redacted while retaining parsed DID equality and wire access. Focused privacy/profile/provider cancellation regressions pass (9).

- IT-015 stale confirmation recovery: the migrated superseded-generation test first returned an OAuth-flow error rather than invalid_handoff, which would leave the client retrying a permanent failure. Confirmation now classifies that stale binding as invalid_handoff without enabling messaging or damaging base access. Focused normal/missing-purpose/stale consent cases pass (4.036s).

- UT-006 / FR-015 transient retention: backend status initially erased known grant capabilities on an eligibility 500. It now keeps those separate from the unavailable service state; core-loss/key/refresh regressions pass (5.524s). Client provider then disposed its same-generation controller on unavailable state; stable admission now retains the loaded controller while authoritative disconnect/owner/generation changes still evict. The real widget gate also hid the preview/composer; it now preserves both with a stale/retry notice and disables network actions until ready. Combined provider/inbox/conversation tests pass (12). A constant-only fixture was corrected before the meaningful widget red. Offline transport-error retention and confirmed local disconnect publication remain the next extensions.

- UT-006 offline status: the focused transport-failure test now passes with same-activation history/draft retention and unavailable state; a definitive authorization failure clears prior authority and subsequent offline reads cannot borrow it. Status is now an owned AsyncNotifier so confirmed local transitions can publish before a refresh.
- IT-017 confirmed disconnect: new provider test observed enabled content surviving the successful disconnect when the next status fetch was offline (red). The settings controller now publishes the authoritative disabled generation immediately; hidden local drafts remain retained. Provider/settings regressions pass (11 tests).
- UT-006 unchanged-revision recovery: new timer test observed no refresh after failure when the revision stayed equal (red). Recovery now refreshes the visible view and rechecks an unavailable status; sync/provider/route lifecycle regressions pass (14 tests).

- UT-011 permission recovery: missing settings consent callback established red. The explicit Review messaging permissions action now opens the full shared-inbox/privacy/report disclosure, preserves the account push choice and starts no OAuth request until Continue. Settings/routes regressions pass (6 tests).
- UT-010 narrow/large-text/keyboard: actual 320x568, 2x text, five-line uncertain composer overflowed by 846 pixels (red). Its controls now live in a bounded scrollable composer; Check status remains reachable and text is retained. Selected-package viewport/anchor and conversation flows pass (8 tests).
- IT-013 queued navigation: missing outcome guard established red. Private opens now carry a delivery-time owner guard; runtime additionally captures the exact activation and latest open, and navigation rechecks it after any frame delay. Missing guards fail closed for private destinations. Notification coordinator/runtime/effect-host regressions pass (11 tests); a missing import during implementation was corrected and is not behavioral failure evidence.
- IT-001 unsupported inbox union: a future last-message envelope with no usable message identity/date originally hid the entire conversation (red). The decoder now omits only that unsupported summary; ordinary malformed messages still fail validation. Fixed API/inbox regressions pass (6 tests). Inbox refreshes also avoid reusing a disposed controller while its family reloads.
- UT-006 safe recovery metadata: migrated status test exposed missing sync state, last-success time and recovery code (red), then a host-timezone timestamp. Status now projects reviewed sync fields without cursors, and status/heartbeat dates normalize to UTC. The owning migrated test passes (1.095s); typed client decoding and offline regression are being verified.

- UT-006 status client/recovery display: typed safe-state/date decoding preserves the last success through offline degradation. The actual conversation then failed on missing Last updated feedback; localized status and heartbeat notices now show the retained local date/time without replacing the composer. Conversation/lifecycle regressions pass (7 tests).
- IT-017 queued startup: a storage barrier exposed queued initialization restoring an already-removed owner's draft lease (red). Registry initialization now has a monotonic request epoch; removed-owner authority cannot reappear from queued work. Draft/account-cleanup regressions pass (4 tests).
- UT-003 refresh/pagination race: a held older result replaced the new gap cursor (red). Its IDs now merge while the newer continuation remains authoritative; all controller cases pass (10 tests).
- IT-003 cold recipient: changing the typed recipient query retained the prior selected DID (red). The view clears stale search/selection state and rechecks the new recipient without creation; all recipient tests pass (3). An initially retained searching flag was corrected before green.
- UT-004 aggregate refresh after send: the confirmed-send extension observed zero refresh callbacks (red). Confirmed send and confirmed receipt recovery now trigger native aggregate refresh; full controller tests pass (10), with no local unread arithmetic.
- IT-018 receipt diagnostics: accidental formatting exposed the operation UUID (red). Receipt and embedded send outcome formatting now redact; wire JSON retains its required operation identity. Focused race checks pass (2.834s).

- IT-016 relationship changes: the existing block controller left a watched current-viewer message preview cached (red). Confirmed block/unblock and shared relationship refreshes now invalidate compact previews and recipient availability. Profile/controller/conversation regressions pass (16 tests).
- REG-001/REG-003/REG-005 broader verification: the initial full Flutter run exposed expected Settings-order fixture changes, a below-viewport Instagram tap, and two existing account-removal tests lacking the new secure-storage platform mock. Those fixtures were corrected, and focused Settings navigation/account-boundary checks pass. The notification secret scan additionally rejected a direct diagnostic owner in the notifications namespace; private-open failures now report from the chat provider, and the notification adapter returns safe unavailable feedback. Existing Dio/private diagnostic/runtime regressions pass (18 tests).
- Initial full Flutter execution also hit the existing 30-second limits in two large-image fixtures while local checks/builds ran concurrently. It was superseded after those failures by `just app-test --concurrency=2 --timeout=2m`; no image processing implementation or test assertion was changed. Final outcome remains in verification.
- MAN-004 local compilation: web build passed (149.2s). The actual exported NOTICES artifact contains the two selected chat packages, cross_cache and Apache license text. iOS simulator and Android debug compilation are running; no signed build, platform run or OS delivery is claimed.
- Configuration handoff: `.env.local.example` now declares blank dedicated operation-key settings. `06-validation-evidence.md` documents local generation without printing a key, stable-key/rotation safety and the separately authorized production prerequisite. No real key was provisioned, and no live-service chat/report or production operation was started.

## Final requirement-to-evidence index (local implementation)

This index supersedes historical “remaining/pending” notes in the chronological
execution log. All final local gates below have finished. AT IDs are acceptance umbrellas
supported by their owning unit/integration/widget suites; they are not invented
additional live-provider executions. Local implementation does not establish the
owner's deferred MAN-001–MAN-004 or AT-024 evidence.

| Requirements | Test IDs / acceptance umbrellas | Owning executable evidence |
|---|---|---|
| BR-001, FR-002, FR-008 | IT-001/007, AT-001/003 | `chat/inbox_test.go`, `actions_test.go`; Flutter inbox/controller/conversation flows |
| BR-002, FR-001, RULE-004 | UT-001/006, IT-005/015, AT-002/014/017/019, REG-001 | Auth scope/capability/consent/OAuth/lifecycle suites; chat status and Flutter recovery/auth handoff |
| BR-003, FR-010 | UT-005/011, IT-004, AT-010 | Go report evidence/transport/ledger; Flutter consent/report controller/form/settings disclosures |
| FR-003 | UT-003, IT-002, AT-004 | Go history/pagination; Flutter ID/tombstone merge, disjoint/raced continuation, actual package scroll anchoring |
| FR-004, FR-018 | IT-003/014, AT-005/027 | Go availability/canonical start/block; Flutter profile availability, CraftSky search, explicit start and cold recipient |
| FR-005, FR-006 | UT-002/009, IT-006/017, AT-006/007/028 | Independent Go/Dart bounds, actual OAuth/transport fault stack, durable receipt ledger, pending/failed/uncertain/repeat and later-edit preservation |
| FR-007, RULE-002 | UT-004, IT-007/009, AT-008/012 | Explicit presented-ID watermarks, native capped counts, request reads, foreground viewport and authoritative aggregate refresh |
| FR-009, RULE-005 | IT-007/011, AT-009/015, REG-002 | Shared mute/leave/readback, current dispatch eligibility, separate social controls/counts |
| FR-011 | IT-008, AT-011 | Conditional authoritative self-declaration merge/conflict tests and explicit shared-policy settings |
| FR-012 | IT-009/010/012, AT-012/013/029 | Atomic baseline/cursor/dedupe/outbox, deleted/unknown/group events, invalid cursor and bounded recovery |
| FR-013, RULE-003 | UT-007, IT-010/011/012, AT-013/015/029, REG-002 | Permanent request generations, catch-up suppression, per-device final checks, exact-token fencing and serialized Firebase payload canaries |
| FR-014, NFR-001 | IT-013/015/019, AT-016/019/021, REG-003 | Fixed authenticated routes, current member/exact child/owner gates, opaque private opens, activation/queued navigation fences and route reconstruction |
| FR-015, NFR-004, NFR-005 | UT-006, IT-009/020, AT-017/022 | Shared four-second account work, capacity/leases/due selection, no-overlap/backoff/Retry-After, transient retention and foreground lifecycle |
| FR-016 | IT-005/013/017, AT-014/016/028, REG-001/006 | No-activity auth expiry, disconnect generation/cancellation, session revocation and exact-owner handoff/late-result rejection |
| FR-017 | UT-008, IT-016, AT-025, REG-004 | Canonical first-link parser, viewer-authorized post/profile/event resolver, reply focus, zero request/composer lookup and preserved explicit links |
| FR-019 | UT-009, IT-017/018, AT-028 | Device-only storage, monotonic queued initialization/save leases, owner-only purge, hidden disconnect retention and restart recovery |
| RULE-001 | IT-014, AT-018 | Either-direction block gates for availability/start/send/push while accessible history remains readable |
| NFR-002 | IT-018, AT-020, REG-005/006 | Private source contexts/serialized sink canaries, diagnostic/route scans, no transcript persistence, retention and account-deletion inventory |
| NFR-003 | IT-019, AT-021 | Registered-route contract/malicious input tests, fixed upstream allowlist, strict camelCase envelopes, cardinality and bounded opaque cursors |
| NFR-006, NFR-008 | UT-010, AT-023/026, MAN-002/003/004 | Actual selected-package widget rendering/labels, keyboard/large text/scroll/focus and local platform builds; real OS/accessibility/platform execution deferred to owner |
| NFR-007 | AT-024, MAN-001 | **Documented gap:** actual granular grants/deployed chat behavior remains owner manual validation; live report acceptance explicitly assumed |

## Implementation choices and deviations

- The approved account/generation provider graph is retained. Pure ChangeNotifier
  controllers owned/disposed by Riverpod families implement inbox, conversation,
  settings, recipient, report and timer behavior; the status AsyncNotifier retains
  same-activation recovery metadata and immediately publishes confirmed disconnect.
  These are testable ownership details, with no new transport or product behavior.
- The optional typed `recipient-did` query reconstructs profile entry on a cold
  New message route; no profile object or token is carried in navigation extras.
- No Lexicon, Tap collection or public PDS schema changed. Schema generation in
  the backend gate is verification only. Unrelated Dart generated hash/format
  drift is restored; feature-owned outputs stay regenerated.
- No commits, pushes, deployment, production configuration or live report/message
  action are included. Dedicated production operation-key provisioning remains a
  release prerequisite. Local configuration instructions are in `06-validation-evidence.md`.
- Internal fault/race tests prove local invariants, not actual provider acceptance,
  device delivery or exactly-once behavior of an external push provider. Manual
  gaps remain explicit rather than being relabeled as passing automation.

- Final gate scheduling: the standalone full backend race run passed affected auth/chat/API/app/account-deletion/migration packages and was superseded by the already-running required isolated `scripts/appview-check` gate, which executes all packages in both integration and race modes against the latest source. This avoids competing full database test stacks; the standalone run is not claimed as a completed all-package pass.

- IT-013 / REG-002 final host integration: the actual root notification host refreshed social newness on a chat open (red). Chat destinations now bypass social-list/count refresh and suppression. A real typed route opens under its guard, a queued stale guard prevents navigation, and existing social-open behavior remains intact; host/runtime regressions pass (8 tests).

- UT-010 recipient accessibility extension: the actual unavailable-recipient page overflowed by 162 pixels at 320x568, 2x text and a 250-pixel keyboard (red). Its selected-recipient notice/actions now scroll in bounded flex space; all four recipient cases pass (10 seconds), including zero creation and cold-query changes.
- UT-010 inbox accessibility extension: the full account/lifecycle/bucket test exposed horizontal tab overflow at 320x568 and 2x text (red). Flexible tab labels now fit beside the native badge while retaining their semantic text; the same scenario passes (11 seconds), including owner switch and native count behavior.
- Final generation succeeded (`dart run build_runner build`, 200 seconds); 76 unrelated generated hash/format outputs were restored. Owned chat/auth/router/notification outputs remain regenerated. The whole-app analyzer reached zero diagnostics before the last two layout changes; its final repeat is running. `git diff --check` passes after generated-output cleanup.
- Isolated backend gate limitation: gofmt/vet/module/Staticcheck/govulncheck and pinned local Tap prerequisites passed. The unit phase reported one failure in `TestSearchImprovementsRelevanceCorpus/true/stockinette_socks`: its unchanged 30-second schema-setup context expired while applying existing migration `000073_unified_pds_record_mutations.up.sql`, before that case's assertions. The failed run was stopped and its isolated services cleaned up; full race/container stages were not reached. The exact unchanged case then passed against worktree-local PostgreSQL using `scripts/appview-test focused ./internal/api -run '^TestSearchImprovementsRelevanceCorpus$/^true$/^stockinette_socks$' -v` (21.781 seconds). This supports a setup-timing limitation under concurrent host work; it does not turn the incomplete isolated gate into a pass.
- Mobile compilation was interrupted to serialize heavy verification after dependency resolution/native dependency downloads. Those attempts are not recorded as successful builds. Existing Kotlin/Swift-package compatibility warnings were warnings, not proven feature incompatibilities. Sequential final attempts remain pending.

- Final static verification after both layout extensions: whole-app `dart analyze --format=machine` exits 0 with zero diagnostics. `git diff --check` is clean.

- REG-003 final full-suite integration: the first completed broad Flutter run finished with 3,150 passes, 38 configured skips and eight menu/Settings fixture failures. The new Settings row requires 16 disclosure indicators and the rail now has ten destinations. The existing adaptive footer rule needs more than 800 pixels to fit ten routes plus compose/legal/Feedback/version; tall-footer fixtures now use 900 pixels, and the 600/800-pixel route-reachability tests remain explicit. No footer policy or primary branch changed. Both complete affected files pass (77 tests, five configured skips). The final complete Flutter repeat uses `flutter test --no-pub --concurrency=4 --timeout=2m`, with the existing successful host-native setup reused and other heavy checks stopped.

- Final complete Flutter repeat passed: 3,159 tests, 38 configured skips, 264 seconds, after all fixture/layout changes. Android debug APK compilation then passed sequentially (116.7 seconds); iOS simulator compilation is next. No install, signed release, OS tap or live message/report action is included.

- Sequential iOS simulator compilation passed (101.3 seconds). Android and iOS exported license bundles include flutter_chat_ui, flutter_chat_core and cross_cache. No platform run/device/accessibility/OS-push acceptance is inferred from these builds. Final web rebuild and controlled isolated backend retry remain.

- Final web compilation passed (58.7 seconds), with exported chat-package license entries verified. All three supported platform targets now compile. The isolated backend gate is being retried alone with `GOMAXPROCS=2` after the focused unchanged setup-timeout case passed; first-attempt failures remain recorded separately.

- IT-013 / FR-014, FR-016, NFR-001 / AC-016, AC-019 authority-race extension: an already-open conversation retained private history after a definitive 404 (observed red). Definitive access loss now clears in-memory history and conversation state, retains the local draft, and increments a content epoch so held older-page, read and send completions cannot repaint it. The new held-completion test and all eleven conversation-controller cases pass. Transport failures and relationship-block errors retain readable history.
- UT-006 remote authority extension: a heartbeat `chat_disconnected` left cached enabled access after a following offline status refresh (observed red). The coordinator now reports definitive authority loss to its account provider, which immediately publishes denied capabilities before status reconciliation. Four static error paths pass. Heartbeat authorization/restriction states separately exposed the same problem (two observed reds); those states now use the same denial path. All six remote cases and nearby provider/controller/sync tests pass (29 tests). Generation is not invented locally; base-account sessions and durable local drafts remain separate. Expected denials do not emit repeated diagnostic issues.

- After the authority extension, full generation completed in 20 seconds; 76 unrelated generated outputs were restored. Whole-app analysis again exits 0 with zero diagnostics and `git diff --check` is clean. Final broad and platform verification follows this source.

- Controlled isolated backend retry completed all 49 Go packages in both ordinary integration and race modes: 6,498 passing test/subtest results in each, zero skipped tests, zero failures. The subsequent image/binary/operational phases are still running; this is not yet a claim that the complete gate passed.

- Final isolated `GOMAXPROCS=2 scripts/appview-check` completed with exit 0 and “all release gates passed”. In addition to all unit/race packages, the gate passed generation/modules/static/security checks, release/audit binary provenance/scans, image builds, empty-migration fail-closed Compose dependency, bounded media probes, migrated database/packaged-console/health smoke and pinned Tap handshake. The Tap version has no safe deterministic event-injection API; no live event ingestion is claimed. Artifact directory: `/private/tmp/craftsky-dm-appview-check-final`. The original setup timeout remains recorded as a separate failed attempt.

- Final complete Flutter suite after authority races and generation passed with exit 0: 3,166 passing tests, 38 configured skips, 274 seconds. The six remote-authority cases also assert the base active-account lease remains unchanged and the owner draft is retained. No chat test is configured to skip. Android/iOS/web final-source rebuilds follow.


## Final completion: 2026-10-19

- Local feature implementation and all selected automated loops are complete. The final source passes the isolated backend release gate, all 49 ordinary/race test packages (6,498 passing test/subtest results per mode, no skipped tests), complete Flutter suite (3,166 passes, 38 configured skips), whole-app static analysis with zero diagnostics, and clean diff checks.
- Final-source supported builds pass: Android debug APK 17.2s, iOS simulator 43.5s, web 63.1s. The exported license bundles for all three contain the selected chat packages and Apache license text. These are compilation/artifact checks, not signed installs or runtime/device acceptance.
- Regeneration is complete and unrelated generated drift restored. Migration prefixes 000001–000102 are unique. No Lexicon/public PDS schema, production configuration, production database, live message or report was changed. No commit, push or deployment was made.
- MAN-001–MAN-004 and AT-024 remain explicitly assigned to the owner; live Bluesky report acceptance is assumed/unverified. Local operation-key provisioning is documented in `06-validation-evidence.md` and was not performed. Production key/release work remains separate.
- Implementation review is deferred to the user's next-stage choice, as required by the implement-tdd exit gate. The implementation plan and final evidence were written and read back before handoff.


## Required-change TDD pass: 2026-10-20

The owner selected “address required changes” from `06-implementation-review.md`.
The existing local high-risk approval remains applicable. This pass supersedes
the earlier completion conclusion for the five reviewed gaps; it does not
perform a new review, live validation, production change, commit or deployment.

| Order | Finding / test | Requirements | Acceptance | Status |
|---|---|---|---|---|
| 1 | IR-001 / IT-013 | FR-014, FR-016, NFR-001 | AC-016, AC-019 | Complete |
| 2 | IR-002 / IT-020 | NFR-004, NFR-005 | AC-017, AC-022 | Complete |
| 3 | IR-003 / UT-006, IT-019 | NFR-004, NFR-005 | AC-017, AC-022 | Complete |
| 4 | IR-004 / IT-006, IT-019 | FR-005, FR-006 | AC-006, AC-007 | Complete |
| 5 | IR-005 / UT-009, IT-006 | FR-006 | AC-007 | Complete |
| 6 | Final verification / evidence | Affected regressions | Existing criteria | Complete |

### IR-001 / IT-013: upstream access-denial contract
- Start with a real migrated chat service, fixed upstream transport and registered
  consumer handlers; exercise authoritative getConvo denial, preserving private
  typed causes and safe consumer errors. Then verify the wire classification
  through ApiChatRepository and the loaded client controller, alongside the
  existing held-completion/privacy regression.

- Red: `TestChatUpstreamConversationDenialReachesConsumer` reached the real migrated
  service/transport and failed on upstream 403 becoming consumer 400/rejected.
  An initial setup attempt found the local test DB stopped; only PostgreSQL/MinIO
  were started, with no AppView or messaging worker. This was not a behavioral red.
- Green: currentConversation now joins authoritative 403/404/410 access denials
  with ErrInaccessibleConversation while preserving the typed provider cause.
  Registered conversation and heartbeat handlers return safe 404/unavailable;
  a 500 control remains transient 503. The upstream/service/HTTP test passes.
- Client regression: the existing held-history/read/send test now supplies the
  consumer HTTP response through ApiChatRepository and ErrorMappingInterceptor,
  rather than injecting an already-normalized ApiException. All eleven controller
  tests pass; local newer drafts remain, private content and delayed bodies do not.

### IR-002 / IT-020: preflight failure backoff and fairness
- Red: two oldest owners failing before the session callback starved a healthy
  third owner in the new real-PostgreSQL scheduler test.
- Green: acquire and commit account/global poll claims before selector and session
  discovery; the whole operation has a 20-second deadline and all later failures
  release their own claim by CAS into the existing durable backoff state. Current
  connection generation is rechecked before reads. No new schema or credentials.
- A scheduler pulse also bounds retry after a local claim/finalization failure.
  Removed the scheduler's now-redundant second authorization pause/failure update.
- The capacity test was adjusted to accept denial arriving before either admitted
  request starts; it still requires two concurrent polls and exactly one rejection.
  This is an ordering-fixture adjustment, not weakened capacity coverage.
- Five focused scheduler/poll tests pass with -race (5.745s), including selector
  failures, exponential delay, 120-second provider delay, retained cursor, released
  claims, cross-worker capacity and healthy-owner progress/shutdown.

### IR-003 / UT-006, IT-019: HTTP retry deadline
- Red: the real chat service/registered heartbeat handler returned 429 with an
  empty Retry-After header for an upstream 120-second rate limit.
- Green: writeChatError emits normalized integer seconds from RateLimitError.
  Real service/HTTP denial and rate tests pass (1.998s).
- Four Flutter coordinator tests pass, including actual ApiChatRepository,
  ErrorMappingInterceptor and HTTP header decoding. Manual retry plus background/
  resume cannot bypass 120 seconds; loaded revision remains intact. Fixture-only
  matcher, JSON content-type, fake-clock queue and timer-cleanup issues were
  corrected separately and are not behavior-red evidence.

### IR-004 / IT-006, IT-019: authoritative failed send receipts
- Red: a real rejected send was returned as HTTP 400 instead of its failed receipt.
- Green: the send handler serializes failed receipts as successful protocol
  results, including duplicate submissions, preserving kind/UUID/failure metadata.
  The real migrated service/ledger/HTTP test proves exactly one upstream send;
  all three correction API tests pass under -race (3.766s).
- Twelve Flutter controller tests pass, including a failed HTTP receipt decoded
  by the production repository; text and the failed state persist locally, with
  no confirmed message or false uncertainty.

### IR-005 / UT-009, IT-006: reconcile before a fresh operation
- Red: a restored uncertain attempt with an available confirmed original receipt
  still allocated a new UUID and made another send. The focused public-controller
  regression failed with one send where zero was required.
- Green: uncertainty initiates original-receipt reconciliation and an independent
  authorized history refresh. Explicit warned resend repeats those safe reads
  before allocating a fresh UUID; confirmed originals resolve without sending.
  Busy state spans reconciliation, while later edits remain local and retained.
- History refresh runs for unresolved/missing receipts too; matching text never
  confirms delivery. The existing content epoch and active-account guards fence
  all private body publication. Definitive access loss still hides history.
- Focused controller tests pass, including uncertain response, lost response,
  confirmed original, held lookup blocking, newer edit preservation, failed
  storage rollback, missing receipt with matching history, explicit acknowledgment
  and no automatic replay. Dart closure/cascade syntax mistakes in the added
  verification fixtures were corrected; these were not behavioral red evidence.
- Whole chat/notification regression invocation exits 0 (210 tests), and whole-app
  analysis exits 0; final artifact readback below corrects the initial zero-diagnostic
  interpretation. Full backend/Flutter/platform gates follow.

### Broad verification correction
- The first correction gate passed all 49 ordinary packages (6,503 test/subtest
  passes, zero skipped tests), then exposed a test-ordering failure under -race:
  the new fairness test cancelled on healthy progress before both admitted early
  owners necessarily entered their failure callbacks (one observed call count 0).
- The fixture now waits for both failure callbacks before cancellation. The
  original exact one-call, durable delay, retained cursor, released-claim and
  healthy-progress assertions remain. This is not a production behavior change.
- The first failed gate remains separate evidence in
  `/private/tmp/craftsky-dm-corrections-check`; a fresh gate will verify the final
  fixture/source, alongside repeated focused race coverage.

- The corrected fairness fixture passes ten consecutive -race repetitions
  (8.438 seconds) in `/private/tmp/craftsky-dm-ir002-race-repeat.log`. The first
  full race run otherwise passed 48 packages / 6,502 test-subtest results; its one
  fixture failure prevented later container stages. A fresh complete gate is
  running in `/private/tmp/craftsky-dm-corrections-check-final`.

- The fresh gate's ordinary and race suites each pass all 49 packages and 6,503
  test/subtest results, with zero failed or skipped tests. Container, media and
  operational phases are still in progress; full-gate completion is not yet claimed.

- The fresh complete backend gate exits 0 and reports “all release gates passed”.
  It includes both full ordinary/race suites, generation/modules/static/security
  checks, release/audit binary provenance/scans, image builds, empty-migration
  fail-closed dependency, bounded media probes, migrated database/console/health
  smoke and pinned Tap handshake. No live event injection is claimed. Artifacts:
  `/private/tmp/craftsky-dm-corrections-check-final`, enclosing `.log` alongside it.

- Complete Flutter suite passes with exit 0: 3,171 tests, 38 configured skips,
  285 seconds, `/private/tmp/craftsky-dm-corrections-flutter-full.log`. No chat
  test is skipped. Final analyzer artifact verification follows the test-only lint cleanup below. Supported-platform final-source builds are in progress.

- Final artifact readback corrected an analyzer interpretation error: exit 0
  included three informational prefer-const lints in a new test fixture, rather
  than zero diagnostics. The fixture's immutable input map is now const. No
  production source changed after the full suite and supported builds. A fresh
  analyzer and controller regression run verify this final fixture-only cleanup.

## Required-change completion: 2026-10-20

- IR-001–IR-005 are implemented and covered by observed red/green boundary
  regressions. All correction loops and final verification are complete.
- Final-source backend gate passes with exit 0: all 49 ordinary/race packages,
  6,503 passing test/subtest results per mode, zero failed/skipped tests, plus
  the complete release/container/operational checks described above.
- Complete Flutter suite passes: 3,171 tests, 38 configured skips, 285 seconds.
  Chat/notification focused coverage passes 210 tests. After the final test-only
  const cleanup, all 15 controller regressions pass again and fresh whole-app
  analysis exits 0 with an empty output artifact: zero diagnostics. No production
  source changed after the full Flutter suite or supported builds.
- Supported builds pass: Android debug APK 17.3s, iOS simulator 52.6s, web 63.1s.
  All three exported license bundles include flutter_chat_ui, flutter_chat_core,
  cross_cache and Apache license text. These are compilation checks; no install,
  signing, device/runtime or live validation is claimed.
- Go formatting and diff checks pass. Dart annotations and SQL/schema/dependencies
  did not change in this correction pass, so no additional generation was required.
  The first correction gate's fixture-order failure, its ten-run race repair, and
  the analyzer exit-code interpretation correction remain explicitly recorded.
- `06-validation-evidence.md` records current results and local artifact paths.
  The original `06-implementation-review.md` verdict remains historical; a fresh
  implementation review has not been performed automatically.
- MAN-001–MAN-004 and AT-024 remain owner-run. Live Bluesky report acceptance is
  assumed/unverified. Operation-key provisioning remains a local/release setup
  prerequisite. No production change, live message/report, commit, push or deploy
  was performed. This completion supersedes the pending correction states above.
- Implementation notes and evidence were written and read back before handoff.
  Per implement-tdd, the next-stage choice is fresh implementation review, a
  manual note, or stop.

## IR-006 correction pass: 2026-10-20

The owner selected “address issues” after the renewed implementation review.
This pass addresses only IR-006, using the existing local implementation scope;
manual/live validation and production changes remain separate. The current
review's finding is the correction input, not a blocking change to requirements.

| Order | Finding / test | Requirements | Acceptance | Status |
|---|---|---|---|---|
| 1 | IR-006 / UT-006, IT-020 | NFR-004, NFR-005, FR-015 | AC-017, AC-022 | Complete |
| 2 | Relevant regressions and evidence | Same criteria | Same criteria | Complete |

- First failing test: hold a same-owner heartbeat, background/resume while it is
  pending, release it as 429 with Retry-After 120, and require no queued/manual
  retry before the deadline while the loaded snapshot remains unchanged.
- After focused green, extend the same behavior to view switching, a response
  completed while backgrounded, and disposal/owner invalidation. Preserve stale
  publication fences and the existing HTTP header propagation tests.
- No backend, auth, permission, migration, dependency or generated-source change
  is planned for this correction. The recent complete backend gate remains
  relevant unchanged-source evidence, rather than a new live/provider claim.

### IR-006 / UT-006: observed red and focused green
- Red: the held heartbeat returned a 120-second rate limit after background/resume;
  the fake-clock test observed 32 calls where two total were allowed before the
  deadline. `/private/tmp/craftsky-dm-ir006-red.log` records the meaningful failure.
- Minimal implementation: scheduling backoff is recorded whenever this captured
  account/generation is current. The existing foreground/view-version check now
  applies after that scheduling update, before failure UI/callback publication.
  Disposed/replaced-owner work remains discarded; no stale view result publishes.
- The same first regression and all existing coordinator tests pass (five tests),
  then view-switch/backgrounded-response and delayed production HTTP mapping
  variants pass (eight tests). Owner-activation/disposal boundaries extend this
  to ten coordinator cases. No additional production change was needed.
- Broader chat/notification coverage passes all 216 tests, exit 0, 24 seconds:
  `/private/tmp/craftsky-dm-ir006-focused.log`. Initial whole-app analysis exited
  0 with one informational if-element style lint in the new fixture; it is fixed
  before final analysis/full-suite verification rather than called zero diagnostics.
- Only the coordinator, its test file and implementation/validation notes change
  in this pass. Backend, API/auth, schema, dependencies and generated code remain
  unchanged. The production diff is the scheduling guard/comment plus the moved
  view-publication guard, with no new timer, queue or remote operation.

- Final coordinator fixture passes all ten tests after style cleanup:
  `/private/tmp/craftsky-dm-ir006-coordinator-final.log`. Final whole-app analysis
  exits 0 with an empty output file, zero diagnostics:
  `/private/tmp/craftsky-dm-ir006-analyze-final.log`. Diff checks pass. The complete
  Flutter suite and supported-platform builds follow; backend is unchanged and
  its recent complete gate is retained without an unnecessary rerun.

- Complete final-source Flutter suite passes with exit 0: 3,177 tests, 38 configured
  skips, 285 seconds; `/private/tmp/craftsky-dm-ir006-flutter-full.log`. No chat
  test is skipped. Supported-platform rebuilds follow; backend remains unchanged.

### IR-006 completion

- The required correction and all selected verification are complete. Final
  source passes all ten coordinator cases, 216 chat/notification regressions,
  the complete Flutter suite (3,177 passes, 38 configured skips, 285s), whole-app
  analysis (exit 0, empty output) and clean diff checks.
- Final-source builds pass: Android debug 16.6s, iOS simulator 44.4s and web 63.8s,
  using the test origin. All three license bundles contain selected chat packages
  and Apache license text. No signing, installation or device/runtime acceptance
  is claimed. Build logs: `/private/tmp/craftsky-dm-ir006-{apk,ios,web}.log`.
- The existing complete backend gate remains valid unchanged-source evidence;
  it was not rerun for this Flutter-only correction. No generation was needed.
- `06-validation-evidence.md` contains this pass's artifacts and boundaries.
  `06-implementation-review.md` is retained as correction input, not relabeled
  approved. A fresh implementation review remains the owner's next-stage choice.
- MAN-001–MAN-004 and AT-024 remain owner-run; live Bluesky report acceptance is
  assumed/unverified. No commit, push, production change, live message/report or
  deployment occurred. No new key or credential was provisioned.
- Implementation notes and validation evidence were written and read back before
  handoff. Next choices per implement-tdd: implementation review, manual note,
  or stop.
