# Acceptance Test Specification: Direct Messaging

## Delete for me cases: 2026-10-31

DEL-001 / FR-020 / AC-030: fixed DELETE conversation/message route accepts no
body/query/caller authority. Server requires the narrow delete scope and exact
active child, supported direct membership and CraftSky profiles; revalidates
current message (including a loaded older page) and requires sender=owner before
fixed-audience deleteMessageForSelf. Reject malformed IDs, foreign/received/
unsupported/deleted messages and invalid provider response; no automatic retry.
DEL-002: repository uses fixed URL/encoded IDs and validates deletion response.
Controller removes only after confirmation, retains draft/other messages/cursor,
blocks duplicates, rejects non-owned/local messages, and suppresses stale pages/
outgoing placeholders after success. Failure/stale owner/access/disposal never
publishes success; no stored message archive or schema change.
DEL-003: shared red Delete for me only on confirmed owned text. Explicit shared
confirmation names recipient retention; opening/cancelling sends nothing. Held
request blocks duplicates, successful delete removes bubble with themed success,
failure retains it with themed error. Guard menu/confirmation callbacks across
owner, message/revision, route and lifetime changes; preserve previews/draft.

## Conversation account block cases: 2026-10-31

UT-013 / FR-009, AC-009/023: accepted/request conversation options expose red
Block account. Confirm with existing public-block disclosure, then call shared
profile block mutation for the peer only. CraftSky success/error feedback, no
automatic report/leave and unsent draft retained. No mutation before confirmation
or after cancel, owner/activation change, disposal or connection loss. Held
mutation prevents duplicate dispatch/send/accept; accepted block prevents send
even while the conversation summary is stale. Failures restore controls.

## Conversation report cases: 2026-10-31

IT-004 / FR-010, AC-010: existing POST /v1/chat/reports accepts explicit
subjectType=conversation with operationId, conversationId, reasonType, optional
reason. Message form remains compatible. Reject unknown subject types and mixed
message/conversation targets or caller-supplied peer/transcript/authority fields.
Backend revalidates active exact child/enablement/scopes, direct membership and
CraftSky profiles; derives the peer DID and sends only convoRef to Bluesky's
report audience. Duplicate operation submits once, target changes conflict,
uncertainty never replays; invalid/inaccessible/group/foreign conversations deny.
UT-005: client serializes the conversation target without messageId or DID;
controller shares report validation, operation identity and receipt-only recovery.
UT-013 / IT-013: red More action opens full-screen Report conversation, reason
empty/Submit disabled initially. Explicit Submit only; opening/closing preserves
draft and triggers no report, leave or block. Hide action without capability.

## Messages navigation correction cases: 2026-10-31

UT-010 / IT-013: ready and first-use Messages roots expose the shared drawer
button with no Back. The actual compact shell opens its drawer with Messages
selected; root loading/error and setup use the same leading policy. Children
retain Back. On iOS, swipe conversation details to Messages without opening the
drawer, flushing the latest draft even before the debounce interval; reopening
retains it. Existing explicit Back, canonical route and navigation tests pass.

## Time section cases: 2026-10-31

UT-010 / FR-003/005, NFR-006/008: real adapter shows full-width centred headers
for first/day-change/exact-five-minute boundaries; shorter gaps and sender changes
alone do not add headers. Hide per-bubble time, retain send/error indicators.
UT-010: verify relative/date/year formatting, 12/24h preference and local calendar
boundaries. Older-page insertion recomputes headers without duplicate boundaries
or losing the scroll anchor. Existing pending/preview/read/composer tests pass.

## Immediate composer clearing cases: 2026-10-31

UT-010 / AT-006: hold a send response and assert the input is already empty,
the pending bubble retains the submitted text, and subsequent input survives
confirmation. UT-009: typing the identical text as a new draft must count as a
new edit, and existing failure/restoration/retry tests continue to pass.

## Immediate send bubble cases: 2026-10-31

UT-010 / FR-005, FR-006 / AC-006, AC-007: hold the send response; verify an
immediate standard bubble with spinner and no timestamp. Confirmation replaces
it once, restores the server timestamp and keeps later composer edits. A receipt
without a message retains the bubble without a spinner until history arrives.
UT-009 / FR-006 / AC-007: failure and uncertainty stop the spinner and retain
warning/recovery controls. Restored pending attempts are uncertain, never replayed.
Exercise history-before-receipt, same-text messages, recovery confirmation and
owner/conversation changes. Local placeholders never report read watermarks,
resolve previews, or offer report actions. Confirmed previews still work.

## Full-screen report presentation: 2026-10-30

UT-011 / UT-013 / AT-010 / IT-013: Report message is red in compact and wide
menus. Opening it uses the existing FullscreenModalRoute, covering the full
compact screen, with close and Submit in the app bar. No partial-height sheet.
The Bluesky-only notice remains and the second evidence paragraph is absent.
The enabled composer send control uses the primary variant; an empty draft
still disables it. The form stays
scrollable above the keyboard; closing it restores the unsent conversation
draft. Opening/closing does not submit. Keep existing validation, reason/body
mapping, failure/uncertainty handling and status checking without replay. The
reason starts empty and Submit is disabled, including after entering only an
explanation. Choosing a reason enables Submit without submitting the report.

## Package-owned conversation presentation cases: 2026-10-30

UT-010 / UT-013 / UT-008 / IT-017: plain and tombstone messages use package
bubbles and timestamps, themed to CraftSky without outlines. Preserve the
opposite-side insets, loaded history anchor and visible-only read reporting.
Profile/post previews render once through the package preview slot and keep
navigation; unknown/deleted messages never request previews or offer reports.
Received text reports are available by long press and secondary click; owned
text has Copy/links only. No report before explicit selection. Open link uses
the exact original URI and performs no automatic fetch; Copy is explicit.
Stale menu actions after owner/history/disposal changes are discarded.

## Initial-load optimization cases: 2026-10-21

IT-002 / IT-015 / IT-017 / IT-019, UT-009 / FR-003, FR-019 / NFR-001/002/003/004:
combined details returns metadata plus the first bounded message page using one
getStatus, one getConvo and one getMessages call, with no upstream mutation.
Missing scopes, disabled/revoked child, inaccessible membership or missing
CraftSky profile deny content before history. Preserve blocked readability with
canSend false, request state, tombstones/unknown mapping and opaque continuation.
No partial successful response when any stage fails. The new read route requires
consumer/device auth, does not extend activity, rejects extra input and is
no-store. Flutter uses one combined request, validates its conversation identity,
starts local draft restoration while network is held, waits for both and discards
late owner/access results. Existing pagination, draft, send and cache tests pass.

## Conversation loading header cases: 2026-10-21

UT-010 / NFR-006 / AC-023: conversation status/content acquisition both render
a profile skeleton inside the AppBar and message skeletons in the body, with no
spinner. Both identity lines fit narrow/large-text layouts. List headers stay
unchanged; shared skeleton semantics, reduced-motion and noninteraction apply.

## Silent background refresh cases: 2026-10-21

UT-006 / UT-010 / UT-017 / FR-015 / NFR-006: routine synchronization and held
inbox/conversation refreshes retain content/empty states without an AppBar
spinner or routine snackbar. Warnings/errors still use CraftSky messages with
existing retry actions. Initial skeletons and explicit send/accept/ignore/search
selection/settings mutation progress remain unchanged. Polling, request counts,
account fencing and refresh completion behavior remain covered.

## Conversation header profile navigation: 2026-10-21

UT-010 / IT-013 / FR-003 / AC-023: tapping either the conversation header avatar
or identity pushes the peer DID's profile with the compact presentation request.
Returning preserves the conversation and unsent draft. Keep current-account
navigation guards, existing profile presentation behavior and accessible action
semantics. Loading/error headers without a peer do not offer the action.

## Messaging identity cases: 2026-10-21

UT-010 / NFR-006 / AC-023: accepted/request list rows show name followed by a
secondary @handle and preserve the message preview. The conversation AppBar
shows the same identity without clipping at narrow widths and large text.
Handle-only peers remain legible without a duplicate identity line; navigation,
refresh state, badges and conversation actions remain unchanged.

## Accept/Ignore request cases: 2026-10-21

AT-003, AT-009, UT-010 / FR-008, FR-009 / AC-009, AC-012: request details show
Accept request and Ignore without the explainer or reply composer. Ignore calls
leave exactly once, makes no accept/send call and closes only on success, with
no confirmation modal. While pending, disable both actions and show progress on
Ignore. Failure keeps the request open with CraftSky error feedback. Owner change,
covered/disposed view and capability loss suppress inappropriate late navigation.
Accept still reveals the composer; ignore is absent for accepted conversations.
Controls remain usable on narrow screens with large text.

## Skeleton loading cases: 2026-10-21

UT-010 / NFR-006 / AC-023: initial chats/requests and conversation details show
shared skeletons during both status and content acquisition, without a centered
or title spinner. Recipient-search initial results use account-row skeletons.
Preserve loaded content and empty states during background refresh, showing the
existing 16px title indicator. Cover narrow layout and large text; inherit the
shared skeleton's reduced-motion, dark-theme and loading semantics behavior.

## Account runtime/cache cases: 2026-10-21

UT-006 / IT-013: authenticated ownership preloads status and eligible counts,
retains repository/coordinator across page changes, and does not poll or load
history/lists without a visible messaging view. Signed-out/disabled states do
not fetch private messaging data. Resume refreshes status/counts without hidden
history reads. UT-010 / IT-017: reopen successful accepted/request lists without
another initial fetch; retain up to three idle conversation controllers for two
minutes, evict oldest idle history, cancel eviction on resubscription, and keep
active conversations. Account/activation, generation, logout, disablement and
permission-loss boundaries clear retained private state; pending results remain
fenced. Recipient/report providers still dispose on page exit; draft persistence
and explicit uncertain-send recovery are unchanged.

## Direct profile action cases: 2026-10-21

IT-003, IT-013 / FR-004 / AC-005: profile viewing only checks availability.
An allowed explicit Message tap makes one canonical start request and navigates
directly to conversation details; AppBar/system Back reaches Messages. Denied
recipients dispatch the existing info snackbar and make no start request. Held
starts prevent repeated-tap overlap; failures use themed error feedback and
preserve original diagnostics without recipient/prose leakage. Owner changes
and disposal suppress late navigation/feedback. This supersedes the earlier
profile-to-New-message entry assertions, while cold recipient links stay read-only.

## Messages child-route cases: 2026-10-21

IT-013 / FR-002, FR-004, FR-018: inspect the generated production hierarchy
for Requests, New message and Conversation beneath Messages. For Requests and
New message, verify AppBar/system Back reaches Messages from inbox navigation
and cold URLs; New message profile entry also returns to Messages. An actual
recipient-route Start action opens Messages → Conversation and Back skips the
picker. Existing conversation cold-link/notification coverage remains required.
Canonical URLs and the separate settings destination remain unchanged.

