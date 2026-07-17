# Acceptance Test Specification: Direct Push Notification Routing

## 1. Test Strategy

Risk level: High.

The test design proves the replacement notification-fact contract from both ends. Go unit and integration tests verify that AppView projects only the minimum canonical facts from durable notification events. Flutter unit tests verify strict provider-data parsing, category-to-destination inference, account binding, readiness, and error classification without Firebase. Widget/router acceptance tests verify immediate navigation, reply focus, safe fallbacks, permanent-unavailable UI, retry behavior, sign-out behavior, and non-interactive generic rows. Regression tests protect notification delivery, preferences, newness, and provider-neutral architecture while confirming that notification-ID resolution is gone.

Provider-data parsing is specified as a provider-neutral open attempt with two independently testable results: account-binding validity and notification-fact validity. A valid binding therefore survives a malformed/legacy fact contract long enough for the coordinator to apply the required Notifications fallback, while an absent, malformed, stale, or mismatched binding blocks every route, including fallback navigation. This describes the observable boundary without requiring a particular class hierarchy.

Physical-device checks are limited to behavior that host tests cannot faithfully prove: actual Android/iOS background and terminated delivery, stale accepted pushes, multi-account OS notification taps, and qualitative cold-start latency. No test in this specification treats push facts as authorization; destination reads must still pass through authenticated AppView APIs.

The non-blocking latency question from the requirements does not block test design. This slice uses deterministic structural proof that no notification-specific network call precedes navigation, plus a manual cold-start observation. It does not introduce a numeric performance threshold or new identifier-bearing telemetry.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-007 | AT-001, UT-007, IT-003 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-006, AC-008 | AT-002, AT-006, IT-004, IT-005 | Acceptance / Integration | Yes |
| FR-001 | AC-002, AC-014 | UT-001, UT-011, IT-001, IT-002, REG-008 | Unit / Integration / Regression | Yes |
| FR-002 | AC-002, AC-003, AC-018 | AT-001, UT-002, UT-004, UT-011, UT-012, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003, AC-016 | AT-001, UT-004, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-004 | UT-011, IT-002, REG-001 | Unit / Integration / Regression | Yes |
| FR-005 | AC-005 | UT-001, UT-002, UT-003, UT-010 | Unit | Yes |
| FR-006 | AC-006 | AT-002, UT-001, UT-006, IT-004, MAN-004 | Acceptance / Unit / Integration / Manual | Yes, plus manual |
| FR-007 | AC-001, AC-007 | AT-001, UT-007, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-008 | AT-006, IT-005 | Acceptance / Integration | Yes |
| FR-009 | AC-009 | AT-006, IT-006, MAN-003 | Acceptance / Integration / Manual | Yes, plus manual |
| FR-010 | AC-010 | AT-007, UT-009, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-011 | AT-003, UT-001, UT-005, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-012, AC-021 | AT-003, UT-001, UT-002, UT-005 | Acceptance / Unit | Yes |
| FR-013 | AC-013 | AT-004, IT-004, REG-010, MAN-001, MAN-002 | Acceptance / Integration / Regression / Manual | Yes, plus manual |
| FR-014 | AC-014 | UT-001, UT-011, IT-002, REG-008 | Unit / Integration / Regression | Yes |
| FR-015 | AC-015 | AT-009, UT-013, IT-008, REG-008 | Acceptance / Unit / Integration / Regression | Yes |
| FR-016 | AC-003, AC-016 | AT-005, UT-004, UT-014, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-003, AC-016 | UT-011, IT-001 | Unit / Integration | Yes |
| FR-018 | AC-022 | AT-008, UT-008, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-005, AC-021 | AT-003, UT-002, UT-003 | Acceptance / Unit | Yes |
| FR-020 | AC-023 | AT-010, UT-016, UT-017, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-024 | UT-018, UT-020, UT-021 | Unit / Widget | Yes |
| FR-022 | AC-025 | UT-022, UT-023, IT-014 | Unit / Widget / Integration | Yes |
| NFR-001 | AC-007 | AT-001, UT-007, IT-003, MAN-005 | Acceptance / Unit / Integration / Manual | Yes, plus manual |
| NFR-002 | AC-017 | UT-010, IT-010, REG-007 | Unit / Integration / Regression | Yes |
| NFR-003 | AC-018 | UT-012, IT-002 | Unit / Integration | Yes |
| NFR-004 | AC-019 | UT-015, REG-007 | Unit / Regression | Yes |
| NFR-005 | AC-009, AC-010 | AT-006, AT-007, IT-006, IT-007 | Acceptance / Integration | Yes |
| RULE-001 | AC-004, AC-017 | UT-011, IT-002, IT-010 | Unit / Integration | Yes |
| RULE-002 | AC-008 | AT-006, IT-005 | Acceptance / Integration | Yes |
| RULE-003 | AC-006 | AT-002, UT-006, IT-004, MAN-004 | Acceptance / Unit / Integration / Manual | Yes, plus manual |
| RULE-004 | AC-012 | AT-003, UT-005 | Acceptance / Unit | Yes |
| RULE-005 | AC-008, AC-009 | AT-006, IT-005, IT-006, MAN-003 | Acceptance / Integration / Manual | Yes, plus manual |
| RULE-006 | AC-020 | REG-001, REG-002, REG-003, REG-004, REG-005, REG-006, REG-010 | Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Known notification facts navigate immediately to Flutter-inferred destinations

