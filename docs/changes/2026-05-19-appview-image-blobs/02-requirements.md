# Business Requirements: AppView Image Blob Handling

## 1. Summary
Add AppView backend support for image blobs so authenticated Craftsky users can upload image files to their own PDS, create image posts using Craftsky's existing top-level `images` field, and receive render-ready image metadata on post read responses. This pass is limited to images and excludes video.

## 2. Problem / Opportunity
Craftsky's post lexicon and indexer already support image references, but the AppView cannot currently upload blobs, cannot create posts with images, and does not expose indexed image metadata in post responses. This blocks end-to-end image posting from Craftsky clients.

## 3. Goals
- G-001: Enable authenticated backend-mediated image upload to the caller's PDS.
- G-002: Enable AppView post creation with up to 4 images using Craftsky's existing top-level `images` record shape.
- G-003: Return render-ready image data on post reads so frontend work can proceed without inventing a second media contract.
- G-004: Preserve the current Craftsky top-level image model while adopting Bluesky-like upload flow and image metadata where useful.

## 4. Non-Goals
- NG-001: Video upload, storage, playback, or response rendering.
- NG-002: Switching Craftsky post media from top-level `images` to Bluesky-style `embed.images`.
- NG-003: Server-side image resizing, transcoding, optimization, EXIF stripping, or image content inspection.
- NG-004: Craftsky-owned image proxy/CDN or blob-serving infrastructure.
- NG-005: Flutter/frontend implementation.
- NG-006: Avatar or banner upload support.

## 5. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Authenticated Craftsky user | A signed-in user creating a post with images | Upload images and publish an image post through AppView without holding PDS tokens |
| Craftsky AppView | Backend mediating PDS writes and serving read APIs | Validate upload/create requests, proxy blob upload, and return stable response shapes |
| Craftsky frontend client | Flutter app or future client consuming `/v1/*` | Receive image metadata and render-ready URLs for image posts |
| User's PDS | atproto Personal Data Server holding blobs and records | Receive authenticated blob and record writes from AppView |

## 6. Current Behavior
The AppView can index post images when they already exist in `social.craftsky.feed.post` records written elsewhere, flattening them into `craftsky_posts.images`. However, `POST /v1/posts` rejects `images`, AppView has no blob upload endpoint, `PDSClient` has no blob upload method, and `PostResponse` omits image data entirely.

## 7. Desired Behavior
An authenticated client uploads an image to `POST /v1/blobs/images`, receives blob metadata suitable for later record creation, then calls `POST /v1/posts` with top-level `images` containing blob refs, alt text, and optional aspect ratio. The AppView writes the post to the user's PDS, and post read/list responses include image objects with render-ready URLs and available metadata such as CID, MIME type, size, alt text, and aspect ratio.

