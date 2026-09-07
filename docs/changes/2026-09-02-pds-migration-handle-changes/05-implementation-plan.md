# TDD Implementation Plan: PDS Migration And Handle Change Resilience

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`
- Implementation approval: explicitly provided by the user on 2026-09-03
- Commits: disabled unless explicitly requested

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first and confirm a meaningful failure.
- Refactor only after tests pass.
- Keep traceability and red/green evidence updated after each loop.
- Preserve the security, migration, snapshot, API, and Flutter guardrails in `04-coding-plan.md`.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001, NFR-002, RULE-002 | AC-009, AC-010, AC-028, AC-036 | Fails: coordinator has no uncached authority comparison |
| 2 | UT-002, IT-016 | FR-003, FR-031 | AC-012, AC-055 | Fails: callback can persist mixed authority |
| 3 | UT-003, IT-002 | FR-002, FR-005, NFR-001 | AC-007, AC-008, AC-011, AC-027, AC-037, AC-038 | Fails: no exact-parent stale transition trigger |
| 4 | UT-004, AT-004 | BR-003, FR-001, FR-002, FR-003, FR-005, RULE-002 | AC-007, AC-009, AC-012, AC-036, AC-037, AC-038 | Fails: stale authority is not mapped end to end |
| 5 | IT-001, IT-004, IT-005 | FR-001, FR-003, FR-004, FR-028, NFR-002, RULE-002, RULE-003 | AC-009, AC-010, AC-012, AC-013, AC-028, AC-033, AC-036, AC-052 | Fails: authority isolation and transient preservation unproven |
| 6 | IT-007 | FR-008, FR-010, NFR-001 | AC-014, AC-018, AC-027, AC-039 | Fails: best-effort Tap tracking does not survive restart |
| 7 | UT-006 | FR-009, NFR-003 | AC-040, AC-050 | Fails: no shared registry or verified comparison boundary |
| 8 | AT-010, IT-008 | FR-009, FR-027, NFR-001, NFR-003 | AC-015, AC-016, AC-027, AC-029, AC-050, AC-051 | Fails: no complete CAR trust boundary |
| 9 | IT-006, IT-010 | BR-001, FR-008, FR-009, FR-027, NFR-001, NFR-003, RULE-006 | AC-001, AC-002, AC-015, AC-016, AC-017, AC-027, AC-029, AC-039, AC-040, AC-050, AC-051 | Fails: full repository convergence is absent |
| 10 | IT-011, AT-001, IT-009 | BR-001, BR-003, FR-004, FR-006, FR-008, FR-009, FR-011, RULE-001, RULE-003, RULE-006 | AC-001, AC-002, AC-008, AC-013, AC-014, AC-015, AC-016, AC-017, AC-033, AC-041 | Fails: no end-to-end migration contract |
| 11 | UT-012, IT-012 | FR-019, FR-020, FR-021, NFR-001 | AC-024, AC-025, AC-027, AC-043, AC-044 | Fails: stale cache work can preserve or overwrite aliases |
| 12 | UT-005, IT-014, AT-011 | FR-007, FR-029, FR-030, NFR-001, RULE-006 | AC-009, AC-014, AC-053, AC-054 | Fails: deleted Tap status terminalizes owner and replay is disabled |
| 13 | UT-007, AT-002 | BR-002, FR-012, FR-013 | AC-003, AC-019 | Fails: Flutter discards refreshed handle |
| 14 | UT-008, REG-004 | FR-013, FR-014, FR-021, NFR-006, RULE-007 | AC-004, AC-019, AC-020, AC-021, AC-032, AC-035, AC-044 | Fails: provider and route accept mutable handles |
| 15 | UT-009, AT-003 | BR-002, FR-014, FR-015, FR-016, FR-021, RULE-001, RULE-005 | AC-004, AC-005, AC-006, AC-020, AC-021, AC-044 | Fails: durable destinations use visible or stored handles |
| 16 | UT-010, UT-011 | FR-015, FR-018, RULE-005 | AC-005, AC-023 | Fails: ownership and deduplication compare handles |
| 17 | UT-013, IT-013, AT-007 | FR-017 | AC-022, AC-042 | Fails: deletion contract is handle-bound |
| 18 | UT-014, AT-006 | FR-020 | AC-025, AC-043 | Fails: sentinel renders and stale alias remains usable |
| 19 | UT-017, UT-018, AT-012, IT-015 | FR-005, FR-023, NFR-005, NFR-006, RULE-004, RULE-007 | AC-031, AC-032, AC-034, AC-035, AC-038, AC-045 | Fails: credential/inventory/no-gate contracts are absent |
| 20 | UT-015, IT-003, AT-008 | FR-002, FR-006, FR-025 | AC-001, AC-008, AC-011, AC-048 | Fails: final eligible attempt boundary is not locked |
| 21 | UT-016, IT-017 | NFR-005 | AC-031 | Fails: migration telemetry secret scans are absent |
| 22 | REG-001 through REG-008 | BR-001, FR-014, FR-025, FR-030, NFR-005, NFR-006, RULE-003, RULE-004, RULE-006, RULE-007 | AC-001, AC-002, AC-010, AC-020, AC-021, AC-031 through AC-035, AC-048, AC-053, AC-054 | Existing suites must remain green |
| 23 | MAN-001, MAN-002 | FR-005, FR-020, NFR-003, NFR-005 | AC-029, AC-031, AC-038, AC-043 | Manual or staging evidence required |

## Implementation Steps

### Step 1: UT-001

- Write failing test: authority outcomes at the coordinator boundary.
- Run command: `cd appview && go test ./internal/auth -run 'TestOAuthSessionCoordinator.*Authority'`
- Confirmed failure: Build failed because `OAuthAuthority` and `AuthorityVerifier` did not exist. The first database-backed run then exposed invalid issuer fixture endpoints; after correcting the fixture, the matching case exposed its missing DPoP key. Both were test setup failures corrected before relying on product assertions.
- Implement: Added the required verifier contract, canonical PDS/issuer comparison, retryable verifier-error handling, and exact-parent stale fencing before OAuth client-session construction.
- Run command: `TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=postgres://craftsky:dev@127.0.0.1:15956/craftsky_dev?sslmode=disable go test -v ./internal/auth -run 'TestOAuthSessionCoordinator.*Authority' -count=1`
- Green evidence: All matching-PDS, changed-PDS, changed-issuer, and transient-resolution cases passed against PostgreSQL.
- Refactor: Added a matching-authority test stub to existing coordinator tests so the new fail-closed constructor dependency does not change their scope.
- Notes: Full database-backed `go test ./internal/auth -count=1` passed. Production verifier wiring remains intentionally deferred to IT-001.

