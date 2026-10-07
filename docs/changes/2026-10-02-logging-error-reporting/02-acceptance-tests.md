# Acceptance Test Specification: Useful Logging and Error Reporting

## Sentry-led acceptance amendment approved 2026-10-07

| Test | Requirement / criterion | Observable behavior / order |
|---|---|---|
| SDK-T01 | SDK-002, SDK-003 / AC-001, AC-012 | Serialized Flutter native cause chain retains reviewed static explanation and caught frame; no parallel failure context; unknown/HTTP/private causes stay protected. |
| SDK-T02 | SDK-002, SDK-003 / AC-001, AC-002 | Go native exception chain retains attached stack and safe explanation with request correlation; stackless ordinary errors do not gain fabricated origin stacks; joined/cyclic guards remain bounded. |
| SDK-T03 | SDK-004 / AC-005, AC-012 | Serialized Flutter/Go issues include safe logging/connectivity/navigation timeline with private fields/capability URLs omitted. |
| SDK-T04 | SDK-001 / AC-002, AC-009 | Installed SDK framework/platform/zone integrations capture reportable original failures once with native mechanism and supplied frames; cancellation/offline/expiry/not-found/validation and explicitly non-reportable errors produce no issues. Reportability overrides and unexpected cause chains remain reportable. Explicit terminal network/expiry captures survive final filtering; direct/wrapped cancellation remains suppressed even with terminal/reportability overrides. Local callbacks do not recapture or replace SDK handlers and retain protected output for expected failures. Disabled path retains protected local output and UI fallback. |
| SDK-T05 | SDK-005 / AC-005, AC-015 | Official integrations export structured logs and breadcrumbs with event capture disabled; original operation/workflow/correlation selected; expected failures still produce no issues. |
| SDK-T06 | SDK-006 / AC-018, AC-019 | Full suites/analyzer and guide examples pass; final diff maps to this amendment; device/live checks explicit. |

Tests execute sequentially SDK-T01–SDK-T06 with one red-green-refactor behavior per loop. Earlier generic-message and custom-handler expectations are superseded only for explicitly reviewed static errors and SDK capture ownership, not arbitrary private prose.


## Refined acceptance coverage approved 2026-10-07

The refined SIM-001–SIM-006 contract supersedes tests of server prose/path mirroring, implicit log capture, ownership flags, exact multi-chunk sink parity, generic Flutter retry windows and global message registration. Retain credential/private exception/value canaries and original business-policy regressions.

| Test | Requirements | Observable result |
|---|---|---|
| SIM-T01 | SIM-001 / FR-006 | New server code/status/method/request ID retained through both mappers; server prose/fields/path absent; existing localized classification unchanged. |
| SIM-T02 | SIM-002 / FR-008 | Severe supporting logs export but never capture; provider/unhandled owners capture once; consumed failures explicitly reported where appropriate. |
| SIM-T03 | SIM-003 / FR-016, NFR-002 | Brief console record is parseable and bounded, type/useful frame/request/job reference survive, private prose/oversized identifiers omitted. No chunk reconstruction or generic retry coalescing contract. |
| SIM-T04 | SIM-004, SIM-005 / FR-001, FR-010 | Actual serialized SDK event/log retains type/stack/operational context and normal metadata, strips secrets/private payload/unknown exception values; ordinary static messages work without registry. |
| SIM-T05 | SIM-005 / FR-010 | Actual Go local/exported diagnostics admit ordinary scrubbed messages without registration/nonce, retain selected causes/private failure refs, omit credential/private canaries. |
| SIM-T06 | SIM-006 / FR-014, NFR-003 | Guide examples agree; full Go/Flutter suites and analysis pass; no business outcome/production changes. |


## 1. Test Strategy

Status: Draft for document review. Risk level: High, carried forward from requirements. This is test design only: the cases below are proposed, not implemented or passing. No application tests, live telemetry checks or production changes were performed in this stage.

Use Go `testing`, `httptest`, buffered JSON slog output, `sentry.MockTransport`, the existing in-memory metric recorder and package-specific stubs. Use Flutter `flutter_test`, Riverpod `ProviderContainer`, recording reporters, Dio response fixtures and a proposed SDK envelope-recording transport. Assert final serialized local/exported output as well as helper results: a fake reporter alone does not prove SDK sanitization. Separate log envelopes, issue events and transactions when counting records; batching or supporting breadcrumbs must not be mistaken for duplicate issues.

Acceptance scenarios express operator-visible outcomes. Automate them through component integration harnesses and a shared synthetic API-envelope fixture rather than introducing a new Gherkin runner or depending on live Sentry. Real PostgreSQL/MinIO tests use the existing local compose test infrastructure. Manual checks cover attached-device release output, guide usability and evidence-based investigations. Production retrieval is a separate authorized rollout check.

