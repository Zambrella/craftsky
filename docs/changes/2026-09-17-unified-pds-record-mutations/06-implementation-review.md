# Implementation Review: Unified Tap-Authoritative PDS Record Mutations

## Verdict
Status: Approved with notes (follow-up corrections and legacy removal, 2026-09-28)
Reviewer: OpenCode
Date: 2026-09-28
Risk level: High

## Summary
The September 25 review closed its seven original findings. A follow-up audit found five more correctness gaps, which were corrected and verified. A subsequent user-requested cleanup removed the four winner-era physical set tables and their runtime callers in migration 75; its rollback recreates empty compatibility tables for historical migrations.

## Follow-up Findings (2026-09-28)

| ID | Severity | Area | Finding | References | Required action |
|---|---|---|---|---|---|
| IR-008 | Critical | Flutter / lost response | Only an HTTP `202` enters the ambiguous same-key retry path. A receive timeout or connection loss *after AppView/PDS accepts a write* becomes `ApiNetworkError`; every migrated provider's general catch marks the operation failed. Repeating create allocates a new key and may publish a second post/event. An HTTP 5xx or malformed success after an uncertain dispatch has the same problem. | BR-002, FR-030, FR-032, AC-003, AC-021, AC-036; `app/lib/shared/api/pds_mutation_contract.dart:31-53`, `app/lib/shared/api/providers/error_mapping_interceptor.dart:23-49`, `app/lib/feed/providers/create_post_provider.dart:119-164`, `app/lib/feed/providers/toggle_like_post_provider.dart:67-103` | Centralize mutation response/transport classification. Treat post-dispatch transport failures and unknown server outcomes as unresolved, retaining the frozen key/input and allowing bounded automatic plus explicit same-key retry. Only proven failures may release the operation. Add API-client and provider tests for a dropped response after a successful server write. |
| IR-009 | Critical | Tap projection | `TransactionalDispatcher.Project` returns `PermanentInvalid` before calling the feature projector for a newer invalid create/update. A previously valid URI therefore keeps its old `pds_set_sources`/aggregate (or post/profile serving row) even though `tap_source_records` now holds the invalid current version. The existing valid→invalid→valid integration test calls `projectSetSourceTx` directly, bypassing this production short circuit. | FR-004, FR-006, FR-008, AC-009, AC-012; `appview/internal/index/transactional_dispatcher.go:74-87`, `appview/internal/ingestion/store.go:648-655`, `appview/internal/index/set_source_projector.go:21-26`, `appview/internal/index/set_source_projection_integration_test.go:17-55,128-145` | On invalid latest versions, remove the previous serving projection and normalized fact under the same projection transaction while retaining invalid raw source evidence and quarantine status. Test valid→invalid→valid via `ProcessProjection` with the production dispatcher for both a set record and a non-set serving record. |
| IR-010 | Important | Set-command recovery | After a lost create response, `reconcilePresent` sees the selected URI is absent and dispatches again even when the command is already `ambiguous`. If the first create succeeded and another client subsequently deleted the record, the retry recreates an action that the external client removed. For ambiguous remove with an intervening new match, the path instead stays ambiguous indefinitely. Current tests cover lost responses only while the first effect still exists. | FR-017, FR-019, FR-024, AC-021, AC-029; `appview/internal/pdscommands/set_command_service.go:174-220,235-275`, `appview/internal/pdscommands/state.go:11-20`, `appview/internal/pdscommands/set_command_service_integration_test.go:79-102` | Reconcile the retained attempt against repository/record history or preserve ambiguity when a later change prevents proof; never redispatch an uncertain attempt solely because current logical state is absent. Test create→response loss→external delete→retry and remove→response loss→external create→retry. Known `InvalidSwap` is a separate safe-to-replan case. |
| IR-011 | Important | Addressed/append recovery | An uncertain addressed delete is marked *rejected* if the record has a new CID on retry; an uncertain append is marked *rejected* if the selected URI exists with different content. Either state may result from a successful first write followed by an external update/recreate, so a permanent rejected command contradicts actual PDS acceptance. | FR-016, FR-017, FR-019, AC-021, AC-024; `appview/internal/pdscommands/addressed_command_service.go:254-270,342-375`, `appview/internal/pdscommands/append_command_service.go:238-256` | Separate pre-dispatch conflict from post-dispatch uncertainty. If a dispatch is possible but later state cannot prove its outcome, keep it ambiguous or use commit/attempt evidence rather than terminal rejection. Add lost-response + external overwrite/recreate cases. |
| IR-012 | Important | Validation consistency | Ingestion accepts a record without `$type` as valid; set command matchers reject it because their decoders require `LexiconTypeID`. Thus an external follow/like/repost can appear in the Tap aggregate while a Craftsky create writes a duplicate and an explicit remove skips that valid source. | FR-003, FR-005, FR-022, FR-041, AC-001, AC-027, AC-049; `appview/internal/sourcevalidation/registry.go:51-73,219-230`, `appview/internal/api/post_like_commands.go:183-186,212-223`, `appview/internal/api/post_repost_commands.go:184-186,213-223`, `appview/internal/api/follow.go:205-207,228-233` | Use one source-validation and semantic matching contract for ingestion and authoritative set reads. Either reject missing `$type` in both places if that is the intended record policy, or accept it in both. Cover an externally written record accepted by ingestion through create no-op and remove-all. |

