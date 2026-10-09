# Acceptance Test Specification: Service Status and Maintenance Messaging

## 1. Test Strategy

Design status communication tests against the approved `01-requirements.md`, including the 2026-10-09 simplifications. Risk remains **Medium**. This document specifies future tests; no executable tests, implementation, provisioning or live changes have been performed at this stage.

Use Flutter unit/controller tests for contract validation, polling, freshness and revision decisions; widget acceptance tests for presentation and retained state; integration tests for anonymous HTTP, persistence, operator commands, serialized diagnostics and hosted delivery. Protect existing auth/account/composer and website behavior with focused regressions. Limit manual checks to real assistive-technology usability. Each scenario's automation target is proposed unless marked existing; target paths do not prescribe production architecture.

Repository conventions inspected read-only:

- `app/test/app_test.dart`: `flutter_test`, Riverpod overrides, delayed initialization and mocked SharedPreferences.
- `app/test/auth/widgets/active_account_initialization_gate_test.dart`, `app/test/router/account_switch_routing_test.dart`, and composer tests: existing account gates, routing and editor fixtures.
- `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart`: existing sign-out contract.
- `app/test/observability/sdk_emission_test.dart` and `app/test/test_support/serialized_sentry_transport.dart`: serialized SDK evidence with positive assertions and protected canaries.
- `app/integration_test/critical_journeys_test.dart` and its support harness: production mobile app/providers/HTTP path with loopback services, real preferences/plugins, no production credentials. This suite supports Android/iOS, not web/desktop.
- `scripts/test_web_deploy.py` and `scripts/test_cloudflare_pages_build.py`: Python `unittest`, temporary directories and mocked publication/network calls.

Use one controlled-time controller suite for FR-004/FR-008, then cross-link it to NFR-001 and RULE-001. Do not repeat timing tests in each UI suite or use real multi-minute sleeps. Acceptance tests can inject accepted statuses without retesting the parser. Use a separate loopback status service from AppView and assert actual outbound requests. Hosted browser/CORS/cache tests require an isolated Cloudflare resource and browser harness; mocked headers do not prove live delivery.

Contract fixture keys/revision representation, finite bounds and Cloudflare/cache choices remain implementation design decisions. Describe fixtures semantically until these are selected; do not require publication time, a persisted maintenance cache, clock reconciliation across restarts, or the earlier illustrative size limits. See GAP-001–GAP-003.

## 2. Requirement Coverage Matrix

All 20 active Must requirements and all 17 active acceptance criteria have planned coverage. “Yes” means intended automation, not executed evidence. Withdrawn FR-010/FR-011 and AC-013/AC-014 stay reserved.

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003, AC-004 | AT-001, AT-002, AT-003 | Acceptance | Yes |
| BR-002 | AC-002, AC-015 | AT-008, IT-003, IT-004, IT-005 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-002, AC-015 | AT-001, AT-008, IT-005, IT-007 | Acceptance / Integration | Yes; hosted execution pending |
| FR-002 | AC-003, AC-004, AC-005 | AT-002, AT-003, UT-001, UT-002, UT-006 | Acceptance / Unit | Yes |
| FR-003 | AC-005 | UT-001, UT-002, UT-004, IT-003 | Unit / Integration | Yes |
| FR-004 | AC-006, AC-007 | UT-003, UT-004, AT-006 | Unit / Acceptance | Yes |
| FR-005 | AC-001, AC-003, AC-008 | AT-001, AT-002, AT-004 | Acceptance | Yes |
| FR-006 | AC-004, AC-009 | AT-003, AT-007, UT-005, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-010 | AT-005, AT-007, REG-001 | Acceptance / Regression | Yes |
| FR-008 | AC-007, AC-011 | UT-004, AT-006, AT-007 | Unit / Acceptance | Yes |
| FR-009 | AC-008, AC-012 | AT-004, AT-006, REG-002, REG-003 | Acceptance / Regression | Yes |
| FR-012 | AC-002, AC-015 | AT-008, IT-003, IT-004, IT-005 | Acceptance / Integration | Yes |
| NFR-001 | AC-005, AC-006, AC-010, AC-011 | UT-002, UT-003, UT-004, AT-005, AT-007 | Unit / Acceptance | Yes |
| NFR-002 | AC-016 | IT-004, IT-007 | Integration | Yes; hosted execution pending |
| NFR-003 | AC-017 | UT-006, AT-002, AT-009, MAN-001 | Unit / Acceptance / Manual | Mixed |
| NFR-004 | AC-018 | IT-006 | Integration | Yes |
| RULE-001 | AC-003, AC-010, AC-011 | AT-002, AT-005, UT-004, UT-006 | Acceptance / Unit | Yes |
| RULE-002 | AC-019 | IT-001, IT-003, IT-006 | Integration | Yes |
| RULE-003 | AC-005, AC-019 | UT-002, UT-007, IT-001, AT-002 | Unit / Integration / Acceptance | Yes |
| RULE-004 | AC-008, AC-012 | AT-004, AT-006, REG-001, REG-002, REG-003 | Acceptance / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Maintenance before account initialization succeeds

