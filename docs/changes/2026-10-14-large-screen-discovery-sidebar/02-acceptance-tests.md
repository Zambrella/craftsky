# Acceptance Test Specification: Large-screen discovery sidebar

## 1. Test Strategy

Source: `01-requirements.md`, approved by the user with “LGTM. Move to $write-acceptance-tests” on 2026-10-14. Its assumptions remain the baseline. Both earlier documents were subsequently revised at the user's request to address `03-document-review.md`; stable requirement, acceptance-criterion and test IDs are preserved.

Risk level: Medium. Recommend document review before coding planning.

All tests below are planned, not implemented or executed. Automate user-visible scenarios with Flutter widget tests using the real router, provider overrides and recording repositories. Use Go/Postgres integration tests for member selection, visibility and wire contracts. Use pure unit tests only where logic naturally exists; do not extract production helpers just to mirror the implementation in tests. Reserve manual checks for visual quality and device assistive technology.

Existing conventions:

- Flutter: `flutter_test`, Riverpod `ProviderScope` overrides, `FakeProjectRepository`, signed-in/out auth fakes, localized `AppTheme` harnesses, and `app/test/test_support/deterministic_pump.dart`.
- Use controlled futures and bounded pumps for loading, failure and account-switch races; complete or cancel outstanding test futures during teardown. Avoid settling indefinitely on loading animations.
- Go: `testing`, `httptest`, `testdb.WithSchema`, route policy tests, and real Postgres fixtures. Relevant examples are `appview/internal/routes/onboarding_route_test.go` and API search/profile store tests.
- Proposed new file paths below are automation targets, not files created in this stage. Endpoint names, query keys and provider names remain coding-plan decisions.

## 2. Requirement Coverage Matrix

“Planned” means automation is specified but coverage does not exist yet.

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-002, AC-003 | AT-002, AT-003, UT-001 | Acceptance / Unit | Planned |
| BR-002 | AC-005, AC-006 | AT-005, IT-001 | Acceptance / Integration | Planned |
| FR-001 | AC-001 | AT-001, REG-001 | Acceptance / Regression | Planned |
| FR-002 | AC-002 | AT-002, UT-001 | Acceptance / Unit | Planned |
| FR-003 | AC-003 | AT-003, UT-002, REG-002, REG-003 | Acceptance / Unit / Regression | Planned |
| FR-004 | AC-004 | AT-004 | Acceptance | Planned |
| FR-005 | AC-005 | AT-005, IT-002 | Acceptance / Integration | Planned |
| FR-006 | AC-008 | AT-006 | Acceptance | Planned |
| FR-007 | AC-009 | AT-007, IT-003 | Acceptance / Integration | Planned |
| NFR-001 | AC-010 | AT-008, MAN-001, MAN-002 | Acceptance / Manual | Planned + manual |
| RULE-001 | AC-006 | IT-001, IT-004 | Integration | Planned |
| RULE-002 | AC-007 | IT-002, IT-003 | Integration | Planned |
| RULE-003 | AC-011 | IT-005, IT-006 | Integration | Planned |

## 3. Acceptance Scenarios

### AT-001: Sidebar respects the existing responsive scope

Requirement IDs: FR-001

Acceptance Criteria: AC-001

Priority: Must

Level: Acceptance (widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed), reusing the shell harness patterns.

```gherkin
Scenario Outline: Sidebar follows the existing desktop boundary
  Given an authenticated shared shell at <width> logical pixels
  When I open a route that currently uses that shell
  Then sidebar visibility is <visible>
  And when visible Your crafts appears above New to CraftSky
  And the empty placeholder card is absent
  And the main content and navigation remain usable

  Examples:
    | width | visible |
    | 600   | no      |
    | 900   | no      |
    | 1200  | no      |
    | 1201  | yes     |
    | 1440  | yes     |
```

Parameterize across existing shared-shell destinations, not just Feed; derive the route set from the shell test harness rather than inventing new routes.

### AT-002: Show only the active user's preferred crafts

Requirement IDs: BR-001, FR-002

Acceptance Criteria: AC-002

Priority: Must

Level: Acceptance (widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed).

```gherkin
Scenario: Preferred crafts form a distinct supported list
  Given my profile selects supported crafts in reverse catalog order
  And it includes a duplicate selection and an unsupported craft
  When the loaded sidebar appears
  Then each supported selected craft appears exactly once in catalog order
  And each uses the existing craft label
  And unselected and unsupported crafts do not appear as links
```

