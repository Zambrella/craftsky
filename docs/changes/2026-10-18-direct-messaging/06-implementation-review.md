# Implementation Review: Direct Messaging

## Verdict

Status: Changes required  
Reviewer: Codex, renewed source and evidence review in the current worktree  
Date: 2026-10-20  
Risk level: High

## Summary

The five original findings, IR-001–IR-005, are addressed in the current source
with substantially stronger boundary regressions. One additional rate-limit
lifecycle gap remains: a heartbeat's 429 response loses its retry deadline when
its view/foreground version changes before that response completes. A queued
restart or subsequent resume can then retry immediately.

IR-006 is required before merge or implementation handoff. It is established by
source control flow, not a newly written/executed failing reproduction. Existing
focused tests pass but do not cover this ordering. This review changes only this
artifact; it does not apply the fix or alter tests/implementation notes.

The owner retains MAN-001–MAN-004 and AT-024. Live Bluesky report acceptance is
assumed/unverified under the owner's instruction. Those accepted evidence
boundaries are separate from the remaining local scheduling defect.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-006 | Important | Behavior / Tests | [P2] In-flight heartbeat rate limits are discarded after a view/lifecycle change. The catch block records Retry-After only while `_canPoll && version == _version`; changing foreground or view increments the version. A late 429 therefore records no deadline, and the queued restart can issue another request immediately rather than waiting 120 seconds. | NFR-004, NFR-005, FR-015; AC-017/022; UT-006, IT-020; `app/lib/chat/services/chat_sync_coordinator.dart:106`, `:180`, `:200`, `:233`; `app/test/chat/sync_coordinator_test.dart:82` | Preserve rate-limit scheduling guidance for the same current account/messaging generation even when its presentation version or foreground state changes. Keep stale-content publication fenced, exclude disposed/changed-owner work, and ensure queued/manual/resume requests cannot precede the retained deadline. Add held-response fake-clock regressions before implementation. |

### IR-006: Lifecycle change while the rate-limited response is pending

Concrete same-owner/same-generation sequence:

1. Start a heartbeat and hold its response. `_poll()` captures version V and is
   busy (`chat_sync_coordinator.dart:139–155`).
2. Background and resume before completion. Both transitions increment `_version`;
   resume calls `retry()`, which sees busy and sets `_restart = true`
   (`:106–114`, `:130–134`). An active conversation/view switch has the same
   version/restart behavior (`:92–103`).
3. Complete the original heartbeat with HTTP 429 / Retry-After 120. The HTTP
   mapping fixed by IR-003 correctly supplies that delay, but the catch guard
   rejects the old presentation version (`:180`). `_retryNotBefore` is never
   assigned (`:192–200`). The captured account operation is still current;
   ordinary foreground transitions do not invalidate its account lease.
4. Finally clears busy and creates a zero-delay timer because restart is queued
   and no deadline was retained (`:230–236`). The next heartbeat runs immediately.
   If the 429 instead arrives while backgrounded, `_canPoll` is false and the
   delay is likewise discarded; a later resume can immediately retry.

The production `ChatActivityScope` keeps watching the coordinator while its
AppLifecycleListener changes the view registration
(`app/lib/chat/widgets/chat_activity_scope.dart:27`, `:65–85`), so this ordering
does not require changing accounts or constructing an artificial API caller.

This matters beyond a cheap local revision read: `Heartbeat()` performs current
eligibility and, for a conversation, `getConvo` through the provider path
(`appview/internal/chat/activity.go:46–49`, `service.go` withRPC). Retrying that
consumer call can repeat the rate-limited upstream work.

Current retry tests wait for the error to settle and establish its deadline
before invoking retry/background/resume (`sync_coordinator_test.dart:82–90`,
`:163–174`). The held-response lifecycle test releases a successful response,
not a rate-limited one (`:217–226`). Neither protects this ordering. A safe fix
must distinguish account-scoped scheduling metadata from view-specific result
publication; it must not transfer a previous owner's deadline or content to a
new activation.

## Disposition Of Prior Findings

