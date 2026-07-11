# Acceptance Test Specification: AppView Push Notifications

## 1. Test Strategy

This high-risk change requires layered, predominantly automated coverage. Pure policy and timing rules belong in table-driven unit tests. Persistence, transactionality, pagination, fan-out, lifecycle changes, and concurrent claims require isolated real-Postgres integration tests using `internal/testdb.WithSchema`. HTTP contracts use `httptest` through the registered routes and existing middleware. The dispatcher is tested with an injected fake sender and controllable clock; no automated test may contact FCM. One Android and one iOS delivery through a non-production Firebase project remain manual provider checks.

The most important test seam is a single notification-ingestion service called inside each source indexer's database transaction. Producer tests must prove both the indexed source row and notification/outbox effects commit or roll back together. Dispatcher tests treat delivery as at-least-once and explicitly preserve the documented crash window after provider acceptance.

Risk level: **High** (carried forward). Document review and explicit approval are required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-009 | IT-001, IT-006 | Integration | Yes |
| BR-002 | AC-002, AC-014, AC-015 | IT-002, IT-008, IT-009 | Integration / HTTP | Yes |
| BR-003 | AC-003, AC-011 | UT-001, IT-007 | Unit / HTTP | Yes |
| FR-001 | AC-001, AC-004, AC-009, AC-032, AC-035 | IT-001, IT-003, IT-006, IT-017, IT-020 | Integration | Yes |
| FR-002 | AC-004 | IT-003 | Integration | Yes |
| FR-003 | AC-005, AC-016 | IT-004, AT-006 | Integration / Acceptance | Yes |
| FR-004 | AC-001, AC-009, AC-010 | IT-001, IT-006, IT-005 | Integration | Yes |
| FR-005 | AC-009 | IT-006 | Integration | Yes |
| FR-006 | AC-011, AC-012 | IT-007 | HTTP integration | Yes |
| FR-007 | AC-011, AC-012 | UT-002, IT-007 | Unit / HTTP | Yes |
| FR-008 | AC-013, AC-037 | UT-003, IT-010 | Unit / Integration | Yes |
| FR-009 | AC-014, AC-017, AC-018, AC-038 | IT-008, IT-011, IT-012, IT-021 | HTTP / DB integration | Yes |
| FR-010 | AC-014, AC-018, AC-038 | IT-008, IT-012, IT-021 | Integration | Yes |
| FR-011 | AC-002, AC-015, AC-035 | IT-002, IT-009, IT-020 | Integration | Yes |
| FR-012 | AC-016, AC-019, AC-020 | AT-006, IT-013, IT-014 | Worker integration | Yes |
| FR-013 | AC-019, AC-036 | UT-006, IT-013, IT-016 | Unit / Integration | Yes |
| FR-014 | AC-020 | IT-014 | Worker integration | Yes |
| FR-015 | AC-021 | IT-015 | Worker integration | Yes |
| FR-016 | AC-017, AC-038 | IT-011, IT-021 | Integration | Yes |
| FR-017 | AC-006, AC-007 | AT-002, AT-003 | Acceptance | Yes |
| FR-018 | AC-035 | IT-020 | Integration | Yes |
| FR-019 | AC-008 | UT-004, AT-004 | Unit / Acceptance | Yes |
| FR-020 | AC-006, AC-008 | AT-002, AT-004 | Acceptance | Yes |
| FR-021 | AC-005, AC-010, AC-032, AC-033 | IT-004, IT-005, IT-017, IT-018 | Integration | Yes |
| FR-022 | AC-022, AC-023 | UT-005, IT-019 | Unit / Integration | Yes |
| FR-023 | AC-023, AC-024, AC-032, AC-039 | IT-017, IT-019, IT-022 | Integration | Yes |
| FR-024 | AC-025, AC-040 | UT-007, IT-023 | Unit / Worker integration | Yes |
| FR-025 | AC-026 | REG-005 | Regression | Yes |
| FR-026 | AC-032, AC-034 | IT-017, IT-024 | HTTP integration | Yes |
| FR-027 | AC-041 | IT-025 | Integration | Yes |
| FR-028 | AC-036 | UT-006, IT-016 | Unit / Worker integration | Yes |
| FR-029 | AC-040 | IT-023 | Integration | Yes |
| FR-030 | AC-039 | IT-022 | Integration | Yes |
| FR-031 | AC-042 | IT-026 | Integration | Yes |
| FR-032 | AC-043 | REG-006 | Regression / structure | Yes |
| FR-033 | AC-044 | IT-031 | HTTP / DB integration | Yes |
| NFR-001 | AC-016 | AT-006 | Acceptance | Yes |
| NFR-002 | AC-027 | UT-008, IT-027 | Unit / Integration | Yes |
| NFR-003 | AC-028 | AT-007 | HTTP acceptance | Yes |
| NFR-004 | AC-029 | UT-009, REG-004 | Unit / Regression | Yes |
| NFR-005 | AC-030 | IT-028 | Integration / schema | Yes |
| NFR-006 | AC-031 | IT-029 | Integration | Yes |
| RULE-001 | AC-013 | UT-003, IT-010 | Unit / Integration | Yes |
| RULE-002 | AC-011, AC-012, AC-037 | UT-002, UT-003, IT-007, IT-010 | Unit / Integration | Yes |
| RULE-003 | AC-010, AC-013, AC-032 | IT-005, IT-010, IT-017 | Integration | Yes |
| RULE-004 | AC-021 | IT-015, REG-007 | Integration / Regression | Yes |
| RULE-005 | AC-003, AC-011 | UT-001, IT-007 | Unit / HTTP | Yes |
| RULE-006 | AC-003, AC-007 | UT-001, AT-003 | Unit / Acceptance | Yes |