### AT-003: Craft links select and query the requested Projects tab

Requirement IDs: BR-001, FR-003

Acceptance Criteria: AC-003

Priority: Must

Level: Acceptance (router/widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed), with recording `FakeProjectRepository`; extend `app/test/projects/pages/projects_page_test.dart` as needed.

```gherkin
Scenario Outline: Select a craft from either navigation state
  Given <starting state>
  When I activate the sidebar link for Knitting
  Then Projects is the active destination
  And the Knitting tab is selected
  And the active project query filters by the Knitting catalog identity
  When my preference data subsequently reorders the tabs
  Then Knitting remains selected and drives the active project query

  Examples:
    | starting state                                |
    | Feed is open and Projects has not been mounted |
    | Projects is already showing another craft     |
```

Also activate a second craft and then the same craft again on the mounted page. Assert the final selected tab and query, not just the route location. For an empty craft response, assert the requested tab remains selected with the existing Projects empty state (EC-002).

### AT-004: Configure crafts from an empty preference list

Requirement IDs: FR-004

Acceptance Criteria: AC-004

Priority: Must

Level: Acceptance (widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed), using the existing profile-edit flow with fake persistence.

```gherkin
Scenario: Choose crafts without restarting
  Given my profile has loaded without supported selected crafts
  When I activate Choose your crafts
  Then the existing profile craft-editing flow opens
  When I successfully save a supported craft selection
  Then its sidebar link appears without restarting the app
```

Run with an empty array and an unsupported-only array; a pending profile must show loading rather than this prompt.

### AT-005: Member rows are compact profile links

Requirement IDs: BR-002, FR-005

Acceptance Criteria: AC-005

Priority: Must

Level: Acceptance (widget/router)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed).

```gherkin
Scenario: Open a recent member's profile
  Given the member read returns five eligible public summaries
  And one summary has no display name and another has no avatar
  When New to CraftSky appears
  Then five distinct rows appear in response order
  And each row shows the appropriate name, handle and avatar or existing fallback
  When I activate a member row
  Then the existing profile flow opens that member's identity
```

Parameterize counts 1, 4 and 5. If the target becomes unavailable before activation, assert the existing profile-unavailable behavior rather than a crash (EC-005). Server cardinality and eligibility are tested separately by IT-002.

### AT-006: Each section recovers independently

Requirement IDs: FR-006

Acceptance Criteria: AC-008

Priority: Must

Level: Acceptance (widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed).

```gherkin
Scenario Outline: Local failure does not block the rest of the screen
  Given <section> is waiting on a controlled read
  And the other section has loaded useful content
  Then only the pending section shows its loading state
  And the other section and main-page navigation remain usable
  When the pending read fails
  Then that section shows an error with a retry action
  When I retry and the replacement read succeeds
  Then that section shows the returned content

  Examples:
    | section |
    | crafts  |
    | members |
```

Add cases for simultaneous loading, simultaneous failure, and successful zero-member results. Zero members must produce a brief empty state rather than an error or empty bordered card. Keep assertions localized to sections.

### AT-007: Account switches discard prior viewer results

Requirement IDs: FR-007

Acceptance Criteria: AC-009

Priority: Must

Level: Acceptance (widget/provider integration)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed), reusing account-switcher fakes.

```gherkin
Scenario: Late responses belong to the previous account
  Given account A has visible sidebar content and outstanding reads
  When I switch to account B
  Then A's viewer-specific content is no longer displayed
  When B's profile and member reads complete
  Then B's preferences and eligible member rows appear
  When A's outstanding reads complete later
  Then the sidebar still displays only B's results
```

Test both completion orders and a successful preference update without switching accounts. Distinct fixtures must make stale results observable.

### AT-008: Actions remain accessible in constrained wide layouts

Requirement IDs: NFR-001

Acceptance Criteria: AC-010

Priority: Must

Level: Acceptance (widget)

Automation Target: `app/test/router/app_shell_discovery_sidebar_test.dart` (proposed).

```gherkin
Scenario: Reach every sidebar action with enlarged text
  Given a desktop viewport 1201 logical pixels wide and 600 high
  And enlarged text, long member names and all supported preferred crafts
  When I navigate through the sidebar using keyboard focus and scrolling
  Then every link and retry action can be reached and activated
  And there are no layout overflow exceptions or clipped unreachable actions
  And semantics identify each action and its destination
```

