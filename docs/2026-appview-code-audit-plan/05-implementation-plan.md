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
| 1 | `GATE-001` | AV-012, AV-013, AV-033, AV-036 | Migration-source errors currently succeed; DB tests skip; vulnerable/unpinned build and formatting gap | Complete |
| 2 | `CFG-001` | AV-022, AV-023, AV-024, AV-030 | Host-derived metadata, production localhost fallback, LAN dev impersonation, invalid duration geometry | Complete; deployment proof pending |
| 3 | `NET-001` | AV-001, AV-017 | Malicious discovered destinations connect; response/deadline budgets are unbounded | Complete; public-environment smoke pending |
| 4 | `MEDIA-001` | AV-016 | Oversized decoded geometry reaches full image decode without aggregate admission | Automated code/client/container/concurrency proof complete; authorized legacy cleanup pending |
| 5 | `LIFE-001` | AV-002, AV-003, AV-006, AV-007 | Ghost terminal owner, stale member authority, upload/delete orphan race, background late follow | Complete; controlled data cutover pending |
| 6 | `AUTH-001` | AV-008, AV-018, AV-019 | Bearer crosses browser link; callback loses routing metadata; partial callback strands credentials | Complete; verified-link release gates pending |
| 7 | `SESSION-001` | AV-009, AV-010, AV-011, AV-020, AV-021, AV-035 | Refresh race, non-local-first logout, 401/503 collapse, unenforced expiry, partial logout-all, growing maps | Complete |
| 8 | `HTTP-001` | AV-014, AV-015, AV-031, AV-032 | Pre-header exhaustion, bypassable/unbounded admission, PATCH preflight drift, text/redirect fallthrough | Complete; deployment proxy smoke pending |
| 9 | `TAP-001` | AV-004, AV-005 | Retry count causes ACK/loss and ordering gates discard valid source records | Complete; controlled Tap cutover pending |
| 10 | `PGX-001` | AV-026, AV-027 | Shadowed acquisition error and prefix commits after terminal iterator error | Complete |
| 11 | `INDEX-001` | AV-028, AV-034 | Missing FK-support paths and exact duplicate indexes | Complete |
| 12 | `PUSH-001` | AV-025 | Serial pre-claims outlive leases; provider payload lacks honest at-least-once presentation contract | Complete; physical-device release gates pending |
| 13 | `MOD-001` | AV-029 | Moderation output and restoration scheduling can split or duplicate | Complete |
| 14 | `ARCH-001` | AV-037 | Broad stores/composers obscure capability ownership after behavior is corrected | Complete |
| 15 | `FINAL-001` | AV-001 through AV-037 | Clean rebuild, migration round trip, Tap rebootstrap/replay, full race/static/vulnerability/release smoke | Automated gate complete; external cutover/release gates pending |

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
  unit-only Go suite passes after the dependency upgrade.
- Live scanning moved the toolchain baseline beyond the audit minimum: Go
  1.26.5 still had seven reachable standard-library findings in the current
  vulnerability database, so the implementation now uses Go 1.26.6. Source
  `govulncheck` 1.7.0 is clean after also upgrading pgx, gRPC, `x/net`, and
  `x/crypto`. Binary scanning exposes a govulncheck all-symbol OpenPGP false
  positive even though neither `go list -deps` nor an unstripped binary contains
  an OpenPGP package/symbol. The release gate runs the pinned scanner binary
  directly so its documented exit status is preserved, admits only that proven
  exact-ID/dependency/symbol exception, and fails every other finding until the
  upstream binary scanner can represent package absence.
- Final evidence on 2026-08-20:
  `APPVIEW_CHECK_ARTIFACT_DIR=/private/tmp/craftsky-appview-final-gate-20260820-4
  ./scripts/appview-check` passes end to end with Go 1.26.6, Staticcheck 2026.1,
  govulncheck 1.7.0, module/format/vet/static-analysis checks, fresh PostgreSQL
  16 and MinIO non-race and race suites with required-test skip detection,
  exact release binaries, migration down-to-zero/up, container builds, and
  release-image health smoke. The gate also ran `just lexgen-check` and retained
  its drift log in the same artifact. A live
  source scan found reachable `GO-2026-6222` in `x/image` 0.44.0; the dependency
  was upgraded narrowly to 0.45.0 and the final source scan reports zero
  reachable vulnerabilities.
