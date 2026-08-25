# Acceptance Test Specification: Follower Growth Metrics

## 1. Test Strategy

Risk level: Medium, unchanged from `01-requirements.md`.

The test suite should establish the feature from the database outward. Real-Postgres integration tests will prove snapshot constraints, canonical member-scoped counts, atomicity, concurrency, retention, deletion, and bounded reads. Go unit and route tests will cover period calculation, response shaping, worker lifecycle, authorization, error envelopes, and telemetry redaction. Flutter model, API-client, provider, route, and widget tests will cover account isolation, all history states, responsive layout, localization, and semantics.

Acceptance scenarios are automated through focused Go integration tests and Flutter widget tests rather than a new device-level suite. Manual checks are limited to real screen-reader and final chart legibility checks that Flutter's test semantics tree cannot fully prove. Production-scale query duration remains a documented test gap; query shape and plans are automated.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, UT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-003 | AT-002, UT-007 | Acceptance / Unit | Yes |
| BR-003 | AC-004, AC-005 | AT-003, UT-002, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-006, AC-007 | IT-001, IT-004 | Integration | Yes |
| FR-002 | AC-006 | IT-001 | Integration | Yes |
| FR-003 | AC-007, AC-008 | UT-005, IT-004, IT-005 | Unit / Integration | Yes |
| FR-004 | AC-009, AC-010 | AT-007, IT-002, IT-003 | Acceptance / Integration | Yes |
| FR-005 | AC-011 | IT-004 | Integration | Yes |
| FR-006 | AC-007, AC-012 | IT-004 | Integration | Yes |
| FR-007 | AC-013 | IT-006 | Integration | Yes |
| FR-008 | AC-014 | AT-007, IT-006, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-009 | AC-015, AC-016 | AT-004, IT-007 | Acceptance / Integration | Yes |
| FR-010 | AC-015, AC-017 | UT-004, IT-007, UT-006 | Unit / Integration | Yes |
| FR-011 | AC-018 | UT-001, IT-008 | Unit / Integration | Yes |
| FR-012 | AC-004, AC-019 | AT-003, UT-002, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-020 | IT-009 | Integration | Yes |
| FR-014 | AC-021 | UT-003 | Unit | Yes |
| FR-015 | AC-022 | AT-003, UT-002, UT-007 | Acceptance / Unit | Yes |
| FR-016 | AC-023 | AT-006, IT-012 | Acceptance / Widget | Yes |
| FR-017 | AC-001, AC-024, AC-025 | AT-001, UT-007 | Acceptance / Widget | Yes |
| FR-018 | AC-026 | AT-003, UT-007 | Acceptance / Widget | Yes |
| FR-019 | AC-003, AC-027 | AT-002, UT-007 | Acceptance / Widget | Yes |
| FR-020 | AC-028 | AT-005, UT-008 | Acceptance / Provider | Yes |
| RULE-001 | AC-009, AC-014 | AT-007, IT-002, REG-001 | Acceptance / Integration / Regression | Yes |
| RULE-002 | AC-018, AC-029 | UT-001, UT-006, IT-008 | Unit / Integration | Yes |
| RULE-003 | AC-030 | IT-010 | Integration | Yes |
| RULE-004 | AC-031 | IT-011, REG-005 | Integration / Regression | Yes |
| RULE-005 | AC-032 | REG-003, REG-004 | Regression | Yes |
| NFR-001 | AC-007, AC-012 | IT-001, IT-004 | Integration | Yes |
| NFR-002 | AC-020, AC-033 | AT-007, UT-005, IT-005, IT-009 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-018, AC-034 | UT-001, IT-008 | Unit / Integration | Yes |
| NFR-004 | AC-035 | AT-006, UT-009 | Acceptance / Widget | Yes |
| NFR-005 | AC-036 | AT-006, UT-010, MAN-001 | Acceptance / Widget / Manual | Partial |
| NFR-006 | AC-037 | UT-006, UT-011 | Unit / Widget | Yes |
| NFR-007 | AC-038 | UT-012, IT-013 | Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Owner Views And Changes Growth Period
Requirement IDs: BR-001, FR-017
Acceptance Criteria: AC-001, AC-002, AC-024, AC-025
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/follower_growth_page_test.dart`

```gherkin
Feature: Owner follower growth
  Scenario: Growth defaults to 30 days and supports every period
    Given a current member has distinct snapshot responses for 7 days, 30 days, and 1 year
    When the member opens Growth from Settings
    Then the 30-day control is selected
    And the 30-day summary and observations are displayed
    When the member selects 7 days and then 1 year
    Then each selected control, summary, labels, and graph use its corresponding response
