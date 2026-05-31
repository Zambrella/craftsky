# TDD Implementation Plan: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Preserve privacy boundaries: report details and internal moderation reasons stay AppView-private.
- Do not submit reports to PDS/Ozone and do not mutate PDS records.
- Do not modify lexicons.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | IT-001 | BR-002, FR-007 | AC-003, AC-004, AC-042 | Missing moderation tables/store |
| 2 | UT-002 | FR-004, FR-027, FR-021 | AC-011, AC-027, AC-041 | No report decoder/normalizer |
| 3 | UT-004 | BR-002, BR-003, FR-007, FR-008, NFR-005 | AC-004, AC-005, AC-035, AC-046 | No placeholder forwarder or metadata contract |
| 4 | IT-004 / UT-008 | FR-001, FR-002, FR-007, FR-026, BR-002 | AC-001, AC-002, AC-003, AC-004, AC-046 | Routes/handlers/response absent |
| 5 | IT-006 / IT-007 / UT-001 / UT-003 / UT-010 | FR-001, FR-002, FR-004, FR-005, FR-006, BR-001, RULE-005 | AC-007, AC-008, AC-010, AC-021, AC-034, AC-043, AC-044 | Validation/canonicalization absent |
| 6 | IT-008 | FR-003 | AC-009 | Routes not registered or middleware missing |
| 7 | UT-011 / IT-017 | FR-009, NFR-001, RULE-006 | AC-020, AC-036, AC-037 | Config fields and gated route absent |
| 8 | IT-002 / UT-009 | FR-010, FR-011, FR-012, RULE-006 | AC-019, AC-023, AC-038 | Moderation store/request absent |
| 9 | UT-005 / UT-006 / IT-018 | FR-011, FR-012, FR-023, FR-024, BR-004, RULE-003 | AC-024, AC-038, AC-040 | Policy not implemented |
| 10 | IT-009 / IT-010 / IT-011 / IT-012 / IT-014 | BR-004, FR-013, FR-014, FR-015, FR-016, FR-025, NFR-003, RULE-003 | AC-012, AC-013, AC-014, AC-015, AC-016, AC-033, AC-040 | Read paths leak moderated rows |
| 11 | IT-019 | NFR-004 | AC-031 | Bounded query pattern unverified |
| 12 | UT-007 / IT-015 / REG-002 / REG-003 | BR-005, FR-017, FR-018, FR-022, NFR-002 | AC-017, AC-018, AC-030, AC-039 | No metadata or raw reason leakage risk untested |
| 13 | IT-013 | FR-019, FR-021 | AC-001, AC-002, AC-028, AC-045 | Flutter client/repository methods absent |
| 14 | UT-014 | FR-021, RULE-002 | AC-028, AC-029, AC-045 | Providers absent |
| 15 | UT-012 | FR-019, FR-020, FR-021, BR-001 | AC-021, AC-025, AC-026, AC-027 | Report actions/UI absent |
| 16 | UT-013 | FR-022, BR-005, NFR-002 | AC-030, AC-039 | Warning banner absent or raw text shown |
| 17 | REG-001..REG-008 | See `02-acceptance-tests.md` regression section | See regression section | Regression coverage pending |
| 18 | MAN-001..MAN-004 | See `02-acceptance-tests.md` manual checks | See manual checks | Manual acceptance pending |

## Implementation Steps

### Step 1: IT-001
- Linked requirements: BR-002, FR-007
- Acceptance criteria: AC-003, AC-004, AC-042
- Write failing test: `appview/internal/api/report_store_test.go` should verify private post/profile report rows with canonical snapshots, normalized details, device ID, timestamps, forwarding status, and forwarding schema metadata.
- Run command: from `appview/`, `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestReportStore|TestModeration'`
- Confirmed failure: Red failure from focused command was meaningful compile failure because `api.NewReportStore`, `api.CreateReportInput`, `api.ReportSubjectPost`, `api.ReportSubjectAccount`, and `api.ReportRow` did not exist.
- Implement: Added `000014_moderation_flow` migration with private `moderation_reports` and future `moderation_outputs` tables; added `ReportStore.CreateReport` with generated report IDs, canonical subject snapshot fields, safe forwarding status/schema/timestamps, and no uniqueness constraint over reporter/subject/reason.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestReportStore|TestModeration'` passed after starting the compose `postgres` service. Nearby `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` also passed.
- Refactor: None beyond `gofmt`; kept store narrowly scoped to persistence.
- Notes: Current highest migration discovered before implementation was `000013`; created `000014`. An initial focused run after implementation failed because local Postgres was not running (`connection refused`); started `docker compose up -d postgres` and reran successfully. IT-001 covers duplicate-report allowance at the store constraint level as part of the persistence loop.