Each case must assert useful retained fields alongside absence of protected values. Compare structured fields, cause types and relevant frames; avoid brittle complete-message, stack, timing or generated-ID snapshots. Flush test transports deterministically, restore global log/Sentry hooks, dispose Riverpod containers and isolate test data. Use gates/barriers for stuck-request and account-switch tests instead of wall-clock sleeps.

All 24 Must requirements and the Should requirement have designed coverage. All 24 acceptance criteria are linked below. Coverage is a design claim; gaps in harnesses, release-device verification and production verification remain explicit in section 9.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003 | AT-001, AT-002, AT-003 | Acceptance | Yes |
| FR-001 | AC-001, AC-002, AC-021 | AT-001, AT-002, UT-001, UT-002, UT-003, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-001, AC-006 | AT-001, UT-004, UT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003 | AT-003, UT-006, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-004, AC-020 | AT-004, UT-007, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-005 | AT-005, UT-008, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-007 | UT-009, IT-008 | Unit / Integration | Yes |
| FR-007 | AC-008 | UT-010, IT-003, IT-009 | Unit / Integration | Yes |
| FR-008 | AC-009 | UT-011, IT-006, IT-009 | Unit / Integration | Yes |
| FR-009 | AC-010, AC-011 | AT-006, IT-006, IT-007, IT-010 | Acceptance / Integration | Yes |
| FR-010 | AC-012, AC-013 | AT-007, UT-012, UT-013, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-006, AC-013 | UT-005, UT-013, IT-006 | Unit / Integration | Yes |
| FR-012 | AC-014 | UT-014 | Unit | Yes |
| FR-013 | AC-015 | IT-012, REG-004 | Integration / Regression | Yes |
| FR-014 | AC-019 | IT-014, MAN-002 | Integration / Manual | Partly; example/link checks plus human review |
| FR-015 | AC-022 | IT-013 | Integration | Yes |
| FR-016 | AC-023 | IT-015, MAN-001 | Integration / Manual | Partly; real release-device confirmation manual |
| FR-017 | AC-024 | MAN-003, MAN-004 | Manual | No; evidence review and authorized rollout |
| NFR-001 | AC-016 | UT-015 | Unit | Yes |
| NFR-002 | AC-013, AC-017 | UT-013, UT-016, IT-016 | Unit / Integration | Yes |
| NFR-003 | AC-018 | REG-001, REG-002, REG-003 | Regression | Yes |
| RULE-001 | AC-012 | AT-007, UT-012, IT-011, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-002 | AC-011, AC-012 | AT-006, AT-007, IT-007, IT-011 | Acceptance / Integration | Yes |
| RULE-003 | AC-012, AC-013, AC-021 | UT-003, UT-012, UT-013, IT-011 | Unit / Integration | Yes |
| RULE-004 | AC-020 | AT-004, UT-007, IT-001, IT-011 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

Automation targets named as proposed do not exist yet. Existing paths name suites to extend, not evidence of current compliance.

### AT-001: Investigate a public-record failure

Requirement IDs: BR-001, FR-001, FR-002  
Acceptance Criteria: AC-001  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/app/logging_error_reporting_acceptance_test.go (proposed)`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Investigate a public-record failure
    Given a published post with distinct actor DID, target DID, AT URI and CID
    And a known-safe wrapped dependency error with two concrete cause types
    When its API read fails with local logging and Sentry test transports enabled
    Then an operator can identify the operation, failed stage, public target and actionable cause
    And local and exported diagnostics retain sanitized messages and available frame information
    And classification supplements the cause instead of replacing it
```

### AT-002: Investigate recovered and unhandled failures

Requirement IDs: BR-001, FR-001  
Acceptance Criteria: AC-002  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/middleware/recovery_test.go; appview/internal/tap/consumer_test.go; app/test/observability/error_handlers_test.dart`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Investigate recovered and unhandled failures
    Given known-safe HTTP and Tap panic values and a Flutter unhandled exception with a supplied stack
    When each component handles its failure
    Then its emitted diagnostics retain concrete type, safe detail and useful frames
    And Go panic frames identify the recovery chain
    And Flutter retains the supplied stack
    And an ordinary Go error without an origin stack is not attributed a fabricated origin stack
```

### AT-003: Follow one request across components without sampled tracing

Requirement IDs: BR-001, FR-003  
Acceptance Criteria: AC-003  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/app/logging_error_reporting_acceptance_test.go; app/test/observability/request_correlation_test.dart (both proposed)`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Follow one request across components without sampled tracing
    Given a synthetic failed API request with tracing disabled or a zero sampling rate
    When the Go harness emits its envelope and diagnostics and the Flutter harness consumes that envelope
    Then the unchanged requestId links arrival, completion, cause, enabled exported logs and issue events
    And Flutter diagnostics retain the same appViewRequestId
    And enabled spans attach available trace and span IDs without replacing request correlation
    And disabling Sentry preserves the local and API request trail
```

### AT-004: See actual incoming paths and stuck requests safely

