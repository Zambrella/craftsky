# Requirements: Video Posts

## 1. Initial Request
Add the ability for users to upload videos that are ultimately stored on their
PDS and to play videos stored on a PDS when those videos are embedded in
Craftsky posts. Follow the Bluesky video-upload guidance. The current product
limits are 10 minutes per video and 300 MB per file.

## 2. Current Codebase Findings
- Relevant files:
  - `lexicon/social/craftsky/feed/post.json` defines the Craftsky post record,
    top-level images, and an open `embed` union that currently supports quote
    and external embeds.
  - `appview/internal/api/blob.go` implements the authenticated, AppView-mediated
    image blob upload flow.
  - `appview/internal/api/post_request.go` accepts images and quote/external
    embeds but currently rejects a video field as unknown.
  - `appview/internal/index/craftsky_post.go` indexes images and retains the raw
    embed in the existing post record projection.
  - `appview/internal/api/post_response.go` builds the canonical post response
    and currently exposes images and external cards, but no native video view.
  - `app/lib/feed/widgets/post_composer_sheet.dart` owns immediate top-level
    composition, media submission gating, drafts, and scheduling integration.
  - `app/lib/feed/widgets/composer_image_attachment_section.dart` is the shared
    image attachment widget used by standard posts, project posts, business
    products, and business events; it currently exposes image-only copy and an
    image-only add action.
  - `app/lib/feed/widgets/submission_blocking_overlay.dart` currently shows only
    generic publishing/scheduling feedback while submission is blocked.
  - `app/lib/drafts/data/file_local_post_draft_repository.dart` persists and
    verifies image-oriented media as whole byte arrays; maximum-size video
    drafts require a streaming file path instead.
  - `app/lib/feed/models/post.dart` and `app/lib/feed/widgets/post_card.dart`
    model and render post media, but support only images and external cards.
  - `app/pubspec.yaml` includes image and YouTube media dependencies but no
    native MP4/HLS playback dependency. Current package research selected the
    verified-publisher `media_kit`, `media_kit_video`, and
    `media_kit_libs_video` stack: it supports HLS, URI-based WebVTT tracks,
    adaptive controls, full screen, lifecycle streams, and all six Flutter
    targets present in this repository.
- Existing patterns:
  - Flutter clients authenticate to AppView. OAuth access/refresh tokens and
    DPoP keys remain in AppView.
  - Writes are mediated by AppView and sent to the authenticated user's PDS;
    indexed reads come from AppView.
  - `/v1/*` uses camelCase JSON, bearer authentication, a device ID, and the
    standard `{error, message, requestId}` error envelope.
  - Post-shaped endpoints share one canonical response model.
- Current behavior:
  - Craftsky supports text posts with optional images, quote embeds, or external
    cards. Native video upload, indexing, response hydration, and playback are
    absent.
  - YouTube links may play through the separate external-card consent flow;
    this is not native PDS video support.
- Constraints discovered:
  - Bluesky's recommended flow obtains a short-lived PDS-signed service JWT for
    `com.atproto.repo.uploadBlob`, uploads directly to
    `app.bsky.video.uploadVideo`, and polls `app.bsky.video.getJobStatus` until a
    blob is available.
  - The service JWT is not the user's OAuth access token and does not include a
    refresh token or DPoP private key. It is nevertheless a replayable bearer
    credential until expiry and must be treated as a secret.
  - The current `app.bsky.embed.video` lexicon accepts `video/mp4` blobs up to
    exactly 300,000,000 bytes and supports alt text, aspect ratio, captions, and
    a presentation hint.
  - `app.bsky.video.getUploadLimits` currently reports `canUpload`, remaining
    daily video/byte allowances, and service error/message fields.
  - Bluesky-hosted accounts may require verified email and are subject to
    account-level daily video limits imposed by the video service.
  - Adding a video variant to the Craftsky post lexicon requires an ADR and
    regenerated Go lexicon types before implementation.
  - The standard Bluesky AppView builds `app.bsky.embed.video#view` playback
    URLs from percent-encoded author DID and video CID using configurable
    playlist and thumbnail templates. A verified live response on 2026-09-03
    used `https://video.bsky.app/watch/{did}/{cid}/playlist.m3u8` and the
    matching `thumbnail.jpg` path.
- Test/build commands discovered:
  - `just test` runs Go tests against the compose Postgres service.
  - Flutter tests and static analysis run from `app/` using the repository's
    existing Flutter/Dart workflow.

## 3. Clarifying Questions And Decisions
### Q1: Which upload architecture should Craftsky use?
Answer: Direct Flutter upload to the Bluesky video service using a short-lived
service token obtained by AppView from the user's PDS.

Decision / implication: AppView retains OAuth access/refresh tokens and DPoP keys
but may return one purpose-bound service JWT to Flutter. Flutter keeps it only in
memory, sends it only to the configured `video.bsky.app` upload endpoint, streams
the MP4 directly, and polls public job status directly. AppView remains the post
write boundary and independently verifies the completed job owner and exact blob
before publication.

### Q2: How should native video be represented in a Craftsky post?
Answer: Use the standard `app.bsky.embed.video` object.

Decision / implication: Reuse the standard `app.bsky.embed.video` object as a
new variant in the existing open `social.craftsky.feed.post.embed` union. This
is the narrowest interoperable choice and avoids inventing a duplicate video
record shape. The required lexicon ADR must ratify the confirmed design before
implementation.

### Q3: What does playback of a PDS-hosted video require?
Answer: The initial request explicitly requires playback of videos stored on a
PDS.

Decision / implication: AppView responses must provide a playable source for
the embedded blob. This release supports the video service's processed HLS
playlist only. Direct playback of the raw PDS MP4 blob is deferred.

### Q4: When should video upload and processing begin, and how should users add media?
Answer: Video must remain local while composing and upload only after the user
chooses to publish. The publishing screen must show general video upload and
processing feedback. When video is supported, the existing image attachment
widget must say so and ask whether the user wants to add photos or a video. The
widget must accept a capability flag because business products and other
image-only consumers must not offer video.

Decision / implication: Selecting a video creates local composer state only.
The publish action starts upload, processing, and final post creation as one
blocking submission sequence. The shared attachment widget gains an explicit
`supportsVideo` capability (name illustrative but preferred), capability-aware
copy, and a photo/video choice when enabled; existing image-only consumers keep
their current behavior.

### Q5: What additional product and operational decisions were confirmed during requirements grilling?
Answer:
- Use `app.bsky.embed.video`; allow one native media type per post; and do not
  support video quote-posts.
- Enable video for immediate top-level standard and project posts. Keep replies
  and scheduled posts out of scope.
- Save selected video locally in both standard and project drafts without
  uploading it. Apply a decimal 1,000,000,000-byte saved-video quota per account,
  block over-quota saves without eviction, and store a small local poster frame.
- Show the photo/video choice only while the attachment widget is empty. Once
  media exists, keep controls specific to that media type.
- Prompt for video alt text but allow users to skip it; provided alt text is
  limited to 1,000 graphemes. Render existing WebVTT captions but do not upload
  captions in this release.
- Select existing videos only. If duration cannot be read locally, allow the
  video service to make the final decision.
- Allow cancel without an orphan-blob warning during upload and processing;
  disable cancel once final post creation starts.
- If the app is interrupted, retain the local draft but restart publication
  from scratch when the user next publishes. A manual retry also restarts the
  full sequence.
- Show upload and processing percentages when available plus a final publishing
  stage; use indeterminate progress when percentages are unavailable.
- Preflight provider/account eligibility when Video is chosen and recheck at
  publish. Keep an ineligible Video choice visible with an explanation. Show
  remaining daily quota only when constrained.
- Keep service credentials and remote job continuation state ephemeral on the
  client. Poll with backoff from one to five seconds and honor `Retry-After`.
