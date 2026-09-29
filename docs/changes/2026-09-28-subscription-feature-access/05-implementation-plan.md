# TDD Implementation Plan: Subscription Feature Access

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (approved with notes)
- Coding plan: `04-coding-plan.md` (approved by product owner for implementation)

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one failing test before each implementation step.
- Run the smallest relevant test first; refactor only after green.
- Keep traceability and execution evidence updated after each loop.
- Preserve the DID-scoped authority, free baseline, and PDS source records.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001, FR-002, RULE-001, RULE-003 | AC-001, AC-002, AC-005, AC-020 | No shared feature predicate |
| 2 | IT-001, AT-002 | FR-001, FR-002, NFR-002 | AC-001, AC-002, AC-005, AC-006 | Direct paid requests succeed |
| 3 | IT-002–IT-005, UT-003 | FR-002, FR-006, BR-001 | AC-002, AC-003, AC-006, AC-015–AC-017 | Paid routes, lapse and projections unguarded |
| 4 | UT-002, IT-006, IT-012, AT-003 | FR-004, FR-008, RULE-001, BR-002 | AC-004, AC-019 | Legacy account type persists |
| 5 | IT-007, IT-008, AT-004, AT-008 | FR-004, FR-005, FR-008, RULE-002 | AC-009–AC-011 | Business owner and public serving unguarded |
| 6 | UT-004, IT-009, IT-011, AT-009, AT-010 | FR-006, FR-007, RULE-003 | AC-012, AC-016–AC-018, AC-020 | No transition/effect fence |
| 7 | UT-005, AT-001, AT-005–AT-008, IT-010 | FR-003, FR-004, FR-006, NFR-001 | AC-001, AC-007–AC-009, AC-013, AC-015–AC-017 | No Flutter paid presentation |
| 8 | REG-001–REG-006, MAN-001 | FR-001, FR-003, FR-005, FR-007, NFR-001, NFR-002, RULE-003 | AC-005, AC-007–AC-008, AC-010–AC-014, AC-020 | Regression verification pending |

## Implementation Steps
### Step 1: UT-001
- Write failing test: `TestFeatureAccessFollowsAssignedDIDAndEffectiveTier` in `feature_access_test.go`, testing payer, assigned Plus, accessible Business and dormant Business.
- Run command: `go test ./internal/subscriptions -run 'Test.*FeatureAccess'` from `appview/`.
- Confirmed failure: `SelfAccess.AllowsPlus` and `AllowsBusiness` undefined.
- Implement: add access predicates on the effective DID projection.
- Run command: `go test ./internal/subscriptions -run 'TestFeatureAccessFollowsAssignedDIDAndEffectiveTier|TestSelfAccessProjection'` — passed.
- Refactor: not needed.
- Notes: predicates require both effective tier and `GivesAccess`; cancelled-but-accessible projection stays paid.

### Step 2: IT-001, AT-002
- Write failing test: `TestSubscriptionFeatureAccessAssignedDIDNotBillingOwner`, real isolated Postgres licence, direct folder POST via registered route, beneficiary vs payer.
- Run command: `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestSubscriptionFeatureAccessAssignedDIDNotBillingOwner$' -count=1 -v`.
- Confirmed failure: payer received 201 instead of 403. First run without `TEST_DATABASE_URL` was skipped; re-ran with compose Postgres on port 15830.
- Implement: DID-scoped `SelfAccess` check at route boundary for folder creation; standardized `subscription_required` / `subscription_unavailable` envelope.
- Run command: same focused command — passed.
- Refactor: shared `requireTier` wrapper after both Plus and Business checks went green.
- Notes: Real-Postgres DID/owner route proof covers scheduled create/update/manual publish/media staging, folders, pins, metrics and customisation. The suspension-policy fixture now has a paid beneficiary so its earlier assertion remains meaningful. Stateful checks follow below.

### Step 3: IT-002–IT-005, UT-003
- IT-002: scheduled create/update/manual publish/staging are guarded; owner list/get/delete remain accessible. Red: Free scheduled create returned 201. Green: route suite and worker checks.
- IT-003 / UT-003: folder create/rename/delete and Free folder-assignment saves are denied while ordinary saves work. Free list uses all-saved scope and strips folder IDs; Postgres folder associations survive loss and restore without re-saving. Red: rename returned 201; Free save with `folderId` reached the store; flat presentation included folder IDs. Green: `TestSavedPostFolderAssignmentRequiresPlusWithoutBlockingOrdinarySave`, `TestSavedPostsWithoutPlusAreFlatWithoutLosingFolderMembership`, `TestSavedFolderAssociationsSurviveEffectivePlusLossAndReturnOnRestoration` and saved-post widget/API suites.
- IT-004: direct Free pin PUT/DELETE denied, Plus accepted; both pin slots clear transactionally on access loss, never on cancellation with `gives_access=true`, and remain empty after restoration. Pin mutation rechecks access under a shared per-DID fence. Red: Free PUT returned 201; pins remained after unassignment and snapshot expiry. Green: pin route, store, lifecycle and interleaving tests.
- IT-005: Free follower-growth GET/customisation PUT denied. Saved customisation stays stored but visible identity JSON defaults for Free; embedded `viewerSavedFolderId` redacted from Free post responses. Red: Free growth returned 201 and hydrator showed saved choices. Green: route suite, hydrator, redactor and existing owner-only metric tests.