## 8. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must support backend-mediated image posting without exposing PDS tokens to clients. | Preserves the existing AppView/PDS trust model while enabling image posts. | Discovery, architecture rules | AC-001, AC-002 |
| BR-002 | Business | Must | This change must remain limited to images and must not expand scope to video. | Keeps the slice small and aligned with the confirmed request. | Prompt, discovery | AC-003 |
| BR-003 | Business | Must | Craftsky must preserve its existing top-level post `images` model for this pass rather than switching to Bluesky's `embed.images` media model. | Avoids reopening the post media architecture while still enabling image support. | User answer, discovery | AC-004 |
| FR-001 | Functional | Must | The system shall provide an authenticated upload endpoint at `POST /v1/blobs/images` that accepts a single image file and uploads it to the caller's PDS using `com.atproto.repo.uploadBlob`. | This is the required first step in the confirmed upload-then-create flow. | User answer, discovery | AC-001, AC-005 |
| FR-002 | Functional | Must | The upload endpoint shall require normal `/v1/*` authentication and `X-Craftsky-Device-Id` handling. | Keeps the new endpoint aligned with existing AppView API security conventions. | API architecture spec, discovery | AC-001 |
| FR-003 | Functional | Must | The upload endpoint shall accept only supported image MIME types allowed by Craftsky's post image contract: `image/jpeg`, `image/png`, and `image/webp`. | Aligns AppView validation with the Craftsky lexicon and avoids unsupported blob uploads. | Discovery, codebase findings | AC-006 |
| FR-004 | Functional | Must | The upload endpoint shall reject image uploads larger than 15 MB. | Matches the confirmed Craftsky image size limit. | User answer, discovery | AC-007 |
| FR-005 | Functional | Must | A successful upload response shall return blob metadata sufficient for a subsequent post-create request, including the atproto blob object and accessible metadata for CID, MIME type, and size. | The create flow depends on reusing the uploaded blob reference without additional lookup. | Discovery open question resolution | AC-005 |
| FR-006 | Functional | Must | `POST /v1/posts` shall accept up to 4 top-level images, each containing a blob reference, required alt text, and optional aspect ratio. | Enables image post creation while preserving the existing Craftsky media model. | Discovery, lexicon direction | AC-008, AC-009 |
| FR-007 | Functional | Must | When creating a post with images, the AppView shall write the images into the top-level `images` field of the `social.craftsky.feed.post` record written to the user's PDS. | Ensures the stored record matches the chosen Craftsky record shape. | User answer, discovery | AC-004, AC-008 |
| FR-008 | Functional | Must | The system shall support an additive lexicon update so each post image may include optional `aspectRatio.width` and `aspectRatio.height` values, both positive integers when present. | The reviewed discovery direction requires Bluesky-like aspect ratio metadata. | Plannotator feedback, user confirmation | AC-009, AC-010 |
| FR-009 | Functional | Must | The AppView shall not inspect or transform image bytes beyond request validation and forwarding to the PDS. | The user explicitly chose validate/pass-through behavior. | User answer, discovery | AC-011 |
| FR-010 | Functional | Must | Post read and list responses that include images shall return render-ready image metadata for each indexed image, including `thumb` and `fullsize` URLs plus available image metadata. | Frontend work depends on a stable response contract for rendering image posts. | User answer, discovery | AC-012, AC-013 |
| FR-011 | Functional | Must | Post read and list responses shall include image `alt` text and shall surface image `size` and `aspectRatio` when those values are available from the stored record/blob metadata. | Preserves accessibility data and the reviewed metadata addition. | Discovery, Plannotator feedback | AC-012, AC-013 |
| FR-012 | Functional | Must | Text-only post creation and read behavior shall remain backward compatible. | Prevents regression of existing post flows. | Discovery | AC-014 |
| NFR-001 | Non-functional | Must | The system shall bound request handling so oversized upload bodies are rejected rather than forwarded to the PDS. | Prevents resource pressure and matches the identified upload risk. | Discovery risk | AC-007 |
| NFR-002 | Non-functional | Must | The system shall preserve existing `/v1/*` error-envelope conventions for upload and create-post validation failures. | Keeps the new surface consistent with the existing API contract. | API architecture spec, discovery | AC-006, AC-007, AC-009 |
| NFR-003 | Non-functional | Should | Logs for upload and create-post failures should include enough context for operational diagnosis without logging sensitive credentials or image bytes. | Needed because the new endpoint adds higher operational risk. | Discovery risk | AC-015 |
| RULE-001 | Business rule | Must | Each post may contain at most 4 images. | Mirrors both Craftsky and Bluesky image-count expectations. | Discovery, existing lexicon | AC-008 |
| RULE-002 | Business rule | Must | Each post image must include non-empty alt text. | Accessibility is required by the current Craftsky image object shape. | Existing lexicon, discovery | AC-009 |
| RULE-003 | Business rule | Must | Image blobs uploaded through this flow must be treated as public media under current atproto/PDS behavior. | Prevents misleading privacy assumptions. | Discovery | AC-016 |
| RULE-004 | Business rule | Must | The upload route path for this change shall be `POST /v1/blobs/images`. | Locks the contract for downstream test design and implementation. | User answer | AC-001 |

