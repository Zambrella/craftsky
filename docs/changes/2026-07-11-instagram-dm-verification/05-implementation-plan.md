# TDD Implementation Plan: Instagram DM Ownership Verification And Follow Discovery

## Inputs

- Requirements: `01-requirements.md`
- Acceptance tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes
- Coding plan: `04-coding-plan.md`
- Original design: `design-plan.md`; superseded where `01`/`02` are more exact
- Implementation approval: explicitly confirmed by the user on 2026-07-19

## Implementation Rules

- Do not implement behavior without a linked requirement and acceptance test.
- Add or change one focused test, observe its meaningful failure, implement the
  minimum behavior, rerun green, and refactor only while green.
- Keep parallel work in disjoint files and independently green packages.
- Use only wholly synthetic/redacted Instagram inputs in source and tests.
- Never contact Meta from automated tests; use fakes or `httptest.Server`.
- Keep private graph data in AppView Postgres and write only explicitly accepted
  ordinary follows to the PDS.
- Require current membership and the complete fail-closed eligibility policy at
  every boundary enumerated by the requirements.
- Keep fixed-account Flutter operations fenced by `ActiveAccountLease` after
  every await.
- Do not edit lexicons, commit, push, or enable a production integration.
- Keep red/green commands and meaningful evidence current in the execution log.

## Ordered Slices

| Step | Test IDs | Requirements / criteria | Initial expectation |
|---:|---|---|---|
| 1 | UT-001 | FR-002; AC-003, AC-004 | Challenge package absent; focused Go test fails to compile |
| 2 | UT-008, UT-016, IT-013 | FR-001, FR-003, FR-027; AC-001, AC-002, AC-040 | Config/limit/integration wiring absent |
| 3 | IT-001, UT-002 | FR-004–FR-010, FR-012, FR-015; AC-005–AC-010 | Schema and state machines absent |
| 4 | IT-020 | FR-030; AC-048 | Shared current-member boundary absent |
| 5 | IT-021, TD-011 | FR-011–FR-026; AC-011–AC-038 | Shared wire corpus absent |
| 6 | IT-002, UT-003, UT-004, IT-003 | FR-004–FR-009; AC-005–AC-014 | Verification routes/webhook absent |
| 7 | UT-007, IT-004 | FR-003, FR-007–FR-010; AC-013–AC-016 | Durable worker/Meta adapter absent |
| 8 | UT-006, IT-005, IT-006 | FR-008–FR-011, FR-014, FR-030; AC-017–AC-022 | Link/conflict/eligibility services absent |
| 9 | UT-005, IT-007, IT-008 | FR-012–FR-018; AC-023–AC-029 | Import/match/reconciliation absent |
| 10 | IT-009 | FR-017, FR-018; AC-030, AC-031 | Stable-rkey follow acceptance absent |
| 11 | IT-010 | FR-028; AC-041, AC-042 | Retention/export primitives absent |
| 12 | UT-013, UT-014, IT-011, IT-012 | FR-019–FR-022; AC-032–AC-038 | Social/system union and coalescing absent |
| 13 | IT-018, IT-019 | FR-028, FR-029; AC-043, AC-044 | Operator and worker-control paths absent |
| 14 | UT-009, UT-010, IT-014 | FR-023, FR-024; AC-039–AC-043 | Flutter parser/API/repository absent |
| 15 | UT-011, IT-015, IT-016 | FR-023–FR-026, FR-030; AC-044–AC-047 | Flutter providers/routes/page absent |
| 16 | UT-012, IT-017 | FR-019, FR-025; AC-036–AC-038 | Actorless Flutter notification absent |
| 17 | UT-015, REG-001–REG-012, TD-001–TD-012 | NFR/security/privacy requirements; AC-039–AC-049 | Full verification and privacy review pending |
| 18 | IT-002, IT-022, REG-012 | FR-003, FR-024, NFR-008; AC-042, AC-049 | Owner-current lookup and secure resumable display snapshot absent |

The exact requirement/test matrix in `02-acceptance-tests.md` remains
authoritative if this condensed table omits a secondary linkage.

## Implementation Steps And Evidence

