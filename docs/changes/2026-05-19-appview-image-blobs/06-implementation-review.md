# Implementation Review: AppView Image Blob Handling

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-05-19
Risk level: High

## Summary
The implementation covers most of the approved image-blob slice: the PDS upload adapter, authenticated `/v1/blobs/images` route, upload size/MIME validation, top-level post `images` write path, additive image `aspectRatio` lexicon/codegen work, index/read response metadata, and text-only/no-video regressions are present with passing focused tests.

One Must acceptance criterion is still under-implemented: create-post image validation only rejects a completely missing/empty `image` object, but it does not validate the required blob metadata inside that object. A request such as `{"image":{"foo":"bar"},"alt":"ok"}` can pass AppView validation and be forwarded to the PDS instead of failing with a standard validation error. Because `AC-009` explicitly requires missing blob metadata to fail at AppView validation, this is blocking.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | `POST /v1/posts` image validation checks only that `images[i].image` is non-empty; it does not validate required atproto blob metadata such as `ref.$link`, `mimeType`, and `size`. The implemented negative test covers only an entirely missing `image` object, so partial/malformed blob metadata can reach `CreateRecord` instead of returning the required standard validation error. | `02-requirements.md` FR-006, FR-008, RULE-002; `AC-009`; `03-acceptance-tests.md` UT-003 / TD-005; `appview/internal/api/post_request.go` `ValidatePostCreate`; `appview/internal/api/post_request_test.go` `TestValidatePostCreate_RejectsMissingAltOrBlobOrInvalidAspectRatio` | Add request validation for required blob metadata inside each image object and extend tests to cover partial/malformed blob objects (for example missing `ref`, missing `ref.$link`, missing/empty `mimeType`, and missing/non-positive `size`) before PDS write. |

## Requirement And Test Traceability
- Requirements implemented:
  - `BR-001`, `FR-001` through `FR-005`, `FR-009`, `NFR-001`, `NFR-002`, and `RULE-004`: upload endpoint, PDS adapter, MIME/size validation, route auth/device wiring, and normalized upload response are implemented.
  - `BR-003`, `FR-006`, `FR-007`, `FR-008`, `RULE-001`, and `RULE-002`: create-post accepts top-level images, enforces count/alt/aspect-ratio validation, and writes top-level `images` records.
  - `FR-010`, `FR-011`: indexed images are mapped to response image views with metadata and safe URL synthesis.
  - `FR-012`, `BR-002`: text-only and no-video regression coverage exists.
- Tests implemented:
  - Evidence found for `IT-001`, `UT-001`, `UT-002`, `IT-002`, `IT-006`, `UT-003`, `UT-004`, `IT-004`, `IT-005`, `UT-005`, `UT-006`, `IT-003`, `AT-003`, `REG-001` through `REG-004`, and `UT-007`.
- Unplanned behavior:
  - None identified in the implementation commit beyond the blocking validation gap above.
  - Current working tree still contains an unstaged `01-discovery-notes.md` change not included in the implementation commit; it was not reviewed as part of the committed implementation.
- Remaining gaps:
  - Blocking: malformed/partial blob metadata in create-post image payloads is not rejected by AppView validation (`IR-001`).
  - Manual/public-media checks (`MAN-002`, optional `MAN-003`) remain non-automated as expected by the test design.

## Test Evidence
- Commands reviewed from implementation notes:
  - `cd appview && go test ./internal/auth ./internal/api ./internal/index ./internal/routes`
  - `just lexgen`
  - focused per-loop `go test` commands listed in `05-implementation-plan.md`
- Commands rerun during review:
  - `cd appview && go test ./internal/auth ./internal/api ./internal/index ./internal/routes` — passed.
  - `just lexgen-check` — passed; generated lexicon/codegen files are current.
- Failing or skipped tests:
  - No focused review command failed.
  - `05-implementation-plan.md` notes DB-backed tests may skip without a configured test DB. The broader `go test ./internal/...` was not treated as required evidence because repository docs indicate full DB-backed verification requires the compose Postgres workflow.

## Risk Review
- Risk level: High.
- Risk notes:
  - Authenticated file upload is bounded to 15 MB + 1 byte and invalid MIME/oversize paths are tested not to call PDS upload.
  - PDS tokens remain server-side; upload and route tests do not expose PDS tokens in responses.
  - Logging-safety tests cover sentinel upload bytes, request text, and bearer token values.
  - Lexicon governance was followed with `adr/003-post-image-aspect-ratio.md`, schema update, `just lexgen`, and generated file updates.
- Approval notes:
  - Most high-risk areas are covered, but the missing create-post blob-metadata validation leaves a Must acceptance criterion incomplete.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This implementation is backend/API/lexicon only and does not include user-facing UI changes.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes:
  - Address `IR-001` by validating each create-post image blob object for required metadata before PDS write.
  - Add focused failing tests under `UT-003` / `IT-004` for partially malformed image blob metadata.
- Suggested next failing test:
  - `cd appview && go test ./internal/api -run 'TestValidatePostCreate_RejectsImageWithMissingBlobMetadata|TestCreatePost_WithInvalidImageBlobMetadata_422WithoutPDSWrite'`
- Verification to rerun:
  - `cd appview && go test ./internal/api -run 'TestValidatePostCreate|TestCreatePost_.*Image'`
  - `cd appview && go test ./internal/auth ./internal/api ./internal/index ./internal/routes`
  - `just lexgen-check`