## 9. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002, RULE-004 | Given an authenticated request with a valid device ID, when the client sends a supported image to `POST /v1/blobs/images`, then the AppView forwards the upload to the caller's PDS and returns a success response using the standard `/v1/*` conventions. |
| AC-002 | BR-001 | Given a Craftsky client using only its Craftsky session token, when it uploads an image or creates an image post, then no PDS token is required from or returned to the client. |
| AC-003 | BR-002 | Given this change is implemented, when the client attempts a video-oriented flow, then no video upload or video post behavior is introduced by this scope. |
| AC-004 | BR-003, FR-007 | Given a successful image post creation, when the written record is inspected, then images are stored in the top-level `images` field of `social.craftsky.feed.post` rather than a Bluesky-style media embed. |
| AC-005 | FR-001, FR-005 | Given a successful upload, when the response is returned, then it contains blob metadata sufficient for a later create-post request, including the returned blob object and accessible metadata for CID, MIME type, and size. |
| AC-006 | FR-003, NFR-002 | Given an upload request with an unsupported content type or malformed upload request, when the AppView validates it, then the request is rejected with the standard `/v1/*` error envelope. |
| AC-007 | FR-004, NFR-001, NFR-002 | Given an upload request larger than 15 MB, when the AppView receives it, then the request is rejected before a successful PDS upload occurs and the client receives a standard error response. |
| AC-008 | FR-006, RULE-001 | Given a create-post request containing 1 to 4 valid images, when the AppView validates and writes the post, then the request succeeds; and given a request containing more than 4 images, when validated, then it fails with a standard validation error. |
| AC-009 | FR-006, FR-008, RULE-002, NFR-002 | Given a create-post request with images, when any image is missing alt text, missing blob metadata, or has invalid non-positive aspect ratio values, then the request fails with a standard validation error; and when all required fields are valid, then the request may succeed. |
| AC-010 | FR-008 | Given an image post with optional aspect ratio metadata, when the record is written and later read back, then the aspect ratio values are preserved and surfaced when available. |
| AC-011 | FR-009 | Given a valid upload request, when the AppView processes it, then it validates and forwards the bytes to the PDS without resizing, transcoding, stripping metadata, or otherwise transforming the image content. |
| AC-012 | FR-010, FR-011 | Given an indexed post with images, when a client fetches the single-post read endpoint, then each image in the response includes render-ready `thumb` and `fullsize` URLs plus available `cid`, `mime`, `size`, `alt`, and `aspectRatio` metadata. |
| AC-013 | FR-010, FR-011 | Given an indexed post with images, when a client fetches a post list endpoint that returns that post, then the response includes the same image rendering contract as the single-post read endpoint. |
| AC-014 | FR-012 | Given a text-only post flow that succeeds today, when this change is released, then text-only post create/read behavior remains unchanged except for additive response fields that are absent or empty for posts without images. |
| AC-015 | NFR-003 | Given an upload or image-post failure, when the AppView logs the event, then operators can diagnose the failure cause without image bytes or sensitive credentials being written to logs. |
| AC-016 | RULE-003 | Given a user uploads or posts an image through this flow, when the feature is described or documented, then the system does not imply that uploaded media is private under current atproto/PDS behavior. |

## 10. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Upload succeeds but the client never creates a referencing post | The workflow remains valid; unreferenced blobs may expire per PDS behavior and this pass does not add special recovery handling. | FR-001, FR-005, ASM-002 |
| EC-002 | Create-post request mixes valid and invalid images | The request is rejected rather than partially accepted. | FR-006, FR-008, RULE-001, RULE-002 |
| EC-003 | Post record contains images but some optional metadata is absent | Read responses still return image URLs and required fields; optional metadata is omitted or null/empty per final response contract. | FR-010, FR-011 |
| EC-004 | Unknown or unsupported MIME appears in stored image metadata during read | URL synthesis behavior must follow the final documented rule and fail safely rather than producing invalid URLs. | FR-010, FR-011, ASM-003 |
| EC-005 | Client supplies aspect ratio values that are syntactically valid but do not match the real file dimensions | AppView accepts or rejects based only on declared validation rules for this pass and does not inspect the image content. | FR-008, FR-009 |

