# Acceptance Test Specification: Video Posts

## 1. Test Strategy

Risk level: High. The feature hands an untrusted Flutter client a short-lived
PDS-signed bearer, crosses a direct 300 MB third-party streaming boundary,
verifies untrusted completion data, changes a project-wide credential rule, and
also changes a PDS record contract, local draft storage, and native playback.
Implementation therefore requires explicit approval after document review.

The test strategy separates concerns so failures identify the affected layer:

- Unit tests cover media validation, grapheme limits, service-state
  normalization, mutual exclusion, quota presentation, polling, ephemeral
  credential lifecycle, retry behavior, and redaction.
- Go integration tests cover authenticated authorization/limits routes, exact
  service-JWT claims, independent job/blob verification, PDS writes, Tap
  indexing, canonical responses, and API conventions.
- Flutter provider, repository, and widget tests automate user-visible composer,
  direct external upload/polling, credential containment, draft, progress,
  player, accessibility, and failure behavior using injected transports,
  pickers, clocks, and an app-owned abstraction over `media_kit`.
- Regression tests protect existing image, text, quote, external-card, reply,
  scheduling, business, and YouTube behavior.
- Manual checks are limited to real HLS playback and platform/plugin behavior
  that cannot be established by a fake controller alone.

External Bluesky services and a real PDS are not required for the deterministic
suite. A scripted fake video service and fake PDS verify wire behavior. A small
optional live-service smoke test remains a documented gap until the playlist
contract and eligible test account are confirmed.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-007 | AT-001, IT-005 | Acceptance / Integration | Yes |
| BR-002 | AC-009, AC-010 | AT-010, IT-007, IT-008 | Acceptance / Integration | Yes |
| BR-003 | AC-002 | IT-002, IT-010, IT-018 | Integration | Yes |
| BR-004 | AC-016 | REG-001, REG-002, REG-003, REG-004, REG-005 | Regression | Yes |
| FR-001 | AC-001, AC-003 | AT-001, IT-001 | Acceptance / Integration | Yes |
| FR-002 | AC-002 | IT-002, UT-012 | Unit / Integration | Yes |
| FR-003 | AC-003, AC-014 | IT-001, IT-012 | Integration | Yes |
| FR-004 | AC-004, AC-005 | AT-004, IT-003 | Acceptance / Integration | Yes |
| FR-005 | AC-004, AC-007, AC-013 | IT-003, IT-005 | Integration | Yes |
| FR-006 | AC-006 | UT-004, IT-003 | Unit / Integration | Yes |
| FR-007 | AC-007, AC-041 | UT-003, IT-005 | Unit / Integration | Yes |
| FR-008 | AC-008 | IT-006 | Integration | Yes |
| FR-009 | AC-009, AC-021 | IT-007 | Integration | Yes |
| FR-010 | AC-009, AC-010 | UT-008, IT-008 | Unit / Integration | Yes |
| FR-011 | AC-010, AC-011 | UT-005, IT-008, AT-009 | Unit / Integration / Acceptance | Yes |
| FR-012 | AC-001, AC-012, AC-026, AC-042 | AT-001, AT-002 | Acceptance | Yes |
| FR-013 | AC-003, AC-015 | UT-001 | Unit | Yes |
| FR-014 | AC-010 | AT-010 | Acceptance | Yes |
| FR-015 | AC-017, AC-018 | AT-008, UT-010 | Acceptance / Unit | Yes |
| FR-016 | AC-011, AC-019 | AT-009 | Acceptance | Yes |
| FR-017 | AC-020, AC-041 | AT-011 | Acceptance | Yes |
| FR-018 | AC-021 | AT-011, IT-007 | Acceptance / Integration | Yes |
| FR-019 | AC-027, AC-028 | AT-007, REG-006 | Acceptance / Regression | Yes |
| FR-020 | AC-027, AC-044 | AT-007 | Acceptance | Yes |
| FR-021 | AC-028 | AT-007, REG-006 | Acceptance / Regression | Yes |
| FR-022 | AC-001, AC-004, AC-029 | AT-001, AT-004 | Acceptance | Yes |
| FR-023 | AC-005, AC-030, AC-035 | AT-004 | Acceptance | Yes |
| FR-024 | AC-031, AC-032 | AT-006, IT-011 | Acceptance / Integration | Yes |
| FR-025 | AC-031 | AT-006, UT-007, IT-011 | Unit / Integration / Acceptance | Yes |
| FR-026 | AC-033 | AT-005 | Acceptance | Yes |
| FR-027 | AC-034 | AT-005 | Acceptance | Yes |
| FR-028 | AC-036, AC-037 | AT-003, UT-006, IT-013 | Unit / Integration / Acceptance | Yes |
| FR-029 | AC-038 | UT-011, IT-009 | Unit / Integration | Yes |
| FR-030 | AC-039 | UT-009, AT-005 | Unit / Acceptance | Yes |
| FR-031 | AC-035, AC-040 | UT-013, IT-010 | Unit / Integration | Yes |
| FR-032 | AC-041 | AT-011, UT-002 | Unit / Acceptance | Yes |
| NFR-001 | AC-001, AC-005 | IT-001, IT-003, IT-017 | Integration | Yes |
| NFR-002 | AC-014 | IT-012, IT-020 | Integration | Yes |
| NFR-003 | AC-002, AC-022 | UT-012, IT-018 | Unit / Integration | Yes |
| NFR-004 | AC-023 | AT-008, MAN-002 | Acceptance / Manual | Partial |
| NFR-005 | AC-031, AC-043 | IT-011, MAN-004 | Integration / Manual | Partial |
| RULE-001 | AC-003, AC-015 | UT-001 | Unit | Yes |
| RULE-002 | AC-003 | UT-001, IT-001 | Unit / Integration | Yes |
| RULE-003 | AC-007, AC-024 | UT-003, IT-005, AT-007 | Unit / Integration / Acceptance | Yes |
| RULE-004 | AC-012, AC-016, AC-031 | AT-002, AT-006, REG-002 | Acceptance / Regression | Yes |
| RULE-005 | AC-004, AC-007, AC-026 | AT-001, AT-002, IT-005 | Acceptance / Integration | Yes |
| RULE-006 | AC-005 | UT-004, AT-003, AT-004 | Unit / Acceptance | Yes |
| RULE-007 | AC-025 | AT-012 | Acceptance | Yes |
| RULE-008 | AC-024, AC-027 | AT-007 | Acceptance | Yes |
| RULE-009 | AC-042 | AT-002 | Acceptance | Yes |
| RULE-010 | AC-041 | UT-002, AT-011 | Unit / Acceptance | Yes |
| RULE-011 | AC-045 | AT-013 | Acceptance | Yes |

