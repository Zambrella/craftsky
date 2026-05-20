# Discovery Notes: Flutter Image Posts

## Initial Request
Wire the newly implemented AppView image upload backend into the Flutter app so a signed-in user can add images to a top-level post, see upload progress over selected image previews, delete selected images before posting, submit only after uploads are complete, and see posted images rendered in feed cards. Image posts should use a carousel for multiple images, support tap-to-full-screen gallery with swipe, use a hero transition where appropriate, and support pinch-to-zoom directly in the feed and in the full-screen viewer.

## Current Codebase Findings
- Relevant backend workflow: `docs/changes/2026-05-19-appview-image-blobs/`.
  - `POST /v1/blobs/images` accepts one raw image body with MIME `image/jpeg`, `image/png`, or `image/webp`, max 15 MB, and returns `{blob, cid, mime, size}`.
  - `POST /v1/posts` now accepts top-level `images[]` with `{image, alt, aspectRatio?}`.
  - Post read/list responses now include `images[]` with `{cid, mime, size, alt, aspectRatio, thumb, fullsize}` when indexed image metadata is available.
- Relevant ADRs:
  - `adr/003-post-image-aspect-ratio.md` accepts optional `images[].aspectRatio = {width,height}` in the post lexicon.
  - `adr/004-feed-card-image-display.md` accepts feed card image display as an inline bounded-aspect carousel with count, dots, tap-to-gallery, aspect-ratio-aware layout, and feed-safe min/max height bounds.
- Current Flutter behavior:
  - `app/lib/feed/models/post.dart` explicitly says `images` is omitted and the model has no image fields.
  - `PostApiClient.createPost`, `PostRepository.create`, `ApiPostRepository.create`, and `CreatePost.create` support text and optional reply only.
  - `PostComposerSheet` is text-only. Submit is gated by non-empty text, <=2000 characters, and create loading state.
  - `PostCard` renders author, text, actions, delete menu, and normal card tap behavior only; no image UI exists.
- Existing image patterns:
  - `cached_network_image` and `flutter_cache_manager` are already dependencies.
  - `shared/image/image_cache_managers.dart` already defines `FeedImageCacheManager`, currently with no feed image call sites.
  - `ProfileAvatar` shows the current pattern for using `CachedNetworkImage`, a Riverpod cache manager provider, placeholder, error fallback, and short fade durations.
- Existing app architecture patterns:
  - API access goes through shared Dio from `shared/api/providers/dio_provider.dart`, with session auth and device ID interceptors attached.
  - API clients unwrap errors through `unwrapApi` and use camelCase JSON per app conventions.
  - Models use `dart_mappable`; adding image request/response models will require generated mapper updates.
  - Route definitions use `go_router_builder`; full-screen gallery routes may require generated router updates unless implemented as a direct `Navigator` overlay/page from the widget.
- Dependency gap:
  - No image picking package is currently present. A future implementation will likely need an image selection dependency such as `image_picker` or equivalent, plus platform permission handling/copy as required by that package.
  - No compression/transcoding dependency is currently present, and the chosen direction does not need one.
- Test/build commands discovered:
  - Backend: `just test` for Go tests when compose Postgres is running.
  - Flutter tests are under `app/test/**`; likely command is `cd app && flutter test`.
  - Generated Dart code exists for providers/router/mappers; likely command after model/provider/router changes is `cd app && dart run build_runner build --delete-conflicting-outputs`.

## Clarifying Questions
### Q1: For the first Flutter image-posting implementation, how should the app handle selected image bytes before upload?
Answer: Validate only.

Initial decision / implication: The app would upload original selected bytes unchanged. It would validate count, apparent MIME/type, and size client-side where possible, but would not resize, compress, transcode, strip EXIF, or otherwise alter image content in this slice.

Later requirements review update: This decision was superseded for privacy. The app should strip EXIF/GPS/camera and other non-essential embedded metadata before upload while preserving visible image content and supported format compatibility. This is metadata stripping only, not compression, resizing, or transcoding for optimization.

### Q2: Backend image posts require non-empty alt text per image. What should the first app UX do for alt text?
Answer: Require alt text before posting.

Decision / implication: Each selected image needs an alt-text entry point/field. Submit should remain disabled until all selected images have uploaded successfully and each selected image has non-empty alt text.

### Q3: For pinch-to-zoom, what should be in scope for the first implementation?
Answer: Feed and full-screen.

Decision / implication: Inline feed media and the full-screen gallery both need pinch-to-zoom. This increases gesture-handling risk because feed cards also need vertical scrolling, carousel horizontal paging, image-area tap, and non-image card tap behavior.

### Q4: Should image attachment be available when composing replies, or only for top-level posts in this first app integration?
Answer: Top-level posts only.

Decision / implication: The existing reply composer remains text-only in this slice. Image selection/upload UI should be hidden or disabled when `PostComposerSheet` is opened for a reply.

