# Requirements: Useful Logging and Error Reporting

## Sentry-led refinement approved 2026-10-07

The maintainer explicitly approved and requested implementation of the second simplification. This amendment overrides conflicting SIM/legacy requirements. Privacy changes below are within that explicit approval; no additional approval is pending.

| ID | Approved requirement |
|---|---|
| SDK-001 | Sentry owns Flutter framework/platform/unhandled capture when enabled. Local handlers supply protected console output only, and release rendering fallback remains. When Sentry is disabled/unavailable, local diagnostics still work. Riverpod and consumed actionable failures keep explicit capture. |
| SDK-002 | Native SDK exception chains and supplied/attached stacks are authoritative. Remove parallel SDK cause/stack JSON. Use supported Dart cause extractors and Go native exception conversion; keep bounded traversal where needed to protect cyclic graphs. Do not invent an origin stack for ordinary Go errors. |
| SDK-003 | Retain scrubbed, bounded developer-controlled static failure explanations and safe runtime diagnostics. No message registry. Reviewed application errors declare that their message is static; HTTP/database/platform errors use selected status/code/explanations. Unknown dependency prose remains generic. Private bodies, inputs, credentials and capabilities remain excluded in all sinks. |
| SDK-004 | Preserve safe logging, lifecycle, connectivity, navigation and operation breadcrumbs as an error timeline. Filter their structured fields and URLs; unknown categories/prose remain conservative. Go no longer drops every breadcrumb. |
| SDK-005 | Use official Dart logging and Go slog integrations wherever they replace adapters cleanly. Logs/breadcrumbs never implicitly create issues; log event thresholds remain off. Keep one small error reporter plus selected context and independently protected console output. Remove redundant message/log reporter methods and unnecessary duplicate attributes. |
| SDK-006 | Update guide and workflow evidence; verify actual serialized issues/logs/breadcrumbs with useful fields and protected canaries, SDK automatic ownership, expected failures, and existing business regressions. Full Go/Flutter tests and analysis. No tracing enablement, production operation, API/lexicon/schema/business-policy change, or commit/push without a new request. Symbolication and device/live checks remain separately pending. |


## Refined contract approved 2026-10-07

The maintainer approved the smaller-contract recommendation and explicitly requested implementation. This section takes precedence over conflicting earlier text, test expectations and review findings. Earlier sections remain discovery/execution history; old exact-budget and cross-runtime equivalence requirements are superseded, not silently left unmet.

| ID | Current requirement | Supersedes / refines |
|---|---|---|
| SIM-001 | Flutter retains HTTP status, bounded stable error code, method, requestId and original client cause/stack. It owns localized UI messages and feature operations. Never ingest server message/validation prose or classify resource paths for telemetry. AppView keeps rich local diagnostics and sanitized paths; the API envelope stays unchanged. Remove mirrored Flutter message/code/route catalogues. | FR-006 / AC-007; client portions of FR-004, RULE-004, AC-020 |
| SIM-002 | Logs do not implicitly capture issues. Provider/framework/platform/unhandled boundaries own propagated failures; catch sites capture actionable consumed failures explicitly. Remove diagnosticIssueOwned/issueOwned propagation. Preserve expected-failure classification and operation outcomes. | FR-008 / AC-009; previous runtime ownership markers |
| SIM-003 | Release console fallback is one bounded, parseable brief record with useful original cause type, limited useful frames and available correlation. SDK reports retain richer details. Use small fixed field/count limits and whole-field omission; remove complete multi-chunk reconstruction and generic Flutter retry suppression machinery. Keep targeted existing Go worker suppression. | FR-016, NFR-002 / AC-013, AC-017, AC-023; exact sink parity/chunk/retry-count guarantees |
| SIM-004 | Sentry final callbacks are the main SDK filtering boundary. Retain native SDK exception/stack structure and useful standard metadata, strip request/user/private extras/attachments and unknown exception prose, admit only small operational/custom context. No Expando/nonce provenance tracking or rebuilding complete SDK events. | FR-001, FR-010, RULE-003 / AC-001, AC-012, AC-021; strict runtime provenance of every SDK attribute |
| SIM-005 | Ordinary developer-written diagnostic messages need no central registry. Messages are bounded and scrubbed for recognized credentials, email/local paths and capability URLs. Never interpolate private payloads; never log unknown object/error strings. Keep compact typed private/public contexts and vetted provider codes/causes. Use a small local boundary filter independent of Sentry. Remove Go/Flutter global message registries and log nonce machinery. | FR-010, RULE-001–RULE-003 / AC-012, AC-021; every-message registration |
| SIM-006 | Contributor guide describes the simplified boundaries; demonstrate retained type/stack/correlation and credential/private-payload exclusion in actual local and mock SDK output. Run affected and broad suites. No production operation, configuration/access/retention change, API/lexicon/schema change or business-policy change. | FR-014, FR-017, NFR-003 / AC-018, AC-019 |

Retained protections: credentials, private drafts/schedules/moderation/media/recipient data remain excluded; private operational attribution is failure-only. Unknown exception prose remains untrusted. Scrubbing is defense in depth, not a claim to identify arbitrary private sentences. Standard SDK metadata may remain; static developer messages are trusted by code review rather than a runtime permission registry. Native device/production checks remain separately tracked. Existing source-selected AppView private-PDS propagation and original media causes are retained.


## 1. Initial Request

Audit AppView and Flutter logging and error reporting to recover diagnostic value currently lost through excessive redaction or omission. Authentication tokens, credentials and secrets must remain excluded. Public Atmosphere data can be included when it helps diagnose an issue.

This document records read-only discovery and requirements, revised after the user confirmed the grilling decisions on 2026-10-05. No implementation, application tests, deployment, settings changes or commits were performed.

## 2. Current Codebase Findings

Findings below were verified in this checkout on 2026-10-02. They describe source behavior, not an inspection of live Render or Sentry events. Priority indicates implementation urgency within this change.