- Always allow metered-network uploads without an extra warning or confirmation.
- A failed or ambiguous upload retry starts a fresh authorization/upload
  sequence. Treat `already_exists` with a blob as completion; the video API has
  no documented Craftsky idempotency-key contract.
- Support HLS playback only in this release; do not fall back to raw PDS MP4.

Decision / implication: These decisions replace the earlier assumptions and
open questions they resolve. Local draft support is now in scope, while
scheduled publication and raw PDS playback remain explicit non-goals.

### Q6: Which Flutter playback package and platform scope should Craftsky use?
Answer: Use `media_kit` with `media_kit_video` and `media_kit_libs_video` on
Android, iOS, macOS, Windows, Linux, and web.

Decision / implication: `media_kit` is preferred over Flutter's low-level
`video_player` because the latter omits Windows/Linux and requires a separate
controls layer. It is preferred over `better_player_plus` because that package
supports only Android/iOS and uses an unverified uploader. Craftsky shall wrap
`media_kit` behind an app-owned player/controller abstraction, initialize it
once at startup, open media with `play: false`, use URI-based WebVTT subtitle
tracks, and explicitly pause/dispose players as cards leave active use. Real
HLS, captions, and full-screen behavior remain subject to the platform matrix
checks because web playback is browser-dependent and native builds add bundled
media libraries.

### Q7: How will Flutter preflight limits and how will AppView build playback URLs?
Answer: Add authenticated `GET /v1/blobs/videos/limits`; derive HLS and
thumbnail URLs through configurable templates.

Decision / implication: The limits route returns camelCase `canUpload`, optional
`remainingDailyVideos`, optional `remainingDailyBytes`, and an optional stable
`reason` token: `email_unverified`, `quota_exhausted`,
`provider_unsupported`, or `unknown`. Upstream availability failures use the
standard API error envelope rather than being reported as account ineligibility.
AppView builds playlist and thumbnail URLs from percent-encoded author DID and
video CID. Defaults match the selected Bluesky service's current
`https://video.bsky.app/watch/{did}/{cid}/...` contract, while configuration
keeps CDN changes isolated. Craftsky does not query the Bluesky AppView for its
custom records.

### Q8: How is completion trusted after moving upload and polling to Flutter?
Answer: Flutter submits both the video-service job ID and returned BlobRef;
AppView independently verifies them before writing the post.

Decision / implication: AppView calls public
`app.bsky.video.getJobStatus`, requires `jobStatus.did` to equal the
authenticated DID, requires a completed result or documented `already_exists`
result with a blob, and requires the returned blob to exactly match the submitted
BlobRef. A mismatch, incomplete job, unknown job, or unavailable verification
blocks the post atomically. This avoids trusting an altered client while removing
AppView upload proxying and durable video-job state.

## 4. Candidate Approaches
### Option A: Direct PDS blob upload
Summary: Stream MP4 bytes through AppView directly to
`com.atproto.repo.uploadBlob`, then publish the post immediately.

Pros:
- Uses only the user's PDS and existing Craftsky write architecture.
- Avoids a Bluesky-operated processing dependency.
- Applies uniformly to PDS providers that accept the blob.

Cons:
- The post may be visible before downstream video processing is complete.
- Craftsky receives no standardized encoding/scanning progress.
- Raw source compatibility and efficient playback are less predictable.

Risks:
- Poor first-play experience and temporarily missing media.
- Greater variation between source videos and PDS implementations.

### Option B: AppView-mediated Bluesky preprocessing
Summary: AppView obtains scoped service auth, proxies the MP4 to Bluesky's video
service, persists/polls the job, and publishes the returned PDS blob reference.

Pros:
- Follows the supplied documentation's recommended method.
- Exposes upload and processing states before publication.
- Avoids publishing a post whose video is still being processed.
- Reuses the standard `app.bsky.embed.video` contract.

Cons:
- Adds an external service dependency and account-eligibility rules.
- Requires asynchronous job ownership, recovery, and error handling.
- May not work for every independently hosted PDS/account configuration.

Risks:
- Service policy, quota, availability, or compatibility changes can block new
  uploads even when a user's PDS is otherwise available.
- AppView bears the bandwidth, memory-pressure, timeout, and durable-job burden.

### Option C: Capability-based hybrid
Summary: Prefer video-service preprocessing but fall back to direct PDS upload
when preprocessing is unavailable or unsupported.

Pros:
- Broadest potential PDS compatibility.
- Retains preprocessing where available.

Cons:
- Introduces two publication paths with materially different behavior.
- Doubles failure, retry, testing, and support states.

Risks:
- Inconsistent user experience and difficult-to-diagnose provider differences.

### Option D: Ephemeral service-token handoff and direct client upload
Summary: AppView uses its server-held OAuth session to obtain a narrowly scoped
PDS service JWT, returns that credential to Flutter, and Flutter uploads/polls
directly against `video.bsky.app`. AppView verifies job ownership and blob
identity before creating the post.

Pros:
- Removes up to 300 MB of duplicate AppView ingress/egress per publication.
- Follows the direct video-service flow used by Bluesky clients.
- Removes private AppView job persistence, cleanup, and restart recovery.
- Keeps OAuth access/refresh tokens and DPoP keys server-side.

Cons:
- Flutter temporarily holds a replayable PDS-issued bearer credential.
- Requires a dedicated external transport that cannot leak either the service
  JWT or Craftsky bearer across hosts or redirects.
- Browser direct upload depends on `video.bsky.app` CORS behavior.
- AppView must make a final public status call to verify the untrusted result.

Risks:
- Token exposure through persistence, logs, crash reports, redirects, or broad
  HTTP interceptors could authorize blob upload until expiry.
- A video-service outage during final verification can block post creation after
  processing completed.

## 5. Recommended Direction
Recommended approach: Option D, ephemeral service-token handoff and direct client
upload, as explicitly selected by the user. Reuse `app.bsky.embed.video` in
Craftsky post records, derive processed playback URLs through configurable
Bluesky-compatible templates, and use `media_kit` for Flutter playback.

Why: This follows the supplied direct-upload guidance without making AppView a
300 MB media proxy. The exception is narrow: Flutter receives no OAuth token,
refresh token, or DPoP key, does not contact a PDS, and may use the service JWT
only for the exact video upload. Independent completion verification preserves
the untrusted-client boundary.

## 6. Problem / Opportunity
Craftsky users can share images but cannot demonstrate moving craft techniques,
progress, or finished work through native video. Craftsky also cannot render a
standard video blob embedded in an indexed post, leaving valid public PDS media
inaccessible in the app. Supporting asynchronous upload and federated playback
closes both sides of that user journey.

## 7. Goals
- G-001: Let an eligible signed-in user select one local MP4 and publish it on
  an immediate top-level Craftsky post, starting upload and processing only
  after publish is chosen by using an ephemeral, purpose-bound service JWT for a
  direct video-service upload.
- G-002: Enforce a 10-minute and 300,000,000-byte source-video limit before
  publication, with the video service remaining the final media validator.
- G-003: Index standard video embeds and return a canonical, render-ready video
  view from every post-shaped AppView response.
- G-004: Play embedded PDS-hosted video through the processed HLS playlist on
  supported Flutter platforms with accessible controls and soft failure.
- G-005: Preserve existing text, image, quote, external-card, and YouTube
  behavior for posts without native video.
- G-006: Preserve selected video locally in standard and project drafts without
  starting remote upload before publish.

## 8. Non-Goals
- NG-001: Multiple videos in one post.
- NG-002: Mixed native video with images, quote embeds, or external cards in the
  same post.
- NG-003: Video in replies, scheduled posts, imported Instagram posts, business
  events/products, avatars, or profile media.
- NG-004: Craftsky-owned video transcoding, optimization, scanning, thumbnail
  generation, CDN, or media proxying.
