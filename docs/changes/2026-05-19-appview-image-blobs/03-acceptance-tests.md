# Acceptance Test Specification: AppView Image Blob Handling

## 1. Test Strategy
Use test-first coverage around the AppView backend API contract, validation rules, PDS adapter boundary, post record construction, indexed image read mapping, and text-only regressions. Most checks should be automated in Go using existing `appview/internal/api`, `appview/internal/auth`, `appview/internal/index`, `appview/internal/routes`, and `appview/internal/api/*_test.go` patterns. Manual checks are limited to public-media wording and optional real-PDS smoke testing that local unit/integration tests cannot fully prove.

User-resolved test-design contract:
- Upload response: normalized wrapper containing the raw atproto blob plus `cid`, `mime`, and `size`.
- Post response images: `cid`, `mime`, `size`, `alt`, `aspectRatio`, `thumb`, and `fullsize` when available.
- Unknown/unsupported MIME during read URL synthesis: omit `thumb` and `fullsize` rather than forcing `@jpeg`.

Risk-based review recommendation: **High risk; review required before implementation** because this adds authenticated upload handling, PDS media writes, public-media behavior, and API contract expansion.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, IT-001, IT-002, IT-006 | Acceptance / Integration | Yes |
| BR-002 | AC-003 | REG-003 | Regression | Yes |
| BR-003 | AC-004 | AT-002, UT-004, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-001, AC-005 | AT-001, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-002 | AC-001 | AT-001, IT-002, IT-006 | Acceptance / Integration | Yes |
| FR-003 | AC-006 | UT-001, IT-002 | Unit / Integration | Yes |
| FR-004 | AC-007 | UT-002, IT-002 | Unit / Integration | Yes |
| FR-005 | AC-005 | AT-001, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-006 | AC-008, AC-009 | AT-002, UT-003, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-004, AC-008 | AT-002, UT-004, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-009, AC-010 | AT-002, UT-003, UT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-011 | AT-001, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-010 | AC-012, AC-013 | AT-003, UT-005, UT-006, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-012, AC-013 | AT-003, UT-006, IT-003, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-014 | REG-001, REG-002 | Regression | Yes |
| NFR-001 | AC-007 | UT-002, IT-002 | Unit / Integration | Yes |
| NFR-002 | AC-006, AC-007, AC-009 | UT-001, UT-002, UT-003, IT-002, IT-004 | Unit / Integration | Yes |
| NFR-003 | AC-015 | UT-007, MAN-001 | Unit / Manual | Partial |
| RULE-001 | AC-008 | UT-003, AT-002 | Unit / Acceptance | Yes |
| RULE-002 | AC-009 | UT-003, AT-002 | Unit / Acceptance | Yes |
| RULE-003 | AC-016 | MAN-002 | Manual | No |
| RULE-004 | AC-001 | AT-001, IT-006 | Acceptance / Integration | Yes |

## 3. Acceptance Scenarios
### AT-001: Authenticated Image Upload Returns Reusable Blob Metadata
Requirement IDs: BR-001, FR-001, FR-002, FR-005, FR-009, RULE-004
Acceptance Criteria: AC-001, AC-002, AC-005, AC-011
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/blob_test.go` or `appview/internal/api/post_test.go`

```gherkin
Feature: AppView image upload
  Scenario: Authenticated user uploads a supported image
    Given an authenticated Craftsky session for DID "did:plc:alice"
    And the request includes a valid X-Craftsky-Device-Id header
    And the PDS upload adapter returns a blob with cid "bafkimage", MIME "image/jpeg", and size 253496
    When the client POSTs JPEG bytes to "/v1/blobs/images"
    Then the AppView forwards the bytes to the caller's PDS without transforming them
    And the response includes the raw atproto blob object
    And the response includes cid "bafkimage", mime "image/jpeg", and size 253496
    And the response does not include any PDS token
```

### AT-002: Create Image Post Uses Top-Level Craftsky Images
Requirement IDs: BR-003, FR-006, FR-007, FR-008, RULE-001, RULE-002
Acceptance Criteria: AC-004, AC-008, AC-009, AC-010
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_test.go`