## 3. Acceptance Scenarios

### AT-001: Publish An Eligible Video Post
Requirement IDs: BR-001, FR-001, FR-012, FR-022, RULE-005
Acceptance Criteria: AC-001, AC-004, AC-007
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_video_publish_test.dart`

```gherkin
Feature: Native video publication
  Scenario: An eligible user publishes a valid video
    Given a signed-in user with a valid device ID selected a valid local MP4
    And no upload or remote job has started
    When the user chooses Publish
    Then the blocking screen shows upload, processing, and publication stages
    And AppView returns only a short-lived upload service JWT
    And the client streams directly to the approved video.bsky.app endpoint
    And the client polls public job status directly
    And AppView verifies the completed owner-matched job and exact blob
    And one video post is created on the user's PDS
```

### AT-002: Select Existing Video Without Uploading
Requirement IDs: FR-012, RULE-004, RULE-005, RULE-009
Acceptance Criteria: AC-012, AC-026, AC-042
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/composer_videos_provider_test.dart`

```gherkin
Feature: Local video selection
  Scenario: Selection remains local until publish
    Given an immediate top-level standard or project composer
    When the user chooses Video and selects an existing MP4
    Then one local preview supports replace and remove
    And no upload request or remote job occurs before Publish
    And no camera recording option is offered
    And reply and scheduled-post composers expose no video attachment path
```

### AT-003: Explain Eligibility And Constrained Quota
Requirement IDs: FR-028, RULE-006
Acceptance Criteria: AC-005, AC-036, AC-037
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/composer_video_eligibility_test.dart`

```gherkin
Feature: Video eligibility preflight
  Scenario Outline: The video choice communicates account eligibility
    Given an empty video-capable attachment widget
    And preflight returns <state> with <allowance>
    When the user chooses Video
    Then the Video choice remains visible
    And the UI shows <message>
    And the picker opens only after an eligible preflight result
    But the picker does not open for an ineligible or failed preflight result
    And eligibility is checked again when Publish is chosen

    Examples:
      | state | allowance | message |
      | eligible | unconstrained | no routine quota detail |
      | eligible | one video remaining | remaining video quota |
      | eligible | fewer than 300000000 bytes | remaining byte quota |
      | ineligible | upload disallowed | a specific actionable explanation |
```

### AT-004: Show Progress And Recover From Failure
Requirement IDs: FR-004, FR-022, FR-023, FR-031, RULE-006
Acceptance Criteria: AC-004, AC-005, AC-029, AC-030, AC-035
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/video_submission_overlay_test.dart`

```gherkin
Feature: Video publication feedback
  Scenario: Processing reports progress and then fails
    Given publication has started for a selected video
    When upload reports a percentage and processing has no percentage
    Then the screen shows determinate upload and indeterminate processing progress
    And stage labels are localized rather than raw service constants
    When processing fails with a normalized actionable reason
    Then no post is created
    And the editable composer retains the local video when safe
    And Retry obtains fresh authorization and restarts validation through publication
```

### AT-005: Cancel Or Interrupt Publication Safely
Requirement IDs: FR-026, FR-027, FR-030
Acceptance Criteria: AC-033, AC-034, AC-039
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/composer/video_publication_lifecycle_test.dart`

```gherkin
Feature: Video publication lifecycle
  Scenario: User cancels while processing
    Given video upload or processing is active
    When the user cancels
    Then local transfer and polling stop where possible
    And the composer returns with the local video intact
    And no orphan warning is shown

  Scenario: Publication reaches final post creation
    Given processing returned a valid blob
    When final post creation begins
    Then cancellation is unavailable

  Scenario: The application is interrupted
    Given a remote video job was active when the application stopped
    When the user reopens the retained draft
    Then the old credential and job are absent from persisted state
    And no post is silently published
    And choosing Publish starts with fresh authorization