- NG-005: Video trimming, filters, editing, compression, format conversion, or
  capture inside Craftsky.
- NG-006: Uploading caption files or generating captions/transcripts in this
  slice.
- NG-007: Supporting source formats other than MP4 or source files above the
  stated limits.
- NG-008: A fallback direct-upload publication path when the Bluesky video
  service is unavailable or incompatible.
- NG-009: Changing the existing external YouTube consent/player flow.
- NG-010: Direct playback or fallback through the raw PDS MP4 blob in this
  release.
- NG-011: Recording video from the camera inside Craftsky.
- NG-012: Handing Flutter an OAuth access token, refresh token, DPoP key, generic
  PDS credential, or authority for any operation other than the exact video
  upload.
- NG-013: Persisting the service JWT or remote video job for automatic resume.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Eligible signed-in user | A Craftsky member whose account and PDS are accepted by the video service | Select a valid video, understand upload/processing state, recover from errors, and publish only when ready |
| Post viewer | A signed-in user viewing a native video post | See a stable poster or error state and play, pause, seek, mute, and enter full screen when HLS is available |
| Flutter app | The untrusted client coordinating selection, direct upload, polling, publication, and playback | Hold one upload-only service JWT ephemerally, contain it to the approved endpoint, and use stable AppView contracts for authorization/writes/reads |
| Craftsky AppView | The trusted backend mediating OAuth, service authorization, final job/blob verification, PDS writes, indexing, and read hydration | Keep OAuth/DPoP material server-side, issue only narrow credentials, and reject unverified completion |
| Bluesky video service | The selected preprocessing service | Receive scoped upload authorization, process the MP4, and store the resulting blob on the user's PDS |
| User's PDS | The authoritative store for the user's blob and post record | Accept the processed blob and standard video embed record |

## 10. Current Behavior
The post lexicon has no native video embed variant. `POST /v1/posts` rejects
video input, the AppView has no video job endpoints, the index/read paths do not
hydrate native video, and Flutter has no native video model, composer state, or
player. Existing image and YouTube behavior does not satisfy PDS video upload
or playback.

## 11. Desired Behavior
An eligible user taps the shared attachment control in an immediate top-level
composer. Because that surface supports video, its copy identifies both photos
and video and tapping it asks which media type to add. Selecting one MP4 creates
a local preview and validates locally available type, size, and duration data,
but does not upload or start a remote job. Image-only consumers of the shared
widget, including business products and events, do not mention or offer video.
Standard and project drafts may save the selected video and a local poster frame
in account-scoped local storage without starting upload.

When the user chooses publish, Flutter opens the existing blocking publishing
experience and starts the video sequence. The publishing screen gives general,
accessible feedback as the video is uploaded, processed, and the post is
created. Flutter asks AppView for a short-lived service JWT, keeps it in memory,
and streams the MP4 directly to the exact configured `video.bsky.app` upload
endpoint. Flutter polls public job status directly and normalizes failed,
ineligible, rate-limited, and retryable states. The publishing screen may be
canceled during upload or processing and shows stage labels plus percentages
when available. Flutter submits the completed job ID and BlobRef to AppView;
AppView independently verifies the owner, completion, and exact blob before
writing `app.bsky.embed.video`, after which cancellation is disabled.

