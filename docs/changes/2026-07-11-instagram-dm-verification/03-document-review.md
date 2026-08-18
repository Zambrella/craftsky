# Document Review: Instagram DM Ownership Verification And Automatic Following

> **2026-08-14 status:** The original automatic-follow review is historical.
> Section 7 records the approved AppView audit correction and is authoritative
> for coding readiness.

## Verdict

Status: Approved with notes
Reviewer: Codex workflow review
Date: 2026-07-27
Risk level: High

## Summary

The revised requirements and acceptance-test specification are ready for
coding-plan work. They consistently replace the member-facing suggestion flow
with a durable AppView automatic-follow pipeline, actorful per-target
notifications, and verification-lifetime manual-unfollow suppression. The UI,
API, privacy, lifecycle, multi-account, and failure contracts are explicit and
testable.

All 55 Must requirements link to acceptance criteria and automated or justified
manual coverage. All 56 acceptance criteria have concrete test coverage. The
requirement matrix, test-ID sets, and cross-document references passed
automated completeness and continuity audits.

No substantive requirement or test-design change is required. The notes below
record workflow metadata and stale downstream-artifact risks that the coding
plan must handle.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-014 | Suggestion | Requirements / Approval | The requirements review metadata still says `Draft` and `Awaiting user approval`, although the user explicitly approved the revised requirements before acceptance-test design. This is an administrative mismatch rather than missing approval. | `01-requirements.md` §22; user approval on 2026-07-27 | Treat the recorded user approval as authoritative. If earlier documents are revised later, update §22 without changing requirement or acceptance-criterion IDs. |
| DR-015 | Important | Coding-plan readiness | The existing `04-coding-plan.md` predates the 2026-07-27 revision and still directs implementation of suggestion list/accept/dismiss routes, explicit acceptance, actorless digest/count notifications, five-minute coalescing, and People You May Know UI. It is unsafe to use as an implementation plan. | `04-coding-plan.md` §§2–12; `FR-016`–`FR-026`, `FR-032`, `RULE-012`; `IT-008`–`IT-025` | The next `write-coding-plan` stage must revise the existing plan comprehensively from current `01-requirements.md` and `02-acceptance-tests.md` before implementation begins. Preserve useful completed baseline context but remove every superseded suggestion/digest instruction. |
| DR-016 | Suggestion | Planning provenance | No `00-initial-prompt.md` exists, and the older `design-plan.md` describes the superseded reviewable-suggestion direction. The revised request and decision history are embedded in requirements Q9 and the 2026-07-27 follow-on section. | `01-requirements.md` §§1, 3 Q9, 5, 10–12; `design-plan.md` | Treat the current requirements and tests as authoritative over older planning sketches. Do not infer product behavior from the old design plan where it conflicts. |

## Traceability Review

- Planning to requirements: The confirmed durable automatic-follow
  recommendation, actorful notification behavior, manual-unfollow suppression,
  UI terminology/placement/theme changes, removed suggestion surface, and
  default export option are all represented in goals, requirements, non-goals,
  risks, and lifecycle rules.
- Requirements to acceptance criteria: All 58 requirement IDs are unique. All
  55 Must requirements link to at least one criterion. Requirement-to-criterion
  references are reciprocal and complete.
- Acceptance criteria to tests: `AC-001` through `AC-056` are continuous and
  covered. The specification defines 9 acceptance scenarios, 20 unit tests, 25
  integration tests, 14 regression tests, and 5 justified manual checks.
- Test IDs and targets: AT/UT/IT/REG/TD/MAN/GAP sets are monotonic with no gaps
  or duplicates. Every coverage-matrix test reference resolves to a defined
  test, and every referenced requirement ID exists.

## Coverage Review