```

### AT-002: Growth Explains Scope And Freshness
Requirement IDs: BR-002, FR-019
Acceptance Criteria: AC-003, AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/follower_growth_page_test.dart`

```gherkin
Feature: Trustworthy metric presentation
  Scenario: Member can identify metric scope and age
    Given the latest persisted snapshot was captured yesterday
    When Growth is displayed
    Then the metric is labelled "Followers"
    And nearby text says counts reflect Craftsky members and update daily
    And the latest snapshot date or capture time is visible
```

### AT-003: Partial History And Gaps Are Honest
Requirement IDs: BR-003, FR-012, FR-015, FR-018
Acceptance Criteria: AC-004, AC-005, AC-019, AC-022, AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/follower_growth_page_test.dart`, `appview/internal/api/follower_growth_test.go`

```gherkin
Feature: Incomplete follower history
  Scenario Outline: Growth renders an incomplete trustworthy series
    Given the selected response is in the <state> state
    When Growth finishes loading
    Then all real observations are displayed
    And unavailable dates are not interpolated or carried forward
    And the page explains the available history without an uncaught error

    Examples:
      | state           |
      | no history      |
      | one point       |
      | partial history |
      | missing date    |
```

### AT-004: Endpoint Is Authenticated And Owner-Only
Requirement IDs: FR-009
Acceptance Criteria: AC-015, AC-016
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/follower_growth_route_test.go`

```gherkin
Feature: Private follower growth API
  Scenario: Only a current member can request their own bounded history
    Given Alice and Bob are Craftsky accounts
    When authenticated current member Alice requests /v1/profiles/me/follower-growth with a supported period
    Then only Alice's history is returned
    When the request is unauthenticated, Alice is no longer current, an arbitrary target is attempted, or the period is unsupported
    Then the request fails through the standard camelCase error envelope
```

### AT-005: Late Response Cannot Cross Accounts
Requirement IDs: FR-020
Acceptance Criteria: AC-028
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/providers/follower_growth_provider_test.dart`

```gherkin
Feature: Multi-account growth privacy
  Scenario: Account changes while a growth request is pending
    Given account A's growth request is outstanding
    When the active account changes to account B
    And account A's request completes later
    Then account A's response is never published or rendered for account B
    And account B loads only through account B's repository binding
```

### AT-006: Growth Is Responsive And Accessible
Requirement IDs: FR-016, NFR-004, NFR-005
Acceptance Criteria: AC-023, AC-035, AC-036
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/follower_growth_page_test.dart`, `app/test/router/settings_routes_test.dart`

```gherkin
Feature: Accessible Growth page
  Scenario Outline: Member uses Growth at a supported layout
    Given the app uses a <layout> layout with the tested large-text scale
    When the member opens Growth from Settings
    Then the typed route, shell, and back stack are preserved
    And controls, summary, and graph do not clip or overflow
    And semantics describe the period, latest count, trend, range, freshness, and missing dates without relying on colour

    Examples:
      | layout  |
      | compact |
      | large   |
```

