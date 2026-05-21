# Acceptance Test Specification: Flutter Image Posts

## 1. Test Strategy
Use test-first coverage around the Flutter app's image-posting write path, local media preparation, AppView API contracts, composer state, feed/gallery rendering, gesture routing, accessibility, and text-only regressions. Most checks should be automated in Flutter tests using existing `app/test/feed/**`, `http_mock_adapter`, Riverpod provider tests, widget tests, and fake cache managers. Metadata stripping should have focused pure/unit coverage where the implementation allows byte-level inspection, plus manual verification with real device photos because platform media libraries can behave differently.

Risk-based review recommendation: **High risk; review required before implementation** because this adds local media access, privacy-sensitive metadata stripping, authenticated media upload, user-visible composer state, generated model/API changes, complex nested gestures, and accessibility-visible image behavior.

Recommended first failing test: **UT-001** for local app media configuration constants (`maxImages`, `maxImageBytes`, `maxAltTextCharacters`) because many later tests should consume those values rather than hard-coded literals.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, IT-001, IT-002, IT-005 | Acceptance / Integration | Yes |
| BR-002 | AC-003, AC-004 | AT-003, UT-008, IT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-005 | AT-005, REG-002 | Acceptance / Regression | Yes |
| BR-004 | AC-006 | REG-004 | Regression | Yes |
| BR-005 | AC-026 | AT-002, UT-006, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-001 | AC-001, AC-007 | AT-001, AT-002, UT-001, UT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-007 | AT-002, UT-001, UT-002, UT-003 | Acceptance / Unit | Yes |
| FR-003 | AC-001, AC-008 | AT-001, IT-001, IT-005 | Acceptance / Integration | Yes |
| FR-003A | AC-008, AC-010 | AT-002, UT-005, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-009 | UT-004, IT-004, MAN-002 | Unit / Integration / Manual | Partial |
| FR-004A | AC-009 | UT-004, IT-004, MAN-002 | Unit / Integration / Manual | Partial |
| FR-004B | AC-009 | UT-004, IT-004, MAN-002 | Unit / Integration / Manual | Partial |
| FR-004C | AC-009A | UT-004, IT-004, MAN-002 | Unit / Integration / Manual | Partial |
| FR-005 | AC-008 | AT-001, AT-002, IT-005 | Acceptance / Integration | Yes |
| FR-006 | AC-010 | AT-002, UT-005, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-011 | AT-002, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-012 | AT-002, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-008A | AC-012 | AT-001, AT-002, IT-005 | Acceptance / Integration | Yes |
| FR-008B | AC-012, AC-027 | AT-002, UT-001, UT-006 | Acceptance / Unit | Yes |
| FR-009 | AC-013 | AT-001, UT-010, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-002, AC-013 | AT-001, UT-009, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-014, AC-026 | AT-002, UT-006, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-001, AC-007 | AT-001, AT-002, UT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-011, AC-015 | AT-001, UT-007, IT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-013A | AC-015A | AT-001, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-003, AC-016 | AT-003, UT-008, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-017 | AT-003, UT-011, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-017 | AT-003, IT-006 | Acceptance / Integration | Yes |
| FR-016A | AC-017A | AT-003, UT-012, IT-006, MAN-004 | Acceptance / Unit / Integration / Manual | Partial |
| FR-017 | AC-018 | AT-003, AT-004, IT-007 | Acceptance / Integration | Yes |
| FR-018 | AC-004, AC-019 | AT-003, UT-013, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-018 | AT-003, IT-007, MAN-005 | Acceptance / Integration / Manual | Partial |
| FR-019A | AC-019A | AT-003, UT-013, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-020 | AT-004, IT-007, MAN-003 | Acceptance / Integration / Manual | Partial |
| FR-021 | AC-020, AC-021 | AT-004, IT-006, IT-007, MAN-003 | Acceptance / Integration / Manual | Partial |
| FR-022 | AC-005 | AT-005, REG-002 | Acceptance / Regression | Yes |
| FR-023 | AC-022 | REG-001, REG-002, REG-003 | Regression | Yes |
| NFR-001 | AC-001, AC-002, AC-010 | IT-001, IT-002, IT-005, REG-005 | Integration / Regression | Yes |
| NFR-002 | AC-012, AC-023 | AT-004, UT-013, IT-006, IT-007 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-024 | IT-006, IT-008 | Integration | Yes |
| NFR-004 | AC-008, AC-014 | AT-002, IT-005 | Acceptance / Integration | Yes |
| RULE-001 | AC-005 | AT-005, REG-002 | Acceptance / Regression | Yes |
| RULE-002 | AC-007 | UT-001, UT-002, AT-002 | Unit / Acceptance | Yes |
| RULE-003 | AC-012, AC-014 | AT-002, UT-006 | Acceptance / Unit | Yes |
| RULE-004 | AC-011 | AT-002, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-025 | MAN-001, REG-005 | Manual / Regression | Partial |
| RULE-006 | AC-027 | UT-001, UT-006, AT-002 | Unit / Acceptance | Yes |
| RULE-007 | AC-014, AC-026 | AT-002, UT-006, REG-001 | Acceptance / Unit / Regression | Yes |
| RULE-008 | AC-007, AC-009B | UT-003, AT-002 | Unit / Acceptance | Yes |

