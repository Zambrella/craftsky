# TDD implementation plan: AppView code-audit remediation

## Inputs

- Audit: [`../2026-08-12-appview-code-audit.md`](../2026-08-12-appview-code-audit.md)
- Grouped coding plans: [`README.md`](README.md) and the fourteen AV plan documents in this directory
- Audit snapshot: `7615d1774fef9e601e5024693573fdd93b3181d5`
- Implementation branch baseline: `254fa25b6d9ea8c4ad1a4d046a6e3a5af0eae76d`

The audit plans are the approved contract for this implementation. They contain
the requirement IDs, acceptance criteria, test matrices, migrations, reset and
reconciliation requirements that a normal requirements-to-TDD workflow would
split across several documents.

## Approved product decisions

On 2026-08-14 the product owner approved both recommended branches identified
by the audit-plan index:

1. AV-007 uses strict capability removal. Instagram matching must not perform a
   background PDS follow write. It may create a private, lifecycle-bound follow
   suggestion; only an explicit current-member action may request a follow.
2. Explicit CraftSky deletion may retain minimal, exact-key, non-secret safety
   tombstones while a PDS or object-store effect remains outcome-uncertain. The
   tombstones and job-bound authority are removed after convergence.

Production verified links remain gated on the canonical HTTPS association-file
host and the Android release signing certificate SHA-256 fingerprint. This does
not permit the insecure bearer-in-custom-scheme contract to remain enabled in a
release build.

## Implementation rules

- Implement no behavior without a linked AV ID and acceptance criterion.
- Add or update a focused failing test before each behavior change.
- Run the smallest focused test first, then the affected package, then the
  database/race/release gate appropriate to the change.
- Do not refactor while a focused test is red.
- Keep migrations paired and verify up/down/up on PostgreSQL 16.
- Treat skipped real-PostgreSQL tests as a failure in the full gate.
- Record only commands that actually ran and distinguish unit-only evidence.
- Do not commit or push implementation changes without a new explicit request.

## Test order and status

| Step | Test-loop ID | Findings | Initial failure or required proof | Status |
|---:|---|---|---|---|
| 1 | `GATE-001` | AV-012, AV-013, AV-033, AV-036 | Migration-source errors currently succeed; DB tests skip; vulnerable/unpinned build and formatting gap | In progress |
| 2 | `CFG-001` | AV-022, AV-023, AV-024, AV-030 | Host-derived metadata, production localhost fallback, LAN dev impersonation, invalid duration geometry | Pending |
| 3 | `NET-001` | AV-001, AV-017 | Malicious discovered destinations connect; response/deadline budgets are unbounded | Core complete; integration pending |
| 4 | `MEDIA-001` | AV-016 | Oversized decoded geometry reaches full image decode without aggregate admission | Partial — core/client complete; release gates pending |
| 5 | `LIFE-001` | AV-002, AV-003, AV-006, AV-007 | Ghost terminal owner, stale member authority, upload/delete orphan race, background late follow | Core complete; required-DB and remaining writer integration gates pending |
| 6 | `AUTH-001` | AV-008, AV-018, AV-019 | Bearer crosses browser link; callback loses routing metadata; partial callback strands credentials | Core complete; aggregate/release gates pending |
| 7 | `SESSION-001` | AV-009, AV-010, AV-011, AV-020, AV-021, AV-035 | Refresh race, non-local-first logout, 401/503 collapse, unenforced expiry, partial logout-all, growing maps | Core complete; aggregate/worker gates pending |
| 8 | `HTTP-001` | AV-014, AV-015, AV-031, AV-032 | Pre-header exhaustion, bypassable/unbounded admission, PATCH preflight drift, text/redirect fallthrough | Core complete; shared release gate pending |
| 9 | `TAP-001` | AV-004, AV-005 | Retry count causes ACK/loss and ordering gates discard valid source records | Core complete; startup/cutover gates pending |
| 10 | `PGX-001` | AV-026, AV-027 | Shadowed acquisition error and prefix commits after terminal iterator error | Complete |
| 11 | `INDEX-001` | AV-028, AV-034 | Missing FK-support paths and exact duplicate indexes | Complete |
| 12 | `PUSH-001` | AV-025 | Serial pre-claims outlive leases; provider payload lacks honest at-least-once presentation contract | Core complete; lifecycle/config/release gates pending |
| 13 | `MOD-001` | AV-029 | Moderation output and restoration scheduling can split or duplicate | In progress |
| 14 | `ARCH-001` | AV-037 | Broad stores/composers obscure capability ownership after behavior is corrected | Pending |
| 15 | `FINAL-001` | AV-001 through AV-037 | Clean rebuild, migration round trip, Tap rebootstrap/replay, full race/static/vulnerability/release smoke | Pending |

