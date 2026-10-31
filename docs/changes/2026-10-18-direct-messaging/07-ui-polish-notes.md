# UI Polish Notes: Direct Messaging

## Full-screen Send via chat picker: 2026-10-31

Owner requested the existing full-screen bottom-up presentation. Send via chat
now uses responsiveModalNavigator and FullscreenModalRoute, matching reports and
other task modals. Replaced the 75%-height sheet with a Scaffold, standard AppBar
title/CloseButton and keyboard-safe body. Recipient/search/draft/send behavior
is unchanged (FR-021 / AC-031). No backend/permission/persistence change.

Actual launch/close test verifies full-screen route type, viewport height, app-bar
title and absence of BottomSheet; all existing picker journeys still pass.
Picker plus responsive-modal tests: **13 passed**, exit 0. Whole-app analysis,
formatting and diff checks clean. Logs:
/private/tmp/craftsky-share-fullscreen-{tests,analyze}.log.
Simulator hot reload succeeds; runtime errors checked separately. No live sends,
commit, push or production action. Physical-device appearance remains owner-run.

## Remove Open messages settings shortcut: 2026-10-31

Owner requested removal of the Open messages button. Removed it and its unused
callback wiring from MessagingSettingsPage/View and the settings route. Existing
controls and the automatic first-enable inbox handoff remain unchanged.
Settings-controller and chat-route regressions: **23 passed**
(/private/tmp/craftsky-dm-remove-open-messages-tests.log). Targeted analysis clean;
format/diff checks pass. Hot reload succeeds; runtime errors empty. No backend,
permission, persistence, dependency, live action, commit or push change.

## Filled selected Messages navigation icon: 2026-10-31

Owner requested a filled selected icon. The shared Messages destination now uses
CraftskyIconsFill.comment, updating both drawer and rail selection. Unselected
and feedback icons remain as before. Targeted Flutter analysis is clean; hot
reload succeeds. No new tests for the glyph-only change. No business behavior,
dependencies, API, persistence, live action, commit or push change.

## Empty initial report reason: 2026-10-30

Owner requested no default report reason. The reason starts null, and Submit
remains disabled until an explicit reason is selected. Entering an explanation
alone does not enable it. The submit handler also guards the absent reason.
Existing supported reasons, explanation validation, report payloads, uncertain
outcomes and status checks without replay remain unchanged.

The regression first reproduced the pre-enabled Submit action. Picker tests use
an explicit search/selection rather than relying on the old preselection scroll
position. Report and routing checks: **17 passed**; targeted analysis: no issues.
Logs: `/private/tmp/craftsky-dm-empty-report-reason-{red,green,analyze}.log`.
Formatting/diff checks pass. DTD discovery found no running debug app, so hot
reload was unavailable. No live report submitted; no backend or permission
change, commit, push or production action.

## Filled send icon: 2026-10-30

Owner requested a filled send icon. Added the semantic sendFilled icon using
Phosphor's filled paperPlaneTilt and selected it in the conversation composer.
Primary button styling, disabled state and send behavior stay unchanged.
Targeted Flutter analysis: no issues. Hot reload succeeds; diff check passes.
No new tests for this glyph-only styling change; no live message sent.

## Full-screen report flow and primary send button: 2026-10-30

### Summary

The owner's reference clarifies that the report presentation should match the
existing full-screen report flow. This supersedes the partial-height sheet below.

### Polish items

| ID | Request | Change | Status |
|---|---|---|---|
| UIP-REPORT-FULL | Full-screen report with app-bar Submit | Messaging uses responsiveModalNavigator and FullscreenModalRoute, the same bottom-up route as report-post. Close remains in the app bar; Submit uses its shared label, text-button styling and action progress. | Done |
| UIP-SEND-PRIMARY | Active send button uses primary colour | ChunkyIconButton accepts the existing button variant, defaulting to secondary for other callers. The composer selects primary, preserving shared disabled styling and current send guards. | Done |

### Verification

- Updated report/route tests first failed against the partial-height sheet and
  old body action: `/private/tmp/craftsky-dm-fullscreen-report-red.log`.
- Focused report/route/composer tests: **29 passed**. Full-screen height, modal
  route, app-bar Submit, keyboard reachability, close/draft retention and report
  status checking without replay are covered. Composer assertions cover empty
  draft disabling and the enabled primary variant.
- Messaging, existing reports, responsive modal, animated buttons and profile
  action/customisation regressions: **204 passed**, exit 0, 14 seconds.
  Command from app/: `fvm flutter test --no-pub test/chat test/moderation/widgets/report_flow_test.dart test/moderation/widgets/report_subject_sheet_test.dart test/router/responsive_modal_navigation_test.dart test/theme/animated_chunky_button_test.dart test/profile/widgets/profile_actions_test.dart test/profile/widgets/profile_customisation_controls_test.dart`.
  Log: `/private/tmp/craftsky-dm-fullscreen-report-regressions.log`.
- `fvm flutter analyze`: no issues
  (`/private/tmp/craftsky-dm-fullscreen-report-analyze.log`). Formatting and
  `git diff --check` pass. Simulator hot reload succeeds; runtime errors empty.

### Scope guardrails

Requirement presentation changed at the owner's request. Business logic, API,
models, migrations, permissions and dependencies: unchanged. No live send/report,
commit, push or production action. Physical-device visual acceptance remains
owner-run. Reopen an already-open report to use the new route presentation.

## Superseded partial-height report sheet: 2026-10-30

User requested a red Report option, bottom-sheet report details and removal of
the second paragraph. The message-menu item uses the shared destructive/red
style. The form now uses a rounded, scroll-controlled modal bottom sheet with
drag handle, a close button, safe areas and a bounded width. Compact screens
use the root navigator; wider screens use their content navigator. Keyboard
insets are applied once outside the inner Scaffold so the form scrolls without
being compressed twice. The first Bluesky-only notice stays; the evidence
paragraph is removed. Submission/controller/payload/uncertain-result logic stays.

Red reproduced the black menu label, full-screen Dialog and unwanted copy
(`/private/tmp/craftsky-dm-report-sheet-red.log`). Focused report/menu/route tests:
24 pass. The report route also verifies keyboard-safe scrolling, close and draft
retention without submitting (`/private/tmp/craftsky-dm-report-sheet-keyboard.log`).
Final chat/existing-report/modal/menu/feedback regressions: **225 passed**, exit 0,
14 seconds (`/private/tmp/craftsky-dm-report-sheet-regressions.log`). Whole-app
analysis: no issues (`/private/tmp/craftsky-dm-report-sheet-analyze-final.log`).
Formatting/diff checks pass. Simulator hot reload succeeds; runtime-error query
is empty. No live report, backend, permissions, API, dependency, commit, push or
production action. An already-open old dialog must be reopened to use the new
route presentation; no hot restart was needed.