## Search-result start cases: 2026-10-21

IT-003 / FR-004, FR-018 / AC-005, AC-027: one tap on an allowed CraftSky
search result checks availability and opens the canonical conversation, without
a separate Start button. A denied result dispatches localized info through
AppMessenger, with no inline denial or create/send. Repeated denied taps can
show feedback again. Held availability/start requests prove repeated taps do
not overlap; owner changes suppress creation/navigation; request failures use
shared error feedback. Cold recipient routes still require an explicit action.

## Feedback correction cases: 2026-10-21

UT-017 / NFR-006, FR-015 / AC-017, AC-023: synchronization and operation alerts reach the existing AppMessenger. Verify actual floating snackbar rendering, unchanged Scaffold/AppBar origin, deduplication across rebuilds, retry callbacks, hidden/disposed-page guards and no dismissal of unrelated messages. Routine syncing emits no alert and shows only the 16px title spinner; initial loads retain only the centered loader. Composer focus, drafts, loaded previews, explicit request acceptance and uncertain send/report recovery remain intact. Persistent state text need not become an alert. Remove tests for the abandoned custom banner renderer.

## Approved revision cases: combined OAuth/setup, 2026-10-21

The latest requirements revision supersedes prior additional-consent-at-enable assertions. Execute these vertical loops in order; existing auth/chat suites remain regression coverage.

| Test | Requirements / AC | Observable behavior |
|---|---|---|
| AUTH-001 | BR-002, RULE-004, NFR-001 / AC-002, AC-019 | Real login and registration PAR requests include base plus narrow chat scopes. Base-only callback/session/refresh stays valid; granting authority does not enable messaging. Operator/deletion purposes retain their existing scope boundaries. |
| AUTH-002 | FR-001, FR-016, RULE-004 / AC-002, AC-014 | Permission-recovery handoff preserves the disabled/enabled choice, push preference, generation and sync cursor. Already-enabled accounts resume promptly; disabled accounts stay paused. A stale/disconnected initiating generation still fails closed. |
| SETUP-001 | BR-003, FR-001, FR-011, FR-014, FR-016 / AC-002, AC-011, AC-015 | Off settings disclose shared history/privacy/reporting, show only the enable switch and make no preference fetch/write. Explicit enable from Messages confirms readable access then opens the regular conversation list without another tap. Explicit settings visits stay on settings and reveal existing push/audience and Open messages. Disable/re-enable preserves choices, hides/shows controls and retains drafts; failure does not claim success. |
| ROUTE-001 | FR-001, FR-016, NFR-001, NFR-006 / AC-002, AC-014, AC-016, AC-023 | First Messages visit stays in setup after enable until Open messages. Enabled later visits open the inbox; settings are reachable from app settings and inbox. Missing authority has explicit recovery, no private fetch, and no auto-enable. Account switches/disposal discard pending setup results. Material switches and existing loading/feedback conventions remain. |

## Approved revision: profile actions, 2026-10-20

| Test | Requirements / AC | Acceptance change |
|---|---|---|
| UT-015 | NFR-006, NFR-008 / AC-023 | Profile mute/unmute is available only through the shared More menu in expanded and collapsed headers, reflects relationship state, invokes the existing action, respects canToggleMute and busy state, and works in compact sheets and wide popups. |
| UT-016 | FR-004, NFR-006, NFR-008 / AC-005, AC-023, AC-027 | A denied recipient leaves Message tappable; tapping shows themed informational feedback without opening/creating/sending. Provider/check failures use generic unavailable feedback. Available recipients still open selection; explicit consent and ownership/block/busy protections remain. |


## Approved follow-up cases: 2026-10-20

The revision at the top of `01-requirements.md` supersedes earlier non-CraftSky correspondent/native aggregate/tab assertions in this document.

| Test | Requirements / AC | Observable behavior |
|---|---|---|
| IT-021 | BR-001, FR-002, FR-003, FR-004, FR-018, NFR-001 / AC-030 | Mixed accepted/request pages omit non-CraftSky peers and continue opaque pagination. Known ID/history/actions/availability/start/report/open paths cannot expose or act on missing/inactive-profile participants. Existing external conversations work when both indexed profiles exist, without requiring a local registration. Profile removal revokes subsequent access. |
| UT-012 | FR-007, RULE-002, RULE-005, NFR-004 / AC-008, AC-031 | Enumerate filtered authoritative pages, deduplicate IDs, count eligible accepted unread conversations and all pending requests (even read), exclude hidden peers from both counts, cap at 100. Accepted muted/blocked/disabled peers do not count. Cursor cycles, budget exhaustion and provider/database failures never produce successful partial counts. |
| IT-022 | FR-012, FR-013 / AC-030 | Baseline/recovery/log processing excludes ineligible participants without stalling cursors; final delivery rechecks profile eligibility, including profile removal after enqueue and between initial/final checks. No notification is sent for a hidden peer. |
| UT-013 | FR-002, FR-015, FR-018, NFR-006, NFR-008 / AC-023, AC-031 | No chat tabs; New message is a FAB; AppBar request button has pending badge and opens the separate typed request route. Requests have their own title/list/back behavior. Yarn refresh invokes existing refresh, retains rows, and a small title spinner reflects loading. Composer shows `Message` as hint without heading; field/send alignment, semantics, drafts, narrow/large-text/keyboard usability survive. |

Execution order: IT-021 eligibility, UT-012 counts, IT-022 delivery, then UT-013 Flutter presentation/navigation. Existing acceptance tests remain regression coverage with superseded fixtures/assertions updated to this policy.

## 1. Test Strategy

Status: Test design complete; tests proposed, not implemented or executed. Risk: **High**, carried forward from requirements (private messages, optional authority, ambiguous writes and durable delivery).

Sections 8, 12 and 13 of `01-requirements.md` are authoritative. SIM-01–SIM-04 take precedence over historical discussion. This specification creates no extra feature obligations, separate spike, latency target or performance benchmark. There are no blocking product questions. Live-service assumptions remain verification risks during full implementation.

Use Go `testing`, table-driven fixtures, `httptest` upstream adapters, deterministic clocks and channel-controlled races; real PostgreSQL via `internal/testdb.WithMigratedSchema` for durable constraints, transaction/crash windows and owner fencing. Use Flutter `test`/`testWidgets`, Riverpod overrides, recording repositories and controlled futures for presentation and account-switch races. Each acceptance scenario below assigns its assertions to named Go/Flutter cases or explicit manual checks; no shared umbrella acceptance file is required. Mobile process/plugin assertions use the existing journey harness where specified. These assignments do not claim a single existing end-to-end runner.

New paths below are **proposed automation targets**, to confirm in coding planning. Existing conventions: `appview/internal/auth/*_test.go`, `appview/internal/push/dispatcher_test.go`, `app/test/notifications/providers/notification_open_coordinator_test.dart`, `app/test/router/notification_open_routing_test.dart`, and `app/test/observability/link_preview_privacy_test.dart`. The existing `app/integration_test/critical_journeys_test.dart` exercises production Flutter layers with loopback AppView and injected notification-runtime events; it does not prove OAuth, real AppView/PDS writes, APNs/FCM or OS taps. It targets mobile, not web. Use manual controlled live checks only for those external/device boundaries and assistive-technology behavior impractical to prove in host tests.

All automated coverage is planned, not passing evidence. Manual/live verification must record platform/build, fixture identities by safe aliases, action and authoritative outcome without private payloads. Do not perform live messages, reports or shared setting changes as part of this documentation stage.

## 2. Requirement Coverage Matrix

Automated means planned deterministic coverage; Mixed includes controlled live/device verification. Each listed criterion has its corresponding AT scenario, expanded by the lower-level cases below.

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003, AC-005 | AT-001, AT-003, AT-005, IT-001, IT-003 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-014 | AT-002, AT-014, IT-005, REG-001 | Acceptance / Integration / Regression | Yes |
| BR-003 | AC-002, AC-010 | AT-002, AT-010, UT-011, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-002, AC-014, AC-017 | AT-002, AT-014, AT-017, UT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-001, AC-003 | AT-001, AT-003, IT-001 | Acceptance / Integration | Yes |
| FR-003 | AC-004 | AT-004, UT-003, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-005 | AT-005, IT-003 | Acceptance / Integration | Yes |
| FR-005 | AC-006, AC-028 | AT-006, AT-028, UT-002, UT-009 | Acceptance / Unit | Yes |
| FR-006 | AC-007 | AT-007, IT-006 | Acceptance / Integration | Yes |
| FR-007 | AC-008, AC-012 | AT-008, AT-012, UT-004, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-003, AC-009 | AT-003, AT-009, IT-007 | Acceptance / Integration | Yes |
| FR-009 | AC-009, AC-012 | AT-009, AT-012, IT-007 | Acceptance / Integration | Yes |
| FR-010 | AC-010 | AT-010, UT-005, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-011 | AT-011, IT-008 | Acceptance / Integration | Yes |
| FR-012 | AC-012, AC-013 | AT-012, AT-013, IT-009, IT-010 | Acceptance / Integration | Yes |
| FR-013 | AC-013, AC-014, AC-015, AC-029 | AT-013, AT-014, AT-015, AT-029, IT-010, IT-011, IT-012 | Acceptance / Integration | Yes |
| FR-014 | AC-015, AC-016 | AT-015, AT-016, IT-011, IT-013 | Acceptance / Integration | Yes |
| FR-015 | AC-012, AC-017 | AT-012, AT-017, IT-009, UT-006 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-014, AC-016, AC-028 | AT-014, AT-016, AT-028, IT-005, IT-013, IT-017 | Acceptance / Integration | Yes |
| FR-017 | AC-025 | AT-025, UT-008, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-027 | AT-027, IT-003 | Acceptance / Integration | Yes |
| FR-019 | AC-028 | AT-028, UT-009, IT-017 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-018 | AT-018, IT-014 | Acceptance / Integration | Yes |
| RULE-002 | AC-008, AC-009, AC-011 | AT-008, AT-009, AT-011, IT-007, IT-008 | Acceptance / Integration | Yes |
| RULE-003 | AC-015 | AT-015, UT-007, IT-011 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-019 | AT-019, UT-001, IT-015 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-008, AC-015 | AT-008, AT-015, UT-004, IT-011 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-016, AC-019 | AT-016, AT-019, IT-013, IT-015 | Acceptance / Integration | Yes |
| NFR-002 | AC-020, AC-028 | AT-020, AT-028, IT-018, IT-017 | Acceptance / Integration | Yes |
| NFR-003 | AC-021 | AT-021, IT-019 | Acceptance / Integration | Yes |
| NFR-004 | AC-017, AC-022 | AT-017, AT-022, UT-006, IT-020 | Acceptance / Unit / Integration | Yes |
| NFR-005 | AC-022 | AT-022, UT-006, IT-020 | Acceptance / Unit / Integration | Yes |
| NFR-006 | AC-023 | AT-023, UT-010, MAN-002, MAN-003 | Acceptance / Manual | Mixed |
| NFR-007 | AC-024 | AT-024, MAN-001 | Acceptance / Manual | Mixed |
| NFR-008 | AC-026 | AT-026, UT-010, MAN-004 | Acceptance / Manual | Mixed |