## Simplification Opportunities

1. **Keep the three authorities, reduce the adapters.** PDS existence, Tap-derived read state, and a scoped retry journal solve different problems and are justified. The repeated retry/backoff loops across Flutter feature providers and the near-identical prepare/dispatch/result code in the four Go command services are not. Extract one small feature-neutral mutation runner and one guarded dispatch/outcome helper, leaving typed feature-specific validation, plans, and response shaping at the edges. This would also make IR-008 and IR-010/011 harder to reintroduce.
2. **Remove dead physical set projections once their consumers are migrated.** The branch still carries legacy `craftsky_likes`, `craftsky_reposts`, `atproto_follows`, and `atproto_blocks` writer/test/purge machinery alongside `pds_set_sources` and `pds_set_aggregates`. Inventory actual production readers first, then retire unused paths and their schema in a deliberately fresh development migration, updating old isolated-table tests. Do not remove the durable source facts or logical aggregate.
3. **Wire or remove the signed-CAR fallback contract.** `NewAuthoritativeReaderWithFallback` exists and has an isolated test, but production `SetCommandService.readCollection` always calls `NewAuthoritativeReader`, so a changing head cannot use the documented fallback. Either inject the existing verified snapshot fetcher at the command boundary with source/credential checks, or document that busy repositories fail closed and remove the unused fallback abstraction. See `appview/internal/pdscommands/authoritative_reader.go:70-84,138-156` and `set_command_service.go:223-233`.

## Follow-up Test Evidence

- Reviewed `main...HEAD` (commits `bdeba8de` and `71570436` plus merge) with a clean initial worktree.
- Focused existing tests passed with required real PostgreSQL: `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16063/craftsky_dev?sslmode=disable' go test ./internal/index ./internal/pdscommands -run 'TestProjectSetSourceReplacesValidFactWithLatestValidationState|TestSetCommandServiceReconcilesLostCreateAndRemovesEveryMatchingRecord' -count=1`.
- These tests cover a direct projector call and a lost response with no intervening external mutation; they do not exercise the failure scenarios above. The September 25 full-suite evidence predates this audit. No code fix or new regression test was made in this review.

## Follow-up Handoff (completed)

The correction sequence used failing Flutter dropped-response and production-dispatcher invalid-replacement tests, then added lost-response/external-write regressions, unified set validation, and ran the release gates. UI polish remains optional.

## Correction Results (2026-09-28)

| Finding | Disposition | Evidence |
|---|---|---|
| IR-008 | Resolved | Keyed API clients now classify dropped responses, unknown server outcomes, and unparseable success responses as ambiguous. The shared Flutter runner retains the same token/key across bounded retries. Explicit `video_blob_missing` recovery is preserved. Post create, shared-contract, and feature-provider regressions pass. |
| IR-009 | Resolved for non-membership public facts | Invalid latest follow and Bluesky-profile versions now cause transactional serving cleanup in the production dispatcher; source evidence remains invalid and retained. Craftsky membership profile remains governed by FR-029: malformed profile updates cannot trigger departure or cascade private data. |
| IR-010 | Safe outcome | A set create never blindly redispatches an uncertain attempt on observed absence or an unrelated match. The original command remains ambiguous when external changes make its outcome unprovable. Exact selected-record recovery and definite `InvalidSwap` replanning still work; real-PostgreSQL interruption tests cover both. |
| IR-011 | Resolved | Uncertain append/addressed commands retain ambiguity after external overwrites/recreation. Pre-dispatch conflicts remain definitive. The shared journal marks open attempts ambiguous once and only replans definite `InvalidSwap` attempts within the three-attempt budget. |
| IR-012 | Resolved | Authoritative set matchers use the same current source validation policy as ingestion across follows, blocks, likes, and reposts. A source-valid `$type`-omitting follow is recognized as an existing match; invalid-key fixtures were corrected to reflect actual TID policy. |

The branch also re-enabled `TAP_NO_REPLAY` in Compose; the durable-cursor acceptance test caught it. Removing that setting restored the supported replay behavior. Final `just test`, `just appview-check`, `just app-test` (2,427 tests), `just app-analyze`, and `git diff --check` passed. No production resources were changed.

**Legacy removal (subsequent request):** Migration 75 drops `craftsky_likes`, `craftsky_reposts`, `atproto_follows`, and `atproto_blocks`. Their obsolete indexer/reader and purge paths were removed. Historical migrations and their tests retain the names so old schema versions and the down chain remain verifiable; current production Go has no callers. A conformance test enforces this. The migration discards physical rows and its down migration cannot recover them; no production database or service was changed here. Full backend and AppView release gates passed after the removal. Commit `7de1cae7` precedes this separately committed cleanup, as requested.

