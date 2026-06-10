# Document Review: Flutter Project Post Models And Providers

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-06-10
Risk level: Medium

## Summary

`01-requirements.md` and `02-acceptance-tests.md` are consistent and ready for coding planning. The requirements preserve the selected direction: Flutter-only project-post model/data/provider plumbing, sealed known details variants with an unknown/raw fallback, AppView-only reads/writes, no UI, no lexicon/AppView/backend changes, and no new dependencies. The acceptance test specification traces Must requirements through acceptance criteria and concrete automated tests, with only justified non-blocking manual checks for generated-file/dependency and file-layout review.

No blocking contradictions, missing Must coverage, or unresolved questions were found. Implementation may proceed from the recommended first failing test, `UT-001` in `app/test/projects/models/project_test.dart`.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk | The slice is medium risk because it touches generated mappers, `Post` wire parsing, create signatures, a new provider family, and live cache mutation behavior. The test plan covers these risks, but implementation should keep tight TDD loops and run generation/analyze/test before handoff. | `01-requirements.md` §§19, 22-23; `02-acceptance-tests.md` §§1, 6, 8, 11 | Non-blocking: start with `UT-001`, follow the suggested test order, then run `dart run build_runner build --delete-conflicting-outputs`, `flutter analyze`, and `flutter test` from `app/`. |
| DR-002 | Suggestion | Tests | `AT-007` combines Must and Should priorities because it covers Must requirements (`FR-008`, `RULE-002`) plus Should requirements (`FR-010`, `NFR-003`). This is understandable, but coding planning should preserve the distinction: project endpoint preservation/state shape are mandatory, while exact pagination parity is still expected but sourced from a Should NFR. | `01-requirements.md` rows `FR-008`, `FR-010`, `RULE-002`, `NFR-003`; `02-acceptance-tests.md` `AT-007` | Non-blocking: in the coding plan, map provider tests back to the specific Must vs Should behavior being proven. |

## Traceability Review

- Planning to requirements: The confirmed decisions in `01-requirements.md` §3 are reflected in goals, non-goals, requirements, edge cases, and assumptions. The selected Option A sealed-details approach appears in `FR-001` through `FR-004`; create plumbing is covered by `FR-005`, `FR-006`, `FR-011`, and `FR-012`; profile project list/provider support is covered by `FR-007` and `FR-008`; cache behavior is covered by `FR-009`; architecture/scope constraints are carried into `BR-002`, `RULE-001`, `RULE-002`, `NG-001` through `NG-007`, and `NFR-004`.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `RULE`, and `NFR` requirement has linked acceptance criteria in `01-requirements.md` §12 and detailed criteria in §13. Should items (`FR-010`, `NFR-003`) also have appropriate criteria without blocking the core data-plumbing path.
- Acceptance criteria to tests: `02-acceptance-tests.md` §2 maps each requirement to AC IDs and test IDs. AC coverage is carried into acceptance scenarios (`AT-001` through `AT-009`), unit tests (`UT-001` through `UT-020`), integration tests (`IT-001` through `IT-010`), regressions (`REG-001` through `REG-007`), and manual checks (`MAN-001`, `MAN-002`). No orphaned acceptance criteria or Must requirements were identified.

## Coverage Review

- Must requirements covered: Yes. Must coverage includes project model parsing/serialization, sealed known detail variants, unknown/raw details preservation, `Post.project`, project create serialization and invalid project-plus-reply rejection, profile projects API/repository/provider support, project-aware cache updates, synthetic create response patching, common-only embroidery creates, AppView architecture constraints, split profile Posts/Projects semantics, constructor non-validation, camelCase JSON, forward compatibility, and codegen/bootstrap consistency.
- Missing or weak coverage: None blocking. `GAP-001` through `GAP-003` are valid non-blocking scope boundaries: UI end-to-end behavior belongs to later UI/composer workflows, package/dependency review is partially manual by nature, and arbitrary unknown-details authoring is intentionally not a supported create surface.
- Manual-only coverage: `MAN-001` for package layout and `MAN-002` for generated-file/dependency consistency are justified. They are paired with automated regression coverage (`REG-006`, `REG-007`) and the documented codegen/analyze/test commands.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Review recommended and now completed. No additional pre-coding approval is required by these documents.
- Approval notes: The highest-risk areas are mapper/discriminator behavior, create request signature changes, project-plus-reply guards, live cache fan-out, and provider pagination. These are all represented in tests and should be planned in the documented order.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Add `UT-001` in `app/test/projects/models/project_test.dart` for parsing/serializing `Project` + `ProjectCommon` camelCase JSON with `dart_mappable` value semantics.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Keep implementation Flutter-only: no UI, no AppView/backend changes, no lexicon changes, no migrations, and no dependency additions.
- Follow the suggested test order from `02-acceptance-tests.md` §11.
- Preserve the AppView architecture boundary: Flutter uses AppView JSON/HTTP and Craftsky session behavior, not direct PDS access or PDS tokens.
- After model/provider changes, run code generation before final analysis/tests:
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter analyze`
  - `cd app && flutter test`