Requirement IDs: BR-001, FR-002, FR-003, FR-007, NFR-001

Acceptance Criteria: AC-001, AC-003, AC-007

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/notification_open_flow_test.dart`, `app/test/router/notification_open_routing_test.dart` (new)

```gherkin
Feature: Direct notification routing
  Scenario Outline: A known notification opens its Flutter-owned destination
    Given the app is ready for the authenticated account
    And the notification has payload version 1 and a matching account-subscription binding
    And the notification type is <type> with <facts>
    When the member opens the notification
    Then Flutter navigates immediately to <destination>
    And no notification-resolution request is made before navigation

    Examples:
      | type           | facts                         | destination                         |
      | follow         | actorDid                      | actor DID profile                   |
      | like           | subjectUri and rootUri        | root thread, subject focused if different |
      | repost         | subjectUri and rootUri        | root thread, subject focused if different |
      | mention        | sourceUri                     | source post thread                  |
      | quote          | sourceUri                     | quoting source post thread          |
      | reply          | subjectUri and sourceUri      | subject thread focused on source    |
      | everythingElse | no category-specific facts    | Notifications                       |
```

### AT-002: Account binding gates navigation and fallback

Requirement IDs: BR-002, FR-006, RULE-003

Acceptance Criteria: AC-006

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/providers/notification_open_coordinator_test.dart`

```gherkin
Feature: Account-isolated notification routing
  Scenario Outline: An unbound notification cannot navigate
    Given the app is ready for the current authenticated DID
    And all non-binding facts are valid
    And the binding state is <binding state>
    When the member opens the notification
    Then no destination or Notifications fallback navigation occurs
    And no notification-resolution or destination request occurs
    And generic unavailable feedback is shown

    Examples:
      | binding state              |
      | missing local binding      |
      | missing payload binding    |
      | malformed payload binding  |
      | mismatched binding         |
      | stale different-account ID |
```

### AT-003: Invalid payloads and future types use distinct safe fallbacks

Requirement IDs: FR-011, FR-012, FR-019, RULE-004

Acceptance Criteria: AC-011, AC-012, AC-021

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/notification_open_flow_test.dart`

```gherkin
Feature: Safe notification-fact fallback
  Scenario Outline: Invalid or future facts do not create unintended routes
    Given the notification binding matches the current account
    And the provider data has <condition>
    When the member opens the notification
    Then the app opens Notifications
    And feedback is <feedback>
    And no notification-resolution request is made

    Examples:
      | condition                                      | feedback                         |
      | no payloadVersion                              | brief unable-to-open feedback    |
      | malformed or unsupported payloadVersion        | brief unable-to-open feedback    |
      | malformed type                                 | brief unable-to-open feedback    |
      | known type with missing or malformed fact      | brief unable-to-open feedback    |
      | unknown syntactically valid type with extras   | none                              |

  Scenario: Extras do not change a valid known-category route
    Given the notification binding matches the current account
    And a known type has all required valid facts plus unexpected extras
    When the member opens the notification
    Then Flutter opens the normal inferred destination
    And the extras have no effect on parsing or routing
    And no unable-to-open feedback is shown
    And no notification-resolution request is made
```

### AT-004: Every provider-open source uses the same policy

Requirement IDs: FR-013

Acceptance Criteria: AC-013

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/notification_open_flow_test.dart`, `app/test/notifications/notification_effect_host_test.dart`

```gherkin
Feature: Provider-neutral notification opens
  Scenario Outline: Equivalent callbacks produce the same navigation
    Given equivalent valid notification facts arrive from <source>
    And readiness and account binding succeed
    When the callback is consumed
    Then the same parsing, inference, and navigation policy runs
    And exactly one navigation outcome is emitted for that callback

    Examples:
      | source                    |
      | foreground banner tap     |
      | background notification   |
      | terminated initial open   |
```

### AT-005: Reply notifications preserve thread context and focus

Requirement IDs: FR-016

Acceptance Criteria: AC-003, AC-016

Priority: Must

Level: Acceptance

Automation Target: `app/test/router/notification_open_routing_test.dart` (new), `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Reply notification routing
  Scenario: A reply opens its parent thread focused on the reply
    Given a valid reply notification contains a subjectUri and sourceUri
    When Flutter infers and opens the destination
    Then the route path identifies the subject post thread
    And the focus query identifies the source reply
    And the focused reply branch is requested and rendered
```