## 3. Acceptance Scenarios

All scenarios are Must priority. Each Automation Target identifies the owning cases, proposed paths and assertion split. Paths remain proposed until coding planning; allocations identify verification responsibility without requiring duplicate Go/Flutter umbrella files. Automated UI/adapter tests establish only their named boundaries. MAN-001 owns actual service behavior, MAN-002 owns real mobile push/OS taps, MAN-003 owns web/platform/assistive-technology interaction, and MAN-004 owns package-fit planning and platform execution.

### AT-001: Interoperable paginated inbox
Requirement IDs: BR-001, FR-002  
Acceptance Criteria: AC-001  
Priority: Must  
Level: Acceptance  
Automation Target: IT-001: Go `appview/internal/chat/inbox_test.go` verifies upstream filters/cursors and external identities; Flutter `app/test/chat/inbox_page_test.dart` verifies visible rows, request distinction and pagination. Group push/count exclusions: IT-011/UT-004.

```gherkin
Feature: Direct messaging
  Scenario: Interoperable paginated inbox
    Given accepted direct conversations created in CraftSky and another client, a non-CraftSky correspondent, and group-only intervening pages
    When the owner paginates accepted conversations and requests to cursor exhaustion
    Then all supported direct conversations appear once with upstream identities, latest-message, unread, mute and request state
    And filtered-empty pages do not end pagination and groups produce no messages, badge contribution or push
```

### AT-002: Optional enablement and honest state
Requirement IDs: BR-002, BR-003, FR-001  
Acceptance Criteria: AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: UT-011: Flutter `app/test/chat/messaging_consent_test.dart` verifies disclosed meaning and absence of prohibited promises before authorization. UT-001/IT-005/IT-015: Go capability/OAuth/lifecycle suites verify grant outcomes; Flutter `app/test/chat/sync_state_test.dart` verifies the resulting recovery states. Actual consent/eligibility: MAN-001.

```gherkin
Feature: Direct messaging
  Scenario: Optional enablement and honest state
    Given messaging is disabled and ordinary CraftSky access is valid
    When login/registration requests chat permissions alongside base authority
    Then a base-only grant retains ordinary access and OAuth alone does not enable messaging
    When the owner visits Messages and explicitly enables CraftSky messaging
    Then Direct messages setup explains shared history/settings, Bluesky storage and Bluesky-only reporting before enablement
    And push/audience controls appear only once enabled and Open messages explicitly enters the inbox
    And enablement and accompanying messaging explanations promise neither a CraftSky-only inbox/privacy boundary, PDS message portability nor end-to-end encryption
    And these meaning-based assertions cover localized copy and allow an accurate explicit denial of those promises
    And ordinary access survives rejected chat authority
    And ready/syncing, authorization-needed, restricted and unavailable states reflect the result without fabricated restriction reasons or an empty-success inbox
```

### AT-003: Explicit request acceptance
Requirement IDs: BR-001, FR-002, FR-008  
Acceptance Criteria: AC-003  
Priority: Must  
Level: Acceptance  
Automation Target: IT-001/IT-007: Go `appview/internal/chat/inbox_test.go` and `appview/internal/chat/actions_test.go` verify request/read/accept contracts; Flutter `app/test/chat/inbox_page_test.dart` and `app/test/chat/conversation_actions_test.dart` verify request labels and disabled composer until success.

```gherkin
Feature: Direct messaging
  Scenario: Explicit request acceptance
    Given a direct request and an accepted conversation exist
    When the owner opens and reads the request, then explicitly accepts it
    Then the request is visibly distinct and reading updates unread state without acceptance or removal
    And the composer stays disabled until authoritative acceptance succeeds, including an already-accepted result without a revision
    And failure leaves the composer disabled without using send as implicit acceptance
```

### AT-004: Stable safe history
Requirement IDs: FR-003  
Acceptance Criteria: AC-004  
Priority: Must  
Level: Acceptance  
Automation Target: UT-003/IT-002: Go `appview/internal/chat/history_test.go` verifies history adapter/cursors; Flutter `app/test/chat/history_controller_test.dart` verifies mapping, dedupe and deletion; Flutter `app/test/chat/conversation_page_test.dart` (UT-010) verifies the scroll anchor under page insertion.

```gherkin
Feature: Direct messaging
  Scenario: Stable safe history
    Given history pages contain text, duplicates, deleted, embed/reply/reaction and unknown variants, and short or filtered-empty pages with cursors
    When the owner loads older history
    Then chronological display and visible scroll anchor are preserved
    And valid cursors continue, messages deduplicate, and unsupported/deleted content has safe placeholders without raw payloads
```

### AT-005: Availability before canonical creation
Requirement IDs: BR-001, FR-004  
Acceptance Criteria: AC-005  
Priority: Must  
Level: Acceptance  
Automation Target: IT-003: Go `appview/internal/chat/start_test.go` verifies no-create availability, exactly-one-peer validation and canonical reuse; Flutter `app/test/chat/new_message_page_test.dart` verifies explicit start and denied selections.

```gherkin
Feature: Direct messaging
  Scenario: Availability before canonical creation
    Given a profile and an authenticated owner with no existing direct conversation
    When availability is checked and then Start message is explicitly selected repeatedly
    Then checking alone creates nothing and permitted explicit starts reuse the canonical direct conversation
    And self-targets, extra participants, either-direction blocks, disabled incoming policy and unavailable recipients create or send nothing
```

### AT-006: Valid text and confirmed send states
Requirement IDs: FR-005  
Acceptance Criteria: AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: UT-002: Go `appview/internal/chat/validation_test.go` and Flutter `app/test/chat/message_validation_test.dart` independently verify text boundaries/no truncation. IT-006: Go `appview/internal/chat/send_integration_test.go` verifies no invalid upstream call; Flutter `app/test/chat/send_flow_test.dart` verifies retained text and pending/confirmed/failed/uncertain UI.

```gherkin
Feature: Direct messaging
  Scenario: Valid text and confirmed send states
    Given text fixtures spanning blank input, emoji, combining marks and both upstream length limits
    When the owner submits each fixture through Flutter and directly through AppView
    Then both layers enforce nonblank, 1000 graphemes and 10000 UTF-8 bytes without truncation
    And invalid text is retained without upstream send
    And valid text progresses through pending to confirmed only after an authoritative result, or accurately shows failed or uncertain
```

### AT-007: Uncertain sends and deliberate recovery
Requirement IDs: FR-006  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: IT-006: Go `appview/internal/chat/send_integration_test.go` owns durable identity, retry/crash barriers and upstream send-call counts; Flutter `app/test/chat/send_flow_test.dart` owns history-refresh recovery, retained text and explicit duplicate-risk confirmation before a new operation.

```gherkin
Feature: Direct messaging
  Scenario: Uncertain sends and deliberate recovery
    Given a stable owner-scoped send operation and an upstream response lost after possible acceptance
    When the same operation is submitted concurrently, retried after reconnect or resumed after crash
    Then no additional upstream send occurs and unresolved delivery remains uncertain
    And history refresh never confirms by identical text alone or automatically replays
    And text remains available and only explicit Send again after a duplicate-risk explanation creates a new operation
```

### AT-008: Native unread counts and presented-message reads
Requirement IDs: FR-007, RULE-002, RULE-005  
Acceptance Criteria: AC-008  
Priority: Must  
Level: Acceptance  
Automation Target: UT-004/IT-007: Go `appview/internal/chat/read_test.go` and `appview/internal/chat/actions_test.go` own explicit message watermark/upstream calls; Flutter `app/test/chat/unread_state_test.dart` and `app/test/chat/conversation_actions_test.dart` own row/badge mapping, capped refresh and mute/request presentation.

```gherkin
Feature: Direct messaging
  Scenario: Native unread counts and presented-message reads
    Given a foreground conversation has presented message M, later unseen arrivals, requests and capped upstream aggregate counts
    When read, remote-read, conversation-mute and profile-mute changes occur
    Then updateRead names M and does not mark unseen or background arrivals read
    And rows show upstream message counts and badges use only getUnreadCounts with includeGroupChats=false and upstream exclusions
    And sentinel 100 displays as 99+ and causes refresh instead of exact optimistic subtraction
    And reading a request does not accept/remove it, conversation mute does not mark read, and profile mute changes neither DM read state nor badge
```

### AT-009: Shared actions and truthful leave
Requirement IDs: FR-008, FR-009, RULE-002  
Acceptance Criteria: AC-009  
Priority: Must  
Level: Acceptance  
Automation Target: IT-007: Go `appview/internal/chat/actions_test.go` verifies authoritative actions, failed responses and pending-push cancellation; Flutter `app/test/chat/conversation_actions_test.dart` verifies confirmation/cancel and truthful leave copy. MAN-001 verifies actual peer history and service-dependent leave/restart effects.

```gherkin
Feature: Direct messaging
  Scenario: Shared actions and truthful leave
    Given an accessible direct conversation or request with pending push
    When the owner accepts, mutes, unmutes or confirms or cancels leave, including failed responses
    Then only authoritative successes change confirmed UI and delivery eligibility
    And leave confirmation explains verified owner-history effects without promising permanent dismissal or recoverable history
    And cancellation changes nothing, successful leave cancels pending push and the peer history is unaffected
```

### AT-010: Bluesky-only message reports
Requirement IDs: BR-003, FR-010  
Acceptance Criteria: AC-010  
Priority: Must  
Level: Acceptance  
Automation Target: UT-005/IT-004: Go `appview/internal/chat/report_test.go` and `appview/internal/chat/report_integration_test.go` own evidence, bounds, routing and retry-stack faults; Flutter `app/test/chat/report_page_test.dart` plus UT-011 owns Bluesky destination/evidence disclosure and accepted/failed/uncertain UI. MAN-001 owns actual service acceptance.

```gherkin
Feature: Direct messaging
  Scenario: Bluesky-only message reports
    Given an accessible message, supported reason and optional explanation
    When the owner sees Bluesky named as the report recipient and the evidence scope, then confirms reporting
    Then the fixed moderation audience receives reasonType, bounded optional reason and an authorized messageRef with the sender DID, conversation and message IDs
    And no transcript, moderator context RPC or CraftSky moderator report is created
    And only accepted results show success; failures or uncertain results stay recoverable without blind replay or destination fallback
```

