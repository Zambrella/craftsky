# Coding Plan: Video Posts

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`, Approved with notes
- Credential decision: `adr/012-ephemeral-video-service-token-handoff.md`
- Lexicon decision: `adr/013-standard-video-embed.md`
- Risk: High
- Implementation status: Not started
- Approval gate: Explicit user approval is required before the first TDD change.

## 2. Implementation Strategy

Implement video as an additive standard post embed while retaining the existing
AppView BFF and Tap-indexing architecture:

1. Add `app.bsky.embed.video` to the open CraftSky post embed union and regenerate
   Go types from explicit Indigo schemas.
2. AppView exposes authenticated limits and authorization endpoints. OAuth and
   DPoP credentials remain inside the coordinated server session.
3. Flutter uses an isolated HTTP client to stream MP4 directly to the configured
   Bluesky upload endpoint with the ephemeral service JWT, then polls public job
   status without authorization.
4. Flutter submits `jobId`, `BlobRef`, alt text, and aspect ratio to
   `POST /v1/posts`. AppView independently verifies status, owner DID, and exact
   blob before constructing any PDS effect.
5. Tap remains authoritative for public indexing. Existing full-record storage
   supplies video metadata to canonical response hydration; no video columns or
   private job table are added.
6. Flutter renders canonical HLS through an app-owned player abstraction and
   obtains standard WebVTT captions through one constrained authenticated
   AppView route.

The former AppView upload/status proxy, `video_jobs` migration, idempotency key,
cleanup worker, and restart continuation are prohibited.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Public record | Optional open post embed union and generated Go types | Add standard video variant and explicit external schemas | FR-007, FR-008, RULE-003 | UT-003, IT-006, REG-007 |
| OAuth capability | Narrow operations run through the active-session coordinator | Add purpose-specific upload and limits service-auth operations | BR-003, FR-001, FR-002, FR-028 | IT-002, IT-013, IT-017 |
| Video service | Hardened purpose-specific outbound clients | Add limits, public job status, and completion verification; no body proxy | FR-004 through FR-006, NFR-002 | UT-004, IT-003 through IT-005, IT-012 |
| Post creation | Strict request decode/validation then PDS effect | Accept proof, verify it, and author `app.bsky.embed.video` | FR-005 through FR-007, RULE-001 through RULE-004 | UT-003, IT-003, IT-005 |
| Index/read | Full record JSON retained and shared post response shape | Validate standard embeds and hydrate HLS/caption metadata | FR-009 through FR-011, FR-018 | UT-005, IT-007, IT-008, IT-019 |
| Flutter transport | Authenticated Dio is scoped to AppView | Add a separate no-interceptor direct service client | FR-001, FR-003, FR-004, FR-029 through FR-031 | UT-001, UT-011 through UT-013, IT-001, IT-009, IT-010, IT-018 |
| Composer | Riverpod state, image attachment section, submission coordinator | Add capability-gated local video and staged publication | FR-012 through FR-014, FR-019 through FR-023, FR-026, FR-032 | UT-008, UT-009, UT-014, AT-001 through AT-005, AT-007, AT-012 |
| Drafts | Account-scoped revisioned file store and manifests | Add streamed source plus poster with source-byte quota | FR-024, FR-025, NFR-005 | UT-007, UT-015, IT-011, AT-006 |
| Playback | Post card media branch and lifecycle-aware widgets | Add app-owned `media_kit` HLS player and captions | FR-014 through FR-018, NFR-004 | UT-010, UT-016, IT-014, IT-015, AT-008 through AT-011 |
| Operations | Bounded config, metrics, and redaction | Add video service/playback config and bounded labels | NFR-001 through NFR-003 | UT-012, IT-017, IT-018, IT-020 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `lexicon/social/craftsky/feed/post.json` | Change | Add standard video to the open union and update descriptions | FR-008 | IT-006 |
| `lexicon/README.md`, `justfile` | Change | Document reuse and add `video.json` plus `defs.json` lexgen inputs | FR-008 | IT-006 |
| `appview/internal/lexicon/craftsky/*` | Generate | Add the `appbsky.EmbedVideo` union branch; never hand-edit | FR-008 | IT-006 |
| `appview/internal/video/service_auth.go` | Create | Coordinated, purpose-specific upload/limits authorization | FR-001, FR-002, FR-028 | IT-002, IT-013 |
| `appview/internal/video/service_client.go` | Create | Bounded limits and public job-status calls | FR-004, FR-028, NFR-002 | UT-004, IT-003, IT-013 |
| `appview/internal/video/completion_verifier.go` | Create | Verify completed state, owner, and exact canonical blob | FR-005, FR-006 | UT-004, IT-003 through IT-005 |
| `appview/internal/video/playback.go` | Create | Build only percent-encoded HLS/thumbnail URLs | FR-010, FR-011 | UT-005, IT-008 |
| `appview/internal/video/caption_fetcher.go` | Create | Fetch only an indexed member WebVTT blob under strict bounds | FR-018 | IT-008, IT-017 |
| `appview/internal/api/video_authorization.go`, `video_limits.go`, `video_caption.go` | Create | Implement the three authenticated API handlers | FR-001, FR-018, FR-028 | IT-002, IT-008, IT-013, IT-017 |
| `appview/internal/api/post_request.go`, `post_create.go` | Change | Decode/validate proof and use only a verifier-returned blob | FR-005 through FR-007 | UT-003, IT-003, IT-005 |
| `appview/internal/index/craftsky_post.go` | Change | Recognize and safely validate standard video/captions | FR-009 | IT-007 |
| `appview/internal/api/post_response.go`, `post_response_shape.go` | Change | Hydrate canonical video and caption metadata on all post shapes | FR-010, FR-011, FR-018 | IT-008, IT-019 |
| `appview/internal/routes/policy.go`, `routes_video.go`, `dependencies.go`, `routes.go` | Change/Create | Register exact route policies and dependencies | NFR-001 | IT-017 |
| `appview/internal/app/config.go`, `federated.go`, `deps.go`, `deps_pds.go`, `routes_adapter.go` | Change | Build bounded clients, templates, service DIDs, and dependency graph | FR-001, FR-005, FR-011, FR-028 | IT-002, IT-003, IT-008, IT-013 |
| `appview/internal/observability/*` | Change | Add bounded operation results and secret-safe diagnostics | NFR-003 | IT-018, IT-020 |
| `appview/environments/*.env`, `appview/README.md` | Change | Document video service and playback configuration | FR-001, FR-011 | UT-005, IT-010 |
| `app/lib/feed/models/post.dart` and generated mapper | Change/Generate | Add canonical video, caption, blob, and aspect-ratio views | FR-010 | UT-008, IT-019 |
| `app/lib/feed/models/create_post_video.dart` | Create | Model final job/blob proof without storing the JWT | FR-007, FR-029 | UT-008, IT-009 |
| `app/lib/feed/data/post_api_client.dart`, repository/provider chain | Change | Call limits/authorization/caption routes and serialize proof | FR-001, FR-007, FR-018, FR-028 | IT-002, IT-013, IT-019 |
| `app/lib/feed/data/video_service_client.dart` | Create | Isolated direct upload and tokenless polling | FR-003, FR-004, FR-030 | UT-001, UT-004, UT-012, UT-013, IT-001, IT-010 |
| `app/lib/feed/providers/video_service_client_provider.dart` | Create/Generate | Provide only dedicated external transport/config | FR-003, FR-029 | IT-010, IT-018 |
| `app/lib/feed/providers/composer_video_provider.dart` | Create/Generate | Own local selection, metadata, poster, alt, and validation | FR-012, FR-013, FR-032 | UT-008, AT-002, AT-007 |
| `app/lib/feed/providers/video_publish_attempt_provider.dart` | Create/Generate | Own ephemeral auth/upload/poll/create state and cancellation | FR-022, FR-023, FR-026, FR-027, FR-029 through FR-031 | UT-009, UT-011, UT-013, UT-014, AT-001, AT-003 through AT-005 |
| `app/lib/feed/widgets/composer_image_attachment_section.dart`, standard/project composer sheets | Change | Add explicit `supportsVideo` and media-specific controls | FR-019 through FR-021 | AT-007, AT-013 |
| `app/lib/feed/widgets/submission_blocking_overlay.dart` | Change | Render upload, processing, and final publication stages | FR-022, FR-026 | AT-001, AT-004 |
| `app/lib/drafts/data/*`, `models/*`, `composer/*` | Change | Stream source files, retain poster, version manifest, enforce quota | FR-024, FR-025, NFR-005 | UT-007, UT-015, IT-011, AT-006 |
| `app/lib/feed/widgets/native_video_player.dart`, `native_video_controller.dart` | Create | Isolate `media_kit`, visibility, lifecycle, captions, and errors | FR-015 through FR-018 | UT-010, UT-016, IT-014, IT-015 |
| `app/lib/feed/widgets/post_card.dart` | Change | Add canonical native-video media branch without changing YouTube | FR-014, BR-004 | AT-008 through AT-011, REG-001 through REG-006 |
| `app/pubspec.yaml`, `app/lib/main.dart`, platform manifests/entitlements | Change | Add pinned player/visibility packages and platform initialization | FR-015 | IT-014, MAN-001 through MAN-003 |

## 5. Services, Interfaces, And Data Flow

### AppView capabilities

Keep service authorization purpose-specific. Do not expose a generic method that
accepts arbitrary `aud` or `lxm` values and do not widen every ordinary PDS mock
with raw credential access.

```text
type ServiceAuthIssuer interface {
  IssueUpload(ctx, ownerDID) -> {token, expiresAt}
  IssueLimits(ctx, ownerDID) -> internal credential
}

type VideoServiceClient interface {
  GetUploadLimits(ctx, internalCredential) -> UploadLimits
  GetJobStatus(ctx, jobID) -> JobStatus
}

type CompletionVerifier interface {
  Verify(ctx, ownerDID, jobID, submittedBlob) -> verifiedBlob
}
```

`IssueUpload` derives the user's PDS audience from the validated active OAuth
session, requests `com.atproto.repo.uploadBlob`, and caps expiry at 30 minutes.
`IssueLimits` requests the video-service audience and
`app.bsky.video.getUploadLimits`, then consumes the result inside AppView. Only
the upload token and expiry cross to Flutter.

`CompletionVerifier.Verify` calls public `app.bsky.video.getJobStatus` and:

- requires the returned job ID to match;
- requires terminal completed state, or documented `already_exists` with blob;
- parses and compares `jobStatus.did` with the authenticated `syntax.DID`;
- canonicalizes and exactly compares CID, MIME, and size;
- requires `video/mp4`, positive size, and at most 300,000,000 bytes;
- returns generic typed failures with no foreign identifiers or raw messages.

Post creation validates locally before verification and performs no PDS effect
until verification succeeds. The authored record uses only the verifier-returned
blob plus request alt/aspect ratio.

### Flutter direct transport

```text
type VideoServiceClient {
  upload(source, ownerDID, EphemeralVideoToken, onProgress, cancel) -> JobStatus
  poll(jobID, cancel) -> CompletedVideoProof
}
```

The dedicated Dio instance is not derived from `dioProvider`,
`accountDioProvider`, or `baseDioOptions`; it has no CraftSky interceptors. Before
attaching `Authorization`, it compares the configured URL's scheme, normalized
host, effective port, and exact upload path. It constructs only the documented
owner-DID query, rejects redirects, and never forwards the header. Poll requests
carry no authorization header. Both paths cap response bytes and time.

The source is opened as a stream with known length. Preflight rejects an
oversized source before authorization; the streaming adapter also counts actual
bytes and aborts if metadata is wrong. No `readAsBytes()` path is permitted for
the source video.

### Record and response data flow

```text
composer source
  -> limits recheck
  -> AppView upload authorization
  -> direct uploadVideo stream
  -> direct public getJobStatus polling
  -> {jobId, blob, alt, aspectRatio}
  -> POST /v1/posts
  -> AppView public-status verification
  -> PDS app.bsky.embed.video record
  -> Tap index of complete record
  -> canonical video/HLS/caption response
  -> Flutter player
```

`craftsky_posts.record` and `PostRow.RawEmbed` remain authoritative. Add no video
columns. One response builder parses safe video metadata and is used by create,
timeline, author feed, detail/conversation, saved, search, and notification post
shapes. Malformed playback metadata omits the playable object instead of failing
the rest of the post.

### Captions

`GET /v1/posts/{did}/{rkey}/video-captions/{captionCid}` parses typed identifiers,
loads the indexed post, and proves the exact caption CID is in its video embed.
It then fetches only that author's blob, allows `text/vtt`, caps at 20,000 bytes,
checks a WebVTT signature, and returns `text/vtt`. It never accepts an arbitrary
URL or acts as a general blob proxy.

Flutter downloads through authenticated AppView Dio to a temporary native file
or a revocable web object URL and supplies `SubtitleTrack.uri`. Disposal removes
the temporary resource.

## 6. State, Providers, Controllers, Or DI

Use Riverpod's existing generated-provider pattern. Keep the credential inside
an auto-disposed publication attempt, not in the selected-media state, draft
model, repository model, equality/debug output, or retry payload.

```text
videoServiceConfigurationProvider
  -> videoServiceTransportProvider
  -> videoServiceClientProvider

composerVideoProvider(composerId)
  -> local source handle
  -> metadata + poster + alt + validation

videoPublishAttemptProvider(composerId)
  -> postApiClientProvider for limits/authorization/create
  -> videoServiceClientProvider for upload/poll
  -> ActiveAccountLease checks
  -> ephemeral stage/progress/failure state
```

Capture `ActiveAccountLease` before the first asynchronous operation and check it
between stages. Auto-dispose, success, failure, cancellation, account switch, and
app interruption all clear the JWT, raw header, job continuation, timers, and
cancel token. Interruption never resumes a remote job. Retry starts with fresh
validation, limits, and authorization.

Cancellation is available during upload and processing, preserves the local
source, and stops local work where possible. It becomes unavailable immediately
before final `POST /v1/posts` to avoid implying cancellation of a potentially
committed PDS write.

AppView dependencies add a coordinated `ServiceAuthIssuer`, bounded
`VideoServiceClient`, `CompletionVerifier`, `PlaybackURLBuilder`, and constrained
caption fetcher through existing app/routes dependency structs. None is global
mutable state.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Routes

```text
GET  /v1/blobs/videos/limits
POST /v1/blobs/videos/authorization
GET  /v1/posts/{did}/{rkey}/video-captions/{captionCid}
POST /v1/posts                              # extended request only
```

All new routes require current-member bearer and device ID, preserve request IDs,
use camelCase JSON, and return the standard error envelope. Authorization uses
the upload rate class; limits/captions use bounded read policies. There is no
`POST /v1/blobs/videos`, status proxy, or generic blob route.

Authorization success:

```json
{"token":"<redacted service JWT>","expiresAt":"2026-09-03T12:30:00Z"}
```

Final post proof:

```json
{
  "embed": {
    "video": {
      "jobId": "job-id",
      "blob": {
        "$type": "blob",
        "ref": {"$link": "bafy..."},
        "mimeType": "video/mp4",
        "size": 123
      },
      "alt": "A knitted scarf on a table",
      "aspectRatio": {"width": 16, "height": 9}
    }
  }
}
```

### Composer

Add explicit `supportsVideo` to the shared attachment section. Standard and
project composers pass true; replies, scheduled posts, business products/events,
and all other existing image-only callers pass false. Empty capable composers
offer photo(s) or one video only after successful eligibility preflight. Once one
media kind is selected, only that kind's remove/replace/alt controls appear.

The publish overlay shows localized validating, uploading, processing, and
publishing stages, with determinate progress only when available. Failures map to
actionable categories without raw provider state or messages and return to the
editable composer with the local source intact when safe.

### Drafts

Version the manifest to distinguish image, video source, and video poster. Stream
source copy, incremental SHA-256, and actual byte counting into revision-scoped
temporary files before atomic manifest publication. Only source-video bytes
count toward the exact 1,000,000,000-byte per-account quota. Over-quota or failed
writes preserve existing drafts and remove only incomplete temporary files.
Version-1 image drafts remain readable.

### Playback

Add pinned dependencies `media_kit: 1.2.6`, `media_kit_video: 2.0.1`,
`media_kit_libs_video: 1.0.7`, and `visibility_detector: 0.4.0+2`. Initialize
MediaKit once in `main.dart` after Flutter bindings and before app bootstrap.

`PostCard` chooses native video for canonical `post.video`; existing image,
external-card, and YouTube branches retain their behavior. The app-owned player
opens with autoplay false, pauses offscreen/background, prevents competing audio,
disposes controllers/resources, exposes controls and semantics, and leaves post
text/actions usable on HLS failure. No raw PDS MP4 fallback is constructed.

Platform work includes Android network permission, iOS photo-library copy that
mentions video, and macOS network-client entitlements. Windows, Linux, and web
receive package initialization/assets required by the selected adapters.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Eligibility loading/failure | Do not open picker; show normalized retryable explanation | FR-028 | AT-003, IT-013 |
| Ineligible/quota constrained | Keep Video visible but disabled/explained; show quota only under defined threshold | FR-028 | AT-003, IT-013 |
| Invalid type/size/duration | Reject locally before authorization; unknown duration may continue | FR-003, FR-013, RULE-001, RULE-002 | UT-001, AT-002 |
| Redirect or destination mismatch | Abort before forwarding authorization; do not follow redirect | FR-003, NFR-003 | UT-012, IT-010, IT-018 |
| Upload disconnect/ambiguous result | Clear attempt credential; retain local source; retry starts over | FR-023, FR-031 | UT-013, AT-005 |
| Processing without progress | Show indeterminate stage and bounded polling | FR-022, FR-030 | UT-009, AT-004 |
| Cancel upload/processing | Stop local stream/poll, clear secrets, retain source | FR-026, FR-029 | UT-011, UT-014, AT-005 |
| App/account interruption | Dispose attempt; never resume or silently publish | FR-027, FR-029 | IT-009, IT-012, AT-005 |
| Foreign/incomplete/mismatched proof | Generic rejection and zero PDS effects | FR-005 | UT-004, IT-003, IT-005 |
| Verification outage | Fail closed with standard upstream error; retain editable source | FR-005, NFR-002 | IT-003, AT-005 |
| `already_exists` with exact blob | Treat as completion but still verify owner and blob | FR-006, FR-031 | UT-004, IT-003 |
| Malformed indexed video | Preserve post ingestion safety; omit unsafe playback metadata | FR-009, FR-010 | IT-007, IT-008 |
| Missing/broken HLS | Show recoverable media error; preserve text/actions; no MP4 fallback | FR-016 | UT-016, AT-009 |
| Invalid/nonmember caption | Reject without fetching arbitrary blobs or exposing PDS access | FR-018 | IT-008, IT-017 |
| Draft over quota/write failure | Keep prior revision intact and show cleanup guidance | FR-024, FR-025, NFR-005 | UT-007, IT-011, AT-006 |

## 9. Test Implementation Plan

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-003 | `appview/internal/api/post_request_test.go` | Valid/malformed video proof and mixed media/reply matrix | Decoder lacks video; validation does not enforce mutual exclusion |
| 2 | IT-006 | Lexicon/generated contract tests | Minimal/full video records with captions/aspect ratio | Union has no standard video branch |
| 3 | UT-004, IT-003, IT-005 | `internal/video` verifier and create handler | Alice/Bob, incomplete, mismatch, `already_exists`, outage | Client proof is not independently verified |
| 4 | IT-002, IT-013, IT-017 | Service auth, limits handlers, route catalogue | Fake coordinated OAuth/PDS/video service | Purpose-bound operations/routes do not exist |
| 5 | UT-001, UT-011 through UT-013, IT-001, IT-009, IT-010, IT-012, IT-018 | Flutter external transport and attempt lifecycle | Generated streams, redirect server, storage/log spies, fake clock | No isolated streaming client or ephemeral lifecycle exists |
| 6 | IT-007, UT-005, IT-008, IT-019 | Index and canonical response/caption route | Tap records, malformed embeds, all post surfaces | Video is not indexed or hydrated |
| 7 | UT-008, IT-013, AT-002, AT-003, AT-007, AT-013 | Composer model and widgets | Picker/metadata/eligibility/provider overrides | No capability-aware video selection exists |
| 8 | UT-007, UT-015, IT-011, AT-006 | Draft store/repository | Sparse files, failure injection, v1/v2 manifests | Storage buffers images and has no source quota |
| 9 | UT-009, UT-014, AT-001, AT-004, AT-005 | Publish orchestration and overlay | Fake staged service, progress, cancellation | No video stage machine exists |
| 10 | UT-010, UT-016, IT-014, IT-015, AT-008 through AT-011 | Player adapter and post surfaces | Fake controller, visibility/lifecycle, HLS/captions | Post card has no native-video branch |
| 11 | IT-020, AT-012, REG-001 through REG-008 | Metrics, security, and regression suites | Bounded-label recorder and existing post fixtures | Required telemetry/redaction/regression assertions are absent |
| 12 | MAN-001 through MAN-005 | Six-platform/manual release matrix | Real eligible account and release builds | Environmental evidence has not been collected |

Run each row red-green-refactor before moving to dependent rows. Generated files
are refreshed only after the lexicon test fails for the intended reason.

## 10. Sequencing And Guardrails

- First TDD step: Add only request decode, BlobRef structure, alt/aspect-ratio,
  and mutual-exclusion cases for `UT-003`; keep remote owner/status checks out of
  this unit.
- First focused command: from `appview/`,
  `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run 'Test.*Post.*Video'`.
- Lexicon sequence: amend schema/lexgen inputs, run `just lexgen`, inspect all
  generated changes, then run `just lexgen-check` and the generated contract test.
- Server sequence: verifier before handler integration; service auth before
  authorization/limits routes; response hydration before Flutter parsing.
- Client sequence: isolated transport and secret lifecycle before composer UI;
  streamed draft primitives before draft video integration; player abstraction
  before post-card integration.
- Guardrail: Never expose OAuth tokens, refresh tokens, DPoP keys, PDS URL, raw
  PDS client, generic service-auth parameters, or the internal limits credential.
- Guardrail: Never attach the service JWT through a general interceptor, persist
  it, log it, include it in a URL, follow a redirect with it, or use it for polling.
- Guardrail: Never trust client job/blob proof or perform a PDS effect before
  successful independent verification.
- Guardrail: Never add AppView video-job persistence, cleanup, automatic resume,
  upload/status proxy routes, generic blob proxying, or invented idempotency.
- Guardrail: Never buffer an additional full source video in Flutter or AppView.
- Guardrail: Never construct raw PDS MP4 playback fallback.
- Out of scope: camera capture, transcoding, caption upload/editing, mixed media,
  replies, scheduled video, video quotes, business-media video, resumable upload,
  processed-blob deletion, and general TMB credential handoff.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking release gate | Browser CORS may reject direct upload or HLS/caption resources. | Web could lack authoring or playback. | Verify with a real eligible account before release; do not silently add an AppView upload proxy fallback. |
| CPQ-002 | Non-blocking implementation check | Exact Indigo coordinated call shape for `getServiceAuth` must fit the pinned API. | May change internal adapter signatures. | Inspect the pinned package during the service-auth TDD step; keep the public purpose-specific interface fixed. |
| CPQ-003 | Non-blocking | Pinned Indigo's bundled video schema still describes 100 MB although current service/schema policy is 300 MB. | Schema comments/inputs drift from current policy. | ADR 013 accepts the compatible generated shape and application-level 300 MB checks; upgrade Indigo only after separate review if required. |
| CPQ-004 | Non-blocking release gate | `media_kit` HLS, WebVTT, full-screen, and native libraries vary by target. | Platform-specific playback defects or binary growth. | Keep package types behind the adapter and complete MAN-001 through MAN-003. |
| CPQ-005 | Non-blocking release gate | Public job status availability is required for final server verification. | Publication fails closed during service outage or contract drift. | Keep bounded outage behavior; require a new ADR before adopting receipts or registration. |
| CPQ-006 | Residual security risk | A stolen service JWT can be replayed until expiry. | Unauthorized blob upload within narrow scope. | Shortest accepted lifetime capped at 30 minutes, memory-only lifecycle, exact destination, rejected redirects, and secret scans. |

No blocking implementation-design question remains.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-003` request decoding and local invariants, followed by
  `IT-006` lexicon generation and `UT-004` completion verification.
- Focused command: from `appview/`,
  `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run 'Test.*Post.*Video'`.
- Broader verification commands:
  - `just lexgen-check`
  - `just appview-test-unit`
  - `just test` after `just dev-d`
  - `just appview-check`
  - `just app-test <test path or directory>`
  - `just app-test`
  - `just app-analyze`
- Approval: Do not invoke `implement-tdd`, change source, add dependencies, run
  migrations, or generate code until the user explicitly approves this high-risk
  plan.
