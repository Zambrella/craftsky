# TDD Implementation Plan: Video Posts

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`
- Architecture decisions: `adr/012-ephemeral-video-service-token-handoff.md`, `adr/013-standard-video-embed.md`
- Implementation approval: Granted by the user's 2026-09-03 request to invoke `implement-tdd` and continue this workflow.

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and loop evidence updated after every test ID.
- Do not add an AppView upload/status proxy, persisted remote jobs, a generic blob proxy, raw-PDS playback, or an invented idempotency contract.
- Keep OAuth and DPoP material server-side; keep the upload service JWT ephemeral and exact-destination-bound.

## Test Order
| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-003 | FR-007, RULE-003 | AC-007, AC-024 | Fails because `embed.video` is unknown |
| 2 | IT-006 | FR-008 | AC-008 | Fails because the post union lacks video |
| 3 | UT-004, IT-003, IT-005 | FR-004 through FR-007, RULE-003, RULE-005, RULE-006, BR-001, NFR-001 | AC-004 through AC-007, AC-013, AC-024 | Fails because completion verification and verified creation do not exist |
| 4 | IT-002, IT-013, IT-017 | BR-003, FR-001, FR-002, FR-018, FR-028, NFR-001 | AC-001, AC-002, AC-005, AC-021, AC-036, AC-037 | Fails because purpose-bound auth, limits, caption, and route policies do not exist |
| 5 | UT-001, UT-011, UT-012, UT-013, IT-001, IT-009, IT-010, IT-012, IT-018 | FR-001 through FR-003, FR-013, FR-029, FR-031, NFR-002, NFR-003, RULE-001, RULE-002, BR-003 | AC-002, AC-003, AC-014, AC-015, AC-022, AC-035, AC-038, AC-040 | Fails because isolated streaming transport and ephemeral lifecycle do not exist |
| 6 | IT-007, UT-005, IT-008, IT-019 | BR-002, FR-007, FR-009 through FR-011, FR-018 | AC-007, AC-009 through AC-011, AC-021 | Fails because video indexing, playback URLs, captions, and hydration do not exist |
| 7 | UT-008, AT-002, AT-003, AT-007, AT-013 | FR-010, FR-012, FR-019 through FR-021, FR-028, RULE-003 through RULE-006, RULE-008, RULE-009, RULE-011 | AC-009, AC-010, AC-012, AC-024, AC-026 through AC-028, AC-036, AC-037, AC-042, AC-044, AC-045 | Fails because Flutter video models and capability-aware selection do not exist |
| 8 | UT-007, UT-015, IT-011, AT-006 | FR-024, FR-025, NFR-005, RULE-004 | AC-031, AC-032, AC-043 | Fails because streamed video draft storage and quota do not exist |
| 9 | UT-009, UT-014, AT-001, AT-004, AT-005 | BR-001, FR-001, FR-004, FR-012, FR-022, FR-023, FR-026, FR-027, FR-030, FR-031, RULE-005, RULE-006 | AC-001, AC-004, AC-005, AC-007, AC-029, AC-030, AC-033 through AC-035, AC-039 | Fails because publication orchestration does not exist |
| 10 | UT-010, UT-016, IT-014, IT-015, AT-008, AT-009, AT-010, AT-011 | BR-002, FR-010, FR-011, FR-014 through FR-018, FR-023, FR-026, FR-027, FR-032, NFR-004, RULE-010 | AC-010, AC-011, AC-017 through AC-023, AC-030, AC-033, AC-034, AC-041 | Fails because native playback and lifecycle handling do not exist |
| 11 | IT-020, AT-012, REG-001 through REG-008 | BR-004, FR-001, FR-004, FR-009, FR-019, FR-021, FR-024, FR-025, FR-029, FR-030, NFR-002, RULE-004, RULE-007 | AC-001, AC-004, AC-009, AC-012, AC-014, AC-016, AC-025, AC-028, AC-031, AC-038, AC-039 | Fails until metrics and explicit regressions are covered |
| 12 | MAN-001 through MAN-005 | Release-gate requirements listed in `02-acceptance-tests.md` | AC-001, AC-002, AC-004, AC-007, AC-010, AC-017, AC-018, AC-020, AC-021, AC-023, AC-043 | Environmental evidence unavailable until real platform/service checks |

Within grouped rows, each test ID is executed and recorded separately before moving to the next ID. Repeated test IDs in the coding plan are implemented at their first dependency-correct position and rerun later as regression evidence.

## Implementation Steps
### Step 1: UT-003
- Write failing test: Decode and validate exactly one `embed.video` proof; reject malformed blob data, overlong alt text, invalid aspect ratio, replies, and coexistence with images, quote, or external embeds.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run 'Test.*Post.*Video'` from `appview/`.
- Confirmed failure: Build failed because `EmbedRequest` had no `Video` field.
- Implement: Added the strict `embed.video` proof shape and local validation for job ID, canonical blob CID, blob type, MP4 MIME, 300,000,000-byte bound, 1,000-grapheme alt text, positive aspect ratio, reply exclusion, and native-media mutual exclusion.
- Run command: Focused command passed; `GOTOOLCHAIN=go1.26.6 go test ./internal/api` also passed.
- Refactor: Kept video validation in one request-boundary helper; no remote dependencies added.
- Notes: Remote status, owner, and blob equality checks remain in UT-004/IT-003/IT-005. Top-level legacy `video` remains rejected; the supported shape is `embed.video`.