## Execution notes

### Step 1: `GATE-001`

- First failing test: migration inspector distinguishes empty from missing,
  non-directory, and unreadable sources; operational commands reject all of
  them before connecting to PostgreSQL.
- Focused command: `go test ./cmd/cli -run TestInspectMigrationsDir -count=1`.
- Additional work: database-required test mode, serialized `pg_trgm` bootstrap,
  known fixture/timestamp corrections, explicit unit/full `just` targets,
  PostgreSQL 16 and MinIO CI, pinned toolchain/dependencies/analyzers/scanners,
  migration up/down/up and exact-artifact vulnerability scan.
- Evidence so far: `go test ./...` passed on 2026-08-14 but skipped real
  PostgreSQL tests; `go vet ./...` passed; `gofmt -l .` reported
  `internal/api/scheduled_post_response.go`; installed staticcheck 2025.1.1
  reported that `./...` matched no packages and therefore is not valid gate
  evidence.
- Implemented evidence: the focused migration/test-database tests pass; real
  PostgreSQL 16 tests for `internal/accountdeletion`, `internal/db`, and
  `internal/testdb` pass with `TEST_DATABASE_REQUIRED=true`; the complete
  unit-only Go suite passes after the dependency upgrade. The full database
  and release-equivalent gates remain pending.
- Live scanning moved the toolchain baseline beyond the audit minimum: Go
  1.26.5 still had seven reachable standard-library findings in the current
  vulnerability database, so the implementation now uses Go 1.26.6. Source
  `govulncheck` 1.7.0 is clean after also upgrading pgx, gRPC, `x/net`, and
  `x/crypto`. Binary scanning exposes a govulncheck all-symbol OpenPGP false
  positive even though neither `go list -deps` nor an unstripped binary contains
  an OpenPGP package/symbol; the release gate must encode only that proven,
  exact-ID exception and fail every other finding until the upstream binary
  scanner can represent package absence.

### Step 2: `CFG-001`

- Start with table-driven invalid-origin, missing-production-secret,
  Host-poisoning, remote-dev, and duration relationship tests.
- Use one typed deployment identity and one complete startup validator.
- Coordinate every new duration from later loops through this validator.

### Step 3: `NET-001`

- Core complete: `internal/federatedhttp` now provides one reusable boundary for
  canonical HTTPS URL validation, public-destination enforcement, dial-time DNS
  revalidation and IP pinning, same-origin redirect validation, finite transport
  and purpose-specific client budgets, bounded response bodies, and typed,
  redacted failure categories. Unit tests prove private/special-use and mixed DNS
  answers fail closed, a DNS change to loopback makes zero socket attempts,
  redirect hops are revalidated, slow operations time out, and oversized success
  and error responses fail at the streaming boundary with their bodies closed.
- Exact focused evidence: `GOCACHE=/private/tmp/craftsky-go-cache go test
  ./internal/federatedhttp -count=10`, `GOCACHE=/private/tmp/craftsky-go-cache go
  test -race ./internal/federatedhttp -count=1`, and
  `GOCACHE=/private/tmp/craftsky-go-cache go vet ./internal/federatedhttp` pass;
  `gofmt -l internal/federatedhttp` is empty. The nearby
  `GOCACHE=/private/tmp/craftsky-go-cache go test ./internal/auth -count=1` also
  passes when local test-listener access is available, and `git diff --check`
  passes.
- Integration pending: construct one boundary at startup; inject its metadata,
  OAuth, PDS JSON, and upload clients through every Indigo and anonymous PDS
  path; prevalidate persisted issuer/PDS and discovered OAuth endpoints through
  that same boundary; give handlers and workers operation-level contexts; and
  translate/log failures through `federatedhttp.Classify` so outer URL errors do
  not expose destinations. Instrumented real-listener integration tests and the
  final database/race/release gate remain outstanding.

### Step 4: `MEDIA-001`

- Core complete: scheduled uploads now pass through one mandatory,
  process-wide `ImageValidator`. It admits bounded work before codec parsing,
  runs `DecodeConfig` before full decode, enforces non-disableable axis, pixel,
  and aspect ceilings with overflow-safe arithmetic, rejects format/MIME/bounds
  disagreement and corruption, contains decoder panics, and returns safe `422`
  or retryable `503` envelopes before storage. Low-cardinality validation,
  duration, and in-flight metrics are wired through the route observer.
