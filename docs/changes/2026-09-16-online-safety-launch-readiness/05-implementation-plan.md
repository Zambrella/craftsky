# TDD Implementation Plan: Pre-Launch Online Safety Readiness

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Authorization And Scope

- The accountable user explicitly approved bounded implementation on 22 September
  2026, including private migrations and safety/privacy/security code.
- The production IWF adapter, guessed provider contracts, mailbox-provider
  automation, production AWS controls, external legal decisions, and launch approval
  remain out of scope.
- DR-001 through DR-008 and GAP-001 through GAP-012 remain open. Passing repository
  tests does not close them or authorize public launch.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and red/green evidence updated after every loop.
- Use only benign synthetic media and protocol-neutral scanner fixtures.
- Never delete a source PDS record or expose restricted data through ordinary sinks.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-003 | AC-003, AC-006 | Fails: scan-state model absent |
| 2 | UT-002, UT-003 | FR-002, FR-004 | AC-002, AC-004, AC-005, AC-007 | Fails: visibility/cache rules absent |
| 3 | UT-004, UT-014, UT-015, UT-016 | FR-006, FR-007, FR-008, FR-034, NFR-005, RULE-004 | AC-003, AC-009, AC-027, AC-028, AC-041 | Fails: retry/config/stub/rescan rules absent |
| 4 | IT-001 | FR-002, FR-009, FR-010, FR-011, FR-018, FR-023, RULE-003 | AC-004, AC-008, AC-010, AC-011, AC-014, AC-019, AC-032, AC-034 | Fails: private schemas absent |
| 5 | AT-002, IT-003 | BR-002, FR-001, FR-003, FR-004, FR-025, RULE-005 | AC-002, AC-003, AC-007, AC-038 | Fails: image records currently project without clearance |
| 6 | AT-003, UT-017, IT-002, IT-004 | FR-001, FR-032 | AC-002, AC-038, AC-039 | Fails: extraction/profile fallback absent |
| 7 | IT-005, REG-001 | FR-002, FR-026, FR-033, FR-034, NFR-001 | AC-004, AC-005, AC-024, AC-040, AC-041 | Fails: replay and local-post invariants unconnected |
| 8 | IT-007, IT-008, IT-021 | FR-006, FR-007, FR-008, FR-028, NFR-001, NFR-004, RULE-004 | AC-003, AC-009, AC-027, AC-028, AC-029 | Fails: worker/readiness/health absent |
| 9 | UT-005, IT-006, AT-008 | FR-003, FR-005, FR-009, FR-011, FR-035, FR-036, RULE-001, RULE-009 | AC-006, AC-008, AC-042, AC-043, AC-044, AC-045, AC-051 | Fails: restricted detection incident absent |
| 10 | UT-010, IT-011, IT-013 | FR-011, FR-013, FR-035, NFR-002, NFR-007, RULE-009 | AC-011, AC-042, AC-044, AC-050, AC-051 | Fails: evidence boundary/authorization absent |
| 11 | UT-009, UT-019, IT-012, IT-022, AT-009 | BR-004, FR-011, FR-012, FR-029 | AC-012, AC-022 | Fails: holds and retention absent |
| 12 | UT-006, UT-007, UT-008, IT-009, IT-014, REG-010 | FR-009, FR-014, FR-015, FR-016, FR-036, RULE-001 | AC-013, AC-016, AC-017, AC-043, AC-045, AC-046 | Fails: separated mappings/system detection absent |
| 13 | AT-007, IT-010 | BR-004, FR-010, FR-013 | AC-010, AC-011 | Fails: CSEA lifecycle absent |
| 14 | AT-013, UT-011, IT-027 | FR-010, FR-021, NFR-006 | AC-010, AC-021, AC-023 | Fails closed until an approved target is configured |
| 15 | AT-014, IT-028 | FR-022, NFR-006 | AC-023, AC-048 | Fails: threat/authority workflow absent |
| 16 | UT-006, UT-023, IT-026, AT-004 | BR-003, FR-014, FR-015, FR-017, FR-022, FR-038 | AC-013, AC-016, AC-046, AC-048 | Fails: grouped report routes absent |
| 17 | UT-018, IT-015, IT-024, AT-005 | BR-003, FR-018, FR-019, FR-037, RULE-006 | AC-014, AC-015, AC-047 | Fails: external intake absent |
| 18 | IT-016, AT-006 | BR-003, FR-018, FR-020, NFR-006, RULE-006 | AC-015, AC-018 | Fails: intake cannot bind appeal chronology |
| 19 | IT-017, AT-010, REG-008 | BR-005, FR-023, RULE-010 | AC-019, AC-049 | Fails: declaration/version persistence absent |
| 20 | UT-012, IT-018, AT-011 | BR-005, FR-024, RULE-010 | AC-020, AC-049 | Fails: eligibility restriction absent |
| 21 | UT-022, IT-020, AT-015, REG-007 | FR-027 | AC-025 | Fails: production video paths remain callable |
| 22 | IT-019, REG-002, REG-003, REG-005, REG-006 | BR-003, FR-020, FR-025, FR-026, RULE-001, RULE-002, RULE-003 | AC-007, AC-013, AC-015, AC-017, AC-024, AC-032, AC-034 | Fails: cross-surface assertions incomplete |
| 23 | UT-013, UT-020, IT-025, AT-016 | FR-022, FR-028, NFR-002, RULE-003, RULE-007 | AC-023, AC-029, AC-032, AC-034, AC-036 | Fails: redaction/priority/admin coverage absent |
| 24 | IT-023, AT-017 | FR-030 | AC-030 | Fails: browser harness absent |
| 25 | UT-021, AT-001, AT-012, REG-004, REG-009 | BR-001, BR-006, FR-008, FR-018, FR-027, FR-028, RULE-008 | AC-001, AC-014, AC-026, AC-028, AC-029, AC-037 | Fails: release/policy gates absent |
| 26 | AT-018, AT-019, MAN-001 through MAN-014 | BR-001, BR-004, BR-005, BR-006, FR-007, FR-013, FR-021, FR-022, FR-029, FR-031, FR-037, NFR-003, NFR-005, NFR-007, RULE-004 through RULE-007, RULE-009 | AC-001, AC-010, AC-011, AC-013 through AC-015, AC-019 through AC-023, AC-026 through AC-033, AC-035, AC-036, AC-047, AC-048, AC-050, AC-051 | Blocked on external/manual evidence |