### Step 2: UT-002
- Linked requirements: FR-004, FR-027, FR-021
- Acceptance criteria: AC-011, AC-027, AC-041
- Write failing test: Added `appview/internal/api/report_request_test.go` for omitted/empty/whitespace details, trimmed details, 1,000-character details, 1,001-character rejection, and `other` without details.
- Run command: `go test ./internal/api -run 'TestNormalizeReportDetails|TestValidateReportRequest'`
- Confirmed failure: Red failure was meaningful compile failure because `api.NormalizeReportDetails`, `api.ValidateReportRequest`, and `api.ReportRequest` did not exist.
- Implement: Added `report_request.go` with `ReportRequest`, approved reason taxonomy helper, detail normalization, 1,000-character validation, and report request validation that does not require details for `other`.
- Run command: Focused command passed; combined report-focused command `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestReportStore|TestNormalizeReportDetails|TestValidateReportRequest'` passed.
- Refactor: None beyond `gofmt`.
- Notes: Normalization returns nil for omitted/empty/whitespace details and keeps details as private plain text for later persistence.

### Step 3: UT-004
- Linked requirements: BR-002, BR-003, FR-007, FR-008, NFR-005
- Acceptance criteria: AC-004, AC-005, AC-035, AC-046
- Write failing test: Added `appview/internal/api/report_forwarder_test.go` for placeholder forwarding metadata with private details and canonical subject input.
- Run command: `go test ./internal/api -run 'TestPlaceholderReportForwarder'`
- Confirmed failure: Red failure was meaningful compile failure because `api.NewPlaceholderReportForwarder`, `api.ReportForwardingInput`, and `api.ReportSubjectSnapshot` did not exist.
- Implement: Added `report_forwarder.go` with `ReportForwarder`, `PlaceholderReportForwarder`, `ReportForwardingInput`, `ReportSubjectSnapshot`, and safe `ForwardingMetadata` containing only status/schema/prepared timestamp.
- Run command: Focused command passed. Nearby `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` passed.
- Refactor: None beyond `gofmt`.
- Notes: Metadata JSON was tested not to leak private details, reporter DID, subject DID/rkey/CID, or reason text.

### Step 4: IT-004 / UT-008
- Linked requirements: FR-001, FR-002, FR-007, FR-026, BR-002
- Acceptance criteria: AC-001, AC-002, AC-003, AC-004, AC-046
- Write failing test: Added `report_response_test.go` for minimal accepted response and `report_test.go` for valid post/profile report handlers using fakes.
- Run command: `go test ./internal/api -run 'TestReport(Post|Profile)Handler_AcceptsValidRequest|TestAcceptedReportResponse'`
- Confirmed failure: Red failures were meaningful compile failures for missing `AcceptedReportResponse`, report target types, and report handlers.
- Implement: Added minimal accepted report response serialization; added report handlers that decode/validate request bodies, resolve canonical post/profile report targets, invoke placeholder forwarder, persist reports with safe forwarding metadata, and return only `{reportId,status}`.
- Run command: Focused command passed. Nearby `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` passed.
- Refactor: None beyond `gofmt`.
- Notes: Tests assert private details/reason are excluded from the user-facing response. Route registration and middleware coverage are still pending in IT-008.

### Step 5: IT-006 / IT-007 / UT-001 / UT-003 / UT-010
- Linked requirements: FR-001, FR-002, FR-004, FR-005, FR-006, BR-001, RULE-005
- Acceptance criteria: AC-007, AC-008, AC-010, AC-021, AC-034, AC-043, AC-044
- Write failing test: Added tests for approved/missing/unsupported report reason taxonomy, direct self-report rejection, and post/profile report-target canonicalization.
- Run command: `go test ./internal/api -run 'TestValidateReportRequest|TestReport(Post|Profile)Handler_RejectsSelfReport'` and `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(PostStore_ResolvePostReportTarget|ProfileStore_ResolveAccountReportTarget)'`
- Confirmed failure: Self-report tests initially failed with `201` accepted responses. Target resolver tests initially failed to compile because store resolver methods were absent.
- Implement: Added self-report rejection with `422 invalid_report_target`; added `PostStore.ResolvePostReportTarget` for canonical indexed post snapshots; added `ProfileStore.ResolveAccountReportTarget` for canonical DID profile snapshots.
- Run command: Focused commands passed. Nearby `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` passed.
- Refactor: None beyond `gofmt`.
- Notes: Current profile report resolver supports DID-based canonicalization; handle-to-DID resolution for production profile report routing remains a follow-up within the report handler/route integration work.