| Finding | Disposition | Current evidence |
|---|---|---|
| IR-001: upstream access denial | Addressed | currentConversation joins authoritative 403/404/410 with ErrInaccessibleConversation while retaining the typed cause. Real migrated service/registered conversation and heartbeat tests assert safe 404 codes and a transient 500 control. Flutter feeds that response through the production repository/interceptor into the held-completion privacy test. |
| IR-002: preflight starvation | Addressed | Poll claims before selection/discovery, imposes the total deadline and finalizes errors through claim-token CAS/backoff. Scheduler failures yield to other owners. Real-DB tests cover two early failures plus a healthy third owner, selector/120-second backoff and cross-worker capacity. The corrected fixture joins both failure callbacks before cancellation. |
| IR-003: missing HTTP Retry-After | Addressed for its original boundary | writeChatError emits normalized RetrySeconds. The real service/HTTP and repository/coordinator tests preserve 120 seconds once the response settles. IR-006 is a distinct remaining ordering gap after propagation, not a claim that the header fix is absent. |
| IR-004: failed send receipt | Addressed | First and duplicate rejected sends return the authoritative failed receipt with one physical upstream send. The repository/controller preserves failed state and local text. |
| IR-005: uncertain-send reconciliation | Addressed | Uncertainty triggers receipt and independent authorized history reads. Explicit warned repeat reconciles before allocating a fresh UUID. Confirmed originals prevent another send, missing receipts still permit history refresh, matching text never confirms, and busy/owner/content-epoch guards preserve later edits and private boundaries. |

## Requirement And Test Traceability

- The requirement-to-evidence index in `05-implementation-plan.md:638` maps all
  local Must behaviors to their owning Go/Flutter suites. Reviewed high-risk
  seams retain optional capability consent, fixed chat/report audiences,
  server-held OAuth/DPoP, single-attempt ledgers, exact-child/lifecycle/generation
  fencing, no transcript archive and account-isolated device drafts.
- The correction pass follows the five review-directed red/green loops and
  records their requirement/test links, fixture corrections and broad gates.
  New tests exercise real service/HTTP and repository/controller boundaries;
  they are not only injected normalized errors.
- The failed-send HTTP 200 receipt is a justified correction to the earlier
  coding-plan error-envelope description: IR-004 requested authoritative receipt
  preservation, and the implementation notes plus executable consumer contract
  record the resulting behavior. No authority expansion is involved.
- No unrelated feature, public Lexicon change, generic RPC proxy, broader chat
  permission fallback, desktop app target or production action was identified
  in the reviewed correction paths. The existing feature diff remains local and
  uncommitted; this is a renewed review of that feature, focused on corrections
  and their adjacent failure paths.
- Remaining local gap: IR-006 / UT-006, IT-020 / AC-017, AC-022. Existing passing
  tests do not establish the missing in-flight rate-limit lifecycle behavior.
- Remaining external evidence: MAN-001–MAN-004, AT-024, actual granular grants,
  third-party chat/report behavior, real OS push/taps, secure-storage platform
  execution and assistive technology remain owner-run. Dedicated operation-key
  provisioning remains a local/release configuration prerequisite.

## Test Evidence

Newly executed during this review:

- `flutter test --no-pub test/chat/conversation_controller_test.dart test/chat/sync_coordinator_test.dart test/chat/chat_providers_test.dart test/chat/chat_api_client_test.dart test/chat/report_controller_test.dart`
  — 41 passing tests, exit 0, 10 seconds;
  `/private/tmp/craftsky-dm-rereview-focused.log`.
- `scripts/appview-test focused ./internal/api ./internal/chat -p=1 -run '^(TestChatUpstreamConversationDenialReachesConsumer|TestChatConsumerPreservesUpstreamRetryAfter|TestChatSendRejectedReceiptSurvivesHTTPAndDuplicate|TestSchedulerBacksOffEarlyFailuresAndServesThirdOwner|TestPollSelectorFailureUsesClaimedBackoff|TestPollClaimsBoundConcurrentAccountsAcrossWorkerInstances)$' -race -v`
  — all six existing correction regressions pass, exit 0; API 4.017s, chat 4.436s;
  `/private/tmp/craftsky-dm-rereview-go.log`.
- `git diff --check` passes. No new reproduction test was written during review.

Existing final artifacts inspected, not rerun by this review:

- `/private/tmp/craftsky-dm-corrections-check-final`: both ordinary and race JSON
  results contain 49 passing packages and 6,503 passing test/subtest results each,
  zero failures and zero skipped tests. The enclosing gate log reports all
  release gates passed, including generation/static/security, binaries/images,
  migrations, bounded media, packaged console, health and pinned Tap handshake.
