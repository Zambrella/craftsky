# Acceptance Test Specification: Multi-Account Sessions And Notification Routing

## 1. Test Strategy

This is a high-risk Flutter authentication, persistence, routing, and notification change. The primary risk is not presentation: it is allowing a stale request, response, mutation, route, credential, notification, or cleanup operation from one DID to affect another DID. The automated design therefore starts with a versioned session-registry model and account-bound request/invalidation tests, then adds controller/provider tests for activation, validation, sign-out, push registration, and notification routing, and finally adds widget/router coverage for the compact and large switchers.

Existing Flutter test conventions support `package:flutter_test`, `ProviderContainer.test`, secure-storage mock values, fake API clients, Dio interceptor tests, and widget/router harnesses. No AppView route or database change is expected, but the existing independent-session and per-account notification-subscription contracts are load-bearing. `IT-013` and normal AppView verification therefore protect shared-installation logout isolation even if production server code is unchanged.

Most behavior is automated. Manual checks are limited to physical-device push delivery from background/terminated states, platform secure-storage inspection, and final compact/large visual and accessibility confirmation. Risk remains **High**. Document review and explicit approval are required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003 | AT-001, AT-002, UT-001, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-010, AC-011, AC-012, AC-020, AC-024, AC-028 | AT-005, AT-012, UT-006, UT-007, UT-008, IT-005, REG-006, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Partial |
| BR-003 | AC-015, AC-016, AC-029 | AT-008, AT-011, UT-012, UT-013, IT-007, IT-008, IT-010, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001, AC-015, AC-018 | AT-001, AT-008, UT-001, UT-003, IT-001, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003, AC-004 | AT-002, UT-002, UT-020, UT-022, IT-002, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-001 | AC-004 | AT-002, UT-002, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-005, AC-006 | AT-003, UT-016, IT-011, REG-002, MAN-001 | Acceptance / Unit / Integration / Regression / Manual | Partial |
| FR-005 | AC-005, AC-007, AC-018 | AT-003, UT-003, UT-016, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-006 | AT-003, IT-011, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-007 | AC-008, AC-018 | AT-004, UT-003, UT-004, IT-003 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-008, AC-009 | AT-004, UT-004, UT-005, IT-003, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-008 | AC-009, AC-016 | AT-004, AT-008, UT-005, IT-003, IT-008, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-008, AC-009 | AT-004, UT-004, IT-003, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-010 | AC-010, AC-014 | AT-005, AT-007, UT-011, IT-004, REG-007, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Partial |
| FR-011 | AC-010, AC-014 | AT-007, UT-011, IT-004, REG-007, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Partial |
| RULE-002 | AC-010, AC-013 | AT-005, AT-006, UT-009, UT-011, IT-004, IT-006, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-012 | AC-011, AC-012, AC-020 | AT-005, UT-006, UT-008, IT-005, REG-006, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Partial |
| FR-013 | AC-011 | AT-005, UT-006, UT-007, IT-005, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| FR-014 | AC-012, AC-028 | AT-005, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-011, AC-012 | AT-005, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-015, AC-029 | AT-008, AT-011, UT-012, UT-013, IT-007, IT-010, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-016 | AT-008, UT-005, UT-010, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-015, AC-016, AC-018 | AT-008, UT-003, UT-012, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-022 | AT-004, UT-021, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-023, AC-031 | AT-006, UT-009, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-024 | AT-012, UT-017, IT-005, MAN-001 | Acceptance / Unit / Integration / Manual | Partial |
| FR-022 | AC-025 | AT-003, UT-018, IT-011, MAN-001 | Acceptance / Unit / Integration / Manual | Partial |
| FR-023 | AC-026 | AT-009, UT-015, IT-012, REG-008 | Acceptance / Unit / Integration / Regression | Yes |
| FR-024 | AC-027 | AT-010, UT-010, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-025 | AC-028 | AT-005, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-026 | AC-029 | AT-011, UT-013, IT-010, IT-013, MAN-002 | Acceptance / Unit / Integration / Manual | Partial |
| RULE-003 | AC-017 | UT-016, REG-009 | Unit / Regression | Yes |
| RULE-004 | AC-021 | AT-002, UT-002, UT-016, IT-011 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-030 | AT-003, UT-016, IT-011 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-019, AC-029 | UT-001, UT-013, UT-014, IT-001, IT-010, REG-010, MAN-003 | Unit / Integration / Regression / Manual | Partial |
| NFR-003 | AC-008 | AT-004, UT-004, IT-003, MAN-001 | Acceptance / Unit / Integration / Manual | Partial |
| NFR-004 | AC-006 | AT-003, REG-002 | Acceptance / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Restore Multiple Retained Accounts
Requirement IDs: BR-001, FR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/account_session_registry_provider_test.dart`, `app/test/auth/providers/auth_session_provider_test.dart`

```gherkin
Feature: Durable multi-account sessions
  Scenario: Restart with two retained accounts
    Given accounts A and B have valid independently stored CraftSky sessions
    And B was the active account when the app stopped
    When the app starts again
    Then both accounts remain retained
    And B is restored immediately as active without waiting for the network
    And A remains available for later activation
```

### AT-002: Add Or Refresh An Account Without Replacing Others
Requirement IDs: BR-001, FR-003, RULE-001, RULE-004
Acceptance Criteria: AC-003, AC-004, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/auth_controller_test.dart`, `app/test/router/router_redirect_test.dart`