## Prior Review: Finding Dispositions (2026-09-25)
| ID | Original Severity | Disposition | Correction Evidence |
|---|---|---|---|
| IR-001 | Important | Resolved | Terminal dispatch and command completion now use the atomic `CompleteDispatchAndCommand` store transition. Invalid-swap recovery also handles a completed dispatch safely. Real-PostgreSQL coverage verifies atomic accepted completion and completed invalid-swap recovery in `appview/internal/pdscommands/store_integration_test.go`. |
| IR-002 | Critical | Resolved | `PrepareOrReplay` now enters the owner lifecycle active-effects scope before selecting identity or inserting command state. A terminal owner is rejected without creating a `pds_commands` row, covered by `TestPrepareOrReplayRejectsTerminalOwnerWithoutCreatingState`. All command services use this store boundary. |
| IR-003 | Important | Resolved | Production constructs `CompactionProcessor`, exposes it through application dependencies, and runs it as a bounded, cancellable batch worker. Poll interval and batch size are bounded configuration values. Store integration coverage verifies the 24-hour replay-to-tombstone transition and rejects key reuse after compaction. |
| IR-004 | Important | Resolved | Flutter providers preserve exhausted ambiguity as an actionable unresolved operation and expose explicit retry using the frozen endpoint, body, and operation key. Delete, like/repost, create, and Instagram paths have focused exhaustion/retry coverage; ambiguous delete no longer publishes success. |
| IR-005 | Important | Resolved | Quote pages now compose shared post-record and interaction overlays for initial, continuation, and refresh reads. Mutation callbacks invalidate the quote provider, and direct mutation-result `replace`/`remove` cache patching was removed. IT-012 coverage verifies stale accepted interaction retention through expiry and accepted-delete masking. |
| IR-006 | Important | Resolved | `PostApiClient.createPost` requires a caller-owned `operationKey`; its generated fallback was removed. Client tests verify the exact key is sent as `Idempotency-Key`, including unchanged retries. |
| IR-007 | Important | Resolved | UT-019 now scans all production Dart sources except four documented private-state exclusions and rejects obsolete feature-cache mutation helpers. REG-009 injects a forbidden helper into an in-memory source and proves the guard detects it. Generic post-cache mutation APIs and their obsolete tests were removed. |

## Prior Review: Requirement And Test Traceability (2026-09-25)
- Requirements implemented: All planned Must requirements are represented across migration 73, authoritative source validation and projection, durable command identity and recovery, HTTP contracts, Flutter operation control and overlays, vertical slices, retention, and legacy removal.
- Acceptance coverage: The planned unit, integration, acceptance, regression, real-PostgreSQL, protocol, route, provider, matrix, and architecture checks are present and pass.
- Corrective coverage: IR-001 through IR-007 are each tied to focused regression tests described above.
- Unplanned behavior: None identified outside the approved scope.
- Remaining gaps identified at the time: None. See IR-008 through IR-012 for gaps found by the follow-up audit.

## Prior Review: Test Evidence (2026-09-25)
- `CGO_ENABLED=0 TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16063/craftsky_dev?sslmode=disable' just test`: passed; required PostgreSQL tests ran without skips.
- `CGO_ENABLED=0 just appview-check`: passed, including tests, production builds, vulnerability scans, health checks, and the pinned Tap handshake.
- `just app-test`: passed, 2,412 tests.
- Focused Flutter correction suite: passed, 141 tests covering API-client keys, quote composition, interaction reconciliation, legacy cache-helper removal, and architecture conformance.
- `just app-analyze`: passed with `No issues found`.
- `git diff --check`: passed.
- Native-cgo execution remains unavailable because the host is missing `/Library/Developer/CommandLineTools/SDKs/MacOSX27.sdk`; the approved `CGO_ENABLED=0` verification path was used.

## Prior Review: Risk Assessment (2026-09-25)
- Risk level: Medium
- Risk notes: This remains a broad cross-stack mutation-path change with durable command and account-lifecycle implications. The focused boundary tests and full release gates reduce the identified correctness risks, but rollout should retain normal command-state, compaction, Tap projection, and mutation-error monitoring.
- Residual test risk: One earlier independent run observed an intermittent `TestPushLifecycleTransitionWaitsThroughSendFinalization` failure; its isolated rerun and the final full gates passed. Treat it as an existing flake signal rather than a blocker for this implementation.
- Approval notes at the time: The original Critical and Important findings were resolved. This assessment is superseded by the follow-up verdict above.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The corrected ambiguity state and explicit same-key retry are functional and tested. A later polish pass may standardize wording and presentation across mutation surfaces without changing behavior.

## Prior Review: Handoff (2026-09-25)
- Review result at the time: No open Must-level findings. Superseded by IR-008 through IR-012 above.
- Merge preparation: Review the complete worktree diff and stage only the intended unified-mutation files; the worktree contains a large set of related modifications and generated outputs.
- Production follow-up: Use the existing release process and monitor command compaction, ambiguous command rates, Tap convergence, and mutation response errors after deployment.