### Q5: Please confirm the discovery direction before writing notes.
Answer: Confirm Option A.

Decision / implication: Discovery should hand off a full first-pass top-level image posting and rendering integration, constrained by validate-only originals, required alt text, feed and full-screen zoom, and text-only replies.

## Candidate Approaches
### Option A: Full first-pass app integration, originals only
Summary: Implement top-level image post composition and image post display end-to-end in the Flutter app using the existing backend contract. Pick up to the configured image-count limit, validate locally, strip non-essential metadata, upload prepared bytes through AppView, show preview/progress/delete/reorder/alt UI, create text posts with uploaded blob metadata and aspect ratio, parse returned image metadata, render feed images as ADR 004 carousel, and provide full-screen gallery with swipe, hero transition, and pinch zoom. Also support pinch zoom inline in feed.

Pros:
- Meets the user's primary goal in one coherent feature slice.
- Uses the backend exactly as implemented: upload first, create post second, top-level Craftsky `images` field.
- Aligns with accepted feed display ADR 004.
- Avoids compression/transcoding quality and dependency decisions.
- Lets existing `FeedImageCacheManager` become active at its intended feed-image call sites.

Cons:
- Larger Flutter implementation surface across dependency setup, API models, upload state, composer UI, feed rendering, gallery, accessibility, and tests.
- Gesture handling is complex because inline media must distinguish vertical feed scroll, horizontal carousel swipe, pinch zoom, image tap, and card tap.
- Requires careful state cleanup for deleted images, failed uploads, composer dismissal, and retries.

Risks:
- Inline pinch-to-zoom may conflict with `PageView`/carousel gestures and parent feed scrolling.
- Upload progress availability depends on the selected HTTP/upload implementation and request body shape.
- Client-side MIME/size validation can reduce obvious failures but cannot replace backend validation.

### Option B: Display first, compose second
Summary: First update `Post` parsing, feed carousel, full-screen gallery, and image cache usage for already-indexed image posts. Defer picker/upload/composer changes to a later workflow.

Pros:
- Smaller first implementation with lower state-management risk.
- Validates read/display behavior against the new backend response contract before write UX.
- Easier to isolate carousel/gallery gesture tests.

Cons:
- Does not satisfy the immediate user goal that a user can add images to a post.
- Requires a second workflow to complete the end-to-end feature.
- May duplicate design/test effort if composer previews need similar widgets later.

Risks:
- Product feedback on posting flow would be delayed.
- Existing backend upload work remains unused by the app until a later change.

### Option C: Enhanced image processing flow
Summary: Add client-side resizing/compression or user-choice optimization before upload, in addition to the picker/upload/display flow.

Pros:
- Can reduce upload time and avoid some 15 MB failures.
- May improve mobile data usage for very large camera images.

Cons:
- Conflicts with the confirmed validate-only/originals direction.
- Adds dependencies and platform-specific image processing behavior.
- Requires extra product decisions around quality, dimensions, metadata, EXIF, and whether aspect ratio describes original or transformed bytes.

Risks:
- More potential for image quality regressions or inconsistent platform results.
- Larger test and QA matrix.

## Recommendation
Recommended approach: Option A — full first-pass app integration, originals only.

Why: It directly satisfies the requested user outcome while staying aligned with the backend's completed upload/create/read contract and accepted feed display ADR. The scope remains coherent by limiting posting to top-level posts, requiring alt text, stripping only non-essential metadata for privacy, avoiding video/compression/resizing/transcoding, and treating advanced gesture behavior as an explicit high-risk requirement rather than an accidental implementation detail.

## Scope Boundaries
In scope:
- Top-level post image attachments only; replies remain text-only.
- Top-level posts still require text; images are optional attachments for this slice.
- Select up to a locally configured maximum number of images, defaulting to the backend-supported limit.
- Client-side validation for supported image type, configured image size limit, and configured alt-text length.
- Local image preparation before upload: strip EXIF/GPS/camera and other non-essential embedded metadata while preserving visible image content and supported format compatibility.
- Upload prepared image bytes through AppView after validation and metadata stripping.
- Per-image local preview, upload progress/status overlay, retry/error handling requirements, delete-before-post behavior, and orphaned uploaded blob acceptance.
- Required alt text for each selected image before submit, capped by local media config at 300 characters.
- Aspect ratio capture from selected image dimensions where feasible and passing it in create-post requests.
- Manual reordering of selected images before submit; final create payload order follows the current composer order, not upload completion order.
- `Post` model/image response parsing and create-post request support.
- Feed-card image carousel per ADR 004, using `FeedImageCacheManager`.
- Full-screen gallery with image swipe, tap entry from feed image area, hero transition where feasible, and pinch zoom.
- Inline feed pinch zoom, with explicit gesture conflict requirements.
- Accessibility semantics for alt text and gallery entry points.
- Text-only post regressions.