- Migration inspection executes populated, empty, missing, non-directory, and
  unreadable sources; `up`, `down`, `status`, and `redo` all reject an empty
  bundle before a database connection. The actual Compose failure path makes
  `migrate` exit nonzero for an empty bundle and proves AppView never starts.
- The exact AppView/CLI binary exception admits only `GO-2026-5932` while the
  source scan is clean and dependency/symbol evidence contains no OpenPGP.
  AppView maintainers own the Step-1 review; it expires on 2026-09-20 and the
  gate hard-fails after expiry or for any other finding.

### Step 2: `CFG-001`

- Complete: one typed OAuth deployment bundle derives every public endpoint
  from a canonical origin. Production rejects localhost, incomplete credentials,
  legacy callback/hostname variables, mutable Host-derived metadata, and
  malformed key material before opening the database. Discovery, JWKS, and
  metadata are immutable and no-store; `ExpectedHost` ignores forwarded-host
  spoofing and returns the canonical request-ID envelope.
- Complete: disabled/local/credentialed-remote development authentication is an
  explicit startup policy. Remote use requires the configured hashed secret;
  production rejects the development headers. Compose publishes AppView,
  PostgreSQL, and MinIO independently on loopback by default, validates explicit
  remote AppView exposure, and pins database/object-store images by digest.
- Complete: HTTP admission, OAuth/session/worker, Tap, push, scheduled-object,
  deletion, and image-decode budgets have bounded values plus their cross-field
  timing relationships. Scheduled image settings may only lower the compiled
  safe ceilings, and production rejects more than one AppView replica while the
  outer limiter remains process-local. Config/package tests and the complete
  release gate pass.
- Deployment proof remains external: provision the real secret values, exact
  immediate-proxy CIDRs, and protected transport; then prove PostgreSQL/MinIO
  and the development credential are unreachable from an untrusted network.

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
- Actual AppView composition now injects the same boundary into protected-
  resource and authorization-server discovery, PAR, callback/token exchange,
  pending onboarding, persisted-session resume, PDS JSON reads/writes, blob
  upload, anonymous profile backfill, and token revocation. Test-only resolver,
  dialer, and CA seams exercise the production constructors without weakening
  the production public-destination policy.
- Live loopback traps advertised as PAR, token, revocation, or PDS endpoints are
  rejected with typed/redacted categories before any trap or private base-dialer
  connection. Mixed DNS and DNS-rebinding cases also make zero forbidden
  connections. The full PostgreSQL-backed real flow proves purpose routing,
  finite caps/timeouts, pinned public dials, connection reuse, body closure, and
  listener-goroutine shutdown. Full `internal/federatedhttp`, `internal/auth`,
  and `internal/app` packages, focused index backfill, ten-repeat listener/flow
  tests, focused race tests, and vet all pass.
- External release smoke remains: repeat a consent/callback, ordinary PDS
  write, and revocation against an approved public-DNS/public-CA test service in
  the deployed egress/proxy environment and inspect the bounded metric labels.

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
  Those development values are retained for comparison; the exact release
  evidence below supersedes them for deployment sizing.
- Both upload routes now share one bounded permit acquired after authentication
  and current-member admission but before the first body read. It admits one
  active request and one bounded waiter, rejects further requests without
  reading their bodies, and remains held through decode and remote-write work.
- The final release-runtime cgroup proof used a `536,870,912`-byte limit and a
  `134,217,728`-byte safety margin. AppView baseline was `15,474,688` bytes.
  JPEG, PNG, and WebP peaks were `23,138,304`, `23,945,216`, and `78,979,072`
  bytes; their conservative AppView-baseline-plus-increment-plus-margin totals
  were `164,511,744`, `165,318,656`, and `220,352,512` bytes. The maximum-size
  admitted upload plus concurrent unread rejection peaked at `131,141,632`
  bytes and produced a conservative total of `272,515,072` bytes. All remain
  below 512 MiB. Only the authorized legacy private scheduled-row/object
  cleanup and queue/bucket convergence remain before deployment closure.

### Step 5: `LIFE-001`