```gherkin
Feature: Add account
  Scenario Outline: OAuth completion updates the retained registry
    Given account A is retained
    When OAuth completes for <account>
    Then <registry outcome>
    And <route outcome>

    Examples:
      | account | registry outcome | route outcome |
      | new onboarded B | A and B remain, with B active | B lands on Home |
      | new incomplete B | A and B remain, with B active | B continues onboarding |
      | retained A | A's token and cached identity are replaced without a duplicate | A follows its current onboarding state |
      | sixth distinct DID | the five existing accounts remain unchanged | the limit error is shown |
```

### AT-003: Open And Use The Responsive Account Switcher
Requirement IDs: FR-004, FR-005, FR-006, FR-022, RULE-005, NFR-004
Acceptance Criteria: AC-005, AC-006, AC-007, AC-025, AC-030
Priority: Must
Level: Acceptance
Automation Target: `app/test/router/app_shell_account_switcher_test.dart`

```gherkin
Feature: Account switcher entry point
  Scenario: Profile destination supports tap and account-switch actions
    Given multiple accounts are retained
    When the member presses and holds Profile on a compact layout
    Then a modal bottom sheet opens without navigating to Profile
    And the active account is first
    And inactive accounts follow in most-recent-use order
    And each account remains identifiable with cached-handle fallbacks
    And inactive rows have no remove or sign-out action
    When the member normally taps Profile
    Then ordinary Profile navigation occurs and no switcher opens
    When the same account-switch action is invoked on a large layout
    Then an anchored menu with equivalent actions and semantics opens beside Profile
```

### AT-004: Switch Accounts Across A Hard State Boundary
Requirement IDs: FR-007, FR-008, FR-009, FR-019, NFR-001, NFR-003
Acceptance Criteria: AC-008, AC-009, AC-022
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/account_activation_coordinator_test.dart`, `app/test/router/account_switch_routing_test.dart`

```gherkin
Feature: Account activation
  Scenario: Switch while old-account work is in flight
    Given A is active on a non-Home route with viewer-specific state
    And a request and optimistic mutation for A are in flight
    And B is retained
    When the member selects B while online or offline
    Then the UI enters a non-interactive transition state
    And B becomes active from secure local state
    And navigation resets to B's Home root
    And subsequent active-context requests use only B's token
    And late A results cannot populate or mutate B's state
    And an initial offline content failure does not reactivate A
```

### AT-005: Route Every Notification Through Its Recipient Account
Requirement IDs: BR-002, FR-012, FR-013, FR-014, FR-015, FR-025
Acceptance Criteria: AC-011, AC-012, AC-020, AC-028
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/notification_open_coordinator_test.dart`, `app/test/notifications/notification_open_flow_test.dart`

```gherkin
Feature: Account-safe notification opens
  Scenario Outline: Handle a notification open
    Given A is active
    When an open has <routing state>
    Then <account outcome>
    And <navigation outcome>

    Examples:
      | routing state | account outcome | navigation outcome |
      | exact unique binding to retained B | B activates before destination handling | the typed destination reads with B's session |
      | missing, malformed, or ambiguous binding | A remains active | generic unavailable feedback appears and no destination opens |
      | well-formed binding for removed B | A remains active | signed-out-account feedback appears and no destination or sign-in opens |
      | in-app row returned under A before a switch to B | B remains active | the stale A row cannot open under B |
```

### AT-006: Show Isolated Inactive-Account Notification Counts
Requirement IDs: FR-020, RULE-002
Acceptance Criteria: AC-013, AC-023, AC-031
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/providers/account_notification_new_count_provider_test.dart`, `app/test/router/app_shell_account_switcher_test.dart`

```gherkin
Feature: Account-scoped notification activity
  Scenario: Open the switcher while inactive counts refresh
    Given A is active and inactive B and C are retained
    And cached counts are 3 for B and 120 for C
    When the switcher opens
    Then it is immediately usable
    And B shows 3 and C shows 99+
    And zero and unknown counts show no badge
    When C's refresh fails and B's succeeds
    Then B updates independently and account selection remains usable
    When a duplicate foreground push for B is delivered
    Then B's authoritative count is fetched instead of incrementing a local count
```

### AT-007: Register Every Eligible Retained Account After Token Rotation
Requirement IDs: FR-010, FR-011
Acceptance Criteria: AC-010, AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/services/notification_registration_coordinator_test.dart`

```gherkin
Feature: Multi-account push registration
  Scenario: Provider token changes while several accounts are retained
    Given notification permission is authorized
    And eligible accounts A and B are retained with independent sessions
    When the provider token is created or refreshed
    Then registration is attempted for A and B using each account's session
    And successful DID-keyed routing bindings are retained independently
    And a failure for one account stays retryable without blocking the other
    And neither account must be manually activated
```

### AT-008: Remove Or Invalidate Only One Account
Requirement IDs: BR-003, FR-016, FR-017, FR-018
Acceptance Criteria: AC-015, AC-016, AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/auth_controller_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart`

```gherkin
Feature: Account-scoped session removal
  Scenario Outline: An account becomes unusable
    Given A and B are retained
    And B is the most recently used valid fallback
    When A is removed by <cause>
    Then only A's usable token, binding, subscription, and account-scoped state are removed
    And B remains signed in, becomes active if needed, and lands on Home
    And the app becomes fully signed out only if no valid sessions remain

    Examples:
      | cause |
      | confirmed sign-out |
      | a 401 tied to A's request token |
      | authoritative validation failure for A |
