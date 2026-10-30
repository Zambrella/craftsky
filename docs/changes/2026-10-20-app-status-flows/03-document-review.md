# Document Review: App Status Flows

## Verdict

Status: Approved with notes

Reviewer: Codex, document consistency and test-design review

Date: 2026-10-20

Risk level: Medium

## Summary

The requirements and test design are ready for coding planning. The user confirmed the two recommended behavior resolutions and requested updates to both documents. Those decisions are now normative in D-011/D-012, RULE-003, FR-008, and AC-021/AC-022; their tests no longer depend on provisional assumptions.

All 22 Must requirements have linked acceptance criteria and designed tests. All 23 acceptance criteria have designed verification paths. This review approves planning readiness, not implementation or release verification. No executable tests were run, commits made or live resources changed.

The original review verdict was Changes required. The re-review below records both resolved blockers and remaining non-blocking notes; stable finding IDs are preserved.

## Findings

| ID | Severity | Area | Finding / Resolution | References | Action / Status |
|---|---|---|---|---|---|
| DR-001 | Important — resolved | Requirements / Tests | User confirmed that any failed, invalid or timed-out refresh releases the accepted update policy. Fresh maintenance retains only its existing five-minute trust; expiry cannot revive the invalidated update policy. A later valid snapshot can enforce again. | `01-requirements.md` D-011, RULE-001/003, AC-021, EC-001/003, resolved ASM-001; `02-acceptance-tests.md` AT-011, UT-008, resolved GAP-001 | Closed. The requirements and test cases cover failure with/without maintenance, expiry and subsequent valid-policy recovery. |
| DR-002 | Important — resolved | Requirements / Tests | User confirmed successful CTA activation dismisses the captured message revision. Activation failure retains it with retryable feedback. If saving the dismissal fails after activation succeeds, hide it for the current session but allow recurrence after restart. Older completions cannot dismiss replacements or bypass a new blocking flow. | `01-requirements.md` D-012, FR-008/010, AC-022, resolved ASM-004; `02-acceptance-tests.md` AT-008, IT-002/004, resolved GAP-002 | Closed. Activation success, activation failure, persistence failure and delayed replacement/blocking transitions now have explicit expected results. Persistence failure must not replay destination activation. |
| DR-003 | Suggestion | Requirements / Persistence | Revision reuse after another message has itself been dismissed remains unspecified: current storage holds one revision, so A dismissed → B dismissed → A republished can show A again. Do not silently expand current-message dismissal into unbounded historical storage. | `01-requirements.md` FR-008/009, AC-012/013, ASM-005; `02-acceptance-tests.md` UT-006, IT-002, TD-003 | Coding-plan note: define a bounded revision lifecycle, preferably fresh message revisions for new publications rather than reuse after later campaigns. Add a literal sequence test for the chosen contract; seek a decision if historical suppression beyond existing behavior is proposed. |
| DR-004 | Suggestion | Test design / Automation | Actual target-tab reload needs a harness that survives reload; injecting a new version value alone does not demonstrate loading a new compiled app. Local adapter proof is distinct from hosted cache proof. | `02-acceptance-tests.md` AT-005, IT-005, MAN-002, GAP-004; `01-requirements.md` FR-005, AC-008 | Coding-plan note: select an external browser controller or equivalent runner, serve known builds, and observe loaded version/URL/reload count. Keep the cheap adapter test and hosted cache check separate. |
| DR-005 | Suggestion — resolved | Test specification clarity | AT-009 now has concrete Gherkin Examples. REG-003 no longer lists AC-008 as though retained-editor assertions independently verify reload. | `02-acceptance-tests.md` AT-009, REG-003, §2 | Closed. The matrix describes designed coverage, not proof that each listed test independently covers all criteria. |

User-requested addition D-013 is reconciled across all artifacts: FR-014/AC-023 require a checked-in formal JSON Schema and fail-closed offline validation before every local/hosted upload. UT-009 and IT-003 cover meta-validation, structural/semantic rejection, final-payload validation and zero writes on schema/validator failure. The schema is an implementation target, not a file created during planning.

No blocking findings remain. DR-003 and DR-004 are implementation-design notes and must not cause unrelated scope expansion.