## Quote-style CraftSky embeds: 2026-10-30

User requested messaging embeds match quote reposts, reuse the same post embed,
and remove CraftSky post/profile type labels. Extracted QuoteEmbedSurface from
the existing quote-repost wrapper (paper2, borderHair, r2, clipped Material; no
chunky shadow). Both quote reposts and messaging previews now use that surface.

The existing guarded post lookup now retains a compact PostSummaryData.fromPost
in its redacted, memory-only ChatPreview. Message post embeds render the same
PostSummary widget as quote reposts: author/avatar/crafts/handle, timestamp,
project title/text, media and external-card presentation. No additional post
fetch, nested social action controls or backend change. Current-viewer protection
checks, first-target-only resolution, reply focus navigation and request-preview
suppression remain. Profile embeds retain the shared profile identity, crafts
and hero transition, inside the matching surface. Type labels are removed.

Red: post-summary data was absent; preview widget tests found the old chunky card
(`/private/tmp/craftsky-dm-quote-{summary,widget}-red.log`). Focused green: 3 pass
(`/private/tmp/craftsky-dm-quote-widget-green.log`). Final messaging, PostCard,
PostSummary and profile-navigation regressions: **285 passed**, exit 0, 19 seconds
(`/private/tmp/craftsky-dm-quote-regressions.log`). Whole-app analysis: no issues
(`/private/tmp/craftsky-dm-quote-analyze-final.log`). Formatting/diff checks pass.
Simulator hot reload succeeds; runtime errors are empty. No dependency, API,
backend, migration, live message/report, commit, push or production change.

## Muted message timestamps: 2026-10-30

User requested lighter datetime styling inside message bubbles. Configure the
package's timeStyle using CraftSky labelSmall at regular weight (w400), zero
letter spacing and the themed onSurfaceVariant colour for sent and received
bubbles. Timestamp formatting, message content and package rendering remain.
Existing conversation layout tests: 4 pass; whole-app analysis is clean.
Artifacts: `/private/tmp/craftsky-dm-muted-time-{tests,analyze}.log`.
Hot reload succeeds; runtime-error inspection returns none. No live actions,
commit, push or deployment.

## Package-owned message styling: 2026-10-30

User approved package-native rendering and dropping outlined bubbles. The current
conversation uses themed SimpleTextMessage bubbles and ChatMessage wrappers,
including timestamps, grouping/animation and prior opposite-side insets. Existing
previews render through linkPreviewBuilder. The shared long-press/context menu
contains Copy/Open link and eligible received-text reporting; inline report flags
and the custom selectable-link renderer are removed. This supersedes earlier
notes describing outlined custom message bubbles. Draft/request controls and
inbox PostCard-style timestamps remain independent. See 05/06 for final test and
runtime receipts (320 regression tests; clean analysis and simulator reload).

## Conversation timestamps and bubble insets: 2026-10-30

User requested PostCard-style last-message datetime at the far right of each
conversation row, and opposite-side message insets. Accepted/request rows now
reuse RelativeTimeText with the last message's sentAt: compact relative/date
label and localized full datetime tooltip. The date sits at the upper right;
muted/unread indicators remain at the right of the preview below. At narrow
widths with large text the date takes its own right-aligned line; full date
content is preserved instead of overflowing the ListTile. No timestamp is
fabricated for conversations without a last message.

Message bubbles retain their sender alignment. Sent bubbles reserve 32px left
and 12px right; received bubbles reserve 12px left and 32px right, using the
existing spacing tokens. Text, links, previews, reports, visible-message tracking
and history anchoring retain their existing paths.

Red: timestamp was absent and sent/received long bubbles had identical left
edges (`/private/tmp/craftsky-dm-time-insets-red.log`). The first timestamp
layout exposed a real large-text trailing-width overflow; the responsive final
layout passes the existing 320px / 2x text-scale account-isolation tests.
Focused geometry/timestamp suite: **6 passed**
(`/private/tmp/craftsky-dm-time-insets-green3.log`). Messaging/shared timestamp
regressions: **172 passed**, exit 0, 11 seconds
(`/private/tmp/craftsky-dm-time-insets-regressions.log`). Whole-app analysis:
**no issues found** (`/private/tmp/craftsky-dm-time-insets-analyze.log`).
Formatting/diff checks pass. Simulator hot reload succeeds. No API, backend,
persistence, permission, live message/report, commit, push or deployment change.

## Complete profile-data and hero integration: 2026-10-30

Correction to the preceding polish pass: sharing CompactProfileIdentity alone
could not display crafts absent from chat actor payloads, and the header still
used direct navigation without ProfileAvatarHeroSource. The owner identified
both missing behaviors. This section supersedes the preceding claim of complete
shared-profile integration.

Added ChatPeerProfile, backed by the existing account-fenced userProfileProvider
for the peer DID. Conversation/request rows and the conversation header reuse
that shared read for authoritative handles, crafts and avatar customisation.
The chat identity remains visible while the profile read is pending or fails;
profile metadata never delays message history. Reads share the existing provider
cache and may add public AppView profile requests for uncached peers. No backend,
API, migration, auth scope or private-data persistence change.

The adapter wraps avatar and identity in ProfileAvatarHeroSource. Header and
profile-link-preview taps invoke its openProfile callback under the existing
active-account/route guards, carrying avatar, identity and craft transition
metadata to the destination. Recipient rows keep start-conversation behavior;
conversation/request rows keep open-conversation behavior. Profile previews also
use the enriched shared identity and hero. Shared reduced-motion handling applies.

Red: the route regression with a real knitting profile found no CraftIcon
(`/private/tmp/craftsky-dm-profile-hero-red.log`). Green: 15 route tests verify
craft rendering and transition extras through header avatar/name taps, and Back
retains the draft (`/private/tmp/craftsky-dm-profile-hero-green.log`).
Messaging/shared-profile regressions: **277 passed**, exit 0
(`/private/tmp/craftsky-dm-profile-hero-regressions.log`). A separate animation
regression verifies actual avatar/name/handle/craft flights on both push and pop,
plus usable fallback identity while the profile read is pending: **1 passed**
(`/private/tmp/craftsky-dm-profile-hero-flight-final.log`). Whole-app analysis:
**no issues found** (`/private/tmp/craftsky-dm-profile-hero-analyze-final2.log`).
Simulator hot reload succeeds; runtime-error query returns none. No live sends,
reports, production action, commit or push. Manual visual acceptance remains
owner-run; these are local widget-flight and simulator reload receipts.

