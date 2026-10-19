# Requirements: Direct Messaging

## 1. Initial Request

Add an optional Bluesky-backed inbox to CraftSky, initially for one-to-one text messages. Include conversation requests, paginated history, unread counts, read updates, mute/unmute, leave, reporting, profile entry points, notification navigation, and incoming-message synchronization. Flutter continues to use its CraftSky session with AppView; AppView retains OAuth credentials and accesses the chat service through the user's PDS.

The brief proposes roughly 10–12 CraftSky API operations, explicit chat RPC permissions instead of the broad Full Chat Client permission set, and a background `getLog` worker extending durable push delivery. Endpoint count is an estimate, not an acceptance constraint. The inbox includes conversations started outside CraftSky. A live integration spike must establish feasibility before implementation commitments.

This artifact defines requirements and records the requested read-only feasibility audit. It does not start feature implementation, executable test development, migrations, deployment, or live messaging actions. The audit includes a dependency-resolution dry run in a temporary directory, not an app dependency change.

Follow-up on 2026-10-19: CraftSky links in messages load content previews when possible. The owner selected `flutter_chat_ui` for the conversation interface and completed three grilling rounds. All 24 resulting decisions below are confirmed; this revision records them without starting test design or implementation.

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
- Earlier raw-link failures are resolved by the pinned audit in section 24: packages/api contains link files for authFullChatClient and leaveConvo, whose actual definitions live in lexicons-overrides. Section 24 supplies the complete RPC inventory and exact permission/routing matrix. Schemas and official client behavior are verified; live grant acceptance, reporting acceptance, and recovery semantics remain spike deliverables.
- Grilling source checks found upstream schemas now under `packages/api/lexicons/`; the contract links above use that location. [Official request UI](https://raw.githubusercontent.com/bluesky-social/social-app/main/src/screens/Messages/components/RequestButtons.tsx) describes reply-as-acceptance, but CraftSky deliberately requires explicit Accept first. [Official leave copy](https://raw.githubusercontent.com/bluesky-social/social-app/main/src/components/dms/LeaveConvoPrompt.tsx) describes deletion for the leaving viewer, not other participants. Reappearance/history restoration/new-request transitions remain unverified spike questions.
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

Other direction comes from the initial brief: optional enablement, granular permissions, one-to-one text only, shared external conversations, AppView-mediated access, and a feasibility spike. Section 20 distinguishes resolved assumptions from the remaining provider-feasibility assumption.

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
| GQ-05 | Main badge | Count unread accepted direct conversations, excluding muted conversations. Requests have a separate indicator. |
| GQ-06 | Freshness | Updated by the owner after the feasibility audit: use a 4-second foreground polling interval to match Bluesky’s client. This replaces the earlier 3-second message-arrival target. Retain the 60-second background detection target, excluding OS push time. Polling cadence does not guarantee end-to-end arrival latency. |
| GQ-07 | New message | Include a New message action with CraftSky profile search. Defer global Bluesky handle search; preserve existing external conversations and profile entry points. |
| GQ-08 | Local drafts | Preserve one local draft per account/conversation across navigation and restarts. Clear on send or explicit discard; remove the account’s drafts on logout from that device. No cross-device draft sync. |
| GQ-09 | Uncertain sends | Refresh history first; if unresolved, retain text and offer explicit Send again with a duplicate-risk explanation. Never automatically resend. |
| GQ-10 | Preview targets | Support posts, profiles, and business events. A focused reply link previews the reply and opens the thread focused on it. |
| GQ-11 | No composer preview | Show previews only in message bubbles, never while composing. |
| GQ-12 | Conversation UI | Use `flutter_chat_ui` for the conversation interface, with CraftSky-specific presentation/adapters as needed. Validate compatibility; raise concrete blockers before substituting a different package or custom interface. |
| GQ-13 | Push toggle scope | Use one account-wide DM push toggle; retain per-device OS notification permissions. |
| GQ-14 | Requests indicator | Count unread request conversations. Reading clears their indicator without accepting; read requests remain in the requests list. |
| GQ-15 | Web boundary | Support web inbox and foreground updates. Background push is iOS/Android only in v1. |
| GQ-16 | Session expiry | Preserve existing session expiry, normally 30 days of inactivity. Stop background access when no usable session remains; explain paused notifications and reauthentication. |
| GQ-17 | Drafts on disable | Disabling messaging retains local drafts, hidden until re-enabled. Logout and explicit discard still remove them; explain this distinction during disconnect. |
| GQ-18 | Preview limit | Show at most one card per message, for the first supported, accessible CraftSky link. Keep every URL usable as text. |
| GQ-19 | Compact previews | Show identifying text, short excerpt/event details, and a permitted thumbnail where available. Open full content on tap; no inline video playback or nested interactive post UI. |
| GQ-20 | Explicit acceptance | Require Accept before enabling a request’s composer. Reading alone never accepts. |
| GQ-21 | Request deduplication | Leaving, reinstalling, or reconnecting does not reset request notification eligibility. Allow one new notification only after acceptance followed by a genuinely new request transition, subject to verified upstream behavior. |
| GQ-22 | Catch-up | After a temporary interruption or reauthentication, allow one catch-up notification per eligible unread accepted conversation instead of replaying missed messages. Do not notify pre-enable history; respect current read/mute/block settings. |
| GQ-23 | Request previews | Before acceptance show text and usable links, without automatic content-preview cards. Cards become eligible after acceptance. |
| GQ-24 | Offline sends | Do not queue offline sends for automatic delivery. Retain the draft, explain connection requirements, and require the owner to send after reconnecting. Already-submitted uncertain sends use GQ-09. |

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

Use Option A, conditional on a live integration spike. Keep Bluesky authoritative for conversations, messages, request state, read state, and chat mute state. Store only the private operational metadata CraftSky needs for enablement, synchronization, safe command handling, preferences, and notification delivery. Do not create CraftSky message records or index DMs through Tap/firehose ingestion.

This follows the supplied direction and existing token-mediating backend boundary. The spike must prove the capability set, effective grants after refresh, one-to-one send/read/request behavior, reporting, shared preferences, and background event detection. If required granular authority is unavailable, return to requirements review rather than silently widening access or dropping safety features.

## 6. Problem / Opportunity

Crafters need private one-to-one communication without leaving CraftSky. A hosted chat integration can provide it without operating a separate message storage and delivery service, provided users understand the shared inbox and CraftSky handles privacy, safety, authorization, and failures reliably.

## 7. Goals

- G-001: Provide a useful one-to-one inbox with conversation requests and unread state.
- G-002: Preserve server-side OAuth authority and make messaging optional and reversible.
- G-003: Deliver timely CraftSky notifications without duplicate jobs, stale eligibility, or message previews.
- G-004: Respect blocks, shared messaging preferences, and the selected Bluesky reporting boundary.
- G-005: Establish live feasibility before committing to a release implementation.

## 8. Non-Goals

- NG-001: Group chat creation, participation UI, invitations, or group notifications.
- NG-002: Sending attachments, media, upstream record embeds, reactions, reply threading, editing, deletion, export, or chat-account deletion. Local rendering of previews for CraftSky links in message text is included by FR-017; composer and unaccepted-request previews are excluded.
- NG-003: A CraftSky-only conversation namespace, recipient membership requirement, or filtering direct conversations by which app started them.
- NG-004: A new message transport/storage service, public message records, Tap indexing, or a promise of end-to-end encryption or PDS-based message portability.
- NG-005: Full offline inbox/history, durable offline sends, cross-device draft synchronization, typing indicators, presence, or recipient read receipts in the UI. Local unsent drafts are included; automatic offline-send queues are excluded.
- NG-006: Cross-app push deduplication, changing Bluesky's notification preferences, or a CraftSky DM moderation console.
- NG-007: Global Bluesky recipient search, monetization changes, release scheduling, or native desktop build targets. Include New message with CraftSky profile search, profile entry points, and existing inbox conversations; global Bluesky search remains excluded.

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

An owner chooses account-wide Enable messaging, sees that the inbox and selected settings are shared with Bluesky, and grants the additional minimal authority. Their existing one-to-one conversations and requests appear, including correspondents outside CraftSky. The owner can start a conversation from a profile or New message with CraftSky profile search when permitted, send text, read history, accept requests, mute/unmute, leave, report to Bluesky, and change who may message them.

Foreground views refresh from authoritative state. Authorized background synchronization detects new incoming messages and schedules eligible sender-name-only pushes. Disabling messaging stops CraftSky access and queued delivery without deleting the upstream conversations, changing incoming-message policy, or affecting unrelated CraftSky features.

Supported CraftSky content links display an accessible preview when their target can be resolved for the current viewer. Tapping the preview opens that content in CraftSky; unavailable previews leave the message and link intact. Use `flutter_chat_ui` for the conversation timeline/composer, with CraftSky-specific adapters/cards. Show compact previews only in accepted-conversation message bubbles, never in the composer or unaccepted requests. Persist local drafts as specified in FR-019; offline drafts never auto-send.

## 12. Requirements

Sources: P = initial prompt; U1–U6 = answers and follow-up requests in section 3; C = codebase findings; E = external contracts in section 2; GQ-01–GQ-24 = confirmed grilling decisions above; A = remaining explicit assumption in section 20.

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
| FR-007 | Functional | Must | Mark read through the last message actually presented in an active conversation and reconcile remote reads. Show upstream unread-message counts per row; the main badge counts unread accepted direct conversations excluding muted ones. Upstream aggregate counts may be capped at 99+; do not present the sentinel 100 as an exact total or decrement it as exact. A separate indicator counts unread request conversations; reading clears that indicator without accepting or removing the request. | Correct unread state | P, GQ-05, GQ-14 | AC-008, AC-012 |
| FR-008 | Functional | Must | Distinguish requests from accepted conversations; require confirmed explicit acceptance before enabling the request composer. Reading does not accept. Support confirmed leave as dismissal, following verified upstream semantics. | Request control | P, GQ-20 | AC-003, AC-009 |
| FR-009 | Functional | Must | Support shared upstream conversation mute/unmute and explicit confirmed leave; reconcile external changes. Leave confirmation explains its effect on the owner’s history/visibility and that the other participant’s messages are unaffected; do not promise permanent dismissal or recoverable old history. | Accurate lifecycle controls | P, E | AC-009, AC-012 |
| FR-010 | Functional | Must | Report an accessible message with a selected supported reason and optional explanation through com.atproto.moderation.createReport to Bluesky’s moderation audience, using the authenticated messageRef (sender DID, convoId, messageId), after showing the destination/evidence scope; display accepted, failed, or uncertain outcome accurately. | Selected safety path | U1 | AC-010 |
| FR-011 | Functional | Must | Read and explicitly update the shared incoming-message policy: everyone, people I follow, or nobody. Preserve it on enable/disable and preserve unrelated declaration fields (including allowGroupInvites and future fields) on update. Read the authoritative self record and use conditional create/update; on concurrent changes refetch and merge only the intended field. | Shared privacy control | U4 | AC-011 |
| FR-012 | Functional | Must | Persist account-scoped sync progress and durable event deduplication; make progress and resulting delivery work crash-consistent. Seed a new authorization baseline separately from normal resume; never replace a valid saved cursor with a fresh head on transient failure. Preserve unknown event envelopes/revisions so decoder gaps cannot stall progress. Recover interrupted polling or invalid cursors by reconciling authoritative state under the confirmed notification policies. | Durable detection | P, E | AC-012, AC-013 |
| FR-013 | Functional | Must | Schedule eligible incoming direct-message push with current authorization, subscription, read/mute, block, leave, and preference checks at dispatch. Notify once per new request until accepted; leaving/reinstall/reconnect does not reset this. After acceptance, permit another request notification only for a genuine new request transition. On outage/reauth recovery send at most one catch-up alert per eligible unread accepted conversation, without replaying individual missed messages or pre-enable history. Never notify owner-sent messages. | Relevant bounded notifications | P, U2, GQ-03, GQ-21, GQ-22 | AC-013, AC-014, AC-015, AC-029 |
| FR-014 | Functional | Must | Provide one account-wide DM push toggle, enabled by default after messaging enablement and visible during enablement, subject to device OS permission and active subscriptions. Support account-safe conversation notification navigation across cold start, foreground, logout, and account switch. | Notification control | P, C, GQ-02, GQ-13 | AC-015, AC-016 |
| FR-015 | Functional | Must | Refresh open inbox/conversation views after changes, on foreground/resume, and after reconnect; show stale/degraded state and provide retry rather than clearing previously loaded content on transient failure. | Consistent active clients | P | AC-012, AC-017 |
| FR-016 | Functional | Must | Enable/disconnect messaging account-wide; disconnect stops new chat work and cancels pending chat delivery across devices. Ordinary logout revokes that device’s access/subscription while other valid sessions continue; sign-out-all or loss of the last usable session stops background access. Preserve existing session expiry without silent extension, explain reauthentication/paused notifications, and fence stale work/private UI. Retain hidden local drafts on disconnect as FR-019 requires. | Revocable access with device-safe logout | P, C, GQ-01, GQ-16, GQ-17 | AC-014, AC-016, AC-028 |
| FR-017 | Functional | Must | In accepted-conversation message bubbles, show at most one compact card for the first supported accessible CraftSky link: post, profile, or business event. A focused reply link previews that reply and opens its thread focused on it. Cards use identifying text, short excerpt/event details, and a permitted thumbnail; no inline playback or interactive nested post. No composer/request cards. Preserve every URL and message on missing/failed/restricted targets. | Useful bounded CraftSky previews | U5, GQ-10, GQ-11, GQ-18, GQ-19, GQ-23 | AC-025 |
| FR-018 | Functional | Must | Provide a New message action with CraftSky profile search and availability-aware recipient selection, alongside existing profile and inbox entry points. Do not add global Bluesky handle search or require CraftSky membership for existing external conversations. | Start conversations from the inbox | GQ-07 | AC-027 |
| FR-019 | Functional | Must | Preserve local unsent drafts per account/conversation across navigation and app restart, without cross-device synchronization. Clear on confirmed send or explicit discard; logout removes that account’s drafts on that device. Disable messaging retains drafts hidden until re-enabled and explains this behavior. Other accounts must never see them. | Preserve the owner’s unsent work | GQ-08, GQ-17, GQ-24 | AC-028 |
| RULE-001 | Business rule | Must | Either-direction CraftSky blocks disable new conversation creation and sending and suppress CraftSky DM push; retain upstream-accessible history. Upstream restrictions may further deny access. | Confirmed block policy | U2 | AC-018 |
| RULE-002 | Business rule | Must | Request acceptance, mute, leave, read state, and incoming-message policy use upstream semantics and may affect other clients; nobody/following availability must not be described as erasing existing conversations. | Avoid false privacy guarantees | P, U4, E | AC-008, AC-009, AC-011 |
| RULE-003 | Business rule | Must | Visible pushes contain sender display name (handle fallback, generic fallback if unavailable) and generic message copy only; message text, report prose, and embed previews are excluded from all provider payload fields. | Selected notification privacy | U3 | AC-015 |
| RULE-004 | Business rule | Must | Use explicit implemented RPC permissions bound to the chat audience, plus the fixed moderation audience for reporting and create/update-only declaration permission. Evaluate RPC authority semantically by method and audience, including equivalent combined/normalized scope forms. Keep advertised optional permissions separate from each flow’s requested/granted baseline so non-chat sessions remain valid. Do not request Full Chat Client, wildcard chat methods, or unrelated export/group/account-delete authority. | Least authority | P | AC-019 |
| RULE-005 | Business rule | Must | CraftSky profile mute affects social content, not DM eligibility. Shared conversation mute suppresses DM push and excludes the conversation from the main badge without marking it read; blocks retain RULE-001. | Separate social and chat controls | GQ-04, GQ-05 | AC-008, AC-015 |
| NFR-001 | Non-functional | Must | Flutter uses only authenticated CraftSky APIs; OAuth tokens and DPoP keys remain server-side. Enforce owner, membership, lifecycle, and capability checks for every request, queued operation, and private destination resolution. | Existing authority boundary | P, C | AC-016, AC-019 |
| NFR-002 | Non-functional | Must | Keep message/draft bodies, private routing, report evidence, credentials, and raw upstream payloads out of diagnostics, analytics, public records, and social notification storage. Persist no chat history archive; local unsent drafts are the explicit FR-019 exception. Minimize operational metadata under documented retention/cleanup that preserves required request deduplication and uncertain-send recovery across disconnect. Account deletion removes CraftSky private state without deleting upstream chat. | Private data minimization | P, C, GQ-08, GQ-17, GQ-21 | AC-020, AC-028 |
| NFR-003 | Non-functional | Must | New consumer APIs follow `/v1/`, bearer/device authentication, camelCase JSON, `{error, message, requestId}` failures, opaque bounded pagination, and a fixed upstream operation allowlist. | Existing API contract | C | AC-021 |
| NFR-004 | Non-functional | Must | Polling and request handling have bounded concurrency, timeouts, payloads, and retries with backoff; honor upstream rate limits and isolate per-account failures without starving other accounts. | Shared service resilience | P, C | AC-017, AC-022 |
| NFR-005 | Non-functional | Must | With a healthy provider and active authorization, use a 4-second foreground polling interval while a conversation is open, matching the audited Bluesky client cadence. Share account polling across active views/devices; rate-limit backoff, in-flight request exclusion, and lifecycle fencing take precedence. Retain the 60-second background detection target, excluding OS push time. Measure end-to-end arrival latency separately; a 4-second interval is not a 4-second delivery guarantee. | Confirmed polling cadence | GQ-06 | AC-022 |
| NFR-006 | Non-functional | Must | Support iOS, Android, and web inbox flows with existing localization, accessibility, loading/error, and keyboard conventions. Provide mobile background push on iOS/Android and foreground web updates; web push is outside v1. | Supported product platforms | C, GQ-15 | AC-023 |
| NFR-007 | Non-functional | Must | Complete and review the live feasibility spike in AC-024 before committing to the implementation design; record observed semantics, effective permissions, and any requirement changes against section 24’s pinned audit. Verify rather than assume that every newly published operation is deployed and allowed for the selected accounts. | Resolve external-service risks | P | AC-024 |
| NFR-008 | Non-functional | Must | Use `flutter_chat_ui` for the conversation interface with CraftSky adapters/custom cards as needed. Validate platform, license, SDK/dependency, styling, accessibility, pagination, custom-preview and message-state fit during coding planning. Keep AppView transport and credentials server-side; bring concrete incompatibilities back for review before substituting another package or a fully custom interface. | Confirmed package choice | U6, GQ-12 | AC-026 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002 | Given existing direct conversations started in both CraftSky and another client, including a non-CraftSky correspondent, when the owner opens and paginates the inbox, then all supported conversations are reachable with correct identities/state and no duplication or premature end caused by intervening group conversations. Use listConvos(kind=direct,status=accepted) and listConvoRequests; request responses may contain group union variants. Exhaust valid cursors even across filtered-empty pages. Groups produce no v1 messages, badge contribution, or push. |
| AC-002 | BR-002, BR-003, FR-001 | Given messaging is disabled, when the owner enables it, then shared-service disclosures precede additional authorization. Cancelled/refused/insufficient grants leave ordinary CraftSky usable and chat disabled or authorization-needed. Full valid grants produce ready/syncing state; a service outage produces an unavailable state rather than an empty-success inbox. Valid OAuth plus getStatus.chatDisabled=true produces a service-restricted state; generic status does not fabricate a reason or offer CraftSky-only verification. |
| AC-003 | BR-001, FR-002, FR-008 | Given both requests and accepted direct conversations, when the inbox is loaded or a request is opened, then they are distinguishable, unread state is visible, and reading the request alone does not silently accept it. The composer remains disabled until explicit acceptance succeeds; failure leaves it disabled. Upstream implicit acceptance on send is not used as a shortcut. |
| AC-004 | FR-003 | Given multiple history pages with deleted messages, embeds/replies/reactions, or an unknown variant, when the owner loads older history, then text is ordered correctly with stable scroll position, pagination continues while a cursor exists even if a page is short/empty after filtering, duplicates are omitted, and missing/unsupported content uses a safe placeholder without crashing or exposing raw payloads. |
| AC-005 | BR-001, FR-004 | Given a profile, when availability is checked then no conversation is created; when Start message is explicitly selected and permitted, then the canonical direct conversation opens. Repeated starts reuse it. Self-targeting, extra participants, blocks, disabled incoming messages, and unavailable recipients cause no unintended conversation or send. |
| AC-006 | FR-005 | Given plain text including emoji/combining marks and multibyte characters, when submitted, then both layers enforce nonblank, 1,000-grapheme and 10,000-byte bounds. Invalid input makes no upstream send and is retained for correction; valid input is not truncated and is shown as confirmed only after an authoritative result. |
| AC-007 | FR-006 | Given a send operation, when it is double-submitted, retried after reconnect, or resumed after an AppView crash, then the same operation does not issue an additional upstream send. A timeout after possible acceptance displays uncertain status and refreshes history without automatic replay; identical message text alone does not prove delivery. If history refresh still cannot establish the outcome, retain the text and offer Send again with an explicit duplicate-risk explanation; only the owner’s confirmation creates a new send operation. |
| AC-008 | FR-007, RULE-002, RULE-005 | Given unread messages, when the owner views through a known foreground message, read state advances only through it; later messages remain unread and remote reads reconcile. Row counts show unread messages. The main badge counts only unmuted unread accepted direct conversations; the separate request indicator counts unread request conversations. Reading a request clears its indicator without acceptance/removal. If using getUnreadCounts, send includeGroupChats=false and display 100 as 99+, preserving the capped/unknown-total state on optimistic changes. Its service counts also exclude muted requests, locked/blocked/deactivated cases; do not silently change the confirmed request-indicator semantics—reconcile the dedicated request list where aggregate filtering differs, or raise a mismatch during design review. Conversation mute does not mark read; profile mute alone changes neither DM badge nor read state. |
| AC-009 | FR-008, FR-009, RULE-002 | Given an accessible direct conversation/request, when accept, mute, unmute, or confirmed leave succeeds, then the next authoritative view and delivery eligibility reflect the result. Failed actions do not show confirmed success. Leave confirmation describes the verified owner-history effect; cancelling has no effect. Successful leave cancels pending CraftSky push without deleting the correspondent’s history and does not promise permanently preventing future contact. |
| AC-010 | BR-003, FR-010 | Given an accessible message and supported reason, when reporting is confirmed, then submit reasonType, optional reason (within 2,000 graphemes/20,000 UTF-8 bytes), and an open-union chat.bsky.convo.defs#messageRef derived from an authorized message view to the fixed Bluesky moderation proxy. Do not attach a copied transcript or use moderator-only context APIs. The report audience is distinct from chat, and only the required reference/evidence is submitted; CraftSky moderators receive no report. Success is shown only on acceptance. Failure/uncertainty preserves a recoverable state and does not silently switch destinations or blindly resubmit. |
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
| AC-022 | NFR-004, NFR-005 | Given an authorized owner with an open conversation and healthy completed requests, verify a configured 4-second foreground poll cadence, shared across that account’s active views/devices rather than duplicated per viewer. Given 429, slow/in-flight calls, or lifecycle loss, verify backoff/non-overlap/fencing instead of forcing the cadence, with continued healthy-account progress. Measure actual foreground arrival latency separately and retain the 60-second background detection target; do not assert message arrival within four seconds solely from the configured interval. Record provider delay separately from OS push arrival. |
| AC-023 | NFR-006 | Given each supported platform, when using enablement, inbox, requests, profile entry, conversation, reporting, preferences, and recovery flows, then keyboard/screen-reader labels, focus, localized copy, and narrow/wide layouts are usable. Verify mobile push opens on iOS/Android; verify web foreground refresh without assuming web push. |
| AC-024 | NFR-007 | Given controlled consenting test accounts, when the separately authorized spike runs, then record effective granular grants through refresh; PDS proxy routing; canonical direct-conversation reuse; availability/request acceptance; send/uncertain outcomes; read and remote-state changes; mute/leave; reporting destination/permissions; declaration preservation; the added status/request/count RPCs and their grants; muted-request/count-cap behavior; initial/incremental log cursors and recovery without head reseeding or missing unknown revisions; request-read behavior without acceptance; conditional declaration propagation; message arrival latency and polling capacity at the configured 4-second foreground interval; and background detection while the app is closed but its session remains valid. Verify that sign-out-all and last-session expiry stop access, and verify provider eligibility restrictions without bypassing them. Unsupported behavior is documented and requirements revised before implementation design approval. |
| AC-025 | FR-017 | Given accepted direct messages containing post, focused-reply, profile, event, multiple, inaccessible, and unsupported links, show at most one compact card for the first supported accessible target, with permitted identifying text/excerpt/details/thumbnail and canonical navigation. A reply card previews the reply and focuses it in the thread. Preserve all text URLs on failure and do not block chat. No cards or preview resolution occur in the composer or unaccepted requests; acceptance makes existing request messages eligible. No autoplay/nested post controls, arbitrary URL fetches, third-party disclosure of private text, stale cross-account results, or preview data in push/diagnostics. |
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
| EC-011 | CraftSky preview target is removed, restricted, slow, or resolves after an account switch | Fall back to the original link; cancel/discard stale results and apply current-viewer access/moderation rules | FR-017 |

## 15. Data / Persistence Impact

- New private state is expected for owner enablement/capability status, account sync cursor/baseline, bounded deduplication metadata, send-operation identity/outcome, DM push preferences, and durable notification routing/delivery, including request-alert lifetime and coalesced recovery state. Exact tables and retention periods belong to the implementation design. Request-notification state must survive reconnect/reinstall and cannot be discarded in a way that resets GQ-21; uncertain-send operation state must likewise prevent unsafe replay.
- Existing OAuth/session and owner-lifecycle models must accommodate optional effective chat capability. Existing notification delivery must support a private conversation destination without copying message bodies into social notification rows.
- No authoritative message-history database, public message records, or persisted report evidence for CraftSky moderators. Received/sent history is transient authenticated response content. Local unsent drafts are the explicit exception: persist per account/conversation on the device only, hidden on messaging disable and removed for that account on device logout. They are not an AppView message archive.
- A migration is expected for operational state; none is designed or run here. Retention and cleanup periods must be fixed and tested before implementation approval, including dedupe retention versus cursor replay horizons.
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
- Chat RPC allowlist: `listConvos`, `listConvoRequests`, `getConvo`, `getMessages`, `getConvoAvailability`, `getConvoForMembers`, `getUnreadCounts`, `sendMessage`, `updateRead`, `acceptConvo`, `muteConvo`, `unmuteConvo`, `leaveConvo`, and `getLog` under `chat.bsky.convo`, plus `chat.bsky.actor.getStatus`. These 15 methods map to requirements in section 24; deployed availability and effective grant acceptance still need the spike. Example: `rpc:chat.bsky.convo.sendMessage?aud=did:web:api.bsky.chat%23bsky_chat`.
- Incoming-policy writes need limited create/update authority for `chat.bsky.actor.declaration` and a safe read/modify/write contract. Reporting uses `com.atproto.moderation.createReport`, an open-union `chat.bsky.convo.defs#messageRef`, and Bluesky’s fixed `did:plc:ar7c4by46qjdydhdevvrndac#atproto_labeler` proxy/audience. That contract is verified in official client/PDS source; live acceptance remains to be tested. It is not part of the chat audience/permission set. No local Lexicon modification is proposed. Any later change under `lexicon/` invokes the repository's skill and ADR requirements.
- CLI: no new user-facing command required. A separately authorized spike may use a development harness; that is later-stage work.
- Background jobs: account-scoped durable polling and reconciliation feeding existing delivery infrastructure. Foreground update transport is an implementation choice; no separate realtime-service requirement is introduced.
- Link previews: recognize supported canonical CraftSky content links and reuse existing content resolution/navigation where possible. Do not require changes to the upstream message format or a general external-link scraping service. Supported targets are posts/focused replies, profiles, and business events. Use one compact card for the first accessible supported link in accepted-conversation message bubbles only; no composer or unaccepted-request cards.
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
- Metrics: aggregate sync lag, poll outcomes/rate limits, active authorized polling accounts, queued/failed delivery, recovery counts, and uncertain sends. Avoid DID, conversation, message, cursor, and device labels.
- Alerts: actionable sustained sync failure/lag and delivery backlog thresholds to be chosen in the implementation design. Expected per-account reauthorization is a user recovery state, not an unbounded operator issue stream.
- Verification: inspect serialized sinks with positive expected fields and protected-value canaries. Transport acceptance, a mock event, or HTTP 200 alone does not prove real message delivery or notification navigation.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Granular permissions or PDS/provider behavior differ from published contracts | Feature cannot work with selected authority | Required spike; pin verified schemas; return for scope review rather than use broad fallback |
| RISK-002 | Provider accepts a send while its response is lost | Duplicate messages or false success | Stable operation identity, durable outcome, uncertain UI, no automatic replay |
| RISK-003 | Cursor/baseline/replay behavior is misunderstood | Lost incoming events or notification floods | Crash-consistent progress, dedupe, authoritative reconciliation, explicit baseline tests |
| RISK-004 | Read/mute/block state changes outside CraftSky or after final dispatch check | Stale notification or attempted send | Upstream enforcement plus CraftSky checks; refresh before dispatch; document residual in-flight race |
| RISK-005 | Private content enters diagnostics, push, reports, or another account's UI | Privacy breach | Minimal persistence, explicit Bluesky report boundary, sink canaries, owner fencing and account-switch tests |
| RISK-006 | Shared chat differs from broad product portability/moderation promises | Misleading consent or expectations | Visible hosted/shared-service disclosure; explicit DM qualification of product principles |
| RISK-007 | Per-account polling exceeds service limits or singleton capacity | Lag, rate limiting, backend contention | Bounded fair scheduling/backoff; measure targets in spike and design |
| RISK-008 | Missing or partial upstream capability includes reporting/preferences | Incomplete safety/control surface | Treat required capabilities as release blockers; do not silently ship without them |
| RISK-009 | Upstream schema/service evolution introduces groups and new variants | Crashes, leaked unsupported notifications, stalled cursors | Direct-kind filtering, tolerant variants, contract pinning and fixture coverage |

## 20. Assumptions

The original assumption IDs are preserved for traceability. ASM-001–ASM-005 are resolved by confirmed grilling decisions and no longer authorize alternative behavior.

| ID | Assumption / resolution | Impact If Wrong |
|---|---|---|
| ASM-001 | Resolved by GQ-02, GQ-03, GQ-13, GQ-21, GQ-22: account-wide push enabled by default, once-per-request alerts, no pre-enable backlog, coalesced accepted-conversation catch-up. See FR-013/FR-014. | A change requires an explicit product revision |
| ASM-002 | Resolved by GQ-05/GQ-14: main badge counts unmuted unread accepted conversations; separate indicator counts unread request conversations. | A change requires an explicit product revision |
| ASM-003 | Resolved by updated GQ-06: fixed 4-second foreground polling cadence and a 60-second background detection target. End-to-end arrival latency is measured separately, not inferred from the interval. | Cadence changes require review; backoff and latency must be reported honestly |
| ASM-004 | Resolved by GQ-15: iOS/Android background push; web inbox/foreground updates without web push. | Adding web push expands v1 scope |
| ASM-005 | Resolved by GQ-07/GQ-08/GQ-17/GQ-24: New message with CraftSky profile search; local drafts retained on disable, removed on logout; no global Bluesky search, offline history, draft sync, or auto-send queue. | A change requires an explicit product revision |
| ASM-006 | Remaining feasibility assumption: compatible accounts can obtain required granular chat/report/declaration permissions and satisfy service eligibility. Unsupported accounts receive clear unavailable/reauthentication guidance; establish exact provider/account prerequisites in the spike. | Selected integration may require revision |

## 21. Open Questions

- [ ] **Implementation-design blocker:** validate live acceptance of section 24’s exact least-authority permission matrix and report reference/audience, declaration update/propagation, and supported PDS/account eligibility prerequisites in the spike. The published report contract and intended methods are now known, not open product choices. No automatic broad-scope fallback.
- [ ] **Implementation-design blocker:** establish initial log baseline, pagination/replay/cursor-expiry behavior, recovery mechanics consistent with confirmed catch-up policy, upstream acceptance and leave/reopen semantics (including whether a genuinely new request transition can occur), and how read/mute/block state is refreshed before dispatch.
- [ ] **Implementation-design blocker:** specify operational metadata retention/cleanup periods, idempotent-operation retention, request-notification dedupe survival across leave/reinstall/disconnect, and recovery without duplicate send/notification effects. Include account-isolated local draft storage and logout cleanup. Avoid an unbounded private history store.
- [ ] **Remaining feasibility assumption:** verify ASM-006; ASM-001–ASM-005 are resolved product decisions. Verify any service-specific email/age/other eligibility requirements rather than attempting a CraftSky bypass.
- [ ] **Non-blocking design details:** navigation placement, exact route bodies/lifecycle routes, active-view transport, localization copy, and operational alert thresholds.
- [ ] **Non-blocking design details:** exact compact card layout and flutter_chat_ui adapters/runtime compatibility. Published version 2.12.0 resolves in the isolated dry run; no downgrade/override was needed. Target types, preview limits/placement, and package selection are settled; do not reopen them as undecided scope.
- No unanswered product question prevents this requirements draft. No authenticated live experiments have been performed. Section 24 establishes API/library-level feasibility and the precise remaining live-service gates; it does not claim production integration success.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer: Owner confirmed GQ-01–GQ-24 during grilling; revised artifact awaits any subsequent stage approval
Date: 2026-10-19 (grilling decisions and pinned feasibility audit incorporated; initial draft 2026-10-18)
Notes: High risk reflects private messages, new optional OAuth authority, shared-service settings, ambiguous upstream writes, and background delivery. Product choices in section 3, including all GQ-01–GQ-24, are confirmed. The owner ended grilling and requested this revision; that instruction does not start test design or implementation. The revised artifact remains Draft pending any later stage approval. Under the write-requirements skill's high-risk policy, explicit approval is required before test design or implementation. The spike is a later authorized activity, not work performed in this stage. No commit created. Section 24 records the requested API audit and isolated package-resolution evidence; source implementation remains unchanged.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-18-direct-messaging/01-requirements.md`.
- Next test specification: `02-acceptance-tests.md`, only after explicit high-risk requirements approval.
- Must-cover requirement IDs: BR-001–BR-003; FR-001–FR-019; RULE-001–RULE-005; NFR-001–NFR-008.
- Suggested test levels: unit tests for capability evaluation, count/eligibility rules, validation and safe mapping; real-PostgreSQL integration tests for leases/cursors/dedupe/send/lifecycle fencing; mocked upstream fault and contract tests; Flutter provider/widget/routing tests; iOS/Android/web integration checks; controlled live two-account feasibility checks and mobile push-open checks.
- Test-design work can specify spike scenarios and isolate unresolved upstream expectations; do not convert guesses into asserted provider guarantees. Final implementation design must resolve the blockers in section 21.
- Include Unicode boundaries, lost-response sends, crash/replay windows, multi-session grants, multi-account notification navigation, remote read/mute/leave, and serialized privacy canaries. Do not claim real service/device verification from mocks or local tests.
- Include accessible/inaccessible CraftSky content previews, failed lookups, canonical navigation, account-switch races, and absence of preview content from push/diagnostics. Validate the selected flutter_chat_ui package as design evidence; behavior tests remain independent of package internals. Cover account/device logout boundaries, request-alert lifetime/reset, coalesced recovery, separate badges/mutes, local draft cleanup, and absence of composer/request preview fetches.
- Blocking before test design: explicit owner approval of this high-risk requirements artifact. Blocking before implementation design approval: AC-024 evidence and resolution of the marked implementation-design questions.

## 24. Feasibility Audit (2026-10-19)

### Verdict and verification boundary

**The proposed one-to-one integration is feasible at the published API, existing Go transport, and Flutter dependency-resolution levels. It is not yet proven against authenticated live chat/moderation services.** No confirmed product decision needs to be abandoned. The audit corrected missing RPCs, capped-count handling, report routing/evidence, optional-scope validation, and cursor/pagination assumptions. A mismatch between upstream aggregate filtering and the confirmed request-indicator policy needs explicit reconciliation, not a silent product change.

There is no CraftSky DM implementation yet in the inspected paths. This audit assesses the planned implementation and existing integration foundations; it does not claim that an HTTP success, a schema, or dependency resolution proves working delivery.

Pinned source baselines:

- AT Protocol/Bluesky schemas and public PDS/OAuth implementation: [`9f66a7e54b030700c03eda0bdbf35ffe756f4cd8`](https://github.com/bluesky-social/atproto/commit/9f66a7e54b030700c03eda0bdbf35ffe756f4cd8) (commit dated 2026-10-17). Recursive repository tree was complete, not truncated; all chat schema copies and the actual override targets were retrieved.
- Official Bluesky client: [`24e0ebbcd18b794934d617de20477c82a37ac2f0`](https://github.com/bluesky-social/social-app/commit/24e0ebbcd18b794934d617de20477c82a37ac2f0) (commit dated 2026-10-19). Inspected message event/conversation agents, query hooks, account declaration updates, service clients, and report submission.
- CraftSky Go dependency: Indigo `v0.0.0-20260903211445-41278964ec8e`, verified against the local module cache and `appview/go.mod`.
- Flutter registry: published `flutter_chat_ui 2.12.0`, `flutter_chat_core 2.9.1`, `cross_cache 1.1.1` metadata, plus the selected package’s builder APIs. See [package](https://pub.dev/packages/flutter_chat_ui/versions/2.12.0) and [custom builders](https://pub.dev/documentation/flutter_chat_core/latest/flutter_chat_core/Builders-class.html).

The Bluesky chat backend itself was not exercised or available as an inspected server implementation. Published schemas and official-client behavior can differ from deployed account/provider capabilities; those differences are why AC-024 remains a gate.

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

For each of the following 15 methods, request `rpc:<full NSID>?aud=did:web:api.bsky.chat%23bsky_chat`. These are **upstream RPC operations**, not a requirement for fifteen Flutter-facing routes; existing proposed routes can aggregate them.

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
| Aggregate badge semantics are close, not identical | getUnreadCounts excludes muted, locked, blocked, deleted/deactivated cases and caps both values at 100; the confirmed request indicator does not explicitly exclude muted requests. | Display 99+ for capped service totals. Derive/reconcile either badge bucket from authoritative list state wherever its confirmed policy differs, including request pages where necessary; do not silently inherit aggregate filtering. Report any unresolved policy mismatch before implementation. |
| Polling is supported; foreground cadence aligned after audit | Official client uses 4-second active polling and preserves the previous cursor on transient resume. The owner selected that same cadence, replacing the earlier three-second arrival target. Baseline seeding uses a cursorless getLog call. There is no normal chat subscription endpoint among the 50 RPCs; moderation streaming is privileged. | Share one per-owner event stream across devices/views and measure combined PDS/auth/worker/client latency and rate limits. Approximate steady load is inactive-authorized-accounts/60 plus active-chat-accounts/4 requests per second, before reconciliation/UI requests. Four-second polling is not a four-second end-to-end delivery guarantee. |
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

### Remaining live-service gates

AC-024 must prove: all 15 scoped methods and the separate report permission on supported PDSes; effective grant forms after refresh; non-chat login survival; chatDisabled/account eligibility behavior; live messageRef report acceptance; explicit request acceptance/read behavior; direct leave/restart/history semantics; cursor retention/expiry and recovery under a polling interruption; muted-request count reconciliation; declaration conditional updates and service propagation; and measured foreground/background latency within the intended capacity/rate budget. Do not create synthetic production reports, send messages to uninvolved people, or perform account mutations as part of a read-only audit.

The audit did not change production, request new user OAuth consent, send a message/report, install a repository dependency, or run feature tests/builds. It narrows the spike to specific external guarantees instead of leaving the whole design unexamined.