## 3. Acceptance Scenarios
### AT-001: Compose and publish a text post with images
Requirement IDs: BR-001, FR-001, FR-003, FR-005, FR-008A, FR-009, FR-010, FR-012, FR-013, FR-013A
Acceptance Criteria: AC-001, AC-002, AC-008, AC-013, AC-015, AC-015A
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart` plus provider/API fakes

```gherkin
Feature: Top-level image post composition
  Scenario: User prepares, uploads, reorders, and publishes image attachments
    Given a signed-in user opens the top-level post composer
    And the local media config allows 4 images of 15 MB each and 300-character alt text
    When the user selects two supported images
    Then each image is shown as a draft preview in preparing or upload state
    And each image is prepared and uploaded through AppView without exposing a PDS token
    When the user enters valid post text and valid alt text for both images
    And the user reorders the images before submitting
    Then the submit action becomes enabled after both remaining images are uploaded
    When the user submits the post
    Then the create-post payload contains top-level images in current composer order
    And each image entry includes uploaded blob metadata, alt text, and aspectRatio when available
```

### AT-002: Invalid image drafts block submission until resolved
Requirement IDs: BR-005, FR-001, FR-002, FR-003A, FR-006, FR-007, FR-008, FR-008B, FR-011, FR-012, RULE-002, RULE-003, RULE-004, RULE-006, RULE-007, RULE-008
Acceptance Criteria: AC-007, AC-010, AC-011, AC-012, AC-014, AC-026, AC-027
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Composer validation and recovery
  Scenario: User cannot submit an invalid image draft
    Given a signed-in user opens the top-level post composer
    When the user selects an unsupported image, a prepared image over the configured size limit, or more images than the configured cap
    Then the invalid or excess images are rejected with visible feedback and are not uploaded
    When a selected supported image fails metadata stripping or upload
    Then that image shows an error state
    And the submit action remains disabled
    When the user removes the failed image
    And the draft has valid post text and valid alt text for all remaining images
    Then the submit action becomes enabled
```

### AT-003: Feed carousel and full-screen gallery render image posts
Requirement IDs: BR-002, FR-014, FR-015, FR-016, FR-016A, FR-017, FR-018, FR-019, FR-019A
Acceptance Criteria: AC-003, AC-004, AC-016, AC-017, AC-017A, AC-018, AC-019, AC-019A
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart` and future gallery widget test file

```gherkin
Feature: Feed image display
  Scenario: User views image posts in feed and gallery
    Given the app has a post response with four images in a known order
    When the post is parsed and rendered in the feed
    Then the post card shows an inline bounded-aspect carousel
    And the carousel shows one large image at a time
    And the count and page-dot indicators are legible against image content
    When the user taps the image area
    Then a full-screen gallery opens with a hero-style transition where feasible
    And the gallery displays the tapped image's alt text at the bottom
    When the user swipes horizontally
    Then the gallery moves through images in post order
```

### AT-004: Image gestures remain distinct and accessible
Requirement IDs: FR-017, FR-020, FR-021, NFR-002
Acceptance Criteria: AC-018, AC-020, AC-021, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart` and future gallery widget test file