## Shared profile presentation: 2026-10-30

User requested reuse of the app's shared profile widgets throughout messaging.
Conversation/request rows and the conversation AppBar now delegate identity
rendering to CompactProfileIdentity through the existing chat-data adapter:
shared typography, secondary @handle colour, truncation and missing-handle
presentation. Recipient search also uses CompactProfileIdentity with the crafts
already present in its result data. Search and inbox row spacing matches the
shared account tile (12px avatar gap, no additional minimum leading width).
All avatar displays reuse ProfileAvatar; profile-link previews now use it too,
including the shared initial fallback instead of a square generic thumbnail.
Post/event preview thumbnails retain their existing presentation.

No extra profile fetch, API/model change, persisted field or invented craft /
customisation data. Recipient selection still starts the conversation; row and
header tap actions, busy guards, badges, previews and account fences remain.
The prior uncommitted first-enable change is preserved independently.

Validation: messaging plus CompactProfileIdentity regressions **174 passed**,
exit 0, 9 seconds (`/private/tmp/craftsky-dm-shared-profile-tests-final.log`).
The first command included a nonexistent account-tile test path; all actual
cases passed, and the corrected final command is green. Whole-app analysis:
**no issues found** (`/private/tmp/craftsky-dm-shared-profile-analyze.log`).
Formatting/diff checks pass. Simulator hot reload succeeds; runtime-error query
returns none. No live messages/reports, commit, push or deployment action.

## Conversation profile-header skeleton: 2026-10-21

NFR-006 / FR-003 / AC-023 / UT-010: conversation details now show an avatar,
name and handle skeleton in the AppBar during initial status and content
acquisition, plus the existing message skeleton body. The placeholder matches
the loaded avatar size, identity typography and hit-area height, uses the shared
CraftskySkeleton effect/semantics and remains noninteractive. Loading title
height adapts to text scaling. Loaded profile navigation and silent background
refresh remain unchanged; list and settings headers keep their existing loading
presentation.

Meaningful red: neither acquisition stage had a skeleton in the AppBar.
`/private/tmp/craftsky-dm-header-skeleton-red.log`. The corrected loading,
conversation, route and shared skeleton suites pass **42 tests**, exit 0,
3 seconds; `/private/tmp/craftsky-dm-header-skeleton-green.log`. Scoped analysis
has no issues; `/private/tmp/craftsky-dm-header-skeleton-analyze.log`. Formatting
and diff checks pass. Simulator hot reload succeeds. No data/API, controller,
permissions, persistence, dependencies, commit/deployment or live remote action
changed.

## Silent background refresh: 2026-10-21

FR-015 / NFR-006 / AC-017/AC-023 / UT-006/UT-010/UT-017: removed background
sync/refresh/pagination spinners from messaging AppBars. Deleted the inherited
background-loading presentation signal; inbox and conversation titles render
normally while work continues. The title progress helper remains only for
explicit settings mutations and recipient selection/start actions. Initial
skeletons, action progress, loaded content/empty states, coordinator scheduling,
account fencing, and CraftSky warning/error messages with retry remain intact.

Meaningful red: the routine-sync test found the unwanted title spinner.
`/private/tmp/craftsky-dm-silent-refresh-red.log`. Updated held inbox/conversation
refresh assertions verify retained content with no AppBar spinner. All **174
chat/shared-feedback tests pass**, exit 0, 11 seconds:
`/private/tmp/craftsky-dm-silent-refresh-green.log`. Scoped analysis has no issues:
`/private/tmp/craftsky-dm-silent-refresh-analyze.log`. Formatting/diff checks pass.
Simulator hot reload succeeds and runtime-error inspection returns none. No
business logic, API, permissions, persistence, dependency, commit/deployment or
live remote action change.

## Conversation header opens compact profile: 2026-10-21

FR-003 / NFR-006 / AC-023 / UT-010 / IT-013: the loaded conversation header's
avatar and identity now form one accessible profile action, using the existing
showUserProfileCard helper and peer DID. The header has a minimum 48px hit target,
keeps its prior typography/layout and leaves the background spinner outside the
action. Current account/route checks prevent stale navigation; absent peers keep
an ordinary loading/error title. Back returns to the conversation with its draft.

Red/green and final evidence are recorded in `05-implementation-plan.md`.
**176 chat/compact-profile tests pass**, scoped analysis is clean, and formatting/
diff checks pass. Simulator hot reload succeeds. The change reuses the existing
compact route and expansion behavior; no API, persistence, permission, dependency,
new transport, deployment or live message action changed.

## Participant handles: 2026-10-21

NFR-006 / FR-002/FR-003 / AC-023 / UT-010: added a shared compact chat identity
using the existing ProfileHandle formatting/fallback rules and compact profile
AppBar typography. Chats, pending requests and conversation details show a
secondary muted @handle below a display name. Peers without a name keep one
handle fallback, without duplication. List previews, avatar/badges/actions and
navigation remain intact. Long lines ellipsize; conversation toolbar height
accommodates larger system text without clipping either identity line.

Verification: **29 widget/flow/routing tests pass**, exit 0, 3 seconds, including
both list buckets with preserved previews and narrow/large-text AppBar identity.
Log: `/private/tmp/craftsky-dm-handles-tests.log`. Scoped analysis passes with no
issues: `/private/tmp/craftsky-dm-handles-analyze-final.log`. Formatting/diff checks
pass. Simulator hot reload succeeds and runtime-error inspection returns none.
No business logic, data/API, permissions, dependency, persistence, commit,
deployment or live message action changed.

## Request Accept/Ignore controls: 2026-10-21

FR-008/FR-009, AC-009/AC-012, AT-003/AT-009/UT-010: removed the request explainer,
retained primary Accept request and added secondary Ignore. Buttons share a row
at normal widths and stack for narrow layouts/large text. Ignore uses the
existing guarded leave operation directly and closes only on success; no extra
confirmation. The More-menu Leave confirmation remains unchanged. Pending Ignore
shows progress on that control and disables both choices. Failure uses existing
CraftSky feedback and keeps the request available; accepted chats hide Ignore.

