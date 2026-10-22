# Acceptance Test Specification: App Status Flows

## 1. Test Strategy

Source: `01-requirements.md` in this folder, including its discovery decisions, edge cases and assumptions. The user authorized test design and review on 2026-10-20, then confirmed D-011/D-012 to resolve DR-001/DR-002. Document review is now approved with notes. This specification authorizes no implementation.

Use existing `flutter_test`/Riverpod provider overrides, controlled request completers, injectable clocks and widget fixtures under `app/test/service_status/`. Use literal shared contract fixtures in `service-status/contract-fixtures.json` for Dart/Python validation parity, Python `unittest` for operator workflows, and Node's test runner for the Worker. Add to existing suites where practical; proposed new file names below are planning targets, not files created by this stage.

Acceptance scenarios are automated at widget/app-boundary level unless explicitly assigned to a device/browser check. They test observable behavior through the real status host/router and controller, replacing external status, launcher and image services at their boundaries. Unit tests cover version/eligibility/validation rules and controlled timing. Integration checks cover actual transport, storage adapters and operator boundaries. Preserve existing maintenance, retained-editor, account-gate and diagnostic regressions.

Native store handoff, deployed web caching, platform back gestures and screen readers require separate real-platform evidence. Fake launch success, local browser reload and mocked preferences do not prove those behaviors. No runtime tests, hosted changes or publication are performed during test design.

Assumptions are not promoted to confirmed requirements:

- Former ASM-001 and ASM-004 are resolved by D-011/D-012. Failed refresh releases an existing update gate (AC-021). Successful CTA activation dismisses the captured message; persistence failure hides it for the session but allows recurrence after restart (AC-022). These are normative test expectations.
- ASM-002/003/005/006/007 remain visible dependencies on local-version fallback, destination list, legacy dismissal, decorative images and prerelease handling. Exact schema names and destination names must be fixed during coding planning before executable fixture creation.
- All test IDs are scoped to this change folder; older test comments may use the same IDs for earlier features.

User amendment D-014 (2026-10-21): optional images now use standard `Image.network` from a trusted public HTTPS host. Image-specific request omission/redirect/size/deadline/cancellation assertions are retired. Preserve AT-009 layout/fallback/usability and IT-006 diagnostics; assert no explicit AppView/account/device headers are supplied. Strict anonymous status JSON transport tests remain unchanged.

User amendment D-015 (2026-10-21): version parsing uses the existing `pub_semver` package behind strict syntax/build-omission handling. UT-001/007 and the shared corpus accept core/numeric prerelease components through 15 digits and reject longer numeric components; ignored build identifiers have no such numeric cap. Run eligibility tests on native and Chrome for consistent ordering.

User amendment D-016 (2026-10-22): message eligibility also requires resolved signed-in auth. Test signed-out/loading/error (including retained previous signed-in data) suppression, sign-in reveal, sign-out hide without dismissal, saved dismissal across sign-in cycles and next-revision eligibility. Assert status requests/blockers proceed while auth is pending. `message_auth_visibility_test.dart` exercises the actual auth projection; generic version/message fixtures explicitly model a signed-in audience.

