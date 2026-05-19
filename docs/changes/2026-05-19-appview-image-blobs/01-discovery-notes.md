# Discovery Notes: AppView Image Blob Handling

## Initial Request
Add backend/AppView support for blobs, limited to images for this pass. Video is out of scope. The user wants the flow to follow Bluesky image handling where appropriate, then handle Flutter/frontend work later.

## Current Codebase Findings
- Relevant files:
  - `lexicon/social/craftsky/feed/post.json` already defines top-level `images` on `social.craftsky.feed.post`, with up to 4 images, required `alt`, accepted MIME types `image/jpeg`, `image/png`, `image/webp`, and `maxSize: 15728640` (15 MB).
  - `appview/internal/lexicon/craftsky/feedpost.go` contains generated image/blob types for that lexicon.
  - `appview/internal/index/craftsky_post.go` already flattens indexed post images into `craftsky_posts.images` as `[{cid, mime, alt}, ...]`.
  - `appview/migrations/000010_craftsky_posts.up.sql` already has `images JSONB`.
  - `appview/internal/api/post_request.go` currently rejects `images` on `POST /v1/posts`.
  - `appview/internal/api/post_response.go` currently omits images from `PostResponse`.
  - `appview/internal/auth/pds_client.go` has record write/delete methods, but no blob upload method yet.
  - `appview/internal/auth/pds_client_indigo.go` wraps indigo's generic XRPC client for record methods. Indigo also has a generated `RepoUploadBlob` helper for `com.atproto.repo.uploadBlob`.
  - `appview/internal/api/profile_response.go` already synthesizes Bluesky CDN URLs for avatar/banner blobs.
- Existing patterns:
  - Flutter never holds PDS tokens; AppView mediates authenticated PDS writes via `auth.PDSClientFactory`.
  - `/v1/*` routes are REST/JSON where possible, authenticated with Craftsky bearer token plus `X-Craftsky-Device-Id`.
  - Write endpoints call the PDS and rely on firehose indexing for durable Postgres state; synthetic responses are acceptable immediately after a write.
  - Post rows are read through `PostStore`, then converted to a stable `PostResponse` wire shape.
- Current behavior:
  - AppView can index image refs from records created elsewhere, but does not expose them in read responses.
  - AppView cannot upload image blobs to a user's PDS.
  - AppView cannot create posts with images; `images` is an explicitly rejected field.
- Bluesky behavior reviewed:
  - Upload image bytes to `com.atproto.repo.uploadBlob` with the image MIME type as `Content-Type`.
  - PDS returns a blob object containing ref/CID, MIME type, and size.
  - Post records reference the uploaded blob and include required alt text.
  - Bluesky's current `app.bsky.embed.images` allows up to 4 images, accepts `image/*`, currently caps each image at 2 MB, and may carry optional `aspectRatio`.
  - Bluesky AppView image views expose render-ready `thumb` and `fullsize` URLs.
- Constraints discovered:
  - Lexicon changes are load-bearing and require project lexicon workflow/ADR. The confirmed direction avoids changing the record shape in this pass.
  - Image blobs on PDS are public under current atproto behavior; uploads must be treated as public media.
  - Blob upload should be bounded carefully to avoid large request memory pressure.
- Test/build commands discovered:
  - `just test` runs Go tests against compose Postgres.
  - `just fmt` runs `gofmt` and `go vet`.
  - `just lexgen` is needed only if lexicon JSON changes; not expected for the recommended direction.

## Clarifying Questions
### Q1: Should Craftsky switch post images to Bluesky-style `embed.images`, or keep the existing top-level `images` field?
Answer: Keep top-level images.

Decision / implication: The backend pass should preserve the existing Craftsky post lexicon shape and avoid reopening the prior top-level-vs-embed lexicon decision. "Like Bluesky" means upload flow and response ergonomics, not record-shape replacement.

### Q2: Should AppView enforce Bluesky's current 2 MB/image limit, or Craftsky's existing 15 MB/image lexicon limit?
Answer: Keep 15 MB.

