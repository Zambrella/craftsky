# Document Review: Business Profiles

## Verdict
Status: Approved
Reviewer: OpenCode
Approver: User
Date: 2026-08-28
Risk level: High

## Summary

The revised requirements consistently follow the selected architecture: private AppView-authoritative account type, an optional public declaration with embedded products, and first-class public event records. Scope boundaries, security concerns, lifecycle behavior, and non-goals are thorough. The acceptance specification maps all 48 Must requirements and all 43 acceptance criteria to proposed tests.

The revisions resolve the owner-management, external-reference, validation, visibility, temporal, projection-order, and traceability findings below. The user explicitly approved the full revised contract on 2026-08-28. The documents are ready for coding planning and TDD implementation, with the High risk controls retained.

## Prior Findings

The following rows preserve the initial review history. Every required action was resolved in the revised `01-requirements.md` and `02-acceptance-tests.md` or explicitly approved by the user; none remains an active blocker.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Critical | Risk | The requirements remain Draft and explicitly require approval, while the acceptance document records that implementation cannot start without explicit high-risk approval. No approval is recorded. | `01-requirements.md` §22-23; `02-acceptance-tests.md` §1, GAP-006 | Resolve the findings below, record reviewer/date/approval in the workflow documents, and obtain explicit approval before coding planning continues. |
| DR-002 | Critical | Requirements | The owner all-events management surface is mandatory, but its route, authorization details, ordering/pagination bounds, response shape, and closed suppression reason-code catalog are undefined. The requirements call this a non-blocking coding detail while the acceptance specification correctly marks it as blocking. | FR-019; AC-023, AC-029; `01-requirements.md` §16, §21; AT-007; IT-010; GAP-001 | Define the route and wire contract, owner authorization, order and pagination, and complete bounded reason-code catalog. Align FR-019, AC-023, AC-029, AT-007, IT-010, and GAP-001. |
| DR-003 | Critical | Requirements | The exact reproducible pin for `community.lexicon.location.address` is a Must but remains undecided. Lexicon generation and drift tests cannot be finalized without it, and the requirements and test document disagree over whether this is blocking. | FR-006; NFR-001; AC-041; `01-requirements.md` §21; AT-013; IT-016; MAN-001; GAP-002 | Resolve the mechanism in the required ADR, including immutable source identity/version or digest, local resolution behavior, and drift detection. Update both documents to treat the decision consistently. |
| DR-004 | Important | Requirements | Location scope is internally inconsistent. FR-006 defines declaration location, but AC-013 says invalid location must not suppress the containing “profile/event”; no event requirement or event schema field defines structured location. | FR-006; AC-013; EC-010, EC-011; IT-004, IT-008 | If events do not have structured location, remove event language from AC-013 and event tests. If they do, define the event field, schema, API shape, bounds, and hydration rules. |
| DR-005 | Important | Security | Outbound destination validation is incomplete outside the primary action. Product URI has no explicit scheme, credential, fragment, or length policy. Event and registration links require HTTPS but do not state credential and byte-limit rules. “Simple mailto recipient” and country validity also lack exact grammars. | FR-006, FR-007, FR-008, FR-015; AC-012, AC-015, AC-016, AC-024; UT-006, UT-009, UT-010, UT-018; `01-requirements.md` §17 | Define whether country accepts assigned ISO codes or any two ASCII letters; define email/percent-encoding/Unicode policy; define product URI schemes and limits; and state whether event links share credential and 2048-byte restrictions. |
| DR-006 | Important | Requirements | Money validation is not deterministic. “Canonical decimal” does not specify whether currency scale is exact or a maximum, so values such as USD `1`, `1.0`, and `1.00` have no defined outcome. The source/version and withdrawn-currency policy for the known ISO 4217 table are also unspecified. | FR-010; AC-018; RISK-012; ASM-009; UT-012, UT-013 | Define an exact amount grammar with accepted/rejected examples, scale semantics, the currency data source/version, and treatment of withdrawn and exceptional minor-unit currencies. |
| DR-007 | Important | Coverage | The Must behavior “`isAllDay` defaults false” has no effective acceptance assertion. AC-020 does not mention the default, and UT-014 does not include omitted `isAllDay` as an input or expected result. | FR-012; AC-020; UT-014 | Add the default to AC-020 or a new stable AC, then test omitted `isAllDay` through validation and hydrated HTTP output. |
| DR-008 | Important | Requirements | Event update ownership of `createdAt` is ambiguous. FR-025 can be read as rejecting only a changed value; UT-023 rejects any supplied value, including an identical one. | FR-025; AC-038; AT-005; UT-023 | Specify whether update bodies must omit `createdAt`, whether an identical supplied value is accepted, and the exact validation error. Align all affected tests. |
| DR-009 | Important | Requirements | Direct event GET behavior is underspecified. FR-016 frames item GET as owner CRUD while FR-017 defines public direct reads. Outcomes are unclear for an owner who is regular or departed, blocked visitors, moderated records, over-duration records, and other suppressed records. | FR-016, FR-017, FR-019, FR-024; AC-025 through AC-029; AT-005, AT-007; IT-007, IT-008, IT-010; AC-031 | Define item-GET audiences and route-level outcomes for owner versus visitor, including whether suppression returns not found, a management representation, or a redacted shape. Define blocked event list/direct behavior explicitly. |
| DR-010 | Important | Tests | Permanent deletion tests do not assert the complete required order. They assert events before declaration and account type before membership, but not declaration before account type. | FR-022; RULE-006; AC-035; AT-010; IT-015; REG-007 | Add one ordered-effect assertion for events → declaration → private account type → membership, including retry and absent-membership cases. |
| DR-011 | Important | Traceability | Several acceptance scenarios cite criteria whose full behavior is not exercised by the scenario. Examples include independent unsafe hydration and validation boundaries in AT-003, write validation and createdAt rejection in AT-005, raw suppression detail in AT-007, complete deletion ordering in AT-010, declaration replacement semantics in AT-011, and destination grammar in AT-014. | AT-003: AC-013, AC-016 through AC-018; AT-005: AC-019, AC-022, AC-024, AC-038; AT-007: AC-023; AT-010: AC-035; AT-011: AC-036; AT-014: AC-015 | Add the missing scenario steps or remove the AC reference from that acceptance scenario and rely on accurately scoped unit/integration cases. Each test should claim only behavior it executes. |
| DR-012 | Important | Traceability | Some lower-level test references are inaccurate. UT-005 tests product-title bounds but omits FR-008/AC-016. UT-020 tests FR-023/AC-036 replacement behavior without citing them. UT-015 cites RULE-011 without exercising its catalog/mode rules. IT-004 cites first-party rejection criteria although its setup describes independent projection/hydration. | UT-005, UT-015, UT-020, IT-004 | Correct references or split mixed test cases so each row has an action and expected result for every cited requirement and criterion. |
| DR-013 | Important | Traceability | The coverage matrix is not exhaustive relative to individual test declarations. It omits tests including UT-004, IT-017 through IT-020, and several regressions; it lists IT-007 under NFR-005 without an image assertion and omits UT-009 under NFR-007. | `02-acceptance-tests.md` §2 versus UT-004, UT-009, IT-007, IT-017 through IT-020, REG-001, REG-003, REG-004, REG-007 | Rebuild the matrix from the corrected test rows and reconcile every listed test ID in both directions. |
| DR-014 | Important | Coverage | AC-017 requires the schema to permit 20 products, but UT-011 exercises only four/five and duplicates. IT-016 verifies schema maxima only under AC-041. | AC-017; UT-011; IT-016; TD-006 | Add a schema-validation case for 20 products and cite AC-017 from that test. |
| DR-015 | Important | Requirements | Out-of-order convergence is not defined for different CIDs on the same URI. AT-009 inaccurately describes create and update for the same URI/CID, while a real update changes CID. Duplicate delivery is covered, but stale create/update arrival after a newer update is not. | FR-021; AC-032; RISK-006; AT-009; IT-005, IT-006 | Define stale and reordered operation semantics, then test duplicate delivery per CID, an update with a distinct CID, and stale/reordered create/update/delete delivery. |
| DR-016 | Important | Tests | AC-030’s “without N+1 reads” condition is not backed by a mandatory measurable assertion. IT-011 allows query-count or plan inspection only “where supported.” | FR-020; AC-030; AT-008; IT-011 | State a measurable bulk-loading/query-count expectation and require an unconditional SQL-call-count, query-plan, or equivalent assertion for list hydration. |
| DR-017 | Suggestion | Requirements | AC-040 is unclear whether Craftsky must display a seller-authored/staleness disclaimer or merely avoid adding commerce guarantees. Current tests can pass vacuously by finding no forbidden fields or strings. | RULE-004; AC-040; AT-003; IT-020; REG-008 | Identify the exact contract artifact to inspect and clarify whether affirmative seller-authored wording is required or only the absence of inventory/checkout semantics. |