- Complete Flutter suite: 3,171 passes and 38 configured skips, 285 seconds.
  No chat test is skipped. The final analyzer artifact is empty and its successful
  execution is recorded. Android debug 17.3s, iOS simulator 52.6s and web 63.1s
  logs each report successful compilation. These are not runtime/live evidence.
- The initial correction gate's race fixture failure is disclosed and followed
  by ten passing race repetitions and the complete fresh gate. The initial
  analyzer exit 0 with three informational lints is also disclosed; its final
  test-only const cleanup has fresh zero-diagnostic analysis and 15 passing
  controller regressions. No production source changed after the full suite/builds.
- The earlier implementation's migration-000073 setup timeout and unchanged
  focused retry remain disclosed separately in `06-validation-evidence.md`.
  None of those passing gates cover IR-006's missing ordering case.

## Risk Review

- Risk level: High for the feature due to private data, OAuth authority,
  non-idempotent remote effects and asynchronous account/device lifecycle work.
- The original five risk/behavior findings are corrected. IR-006 still violates
  the required provider backoff across presentation/lifecycle transitions.
- Preserve the existing account/messaging-generation and stale-content fences
  when fixing scheduling; do not solve the issue by publishing old view results
  or borrowing another activation's state.
- Existing local high-risk implementation approval remains recorded. This turn
  performs review only: no source, tests, dependencies, migrations, generated
  files or implementation/evidence documents were edited, and no commit, push,
  deployment, live message or report was performed.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: the remaining issue concerns retry scheduling and behavior, not styling.
  Existing package/layout/accessibility tests were inspected; this review did not
  run a live UI/device session or identify a separate actionable cosmetic issue.
- Suggested polish notes: none.

## Handoff Back To TDD Builder

- Required fix: IR-006.
- First failing test: UT-006 in `sync_coordinator_test.dart`. Establish a loaded
  snapshot, hold the next heartbeat, background/resume the same owner/generation,
  then release it as a production-mapped 429 with Retry-After 120. Assert no next
  request before the deadline, including queued/manual retry; preserve loaded
  content and do not publish the obsolete view result.
- Extend the same loop to a view switch while pending and a 429 completing while
  still backgrounded before later resume. Verify disposal/account switch discards
  the old activation's guidance, while same-owner/same-generation transitions
  retain it. Retain all current 120-second wire propagation regressions.
- Verify the focused coordinator/provider/chat tests first, then relevant broader
  Flutter regressions and analysis. This finding requires no backend/schema,
  permission, dependency or production change. Additional platform builds should
  follow if the final client changes warrant them.
- Review again after the correction. Do not change the verdict based only on the
  existing passing component/gate results.

## Previous Review (Historical)

The 2026-10-19 review below is retained unchanged. Its original findings are
closed by the disposition table above; the current verdict concerns IR-006.

<details>
<summary>2026-10-19 implementation review</summary>

# Implementation Review: Direct Messaging

## Verdict

Status: Changes required  
Reviewer: Codex, source and evidence review in the current worktree  
Date: 2026-10-19  
Risk level: High

## Summary

The implementation covers the planned architecture and has substantial passing
local verification. Five gaps remain at the boundaries between otherwise tested
components: upstream access-denial translation, scheduler failures before claim
acquisition, HTTP rate-limit propagation, failed-send receipt propagation, and
uncertain-send reconciliation before a fresh operation. These affect Must
behavior and should be corrected before merge or implementation handoff.

The findings below follow concrete control flow in the current source. They are
not claims of observed production incidents or newly failing executable tests.
This review ran existing focused tests and inspected the completed gate artifacts;
it did not add reproduction tests or apply fixes. The next TDD pass should first
demonstrate each finding with the specified public-boundary test.