| Finding | Priority | Relevant files | Verified behavior and improvement opportunity |
|---|---|---|---|
| F-01 | First | `appview/internal/observability/sentry.go`, `error_classifier.go` | `CaptureError` replaces the actual error with a broad classified message and an `AppViewError` exception whose value is a classification code. The event constructed here has no stack. Preserve sanitized concrete error type, message, wrapped causes and available stacks alongside classification. |
| F-02 | First | `appview/internal/observability/sentry.go`, `appview/internal/middleware/recovery.go`, `appview/internal/tap/consumer.go` | Panic events retain recovered type but replace value with `redacted` and attach no stack. HTTP recovery logs omit recovered value and stack. Tap converts a panic to its type. Capture the stack at recovery and retain sanitized panic details. |
| F-03 | First | `appview/internal/middleware/metrics.go`, `recovery.go`, `appview/internal/observability/sentry.go` | Middleware supplies `run_id`, but `allowedEventContextKeys` excludes it. `request_id` is allowed under a moderation-oriented validator. Fix end-to-end correlation between local `run_id`, API `requestId`, Flutter `appViewRequestId`, Sentry events and spans. |
| F-04 | First | `appview/internal/middleware/logging.go` | Request start is INFO and contains method/run ID; completion is DEBUG. At INFO, route, status, duration and response size are absent from this request log pipeline. Emit a useful completion record at normal production verbosity. |
| F-05 | First | `appview/internal/api/telemetry_log.go`, `post_read.go`, `post_create.go`, `profile.go`, `timeline.go`, `appview/internal/auth/telemetry_log.go` | Shared API/auth helpers retain operation/category but no underlying error. Numerous handlers use those helpers after receiving concrete errors. Some other handlers log raw `slog.Any("error", err)` instead. Provide consistent sanitized causes and public target context across both patterns. |
| F-06 | First | `appview/internal/middleware/metrics.go`, `appview/internal/observability/db.go` | Generic 5xx fallback has no underlying cause when a deeper layer has not captured it. DB observation records operation/result/duration but returns errors without capture. Capture at the layer with the cause; keep the generic fallback for genuinely unknown causes. |
| F-07 | First | `app/lib/bootstrap.dart` | Provider failures log provider/mutation names, then report `RedactedProviderFailure` with the supplied stack instead of the original exception. Provider updates often collapse values to type/count. Retain sanitized exception type/message and purposeful public context without dumping provider state. |
| F-08 | First | `app/lib/shared/observability/log_forwarder.dart`, `sentry_error_reporter.dart` | Only SEVERE+ or explicitly promoted WARNING records forward; normal root forwarding never supplies `promotedWarning`. Message-only records become `App log`. Exception records lose their original log message as operation context. Retain messages; make warnings and relevant breadcrumbs available without creating an issue for every warning. |
| F-09 | First | `app/lib/shared/observability/sentry_sanitizer.dart`, `appview/internal/observability/sentry.go`, `validation.go` | Flutter drops contexts/breadcrumbs containing DIDs or HTTP URLs and limits strings to 160 characters. Its key allowlist also drops `source` produced by `AppErrorMapper`. AppView excludes public identities/references through its allowlist and shares metric normalization with event contexts. Separate bounded metrics from useful event/log context. |
| F-10 | First | `app/lib/shared/api/providers/error_mapping_interceptor.dart`, `api_exception.dart`, `app/lib/shared/errors/app_error_mapper.dart` | Flutter keeps some 4xx `appViewMessage` values but excludes all 5xx messages; mapping to AppError omits even the retained message. Validation `fields` are not retained in `ApiFailureDetails`. Endpoint categories are hand-enumerated and newer/uncovered routes fall back to `appview.unknown`. Preserve approved diagnostic envelope fields and identify all supported routes. |
| F-11 | First | `app/lib/auth/providers/active_account_initialization_provider.dart` | `logActiveAccountInitializationFailure` logs only a fixed sentence and explicitly omits the underlying error and account. Attach sanitized cause, stack, initialization stage and public account DID. |
| F-12 | First | `app/lib/main.dart`, `app/lib/shared/observability/sentry_error_reporter.dart`, `appview/internal/observability/logs.go`, `appview/internal/app/deps_foundation.go` | Flutter passes raw exceptions to Sentry; inspected adapter sanitizes context/breadcrumbs but has no final exception/log event scrubber. Debug output prints raw message/error/stack. AppView direct slog and `localOnlyAttrs` bypass `SanitizeEventContext`. Protect secrets consistently at emission boundaries, including third-party diagnostics. |
| F-13 | Next | `app/lib/main.dart` | Root-zone, framework and platform handlers both emit a severe record and call `captureException`; severe forwarding can also capture it. Define one issue capture owner and retain supporting logs/breadcrumbs. SDK deduplication alone is not a demonstrated guarantee here. |
| F-14 | Next | `appview/internal/observability/pds.go`, `appview/internal/tap/consumer.go`, `appview/internal/index/` | PDS observation classifies failures but omits public repository/record context. Tap logs ID/NSID/category, dropping the failing record URI/DID/CID and detailed cause. Attach public target identifiers, dependency origin and ack/retry/quarantine outcomes. |
| F-15 | Next | `app/lib/observability/video_diagnostics.dart`, `app/lib/feed/composer/video_publication_coordinator.dart`, `app/lib/feed/providers/composer_images_provider.dart`, `appview/internal/api/video_caption.go` | Video diagnostics expose operation/outcome/byte band and optional request ID; coordinator builds them with operation/outcome only and can log a failure stack without its cause. Caption fetch logs only `unavailable`. Capture the failed phase, cause, provider status and available public references; never video JWTs or local media bytes. |
| F-16 | Next | `appview/internal/api/scheduled_post.go`, `scheduled_media.go`, `notification_preferences.go`, `instagram_account.go`, `instagram_imports.go`, `instagram_verifications.go`, `appview/internal/accountdeletion/worker.go`, `scheduledposts/worker.go`, `push/dispatcher.go` | Several private-workflow failures log only an error class, store/effect category or fixed message. Account deletion logs job ID/category/attempt; push intentionally reduces provider errors to result classes. Add safe causes, stage and opaque workflow correlation while retaining private-content protection. Audit persistence/return boundaries as well as log calls. |
| F-17 | Next | `app/lib/auth/providers/auth_controller.dart`, `secure_token_storage.dart`, `app/lib/shared/device/device_id_provider.dart`, `app/lib/shared/rich_text/data/appview_facet_suggestion_repository.dart` | Some auth/storage failures omit causes; others retain raw errors in WARNING records that are not ordinarily exported. Preserve sanitized storage/network failure details, retry state and feature context. Do not log session storage or search/mention input wholesale. |
| F-18 | Next | `app/lib/settings/models/settings_identity.dart`, `app/lib/auth/models/auth_state.dart`, `active_account_identity.dart`, `app/lib/shared/widgets/post_summary.dart`, `app/lib/business/providers/business_event_detail_provider.dart`, `profile_business_events_provider.dart` | Fully redacted representations also hide public identity/content references. Add explicit diagnostic summaries of public fields, avoiding generic serialization of models that can also hold private/auth data. |
| F-19 | Keep protection | `app/lib/drafts/`, `app/lib/saved_posts/`, `app/lib/auth/models/stored_session.dart`, `pending_handoff.dart`, `app/lib/feed/models/video_upload_limits.dart`, notification recipient models | These representations protect genuinely private content, session tokens, handoff state, service JWTs or device/recipient identifiers. Keep protected fields excluded; expose stage/type/count/failure details separately. Public targets do not make save/folder membership or delivery history public. |
| F-20 | Next | `appview/internal/observability/observer.go`, `logs.go`, `appview/internal/app/config.go`, `render.yaml`, `app/lib/main.dart` | AppView Sentry log export uses Observer emission rather than automatically exporting every direct slog call; export/sampling are configurable. Flutter stdout sink is debug-only. Define which signals are available in each environment and sink, and report exporter failures locally without recursion. No live configuration was queried. |