## Implementation Steps

Each step follows: add one focused test, run the focused command, record a meaningful
failure, add the minimum implementation, rerun to green, run nearby tests, refactor
only while green, and record the result below.

### Step 1: UT-001

- Write failing test: Added `state_test.go` covering the five approved states,
  unknown/empty values, terminal behavior, and display eligibility.
- Run command: `cd appview && go test ./internal/imagesafety -run TestScanState`.
- Confirmed failure: Build failed only because `State`, its constants, and methods
  did not exist.
- Implement: Added the closed `State` type with `Valid`, `Terminal`, and
  `DisplayEligible`; only `clear` is display-eligible.
- Green verification: Focused test passed.
- Refactor: None required.
- Notes: `clear` and `match` are terminal scan outcomes; `pending`, `unavailable`,
  and `error` remain non-terminal workflow states because the latter two are retried.

### Step 2: UT-002 And UT-003

- UT-002 red: `ParentDisplayEligible` was undefined.
- UT-002 green: added the all-clear visibility rule; empty requirements and all-clear
  requirements pass, while every non-clear or unknown state fails closed.
- UT-003 red: `ScanKey`, `Result`, and cache reuse behavior were undefined.
- UT-003 green: added a typed `syntax.CID` cache key containing scanner, policy, and
  corpus identity. Only complete, identical keys with terminal results are reusable.
- Verification: `go test ./internal/imagesafety` passed.
- Refactor: None required.

### Step 3: UT-004, UT-014, UT-015, And UT-016

- UT-004 red: retry policy was undefined.
- UT-004 green: added deterministic bounded exponential retry; invalid/exhausted
  policies return no retry and cannot clear content.
- UT-014 red: scanner deployment configuration/readiness was undefined.
- UT-014 green: only complete dev/test stub configuration is ready. Production,
  unknown, unconfigured, and incomplete configurations cannot mark results clear.
- UT-015 red: scanner contract and fixture stub were undefined.
- UT-015 green: added a provider-neutral reader-based contract and deterministic
  fixture scanner. Unknown/invalid fixtures return `error` and `ErrUnknownFixture`.
- UT-016 red: result completion and rescan policy were undefined.
- UT-016 green: elapsed time is ignored; targeted requests, changed approved keys,
  and non-terminal results require scanning.