Your decision to perform MAN-001–MAN-004 yourself is preserved. Live Bluesky report
acceptance remains assumed/unverified, as requested. Those disclosed boundaries
are separate from the required local corrections below.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Risk | [P1] A real upstream conversation 404 becomes consumer HTTP 400 `chat_operation_rejected`. The client therefore treats membership/access loss as an ordinary failure and retains already-loaded private history. The recent access-loss test injects a different, already-normalized error. | FR-014, FR-016, NFR-001; AC-016/019; IT-013; `appview/internal/chat/report_service.go:21`, `appview/internal/api/chat_mutations.go:86`, `app/lib/chat/services/chat_conversation_controller.dart:125`, `app/lib/chat/services/chat_sync_coordinator.dart:203` | Normalize authoritative conversation-access denials at the appropriate server boundary, preserving their typed private cause. Prove the real upstream → registered HTTP handler → repository/controller path hides history and fences delayed completions. Keep transient failures and the explicit readable-history block policy distinct. |
| IR-002 | Important | Behavior / Risk | [P1] Transient failures before a sync claim is acquired receive no scheduling backoff. The scheduler immediately reselects the still-due owner; two such failing owners can occupy both slots repeatedly and starve a later healthy owner. | NFR-004/005, FR-015; AC-017/022; UT-006, IT-020; `appview/internal/chat/sync.go:89`, `appview/internal/chat/sync.go:95`, `appview/internal/chat/scheduler.go:68`, `appview/internal/auth/session_coordinator.go:252` | Cover selector/session-resume/authority-resolution and claim-acquisition failures with bounded per-owner backoff and fair yielding. Handle the no-claim path without clearing or stealing another poller's claim. Test two early-failing owners plus a healthy third owner. |
| IR-003 | Important | Behavior / Tests | [P2] The chat transport preserves upstream `Retry-After` in a typed error, but `writeChatError` emits HTTP 429 without that header. Flutter only derives the delay from the HTTP header, so foreground heartbeat retries use the short local delay instead of the provider's requested delay. | NFR-004/005; AC-017/022; UT-006, IT-019/020; `appview/internal/chat/transport.go:117`, `appview/internal/api/chat_mutations.go:84`, `app/lib/shared/api/providers/error_mapping_interceptor.dart:64` | Propagate a safe, normalized retry delay through the consumer response and test it through the real API/repository/coordinator boundary, including a delay longer than 60 seconds and lifecycle/manual retry attempts. Reuse the existing typed `RetrySeconds()` convention where appropriate. |
| IR-004 | Important | Behavior / Tests | [P2] The ledger's authoritative `failed` send receipt is discarded by `ChatSendHandler`. A definite upstream rejection is returned as an error envelope; Flutter catches it and marks the draft `uncertain`. A duplicate submission of an already-failed operation has no service error and is instead returned as HTTP 503. | FR-005/006; AC-006/007; AT-006/007, IT-006/019; `appview/internal/chat/operations.go:159`, `appview/internal/api/chat_mutations.go:128`, `app/lib/chat/services/chat_conversation_controller.dart:510` | Preserve authoritative failed-operation metadata across HTTP and display failed rather than uncertain. Test an initial rejected send and duplicate submission of its operation through the real handler/repository, retaining text and proving one physical upstream send. Preserve uncertainty for genuinely ambiguous outcomes. |
| IR-005 | Important | Behavior / Tests | [P2] An uncertain send does not initiate the required receipt/history reconciliation. `checkReceipt()` refreshes history only for a confirmed receipt, while `sendAgain()` can allocate and submit a new UUID immediately after the warning. A separate periodic refresh is neither guaranteed nor a prerequisite for this action. | FR-006; AC-007; AT-007, UT-009, IT-006; coding plan §§5.4/7.1; `app/lib/chat/services/chat_conversation_controller.dart:427`, `app/lib/chat/services/chat_conversation_controller.dart:453`, `app/lib/chat/services/chat_conversation_controller.dart:504`, `app/lib/chat/pages/chat_conversation_page.dart:160` | Reconcile the original receipt and refresh authorized history after uncertainty and before permitting a fresh send operation. A confirmed original receipt must resolve without another send; unresolved results must retain text and require the existing explicit duplicate-risk acknowledgment. History resemblance must never establish confirmation. |

### IR-001: Upstream denial does not reach the client denial branch

1. `currentConversation()` returns an upstream `*atclient.APIError` unchanged when
   `getConvo` responds 404.
2. The generic provider-4xx branch in `writeChatError()` returns status 400 and
   `chat_operation_rejected`, rather than `chat_conversation_unavailable`.
3. `_contentFailure()` clears history only for its known denial codes or
   401/403/404/410 classifications. The emitted 400/code matches neither; it sets
   `failed` while retaining the conversation/history. Heartbeat denial handling
   also does not recognize this code.