Follow-up read-only checks during grilling confirmed that the checked-in Blueprint enables Sentry logs and HTTP/Tap tracing at 1% sampling, but this is not verification of effective production settings. Direct middleware slog output does not automatically export through Observer. Paths can contain private targets or search terms: `/v1/posts/{did}/{rkey}/saves`, `/reports`, and `/v1/search/hashtags/{tag}/posts`. The existing root contributor guide is `AGENTS.md`; `.agents/skills/` exists, while its references to `.claude/skills/` are stale in this checkout. The new logging guide must use paths that actually exist after implementation.

Existing regression coverage includes `appview/internal/observability/*_test.go`, middleware logging/recovery/metrics tests, feature privacy tests, `app/test/observability/`, `app/test/bootstrap/provider_logger_test.dart`, and API/error-mapping tests. Several encode the old rule that raw public identifiers or all error text must be absent; distinguish obsolete public-data expectations from retained secret/private-data guarantees.

Constraints: `AGENTS.md` and `atproto-craft-social-app-reference.md` retain the public-PDS/private-AppView split and server-side OAuth credentials. The API architecture spec retains `/v1/`, camelCase error envelopes and opaque pagination. Previous requirements under `docs/changes/2026-06-29-appview-observability/`, `2026-07-02-appview-sentry-observability/`, and `2026-07-03-flutter-error-handling-sentry/` describe the restrictive policy; this document supersedes their diagnostic restrictions only where explicitly stated below.

Discovered verification commands for later implementation: `just appview-test-unit`, `just test` (host Go suite against compose PostgreSQL), `just app-test <test paths>`, and `just app-analyze`. None were run in this requirements stage.

## 3. Clarifying Questions And Decisions

### Q1: Which information may logs and Sentry retain?

Answer: The user selected “Yes—use that boundary” for public DIDs, handles, AT URIs and CIDs; private drafts/schedules, moderation reports, email, device/push identifiers and payment details remain protected. Public text/payloads may be captured only when useful for failure diagnosis, with size limits.

Decision / implication: Use the same data-classification boundary for local logs, exported logs, error events, breadcrumbs and trace context. A public identity is allowed, but a person's private activity associated with that identity is not automatically public. Authentication credentials and capability-bearing values remain excluded regardless of source or environment.

### Q2: How should repository contributors and agents learn the logging/reporting conventions?

Answer: The user requested an additional requirement for useful repository rules or a skill and invited a recommendation.

Decision / implication: The user confirmed a concise mandatory logging/exception-reporting section in root `AGENTS.md`, linked to one canonical practical guide at `docs/engineering/logging-and-error-reporting.md`. The checkout already uses `AGENTS.md` for contributor rules and `.agents/skills/` for task workflows; no `.agent/rules` convention was found in the inspected locations. Use the existing contributor entry point rather than introduce a tool-specific rules directory with unverified discovery. A dedicated skill is optional future workflow support, not the source of mandatory policy. This stage adds the requirement only; the guide and AGENTS.md changes belong to implementation.

### Grilling decisions confirmed on 2026-10-05

The user accepted all recommendations in both rounds and then confirmed shared understanding and authorization to update this document. Round question numbers below refer to the interview, separately from Q1/Q2 above.

| Interview question | Confirmed decision | Requirement references |
|---|---|---|
| Round 1 Q1; Round 2 Q10 | Log sanitized actual paths on arrival and completion at INFO for ordinary requests, including unmatched requests. Successful `/health` and `/healthz` probes use DEBUG; failed probes remain WARNING/ERROR. Keep route patterns separately. | FR-004, RULE-004 |
| Round 1 Q2 | Private-workflow operational failures may identify the account DID, operation, failed stage and opaque non-capability workflow reference. Private content, targets/membership and recipient details remain protected. | FR-009, RULE-002 |
| Round 1 Q3 | Export WARNING/ERROR logs independently and selected INFO request/operation records when Sentry logging is enabled; retain navigation/lifecycle breadcrumbs. Issue capture is separate from log export. | FR-005, FR-015 |
| Round 1 Q4 | Mandatory `AGENTS.md` instructions link to one practical guide. A dedicated skill is outside this deliverable. | FR-014 |
| Round 1 Q5 | Implementation chooses documented and tested size/count/depth limits. Preserve core diagnostic fields and useful frames before payload excerpts. | NFR-002 |
| Round 1 Q6 | Demonstrate representative investigations using local/test sinks during implementation; record production verification as a separate authorized rollout step. | FR-017 |
| Round 2 Q7 | Redact private targets/search terms within paths. Omit query strings by default; capture only explicitly approved typed operational parameters separately. | RULE-004 |
| Round 2 Q8 | For private workflows and unknown third-party exceptions, retain concrete type, stack, stage/status/code and vetted explanatory text rather than automatically exporting complete messages. Known-safe messages may be retained after sanitization. | FR-001, RULE-003 |
| Round 2 Q9 | Flutter release builds emit bounded sanitized WARNING/ERROR diagnostics to platform logging, including when Sentry is disabled/unavailable. No persistent buffer or support UI. | FR-016 |

These decisions refine the original privacy boundary for operational failures without permitting private content or target/activity disclosure.

## 4. Candidate Approaches

### Option A: Relax existing redaction expressions only

Summary: Add identifiers/messages to existing allowlists and remove blanket DID/URL rejection.
Pros: Small initial change; immediately restores some context.
Cons: Does not repair removed causes, stacks, request correlation, duplicate capture, completion verbosity or missing sink coverage.
Risks: Raw exception and direct logger paths can still leak secrets, while other paths stay unhelpful.