- Complete foundation: migrations `000038_owner_auth_lifecycle` and
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
- Immediate profile updates and image-blob uploads now receive only a durable
  `pdseffects.ExecutorFactory`, never a raw PDS client. A narrow `ReadRecord`
  capability preserves the existing Bluesky profile and returns its
  authoritative CID while holding the same owner/session lifecycle boundary;
  it creates no mutation attempt. Profile writes read both `self` records before
  issuing separate serial durable puts, carry each prior CID as a conditional
  precondition, and report bounded partial results. Blob uploads persist the
  exact owner generation and content-derived fingerprint through the durable
  executor and do not grant the explicit-deletion worker blob authority.
- Focused evidence on 2026-08-20: the required-PostgreSQL
  `go test ./internal/pdseffects -count=1` and `go test -race
  ./internal/pdseffects -count=1` passed; focused profile/blob tests passed; the
  complete unit and race runs for `./internal/api` passed; the complete
  required-PostgreSQL API run passed after the coordinated scheduled/terminal
  integration settled; and `go vet ./internal/pdseffects ./internal/api
  ./internal/scheduledposts` passed.
- Required-PostgreSQL evidence on 2026-08-20 now covers migration `000048`,
  bounded departure/rejoin, the real durable guarded executor, the one-boundary
  `MaxConns=2` race, publication finalization under both locks, and the complete
  scheduled-post package against PostgreSQL and MinIO, including its race run.
  Focused terminal-purge checks for scheduled objects, push, moderation, and
  the schema inventory also pass. Startup composes the bounded scheduled and
  unresolved-PDS-attempt participants into active-to-non-active and terminal
  transitions. The aggregate migration `000046`/`000048` round trip and full
  repository release gate pass under `FINAL-001`.
- Transaction-scoped lifecycle guards now cover projector actor/target writes
  and the reviewed private mutation creators; unknown external targets remain
  fenced while terminal targets fail closed. Exact ordinary
  PDS-attempt/source reconciliation now closes LIFE/Tap provenance as recorded
  in the TAP section; it is no longer a snapshot-only membership predicate.

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
- No finite S3/MinIO server-side settlement guarantee was supplied or proven.
  Runtime wiring therefore configures no finite bound and outcome-uncertain
  exact-key work intentionally remains reconciling rather than falsely
  finalizing. Ordinary PDS-effect reconciliation and the atomic
  operation/OAuth/safety-row finalizer are implemented in the coordinated
  lifecycle/account-deletion lanes.

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
- A persisted endpoint that fails that boundary is no longer left active.
  Ordinary access atomically moves the exact parent version to
  `revocation_pending` and revokes every child; deletion-only access preserves
  accepted-operation authority while moving it to `reauth_required`. Both
  paths retry once on a concurrent row-version change so a corrected credential
  is reloaded rather than terminalized from stale data, and PostgreSQL tests
  prove the invalid endpoint reaches neither the operation callback nor the
  network transport.
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
- Final aggregate evidence: the complete required-PostgreSQL `internal/auth`,
  `internal/accountdeletion`, and owner-lifecycle suites pass in both normal and
  race modes; revocation, auxiliary cleanup, session expiry, callback, handoff,
  and coordinated PDS workers are runtime-wired; the full repository release
  gate and all 1,489 Flutter tests pass. Production verified links remain
  externally blocked on the canonical host, association-file publication,
  Android release signing fingerprint, Apple entitlement/provisioning setup,
  and physical-device link/login/deletion verification.

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
- `BodyLimit` now wraps response hydration/decorators, so
  `http.NewResponseController` reaches the real server writer and installs the
  deadline in the production `AddRoutes` chain. Every audited rejection that
  occurs before body ownership—including unsupported media, inner rate limits,
  membership failures, CORS short-circuits, and persistent Instagram limits—
  detaches the unread body and disables connection reuse. Real socket tests
  exercise both slow chunked input through the hydrator and an early 415.