Repeat with 1440 by 900, normal and 2x text scaling, plus loading/error states. Assert section copy against localization resources rather than hardcoded implementation strings. Use the supported locale configuration; do not require adding a language. Platform screen-reader presentation remains MAN-002.

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-001, FR-002 | AC-002 | Preferred-only craft derivation, if implemented as pure logic/provider | TD-001, empty and unsupported-only variants | Exactly the supported selected set in catalog order, existing labels; does not include all remaining catalog crafts | `app/test/projects/providers/preferred_craft_links_test.dart` (proposed; final location depends on coding plan) |
| UT-002 | FR-003 | AC-003 | Craft navigation boundary validation | Valid catalog identities; missing, malformed and unsupported values | Valid identities select the matching craft; invalid/absent input uses the existing default without crashing or writing preferences (EC-003) | Existing router/project tests or proposed route-input unit suite, depending on design |

If no separate pure logic is introduced, cover UT-001/UT-002 inputs through widget/router tests rather than introducing helpers solely for tests. Database selection rules belong in integration tests, not mocked SQL tests.

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-002, RULE-001 | AC-006 | Newest ordering and ties | TD-002 with fixed onboarding times, conflicting profile indexing dates, account ages and popularity; two equal timestamps | Read repeatedly; alter unrelated profile/indexing data; read again | Same five newest identities in descending join order with repeatable tie ordering; unrelated attributes never change recency | `appview/internal/api/recent_members_store_test.go` (proposed) |
| IT-002 | FR-005, RULE-002 | AC-005, AC-007 | Exclude before limit and preserve cardinality | TD-002/TD-003: newest candidates include every exclusion class and at least six older eligible members | Read as viewer; repeat with eligible counts 0, 1, 4, 5 and 6 | Viewer and excluded accounts absent; exactly min(5, eligible count) distinct rows, including older eligible rows needed to fill five; empty is successful | `appview/internal/api/recent_members_store_test.go` (proposed) |
| IT-003 | FR-007, RULE-002 | AC-009, AC-007 | Eligibility is viewer-specific | Two viewers with opposite block/mute relationships to candidates; one already-followed eligible candidate | Read consecutively as A, B, then A; update a relationship and read again | Each response applies its viewer's rules with no cross-account caching; followed accounts remain eligible per ASM-003 | `appview/internal/api/recent_members_store_test.go` (proposed) |
| IT-004 | RULE-001 | AC-006 | First joining is not rewritten | TD-005 with older completion, repeat completion, profile update/reindex and missing completion | Exercise supported local onboarding/profile paths and then member read | Repeat completion/profile changes preserve effective first-join order; missing reliable metadata is omitted per ASM-002; no fabricated recent date | Existing onboarding route tests plus proposed recent-member store tests; see GAP-001 |
| IT-005 | RULE-003 | AC-011 | Authentication and route contract | Real route mux using existing authenticated test setup | Request without/with valid viewer credentials and exercise existing route-policy checks | Unauthenticated requests rejected; valid viewer gets standard success contract; new operation uses established `/v1/`, read policy and camelCase/error conventions | `appview/internal/routes/recent_members_route_test.go` (proposed) |
| IT-006 | RULE-003 | AC-011 | Serialized summaries contain only permitted public fields | Public fields plus unmistakable private canaries in local fixtures | Serialize successful response and induced error response through HTTP handler | Positive public-field assertions pass; credentials, private payloads and unnecessary onboarding metadata absent; error has standard envelope and permitted references | Proposed recent-member route tests; client decoding tests under `app/test/profile/data/` if a new client contract is introduced |

Use the project's real visibility/lifecycle fixtures and policies when implementing exclusions. Do not invent moderation-state values or rely solely on a mocked eligible list. Exact tie-break rule and public summary schema must be fixed in coding planning so assertions have an independent expected result.

### Required variant checklist (DR-004)

Each row is mandatory planned coverage under its existing test ID and inherits that test's requirement and AC references. Parameterized cases should be separately named so failures identify the variant. These are not optional prose suggestions.

