# Implementation Review: Follower Growth Metrics

## Verdict

Status: Approved with notes
Reviewer: OpenCode
Date: 2026-08-26
Risk level: Medium

## Summary

The implementation satisfies the approved requirements and acceptance tests. The final correction closes IR-018 by requiring in-range global availability and latest metadata to identify the actual first and last observations. No remaining Must-level implementation defect was identified. Release-device checks and a clean repeat of the capacity-affected AppView gate remain non-blocking operational follow-ups.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: Daily canonical persistence, atomic capture, shared capture/purge fencing, canonical membership counts, bounded sparse reads, owner-only API, deletion, bounded telemetry, account-keyed state, typed navigation, localized Growth states, chart rendering, accessibility summaries, and strict wire-boundary validation map to the approved requirements.
- Tests implemented: Backend persistence/concurrency/privacy and API tests, Flutter model/API/provider/router/widget tests, stale account/period tests, trend semantics, compact non-default-locale render-bound evidence, and malformed global-boundary payload regressions are present.
- Unplanned behavior: None identified. No lexicon, PDS-write, public analytics, ranking, recommendation, advertising, or client-analytics surface was added.
- Remaining gaps: MAN-001 and MAN-002 remain explicitly outstanding release-candidate checks. GAP-001 remains the documented production-scale capture-timing follow-up.
- Prior findings closed: IR-001 through IR-018 are closed.

## Test Evidence

- Commands reviewed: Focused correction suites; full Flutter tests and analysis; `just test`; `just appview-check`; `git diff --check`.
- Passing evidence: All 1,510 Flutter tests passed; Flutter analysis and focused Dart analysis reported no issues; `just test` passed all AppView packages with real Postgres and the race detector; `git diff --check` passed. The exact `just appview-check` gate passed immediately before the client-only IR-018 correction.
- Failing or skipped tests: Two post-correction `just appview-check` reruns stopped when disposable Postgres returned `out of shared memory (SQLSTATE 53200)` during unrelated package schema setup. Different failing packages across the reruns, the preceding clean gate, and the current clean `just test` evidence classify this as infrastructure capacity rather than a follower-growth regression. MAN-001 and MAN-002 were not performed because a supported release-device session was unavailable.

## Risk Review

- Risk level: Medium
- Risk notes: Privacy, deletion, ownership, capture atomicity, wire consistency, stale account/period isolation, and outage-freshness risks are covered. Residual risk is limited to release-device presentation/accessibility validation and production-scale capture timing.
- Approval notes: The feature is ready for handoff. Repeat `just appview-check` when disposable Postgres has sufficient capacity and complete MAN-001/MAN-002 before release; neither note requires another implementation correction pass.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The functional hierarchy, state copy, responsive behavior, localized chart labels, and accessible summaries are coherent. Small visual refinements may still be useful but are not approval blockers.
- Suggested polish notes: Consider a subtle latest-count/change/date summary surface. Do not substitute polish for release-device accessibility checks.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None.
- Verification to rerun: Repeat `just appview-check` when the disposable Postgres capacity issue is clear; complete MAN-001 and MAN-002 on a release candidate.
