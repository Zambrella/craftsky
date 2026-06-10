# Implementation Review: Flutter Project Post Models And Providers

## Verdict

Status: Changes required  
Reviewer: gpt-5.5 implementation reviewer  
Date: 2026-06-10  
Risk level: Medium

## Summary

The implementation substantially matches the approved plan: project models live under `app/lib/projects`, `Post.project` parses AppView project payloads, create/list repository plumbing uses the AppView API, `userProjectsProvider` follows the existing cursor-accumulating provider pattern, and live cache helpers avoid profile Posts pollution for project posts. Static analysis and the full Flutter test suite pass.

One required test-coverage gap remains before handoff. The mutation cache implementation appears correct by inspection, but the planned Must coverage for project interaction branches and rollback behavior is incomplete.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Tests | Project interaction cache tests do not fully cover the Must cache-mutation matrix. Current tests cover successful project like/repost updates for a handle-keyed `userProjectsProvider`, but they do not exercise project unlike/unrepost branches, project cache rollback on repository failure, or DID-keyed live project cache entries for the project-specific create/delete/interaction paths. The code paths look symmetric and likely correct, but the approved acceptance tests explicitly call for project-cache consistency across delete, like, unlike, repost, unrepost, success/failure rollback, and handle/DID live cache keys. | `02-acceptance-tests.md` `AT-005`, `AT-008`, `IT-007`, `IT-008`, `IT-009`; `01-requirements.md` `FR-009`, `AC-008`, `AC-013`; `app/test/feed/providers/create_post_provider_test.dart`; `app/test/feed/providers/delete_post_provider_test.dart`; `app/test/feed/providers/toggle_post_interactions_provider_test.dart` | Add focused tests that cover project-specific cache behavior for both author handle and DID keys, and add failure rollback coverage for project like/repost mutations. Include unlike/unrepost project branches or otherwise explicitly cover both toggle directions for project caches. Rerun `flutter analyze` and `flutter test`. |
| IR-002 | Suggestion | Code Quality | Detail wire serialization is correct through `Project.toMap()` / `Project.toCreateMap()`, but direct concrete detail mapper calls such as `KnittingProjectDetailsMapper.toMap()` or `UnknownProjectDetailsMapper.toMap()` do not produce the AppView details wire shape with flattened raw fields and `$type`. This is acceptable for the current create path, but it is easy for future UI code to misuse. | `app/lib/projects/models/project.dart` lines 15-18, 165-208; `01-requirements.md` `FR-002`, `FR-003`, `FR-005`; `04-coding-plan.md` `CPQ-001` | Non-blocking: document that project details should be serialized through `Project` / `ProjectDetailsMapper`, or add a small public helper/extension if future composer code needs details-only serialization. |

## Requirement And Test Traceability

- Requirements implemented: `FR-001` through `FR-008`, `FR-010` through `FR-012`, `RULE-001` through `RULE-003`, and `NFR-001` through `NFR-004` are implemented and covered by source/test changes reviewed.
- Tests implemented: model parsing/serialization, known and unknown details variants, `Post.project`, create payloads and project-plus-reply guards, profile projects API/repository/provider pagination, user project state, create/delete/cache helpers, and regression coverage for general posts.
- Unplanned behavior: None identified. The touched UI test file only adapts fake repository usage for the changed create signature.
- Remaining gaps: `FR-009` / `AC-013` mutation-cache test coverage needs the additional project branch/rollback/key assertions described in `IR-001`.

## Test Evidence

- Commands reviewed:
  - Implementation handoff reports `cd app && dart run build_runner build --delete-conflicting-outputs`, `cd app && flutter analyze`, and `cd app && flutter test` passed in `05-implementation-plan.md`.
  - Reviewer reran `flutter analyze` from `app/` — passed.
  - Reviewer reran `flutter test` from `app/` — passed: `+527: All tests passed!`.
- Passing evidence: Static analysis and full Flutter tests pass after the implementation commit `b2f56b0`.
- Failing or skipped tests: A first reviewer attempt ran Flutter commands concurrently and hit a local Flutter startup/ephemeral-file conflict; rerunning `flutter test` after analysis completed passed. No persistent test failure remains.

## Risk Review

- Risk level: Medium.
- Risk notes: The highest-risk areas are project-details discriminator mapping, create request guards, provider pagination state, and cross-provider cache fan-out. Model/API/provider behavior is well covered; project interaction cache rollback coverage is the main residual risk.
- Approval notes: No architecture, dependency, lexicon, AppView, migration, route, or UI-scope deviation was found.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This slice intentionally does not implement user-facing project UI.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes:
  - Add the missing project-cache mutation tests from `IR-001`.
- Suggested next failing test:
  - Add a failing test in `app/test/feed/providers/toggle_post_interactions_provider_test.dart` proving a project post like/repost failure rolls back `userProjectsProvider` for both `alice.craftsky.social` and `did:plc:alice` without changing `userPostsProvider`.
- Verification to rerun:
  - `cd app && flutter test test/feed/providers/toggle_post_interactions_provider_test.dart`
  - `cd app && flutter test test/feed/providers/create_post_provider_test.dart test/feed/providers/delete_post_provider_test.dart`
  - `cd app && flutter analyze`
  - `cd app && flutter test`