User amendment D-017 (2026-10-22): message eligibility additionally requires completed onboarding for the active session. `message_onboarding_visibility_test.dart` covers incomplete/loading/error states, refresh with retained completion, completion reveal without another fetch, account switching and late previous-account completion. Suppression never writes dismissal. Auth fixtures now model completed onboarding where appropriate; generic message fixtures model a ready audience.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, UT-003, IT-003 | Acceptance / Unit / Integration | Yes |
| BR-002 | AC-003, AC-004 | AT-002, UT-001, UT-002 | Acceptance / Unit | Yes |
| FR-001 | AC-005 | UT-004, IT-001, REG-001 | Unit / Integration / Regression | Yes |
| FR-002 | AC-003 | AT-002, UT-002 | Acceptance / Unit | Yes |
| FR-003 | AC-004, AC-006 | AT-002, AT-003, MAN-003 | Acceptance / Manual | Automated plus platform check |
| FR-004 | AC-007 | AT-004, MAN-001 | Acceptance / Manual | Adapter automated; store handoff manual |
| FR-005 | AC-008 | AT-005, IT-005, MAN-002 | Acceptance / Integration / Manual | Local browser automated; deployed cache manual |
| FR-006 | AC-009 | UT-005, REG-001 | Unit / Regression | Yes |
| FR-007 | AC-010, AC-011 | AT-006, UT-001, UT-002 | Acceptance / Unit | Yes |
| FR-008 | AC-012, AC-022 | AT-006, AT-008, UT-006, IT-002, IT-004, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-013 | AT-007, UT-006, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-014, AC-022 | AT-008, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-015 | AT-009 | Acceptance | Yes |
| FR-012 | AC-016 | UT-007, IT-003, REG-004 | Unit / Integration / Regression | Yes |
| FR-013 | AC-013, AC-017 | AT-007, IT-003, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-014 | AC-023 | UT-009, IT-003 | Unit / Integration | Yes |
| RULE-001 | AC-002, AC-021 | AT-001, AT-011, UT-003, UT-008 | Acceptance / Unit | Yes |
| RULE-002 | AC-003, AC-010, AC-016 | UT-001, UT-002, UT-007 | Unit | Yes |
| RULE-003 | AC-018, AC-021 | AT-010, AT-011, UT-008 | Acceptance / Unit | Yes |
| NFR-001 | AC-006, AC-008 | AT-003, AT-005, IT-005, REG-003 | Acceptance / Integration / Regression | Yes; deployed cache check separate |
| NFR-002 | AC-019 | AT-009, REG-002, MAN-003 | Acceptance / Regression / Manual | Automated plus real accessibility check |
| NFR-003 | AC-020 | IT-001, IT-006, REG-004 | Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Independent controls follow priority and recover without relaunch

Requirement IDs: BR-001, RULE-001  
Acceptance Criteria: AC-001, AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_app_test.dart`

```gherkin
Scenario: Maintenance then required update then message
  Given a valid snapshot with fresh maintenance, an applicable platform minimum and an eligible undismissed message A
  When the app accepts the snapshot
  Then only maintenance is displayed
  When a new accepted snapshot removes maintenance but retains the minimum and message A
  Then only the update page is displayed
  When a new accepted snapshot removes the minimum but retains message A
  Then message A is displayed without a relaunch
  And suppression has not stored a dismissal for A
  When an accepted snapshot removes the message
  Then ordinary app content is interactive
```

Also exercise an empty-controls snapshot, a minimum that does not apply to the current platform/version, and a dismissed or out-of-range message. Retaining a section must retain its configuration rather than implicitly clearing it.

### AT-002: Required update applies only below the platform minimum

Requirement IDs: BR-002, FR-002, FR-003  
Acceptance Criteria: AC-003, AC-004  
Priority: Must  
Level: Acceptance  
Automation Target: proposed `app/test/service_status/update_required_test.dart`

```gherkin
Scenario Outline: Platform-specific enforcement
  Given no maintenance or message and the configured platform minima
  And the app is <platform> version <version> in <accountState>
  When a valid status snapshot is accepted
  Then <presentation> is shown

  Examples:
    | platform | version  | accountState | presentation |
    | iOS      | 1.7.9    | signed out   | update page  |
    | iOS      | 1.7.9    | signed in    | update page  |
    | iOS      | 1.8.0+13 | signed in    | ordinary app |
    | Android  | 1.8.0    | signed in    | update page  |
    | web      | 1.0.0    | signed out   | ordinary app |
```

Use iOS minimum `1.8.0`, Android minimum `1.9.0`, and no web minimum. On the update page, back dispatch, Escape and outside taps cannot dismiss the page. The signed-out ordinary app expectation means its normal welcome/sign-in screen, not protected content.

### AT-003: Blocking retains state and prevents navigation bypass

Requirement IDs: FR-003, NFR-001  
Acceptance Criteria: AC-004, AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_editor_test.dart`, `service_status_recovery_test.dart`, proposed `update_required_test.dart` in the same directory

```gherkin
Scenario: Update gate over an active editor
  Given a retained editor with unsaved text, media selections and a back stack
  And a write already submitted before the gate
  When an applicable update policy is accepted
  Then the update page is the only interactive status experience
  When back, an internal navigation action and an incoming deep link are attempted
  Then none exposes interactive ordinary content while the gate applies
  When the in-flight write completes
  Then the status layer has neither canceled nor resubmitted it
  When a valid snapshot lowers or removes the minimum
  Then the retained editor state and session remain intact
  And normal account initialization and policy restrictions still apply
```