Decision / implication: AppView validation should align with the Craftsky lexicon, not Bluesky's smaller `app.bsky.embed.images` cap.

### Q3: Should post responses return image URLs or only blob refs?
Answer: Return URLs.

Decision / implication: `PostResponse` should grow an additive image field with render-ready URLs, likely `thumb` and `fullsize`, plus metadata needed by the frontend.

### Q4: Should AppView process images server-side or only validate and pass through to the PDS?
Answer: Validate/pass-through.

Decision / implication: This pass should not add resizing, transcoding, or EXIF stripping. It should enforce authentication, supported MIME types, max size, and PDS upload only.

### Q5: Which approach should be documented as the confirmed direction?
Answer: Option A — separate AppView upload endpoint, top-level images, 15 MB, pass-through validation, URL-returning responses.

Decision / implication: Requirements should specify a two-step flow: upload image blob first, then create a post that references the returned blob metadata.

## Candidate Approaches
### Option A: Separate AppView Upload, Craftsky Top-Level Images
Summary: Add an authenticated AppView image upload endpoint that proxies raw image bytes to `com.atproto.repo.uploadBlob`, then extend `POST /v1/posts` and post read responses to use Craftsky's existing top-level `images` field.

Pros:
- Preserves the existing Craftsky lexicon and indexer shape.
- Closely matches Bluesky's upload-then-create flow.
- Reuses existing `craftsky_posts.images JSONB` storage.
- Avoids image-processing dependencies and lexicon churn.
- Cleanly reusable for future image upload surfaces.

Cons:
- A user can upload a blob and abandon the composer before creating a referencing post; the PDS will eventually clean up unreferenced blobs.
- Requires frontend to orchestrate two calls.
- Response URLs will initially depend on Bluesky CDN URL conventions, as profile images already do.

Risks:
- File upload endpoints have security and resource-pressure risks.
- PDS upload success followed by post-create failure leaves a temporary unreferenced blob.
- CDN URL synthesis may need replacement if Craftsky later runs its own media proxy/CDN.

### Option B: Combined Multipart Create Post
Summary: Let `POST /v1/posts` accept text plus image files in one multipart request. AppView uploads each file, builds the record, and creates the post.

Pros:
- Simpler frontend flow with one API call.
- Reduces abandoned-upload cases.
- AppView can validate post text and files together before making PDS writes.

Cons:
- Less Bluesky-like than the two-step blob upload flow.
- More handler complexity and less reuse.
- Partial failure remains possible if blob uploads succeed but record creation fails.

Risks:
- Multipart parsing and mixed file/JSON validation increase implementation and test scope.
- Harder to reuse for future profile-image or project-media upload flows.

### Option C: Add Craftsky Image Proxy/CDN Now
Summary: Upload blobs to the PDS but return Craftsky-owned image URLs and serve/proxy the blobs through AppView or a Craftsky media layer.

Pros:
- Avoids coupling clients to `cdn.bsky.app`.
- Gives Craftsky control over caching, moderation, transformations, and future CDN behavior.

Cons:
- Much larger scope than enabling backend image posts.
- Requires DID/PDS blob fetch routing, cache strategy, error semantics, and operational design.
- Duplicates media-serving infrastructure before product needs prove it necessary.

Risks:
- High operational and security complexity for a first image pass.
- Could delay frontend image work substantially.

## Recommendation
Recommended approach: Option A — separate AppView image upload endpoint plus top-level image support on existing post create/read flows.

Why:
- It is the smallest coherent backend slice that unblocks frontend image work.
- It honors the confirmed decisions: top-level Craftsky images, 15 MB limit, validate/pass-through only, return render-ready URLs.
- It fits existing AppView patterns for PDS-mediated writes and synthetic write responses.
- It avoids lexicon changes while preserving future options for aspect ratio, server-side processing, or a Craftsky image proxy.