### Option B: Classify data and repair diagnostic paths end to end

Summary: Permit useful public context, retain sanitized original errors, and apply consistent protection at every sink; repair correlation and coverage at affected call sites.
Pros: Addresses the verified causes of poor diagnostics across AppView and Flutter; produces testable protection and usefulness guarantees.
Cons: Wider change requiring feature-by-feature verification and updates to old redaction tests.
Risks: Incorrect content classification, volume growth, issue grouping changes and private activity leakage.

## 5. Recommended Direction

Recommended approach: Option B. Implement shared diagnostic contracts first, then fix the F-01–F-13 bottlenecks and extend the same contract across F-14–F-20. All Must requirements remain in scope; ordering does not defer completion.

Why: Allowlisting public IDs alone cannot recover errors already discarded by callers. Classification and sanitization must preserve useful causes without exposing credentials in upstream exception strings, URLs or payloads.

## 6. Problem / Opportunity

A maintainer cannot reliably identify the failing record, dependency, stage or underlying cause from current production diagnostics. Public reference data is unnecessarily discarded, while restrictive summaries and generic fallbacks make unrelated failures look alike. Useful logging should make a failure diagnosable without routinely reproducing it locally or enabling unsafe logging.

## 7. Goals

- G-001: Locate the affected public account/record and the concrete failure cause from one correlated diagnostic trail.
- G-002: Make Flutter and AppView error reports useful at normal production verbosity.
- G-003: Preserve secrets and private-by-intent content across all emission paths.
- G-004: Keep event volume, payload size and issue grouping operationally manageable.
- G-005: Give contributors and coding agents practical repository guidance that prevents logging and reporting regressions.

## 8. Non-Goals

- NG-001: Logging entire requests, responses, repositories, provider state, database rows or media by default.
- NG-002: Exporting private content, authentication material, device identifiers or payment details.
- NG-003: Changing lexicons, persistence schemas, authentication, authorization, publication or worker retry behavior.
- NG-004: Revealing internal stack traces or upstream responses through public API errors or end-user messages.
- NG-005: Adding session replay, screenshots, user recordings, new telemetry vendors or a new metrics stack.
- NG-006: Production deployment, Render/Sentry configuration changes, retention/access changes or live telemetry investigation in this requirements stage. Production verification is a separately authorized rollout step.
- NG-007: A persistent Flutter diagnostic buffer, log-export support UI or dedicated logging skill in this deliverable.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Maintainer/operator | Investigates AppView and app failures | Cause, stack, stage, affected public target and correlation |
| App user | Encounters failure or retries an operation | Clear localized explanation and useful recovery action |
| Contributor | Implements or tests features | Predictable diagnostic contract and safe local output |
| Telemetry sink | Local stdout/developer output and configured Sentry products | Structured, bounded, sanitized diagnostic records |

## 10. Current Behavior

See F-01–F-20. Some paths deliberately discard causes before logging; others hand raw exceptions to a sink. Request IDs exist in API envelopes and local logs but may disappear on export. Flutter has localized error presentation and API classifications, but exported failures often contain placeholder exceptions or fixed messages.

## 11. Desired Behavior

