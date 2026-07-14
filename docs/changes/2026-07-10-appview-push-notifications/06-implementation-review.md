# Implementation Review: AppView Push Notifications

## Verdict

Status: Approved
Reviewer: Codex
Date: 2026-07-14
Risk level: High

## Summary

The account-wide notification newness follow-up satisfies the approved requirements and preserves the existing push-notification architecture. Migration 000022 adds monotonic activation revisions, separate active-count and all-state high-water indexes, and one acknowledgement row per account. Exact replay and retraction retain a revision; genuine reactivation/source replacement advances it without creating a second push.

The API exposes read-only `GET /v1/notifications/new-count` and bodyless `POST /v1/notifications/seen`. Count visibility matches the durable list's active/actor-moderation boundary. Mark-seen uses one SQL statement, so its revision scan and greatest-value upsert share a statement snapshot; a notification committed while the marker row is locked remains new. Route-level coverage proves acknowledgement is account-wide across devices and isolated between accounts sharing one device.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: BR-004, FR-034 through FR-038, RULE-007, and NFR-007 are implemented without changing notification eligibility, push fan-out, FCM delivery, payloads, preferences, hydration item shapes, or lexicons.
- Tests implemented: IT-032 covers migration and activation revision lifecycle; IT-033 and REG-009 cover first-use count, marker thresholds, visibility agreement, and read-only GET behavior; IT-034 covers the real statement-snapshot race and 204 handler contract; AT-010/IT-035 cover account-wide cross-device and multi-account isolation through the registered mux.
- Unplanned behavior: None. Per-notification unread state, per-device markers, list-GET mutation, Flutter badge rendering, and individual mark-read operations remain out of scope.
- Remaining gaps: None for this AppView newness addition. Existing MAN-001 and MAN-002 remain pre-production provider/device checks for the parent push-delivery feature and are unaffected by this change.

## Test Evidence

- Commands reviewed:
  - `go test ./internal/notifications -run TestNotificationNewness -count=1`
  - focused migration, count, snapshot, route, and policy test commands recorded in `05-implementation-plan.md`
  - `go test -race ./internal/notifications ./internal/db ./internal/api ./internal/routes -count=1`
  - repository-root `just test`
  - `go vet ./...`
  - `git diff --check`
- Passing evidence: All focused tests passed against local Compose Postgres. The focused race suite passed. Canonical `just test` passed with `go test -race ./...` across the AppView. `go vet ./...` and `git diff --check` passed.
- Failing or skipped tests: Every TDD step first failed for the intended missing behavior and then passed. No automated test remains failing or skipped for this follow-up. Real FCM/APNs manual checks are unrelated rollout gates from the parent feature.

## Risk Review

- Risk level: High for the complete push-notification feature; Medium for this additive schema/API/concurrency slice.
- Risk notes: The main incremental risks were clearing a concurrent notification, allowing stale device-local state, drifting count visibility from the list, or making GET requests mutate state. Monotonic revisions, a single-statement snapshot upsert, greatest-value conflict handling, shared actor-visibility policy, and route regressions address those risks.
- Approval notes: Ready to merge or hand off as implemented, subject to the parent feature's existing manual provider rollout gates before production push enablement.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This change adds AppView contracts only and contains no Flutter or other user-facing UI changes.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None for this scope.
- Verification to rerun: If the branch changes before merge, rerun repository-root `just test`, `go vet ./...` from `appview/`, and `git diff --check`.