| Test ID | Required variants | Independent oracle |
|---|---|---|
| AT-003 | First navigation from Feed; already mounted Projects on another craft; select a second craft; reselect the same craft; late preference reorder; successful empty craft results | Explicit requested craft identity, visible selected tab and recording repository query; preferences remain unchanged |
| AT-006 | Crafts pending with members ready; members pending with crafts ready; each section fails/retries independently; both pending; both failed; successful zero-member response | Local section state, usable other-section actions/main navigation, one recovered section at a time, successful empty state distinct from error |
| AT-007 | Switch with cached A content; A completes before B; A completes after B; late A profile and member reads independently; successful preference update without switching | Distinct A/B fixture identities and selections; no A viewer-specific content after switch; only the active profile's saved changes appear |
| IT-002 | Eligible counts 0, 1, 4, 5, 6; each exclusion class individually newer than eligible members; all exclusion classes combined ahead of at least six eligible candidates; all candidates excluded | Handwritten expected eligible identities and min(5, eligible count); exclusions applied before limit; no duplicate identities |
| IT-003 | Reads A, B, A with opposite relationships; viewer block, mute and blocking-viewer exclusions; relationship update then next read; already-followed eligible candidate | Explicit per-viewer expected sets, updated on the next read; followed status alone does not exclude |
| IT-004 | Initial completion then repeated completion; profile edit; reindex; missing completion metadata; relevant lifecycle transitions identified during planning | Fixed original join time/order; missing metadata omitted; lifecycle-hidden accounts absent; do not assume timestamp behavior before verifying it |

### Contract finalization checklist (DR-001, DR-002)

Before implementing the member data path, the coding plan must provide:

1. Source-backed verification of onboarding writes, repeat completion and lifecycle semantics, plus the missing-metadata policy. Distinguish local fixture coverage from any actual legacy-data evidence; actual legacy coverage is currently unverified.
2. The method/path, auth/access policy, exact public-summary field allowlist and missing-field behavior, and success/error wire examples. IT-005/IT-006 must assert that contract, including a schema-aware check that unknown/private fields cannot leak.
3. A deterministic tie-break rule and a fixture whose tied identities have an explicit expected order, including a tie crossing the fifth-member cutoff. IT-001 must assert the same five identities across repeated reads and unrelated profile/indexing changes.
4. Final test file locations and fixtures, preserving existing IDs. If API reuse avoids a new route, apply the same behavioral contract tests to the reused operation.

These are required planning outputs, not additional product features or claims that verification has already occurred. A metadata discrepancy must be resolved before member selection is implemented; no production backfill is authorized.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Compact navigation, desktop navigation and main content | FR-001 | AC-001 | Retain applicable assertions in `app/test/router/app_shell_navigation_menu_test.dart`; update only placeholder-specific expectations; exercise existing destinations/composer and breakpoint transitions |
| REG-002 | Projects default craft ordering without explicit navigation | FR-003 | AC-003 | Existing `app/test/projects/pages/projects_page_test.dart`: opening Projects normally retains preference-aware default behavior and chronological query defaults |
| REG-003 | Explicit craft choice survives late preference reorder | FR-003 | AC-003 | Extend existing Projects tests with late profile completion; assert selected craft identity and query after reorder rather than index alone |
| REG-004 | Existing account-switcher navigation and identity presentation | FR-007 | AC-009 | Run `app/test/router/app_shell_account_switcher_test.dart`; sidebar loading/failure must not prevent switching or alter the active account |

## 7. Test Data

Use local synthetic fixtures only. Fixed UTC times and stable valid identities make expected ordering independent of the wall clock.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Preference derivation | Two supported selections in reverse catalog order, one duplicate, one unknown token, unselected catalog entries; empty/unknown-only/all-supported variants | AT-002, AT-003, AT-004, AT-008, UT-001, UT-002 |
| TD-002 | Recency and cardinality | At least six eligible members, distinct fixed join times including a tie; names/handles/avatars; newer indexing dates on older members; divergent account ages/popularity | AT-005, IT-001, IT-002 |
| TD-003 | Visibility exclusions | Viewer, viewer-blocked, viewer-muted, blocking-viewer, and each existing moderation/lifecycle-hidden class, all newer than eligible candidates; separate public eligible rows | IT-002, IT-003 |
| TD-004 | Async and account isolation | A and B with disjoint preferences and distinguishable member results; controlled reads with success/error/retry and both completion orders | AT-006, AT-007, IT-003 |
| TD-005 | Join lifecycle | Older immutable first completion, later repeat completion, recently reindexed existing member, profile without completion metadata | IT-004 |
| TD-006 | Presentation and privacy | Missing avatar/name, long names and handles, all craft labels; public summary fields and unique private canaries | AT-005, AT-008, IT-006, MAN-001, MAN-002 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-001 | AC-010 | Tablet/desktop visual quality | Inspect on the reported iPad large format and a desktop viewport, including short height, long names, large text and light/dark themes supported by the app; scroll to the last action | Balanced spacing, readable labels, recognizable avatars, no clipping or competing main-content/side-column scroll gestures |
| MAN-002 | NFR-001 | AC-010 | Platform assistive technology | Use VoiceOver on iPad to visit sections, craft links, member rows and retry; activate representative links | Sensible reading order, meaningful announced destinations, correct navigation and no trapped focus |