### Step 2: IT-006
- Write failing test: Added minimal/full standard-video JSON and CBOR round trips plus schema open-union inspection.
- Run command: `just lexgen-check` plus focused generated contract tests.
- Confirmed failure: Schema assertion reported refs `[#quoteEmbed app.bsky.embed.external]`, missing `app.bsky.embed.video`.
- Implement: Invoked `atproto-lexicon`, added only the accepted video union variant and Indigo `video.json`/`defs.json` generator inputs, documented reuse, and ran `just lexgen`.
- Run command: Focused generated contract/compatibility tests passed; `just lexgen-check` passed.
- Refactor: None.
- Notes: Generated files remain generator-owned. Existing no-embed, quote, and external contracts remain covered.

### Steps 3-12
- Write failing test: Execute one listed test ID at a time in the approved order.
- Run command: Use the focused target in `02-acceptance-tests.md` and `04-coding-plan.md`.
- Confirmed failure: Pending.
- Implement: Minimum linked behavior only.
- Run command: Focused test, then nearby tests for shared behavior.
- Refactor: Only while green.
- Notes: Add a per-test outcome below as each loop completes.

### Step 3a: UT-004
- Write failing test: Added AppView owner/job/blob completion verification matrix and Flutter service-state normalization matrix.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/video -run TestCompletionVerifier`; `flutter test test/feed/models/video_service_result_test.dart`.
- Confirmed failure: AppView had no `internal/video` package; Flutter had no video service result model.
- Implement: Added a narrow completion verifier with generic bounded errors and an immutable Flutter result parser that discards raw provider messages after categorization.
- Run command: Both focused suites pass; nearby AppView API, generated lexicon, and video packages pass.
- Refactor: Kept HTTP transport outside the verifier and package-specific service strings outside UI state.
- Notes: The verifier preserves cancellation/deadline errors, accepts `already_exists` only with a blob, and otherwise requires completed state, exact job, owner DID, CID, MIME, and size.

### Step 3b: IT-003
- Write failing test: Added create-handler cases for verified, rejected, and unavailable completion proofs with PDS-effect assertions.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run TestCreatePost_VideoProofVerifiedBeforePDSWrite`.
- Confirmed failure: `CreatePostHandler` had no verifier dependency.
- Implement: Added a narrow optional verifier dependency, invoked it immediately after local validation, and failed closed before constructing a PDS executor.
- Run command: Focused test passed.
- Refactor: Passed the verifier-owned blob separately into record/synthetic-row construction rather than mutating the untrusted request.
- Notes: Production dependency wiring remains in the service-auth/routes sequence.

