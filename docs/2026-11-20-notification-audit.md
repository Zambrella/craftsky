# CraftSky notification audit

20 November 2026

The review followed social and system notifications from record ingestion and private event creation through Activity, badge counts, preferences, push delivery, and notification opens. It covered AppView, Flutter on iOS and Android, and the Svelte web client, including browser service workers and direct-message push. The fixes preserve the public AppView routes and response shapes.

## AppView activity and preferences

| Priority | Finding | Change |
| --- | --- | --- |
| High | PostgreSQL assigns sequence numbers before transactions commit. If a higher revision committed first, the account-wide watermark could mark a lower revision seen before it was ever available to read. A concurrent update could also replace the activation being acknowledged. | Store newness on the event and acknowledge the activation revisions visible in one statement snapshot. A replacement or a later commit stays new. Migration `000129` backfills existing seen state, indexes active unseen rows and accepts legacy acknowledgements during the AppView rollout. |
| Medium | Two devices could simultaneously set scope and push for a category with no saved row. The second write could restore defaults over the first device’s change. | Serialize PATCH operations for the same account before reading saved preferences. Partial patches now merge both fields. |
| Medium | A post notification job could hold an unrelated mention behind a missing reply parent or quote. An unresolved dependency could outlast the push delivery window. | Resolve independent recipients immediately when the missing reference has a different known owner. Keep the dependency for recipients whose category may change and preserve reply, quote, then mention precedence. |
| Medium | A future-dated PDS record could extend the six-hour push window beyond provider limits. | Bound the deadline by the activation time while retaining the original activity date and the cutoff for old backfill. |
| Optimisation | Event-time eligibility read preferences, relationships and, conditionally, follow state in separate queries. The Activity list and badge count also maintained their own copies of the visibility policy. | Use one SQL statement for event-time eligibility and share Activity/count visibility rules. The unread index covers the active unseen subset rather than every notification. |

Entry points and regression coverage are in [event newness](../appview/internal/api/notification_newness.go), [preference PATCH](../appview/internal/api/notification_preferences.go), [publication projection](../appview/internal/index/post_notification_projection.go), [event eligibility](../appview/internal/notifications/service.go), and [migration 000129](../appview/migrations/000129_notification_seen_per_event.up.sql).

## AppView push delivery

AppView dispatches social/system push and chat push from separate durable queues. The review checked device and owner fencing, claim and retry behavior, current visibility, expiry, routing data, and provider error classification.