The existing IT-013 controller test supplies
`ApiBadRequest('chat_conversation_unavailable', statusCode: 404)` directly. It
verifies the final branch correctly but not that the production server reaches it.

### IR-002: Errors outside the claimed-poll callback bypass backoff

`Poll()` selects a session and enters `WithCapability()` before `ClaimChatSync`.
The coordinator can fail in `ResolveCurrent()` before invoking that callback.
`FailChatSync` and its exponential/provider backoff execute only after a claim
and `pollClaim()` failure. The scheduler defers `ErrPollNotDue` and pauses known
authorization errors, but an ordinary pre-claim discovery/network error only
reports a diagnostic. The next loop reads the same unchanged `next_due_at` and
starts that owner again. The current fairness test holds a call after claim
acquisition; it does not exercise this early-failure path.

### IR-003: Retry delay stops at the HTTP boundary

For example, an upstream heartbeat dependency returns 429 with `Retry-After: 120`.
`Transport` returns a `RateLimitError` retaining 120 seconds, but the consumer
response contains no `Retry-After`. The error-mapping interceptor then leaves
`retryAfterSeconds` null and the coordinator can poll again after its local
cadence/backoff, well before 120 seconds. The existing Go worker and Flutter
coordinator tests each preserve/inject the delay within their own component;
neither tests its passage through the consumer HTTP response.

### IR-004: Failed receipts are not a supported send response

The ledger persists `failed/rejected` for its definite rejection cases. The send
handler explicitly serializes confirmed and uncertain receipts, with no failed
case. Its default path drops the receipt and writes the underlying error. If a
duplicate failed operation returns its receipt with `err == nil`, that default
path substitutes `ErrOperationUnavailable` and emits 503. Flutter consequently
marks both paths uncertain until a later, separate successful receipt lookup.
This contradicts the immediate failed-versus-uncertain state contract even though
the ledger still prevents replay of the original operation ID.

### IR-005: A fresh operation can precede reconciliation

Both an uncertain receipt response and a thrown send error leave the local
attempt unresolved without initiating receipt/history reads. “Check status”
fetches history only if the receipt is confirmed; it skips history when the
receipt remains uncertain. “Send again” opens the warning and then directly
creates a new operation. Thus a lost response for an already-confirmed original
operation can lead to another send before the available receipt is checked.
Keep the explicit retry choice, but perform the planned safe reconciliation
first. Never clear a draft merely because an identical message appears in history.

## Requirement And Test Traceability

- Requirements implemented in substantial local coverage: optional consent and
  least-authority routing (BR-002/003, FR-001/016, RULE-004, NFR-001); direct inbox,
  requests, canonical start, history and actions (BR-001, FR-002–004/007–009/018,
  RULE-001/002/005); text/report/declaration handling (FR-005/006/010/011); durable
  sync, private push, retention and cleanup (FR-012–016, RULE-003, NFR-002–005);
  previews, drafts and selected-package/platform support (FR-017/019, NFR-006/008).
  This mapping is not a claim that the findings above satisfy their criteria.
- Tests implemented: the owning suites and AT/UT/IT/REG mappings in the final
  evidence index were reviewed, with particular attention to IT-006/013/015/019/020
  and UT-006/009. The missing tests are integration-boundary/failure-order cases,
  not grounds to duplicate every acceptance umbrella in another suite.
- Unplanned behavior: no unrelated feature, public Lexicon, general RPC proxy,
  broader permission fallback, native desktop target, or production action was
  identified in the reviewed diff. The additional routes remain fixed domain
  operations; endpoint count was explicitly an estimate.
- Justified implementation choices: Riverpod-owned ChangeNotifier controllers,
  generation-scoped content transports, feature-scoped sqlc generation and the
  cold-route recipient query are recorded in the implementation notes.
- Remaining gaps: IR-001–IR-005 are required local corrections. MAN-001–MAN-004,
  AT-024, actual OS delivery/taps, real secure-storage execution and assistive
  technology checks remain owner-run. Actual report acceptance is not established.
  Dedicated operation-key provisioning remains a configuration/release prerequisite.
- Traceability correction needed in the next implementation pass: append the new
  red/green evidence to `05-implementation-plan.md` and update its completion
  conclusion after these corrections. This review does not edit that artifact.

## Test Evidence

