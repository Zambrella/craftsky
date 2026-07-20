# Acceptance Test Specification: Account Mutes And Blocks

## 1. Test Strategy

This is a high-risk, cross-stack safety feature. The test design treats indexed AppView state as the server enforcement boundary and Flutter as an account-scoped optimistic client that deliberately bridges the public-write Tap convergence window, with the following layers:

- Unit tests define the relationship-policy decision table, precedence, response shaping, lifecycle decisions, cursor rules, client optimistic state, localization, accessibility semantics, and telemetry redaction.
- Real-Postgres integration tests verify migrations, private mute ownership, public block indexing, PDS-confirmed writes with delayed-Tap convergence, eligible-row pagination, profile-row membership lifecycle, notifications, push cancellation, and every affected API/store path.
- Flutter widget, provider, repository, and routing tests verify profile and content actions, confirmations, distinct relationship states, placeholders, temporary reveal, settings lists, retry/rollback, stale-data eviction, and multi-account isolation.
- Acceptance scenarios describe the complete user-visible workflows. They are automated through the existing Go handler/store suites and Flutter widget/provider suites; an implementation may add a small `app/integration_test/` harness if cross-page orchestration cannot be expressed reliably at those established seams.
- Regression tests protect direct muted-content access, allowed cleanup/reporting, non-destructive public records, unrelated aggregate contributions, platform-moderation precedence, and server enforcement for older clients.
- Manual checks are limited to real assistive-technology navigation and one compatible-client/local-PDS interoperability smoke test.

The risk level remains **High**. Test design found no blocking requirement gap, but document review and explicit approval are required before implementation begins.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, IT-028, REG-008 | Acceptance / Integration / Regression | Yes |
| BR-002 | AC-003, AC-004 | IT-005, IT-007, IT-032, MAN-002 | Integration / Manual | Yes, except live smoke |
| BR-003 | AC-005, AC-006 | AT-002, IT-003, UT-012 | Acceptance / Integration / Unit | Yes |
| BR-004 | AC-051 | AT-015, IT-024, UT-014 | Acceptance / Integration / Unit | Yes |
| FR-001 | AC-007 | IT-001, UT-002 | Integration / Unit | Yes |
| FR-002 | AC-008, AC-009 | IT-002, IT-035 | Integration | Yes |
| FR-003 | AC-010, AC-011 | IT-005, IT-006, UT-010 | Integration / Unit | Yes |
| FR-004 | AC-012 | IT-007, IT-035, IT-036, UT-010 | Integration / Unit | Yes |
| FR-005 | AC-013, AC-059 | IT-008 | Integration | Yes |
| FR-006 | AC-014 | IT-006, IT-009, UT-010 | Integration / Unit | Yes |
| FR-007 | AC-015 | IT-010, UT-003 | Integration / Unit | Yes |
| FR-008 | AC-016 | AT-011, IT-011, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-009 | AC-017 | IT-012 | Integration | Yes |
| FR-010 | AC-018 | AT-004, UT-005 | Acceptance / Unit | Yes |
| FR-011 | AC-019 | AT-005, UT-006, IT-033 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-020 | AT-003, REG-001 | Acceptance / Regression | Yes |
| FR-013 | AC-021, AC-022 | AT-006, IT-018, IT-019 | Acceptance / Integration | Yes |
| FR-014 | AC-023, AC-024 | AT-007, AT-014, IT-013, IT-014 | Acceptance / Integration | Yes |
| FR-015 | AC-025 | AT-008, IT-015, IT-033 | Acceptance / Integration | Yes |
| FR-016 | AC-026 | AT-009, IT-010, UT-004 | Acceptance / Integration / Unit | Yes |
| FR-017 | AC-027 | IT-016, UT-008 | Integration / Unit | Yes |
| FR-018 | AC-028 | IT-017, UT-008, REG-004 | Integration / Unit / Regression | Yes |
| FR-019 | AC-029, AC-030 | AT-006, IT-018, IT-019 | Acceptance / Integration | Yes |
| FR-020 | AC-031, AC-052 | IT-020, IT-025, REG-002 | Integration / Regression | Yes |
| FR-021 | AC-032, AC-060 | IT-021, IT-022 | Integration | Yes |
| FR-022 | AC-033 | AT-001 | Acceptance | Yes |
| FR-023 | AC-034 | AT-009 | Acceptance | Yes |
| FR-024 | AC-035 | AT-011 | Acceptance | Yes |
| FR-025 | AC-036, AC-037 | AT-012, UT-011, IT-009, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| FR-026 | AC-038 | IT-023, REG-007 | Integration / Regression | Yes |
| FR-027 | AC-054, AC-055 | AT-010 | Acceptance | Yes |
| FR-028 | AC-051, AC-052 | AT-015, IT-007, IT-024, IT-025, UT-014 | Acceptance / Integration / Unit | Yes |
| FR-029 | AC-052, AC-053 | IT-025, IT-026, IT-035, UT-016 | Integration / Unit | Yes |
| FR-030 | AC-056 | IT-027 | Integration | Yes |
| FR-031 | AC-057 | AT-006, IT-020, UT-007 | Acceptance / Integration / Unit | Yes |
| NFR-001 | AC-006, AC-039 | IT-003, UT-011, UT-012, REG-009 | Integration / Unit / Regression | Yes |
| NFR-002 | AC-040 | UT-001, IT-028, REG-008 | Unit / Integration / Regression | Yes |
| NFR-003 | AC-041 | IT-029, UT-009 | Integration / Unit | Yes |
| NFR-004 | AC-042 | IT-030, IT-035, GAP-003 | Integration | Partial; no latency SLA |
| NFR-005 | AC-043 | AT-013, UT-013, MAN-001 | Acceptance / Unit / Manual | Yes, plus device smoke |
| NFR-006 | AC-044 | IT-031, UT-012 | Integration / Unit | Yes |
| RULE-001 | AC-005, AC-045 | AT-002, UT-001, REG-001 | Acceptance / Unit / Regression | Yes |
| RULE-002 | AC-003, AC-046 | IT-005, IT-032, UT-001 | Integration / Unit | Yes |
| RULE-003 | AC-047 | UT-001, UT-002, REG-003 | Unit / Regression | Yes |
| RULE-004 | AC-007 | IT-001, UT-002 | Integration / Unit | Yes |
| RULE-005 | AC-031, AC-048 | IT-020, REG-002 | Integration / Regression | Yes |
| RULE-006 | AC-019, AC-025, AC-049 | AT-005, AT-008, AT-014, IT-033, UT-006, UT-015 | Acceptance / Integration / Unit | Yes |
| RULE-007 | AC-050 | IT-004, UT-008 | Integration / Unit | Yes |
| RULE-008 | AC-058 | IT-034, REG-006 | Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Manage relationships from an eligible visitor profile