- Flutter scheduled-media preparation now decodes both newly uploaded and
  already-prepared local JPEG/PNG media off the UI isolate, proportionally
  downsizes it to the same 8,192-axis/16-megapixel/20:1 policy, re-encodes it,
  preserves alt text, and rejects an aspect ratio that cannot be fixed without
  cropping. Immediate public-upload behavior is unchanged.
- TDD/fault evidence: compact over-limit headers never enter full decode;
  corrupt/truncated and header/decode disagreement fail closed; registered
  JPEG, PNG, and WebP succeed; waiter saturation, cancellation, decoder panic,
  observer lifecycle, and permit recovery pass. Time-bounded fuzz runs passed
  for the geometry predicate (73,757 executions in 3 seconds) and real decoder
  boundary (42,299 executions in 3 seconds). Focused Flutter resize/edit tests
  passed (`30` tests).
- Reproducible 4,000-by-4,000 (16-megapixel) benchmark fixtures exist for every
  accepted codec. A one-iteration Darwin/arm64 host probe measured JPEG at
  `16,042,928 B/op` / `41,172,992` peak RSS, PNG at `16,070,768 B/op` /
  `41,025,536` peak RSS, and WebP at `72,043,992 B/op` / `96,911,360` peak RSS.
  These are development evidence only, not the required release-container
  baseline-plus-decoder-plus-margin proof.
- Still open before AV-016 closure: run each benchmark inside the exact release
  container and prove AppView baseline plus worst-codec peak plus safety margin
  fits its hard memory limit (otherwise isolate decoding in a constrained
  worker); apply any lower-only startup settings through `CFG-001`; purge every
  legacy private scheduled row/object through the durable cleanup lifecycle and
  verify the queue and bucket converge; then run the container-level concurrent
  upload and final database/race/static/vulnerability gates.

### Step 5: `LIFE-001`

- Foundation partial: migrations `000038_owner_auth_lifecycle` and
  `000039_owner_effects_terminal_purge` add the positive generation/auth-epoch
  owner authority, irreversible terminal timestamp, finite leased component
  ledger, and deterministic pre-call effect-attempt state. Database constraints
  and a monotonicity trigger prevent generation/epoch rollback, state changes
  without a one-step generation advance, terminal reversal, and terminal-time
  rewriting.
- `internal/ownerlifecycle` now supplies typed `syntax.DID` state/transition
  primitives, first-login departed creation, active-generation rechecks,
  DID-wide auth-epoch advancement, atomic transition/terminal participants for
  later auth-row composition, and canonical shared/exclusive owner fences. All
  fenced SQL uses the same dedicated connection that holds the session advisory
  lock; tests cover `MaxConns=1` and two simultaneous fences saturating a
  two-connection pool without self-starvation.
- Effect attempts allocate a remote identity unique on owner, generation, kind,
  and deterministic key. A same-identity/same-fingerprint retry returns the
  canonical attempt, while fingerprint or expected-CID drift conflicts. The
  dispatched boundary is non-repeatable; departure converts unresolved calls
  to hidden outcome-unknown state; terminal denies every generation; and a
  legitimate rejoin needs an explicit active-fenced current-CID confirmation
  before a reconciled accepted record becomes eligible again.
- Terminal delivery atomically installs the bounded component catalogue and
  closes attempts without scanning owner rows. Leased purge components support
  claim, completion, failure rescheduling, expiry reclaim, and finalization only
  after every component is complete. Participant failure rolls back lifecycle,
  auth/effect closure, and ledger work together.
- PostgreSQL 16 evidence: `TEST_DATABASE_REQUIRED=true go test -race
  ./internal/ownerlifecycle -count=1` passed, including two-pool barriers,
  canonical inverse acquisition, cancellation cleanup, pool-capacity proofs,
  transition races, terminal replay, effect ambiguity/rejoin confirmation, and
  leased purge retry. `TEST_DATABASE_REQUIRED=true go test ./internal/db
  -count=1` passed, including lifecycle migration up/down/up, constraints,
  indexes, and the full database migration-test package. `go vet
  ./internal/ownerlifecycle`, pinned `staticcheck 2026.1
  ./internal/ownerlifecycle`, `gofmt -l`, and whitespace checks passed.