The Should requirements NFR-005 and NFR-006 are included because query shape and delivery latency are major operational risks. Every AC-001 through AC-044 is covered.

## 3. Acceptance Scenarios

### AT-001: Eligible activity becomes durable feed and push intent
Requirement IDs: BR-001, BR-002, FR-001, FR-004, FR-011  
Acceptance Criteria: AC-001, AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_ingestion_test.go`

```gherkin
Feature: Durable notification ingestion
  Scenario: Eligible activity reaches the feed and every current subscription
    Given a recipient with push enabled and two active account subscriptions
    When another member creates an eligible social event
    Then one active notification with a stable opaque ID is committed
    And it appears in the recipient's notification feed
    And exactly one pending delivery exists for each active subscription
```

### AT-002: Direct interaction and self-exclusion
Requirement IDs: FR-017, FR-020  
Acceptance Criteria: AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_interaction_test.go`

```gherkin
Scenario Outline: Likes and reposts notify only the post author
  Given Bob authored a post
  When <actor> performs <action> on the post
  Then Bob has <count> new <category> notifications

  Examples:
    | actor | action  | category | count |
    | Alice | likes   | like     | 1     |
    | Alice | reposts | repost   | 1     |
    | Bob   | likes   | like     | 0     |
    | Bob   | reposts | repost   | 0     |
```

### AT-003: Repost ownership never implies attribution
Requirement IDs: FR-017, RULE-006  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_interaction_test.go`

```gherkin
Scenario: Interaction with an underlying post does not notify its reposter
  Given Carol authored a post and Bob reposted it
  When Alice likes or reposts the underlying post
  Then Carol may receive the direct notification
  And Bob receives no notification solely because Bob reposted it
```

### AT-004: Per-recipient canonical post precedence
Requirement IDs: FR-019, FR-020  
Acceptance Criteria: AC-008  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_post_test.go`

```gherkin
Scenario Outline: One post produces one canonical category per recipient
  Given a post qualifies the same recipient for <reasons>
  When the post is indexed
  Then exactly one notification is created for that recipient
  And its category is <category>

  Examples:
    | reasons                 | category |
    | reply, quote, mention   | reply    |
    | quote, mention          | quote    |
    | mention                 | mention  |
```

### AT-005: Undo, safe resolution, and reactivation
Requirement IDs: FR-018, FR-021, FR-026  
Acceptance Criteria: AC-010, AC-032, AC-033, AC-035  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_lifecycle_test.go`, `appview/internal/api/notification_resolution_test.go`

```gherkin
Scenario: An undone and recreated action keeps identity without push spam
  Given a delivered notification for a like, repost, or follow
  When the source action is deleted
  Then the notification is retracted and absent from the normal feed
  And its unsent deliveries are cancelled
  And resolving its ID returns an authorized safe fallback
  When the same relationship is recreated
  Then the same notification ID is active at the top of the feed
  And no second delivery is created
