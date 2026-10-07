# Logging and error reporting

Choose public/private context at the operation boundary, preserve the original typed cause and caught stack, and keep diagnostics independent of business behavior. Logging must not change responses, retries, authorization, acknowledgements or localized recovery actions.

## Ownership and entry points

Logs never implicitly create an issue. Sentry's Flutter framework/platform/unhandled integrations own automatic unexpected failures; Riverpod owns provider failures. An operation that consumes an actionable failure owns an explicit capture. Supporting logs need no ownership flag. Expected cancellation, offline, validation, expiry and not-found failures remain logs; an exhausted actionable failure can be marked terminal without changing its retry policy. Sentry's normal duplicate-exception handling remains enabled. The final event hook applies the existing expected-failure classifier to the original throwable decorated by SDK automatic integrations. Explicit owners classify before capture with their operation outcome, so terminal overrides are preserved; cancellation remains suppressed.

AppView uses the process `slog` logger through `NewDiagnosticHandler` and the existing `observability.Observer`. Use `LogDiagnostic` for a cause-bearing supporting log, `CaptureDiagnostic` at the issue owner, `ReportRequestFailure` in HTTP handlers, and `ObservePrivateFailure` at worker failures. Preserve `%w` and typed errors. Ordinary errors have no invented origin stack; recovered panics retain recovery-time frames. Fresh command causes stay in memory, outside persisted command results.

Flutter uses `Logger`, optional `DiagnosticMessage` context and the provided `ErrorReporter.captureException`. Pass the original exception and caught stack. Never import Sentry in feature code. The reporter exposes only exception capture and enabled state; `GuardedErrorReporter` keeps its failures out of application control flow. Install local-only framework/platform callbacks before Sentry initialization, then leave SDK handlers installed. The SDK appRunner owns the application error zone where the platform requires one. Disabled/unavailable Sentry still runs the app with protected local output and the rendering fallback.

## Messages and private data

Developer-written static log messages need no registry. The local filter and Sentry final hooks bound them and scrub recognized credentials, email, local paths and capability URLs. This is a safety net, not a classifier for private prose: never interpolate an exception, draft, model, request body/header or arbitrary user text into a message. Unknown dependency exception messages remain generic. Reviewed static application state failures use `DiagnosticStateError('static explanation')` (still a StateError); Go static wrapper explanations use `WrapError('static explanation', originalCause)`. Never put interpolation, input, payloads or dependency prose in these explanations. Safe runtime explanations and selected provider status/code survive; HTTP/database/platform exceptions never lend their raw payloads permission. There is no per-message registry.

Always exclude tokens/JWTs, OAuth state/code/PKCE, DPoP/private keys, cookies, authorization, DSNs, signed URLs, email, payment details, device/push IDs, moderation/report/evidence content, recipient routing, drafts, scheduled payloads and private membership/targets. Public identifiers do not make exception prose public.

Use `PublicRecordContext` only for an explicitly public workflow: actor, distinct target, AT URI, CID, NSID, handle and record key already known at the boundary. Do not fetch data solely to enrich diagnostics. `PublishedRecordParseFailureContext` permits a bounded excerpt from an already published record failing parsing/indexing; failed publication input remains private.

Use `PrivateFailureContext`/`ObservePrivateFailure` in Go and `PrivateOperationalFailureContext` in Flutter only on actual failures, selecting the initiating account DID and an existing non-capability UUID work reference. Do not log private success history. Scheduled Go work applies `WithPrivateDiagnosticContext` before dependency calls; lower PDS reports retain only those permitted references. Capture the initiating identity before awaiting work so an account switch cannot relabel a late failure.

## Client/server separation and correlation

The JSON API remains `{error, message, requestId}`. Flutter diagnostics select bounded stable error code, HTTP status/method, the returned `appViewRequestId`, and the original client cause/stack. The client owns localized UI messages and feature operation names. It does not ingest server message/validation prose or resource paths into telemetry and has no mirrored route/message/code catalogue. Unknown syntactically valid stable codes remain useful.

AppView retains detailed server diagnostics, `run_id`, server `request_id`, trace/span IDs and classified sanitized incoming paths separate from low-cardinality route patterns. Reuse the request context and route resolver; never feed raw paths/query/capability values to a sink. Private/search/OAuth/unmatched segments are redacted. IDs belong in diagnostic context, outside metric labels and issue grouping tags. Missing correlation stays missing.

## Sinks and bounds