## 11. Data / Persistence Impact
- New fields:
  - Additive lexicon field: optional `aspectRatio` on each post image, with `width` and `height`.
  - Additive API response fields for image metadata, including URLs and available metadata.
- Changed fields:
  - `POST /v1/posts` request shape expands to allow image payloads.
  - Post response shapes expand to include images.
- Migration required:
  - Likely none for base persistence because `craftsky_posts.images JSONB` already exists, but implementation may need additive storage/read mapping updates.
- Backwards compatibility:
  - Text-only posts remain valid.
  - Existing image-less records remain valid.
  - Existing image records without aspect ratio remain valid because `aspectRatio` is optional.

## 12. UI / API / CLI Impact
- UI:
  - No UI implementation in scope, but the API must support future frontend image posting and rendering.
- API:
  - New authenticated route: `POST /v1/blobs/images`.
  - Expanded `POST /v1/posts` request contract.
  - Expanded post read/list response contracts.
- CLI:
  - None required by this stage.
- Background jobs:
  - None identified.

## 13. Security / Privacy / Permissions
- Authentication:
  - Upload and image-post flows require existing `/v1/*` authentication and device ID handling.
- Authorization:
  - Uploads and writes occur only against the authenticated user's PDS session.
- Sensitive data:
  - PDS OAuth tokens remain server-side only.
  - Image bytes are user content and must not be logged.
- Abuse cases:
  - Oversized upload attempts.
  - Unsupported MIME types.
  - Repeated upload attempts intended to pressure AppView or PDS resources.
  - Misunderstanding uploaded media as private when it is publicly accessible under current atproto behavior.

## 14. Observability
- Events:
  - None required.
- Logs:
  - Upload failure reason.
  - Create-post image validation failure reason.
  - PDS upload/write failure reason.
- Metrics:
  - None required by this stage, but upload failures and payload-size rejections are good future candidates.
- Alerts:
  - None required by this stage.

## 15. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Authenticated file-upload endpoint increases resource-pressure and abuse risk | Could degrade AppView reliability or create operational issues | Enforce auth, supported MIME validation, and 15 MB max-size rejection before successful upload |
| RISK-002 | Public-media behavior may be misunderstood by users or implementers | Could create incorrect privacy expectations | State public-media behavior explicitly in requirements, tests, and feature documentation |
| RISK-003 | Response URL synthesis may be coupled to current Bluesky CDN conventions | Could require future contract changes if Craftsky later adopts its own proxy/CDN | Keep image proxying out of scope and document the current response contract clearly |
| RISK-004 | Additive lexicon update for `aspectRatio` may be implemented inconsistently across write, index, and read paths | Could produce missing or mismatched metadata | Require end-to-end acceptance coverage for aspect ratio create/index/read behavior |

## 16. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The discovery note's Initial Request section is the source of truth in place of a missing `00-initial-prompt.md`. | If wrong, requirements could be misaligned with the intended change request. |
| ASM-002 | PDS behavior for unreferenced uploaded blobs is acceptable for this pass and does not require AppView cleanup logic. | If wrong, the workflow may need extra product or operational requirements for abandoned uploads. |
| ASM-003 | Reusing existing Bluesky-style image URL conventions is acceptable for this backend pass. | If wrong, the API contract would need revision toward Craftsky-owned media URLs or a different safe fallback. |

## 17. Review Status
Status: Draft
Risk level: High
Review recommended: Required
Reviewer:
Date:
Notes: Discovery marked this change as high risk due to authenticated file upload, public media handling, and API contract expansion.

## 18. Handoff To Test Design
- Requirements file: `02-requirements.md`
- Must-cover requirement IDs:
  - BR-001
  - BR-002
  - BR-003
  - FR-001 through FR-012
  - NFR-001 through NFR-002
  - RULE-001 through RULE-004
- Suggested test levels:
  - API handler tests
  - PDS adapter tests
  - Request/response contract tests
  - Integration tests for create/index/read image flow
  - Regression tests for text-only post behavior
- Blocking open questions:
  - Upload response shape finalization.
  - Read response image field finalization for optional metadata presence semantics.
  - Feed-image URL extension behavior for unsupported/unknown MIME metadata.