```gherkin
Feature: Media gestures and accessibility
  Scenario: User scrolls, pages, zooms, taps, and navigates without gesture conflicts
    Given the feed contains image posts with multiple images
    When the user vertically scrolls the feed
    Then feed scrolling continues to work
    When the user horizontally pages inside the carousel
    Then the image page changes without triggering post navigation
    When the user pinches on an inline image or gallery image
    Then the image zooms without breaking surrounding browsing behavior
    When assistive technology focuses image media or gallery entry controls
    Then the image alt text is exposed through semantics
```

### AT-005: Reply composer remains text-only
Requirement IDs: BR-003, FR-022, RULE-001
Acceptance Criteria: AC-005
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Reply composition scope
  Scenario: User replies to an image post
    Given a post with images exists
    When the user opens the reply composer for that post
    Then the reply composer shows the existing text-only reply UI
    And no image selection, upload, alt text, or reordering UI is shown
```

### AT-006: Public media messaging does not imply privacy
Requirement IDs: RULE-005
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance / Manual-supported
Automation Target: Copy/widget tests where text exists; manual review for final wording

```gherkin
Feature: Public media expectations
  Scenario: User sees image posting copy
    Given the image posting flow contains user-facing copy about uploaded images
    When the copy is reviewed
    Then it does not imply that PDS-hosted post images are private
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, FR-002, FR-008B, RULE-002, RULE-006 | AC-007, AC-027 | Media config exposes centralized limits. | Default app media config | `maxImages == 4`, `maxImageBytes == 15 MB`, `maxAltTextCharacters == 300`; callers use config object/constant. | Future `app/test/feed/media/media_config_test.dart` |
| UT-002 | FR-001, FR-002, FR-012, RULE-002 | AC-007 | Selected image validation accepts supported MIME/extensions and rejects unsupported or excess images. | JPEG/PNG/WebP, GIF, empty type, existing draft counts 0-4 | Supported images within cap accepted; unsupported/excess rejected before upload. | Future `app/test/feed/media/image_selection_validator_test.dart` |
| UT-003 | FR-002, RULE-008 | AC-007, AC-009B | Prepared upload byte size validation uses prepared bytes. | Original size above/below limit; prepared bytes at limit and over limit | Prepared bytes at limit pass; over limit fails and is not uploaded. | Future `app/test/feed/media/image_upload_preparer_test.dart` |
| UT-004 | FR-004, FR-004A, FR-004B, FR-004C | AC-009, AC-009A | Metadata stripping removes privacy-sensitive metadata while preserving display/format behavior. | Test images with GPS, make/model, timestamp, software/comment metadata; PNG transparency sample | Removable metadata absent in output; visible dimensions/content valid; supported format preserved where possible; PNG alpha preserved. | Future `app/test/feed/media/image_metadata_stripper_test.dart` |
| UT-005 | FR-003A, FR-006 | AC-008, AC-010 | Draft image lifecycle transitions are deterministic. | Validate success/failure, strip success/failure, upload progress/success/failure | States progress through `preparing` → `uploading` → `uploaded`, or to error with retry/remove. | Future `app/test/feed/providers/image_draft_controller_test.dart` |
| UT-006 | FR-008, FR-008B, FR-011, RULE-003, RULE-006, RULE-007 | AC-012, AC-014, AC-026, AC-027 | Submit gate enforces required text, valid image states, and alt text length. | Empty text, 2001-char text, valid text, missing alt, 301-char alt, preparing/uploading/failed/uploaded images | Submit enabled only for valid text plus zero or more valid uploaded images. | Future `app/test/feed/providers/image_post_submit_gate_test.dart` |
| UT-007 | FR-007, FR-013, FR-013A, RULE-004 | AC-011, AC-015, AC-015A | Draft order and deletion are independent of upload completion. | Three images, upload completion events out of order, reorder, delete pending image | Composer order controls payload; deleted image excluded even if upload completes later. | Future `app/test/feed/providers/image_draft_controller_test.dart` |
| UT-008 | FR-014 | AC-003, AC-016 | Post model parses and round-trips image metadata. | Post JSON with `images[]` including cid, mime, size, alt, aspectRatio, thumb, fullsize | `Post.images` contains metadata in response order; missing optional fields fail soft. | `app/test/feed/models/post_test.dart` |
| UT-009 | FR-010 | AC-002 | Create-post request image model serializes top-level images. | Uploaded blob metadata, alt text, aspectRatio | Payload contains `images[]` at top level with blob, alt, and optional aspectRatio; no embed media shape. | `app/test/feed/data/post_api_client_test.dart` or future model test |
| UT-010 | FR-009 | AC-013 | Aspect ratio extraction is optional and positive. | Image dimensions known, unknown, zero/invalid | Known dimensions produce positive width/height; unknown omits aspectRatio without blocking. | Future `app/test/feed/media/image_dimensions_test.dart` |
| UT-011 | FR-015 | AC-017 | Feed carousel layout computes bounded aspect frame. | Aspect ratios: missing, 1:1, very tall, very wide | Frame preserves usable aspect ratio within min/max height bounds; fallback stable when missing. | Future `app/test/feed/widgets/post_image_carousel_test.dart` |
| UT-012 | FR-016A | AC-017A | Carousel indicators use contrast treatment. | Light/dark/mixed fake image backgrounds or widget config | Indicator count/dots render with contrasting text/icon color or semi-transparent backing. | Future `app/test/feed/widgets/post_image_carousel_test.dart` |
| UT-013 | FR-018, FR-019A, NFR-002 | AC-019, AC-019A, AC-023 | Gallery state exposes current image, visible alt text, and semantics. | Gallery images with alt text, page changes | Current alt text displays at bottom and is exposed to semantics; page state follows swipes. | Future `app/test/feed/widgets/post_image_gallery_test.dart` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-003, NFR-001 | AC-001 | API client uploads prepared image bytes to AppView. | Dio with `http_mock_adapter`; prepared JPEG bytes; content type `image/jpeg`; progress callback if implemented | Call image upload API method | POSTs `/v1/blobs/images` with raw bytes/content type; parses `{blob,cid,mime,size}`; errors map through existing API conventions. | `app/test/feed/data/post_api_client_test.dart` or future `image_upload_api_client_test.dart` |
| IT-002 | BR-001, FR-009, FR-010, FR-013 | AC-002, AC-013, AC-015 | API client sends image create payload in composer order. | Two uploaded images, reordered draft, valid text, aspect ratio on one image | Call `createPost` with image attachments | POST `/v1/posts` body contains text and top-level `images[]` in composer order with blob, alt, aspectRatio where available. | `app/test/feed/data/post_api_client_test.dart` |
| IT-003 | FR-014 | AC-003, AC-016 | Read/list endpoints parse image posts. | Mock `getPost` and `listPostsByAuthor` responses with image arrays | Fetch post/list | Parsed posts expose image metadata to callers without breaking text-only posts. | `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/models/post_page_test.dart` |
| IT-004 | FR-004, FR-004A, FR-004B, FR-004C | AC-009, AC-009A | Image preparer strips metadata before uploader receives bytes. | Fake image picker output with inspectable EXIF/GPS; fake uploader records bytes | Run prepare/upload pipeline | Uploader receives prepared bytes with removable metadata absent and supported display behavior intact. | Future `app/test/feed/media/image_upload_pipeline_test.dart` |
| IT-005 | FR-001, FR-003A, FR-005, FR-006, FR-007, FR-008A, FR-011, FR-013A | AC-001, AC-008, AC-010, AC-011, AC-012, AC-014, AC-015A | Composer widget drives full image draft lifecycle using fakes. | ProviderScope with fake picker/preparer/uploader/repository; deterministic progress and failure controls | Select images, observe progress, fail one, retry/remove, enter alt, reorder, submit | UI state and submitted repository call match lifecycle and current order; submit gating is correct. | `app/test/feed/widgets/post_composer_sheet_test.dart` |
| IT-006 | FR-015, FR-016, FR-016A, NFR-002, NFR-003 | AC-017, AC-017A, AC-023, AC-024 | Feed card renders carousel using feed image cache and semantics. | Post with one/two/four images; `feedImageCacheManagerProvider` overridden with fake | Pump `PostCard`, page carousel | CachedNetworkImage/feed image loader uses feed cache; one image hides dots/count; multi-image shows legible indicators and image semantics. | `app/test/feed/widgets/post_card_test.dart` |
| IT-007 | FR-017, FR-018, FR-019, FR-019A, FR-020, FR-021 | AC-018, AC-019, AC-019A, AC-020, AC-021 | Gallery opens from image area and supports page/zoom/tap routing. | Pump feed card/gallery route with multiple images | Tap image area, swipe gallery, pinch gesture via tester, tap non-image area separately | Gallery opens; alt text displays; swipes page; pinch changes zoom state; non-image tap triggers post navigation not gallery. | `app/test/feed/widgets/post_card_test.dart`, future `post_image_gallery_test.dart` |
| IT-008 | NFR-003 | AC-024 | Feed and gallery image loads share feed cache manager. | Fake feed cache manager recording requested URLs | Render feed image and gallery image | Requested URLs go through `feedImageCacheManagerProvider`, not profile cache/default cache. | Future `app/test/feed/widgets/post_image_cache_test.dart` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Text-only top-level posts remain valid and text is still required. | FR-023, BR-005, RULE-007 | Keep/update `post_composer_sheet_test.dart`, `create_post_provider_test.dart`, and `post_api_client_test.dart` so text-only create still succeeds and empty text remains disabled/rejected even if image UI exists. |
| REG-002 | Reply composer remains text-only and forwards reply refs. | BR-003, FR-022, RULE-001, FR-023 | Existing reply tests continue to assert reply title/hint, mention prefill, and reply refs; add assertion that image controls are absent in reply mode. |
| REG-003 | Existing post card actions and non-media taps continue working. | FR-017, FR-021, FR-023 | Existing `post_card_test.dart` action/tap tests remain green; add image-card variant asserting action buttons do not open gallery. |
| REG-004 | No video behavior is introduced. | BR-004 | Validation/unit tests reject video MIME/types; composer does not present video-specific UI. |
| REG-005 | Existing AppView auth/error conventions remain in use. | NFR-001, RULE-005 | API tests still use shared Dio/error mapping; no PDS token is requested/returned; copy tests/manual review ensure privacy is not implied. |
| REG-006 | Posts without images parse and render without media UI. | FR-014, FR-015, FR-023 | Existing `post_test.dart` minimal payload and `post_card_test.dart` text-only rendering remain green. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Successful uploaded image blob | `{blob: {$type: 'blob', ref: {$link: 'bafkimage1'}, mimeType: 'image/jpeg', size: 253496}, cid: 'bafkimage1', mime: 'image/jpeg', size: 253496}` | AT-001, IT-001, IT-002, UT-009 |
| TD-002 | Multi-image post response | Post JSON with 4 images: JPEG/PNG/WebP mix, `alt`, `size`, `aspectRatio`, `thumb`, `fullsize` in fixed order | AT-003, UT-008, IT-003, IT-006, IT-007 |
| TD-003 | Invalid selections | `image/gif`, `video/mp4`, empty MIME, prepared bytes `maxImageBytes + 1`, selection count `maxImages + 1` | AT-002, UT-002, UT-003, REG-004 |
| TD-004 | Metadata-rich image fixtures | JPEG with GPS/camera/timestamp/software/comment metadata; PNG with alpha; WebP sample if supported by chosen library | UT-004, IT-004, MAN-002 |
| TD-005 | Alt text boundaries | Empty/whitespace, 1 char, 300 chars, 301 chars, craft-detail alt text | AT-002, UT-006, UT-013 |
| TD-006 | Aspect ratio cases | Known 919x2000 tall image, 2000x919 wide image, 1:1 image, missing dimensions | UT-010, UT-011, IT-006 |
| TD-007 | Upload lifecycle fakes | Preparer success/failure, upload progress 0/50/100, upload failure after deletion, out-of-order completion | AT-001, AT-002, UT-005, UT-007, IT-005 |
| TD-008 | Indicator contrast backgrounds | Very light, very dark, mixed busy image placeholders | UT-012, IT-006, MAN-004 |