```gherkin
Feature: Create post with images
  Scenario: Authenticated user creates a post with image metadata
    Given an authenticated Craftsky user
    And the client has uploaded an image blob with cid "bafkimage", MIME "image/jpeg", and size 253496
    When the client POSTs "/v1/posts" with text, one top-level image, alt text, and aspectRatio width 919 height 2000
    Then the AppView writes a social.craftsky.feed.post record to the user's PDS
    And the record contains an "images" array at the top level
    And the record does not encode the image as a Bluesky-style embed.images media embed
    And the image object includes the blob, alt text, and aspectRatio values
```

### AT-003: Read Image Posts With Render-Ready Metadata
Requirement IDs: FR-010, FR-011
Acceptance Criteria: AC-012, AC-013
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_response_test.go`, `appview/internal/api/post_store_test.go`, `appview/internal/api/post_test.go`

```gherkin
Feature: Read image posts
  Scenario: Client reads an indexed image post
    Given an indexed post by DID "did:plc:alice" with one image cid "bafkimage", MIME "image/jpeg", size 253496, alt "project photo", and aspectRatio 919 by 2000
    When the client fetches the post through a single-post or list-post endpoint
    Then the post response includes one image object
    And the image includes cid, mime, size, alt, and aspectRatio
    And the image includes thumb and fullsize URLs for the author DID and image CID
```

### AT-004: Invalid Image Inputs Fail With Standard Error Envelopes
Requirement IDs: FR-003, FR-004, FR-006, FR-008, NFR-001, NFR-002, RULE-001, RULE-002
Acceptance Criteria: AC-006, AC-007, AC-008, AC-009
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/blob_test.go`, `appview/internal/api/post_request_test.go`, `appview/internal/api/post_test.go`

