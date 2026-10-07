# Logging and error reporting

Choose public/private context at the operation boundary, preserve the original typed cause and caught stack, and keep diagnostics independent of business behavior. Logging must not change responses, retries, authorization, acknowledgements or localized recovery actions.

## Ownership and entry points

Logs never implicitly create an issue. Provider, framework, platform and unhandled boundaries own propagated unexpected failures. An operation that consumes an actionable failure owns an explicit capture. Supporting logs need no ownership flag. Expected cancellation, offline, validation, expiry and not-found failures remain logs; an exhausted actionable failure can be marked terminal without changing its retry policy. Sentry's normal duplicate-exception handling remains enabled.

AppView uses the process `slog` logger through `NewDiagnosticHandler` and the existing `observability.Observer`. Use `LogDiagnostic` for a cause-bearing supporting log, `CaptureDiagnostic` at the issue owner, `ReportRequestFailure` in HTTP handlers, and `ObservePrivateFailure` at worker failures. Preserve `%w` and typed errors. Ordinary errors have no invented origin stack; recovered panics retain recovery-time frames. Fresh command causes stay in memory, outside persisted command results.

Flutter uses `Logger`, optional `DiagnosticMessage` context and the provided `ErrorReporter`. Pass the original exception and caught stack. Never import Sentry in feature code. `GuardedErrorReporter` and local fallback keep reporter failures out of application control flow.

## Messages and private data

Developer-written static log messages need no registry. The local filter and Sentry final hooks bound them and scrub recognized credentials, email, local paths and capability URLs. This is a safety net, not a classifier for private prose: never interpolate an exception, draft, model, request body/header or arbitrary user text into a message. Unknown exception messages are generic. Vetted typed explanations, provider status/code, original types and available stacks remain useful.

Always exclude tokens/JWTs, OAuth state/code/PKCE, DPoP/private keys, cookies, authorization, DSNs, signed URLs, email, payment details, device/push IDs, moderation/report/evidence content, recipient routing, drafts, scheduled payloads and private membership/targets. Public identifiers do not make exception prose public.

Use `PublicRecordContext` only for an explicitly public workflow: actor, distinct target, AT URI, CID, NSID, handle and record key already known at the boundary. Do not fetch data solely to enrich diagnostics. `PublishedRecordParseFailureContext` permits a bounded excerpt from an already published record failing parsing/indexing; failed publication input remains private.

Use `PrivateFailureContext`/`ObservePrivateFailure` in Go and `PrivateOperationalFailureContext` in Flutter only on actual failures, selecting the initiating account DID and an existing non-capability UUID work reference. Do not log private success history. Scheduled Go work applies `WithPrivateDiagnosticContext` before dependency calls; lower PDS reports retain only those permitted references. Capture the initiating identity before awaiting work so an account switch cannot relabel a late failure.

## Client/server separation and correlation

The JSON API remains `{error, message, requestId}`. Flutter diagnostics select bounded stable error code, HTTP status/method, the returned `appViewRequestId`, and the original client cause/stack. The client owns localized UI messages and feature operation names. It does not ingest server message/validation prose or resource paths into telemetry and has no mirrored route/message/code catalogue. Unknown syntactically valid stable codes remain useful.

AppView retains detailed server diagnostics, `run_id`, server `request_id`, trace/span IDs and classified sanitized incoming paths separate from low-cardinality route patterns. Reuse the request context and route resolver; never feed raw paths/query/capability values to a sink. Private/search/OAuth/unmatched segments are redacted. IDs belong in diagnostic context, outside metric labels and issue grouping tags. Missing correlation stays missing.

## Sinks and bounds

Sentry `beforeSend`, `beforeSendLog`, `beforeBreadcrumb` and transaction callbacks are the final SDK filtering boundary. Preserve SDK exception/stack structure and ordinary deployment/runtime metadata. Strip request/user/private extra data, attachments, unknown custom contexts, exception prose and frame locals/source. Go excludes thread payloads while preserving protected exception stacks. Flutter also filters attachment envelope items at the transport boundary, because the SDK adds scope attachments after event callbacks; their loaders never run or enter the SDK cache. Custom operation/workflow contexts have small scalar limits. No nonce registry, event provenance Expando or complete event reconstruction is needed.

Local output remains independently protected when Sentry is disabled or fails. Go keeps its bounded structured `slog` record and targeted worker retry suppression. Flutter prints one parseable brief JSON record (at most 8192 bytes), containing original cause types, up to two useful frames, operation and normal request/job correlation. It uses fixed field/count limits and whole-field omissions for oversized identifiers; Sentry carries richer detail. There is no chunk reconstruction or generic Flutter retry-window manager. Developer text is scrubbed before UTF-8 bounding, ordinarily at most 2048 bytes.

Flutter automatic request capture, performance tracing, replay and default PII collection remain disabled. Logs, traces and metrics retain their independent configured gates. The product AOT fixture verifies the console adapter; attached iOS/Android release-console retrieval remains a separate device check. Export/flush failures use direct local fallback without re-entering telemetry. Do not change production gates/access/retention as part of a logging refactor.

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

- Preserve original typed causes, supplied stacks, vetted status/code and permitted references.
- Choose public/private context at the source; never serialize models or interpolate private prose.
- Assign one explicit owner; supporting logs never capture implicitly.
- Keep server path diagnostics and client operation/correlation diagnostics separate.
- Verify serialized local and SDK output with both retained positive fields and protected-value canaries.
- Run focused suites, `just test`, Flutter tests and analysis. Preserve response/UI/retry/fencing/ACK behavior.
- Record unexecuted device/production checks in [validation evidence](../changes/2026-10-02-logging-error-reporting/06-validation-evidence.md). Mock envelopes do not prove production retrieval; do not inject synthetic production failures.
