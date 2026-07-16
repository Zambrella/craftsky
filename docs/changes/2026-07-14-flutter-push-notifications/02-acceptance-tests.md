# Acceptance Test Specification: Flutter Push Notifications

## 1. Test Strategy

This high-risk change requires layered, predominantly automated coverage while keeping Firebase and native provider behavior outside the normal test process. Pure permission, payload-validation, routing, badge, acknowledgement, preference, and cleanup rules belong in Dart unit tests. Riverpod and widget tests with fake provider-neutral services cover auth/onboarding readiness, listener ownership, foreground banners, navigation, badge rendering, successful-render acknowledgement, and settings behavior. Mocked-Dio tests cover the existing AppView device, list, count, seen, resolution, and preference contracts without contacting a server. Focused Go tests cover the narrow AppView default-APNs-sound change and the non-production sender gate. Static regression tests protect Firebase import boundaries, background-handler shape, native configuration, listener ownership, and sensitive-data redaction.

Automated Flutter tests must not initialize Firebase, contact FCM, request real OS permission, open native settings, or depend on physical-device state. One physical Android device and one physical iOS device remain necessary for provider delivery, native permission, background/terminated presentation, sound/vibration, token lifecycle, and settings-recovery checks. A manual non-production delivery session must begin with `PUSH_ENABLED=false`, enable sending only for the bounded check, and restore the disabled state afterward.

The highest-risk seam is one keep-alive notification coordinator consuming provider-neutral streams. Its tests must prove readiness gating, one initial-message consumption, one live subscription per stream, secure DID-keyed routing validation before resolution, generic normalization of syntactically valid future push `type` values, safe disposal, and no carry-over through a required sign-in. The other critical seam is the Notifications page's successful-first-page-render acknowledgement: route entry, loading, failure, prefetch, foreground receipt, and list/count reads must remain read-only. Native configuration tests must also prove that FCM's Android manifest default-channel metadata points to the one created Craftsky channel, because the AppView payload intentionally omits Android `channel_id`.

Risk level: **High** (carried forward). Test design found no blocking coverage gap, but document review and explicit approval remain required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-004 | AT-001, AT-002, IT-001, REG-006, MAN-001, MAN-002 | Acceptance / Integration / Regression / Manual | Partial |
| BR-002 | AC-006, AC-007 | AT-003, AT-004, IT-004, IT-007, MAN-001, MAN-002 | Acceptance / Integration / Manual | Partial |
| BR-003 | AC-011, AC-012 | AT-007, UT-006, UT-007, UT-008, IT-005, IT-008 | Acceptance / Unit / Integration | Yes |
| BR-004 | AC-013, AC-014 | AT-008, UT-009, UT-010, IT-006, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001 | AT-001, REG-006, MAN-001, MAN-002 | Acceptance / Regression / Manual | Partial |
| FR-002 | AC-002, AC-015 | REG-001, IT-010 | Regression / Integration | Yes |
| FR-003 | AC-003 | AT-002, UT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-004 | AT-002, IT-001, UT-015 | Acceptance / Integration / Unit | Yes |
| FR-005 | AC-005 | UT-013, IT-002, MAN-003 | Unit / Integration / Manual | Yes |
| FR-006 | AC-003, AC-005 | AT-002, UT-001, UT-013, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-006, AC-019 | AT-003, UT-018, IT-007, MAN-001, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| FR-008 | AC-007 | AT-004, UT-002, IT-004, MAN-001, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| FR-009 | AC-008 | AT-005, UT-002, UT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-007, AC-009, AC-025 | AT-004, UT-004, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-010, AC-028 | AT-006, UT-005, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-010, AC-029 | AT-006, UT-005, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-011, AC-020 | AT-007, UT-006, UT-007, IT-008, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-014 | AC-012, AC-021 | AT-007, UT-008, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-013, AC-022 | AT-008, IT-009 | Acceptance / Integration | Yes |
| FR-016 | AC-014, AC-023 | AT-008, UT-009, UT-010, UT-011, IT-006, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-013 | AT-008, IT-009 | Acceptance / Integration | Yes |
| FR-018 | AC-003, AC-013 | AT-002, AT-008, IT-009, MAN-004 | Acceptance / Integration / Manual | Partial |
| FR-019 | AC-015 | AT-009, UT-012, IT-010, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| FR-020 | AC-016 | AT-009, REG-005 | Acceptance / Regression | Yes |
| FR-021 | AC-017, AC-024 | AT-010, UT-019, IT-011, MAN-003 | Acceptance / Unit / Integration / Manual | Yes |
| FR-022 | AC-026 | AT-011, UT-014, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-023 | AC-027 | AT-012, REG-006, MAN-001, MAN-002 | Acceptance / Regression / Manual | Partial |
| FR-024 | AC-013 | AT-008, IT-009, MAN-004 | Acceptance / Integration / Manual | Yes |
| FR-025 | AC-027 | IT-014, REG-007, MAN-002 | Integration / Regression / Manual | Partial |
| FR-026 | AC-030 | IT-015, MAN-005 | Integration / Manual | Yes |
| NFR-001 | AC-018 | UT-002, REG-002 | Unit / Regression | Yes |
| NFR-002 | AC-015, AC-018 | IT-010, REG-001, REG-002 | Integration / Regression | Yes |
| NFR-003 | AC-003, AC-005 | AT-002, UT-013, IT-002 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-006, AC-011, AC-013 | AT-003, AT-007, AT-008, IT-007, IT-008, IT-009 | Acceptance / Integration | Yes |
| RULE-001 | AC-003 | AT-002, UT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-006 | AT-003, IT-007, MAN-001, MAN-002 | Acceptance / Integration / Manual | Partial |
| RULE-003 | AC-007, AC-008, AC-009 | AT-004, AT-005, UT-002, UT-003, UT-004, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-013 | AT-008, IT-009, MAN-004 | Acceptance / Integration / Manual | Yes |
| RULE-005 | AC-012 | AT-007, UT-008, IT-005, REG-008 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-006 | AC-004, AC-008 | AT-002, AT-005, UT-003, UT-015, IT-012 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-019 | AT-003, UT-018, REG-004 | Acceptance / Unit / Regression | Yes |
| RULE-008 | AC-013 | AT-008, IT-009, MAN-004 | Acceptance / Integration / Manual | Yes |