When AppView indexes any valid supported video embed, canonical post responses
include video metadata, an HLS playlist when available, and a thumbnail when
available. Flutter shows a non-autoplaying inline HLS player with accessible
controls. Missing or failed HLS does not trigger a raw PDS fetch and must not
prevent the rest of the post from rendering.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must support an end-to-end native video-post flow in which the resulting video blob and post record belong to the publishing user's PDS. | Delivers the requested user-owned media outcome. | Prompt | AC-001, AC-007 |
| BR-002 | Business | Must | Craftsky must play supported standard video embeds found in indexed Craftsky posts, including embeds authored outside the Craftsky client. | Read interoperability is explicitly part of the request. | Prompt, codebase | AC-009, AC-010 |
| BR-003 | Business | Must | Flutter may receive only an ephemeral PDS-signed service JWT bound to the exact video upload operation; OAuth access/refresh tokens and DPoP keys must remain AppView-only. | Enables direct upload without converting Flutter into a general PDS/OAuth client. | Architecture amendment, user decision | AC-002 |
| BR-004 | Business | Must | Existing non-video post creation, rendering, and external YouTube behavior must remain available and unchanged except for additive native-video support. | Prevents regressions and conflation of native and external video. | Codebase | AC-016 |
| FR-001 | Functional | Must | AppView shall expose authenticated `POST /v1/blobs/videos/authorization` and return one short-lived upload authorization; Flutter shall then stream one raw `video/mp4` directly to the configured `https://video.bsky.app/xrpc/app.bsky.video.uploadVideo` endpoint. | Removes the 300 MB AppView media proxy while preserving server-mediated authorization. | User amendment, Bluesky guidance | AC-001, AC-003 |
| FR-002 | Functional | Must | AppView shall obtain a service JWT expiring after issuance and no more than 30 minutes later with only the user's PDS audience and `com.atproto.repo.uploadBlob` method; the response may contain that JWT and its expiry but no OAuth access/refresh token, DPoP key, PDS URL, or broader authority. | Establishes the narrow credential exception. | Bluesky docs, architecture amendment, user decision | AC-002 |
| FR-003 | Functional | Must | Flutter shall validate and stream the upload without buffering an additional complete file, reject sources above 300,000,000 bytes before requesting authorization, and use a dedicated external client that sends the service JWT only to the exact configured HTTPS origin and upload path. | Bounds device memory and bearer exposure. | User amendment, security review | AC-003, AC-014 |
| FR-004 | Functional | Must | Flutter shall poll public `app.bsky.video.getJobStatus` directly, normalize progress and failures locally, and retain the completed job ID and BlobRef only for the current publication attempt. | Video processing is asynchronous but no longer needs AppView job proxying. | Bluesky docs, user amendment | AC-004, AC-005 |
| FR-005 | Functional | Must | Before post creation, AppView shall independently query the submitted job ID and require that the service result is completed, `jobStatus.did` equals the authenticated DID, and the service BlobRef exactly matches the submitted BlobRef. | Prevents an altered client from supplying a foreign, incomplete, or arbitrary blob. | Security decision, user confirmation | AC-004, AC-007, AC-013 |
| FR-006 | Functional | Must | Flutter and AppView shall treat a video-service `already_exists` result that includes a matching BlobRef as successful completion. | The supplied documentation identifies this as an expected reusable-result path. | Bluesky docs | AC-006 |
| FR-007 | Functional | Must | `POST /v1/posts` shall accept one completed video payload under `embed.video` containing `jobId`, `blob`, optional alt text of at most 1,000 graphemes, and optional aspect ratio; after FR-005 verification it shall write the standard `app.bsky.embed.video` variant. | Reuses the interoperable shape without trusting client completion claims. | Recommended direction, user confirmation | AC-007, AC-041 |
| FR-008 | Functional | Must | The Craftsky post lexicon shall add `app.bsky.embed.video` to its existing open `embed` union without making any new video field required on existing posts. | Provides additive record compatibility. | Lexicon guidance, recommended direction | AC-008 |
| FR-009 | Functional | Must | The post indexer shall recognize valid `app.bsky.embed.video` records and preserve at least blob CID, MIME type, size, alt text, aspect ratio, and valid WebVTT caption blob references/languages when present. | Read hydration and confirmed caption rendering require stable indexed metadata. | Prompt, codebase, requirements grilling | AC-009, AC-021 |
| FR-010 | Functional | Must | Every canonical post-shaped AppView response shall expose an optional `video` object with indexed metadata, an HLS `playlist` when available, an optional `thumbnail`, and safe caption metadata when available. | Gives Flutter one stable HLS rendering contract across surfaces. | Prompt, codebase, requirements grilling | AC-009, AC-010 |
| FR-011 | Functional | Must | AppView shall derive the processed HLS playlist and optional thumbnail from percent-encoded author DID and video CID through configurable URL templates. Defaults shall match the selected Bluesky service's current `https://video.bsky.app/watch/{did}/{cid}/playlist.m3u8` and `thumbnail.jpg` contract. No raw PDS MP4 source is exposed in this release. | Locks the confirmed HLS-only playback boundary while isolating CDN contract changes. | Requirements grilling, Bluesky AppView source and live response | AC-010, AC-011 |
| FR-012 | Functional | Must | Flutter shall let a signed-in user select one existing MP4 for an immediate top-level post, validate and preview it locally without uploading it, allow removal or replacement before submission, and start remote upload/processing only when the user chooses publish. | Aligns video behavior with the confirmed publish-time image workflow and excludes in-app recording. | User feedback, requirements grilling | AC-001, AC-012, AC-026, AC-042 |
| FR-013 | Functional | Must | Flutter shall validate MIME/type, source byte size, and duration where platform metadata permits before upload, while allowing a source with unknown local duration to proceed to authoritative video-service validation. | Avoids predictable large uploads without falsely rejecting valid files when platform metadata is unavailable. | Prompt, requirements grilling | AC-003, AC-015 |
| FR-014 | Functional | Must | Flutter shall parse the canonical video response and render native video on standard full post surfaces, including timeline/feed cards, profile post lists, saved/search results that use the canonical post model, and post detail/thread views. | Users must be able to play posted video wherever normal posts are viewed. | Prompt, codebase | AC-010 |
| FR-015 | Functional | Must | Flutter shall use an app-owned abstraction over `media_kit`, `media_kit_video`, and `media_kit_libs_video` on Android, iOS, macOS, Windows, Linux, and web. The native player shall open with autoplay disabled; provide play/pause, seek, mute/volume as supported by the platform, elapsed/duration state, replay, and full-screen controls; and pause when its post is no longer visible or the app is backgrounded. | Controls data use, audio surprise, accessibility, resource consumption, and package coupling across the repository's declared targets. | Package research, accessibility and UX constraints | AC-017, AC-018 |
| FR-016 | Functional | Must | If the HLS playlist is absent or cannot play, Flutter shall show a recoverable media error while leaving post text and actions usable and shall not attempt raw PDS MP4 playback. | Fails soft while preserving the confirmed HLS-only scope. | Requirements grilling | AC-011, AC-019 |
| FR-017 | Functional | Must | Flutter shall expose provided video alt text through accessibility semantics and visible descriptive context. | Preserves standard embed accessibility metadata when the author supplies it. | `app.bsky.embed.video`, requirements grilling | AC-020, AC-041 |
| FR-018 | Functional | Must | AppView shall expose authenticated `GET /v1/posts/{did}/{rkey}/video-captions/{captionCid}`, verify that the exact bounded `text/vtt` blob belongs to that indexed post's video embed, and proxy it without exposing a general PDS/blob fetcher; Flutter shall supply each resulting URI to `media_kit` as a selectable `SubtitleTrack.uri`. Caption upload remains out of scope. | Avoids direct PDS reads and preserves interoperable captions through a constrained route. | `app.bsky.embed.video`, `media_kit` documentation, requirements grilling | AC-021 |
| FR-019 | Functional | Must | The shared composer attachment widget shall accept an explicit boolean video capability such as `supportsVideo`; video-capable post composers shall enable it, while business products, business events, and all other image-only consumers shall disable it. | The shared widget serves surfaces with different media contracts and must not infer capability from context. | User feedback, codebase | AC-027, AC-028 |
| FR-020 | Functional | Must | When video capability is enabled and the attachment widget is empty, it shall use localized copy that identifies support for photos and video, and activation shall ask whether to add photo(s) or one video before opening the corresponding picker; after selection, controls and copy shall be specific to the active media type. | Makes available media discoverable without repeatedly offering invalid mixed-media choices. | User feedback, requirements grilling | AC-027, AC-044 |
| FR-021 | Functional | Must | When video capability is disabled, the attachment widget shall retain photo-only copy and behavior and shall expose no video choice or video picker path. | Prevents unsupported video selection on business and other image-only records. | User feedback | AC-028 |
| FR-022 | Functional | Must | After publish is chosen, the blocking publishing screen shall show accessible stages for video upload, video processing, and post publication; upload and processing shall show percentage when available and indeterminate progress otherwise, without exposing raw service-state names. | Long-running video work needs understandable feedback at the point where remote work begins. | User feedback, requirements grilling | AC-001, AC-004, AC-029 |
| FR-023 | Functional | Must | If publish-time upload or processing fails, the publishing screen shall identify the actionable failure, return to an editable composer with the local video intact when safe, and offer a user-directed Retry that restarts the complete validate/auth/upload/process/publish sequence. | Preserves work and establishes simple deterministic retry behavior. | User feedback, requirements grilling | AC-005, AC-030, AC-035 |
| FR-024 | Functional | Must | Standard and project local drafts shall persist the selected source video and a derived local poster frame in account-scoped storage without uploading either asset. | Preserves video work while honoring publish-time upload. | Requirements grilling, codebase | AC-031, AC-032 |
| FR-025 | Functional | Must | Saved source-video bytes shall have a decimal 1,000,000,000-byte quota per account; a save that would exceed it shall fail with actionable cleanup guidance and shall not evict, alter, or partially save existing drafts. | Prevents unbounded local storage without destructive eviction. | Requirements grilling | AC-031 |
| FR-026 | Functional | Must | The publishing screen shall allow cancellation without an orphan-blob warning during upload and processing, preserve the local video, stop local upload/polling where possible, and disable cancellation once final post creation begins. | Gives users control during long work while avoiding ambiguous PDS-write outcomes. | Requirements grilling | AC-033 |
| FR-027 | Functional | Must | If Flutter is backgrounded, terminated, or otherwise interrupted during video publication, it shall not automatically resume the prior job or silently publish; reopening the retained draft and choosing publish starts a new sequence. | Matches the confirmed restart-from-scratch behavior. | Requirements grilling | AC-034 |
| FR-028 | Functional | Must | AppView shall expose authenticated `GET /v1/blobs/videos/limits`, requiring the Craftsky bearer session and device ID, to preflight account/provider eligibility before Flutter opens the video picker. AppView shall obtain the separate `app.bsky.video.getUploadLimits` service authorization and keep it server-side. The camelCase response contains `canUpload`, optional `remainingDailyVideos`, optional `remainingDailyBytes`, and optional normalized `reason` (`email_unverified`, `quota_exhausted`, `provider_unsupported`, or `unknown`); upstream availability failures use the standard error envelope. Flutter shall keep an ineligible Video option visible with a specific localized explanation and treat the publish-time recheck/service response as final. Quota is constrained when upload is disallowed, one or fewer daily videos remain, or fewer than 300,000,000 daily bytes remain; only then shall remaining quota be shown. | Prevents late avoidable failures without violating the client token boundary or cluttering the normal path. | Requirements grilling, current video API, API architecture | AC-036, AC-037 |
| FR-029 | Functional | Must | The service JWT, raw authorization header, and remote job continuation state shall remain memory-only in Flutter and shall be cleared on terminal outcome, cancellation, account change, or app interruption; AppView shall not persist video jobs or service JWTs. | Limits credential exposure and prevents silent resume. | Architecture amendment, user decision | AC-038 |
| FR-030 | Functional | Must | Flutter shall poll job status beginning at approximately one-second intervals, back off to no more frequently than every five seconds, honor `Retry-After`, and stop on terminal state, user cancellation, app interruption, or active-job expiry. | Balances responsive progress with downstream load. | Requirements grilling | AC-039 |
| FR-031 | Functional | Must | An ambiguous upload failure or explicit Retry shall obtain a fresh service JWT and restart upload/processing; Craftsky shall not invent an unsupported idempotency key, and may reuse a BlobRef only when `already_exists` returns it and AppView verifies it. | Matches the documented video-service contract and avoids false deduplication guarantees. | Bluesky docs, user amendment | AC-035, AC-040 |
| FR-032 | Functional | Must | The composer shall prompt for optional video alt text, allow an explicit skip, and reject provided alt text over 1,000 graphemes. | Balances accessibility encouragement with the confirmed optional policy and a bounded Craftsky input. | Requirements grilling | AC-041 |
| NFR-001 | Non-functional | Must | All new `/v1/*` endpoints and failures shall follow existing Craftsky authentication, device-ID, camelCase JSON, request-ID, and error-envelope conventions. | Keeps the API coherent and testable. | API architecture spec | AC-001, AC-005 |
| NFR-002 | Non-functional | Must | Direct upload, polling, AppView authorization, and final verification shall use bounded timeouts, cancellation, response limits, and retry classification so disconnects or service outages leave no unbounded operation. | Protects device and service reliability. | Risk analysis | AC-014 |
| NFR-003 | Non-functional | Must | Flutter/AppView logs, telemetry, crash reports, URLs, and persistent storage shall not contain the service JWT, OAuth credentials, DPoP material, authorization headers, video bytes, or authored alt/caption text. | Direct token handoff creates a new client-side secret boundary. | Security constraint | AC-002, AC-022, AC-038 |
| NFR-004 | Non-functional | Should | Playback UI should maintain the declared aspect ratio when present, use a stable fallback layout otherwise, and avoid disruptive feed reflow when player state changes. | Prevents unstable media-heavy scrolling. | Existing media pattern | AC-023 |
| NFR-005 | Non-functional | Must | Local draft save, verification, reopen, and publish shall process source videos without requiring an additional complete in-memory copy of a file that may be 300,000,000 bytes. | The existing image-oriented byte-buffering repository pattern is unsafe for maximum-size video drafts. | Codebase, requirements grilling | AC-031, AC-043 |
| RULE-001 | Business rule | Must | A source video may be no longer than 600 seconds and no larger than 300,000,000 bytes. | Captures the user-provided limits and current standard lexicon size. | Prompt, current `app.bsky.embed.video` | AC-003, AC-015 |
| RULE-002 | Business rule | Must | This slice accepts only `video/mp4`; extension alone is not sufficient when MIME/content validation reports another type. | Matches the standard embed contract. | Current `app.bsky.embed.video` | AC-003 |
| RULE-003 | Business rule | Must | A post may contain at most one native video, and native video is mutually exclusive with top-level images, quote embeds, and external cards; video quote-posts are unsupported. | Prevents ambiguous rendering and follows the confirmed direct standard embed shape. | Requirements grilling | AC-007, AC-024 |
| RULE-004 | Business rule | Must | Native video composition is limited to top-level standard and project posts. Their local drafts may preserve video, but replies and scheduled posts do not support video. | Locks the confirmed surface boundary. | Requirements grilling | AC-012, AC-016, AC-031 |
| RULE-005 | Business rule | Must | After the user chooses publish, the post record shall not be submitted until the video service has returned a valid blob reference for that user's job. | Prevents publication of missing/unprocessed media while preserving publish-time upload semantics. | Bluesky recommended flow, user feedback | AC-004, AC-007, AC-026 |
| RULE-006 | Business rule | Must | Video-service eligibility, email-verification, daily quota, and provider-compatibility failures shall block video publication and be presented as distinct actionable states rather than generic post failures. | Users need to understand account/service policy failures. | Bluesky docs, selected dependency | AC-005 |
| RULE-007 | Business rule | Must | Uploaded and posted video is public PDS media under the current atproto model; the UI must not imply that it is private. | Prevents incorrect privacy expectations. | Architecture rules | AC-025 |
| RULE-008 | Business rule | Must | Photo and native-video draft selections are mutually exclusive: the widget shall not silently delete existing media, and the user must remove the current media before selecting the other type. | Enforces the post media rule without destructive or surprising picker behavior. | User feedback, RULE-003 | AC-024, AC-027 |
| RULE-009 | Business rule | Must | Craftsky shall select existing videos through the platform gallery/file picker only and shall not offer camera recording in this release. | Keeps capture permissions and recording lifecycle out of scope. | Requirements grilling | AC-042 |
| RULE-010 | Business rule | Must | Video alt text is optional for Craftsky-authored posts, but when supplied it may not exceed 1,000 graphemes. | Captures the confirmed accessibility policy. | Requirements grilling | AC-041 |
| RULE-011 | Business rule | Must | A metered or cellular connection shall not block video publication or introduce an additional data-usage confirmation in this release. | Captures the confirmed no-warning network policy. | Requirements grilling | AC-045 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-012, FR-022, NFR-001 | Given an eligible signed-in user with a valid device ID has selected a valid local MP4, when they choose publish, then Flutter obtains upload authorization from AppView, streams the MP4 directly to `video.bsky.app`, polls processing directly, and shows general stage feedback before AppView creates the PDS post. |
| AC-002 | BR-003, FR-002, NFR-003 | Given any successful or failed authorization/upload flow, when responses, memory, storage, requests, redirects, logs, and crash diagnostics are inspected, then Flutter receives only a service JWT bound to the user's PDS audience and `com.atproto.repo.uploadBlob`, expiring no more than 30 minutes after issuance; it receives no OAuth access/refresh token or DPoP key, persists no service JWT, and sends it only to the exact approved upload URL. |
| AC-003 | FR-001, FR-003, FR-013, RULE-001, RULE-002 | Given a non-MP4 upload or a payload over 300,000,000 bytes, when validated, then it is rejected before successful video-service upload; and given a source known to exceed 600 seconds, then publication is blocked with actionable validation feedback. |
| AC-004 | FR-004, FR-005, FR-022, RULE-005 | Given a publish-time video job, when Flutter polls the public video service, then normalized progress advances locally and post creation remains blocked until a valid job ID and BlobRef are returned and independently verified by AppView. |
| AC-005 | FR-004, FR-023, NFR-001, RULE-006 | Given the video service reports ineligibility, unverified email, quota exhaustion, incompatibility, validation failure, processing failure, or a retryable outage, when Flutter normalizes the result, then it presents the applicable action or retry guidance, preserves local selection when safe, and avoids creating the post. |
| AC-006 | FR-006 | Given the video service reports `already_exists` and includes a blob, when Flutter handles the result and AppView verifies it, then the job completes successfully rather than being shown as failed. |
| AC-007 | BR-001, FR-005, FR-007, RULE-003, RULE-005 | Given valid post text and submitted job/blob data, when the user publishes, then AppView verifies completion, authenticated DID ownership, and exact BlobRef equality before writing one `app.bsky.embed.video`; incomplete, foreign, mismatched, or mixed-media input is rejected atomically. |
| AC-008 | FR-008 | Given an existing valid Craftsky post without video and a new valid post with an `app.bsky.embed.video` variant, when both are validated against the updated lexicon, then both remain valid and the video field is not required on the old post. |
| AC-009 | BR-002, FR-009, FR-010 | Given a supported video embed arrives through Tap, including one authored outside Craftsky, when it is indexed and read through any canonical post endpoint, then its response includes one normalized video object preserving available CID, MIME, size, alt text, and aspect ratio plus playback metadata. |
| AC-010 | BR-002, FR-010, FR-011, FR-014 | Given a post response containing native video, when AppView derives playback metadata and a standard post surface displays it, then the playlist and optional thumbnail use the configured URL templates with percent-encoded author DID and CID, and Flutter initializes `media_kit` from that metadata without reading the post record or raw blob directly from the PDS. |
| AC-011 | FR-011, FR-016 | Given the HLS playlist is absent or fails, when playback is attempted, then Flutter shows a recoverable media error, leaves the rest of the post usable, and makes no raw PDS MP4 fallback request. |
| AC-012 | FR-012, RULE-004 | Given a top-level standard or project composer, when a user selects a video, then one local video preview is available with remove or replacement behavior and no network upload/job begins; reply and scheduled-post flows do not gain video attachment behavior. |
| AC-013 | FR-005 | Given Alice submits Bob's job ID or a BlobRef different from the verified service result, when AppView verifies publication, then it discloses no foreign job detail and writes no post. |
| AC-014 | FR-003, NFR-002 | Given a maximum-sized direct upload, disconnect, cancellation, timeout, redirect, or video-service outage, when Flutter handles it, then memory remains bounded, work stops where possible, the service JWT is not forwarded to another origin/path, and the outcome is classified without an unbounded operation. |
| AC-015 | FR-013, RULE-001 | Given videos at 600 seconds and 300,000,000 bytes, when otherwise valid, then they may proceed; given a video above either limit, then it cannot be published, with the video service acting as final authority when local duration metadata is unavailable or inaccurate. |
| AC-016 | BR-004, RULE-004 | Given existing text, image, quote, external-card, reply, scheduled-post, or YouTube flows, when native video support is released, then those flows retain their prior behavior except that top-level standard/project composers and their local drafts support video. |
| AC-017 | FR-015 | Given a native video post first enters the viewport on Android, iOS, macOS, Windows, Linux, or web, when the user has not activated playback, then `media_kit` opens it without autoplay; when activated, the user can play/pause, seek, mute or adjust volume where supported, see time state, replay, and use full screen. |
| AC-018 | FR-015 | Given a playing video, when its post leaves the active viewport or the app enters the background, then playback pauses and does not continue unexpected audio. |
| AC-019 | FR-016 | Given HLS playback fails, when the post renders, then the player shows a recoverable media error while post text, author, navigation, and actions remain usable and no raw PDS fallback is attempted. |
| AC-020 | FR-017 | Given a video embed with alt text, when the post/player is reached by assistive technology, then the description is exposed through semantics and is available as descriptive context. |
| AC-021 | FR-018 | Given an indexed video embed with a valid supported WebVTT caption source, when playback begins, then Flutter supplies it as a selectable `media_kit` URI subtitle track and renders selected cues; absence of captions does not block playback. |
| AC-022 | NFR-003 | Given authorization, direct upload, processing, verification, indexing, or playback failures, when logs, telemetry, and crash reports are captured on client and server, then they contain safe bounded state but omit credentials, authorization headers, bytes, identifiers in URLs, and authored accessibility text. |
| AC-023 | NFR-004 | Given video with or without declared aspect ratio and thumbnail metadata, when its card changes between idle, loading, playing, and error states, then it retains a stable bounded layout without disruptive feed reflow. |
| AC-024 | RULE-003, RULE-008 | Given a create-post request containing native video plus images, quote, external card, or a second video, when validated, then the entire request is rejected with field-level validation feedback and no post is written; composer UI does not silently replace one selected media type with another. |
| AC-025 | RULE-007 | Given video composition or explanatory UI is shown, when media storage/privacy is described, then the UI does not state or imply that uploaded video is private. |
| AC-026 | FR-012, RULE-005 | Given a valid local video selection, when the user continues editing, leaves the publish action untouched, or closes the composer, then no video bytes are sent to AppView or the video service and no remote video job is created; only choosing publish starts remote work. |
| AC-027 | FR-019, FR-020, RULE-008 | Given a standard or project post attachment widget with video capability enabled and no media selected, when it renders and is activated, then localized copy says that photos and video are supported and the user is asked to choose photo(s) or video; once one media type is selected, the widget becomes type-specific and changing type requires explicit removal rather than silently discarding media. |
| AC-028 | FR-019, FR-021 | Given a business product, business event, or another attachment widget with video capability disabled, when it renders and is activated, then its copy and picker remain photo-only and no video option or picker path is exposed. |
| AC-029 | FR-022 | Given a user publishes a local video draft, when submission advances, then the blocking publishing screen announces and displays upload, processing, and final publication stages, shows upload/processing percentage when available, uses indeterminate progress otherwise, and does not expose raw service-state constants. |
| AC-030 | FR-023 | Given publish-time upload or processing fails, when the publishing flow exits its blocking state, then no post is created, the composer remains editable with its local video retained when safe, the failure is actionable, and pressing Retry starts the complete sequence again. |
| AC-031 | FR-024, FR-025, NFR-005, RULE-004 | Given standard and project drafts with selected videos, when saved within the account's 1,000,000,000-byte source-video quota, then each video remains local and can be reopened; when a save would exceed the quota, then it is blocked with cleanup guidance and no existing or partial draft data is removed. |
| AC-032 | FR-024 | Given a saved video draft, when the drafts list renders it, then a stored small local poster frame identifies the video without initializing live video playback. |
| AC-033 | FR-026 | Given video upload or processing is active, when the user cancels, then local upload/polling stops where possible and the editable composer returns with the local video intact and no orphan warning; once final post creation starts, cancel is unavailable. |
| AC-034 | FR-027 | Given the app is interrupted during upload or processing, when the user later reopens the retained draft, then Craftsky does not resume or silently publish the prior job; choosing publish starts from validation with a new attempt. |
| AC-035 | FR-023, FR-031 | Given a terminal or ambiguous publish failure, when the user presses Retry, then Craftsky obtains fresh authorization and restarts validation, upload, processing, and publication rather than resuming the failed job. |
| AC-036 | FR-028 | Given the empty video-capable media chooser, when the user selects Video, then Flutter calls authenticated `GET /v1/blobs/videos/limits` and waits for an eligible result before opening the picker; an ineligible result does not open the picker, leaves the choice visible, and presents its normalized localized explanation, while an upstream failure uses standard retryable error behavior and eligibility is checked again at publish. |
| AC-037 | FR-028 | Given upload limits are returned, when upload is disallowed, one or fewer daily videos remain, or fewer than 300,000,000 daily bytes remain, then the chooser shows the relevant allowance; otherwise routine quota detail is omitted. |
| AC-038 | FR-029 | Given authorization or processing begins, when the attempt completes, fails, is canceled, the account changes, or the app is interrupted, then service JWT/header state and remote continuation state are absent from persistent storage and cleared from live state; AppView has no private video-job row to clean. |
| AC-039 | FR-030 | Given a processing job, when Flutter polls, then requests begin around one second apart, back off to at most one request per five seconds, honor a longer `Retry-After`, and stop on completion, failure, cancel, app interruption, or one-hour expiry. |
| AC-040 | FR-031 | Given an upload response is ambiguous, when publication retries, then Flutter requests fresh authorization and restarts the upload; if the service returns `already_exists` with a blob, AppView still independently verifies the owner and exact blob before publication. |
| AC-041 | FR-017, FR-032, RULE-010 | Given a selected video, when the composer prompts for alt text, then the user may provide up to 1,000 graphemes or explicitly skip it; over-limit text blocks publish, and supplied text is exposed in playback semantics. |
| AC-042 | FR-012, RULE-009 | Given the user chooses Add video, when selection begins, then Craftsky opens a picker for existing gallery/files and does not offer camera recording. |
| AC-043 | NFR-005 | Given a maximum-size saved video draft, when it is saved, verified, reopened, or prepared for publication, then Craftsky does not require an additional complete 300,000,000-byte in-memory copy. |
| AC-044 | FR-020 | Given photos are selected, when the attachment control renders, then it offers photo-specific add/reorder/remove behavior; given a video is selected, then it offers video-specific replace/remove behavior and does not show the empty photo/video chooser. |
| AC-045 | RULE-011 | Given a metered or cellular connection, when an otherwise valid video is published, then Craftsky permits the attempt without an extra network warning or confirmation. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User removes the video or closes the composer before publishing | Removal discards the local selection. On close, the user may save it as a local draft; otherwise it is discarded. No upload, remote job, blob, or post is created. | FR-012, FR-024 |
| EC-002 | Client briefly loses connectivity after job creation without app interruption | Tokenless polling may resume while the ephemeral attempt is alive; an ambiguous upload or explicit Retry obtains fresh authorization and restarts upload. | FR-030, FR-031 |
| EC-003 | Video service returns failure plus a blob | Presence of a valid blob wins for the documented `already_exists` case; other contradictory responses fail safely and are logged without secrets. | FR-006, NFR-003 |
| EC-004 | Local duration metadata is absent or inaccurate | Local validation may pass, but the service result remains authoritative and can reject a video above 600 seconds. | FR-013, RULE-001 |
| EC-005 | PDS moved after the post was indexed | HLS identification remains based on stable author DID and video CID rather than handle; this release does not construct a raw PDS playback source. | FR-011 |
| EC-006 | Playlist is available but thumbnail is absent | The player uses a stable neutral placeholder and remains playable. | FR-010, NFR-004 |
| EC-007 | HLS playlist is absent, expired, or unsupported by the platform | The player reports a recoverable failure, makes no raw PDS request, and leaves the rest of the post usable. | FR-016 |
| EC-008 | Unsupported or malformed video embed arrives through Tap | The indexer fails safely according to existing ingestion rules and does not emit a malformed playable response. | FR-009, NFR-002 |
| EC-009 | Multiple video cards are visible | None autoplay; only user-activated playback runs, and starting or scrolling away from playback does not leave unexpected competing audio. | FR-015 |
| EC-010 | Video service rejects an otherwise lexicon-valid MP4 due to account policy | Composer presents the normalized eligibility/quota/provider state and does not submit the post. | RULE-006 |
| EC-011 | Existing standard embed contains captions although Craftsky cannot upload them | Read/playback may expose valid captions; caption absence or failure does not invalidate the video. | FR-018 |
| EC-012 | Upload/processing succeeds but final post creation fails | The publishing screen reports the post failure and returns to the editable composer when safe. The processed unreferenced blob follows normal PDS lifecycle behavior; this slice does not delete it directly. | FR-023, RULE-005, ASM-004 |
| EC-013 | User has photos selected and asks to add video, or has video selected and asks to add photos | The widget explains the mutual-exclusion rule and requires explicit removal of current media; it does not silently clear the draft. | RULE-003, RULE-008 |
| EC-014 | Saving a video draft would exceed the account quota or device storage is unavailable | The save fails atomically with actionable guidance; existing drafts and source media remain untouched. | FR-025, NFR-005 |
| EC-015 | App is terminated after remote processing starts | The remote service may finish, but Craftsky does not resume that job or publish automatically. The retained local draft starts a new attempt only after the user publishes again. | FR-027, FR-031 |
| EC-016 | User cancels processing immediately before completion | Craftsky returns to the composer without a warning. Any remote completion remains unreferenced unless a later full retry receives `already_exists`. | FR-006, FR-026 |

