# Document Review: AppView Image Blob Handling

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-05-19
Risk level: High

## Summary
The workflow documents are consistent enough to proceed to TDD implementation. Discovery, requirements, and test design all preserve the confirmed direction: AppView-only image support, two-step upload-then-create flow, existing top-level Craftsky `images` record shape, 15 MB/image limit, validate/pass-through handling, render-ready post response metadata, and optional image `aspectRatio`/blob `size` support.

No blocking traceability or coverage gaps were found. The main implementation note is that the accepted requirements include an additive lexicon change, so the implementation stage must follow the repository's lexicon-change rules: create/update an ADR as appropriate, run `just lexgen`, and commit generated lexicon-derived Go types with the schema change.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk / Implementation readiness | The documents require an additive `social.craftsky.feed.post#image.aspectRatio` lexicon change, but the requirements/test-design documents do not explicitly list the repository's mandatory ADR step for lexicon edits. This is not blocking because `01-discovery-notes.md` and `AGENTS.md` both identify lexicon governance, but the implementation agent must not skip it. | `01-discovery-notes.md` lines 32-39, 67-70; `02-requirements.md` FR-008 and section 11; `03-acceptance-tests.md` suggested order step 5; `AGENTS.md` lexicon rule | Before editing `lexicon/`, use the project lexicon workflow and add the required ADR or amend existing ADR coverage, then run `just lexgen`. |
| DR-002 | Suggestion | Traceability | Discovery still lists route/response/MIME URL behavior as open questions, while requirements and test design resolve them. The later-stage documents are clear and should be treated as the implementation source of truth, but the stale discovery checklist may be confusing. | `01-discovery-notes.md` Open Questions lines 165-170; `02-requirements.md` RULE-004, FR-005, FR-010/FR-011; `03-acceptance-tests.md` Test Strategy lines 6-9 | No required change before implementation. TDD builder should treat `02-requirements.md`, `03-acceptance-tests.md`, and this review as superseding those discovery open questions. |
| DR-003 | Suggestion | Tests / Manual coverage | Public-media communication is covered by a manual check rather than automation. This is acceptable because it concerns wording and product communication, but it should remain visible during implementation/review. | `02-requirements.md` RULE-003 and AC-016; `03-acceptance-tests.md` MAN-002 and GAP-003 | Keep MAN-002 in the implementation review checklist; do not claim privacy for PDS blobs in API docs or user-facing copy. |

## Traceability Review
- Discovery to requirements: Confirmed. `02-requirements.md` carries forward the discovery recommendation (Option A), top-level images, 15 MB limit, validate/pass-through behavior, URL-returning responses, image-only scope, and aspect-ratio follow-up from Plannotator feedback.
- Requirements to acceptance criteria: Confirmed. Every Must `BR`, `FR`, `NFR`, and `RULE` has at least one linked `AC`. The acceptance criteria are externally verifiable through API responses, PDS-adapter calls, record bodies, validation errors, response shapes, and manual copy review where appropriate.
- Acceptance criteria to tests: Confirmed. `03-acceptance-tests.md` maps every acceptance criterion to acceptance, unit, integration, regression, or manual coverage. High-risk paths have multiple concrete automated tests.

## Coverage Review
- Must requirements covered: Yes. BR-001 through BR-003, FR-001 through FR-012, NFR-001/NFR-002, and RULE-001 through RULE-004 all have linked test IDs in the coverage matrix.
- Missing or weak coverage: None blocking. `RULE-003` / `AC-016` is manual-only, but the test design documents why this is appropriate for public-media wording.
- Manual-only coverage: MAN-002 for public-media wording; MAN-003 for optional real-PDS smoke testing. Both are justified because local tests cannot prove user-facing copy or external CDN/PDS behavior fully.

## Risk And Approval Review
- Risk level: High.
- Review requirement: Required before implementation due to authenticated upload handling, public media, PDS writes, API contract expansion, and lexicon change.
- Approval notes: Requirements were approved in Plannotator. Test-design review via Plannotator was attempted but aborted; this document review finds no blockers. Implementation should proceed only in a fresh TDD session and should keep security/resource-limit tests near the front of the work.

## Implementation Readiness
- Ready for TDD implementation: Yes.
- Recommended first step: Start with `IT-001` for `PDSClient.UploadBlob` in `appview/internal/auth/pds_client_indigo_test.go`, as recommended by `03-acceptance-tests.md`. This isolates the PDS adapter boundary before building upload handler behavior.
- Blocking issues: None.

## Notes For Next Stage
- Start implementation in a fresh OpenCode session with the TDD builder.
- Treat `02-requirements.md`, `03-acceptance-tests.md`, and `04-document-review.md` as the source of truth. Use `01-discovery-notes.md` for background only where it does not conflict with later resolved requirements.
- Before editing `lexicon/social/craftsky/feed/post.json`, follow the repository lexicon-change process and capture the aspect-ratio decision in an ADR or an appropriate ADR amendment.
- Run `just lexgen` after the lexicon change and commit regenerated files with the schema change.
- Preserve existing text-only post behavior and existing indexed image `{cid,mime,alt}` support while adding `size` and `aspectRatio`.
- Keep upload validation and request-size bounding early in the implementation sequence to reduce high-risk file-upload exposure.