An operator investigating a failed public post read sees the operation, sanitized database or network cause, request ID, public post URI/author DID, status, duration and available stack. The Flutter event for the same request carries that request ID, endpoint, safe API message and original exception details. A scheduled-publication failure exposes account DID, operation, workflow stage, opaque job reference, retry state and a vetted dependency cause, without unpublished text, private media locations or credentials. Expected cancellation remains visible as context without becoming an unexpected-error issue.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A maintainer shall be able to identify the failed operation, affected public target when applicable, and actionable underlying cause from correlated logs/error reports without enabling unsafe logging. | Diagnose production failures | Prompt; F-01–F-20 | AC-001, AC-002, AC-003 |
| FR-001 | Functional | Must | AppView and Flutter shall preserve concrete error/exception type, relevant cause structure and a stack where one exists. Retain sanitized original messages for known-safe sources; private-workflow and unknown third-party errors shall retain vetted explanatory text, type, stack, stage and approved status/code fields without automatically emitting complete messages. Go panic recovery shall capture a stack at recovery; ordinary Go errors shall not invent an origin stack. Retain existing classifications as additional context. | Recover useful causes without relying on recognition of arbitrary private text | F-01, F-02, F-05–F-07, F-11, F-15; grilling Q8 | AC-001, AC-002, AC-021 |
| FR-002 | Functional | Must | Diagnostic records shall retain relevant public actor/target DIDs, handles, AT URIs, NSIDs, record keys, CIDs and credential-free dependency origins/paths when available, identifying actor and target separately. Apply RULE-002/RULE-004 when a public identifier would disclose a private target or activity. | Locate records and federation failures while protecting private workflows | Q1; F-09, F-14, F-18; grilling Q2/Q7 | AC-001, AC-006 |
| FR-003 | Functional | Must | One AppView request correlation ID shall survive unchanged in local logs, exported logs, error events, spans when enabled, API `requestId` and Flutter diagnostics. Attach available trace/span IDs and release/environment metadata; correlation shall work without trace sampling or a Sentry DSN. | Restore the cross-system trail | F-03, F-20 | AC-003 |
| FR-004 | Functional | Must | AppView shall emit structured arrival and completion records at INFO for ordinary incoming HTTP requests, including unmatched routes. Both records shall include method, sanitized actual path and request ID; completion shall additionally include route pattern, status, duration and response bytes. A request that hangs shall still have its arrival record. Successful `/health` and `/healthz` probes shall use DEBUG for arrival/completion; unsuccessful probe completions shall use WARNING for 4xx and ERROR for 5xx/unexpected failures. Server failures shall retain associated actionable cause diagnostics at ERROR. | Make incoming paths, stuck requests and outcomes diagnosable without probe noise | F-04; grilling Q1/Q10 | AC-004, AC-020 |
| FR-005 | Functional | Must | Flutter log forwarding shall preserve vetted or sanitized original log messages according to provenance, severity, logger/feature and operation. WARNING/ERROR logs shall be independently exported when Sentry logging is enabled, rather than available only as breadcrumbs; selected INFO significant operation outcomes shall also be exported. Relevant INFO navigation/operation/lifecycle records shall remain available as breadcrumbs. DEBUG/FINE remote output shall be opt-in. Log export shall not itself imply issue capture. | Retain meaningful searchable history and failure context | F-08, F-20; grilling Q3/Q8 | AC-005 |
| FR-006 | Functional | Must | Flutter API diagnostics shall retain HTTP method, supported route category/pattern, safe target references, status, AppView error code, sanitized message, validation field reasons and request ID. Safe diagnostic messages shall be retained for both 4xx and 5xx; network/parse failures shall preserve type/cause even without an envelope. | Retain known server detail | F-09, F-10 | AC-007 |
| FR-007 | Functional | Must | Each unexpected failure shall have one issue-capture owner per component/operation occurrence. Supporting logs/breadcrumbs may coexist, and linked Flutter/AppView events may both exist. Lower-layer cause capture shall replace the generic 5xx fallback for that request. | Avoid duplicates while retaining evidence | F-06, F-13 | AC-008 |
| FR-008 | Functional | Must | Expected cancellations, ordinary offline conditions, expired sessions, not-found results, validation rejections and normal retries shall retain appropriate context without becoming unexpected-error issues. Unexpected parse failures, exhausted retries and actionable provider/storage failures shall remain reportable. | Signal without blanket suppression | F-08, F-17; existing error policies | AC-009 |
| FR-009 | Functional | Must | Apply the diagnostic contract across API/auth/PDS/DB boundaries, Tap/indexers, background workers/integrations, Flutter providers/account initialization, media/video, storage and mutations. Failures shall include stage, approved dependency/provider status/code and retryability when available; jobs/events shall include safe correlation and attempt/outcome. Private-workflow operational failures may include the operation account DID and opaque non-capability workflow reference, but not private content, target membership or recipient details. | Close feature gaps while making private-operation failures attributable | F-05–F-07, F-11, F-14–F-20; grilling Q2/Q8 | AC-010, AC-011 |
| FR-010 | Functional | Must | Sanitization and provenance-based field selection shall cover messages, errors/causes, structured context, breadcrumbs, traces, headers/URLs, SDK-generated events and local/debug/platform output. Remove protected values while retaining safe surrounding diagnostics; mark redaction/truncation explicitly. Unknown free text shall follow RULE-003 rather than depend on a universal private-text detection regex. | Close raw exception/direct logging bypasses | F-12; grilling Q8/Q9 | AC-012, AC-013 |
| FR-011 | Functional | Must | Public record text or structured payload fragments shall be captured only for an explicit diagnostic purpose on a failure path and only when their published/public provenance is known. Select relevant fields and sanitize before truncation. Unsubmitted or failed-to-publish content shall remain private. | Honor approved public-payload boundary | Q1 | AC-006, AC-013 |
| FR-012 | Functional | Should | Models with public data should offer diagnostic summaries with useful public references and state/counts; models mixing auth/private state should expose selected fields rather than unrestricted `toString` serialization. | Avoid full-redaction dead ends | F-18, F-19 | AC-014 |
| FR-013 | Functional | Must | Reporter/export failures shall not alter business behavior, crash the app, recurse into reporting or suppress local diagnostics. Disabled Sentry shall still permit safe local logging. | Preserve reliability | Existing guarded reporter; F-20 | AC-015 |
| FR-014 | Functional | Must | Provide repository logging and exception-reporting guidance discoverable from root `AGENTS.md`, with a concise mandatory policy and a link to one canonical practical guide. The guide shall cover AppView and Flutter supported logging/reporting entry points, severity, request/trace/job correlation, request-path/URL treatment, public/private/secret classification, cause/stack preservation, capture ownership and deduplication, expected failures, bounded context, sink/environment behavior and relevant verification commands. Include correct and incorrect Go/Flutter examples using the final implemented interfaces, plus a concise contributor/reviewer checklist. Keep guidance consistent with the implemented contract and link it from the existing project-skills/contributor guidance where appropriate. | Prevent future changes from recreating unusable diagnostics or exposing protected data | User follow-up; Q2; F-01–F-20 | AC-019 |
| FR-015 | Functional | Must | AppView shall emit WARNING/ERROR diagnostic logs and selected INFO records for ordinary HTTP arrival/completion and significant operation outcomes to local structured output and independently to Sentry Logs when enabled. Direct slog calls shall not silently bypass the required export contract. Successful health probes remain DEBUG. Sentry log export shall be independent of trace sampling and issue capture; disabled export shall preserve local output. | Make sink coverage explicit | F-20; grilling Q3/Q10 | AC-022 |
| FR-016 | Functional | Must | Flutter release builds shall emit bounded sanitized WARNING/ERROR diagnostics to platform logging, including when Sentry is disabled or unavailable. Platform logging shall retain useful cause, available stack and correlation according to the shared data policy. This change shall not add an application-managed persistent buffer or support UI. | Provide an attached-device fallback outside debug builds | F-20; grilling Q9 | AC-023 |
| FR-017 | Functional | Must | Implementation validation shall include representative failure investigation exercises using local/test sinks, showing how an operator locates the cause, public target or permitted private-workflow reference, and cross-component request trail. Record effective production-sink verification as a separate rollout step requiring its normal authorization; local validation shall not claim live production verification. | Prove operational usefulness beyond constructing events | BR-001; grilling Q6 | AC-024 |
| NFR-001 | Non-functional | Must | Metrics, transaction names and issue grouping shall remain based on stable operations/routes/error classes. Public identities, request/job IDs and payload text shall be context fields rather than metric labels or grouping keys. | Prevent excessive cardinality and issue fragmentation | F-09; existing metrics architecture | AC-016 |
| NFR-002 | Non-functional | Must | Implementation shall choose, document and test bounded per-field/event sizes, cause/collection counts and nesting depth, with explicit truncation. Preserve concrete error type, operation, correlation and useful stack frames before spending the remaining budget on public payload excerpts. Avoid full-body/media copies, extra diagnostic-only network fetches and unbounded worker log loops. Rate-limited suppression shall retain a summary count and terminal cause. | Bound cost without arbitrary limits destroying diagnostics | Q1; F-14, F-20; grilling Q5 | AC-013, AC-017 |
| NFR-003 | Non-functional | Must | Public API error envelopes and localized app messaging shall preserve current contracts and recovery actions. Internal causes/stacks shall remain in operator diagnostics; safe validation details may inform existing user-facing errors. | Diagnostics are not public error disclosure | API architecture; F-10 | AC-018 |
| RULE-001 | Business rule | Must | Never emit auth/session/access/refresh/service tokens, cookies, passwords, OAuth codes/state/PKCE verifiers, DPoP proofs/private keys, webhook credentials, credential-bearing DSNs/connection strings, signed URLs or capability-bearing identifiers. This rule applies in every environment/sink. | Explicit user constraint | Prompt; Q1; AGENTS.md | AC-012 |
| RULE-002 | Business rule | Must | Exclude unpublished drafts/schedules/media, saved-folder names/membership, mute/search history and terms, private moderation reports/evidence, private targets/recipient details, email, device/push identifiers, payment details and local user file paths. Operational failures in private workflows may retain account DID, operation, stage and opaque non-capability job/workflow references; this exception does not permit routine private-activity histories or target disclosure. | Protect private content while enabling operational attribution | Q1; repository private-data rules; grilling Q2/Q7 | AC-011, AC-012 |
| RULE-003 | Business rule | Must | URLs and arbitrary upstream diagnostics shall be treated as mixed/untrusted. Retain approved origins/paths/public references and strip credentials/capabilities. Select approved structured fields; private-workflow and unknown third-party messages shall use vetted explanatory text plus concrete type, stack, stage and approved status/code rather than complete raw messages. Known-safe messages may be retained after sanitization. No universal regex guarantee for arbitrary private text is assumed. | Preserve actionable causes through explicit provenance and data classification | F-09, F-12; Q1; grilling Q8 | AC-012, AC-013, AC-021 |
| RULE-004 | Business rule | Must | Request path logging shall preserve the actual path for public-resource segments and replace private targets, search terms, capability-bearing and unclassified segments with explicit markers. Apply the same protection to context fields so they cannot reintroduce path-redacted values. Omit raw query strings and fragments by default; any approved typed operational parameters shall be separate fields and shall exclude search terms, cursors and authentication state. Unmatched paths shall be bounded and sanitized, with unclassified values redacted. Route patterns remain separate for aggregation. | Log useful incoming paths without disclosing private values | Grilling Q1/Q7/Q10 | AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002 | Given a public post/profile API failure with a wrapped dependency error, when local and Sentry diagnostics are emitted, then they retain distinct sanitized error types/messages/causes, operation/stage, actor and public target identifiers, and available stack information rather than only a generic classification. |
| AC-002 | BR-001, FR-001 | Given an HTTP, Tap or Flutter panic/unhandled failure containing safe detail, when it is recovered/reported, then diagnostics retain the sanitized detail and useful frame information; Go panic stacks identify the recovery call chain and Flutter retains the supplied stack. A normal Go error lacking an origin stack is not presented as having one. |
| AC-003 | BR-001, FR-003 | Given a failed app request, when its request ID is followed from the API envelope, then the same value locates the AppView completion/cause log and enabled exported event/span and Flutter report. This remains true with tracing disabled or unsampled; with Sentry disabled, local logs and API correlation still work. |
| AC-004 | FR-004 | Given INFO-level AppView logging, when an ordinary matched or unmatched request arrives, then an arrival record contains method/sanitized actual path/request ID even if no completion follows. When success, 4xx, 5xx or cancellation completes, then completion retains the same ID/path plus route pattern/status/duration/bytes; 5xx has associated ERROR cause diagnostics and cancellation retains internal canceled/499 classification without changing response behavior. Successful health probes appear only at DEBUG; failed probe completions are WARNING for 4xx and ERROR for 5xx/unexpected failures. |
| AC-005 | FR-005 | Given distinct Flutter message-only errors, exception logs, warnings and selected INFO operation outcomes, when forwarding runs with Sentry logging enabled, then vetted/sanitized messages remain distinguishable and WARNING/ERROR plus selected INFO records are independently present in exported logs. Exception logs retain operation context and relevant navigation/lifecycle breadcrumbs accompany a later failure. DEBUG/FINE export remains opt-in; warning export alone creates no issue; no message collapses to only `App log`. |
| AC-006 | FR-002, FR-011 | Given an already published public record whose text is needed to explain a parse/indexing failure, when diagnostics capture selected fields, then identifiers and the bounded relevant public fragment survive. Given identical text in an unsent draft or failed publication request, then the text is absent and stage/field/type information remains. |
| AC-007 | FR-006 | Given safe 400/401/404/422/500/503 envelopes and representative registered routes including newer business/video/scheduling/subscription/integration routes, when Flutter maps the response, then method/category/status/code/message/request ID and safe field reasons survive into diagnostics. Missing/malformed envelopes retain network/parse information without dumping HTML or raw bodies. |
| AC-008 | FR-007 | Given a Flutter root/framework/platform failure routed through both logging and explicit reporting, then one issue event represents that occurrence. Given an AppView lower-layer failure already captured, then HTTP fallback does not add a generic duplicate. If no lower-layer capture exists, fallback still reports the 5xx. Linked events across components are not incorrectly suppressed. |
| AC-009 | FR-008 | Given normal cancellation/offline/expiry/not-found/validation/retry conditions, then relevant sanitized context remains available without unexpected-error issues. Given exhausted retries, unexpected parsing or actionable provider/storage failures, then cause and stack are retained in a report. |
| AC-010 | FR-009 | Given representative failures in AppView API/auth/PDS/DB/Tap/indexer and Flutter provider/initialization/media/storage/mutation paths, then each retains its cause, feature/operation, failed stage and available public target/dependency information. Tap includes event/record identity and ack/retry/quarantine outcome; video omits JWTs. |
| AC-011 | FR-009, RULE-002 | Given a scheduled publication, push, Instagram integration or deletion-worker operational failure, then diagnostics may expose the account DID, operation, safe workflow correlation, stage/attempt/outcome and approved provider code/vetted cause while omitting unpublished payloads, private target/membership or recipient details, provider credentials and private moderation evidence. Routine successful private activity does not gain account-linked histories through this exception. |
| AC-012 | FR-010, RULE-001, RULE-002, RULE-003 | Given supported error sources with sentinel secrets/private values in protected fields, recognized credential formats, URLs and nested diagnostics, when each local/exported/SDK/platform sink emits, then those values are absent and safe type/message/public reference context remains. Include OAuth/service JWTs, cookies, signed URLs, connection strings, email, draft text, moderation evidence and device/push tokens. Unknown/private-workflow free text follows the vetted-field policy in AC-021 instead of relying on detection of every possible private sentence. |
| AC-013 | FR-010, FR-011, NFR-002, RULE-003 | Given oversized, nested or malformed mixed-content diagnostics, when collection runs, then provenance-based selection and sanitization occur before truncation, output remains within documented implementation size/count/depth limits, omissions are marked, and concrete type/operation/correlation/useful frames take priority over payload excerpts. No binary media or unknown full objects are copied/logged; a recognized secret spanning a truncation boundary remains excluded. |
| AC-014 | FR-012 | Given a diagnostic summary for a public identity/post/event target, then relevant public references and state are visible; given a mixed private/session model, then tokens/private fields stay absent and permitted summary fields remain useful. |
| AC-015 | FR-013 | Given disabled or failing Sentry transport/reporting, when an application operation fails, then its original behavior and local sanitized diagnostics remain available and no recursive reporting loop occurs. Flutter release fallback is covered specifically by AC-023. |
| AC-016 | NFR-001 | Given many different DIDs/record URIs/request IDs with the same operation/error class, then metric dimensions, transaction names and issue grouping remain stable while logs/events retain the distinct context references. |
| AC-017 | NFR-002 | Given repetitive retries and a large public payload, then bounded enrichment performs no extra fetch solely for logging and never buffers full media; repetitive emission is limited with a count/outcome summary and the terminal cause remains visible. |
| AC-018 | NFR-003 | Given failures after enrichment, then `/v1/` responses preserve camelCase `{error, message, requestId}` plus existing optional `fields`, localized UI remains actionable, and raw internal stacks/provider bodies are absent from public responses and user-facing messages. |
| AC-019 | FR-014 | Given a contributor or agent starting from root `AGENTS.md`, when they follow the logging/reporting instructions, then they can find one canonical guide and choose the supported Go/Flutter entry point, severity, capture owner and safe context without relying on an optional skill invocation. The guide includes concrete examples for a public-record failure, a private-workflow failure, request correlation and URL/path sanitization, demonstrates retained causes/stacks and excluded credentials, explains expected-failure handling and duplicate prevention, and lists relevant verification commands and a review checklist. Every code example agrees with the final implemented interfaces and no guidance contradicts the approved public/private boundary. |
| AC-020 | FR-004, RULE-004 | Given public post/profile paths, private save/report/folder/media paths, path-based hashtag search, OAuth queries and unmatched paths, when arrival/completion/context diagnostics emit, then public-resource path references remain, private targets/search/capability/unclassified values become markers, and raw query strings/fragments are absent. Optional separately approved operational fields cannot contain search terms, cursors or auth state, and other context cannot reintroduce redacted private targets. |
| AC-021 | FR-001, RULE-003 | Given a private-workflow or unknown third-party exception containing arbitrary draft text or private targets, when emitted, then its raw message is not automatically included. Concrete type, available stack, stage and approved status/code/vetted explanation remain. Given a known-safe error with useful message/cause detail, then that sanitized detail survives rather than becoming a generic placeholder. |
| AC-022 | FR-015 | Given Sentry Logs enabled with tracing disabled or unsampled, when AppView emits ordinary request arrival/completion, a significant INFO outcome, WARNING or ERROR diagnostics, then the required records appear in both local structured output and exported Logs with matching correlation where available. Successful health probes do not produce INFO export. Disabling export preserves local records without emitting remote logs. |
| AC-023 | FR-016 | Given a Flutter release build with Sentry disabled or transport unavailable, when a WARNING/ERROR occurs, then bounded sanitized platform output retains permitted cause/stack/correlation for attached-device inspection; secrets/private content remain absent and no application-managed persistent buffer or support UI is introduced. |
| AC-024 | FR-017 | Given representative public-read/dependency, Flutter provider/initialization, Tap and private scheduled-publication failures, when local/test investigation exercises are performed, then recorded evidence shows an operator finding the actionable cause, permitted target/workflow reference and request trail in configured sinks where applicable. A separate production verification checklist records authorized rollout checks and remains explicitly unverified until run. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Public identity embedded beside a token in the same error | Keep identity and explanatory cause; redact token | FR-001, FR-010, RULE-001 |
| EC-002 | OAuth callback or presigned media URL | Strip code/state/signature/capabilities; retain only permitted origin/path context | RULE-001, RULE-003 |
| EC-003 | Published record later deleted or moderated | Do not routinely copy public text; use references and only necessary failure excerpts; do not fetch blocked content to enrich logs | FR-011, NFR-002 |
| EC-004 | Public record URI inside a private save/folder/mute/report workflow | Redact private target/membership even when the target URI is public; operational failure may retain account DID, operation/stage and opaque workflow reference | RULE-002, RULE-004 |
| EC-005 | Account switches while an operation fails | Attach the operation's account context, not whichever account is active when reporting finishes; never attach session tokens | FR-002, FR-009, RULE-001 |
| EC-006 | Upstream error includes SQL detail, row values or HTML | Keep query operation and approved SQLSTATE/constraint/status/code; use vetted explanation for private/unknown text, omit row values/parameters/HTML | FR-001, RULE-002, RULE-003 |
| EC-007 | App failure before reporting initialization or missing request ID | Preserve local sanitized cause/stack; explicitly indicate unavailable correlation, never fabricate server IDs | FR-001, FR-003, FR-013 |
| EC-008 | One failure both logged and captured; later failure has same text | Deduplicate the same occurrence, preserve separate real occurrences | FR-007 |
| EC-009 | Malformed DID/URI from Tap or request boundary | Retain bounded sanitized attempted public identifier and validation reason when safe; do not claim parsed validity | FR-002, FR-010 |
| EC-010 | Panic after HTTP response starts | Capture actionable panic diagnostics without rewriting the response or changing recovery behavior | FR-001, NFR-003 |