- Verification: `go test ./internal/imagesafety` passed.
- Refactor: formatted the package; no behavioral refactor required.

### Step 4: IT-001 (Incremental Migration Coverage)

- Order refinement: IT-001 covers migrations `000073` through `000080`, but each
  schema is introduced immediately before the vertical behavior that owns it. This
  avoids batch-building seven unrelated schemas before their behavior tests.
- `000073` red: the focused migration test failed because the migration file did
  not exist.
- `000073` green: added reversible private image-safety tables, closed state/job/event
  constraints, operational indexes, and the `image_subject_uri` Tap dependency kind.
- Database verification: with the Compose PostgreSQL stack running,
  `TestImageSafetyMigration` passed its up/down/up cycle and
  `TestMigrationVersionsAreUniqueAndPaired` passed.
- Later slices completed migrations `000074` through `000080`; the migration suite
  now covers the complete reversible sequence with closed constraints and indexes.

### Step 5: AT-002 And IT-003

- Initial red: `NewImageSafetyCraftskyPost` and `ReasonImageScanPending` were
  undefined.
- Partial implementation: added a transactional CraftSky post decorator that
  records current requirements, queues missing scans, blocks projection until every
  image is clear, and returns an `image_subject_uri` dependency.
- Focused database test: passes for initial pending, all-clear projection, and a
  later clear-to-error state transition.
- Public-store red: `TestPostStore_ReadPostByURI_NonClearImagePostReturnsNotFound`
  showed that an already projected post remained readable after its current image
  subject state became blocked.
- Public-store green: the shared post visibility predicate now requires a matching
  clear `(subject_uri, source_cid)` revision through the migration-owned
  `appview_image_subject_is_clear` database predicate.
- Query-plan refactor: the opaque stable predicate preserves existing post query
  indexes; `TestPostInteractionListQueriesUseSubjectAndQuoteIndexes` continues to
  require the quote index. Pre-image-safety focused schemas receive an all-clear
  test predicate, while fixtures using migration `000073` remain fail-closed and
  explicitly seed clear subject states.
- Green verification:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/api -count=1`
  and
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/db ./internal/index ./internal/imagesafety ./internal/tap -count=1`
  both passed.
- Status: Complete.

### Step 6: AT-003, UT-017, IT-002, And IT-004

- UT-017 red: profile image selection and fallback types were undefined.
- UT-017 green: added a closed-state selection policy; only `clear` promotes a
  candidate, every other/unknown state keeps the previous clear image, and missing
  history produces the neutral no-URL placeholder.
- AT-003/IT-004 red: no image-safety Bluesky profile projector existed.
- AT-003/IT-004 green: the opt-in profile projector updates safe text immediately,
  reconciles avatar and banner independently, records private candidates, queues
  scans idempotently, promotes current clear candidates, retains prior clear serving
  fields for non-clear replacements, and permits same-revision replay after clearance.
- API surface red/green: the profile PUT response previously echoed newly written,
  unscanned images. It now returns only last-cleared AppView image fields, while an
  explicit null removal is reflected immediately.
- IT-002 red: business image decorators and external-thumbnail extraction were
  absent.
- IT-002 green: post images, external-card thumbnails, business-event images,
  business-product images, avatars, and banners all enter the same versioned scan
  workflow. Shared blobs reuse one result/job while retaining every source link.
- Verification:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/index -count=1`,
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/db ./internal/api ./internal/imagesafety ./internal/tap -count=1`,
  and `CGO_ENABLED=0 go vet ./internal/index ./internal/api ./internal/db ./internal/imagesafety ./internal/tap ./internal/testdb`
  passed.
- Boundary: runtime registration is not assigned an unsafe hardcoded stub identity.
  Validated scanner configuration and production startup/readiness wiring remain
  owned by Step 8 (`IT-007`, `IT-008`, `IT-021`).
- Status: Complete.

### Step 7: IT-005 And REG-001

- REG-001 characterization: an image-bearing create response is prepended only to
  the author's live provider-container cache; direct `postProvider` reads still
  receive `post_not_found`, and authoritative timeline refresh removes the local
  synthetic row while AppView remains closed.
- IT-005 durable lifecycle: added a PostgreSQL integration test using the real Tap
  ingestion store and image-safety projector. Exact redelivery creates no work;
  same-blob edits reuse the clear result; changed blobs create one pending job and
  invalidate old serving eligibility; restart replay reloads the current durable
  source; and a newer delete cannot be resurrected by later scan completion.