- Validated timeout geometry now requires `HTTP_WRITE_TIMEOUT` to exceed the
  entire scheduled-upload path (`HTTP_UPLOAD_BODY_READ_TIMEOUT` plus
  `SCHEDULED_MEDIA_PUT_TIMEOUT`) and a fixed five-second response margin. The
  bounded write-timeout maximum is 20 minutes so all individually permitted
  maxima can form a coherent configuration; defaults are unchanged.
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
- Final aggregate evidence: the complete required-PostgreSQL/MinIO normal and
  race suites, route-policy inventory, real-listener admission tests, vet,
  Staticcheck, container build, and startup health smoke pass. Production must
  remain single-replica until a verified shared edge limiter is supplied,
  configure only the actual immediate-proxy CIDRs, and complete the deployed
  browser/reverse-proxy smoke.

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
- Migration `000050_pds_effect_source_reconciliation` links a Tap source to at
  most one normalized ordinary Put-record attempt using exact owner, URI,
  action, content fingerprint, result CID, and mutation ordering. Ingestion
  persists the operation ID, originating generation, and durable disposition;
  projection re-locks that attempt and cannot exchange provenance across a
  duplicate or A-to-B-to-A history.
- A Put accepted before an AppView crash is non-repeatable and hidden after
  departure. The existing leased `pds_reconcile` worker performs bounded,
  anonymous, read-only PDS observations; only a unique exact still-current
  record may become eligible on legitimate same-DID rejoin. Missing,
  rejected, mismatch, terminal, or indistinguishably duplicated attempts stay
  hidden/retryable and never trigger a PDS mutation. Departure and terminal
  job-enqueue participants commit in the same owner transition transaction.
- Exact original wire-byte accounting admits every source at or below the
  one-megabyte boundary. Migration `000045` stores source records as preserving
  PostgreSQL `JSON` and enforces `octet_length(record::text) = record_bytes`, so
  compact exponent values are not expanded by JSONB normalization and valid
  escaped-NUL JSON is not rejected before durable classification. Boundary,
  exponent-expansion, and escaped-NUL integration tests cover those cases. A
  separate integration test commits source/job/receipt, drops the WebSocket
  before ACK, redelivers after reconnect, and proves one durable row of each
  kind. Another consumer test fails seven retryable deliveries and ACKs exactly
  once only after the eighth attempt commits.
- Migration `000051_tap_quarantine_replay_payload` separates the bounded
  JSONB diagnostic/listing envelope from a private exact-byte `BYTEA` replay
  payload constrained to Tap's shared 2 MiB frame ceiling. The exact bytes are
  committed before an outcome becomes ACK-able, never selected by operator
  list queries, and used only by a leased replay claim. Oversized valid,
  oversized invalid, and malformed JSON frames survive restart and replay byte
  for byte; duplicate redelivery idempotently backfills a legacy null payload.
  Operator replay fails closed for a pre-`000051` row that has not been healed
  by exact frame redelivery, because bounded JSONB evidence cannot reconstruct
  wire framing.
- Operator CLI surfaces now list projection/repository backlog and bounded
  quarantine evidence, queue selected quarantine replay, and enqueue read-only
  DID reconciliation. Quarantine and repository claims retain lease/restart
  APIs used by the runtime workers.
- Focused PostgreSQL evidence on 2026-08-14: database-required `go test -race
  ./internal/ingestion ./internal/index ./internal/tap ./cmd/cli -count=1`
  passed; the migration `000045` up/down/up test passed separately; `go vet`
  passed for the same four packages; `internal/tap` passed five consecutive
  loopback-WebSocket runs; `gofmt -l` and focused whitespace checks are clean.
- Additional required-PostgreSQL evidence on 2026-08-20 covers migration
  `000050`, crash/departure/rejoin, terminal and proved-not-accepted denial,
  CID/content mismatch, onboarding linkage, repository leases, and ambiguous
  A-to-B-to-A histories. Full owner-lifecycle/PDS-effect suites, non-database
  ingestion suites, `go vet`, repository-wide compile-only tests, and
  whitespace checks pass. Startup now constructs `ingestion.Service`, starts
  projection/repository/quarantine workers, uses the reduced pre-ACK operation
  deadline, and has no `TAP_MAX_RETRIES` surface.
- The final quarantine regression ran the complete required-PostgreSQL race
  suites for `internal/ingestion`, `internal/tap`, and `internal/db`; migration
  `000051` passed up/down/up; vet and pinned Staticcheck 2026.1 passed for all
  three packages. A real WebSocket test proves the exact replay bytes commit
  before ACK.