Requirement IDs: BR-001, FR-022  
Acceptance Criteria: AC-001, AC-033  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/widgets/profile_actions_test.dart`, `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Profile mute and block controls
  Scenario: An eligible visitor manages another member
    Given Alice is signed in and Bob is a current unblocked Craftsky member
    When Alice opens Bob's profile
    Then Follow and the current Mute or Unmute icon remain visible actions
    And Share is not a standalone action
    When Alice opens More on a compact screen
    Then it opens as a bottom sheet offering Share, the current Block or Unblock action, and Report
    When Alice opens More on a larger screen
    Then it opens as an anchored popup offering the same actions
    When Alice chooses the visible Mute action
    Then the mute applies immediately with localized feedback
    When Alice chooses Block or Unblock
    Then a localized confirmation explains the consequences and that a block is public
    And the confirmed action is backed by the authenticated AppView operation
```

### AT-002: A mute is private and one-way

Requirement IDs: BR-003, RULE-001  
Acceptance Criteria: AC-005, AC-045  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/profile_test.go`, `appview/internal/api/relationship_policy_test.go`

```gherkin
Feature: Private actor mute
  Scenario: Alice mutes Bob without restricting Bob
    Given Alice and Bob are current Craftsky members
    When Alice mutes Bob
    Then Alice's unsolicited views and deliveries apply the mute policy
    But Bob's own profile and API responses do not reveal that Alice muted him
    And both accounts may still follow, like, repost, reply, quote, mention, and directly view one another
```

### AT-003: Direct muted content remains available

Requirement IDs: FR-012, RULE-001  
Acceptance Criteria: AC-020, AC-045  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Intentional muted-content access
  Scenario: Alice directly opens Bob after muting him
    Given Alice muted Bob
    When Alice directly opens Bob's profile, profile content tab, or post
    Then the requested content remains viewable
    And Bob is visibly marked as muted for Alice
    And Alice is not required to unmute Bob
```

### AT-004: A muted reply branch has temporary reveal

Requirement IDs: FR-010, FR-025  
Acceptance Criteria: AC-018, AC-037  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`, `app/test/feed/providers/post_comment_section_provider_test.dart`

```gherkin
Feature: Muted reply branch
  Scenario: Alice reveals a muted parent and its descendants
    Given muted Bob authored a reply with unmuted descendants
    When Alice opens the thread
    Then the complete Bob branch is collapsed behind localized muted-account copy
    When Alice explicitly reveals the branch
    Then that branch is visible as a unit only in the current thread view
    And refresh, navigation away, or account switch collapses it again
    And no other signed-in account inherits the reveal
```

### AT-005: A muted quote is revealable without hiding the quoting post

Requirement IDs: FR-011, RULE-006  
Acceptance Criteria: AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/feed/models/post_test.dart`

```gherkin
Feature: Muted quote preview
  Scenario: Carol quotes muted Bob
    Given Alice muted Bob but did not mute Carol
    When Alice sees Carol's quoting post
    Then Carol's post remains visible
    And Bob's quote preview is replaced by a localized revealable muted-content control
    And revealing it does not unmute Bob globally
```

### AT-006: Relationship changes update notifications without replaying push

Requirement IDs: FR-013, FR-019, FR-031  
Acceptance Criteria: AC-021, AC-022, AC-029, AC-030, AC-057  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/providers/notification_new_count_provider_test.dart`, `appview/internal/notifications/eligibility_test.go`, `appview/internal/push/dispatcher_test.go`

```gherkin
Feature: Relationship-aware notification delivery
  Scenario Outline: Alice suppresses Bob's notifications
    Given Bob has an eligible retained notification for Alice and an unsent delivery
    When Alice <relationship> Bob
    Then the notification is absent from Alice's list, new count, and badge
    And no pending or leased-unsent delivery is successfully sent
    When Alice removes the relationship while the event is still retained
    Then the eligible event may reappear once
    But no already-sent or cancelled push is replayed

    Examples:
      | relationship |
      | mutes        |
      | blocks       |
```

### AT-007: A block symmetrically removes authored content

Requirement IDs: FR-014, RULE-002  
Acceptance Criteria: AC-023, AC-046  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`, `app/test/search/search_page_test.dart`, `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Symmetric block visibility
  Scenario Outline: Either direction of block is enforced
    Given <blocker> blocks <subject>
    And Tap has indexed the block
    When Alice and Bob load timelines, discovery, search, profile content, projects, comments, and threads
    Then neither account receives the other's authored or attributed content
    And the outcome is identical regardless of block direction

    Examples:
      | blocker | subject |
      | Alice   | Bob     |
      | Bob     | Alice   |
```

### AT-008: Blocked embedded content is never revealable

Requirement IDs: FR-015, RULE-006  
Acceptance Criteria: AC-025, AC-049  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/router/notification_open_routing_test.dart`

```gherkin
Feature: Block-safe indirect content
  Scenario: Carol quotes or reposts blocked Bob
    Given Alice and Bob are blocked in either direction and Carol blocks neither
    And Tap has indexed the block
    When Alice sees Carol's quote or repost through a list, stale cache, or deep link
    Then Carol's quoting post may remain visible
    But Bob's quote is an unrevealable generic unavailable placeholder
    And a straight repost of Bob is omitted
    And opening the reference rechecks current server policy
```

### AT-009: Blocked profiles expose only valid identity and actions

Requirement IDs: FR-016, FR-023  
Acceptance Criteria: AC-026, AC-034  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_actions_test.dart`

```gherkin
Feature: Blocked profile states
  Scenario Outline: Flutter renders each relationship direction
    Given Alice's relationship to Bob is <state>
    When Alice opens Bob's profile
    Then Flutter shows the distinct localized <state> annotation
    And only policy-valid identity, Report, owned Unblock, reciprocal Block, or Unmute actions are available
    And bio, metrics, mutuals, content tabs, follow controls, and activity are absent when a block exists

    Examples:
      | state          |
      | muted          |
      | blocked by you |
      | has blocked you |
```

### AT-010: Every non-self content menu exposes safety controls

Requirement IDs: FR-027  
Acceptance Criteria: AC-054, AC-055  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/projects/widgets/project_card_test.dart`, `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Content-level relationship actions
  Scenario Outline: Alice acts from a content More menu
    Given Bob authored a non-self <contentType>
    When Alice opens its More menu
    Then current-state Mute or Unmute author, Block or Unblock author, and Report post are present
    When Alice successfully mutes Bob
    Then Bob-authored list and discovery items disappear immediately
    And a directly viewed Bob root remains visible with muted state
    And other Bob reply branches collapse as units

    Examples:
      | contentType  |
      | post         |
      | project post |
      | comment      |
      | reply        |
```

