# TDD Implementation Plan: AppView Saved Posts

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Derive every owner from authenticated request context and keep saved data AppView-private.
- Do not create a commit or push without explicit authorization.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | FR-019, NFR-006 | AC-024, AC-030 | Fails because migration files/tables do not exist |
| 2 | UT-001 | FR-004, RULE-007 | AC-007, AC-008 | Fails because folder-name validation does not exist |
| 3 | UT-002 | FR-002, RULE-002 | AC-003, AC-031 | Fails because tri-state request decoding does not exist |
| 4 | UT-003–UT-004 | FR-007, FR-009, NFR-003 | AC-011, AC-027 | Fails because saved/folder cursor codecs do not exist |
| 5 | UT-006–UT-007 | FR-005, FR-020, RULE-003 | AC-009, AC-033, AC-034 | Fails because mutation result/status and timestamp behavior do not exist |
| 6 | IT-002 | FR-002, FR-003, FR-020, NFR-004, RULE-001–RULE-003 | AC-003, AC-004, AC-006, AC-023, AC-031, AC-034 | Fails because saved-post persistence does not exist |
| 7 | IT-003–IT-004 | FR-004–FR-007, FR-017, FR-020, NFR-004, RULE-004, RULE-006–RULE-007 | AC-005, AC-007–AC-011, AC-023, AC-028, AC-032–AC-034 | Fails because folder persistence/handlers do not exist |
| 8 | IT-005 | FR-008–FR-009, FR-017, NFR-003–NFR-004, RULE-002–RULE-003 | AC-012, AC-013, AC-027, AC-032 | Fails because scoped keyset listing does not exist |
| 9 | UT-009–UT-010 | FR-010–FR-011, FR-017 | AC-002, AC-005, AC-009, AC-012, AC-014–AC-015, AC-032 | Fails because saved response/error mapping does not exist |
| 10 | IT-006 | FR-001, FR-003–FR-004, FR-008, FR-015–FR-016, FR-020 | AC-001, AC-006, AC-008, AC-020, AC-022, AC-031–AC-032, AC-034 | Fails because HTTP handler contracts do not exist |
| 11 | IT-007 | FR-001, FR-010, RULE-008 | AC-002, AC-014 | Fails because canonical saved-list hydration does not exist |
| 12 | UT-008, IT-008 | FR-012–FR-013, RULE-006 | AC-016, AC-017 | Fails because saved-list policy/context shaping does not exist |
| 13 | UT-005, IT-009 | FR-013–FR-014, RULE-006, RULE-008 | AC-016–AC-019 | Fails because saved lifecycle and descendant cleanup do not exist |
| 14 | IT-010 | FR-007, FR-010–FR-011, FR-014, FR-019, NFR-002, NFR-004 | AC-015, AC-026, AC-028 | Fails because shared viewer-saved hydration does not exist |
| 15 | IT-011, AT-010 | FR-002, FR-006, FR-016, FR-018, FR-020, RULE-001 | AC-022, AC-023 | Fails because concurrent outcomes are not proven |
| 16 | IT-012 | FR-008, FR-015 | AC-020 | Fails because seven routes/policies are not registered |
| 17 | UT-011, IT-013 | BR-004, FR-015–FR-017, NFR-001, NFR-005, RULE-005 | AC-021, AC-025, AC-029 | Fails because privacy/observability boundaries are not proven |
| 18 | IT-014 | FR-011, FR-019, NFR-002 | AC-026 | Fails because query plans are not guarded |
| 19 | IT-015 | FR-011, NFR-006 | AC-015, AC-030 | Fails because all canonical surfaces lack additive saved viewer state |
| 20 | AT-001–AT-009 | Linked Must requirements in `02-acceptance-tests.md` | AC-001–AC-029, AC-031–AC-034 | Fails until vertical contracts compose |
| 21 | REG-001–REG-007 | FR-010–FR-019, NFR-006, RULE-005, RULE-008 | AC-015, AC-030 | Fails if implementation regresses existing behavior |