- Targeted rescan red: only the pure rescan policy existed; no durable generation
  reset or visibility transition was available.
- Targeted rescan green: added a compare-and-set rescan store that advances one
  `scan_version`, queues one job, blocks current parent subjects and profile serving
  slots, and moves their current projection jobs to the existing
  `image_subject_uri` dependency. Repeating the old expected generation is a no-op.
- Profile replay refinement: transactional profile projection now persists safe
  text/candidates and returns `image_scan_pending` while any current avatar/banner
  candidate is non-clear, leaving a durable job for completion to wake.
- Verification:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/imagesafety ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap -count=1`,
  `CGO_ENABLED=0 go vet ./internal/imagesafety ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap ./internal/testdb`,
  `flutter test test/feed/providers/create_post_provider_test.dart`, and focused Dart
  analysis all passed.
- Status: Complete.

### Step 8: IT-007, IT-008, And IT-021

- IT-007 red: scan rows could be queued by projectors, but no leased worker,
  bounded operation timeout, restart reclaim, dead letter, manual retry, safe event
  trail, or terminal dependency wake existed.
- IT-007 green: added a PostgreSQL worker store and scanner worker with
  compare-and-set lease tokens plus scan generations, bounded retry/backoff,
  expired-lease reclaim, dead letters, idempotent manual retry, and append-only safe
  transition events. Terminal/dead-letter result updates and projection wake-up now
  share one transaction, so a failure cannot strand a completed scan without a
  replay signal. A stale lease cannot complete after a newer claim or targeted
  rescan generation.
- Fetch boundary: added an authoritative-directory PDS blob fetcher that forbids
  redirects and validates response status, declared/response MIME, byte limit,
  declared size, decoded image format, and content CID before scanning.
- Runtime wiring: development uses the validated local-only scanner and starts the
  image worker with the existing process lifecycle. Tap and OAuth profile projection
  now use the same configured scan identity for posts, external thumbnails,
  business images, avatars, and banners.
- IT-008 green: scanner identity and worker geometry are loaded and validated from
  bounded configuration. Stub mode is ready only outside production. Production
  remains degraded and unable to mark clear until a separately validated approved
  adapter sets readiness; no production adapter was added. `/healthz` exposes only
  the scanner-ready boolean and becomes degraded for unsafe/unconfigured production
  state.
- IT-021 green: added moderator-authenticated `GET /v1/admin/safety/status` with safe
  scanner counts, queue age, retry/dead-letter alert state, and a provider boundary
  for later incident/deadline/retention slices. Synthetic providers verify priority,
  deadline, owner/cover, age, and alert ordering without blob, DID, or content data.
- Verification:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/imagesafety ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap ./internal/routes ./internal/app ./cmd/appview -count=1`,
  `CGO_ENABLED=0 go vet ./internal/imagesafety ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap ./internal/routes ./internal/app ./internal/testdb ./cmd/appview`,
  and `git diff --check` passed.
- Status: Complete.

### Step 9: UT-005, IT-006, And AT-008 Detection Scope

- Red: a fixture match could finish as a scan result without a restricted incident,
  subject-specific links, machine-detection event, safe review response, or a hard
  barrier against using the generic completion method to bypass incident creation.
- UT-005 green: one incident is keyed to the shared scan result, every affected
  subject revision is linked idempotently, and replaying the same detection creates
  no duplicate incident, link, or event.
- IT-006 green: match completion now atomically writes provider/integrity references,
  creates or reuses the restricted incident, links affected subjects, appends the
  detected event, keeps every subject blocked, completes the job, and wakes projection
  dependencies. The transaction creates no report, moderation case, decision, effect,
  strike, suspension, notice, rendered-image copy, or raw scanner payload.
- Failure behavior: malformed match metadata and incident-recording failures consume
  the existing bounded retry/dead-letter policy while content remains blocked. The
  generic scan completion method rejects `match`, so callers cannot omit the
  restricted incident transaction.
- AT-008 detection/review scope green: added moderator-authenticated
  `GET /v1/admin/safety/incidents/{incidentReference}`. It returns the opaque incident
  reference, state, scanner/policy/corpus identity, safe provider/integrity references,
  detection time, and aggregate subject-kind counts. It returns no blob CID, DID,
  subject URI, bytes, raw payload, report, evidence object, or enforcement detail.