### Step 6: IT-008
- Linked requirements: FR-003
- Acceptance criteria: AC-009
- Write failing test: Added `TestRoutes_ReportEndpointsRequireAuthenticatedDevice` for post/profile report route auth and device middleware behavior.
- Run command: `go test ./internal/routes -run 'TestRoutes_ReportEndpointsRequireAuthenticatedDevice'`
- Confirmed failure: Route tests were added after handler integration; no red failure was needed for implementation because report routes were already wired during the prior step. This is recorded as a coverage add-on for IT-008.
- Implement: Added `ReportStore` and `ReportForwarder` to app deps and registered `POST /v1/posts/{did}/{rkey}/reports` and `POST /v1/profiles/{handleOrDid}/reports` behind existing auth/device middleware.
- Run command: Focused route command passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app` passed.
- Refactor: None beyond `gofmt`.
- Notes: Full end-to-end profile reports by handle need a resolver-aware target adapter in a later loop.

### Step 7: UT-011 / IT-017
- Linked requirements: FR-009, NFR-001, RULE-006
- Acceptance criteria: AC-020, AC-036, AC-037
- Write failing test: Added config tests for dev moderation requiring a token when enabled, loading token/default labeler/trusted sources in dev, and clearing dev moderation fields in prod.
- Run command: `go test ./internal/app -run 'TestLoadConfig_DevModeration|TestLoadConfig_ProdClearsDevModerationFields'`
- Confirmed failure: Red failure was meaningful compile failure because `Config` lacked `EnableDevModeration`, `DevModerationToken`, `DevLabelerDID`, and `TrustedModerationSourceDIDs`.
- Implement: Added dev moderation config fields, boolean parsing for `APPVIEW_ENABLE_DEV_MODERATION`, token load/validation, dev labeler default, trusted source DID parsing with dev labeler inclusion, and prod clearing of dev moderation fields.
- Run command: Focused app config command passed; `go test ./internal/app` passed.
- Refactor: None beyond `gofmt`.
- Notes: Config portion complete.

### Step 8: IT-017
- Linked requirements: FR-009, NFR-001, RULE-006
- Acceptance criteria: AC-020, AC-036, AC-037
- Write failing test: Added route tests proving `POST /v1/dev/moderation/ozone-events` is unavailable in prod or dev flag-off configuration, and rejects missing/invalid `X-Craftsky-Dev-Moderation-Token` when fully enabled.
- Run command: `go test ./internal/routes -run 'TestRoutes_DevModeration'`
- Confirmed failure: Red failure was meaningful route behavior: enabled dev route returned `404` instead of token-gated `403` for missing/invalid token.
- Implement: Added `DevModerationOzoneEventsHandler` with the dedicated token gate and conditional route registration only when `Env == dev`, `EnableDevModeration == true`, and token is non-empty.
- Run command: Focused route command passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app` passed.
- Refactor: None beyond `gofmt`.
- Notes: Handler returns `501 not_implemented` after a valid dev token until the next moderation request/store loops implement ingestion.

### Step 9: IT-002 / UT-009
- Linked requirements: FR-010, FR-011, FR-012, RULE-006
- Acceptance criteria: AC-019, AC-023, AC-038
- Write failing test: Added `moderation_store_test.go` for storing post/account moderation outputs with trusted source DID, subject identity, value, action, optional expiry/internal reason, created timestamp, and indexed timestamp.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestModerationStore_InsertOutput'`
- Confirmed failure: Red failure was meaningful compile failure because `ModerationStore`, input/row types, and moderation constants did not exist.
- Implement: Added `moderation_store.go` with `ModerationStore.InsertOutput`, output row/input types, subject/value/action constants, generated IDs, default created timestamp, and scan support.
- Run command: Focused command passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app` passed.
- Refactor: None beyond `gofmt`.
- Write failing test (continued): Added `moderation_request_test.go` for valid trusted post/account synthetic requests, default source DID, untrusted source rejection, and batch payload rejection.
- Run command (continued): `go test ./internal/api -run 'TestDecodeSyntheticModerationRequest'`
- Confirmed failure (continued): Red failure was meaningful compile failure because `ModerationRequestConfig` and `DecodeSyntheticModerationRequest` did not exist.
- Implement (continued): Added `moderation_request.go` to decode one synthetic output object, validate trusted/default source DID, post/account subjects, values/actions, RFC3339 expiry, and produce `ModerationOutputInput` with canonical post URI.
- Run command (continued): Focused command passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app` passed.
- Notes: Synthetic handler persistence with valid token remains a future integration step; store and request validation are now available.

### Step 10: UT-005 / UT-006 / IT-018
- Linked requirements: FR-011, FR-012, FR-023, FR-024, BR-004, RULE-003
- Acceptance criteria: AC-024, AC-038, AC-040
- Write failing test: Added `moderation_policy_test.go` for same-source negation, expired output inactivity, cross-source output remaining active, hide/takedown precedence over warn, warn-only visibility, and hide/takedown same user-visible hidden behavior. Added store-level active policy test over persisted outputs.
- Run command: `go test ./internal/api -run 'TestComputeModerationPolicy'` and `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestModerationStore_ActivePolicyForSubject'`
- Confirmed failure: Red failures were meaningful compile failures because `ComputeModerationPolicy`, `ModerationPolicy`, `ModerationSubjectRef`, and `ModerationStore.ActivePolicyForSubject` did not exist.
- Implement: Added pure policy computation with expiry filtering, same-source later negate cancellation, cross-source preservation, hide/takedown dominance, and warning-only policy. Added store query to load subject outputs and compute active policy.
- Run command: Focused commands passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app` passed.
- Refactor: None beyond `gofmt`.
- Notes: Read-path enforcement will consume the policy in later list/direct-read loops.

### Step 11: IT-009 / IT-010 / IT-011 / IT-012 / IT-014
- Linked requirements: BR-004, FR-013, FR-014, FR-015, FR-016, FR-025, NFR-003, RULE-003
- Acceptance criteria: AC-012, AC-013, AC-014, AC-015, AC-016, AC-033, AC-040
- Write failing test: Added store tests for direct hidden post/hidden author `ErrPostNotFound`, timeline omission of hidden post and hidden author, direct hidden profile `ErrProfileNotFound`, and notification omission for hidden actors/subjects.
- Run command: Focused commands for each new read-path test, then `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`.
- Confirmed failure: Red failures were meaningful behavior failures: direct post/profile reads returned visible rows; timeline and notification lists leaked hidden rows.
- Implement: Added active hide/takedown SQL predicates with same-source negate/expiry handling to post direct reads, post list reads, timeline, comment branch joins, profile direct reads, and notification list actor/subject filters. Added moderation table DDL to store test schemas.
- Run command: Focused read-path commands passed. Broader AppView API/routes/app command passed.
- Refactor: None beyond `gofmt`.
- Notes: Pagination/performance-specific verification remains pending in IT-019. Warning metadata remains pending.

### Step 12: IT-019
- Linked requirements: NFR-004
- Acceptance criteria: AC-031
- Write failing test: Added timeline pagination test proving hidden rows are filtered before limit/cursor selection so a page returns the first visible rows and the next cursor advances deterministically.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestTimelineStore_ListTimeline_FiltersBeforeLimitForDeterministicPagination'`
- Confirmed failure: This was a coverage add-on after SQL-level filtering had already been implemented in the prior read-path loop, so it passed on first run.
- Implement: No additional implementation required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: The implementation uses local SQL `NOT EXISTS` predicates in list queries and no remote/per-row service calls. Manual `MAN-004` remains for deeper query-plan review.

