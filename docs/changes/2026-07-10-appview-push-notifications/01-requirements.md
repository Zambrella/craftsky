# Requirements: AppView Push Notifications

## 1. Initial Request

Implement the AppView portion of push notifications while designing the AppView contracts around the eventual Flutter experience. Users must be able to configure notifications for likes, new followers, replies, mentions, quotes, reposts, and everything else. Each category has an `Everyone` or `People I follow` scope and an independent push-enabled toggle. The work should determine whether notifications need durable storage and define the metadata each notification type needs.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/internal/api/notifications.go` exposes the current notification response and handler.
  - `appview/internal/api/notification_store.go` derives notifications at read time from follows, likes, reposts, posts, and materialized mentions.
  - `appview/internal/index/craftsky_interaction.go`, `craftsky_post.go`, and `bluesky_follow.go` index the events that can produce notifications.
  - `appview/migrations/000011_craftsky_interactions.up.sql`, `000012_atproto_follows.up.sql`, and `000017_post_mentions.up.sql` define the current source tables.
  - `appview/internal/app/deps.go` wires indexers; `appview/cmd/appview/main.go` currently runs only the Tap consumer and HTTP server as long-lived processes.
  - `appview/internal/routes/routes.go` already registers `GET /v1/notifications` and the architecture reserves device registration under `/v1/notifications/devices`.
- Existing patterns:
  - App reads come from the AppView; the Flutter app never reads Craftsky data directly from a PDS.
  - `/v1/*` endpoints use Craftsky authentication, `X-Craftsky-Device-Id`, camelCase JSON, the standard error envelope, and opaque cursor pagination.
  - Tap indexers must be idempotent on `(URI, CID)`.
  - Private-by-intent data belongs in AppView Postgres rather than on a public PDS.
- Current behavior:
  - `GET /v1/notifications` derives active follows, likes, reposts, replies, and mentions dynamically.
  - Unfollows are deleted; unlike and unrepost mark interactions inactive. The corresponding derived notifications therefore disappear.
  - There is no quote notification, device-token store, notification preference store, durable notification event, push outbox, or push dispatcher.
  - The current AppView process has no generic background-job runner.
- Constraints discovered:
  - Push cannot be delivered reliably from a read-time union query because events may disappear and there is no durable retry or idempotency boundary.
  - Notification preferences, notification history, push tokens, and delivery state are private AppView data and require no lexicon changes.
  - FCM is the selected push provider; it can deliver to both Android and iOS when APNs is configured through Firebase.
  - The app is not live, so no historical data backfill or compatibility migration is required.
- Test/build commands discovered:
  - From the repository root: `just test`.
  - From `appview/`: `go test ./...` when the test database is available.
  - Focused AppView suites commonly target `./internal/index`, `./internal/api`, `./internal/routes`, and `./internal/app`.

## 3. Clarifying Questions And Decisions

### Q1: How are category preferences represented?

Answer: Each category has an `Everyone` or `People I follow` scope plus an independent push toggle. Turning push off means no push for that category; in-app notifications remain enabled.

Decision / implication: There is no `No one` in-app state in this pass. Preference storage needs `scope` and `pushEnabled` for every category.

### Q2: What are the defaults?

Answer: Every category defaults to `Everyone` with push enabled.

Decision / implication: The API must return effective defaults even before a user has explicitly saved preferences.

### Q3: What happens after an actor undoes an action?

Answer: A deletion event such as unlike, unrepost, unfollow, or deletion of a reply/comment, mention, or quote shall retract the corresponding notification.

Decision / implication: Use a soft-deleted/tombstone lifecycle rather than immediately hard-deleting the row. Retracted notifications disappear from the normal feed and unsent pushes are cancelled, but the stable notification ID and safe fallback routing references remain resolvable for pushes that were already delivered.

### Q4: When do preference changes take effect?

Answer: Preference changes apply only to future notification events.

Decision / implication: Eligibility is evaluated and recorded when the event is ingested. Later preference or follow-graph changes do not rewrite existing notifications, but a source/subject deletion can retract them.

### Q5: What may appear in a push payload?

Answer: Use a minimal combined notification-and-data payload: actor display name, generic action copy, notification ID, notification type, and an opaque installation-specific account-subscription ID used only for local multi-account routing. Do not include post/project/mention text, image URLs, handles, DIDs, or raw AT-URIs in the visible alert or data payload.

Decision / implication: The Flutter app will fetch durable notification metadata after the notification is opened.

### Q6: Is existing activity backfilled?

Answer: No. The app is not live and no data migration is needed.

Decision / implication: A schema migration creates the new tables, but durable notification history starts when the feature is enabled. Existing indexed activity must not generate push retrospectively.

### Q7: Are likes and reposts of a recipient's repost attributable?

Answer: No. AT Protocol like and repost records identify the underlying post but do not identify which repost led the actor to it.

Decision / implication: Omit `likeViaRepost` and `repostViaRepost` completely. Do not expose preferences, create events, or approximate attribution by notifying every reposter.

### Q8: How are repeated action/undo cycles handled?

Answer: Coalesce notifications by recipient, actor, category, and subject. An undo retracts the row; a later repeat reactivates the same row at the top of the feed but never sends a second push for that relationship.

Decision / implication: Replays and genuine re-creations must be distinguished, while notification and push spam from repeated like/unlike, repost/unrepost, or follow/unfollow cycles is bounded.

### Q9: How long may push delivery continue?

Answer: A push expires six hours after its source event. The provider TTL must not exceed the remaining portion of that window.

Decision / implication: AppView retries transient failures only within the delivery window; stale social pushes are not delivered later by either AppView or FCM/APNs.

### Q10: How does one installation support multiple accounts?

Answer: Store one installation/FCM token and multiple authenticated account subscriptions. Each subscription has an opaque installation-specific routing ID included in pushes. Normal logout removes only the current account subscription and cancels its unsent deliveries; all-session logout removes that account's subscriptions across all installations without affecting other accounts.

Decision / implication: Device-token lifecycle and account authorization are separate. No DID or globally stable account identifier is exposed to FCM.

### Q11: What happens when the same FCM token is registered under a different device ID?

Answer: Treat the new authenticated registration as the token's current installation binding. Atomically deactivate the old installation, deactivate its account subscriptions, cancel its unsent deliveries, and bind the token to the new device ID. Do not transfer or infer any account subscription from the old installation; the authenticated account on the new registration receives only its own new subscription.

Decision / implication: This supports reinstall/restore and provider token reuse without allowing a token collision to merge account authorization across installation identities. A stale old installation cannot continue receiving pushes after the rebind.

### Q12: Is notification newness account-wide or device-specific?

Answer: Account-wide. Opening the notification page for an account on one device clears that account's new count on every device. Different accounts on the same installation remain independent.

Decision / implication: Persist one acknowledgement marker per account DID, not per installation, device, session, or account subscription.

### Q13: How does the client read and clear the new count?

Answer: Use a read-only `GET /v1/notifications/new-count` endpoint and an explicit `POST /v1/notifications/seen` operation. Flutter calls the write only once the notification page is actually displayed; fetching or prefetching the notification list does not mutate newness state.

Decision / implication: Preserve the API rule that GET requests are read-only and avoid clearing a badge because of background refresh, retry, or prefetch behavior.

### Q14: What does "new" mean?

Answer: A currently listable active notification is new when its latest genuine activation revision is later than the account's last acknowledged revision. Exact Tap replays do not create a new revision. Retraction removes a notification from the count, while a genuine reactivation or semantic source replacement receives a later revision and becomes new again.

Decision / implication: Use a monotonic server-side revision rather than a timestamp or per-row `readAt`. This makes delayed federation, tied timestamps, reactivation, and concurrent acknowledgement deterministic.

### Q15: What happens when a notification arrives while mark-seen is running?

Answer: Mark-seen advances only through the operation's database snapshot. A notification committed after that snapshot remains new.

Decision / implication: The acknowledgement operation records a captured high-water revision rather than `now()` or an unbounded value selected after the write begins.

## 4. Candidate Approaches

### Option A: Durable notification events plus transactional delivery outbox

Summary: Persist recipient-specific notification events during Tap indexing, queue eligible per-account-subscription push deliveries durably, serve the in-app feed from the notification table, and dispatch pushes asynchronously through FCM.

Pros:
- Gives notification activity an explicit active/retracted lifecycle tied to source deletions.
- Supports deterministic pagination, retry, delivery diagnostics, and multi-device users.
- Keeps Tap ingestion independent of FCM availability.
- Gives every notification type an explicit, evolvable metadata contract.

Cons:
- Adds several private AppView tables and a background worker.
- Requires careful transactional/idempotency design across existing indexers.
- Introduces retention and delivery-failure operations that the derived MVP avoided.

Risks:
- Incorrect fan-out or deduplication can create duplicate notifications or pushes.
- Push delivery is at-least-once; a process failure after FCM accepts a message but before the database update can still produce a rare duplicate.

### Option B: Keep the derived feed and add only a push outbox

Summary: Continue deriving the in-app feed at request time while separately storing push jobs as events arrive.

Pros:
- Smaller change to the current list endpoint.
- Durable push retry can still be added.

Cons:
- In-app and push state can diverge after undo operations because a delivered push cannot be reliably recalled while the derived row disappears.
- Metadata and eligibility logic are duplicated between the derived query and ingestion path.
- Cannot resolve an already-delivered push after the derived source disappears without a retained notification tombstone.

Risks:
- Users may receive a push for an event that is absent when they open the notification feed.

### Option C: Persist notifications but send synchronously from indexers

Summary: Store notifications, then call FCM directly while processing each Tap event.

Pros:
- Fewer tables and no polling worker.
- Low best-case latency.

Cons:
- FCM latency and outages block Tap acknowledgements.
- Retries risk duplicate sends and couple external delivery to firehose convergence.

Risks:
- A push-provider incident can stall indexing for the entire AppView.

## 5. Recommended Direction

Recommended approach: Option A, durable notification events plus a transactional per-account-subscription delivery outbox.

Why: Push delivery is an asynchronous side effect that needs durable intent, retry, and deduplication. A durable notification lifecycle lets the in-app feed, deletion handling, and push-open resolution refer to the same stable object, even after the visible row is retracted. FCM must never sit on the Tap acknowledgement path.

## 6. Problem / Opportunity

The current derived feed is useful for an in-app MVP but cannot reliably drive push notifications. Events can disappear, the AppView cannot retry failed delivery, users cannot control push categories, and the API has no durable identifier that a push can reference. Implementing a durable notification model closes those gaps while keeping private notification state out of public PDS records.

## 7. Goals

- G-001: Deliver timely, preference-aware push notifications to every active installation/account subscription registered by an eligible recipient.
- G-002: Maintain a durable, chronological in-app feed whose rows are retracted when their source activity or required destination is deleted.
- G-003: Provide type-specific metadata sufficient for Flutter to render and navigate every supported notification.
- G-004: Keep Tap ingestion reliable when FCM is slow or unavailable.
- G-005: Protect device tokens and minimize information exposed on lock screens and through push-provider payloads.
- G-006: Let Flutter display an accurate account-wide count of notification activity that is new since the account last acknowledged the notification page.

## 8. Non-Goals

- NG-001: Flutter UI, permission prompts, Firebase client SDK setup, token-refresh listeners, or deep-link implementation.
- NG-002: Per-notification read/unread state, per-device newness, individual mark-read operations, server-rendered badges, grouping, aggregation, or notification digests.
- NG-003: Per-account activity subscriptions such as “notify me whenever this person posts.”
- NG-004: Email, SMS, web push, or any provider other than FCM-backed Android/iOS delivery.
- NG-005: Localized server push copy; first-pass push copy is English.
- NG-006: Historical notification backfill or push delivery for events indexed before enablement.
- NG-007: Automatic notification retention/purging in this pass.
- NG-008: Lexicon changes or notification records stored on a PDS.
- NG-009: A concrete producer for `everythingElse`; the category and preference are reserved for future AppView events.
- NG-010: Likes or reposts of a recipient's repost; AT Protocol activity records do not provide trustworthy repost attribution.
- NG-011: Block- or mute-aware notification eligibility; Craftsky has no implemented block/mute model in this pass.
- NG-012: Temporary account-deactivation behavior.
- NG-013: Server-side burst aggregation, grouping, or digests; each eligible event creates its own push intent.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Notification recipient | Signed-in Craftsky member receiving social activity. | Relevant feed entries, safe fallback navigation for stale pushes, preference controls, and reliable delivery to active devices. |
| Notification actor | Account performing the activity. | At most one appropriate notification per recipient and event. |
| Flutter client | Future consumer of AppView notification and device APIs. | Stable IDs, complete metadata, idempotent registration, and minimal push data that can be resolved after open. |
| Tap indexer | Processes federated repository events. | Transactional, idempotent notification creation without blocking on FCM. |
| Push dispatcher | Long-lived AppView background worker. | Safe job claiming, retry policy, failure classification, and graceful shutdown. |
| Operator | Runs AppView and Firebase configuration. | Metrics and logs that diagnose delays and failures without exposing tokens or payload content. |

## 10. Current Behavior

`GET /v1/notifications` constructs a reverse-chronological union over current follows, active likes/reposts, direct replies, and mentions. The result has no stable AppView notification ID and no durable delivery state. Removing an underlying follow/like/repost removes the activity from the feed. Quote notifications are not generated. There are no device or preference endpoints and no FCM worker.

## 11. Desired Behavior

Tap ingestion identifies eligible recipients, selects one canonical category per recipient/event, applies the recipient's scope at event time, and inserts or reactivates a coalesced durable notification idempotently. Each genuine activation receives a monotonic newness revision; exact replays do not. A first activation with push enabled creates one pending delivery for every active account subscription on registered installations; later reactivations never enqueue another push for the same actor/category/subject relationship. A corresponding deletion event retracts the notification, removes it from `GET /v1/notifications` and the new count, and cancels unsent deliveries. A read-only endpoint returns the number of active, currently listable revisions newer than the account-wide acknowledgement marker, and an explicit mark-seen operation advances that marker through a database snapshot without consuming concurrently created notifications. A notification-resolution endpoint retains enough tombstone metadata to route an already-delivered push to its precise target or a safe fallback. A separate background worker claims pending deliveries and sends combined notification-and-data FCM messages with bounded retries and a six-hour delivery deadline.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky members shall receive durable in-app notifications for eligible social activity directed at them. | Keeps social activity understandable and stable. | Prompt / user decisions | AC-001, AC-009 |
| BR-002 | Business | Must | Eligible activity shall be deliverable as push to each active recipient account subscription according to preferences effective when the event occurs. | Provides timely notification across devices and accounts without retroactive preference rewrites. | Prompt / user decisions | AC-002, AC-014, AC-015 |
| BR-003 | Business | Must | Notification categories shall be `like`, `follow`, `reply`, `mention`, `quote`, `repost`, and `everythingElse`. | Matches the reviewed user-facing controls without unsupported repost attribution. | Prompt / grilling decisions | AC-003, AC-011 |
| BR-004 | Business | Must | Craftsky members shall be able to see how many currently listable notifications are new since they last acknowledged the notification page. | Enables an accurate navigation badge without introducing per-item unread management. | Follow-up user request | AC-045, AC-046 |
| FR-001 | Functional | Must | The AppView shall persist one durable notification per eligible `(recipient, actor, canonical category, subject)`, with a stable opaque notification ID and lifecycle state. | Enables coalescing, pagination, retraction, reactivation, and push lookup. | Recommended direction / grilling decisions | AC-001, AC-004, AC-009, AC-032, AC-035 |
| FR-002 | Functional | Must | Notification creation shall be idempotent when the same Tap event is retried or replayed. | Tap delivery is retryable and must not duplicate user activity. | Indexer invariant | AC-004 |
| FR-003 | Functional | Must | Notification intent and any initial push-delivery rows shall be committed atomically with the corresponding indexed source event, without making an FCM request inside that transaction. | Prevents lost intent and avoids provider coupling. | Discovery | AC-005, AC-016 |
| FR-004 | Functional | Must | `GET /v1/notifications` shall read active durable notification events rather than derive the feed from current interaction tables, excluding retracted tombstones. | Provides stable IDs while keeping deleted activity out of the feed. | User decision | AC-001, AC-009, AC-010 |
| FR-005 | Functional | Must | The durable feed shall remain reverse-chronological with deterministic opaque-cursor pagination and no duplicates or skips across ties. | Preserves API conventions and stable navigation. | Existing API contract | AC-009 |
| FR-006 | Functional | Must | The AppView shall expose authenticated APIs to retrieve and partially update all notification category preferences. | Flutter needs a settings contract. | Prompt / discovery | AC-011, AC-012 |
| FR-007 | Functional | Must | Each preference shall contain `scope` (`everyone` or `peopleIFollow`) and `pushEnabled`; absent persisted preferences shall resolve to `everyone` and `true`. | Implements confirmed controls and defaults. | User decisions | AC-011, AC-012 |
| FR-008 | Functional | Must | Preference scope shall be evaluated when the source event is ingested, using whether the recipient follows the actor at that time; scope applies to both in-app and push eligibility, and later preference or follow changes shall not alter existing notification eligibility or lifecycle. | Implements confirmed prospective behavior and settings semantics. | User decision | AC-013, AC-037 |
| FR-009 | Functional | Must | The AppView shall expose idempotent installation/token registration and authenticated account-subscription removal under `/v1/notifications/devices`, scoped by the session DID and `X-Craftsky-Device-Id`. | Supports token creation, rotation, multiple devices, multiple accounts, and logout. | API architecture / grilling decisions | AC-014, AC-017, AC-018, AC-038 |
| FR-010 | Functional | Must | An installation shall support `ios` or `android`, hold one current send-capable FCM token, and have multiple independently revocable account subscriptions; token rotation shall update the installation without transferring or collapsing its account subscriptions. | Separates physical routing from account authorization. | Grilling decisions | AC-014, AC-018, AC-038 |
| FR-011 | Functional | Must | A first notification activation shall create at most one pending delivery per active recipient account subscription, and a subscription created after the event shall not receive that event retrospectively. | Defines bounded fan-out and prevents backlog surprises. | User prospective-policy decision | AC-002, AC-015, AC-035 |
| FR-012 | Functional | Must | The AppView shall run a cancellable background push dispatcher that safely claims delivery rows across multiple processes, sends through FCM, and records outcomes. | Enables asynchronous, horizontally safe delivery. | Recommended direction | AC-016, AC-019, AC-020 |
| FR-013 | Functional | Must | Retryable FCM failures shall use bounded exponential backoff with jitter only until six hours after the source event; deliveries still unsent at that deadline shall enter a terminal expired state. | Avoids hot loops, infinite retries, and stale social pushes. | Discovery / grilling decision | AC-019, AC-036 |
| FR-014 | Functional | Must | Permanent invalid/unregistered-token responses from FCM shall terminally fail the delivery and deactivate the installation and its account subscriptions. | Stops repeated sends to dead tokens. | FCM integration behavior | AC-020 |
| FR-015 | Functional | Must | Successful deliveries shall not be selected for another normal send attempt. | Prevents routine duplicate pushes. | Recommended direction | AC-021 |
| FR-016 | Functional | Must | Normal logout shall remove only the authenticated account's subscription on the current installation; all-session logout shall remove that account's subscriptions across all installations; both shall cancel affected unsent deliveries without changing other accounts on those installations. | Makes logout a privacy boundary while supporting multiple accounts. | Security review / grilling decisions | AC-017, AC-038 |
| FR-017 | Functional | Must | Like and repost activity on a recipient-authored post shall produce `like` or `repost`; likes or reposts of a post the recipient merely reposted shall not produce a notification without trustworthy attribution. | Avoids misleading and unbounded fan-out. | Grilling decision | AC-006, AC-007 |
| FR-018 | Functional | Must | A re-created like, repost, or follow after its undo shall reactivate the existing `(recipient, actor, category, subject)` notification, update its activity timestamp so it returns to the top of the feed, and shall not enqueue another push. | Keeps current in-app state while preventing toggle spam. | Grilling decisions | AC-035 |
| FR-019 | Functional | Must | Post events shall use canonical per-recipient precedence `reply` over `quote` over `mention` when one post would otherwise notify the same recipient more than once. | Prevents duplicate rows and pushes for one authored event. | Discovery recommendation | AC-008 |
| FR-020 | Functional | Must | Self-generated activity shall not create notifications or push deliveries. | Users should not be alerted about their own actions. | Existing notification rule | AC-006, AC-008 |
| FR-021 | Functional | Must | Deleting or undoing a notification-producing source action shall transactionally retract its notification tombstone with the indexed deletion, exclude it from the normal feed, and cancel delivery rows that have not been successfully sent. | Removes notifications that are no longer relevant while acknowledging that delivered pushes cannot be recalled. | User decision | AC-005, AC-010, AC-032, AC-033 |
| FR-022 | Functional | Must | Notification responses shall provide common metadata plus the type-specific references defined in Section 15, and shall hydrate current display data in a bounded manner. | Flutter needs useful rendering and navigation context. | Prompt / discovery | AC-022, AC-023 |
| FR-023 | Functional | Must | When required destination content is deleted, the AppView shall retract affected notifications while preserving safe tombstone routing metadata; when content is unavailable or taken down, hydration and resolution shall not expose metadata the viewer may no longer see. Permanent actor account/repository deletion instead hard-deletes notifications caused by that actor and their unsent deliveries. | Stale notifications must not bypass deletion or visibility policy, while permanent deletion should not retain unnecessary actor data. | User decision / research / grilling decisions | AC-023, AC-024, AC-032, AC-039 |
| FR-024 | Functional | Must | FCM sends shall use a combined notification-and-data message containing only generic English action copy, actor display name where available, notification ID, category, and an opaque per-installation account-subscription ID; clients shall fetch full metadata from AppView. | Enables reliable OS display and multi-account routing while minimizing lock-screen and provider exposure. | User / grilling decisions | AC-025, AC-040 |
| FR-025 | Functional | Must | The feature shall not enqueue pushes for source events indexed before notification persistence is enabled. | The app is not live and retrospective delivery is unwanted. | User decision | AC-026 |
| FR-026 | Functional | Must | The AppView shall expose authenticated notification resolution by stable notification ID, returning the active precise destination or, for a retracted tombstone, the safest available fallback destination without restoring it to the feed. | Allows already-delivered pushes to open intelligently after deletion. | User clarification | AC-032, AC-034 |
| FR-027 | Functional | Must | Push preference changes remain prospective: disabling a category does not cancel deliveries already queued for that category, and re-enabling it does not create retrospective deliveries. | Avoids complex queue rewrites for normally short-lived jobs. | Grilling decision | AC-041 |
| FR-028 | Functional | Must | Every delivery shall have an absolute deadline six hours after the source event, and AppView shall set FCM/APNs TTL no later than the remaining time to that deadline. | Prevents provider-held messages arriving after AppView's freshness window. | Grilling decision | AC-036 |
| FR-029 | Functional | Must | Each account subscription shall have an opaque routing ID unique to that installation and account pairing; it shall not be a DID or globally stable account identifier. | Enables private local account selection without cross-device correlation. | Grilling decision | AC-040 |
| FR-030 | Functional | Must | Permanent actor account/repository deletion shall hard-delete notifications caused by that actor and cancel or delete their unsent deliveries; an already-delivered unknown ID shall receive the normal non-enumerating not-found response. | Aligns visible activity and retained data with permanent deletion. | Research / grilling decision | AC-039 |
| FR-031 | Functional | Must | The first pass shall send one push intent per eligible event without server-side burst aggregation, grouping, or digests. | Keeps delivery semantics deterministic until real usage demonstrates a need for aggregation. | Grilling decision | AC-042 |
| FR-032 | Functional | Must | Block and mute state shall not be invented by this notification feature; integration with a future authoritative block/mute model is deferred. Existing content availability and takedown checks still apply. | Avoids creating notification-only social policy unsupported elsewhere in Craftsky. | Codebase finding / grilling decision | AC-043 |
| FR-033 | Functional | Must | When an FCM token already belongs to a different active installation, authenticated registration shall atomically rebind it to the current `X-Craftsky-Device-Id`, deactivate the old installation and its subscriptions, and cancel their unsent deliveries without transferring subscriptions to the new installation. The new installation shall gain only the authenticated account's explicitly registered subscription. | Preserves one active token owner while preventing cross-installation account authorization leakage during reinstall, restore, or provider token reuse. | Document review decision | AC-044 |
| FR-034 | Functional | Must | The AppView shall assign a monotonically increasing newness revision whenever a notification is first activated, genuinely reactivated, or replaced by a semantically new source; exact idempotent replays and retractions shall not advance the revision. | Provides a deterministic arrival/activation order independent of source timestamps and process clocks. | Follow-up design decision | AC-047 |
| FR-035 | Functional | Must | `GET /v1/notifications/new-count` shall return `{newCount: N}` for the authenticated account, where `N` is the number of active notifications eligible to appear in the normal notification list whose newness revision is later than the account's acknowledgement marker. Retracted notifications and notifications excluded by current list-level visibility policy shall not be counted. | Keeps the badge consistent with the list surface and preserves read-only GET semantics. | Follow-up user decision | AC-045, AC-048 |
| FR-036 | Functional | Must | `POST /v1/notifications/seen` shall require the existing authenticated account and device middleware, accept no body, atomically advance that account's acknowledgement marker through the greatest notification revision visible to the operation's database snapshot, never move the marker backwards, and return `204 No Content`. | Clears current new activity without consuming a notification committed after the acknowledgement snapshot. | Follow-up user decision | AC-046, AC-049 |
| FR-037 | Functional | Must | Notification newness state shall be scoped by account DID across all devices, installations, sessions, and account subscriptions; acknowledging one account shall not affect another account. | Matches the selected account-wide product semantics and multi-account isolation model. | Follow-up user decision | AC-050 |
| FR-038 | Functional | Must | An account with no acknowledgement row shall treat all active, currently listable durable notifications as new. | Gives deterministic first-use behavior without a registration-time or login-time baseline side effect. | Follow-up design decision | AC-045 |
| NFR-001 | Non-functional | Must | FCM latency or outage shall not block Tap event acknowledgements or database indexing. | Protects AppView convergence. | Architecture | AC-016 |
| NFR-002 | Non-functional | Must | Device tokens and FCM credentials shall never be written to logs, Sentry attributes, metrics labels, API responses after registration, or notification payload diagnostics. | Tokens and credentials are sensitive operational data. | Security / privacy | AC-027 |
| NFR-003 | Non-functional | Must | All new `/v1/*` JSON shall use camelCase, standard error envelopes, existing authentication, and required device-ID middleware. | Preserves the AppView API contract. | AGENTS / API architecture | AC-028 |
| NFR-004 | Non-functional | Must | Production startup shall fail clearly when push delivery is enabled without valid required FCM configuration; development and tests shall support an injected fake or disabled sender. | Avoids silently dropping production push. | Operational discovery | AC-029 |
| NFR-005 | Non-functional | Should | Normal notification page hydration and delivery claiming should use bounded/batched queries and appropriate indexes rather than per-item database lookups. | Controls load on a high-traffic path. | Codebase patterns | AC-030 |
| NFR-006 | Non-functional | Should | The dispatcher should begin processing new jobs promptly under normal operation and expose queue age so latency regressions are measurable. | Timeliness is central to push usefulness. | Product goal | AC-031 |
| NFR-007 | Non-functional | Should | New-count and mark-seen queries should use an account/revision index and bounded snapshot operations rather than scanning or updating individual notification rows. | Keeps badge polling and acknowledgement inexpensive as notification history grows. | Codebase architecture | AC-051 |
| RULE-001 | Business rule | Must | `People I follow` means the recipient follows the notification actor at source-event ingestion time. | Removes ambiguity about graph direction and timing. | User decision / discovery | AC-013 |
| RULE-002 | Business rule | Must | For each category, `scope` decides whether an event creates a notification at all: `everyone` accepts every otherwise-eligible non-self actor, while `peopleIFollow` accepts only actors the recipient follows. Every accepted event appears in the in-app feed. `pushEnabled` independently decides whether that accepted event also creates push deliveries. There is no separate control for hiding accepted events from the in-app feed. | Makes the relationship between scope, in-app visibility, and push delivery explicit. | User decision | AC-011, AC-012, AC-037 |
| RULE-003 | Business rule | Must | Later preference and follow-graph changes shall not retroactively change notifications, but deletion of source activity or required destination content shall retract affected notifications. | Distinguishes preference timing from event relevance. | User decisions | AC-010, AC-013, AC-032 |
| RULE-004 | Business rule | Must | Push delivery is at-least-once; the AppView shall minimize duplicates but cannot guarantee exactly-once delivery across an external FCM acceptance/database-commit boundary. | States an unavoidable distributed-systems limit. | Discovery risk | AC-021 |
| RULE-005 | Business rule | Must | `everythingElse` shall have stored/default preferences but no source-event producer in this pass. | Reserves the requested control without inventing events. | Scope decision | AC-003, AC-011 |
| RULE-006 | Business rule | Must | Like/repost-of-repost categories shall not exist until Craftsky has trustworthy causal attribution; AppView shall never approximate them by notifying every reposter. | Prevents false attribution and unbounded fan-out. | Grilling decision | AC-003, AC-007 |
| RULE-007 | Business rule | Must | "New" is a high-water account state, not a permanent property of an individual notification. Opening or reading a list does not clear it; only the explicit authenticated mark-seen operation advances the account marker. | Prevents prefetches from mutating state and keeps the first pass intentionally simpler than per-item unread behavior. | Follow-up user decision | AC-046, AC-050 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-004 | Given an eligible new social event, when indexing commits, then one durable notification with a stable ID appears in the recipient's notification feed. |
| AC-002 | BR-002, FR-011 | Given push is enabled and the recipient has two active account subscriptions, when an eligible event occurs, then exactly one pending delivery is created for each subscription. |
| AC-003 | BR-003, RULE-005, RULE-006 | Given the preference API is requested, then all seven categories are represented, including `everythingElse` even though it has no producer, and no via-repost category is exposed. |
| AC-004 | FR-001, FR-002 | Given the same source event is replayed one or more times, when indexing completes, then the recipient has one notification and each account subscription has at most one delivery for it. |
| AC-005 | FR-003, FR-021 | Given creation or deletion of a notification-producing source and its notification/outbox lifecycle effects are attempted, then all corresponding effects commit or all roll back; no FCM network request occurs within the transaction. |
| AC-006 | FR-017, FR-020 | Given Alice likes/reposts Bob's post, then Bob receives the direct category; given Bob performs the action on Bob's own post, no notification is created. |
| AC-007 | FR-017 | Given Bob reposted Carol's post and Alice later likes/reposts the underlying post, then Carol may receive the direct category but Bob receives no notification based solely on having reposted it. |
| AC-008 | FR-019, FR-020 | Given one source record qualifies as multiple reasons for the same recipient, then canonical precedence yields one notification; self recipients are excluded. |
| AC-009 | BR-001, FR-004, FR-005 | Given durable notifications with tied timestamps, when pages are followed through opaque cursors, then rows are newest-first with deterministic ordering and no duplicate or skipped IDs. |
| AC-010 | FR-004, FR-021, RULE-003 | Given an active notification exists for a follow, like, repost, reply, mention, or quote, when its source action/record is deleted, then the notification is retracted and no longer appears in `GET /v1/notifications`. |
| AC-011 | BR-003, FR-006, FR-007, RULE-002, RULE-005 | Given a user has never saved preferences, when preferences are fetched, then every category returns `scope: everyone` and `pushEnabled: true`, while in-app delivery remains enabled. |
| AC-012 | FR-006, FR-007, RULE-002 | Given an authenticated user partially updates one category, when preferences are fetched again, then that category reflects the update and all omitted categories retain their prior effective values. |
| AC-013 | FR-008, RULE-001, RULE-003 | Given a category uses `peopleIFollow`, when events occur before and after the recipient follows/unfollows the actor or changes scope, then eligibility uses the relationship and preference effective for each new event, without retroactive preference reclassification. |
| AC-014 | BR-002, FR-009, FR-010 | Given a valid authenticated registration with current device ID, platform, and token, then the AppView upserts one installation and one active account subscription without returning the token. |
| AC-015 | BR-002, FR-011 | Given a recipient registers a device after an event occurred, then no delivery for the earlier event is created for that device. |
| AC-016 | FR-003, FR-012, NFR-001 | Given FCM is slow or unavailable, when Tap events arrive, then indexing and Tap acknowledgement continue while delivery rows remain available for asynchronous processing. |
| AC-017 | FR-009, FR-016 | Given normal logout on an installation, then only that account's subscription and unsent deliveries there are removed; given all-session logout, that account's subscriptions across installations are removed without affecting other accounts. |
| AC-018 | FR-009, FR-010 | Given an FCM token rotates for an installation, then future sends use only the latest token while all authorized account subscriptions on that installation remain intact. |
| AC-019 | FR-012, FR-013 | Given FCM returns a retryable error, then the dispatcher schedules bounded exponential-backoff retry only within the six-hour delivery window and eventually marks the job terminal. |
| AC-020 | FR-012, FR-014 | Given FCM reports a token invalid or unregistered, then the delivery becomes terminal and the installation and its subscriptions are inactive for future fan-out. |
| AC-021 | FR-015, RULE-004 | Given a delivery is recorded successful, then normal polling does not send it again; documentation/tests acknowledge the rare crash-window duplicate allowed by at-least-once delivery. |
| AC-022 | FR-022 | Given each supported category, when its notification is returned, then its common and type-specific references match the metadata matrix in Section 15. |
| AC-023 | FR-022, FR-023 | Given referenced content is unavailable but not confirmed deleted, when the notification is fetched or resolved, then inaccessible content is explicitly unavailable rather than leaking stale content. |
| AC-024 | FR-023, FR-032 | Given content is unavailable or taken down, when notifications are listed or resolved, then available visibility checks are applied and the notification path does not expose inaccessible content; no block/mute model is fabricated. |
| AC-025 | FR-024 | Given a delivery is sent, then it is a combined notification-and-data message containing only permitted minimal fields and no post text, project title, mention text, image URL, handle, DID, or AT-URI. |
| AC-026 | FR-025 | Given source activity predates feature enablement, when the schema/worker is deployed, then no notification or push is produced solely because that old activity already exists. |
| AC-027 | NFR-002 | Given registration, enqueueing, FCM success, and FCM failure paths, then logs/Sentry/metrics/API responses contain no device token, credential, or full push payload. |
| AC-028 | NFR-003 | Given new preference/device endpoints receive missing auth/device headers or invalid JSON, then existing middleware and error-envelope conventions are preserved with camelCase JSON. |
| AC-029 | NFR-004 | Given production push is enabled without valid required FCM configuration, then startup fails with a non-secret configuration error; tests can inject a fake sender without network access. |
| AC-030 | NFR-005 | Given a normal notification page or delivery batch, then database work is bounded/batched and supported by indexes for recipient pagination and pending-job claims. |
| AC-031 | NFR-006 | Given pending deliveries under normal operation, then metrics expose queue depth, oldest pending age, and delivery outcomes so latency can be monitored. |
| AC-032 | FR-001, FR-021, FR-023, FR-026, RULE-003 | Given a notification or its required destination is deleted after a push was delivered, when the notification ID is resolved, then AppView returns a retracted tombstone with the safest authorized fallback and the row remains absent from the normal feed. |
| AC-033 | FR-021 | Given a source deletion is processed before its pending/retry delivery succeeds, then that delivery becomes cancelled and is never selected for a normal send; a push already accepted by FCM is not claimed to be retractable. |
| AC-034 | FR-026 | Given an authenticated user tries to resolve another user's notification ID, then AppView returns the standard not-found response without revealing whether the notification exists. |
| AC-035 | FR-001, FR-011, FR-018 | Given a like, repost, or follow is undone and later recreated, then the same notification ID is reactivated with a new activity timestamp at the top of the feed and no second push delivery is created. |
| AC-036 | FR-013, FR-028 | Given a delivery is pending or accepted by FCM, then its AppView deadline and provider TTL cannot permit delivery more than six hours after the source event. |
| AC-037 | FR-008 | Given a category scope is `peopleIFollow`, then an event from an actor the recipient does not follow creates neither an in-app notification nor a push; for `follow`, this means only a new mutual follow is eligible. |
| AC-038 | FR-009, FR-010, FR-016 | Given one installation has subscriptions for two accounts, then both can receive pushes through one current token; logging out either account preserves the other's subscription and queued work. |
| AC-039 | FR-023, FR-030 | Given an actor account/repository is permanently deleted, then notifications caused by that actor and their unsent deliveries are hard-deleted; resolving an already-delivered ID returns non-enumerating not-found. |
| AC-040 | FR-024, FR-029 | Given the same account registers on two installations, then each receives a different opaque account-subscription routing ID, and pushes use the correct local ID without a DID or global account identifier. |
| AC-041 | FR-027 | Given a user disables push after a delivery is queued, then the queued delivery remains eligible to send; later events do not enqueue while disabled, and re-enabling does not backfill them. |
| AC-042 | FR-031 | Given many eligible events occur in a burst, then each event creates its own push intent and the AppView does not delay them for aggregation or digest construction. |
| AC-043 | FR-032 | Given no authoritative Craftsky block/mute model exists, then this pass neither creates notification-specific block/mute state nor claims block/mute enforcement. |
| AC-044 | FR-033 | Given an active installation owns an FCM token, when an authenticated registration presents that token with a different device ID, then AppView atomically deactivates the old installation and subscriptions, cancels their unsent deliveries, binds the token to the new installation, creates only the authenticated account's new subscription, and transfers no old subscription or routing ID. |
| AC-045 | BR-004, FR-035, FR-038 | Given an account has never acknowledged notifications and has active listable notifications, when it requests the new-count endpoint, then the response is `200 {"newCount": N}` with every such notification counted exactly once. |
| AC-046 | BR-004, FR-036, RULE-007 | Given an account has new notifications, when it successfully posts to mark-seen and then requests the count, then mark-seen returns 204 and the count is zero without requiring notification-list pagination. |
| AC-047 | FR-034 | Given exact Tap replay, retraction, and genuine reactivation cases, then replay and retraction do not allocate a later newness revision, while reactivation/source replacement does and becomes new without creating a second push delivery. |
| AC-048 | FR-035 | Given retracted notifications and notifications currently excluded by the notification list's actor visibility policy, when the count is requested, then neither contributes to `newCount`. |
| AC-049 | FR-036 | Given mark-seen captures revision R and a new notification commits at R+1 before mark-seen returns, then the marker advances through R and the later notification remains new. |
| AC-050 | FR-037, RULE-007 | Given one account is active on two devices and shares one device with another account, when the first account marks notifications seen on either device, then its count clears on both devices and the other account's count is unchanged. |
| AC-051 | NFR-007 | Given a large notification history, when count and mark-seen queries are explained or inspected, then they use account/revision indexes and update only the one account acknowledgement row. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | A post author also reposted their own post. | Only direct authorship can make them eligible for `like`/`repost`; the repost adds no notification reason. | FR-017, RULE-006 |
| EC-002 | An actor interacts with a post they themselves reposted. | No notification is created from repost ownership or the actor's own action. | FR-017, FR-020, RULE-006 |
| EC-003 | A quote or reply also mentions the same recipient. | One notification is created using reply, then quote, then mention precedence. | FR-019 |
| EC-004 | One post event targets different recipients for different reasons. | Each recipient may receive their own canonical notification; precedence is evaluated per recipient. | FR-019 |
| EC-005 | Subject post is not indexed when the event arrives. | No notification requiring that subject/recipient relationship is invented; the event may be logged/metriced as ineligible, without later retrospective push in this pass. | FR-017, FR-022, FR-025 |
| EC-006 | Actor profile has no display name. | Push uses a generic fallback such as “Someone liked your post”; the payload must not substitute a handle or DID. | FR-024 |
| EC-007 | Installation token rotates while old deliveries are pending. | Deliveries remain associated with their account subscriptions; dispatch uses only the installation's active current token without losing other subscribed accounts. | FR-010, FR-012 |
| EC-008 | User disables push after an event was queued but before dispatch. | Event-time preference remains authoritative for that already-created delivery; future events are not queued while disabled unless the source is deleted, which cancels unsent delivery. | FR-021, FR-027, RULE-003 |
| EC-009 | User has no active devices. | Durable in-app notification is created; no delivery row is required. | BR-001, FR-011 |
| EC-010 | Dispatcher crashes after FCM accepts but before success commit. | A retry may duplicate the push; delivery remains at-least-once and the limitation is observable/documented. | RULE-004 |
| EC-011 | Notification source or required destination is later deleted. | The feed row is retracted, unsent delivery is cancelled, and notification-ID resolution returns the safest available fallback without exposing deleted content. | FR-021, FR-023, FR-026 |
| EC-012 | Two AppView processes claim work concurrently. | Row locking/leases ensure a delivery is normally owned by one worker at a time and expired claims can recover. | FR-012 |
| EC-013 | Reply/comment is deleted after its push was delivered. | Resolution routes to the parent/root post when visible; if that is also unavailable, it falls back to the notifications surface. | FR-021, FR-026 |
| EC-014 | Mentioning post is deleted after its push was delivered. | Resolution falls back to the actor profile when visible, otherwise to the notifications surface. | FR-021, FR-026 |
| EC-015 | Quote post is deleted after its push was delivered. | Resolution falls back to the quoted post when visible, otherwise to the notifications surface. | FR-021, FR-026 |
| EC-016 | Like/repost/follow source action is quickly undone. | Retraction is processed without an artificial grace period; the in-app row disappears and unsent delivery is cancelled, but a push already accepted by FCM may still be seen. | FR-021, RULE-004 |
| EC-017 | Like/repost/follow is recreated after retraction. | The same notification is reactivated and returns to the feed top, but no second push is enqueued. | FR-018 |
| EC-018 | Push is disabled while a delivery is queued. | The queued delivery may still send; only future event eligibility changes. | FR-027 |
| EC-019 | One installation has multiple signed-in accounts. | One current FCM token serves separate account subscriptions and opaque local routing IDs; logout affects only the intended account scope. | FR-009, FR-010, FR-016, FR-029 |
| EC-020 | Delivery reaches its six-hour deadline during backoff or after FCM acceptance. | AppView stops retrying, and provider TTL prevents later delivery beyond the original deadline. | FR-013, FR-028 |
| EC-021 | Notification actor permanently deletes their account/repository. | Caused notifications and unsent deliveries are hard-deleted; stale push resolution returns not-found and Flutter falls back to the notifications surface. | FR-023, FR-030 |
| EC-022 | An FCM token is registered under a new device ID while its old installation has several account subscriptions and unsent deliveries. | The old installation and all of its subscriptions are deactivated, its unsent deliveries are cancelled, the token is rebound to the new installation, and only the currently authenticated account is subscribed there with a new opaque routing ID. | FR-033 |
| EC-023 | A notification arrives while mark-seen is executing. | The acknowledgement stops at the captured snapshot revision; the concurrent notification remains new. | FR-036 |
| EC-024 | The list is prefetched or refreshed in the background. | Newness state is unchanged because notification GET routes are read-only. | FR-036, RULE-007 |
| EC-025 | A notification is retracted before acknowledgement. | It no longer contributes to the count; the marker does not need a per-notification update. | FR-034, FR-035 |
| EC-026 | A previously retracted relationship is genuinely recreated. | The stable notification receives a later newness revision and counts as new, but no second push delivery is created. | FR-018, FR-034 |
| EC-027 | Two devices acknowledge the same account concurrently. | Monotonic upsert semantics retain the greatest captured revision and never move the marker backwards. | FR-036, FR-037 |

## 15. Data / Persistence Impact

- New durable data concepts:
  - Notification events keyed by stable ID, recipient DID, category, actor DID, subject identity, current source event identity, original/latest activity timestamps, indexed timestamp, eligibility snapshot, lifecycle state, retraction timestamp/reason, first-push-enqueued state, and safe fallback routing references.
  - Per-user/per-category preferences containing `scope` and `pushEnabled`.
  - Push installations keyed by Craftsky device ID, containing platform, the current send-capable FCM token, active state, and lifecycle timestamps.
  - Account subscriptions joining an installation to an authenticated recipient DID, containing an opaque installation-specific routing ID, active state, and lifecycle timestamps.
  - Per-subscription push delivery/outbox rows containing notification/subscription references, status, attempts, next-attempt time, absolute six-hour deadline, claim/lease state, provider result classification, and timestamps.
  - A monotonic newness revision on each durable notification's latest genuine activation.
  - One account acknowledgement row keyed by recipient DID, containing the greatest acknowledged newness revision and update timestamp.
- Required uniqueness/invariants:
  - Unique coalesced notification for `(recipient DID, actor DID, canonical category, subject identity)`.
  - Unique preference for `(DID, category)`.
  - One active installation per Craftsky device ID and one active installation owner for an FCM token; cross-device token registration atomically deactivates the old owner without transferring subscriptions.
  - One active account subscription per `(installation ID, recipient DID)` with a unique opaque routing ID.
  - Unique delivery for `(notification ID, account subscription ID)`; reactivation cannot create another delivery.
  - One acknowledgement marker per account DID; concurrent writes use greatest-value semantics and never move it backwards.
- Reference policy:
  - Notification event/source references are retained as text/stable identifiers and must not cascade-delete with mutable index rows; deletion processing updates the notification lifecycle explicitly.
  - Retraction creates a tombstone rather than immediately hard-deleting the row so already-delivered notification IDs remain safely resolvable.
  - Permanent actor account/repository deletion is the exception: caused notifications and unsent deliveries are hard-deleted, and stale IDs resolve as not-found.
  - Display names, post text, images, handles, and other presentation snapshots are not copied into the durable event; current safe views are hydrated at read/send time.
- Fallback routing policy:
  - Reply/comment -> visible parent/root post -> notifications surface.
  - Quote -> visible quoted post -> notifications surface.
  - Like/repost -> visible affected post -> notifications surface.
  - Follow or deleted mention -> visible actor profile -> notifications surface.
- Type-specific response metadata:

| Category | Required references | Rendering/navigation intent |
|---|---|---|
| `follow` | Actor and follow event record | Open actor profile. |
| `like` | Actor, like event record, liked post | Explain who liked which recipient-authored post; open post. |
| `repost` | Actor, repost event record, reposted post | Explain who reposted which recipient-authored post; open post. |
| `reply` | Actor, reply post as event/subject, parent post | Show reply context; open thread focused on reply. |
| `mention` | Actor and mentioning post | Show the post containing the mention; open/focus that post. |
| `quote` | Actor, quote post, quoted recipient-authored post | Show quote context; open quote post. |
| `everythingElse` | Stable notification ID and future versioned event reference | Reserved; no producer in this pass. |

- Schema migration required: Yes, including an additive follow-up migration for notification revisions, the account acknowledgement table, and the active account/revision count index.
- Historical data migration/backfill required: No.
- Existing derived notification data: Not copied; durable history begins at feature enablement.
- Retention: No automatic purge in this pass. A future retention policy must preserve product expectations and delivery diagnostics.

## 16. UI / API / CLI Impact

- UI: No Flutter implementation in this pass; requirements intentionally define future settings, registration, rendering, and open behavior.
- API:
  - Change `GET /v1/notifications` to read durable events while preserving authenticated, paginated AppView conventions.
  - Add read-only `GET /v1/notifications/new-count`, returning `{"newCount": N}` for the authenticated account.
  - Add bodyless `POST /v1/notifications/seen`, returning 204 after advancing the authenticated account's snapshot-safe acknowledgement marker.
  - Add `GET /v1/notifications/{notificationId}` to resolve active or retracted notifications into authorized precise/fallback navigation metadata.
  - Add `GET /v1/notifications/preferences`.
  - Add partial update semantics at `PATCH /v1/notifications/preferences`.
  - Add idempotent `POST /v1/notifications/devices` using the authenticated DID and current `X-Craftsky-Device-Id`; it upserts the installation/token and the current account subscription and returns only the opaque local routing ID.
  - If that token is active under another device ID, the same registration atomically performs the safe rebind defined by FR-033; it does not expose old installation or subscription data in the response.
  - Add authenticated account-subscription removal under `/v1/notifications/devices`; ordinary removal affects only the current account/current installation pairing, while all-session logout removes that account across installations.
  - Extend notification response categories and metadata additively for quotes; Flutter rollout must recognize the category before end-to-end enablement.
- CLI: No user-facing CLI required. Operational inspection/retry commands may be considered during coding design but are not required here.
- Background jobs:
  - Add one AppView-managed push dispatcher lifecycle alongside the Tap consumer and HTTP server.
  - Support graceful cancellation and bounded shutdown.
- Configuration:
  - Add explicit FCM project/credential and push-enabled configuration with production validation.
  - Credential material must use deployment secret mechanisms/standard provider authentication rather than database storage or checked-in files.

## 17. Security / Privacy / Permissions

- Authentication: All preference and device endpoints require the existing Craftsky session and device-ID middleware.
- Authorization:
  - Notification list/preferences are always scoped from the authenticated DID, never a request-supplied DID.
  - New-count and mark-seen derive account scope only from the authenticated DID. Device ID remains required by existing `/v1/*` middleware but does not partition the acknowledgement marker.
  - Notification-ID resolution returns data only when the authenticated DID owns that notification; cross-user IDs use non-enumerating not-found behavior.
  - Installation upsert cannot create a subscription for a DID other than the authenticated account; subscription removal cannot affect another account except through explicit invalid-token installation cleanup.
  - Cross-device token rebinding may deactivate the old installation and cancel its unsent work, but it must not transfer, return, or infer any old account subscription or routing ID.
- Sensitive data:
  - FCM tokens are private routing identifiers and must be accessible only to registration and delivery components.
  - FCM credentials are runtime secrets and must never be persisted in application tables or source control.
  - APIs must not echo tokens after registration.
- Push privacy:
  - No user-generated content or globally resolvable AT Protocol identity/record identifiers are sent to FCM.
  - Actor display name is the only user-derived visible value; use a generic fallback if unavailable or disallowed. The account routing ID is opaque, installation-specific, and revocable.
- Abuse cases:
  - Idempotency and uniqueness prevent replay-driven notification spam.
  - Existing content availability/takedown checks apply during notification hydration; block/mute integration is deferred until Craftsky has an authoritative model.
  - Rate limits should cover preference and device mutation endpoints using existing policy patterns.
  - Invalid-token responses disable dead registrations to avoid repeated provider traffic.

## 18. Observability

- Events/logs:
  - Structured non-sensitive logs for notification persistence failure, delivery claim failure, retry scheduling, permanent failure, device deactivation, dispatcher start/stop, and configuration failure.
  - Include run/operation identifiers and category/status where safe; exclude DIDs, tokens, credentials, payload bodies, and content text.
- Metrics:
  - Notifications created by category.
  - Notifications suppressed by self, scope, missing subject, content availability/takedown, or duplicate idempotency key.
  - Pending delivery count and oldest pending age.
  - Delivery attempt/success/retry/permanent-failure counts by platform and safe failure class.
  - Active installations and account subscriptions by platform.
- Alerts:
  - Sustained oldest-pending age above an operational threshold.
  - Elevated retry/permanent-failure rate.
  - Dispatcher not running while push is enabled.
  - Unexpected growth in pending/terminal queues.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Notification fan-out is coupled incorrectly to indexer transactions. | Lost or duplicate events and inconsistent AppView state. | Transactional persistence, database uniqueness, and replay tests for every producer. |
| RISK-002 | A burst of eligible events produces one push per event. | High-volume posts may create notification noise and delivery load. | Preference controls, queue metrics, bounded claiming, and future aggregation only if real usage justifies it. |
| RISK-003 | FCM at-least-once boundary permits rare duplicate push. | User annoyance. | Unique delivery rows, safe claiming, success terminal state, short provider timeouts, and documented crash-window limitation. |
| RISK-004 | Device tokens or push content leak through telemetry. | Privacy/security incident. | Explicit redaction tests, minimal payload contract, no token echo, and safe failure classification. |
| RISK-005 | New notification categories are unknown to the current Flutter decoder. | App notification page may fail until Flutter is updated. | Coordinate enablement with the later Flutter pass; add forward-compatible client handling before end-to-end rollout. |
| RISK-006 | No retention policy causes unbounded table growth. | Long-term storage and query cost. | Correct indexes and metrics now; define retention before production scale. |
| RISK-007 | Missing subject at ingestion prevents timely recipient derivation. | Some federated/out-of-order activity may never notify. | Make omission observable; consider reconciliation as a future feature without retrospective push. |
| RISK-008 | English-only server copy does not match device locale. | Inconsistent localized experience. | Keep copy generic; add locale-aware device metadata/templates in a future Flutter/AppView pass. |
| RISK-009 | Actor display name changes or becomes unavailable between event and dispatch. | Generic or changed push copy. | Hydrate at dispatch and use “Someone” fallback; notification metadata is fetched fresh after open. |
| RISK-010 | A source is deleted while its push delivery is already in flight. | A stale push may still arrive after the feed row is retracted. | Cancel pending/retry jobs transactionally, keep payload minimal, and resolve delivered IDs through tombstone fallback rather than promising recall. |
| RISK-011 | Multi-account routing opens a notification under the wrong local account. | Authorization failure or confusing navigation. | Opaque per-installation subscription IDs, local account mapping, account-scoped resolution, and non-enumerating not-found behavior. |
| RISK-012 | Permanent actor deletion leaves notification-derived personal data behind. | Privacy and product-lifecycle mismatch. | Hard-delete caused notifications and unsent deliveries; make stale push 404 fallback a client contract. |
| RISK-013 | A provider token reused under a new device ID merges or preserves stale account authorization. | Pushes may route to the wrong installation or reveal another account's activity. | Enforce one active token owner; atomically deactivate the old installation/subscriptions, cancel unsent work, and create only the current authenticated subscription without transfer. |
| RISK-014 | Mark-seen uses wall-clock time or observes notifications committed after its snapshot. | A concurrent notification can be cleared before the user sees it. | Use monotonic revisions and capture the acknowledgement high-water value inside one database transaction before the upsert. |
| RISK-015 | Count visibility drifts from the notification list. | The app shows a badge that cannot be reconciled with visible notifications. | Reuse the list-level active/actor-visibility predicate and add count/list regression coverage. |
| RISK-016 | A list GET mutates acknowledgement state. | Prefetch or retry silently clears the badge. | Keep list/count GETs read-only and require explicit POST `/v1/notifications/seen`. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | This AppView pass includes real FCM delivery infrastructure, not only storage/API scaffolding. | Scope would need to split before coding if provider delivery is deferred. |
| ASM-002 | Android and iOS both register FCM tokens; Firebase routes iOS delivery through configured APNs. | A separate APNs provider/device schema would be required if Flutter uses raw APNs tokens. |
| ASM-003 | Notification eligibility applies only to Craftsky members and indexed Craftsky activity/subjects available at event time. | Broader federation support would require new recipient discovery and reconciliation rules. |
| ASM-004 | Preferences are private AppView settings and are not portable PDS records. | A future portable/private-record design would require migration and lexicon review. |
| ASM-005 | A normal schema migration is acceptable even though no historical data migration/backfill is needed. | Without schema migrations, the persistent design cannot be implemented. |
| ASM-006 | Existing indexed content availability and takedown state are the only visibility inputs available to notification hydration in this pass. | An authoritative block/mute or broader moderation model must be integrated when it exists. |
| ASM-007 | Flutter's planned multi-account store can map an opaque per-installation account-subscription ID to the correct local account before resolving a push. | The routing contract would need revision before Flutter implementation. |
| ASM-008 | On first use, existing active listable durable notifications should all appear as new. | A different rollout baseline would require creating acknowledgement rows during deployment/login instead. |

## 21. Open Questions

- [ ] Non-blocking for requirements: choose the concrete Go FCM client/authentication mechanism and deployment credential format during coding design.
- [ ] Non-blocking for this pass: define future producer and actorless-scope semantics for `everythingElse` before emitting that category.
- [ ] Non-blocking before production scale: define notification and terminal-delivery retention periods.
- [ ] Non-blocking for this AppView pass: define the Flutter rollout sequence for new category decoding, permission prompts, token refresh, and notification-open navigation.
- [ ] Deferred until the social-policy feature exists: integrate authoritative block/mute state into notification eligibility and lifecycle.
- [ ] Deferred: define temporary account-deactivation suppression/restoration behavior.

## 22. Review Status

Status: Approved
Risk level: High
Review recommended: Required
Reviewer: Codex
Date: 2026-07-14
Notes: The account-wide new-count addition uses a monotonic activation revision, read-only count endpoint, explicit snapshot-safe mark-seen operation, and per-account marker. The user explicitly approved document updates and implementation on 2026-07-14. The overall change remains High risk because it stores private device routing data, adds an external delivery integration and long-lived worker, changes notification persistence semantics, and must preserve moderation/privacy and concurrency boundaries.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001 through BR-004; FR-001 through FR-038; NFR-001 through NFR-004; RULE-001 through RULE-007. NFR-005 through NFR-007 remain Should-level but should receive automated coverage.
- Suggested test levels:
  - Unit: eligibility/category precedence, preference defaults/patching, coalescing/reactivation, payload redaction, local account routing, retry classification/backoff/deadline, limit/config validation.
  - Database integration: transactional event/outbox creation and retraction (including forced rollback on deletion), hard deletion, idempotency, tombstone resolution, pagination, monotonic newness revision, account-wide acknowledgement, snapshot races, installation/subscription fan-out, token rotation and cross-device safe rebinding, multi-account logout, and concurrent job claims.
  - HTTP integration: auth/device middleware, new-count/mark-seen, preference/installation/subscription contracts, camelCase/error envelopes, ordinary and all-session logout behavior.
  - Worker integration with fake sender: combined payload, success, retry, six-hour expiry/TTL, permanent token failure, cancellation, multi-worker claiming, crash/lease recovery.
  - Regression: current notification categories, content availability filtering, Tap replay behavior, logout/session behavior, and AppView startup/shutdown.
  - Manual/provider sandbox: one Android and one iOS FCM delivery using non-production Firebase configuration.
- Blocking open questions: None. The user explicitly approved implementation on 2026-07-14.