### AT-011: Settings lists paginate and reverse current relationships

Requirement IDs: FR-008, FR-024  
Acceptance Criteria: AC-016, AC-035  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/settings/relationship_list_page_test.dart`, `app/test/settings/settings_page_test.dart`

```gherkin
Feature: Muted and blocked account management
  Scenario Outline: Alice manages a relationship list
    Given Alice's <list> contains more than one page with current and former members
    When Alice opens the Settings screen
    Then loading, empty, error, retry, list, and pagination states use localized copy
    And current members appear once in stable order
    And former members and tombstones are absent
    When Alice reverses a relationship successfully
    Then its row is removed and allowed profile navigation still works

    Examples:
      | list             |
      | Muted accounts   |
      | Blocked accounts |
```

### AT-012: Optimistic mutation remains bound to the initiating account

Requirement IDs: FR-025, NFR-001  
Acceptance Criteria: AC-036, AC-037  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/providers/profile_relationship_provider_test.dart`, `app/test/auth/providers/account_boundary_provider_test.dart`, `app/test/router/account_switch_routing_test.dart`

```gherkin
Feature: Account-scoped optimistic relationship state
  Scenario: Alice switches accounts during a mute or block request
    Given Alice and Carol are signed in on one device
    And Alice starts a relationship mutation against Bob
    When duplicate taps occur and the active account switches to Carol before completion
    Then the request is applied at most once to Alice's keyed state
    And success refreshes Alice's affected data
    And a stale pre-Tap refresh cannot reverse Alice's PDS-confirmed block state
    Or failure rolls Alice back with feedback
    And Carol's providers, caches, lists, notification counts, and UI never change
```

### AT-013: Safety controls are localized and accessible

Requirement IDs: NFR-005  
Acceptance Criteria: AC-043  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/l10n/mute_block_l10n_test.dart`, `app/test/profile/widgets/profile_actions_test.dart`, `app/test/feed/widgets/post_card_test.dart`, `app/test/settings/relationship_list_page_test.dart`

```gherkin
Feature: Accessible relationship controls
  Scenario: A user navigates mute and block UI with assistive input
    Given any supported locale and an assistive navigation mode
    When profile actions, content actions, placeholders, confirmations, and settings lists are traversed
    Then every visible string is localized
    And controls have unambiguous labels, roles, tooltips, focus order, and enabled state
    And destructive block actions are visually and semantically distinguishable
```

### AT-014: A stale direct link cannot reveal blocked content

Requirement IDs: FR-014, RULE-006  
Acceptance Criteria: AC-024, AC-049  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`, `app/test/router/notification_open_routing_test.dart`

```gherkin
Feature: Blocked deep-link protection
  Scenario: Alice opens a cached link to Bob's blocked post
    Given Alice's device cached Bob's post before a block became effective
    When Alice opens the old post or notification deep link
    Then Flutter revalidates through the AppView
    And no cached text, media, author detail, or hydrated notification subject is displayed
    And only a generic blocked or unavailable state is shown
```

### AT-015: Non-members are indistinguishable from unknown accounts