### Step 1 — Challenge contract

- Add `appview/internal/instagram/challenge_test.go` first.
- Prove the 30-symbol alphabet, 13 random symbols, canonical grouping, exact
  whole-message grammar, outer whitespace/ASCII case normalization only,
  injected entropy errors, HMAC digest/equality, and redacted string behavior.
- Red command: `cd appview && go test ./internal/instagram -run TestChallenge -count=1`
- Implement only `challenge.go` and the minimum supporting types.

### Step 2 — Configuration and shared limiting

- Add disabled/local/full/partial-production config tests before fields/parsing.
- Add a transactional Postgres limiter test before persistence implementation.
- Prove trusted peer versus forwarded IP behavior before accepting that header.
- Wire disabled/fake modes without any Meta provider construction/call.

### Step 3 — Schema and state machines

- Add migration inspection/round-trip tests before migrations `000023`/`000024`.
- Assert checks, uniqueness, indexes, absence of membership cascades, sensitive
  fields, support-source multiplicity, deterministic follow operations, and
  system/social union migration.
- Add pure transition table tests before state helpers.

### Step 4 — Current membership

- Add route-policy/middleware and worker-transition failures first.
- Prove reversible membership inactivation/rejoin separately from terminal
  identity purge; reactivation is always explicit and does not extend consent.

### Step 5 — Shared wire corpus

- Add wholly synthetic fixtures and Go validation tests first.
- Include all states, success/error envelopes, cursors, privacy-preserving
  DELETE behavior, conflicts/unavailability, and social/system notifications.
- Flutter tests later consume the exact same files.

### Steps 6–13 — AppView vertical behavior

For each route/service slice, start at the narrowest domain/store test, then the
real handler/mux, then nearby regression packages. Provider calls remain behind
fakes. Store transitions around provider calls are separate transactions with
revalidation. Update this runbook with each meaningful red/green result.

### Steps 14–16 — Flutter behavior

Start with the pure parser and wire models, then fixed-account repository tests,
then controllers, page/widgets, and finally notification union/open flow.
Regenerate Riverpod/go_router/localization output only after hand-written tests
and source are green. Every asynchronous test covers account switch or
switch-away/back fencing where relevant.

### Step 17 — Review and verification

- Run focused Go/Flutter tests during each slice.
- Run migration up/down and real-Postgres integration tests.
- Run `go test ./...`, focused `-race`, `go vet ./...`, full Flutter tests,
  formatting, generated-code drift, analyzer, and `git diff --check`.
- Compare analyzer output with the recorded one-info baseline.
- Use the implementation-review skill, remediate actionable findings test-first,
  re-review, and rerun all affected/full gates.
- Record manual/live Meta, export-shape, safety-adapter, device, accessibility,
  and production privacy checks as pending release gates rather than passing.

## Execution Log