### AT-007: Daily Capture Records The Canonical Observation
Requirement IDs: FR-003, FR-004, FR-008, RULE-001, NFR-002
Acceptance Criteria: AC-008, AC-009, AC-014, AC-033
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/followergrowth/worker_test.go`, `appview/internal/followergrowth/store_integration_test.go`

```gherkin
Feature: Daily follower snapshot capture
  Scenario: Startup capture observes current member-scoped counts
    Given the current UTC date has not been captured
    And follows and membership have changed since the previous snapshot
    When AppView starts the snapshot worker
    Then it immediately attempts one set-based capture for current members
    And each stored count matches the canonical profile follower count
    And no analytics rows were written when the follows or memberships changed
    And normal worker scheduling keeps the newest successful observation within the accepted daily window
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-011, RULE-002, NFR-003 | AC-018, AC-034 | Calculate supported UTC ranges at ordinary and leap-year boundaries. | Fixed UTC dates; `7d`, `30d`, `1y`; current date 2028-02-29. | Inclusive starts are 6 days, 29 days, or the same prior-year month/day; 2028-02-29 clamps to 2027-02-28 and yields 367 points; no range exceeds 367 points or uses pagination. | `appview/internal/followergrowth/period_test.go` |
| UT-002 | BR-003, FR-012, FR-015 | AC-004, AC-005, AC-019, AC-022 | Shape a calendar series without inventing values and derive global availability. | No snapshots; partial history; owner history before the selected range with a leading in-range gap; interior gaps. | Points are chronological with `count: null` for gaps; `availableFrom` is null for no history and otherwise equals the earliest persisted owner snapshot even outside the selected range; no value is carried forward. | `appview/internal/followergrowth/series_test.go` |
| UT-003 | FR-014 | AC-021 | Calculate signed net change from selected-range observations. | Increasing, decreasing, equal, zero-point, and one-point selected ranges with gaps, plus `availableFrom` before the range. | Latest minus earliest in-range non-null count is returned only with at least two observations; otherwise change is null; older global history does not enter the calculation. | `appview/internal/followergrowth/series_test.go` |
| UT-004 | FR-010 | AC-015, AC-017 | Marshal the populated and no-history API contracts. | Complete response and empty response spanning a fixed period. | Keys are always present and camelCase; no-history `availableFrom`, latest date/time/count, and change are null while every in-range point is present with `count: null`; no snake_case keys. | `appview/internal/api/follower_growth_response_test.go` |
| UT-005 | FR-003, NFR-002 | AC-008, AC-033 | Verify worker run-loop timing with an injected clock/timer and cancellation. | Startup with uncaptured/captured dates, date rollover, capture error, canceled context. | Uncaptured date is attempted immediately, completed date is not duplicated, later date is attempted, errors retry according to configured cadence, and cancellation stops the loop. | `appview/internal/followergrowth/worker_test.go` |
| UT-006 | BR-001, FR-010, RULE-002, NFR-006 | AC-002, AC-017, AC-029, AC-037 | Decode Flutter growth models and preserve UTC bucket identity. | CamelCase payload with null counts; devices/locales west and east of UTC. | Model validates supported period, retains UTC date-only identity and chronological points, and exposes locale-format-ready values. | `app/test/profile/models/follower_growth_test.dart` |
| UT-007 | BR-002, FR-015, FR-017, FR-018, FR-019 | AC-003, AC-022, AC-024, AC-025, AC-026, AC-027 | Render all Growth states and period changes. | Pending, error then success, successful all-null no-history response, one point, partial, leading/interior gap, populated; period-specific fixtures. | Loading/retry/history states are distinct, no-history is not treated as an API error or given fabricated latest metadata, 30 days defaults, period changes replace stale content, and scope/daily-update copy remains visible. | `app/test/settings/follower_growth_page_test.dart` |
| UT-008 | FR-020 | AC-028 | Isolate provider requests by account and discard late results. | Account A delayed future; switch to B; complete B then A; logout case. | Only the active account's bound result is publishable; A cannot overwrite B and logout clears private state. | `app/test/profile/providers/follower_growth_provider_test.dart` |
| UT-009 | NFR-004 | AC-035 | Exercise responsive chart constraints. | 320x640 compact and 1200x800 large viewports, light/dark themes, text scale 2.0. | No Flutter exception, clipping, or overflow; controls and summary remain reachable. | `app/test/settings/follower_growth_page_test.dart` |
| UT-010 | NFR-005 | AC-036 | Inspect textual summary and semantics for populated and missing data. | Positive, negative, flat, one-point, and gapped fixtures with semantics enabled. | Semantics convey selected period, latest count, signed/insufficient trend, range, freshness, and gaps without requiring colour. | `app/test/settings/follower_growth_accessibility_test.dart` |
| UT-011 | NFR-006 | AC-037 | Format count and visible snapshot dates under non-default locales. | `en`, one locale with non-English date order, large count, fixed UTC date. | Locale-specific text changes while the model's UTC date remains unchanged. | `app/test/profile/models/follower_growth_format_test.dart` |
| UT-012 | NFR-007 | AC-038 | Redact worker and API telemetry fields. | Synthetic DID, handle, counts, rows, success and failure paths. | Logs and metric dimensions contain only bounded outcome/category fields; private identifiers and per-account values are absent. | `appview/internal/followergrowth/observability_test.go`, `appview/internal/api/follower_growth_observability_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, FR-002, NFR-001 | AC-006, AC-007 | Enforce snapshot row shape and uniqueness. | Apply snapshot migration in an isolated Postgres schema. | Insert valid, negative-count, duplicate profile/date, and same-profile/different-date rows. | Valid row stores typed DID/date/count/timestamp; negative and duplicate rows fail; the next date succeeds; range index exists. | `appview/internal/followergrowth/store_integration_test.go` |
| IT-002 | FR-004, RULE-001 | AC-009 | Match profile-read membership semantics. | Current target, current follower, non-current follower, non-current target, zero-follower member, and active follows. | Capture a fixed UTC date and read current profile counts. | Every current profile has one snapshot and each count equals the canonical profile result; non-current profiles have none. | `appview/internal/followergrowth/store_integration_test.go` |
| IT-003 | FR-004 | AC-010 | Guard set-based capture and pathological query behavior. | Representative member/follow rows and production indexes in isolated Postgres. | Capture all members while recording database calls and inspect the production SQL/plan. | One set-based capture statement handles all members with no per-profile query loop; plan review rejects pathological repeated subplans but does not require an index when PostgreSQL correctly chooses a whole-table scan. | `appview/internal/followergrowth/query_plan_test.go` |
| IT-004 | FR-001, FR-005, FR-006, NFR-001 | AC-007, AC-011, AC-012 | Prove rollback and concurrent duplicate safety. | Multiple members; injected transaction failure plus two pools/process-equivalent goroutines targeting one date. | Run failing capture, then race two successful captures. | Failed run leaves no subset; concurrent retries finish with one complete logical row set and no replacements/duplicates. | `appview/internal/followergrowth/store_integration_test.go` |
| IT-005 | FR-003, NFR-002 | AC-008, AC-033 | Wire startup lifecycle and daily freshness behavior. | App dependencies with recording capture store and controlled clock/timer. | Start AppView worker on uncaptured date, advance through UTC rollover, cancel. | Capture is attempted on startup and after rollover, not repeatedly after success; normal configured cadence satisfies the accepted daily freshness window. | `appview/cmd/appview/follower_growth_worker_test.go` |
| IT-006 | FR-007, FR-008 | AC-013, AC-014 | Observe joins, departures, rejoins, follows, and unfollows only on later capture. | Baseline capture, then mutate membership/follow projection between fixed dates. | Inspect rows before a new capture, then capture the next UTC date. | Event-time mutation creates no analytics row; join/departure/rejoin and follow changes appear only in the next successful canonical snapshot; old gaps remain absent. | `appview/internal/followergrowth/store_integration_test.go` |
| IT-007 | BR-001, FR-009, FR-010 | AC-002, AC-015, AC-016, AC-017 | Verify owner route, policy, query validation, and JSON contract. | Alice with history, Bob with no history, current/departed auth fixtures, standard policy mux. | Request each period as Alice and Bob; omit auth/device; use departed member, unsupported/missing/repeated period, arbitrary-profile path. | Both valid owners receive successful owner-only camelCase responses, including Bob's all-null no-history contract; invalid calls return canonical `{error,message,requestId}` errors; route uses current-member read policy. | `appview/internal/routes/follower_growth_route_test.go` |
| IT-008 | BR-003, FR-011, FR-012, FR-015, RULE-002, NFR-003 | AC-004, AC-005, AC-018, AC-019, AC-022, AC-034 | Read sparse bounded periods and global availability from Postgres. | Earliest owner snapshot before one selected range, snapshots around other range starts, leading/interior gaps, no-history owner, and current date 2028-02-29. | Read `7d`, `30d`, and `1y` at fixed current dates. | Exact inclusive UTC dates are chronological, gaps are null, `availableFrom` is the earliest retained owner snapshot regardless of range or null for no history, 2028-02-29 starts at 2027-02-28 with 367 points, and no cursor exists. | `appview/internal/followergrowth/store_integration_test.go` |
| IT-009 | FR-013, NFR-002 | AC-020 | Refuse live follower overlay. | Persist yesterday's snapshot, then change active follows today. | Request Growth before next capture. | Latest count/date/capture time remain from persisted history and differ legitimately from current profile count. | `appview/internal/routes/follower_growth_route_test.go` |
| IT-010 | RULE-003 | AC-030 | Preserve snapshots during ordinary retention. | Old snapshots older than all existing retention windows. | Run ordinary retention jobs or inventory assertions. | Growth snapshots remain and are not registered for ordinary pruning. | `appview/internal/followergrowth/retention_test.go` |
| IT-011 | RULE-004 | AC-031 | Remove private growth history on permanent owner deletion. | Owner snapshots plus terminal owner-lifecycle/deletion fixtures. | Complete permanent account deletion and terminal purge. | Owner snapshot rows are absent and unreadable; purge inventory/catalogue includes the table and reports completion. | `appview/internal/ownerlifecycle/follower_growth_purge_integration_test.go` |
| IT-012 | FR-016 | AC-023 | Preserve Settings navigation on compact and large layouts. | Production router/provider harness at compact and large widths. | Open Growth from Settings and navigate back. | Typed Growth location is canonical, one appropriate shell/navigation control remains, and back returns to Settings with correct selection. | `app/test/router/settings_routes_test.dart` |
| IT-013 | NFR-007 | AC-038 | Verify real failure/success telemetry remains low-cardinality. | Capturing slog handler/metric recorder; successful, duplicate, failed run and API error. | Execute worker/store and route paths. | Telemetry reports bounded outcome, duration/age/count values as measurements only; dimensions and logs contain no DID, handle, per-account count, or snapshot row. | `appview/internal/followergrowth/observability_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test | Automation Target |
|---|---|---|---|---|---|
| REG-001 | Public profile follower count remains Craftsky-member scoped. | RULE-001 | AC-009 | Existing profile count and new snapshot count return the same result for mixed current/non-current relationships. | `appview/internal/api/profile_store_test.go` |
| REG-002 | Follow and membership indexers only maintain canonical projections and do not synchronously write analytics. | FR-008 | AC-014 | Create/delete follow and change membership; assert no growth row until the snapshot service runs. | `appview/internal/index/follower_growth_boundary_test.go` |
| REG-003 | Feed, recommendation/ranking, discovery, moderation, and advertising boundaries do not depend on private growth snapshots. | RULE-005 | AC-032 | Add an executable architecture inventory that permits follower-growth reads only in the owner growth route/service and deletion path; assert feed/timeline, ranking/recommendation, search/discovery, moderation, and advertising code has no growth-store dependency or query. Where a named subsystem does not exist, assert that absence in the inventory so adding it requires an explicit boundary update. | `appview/internal/routes/follower_growth_architecture_test.go` |
| REG-004 | Growth remains absent from public profile responses and public-profile routes. | RULE-005 | AC-032 | Marshal/read another member's profile and assert no growth series, snapshot date, or net-change fields are exposed. | `appview/internal/api/profile_response_test.go`, `appview/internal/routes/follower_growth_route_test.go` |
| REG-005 | Permanent deletion analytics cleanup does not introduce PDS/public-follow deletion. | RULE-004 | AC-031 | Complete owner snapshot purge and assert public relationship records retain existing deletion-flow behavior. | `appview/internal/ownerlifecycle/follower_growth_purge_integration_test.go` |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Membership-scoped canonical graph | Current members Alice, Bob, Carol; external Dana; active follows including current-to-current and external edges; zero-follower member. | AT-007, IT-002, REG-001 |
| TD-002 | Sparse daily history | Fixed UTC today; earliest snapshot before one selected range; leading and interior absent dates; observations `10, 12, null, 11`; no-history owner; capture timestamps; yesterday-latest variant. | AT-002, AT-003, UT-002, UT-003, IT-007, IT-008, IT-009 |
| TD-003 | Period boundaries | Ordinary date, Jan 1 boundary, current date 2028-02-29 with expected start 2027-02-28, and dates exactly inside/outside 7d/30d/1y ranges. | UT-001, IT-008 |
| TD-004 | API authorization | Alice and Bob current sessions/device IDs, departed account, no session, unsupported/missing/repeated period inputs. | AT-004, IT-007 |
| TD-005 | Concurrency and rollback | Three current members, deterministic capture date, two independent pools, injected constraint/transaction failure. | IT-001, IT-004 |
| TD-006 | Flutter history states | Loading completer; retryable exception; empty; one point; partial; interior gap; positive, negative, flat, and populated responses for all periods. | AT-001, AT-003, UT-007, UT-009, UT-010 |
| TD-007 | Account-switch privacy | Account A delayed response with count 41; account B immediate response with count 7; distinct account keys and repositories. | AT-005, UT-008 |
| TD-008 | Deletion and retention | Owner snapshots across old/recent dates, terminal lifecycle rows, unrelated owner's snapshots, and public follow projection rows. | IT-010, IT-011, REG-005 |
| TD-009 | Observability secrets | Synthetic DID, handle, follower counts, raw snapshot payload, known failure text, capturing log/metric sinks. | UT-012, IT-013 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-005 | AC-036 | Real screen-reader traversal and non-colour comprehension. | On one supported mobile platform, enable VoiceOver or TalkBack; open populated and gapped Growth fixtures; traverse heading, summary, controls, graph, freshness, and gap explanation; enable a colour-vision filter or grayscale. | Reading order is coherent, controls announce selection, the graph has a concise meaningful summary, missing dates and trend remain understandable, and no required meaning depends only on colour. |
| MAN-002 | NFR-004, NFR-006 | AC-035, AC-037 | Final chart legibility across representative form factors and locale direction. | View compact phone, large tablet/desktop, text scale 200%, dark/light themes, and one RTL locale after the chart implementation is chosen. | Labels remain legible, plotting is not clipped, controls remain usable, dates/counts are localized, and UTC bucket order/identity is unchanged. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Production-scale snapshot duration and lock impact are not reproducible in focused tests. | FR-004, FR-005, NFR-002 | Isolated fixtures prove set-based shape and atomicity but not launch-scale cardinality or production hardware behavior. | Add a repeatable benchmark/load fixture in coding design, inspect `EXPLAIN (ANALYZE, BUFFERS)` outside normal tests, and alert on run duration/snapshot age. |
| GAP-002 | Flutter semantics tests cannot prove behavior in every OS screen-reader/version combination. | NFR-005 | Framework semantics assertions do not cover platform speech, navigation gestures, or chart-package platform quirks. | Complete MAN-001 on a release candidate and retain automated semantics coverage as the regression gate. |
| GAP-003 | Exact chart package behavior is unknown until coding design selects an implementation. | NFR-004, NFR-005, NFR-006 | The requirements intentionally defer dependency selection. | Revalidate UT-009, UT-010, MAN-001, and MAN-002 against the selected implementation; reject packages that cannot expose required semantics or RTL behavior. |
| GAP-004 | The exact worker polling cadence and empty-run status model remain design decisions. | FR-003, NFR-002, NFR-007 | Requirements fix startup behavior and freshness, not timer geometry or a run-status table. | Coding design must inject clock/timer boundaries and define the freshness metric source before implementing UT-005, IT-005, and IT-013. |