```

### AT-006: Provider outage cannot block indexing
Requirement IDs: FR-003, FR-012, NFR-001  
Acceptance Criteria: AC-005, AC-016  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/notification_transaction_test.go`, `appview/internal/push/dispatcher_test.go`

```gherkin
Scenario: FCM is unavailable during ingestion
  Given the injected push sender is blocked or failing
  When a Tap event is indexed and acknowledged
  Then source and notification state commit without waiting for the sender
  And a durable delivery remains available for asynchronous processing
  And no provider call occurs inside the indexing transaction
```

### AT-007: New routes preserve the V1 API contract
Requirement IDs: FR-006, FR-009, FR-026, NFR-003  
Acceptance Criteria: AC-028, AC-034  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/routes/notification_routes_test.go`

```gherkin
Scenario Outline: Notification APIs reject invalid requests consistently
  Given a request to a new notification route with <problem>
  When the route handles the request
  Then it returns <status> using the standard camelCase error envelope
  And it reveals no notification, account, or token data

  Examples:
    | problem                       | status |
    | missing session               | 401    |
    | missing device ID             | 400    |
    | invalid JSON                  | 400    |
    | another recipient's stable ID | 404    |
```

### AT-008: Account-scoped logout on a shared installation
Requirement IDs: FR-009, FR-010, FR-016, FR-029  
Acceptance Criteria: AC-017, AC-038, AC-040  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/notification_devices_test.go`

```gherkin
Scenario: Two accounts share one installation
  Given one installation token has active subscriptions for Alice and Bob
  And each account has a distinct opaque local routing ID
  When Alice logs out normally
  Then Alice's subscription and unsent work on that installation are removed
  And Bob's subscription, routing ID, and queued work remain active
  And all future sends use the installation's one current token
```

