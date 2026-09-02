# Implementation Review: Provider-First AT Protocol Registration

## Verdict
Status: Approved with notes
Reviewer: OpenCode implementation correction review
Date: 2026-08-31
Risk level: High

## Summary
The implementation satisfies the approved automated requirements and acceptance tests. The prior credential-lifecycle, stale-exchange, migration-shape, and Flutter error-mapping findings are closed. Runtime validation also found and corrected two development issues: the isolated database had an incrementally applied early version of uncommitted migration 61, and live `bsky.social` registration starts at an authorization-server issuer whose protected-resource endpoint returns 404. Migration replay restored the final schema, and exact-origin issuer fallback now reaches live Bluesky PAR without weakening transient or malformed-discovery failure handling. The required signed-build live account smoke test remains a release-only blocker.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-005 | Important | Release Evidence | Open, release-only. Registration start now completes live discovery and PAR, but the required create-account/callback/onboarding test from a production-like signed build has not run and `release-smoke-evidence.md` does not exist. | `01-requirements.md` AC-028; `02-acceptance-tests.md` MAN-001; `05-implementation-plan.md` live registration-start correction | Before release, use a production-like signed build and disposable Bluesky account, then record every required AC-028 evidence field. |
| IR-003 | Suggestion | Migration Hardening | The Must-level authority, NULL/blank metadata, and partial-owner constraints are resolved. Migration 64 still permits some registration state/authority combinations that runtime does not ordinarily produce. | `appview/migrations/000064_provider_first_registration.up.sql`; `appview/internal/db/provider_registration_migration_test.go`; AC-027, IT-005, IT-021 | Consider exhaustively constraining each registration state against absent and complete authority in a follow-up. |
| IR-006 | Suggestion | Accessibility / Layout | Add Account remains a fixed non-scrollable centered column; compact height, keyboard insets, or large text can reduce action accessibility. | `app/lib/auth/pages/sign_in_page.dart`; FR-001, FR-012; AC-020, AC-021 | Consider compact-height, keyboard-inset, and large-text widget coverage in a polish pass. |
| IR-007 | Suggestion | Concurrency / UI | Completion-page Retry remains enabled while a fresh registration start is pending, allowing repeated starts and browser launches. | `app/lib/auth/pages/auth_complete_page.dart`; FR-014, FR-015; AC-024, AC-026 | Consider a pending state and repeated-tap widget coverage. |
| IR-008 | Suggestion | Traceability / Operations | Planned registration-specific telemetry and provider environment/README documentation remain absent without an explicit recorded deviation. | `04-coding-plan.md`; no corresponding registration-specific `internal/observability/`, `appview/environments/`, or `appview/README.md` changes | Implement the planned operational additions or record an approved deviation. |
| IR-009 | Suggestion | Reliability / Throughput | Reconciliation runs before ordinary sweeping, but a full reconciliation batch does not trigger immediate backlog draining when ordinary sweeping removes nothing. | `appview/cmd/appview/auth_request_sweeper.go`; IT-016, REG-006 | Consider immediate continuation after a full reconciliation batch and add a multi-batch restart test. |

## Requirement And Test Traceability
- Requirements implemented: Provider-first start, server-owned provider selection, exact issuer/PDS authority proof, token quarantine and cleanup, lifecycle fencing, onboarding and code-only handoff, trusted failure return, Flutter entry and multi-account behavior, and existing login/deletion compatibility map to the approved BR/FR/NFR/RULE IDs.
- Tests implemented: Planned automated AT/UT/IT/REG coverage is represented, including recoverable malformed-success cleanup, production stale-exchange reconciliation, migration up/down/up, real 502 mapping, protected-resource discovery, authorization-server issuer fallback, DPoP nonce replay, and provider-neutral completion.
- Unplanned behavior: None identified. The issuer fallback follows AT Protocol URL-input discovery behavior and remains restricted to the validated server-owned configured origin after definitive HTTP 404.
- Remaining gaps: MAN-001 / AC-028 is intentionally manual and unexecuted. Non-blocking hardening and polish items are listed above.

## Test Evidence
- Commands reviewed: `GOFLAGS='-p=1' just appview-check`; `go test -race ./internal/auth ./internal/app -count=1`; focused issuer/protected-resource/failure-classification and prompted-PAR tests; `just app-test`; `just app-analyze`; Dart MCP analysis; `git diff --check`; rebuilt Compose AppView and live `POST /v1/auth/registrations`.
- Passing evidence: The current AppView release gate passed all checks. The focused race suite passed against real PostgreSQL. All 1,616 Flutter tests and Flutter analysis passed in the implementation correction cycle. Live registration start returned HTTP 200 with a `https://bsky.social/oauth/authorize` URL after successful PAR and DPoP nonce replay. Worker and sweeper errors did not recur after migration replay.
- Failing or skipped tests: The first full-gate rerun was terminated only by the 120-second tool timeout and passed when rerun with an adequate timeout. MAN-001 was not run. Binary audit retains existing no-fix `GO-2026-5932` for `golang.org/x/crypto/openpgp`.

## Risk Review
- Risk level: High.
- Risk notes: This feature establishes OAuth credential and owner authority. The implementation now covers malformed successful token responses, cleanup durability, issuer/PDS separation, exact issuer validation, replay, timeout, lifecycle, and migration boundaries with real PostgreSQL and controlled TLS fixtures.
- Approval notes: Ready for merge/handoff based on automated evidence. Do not release until MAN-001 / AC-028 is complete.

## UI Polish Recommendation
- Recommendation: Recommended.
- Reason: Behavior is approved, but compact-height and pending-retry states have visible accessibility and interaction rough edges.
- Suggested polish notes: Adapt Add Account for insets and large text, and make Retry visibly pending without changing registration semantics.

## Handoff Back To TDD Builder
- Required fixes: None before merge/handoff. MAN-001 remains mandatory before release.
- Suggested next failing test: None required for approval. If accepting IR-007, begin with a pending-completer repeated-tap widget test.
- Verification to rerun: After any follow-up change, rerun its focused tests and the affected release gate; execute and record MAN-001 before release.