- Terminal visibility and bounded physical purge are now implemented from an
  executable, schema-checked DID-role inventory. Migration `000046` supplies
  role-leading purge indexes. The worker keeps only the approved lifecycle,
  auth/effect, cleanup, and component-ledger tombstones; deletes other roles in
  keyset batches; pre-drains unbounded FK children while locked parent rows
  block late inserts; fails closed on locked children; handles push installation
  ownership; and archives/cancels/waits for moderation restoration work before
  deleting its restrictive parent. Identifier-free observations expose claim,
  component success/failure, and remaining incomplete backlog.
- Serving and background-effect boundaries now reject terminal owners, current
  membership requires the positive active predicate, and pause-after-terminal-
  ACK tests cover profile/feed/post/search/relationship/moderation/recipient
  notification visibility and effect suppression before physical purge. The
  terminal query inventory is executable, so a reviewed serving/effect query
  cannot silently lose its lifecycle predicate.
- Migration `000048` gives every scheduled post and publication tombstone an
  immutable positive owner generation, scopes idempotency/media attachment to
  it, and deliberately cancels pre-generation work while handing private
  objects to durable cleanup. Create, claim, snapshot, frozen-record persistence,
  PDS effect, failure, and finalization all carry the exact generation. The PDS
  callback acquires the owner-generation fence before the per-schedule effect
  lock, and finalization remains inside both, so edit/delete, departure, and a
  stale worker cannot cross the remote-write window.
- Scheduled publication now receives only a callback-scoped durable PDS-effect
  executor, never a raw or retainable PDS client. The combined owner/session
  boundary is entered once, then the per-schedule lock stays held across
  generation/version/ordinal-scoped blob and record attempt persistence,
  remote reconciliation, and local finalization. A dispatched attempt is not
  repeated; definite blob/record conflicts become member-actionable state and
  ambiguous outcomes remain safely retryable or reconciling.
- `scheduledposts.AccountDeletion.DepartureParticipant()` performs only bounded
  transition work: at most three generation-exact schedules and twelve attached
  media rows, with fail-closed invariant checks, exact-key cleanup handoff, and
  row-count compare-and-set. Unclaimed old-generation media stays inaccessible
  after rejoin and converges through the existing expiry or accepted-deletion
  worker rather than making the lifecycle transition scan an unbounded account.
- Incremental evidence on 2026-08-14: the new migration and generation tests
  first failed on missing migration/fields/participant; the cascade tests first
  exposed unbounded post/delivery/job fan-out, late/locked children, missing
  role classification, and push/moderation parent hazards. Unit-environment
  `go test ./internal/ownerlifecycle -count=1` and `go test ./internal/api
  ./internal/relationships ./internal/push ./internal/index ./internal/testdb
  -count=1` pass, as do compile gates for `internal/scheduledposts`,
  `internal/db`, `internal/ownerlifecycle`, and `internal/app`. Focused non-DB
  scheduled publication/freeze/session/tombstone/worker tests pass. The
  guarded scheduled-effect identity/error-mapping tests first failed on the
  missing generation/version attempt seam, then passed together with the full
  scheduled-post package in the unit environment; its PostgreSQL cases skipped
  because neither required database URL was configured.
- Honest remaining gates: rerun the latest cascade refinements, migration
  `000046`/`000048` up/down/up, and scheduled departure/rejoin/effect barriers
  with `TEST_DATABASE_REQUIRED=true` on PostgreSQL 16. Compose the bounded
  departure participant into every active-to-non-active profile transition.
  Complete the shared transaction-scoped lifecycle guard through all projector
  actor/target writes and the reviewed private mutation entry points before
  closing LIFE-001; a snapshot-only predicate is not sufficient.

#### AV-006 scheduled-object durability loop

- Scope: `internal/scheduledposts`, migrations `000040` and `000041`, and
  migration-backed tests. Dependency wiring remains with the coordinated
  lifecycle/auth lane.
- `UT-026 / FR-028, FR-030, NFR-007, RULE-012 / AC-049, AC-054`: add the
  minimized scheduled-object attempt and account-deletion safety-tombstone
  schema, generation-specific keys, and settlement proof decision table.
- `IT-036 / FR-028, FR-030, RULE-013 / AC-051`: persist the object attempt
  before `Put`, fence owner then exact object key, and reproduce accepted
  `Put` -> lost client/lock -> early absent cleanup -> delayed materialization
  -> repeated exact-key deletion.
- `IT-037 / FR-030, RULE-013 / AC-052`: without a tested server-side
  settlement bound, retain the small exact-key cleanup tombstone and reclaim it
  with bounded leases rather than inferring completion from elapsed time.