173 chat/shared-feedback tests pass; whole-app analysis is clean. Red/green and
boundary evidence are recorded in `05-implementation-plan.md` and
`/private/tmp/craftsky-dm-ignore-{red,green,regressions-final,analyze-final}.log`.
Localization generation and formatting/diff checks pass. Simulator hot reload
succeeds. No backend/API/permission change or live remote action was performed.

## Skeleton viewport coverage correction: 2026-10-21

The fixed six-row placeholders left empty space on taller displays. Chat and
conversation skeletons now calculate a finite row count from the available
height and minimum themed row height. Recipient search uses the remaining
viewport below its field with the same adaptive list skeleton.

Two tall-viewport regressions reproduce the gap before the fix; both now pass.
The focused loading/inbox/recipient suite passes **19 tests**, exit 0:
`/private/tmp/craftsky-dm-skeleton-fill-{red,green}.log`. Scoped static analysis
passes with no issues: `/private/tmp/craftsky-dm-skeleton-fill-analyze.log`.
Formatting/diff checks pass. Simulator hot reload succeeds. No data, controller,
provider, permission, background-refresh or API behavior changes.

## Messaging skeleton loading: 2026-10-21

Maintainer request / UT-010 / NFR-006 / AC-023: replace initial list and message
history spinners with the existing feed skeleton treatment. This supersedes the
earlier centered-spinner choice for these content views.

- Chats, requests and initial recipient lists reuse `CraftskySkeletonList` with
  the shared `AccountRowSkeleton`. Conversation details use alternating
  message-bubble placeholders through the same wrapper/effect. Spacing and
  radius come from existing theme tokens; placeholders contain no fabricated
  identities or message text.
- Both status acquisition and content acquisition use the matching skeleton.
  Recipient search shows account-row placeholders while initial results load.
  Background fetches preserve content/empty states and the small AppBar loader.
  Settings, reports, authorization and action progress remain unchanged.
- Verification: `flutter test --no-pub test/chat
  test/shared/widgets/craftsky_skeleton_test.dart` passes **150 tests**, exit 0,
  18 seconds. Includes six new initial-loading cases across chats, requests and
  conversation details, status/content acquisition, narrow layout and large
  text. Existing refresh/empty-state and shared reduced-motion, dark-theme,
  accessibility, RTL and pointer-blocking checks pass.
  Log: `/private/tmp/craftsky-dm-skeleton-tests.log`.
- Final search assertion confirms skeletons while the query is pending, then
  normal tap-to-conversation behavior. All **9 recipient-view tests pass**,
  exit 0: `/private/tmp/craftsky-dm-skeleton-search-tests.log`.
- Whole-app `dart analyze --fatal-infos` passes with no issues:
  `/private/tmp/craftsky-dm-skeleton-analyze.log`. Formatting and diff checks pass.
  Simulator hot reload succeeds; subsequent runtime-error query reports none.
- Scope: visual loading states only. No controller/provider lifetime, business
  logic, data model, API, permission, migration, dependency, commit, deployment or
  live message/report change. No new manual/live validation is claimed.

## Profile Message opens conversation directly: 2026-10-21

Profile Message now starts/reuses the canonical conversation on the explicit
tap and navigates directly to `ChatConversationRoute`. It no longer opens the
New message recipient picker or exposes its raw-DID/Start UI. Profile viewing
remains read-only. Known denial retains the info snackbar; failed starts use
the shared error snackbar. The existing button progress prevents overlapping
taps. Completion is fenced to the initiating account, recipient, widget and
current route. The compact transition presentation keeps input absorbed.

Red: the profile action previously made zero start requests;
`/private/tmp/craftsky-dm-profile-direct-red.log`. The focused suite passes
21 tests, including real profile-button → conversation → Messages navigation,
duplicate taps, owner change, disposal, errors and protected diagnostic canaries:
`/private/tmp/craftsky-dm-profile-direct-focused.log`.

Chat/profile/router/shared-feedback regressions pass: 677 tests, 7 configured
skips, 45s; `/private/tmp/craftsky-dm-profile-direct-regressions.log`. Whole-app
analysis is clean: `/private/tmp/craftsky-dm-profile-direct-analyze-final.log`.
Initial fixture compilation and four lint findings were corrected before final
verification. Formatting/diff checks pass. Simulator hot reload succeeded with
no subsequent runtime errors. No backend, OAuth, API URL, live conversation/
message/report, commit or deployment change; actual provider testing is owner-run.

## Requests and New message child routes: 2026-10-21

Requests and New message now join Conversation as declared children of Messages.
Canonical URLs remain `/messages/requests`, `/messages/new` (including the
optional `recipient-did` query), and `/messages/conversations/:conversationId`.
Direct messages settings stays separate at `/settings/messaging`. Profile
Message entry uses the declared nested route, so Back reaches Messages as
approved; inbox navigation still pushes children above the retained list.

Updated the production-hierarchy assertion and added inbox/cold-link Back
cases for both children, plus profile entry. Conversation start now exercises
the real recipient route/controller rather than a sibling placeholder;
Back skips the picker and reaches Messages. Both AppBar and system Back pass.
Red: `/private/tmp/craftsky-dm-messages-children-red.log`. Final focused routing
suite: 15 passes, `/private/tmp/craftsky-dm-messages-children-green.log`.

Full chat/router/profile/notification regressions pass: 779 tests, 7 configured
skips, 90s; `/private/tmp/craftsky-dm-messages-children-regressions.log`.
Whole-app analysis has no issues:
`/private/tmp/craftsky-dm-messages-children-analyze.log`. Formatting and diff
checks pass. Typed routes were regenerated; unrelated generator output was
restored from exact pre-generation snapshots. Simulator hot restart succeeded
and runtime-error inspection was empty. Local only; no backend, auth policy,
live message/report, commit, push or deployment changes.

## Conversation details parent route: 2026-10-21

Conversation details is now a declared child of Messages, with the unchanged
`/messages/conversations/:conversationId` URL. Inbox selection, completed New
message flows and cold links build the Messages parent underneath. Notification
conversation opens use the same declarative destination; an imperative push
alone skipped the parent and returned to the notification's originating page.
Account-current checks and social-notification unread behavior are retained.