### Step 3c: IT-005
- Write failing test: Added the exact standard embed/PDS-record assertion after IT-003 was green.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run 'TestCreatePost_(VideoProofVerifiedBeforePDSWrite|VerifiedVideoWritesStandardEmbed)'`.
- Confirmed failure: No separate red was available because the minimum IT-003 implementation necessarily constructed the verified video record to complete its success path. This dependency overlap was anticipated by coding-plan row 3, which groups IT-003 and IT-005.
- Implement: No additional behavior; the test confirms the verifier-returned blob is used, optional alt/aspect ratio are written, and `jobId` is excluded.
- Run command: Focused tests passed.
- Refactor: None.
- Notes: Detailed malformed/mixed-media cases remain in UT-003; all rejected paths write no record.

### Step 4a: IT-002
- Write failing test: Added handler tests for exact two-field authorization output, authenticated DID/session forwarding, `Cache-Control: no-store`, and secret-safe failures.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/api -run TestVideoUploadAuthorizationHandler`.
- Confirmed failure: The upload authorization value and API handler did not exist.
- Implement: Added the narrow authorization value and handler boundary. The handler returns only `token` and RFC 3339 `expiresAt` and never includes issuer error detail.
- Run command: Focused handler and issuer tests pass; `GOTOOLCHAIN=go1.26.6 go test ./internal/video ./internal/api ./internal/routes ./internal/app` passes.
- Refactor: Moved production construction into `deps_video.go` to preserve the repository's feature-composition boundary.
- Notes: The concrete issuer runs inside the active-session coordinator, derives `did:web` from the validated PDS host, fixes `lxm` to `com.atproto.repo.uploadBlob`, caps expiry at 30 minutes, and fails with one bounded error. The route is registered as authenticated, device-bound, current-member, upload-rate-class, no-body, and `no-store`; only the service JWT and expiry cross the API boundary.

### Step 4b: IT-013
- Write failing test: Added coordinated limits authorization/service tests, normalization fixtures, exact outbound request assertions, API response/error tests, and exact route policy/registration tests.
- Run command: Focused tests first failed because the limits service, handler, and route policy did not exist.
- Confirmed failure: The video package lacked `NewUploadLimitsService`; the API package lacked `VideoUploadLimitsHandler`; `mustPolicy` panicked for `GET /v1/blobs/videos/limits`.
- Implement: Added a one-minute internal service authorization bound to `did:web:video.bsky.app` and `app.bsky.video.getUploadLimits`, a non-redirecting bounded service request, stable eligibility normalization, camelCase API response, and current-member read route.
- Run command: `GOTOOLCHAIN=go1.26.6 go test ./internal/video ./internal/api ./internal/routes ./internal/app` passes; `git diff --check` passes.
- Refactor: Kept provider token/message detail out of the API model and composed both video capabilities in `deps_video.go`.
- Notes: Eligible, email-unverified, provider-unsupported, quota-constrained, unknown, and upstream-unavailable AppView paths are covered. Step 7 completed the Flutter API client, picker gate, and publish-time recheck.

### Step 5a: UT-001
- Write failing test: Added exact byte/duration boundary, unknown-duration, declared-MIME mismatch, and deceptive-extension/content cases.
- Run command: `flutter test test/feed/media/video_source_validator_test.dart`.
- Confirmed failure: The validator library, result type, and rejection categories did not exist.
- Implement: Added local source validation requiring declared `video/mp4` and recognized MP4 header magic rather than trusting an extension, plus at most 300,000,000 bytes and at most 600 known seconds. A null duration remains eligible for authoritative service validation.
- Run command: Focused Flutter test passes; Dart MCP analysis reports no errors.
- Refactor: Kept validation synchronous and metadata-only; it does not read or retain the source body.
- Notes: Actual streamed-byte enforcement remains in IT-001/IT-012.

### Steps 5b-5i: Flutter Transport And Attempt Lifecycle
- Write failing tests: Added focused credential lifecycle, retry-attempt, direct streaming, persistence-absence, destination containment, resource-bound, and secret-scan tests for UT-011 through UT-013, IT-001, IT-009, IT-010, IT-012, and IT-018.
- Confirmed failure: Dedicated external transport, ephemeral publication ownership, polling recovery, and diagnostics boundaries were absent.
- Implement: Added immutable video-service configuration, isolated no-Craftsky-interceptor transport, exact HTTPS origin/path checks, redirect rejection, counted source streaming, tokenless bounded polling, fresh-authorization retries, lifecycle/account cancellation, and bounded diagnostics.
- Run command: Focused Flutter video suites pass; final `flutter test --concurrency=1` passes all 2,005 tests and `flutter analyze` is clean.
- Refactor: Removed unused test-only attempt helpers and exercised production coordinator behavior directly.