- Migration `000074_safety_incidents` is reversible and contains metadata/reference
  columns only; up/down/up and closed constraints are covered.
- AT-008 remains partially open for the authorized human-confirmation,
  `systemDetected` case, and owner-safe notice clauses assigned to Step 12.
- Verification:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/imagesafety ./internal/safetyincident ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap ./internal/routes ./internal/app ./cmd/appview -count=1`,
  `CGO_ENABLED=0 go vet ./internal/imagesafety ./internal/safetyincident ./internal/ingestion ./internal/index ./internal/db ./internal/api ./internal/tap ./internal/routes ./internal/app ./internal/testdb ./cmd/appview`,
  and `git diff --check` passed.
- Status: Complete for Step 9 scope; AT-008 remains partial as noted above.

### Step 10: UT-010, IT-011, And IT-013

- Red: incidents had safe references but no separately authorized evidence boundary,
  least-privilege operator model, or access chronology.
- Green: migration `000075` adds restricted evidence and access records, while
  migration `000079` adds digest-only operator credentials, constrained roles,
  permissions, lifecycle state, and assignments. Evidence access requires explicit
  authorization and appends a safe audit event without exposing evidence through
  ordinary APIs, logs, metrics, traces, or error bodies.
- Verification: safety incident authorization, evidence integration, admin auth,
  redaction, migration, route, and sink-canary tests pass.
- Status: Complete. Production operator provisioning remains an external gate.

### Step 11: UT-009, UT-019, IT-012, IT-022, And AT-009

- Red: no legal-hold model or bounded retention worker protected active in-scope
  evidence while removing expired unheld evidence.
- Green: migrations `000075` and `000080` add holds, retention jobs/attempts, and
  append-only non-sensitive outcomes. The worker is bounded, retryable, idempotent,
  and hold-aware. Account deletion removes ordinary owner state but preserves only
  evidence governed by active retention/hold scope.
- Terminal-account integration now inventories every new DID role and supplies
  role-leading keyset indexes. Restricted incident links remain under safety
  retention; onboarding, age, image-source, profile-candidate, and appeal links are
  deleted.
- Verification: retention policy/worker, hold, legal-hold account deletion, terminal
  inventory, index, and full owner lifecycle suites pass.
- Status: Complete.

### Step 12: UT-006, UT-007, UT-008, IT-009, IT-014, And REG-010

- Red: user reports and machine detections shared an incomplete taxonomy and no
  explicit human-confirmation transition into enforcement.
- Green: migration `000076` separates `userReport` and `systemDetected` case origins,
  expands safety reason/legal classifications, and enforces incident linkage for
  system cases. Human confirmation creates or reuses one system-detected case and
  enters the existing decision/effect pipeline without fabricating a user report.
  Owner-facing presentation remains safe and omits restricted detection detail.
- AT-008 is now complete: detection/review behavior from Step 9 and the authorized
  confirmation/owner-safe notice behavior are both covered.
- Verification: moderation taxonomy, policy, presentation, system-detection,
  adjudication, and service integration tests pass.
- Status: Complete.

### Step 13: AT-007 And IT-010

- Red: CSEA incidents lacked a complete append-only chronology and statutory
  deadline state.
- Green: the restricted workflow records confirmation, preservation, reporting
  preparation/submission state, deadlines, and resolution as safe append-only events.
  Detection alone still causes no enforcement or external report.
- Verification: CSEA workflow and deadline tests pass against PostgreSQL.
- Status: Complete. External reporting authority and production provider evidence
  remain manual/external gates.

### Step 14: AT-013, UT-011, And IT-027

- Red: intimate-image escalation had no approved-target guard or deadline model.
- Green: intimate-image workflow policy validates bounded targets and deadlines and
  fails closed when no approved destination is configured. No guessed provider
  contract or transmission adapter was added.
- Verification: intimate-image and deadline tests pass.
- Status: Complete for repository scope; approved target configuration is external.

### Step 15: AT-014 And IT-028

- Red: credible-threat and authority-request handling had no constrained workflow or
  refusal chronology.
- Green: restricted workflows now model priority, responsible actor, authority
  validation/refusal, state transitions, and safe append-only events without granting
  generalized content access.
- Verification: authority, priority, and deadline tests pass.
- Status: Complete for repository scope; legal decisions remain external.

### Step 16: UT-006, UT-023, IT-026, And AT-004

- Red: report reasons could not represent the approved safety groups consistently
  across wire validation, moderation intake, and Flutter UI.
- Green: camelCase report values, grouped safety categories, immediate-danger
  guidance, child-safety non-redistribution guidance, intellectual-property email
  routing, detail bounds, and retry behavior are aligned end to end.
- Verification: Go report request/intake tests and Flutter report model/widget tests
  pass.
- Status: Complete.

### Step 17: UT-018, IT-015, IT-024, And AT-005

- Red: there was no constrained external safety mailbox boundary or moderator intake
  API.
- Green: migration `000077` and the `safetyintake` package store safe message
  references and metadata, reject unsafe attachment handling, deduplicate provider
  references, and expose moderator-only intake/correspondence operations. Raw email
  bodies and unsafe attachments are not persisted.
- Verification: mailbox boundary, email acceptance, API, auth, and route tests pass.
- Status: Complete. Mailbox-provider automation remains out of scope.

### Step 18: IT-016 And AT-006

- Red: external correspondence could not bind verified owner appeals to moderation
  chronology.
- Green: moderator-only appeal linking requires a verified owner DID and durable
  correspondence reference, records one linkage, and preserves the existing appeal
  chronology and authorization boundaries.
- Verification: safety intake API and acceptance tests pass.
- Status: Complete.

### Step 19: IT-017, AT-010, And REG-008

- Red: onboarding had no explicit age declaration or versioned policy acceptance.
- Green: migration `000078`, API models/store, Flutter onboarding state, and generated
  serialization persist only an explicit 16+ declaration and the required policy
  version. Completion is account-isolated, idempotent, and permanently server-backed.
- Verification: onboarding API/route and Flutter declaration, account-isolation,
  repository, provider, and widget tests pass.
- Status: Complete.

### Step 20: UT-012, IT-018, And AT-011

- Red: verified under-age restriction had no service, route matrix, or client gate.
- Green: eligibility state and append-only revisions use enumerated evidence kinds,
  reviewer/reason/guidance fields, and fail-closed reads. Middleware denies ordinary
  authenticated capabilities while retaining eligibility, safety, appeal, and account
  controls. Flutter initialization and routing surface the restricted state without
  leaking one account into another.
- Test composition now supports an injected eligibility reader while production
  defaults to the PostgreSQL store, preserving fail-closed runtime behavior.
- Verification: eligibility service/middleware/policy, route capability matrix, and
  Flutter account eligibility tests pass.
- Status: Complete.

### Step 21: UT-022, IT-020, AT-015, And REG-007

- Red: production video upload and projection paths remained reachable without the
  approved safety controls.
- Green: production configuration, credential issuance, composer eligibility, and
  AppView projection all fail closed for video. Existing image/text behavior remains
  available, and development-only behavior cannot authorize production.
- Verification: video launch-gate, application config, route, index, and Flutter
  composer/provider regressions pass.
- Status: Complete. Production video remains disabled.

### Step 22: IT-019, REG-002, REG-003, REG-005, And REG-006

- Green regression coverage confirms restricted data never enters ordinary profile,
  post, search, feed, notification, moderation-report, logging, metrics, or tracing
  surfaces; non-clear media stays suppressed; profile fallback remains last-clear;
  and no safety workflow deletes source PDS records.
- Verification: privacy-boundary, terminal visibility, projection, API, route, and
  Flutter regressions pass.
- Status: Complete.

### Step 23: UT-013, UT-020, IT-025, And AT-016

- Red: admin payloads and telemetry lacked complete redaction/priority canaries and
  constrained operator lifecycle coverage.
- Green: structured redaction removes restricted identifiers/content, priority and
  deadline ordering is deterministic, admin endpoints require dedicated moderator
  authentication, and sink canaries verify no sensitive values reach logs, metrics,
  traces, or public errors.
- Verification: redaction, priority, moderator auth, safety status, and observability
  tests pass.
- Status: Complete.

### Step 24: IT-023 And AT-017

- Red: the public site loaded analytics without an executable consent harness.
- Green: legal/reporting pages and a Playwright harness verify PostHog is withheld
  before consent, denial stores no analytics identifier, grant loads only the approved
  host, and Do Not Track overrides a stored grant.
- Verification: `cd web && npm test` passes all 10 browser tests.
- Status: Complete.

### Step 25: UT-021, AT-001, AT-012, REG-004, And REG-009

- Red: policy artifacts, P0 evidence, scanner/video controls, and architecture
  constraints had no executable release gate.
- Green: added the five public policy routes, publication manifest/digest tooling,
  P0 register checks, no-Ozone architecture check, and a fail-closed readiness CLI.
  Its unit tests cover malformed/stale/unapproved evidence and the ready case.
- Current real artifacts intentionally evaluate as blocked because approvals,
  production scanner evidence, and external/manual P0 evidence are unresolved.
- Verification: six readiness unit tests pass; browser and repository suites pass;
  no Ozone dependency or safety-path import is present.
- Status: Complete for repository-owned controls. Public release remains blocked.

### Step 26: External And Manual Gates

- Status: Blocked pending the named external evidence in DR-001 through DR-008 and
  GAP-001 through GAP-012.
- Repository implementation must not mark these checks passed.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps
- [x] All planned bounded automated tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Production IWF adapter remains absent
- [x] No launch blocker is incorrectly marked closed
- [x] Documentation and executed-loop notes updated
- [ ] Implementation review completed or explicitly skipped

## Current Execution Status

- Repository-owned implementation loops 1 through 25 are complete. Migration
  coverage includes reversible migrations `000073` through `000080`.
- Final verification on 22 September 2026 passed:
  `CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test -p 1 ./... -count=1`,
  `CGO_ENABLED=0 go vet ./...`, `flutter test` (2,379 tests), Dart MCP analysis,
  `cd web && npm test` (10 tests),
  `PYTHONPATH=scripts python3 -m unittest scripts/test_online_safety_readiness.py`
  (6 tests), and `git diff --check`.
- The local worktree PostgreSQL setting `max_locks_per_transaction` was raised from
  64 to 256 so full migrated-schema cleanup tests can acquire their required relation
  locks. This is test-environment configuration, not an application change.
- Step 26 remains blocked. No public-launch readiness claim is made; the readiness
  gate must remain `ready: false` until all named external/manual evidence is approved.

## Post-Review Remediation

The final implementation review identified ten repository defects. The remediation
pass completed the following bounded changes without closing any external launch
gate:

1. Business event and profile reads now apply the current-revision image-clear
   predicate on every direct, list, derivative, and eligibility path.
2. Missing or partial scanner identity now produces an explicit unconfigured scan
   key, and Tap/profile decorators remain installed so production fails closed.
3. Moderator bearer tokens are SHA-256 digest authenticated against active,
   unexpired, non-revoked PostgreSQL credentials with explicit permissions and
   incident assignments.
4. Moderator-only command routes now expose the restricted evidence, legal-hold,
   intimate-image, credible-threat, authority, disclosure, closure, and CSEA service
   operations. A dedicated private S3-compatible evidence store is required at
   startup and never exposes a generic URL operation.
5. Restricted-evidence retention now uses the `000080` durable job lifecycle with
   enqueueing, bounded leases, expired-lease recovery, attempt history, retry
   backoff, completion, and dead-letter state. The process starts and drains the
   worker with the shared shutdown context.
6. Account deletion injects the same restricted evidence store into
   `DatabasePrivateCleanup`, so eligible object bytes and metadata are removed while
   active held evidence remains preserved.
7. Evidence deletion and hold creation serialize on the evidence row. Deletion
   rechecks active holds while holding that lock, preventing a hold from committing
   after its object bytes have been removed.
8. Legally sensitive workflow services now accept authenticated `Actor` values and
   enforce explicit workflow, authority-report, and disclosure permissions at the
   service boundary rather than trusting caller-supplied actor strings.
9. PostgreSQL triggers reject `UPDATE`, `DELETE`, and `TRUNCATE` on incident events,
   evidence access/export logs, and workflow events. Down migrations remove their
   trigger functions and up/down/up coverage passes.
10. External intake rejects non-CraftSky HTTP canonical subjects and verifies that
    replayed correspondence provider references belong to the requested intake.

Focused red/green evidence includes business current-revision regressions,
unconfigured scanner identity, digest authentication and assignment loading,
external-intake replay isolation, append-only mutation attempts, bounded retention
dead-lettering, private S3 object-store behavior, and concurrent hold/deletion
serialization.

Post-remediation verification on 22 September 2026 passed:
`CGO_ENABLED=0 TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test -p 1 ./... -count=1`,
`CGO_ENABLED=0 go vet ./...`, and `git diff --check`.