### AT-009: Cross-device token registration safely rebinds ownership
Requirement IDs: FR-009, FR-010, FR-033  
Acceptance Criteria: AC-044  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/notification_devices_test.go`

```gherkin
Scenario: A provider token moves to a new device identity
  Given an active installation owns a token and has subscriptions and unsent deliveries for Alice and Bob
  When Alice registers the same token using a different device ID
  Then the old installation and both old subscriptions are inactive
  And their unsent deliveries are cancelled
  And the token belongs only to the new installation
  And only Alice receives a new subscription with a new opaque routing ID
  And no old subscription or routing ID is transferred
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-003, RULE-005, RULE-006 | AC-003 | Enumerate public categories. | Category registry | Exactly seven specified categories; no via-repost category; `everythingElse` has no producer. | `appview/internal/notifications/category_test.go` |
| UT-002 | FR-007, RULE-002 | AC-011, AC-012 | Resolve defaults and merge partial preference patches. | Missing rows; one/many valid patches | Missing values become `everyone/true`; omitted categories retain effective values; invalid scope/category is rejected. | `appview/internal/notifications/preferences_test.go` |
| UT-003 | FR-008, RULE-001, RULE-002 | AC-013, AC-037 | Evaluate event-time eligibility. | Scope, push flag, self flag, follows-actor snapshot | Scope controls both in-app acceptance and push eligibility; push flag controls only delivery; self is rejected. | `appview/internal/notifications/eligibility_test.go` |
| UT-004 | FR-019 | AC-008 | Select canonical post reason independently per recipient. | Sets of reply/quote/mention candidates | `reply > quote > mention`; different recipients can receive different categories. | `appview/internal/notifications/classify_test.go` |
| UT-005 | FR-022 | AC-022 | Validate common and category-specific response references. | One event for each category | Required metadata matrix fields are present and unrelated type fields are absent. | `appview/internal/api/notification_response_test.go` |
| UT-006 | FR-013, FR-028 | AC-019, AC-036 | Calculate retry schedule and provider TTL with an injected clock/jitter source. | Retryable errors near/before deadline | Bounded exponential delay; no next attempt after six hours; TTL is positive and no greater than remaining deadline. | `appview/internal/push/retry_test.go` |
| UT-007 | FR-024 | AC-025 | Build combined FCM payload for named and unnamed actors. | All categories; display name present/missing | Only generic copy, display name or `Someone`, notification ID, category, and routing ID appear; denylisted fields never appear. | `appview/internal/push/payload_test.go` |
| UT-008 | NFR-002 | AC-027 | Verify telemetry sanitization and failure classification. | Token/credential/payload-shaped secrets in success and error inputs | Recorder/logger/Sentry attributes and metric labels contain none of the secrets or full payload. | `appview/internal/observability/push_test.go` |
| UT-009 | NFR-004 | AC-029 | Validate environment/config combinations and sender injection. | Production/dev/test configs, missing/malformed credentials, fake sender | Production enabled without valid config fails safely; disabled and injected-fake modes require no network. | `appview/internal/app/push_config_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup / Action | Expected Result | Automation Target |
|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, FR-004 | AC-001 | Persist and list one eligible event. | Seed subject/actors; index each producer. | Stable active row is returned from durable store. | `appview/internal/index/notification_ingestion_test.go` |
| IT-002 | BR-002, FR-011 | AC-002 | Fan out to two current subscriptions. | Register two active subscriptions; ingest event. | Two unique pending deliveries, one per subscription. | Same suite |
| IT-003 | FR-001, FR-002 | AC-004 | Replay exact and equivalent source events. | Handle same event/CID repeatedly and replay after restart. | One notification and at most one delivery per subscription. | Same suite |
| IT-004 | FR-003, FR-021 | AC-005 | Force failures at source, notification, and delivery insert/cancellation boundaries for both creation and deletion. | Inject transaction callback/constraint failures before and after notification/outbox lifecycle writes, including representative follow, interaction, and post deletion paths. | Source plus notification/outbox effects all commit or all roll back; no active notification survives a committed source deletion, no source remains deleted after lifecycle rollback, and fake sender call count remains zero. | `appview/internal/index/notification_transaction_test.go` |
| IT-005 | FR-004, FR-021, RULE-003 | AC-010 | Delete each producing source type. | Create then delete follow, like, repost, reply, mention, quote. | Tombstone is retracted and absent from list. | `appview/internal/index/notification_lifecycle_test.go` |
| IT-006 | BR-001, FR-005 | AC-009 | Paginate tied timestamps. | Seed more than two pages with identical activity times and stable IDs. | Reverse chronology plus deterministic tiebreak; every ID exactly once; malformed cursor rejected. | `appview/internal/api/notification_store_test.go` |
| IT-007 | FR-006, FR-007, RULE-002 | AC-003, AC-011, AC-012 | GET defaults and PATCH subsets through routes. | New user; patch one then multiple categories. | Seven camelCase entries; defaults correct; omitted state preserved; token never present. | `appview/internal/api/notification_preferences_test.go` |
| IT-008 | FR-009, FR-010 | AC-014 | Register installation and subscription idempotently. | Repeat authenticated POST with device ID/platform/token. | One installation/subscription; opaque routing ID returned; token not echoed. | `appview/internal/api/notification_devices_test.go` |
| IT-009 | FR-011 | AC-015 | Register after event. | Ingest notification, then register installation. | No retrospective delivery. | Same suite |
| IT-010 | FR-008, RULE-001, RULE-003 | AC-013, AC-037 | Change follows and preferences across successive distinct subjects. | Events before/after follow, unfollow, scope change. | Each new event uses its ingestion snapshot; old rows unchanged; non-followed actor produces neither row nor delivery; follow category requires mutual follow under restricted scope. | `appview/internal/index/notification_eligibility_test.go` |
| IT-011 | FR-009, FR-016 | AC-017 | Ordinary and all-session removal. | Shared/multiple installations with mixed accounts and queued work. | Exact account scope removed/cancelled; unrelated subscriptions/work intact. | `appview/internal/api/notification_devices_test.go` |
| IT-012 | FR-010 | AC-018 | Rotate a token with pending work. | Re-register device with new token. | One installation; subscriptions unchanged; dispatch resolves only new token. | Same suite |
| IT-013 | FR-012, FR-013 | AC-019 | Retry transient errors to success and expiry. | Fake sender returns retryable sequence; advance fake clock. | Attempts/backoff bounded; terminal success or expired; no hot loop. | `appview/internal/push/dispatcher_test.go` |
| IT-014 | FR-012, FR-014 | AC-020 | Handle invalid/unregistered token. | Fake permanent response. | Delivery terminal; installation and every subscription inactive; no future fan-out. | Same suite |
| IT-015 | FR-015, RULE-004 | AC-021 | Poll after success and model crash window. | Mark success; poll again; separately return accepted without committing. | Success is not selected normally; lease recovery may resend only in documented crash case. | Same suite |
| IT-016 | FR-013, FR-028 | AC-036 | Enforce absolute deadline and TTL. | Claim at several times including deadline boundary. | No send after deadline; each provider TTL is at most remaining window. | Same suite |
| IT-017 | FR-021, FR-023, FR-026 | AC-032 | Resolve retracted categories and deleted destinations. | Deliver then delete each source/destination; GET by ID as owner. | Retracted response gives category-specific safest authorized fallback and never reappears in feed. | `appview/internal/api/notification_resolution_test.go` |
| IT-018 | FR-021 | AC-033 | Delete while pending/retrying. | Create delivery, put in pending/retry/leased states, process deletion. | Unsent work becomes cancelled and cannot be normally claimed; accepted delivery is not represented as recalled. | `appview/internal/index/notification_lifecycle_test.go` |
| IT-019 | FR-022, FR-023 | AC-022, AC-023, AC-024 | Hydrate all categories under available/unavailable/taken-down content. | Seed each metadata shape and visibility state. | Bounded response metadata; inaccessible content explicitly unavailable with no stale data leakage. | `appview/internal/api/notification_store_test.go` |
| IT-020 | FR-001, FR-011, FR-018 | AC-035 | Undo and recreate relationship. | Like/repost/follow create-delete-create, including changed source URI. | Same ID, newer activity timestamp, active/top-of-feed, original unique deliveries only. | `appview/internal/index/notification_lifecycle_test.go` |
| IT-021 | FR-009, FR-010, FR-016 | AC-038 | Shared installation multi-account isolation. | Register two DIDs on same device, queue both, logout either. | One token, two routing IDs; other account and its work survive. | `appview/internal/api/notification_devices_test.go` |
| IT-022 | FR-023, FR-030 | AC-039 | Permanent actor repository deletion. | Create active/retracted/delivered/unsent notifications, process actor deletion. | Caused notifications hard-deleted, unsent work removed/cancelled, stale resolution is non-enumerating 404. | `appview/internal/index/notification_actor_deletion_test.go` |
| IT-023 | FR-024, FR-029 | AC-025, AC-040 | Route one account on two installations. | Register twice; dispatch both through capturing sender. | Distinct installation-local opaque IDs; correct local ID per payload; no DID/global identifier or forbidden content. | `appview/internal/push/dispatcher_test.go` |
| IT-024 | FR-026 | AC-034 | Cross-account resolution authorization. | Owner and non-owner request same valid/unknown IDs. | Non-owner and unknown responses are indistinguishable standard not-found. | `appview/internal/api/notification_resolution_test.go` |
| IT-025 | FR-027 | AC-041 | Prospective push toggle. | Queue; disable; ingest; re-enable. | Existing queue remains; disabled-period event may be in feed but has no delivery; no backfill after re-enable. | `appview/internal/index/notification_eligibility_test.go` |
| IT-026 | FR-031 | AC-042 | Burst creates per-event intents. | Ingest many distinct eligible subjects rapidly. | One notification and delivery intent per eligible event; no aggregation/delay state. | `appview/internal/index/notification_ingestion_test.go` |
| IT-027 | NFR-002 | AC-027 | Exercise registration, enqueue, send success/failure with recording telemetry. | Use sentinel secrets in all paths. | No response, log, Sentry attribute, metric label, or diagnostic exposes sentinel values/full payload. | `appview/internal/observability/push_integration_test.go` |
| IT-028 | NFR-005 | AC-030 | Verify bounded queries and required migration indexes. | Inspect migration definitions/query plans; hydrate/claim multi-row batches with query-count instrumentation where practical. | Recipient/time/ID pagination and pending/next-attempt claims use supporting indexes; work scales by batch, not per row. | `appview/internal/db/notifications_migration_test.go`, store tests |
| IT-029 | NFR-006 | AC-031 | Record queue and delivery metrics. | Seed aged pending work and each outcome; run observation/dispatch. | Queue depth, oldest age, and classified outcomes recorded with validated low-cardinality attributes. | `appview/internal/observability/push_test.go` |
| IT-030 | FR-012 | AC-016, AC-019 | Concurrent claims and lease recovery. | Run two dispatchers against same rows; stop owner; advance past lease. | Normally one owner/send per claim; uncommitted expired lease is recoverable; cancellation shuts workers down. | `appview/internal/push/dispatcher_concurrency_test.go` |
| IT-031 | FR-009, FR-010, FR-033 | AC-044 | Safely rebind one token across device IDs. | Register a token on device A with multiple account subscriptions and unsent work; register the same token as authenticated Alice on device B. | Device A and all its subscriptions are inactive, its unsent work is cancelled, device B uniquely owns the token, only Alice is subscribed with a new opaque routing ID, and no old authorization is transferred or returned. | `appview/internal/api/notification_devices_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing follow/like/repost/reply/mention categories continue to render while quote is added. | BR-003, FR-022 | AC-003, AC-022 | Extend `internal/api/notifications_test.go` with all supported response variants and camelCase assertions. |
| REG-002 | Tap duplicate delivery remains idempotent for existing indexers. | FR-002 | AC-004 | Extend producer indexer tests to replay create/delete events and assert both source and notification convergence. |
| REG-003 | Content availability/takedown checks remain authoritative. | FR-023, FR-032 | AC-023, AC-024, AC-043 | Existing moderation-policy scenarios plus notification hydration/resolution must not reveal unavailable data. |
| REG-004 | AppView startup/shutdown and existing server operation remain cancellable without push configuration in dev/test-disabled mode. | FR-012, NFR-004 | AC-016, AC-029 | Extend `cmd/appview/server_test.go` and `internal/app/deps_test.go` with disabled/fake dispatcher lifecycle cases. |
| REG-005 | Deployment does not backfill already indexed activity. | FR-025 | AC-026 | Apply migration over seeded old source tables and start worker/indexers; assert zero notifications/deliveries until a new event arrives. |
| REG-006 | No notification-only block/mute data model or filtering claim is introduced. | FR-032 | AC-043 | Migration/schema and classifier tests assert eligibility inputs are only specified preferences, follow snapshot, self, membership, and existing availability/takedown state. |
| REG-007 | Successful jobs remain terminal while at-least-once crash recovery remains explicit. | RULE-004 | AC-021 | Dispatcher tests separately assert no normal resend after success and permitted resend after acceptance-before-commit lease recovery. |
| REG-008 | Existing logout/session behavior does not remove another local account. | FR-016 | AC-017, AC-038 | Run current auth/session regression tests plus shared-installation account removal cases. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Social graph and self/direct attribution | Alice, Bob, Carol DIDs; authored posts; follows in both directions | AT-002–AT-004, IT-010 |
| TD-002 | Every producer and metadata contract | Follow, like, repost, reply, mention, quote records with typed URIs/CIDs and parent/quoted refs | IT-001, IT-005, IT-019 |
| TD-003 | Pagination ties | 55+ notification IDs sharing activity timestamp with deterministic ID ordering | IT-006 |
| TD-004 | Multi-device/account routing and safe token rebinding | Two device IDs/tokens; Alice on both; Bob sharing one; unique routing IDs; one token collision across device IDs with queued work | AT-008, AT-009, IT-008–IT-012, IT-021, IT-023, IT-031 |
| TD-005 | Dispatcher outcomes | Fake sender scripts: success, transient, invalid token, blocked, accepted-before-commit | IT-013–IT-016, IT-030 |
| TD-006 | Time boundaries | Injected UTC source time plus instants before, at, and after six-hour deadline; deterministic jitter | UT-006, IT-013, IT-016 |
| TD-007 | Privacy sentinels | Unique fake token, credential, DID, handle, AT-URI, text, title, and image URL | UT-007, UT-008, IT-027 |
| TD-008 | Visibility/deletion states | Available, missing, taken-down, source-deleted, destination-deleted, actor-repository-deleted | IT-017, IT-019, IT-022 |
| TD-009 | Preference timeline | Defaults, partial patches, push-off interval, scope/follow changes around distinct events | IT-007, IT-010, IT-025 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-002, FR-024, FR-028, NFR-004 | AC-002, AC-025, AC-029, AC-036, AC-040 | Non-production Android FCM delivery | Register a test Android installation/account, create an eligible event, observe delivery and open data. | OS displays generic copy; routing selects the right local account; payload inspection contains only allowed fields and TTL is bounded. |
| MAN-002 | BR-002, FR-024, FR-028, NFR-004 | AC-002, AC-025, AC-029, AC-036, AC-040 | Non-production iOS delivery through Firebase/APNs | Configure sandbox APNs in Firebase, repeat the Android flow on iOS. | Notification displays and routes correctly with the same minimal payload/privacy contract. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | FCM/APNs end-to-end delivery cannot be deterministic in the normal automated suite. | BR-002, FR-024, FR-028 | Requires external credentials, provider state, and physical/emulated platform integration. | Keep provider checks manual; fully automate sender contract with a fake. |
| GAP-002 | Exactly-once delivery cannot be proven. | RULE-004 | FCM acceptance and AppView success commit are not one atomic transaction. | Assert uniqueness/claiming/success terminal behavior and explicitly test the allowed lease-recovery duplicate window. |
| GAP-003 | Query-count instrumentation may not expose all pgx round trips without a test tracer. | NFR-005 | Exact instrumentation mechanism is a coding-design choice. | Require migration indexes plus bounded batch API tests; choose pgx tracer or EXPLAIN assertions in coding plan. |
| GAP-004 | Flutter cannot yet verify decoding, permission, token refresh, or notification-open navigation. | FR-022, FR-024, FR-029 | Flutter work is explicitly outside this AppView pass. | Coordinate a later client acceptance specification before enabling new categories end to end. |
| GAP-005 | Concrete FCM client/auth format remains undecided. | FR-012, NFR-004 | Non-blocking coding-design decision. | Preserve the sender interface/fake contract; select and review implementation during coding planning. |
| GAP-006 | Retention and out-of-order reconciliation are not tested. | FR-025, NFR-005 | Both are explicit non-goals; no policy/behavior exists to assert. | Add requirements and tests when retention/reconciliation is designed. |

