# Coding Plan: Flutter Push Notifications

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes, High risk, 2026-07-15
- Repository guidance: `AGENTS.md`
- Architecture references:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Existing implementation inspected:
  - Flutter bootstrap, dependency readiness, auth/onboarding, secure device identity, Dio interceptors, typed routes, app shell, global messenger, localization, and test harnesses
  - Existing notification API client, repository seam, durable-feed decoder, pagination provider, page, rows, and tests
  - Implemented AppView notification list, count, seen, resolution, preferences, device registration, Firebase sender, and push configuration
  - Android/iOS identifiers, manifests/project files, and current native entry points
- Document-review notes carried into this plan:
  - DR-001: the Android manifest default FCM channel metadata must reference the one Craftsky channel ID, and REG-006 must assert the linkage.
  - DR-002: a syntactically valid bounded future push `type` normalizes to generic activity; valid notification and account-subscription IDs remain mandatory.
- Approval gate: planning may proceed, but the High-risk source implementation still requires explicit user approval after this artifact.

## 2. Implementation Strategy

Implement the client as six cooperating boundaries while keeping AppView authoritative:

1. A provider-neutral `NotificationService` owns permission, token, foreground-receipt, opened-message, initial-message, token-deletion, and native-settings operations. Only its Firebase adapter and bootstrap/background files import Firebase packages. Tests override this service and never initialize Firebase.
2. One keep-alive `NotificationCoordinator` owns service initialization and all provider stream subscriptions. It watches provider-neutral auth/onboarding readiness, retains at most one in-memory open through transient readiness, registers the latest token, validates the DID-keyed local routing binding, resolves notification IDs through AppView, invalidates feed/count state, and emits provider-neutral presentation/navigation effects.
3. A single root `NotificationEffectHost` is the UI bridge. It reports router/presentation readiness, forwards resume events to the coordinator, presents foreground delivery through the existing `AppMessenger`/Craftsky snackbar seam, and maps authorized navigation intents to existing typed post/profile/Notifications routes. It does not subscribe to Firebase streams.
4. Existing notification data code is expanded through narrow repository interfaces for device registration, resolution, new count, seen acknowledgement, and preferences. The durable feed decoder is rewritten for stable IDs, seven known categories, unknown categories, explicit actor/reference availability, quote metadata, and safe tombstones.
5. Riverpod owns account-visible state: new count, permission status, preferences, feed pagination, and seen acknowledgement. The Notifications page schedules acknowledgement only after a successful first-page result has actually rendered; preferences use per-category/per-field optimistic mutation generations so one failed request cannot roll back a newer edit.
6. Native setup aligns both apps to `social.craftsky.app`, configures the single `craftsky-app` Firebase project, creates one Android channel and binds it as FCM's manifest default, and enables iOS push/background capabilities. The only AppView source changes add default APNs sound and strengthen the existing explicit push-enable configuration gate.

The first red test is UT-002. It establishes the privacy and authorization boundary without Firebase: only `notificationId`, bounded/normalized `type`, and `accountSubscriptionId` enter a domain open event; extra destination-shaped provider data is ignored. Routing storage/policy comes next, before permission, token registration, listeners, native configuration, or UI.

No PDS token, PDS write, lexicon, migration, new AppView route, new JSON shape, local notification inbox, app-icon badge, polling timer, receipt deduplication store, or deferred deep-link queue is introduced.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Native identity and Firebase bootstrap | Legacy Android/iOS identifiers; no Firebase packages/configuration | Align both apps to `social.craftsky.app`; configure `craftsky-app`; initialize Firebase before messaging access and degrade safely on failure | BR-001, FR-001, FR-006, NFR-003 | AT-001, AT-002, REG-006, MAN-001, MAN-002 |
| Provider boundary | No messaging service | Add a provider-neutral service and Firebase adapter; map `RemoteMessage`/Firebase enums at the boundary only | FR-002, FR-008, FR-020, NFR-001, NFR-002 | UT-002, AT-009, IT-010, IT-013, REG-001, REG-002, REG-005 |
| Listener and lifecycle ownership | Ready app has no notification owner or lifecycle observer | Add one keep-alive coordinator plus one root effect/lifecycle host; initialize/subscribe/consume initial open exactly once | FR-019, FR-022, NFR-003 | UT-012, UT-014, AT-009, AT-011, IT-010, REG-009 |
| Permission and registration | Auth/onboarding providers exist; no notification permission/token registration | Gate direct prompt on signed-in + onboarded + undetermined; register latest authorized token and retry on eligible triggers | FR-003–FR-006, FR-018, RULE-001, RULE-006 | UT-001, UT-013, AT-002, IT-001, IT-002, MAN-003, MAN-004 |
| Secure routing | Stable secure device ID exists; no account-subscription binding | Store opaque routing IDs in one secure DID-keyed map; validate exact current-account match before resolution | FR-004, FR-009, RULE-003, RULE-006 | UT-003, UT-015, AT-005, IT-012 |
| Open resolution and navigation | Rows navigate from hydrated data; provider opens do not exist | Normalize all push/banner opens, resolve through AppView, map only authorized targets, and use Notifications fallback on safe/offline outcomes | BR-002, FR-008–FR-010, FR-022, RULE-003 | UT-004, AT-004, AT-005, AT-011, IT-004, IT-012 |
| Foreground presentation | Global `AppMessenger` presents themed Craftsky snackbars | Show every valid receipt through `AppMessenger.info` with a localized open action; always invalidate feed/count; never produce OS presentation, sound, vibration, or dedupe | FR-007, FR-023, NFR-004, RULE-002, RULE-007 | UT-016, UT-018, AT-003, IT-007, MAN-001, MAN-002 |
| Durable feed | Five legacy types; required actor/content; URI-based dedupe | Decode stable event ID, seven types, unknown fallback, reference availability, quote, and tombstones; dedupe by stable ID | FR-011, FR-012 | UT-005, AT-006, IT-003, IT-007, REG-003 |
| New count and acknowledgement | No badge/count/seen state | Add read-only count provider and compact/rail badges; acknowledge only a successfully rendered first page and refresh count afterward | BR-003, FR-013, FR-014, RULE-005 | UT-006–UT-008, AT-007, IT-005, IT-008, REG-004, REG-008 |
| Preferences | No client models, API calls, provider, settings route, or UI | Add seven-category full-screen settings, independent scope/push controls, optimistic per-field PATCH, unknown preservation, and denied-device warning | BR-004, FR-015–FR-018, FR-024, RULE-004, RULE-008 | UT-009–UT-011, AT-008, IT-006, IT-009, MAN-004 |
| Sign-out cleanup | Explicit and 401 sign-out clear only session state | Add one idempotent notification cleanup service used before session clearing; preserve device ID; delete FCM token only for unconfirmed cleanup | FR-021 | UT-019, AT-010, IT-011, MAN-003 |
| Privacy and diagnostics | Error mapping already emits bounded endpoint categories and Sentry allowlists | Add safe notification endpoint categories and classification-only lifecycle logs; prevent model/effect stringification from exposing payload or IDs | NFR-001, NFR-002 | UT-002, IT-013, REG-001, REG-002 |
| AppView follow-ups | APNs config has expiration only; push defaults false but project validation is prod-only | Add APNs default sound; require project ID whenever push is explicitly enabled; make dev default-off explicit | FR-025, FR-026 | AT-012, IT-014, IT-015, REG-007, REG-010, MAN-002, MAN-005 |