Sentry `beforeSend`, `beforeSendLog`, `beforeBreadcrumb` and transaction callbacks are the final SDK filtering boundary. Preserve SDK exception/stack structure and ordinary deployment/runtime metadata. Strip request/user/private extra data, attachments, unknown custom contexts, untrusted exception prose and frame locals/source. Preserve reviewed static explanations and native symbolication addresses/package metadata. Go excludes thread payloads while preserving protected exception stacks. Flutter also filters attachment envelope items at the transport boundary, because the SDK adds scope attachments after event callbacks; their loaders never run or enter the SDK cache. Use native SDK exception chains and stacks. Dart registers supported cause extractors for AppError/API/Dio wrappers; Go uses SetException with bounded graph fallback and removes SDK capture-time stacks from ordinary stackless errors. Panic recovery stacks remain separate and intact. Do not encode causes/stacks again in log attributes or a custom failure context. Custom operation/workflow contexts have small scalar limits. No nonce registry, event provenance Expando or complete event reconstruction is needed.

Official `sentry_logging` emits INFO+ Logs and breadcrumbs with its issue threshold OFF; `sentry-go/slog` emits Logs only after local source selection. Logs contain messages, severity, SDK metadata and logger (plus Go selected attributes and a brief scalar error type/status/code). Flutter operation/workflow/request references live on native issues and their supporting log breadcrumbs, rather than duplicated per-log exception JSON. Never serialize LogRecord.object/error/stack wholesale into breadcrumbs. SDK navigation observers are wired to the root, authenticated shell and each tab; GoRouter's static names/path patterns are selected, arguments removed before formatting, and observer transactions disabled. Safe navigation, app lifecycle and connectivity SDK breadcrumbs are retained; private URLs, fields and unknown categories are excluded. Go breadcrumb scopes are isolated per request/operation, including when tracing is disabled.

Local output remains independently protected when Sentry is disabled or fails. Go keeps its bounded structured `slog` record and targeted worker retry suppression. Local cause summaries remain brief and independent of the SDK exception representation. Flutter prints one parseable brief JSON record (at most 8192 bytes), containing original cause types, up to two useful frames, operation and normal request/job correlation. It uses fixed field/count limits and whole-field omissions for oversized identifiers; Sentry carries richer detail. There is no chunk reconstruction or generic Flutter retry-window manager. Developer text is scrubbed before UTF-8 bounding, ordinarily at most 2048 bytes.

Flutter automatic request capture, performance tracing, replay and default PII collection remain disabled. Logs, traces and metrics retain their independent configured gates. The product AOT fixture verifies the console adapter; attached iOS/Android release-console retrieval remains a separate device check. Explicit issue-export/flush failures use direct local fallback without re-entering telemetry; normal local logging does not depend on the SDK log export succeeding. Do not change production gates/access/retention as part of a logging refactor.

## Examples

Flutter at an owner that consumes a failure, using the provided reporter and feature logger:

```dart
final context = ReportContext(
  feature: 'Post', operation: 'read', classification: 'post.read',
  workflow: PublicRecordContext(actorDid: actorDid, targetDid: targetDid),
  safeDiagnostics: {'appViewRequestId': requestId},
);
log.severe(DiagnosticMessage('Post load failed', context: context), error, stack);
await reporter.captureException(error, stackTrace: stack, context: context);
```

If the failure propagates to a provider/framework boundary, let that boundary capture instead. For a private consumed failure, use `PrivateOperationalFailureContext(accountDid: initiatingAccountDid, workflowRef: existingJobId)` rather than public record context.

Go supporting log plus explicit issue owner:

```go
input := observability.DiagnosticInput{
    Error: cause,
    Context: observability.EventContext{
        "component": "api", "operation": "post.read", "failure_stage": "query", "result": "error",
    },
    Workflow: observability.PublicRecordContext{ActorDID: actorDID, TargetDID: targetDID},
}
observability.LogDiagnostic(ctx, logger, input)
observer.CaptureDiagnostic(ctx, input)
```

`ObservePrivateFailure` already owns worker capture eligibility and its supporting log. Do not capture it again.

## Verification checklist

- Preserve native typed cause chains, supplied/attached stacks, reviewed static explanations, vetted status/code, native symbolication metadata and permitted references.
- Choose public/private context at the source; never serialize models or interpolate private prose.
- Let SDK integrations own automatic errors, Riverpod own provider failures, and consumed failures capture explicitly; supporting logs never capture implicitly.
- Keep server path diagnostics and client operation/correlation diagnostics separate.
- Verify serialized local and SDK output with both retained positive fields and protected-value canaries, safe breadcrumb timelines and isolated request scopes. Verify production release symbolication separately; upload configuration alone is not proof.
- Run focused suites, `just test`, Flutter tests and analysis. Preserve response/UI/retry/fencing/ACK behavior.
- Record unexecuted device/production checks in [validation evidence](../changes/2026-10-02-logging-error-reporting/06-validation-evidence.md). Mock envelopes do not prove production retrieval; do not inject synthetic production failures.