- Still open: execute the authorized controlled non-production cutover. It
  must snapshot and stop ingestion, apply migrations, reset public
  source/projections together
  with any Tap event-ID reset, re-register/rebootstrap repositories despite
  `TAP_NO_REPLAY`, drain/review blocked and quarantined work, then resume. It
  must not fabricate events already ACKed and lost or write to a PDS. Any
  pre-`000051` quarantine rows with a null exact payload must be reset/re-ingested
  or healed by the same frame's redelivery before operator replay.
- The full required-PostgreSQL ingestion package now passes. Its final cleanup
  split a multi-command prepared lock-order fixture, derived the terminal
  component expectation from the authoritative catalogue, and narrowed
  projection-quarantine replay decoding to URI/event/fingerprint identity so
  an intentionally informal CID cannot block requeue of an already-persisted
  source. The complete race and repository release gates now pass under
  `FINAL-001`; only the controlled Tap reset/rebootstrap/drain operation remains.

### Step 10: `PGX-001`

- Test both search branches and prefix-then-terminal-error row iterators.
- Ensure no dependent mutation happens before `rows.Err()` is checked.
- Complete: both search branches return their query-acquisition error without
  using invalid rows; notification-subscription and push-claim iterators reject
  a partial prefix on terminal error before any dependent mutation. The
  transactional notification-preference patch path also checks the terminal
  iterator error before resolving preferences, writing dependent rows, or
  committing. Focused `go test -race ./internal/api ./internal/push -count=1`
  passes.

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
- Provider construction remains typed as a unique event. Following the
  superseding 2026-08-20 product decision, Android and iOS both receive a
  bounded notification object plus the existing routing data. Android retains
  TTL and no explicit collapse key; APNs retains default sound and omits
  `apns-collapse-id`. FCM may collapse an undelivered Android notification.
- Flutter no longer presents background notifications itself. Android/iOS own
  background and terminated display, while the adapter maps foreground receipt,
  `onMessageOpenedApp`, and `getInitialMessage` into the provider-neutral
  runtime. Equal foreground callbacks are handled normally without persisted
  receipt suppression. Authenticated routing validation remains unchanged.
- Removed `flutter_local_notifications`, the Android background isolate,
  local gateway/presenter, presentation eligibility layer, SQLite stage cache,
  account-partition cleanup, plugin-only desugaring, and their tests. The
  existing monochrome drawable is now configured as Firebase's default
  provider-rendered notification icon. `sqflite` is test-only again.
- Focused evidence on 2026-08-14: database-required `go test
  ./internal/push -count=1`, database-required `go test -race
  ./internal/push -count=1`, and `go vet ./internal/push` pass; `flutter test
  test/notifications`, `dart analyze lib/notifications test/notifications
  test/bootstrap/firebase_bootstrap_test.dart`, and `flutter build apk --debug
  --no-pub` pass. The developer-machine Staticcheck binary was initially too
  old for Go 1.26; the pinned Go-1.26-compatible Staticcheck 2026.1 gate later
  passed across the repository under `FINAL-001`.
- Lifecycle and observability closure completed on 2026-08-20. Startup already
  passes the validated concurrency/finalization geometry and the real lifecycle
  store into the dispatcher. Required-PostgreSQL barriers prove recipient and
  actor transitions wait through provider send and token-fenced finalization,
  inverse actor/recipient pairs acquire fences without deadlock, recipient-only
  delivery never manufactures an empty actor key, and terminal or stale-
  generation state denies the callback. The worker's fresh fenced recheck means
  no new provider call starts after a completed transition.
- The same token-fenced pre-send query now reloads the current category
  preference and, for `peopleIFollow`, the current indexed follow edge. Real
  PostgreSQL claim-then-change barriers prove category disablement, scope
  narrowing, and unfollow all produce zero provider calls, while a still-current
  followed actor remains deliverable. The complete push package race suite
  passes with these final policy gates.
- Eligibility cancellation now applies the same active-subscription,
  installation-ID, and unchanged-token compare-and-set guard as delivery. A
  deterministic required-PostgreSQL regression rotates the FCM token between
  the eligibility status read and cancellation and proves the old lease is
  observed as stale rather than cancelling the newly routed delivery.