### AT-011: Conditional shared incoming policy
Requirement IDs: FR-011, RULE-002  
Acceptance Criteria: AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: IT-008: Go `appview/internal/chat/declaration_integration_test.go` owns conditional create/update, conflict merge and readback; Flutter `app/test/chat/preferences_page_test.dart` owns explicit cross-app consent and persistence/propagation/failure copy. MAN-001 checks actual shared-service propagation.

```gherkin
Feature: Direct messaging
  Scenario: Conditional shared incoming policy
    Given an existing or missing declaration with unrelated fields and a concurrent writer
    When the owner enables/disables chat or explicitly selects everyone, following or nobody
    Then enable/disable leaves the declaration unchanged
    And explicit changes explain cross-app effect and conditionally merge only allowIncoming, preserving allowGroupInvites and unknown fields
    And create uses an absence precondition and conflicts refetch/merge
    And readback confirms persistence, external changes refresh, failures do not show success, and eventual propagation does not promise erasure of existing conversations
```

### AT-012: Authoritative event reconciliation
Requirement IDs: FR-007, FR-009, FR-012, FR-015  
Acceptance Criteria: AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: IT-009: Go `appview/internal/chat/sync_integration_test.go` owns tolerant envelopes, cursor preservation and authoritative reconciliation; Flutter `app/test/chat/sync_controller_test.dart` owns refreshed views and deleted-content handling. IT-010 owns durable progress barriers.

```gherkin
Feature: Direct messaging
  Scenario: Authoritative event reconciliation
    Given message, read, request, mute, leave, delete, unknown and group events with repeated or reordered pages
    When polling resumes or clients reconnect
    Then views reconcile to authoritative state and deleted cached text does not reappear
    And unknown envelopes and revisions permit progress without crashes
    And no cursor commits ahead of required durable processing
```

### AT-013: Crash-consistent logical notifications
Requirement IDs: FR-012, FR-013  
Acceptance Criteria: AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: IT-010/IT-012: Go `appview/internal/chat/sync_crash_test.go`, `appview/internal/push/chat_dispatcher_test.go` and `appview/internal/chat/notification_policy_integration_test.go` own transaction/replay/dedupe/baseline/job assertions; no duplicate Flutter crash suite is required.

```gherkin
Feature: Direct messaging
  Scenario: Crash-consistent logical notifications
    Given a first-enable baseline and eligible incoming events with injected crashes at each durable processing boundary
    When workers restart and replay pages
    Then pre-enable history never alerts and normal accepted messages deduplicate by account/message
    And each logical notification/subscription has at most one delivery job
    And request and catch-up work follows its separate policy without claiming exactly-once provider delivery
```

### AT-014: Device and account revocation fencing
Requirement IDs: BR-002, FR-001, FR-013, FR-016  
Acceptance Criteria: AC-014  
Priority: Must  
Level: Acceptance  
Automation Target: IT-005: Go `appview/internal/auth/chat_lifecycle_test.go` and `appview/internal/chat/lifecycle_integration_test.go` own session/epoch/expiry and stale work fences; Flutter `app/test/chat/sync_state_test.dart` owns paused/reauthentication states. IT-013/IT-017 own private UI and draft fencing.

```gherkin
Feature: Direct messaging
  Scenario: Device and account revocation fencing
    Given two usable device sessions, pending work and an in-flight worker
    When one device logs out, chat disconnects, sign-out-all occurs, the last usable session expires/revokes, or owner deletion begins
    Then device logout affects only that device while another valid session continues
    And account-wide or last-session loss stops new unauthorized work and fences/cancels stale jobs
    And polling never extends inactivity expiry, paused/reauthentication guidance appears, and unrelated authorized access continues
    And already-issued network requests are not described as retractable
```

### AT-015: Private account-wide push
Requirement IDs: FR-013, FR-014, RULE-003, RULE-005  
Acceptance Criteria: AC-015  
Priority: Must  
Level: Acceptance  
Automation Target: UT-007/IT-011: Go `appview/internal/push/chat_payload_test.go` and `appview/internal/push/chat_dispatcher_test.go` own serialized payloads and dispatch-time suppression. Flutter `app/test/chat/push_preferences_test.dart` owns default-on visible account preference; MAN-002 owns actual per-device permission/delivery behavior.

```gherkin
Feature: Direct messaging
  Scenario: Private account-wide push
    Given incoming private-text canaries, owner-sent messages and multiple device subscriptions
    When a push is prepared and then current read, mute, block, leave, toggle or subscription state changes before dispatch
    Then only eligible incoming direct messages dispatch after current checks
    And visible copy uses sender display name, handle fallback or generic fallback with generic message copy and no private content in any provider field
    And the default-on account-wide toggle is visible during enablement and respects per-device OS permissions without changing Bluesky preferences
    And profile mute alone does not suppress DM push but conversation mute and blocks do
```

### AT-016: Account-safe notification opens and access
Requirement IDs: FR-014, FR-016, NFR-001  
Acceptance Criteria: AC-016  
Priority: Must  
Level: Acceptance  
Automation Target: IT-013: Go `appview/internal/chat/authorization_test.go` owns every-route owner/membership denial; Flutter `app/test/router/chat_notification_open_routing_test.dart` owns retained-account leases, safe routing and delayed-result races. Extend existing `app/integration_test/critical_journeys_test.dart` for process/runtime cold-start navigation with injected opens. MAN-002 owns real OS taps.

```gherkin
Feature: Direct messaging
  Scenario: Account-safe notification opens and access
    Given a notification belongs to account A and the device is cold, logged out, using B, disconnected or missing membership
    When the notification opens or another owner guesses conversation/message identifiers for any operation
    Then routing waits for the correct authenticated permitted owner or shows a safe unavailable/authorization destination
    And no A data is fetched or rendered under B, including delayed responses after a switch
    And cross-owner reads, sends, read updates, mutations and reports are denied
```

### AT-017: Classified failure and retained owner state
Requirement IDs: FR-001, FR-015, NFR-004  
Acceptance Criteria: AC-017  
Priority: Must  
Level: Acceptance  
Automation Target: UT-006/IT-009/IT-020: Go `appview/internal/chat/scheduler_test.go`, `appview/internal/chat/sync_integration_test.go` and `appview/internal/chat/scheduler_integration_test.go` own bounded safe retries and healthy-account progress; Flutter `app/test/chat/sync_state_test.dart` and `app/test/chat/sync_controller_test.dart` own stale/degraded/retry UI and foreground/background read behavior.

```gherkin
Feature: Direct messaging
  Scenario: Classified failure and retained owner state
    Given loaded owner content and offline, expired/revoked credentials, 429, 5xx or timeout failures
    When foreground refresh, resume or reconnect runs
    Then recovery states are classified, stale content remains only for the current authenticated owner, and retry is available
    And safe retries are bounded with backoff, healthy owners continue, and backgrounding marks nothing read
```

### AT-018: Block enforcement without history deletion
Requirement IDs: RULE-001  
Acceptance Criteria: AC-018  
Priority: Must  
Level: Acceptance  
Automation Target: IT-014/IT-011: Go `appview/internal/chat/block_integration_test.go` and `appview/internal/push/chat_dispatcher_test.go` own server rejection/suppression and history/report access; Flutter `app/test/chat/conversation_actions_test.dart` owns disabled send/unavailable history and fresh availability after unblock.

```gherkin
Feature: Direct messaging
  Scenario: Block enforcement without history deletion
    Given an indexed block in either direction or newly confirmed owner block with accessible history and queued push
    When the owner tries starting or sending and the dispatcher runs, then the owner unblocks
    Then server creation/send is rejected, sending is disabled and push is suppressed
    And accessible history/reporting remains available without bypassing upstream history denial
    And unblock triggers fresh availability rather than unconditional sending permission
```

### AT-019: Semantic least-authority capabilities
Requirement IDs: RULE-004, NFR-001  
Acceptance Criteria: AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: UT-001/IT-015: Go `appview/internal/auth/effective_scopes_test.go`, `appview/internal/auth/chat_oauth_integration_test.go` and `appview/internal/chat/transport_test.go` own semantic grants, optional flows, session selection and fixed routing. IT-019/IT-018 verify serialized consumer responses contain no OAuth/DPoP handoff. MAN-001 owns actual granular consent/refresh acceptance.

```gherkin
Feature: Direct messaging
  Scenario: Semantic least-authority capabilities
    Given base-only, partially granted and fully chat-capable sessions with equivalent normalized RPC scope spellings
    When login, refresh, reauthorization, foreground calls and background selection evaluate capability
    Then base login/refresh stays valid despite advertised optional permissions
    And each operation uses only a current owner session with its required effective methods and fixed audience
    And combined lxm and reordered/encoded equivalent scopes work without wildcard or wrong-audience authority
    And no full-chat/group/export/account-delete authority is requested and no OAuth token or DPoP key reaches Flutter
    And missing reporting/declaration authority fails its capability gate while ordinary access remains valid
```

### AT-020: Serialized privacy and minimal retention
Requirement IDs: NFR-002  
Acceptance Criteria: AC-020  
Priority: Must  
Level: Acceptance  
Automation Target: IT-018/REG-005/REG-006: Go `appview/internal/chat/privacy_test.go` owns DB/sink/public-write and cleanup evidence; Flutter `app/test/observability/chat_privacy_test.dart` owns client serialized sinks. IT-017 verifies private local drafts, with mobile storage/process execution in the existing critical journey extension. No source-level keyword scan substitutes for serialized assertions.

```gherkin
Feature: Direct messaging
  Scenario: Serialized privacy and minimal retention
    Given distinct message, draft, report, credential, private-routing and raw-error canaries in successful and failing paths
    When serialized sinks, command results, notification rows, operational storage and public writes are inspected through disconnect and deletion
    Then protected values appear only at intended authenticated destinations or permitted local draft storage
    And no chat-history archive or AppView draft store exists and approved bounded diagnostic fields remain
    And documented cleanup preserves necessary request/uncertain-send metadata across disconnect, hides retained drafts, and removes CraftSky private state on deletion without deleting upstream chat or unrelated namespaces
```

### AT-021: Bounded fixed API surface
Requirement IDs: NFR-003  
Acceptance Criteria: AC-021  
Priority: Must  
Level: Acceptance  
Automation Target: IT-019: Go `appview/internal/api/chat_contract_test.go` owns the registered-route matrix, authentication, bounded pagination and malicious requests; Flutter `app/test/chat/chat_api_client_test.dart` owns consumer wire mapping and error/cursor handling.