## Scope Boundaries
In scope:
- AppView-only backend changes.
- Image blobs only; no video.
- Authenticated image upload endpoint that proxies to the caller's PDS.
- Validation for supported image MIME types, 15 MB max size, and upload request shape.
- Extend `PDSClient` and indigo adapter with blob upload capability.
- Extend `POST /v1/posts` to accept up to 4 uploaded image refs with required alt text.
- Write image refs into the existing top-level `images` field of `social.craftsky.feed.post` records.
- Extend post reads/list responses to include image view objects with URLs.
- Tests for request decoding/validation, PDS adapter behavior, handler behavior, response building, and store image scanning.

Out of scope:
- Flutter/frontend implementation.
- Video upload or playback.
- Lexicon record-shape changes from top-level `images` to `embed.images`.
- Server-side resizing, transcoding, image optimization, or EXIF stripping.
- Craftsky-owned image proxy/CDN.
- Profile avatar/banner upload.
- Project-field materialization beyond existing image support.
- Changing existing post indexing storage unless required to surface current `images JSONB` in reads.

## Risks And Review Recommendation
Risk level: High

Review recommended: Required

Reason: This introduces an authenticated file-upload surface, proxies user media writes to PDSes, returns public media URLs, and changes the public `/v1/posts` request/response contract. The implementation needs careful review for size limits, MIME validation, memory use, error handling, and public-media/privacy wording.

## Open Questions
- [ ] Exact upload route name: likely `POST /v1/blobs/images` or `POST /v1/images`; requirements should choose the canonical path.
- [ ] Exact upload response shape: whether to return the full atproto blob object only, or a normalized `{cid, mime, size, blob}` wrapper for easier create-post use.
- [ ] Exact post response image shape: recommended fields are `cid`, `mime`, `alt`, `thumb`, and `fullsize`; requirements should decide whether to include `size` if available.
- [ ] CDN URL extension behavior for feed images: reuse existing `mimeExt` mapping and omit URL fields for unknown MIME, or normalize all feed image URLs to `@jpeg` as many Bluesky examples do.
- [ ] Whether to add optional `aspectRatio` in a later lexicon pass. Current direction avoids this for now because the existing Craftsky `#image` object lacks that field.

## Decision Summary
- Keep the existing Craftsky top-level `images` record shape.
- Keep the existing 15 MB/image Craftsky limit.
- Validate and pass through image bytes; do not process images server-side.
- Return image URLs in post responses.
- Use a separate upload-then-create backend flow, similar to Bluesky's blob handling.
- Treat source files, tests, dependencies, and lexicon files as untouched during discovery; implementation belongs to later workflow stages.

## Handoff To Requirements
- Inputs the requirements agent should use:
  - This discovery note.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` for `/v1/`, auth headers, error envelope, and API conventions.
  - `docs/superpowers/specs/2026-05-04-feed-post-crud-endpoints-design.md` for current post create/read behavior and the explicit current omission of images.
  - `docs/superpowers/specs/2026-05-04-feed-post-indexing-design.md` for existing `craftsky_posts.images` storage.
  - `lexicon/social/craftsky/feed/post.json` for current image constraints.
  - `appview/internal/api/profile_response.go` for existing blob URL synthesis precedent.
- Requirements areas likely needed:
  - Authenticated image upload endpoint behavior.
  - Upload validation and error mapping.
  - PDS blob upload adapter contract.
  - Create-post request image shape and validation.
  - Post response image view shape and URL synthesis.
  - Backward compatibility for posts without images.
  - Public-media/privacy wording.
- Acceptance criteria areas likely needed:
  - Upload happy path and PDS error handling.
  - Rejection of unsupported MIME types, missing content type, oversized bodies, and unauthenticated requests.
  - Create post with 1-4 images writes top-level `images` to the PDS record.
  - Create post rejects more than 4 images and invalid/missing alt/blob refs.
  - Read/list responses include image URL objects for indexed images and omit/empty images for text-only posts.
  - Existing text-only post behavior remains unchanged.
