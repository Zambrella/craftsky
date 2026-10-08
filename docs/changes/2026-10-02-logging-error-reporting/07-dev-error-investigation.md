# Dev AppView diagnostic investigation — 2026-10-07

## Scope and evidence

Maintainer requested correction of stale release/noisy omission logs plus read-only investigation and recommendations for recent ERROR entries. Read local worktree AppView logs, Sentry project `craftsky/app-view` for environment `dev`, and local Postgres using explicit read-only transactions. No production mutation, Sentry issue mutation, record write/replay, deploy or restart.

Local snapshot: 587 structured records (569 DEBUG, 10 INFO, 4 WARN, 4 ERROR); 568 include the generic omission marker. 512 messages are `tap record event received`; reviewed event ID/action/record-byte metadata was rejected. Preserve that metadata, preserve startup listener/version, quietly drop unsupported/private fields and reserve the marker for actual field/count budget exhaustion. `SENTRY_RELEASE=1.8.14` in dev.env overrides the embedded semantic version; it is now blank. Apply these source/config changes with a local container rebuild/recreation. Current running container still uses the old build.

## Four ERROR entries

Sentry Logs returned the same four entries as local output, at 16:27:16, :17, :18 and :23 UTC. [APP-VIEW-4](https://craftsky.sentry.io/issues/152154813/) contains the two projection quarantine events, grouped under `*errors.errorString: appview unexpected error`.

### Published business profile rejected (event 172)

- NSID `social.craftsky.business.profile`, rkey `self`.
- Stored source validation reason: `invalid_lexicon`; projection reports `malformed_record` and durably quarantines it.
- Read-only Go validation probe reproduces schema error `blob size too large: 11189477`; generated JSON type decoding succeeds.
- Product image blob sizes: 11,189,477 and 12,347,265 bytes. Current business image schema maximum is 2,000,000 bytes.
- Recommendation: enforce the business-specific 2 MB limit at authoring/upload boundaries, compress/reject before submission, and have the owner replace/resave the oversized images. Keep ingestion validation. Do not loosen the lexicon or silently rewrite/delete PDS records.

### Published post rejected (event 178)

- NSID `social.craftsky.feed.post`.
- Stored source validation reason: `invalid_lexicon`; projection reports `malformed_record` and durably quarantines it.
- Public record keys: `$type`, `createdAt`, `langs`, `text`; required `sponsored` boolean is absent. Generated JSON type decoding succeeds but schema/source validation rejects the missing required field.
- Recommendation: repair this old record through an explicit owner-authorized edit including the correct sponsorship declaration. Current API authoring already requires the boolean. Keep source validation; do not silently infer sponsorship in the projection.

### Two repository batch failures

- Both local ERROR records contain `reason=lease_lost`, at 16:27:18.878 and 16:27:23.762 UTC.
- Local jobs all subsequently completed; two PDS reconciliation jobs required two attempts, the remaining observed jobs completed on one attempt. There is no pending backlog in the inspected rows.
- Source explanation: enqueue/upsert resets lease token/owner/expiry even when an existing repository job is processing. Active handlers then fail the conditional finalization/reschedule predicate. New Tap source/identity activity can request another reconciliation during initial sync. Timing (roughly 3 and 8 seconds after startup, versus 45-second configured lease) and successful retry make superseded claims a plausible cause.
- Confidence: `lease_lost` and eventual completion are confirmed. The exact trigger/owner for each failed occurrence is not captured. `repositoryJobFailureReason` prioritizes `errors.Is(ErrProjectionLeaseLost)`, even when an errors.Join may also contain another cause; current batch logs omit the original error. Do not claim expiry or a pure benign supersession was conclusively established.
- Recommendation: retain per-job original typed cause plus public owner/job kind/stage/attempt and distinct lease outcomes. Pure, confirmed supersession can be a contextual lower-severity log; genuine expired leases, mixed failures and repeated inability to progress should remain actionable. Avoid simply increasing lease duration or weakening fencing. Consider coalescing repeat enqueue triggers while preserving a follow-up reconciliation, with dedicated concurrency tests, only as a separate approved business/worker change.

## Diagnostic quality follow-up

Projection validation discards the original validator explanation: dispatcher converts invalid source validation to `malformed_record` with nil cause; Tap diagnostics then manufacture `errors.New("published record rejected")`, which final selection turns generic. This groups unrelated validation failures and hides the useful rule, though record URI/CID/NSID/event ID remain available.

Recommendation: preserve a typed validation failure and reviewed static rule explanation with bounded field path/actual-size/limit or missing-required-field diagnostics; retain `invalid_lexicon` accurately. Do not export arbitrary dependency prose or record bodies. This is a recommendation, not a change made in this usability correction.

## Implemented diagnostic follow-up

Maintainer approved the diagnostic recommendations and explicitly requested lexicon failures at WARN. Typed validator causes and reviewed rule explanations now survive into local and native Sentry WARN Logs without creating an issue. Durable quarantine reason/commit and ACK behavior remain unchanged; the diagnostic reason is `invalid_lexicon`. No record body is exported on this path.

Repository worker logs now retain original typed causes and public owner/job operation/stage/attempt, rather than a cause-less batch summary. Read-only inspection after a failed fenced update distinguishes confirmed supersession from expired or unknown claims. Pure supersession is WARN below the configured attempt alert threshold; mixed, expired, unknown and repeated failures remain ERROR. No queue coalescing or lease policy change was made. Existing image authoring-limit improvements and owner record repairs remain recommendations.

## Verification and limits

Source/config/test changes are confined to diagnostic selection/budget presentation and the dev release override. Temporary probes/output are outside committed artifacts; no credential/DSN/raw payload is included here. Diagnostic follow-ups above are implemented; authoring/record-repair recommendations remain pending and records are unchanged. Go verification is recorded in 05-implementation-plan and 06-validation-evidence. Live Sentry retrieval is confirmed for this dev run; physical Flutter release-device symbolication remains separate.

## Superseded reconciliation follow-up 2026-10-08

The reported 08:08:32 UTC attempt-one failure was followed by successful attempt two at 08:08:34 UTC; all ten inspected local repository jobs were complete. A source-change error joined with a superseded rescheduling lease was previously treated as ERROR, with generic explanations. ERR-003 now makes every pre-threshold recoverable repository failure WARN/no issue, retains the safe source-change explanation and prioritizes `source_changed` rather than hiding it behind lease supersession. Maintainer chose escalation at the existing default five-attempt alert threshold while recovery continues: ERROR plus one explicit Sentry issue per failed attempt thereafter. No queue, PDS, lease, retry or production policy change. Earlier conservative severity recommendation is superseded by this approved amendment.
