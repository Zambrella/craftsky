# Document Review: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## Verdict

Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-05-30
Risk level: High

## Summary

The requirements and acceptance-test specification are consistent and ready for coding-plan work. The selected direction (local report queue, placeholder forwarding seam, synthetic moderation-output ingestion, and read-path enforcement) is carried through from requirements into concrete test coverage. All Must business, functional, non-functional, and rule requirements have acceptance criteria and linked automated tests or justified manual checks. No blocking gaps were identified.

The recommended first implementation step remains `IT-001`: migration/store coverage for private post/profile report rows with canonical subject snapshots and safe forwarding metadata. This is the correct first failing test because it anchors the persistence and privacy contract before handlers, routes, read-path enforcement, and Flutter UI build on it.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | API / Tests | The documents intentionally define the synthetic moderation endpoint by semantics rather than an exact request/response JSON schema. This is acceptable for coding planning, but the coding plan should lock field names, validation errors, status codes, and sample payloads before tests are written. | `01-requirements.md` §16 API, `FR-009` through `FR-012`, `AC-019`, `AC-023`, `AC-036`, `02-acceptance-tests.md` `AT-005`, `AT-006`, `UT-009`, `IT-017` | In coding plan, add a concrete endpoint contract for `POST /v1/dev/moderation/ozone-events`. |
| DR-002 | Suggestion | API / Flutter | Warning response metadata is required and well tested, but its exact wire shape is not specified. This is not blocking because behavior and privacy constraints are clear; however, implementation planning should define a stable nullable/omitted `moderation` shape before AppView and Flutter tests are authored. | `01-requirements.md` `FR-017`, `FR-018`, `FR-022`, `AC-017`, `AC-018`, `AC-030`, `AC-039`, §15 Data / Persistence Impact; `02-acceptance-tests.md` `AT-008`, `UT-007`, `IT-015`, `REG-002` | In coding plan, specify post/profile moderation metadata fields and absence behavior for unmoderated content. |
| DR-003 | Suggestion | Risk / Tests | Performance coverage for moderation lookups is necessarily high-level at this stage. `IT-019` and `MAN-004` are appropriate, but the coding plan should translate “bounded query pattern” into an implementation-specific test strategy once query design is known. | `01-requirements.md` `NFR-004`, `AC-031`, `RISK-004`; `02-acceptance-tests.md` `IT-019`, `MAN-004` | In coding plan, choose how to verify no per-row remote calls and avoid obvious N+1 query behavior. |

## Traceability Review

- Planning to requirements: The confirmed Option A direction is preserved in `01-requirements.md` goals `G-001` through `G-007`, non-goals `NG-001` through `NG-009`, requirements `BR-001` through `RULE-006`, and the high-risk review status. The requirements reflect the user decisions from the clarifying questions, including profile/account reporting, dev synthetic endpoint gating, report detail limits, generic warning copy, self-report rejection, trusted source DIDs, same-source negation, notification omission, and no full forwarding-payload persistence.
- Requirements to acceptance criteria: Every Must requirement has at least one linked acceptance criterion in `01-requirements.md` §12. The acceptance criteria are externally verifiable through API responses, database rows, route/config behavior, read-path results, and Flutter UI rendering. Should requirements `FR-023`, `FR-024`, `NFR-003`, and `NFR-004` are also covered.
- Acceptance criteria to tests: `02-acceptance-tests.md` covers `AC-001` through `AC-046` with acceptance, unit, integration, regression, and manual test IDs. Test cases consistently reference requirement IDs and acceptance criteria IDs, and the handoff explicitly identifies the first failing test and implementation order.

## Coverage Review

- Must requirements covered: Yes. `BR-001` through `BR-005`, `FR-001` through `FR-027`, `NFR-001` through `NFR-005`, and `RULE-001` through `RULE-006` appear in the coverage matrix with linked test IDs.
- Missing or weak coverage: No blocking missing coverage identified. Non-blocking notes are limited to contracts that the coding plan should make concrete: synthetic endpoint JSON shape, moderation metadata wire shape, and performance verification strategy.
- Manual-only coverage: None of the core Must behavior is manual-only. Manual checks are used appropriately for local UX smoke, localization/accessibility review, privacy/log spot checks, and performance sanity review where full automation is either incomplete or dependent on implementation details.

## Risk And Approval Review

- Risk level: High, correctly carried from requirements into the test specification.
- Review requirement: Satisfied by this document-review stage. Coding planning may proceed with the notes above.
- Approval notes: Privacy and safety risks are explicitly represented in requirements and tests: report details stay private, raw reasons are not displayed, synthetic routes are dev/flag/token gated, PDS records are untouched, hide/takedown enforcement covers multiple read surfaces, and regression tests protect unmoderated behavior.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start with `IT-001` from `02-acceptance-tests.md`: create failing migration/store tests for private post/profile report rows with canonical subject snapshots, normalized optional details, forwarding status, device ID, timestamps, and safe audit metadata.
- Blocking issues: None identified.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Define concrete wire contracts during coding planning before tests are written for new report endpoints, the dev synthetic endpoint, accepted report responses, error envelopes, and moderation metadata.
- Keep implementation slices test-first and in the order recommended in `02-acceptance-tests.md` §11.
- Preserve the architectural boundaries: reports and moderation outputs are AppView Postgres data, Flutter never talks directly to PDS for happy-path reads, and this MVP must not write report records or moderation side effects to PDS repositories.
- Do not modify lexicons for this feature; lexicon changes are explicitly out of scope.