### Step 6: IT-007, UT-005, IT-008, IT-019
- Write failing tests: Added standard-video index validation/replay tests, playback-template tests, canonical response/caption tests, and production create-to-Tap-to-read convergence.
- Confirmed failure: Production status verification, video metadata validation, playback hydration, constrained caption delivery, and verifier route wiring were absent.
- Implement: Added a bounded public status client, production completion-verifier DI, index validation without video columns, configurable HLS/thumbnail URL construction, shared full-post hydration, exact caption-membership proof and bounded WebVTT fetch, and convergence through the real dispatcher/store path.
- Run command: Focused suites and `GOTOOLCHAIN=go1.26.6 go test ./...` pass; `just lexgen-check` and `just appview-check` pass.
- Refactor: Kept full records authoritative and omitted malformed playable metadata without failing the surrounding post.

### Step 7: UT-008, IT-013, AT-002, AT-003, AT-007, AT-013
- Write failing tests: Added canonical Flutter model, API limits, picker eligibility, composer attachment/accessibility, project-composer, and metered-network tests.
- Confirmed failure: Flutter had no canonical video proof/view, capability-aware picker, or video composer state.
- Implement: Added API models/parsing, pre-picker and publish-time limits checks, local source/poster state, grapheme-bounded alt text, explicit media exclusivity, standard/project composer support, actionable denial/failure copy, and fresh Retry.
- Run command: Focused model/provider/widget tests and the full Flutter suite pass.

### Step 8: UT-007, UT-015, IT-011, AT-006
- Write failing tests: Added streamed draft store, descriptor compatibility, account quota, repository recovery, and project/standard snapshot tests.
- Confirmed failure: Existing image-only byte-array draft paths could not safely persist maximum-size video sources.
- Implement: Added streamed copy/hash/verification, source/poster descriptors, backward-compatible manifests, atomic replacement/rollback, account-scoped 1,000,000,000-byte source quota, and restore paths that avoid a full source copy.
- Run command: Focused draft suites and full Flutter suite pass.

### Step 9: UT-009, UT-014, AT-001, AT-004, AT-005
- Write failing tests: Added submission state, poll scheduling, publication coordination, cancellation, actionable failure, and retry coverage through production composer paths.
- Confirmed failure: No staged local-to-upload-to-processing-to-publication state machine existed.
- Implement: Added bounded polling with `Retry-After`, staged progress, upload/processing-only cancellation, retained local media on safe failure, lifecycle interruption handling, and fresh full-sequence retries.
- Run command: Focused coordinator/composer tests and full Flutter suite pass.

### Step 10: UT-010, UT-016, IT-014, IT-015, AT-008, AT-009, AT-010, AT-011
- Write failing tests: Added controller, layout, failure, lifecycle, captions, accessibility, and reusable-surface tests.
- Confirmed failure: Canonical HLS video had no app-owned player abstraction or lifecycle behavior.
- Implement: Added `media_kit` initialization and adapter, HLS-only playback, active-player coordination, pause/dispose on visibility and app lifecycle, URI-backed selectable WebVTT resources with cleanup, full-screen/layout behavior, soft errors, and standard/saved surfaces.
- Run command: Focused player/surface tests, Android/iOS/macOS/web builds, and full Flutter suite pass.

### Step 11: IT-020, AT-012, REG-001 Through REG-008
- Write failing tests: Added bounded video metric/redaction tests, privacy-copy coverage, and explicit existing-media/scheduled/business/Tap/draft regressions.
- Confirmed failure: Video-specific bounded telemetry and explicit cross-feature evidence were absent.
- Implement: Added bounded operation/result/reason metrics with no user/job/media identifiers and retained existing image, external, scheduled, business, ingestion, and account-isolation behavior.
- Run command: `just appview-check`, full Flutter tests, platform host builds, and `git diff --check` pass.
- Notes: MAN-001 through MAN-005 remain environmental release gates.