Requirement IDs: FR-004, RULE-004  
Acceptance Criteria: AC-004, AC-020  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/middleware/logging_test.go`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: See actual incoming paths and stuck requests safely
    Given INFO-level logging for a public-resource request, a private-target request and an unmatched request
    When each arrives and the public-resource handler is held at a barrier
    Then its arrival record already contains method, sanitized actual path and request ID
    And no completion exists until the handler is released
    When handlers complete
    Then completion contains the same path and ID plus route pattern, status, duration and response bytes
    And public-resource references survive while private targets and unclassified segments become markers
    And raw query strings and fragments do not appear in any diagnostic context
```

### AT-005: Search meaningful Flutter logs independently of issues

Requirement IDs: FR-005  
Acceptance Criteria: AC-005  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/observability/log_bridge_test.dart; app/test/observability/sdk_emission_test.dart (proposed)`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Search meaningful Flutter logs independently of issues
    Given enabled Sentry Logs and two distinct known-safe message-only errors, an exception log and a warning
    And selected INFO operation outcomes and navigation/lifecycle breadcrumbs
    When root forwarding emits those records
    Then exported logs retain distinct messages, severity, logger and operation context
    And a later issue includes the relevant breadcrumbs
    And warning export alone creates no issue
    And DEBUG or FINE is remotely exported only after explicit opt-in
```

### AT-006: Investigate private-workflow operational failures

Requirement IDs: FR-009, RULE-002  
Acceptance Criteria: AC-010, AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/scheduledposts/observability_test.go; appview/internal/accountdeletion/worker_failure_test.go; appview/internal/observability/push_integration_test.go; appview/internal/instagram/worker_test.go`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Investigate private-workflow operational failures
    Given a scheduled-publication, push, Instagram integration or deletion-worker failure
    And private text, target membership, recipient details and provider credentials in its inputs
    When operational diagnostics emit
    Then the operation account DID, safe workflow reference, stage, attempt and outcome remain useful
    And approved provider status or code and a vetted cause explanation remain
    And unpublished content, private targets, recipient details and credentials are absent
    And successful routine private activity does not gain an account-linked history
```

### AT-007: Protect every sink while preserving public diagnostics

