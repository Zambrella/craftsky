# Implementation Review: Instagram DM Verification And Automatic Following

> **2026-08-20 status:** The 2026-07-27 approval and the later Changes-required
> reopening describe the historical automatic-follow contract. Section
> "AppView Audit Strict-Branch Final Re-review" is the current verdict.

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
| IR-021 | Suggestion | Code Quality / Terminology | The current database schema and storage SQL now use automatic-follow terminology, but some internal policy/request symbols retain superseded suggestion terminology, including `InstagramSuggestionEligibilityPolicy` and `SuggestionEligibilityRequest`. No member-facing suggestion surface remains. | `FR-016`; `04-coding-plan.md` §4.2; `appview/internal/instagram/eligibility_policy.go`; `appview/internal/instagram/automatic_follow_worker.go`; `appview/internal/instagram/reconciliation.go` | Rename the remaining non-storage internal symbols opportunistically without rewriting historical TDD evidence. |

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

### Post-review storage-name follow-up

Migration `000031` subsequently renamed the legacy private suggestion tables,
reference columns, constraints, and indexes to automatic-follow terminology.
Forward/rollback migration tests and the affected Instagram, notification,
push, and API package suites pass against real PostgreSQL. This terminology
change does not alter the approved behavior or external release gates.

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

## AppView Audit Review Reopening (2026-08-14)

Status: Changes required

Risk level: High

AV-007 shows that the approved background writer cannot guarantee the claimed
post-departure boundary across an AppView crash and an independently committing
PDS. The product owner approved the strict correction: private suggestions and
explicit current-member Follow, with no background PDS-write capability.

The earlier implementation remains useful evidence for verification, import
privacy, exact matching, membership inactivation, ZIP parsing, and account
fencing. It is no longer acceptable evidence for match-to-follow authority,
background session selection, automatic `instagramMatch` notifications, or
the absence of a suggestion review surface.

Required implementation evidence is UT-021–UT-023, IT-026–IT-029, and REG-015
from `02-acceptance-tests.md` Section 12. Re-review must also confirm:

- the dependency graph cannot construct the retired writer;
- explicit acceptance uses the common owner/session effect boundary;
- departure and terminal races make no new background external call;
- Flutter restores explicit consent without cross-account effects; and
- cleanup never deletes `app.bsky.graph.follow`.

This review returns to Approved only after the correction execution log in
`05-implementation-plan.md` is complete and the coordinated PostgreSQL/race and
full Go/Flutter gates have actually run.

## AppView Audit Strict-Branch Final Re-review (2026-08-20)

Status: Approved with notes

Risk level: Medium until the external Meta/device/deployment gates run

The strict AV-007 correction is implemented and supersedes the 2026-08-14
`Changes required` snapshot. Matching and reconciliation now terminate at one
caller-private, participant-generation-bound suggestion and have no OAuth/PDS
session selector, follow writer, or public-write capability. Only an explicit
current-member Follow enters the common guarded owner-effect executor. Dismiss,
evidence loss, departure, terminalization, and stale-generation paths make no
new external call, and no cleanup path gains authority to delete
`app.bsky.graph.follow`.

Required-PostgreSQL and race tests cover initial/future/duplicate/reordered
matching, suggestion ownership and lifecycle invalidation, explicit acceptance,
ambiguous-response replay, and dependency capability absence. API,
notification, push, relationship, app-wiring, migration, and focused Flutter
Instagram suites pass. The complete repository release gate, pinned vet/static
analysis, `dart analyze`, and all 1,489 Flutter tests also pass in the combined
AppView audit worktree.

No blocking implementation finding remains for the strict branch. Live Meta,
physical-device, protected-edge, and accessibility checks in the Known External
Gates remain release evidence rather than code-completion work.

## Instagram Match Notification Restoration Review (2026-08-21)

Status: Changes required

Reviewer: Codex

Risk level: High until the required PostgreSQL and Flutter interaction evidence
is complete

### Summary

The implementation restores `instagramMatch` without restoring automatic
following. A newly inserted private suggestion calls notification activation
inside the existing participant-lifecycle transaction, the event is actor-backed
and source-less, suggestion-key uniqueness makes event creation replay-safe, and
the matcher still has no OAuth, PDS-client, or public-write dependency. The
category, fixed-scope preference, regular push payload, API hydration, Flutter
model/settings/row, and non-automatic wording are all present.

Two Must-level acceptance paths are not yet proved by the test suite. The
database tests assert one event but neither schedule an active-subscription
delivery nor force an activation failure to demonstrate that the suggestion and
notification roll back together. They were also skipped in the recorded full Go
run because no PostgreSQL test URL was configured. The Flutter test renders the
Instagram row, while its navigation and Follow/Following interaction coverage
uses the ordinary follow-notification fixture rather than `instagramMatch` and
does not prove the captured-account path required by `IT-031`.

No correctness defect was identified in the inspected production path, but the
missing Must-level evidence requires a `Changes required` verdict under this
workflow.

### Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-025 | Important | Tests / Transactional Delivery | `IT-030` counts one source-less event after duplicate initial/future matching, but no restored test seeds an active push installation/subscription and asserts exactly one delivery, disables push and asserts the in-app event remains without a delivery, or injects an activation failure and proves that both suggestion and notification state roll back. The relevant database-backed cases were skipped in the recorded full Go run, so `AC-064` and the delivery half of `AC-065` lack executing evidence. | `FR-038`, `FR-039`, `NFR-012`, `AC-064`, `AC-065`, `IT-030`; `appview/internal/instagram/private_suggestion_store_test.go`; `appview/internal/instagram/future_match_test.go`; `appview/internal/notifications/instagram_match.go`; `05-implementation-plan.md` restoration verification evidence | Add focused real-PostgreSQL tests for push-enabled active subscriptions, push-disabled in-app-only behavior, replayed delivery idempotency, and a forced notification-activation failure that leaves neither suggestion nor event committed. Run them with the PostgreSQL test URL configured. |
| IR-026 | Important | Tests / Flutter Interaction | `IT-031` renders and decodes `InstagramMatchNotification`, but the row-tap profile assertion and Follow/Following mutation tests instantiate `FollowNotification`. Consequently the required Instagram-row profile destination, explicit follow-state transition, and captured-account isolation are not directly regressed. | `FR-040`, `AC-067`, `IT-031`; `app/test/notifications/notifications_page_test.dart`; `app/lib/notifications/widgets/notification_row.dart` | Add an Instagram-match widget test that taps the row and verifies the matched profile, exercises Follow and Following through the actor-backed row, and proves a switch during an awaited action cannot mutate or navigate under the newly active account. |
| IR-027 | Suggestion | Code Quality / Capability Boundaries | The coding plan called for a narrow transactional match-notifier capability, but `PrivateSuggestionStore` holds the concrete, full `*notifications.Service`. Current code invokes only `ActivateInstagramMatch` and therefore does not violate the no-PDS/OAuth boundary, but a small interface owned by the Instagram package would make the capability explicit and enable deterministic activation-failure injection for `IR-025`. | `04-coding-plan.md` §14 step 2; `appview/internal/instagram/private_suggestion_store.go`; `appview/internal/app/deps_instagram_storage.go` | Prefer an interface exposing only transactional Instagram-match activation, implemented by the notification service. This is non-blocking if equivalent rollback fault injection is achieved another way. |

### Requirement And Test Traceability

- `FR-038` / `NFR-012`: implemented structurally; event creation is inside the
  lifecycle-fenced suggestion transaction and keyed by suggestion ID. Replay
  event counts are covered, but rollback and delivery behavior remain incomplete
  per `IR-025`.
- `FR-039`: implemented. Category registration, immutable `everyone` scope,
  configurable push, API preferences, and identity-free regular-push data/copy
  have focused automated coverage. The push-disabled in-app-only interaction
  still needs the database assertion in `IR-025`.
- `FR-040`: implemented in the Flutter model and row. Decode, copy, icon,
  settings, and push-open routing are covered; Instagram-specific profile and
  account-fenced Follow/Following interaction evidence remains incomplete per
  `IR-026`.
- Unplanned behavior: none identified. Automatic follow remains absent and no
  matcher/reconciler PDS-write capability was reintroduced.

### Test Evidence

- Reviewed passing evidence recorded by the implementation:
  - `GOCACHE=/tmp/craftsky-go-cache go test ./... -count=1` from `appview/`;
  - all 97 Flutter notification tests;
  - the 45-test focused cross-feature/wire-contract run;
  - `flutter analyze`;
  - `GOCACHE=/tmp/craftsky-go-cache go vet ./...`;
  - `git diff --check`.
- Skipped evidence: database-backed Go tests did not execute because neither
  `TEST_DATABASE_URL` nor `DATABASE_URL` was configured.
- Missing evidence: the four PostgreSQL scenarios in `IR-025` and the
  Instagram-specific account-fenced row interactions in `IR-026`.

### Risk Review

- The inspected transaction and filtered unique index make partial commit and
  duplicate event creation unlikely, but these are privacy-facing notification
  writes and Must-level guarantees should be demonstrated against PostgreSQL.
- The regular push payload contains display copy plus account binding, category,
  notification ID, and version; no actor DID, Instagram handle, import ID,
  suggestion ID, or evidence is present in routing data.
- The restored Flutter row shares mature actor-notification controls, but shared
  implementation is not a substitute for direct regression evidence where the
  acceptance test explicitly names the Instagram type and account-switch path.
- Live Meta behavior is unaffected by this change. Physical-device push display
  remains an external release gate rather than a code-completion blocker.

### UI Polish Recommendation

- Recommendation: Not needed before correction.
- Reason: The restored copy is concise, identifies Instagram following as the
  discovery source, and does not imply an automatic follow. The current gap is
  behavioral test evidence rather than visual polish.

### Handoff Back To TDD Builder

- Required fixes: `IR-025` and `IR-026`.
- Suggested next failing test: begin with a real-PostgreSQL
  `TestActivateInstagramMatchHonorsPushAndRollsBackWithSuggestion`, followed by
  an `InstagramMatchNotification` row test that switches accounts while Follow
  is awaiting completion.
- Verification to rerun: focused Instagram/notification PostgreSQL tests with a
  configured database, the full Go suite, focused Flutter notification and
  Instagram wire-contract suites, the full Flutter suite, `go vet`,
  `flutter analyze`, and `git diff --check`.

## IR-021 Resolution (2026-08-21)

Status: Closed as superseded

IR-021 belonged to the historical automatic-follow design and assumed that no
member-facing suggestion surface remained. The authoritative AV-007 correction
later reinstated private, lifecycle-bound suggestions and explicit
Follow/Dismiss, while removing the background automatic-follow writer. The
subsequent `instagramMatch` notification restoration did not reverse that
decision.

The current `InstagramSuggestionEligibilityPolicy` and
`SuggestionEligibilityRequest` names are therefore accurate for the active
domain. Renaming them to automatic-follow terminology would now contradict
Requirements §24 (`BR-005`, `FR-033`–`FR-036`) and make the capability boundary
less clear. IR-021 requires no code change and is closed; its original row is
retained as historical review evidence.