### Step 13: UT-007 / IT-015 / REG-002 / REG-003
- Linked requirements: BR-005, FR-017, FR-018, FR-022, NFR-002
- Acceptance criteria: AC-017, AC-018, AC-030, AC-039
- Write failing test: Added `post_response_test.go` and `profile_response_test.go` coverage for safe generic `moderation.warningKind` metadata and omitted metadata on unwarned posts; added store-level warning hydration tests for direct post/profile reads with private `internal_reason` fixtures.
- Run command: `go test ./internal/api -run 'TestBuild(Post|Profile)Response_.*Moderation'` and `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'Test(PostStore_ReadOne|ProfileStore_Read)_AttachesWarningMetadata'`
- Confirmed failure: Response tests first failed to compile because `PostRow`/`ProfileRow` had no moderation warning field and `PostResponse`/`ProfileResponse` had no moderation metadata. Store tests then failed behaviorally because warn-only post/account outputs left `ModerationWarningKind` nil.
- Implement: Added shared `ModerationMetadata{warningKind}` response DTO, optional `moderation` fields on post/profile responses, response builders that copy only generic warning kind, and SQL warning hydration for post-level warn (`post`), account-level authored-post warn (`author`), and profile warn (`profile`) using active local moderation outputs with expiry/negation checks.
- Run command: Focused warning metadata commands passed. Broader `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` passed.
- Refactor: Ran `gofmt`; kept raw `internal_reason`, source DID, output IDs, and counts out of response structs.
- Notes: Warn-only subjects remain visible because existing hide/takedown filters only suppress `hide`/`takedown`; post-level warning wins over author warning through SQL `CASE` ordering. Flutter warning rendering remains pending in `UT-013`.

### Step 14: IT-013
- Linked requirements: FR-019, FR-021
- Acceptance criteria: AC-001, AC-002, AC-028, AC-045
- Write failing test: Added focused Flutter API client tests in `post_api_client_test.dart` and `profile_api_client_test.dart` for post/profile report requests, accepted report response parsing, and omission of absent optional details.
- Run command: `flutter test test/feed/data/post_api_client_test.dart test/profile/data/profile_api_client_test.dart`
- Confirmed failure: Red failure was meaningful compile failure because moderation report models did not exist and `PostApiClient.reportPost` / `ProfileApiClient.reportProfile` were undefined.
- Implement: Added `ReportSubmission` and `ReportResult` moderation models; added post/profile API client methods for `/v1/posts/{did}/{rkey}/reports` and `/v1/profiles/@{handleOrDid}/reports`; added repository interface/production adapter methods and updated fakes with report callbacks for upcoming provider/UI tests.
- Run command: Focused Flutter API command passed.
- Refactor: Ran `dart format`; kept report result minimal (`reportId`, `status`) and request body limited to `reasonType` plus optional `details`.
- Notes: Provider submit/retry state remains pending in `UT-014`; report action visibility and dialog validation remain pending in `UT-012`.

### Step 15: UT-014
- Linked requirements: FR-021, RULE-002
- Acceptance criteria: AC-028, AC-029, AC-045
- Write failing test: Added `report_post_provider_test.dart` and `report_profile_provider_test.dart` for in-flight duplicate-submit prevention, success result state, post-report error surfacing, and retry after failure.
- Run command: `flutter test test/feed/providers/report_post_provider_test.dart test/profile/providers/report_profile_provider_test.dart`
- Confirmed failure: Red failure was meaningful compile failure because `report_post_provider.dart`, `report_profile_provider.dart`, and their generated Riverpod provider symbols did not exist.
- Implement: Added `ReportPost` and `ReportProfile` Riverpod mutation notifiers returning `AsyncValue<ReportResult?>`; each ignores submissions while loading, calls the corresponding repository report method, surfaces `AsyncError` on failure, exposes accepted result on success, and supports `reset()` back to idle.
- Run command: Ran `dart run build_runner build --delete-conflicting-outputs` to generate providers, then the focused provider tests passed.
- Refactor: Ran `dart format`.
- Notes: Dialog/sheet state will own preserving selected reason/details across provider errors in the upcoming `UT-012` UI loop; provider retry works by calling `submit` again after the error state.