- Must requirements covered: 55 of 55.
- Missing or weak coverage: None identified.
- High-risk automated seams:
  - exact-DID OAuth-session selection and narrow invalidation (`UT-019`,
    `IT-024`);
  - deterministic PDS write, crash recovery, and one notification (`IT-009`);
  - manual-unfollow suppression and fresh-verification reset (`UT-020`,
    `IT-025`, `REG-014`);
  - removed suggestion API/client/UI surface (`IT-008`, `IT-014`, `IT-016`);
  - actorful feed row and identity-free push (`UT-012`, `UT-014`, `IT-012`,
    `IT-017`);
  - verified terminology, green discovery, default export, and bottom
    revocation (`IT-016`, `IT-023`).
- Manual-only coverage: Live Meta capability/access/token/reply behavior,
  additional real export compatibility, physical push/OS lifecycle, native
  file-path/memory behavior, and final mobile accessibility. Each is
  impractical to prove hermetically and is correctly retained as a
  production/release gate rather than an implementation blocker.

## Risk And Approval Review

- Risk level: High. The feature performs background public PDS writes from
  private cross-network evidence and must preserve account/session isolation.
- Review requirement: Satisfied for requirements and test design by explicit
  user approval plus this document review.
- Approval notes: Approval covers coding-plan work. It does not authorize
  commit, push, production enablement, Meta dashboard mutation, or use of real
  or user-derived fixture data.
- Production gates remain: Live unrelated-sender Meta verification,
  access/token/reply validation, trusted edge/multi-replica enforcement,
  additional consented export compatibility, physical push lifecycle, and
  final mobile accessibility/memory inspection.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Revise `04-coding-plan.md` around `UT-019`/`IT-024`
  so exact owner-session selection exists before any background PDS write,
  followed by `UT-020`/`IT-025` for manual-unfollow suppression.
- Blocking issues: None for coding-plan work. Implementation must not start
  from the stale existing coding plan; DR-015 must be resolved by the coding
  planning stage itself.

## Notes For Next Stage

- Use `01-requirements.md` and `02-acceptance-tests.md` as the authoritative
  inputs. Preserve stable requirement, acceptance-criterion, and test IDs.
- Revise the existing coding plan rather than appending a contradictory
  follow-on section.
- Reuse the implemented private operation ledger where practical, but remove
  every public suggestion route/model/provider/widget contract.
- Design background OAuth-session selection as an exact owner-DID query with a
  deterministic most-recent usable choice and narrow invalidation.
- Separate these outcomes explicitly: pre-existing follow means
  `alreadyFollowing` with no automatic-follow notification; deterministic
  worker success means `followed` plus exactly one actorful notification;
  temporary session/PDS failure remains retryable.
- Preserve completed ZIP/parser/privacy work and the existing `instagramJson`
  server contract.

## 7. AppView Audit Re-review

Date: 2026-08-14

Verdict: Approved for correction planning; the automatic-follow implementation
is not acceptable as the final lifecycle contract.

The product owner selected the strict AV-007 branch. Requirements Section 24
and Acceptance Tests Section 12 consistently replace background public writes
with private, generation-bound suggestions and an explicit current-member
Follow action. They preserve verification, exact matching, private import
retention, discoverability, block/mute policy, account fencing, and the rule
that successful ordinary follows are not cleanup targets.

The correction is internally consistent:

- the matcher/reconciliation boundary produces private suggestions only;
- public suggestion list/accept/dismiss APIs and fixed-account Flutter state
  return as an explicit consent surface;
- only accept may enter the ordinary owner-effect/session coordinator;
- the Instagram background graph has no OAuth selector, PDS factory,
  `followwrite.Service`, or record-write capability;
- `instagramMatch` automatic-follow notification behavior is retired; and
- departure/terminal cleanup invalidates unwritten suggestions but never
  deletes `app.bsky.graph.follow`.

Risk remains High because the correction changes public-write authority,
private persistence, routes, and UI. Product approval is recorded; no further
product choice blocks coding. The Meta and physical-device items remain
release gates, not implementation gates.

Coding-plan readiness: Ready after `04-coding-plan.md` Section 13 is followed.
The old Notes For Next Stage instructions to remove suggestions and design a
background OAuth selector are superseded and must not be implemented.