A production-route hierarchy test reproduced the missing child before the fix:
`/private/tmp/craftsky-dm-conversation-child-red.log`. Generated-route widget
tests load real conversation UI and verify AppBar/system Back returns to the
list for inbox, cold-link, New message and notification entry. The notification
effect fixture now mirrors the nested hierarchy and checks Back reaches Messages.

All 455 chat/router/notification regressions pass (7 configured skips, 33s):
`/private/tmp/craftsky-dm-conversation-child-regressions.log`. Whole-app analysis
has no issues: `/private/tmp/craftsky-dm-conversation-child-analyze-final.log`.
The initial analysis's three test-style diagnostics were fixed. Full generation
completed; unrelated whitespace/hash output in 85 previously clean generated
files was removed, retaining the generated router changes. Formatting/diff
checks pass. Simulator hot restart loads the updated route graph. No backend,
API URL, live message/report, commit, push or deployment change.

## Tap search result to open conversation: 2026-10-21

Search-result taps now check availability and explicitly start/open the
conversation in one interaction. The search flow has no Start conversation
button or inline denial text. Denied taps use `context.showInfo` with the
existing localized unavailable-recipient message; operational failures use the
existing AppMessenger error effect. The keyboard dismisses on selection and
the title spinner covers checking/starting. Repeated taps cannot overlap checks
or starts; stale controller, route and account results cannot navigate.

Cold profile-recipient URLs remain read-only until an explicit Start action;
availability itself still never creates. No controller/backend contract changed.
The maintainer's search-result tap is now the explicit start action documented
in requirements/acceptance tests. Red tests demonstrate the former extra button
and missing snackbar: `/private/tmp/craftsky-dm-tap-recipient-red.log`.

All nine recipient tests pass, including repeated denial feedback, held checks
and starts, account changes, failure feedback and large text/keyboard layout.
All 129 chat/shared-feedback regressions pass (15s, no skips); scoped analysis
has no findings and formatting/diff checks pass. Artifacts:
`/private/tmp/craftsky-dm-tap-recipient-{green-final,regressions,analyze}.log`.
Simulator hot reload succeeded; subsequent runtime-error inspection was empty.
No live conversation/message/report, backend change, commit or deployment.

## New message navigation: 2026-10-21

The inbox FAB now uses the existing typed route's `push<void>()` instead of
`go()`, preserving Messages underneath New message. The route regression failed
before the fix because the router could not pop. It now verifies the retained
inbox plus AppBar and system Back returning to it. All nine route/recipient-page
tests pass; scoped analysis, formatting and diff checks pass. Artifacts:
`/private/tmp/craftsky-dm-new-message-stack-{red,green,analyze}.log`.
Simulator hot reload succeeded. No backend, route URL or live messaging changes.

## AppMessenger correction: 2026-10-21

UIP-028 / NFR-006, FR-015, AC-017, AC-023: the maintainer's screenshot showed the custom sync banner above the page Scaffold, leaving a black region and moving the AppBar. The preceding pass reused snackbar styling in a separate inline renderer; it did not complete the requested migration to the existing message delivery system.

Remove the bespoke CraftskyMessageBanner renderer and its renderer-only tests. Chat alerts now dispatch through the existing AppMessenger/context.showInfo/showWarning/showError flow, using a non-rendering ChatMessageEffect to publish controller state after layout. It deduplicates unchanged messages, ignores covered/disposed routes, guards stale retry actions and never dismisses another feature's message. Authorization failures use the existing guarded async message path directly.

Apply this across synchronization/degraded-state warnings, settings/recipient errors, draft saves, send failures/uncertainty and report outcomes. Request instructions, incoming-policy/permission states and durable send/report results remain normal in-page text with their existing controls. Uncertain operations retain their explicit retry checks; no automatic remote action is added.

Routine synchronization emits no alert. An inherited loading value supplies only the existing 16px title spinner. Initial inbox/conversation loads keep their centered spinner without an extra title loader. No feedback wrapper adds geometry above the Scaffold; the composer and keyboard state remain stable.

Observed red: the synchronization test still found the custom inline banner. Green dispatches one warning through the recording AppMessenger with the existing retry action, preserving the same editor/controller. The real ScaffoldMessengerImpl test verifies the floating themed snackbar and an unchanged AppBar origin. Additional regressions cover routine syncing, covered-route/disposal guards, deduplication, draft retention, private previews, request acceptance and uncertain report status without replay.

Verification: all **122 chat/shared-messenger tests pass**, exit 0, 9 seconds: `/private/tmp/craftsky-chat-messenger-tests.log`. The first runs exposed a duplicate test-builder argument, missing test localizations, obsolete timestamp/copy assertions, and an initial/background spinner overlap; all are corrected in the green suite. Final-source verification passes the same 122 tests again after cleanup: `/private/tmp/craftsky-chat-messenger-tests-final.log`. Whole-app analysis exits 0 with an empty diagnostics file: `/private/tmp/craftsky-chat-messenger-analyze-final.log`. Formatting and `git diff --check` pass. No backend, auth/permission, API, migration, dependency or live action changed in this correction.

## Messages navigation order: 2026-10-21

UIP-027: move Messages between Profile and Saved in the shared drawer/rail destination list. Update destination labels, route mappings, selection indices and the unread-badge slot together.

Verification: **69 existing widget tests pass**, covering shell navigation, account switching and message unread icons. Focused analysis has no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-messages-nav-order-{tests,analyze}.log`. Local and uncommitted; no native/device check was performed.

## Transparent profile AppBar More action: 2026-10-21

UIP-026: pass the existing transparent AppBar button style through ProfileMoreActionsButton and CraftskyContextMenuButton. The collapsed visitor More action now matches the other header controls, including foreground contrast. Expanded profile actions retain their existing styling and menu behavior.

Regression: the visitor header style assertions failed in light and dark themes before the fix. Extended the existing header contrast test to cover both themes and both self/visitor actions. Verification: **61 widget tests pass** across profile pages, profile actions and shared context menus; focused analysis has no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-profile-menu-surface-{red,tests,analyze}.log`.

Presentation only; local and uncommitted. No native/device check was performed for this follow-up.

## Animate the Message action during expansion: 2026-10-21

UIP-025: add a presentation slot for the same ProfileMessageButton in the profile card's expanding action row. The route supplies it for visitor profiles without a known block. Message now shares the existing clipped width-factor reveal and opacity interval with More, matching the old mute action's slide/fade; it reaches the full profile's exact position before handoff. The compact layout still omits the action. Input remains absorbed during expansion, and the full profile retains its navigation, consent and snackbar callbacks after handoff.