```

### AT-009: Guard Account Switching When Work Is Unsaved
Requirement IDs: FR-023
Acceptance Criteria: AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/router/account_switch_unsaved_work_test.dart`, `app/test/feed/widgets/post_composer_sheet_discard_test.dart`

```gherkin
Feature: Guarded account activation
  Scenario Outline: Switching would abandon unsaved work
    Given A has an in-progress compose flow or unsaved edit
    When <source> requests activation of B
    Then the existing unsaved-changes confirmation appears before activation
    When the member cancels
    Then A remains active
    And A's DID-scoped draft remains available where applicable
    And the switch or notification-open attempt is destroyed and cannot resume later

    Examples:
      | source |
      | manual account selection |
      | notification tap |
```

### AT-010: Restore First And Validate Accounts Safely
Requirement IDs: FR-024
Acceptance Criteria: AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/auth_session_provider_test.dart`

```gherkin
Feature: Multi-account startup validation
  Scenario: Start with active A and inactive B and C
    Given A, B, and C are cached and A was active
    When the app starts
    Then A is restored before network validation completes
    And authenticated whoami validates A first
    And B and C validate later with bounded concurrency
    And network or server availability failures retain their sessions
    And an unauthorized or identity-mismatch result removes only the result's original DID and token generation
```

### AT-011: Recover Safely From Unconfirmed Offline Sign-Out
Requirement IDs: BR-003, FR-016, FR-026, NFR-002
Acceptance Criteria: AC-029
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/services/notification_sign_out_recovery_test.dart`

```gherkin
Feature: Offline sign-out recovery
  Scenario: A signs out while AppView is unreachable and B remains
    Given A and B are retained on one installation
    When A signs out and AppView logout cannot be confirmed
    Then A is removed from the usable registry immediately
    And A's credential is quarantined for cleanup only and cannot authorize normal requests
    And stale A notification opens are rejected
    And B remains usable while the shared provider token enters recovery
    When connectivity returns
    Then A is logged out or authoritatively found already unauthorized
    And A's subscription is deactivated before its cleanup credential is deleted
    And only then are B and other eligible accounts registered to the replacement provider token
```