## 8. Manual Checks
| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | RULE-005 | Public media wording review | Review composer, error, help, and any permission/copy strings around image posting. | Copy does not imply uploaded PDS post images are private. |
| MAN-002 | FR-004, FR-004A, FR-004B, FR-004C | Real-photo metadata stripping smoke test | On iOS and Android, select a real camera photo with GPS/camera EXIF; post or intercept prepared bytes in a debug/test build; inspect metadata. | GPS/camera/timestamp/comment metadata is removed when removable; image displays correctly; transparency is preserved for PNG. |
| MAN-003 | FR-020, FR-021 | Physical gesture feel check | On a touch device, scroll a feed with image posts, swipe carousel, pinch inline, open gallery, pinch/swipe in gallery. | Gestures feel distinct and usable; no accidental thread navigation during image interaction. |
| MAN-004 | FR-016A | Visual contrast check for count/dots | Test feed carousel over light, dark, and busy images. | Count and dots remain readable due to contrast/backing treatment. |
| MAN-005 | FR-019 | Hero transition polish check | Tap feed images into gallery and back on phone-sized and tablet-sized surfaces. | Transition feels coherent and does not flash wrong image/order. |
| MAN-006 | FR-001, RISK-004 | Platform picker/permission smoke test | Fresh install on iOS/Android; deny and allow image picker permissions; select multiple images. | Permission prompts/copy are acceptable; denied permissions fail gracefully; allowed selection works. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exhaustive metadata stripping cannot be proven by unit tests alone. | FR-004, FR-004A, FR-004B, FR-004C | Real platform pickers/libraries and image formats may preserve or normalize metadata differently. | Combine unit fixtures with MAN-002 on real devices; document any library-specific residual metadata. |
| GAP-002 | Pinch-to-zoom gesture quality is only partially automatable. | FR-020, FR-021 | Widget tests can verify recognizers/state but not full human feel across devices. | Use automated gesture tests plus MAN-003 before approval. |
| GAP-003 | Hero transition polish is difficult to assert exhaustively. | FR-019 | Automated tests can verify navigation and hero tags, but visual smoothness needs review. | Use MAN-005 during implementation review. |
| GAP-004 | App-level config can drift from backend limits. | FR-001, FR-002, RULE-002, RULE-008 | No backend-served config endpoint is in scope. | Add UT-001 and implementation comments tying defaults to backend contract; revisit if backend limits become dynamic. |
| GAP-005 | Actual CDN image loading is not covered by local tests. | FR-014, FR-015, NFR-003 | Widget tests should fake cache/image streams rather than fetch network images. | Use fake cache managers in automation; optional manual smoke against dev/real AppView later. |