Regression: the authenticated synthetic route test failed because Message was absent at the animation midpoint. It now verifies the bold icon is present inside the partially revealed action group, moves horizontally into place, is fully revealed before handoff, and remains within one pixel of that position afterward. Existing no-mute transition assertions remain.

Verification: **73 widget tests pass**, covering profile cards/transitions, profile menus/page and Message-button behavior. Focused analysis exits 0 with no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-profile-message-slide-{red,green,tests-final,analyze}.log`.

Presentation only; local and uncommitted. No new dependency, API, permission, mutation or live-message action. No native/device check was performed for this follow-up.


## Profile icon and transition correction: 2026-10-20

| ID | Request | Change made | Status |
|---|---|---|---|
| UIP-023 | Thicker outlined Message icon | Use the existing bold outlined `CraftskyIconsBold.comment` in ProfileMessageButton. | Done |
| UIP-024 | No mute icon during compact-to-full expansion | Remove the separate leftover mute control from ProfileCard's interpolated action row. The preceding profile-page change had covered expanded/collapsed page controls but missed this separate transition surface. Keep its More action. | Done |

Both targeted assertions failed before the correction. `flutter test --no-pub test/chat/profile_message_button_test.dart test/profile/widgets/profile_card_test.dart` now passes **26 tests**, including the actual expansion midpoint and near-completion frames, route handoff, large-text spacing and existing Message snackbar behavior. Focused analysis has no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-profile-transition-icons-{red,green,analyze}.log`.

Presentation only; local and uncommitted. No relationship/messaging behavior, permission, API, backend or dependency change. No native/device check was performed for this follow-up.


## Profile actions and messaging feedback: 2026-10-20

| ID | Request | Change made | Status |
|---|---|---|---|
| UIP-021 | Move profile mute/unmute into More | Share the visitor More menu between expanded action row and collapsed AppBar. Remove standalone mute icons; retain state-dependent Mute/Unmute, existing callbacks and busy/block guards, and all other menu actions. | Done |
| UIP-022 | Explain unavailable messaging on tap | Keep Message tappable when the recipient cannot receive messages, with the normal Message tooltip. Use the existing account-guarded AppMessenger informational snackbar for denial. Provider/check failures show generic temporary unavailability; available recipients still open explicit message selection. Existing owner, consent, block and caller-busy protections remain. | Done |

This explicitly requested interaction change is recorded in requirements/acceptance/coding revisions and the UT-015/UT-016 red-green evidence in `05-implementation-plan.md`. It is not treated as styling-only work. Verification: **139 chat/profile widget tests pass**, plus **49 final focused passes** after fixture cleanup; chat/profile analysis has no findings; formatting and `git diff --check` pass. Compact/wide menus, actual expanded/collapsed mute flow, unmute callback, blocked/busy state and denied/failed/allowed recipient paths are covered.

Local and uncommitted. No backend/API/permission/dependency change, migration, deployment or live message/report. No native/device check was performed for this follow-up.


## Requests AppBar and empty-state icon: 2026-10-20

UIP-020: remove the settings action from the requests AppBar. Keep settings available on accepted chats. The requests empty state now uses the existing closed-envelope `CraftskyIcons.email`; accepted chats retain their chat icon.

