# Document Review: Flutter Saved Posts

## Verdict

Status: Approved with notes
Reviewer: Codex document review
Date: 2026-07-21
Risk level: Medium

## Summary

The requirements and acceptance-test specification are ready for coding-plan work. The selected direction is consistent from the embedded initial request through the recommended approach, requirements, acceptance criteria, and test layers. All 46 Must requirements link to acceptance criteria, all 32 acceptance criteria have concrete automated coverage, privacy and multi-account risks have explicit verification paths, and no blocking product question remains.

The review found two non-blocking precision issues: the coverage matrix sometimes lists tests that do not repeat the same requirement ID in their detailed row, and folder-list reconciliation after create/rename/delete is described by outcomes but not isolated as a focused pagination test. These should be made explicit in the coding plan so implementation does not lose traceability or leave a stale opaque-cursor state. Two additional test-writing suggestions do not affect readiness.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Traceability | The coverage matrix contains test-to-requirement links that are not repeated by the detailed test case. Representative examples are BR-001 → IT-004/IT-005, BR-002 → IT-007/IT-008, FR-003 → IT-004, and the meta-coverage rows NFR-005/NFR-006 → broad test ranges. Every detailed test still has valid requirement and acceptance-criteria links, and every Must requirement is covered elsewhere, so this is not missing coverage; it is a precision mismatch between the matrix and the detailed rows. | `02-acceptance-tests.md` §2, §4–§6; BR-001, BR-002, FR-003, NFR-005, NFR-006 | In the coding plan, use the detailed test rows as the authoritative ownership links. When `02-acceptance-tests.md` is next revised, either add the claimed requirement ID to the detailed test or remove that test from the corresponding matrix row. Do not copy the mismatch into test names or implementation traceability. |
| DR-002 | Important | Tests / Risk | Folder create, rename, and delete can change alphabetical position or invalidate a partially loaded opaque folder cursor. The documents require server order, no duplicates, no dangling selection, and state convergence, while IT-003 tests pagination without mutations and IT-008 mentions mutations without stating the cursor-reset/deduplication outcome explicitly. The product outcome is clear, but the state transition needs a focused test. | FR-005, FR-010, FR-013, FR-018; AC-005, AC-009, AC-010, AC-017, AC-031; EC-006; AT-005, AT-007, AT-009; IT-003, IT-008 | The coding plan should specify that successful create/rename/delete reconciles or restarts the affected folder list from server order, invalidates any unsafe cursor, deduplicates by opaque ID, updates the open folder title/selection where applicable, and preserves the overview hierarchy/scroll behavior. Add a focused provider/widget test for these transitions. |
| DR-003 | Suggestion | Tests | AT-001 uses one scenario outline with a fixed `<saved>` fixture but then exercises both the unsaved and saved tap branches. The intended behaviors are clear, but a literal implementation of the scenario would require an unstated state transition. | `02-acceptance-tests.md` AT-001; AC-001, AC-002, AC-032 | Split the widget implementation tests into initial rendering/placement, unsaved-tap chooser, and saved-tap optimistic-unsave cases, or add an explicit state change between branches. Preserve AT-001 as the umbrella acceptance ID. |
| DR-004 | Suggestion | Risk | “No new unapproved runtime dependency” is primarily a source/package-diff gate rather than behavior that Flutter or Go runtime tests can prove by themselves. | NFR-006; AC-027; AT-013 | Add an explicit `pubspec.yaml`/lockfile and Go module diff review to the coding-plan completion gate. Keep analysis and full test suites as complementary evidence. |

## Traceability Review

- Planning to requirements: The workflow has no separate `00-initial-prompt.md`, but `01-requirements.md` embeds the initial request, codebase discovery, confirmed decisions, rejected options, recommended direction, goals/non-goals, risks, and review status. The recommended saved-post feature area, Settings-only route, reusable `PostSummary`, and atomic AppView delete mode are carried through without contradiction.
- Requirements to acceptance criteria: All 46 Must requirements reference at least one acceptance criterion. The requirement-table links and the acceptance-criteria-table backlinks are bidirectionally consistent. The two Should requirements are also covered.
- Acceptance criteria to tests: All AC-001–AC-032 appear in concrete acceptance, unit, integration, regression, or manual sections. Every AT/UT/IT/REG/MAN case references requirement IDs and acceptance-criteria IDs. DR-001 records matrix-to-detail precision mismatches, not missing behavior.

## Coverage Review

- Must requirements covered: 46 of 46. Coverage includes 13 acceptance scenarios, 14 unit cases, 12 integration cases, and 9 regression cases, with practical Flutter and AppView automation targets.
- Missing or weak coverage: No Must behavior is missing. Folder mutation plus pagination reconciliation should be isolated per DR-002, and test-to-requirement labels should be normalized per DR-001.
- Manual-only coverage: None. MAN-001 and MAN-002 complement automated AC-023 semantics, focus, tap-target, and constrained-layout assertions for real assistive technology and platform font behavior.

## Risk And Approval Review

- Risk level: Medium. The persistent risks are private multi-account state crossing an account switch, atomic delete-with-saves rollback/privacy, opaque pagination during mutations, and regression-sensitive extraction across quote, notification, and saved surfaces.
- Review requirement: Review is recommended for medium risk but does not require a separate high-risk approval gate. This document review satisfies the recommended pre-plan review.
- Approval notes: Requirements were previously reviewed by the product owner, blocking questions are explicitly `None`, and the test design supplies automated paths for each high-impact risk. Real-Postgres tests must run with `TEST_DATABASE_URL`; a skipped database case is not pass evidence.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Design the first red test as UT-001 in `app/test/feed/models/post_test.dart` for decoding, defaulting, copying, and explicitly clearing `viewerHasSaved` / `viewerSavedFolderId`. Then establish typed saved/folder API and repository contracts before provider or widget work.
- Blocking issues: None.

## Notes For Next Stage

- Keep the documented test order: wire models → typed API/repository → account-scoped saved-state seam → bookmark/chooser → paginated overview/folder pages → atomic AppView deletion → `PostSummary` extraction → account-race/regression/full gates.
- Add the focused folder-mutation pagination/reconciliation test from DR-002 to the provider phase rather than deferring it to broad page tests.
- Treat `SavedPostState` keyed by active account plus canonical post URI as the single mutation/reconciliation seam; do not fork save/move/unsave rules into individual screens.
- Preserve the server boundary: delete-with-saves is one authenticated folder DELETE and never a Flutter loop over hydrated rows.
- Keep folder IDs/names, saved URIs, cursors, and owner-target pairs out of route names, logs, Sentry, traces, metrics dimensions, and user-facing raw errors.
- Use the existing Dio adapter, repository fake, Riverpod provider-container, widget semantics, typed router, `httptest`, and real-Postgres patterns. Do not add a runtime dependency unless the plan documents an unavoidable need and receives approval.
- Completion gates should include focused Flutter/Go tests, `just app-analyze`, `just app-test`, `just fmt`, `just test`, explicit confirmation that real-Postgres cases ran rather than skipped, and dependency-file diff review.