## 15. Data / Persistence Impact
- New fields:
  - Add `app.bsky.embed.video` as an optional variant in the Craftsky post
    lexicon's open `embed` union.
  - Add an optional canonical AppView `video` response object containing `cid`,
    `mime`, `size`, `alt`, `aspectRatio`, `playlist`, `thumbnail`, and safe
    caption metadata when available. Do not expose a raw PDS source in this
    release.
  - Add Flutter post/video, local selected-video, local poster, draft media, and
    ephemeral authorization/job-state models.
- Changed fields:
  - Extend `POST /v1/posts` wire input with verified `embed.video` job/blob data.
  - Extend media mutual-exclusion validation to include video.
- Migration required:
  - No AppView video-job migration; direct client polling removes private job
    persistence.
  - No dedicated post-video column is required by the requirements because the
    existing raw post record/embed projection can remain authoritative, though
    coding design may choose an additive projection for query/read efficiency.
  - The local draft manifest/media schema requires an additive video media kind,
    poster reference, and quota accounting; this is local persistence rather
    than a Postgres migration.
- Backwards compatibility:
  - The lexicon change is additive because `embed` is open and optional.
  - Existing posts and clients that ignore the new response field remain valid.
  - Existing quote and external variants retain their current record shapes.

## 16. UI / API / CLI Impact
- UI:
  - The shared attachment widget gains an explicit video capability flag,
    capability-aware localized copy, and a photo/video choice on enabled
    surfaces. Business products, business events, and other disabled consumers
    remain photo-only.
  - Immediate top-level composer gains local single-video selection, preview,
    optional/skippable alt text, removal/replacement, eligibility preflight, and
    submit validation without pre-publish upload.
  - Standard and project drafts persist source video plus a small local poster;
    over-quota saves direct the user to remove media or delete drafts.
  - The blocking publishing screen gains general upload, processing, and final
    publication feedback, percentage where available, cancel during upload or
    processing, and full-sequence retry behavior.
  - Standard post surfaces gain an inline non-autoplaying `media_kit` player,
    thumbnail/placeholder, controls, full-screen HLS playback, captions, and a
    soft error state without raw PDS fallback.
  - Existing external YouTube UI remains separate.