- Newly run during review:
  `flutter test --no-pub test/chat/conversation_controller_test.dart test/chat/chat_providers_test.dart test/chat/chat_api_client_test.dart test/chat/report_controller_test.dart`
  passed all 33 tests, exit 0. Log:
  `/private/tmp/craftsky-dm-review-focused.log`.
- Existing final implementation artifacts inspected:
  `/private/tmp/craftsky-dm-appview-check-final` contains ordinary and race JSON
  results for 49 passing packages, 6,498 passing test/subtest results per mode,
  and no failed test results. The enclosing gate log reports all release gates
  passed. Its static/security, binary/image, migrated database, migration-failure,
  media, health and pinned Tap-handshake evidence is recorded separately from live
  messaging behavior.
- The final Flutter log reports 3,166 passing tests and 38 configured skips.
  The final whole-app analyzer artifact is empty, consistent with its recorded
  successful zero-diagnostic run. Android debug APK, iOS simulator and web logs
  each report a successful build. These were inspected, not rerun by this review.
- `git diff --check` passes during review.
- The original isolated-gate migration-000073 setup timeout and its unchanged
  focused retry remain disclosed in `06-validation-evidence.md`; the later full
  isolated gate passed. No failing automated result is being hidden.
- The identified regression tests have not yet been written or executed. Passing
  component tests do not establish the missing complete-path behavior.

## Risk Review

- Risk level: High, due to private content, OAuth authority, non-idempotent writes,
  account/device lifecycle fencing and background delivery.
- Positive controls reviewed: fixed chat/moderation audiences and implemented
  scopes; server-held OAuth/DPoP; exact-child consent binding; durable keyed
  operation fence; no transcript/report-prose storage; minimal retained ledgers;
  atomic progress/outbox updates; generation/lease dispatch checks; owner-only
  draft purge; guarded notification navigation and private diagnostic contexts.
- Required risk corrections: access-loss normalization and pre-claim scheduling
  are P1; retry-delay propagation and accurate send recovery/state are P2.
- Approval notes: the existing owner approval covers the planned local high-risk
  implementation. This review changes only this review artifact. No source, test,
  dependency, migration, generated file or earlier workflow document was edited;
  no commit, push, deployment, live message or report was performed.
- Live/manual gaps remain explicit under the owner's instruction and are not
  relabeled as local automation passes or used to invent additional live actions.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: the required findings concern behavior and API/state boundaries, not
  styling. The existing package/widget and narrow/large-text coverage is useful;
  no separate actionable cosmetic defect was identified in this source review.
- Suggested polish notes: none. Owner-run VoiceOver/TalkBack/web interaction
  remains validation work, and is not replaced by this recommendation.

## Handoff Back To TDD Builder

- Required fixes: address IR-001–IR-005 in separate red-green-refactor loops.
- Suggested first failing test: IT-013 — load private conversation content, return
  an actual upstream `getConvo` 404 through the production service/HTTP error
  mapper, and feed the resulting response through `ApiChatRepository` and the
  loaded controller/heartbeat path. Assert content hides, drafts remain local,
  delayed completions cannot repaint it, and base-account access remains valid.
- Next tests:
  1. IT-020 — two due owners fail authority resolution before a claim; a third
     healthy owner progresses, failed owners do not hot-loop, retry deadlines
     hold, and valid saved cursors/other poll claims remain intact.
  2. UT-006 / IT-019 — actual upstream 429 plus 120-second retry delay reaches
     Flutter through the HTTP contract; foreground/manual/resume attempts respect
     that deadline without clearing loaded content.
  3. IT-006 / AT-006 — definitive send rejection and same-operation duplicate
     return/render failed, retain text and make exactly one upstream send attempt.
  4. UT-009 / IT-006 / AT-007 — a lost send response followed by a confirmed original
     receipt resolves without any new send; an uncertain receipt still refreshes
     history and preserves uncertainty/text before explicit warned retry with a
     fresh UUID. Identical text does not confirm delivery.
- Verification to rerun: each owning focused suite first, then affected API,
  scheduler, auth and Flutter chat/notification tests. After corrections, rerun
  the broader backend/Flutter gates appropriate to the changed source and update
  evidence; repeat supported-platform builds when client changes warrant it.
  Avoid competing full database/native build jobs. Leave manual/live checks with
  the owner and keep report acceptance assumed/unverified.

</details>
