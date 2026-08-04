# Document Review: Local Post Drafts And Submit-Time Media Uploads

## Verdict

Status: Approved
Reviewer: Codex
Date: 2026-08-03
Risk level: Medium

## Summary

The revised requirements and acceptance-test specification are consistent, traceable, and ready for coding planning. They preserve the approved product direction: explicit device-local drafts for top-level standard and project posts, durable prepared media, no optimistic upload, unchanged immediate/scheduled API contracts, and one blocking screen-awake submission lifecycle.

The previous blocking finding is resolved. The first release does not attempt to observe terminal account deletion occurring elsewhere; ordinary sign-out and unobservable remote deletion do not purge local files. Drafts remain until explicit local deletion, confirmed successful publication/scheduling, or app-data removal. This keeps the lifecycle implementable without inventing a new API or conflating sign-out with deletion.

Device-local draft storage is explicitly approved as a deliberate product exception to the repository's broader AppView/Postgres guidance. The test design enforces that boundary, and the scheduled API-client contract test now has a concrete new target.

## Findings

None identified.

## Traceability Review

- Planning to requirements: Pass. The embedded Initial Request, confirmed grilling decisions, recommended file-backed repository, shared submission coordinator, exact UI copy, local-only retention boundary, and non-goals are carried through the requirements.
- Requirements to acceptance criteria: Pass. All 38 Must requirements reference at least one acceptance criterion. The single Should requirement, `NFR-004`, references `AC-020`. No requirement or criterion ID is missing or unknown.
- Acceptance criteria to tests: Pass. `AC-001` through `AC-027` each map to a matching acceptance scenario and at least one automated unit or integration target. Test cases preserve requirement and acceptance-criterion links.
- Prior finding resolution: `DR-001` is resolved by `FR-020`, `RULE-006`, `AC-023`, `AT-023`, and `IT-005`; `DR-002` is explicitly accepted and documented; `DR-003` is resolved by the proposed `app/test/scheduled_posts/scheduled_post_api_client_test.dart` target.

## Coverage Review

- Must requirements covered: 38 of 38 have designed automated coverage.
- Should requirements covered: 1 of 1 has designed automated coverage.
- Acceptance criteria covered: 27 of 27 have designed automated coverage.
- Missing or weak coverage: None identified.
- Manual-only coverage: None. Physical-device checks supplement automated tests for real filesystem/process termination, operating-system screen-awake behavior, accessibility/rendering, performance perception, and installation lifecycle.
- Test levels: Appropriate. Pure policies and state transitions are unit-level; filesystem, provider, API-client, account, and coordinator boundaries are integration-level; user-visible workflows use Gherkin/widget targets; existing composer and scheduled behavior has explicit regression coverage.
- Test data and targets: Sufficient and practical. Corruption/path attacks, atomic failure barriers, account leases, timeout/retry outcomes, privacy canaries, maximum-media async behavior, and a concrete scheduled API-client contract suite are included.

## Risk And Approval Review

- Risk level: Medium. Atomic private-file persistence and a shared submission lifecycle touch several existing composer seams, but server/API scope is unchanged and deterministic test boundaries are defined.
- Review requirement: Satisfied. Product decisions from grilling and the document-review corrections are incorporated.
- Approval notes: Device-local storage is deliberate despite the repository's broad AppView/Postgres draft guidance. Remote terminal-deletion propagation is explicitly out of scope. No new AppView storage, route, migration, job, or lexicon change is permitted by this plan.
- Remaining release gates: The four manual checks remain necessary for physical-device filesystem/lifecycle, screen-awake behavior, accessibility/rendering, and local retention evidence.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Design the first red test around `composer_images_provider_test.dart`: selecting a valid image must reach a locally-ready state while a recording blob client receives zero calls.
- Blocking issues: None.
- Planning guardrail: Preserve the existing immediate and scheduled wire contracts while moving invocation behind the explicit submission coordinator.

## Notes For Next Stage

- Keep drafts behind a repository interface backed by versioned manifests and immutable prepared-media files in persistent private application storage.
- Add a direct runtime persistent-directory dependency; do not promote SQLite or use SharedPreferences for draft content.
- Make the shared submission coordinator own validation-to-overlay ordering, exact pre-network snapshotting for existing drafts, target-specific media materialization, per-image timeout, transient immediate-reference reuse, account fencing, overlay cleanup, and screen-awake cleanup.
- Retain current standard/project hydrator and scheduled-post repository patterns where they already express complete payloads and owner-scoped state.
- Add the concrete scheduled API-client contract test alongside the current post API-client contract suite.
- Treat the no-pre-submit-network spy, atomic repository fault injection, privacy canaries, and screen-awake cleanup tests as implementation guardrails rather than late regression work.
- Carry all four manual checks forward as release gates, not substitutes for automated tests.