Requirement IDs: BR-001, FR-001, FR-005
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_app_test.dart` (proposed), using existing app initialization fixtures

```gherkin
Feature: Independently delivered service status
  Scenario Outline: Launch during backend downtime
    Given a <session> client with AppView and database access unavailable
    And dependency or account initialization is pending or has failed
    When the anonymous status service returns valid maintenance
    Then the custom whole-app maintenance screen is visible
    And successful sign-in or account initialization is not required to see it

    Examples:
      | session    |
      | signed-in  |
      | signed-out |
```

IT-001 verifies the separate anonymous request boundary; IT-007 verifies Cloudflare delivery. This widget scenario does not pretend to prove either external boundary.

### AT-002: Maintenance presentation and informational estimate

Requirement IDs: BR-001, FR-002, FR-005, NFR-003, RULE-001, RULE-003
Acceptance Criteria: AC-003, AC-017, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_widgets_test.dart` (proposed)

```gherkin
Feature: Maintenance presentation
  Scenario: Show a public explanation and retry
    Given fresh maintenance with English title and message
    And an optional recovery estimate
    When the status is rendered
    Then its title and message appear as plain text in CraftSky themed UI
    And Try again uses the selected built-in locale
    And the optional estimate is labelled and shown in local time
    And there is no dismiss or external-link action
    And back navigation and tapping covered controls cannot access normal UI
    When the user selects Try again
    Then a status refresh is requested
```

Parameterize estimate present/absent and HTML-like prose. UT-003 owns retry coalescing/timeout; UT-006 owns estimate formatting and passing the estimate without a mode change.

### AT-003: One non-blocking announcement

Requirement IDs: BR-001, FR-002, FR-006
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_app_test.dart` (proposed)

```gherkin
Feature: General announcements
  Scenario: Use the app while a notice is visible
    Given the normal app is ready
    When an announcement is accepted
    Then one dismissible general notice appears
    And navigation and ordinary operations remain available
    When normal or maintenance replaces the announcement
    Then the previous announcement disappears
    And only the current mode is presented
```

Unknown scheduling/feature/link fields must not create extra notices or future transitions (UT-001/UT-007). Persistence is covered by AT-007 and IT-002.

### AT-004: Immediately cover an active editor without discarding work

Requirement IDs: FR-005, FR-009, RULE-004
Acceptance Criteria: AC-008, AC-012
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_editor_test.dart` (proposed), reusing post/project composer fixtures

```gherkin
Feature: Retained work during maintenance
  Scenario Outline: Maintenance arrives while composing
    Given the current account has an open <editor> with unsent text and media
    And navigation state and one delayed operation exist
    When fresh maintenance is accepted
    Then maintenance covers the UI immediately without waiting for that operation
    And the editor, navigation and current session remain retained
    And no extra write is submitted
    When the operation completes under its existing outcome rules
    And normal status is accepted
    Then the existing outcome and appropriate retained editor state are shown
    And no operation is automatically replayed

    Examples:
      | editor  |
      | post    |
      | project |
```