- `REG-017 / FR-031, NFR-007 / AC-053, AC-054`: preserve the narrow
  owner/job/key/generation authority and leave final operation/OAuth/tombstone
  removal to the coordinated account-deletion transaction after proven
  convergence.
- Completed implementation: migration `000040` persists immutable v2
  generation/attempt keys and remote boundaries; migration `000041` supplies
  the minimized exact-owner/job/key safety relation shared with deletion.
  Upload and cleanup hold owner then object fences, cleanup is lease-bounded,
  and an uncertain outcome cannot settle from elapsed client time or an early
  absent `HEAD`.
- Red/green evidence on 2026-08-14: the migration test first failed because
  the new relations did not exist; the settlement/key tests rejected the
  legacy non-generation key path; the delayed accepted-`Put` barrier and
  lease-bound cleanup tests failed before the durable/fenced behavior was
  added. PostgreSQL-backed `go test ./internal/scheduledposts -count=1`, the
  `000040`/`000041` up/down/up test, and PostgreSQL+MinIO `go test -race
  ./internal/scheduledposts -count=1` now pass.
- Honest remaining gate: no finite S3/MinIO server-side settlement guarantee
  was supplied or proven. Runtime wiring therefore configures no finite bound;
  outcome-uncertain exact-key work remains reconciling indefinitely. PDS
  reconciliation and atomic operation/OAuth/safety-row finalization remain in
  the coordinated account-deletion lane.

### Steps 6-7: `AUTH-001`, `SESSION-001`

- Migration `000038` now installs one lifecycle/epoch/version model for OAuth
  requests, parents, CraftSky children, handoff exchanges/receipts, cleanup
  jobs, and the positive deletion-credential generation. Existing development
  credentials are deliberately reset. Its PostgreSQL up/down/up test passes
  while preserving the existing-member lifecycle backfill.
- OAuth callback persistence is create-only and bound to a durable
  `exchange_started` attempt. Login finalization issues only a 256-bit,
  hash-only, device-bound handoff code; exchange creates an inactive child and
  encrypted retry receipt; confirmation atomically activates parent/child and
  destroys recoverable secret material. Callback rendering uses verified HTTPS
  completion links and no-store/security headers.
- Every ordinary PDS operation now constructs its purpose client inside
  `OAuthSessionCoordinator.WithActiveSession`. The coordinator holds the shared
  owner and exclusive parent advisory fences across the operation, persists
  rotating credentials with `row_version` compare-and-set, validates persisted
  PDS/issuer/token/revocation endpoints through the federated boundary, and
  surfaces indeterminate persistence rather than reporting false success.
- Account-deletion OAuth is a separate callback purpose. Its parent becomes
  childless `deletion_only` and is bound atomically to the exact operation and
  credential generation. The deletion worker uses a separate narrow
  list/delete adapter whose authority also includes originating owner
  generation and the current worker lease token. Terminal refresh moves the
  operation to `reauth_required` and queues the old parent instead of retrying
  stale authority.
- Account-deletion intent, accept, cancel, logout-all exemption, and completion
  now compose with owner transitions and session invalidation in the canonical
  lifecycle -> operation -> parent -> child -> handoff row order. Raw account-
  deletion deletes of OAuth/CraftSky session rows were replaced by local-first
  revocation/cleanup states. Only the exact accepted, childless deletion parent
  can survive logout-all, and it is rebased to the new auth epoch.
- Bearer authentication is database-authoritative on every request: child and
  parent lifecycle, owner state/epoch, absolute parent expiry, and child idle
  expiry are enforced together. Activity throttling is conditional SQL rather
  than process-memory maps. Single logout and DID-wide logout commit local
  invalidation plus durable auxiliary cleanup before returning; middleware
  distinguishes invalid credentials (`401`) from infrastructure failure
  (`503` plus `Retry-After`).
- Red/green PostgreSQL evidence on 2026-08-14: the focused AUTH/SESSION set
  (`TestOAuthCallbackUsesDeletionOnlyPurposeWithoutMintingOrdinaryAccess`, all
  `TestHandoff*`, coordinated ordinary/deletion PDS adapters, both two-pool
  session-coordinator tests, all `TestSessionLifecycle*`, and the initial/
  endpoint/deletion-binding store tests) passes with
  `TEST_DATABASE_REQUIRED=true`; the full PostgreSQL-backed
  `internal/accountdeletion` and `internal/ownerlifecycle` packages pass; and
  `TestOwnerLifecycleMigrationsUpDownUp` passes.
