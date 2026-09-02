# Document Review: Flutter Business Profiles

## Verdict

Status: Approved

Reviewer: OpenCode

Date: 2026-08-30

Risk level: Medium

## Summary

The revised requirements and acceptance-test specification are consistent, traceable, and ready for coding planning. They preserve the confirmed integrated Flutter direction and approved AppView/PDS boundaries while resolving every finding from the first review.

The visitor Upcoming Events tab now has a stable loading/error/empty surface; owner Upcoming and History traversals explicitly use independent cutoffs without a cross-view snapshot promise; projection-lag handling compares CID identity without ordering opaque CIDs; and the image, model-scope, filter-admission, unknown-extension, accessibility, and privacy contracts are exact enough to implement without product-level guessing.

## Findings

None identified.

Resolved findings from the prior review:

| ID | Resolution |
|---|---|
| DR-001 | Products and Upcoming Events are always visible on normally visible business profiles, giving both surfaces stable owner/visitor empty states and Upcoming Events explicit initial, retry, incremental-error, and refresh-error transitions. |
| DR-002 | Upcoming and History freeze independent first-page cutoffs. Guarantees apply per traversal; transient cross-view overlap/omission is documented, and refresh/mutation restarts affected views. |
| DR-003 | Accepted overlays use pre-write, accepted, and different CID identity plus account/request generations. Create, update, delete, failure, divergence, and explicit reload behavior are defined without CID chronology. |
| DR-004 | FR-021 freezes one reusable `PostImageView` JSON object, its exact required/optional fields and types, safe MIME behavior, omission rules, mutation reconstruction, and all response surfaces. |
| DR-005 | `accountType` retention is limited to the Flutter full-profile model; compact identity mappers retain their existing unknown-field behavior. |
| DR-006 | FR-023 and AC-041 define unknown/empty/repeated filter failures, malformed and incompatible cursor failures, changing limits, and existing unknown-parameter behavior. |
| DR-007 | IT-014 and REG-011 explicitly cover AppView preservation of unknown declaration extensions for detail-only and product-only replacements. |
| DR-008 | Tests now name the closed diagnostic catalog, profile/business screen catalog, exact viewport/text-scale matrix, concrete network/launcher/observability adapters, and no-client-refilter assertion. |

## Traceability Review

- Planning to requirements: The integrated profile/settings slice, reversible account switching, summary-plus-About presentation, stable business Products and Upcoming Events tabs, in-app detail, combined Save reconciliation, two-view event management, and minimal AppView alignment are preserved.
- Requirements to acceptance criteria: All 37 Must requirements link to one or more of AC-001 through AC-041. Revised criteria precisely cover independent cutoffs, event-tab state transitions, CID-identity reconciliation, exact image JSON, and filter admission.
- Acceptance criteria to tests: All 41 acceptance criteria map to acceptance, unit, integration, or regression tests. The added IT-014/REG-011 path assigns unknown-extension preservation to AppView rather than Flutter.

## Coverage Review

- Must requirements covered: 37 of 37.
- Acceptance criteria covered: 41 of 41.
- Missing or weak coverage: None identified at document-design level.
- Manual-only coverage: None. MAN-001 and MAN-002 complement automated semantics, layout, adapter, and policy tests for real assistive technology and operating-system handoff behavior.
- Practicality: Flutter targets match existing widget, Riverpod, Dio-adapter, typed-router, fake-launcher, and account-boundary patterns. AppView targets match handler/unit and real-Postgres test patterns. Database-backed cases still require `TEST_DATABASE_URL`; skipped database tests are not release evidence.

## Risk And Approval Review

- Risk level: Medium. The work is broad user-visible CRUD and API projection behavior but adds no lexicon, migration, authentication, permission, persistence, or public eligibility change.
- Review requirement: Satisfied by this review.
- Approval notes: Coding planning may proceed. The planner should retain the exact API and reconciliation contracts rather than weakening them into implementation-defined behavior.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with failing AppView contract tests for FR-021/IT-002 and FR-023/IT-003, then implement Flutter models and state only after those wire prerequisites are fixed.
- Blocking issues: None.

## Notes For Next Stage

- Reuse one AppView `PostImageView` response type across profile products and all event response surfaces; do not create a second image wire format.
- Detect filter presence and repetition explicitly so an omitted filter remains distinct from `filter=`.
- Keep each owner event view's cutoff inside its opaque cursor; do not introduce a shared-cutoff mechanism.
- Never compare or sort CID strings. Implement the exact identity transitions in FR-022 and fence every completion by account and request generation.
- Keep unknown declaration-extension merge responsibility in AppView and complete-known-field responsibility in Flutter.
- Preserve public upcoming server authority: Flutter renders every AppView-returned item without lifecycle/status re-filtering.
- Keep the bounded AT-012 screen, viewport, supported-English localization, formatter-locale, semantics, adapter, and observability matrices intact during coding planning.
