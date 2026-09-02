# Document Review: Onboarding Experience

## Verdict
Status: Approved with notes
Reviewer: OpenCode
Date: 2026-08-31
Risk level: Medium

## Summary
The requirements and acceptance-test specification are consistent, traceable, and ready for coding-plan work. All confirmed product decisions are represented, every Must requirement links to acceptance criteria and automated coverage, and no blocking product or technical-design questions remain. The documented manual checks and post-implementation review remain necessary because platform gallery behavior, broad visual usability, OAuth callback changes, private completion persistence, and shared Instagram UI carry residual risk.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests | Real gallery permission/picker behavior and exhaustive responsive, localized visual behavior cannot be fully established by the automated suites. | `02-acceptance-tests.md` MAN-001, MAN-002, GAP-002, GAP-003 | Retain both manual checks as release evidence; this does not block coding planning. |
| DR-002 | Suggestion | Risk | The change combines medium-risk OAuth initialization, private persistence and cleanup, startup routing, atomic profile writes, and extraction of established Instagram UI. Test design covers these areas, but implementation should not be considered merge-ready on test completion alone. | `01-requirements.md` RISK-001 through RISK-011; `02-acceptance-tests.md` IT-001 through IT-009, REG-001 through REG-008 | Run the implementation-review stage against the approved workflow documents before handoff or merge. |

## Traceability Review
- Planning to requirements: The recommended step-scoped-save approach, account-wide completion authority, bounded Bluesky prefill, shared Instagram sections, sequential navigation, and eager best-effort OAuth projection are all represented in goals, requirements, non-goals, risks, and assumptions.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` has at least one linked acceptance criterion. Should requirements FR-011 through FR-013 are also covered.
- Acceptance criteria to tests: AC-001 through AC-043 are represented in the coverage matrix and linked to acceptance, unit, integration, regression, or justified manual tests.

## Coverage Review
- Must requirements covered: BR-001, BR-002; FR-001 through FR-010 and FR-014 through FR-024; NFR-001 through NFR-004; RULE-001 through RULE-007.
- Missing or weak coverage: None blocking. Cross-device behavior is represented by server integration coverage and reconstructed client state rather than a physical multi-device automated test.
- Manual-only coverage: No Must requirement is exclusively manual. FR-023 and NFR-001/NFR-002 have automated coverage supplemented by MAN-001 and MAN-002 for operating-system and visual behavior.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Implementation review is required before merge or handoff; manual gallery and responsive/accessibility checks remain release evidence.
- Approval notes: Product decisions are recorded as requirements, non-goals, assumptions, or explicit accepted risks. The direct OAuth projector remains best-effort, reuses canonical projection semantics, and does not replace Tap/backfill. Existing AppView read-before-write and `ExpectedCID` behavior is preserved while new client concurrency controls remain out of scope.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Design the injected OAuth profile-projector boundary and begin with UT-009, then IT-009, before planning the completion persistence/API and Flutter flow in the documented dependency order.
- Blocking issues: None.

## Notes For Next Stage
- Preserve package boundaries by injecting the canonical Bluesky projection capability into auth initialization rather than importing index implementation details into the auth package.
- Read the AppView API architecture specification before designing the authenticated onboarding-status routes.
- Include migration up/down, account-deletion cleanup, policy registration, camelCase wire shape, and authenticated DID derivation in the completion API design.
- Keep profile payload composition explicit: Flutter supplies the complete client-editable snapshot, while existing AppView read-before-write and server-side `ExpectedCID` behavior preserves applicable PDS fields.
- Treat Instagram work primarily as extraction into shared sections; retain settings-only import history and revocation and keep onboarding suggestion rows non-navigating.
- Sequence the plan around the test order in `02-acceptance-tests.md` section 11 and require a final implementation review against all three workflow artifacts.