- Honest remaining gates: the aggregate `internal/auth` package still contains
  pre-lifecycle fixtures/tests that directly insert parent rows without owner
  authority or call removed raw Indigo-store paths; those fail only when real
  PostgreSQL is required and must be migrated or deleted before `FINAL-001`.
  Revocation/auxiliary cleanup worker runtime gates, full race/static checks,
  and release association-file verification also remain. Production verified
  links are still blocked on the canonical host and Android release signing
  fingerprint identified above.

### Step 8: `HTTP-001`

- Complete: the AppView listener now reserves a bounded connection slot before
  `Accept`, releases it exactly once, and runs `http.Server` with validated
  header/read/write/idle and header-size ceilings. A separate non-blocking
  request semaphore returns the canonical retryable 503 after headers.
- Complete: trusted-peer-aware `Forwarded`/`X-Forwarded-For` resolution walks
  combined header fields from the controlled edge, groups IPv6 at the
  configured prefix, and falls back to the socket peer on malformed or
  untrusted input. Global/client admission precedes Host, routing,
  authentication, and body reads. The local limiter has a hard entry cap,
  idle expiry, atomic multi-key decisions, and coarse fail-closed overflow
  buckets.
- Complete: route policies compile into a handler-policy bijection and own
  canonical `/v1` path classification, JSON 404/405 responses, explicit HEAD
  rejection, `Allow`, route labels, body policy, and catalogue-derived CORS.
  Every policy's allowed preflight is table-tested, including PATCH, requested
  headers, and all three `Vary` dimensions.
- Complete: body admission no longer buffers and rehydrates payloads. Positive
  oversized `Content-Length` and disallowed bodies fail before authentication;
  accepted fixed/chunked bodies use one streaming `http.MaxBytesReader` plus a
  finite JSON/upload read deadline. Boundary errors become canonical 413/408
  responses and disable connection reuse.
- Complete: migration `000038_owner_auth_lifecycle` supplies the single
  normalized, indexed `request_uri` and request-state schema. Pending OAuth
  admission serializes expiry reclamation, the global capacity check, and the
  complete metadata insert in one PostgreSQL transaction. The independent
  bounded sweeper deletes expired `ready` and retained terminal
  `consumed`/`revoked`/`exchange_failed` rows, but never
  `exchange_started`/`exchange_ambiguous`; login maps capacity exhaustion to a
  coarse retryable 503.
- Red/green evidence on 2026-08-14 included unsafe server geometry, repeated
  forwarding-header spoofing, masked wildcard proxy trust, method-neutral
  catalogue bypass, slow-header slot saturation/recovery, oversized headers,
  slow chunked-body timeout, limiter cardinality, full catalogue preflight,
  canonical path/method envelopes, and concurrent PostgreSQL capacity/sweep
  tests. `go test ./internal/middleware ./internal/routes ./cmd/appview
  -count=1`, `go test ./internal/auth -count=1`, the finding-focused real-
  PostgreSQL `go test -race` across those four packages, and `go vet
  ./internal/middleware ./internal/routes ./cmd/appview ./internal/auth
  ./internal/app` pass.
- Honest remaining shared gate: the broader required-PostgreSQL race suite
  exposes pre-existing/stale auth fixtures that do not seed the coordinated
  owner-lifecycle rows, and one account-deletion route test exercises bare
  `AddRoutes`/`ServeMux` instead of the server catalogue boundary. The
  observability test package also still references the removed automatic-
  follow/Instagram-match types. Production must remain single-replica until a
  verified shared edge limiter is supplied, configure only the actual
  immediate proxy CIDRs, and complete the release browser/reverse-proxy smoke.

### Step 9: `TAP-001`

- Core complete: Tap now ACKs only a committed source/job, committed bounded
  quarantine row, or owner-fenced lifecycle result. Delivery counts no longer
  change retryable failures into terminal ACKs. Malformed identifiers, actions,
  identities, supported records, and oversized records use closed reason codes;
  quarantine supports bounded evidence, listing, leased replay, restart, and
  occurrence-safe redelivery.
- Migration `000045_tap_ingestion_durability` installs current source/tombstone,
  projection-job, quarantine, and repository-job state with natural-key
  idempotency, precise dependency and lease indexes, bounded evidence, and
  up/down/up coverage.
- Valid post-before-profile and interaction-before-subject histories now retain
  source records and wake exact blocked dependencies. Newer tombstones win;
  stale redelivery cannot resurrect them. Projection and repository workers use
  validated positive lease/poll/batch/backoff geometry, `SKIP LOCKED`, expired
  lease reclaim, and bounded exponential retry without a terminal attempt cap.