### AT-006: A permanently unavailable destination remains understandable and authorized

Requirement IDs: BR-002, FR-008, FR-009, NFR-005, RULE-002, RULE-005

Acceptance Criteria: AC-008, AC-009

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart` (new), `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Permanent destination unavailability
  Scenario Outline: AppView refuses a stale destination
    Given navigation was inferred from valid notification facts
    And the authenticated destination request returns <error>
    When the destination page renders
    Then the intended route remains visible
    And localized permanent-unavailable copy is shown
    And accessible Back and View notifications actions are available
    And no stale payload content is displayed
    And the app does not automatically redirect

    Examples:
      | error                 |
      | 404 post_not_found    |
      | 404 profile_not_found |
```

### AT-007: Transient and authentication failures keep their normal semantics

Requirement IDs: FR-010, NFR-005

Acceptance Criteria: AC-010

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart` (new), `app/test/profile/profile_page_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart`

```gherkin
Feature: Destination failure handling
  Scenario Outline: A transient destination failure is retryable in place
    Given a notification destination is open
    And its AppView request fails with <failure>
    When the page renders the error
    Then the intended route remains visible
    And localized accessible Retry is available
    And no notification-open retry is persisted or scheduled

    Examples:
      | failure                  |
      | network failure          |
      | 500 server error         |
      | 502 identity_unavailable |

  Scenario: Authentication expires while the destination loads
    Given a notification destination is open
    When its AppView request returns 401
    Then the normal global sign-out flow runs
    And no notification-specific unavailable or retry state remains
```

### AT-008: Only current intent survives transient readiness

Requirement IDs: FR-018

Acceptance Criteria: AC-022

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/services/pending_notification_open_test.dart`, `app/test/notifications/notification_open_flow_test.dart`

```gherkin
Feature: Pending notification opens
  Scenario: The latest open wins during transient readiness
    Given the authenticated account is restoring or completing onboarding
    When multiple notification opens arrive
    And readiness becomes ready
    Then only the most recent open is processed

  Scenario: Required sign-in clears pending intent
    Given a notification open is pending during transient readiness
    When readiness changes to requires sign-in
    Then the pending open is discarded permanently
    And later sign-in to any account, including the same DID, does not revive it
```

### AT-009: Generic notification-feed rows are informational only

Requirement IDs: FR-015

Acceptance Criteria: AC-015

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/notifications_page_test.dart`

```gherkin
Feature: Generic notification rows
  Scenario Outline: A non-destination row cannot trigger resolution or navigation
    Given the Notifications page shows a <row type> row
    When the row is inspected and tapped
    Then it has no interactive tap action
    And no notification-resolution request or route navigation occurs

    Examples:
      | row type         |
      | generic          |
      | unknown category |
```

### AT-010: Notification copy uses the target's conversation role

Requirement IDs: FR-020

Acceptance Criteria: AC-023

Priority: Must

Level: Acceptance

Automation Target: `app/test/notifications/notifications_page_test.dart`, `appview/internal/push/payload_test.go`, `appview/internal/push/dispatcher_test.go`