```gherkin
Feature: Direct messaging
  Scenario: Bounded fixed API surface
    Given the final implemented consumer chat routes and valid, invalid and unauthorized requests
    When requests include tampered identifiers/cursors and arbitrary RPC names, audiences, URLs or raw upstream bodies
    Then routes follow /v1/, bearer/device auth, camelCase JSON, standard error/message/requestId envelopes and opaque bounded pagination
    And tampering cannot change owner/service or turn the API into a generic proxy
```

### AT-022: Shared bounded polling
Requirement IDs: NFR-004, NFR-005  
Acceptance Criteria: AC-022  
Priority: Must  
Level: Acceptance  
Automation Target: UT-006/IT-020: Go `appview/internal/chat/scheduler_test.go` and `appview/internal/chat/scheduler_integration_test.go` own fake-clock cadence, account-sharing, concurrency/backoff/bounds/fairness and background eligibility. Flutter `app/test/chat/sync_state_test.dart` verifies active-view lifecycle signals only; no message-arrival benchmark or duplicate client poller is required.

```gherkin
Feature: Direct messaging
  Scenario: Shared bounded polling
    Given one authorized owner with multiple active views/devices, another healthy owner and a controllable clock
    When conversations open or close and healthy, slow, rate-limited and revoked outcomes occur
    Then healthy completed foreground polls use the configured four-second interval shared per account
    And in-flight polls do not overlap, backoff/rate limits and lifecycle fencing take precedence
    And concurrency, call duration, payloads and retries are bounded, healthy accounts progress and eligible background polling continues with no foreground view
```

### AT-023: Supported platform usability
Requirement IDs: NFR-006  
Acceptance Criteria: AC-023  
Priority: Must  
Level: Acceptance  
Automation Target: UT-010: Flutter `app/test/chat/conversation_page_test.dart` and `app/test/chat/chat_localizations_test.dart` own layout, labels, semantics and keyboard/focus assertions for the chat adapter; Flutter `app/test/chat/messaging_flows_accessibility_test.dart` extends those assertions across enablement, inbox/requests, profile entry, reporting, preferences and recovery. Extend existing `app/integration_test/critical_journeys_test.dart` for mobile process/plugin journeys. MAN-002 owns real iOS/Android push opens; MAN-003 owns actual platform/assistive-technology and web foreground checks. No backend acceptance file can establish platform usability.

```gherkin
Feature: Direct messaging
  Scenario: Supported platform usability
    Given iOS, Android and web with narrow/wide layouts and supported localized text
    When the owner uses enablement, inbox, requests, profile entry, conversation, reporting, preferences and recovery
    Then labels, screen-reader focus, keyboard interaction, loading/error states and layout remain usable
    And iOS/Android support actual push opens while web refreshes in foreground without a web-push promise
```

### AT-024: Actual integrated provider behavior
Requirement IDs: NFR-007  
Acceptance Criteria: AC-024  
Priority: Must  
Level: Acceptance  
Automation Target: MAN-001: controlled consenting-account checks through actual CraftSky/AppView/PDS/chat/moderation own the complete live-service assertions. Related IT cases supply deterministic contract/fault coverage only and cannot satisfy this scenario by mocks.

```gherkin
Feature: Direct messaging
  Scenario: Actual integrated provider behavior
    Given controlled consenting accounts and a completed CraftSky/AppView integration
    When the owner exercises granular consent/refresh, eligibility, PDS routing, status/request/count, messages, requests, read, mute, leave/restart, declaration updates and closed-app log recovery
    Then observable authoritative results satisfy the linked functional criteria and material deployed-contract differences are recorded and resolved
    And no broad-scope fallback or safety/control omission is used
```

### AT-025: First supported CraftSky preview only
Requirement IDs: FR-017  
Acceptance Criteria: AC-025  
Priority: Must  
Level: Acceptance  
Automation Target: UT-008/IT-016: Flutter `app/test/chat/link_preview_policy_test.dart`, `app/test/chat/link_preview_flow_test.dart` and `app/test/router/chat_content_routes_test.dart` own candidate/fetch counts, access-bound resolver responses, navigation and account-switch races. UT-010 owns card presentation; IT-018 checks serialized privacy sinks; MAN-003 complements actual platform card interaction.

```gherkin
Feature: Direct messaging
  Scenario: First supported CraftSky preview only
    Given accepted messages with post, focused-reply, profile, event, unsupported and multiple links, plus composer and unaccepted-request text
    When the first supported target resolves, fails, is restricted/removed or completes after an account switch
    Then at most that first supported target produces a permitted compact card and all text URLs remain usable
    And an inaccessible first target causes no later-target lookup even when later content is accessible
    And reply cards show the reply and open its focused thread
    And composer and unaccepted requests perform no preview resolution, acceptance enables it, and stale-account results are discarded
    And no autoplay/nested controls, arbitrary fetches, third-party text disclosure or preview data in push/diagnostics occurs
```

### AT-026: Selected chat UI package fit
Requirement IDs: NFR-008  
Acceptance Criteria: AC-026  
Priority: Must  
Level: Acceptance  
Automation Target: MAN-004: coding-plan package-fit record owns version/license/SDK/dependency review and actual iOS/Android/web execution; UT-010 Flutter `app/test/chat/conversation_page_test.dart` owns adapter builders, styling, pagination/send/draft states and preview-fetch suppression. MAN-003 owns assistive-technology evidence; IT-015/IT-018 own server transport/credential-boundary assertions.

```gherkin
Feature: Direct messaging
  Scenario: Selected chat UI package fit
    Given coding planning for the selected flutter_chat_ui package
    When its license, SDK/dependencies and iOS/Android/web fit are evaluated alongside rendering tests
    Then planning records styling/accessibility, pagination, send-state, draft and custom-card fit with composer/request preview fetching disabled
    And AppView transport/credential ownership remains intact and concrete incompatibilities return for review before substitution
    And dependency-resolution evidence is not represented as compilation or device proof
```

### AT-027: New message and profile entry
Requirement IDs: FR-018  
Acceptance Criteria: AC-027  
Priority: Must  
Level: Acceptance  
Automation Target: IT-003: Go `appview/internal/chat/start_test.go` owns availability/canonical start rules; Flutter `app/test/chat/new_message_page_test.dart` owns inbox/profile entry, CraftSky search and explicit selection without global lookup.

```gherkin
Feature: Direct messaging
  Scenario: New message and profile entry
    Given the inbox, CraftSky profile search, an existing external conversation and available or blocked/unavailable recipients
    When New message or a profile Message action is used
    Then CraftSky profile search offers availability-aware selection and only explicit start opens/creates the canonical direct conversation
    And blocked/unavailable sends are prevented without global Bluesky search or a membership requirement for existing conversations
```

### AT-028: Private local drafts without offline replay
Requirement IDs: FR-005, FR-016, FR-019, NFR-002  
Acceptance Criteria: AC-028  
Priority: Must  
Level: Acceptance  
Automation Target: UT-009/IT-017: Flutter `app/test/chat/draft_controller_test.dart` and `app/test/chat/draft_storage_test.dart` own per-account/conversation persistence, hiding, clearing and no automatic replay; extend existing `app/integration_test/critical_journeys_test.dart` for real plugin/process restart/logout. IT-018 owns sink/network privacy assertions; explicit send uses IT-006.

```gherkin
Feature: Direct messaging
  Scenario: Private local drafts without offline replay
    Given different drafts for accounts A and B and multiple conversations
    When the owner navigates, restarts, tries offline Send, reconnects, disconnects/re-enables, confirms send, discards or logs out A
    Then only matching account/conversation drafts restore and reconnect/restart never submits automatically
    And offline Send retains text with connection feedback
    And disconnect explains and hides retained drafts until re-enabled, confirmed send/discard clears them, and A logout removes only A drafts
    And drafts stay in private local storage without AppView synchronization, telemetry or push; explicit Send uses authenticated messaging
```

### AT-029: Request lifetime and coalesced catch-up
Requirement IDs: FR-013  
Acceptance Criteria: AC-029  
Priority: Must  
Level: Acceptance  
Automation Target: IT-012/IT-010/IT-011: Go `appview/internal/chat/notification_policy_integration_test.go`, `appview/internal/chat/sync_crash_test.go` and `appview/internal/push/chat_dispatcher_test.go` own lifetime state, crash/replay/recovery and current eligibility. MAN-001 validates service-dependent acceptance/new-request transitions; no device reinstall is allowed to reset server eligibility.

```gherkin
Feature: Direct messaging
  Scenario: Request lifetime and coalesced catch-up
    Given pre-enable history, genuinely new requests and accepted conversations with a recovery interval
    When requests receive more messages, leave/reappear, reinstall, disconnect/reconnect or replay, and an outage/reauthentication recovers
    Then first enablement alerts on no old history or requests
    And a new eligible unread request alerts once while unaccepted and only verified acceptance followed by a genuine new request transition permits another alert
    And recovery creates at most one catch-up per eligible unread accepted conversation/recovery, without individual missed-message replay or re-alert of already-notified state
    And current dispatch checks apply and request eligibility survives device reinstall and reconnect
```

## 4. Unit Test Cases