### Step 16: UT-012
- Linked requirements: FR-019, FR-020, FR-021, BR-001
- Acceptance criteria: AC-021, AC-025, AC-026, AC-027
- Write failing test: Added widget tests for `ReportSubjectSheet` approved reasons and validation, `PostCard` report menu action visibility, and `ProfileActions` visitor-only report action visibility.
- Run command: `flutter test test/moderation/widgets/report_subject_sheet_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_actions_test.dart`
- Confirmed failure: Red failure was meaningful compile failure because report reason models/report sheet were absent, `PostCard` had no `onReport`, and `VisitorProfileActionSet` had no `onReport`.
- Implement: Added approved report reason taxonomy, localized report labels/copy, shared `ReportSubjectSheet` with required reason and 1,000-character detail validation, `PostCard` report menu callback, and visitor-profile report action while self-profile actions remain report-free.
- Run command: Regenerated localization with `flutter gen-l10n`; focused report UI command passed.
- Refactor: Ran `dart format`.
- Notes: Initial loop added report entry points and shared sheet validation. A later gap-closure loop below wires the sheet to page-level provider submissions and transient success/error feedback.

### Step 17: UT-013
- Linked requirements: FR-022, BR-005, NFR-002
- Acceptance criteria: AC-030, AC-039
- Write failing test: Added model/widget tests proving post/profile moderation metadata decodes, post cards render generic post/author warning copy, and profile pages render generic profile warning copy without raw reason text.
- Run command: `flutter test test/feed/models/post_test.dart test/profile/models/profile_test.dart test/feed/widgets/post_card_test.dart test/profile/profile_page_test.dart`
- Confirmed failure: Red failure was meaningful compile failure because moderation metadata model/fields and warning banner widget did not exist.
- Implement: Added `ModerationMetadata` Dart model, wired optional metadata into `Post` and `Profile` mappers, registered its mapper, added localized warning copy, and rendered `ModerationWarningBanner` from `PostCard` and `ProfileMetaSection`.
- Run command: Regenerated localization and mappers, then the focused warning UI command passed.
- Refactor: Ran `dart format`.
- Notes: Warning UI uses only generic localized strings for `post`, `profile`, and `author`; no raw server-side reason fields are represented in the client metadata model.

### Step 17a: AT-001 / AT-002 / AT-012 Flutter report-flow wiring gap closure
- Linked requirements: BR-001, FR-019, FR-020, FR-021, RULE-002
- Acceptance criteria: AC-001, AC-002, AC-025, AC-026, AC-028, AC-029, AC-045
- Write failing test: Added page-level widget coverage proving `FeedPage` reports another user's post through the report sheet/repository and shows the transient success message; `ProfilePage` reports a visitor profile through the report sheet/repository and shows the transient success message; `ReportSubjectSheet` catches submit failure, preserves selected input/details, and allows retry.
- Run command: `flutter test test/feed/feed_page_test.dart test/profile/profile_page_test.dart test/moderation/widgets/report_subject_sheet_test.dart`
- Confirmed failure: Red failures were meaningful behavior gaps: feed page did not expose `Report post`, profile report action only showed a placeholder snackbar so the report sheet/reasons were absent, and report sheet submit failures escaped as test exceptions instead of preserving retryable input.
- Implement: Added `reportSubmitSuccess` and `reportSubmitError` localized copy; added retryable inline error handling to `ReportSubjectSheet`; added `showPostReportSheet` and `showProfileReportSheet` coordinators that invoke the Riverpod report providers, close on success, show the transient confirmation, and preserve the sheet on failure; wired report callbacks for non-self posts in feed, profile post/comment tabs, and post thread cards; wired visitor-profile report action to the profile report sheet.
- Run command: `flutter gen-l10n && dart format lib test`, then `flutter test test/feed/feed_page_test.dart test/profile/profile_page_test.dart test/moderation/widgets/report_subject_sheet_test.dart`
- Result: Passed (`All tests passed!`).
- Refactor: Kept report-flow coordination in a small moderation widget helper; no persisted reported-state UI was added.
- Notes: Self-post/profile report actions remain hidden; AppView still rejects bypass self-report requests.

