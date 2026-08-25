# Document Review: Link Previews (Website Card Embeds)

## Verdict

Status: Approved
Reviewer: OpenCode and User
Date: 2026-08-25
Risk level: High

## Summary

The revised documents preserve the selected AppView-fetch/Flutter-select/standard-record direction and are internally consistent enough for coding-plan work. Every Must requirement links to acceptance criteria and automated tests. The revision defines the thumbnail JSON wire representation, deterministic duplicate-fragment behavior, standard-only preview authoring/scheduling with unchanged photo-required projects, immediate no-debounce fetching, silent Flutter 429 handling, Undo expiry, and durable approval/governance evidence.

No `00-initial-prompt.md` exists in this workflow folder. `01-requirements.md` contains the initial request, discovery findings, decisions, alternatives, and selected direction, so the missing standalone prompt does not create a review gap.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-010 | Important | Risk / Governance | The user prefers no ADR because the app is pre-production, but repository-level `AGENTS.md` unconditionally requires an ADR for every `lexicon/` change. The reviewed documents correctly retain the ADR and `atproto-lexicon` review gates. | `AGENTS.md` Architectural Rule 4; FR-006; AC-007; IT-008 | Keep the ADR in the coding and implementation plans unless the repository policy is explicitly amended before lexicon work begins. |
| DR-011 | Suggestion | Coding Readiness | Exact bounded error codes, fixed User-Agent text, and `Accept-Language` policy were deferred to coding planning. | `01-requirements.md` section 21; `02-acceptance-tests.md` GAP-002, GAP-003; `04-coding-plan.md` section 2 | Resolved: the coding plan defines the taxonomy, fixed User-Agent, and omitted `Accept-Language`. |
| DR-012 | Suggestion | Verification | The Should-level 5-second page-phase p95 target requires a production-like manual measurement; deterministic repository tests can prove only bounds and non-blocking behavior. | NFR-001; MAN-003; GAP-001 | Keep MAN-003 as a production-enablement check and retain privacy-safe duration metrics. |

## Resolved Findings

- DR-001: FR-001/AC-001 now define padded RFC 4648 base64 in JSON with a 1,000,000 decoded-byte limit; UT-014, IT-001, and TD-004 verify round-trip, malformed, and overflow behavior.
- DR-002: FR-006/AC-007 and IT-008 now require a numbered ADR, recorded `atproto-lexicon` review, generated union output, validation, and compatibility evidence.
- DR-003: Coding-plan inspection exposed the existing mandatory-photo project rule. The user selected standard-only preview authoring. FR-016, AC-018, AT-002, AT-005, UT-018, IT-015, IT-016, and TD-009 now cover standard scheduled cards while preserving/rejecting external in photo-required project schedules.
- DR-004: Q1/Q4, FR-001, FR-010, RULE-004, AC-001/AC-010, EC-004, AT-001, UT-001, IT-001, and IT-012 now define fragmentless AppView requests, redirect-only response fragments, and client-side first-occurrence source-fragment inheritance.
- DR-005: IT-012 now cites FR-015/AC-015 and appears in the FR-015 coverage row.
- DR-006: AT-001, UT-003, and IT-012 require the first eligible call without advancing fake time.
- DR-007: Requirements section 22 records the initial user approval, reviewer/date, revision state, and re-approval requirement.
- DR-008: AT-002 and UT-003 cover Undo expiry, continued session dismissal, and restoration in a new session.
- DR-009: GAP-001 is no longer listed as a test ID and remains documented as a measurement gap.
- DR-011: `04-coding-plan.md` resolves the bounded error taxonomy, fixed User-Agent, and `Accept-Language` policy.

## Traceability Review

- Planning to requirements: The embedded discovery record captures all fourteen decisions, rejected alternatives, scope boundaries, architecture constraints, and identified risks. The selected direction is preserved.
- Requirements to acceptance criteria: All 28 Must requirements link to at least one acceptance criterion. The two Should requirements, FR-014 and NFR-001, are also linked.
- Acceptance criteria to tests: AC-001 through AC-023 have practical automation targets. Manual checks supplement rather than replace Must-level automated coverage.
- Test identifiers: Acceptance, unit, integration, regression, manual, data, and gap IDs are monotonic within their categories and reference valid requirement/criterion IDs.

## Coverage Review

- Must requirements covered: 28 of 28.
- Missing or weak coverage: None blocking. Exact taxonomy/header values are deferred with explicit test update points.
- Manual-only coverage: No Must behavior is manual-only. MAN-001 supplements platform-neutral rendering/semantics tests, MAN-002 verifies deployed direct-egress configuration after transport automation, and MAN-003 measures the Should-level p95 target.
- Test practicality: Go targets match colocated `testing`, `httptest`, injected seams, and real-Postgres conventions. Flutter targets match `flutter_test`, Riverpod, `DioAdapter`, existing composer harnesses, and deterministic completer/fake-time patterns.

## Risk And Approval Review

- Risk level: High, unchanged.
- Review requirement: Complete. The user explicitly approved the revised requirements, acceptance tests, document review, and coding plan on 2026-08-25.
- Approval notes: Implementation may proceed under the approved coding plan. Production enablement remains gated on reviewed direct pinned egress.

## Implementation Readiness

- Ready for coding planning: Yes.
- Ready for implementation: Yes.
- Recommended first step: Follow `04-coding-plan.md` and start with the lexicon contract/governance step, then UT-009 for the secure resolver/validator/pinned-dialer boundary.
- Blocking issues: None. The High risk classification still requires the full documented verification and implementation review.

## Notes For Next Stage

- Preserve requirement, acceptance-criteria, and test IDs in `04-coding-plan.md`.
- Keep metadata extraction separate from the SSRF-safe fetch transport.
- Treat DNS resolution, complete-answer validation, and dialing as injectable interfaces so mixed answers, rebinding, redirects, and exact pinned destinations remain deterministic.
- Decode base64 only at the Flutter API boundary and enforce the decoded-byte cap before retaining thumbnail bytes.
- Parameterize standard scheduled external implementation/tests across metadata-only/thumbnail variants; preserve project required-photo behavior and reject project external payloads.
- Before editing `lexicon/`, invoke `atproto-lexicon`, create the required numbered ADR, update external lexgen inputs, run `just lexgen`, and retain generated output with the schema change.
- Recommended first failing test: UT-009 in `appview/internal/linkpreview/transport_test.go` for pre-resolution rejection of forbidden schemes, userinfo, ports, and non-public literals.