Assert retained unsent content when the existing outcome would retain it; a successfully submitted editor may close under existing behavior. Use delayed success/failure/cancellation outcomes and request counts, not a new promise to preserve already-submitted work. REG-003 protects those existing outcomes.

### AT-005: Unknown downtime stays within existing error handling

Requirement IDs: FR-007, NFR-001, RULE-001
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_app_test.dart` (proposed), existing error-mapper fixtures

```gherkin
Feature: Unknown service failures
  Scenario Outline: An error does not establish maintenance
    Given no fresh validated maintenance status
    When <failure> occurs
    Then no custom maintenance screen is inferred
    And the affected screen retains localized error and retry behavior
    And a startup failure may use the existing connection or service error screen
    And otherwise available operations remain usable

    Examples:
      | failure                                  |
      | status endpoint unavailable               |
      | device offline                            |
      | API 503 with normal status                 |
      | API failure without a valid status         |
      | status endpoint returns proxy HTML or 502  |
```

Verify network/service error distinctions rather than requiring one new fallback message. Include announcement and health-probe results as inputs that cannot authorize maintenance. Timing evidence comes from UT-003/UT-004.

### AT-006: Clear or expire maintenance through existing gates

Requirement IDs: FR-004, FR-008, FR-009, RULE-004
Acceptance Criteria: AC-007, AC-011, AC-012
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_recovery_test.dart` (proposed)

```gherkin
Feature: Recovering from maintenance
  Scenario Outline: Remove the cover without bypassing account rules
    Given maintenance covers a retained current-account route
    When <transition> is accepted
    Then maintenance clears without relaunch
    And ready current-account UI resumes at the appropriate route
    And pending initialization or eligibility and moderation gates still apply
    And no sign-out, cross-account restoration or automatic write replay occurs

    Examples:
      | transition                         |
      | valid normal status                |
      | valid announcement status          |
      | freshness expiry from controller   |
```

For expiry, assert that no service-recovered claim appears and a still-failing API uses ordinary errors. Exercise current-account switch and policy state changes under the cover. Inject the expiry event here; UT-004 exclusively verifies its clock boundary and stale-result handling.

### AT-007: Restart with persisted dismissal but no maintenance cache

Requirement IDs: FR-006, FR-007, FR-008, NFR-001
Acceptance Criteria: AC-009, AC-010, AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_restart_test.dart` (proposed)

```gherkin
Feature: Status across process launches
  Scenario: Dismissal survives a new app instance
    Given announcement revision A was dismissed
    When a fresh app instance reads retained preferences and fetches revision A
    Then that announcement remains dismissed
    When announcement revision B is fetched
    Then the revised announcement appears
    When maintenance is fetched
    Then dismissal does not suppress the maintenance screen

  Scenario: Offline restart need not restore maintenance
    Given a previous process displayed maintenance
    When a new process starts without an in-memory status cache
    And its new status fetch fails
    Then initialization follows its existing behavior
    And no restored custom maintenance message is required
    And existing connection or service errors apply where operations fail
```

IT-002 tests the persistence boundary. Do not reuse a controller/cache between “process” instances. This scenario explicitly permits losing the old custom message; it adds no editor durability guarantee.

### AT-008: Independently publish, revise and clear

Requirement IDs: BR-002, FR-001, FR-012
Acceptance Criteria: AC-002, AC-015
Priority: Must
Level: Acceptance
Automation Target: `scripts/test_service_status_publish.py` (proposed, Python command harness; language/filename may follow final command design)

```gherkin
Feature: Operator status publication
  Scenario: Communicate while Render is unavailable
    Given editable valid JSON and authorized status publication access
    And AppView and database are unavailable
    And no app or website release or message-specific Git commit is performed
    When the maintainer validates and publishes after seeing and confirming the intended change
    Then public verification confirms the intended current status
    When the maintainer revises the content and confirms another publication
    Then the new document replaces the previous document
    When the maintainer confirms clear without manually editing JSON
    Then the verified public document has normal mode