- Existing profile, post, like, repost, Bluesky profile, follow, block,
  notification, and count projectors run under the projection worker's outer
  transaction. Fault injection proves a serving mutation rolls back when job
  completion fails. Deterministically malformed supported records are
  quarantined in that same transaction.
- Profile source winner selection, activation/departure generation, configured
  auth/session participant, source tombstone, and durable Tap AddRepo work share
  the owner fence. Same-ID ambiguity commits uncertainty and a read-only PDS
  reconciliation job; authoritative resolution carries an expected source
  fingerprint so a newer live event wins. The no-transition duplicate,
  stale, and uncertainty path is also owner-fenced, so an ordering conflict
  cannot split source state from lifecycle authority. Terminal identity ACK is gated on the
  irreversible lifecycle tombstone, configured auth participant, receipt, and
  finite purge-component catalogue; row-count-dependent purge remains leased
  asynchronous work.
- Exact original wire-byte accounting admits every source at or below the
  one-megabyte boundary even when PostgreSQL JSONB's rendered form is larger.
  An integration test commits source/job/receipt, drops the WebSocket before
  ACK, redelivers after reconnect, and proves one durable row of each kind.
  Another consumer test fails seven retryable deliveries and ACKs exactly once
  only after the eighth attempt commits.
- Operator CLI surfaces now list projection/repository backlog and bounded
  quarantine evidence, queue selected quarantine replay, and enqueue read-only
  DID reconciliation. Quarantine and repository claims retain lease/restart
  APIs used by the runtime workers.
- Focused PostgreSQL evidence on 2026-08-14: database-required `go test -race
  ./internal/ingestion ./internal/index ./internal/tap ./cmd/cli -count=1`
  passed; the migration `000045` up/down/up test passed separately; `go vet`
  passed for the same four packages; `internal/tap` passed five consecutive
  loopback-WebSocket runs; `gofmt -l` and focused whitespace checks are clean.
- Still open: replace the deprecated startup adapter with `ingestion.Service`,
  register/start projection, repository, and queued-quarantine replay workers,
  remove `TAP_MAX_RETRIES` from application configuration/Compose, connect
  bounded metrics/readiness, and execute the controlled non-production rebuild.
  The cutover must snapshot and
  stop ingestion, apply migrations, reset public source/projections together
  with any Tap event-ID reset, re-register/rebootstrap repositories despite
  `TAP_NO_REPLAY`, drain/review blocked and quarantined work, then resume. It
  must not fabricate events already ACKed and lost or write to a PDS.

### Step 10: `PGX-001`

- Test both search branches and prefix-then-terminal-error row iterators.
- Ensure no dependent mutation happens before `rows.Err()` is checked.
- Complete: both search branches return their query-acquisition error without
  using invalid rows; notification-subscription and push-claim iterators reject
  a partial prefix on terminal error before any dependent mutation. Focused
  `go test -race ./internal/api ./internal/push -count=1` passes.

### Step 11: `INDEX-001`

- Add catalog, cascade, representative-plan, and up/down/up tests before the
  index migration.
- Coordinate the AV-029 reconciliation-job FK support index.
- Complete: migration `000043_index_maintenance` adds four non-partial,
  leading-column FK support indexes and removes only the three audited
  duplicate non-unique indexes. Catalog assertions preserve the unique
  indexes, representative plans use each new access path, active and
  soft-deleted interaction cascades succeed, and the migration passes a
  focused up/down/up test plus the complete `internal/db` package against
  PostgreSQL 16.

### Step 12: `PUSH-001`

- Core complete: the dispatcher now claims one row only after a bounded worker
  owns a send slot, carries the exact PostgreSQL-returned `lease_expires_at`,
  and calculates each attempt deadline as the earliest of delivery expiry,
  lease expiry minus the finalization margin, and send timeout. Unsafe option
  geometry returns a key-named validation error; the temporary compatibility
  constructor panics rather than silently clamping an unsafe configuration.
  Recipient DIDs and optional actor DIDs are parsed at row decode, and every
  send/finalization path retains lease-token, exact-expiry, installation,
  subscription, and unchanged-token fencing. Tests cover just-in-time claim
  counts, no-useful-window release, exact persisted expiry, malformed/absent
  actor decoding, timeout bounds, reclaim overlap, and stale finalization.
- Provider construction is now typed as a unique event. Android messages are
  data-only and contain bounded display copy plus routing facts and the durable
  `notificationId`, with neither a notification object nor `collapse_key`.
  APNs messages retain an alert but omit `apns-collapse-id`. A five-event
  regression verifies that one registration token receives five independent
  Android messages rather than entering FCM's four-collapse-key namespace.