Requirement IDs: FR-010, RULE-001, RULE-002  
Acceptance Criteria: AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/app/logging_error_reporting_cross_sink_test.go; app/test/observability/sdk_emission_test.dart (both proposed)`

```gherkin
Feature: Useful and safe failure investigation
  Scenario: Protect every sink while preserving public diagnostics
    Given sentinel credentials and private values in supported structured fields, exceptions, URLs and nested data
    And distinct permitted public references and a known-safe explanatory cause
    When application, direct logger, SDK, breadcrumb, trace and platform emission paths run
    Then no serialized sink contains protected sentinels or their tested encoded forms
    And permitted cause, type, correlation and public references survive
    And unknown private text is omitted through provenance selection rather than presumed regex detection
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
| --- | --- | --- | --- | --- | --- | --- |
| UT-001 | FR-001 | AC-001 | Preserve known-safe cause trees in both runtimes | Typed wrapped and joined Go errors; typed Dart exception and supplied stack; public reference beside token | Concrete types, sanitized safe messages and useful causes retained with classification; credential absent; no generic-only replacement | `appview/internal/observability/sentry_test.go`; `app/test/observability/sentry_error_reporter_test.dart` |
| UT-002 | FR-001 | AC-002 | Distinguish available and absent stacks | Panic stack, supplied Flutter stack, ordinary stackless Go error | Keep available function/frame information; missing origin stack explicit/absent; trim local user path portions safely without erasing useful frames | Same suites as UT-001 |
| UT-003 | FR-001, RULE-003 | AC-021 | Apply provenance before emitting arbitrary messages | Opaque private sentence in unknown third-party and private-workflow fixtures; separate known-safe public explanatory message; SQL row values/HTML | Unknown/private raw text excluded; type, stage, stack, approved SQLSTATE/status/code and vetted explanation retained; known-safe detail survives sanitization | `appview/internal/observability/redaction_test.go`; proposed `app/test/observability/diagnostic_sanitizer_test.dart` |
| UT-004 | FR-002 | AC-001, AC-006 | Keep public actor/target identities distinct | Valid DID/handle/URI/NSID/rkey/CID and safe origin; malformed attempted identifier with validation reason | Safe references retained in correct actor/target fields; malformed value bounded and identified as attempted, never asserted valid | `appview/internal/observability/validation_test.go`; proposed Dart sanitizer suite |
| UT-005 | FR-002, FR-011 | AC-006 | Require published provenance and diagnostic purpose | Identical text marked published parse failure, ordinary success, draft and failed publication; deleted/blocked record reference | Only relevant selected published failure excerpt allowed; references preserved where permitted; no draft or successful routine text dump | Go redaction and proposed Dart sanitizer suites |
| UT-006 | FR-003 | AC-003 | Preserve IDs without fabricating correlation | Generated request ID, missing server ID, available/absent trace IDs, release/environment | Request ID unchanged through selection; unavailable server ID absent or explicitly unavailable; metadata retained; no replacement by fabricated ID | `appview/internal/observability/sentry_test.go`; `app/test/shared/errors/app_error_mapper_test.dart` |
| UT-007 | FR-004, RULE-004 | AC-004, AC-020 | Classify actual path segments and protect all companion context | Public post/profile; save/report/folder/private-media; hashtag search; OAuth callback; unmatched; encoded and double-encoded separators; query/fragment; operational parameter candidate | Public references retained only for public resources; private/search/capability/unclassified values marked; raw query/fragment absent; no companion target reintroduction; only explicitly approved typed operational parameters allowed, never search/cursor/auth state | Proposed `appview/internal/observability/request_path_test.go`; proposed Dart sanitizer suite |
| UT-008 | FR-005 | AC-005 | Choose log export, breadcrumbs and issues independently | WARNING/SEVERE/SHOUT; selected/unselected INFO; FINE; message-only and exception logs; export gates | Messages remain distinct and safe; warnings/errors and selected INFO exported; selected breadcrumbs retained; FINE requires opt-in; no automatic warning issue | `app/test/observability/log_bridge_test.dart` |
| UT-009 | FR-006 | AC-007 | Retain safe envelope fields through both mapper layers | 400/401/404/422/500/503; optional fields; missing/non-string fields; malformed body/HTML; network exception; route fixture manifest | Method/category/status/code/sanitized safe message/request ID/field reasons survive API and AppError mapping; typed parse/network cause preserved without body dump; sensitive field values excluded | `app/test/shared/api/providers/error_mapping_interceptor_test.dart`; `app/test/shared/errors/app_error_mapper_test.dart` |
| UT-010 | FR-007 | AC-008 | Deduplicate occurrences rather than matching text | Same failure logged and captured; two independent occurrences with identical text; reused error object in separate operations | One issue per occurrence; both separate occurrences retained; logs/breadcrumbs coexist; no cross-component suppression | `app/test/observability/error_handlers_test.dart`; `appview/internal/middleware/metrics_test.go` |
| UT-011 | FR-008 | AC-009 | Classify expected and terminal failures | Cancellation/offline/expiry/not-found/validation/retry vs exhausted retry/parse/storage/provider failure | Expected condition context available without unexpected issue; terminal/actionable error reportable with cause and available stack | Go error classifier suite; Dart AppError mapper and ProviderLogger suites |
| UT-012 | FR-010, RULE-001, RULE-002, RULE-003 | AC-012 | Select safe fields and sanitize mixed known formats | TD-002 and TD-003 including nested maps/lists, case variants, URL userinfo, signed query/capability paths and local file paths | All protected sentinels removed; approved public IDs and safe surrounding explanation survive; unknown objects not serialized wholesale | Go redaction/secret scan suites; Dart secret scan and proposed sanitizer suites |
| UT-013 | FR-010, FR-011, NFR-002, RULE-003 | AC-013 | Enforce chosen bounds after selection/sanitization | At limit minus one/limit/plus one; multibyte text; cause cycles; deep/large collections; credential straddling truncation boundary; oversized excerpt | Documented field/event/count/depth limits met; explicit omission markers; type/operation/ID/useful frames prioritized; no partial credential/binary/full unknown object emitted; termination on cycles | Go redaction suite; proposed Dart sanitizer suite |
| UT-014 | FR-012 | AC-014 | Expose selected model summaries (Should) | Public identity/post/event; mixed session/private model | Useful public references/state/counts retained; token/private fields omitted; no generic serialization of mixed state | Proposed `app/test/observability/diagnostic_summary_test.dart` |
| UT-015 | NFR-001 | AC-016 | Separate useful context from aggregation dimensions | Many DIDs/URIs/request/job IDs sharing one route/operation/cause class | Metric labels, transaction names and grouping inputs stable; diagnostic context distinct; different actionable cause types retain appropriate stable classification | `appview/internal/observability/metrics_test.go`, `route_test.go`, `sentry_test.go`; Dart reporter suite |
| UT-016 | NFR-002 | AC-017 | Summarize repeated failures deterministically | Fake clock, burst/retry sequence and terminal error at suppression boundary | Emission bounded under documented policy; summary includes suppressed count/outcome; terminal actionable cause retained | Proposed Go and Dart diagnostic bounds suites |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
| --- | --- | --- | --- | --- | --- | --- | --- |
| IT-001 | FR-004, RULE-004 | AC-004, AC-020 | Actual request lifecycle at INFO | Real mux plus logging/metrics/recovery chain; JSON buffer; public/private/unmatched paths | Run success, 4xx, 5xx, cancellation; hold one request behind barrier | Arrival observable before release; same ID/path on completion; separate stable route pattern (unmatched marker when absent); accurate status/bytes, nonnegative duration; ERROR cause for 5xx; canceled/499 internal classification; response behavior preserved; privacy applies to all context | `appview/internal/middleware/logging_test.go`, `metrics_test.go` |
| IT-002 | FR-004 | AC-004 | Probe severity exception | INFO and DEBUG log thresholds; both health paths | Run 2xx, 4xx, 5xx and unexpected failure | Successful probe arrival/completion DEBUG only; failed completion WARNING or ERROR as specified; no ordinary INFO probe noise | Go middleware logging suite |
| IT-003 | FR-001, FR-007 | AC-002, AC-008 | Recovery and fallback capture ownership | Recovery/metrics chain plus MockTransport; response-started panic; captured/uncaptured lower-layer failure | Panic before/after write; fail with/without deeper capture | Panic has safe detail/recovery frames; exactly one issue for captured occurrence; uncaptured 5xx fallback retained; recovery does not rewrite already-started response; ordinary stackless error not misrepresented | `appview/internal/middleware/recovery_test.go`, `metrics_test.go`; `appview/internal/tap/consumer_test.go` |
| IT-004 | FR-003 | AC-003 | Cross-component fixture trail | Go handler emits synthetic envelope and captured sink fixture; Dart Dio/provider harness consumes same fixture; trace off/zero-rate/enabled; DSN off/on | Fail public read and follow ID | Unchanged requestId found in local logs, enabled logs/issues/spans and Flutter diagnostics; available metadata retained; no trace prerequisite or missing-ID fabrication | Proposed Go acceptance and Dart request correlation suites |
| IT-005 | FR-005 | AC-005 | Root logger reaches actual SDK payload | Root subscription, SDK test transport, breadcrumbs; independent Logs gate | Emit warning/error/selected INFO and later exception; flush | Searchable distinct sanitized Logs with operation context; breadcrumbs attached to later issue; warning export alone no issue; FINE gated | `app/test/observability/log_bridge_test.dart`; proposed `sdk_emission_test.dart` |
| IT-006 | FR-002, FR-008, FR-009, FR-011 | AC-006, AC-009, AC-010 | Boundary coverage through real callers | Fault-injected API/auth/PDS/DB/Tap/indexer adapters, local buffers and SDK transport; published malformed record vs draft | Trigger each boundary failure, expected retry then terminal retry/quarantine | Each caller retains actionable approved cause/stage/code/dependency/public target; Tap event/URI/CID plus ack/retry/quarantine outcome; published excerpt only with purpose; expected retries not unexpected issues; terminal cause reported | Existing API observability, PDS wrapper, DB and Tap tests; proposed auth telemetry test |
| IT-007 | FR-009, RULE-002 | AC-011 | Private worker and integration diagnostics | Existing PostgreSQL/MinIO worker fixtures and provider stubs; private sentinels; two accounts | Fail scheduling/push/Instagram/deletion stage; then run routine success | Failure has operation account, non-capability workflow ID, stage/attempt/outcome, vetted cause/approved code; no private text/target/recipient/credentials; success does not gain account-linked activity history; persisted/returned error summaries also safe | Scheduledposts observability, accountdeletion failure, push integration and Instagram worker suites |
| IT-008 | FR-006 | AC-007 | Registered route and mapper coverage | Synthetic manifest matching Go route registrations and Dart endpoint mapping including business/video/scheduling/subscription/integration | Map safe 4xx/5xx envelopes per supported method/route through Dio then AppError | Every supported route has meaningful stable category/pattern; safe fields survive into reporter; unsupported/malformed paths sanitized; missing envelope retains network/parse cause | Dart error mapping suite; proposed Go route manifest fixture test |
| IT-009 | FR-007, FR-008 | AC-008, AC-009 | Flutter reporting owners and expected failures | Root/framework/platform handlers, root forwarding and real SDK test transport; Riverpod provider with retry controls | Route same occurrence through logging and reporting; repeat independently; run expected and terminal cases | One issue per component occurrence; two independent failures retained even with identical message; expected cases have context but no unexpected issue; terminal cases report; paired server event unaffected | `app/test/observability/error_handlers_test.dart`, `bootstrap/provider_logger_test.dart`; proposed SDK suite |
| IT-010 | FR-009 | AC-010 | Flutter feature coverage and operation-account attribution | Provider/mutation, initialization, video/media and storage existing stubs; begin under account A, switch to B before completion | Fault each phase including storage exception, video failure and caption unavailable | Retained type/cause/stack/feature/stage/approved status/code and permitted target; attribution remains A; missing ID handled; no JWT/local media bytes/path or search input dump | ProviderLogger, active-account initialization, video publication coordinator, secure token storage suites; proposed mutation diagnostic test |
| IT-011 | FR-010, RULE-001, RULE-002, RULE-003, RULE-004 | AC-012, AC-013, AC-020, AC-021 | Final cross-sink sentinel scan | TD-001–TD-004; application/root direct slog, localOnlyAttrs, third-party logger, CLI shared initialization, SDK-generated request/event/log fields, Flutter root/debug/platform adapters | Emit known-safe, private and unknown-origin diagnostics with tracing/Logs on/off; inspect complete serialized output | Secrets/private values and tested encodings absent from messages/causes/attributes/headers/URLs/breadcrumbs/traces/stacks; public positives retained; private path targets cannot reappear in context; bounded explicit markers; arbitrary private prose omitted by provenance | Proposed Go cross-sink and Dart SDK suites; existing secret/privacy suites |
| IT-012 | FR-013 | AC-015 | Disabled, throwing and failing transport | No DSN; Logs off; reporter throws; transport errors/flush failure; bootstrap not initialized | Execute original failing operation and supporting logs | Original result/exception/retry behavior unchanged; local sanitized cause available; no crash, recursion or unbounded retry from telemetry failure | Go logs/Sentry suites; Dart bootstrap/reporter/error-handler suites |
| IT-013 | FR-015 | AC-022 | AppView independent export including direct slog | Configured application logger and Observer; JSON buffer; MockTransport; tracing off/zero-rate; Logs gate | Emit ordinary arrival/completion, significant INFO, WARNING/ERROR via supported helper and configured direct slog; flush | Required matching local and exported Logs independent of issues/traces; successful probes excluded from INFO; Logs off retains local output only; no silent direct-slog bypass | `appview/internal/observability/logs_test.go`; proposed application logging wiring test |
| IT-014 | FR-014 | AC-019 | Guidance links and examples match interfaces | Final AGENTS.md/guide and implemented Go/Dart interfaces | Check canonical link; compile/run equivalent correct examples in tests; exercise incorrect-example failure/privacy assertions | Examples reflect callable interfaces and supported contract; links resolve; public/private/correlation/URL examples verified; commands and checklist match repo; human policy/usability review MAN-002 | Proposed documentation example tests under Go observability and Dart observability suites |
| IT-015 | FR-016 | AC-023 | Flutter release-policy platform adapter | Injected platform recorder with release policy active; Sentry off/failing | Emit WARNING/ERROR before/after initialization | Bounded safe cause/stack/ID sent to platform adapter; private sentinels absent; no persistent app buffer/support UI; real device validation still MAN-001 | Proposed `app/test/observability/platform_log_test.dart` |
| IT-016 | NFR-002 | AC-017 | Bounded enrichment has no fetch or media copy | Counting dependency/network fakes, large payload/media sentinel, repeated worker failure, fake clock | Exercise parse/publication/retry paths | No enrichment-only dependency calls, full-media serialization or buffering; bounded selected fields; suppression count/outcome summary and terminal cause present; business retry decisions unchanged | Go Tap/scheduled worker tests; Dart video coordinator and proposed bounds tests |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test | Automation Target |
| --- | --- | --- | --- | --- | --- |
| REG-001 | API envelope contract and no internal disclosure | NFR-003 | AC-018 | Exercise representative 4xx/5xx; camelCase error/message/requestId and optional fields unchanged; response/UI exclude internal frames and upstream bodies while operator sinks retain permitted details | Existing Go API/route tests; Dart interceptor/AppError mapper tests |
| REG-002 | Localized recovery actions | NFR-003 | AC-018 | Expected auth/offline/validation and unexpected errors retain current localized text/action semantics; enriched diagnostic message does not automatically become user-visible | Existing Dart AppError mapper and active-account initialization gate widget tests |
| REG-003 | Worker/recovery decisions | NFR-003 | AC-018 | Run scheduling, deletion and Tap acceptance tests; enrichment preserves retries, fencing, authorization, acknowledgements and post-response recovery behavior | Existing Go worker acceptance/consumer tests; recovery suite |
| REG-004 | No-telemetry operation | FR-013 | AC-015 | Run representative operations with disabled/unavailable reporter; original business outcome and safe local diagnostics remain; no reporting recursion | Go logs tests; Dart bootstrap/error reporter suites |
| REG-005 | Protected models and auth/video boundaries | RULE-001, RULE-002 | AC-012 | Keep draft/save/session/recipient redaction and video JWT tests; synthetic protected model canaries never reach final sinks | Existing Go redaction/secret scan tests; Dart secret scan/video credential lifecycle/persistence tests |
| REG-006 | Automatic collection remains conservative | FR-010, RULE-001, RULE-002 | AC-012 | Keep replay/screenshots/raw bodies/default PII and automatic failed-request capture disabled where currently disabled; explicit allowed context enrichment does not broaden SDK collection | Go Sentry configuration tests; Dart Sentry options and import-boundary tests |