- Privacy-safe, closed-label observations now cover reclaimed-lease counts,
  insufficient processing windows, and lease/send/finalization latency and
  outcomes with platform plus unique-event/replacement semantics. Provider
  success followed by a database error or stale finalization is explicitly
  classified as `accepted_unfinalized`; no token, DID, message content,
  provider error, notification ID, delivery ID, or arbitrary replacement value
  becomes a label or log field.
- No client presentation revalidation or network revalidation endpoint is
  retained. A provider-accepted Android or iOS message may arrive after an
  AppView process crash releases its fences and a lifecycle transition or
  logout completes; the OS-rendered alert cannot be recalled. Ambiguous retries
  may display twice, and Android may collapse undelivered notification messages.
  Notification-tap routing still validates the retained account and authorized
  destination.
- Earlier focused evidence on 2026-08-20: required-PostgreSQL `go test` and `go test
  -race` pass across `./internal/push ./internal/app ./internal/observability`;
  focused `go vet` passes for the same packages. Superseding client evidence is
  recorded in the Flutter notification implementation plan. The remaining
  release-only gate is physical Firebase delivery on both platforms across
  foreground, background, terminated, tap, delayed, and ambiguous-retry cases.

### Step 13: `MOD-001`

- Migration `000044_moderation_restoration_outbox` installs the live intent,
  DID-free history, and bounded idempotency-receipt schema, including indexed
  retention and `ON DELETE SET NULL` reconciliation-job relationships. Output,
  outbox, and idempotency receipt insertion is transactional; the relay can
  create only strict private-suggestion reconciliation work and has no PDS-write
  capability.
- Source-owner terminalization now promotes and preserves qualifying restoration
  for the target owner, while target terminalization cancels it. Both paths take
  canonical source/target owner fences and lifecycle row locks before the
  outbox/job transition. Processing work remains untouched and retryable.
- Terminal and accepted-deletion cleanup share the same role-aware, bounded
  order: write DID-free history, remove the live outbox, delete the receipt by
  output ID, then delete the restrictive parent. This closes the prior
  `ON DELETE RESTRICT` failure for qualifying account-deletion rows.
- Live processed/cancelled DID-bearing outbox state is retained for 30 days;
  DID-free restoration history is retained for 365 days. Exact-boundary tests
  use a controlled clock, skip pending/processing work, and cover both source-
  first and target-first transition barriers.
- Real-PostgreSQL focused lifecycle/account-deletion tests, exact retention and
  migration round-trip tests, deterministic fence barriers repeated five times,
  focused race suites across owner lifecycle/account deletion/Instagram, and
  focused `go vet` all pass. The aggregate all-package normal/race and release
  gates also pass under `FINAL-001`.

### Step 14: `ARCH-001`

- Complete: `routes.AddRoutes` is now a composition-only registrar. Public/auth,
  profile/notification, search/migration, and scheduled/post route groups receive
  named capability bundles owned by `internal/routes`; the sole app-to-route
  adapter lives in `internal/app`. Executable architecture and inventory tests
  prohibit production route imports of `internal/app`, direct registration in
  the composer, aggregate dependencies in registrars, and route/policy drift.
- Complete: post handlers and stores are split by create, read, conversation,
  interactions, author-feed, response-shape, lookup, relationship, engagement,
  and interaction capabilities. Search handlers consume narrow profile,
  hashtag, post, project, suggestion, and recent-search interfaces; its profile,
  hashtag, post, and project SQL lives in focused stores. Scheduled draft/resource,
  publication-state, and publication-effect stores expose only the operations
  consumed by their API/service/worker callers. Characterization tests compare
  the moved declarations and preserve SQL, ranking, cursors, envelopes,
  transaction boundaries, and AV-026 acquisition-error behavior.
- Complete: `newDeps` delegates feature construction to named foundation,
  observability, owner/auth/PDS, content, Instagram, scheduled, account-deletion,
  push, Tap, admission, and worker constructors. Feature constructors return
  small bundles and register owned cleanup with the root's reverse-order,
  exactly-once cleanup stack; static tests reject direct feature construction in
  `newDeps` and direct shared-pool closure below the root. The runtime `Deps`
  surface exposes durable/guarded PDS effects rather than a raw reusable PDS
  client factory.