Manual checks supplement automated layout/semantics tests; they do not substitute for membership/visibility assertions.

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Join metadata immutability and legacy coverage are not verified | RULE-001, BR-002 | Requirements use first onboarding completion, but lifecycle behavior/data coverage still needs verification | Complete the source-backed membership verification deliverable in requirements section 15 and the contract checklist above; finalize the IT-004 lifecycle variants. If semantics do not support the approved behavior, resolve the design before implementing member selection; no production backfill is authorized |
| GAP-002 | Discovery API and response schema are not selected | RULE-003, FR-005 | Requirements intentionally defer interface design | Complete the contract checklist above: operation/access policy, allowed fields/fallbacks, wire examples, tie-breaker and explicit cutoff-tie oracle; finalize IT-001/IT-005/IT-006 and any client decoding target without changing behavioral IDs |
| GAP-003 | No feature test execution evidence exists yet | All Must requirements | This stage designs tests only | Implement and run the planned tests during TDD; do not report planned coverage as passing coverage |
| GAP-004 | Real-device visual and VoiceOver evidence unavailable in widget tests | NFR-001 | Platform presentation and gestures differ from the test renderer | Complete MAN-001/MAN-002 after implementation and record device/context and observations |

There are no blocking gaps for document review. GAP-001 is an implementation dependency; GAP-002 must be resolved in coding planning. Risk remains Medium.

## 10. Out Of Scope

- Follow buttons, engagement ranking, trending, spotlights, member pagination/directory, drafts and reply panels: excluded by the requirements.
- Lexicon changes, preference persistence redesign, production deployment/backfill and breakpoint changes: outside this feature.
- Continuous polling/live-refresh tests: not required under ASM-004.
- New diagnostics infrastructure or exhaustive observability tests: no new diagnostic behavior is required. If implementation changes diagnostics, add guide-required serialized sink/canary verification to the later plan.
- Separate end-to-end suite duplicating every widget test: the scoped workflows have practical router/widget and real server integration targets. Manual device checks cover the remaining platform behavior.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md` (user-approved baseline; approval status corrected and planning deliverables clarified in this revision).
- Test specification: `02-acceptance-tests.md` (reviewed with notes; revised at the user's request to address DR-001–DR-004).
- Review artifact: `03-document-review.md` (existing review of the prior document versions; retained unchanged as the review record).
- Findings disposition: DR-001 → explicit membership verification deliverable and IT-004 variants; DR-002 → contract finalization checklist and independent tie/schema oracles; DR-003 → corrected approval status in both earlier documents; DR-004 → mandatory variant checklist. Outstanding verification remains visible in GAP-001/GAP-002.
- External Plannotator review, if initiated by the user: `docs/changes/2026-10-14-large-screen-discovery-sidebar/`.
- Recommended first failing test: AT-003, asserting that a craft link changes both the Projects selected tab and active query on an already mounted page.
- Suggested order: craft navigation and preferred-only derivation (AT-002/AT-003, UT-001/UT-002); empty craft editing (AT-004); server membership order/visibility/contracts (IT-001–IT-006); member rows and independent states (AT-005/AT-006); account races (AT-007); layout/accessibility (AT-001/AT-008); regressions and manual checks.
- Commands discovered: `just app-test test/projects/pages/projects_page_test.dart`; `just app-test test/router/app_shell_navigation_menu_test.dart test/router/app_shell_account_switcher_test.dart`; once created, `just app-test test/router/app_shell_discovery_sidebar_test.dart`; `just app-analyze`; `just test` for Go with local services and race checks; `just appview-check` for the full backend gate. `./scripts/appview-test unit` deliberately skips Postgres/MinIO integration and cannot verify IT-001–IT-006.
- Service context: use the repository compose stack (`just dev`); Go tests run on the host. Never start AppView via `go run`.
- Blocking gaps: None for coding planning. GAP-001/GAP-002 are explicit plan-completion dependencies for the member data path; carry every mandatory variant above into the implementation checklist.
- No tests, migrations, source edits, commits or pushes were performed in this stage.