## Implementation Steps

### Step 1: IT-001

- Write failing test: Add `internal/db/saved_posts_migration_test.go` for up/down/up, constraints, FK actions, duplicate names, isolation, and indexes.
- Run command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/db -run TestSavedPostsMigration -count=1`
- Confirmed failure: `read up migration: ... 000024_saved_posts.up.sql: no such file or directory`.
- Implement: Add reversible `000024_saved_posts` migration only.
- Run command: Same focused command.
- Refactor: No refactor was needed.
- Notes: Green with the focused real-Postgres command. The migration creates only `saved_post_folders` and `saved_posts`; the test proves duplicate names, owner/post uniqueness, same-owner folder integrity, exact-post and owner cascades, non-destructive folder deletion with stable `savedAt`, the four planned indexes, unrelated-schema preservation, and up/down/up reversal.

### Step 2: UT-001

- Write failing test: Add the folder-name Unicode/validation table.
- Run command: `cd appview && go test ./internal/api -run TestSavedPostFolderName -count=1`
- Confirmed failure: Compile failed because `api.NormalizeSavedPostFolderName` did not exist.
- Implement: Centralized trim/rune-count/slash/backslash/control validator.
- Run command: Same focused command.
- Refactor: No refactor was needed.
- Notes: Green with the focused unit command. The validator trims outer whitespace, counts Unicode code points, preserves accepted casing, permits emoji/punctuation/duplicate display values, and rejects empty, over-100, slash, backslash, and control-character inputs with a `validation_failed` name field.

### Step 3: UT-002

- Write failing test: Add absent/empty/omitted/null/value/malformed request cases.
- Run command: `cd appview && go test ./internal/api -run TestSavedPostRequest -count=1`
- Confirmed failure: Compile failed because `api.DecodeSavePostRequest` did not exist.
- Implement: Strict optional-body tri-state decoder.
- Run command: Same focused command.
- Refactor: Kept body-error construction in one private helper while green.
- Notes: Green with the focused unit command. No body/whitespace/empty object preserve omission, explicit null is distinct, an opaque value round-trips, and malformed, wrong-type, trailing, and unknown-field requests are rejected without inventing an owner field.

### Step 4: UT-003–UT-004

- Write failing test: Add saved-list and folder cursor round-trip/compatibility/order cases.
- Run command: `cd appview && go test ./internal/api -run TestSavedPostCursor -count=1`
- Confirmed failure: Compile failed because the saved/folder cursor types and codecs did not exist.
- Implement: Scope/sort-bound envelope cursor codecs without owner DID.
- Run command: Same focused command.
- Refactor: Kept scope/sort validation in two narrow private helpers while green.
- Notes: Green with the complete focused cursor set. Saved cursors bind all/folder/unfiled scope and newest/oldest sort, bind the exact folder when scoped, include savedAt/URI keysets, reject malformed or incompatible reuse, and omit owner DID. Folder cursors round-trip duplicate folded names using distinct opaque IDs.

### Step 5: UT-006–UT-007

- Write failing test: Add mutation status and timestamp-effect tables.
- Run command: `cd appview && go test ./internal/api -run 'TestSavedPost(Status|Timestamp)' -count=1`
- Confirmed failure: Compile failed because mutation status and timestamp helpers did not exist.
- Implement: Result types and narrow status/timestamp helpers.
- Run command: Same focused command.
- Refactor: Kept status selection and timestamp decisions narrow and side-effect free while green.
- Notes: Green with the focused unit command. Created saves map to 201, every existing-save result maps to 200, existing saves preserve `savedAt`, resaves receive the new server time, and only rename advances folder `updatedAt`.

### Step 6: IT-002

- Write failing test: Add real-Postgres save upsert/move/unfile/unsave/resave and cross-owner cases.
- Run command: Focused real-Postgres store test with `TEST_DATABASE_URL`.
- Confirmed failure: Compile failed because `SavedPostStore`, its options, and stable feature errors did not exist.
- Implement: Owner-scoped atomic save store behavior.
- Run command: Same focused command and nearby store tests.
- Refactor: Localized the save transaction, state scan, folder ownership lock, and optional-string comparison while green.
- Notes: Green with the focused real-Postgres command. The store serializes one owner/URI mutation, validates same-owner folder assignment inside the transaction, preserves omission versus explicit null, keeps the initial `savedAt` through repeats/moves, isolates Bob's row, makes unsave idempotent without target resolution, and stamps a later resave with the injected server clock.

### Step 7: IT-003–IT-004

- Write failing test: Add folder CRUD/order/timestamp and non-destructive delete cases.
- Run command: Focused real-Postgres folder store/handler tests.
- Confirmed failure: IT-003 first failed because folder types/CRUD/list methods did not exist; IT-004 then failed because `DeleteFolder` did not exist.
- Implement: Folder queries and handler-facing results.
- Run command: Same focused commands and nearby tests.
- Refactor: Localized folder parsing, owner predicates, row scans, and cursor construction while green.
- Notes: Both focused real-Postgres tests are green. Duplicate/case-variant names receive distinct opaque IDs; owner-private pages sort by `lower(name), id`; rename preserves ID/createdAt and advances only updatedAt; adding a save does not touch folder metadata; non-owner rename is indistinguishable from missing; delete atomically unfiles with stable `savedAt` and is an idempotent no-op for repeated, missing, malformed, or foreign IDs.

### Step 8: IT-005

- Write failing test: Add all/folder/unfiled, newest/oldest, tie, cursor mismatch, and isolation pages.
- Run command: Focused real-Postgres saved-list tests.
- Confirmed failure: Compile failed because saved-reference/filter types and `ListSavedRefs` did not exist.
- Implement: Indexed owner-scoped keyset listing.
- Run command: Same focused command.
- Refactor: Limited dynamic SQL to fixed comparator/direction fragments selected from validated enums; kept all values parameterized.
- Notes: Green with 103 real saves. All/folder/unfiled scopes traverse exactly once across multiple pages in newest and oldest saved-time/URI order; Bob's same-post save is isolated; malformed and cross-scope/sort cursors return invalid cursor; missing, malformed-storage, and Bob-owned folder scopes return the same folder-not-found error.

### Step 9: UT-009–UT-010

- Write failing test: Add saved DTO/viewer JSON and operation-specific folder error cases.
- Run command: Focused response/error unit tests.
- Confirmed failure: Compile failed because saved response DTO/viewer fields and operation-specific folder error mapping did not exist.
- Implement: Additive response fields and stable error translation.
- Run command: Same focused commands plus existing post-response tests.
- Refactor: Kept folder-error translation as one handler-facing helper and left quote previews unchanged.
- Notes: Green with focused response/error tests. Saved items serialize exact post/reply identity plus camelCase save metadata; canonical posts always expose `viewerHasSaved` and nullable `viewerSavedFolderId`; no folder name is embedded; shared engagement application carries both fields; assignment/rename/scoped-list not-found maps to the same 404 while delete maps to no content.

### Step 10: IT-006

- Write failing test: Add seven-handler success, validation, standard-error, and immediate-read cases.
- Run command: Focused saved handler tests.
- Confirmed failure: Compile failed because saved page/interface types and all seven handlers did not exist.
- Implement: Handler DTOs/interfaces and mutation/list endpoints.
- Run command: Same focused commands.
- Refactor: Centralized owner/path extraction, JSON/error writing, body decoding, and query parsing while green.
- Notes: Green with focused handler tests. Save returns 201/200 from committed store outcomes; unsave constructs the canonical URI without resolving the target; folder create/rename/delete/list and saved-list filters use camelCase contracts, defaults, stable errors, and owner context; invalid identifiers, unknown JSON fields, and incompatible filters are rejected through standard envelopes. Route-level auth/device/body/rate enforcement remains assigned to IT-012.

### Step 11: IT-007

- Write failing test: Add ordinary/project/quote/comment/nested-reply hydration cases.
- Run command: Focused saved hydration tests.
- Confirmed failure: Compile failed because `NewSavedPostService` and the batch saved-list hydration boundary did not exist.
- Implement: Bounded canonical post/quote/handle hydration while preserving saved order.
- Run command: Same focused commands and nearby canonical post tests.
- Refactor: Reused existing handle resolution, canonical response building, shared engagement application, and quote hydration; preserved reference order in one service loop.
- Notes: Green with the focused hydration test. Ordinary, project, quote, direct-comment, and nested-reply saves retain their exact URI/order, save metadata, project/quote shapes, and canonical reply root/parent references while using batch row and engagement calls.

### Step 12: UT-008, IT-008

- Write failing test: Add direct-access mute and strict block/moderation/membership/context cases.
- Run command: Focused saved policy tests.
- Confirmed failure: Unit policy tests first failed because the saved policy decision function did not exist. The real-Postgres context test then failed because the post store did not expose required-context state.
- Implement: Current-policy shaping with non-destructive suppression.
- Run command: Same focused commands and relationship/moderation regressions.
- Refactor: Kept current-viewer policy decisions separate from the set-based required-context query while green.
- Notes: Green with focused unit/service tests and real-Postgres context tests. Direct muted saves remain visible, while direct blocked, moderated, non-member, and missing targets are omitted. Replies require an eligible current root-and-parent chain both when creating the save and when listing it; missing, blocked, moderated, or non-member context maps create to not-found and suppresses an existing saved-list item without deleting private state.

### Step 13: UT-005, IT-009

- Write failing test: Add event-decision table and exact/root/intermediate/member deletion cases.
- Run command: Focused lifecycle/index tests with real Postgres.
- Confirmed failure: The lifecycle unit test first failed because the destructive-event decision table did not exist. The real-Postgres index test then retained a saved nested reply after deleting its intermediate ancestor and retained every descendant save after deleting the root.
- Implement: Transactional descendant-save cleanup and owner cascade wiring only.
- Run command: Same focused commands and nearby index/lifecycle tests.
- Refactor: Seeded recursive cleanup from direct children plus root-linked replies whose indexed parent is already absent, then used a cycle-safe parent traversal in the existing transaction.
- Notes: Green with the focused lifecycle unit test and complete Craftsky post-delete suite. Exact, intermediate-ancestor, root, and missing-parent safety-net cleanup remove affected private saves for all owners before deleting only the event post; still-indexed descendants and unrelated saves/posts survive. Owner membership cleanup remains enforced by the migration cascades, while session/device/account and temporary eligibility events are explicitly non-destructive.

### Step 14: IT-010

- Write failing test: Add Alice/Bob full-page shared engagement hydration and bounded-call assertions.
- Run command: Focused engagement/saved response tests.
- Confirmed failure: The real-Postgres engagement summary returned correct public counts and interaction state but left Bob's saved post false with no folder ID.
- Implement: One set-based saved-state lookup inside `EngagementSummaries`.
- Run command: Same focused commands and current engagement tests.
- Refactor: Kept the private owner predicate in one batch query and merged its compact state alongside the existing interaction/reply batches.
- Notes: Green with the complete focused engagement-summary suite. Bob's filed save and Alice's unfiled save hydrate independently across a two-post page; unsaved rows remain false/null, folder names are never selected, and the shared summary seam uses one additional owner-scoped query for the whole URI set.

### Step 15: IT-011, AT-010

- Write failing test: Add controlled duplicate/move/delete/unfile/unsave races.
- Run command: Focused real-Postgres store test under `-race`.
- Confirmed failure: A controlled delete trigger made unsave win after a concurrent save had read the old row; the save then failed with `saved post save update: no rows in result set`.
- Implement: Minimum transaction/constraint adjustments required for serial-valid outcomes.
- Run command: Same focused race command.
- Refactor: Reused the existing owner/post advisory-lock key for unsave and kept the delete inside that transaction.
- Notes: Green under `go test -race` with concurrent duplicate saves, competing folder moves, controlled unsave/resave, folder delete/unfile, and move/delete. Duplicate saves produce exactly one created outcome and one stable timestamp; every final folder/save state is equivalent to a valid serial order.

### Step 16: IT-012

- Write failing test: Add route registration and auth/device/body/rate policy cases.
- Run command: `cd appview && go test ./internal/routes -run 'TestSaved|TestV1RoutePolicies|TestAddRoutes' -count=1`
- Confirmed failure: The focused route test reported all seven saved-post policy keys missing.
- Implement: Register seven routes and policies with local store wiring.
- Run command: Same focused command.
- Refactor: Constructed one saved store/service beside the existing post store and reused the shared middleware policy stack.
- Notes: Green with the saved-route test and global policy-enforcement regressions. All seven routes require an authenticated device; reads use the read rate class, mutations use write, JSON mutations use the standard body cap, and delete/list routes reject bodies. The save resolver applies current direct-access eligibility before persistence, while unsave remains target-independent.

### Step 17: UT-011, IT-013

- Write failing test: Add private sentinels, bounded telemetry, no-external-call, and author non-disclosure cases.
- Run command: Focused saved observability/privacy tests.
- Confirmed failure: The focused test first failed because there was no sanitized saved-list identity-error class or 502 mapping; the resolver's private sentinel error would otherwise have propagated from the service.
- Implement: Bounded diagnostics only where required.
- Run command: Same focused commands.
- Refactor: Collapsed handle-resolution failures to a fixed saved-list sentinel, mapped it to the standard 502 envelope, and removed the viewer DID from the saved-state query error context.
- Notes: Green with private sentinel response/metric validation and a real-Postgres author non-disclosure test. HTTP telemetry contains only the fixed method, route pattern, and status class; private DIDs, URI, folder ID/name, and owner-target pairs are absent. Saving changes only the owner-scoped private row: public engagement counts and the author's viewer state do not change, and no PDS/Tap/notification collaborator exists in the mutation seam.

### Step 18: IT-014

- Write failing test: Add representative `EXPLAIN (FORMAT JSON)` and bounded-query assertions.
- Run command: Focused real-Postgres query-plan test.
- Confirmed failure: No production behavior failed; the missing query-plan guard was the gap, and the approved migration indexes satisfied the first representative-cardinality run.
- Implement: Index/query-shape adjustment only if the meaningful plan test requires it.
- Run command: Same focused command.
- Refactor: No production adjustment was needed.
- Notes: Green against 1,200 saved rows after `ANALYZE` with sequential scans disabled. All/folder/unfiled pages use their owner-scoped ordering indexes, folder pages use the folded-name index, and shared viewer-state hydration uses the composite primary key.

### Step 19: IT-015

- Write failing test: Add saved/unsaved assertions across canonical post-shaped surfaces.
- Run command: Focused existing API surface suites.
- Confirmed failure: No canonical consumer bypass was found; the missing gap was explicit cross-surface assertions after the shared summary seam had been extended.
- Implement: Route any bypassing canonical consumer through the shared summary seam.
- Run command: Same focused commands.
- Refactor: No additional production path was needed; tests populate the same engagement summary seam used by each existing handler.
- Notes: Green across single post, profile posts, projects, comments/replies, timeline, notifications, quote-post outer responses, search, and the already-covered saved list. Filed, unfiled, and unsaved states remain additive and viewer-specific; existing fields/order remain unchanged and the target author sees no save signal.

### Step 20: AT-001–AT-009

- Write failing test: Compose existing focused fixtures into the approved business scenarios where not already covered.
- Run command: Focused saved handler/store/index acceptance suites.
- Confirmed failure: The final traceability review found that save creation checked the target row but not a reply's required context; the new real-Postgres regression returned success after its indexed parent was removed.
- Implement: Only missing behavior linked to the failing acceptance scenario.
- Run command: Same focused commands.
- Refactor: Reused the existing bounded required-context query from saved-list hydration in `ResolveSavedPostTarget`.
- Notes: Green with the composed API, migration, index/lifecycle, and route suites. AT-001–AT-009 are covered by the focused handler/store/policy/privacy/context tests; invalid reply context now produces the same `post_not_found` contract before any private mutation.

### Step 21: REG-001–REG-007

- Write failing test: Extend existing regression assertions only where the additive fields/lifecycle require it.
- Run command: Relevant existing API/index/lifecycle/migration suites.
- Confirmed failure: No remaining compatibility regression was found after the acceptance fix; additive cross-surface assertions and shared fixture updates were the required shields.
- Implement: Minimum compatibility adjustment if a meaningful regression fails.
- Run command: Focused regressions, then full gate.
- Refactor: Formatted the full Go tree and retained every existing public response field, ordering rule, and lifecycle behavior.
- Notes: Green with `just fmt` (`gofmt` plus `go vet ./...`) and `just test` (`go test -race ./...`) against compose PostgreSQL. Existing API, auth, middleware, index, notification, observability, relationship, route, and Tap packages all pass.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Workflow document review was approved; implementation review remains the explicit next-stage choice

## Final Verification

- Passed: focused unit, real-Postgres store/context/concurrency, migration, index/lifecycle, route, privacy, canonical-surface, and JSON query-plan commands.
- Passed: `just fmt` (`gofmt -w .` and `go vet ./...`).
- Passed: `just test` with compose PostgreSQL available (`go test -race ./...`).
- Passed: `git diff --check` and final traceability review; no unlinked source, lexicon, dependency, Flutter, commit, push, or PR change was introduced.
- Passed after implementation-review corrections: focused race-enabled saved API/auth/migration/index suites and the full `just test` repository gate.
- Correction coverage now includes real auth/session lifecycle preservation, dense and mutating keyset pages, hidden-page continuation, production query plans in both directions, unrelated migration data, and diagnostics redaction.

## Stage Notes

- 2026-07-21: Workflow documents loaded from disk; review status Approved; blocking questions None.
- 2026-07-21: Confirmed `000024` remains the next migration number and the worktree was clean before implementation.
- 2026-07-21: No stage commit is enabled.
- 2026-07-21: IT-001 completed red-green. The initial test failed only on absent migration files; the focused real-Postgres test passes after adding `000024_saved_posts`.
- 2026-07-21: UT-001 completed red-green with centralized folder display-name validation.
- 2026-07-21: UT-002 completed red-green with strict tri-state save request decoding.
- 2026-07-21: UT-003–UT-004 completed red-green with scope-bound saved and deterministic folder cursors.
- 2026-07-21: UT-006–UT-007 completed red-green with mutation result and timestamp policies.
- 2026-07-21: IT-002 completed red-green with owner-scoped atomic save persistence.
- 2026-07-21: IT-003–IT-004 completed as two red-green loops for folder CRUD/order and non-destructive deletion.
- 2026-07-21: IT-005 completed red-green with indexed owner-scoped saved-reference pagination.
- 2026-07-21: UT-009–UT-010 completed red-green with additive viewer JSON and stable folder error mapping.
- 2026-07-21: IT-006 completed red-green with all seven handler contracts and strict request parsing.
- 2026-07-21: IT-007 completed red-green with canonical batch saved-list hydration.
- 2026-07-21: UT-008 and IT-008 completed red-green with direct-access mute behavior and strict current target/context policy.
- 2026-07-21: UT-005 and IT-009 completed red-green with lifecycle decisions and transactional exact/descendant cleanup.
- 2026-07-21: IT-010 completed red-green with one owner-scoped viewer-saved batch query in `EngagementSummaries`.
- 2026-07-21: IT-011 completed red-green under `-race`; unsave now shares the owner/post advisory lock with save.
- 2026-07-21: IT-012 completed red-green with seven authenticated device-bound route policies and registrations.
- 2026-07-21: UT-011 and IT-013 completed red-green with sanitized identity errors, bounded metrics, and author non-disclosure.
- 2026-07-21: IT-014 added a green representative-cardinality `EXPLAIN (FORMAT JSON)` index guard; no production index change was required.
- 2026-07-21: IT-015 extended canonical post, profile, project, reply, timeline, notification, search, quote, and saved-list assertions.
- 2026-07-21: Final acceptance review found and closed missing reply-context validation on save creation.
- 2026-07-21: Final `just fmt`, `go vet ./...`, `git diff --check`, and `just test` (`go test -race ./...`) passed. No commit was created.
- 2026-07-21: IR-001–IR-005 correction pass completed in approved order. Two production changes were required: URI-free descendant-cleanup error context and an unexported production saved-list SQL builder used by both the store and plan guard.
- 2026-07-21: Correction tests added real account/session/token lifecycle evidence, 103-folder and between-page mutation coverage, fully suppressed page continuation, all 12 saved-list production plan shapes, and representative interaction/relationship/notification migration preservation.
- 2026-07-21: Post-correction `just fmt`, `go vet ./...`, `git diff --check`, focused `go test -race` across API/auth/db/index, and full `just test` (`go test -race ./...`) passed. No commit was created.

## Implementation Review Correction Pass

### Inputs

- Implementation review: `06-implementation-review.md` — Changes required
- Authorized action: Address required changes
- Open findings: IR-001–IR-005

### Correction Test Order

| Step | Finding / Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| C1 | IR-004 / UT-011 / IT-013 | NFR-001, NFR-005 | AC-025, AC-029 | Fails because the saved-descendant cleanup error includes the event URI and diagnostics coverage is narrower than specified |
| C2 | IR-001 / IT-009 / REG-005 | FR-014, NFR-006 | AC-019, AC-030 | Fails because no real session/device/account lifecycle regression exercises persisted saved state |
| C3 | IR-002 / IT-003 / IT-005 | FR-007, FR-009, NFR-003, NFR-006 | AC-011, AC-013, AC-027, AC-030 | Fails because dense folders, hidden candidates, and rows arriving between page requests are not covered |
| C4 | IR-003 / IT-014 | FR-019, NFR-002 | AC-026 | Fails because the query-plan guard uses simplified newest-only SQL instead of production scope/cursor shapes in both directions |
| C5 | IR-005 / IT-001 / REG-007 | FR-019, NFR-006 | AC-024, AC-030 | Fails because the migration fixture does not preserve representative unrelated interaction, relationship, and notification schema/data |

### Correction Steps

#### C1: IR-004 / UT-011 / IT-013

- Write failing test: Added a real-Postgres indexer failure case with a private saved-URI sentinel and strengthened saved HTTP telemetry coverage for both successful and failed requests, including captured traces/errors and metrics.
- Run command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/index -run TestCraftskyPost_DeleteCleanupErrorRedactsSavedURI -count=1`
- Confirmed failure: The cleanup error contained the complete sentinel URI in `delete saves for <uri> and descendants`.
- Implement: Removed the event URI from the saved-descendant cleanup error context; retained a fixed operation description and wrapped database error. Kept saved-operation telemetry on the established bounded HTTP middleware seam rather than adding a second feature metric/log pipeline.
- Run command: Focused index deletion/redaction tests and `go test ./internal/api -run TestSavedPostDiagnosticsRedactPrivateStateAndClassifyIdentityFailure -count=1`.
- Refactor: No additional production abstraction was needed.
- Notes: Green. The observability test now proves 2xx and 5xx metrics plus captured traces/errors contain no owner DID, target DID, post URI, or folder sentinel. Saved operations emit no feature-specific logs, which minimizes private-data exposure; the shared HTTP observer supplies bounded method, route pattern, status class, result, and classified error/trace context.

