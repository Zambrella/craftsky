# TDD Implementation Plan: AppView Push Notifications

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`
- Implementation approval: explicitly granted by the user on 2026-07-11

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first and refactor only while green.
- Keep source mutation and notification lifecycle changes in the same transaction.
- Never contact FCM from automated tests or a Tap/indexer transaction.
- Keep traceability and executed commands updated below.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | UT-001 | BR-003, RULE-005, RULE-006 | AC-003 | Fails |
| 2 | UT-002 | FR-007, RULE-002 | AC-011, AC-012 | Fails |
| 3 | UT-003 | FR-008, RULE-001, RULE-002 | AC-013, AC-037 | Fails |
| 4 | IT-028 | NFR-005 | AC-030 | Fails |
| 5 | IT-004 like create | FR-003, FR-021 | AC-005 | Fails |
| 6 | IT-001, IT-002, IT-003 like slice | BR-001, BR-002, FR-001, FR-002, FR-004, FR-011 | AC-001, AC-002, AC-004 | Fails |
| 7 | IT-004 like delete, IT-005 like | FR-003, FR-004, FR-021 | AC-005, AC-010 | Fails |
| 8 | IT-020 like | FR-001, FR-011, FR-018 | AC-035 | Fails |
| 9 | AT-002, AT-003 | FR-017, FR-020, RULE-006 | AC-006, AC-007 | Fails |
| 10 | IT-010 follow, IT-004 follow | FR-003, FR-008, RULE-001, RULE-003 | AC-005, AC-013, AC-037 | Fails |
| 11 | UT-004, AT-004, IT-004 post | FR-003, FR-019, FR-020 | AC-005, AC-008 | Fails |
| 12 | IT-005, IT-018 all producers | FR-021 | AC-010, AC-033 | Fails |
| 13 | IT-006, UT-005, REG-001 | BR-001, FR-004, FR-005, FR-022 | AC-009, AC-022 | Fails |
| 14 | IT-019, REG-003 | FR-022, FR-023 | AC-022, AC-023, AC-024 | Fails |
| 15 | IT-017, IT-024 | FR-021, FR-023, FR-026 | AC-032, AC-034 | Fails |
| 16 | IT-007, AT-007 preferences | FR-006, FR-007, NFR-003, RULE-002 | AC-003, AC-011, AC-012, AC-028 | Fails |
| 17 | IT-008, IT-012 | FR-009, FR-010 | AC-014, AC-018 | Fails |
| 18 | IT-021, IT-031, AT-008, AT-009 | FR-009, FR-010, FR-016, FR-033 | AC-038, AC-044 | Fails |
| 19 | IT-011, REG-008 | FR-009, FR-016 | AC-017 | Fails |
| 20 | UT-006 | FR-013, FR-028 | AC-019, AC-036 | Fails |
| 21 | UT-007 | FR-024 | AC-025 | Fails |
| 22 | IT-013 through IT-016, REG-007 | FR-012 through FR-015, FR-028, RULE-004 | AC-019, AC-020, AC-021, AC-036 | Fails |
| 23 | IT-023, IT-025, IT-026 | FR-024, FR-027, FR-029, FR-031 | AC-026, AC-040, AC-041, AC-042 | Fails |
| 24 | IT-030 | FR-012, RULE-004 | AC-043 | Fails |
| 25 | IT-022 | FR-023, FR-030 | AC-039 | Fails |
| 26 | UT-008, IT-027, IT-029 | NFR-002, NFR-006 | AC-027, AC-031 | Fails |
| 27 | UT-009, REG-004 | NFR-004 | AC-029 | Fails |
| 28 | REG-005, REG-006 | FR-025, FR-032 | AC-026, AC-043 | Fails |
| 29 | MAN-001, MAN-002 | BR-002, FR-024 | AC-002, AC-025 | Manual/provider environment required |

## Implementation Steps

Each step follows: write one focused test, record the meaningful red result, implement the minimum behavior, rerun focused and nearby tests, refactor while green, and record commands/results here.

### Step 1: UT-001
- Write failing test: exact approved values, invalid via-repost values, reserved producer, immutable registry
- Run command: `cd appview && go test ./internal/notifications`
- Confirmed failure: package symbols were absent (`undefined: Category`, `Like`, and registry functions)
- Implement: added the closed `Category` model, registry copy, validation, and producer predicate
- Refactor: formatted the package; kept the public surface minimal
- Notes: Green with focused package test. Initial sandbox cache denial was environmental; approved rerun passed.

### Step 2: UT-002
- Write failing test: defaults, partial merge, and invalid category/scope rejection
- Run command: `cd appview && go test ./internal/notifications`
- Confirmed failure: preference types and resolver were absent
- Implement: effective seven-category defaults and atomic partial resolution
- Refactor: shared the same policy between ingestion and HTTP persistence
- Notes: Green; effective defaults and all-or-nothing validation are exercised through pure and HTTP tests.

## Execution Log

| Step | Status | Red evidence | Green evidence | Notes |
|---:|---|---|---|---|
| 1 | Complete | Missing category package symbols | `go test ./internal/notifications` passed | Exact seven-category registry |
| 2 | Complete | Missing preference model and resolver symbols | `go test ./internal/notifications` passed | Defaults and atomic partial validation |
| 3 | Complete | Missing eligibility model and evaluator symbols | `go test ./internal/notifications` passed | Scope and push decisions separated |
| 4 | Complete | Migration file absent, then explicit constraint-name failure | Focused `internal/db` migration test passed | Five private tables; zero backfill |
| 5 | Complete | Lifecycle types/injection absent | Focused rollback and nearby interaction tests passed | Shared transaction proven |
| 6 | Complete | Durable service constructor absent | Focused ingestion and rollback tests passed | Stable row, two-way fan-out, replay idempotency |
| 7 | Complete | Delete left notification active and delivery pending | Focused lifecycle and nearby deletion tests passed | Atomic tombstone and cancellation |
| 8 | Complete | Recreated relationship remained retracted on old source URI | Focused like reactivation suite passed | Stable ID, no second push |
| 9 | Complete | Initially green after shared direct-author seam | Focused like/repost attribution suite passed | No self/reposter attribution |
| 10 | Complete | Follow constructor lacked lifecycle and transactional notification path | Focused follow and existing indexer suites passed | Event-time mutual-follow scope |
| 11 | Complete | Missing post classifier and canonical producer path | Policy and post precedence suites passed | Reply over quote over mention |
| 12 | Complete | Post deletion left notification active | Post lifecycle and deletion regressions passed | All current producer deletes transactional |
| 13 | Complete | Feed still used derived union; durable table absent from old fixtures | Durable pagination test and full API suite passed | Stable IDs, active rows only; obsolete derived tests replaced |
| 14 | Complete | Durable reads initially omitted explicit unavailable metadata | Durable store/takedown and API suites passed | Rows remain resolvable while hidden content is withheld |
| 15 | Complete | Resolution store and owner-only target model absent | Owner/tombstone/cross-owner and route suites passed | Category-specific precise/fallback targets |
| 16 | Complete | Preference handlers/routes/store absent | Focused defaults/partial-patch test and API/route suites passed | Seven effective values; atomic validation |
| 17 | Complete | Device registration contract absent | Focused idempotency/token-rotation test passed | Opaque routing ID; token not echoed |
| 18 | Complete | Account-scoped removal handler absent; cross-device case confirmed existing rebind transaction | Focused shared-device/rebind/removal tests passed | No routing/account transfer |
| 19 | Complete | Logout had no notification cleanup dependency | Auth/API/route suites passed | Cleanup runs first and fails closed |
| 20 | Complete | Retry/TTL policy absent | `go test ./internal/push` passed | 15-minute cap and absolute deadline |
| 21 | Complete | Minimal payload builder absent | `go test ./internal/push` passed | Generic copy and allowlisted routing data |
| 22 | Complete | Sender/claim/finalize loop absent | Dispatcher success/retry/expiry/invalid-token tests passed | Fake sender only in automation |
| 23 | Complete | Prospective toggle and routing edges unproved | Ingestion/device/dispatcher suites passed | No backfill or server aggregation |
| 24 | Complete | Concurrent claim and recovery absent | Two-worker and expired-lease tests passed | `SKIP LOCKED`; documented at-least-once crash window |
| 25 | Complete | Tap identity deletion was dropped | Actor deletion and Tap/app wiring suites passed | Only terminal `deleted` hard-deletes caused notifications |
| 26 | Complete | Push metrics/signals absent | Observability suite passed | Bounded labels; no token/payload/DID parameters |
| 27 | Complete | Push config, Firebase adapter, and worker lifecycle absent | Config, Firebase payload, app, command, and full suites passed | ADC; disabled mode creates no sender/worker |
| 28 | Complete | Structural guards absent | Migration and full suite passed | Zero backfill; no notification block/mute model |
| 29 | Pending manual | Requires non-production Firebase credentials and physical Android/iOS devices | Automated Firebase-message construction and fake sender tests passed | MAN-001/MAN-002 must run before production enablement |

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Manual Firebase checks explicitly recorded as pending before production enablement
- [x] Initial implementation review completed and all IR-001 through IR-009 findings remediated
- [x] Re-review findings IR-010 through IR-013 remediated
- [ ] Independent post-remediation implementation review (next workflow stage)

## Final Verification

- `just test` passed (race-enabled full AppView suite with Compose Postgres).
- `git diff --check` passed.
- A timing-sensitive pre-existing Tap reconnect test failed once in a non-race full run and passed immediately in isolation; the subsequent canonical race-enabled full suite passed.
- Automated tests never contacted Firebase. The official Firebase Admin Go adapter was tested through an injected capture client and the dispatcher through scripted fakes.
- MAN-001 (Android) and MAN-002 (iOS/APNs) remain operational rollout checks because this worktree has no non-production Firebase credentials or physical test devices.

## Implementation Review Remediation

Review input: `06-implementation-review.md` (`Changes required`, 2026-07-11).

| Order | Finding | Test IDs | Requirement IDs | Status |
|---:|---|---|---|---|
| 30 | IR-001 dispatcher lease/cancellation/token fencing | IT-014, IT-015, IT-018, IT-030, REG-007 | FR-012, FR-014–FR-016, FR-021, RULE-004 | Complete |
| 31 | IR-002 moderation-safe resolution | IT-017, IT-019, REG-003 | FR-023, FR-026, FR-032 | Complete |
| 32 | IR-003 per-delivery clock and TTL | IT-013, IT-016 | FR-013, FR-028 | Complete |
| 33 | IR-004 recipient membership checks | IT-010, REG-006 | BR-001, FR-008, FR-020 | Complete |
| 34 | IR-005 active semantic source replacement | IT-003, IT-020, REG-002 | FR-001, FR-002, FR-018, FR-021 | Complete |
| 35 | IR-006 profile and identity actor deletion paths | IT-022 | FR-023, FR-030 | Complete |
| 36 | IR-007 bounded indexed actor hydration | IT-019, IT-028 | FR-022, NFR-005 | Complete |
| 37 | IR-008 missing integration/telemetry evidence | IT-004, IT-005, IT-027, IT-029, IT-030 | FR-003, FR-021, NFR-002, NFR-006 | Complete |
| 38 | IR-009 dispatcher operational supervision | REG-004 | FR-012, NFR-006 | Complete |

Each remediation step follows the same red-green-refactor rule as the original implementation. The execution log and completion checklist will be reconciled after focused and repository-wide verification.

### Remediation Execution Evidence

- IR-001: deterministic tests proved claimed cancellation, old-token finalization, invalid-token rotation, and stale-worker completion failures before lease/status/subscription/installation/token fencing was added. Cancelled unstarted rows are now skipped before send.
- IR-002: active and retracted resolution tests across follow, like, repost, reply, mention, and quote initially exposed moderated direct targets; resolution now applies the same active hide/takedown and negate policy used by durable listing.
- IR-003: a two-item batch test advances the injected clock during the first send; the second item expires without a provider call and every TTL is calculated from the per-item send instant.
- IR-004: follow and mention tests initially created durable rows for non-member DIDs; both producers now check recipient membership inside their source transaction.
- IR-005: active changed-source tests for like, repost, and follow initially left the old source current and allowed a stale delete to retract it; activation now atomically retargets changed active sources while preserving stable ID and first-push fan-out.
- IR-006: profile-record deletion now calls transaction-aware actor hard deletion before membership removal; Tap tests prove only terminal `deleted` identity status reaches the deletion handler.
- IR-007: a 50-item handler test initially failed on the first external directory error; notification pages now perform one indexed identity-cache batch query and return explicit unavailable actors for missing cache rows.
- IR-008: added interaction/follow/post deletion rollback tests, a six-producer pending/retry/leased lifecycle matrix, persisted queue depth/oldest-age evidence, end-to-end provider-error telemetry sentinels, and the cancellation/stale-lease races listed under IR-001.
- IR-009: the worker initially exited on a missing-table/transient store error; it now retries operational failures with cancellable exponential supervision capped at 30 seconds and resumes after recovery.

### Remediation Verification

- Focused race suite passed: `go test -race ./internal/push ./internal/api ./internal/index ./internal/notifications ./internal/tap ./internal/app ./cmd/appview -count=1`.
- Canonical repository suite passed: `just test` (`go test -race ./...` against Compose Postgres).
- `go vet ./...` passed.
- `git diff --check` passed.
- Automated tests did not contact Firebase. MAN-001 and MAN-002 remain pre-production provider/device gates.

## Re-review Remediation

Re-review input: `06-implementation-review.md` (`Changes required`, 2026-07-11).

| Order | Finding | Test IDs | Requirement IDs | Status |
|---:|---|---|---|---|
| 39 | IR-010 generation-safe lease fencing | IT-015, IT-030, REG-007 | FR-012, FR-015, RULE-004 | Complete |
| 40 | IR-011 in-flight absolute deadline | IT-013, IT-016 | FR-013, FR-028 | Complete |
| 41 | IR-012 per-reference moderated hydration | UT-005, IT-019, REG-003 | FR-022, FR-023, FR-032 | Complete |
| 42 | IR-013 response and lifecycle matrices | UT-005, IT-018, IT-019 | FR-021–FR-023 | Complete |
| 43 | IR-013 end-to-end privacy sentinels | IT-027 | NFR-002 | Complete |

MAN-001 and MAN-002 remain explicit pre-production rollout gates. An independent post-remediation review remains the next workflow stage.

### Re-review Execution Evidence

- IR-010: the stale-worker regression initially showed that two overlapping workers using the production worker name `appview` shared the same lease identity, allowing a stale generation to overwrite a recovered delivery. Each claim now receives a unique UUID lease token, and every ownership, finalization, cancellation, and invalid-token mutation also requires the exact token and an unexpired lease. A second regression proves a worker cannot finalize after its own lease expires.
- IR-011: a context-blocking provider double initially ran until the configured one-second send timeout even though the delivery's absolute deadline arrived after roughly 250 milliseconds. The dispatcher now bounds the provider context by the earlier of send timeout and remaining delivery lifetime, then re-reads the clock after the provider returns before final state or retry scheduling.
- IR-012: real-Postgres HTTP tests initially exposed that moderation of the notification source did not independently protect the subject, reply parent, reply root, or quoted target. The durable page query now bounds the event page first, hydrates every referenced post through the same moderation policy, emits explicit `available: false` metadata, and omits unavailable URI, CID, rkey, text, and embedded quote identifiers.
- IR-013 response/lifecycle breadth: the response contract matrix now covers follow, like, repost, reply, mention, quote, and everything-else categories with required and forbidden reference roles. The deletion lifecycle matrix now covers all six producers across pending, retry, and leased delivery states (18 combinations).
- IR-013 privacy breadth: IT-027 now exercises real device registration, durable activation/enqueue, dispatcher retry and success, persisted delivery state, metrics, structured stdout, and captured Sentry events. Token, credential, DID, handle, AT URI, text, title, image URL, and full-payload sentinels are asserted absent everywhere except the provider sender boundary.

### Re-review Verification

- Focused moderation tests passed: `go test ./internal/api -run 'TestNotification(StoreModeratesEveryReferenceRoleAcrossStates|ListWithholdsModeratedReplySourceAndQuoteTarget|StoreListsOnlyActiveDurableEventsWithStablePagination)' -count=1`.
- Full API package passed: `go test ./internal/api -count=1`.
- Focused race suite passed: `go test -race ./internal/push ./internal/api ./internal/index ./internal/notifications ./internal/observability -count=1`.
- Canonical repository suite passed: `just test` (`go test -race ./...` against Compose Postgres).
- `go vet ./...` passed.
- `git diff --check` passed.
- The first sandboxed `just test` attempt was environmental: local port binding and the Compose Postgres connection were denied. The identical approved local-network rerun passed.
- Automated tests did not contact Firebase. MAN-001 and MAN-002 remain pre-production provider/device gates.