## 7. Test Data

| ID | Purpose | Data | Used By |
| --- | --- | --- | --- |
| TD-001 | Positive diagnostic/correlation oracle | Synthetic public actor A/target B, valid typed DID/handle/AT URI/NSID/rkey/CID, published text fragment, request ID, fake release/environment, stable wrapped error types and named stack frames. Use example.invalid origins; no real accounts. One synthetic envelope shared between Go and Dart fixtures. | AT-001–AT-007; UT-001–UT-016; IT-001–IT-016 |
| TD-002 | Credentials and mixed URL canaries | Unique synthetic access/refresh/session/service JWT/cookie/password/OAuth code/state/PKCE/DPoP/private key/webhook/DSN sentinels; signed URLs and capability IDs. Embed beside public references, nested fields, URL userinfo/path/query, supported encoded forms and across prospective truncation boundaries. Never use real secrets. | AT-007; UT-001, UT-007, UT-012, UT-013; IT-010, IT-011, IT-015; REG-005, REG-006 |
| TD-003 | Private intent and account switching | Unique draft/schedule/media/save-folder/mute/search/report/evidence/email/device/push/payment/local-path sentinels; public target used inside private action; safe non-capability job ID; arbitrary opaque sentence with no recognizable secret pattern; operation A completes after account B activation. | AT-006, AT-007; UT-003, UT-005, UT-007, UT-012, UT-014; IT-007, IT-010, IT-011; REG-005 |
| TD-004 | Bounds/adversarial inputs | Unicode/newlines/control characters, malformed identifiers/envelopes, HTML/SQL row text, nested collections/cyclic cause graph, binary/media sentinel and sizes relative to chosen documented limits; repeated failures driven by fake clock. Assert structured JSON stays parseable. | UT-003, UT-004, UT-009, UT-012, UT-013, UT-016; IT-011, IT-016 |
| TD-005 | Outcome, route and environment matrices | 2xx/4xx/5xx/canceled; public/private/search/OAuth/unmatched routes; health/healthz; route manifest; error/log/tracing gates independently varied; DSN absent; failing reporter/transport; release-policy adapter; expected and terminal failure classes. | AT-003–AT-005; UT-007–UT-011; IT-001–IT-005, IT-008, IT-009, IT-012, IT-013, IT-015; MAN-001 |