### Step 18: REG-001..REG-008 and final verification
- Linked requirements: regression coverage from `02-acceptance-tests.md` §6 plus all completed Must requirements.
- Acceptance criteria: regression section plus AC-001 through AC-046 where covered by automated focused tests.
- Write failing test: Regression coverage was accumulated in the focused loops above: unmoderated read-path behavior, omitted `moderation` metadata compatibility, warn-only visibility, not-found behavior for hidden direct reads, notification response shape, middleware behavior, no PDS submission/write path, and unchanged Flutter post/profile controls.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
- Result: Passed.
- Run command: from repo root, `just test`
- Result: Passed (`go test -race ./...` under `appview/`).
- Run command: from `app/`, `flutter test test/feed test/profile test/moderation test/notifications`
- Result: Passed (`All tests passed!`).
- Refactor: None.
- Notes: Manual checks `MAN-001` through `MAN-004` were not run by this agent; they remain recorded for human/local follow-up because they require local UX/accessibility/log/performance inspection beyond automated test execution.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
- [x] Stage completion commit created

## Execution Notes
- 2026-05-30: Read workflow documents and coding plan from disk. User approved implementation after plan review. Created native TODO list from approved test order.
- 2026-05-30: Completed IT-001 red-green loop for BR-002/FR-007 and AC-003/AC-004/AC-042. Added private report persistence migration/store and verified focused + nearby AppView API tests.
- 2026-05-30: Completed UT-002 red-green loop for FR-004/FR-027/FR-021 and AC-011/AC-027/AC-041. Added report details normalization and request validation foundation.
- 2026-05-30: Completed UT-004 red-green loop for BR-002/BR-003/FR-007/FR-008/NFR-005 and AC-004/AC-005/AC-035/AC-046. Added placeholder forwarder seam with safe metadata only and no PDS/Ozone submission dependency.
- 2026-05-30: Completed IT-004/UT-008 red-green loop for valid AppView report intake and minimal response privacy. Handler route registration remains pending for IT-008.
- 2026-05-30: Completed report validation/canonicalization loops for UT-001/UT-003/UT-010 and handler validation portions of IT-006/IT-007. Added route registration/middleware coverage for IT-008.
- 2026-05-30: Completed UT-011 and IT-017 dev moderation config/route gating. Synthetic ingestion validation and persistence remain pending.
- 2026-05-30: Completed IT-002 moderation-output persistence store. UT-009 request/trusted-source validation remains pending.
- 2026-05-30: Completed UT-009 synthetic moderation request/trusted-source validation.
- 2026-05-30: Completed UT-005/UT-006/IT-018 policy semantics for apply/negate/expiry/precedence and hide/takedown enforcement equivalence.
- 2026-05-30: Completed initial read-path hide/takedown enforcement for timeline, direct post/profile, comment list joins, and notifications. IT-019 pagination/performance and warning metadata remain pending.
- 2026-05-30: Completed IT-019 pagination coverage for timeline filtering-before-limit. Warning metadata remains pending.
- 2026-05-30: Completed UT-007/IT-015/REG-002/REG-003 AppView warning metadata slice. Post/profile responses now emit only generic `moderation.warningKind` for active warn outputs and omit moderation metadata when unwarned; raw internal reasons remain private.
- 2026-05-30: Completed IT-013 Flutter API/repository report-method slice. Post/profile clients send accepted report requests and parse minimal accepted responses; repository interfaces/adapters/fakes expose the methods for provider/UI loops.
- 2026-05-30: Completed UT-014 Flutter report mutation providers. Providers prevent duplicate requests while loading, expose accepted report results, and allow retry after errors.
- 2026-05-30: Completed UT-012 Flutter report action/validation slice. Post and visitor-profile report entry points are present, self-profile actions omit report, and the shared report sheet exposes approved reasons plus detail-length validation.
- 2026-05-30: Completed UT-013 Flutter warning banner slice. Post/profile models decode optional moderation metadata and generic inline warning banners render for post, profile, and author warnings without raw reason text.
- 2026-05-30: Completed AT-001/AT-002/AT-012 Flutter report-flow gap closure. Feed and profile pages now open the report sheet, submit through providers/repositories, show transient success, and preserve retryable input after failures.
- 2026-05-30: Completed REG-001 through REG-008 automated regression/final verification. Focused AppView API/routes/app tests, broader `just test` race suite, and Flutter feed/profile/moderation/notifications tests all passed. Manual checks MAN-001 through MAN-004 were not run in-agent and are documented as human follow-up.
- 2026-05-31: Read `06-implementation-review.md` and reopened TDD implementation for required review fixes IR-001 through IR-004. Added review-fix loops for valid synthetic route ingestion, profile report handle-or-DID resolution, and remaining read-path hide/takedown enforcement gaps. Prior completion checklist above reflects the pre-review implementation state and will be superseded by the review-fix completion notes below.

## Implementation Review Fix Plan

| Step | Review Finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|---|
| RF-1 | IR-001 | IT-017 / AT-006 | FR-009, FR-010, FR-011, FR-012, RULE-006 | AC-019, AC-023, AC-036, AC-038 | Valid dev route returns `501`; no route-to-store persistence test |
| RF-2 | IR-002 | IT-007 / UT-010 / AT-002 | FR-002, FR-006, RULE-005 | AC-002, AC-008, AC-044 | Handle-based profile reports do not resolve to canonical DID |
| RF-3 | IR-003 | IT-010 / IT-011 / IT-019 / AT-007 | BR-004, FR-013, FR-014, FR-015, NFR-003, NFR-004, RULE-003 | AC-012, AC-013, AC-014, AC-040 | Profile authored lists and thread/comment paths can leak moderated rows |