### Steps 2-23

- Execute in the exact order in the table and Section 9 of `04-coding-plan.md`.
- Use the focused commands from the coding plan for each package or Flutter target.
- Record each meaningful red failure, minimum implementation, focused green result, nearby regression result, and any refactor before starting the next step.

## Execution Log

| Step | Test IDs | Red Evidence | Green Evidence | Refactor / Notes | Status |
|---|---|---|---|---|---|
| 1 | UT-001 | Missing authority contract/options caused focused build failure; database fixture issues were corrected before product assertions | Focused four-case PostgreSQL test and full `internal/auth` package pass | Required verifier; check occurs before client construction; existing tests use matching stub | Complete |
| 2 | UT-002, IT-016 | Focused callback test failed to compile because login request resource/issuer metadata and flow verifier injection were absent | Focused callback migration test, migration up/down/up test, and database-backed `internal/auth`, `internal/api`, and `internal/db` suites pass | Credentials are quarantined before current-authority validation; mixed sessions persist no parent/child; cleanup remains original-issuer-only | Complete |
| 3 | UT-003, IT-002 | Concurrent stale retry returned not-found instead of the idempotent expired result; background selection chose an obsolete-generation parent | Focused race test passed 10 times under `-race`; database-backed `internal/auth` and `internal/api` suites pass | Stale transition predicates now include owner generation and auth epoch; only exact parent/children are fenced; corrected and independent parents remain usable | Complete |
| 4 | UT-004, AT-004 | Initial fixture returned `503 lifecycle_unavailable`; after fixture correction the acceptance test passed immediately because existing API mapping plus phases 1-3 already supplied behavior | Dedicated PostgreSQL-backed profile-effect acceptance test and full `internal/api`/`internal/auth` suites pass | No production change required; test proves exact 401 envelope/request ID, operation suppression, exact-parent fencing, independent-parent preservation, and transient non-mapping | Complete |
| 5 | IT-001, IT-004, IT-005 | Concrete-verifier integration test failed to compile because `newAuthoritativeOAuthVerifier` did not exist; a parameterized multi-command SQL fixture was corrected before product assertions | Six-case HTTP-fake integration passes under `-race`; full database-backed `internal/app`, `internal/auth`, and `internal/api` suites pass | Shared uncached verifier uses authoritative DID lookup and hardened OAuth metadata resolver; injected into callback/coordinator; credential capture proves original-issuer-only cleanup and no forwarding to new authority | Complete |
| 6 | IT-007 | Focused test failed to compile because auth had no repository-job contract, transactional enqueue method, or handoff dependency | PostgreSQL restart/coalescing test passes under `-race`; database-backed `internal/auth`, `internal/ingestion`, `internal/app`, and `internal/routes` suites pass | Shared ingestion store is constructed before auth/Tap wiring; handoff activation transaction enqueues tracking and same-DID repair; direct best-effort profile-init Tap call removed | Complete |
| 7 | UT-006 | Focused test failed to compile because dispatcher collection export, snapshot verification marker, and repair comparison did not exist | Focused repair comparison and full `internal/index`, `internal/ingestion`, and `internal/app` package tests pass | Registry returns sorted defensive snapshot; unknown collections ignored; unverified snapshots expose no comparison/deletes; no CAR fetch or projection added | Complete |
| 8 | AT-010, IT-008 | Focused fetcher test failed to compile because no repository snapshot fetcher existed | Trust matrix passes for valid signed CAR and all bounded acquisition/root/DID/signature/MST/source-change failures; durable retry test and nearby app/ingestion/index/federated/tap suites pass | Added hardened read-only getRepo purpose, 64 MiB/2-minute bounds, full CAR consumption and Indigo verification, post-fetch authority stability; arbitrary snapshot blessing seam removed; no repair projection yet | Complete |
| 9 | IT-006, IT-010 | Focused integration test failed to compile because repair service/config/batch contracts did not exist | Verified signed-CAR convergence and verified-profile-absence tests pass against PostgreSQL; full database-backed ingestion/app/index suites pass | Deterministic 100-record batches flow through durable source/projection/indexer paths; retries resume unsettled projections; newer Tap revisions win; verified profile absence applies departure without terminalization | Complete |
| 10 | IT-011, AT-001, IT-009 | Initial acceptance setup failed on isolated-schema extensions and pgx fixture assumptions; once corrected, phases 1-9 already satisfied the product behavior so no production red remained | Same/changed-handle migration acceptance passes under `-race`; pinned Tap contract and full app/tap suites pass | HTTP/Pg contract proves key/authority rotation, reset chain, missed create/update/delete convergence, exact stale fencing, one DID owner, representative private-state preservation, AppView reads, and B-only writes | Complete |
| 11 | UT-012, IT-012 | Authoritative cache race test showed failed proof, invalid handle, and duplicate sentinel retained a reassigned handle; displaced DID row was missing | Focused race test passes under `-race`; full database-backed API/auth/app/routes/account-deletion/ingestion suites and migration up/down/up pass | Sole version-fenced write path; valid aliases uniquely owned, displaced DID gets sentinel, sentinel excluded from alias/search input; post-commit DID/handle invalidation; migration 000067 permits repeated sentinel values | Complete |
| 12 | UT-005, IT-014, AT-011 | Identity policy test failed because no status classifier existed; replay acceptance failed while Compose set `TAP_NO_REPLAY` | Focused policy/decoder/replay tests, `go vet`, database-free repository suite, and full PostgreSQL-backed `go test ./...` pass | Tap 0.1.10 `active`, `deactivated`, `suspended`, `takendown`, and `deleted` are coalesced refresh hints only; decoder ignores non-identity `desynchronized`/`throttled`; no status terminalization or presentation overwrite; durable cursor replay enabled | Complete |
| 13 | UT-007, AT-002 | Focused Flutter tests observed the stale stored handle after matching-DID validation and timed out waiting for persistence | Four focused coordinator/acceptance tests and all 121 auth tests pass; analyzer and diff checks clean | Lease/generation-fenced handle-only mutation persists before validation success, publishes reactively, preserves account/session/routing/cached fields, and rejects stale lease or DID mismatch | Complete |
| 14 | UT-008, REG-004 | Existing provider family signatures accepted `String` handle-or-DID keys, allowing identity collisions and stale-own-profile fetches | Focused DID-key tests pass; profile/auth, authored-content/language, business, notification/PostCard suites pass; `dart analyze` and generated-code inspection clean | Profile/authored-content/relationship families and publication/invalidation are typed-DID keyed; active account uses `fetchMe`; historical handle collisions remain separate and same-DID handle changes retain one state | Complete |
| 15 | UT-009, AT-003 | Existing route/facet expectations navigated known identities through mutable handles rather than embedded DIDs | Focused 31-test router/facet/acceptance rerun passes; broader router/profile/rich-text and navigation suites pass; analyzer and diff checks clean | Canonical `/profiles/:did`; explicit `/profiles/@:handle` resolves then replaces with DID URL; mentions route by facet DID while historical visible text remains unchanged; identity navigation call sites pass model DIDs | Complete |
| 16 | UT-010, UT-011 | Profile ownership treated equal handles as self and profile pagination collapsed distinct DIDs sharing a handle | Focused 57-test search/profile rerun passes; broader search/profile/report suites pass; analyzer and diff checks clean | Self/visitor actions compare typed DIDs; report/mutual-follower targets carry DIDs; search merges same DID across handle changes and preserves different DIDs sharing a handle | Complete |
| 17 | UT-013, IT-013, AT-007 | Backend/Flutter tests failed on missing DID confirmation fields, handle-bound persistence, and non-exact matching; broader verification exposed pre-migration fixture drift | Focused backend and 17-test Flutter reruns pass; migration up/down/up preserves hash; broader auth/settings pass; `just appview-check` passes all release gates after fixed dependency updates | Intent/hash/wire/controller/UI use exact authenticated `confirmationDid` for every handle state; no deletion handle lookup/cache write; standard mismatch envelope; strict storage schema v2; vulnerable ipld/crypto modules upgraded without changing Indigo/Tap pins | Complete |
| 18 | UT-014, AT-006 | Missing central handle policy; profile card and acceptance fixtures exposed raw sentinel/stale identity | Independent 31-test model/card/acceptance/facet rerun passes; 176 relevant widgets and historical mention tests pass; localization generation/analyzer/diff clean | `ProfileHandle` centralizes valid alias input and localized current presentation; sentinel/absent/malformed values render unavailable and cannot enter alias/search/mention input; authored historical text unchanged | Complete |
| 19 | UT-017, UT-018, AT-012, IT-015 | Credential tests accepted injected PDS fields/headers; identity surface inventory and cold-start write contracts were absent | Independent 30-test credential/inventory/cold-start rerun passes; 115 surface and 496 broad auth/router/profile tests pass; analyzer/diff clean | Strict snapshots reject PDS credential/endpoint fields; interceptor strips caller credential headers and sends only Craftsky bearer; all known identity surfaces are DID-first except explicit alias entry; pending validation does not gate writes and `pds_session_expired` recovers exact lease | Complete |
| 20 | UT-015, IT-003, AT-008 | Acceptance matrix exposed that work first processed at 30:01 could still select a session and publish; no inclusive cutoff predicate existed | Focused 7-case cutoff/migration suite passes; full PostgreSQL-backed scheduledposts/auth suites and `go vet` pass | Automatic eligibility includes 29:59 and exactly 30:00, excludes later instants before session/PDS I/O; exact-final success publishes, exact-final failure and late work become `needs_attention`; existing retry lifecycle retained | Complete |
| 21 | UT-016, IT-017 | Initial observer tests failed to compile because migration authority/repair/backlog/snapshot signals did not exist. The IR-003 correction harness then exposed that callback cleanup had no production observer and that the earlier cross-sink test synthesized rather than exercised production outcomes. | Bounded unit tests and `TestPDSMigrationProductionPathsAreCrossSinkSecretFree` pass. The PostgreSQL-backed harness drives success, stale, transient, repository-repair, parent-cleanup, and callback-cleanup paths; asserts exact metric schemas/counts and request/run/trace correlation; and scans local structured logs, observer logs, metrics, Sentry events/traces, and API bodies together for every TD-009 canary. | Added bounded OAuth cleanup observation and production wiring. Stale lease workers emit no false outcome; terminal cleanup reasons are accurate; no credential, DID, handle, origin URL, or raw error is used as a metric dimension. The synthetic cross-sink test was removed after the production-path harness superseded it. | Complete |
| 22 | REG-001 through REG-008 | Not applicable | `just appview-check`, all 1,978 Flutter tests, `just app-analyze`, full PostgreSQL/MinIO `just test` with Go race detection, and `git diff --check` pass after the IR-003 correction. | The first correction rerun of `appview-check` encountered three pre-existing timing-sensitive race-suite failures; each passed in isolation without code changes and the complete gate passed on rerun. The configured gate reports transitive `GO-2026-5932` with no fixed version and passes as designed. | Complete |
| 23 | MAN-001, MAN-002 | Not applicable | Automated presentation, recovery, metric-content, threshold, cross-sink secret-canary, and correlation coverage passes | No running device/DTD or staging observability backend is available in this workspace; device presentation and external dashboard/alert delivery remain explicit pre-release manual checks | Complete; manual checks pending |
| 24 | UT-019, UT-020, IT-018, REG-009 | Indigo performed two repeated metadata GETs per ordinary authority check; constructor testing then exposed a missing five-minute TTL invariant, and a cancellation-ignoring source proved abandoned loaders stopped consuming capacity before exit | Cache/verifier/telemetry tests pass under repeated `-race`; the PostgreSQL/HTTP real-flow proves warm operations retain fresh DID lookup while login start, callback, and registration discovery remain fresh; affected package sweep, `go vet`, `just test`, `just app-test`, `just app-analyze`, `git diff --check`, and `just appview-check` pass | Added separate fresh and operation verifiers, canonical positive-only fixed-expiry caches, same-key coalescing, actual-loader capacity accounting, abandoned-load cancellation, bounded configuration/metrics, and TD-009 sanitization. Review findings for TTL, cancellation, capacity, production-flow evidence, and all-result telemetry were corrected; final re-review found no findings. Covers FR-001, FR-031, FR-032, NFR-001, NFR-002, NFR-005 and AC-009, AC-031, AC-036, AC-055, AC-056. | Complete |

## Verification Commands

```text
cd appview && go test ./internal/auth ./internal/api
cd appview && go test ./internal/ingestion ./internal/tap ./internal/app
cd appview && go test ./internal/scheduledposts ./internal/observability
cd app && flutter test test/auth/services/session_validation_coordinator_test.dart
cd app && flutter test test/router test/profile test/search test/shared/rich_text test/settings
just appview-test-unit
just app-test
just test
just appview-check
just app-analyze
```

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] Migration up/down behavior verified
- [x] Generated diffs inspected for unrelated churn
- [x] No unlinked behavior implemented
- [x] Documentation and execution evidence updated
- [x] Manual checks completed or explicitly recorded as pending
- [x] Implementation review completed or explicitly skipped