| Priority | Finding | Change |
| --- | --- | --- |
| High | The social dispatcher acquired an owner-fenced database connection, then asked the pool for another connection to check eligibility, read previews or finalize the send. Concurrent workers could exhaust the pool and wait on themselves. | Run the short database transactions on the connection holding the owner fence. Release each transaction before the provider request. A real PostgreSQL test exercises delivery and finalization with a single connection. |
| High | A queued delivery from a previous registration could become eligible if the same account and installation were reactivated before the worker processed it. | Retain the last deactivation time and reject older social and chat jobs. Verify that a fresh job still sends after registration resumes. |
| High | A hidden or taken-down actor could still appear in push even though Activity and its badge excluded the actor. | Share the active actor-moderation predicate across all three paths and recheck at dispatch. Existing preview eligibility continues to govern which post text may appear. |
| Medium | A slow preview could consume the remaining send window. Cancellation from shutdown or a failing sibling could also prevent recording a provider acceptance. | Recheck the clock before sending and cap the request by both the lease and delivery deadline. Finalize a provider result with a bounded context that survives shutdown cancellation. |
| Medium | Some temporary Firebase errors became permanent failures. Quota errors could retry too quickly, and one chat recipient’s error could delay unrelated chat deliveries. | Classify transient SDK and HTTP errors, honor Firebase’s retry guidance and keep retries scoped to the affected delivery. [Firebase error codes](https://firebase.google.com/docs/cloud-messaging/error-codes) |

See [social dispatch](../appview/internal/push/dispatcher.go), [Firebase sender](../appview/internal/push/firebase_sender.go), [device registration](../appview/internal/api/notification_devices.go), and [chat delivery](../appview/internal/chat/delivery.go).

## Flutter

| Priority | Finding | Change |
| --- | --- | --- |
| High | The notification host treated unresolved account initialization like sign-out. A launch notification could be dropped or checked against an unhydrated registry. | Keep the pending open until readiness resolves. Check the binding again after selecting or restoring the recipient account. |
| High | A foreground notification whose binding was invalid or removed could still show a banner and trigger a refresh on the current account. | Gate banner and refresh effects on valid recipient bindings. Account and request ownership are checked across asynchronous work. |
| Medium | The host refreshed permission state on resume without resuming registration. Retained inactive accounts could also miss token renewal after startup. | Reconcile registrations when the app resumes and observe onboarding readiness for retained sessions so each eligible account can register. |
| Medium | Older count requests could overwrite newer results. The badge refresh after a push open could suppress a count on the account selected while the request was in flight. | Version count requests and bind the post-open adjustment to the originating account. |
| Medium | Native push for a mention or quote inside a reply could open the reply itself as a thread root and reach an invalid comments endpoint. | Carry the known root on native push and open that thread with the source reply focused. The field is optional for older payloads. |
| Medium | Pagination could restore rows hidden by a local block applied while a request was running. A fully filtered page could also leave the user with no way to reach the next page. | Merge against current account and relationship state and allow pagination through filtered-out pages. |
| Medium | Fast changes to notification preferences could finish out of order, reverting a newer setting or restoring the wrong value when a save failed. | Serialize writes and reconcile optimistic state with the last confirmed preferences for that account. |

See [notification runtime](../app/lib/notifications/services/notification_runtime.dart), [device registration](../app/lib/notifications/services/notification_registration_coordinator.dart), [Activity state](../app/lib/notifications/providers/notifications_provider.dart), [notification badge](../app/lib/notifications/providers/notification_new_count_provider.dart), and [preferences](../app/lib/notifications/providers/notification_preferences_provider.dart).

## Web

| Priority | Finding | Change |
| --- | --- | --- |
| High | The Firebase worker could focus an existing tab without navigating when a notification was clicked. During a cold open, Activity could mount and acknowledge the wrong account before the subscription resolved. | Send web-only data messages and let the worker display the alert and open a validated same-origin route. Gate destination rendering and acknowledgement until the owning account is selected. Stale or unknown bindings fail closed. |
| Medium | Returning from a destination could lose the loaded page depth and scroll position. Retrying an Activity load could bypass the badge acknowledgement. | Restore page depth through the shared page history, fetch the latest records, abort abandoned reads, and send retries through the same load and acknowledgement path. |
| Medium | A nested reply notification could land on the parent after the thread route redirected. Reply wording could also describe the source comment instead of the thing the recipient wrote. | Open the canonical thread root with the new reply as the focus and derive wording from the subject’s content role. The separate actor link still opens the actor’s profile. |
| Medium | Browser registration changes and stale foreground completions could leave subscription state or badge data out of sync with the current browser/account. | Serialize registration mutations across supported tabs and validate the recipient before refreshing Activity or badges. |
| Medium | Activity badge polling waited for chat, so a slow chat read could hold up the social count. Older chat polls could also overwrite newer data. | Reconcile the two independently and discard stale poll results. Refresh Activity and its badge on valid foreground push. |
| Low | Browser permission and renewal failures could leave settings with an obsolete binding or unclear retry feedback. | Clear bindings rejected by AppView, roll back a new server subscription when local persistence fails, and keep retry feedback attached to the relevant settings action. |

See the [service worker](../web-client/src/service-worker/index.ts), [push client](../web-client/src/lib/features/push/BrowserPush.svelte), [Activity page](../web-client/src/lib/features/social/Activity.svelte), and [browser regression tests](../web-client/tests/authenticated-notification-badge.spec.ts).

## Verification and rollout

| Surface | Local verification |
| --- | --- |
| AppView | The full Go suite passed under the race detector with disposable PostgreSQL 16 and MinIO. Focused race tests also passed for the final native push payload and migration bridge. These covered migration upgrade and downgrade, concurrent seen and preference mutations, recipient isolation, publication replay, relationship visibility and lifecycle cleanup. Delivery tests used provider doubles for queue, lease, shutdown and retry behavior. The pinned chat SQL generator completed. |
| Flutter | 441 notification, routing, settings and chat tests passed. A further 15 focused tests passed after the native thread-root change, including mention and quote routing. Targeted static analysis passed. No running debug process was available for hot reload. |
| Web | 619 Vitest tests and 23 authenticated Chromium Playwright tests passed. Svelte check, ESLint, and the production build passed. The worker tests exercise display and click behavior with simulated messages. |
| Documentation | The change catalogue was regenerated and all 19 catalogue tests passed. |

Live Firebase, APNs, Android and installed-PWA delivery and tap behavior remain unverified. All tests and data changes were local.

The full Go invocation was `go test -p=1 -race -count=1 -timeout=30m ./...` with Go 1.27.4. The disposable PostgreSQL instance used a larger lock table for isolated schemas. Disk flushes were disabled for the later packages to speed up fixture creation.

The browser service worker should go live before AppView starts sending web data-only payloads. Browsers need to activate the updated worker to display those messages. The existing native payload version remains valid and gains a thread root for mention and quote opens when available. [Firebase web message handling](https://firebase.google.com/docs/cloud-messaging/web/receive-messages)

Migration `000129` retains the old seen watermark and indexes while the preceding AppView binary is still serving. The migration holds a table lock during backfill and index creation. A binary-only rollback can temporarily re-show newer acknowledgements until Activity is acknowledged again. The down migration preserves the seen prefix and may re-show previously acknowledged rows after a gap.

## Remaining tradeoffs

- A successful Activity visit acknowledges the whole account. This includes other categories and notifications that commit before the leave acknowledgement, even when a filter or pagination kept them off screen. That is the existing product behavior.
- Opening a native social push can still subtract one from the displayed badge until the next refresh. That local adjustment is scoped to the recipient account and can differ from the server count if another client already acknowledged the event.
- Push delivery remains at least once across acknowledgements to Firebase. A process or database failure after provider acceptance can still leave the result unknown and trigger another attempt.
- An alert already queued by Firebase or the operating system may still arrive after preferences or relationships change. Native notification messages also retain Firebase’s provider-side collapse behavior. [Firebase message collapse](https://firebase.google.com/docs/cloud-messaging/customize-messages/collapsible-message-types)
- Social notification rows and their delivery history have no general age-based pruning in the reviewed paths. Account lifecycle cleanup removes associated rows. A separate retention decision would need to distinguish Activity history and terminal queue records.
- Moderation enqueues durable delivery work even when push is disabled. The dispatcher applies the preference when it attempts delivery, so enabling push before the first attempt can permit a queued alert. This is the defined moderation policy. [Moderation delivery implementation](changes/2026-09-09-moderation-strikes/05-implementation-plan.md#L182)