Instrument write-call count and retained editor values. Include an account switch while covered; after recovery, protected content belongs only to the currently initialized account. Do not prescribe whether a deep-link destination is queued or discarded: requirements only prohibit bypass and lost retained state.

### AT-004: Mobile update CTA opens the correct listing and handles failure

Requirement IDs: FR-004  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: proposed `app/test/service_status/update_required_test.dart`

```gherkin
Scenario Outline: Mobile launcher outcome
  Given the update page on <platform>
  When the user taps Update and the launcher <result>
  Then the launcher received that platform's configured official CraftSky listing
  And <outcome>

  Examples:
    | platform | result          | outcome                                          |
    | iOS      | succeeds        | the page remains gated until policy reevaluation |
    | Android  | succeeds        | the page remains gated until policy reevaluation |
    | iOS      | returns failure | retryable feedback is shown and Update is usable |
    | Android  | throws          | retryable feedback is shown and Update is usable |
```

Returning from the store triggers normal foreground refresh, not a presumed successful upgrade. Exact native handoff is MAN-001, not proven by the launcher fake.

### AT-005: Web update requires a user action and reevaluates after reload

Requirement IDs: FR-005, NFR-001  
Acceptance Criteria: AC-008  
Priority: Must  
Level: Acceptance  
Automation Target: proposed `app/test/service_status/update_required_test.dart`; browser execution in IT-005

```gherkin
Scenario: Explicit reload
  Given a web update page at a URL with its existing path and query
  When the page appears and receives repeated refreshes
  Then no automatic page reload occurs
  When the user taps the update CTA once
  Then one reload of the current app URL is requested
  And in-memory editor preservation across that explicit reload is not promised
```

After restart, a supported version plus a valid snapshot permits ordinary access; an older version plus a valid minimum remains gated. Test initial status failure separately in AT-010. Real hosted cache behavior is MAN-002.

### AT-006: Eligible messages render and persist each dismissal path

Requirement IDs: FR-007, FR-008  
Acceptance Criteria: AC-010, AC-011, AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_widgets_test.dart`, `announcement_dismissal_test.dart`

```gherkin
Scenario Outline: Dismiss a minimal eligible message
  Given version 1.8.0+13 and message A for inclusive range 1.8.0 to 1.9.0
  And the message contains title and description but no image or CTA
  When the message is accepted
  Then one CraftSky dialog shows the title, description and dismissal action
  When the user dismisses using <action>
  Then the dialog closes and message revision A is persisted as dismissed
  When the controller restarts using the same storage and accepts A
  Then the message stays dismissed

  Examples:
    | action          |
    | explicit button |
    | barrier tap     |
    | Escape          |
    | back            |
```

Repeat eligibility at both inclusive endpoints, below/above range and without a maximum. Storage durability uses IT-002; account-change and delayed-dismissal races use UT-006/REG-002.

### AT-007: Unrelated publications preserve message dismissal

Requirement IDs: FR-009, FR-013  
Acceptance Criteria: AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/announcement_dismissal_test.dart`; publication boundary in IT-003

```gherkin
Scenario: Document revision differs from message revision
  Given message A is dismissed under document revision D1
  When document D2 changes maintenance but retains message revision A
  And document D3 changes platform minima but retains message revision A
  And all blocking controls subsequently clear
  Then A remains dismissed
  When a valid document D4 publishes eligible message revision B
  Then B is displayed
```

### AT-008: Message CTA uses a supported destination through ordinary gates

Requirement IDs: FR-008, FR-010  
Acceptance Criteria: AC-014, AC-022  
Priority: Must  
Level: Acceptance  
Automation Target: proposed `app/test/service_status/message_cta_test.dart`

```gherkin
Scenario Outline: Follow a message action
  Given an eligible message with CTA text and one valid <destination>
  When the user activates the CTA
  Then the intended destination is requested through the normal launcher or app router
  And protected destinations retain normal authentication, account and policy gates

  Examples:
    | destination                  |
    | public HTTPS URL             |
    | named supported app location |
```

Parameterize supported names once coding planning fixes the list. Never pass arbitrary remote paths directly to the router. Invalid CTA shapes are UT-007. Under confirmed D-012/AC-022, successful activation hides and attempts to persist dismissal for the captured revision. Rejected/thrown activation retains the message with retryable feedback. If persistence fails after activation succeeds, A stays hidden through refreshes/account switches in the current process; restart may show A again. Test destination success/persistence success, destination failure/no dismissal, and destination success/persistence failure separately. Include replacement B and a newly active maintenance/update gate while activation is pending: A's completion cannot dismiss B or reveal covered content. Do not retry destination activation merely because persistence failed.