## 15. Data / Persistence Impact

- New fields: Diagnostic fields/context only; no application data columns proposed. Flutter platform logging adds no application-managed persistent diagnostic storage.
- Changed fields: Broader approved log/event context and sanitized exception details; request correlation preserved on export.
- Migration required: None.
- Backwards compatibility: Existing API contracts retained. Logging schemas and issue grouping may change; update affected operational queries/tests during implementation. Production telemetry retention/access settings remain unchanged by this scope.

## 16. UI / API / CLI Impact

- UI: Preserve localized messages and existing retry/sign-in/navigation actions. No new developer detail screens or mandatory support-code UI.
- API: No new routes or expanded public internal-error disclosure. Flutter retains existing safe error envelope fields for diagnostics.
- Contributor tooling: Add concise mandatory guidance in root `AGENTS.md` and the canonical guide at `docs/engineering/logging-and-error-reporting.md` during implementation; no new `.agent/rules` directory or mandatory skill is proposed.
- CLI: Apply the same emission policy to shared AppView code used by CLI; credentials/config dumps remain excluded.
- Background jobs: Add failure and lifecycle correlation without changing scheduling, retry, authorization, deletion or acknowledgement decisions.

## 17. Security / Privacy / Permissions

- Authentication: Server-side OAuth/TMB and the memory-only video service JWT boundary stay intact.
- Authorization: Diagnostic enrichment cannot fetch additional records or bypass blocked/private access.
- Sensitive data: Q1 and RULE-001–RULE-003 govern local and exported diagnostics equally. “Public” refers to published provenance, not a field that may become public later.
- Abuse cases: Bound attacker-controlled strings, preserve structured encoding, redact nested credentials, and avoid logging arbitrary headers/URLs or binary data. No raw request/response-body mode is enabled by this requirement.