No blocking test-design gaps were identified. Medium-risk document review remains recommended before coding.

## 10. Out Of Scope

- Live, intraday, gained/lost, churn, conversion, or event-ledger tests because those behaviors are explicit non-goals.
- Global AT Protocol follower reconciliation because the metric is intentionally Craftsky-member scoped.
- Historical backfill or reconstruction tests for pre-activation and missed dates; tests instead require explicit null gaps.
- Device-timezone rebucketing, custom ranges, previous-window navigation, and saved period selection.
- Public or arbitrary-profile analytics access, entitlement, advertising, ranking, or recommendation behavior beyond regression non-interference checks.
- Lexicon/PDS record validation because this feature adds private derived AppView storage only.
- Client product-analytics events because none are required.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-08-25-follower-growth-metrics/01-requirements.md`
- Test specification: `docs/changes/2026-08-25-follower-growth-metrics/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-08-25-follower-growth-metrics/03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-25-follower-growth-metrics/`
- Recommended first failing test for implementation: IT-001, the Postgres schema invariant test for one non-negative snapshot per profile DID and UTC date.
- Suggested test order for implementation: IT-001; IT-002 and REG-001; IT-004; UT-001 through UT-005; IT-005 through IT-009; UT-004 and IT-007; IT-010 and IT-011; UT-006 through UT-012; AT-001 through AT-006; regression and manual checks.
- Commands discovered: `just appview-test-unit` for fast Go feedback; `just test` with the dev stack for real-Postgres coverage; `just appview-check` for the release-equivalent AppView gate; `just app-test` or `just app-test test/settings/follower_growth_page_test.dart` for Flutter tests; `just app-analyze` for Flutter static analysis.
- Blocking gaps: None.
- Review recommendation: Proceed to document review because atomic all-member capture, private account-bound state, deletion integration, and accessible chart behavior make this a medium-risk cross-stack change.