## Execution Log
| Test ID | Red Evidence | Green Evidence | Implementation / Refactor Notes | Status |
|---|---|---|---|---|
| UT-003 | Missing `EmbedRequest.Video` caused the focused test build to fail | Focused video test and full `internal/api` package pass | Added request-only video proof validation; no PDS effect or remote verification | Complete |
| IT-006 | Schema assertion showed the video ref was absent | Video JSON/CBOR contract tests and `just lexgen-check` pass | Added the approved open-union branch and regenerated `FeedPost_Embed` | Complete |
| UT-004 | Go package and Dart result model were absent | Go verifier matrix and Flutter normalization matrix pass | Added fail-closed verification and bounded actionable client outcomes | Complete |
| IT-003 | Handler constructor lacked a verifier seam | Handler verification matrix passes | Verification now precedes every PDS effect | Complete |
| IT-005 | Dependency overlap: IT-003's successful path already required record construction | Exact standard embed test passes | Verified blob enters the record; job proof does not | Complete |
| IT-002 | Handler authorization types were absent | Handler, coordinated issuer, route policy/inventory, and app composition suites pass | Purpose-bound authorization is production-wired without exposing OAuth/DPoP material | Complete |
| IT-013 | Limits service, handler, route, and Flutter gate were absent | AppView and Flutter limits/picker/publish-time suites pass | Credential stays server-side; normalized eligibility gates both selection and publication | Complete |
| UT-001 | Validator library and types were absent | Focused Flutter boundary test passes; analyzer reports no errors | Metadata-only validation accepts exact limits and unknown duration, rejects predictable invalid sources | Complete |
| UT-011, UT-012, UT-013 | Ephemeral lifecycle, redaction, and retry state were absent | Production coordinator lifecycle/retry and secret-scan suites pass | Secrets/jobs clear at every boundary; retries obtain fresh authorization | Complete |
| IT-001, IT-009, IT-010, IT-012, IT-018 | No isolated streaming client or live-path containment evidence | Direct transport, persistence, resource-bound, destination, and diagnostics suites pass | Dedicated external client streams without full buffering or Craftsky headers | Complete |
| IT-007, UT-005, IT-008, IT-019 | No video index/hydration/caption/convergence path | Index, playback, caption, response, and production convergence suites pass | Standard records remain authoritative; canonical HLS view converges once | Complete |
| UT-008, AT-002, AT-003, AT-007, AT-013 | No video-aware composer model or eligibility UI | Standard/project composer, picker, limits, alt, accessibility, and metered suites pass | Capability and media exclusivity are explicit; failures are actionable | Complete |
| UT-007, UT-015, IT-011, AT-006 | Draft repository was image-only and byte-buffered | Stream store, descriptor, quota, recovery, and snapshot suites pass | Video source/poster persistence is atomic, bounded, and account-scoped | Complete |
| UT-009, UT-014, AT-001, AT-004, AT-005 | Publication state machine was absent | Coordinator, poller, overlay, cancel, failure, and retry suites pass | Local media survives safe failure; terminal remote state is never resumed | Complete |
| UT-010, UT-016, IT-014, IT-015, AT-008, AT-009, AT-010, AT-011 | Native player/controller and caption lifecycle were absent | Controller, player, URI-caption, layout, lifecycle, accessibility, and surface suites pass | HLS-only playback fails soft and coordinates active audio | Complete |
| IT-017, IT-020, AT-012, REG-001 through REG-008 | Route/metric/privacy/regression evidence was incomplete | Exact route inventory, bounded metrics, privacy copy, broad Go/Flutter regressions pass | No proxy/status routes or sensitive metric dimensions added | Complete |

## Manual And Environmental Gates
- MAN-001 through MAN-005 require real supported platforms and an eligible live account. If those resources are unavailable, retain them as explicit blocked release evidence rather than claiming completion.

## Completion Checklist
- [x] All Must requirements covered by tests or documented environmental gaps
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Documentation and generated artifacts updated
- [x] Manual/environmental gaps recorded
- [x] Implementation review updated after required fixes