Both existing inbox widget cases pass, verifying action visibility, the empty-state icon, large-text layout, initial/background loading and account isolation. Focused Dart analysis has no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-dm-requests-ui-{tests,analyze}.log`. Presentation only; local and uncommitted.


## Clarify notification toggle copy: 2026-10-20

UIP-019: the maintainer clarified that the notification toggle remains; only its copy changes. Both consent and settings now say **Push notifications** / **Receive push notifications about new messages.** This replaces the confusing preview-focused explanation. No toggle-removal implementation was made and notification eligibility/payloads are unchanged.

Updated `app/lib/l10n/app_en.arb` and regenerated localizations with `flutter gen-l10n` (passed). Verified both call sites share these strings and `git diff --check` passes. No additional tests were added or rerun for this copy-only change. Local and uncommitted.


## Retain empty state during refresh: 2026-10-20

| ID | Request / Source | Change made | Files | Status |
|---|---|---|---|---|
| UIP-018 | Keep the empty state visible during the small background loader | Remove the loading exclusion from the chats/requests empty-state condition. Initial loads keep the centered spinner; background refresh keeps the successful empty result visible alongside the 16px AppBar spinner until the refreshed result replaces it. | `app/lib/chat/pages/messaging_inbox_page.dart`, `app/test/chat/inbox_page_test.dart` | Done |

Verification: the new retained-empty assertions failed for both accepted chats and requests before the fix. Afterward, `flutter test --no-pub test/chat/inbox_page_test.dart` passes both widget cases, including initial centered loading, background empty-state retention, arriving conversations replacing the empty state and account isolation. Focused Dart analysis exits 0 with no findings; formatting and `git diff --check` pass. Logs: `/private/tmp/craftsky-dm-empty-refresh-{red,green,analyze}.log`.

Scope: presentation only, local and uncommitted. No controller, authorization, API, backend or dependency changes. Native/device checks were not repeated for this one-condition rendering correction.


## Initial loading and Material switches: 2026-10-20

Latest maintainer clarification supersedes the initial-load wording in UIP-011 below: center the initial spinner; reserve small AppBar progress for background fetches. Use Material switches on every messaging platform.

| ID | Request / Source | Change made | Files | Status |
|---|---|---|---|---|
| UIP-016 | Separate initial and background loading presentation | Restore the centered 36px stitch spinner in `ChatLoadingPage`, the inbox/request initial body, and unloaded conversation/settings views. Retain the 16px title spinner for refreshes/pagination with content present and background recipient lookup. Consent/report/start actions retain their existing action progress; remove their redundant title progress. Authorization progress is centered instead of a small top overlay. | `app/lib/chat/widgets/chat_app_bar_title.dart`; messaging inbox, conversation, settings, consent, report, recipient and route pages | Done |
| UIP-017 | Material switches | Use `SwitchListTile` rather than `SwitchListTile.adaptive` in consent and settings, retaining values, callbacks, disabled states and disclosure text. | `messaging_consent_page.dart`, `chat_settings_page.dart` | Done |

Verification:

- `flutter test --no-pub test/chat`: **92 passing tests**, no failures/skips, eight seconds. Log: `/private/tmp/craftsky-dm-loading-tests.log`.
- After simplifying a test-fixture lint, `flutter test --no-pub test/chat/settings_controller_test.dart`: **4 passing tests**. Log: `/private/tmp/craftsky-dm-loading-settings-final.log`.
- `dart analyze --format=machine lib/chat test/chat`: exit 0, no findings. Log: `/private/tmp/craftsky-dm-loading-analyze-final.log`.
- Both inbox buckets verify delayed initial loading is centered with no title spinner, then retain the existing actual pull-refresh and 16px title-spinner assertions. Settings verifies initial-load geometry, retained controls during refresh and spinner removal afterward. Consent and settings run Material-switch assertions with the iOS target platform.
- Formatting and `git diff --check` pass. No native rebuild or physical-device test was performed for this small presentation change.

Scope: UI presentation only; loading admission, ownership, account data, permission flow, switch persistence, APIs, dependencies and backend are unchanged. Local changes remain uncommitted.


## Shared notification presentation: 2026-10-20

Maintainer request: use the existing CraftSky themed snackbar/banner system throughout messaging, including the stale-sync notice shown during a transient upstream failure.

| ID | Request / Source | Change made | Files | Status |
|---|---|---|---|---|
| UIP-013 | Shared notice styling | Add a thin persistent `CraftskyMessageBanner` adapter around the existing `CraftskySnackBarContent`, `MessageAction`, semantic colour surfaces, snackbar shape/elevation and spacing tokens. State owns banner lifetime, so recovery still clears a warning and route disposal cannot dismiss unrelated global snackbars. | `app/lib/shared/messaging/widgets/craftsky_message_banner.dart`, `craftsky_snack_bar.dart` | Done |
| UIP-014 | Apply throughout messaging | Replace plain custom sync/status/authorization panels and inline operational notices with the shared presentation. Covers sync/degraded status, authorization failure, draft-save/send states, request guidance, inability to send, recipient availability, settings/disconnection/policy and report results. Existing themed full-page failure/empty states and form validation remain in their existing components. All callbacks, disabled states, ownership checks, status text, timestamps and retry rules are retained. | `app/lib/chat/widgets/chat_activity_scope.dart`, `chat_content_gate.dart`; `app/lib/chat/pages/chat_conversation_page.dart`, `chat_settings_page.dart`, `messaging_route_page.dart`, `new_message_page.dart`, `report_page.dart` | Done |
| UIP-015 | Readable shared notices | Move the shared notice action below the message on narrow screens or at large text sizes; preserve disabled actions and the snackbar close control. Fix the existing light-theme yellow-on-yellow warning icon by using themed ink. | Shared notice content and widget tests | Done |

Verification (from `app/`):

- `flutter test --no-pub test/chat test/shared/messaging`: **119 passing tests**, no failures or skips, 16 seconds. Log: `/private/tmp/craftsky-dm-banner-tests-final.log`.
- `dart analyze --format=machine lib/chat lib/shared/messaging test/chat test/shared/messaging`: exit 0, no findings. Log: `/private/tmp/craftsky-dm-banner-analyze-final.log`.
- New coverage verifies sync-warning recovery retains the same draft editor, all three severity surfaces in light/dark themes at 320px / 2x text, action enablement, warning-icon contrast and reachable snackbar action/dismiss controls.
- The initial narrow-screen check exposed an action-row overflow, fixed in the shared content. Synthetic light/dark rendering with bundled text/icon fonts exposed the invisible light-theme warning icon; the correction is covered by a contrast regression. The temporary rendering harness was removed.
- Formatting and `git diff --check` pass.

Scope guardrails: presentation only. No requirement behavior, business logic, API, model, migration, permission, dependency, diagnostic or provider retry changes. No live messages/reports, credentials, deployment or commit. Changes remain local and uncommitted. Backend tests and native platform rebuilds were not repeated for this UI-only pass; physical-device/live validation remains maintainer-owned.


## Follow-up requested 2026-10-20

This follow-up supersedes the tab/navigation and shared-correspondent scope of the earlier polish pass below. The CraftSky-profile rule is an explicit product behavior change, implemented through the revised requirements and TDD cases recorded in `01-requirements.md`, `02-acceptance-tests.md`, `04-coding-plan.md` and `05-implementation-plan.md`; it is not classified as styling-only work.

| ID | Request | Final change | Verification |
|---|---|---|---|
| UIP-007 | Move New message to FAB | Use the existing branded FAB on accepted chats; remove its AppBar action. | Typed navigation and 320px / 2x-text widget tests |
| UIP-008 | Separate requests route and badge | Remove both tabs and the tab view; AppBar request action opens `/messages/requests`, with a badge for pending eligible requests, including requests already read. Request back/cold fallback returns to chats. | UT-013 route test; accepted/request rendering and owner isolation |
| UIP-009 | CraftSky profiles only | Server-filter lists, known-ID reads/actions, availability/start, notification opening, synchronization and final delivery. Badges use the same eligible profiles, with capped/bounded authoritative enumeration. Existing private opaque baseline suppression IDs remain so later membership cannot replay old messages or reset request suppression; no peer identity/message body is stored by that exception. New consent no longer requests the unused global-count RPC permission. | IT-021 / UT-012 / IT-022, migrated database and race tests; see `05-implementation-plan.md` |
| UIP-010 | Themed pull-to-refresh | Use `YarnRefreshScrollView` for chats and requests, preserving rows and scroll identity. Retained errors omit a false success check. | Actual pull gestures invoke refresh in both buckets |
| UIP-011 | Discreet loading | Use a 16px stitch spinner next to AppBar titles for initial load/refresh/pagination and related messaging loading states; remove in-history progress overlays. | Controlled refresh tests retain visible rows/messages and verify title feedback; pagination anchor regression |
| UIP-012 | Composer hint and alignment | Hide the visible Message heading, retain its accessible label, show localized Message placeholder, and center input/send control alignment. | Hint/alignment assertions, narrow/large-text/keyboard tests, synthetic render |

Final Flutter validation: **233 passing tests, no skips/failures**, covering `test/chat`, `test/notifications`, `test/router/notification_open_routing_test.dart` and `test/theme/yarn_refresh_scroll_view_test.dart`. Whole-app analysis reports no findings. Three synthetic rendering tests passed; inspected chats, requests, composer and background refresh with bundled fonts. Temporary capture code was removed; images remain at `/private/tmp/craftsky-dm-followup-{inbox-accepted,inbox-request,conversation,refresh}.png`. Final backend/runtime evidence is recorded in `05-implementation-plan.md`.

Scope: the requested local behavior and UI changes are implemented; no migration, dependency addition, production deployment, live message/report or new credential was involved. Changes remain uncommitted. The local AppView is rebuilt for maintainer testing. Physical-device/assistive-technology and live provider validation remain maintainer-owned. Counts fail unavailable rather than return partial totals if enumeration exceeds 20 pages per bucket or the 20-second request deadline.

## Earlier polish pass

## Summary

Requested on 2026-10-20: integrate main first, then align messaging with the existing app, reusing profile, modal, context-action and loading patterns.

Main was fetched and integrated locally in `e99e8d75` (parents `4821d2e1`, `a1a86188`). The localization conflict retained all chat strings and main's additional-crafts labels; generated localizations were regenerated. All 102 migration versions are unique and have matching down migrations. Main's onboarding, settings, platform-link and dependency changes coexist with DM consent and private-storage protections. Nothing was pushed or deployed.

The subsequent UI polish follows NFR-006 / AC-023 and NFR-008 / AC-026. Preview presentation retains FR-017 / AC-025 constraints. UI edits remain uncommitted for review.

## Polish Items

Paths below are relative to `app/lib/chat/` unless otherwise stated.

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | Reuse app controls and forms | Use `ChunkyButton`, `ChunkyIconButton`, `ActionProgress`, `BrandTextField`, shared icons and spacing. Report explanation uses `CraftskyFieldScaffold` around its existing read-only-capable text field; validation and input limits are unchanged. | `pages/messaging_consent_page.dart`, `pages/new_message_page.dart`, `pages/chat_settings_page.dart`, `pages/chat_conversation_page.dart`, `pages/report_page.dart` | Done |
| UIP-002 | Match profile styling | Use existing `ProfileAvatar` for inbox rows, recipient results and conversation headers, including branded initial fallbacks. Recipient results retain existing profile customisation. No new identity lookup or profile action was added. | `pages/messaging_inbox_page.dart`, `pages/new_message_page.dart`, `pages/chat_conversation_page.dart` | Done |
| UIP-003 | Reuse modals and context actions | Use responsive `CraftskyContextMenuButton` for conversation actions, neutral confirmation for uncertain resend, destructive confirmation for leave/disconnect, and `showCraftskyModal` for fullscreen report/permission review. Preserve fullscreen safe areas and confirmation-before-mutation/account checks. | `pages/chat_conversation_page.dart`, `pages/chat_settings_page.dart`, `pages/messaging_route_page.dart` | Done |
| UIP-004 | Match loading, empty and failure states | Use `StitchProgressIndicator` and `CraftskyEmptyState`; retain refresh/retry callbacks, read/pagination logic and the history loading overlay that preserves scroll anchors. | `widgets/chat_content_gate.dart`, `pages/messaging_inbox_page.dart`, `pages/messaging_route_page.dart`, `pages/new_message_page.dart`, `pages/chat_settings_page.dart`, `adapters/flutter_chat_adapter.dart` | Done |
| UIP-005 | Match conversation and preview surfaces | Use the app scaffold background, paper/colour/radius/spacing tokens and ink borders for message bubbles; use `CraftskyCard` for supported content previews. Link targeting, thumbnail authority and preview suppression are unchanged. | `adapters/flutter_chat_adapter.dart`, `widgets/chat_message_preview.dart` | Done |
| UIP-006 | Preserve narrow/large-text usability | Make the recipient page one scrollable surface so the taller branded field, results and Start action remain reachable with the keyboard open. Fixed an overflow found by the existing 320px / 2x-text / keyboard test. Update component finders for shared controls without removing flow assertions. | `pages/new_message_page.dart`; `app/test/chat/conversation_flow_test.dart`, `conversation_page_test.dart`, `settings_controller_test.dart` | Done |

## Verification

Main integration, before polish:

- `flutter pub get --enforce-lockfile` and `flutter gen-l10n`: passed.
- Whole-app `dart analyze --format=machine`: exit 0, no findings.
- `flutter test --no-pub test/auth test/chat test/settings test/notifications`: 515 passes, four existing configured skips.
- `scripts/appview-test focused ./internal/auth -run 'Chat|Onboarding|CallbackAttempt|Optional|EffectiveScope' -race -v`: passed.
- No unresolved Git conflicts, duplicate migration versions or changes to main's lexicon schema content.

Final UI source (commands run from `app/`):

- `dart format` on affected UI and test files: passed, final formatting unchanged.
- `dart analyze --format=machine lib/chat test/chat`: exit 0, no findings. Log: `/private/tmp/craftsky-dm-ui-analyze.log`.
- `flutter test --no-pub test/chat test/profile/widgets/profile_avatar_test.dart test/settings/settings_page_test.dart`: **103 passes, four existing configured subscription skips**, no failures, nine seconds. All messaging tests ran. Log: `/private/tmp/craftsky-dm-ui-tests.log`.
- Existing flow assertions cover request acceptance, mute, report entry, cancelled/confirmed leave, uncertain resend confirmation, disconnect, explicit permission consent, unchanged draft/editor state, visibility/read watermark and pagination anchors. Existing large-text/keyboard tests pass.
- Rendered and visually inspected synthetic conversation, context sheet, destructive confirmation, consent and large-text inbox screens using bundled fonts. Three selected synthetic capture tests passed. The temporary capture harness was removed; images remain under `/private/tmp/craftsky-dm-ui-{conversation,menu,leave,consent,inbox}.png`. No live account/message data was used. Log: `/private/tmp/craftsky-dm-ui-capture.log`.
- `git diff --check`: passed.
- Full backend gate and platform rebuilds were not repeated for presentation-only edits. Physical-device, assistive-technology and actual messaging validation remain maintainer-owned; these local checks do not establish those results. No live report was submitted and no third-party report-testing procedure is claimed.

## Scope Guardrails

- Requirement behavior changed: No.
- Business logic changed: No.
- APIs, data models, migrations, permissions or dependencies changed by UI polish: No. The separate main integration includes upstream changes as requested.
- Account guards, consent text, selection/start callbacks, send/report idempotency and recovery, private-content fetching and controller/storage behavior were retained.
- No production changes, credentials, live messages or live reports were involved. No shared component implementation was modified and no dependencies were added for polish.

## Follow-ups

Manual device/live validation remains with the maintainer as previously agreed. Final implementation review or further small UI feedback can follow this pass.