## 18. Observability

- Events: Sanitized original exceptions and panic stacks, stable classifications, release/environment, correlation and useful safe context.
- Logs: Structured AppView arrival/completion/cause logs with sanitized actual paths, meaningful Flutter messages/warnings and selected INFO operation outcomes; operational job failures and dependency failures. WARNING/ERROR and selected INFO export is independent of issue capture and trace sampling.
- Metrics: Keep existing bounded vocabulary. Do not introduce raw identities, payloads or per-request dimensions.
- Alerts: Existing error monitoring remains; no new alert destinations or thresholds required. Retain terminal failures and suppression summaries for investigation.
- Bounds: Implementation selects and documents tested per-field/context/event, cause-count and nesting limits appropriate to each sink. Preserve type, operation, correlation and useful stack frames before optional public excerpts; sanitize/select before truncation and obey SDK limits. No numerical limits are fixed by this requirements document.
- Severity: Ordinary HTTP arrival/completion INFO, including unmatched requests; successful health probes DEBUG; failed probes WARNING for 4xx or ERROR for 5xx/unexpected failures. Unexpected/terminal failures ERROR; recoverable retry/degradation WARNING; verbose intermediate state DEBUG/FINE. Ordinary warning export does not imply issue capture.
- Sink availability: AppView local structured output remains independent of DSN; enabled Sentry Logs receive required WARNING/ERROR and selected INFO records through an explicit export path. Flutter uses enabled Sentry logging/reporting plus bounded sanitized platform WARNING/ERROR logging in release builds; debug output stays safe. No new persistent buffer. Existing export/tracing controls remain; production settings changes require their normal review.
- Investigation evidence: Local/test failure exercises establish operational usefulness during implementation. Effective production export and operator retrieval are verified in a separate authorized rollout step, with pending checks explicitly recorded.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Expanded errors leak embedded secrets or unpublished/private values | High | Explicit field classification and emission-boundary sanitization; cross-sink sentinel tests; approve policy before test design |
| RISK-002 | Public identifiers reveal private targets or activity when combined with paths/context | High | Redact private targets/search terms across paths and context; limit account/operation attribution to operational failures and safe workflow references |
| RISK-003 | Volume/cardinality and copied public text grow storage and noise | Medium | Selected failure excerpts, bounded enrichment, stable grouping/metric labels and repeated-error summaries |
| RISK-004 | Restored exception types change Sentry issue grouping | Medium | Keep stable classification, verify distinct causes group sensibly and do not use record IDs as grouping keys |
| RISK-005 | Legacy tests/specs reject useful public context | Medium | Explicitly replace superseded public-data expectations while retaining credential/private-content regression tests |
| RISK-006 | Source behavior differs from effective production sink configuration | Medium | Follow-up verification with test transports and authorized deployment/configuration checks; no claim that live diagnostics are already improved |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Existing telemetry access and retention arrangements remain appropriate for selectively included public context | Review operational policy before production release; no settings are changed here |
| ASM-002 | Opaque internal workflow IDs are non-capability identifiers and useful for private-workflow debugging | Omit any identifier that grants access; use a purpose-built non-capability correlation reference |
| ASM-003 | Appropriate bounded context sizes can be chosen against the final interfaces and sink limits during implementation | Document and test chosen bounds, prioritizing core cause/correlation/frames; no fixed numerical limits are assumed |
| ASM-004 | Scope is requirements for improvement across both codebases, followed by a separate approved implementation stage | No source behavior is changed in this stage |

