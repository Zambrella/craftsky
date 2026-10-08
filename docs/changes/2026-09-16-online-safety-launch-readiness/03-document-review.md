# Document Review: Pre-Launch Online Safety Readiness

## Verdict

Status: Approved with notes

Reviewer: OpenCode document review

Date: 17 September 2026

Risk level: High

## Summary

The requirements and acceptance-test specification are internally consistent,
preserve the confirmed scope, and provide complete ID-level traceability. All 61
requirements, including every Must requirement, link to acceptance criteria and
tests. All 51 acceptance criteria have a concrete automated, acceptance, regression,
or manual verification path. The test design correctly avoids real illegal imagery
and does not invent the private IWF protocol.

Bounded coding planning may proceed. On 17 September 2026, the accountable owner
confirmed that specialist/provider approval of index-time-only containment had been
obtained; its documentary reference remains mandatory implementation-completion and
launch evidence. The owner also explicitly authorized a provider-neutral coding
plan while the private IWF contract is unavailable. That plan must exclude the
production IWF adapter and must not invent provider payloads or semantics. The
remaining unresolved legal and operational items are serious launch blockers, but
can be represented as configurable values, role assignments, deployment controls,
or external evidence without changing the bounded architecture.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk / Approval | The accountable owner confirmed that specialist/provider approval for index-time-only containment was obtained, resolving the boundary for coding planning. The reviewer, date, evidence reference, and conditions remain unrecorded, so implementation completion and launch cannot rely on the approval yet. | `01-requirements.md` Q2, Q12, Option A, RISK-002, Section 21; BR-002, RULE-005; AC-002, AC-035; `02-acceptance-tests.md` GAP-002, MAN-011 | Attach and verify the approval evidence before implementation completion or launch. If its conditions reject or alter Option A, return to requirements and test design. |
| DR-002 | Important | Risk / Integration | IWF eligibility, protocol, result semantics, policy/corpus identity, benign test contract, retention, security, and rate limits remain unknown. The owner authorized only a provider-neutral coding plan while those facts are unavailable. | `01-requirements.md` Q3, FR-002, FR-007, FR-008, NFR-005, RULE-004, RISK-001, ASM-001 through ASM-003, Section 21; AC-027, AC-028; `02-acceptance-tests.md` GAP-001, MAN-009 | Exclude the production IWF adapter from the coding plan. Obtain IWF documentation/access and re-review the adapter contract before implementation completion or launch. |
| DR-003 | Important | Requirements / Operations | The intimate-image response target and coverage are unresolved. The test design properly treats the target as configuration, but implementation and operational acceptance cannot select a value or prove deadline behavior until specialist review finishes. | `01-requirements.md` Q10, FR-021, ASM-009, Section 21; AC-021; `02-acceptance-tests.md` AT-013, UT-011, IT-027, MAN-004, GAP-003 | Obtain and record the approved target and coverage before implementing production deadline configuration or claiming AC-021 complete. |
| DR-004 | Important | Security / Operations | The eligible NCA administrator, named helpers, training, availability, and exact permissions are not assigned. Role boundaries are clear, but production authorization and coverage cannot be accepted from abstract roles alone. | `01-requirements.md` Q7, Q24, RULE-007, RULE-009, ASM-010, Section 21; AC-029, AC-036, AC-050, AC-051; `02-acceptance-tests.md` MAN-002, MAN-003, MAN-013, GAP-004 | Name and authorize the roles, approve the permission matrix and training, then execute the coverage and restricted-access checks before launch. |
| DR-005 | Important | Integration / Operations | No mailbox provider/configuration has been selected that can prove attachment rejection or quarantine without preview or forwarding. This leaves the external reporting boundary operationally undefined. | `01-requirements.md` FR-018, FR-037, RULE-006, RISK-013, Section 21; AC-014, AC-015, AC-047; `02-acceptance-tests.md` AT-005, IT-024, MAN-006, GAP-005 | Select and validate a provider. Keep any coding plan behind a provider-neutral intake boundary until MAN-006 passes. |
| DR-006 | Important | Tests / Scope | The implementation owner and browser harness for the public PostHog consent control are not identified in the repository. The acceptance behavior is clear, but the coding planner cannot name the affected application or practical test target without guessing. | `01-requirements.md` FR-030, AC-030; `02-acceptance-tests.md` AT-017, IT-023, GAP-010 | Identify the public-site codebase and test harness, or explicitly classify this control as external to this repository with linked release evidence. |
| DR-007 | Important | Risk / Publication | Final operator/controller details, DPO position, worldwide jurisdiction review, and commercial-policy scope remain unresolved. These do not require speculative application architecture, but they block policy publication and worldwide launch. | `01-requirements.md` BR-001, BR-005, BR-006, FR-031, RISK-010, RISK-011, Section 21; AC-001, AC-026, AC-031; `02-acceptance-tests.md` AT-001, AT-012, MAN-007, MAN-010, MAN-014, GAP-006, GAP-007, GAP-012 | Resolve and approve the legal details and launch territories before policy publication. Keep placeholders and unsupported commercial claims out of published text. |
| DR-008 | Important | Tests / Retention | FR-029 covers systems outside AppView control, but the current automation target covers only implemented database/S3 records. Email, Sentry, PostHog, backups, and local drafts still need provider/device-specific deletion evidence. | `01-requirements.md` FR-029, AC-022; `02-acceptance-tests.md` IT-022, GAP-009 | Inventory each controller, define its enforceable deletion/hold mechanism and evidence owner, and add provider/device checks before AC-022 can pass. |