| Step | Status | Red evidence | Green evidence | Notes |
|---:|---|---|---|---|
| 1 | Complete | Package symbols were absent (`NewChallengeCodec`, canonicalizer, digest types) | `go test ./internal/instagram -run TestChallenge -count=1` passed | 10,000 deterministic values; rejection sampling; keyed storage-safe digest |
| 2 | Complete | Instagram config and persistent limiter symbols were absent; a pointer-format test then exposed key bytes | Config/race/vet green; shared Postgres limiter exact-boundary and 100-way race tests green | Trusted request-IP and fail-closed dependency wiring complete |
| 3 | Complete | Migration files and closed state symbols were absent | Real-Postgres `000023`/`000024` schema tests and pure transition matrices passed | Current username ownership is now durably unique |
| 4 | Complete | Current-member middleware/store symbols were absent | Middleware, real membership-store, verification mux, worker inactivation, and lifecycle tests passed | All member-facing routes and workers use the shared boundary |
| 5 | Complete | Shared Go/Dart corpus consumers were absent | Go and Dart consume the same wholly synthetic wire corpus | No user-derived fixture data |
| 6 | Complete | Verification domain/store/handler/route symbols were absent; conflict SQL initially had an ambiguous timestamp parameter | Real-Postgres create/supersede/redeem/confirm/replay/conflict/username-refresh tests plus API/mux wire tests passed | Meta remains disabled without complete configuration |
| 7 | Complete | Durable webhook work and bounded retry behavior were absent | Signed ingress, dedupe, lease recovery, retry, membership, and configured-limit tests passed | Provider calls use fakes or `httptest` only |
| 8 | Complete | Link/conflict policy services were absent | Unique identity claims, fail-closed policy, collision, lifecycle, and restoration-hook tests passed | Production relationship-safety adapter remains an external gate |
| 9 | Complete | Import/matcher/reconciliation services were absent | Exact normalized matching, additive support, retention, future-match, invalidation, and notification-boundary tests passed | Reconciliation is targeted and durable |
| 10 | Complete | Shared explicit-follow writer was absent | Ordinary create and deterministic Instagram `PutRecord` share one service; failure/replay/already-following tests pass | No automatic follow behavior |
| 11 | Complete | Retention/export services were absent | Real-Postgres expiry, export, purge, and batch-bound tests passed | Batch remains at most 500 |
| 12 | Complete | Preference scope, actorless feed scans, post-union social activation, partial-retraction newness, and post-lease preference races failed before implementation | Notification/API/push real-Postgres suites cover coalescing, retraction, delivery, feed, open, and newness | Every declared eligibility stage has a production caller |
| 13 | Complete | Operator commands and redacted result types were absent | Conflict/link/job/retention CLI and operator tests passed | Output uses opaque identifiers only |
| 14 | Complete | Flutter parser/API/repository layers were absent | Parser, JSON-only import, wire, repository, and error-envelope tests passed | ZIP remains intentionally unsupported |
| 15 | Complete | Flutter controllers/routes/page were absent | Provider and widget tests cover fixed-account fencing, verification, import, and suggestions | Active-account lease checks remain after awaits |
| 16 | Complete | Actorless notification model/open behavior was absent | Notification model, row, payload, routing, and Instagram destination tests passed | Existing social variants remain intact |
| 17 | In progress | Cross-stack verification and privacy review were pending | Focused real-Postgres, Go, Dart, analyzer, format, canary, and diff checks pass | Final broad suites/re-review await the workflow exit choice; live Meta/export/device/edge checks remain external gates |
| D1 | Complete | A live signed Meta delivery returned `200` but produced no durable webhook work, and the attempt remained `pendingDm`; no reducer-boundary diagnostics existed | `go test ./internal/app ./internal/integrations/instagrammeta` passes; rebuilt AppView confirms the dev flag is enabled | User-authorized temporary capability-spike exception: `INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES=true` logs raw signed webhook bodies only in dev and is forcibly disabled in prod; Meta tokens/secrets remain excluded |
| D2 | Complete | Store/API/client symbols for current-attempt reads were absent; the resume test then showed the challenge was not cached; confirmed sign-out did not clear private session state | `go test ./...`, focused Go race, `go vet ./...`, and all 964 Flutter tests pass; focused Dart formatting and `git diff --check` pass; analyzer reports only 13 pre-existing diagnostics outside this slice | Owner-scoped current lookup, DID-scoped secure display snapshot, AppView reconciliation, polling/confirmation resumption, terminal cleanup, and session cleanup are complete; ordinary page disposal/account switching retains the bounded snapshot |
| D3 | Complete | The candidate confirmation page placed static discovery copy before an unselected selector | Focused widget test failed on selector order before implementation, then passed with selector state/copy assertions | The selector now follows the account, defaults to discovery allowed, updates the explanation for both choices, and confirms or cancels through existing state operations |

## Implementation Review Remediation (2026-07-21)

The user explicitly approved addressing the `06-implementation-review.md`
findings before the requested UI polish. Remediation keeps the original
requirements and acceptance tests authoritative and proceeds test-first in this
order:

| Step | Finding | Test IDs | Requirements | Initial expectation |
|---:|---|---|---|---|
| R1 | IR-001 | IT-001, IT-005 | FR-009 | Different IGSIDs can currently claim the same normalized username |
| R2 | IR-002 | IT-010 | FR-028, NFR-003 | A superseded link currently retains plaintext username fields |
| R3 | IR-003 | IT-011, IT-012 | FR-010, FR-018, FR-020, FR-028 | Dependent invalidation is duplicated and incomplete |
| R4 | IR-004 | IT-011, IT-012 | FR-015, FR-020, FR-022 | Delivery, feed, and open do not all re-evaluate eligibility |
| R5 | IR-005 | IT-020 | FR-028, FR-030 | Worker-observed membership loss only rejects the attempt |
| R6 | IR-006 | IT-013 | NFR-002, NFR-004 | Private writes can bypass a missing persistent limiter |
| R7 | IR-007 | IT-006, IT-011 | FR-011, FR-019 | Username refresh and safety-restoration enqueue seams are absent |
| R8 | IR-008 | IT-009 | FR-017 | Ordinary and Instagram follows use separate PDS write services |
| R9 | IR-009 | UT-007, UT-016 | NFR-002 | Validated runtime limit settings are not carried to workers/lifecycle |
| R10 | IR-010 | UT-015 | NFR-003 | Controlled privacy canaries do not cover every prohibited surface |
| R11 | IR-011 | REG-001–REG-011 | NFR-005 | Static gates and this execution log are stale |

For each row: add one focused regression, observe a meaningful failure, make
the minimum implementation change, rerun the focused and neighboring suites,
then record the red/green evidence here. UI polish begins only after R1–R11 are
green.

### Remediation Evidence

| Finding | Status | Green evidence |
|---|---|---|
| IR-001 | Complete | Concurrent different-IGSID/same-username confirmations leave one authoritative current link and one private conflict |
| IR-002 | Complete | Superseded tombstones contain no plaintext IGSID or username immediately after confirmation |
| IR-003 | Complete | Account/import/conflict/supersession transitions invalidate accepting work, follow operations, notification support, leased pushes, and targeted jobs transactionally |
| IR-004 | Complete | Delivery, feed, and open/list callers exercise the shared policy; stale support retracts and unavailable safety data fails closed |
| IR-005 | Complete | Worker-observed departed membership invokes full private-data inactivation before terminal work handling |
| IR-006 | Complete | Missing persistent limiter returns stable `instagram_unavailable` for private writes while local reads/privacy deletes remain available |
| IR-007 | Complete | Same-IGSID DM re-verification refreshes usernames safely; a narrow fake-backed restoration enqueue interface is wired |
| IR-008 | Complete | Ordinary and deterministic Instagram follows use `internal/followwrite.Service` |
| IR-009 | Complete | Tightened lease/retry/processing-age and notification window/count settings alter runtime behavior within fixed maxima |
| IR-010 | Complete | Synthetic Go/Dart canaries cover diagnostics, Sentry/logs/metrics, push, PDS records, URLs, errors, and stringification |
| IR-011 | In progress | Go/Dart formatting, focused analyzer/widget checks, and `git diff --check` pass; final broad gates remain for the final-review stage |

## Completion Checklist

- [x] Every Must requirement is implemented or recorded as an external release gate
- [x] All planned automated Must tests pass
- [x] AppView disabled/local/fake/full configuration paths are covered
- [x] No automated test contacts Meta
- [x] No real/user-derived private fixture is committed
- [x] Every Instagram API/worker transition enforces current membership
- [x] Eligibility is shared and fails closed at every required boundary
- [x] Membership loss and terminal account deletion remain distinct
- [x] Flutter operations remain fixed-account and redacted
- [x] Existing social notification and follow behavior remains green
- [ ] Full Go and Flutter verification is complete
- [ ] Final implementation re-review is complete
- [x] Live Meta/export/safety/device/accessibility gates are explicitly reported

## Known External Gates

- Meta app credentials, webhook subscription, live unrelated-sender DM,
  profile lookup, token/permission, and reply behavior
- Approved observation of current Accounts Center JSON export variants
- Production block/mute safety-data adapter
- Deployed trusted-edge/replica limit behavior
- Physical-device push/open/file-picker/accessibility validation
- Final security/privacy/operator-access review before production enablement
- Remove the temporary raw-webhook diagnostic and unset
  `INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES` after the live Meta capability spike.
