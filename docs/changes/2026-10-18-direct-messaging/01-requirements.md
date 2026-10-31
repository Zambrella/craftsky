# Requirements: Direct Messaging

## Approved Delete for me: 2026-10-31

FR-020 / AC-030: user explicitly approved Delete for me for their own confirmed
sent text messages, after clarification that the recipient retains their copy.
Add a red long-press/context-menu action with shared destructive confirmation
explaining that consequence. Only a confirmed upstream deletion removes the
message from the owner's view; failure retains content without false success.
Preserve draft, other messages, previews/grouping, pagination and account fences.
No action for received, pending/uncertain local, deleted or unsupported messages.
Server validates active exact child, messaging enablement/restrictions, scopes,
supported direct membership/CraftSky profiles and upstream sender ownership.
Add only chat.bsky.convo.deleteMessageForSelf to messaging OAuth/transport allowlists.
User explicitly said existing sessions need no compatibility handling; standard
reauthorization is sufficient. No delete-for-everyone, PDS record deletion,
migration, automatic replay, live deletion or production action is authorized.
This approved scope supersedes NG-002's exclusion of per-message deletion only
for the described own-message Delete for me action.

## Conversation account block action: 2026-10-31

FR-009 / NFR-006 / AC-009/023: user requested Block account in conversation
options. Use the existing public-profile block mutation, public-block confirmation
copy and CraftSky destructive styling/feedback. Explicit confirmation is required;
opening/cancelling does not mutate. Preserve the draft and conversation, without
automatically leaving/reporting. Disable send/accept while blocking and after an
accepted block. Existing owner/route/lifetime fences and duplicate protection
apply. No new backend, permission, migration or live block test is authorized.

## Approved conversation reporting: 2026-10-31

FR-010 / BR-003 / NFR-002/003/006 / AC-010/023: add red Report conversation in
the details More menu when the current account has reporting capability. Reuse
the full-screen report flow with an empty initial reason, optional explanation,
app-bar Submit and Bluesky-only notice. Submit a chat.bsky.convo.defs#convoRef
with conversation and peer DID derived from the current authorized direct convo.
No transcript upload, automatic leaving/blocking or CraftSky moderation report.
Existing bounds, explicit single submission, uncertain outcomes/status checking,
owner/connection fences and message reporting remain. User explicitly approved
this local backend/client extension; live report submission remains excluded.

## Approved Messages navigation corrections: 2026-10-31

FR-003/016, NFR-006, AC-004/014/023: Messages behaves as a top-level shell
section like Feed/Saved/Scheduled, with drawer access instead of Back, including
loading, first-use setup and error states. Requests/new conversation/details stay
child routes. Conversation details supports the standard iOS edge-swipe Back;
flush its local draft on a successful pop, retain later edits and known save-error
guards, and preserve explicit Back/leave behavior and account isolation.

## Approved time sections: 2026-10-31

FR-003/005, NFR-006/008, AC-004/006/023: show a centred, muted date/time label
before the first loaded authoritative message, a new local calendar day, or an
elapsed gap of at least five minutes, independent of sender. Use Today/Yesterday
or weekday/date with the local time; include year for another year. Nearby
messages from the same sender use tighter package grouping. Hide normal bubble
timestamps, retain pending/error indicators and profile/post previews. Local
outgoing placeholders have no fabricated date. Preserve anchoring, read tracking,
identity/ownership, recovery and composer behavior.

## Approved immediate composer clearing: 2026-10-31

FR-005/006, AC-006/007: clear the visible input immediately on explicit Send,
while the pending bubble and existing durable attempt retain the submitted text.
Subsequent edits, including an identical message or deliberately empty input,
must survive confirmation. Preserve existing failure/uncertainty recovery and
local-save rollback; no network, persistence format or replay change.

## Approved immediate send presentation: 2026-10-31

FR-005 / FR-006 / NFR-006 / NFR-008 / AC-006 / AC-007: show the outgoing
text bubble immediately after explicit Send, using the existing package renderer.
While pending, replace its timestamp with the package sending spinner. On
confirmation, reconcile by operation receipt/message ID into one bubble with
the authoritative timestamp. Keep profile/post previews on confirmed messages.
Failed/uncertain attempts stop spinning and retain existing themed warning and
explicit recovery actions; no inferred delivery/read state, text-based success
matching, automatic replay or new persistence. Preserve later composer edits,
account/access/route isolation, request restrictions and visible-only read logic.

## Report presentation revision: 2026-10-30

FR-010 / NFR-006 / AC-010 / AC-023: style Report message with the shared red
menu action. Present the message-report form using the existing full-screen,
bottom-up report modal route, with a close control and Submit action in the
app bar. Preserve keyboard-safe scrolling. The enabled composer send button
uses the primary button variant, retaining the shared disabled styling. Keep
only the Bluesky-only destination notice; remove the second evidence paragraph
at the owner's explicit request. Report payload, reasons, validation, explicit
submission, uncertain outcomes and no-replay behavior remain unchanged. The
reason starts unselected, and Submit stays disabled until a reason is chosen.

## Approved package-owned conversation UI: 2026-10-30

NFR-008 / NFR-006 / FR-003 / FR-010 / FR-012 / AC-004 / AC-010 / AC-023:
use flutter_chat_ui's standard message bubbles, timestamps, grouping and animated
wrappers, themed from CraftSky. Bubble outlines may be dropped. Preserve
profile/post previews and their guarded navigation, custom draft-aware composer,
request controls, history anchoring, visibility/read checks and disabled private
media caching. Use the package's preview extension point. Reports move to the
shared long-press/secondary-click menu for received supported messages only;
opening/dismissing a menu never submits a report. Standard text replaces custom
inline-link rendering; explicit Copy and Open link menu actions retain access to
message text and original URLs. Native delivery/read icons remain hidden because
we do not have authoritative recipient delivery/seen state. No dependencies,
backend, permissions, migration, live message/report or production change.

## Shared profile integration correction: 2026-10-30

FR-003 / NFR-006 / AC-023: messaging profile displays must use the existing
shared profile read and identity/avatar widgets to show actual crafts and
customisation. Keep chat content usable during metadata loading/failure.
Profile-opening taps use the shared avatar/name/handle/craft hero transition in
both directions, preserving reduced motion, account/route fences and drafts.
Conversation and recipient rows retain their current conversation actions.

## Approved initial-load optimization: 2026-10-21

FR-003, FR-019 / NFR-001, NFR-002, NFR-003, NFR-004 / AC-004, AC-016, AC-021,
AC-028: load conversation metadata and its first message page through one
AppView endpoint. Reuse one current session/capability/enablement/status boundary
and one membership/profile lookup; preserve blocks, history mapping, opaque
pagination and fail-closed errors. Restore the local draft concurrently with
that request; publish the initial view only when both complete under the same
active lease/content epoch. Retain typed failures, cache policy and no automatic
read/accept/send/report mutation. Existing summary and history routes remain.
No session-lock redesign, permission widening, persisted message cache, schema
change or live performance claim is authorized or required.

## Approved conversation loading header: 2026-10-21

NFR-006 / FR-003 / AC-023: during initial conversation-details loading, the
AppBar shows a themed profile skeleton (avatar, name and handle) alongside the
message skeleton body. Apply during both status and content acquisition. Reuse
the shared shimmer/reduced-motion treatment; the placeholder is noninteractive.
Loaded identity, compact-profile action and silent background refresh remain.

## Approved silent background refresh: 2026-10-21

FR-015 / NFR-006 / AC-017, AC-023: remove the mini AppBar spinner for messaging
background synchronization, refresh and pagination. Continue all existing fetch
and polling behavior silently with loaded content and empty states visible.
Keep initial skeletons and explicit user-action progress. Existing CraftSky
warning/error messages and retry actions still report degraded or failed work.
This supersedes earlier title-spinner requirements for background loading.

## Approved conversation profile action: 2026-10-21

FR-003 / NFR-006 / AC-023: tapping the peer's avatar or name/handle in the
conversation details AppBar opens the existing compact profile presentation for
that peer's DID. Back returns to the same conversation with its draft intact.
Use the established profile route/helper and current-account navigation guard;
no new modal, profile fetching path or automatic conversation mutation.

## Approved identity presentation: 2026-10-21

NFR-006 / FR-002, FR-003 / AC-023: chats and pending-request rows, plus the
conversation details AppBar, show the participant's handle under their display
name using the existing compact profile identity treatment. Preserve list
message previews, avatars, badges and actions. Truncate long identity lines;
support large text and retain a single handle fallback when no name is present.

## Approved request-action revision: 2026-10-21

FR-008, FR-009 / AC-009, AC-012 / NFR-006: request conversation details remove
the explainer text and retain Accept request alongside an Ignore action. Ignore
calls the existing guarded leave operation directly, without another confirmation,
and closes details only after confirmed success. It does not accept, send, block
or change incoming policy. Both actions prevent overlap while pending; failures
use existing themed feedback. The More-menu Leave confirmation remains unchanged.

## Approved loading-style revision: 2026-10-21

NFR-006 / FR-015 / AC-023: initial chats, requests, recipient lists and
conversation details use the existing CraftSky feed skeleton treatment instead
of centered spinners. Reuse theme-aware shimmer, reduced-motion and loading
semantics. Conversation placeholders reflect message bubbles without fabricated
private content. Keep loaded content/empty states and the small AppBar loader
for background fetches; action, authorization and settings progress is unchanged.
This supersedes the earlier centered-spinner requirement for these content views.

## Approved revision: active-account messaging lifetime, 2026-10-21

FR-002, FR-007, FR-015, FR-016, FR-019 / NFR-002, NFR-004, NFR-005:
the authenticated shell owns lightweight messaging status/repository state for
the active account, and eligible unread counts/sync coordination while enabled.
Successful chats/requests lists remain loaded after first use. Conversation
history uses a bounded memory-only cache; recipient search/report forms remain
page-scoped. Retention does not cause hidden-page polling, reads, sends or reports.
Account switch/logout, disablement, permission loss and generation changes
evict affected private caches; durable draft rules remain unchanged. Initial
implementation policy: at most three idle conversations for up to two minutes;
actively displayed conversations are not evicted by that idle-cache bound.

## Approved correction: direct profile Message action, 2026-10-21

FR-004 / AC-005: an explicit Message tap on an available profile starts/reuses
the canonical conversation and navigates directly to its details under Messages.
Do not route the profile action through New message or require another Start
button. Keep unavailable-recipient snackbar feedback and disabled-account setup.
Profile viewing and cold recipient URLs remain read-only. Pending actions show
the existing button progress and prevent duplicate or stale-owner navigation.

## Approved revision: Messages child routes, 2026-10-21

FR-002, FR-004, FR-018 / AC-005, AC-027: Requests, New message and Conversation
are declared children of Messages. Keep their existing canonical URLs. Cold
links and profile entry build the Messages parent so Back reaches the inbox;
opening a conversation from New message uses Messages → Conversation, without
retaining the recipient picker. Direct messages settings stays separate.

## Approved revision: search-result conversation start, 2026-10-21

FR-004, FR-018 / AC-005, AC-027: tapping a CraftSky profile search result in
New message is the explicit start action. Check availability and open/create
the canonical direct conversation without an additional Start conversation
button. A denied recipient shows the existing CraftSky snackbar and causes no
create/send. Prevent overlapping selection/start requests. Merely searching,
checking availability or loading a cold recipient route remains read-only.

## Approved correction: CraftSky feedback delivery, 2026-10-21

NFR-006, FR-015 / AC-017, AC-023: messaging alerts must use the existing AppMessenger and its standard floating snackbar, including info/warning/error severity and retry/dismiss semantics. A custom banner that only reuses snackbar styling does not satisfy this requirement. No alert layout may sit above the page Scaffold and displace its AppBar. Routine synchronization uses the discreet title loader; initial loads remain centered. Persistent request, permission and uncertain-operation explanations remain ordinary in-page content with existing explicit actions. This supersedes earlier custom inline-banner presentation decisions, without changing messaging business behavior.