## Traceability Review

- Planning to requirements: The selected Option C is preserved. Goals, non-goals, public/private data boundaries, membership independence, chronological serving, and no-pay-to-play principles remain consistent with discovery.
- Requirements to acceptance criteria: Every Must row links to at least one acceptance criterion. The revisions define the `isAllDay` default, owner-management contract, external pin, direct-read visibility outcomes, validation grammars, and stale operation semantics.
- Acceptance criteria to tests: AC-001 through AC-043 are mapped to accurately scoped acceptance, unit, integration, regression, and manual targets in the revised matrix.

## Coverage Review

- Must requirements covered: 48 of 48 have at least one test ID or approved documented risk.
- Resolved coverage: The revised tests cover the `isAllDay` default, 20-product schema ceiling, complete permanent-deletion ordering, stale/reordered CID convergence, owner management, direct/blocked event outcomes, deterministic money, and outbound-link validation.
- Manual-only coverage: ADR/schema evolution approval and telemetry configuration review are appropriately manual supplements, but both also have automated drift/canary targets. No Must requirement relies solely on a manual check.

## Risk And Approval Review

- Risk level: High.
- Review requirement: Completed.
- Approval notes: Approved by the user on 2026-08-28. Durable lexicons, an external schema dependency, public contact/location data, broad response-shape changes, optimistic concurrency, moderation, and permanent deletion justify retaining the High rating and all specified automated/manual controls.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Use approved `04-coding-plan.md`; begin TDD with UT-001, followed by IT-001.
- Blocking issues: None. The required ADR remains a sequenced prerequisite to the lexicon implementation slice, not to the initial account-type TDD slice.

## Notes For Next Stage

- Preserve approved requirement, acceptance-criteria, and test IDs during implementation.
- Keep the coverage matrix aligned if an approved test target must move.
- Keep Flutter outside this implementation slice.
- Do not begin lexicon implementation before the required ADR fixes the external reference and durable schema decisions.
- Retain `UT-001` followed by `IT-001` as the first TDD slice.
