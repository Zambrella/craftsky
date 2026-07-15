# Requirements: Flutter Push Notifications

## 1. Initial Request

Continue the completed AppView push-notification work in a separate Flutter client slice. Add Firebase Cloud Messaging support for Android and iOS, handle notifications received while the app is foregrounded and notifications opened from background or terminated states, update the notification list to consume the durable AppView contract, expose notification preferences, and use the stream-based/provider-neutral pattern proven in `/Users/douglastodd/Projects/stash_hub/app` so Firebase-specific types remain isolated. One narrowly scoped AppView payload follow-up is included so iOS background notifications request the standard OS sound; notification eligibility, persistence, routing, and API contracts remain unchanged.

The user confirmed that both native applications should use the identifier `social.craftsky.app`. A Firebase project named `Craftsky` with project ID `craftsky-app` has been created, and Android and iOS applications with that shared identifier have been registered.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/main.dart` and `app/lib/bootstrap.dart` own process/bootstrap initialization and global error handling.
  - `app/lib/app.dart`, `app/lib/router/router.dart`, and `app/lib/router/app_shell.dart` own the ready application, typed navigation, and Notifications destination.
  - `app/lib/auth/providers/auth_session_provider.dart` is the single-account authentication source of truth; `onboarding_status_provider.dart` exposes whether that account completed onboarding.
  - `app/lib/notifications/` contains the existing API client, repository, models, Riverpod pagination provider, list page, and notification rows.
  - `app/lib/shared/device/device_id_provider.dart` persists the installation-scoped Craftsky device ID independently from the session.
  - `appview/internal/api/notification_devices.go`, `notification_newness.go`, `notification_resolution.go`, and `notifications.go` define the implemented client contract.
  - `appview/internal/push/firebase_sender.go` builds the combined FCM message. It currently supplies title/body and APNs expiration but does not request the standard iOS notification sound.
  - `app/android/app/build.gradle.kts` and `app/ios/Runner.xcodeproj/project.pbxproj` currently use different legacy identifiers and have no Firebase configuration.
- Existing patterns:
  - Riverpod providers and narrow repository interfaces separate plugins/networking from presentation logic.
  - The session `Dio` already injects the Craftsky bearer token and `X-Craftsky-Device-Id` for authenticated `/v1/*` calls.
  - Secure storage already holds the Craftsky session and stable device ID; sensitive values are redacted from logs and error reporting.
  - Typed `go_router` routes already support notification destinations: profiles, post threads, and the Notifications surface.
  - User-facing copy is localized through ARB files and generated `AppLocalizations` APIs.
- Current behavior:
  - The Flutter app has no `firebase_core` or `firebase_messaging` dependency, native Firebase configuration, permission flow, FCM token registration, token-refresh listener, foreground stream, background-open stream, cold-start notification handling, or Android notification channel.
  - The notification decoder predates the durable AppView response: it has no stable notification ID, reference-availability model, `quote`, or forward-compatible unknown category.
  - The notification page lists five legacy categories and navigates directly from hydrated list data. It does not fetch or clear `newCount`, show a navigation badge, or expose preferences.
  - The app currently persists one signed-in account, not a multi-account session collection.
- Constraints discovered:
  - Push payloads contain only `notificationId`, `type`, and opaque `accountSubscriptionId` data plus generic visible copy. The client must fetch current authorized metadata from AppView before navigating.
  - Firebase background handlers for Flutter must be top-level, non-anonymous, and retained with `@pragma('vm:entry-point')`; they cannot update application UI state.
  - Combined notification-and-data messages display through the OS while backgrounded. Foreground notification messages do not display automatically and need app-owned presentation.
  - iOS delivery additionally requires Push Notifications and Remote notifications capabilities plus an APNs authentication key uploaded to Firebase. The client should request alert and sound authorization, but not app-icon badge authorization because icon badges are outside scope.
  - Android background delivery needs one user-visible notification channel with standard importance, sound, and vibration. Because the AppView message does not set an Android `channel_id`, the manifest must bind `com.google.firebase.messaging.default_notification_channel_id` to that created channel; category preferences remain account-level Craftsky settings rather than Android channels.
  - AppView push sending already defaults to disabled through `PUSH_ENABLED=false`. With one shared Firebase project, non-production sending must remain disabled except during an explicitly enabled manual delivery check.
  - Web push, desktop platforms, and a client-owned persistent inbox are outside this slice.
- Test/build commands discovered:
  - Focused Flutter tests from `app/`: `flutter test test/notifications`.
  - Static analysis from `app/`: `dart analyze`.
  - Code generation from `app/`: `dart run build_runner build`.
  - Native/provider delivery remains a manual Android/iOS check after automated fake-adapter coverage.

## 3. Clarifying Questions And Decisions

### Q1: Should Android and iOS use the same application identifier?

Answer: Yes.

Decision / implication: Use `social.craftsky.app` for both Android `applicationId`/namespace and the iOS bundle identifier. The Firebase project is `craftsky-app`.

### Q2: When should notification permission be requested?

Answer: Immediately after the app reaches signed-in and onboarded state, including the first eligible launch. The product has not launched, so no legacy-user migration is needed.

Decision / implication: Permission is contextual to an authenticated Craftsky experience but is not tied to opening the Notifications page. A denial is not repeatedly prompted by the app.

### Q3: How should foreground notifications appear?

Answer: Show a Craftsky in-app banner and refresh notification data.

Decision / implication: Do not create a duplicate OS notification while foregrounded. The same domain event stream drives banner presentation and list/count invalidation.

### Q4: What should happen when a user taps a foreground banner or an OS notification?

Answer: Resolve the notification ID through AppView and open the precise authorized post/profile target, falling back to the Notifications surface.

Decision / implication: The client never trusts the push payload as a direct content destination and never reconstructs an AT-URI or DID from provider data.

### Q5: How should new notification activity appear in app navigation?

Answer: Show a numeric badge on the Notifications tab and navigation rail, visually capped at `99+`.

Decision / implication: The badge reads the account-wide `GET /v1/notifications/new-count` contract. Fetching does not clear it.

### Q6: When should the account-wide new marker be cleared?

Answer: Only after the Notifications page's first page has loaded successfully and rendered, including a successful empty result.

Decision / implication: The page calls `POST /v1/notifications/seen` after successful first-page rendering, not during route entry, loading/failure, prefetch, bootstrap, or background refresh. It then refreshes the count.

### Q7: Are category preferences part of this slice?

Answer: Yes. A settings button in the notification-list app bar opens controls for all seven categories.

Decision / implication: Each category exposes `Everyone` / `People I follow` scope and an independent push toggle.

### Q8: How are preference edits saved?

Answer: Immediately per control with optimistic UI.

Decision / implication: Patch only the changed category/field; roll back that control and show error feedback if AppView rejects or fails the update.

### Q9: What happens to category push toggles when OS permission is denied?

Answer: They remain editable.

Decision / implication: Account preference and device permission remain separate axes. Show a device-permission warning with an Open settings action rather than disabling server preferences.

### Q10: How should Firebase be isolated from application logic?

Answer: Use a provider-neutral stream-based notification service patterned after Stash Hub.

Decision / implication: Firebase message/token/settings types remain inside one adapter. Domain streams represent foreground receipt and user-open events; test fakes need no Firebase initialization.

### Q11: How should the current single-account app use `accountSubscriptionId`?

Answer: Persist the returned opaque ID securely for the authenticated account and validate incoming routing IDs before resolution, while keeping the local shape capable of adding more account mappings later.

Decision / implication: A stale or mismatched routing ID must not be resolved using the current account. This slice does not add multi-account sign-in UI or session storage.

### Q12: Should development and production use separate Firebase projects?

Answer: No. Use the single `craftsky-app` Firebase project.

Decision / implication: Both platform registrations and every build environment use that project. The client does not add Firebase project flavors or environment-specific Firebase apps.

### Q13: How is accidental non-production delivery controlled with one Firebase project?

Answer: Keep real FCM sending disabled by default in non-production AppView environments and enable it only temporarily for explicit physical-device delivery tests.

Decision / implication: Normal development cannot send real alerts. Production enablement remains a separate launch operation.

### Q14: Should Craftsky show a primer before the OS permission dialog?

Answer: No. Notification permission is expected for this social application.

Decision / implication: Once signed-in/onboarded readiness is reached and OS state is undetermined, request the OS permission directly. Do not add primer state or a custom pre-prompt.

### Q15: Should foreground banners be suppressed or deduplicated?

Answer: No. Show the foreground banner even while the Notifications page is visible, and do not add client-side session or persistent deduplication.

Decision / implication: Every foreground provider callback is handled normally. The client accepts the AppView/FCM at-least-once duplicate window rather than adding receipt state.

### Q16: What happens when a tap belongs to a stale or different account?

Answer: Do not resolve or navigate it. Once the app is ready, show a generic message that the notification is no longer available.

Decision / implication: Only a routing ID matching the current authenticated DID's secure binding can initiate resolution. A mismatch does not redirect the user to the current account's Notifications page.

### Q17: Does “badge” include the home-screen app icon, and should it poll?

Answer: No. Badge means only the numeric indicator on Craftsky's Notifications navigation item. Do not periodically poll.

Decision / implication: Refresh `newCount` when an authenticated app becomes ready, the app resumes, a foreground FCM message arrives, or the Notifications page refreshes/marks seen. The home-screen app icon is never updated by this slice.

### Q18: What must occur before notifications are marked seen?

Answer: The first notification page must load successfully and be rendered, including a successful empty result.

Decision / implication: Route visibility, a loading state, or a failed first-page request does not acknowledge newness and does not clear the badge.

### Q19: How should notification preferences be presented?

Answer: Open a dedicated full-screen Notification Settings route from the Notifications app bar. Do not add a master push switch.

Decision / implication: The page scrolls and exposes only independent per-category controls. Back returns to Notifications.

### Q20: How should local push routing be cleaned up during sign-out?

Answer: Clear the signed-out DID's local routing binding in every sign-out path. If AppView logout succeeds, retain the FCM token. If logout fails or a `401` forces local sign-out without confirmed server cleanup, make a best-effort FCM token deletion before clearing the session.

Decision / implication: Offline/local sign-out remains available, while an unconfirmed server cleanup cannot intentionally leave the old provider token active. A later eligible sign-in obtains and registers a new token.

### Q21: Should a notification tap survive a new login?

Answer: No. Hold it only through transient bootstrap/router readiness and onboarding completion for an already authenticated account. Discard it if actual sign-in is required.

Decision / implication: A user cannot log into a different account and inherit the prior notification destination. Normal list/count refresh exposes activity after sign-in.

### Q22: What happens when authorized tap resolution is offline or times out?

Answer: Navigate to Notifications, show normal loading/error UI plus brief unable-to-open feedback, and do not persist or automatically retry the deep link.

Decision / implication: Precise routing always depends on a successful AppView resolution. Recovery occurs through the normal list refresh and row navigation.

### Q23: Should notifications use sound or vibration?

Answer: Background notifications use the standard OS sound; Android uses the shared channel's standard vibration. Foreground in-app banners are silent and do not trigger vibration.

Decision / implication: iOS requests alert and sound authorization but not app-icon badge authorization. The AppView message requests the default APNs sound. No custom audio, local-notification presentation, or foreground haptics are added.

### Q24: How many Android notification channels should exist?

Answer: One channel named “Craftsky notifications.”

Decision / implication: It uses standard importance, sound, and vibration. The Android manifest binds FCM's `com.google.firebase.messaging.default_notification_channel_id` metadata to this channel so background notification messages without an explicit `channel_id` cannot fall back to an unintended provider channel. Per-category preference behavior is not duplicated through Android channel settings.

### Q25: Are push preferences account-wide or device-specific?

Answer: Account-wide. OS authorization remains device-specific.

Decision / implication: Settings explain that category preferences apply to all devices for the account, while denied-permission guidance says notifications are disabled on this device.

### Q26: How should future unknown categories behave?

Answer: An unknown feed category remains visible as a safe generic “New activity” row and uses AppView resolution on tap. A syntactically valid future push `type` is normalized to the same generic domain category and still resolves by validated notification ID. Unknown preference categories are not rendered by an older client.

Decision / implication: Known preference patches preserve unknown server values. Provider parsing still requires valid `notificationId` and `accountSubscriptionId`, but an unknown bounded `type` is not treated as a malformed destination or authorization signal. The client never invents category-specific copy or a setting label it cannot explain.

### Q27: How should unavailable actors or content render?

Answer: Keep a visible but non-navigable tombstone row. Use “Someone” for an unavailable actor and category-appropriate unavailable-content copy when the referenced content cannot be opened.

Decision / implication: A tap shows brief unavailable feedback. Raw identifiers and deleted/hidden content are never used as display fallbacks.

## 4. Candidate Approaches

### Option A: Provider-neutral stream service with a Firebase adapter

Summary: Define domain notification/open events and a narrow service interface. Implement Firebase listeners and token APIs in one adapter, expose them through Riverpod, and let coordinators handle registration, routing, refresh, banners, and navigation.

Pros:
- Matches the user's proven Stash Hub pattern.
- Keeps Firebase-specific objects out of UI, routing, repositories, and tests.
- Makes foreground, background-open, cold-start, and token-refresh behavior independently testable.
- Leaves a clean seam for future multi-account routing.

Cons:
- Introduces several small interfaces/providers and explicit lifecycle coordination.
- Requires care to initialize listeners exactly once and dispose them safely.

Risks:
- A coordinator mounted twice could duplicate banners, navigation, or registration unless keep-alive ownership is explicit.

### Option B: Register Firebase listeners directly in bootstrap and widgets

Summary: Initialize Firebase in `main.dart`, listen directly in the root widget/page, and navigate from `RemoteMessage` callbacks.

Pros:
- Fewer files and less initial abstraction.
- Closely resembles small Firebase examples.

Cons:
- Couples Firebase types to app lifecycle, routing, and UI.
- Harder to test cold-start and token refresh without plugin channels.
- Listener ownership becomes fragile across loading/ready/auth transitions.

Risks:
- Duplicate subscriptions or lost initial messages during router/auth initialization.

### Option C: Persist a separate local notification inbox

Summary: Store received push events locally and drive UI/navigation from that local inbox.

Pros:
- Can retain receipt state while offline.

Cons:
- Duplicates the authoritative durable AppView feed.
- Creates reconciliation, deletion, moderation, and account-isolation problems.
- Adds storage/migration work without a confirmed product need.

Risks:
- Stale local content could disagree with AppView authorization and lifecycle.

## 5. Recommended Direction

Recommended approach: Option A, a provider-neutral stream service with a Firebase adapter.

Why: It preserves Firebase as a replaceable delivery mechanism rather than an application-wide type system, matches the user's successful existing pattern, and creates deterministic seams for the highest-risk behaviors: permission timing, token ownership, cold-start routing, foreground presentation, and listener lifecycle. AppView remains the source of truth for notification content, availability, preferences, newness, and navigation authorization.

## 6. Problem / Opportunity

The AppView can now persist, deliver, resolve, count, and configure notifications, but the Flutter client cannot register a device or consume that system. Its existing notification UI also decodes an obsolete response shape. This slice connects the native applications to FCM while preserving AppView authority and gives users a coherent foreground experience, badge, current notification feed, and preference controls.

## 7. Goals

- G-001: Reliably register the current Android/iOS installation and authenticated account for FCM delivery.
- G-002: Present and route foreground, background-open, and terminated-open notifications safely.
- G-003: Render every durable notification category and availability state supported by AppView.
- G-004: Show and clear accurate account-wide notification newness without turning list fetches into writes.
- G-005: Let users configure category scope and push intent independently from OS permission.
- G-006: Keep Firebase-specific code isolated behind provider-neutral, testable streams.
- G-007: Protect FCM tokens, account-subscription IDs, session identity, and notification payloads from logs and cross-account routing.

## 8. Non-Goals

- NG-001: Web push, macOS, Windows, Linux, or raw APNs delivery.
- NG-002: Multi-account sign-in UI or replacing the current single-session authentication store.
- NG-003: A client-owned persistent notification inbox or offline copy of AppView notification content.
- NG-004: Per-notification read/unread state, individual mark-read actions, or per-device newness.
- NG-005: Server-side notification eligibility, outbox, retry, retention, or schema changes; only the default APNs sound value in the existing FCM message is included.
- NG-006: New AppView routes or changes to existing JSON wire contracts.
- NG-007: Notification grouping, digests, custom sounds, rich media, action buttons, or local scheduling.
- NG-008: Changing server-supplied generic English lock-screen copy.
- NG-009: Block/mute policy or client-side reconstruction of hidden/deleted content.
- NG-010: Production APNs credential management beyond documenting and executing the existing manual rollout gate.
- NG-011: Separate Firebase projects, environment-specific Firebase app registrations, or native dev/prod flavors.
- NG-012: Home-screen app-icon badges or periodic notification-count polling.
- NG-013: Client-side notification receipt deduplication or a persistent/deferred deep-link queue.
- NG-014: A permission primer, category-specific Android channels, a master push switch, custom notification sounds, or foreground sound/haptics.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in member | An onboarded Craftsky user on Android or iOS. | Timely, understandable alerts; safe navigation; visible new count; preference control. |
| Device installation | One native Craftsky installation identified by the existing stable device ID. | One current FCM token and an authenticated account-subscription binding. |
| Flutter application | The current single-account client consuming AppView. | Provider-neutral events, deterministic listener lifecycle, and safe routing validation. |
| Firebase Cloud Messaging | External native push transport. | Correct platform configuration, token lifecycle, and minimal payload handling. |
| AppView | Authoritative notification API and push sender. | Authenticated registration, resolution, preference, count, and acknowledgement calls. |

## 10. Current Behavior

Craftsky can display a paginated notification list only when the user opens the page and the legacy response decoder succeeds. It has no Firebase initialization, native provider configuration, device registration, permission prompt, listener, banner, badge, mark-seen call, resolution call, quote support, or preference UI. Notification taps from the OS cannot be handled because the app has no push integration.

## 11. Desired Behavior

On Android and iOS, Firebase initializes before any messaging-dependent object is created. Both platforms use the single `craftsky-app` Firebase project. Once auth and onboarding are ready, a single client coordinator checks permission, opens the OS prompt directly only when undetermined, obtains the current token when authorized, and registers it with AppView. It stores the returned opaque account-subscription ID securely for the signed-in DID and repeats registration when Firebase refreshes the token. Non-production AppView delivery remains disabled unless explicitly enabled for a manual device check.

One provider-neutral service publishes foreground receipt and user-open streams. Every foreground receipt shows a silent Craftsky banner, including while Notifications is visible, and invalidates notification list/count state without creating a duplicate OS alert. No receipt deduplication is added. Background and terminated messages are displayed using the standard OS sound and, on Android, one shared standard channel bound as FCM's manifest default. Tapping them produces the same domain open event; a syntactically valid unknown push `type` becomes generic activity rather than blocking authorized resolution. The coordinator validates the opaque routing ID against the current local binding, resolves the stable notification ID through AppView, and uses the returned authorized target to navigate to a post, profile, or Notifications fallback after transient auth/onboarding/router readiness. A routing mismatch does not navigate; a tap is discarded if a new sign-in is required; and an offline resolution falls back to Notifications without a deferred retry.

The Notifications destination shows an in-app numeric `newCount` badge capped visually at `99+`; the home-screen app icon is unchanged. The count refreshes on authenticated readiness, app resume, foreground FCM receipt, page refresh, and successful mark-seen, with no periodic polling. The page decodes the current durable response, including quote, unknown categories, and visible non-navigable tombstones for unavailable actors/content. It marks the current account snapshot seen only after the first page successfully renders. Its app bar opens a dedicated full-screen Notification Settings route containing independent controls for all seven categories and no master switch. Scope and push fields save immediately and optimistically; failures roll back the affected control, while unknown future preference entries are preserved but hidden. Copy explains that category preferences are account-wide and OS permission is device-specific. OS permission denial is shown separately with an Open settings action and does not disable server preference controls.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Signed-in Craftsky members shall be able to receive native push notifications for eligible AppView activity on Android and iOS. | Completes the user-facing half of the implemented delivery system. | Initial request | AC-001, AC-004 |
| BR-002 | Business | Must | Members shall receive a coherent foreground alert and safe destination when they open a notification from any supported app state. | Makes push useful without bypassing authorization. | User decisions | AC-006, AC-007 |
| BR-003 | Business | Must | Members shall see an accurate account-wide new-notification badge and clear it by actually viewing the notification page. | Makes notification activity discoverable without per-item read state. | User decisions | AC-011, AC-012 |
| BR-004 | Business | Must | Members shall be able to configure scope and push intent for all seven notification categories from the notification list. | Exposes the implemented preference model. | User decision | AC-013, AC-014 |
| FR-001 | Functional | Must | The Android and iOS applications shall use `social.craftsky.app`, connect to the single Firebase project `craftsky-app` in every environment, and initialize Firebase before messaging-dependent services are created. | Aligns native identity, records the one-project decision, and prevents initialization-order failures. | User decision / Firebase setup | AC-001 |
| FR-002 | Functional | Must | The client shall define a provider-neutral notification service exposing permission state/actions, token retrieval/refresh, foreground-receipt events, and user-open events; Firebase-specific types shall not escape its adapter. | Preserves testability and the chosen architecture. | Confirmed direction | AC-002, AC-015 |
| FR-003 | Functional | Must | After the app reaches signed-in and onboarded state, it shall check notification authorization and open the OS permission dialog directly, without a primer, only when the OS state is undetermined. | Implements the selected permission timing without repeated or custom prompts. | User decision | AC-003 |
| FR-004 | Functional | Must | When permission is authorized, the client shall obtain the current FCM token, call `POST /v1/notifications/devices` with the native platform and token, and securely persist the returned `accountSubscriptionId` for the authenticated DID. | Establishes the authorized device/account delivery route. | AppView contract / confirmed direction | AC-004 |
| FR-005 | Functional | Must | The client shall re-register a refreshed FCM token while signed in; if signed out or not yet ready, it shall defer registration until an authenticated onboarded state exists. | Keeps provider routing current without unauthenticated calls. | Initial request / discovery | AC-005 |
| FR-006 | Functional | Must | Permission, token retrieval, and registration failures shall not block onboarding completion or make the ready application unusable; retryable work shall be attempted again on a later eligible lifecycle trigger. | Push is valuable but not a prerequisite for using Craftsky. | Discovery risk | AC-003, AC-005 |
| FR-007 | Functional | Must | Every valid foreground provider callback shall emit a domain receipt event, show a silent Craftsky in-app banner using the provider-visible generic copy even when Notifications is visible, and invalidate notification list/count state without creating an OS notification. | Implements the confirmed foreground experience without suppression, sound, or deduplication state. | User decision | AC-006, AC-019 |
| FR-008 | Functional | Must | Background and terminated notification taps shall be normalized with foreground banner taps into one domain open event containing only validated `notificationId`, `type`, and `accountSubscriptionId` values. A syntactically valid but unknown bounded `type` shall normalize to generic activity and remain eligible for AppView resolution; missing or malformed required values shall reject the event. | Gives every app state one routing path while remaining forward-compatible without trusting provider data as a destination. | Initial request / Firebase contract / document review | AC-007 |
| FR-009 | Functional | Must | Before resolving a push open, the client shall require the payload `accountSubscriptionId` to match the securely stored binding for the current authenticated DID; mismatched or absent bindings shall not resolve or navigate and shall show only generic unavailable feedback once the app is ready. | Prevents stale or cross-account routing without redirecting the current account. | Confirmed direction / AppView privacy model | AC-008 |
| FR-010 | Functional | Must | For a valid open event, the client shall call `GET /v1/notifications/{notificationId}` and navigate from its authorized target: post, actor profile, or Notifications fallback; not-found and unavailable destinations shall fall back safely without reconstructing identifiers from the push. A network/timeout failure shall open Notifications with brief unable-to-open feedback and no persisted or automatic deep-link retry. | Preserves AppView authorization and avoids a deferred routing queue. | User decision / AppView contract | AC-007, AC-009, AC-025 |
| FR-011 | Functional | Must | The Flutter notification model shall decode stable IDs, all seven categories, actor/reference availability, durable type-specific metadata including quote, and an unknown-category fallback that does not crash the page. Unknown categories shall render as safe generic activity and still use AppView resolution on tap. | Aligns the client with the additive AppView contract. | Codebase finding / user decision | AC-010, AC-028 |
| FR-012 | Functional | Must | Notification rows shall render safe category-specific copy/content and navigate using available hydrated data. Unavailable actors/content shall remain visible as non-navigable tombstones using “Someone” or category-appropriate unavailable copy, with brief unavailable feedback and no raw identifier fallback. | Keeps the list understandable and moderation-safe. | AppView contract / user decision | AC-010, AC-029 |
| FR-013 | Functional | Must | The client shall show `GET /v1/notifications/new-count` as an in-app numeric badge on compact and large-form-factor Notifications navigation, displaying `99+` above 99 and never updating the home-screen app icon. It shall refresh on authenticated readiness, app resume, foreground FCM receipt, Notifications refresh, and mark-seen completion, with no periodic polling. | Implements the confirmed badge scope and refresh behavior. | User decision | AC-011, AC-020 |
| FR-014 | Functional | Must | The client shall call bodyless `POST /v1/notifications/seen` only after the first Notifications page has loaded successfully and rendered, including a successful empty page, then refresh the badge; route visibility, loading/failure, prefetch, and foreground receipt shall not mark notifications seen. | Prevents clearing newness before content is actually shown. | User decision / AppView rule | AC-012, AC-021 |
| FR-015 | Functional | Must | The Notifications page app bar shall open a dedicated full-screen Notification Settings route with independent controls for `like`, `follow`, `reply`, `mention`, `quote`, `repost`, and `everythingElse`, each with scope and push fields loaded from `GET /v1/notifications/preferences`; no master push switch shall be shown. | Exposes every implemented preference in a scalable surface. | User decision | AC-013, AC-022 |
| FR-016 | Functional | Must | Changing one known preference control shall update the UI optimistically and immediately PATCH only that category/field; success shall retain it, while failure shall roll back the affected control and show user-visible error feedback. Unknown server preference categories shall be ignored in UI and preserved by known-category patches. | Keeps settings responsive and forward-compatible without corrupting adjacent state. | User decision | AC-014, AC-023 |
| FR-017 | Functional | Must | Category push controls shall remain editable when OS permission is denied. | Keeps account intent independent from one device's permission state. | User decision | AC-013 |
| FR-018 | Functional | Must | When OS permission is denied, the settings surface shall show a device-level warning and an Open settings action; it shall not repeatedly invoke the system permission prompt. | Gives a recoverable consent path without nagging. | User decision | AC-003, AC-013 |
| FR-019 | Functional | Must | Messaging listeners and coordinators shall have one explicit keep-alive owner, process an initial terminated-open message once, subscribe once to live streams, and cancel subscriptions on disposal. | Prevents duplicate registration, banners, and navigation. | Discovery risk | AC-015 |
| FR-020 | Functional | Must | The Firebase background handler shall be a retained top-level entry point and shall perform no UI navigation, provider mutation, or sensitive payload logging. | Satisfies platform execution constraints and privacy boundaries. | Firebase guidance / security | AC-016 |
| FR-021 | Functional | Must | Every explicit or `401`-forced sign-out shall clear local account-subscription bindings while preserving the stable Craftsky device ID. A confirmed AppView logout shall retain the FCM token; a failed/unconfirmed logout or `401` path shall make a best-effort FCM token deletion before local session cleanup. | Preserves reliable local sign-out while reducing post-logout delivery risk. | AppView logout contract / user decision | AC-017, AC-024 |
| FR-022 | Functional | Must | A notification open may wait through transient bootstrap, router readiness, and onboarding completion for an already authenticated account, but shall be discarded if the resolved state requires an actual sign-in. | Prevents a notification destination from carrying into a different account. | User decision | AC-026 |
| FR-023 | Functional | Must | Background notifications shall request the standard OS sound. iOS shall request alert and sound authorization but not app-icon badge authorization. Android shall create one “Craftsky notifications” channel with standard importance, sound, and vibration, and shall bind `com.google.firebase.messaging.default_notification_channel_id` to that channel because the AppView message omits Android `channel_id`. Foreground banners shall remain silent and non-vibrating. | Provides expected social-app delivery without custom media, duplicate foreground alerts, or unintended Android fallback-channel behavior. | User decision / native platform constraints / document review | AC-027 |
| FR-024 | Functional | Must | Notification Settings shall explain that category scope/push preferences apply to every device for the account, while OS-denied guidance shall identify only the current device. | Prevents users confusing account intent with device authorization. | User decision / AppView data model | AC-013 |
| FR-025 | Functional | Must | The existing AppView FCM message shall request the default APNs sound without changing payload data, notification eligibility, delivery timing, routing, or any JSON API contract. | Completes standard iOS background sound, which the client cannot add after OS delivery. | User decision / codebase finding | AC-027 |
| FR-026 | Functional | Must | Real FCM sending shall remain disabled by default in non-production AppView environments and shall be enabled only explicitly and temporarily for manual physical-device delivery verification. | Limits accidental cross-environment alerts when all builds share one Firebase project. | User decision / existing AppView configuration | AC-030 |
| NFR-001 | Non-functional | Must | FCM tokens, account-subscription IDs, notification IDs, raw payloads, DIDs, handles, AT-URIs, and Firebase configuration credentials shall not appear in application logs, Sentry contexts, analytics, or user-facing diagnostic errors. | Protects routing and identity data. | Security model | AC-018 |
| NFR-002 | Non-functional | Must | Automated Flutter tests shall use fake provider-neutral services and mocked HTTP; they shall not initialize Firebase, contact FCM, or require real device permission. | Keeps tests deterministic and safe. | Confirmed architecture | AC-015, AC-018 |
| NFR-003 | Non-functional | Should | Notification initialization and stream handling should avoid delaying first usable UI beyond existing app dependency initialization, and transient push failures should be reported without a crash loop. | Push must not degrade core app startup. | Discovery | AC-003, AC-005 |
| NFR-004 | Non-functional | Should | New banner, badge, settings, warning, empty, loading, and error copy should use localization and accessible semantics/tap targets. | Preserves app-wide UX quality. | Existing conventions | AC-006, AC-011, AC-013 |
| RULE-001 | Business rule | Must | Notification permission is requested only after the account is both signed in and onboarded, and only while authorization is undetermined. | Captures the approved consent timing. | User decision | AC-003 |
| RULE-002 | Business rule | Must | Foreground delivery uses one in-app Craftsky banner plus data refresh, not an additional OS notification. | Prevents duplicate alerts. | User decision | AC-006 |
| RULE-003 | Business rule | Must | Push payload data identifies a notification and local account subscription but never authorizes or defines a content destination; AppView resolution is authoritative. | Prevents provider payloads bypassing auth/moderation. | User decision / AppView contract | AC-007, AC-008, AC-009 |
| RULE-004 | Business rule | Must | OS permission and AppView category preferences are independent: denied permission affects this device's delivery capability but does not rewrite or disable account preference controls. | Preserves the two-layer model. | User decision | AC-013 |
| RULE-005 | Business rule | Must | Notification GET requests and foreground refreshes are read-only; only successful page presentation triggers explicit mark-seen. | Prevents accidental badge clearing. | AppView contract / user decision | AC-012 |
| RULE-006 | Business rule | Must | This slice supports the current single authenticated account while storing routing bindings by DID so future multi-account support does not require changing the provider-neutral event contract. | Keeps scope honest and avoids a dead-end local shape. | Confirmed direction | AC-004, AC-008 |
| RULE-007 | Business rule | Must | The client accepts provider/AppView at-least-once delivery and shall not add session or persistent notification receipt deduplication. | Records the deliberate simplicity tradeoff. | User decision | AC-019 |
| RULE-008 | Business rule | Must | Category preferences are account-wide; OS permission and Android channel state are device-specific. | Keeps server preference and native authorization semantics distinct. | User decision / AppView contract | AC-013 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given Android and iOS builds in any environment, when each native app starts with the checked-in configuration, then both connect to the single `craftsky-app` Firebase project under the shared `social.craftsky.app` identifier and messaging services are created only afterward. |
| AC-002 | FR-002 | Given application code outside the Firebase adapter, when notification streams, permission, or tokens are consumed, then only provider-neutral domain types/interfaces are referenced. |
| AC-003 | FR-003, FR-006, FR-018, NFR-003, RULE-001 | Given permission states `notDetermined`, `authorized`, and `denied`, when a signed-in account becomes onboarded, then the OS prompt opens directly without a primer only for `notDetermined`; other states do not prompt, and failures do not block the ready app. |
| AC-004 | BR-001, FR-004, RULE-006 | Given authorized permission and an authenticated onboarded account, when the current token is obtained, then AppView receives the correct platform/token registration, the token is not echoed or logged, and the returned routing ID is stored securely under that DID. |
| AC-005 | FR-005, FR-006, NFR-003 | Given token refresh before or after auth readiness and transient registration failure, when an eligible lifecycle trigger occurs, then only authenticated registration is attempted, the latest token is used, and the app remains usable while deferred/retryable work is retained. |
| AC-006 | BR-002, FR-007, NFR-004, RULE-002 | Given one valid foreground callback while any route, including Notifications, is visible, when it arrives, then one localized silent Craftsky banner appears with accessible semantics, notification list/count providers refresh, and no OS alert, sound, or vibration is created. |
| AC-007 | BR-002, FR-008, FR-010, RULE-003 | Given foreground-banner, background-open, and terminated-open messages for the same valid binding, including a message with a syntactically valid future `type`, when the user taps each, then each produces the same domain open flow, the future type is represented as generic activity, the stable ID resolves through AppView, and navigation uses only the returned authorized target or Notifications fallback. |
| AC-008 | FR-009, RULE-003, RULE-006 | Given a payload with a missing, malformed, stale, or different account-subscription ID, when it is opened, then the client does not call owner-scoped resolution, does not navigate, and shows only generic unavailable feedback once ready. |
| AC-009 | FR-010, RULE-003 | Given AppView returns post, actor-profile, notifications, retracted, unavailable, or non-enumerating not-found outcomes, when an open is processed, then only the supplied authorized target is used and every other outcome lands safely on Notifications. |
| AC-010 | FR-011, FR-012 | Given durable list responses for follow, like, repost, reply, mention, quote, everythingElse, unavailable references, and an unknown future category, when decoded and rendered, then the page remains usable, never exposes raw identifiers/hidden content, and only supported available rows navigate. |
| AC-011 | BR-003, FR-013, NFR-004 | Given counts 0, 1, 99, and 100, when compact navigation and the large navigation rail render, then no badge is shown for 0 and the labels are `1`, `99`, and `99+` with accessible semantics. |
| AC-012 | BR-003, FR-014, RULE-005 | Given new notifications and background/list/count prefetches, when the first page is loading or fails, then no seen write occurs; when the first page successfully renders content or an empty result, then one mark-seen call occurs and the refreshed badge reflects AppView's snapshot result. |
| AC-013 | BR-004, FR-015, FR-017, FR-018, FR-024, NFR-004, RULE-004, RULE-008 | Given effective preferences with OS permission authorized or denied, when the full-screen settings route opens from the Notifications app bar, then all seven categories show independent scope/push controls and no master switch; copy identifies account-wide preferences, while denied permission adds a current-device Open settings warning without disabling controls. |
| AC-014 | FR-016 | Given a scope or push control change, when PATCH succeeds, then the optimistic value remains and omitted fields/categories are unchanged; when PATCH fails, then only the affected control rolls back and visible error feedback appears. |
| AC-015 | FR-002, FR-019, NFR-002 | Given repeated widget rebuilds, auth/onboarding changes, and one initial terminated message, when the notification system runs under fakes, then listeners initialize once, the initial message is consumed once, each stream emission is forwarded once, and disposal cancels subscriptions without Firebase initialization. |
| AC-016 | FR-020 | Given a background data callback, when the OS invokes it in a background isolate/process context, then the retained top-level handler completes without UI/state mutation and without logging payload or identifier values. |
| AC-017 | FR-021 | Given a registered account signs out successfully, when local cleanup completes, then that DID's routing binding is removed, another future DID mapping would be preserved, and the stable Craftsky device ID remains unchanged. |
| AC-018 | NFR-001, NFR-002 | Given sentinel tokens, routing IDs, notification IDs, identities, URIs, and payload text across success/failure paths, when logs, error reports, UI diagnostics, and automated tests are inspected, then no sensitive sentinel appears outside the fake provider/API boundary. |
| AC-019 | FR-007, RULE-007 | Given two provider callbacks carrying the same notification ID during one session, when both are received in the foreground, then each callback follows the normal banner/refresh path and no client receipt-deduplication store is consulted or persisted. |
| AC-020 | FR-013 | Given an authenticated app becomes ready, resumes, receives foreground FCM, refreshes Notifications, or completes mark-seen, when that trigger occurs, then `new-count` refreshes; elapsed foreground time alone causes no polling, and no home-screen icon badge is updated. |
| AC-021 | FR-014 | Given route entry followed by loading, failure, content success, and empty success, when each state occurs, then seen is called only after each successful first-page render and the badge remains unchanged by loading/failure alone. |
| AC-022 | FR-015 | Given the Notifications app-bar settings action, when it is tapped, then a dedicated scrollable full-screen typed route opens, Back returns to Notifications, all seven categories are present, and no master switch exists. |
| AC-023 | FR-016 | Given a preferences response includes a future unknown category, when the page renders and a known field is patched, then the unknown category has no UI control, the known optimistic result is handled normally, and the unknown server value is not overwritten. |
| AC-024 | FR-021 | Given successful logout, failed logout, and `401`-forced sign-out, when local cleanup runs, then every path clears routing bindings and preserves the stable Craftsky device ID; success retains the FCM token, while failed/unconfirmed paths attempt token deletion before clearing the session without blocking local sign-out if deletion fails. |
| AC-025 | FR-010 | Given a valid matching notification open whose resolution fails offline or times out, when the failure is handled, then Notifications opens with brief unable-to-open feedback and no deep-link retry is persisted or automatically scheduled. |
| AC-026 | FR-022 | Given a notification open during transient restoration, onboarding, and actual signed-out states, when readiness resolves, then transient states retain one pending open for the existing account, while a required new sign-in discards it before any destination navigation. |
| AC-027 | FR-023, FR-025 | Given iOS background, Android background, and foreground delivery, when notification behavior and native configuration are inspected, then iOS requests alert/sound but not badge authorization and receives default APNs sound; Android creates one “Craftsky notifications” channel with standard sound/vibration and its manifest `com.google.firebase.messaging.default_notification_channel_id` points to that channel; and foreground banners produce neither sound nor vibration. |
| AC-028 | FR-011 | Given an unknown future feed category with safe hydrated actor data, when rendered and tapped, then it appears as generic localized activity and routes only through AppView resolution without inferred category copy or raw identifiers. |
| AC-029 | FR-012 | Given unavailable actor, unavailable content, and fully available rows, when rendered and tapped, then unavailable rows remain visible with “Someone” or category-appropriate tombstone copy and brief feedback but no navigation, while the fully available row behaves normally. |
| AC-030 | FR-026 | Given normal non-production startup and an explicit manual delivery session, when AppView configuration is loaded, then real FCM sending is disabled in the normal case and runs only when the operator has temporarily enabled it with valid project credentials. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Permission is denied during the post-onboarding request. | The app remains ready, does not prompt again automatically, shows device guidance in notification settings, and keeps account preferences editable. | FR-003, FR-006, FR-017, FR-018 |
| EC-002 | FCM returns no token immediately after permission grant. | Registration is skipped safely and retried on a later eligible trigger/token refresh; no empty token is sent. | FR-004, FR-005, FR-006 |
| EC-003 | Token refresh fires while signed out. | The latest token is deferred and registered only after a signed-in onboarded account exists. | FR-005 |
| EC-004 | Registration succeeds repeatedly with the same token/account. | Local routing storage converges on the returned ID; AppView idempotency prevents duplicate subscriptions. | FR-004 |
| EC-005 | A foreground message arrives while the Notifications page is visible. | One silent banner appears and list/count refresh; page visibility does not suppress the banner and receipt does not control mark-seen. | FR-007, FR-014, RULE-005 |
| EC-006 | A cold-start notification arrives before auth, onboarding, or router readiness. | The open event is held through transient readiness for the existing account; it is discarded if readiness resolves to signed out and a new login is required. | FR-008, FR-019, FR-022 |
| EC-007 | The stored routing ID belongs to a prior signed-out account. | The current account does not resolve or navigate it; generic unavailable feedback appears and stale binding cleanup is attempted through normal lifecycle. | FR-009, FR-021 |
| EC-008 | AppView resolves a delivered notification to not-found after permanent actor deletion. | The client opens the Notifications surface without revealing whether the old ID existed. | FR-010 |
| EC-009 | The notification category or push `type` is newer than this client. | The list and provider event use safe generic activity; taps still rely on server resolution after required identifier/binding validation. Unknown preference categories are hidden and preserved. | FR-008, FR-011, FR-016 |
| EC-010 | A reference or actor is marked unavailable. | A visible non-navigable tombstone uses “Someone” or unavailable-content copy; no hidden URI, identity, or content is shown or used for navigation. | FR-011, FR-012 |
| EC-011 | New activity commits while mark-seen is in flight. | The page refreshes the server count after acknowledgement; the newer revision remains badged according to AppView snapshot semantics. | FR-013, FR-014 |
| EC-012 | Two preference edits overlap and one fails. | Each field/category mutation tracks its own optimistic baseline; the failed mutation cannot roll back a later successful value. | FR-016 |
| EC-013 | The OS settings screen changes permission while Craftsky is backgrounded. | Permission state refreshes on resume and registration occurs if authorization becomes available. | FR-003, FR-004, FR-018 |
| EC-014 | Firebase initialization or stream emits an error. | The error is classified without sensitive values, push features degrade safely, and the core app does not enter a repeated crash/listener loop. | FR-006, FR-019, NFR-001, NFR-003 |
| EC-015 | The same notification ID is delivered twice while foregrounded. | Both callbacks follow the normal silent banner/refresh behavior; no client receipt deduplication is introduced. | FR-007, RULE-007 |
| EC-016 | First-page notification loading fails. | The error state remains visible and the new-count badge is not acknowledged or cleared. | FR-014, RULE-005 |
| EC-017 | Explicit logout fails or a `401` forces sign-out. | Local sign-out continues, routing bindings are cleared, and FCM token deletion is attempted without preventing session cleanup if provider deletion also fails. | FR-021 |
| EC-018 | Resolution fails because the device is offline. | Notifications opens with unable-to-open feedback; no target is inferred and no automatic deep-link retry is queued. | FR-010 |
| EC-019 | An unknown preference category is returned. | It is omitted from the page and remains unchanged when a known category/field is patched. | FR-016 |
| EC-020 | The app remains foregrounded for a long session with no lifecycle or FCM event. | No periodic new-count request occurs; the badge refreshes on the next agreed lifecycle/page trigger. | FR-013 |
| EC-021 | Non-production AppView starts normally. | Real FCM sending remains disabled unless an operator explicitly enables a bounded manual delivery session. | FR-026 |

## 15. Data / Persistence Impact

- New local data:
  - Secure mapping from authenticated DID to opaque `accountSubscriptionId` for the current installation.
  - No raw FCM token is persisted by Craftsky application code beyond Firebase SDK/platform management and the immediate authenticated registration request.
- Changed local data:
  - Existing secure Craftsky session and stable device ID formats remain unchanged.
- Server data:
  - No new tables, fields, migrations, retention behavior, or JSON contract; the existing AppView FCM payload adds only the default APNs sound instruction.
- Migration required: No database or local inbox migration. The app has not launched, so native application identifier alignment has no production installation migration.
- Backwards compatibility:
  - The decoder must tolerate future unknown categories and current unavailable references.
  - The routing store is keyed by DID even though this client slice exposes one active account.

## 16. UI / API / CLI Impact

- UI:
  - Silent foreground Craftsky notification banner on every valid foreground callback, including while Notifications is visible.
  - In-app numeric `newCount` badge on bottom navigation and navigation rail, capped at `99+`; no home-screen app-icon badge or periodic polling.
  - Durable notification rows for all current categories, including quote, generic unknown-category fallback, and visible non-navigable unavailable tombstones.
  - Settings button in the Notifications app bar.
  - Dedicated full-screen Notification Settings route with seven independent account-wide categories, no master switch, immediate optimistic controls, loading/error states, and current-device denied-permission guidance/Open settings action.
- API consumption:
  - `GET /v1/notifications`
  - `GET /v1/notifications/new-count`
  - `POST /v1/notifications/seen`
  - `GET /v1/notifications/{notificationId}`
  - `GET /v1/notifications/preferences`
  - `PATCH /v1/notifications/preferences`
  - `POST /v1/notifications/devices`
  - Existing logout endpoints continue to perform AppView subscription cleanup.
- CLI: No user-facing CLI changes.
- Background execution:
  - One top-level Firebase background handler for platform-required message processing.
  - OS displays combined messages with standard sound while backgrounded/terminated; app routing begins only after user interaction and app readiness.
  - AppView real FCM sending remains disabled by default outside production and explicit manual test sessions.
- Native configuration:
  - Align Android and iOS identifiers to `social.craftsky.app`.
  - Add Firebase Android/iOS configuration for the single `craftsky-app` project.
  - Enable iOS Push Notifications and Remote notifications capabilities; request alert/sound but not app-icon badge authorization.
  - Create one Android “Craftsky notifications” channel with standard importance, sound, and vibration, then bind `com.google.firebase.messaging.default_notification_channel_id` in the Android manifest to that channel ID.

## 17. Security / Privacy / Permissions

- Authentication:
  - Registration, resolution, list, count, seen, and preference calls use the existing Craftsky session and device-ID middleware.
  - No messaging registration request is made while signed out.
- Authorization:
  - The current authenticated DID selects the local routing binding; payload values never select a bearer session.
  - A routing mismatch prevents owner-scoped resolution and navigation under the current account.
  - Pending opens are discarded when a new sign-in is required.
  - AppView resolution is the only authority for post/profile destinations.
- Sensitive data:
  - Store opaque account-subscription IDs in secure storage, keyed by DID.
  - Never log FCM tokens, routing IDs, raw payloads, notification IDs, session tokens, DIDs, handles, or AT-URIs.
  - Firebase client configuration identifiers may be checked in as platform configuration; server credentials/APNs keys are not client source.
- Permissions:
  - Request OS notification permission directly, without a primer, only after signed-in/onboarded readiness and only from undetermined state.
  - Request iOS alert/sound authorization but not app-icon badge authorization.
  - Denial is respected; recovery uses an explicit Open settings action.
  - Account-wide push preferences remain independent from current-device authorization.
- Abuse/failure cases:
  - Payload routing ID and notification ID are treated as untrusted input and validated before use.
  - Unknown categories, unavailable content, malformed data, stale IDs, and non-enumerating 404s fall back without crashing or leaking existence.
  - Failed/unconfirmed logout and `401`-forced sign-out attempt FCM-token deletion and always clear local routing bindings without blocking local sign-out.

## 18. Observability

- Events/logs:
  - Safe lifecycle events for Firebase initialization failure, permission outcome class, token availability class, registration success/failure class, token-deletion outcome class, foreground receipt, open-source class, routing mismatch, resolution outcome class, and listener start/stop.
  - Never attach raw message data or identifier values.
- Metrics:
  - No new product analytics are required in this slice.
- Error reporting:
  - Use existing error taxonomy/reporter with bounded classifications and redacted diagnostics.
  - User-denied permission is an expected state, not a reportable exception.
- Alerts: None for the Flutter client in this slice; AppView delivery/queue alerts remain authoritative.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Firebase types leak into presentation/routing code. | Tight coupling and plugin-dependent tests. | Provider-neutral interface, domain events, import-boundary regression tests. |
| RISK-002 | Permission is requested too early or repeatedly. | Poor consent experience and reduced opt-in. | Auth/onboarding readiness gate plus undetermined-only rule and explicit settings recovery. |
| RISK-003 | Listeners initialize more than once. | Duplicate banners, registration calls, or navigation. | Single keep-alive owner, initial-message consumption guard, disposal tests. |
| RISK-004 | A stale routing ID opens under the wrong account. | Privacy/authorization incident. | Secure DID-keyed binding validation before resolution; safe fallback on mismatch. |
| RISK-005 | The client trusts payload destination data. | Moderation/deletion bypass or identifier leakage. | Payload allowlist and mandatory AppView resolution. |
| RISK-006 | Token/identifier values leak through logs or Sentry. | Routing and identity exposure. | Redacted interfaces, sentinel tests, no raw payload logging. |
| RISK-007 | Mark-seen fires during route entry, loading failure, background refresh, or prefetch. | Badge clears before the user sees a successfully loaded notification page. | Successful-first-page-render gate and read-only provider tests. |
| RISK-008 | Optimistic preference requests race. | UI rolls back a newer choice or diverges from server state. | Per-field/category mutation sequencing and targeted rollback tests. |
| RISK-009 | Unknown/quote/unavailable responses crash the legacy decoder. | Notifications page becomes unusable during rollout. | Durable model rewrite, forward-compatible fallback, complete response matrix tests. |
| RISK-010 | Push initialization failure blocks the whole app. | Users cannot use Craftsky during provider/config outage. | Non-blocking coordinator, safe degradation, bounded retries, initialization/error tests. |
| RISK-011 | iOS native capability or APNs configuration is incomplete. | Android works while iOS silently receives nothing. | Native project verification plus explicit physical-device MAN checks before enablement. |
| RISK-012 | Changing native identifiers after distribution would create new app identities. | Upgrade/signing/store continuity failure. | Complete alignment before first launch; treat `social.craftsky.app` as stable afterward. |
| RISK-013 | One Firebase project is shared across environments. | A mistakenly enabled non-production AppView could send real alerts to shared-project tokens. | Default non-production `PUSH_ENABLED=false`; allow only explicit temporary manual-test enablement. |
| RISK-014 | Server logout and best-effort token deletion both fail. | A signed-out device may briefly receive already queued or stale-subscription pushes. | Clear local bindings/session, retry only through normal future lifecycle, and let AppView deactivate the token on provider `unregistered`; document that best effort cannot recall accepted pushes. |
| RISK-015 | AppView and native sound/channel configuration drift. | iOS may deliver silently or Android may use an unintended fallback channel. | Assert the default APNs sound payload, create one named Android channel, statically assert the manifest default-channel metadata points to it, and include physical-device sound/channel checks. |
| RISK-016 | Lifecycle-only badge refresh becomes stale during a long uninterrupted foreground session with push disabled. | New in-app activity may not appear in the badge immediately. | Accept eventual refresh on the next agreed lifecycle/page trigger; do not add periodic polling without a new product decision. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Craftsky has not shipped, so changing both native identifiers to `social.craftsky.app` has no installed-user or store migration impact. | A released identifier would require a different migration/release strategy. |
| ASM-002 | The current Flutter app continues to expose one active account during this slice. | Multi-account UI would require session selection and multiple concurrent routing bindings/coordinators. |
| ASM-003 | AppView's implemented notification/device/preference/newness/resolution routes are the authoritative wire contract. | Any server drift must be reconciled before client implementation. |
| ASM-004 | Combined FCM notification-and-data messages are used, so the OS displays background alerts and provides data on user open. | Data-only delivery would require local-notification presentation and different background rules. |
| ASM-005 | Server generic title/body copy is acceptable for foreground banners in this first pass. | Client-localized action copy would require an approved payload/template policy change. |
| ASM-006 | Firebase SDK-managed token storage is sufficient; Craftsky only needs to secure the opaque account-subscription binding. | A new threat model could require additional client token-at-rest controls. |
| ASM-007 | Android and iOS are the only supported push platforms for launch. | Web or desktop would require additional provider configuration, permission, and background handling. |
| ASM-008 | AppView may be unreachable when a notification is opened; offline opens fall back to Notifications and recover through normal list refresh rather than retrying the deep link. | Explicit deferred-deep-link persistence would require new requirements. |
| ASM-009 | A single Firebase project is acceptable for every build environment when non-production senders remain disabled by default. | Stronger isolation would require another Firebase project or native build variants, both explicitly excluded. |
| ASM-010 | Standard OS sound/vibration behavior is sufficient; Craftsky does not need custom sounds or foreground audio/haptics. | Custom media or per-category channel behavior would expand native and server scope. |

## 21. Open Questions

- [ ] Non-blocking rollout requirement: upload/configure the Apple Push Notification authentication key for Firebase project `craftsky-app` before MAN-iOS delivery verification.
- [ ] Non-blocking implementation choice: select the exact in-app banner component using existing Craftsky messenger/theming primitives during coding design; behavior and copy source are already fixed.
- [ ] Deferred: define multi-account session selection and simultaneous routing when the Flutter app gains multi-account support.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer:
Date: 2026-07-15
Notes: This client slice touches OS permissions, secure routing state, authentication-gated registration, background execution, native identifiers/capabilities, standard notification sound, a narrow AppView payload follow-up, and privacy-sensitive push payload handling. The user completed a detailed grilling pass and approved every decision captured above. Requirements review remains required before implementation.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001 through BR-004; FR-001 through FR-026; NFR-001 and NFR-002; RULE-001 through RULE-008. NFR-003 and NFR-004 are Should-level and should receive automated coverage where practical.
- Suggested test levels:
  - Unit: payload allowlist/validation including generic normalization of a syntactically valid unknown push `type`, permission state policy, routing-binding match/no-navigation, durable response decoding, unknown-category and tombstone fallback, badge formatting/triggers, resolution-target/offline mapping, preference optimistic sequencing/rollback/unknown preservation.
  - Provider/service: fake stream initialization, initial message consumed once, callback pass-through without receipt dedupe, transient-ready versus sign-in-required open handling, token refresh/defer/retry/deletion, listener disposal, background-handler constraints, no Firebase initialization in tests.
  - HTTP integration with mocked Dio: device registration, list, lifecycle-driven new-count, successful-render seen gate, resolution and offline fallback, preference GET/PATCH, successful/failed/401 logout-local-binding cleanup.
  - Riverpod/widget: direct auth/onboarding permission request, always-visible silent foreground banner and provider invalidation, in-app-only badge on bottom bar/rail, successful first-page acknowledgement, all category/unknown/tombstone rows, full-screen settings without master control, account/device copy, permission warning/Open settings, error/loading/empty states.
  - Static/privacy regression: Firebase import boundary, sensitive sentinel scan, listener ownership, Android manifest default-channel linkage, absence of polling/deep-link persistence/receipt dedupe, existing auth/router/notification regressions.
  - AppView focused: default APNs sound construction and non-production push-disabled configuration regression.
  - Native/manual: Android single-channel foreground/background/terminated receipt/open/sound/vibration; iOS foreground/background/terminated receipt/open/default sound without app-icon badge; token rotation/deletion; denied/granted permission; physical-device settings recovery; explicitly enabled non-production sender reset to disabled afterward.
- Blocking open questions: None for test design. High-risk document review and explicit implementation approval are required before source changes.