### AT-012: Identify An Inactive Recipient In A Foreground Banner
Requirement IDs: BR-002, FR-021
Acceptance Criteria: AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/notifications/notification_effect_host_test.dart`, `app/test/notifications/notification_open_flow_test.dart`

```gherkin
Feature: Foreground notification identity
  Scenario: A foreground notification belongs to inactive B
    Given A is active and B is retained
    When a foreground notification maps uniquely to B
    Then the banner keeps its existing title and body
    And it shows B's avatar and For @handle
    And cached handle and generic avatar fallbacks remain identifiable
    When the banner is tapped
    Then B activates before the typed destination opens
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-001, FR-001, NFR-002 | AC-001, AC-018, AC-019 | Decode, validate, persist, and recover a versioned DID-keyed registry without exposing secrets. | Empty, valid multi-entry, corrupt active pointer, corrupt one-entry, interrupted/partial, unknown-version, and all-invalid blobs. | Independently valid entries survive; active DID falls back by MRU; all-invalid data becomes signed out; tokens never appear in `toString` or diagnostics. | `app/test/auth/models/session_registry_test.dart`, `app/test/auth/providers/secure_token_storage_test.dart` |
| UT-002 | FR-003, RULE-001, RULE-004 | AC-003, AC-004, AC-021 | Enforce additive upsert, duplicate-DID replacement, and five-distinct-DID limit in the domain layer. | Registry sizes 0, 4, and 5; new DID; existing DID with replacement token/identity. | New DID is added below limit; existing DID is replaced even at limit; sixth distinct DID is rejected atomically without disturbing entries. | `app/test/auth/models/session_registry_test.dart` |
| UT-003 | FR-001, FR-005, FR-007, FR-018 | AC-018 | Compute active-first/MRU ordering and deterministic fallback. | Tied and distinct activation timestamps; removed active DID; invalid entries. | Active account is first; inactive accounts have stable MRU order; removal selects the most recently activated valid account. | `app/test/auth/models/session_registry_test.dart` |
| UT-004 | FR-007, FR-009, NFR-001, NFR-003 | AC-008, AC-009 | Reject stale provider, response, and mutation completions across an activation generation. | A operation started before switch; B activation; late success/error/optimistic rollback from A. | B reaches a visible non-interactive transition and late A completions cannot read or write B-scoped state. | `app/test/auth/providers/account_activation_coordinator_test.dart` |
| UT-005 | FR-008, FR-017, NFR-001 | AC-009, AC-016 | Capture request DID/token context and scope `401` invalidation to it. | Concurrent A and B requests; active switch; A `401`; repeated overlapping `401`s. | Each request carries its selected token; only A is invalidated; cleanup coalesces per account/generation without clearing B. | `app/test/shared/api/providers/session_auth_interceptor_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart` |
| UT-006 | BR-002, FR-012, FR-013, FR-014, FR-025 | AC-011, AC-012, AC-028 | Reverse-resolve a routing ID to exactly one retained DID and classify invalid versus removed-account outcomes. | Exact binding, duplicate/ambiguous binding, malformed/missing ID, well-formed unbound ID, binding for non-retained DID. | Exact retained match selects its DID; malformed/ambiguous uses generic fallback; well-formed removed-account input uses signed-out-account feedback. | `app/test/notifications/data/notification_routing_storage_test.dart`, `app/test/notifications/providers/notification_open_coordinator_test.dart` |
| UT-007 | FR-013, FR-015 | AC-011, AC-012 | Preserve latest-pending-open semantics across account readiness and account generation. | Two opens during startup/switch; target removal; target reauthentication; stale first completion. | Only latest eligible open continues; no open crosses removal or reauthentication; activation precedes destination inference/navigation. | `app/test/notifications/services/pending_notification_open_test.dart`, `app/test/notifications/providers/notification_open_coordinator_test.dart` |
| UT-008 | BR-002, FR-012 | AC-020 | Bind an in-app notification row to the DID/generation that produced it. | A row, switch to B, tap stale row; unchanged A context, tap current row. | Stale row is rejected under B; current row opens only under its producing account. | `app/test/notifications/notification_open_flow_test.dart` |
| UT-009 | FR-020, RULE-002 | AC-013, AC-023, AC-031 | Fetch, cache, format, and refresh new counts per DID. | Counts 0, 3, 99, 100, 120; one failure; duplicate foreground delivery; unauthorized inactive session. | Zero/unknown is hidden; 100+ renders `99+`; failures stay isolated; foreground delivery fetches authoritative count and never increments locally. | `app/test/notifications/providers/account_notification_new_count_provider_test.dart` |
| UT-010 | FR-017, FR-024 | AC-016, AC-027 | Schedule active-first startup validation with bounded inactive concurrency and result scoping. | Active A; inactive B/C/D; network error; `401`; identity mismatch; session replaced while validation is in flight. | A validates first; inactive concurrency stays within bound; transient errors retain accounts; authoritative results affect only the original unchanged session generation. | `app/test/auth/providers/auth_session_provider_test.dart` |
| UT-011 | FR-010, FR-011, RULE-002 | AC-010, AC-014 | Register and retry all retained eligible accounts using account-bound sessions. | Authorized/denied permission; active/inactive sessions; token refresh; per-account transient failure; account removed mid-run. | Authorized runs register each eligible DID independently; denied permission registers none; failures remain per-DID retryable; removed account cannot save a late binding. | `app/test/notifications/services/notification_registration_coordinator_test.dart` |
| UT-012 | FR-016, FR-018 | AC-015, AC-018 | Apply confirmed account-scoped sign-out cleanup and fallback activation. | Active A, remaining B/C with MRU order; confirmed logout success; last-account removal. | Only A's registry entry, binding, subscription, and local state are removed; fallback lands Home; last removal becomes signed out. | `app/test/auth/providers/auth_controller_test.dart`, `app/test/notifications/services/notification_sign_out_cleanup_test.dart` |
| UT-013 | FR-016, FR-026, NFR-002 | AC-029 | Enforce the offline cleanup state machine and ordering. | Logout network failure, unauthorized cleanup result, retryable cleanup failure, token rotation, remaining registrations. | Removed token is cleanup-only; removed subscription cleanup completes before credential deletion and replacement-token registration; retry persists safely. | `app/test/notifications/services/notification_sign_out_recovery_test.dart` |
| UT-014 | NFR-002 | AC-019 | Permit the opaque routing ID only in the authenticated registration response and provider payload; redact credentials and routing IDs from every output surface; and confine cached account identity to designated UI. | Registration response and provider event containing a sentinel routing ID; models/errors/events containing sentinel credentials and identity values; switcher/banner/Profile identity fixtures; success and failure paths. | The provider event can parse the opaque ID for routing; credentials and routing IDs appear in no log, analytics, crash, URL, UI, or string output; DIDs, handles, raw payloads, and retained-account lists appear in no diagnostics; intended cached handle/avatar identity appears only in the switcher, Profile destination, and inactive-recipient banner; the local DID-to-ID map remains secure-storage-only. | `app/test/observability/secret_scan_test.dart`, `app/test/notifications/models/notification_open_event_test.dart`, `app/test/auth/models/session_registry_test.dart`, `app/test/router/app_shell_account_switcher_test.dart` |
| UT-015 | FR-023 | AC-026 | Coordinate unsaved-work confirmation for manual and notification activation. | Clean/dirty compose state; confirm/cancel; repeated and overlapping requests. | Clean work switches directly; confirmation precedes dirty switch; cancel preserves A and its draft and permanently discards that attempt. | `app/test/auth/providers/account_activation_coordinator_test.dart` |
| UT-016 | FR-004, FR-005, RULE-003, RULE-004, RULE-005 | AC-005, AC-007, AC-017, AC-021, AC-030 | Build the switcher view model and available actions. | 1–5 accounts, missing profile metadata, active/MRU states, inactive rows. | Active-first ordering, cached-handle fallback, Add visibility, disabled five-account helper, no bulk sign-out, and no inactive removal action are produced. | `app/test/auth/models/account_switcher_state_test.dart` |
| UT-017 | FR-021 | AC-024 | Build inactive-recipient foreground banner identity. | Full cached identity, missing avatar, missing display name. | Existing title/body stay unchanged; recipient line uses `For @handle`; generic avatar and cached handle are used as fallbacks. | `app/test/notifications/models/foreground_notification_event_test.dart`, `app/test/notifications/notification_effect_host_test.dart` |
| UT-018 | FR-022 | AC-025 | Render active-account Profile destination identity. | Active identity with avatar, failed avatar, no avatar; selected/unselected states. | Avatar appears when available; generic person fallback appears otherwise; selected state remains visually and semantically clear. | `app/test/router/app_shell_account_switcher_test.dart` |
| UT-019 | RULE-003 | AC-017 | Verify account actions omit installation-wide logout. | Switcher and settings action models. | No sign-out-all label, command, or semantics action exists. | `app/test/settings/sign_out_tile_test.dart`, `app/test/router/app_shell_account_switcher_test.dart` |
| UT-020 | FR-003 | AC-003 | Route additive OAuth completion by the new active account's onboarding state. | Completed OAuth response for onboarded and incomplete B while A remains retained. | B becomes active; incomplete B goes to onboarding and onboarded B goes Home; A remains stored. | `app/test/auth/providers/auth_controller_test.dart`, `app/test/router/router_redirect_test.dart` |
| UT-021 | FR-019 | AC-022 | Complete manual activation locally while offline and reset navigation. | A on nested route; retained B; network unavailable; B content request fails. | B stays active, router lands at Home, retry UI may appear, and neither A's route nor B's prior route is restored. | `app/test/router/account_switch_routing_test.dart` |
| UT-022 | FR-003 | AC-003 | Preserve the current registry and account context when Add account is canceled or fails, covering `EC-004`. | A retained and active; OAuth cancellation, browser-launch failure, handoff failure, rejected completion, and partial B result. | A's registry entry, token, active DID, MRU order, route, and account-scoped state remain unchanged; no partial B entry is retained. | `app/test/auth/providers/auth_controller_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, NFR-002 | AC-001, AC-019 | Secure-storage registry survives restart and partial corruption. | Mock platform secure storage with two valid entries plus valid/corrupt active-pointer and entry variants. | Build, dispose, and rebuild auth providers. | Recoverable sessions persist, deterministic active fallback is selected, and no token appears in emitted diagnostics. | `app/test/auth/providers/secure_token_storage_test.dart`, `app/test/auth/providers/auth_session_provider_test.dart` |
| IT-002 | BR-001, FR-003, RULE-001 | AC-003, AC-004 | OAuth completion is additive and route-aware. | Retain A; fake handoff/whoami for new or repeated DID B/A; fake onboarding state. | Complete auth deep link. | Registry upserts returned DID, activates it, preserves other DIDs, and routes to onboarding or Home. | `app/test/auth/providers/auth_controller_test.dart`, `app/test/router/router_redirect_test.dart` |
| IT-003 | FR-007, FR-008, FR-009, FR-019, NFR-001, NFR-003 | AC-008, AC-009, AC-022 | Full activation boundary resets route/providers and rejects stale async work. | Render A on a nested route with account-scoped providers and controlled in-flight calls; retain B. | Select B and release A completions after transition. | B is visibly active at Home; B requests use B token; A data, mutations, errors, and route do not appear under B, online or offline. | `app/test/router/account_switch_routing_test.dart`, `app/test/shared/api/providers/session_auth_interceptor_test.dart` |
| IT-004 | FR-010, FR-011, RULE-002 | AC-010, AC-014 | Registration lifecycle covers every eligible account. | Retain active A and inactive B with different sessions; authorize permission; fake device-registration API and routing storage. | Start readiness, refresh provider token, and retry a single-account failure. | Both account subscriptions and bindings settle independently against the latest token without manual activation. | `app/test/notifications/services/notification_registration_coordinator_test.dart`, `app/test/notifications/data/notification_device_registration_test.dart` |
| IT-005 | BR-002, FR-012, FR-013, FR-014, FR-015, FR-021, FR-025 | AC-011, AC-012, AC-024, AC-028 | Foreground, background, and terminated-open simulations activate exact recipient before navigation. | A active; retained B; bindings and identities; fake runtime lifecycle/router; malformed, ambiguous, and removed-account variants. | Feed open attempts through the existing runtime path. | Exact B opens activate B before navigation; invalid inputs use generic fallback; removed B uses exact signed-out message; inactive foreground banner identifies B. | `app/test/notifications/notification_open_flow_test.dart`, `app/test/notifications/services/notification_runtime_lifecycle_test.dart` |
| IT-006 | FR-020, RULE-002 | AC-013, AC-023, AC-031 | Notification lists, preferences, seen state, and new counts remain account-scoped. | A/B repositories with different tokens, preferences, lists, counts, and controlled errors. | Switch accounts, open switcher, mark seen, and deliver duplicate foreground event for B. | Each request/mutation stays on its DID; count failures isolate; authoritative refresh prevents duplicate inflation. | `app/test/notifications/providers/notifications_provider_test.dart`, `app/test/notifications/providers/notification_preferences_provider_test.dart`, `app/test/notifications/providers/account_notification_new_count_provider_test.dart` |
| IT-007 | BR-003, FR-016, FR-018 | AC-015, AC-018 | Confirmed sign-out removes only active account and routes to MRU fallback. | A active; B/C retained; fake successful logout and subscription cleanup; account-scoped provider state. | Sign out A. | Only A is deleted; correct fallback activates and lands Home; B/C state and sessions remain. | `app/test/auth/providers/auth_controller_test.dart`, `app/test/settings/sign_out_tile_test.dart` |
| IT-008 | BR-003, FR-008, FR-017, FR-018 | AC-016 | Concurrent `401` invalidation cannot remove the wrong account. | Start request under A; switch to B; start B request; return A `401`, then repeat variants. | Complete interceptor handling. | Only the session captured by the unauthorized request is invalidated; B remains; fallback routing occurs only if invalidated DID was active. | `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart` |
| IT-009 | FR-017, FR-024 | AC-016, AC-027 | Startup validation isolates mixed outcomes. | Active A and inactive B/C; controlled whoami calls with network failure, unauthorized, identity mismatch, and delayed completion. | Start auth session and release calls in chosen order. | A restores and validates first; bounded inactive validation does not block use; each authoritative result applies only to its original account/generation. | `app/test/auth/providers/auth_session_provider_test.dart` |
| IT-010 | BR-003, FR-016, FR-026, NFR-002 | AC-029 | Offline sign-out recovery preserves ordering across secure storage, provider-token rotation, server cleanup, and re-registration. | A/B retained; AppView unavailable on A logout; fake token rotation and later recovery; capture every operation. | Sign out A offline, then restore connectivity. | A is immediately non-activatable; stale opens fail; operation order is deactivate A, delete cleanup credential, then register B to replacement token; no secret is logged. | `app/test/notifications/services/notification_sign_out_recovery_test.dart` |
| IT-011 | FR-004, FR-005, FR-006, FR-022, RULE-004, RULE-005 | AC-005, AC-006, AC-007, AC-021, AC-025, AC-030 | Compact and large shell presentations expose equivalent account-switch behavior. | Pump shell under compact and large form factors with five accounts and missing identity variants. | Tap/long-press/keyboard-activate Profile and inspect semantics/actions. | Correct bottom sheet or anchored menu appears; normal tap is unchanged; avatar fallback, ordering, disabled helper, accessibility, and action omissions match. | `app/test/router/app_shell_account_switcher_test.dart` |
| IT-012 | FR-023 | AC-026 | Existing unsaved-work protection gates manual and notification switches. | Pump dirty composer/edit flow under A; retain B; capture activation and navigation. | Request B manually and through notification; cancel and confirm variants. | Confirmation occurs before activation; cancel destroys attempt; confirm activates B and follows source-specific destination. | `app/test/router/account_switch_unsaved_work_test.dart`, `app/test/feed/widgets/post_composer_sheet_discard_test.dart` |
| IT-013 | BR-003, FR-016, FR-026 | AC-015, AC-029 | AppView logout deactivates only the authenticated account on a shared installation and fails closed if subscription cleanup fails. | Seed active CraftSky sessions and push subscriptions for A and B on the same device; include a notification-cleanup failure variant. | POST A's `/v1/auth/logout`; then exercise the failure variant. | Success revokes A's session and deactivates only A's current-installation subscription while B remains active; cleanup failure returns an error without revoking A, so retry can preserve deactivation-before-re-registration ordering. | `appview/internal/auth/handlers_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Single-account sign-in and onboarding routing remain simple. | FR-003, NFR-004 | AC-003, AC-006 | With no retained account, sign in once and verify the existing onboarding/Home decisions without extra prompts. |
| REG-002 | Ordinary Profile tap and single-account navigation remain unchanged. | FR-004, FR-006, NFR-004 | AC-006 | Tap Profile on compact and large layouts and verify existing branch navigation; the switcher appears only through the account-switch action. |
| REG-003 | Anonymous routes omit bearer auth and active-account API calls select the active bearer. | FR-008 | AC-009 | Re-run interceptor tests for `/v1/auth/login`, device ID, and signed-in requests after introducing account-bound clients. |
| REG-004 | Viewer-specific feed/profile/project/saved/optimistic state never survives under the wrong identity. | FR-009, NFR-001 | AC-008, AC-009 | Switch accounts with representative providers populated and assert invalidation/DID-keying plus stale-result rejection. |
| REG-005 | Notification preferences, notification list, new-count, and mark-seen behavior retain current API semantics. | RULE-002 | AC-013 | Re-run existing notification repository/provider/seen-flow suites under a one-account registry. |
| REG-006 | Existing typed notification destinations and latest-open routing remain provider-neutral. | BR-002, FR-012 | AC-011, AC-012, AC-020 | Re-run destination inference, navigation, open-event, pending-open, and router tests with account selection inserted before inference. |
| REG-007 | Permission gating, retry behavior, and OS-visible notification title/body remain unchanged. | FR-010, FR-011 | AC-010, AC-014, AC-024 | Re-run registration lifecycle and background-handler suites; assert OS-visible payload copy does not gain recipient identity. |
| REG-008 | Existing composer/edit discard confirmation still protects unsaved work. | FR-023 | AC-026 | Re-run composer discard tests and verify the shared guard is used rather than bypassed by account activation. |
| REG-009 | Existing per-account Sign out remains available without adding bulk logout. | FR-016, RULE-003 | AC-015, AC-017 | Inspect/pump settings under active account and verify one-account sign-out exists and sign-out-all does not. |
| REG-010 | Secrets and routing data remain excluded from diagnostic and user-visible output; identity data remains excluded from diagnostics and appears only on designated account-identification UI; the opaque routing ID remains available to the provider-routing parser. | NFR-002 | AC-019, AC-029 | Run sentinel scans across registry, recovery, routing, and observability paths; assert registration-response/provider-payload parsing of `accountSubscriptionId`; assert cached handle/avatar rendering only in the switcher, Profile destination, and inactive-recipient banner. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Stable retained accounts | A=`did:plc:alice`, B=`did:plc:bob`, C=`did:plc:carol`; distinct tokens, handles, onboarding states, avatars, and activation times. | AT-001–AT-012, UT-001–UT-022, IT-001–IT-013 |
| TD-002 | Registry recovery fixtures | Versioned empty/valid registry; missing active DID; one corrupt entry; partial/interrupted write; unsupported version; all-invalid data. | AT-001, UT-001, IT-001 |
| TD-003 | Account-limit fixtures | Five distinct DIDs, a sixth distinct DID, and a replacement session for one of the existing five. | AT-002, UT-002, UT-016, IT-011 |
| TD-004 | Controlled async account work | Completers for A/B requests, validation, optimistic mutation, registration, activation, and navigation with account-generation markers. | AT-004, AT-010, UT-004, UT-005, UT-010, IT-003, IT-008, IT-009 |
| TD-005 | Notification routing fixtures | Unique A/B bindings, duplicate binding, malformed/missing ID, valid unbound ID, removed-account binding history, typed destinations. | AT-005, AT-012, UT-006–UT-008, IT-005 |
| TD-006 | Account-scoped notification state | Different A/B preferences, lists, seen state, and counts `0`, `3`, `99`, `100`, `120`; per-account success/401/network outcomes. | AT-006, UT-009, IT-006 |
| TD-007 | Push-registration fixtures | Authorized/denied permission, old/new provider tokens, eligible/ineligible accounts, per-DID bindings, transient failure. | AT-007, UT-011, IT-004, MAN-002 |
| TD-008 | Sign-out recovery trace | A cleanup-only credential, B usable session, unreachable/recovered AppView, rotated provider token, ordered operation recorder. | AT-011, UT-013, IT-010 |
| TD-009 | Secret sentinels | Recognizable fake token, routing ID, raw payload, DID, handle, and retained-account-list values. | UT-014, REG-010, MAN-003 |
| TD-010 | Responsive identity fixtures | Compact/large constraints, complete/missing avatar and display-name data, cached handles, five-account switcher. | AT-003, AT-012, UT-016–UT-018, IT-011, MAN-001 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-004, FR-021, FR-022, NFR-003 | AC-005, AC-006, AC-008, AC-024, AC-025 | Final responsive, visual, keyboard, and screen-reader behavior. | On phone and large layout, inspect Profile avatar/selected state; open switcher by long press and keyboard/accessibility action; inspect transition state, five-account helper, count badges, identity fallbacks, and inactive-recipient banner. | Bottom sheet/popover placement is clear; semantics announce identity/state/count/action; transition prevents accidental actions; banner and avatar fallbacks are visually unambiguous. |
| MAN-002 | BR-002, FR-010, FR-011, FR-012, FR-013, FR-026 | AC-010, AC-011, AC-014, AC-024, AC-029 | Physical-device multi-account push delivery and recovery. | Retain two test accounts on a device; deliver foreground, background, and terminated notifications to each; rotate FCM token; perform offline sign-out and reconnect. | Both eligible accounts receive pushes; opens activate exact recipients; inactive foreground banner is identified; removed account cannot open; retained account resumes on replacement token only after cleanup ordering. |
| MAN-003 | NFR-002 | AC-019, AC-029 | Platform secure-storage, permitted routing transport, diagnostic redaction, and intentional identity UI. | Inspect debug storage backend/keys without printing values; confirm the opaque `accountSubscriptionId` appears only in the authenticated registration response and provider payload; exercise add, switch, sign-out, crash/error, and recovery paths; search captured logs, URLs, and string output using sentinel credentials/routing/identity fixtures; inspect account-identification UI. | Tokens, cleanup credentials, and the DID-to-ID map remain in platform secure storage; the opaque ID is usable only for provider routing and never appears in diagnostic or user-visible output; identity values remain absent from diagnostics while the intended cached handle/avatar appears only in the switcher, Profile destination, and inactive-recipient banner; cleanup-only credentials disappear after terminal recovery. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real provider delivery in background and terminated application states is not fully reproducible in hermetic Flutter tests. | BR-002, FR-010–FR-015, FR-021 | Firebase/APNs lifecycle behavior depends on provider and OS process state. | Keep runtime-path simulations automated and require MAN-002 on supported device platforms before release. |
| GAP-002 | Platform secure-storage encryption and OS keychain/keystore access controls cannot be proven by Dart unit tests. | NFR-002 | Flutter mocks verify application behavior, not platform security implementation. | Retain architecture/secret-scan tests, inspect platform configuration, and complete MAN-003. |
| GAP-003 | The existing AppView contract is load-bearing even though no production server change is expected. | FR-008, FR-010, FR-011, FR-016, FR-026 | Current handlers support independent sessions, per-account registration, and current-installation logout cleanup, but client-only work could otherwise skip verification of that contract. | Implement `IT-013` and run the focused/full Go contract tests for this feature regardless of whether production AppView code changes. |
| GAP-004 | Operational alert thresholds for routing and registration failures are unspecified. | FR-010–FR-015, FR-026 | Requirements deliberately leave thresholds as an operational follow-up. | Define dashboards/thresholds before production readiness; this does not block client TDD because the app has no active users. |
| GAP-005 | Partial-write recovery depends on the chosen registry write protocol and secure-storage backend guarantees. | FR-001, NFR-002 | Requirements define the recovery outcome but not the final atomicity design. | Document the versioned write/recovery protocol in the coding plan and retain TD-002 coverage for every chosen failure point. |