## 21. Open Questions

- Blocking discovery questions: None; the public/private data boundary is confirmed.
- No unresolved design choices after both grilling rounds. Implementation shall document and verify the chosen size/count/depth limits under NFR-002.
- Non-blocking release follow-up: Verify effective sink enablement and existing telemetry retention/access before rollout; live credentials/configuration are outside this requirements audit.

## 22. Review Status

Status: Reviewed
Risk level: High
Review recommended: Required; completed through the user-confirmed grilling decisions
Reviewer: User/maintainer through the grilling interview
Date: 2026-10-05
Notes: The user agreed to all recommendations in both grilling rounds, confirmed shared understanding, and authorized this document update. Those decisions are incorporated here. This records review and decision confirmation; the next workflow stage has not been requested. Expanding diagnostics still carries High risk around credential-bearing paths and private workflows. No source or production behavior changed.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-02-logging-error-reporting/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001; FR-001–FR-011; FR-013–FR-017; NFR-001–NFR-003; RULE-001–RULE-004.
- Suggested test levels: Sanitizer/error-envelope/log-forwarder units; middleware and provider capture integration tests with fake transports; representative feature/worker regression tests; cross-sink secret/private sentinel tests; manual local failure investigations for public reads, OAuth/PDS failures, provider initialization, Tap malformed records and private scheduled publication.
- Verify usefulness positively: assert concrete type/message/cause/stack/public-reference retention, not merely absence of sensitive markers. Verify final emitted SDK payloads as well as helper output.
- Verify guidance discoverability and consistency against the final Go/Flutter interfaces and approved policy; review the examples and checklist as documentation acceptance coverage.
- Verify no duplicates and unsampled request/log correlation. Cover account-switch attribution, provenance-based exception selection, actual-path privacy, probe exceptions, independent log export and Flutter release platform fallback.
- Record local/test investigation evidence and a separate pending production verification checklist; production mutation/verification requires its normal authorization.
- Blocking open questions: None. User-confirmed grilling decisions are recorded; proceeding to `write-acceptance-tests` is a separate requested stage and constitutes approval to use these revised requirements. No production action is authorized by document review.