Targets are proposed unless explicitly identified as existing. Parameterized rows require each listed variant, not one representative happy path.

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, RULE-004, NFR-001 | AC-002, AC-019 | Semantic capability and eligibility evaluation | Base-only, partial, complete, expired/revoked grants; combined lxm, reordered and encoded parameters; wrong audiences, wildcard/unrelated permissions; chatDisabled and generic service failures | Exact needed method/audience authority recognized semantically; optional metadata never becomes base requirement; missing report/declaration permissions gate their operations; restricted status has no invented reason; no broad authority requested | `appview/internal/auth/effective_scopes_test.go`, `appview/internal/chat/capability_test.go` |
| UT-002 | FR-005 | AC-006 | Independent Dart/Go text validation | Empty, whitespace-only, 999/1000/1001 graphemes; 9999/10000/10001 UTF-8 bytes with grapheme counts within limit; ZWJ emoji and combining sequences; over-byte under-grapheme and converse fixtures | Both layers agree; valid text byte-for-byte unchanged; invalid text makes zero send calls and stays editable | `app/test/chat/message_validation_test.dart`, `appview/internal/chat/validation_test.go` |
| UT-003 | FR-003 | AC-004 | History mapping/merge and view anchor | Overlapping/reordered pages, same text with different IDs, deleted tombstones, unsupported union variants and valid cursor on empty page | Stable chronological IDs; no text-based dedupe; deletion replaces content safely; anchor survives older-page insertion; only absence of cursor ends paging | `app/test/chat/history_controller_test.dart` |
| UT-004 | FR-007, RULE-002, RULE-005 | AC-008 | Read watermark and native badge mapping | Presented M, unseen M+1, background view; accepted/request counts 0/1/99/100; contradictory list totals; mute/profile-mute changes | Read names last presented message only; badges use upstream buckets and exclusions; 100 becomes 99+ and refreshes; no list recomputation or exact decrement; mute does not mark read | `app/test/chat/unread_state_test.dart`, `appview/internal/chat/read_test.go` |
| UT-005 | FR-010 | AC-010 | Report validation and evidence construction | Supported/unsupported reasons, explanation 1999/2000/2001 graphemes and 19999/20000/20001 bytes, sender differing from reporter; forged references | Bounds enforced without truncation; authorized message view determines messageRef including sender; invalid/inaccessible evidence rejected; fixed moderation audience distinct from chat; no copied transcript | `appview/internal/chat/report_test.go`, `app/test/chat/report_controller_test.dart` |
| UT-006 | FR-001, FR-015, NFR-004, NFR-005 | AC-017, AC-022 | Recovery classification and deterministic scheduling | Fake clock, successful polls, in-flight barrier, 429/retry guidance, 5xx/timeouts, multiple subscribers/owners, foreground closure and revocation | Four-second healthy foreground configuration; one shared account poll; no overlap; finite configured concurrency/timeouts/payload/retry budgets and backoff; background eligibility retained; healthy owners progress; stale owner results discarded | `appview/internal/chat/scheduler_test.go`, `app/test/chat/sync_state_test.dart` |
| UT-007 | FR-013, RULE-003 | AC-015 | Push identity and payload construction | Display name, handle-only, missing identity, renamed sender; message/report/preview canaries | Name then handle then generic fallback; only generic message copy alongside allowed identity; every serialized provider field excludes private canaries, including missing-name paths | `appview/internal/push/chat_payload_test.go` |
| UT-008 | FR-017 | AC-025 | Preview candidate and presentation policy | Unsupported URL before supported URL; post/reply/profile/event routes; inaccessible first supported target plus accessible later target; malformed/host-lookalike URLs; composer/request states | Only first recognized supported candidate attempted once; no fallback; correct reply identity/focus; no fetch for composer or unaccepted request; no arbitrary external fetch; safe text remains | `app/test/chat/link_preview_policy_test.dart` |
| UT-009 | FR-005, FR-016, FR-019, NFR-002 | AC-028 | Draft reducer and send transitions | Account/conversation keys; pending, confirmed, failed and uncertain send results; offline send, reconnect/restart, discard, disable/re-enable and logout | Private drafts isolated, retained on failure/uncertainty and offline; clear only on confirmed send/discard or relevant account logout; hidden on disable; no offline auto-submit or draft-sync call | `app/test/chat/draft_controller_test.dart` |
| UT-010 | NFR-006, NFR-008 | AC-023, AC-026 | Conversation adapter and messaging-flow rendering, semantics and package feature controls | Enablement, inbox/requests, profile entry, reporting, preferences and recovery; flutter_chat_ui with narrow/wide constraints, long localized labels, large text, text scaling, keyboard/focus traversal, older-page insertion, pending/failed/uncertain states and custom cards | CraftSky styling and accessible semantics; usable focus/layout; stable pagination; accurate states/drafts; no package-default composer/request preview network activity or client-owned OAuth transport | `app/test/chat/conversation_page_test.dart`, `app/test/chat/chat_localizations_test.dart`, `app/test/chat/messaging_flows_accessibility_test.dart` |
| UT-011 | BR-003 | AC-002, AC-010 | Consent and reporting copy semantics | Rendered enablement and accompanying messaging explanations plus report confirmation in supported locales; recording authorization/report triggers | Shared history/settings, Bluesky hosting and Bluesky-only reporting appear before authority is requested; no promise of CraftSky-only privacy/inbox, PDS message portability or end-to-end encryption; accurate explicit denials are allowed; report confirmation names Bluesky/evidence scope before submission without implying CraftSky moderator delivery | `app/test/chat/messaging_consent_test.dart`, `app/test/chat/report_page_test.dart` |

## 5. Integration Test Cases