## 10. Out Of Scope

- Flutter UI, permission prompts, Firebase client SDK integration, token-refresh listeners, and deep-link implementation.
- Read/unread state, badges, grouping, aggregation, digests, and server-side burst suppression.
- Historical backfill, automatic retention/purge, or reconciliation for missing subjects.
- Email, SMS, web push, raw APNs delivery, notification-specific block/mute state, and temporary deactivation behavior.
- Lexicon/PDS notification records and via-repost attribution.
- A producer for `everythingElse`; only its preference/default contract is tested.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-10-appview-push-notifications/`
- Recommended first failing test for implementation: `UT-001`, defining the closed category set and excluding unsupported via-repost categories. The first vertical database sequence is then migration/index invariants in `IT-028`, followed by one producer's creation and deletion atomicity cases in `IT-004`.
- Suggested test order for implementation: category/preference/eligibility unit tests; migration invariants; transactional ingestion and replay; producer classification; durable list/resolution; device/subscription APIs; lifecycle/reactivation/deletion; dispatcher retry/TTL/payload; concurrency; observability; route and startup regressions; provider manual checks.
- Commands discovered:
  - Full suite from repository root (compose Postgres required): `just test`
  - Full AppView suite from `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`
  - Focused packages from `appview/`: `go test ./internal/notifications ./internal/index ./internal/api ./internal/routes ./internal/push ./internal/app ./internal/observability`
  - Tests using `internal/testdb.WithSchema` skip when neither `TEST_DATABASE_URL` nor `DATABASE_URL` is set.
- Blocking gaps: None for document review. Because risk is High, document review and explicit approval are required before implementation.