### Review Fix RF-1: IT-017 / AT-006
- Linked review finding: IR-001
- Linked requirements: FR-009, FR-010, FR-011, FR-012, RULE-006
- Acceptance criteria: AC-019, AC-023, AC-036, AC-038
- Write failing test: Add route-level coverage for a fully enabled dev synthetic moderation request that persists one trusted output and returns `201 indexed`, plus invalid/untrusted requests that do not mutate state.
- Run command: `go test ./internal/routes -run 'TestRoutes_DevModerationRoute(PersistsValidOutput|RejectsInvalidWithoutMutation)'`
- Confirmed failure: Red failure was meaningful compile failure because `app.Deps` had no `ModerationStore` and the registered dev route handler accepted only the token, so the route had no persistence dependency. This matches IR-001's missing route-to-store seam.
- Implement: Added `ModerationStore` to app deps and initialized it in `newDeps`; wired `ModerationRequestConfig` plus `ModerationStore` into dev route registration; updated `DevModerationOzoneEventsHandler` to validate the dev token, decode one trusted synthetic request, persist it through `InsertOutput`, map malformed/untrusted/validation errors to documented envelopes, and return `201 {"outputId":"...","status":"indexed"}`.
- Run command: `go test ./internal/routes -run 'TestRoutes_DevModerationRoute(PersistsValidOutput|RejectsInvalidWithoutMutation)'` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: This closes the current `501 not_implemented` seam and adds negative-path mutation protection for untrusted sources.

### Review Fix RF-2: IT-007 / UT-010 / AT-002
- Linked review finding: IR-002
- Linked requirements: FR-002, FR-006, RULE-005
- Acceptance criteria: AC-002, AC-008, AC-044
- Write failing test: Add DB-backed profile report target tests for handle and DID resolution, submitted handle snapshots, malformed/unresolvable identifiers, and hidden-but-indexed eligibility.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestProfileStore_ResolveAccountReportTarget|TestReportProfileHandler'`
- Confirmed failure: Red failure was meaningful compile failure because `api.NewProfileReportTargetResolver` did not exist; the production route only injected `ProfileStore`, whose `ResolveAccountReportTarget` accepted DID inputs but not handles.
- Implement: Added `ProfileReportTargetResolver`, which strips optional `@`, accepts direct DIDs through the store path, resolves handles via the configured `HandleResolver`, checks indexed profile existence without applying profile visibility filtering, canonicalizes to DID, and records the submitted handle snapshot. Updated profile report route wiring to use this resolver wrapper.
- Run command: `go test ./internal/api -run 'TestProfile(Store_ResolveAccountReportTarget|ReportTargetResolver)'` passed.
- Refactor: Ran `gofmt` on touched files.
- Notes: The added handle test seeds an active account hide output and still resolves the indexed account for report eligibility, preserving RULE-005 / AC-044.

### Review Fix RF-3: IT-010 / IT-011 / IT-019 / AT-007
- Linked review finding: IR-003
- Linked requirements: BR-004, FR-013, FR-014, FR-015, NFR-003, NFR-004, RULE-003
- Acceptance criteria: AC-012, AC-013, AC-014, AC-040
- Write failing test: Add store tests for profile posts, profile comments, root comments, branch replies, branch-around focus, hidden parent/focus hydration, account-level hide, and pagination where hidden rows precede visible rows.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestPostStore_.*Moderation|TestPostStore_List.*Filters'`
- Confirmed failure: Red failures were meaningful behavior failures against Postgres: `ListByAuthor`, `ListCommentsByAuthor`, and `ListRootComments` returned hidden rows; `ListCommentBranchReplies` filtered after its recursive page limit and returned a short page when a hidden row preceded visible rows.
- Implement: Applied `postVisibleModerationPredicate` to `ReadPostByURI`, profile-authored post/comment lists, root comment lists, branch pagination CTEs before `LIMIT`, branch-around CTEs before `LIMIT`, and the branch `hasMore` cursor check so focus/parent hydration and pagination use visible rows only.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestPostStore_(List.*FiltersModeratedRowsBeforeLimit|ReadPostByURI_HiddenPostReturnsNotFound)'` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Added coverage for profile-authored posts, profile-authored comments, root comments, branch replies, and URI hydration of hidden posts. Account-level filtering is covered through the shared predicate and existing direct/list tests.

### Review Fix Final Verification
- Linked review finding: IR-004
- Linked requirements: All requirements touched by IR-001 through IR-003.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
- Result: Passed.
- Run command: `just test`
- Result: Passed (`go test -race ./...` under `appview/`).
- Run command: from `app/`, `flutter test test/feed test/profile test/moderation test/notifications`
- Result: Passed (`All tests passed!`).
- Remaining gaps: Manual checks `MAN-001` through `MAN-004` were not run by this agent and remain human/local follow-up as previously documented.

