# Document Review: Moderation Cases, Strikes, Appeals, And Suspension

## Verdict
Status: Approved
Reviewer: OpenCode
Date: 2026-09-09
Risk level: High

## Summary

The revised requirements and acceptance-test specification are consistent, traceable, testable, and ready for coding-plan work. The review covered the embedded initial request and discovery context because no separate `00-initial-prompt.md` exists.

The revision resolves the prior DR-001 through DR-014 findings: projection-authoritative strike expiry, complete suspended capability categories, moderation preference UI/API persistence, complete notification-event eligibility, a source-neutral current-stage adapter boundary, deterministic no-backfill legacy treatment, moderator-confirmed appeal activation, explicit log privacy, concrete operational alerts, bidirectional traceability, precise reason/strike wording, secure automatic account activation, and separate backend/Flutter notification tests.

All 68 Must requirements and all 48 acceptance criteria have concrete test paths. Coding planning may proceed. Because the feature remains high risk, the user must explicitly approve the requirements and test contract before implementation begins.

## Findings

None identified.

## Traceability Review

- Planning to requirements: The AppView-owned adjudication service, Retool-as-client boundary, private moderation storage, federated/PDS constraints, owner transparency, three-strike policy, appeal workflow, notifications, and staged future Ozone direction are preserved.
- Requirements to acceptance criteria: BR-001 through BR-007, FR-001 through FR-036, NFR-001 through NFR-008, and RULE-001 through RULE-017 each link to at least one externally verifiable AC.
- Acceptance criteria to tests: AC-001 through AC-048 each have one or more acceptance, unit, integration, regression, or justified manual verification paths.
- Bidirectional links: Requirement rows, AC Requirement IDs, acceptance scenarios, and the coverage matrix are aligned.

## Coverage Review

- Must requirements covered: 68 of 68.
- Acceptance criteria covered: 48 of 48.
- Missing or weak coverage: None blocking. Concrete route names and credential configuration remain coding-design details under fixed behavioral contracts.
- Manual-only coverage: Retool configuration/privileges, real email-app behavior, provider/OS presentation, and final visual/accessibility review are appropriately manual. Their deterministic application boundaries are automated.
- Automation targets: Go policy/unit tests, PostgreSQL integration tests using the existing `testdb` conventions, HTTP route tests, notification/push tests, Flutter repository/model/widget/router tests, migration tests, and observability threshold tests are practical for this repository.
- Release evidence: `just appview-check` is required; skipped PostgreSQL tests or `just appview-test-unit` alone are insufficient.

## Risk And Approval Review

- Risk level: High, unchanged.
- Review requirement: Document review is complete and approved for coding planning.
- Approval notes: Explicit user approval of the revised requirements and acceptance-test contract remains required before implementation. Merge-blocking tests identified in `02-acceptance-tests.md` must pass before handoff.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Design the persistence schema and source-neutral adjudication interfaces around `IT-020`, the recommended first failing test, then bind the fixed suspended-capability policy to the concrete route catalogue.
- Blocking issues: None for coding planning. Implementation remains gated on explicit user approval.

## Notes For Next Stage

- Preserve stable requirement, AC, and test IDs in `04-coding-plan.md` and map each implementation slice back to them.
- Start with legacy-safe migrations and database invariants before concurrent report grouping and adjudication effects.
- Keep the stored enforcement projection authoritative until the expiry worker's atomic commit; do not add request-time wall-clock restoration.
- Generate an exhaustive route decision table from the current catalogue and fail closed for every unclassified mutation.
- Keep notification intent creation inside adjudication transactions and provider suppression in dispatch.
- Reuse the existing opaque account-subscription binding and exact retained-account activation flow; do not add raw DIDs to provider payloads for routing.
- Treat live Ozone payload mapping and ingestion as Stage 3, outside this coding plan.
