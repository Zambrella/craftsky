# Implementation Review: Instagram DM Verification And Automatic Following

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-27
Risk level: Medium

## Summary

The implementation is ready for merge or handoff, subject to the documented
external production gates. The second correction pass closes the remaining
Must-level current-membership gap from `IR-022`: reconciliation now checks the
job owner before candidate loading, so a departed member's import or verified
mapping is inactivated even when the job resolves to zero candidate rows.
Membership lookup and inactivation failures continue through the bounded
retry path, and no PDS follow operation or notification is created.

The job-owner contract is consistent across production producers. Link-scoped
jobs store the linked target member as `owner_did`, import-scoped jobs store the
importer, and pair-scoped safety-restoration jobs store the importer with a
separate target. Existing candidate-level checks remain in place for the other
party. Real-Postgres regressions cover the zero-candidate owner path, the
existing-candidate target path, and both membership failure modes.

No new blocking behavior, privacy, authorization, migration, or API-contract
issue was identified. The remaining findings are non-blocking defense-in-depth,
test-symmetry, and internal-terminology improvements.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-023 | Suggestion | Limits / Defense In Depth | The production loop and real store enforce the automatic-follow batch maximum of 100, but `AutomaticFollowWorker.ProcessBatch` itself rejects only values below one. Current production wiring remains bounded; a future direct caller or test store could bypass the component-level maximum. | Requirements §12.4; `FR-017`, `NFR-005`, `TD-012`; `appview/internal/instagram/automatic_follow_worker.go`; `appview/internal/instagram/automatic_follow_store.go`; `appview/cmd/appview/main.go` | Opportunistically add the same `AutomaticFollowBatchMax` check at the worker entry point and a focused exact-100/101 test. No change is required before this implementation is accepted. |
| IR-024 | Suggestion | Tests / Multi-account | Owner-scoped notification regression tests prove inactive-owner rejection and switch-during-await fencing for the Unfollow branch. Follow and Unfollow share the same owner-lease guard, and shared notification tests exercise both actions, so no behavioral defect is present; a symmetric owner-scoped Follow case would make `IT-017` traceability more explicit. | `FR-026`, `NFR-008`, `IT-017`; `app/test/notifications/instagram_match_notification_test.dart`; `app/test/notifications/notifications_page_test.dart` | Add one retained-owner or switch-during-await Follow case as an optional regression improvement. |
| IR-021 | Suggestion | Code Quality / Terminology | Some internal policy/request symbols retain the superseded suggestion terminology, including `InstagramSuggestionEligibilityPolicy` and `SuggestionEligibilityRequest`. The coding plan permits truthful legacy internal/storage naming, and no member-facing suggestion surface remains. | `FR-016`; `04-coding-plan.md` §4.2; `appview/internal/instagram/eligibility_policy.go`; `appview/internal/instagram/automatic_follow_worker.go`; `appview/internal/instagram/reconciliation.go` | Rename non-storage internal symbols opportunistically without rewriting historical TDD evidence. |

## Requirement And Test Traceability

- Requirements implemented: DM challenge creation/redemption/confirmation;
  current-member and ownership boundaries; discovery/revocation controls;
  following-only manual/JSON/streaming-ZIP imports; verification-lifetime
  private retention; deterministic initial/future automatic follows;
  exact-owner background OAuth session selection; manual-unfollow suppression;
  actorful type-only notifications; identity-free push routing; removed
  suggestion APIs/UI; safety-restoration triggers; and account-scoped Flutter
  controls.
- Tests implemented: the planned AppView, Flutter, shared-wire, lifecycle,
  privacy, concurrency, crash-recovery, owner-session, restoration, and
  multi-account suites are present. The `IT-020` correction adds a
  real-Postgres zero-candidate departed-owner test and explicit retry controls
  for membership lookup/inactivation failure while retaining the departed
  candidate-target control.
- Unplanned behavior: none identified.
- Remaining gaps: no implementation gap. Live Meta behavior, additional
  approved export variants, trusted-edge/multi-replica enforcement,
  physical-device push/file-picker/memory/accessibility behavior, production
  safety-adapter validation, and final security/privacy/operator review remain
  documented external release gates.

## Test Evidence

- Commands reviewed from the second correction pass:
  - focused real-Postgres departed-owner, departed-target, membership-lookup,
    and inactivation-failure reconciliation tests;
  - real-Postgres `go test ./internal/instagram -count=1`;
  - real-Postgres `go test ./... -count=1`;
  - real-Postgres `go test -race ./internal/instagram -count=1`;
  - `go vet ./...`, focused `gofmt -l`, and `git diff --check`;
  - 17 focused Flutter notification tests;
  - all 1,096 Flutter tests;
  - `flutter analyze`.
- Passing evidence: every command above passed; Flutter analysis reported no
  issues.
- Failing or skipped tests: none remain. Automated tests remain synthetic and
  make no Meta calls.

## Risk Review

- Risk level: Medium.
- Risk notes:
  - Reconciliation now enforces the job-owner membership boundary before a
    query can return zero candidates, and still checks candidate importer and
    target membership before policy or persistence work.
  - Confirmed departure invokes the shared transactional private-data
    inactivator. The inactivator terminalizes the processing job, so the
    worker's later lease-qualified terminal update safely becomes a no-op.
  - Membership lookup/inactivation errors return through `ProcessBatch`,
    release the lease, increment the bounded attempt count, and schedule the
    expected retry.
  - Exact-owner OAuth selection, final eligibility, deterministic writes,
    transactional notification completion, and manual-unfollow suppression
    remain covered and unchanged by this correction.
  - Live Meta and physical-device behavior remain external and the integration
    stays disabled by default until those gates pass.
- Approval notes: no Must-level correction remains. The feature is acceptable
  for merge or handoff without addressing `IR-021`, `IR-023`, or `IR-024`.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: The reviewed UI already uses verified terminology, semantic discovery
  styling, default Instagram Export selection, actorful notification rows,
  account-scoped follow controls, and a bottom destructive revocation action.
  This correction changed only server lifecycle behavior.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None required. If optional follow-up work is
  selected, start with the exact-100/101 worker-level batch-limit test from
  `IR-023`, or the owner-scoped Follow case from `IR-024`.
- Verification to rerun: no additional verification is required for this
  review. Before a future merge after further edits, rerun the affected focused
  tests, real-Postgres AppView suite, Flutter suite/analysis, formatting, and
  `git diff --check`.