### AT-009: Optional media and long content remain usable and accessible

Requirement IDs: FR-011, NFR-002  
Acceptance Criteria: AC-015, AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_accessibility_test.dart`, `service_status_widgets_test.dart`

```gherkin
Scenario Outline: Resilient message layout
  Given an eligible message with long valid title and description, CTA and <imageState>
  And <width>, <theme> and supported large text scale
  When the message is displayed
  Then its text, CTA and dismissal are reachable without overflow
  And covered content is excluded from focus, semantics and pointer interaction
  And image loading or failure does not block interaction

  Examples:
    | imageState    | width | theme |
    | absent        | 320   | light |
    | pending       | 320   | dark  |
    | valid         | 1280  | light |
    | broken        | 1280  | dark  |
    | extreme ratio | 320   | light |
```

Cover image absent, pending, valid, broken and extreme aspect ratio; widths 320 and 1280 logical pixels; light/dark; default and large supported text scaling (at least the existing 1.5 fixture). Exercise the update page with long localized copy under the same layout/focus matrix. After message dismissal or gate removal, restore valid retained focus. The decorative-image assumption is ASM-006. Real reader validation remains MAN-003.

### AT-010: Startup status failure allows ordinary access

Requirement IDs: RULE-003  
Acceptance Criteria: AC-018  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_restart_test.dart`, `service_status_app_test.dart`

```gherkin
Scenario Outline: Startup fails open
  Given a newly started app with previously stored message dismissal
  And no valid current-process status snapshot
  When the startup status check <failure>
  Then ordinary access attempts are permitted no later than the three-second deadline
  And no old minimum or maintenance is restored as a blocking gate
  And normal sign-in, account initialization and policy gates still apply

  Examples:
    | failure                  |
    | is offline               |
    | times out                |
    | returns malformed JSON   |
    | uses unsupported schema  |
```

Unknown local version behavior under an otherwise valid snapshot is UT-008 (ASM-002). Do not infer recovery or successful update from unavailable status.

### AT-011: Failed refresh invalidates update enforcement without extending maintenance

Requirement IDs: RULE-001, RULE-003  
Acceptance Criteria: AC-021  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/service_status/service_status_app_test.dart`, `service_status_controller_test.dart`

```gherkin
Scenario Outline: Failure releases an accepted update policy
  Given an accepted applicable update minimum and an eligible undismissed message
  And maintenance is <maintenanceState>
  When a subsequent status refresh <failure>
  Then the accepted update policy no longer blocks access
  And fresh maintenance, if present, remains only until its original five-minute deadline
  When that maintenance deadline passes without a valid renewal
  Then no stale update gate reappears
  And normal priority selects the eligible message or ordinary content
  When a later valid snapshot contains an applicable minimum
  Then update enforcement applies again

  Examples:
    | maintenanceState | failure                   |
    | absent           | fails with network error  |
    | absent           | times out                 |
    | absent           | returns invalid JSON      |
    | fresh            | fails with network error  |
    | fresh            | times out                 |
    | fresh            | returns invalid JSON      |
