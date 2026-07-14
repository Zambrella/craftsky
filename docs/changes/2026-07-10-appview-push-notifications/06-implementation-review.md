# Implementation Review: AppView Push Notifications

## Verdict

Status: Changes required  
Reviewer: Codex  
Date: 2026-07-11  
Risk level: High

## Summary

The remediation materially improves the implementation. Prior findings IR-002, IR-004 through IR-007, and IR-009 are addressed: resolution now applies moderation policy, recipients are membership-checked, active semantic sources are retargeted, profile deletion invokes actor cleanup transactionally, notification handles are hydrated in one indexed batch, and dispatcher operational errors are supervised. The race-enabled focused and repository-wide suites pass.

The change is still not ready to merge or enable. IR-001 and IR-003 are only partially closed: lease finalization is fenced by a caller-supplied owner string that is identical in every production process, and provider sends are not bounded by the delivery's absolute deadline. Feed hydration also applies moderation only to the single joined `subject_uri`; reply source records and quoted targets can still expose inaccessible identifiers and can be reported as available. The corresponding Must-level acceptance coverage remains incomplete even though `05-implementation-plan.md` marks it complete.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-010 | Critical | Behavior / Risk | Lease fencing is not generation-safe. Every running AppView passes the literal owner `"appview"`, while finalization checks only `status='leased' AND lease_owner=$worker`. If a lease expires and another process or restarted worker reclaims it using the same configured owner, the stale sender still satisfies the fence and can overwrite the newer result. The remediation test uses distinct owners (`stale` and `fresh`), so it cannot expose the production case. Lease expiry is also not part of the finalization predicate. | IR-001; FR-012, FR-015, RULE-004; AC-020, AC-021; IT-015, IT-030, REG-007; `appview/cmd/appview/main.go:99`; `appview/internal/push/dispatcher.go:62-97, 122-143, 171-214`; `appview/internal/push/dispatcher_test.go:238-266` | Give every claim a unique lease token/generation and fence every pre-send ownership check, terminal/retry write, and invalid-token transaction with that token. Add a stale-worker recovery test in which old and new dispatcher invocations use the same process-level worker name, plus coverage for completion after lease expiry/reclaim. |
| IR-011 | Important | Behavior / Risk | The dispatcher re-reads the clock before each item, but a send that starts just before `deadline_at` receives the full configured send timeout. FCM can therefore accept the request after the six-hour absolute deadline, and its TTL was calculated before the blocking network call. Success/retry timestamps and retry decisions also use that pre-send instant. The current advancing-clock test returns immediately and does not exercise a send crossing the deadline. | IR-003; FR-013, FR-028; AC-019, AC-036; IT-013, IT-016; `appview/internal/push/dispatcher.go:119-143`; `appview/internal/push/dispatcher_test.go:317-340` | Bound the provider context by the earlier of the send timeout and `deadline_at`, re-read time after the provider call for persistence/retry decisions, and add a blocking-send test that crosses the absolute deadline. |
| IR-012 | Critical | Behavior / Risk | Durable feed hydration does not apply visibility independently to every required reference. The SQL checks moderation only for `e.subject_uri`. For replies, that is the parent post, so a hidden/taken-down reply source can still be exposed through `uri`, `cid`, `rkey`, and `reply`, while `contentAvailable` is `true` because the parent is visible. For quotes, the source quote post is checked but the quoted destination is returned as a raw strong reference without a visibility check or explicit unavailable state. This bypasses the Must rule that notification hydration not expose inaccessible metadata. | FR-022, FR-023, FR-032; AC-022 through AC-024; IT-019, REG-003; metadata matrix in `01-requirements.md` section 15; `appview/internal/index/craftsky_post.go:252-267, 313-325`; `appview/internal/api/notification_store.go:82-135`; `appview/internal/api/notifications.go:126-155`; `appview/internal/api/durable_notification_store_test.go` | Batch-hydrate and moderate each category's source, subject, parent/root, and quoted references independently. Return explicit unavailable representations and omit inaccessible URI/CID/content rather than deriving one `contentAvailable` flag from `subject_uri`. Add available, missing, hidden, and taken-down list tests for every category and reference role. |
| IR-013 | Important | Tests / Traceability | The remediation does not complete the planned Must evidence, but `05-implementation-plan.md` says all automated Must tests pass and all IR findings are remediated. Notification response tests still cover only follow batching plus like/reply shapes; the durable takedown test covers only a liked post; IT-027 does not exercise registration, enqueueing, logs, Sentry, API responses, credentials, DIDs, handles, AT-URIs, text, titles, and image URLs; and the producer lifecycle test assigns one unsent state to each producer instead of exercising every producer/state combination. The stale-worker test also misses the real shared owner value, and deadline coverage misses an in-flight crossing. | IR-008; UT-005, IT-018, IT-019, IT-027, IT-030, REG-003, REG-007; `appview/internal/api/notifications_test.go`; `appview/internal/api/durable_notification_store_test.go`; `appview/internal/index/notification_lifecycle_test.go:231-259`; `appview/internal/push/dispatcher_test.go:238-340, 400-419`; `05-implementation-plan.md:105-113, 123-159` | Add the missing acceptance matrices and end-to-end privacy sentinel coverage through real public seams, then reconcile the implementation plan so completion claims match the executed evidence. Preserve MAN-001 and MAN-002 as explicit rollout gates. |