## 10. Out Of Scope

- A server-side linked-account group, cross-account token exchange, or any API that exposes one account's retained sessions to another.
- A legacy migration from the existing single-session blob; the requirements establish that there are no signed-in installations to migrate.
- New AppView routes, database migrations, or notification payload recipient copy; the existing AppView contract is verified through `IT-013` and normal server tests.
- A sign-out-all action or direct removal of an inactive account from the switcher.
- Per-account saved navigation locations; manual switching always resets to the selected account's Home root.
- Provider-specific destination logic or trusting notification payload facts as authorization.
- Alert threshold tuning and production operations rollout.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-17-multi-account-sessions/`
- Risk level: **High**; document review and explicit approval are required before implementation.
- Recommended first failing test for implementation: `UT-001` in proposed `app/test/auth/models/session_registry_test.dart`, proving versioned two-account registry round-trip plus recovery from a missing/corrupt active pointer before any provider or UI depends on the model.
- Suggested test order for implementation: `UT-001`–`UT-003` registry and MRU rules; `IT-001` storage/provider restoration; `UT-005` request-token and `401` scoping; `UT-004`/`IT-003` activation boundary; `IT-002` additive OAuth and `UT-022` cancellation/failure preservation; `UT-010`/`IT-009` validation; `UT-012`/`IT-007` confirmed sign-out plus AppView contract `IT-013`; `UT-006`–`UT-008` and `IT-005` routing; `UT-011`/`IT-004` registration; `UT-009`/`IT-006` counts; `UT-015`/`IT-012` unsaved-work guard; `UT-013`/`IT-010` offline recovery; `IT-011` switcher UI; remaining regression and manual checks.
- Commands discovered:
  - From `app/`: `flutter test test/auth test/notifications test/router`
  - From `app/`: `flutter test test/shared/api/providers/session_auth_interceptor_test.dart test/shared/api/providers/sign_out_on_401_interceptor_test.dart`
  - From `app/`: `flutter test test/settings test/feed/widgets/post_composer_sheet_discard_test.dart`
  - From `app/`: `dart analyze`
  - From `app/`, when generated Riverpod/mapper code changes: `dart run build_runner build`
  - From repository root, required for the AppView contract even if production server code is unchanged: `just test`
- Blocking gaps: None after document correction. `GAP-001`, `GAP-002`, and `GAP-005` require explicit treatment because risk is High; `GAP-003` is addressed by planned `IT-013` and mandatory AppView verification.

## 12. Approved Simplification Test Amendment

Approved by the user on 2026-07-18. These tests supersede the prior partial-write journal, offline cleanup recovery, inactive startup sweep, inactive switcher count, and global transition-overlay expectations in UT-001, UT-004, UT-009, UT-010, UT-013, IT-001, IT-003, IT-006, IT-009, IT-010, AT-004, AT-006, AT-010, AT-011, MAN-001, MAN-002, MAN-003, TD-002, TD-006, TD-008, GAP-003, and GAP-005. Other account-isolation assertions in those tests remain applicable.

| Order | Test ID | Requirement IDs | Acceptance Criteria | Focused Behavior | Automation Target |
|---|---|---|---|---|---|
| 1 | SIM-UT-001 | SIM-FR-001, NFR-002 | SIM-AC-001 | One secure snapshot round-trips valid accounts, missing/corrupt/unsupported/read-failed storage returns empty signed-out state, and a failed write does not publish. | `app/test/auth/models/session_registry_test.dart`, `app/test/auth/providers/secure_token_storage_test.dart`, `app/test/auth/providers/account_session_registry_provider_test.dart` |
| 2 | SIM-UT-002 | SIM-FR-002, FR-016, FR-018 | SIM-AC-002 | Transient manual logout failure retains the complete registry and active account with no success result; success and authoritative unauthorized remove only the selected lease. | `app/test/auth/providers/auth_controller_test.dart`, `app/test/settings/sign_out_tile_test.dart` |
| 3 | SIM-UT-003 | SIM-FR-003, FR-017 | SIM-AC-003 | Startup calls `whoami` only for the active lease; inactive sessions remain until selected/used, where account-bound unauthorized handling removes only that lease. | `app/test/auth/providers/auth_session_provider_test.dart`, `app/test/auth/services/session_validation_coordinator_test.dart` |
| 4 | SIM-UT-004 | SIM-FR-004, RULE-002 | SIM-AC-004 | Opening the switcher creates no inactive count requests and renders no inactive badges; active navigation count behavior remains green. | `app/test/auth/models/account_switcher_state_test.dart`, `app/test/router/app_shell_account_switcher_test.dart` |
| 5 | SIM-IT-005 | SIM-FR-005, FR-007, FR-009, NFR-001 | SIM-AC-005 | A pending manual activation disables switcher rows/Add and shows progress; success dismisses it and resets Home; there is no global transition overlay and stale completion protection remains. | `app/test/auth/providers/account_activation_coordinator_test.dart`, `app/test/router/app_shell_account_switcher_test.dart`, `app/test/auth/providers/account_boundary_provider_test.dart` |
| 6 | SIM-REG-006 | SIM-NFR-001, SIM-FR-001, SIM-FR-005 | SIM-AC-006 | Source/call-site scan and focused regressions prove obsolete compatibility/recovery/transition/count/validation symbols are absent while serialized registry mutation and account boundaries remain green. | `app/test/observability/secret_scan_test.dart` plus `rg`/generated-output verification |

### Revised Gaps And Manual Checks

- Platform secure-storage encryption remains a manual platform concern, but partial entry/journal recovery is no longer required. The accepted failure mode is local sign-out and reauthentication.
- Physical-device multi-account push delivery and exact recipient opens remain a manual check. Offline sign-out and replacement-token recovery are no longer part of that check.
- Visual/accessibility inspection should verify switcher-local progress and disabled semantics; no global identity transition overlay is expected.
- The AppView shared-installation confirmed-logout isolation test remains required. Cleanup-failure behavior now proves that the client can retry while its local account stays retained.
