# Document Review: AppView Saved Posts

## Verdict

- Status: Approved
- Reviewer: Codex document review
- Date: 2026-07-21
- Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent, traceable, and ready for coding-plan work. The reviewed direction remains a normalized, owner-private AppView design with one optional folder per save, duplicate folder names distinguished by opaque-string IDs, saved-time ordering, exact comment/reply identity, current-policy hydration, and no PDS or Tap write dependency.

All 37 Must requirements link to acceptance criteria and automated tests. All 34 acceptance criteria appear in the test specification. The proposed 10 acceptance scenarios, 11 unit tests, 15 integration tests, and 7 regression tests cover persistence, API contracts, privacy, lifecycle, moderation/relationship policy, pagination, batch hydration, concurrency, migration reversal, and existing post-response behavior. No blocking product question, manual-only requirement, or untested Must requirement remains.

The four prior review findings have been incorporated into the requirements and test specification. The documents now require transactional descendant-save cleanup in the post indexer, extension of the shared `EngagementSummaries` hydration seam, the existing non-confidential base64url-JSON cursor contract, and folder IDs that remain wire-opaque rather than UUID-specific.

## Findings

None identified.

## Resolved Findings

| ID | Resolution | Updated References |
|---|---|---|
| DR-001 | The exact-target foreign key is explicitly limited to exact saves. The post-indexer deletion transaction must determine and remove saves for still-indexed descendant replies before or while deleting a root/intermediate ancestor, without deleting descendant public post rows. | `01-requirements.md` Q17, FR-013, AC-018, Section 15; `02-acceptance-tests.md` AT-005, IT-001, IT-009 |
| DR-002 | Viewer saved state is explicitly added to the existing shared `EngagementSummaries` batch seam, and every canonical post-shaped consumer must use it without a parallel or per-item path. | `01-requirements.md` Q18, FR-011, NFR-002, AC-015, AC-026; `02-acceptance-tests.md` IT-010, IT-014, IT-015 |
| DR-003 | Cursors retain the existing base64url-JSON opacity contract and are not treated as encrypted or confidential. Scope/keyset fields may be encoded, owner DID is omitted, and authentication remains the owner boundary. | `01-requirements.md` Q19, NFR-003, AC-025, AC-027; `02-acceptance-tests.md` AT-007, UT-003, IT-005 |
| DR-004 | Folder IDs are now explicitly opaque JSON strings. UUIDs remain an allowed storage choice, but client behavior and contract tests cannot depend on UUID shape. | `01-requirements.md` Q20, FR-004, AC-007, Section 16; `02-acceptance-tests.md` IT-001, IT-003, TD-010 |

## Traceability Review

- Planning to requirements: The initial private AppView request, single-folder decision, saved-time ordering, reply-context behavior, folder lifecycle, duplicate-name behavior, validation, mute shaping, response fields, status codes, and timestamp semantics are all preserved in Sections 3, 7, 8, 11, and 12 of `01-requirements.md`.
- Requirements to acceptance criteria: All 37 Must requirements—BR-001–BR-004, FR-001–FR-020, NFR-001–NFR-004, NFR-006, and RULE-001–RULE-008—link to at least one AC. The Should requirement NFR-005 is also covered. No requirement references a missing AC.
- Acceptance criteria to tests: AC-001–AC-034 all appear in `02-acceptance-tests.md`. Each test definition links back to requirement and AC IDs, and the coverage matrix contains every Must requirement.

## Coverage Review

- Must requirements covered: 37 of 37.
- Acceptance criteria covered: 34 of 34.
- Test design: 10 acceptance scenarios, 11 unit cases, 15 integration cases, and 7 regression cases.
- Missing or weak coverage: None identified. The prior lifecycle, hydration, cursor, and folder-ID ambiguities are now explicit requirements with automated test expectations.
- Manual-only coverage: None. The AppView-only behavior is practical to automate with `httptest`, fakes, existing response/store suites, and isolated real-Postgres schemas.
- Database verification caveat: Real-Postgres tests must run with `TEST_DATABASE_URL`; skipped database tests are not completion evidence for migration, concurrency, lifecycle, pagination, or query plans.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Review recommended by the requirements and test-design stages; this document review satisfies that recommendation.
- Approval notes: Approved. No additional product approval or document revision is required before coding planning. Implementation itself remains a separate action and is not authorized by this review.
- Highest-risk areas: owner isolation, composite owner/folder integrity, non-destructive folder deletion, exact versus ancestor post deletion, current-policy hydration, cursor compatibility, set-based viewer state, and concurrent mutation ordering.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Plan from `IT-001` in `appview/internal/db/saved_posts_migration_test.go`, after re-checking the current highest migration number. Define the two private tables, owner/post uniqueness, composite owner/folder integrity, delete actions, duplicate-name allowance, and ordering indexes before store code depends on them.
- Blocking issues: None.

## Notes For Next Stage

- Preserve the API distinction between an absent/omitted `folderId` and explicit `null`; the request decoder needs a tri-state representation rather than a plain nullable string.
- Keep unsave independent of current target resolution so a retry after indexed deletion still returns 204.
- Implement descendant-save cleanup in the existing post-indexer deletion transaction. Determine affected root/ancestor descendants before the event URI is removed, delete only their save rows, and retain descendant public post rows unless separate delete events arrive.
- Extend `EngagementSummaries` with saved viewer state and enumerate post, timeline, search, profile content, project, saved-list, comments/replies, notifications, and quote consumers; do not add parallel per-surface lookups.
- Use the existing `/v1/` policy registry: the save POST needs bounded optional JSON, reads/deletes need no-body enforcement, and every route needs authenticated device and read/write rate classification.
- Keep folder name validation centralized: trim, count Unicode characters, reject slash, backslash, and control characters, allow duplicate/case-variant names, and order case-insensitively with opaque ID as the tie-breaker.
- Treat cursors as scope-bound base64url-JSON transport tokens consistent with the existing envelope helper; include kind, scope, sort direction, and keyset values needed to reject incompatible reuse, omit owner DID, and do not add encryption.
- Treat folder IDs as opaque strings at the API and test boundaries even if the migration selects a UUID storage type.
- Ensure permanent target/root/ancestor deletion, owner-membership deletion, temporary policy retention, and session/device retention are separate lifecycle paths with independent tests.
- The full implementation gate remains `just test` with compose Postgres running; focused real-Postgres commands must set `TEST_DATABASE_URL` explicitly.