- Complete: Tap production wiring now exposes and registers only the
  transactional projector. The obsolete nontransactional dispatcher facade and
  its isolated tests, legacy `Deps.Indexer` forwarding, ignored terminal-purge
  component forwarding, and the unreachable one-shot profile-backfiller
  injection are removed. Profile projection uses an explicitly no-remote-work
  constructor; durable repository tracking/backfill remains owned by the
  repository-job worker.
- Documentation in `appview/README.md` maps route, post, search, scheduled, and
  dependency-construction ownership and the extension rule. Required-PostgreSQL
  API/routes/scheduled tests, their race runs, app/cmd compile tests, focused
  architecture tests, `go vet`, formatting, and whitespace checks pass. The
  full required-PostgreSQL and race suites are recorded under `FINAL-001`.
- Final AV-037 cleanup evidence on 2026-08-20: required-PostgreSQL normal and
  race suites pass for `./internal/index ./internal/app ./internal/ingestion`;
  `go vet` passes for the same packages; and pinned Staticcheck 2026.1 reports
  no findings.

### Step 15: `FINAL-001`

- Complete automated evidence on 2026-08-20: `./scripts/appview-check` rebuilt
  disposable PostgreSQL 16/MinIO state; ran every required-database package in
  normal and race modes with skip detection; verified modules, formatting, vet,
  Staticcheck 2026.1, and a zero-reachable-vulnerability source scan; built the
  exact AppView/CLI release artifacts; applied the narrow, evidence-backed
  binary-scanner exception; migrated up/down-to-zero/up; and passed the release
  container health smoke. The authoritative artifact is
  `/private/tmp/craftsky-appview-final-gate-20260820-4`; its tool record names
  baseline commit `2b5d5efa7fee4fa803a2ed64e6a684efb8abad92` and correctly marks
  the tested repository state dirty. Its release-container memory proof records
  a 512 MiB cgroup limit, 128 MiB safety margin, 15,474,688-byte AppView
  baseline, and maximum budget totals of 164,511,744 bytes for JPEG,
  165,318,656 bytes for PNG, 220,352,512 bytes for WebP, and 272,515,072 bytes
  for the maximum admitted concurrent-upload probe. The separate full Flutter
  suite passed 1,489 tests and `dart analyze` reported no issues.
- External/destructive gates remain deliberately unexecuted: production
  association files/signing/provisioning and physical-device link/push checks;
  trusted-proxy/protected-transport/LAN reachability smoke; the public-DNS/
  public-CA federated-flow smoke; authorized legacy scheduled-media cleanup;
  the retained-environment moderation dry-run and destructive reset before
  migration `000044`; and the controlled Tap snapshot, reset/rebootstrap,
  replay, queue drain, and quarantine review. These require deployment
  credentials, infrastructure, devices, or explicit destructive-cutover
  authorization rather than additional application code. A clean committed
  tree must rerun the complete CI/release gate before release.

## Completion checklist

- [x] Every AV-001 through AV-037 acceptance criterion has executed evidence or
  an explicitly recorded external release gate.
- [x] Every schema migration passes PostgreSQL 16 up/down/up.
- [x] Full tests report zero skipped real-PostgreSQL cases.
- [x] PostgreSQL/MinIO race suite passes.
- [x] Flutter unit/widget/platform contract tests pass.
- [x] `gofmt -l`, `go vet`, pinned Staticcheck, module drift, and
  `just lexgen-check` pass inside `appview-check`.
- [x] Source and exact built AppView/CLI binaries pass pinned `govulncheck` or
  the exact evidence-backed all-symbol false-positive exception.
- [x] Clean disposable migration and release-container startup smoke pass.
- [x] Exact release-runtime media memory and maximum-admitted concurrent-upload
  proof pass under the 512 MiB cgroup limit.
- [ ] Legacy private scheduled-media cleanup awaits authorized cutover and
  queue/bucket convergence verification.
- [ ] Retained-environment moderation legacy dry-run/reset awaits explicit
  authorization before migration `000044`.
- [ ] Tap reset/rebootstrap, replay/reindex, and worker/quarantine drain await
  the authorized controlled cutover.
- [ ] The committed clean tree reruns the complete CI/release gate.
- [x] No insecure compatibility path from the audited behavior remains.
- [x] Implementation review is complete with an `Approved with notes` verdict;
  the notes are the external/destructive and clean-commit release gates above.