## Requirement And Test Traceability

- Requirements implemented: The core architecture and most behavior for BR-001 through BR-003, FR-001 through FR-033, NFR-001 through NFR-006, and RULE-001 through RULE-006 is present. IR-010 through IR-012 identify remaining Must-level concurrency, freshness, and visibility violations.
- Tests implemented: The remediation adds cancellation/token fencing cases, moderated resolution across categories, per-item clock advancement, non-member follow/mention cases, like/repost/follow source replacement, profile deletion cleanup, bounded handle hydration, deletion rollback cases, producer deletion states, persisted queue metrics, a provider-error sentinel, and transient worker recovery.
- Unplanned behavior: No intentional feature expansion was identified. The unrelated untracked `docs/changes/2026-07-11-instagram-dm-verification/` folder remains excluded from this review.
- Remaining gaps: IR-010 through IR-013, plus MAN-001 and MAN-002 before production enablement.

## Test Evidence

- Commands reviewed: Remediation evidence in `05-implementation-plan.md`; focused `go test -race ./internal/push ./internal/api ./internal/index ./internal/notifications ./internal/tap ./internal/app ./cmd/appview -count=1`; repository-root `just test`; `go vet ./...`; `git diff --check`.
- Passing evidence: The focused race suite passed on 2026-07-11. `just test` then passed outside the sandbox restriction with `go test -race ./...` against the local Compose Postgres. `go vet ./...` and `git diff --check` passed.
- Failing or skipped tests: The sandboxed `just test` attempt failed only because localhost Postgres and loopback test-server access were denied; the same command passed with that restriction removed. MAN-001 (Android FCM) and MAN-002 (iOS/APNs) remain pending. Missing automated cases are detailed in IR-013.

## Risk Review

- Risk level: High
- Risk notes: The feature stores private device tokens and coordinates external sends with deletion, token rotation, moderation, and lease recovery. A non-generation lease fence can corrupt terminal state after recovery, and incomplete per-reference moderation can expose inaccessible identifiers through the notification feed.
- Approval notes: Not ready to merge, hand off as complete, or enable in production. Passing existing tests does not cover the remaining counterexamples.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This implementation contains no Flutter or other user-facing UI changes.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: Address IR-010 through IR-013, then update the implementation plan's remediation claims and verification evidence.
- Suggested next failing test: Claim a delivery with worker name `appview`, block its sender, advance past the lease, reclaim and succeed using another dispatcher also named `appview`, then release the stale sender with a permanent result. Assert the newer success remains terminal. Follow with a notification-list matrix where a reply source and quote target are independently taken down while their other references remain visible, asserting no inaccessible URI/CID or stale availability state is returned.
- Verification to rerun: Focused race suites for `./internal/push`, `./internal/api`, `./internal/index`, and `./internal/notifications`; then repository-root `just test`, `go vet ./...`, and `git diff --check`. Run MAN-001 and MAN-002 only after automated review is green and before production enablement.