Requirement IDs: BR-004, FR-028  
Acceptance Criteria: AC-051  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/search/search_page_test.dart`, `app/test/shared/rich_text/facet_autocomplete_editor_test.dart`

```gherkin
Feature: Craftsky membership boundary
  Scenario: Alice targets a resolvable account without a Craftsky profile
    Given the account's DID and handle resolve on atproto but it is not a current Craftsky member
    When Alice uses profile, search, suggestion, graph, report, relationship, or directed-interaction UI
    Then collection surfaces omit the account
    And direct surfaces show the same standard not-found result as an unknown account
    And no external identity detail or relationship record is exposed
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | NFR-002, RULE-001, RULE-002, RULE-003 | AC-040, AC-045, AC-046, AC-047 | Table-test the shared relationship-policy decision across no relation, outbound mute, outbound block, inbound block, mutual block, mute plus block, hide, and takedown. | Viewer DID, author/subject DIDs, mute/block directions, platform moderation state, operation/surface. | One deterministic decision is returned: platform hide/takedown wins, otherwise block wins over mute; mute remains one-way and block remains symmetric. | `appview/internal/relationships/policy_test.go` |
| UT-002 | FR-001, RULE-003, RULE-004 | AC-007, AC-047 | Validate and canonicalize target identity and reject invalid/self targets before policy or persistence. | Handle, DID, malformed identifier, caller DID, membership result. | Valid member handle canonicalizes once to DID; invalid/self fails without state; non-member uses not-found; stricter moderation remains authoritative. | `appview/internal/relationships/target_test.go` |
| UT-003 | FR-007 | AC-015 | Shape viewer-relative profile fields for every relationship direction. | No relation, muted, blocking, blockedBy, mutual block; Alice/Bob/Carol viewers. | Only the requesting viewer receives correct `muted`, `blocking`, and `blockedBy`; Alice's mute never appears for Bob or Carol. | `appview/internal/api/profile_response_test.go` |
| UT-004 | FR-016 | AC-026 | Shape the minimal blocked-profile shell. | Full profile row plus each block direction. | Response retains allowed identity/annotation/report or owned-unblock state and strips bio, metrics, mutuals, tabs, follow state, and activity. | `appview/internal/api/profile_response_test.go` |
| UT-005 | FR-010, FR-025 | AC-018, AC-037 | Shape a muted reply and all descendants as one branch and model temporary reveal by account/thread instance. | Reply tree with muted parent and mixed descendants; refresh/navigation/account-switch events. | Initial single placeholder; reveal exposes only that branch; every reset event clears reveal; other accounts remain collapsed. | `appview/internal/api/post_response_test.go`, `app/test/feed/models/post_comment_section_test.dart` |
| UT-006 | FR-011, FR-015, RULE-006 | AC-019, AC-025, AC-049 | Apply quote/repost/embed policy including stale preview eviction. | Quoter policy, quoted-author mute/block state, straight repost, cached preview. | Muted quote is revealable; blocked quote is unrevealable; blocked straight repost is omitted; protected cached payload is discarded. | `appview/internal/api/post_response_test.go`, `app/test/feed/models/post_test.dart` |
| UT-007 | FR-013, FR-019, FR-031 | AC-021, AC-029, AC-057 | Decide notification eligibility, newness, badge, outbox creation, and restoration. | Notification categories, actor/recipient directions, relationship effective times, retained/expired/cancelled/sent states. | Muted actor or blocked pair is excluded; eligible retained history may restore once; no sent/cancelled push is recreated. | `appview/internal/notifications/eligibility_test.go`, `appview/internal/notifications/newness_test.go` |
| UT-008 | FR-017, FR-018, RULE-007 | AC-027, AC-028, AC-050 | Authorize interaction, cleanup, reporting, reciprocal block, and relationship ownership operations. | Operation matrix across block directions and resource ownership. | New directed interactions deny with `interaction_blocked`; owned deletes, reports, and reciprocal block remain allowed; foreign mute/block mutation denies. | `appview/internal/relationships/authorization_test.go` |
| UT-009 | FR-008, FR-026, NFR-003 | AC-016, AC-038, AC-041 | Validate list limits and encode/decode opaque relationship cursors. | Missing/0/50/100/101 limits, valid/tampered cursor, eligible sort keys. | Default 50, maximum 100, stable opaque round-trip, invalid cursor standard error, no protected identifier exposed. | `appview/internal/api/relationship_request_test.go`, `appview/internal/api/envelope/cursor_test.go` |
| UT-010 | FR-003, FR-004, FR-006 | AC-010, AC-011, AC-012, AC-014 | Decide canonical indexed block record and ordered Tap reconciliation. | Duplicate pair URIs, create/update/delete CID order, duplicate replay, and missing-index lookup before unblock. | Craftsky selects/deletes its owned active record deterministically; repository-ordered events and duplicate replay converge idempotently. | `appview/internal/index/bluesky_block_test.go` |
| UT-011 | FR-025, NFR-001 | AC-036, AC-037 | Exercise Flutter optimistic mutation state with account key and session generation. | Success, failure, duplicate tap, account switch, late completion, and stale pre-Tap refresh callbacks. | One in-flight mutation per account/subject/action; rollback and feedback are account-local; confirmed overlays survive stale refreshes; late results cannot mutate the newly active account. | `app/test/profile/providers/profile_relationship_provider_test.dart` |
| UT-012 | BR-003, NFR-001, NFR-006 | AC-006, AC-039, AC-044 | Verify diagnostic redaction and bounded dimensions for mute/block operations. | Logs, errors, breadcrumbs, traces, metric attributes containing target DID/handle/pair/rkey/post URI. | Private mute pair and target identifiers are absent; public block targets are not unbounded labels; operation/result/stage/error class remain observable. | `appview/internal/observability/relationship_test.go`, `app/test/shared/errors/sentry_redaction_test.dart` |
| UT-013 | NFR-005 | AC-043 | Verify localization key completeness and widget semantics for every new control, state, placeholder, confirmation, list state, and feedback message. | All supported locales; enabled/disabled/destructive/reveal states. | No fallback/raw keys; stable accessible labels, roles, hints, tooltips, focus order, and destructive semantics. | `app/test/l10n/mute_block_l10n_test.dart`, relevant widget tests |
| UT-014 | BR-004, FR-028 | AC-051, AC-052 | Apply the canonical current-membership predicate to direct and collection decisions. | Current member, absent former member, resolvable never-member, unknown DID. | Current member eligible; all others share not-found or omission behavior without external hydration; retained records are not destroyed. | `appview/internal/relationships/membership_test.go` |
| UT-015 | RULE-006 | AC-049 | Revalidate every indirect reference/open destination against current policy. | Quote subject, notification subject, push route, stale post cache, later-page cursor item. | Current server decision replaces cached state and protected content is never materialized into the response/UI. | `appview/internal/relationships/reference_test.go`, `app/test/router/notification_open_routing_test.dart` |
| UT-016 | FR-029 | AC-052, AC-053 | Distinguish subject departure, owner permanent removal, sign-out, device removal, and account switch lifecycle actions. | Lifecycle event plus owner/subject DID role. | Subject departure hides but retains; owner permanent removal deletes owned private mutes; device/session/account events do not delete server mute rows. | `appview/internal/relationships/lifecycle_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, RULE-004 | AC-007 | Target resolution and rejection across mute/block handlers. | Authenticated Alice; current Bob; resolvable non-member, unknown, invalid, and Alice identifiers. | POST/DELETE mute and block routes by handle and DID. | Bob canonicalizes to DID; invalid/self causes no persistence/PDS call; unknown and non-member return identical `404 profile_not_found`. | `appview/internal/api/relationship_test.go` |
| IT-002 | FR-002 | AC-008, AC-009 | Private mute persistence is owner-scoped, immediate, unique, and idempotent. | Real Postgres with Alice, Bob, and Carol sessions. | Mute/unmute Bob repeatedly as Alice and query immediately as Alice/Carol. | Exactly one `(viewer_did, subject_did)` row at most; success is visible before response; unmute converges; Carol is unchanged. | `appview/internal/api/mute_store_test.go`, `appview/internal/api/relationship_test.go` |
| IT-003 | BR-003, NFR-001 | AC-005, AC-006 | Private mute cannot be enumerated or inferred by another viewer/account/device. | Alice mutes Bob; Bob and Carol have authenticated sessions on same and different device IDs. | Read all profile/summary/list shapes and inspect serialized caches/diagnostics for each viewer. | Only Alice receives her mute-derived state/list; Bob and Carol see ordinary state; target/pair is absent from shared state and diagnostics. | `appview/internal/api/profile_test.go`, `appview/internal/api/relationship_list_test.go`, Flutter provider tests |
| IT-004 | RULE-007 | AC-050 | Relationship ownership and inbound-block authorization. | Alice owns mute and outbound block; Bob has inbound view; Carol is unrelated. | Attempt foreign mute enumeration/mutation, foreign block deletion, and enforcement reads. | Foreign operations deny without state change; only Alice deletes her record; inbound block remains readable solely for policy enforcement. | `appview/internal/api/relationship_test.go`, `appview/internal/api/relationship_list_test.go` |
| IT-005 | BR-002, FR-003, RULE-002 | AC-003, AC-010, AC-046 | Block creates one canonical public PDS record before success without a synchronous projection. | Fake/recording PDS for Alice; current Bob; Tap delivery held; block store spy. | Alice blocks Bob and inspects the response before Tap delivery. | PDS receives one `app.bsky.graph.block` with Bob DID and valid `createdAt`; response waits for PDS and returns URI/CID/rkey plus blocking state; no local block upsert occurs. | `appview/internal/api/block_test.go` |
| IT-006 | FR-003, FR-006 | AC-011, AC-014 | Block/unblock retry and rapid pre-index convergence. | One or more matching PDS records; Tap delivery held; local index initially missing. | Retry block, unblock before the create event, retry unblock, then deliver repository-ordered create/delete events with duplicates. | Craftsky creates no additional record while any match exists, locates and deletes every matching owned rkey when the local index is missing, and the indexed final state converges after Tap. | `appview/internal/api/block_test.go`, `appview/internal/index/bluesky_block_test.go` |
| IT-007 | BR-002, FR-004, FR-028 | AC-004, AC-012, AC-051 | Index external compatible block lifecycle and replay without surfacing non-members. | Real Postgres; valid and invalid external `app.bsky.graph.block` Tap events including duplicate pair URIs/CIDs and a current-member-owned record targeting a non-member. | Deliver repository-ordered create/update/delete events with duplicate replay, then query user-facing surfaces for the absent subject. | Valid external records are retained and converge without Craftsky recreation even when the subject is absent; duplicate replay is idempotent; the non-member remains omitted and unmanageable; malformed events fail safely. | `appview/internal/index/bluesky_block_test.go` |
| IT-008 | FR-005 | AC-013, AC-059 | Profile-row membership and block backfill converge across both block-owner directions and process restart. | Fixture A: the joining repository owns an outbound block targeting current Alice. Fixture B: current Alice already owns a retained indexed block targeting the joining DID. Tap backfill can be held and resumed after service recreation. | Index the joining profile, interrupt and restart repository backfill, then resume it while reading both relationship directions. | The profile row alone establishes membership; Fixture B applies immediately; Fixture A may be temporarily absent but becomes enforced when Tap resumes; no activation/readiness row exists. | `appview/internal/index/bluesky_backfiller_test.go`, `appview/internal/auth/initialize_profile_test.go` |
| IT-009 | FR-006, FR-025 | AC-014, AC-036 | PDS-confirmed response and Flutter overlay survive Tap lag. | PDS success; local block-store spy; delayed Tap; stale server read returned to the client provider. | Block, observe the API response and Flutter state, issue a stale refresh, then deliver Tap. | API performs no local projection; Flutter keeps the confirmed blocked state across the stale refresh; server policy changes only after Tap and then matches the client. | `appview/internal/api/block_test.go`, `app/test/profile/providers/profile_relationship_provider_test.dart` |
| IT-010 | FR-007, FR-016 | AC-015, AC-026 | Profile and summary shaping for no relation, mute, outbound, inbound, and mutual block. | Alice/Bob/Carol profiles with full metadata and all relationship states. | Fetch full profiles and summaries as each viewer. | Viewer flags are correct and private; blocked pair receives only minimal shell; Carol receives full eligible profile without Alice's mute state. | `appview/internal/api/profile_test.go`, `appview/internal/api/profile_response_test.go` |
| IT-011 | FR-008 | AC-016 | Authenticated mute/block list pagination and isolation. | More than 110 relationships with equal timestamps, handle changes, current/former subjects, duplicate external block pairs. | Traverse defaults, limit 100, and all cursors as Alice; attempt as Bob. | Stable DID-based rows appear once; eligible pages fill; current handle is displayed; former members absent; cursors opaque; Bob cannot enumerate Alice's list. | `appview/internal/api/relationship_list_test.go`, `appview/internal/api/relationship_store_test.go` |
| IT-012 | FR-009 | AC-017 | Muted top-level content and repost activity are filtered at query time. | Dense mixed authorship fixture across timeline, post/project/hashtag search, project discovery, straight reposts, and reposts of Bob's content. | Alice mutes Bob and traverses multiple pages on every surface. | Bob-authored and Bob-attributed top-level items are absent; eligible later rows fill pages; direct Bob/profile-tab paths are unaffected. | `appview/internal/api/timeline_store_test.go`, `appview/internal/api/search_store_test.go`, `appview/internal/api/profile_store_test.go` |
| IT-013 | FR-014 | AC-023 | Blocked authored/attributed content is symmetrically filtered on every read inventory surface. | Alice outbound, Bob outbound, and mutual-block fixtures with ordinary/project posts, comments, replies, reposts, and profile tabs. | Read timeline, discovery, all search modes, project lists, profile content, comments, and threads as both users. | Other-party content is absent for every direction and surface; no client flag is required for enforcement. | Existing API store/handler suites plus `appview/internal/api/relationship_policy_inventory_test.go` |
| IT-014 | FR-014 | AC-024 | Direct blocked-content response is data-minimal. | Cached/indexed Bob post with text, facets, media, quote, counts, and viewer state; block in either direction. | Fetch direct post/comment/thread endpoint and stale deep-link destination. | Stable generic blocked/unavailable response contains no text, media, protected author detail, quote payload, or viewer metric detail. | `appview/internal/api/post_test.go`, `appview/internal/api/post_response_test.go`, Flutter thread tests |
| IT-015 | FR-015 | AC-025 | Intermediary quote/repost cannot bypass block. | Carol quotes and reposts Bob; Alice and Bob blocked; Carol unblocked. | Load all surfaces as Alice. | Carol quote stays only where otherwise eligible with unrevealable placeholder; straight repost omitted; no Bob payload survives hydration. | `appview/internal/api/post_store_test.go`, `appview/internal/api/timeline_store_test.go` |
| IT-016 | FR-017 | AC-027 | Every directed interaction write fails before PDS mutation across either block direction. | Recording PDS client; existing block Alice→Bob and Bob→Alice cases. | Attempt follow, like, repost, reply, quote/embed, and mention from each party. | Each returns standard `interaction_blocked`; PDS create/update counters remain zero; no optimistic interaction row/event is created. | `appview/internal/api/relationship_interaction_test.go` plus affected handler tests |
| IT-017 | FR-018 | AC-028 | Cleanup, reporting, and reciprocal block remain available. | Existing owned follow/like/repost/content plus one-direction block. | Delete each owned record/content, report profile/addressable content, and create reciprocal block. | Valid cleanup/report/PDS block calls succeed without exposing hidden content; deleting foreign resources still denies. | Affected API handler tests and `appview/internal/api/relationship_interaction_test.go` |
| IT-018 | FR-013, FR-019 | AC-021, AC-029 | Notification ingestion and outbox creation enforce mute/block policy. | Activity categories: follow, like, repost, reply, mention, quote; mute and both block directions effective before ingestion. | Ingest events and compute list/newness/badge/outbox. | Muted actor events and either-direction blocked-pair events are retained only if policy allows audit retention, but never eligible/listed/counted and create no push delivery. | `appview/internal/index/notification_ingestion_test.go`, `appview/internal/notifications/eligibility_test.go`, `appview/internal/push/dispatcher_test.go` |
| IT-019 | FR-013, FR-019 | AC-022, AC-030 | Effective relationship cancels pending/retry/leased-unsent deliveries race-safely. | Listed notifications and outbox rows in pending, retry, leased-before-send, sending, and provider-sent states. | Mute/block becomes effective concurrently with dispatcher lease/final check. | List/newness/badge update immediately; all not-yet-provider-sent rows become unsendable/cancelled; only already-provider-sent delivery is outside retraction guarantee. | `appview/internal/push/dispatcher_test.go`, `appview/internal/api/durable_notification_store_test.go` |
| IT-020 | FR-020, FR-031, RULE-005 | AC-031, AC-048, AC-057 | Block/unblock preserves source records and restores eligible history exactly once. | Pre-existing follow, content, interactions, aggregate rows, retained notifications, sent/cancelled push records. | Block then unblock, and separately mute then unmute, before retention expiry. | Public/source records never delete; eligible views/follow state/history restore; no duplicate notification and no push replay; follow graph remains unchanged. | `appview/internal/api/relationship_restore_test.go`, `appview/internal/notifications/eligibility_test.go` |
| IT-021 | FR-021 | AC-032 | Follow state and graph lists are hidden across a block without deleting follow rows. | Alice follows Bob; Bob follows Alice; mutual followers; block in each direction. | Fetch profile viewer state, follower/following/mutual lists and counts as both parties and Carol. | Pair entries and follow affordance are suppressed only for protected pair; underlying rows remain; eligible Carol views preserve public graph contributions. | `appview/internal/api/profile_store_test.go`, `appview/internal/api/follow_store_test.go` |
| IT-022 | FR-021 | AC-060 | Actor search omits blocked account except exact-handle management lookup. | Indexed Alice/Bob profiles with block in each direction. | Ordinary prefix/text search and exact-handle lookup as each party. | Ordinary results omit the other account; exact handle returns only minimum annotated blocked shell needed to manage/report relationship. | `appview/internal/api/search_store_test.go`, `appview/internal/api/search_response_test.go` |
| IT-023 | FR-026 | AC-038 | New route contract matrix follows `/v1/` architecture. | Router with real middleware and fakes for each mute/block/list handler. | Exercise missing/invalid token, missing device ID, bad input, limit/cursor, rate-limit class, PDS/store error, and success. | Status codes, camelCase JSON, `{error,message,requestId}`, auth/device/rate classes, and opaque cursor conventions match existing routes. | `appview/internal/routes/routes_test.go`, `appview/internal/api/relationship_test.go` |
| IT-024 | BR-004, FR-028 | AC-051 | Uniform non-member exclusion covers the complete account-target inventory. | Resolvable non-member and unknown DID/handle with public records in storage. | Exercise profile, search, mention suggestions/resolution, graph lists/counts, relationship lists, reports, follow/mute/block, and every directed interaction. | Direct targets return indistinguishable `404 profile_not_found`; collections omit; no profile hydration or new public/private relationship/PDS write occurs. | `appview/internal/api/membership_policy_inventory_test.go` |
| IT-025 | FR-020, FR-028, FR-029 | AC-052 | Subject leave/rejoin hides, retains, and restores relationships through profile membership plus ordinary Tap convergence. | Alice has follow, Alice-owned block, Bob-owned block, mute, and content interactions naming Bob; Bob's profile row is removed then re-added while joining-owned backfill is held. | Read/manage during absence; re-add the same DID; read before and after Tap resumes. | No Bob row/count/tombstone/action appears while absent and no underlying relationship is deleted; retained indexed state reappears with the profile row; joining-owned public state converges after Tap without a separate activation state. | `appview/internal/auth/initialize_profile_test.go`, relationship store/API tests |
| IT-026 | FR-029 | AC-053 | Private mute owner lifecycle cleanup is correctly scoped. | Alice owns mutes; Bob is only a subject; multiple device sessions and another signed-in account exist. | Sign out Alice, remove one device, switch account, remove Bob membership, then permanently remove Alice membership. | Session/device/switch and subject removal retain rows; permanent Alice membership removal deletes all and only Alice-owned private mutes; public PDS records remain. | `appview/internal/auth/handlers_test.go`, `appview/internal/api/mute_store_test.go` |
| IT-027 | FR-030 | AC-056 | Third-party rendering cannot connect a blocked pair. | Alice blocks Bob; Carol blocks neither; reply chains, mentions, quotes/embeds, and notification/reference records connect Alice and Bob. | Carol loads each thread/read surface. | Block-violating reference is omitted or generically unavailable according to surface; neither protected party's content is exposed through the other. | `appview/internal/api/post_store_test.go`, `appview/internal/api/post_response_test.go`, notification hydration tests |
| IT-028 | BR-001, NFR-002 | AC-002, AC-040 | Affected-path inventory proves shared server policy enforcement. | Enumerated read/write/notification/push stores and handlers with a spy relationship policy. | Execute one allowed and one denied/protected case per inventory entry. | Every path invokes the shared decision/predicate with viewer/subject context; tests fail when any route relies only on Flutter filtering. | `appview/internal/api/relationship_policy_inventory_test.go`, package import-boundary test |
| IT-029 | NFR-003 | AC-041 | Dense protected-row pagination has no short pages, skips, duplicates, or cursor leakage. | At least three pages with hidden rows before, between, and after eligible rows for each paginated feed/list/search. | Traverse forward pages at small limits and decode only via server. | Pages fill from eligible rows where available; union equals eligible set once; order stable; cursor is opaque and discloses no hidden identity. | `appview/internal/api/relationship_pagination_test.go` and affected store tests |
| IT-030 | NFR-004 | AC-042 | Relationship filtering uses bidirectional indexes and bounded store calls. | Representative feed/search/thread/list fixtures at small and large sizes; query instrumentation. | Run `EXPLAIN`/query-plan assertions and count store/DB calls. | Blocker→subject and subject→blocker indexes are used where applicable; no per-item relationship query; call count remains bounded as page size grows. | `appview/internal/api/relationship_query_plan_test.go`, `appview/internal/db/mutes_blocks_migration_test.go` |
| IT-031 | NFR-006 | AC-044 | Operational signals cover failures and denials without identifier leakage. | Captured logs/metrics/traces for index lag, malformed event, PDS/store failure, `interaction_blocked`, notification suppression, and push cancellation. | Trigger each outcome and inspect emitted fields/labels. | Bounded operation/result/stage/error class and latency/lag are present; target DID/handle/pair/post URI/rkey are absent as labels and routine success fields. | `appview/internal/observability/relationship_integration_test.go` |
| IT-032 | RULE-002 | AC-003, AC-046 | Mutual blocks remain effective until the final direction is removed. | Alice→Bob and Bob→Alice public records coexist. | Remove Alice's block, test both parties, then remove Bob's block. | After first removal all symmetric restrictions remain due to inbound block; only final removal restores otherwise-eligible state; both PDS records remain independently owned. | `appview/internal/api/block_test.go`, `appview/internal/index/bluesky_block_test.go` |
| IT-033 | RULE-006, FR-011, FR-015 | AC-019, AC-025, AC-049 | All indirect hydration paths recheck policy after cache/state changes. | Previously hydrated quote, repost, notification subject, push deep link, and later page; relationship changes after caching. | Refresh/open each reference under mute and block. | Muted content becomes revealable placeholder where specified; blocked content becomes unrevealable/omitted; no stale protected fields survive. | Affected API response/store and Flutter routing tests |
| IT-034 | RULE-008 | AC-058 | Policy and membership do not rewrite unrelated public aggregates. | Pre-existing follows, likes, reposts, replies, and aggregate counts; Alice/Bob protected; Carol unrelated; Bob then absent. | Compare storage and viewer-relative responses before/after mute, block, membership loss, and restoration. | Canonical records and Carol's eligible record-based contributions remain intact; Alice/Bob viewer state and metrics are hidden as required. | `appview/internal/api/relationship_restore_test.go`, affected aggregate store tests |
| IT-035 | FR-002, FR-004, FR-029, NFR-004 | AC-008, AC-012, AC-042, AC-053 | Migrations are reversible and enforce relationship ownership/index invariants. | Real Postgres at the pre-feature migration, plus owner/subject relationship fixtures. | Migrate up, inspect constraints/indexes, exercise uniqueness and owner/subject lifecycle, migrate down, then migrate up again. | Mute pairs are unique by owner/subject; block URI and owner/rkey constraints support idempotent projection while pair indexes allow duplicate external records; both directions and owned lists have usable indexes; owner deletion and subject retention semantics are possible; down migration removes only the two introduced tables. | `appview/internal/db/mutes_blocks_migration_test.go` |
| IT-036 | BR-002, FR-004 | AC-004, AC-012 | Tap collection filtering and dispatcher wiring include the canonical block collection exactly once. | Production dependency assembly and compose/Tap configuration under test. | Build dependencies, enumerate registered NSIDs and configured collection filters, and dispatch a representative block event. | `app.bsky.graph.block` is subscribed and mapped to one idempotent block indexer; no local Craftsky block NSID or duplicate registration exists. | `appview/internal/app/indexer_wiring_test.go`, `appview/cmd/cli/tap_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Direct profile/content reads and ordinary interactions remain allowed across a mute. | FR-012, RULE-001 | AC-020, AC-045 | Extend existing profile/post/follow/interaction suites so mute suppresses unsolicited delivery only and does not return `interaction_blocked`. |
| REG-002 | Block is not a destructive soft-block and unblock restores eligible state. | FR-020, RULE-005 | AC-031, AC-048 | Assert stored follow/content/interaction rows survive block/unblock and graph state returns without recreation. |
| REG-003 | Existing platform hidden/taken-down moderation remains stricter than relationship placeholders. | RULE-003 | AC-047 | Extend moderation response/store tests so hidden/taken-down content never becomes revealable through mute handling and block remains stricter than mute. |
| REG-004 | Existing delete and report flows remain usable when a block exists. | FR-018 | AC-028 | Run owned delete and profile/content report handler/widget suites under inbound and outbound blocks without exposing hidden raw content. |
| REG-005 | Existing public records involving a former/non-member are preserved even though user-facing membership rules tighten. | FR-020, FR-028 | AC-051, AC-052 | Seed legacy follow/content/block records, remove membership, verify omission/no new write, then rejoin and restore after backfill. |
| REG-006 | Unrelated viewers retain record-based aggregates and eligible graph/content views. | RULE-008 | AC-058 | Compare Carol's responses and underlying counts before/after Alice/Bob mute, block, and membership absence. |
| REG-007 | Existing API middleware, camelCase, envelope, cursor, and request-ID behavior remains unchanged. | FR-026 | AC-038 | Add every new route to current route/middleware contract tables and reuse envelope/cursor tests. |
| REG-008 | Older or stale clients cannot bypass AppView policy. | BR-001, NFR-002 | AC-002, AC-040 | Call raw API routes without new viewer-side filtering and verify protected reads/writes/deliveries still enforce policy. |
| REG-009 | Existing multi-account activation, late-completion, cache, and notification boundaries remain isolated. | FR-025, NFR-001 | AC-006, AC-037 | Extend account-boundary and account-switch routing/provider tests with mute, block, list, reveal, and notification state. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Relationship identity matrix | Alice, Bob, Carol current-member DIDs/handles; Alice alias/changed handle; resolvable never-member; former member; unknown; invalid identifier. | AT-001–AT-015, UT-002–UT-004, UT-014, IT-001–IT-034 |
| TD-002 | Relationship state matrix | None, Alice mutes Bob, Alice blocks Bob, Bob blocks Alice, mutual blocks, mute plus either block, platform hide/takedown. | UT-001, UT-003–UT-008, IT-003, IT-010, IT-013–IT-022, IT-027, IT-032–IT-034 |
| TD-003 | Canonical block event history | Valid `app.bsky.graph.block` records with explicit owner repository and subject DID: joining-owner → current-subject, current-owner → absent/joining-subject, duplicate pair URIs, repository-ordered create/update/delete with duplicate replay, malformed subject/time, delayed Tap, and rapid pre-index unblock. | UT-010, IT-005–IT-009, IT-032, IT-036, MAN-002 |
| TD-004 | Dense content graph | Ordinary/project top-level posts, comments, nested replies, straight reposts, quotes, embeds, mentions, media/facets, mixed authors, and third-party references connecting a blocked pair. | AT-003–AT-010, UT-005–UT-006, IT-012–IT-017, IT-027, IT-029, IT-033–IT-034 |
| TD-005 | Notification lifecycle | All supported categories with list/newness/badge state and outbox rows in pending, retry, leased-unsent, sending, sent, cancelled, expired, and retained-hidden states. | AT-006, UT-007, IT-018–IT-020, IT-031, IT-033 |
| TD-006 | Pagination fixture | At least 125 relationships and three content pages with equal timestamps, stable DID tie-breakers, current/former members, and dense protected rows at page boundaries. | AT-011, UT-009, IT-011–IT-012, IT-021–IT-022, IT-029–IT-030 |
| TD-007 | Multi-account client state | Alice and Carol sessions, fixed account-bound repositories/clients, per-account providers/caches/lists/counts, session generations, in-flight mutation, confirmed optimistic overlay, stale pre-Tap refresh, and revealed branch. | AT-004, AT-012, UT-011, IT-003, IT-009, REG-009 |
| TD-008 | Existing public record restoration | Follow, like, repost, reply, mention, report, block, notification, and aggregate rows created before relationship or membership changes. | IT-017, IT-020–IT-021, IT-025–IT-026, IT-034, REG-002, REG-004–REG-006 |
| TD-009 | API contract requests | Valid/missing/expired bearer token, valid/missing device ID, malformed identifier, self target, valid/tampered cursor, limits 0/50/100/101, injected PDS/store errors and rate limit. | IT-001–IT-006, IT-011, IT-023–IT-024, REG-007 |
| TD-010 | Localization and semantics states | Every supported locale plus mute/unmute, outbound/inbound block, confirmation, destructive, revealable/unrevealable, loading/empty/error/retry/pagination/disabled states. | AT-001, AT-004–AT-015, UT-013, MAN-001 |
| TD-011 | Membership lifecycle | Subject departure/rejoin, owner permanent removal, sign-out, device removal, account switch, profile row absent/present, joining-owned backfill held/resumed, process exit during Tap work, service recreation, and idempotent retry. | UT-014, UT-016, IT-008, IT-024–IT-026, IT-035, REG-005 |
| TD-012 | Diagnostic capture | In-memory slog handler, trace/event recorder, bounded metric sink, Flutter error reporter, and deliberately sensitive target/pair fields. | UT-012, IT-031 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-005 | AC-043 | Real VoiceOver/TalkBack and keyboard navigation smoke test. | On phone and large layout, traverse visitor profile More, post/project/comment/reply More, block/unblock confirmations, muted branch/quote reveal, blocked placeholder, and both Settings lists in at least one non-default locale. | Spoken labels, roles, state, focus order, escape/back behavior, and destructive distinction are clear; no unlabeled action, focus trap, clipped critical copy, or raw localization key. |
| MAN-002 | BR-002 | AC-003, AC-004 | Compatible-client/local-PDS interoperability smoke test. | Against the local full stack, create a block in Craftsky and inspect the immediate UI/PDS result before Tap; create/delete a valid `app.bsky.graph.block` with a compatible client or repo tool; allow Tap/backfill to converge; inspect both Craftsky accounts. | Craftsky writes one canonical public record, keeps its immediate Flutter state without a synchronous AppView row, honors external create/delete without recreation, and enforces both directions after Tap convergence. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | A provider-accepted push cannot be retracted or proven absent after it has left Craftsky. | FR-013, FR-019, FR-031 | This is an explicit platform boundary, not an implementation promise. | Automate final pre-send eligibility and no-replay behavior with a deterministic sender; document already-provider-sent delivery as outside the guarantee. |
| GAP-002 | Automated tests synthesize compatible block records but do not prove behavior against every external atproto client. | BR-002, FR-004, FR-005 | External client implementations and PDS deployments are outside the repository test boundary. | Keep canonical lexicon fixtures and Tap/backfill tests as the gate; run MAN-002 before release or interoperability-significant changes. |
| GAP-003 | Requirements specify indexed/batched checks but no numerical latency or query-cost budget. | NFR-004 | Correctness can be gated by plan/index/no-N+1 assertions, but there is no pass/fail performance SLA. | Capture baseline query plans and call counts during implementation; propose a reviewed budget if representative data shows material regression. |
| GAP-004 | The affected endpoint inventory can drift as new read/write surfaces are added later. | NFR-002, RULE-006 | A static list can become incomplete after this feature ships. | Add a central policy/import boundary and require new post/profile/notification/push routes to register in the inventory test. |
| GAP-005 | Tap backfill can be delayed or interrupted after profile membership becomes visible. | FR-005, FR-028 | The accepted contract permits temporary divergence for joining-owned historical blocks, so tests cannot assert zero exposure during every failure interval. | IT-008/IT-025 must prove retained inbound state applies immediately, joining-owned state eventually converges after restart/retry, and lag/failure is observable. |