Every Must requirement has at least one automated test. “Partial” means the contract and application behavior are automated but the last native/provider assertion also requires a physical-device check. Both Should requirements are automated because startup resilience, localization, and accessibility are material risks for this feature. Every AC-001 through AC-030 is covered.

## 3. Acceptance Scenarios

### AT-001: Native applications initialize the one Firebase project first
Requirement IDs: BR-001, FR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/firebase_configuration_test.dart`, `app/test/bootstrap/firebase_bootstrap_test.dart`

```gherkin
Feature: Native Firebase setup
  Scenario: Android and iOS start with the shared identity and project
    Given the checked-in Android and iOS Firebase configuration
    When application dependencies are initialized in any build environment
    Then both native applications identify as social.craftsky.app
    And both configurations select Firebase project craftsky-app
    And Firebase initialization completes before messaging-dependent services are created
```

### AT-002: Eligible readiness requests permission and registers safely
Requirement IDs: BR-001, FR-003, FR-004, FR-006, FR-018, NFR-003, RULE-001, RULE-006
Acceptance Criteria: AC-003, AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_coordinator_test.dart`

```gherkin
Scenario Outline: Readiness drives permission and registration without blocking the app
  Given the app is <auth> and onboarding is <onboarding>
  And notification permission is <permission>
  When notification coordination becomes active
  Then the OS permission request count is <promptCount>
  And authenticated registration occurs only when permission is authorized and a token exists
  And any provider or registration failure leaves the ready application usable

  Examples:
    | auth       | onboarding | permission    | promptCount |
    | signed out | incomplete | notDetermined | 0           |
    | signed in  | incomplete | notDetermined | 0           |
    | signed in  | complete   | notDetermined | 1           |
    | signed in  | complete   | authorized    | 0           |
    | signed in  | complete   | denied        | 0           |
```

### AT-003: Every foreground callback shows one silent banner and refreshes data
Requirement IDs: BR-002, FR-007, NFR-004, RULE-002, RULE-007
Acceptance Criteria: AC-006, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/foreground_notification_test.dart`

```gherkin
Scenario: Duplicate foreground callbacks are handled normally
  Given the app is foregrounded on the Notifications page
  When the fake provider emits the same valid foreground notification twice
  Then two localized Craftsky banners are presented in callback order
  And each callback invalidates the list and new-count state
  And neither callback creates an OS alert, sound, vibration, or receipt record
```

### AT-004: Every notification open uses one authorized resolution flow
Requirement IDs: BR-002, FR-008, FR-010, FR-022, RULE-003
Acceptance Criteria: AC-007, AC-009, AC-025, AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_open_coordinator_test.dart`, `app/test/router/notification_open_routing_test.dart`

```gherkin
Scenario Outline: A valid open navigates only from AppView resolution
  Given an authenticated account has the matching secure routing binding
  And a <source> open contains one validated notification ID
  When AppView resolution returns <outcome>
  Then the client navigates to <destination>
  And no destination is reconstructed from provider data
  And no retryable deep link is persisted
  And a syntactically valid unknown provider type is represented as generic activity before the same resolution flow

  Examples:
    | source             | outcome          | destination                  |
    | foreground banner  | post target       | authorized post thread       |
    | background open    | actor target      | authorized actor profile     |
    | terminated open    | notifications     | Notifications                |
    | background open    | unavailable or 404| Notifications                |
    | terminated open    | network timeout   | Notifications plus feedback  |
```

### AT-005: Stale or cross-account opens never resolve
Requirement IDs: FR-009, RULE-003, RULE-006
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_open_coordinator_test.dart`

```gherkin
Scenario Outline: An invalid routing binding is unavailable
  Given the ready app has one current authenticated DID
  And the notification open has a <binding>
  When the coordinator validates the open
  Then AppView resolution is not called
  And no navigation occurs
  And generic unavailable feedback appears without identifier details

  Examples:
    | binding                    |
    | missing                    |
    | malformed                  |
    | stale for the current DID  |
    | valid for a different DID  |