### Step 4: UT-002, IT-006, IT-012, AT-003
- UT-002 / AT-003: effective Business classification overrides legacy flags and is projected in bounded DID batches. Account settings display it read-only. Red: Plus DID was shown as business; selector was still interactive. Green: `TestDerivedBusinessAccountTypeIgnoresLegacyFlag`, `TestFeatureAccessBatchUsesSameEnvironmentAndProviderCriteriaAsSelfAccess`, and AccountPage widget tests.
- IT-006: former account-type route removed for every tier; direct Business declaration/event writes require Business access. Red: old mutation route returned 401 under auth rather than 404/405 and Plus declaration reached handler. Green: Business route dispatch/guard tests.
- IT-012: migration `000078` drops only `craftsky_account_types`, down recreates the empty historical shape. Removed live SQL/mutation handler and obsolete account-deletion/terminal-inventory cleanup. Red: migrated schema retained table. Green: full production migration up/down test, seeded legacy-row removal test, deletion/terminal suites.

### Step 5: IT-007, IT-008, AT-004, AT-008
- IT-007 / AT-008: Free/Plus cannot use Business owner GET/POST/PUT/DELETE, including own event detail; a visitor of any tier can read a licensed owner's eligible event. Flutter hides Business sections and fences owner deep links and already-open management content by the active lease. Red: unlicensed owner reached GET/PUT and stale Business label opened Products. Green: registered-route tests, router/settings/manager widget suites.
- IT-008 / AT-004: declaration, event direct/list, owner list, upcoming indicator, and effective label now use active assigned Business licences (environment, app, mapped tier and anomaly checks). Existing membership/block/moderation filters stay intact; PDS/index source rows persist through loss and restore. Red: licensed owner with old `regular` flag was hidden; old `business` flag served an unlicensed owner. Green: real-Postgres business serving/lapse/restore tests and ingestion/regression suites.

### Step 6: UT-004, IT-009, IT-011, AT-009, AT-010
- UT-004 / IT-011 / AT-009: dedicated `subscription_required` needs-attention code; preflight check before automatic cutoff and a final shared per-DID session fence across private media/PDS append and settlement. Snapshot, assignment and unassignment hold the exclusive matching transaction fence through commits. A claimed job paused before its PDS boundary cannot append after access loss commits. Red: claimed due work still published, expired jobs reported cutoff, owner responses omitted the safe code, and media upload preceded the last check. Green: publication processor/worker interleaving and owner-list/detail response tests.
- IT-009 / AT-010: cancellation with continuing provider access keeps pins and paid classification; effective Plus loss clears pin slots. Unassignment and reassignment fences clear the former beneficiary's pins; restoration never resurrects them. Red: pins survived unassignment/reassignment and snapshot loss. Green: lifecycle tests with held publishing-effect interleavings.
- Lock order: lifecycle shared owner fence/row (where applicable) → billing owner row → subscription/licence rows → access fence → local writes. Scheduled worker holds the existing owner/effect fence before a shared access fence; its own access read takes no conflicting billing row lock. The access fence is domain-separated from owner/OAuth locks and acquisition is bounded to five seconds. Integration tests cover claim/loss, snapshot/effect, unassign/effect and reassign/effect orderings. No PDS append or media upload occurs after a committed access loss.

### Step 7: UT-005, AT-001, AT-005–AT-008, IT-010
- UT-005 / AT-001: Flutter access is active-lease/DID bound and fails closed on pending/error; Free < Plus < Business, including Plus follower growth. Pure predicate, provider lease-switch, keyboard and widget tests pass.
- AT-005–AT-008 / IT-010: reusable labelled Plus lock/dialog/CTA with retry, wired into scheduling, saved folders/move, pin menu, profile customisation and follower growth. Free saved posts use one flat all-saved view without fetching folders; cached Business/customisation owner profiles and shell avatars present defaults while data remains stored. Business section, owner pages and editor options require confirmed Business access. Tier change invalidates active profile, pins, saved folders/posts and growth projections. Red: old selector and Business deep links remained visible; Free foldered post disappeared; Plus control opened paid action; cached Business/colour survived lapse. Green: focused and full Flutter suites.

### Step 8: REG-001–REG-006, MAN-001
- REG-001–REG-006: full Flutter widget/provider suite, real-Postgres race-enabled Go suite, neutrality tests, production migration down/up and AppView release-equivalent gate pass. Ordinary posts/saves remain free and billing does not affect timeline/search ranking.
- MAN-001: **blocked as a physical-device manual check**. No running DTD app/session was available for VoiceOver/TalkBack and keyboard testing inside the signed-in app. Automated semantics and keyboard CTA tests (AC-013) pass; complete the iOS/Android assistive-technology walkthrough before release.

## Execution Status