```

Use an isolated remote adapter/fake public origin for command automation. IT-003/IT-004 own negative command outcomes; IT-005 owns website independence; IT-007 owns real hosted verification. Do not use production Cloudflare resources here.

### AT-009: Accessible, localized and responsive status UI

Requirement IDs: NFR-003
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `app/test/service_status/service_status_accessibility_test.dart` (proposed)

```gherkin
Feature: Readable status communication
  Scenario Outline: Render on supported UI configurations
    Given <mode> with custom English text at the selected allowed bounds
    When rendered with a supported theme, locale, form factor and text scale
    Then all text and actions are reachable without overflow
    And built-in text follows the selected locale while custom prose stays English
    And semantics expose the status and labelled actions
    And covered app controls are not exposed as active maintenance actions
    And any recovery estimate is labelled in local time

    Examples:
      | mode         |
      | maintenance  |
      | announcement |
```

Use the repository's supported locales/themes and declared platform/layout range, including a narrow mobile viewport and web layout. Check semantics, keyboard focus and applicable accessibility guidelines in widgets. MAN-001 verifies real screen-reader reading/focus behavior; a semantics assertion alone is insufficient.

## 4. Unit Test Cases

Each numbered row is one parameterized case family. Expand boundary inputs inside that family without creating redundant suites.

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, FR-003 | AC-003, AC-004, AC-005 | Supported contract and replacement semantics | Each of three modes; schema/revision; announcement/maintenance text; estimate present/absent; no publication time; camelCase wire fixture; harmless and scheduling/feature/link unknown optional fields | Supported required semantics accepted; no publication time required; one current mode; unknown optional fields do not enable behavior | `app/test/service_status/service_status_document_test.dart` (proposed); reuse fixture corpus for publisher validator |
| UT-002 | FR-002, FR-003, NFR-001, RULE-003 | AC-005, AC-019 | Bounded validation rejects unsafe/invalid documents | Malformed JSON, HTML, wrong types, unsupported schema/mode, missing required fields/text, invalid supplied estimate; selected UTF-8 body/text limits at boundary and over; response exceeding body bound while streaming | Reject invalid/oversized input before using it; supported boundary accepted; previous accepted status/age unchanged (state assertions in UT-004); no remote rendering or unsupported behavior | `app/test/service_status/service_status_document_test.dart` (proposed), plus publisher validator contract tests |
| UT-003 | FR-004, NFR-001 | AC-006 | Single polling/timeout authority | Controlled clock in all three modes; launch/resume/retry; simultaneous triggers; pending response; 60-second ticks; background/resume; disposal | Startup continues before fetch finishes; hung fetch resolves by three seconds; foreground checks every 60 seconds including normal; no background polling; triggers share one pending fetch; no overlapping requests or timers surviving disposal | `app/test/service_status/service_status_controller_test.dart` (proposed); widget smoke wiring in existing app tests |
| UT-004 | FR-003, FR-004, FR-008, NFR-001, RULE-001 | AC-005, AC-007, AC-011 | Single freshness and supersession authority | Valid maintenance at T0; just before/at/after T0+5 min; success unchanged at T1; invalid/failure at T1; normal/announcement; delayed result after timeout/disposal or newer accepted result; long background then resume | Fresh maintenance blocks only within trust period; stops no later than five minutes; successful unchanged validation renews from T1; invalid/failure does not renew/replace; valid normal/announcement clears immediately; superseded results never reinstate maintenance; expired cache does not claim recovery | `app/test/service_status/service_status_controller_test.dart` (proposed) |
| UT-005 | FR-006 | AC-004, AC-009 | Revision-scoped dismissal | Undismissed A; dismissed A; repeated A; revised/replacement B; transitions to normal/maintenance | Same-revision announcement stays dismissed; new revision appears; replacement leaves one mode; dismissal never suppresses maintenance | `app/test/service_status/announcement_dismissal_test.dart` (proposed) |
| UT-006 | FR-002, NFR-003, RULE-001 | AC-003, AC-017 | Estimate is local-time information only | Valid timestamps with zone offsets; configured local zone including DST boundary; absent estimate; advance past estimate while successful unchanged fetches keep maintenance fresh | Same instant displayed in local time with localized estimate label; absent estimate omitted; passing estimate does not change mode or override freshness | `app/test/service_status/service_status_estimate_test.dart` (proposed) |
| UT-007 | FR-002, RULE-003 | AC-005, AC-019 | Plain text and unknown control fields | HTML/script markup, URL-looking prose, unknown external link/API destination/schedule/feature fields within supported schema | Prose treated literally, no code execution, link action, arbitrary request destination or remote feature control; required semantics unchanged | `app/test/service_status/service_status_widgets_test.dart` and document tests (proposed) |

UT-002's common corpus must be run through both consumers and publication validation; shared data does not require sharing Dart/Python implementation. Select field names/types and finite limits in the coding plan, then pin independent literal boundary fixtures rather than computing all expected results from production constants.

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, RULE-002, RULE-003 | AC-001, AC-019 | Anonymous status HTTP and redirect boundary | Signed-in account with distinct bearer/DID/device/OAuth canaries; separate loopback AppView/status origins; redirect target recording all requests | Fetch valid, failed and redirected responses through production status adapter | Initial and any permitted redirect requests carry no auth/account/device headers, query, cookies or body data; no inherited authenticated interceptors; arbitrary destinations refused under selected redirect policy; remote unknown destination fields never cause requests; status works with AppView unavailable | `app/test/service_status/service_status_http_test.dart` (proposed); mobile socket smoke in existing `app/integration_test/critical_journeys_test.dart` |
| IT-002 | FR-006, FR-008 | AC-009, AC-011 | Dismissal survives a fresh app instance | Real preferences in isolated mobile test; dismissed A; independent new dependency/provider instances; no reused status cache | Reopen app instance, fetch A then B/maintenance; separately restart with failed fetch | A remains dismissed, B appears, maintenance unaffected; every launch fetches anew; no maintenance persistence required; existing fallback without fetched status | Extend `app/integration_test/critical_journeys_test.dart`; fast preference adapter cases in `app/test/service_status/announcement_persistence_test.dart` (proposed) |
| IT-003 | BR-002, FR-003, FR-012, RULE-002 | AC-002, AC-005, AC-015, AC-019 | Validate/confirm before mutation | Temporary editable file; mocked Cloudflare adapter and credential source; valid/invalid corpus; decline/EOF; isolated target config | Validate only; publish invalid; decline publish/clear; confirm valid publish/clear | Validate never mutates; invalid or unconfirmed command makes no mutation call; intended target/mode/revision/text shown before confirmation; clear creates valid normal without editing input; no dependency on clean Git/site release/Render; credentials absent from output and public document; preview target cannot overwrite production | `scripts/test_service_status_publish.py` (proposed); Python `unittest` subprocess/adapter mocks |
| IT-004 | BR-002, FR-012, NFR-002 | AC-002, AC-015, AC-016 | Verify intended public document honestly | Fake publication adapter and public fetch; exact intended result, old result, wrong content/type, unreachable origin, auth/mutation failure | Confirm publish or clear and inspect command result/stdout/stderr and calls | Success only after public response verifies intended document; cannot verify only upload response; mismatch/failure reports unverified and may already be live; no false success or automatic rollback; credentials not emitted | `scripts/test_service_status_publish.py` (proposed) |
| IT-005 | BR-002, FR-001, FR-012 | AC-002, AC-015 | Website release/rollback independent of status | Separate fake resource stores; status maintenance with known document/revision; current website and prior website versions | Publish/revise/clear status without site build/commit; execute site deploy and rollback adapter paths | Status commands leave site versions untouched; site commands address only website resource and leave status bytes/revision untouched; no bundled live status artifact | Proposed status command suite plus existing `scripts/test_web_deploy.py` and `scripts/test_cloudflare_pages_build.py`; real resource isolation verified in IT-007 |
| IT-006 | NFR-004, RULE-002 | AC-018, AC-019 | Inspect serialized diagnostics and operator output | Existing recording logger/serialized SDK transport; original typed exception and available stack; protected canaries in body, message, credentials, editor/account context; failing reporter | Exercise status transitions, repeated expected maintenance/offline/validation polling, one unexpected consumed failure and publishing failure; await observed sink output | Bounded approved operation/classification/outcome retained with available original cause/stack; no remote prose/full document or protected values; one owner for actionable occurrence; expected polls do not repeatedly capture issues; no revision/account metric labels; reporter failure does not change UI/retries/command result | `app/test/observability/service_status_diagnostics_test.dart` (proposed), reusing `diagnostic_evidence.dart` and `serialized_sentry_transport.dart`; command output canary assertions in Python suite |
| IT-007 | FR-001, FR-012, NFR-002, RULE-002 | AC-001, AC-015, AC-016, AC-019 | Real hosted JSON, CORS/cache and resource isolation | Authorized isolated Cloudflare status origin and separate disposable landing resource; selected equivalent production cache configuration; supported native HTTP client and web browser harness | Publish A, read anonymously from native/web; publish B then normal; apply scheduled foreground refresh checks; deploy/rollback disposable landing site; verify public response | Correct JSON/content type; web cross-origin reads permitted without credentials; native reads work; new publications observable via required refresh policy without indefinitely renewing A; verification reads intended document; site release/rollback leaves current status unchanged | Proposed hosted smoke/browser suite; final runner/location depends on hosting design (GAP-002) |

For IT-001, select and test an explicit redirect policy; refusing all redirects is acceptable. Do not require accepting arbitrary redirects. For web, application code must not add account-specific data; CORS/browser tests also verify credentials are omitted. Publication secrets may authorize the operator's mutation calls, but must never enter the public JSON, app binary/configuration or captured output.

IT-007 is not executed against production by this specification. Its timing checks verify delivery/cache policy; UT-003/UT-004 remain the sole client timer/freshness logic tests. Finite origin request timeouts and retry budgets should make the hosted verification fail clearly rather than hang.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test | Automation Target |
|---|---|---|---|---|---|
| REG-001 | 401 invalidates the captured session lease; network/503 errors do not sign out | FR-007, RULE-004 | AC-010, AC-012 | Run existing interceptor/error taxonomy tests; add status/unavailability cases retaining tokens/current account and a true 401 control case | Existing `app/test/shared/api/providers/sign_out_on_401_interceptor_test.dart`, `app/test/shared/errors/app_error_mapper_test.dart` |
| REG-002 | Current-account fencing, OAuth handoff and initialization/policy gates | FR-009, RULE-004 | AC-012 | Maintenance across account switch, delayed old-account completion and OAuth handoff boundary; clear/expiry restores only current-account allowed UI, without cancelling/replaying OAuth or bypassing restrictions | Existing `app/test/router/account_switch_routing_test.dart`, `oauth_handoff_route_test.dart`, `restricted_account_routes_test.dart` and account-gate tests; extend fixtures as needed |
| REG-003 | Existing in-flight success/failure/cancellation and publication counts | FR-009, RULE-004 | AC-008, AC-012 | Pause accepted upload/post/draft request; enter/exit maintenance and finish request; existing success cleanup/error retention/cancellation remains, exactly original request count, no automatic resubmission | Existing composer submission/video/draft tests and `critical_journeys_test.dart`; use AT-004 fixture rather than duplicate its UI assertions |
| REG-004 | Website allowlisting, preview promotion and rollback verification | FR-001, FR-012 | AC-015 | Run website build/deploy tests with independent status resource configured; verify static artifact and deploy/rollback mutation targets cannot overwrite live status | Existing `scripts/test_web_deploy.py`, `scripts/test_cloudflare_pages_build.py`; share IT-005 resource-isolation fixture |

All regression cases are planned automated checks. No new AppView admission/worker tests are required because those systems are unchanged; scope review must confirm that status implementation does not introduce backend controls.

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Valid contract corpus | Semantic schema/version and revisions A/B; all modes; English title/message; optional estimate absent/present; no publication time; allowed unknown fields; exact camelCase keys pinned after design | AT-001–AT-003, AT-007–AT-009, UT-001, UT-005–UT-007, IT-001–IT-005, IT-007 |
| TD-002 | Invalid and bounded inputs | Malformed/truncated JSON; HTML/error response; wrong types; missing required fields; unsupported schema/mode; invalid estimate; UTF-8 body and text at/over selected limits; chunked oversized response | AT-005, UT-002, UT-004, IT-003, IT-004 |
| TD-003 | Deterministic time/lifecycle/network | Injected clock; T0, just-before/at/after five minutes; T1 renewal; foreground/background/resume; delayed/hung/late HTTP completions and counts | AT-002, AT-005, AT-006, UT-003, UT-004, UT-006 |
| TD-004 | Sessions/editor/gates | Two synthetic accounts and fake session registry; ready/pending/failed/limited initialization; unsent post/project text and media; controlled request success/failure/cancellation | AT-001, AT-004–AT-007, IT-001, REG-001–REG-003 |
| TD-005 | Persistent dismissal | Fresh app/provider instances sharing only isolated preference storage; dismissed revision A; new B; no carried maintenance cache | AT-007, UT-005, IT-002 |
| TD-006 | Privacy/injection canaries | Unique non-secret synthetic bearer/device/DID/OAuth/publishing-secret canaries, private editor text, remote prose, markup/URLs, original exception/stack; recording requests/local/SDK output | AT-002, UT-007, IT-001, IT-003, IT-004, IT-006, IT-007 |
| TD-007 | Operator/resource state | Temporary editable JSON, isolated resource identifiers, confirmation/decline/EOF; fake public exact/stale/mismatched/unreachable responses; two website versions; Render adapter disabled | AT-008, IT-003–IT-005, REG-004 |
| TD-008 | Accessibility/localization | Supported light/dark themes and locales, narrow/mobile/web sizes, supported large text scaling, allowed-length multiline prose, local-zone/DST instants | AT-002, AT-009, UT-006, MAN-001 |
| TD-009 | Hosted delivery environment | Disposable independently hosted status and landing resources, test-only credentials outside repository, browser cache state and native/web clients; documented production-equivalent cache/CORS configuration | IT-007 |

Fixtures use synthetic identities and no real PDS writes, private production content or production credentials. Fixture-sharing across validator languages uses literal expected acceptance/rejection results. Device persistence tests isolate keys/storage and clean up. “Restart” coverage means fresh app/provider instances using actual preferences; a true OS process kill is optional corroboration, not a promise of restored editor work.

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-003 | AC-017 | Real screen-reader interaction | On supported iOS/Android and web with their screen readers, open announcement then maintenance with long text/large scaling; traverse actions; retry; clear while an editor remains underneath; inspect reading and keyboard/focus order | Title/message/estimate/actions read intelligibly; maintenance cannot focus or activate covered controls; notice preserves access to normal app; retry and dismissal labelled; recovery focus reaches appropriate current UI; content remains reachable |

Use supported test devices/browser and local or isolated status fixtures. Widget layout/semantics tests automate the broad matrix; this check tests actual assistive-technology behavior rather than duplicating screenshots. Record device/browser/reader versions and observed limitations.

## 9. Test Gaps And Risks

Risk remains Medium: stale status, account/editor preservation, publication independence and anonymous content handling have concrete verification paths. Planned coverage is complete at the behavioral level; it is not evidence of passing tests. The following dependencies must be resolved before their relevant tests can run.

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Literal contract fixtures and boundary numbers pending | FR-002, FR-003, NFR-001 | AC-005 | Requirements intentionally leave field names/types and finite bounds to design; earlier numeric examples are unapproved | Pin version/revision representation, exact required keys and independent body/text boundary values in coding plan before writing validator tests; run shared corpus on publisher and app |
| GAP-002 | Hosted delivery/CORS/cache evidence unavailable | FR-001, FR-012, NFR-002 | AC-001, AC-015, AC-016 | No status resource or permissions verified; existing mobile harness cannot run web; real edge behavior cannot be proven by mocks | Select independent Cloudflare resource, cache policy and browser runner; obtain separately authorized isolated provisioning/access; implement IT-007 and record configuration equivalence/remaining production smoke checks before rollout |
| GAP-003 | Proposed command/harness paths not implemented | BR-002, FR-012 | AC-002, AC-015 | Command syntax/language and remote adapter are design choices | Define command interface and injectable mutation/public-fetch boundaries in coding plan; retain confirm-before-mutate and explicit unverified outcome assertions regardless of language |
| GAP-004 | Real screen-reader evidence pending | NFR-003 | AC-017 | Widget semantics cannot demonstrate VoiceOver/TalkBack/browser reader usability | Execute MAN-001 on supported targets before rollout and record failures; do not count automated semantics as manual completion |
| GAP-005 | External OAuth/OS media boundaries remain existing limitations | FR-009, RULE-004 | AC-008, AC-012 | Existing integration harness does not automate disposable-PDS OAuth or camera/gallery permissions | Cover handoff/account races and already-selected media deterministically through existing fixtures; no new external OAuth/media capability in scope; reassess if implementation changes those boundaries |

No blocking product question prevents completing test design. GAP-001/GAP-003 are coding-plan prerequisites; GAP-002/GAP-004 prevent claiming hosted/accessibility release evidence until executed. Do not silently replace real-host checks with mocks or assume the chosen status address is already live. Requirements approval does not authorize production provisioning/publication.

## 10. Out Of Scope

- Withdrawn FR-010/FR-011 and AC-013/AC-014: AppView admission enforcement, backend draining/worker pause, maintenance API error codes and health-route changes.
- Feature-specific restrictions, scheduled activation/clearing, simultaneous notices/history, human status page, external links, translated custom prose and automatic monitoring publication.
- Persistent maintenance-cache restoration, required publication time, wall-clock reconciliation across launches, durable unsent editor storage or replay of writes.
- Live production deployment, DNS mutation, real PDS writes, load-testing infrastructure, new dependencies or implementation in this document stage.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-10-08-service-status-maintenance/01-requirements.md`.
- Test specification: `docs/changes/2026-10-08-service-status-maintenance/02-acceptance-tests.md`.
- Next review artifact: `03-document-review.md` in the same workflow folder.
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-10-08-service-status-maintenance/`. Await returned feedback before applying it.
- Recommended first failing test: **UT-001**, accepting a minimal valid maintenance document without publication time and recognizing the three modes. Pin the chosen contract first under GAP-001.
- Suggested test order: UT-001/UT-002 contract corpus → UT-003/UT-004 timer/freshness/supersession → IT-001 anonymous boundary → AT-001/AT-002/AT-005 startup and fallback → UT-005/IT-002/AT-003/AT-007 dismissal/restart → AT-004/AT-006 and REG-001–REG-003 state/gates → UT-006/UT-007/AT-009 and IT-006 presentation/privacy → AT-008/IT-003–IT-005/REG-004 publication independence → IT-007 and MAN-001 release evidence.
- Review recommendation: proceed to document review at Medium risk; the user may explicitly skip review and request the coding plan. Do not implement automatically.
- Blocking gaps: none for test design. Resolve GAP-001/GAP-003 for concrete implementation tests, GAP-002 for real hosted verification and GAP-004 for manual accessibility evidence.

Commands discovered (not run during this stage):

```sh
# Existing recipes; proposed focused path works once its tests exist.
just app-test test/service_status/
just app-test test/shared/api/providers/sign_out_on_401_interceptor_test.dart
just app-test test/observability/
just app-test
just app-analyze
just app-test-integration <device-id>
just app-test-integration-ci <device-id>
python3 -m unittest discover -s scripts -p 'test_web_deploy.py'
python3 -m unittest discover -s scripts -p 'test_cloudflare_pages_build.py'
# Proposed command-test filename, subject to GAP-003.
python3 -m unittest discover -s scripts -p 'test_service_status_publish.py'
just web-check
```

The existing integration recipes run `critical_journeys_test.dart` specifically; extend that suite or deliberately update the recipe during authorized implementation if a separate device suite is chosen. `just web-check` builds/checks the existing website and does not verify the new hosted status resource. No Go suite is required for this client/status-only design unless later approved changes affect AppView. Hosted browser command discovery belongs to GAP-002.