```

### AT-006: The durable notification feed remains safe and usable
Requirement IDs: FR-011, FR-012
Acceptance Criteria: AC-010, AC-028, AC-029
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/models/notification_test.dart`

```gherkin
Scenario: All durable and forward-compatible rows render safely
  Given a page containing all seven known categories, one unknown category, unavailable references, and an unavailable actor
  When the page is decoded, rendered, and each row is tapped
  Then known available rows use category-appropriate localized content and navigation
  And the unknown row displays generic New activity and resolves through AppView
  And unavailable rows remain visible with Someone or unavailable-content copy
  And unavailable rows provide feedback without navigation or raw identifier fallback
```

### AT-007: Newness clears only after successful first-page rendering
Requirement IDs: BR-003, FR-013, FR-014, RULE-005
Acceptance Criteria: AC-011, AC-012, AC-020, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notification_newness_flow_test.dart`, `app/test/router/app_shell_notification_badge_test.dart`

```gherkin
Scenario Outline: First-page state controls acknowledgement
  Given the in-app badge shows new activity
  When the Notifications route reaches <state>
  Then the seen request count is <seenCalls>
  And new-count refreshes only after successful acknowledgement

  Examples:
    | state                       | seenCalls |
    | route entry                 | 0         |
    | first-page loading          | 0         |
    | first-page failure          | 0         |
    | successful rendered content | 1         |
    | successful rendered empty   | 1         |
```

### AT-008: Notification Settings keeps account and device controls separate
Requirement IDs: BR-004, FR-015, FR-016, FR-017, FR-018, FR-024, NFR-004, RULE-004, RULE-008
Acceptance Criteria: AC-013, AC-014, AC-022, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notification_settings_page_test.dart`

```gherkin
Scenario: Denied device permission does not disable account preferences
  Given OS notification permission is denied
  And AppView returns all seven known preferences plus one unknown future category
  When the member opens Notification Settings from Notifications
  Then a scrollable full-screen route shows seven independent scope and push controls
  And no master switch or unknown-category control is shown
  And copy says preferences apply to every account device
  And a current-device warning offers Open settings
  And known controls remain editable and patch only the changed field
```

### AT-009: One owner consumes provider streams exactly once
Requirement IDs: FR-019, FR-020
Acceptance Criteria: AC-015, AC-016
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_service_owner_test.dart`, `app/test/notifications/background_handler_test.dart`

```gherkin
Scenario: Rebuilds and disposal do not duplicate notification work
  Given a fake service has one initial terminated open and live receipt/open streams
  When the ready widget tree rebuilds repeatedly and auth/onboarding state refreshes
  Then the service initializes once
  And the initial open is consumed once
  And every live event is forwarded once
  When the owner is disposed
  Then all subscriptions are cancelled
  And the retained top-level background handler never mutates UI/provider state or logs payload data
```

### AT-010: Every sign-out path cleans local routing safely
Requirement IDs: FR-021
Acceptance Criteria: AC-017, AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/notification_sign_out_cleanup_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart`

```gherkin
Scenario Outline: Sign-out preserves installation identity and applies token policy
  Given the secure store contains bindings for the signed-in DID and a future other DID
  When <signOutPath> completes local cleanup
  Then the signed-in DID binding is removed and the other DID binding is preserved
  And the stable Craftsky device ID is unchanged
  And FCM token deletion attempts equal <deleteCalls>
  And local sign-out succeeds even if token deletion fails

  Examples:
    | signOutPath              | deleteCalls |
    | confirmed AppView logout | 0           |
    | failed AppView logout    | 1           |
    | 401-forced sign-out      | 1           |
```

### AT-011: A pending open cannot survive a required sign-in
Requirement IDs: FR-022
Acceptance Criteria: AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_open_coordinator_test.dart`

```gherkin
Scenario Outline: Readiness either retains or discards one open
  Given a notification open arrives during <state>
  When readiness resolves
  Then the pending open is <result>

  Examples:
    | state                                      | result                                  |
    | transient bootstrap for existing account  | processed once when router is ready     |
    | onboarding for existing account            | processed once after onboarding         |
    | actual signed-out state                     | discarded before destination navigation |
```

### AT-012: Platform presentation uses standard background effects only
Requirement IDs: FR-023, FR-025
Acceptance Criteria: AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/firebase_configuration_test.dart`, `appview/internal/push/firebase_sender_test.go`