## Review Fix Completion Checklist
- [x] IR-001 fixed with route-to-persistence coverage for valid synthetic moderation ingestion.
- [x] IR-002 fixed with handle-or-DID profile report target resolution and hidden-indexed eligibility coverage.
- [x] IR-003 fixed with store-level moderation filtering coverage for profile-authored and thread/comment surfaces plus URI hydration.
- [x] IR-004 addressed with red-phase regression tests and updated implementation evidence.
- [x] Focused AppView verification passed.
- [x] Full AppView `just test` verification passed.
- [x] Focused Flutter verification passed.
- [x] Stage completion commit created for review fixes: `fix: address moderation flow review`.

## Second Implementation Review Fix Plan

| Step | Review Finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|---|
| RF-4 | IR-005 | IT-010 / UT-007 / REG-004 | BR-004, BR-005, FR-016, FR-018, RULE-003 | AC-016, AC-018, AC-039, AC-040 | `readNonCraftsky` fallback bypasses account hide/takedown and misses warn metadata. |
| RF-5 | IR-006 | IT-007 / UT-010 / AT-002 | BR-001, FR-002, FR-006, RULE-005 | AC-002, AC-008, AC-044 | Profile report target resolution only accepts `craftsky_profiles` rows. |

### Review Fix RF-4: IT-010 / UT-007 / REG-004
- Linked review finding: IR-005
- Linked requirements: BR-004, BR-005, FR-016, FR-018, RULE-003
- Acceptance criteria: AC-016, AC-018, AC-039, AC-040
- Write failing test: Added profile-store tests for a hidden Craftsky account with a `bluesky_profiles` cache row, a hidden/taken-down non-Craftsky account, and a warn-only non-Craftsky account with private raw reason fixture.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestProfileStore_ReadByDID_(HiddenCraftskyAccountWithBlueskyCacheReturnsNotFound|HiddenNonCraftskyAccountReturnsNotFound|WarnedNonCraftskyAccountAttachesWarningMetadata)' -v`
- Confirmed failure: Red failures were meaningful behavior failures: hidden Craftsky and non-Craftsky profiles returned visible rows through fallback, and warn-only non-Craftsky profile rows had `ModerationWarningKind = nil`.
- Implement: Added account-level moderation policy lookup before non-Craftsky cached/hydrated reads; `hide`/`takedown` now returns `ErrProfileNotFound` before fallback/hydration, and warn-only fallback profiles attach generic `profile` warning metadata without raw reason fields.
- Run command: Focused RF-4 command passed. Nearby `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestProfileStore|TestProfileReportTargetResolver|TestReportProfileHandler'` passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Hidden/taken-down profile responses still use the same not-found path and do not reveal moderation-vs-absence. Warning metadata remains limited to `warningKind: profile`.

### Review Fix RF-5: IT-007 / UT-010 / AT-002
- Linked review finding: IR-006
- Linked requirements: BR-001, FR-002, FR-006, RULE-005
- Acceptance criteria: AC-002, AC-008, AC-044
- Write failing test: Added report-target tests for cached non-Craftsky DID resolution, cached non-Craftsky handle resolution with submitted handle snapshot, hydratable non-Craftsky handle resolution, and unresolvable non-Craftsky failure behavior.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestProfile(Store_ResolveAccountReportTarget|ReportTargetResolver)' -v`
- Confirmed failure: Red failures were meaningful behavior failures: cached and hydratable non-Craftsky profile report targets returned `profile: not found` because resolution checked only `craftsky_profiles`.
- Implement: Expanded account report target existence checks to accept `craftsky_profiles` or `bluesky_profiles`, and to hydrate a non-Craftsky profile through the existing profile hydration seam before deciding not-found. Kept moderation visibility out of report eligibility and preserved canonical DID plus submitted handle snapshots.
- Run command: Focused RF-5 command passed. Nearby profile/report command passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Malformed/unresolvable identities still fail with `ErrProfileNotFound`; hidden-but-indexed eligibility remains independent from read visibility.

### Second Review Fix Final Verification
- Linked review findings: IR-005, IR-006
- Linked requirements: All requirements touched by RF-4 and RF-5.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
- Result: Passed.
- Run command: `just test`
- Result: Passed (`go test -race ./...` under `appview/`).
- Run command: from `app/`, `flutter test test/feed test/profile test/moderation test/notifications`
- Result: Passed (`All tests passed!`).
- Dart MCP analyzer: Ran over the Flutter app root. Initial run found one analyzer error in `app/lib/profile/data/dummy_profile_repository.dart` because it had not implemented newer `ProfileRepository` methods; fixed by adding dummy report and social-list implementations. Re-run reported no Dart analyzer errors/syntax errors; remaining diagnostics are existing infos/warnings only.
- Remaining gaps: Manual checks `MAN-001` through `MAN-004` were not run by this agent and remain human/local follow-up as previously documented.

## Second Review Fix Completion Checklist
- [x] IR-005 fixed with non-Craftsky fallback hide/takedown enforcement and warning metadata coverage.
- [x] IR-006 fixed with cached/hydratable non-Craftsky profile report target coverage.
- [x] Focused AppView verification passed.
- [x] Full AppView `just test` verification passed.
- [x] Focused Flutter verification passed.
- [x] Dart MCP analyzer reports no Flutter syntax/analyzer errors.
- [x] Stage completion commit created for second review fixes: `fix: address profile moderation review`.