- API:
  - Add authenticated `POST /v1/blobs/videos/authorization`, returning only the
    ephemeral service JWT and expiry.
  - Add authenticated `GET /v1/blobs/videos/limits` returning `canUpload`,
    optional remaining allowances, and an optional normalized reason token.
  - Extend `POST /v1/posts` with `embed.video` job/blob proof.
  - Add optional `video` to canonical post-shaped responses.
  - Add authenticated
    `GET /v1/posts/{did}/{rkey}/video-captions/{captionCid}` for bounded indexed
    WebVTT captions.
- CLI:
  - None identified.
- Background jobs:
  - None in AppView. Video processing and job state remain in the selected video
    service; Flutter polls directly only while the publication attempt is alive.

## 17. Security / Privacy / Permissions
- Authentication:
  - Authorization and limits routes require the existing Craftsky bearer session
    and device ID.
  - AppView obtains service auth through its server-held OAuth session and hands
    Flutter only the short-lived upload service JWT.
  - Limits use a separate method-bound service authorization kept inside
    AppView; caption delivery uses the normal Craftsky session and device ID.
- Authorization:
  - The service JWT is bound to the user's PDS audience and uploadBlob method.
  - AppView independently verifies job DID, completion, and exact blob before
    creating the post.
- Sensitive data:
  - OAuth tokens and DPoP keys remain server-side. The service JWT is client-side
    memory only and excluded from storage, logs, telemetry, and crash reports.
  - Published blobs and embeds remain public PDS data.