```

An invalid snapshot never partially updates sibling controls. Invalidation of update trust is the specified failure consequence, not acceptance of fields from that snapshot. Maintenance is neither cleared early nor renewed by failure.

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | RULE-002, FR-007 | AC-003, AC-010 | Semantic-version ordering and inclusive boundaries | TD-002, including numeric minor comparison, equal metadata variants, prereleases and absent maximum | Standard precedence; build metadata irrelevant; endpoints included; no lexical sorting | Proposed `app/test/service_status/app_version_eligibility_test.dart` |
| UT-002 | BR-002, FR-002, FR-007, RULE-002 | AC-003, AC-010, AC-011 | Platform and message eligibility | TD-001/002; each supported platform, omitted minima, below/equal/above thresholds and message range | Only selected platform threshold applies; message evaluated independently; required-only content accepted | Same proposed eligibility suite |
| UT-003 | BR-001, RULE-001 | AC-001, AC-002 | Priority truth table and section removal | All eight combinations of applicable maintenance/update/message, dismissed/out-of-range message and expired maintenance | At most one visible flow in priority order; unrelated controls retained; suppression never writes dismissal | `app/test/service_status/service_status_controller_test.dart` |
| UT-004 | FR-001 | AC-005 | Bounded lifecycle, coalescing and stale-result fencing | Controlled completers, fake time at 2,999/3,000 ms and 59/60 s; background/resume/disposal; late prior completion | Three-second cancellation; one coalesced request; no background polls; resume refresh; late/disposed result ignored | Existing controller suite |
| UT-005 | FR-006 | AC-009 | Maintenance trust and retry remain independent of other sections | Accepted maintenance at t0; valid unchanged renewal; failed read; 4:59.999/5:00 boundary; wall/elapsed clock changes | Trust renewed only by valid reads; expires at five minutes; rollback never extends trust; estimate informational; no restart restoration | Existing controller and `service_status_estimate_test.dart` suites |
| UT-006 | FR-008, FR-009 | AC-012, AC-013 | Dismissal identity and async races | TD-003; delayed read/write, new document with same message, new message B while A dismissal pending, account switch | Same message stays dismissed; replacement stays eligible; only captured revision written; maintenance never dismissed | `app/test/service_status/announcement_dismissal_test.dart` |
| UT-007 | FR-010, FR-012, RULE-002 | AC-014, AC-016 | Literal contract parity and atomic rejection | TD-004; independently authored valid/invalid fixtures, raw boundary bytes | Dart/Python agree; invalid section rejects complete snapshot; no valid sibling section partially applies; unsupported destinations rejected | `service_status_document_test.dart`; `scripts/test_service_status_publish.py` |
| UT-008 | RULE-001, RULE-003 | AC-018, AC-021 | Local version failure and current-session fail-open | Missing/invalid runtime version; accepted update with/without maintenance followed by network failure/timeout/invalid document; fake-clock expiry and later valid policy | Under ASM-002, version-targeted flows suppressed while maintenance works. Confirmed D-011: failure invalidates update enforcement, does not renew maintenance, cannot revive update at maintenance expiry, and permits later valid-policy gating | Existing controller/restart suites |
| UT-009 | FR-014 | AC-023 | Formal schema and semantic validation layers | Checked-in schema/meta-schema; valid examples, invalid structure/types/CTA; structurally valid reversed range; missing/broken schema/validator | Schema itself validates; structural faults fail schema; reversed ranges fail semantics; validator fails closed offline with bounded safe errors | `scripts/test_service_status_publish.py`, shared contract fixtures |

UT-008 traces startup/local-version behavior to AC-018 and confirmed current-session invalidation to AC-021. Test failure after a combined snapshot and before maintenance expiry, so independent priority/freshness tests cannot mask stale-gate revival.

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, NFR-003 | AC-005, AC-020 | Independent anonymous transport on native/web | Existing loopback HTTP and browser transport fixtures; AppView/auth unavailable; protected request canaries | Fetch valid new-format status; test deadline, redirect rejection, non-200 and oversized body | Independent request, bounded body/deadline, no API auth/cookies/account/device headers or credential forwarding; configuration accepted only on valid transport result | `app/test/service_status/service_status_http_test.dart`, `service_status_repository_web.browser.dart`; device loopback helper where needed |
| IT-002 | FR-008 | AC-012, AC-022 | Durable message dismissal across storage/controller restart | Actual preferences adapter with isolated test storage and distinct instances; device harness for real persistence | Dismiss A, dispose/recreate, reload storage, fetch A then B; repeat across account switch | A remains dismissed; B eligible; no restored maintenance/update gate; CTA activation plus failed storage hides only the captured revision for the current session, with possible recurrence after restart; activation failure does not persist dismissal | `app/test/service_status/announcement_persistence_test.dart`, `app/integration_test/critical_journeys_test.dart` |
| IT-003 | BR-001, FR-009, FR-012, FR-013, FR-014 | AC-001, AC-013, AC-016, AC-017, AC-023 | Operator contract, independent edits, publication and clear | TD-001/003/004; fake store/verifier plus actual local Worker/R2 for smoke check | Validate; publish all sections; publish maintenance-only change retaining other sections/message revision; clear all; decline/EOF/invalid input; simulate upload then verification failure | Validated snapshots preserve independent controls/message ID while document revisions change; clear explicitly contains no active controls; no write on declined/invalid input; input draft unchanged; post-upload verification failure reports unverified without auto retry/rollback | `scripts/test_service_status_publish.py`, `service-status/test/worker.test.js`, local `scripts/service-status-dev` workflow |
| IT-004 | FR-008, FR-010 | AC-014, AC-022 | Named route resolution through real app gates | TD-005; actual router with injected account/session state and fake external launcher | Activate every supported named destination signed out, uninitialized, policy-restricted and initialized; activate HTTPS action | Route mapping correct; existing gates decide access; no direct protected-page construction or credential forwarding; failure observable without claiming success | Proposed `app/test/service_status/message_cta_test.dart` with existing app-gate fixtures |
| IT-005 | FR-005, NFR-001 | AC-008 | Actual browser reload adapter | Disposable browser test page/app at path plus query; reload counter persisted outside document; controllable compiled version/status | Explicit CTA reload; restart with version below/equal/above minimum and unavailable status | Real reload retains URL; no automatic loop; version/status reevaluated; updated version can pass, still-old version gates on valid policy, unavailable startup fails open | Proposed dedicated `.browser.dart` suite under `app/test/service_status/`; exact runner setup in coding plan |
| IT-006 | NFR-003 | AC-020 | Status request isolation, trusted-host images and serialized diagnostic sinks | Existing diagnostic harness, TD-006, independent image host fixture; injected typed exceptions/stacks and throwing reporter | Render trusted-host images without app headers; fetch anonymous status; fail CTA/image/status/dismissal paths; serialize local/remote diagnostic output | No AppView credentials in status transport or explicitly supplied image headers; platform image networking follows D-014; positive approved operation/classification/type/stack evidence retained where permitted; canaries/prose/full URLs absent; expected failures do not become unexpected issues; no duplicate issue owner; reporter failure does not change business outcomes | `app/test/observability/service_status_diagnostics_test.dart`, existing sanitizer/log/sink harnesses, native/web request fixtures |

Mocked preference round-trips are adapter tests only. Claim durable native/web storage coverage only after the real storage/device variant has actually run. IT-005 verifies browser behavior locally; only MAN-002 establishes deployed cache behavior. Local Worker smoke must use `--local` and must not create or publish hosted resources.

For IT-003, exercise the real shared schema validator on publish/local-publish/setup/clear, then on the exact final payload after canonicalization and revision generation. Inject invalid generated output and missing/broken schema/validator: each causes zero store writes. Prohibit network schema resolution and verify all references resolve locally. Do not mock schema validation to unconditional success.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Maintenance lifecycle and status request timing | FR-001, FR-006 | AC-005, AC-009 | Extend/run `service_status_controller_test.dart`, `service_status_estimate_test.dart`, `service_status_restart_test.dart`: five-minute trust, failed-read nonrenewal, retry, foreground-only polling, three-second timeout and restart fail-open. Assertions must remain valid with simultaneous independent sections. |
| REG-002 | Existing modal dismissal, semantics and focus | FR-008, NFR-002 | AC-012, AC-019 | `service_status_widgets_test.dart`, `announcement_dismissal_test.dart`, `service_status_accessibility_test.dart`: existing dismissal paths, underlying control placement/state, no covered focus/semantics, focus restoration, delayed-revision safety. |
| REG-003 | Editor, session and active-account recovery | NFR-001 | AC-006 | `service_status_editor_test.dart`, `service_status_recovery_test.dart`, existing account-initialization gate fixtures: no editor reset, stale-account exposure, sign-out or write replay from status transitions. Explicit web reload is separately tested, not included in retained-state promises. |
| REG-004 | Publisher/Worker, transport and diagnostics boundaries | FR-012, FR-013, NFR-003 | AC-016, AC-017, AC-020 | Python/Node status suites plus HTTP/browser and diagnostic suites: byte/header/redirect bounds, no-store/CORS, explicit environment/confirmation, honest verified/unverified outcomes and protected sink canaries remain intact under the new contract. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Independent configuration combinations | Valid empty controls; maintenance-only; minima-only; required-fields-only message; all sections; separately omitted sections. Snapshot D1/D2 with same message A, then D3/message B. Fixed deterministic clock and controlled requests. Wire names are placeholders until coding plan. | AT-001, AT-002, UT-002, UT-003, IT-003 |
| TD-002 | Version precedence/platform targeting | Runtime `1.7.9`, `1.8.0`, `1.8.0+13`, `1.8.0+99`, `1.8.1`, `1.9.0`, `1.9.1`, `1.10.0`; prereleases `1.8.0-alpha.2`, `1.8.0-alpha.10`, `1.8.0-beta`, `1.8.0`; range min `1.8.0`, max `1.9.0` or absent; iOS/Android thresholds as AT-002 | AT-002, AT-006, UT-001, UT-002 |
| TD-003 | Message identity and asynchronous completion | Document revisions D1–D4; message revisions A/B/C; persisted A; paused read/write/activation; replacement B before A completion; shared storage across account changes | AT-006, AT-007, AT-008, UT-006, IT-002, IT-003 |
| TD-004 | Literal validation corpus and raw bounds | Valid/invalid schema/types; malformed UTF-8/JSON, non-object roots; 16,384/16,385 bytes; 160/161-scalar title, 2,400/2,401-scalar description and non-BMP text; blank strings; invalid/reversed versions, leading-zero numeric components; invalid timestamps; each CTA missing/dual target; unknown named destination; HTTP/javascript/data/custom scheme URLs; valid HTTPS; absent versus invalid supplied optional image/CTA | UT-007, IT-003 |
| TD-005 | Route and launcher boundary | Final supported name list paired with canonical expected route; signed-out/uninitialized/policy-restricted/initialized fixtures; public HTTPS URL; launcher false/throw/success; official per-platform store listing from app configuration | AT-004, AT-008, IT-004, MAN-001 |
| TD-006 | Privacy, media and layout | Unique token/cookie/account/device canaries, private prose and capability-like URLs; typed failures/caught stacks; positive permitted diagnostic fields; public image fixture with no auth; absent/pending/404/broken/very-wide/tall images; maximum valid text; light/dark, narrow/wide and large text | AT-009, IT-001, IT-006, MAN-003 |

Literal expected eligibility/rejection results must be independently authored, not computed with the comparator/validator under test. Shared Dart/Python corpus parity supplements independent expected outcomes; agreement between two incorrect validators is not sufficient. Preserve actual byte and Unicode-scalar boundaries rather than character-count approximations.

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-004 | AC-007 | Real official store handoff | On physical iOS and Android with a disposable applicable minimum from local/authorized preview status, tap Update, inspect actual listing, return to app, refresh while version unchanged. If feasible exercise unavailable-launch recovery separately. | Correct CraftSky listing, functional CTA, no presumed upgrade from returning; adapter evidence is reported separately from real store behavior. Never raise production minima for this test. |
| MAN-002 | FR-005, NFR-001 | AC-008 | Deployed web cache recovery | In an authorized preview environment, leave version A open on a non-root app URL, deploy supported version B through separately authorized workflow, publish applicable preview minimum, tap reload, inspect loaded build/version and URL. Repeat still-old build and unavailable status cases. | CTA obtains B under actual browser/CDN/service-worker configuration, preserves URL, reevaluates eligibility and avoids automatic loops. Record hosted headers/build evidence; do not clear all browser data as a substitute for the CTA. |
| MAN-003 | FR-003, NFR-002 | AC-004, AC-006, AC-019 | Platform back and accessibility | On iOS/Android/web, exercise update and maintenance blocking, message dismissal, long scaled content, both themes, keyboard, VoiceOver/TalkBack/supported web reader, Android predictive back and browser history back. Remove gate and inspect focus/current account. | Gates remain effective; messages dismiss; controls reachable and labelled; covered content unavailable to readers/focus; appropriate focus restored and account gates respected. |

No manual check authorizes infrastructure mutation, publication or deployment. Use local mocks when possible and request separate scoped live authorization only when needed for a concrete release check.

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Resolved: same-session failed refresh | RULE-001, RULE-003 | AC-021 | User confirmed D-011 after review DR-001. | AT-011/UT-008 require invalidation, maintenance-expiry safety and later valid-policy recovery. |
| GAP-002 | Resolved: CTA activation and persistence outcomes | FR-008, FR-010 | AC-012, AC-014, AC-022 | User confirmed D-012 after DR-002, including current-session hiding on persistence failure. | AT-008 and IT-002/004 require success/failure/persistence-failure and delayed replacement/blocking-flow cases. |
| GAP-003 | Exact schema, named destination list and legacy dismissal mapping pending | FR-008, FR-010, FR-012, FR-013 | AC-012, AC-014, AC-016, AC-017 | Coding-planning choices under ASM-003/005; supplied invalid optional-field and unknown-field policy must be consistent. | Fix types/names/null/unknown-field behavior and legacy preserve/reset policy in coding plan; materialize TD-004/005 with explicit expected results before red tests. |
| GAP-004 | Browser reload harness/caching proof incomplete | FR-005, NFR-001 | AC-008 | Existing browser suite covers transport, not full app reload; local harness cannot establish hosted caching. | Design runnable IT-005 harness in coding plan; run MAN-002 after separately authorized preview deployment and publication. Do not mark AC-008 fully verified from widget mocks. |
| GAP-005 | Native store, persistence and assistive-technology evidence | FR-003, FR-004, FR-008, NFR-002 | AC-004, AC-007, AC-012, AC-019 | Host tests cannot establish store routing, true device persistence, native back or real screen-reader output. | Execute IT-002 device variant and MAN-001/003 before release; record platform and actual outcome without conflating local tests with live proof. |
| GAP-006 | Public image fetch behavior differs across platforms | FR-011, NFR-003 | AC-015, AC-020 | Native and browser image paths may have different cookie/referrer/redirect behavior; image isolation must be achievable for the selected implementation. | Resolved by D-014: trusted-host Image.network uses platform defaults; no custom image privacy transport is required. Keep status isolation, no explicit app headers and image diagnostic/fallback checks. |
| GAP-007 | Update-only startup sequencing not fully specified | FR-001, FR-003, RULE-003 | AC-004, AC-005, AC-018 | Requirements mandate gating once policy is accepted and bounded fail-open, but do not explicitly require a startup loading screen before the result. | Coding plan must preserve these observable guarantees without inventing an indefinite startup gate. Seek a product decision if stricter pre-check interaction blocking is proposed. |

Risk remains Medium. No coverage has been executed for the new behavior. Gaps 001/002 are resolved; gap 003 requires concrete choices during coding planning before relevant coding; 004–006 require actual integration/platform/release evidence, not extra mock assertions. Document review is approved with notes after the confirmed revisions; see `03-document-review.md`.

## 10. Out Of Scope

- Executable tests, implementation, dependencies, commits, live publication, deployment or provisioned resources in this stage.
- Backend version enforcement, PDS/lexicon changes, scheduling, multiple-message queues, user targeting, analytics and automatic update installation.
- Native macOS/Windows/Linux targets; host-native libraries for Flutter tests remain normal tooling.
- Claims of backward support for old clients that cannot parse the new contract, or cross-device dismissal synchronization.
- Automatic web reload, arbitrary in-app path/custom-scheme actions, new image hosting/upload UI and tests for hypothetical features outside the requirements.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-10-20-app-status-flows/01-requirements.md`.
- Test specification: `docs/changes/2026-10-20-app-status-flows/02-acceptance-tests.md`.
- Next artifact: `03-document-review.md`, using `review-workflow-documents`. D-011/D-012 and AC-021/022 resolve the former provisional behavior extensions.
- External Plannotator review, if user-initiated: this workflow folder. No review tool was opened by this stage.
- Recommended first failing test: UT-007, one literal valid snapshot with all independent controls; current schema-1-only validation cannot accept it. Then UT-001/002 for version semantics before UI/state tests. Fixture creation waits for the coding plan's concrete contract.
- Suggested order: contract/validator parity → version/platform eligibility → priority/timing/freshness → revision persistence/races → blocking/retained-state widgets → launcher/CTA/image adapters → diagnostics/privacy → local publisher/Worker/browser integration → regression and real-platform release checks. Implement each approved behavioral slice with its own red-green-refactor loop.
- Commands discovered (to run during implementation, not run by this stage):

```sh
just service-status-test
just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart
just app-test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart
just app-test-integration DEVICE --no-pub
just app-analyze
```

- Flutter recipes run from the repository root and change into `app/`; `app-test` refreshes checkout-local Rive test libraries. Add the planned reload browser suite to the appropriate browser command after coding planning fixes its runner. Local Worker smoke commands are documented in `docs/operations/service-status.md`; do not run installation/publication during test design.
- Blocking for completion of this design stage: none. Former blockers GAP-001/002 are resolved; concrete contract/destination decisions in GAP-003 remain coding-plan tasks. Release verification gaps remain explicit and must not be represented as passing.
- Stage status: Reviewed with non-blocking notes; documentation only; no commit requested. All 22 Must requirements and all 23 acceptance criteria have designed coverage; real-platform evidence remains separate.