```gherkin
Feature: Role-aware notification wording
  Scenario Outline: Visible notification copy names the actual target role
    Given notification activity targets a <target role>
    When the OS notification is built or the in-app row is rendered
    Then the visible copy uses <expected noun or action>

    Examples:
      | target role              | expected noun or action       |
      | root post                | post                          |
      | direct comment           | comment                       |
      | nested reply             | reply                         |
      | new direct child of post | commented on your post        |
      | response to comment      | replied to your comment       |
      | response to reply        | replied to your reply         |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-005, FR-006, FR-011, FR-012, FR-014 | AC-002, AC-005, AC-006, AC-011, AC-012, AC-014 | Parse the common provider envelope into a structured provider-neutral open attempt. | Valid/missing/malformed `payloadVersion`, `type`, and `accountSubscriptionId`; legacy `notificationId`. | A valid binding is preserved independently from fact validity; missing/malformed version or type produces an invalid-facts outcome that can be handled after binding validation; missing/malformed binding produces an invalid-binding outcome; `notificationId` is ignored and absent from the domain attempt. | `app/test/notifications/models/notification_open_event_test.dart` |
| UT-002 | FR-002, FR-012, FR-019 | AC-002, AC-005, AC-012, AC-021 | Enforce the per-category required-fact matrix while ignoring extras. | All known categories, required fields, missing fields, and destination-shaped extras, each with a valid binding. | Each category accepts only its valid required facts; missing/malformed facts produce an invalid-facts outcome that retains the valid binding; extras do not invalidate or influence routing. | `app/test/notifications/models/notification_open_event_test.dart` |
| UT-003 | FR-005, FR-019 | AC-005, AC-021 | Validate public identifiers at the provider boundary. | Valid/invalid DIDs; valid post AT-URIs; non-post collections; malformed and arbitrary URLs. | Only typed DIDs and `social.craftsky.feed.post` AT-URIs reach domain logic; extras remain unused. | `app/test/notifications/models/notification_open_event_test.dart` |
| UT-004 | FR-002, FR-003, FR-016 | AC-003, AC-016 | Infer destinations from notification category and canonical facts. | Complete version 1 fact fixtures for all categories. | Exact Q4 mapping is returned, including like/repost root thread with differing subject focus and reply subject thread with source focus; no AppView/GoRouter path input is required. | `app/test/notifications/services/notification_destination_inference_test.dart` (new) |
| UT-005 | FR-011, FR-012, RULE-004 | AC-011, AC-012 | Classify fact outcomes after successful binding validation. | Pre-cutover payload; unsupported/malformed version; malformed type; malformed known facts; unknown valid type, all carrying a valid matching binding. | Invalid/legacy fact outcomes request Notifications plus feedback; unknown valid type requests Notifications quietly. | `app/test/notifications/services/notification_destination_inference_test.dart` (new) |
| UT-006 | FR-006, RULE-003 | AC-006 | Apply the DID-keyed account-subscription gate before any routing or fact fallback. | Matching binding; missing/malformed payload binding; missing local binding; stale and mismatched bindings. | Only a valid matching binding proceeds to fact inference/fallback; every other state emits generic unavailable feedback with zero route/network calls, including zero Notifications fallback calls. | `app/test/notifications/providers/notification_open_coordinator_test.dart` |
| UT-007 | BR-001, FR-007, NFR-001 | AC-001, AC-007 | Prove navigation emission is not serially dependent on network work. | Valid event, matching binding, route spy, destination-load future held incomplete. | Typed navigation emits while the destination future remains incomplete and no resolver exists/calls. | `app/test/notifications/providers/notification_open_coordinator_test.dart` |
| UT-008 | FR-018 | AC-022 | Retain latest only and clear on authentication boundary. | Two or more opens; transient, ready, and requires-sign-in transitions. | Only latest is yielded once on ready; requires-sign-in clears pending state permanently. | `app/test/notifications/services/pending_notification_open_test.dart` |
| UT-009 | FR-010 | AC-010 | Classify destination errors. | Network error, each `5xx`, `502 identity_unavailable`, `401`, `404 post_not_found`, and `404 profile_not_found`. | Transient errors map to in-place Retry, `401` to global sign-out, named `404`s to permanent unavailable; no automatic open retry. | `app/test/shared/errors/notification_destination_error_test.dart` (new) |
| UT-010 | FR-005, NFR-002 | AC-005, AC-017 | Keep domain diagnostics identifier-free. | Sentinel account-subscription ID, DID, AT-URI, token, payload, and parse failures. | `toString`, exceptions, feedback, and observer values expose only bounded outcome/category classes. | `app/test/notifications/models/notification_open_event_test.dart`, `app/test/shared/errors/sentry_redaction_test.dart` |
| UT-011 | FR-001, FR-002, FR-004, FR-014, FR-017, RULE-001 | AC-002, AC-004, AC-014, AC-016 | Build the exact AppView payload matrix from canonical facts. | One durable fact fixture per category, including everything-else. | Common fields and exact minimum category facts are present; `notificationId`, route policy, handles, content, and unnecessary references are absent. | `appview/internal/push/payload_test.go` |
| UT-012 | FR-002, NFR-003 | AC-018 | Bound the largest provider data map. | Reply payload plus sentinel-size valid values. | Flat string map stays within declared field/value bounds with no nested serialization or content body. | `appview/internal/push/payload_test.go` |
| UT-013 | FR-015 | AC-015 | Render generic/unknown feed rows without interaction. | Generic and unknown `CraftskyNotification` rows. | Row is informational, has no tap callback/semantics action, and cannot call a resolution repository. | `app/test/notifications/notifications_page_test.dart` |
| UT-014 | FR-016 | AC-003, AC-016 | Construct the typed reply route safely. | Valid subject post URI and source reply URI. | Path uses subject DID/rkey and `focus` contains the source reply URI; no arbitrary path is executed. | `app/test/router/notification_open_routing_test.dart` (new) |
| UT-015 | NFR-004 | AC-019 | Preserve the provider-neutral testing boundary. | Source/import scan and fakes for parser, inference, readiness, binding, and navigation. | Domain/coordinator suites run without Firebase initialization, FCM, OS permission, or a device; Firebase stays in its adapter. | `app/test/notifications/notification_architecture_test.dart` |
| UT-016 | FR-020 | AC-023 | Render role-aware in-app row copy. | Like, repost, and response rows whose hydrated target is a root post, direct comment, or nested reply. | Rows use post/comment/reply nouns and use `commented on` only for a direct child of a root. | `app/test/notifications/notifications_page_test.dart` |
| UT-017 | FR-020 | AC-023 | Build role-aware OS-visible copy without adding routing data. | Current categories with post, comment, and reply target roles. | Notification title/body uses the correct bounded English action while the exact provider data map remains unchanged. | `appview/internal/push/payload_test.go` |
| UT-018 | FR-021 | AC-024 | Render complete Bluesky-style in-app notification identity/action/recency context. | Follow, like, repost, reply, mention, quote, and generic rows with an actor avatar URL and fixed creation time. | Every row contains `ProfileAvatar` above its actor copy, a bold actor name, the outlined icon shared with notification settings, compact relative time, and a full timestamp tooltip. | `app/test/notifications/notifications_page_test.dart` |
| UT-020 | FR-021 | AC-024 | Return a display-ready actor avatar from the notification API. | Indexed actor avatar CID and MIME plus resolved handle. | The additive actor `avatar` field uses the canonical CDN URL while the existing CID remains available. | `appview/internal/api/notifications_test.go` |
| UT-021 | FR-021 | AC-024 | Preserve real public avatars while the additive AppView avatar URL is absent. | Notification actor JSON containing DID and public avatar CID but no `avatar` field. | The Flutter model derives the canonical CDN avatar URL; a development-media CID remains on the standard initial fallback rather than inventing an invalid URL. | `app/test/notifications/notifications_page_test.dart` |
| UT-022 | FR-022 | AC-025 | Serialize current actor follow state in the existing notification response. | A follow notification row whose actor is currently followed by the authenticated viewer. | The additive camelCase actor `viewerIsFollowing` field is `true`; absent/false state remains `false`. | `appview/internal/api/notifications_test.go` |
| UT-023 | FR-022 | AC-025 | Render and toggle the follow-notification relationship control. | A follow row starting at `viewerIsFollowing=true` with a fake profile repository returning unfollowed then followed profiles. | The row starts at Unfollow, invokes unfollow by actor DID, changes to Follow, invokes follow by actor DID, returns to Unfollow, and the nested button does not trigger row navigation; the category matrix contains exactly one follow control. | `app/test/notifications/notifications_page_test.dart` |
| IT-014 | FR-022 | AC-025 | Project the authenticated viewer's current actor relationship with the notification page. | Real Postgres notification events plus an active `atproto_follows` row from viewer to actor. | Every listed notification row for that actor has `ActorViewerIsFollowing=true`, including across pagination. | `appview/internal/api/durable_notification_store_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, FR-002, FR-003, FR-017 | AC-002, AC-003, AC-016 | Project durable notification facts into a provider send request. | Seed one durable event per category with distinct actor/source/subject sentinels and an active account subscription. | Claim and dispatch each delivery to a recording sender. | The request's routing metadata contains only category-required canonical facts and account binding, with no client route policy or localized-copy-derived target; existing token, platform, TTL, and generic visible-copy inputs remain available and unchanged. | `appview/internal/push/dispatcher_test.go` |
| IT-002 | FR-001, FR-004, FR-014, NFR-003, RULE-001 | AC-002, AC-004, AC-014, AC-018 | Verify serialized provider message shape. | Build FCM messages for every category and the maximum reply fixture. | Serialize the combined notification/data message. | Data contains version 1, type, binding, and exact minimum facts; visible copy stays generic; forbidden fields, `notificationId`, and nested values are absent. | `appview/internal/push/firebase_sender_test.go`, `appview/internal/push/payload_test.go` |
| IT-003 | BR-001, FR-007, FR-011, FR-018, NFR-001 | AC-001, AC-007, AC-011, AC-022 | Exercise the provider-neutral runtime from readiness through navigation. | Memory routing storage, effect stream, structured valid/legacy open attempts, incomplete destination future, and readiness transitions. | Receive opens before and after readiness. | The runtime validates binding before consuming the fact outcome; a valid latest event emits typed navigation without resolver HTTP, a matching-bound legacy event opens Notifications with feedback, an invalid binding emits only generic unavailable feedback, and a sign-in boundary clears pending state. | `app/test/notifications/notification_open_flow_test.dart` |
| IT-004 | BR-002, FR-006, FR-013, RULE-003 | AC-006, AC-013 | Apply the same binding gate to every open source. | Equivalent foreground, background, and initial events with matching and mismatched bindings. | Send each event through its adapter into the runtime. | Matching events produce the same one navigation outcome; mismatches produce generic feedback with zero destination/resolver calls. | `app/test/notifications/notification_open_flow_test.dart`, `app/test/notifications/notification_effect_host_test.dart` |
| IT-005 | BR-002, FR-008, RULE-002, RULE-005 | AC-008 | Confirm destination APIs remain the authorization/moderation boundary. | Authenticated post/profile API tests with visible, hidden, taken-down, unauthorized, and deleted targets. | Request targets whose identifiers could have come from a push. | Responses match ordinary AppView visibility/moderation policy; the identifier alone grants no content. | `appview/internal/api/post_test.go`, `appview/internal/api/profile_test.go` |
| IT-006 | FR-009, NFR-005, RULE-005 | AC-009 | Render named permanent-unavailable errors on destination pages. | Widget repositories return mapped `post_not_found` and `profile_not_found` errors. | Pump post/profile destinations and invoke their actions. | Permanent UI stays in place, hides stale content, exposes accessible Back/View notifications actions, and never auto-redirects. | `app/test/feed/pages/post_thread_page_test.dart` (new), `app/test/profile/profile_page_test.dart` |
| IT-007 | FR-010, NFR-005 | AC-010 | Integrate transient retry and global `401` handling. | Widget/API fakes return network, `5xx`, `502 identity_unavailable`, and `401`, then optionally succeed on retry. | Load destination and tap Retry or observe interceptor sign-out. | Transient failures retry in place without persisted open retry; `401` runs global sign-out and removes notification-specific UI. | `app/test/feed/pages/post_thread_page_test.dart` (new), `app/test/profile/profile_page_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart` |
| IT-008 | FR-015 | AC-015 | Prove the clean resolver cutover at the HTTP and client boundaries. | AppView route registry and Flutter notification repository/runtime source. | Request the former notification-ID path and run structural API/client checks. | No notification-resolution handler or owner-scoped store method is registered; no Flutter resolution model/repository/policy/call remains. | `appview/internal/routes/routes_test.go`, `app/test/notifications/notification_architecture_test.dart` |
| IT-009 | FR-016 | AC-003, AC-016 | Carry reply focus through typed router into the thread fetch. | Router harness with subject URI and source focus URI; recording post repository. | Navigate from a reply fact event. | Route decodes subject DID/rkey and passes source URI as focus to the comment-section request/rendering path. | `app/test/router/notification_open_routing_test.dart` (new), `app/test/feed/pages/post_comment_section_page_test.dart` |
| IT-010 | NFR-002, RULE-001 | AC-017 | Scan success/fallback/failure paths for identifier leakage. | Unique sentinels for token, binding, notification ID, DID, AT-URI, focus URI, payload, and provider error. | Exercise AppView build/send diagnostics and Flutter parse/navigation/error/reporting paths. | Logs, Sentry, analytics, metrics labels, exceptions, feedback, and snapshots contain none of the sentinels or raw payload. | `appview/internal/observability/push_integration_test.go`, `app/test/shared/errors/sentry_redaction_test.dart`, `app/test/notifications/notification_architecture_test.dart` |
| IT-011 | FR-015 | AC-015 | Keep known notification rows useful while generic rows become inert. | Notifications page with every known category, generic, unknown, and unavailable rows. | Tap each row where interactive. | Known rows retain profile/post/reply navigation; generic/unknown rows do nothing; unavailable rows retain explicit unavailable feedback. | `app/test/notifications/notifications_page_test.dart` |
| IT-012 | FR-020 | AC-023 | Project the target's conversation role into provider-visible copy. | Durable events targeting indexed root posts, direct comments, and nested replies. | Claim and dispatch each delivery. | The send request carries only a bounded internal target-role enum derived from indexed reply structure; provider routing data gains no new key. | `appview/internal/push/dispatcher_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test | Automation Target |
|---|---|---|---|---|---|
| REG-001 | Visible push title/body remains content-free and bounded. | FR-004, FR-020, RULE-006 | AC-004, AC-020, AC-023 | Assert the fallback actor label and content exclusions remain unchanged while role-aware category wording varies only by target structure. | `appview/internal/push/payload_test.go` |
| REG-002 | Delivery claiming, fencing, TTL, cancellation, provider retry, and invalid-token cleanup remain unchanged. | RULE-006 | AC-020 | Run existing dispatcher, retry, and sender suites after adding canonical facts. | `appview/internal/push/dispatcher_test.go`, `appview/internal/push/retry_test.go`, `appview/internal/push/firebase_sender_test.go` |
| REG-003 | Eligibility, preferences, coalescing, and preference snapshots remain unchanged. | RULE-006 | AC-020 | Run existing notification policy/lifecycle suites. | `appview/internal/notifications/*_test.go`, `appview/internal/index/notification_*_test.go` |
| REG-004 | Notification list, new count, seen acknowledgement, sound, and badge behavior remain unchanged. | RULE-006 | AC-020 | Run AppView newness/list tests and Flutter list/badge/seen suites. | `appview/internal/api/notifications_test.go`, `appview/internal/api/notification_newness_test.go`, `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/notification_seen_flow_test.dart`, `app/test/notifications/app_shell_notification_badge_test.dart` |
| REG-005 | Known in-app notification rows still open the same profile/post/reply contexts. | RULE-006 | AC-020 | Assert follow, like, repost, mention, quote, and reply row routes, including reply focus. | `app/test/notifications/notifications_page_test.dart` |
| REG-006 | Sign-out clears local routing state and `401` still owns authentication loss. | RULE-006 | AC-010, AC-020 | Run explicit sign-out and interceptor cleanup suites. | `app/test/notifications/services/notification_sign_out_cleanup_test.dart`, `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart` |
| REG-007 | Firebase remains an adapter and diagnostics remain redacted. | NFR-002, NFR-004 | AC-017, AC-019 | Run architecture/import and sentinel-redaction checks without Firebase initialization. | `app/test/notifications/notification_architecture_test.dart`, `app/test/shared/errors/sentry_redaction_test.dart` |
| REG-008 | The pre-launch resolver contract is removed rather than retained in parallel. | FR-001, FR-014, FR-015 | AC-014, AC-015 | Structural scan rejects provider `notificationId`, former resolver route/store, and Flutter resolution-only types/calls. | `app/test/notifications/notification_architecture_test.dart`, `appview/internal/routes/routes_test.go` |
| REG-009 | Push data cannot become an arbitrary external or literal deep-link surface. | FR-003, FR-005 | AC-003, AC-005 | Feed arbitrary URLs, route-like extras, and non-post AT-URIs through parsing/inference and assert safe fallback/ignore behavior. | `app/test/notifications/models/notification_open_event_test.dart` |
| REG-010 | Each provider callback retains existing at-least-once semantics without adding a deduplication store. | FR-013, RULE-006 | AC-013, AC-020 | Assert one outcome per delivered callback and no new receipt/dedupe persistence. | `app/test/notifications/notification_runtime_lifecycle_test.dart`, `app/test/notifications/notification_architecture_test.dart` |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Valid common version 1 contract | `payloadVersion=1`, bounded known `type`, valid `accountSubscriptionId`, and explicit source enum. | AT-001, AT-004, UT-001, IT-003, IT-004 |
| TD-002 | Exact category fact matrix | Follow actor DID; like/repost subject and canonical root post URIs; mention/quote source post URI; reply subject and source post URIs; everything-else no reference. Use distinct sentinels so accidental substitutions are visible. | AT-001, AT-005, UT-002, UT-004, UT-011, IT-001, IT-002, IT-009 |
| TD-003 | Malformed facts with valid binding | Missing fact fields, empty/oversized fact values, invalid DID, arbitrary URL, non-post AT-URI, invalid type characters, and malformed/unsupported versions, all paired with a valid matching `accountSubscriptionId`. | AT-003, UT-001, UT-002, UT-003, UT-005, REG-009 |
| TD-004 | Forward-compatible unknown type | Syntactically valid bounded type such as `projectInvite2` plus destination-shaped extras. | AT-003, UT-002, UT-005 |
| TD-005 | Binding isolation | Current DID with matching, missing/malformed payload, missing local, stale, and different-account `accountSubscriptionId` values; non-binding facts remain valid. | AT-002, UT-001, UT-006, IT-004, MAN-004 |
| TD-006 | Destination error matrix | `404 post_not_found`, `404 profile_not_found`, network error, representative `5xx`, `502 identity_unavailable`, and `401`. | AT-006, AT-007, UT-009, IT-006, IT-007 |
| TD-007 | Durable AppView notifications | One active event per category with distinct actor DID, source URI, subject URI, and active account subscription; optional unrelated canonical references. | UT-011, IT-001, IT-002 |
| TD-008 | Privacy sentinels | Unique strings for notification ID, binding, token, DID, AT-URI, focus URI, payload body, and provider error. | UT-010, IT-010, REG-007 |
| TD-009 | Readiness timeline | Multiple ordered open events across transient, ready, requires-sign-in, and later same-DID sign-in transitions. | AT-008, UT-008, IT-003 |
| TD-010 | Stale pre-cutover payload | Matching binding, old `notificationId` and `type`, but no `payloadVersion` or version 1 facts. | AT-003, UT-005, IT-003 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-013 | AC-013 | Android background and terminated delivery | On a physical Android device with an eligible account, send known-category pushes, open one from background and one from a terminated process, and repeat through a foreground banner. | Each source produces the same intended route exactly once; reply focus is preserved; no resolver wait is observable. |
| MAN-002 | FR-013 | AC-013 | iOS background and terminated delivery | Repeat MAN-001 on a signed/configured physical iOS device with APNs/FCM delivery working. | Each source produces the same intended route exactly once using the same payload contract. |
| MAN-003 | FR-009, RULE-005 | AC-009 | Accepted push becomes stale | Deliver a post/profile notification, then delete, hide, or take down the target before tapping the accepted OS notification. | App opens the intended route, shows permanent-unavailable UI with Back/View notifications, and does not expose stale content or auto-redirect. |
| MAN-004 | FR-006, RULE-003 | AC-006 | Multi-account stale OS notification | Receive a notification for account A, switch/sign in to account B so the current secure binding differs, then tap account A's retained OS notification. | No destination or Notifications fallback route opens under account B; only generic unavailable feedback appears. |
| MAN-005 | NFR-001 | AC-007 | Qualitative cold-start latency and request order | Capture a local debug network trace while opening a valid notification from a terminated app. | Typed route transition begins before the destination AppView request completes, and no request to the removed notification-resolution endpoint appears. No numeric latency threshold is required. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Host tests cannot prove real FCM/APNs background and terminated callback delivery. | FR-013 | Native OS/provider lifecycle behavior requires signed physical-device builds and configured projects. | Complete MAN-001 and MAN-002 before release readiness; not blocking document or local TDD work. |
| GAP-002 | A provider-accepted push cannot be deterministically mutated after acceptance in automated local tests. | FR-009, RULE-005 | The true stale-delivery sequence crosses provider and content lifecycle boundaries. | Automate destination error UI and complete MAN-003 for end-to-end confirmation. |
| GAP-003 | Requirements intentionally define no numeric latency service level. | BR-001, NFR-001 | The objective is removal of one serial request, not a device/network-specific timing promise. | Use UT-007/IT-003 as the release gate and MAN-005 as qualitative confirmation; add privacy-safe timing only in a later approved slice if needed. |
| GAP-004 | Pre-cutover clients are intentionally incompatible with the replacement payload. | FR-015 | The app is not live and compatibility was explicitly rejected. | Keep REG-008 as structural proof of a clean cutover; developers reset/update local builds and data. |