Use real production-schema PostgreSQL fixtures for state/constraints and deterministic upstream servers for call counts, bodies, headers and failures. Keep test synchronization explicit with barriers; avoid wall-clock sleeps. Auth transport tests must include the entire configured retry stack, including SDK/auth challenge handling. Assert ordinary network ambiguity never causes send/report replay; genuine explicit nonce/expiry challenges must be distinguished from ambiguous acceptance, not used as a blanket replay exemption.

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-002, FR-008 | AC-001, AC-003 | Direct inbox/request adapter | TD-001, TD-002; upstream recorder with multiple opaque pages and mixed request unions | Enumerate all pages through UI repository and AppView adapter | listConvos uses kind=direct/status=accepted; dedicated listConvoRequests used; cursors survive filtered-empty pages; groups excluded from views/push and count calls exclude groups; external identities need no CraftSky profile | `appview/internal/chat/inbox_test.go`, `app/test/chat/inbox_page_test.dart` |
| IT-002 | FR-003 | AC-004 | History pagination and deletion | TD-002 with overlapping IDs, unsupported variants and tombstones | Load older pages through actual client response mapping and scroll widget | Authoritative text ordered once; no raw payload leakage; placeholders safe; anchor stable; cursor, not item count, controls completion | `appview/internal/chat/history_test.go`, `app/test/chat/history_controller_test.dart` |
| IT-003 | BR-001, FR-004, FR-018 | AC-005, AC-027 | Entry points, availability and canonical start | TD-001; recipient availability matrix and recording create/send adapter | New message search, profile availability, explicit/repeated start, malicious self/multiple DID input and blocked/unavailable selections | Search is CraftSky-only; availability performs zero creates; explicit allowed start gets canonical direct convo for exactly one peer; denied paths perform zero unintended writes/sends | `appview/internal/chat/start_test.go`, `app/test/chat/new_message_page_test.dart` |
| IT-004 | BR-003, FR-010 | AC-010 | Real report adapter boundary under faults | TD-003, TD-006; authorized/inaccessible message views; PDS and moderation proxy recorder | Submit valid/invalid report, timeout after acceptance and failure before acceptance; retry same unresolved action | Correct fixed moderation proxy audience and messageRef sender; limits enforced; no transcript or moderator API; accepted/failed/uncertain UI accurate; complete retry stack does not blindly duplicate uncertain report; no CraftSky moderation queue/evidence | `appview/internal/chat/report_integration_test.go`, `app/test/chat/report_page_test.dart` |
| IT-005 | BR-002, FR-001, FR-013, FR-016 | AC-002, AC-014 | Multi-session enablement, expiry and lifecycle fences | TD-001, TD-004; two devices with different permissions; worker blocked before admission/commit/dispatch | Cancel consent, enable/disconnect account-wide, logout one device, expire/revoke last session, sign-out-all and begin deletion | Ordinary access unaffected where authorized; default inactivity expiry unchanged by polling; correct session selected; one logout preserves other usable sessions; account loss rejects fresh work and stale commit/enqueue/dispatch; UI explains pause; hidden drafts delegated to IT-017 | `appview/internal/auth/chat_lifecycle_test.go`, `appview/internal/chat/lifecycle_integration_test.go` |
| IT-006 | FR-005, FR-006 | AC-006, AC-007 | Durable send operation and all crash windows | TD-003; owner-scoped stable operation identity, real DB; concurrent clients and upstream recorder | Double-submit; crash before claim, after durable claim before request, after possible acceptance before result persistence, and after confirmed persistence before client response; reconnect/restart | Same operation never adds upstream send; pre-request uncertain crash can conservatively remain unresolved; definitive no-send failure distinguishable; possible acceptance stays uncertain; history refresh never auto-replays and identical text alone never confirms; explicit warned Send again uses new identity; peer owner cannot reuse another owner's operation | `appview/internal/chat/send_integration_test.go`, `app/test/chat/send_flow_test.dart` |
| IT-007 | FR-007, FR-008, FR-009, RULE-002 | AC-003, AC-008, AC-009 | Read/accept/mute/leave authoritative round trips | TD-002; newer arrival after displayed watermark, remote reads/settings and failures; accept response without rev | Read request, accept, mute/unmute, cancel/confirm leave, refresh remote changes | updateRead includes exact presented ID; no background read or silent acceptance; accept-without-rev succeeds when authoritative state confirms; counts refresh from native direct aggregates; capped counts not decremented exactly; no failed action falsely succeeds; leave cancels pending push without peer deletion | `appview/internal/chat/actions_test.go`, `app/test/chat/conversation_actions_test.dart` |
| IT-008 | FR-011, RULE-002 | AC-011 | Conditional declaration merge | TD-005; missing/existing self record, swap-CID conflict and unrelated field added concurrently | Enable/disable, explicitly change allowIncoming, force conflict, read back and refresh external change | No lifecycle-triggered declaration write; absent record uses absence precondition; update conditional on latest record; refetch/merge preserves unknown fields and allowGroupInvites; all/following/none mapped correctly; failure not success; UI distinguishes persisted value from eventual service propagation | `appview/internal/chat/declaration_integration_test.go`, `app/test/chat/preferences_page_test.dart` |
| IT-009 | FR-007, FR-009, FR-012, FR-015 | AC-012, AC-017 | Cursor and forward-compatible event decoding | TD-007; saved cursor, unknown raw union envelopes/revisions, group and deleted events | Repeat/reorder pages, transient getLog failure, invalid cursor, refresh/resume/reconnect | Normal failure retains valid saved cursor, never fresh-head resets it; unknown/irrelevant events advance safely after required processing; invalid cursor reconciles authoritative state under request/catch-up policies; deleted text not resurrected; stale content remains only under correct owner | `appview/internal/chat/sync_integration_test.go`, `app/test/chat/sync_controller_test.dart` |
| IT-010 | FR-012, FR-013 | AC-013, AC-029 | Atomic progress, dedupe and outbox | TD-007; migrated schema, competing workers and crash barriers | Crash before processing, after metadata/outbox writes before commit, after commit before acknowledgement and after delivery claim; replay account/message | Progress and required work commit together or roll back; uniqueness gives one logical notification and at most one job/subscription; other accounts with same upstream IDs remain independent; initial baseline has zero push; delivery leases/retries do not create duplicate logical work; provider redelivery remains possible | `appview/internal/chat/sync_crash_test.go`, `appview/internal/push/chat_dispatcher_test.go` |
| IT-011 | FR-013, FR-014, RULE-001, RULE-003, RULE-005 | AC-015, AC-018 | Dispatch-time eligibility and payload privacy | TD-004, TD-006; queued private notification, real subscription/preferences and recording sender | Individually change read, mute, either-direction block, leave, push toggle, lifecycle, OS/device eligibility and subscription before final dispatch; test owner-sent/group events and profile mute alone | Current checks suppress each ineligible delivery; profile mute alone allows eligible DM; owner/group never notify; default toggle is shown/enabled account-wide; per-device permissions respected; no Bluesky preference write; serialized payload canaries absent; allowed sender/generic fields present | `appview/internal/push/chat_dispatcher_test.go`, `app/test/chat/push_preferences_test.dart` |
| IT-012 | FR-013 | AC-013, AC-029 | Lifetime request and recovery state machine | TD-008; persisted request-alert history plus accepted messages, recovery identity and notifications already sent | Initial enable, repeated request messages, leave/reappearance, reinstall, disconnect/reconnect, replay, acceptance then genuine new request, repeated recovery and read/mute/block changes | Baseline silent; one alert per eligible new request until verified acceptance and genuine new transition; device/lifecycle churn never resets eligibility; at most one eligible accepted-conversation catch-up/recovery, no per-message replay or already-notified re-alert; request policy stays separate and current dispatch checks win | `appview/internal/chat/notification_policy_integration_test.go` |
| IT-013 | FR-014, FR-016, NFR-001 | AC-016, AC-019 | Owner/membership enforcement and late response fencing | TD-001, TD-004; A/B retained sessions, opaque push routing, delayed history/preview/send/navigation results | Cold/foreground open, logout, A-to-B switch, reauth lease replacement, disconnect, membership loss; try every route with forged IDs | Authenticate/activate exact permitted owner before fetch/render; stale results cannot paint private UI or navigate; safe fallback when access missing; all reads, sends, read updates, accept/mute/leave/settings/report and operation lookups owner/capability fenced | `appview/internal/chat/authorization_test.go`, `app/test/router/chat_notification_open_routing_test.dart`, mobile critical journey extension |
| IT-014 | RULE-001 | AC-018 | Indexed/newly confirmed block races | TD-004; each block direction, stale availability and accessible/denied history | Block between availability and send/start or before dispatch, report history, then unblock | Server blocks writes and push; UI disables send; permitted history/report survives; denied history stays unavailable without bypass; unblock requires new availability; final-network-check race recorded honestly | `appview/internal/chat/block_integration_test.go` |
| IT-015 | RULE-004, NFR-001 | AC-019 | Optional OAuth flow and routed authority | TD-009; OAuth/PDS recorder and sessions with disjoint effective grants | Advertise/request optional permissions through PAR/callback, refresh base/chat sessions, select worker session and perform each implemented chat/report/declaration operation | Base session remains valid; operation-level required grants checked semantically; no borrowing another owner/session authority; fixed PDS proxy chat/moderation routing and create/update-only declaration authority; no generic proxy, broad include, group/export/account-delete grant or client credential handoff | `appview/internal/auth/chat_oauth_integration_test.go`, `appview/internal/chat/transport_test.go` |
| IT-016 | FR-017 | AC-025 | Preview access, fetch counts and navigation races | TD-010; actual authenticated content client with recording resolver and delayed results | Resolve each target type, first-target failures followed by accessible later URLs, request-to-accepted transition and account switch | At most first supported target lookup; no later fallback or composer/request lookup; existing access/moderation applied; reply previews/focus correct; every URL intact; bounded card details/thumbnail, no nested UI/playback; no arbitrary/third-party fetch; switched-owner response ignored and canaries absent from sinks/push | `app/test/chat/link_preview_flow_test.dart`, `app/test/router/chat_content_routes_test.dart` |
| IT-017 | FR-005, FR-016, FR-019, NFR-002 | AC-028 | Real local draft storage and process lifecycle | TD-011; production draft storage adapter in temporary/private app storage; A/B conversations | Save, navigate, recreate controller/process, offline send, reconnect, disable/re-enable, send/clear/discard, logout A | Same-account/conversation restoration; no network until explicit send; disable retains hidden with explanation; send confirmation/discard clears; logout removes only A; no storage/telemetry/AppView draft transfer; stale async save cannot restore A after logout | `app/test/chat/draft_storage_test.dart`, mobile critical journey extension |
| IT-018 | NFR-002 | AC-020, AC-028 | Serialized sink/storage and cleanup inspection | TD-006; production Go/Flutter diagnostic serializers, recording public/PDS writes, DB and local storage | Success/error/timeout paths for auth, send, sync, report, preview, push and drafts; disconnect/deletion cleanup | Canaries absent from logs/errors/analytics/command results/social notifications/public writes, with positive approved fields present; no history archive/AppView drafts/report evidence; private routing only in necessary owner-scoped metadata/opaque push routing; retention plan preserves dedupe/uncertain-send safety across disconnect; deletion removes CraftSky private state only | `appview/internal/chat/privacy_test.go`, `app/test/observability/chat_privacy_test.dart` |
| IT-019 | NFR-003 | AC-021 | Registered-route contract and malicious inputs | TD-009; table enumerating final routes, methods, auth requirements and allowed operations | Valid/invalid/unauthorized requests; missing/wrong bearer/device headers; malformed/oversized input, excessive page limits, cursor/identifier tampering, arbitrary upstream controls | Existing API casing/error envelope/status conventions; bounded opaque pagination; fixed route/operation/audience; no cross-owner or arbitrary network target; API contract tests updated to approved final routes rather than assuming the candidate count | `appview/internal/api/chat_contract_test.go`, `app/test/chat/chat_api_client_test.dart` |
| IT-020 | NFR-004, NFR-005 | AC-017, AC-022 | Fair bounded shared scheduler under faults | TD-007; fake time, configured budgets, two owners, multiple views/devices, upstream concurrency recorder | Open/close views, drive four-second intervals, hold slow calls, return 429/5xx/oversized payloads, lose authorization | Shared healthy foreground cadence; no concurrent account poll; configured timeouts/payload/retry/concurrency bounds enforced; rate-limit backoff takes precedence; healthy account not starved; authorized background work continues and stops on lifecycle loss; no arrival-latency benchmark | `appview/internal/chat/scheduler_integration_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Ordinary OAuth/session access, expiry and device-scoped logout | BR-002, FR-016, RULE-004 | AC-002, AC-014, AC-019 | Run existing `appview/internal/auth` suites alongside optional-grant cases; base-only login/refresh and non-chat APIs remain usable, session expiry is not extended by polling and one logout leaves other devices active. |
| REG-002 | Social notification categories, audience/profile-mute filters and durable leases | FR-013, RULE-005 | AC-013, AC-015 | Existing `appview/internal/push` and Flutter notification suites continue; adding DM policy must preserve social suppression and retry behavior while proving profile mute alone does not suppress DM. |
| REG-003 | Account-safe existing notification destinations | FR-014, NFR-001 | AC-016 | Existing `app/test/notifications/providers/notification_open_coordinator_test.dart`, `app/test/router/notification_open_routing_test.dart` and account-switch routing tests still route existing destinations under correct owner leases. |
| REG-004 | Existing content link navigation and access controls | FR-017 | AC-025 | Run existing shared-link/preview, post/profile/event router and observability preview suites; chat reply/profile/event cards preserve canonical routes and current-viewer restrictions without broadening composer preview behavior. |
| REG-005 | Existing diagnostic sanitization and issue ownership | NFR-002 | AC-020 | Existing Go observability and Flutter `app/test/observability` suites plus chat sink canaries; typed causes/approved bounded context retained without a second issue owner or private upstream prose. |
| REG-006 | Owner deletion namespace boundaries | FR-016, NFR-002 | AC-014, AC-020 | Existing account-deletion suites plus IT-005/IT-018: DM cleanup removes CraftSky private state, fences work, and adds no chat-account/message/blob/declaration or unrelated-namespace deletion. |

## 7. Test Data

All data is synthetic except controlled consenting accounts in MAN-001. Never put real credentials, message/report content or private identifiers into committed fixtures or test artifacts.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Interoperability and owner isolation | Owners A/B; A devices A1/A2; peer C absent from CraftSky; known CraftSky peer D; distinct and colliding conversation/message/operation IDs scoped by owner; missing/changed sender names | AT-001, AT-005, AT-014, AT-016, AT-027; IT-001, IT-003, IT-005, IT-013 |
| TD-002 | Pagination and native counts | Accepted/request/group unions, duplicate/empty/short pages with valid opaque cursors, messages in reverse upstream order, deleted/unknown payloads, counts 0/1/99/100, contradictory list sums; muted/locked/blocked/deactivated provider exclusions | AT-001, AT-003, AT-004, AT-008, AT-009; UT-003, UT-004; IT-001, IT-002, IT-007 |
| TD-003 | Unicode, reporting and ambiguous writes | Programmatically generated boundary strings for UT-002/UT-005; identical text with distinct message IDs; before-acceptance failure, after-acceptance timeout and delayed response scripts; stable/new operation keys | AT-006, AT-007, AT-010; UT-002, UT-005; IT-004, IT-006 |
| TD-004 | Dispatch/session race matrix | Active/expired/revoked/disjoint-grant sessions, owner epochs and membership changes; both block directions, profile/conversation mutes, read watermark, left/request states, preferences, OS permissions and subscriptions | AT-014–AT-018; IT-005, IT-011, IT-013, IT-014 |
| TD-005 | Conditional shared policy | Missing record; all/following/none; allowGroupInvites and unknown nested fields; record CID changes and concurrent writer | AT-011; IT-008 |
| TD-006 | Privacy canaries with positive controls | Unique body/draft/report/token/DPoP/private-routing/cursor/raw-error/preview markers, distinct allowed outcome/category markers; production serializers and recording sender/public-write sinks | AT-010, AT-015, AT-020, AT-025, AT-028; UT-007; IT-004, IT-011, IT-018 |
| TD-007 | Durable sync and scheduler faults | First baseline, saved/expired cursors, raw unknown events with revisions, repeated/reordered pages, tombstones and groups; transaction/worker barriers and fake clock; rate-limit/timeout/oversized responses | AT-012, AT-013, AT-017, AT-022; UT-006; IT-009, IT-010, IT-020 |
| TD-008 | Request lifetimes and catch-up | Pre-enable request, new request R1, repeated messages, leave/reappearance, verified acceptance, genuinely new R2; accepted unread conversations with already-notified and unseen states across distinct recovery intervals | AT-013, AT-029; IT-010, IT-012 |
| TD-009 | Permission and API contracts | Pinned required-method/audience matrix from requirements section 24 as fixture seed, normalized equivalent scopes, wrong/missing/wildcard grants; final approved route table; tampered cursors/IDs and prohibited RPC/URL/body controls | AT-019, AT-021; UT-001; IT-015, IT-019 |
| TD-010 | First-link-only previews | Canonical post, focused reply, profile and event; unsupported and host-lookalike links; denied/deleted/timeout first target followed by accessible target; delayed A response after B activation; request-to-accepted transition | AT-025; UT-008; IT-016 |
| TD-011 | Drafts, platforms and chat adapter | Rendered enablement/reporting copy in supported locales; A/B drafts for distinct conversations; fresh/restarted production storage; narrow/wide, long localized and large-text fixtures; selected package builder configuration and message states | AT-023, AT-026, AT-028; UT-009, UT-010, UT-011; IT-017; MAN-002–MAN-004 |

## 8. Manual Checks

These complement automation at external/device boundaries; they are not substitutes for deterministic fault tests. Live actions require controlled consenting accounts and suitable environment authorization at execution time. This document performs none.

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-007, RULE-004, FR-001, FR-002, FR-004, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013 | AC-024, AC-001, AC-002, AC-005, AC-008–AC-015, AC-019, AC-029 | Actual CraftSky/AppView/PDS/chat/moderation integration during full implementation | With consenting test accounts, grant granular authority through CraftSky and refresh; check base-only/partial/restricted cases where available. Exchange direct/request messages including a non-CraftSky peer; verify status/list/count/read/mute/accept. Change/read back declaration with unrelated fields; compare another client. Confirm leave effects and later restart/history/request behavior. Close app with valid session, send peer messages, interrupt/resume worker and exercise service cursor recovery when available. Submit only an authorized controlled moderation test report through the permitted destination/procedure; do not send a fabricated abusive report to production. Record safe outcome evidence. | Authoritative peer/client results establish behavior through actual integration, including fixed chat versus moderation proxy audiences, optional grants, shared-state propagation, background detection and request/catch-up semantics. No broad-scope fallback. Material differences documented and resolved against related criteria; an unexercised report/cursor/restriction case remains explicitly unverified. No separate spike or latency/capacity test. |
| MAN-002 | NFR-006, FR-014, FR-016, NFR-001 | AC-023, AC-015, AC-016 | APNs/FCM and real OS taps on iOS and Android | On signed configured test builds, check default toggle and denied/granted OS permission; receive controlled sender-name-only push with app closed/backgrounded and foregrounded; open under correct account, B active, A logged out, disconnected and membership lost. Inspect full provider payload securely without retaining private identifiers in evidence. | Delivery follows account/device eligibility; no private content in any payload field; tap opens only authorized A content or safe fallback; no B disclosure. Runtime injection alone does not satisfy this check. |
| MAN-003 | NFR-006, NFR-008, FR-017 | AC-023, AC-025, AC-026 | Assistive technology and actual platform interaction | On iOS VoiceOver, Android TalkBack and a web browser with keyboard/screen reader, traverse enablement, inbox/requests, profile entry, history, composer/states, report and settings; test narrow/wide, large text and supported localization. On web resume/reconnect check foreground updates. Tap reply/event/profile cards. | Meaningful labels/focus order and usable keyboard/scroll behavior; controls and cards fit; focused reply opens correctly; no autoplay/nested controls; web makes no background-push promise. Widget semantics tests remain required. |
| MAN-004 | NFR-008, NFR-006 | AC-026, AC-023 | Planning record and real package/platform fit | In coding planning, record selected flutter_chat_ui version/license/SDK/dependency fit and review custom builders, request/composer preview suppression, draft and send states, pagination and accessibility. During implementation build/run iOS, Android and web with those adapters. | Concrete compatibility evidence with no silent package replacement. Solver success is labeled resolution-only; missing platform execution remains open rather than reported as validated. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | All DM tests and proposed targets await implementation | All requirements in section 2 | AC-001–AC-029 | Current repository has no DM implementation/test suite; this is a design artifact | Confirm target paths and test seams in coding planning; implement the tests and retain actual results. No test pass is claimed here. |
| GAP-002 | Deployed granular authority, eligibility, reporting, leave/restart and cursor contracts unproven | NFR-007, RULE-004, FR-001, FR-009, FR-010, FR-012, FR-013 | AC-024, AC-002, AC-009, AC-010, AC-012, AC-019, AC-029 | Source audit/mocks cannot prove deployed service behavior; some cases require a consenting restricted account, an expired cursor or an approved report path | Execute MAN-001 during full integration; record unavailable cases and resolve material differences before claiming functional acceptance. This does not block coding planning or create a separate spike. |
| GAP-003 | Existing device harness injects notification opens and loopback AppView | NFR-006, NFR-007, FR-014, NFR-001 | AC-023, AC-024, AC-016 | It does not exercise external OAuth, actual AppView/PDS messages or APNs/FCM OS taps; mobile-only harness uses dart:io | Extend deterministic mobile journeys where useful; use MAN-001/MAN-002 for actual external behavior and MAN-003 for web/assistive technology. |
| GAP-004 | Retention periods, operation/recovery identity schema and local draft storage need design | FR-006, FR-012, FR-013, FR-019, NFR-002 | AC-007, AC-013, AC-020, AC-028, AC-029 | Requirements intentionally defer representation and periods | Coding plan must define boundaries/cleanup compatible with lifetime request dedupe and uncertain sends. IT-006/IT-010/IT-012/IT-017/IT-018 then verify the chosen design, including cleanup/re-enable. No arbitrary retention value added here. |
| GAP-005 | In-flight provider races cannot be retracted; transport delivery may repeat | FR-013, FR-016, RULE-001 | AC-013, AC-014, AC-015, AC-018 | State can change after last eligibility check; provider redelivery is external | Barrier tests establish rejection before admission/final dispatch and stale commit fencing. Document residual already-issued request race; do not claim exactly-once transport or retroactive cancellation. |
| GAP-006 | Package resolution is not platform/accessibility evidence | NFR-006, NFR-008 | AC-023, AC-026 | Only isolated dependency resolution was established during requirements | Complete planning record and UT-010/MAN-003/MAN-004; return concrete blockers before substitution. |
| GAP-007 | Scheduler resource budgets and final route inventory not yet specified | NFR-003, NFR-004, NFR-005 | AC-021, AC-022 | Requirements fix cadence and boundedness, not every numeric budget or API shape | Coding plan records finite limits/timeouts/retries and final route list. IT-019/IT-020 test configured boundaries; do not invent arrival-time or capacity targets. |

No unanswered product question blocks this test design. The feature retains **High** risk. Explicit owner approval is required before implementation continues under the write-acceptance-tests skill; document review is recommended next. Unrun live/device checks are acceptance evidence still to obtain during implementation, not prerequisites to writing the coding plan.

## 10. Out Of Scope

- Group UI/management/notifications; attachments, media sends, upstream record embeds, reactions, threads, editing, deletion/export and chat-account deletion. Unsupported incoming variants still require safe handling.
- Generic chat proxy, Full Chat Client or wildcard authority, alternate transport/storage backend, public DM records or Tap indexing, private transcript archives and CraftSky DM moderation workflows.
- Offline history/inbox, automatic offline sends, cross-device drafts, typing/presence and peer read receipts. Private local draft persistence is included.
- Global Bluesky recipient search, changing Bluesky notification preferences, cross-app push deduplication, web background push and native desktop targets.
- Composer/unaccepted-request preview fetches, arbitrary external scraping, later-link fallback, inline playback and interactive nested post cards.
- Separate feasibility spike, pre-design live-service gate, message-arrival latency target/measurement and performance/capacity benchmarks. Four-second polling configuration, fairness, backoff and bounded resource checks remain included.
- Deployment, migrations, dependency installation, source/test-code edits and live actions in this documentation stage.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md` (AC-002/AC-010 clarified in this revision).
- Test specification: `02-acceptance-tests.md`.
- Review artifact: `03-document-review.md` (pre-revision record; verdict Approved with notes).
- Revision response: DR-001 addressed by AC-002/AC-010 clarification in requirements and AT-002/AT-010/UT-011 here. DR-002 addressed by assertion-level suite allocation on every AT scenario, including explicit mobile/web/live ownership. DR-003/DR-004 remain coding-plan tasks; `03-document-review.md` is retained as the pre-revision review record.
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-10-18-direct-messaging/`.
- Coverage: 35 Must requirements mapped; 29 acceptance scenarios, 11 unit cases, 20 integration cases, 6 regression cases, 11 data sets and 4 manual checks. All AC-001–AC-029 have explicit scenarios. No executable tests were created or run.
- Recommended first failing test: UT-001/IT-015 base-only login and refresh remain valid when optional messaging authority is advertised, while a chat operation without its required grant is rejected. This establishes optionality and least authority before private data is exposed.
- Suggested order: optional grants/capability and owner fencing → text/report validation and fixed transport/contracts → durable send uncertainty → inbox/history/requests/read/shared actions and declaration merge → atomic sync/baselines/dedupe/request lifetime/catch-up → dispatch eligibility/privacy/lifecycle → Flutter package adapters/navigation/drafts/first-link previews → scheduler fault checks/regression → controlled live/platform functional evidence. Add the smallest meaningful failing test before each implementation slice; no separate integration spike.
- Commands discovered (not run in this stage):
  - `just appview-test-unit`: quick Go suite; explicitly skips PostgreSQL/MinIO and is incomplete integration evidence.
  - `just test`: host Go race suite using current dev Compose PostgreSQL/MinIO; requires the dev services running. Use for real persistence/concurrency evidence.
  - `just appview-check`: isolated release-equivalent backend gate including real migrated storage; use for final backend validation.
  - `just app-test test/chat`: proposed focused Flutter directory after tests exist; root recipe prepares host Rive libraries before `flutter test`.
  - `just app-test test/notifications test/router test/observability`: existing related Flutter regression suites.
  - `just app-test`: complete Flutter host tests; `just app-analyze`: static analysis.
  - `just app-test-integration <device-id>`: existing mobile critical-journeys suite; any new standalone integration file needs an explicit command because this recipe selects `critical_journeys_test.dart` only. This runner does not establish real OAuth/push delivery or web coverage.
- Fixtures/helpers discovered: Go `internal/testdb.WithSchema` for focused tests and `WithMigratedSchema` for PostgreSQL 16 production-schema tests; existing recording/blocking push senders; Flutter retained-account leases, controlled `Completer` races, Riverpod overrides and loopback journey harness. Missing DB configuration can skip tests, so require the fail-closed configured integration path when claiming persistence evidence.
- Blocking gaps: none for document review/coding planning. GAP-002/GAP-003/GAP-006 require genuine external/platform evidence during implementation; GAP-004/GAP-007 need implementation design before executable assertions can be finalized. High-risk implementation requires explicit owner approval; selecting a next documentation stage does not authorize production actions.
- Revision changes: the owner authorized `01-requirements.md` and `02-acceptance-tests.md` updates for DR-001/DR-002. No source/tests/config/dependencies changed, no migration/deployment/live action, no commit or push. The earlier review artifact remains unchanged.