Out of scope:
- Video selection, upload, display, or playback.
- Image-only posts; this slice keeps backend-compatible text-required post creation.
- Image compression, resizing, transcoding, or optimization beyond metadata stripping.
- AppView/backend API changes beyond consuming the existing image upload/create/read contract.
- Craftsky-owned image proxy/CDN changes.
- Avatar/banner upload.
- Adding images to replies in this first pass.
- Cleanup of orphaned blobs after user deletion or abandoned composer sessions.

## Risks And Review Recommendation
Risk level: High.

Review recommended: Required before implementation.

Reason: This is a broad user-visible Flutter change involving local media selection permissions, authenticated media upload through AppView, public PDS-hosted media, create-post request expansion, new dependencies, generated model/provider/router code, accessibility obligations, and complex gesture behavior in feed cards and gallery views. Inline pinch zoom in a carousel inside a vertical feed is the highest UI risk.

## Open Questions
- [ ] Exact feed carousel min/max height bounds and fallback aspect ratio should be set in requirements or implementation design, consistent with ADR 004.
- [ ] The image selection package and platform permission copy need to be chosen during requirements/design.
- [ ] Requirements should define upload failure behavior: retry affordance, whether failed images can be deleted, and how errors are surfaced.
- [ ] Requirements should define whether users may add more images after some uploads have already completed, up to the configured image cap.
- [x] Requirements should define whether image order is fixed by selection order and whether deletion preserves remaining order. Decision: image order follows the current composer order, including manual reordering, independent of upload completion order.
- [ ] Requirements should define whether full-size or thumbnail URLs are used in each surface: composer preview, feed carousel, and full-screen gallery.

## Decision Summary
- Use the backend's upload-then-create flow: prepare/upload images after selection, then include returned blob metadata in `POST /v1/posts`.
- Limit posting support to top-level posts; replies remain text-only.
- Keep top-level post text required; images are optional attachments and image-only posts are out of scope.
- Validate selected images client-side where possible, strip EXIF/GPS/camera and other non-essential metadata before upload, and upload prepared bytes.
- Enforce app-level media config: backend-aligned image count and size defaults, JPEG/PNG/WebP, and 300-character max alt text.
- Disable submit while any selected image is preparing, uploading, failed without resolution, missing/too-long alt text, or while create-post is loading.
- Preserve current composer order in the create payload, including manual reordering and independent of upload completion order.
- Render feed images with ADR 004's inline bounded-aspect carousel and count/dot indicators.
- Support image-area tap to full-screen gallery and preserve non-image card tap behavior.
- Support pinch zoom both inline in feed media and in full-screen gallery.
- Treat deletion of already-uploaded images before posting as acceptable orphaning; no cleanup endpoint or client cleanup is required.

## Handoff To Requirements
- Inputs the requirements agent should use:
  - This file.
  - `docs/changes/2026-05-19-appview-image-blobs/` for backend API contract and constraints.
  - `adr/003-post-image-aspect-ratio.md` for aspect-ratio metadata.
  - `adr/004-feed-card-image-display.md` for feed image display policy.
  - Flutter files: `app/lib/feed/models/post.dart`, `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`, `app/lib/feed/data/api_post_repository.dart`, `app/lib/feed/providers/create_post_provider.dart`, `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/feed/widgets/post_card.dart`, `app/lib/shared/image/image_cache_managers.dart`, `app/lib/shared/image/image_cache_providers.dart`, `app/lib/router/router.dart`, and `app/pubspec.yaml`.
- Requirements areas likely needed:
  - Image selection and validation.
  - Upload lifecycle and progress state.
  - Composer preview, alt text, deletion, retry, and submit gating.
  - API request/response models and create-post payload shape.
  - Feed carousel display and aspect-ratio layout.
  - Inline and full-screen gestures.
  - Full-screen gallery navigation/hero behavior.
  - Accessibility and alt text semantics.
  - Error messaging and public-media wording.
  - Text-only and reply regression behavior.
- Acceptance criteria areas likely needed:
  - Successful top-level image post from selection through upload, create, and rendered feed card.
  - Rejection or disabled selection for unsupported type, oversize image, and more images than the configured cap.
  - Submit disabled during upload, on upload failure, and when alt text is missing.
  - Deleting selected images removes them from the create payload without requiring blob cleanup.
  - Feed carousel renders one, two, and four images with count/dots and stable aspect-ratio bounds.
  - Image-area tap opens gallery; non-image card tap still opens the thread.
  - Pinch zoom works inline and in full-screen without breaking vertical scrolling or horizontal image paging.
  - Full-screen gallery supports swipe between images and back/dismiss behavior.
  - Image alt text is exposed to accessibility semantics.
  - Existing text-only top-level posts and text-only replies continue to work.