## Approved revision: combined OAuth and messaging setup, 2026-10-21

The maintainer agreed to the proposed combined flow and explicitly requested implementation, including the local auth/permission changes. This revision supersedes earlier requirements that chat permissions are requested only at enablement; the separate OAuth step is retained solely as recovery for missing/revoked authority, not as the normal flow.

- BR-002, RULE-004 / AC-002, AC-019: login and registration request the existing narrow chat permission bundle alongside base permissions. Base permission validation remains separate: a base-only grant/session/refresh keeps ordinary CraftSky usable. OAuth completion never enables messaging or overwrites account push/audience choices.
- BR-003, FR-001, FR-016 / AC-002, AC-014: a disabled account's first Messages visit shows Direct messages settings, a concise shared-inbox/storage/privacy/reporting explainer and an initially off Enable messaging Material switch. Enabling is an explicit account-wide CraftSky action; it opens the regular conversation list as soon as confirmed readable access is published. Explicit settings visits remain on settings after toggling enablement. Later Messages visits open the inbox when enabled. Existing enabled accounts retain their choice.
- FR-011, FR-014 / AC-011, AC-015: only enabled messaging shows Push notifications and Who can message you. Push defaults on for new enablement; prior choices survive disable/re-enable. Read the existing shared incoming policy, never overwrite it on enable/disable. Explain that its effect includes Bluesky/other apps.
- FR-016, FR-019 / AC-014, AC-028: switching off stops CraftSky messaging access and delivery across devices, keeps hidden local drafts and does not disable Bluesky messaging, delete conversations or revoke ordinary CraftSky access. Re-enabling stays possible on the same settings screen.
- FR-001, NFR-001, NFR-006 / AC-002, AC-016, AC-023: missing permission has explicit OAuth recovery; recovery grants capability without silently enabling messaging or changing preferences. Already-enabled accounts resume from retained sync progress after permission recovery; disabled accounts remain paused. Maintain account/generation fences, restricted/unavailable states, themed feedback, central initial loaders and discreet background loaders.

No production change, migration, broad scope, live message/report or automatic retry is authorized. Owner-run live validation remains separate.

## Approved revision: profile actions, 2026-10-20

The maintainer explicitly requests FR-004 / NFR-006 / NFR-008, AC-005 / AC-023 / AC-027: move profile mute/unmute from the primary action row into the existing More profile actions menu. Preserve current relationship state, availability guards, busy disabling and mutation callbacks. Keep the profile Message button tappable when recipient availability denies starting a conversation; tapping shows the existing themed informational snackbar and performs no create/send/navigation. Failed or incomplete availability checks use generic unavailable feedback rather than claim the recipient has disabled messages. Existing explicit enablement, account ownership and block restrictions remain.


## Approved revision: 2026-10-20 follow-up

The maintainer explicitly requested these changes after UI polish. This revision supersedes conflicting earlier external-correspondent, native badge aggregate and tab-navigation wording, including SIM-01, BR-001, FR-007, FR-018 and EC-006 below.

- BR-001 / FR-002 / FR-004 / FR-018, AC-030: AppView exposes messaging only where the owner and peer have indexed CraftSky profiles and no known inactive owner lifecycle. Conversations begun elsewhere remain eligible when both participants meet that rule. Filter accepted/request lists, direct conversation/history access, availability/start, actions/reporting, notification resolution, synchronization and final push delivery. A missing profile makes the conversation unavailable; never delete or alter upstream history to implement this filter. Profile existence is distinct from a local OAuth/session registration.
- FR-007, AC-008 / AC-031: Accepted badge counts eligible unread direct conversations, excluding muted, blocked and upstream chat-disabled peers. Request badge counts **all eligible pending direct requests**, including requests already read; reading never accepts/removes them. Compute counts from paginated authoritative lists with profile filtering because global upstream aggregates include hidden peers. Deduplicate conversations, cap at sentinel 100 (`99+`), and refresh after actions. Bound provider work; incomplete/failed enumeration must not publish a partial count as complete. Keep read watermarks unchanged.
- FR-002 / FR-018 / NFR-006, AC-031: Accepted chats have a New message FAB. Replace the tab view with a separate `/messages/requests` route accessed through an AppBar button with the pending-request badge. Keep typed route navigation and account guards.
- FR-015 / NFR-006 / NFR-008, AC-023 / AC-031: Pull to refresh uses the shared yarn control. Background refresh keeps current content and shows a small stitch indicator next to the AppBar title rather than a prominent in-content loader. The conversation composer has the hint `Message`, no visible field heading, and aligned field/send controls; retain an accessible label, multiline editing, validation and keyboard usability.

These explicit local implementation instructions authorize the narrower eligibility behavior. Production deployment, live messages/reports, credential changes and database migrations remain outside this revision.

## 1. Initial Request

Add an optional Bluesky-backed inbox to CraftSky, initially for one-to-one text messages. Include conversation requests, paginated history, unread counts, read updates, mute/unmute, leave, reporting, profile entry points, notification navigation, and incoming-message synchronization. Flutter continues to use its CraftSky session with AppView; AppView retains OAuth credentials and accesses the chat service through the user's PDS.

The brief proposes roughly 10–12 CraftSky API operations, explicit chat RPC permissions instead of the broad Full Chat Client permission set, and a background `getLog` worker extending durable push delivery. Endpoint count is an estimate, not an acceptance constraint. The inbox includes conversations started outside CraftSky. Implement the full integration in one effort, without a separate feasibility spike or latency-measurement requirement.

Sections 8, 12, and 13 define the scope exclusions, requirements, and acceptance criteria. The remaining sections provide decision history, supporting evidence, risks, and planning context; they do not add separate acceptance obligations.

This artifact defines requirements and records the requested read-only feasibility audit. It does not start feature implementation, executable test development, migrations, deployment, or live messaging actions. The audit includes a dependency-resolution dry run in a temporary directory, not an app dependency change.

Follow-up on 2026-10-19: CraftSky links in messages load content previews when possible. The owner selected `flutter_chat_ui` for the conversation interface and completed three grilling rounds. All 24 resulting decisions below are confirmed, with the later SIM-01–SIM-04 simplifications taking precedence; this revision does not start test design or implementation.

## 2. Current Codebase Findings

Read-only inspection on 2026-10-18 found:

- `appview/internal/auth/scopes.go`: current default OAuth permissions contain no chat authority. They grant CraftSky collections and selected Bluesky profile, follow, block, and media operations.
- `appview/internal/auth/session_coordinator.go`: effective session permissions are checked against configured app permissions. Optional messaging needs capability-aware checks; merely adding mandatory chat scopes to every existing login would conflict with optional enablement.
- `appview/internal/auth/background_session_selector.go`: background work selects active OAuth sessions backed by active, unexpired CraftSky sessions and the current owner lifecycle/auth epoch. This provides a pattern, but currently does not select specifically for chat capability.
- `appview/internal/push/dispatcher.go`: durable push delivery uses leases/retries and active account subscriptions/installations. It currently joins social notification data and suppresses delivery for owner lifecycle, actor mutes, and either-direction indexed blocks. Chat needs explicit eligibility semantics rather than assuming social notification policy is identical.
- `appview/internal/notifications/preferences.go` and `app/lib/notifications/models/notification_preferences.dart`: category preferences include push enablement and social-audience filters. Existing defaults enable push. A DM preference must distinguish push delivery from the shared incoming-message policy.
- `app/lib/notifications/models/notification_destination.dart`, `app/lib/notifications/services/notification_navigation.dart`, and `app/lib/router/router.dart`: existing typed notification destinations and routing need a conversation destination with account-safe resolution. No chat feature was found in the inspected auth, route, notification, or Flutter feature paths.
- `appview/internal/routes/routes_profile_notification.go`: existing profile/notification route registration is relevant to entry points and preferences.
- Follow-up inspection on 2026-10-19: `app/lib/shared/sharing/record_links.dart` generates canonical `https://app.craftsky.social` links for posts (including focused replies), profiles, and business events. `app/lib/feed/models/link_preview.dart` provides an existing composer preview model, but does not establish a DM content-preview resolver or chat UI package choice.
- Grilling follow-up: ordinary logout is device/session-scoped (`appview/internal/auth/session_lifecycle.go`), while sign-out-all advances the owner epoch. Background selection requires a usable active child session; the current default inactivity limit is 30 days (`appview/internal/auth/craftsky_session.go`). Do not let polling extend it implicitly.
- Profile mute uses CraftSky-local `actor_mutes` (`appview/internal/relationships/store.go`), distinct from shared chat conversation mute. Reusing the social push dispatcher must not accidentally inherit profile-mute suppression for DMs.
- Constraints: [architecture reference](../../../atproto-craft-social-app-reference.md), [API architecture](../../superpowers/specs/2026-04-21-appview-api-architecture-design.md), [wire casing](../../superpowers/specs/2026-04-22-api-wire-alignment-design.md), [logging guide](../../development/logging-error-reporting.md), and repository AGENTS.md. Consumer APIs use `/v1/`, CraftSky bearer/device headers, camelCase JSON, standard error envelopes, and opaque cursors. Supported app targets are iOS, Android, and web.
- Commands discovered for later stages: `just test` (host Go tests against the dev stack), `just appview-check` (full backend gate), `just app-test` (prepares host Rive libraries, then Flutter tests), and `just app-analyze`. None were run for this documentation-only change.

External evidence checked during requirements work:

- [Permission specification](https://atproto.com/specs/permission): explicit RPC permissions can bind methods to a service audience. Published permission sets can change on refresh, reinforcing the requested explicit allowlist.
- [API routing](https://bsky.network/docs/api-directory/): authenticated chat calls are PDS-proxied to `did:web:api.bsky.chat#bsky_chat`.
- [Conversation creation contract](https://github.com/bluesky-social/atproto/blob/main/packages/api/lexicons/chat/bsky/convo/getConvoForMembers.json): retrieves or creates the same direct conversation for the participants.
- [Chat definitions](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky/convo/defs.json): distinguish direct/group conversations, request/accepted states, deleted messages, and text bounds of 1,000 graphemes and 10,000 UTF-8 bytes. These are upstream bounds, independent of CraftSky post limits.
- [Event log contract](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky/convo/getLog.json): supplies optional cursors and typed events, including events outside this release's scope. Cursor expiry, initial-baseline behavior, and recovery guarantees were not established by this schema.
- [Send contract](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky/convo/sendMessage.json): the retrieved schema has no caller-supplied idempotency field. This is not evidence of exactly-once sending support.
- Earlier raw-link failures are resolved by the pinned audit in section 24: packages/api contains link files for authFullChatClient and leaveConvo, whose actual definitions live in lexicons-overrides. Section 24 supplies the complete RPC inventory and exact permission/routing matrix. Schemas and official client behavior are verified; live grant acceptance, reporting acceptance, and recovery semantics remain to be validated during full integration (NFR-007/AC-024).
- Grilling source checks found upstream schemas now under `packages/api/lexicons/`; the contract links above use that location. [Official request UI](https://raw.githubusercontent.com/bluesky-social/social-app/main/src/screens/Messages/components/RequestButtons.tsx) describes reply-as-acceptance, but CraftSky deliberately requires explicit Accept first. [Official leave copy](https://raw.githubusercontent.com/bluesky-social/social-app/main/src/components/dms/LeaveConvoPrompt.tsx) describes deletion for the leaving viewer, not other participants. Reappearance/history restoration/new-request transitions remain unverified service behavior to check during integration.
- Chat message schemas permit specific record/join-link embeds, not a generic external card payload. CraftSky previews therefore hydrate text links locally; their display in another client is not promised. [flutter_chat_ui](https://pub.dev/packages/flutter_chat_ui) and its [builder API](https://pub.dev/documentation/flutter_chat_core/latest/flutter_chat_core/Builders-class.html) support a backend-independent UI with custom cards/composer; an isolated dependency dry run subsequently resolved flutter_chat_ui 2.12.0 with CraftSky’s existing constraints (section 24). No repository dependencies, application code, builds, or device checks were changed/run.
- The linked [product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit) was fetched through Google Drive. Its portability and in-house moderation commitments need an explicit DM qualification: hosted chat is outside public PDS record storage, and v1 DM reports go only to Bluesky. Earlier vision discussion placed DMs after launch; this request plans the feature without changing the release schedule or adding group chat.

## 3. Clarifying Questions And Decisions

### Q1: Who receives a message report?
Answer: Bluesky only.
Decision / implication: Private-message evidence must not enter CraftSky's moderator queue. The report flow names Bluesky as the recipient and confirms success only after upstream acceptance.

### Q2: What happens to an existing conversation when either participant blocks the other?
Answer: Keep history readable; disable sending and suppress push.
Decision / implication: CraftSky enforces this on the server as well as in the UI. History remains available only where the upstream service permits it; CraftSky does not bypass upstream restrictions or retain an independent archive to defeat them.

### Q3: What appears in DM push notifications?
Answer: Sender name only.
Decision / implication: A notification can identify its sender and say that they sent a message, but contains no message text or embedded-content preview. Missing sender names use generic copy.

### Q4: Include “Who can message me” in v1?
Answer: Yes: everyone / people I follow / nobody.
Decision / implication: Explain that this is the shared Bluesky chat setting and affects other apps. Enabling CraftSky messaging must preserve the existing setting until the owner explicitly changes it.

Other direction comes from the initial brief: optional enablement, granular permissions, one-to-one text only, shared external conversations, and AppView-mediated access. SIM-02 below supersedes the initial proposal for a separate feasibility spike. Section 20 distinguishes resolved assumptions from the remaining provider-feasibility assumption.

### U5: CraftSky content link previews
User request: CraftSky links load a preview of the content when possible.
Decision / implication: Render a preview for supported CraftSky content links in sent and received messages when the viewer can access the content. Keep the original link usable when a preview cannot be loaded. This is a conversation display feature and does not require publishing a Bluesky record embed or including previews in push notifications.

### U6: Reuse a Flutter chat interface package
User preference: Strongly prefer an existing Flutter package for the chat/conversation interface.
Decision / implication: Initially a Should reuse preference; superseded by GQ-12 and Must NFR-008 selecting flutter_chat_ui. The dependency dry run passed; package adapters, styling, accessibility, rendering, and platform builds remain to be validated.

### Confirmed grilling decisions (2026-10-19)

The owner accepted the recommendations for GQ-01–GQ-10 and GQ-13–GQ-24, explicitly rejected composer previews in GQ-11, selected `flutter_chat_ui` in GQ-12, and ended the grilling with an instruction to update this artifact. These are product decisions, not unresolved assumptions or proof of live provider behavior.

| Decision | Topic | Confirmed outcome |
|---|---|---|
| GQ-01 | Account lifecycle | Enable/disconnect messaging account-wide. Ordinary logout affects only that device; other valid sessions continue. Signing out everywhere stops background access. |
| GQ-02 | Push default | DM push starts enabled after messaging enablement, subject to OS permission/subscriptions; show its toggle during enablement. |
| GQ-03 | Request push | Notify once for a new unaccepted conversation; suppress further message pushes until acceptance. |
| GQ-04 | Mute separation | CraftSky profile mute affects social content only. Shared conversation mute controls DM notifications; blocks keep the stronger confirmed policy. |
| GQ-05 | Main badge | Updated by SIM-01: use the upstream accepted-direct unread aggregate and its filtering/cap semantics (FR-007). Requests have a separate upstream indicator. |
| GQ-06 | Freshness | Use a 4-second foreground polling interval to match Bluesky’s client. SIM-02 removes the earlier arrival-latency targets and measurements; NFR-005 defines the cadence and backoff rules. |
| GQ-07 | New message | Include a New message action with CraftSky profile search. Defer global Bluesky handle search; preserve existing external conversations and profile entry points. |
| GQ-08 | Local drafts | Preserve one local draft per account/conversation across navigation and restarts. Clear on send or explicit discard; remove the account’s drafts on logout from that device. No cross-device draft sync. |
| GQ-09 | Uncertain sends | Refresh history first; if unresolved, retain text and offer explicit Send again with a duplicate-risk explanation. Never automatically resend. |
| GQ-10 | Preview targets | Support posts, profiles, and business events. A focused reply link previews the reply and opens the thread focused on it. |
| GQ-11 | No composer preview | Show previews only in message bubbles, never while composing. |
| GQ-12 | Conversation UI | Use `flutter_chat_ui` for the conversation interface, with CraftSky-specific presentation/adapters as needed. Validate compatibility; raise concrete blockers before substituting a different package or custom interface. |
| GQ-13 | Push toggle scope | Use one account-wide DM push toggle; retain per-device OS notification permissions. |
| GQ-14 | Requests indicator | Updated by SIM-01: use the upstream direct-request unread aggregate (FR-007). Reading does not accept a request or remove it from the list. |
| GQ-15 | Web boundary | Support web inbox and foreground updates. Background push is iOS/Android only in v1. |
| GQ-16 | Session expiry | Preserve existing session expiry, normally 30 days of inactivity. Stop background access when no usable session remains; explain paused notifications and reauthentication. |
| GQ-17 | Drafts on disable | Disabling messaging retains local drafts, hidden until re-enabled. Logout and explicit discard still remove them; explain this distinction during disconnect. |
| GQ-18 | Preview limit | Updated by SIM-03: attempt only the first supported CraftSky link per message; do not try later links if it cannot produce a card (FR-017). Keep every URL usable as text. |
| GQ-19 | Compact previews | Show identifying text, short excerpt/event details, and a permitted thumbnail where available. Open full content on tap; no inline video playback or nested interactive post UI. |
| GQ-20 | Explicit acceptance | Require Accept before enabling a request’s composer. Reading alone never accepts. |
| GQ-21 | Request deduplication | Leaving, reinstalling, or reconnecting does not reset request notification eligibility. Allow one new notification only after acceptance followed by a genuinely new request transition, subject to verified upstream behavior. |
| GQ-22 | Catch-up | After a temporary interruption or reauthentication, allow one catch-up notification per eligible unread accepted conversation instead of replaying missed messages. Do not notify pre-enable history; respect current read/mute/block settings. |
| GQ-23 | Request previews | Before acceptance show text and usable links, without automatic content-preview cards. Cards become eligible after acceptance. |
| GQ-24 | Offline sends | Do not queue offline sends for automatic delivery. Retain the draft, explain connection requirements, and require the owner to send after reconnecting. Already-submitted uncertain sends use GQ-09. |

### Confirmed simplifications (2026-10-19)

These owner-approved decisions supersede conflicting earlier wording, including the original spike proposal and the affected grilling outcomes.

| Decision | Confirmed outcome | Authoritative requirements |
|---|---|---|
| SIM-01 | Use Bluesky’s unread aggregates directly, including its exclusions and capped totals; do not add a custom badge reconciliation system. | FR-007, AC-008 |
| SIM-02 | Proceed with full integration without a separate spike or pre-design live-service gate. Remove latency targets and measurements; retain four-second foreground polling and functional integration validation. | NFR-005, NFR-007, AC-022, AC-024 |
| SIM-03 | Attempt the first supported CraftSky link only. If it cannot produce a preview, leave the links as text rather than searching later candidates. | FR-017, AC-025 |
| SIM-04 | Keep scope exclusions, requirements, and acceptance criteria in sections 8, 12, and 13 authoritative. Other sections retain provenance and reference material without adding obligations. | Sections 8, 12, 13 |

## 4. Candidate Approaches

### Option A: Embedded Bluesky inbox with explicit permissions
Summary: AppView proxies an allowlisted chat API and runs account-scoped synchronization; Flutter provides the inbox.
Pros: Matches the request; reuses hosted storage/delivery and existing identity; limits requested authority.
Cons: Requires OAuth lifecycle work, sync, push, and careful retry handling; settings and conversations are shared with other clients.
Risks: Granular grant compatibility, provider availability/rate limits, private-data exposure, and ambiguous sends.

### Option B: Embedded Bluesky inbox with Full Chat Client access
Summary: Use the same inbox architecture with the published broad permission set.
Pros: Less initial permission enumeration.
Cons: Grants operations outside v1 and can expand through permission-set updates.
Risks: Excess authority and weaker correspondence between consent and feature scope. Not selected; no automatic fallback is authorized.

### Option C: External chat-app handoff or a CraftSky messaging backend
Summary: Link users to another app, or build a separate private messaging service.
Pros: A handoff avoids an embedded chat client; an independent backend provides greater product control.
Cons: A handoff does not deliver the requested CraftSky inbox; an independent backend adds storage, delivery, and interoperability work.
Risks: Fragmented experience or substantially larger operational scope. Not selected.

## 5. Recommended Direction

Use Option A as one full integration effort. Keep Bluesky authoritative for conversations, messages, request state, read state, and chat mute state. Store only the private operational metadata CraftSky needs for enablement, synchronization, safe command handling, preferences, and notification delivery. Do not create CraftSky message records or index DMs through Tap/firehose ingestion.

This follows the supplied direction and existing token-mediating backend boundary. Validate the required capabilities and behavior during implementation under NFR-007/AC-024; no separate spike or pre-design live-service gate applies. If required granular authority is unavailable, return to requirements review rather than silently widening access or dropping safety features.

## 6. Problem / Opportunity

Crafters need private one-to-one communication without leaving CraftSky. A hosted chat integration can provide it without operating a separate message storage and delivery service, provided users understand the shared inbox and CraftSky handles privacy, safety, authorization, and failures reliably.

## 7. Goals

- G-001: Provide a useful one-to-one inbox with conversation requests and unread state.
- G-002: Preserve server-side OAuth authority and make messaging optional and reversible.
- G-003: Deliver timely CraftSky notifications without duplicate jobs, stale eligibility, or message previews.
- G-004: Respect blocks, shared messaging preferences, and the selected Bluesky reporting boundary.
- G-005: Integrate hosted chat through CraftSky’s existing auth, API, and delivery boundaries, validating functional behavior as part of the implementation.

## 8. Non-Goals

- NG-001: Group chat creation, participation UI, invitations, or group notifications.
- NG-002: Sending attachments, media, upstream record embeds, reactions, reply threading, editing, deletion, export, or chat-account deletion. Local rendering of previews for CraftSky links in message text is included by FR-017; composer and unaccepted-request previews are excluded.
- NG-003: A CraftSky-only conversation namespace, recipient membership requirement, or filtering direct conversations by which app started them.
- NG-004: A new message transport/storage service, public message records, Tap indexing, or a promise of end-to-end encryption or PDS-based message portability.
- NG-005: Full offline inbox/history, durable offline sends, cross-device draft synchronization, typing indicators, presence, or recipient read receipts in the UI. Local unsent drafts are included; automatic offline-send queues are excluded.
- NG-006: Cross-app push deduplication, changing Bluesky's notification preferences, or a CraftSky DM moderation console.
- NG-007: Global Bluesky recipient search, monetization changes, release scheduling, or native desktop build targets. Include New message with CraftSky profile search, profile entry points, and existing inbox conversations; global Bluesky search remains excluded.
- NG-008: A separate integration spike, latency targets/measurements, or performance benchmarking. The four-second foreground polling configuration remains in scope (NFR-005).

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| CraftSky account owner | Enables messaging for their existing identity | Inbox, control, safe sends, and clear consent |
| Correspondent | Same upstream conversation participant; may not use CraftSky | Standard interoperable one-to-one delivery |
| AppView | Mediates authenticated operations and background work | Current capability, lifecycle fencing, durable progress |
| Bluesky chat/moderation services | Host chat and receive DM reports | Correct routing, authority, and evidence contracts |
| CraftSky operator | Maintains availability and delivery | Bounded operational diagnostics without access to message evidence |

## 10. Current Behavior

The inspected CraftSky paths implement social notifications and server-side OAuth, with no chat inbox or chat permissions. Existing private report handling and social notification rows must not accidentally become a permanent DM content store or a private-message moderation queue.

## 11. Desired Behavior

Login and registration request the narrow messaging authority alongside base permissions, without enabling messaging. On the first Messages visit, an owner sees Direct messages settings with shared-inbox, storage/privacy and reporting disclosures, then explicitly chooses account-wide Enable messaging. First enablement from Messages opens the conversation list immediately after confirmation; enabled users can adjust push and audience preferences in Direct messages settings. Subsequent enabled visits open the inbox directly. Their existing one-to-one conversations and requests appear, including correspondents outside CraftSky. The owner can start a conversation from a profile or New message with CraftSky profile search when permitted, send text, read history, accept requests, mute/unmute, leave, report to Bluesky, and change who may message them.

Foreground views refresh from authoritative state. Authorized background synchronization detects new incoming messages and schedules eligible sender-name-only pushes. Disabling messaging stops CraftSky access and queued delivery without deleting the upstream conversations, changing incoming-message policy, or affecting unrelated CraftSky features.

The first supported CraftSky content link in a message is eligible for a preview when its target can be resolved for the current viewer (FR-017); later links are not fallback candidates. Tapping the preview opens that content in CraftSky; unavailable previews leave the message and link intact. Use `flutter_chat_ui` for the conversation timeline/composer, with CraftSky-specific adapters/cards. Show compact previews only in accepted-conversation message bubbles, never in the composer or unaccepted requests. Persist local drafts as specified in FR-019; offline drafts never auto-send.

## 12. Requirements

Sources: P = initial prompt; U1–U6 = answers and follow-up requests in section 3; C = codebase findings; E = external contracts in section 2; GQ-01–GQ-24 = confirmed grilling decisions above; SIM-01–SIM-04 = approved simplifications; A = remaining explicit assumption in section 20.

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Owners can conduct one-to-one text conversations within CraftSky, including conversations begun in other apps and with non-CraftSky correspondents. | Useful interoperable inbox | P | AC-001, AC-003, AC-005 |
| BR-002 | Business | Must | Messaging is optional; declining, disabling, or losing chat authority does not remove ordinary CraftSky access. | Independent consent and recovery | P, C | AC-002, AC-014 |
| BR-003 | Business | Must | Enablement explains shared history/settings, Bluesky-hosted storage, and Bluesky-only DM reporting without promising CraftSky-only privacy, PDS portability, or end-to-end encryption. | Honest product boundary | P, U1, U4, vision | AC-002, AC-010 |
| FR-001 | Functional | Must | Expose disabled, authorization-needed, ready, syncing/degraded, and unavailable messaging states with appropriate recovery. Consult chat.bsky.actor.getStatus and service errors to distinguish an effective grant from actual chat eligibility; do not invent an age/email bypass or infer a precise restriction reason from a generic disabled flag. | Avoid misleading empty inboxes | P, C | AC-002, AC-014, AC-017 |
| FR-002 | Functional | Must | List supported direct conversations and requests with participant identity, latest-message state, unread count, and mute/request status; use direct-kind accepted listing and dedicated request listing, continuing opaque pagination through short/empty filtered pages. Discard group/join-request variants without treating them as direct conversations. | Navigable inbox | P, E | AC-001, AC-003 |
| FR-003 | Functional | Must | Open paginated message history in chronological display order, preserving position while loading older messages and safely rendering deleted or unsupported content. | Usable history | P, E | AC-004 |
| FR-004 | Functional | Must | Check availability without creating a conversation; explicitly starting a conversation retrieves or creates the canonical direct conversation for the authenticated owner and exactly one other DID. | Prevent accidental creation or groups | P, E | AC-005 |
| FR-005 | Functional | Must | Send plain text with visible pending, confirmed, failed, and uncertain states; enforce nonblank input and upstream grapheme/UTF-8 limits in both layers without silent truncation. Offline sending preserves a local draft and explains the connection requirement; reconnecting never automatically submits it. | Predictable valid sends | P, E, GQ-24 | AC-006, AC-028 |
| FR-006 | Functional | Must | Associate sends with stable owner-scoped operation identities; duplicate submission must not cause another upstream send. On an uncertain outcome, refresh history without automatic replay or false confirmation; if still unresolved retain text and offer explicit Send again with a duplicate-risk explanation as a new operation. | Prevent avoidable duplicates | P, E, GQ-09 | AC-007 |
| FR-007 | Functional | Must | Mark read through the last message actually presented in an active conversation and reconcile remote reads. Show upstream unread-message counts per row. Use getUnreadCounts(includeGroupChats=false) directly for the accepted-conversation badge and separate request indicator, preserving upstream exclusions (including muted, locked, blocked, and deleted/deactivated cases). Display the sentinel 100 as 99+, not an exact total, and refresh rather than decrementing it as exact. Do not derive alternate badge counts from conversation lists. Reading a request updates upstream unread state without accepting or removing it. | Native unread state without custom counting | P, GQ-05, GQ-14, SIM-01 | AC-008, AC-012 |
| FR-008 | Functional | Must | Distinguish requests from accepted conversations; require confirmed explicit acceptance before enabling the request composer. Reading does not accept. Support confirmed leave as dismissal, following verified upstream semantics. | Request control | P, GQ-20 | AC-003, AC-009 |
| FR-009 | Functional | Must | Support shared upstream conversation mute/unmute and explicit confirmed leave; reconcile external changes. Leave confirmation explains its effect on the owner’s history/visibility and that the other participant’s messages are unaffected; do not promise permanent dismissal or recoverable old history. | Accurate lifecycle controls | P, E | AC-009, AC-012 |
| FR-010 | Functional | Must | Report an accessible message with a selected supported reason and optional explanation through com.atproto.moderation.createReport to Bluesky’s moderation audience, using the authenticated messageRef (sender DID, convoId, messageId), after showing the destination/evidence scope; display accepted, failed, or uncertain outcome accurately. | Selected safety path | U1 | AC-010 |
| FR-011 | Functional | Must | Read and explicitly update the shared incoming-message policy: everyone, people I follow, or nobody. Preserve it on enable/disable and preserve unrelated declaration fields (including allowGroupInvites and future fields) on update. Read the authoritative self record and use conditional create/update; on concurrent changes refetch and merge only the intended field. | Shared privacy control | U4 | AC-011 |
| FR-012 | Functional | Must | Persist account-scoped sync progress and durable event deduplication; make progress and resulting delivery work crash-consistent. Seed a new authorization baseline separately from normal resume; never replace a valid saved cursor with a fresh head on transient failure. Preserve unknown event envelopes/revisions so decoder gaps cannot stall progress. Recover interrupted polling or invalid cursors by reconciling authoritative state under the confirmed notification policies. | Durable detection | P, E | AC-012, AC-013 |
| FR-013 | Functional | Must | Schedule eligible incoming direct-message push with current authorization, subscription, read/mute, block, leave, and preference checks at dispatch. Notify once per new request until accepted; leaving/reinstall/reconnect does not reset this. After acceptance, permit another request notification only for a genuine new request transition. On outage/reauth recovery send at most one catch-up alert per eligible unread accepted conversation, without replaying individual missed messages or pre-enable history. Never notify owner-sent messages. | Relevant bounded notifications | P, U2, GQ-03, GQ-21, GQ-22 | AC-013, AC-014, AC-015, AC-029 |
| FR-014 | Functional | Must | Provide one account-wide DM push toggle, enabled by default for a newly enabled account and visible only while messaging is enabled, subject to device OS permission and active subscriptions. Support account-safe conversation notification navigation across cold start, foreground, logout, and account switch. | Notification control | P, C, GQ-02, GQ-13 | AC-015, AC-016 |
| FR-015 | Functional | Must | Refresh open inbox/conversation views after changes, on foreground/resume, and after reconnect; show stale/degraded state and provide retry rather than clearing previously loaded content on transient failure. | Consistent active clients | P | AC-012, AC-017 |
| FR-016 | Functional | Must | Enable/disconnect messaging account-wide; disconnect stops new chat work and cancels pending chat delivery across devices. Ordinary logout revokes that device’s access/subscription while other valid sessions continue; sign-out-all or loss of the last usable session stops background access. Preserve existing session expiry without silent extension, explain reauthentication/paused notifications, and fence stale work/private UI. Retain hidden local drafts on disconnect as FR-019 requires. | Revocable access with device-safe logout | P, C, GQ-01, GQ-16, GQ-17 | AC-014, AC-016, AC-028 |
| FR-017 | Functional | Must | In accepted-conversation message bubbles, attempt one compact preview for the first supported CraftSky link: post, profile, or business event. If that target is missing, restricted, or fails, show no card and do not try later links. A focused reply link previews that reply and opens its thread focused on it. Cards use identifying text, short excerpt/event details, and a permitted thumbnail; no inline playback or interactive nested post. No composer/request cards. Preserve every URL and the message regardless of preview outcome. | Useful bounded CraftSky previews | U5, GQ-10, GQ-11, GQ-18, GQ-19, GQ-23, SIM-03 | AC-025 |
| FR-018 | Functional | Must | Provide a New message action with CraftSky profile search and availability-aware recipient selection, alongside existing profile and inbox entry points. Do not add global Bluesky handle search or require CraftSky membership for existing external conversations. | Start conversations from the inbox | GQ-07 | AC-027 |
| FR-019 | Functional | Must | Preserve local unsent drafts per account/conversation across navigation and app restart, without cross-device synchronization. Clear on confirmed send or explicit discard; logout removes that account’s drafts on that device. Disable messaging retains drafts hidden until re-enabled and explains this behavior. Other accounts must never see them. | Preserve the owner’s unsent work | GQ-08, GQ-17, GQ-24 | AC-028 |
| RULE-001 | Business rule | Must | Either-direction CraftSky blocks disable new conversation creation and sending and suppress CraftSky DM push; retain upstream-accessible history. Upstream restrictions may further deny access. | Confirmed block policy | U2 | AC-018 |
| RULE-002 | Business rule | Must | Request acceptance, mute, leave, read state, and incoming-message policy use upstream semantics and may affect other clients; nobody/following availability must not be described as erasing existing conversations. | Avoid false privacy guarantees | P, U4, E | AC-008, AC-009, AC-011 |
| RULE-003 | Business rule | Must | Visible pushes contain sender display name (handle fallback, generic fallback if unavailable) and generic message copy only; message text, report prose, and embed previews are excluded from all provider payload fields. | Selected notification privacy | U3 | AC-015 |
| RULE-004 | Business rule | Must | Use explicit implemented RPC permissions bound to the chat audience, plus the fixed moderation audience for reporting and create/update-only declaration permission. Evaluate RPC authority semantically by method and audience, including equivalent combined/normalized scope forms. Request this narrow bundle in consumer login/registration while validating ordinary access against base permissions independently, so base-only grants and sessions remain valid. OAuth permission and explicit CraftSky enablement remain separate. Do not request Full Chat Client, wildcard chat methods, or unrelated export/group/account-delete authority. | Least authority | P | AC-019 |
| RULE-005 | Business rule | Must | CraftSky profile mute affects social content, not DM eligibility. Shared conversation mute suppresses DM push without marking it read; badge filtering follows FR-007’s upstream aggregates. Blocks retain RULE-001. | Separate social and chat controls | GQ-04, GQ-05, SIM-01 | AC-008, AC-015 |
| NFR-001 | Non-functional | Must | Flutter uses only authenticated CraftSky APIs; OAuth tokens and DPoP keys remain server-side. Enforce owner, membership, lifecycle, and capability checks for every request, queued operation, and private destination resolution. | Existing authority boundary | P, C | AC-016, AC-019 |
| NFR-002 | Non-functional | Must | Keep message/draft bodies, private routing, report evidence, credentials, and raw upstream payloads out of diagnostics, analytics, public records, and social notification storage. Persist no chat history archive; local unsent drafts are the explicit FR-019 exception. Minimize operational metadata under documented retention/cleanup that preserves required request deduplication and uncertain-send recovery across disconnect. Account deletion removes CraftSky private state without deleting upstream chat. | Private data minimization | P, C, GQ-08, GQ-17, GQ-21 | AC-020, AC-028 |
| NFR-003 | Non-functional | Must | New consumer APIs follow `/v1/`, bearer/device authentication, camelCase JSON, `{error, message, requestId}` failures, opaque bounded pagination, and a fixed upstream operation allowlist. | Existing API contract | C | AC-021 |
| NFR-004 | Non-functional | Must | Polling and request handling have bounded concurrency, timeouts, payloads, and retries with backoff; honor upstream rate limits and isolate per-account failures without starving other accounts. | Shared service resilience | P, C | AC-017, AC-022 |
| NFR-005 | Non-functional | Must | With a healthy provider and active authorization, use a 4-second foreground polling interval while a conversation is open, matching the audited Bluesky client cadence. Share account polling across active views/devices; rate-limit backoff, in-flight request exclusion, and lifecycle fencing take precedence. Continue bounded background polling for authorized accounts. No message-arrival latency target, latency measurement, or performance benchmark is required. | Confirmed polling cadence | GQ-06, SIM-02 | AC-022 |
| NFR-006 | Non-functional | Must | Support iOS, Android, and web inbox flows with existing localization, accessibility, loading/error, and keyboard conventions. Provide mobile background push on iOS/Android and foreground web updates; web push is outside v1. | Supported product platforms | C, GQ-15 | AC-023 |
| NFR-007 | Non-functional | Must | Validate required granular permissions and chat/report/settings/sync behavior through the actual CraftSky integration as part of full implementation. Check deployed operation support and account eligibility rather than treating published schemas as live proof. Record material contract differences and resolve them without silently widening authority or dropping required behavior. No separate feasibility spike or pre-design live-service gate applies. | Resolve external-service risks within implementation | P, SIM-02 | AC-024 |
| NFR-008 | Non-functional | Must | Use `flutter_chat_ui` for the conversation interface with CraftSky adapters/custom cards as needed. Validate platform, license, SDK/dependency, styling, accessibility, pagination, custom-preview and message-state fit during coding planning. Keep AppView transport and credentials server-side; bring concrete incompatibilities back for review before substituting another package or a fully custom interface. | Confirmed package choice | U6, GQ-12 | AC-026 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002 | Given existing direct conversations started in both CraftSky and another client, including a non-CraftSky correspondent, when the owner opens and paginates the inbox, then all supported conversations are reachable with correct identities/state and no duplication or premature end caused by intervening group conversations. Use listConvos(kind=direct,status=accepted) and listConvoRequests; request responses may contain group union variants. Exhaust valid cursors even across filtered-empty pages. Groups produce no v1 messages, badge contribution, or push. |
| AC-002 | BR-002, BR-003, FR-001 | Given messaging is disabled, when the owner enables it, then Direct messages settings disclose shared history/settings, Bluesky-hosted storage and Bluesky-only DM reporting before explicit CraftSky enablement. Login/registration already request the narrow messaging permissions; missing authority has explicit recovery that does not auto-enable messaging. Enablement and any accompanying messaging explanation make no promise of a CraftSky-only inbox/privacy boundary, PDS message portability or end-to-end encryption. Verify the meaning of the displayed copy, including localized variants, rather than rejecting an accurate statement that explicitly denies such a promise. Cancelled/refused/insufficient grants leave ordinary CraftSky usable and chat disabled or authorization-needed. Full valid grants produce ready/syncing state; a service outage produces an unavailable state rather than an empty-success inbox. Valid OAuth plus getStatus.chatDisabled=true produces a service-restricted state; generic status does not fabricate a reason or offer CraftSky-only verification. |
| AC-003 | BR-001, FR-002, FR-008 | Given both requests and accepted direct conversations, when the inbox is loaded or a request is opened, then they are distinguishable, unread state is visible, and reading the request alone does not silently accept it. The composer remains disabled until explicit acceptance succeeds; failure leaves it disabled. Upstream implicit acceptance on send is not used as a shortcut. |
| AC-004 | FR-003 | Given multiple history pages with deleted messages, embeds/replies/reactions, or an unknown variant, when the owner loads older history, then text is ordered correctly with stable scroll position, pagination continues while a cursor exists even if a page is short/empty after filtering, duplicates are omitted, and missing/unsupported content uses a safe placeholder without crashing or exposing raw payloads. |
| AC-005 | BR-001, FR-004 | Given a profile, when availability is checked then no conversation is created; when Start message is explicitly selected and permitted, then the canonical direct conversation opens. Repeated starts reuse it. Self-targeting, extra participants, blocks, disabled incoming messages, and unavailable recipients cause no unintended conversation or send. |
| AC-006 | FR-005 | Given plain text including emoji/combining marks and multibyte characters, when submitted, then both layers enforce nonblank, 1,000-grapheme and 10,000-byte bounds. Invalid input makes no upstream send and is retained for correction; valid input is not truncated and is shown as confirmed only after an authoritative result. |
| AC-007 | FR-006 | Given a send operation, when it is double-submitted, retried after reconnect, or resumed after an AppView crash, then the same operation does not issue an additional upstream send. A timeout after possible acceptance displays uncertain status and refreshes history without automatic replay; identical message text alone does not prove delivery. If history refresh still cannot establish the outcome, retain the text and offer Send again with an explicit duplicate-risk explanation; only the owner’s confirmation creates a new send operation. |
| AC-008 | FR-007, RULE-002, RULE-005 | Given unread messages, viewing through a known foreground message advances read state only through it; later messages remain unread and remote reads reconcile. Row counts show upstream unread messages. Both badges use getUnreadCounts(includeGroupChats=false), including its accepted/request buckets and service exclusions, without list-derived alternatives. Display 100 as 99+ and refresh capped totals rather than treating them as exact during optimistic changes. Reading a request refreshes its upstream count without accepting/removing it. Conversation mute refreshes the aggregates without marking read; profile mute alone changes neither DM badge nor read state. |
| AC-009 | FR-008, FR-009, RULE-002 | Given an accessible direct conversation/request, when accept, mute, unmute, or confirmed leave succeeds, then the next authoritative view and delivery eligibility reflect the result. Failed actions do not show confirmed success. Leave confirmation describes the verified owner-history effect; cancelling has no effect. Successful leave cancels pending CraftSky push without deleting the correspondent’s history and does not promise permanently preventing future contact. |
| AC-010 | BR-003, FR-010 | Given an accessible message and supported reason, when reporting is confirmed, then the pre-submit disclosure names Bluesky as the report recipient and explains the evidence scope without suggesting CraftSky moderators receive it. Submit reasonType, optional reason (within 2,000 graphemes/20,000 UTF-8 bytes), and an open-union chat.bsky.convo.defs#messageRef derived from an authorized message view to the fixed Bluesky moderation proxy. Do not attach a copied transcript or use moderator-only context APIs. The report audience is distinct from chat, and only the required reference/evidence is submitted; CraftSky moderators receive no report. Success is shown only on acceptance. Failure/uncertainty preserves a recoverable state and does not silently switch destinations or blindly resubmit. |
| AC-011 | FR-011, RULE-002 | Given an existing incoming-message setting and unrelated declaration fields, when messaging is enabled/disabled then the declaration is unchanged. When the owner explicitly chooses everyone/following/nobody, then the UI explains cross-app effect, the self record is read and conditionally updated with only allowIncoming changed; allowGroupInvites and unknown fields survive. A missing record uses an absence precondition, and concurrent writes refetch/merge rather than overwrite. Verified PDS readback confirms persistence; shared-service propagation may be eventual and must not be described as instantly disabling all existing conversations. External changes appear on refresh; failure never reports a successful preference change. |
| AC-012 | FR-007, FR-009, FR-012, FR-015 | Given message, read, request, mute, leave, delete, unknown, and group events, when polling repeats/reorders pages or either client reconnects, then supported views reconcile to authoritative state, deletes do not resurrect cached text, unsupported events do not crash or stall synchronization, and cursor progress is not committed ahead of required durable processing. |
| AC-013 | FR-012, FR-013 | Given eligible incoming events and worker crashes before/after processing/cursor persistence, recovery produces durable deduplicated logical notifications and at most one delivery job per logical notification/subscription. Normal accepted messages dedupe by account/message; request and catch-up alerts follow AC-029 rather than creating one job per replayed message. First enablement baselines prior history without push; recovery reconciles counts and applies catch-up rules. Transport/provider redelivery is not claimed exactly once. |
| AC-014 | BR-002, FR-001, FR-013, FR-016 | Given two valid device sessions and queued chat work, when one device logs out, only its access/delivery is revoked and the other continues. When messaging is disconnected account-wide, sign-out-all occurs, the last usable session expires/revokes, or owner deletion begins, no new unauthorized work is admitted and stale jobs are fenced/cancelled. Existing expiry is preserved; UI explains paused notifications/reauthentication without breaking unrelated authorized CraftSky use. Already-issued network requests are not claimed retractable. |
| AC-015 | FR-013, FR-014, RULE-003, RULE-005 | Given incoming messages with private-text canaries, when generating and dispatching pushes, then sender identity and generic copy are the only visible content; no provider field includes the canaries. Current read, mute, block, leave, disabled push, and inactive subscription states suppress dispatch. Owner-sent messages never notify the owner. The DM toggle starts enabled after messaging enablement, is shown during enablement, and applies account-wide while respecting each device’s OS permission. Changing it affects CraftSky delivery without changing Bluesky preferences. Profile mute alone does not suppress DM push; conversation mute and blocks do. |
| AC-016 | FR-014, FR-016, NFR-001 | Given a notification for account A, when opened after cold start, logout, account switch to B, membership loss, or chat disconnect, then no A content appears under B. Route only after the correct owner is authenticated and permitted; otherwise show a safe unavailable/authorization destination. Guessed conversation/message identifiers cannot read, send, mark read, mutate, or report across owners. |
| AC-017 | FR-001, FR-015, NFR-004 | Given offline state, revoked/expired credentials, upstream 429/5xx, or a timed-out call, when requests/polls fail, then show classified recovery state, retain already displayed content only within the current authenticated owner session, retry only safe operations with bounded backoff, and continue processing healthy accounts. Backgrounding does not mark unread messages read. |
| AC-018 | RULE-001 | Given an indexed block in either direction or a newly confirmed owner block, when creating/sending or dispatching pending push, then server enforcement rejects/suppresses the action and the UI disables sending. Existing upstream-accessible history and reporting remain available. Upstream history denial produces an unavailable state. Unblock causes fresh availability evaluation rather than unconditional permission to send. |
| AC-019 | RULE-004, NFR-001 | Given login, refresh, reauthorization, and multiple device sessions, when chat capability is evaluated, then only sessions with the exact effective required permissions can perform the operation or supply a background worker. No token/DPoP key reaches Flutter. Scope inspection confirms fixed audiences and implemented operations, with no full chat/group/export/account-delete authority. Equivalent method/audience grants expressed as combined lxm parameters or reordered/encoded parameters are recognized without accepting wildcard/unexpected authority. Advertising optional chat scopes in metadata does not make them mandatory for base login/refresh, and selecting a background session verifies its needed capability. Missing reporting/declaration permission fails the relevant capability gate rather than invalidating otherwise valid ordinary CraftSky access. |
| AC-020 | NFR-002 | Given message/report-content and credential canaries across successful and failed chat operations, when local logs, serialized error sinks, telemetry, persisted command results, notification rows, and public writes are inspected, then canaries are absent except from the intended authenticated chat response/report destination and explicitly permitted account-isolated local draft storage. Draft content never reaches AppView storage or telemetry. Private routing identifiers are absent from diagnostics/public writes and restricted to necessary owner-scoped operational storage and opaque push routing. Reviewed bounded diagnostic fields remain present. Disconnect retains only required minimal request-notification/uncertain-send metadata and hidden local drafts under the documented policy; deletion cleans up CraftSky private state. Neither flow deletes upstream chat or unrelated namespace records. |
| AC-021 | NFR-003 | Given every new chat route, when valid, invalid, unauthorized, and paginated requests are exercised, then responses follow the existing consumer API conventions. Cursor/identifier tampering cannot change owner/service. Client-supplied arbitrary RPC names, audiences, URLs, and raw upstream bodies are rejected. |
| AC-022 | NFR-004, NFR-005 | Given an authorized owner with an open conversation and healthy completed requests, verify the configured 4-second foreground poll cadence shared across that account’s active views/devices. Given 429, slow/in-flight calls, or lifecycle loss, verify bounded backoff, non-overlap, and fencing, with continued healthy-account progress. Verify background polling remains enabled for eligible accounts when no foreground view is open. These are configuration and functional scheduler checks, not message-arrival latency measurements or performance benchmarks. |
| AC-023 | NFR-006 | Given each supported platform, when using enablement, inbox, requests, profile entry, conversation, reporting, preferences, and recovery flows, then keyboard/screen-reader labels, focus, localized copy, and narrow/wide layouts are usable. Verify mobile push opens on iOS/Android; verify web foreground refresh without assuming web push. |
| AC-024 | NFR-007 | During full implementation, use controlled consenting test accounts through the actual CraftSky/AppView path to validate granular authorization and refresh, PDS chat/report routing, account eligibility, direct messaging and request/read/mute/leave behavior, shared declaration updates, and background log detection/recovery while the app is closed with a valid session. Verify the status/request/count operations and confirm service-dependent leave/restart and cursor behavior against the related functional criteria. Record and resolve material differences without broad-scope fallback or dropping safety/control features. This is ordinary functional integration validation, not a separate spike, pre-design gate, or latency/capacity benchmark. |
| AC-025 | FR-017 | Given accepted direct messages with post, focused-reply, profile, event, unsupported, or multiple links, attempt only the first supported CraftSky link and show at most one compact card with permitted details and canonical navigation. A reply card previews and focuses the reply. If the first supported target is inaccessible, removed, or fails—even when a later link is accessible—show no card and perform no fallback lookup of later links. Preserve every text URL and do not block chat. No cards or preview resolution occur in the composer or unaccepted requests; acceptance makes existing messages eligible. No autoplay/nested controls, arbitrary URL fetches, third-party private-text disclosure, stale cross-account results, or preview data in push/diagnostics. |
| AC-026 | NFR-008 | Given coding planning, record `flutter_chat_ui` as the selected conversation UI package and validate its SDK/dependencies, license, iOS/Android/web behavior, CraftSky styling/accessibility, history pagination, pending/failed/uncertain sends, local drafts, custom cards, and disabled link previews in the composer and unaccepted requests. Retain AppView-mediated transport with no client OAuth credentials or separate messaging backend. Report concrete incompatibilities for review rather than silently changing the selected approach. Dependency resolution passed in an isolated dry run; no repository dependency installation, feature compilation, or platform build validation is claimed. |
| AC-027 | FR-018 | Given the inbox, New message opens CraftSky profile search; selecting an available recipient retrieves/creates the canonical direct conversation only after explicit start. Unavailable/blocked recipients cannot be sent to, search does not add global Bluesky lookup, and existing external conversations/profile entry remain usable. |
| AC-028 | FR-005, FR-016, FR-019, NFR-002 | Given a typed local draft, navigation/restart restores it only for the same account/conversation; confirmed send or explicit discard clears it. Device logout removes that account’s local drafts without removing other accounts’ drafts. Messaging disconnect retains them hidden until re-enabled and explains this distinction. Offline Send retains the draft with connection feedback; reconnect/restart never auto-sends. Local draft storage is private and never copied into logs, push, or analytics, and is never synchronized to AppView as a draft. Explicit Send uses the normal authenticated message path. |
| AC-029 | FR-013 | Given a genuinely new eligible unread request, schedule one request alert, then suppress further alerts while it remains unaccepted. Leave/reappearance, reinstall, disconnect/reconnect, and worker replay do not reset eligibility. Only verified acceptance followed by a genuinely new request transition permits one new request alert. Initial enablement baselines existing requests/history without push. After an outage/reauthentication, coalesce still-eligible unread accepted messages into at most one catch-up alert per conversation/recovery, with no per-message replay or re-alert of already-notified message state; apply current dispatch checks and preserve the separate once-per-request rule. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Provider accepted a message but response was lost | Uncertain state; no blind resend or text-only delivery matching | FR-005, FR-006 |
| EC-002 | Several sessions have different effective grants | Select only current, chat-capable sessions for the owner; never borrow another account's authority | FR-016, NFR-001 |
| EC-003 | Read/mute/block/leave changes while push is queued | Re-evaluate immediately before dispatch; reconcile remote state; acknowledge the unavoidable race after final check | FR-013, RULE-001 |
| EC-004 | Conversation or cursor no longer accessible | Reconcile state, preserve request dedupe, and coalesce eligible accepted-conversation catch-up without replay floods or stale disclosure | FR-012, FR-015 |
| EC-005 | Existing group, embed, reaction, or future event type | Exclude groups; render supported direct text with safe placeholders; advance safely past irrelevant events | FR-002, FR-003, FR-012 |
| EC-006 | No cached CraftSky profile for a correspondent | Use upstream identity and handle/generic fallback; no CraftSky membership requirement | BR-001, RULE-003 |
| EC-007 | Blocked participant history is unavailable upstream | Explain unavailability; do not bypass restrictions or archive history to defeat them | RULE-001 |
| EC-008 | User changes the shared incoming policy in another app | Refresh authoritative policy; explain scope; do not overwrite unrelated declaration fields | FR-011 |
| EC-009 | Sender name missing or changes before push | Use current safe display identity or generic copy; never substitute private message text | RULE-003 |
| EC-010 | Leave followed by a later incoming message/recreated direct view | Follow verified upstream semantics; do not promise permanent blocking from leave; re-evaluate eligibility | FR-009, FR-013 |
| EC-011 | First supported CraftSky preview target is removed, restricted, slow, or resolves after an account switch | Keep original links without trying later candidates; cancel/discard stale results and apply current-viewer access/moderation rules (AC-025) | FR-017 |

## 15. Data / Persistence Impact

- New private state is expected for owner enablement/capability status, account sync cursor/baseline, bounded deduplication metadata, send-operation identity/outcome, DM push preferences, and durable notification routing/delivery, including request-alert lifetime and coalesced recovery state. Exact tables and retention periods belong to the implementation design. Request-notification state must survive reconnect/reinstall and cannot be discarded in a way that resets GQ-21; uncertain-send operation state must likewise prevent unsafe replay.
- Existing OAuth/session and owner-lifecycle models must accommodate optional effective chat capability. Existing notification delivery must support a private conversation destination without copying message bodies into social notification rows.
- No authoritative message-history database, public message records, or persisted report evidence for CraftSky moderators. Received/sent history is transient authenticated response content. Local unsent drafts are the explicit exception: persist per account/conversation on the device only, hidden on messaging disable and removed for that account on device logout. They are not an AppView message archive.
- A migration is expected for operational state; none is designed or run here. NFR-002 covers documented retention/cleanup; implementation planning determines periods compatible with deduplication and cursor replay.
- No shipped-client backwards compatibility is required by AGENTS.md, but production data, credentials, and infrastructure controls remain applicable. This document authorizes no production migration, consent change, or Blueprint mutation.

## 16. UI / API / CLI Impact

- UI: account-wide messaging enable/disconnect flow; direct inbox/requests; New message with CraftSky profile search; flutter_chat_ui conversation/composer; local drafts and explicit uncertain-send recovery; profile Message action; separate main/request unread indicators; accept/mute/leave/report controls; shared incoming policy; DM push toggle; authorization/degraded/uncertain/unavailable states; conversation notification destination. Exact navigation placement is a design detail.
- API: retain the brief's candidate surface below as proposed, not an approved route/body contract. Preference/enablement lifecycle operations are additional and may extend existing settings/auth routes; their design is not constrained to twelve total operations.

```text
GET    /v1/chat/conversations
POST   /v1/chat/conversations
GET    /v1/chat/conversations/{id}
GET    /v1/chat/conversations/{id}/messages
POST   /v1/chat/conversations/{id}/messages
PUT    /v1/chat/conversations/{id}/read
POST   /v1/chat/conversations/{id}/accept
PUT    /v1/chat/conversations/{id}/mute
DELETE /v1/chat/conversations/{id}/mute
POST   /v1/chat/conversations/{id}/leave
POST   /v1/chat/reports
GET    /v1/chat/status
```

- Availability may extend profile data or use a separate capability query, but must not create conversations. `status` conveys capability/sync health, not raw upstream credentials/errors. IDs are opaque upstream chat identifiers, not AT-URIs.
- Chat RPC allowlist: `listConvos`, `listConvoRequests`, `getConvo`, `getMessages`, `getConvoAvailability`, `getConvoForMembers`, `getUnreadCounts`, `sendMessage`, `updateRead`, `acceptConvo`, `muteConvo`, `unmuteConvo`, `leaveConvo`, and `getLog` under `chat.bsky.convo`, plus `chat.bsky.actor.getStatus`. These 15 methods map to requirements in section 24; deployed availability and effective grant acceptance are validated during full implementation (NFR-007/AC-024). Example: `rpc:chat.bsky.convo.sendMessage?aud=did:web:api.bsky.chat%23bsky_chat`.
- Incoming-policy writes need limited create/update authority for `chat.bsky.actor.declaration` and a safe read/modify/write contract. Reporting uses `com.atproto.moderation.createReport`, an open-union `chat.bsky.convo.defs#messageRef`, and Bluesky’s fixed `did:plc:ar7c4by46qjdydhdevvrndac#atproto_labeler` proxy/audience. That contract is verified in official client/PDS source; live acceptance remains to be tested. It is not part of the chat audience/permission set. No local Lexicon modification is proposed. Any later change under `lexicon/` invokes the repository's skill and ADR requirements.
- CLI: no new user-facing command required.
- Background jobs: account-scoped durable polling and reconciliation feeding existing delivery infrastructure. Foreground update transport is an implementation choice; no separate realtime-service requirement is introduced.
- Link previews: recognize supported canonical CraftSky content links and reuse existing content resolution/navigation where possible. Do not require changes to the upstream message format or a general external-link scraping service. Supported targets are posts/focused replies, profiles, and business events. FR-017/AC-025 define the first-supported-link limit and fallback behavior; composer and unaccepted-request previews remain excluded.
- Flutter dependencies: use the selected `flutter_chat_ui` conversation UI with custom CraftSky cards/adapters and existing state/repository boundaries. Validate compatibility before installation/implementation; no dependency file is changed here.

## 17. Security / Privacy / Permissions

- Authentication: existing CraftSky consumer sessions and device headers; existing server-side DPoP/PDS discovery and refresh. No chat service token exception for Flutter.
- Authorization: current owner lifecycle, messaging enablement, effective operation permissions, conversation membership, block state, and notification subscription. Validate opaque references at the server boundary. Use existing lifecycle coordination for stale workers and session replacement.
- Sensitive data: messages and report prose are private even when sender identities also have public profiles. Conversation membership, IDs, cursors, recipient routes, and operation data must not be treated as public log context. Sender names in push are the explicit U3 exception for delivery, not diagnostic permission.
- Shared settings: clearly distinguish public PDS declaration records from private messages. The incoming-policy declaration is shared/public metadata; do not encode private DM content in it. “People I follow” means the upstream Bluesky follow graph used by the service, not a new CraftSky-only interpretation.
- Abuse cases: forged conversation IDs, cross-account notification opens, sending during a block/permission race, report evidence for inaccessible messages, generic upstream proxying, malicious upstream text/URLs, and uncontrolled polling. CraftSky previews resolve only recognized content routes through existing access/moderation controls; arbitrary external URL fetching and unrestricted RPC forwarding are excluded. Do not send private message text or conversation context to third-party preview services.
- CraftSky disconnect stops background access/delivery while retaining only the private metadata needed for safe re-enable/deduplication and hidden local drafts under FR-019. Account deletion removes CraftSky private state under the approved retention policy. Ordinary device logout clears that device’s account drafts/access, without disabling other valid devices. Sign-out-all stops background access; clients clear affected private state when they process revocation. It must not delete the Bluesky chat account, messages, declaration, or records in other namespaces.

## 18. Observability

- Events: bounded operation outcomes for enablement, authorization-needed, sync/recovery, send uncertainty, reporting result, and delivery suppression. Never record message/report text, private counterpart routing, or raw upstream payloads.
- Logs: follow `docs/development/logging-error-reporting.md`; preserve typed causes internally, choose private context at source, retain only permitted operational references, and keep one issue owner. Do not emit upstream error prose containing private data.
- Operational diagnostics supporting NFR-002/NFR-004 may include aggregate poll outcomes/rate limits, active authorized polling accounts, queued/failed delivery, recovery counts, and uncertain sends. Avoid DID, conversation, message, cursor, and device labels. No latency measurement or benchmark requirement is introduced.
- Alerts: implementation may use existing operational monitoring for sustained sync failures or delivery backlogs. Expected per-account reauthorization is a user recovery state, not an unbounded operator issue stream.
- Verification: inspect serialized sinks with positive expected fields and protected-value canaries. Transport acceptance, a mock event, or HTTP 200 alone does not prove real message delivery or notification navigation.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Granular permissions or PDS/provider behavior differ from published contracts | Feature cannot work with selected authority | Validate contracts during full integration (NFR-007); return for scope review rather than use broad fallback |
| RISK-002 | Provider accepts a send while its response is lost | Duplicate messages or false success | Stable operation identity, durable outcome, uncertain UI, no automatic replay |
| RISK-003 | Cursor/baseline/replay behavior is misunderstood | Lost incoming events or notification floods | Crash-consistent progress, dedupe, authoritative reconciliation, explicit baseline tests |
| RISK-004 | Read/mute/block state changes outside CraftSky or after final dispatch check | Stale notification or attempted send | Upstream enforcement plus CraftSky checks; refresh before dispatch; document residual in-flight race |
| RISK-005 | Private content enters diagnostics, push, reports, or another account's UI | Privacy breach | Minimal persistence, explicit Bluesky report boundary, sink canaries, owner fencing and account-switch tests |
| RISK-006 | Shared chat differs from broad product portability/moderation promises | Misleading consent or expectations | Visible hosted/shared-service disclosure; explicit DM qualification of product principles |
| RISK-007 | Per-account polling exceeds service limits or singleton capacity | Lag, rate limiting, backend contention | Bounded fair scheduling, concurrency, and backoff under NFR-004/NFR-005; respect provider rate limits |
| RISK-008 | Missing or partial upstream capability includes reporting/preferences | Incomplete safety/control surface | Treat required capabilities as release blockers; do not silently ship without them |
| RISK-009 | Upstream schema/service evolution introduces groups and new variants | Crashes, leaked unsupported notifications, stalled cursors | Direct-kind filtering, tolerant variants, contract pinning and fixture coverage |

## 20. Assumptions

The original assumption IDs are preserved for traceability. ASM-001–ASM-005 are resolved by confirmed grilling decisions and no longer authorize alternative behavior.

| ID | Assumption / resolution | Impact If Wrong |
|---|---|---|
| ASM-001 | Resolved by GQ-02, GQ-03, GQ-13, GQ-21, GQ-22: account-wide push enabled by default, once-per-request alerts, no pre-enable backlog, coalesced accepted-conversation catch-up. See FR-013/FR-014. | A change requires an explicit product revision |
| ASM-002 | Resolved by SIM-01, superseding GQ-05/GQ-14 where needed: both badges use native upstream direct-conversation aggregates and filtering/caps (FR-007). | A change requires an explicit product revision |
| ASM-003 | Resolved by updated GQ-06/SIM-02: retain four-second foreground polling (NFR-005); no arrival-latency target, measurement, or performance benchmark. | Cadence changes require review; backoff remains applicable |
| ASM-004 | Resolved by GQ-15: iOS/Android background push; web inbox/foreground updates without web push. | Adding web push expands v1 scope |
| ASM-005 | Resolved by GQ-07/GQ-08/GQ-17/GQ-24: New message with CraftSky profile search; local drafts retained on disable, removed on logout; no global Bluesky search, offline history, draft sync, or auto-send queue. | A change requires an explicit product revision |
| ASM-006 | Remaining feasibility assumption: compatible accounts can obtain required granular chat/report/declaration permissions and satisfy service eligibility. Validate provider/account prerequisites during full implementation (NFR-007); unsupported accounts receive clear unavailable/reauthentication guidance. | Selected integration may require revision |

## 21. Open Questions

No unanswered product question prevents the full integration effort. The following are implementation considerations, not a separate spike or prerequisites to beginning implementation:

- Validate live permission acceptance, reporting, eligibility, and service-dependent cursor/leave behavior during normal integration validation (NFR-007/AC-024). Section 24 records what is source-verified and what remains unproven; no broad-scope fallback is authorized.
- Choose operational metadata retention, deduplication/recovery storage, local draft storage, and cleanup details to satisfy FR-006, FR-012, FR-013, FR-019, and NFR-002.
- Settle navigation placement, route bodies/lifecycle routes, active-view transport, localization, compact card layout, and flutter_chat_ui adapters within the existing requirements. Package selection, preview targets/limits, and unread semantics are settled.
- ASM-006 remains an external-service assumption to validate during implementation. No authenticated live experiments have been performed, and the source audit does not claim working production integration.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer: Owner confirmed GQ-01–GQ-24 and the superseding SIM-01–SIM-04 decisions
Date: 2026-10-19 (initial draft 2026-10-18)
Notes: High risk reflects private messages, optional OAuth authority, shared settings, ambiguous writes, and background delivery. The document review returned Approved with notes; the owner then authorized requirements/test revisions addressing DR-001/DR-002. This revision clarifies consent acceptance assertions and test handoff without changing feature scope. The artifact retains its existing Draft workflow status; implementation has not started. DR-003/DR-004 remain coding-plan tasks. Section 24 remains reference evidence rather than an additional acceptance contract; no separate spike/pre-design live-service gate applies. No commit, application change, or production action was performed.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-18-direct-messaging/01-requirements.md`.
- Test specification: `02-acceptance-tests.md`, completed and revised after document review. The owner authorized revisions to requirements and acceptance tests for DR-001/DR-002; stable requirement/criterion IDs and feature scope are preserved.
- DR-001 is clarified in AC-002/AC-010: consent copy must avoid the prohibited privacy/portability claims and reporting must name Bluesky. DR-002 is addressed by scenario-specific verification assignments in the test specification; test-file layout is not a product requirement. DR-003/DR-004 remain coding-plan tasks for retention/recovery design and controlled live evidence. The existing `03-document-review.md` records the review before these revisions; it is not rewritten by this revision.
- Must-cover requirement IDs: BR-001–BR-003; FR-001–FR-019; RULE-001–RULE-005; NFR-001–NFR-008. Sections 8, 12, and 13 are authoritative; reference sections add no separate gates.
- Suggested test levels: unit tests for capability evaluation, upstream-count mapping, eligibility, and validation; PostgreSQL integration tests for durable progress/deduplication/operation/lifecycle fencing; upstream fault/contract tests; Flutter widget/routing tests; iOS/Android/web checks; controlled live functional integration and mobile push-open checks.
- Derive coverage from AC-001–AC-029, including Unicode limits, ambiguous sends, crash/replay windows, optional/multi-session grants, account-safe navigation, remote state changes, draft cleanup, request/catch-up policies, and serialized privacy canaries. Mocks/local tests do not prove real service/device behavior.
- Test first-supported-link previews, including an inaccessible first target followed by an accessible later link with no fallback lookup. Cover account-switch races and absence of composer/request preview fetches or private content in diagnostics/push.
- AC-024 is functional validation within full implementation. No separate integration spike, pre-design live-service gate, latency target/measurement, or performance benchmark is required. Preserve the four-second polling configuration and backoff checks in AC-022.

## 24. Feasibility Audit (2026-10-19)

Reference evidence only: this audit supports the requirements in sections 12–13 and creates no separate implementation gate. SIM-01–SIM-04 supersede earlier policy/gating interpretations.

### Verdict and verification boundary

**The proposed one-to-one integration is feasible at the published API, existing Go transport, and Flutter dependency-resolution levels. It is not yet proven against authenticated live chat/moderation services.** No confirmed product decision needs to be abandoned. The audit corrected missing RPCs, capped-count handling, report routing/evidence, optional-scope validation, and cursor/pagination assumptions. SIM-01 subsequently resolved the aggregate-filtering mismatch by adopting the upstream badge semantics directly (FR-007/AC-008).

There is no CraftSky DM implementation yet in the inspected paths. This audit assesses the planned implementation and existing integration foundations; it does not claim that an HTTP success, a schema, or dependency resolution proves working delivery.

Pinned source baselines:

- AT Protocol/Bluesky schemas and public PDS/OAuth implementation: [`9f66a7e54b030700c03eda0bdbf35ffe756f4cd8`](https://github.com/bluesky-social/atproto/commit/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8) (commit dated 2026-10-17). Recursive repository tree was complete, not truncated; all chat schema copies and the actual override targets were retrieved.
- Official Bluesky client: [`24e0ebbcd18b794934d617de20477c82a37ac2f0`](https://github.com/bluesky-social/social-app/commit/24e0ebbcd18b794934d617de20477c82a37ac2f0) (commit dated 2026-10-19). Inspected message event/conversation agents, query hooks, account declaration updates, service clients, and report submission.
- CraftSky Go dependency: Indigo `v0.0.0-20260903211445-41278964ec8e`, verified against the local module cache and `appview/go.mod`.
- Flutter registry: published `flutter_chat_ui 2.12.0`, `flutter_chat_core 2.9.1`, `cross_cache 1.1.1` metadata, plus the selected package’s builder APIs. See [package](https://pub.dev/packages/flutter_chat_ui/versions/2.12.0) and [custom builders](https://pub.dev/documentation/flutter_chat_core/latest/flutter_chat_core/Builders-class.html).

The Bluesky chat backend itself was not exercised or available as an inspected server implementation. Published schemas and official-client behavior can differ from deployed account/provider capabilities; NFR-007/AC-024 validate material differences during full implementation, without a separate spike or pre-design gate.

### Complete published chat RPC inventory

The pinned API namespace contains **50 RPC definitions**: 43 consumer-facing operations and 7 moderation/operator operations. Record/definition/embed schemas and the permission set are not extra RPCs. The seven moderator operations are not a consumer message stream or an ordinary reporting API.

| Namespace | Count | Operations reviewed | CraftSky v1 disposition |
|---|---|---|---|
| `chat.bsky.actor` | 3 | `deleteAccount`, `exportAccountData`, `getStatus` | Use getStatus; exclude account deletion/export. |
| `chat.bsky.convo` | 22 | `acceptConvo`, `addReaction`, `deleteMessageForSelf`, `getConvo`, `getConvoAvailability`, `getConvoForMembers`, `getConvoMembers`, `getLog`, `getMessages`, `getUnreadCounts`, `leaveConvo`, `listConvoRequests`, `listConvos`, `lockConvo`, `muteConvo`, `removeReaction`, `sendMessage`, `sendMessageBatch`, `unlockConvo`, `unmuteConvo`, `updateAllRead`, `updateRead` | Use the 14 mapped methods below; exclude reactions, per-message deletion, batch sends, global mark-read, lock/unlock, and extra member listing. |
| `chat.bsky.group` | 16 | `addMembers`, `approveJoinRequest`, `createGroup`, `createJoinLink`, `disableJoinLink`, `editGroup`, `editJoinLink`, `enableJoinLink`, `getJoinLinkPreviews`, `listJoinRequests`, `listMutualGroups`, `rejectJoinRequest`, `removeMembers`, `requestJoin`, `updateJoinRequestsRead`, `withdrawJoinRequest` | Exclude all group management/invitation/join-link operations. |
| `chat.bsky.moderation` | 7 | `getActorMetadata`, `getConvo`, `getConvoMembers`, `getConvos`, `getMessageContext`, `subscribeModEvents`, `updateActorAccess` | Exclude all operator RPCs, including subscribeModEvents; reporting uses the separate com.atproto method. |
| `chat.bsky.notification` | 2 | `getPreferences`, `putPreferences` | Exclude both: CraftSky controls its own push and does not modify Bluesky notification preferences. |

Sources: [chat schema directory](https://github.com/bluesky-social/atproto/tree/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky), [actual full permission definition](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/lexicons-overrides/chat/bsky/authFullChatClient.json), [actual leave definition](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/lexicons-overrides/chat/bsky/convo/leaveConvo.json). The package-level full-permission and leave files are repository links to these override files. The full permission manifest is not a complete inventory: at this revision it omits getUnreadCounts and notification preferences, and names convo.exportAccountData while the actual export RPC is actor.exportAccountData. This is another reason to enumerate required methods explicitly rather than infer scope from that manifest.

### Required method and permission mapping

Under RULE-004, each of the following 15 methods maps to `rpc:<full NSID>?aud=did:web:api.bsky.chat%23bsky_chat`. These are **upstream RPC operations**, not a requirement for fifteen Flutter-facing routes; existing proposed routes can aggregate them.

| Upstream method | Purpose / constraint | Requirements |
|---|---|---|
| `chat.bsky.actor.getStatus` | Detect viewer chatDisabled separately from OAuth success; ignore irrelevant group-create capabilities. | FR-001 |
| `chat.bsky.convo.listConvos` | Accepted inbox with kind=direct and status=accepted; opaque limit/cursor pagination. | FR-002 |
| `chat.bsky.convo.listConvoRequests` | Preferred request listing; union includes group-related request variants, so filter and continue pagination. | FR-002, FR-008 |
| `chat.bsky.convo.getConvo` | Current membership, per-viewer status/mute/unread state and dispatch reconciliation. | FR-003, FR-013 |
| `chat.bsky.convo.getMessages` | Paginated history; optional relatedProfiles, deleted/system variants; cursor—not item count—determines completion. | FR-003 |
| `chat.bsky.convo.getConvoAvailability` | CanChat and optional existing direct conversation without creation. | FR-004, FR-018 |
| `chat.bsky.convo.getConvoForMembers` | Explicit get/create of the canonical direct conversation; enforce one other DID in CraftSky despite broader upstream array bounds. | FR-004 |
| `chat.bsky.convo.getUnreadCounts` | includeGroupChats=false; accepted/request aggregates, each capped at sentinel 100 (=99+). | FR-007 |
| `chat.bsky.convo.sendMessage` | Plain text/facets only for v1; no caller idempotency field. | FR-005, FR-006 |
| `chat.bsky.convo.updateRead` | Supply last actually presented messageId; omitting it must not accidentally mark unseen arrivals read. | FR-007 |
| `chat.bsky.convo.acceptConvo` | Explicit accept; absent output rev can mean already accepted, not failure. | FR-008 |
| `chat.bsky.convo.muteConvo` | Shared chat mute, independent of CraftSky actor mute. | FR-009, RULE-005 |
| `chat.bsky.convo.unmuteConvo` | Shared unmute; refresh counts/eligibility. | FR-009 |
| `chat.bsky.convo.leaveConvo` | Changes leaving viewer’s enumeration state; does not remove direct membership or erase peer history. | FR-009 |
| `chat.bsky.convo.getLog` | Authenticated account event polling, including types outside v1; no normal-consumer subscription/webhook is in this inventory. | FR-012–FR-015 |

Additional authority:

```text
repo:chat.bsky.actor.declaration?action=create&action=update
rpc:com.atproto.moderation.createReport?aud=did:plc:ar7c4by46qjdydhdevvrndac%23atproto_labeler
```

These are in addition to `atproto` and the existing CraftSky scopes. The declaration is public PDS metadata at `chat.bsky.actor.declaration/self`; getRecord does not need a new private-read RPC scope. Existing PDS read/conditional-put support can preserve allowGroupInvites and unknown fields. Do not grant declaration delete, group/export/account-delete operations, notification preference writes, moderator APIs, or wildcard chat authority for this release.

The report is routed through the user’s PDS with the moderation service’s proxy header, not `bsky_chat`. Its subject is `{$type: "chat.bsky.convo.defs#messageRef", did: <message sender>, convoId, messageId}`; did is not the reporter by default. The createReport schema’s subject union is open, and the official client intentionally passes chat references through that extension. No full transcript upload or moderator getMessageContext permission is needed. Sources: [official report construction/routing](https://github.com/bluesky-social/social-app/blob/24e0ebbcd18b794934d617de20477c82a37ac2f0/src/components/moderation/ReportDialog/action.ts), [createReport schema](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/lexicons/com/atproto/moderation/createReport.json), [PDS permission enforcement](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/pds/src/api/com/atproto/moderation/createReport.ts), and [service directory](https://bsky.network/docs/api-directory/).

### Integration findings and corrections

| Finding | Evidence and implication | Disposition |
|---|---|---|
| Optional chat access needs a real OAuth capability model | CraftSky OAuth authorization/callback and session coordinator currently use one Config.Scopes baseline. effective_scopes.go semantically handles repo authority but accepts RPC requirements only by exact string match. The official RPC parser accepts multiple methods and normalized syntax. Advertising optional methods cannot make base sessions invalid, and equivalent granted RPC syntax must work. | RULE-004/AC-019 clarified; implementation must adapt metadata/PAR/callback/refresh/background capability selection together. |
| Chat transport fits existing server-side OAuth | Indigo ClientSession.APIClient plus WithService supports PDS proxying while retaining DPoP/refresh on the server. CraftSky’s federated.go already creates bounded session clients, but PDSClient currently exposes repository operations only. | Add a narrow chat/report adapter through the coordinated session and hardened boundary; do not expose a generic proxy or bypass owner fencing. |
| Current Go types have a gap | Pinned Indigo includes status/request methods but no generated getUnreadCounts helper. Its generated getLog union discards unknown variants, including their revision envelope. | Use reviewed typed transport views/adapters or a justified SDK update, and preserve raw event envelopes for forward-compatible progress. Existing generated types alone do not satisfy FR-012. |
| Aggregate badge filtering differs from the original grilling wording | getUnreadCounts excludes muted, locked, blocked, deleted/deactivated cases and caps both values at 100. | Resolved by SIM-01: use the provider’s direct accepted/request aggregates and 99+ cap display (FR-007/AC-008), without custom list reconciliation. |
| Polling is supported; foreground cadence aligned after audit | Official client uses 4-second active polling and preserves the previous cursor on transient resume. The owner selected that same cadence, replacing the earlier three-second arrival target. Baseline seeding uses a cursorless getLog call. There is no normal chat subscription endpoint among the 50 RPCs; moderation streaming is privileged. | NFR-004/NFR-005 define shared account polling, bounded concurrency, rate-limit backoff, and the four-second foreground cadence. SIM-02 removes arrival-latency targets/measurements and performance benchmarks. |
| Sending uncertainty is unavoidable upstream | sendMessage/messageInput have no client operation key. Indigo retries explicit 401 nonce/expiry challenges, not arbitrary network failures. The official client’s generic retry behavior is not an exactly-once guarantee. | Keep the stricter confirmed CraftSky uncertain-send policy and durable operation fence; verify the complete retry stack cannot replay an ambiguous send/report. |
| Listing, read, accept, and leave have subtle contracts | listConvos filters direct/accepted; dedicated request union can include groups; history may be short with a valid cursor. updateRead accepts messageId; accept output rev is optional. The PDS schema describes direct restart by a new message, while exact retained-history/request transitions remain service-dependent. | FR-002, FR-008, AC-001/004/024 clarified. Keep leave-state reconciliation and once-per-request memory across reappearance. No peer-facing Seen claim. |
| Shared policy update is feasible without expanding authority | Declaration uses allowIncoming=all/none/following, plus separate optional allowGroupInvites. Official client reads/writes self through PDS; CraftSky already supports conditional record updates. | FR-011/AC-011 require preserving unknown fields and conditional merge, with eventual chat-service propagation distinguished from successful PDS persistence. |
| CraftSky previews need local hydration | Current messageInput supports text/facets and specific record/join-link embeds, not arbitrary external cards. | Confirmed CraftSky-only cards remain feasible via existing authenticated content reads. No chat embed scope, generic scraper, composer/request fetch, or third-party private-text service is needed. |
| Privacy-safe push reuse requires adaptation | Existing dispatcher is tied to social notification rows and actor_mutes. The confirmed DM policy deliberately separates actor mute, shared conversation mute, and request/catch-up state. | Reuse durable leasing/delivery machinery but add private routing/eligibility rather than pretending the current dispatcher already handles DMs. |

Sources: [RPC parser](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/oauth/oauth-scopes/src/scopes/rpc-permission.ts), [unread counts contract](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky/convo/getUnreadCounts.json), [request listing](https://github.com/bluesky-social/atproto/blob/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8/packages/api/lexicons/chat/bsky/convo/listConvoRequests.json), [event baseline/resume](https://github.com/bluesky-social/social-app/blob/24e0ebbcd18b794934d617de20477c82a37ac2f0/src/state/messages/events/agent.ts), [active intervals](https://github.com/bluesky-social/social-app/blob/24e0ebbcd18b794934d617de20477c82a37ac2f0/src/state/messages/convo/const.ts), [history pagination](https://github.com/bluesky-social/social-app/blob/24e0ebbcd18b794934d617de20477c82a37ac2f0/src/state/messages/convo/agent.ts), and the local files named above.

### Flutter verification performed

A temporary directory contained copies of CraftSky’s actual pubspec.yaml and pubspec.lock. Published flutter_chat_ui 2.12.0 metadata and its transitive constraints fit the declared Dart/Flutter, intl, dio, path_provider, and material_ui constraints. An online Flutter dry run resolved dependencies but its post-step expected a package_config file that dry run does not create; a subsequent direct Dart offline dry run completed successfully against that freshly populated isolated cache:

```text
DART_SUPPRESS_ANALYTICS=true PUB_CACHE=/tmp/craftsky-dm-pub-cache \
FLUTTER_ROOT=/Users/douglastodd/fvm/versions/stable \
/Users/douglastodd/fvm/versions/stable/bin/cache/dart-sdk/bin/dart pub add \
  --directory=/tmp/craftsky-dm-pubcheck --offline --dry-run \
  --no-precompile --no-example flutter_chat_ui@^2.12.0
```

The solver would add nine packages (flutter_chat_ui, flutter_chat_core, cross_cache, diffutil_dart, idb_shim, nested, provider, scrollview_observer, sembast), with no existing dependency downgrade/override. Both temporary dependency files still matched the repository files byte-for-byte afterward. The package’s internal Provider usage can stay behind its widget boundary; CraftSky’s Riverpod/business-state architecture need not be replaced. This proves dependency resolution only, not rendering/accessibility behavior, mobile/web compilation, draft persistence, or privacy-safe builder configuration.

### Unverified service behavior (reference)

Live acceptance of the permission matrix, account eligibility, reporting, declaration propagation, and service-dependent cursor/leave transitions was not established by this source audit. NFR-007/AC-024 cover functional validation during full integration; FR-007/AC-008 define the selected native badge semantics. There is no separate spike, pre-design live-service gate, or latency/capacity benchmark. The read-only audit performed no live messaging, reporting, or account mutations.

The audit did not change production, request new user OAuth consent, send a message/report, install a repository dependency, or run feature tests/builds. Its findings inform the full implementation and its functional acceptance checks.