```

### AT-006: Save And Reopen Video Drafts Within Quota
Requirement IDs: FR-024, FR-025, RULE-004
Acceptance Criteria: AC-031, AC-032
Priority: Must
Level: Acceptance
Automation Target: `app/test/drafts/video_draft_flow_test.dart`

```gherkin
Feature: Local video drafts
  Scenario: Standard and project video drafts remain local
    Given an account has enough of its 1000000000-byte video-draft quota
    When a standard or project draft with a source video is saved and reopened
    Then the source video and small local poster are restored
    And the drafts list uses the poster without starting playback
    And no media is uploaded

  Scenario: Saving would exceed quota
    Given existing drafts consume the available account quota
    When another video draft would exceed 1000000000 source-video bytes
    Then the save fails with cleanup guidance
    And no existing draft is evicted, altered, or partially replaced
```

### AT-007: Respect Attachment Capabilities And Media Exclusivity
Requirement IDs: FR-019, FR-020, FR-021, RULE-003, RULE-008
Acceptance Criteria: AC-024, AC-027, AC-028, AC-044
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/composer_media_attachment_section_test.dart`

```gherkin
Feature: Capability-aware composer attachments
  Scenario: Empty video-capable post widget offers media type choice
    Given a standard or project attachment widget has video capability enabled
    And it has no media
    When the user activates it
    Then localized copy identifies photos and video
    And the user chooses photo or video before the matching picker opens
    And selected media receives only type-specific controls

  Scenario: Existing media cannot be silently replaced by another type
    Given photos or a video are already selected
    When the user tries to select the other media type
    Then the current media remains intact
    And the user is told to remove it explicitly first

  Scenario: Image-only widget stays image-only
    Given a business product or event widget has video capability disabled
    When it renders and is activated
    Then its copy and picker are photo-only
    And no video path is exposed
```

### AT-008: Control HLS Playback And Player Lifecycle
Requirement IDs: FR-015, NFR-004
Acceptance Criteria: AC-017, AC-018, AC-023
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/native_video_player_test.dart`

```gherkin
Feature: Native video playback
  Scenario: Playback starts only after user action
    Given a native video card enters the viewport
    Then video and audio do not autoplay
    When the user starts playback
    Then play, pause, seek, platform-supported volume, time, replay, and full-screen controls are available
    When the card leaves the active viewport or the app enters the background
    Then playback pauses
    And idle, loading, playing, and error states keep a stable bounded layout
```

### AT-009: Fail Soft Without Raw PDS Playback
Requirement IDs: FR-011, FR-016
Acceptance Criteria: AC-011, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/native_video_player_failure_test.dart`

```gherkin
Feature: HLS-only playback failure
  Scenario Outline: HLS cannot be played
    Given a video post whose HLS playlist is <condition>
    When the post renders or playback is attempted
    Then a recoverable media error is shown
    And post text, author, navigation, and actions remain usable
    And no raw PDS MP4 request is made

    Examples:
      | condition |
      | absent |
      | expired |
      | unsupported |
      | failed |
```

### AT-010: Render Native Video Across Standard Post Surfaces
Requirement IDs: BR-002, FR-014
Acceptance Criteria: AC-009, AC-010
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/native_video_surfaces_test.dart`

```gherkin
Feature: Canonical native video rendering
  Scenario Outline: Indexed video appears on a standard post surface
    Given a canonical post response contains an HLS-backed native video
    When it is rendered in <surface>
    Then the native player receives only AppView playback metadata

    Examples:
      | surface |
      | timeline card |
      | profile post list |
      | saved result |
      | search result |
      | post detail thread |
```

### AT-011: Author And Consume Accessible Video Metadata
Requirement IDs: FR-017, FR-018, FR-032, RULE-010
Acceptance Criteria: AC-020, AC-021, AC-041
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/video_accessibility_test.dart`

```gherkin
Feature: Video accessibility metadata
  Scenario: Author supplies optional alt text
    Given a selected video
    When the author enters no more than 1000 graphemes or explicitly skips
    Then publication may proceed
    And supplied text is exposed through player semantics and descriptive context
    But text over 1000 graphemes blocks publication

  Scenario: An indexed video has WebVTT captions
    Given AppView provides a safe valid caption source
    When playback begins
    Then the caption track can be enabled
    And absence of captions does not block playback
```

### AT-012: Describe Published Video As Public
Requirement IDs: RULE-007
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/video_privacy_copy_test.dart`

```gherkin
Feature: Video privacy copy
  Scenario: Composer explains video storage
    Given video composition or explanatory copy is visible
    When the copy is inspected
    Then it does not state or imply that uploaded video is private
```

### AT-013: Publish On A Metered Network Without An Extra Prompt
Requirement IDs: RULE-011
Acceptance Criteria: AC-045
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/composer/video_metered_network_test.dart`