## Traceability Review

- Planning to requirements: D-001–D-013 reflect the confirmed independent controls, priority, inclusive ranges, platform minima, local dismissal, semantic versions, startup/current-session fail-open and CTA semantics. Existing maintenance and supported-platform boundaries remain intact.
- Requirements to acceptance criteria: all 22 Must requirements have reciprocal criterion links. New AC-021 specifically covers current-session update trust; AC-022 covers CTA activation/persistence behavior rather than attaching these behaviors only to startup/navigation criteria.
- Acceptance criteria to tests: all 23 criteria have designed coverage through AT-001–AT-011, UT-001–UT-009, IT-001–IT-006, REG-001–REG-004 and MAN-001–MAN-003. Every test references requirements and criteria.
- Status consistency: former ASM-001/004 and GAP-001/002 are marked resolved, requirements are Reviewed, and both earlier artifacts point to this approved-with-notes review. Remaining schema/destination/harness decisions belong in coding planning.

## Coverage Review

- Must requirements covered: 22 of 22 have designed coverage; no missing-ID blocker found.
- Acceptance criteria covered: 23 of 23 have designed verification paths. This is design completeness, not executed-test evidence.
- Newly explicit transitions: failed refresh under fresh maintenance, expiry without stale update revival, later valid enforcement, successful CTA with failed storage, account switch/repeat reads during session-only suppression, replacement message and new blocking flow during pending activation.
- Remaining weak points: DR-003 needs a concrete bounded revision contract and sequence test in coding planning. DR-004 needs a runnable browser harness before implementation claims reload integration coverage.
- Manual/platform checks are justified: native store handoff (MAN-001), actual hosted web cache recovery (MAN-002), physical back gestures and screen readers (MAN-003). Real IT-002 storage/device coverage is required before claiming durable persistence; host mocks alone do not establish it.
- Existing regression coverage remains appropriate: maintenance freshness, request deadlines, editor/session/account recovery, independent revisions, publication confirmation and diagnostic privacy. Independently authored fixtures and serialized-sink canaries remain part of the test design.

## Risk And Approval Review

- Risk remains Medium: blocking client UI, navigation, persistence and public JSON contract changes; no new auth authority or destructive data operation.
- User explicitly approved the DR-001/002 recommendations and authorized the document updates. This re-review finds no remaining blocker to coding planning.
- No implementation, commit, deployment, hosted publication or infrastructure mutation is authorized by this review artifact. Existing workflow stage boundaries remain in place.
- Store availability, schema rollout, native behavior and hosted cache recovery require separate release evidence. Old installed clients cannot acquire the new flow merely through publication of a new JSON schema.
- Privacy requirements remain unchanged: public fetches do not inherit AppView credentials, account/policy gates remain active, and diagnostics retain typed failures/stacks without exporting protected values or arbitrary remote prose.

## Coding Plan Readiness

- Ready for coding planning: Yes, with DR-003/004 notes.
- Blocking issues: None.
- Next artifact: `04-coding-plan.md`, using `write-coding-plan` when requested.
- Recommended first step: define the concrete versioned contract, destination set and bounded revision semantics, then plan the literal all-controls validation failure (UT-007), semantic-version/platform tests and user-visible behavioral slices.

## Notes For Next Stage

- Preserve stable requirement, criterion, test and finding IDs; do not restore provisional language for resolved D-011/D-012 decisions.
- Keep update-policy trust invalidation separate from accepting a malformed snapshot: no sibling fields from invalid JSON become authoritative, and maintenance trust is neither renewed nor cleared early by failure.
- CTA activation failure, successful activation with durable dismissal, and successful activation with session-only suppression are distinct outcomes. Fence completion by captured message identity and retain active maintenance/update priority.
- Define supported destinations, null/unknown-field policy, legacy dismissal preserve/reset handling, image transport isolation and the reload harness in the coding plan. These routine design tasks do not require additional product approval unless they change agreed behavior.
- Keep message identity independent of document publication revision. Keep status failure distinct from AppView failure, avoid write replay/status-induced sign-out, and retain existing maintenance trust/restart behavior.
- No commit was requested. All three planning artifacts remain uncommitted.