- Abuse cases:
  - Oversized/malformed media, malicious redirects, token replay, broad
    interceptors, repeated authorization/polling, guessed job IDs, quota evasion,
    and attempts to post another account's or a mismatched blob/job.

## 18. Observability
- Events:
  - None required as a product analytics event in this slice.
- Logs:
  - Safe authorization/upload/job transition, normalized terminal reason,
    request ID, byte-count band, and latency.
  - Playback source-resolution failures without logging full sensitive query
    data or user-authored text.
- Metrics:
  - Release instrumentation must cover authorization and direct-upload attempts,
    accepted/rejected byte bands, jobs by normalized terminal state, processing
    duration, verification failures, eligibility/quota failures, status polling
    rate, publish cancellation/retry, and HLS playback failure
    where observable. Metric labels must use bounded state/reason categories and
    exclude DIDs, job IDs, CIDs, URLs, filenames, and authored text.
- Alerts:
  - No new deployed alert configuration is required while the app has no
    production environment. Before production launch, alert policy should cover
    sustained video-service failure, abnormal upload rejection/resource
    pressure, and jobs stuck beyond a defined processing threshold.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | The selected Bluesky service may reject non-Bluesky PDSes/accounts or change policy | Some Craftsky users cannot upload despite having a capable PDS | Surface compatibility as a distinct state, document the dependency, and keep direct-upload fallback explicitly out of this slice rather than silently changing behavior |