## 4. Files And Modules

Paths below are implementation targets. Generated `.g.dart` and localization files are regenerated beside their source and are not hand-edited.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/pubspec.yaml`, `app/pubspec.lock` | Change | Add `firebase_core`, `firebase_messaging`, and a small native app-settings launcher dependency; do not add a local-notification package | FR-001, FR-002, FR-018 | AT-001, AT-008, REG-001, MAN-004 |
| `app/lib/firebase_options.dart` | Generate | Checked-in non-secret FlutterFire options for Android/iOS in `craftsky-app` | FR-001 | AT-001, REG-006 |
| `app/lib/notifications/models/notification_open_event.dart` | Create | Redacted value types and allowlisted provider-data parser; bounded unknown `type` becomes generic | FR-008, FR-009, NFR-001, RULE-003 | UT-002, UT-003, AT-004, AT-005 |
| `app/lib/notifications/models/notification_delivery_event.dart` | Create | Provider-neutral foreground receipt plus initial/background/foreground open source enum and bounded visible copy | FR-002, FR-007, FR-008 | AT-003, AT-004, UT-018 |
| `app/lib/notifications/models/notification_permission.dart` | Create | `notDetermined` / `authorized` / `denied` domain status and pure permission action policy | FR-003, FR-006, FR-018, FR-023 | UT-001, UT-016, AT-002 |
| `app/lib/notifications/models/notification_resolution.dart` | Create | Decode AppView resolution and represent post/profile/Notifications navigation intents without provider destinations | FR-010, RULE-003 | UT-004, IT-004 |
| `app/lib/notifications/models/notification_preferences.dart` | Create | Known categories/scopes, effective values, hidden raw unknown entries, one-field patches, and optimistic field keys | FR-015–FR-017, FR-024 | UT-009–UT-011, IT-006 |
| `app/lib/notifications/models/craftsky_notification.dart` | Change | Replace the legacy decoder with stable IDs, seven known categories plus unknown, explicit actor/reference availability, quote/everythingElse metadata, and tombstone-safe accessors | FR-011, FR-012 | UT-005, AT-006, IT-003, IT-007 |
| `app/lib/notifications/models/notification_page.dart`, `notifications_state.dart` | Change | Preserve cursor pagination, dedupe by stable notification ID, and carry one first-page render token across load-more states | FR-011, FR-014 | UT-005, UT-008, IT-003, IT-005, REG-003 |
| `app/lib/notifications/services/notification_service.dart` | Create | Narrow provider-neutral service for init/dispose, permission, token/delete, token refresh, receipts, opens, initial open, and native settings | FR-002, FR-019, FR-021 | AT-009, IT-010, REG-001 |
| `app/lib/notifications/services/firebase_notification_bootstrap.dart` | Create | Initialize Firebase safely, register background handler before `runApp`, and return an available/unavailable `NotificationService` without leaking Firebase types | FR-001, FR-006, FR-020, NFR-003 | AT-001, AT-002, IT-013, REG-005 |
| `app/lib/notifications/services/firebase_notification_service.dart` | Create | Sole Firebase messaging adapter; map settings/messages/tokens to domain types, disable foreground OS presentation, and open native settings through the selected launcher | FR-002–FR-008, FR-018, FR-019, FR-021, FR-023 | UT-001, UT-002, UT-012, UT-013, UT-016, AT-002–AT-004, AT-009 |
| `app/lib/notifications/services/firebase_notification_background_handler.dart` | Create | Top-level `@pragma('vm:entry-point')` handler; initialize only what the background isolate requires and perform no UI/state/logging work | FR-020 | AT-009, IT-013, REG-005 |
| `app/lib/notifications/services/notification_routing_storage.dart` | Create | Secure JSON map keyed by DID, with read/replace/remove-one operations and corruption-safe cleanup | FR-004, FR-009, FR-021, RULE-006 | UT-003, UT-015, UT-019, IT-011, IT-012 |
| `app/lib/notifications/services/notification_resolution_policy.dart` | Create | Pure mapping of server target/404/unavailable/offline/timeout into navigation plus optional feedback | FR-010, FR-022, RULE-003 | UT-004, UT-014, IT-004 |
| `app/lib/notifications/services/notification_seen_policy.dart` | Create | Permit seen only for a successful post-frame first-page render token | FR-014, RULE-005 | UT-008, AT-007, IT-005, REG-008 |
| `app/lib/notifications/services/notification_sign_out_cleanup.dart` | Create | Coalesce cleanup per DID, best-effort delete token for unconfirmed paths, remove only that binding, and never block local sign-out | FR-021 | UT-019, AT-010, IT-011 |
| `app/lib/notifications/data/notification_api_client.dart` | Change | Add exact existing camelCase routes for register, resolve, count, seen, preferences GET/PATCH; redact endpoint classification | FR-004, FR-010, FR-013–FR-016 | IT-001, IT-003–IT-006, IT-012 |
| `app/lib/notifications/data/notification_repository.dart` | Change | Keep list as its narrow interface; add separate narrow device/count/seen/resolution/preferences interfaces | FR-004, FR-010, FR-013–FR-016 | IT-001, IT-003–IT-006 |
| `app/lib/notifications/data/api_notification_repository.dart` | Change | One concrete AppView adapter may implement the narrow interfaces while tests override each seam independently | Same as repository interfaces | Same as repository interfaces |
| `app/lib/notifications/providers/notification_repository_provider.dart` | Change | Expose the concrete adapter through narrow interface providers | FR-004, FR-010, FR-013–FR-016 | IT-001, IT-003–IT-006 |
| `app/lib/notifications/providers/notification_service_provider.dart` | Create | Always-alive service injection seam; default unavailable fake-safe service, overridden by production bootstrap | FR-002, FR-006, NFR-002 | AT-002, AT-009, IT-010, REG-001 |
| `app/lib/notifications/providers/notification_permission_provider.dart` | Create | Shared permission state/check/request/open-settings controller; checking never prompts | FR-003, FR-006, FR-018, FR-023 | UT-001, UT-016, AT-002, AT-008, IT-002, IT-009 |
| `app/lib/notifications/providers/notification_new_count_provider.dart` | Create | Async count state plus allowlisted refresh triggers; no timer or app-icon API | FR-013 | UT-006, UT-007, IT-008, REG-004 |
| `app/lib/notifications/providers/notification_seen_provider.dart` | Create | Serialize one seen call per render token, refresh count on success, and keep count untouched on failure | FR-014, RULE-005 | UT-008, AT-007, IT-005, REG-008 |
| `app/lib/notifications/providers/notification_preferences_provider.dart` | Create | Load known/unknown values and sequence optimistic edits per `(category, field)` with targeted rollback | FR-015–FR-017 | UT-010, UT-011, AT-008, IT-006, IT-009 |
| `app/lib/notifications/providers/notification_coordinator_provider.dart` | Create | Provider-neutral readiness projection and one keep-alive coordinator owner with injected callbacks/repositories | FR-003–FR-010, FR-013, FR-019, FR-022 | UT-012–UT-014, AT-002–AT-005, AT-009, AT-011, IT-002, IT-004, IT-010 |
| `app/lib/notifications/providers/notifications_provider.dart` | Change | Decode new model, dedupe by stable ID, preserve load-more behavior/render token, and expose explicit refresh | FR-007, FR-011, FR-014 | UT-005, AT-003, AT-006, AT-007, IT-003, REG-003 |
| `app/lib/notifications/widgets/notification_effect_host.dart` | Create | Single root lifecycle/effect bridge to `AppMessenger` and typed routes; no Firebase imports | FR-007–FR-010, FR-013, FR-019, FR-022, NFR-004 | AT-003, AT-004, AT-009, AT-011, IT-004, IT-007, IT-010 |
| `app/lib/notifications/widgets/notification_badge.dart` | Create | Format/hide `0`, cap `100+` as `99+`, and expose localized semantics | FR-013, NFR-004 | UT-006, AT-007, IT-008 |
| `app/lib/notifications/widgets/notification_row.dart` | Change | Render seven/unknown/tombstone variants; navigate known available data, resolve unknown/generic rows, and show unavailable feedback | FR-011, FR-012, NFR-004 | UT-005, AT-006, IT-007 |
| `app/lib/notifications/pages/notifications_page.dart` | Change | Add settings action, refresh trigger, success-render seen scheduling, and safe loading/error/empty states | FR-013–FR-015 | AT-007, IT-005, IT-008, REG-008 |
| `app/lib/notifications/pages/notification_settings_page.dart` | Create | Dedicated scrollable route with seven scope/push controls, no master switch, account/device copy, warning, retry, and Open settings | FR-015–FR-018, FR-024, RULE-004, RULE-008 | AT-008, IT-009, MAN-004 |
| `app/lib/app.dart` | Change | Mount `NotificationEffectHost` once inside the ready `MaterialApp.router` builder | FR-019, FR-022 | AT-009, AT-011, IT-010, REG-009 |
| `app/lib/bootstrap.dart` | Change | Obtain the Firebase-backed or unavailable service before the production `ProviderScope`; report only safe init classification and never block usable UI | FR-001, FR-006, NFR-003 | AT-001, AT-002, IT-010 |
| `app/lib/router/router.dart`, `route_locations.dart`, generated `router.g.dart` | Change / Generate | Add `/notifications/settings` as a typed child lifted to the root navigator so it covers the shell and Back returns to Notifications | FR-015 | AT-008, IT-009 |
| `app/lib/router/app_shell.dart` | Change | Watch new count and wrap both compact/rail notification icons in the same accessible badge widget | BR-003, FR-013, NFR-004 | UT-006, AT-007, IT-008 |
| `app/lib/auth/providers/auth_controller.dart`, `auth_session_provider.dart` | Change | Invoke notification cleanup with current DID and confirmed/unconfirmed mode before session state is cleared | FR-021 | UT-019, AT-010, IT-011 |
| `app/lib/shared/api/providers/sign_out_on_401_interceptor.dart` | Change | Make 401 cleanup asynchronous, ordered, and single-flight before local session clearing/sign-out state | FR-021 | AT-010, IT-011 |
| `app/lib/shared/api/providers/error_mapping_interceptor.dart` | Change | Classify fixed notification routes and ID routes without including path parameters in diagnostics | NFR-001 | REG-002 |
| `app/lib/l10n/app_en.arb`, generated localization files | Change / Generate | Add banner action, badge semantics, seven row/category labels, tombstones, settings/account-device guidance, and failure feedback | FR-007, FR-011–FR-018, FR-024, NFR-004 | UT-017, AT-003, AT-006–AT-008, IT-007–IT-009 |
| `app/android/settings.gradle.kts`, `app/android/app/build.gradle.kts` | Change | Apply generated Firebase/Google Services setup and set namespace/application ID to `social.craftsky.app` | FR-001 | AT-001, REG-006 |
| `app/android/app/google-services.json` | Create | Checked-in Firebase Android client configuration for `craftsky-app` / `social.craftsky.app` | FR-001 | AT-001, REG-006 |
| `app/android/app/src/main/AndroidManifest.xml` | Change | Add notification permission and default-channel metadata referencing the shared channel-ID string resource | FR-023, DR-001 | AT-012, REG-006, MAN-001 |
| `app/android/app/src/main/res/values/strings.xml` | Create/Change | Single source for channel ID/name used by manifest and Kotlin | FR-023 | AT-012, REG-006 |
| `app/android/app/src/main/kotlin/social/craftsky/app/MainActivity.kt` | Move/Change | Match namespace and create exactly one default-importance channel with standard sound/vibration | FR-001, FR-023 | AT-001, AT-012, REG-006, MAN-001 |
| `app/ios/Runner/GoogleService-Info.plist` | Create | Checked-in Firebase iOS client configuration for `craftsky-app` / `social.craftsky.app` | FR-001 | AT-001, REG-006 |
| `app/ios/Runner/Runner.entitlements`, `Info.plist`, `Runner.xcodeproj/project.pbxproj` | Create/Change | Set bundle ID, add Firebase plist resource, enable Push Notifications and remote-notification background mode, and wire entitlements | FR-001, FR-023 | AT-001, AT-012, REG-006, MAN-002 |
| `appview/internal/push/firebase_sender.go`, `_test.go` | Change | Add APNs `aps.sound = "default"`; assert complete payload parity otherwise | FR-025 | AT-012, IT-014, REG-007, MAN-002 |
| `appview/internal/app/config.go`, `push_config_test.go` | Change | Keep false default and require project ID whenever `PUSH_ENABLED=true`, including dev | FR-026 | IT-015, REG-010, MAN-005 |
| `appview/environments/dev.env` | Change | Make `PUSH_ENABLED=false` explicit and document temporary manual override without credentials in source | FR-026 | IT-015, MAN-005 |
| `app/test/notifications/**`, named auth/router/l10n/observability tests, and focused AppView tests | Create/Change | Implement UT-001–UT-019, AT-001–AT-012, IT-001–IT-015, REG-001–REG-010 using fakes/mocked Dio/static inspection | All | All automated IDs |

## 5. Services, Interfaces, And Data Flow

### 5.1 Provider-neutral delivery contract

Firebase types stop at the adapter. Domain identifiers are validated wrappers whose `toString()` is redacted; only API/storage adapters can access their wire value. `accountSubscriptionId` remains opaque: validate it as a bounded ASCII identifier and compare exact values, rather than deriving account identity from it. `notificationId` uses the AppView UUID shape. Push `type` uses a bounded ASCII identifier (`^[A-Za-z][A-Za-z0-9]{0,63}$`); a valid unknown value maps to `NotificationCategory.unknown`, while malformed values reject the event.

```text
enum NotificationPermission { notDetermined, authorized, denied }
enum NotificationOpenSource { foregroundBanner, backgroundOpen, initialOpen }

NotificationOpenEvent? parseProviderData(Map<String, Object?> data)
  // Reads only notificationId, type, accountSubscriptionId.
  // Ignores destination-like and unknown extra keys.

abstract interface class NotificationService {
  Future<void> initialize();
  Future<void> dispose();
  Future<NotificationPermission> getPermission();
  Future<NotificationPermission> requestPermission(); // alert/sound, no badge
  Future<String?> getToken();
  Stream<String> get tokenRefreshes;
  Stream<ForegroundNotificationReceipt> get foregroundReceipts;
  Stream<NotificationOpenEvent> get openedNotifications;
  Future<NotificationOpenEvent?> takeInitialOpen();
  Future<void> deleteToken();
  Future<void> openSystemNotificationSettings();
}
```

The adapter calls `setForegroundNotificationPresentationOptions(alert: false, badge: false, sound: false)` and emits app-owned receipts instead. The top-level background handler does not parse, persist, navigate, mutate Riverpod, or log a message; OS presentation comes from the existing combined notification-and-data payload.

### 5.2 Narrow AppView repositories

Keep presentation/providers decoupled from Dio while avoiding one large notification interface:

```text
NotificationRepository.list(cursor?, limit?) -> NotificationPage
NotificationDeviceRepository.register(platform, token) -> AccountSubscriptionId
NotificationNewCountRepository.fetch() -> int
NotificationSeenRepository.markSeen() -> void
NotificationResolutionRepository.resolve(NotificationId) -> NotificationResolution
NotificationPreferencesRepository.load() -> NotificationPreferences
NotificationPreferencesRepository.patch(NotificationPreferencePatch) -> NotificationPreferences
```

`NotificationApiClient` is the only HTTP implementation and uses the existing session Dio, bearer token, device-ID header, camelCase bodies, and standard error mapping. Requests are exact:

```text
POST  /v1/notifications/devices
  {"platform":"ios|android","token":"..."}
  -> {"accountSubscriptionId":"..."}

GET   /v1/notifications/new-count -> {"newCount":N}
POST  /v1/notifications/seen      -> bodyless 204
GET   /v1/notifications/{notificationId}
GET   /v1/notifications/preferences
PATCH /v1/notifications/preferences
  {"preferences":{"like":{"pushEnabled":false}}}
```

No client request sends a DID. Existing middleware selects the current account.

### 5.3 Registration and secure binding flow

```text
readiness becomes signed-in + onboarded
  -> check permission without prompting
  -> if notDetermined: request once directly
  -> if authorized: get latest token
  -> register(platform, token)
  -> store returned opaque routing ID under current DID
  -> refresh new count

token refresh
  -> remember latest token in memory
  -> if eligible now: register
  -> otherwise: defer until next eligible readiness/resume trigger
```

Registration errors keep the app ready. There is no tight retry loop: the next readiness change, resume, token refresh, or permission recovery is the retry opportunity. Craftsky does not persist the FCM token; Firebase owns provider token storage.

Secure storage contains one feature key with a DID-to-routing-ID map. Re-registration replaces only the current DID value. Sign-out removes only that DID and never clears `craftsky_device_id` or another future DID binding.

### 5.4 Open and navigation flow

```text
provider open or foreground banner action
  -> normalized NotificationOpenEvent
  -> if transient bootstrap/router/onboarding: hold one in-memory pending open
  -> if actual sign-in required: discard
  -> look up secure binding for current DID
  -> exact routing-ID match?
       no  -> generic unavailable effect; no HTTP; no navigation
       yes -> GET owner-scoped notification resolution
  -> map authorized target only
       post URI      -> parseCraftskyPostUri -> PostThreadRoute
       actor DID     -> UserProfileRoute(handle: did), supported by profile API
       notifications -> NotificationsRoute
       404/unavailable/retracted -> NotificationsRoute
       offline/timeout -> NotificationsRoute + unable-to-open feedback
```

The coordinator keeps one non-persistent pending slot only while the same existing account is transiently becoming ready; a later open replaces an earlier still-pending open. This is readiness bounding, not receipt deduplication. Once ready, every callback is processed normally. No provider destination field, DID, handle, or URI is trusted.

Known available feed rows continue to navigate from AppView-hydrated data. Unknown and generic/everythingElse rows call the same resolution repository by stable notification ID. Unavailable rows do not navigate.

### 5.5 Foreground, count, and seen flow

Every valid foreground receipt performs three independent actions:

1. Emit a banner effect. `NotificationEffectHost` presents the provider-visible generic title/body through `AppMessenger.info` with a localized `MessageAction` that invokes the shared open flow.
2. Invalidate the first notification page.
3. Refresh new count using the `foregroundReceipt` trigger.

There is no page-visibility check, duplicate-ID check, sound/vibration call, or local notification.

`NotificationNewCount` accepts exactly `ready`, `resume`, `foregroundReceipt`, `pageRefresh`, and `markSeen` triggers. The app shell watches it and renders the same badge widget for bottom navigation and rail. A timer, periodic provider, platform icon-badge call, or elapsed-time trigger is prohibited.

The feed state creates one opaque first-page render token for each successful first-page load and preserves it during load-more. `NotificationsPage` schedules a post-frame callback only after it built content or the empty state. `NotificationSeen` consumes each token once, calls bodyless seen, and refreshes count after success. Loading, errors, route entry, prefetch, coordinator invalidation, and count/list GETs cannot call seen. A seen failure leaves the badge state unchanged and allows a later successfully rendered first page to retry.

### 5.6 Preference mutation flow

The decoder retains raw unknown server entries separately from the seven known UI values. The page iterates the closed known category order only. A mutation key is `(category, scope|pushEnabled)` and each key has a monotonically increasing local generation:

```text
edit(key, value)
  previous = current field value
  generation = nextGeneration(key)
  state = optimistic value immediately
  PATCH only {category: {field: value}}
  on success if generation is still current:
    retain server value for that field; preserve other optimistic fields
  on failure if generation is still current:
    restore previous field only; emit localized failure sequence
  stale completion:
    do not overwrite or roll back newer state
```

OS denial only adds a current-device warning and Open settings action. It never disables, rewrites, or patches account-wide preferences and never invokes the OS prompt again.

### 5.7 Sign-out cleanup

`NotificationSignOutCleanup` is idempotent/single-flight for the current DID and is invoked before the session blob or auth provider is cleared:

```text
confirmed AppView logout
  -> retain Firebase token
  -> remove current DID routing binding
  -> clear session and publish SignedOut

failed AppView logout / 401-forced local sign-out
  -> best-effort delete Firebase token
  -> remove current DID routing binding
  -> clear session and publish SignedOut
```

Token deletion failure is classified without token/error payload details and never blocks binding/session cleanup. `SignOutOn401Interceptor` holds one in-flight cleanup future so concurrent 401s do not start duplicate cleanup. `AuthSession` keeps a fallback cleanup path for fake clients/background validation but shares the same idempotent service.

### 5.8 AppView follow-ups

The Firebase sender adds only the APNs default sound request under `APNS.Payload.Aps.Sound`. Its regression test compares token, notification copy, data map, Android TTL/config, APNs expiration, and result classification before asserting the new sound field.

`LoadConfig` continues to default `PUSH_ENABLED` to false in every environment. If explicitly true in dev or prod, `FIREBASE_PROJECT_ID` must be non-empty before dependency construction. `appview/environments/dev.env` records the normal `false` state explicitly. Credentials remain Application Default Credentials supplied outside the repository.

## 6. State, Providers, Controllers, Or DI

Use manual always-alive `Provider`/`AsyncNotifierProvider` declarations in the existing notification feature style unless code generation materially simplifies a family. The coordinator provider must not rebuild when readiness changes: it creates one owner, uses `ref.listen` to forward readiness, and disposes the owner only with the container.

```text
bootstrapFirebaseNotificationService()
  -> notificationServiceProvider override

authSessionProvider + onboardingStatusProvider(current DID)
  -> notificationReadinessProvider
  -> ref.listen inside notificationCoordinatorProvider (always alive)

notificationServiceProvider
notificationPermissionProvider
notification routing storage/repositories
notificationReadinessProvider
  -> notificationCoordinatorProvider
       owns service subscriptions once
       exposes private effect stream

notificationCoordinatorProvider.effects
  -> NotificationEffectHost (one active UI subscription)
       -> AppMessenger
       -> typed GoRouter routes
       -> lifecycle resume callback

notificationNewCountRepositoryProvider
  -> notificationNewCountProvider
       -> AppShell badge

notificationRepositoryProvider
  -> notificationsProvider
       -> NotificationsPage render token
       -> notificationSeenProvider
       -> notificationNewCountProvider refresh

notificationPreferencesRepositoryProvider
notificationPermissionProvider
  -> notificationPreferencesProvider
  -> NotificationSettingsPage
```

Provider choices:

- `notificationServiceProvider`: `Provider<NotificationService>`, always alive, production override from bootstrap, fake-safe unavailable default.
- `notificationCoordinatorProvider`: `Provider<NotificationCoordinator>`, always alive, explicit `start`/`dispose`; one owner of provider streams.
- `notificationReadinessProvider`: derived `Provider<NotificationReadiness>` watching the current auth state and only that DID's onboarding provider.
- `notificationPermissionProvider`: `AsyncNotifierProvider`; `build/check` has no prompt side effect, while `requestIfEligible` is explicit.
- `notificationNewCountProvider`: `AsyncNotifierProvider<int>`; explicit trigger method, no periodic invalidation.
- `notificationsProvider`: retain existing `AsyncNotifierProvider` and pagination behavior, with stable-ID dedupe and first-page render token.
- `notificationSeenProvider`: `AsyncNotifierProvider<void>` or mutation controller with token guard; its state is not used to clear the badge optimistically.
- `notificationPreferencesProvider`: `AsyncNotifierProvider<NotificationPreferencesState>` with per-field generations and failure sequence.

Do not put provider events containing raw notification IDs, routing IDs, DIDs, URIs, tokens, or payload copy into Riverpod state because `ProviderLogger` observes state changes. Coordinator effects remain on its private stream, value types have redacted stringification, and logs use allowlisted lifecycle/outcome classes only.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Foreground banner

- Use the existing global `AppMessenger` and Craftsky snackbar styling rather than adding a local-notification or overlay subsystem.
- Display the provider-visible generic title/body as one informational banner.
- Supply a localized Open action wired to the same coordinator open event.
- Preserve the current messenger replacement policy. Every provider callback is still handled and tested; banner replacement is presentation behavior, not receipt suppression.
- Keep the banner silent and non-vibrating on every route, including Notifications.

### Notifications feed

- Keep `CustomScrollView`, pinned `SliverAppBar`, existing load-more concurrency/retry, and page-size conventions.
- Add a settings icon button to the app bar.
- Add pull/explicit refresh only if it reuses the existing page reload seam; its count refresh trigger is `pageRefresh` and it does not mark seen until the result renders.
- Known rows use localized category copy and safe hydrated destination data.
- `quote`, `everythingElse`, and unknown types receive explicit generic/category copy.
- Unavailable actors display localized “Someone”; unavailable content remains visible with category-appropriate tombstone copy and brief feedback on tap.
- Never display raw DID, handle, AT-URI, CID, notification ID, or provider data as fallback copy.

### App-shell badge

- Wrap both outlined and selected Notifications icons in `NotificationBadge` in compact navigation and rail.
- `0` renders no badge; `1..99` renders the number; `100+` renders `99+`.
- Semantics announce localized new activity independent of the visual cap.
- Do not touch the launcher/app icon.

### Notification Settings route

Add `RouteLocations.notificationSettingsChild = 'settings'` below `NotificationsRoute`, with `NotificationSettingsRoute.$parentNavigatorKey` set to the root key. The resulting path is `/notifications/settings`, covers bottom navigation, and pops back to Notifications.

The page is one scrollable `Scaffold` containing:

- App bar title and normal Back behavior.
- Introductory copy stating category preferences apply to all devices on the account.
- When denied, a current-device warning and Open settings action; no repeated prompt.
- Seven sections in AppView order: Like, Follow, Reply, Mention, Quote, Repost, Everything else.
- A two-value scope control (`Everyone`, `People I follow`) and independent push switch per category.
- No master switch and no controls for unknown server categories.
- Existing progress indicator, retry surface, accessible control labels/tap targets, and `AppMessenger` error feedback for targeted rollback.

### Native surfaces

- Android requests notification permission where required by the target SDK.
- `MainActivity` creates one `IMPORTANCE_DEFAULT` channel using the shared string-resource ID/name and standard sound/vibration.
- The manifest metadata `com.google.firebase.messaging.default_notification_channel_id` points to that same resource. No Android `channel_id` is added to the AppView payload.
- iOS enables Push Notifications and `remote-notification` background mode. Permission requests set alert/sound true and badge false.
- APNs authentication-key upload remains an external rollout step and is never committed.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Firebase initialization fails | Provide unavailable service, report safe classification, render normal app, retry only on later process start | FR-006, NFR-003 | AT-002, IT-010 |
| Permission not determined before readiness | Do not prompt | FR-003, RULE-001 | UT-001, AT-002 |
| Permission not determined at eligible readiness | Open direct OS request once; no primer | FR-003, RULE-001 | UT-001, AT-002, MAN-004 |
| Permission denied | No automatic re-prompt/registration; settings controls stay editable with current-device warning | FR-017, FR-018, RULE-004 | AT-008, IT-009, MAN-004 |
| Permission becomes authorized in OS settings | Resume refreshes permission, token registration, and count | FR-003, FR-004, FR-018 | IT-002, MAN-004 |
| Permission/token/registration call fails | Keep app usable; classify safely; retry on next eligible trigger | FR-006, NFR-003 | UT-001, UT-013, AT-002, IT-002 |
| Token is null/empty | Skip registration and wait for later token/trigger | FR-004–FR-006 | UT-013, IT-002 |
| Token refresh while signed out/incomplete | Retain latest token in memory only; register after eligible readiness | FR-005 | UT-013, IT-002 |
| Repeated registration | Replace/converge current DID binding with returned routing ID | FR-004 | UT-015, IT-001 |
| Malformed provider event | Reject without resolution, navigation, user identifier details, or raw logging | FR-008, NFR-001 | UT-002, REG-002 |
| Valid unknown bounded push type | Normalize to generic activity and continue binding validation/resolution | FR-008, DR-002 | UT-002, AT-004, IT-004 |
| Extra provider destination/identity fields | Ignore completely; never persist/log/use | RULE-003, NFR-001 | UT-002, REG-002 |
| Routing binding missing/malformed/stale/other account | No HTTP or navigation; generic unavailable feedback once ready | FR-009 | UT-003, AT-005, IT-012 |
| Open during bootstrap/router/onboarding | Retain one latest in-memory event for the same existing account | FR-022 | UT-014, AT-011, IT-004 |
| Open requires sign-in | Discard before login; never carry into another account | FR-022 | UT-014, AT-011, IT-004 |
| Resolution post target malformed/unavailable/404 | Open Notifications without inferred destination | FR-010 | UT-004, AT-004, IT-004 |
| Resolution offline/timeout | Open Notifications, show unable-to-open feedback, persist no retry | FR-010 | UT-004, AT-004, IT-004, REG-004 |
| Repeated foreground callback | Emit banner + list/count refresh for every callback; no dedupe | FR-007, RULE-007 | UT-018, AT-003, REG-004 |
| Foreground callback while Notifications visible | Same banner and refresh behavior; page visibility does not suppress | FR-007 | AT-003, IT-007 |
| Feed unknown category | Render generic New activity and resolve on tap | FR-011 | UT-005, AT-006, IT-003 |
| Feed unavailable actor/content | Visible safe tombstone; no raw fallback or navigation | FR-012 | UT-005, AT-006, IT-007 |
| First page loading/error/route entry/prefetch | No seen request; retain badge | FR-014, RULE-005 | UT-008, AT-007, IT-005, REG-008 |
| First page content or empty renders | One post-frame seen request for that render token; refresh count on success | FR-014 | UT-008, AT-007, IT-005 |
| New activity during seen request | Server snapshot semantics plus post-success count refresh preserve newer newness | FR-013, FR-014 | IT-005 |
| Seen fails | Do not optimistically clear count; retry after a later successful first-page render | FR-014 | IT-005, REG-008 |
| Count 0 / 1 / 99 / 100+ | Hide / literal / literal / `99+`; localized semantics use actual server count | FR-013, NFR-004 | UT-006, IT-008 |
| Long foreground session | No timer/poll; refresh only on approved next trigger | FR-013 | UT-007, REG-004 |
| Preference initial load fails | Full-page error with Retry; no guessed values | FR-015 | AT-008, IT-009 |
| Preference PATCH fails | Roll back only still-current field generation and show error | FR-016 | UT-010, AT-008, IT-006, IT-009 |
| Overlapping preference mutations | Stale success/failure cannot overwrite newer/sibling optimistic state | FR-016 | UT-010, IT-006 |
| Unknown preference category | Retain in decoded state, render no control, omit from known PATCH | FR-016 | UT-009, UT-011, AT-008, IT-006 |
| Successful logout | Remove current binding, preserve token/other binding/device ID | FR-021 | UT-019, AT-010, IT-011 |
| Failed logout or 401; token deletion fails | Attempt deletion before session clear, then always remove binding and complete local sign-out | FR-021 | UT-019, AT-010, IT-011 |
| Background handler invoked | Complete without UI/provider mutation or payload logging | FR-020 | IT-013, REG-005 |
| Non-production normal startup | Push dispatcher disabled; no real Firebase sender | FR-026 | IT-015, MAN-005 |
| Explicit push enablement missing project ID | Fail AppView config validation with safe key name | FR-026 | IT-015 |

## 9. Test Implementation Plan

The order is red-green-refactor order. Each target uses fake `NotificationService`, fake narrow repositories, mocked Dio, or static source inspection; normal automated Flutter tests never initialize Firebase or contact FCM.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | UT-002 | `app/test/notifications/models/notification_open_event_test.dart` | TD-002 known/future/malformed/extra/sentinel payload matrix | Domain parser/value types do not exist |
| 2 | UT-003, UT-015 | Routing policy/storage tests | TD-003 two DIDs, replacement, malformed/stale values, fake secure storage | No DID-keyed routing seam |
| 3 | UT-004 | `notification_resolution_policy_test.dart` | TD-005 target/failure matrix | No authorized navigation-intent policy |
| 4 | UT-001 | `notification_permission_policy_test.dart` | TD-001 auth/onboarding/permission matrix | No explicit readiness permission policy |
| 5 | UT-013 | `notification_registration_coordinator_test.dart` | Token A/B, null, pre/post readiness, failure/resume | No deferred latest-token registration |
| 6 | IT-001, IT-002, AT-002 | Registration/coordinator tests | Fake service + mocked Dio + readiness transitions | No registration endpoint/provider orchestration |
| 7 | UT-012, IT-010, AT-009 | Service-owner/provider lifecycle tests | Initial open, live streams, rebuild/readiness/dispose counters | No one-owner initialization/subscription lifecycle |
| 8 | UT-014, AT-011 | Pending-open tests | Bootstrap/onboarding/signed-out transitions | No bounded in-memory readiness slot |
| 9 | AT-001, REG-001, REG-005, REG-006 | Bootstrap/import/native static tests | Source/project inspection; fake bootstrap result | No Firebase/native boundary or configuration |
| 10 | UT-016, AT-012 | Presentation/native policy tests | Permission options, foreground options, channel linkage fixture | Sound/badge/channel policy absent |
| 11 | UT-018, AT-003, IT-007 | Foreground service/host widget tests | Recording messenger + duplicate fake receipts on multiple routes | No banner/effect/list-count invalidation flow |
| 12 | AT-004, AT-005, IT-004, IT-012 | Open coordinator/routing tests | Matching/nonmatching bindings, test router, mocked resolution | No safe shared open flow |
| 13 | UT-005, IT-003, AT-006 | Durable model/API/page tests | TD-004 seven/unknown/quote/availability matrix | Legacy decoder throws or assumes availability |
| 14 | UT-006, UT-007, IT-008 | Badge/new-count provider/shell tests | TD-006 counts and approved/unapproved triggers | No count state/badge/trigger policy |
| 15 | UT-008, IT-005, AT-007, REG-008 | Seen policy/provider/page tests | Loading/error/content/empty/render token/concurrent count | No successful-render acknowledgement gate |
| 16 | UT-009, UT-011, IT-006 | Preference model/API tests | TD-007 seven known + one unknown; one-field bodies | No preference decoder/partial PATCH |
| 17 | UT-010 | Preference provider race tests | Same/different-field overlapping completion orders | No mutation-generation rollback protection |
| 18 | AT-008, IT-009 | Settings page/typed route tests | Authorized/denied service, test router, success/failure patches | No full-screen settings UI |
| 19 | UT-019, AT-010, IT-011 | Sign-out cleanup/auth/401 tests | TD-003 plus confirmed/failed/401/delete failure | Existing paths clear session without notification cleanup |
| 20 | UT-017 | `app/test/l10n/notifications_l10n_test.dart` | Generated localization surface and semantics finders | New user-facing copy/labels absent |
| 21 | REG-002, REG-003, REG-004, REG-009 | Observability/structural/nearby suite | Sentinel scan, endpoint categories, no timer/store/queue, one owner | New privacy/absence guards absent |
| 22 | IT-013 | Background handler test | Constrained no-widget/provider harness with sentinels | Handler entry point absent |
| 23 | IT-014, REG-007 | `appview/internal/push/firebase_sender_test.go` | Complete captured message before/after comparison | Default APNs sound absent |
| 24 | IT-015, REG-010 | `appview/internal/app/push_config_test.go` plus focused suites | Dev/prod false/true/project-ID matrix | Dev enabled path accepts missing project ID |
| 25 | MAN-001–MAN-005 | Physical Android/iOS and bounded AppView session | TD-009 with explicit start/end gate | Native/provider path not proven by automation |

Focused commands by phase:

```text
# First TDD step, from app/
flutter test test/notifications/models/notification_open_event_test.dart

# Pure/service/provider notification loop, from app/
flutter test test/notifications

# Cross-feature auth/router/observability regression, from app/
flutter test test/auth test/onboarding test/router test/notifications test/observability test/shared/api/providers

# Generation and analysis, from app/
dart run build_runner build
dart analyze

# Narrow AppView follow-ups, from appview/
go test ./internal/push ./internal/app -count=1

# Repository-level canonical checks
just app-test
just app-analyze
just test
```

Native/provider checks run only after all automated suites are green and the manual session starts with `PUSH_ENABLED=false`. MAN-001/MAN-002 temporarily enable sending for the named devices and restore/verify false afterward.

## 10. Sequencing And Guardrails

- First TDD step: write UT-002 for the allowlisted provider-data parser, including one valid future `type`, malformed required values, ignored destination-shaped keys, and sensitive sentinels.
- Dependencies between work items:
  - Domain identifiers/parser precede routing, service adapter, coordinator, and navigation.
  - Routing storage/policy precede any resolution HTTP call.
  - Permission and latest-token registration policy precede Firebase listeners/native setup.
  - One-owner lifecycle tests precede mounting the root effect host.
  - Authorized open policy precedes foreground banner and OS-open navigation.
  - Durable model/API alignment precedes page/tombstone rendering.
  - Count provider precedes badge; render-token policy precedes seen mutation.
  - Preference model/PATCH tests precede optimistic provider and settings UI.
  - Sign-out cleanup service precedes modifications to explicit logout, auth validation, and 401 interception.
  - AppView sound/config tests remain a final narrow server slice.
- Guardrails:
  - Firebase-specific types/imports are allowed only in Firebase adapter/bootstrap/background files and generated configuration.
  - Firebase initialization failure never blocks the core ready app.
  - Exactly one keep-alive coordinator owns provider streams; widgets never subscribe to `FirebaseMessaging`.
  - Parse/validate provider data once at the adapter/domain boundary; trust typed/redacted values internally.
  - AppView resolution is the only push-open destination authority.
  - Require exact current-DID routing binding before owner-scoped resolution.
  - Never log/stringify raw messages, tokens, routing IDs, notification IDs, DIDs, handles, AT-URIs, provider copy, Firebase credentials, or resolution paths.
  - Never persist FCM tokens, receipts, provider messages, notification content, open events, or deep-link retries in Craftsky storage.
  - Do not suppress foreground banners on Notifications and do not dedupe repeated callbacks.
  - Do not use foreground OS presentation, local notifications, sound, vibration, or haptics.
  - Only a post-frame successful first-page render may call seen; GETs and invalidations stay read-only.
  - New count has five explicit triggers and no timer/polling/icon-badge side effect.
  - Preference PATCH contains exactly one known category and one changed field; mutation generations prevent stale rollback.
  - Successful server logout retains the FCM token; unconfirmed cleanup attempts deletion before session clear; every path removes only the current DID binding and preserves device ID.
  - Android manifest metadata and channel creation use the same resource ID; keep the AppView Android payload unchanged.
  - The AppView change is limited to APNs default sound and enabled-config validation; no route, payload data, eligibility, delivery, database, or lexicon change.
  - `PUSH_ENABLED=false` is the verified state before and after every non-production physical-device check.
  - Preserve unrelated dirty-worktree changes; this stage edits only `04-coding-plan.md`.
- Out of scope:
  - Web/desktop push, raw APNs, multi-account UI/simultaneous sessions, local inbox/cache, per-item read state, per-device unread state.
  - Home-screen app-icon badges, periodic polling, persistent/session receipt dedupe, deferred-deep-link queue.
  - Permission primer, master push switch, category Android channels, custom sounds, rich notifications, action buttons, grouping/digests.
  - New AppView routes/contracts, server notification eligibility/lifecycle/schema changes, PDS/lexicon changes, production credential storage.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Blocking | High-risk implementation approval has not yet been given for this stage | Source/native/dependency work must not begin | Ask for explicit continuation with `implement-tdd` after this plan |
| CPQ-002 | Non-blocking rollout | APNs authentication key is not yet uploaded/configured in Firebase | Blocks MAN-002 and iOS launch delivery, not automated implementation | User/operator configures key outside repo before MAN-002 |
| CPQ-003 | Resolved | Exact foreground banner seam | Could otherwise create a second presentation system | Use existing `AppMessenger.info` plus localized `MessageAction`; no local-notification dependency |
| CPQ-004 | Resolved | Unknown push-type policy from DR-002 | Parser forward compatibility | Accept bounded identifier syntax and normalize unknown to generic while still requiring both IDs |
| CPQ-005 | Resolved | Android channel drift from DR-001 | Background messages could use FCM fallback channel | One string-resource channel ID is consumed by Kotlin and manifest metadata; REG-006 asserts equality |
| CPQ-006 | Resolved | Native Open settings implementation | Cross-platform recovery action needs a stable adapter seam | Use a small app-settings launcher dependency behind `NotificationService`; no plugin types escape |
| CPQ-007 | Non-blocking implementation prerequisite | FlutterFire CLI/session or downloaded Firebase client files may be unavailable in the implementation environment | Native config generation could pause while pure Dart/AppView TDD continues | Generate against existing `craftsky-app` registrations; if access is unavailable, request only the two client config files, never credentials |
| CPQ-008 | Resolved | Profile resolution returns DID while typed route is named `handle` | Could tempt an unauthorized client-side identity lookup | Pass DID to `UserProfileRoute`; existing profile API/provider accepts handle or DID |
| CPQ-009 | Resolved | Multiple opens while transiently not ready | A persistent queue is out of scope | Keep one in-memory latest-wins slot for the existing account; process every callback normally once ready |
| CPQ-010 | Residual | Best-effort token deletion cannot recall already accepted provider delivery | A signed-out device may briefly receive stale queued notification | Clear local binding/session, rely on AppView/provider lifecycle, document in MAN-003; do not block sign-out |
| CPQ-011 | Residual | Shared Firebase project across environments | Accidental dev delivery reaches shared-project devices | Explicit false default, project validation when enabled, bounded MAN gate, and final disabled verification |
| CPQ-012 | Residual | Static native tests cannot prove signing/APNs/OS behavior | Source can be correct while delivery fails | Require MAN-001 and MAN-002 on physical devices before launch enablement |

No product or architecture question blocks a useful coding plan. CPQ-001 is the workflow approval gate, not a missing design decision.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md` will be created/maintained by `implement-tdd` during the approved implementation stage.
- Start with test: UT-002, `app/test/notifications/models/notification_open_event_test.dart`
- Focused command: from `app/`, `flutter test test/notifications/models/notification_open_event_test.dart`
- First production target after the red test: `app/lib/notifications/models/notification_open_event.dart`; no Firebase import or initialization.
- Next slice: UT-003/UT-015 routing policy and secure DID-keyed binding storage.
- Notes:
  - Keep strict red-green-refactor order and the test sequence in Section 9.
  - Do not perform native/provider manual checks until automated suites pass and the bounded sender gate is recorded.
  - Stop for explicit user approval before dependency, source, native project, AppView, or generated-file changes.