No blocking test-design gaps are identified. GAP-001 is an explicit delivery boundary; GAP-002 and MAN-002 provide a live interoperability smoke path; GAP-003 concerns an unspecified performance threshold rather than functional coverage; GAP-005 records the explicitly accepted membership/backfill convergence window exercised by IT-008 and IT-025.

## 10. Out Of Scope

- Muted words, tags, phrases, languages, threads, individual-post hides, snooze, temporary mute, and expiry behavior.
- Moderation-list mute/block subscriptions, list blocks, starter packs, and DM/group-chat restrictions.
- Changes to reports, Ozone labels, platform moderation taxonomy, or anonymous/public browsing.
- A local Craftsky lexicon or generated lexicon changes; blocks use `app.bsky.graph.block` and mutes remain private AppView rows.
- Synchronizing private Craftsky mutes with Bluesky's hosted AppView or another third-party service.
- Proving public repository content or public block records are inaccessible outside compliant authenticated views.
- Destructive deletion of public posts, follows, blocks, likes, reposts, replies, mentions, reports, or record-based aggregate contributions as a relationship side effect.
- Pixel-identical Bluesky UI. Tests cover behavioral parity, localized copy, clear state, and accessible controls.
- Retracting a push after the provider has already accepted/sent it.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-18-mutes-and-blocks/`
- Risk level: High; explicit approval is required after document review before implementation.
- Recommended first failing test for implementation: `IT-035` in proposed `appview/internal/db/mutes_blocks_migration_test.go`, proving the minimum reversible mute/block schema, ownership constraints, and bidirectional indexes against real Postgres. The first feature-behavior test remains `IT-002` for owner-scoped unique mute persistence and immediate/idempotent behavior.
- Suggested test order for implementation:
  1. Write `IT-035` and `IT-002` together; make `IT-035` the first red schema contract, then green only the minimum reversible mute/block migration and owner-scoped mute store behavior. Add `UT-001`, `UT-014`, and `UT-016` for shared policy, profile-row membership, and lifecycle semantics.
  2. `IT-005`–`IT-009`, `IT-036`, and `UT-010` for PDS-confirmed blocks, no synchronous projection, ordered Tap convergence, rapid pre-index unblock, and join-time backfill.
  3. `IT-010`–`IT-017`, `IT-021`–`IT-024`, and `IT-027`–`IT-034` for response shaping, complete read/write inventory, pagination, privacy, and restoration.
  4. `IT-018`–`IT-020` for notification/newness/badge/push races and restoration.
  5. `UT-003`–`UT-009`, `UT-011`–`UT-013`, `UT-015`, then `AT-001`–`AT-015` for Flutter models, repositories, providers, widgets, routing, localization, and account isolation.
  6. `REG-001`–`REG-009`, full Go/Flutter gates, MAN-001, and MAN-002.
- Commands discovered:
  - Full Go gate from repository root: `just test` (requires `just dev-d` and compose Postgres on host port 5433).
  - Focused AppView packages from `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/index ./internal/notifications ./internal/push ./internal/db`
  - Focused Flutter suites from `app/`: `flutter test test/profile test/feed test/projects test/settings test/notifications test/search test/router test/auth`
  - Full Flutter tests from repository root: `just app-test`
  - Flutter static analysis from repository root: `just app-analyze`
- Existing automation conventions:
  - Go handler tests use `net/http/httptest` with explicit auth context and recording fakes.
  - Store/indexer/migration tests use `internal/testdb` against real Postgres schemas.
  - Flutter tests use `flutter_test`, Riverpod provider overrides/`ProviderContainer`, repository fakes, localized `MaterialApp`, and `GoRouter` harnesses.
- Blocking gaps: None.