```gherkin
Feature: Image validation
  Scenario Outline: Invalid image upload or post input is rejected
    Given an authenticated Craftsky user
    When the client submits <invalid input>
    Then the AppView rejects the request
    And the response uses the standard error envelope

    Examples:
      | invalid input |
      | an upload with unsupported MIME type image/gif |
      | an upload larger than 15 MB |
      | a create-post request with 5 images |
      | a create-post image missing alt text |
      | a create-post image with aspectRatio width 0 |
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, NFR-002 | AC-006 | Validate accepted and rejected image MIME types. | `image/jpeg`, `image/png`, `image/webp`, `image/gif`, empty content type | Supported types pass; unsupported/empty types return validation errors in envelope-compatible form. | `appview/internal/api/blob_request_test.go` |
| UT-002 | FR-004, NFR-001, NFR-002 | AC-007 | Validate upload size limit and request bounding. | 15 MB body, 15 MB + 1 byte body | Body at limit may pass; body over limit fails before successful PDS upload. | `appview/internal/api/blob_request_test.go` / `blob_test.go` |
| UT-003 | FR-006, FR-008, NFR-002, RULE-001, RULE-002 | AC-008, AC-009 | Validate create-post image request rules. | 0-4 valid images, 5 images, missing blob, missing alt, blank alt, aspectRatio width/height <= 0 | Valid image arrays pass; invalid image arrays fail with field-specific validation errors. | `appview/internal/api/post_request_test.go` |
| UT-004 | BR-003, FR-007, FR-008 | AC-004, AC-008, AC-010 | Build lexicon record body with top-level images. | Valid create-post request with image blob and aspectRatio | Generated record body has `images` at top level, no `embed.images`, and preserves blob/alt/aspectRatio. | `appview/internal/api/post_test.go` |
| UT-005 | FR-010 | AC-012, AC-013 | Synthesize image URLs safely. | Known MIME `image/jpeg`; unknown MIME `image/tiff` | Known MIME yields `thumb`/`fullsize`; unknown MIME omits URL fields. | `appview/internal/api/post_response_test.go` |
| UT-006 | FR-010, FR-011 | AC-012, AC-013 | Build post response image objects from row image metadata. | Row images JSON with cid/mime/size/alt/aspectRatio | Response includes cid, mime, size, alt, aspectRatio, thumb, fullsize where available. | `appview/internal/api/post_response_test.go` |
| UT-007 | NFR-003 | AC-015 | Verify upload/create failure logging does not include image bytes or credentials. | Logger buffer, failed upload/create inputs containing sentinel byte string/token | Logs include failure context but not sentinel image bytes or token values. | `appview/internal/api/blob_test.go`, `post_test.go` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, FR-005, FR-009 | AC-001, AC-002, AC-005, AC-011 | Indigo PDS adapter uploads blobs through `com.atproto.repo.uploadBlob`. | `httptest.Server` emulating XRPC upload response | Call `PDSClient.UploadBlob` with content type and body | Server receives upload endpoint request with correct content type/body; adapter returns parsed blob; no token exposed to caller. | `appview/internal/auth/pds_client_indigo_test.go` |
| IT-002 | FR-001, FR-002, FR-003, FR-004, NFR-001, NFR-002 | AC-001, AC-006, AC-007 | Upload handler auth, validation, and PDS forwarding. | Fake authenticated request and fake PDS client | POST `/v1/blobs/images` with valid/invalid inputs | Valid request calls fake PDS once and returns metadata; invalid/auth failures return expected status/error and do not call PDS. | `appview/internal/api/blob_test.go` |
| IT-003 | FR-010, FR-011 | AC-012, AC-013 | Store reads image JSONB and response builder surfaces it. | Test Postgres schema with `craftsky_posts.images` JSONB containing cid/mime/size/alt/aspectRatio | `PostStore.ReadOne` and list methods fetch rows | Rows preserve image metadata; responses include expected image view objects. | `appview/internal/api/post_store_test.go`, `post_response_test.go` |
| IT-004 | BR-003, FR-006, FR-007, FR-008, NFR-002 | AC-004, AC-008, AC-009, AC-010 | Create-post handler validates images and writes PDS record. | Fake authenticated request, fake PDS client, fake author store | POST `/v1/posts` with image payload | PDS `CreateRecord` receives top-level images with blob/alt/aspectRatio; invalid requests return validation errors. | `appview/internal/api/post_test.go` |
| IT-005 | FR-008, FR-011 | AC-010, AC-012, AC-013 | Indexer flattens image size and aspect ratio from feed records. | Test Postgres schema, member profile, tap event with image blob and aspectRatio | Run `CraftskyPost.Handle` | `craftsky_posts.images` stores cid/mime/size/alt/aspectRatio for read side. | `appview/internal/index/craftsky_post_test.go` |
| IT-006 | BR-001, FR-002, RULE-004 | AC-001 | Routes wire upload endpoint behind auth and device middleware. | Route setup with fake deps/middleware patterns | Register routes and exercise `POST /v1/blobs/images` | Route exists at exact path and unauthenticated/missing-device requests are rejected consistently with `/v1/*`. | `appview/internal/routes/routes_test.go` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Text-only `POST /v1/posts` create flow remains valid. | FR-012 | Existing create-post happy path still returns created post and PDS record without requiring `images`. |
| REG-002 | Text-only post read/list responses remain compatible. | FR-012 | Posts without images return absent/empty image field without changing existing text, facets, reply, quote, engagement, author, and timestamp behavior. |
| REG-003 | Video remains out of scope. | BR-002 | No video upload route is registered and create-post validation does not accept video media fields. |
| REG-004 | Existing image indexing of cid/mime/alt remains intact. | FR-010, FR-011 | Existing indexer image test continues to pass while adding size/aspectRatio assertions. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Valid upload payload | Byte body `fake-jpeg-bytes`, `Content-Type: image/jpeg`, size below 15 MB | AT-001, IT-001, IT-002 |
| TD-002 | Oversized upload payload | Body length `15728641` bytes with supported MIME | AT-004, UT-002, IT-002 |
| TD-003 | Uploaded blob object | `{ "$type":"blob", "ref":{"$link":"bafkimage"}, "mimeType":"image/jpeg", "size":253496 }` | AT-001, AT-002, IT-001, IT-004 |
| TD-004 | Valid create-post image | `{ "image": TD-003.blob, "alt":"project photo", "aspectRatio":{"width":919,"height":2000} }` | AT-002, UT-003, UT-004, IT-004 |
| TD-005 | Invalid create-post image set | Five valid images; one missing alt; one missing blob; one with width `0`; one with height `-1` | AT-004, UT-003, IT-004 |
| TD-006 | Stored image JSONB | `[{"cid":"bafkimage","mime":"image/jpeg","size":253496,"alt":"project photo","aspectRatio":{"width":919,"height":2000}}]` | AT-003, UT-006, IT-003 |
| TD-007 | Unknown MIME stored image | `[{"cid":"bafkunknown","mime":"image/tiff","size":100,"alt":"scan"}]` | UT-005, EC-004 |

## 8. Manual Checks
| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-003 | Operational log sanity check | Trigger an upload validation failure and a PDS upload failure in a local/dev environment; inspect logs. | Logs identify route/failure category and request ID but do not include image bytes, bearer tokens, OAuth tokens, or raw blob payloads. |
| MAN-002 | RULE-003 | Public-media wording check | Review API docs, release notes, or user-facing copy produced with implementation. | Wording does not imply uploaded PDS media is private; any privacy caveat is accurate for current atproto behavior. |
| MAN-003 | FR-001, FR-010, FR-011 | Optional real-PDS smoke test | In dev with OAuth, upload a small JPEG, create a post using the returned blob, wait for indexing, and fetch the post. | Upload succeeds, post is indexed, and returned image URLs/metadata are usable. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Automated tests may not prove Bluesky CDN URLs are actually fetchable. | FR-010, FR-011 | Local tests can verify deterministic URL synthesis but not external CDN behavior. | Use MAN-003 or future media-proxy integration tests if this becomes critical. |
| GAP-002 | Aspect ratio correctness against actual image dimensions is not verified. | FR-008, FR-009 | Requirements explicitly exclude image byte inspection/processing. | If correctness becomes required, add image decode/metadata extraction requirements in a later pass. |
| GAP-003 | Public-media communication is mainly a documentation/product check. | RULE-003 | The backend can avoid privacy claims, but user understanding depends on copy outside code. | Keep MAN-002 in review checklist and revisit when frontend copy is written. |

## 10. Out Of Scope
- Video upload, validation, indexing, playback, and response tests.
- Flutter widget, state-management, and client upload tests.
- Craftsky-owned media proxy/CDN tests.
- Server-side image optimization, transcoding, EXIF stripping, or image-dimension extraction tests.
- Avatar/banner upload tests.
- Tests that require deleting uploaded blobs from a PDS.

## 11. Handoff To Document Review
- Requirements file: `02-requirements.md`
- Test specification: `03-acceptance-tests.md`
- Next review artifact: `04-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-19-appview-image-blobs/`
- Recommended first failing test for implementation: `IT-001` for `PDSClient.UploadBlob` in `appview/internal/auth/pds_client_indigo_test.go`, because the PDS adapter is the smallest required boundary and unlocks the upload handler.
- Suggested test order for implementation:
  1. `IT-001` PDS adapter upload.
  2. `UT-001` and `UT-002` upload validation.
  3. `IT-002` upload handler happy/error paths.
  4. `IT-006` route wiring for `POST /v1/blobs/images`.
  5. Lexicon aspect ratio update + generated types, then `UT-003` create-post image validation.
  6. `UT-004` / `IT-004` top-level image record creation.
  7. `IT-005` indexer image size/aspectRatio flattening.
  8. `UT-005`, `UT-006`, `IT-003`, and `AT-003` response image views.
  9. `REG-001` through `REG-004` text-only/no-video regressions.
  10. `UT-007` and manual checks for logging/public-media wording.
- Commands discovered:
  - `just test` — full Go test suite with race detector; requires compose Postgres via `just dev-d`.
  - `just fmt` — `gofmt` plus `go vet`.
  - `just lexgen` — regenerate lexicon Go types after the image `aspectRatio` lexicon change.
  - Focused command examples: `cd appview && go test ./internal/auth ./internal/api ./internal/index ./internal/routes` and `cd appview && go test ./internal/api -run 'Test.*Image|Test.*Blob|TestCreatePost'`.
- Blocking gaps: None after user confirmed the response-shape and unknown-MIME URL behavior.