Fixture discipline: include a positive retained-field oracle for each negative canary group. Published/private pairs must differ by provenance, not by the text alone. Compare raw serialized sink output and supported decoded representations; a source-text scan supplements runtime checks but does not replace them. Never seed synthetic failures into production.

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
| --- | --- | --- | --- | --- | --- |
| MAN-001 | FR-016 | AC-023 | Attached-device release logging | Build/run release on supported iOS and Android test devices using test configuration; collect platform console; trigger synthetic warning/error before initialization, with Sentry disabled, and with unavailable transport; repeat with canaries. Record platform/build/collector and sanitized excerpts. | WARNING/ERROR cause, useful frames and correlation visible within bounds; credentials/private values absent; no application-managed persisted buffer or support UI. Adapter tests alone cannot satisfy this check. |
| MAN-002 | FR-014 | AC-019 | Contributor guide usability and policy consistency | Start at root AGENTS.md; follow canonical guide; choose API public failure, private worker failure and Flutter provider failure entry point/severity/capture owner/context; review correct/incorrect Go/Dart examples and verification checklist against implemented interfaces. | Single discoverable guide; all FR-014 topics covered; examples safe and usable without optional skill; compile checks IT-014 pass; no policy contradiction. |
| MAN-003 | FR-017 | AC-024 | Local/test investigation evidence | Inject public-read/dependency, Flutter provider and initialization, Tap malformed-record/indexer and private scheduled-publication failures using existing test/dev fixtures. From emitted artifacts alone locate cause, permitted target/workflow reference and request trail where applicable. Record commands, config gates, sink excerpts, IDs and investigation steps in implementation validation evidence; include no real secrets. | Operator can locate actionable cause and applicable public/private permitted reference; Go/Flutter API request trail aligns; Tap/jobs use their safe event/job correlation, no fabricated HTTP trail; configured export and local paths proven independently; unresolved findings recorded. |
| MAN-004 | FR-017 | AC-024 | Separate production verification checklist | During implementation write pending checks for effective release/environment/Logs gates, unsampled correlation, operator access/retrieval and retention/access policy. Execute live checks only in a separately authorized rollout; use ordinary deployed diagnostics, not synthetic production failures. | Checklist explicitly unverified until authorized execution; local evidence not labeled production evidence; later record effective sink retrieval and any failures without changing configuration implicitly. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
| --- | --- | --- | --- | --- |
| GAP-001 | SDK final-envelope capture harness not yet established for Flutter | FR-005, FR-010, FR-016, RULE-001, RULE-002, RULE-003 | Existing recording reporters verify adapter calls; they do not demonstrate final SDK payloads. Go MockTransport is already available. | Add in-memory transport/envelope interception during implementation; verify Logs, issue and trace payloads through IT-005/IT-011 before claiming sink coverage. |
| GAP-002 | Numerical limits and suppression policy intentionally unspecified | NFR-002 | Approved requirements leave suitable sink-specific values to implementation. | Document selected field/event/depth/cause limits and suppression window/count policy; instantiate UT-013/UT-016 boundary fixtures against those values. Blocks completion of bounds verification, not test design. |
| GAP-003 | Release-device output remains unverified | FR-016 | Flutter unit tests cannot prove physical platform output in a compiled release build. | Run MAN-001 for supported iOS/Android devices; report unavailable device as remaining verification gap, not passed. |
| GAP-004 | Live Sentry configuration/retrieval not inspected | FR-003, FR-005, FR-015, FR-017 | Source flags and fake transports do not establish production sink behavior or access/retention suitability. | Record MAN-004 as pending and perform only during separately authorized rollout; no production action in this stage. |
| GAP-005 | Widespread callers could retain privacy/export bypasses | FR-009, FR-010, FR-015, RULE-001, RULE-002 | Representative tests alone do not establish coverage of all audited F-01–F-20 paths. | Implementation call-site inventory must map each finding/boundary to its supported emission path and test evidence; extend IT-006/IT-010/IT-011 for uncovered classes. Static searches are supplementary. |
| GAP-006 | Route classifier fixtures can drift or over-disclose | FR-004, FR-006, RULE-004 | Public-resource and private-action paths can share identifiers; unmatched/encoded paths need safe defaults. | IT-008 manifest comparison and UT-007 private-context cases; maintain actual registered route fixtures. Default unclassified values to redaction. |
| GAP-007 | Legacy tests encode blanket public-ID/ID stripping and generic messages | FR-001, FR-002, FR-003, FR-005, FR-006 | Existing Go logs/Sentry and Flutter log bridge/interceptor tests explicitly expect discarded request IDs/public context or App log. | Replace only superseded expectations with positive useful-context assertions plus preserved credential/private regression cases; do not delete protections to make tests pass. |
| GAP-008 | Guide and investigation evidence require human judgment | FR-014, FR-017 | Link/compiler checks cannot prove the instructions or emitted cause are operationally useful. | MAN-002/MAN-003 review recorded evidence against AC-019/AC-024; unresolved failures remain explicit. |

