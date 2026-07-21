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
| 17 | UT-015, REG-001–REG-011, TD-001–TD-012 | NFR/security/privacy requirements; AC-039–AC-048 | Full verification and privacy review pending |

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
| 2 | In progress | Instagram config and persistent limiter symbols were absent; a pointer-format test then exposed key bytes | Config/race/vet green; shared Postgres limiter exact-boundary and 100-way race tests green | Trusted request-IP and final dependency lifecycle wiring remain |
| 3 | In progress | Migration files and closed state symbols were absent | Real-Postgres `000023`/`000024` schema tests and pure transition matrices passed | Stores/system notification behavior continue in later slices |
| 4 | In progress | Current-member middleware/store symbols were absent | Middleware, real membership-store, and verification mux boundary tests passed | Apply the same guard to account/import/suggestion routes and workers as they land |
| 5 | Pending | — | — | Synthetic shared corpus only |
| 6 | In progress | Verification domain/store/handler/route symbols were absent; conflict SQL initially had an ambiguous timestamp parameter | Real-Postgres create/supersede/redeem/confirm/replay/conflict tests plus API/mux wire tests passed | Durable webhook and rate-policy completion remain |
| 7 | Pending | — | — | Bounded durable work |
| 8 | Pending | — | — | Production safety source initially unavailable/fail-closed |
| 9 | Pending | — | — | Exact matching; additive imports |
| 10 | Pending | — | — | Explicit follow only |
| 11 | Pending | — | — | Batch at most 500 |
| 12 | In progress | Preference scope, actorless feed scans, post-union social activation, partial-retraction newness, and post-lease preference races failed before implementation | Notification/API/push package race suites and focused real-Postgres union/coalescing/retraction/newness tests passed | AppView union/outbox service is complete; future-match and terminal lifecycle callers still need transactional wiring; Flutter is Step 16 |
| 13 | Pending | — | — | Redacted operator output |
| 14 | Pending | — | — | JSON only, no ZIP |
| 15 | Pending | — | — | Active-account lease fencing |
| 16 | Pending | — | — | Preserve all existing social variants |
| 17 | Pending | — | — | Includes implementation review |

## Completion Checklist

- [ ] Every Must requirement is implemented or recorded as an external release gate
- [ ] All planned automated Must tests pass
- [ ] AppView disabled/local/fake/full configuration paths are covered
- [ ] No automated test contacts Meta
- [ ] No real/user-derived private fixture is committed
- [ ] Every Instagram API/worker transition enforces current membership
- [ ] Eligibility is shared and fails closed at every required boundary
- [ ] Membership loss and terminal account deletion remain distinct
- [ ] Flutter operations remain fixed-account and redacted
- [ ] Existing social notification and follow behavior remains green
- [ ] Full Go and Flutter verification is complete
- [ ] Implementation review and remediation are complete
- [ ] Live Meta/export/safety/device/accessibility gates are explicitly reported

## Known External Gates

- Meta app credentials, webhook subscription, live unrelated-sender DM,
  profile lookup, token/permission, and reply behavior
- Approved observation of current Accounts Center JSON export variants
- Production block/mute safety-data adapter
- Deployed trusted-edge/replica limit behavior
- Physical-device push/open/file-picker/accessibility validation
- Final security/privacy/operator-access review before production enablement