#### C2: IR-001 / IT-009 / REG-005

- Write failing test: Added a real auth/Postgres lifecycle regression with independent Alice/Bob folders and saves. It exercises single-session/device logout, all-session logout, a replacement installation/session, lazy OAuth token/session expiry cleanup, and permanent membership deletion.
- Run command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/auth -run TestSavedPostStateSurvivesSessionDeviceAndAccountLifecycle -count=1`
- Confirmed failure: The new regression was green on first execution because the production schema is already correctly decoupled from auth/session rows; IR-001 was a missing-evidence finding rather than missing runtime behavior.
- Implement: No lifecycle behavior change was required. The regression proves session/device/account/token events retain both owners' saved state and the membership foreign-key cascade removes only Alice.
- Run command: Focused auth, saved timestamp, and descendant deletion regression suite.
- Refactor: Removed the production-only `savedPostLifecycleEvent` decision enum and its self-referential unit test; real auth, migration, policy, and index tests now own the lifecycle guarantees.
- Notes: Green. Account switching/reinstall is represented by revoking the old session and creating a replacement session against unchanged membership-owned saved rows.

#### C3: IR-002 / IT-003 / IT-005

- Write failing test: Added a 103-folder duplicate/case-variant real-Postgres traversal, inserted newer saves between newest and oldest page requests, and made the saved-list policy test exercise an entirely hidden candidate page with a continuation cursor.
- Run command: Focused `TestSavedPostStoreListsMoreThanOneHundredDuplicateFoldersExactlyOnce`, `TestSavedPostStoreListsAllFolderAndUnfiledInBothDirections`, and `TestSavedPostServiceAppliesDirectPolicyAndRetainsSuppressedReferences` commands.
- Confirmed failure: The added regressions were green on first execution because the existing keyset implementation already had the required semantics; IR-002 was missing density/mutation evidence.
- Implement: No production behavior change was required.
- Run command: The same focused real-Postgres and service tests.
- Refactor: Kept the new cases beside the existing folder, saved-reference, and policy pagination fixtures; no helper abstraction was needed.
- Notes: Green. Newest traversal excludes a save inserted before its cursor while returning every original row once; oldest traversal appends newer arrivals once; fully hidden candidate pages remain empty but advance with the store cursor; 103 Alice folders page exactly once and exclude Bob.

#### C4: IR-003 / IT-014

- Write failing test: Replaced simplified literal newest-only queries with a 12-case matrix that calls the production saved-reference query builder for all/folder/unfiled scope, newest/oldest direction, and first/cursor page.
- Run command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run TestSavedPostQueryPlansUseOwnerScopedIndexes -count=1`
- Confirmed failure: Compile failed because `savedPostListQuery` did not exist; the production SQL was still embedded directly in `ListSavedRefs`, so the plan test could not guard the exact query shape.
- Implement: Extracted the validated scope/sort production query builder without changing its SQL or parameterization and made `ListSavedRefs` use it.
- Run command: The same focused real-Postgres query-plan command.
- Refactor: Kept the builder unexported and colocated with the store; retained folder-list and shared viewer-state plan checks beside the production saved-list matrix.
- Notes: Green. At 1,200 saved rows, all 12 production shapes use the intended all/folder/unfiled owner-scoped index. Newest uses the indexes' forward direction, oldest uses backward scans, and neither first nor cursor pages use a saved-state sequential scan.

#### C5: IR-005 / IT-001 / REG-007

- Write failing test: Added exact representative-data assertions for existing likes, reposts, follows, mutes, blocks, and notification events after migration up, down, and second up.
- Run command: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/db -run TestSavedPostsMigration -count=1`
- Confirmed failure: Every new assertion failed with `relation ... does not exist` because the migration pre-state fixture contained only profiles, posts, and a generic sentinel.
- Implement: Expanded the pre-state with production-shaped interaction, relationship, moderation, and notification tables plus one stable row in each category; the saved-post migration SQL itself required no change.
- Run command: The same focused real-Postgres migration command.
- Refactor: Centralized unrelated-row verification in one table-driven helper and retained the original generic sentinel/schema checks.
- Notes: Green. Representative public interaction, relationship/moderation, and private notification schema/data survive saved-post up, down, and up again byte-for-value at their stable identifying fields.