### Review remediation (2026-09-29)
- IR-001 / IT-003 / FR-006 / AC-016: added a real-Postgres HTTP test for an ordinary Free re-save of a retained foldered post. Initial run failed during fixture setup (owner lifecycle not active); after correcting the fixture it failed by returning the folder UUID in `folderId`. The handler now redacts the mutation response using the current DID's effective access after saving, failing closed for folder presentation while keeping ordinary saves available during access-read errors. Focused test and nearby saved-post handler tests pass. Stored membership remains unchanged and is checked after restoration.
- IR-002 / IT-010 / FR-005 / AC-010–AC-011: a widget test mounts a licensed owner's loaded event detail, lapses to Free and restores Business without navigating. Red: the cached event remained visible on lapse. The owner detail now requires confirmed active-lease Business access before rendering; visitor detail is unaffected. The focused test and complete event-detail widget file pass.
- IR-003 / AT-008 / FR-004 / AC-009: route tests start on each Business manager page and drop access to Free. Red: both remained mounted at their management routes with a Business heading. On a known tier transition away from Business, the active shell now navigates from those routes to Settings; the complete Business route widget file passes.
- IR-004 / IT-009 / FR-006 / RULE-003 / AC-016–AC-017/AC-020: expanded the real-Postgres snapshot sequence to include a second lapsed snapshot and restored access after the cancellation/loss sequence; pins remain empty and folder membership and customisation remain stored at both steps. The focused test passed on its first run (coverage gap, no behavioral failure).
- IR-004 / IT-011 / FR-007 / AC-018: expanded the real-Postgres claimed-job/lapse worker sequence with a second future pending item. It remains scheduled during lapse, does not publish before due after restoration, publishes exactly once at its original due time, and leaves the missed item in needs-attention. The focused test passed on its first run (coverage gap, no behavioral failure).
- Remediation verification: `just test` passed with Go race detection and real PostgreSQL/MinIO (the first attempt timed out at the command runner's 240-second limit; retry with a longer limit passed); `just app-test --no-pub --reporter compact` passed; targeted Dart analysis reported no errors; `just appview-check` passed all release gates; `git diff --check` passed. MAN-001 remains a physical-device follow-up.

### Re-review remediation (2026-09-29)
- IR-006 / IT-010 / FR-001 / FR-005 / AC-010 / EC-004: a mounted owner detail now refreshes its lease-bound access into a controlled pending Future and then an error. Red: Riverpod retained the old Business value during loading and the previously rendered event stayed visible. The detail now requires confirmed non-loading, non-error Business data for the matching DID and lease. Focused test and the entire event-detail widget file pass; existing visitor detail remains unaffected.
- IR-007 / AT-008 / FR-001 / FR-004 / AC-009 / EC-004: tests manually invalidate a previously confirmed Business access provider while its next read is held pending. Riverpod returns `AsyncData` with both the previous Business value and `isLoading=true`. Red: Business Settings entries and the Products/Events create FABs remained available; the first Settings test used a dependency reload (which instead produced `AsyncLoading`) and was corrected to exercise the refresh state. Green: Settings hides Business entries, displays "Loading tier" rather than a stale tier, and both managers hide their create FAB until a confirmed, matching-DID Business result returns. Focused tests and all five nearby Business/Settings/router test files pass.
- UT-005 / FR-001 / EC-004: the shared Plus lock exhibited the same stale `AsyncData(isLoading: true)` authorization issue. A controlled-refresh widget test failed because tapping the pinned-post affordance called its paid action. It now requires confirmed non-loading, non-error access and offers the retry dialog during uncertainty; the focused lock test file passes.
- Re-review remediation verification: `just app-test --no-pub --reporter compact` passed after the last Plus-lock fix; focused Business detail, Settings, Products, Events, router and Plus-lock widget files passed. Targeted Dart analysis reported no errors and `git diff --check` passed. Go and AppView release gates from the preceding remediation are unaffected by these Flutter-only changes. MAN-001 remains a physical-device release follow-up.

- Automated coverage: UT-001–UT-005, IT-001–IT-012, AT-001–AT-010, REG-001–REG-006. MAN-001 blocked as above; no test was silently skipped. The old sandbox fixture's multi-statement prepared query was corrected while verifying environment isolation.
- Final checks: `just test` (Go `-race`, real PostgreSQL + MinIO) **passed**; `just app-test --no-pub --reporter compact` **passed** (full Flutter suite, then focused Business/route/accessibility tests after the last UI changes); `just appview-check` **passed** all release gates, including exact migration up/down, Docker builds and security checks. `git diff --check` passed.
- `just app-analyze` reports one pre-existing `avoid_catching_errors` info in unchanged `app/lib/shared/api/pds_mutation_contract.dart:45`; no diagnostics were introduced in changed files. The recipe exits 1 on infos.
- No commits or pushes requested; stage is left as an uncommitted diff for implementation review. MAN-001 is the only outstanding platform check.

## Completion Checklist
- [x] All Must requirements covered by automated tests or documented manual gap
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Physical-device MAN-001 (VoiceOver/TalkBack and focus order) — requires signed-in device session
- [x] Implementation review recorded in `06-implementation-review.md`; IR-001–IR-004 and IR-006–IR-007 addressed and ready for re-review (IR-005/MAN-001 pending)