```gherkin
Feature: Metered-network publication
  Scenario: Cellular connection does not add a confirmation
    Given an otherwise valid video and a metered or cellular connection
    When the user chooses Publish
    Then publication starts without an additional data-usage warning or confirmation
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, FR-013, RULE-001, RULE-002 | AC-003, AC-015 | Validate source type, bytes, and known duration at exact boundaries before authorization. | Valid MP4; false `.mp4`; MIME mismatch; 300000000 and 300000001 bytes; 600, 600.001, and unknown seconds | Exact limits pass; wrong content/type and excess values fail before token request; unknown duration defers to service. | `app/test/feed/media/video_source_validator_test.dart` |
| UT-002 | FR-032, RULE-010 | AC-041 | Count Unicode grapheme clusters rather than code units for optional alt text. | Empty, skip, 1000 graphemes, 1001 graphemes, combining marks, emoji sequences | Empty/skip and 1000 pass; 1001 fails without splitting a grapheme. | `app/test/feed/media/video_alt_text_test.dart` |
| UT-003 | FR-007, RULE-003 | AC-007, AC-024 | Validate one video proof and native-media mutual exclusion at the request boundary. | Job ID/blob alone; video plus images/quote/external/reply; duplicate variant; malformed CID/MIME/size; invalid alt/aspect ratio | Exactly one structurally valid proof is accepted; all local conflicts fail before remote verification or a PDS effect. | `appview/internal/api/post_request_test.go` |
| UT-004 | FR-004, FR-005, FR-006, RULE-006 | AC-005, AC-006, AC-013 | Normalize direct service states and independently verify completion proof. | Client status matrix plus Alice/Bob, incomplete, matching/mismatched blob, `already_exists`, contradictory payload, and outage results | Flutter produces stable actionable states; AppView accepts only completed owner/exact-blob proof, handles documented `already_exists`, and otherwise fails closed without foreign detail. | `app/test/feed/models/video_service_result_test.dart`, `appview/internal/video/completion_verifier_test.go` |
| UT-005 | FR-011 | AC-010, AC-011 | Derive HLS identity from stable DID/CID and prohibit raw-PDS output. | Valid DID/CID containing reserved characters, configured templates, current Bluesky defaults, moved handle, missing configuration | DID/CID are percent-encoded into the configured `playlist.m3u8` and `thumbnail.jpg` templates; invalid configuration fails closed; no raw PDS MP4 URL is generated. | `appview/internal/video/playback_test.go` |
| UT-006 | FR-028 | AC-036, AC-037 | Decide whether quota detail is constrained and format display state. | `canUpload=false`; videos 0/1/2; bytes 299999999/300000000; omitted fields | Quota appears only in the three specified constrained cases; specific ineligibility remains visible. | `app/test/feed/models/video_upload_limits_test.dart` |
| UT-007 | FR-025 | AC-031 | Compute account-scoped source-video draft quota without posters/images and plan atomic saves. | Existing account media, replacement, deletion, exact 1000000000 total, one byte over, second account | Exact quota passes; excess fails before mutation; accounts are isolated; no eviction is planned. | `app/test/drafts/data/video_draft_quota_test.dart` |
| UT-008 | FR-010 | AC-009, AC-010 | Parse and serialize optional canonical video metadata. | Full and minimal camelCase video JSON, caption list, absent video, unknown additive fields | Models round-trip supported fields and remain compatible when video or optional metadata is absent. | `app/test/feed/models/post_test.dart` |
| UT-009 | FR-030 | AC-039 | Calculate direct polling delays and stop conditions with an injected clock. | Initial state, repeated processing, `Retry-After`, terminal states, cancel, interruption, one-hour expiry | Starts near one second, backs off to at least five-second spacing, honors longer header, and stops for every terminal condition. | `app/test/feed/composer/video_job_poller_test.dart` |
| UT-010 | FR-015 | AC-017, AC-018 | Drive the app-owned `media_kit` adapter lifecycle with a fake controller. | Open with `play: false`, enter/leave viewport, background/resume, multiple cards, dispose | No autoplay; offscreen/background pauses; players dispose; no competing audio persists. | `app/test/feed/widgets/native_video_controller_test.dart` |
| UT-011 | FR-029 | AC-038 | Clear ephemeral credential/job state on every boundary. | Success, failure, cancel, account change, lifecycle interruption, provider disposal | Service JWT, authorization header, and remote continuation data are cleared and are never serializable. | `app/test/feed/composer/video_credential_lifecycle_test.dart` |
| UT-012 | FR-002, FR-029, NFR-003 | AC-002, AC-022, AC-038 | Redact service auth and user media from client/server errors, logs, storage, and diagnostics. | OAuth tokens, DPoP key, service JWT, authorization header, video bytes, filename/path, job/DID/CID/URL, alt/caption text | Secrets/content never appear; only bounded state, request ID, byte band, and safe failure class may appear. | `appview/internal/observability/video_redaction_test.go` and `app/test/observability/video_secret_scan_test.dart` |
| UT-013 | FR-031 | AC-035, AC-040 | Manage fresh-authorization retry attempts. | Poll recovery, ambiguous upload, explicit Retry, interruption, owner change, `already_exists` | Poll recovery may keep the current job; upload/manual/interrupted retry discards old credentials, obtains fresh authorization, and accepts reuse only through verified `already_exists`. | `app/test/feed/composer/video_publish_attempt_test.dart` |
| UT-014 | FR-022, FR-023, FR-026 | AC-029, AC-030, AC-033 | Validate publication state-machine transitions and cancellation boundary. | Local, validating, uploading, processing, publishing, complete, failed, canceled | Only valid transitions occur; cancellation is enabled only during upload/processing; retry starts at validation. | `app/test/feed/composer/video_submission_state_test.dart` |
| UT-015 | FR-024, NFR-005 | AC-031, AC-043 | Encode video source/poster descriptors without embedding bytes or private paths in diagnostics. | Standard/project manifests, source descriptor, poster descriptor | Additive manifests round-trip and diagnostics remain content-free. | `app/test/drafts/models/draft_manifest_codec_test.dart` |
| UT-016 | FR-010, NFR-004 | AC-023 | Select declared aspect ratio or stable fallback dimensions. | Valid ratios, absent ratio, invalid ratio, missing thumbnail | Layout remains positive, bounded, and stable across player states. | `app/test/feed/widgets/native_video_layout_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, FR-003, NFR-002, RULE-002 | AC-001, AC-003, AC-014 | Exercise direct Flutter streaming and source admission. | Dedicated fake external transport; exact/over-size streams, MIME/header fixtures, cancellation, timeout | Publish exact-limit, over-limit, wrong content, and disconnected streams | Exact valid MP4 streams directly without an extra full copy; invalid input fails before authorization/upload; cancellation closes the body. | `app/test/feed/data/video_service_client_test.dart` |
| IT-002 | BR-003, FR-002, NFR-001 | AC-002 | Verify bounded service-auth handoff and OAuth credential separation. | Authenticated AppView route, fake coordinated PDS client, fixed clock, sentinel OAuth/DPoP/service values | Request successful/failed authorization | Response contains only service JWT plus expiry; claims use only user PDS audience and `com.atproto.repo.uploadBlob`; expiry is within 30 minutes; OAuth/DPoP values never leave AppView. | `appview/internal/api/video_authorization_test.go`, `appview/internal/routes/video_routes_test.go` |
| IT-003 | FR-004, FR-005, FR-006, NFR-001 | AC-004, AC-005, AC-006, AC-013 | Poll directly and independently verify submitted completion. | Fake public video service with Alice/Bob, incomplete, matching/mismatched, `already_exists`, and outage results | Flutter polls; then POST video proof through AppView | Flutter normalizes public states; AppView accepts only completed owner DID plus exact blob, discloses no foreign details, and fails closed on verification outage. | `app/test/feed/data/video_service_client_test.dart`, `appview/internal/video/completion_verifier_test.go`, `appview/internal/api/post_test.go` |
| IT-004 | FR-004, FR-022 | AC-004, AC-029 | Map direct video-service responses into Flutter progress states. | Dedicated external client with upload, processing, determinate, indeterminate, and terminal payloads | Client uploads and polls a job | Service fields map into general localized stages without exposing raw constants in UI. | `app/test/feed/data/video_service_client_test.dart` |
| IT-005 | BR-001, FR-005, FR-007, RULE-003, RULE-005 | AC-007, AC-013, AC-024 | Create exactly one standard video embed after independent verification. | Authenticated user, fake completion verifier and PDS effects | POST matching, incomplete, foreign, mismatched, malformed, and mixed-media create requests | Only a matching owner-completed job/blob writes `app.bsky.embed.video`; rejected requests create no PDS effect/post. | `appview/internal/api/post_request_test.go`, `appview/internal/api/post_test.go` |
| IT-006 | FR-008 | AC-008 | Validate additive lexicon and generated type compatibility. | Existing non-video records and full/minimal video records | Validate, marshal, and unmarshal records after `just lexgen` | Old records remain valid; new variant round-trips; video stays optional. | `appview/internal/lexicon/craftsky/feedpost_test.go`, `appview/internal/lexicon/craftsky/authoring_path_test.go` |
| IT-007 | BR-002, FR-009, FR-018 | AC-009, AC-021 | Index Craftsky-authored and external standard video embeds idempotently. | Tap create/update/replay/delete events with metadata, valid captions, malformed captions/embed | Dispatch events and query projection | CID, MIME, size, alt, ratio, and safe valid WebVTT metadata survive; malformed embeds fail safely; replay is idempotent. | `appview/internal/index/craftsky_post_test.go` |
| IT-008 | BR-002, FR-010, FR-011, FR-018 | AC-009, AC-010, AC-011, AC-021 | Hydrate canonical video on every post-shaped endpoint and constrain caption delivery. | Indexed video post, configured/current-default templates, valid/foreign/oversized/invalid WebVTT captions, missing thumbnail, and invalid configuration | Read feed/profile/saved/search/detail responses and request caption URIs | Every post shape matches; valid member captions return bounded `text/vtt`; invalid/nonmember captions fail without general PDS/blob access; playback failures are soft and no raw MP4 source appears. | `appview/internal/api/post_response_test.go`, `appview/internal/api/saved_post_response_test.go`, `appview/internal/api/video_caption_test.go` |
| IT-009 | FR-029, NFR-003 | AC-038 | Prove ephemeral state is not persisted. | Provider container, fake token/job, account registry, draft repository, secure storage spy | Succeed, fail, cancel, switch account, interrupt, and reconstruct app state | No service JWT/header/job continuation appears in drafts, secure storage, account state, or reconstructed providers; AppView has no video-job table. | `app/test/feed/composer/video_credential_persistence_test.dart`, AppView migration/schema assertions |
| IT-010 | BR-003, FR-003, FR-031 | AC-002, AC-014, AC-040 | Contain service authorization to one exact external request. | Separate Dio client; approved URL plus cross-origin/same-origin redirects, alternate paths, retries, and interceptor spies | Attempt upload and each redirect/retry case | Craftsky bearer is never attached; service JWT is attached only to exact HTTPS upload origin/path; redirects are rejected; fresh upload retry obtains fresh authorization. | `app/test/feed/data/video_service_transport_security_test.dart` |
| IT-011 | FR-024, FR-025, NFR-005 | AC-031, AC-032, AC-043 | Persist and recover source video/poster atomically with bounded file operations. | Temporary account directories, injected clock/UUID, sparse or generated large stream, failure injection | Save/reopen/replace over quota, restart repository, interrupt copy/manifest replacement | Source/poster recover; quota is account-scoped; failures leave prior drafts intact; operations use streaming APIs without a full extra byte array. | `app/test/drafts/data/file_local_post_draft_repository_video_test.dart` |
| IT-012 | FR-003, NFR-002 | AC-014 | Bound direct client resources across disconnect, cancellation, slow source, timeout, redirect, and outage. | Counting stream, blocking external adapter, short injected deadlines, completion probes | Abort each phase and exceed limits | Cancellation propagates; streams/futures finish; redirects leak no auth; retry classification is stable; memory is independent of full payload size. | `app/test/feed/data/video_service_resource_bounds_test.dart` |
| IT-013 | FR-028 | AC-036, AC-037 | Implement the limits route, internal method-bound authorization, picker gate, and publish-time recheck. | Auth/device middleware, fake coordinated service auth, and `getUploadLimits` responses for eligible, constrained, email, provider, quota, unknown, and upstream failure | Call `GET /v1/blobs/videos/limits`, select Video, and later publish through the API client | AppView keeps the limits credential server-side; the route returns only normalized camelCase limits or standard errors; Flutter waits for eligible success before opening the picker and rechecks at publish. | `appview/internal/api/video_limits_test.go`, `appview/internal/routes/video_routes_test.go`, `app/test/feed/data/video_api_client_test.dart`, `app/test/feed/widgets/composer_video_eligibility_test.dart` |
| IT-014 | FR-012, FR-023, FR-026, FR-027 | AC-026, AC-030, AC-033, AC-034 | Coordinate local selection, publish, retry, cancel, and interruption with a fake repository. | Provider container, fake picker/API/repository, completer-controlled job | Exercise complete lifecycle and reconstruct provider after interruption | No pre-publish network call; local media survives safe failures/cancel; old job is not resumed; new user attempt starts fresh. | `app/test/feed/composer/video_publication_coordinator_test.dart` |
| IT-015 | FR-015, FR-016, FR-018 | AC-017, AC-018, AC-019, AC-021 | Integrate the player widget with an injected `media_kit` adapter and caption source. | Fake adapter reporting HLS events, viewport, lifecycle, errors, and URI WebVTT tracks | Open with autoplay disabled, play, scroll away, background, fail, enable captions, enter full screen, and dispose | Expected controls/lifecycle/error/caption behavior occurs without real network or raw fallback, and package objects do not escape the adapter boundary. | `app/test/feed/widgets/native_video_player_test.dart` |
| IT-016 | FR-014 | AC-010 | Verify canonical model reaches all reusable full-post surfaces. | Canonical video fixture and player-builder override | Render each standard full-post route/widget | Each surface instantiates native player exactly once with the same video metadata. | `app/test/feed/widgets/native_video_surfaces_test.dart` |
| IT-017 | NFR-001, FR-018 | AC-001, AC-005, AC-021 | Verify authorization, limits, and caption route catalogue, auth class, device ID, request ID, and error envelope. | Registered routes and malformed/unauthorized requests | Inspect route metadata and HTTP responses | Authorization, limits, and constrained caption routes use the expected method/auth/body class and standard camelCase failures; no upload/status or general blob-proxy routes exist. | `appview/internal/routes/catalogue_test.go`, `appview/internal/routes/video_routes_test.go`, `appview/internal/routes/video_caption_routes_test.go` |
| IT-018 | BR-003, NFR-003 | AC-002, AC-022, AC-038 | Capture client/server diagnostics across authorization, direct upload/polling, verification, indexing, and playback failures. | Recording Dart reporter/slog handler with injected OAuth, DPoP, service JWT, headers, media, authored text, IDs, and URLs | Trigger each success/failure path | Sensitive values and high-cardinality identifiers are absent; approved bounded operational fields remain. | `appview/internal/observability/video_redaction_test.go`, `app/test/observability/video_secret_scan_test.dart` |
| IT-019 | FR-007, FR-009, FR-010 | AC-007, AC-009 | Verify create-to-Tap-to-read convergence for native video. | Fake PDS write followed by matching Tap event and canonical API read | Publish completed video and ingest emitted record | PDS record has standard embed and eventual AppView response has the same normalized video identity once. | `appview/internal/app/video_post_flow_integration_test.go` |
| IT-020 | FR-001, FR-004, FR-029, FR-030, NFR-002 | AC-001, AC-004, AC-014, AC-038, AC-039 | Verify operational video metrics use bounded labels. | Recording metrics sinks and authorization/upload/job/verification/playback success, rejection, retry, cancellation, and failure paths | Exercise each path with many user/job identifiers | Required counters/durations change; labels use bounded state/reason categories and never contain DID, job ID, CID, URL, filename, token, or authored text. | `appview/internal/observability/video_metrics_test.go` and `app/test/observability/video_metrics_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Text, image, quote, and external-card create/read behavior | BR-004 | AC-016 | Run existing AppView post request, create, index, response, and Flutter post-model/card suites unchanged alongside video cases. |
| REG-002 | Replies and scheduled posts remain video-free while standard/project immediate posts support video | BR-004, RULE-004 | AC-012, AC-016 | Assert reply and schedule contracts reject/omit video and existing scheduling media behavior remains unchanged. |
| REG-003 | YouTube external-card consent and lazy iframe behavior remain separate | BR-004 | AC-016 | Existing YouTube URL, consent, external-card, and compact-card tests continue to pass; native video never bypasses consent logic or uses the YouTube player. |
| REG-004 | Existing image attachment preparation, upload, alt text, reordering, and draft restoration | BR-004 | AC-016 | Existing composer image provider/uploader/draft suites retain prior assertions and image limits. |
| REG-005 | Posts without `video` remain wire-compatible on feed, profile, saved, search, and detail | BR-004 | AC-016 | Minimal and fully populated pre-video fixtures decode and render without a video player. |
| REG-006 | Business product/event attachment widgets stay photo-only | FR-019, FR-021 | AC-028 | Existing business composer tests plus explicit disabled-capability cases expose no video copy, control, or picker. |
| REG-007 | Invalid Tap records follow existing quarantine/idempotent ingestion behavior | FR-009 | AC-009 | Malformed video does not weaken generic ingestion error, ack, replay, or quarantine guarantees. |
| REG-008 | Existing draft privacy and account isolation remain intact | FR-024, FR-025 | AC-031 | Existing privacy, path, recovery, and account-switch suites pass with additive video descriptors. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | MP4 boundary validation | Minimal valid MP4 fixtures or generated streams at small size, declared 300000000 bytes, 300000001 bytes, 600 seconds, over 600 seconds, unknown duration, MIME mismatch, and deceptive extension | AT-001, AT-002, UT-001, IT-001, IT-012 |
| TD-002 | Video-service state matrix | Direct upload/public-status fixtures for pending, processing, progress/no progress, completed, failed categories, retryable outage, `already_exists`, quota, email, provider, and contradictory blob/error | AT-003, AT-004, UT-004, IT-003, IT-004, IT-013 |
| TD-003 | Identity and authorization | Alice/Bob Craftsky sessions, device IDs, PDS audiences, OAuth/DPoP/service-token sentinels, matching/foreign job DIDs, exact/mismatched blobs, approved and redirect URLs | IT-002, IT-003, IT-005, IT-009, IT-010, IT-017, IT-018 |
| TD-004 | Standard embed records | Existing no-video post plus full/minimal `app.bsky.embed.video` records with CID, MP4 blob, size, alt, aspect ratio, thumbnail, playlist identity, captions, malformed variants, create/update/delete events | IT-006, IT-007, IT-008, IT-019, REG-001, REG-007 |
| TD-005 | HLS playback | Deterministic valid master/media playlist, missing/expired/unsupported URLs, thumbnail present/absent, valid/invalid WebVTT, injected controller events | AT-008, AT-009, AT-010, AT-011, IT-015, MAN-001 |
| TD-006 | Unicode alt text | Empty, explicit skip, ASCII at 1000/1001 graphemes, combining characters, flags, emoji with skin tone and zero-width joiners | AT-011, UT-002 |
| TD-007 | Draft storage | Standard/project drafts, source streams, poster bytes, exact/over 1000000000-byte account totals, second account, replacement, disk failure, interrupted atomic write, corrupt source | AT-006, UT-007, UT-015, IT-011, REG-008 |
| TD-008 | Media combinations | Video alone, 1-4 photos, video plus photos/quote/external, duplicate video, existing selected media of each type | AT-007, UT-003, IT-005 |
| TD-009 | Sensitive diagnostics | Synthetic Craftsky/OAuth/service tokens, DPoP keys, authorization headers, byte sentinels, filenames, local paths, DID/job/CID/URLs, alt/caption text | UT-012, IT-009, IT-010, IT-018 |
| TD-010 | Platform matrix | Android, iOS, macOS, Windows, Linux, and supported web browsers with one known-good Bluesky-compatible HLS stream, URI WebVTT track, and eligible upload account | MAN-001, MAN-002, MAN-003, MAN-005 |

Large-file tests should use sparse files, bounded generated readers, counting
allocators/readers, or declared content length where semantics permit. The test
suite must not check a 300 MB fixture into Git or repeatedly allocate 300 MB in
ordinary unit tests.

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-014, FR-015, FR-018 | AC-010, AC-017, AC-018, AC-021 | Real HLS and captions on each supported target | Open a known video post, play, seek, mute, full-screen, enable captions, background, and scroll away on each declared target. | Playback and captions work where supported; lifecycle pauses media and no unexpected audio remains. |
| MAN-002 | NFR-004 | AC-023 | Visual stability under realistic network delay | Throttle HLS/thumbnail loading and move cards among idle, buffering, playing, full-screen return, and error at narrow/wide sizes and large text. | Aspect ratio/fallback remains bounded with no disruptive feed jump or overflow. |
| MAN-003 | FR-017 | AC-020 | Screen-reader control and alt-text quality check | Navigate a video post with VoiceOver/TalkBack and keyboard where supported. | Description, controls, time, caption, error, and progress state are announced in a usable order without duplicate labels. |
| MAN-004 | NFR-005 | AC-043 | Device memory smoke check for maximum-size draft | On a representative constrained device, save, reopen, verify, and begin publication of a near-300 MB local draft while observing memory. | No second full-file memory spike, crash, or unresponsive UI occurs. |
| MAN-005 | BR-001, BR-003, FR-001, FR-004, FR-005 | AC-001, AC-002, AC-004, AC-007 | Live direct-upload and verification smoke check | With an eligible test account on native and supported web, request authorization, upload a small MP4, poll completion, publish, and inspect browser/network behavior without recording credentials. | CORS permits the exact upload, redirects are not followed, polling is tokenless, AppView verifies the result, and the authored post converges through Tap. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Bluesky HLS/CDN patterns may change after the verified contract snapshot. | FR-010, FR-011 | The standard AppView uses configurable templates and current public responses use `video.bsky.app/watch/{encodedDid}/{encodedCid}`; this is an operational dependency rather than an unresolved design. | Resolved for coding: use configurable playlist/thumbnail templates with current Bluesky defaults, contract fixtures, soft failure, and no raw fallback. Reverify before release. |
| GAP-002 | `media_kit` behavior and binary impact require real-platform evidence. | FR-014, FR-015, FR-018, NFR-004 | Package research confirms APIs and six-platform declarations, but fake-controller tests cannot prove decoders, browser HLS/CORS, full screen, WebVTT, or release bundle size. | Resolved for coding: use `media_kit`, `media_kit_video`, and `media_kit_libs_video` behind an app-owned adapter on Android, iOS, macOS, Windows, Linux, and web; execute `MAN-001` to `MAN-003` and measure release artifacts. |
| GAP-003 | No always-on live Bluesky video-service/PDS acceptance environment is identified. | BR-001, FR-001, FR-002, FR-005, FR-011 | Live authorization, direct upload CORS, eligibility, quotas, completion verification, and CDN behavior are nondeterministic and require credentials. | Keep deterministic fake-service tests mandatory; add an opt-in secret-safe smoke suite once an eligible test account exists. |
| GAP-004 | Automated peak-memory assertions are platform/toolchain sensitive. | FR-003, NFR-005 | Allocation counters cannot prove total process memory uniformly across Flutter targets. | Require stream/counting-adapter tests and cancellation probes; supplement with direct-upload and draft device memory checks in `MAN-004`. |
| GAP-005 | URI WebVTT support may vary by `media_kit` backend and browser. | FR-018 | `SubtitleTrack.uri` and WebVTT-over-HLS are documented, but CDN CORS and platform backend behavior still require real playback evidence. | Resolved for coding: use selectable `SubtitleTrack.uri` tracks, automate the adapter contract, run the six-target caption smoke matrix, and document any target exception before release. |
| GAP-006 | A service JWT is bearer-bound rather than DPoP-bound at the video endpoint. | BR-003, FR-002, FR-003, NFR-003 | Possession permits replay until expiry even though scope and audience are narrow. | Require memory-only handling, exact-origin/path attachment, disabled redirects, shortest accepted expiry up to 30 minutes, and security review of every diagnostic/storage boundary. |

No gap removes Must-level test coverage or blocks coding planning. `GAP-001`,
`GAP-002`, `GAP-005`, and `GAP-006` have selected coding directions with mandatory
release verification; all gaps must be dispositioned before implementation is
declared complete.

## 10. Out Of Scope

- Multiple videos or mixed photo/video posts beyond rejection tests.
- Video replies, scheduled video posts, imported Instagram video, business
  product/event video, profile media, and camera recording beyond capability and
  regression assertions.
- Caption upload, caption generation, transcription, trimming, filtering,
  compression, conversion, scanning, and Craftsky transcoding.
- Raw PDS MP4 playback or any upload destination other than the configured
  Bluesky video endpoint.
- Service load, CDN throughput, and transcoding-quality benchmarking owned by
  Bluesky; Craftsky tests its direct transport and user-visible behavior.
- Destructive cleanup of unreferenced PDS blobs. Tests must assert Craftsky does
  not issue such deletion.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill:
  `docs/changes/2026-09-03-video-posts/`
- Risk level: High; explicit approval is required before implementation.
- Recommended first failing test for implementation: `UT-003`, extending
  `appview/internal/api/post_request_test.go` to accept exactly one video
  job/blob proof while rejecting mixed media, incomplete/foreign jobs, and blob
  mismatches. This establishes the trust boundary before transport or UI work.
- Suggested test order for implementation:
  1. `UT-003`, `IT-006`, and `IT-005`: lexicon, generated types, post contract,
     and PDS write invariant.
  2. `UT-001`, `UT-004`, `UT-011` through `UT-013`, `IT-001` through `IT-005`,
     `IT-009`, `IT-010`, `IT-012`, and `IT-017` through `IT-020`: service-JWT
     handoff, direct streaming/polling, exact-origin containment, ephemeral
     state, completion verification, redaction, and metrics.
  3. `IT-007`, `UT-005`, `IT-008`, and `IT-019`: indexing and canonical HLS
     read contract.
  4. `UT-008`, `IT-004`, `AT-002`, `AT-003`, `AT-004`, and `AT-007`: Flutter
     wire model, local selection, eligibility, progress, and attachments.
  5. `UT-007`, `UT-015`, `IT-011`, and `AT-006`: streaming local drafts and
     quota.
  6. `UT-009`, `UT-013`, `UT-014`, `IT-014`, `AT-001`, and `AT-005`: publish
     orchestration, polling, cancellation, interruption, and retry.
  7. `UT-010`, `UT-016`, `IT-015`, and `AT-008` through `AT-011`: playback,
     lifecycle, failures, surfaces, alt text, and captions.
  8. `UT-012`, `AT-012`, `AT-013`, all `REG-*`, and all `MAN-*`: privacy,
     observability, metered behavior, regression, and platform evidence.
- Commands discovered:
  - `just appview-test-unit`
  - `just lexgen-check`
  - `just test` after `just dev-d`
  - `just appview-check`
  - `just app-test <test path or directory>`
  - `just app-test`
  - `just app-analyze`
  - Focused Go: from `appview/`,
    `GOTOOLCHAIN=go1.26.6 go test ./internal/api ./internal/video ./internal/index`
- Blocking gaps for document review: None.
- Blocking gaps for coding design: None. Configurable Bluesky playback templates
  and the `media_kit` six-platform adapter are selected; platform smoke evidence
  remains required before implementation completion.