No blocking requirements questions were found. High-risk privacy verification requires the complete final-sink matrix, provenance cases and private-path/context cases before implementation is considered complete. Test design has not reduced the High risk classification.

## 10. Out Of Scope

- Application/test implementation in this stage; no source, test files, dependencies, generated code, guidance or runtime configuration changed.
- Live Render/Sentry inspection, production mutation/deployment, retention/access changes and publishing synthetic errors to a remote account.
- New routes, lexicons, migrations, retry/auth changes, persistent device diagnostic storage or support UI.
- Full raw request/response/provider/model/media capture, extra network fetches for enrichment and attempts to infer arbitrary private prose through universal regex rules.
- A new Gherkin runner, telemetry vendor or mandatory logging skill. The canonical guide and AGENTS.md policy remain implementation deliverables.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-10-02-logging-error-reporting/01-requirements.md` (Reviewed; user-confirmed decisions).
- Test specification: `docs/changes/2026-10-02-logging-error-reporting/02-acceptance-tests.md` (Draft for review).
- Next review artifact: `docs/changes/2026-10-02-logging-error-reporting/03-document-review.md`.
- External Plannotator review, if initiated by the user outside this skill: the same workflow folder. No Plannotator session opened by this stage.
- Recommended first failing test: UT-003 paired with UT-001. Prove known-safe original causes survive while arbitrary private sentences from unknown/private sources are omitted with useful type/stack/stage/code retained. This establishes the provenance contract before expanding callers.
- Suggested test order: provenance/error and secret units → actual path/context and bounds → final local/SDK emission harnesses → request lifecycle/correlation/export → capture ownership/expected errors → mapper/route coverage → feature and private worker boundary tests → regressions → guide example checks and manual release/investigation evidence.
- Commands discovered (read-only inspection; none executed in this stage):
  - `just appview-test-unit`: existing fast Go path, explicitly skips PostgreSQL/MinIO integrations; cannot establish full validation.
  - From `appview/`, `go test ./internal/observability ./internal/middleware -run '<selected test name>' -count=1`: focused isolated suites; use the repository toolchain configuration. New cases need final test names.
  - `just test`: host Go full suite with race detector against the local compose PostgreSQL/MinIO setup, as configured by `scripts/appview-test full`; existing dev stack must be available. AppView remains Docker-hosted in dev.
  - `just app-test test/observability test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart test/shared/errors/app_error_mapper_test.dart`: focused Flutter test suites; add affected feature suites as implemented.
  - `just app-analyze`: Flutter static analysis after implementation.
  - `just app-test-integration <DEVICE>`: existing separate device integration recipe; it does not establish release-platform console output without MAN-001.
- Blocking gaps for test design: None. Missing harnesses and selected bounds are implementation tasks. Release-device and production checks remain independently tracked verification gaps.
- Stage commit: Not requested for this stage. The earlier request committed the requirements artifact; it did not enable automatic commits for later workflow stages.
- Risk approval: The requested acceptance-test stage authorizes this design work. The revised test specification still needs the next review decision and explicit approval before implementation. Choosing a coding plan does not itself implement code.
