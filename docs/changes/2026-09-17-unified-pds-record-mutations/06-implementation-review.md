# Implementation Review: Unified Tap-Authoritative PDS Record Mutations

## Verdict
Status: Ready for handoff
Reviewer: OpenCode
Date: 2026-09-25
Risk level: Medium

## Summary
The implementation now satisfies the reviewed Must-level source, aggregate, command, API, Flutter reconciliation, and legacy-removal requirements. The seven findings from the initial review were corrected with focused regression coverage. Full backend, AppView, Flutter, static-analysis, and diff-integrity gates pass. No open correctness or conformance finding blocks handoff.

## Finding Dispositions
| ID | Original Severity | Disposition | Correction Evidence |
|---|---|---|---|
| IR-001 | Important | Resolved | Terminal dispatch and command completion now use the atomic `CompleteDispatchAndCommand` store transition. Invalid-swap recovery also handles a completed dispatch safely. Real-PostgreSQL coverage verifies atomic accepted completion and completed invalid-swap recovery in `appview/internal/pdscommands/store_integration_test.go`. |
| IR-002 | Critical | Resolved | `PrepareOrReplay` now enters the owner lifecycle active-effects scope before selecting identity or inserting command state. A terminal owner is rejected without creating a `pds_commands` row, covered by `TestPrepareOrReplayRejectsTerminalOwnerWithoutCreatingState`. All command services use this store boundary. |
| IR-003 | Important | Resolved | Production constructs `CompactionProcessor`, exposes it through application dependencies, and runs it as a bounded, cancellable batch worker. Poll interval and batch size are bounded configuration values. Store integration coverage verifies the 24-hour replay-to-tombstone transition and rejects key reuse after compaction. |
| IR-004 | Important | Resolved | Flutter providers preserve exhausted ambiguity as an actionable unresolved operation and expose explicit retry using the frozen endpoint, body, and operation key. Delete, like/repost, create, and Instagram paths have focused exhaustion/retry coverage; ambiguous delete no longer publishes success. |
| IR-005 | Important | Resolved | Quote pages now compose shared post-record and interaction overlays for initial, continuation, and refresh reads. Mutation callbacks invalidate the quote provider, and direct mutation-result `replace`/`remove` cache patching was removed. IT-012 coverage verifies stale accepted interaction retention through expiry and accepted-delete masking. |
| IR-006 | Important | Resolved | `PostApiClient.createPost` requires a caller-owned `operationKey`; its generated fallback was removed. Client tests verify the exact key is sent as `Idempotency-Key`, including unchanged retries. |
| IR-007 | Important | Resolved | UT-019 now scans all production Dart sources except four documented private-state exclusions and rejects obsolete feature-cache mutation helpers. REG-009 injects a forbidden helper into an in-memory source and proves the guard detects it. Generic post-cache mutation APIs and their obsolete tests were removed. |

## Requirement And Test Traceability
- Requirements implemented: All planned Must requirements are represented across migration 73, authoritative source validation and projection, durable command identity and recovery, HTTP contracts, Flutter operation control and overlays, vertical slices, retention, and legacy removal.
- Acceptance coverage: The planned unit, integration, acceptance, regression, real-PostgreSQL, protocol, route, provider, matrix, and architecture checks are present and pass.
- Corrective coverage: IR-001 through IR-007 are each tied to focused regression tests described above.
- Unplanned behavior: None identified outside the approved scope.
- Remaining gaps: No blocking requirement or acceptance-test gap remains.

## Test Evidence
- `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16063/craftsky_dev?sslmode=disable' just test`: passed; required PostgreSQL tests ran without skips.
- `CGO_ENABLED=0 just appview-check`: passed, including tests, production builds, vulnerability scans, health checks, and the pinned Tap handshake.
- `just app-test`: passed, 2,412 tests.
- Focused Flutter correction suite: passed, 141 tests covering API-client keys, quote composition, interaction reconciliation, legacy cache-helper removal, and architecture conformance.
- `just app-analyze`: passed with `No issues found`.
- `git diff --check`: passed.
- Native-cgo execution remains unavailable because the host is missing `/Library/Developer/CommandLineTools/SDKs/MacOSX27.sdk`; the approved `CGO_ENABLED=0` verification path was used.

## Risk Review
- Risk level: Medium
- Risk notes: This remains a broad cross-stack mutation-path change with durable command and account-lifecycle implications. The focused boundary tests and full release gates reduce the identified correctness risks, but rollout should retain normal command-state, compaction, Tap projection, and mutation-error monitoring.
- Residual test risk: One earlier independent run observed an intermittent `TestPushLifecycleTransitionWaitsThroughSendFinalization` failure; its isolated rerun and the final full gates passed. Treat it as an existing flake signal rather than a blocker for this implementation.
- Approval notes: The original Critical and Important findings are resolved. The implementation is ready for maintainer handoff and normal merge review.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The corrected ambiguity state and explicit same-key retry are functional and tested. A later polish pass may standardize wording and presentation across mutation surfaces without changing behavior.

## Handoff
- Review result: No open Must-level findings.
- Merge preparation: Review the complete worktree diff and stage only the intended unified-mutation files; the worktree contains a large set of related modifications and generated outputs.
- Production follow-up: Use the existing release process and monitor command compaction, ambiguous command rates, Tap convergence, and mutation response errors after deployment.