| RISK-002 | Direct upload exposes a replayable service JWT to the untrusted client | A leak can authorize blob upload until expiry | Keep it memory-only, short-lived, purpose-bound, exact-origin/path constrained, redirect-safe, and excluded from diagnostics/storage |
| RISK-003 | Service-auth scope/audience or token handoff is implemented incorrectly | Upload failure, broader authority, or credential exposure | Keep OAuth/DPoP server-only; validate service JWT claims/expiry; cover the response and external transport with secret-scan tests |
| RISK-004 | Bluesky may change its HLS/CDN URL patterns | Generated playlists or thumbnails may break and there is no raw PDS fallback in this release | Use configurable percent-encoding URL templates with current Bluesky defaults, contract-test them, fail soft, and measure HLS failures |
| RISK-005 | `media_kit` HLS, WebVTT, full-screen, or bundled native-library behavior may vary across Flutter targets and browsers | Some supported app targets may be unable to play otherwise valid videos or may increase binary size | Keep an app-owned abstraction, initialize/dispose explicitly, run the six-target smoke matrix, verify web assets/CORS, measure release binary impact, and preserve usable post content on failure |
| RISK-006 | Adding video to the embed union changes media coexistence behavior | Clients may create or render ambiguous mixed-media records | Require an ADR, explicit mutual-exclusion validation, generated types, and end-to-end record tests |
| RISK-007 | Player lifecycle in scrolling lists leaks decoders, bandwidth, or audio | Feed performance and user trust degrade | No autoplay, pause offscreen/background, dispose controllers, and test multi-card scrolling |
| RISK-008 | Duration metadata can be absent or forged client-side | Oversized-duration uploads consume resources before rejection | Validate locally where possible and rely on video-service media validation as final authority |
| RISK-009 | AppView trusts client-supplied job/blob data | Foreign or unprocessed media could enter a post | Independently query job status and require authenticated DID and exact BlobRef equality before the PDS write |
| RISK-010 | A capability-aware shared attachment widget regresses image-only business surfaces | Unsupported video may be offered or existing product/event image UX may change | Require an explicit flag, default/usage audit, capability-specific copy, and widget regressions for every current consumer |
| RISK-011 | Publish-time upload makes submission substantially longer and failures occur after the user commits to publish | Users may think the app is stuck or lose composer work | Show accessible stage feedback, preserve local selection on recoverable failure, and require user-directed retry |
| RISK-012 | Saved video drafts can consume substantial local storage | Drafts may exhaust device storage or make file operations slow | Enforce a 1,000,000,000-byte account quota, atomic saves, no automatic eviction, cleanup guidance, and streaming file operations |
| RISK-013 | Cancellation or app interruption can leave remote processed blobs unreferenced | User PDS storage may contain media not referenced by a post | Do not delete PDS blobs, rely on normal PDS lifecycle, use `already_exists` on later retries, and avoid misleading warnings per the confirmed UX decision |
| RISK-014 | Browser CORS or platform redirect behavior prevents or leaks direct upload | Web publication fails or bearer scope escapes | Verify CORS on web, use an external client with redirects disabled, and require exact HTTPS origin/path before attaching authorization |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The required lexicon ADR will ratify the confirmed use of `app.bsky.embed.video` in `social.craftsky.feed.post`. | If the ADR uncovers a protocol incompatibility, the approved record/API/indexing design must be revisited before implementation. |
| ASM-002 | “DPS” in the initial wording means “PDS.” | If a different storage/service concept was intended, the playback and ownership requirements are mis-scoped. |
| ASM-003 | The current Bluesky service-auth and video job APIs permit AppView to acquire the service JWT and Flutter to upload/poll directly for compatible account providers. | The selected architecture may require a provider capability gate or reconsideration. |
| ASM-004 | Unreferenced processed blobs may follow normal PDS lifecycle behavior; Craftsky will not directly delete abandoned blobs in this slice. | A stricter storage policy would require a separate authorized cleanup design that respects the PDS deletion rules. |
| ASM-005 | Text remains required under the existing Craftsky post contract when video is attached. | Supporting video-only posts would require a separate change to current create-post validation/product behavior. |
| ASM-006 | `media_kit` can play the returned HLS source and URI WebVTT captions on Android, iOS, macOS, Windows, Linux, and supported web browsers. | Platform-specific behavior or a documented reduction in target support may be required because raw MP4 fallback is out of scope. |
| ASM-007 | Standard and project top-level posts and their local drafts can use native video under the same rules. | If project records impose a conflicting media rule, composer and validation scope must be narrowed. |
| ASM-008 | Public `getJobStatus` remains available long enough for AppView to verify the submitted job and BlobRef at publication. | A server-verifiable completion receipt or narrow AppView job registration would be required. |

## 21. Open Questions
- [ ] Non-blocking for coding design: How will the draft file repository stream,
  hash, and atomically verify up to 300,000,000 bytes without its current
  whole-file `Uint8List` behavior?

## 22. Review Status
Status: Approved with notes for coding planning

Risk level: High

Review completed: `03-document-review.md`

Reviewer: OpenCode

Date: 2026-09-03

Notes: Explicit approval remains required before implementation. The amended
change hands Flutter a narrow PDS-signed bearer, performs a direct 300 MB
third-party upload, independently verifies completion server-side, changes a
project-wide credential rule, adds a load-bearing lexicon variant, and introduces
HLS playback plus large local drafts. The 2026-09-03 amendment replaces the
earlier AppView upload proxy/private-job design while retaining the limits route,
playback templates, and `media_kit` platform choice.

## 23. Handoff To Test Design
- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-004`, `FR-001` through
  `FR-032`, `NFR-001` through `NFR-003`, `NFR-005`, and `RULE-001` through
  `RULE-011`.
- Suggested test levels:
  - Lexicon compatibility and generated-type tests.
  - AppView authorization/limits routes, exact service-JWT claim tests, and
    OAuth/DPoP non-disclosure tests.
  - Flutter direct-upload exact-origin/path, redirect, CORS, streaming-memory,
    cancellation, token clearing, polling backoff, and `already_exists` tests.
  - AppView final job-DID/blob verification and mismatch/non-disclosure tests.
  - Create/index/read integration tests for standard video embeds authored both
    through Craftsky and externally.
  - HLS playlist/thumbnail resolution and explicit no-raw-PDS-fallback tests.
  - Flutter API/model/provider tests proving selection is local until publish,
    then covering eligibility preflight/recheck, constrained quota copy, job
    transitions, cancellation boundary, interruption restart, full-sequence
    retry, retained selection, optional alt text, and submit gating.
  - Local draft tests for standard/project source video and poster persistence,
    1,000,000,000-byte per-account quota, atomic over-quota failure, cleanup,
    and bounded-memory file handling.
  - Shared attachment-widget tests for capability-aware copy and picker choice
    on post/project surfaces plus photo-only business product/event regressions.
  - Publishing-screen tests for accessible stages, determinate/indeterminate
    progress, cancellation, publication lock-in, and failure feedback.
  - Flutter widget/integration tests for all player states, controls,
    accessibility, HLS failure, scrolling lifecycle, background pause, stable
    layout, and multiple visible cards.
  - Regression tests for text, image, quote, external-card, project, reply,
    draft, scheduled-post, and YouTube flows.
- Blocking open questions: None for test specification or coding planning.
  High-risk review approval remains the stage gate before implementation.
