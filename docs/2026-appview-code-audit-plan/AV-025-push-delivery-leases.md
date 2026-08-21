# AV-025 — Push delivery leases

- **Included finding:** AV-025, Medium — Push lease geometry allows duplicate provider sends
- **Priority/order:** API correctness; complete after the duration-contract changes and before treating push as launch-ready
- **Status:** Planned
- **Audit source:** [AV-025](../2026-08-12-appview-code-audit.md#av-025--push-lease-geometry-allows-duplicate-provider-sends)

> **Superseding product decision (2026-08-20):** Android now uses the same
> standard notification-plus-data contract as iOS. The operating system owns
> background and terminated presentation; Flutter retains foreground effects
> and authenticated open routing only. The product explicitly accepts FCM's
> notification-message collapsing, possible duplicate OS presentation after an
> ambiguous retry, and possible late display after logout/lifecycle change.
> The data-only/local-presentation/durable-client-dedupe steps below document
> the earlier audit design and are no longer normative.

## Shared implementation strategy

Change the dispatcher from “claim a large batch, then send it serially” to bounded just-in-time claims. A worker may claim a delivery only when it owns an available send slot, and the provider deadline must fit wholly inside the persisted lease with a finalization margin. Continue to use lease-token compare-and-set updates, but state the external guarantee accurately: PostgreSQL and Firebase cannot provide an atomic exactly-once boundary, so the remaining post-acceptance crash window is at-least-once. Provider collapse fields are semantic replacement controls, not arbitrary idempotency keys; duplicate-presentation mitigation therefore uses platform-specific payloads, stable client display identity, and bounded `notificationId` deduplication.

The pre-production window is used deliberately: redefine `PUSH_BATCH_SIZE` as a per-poll work budget, add explicit concurrency and finalization-margin settings, and reject old unsafe configurations instead of preserving their accidental serial semantics.

## Finding closure

### AV-025 — Push lease geometry allows duplicate provider sends

`push.Dispatcher.claim` currently gives up to 100 rows the same 60-second expiry, while `ProcessBatch` can spend up to 10 seconds on each serial send. A row near the back can therefore be reclaimed before its first attempt. Fencing prevents a stale final database update, but cannot undo a notification already accepted by Firebase.

This update closes AV-025 by establishing these invariants:

- A delivery never waits behind a full serial batch while already leased.
- The provider context deadline is earlier than both `deadline_at` and `lease_expires_at` by a defined margin.
- Startup rejects `lease duration <= send timeout + finalization margin` and other unsafe geometry.
- The irreversible send is preceded by a fresh ownership, token, installation, preference, relationship, and retraction check.
- A worker cannot begin a send when its final ownership check shows an expired/reclaimed lease, and stale results cannot finalize a replacement claim. If a lease expires during an already-started provider call, overlap remains an explicit at-least-once case.
- `push_deliveries.id` is never projected into Android `collapse_key`, APNs `apns-collapse-id`, or described as provider idempotency.
- An Android notification for which each logical event matters is sent as a data message without a `notification` object and without `collapse_key`, then rendered by the app. FCM notification messages are not used for this mode because they are always collapsible and ignore a supplied `collapse_key`.
- Android local presentation uses a stable, collision-safe display tag keyed by the durable `notificationId`, and the client keeps a bounded recent-`notificationId` dedupe store so an ordinary retry does not create a second banner, sound, navigation effect, or visible tray entry.
- APNs omits `apns-collapse-id` for unique-event alerts. It sets a stable semantic collapse ID only for an explicitly approved “replace the older state” category, never once per delivery.
- The unavoidable provider-accepted/database-not-finalized window is observable and documented as at-least-once.

## Scope and design decisions

### In scope

- Claim scheduling, concurrency, persisted lease expiry, provider deadlines, token fencing, and result finalization.
- Firebase Android/APNs payload construction with an explicit unique-event versus intentional-replacement policy.
- Push configuration, metrics, and concurrency/fault tests.
- Required Android app-mediated display for unique-event data messages, stable display tags, bounded client-side suppression keyed by the existing `notificationId`, and local-notification open routing.

### Out of scope

- Notification fan-out, category, preference, or routing-URL changes.
- Claiming exactly-once delivery across PostgreSQL and Firebase.
- Changing the `/v1/notifications/*` JSON contract.

### Decisions

1. Use a bounded worker loop. Each slot claims one due row, processes it, then may claim another until the batch budget is consumed.
2. Carry the exact persisted `lease_expires_at` in `claimedDelivery`; do not reconstruct it later from an in-memory clock.
3. Define `attemptDeadline = min(delivery deadline, lease expiry - margin, now + send timeout)`. If no useful window remains, release or renew through a lease-token-guarded update without calling the provider.
4. Keep privacy/eligibility checks in one `processClaim` capability so concurrent workers cannot drift apart.
5. Make delivery semantics typed and closed, for example `UniqueEvent` and `ReplaceLatest(ReplacementClass)`. The server derives this value from an audited category policy; neither an HTTP field nor a raw database/provider string may select a collapse key. All current event notifications default to `UniqueEvent` unless product requirements explicitly say an older pending state is disposable. Any Android replacement classes form a static allowlist of at most four total values per installation and may not be parameterized by delivery, notification, actor, thread, or object IDs.
6. For Android `UniqueEvent`, send an FCM data message: omit both `messaging.Message.Notification` and `AndroidConfig.CollapseKey`, and include bounded display copy, routing facts, payload version, and `notificationId` in `Data`. This is the only way in this design to avoid FCM's notification-message collapsing for events where each pending item matters. Do not generate one collapse key per delivery: FCM permits at most four active keys per registration token, does not define which keys survive when that limit is exceeded, and notification messages ignore custom `collapse_key` anyway.
7. Render Android unique-event messages through an app-owned presenter. Use the full `notificationId` as the native Android string tag with a fixed type-scoped integer ID so retrying the same logical notification updates the same visible entry while distinct IDs remain distinct. If the Flutter bridge cannot expose a string tag, add a native bridge or collision-aware bounded mapping; do not truncate/hash UUIDs into an unchecked integer namespace. Before foreground banners, local tray presentation, and notification effects, consult a bounded, persisted TTL/LRU dedupe store keyed only by validated `notificationId`. Stable tag plus client dedupe is mitigation, not an exactly-once transaction with the operating system.
8. Treat APNs independently. A unique event omits `apns-collapse-id`; an intentional replacement may use an allowlisted, stable semantic ID such as a specific replaceable state stream, never `push_deliveries.id` or `notificationId`. If no current category has documented replacement semantics, send no APNs collapse ID. Client dedupe suppresses duplicate app-handled effects, but an APNs alert already rendered by the OS can still duplicate after an ambiguous provider retry; keep that at-least-once residual explicit.
9. Carry the required recipient DID and an optional, boundary-validated actor DID in each claim. System categories such as `instagramMatch` legitimately have no actor and must never derive a lock key from an empty DID. Always acquire the recipient owner-lifecycle effect fence; for actor-backed categories, acquire both fences in canonical DID order (one fence when equal) before the final delivery-state check, then hold the acquired fences through the provider call and local finalization while the process lives. This guarantees no new send begins after a completed transition and makes a normally progressing send delay that transition. It does not cancel a provider-accepted send after an AppView crash releases connection-scoped fences; late delivery is an explicit cross-system residual governed by the client presentation contract.

## Unified implementation plan

1. Extend `push.DispatcherOptions` in `appview/internal/push/dispatcher.go` with bounded concurrency and a finalization margin. Make the intentional breaking constructor `NewDispatcher(pool, sender, options) (*Dispatcher, error)` (or the equivalent full existing dependency signature plus error) so invalid geometry cannot silently produce a dispatcher.
2. Extend `claimedDelivery` with the actual lease expiry, validated `notificationId`, typed delivery semantics, required recipient DID, and optional actor DID needed for lifecycle fencing. Do not add a generic provider delivery/collapse key. Parse a present actor at the row-decoding boundary into a typed DID and fail the claim closed if that non-null value is malformed; preserve SQL `NULL` as absence for system notifications.
3. Split `claim` into narrowly named operations: expired-lease recovery, one-row transactional claim, and one-claim processing. Include the AV-027 terminal `rows.Err()` rule if a row iterator remains.
4. Replace the serial loop in `ProcessBatch` with a bounded worker group that stops at `BatchSize`, stops claiming on cancellation, and never queues already-leased work behind a semaphore.
5. Acquire the recipient owner-lifecycle effect fence and, when a valid actor is present, the actor fence in canonical DID order. Then recheck ownership and all privacy/routing state immediately before `Sender.Send`; calculate the bounded attempt deadline from the persisted timestamps. Hold the acquired fence set through provider result and token-fenced finalization while the worker lives. On restart, the durable delivery/lease state preserves ambiguity; do not infer that disappearance of the advisory lock canceled Firebase processing.
6. Keep every final update guarded by delivery ID, lease token, unexpired lease, active subscription/installation, and unchanged token.
7. Replace the generic delivery-key proposal with typed provider construction in `appview/internal/push/sender.go`, `payload.go`, and `firebase_sender.go`. Switch exhaustively on the boundary-validated installation platform and fail closed on unknown values rather than falling back to a notification message. For Android unique events, build a data-only `messaging.Message` with no notification object or collapse key; put validated/bounded display copy and existing routing facts in data. An approved Android replacement also remains data-only and may set only one of the at-most-four static semantic collapse keys; do not use an FCM notification message when a custom replacement group matters because its key would be ignored. For APNs unique events, omit `apns-collapse-id`. Permit APNs replacement only through an allowlisted semantic class whose product contract intentionally discards older pending state. Never accept an arbitrary provider string or use a delivery/notification UUID.
8. Add Android app-mediated presentation in `app/lib/notifications/services/firebase_notification_service.dart`, `firebase_notification_background_handler.dart`, and a focused presenter abstraction. Parse the data-only display envelope once, validate `notificationId` and bounds, perform account/lifecycle revalidation where required, and display with native `(tag = notificationId, fixed type-scoped id)` identity. Route local-notification taps into the existing `NotificationOpenAttempt` path without persisting `notificationId` as routing authority.
9. Add a small persisted bounded TTL/LRU delivery-dedupe store behind a cross-isolate-safe interface used by foreground, background, and local-open paths. Each validated `notificationId` has separate compare-and-set stages such as `presented`, `foregroundEffectEmitted`, and `opened`, so recording local display does not suppress the first legitimate tap while duplicate callbacks at the same stage are ignored. Serialize foreground/background-isolate updates through storage that provides the required atomicity; an in-memory map or uncoordinated preferences writes are insufficient. Distinct IDs remain independently presentable; capacity eviction may permit a much-later duplicate but must never suppress a new ID. Define a safe minimum above four, entry/age ceilings, deterministic eviction, corruption recovery, and account-partition/clearing behavior. Test the presentation-versus-record crash boundary and retain the residual honestly rather than describing the cache as atomic with the OS.
10. Wire `PUSH_CONCURRENCY`, the lease margin, and client dedupe bounds through `appview/internal/app/config.go`, `appview/internal/app/deps.go`, Flutter configuration/constants, environment examples, and Compose/deployment configuration. Coordinate positive/relationship validation with AV-030.
11. Extend privacy-safe observations with reclaimed-lease count, insufficient-window releases, attempt latency, result class, and low-cardinality delivery semantics/platform. Never emit tokens, message content, provider error strings, `notificationId`, replacement values, or high-cardinality delivery IDs.
12. Document both presentation residuals. Android data-only/local presentation plus stable tag/dedupe prevents ordinary retries from creating unrelated or duplicate entries, but it cannot make FCM/device delivery exactly once. APNs OS-rendered alert duplicates remain possible after an ambiguous accepted send unless a category intentionally has replacement semantics. If presentation after actor/recipient terminal/departure completion is forbidden, the app-mediated path must revalidate the current active account plus notification state before displaying; an OS-rendered notification payload cannot be recalled after provider acceptance. Make delayed-delivery and duplicate-retry tests required.

## Migration, reconciliation, and operations plan

No AppView schema migration is expected: the existing notification identity/routing fact and `lease_expires_at` are sufficient for unique-event delivery, and the delivery UUID is deliberately not provider collapse metadata. If implementation introduces an intentional replacement class that must be persisted, add a constrained semantic field in the next-numbered migration rather than overloading an ID or editing historical migrations, and coordinate numbering with the other audit updates. The bounded Flutter dedupe store is client-private cache state, not server authority.

For a shared development database, allow legacy leases to expire or run only the existing token-fenced recovery path before enabling the new dispatcher. Do not mark leased rows succeeded because prior provider acceptance is unknowable. Existing pending/retry rows can be processed under the new geometry; successful/terminal rows are not replayed.

Operations must document the new setting semantics and alert separately on queue/storage failure, provider retry, lease reclaim, and insufficient remaining lease window.

## API, client, configuration, and operational impact

- The `/v1/notifications/*` API and canonical JSON envelope do not change.
- `PUSH_BATCH_SIZE` intentionally changes from serial claim size to per-poll budget; new concurrency/finalization settings must be updated in examples, Compose, deployment secrets/config, and startup tests together.
- Android unique-event notifications become data-only FCM payloads with no `notification` object and no `collapse_key`; Flutter owns their foreground/background local presentation and open routing. Existing routing facts and `notificationId` remain stable.
- Flutter implements bounded persisted recent-`notificationId` suppression and collision-safe Android display tags. This prevents ordinary duplicate retries from creating multiple app-rendered presentations but does not erase the at-least-once provider/crash residual.
- APNs unique-event alerts omit `apns-collapse-id`. Only an explicitly documented semantic replacement class may set it; duplicate OS-rendered alerts after an ambiguous retry remain possible and documented.
- Operators gain separate lease-window/reclaim observations and must not interpret database fencing as exactly-once delivery.

## Security, failure, and race considerations

- Lease renewal/release must compare the current token so a stale worker cannot extend a newer lease.
- Do not wait for a concurrency slot after claiming; acquire the slot first.
- Provider timeout or lease expiry during an already-started call is ambiguous and may follow acceptance. A second worker may send after reclaim; retain retry availability, use stable `notificationId` display identity and bounded client suppression where the app renders, measure the overlap, and never claim exactly-once. Do not trade unrelated-message loss for apparent deduplication by inventing per-delivery collapse keys.
- FCM “non-collapsible” means the service does not intentionally replace the message by collapse key; it is not a delivery guarantee. Provider/device storage limits, TTL, token invalidation, operating-system policy, and network failure can still lose or delay a message.
- Android data-only delivery requires the app/background handler to execute before local presentation and therefore has different force-stop, battery, and background-restriction behavior from provider-rendered notification messages. Verify and document that product tradeoff on supported Android versions; do not silently fall back to a collapsible notification payload.
- Notification-message payloads must not enter the Android unique-event builder: FCM always collapses them and ignores custom `collapse_key`. Tests inspect the actual `messaging.Message` shape, not only an AppView enum.
- Validate and bound all data-only display fields before local presentation. Keep `notificationId` out of logs/metrics and use it only as a delivery/display-dedupe identity, never as authorization or routing authority.
- APNs collapse IDs are separate platform semantics. An allowlisted replacement ID must group only messages whose older pending state is intentionally disposable; unique-event paths omit it, and app-side dedupe cannot retroactively suppress an alert already displayed by iOS.
- AppView crash after Firebase accepts a call releases database/advisory fences while the provider may still deliver later. Actor/recipient lifecycle transitions may then complete; the server cannot recall the message. Do not promise cross-system cancellation—apply the selected data-only/client-revalidation policy or explicitly accept and test late presentation.
- Cancellation prevents new claims and bounds in-flight provider calls. No provider call starts after the worker context is done.
- Use a consistent time authority for lease comparisons, or test application/database clock skew explicitly.
- Preserve current mute, block, preference, event-state, installation, and token-rotation gates.
- Use canonical actor/recipient DID order when an actor exists, then the global owner/work/session/transaction order defined by the account-lifecycle plan. Never acquire an owner fence for an empty/system actor. Push cleanup/departure must not acquire its database/lease locks and then wait for either real owner fence in reverse.

## Unified test plan

1. **Unit:** Test option relationships, deadline selection, zero remaining lease, and the typed provider-payload matrix. Assert Android `UniqueEvent` has data plus bounded display/routing fields but no `Notification` and no `CollapseKey`; the closed Android replacement-key set has no more than four distinct static values and never uses a notification payload; APNs unique delivery has no `apns-collapse-id`; only allowlisted intentional replacement produces a semantic platform collapse value; and neither `push_deliveries.id` nor `notificationId` is ever copied into provider collapse metadata.
2. **Database integration:** Assert a just-in-time claim stores/returns one exact expiry and every transition remains lease-token fenced.
3. **Concurrency:** Run two dispatchers with a blocking sender and deliberately short leases. Assert queued rows are not pre-leased and a normally progressing claim receives only one concurrent provider call. Then deliberately let a lease expire during the provider call, reclaim it, and prove duplicate attempts are treated as permitted at-least-once behavior while only the current token can finalize.
4. **Fault injection:** Cover database failure before send, provider timeout, database failure after success, cancellation, token rotation, preference disablement, relationship change, and retraction between claim and send.
5. **Race:** Run the push package under `go test -race` with concurrent dispatcher/sender fakes.
6. **Lifecycle barriers:** For actor-backed categories, pause immediately before provider send and independently complete recipient and actor departure/terminal deletion; after either transition, release the worker and assert it makes no provider call. Conversely, a normally progressing already-started send holds both shared fences and neither transition reports completion until finalization returns. Exercise opposite actor/recipient pairs concurrently to prove canonical ordering avoids deadlock. Separately claim and send a system `instagramMatch` notification with a null actor, proving it takes only the recipient fence and never constructs an empty-DID owner key. Finally let the provider accept a send, crash AppView so fences disappear, complete a lifecycle transition, and release delayed delivery; assert the chosen late-presentation policy suppresses/revalidates it or the documented contract explicitly permits it.
7. **More-than-four regression:** Queue at least five distinct unique-event notifications for one registration token. A provider fake that models FCM's four-active-collapse-key limit must observe five Android data-only requests with no collapse keys, and the client harness must present five distinct stable tags/IDs with no unrelated replacement. Where a Firebase device test is practical, repeat with the device temporarily unable to receive, while treating the result as integration evidence rather than a universal delivery guarantee.
8. **Duplicate retry presentation:** Deliver the same Android data message/`notificationId` twice, including one retry after an AppView accepted-send/database-finalization failure. Assert one logical foreground effect and one visible local-notification identity, a stable native tag, and bounded persisted dedupe across service reconstruction and foreground/background-isolate contention. Tap the displayed notification and prove the first open still routes once even though presentation was recorded, while a duplicate open callback does not navigate again. Then deliver a distinct ID and prove it still presents. Add a presentation/checkpoint crash-boundary test documenting the remaining non-atomic client/OS residual.
9. **APNs semantics:** Assert unique alerts omit collapse ID and two distinct notifications cannot replace one another through AppView metadata. For an explicitly replaceable fixture, assert only the stable semantic class is used. Duplicate an OS-rendered unique alert and document that provider-accepted retry remains at-least-once because client dedupe cannot retract the first presentation.
10. **End to end:** Against a Firebase test project, inspect the actual Android payload mode for unique events and exercise local presentation/open routing. Do not interpret a successful run as provider exactly-once proof; the deterministic fake, client dedupe tests, and explicit crash residual are the acceptance authority.

## Traceability and acceptance criteria

### AV-025

- **Implementation seams:** `internal/push/dispatcher.go`, `internal/push/sender.go`, `internal/push/payload.go`, `internal/push/firebase_sender.go`, `internal/app/config.go`, `internal/app/deps.go`, `app/lib/notifications/services/firebase_notification_service.dart`, `firebase_notification_background_handler.dart`, and focused Flutter presenter/dedupe-store files.
- **Verification seams:** `internal/push/dispatcher_test.go`, `internal/push/firebase_sender_test.go`, AppView configuration tests, and required Flutter foreground/background/local-presentation, open-routing, and dedupe-store tests.

- [ ] No delivery waits behind a serial full-size batch while already leased.
- [ ] Every provider deadline is strictly earlier than persisted lease expiry by the configured margin.
- [ ] Unsafe lease/timeout/concurrency geometry fails during startup with the relevant key named.
- [ ] Two dispatchers cannot concurrently send a normally progressing claim.
- [ ] Stale results cannot mutate reclaimed rows.
- [ ] `push_deliveries.id` and `notificationId` never populate Android `collapse_key`, APNs `apns-collapse-id`, or a generic provider-idempotency field.
- [ ] Android unique-event messages are data-only, contain no notification object or collapse key, and are locally rendered under a collision-safe stable tag derived from the full `notificationId`.
- [ ] Any Android intentional-replacement path is data-only and uses one of at most four static, audited semantic keys; it cannot derive a key from a delivery, notification, actor, thread, or object ID.
- [ ] A test with more than four distinct pending deliveries for one token proves all five use the non-collapsible unique-event path and no unrelated notification is replaced by collapse-key exhaustion.
- [ ] Duplicate retry tests prove bounded persisted `notificationId` dedupe suppresses repeat app-rendered presentation/effects while a distinct ID still presents; the client/OS crash boundary remains explicitly at-least-once.
- [ ] APNs unique-event alerts omit collapse ID; APNs collapse IDs exist only for allowlisted intentional semantic replacement and never per delivery.
- [ ] Lease expiry during an in-flight call has an explicit overlap test and is documented as at-least-once, not prevented by a stale pre-send check.
- [ ] While AppView remains alive, actor-backed sends fence both actor and recipient departure/terminal completion, with deterministic barriers and deadlock-free DID ordering tests; null-actor system notifications send under the recipient fence alone.
- [ ] Provider acceptance followed by AppView crash, lifecycle completion, and delayed delivery is explicitly tested; the selected data-only/client-revalidation policy suppresses presentation, or the product contract explicitly permits the late message.
- [ ] All current privacy, relationship, installation, and token guards still pass.
- [ ] Ambiguous post-send failures are documented and observable as at-least-once behavior.
- [ ] PostgreSQL integration and race tests pass.

## Dependencies and coordination

- Coordinate duration and relationship validation with **AV-030**.
- Apply the row-iterator rule from **AV-027** if the claim query still iterates results.
- Run through the release gates in **AV-012/AV-013/AV-033/AV-036**.
- Align `claim/process/send` capability seams with **AV-037** rather than moving them twice.
- Reuse owner-effect fences and the global lock order from **AV-002/AV-003/AV-006/AV-007**; always fence the recipient, acquire the optional actor second according to canonical DID order, and ensure notification/subscription cleanup never acquires locks in reverse.

## References

- [API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [Notifications MVP coding plan](../changes/2026-05-29-notifications-mvp/04-coding-plan.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Go `context` documentation](https://pkg.go.dev/context)
- [Firebase: Non-collapsible and collapsible message types](https://firebase.google.com/docs/cloud-messaging/customize-messages/collapsible-message-types)