- Flutter now uses `flutter_local_notifications` for Android app-owned display
  with `(tag = full canonical notificationId, id = fixed type ID)`. A SQLite
  TTL/LRU cache provides persisted compare-and-set stages for `presented`,
  `foregroundEffectEmitted`, and `opened`; its primary key is notification ID
  plus stage, and real isolate contention tests prove one winner. Foreground,
  reconstructed background, local-open reconstruction, five-distinct-ID, and
  presentation/checkpoint crash-boundary tests pass; a corrupt disposable
  cache is reset once. A real plugin method-channel test verifies the full tag
  and fixed ID reach Android. The Android integration includes core-library
  desugaring and a drawable status-bar icon verified by an APK build.
- Dependency impact: `flutter_local_notifications` and `sqflite` are runtime
  dependencies; `sqflite_common_ffi` is a test dependency. Flutter-generated
  macOS/Windows plugin registrants and Android desugaring change accordingly.
  The `/v1/notifications/*` API and routing envelope do not change.
- Focused evidence on 2026-08-14: database-required `go test
  ./internal/push -count=1`, database-required `go test -race
  ./internal/push -count=1`, and `go vet ./internal/push` pass; `flutter test
  test/notifications`, `dart analyze lib/notifications test/notifications
  test/bootstrap/firebase_bootstrap_test.dart`, and `flutter build apk --debug
  --no-pub` pass. The installed Staticcheck binary cannot analyze the upgraded
  Go 1.26 module, so pinned Go-1.26-compatible Staticcheck remains a final gate
  rather than claimed evidence.
- Still open: wire `PUSH_CONCURRENCY`, the finalization margin, and
  `NewDispatcherValidated` through startup/configuration; add the recipient and
  optional-actor owner-lifecycle effect fences; revalidate active
  account/lifecycle eligibility before background local presentation; extend
  the privacy-safe lease observations; connect account-removal cache clearing;
  and run Firebase-device delayed-delivery and APNs duplicate-alert release
  tests. The provider-accepted/database-not-
  finalized and client-checkpoint/OS-presentation gaps remain explicitly
  at-least-once rather than exactly-once guarantees.

### Step 13: `MOD-001`

- Start with output/outbox/receipt fault injection and same-key ambiguous
  commit replay.
- Promotion targets a private suggestion under the approved strict AV-007
  branch and never gains a PDS-write capability.
- Partial evidence: migration `000044_moderation_restoration_outbox` installs
  the live intent, DID-free history, and bounded idempotency-receipt schema,
  including the indexed `ON DELETE SET NULL` reconciliation-job relationship.
  Its focused up/down/up/constraint test passes. A database-only relay now
  promotes `pending` intents to one reconciliation job or `no_work` in the same
  transaction; rollback, two-worker `SKIP LOCKED`, job-retention `SET NULL`,
  idempotent rerun, and race tests pass. Transactional output/receipt insertion,
  owner-lifecycle fencing, worker wiring, retention, and handler error contracts
  remain pending.

### Step 14: `ARCH-001`

- Capture route, wire, SQL, transition, and startup-cleanup characterization
  before moves.
- Keep the refactor behavior-, schema-, SQL-, configuration-, and timing-neutral.

### Step 15: `FINAL-001`

- Rebuild disposable PostgreSQL/MinIO/Tap state.
- Run migration up/down/up, Tap membership rebootstrap and public replay,
  database-required `go test -race ./...`, formatting, vet, pinned staticcheck,
  source and binary `govulncheck`, container build/startup smoke, Flutter tests,
  and verified-link platform checks that have the required release credentials.

## Completion checklist

- [ ] Every AV-001 through AV-037 acceptance criterion has executed evidence or
  an explicitly recorded external release gate.
- [ ] Every schema migration passes PostgreSQL 16 up/down/up.
- [ ] Full tests report zero skipped real-PostgreSQL cases.
- [ ] PostgreSQL/MinIO race suite passes.
- [ ] Flutter unit/widget/platform contract tests pass.
- [ ] `gofmt -l`, `go vet`, pinned staticcheck, and generated/module drift checks pass.
- [ ] Source and exact built AppView/CLI binaries pass pinned `govulncheck`.
- [ ] Clean-stack migration, Tap replay/reindex, worker drain, and startup smoke pass.
- [ ] No insecure compatibility path from the audited behavior remains.
- [ ] Implementation review is completed or explicitly skipped by the product owner.