```gherkin
Scenario: Platform notification presentation remains bounded
  Given the native configuration and AppView combined FCM message
  When background and foreground behavior is inspected
  Then iOS requests alert and sound but not app-icon badge authorization
  And AppView requests the default APNs sound
  And Android defines one Craftsky notifications channel with standard importance, sound, and vibration
  And the Android manifest default notification channel metadata references that channel ID
  And foreground banners request no sound, vibration, or local notification
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, FR-006, RULE-001 | AC-003 | Evaluate permission action from readiness and OS state. | Signed-out/signed-in; onboarding incomplete/complete; `notDetermined`/authorized/denied; provider failure | Prompt only for signed-in + onboarded + `notDetermined`; no primer/repeat; failures return safe degraded state. | `app/test/notifications/services/notification_permission_policy_test.dart` |
| UT-002 | FR-008, FR-009, NFR-001, RULE-003 | AC-007, AC-008, AC-018 | Parse provider data through an allowlist. | Known and syntactically valid future type; missing/malformed/extra keys; sentinel IDs and destination-like fields | Only validated notification ID, normalized type, and routing ID enter the domain event; an unknown bounded type becomes generic activity; missing/malformed required values reject the payload; extra destination data is ignored and never logged. | `app/test/notifications/models/notification_open_event_test.dart` |
| UT-003 | FR-009, RULE-003, RULE-006 | AC-008 | Match an open against DID-keyed secure routing bindings. | Current DID; matching/missing/stale/other-DID bindings | Only exact current-DID binding match permits resolution; mismatch returns generic unavailable outcome. | `app/test/notifications/services/notification_routing_policy_test.dart` |
| UT-004 | FR-010, RULE-003 | AC-009, AC-025 | Map authorized resolution and failures to navigation intent. | Post/profile/Notifications/retracted/unavailable/404/network/timeout outcomes | Uses only server target; safe outcomes select Notifications; network/timeout adds feedback; no inferred target or retry state. | `app/test/notifications/services/notification_resolution_policy_test.dart` |
| UT-005 | FR-011, FR-012 | AC-010, AC-028, AC-029 | Decode and classify durable notification rows. | Seven known categories; quote metadata; availability combinations; unknown category | Stable ID and metadata retained; unknown becomes generic type; unavailable actor/content becomes safe tombstone; no decode crash. | `app/test/notifications/models/notification_test.dart` |
| UT-006 | BR-003, FR-013 | AC-011 | Format the in-app badge. | Counts 0, 1, 99, 100, larger positive count | 0 hides badge; 1/99 render literally; 100+ renders `99+`; accessible label reflects new activity. | `app/test/notifications/models/notification_badge_test.dart` |
| UT-007 | FR-013 | AC-020 | Classify allowed new-count refresh triggers. | Ready, resume, foreground receipt, page refresh, mark-seen, elapsed timer, unrelated rebuild | Exactly the five approved event classes refresh; elapsed time/rebuild does not poll; no app-icon update action exists. | `app/test/notifications/providers/notification_new_count_provider_test.dart` |
| UT-008 | FR-014, RULE-005 | AC-012, AC-021 | Gate acknowledgement on successful first-page render. | Route/loading/error/content-rendered/empty-rendered/prefetch/background-refresh states | One seen intent only for each successful first-page render; all reads and non-render states remain read-only. | `app/test/notifications/services/notification_seen_policy_test.dart` |
| UT-009 | FR-016 | AC-014, AC-023 | Build a one-field preference PATCH. | Each known category; scope/push field; response containing unknown category | Request contains exactly one known category and one changed field; omitted values remain absent; unknown values are not serialized from UI state. | `app/test/notifications/models/notification_preference_patch_test.dart` |
| UT-010 | FR-016 | AC-014 | Sequence optimistic edits and targeted rollback. | Overlapping edits on same/different category fields; success/failure in both orders | Immediate UI state; failure restores only its still-current optimistic baseline and cannot roll back a newer successful edit. | `app/test/notifications/providers/notification_preferences_provider_test.dart` |
| UT-011 | FR-016 | AC-023 | Preserve unknown server preferences across known edits. | Known seven entries plus future entry | Future entry has no UI model/control and remains untouched by known PATCH requests. | `app/test/notifications/models/notification_preferences_test.dart` |
| UT-012 | FR-019 | AC-015 | Control initial-open and subscription lifecycle. | Repeated start/build/readiness calls; one initial event; live streams; dispose | Start and initial consumption once; one listener per stream; one forwarding per event; all subscriptions cancelled on disposal. | `app/test/notifications/services/notification_service_owner_test.dart` |
| UT-013 | FR-005, FR-006, NFR-003 | AC-005 | Defer, coalesce, and retry token registration. | Token before/after readiness; token A then B; no token; transient failures; later lifecycle trigger | No unauthenticated request; latest available token used; empty token skipped; bounded work retries later without blocking UI or looping. | `app/test/notifications/services/notification_registration_coordinator_test.dart` |
| UT-014 | FR-022 | AC-026 | Retain one open only through transient readiness. | Bootstrap/loading/onboarding/signed-out transitions | Existing-account transient states retain one open; sign-in-required state discards it; no persistent queue is created. | `app/test/notifications/services/pending_notification_open_test.dart` |
| UT-015 | FR-004, RULE-006 | AC-004, AC-008 | Store routing IDs securely by DID. | Two DIDs; replacement ID; remove one DID | Read/write converges per DID; removing one mapping preserves the other; routing IDs never enter shared preferences or plain logs. | `app/test/notifications/data/notification_routing_storage_test.dart` |
| UT-016 | FR-023 | AC-027 | Map permission and foreground-presentation requests. | iOS request; foreground banner configuration | iOS requests alert/sound and `badge: false`; foreground presentation requests no sound, vibration, or local notification. | `app/test/notifications/services/notification_presentation_policy_test.dart` |
| UT-017 | NFR-004 | AC-006, AC-011, AC-013 | Verify localization keys and semantic labels used by notification UI. | Banner, badge, settings, warning, empty/loading/error states | No hard-coded user copy in feature widgets; meaningful semantics and tap labels exist for badge and actions. | `app/test/l10n/notifications_l10n_test.dart` |
| UT-018 | FR-007, RULE-007 | AC-019 | Forward repeated receipts without deduplication. | Two equal notification IDs and provider copies | Two ordered domain receipts and refresh intents; no receipt store or suppression lookup. | `app/test/notifications/services/foreground_notification_service_test.dart` |
| UT-019 | FR-021 | AC-017, AC-024 | Select sign-out cleanup policy. | Confirmed logout, failed logout, 401; token deletion succeeds/fails | Every path removes current binding and preserves device ID; only failed/unconfirmed paths attempt token deletion; provider failure never blocks session clearing. | `app/test/notifications/services/notification_sign_out_cleanup_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-004 | AC-004 | Register a native device through mocked Dio and persist its routing ID. | Authorized fake service, signed-in/onboarded container, mocked `POST /v1/notifications/devices` | Run eligible registration for Android and iOS | CamelCase platform/token request is authenticated by existing Dio; token is not echoed; returned opaque ID is saved under current DID. | `app/test/notifications/data/notification_device_registration_test.dart` |
| IT-002 | FR-003, FR-005, FR-006, NFR-003, RULE-001 | AC-003, AC-005 | Coordinate permission, token refresh, resume retry, and readiness. | ProviderContainer with fake auth/onboarding/service/repository and controllable failures | Transition through readiness, refresh token before/after auth, fail registration, resume | Prompt/registration counts follow policy; latest token wins; app state stays ready; later eligible trigger retries once. | `app/test/notifications/providers/notification_coordinator_test.dart` |
| IT-003 | FR-011 | AC-010, AC-028 | Consume all current notification HTTP shapes. | Mocked list responses with seven categories, quote, availability metadata, unknown category, opaque cursor | Call notification API/repository and paginate | Decoder preserves stable IDs/cursor, produces safe models for all rows, and does not fail on additive category. | `app/test/notifications/data/api_notification_repository_test.dart` |
| IT-004 | BR-002, FR-008, FR-010, FR-022, RULE-003 | AC-007, AC-009, AC-025, AC-026 | Resolve and route normalized opens across readiness states. | Fake service streams, DID binding store, mocked resolution endpoint, test GoRouter/messenger | Emit foreground/background/initial opens with known and future types, resolution outcome matrix, and offline failure | One shared flow normalizes a future type to generic activity, makes one owner-scoped GET, navigates only from server target, safely falls back, and discards opens on required sign-in. | `app/test/notifications/notification_open_flow_test.dart` |
| IT-005 | BR-003, FR-014, RULE-005 | AC-012, AC-021 | Tie mark-seen to actual first-page rendering. | Widget/provider harness with queued list states and mocked seen/count endpoints | Exercise route entry, loading, failure, content success, empty success, and concurrent newer count | No early write; exactly one bodyless seen POST after each successful first-page render; then count refresh preserves any newer server revision. | `app/test/notifications/notification_seen_flow_test.dart` |
| IT-006 | FR-016 | AC-014, AC-023 | Load and patch preferences through mocked Dio. | Seven known plus one unknown server category; queued success/failure PATCH replies | Edit scope/push fields, including overlapping edits | Requests patch one category/field; optimistic state holds on success; targeted rollback/error on failure; unknown value untouched. | `app/test/notifications/data/notification_preferences_api_test.dart`, `app/test/notifications/providers/notification_preferences_provider_test.dart` |
| IT-007 | BR-002, FR-007, FR-012, NFR-004, RULE-002 | AC-006, AC-010, AC-029 | Render foreground banner and safe rows with fake streams. | Localized widget harness, fake service, notification provider invalidation recorder | Emit callbacks on ordinary route and Notifications route; render available/tombstone rows | One silent accessible banner per callback on every route; list/count invalidated; safe row navigation/feedback; no OS-presentation adapter call. | `app/test/notifications/foreground_notification_test.dart`, `app/test/notifications/notifications_page_test.dart` |
| IT-008 | BR-003, FR-013, NFR-004 | AC-011, AC-020 | Render and refresh badge on compact and large app shell. | Form-factor harness; fake new-count repository; trigger recorder | Render counts 0/1/99/100 and fire each approved trigger | Correct bottom-bar/rail badge and semantics; exactly approved refresh calls; no timer-driven or home-screen badge call. | `app/test/router/app_shell_notification_badge_test.dart` |
| IT-009 | BR-004, FR-015, FR-016, FR-017, FR-018, FR-024, NFR-004, RULE-004, RULE-008 | AC-013, AC-014, AC-022, AC-023 | Exercise full-screen Notification Settings route. | Test GoRouter, preference fake, authorized/denied permission fakes, localized app | Open from app bar, scroll all categories, edit controls, tap Open settings, navigate Back | Dedicated typed route; seven category controls; no master/unknown control; account/device copy correct; denial leaves controls enabled; optimistic success/failure works; Back returns. | `app/test/notifications/notification_settings_page_test.dart`, `app/test/router/notification_settings_route_test.dart` |
| IT-010 | FR-002, FR-019, NFR-002 | AC-015 | Run listener ownership under provider fakes only. | ProviderContainer/widget harness with fake notification service and counters | Rebuild, change auth/onboarding, emit initial/live events, dispose | Exactly one initialization/initial consumption/subscription; events forwarded once; subscriptions cancelled; no Firebase method/channel initializes. | `app/test/notifications/providers/notification_service_owner_test.dart` |
| IT-011 | FR-021 | AC-017, AC-024 | Integrate cleanup with explicit logout and global 401 handling. | Fake auth API, service token deletion, routing storage, secure session storage, stable device ID | Run successful/failed logout and intercept 401; force token deletion failure | Binding cleanup and token policy match each path; other DID mapping/device ID survive; local session always clears. | `app/test/auth/providers/auth_controller_notification_cleanup_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart` |
| IT-012 | FR-009, RULE-006 | AC-008 | Enforce routing-store isolation before HTTP resolution. | Two DID bindings, current single-account session, mocked resolution endpoint | Open with matching and non-matching routing IDs | Only exact current-DID match reaches endpoint; every mismatch has zero HTTP/navigation calls and generic feedback. | `app/test/notifications/notification_routing_isolation_test.dart` |
| IT-013 | FR-020 | AC-016 | Invoke the background handler entry point in a constrained harness. | Fake adapter/logger and payload sentinels; no widget/provider container | Call the retained top-level handler | Future completes without navigation/provider mutation and no sentinel is recorded. | `app/test/notifications/background_handler_test.dart` |
| IT-014 | FR-025 | AC-027 | Build the AppView combined FCM message. | Existing capturing Firebase client and send request | Send one message | APNs payload requests `default` sound while data, copy, TTL, category, routing, and eligibility inputs remain unchanged. | `appview/internal/push/firebase_sender_test.go` |
| IT-015 | FR-026 | AC-030 | Validate the AppView non-production push gate. | Dev/default config and explicitly enabled config with/without project credentials | Load configuration | Normal non-production path is disabled; enabled path requires explicit flag and valid project configuration. | `appview/internal/app/push_config_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Firebase packages remain isolated from application presentation, routing, repositories, and tests. | FR-002, NFR-002 | AC-002, AC-015 | Add a file-scan test allowing Firebase imports only in the adapter/bootstrap/background-handler boundary; fake-driven tests must have no Firebase import or initialization. |
| REG-002 | Logs, Sentry, analytics, and diagnostics remain redacted. | NFR-001, NFR-002 | AC-018 | Extend `app/test/observability/secret_scan_test.dart` and Sentry redaction tests with sentinel token, routing ID, notification ID, DID, handle, AT-URI, payload, and credential values across success/error paths. |
| REG-003 | Existing auth, onboarding, typed router, notification pagination, and row navigation remain functional. | FR-006, FR-011, FR-012, FR-019, FR-021 | AC-005, AC-010, AC-015, AC-017 | Run and extend current `test/auth`, `test/onboarding`, `test/router`, and `test/notifications` suites; preserve load-more concurrency/retry and existing destinations while adding new flows. |
| REG-004 | The client has no periodic count polling, home-screen icon badge updates, notification receipt store/dedupe, or persistent deep-link queue. | FR-007, FR-010, FR-013, RULE-007 | AC-019, AC-020, AC-025 | Structural/file-scan assertions plus fake-clock tests prove no timer-driven count calls and no new persistence writes for receipts, icon badges, or opens. |
| REG-005 | The Firebase background handler remains platform-callable and side-effect bounded. | FR-020 | AC-016 | File-scan/compile test asserts top-level non-anonymous declaration, `@pragma('vm:entry-point')`, no UI/router/provider imports, and no payload logging. |
| REG-006 | Native identity, project, capability, permission, and single-channel configuration do not drift. | BR-001, FR-001, FR-023 | AC-001, AC-027 | Inspect Android/iOS project and Firebase configuration: shared `social.craftsky.app`, project `craftsky-app`, iOS push/remote-notification capabilities, alert/sound without badge request, exactly one named Android channel, and `com.google.firebase.messaging.default_notification_channel_id` referencing that channel's ID. |
| REG-007 | The AppView message changes only the default APNs sound. | FR-025 | AC-027 | Extend the existing capturing-sender test to compare the complete data/notification/Android/APNs contract and assert no field other than APNs sound changed. |
| REG-008 | Notification list/count GETs and prefetch remain read-only. | FR-014, RULE-005 | AC-012, AC-021 | Mocked-Dio call ledger proves list/count/prefetch/foreground invalidation never call seen; seen appears only after the render gate. |
| REG-009 | One explicit keep-alive owner remains the only notification coordinator. | FR-019 | AC-015 | Import/ownership scan and provider lifecycle tests fail if a second root listener or widget-owned Firebase subscription is introduced. |
| REG-010 | Existing AppView notification JSON routes and eligibility behavior remain unchanged by the sound follow-up. | FR-025, FR-026 | AC-027, AC-030 | Run focused AppView API/push suites and assert the sender/config-only change introduces no route or response-shape diff. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Auth/onboarding/permission matrix | Signed out; signed in as Alice; onboarding incomplete/complete; undetermined/authorized/denied permission; throwing service | AT-002, UT-001, IT-002 |
| TD-002 | Provider event normalization and privacy | Valid foreground/open events with known and syntactically valid future types; missing/malformed fields; same ID twice; extra DID/handle/AT-URI/destination/text/image fields | AT-003–AT-005, UT-002, UT-018, IT-004, IT-012, REG-002 |
| TD-003 | Secure routing and account transitions | Alice and Bob DIDs; distinct opaque routing IDs; stale prior-account ID; replacement binding; stable device ID | AT-002, AT-005, AT-010, UT-003, UT-015, UT-019, IT-001, IT-011, IT-012 |
| TD-004 | Durable feed matrix | Stable notification IDs for like/follow/reply/mention/quote/repost/everythingElse; unknown category; available/unavailable actors and references; safe hydrated targets | AT-006, UT-005, IT-003, IT-007 |
| TD-005 | Resolution outcomes | Authorized post/profile/Notifications targets; retracted/unavailable; non-enumerating 404; network failure; timeout | AT-004, UT-004, IT-004 |
| TD-006 | Newness lifecycle | Counts 0/1/99/100; first-page loading/failure/content/empty; newer server revision during acknowledgement; ready/resume/foreground/page-refresh/seen triggers | AT-007, UT-006–UT-008, IT-005, IT-008, REG-008 |
| TD-007 | Preference and race matrix | Seven known categories, future unknown category, authorized/denied OS state, overlapping same-field and different-field success/failure responses | AT-008, UT-009–UT-011, IT-006, IT-009 |
| TD-008 | Listener and pending-open lifecycle | One initial terminated open; repeated rebuild/readiness changes; live events; disposal; transient bootstrap/onboarding; actual signed-out state | AT-009, AT-011, UT-012, UT-014, IT-010, IT-013 |
| TD-009 | Native/provider delivery | One registered Android device and one registered iOS device; foreground/background/terminated messages; one created Android channel plus manifest default-channel binding; permission grant/denial; rotated token; explicit non-production sender window | AT-001, AT-012, REG-006, MAN-001–MAN-005 |
| TD-010 | Sensitive sentinels | Unique fake FCM token, routing ID, notification ID, DID, handle, AT-URI, raw payload text, Firebase credential-shaped value | UT-002, IT-013, REG-002 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, BR-002, FR-001, FR-007, FR-008, FR-023, RULE-002 | AC-001, AC-006, AC-007, AC-027 | Android physical-device delivery and open matrix | With normal non-production sending disabled, explicitly enable one bounded test session. Install a build registered as `social.craftsky.app`, grant permission, deliver foreground/background/terminated messages without an explicit Android `channel_id`, tap banner/OS notifications, inspect the selected channel, then disable sending. | Foreground shows one silent/non-vibrating Craftsky banner and refreshes data; background/terminated uses the manifest-bound single “Craftsky notifications” channel with standard sound/vibration rather than an FCM fallback channel; every tap resolves through AppView to the authorized target/fallback. |
| MAN-002 | BR-001, BR-002, FR-001, FR-007, FR-008, FR-023, FR-025, RULE-002 | AC-001, AC-006, AC-007, AC-027 | iOS physical-device delivery and open matrix | Upload/configure the APNs authentication key in Firebase, explicitly enable one bounded non-production session, install the `social.craftsky.app` build, grant alert/sound permission, deliver foreground/background/terminated messages, inspect app-icon behavior, then disable sending. | Foreground banner is silent and does not update app-icon badge; background/terminated alerts use default sound; taps resolve safely; no badge authorization or app-icon count is used. |
| MAN-003 | FR-005, FR-021 | AC-005, AC-017, AC-024 | Token rotation and sign-out cleanup | On a physical device, observe registration, rotate/reinstall as appropriate to obtain a refreshed token, then exercise successful logout and a controlled failed/unconfirmed logout or 401 path. | Latest token re-registers; every sign-out clears the local routing binding and retains device ID; success retains provider token; unconfirmed cleanup attempts token deletion; later sign-in obtains/registers a usable token. |
| MAN-004 | FR-018, FR-024, RULE-004, RULE-008 | AC-003, AC-013 | Denied permission and native settings recovery | Deny the first direct OS prompt, open Notification Settings, edit category controls, use Open settings, authorize notifications in the OS, and resume Craftsky. | No repeated prompt/primer; warning identifies this device; account-wide controls remain editable; native settings opens; resume refreshes permission and registration becomes eligible. |
| MAN-005 | FR-026 | AC-030 | Bounded non-production sender gate | Verify AppView starts with `PUSH_ENABLED=false`; record the intended device/account and time window; temporarily enable with valid project credentials; send only the planned checks; restore disabled configuration and restart/verify. | No real alert is possible before explicit enablement; delivery occurs only during the bounded session; the final verified state is disabled. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | FCM/APNs delivery, native OS presentation, and token rotation are not deterministic in the normal automated suite. | BR-001, BR-002, FR-001, FR-005, FR-007, FR-008, FR-023, FR-025 | They require provider credentials, native OS state, and physical devices. | Keep provider-neutral behavior and configuration contracts automated; require MAN-001 through MAN-003 before launch enablement. |
| GAP-002 | The APNs authentication key is not yet configured for the Firebase project. | FR-023, FR-025 | This is an external rollout prerequisite, not source-controlled test data. | Upload/configure the key before MAN-002; never commit or expose the credential. |
| GAP-003 | Exact in-app banner component is intentionally deferred to coding design. | FR-007, NFR-004 | Requirements fix behavior, copy source, silence, and accessibility but not the concrete existing messenger primitive. | Coding plan must select an existing Craftsky messenger/theming seam without weakening AT-003 or IT-007. |
| GAP-004 | Static native configuration checks cannot prove platform delivery. | FR-001, FR-023 | Correct plist/manifest/project shape may still fail due to signing, APNs, or device settings. | Pair REG-006 with MAN-001 and MAN-002. |
| GAP-005 | Best-effort token deletion cannot recall a push already accepted by FCM. | FR-021 | Provider acceptance and local/server cleanup are not atomic. | Assert cleanup ordering and non-blocking behavior; document and manually observe the bounded residual risk. |
| GAP-006 | Long uninterrupted foreground sessions can retain a stale badge when no approved trigger occurs. | FR-013 | Periodic polling was explicitly rejected. | Assert absence of polling; accept refresh at the next readiness/resume/foreground-message/page/seen trigger unless requirements change. |
| GAP-007 | Exact file locations for new service/coordinator classes may change during coding design. | FR-002, FR-019 | This stage specifies behavior and practical suite ownership, not the final implementation layout. | Preserve test IDs and behavioral seams; adjust proposed automation paths in `04-coding-plan.md` if architecture review selects different names. |

No Must requirement lacks a verification path. The remaining gaps are native/provider or explicitly deferred implementation-layout concerns, not blockers for document review.

## 10. Out Of Scope

- Web, macOS, Windows, Linux, raw APNs, and any platform beyond Android/iOS.
- Multi-account sign-in UI, simultaneous sessions, or cross-account selection; tests cover only DID-keyed storage/routing shape under one active session.
- A local persistent notification inbox, offline content cache, per-item read state, per-device unread state, app-icon badge, polling, receipt deduplication, or deferred deep-link queue.
- Server notification eligibility, persistence, outbox, retry, retention, routing, and database behavior already covered by the completed AppView slice.
- New AppView routes, JSON contract changes, payload copy policy, digests/grouping, rich media, actions, custom sounds, local scheduling, or category-specific Android channels.
- Production credential provisioning and launch enablement beyond the explicit manual gate; credentials must never enter tests or the repository.
- Exact visual styling beyond localized, accessible, full-screen, scrollable behavior and existing Craftsky theming conventions.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-14-flutter-push-notifications/`
- Recommended first failing test for implementation: `UT-002`, the provider-data allowlist parser, because it establishes the privacy-safe domain event boundary, including generic normalization of an unknown bounded `type`, used by foreground, background-open, terminated-open, and routing tests without Firebase initialization.
- Suggested test order for implementation: `UT-002` → `UT-003`/`UT-015` → `UT-001`/`UT-013` → `UT-012`/`IT-010` → `IT-001`/`IT-002` → `AT-003`/`IT-007` → `AT-004`/`IT-004` → `UT-005`/`IT-003`/`AT-006` → `UT-006`–`UT-008`/`IT-005`/`IT-008` → `UT-009`–`UT-011`/`IT-006`/`IT-009` → `UT-019`/`IT-011` → static/native regressions → `IT-014`/`IT-015` → manual checks.
- Commands discovered:
  - From `app/`: `flutter test test/notifications`
  - From `app/`: `flutter test test/auth test/onboarding test/router test/notifications test/observability test/shared/api/providers`
  - From the repository root: `just app-test test/notifications`
  - From the repository root: `just app-test`
  - From `app/`: `dart analyze`
  - From the repository root: `just app-analyze`
  - From `app/`: `dart run build_runner build`
  - From `appview/`: `go test ./internal/push ./internal/app -count=1`
  - Canonical AppView verification after the narrow sound/config change: repository-root `just test`
- Blocking gaps: None for document review. Because risk remains High, implementation may not begin until workflow document review is complete and the user explicitly approves implementation.