## 10. Out Of Scope
- Backend API, lexicon, or AppView validation tests; this Flutter stage consumes the backend contract from `docs/changes/2026-05-19-appview-image-blobs/`.
- Video upload/playback tests because video remains out of scope.
- Image-only post acceptance tests because this slice keeps top-level post text required.
- Avatar/banner upload tests.
- Orphaned blob cleanup tests because deletion from draft intentionally does not call cleanup.
- Full visual snapshot testing for every aspect ratio/device combination; targeted widget tests plus manual checks are sufficient for this stage.

## 11. Handoff To Document Review
- Requirements file: `02-requirements.md`
- Test specification: `03-acceptance-tests.md`
- Next review artifact: `04-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-20-flutter-image-posts/`
- Recommended first failing test for implementation: `UT-001` for local app media config constants in a future `app/test/feed/media/media_config_test.dart`.
- Suggested test order for implementation:
  1. `UT-001` media config constants.
  2. `UT-002` / `UT-003` image selection and prepared-byte size validation.
  3. `UT-004` metadata stripping policy with fixture images.
  4. `IT-001` upload API client for `/v1/blobs/images`.
  5. `UT-008` post image response model parsing.
  6. `UT-009` / `IT-002` create-post image payload shape in composer order.
  7. `UT-005` / `UT-006` draft lifecycle and submit gating.
  8. `IT-005` composer widget end-to-end with fakes.
  9. `UT-011` / `IT-006` feed carousel layout/cache/semantics.
  10. `UT-013` / `IT-007` gallery alt text, swipe, tap routing, and zoom state.
  11. `REG-001` through `REG-006` existing behavior regression suite.
  12. Manual checks `MAN-001` through `MAN-006` before implementation review.
- Commands discovered:
  - Full Flutter suite: `cd app && flutter test`
  - Focused examples: `cd app && flutter test test/feed/data/post_api_client_test.dart`, `cd app && flutter test test/feed/widgets/post_composer_sheet_test.dart`, `cd app && flutter test test/feed/widgets/post_card_test.dart`
  - Code generation after model/provider/router changes: `cd app && dart run build_runner build --delete-conflicting-outputs`
  - Existing backend suite, if backend regressions are suspected: `just test` after `just dev-d`
- Blocking gaps: None blocking for implementation. High-risk behaviors have automated coverage targets plus manual checks where automation is insufficient.