## Traceability Review

- Planning to requirements: Complete. The recommended index-time, fail-closed IWF
  direction, worldwide/UK launch, 16+ minimum, founder-led operation, email intake,
  existing moderation platform, and video/Ozone exclusions are preserved.
- Requirements to acceptance criteria: Complete. Every BR, FR, NFR, and RULE has at
  least one linked acceptance criterion; every Must requirement is externally
  verifiable or explicitly depends on identified manual evidence.
- Acceptance criteria to tests: Complete. All AC-001 through AC-051 are referenced
  by stable test IDs. Acceptance, unit, integration, regression, and manual checks
  consistently include requirement and acceptance-criterion links.

## Coverage Review

- Must requirements covered: 60 of 60.
- Additional Should requirements covered: NFR-005 is covered by UT-015 and MAN-009.
- Acceptance criteria covered: 51 of 51.
- Missing or weak coverage: No missing ID coverage. DR-006 and DR-008 identify test
  ownership and external-system evidence that are not yet practical automation
  targets.
- Manual-only coverage: No Must requirement is entirely manual-only, but launch
  approval, CSEA/NCA operations, AWS deployment controls, intimate-image response,
  threat/authority handling, mailbox behavior, privacy exercises, IWF validation,
  jurisdiction/age consistency, accessibility, staffing cover, and policy
  publication require manual evidence in addition to automated tests.
- Test data safety: Appropriate. The specification requires benign synthetic media,
  canary values, and no real illegal imagery, victim media, production provider
  responses, mailbox data, or authority credentials.

## Risk And Approval Review

- Risk level: High.
- Review requirement: This review approves only bounded coding planning. Review is
  required again before production-adapter work, implementation completion, or
  launch approval.
- Approval notes: The accountable owner approved `01-requirements.md` for
  acceptance-test design and bounded coding planning, confirmed the index-time
  boundary approval, and returned no annotations on `02-acceptance-tests.md`.
  Production IWF adapter work, implementation completion, and launch remain
  unauthorized.
- Launch blockers: DR-001 through DR-008 and GAP-001 through GAP-012 remain open.
- Bounded-plan constraints: treat index-time containment as the selected boundary;
  use only protocol-neutral scanner interfaces and deterministic stub behavior; do
  not plan production IWF payloads, result semantics, or credentials.

## Coding Plan Readiness

- Ready for coding planning: Yes, within the bounded scope.
- Recommended first step: Design the private schema and protocol-neutral scanner
  state machine around UT-001, UT-002, and IT-003, with the deterministic stub as
  the only concrete scanner implementation.
- Blocking issues: None for bounded coding planning. DR-001 and DR-002 remain
  blockers for production-adapter work, implementation completion, and launch.
- Deferred launch issues: DR-003 through DR-008 remain mandatory release evidence
  even if a later review allows bounded coding planning.

## Notes For Next Stage

- Preserve stable requirement, acceptance-criterion, and test IDs when recording
  decisions.
- Coding planning should begin with the private schema and protocol-neutral scanner
  state machine, following UT-001, UT-002, and IT-003.
- If pre-upload containment is required, return to requirements and test design;
  do not patch the coding plan around Option A.
- Do not encode an assumed IWF payload, uncertain result state, guessed corpus
  version, unapproved deadline, or specific mailbox behavior.
- Keep production-adapter completion, public policy publication, and public launch
  as separate gates with their own evidence.