Blocking gaps: None.

## 10. Out Of Scope

- Compatibility tests for old clients, old provider payloads, or preserved development notification data; the app is not live and the cutover is intentionally clean.
- Tests for notification eligibility, preference, coalescing, newness, seen, TTL, sound, badge, or delivery policy changes; those behaviors are protected only as regressions because this slice must not change them.
- PDS-direct content reads or payload-based authorization; all destination content remains an authenticated AppView concern.
- Universal links, Android App Links, iOS Universal Links, a public `craftsky://` scheme, or execution of literal URLs from provider data.
- Encrypted payload capsules, key management, notification receipt deduplication, persisted open queues, or automatic open retries.
- Numeric latency benchmarks or new analytics identifiers.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-16-direct-push-notification-routing/`
- Risk level: High; document review and explicit approval are required before implementation.
- Recommended first failing test for implementation: `UT-001` in `app/test/notifications/models/notification_open_event_test.dart`, because the provider-neutral open-attempt boundary must preserve binding validity independently from fact validity, is shared by foreground, background, and terminated opens, and removes `notificationId` before routing behavior changes.
- Suggested test order for implementation:
  1. `UT-001`-`UT-003`: structured provider-envelope parsing, category fact validation, and typed identifiers.
  2. `UT-006`, `AT-002`: mandatory binding gate before every routing/fallback outcome.
  3. `UT-004`, `UT-005`, `AT-003`: Flutter inference and post-binding fact fallback classification.
  4. `UT-008`, `AT-008`: latest-only readiness and sign-in discard.
  5. `UT-011`, `UT-012`, `IT-001`, `IT-002`: AppView canonical-fact projection and provider payload contract.
  6. `UT-007`, `UT-014`, `IT-003`, `IT-004`, `IT-009`, `AT-001`, `AT-004`, `AT-005`: runtime and typed route integration.
  7. `UT-009`, `AT-006`, `AT-007`, `IT-005`-`IT-007`: destination authorization and error UI.
  8. `UT-013`, `IT-008`, `IT-011`, `REG-008`: resolver removal and generic-row interaction cleanup.
  9. `UT-010`, `UT-015`, `IT-010`, all regressions, then manual checks.
- Commands discovered:
  - From `app/`: `flutter test test/notifications/models/notification_open_event_test.dart`
  - From `app/`: `flutter test test/notifications`
  - From `app/`: `flutter test test/notifications test/feed/pages/post_comment_section_page_test.dart test/profile/profile_page_test.dart test/router`
  - From `app/`: `dart analyze lib/notifications lib/feed/pages/post_thread_page.dart lib/profile test/notifications test/feed/pages/post_comment_section_page_test.dart test/profile/profile_page_test.dart test/router`
  - From `appview/`: `go test ./internal/push ./internal/api ./internal/routes`
  - From repository root: `just app-test`, `just app-analyze`, `just test`, and `git diff --check`
- Blocking gaps: None.
