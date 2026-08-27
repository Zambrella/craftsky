# TDD Implementation Plan: Link Previews

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and red/green evidence updated after every loop.
- Use the fixed contract in requirements and the coding plan when a test description is inconsistent.
- Do not use public DNS or public websites in automated tests.
- Do not create a stage commit unless the user explicitly enables commits.

## Execution Notes

- The approved High-risk security and lexicon work may proceed under the explicit approval recorded in all four input documents and the user's implementation request.
- `04-coding-plan.md` controls the phase order. Its lexicon contract phase precedes the secure transport tests recommended in `02-acceptance-tests.md`.
- After the lexicon phase, `UT-009` is the first outbound-boundary test, as recommended by `03-document-review.md`.
- The fixed fragment contract controls `UT-011`: AppView removes the input fragment and returns a fragment only when a redirect supplies one. Flutter applies an eligible source fragment later.
- The repository currently resolves `golang.org/x/net v0.58.0`; implementation will promote that resolved compatible version instead of downgrading to the plan's stale `v0.55.0` reference.
- `just lexgen-check` is a clean-tree drift guard and cannot pass against expected uncommitted generated changes. Generation stability will be checked by rerunning `just lexgen` and inspecting for no additional generated diff; the clean-tree guard remains a pre-merge check.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-008 | FR-006 | AC-007 | Fails: external union variant absent |
| 2 | REG-003 | FR-006 | AC-007 | Passes before change; remains green with expanded fixtures |
| 3 | UT-009 | FR-005, NFR-002 | AC-004 | Fails: safe URL policy absent |
| 4 | UT-010 | FR-005, NFR-002 | AC-004, AC-005 | Fails: address policy absent |
| 5 | UT-011 | FR-005, RULE-004 | AC-001, AC-005 | Fails: redirect fetcher absent |
| 6 | IT-002 | FR-005, NFR-002 | AC-004, AC-005 | Fails: pinned transport absent |
| 7 | IT-003 | FR-005, NFR-002, RULE-004 | AC-001, AC-005 | Fails: per-hop revalidation absent |
| 8 | IT-019 | FR-005, FR-018, NFR-002, RULE-006 | AC-005, AC-017 | Fails: direct safe transport absent |
| 9 | UT-004 | FR-002 | AC-002 | Fails: metadata extraction absent |
| 10 | UT-005 | FR-002 | AC-002 | Fails: Unicode clamp absent |
| 11 | UT-006 | FR-002, FR-004 | AC-006, AC-023 | Fails: bounded head decoder absent |
| 12 | UT-007 | FR-003 | AC-003 | Fails: image candidate extraction absent |
| 13 | UT-008 | FR-003 | AC-003 | Fails: image validation absent |
| 14 | IT-004 | FR-002, FR-003, FR-004, NFR-001 | AC-002, AC-003, AC-006, AC-023 | Fails: fetch pipeline incomplete |
| 15 | UT-016 | FR-018, RULE-006 | AC-017 | Fails: preview config absent |
| 16 | UT-017 | FR-015 | AC-015 | Fails: preview rate class absent |
| 17 | UT-012 | FR-001, FR-007, RULE-001, RULE-002 | AC-008, AC-014, AC-017 | Fails: preview/external request shapes absent |
| 18 | IT-001 | BR-001, FR-001, RULE-004 | AC-001 | Fails: preview handler absent |
| 19 | IT-005 | FR-001, FR-018, RULE-003 | AC-017 | Fails: route absent |
| 20 | IT-006 | FR-015 | AC-015 | Fails: preview middleware budgets absent |
| 21 | AT-007 | FR-001, FR-005, FR-015, FR-018, NFR-002, RULE-003, RULE-006 | AC-004, AC-005, AC-015, AC-017 | Fails: endpoint stack incomplete |
| 22 | REG-007 | FR-015 | AC-015 | Passes before change; add quota-isolation coverage |
| 23 | UT-013 | FR-008, FR-019, RULE-002 | AC-009, AC-016, AC-020, AC-021 | Fails: external response shaping absent |
| 24 | UT-014 | FR-001, FR-007, FR-008 | AC-001, AC-008, AC-009 | Fails: Flutter/server external models absent |
| 25 | IT-007 | FR-007, RULE-001, RULE-002, RULE-005 | AC-008, AC-014 | Fails: external writes absent |
| 26 | IT-009 | FR-008 | AC-009, AC-021 | Fails: raw embed projection absent |
| 27 | IT-010 | FR-019, RULE-002 | AC-020 | Fails: images-win shaping absent |
| 28 | IT-011 | FR-008, FR-014 | AC-016 | Fails: quote external hydration absent |
| 29 | AT-008 | FR-019, RULE-002 | AC-020 | Fails: federated external read behavior absent |
| 30 | REG-002 | FR-007, RULE-001, RULE-002 | AC-008, AC-014 | Passes before change; add external conflict cases |
| 31 | REG-004 | FR-008 | AC-009, AC-016, AC-021 | Passes before change; preserve absent-external shape |
| 32 | REG-008 | FR-008, FR-019 | AC-009, AC-020, AC-021 | Passes before change; add raw external lifecycle fixture |
| 33 | UT-001 | FR-009, FR-010, RULE-004 | AC-001, AC-010 | Fails: candidate normalization absent |
| 34 | UT-002 | FR-009 | AC-010 | Fails: candidate derivation absent |
| 35 | UT-003 | FR-009, FR-010, FR-011, NFR-001, RULE-002, RULE-005 | AC-010, AC-011, AC-014 | Fails: preview controller absent |
| 36 | IT-012 | FR-009, FR-010, FR-011, FR-015, NFR-001, RULE-004, RULE-005 | AC-010, AC-011, AC-015 | Fails: controller/API integration absent |
| 37 | AT-001 | BR-001, FR-009, FR-010, NFR-001, NFR-003, RULE-004, RULE-005 | AC-010 | Fails: composer carousel absent |
| 38 | AT-002 | FR-011, RULE-001, RULE-002 | AC-011, AC-014 | Fails: composer suppression/dismissal absent |
| 39 | IT-020 | NFR-003 | AC-010, AC-013 | Fails: preview network boundary not encoded |
| 40 | IT-013 | FR-012 | AC-012 | Fails: external submission absent |
| 41 | AT-003 | FR-007, FR-012 | AC-008, AC-012, AC-014 | Fails: composer external submission absent |
| 42 | REG-001 | FR-012 | AC-012 | Passes before change; preserve ordinary facets |
| 43 | UT-018 | FR-016 | AC-018 | Fails: scheduled external payload absent |
| 44 | IT-015 | FR-016 | AC-018 | Fails: external thumbnail staging absent |
| 45 | IT-016 | FR-016 | AC-018 | Fails: frozen external publication absent |
| 46 | AT-005 | FR-016 | AC-018 | Fails: scheduled external flow absent |
| 47 | REG-005 | FR-016 | AC-018 | Passes before change; preserve scheduled behavior |
| 48 | UT-019 | FR-017 | AC-019 | Passes before change; add explicit exclusion assertions |
| 49 | IT-017 | FR-017 | AC-019 | Fails: reopened preview session absent |
| 50 | AT-006 | FR-017 | AC-019 | Fails: draft/composer integration absent |
| 51 | REG-006 | FR-017 | AC-019 | Passes before change; preserve draft schema |
| 52 | UT-020 | NFR-004 | AC-022 | Fails: bounded preview telemetry absent |
| 53 | IT-018 | NFR-004 | AC-022 | Fails: privacy coverage incomplete |
| 54 | AT-009 | NFR-004 | AC-022 | Fails: cross-pillar privacy acceptance incomplete |
| 55 | UT-015 | FR-013 | AC-013 | Fails: external card widget absent |
| 56 | IT-014 | BR-001, FR-013, NFR-003 | AC-013 | Fails: full-surface rendering absent |
| 57 | AT-004 | BR-001, FR-008, FR-013, FR-014, NFR-003 | AC-009, AC-013, AC-016, AC-021 | Fails: full/compact rendering incomplete |

## Implementation Steps

## Execution Log

### Step 1: IT-008

- Write failing test: Added `TestFeedPostExternalEmbedContract` for the open source union and generated JSON/CBOR dispatch.
- Run command: `go test ./internal/lexicon/craftsky -run '^TestFeedPostExternalEmbedContract$' -count=1`.
- Confirmed failure: `main.embed.refs` contained only `#quoteEmbed`.
- Implement: Added ADR 009 with completed lexicon-review evidence, referenced `app.bsky.embed.external` from the open union, added the exact pinned external schema to `just lexgen`, and regenerated.
- Run command: The same focused test passes.
- Refactor: None required.
- Notes: Generated output uses Indigo's `*bsky.EmbedExternal`; no local duplicate type was introduced.

### Step 2: REG-003

- Write failing test: Added `TestFeedPostEmbedCompatibility` for local quote JSON/CBOR dispatch and absent-embed omission.
- Run command: `go test ./internal/lexicon/craftsky -run '^TestFeedPostEmbedCompatibility$' -count=1`.
- Confirmed failure: Not expected; this is an additive-regression guard and passed against the regenerated union.
- Implement: No additional production change required.
- Run command: The focused test passes.
- Refactor: None required.
- Notes: Existing quote and ordinary post behavior is preserved.

### Step 3: UT-009

- Write failing test: Added `TestValidateURLSyntax` for schemes, credentials, ports, malformed hosts, public/non-public literals, normalization, and fragment removal.
- Run command: `go test ./internal/linkpreview -run '^TestValidateURLSyntax$' -count=1`.
- Confirmed failure: `ValidateURL` and `ErrNotAllowed` were absent.
- Implement: Added the pre-resolution HTTP(S) syntax and literal policy with destination-free rejection.
- Run command: The same focused test passes after `gofmt`.
- Refactor: None required.
- Notes: Complete special-use address classification is intentionally deferred to the next test loop.

### Step 4: UT-010

- Write failing test: Added `TestValidateAddresses` for public, special-use, mapped, empty, invalid, and mixed answer sets.
- Run command: `go test ./internal/linkpreview -run '^TestValidateAddresses$' -count=1`.
- Confirmed failure: `ValidateAddresses` was absent.
- Implement: Added explicit IPv4/IPv6 special-use prefix tables and all-or-nothing retained answer validation.
- Run command: The focused test and `UT-009` both pass after `gofmt`.
- Refactor: Moved public-address classification out of `transport.go` into `addresses.go` while green.
- Notes: Returned addresses are copied for later pinned dialing.

### Step 5: UT-011

- Write failing test: Added `TestFetcherRedirectPolicyAndFragments` for zero/five/six hops, forbidden next hops, and fragment rules.
- Run command: `go test ./internal/linkpreview -run '^TestFetcherRedirectPolicyAndFragments$' -count=1`.
- Confirmed failure: `NewFetcher` was absent.
- Implement: Added an injected resolver/doer fetcher with manual redirects, per-hop complete-answer validation, body closure, and a five-redirect cap.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: The implementation follows the fixed contract by discarding input fragments and retaining only redirect-supplied fragments.

### Step 6: IT-002

- Write failing test: Added `TestPinnedTransportDialsValidatedAddress` using a real `net/http` exchange over `net.Pipe`.
- Run command: `go test ./internal/linkpreview -run '^TestPinnedTransportDialsValidatedAddress$' -count=1`.
- Confirmed failure: `NewPinnedTransport` was absent.
- Implement: Added a bounded direct transport with `Proxy: nil`, per-connection complete-answer validation, and numeric-IP dialing.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: HTTP Host remains the original hostname while the socket destination is pinned.

### Step 7: IT-003

- Write failing test: Added `TestPinnedTransportRevalidatesDNSAndRedirects` for rebinding and public-to-private redirect attempts.
- Run command: `go test ./internal/linkpreview -run '^TestPinnedTransportRevalidatesDNSAndRedirects$' -count=1`.
- Confirmed failure: Not expected; the preceding pinned-transport and redirect loops already supplied this integrated behavior.
- Implement: No additional production change required.
- Run command: The focused test passes with zero forbidden dials.
- Refactor: None required.
- Notes: The resolver is called once for admission and again immediately before socket dialing.

### Step 8: IT-019

- Write failing test: Added `TestPinnedTransportIgnoresProxyAndForwardsOnlyFixedHeaders` with proxy canaries and captured wire headers.
- Run command: `go test ./internal/linkpreview -run '^TestPinnedTransportIgnoresProxyAndForwardsOnlyFixedHeaders$' -count=1`.
- Confirmed failure: `net/http` emitted `Go-http-client/1.1` instead of the approved fixed identity.
- Implement: Added fixed page `User-Agent` and `Accept` headers to newly constructed requests.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: `Accept-Language`, credentials, cookies, and referer remain absent; transport proxy selection is nil.

### Step 9: UT-004

- Write failing test: Added `TestExtractMetadataFallbacks` for independent OG/Twitter/HTML/hostname selection and ignored destination metadata.
- Run command: `go test ./internal/linkpreview -run '^TestExtractMetadataFallbacks$' -count=1`.
- Confirmed failure: `ExtractMetadata` was absent.
- Implement: Added token-based extraction over already bounded/decoded head bytes and authoritative final-URL retention.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: Streaming bounds and charset conversion remain assigned to `UT-006`.

### Step 10: UT-005

- Write failing test: Added `TestClampMetadata` for whitespace, combining marks, emoji clusters, byte limits, and overlong single graphemes.
- Run command: `go test ./internal/linkpreview -run '^TestClampMetadata$' -count=1`.
- Confirmed failure: `ClampMetadata` was absent.
- Implement: Added whitespace normalization and dual grapheme/UTF-8 byte clamping with `uniseg`.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: Output remains valid UTF-8 and never splits a grapheme cluster.

### Step 11: UT-006

- Write failing test: Added `TestDecodeHTMLHead` for UTF-8, Windows-1252, ISO-8859-15 XHTML, malformed/unsupported declarations, early head completion, missing close, and raw overflow.
- Run command: `go test ./internal/linkpreview -run '^TestDecodeHTMLHead$' -count=1`.
- Confirmed failure: `DecodeHTMLHead` and `maxPageBytes` were absent; the first implementation also exposed `charset.NewReader` silently falling back for an explicitly unknown HTTP charset.
- Implement: Added explicit response-charset validation, bounded charset conversion, tokenized completed-head stopping, and conservative 2,000,000-byte raw exhaustion rejection.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: Tests assert the source reader never yields more than 2,000,000 raw bytes; no network is used.

### Step 12: UT-007

- Write failing test: Added `TestExtractThumbnailCandidates` for invalid/first-valid bases, relative/cross-origin/query-signed values, OG-before-Twitter priority, duplicates, and four-plus candidates.
- Run command: `go test ./internal/linkpreview -run '^TestExtractThumbnailCandidates$' -count=1`.
- Confirmed failure: `Metadata.ThumbnailCandidates` was absent.
- Implement: Extended the existing tokenizer pass to retain the first valid HTTP(S) base and return at most three distinct resolved OG/Twitter candidates in priority/document order.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: Candidate query values are preserved; no candidate is fetched or resolved through DNS at extraction time.

### Step 13: UT-008

- Write failing test: Added `TestValidateThumbnail` for valid JPEG/PNG/WebP, MIME derivation, corrupt/truncated data, GIF/AVIF/SVG, byte overflow, 8,193-pixel side, and greater-than-16-MP area.
- Run command: `go test ./internal/linkpreview -run '^TestValidateThumbnail$' -count=1`.
- Confirmed failure: `ValidateThumbnail` and thumbnail limits were absent.
- Implement: Added size-first validation, decoded-header geometry checks, full decode/format/dimension agreement, byte-derived MIME, and copied validated bytes.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: None required.
- Notes: The over-dimension and over-area fixtures are rejected after `DecodeConfig`, before full image allocation.

### Step 14: IT-004

- Write failing test: Added `TestServiceFetchPreviewPipeline` for legacy HTML/XHTML decoding, metadata/image fallback, three-attempt cap, MIME/status rejection, raw overflow, page deadline, and metadata survival after image exhaustion.
- Run command: `go test ./internal/linkpreview -run '^TestServiceFetchPreviewPipeline$' -count=1`.
- Confirmed failure: `NewService` and phase-budget orchestration were absent.
- Implement: Added a 10-second total/6-second page service, strict 2xx HTML/XHTML admission, bounded decode/extraction, resource-specific fixed Accept headers, and sequential best-effort image validation.
- Run command: The focused test and complete `internal/linkpreview` package pass.
- Refactor: Reused one content-type predicate after the focused test was green.
- Notes: Page bodies and every acquired image body are closed; image errors and total-budget exhaustion return successful metadata without a thumbnail.

### Step 15: UT-016

- Write failing test: Added `TestLoadConfigLinkPreviewEnablement` for development defaults/override, production explicit opt-in, explicit false, and malformed values.
- Run command: `go test ./internal/app -run '^TestLoadConfigLinkPreviewEnablement$' -count=1`.
- Confirmed failure: `Config.LinkPreviewsEnabled` was absent.
- Implement: Added strict `LINK_PREVIEWS_ENABLED` loading with non-production default true and production default false, plus dev/prod environment documentation.
- Run command: The focused config test and the direct no-proxy/service boundary tests pass.
- Refactor: None required.
- Notes: `NewPinnedTransport` remains unconditional direct egress with `Proxy: nil`; enabling the feature does not select ambient proxy settings.

### Step 16: UT-017

- Write failing test: Added `TestDefaultRateLimitConfigLinkPreview` for 59/60/61 token, 119/120/121 device, exact hourly rollover, and key-type attribution.
- Run command: `go test ./internal/app -run '^TestDefaultRateLimitConfigLinkPreview$' -count=1`.
- Confirmed failure: `middleware.RateClassLinkPreview` was absent.
- Implement: Added the dedicated `link_preview` class with a one-hour, 60/token, 120/device default limit.
- Run command: The focused test and complete `internal/middleware` package pass.
- Refactor: None required.
- Notes: Existing write and upload class definitions are unchanged; joint quota-isolation evidence remains assigned to `REG-007`.

### Step 17: UT-012

- Write failing test: Added `TestDecodeAndValidateLinkPreviewAndExternalRequests` for strict singular preview JSON, 8,192-byte bounds, valid metadata/thumb external, metadata/blob limits, and quote/images/project conflicts.
- Run command: `go test ./internal/api -run '^TestDecodeAndValidateLinkPreviewAndExternalRequests$' -count=1`.
- Confirmed failure: `DecodeLinkPreviewRequest` and the external embed request shape were absent; the first EOF helper also accepted a trailing second JSON value.
- Implement: Added strict preview decoding, external request types and URL/metadata/thumb validation, and mutual-exclusion checks without facet coupling.
- Run command: The focused test and nearby post request decode/validation tests pass.
- Refactor: Extracted numeric blob-size parsing so existing image validation and the external 1,000,000-byte ceiling share one representation-safe check.
- Notes: Validation completes before handlers can invoke preview egress or PDS writes.

### Step 18: IT-001

- Write failing test: Added `TestLinkPreviewHandlerContract` for source-fragment removal, final redirect fragment retention, camelCase response fields, padded RFC 4648 base64, thumbnail metadata, and request ID on standard errors.
- Run command: `go test ./internal/api -run '^TestLinkPreviewHandlerContract$' -count=1`.
- Confirmed failure: `api.LinkPreviewHandler` was absent.
- Implement: Added the feature-gated handler, strict decode, fragmentless syntax admission, fixed error taxonomy mapping, and successful response DTO encoding.
- Run command: The focused test and nearby link-preview API tests pass.
- Refactor: None required.
- Notes: Error messages contain no destination data; successful thumbnail bytes use `base64.StdEncoding`.

### Step 19: IT-005

- Write failing test: Added `TestAddRoutesLinkPreviewAdmission` for unconditional registration, wrong method, missing auth/device, disabled production, malformed body, valid admission, standard envelopes, and zero service calls on rejection.
- Run command: `go test ./internal/routes -run '^TestAddRoutesLinkPreviewAdmission$' -count=1`.
- Confirmed failure: Route config/dependency fields and registration were absent.
- Implement: Added the current-member/default-JSON/`link_preview` route policy, dedicated middleware mapping, unconditional capability registration, config/service adapters, and direct safe-service process wiring.
- Run command: The focused route test, route-policy coverage, and focused app config tests pass.
- Refactor: Kept route registration in a narrow capability bundle rather than adding an inline mux registration.
- Notes: Disabled requests reach the handler only after auth/device/rate admission and return 503 before service work.

### Step 20: IT-006

- Write failing test: Added `TestLinkPreviewRateLimitMiddlewareBudgets` for actual token/device limits, 429 envelope, `Retry-After`, suppressed handler work, distinct identities, and hourly reset.
- Run command: `go test ./internal/middleware -run '^TestLinkPreviewRateLimitMiddlewareBudgets$' -count=1`.
- Confirmed failure: Not expected; the generic atomic limiter plus `UT-017`'s class definition already supplied the integrated HTTP behavior.
- Implement: No additional production change required.
- Run command: The focused test passes.
- Refactor: Added a small response assertion helper while green.
- Notes: Full route assignment to this class is covered by `IT-005`; cross-class quota preservation remains `REG-007`.

### Step 21: AT-007

- Write failing test: Added `TestLinkPreviewEndpointSecurityStack` combining current-member route admission, disabled mode, forbidden literal rejection, and dedicated rate exhaustion with service call counts.
- Run command: `go test ./internal/routes -run '^TestLinkPreviewEndpointSecurityStack$' -count=1`.
- Confirmed failure: Not expected; the preceding transport, handler, route, and limiter loops now compose the acceptance behavior.
- Implement: No additional production change required.
- Run command: The focused acceptance test passes.
- Refactor: None required.
- Notes: All fixtures are local fakes/test Postgres; no DNS lookup, external HTTP request, or forbidden socket is attempted.

### Step 22: REG-007

- Write failing test: Added `TestDefaultRateLimitsKeepLinkPreviewQuotaIndependent` against one shared limiter and identical token/device keys.
- Run command: `go test ./internal/app -run '^TestDefaultRateLimitsKeepLinkPreviewQuotaIndependent$' -count=1`.
- Confirmed failure: Not expected; class-prefixed bucket keys already isolate the newly added class.
- Implement: No additional production change required.
- Run command: The focused regression test passes.
- Refactor: None required.
- Notes: Preview exhaustion leaves the first write/upload available, and write traffic leaves preview capacity available.

### Step 23: UT-013

- Write failing test: Added `TestBuildPostResponseExternalEmbed` for metadata-only/thumb full responses, visible compact quote parity, absent/unknown/malformed embeds, synthesized thumbnail URL, and images-win.
- Run command: `go test ./internal/api -run '^TestBuildPostResponseExternalEmbed$' -count=1`.
- Confirmed failure: `PostRow.RawEmbed` and external response fields/types were absent.
- Implement: Added shared standard-external JSON decoding from authoritative raw embed data and attached it to full/compact responses only when decoded top-level images are absent.
- Run command: The focused test and nearby post-response tests pass.
- Refactor: Kept decoding in one shared response helper used by both full and compact views.
- Notes: Unknown or malformed union variants are omitted safely; no migration, external column, or per-row lookup was introduced.

### Step 24: UT-014

- Write failing test: Added Flutter wire-model coverage for valid padded base64 boundaries, malformed/oversized thumbnail bytes, external create encoding, full/compact external decoding, metadata-only cards, and uploaded thumbnails.
- Run command: `flutter test test/feed/data/link_preview_api_client_test.dart`.
- Confirmed failure: Preview/external models, `fetchLinkPreview`, create-request external support, and full/compact response fields were absent.
- Implement: Added bounded preview decoding, external create/read models, preview API client support, create embed conflict assertions, and generated post mappers.
- Run command: The focused test, `test/feed/models/post_test.dart`, `test/feed/data/post_api_client_test.dart`, and focused analyzer run pass.
- Refactor: Reused the existing post mapper and API error path; no additional state or service layer was introduced.
- Notes: Canonical padded RFC 4648 base64 is enforced before decoding, and decoded thumbnails cannot exceed 1,000,000 bytes.

### Step 25: IT-007

- Write failing test: Added handler-level metadata-only/thumbnail cases with exact standard external record assertions and quote/image/project conflict cases with zero-write assertions.
- Run command: `go test ./internal/api -run '^TestCreatePost_ExternalEmbed_WritesStandardShapeAndRejectsConflicts$' -count=1`.
- Confirmed failure: Valid external requests reached the PDS boundary without an `embed` record.
- Implement: Translated create requests to the `app.bsky.embed.external` record shape and retained the same authoritative embed JSON for the immediate 201 response.
- Run command: The focused test and broader create/decode/validation test selection pass.
- Refactor: Centralized standard external record construction so the PDS write and synthetic response cannot drift.
- Notes: External creation has no facet coupling; every conflicting request exits before durable PDS effect creation.

### Step 26: IT-009

- Write failing test: Added real-Postgres metadata-only/thumbnail fixtures covering detail, profile, authored comments, root replies, and timeline reads.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15571/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/api -run '^(TestPostStore_ProjectsExternalEmbedAcrossCanonicalReads|TestTimelineStore_ProjectsExternalEmbedFromAuthoritativeRecord)$' -count=1`.
- Confirmed failure: Canonical rows did not carry the authoritative record's external embed into `PostRow.RawEmbed`.
- Implement: Projected `p.record -> 'embed'` in the shared post select and extended post, timeline, and scored-search scanners in lockstep.
- Run command: The focused integration tests and full `internal/api` package pass against real Postgres.
- Refactor: Kept raw embed selection in `postSelectColumns`; no migration, redundant column, or per-row lookup was added.
- Notes: The shared response helper now receives identical authoritative embed JSON on every post-shaped store surface.

### Step 27: IT-010

- Write failing test: Added a real-Postgres federated images-plus-external lifecycle fixture covering create, identical replay, update, canonical response shaping, and delete.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15571/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/index -run '^TestCraftskyPost_FederatedImagesAndExternalLifecycleUsesImagesWin$' -count=1`.
- Confirmed failure: The first run exposed an invalid test blob fixture; after correcting it to the canonical blob shape, no product failure remained because raw-record persistence and images-win response shaping were already implemented by existing indexing plus `UT-013`.
- Implement: No additional production change required.
- Run command: The focused lifecycle test and full `internal/index` package pass against real Postgres.
- Refactor: None required.
- Notes: The JSONB source record retains both union fields while public reads expose only indexed images; replay preserves `indexed_at`, update replaces the record, and delete converges.

### Step 28: IT-011

- Write failing test: Added real-Postgres compact quote hydration coverage for a thumbnail external record alongside hidden and unavailable targets.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15571/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/api -run '^TestPostStore_QuoteViewRows_ProjectsExternalWithoutChangingVisibility$' -count=1`.
- Confirmed failure: Not expected; `IT-009` had already extended the shared quote-row scanner and `UT-013` had already added compact response shaping.
- Implement: No additional production change required.
- Run command: The focused integration test and related quote-view/external-embed tests pass against real Postgres.
- Refactor: None required.
- Notes: Only visible quotes expose compact external data; hidden and unavailable targets retain their existing state-only shapes.

### Step 29: AT-008

- Write failing test: Reused the completed `IT-010` federated lifecycle fixture together with the `UT-013` canonical/compact response acceptance assertions.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15571/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/index ./internal/api -run '^(TestCraftskyPost_FederatedImagesAndExternalLifecycleUsesImagesWin|TestBuildPostResponseExternalEmbed)$' -count=1`.
- Confirmed failure: Not expected; the preceding index lifecycle and shared response-shaping loops compose the acceptance behavior.
- Implement: No additional production change required.
- Run command: The focused acceptance selection passes against real Postgres.
- Refactor: None required.
- Notes: Tap-originated mixed records remain indexed with both source fields, while full and compact CraftSky response builders expose images and omit external.

### Step 30: REG-002

- Write failing test: Reused strict request and handler conflict cases for external-plus-quote, external-plus-images, and external-plus-project.
- Run command: `go test ./internal/api -run '^(TestDecodeAndValidateLinkPreviewAndExternalRequests|TestCreatePost_ExternalEmbed_WritesStandardShapeAndRejectsConflicts)$' -count=1`.
- Confirmed failure: Not expected; conflict validation was completed in `UT-012` and zero-write handler evidence in `IT-007`.
- Implement: No additional production change required.
- Run command: The focused regression selection passes.
- Refactor: None required.
- Notes: Existing quote, image, and project create behavior remains intact while invalid external combinations stop before PDS work.

### Step 31: REG-004

- Write failing test: Reused full/compact Go omission cases and existing Flutter minimal/absent optional model round trips.
- Run command: `go test ./internal/api -run '^TestBuildPostResponseExternalEmbed$' -count=1` and `flutter test test/feed/models/post_test.dart test/feed/data/link_preview_api_client_test.dart`.
- Confirmed failure: Not expected; external remains optional on both server and client wire models.
- Implement: No additional production change required.
- Run command: Both focused regression selections pass.
- Refactor: None required.
- Notes: Ordinary, unknown-union, malformed-union, and absent-external responses preserve their prior shape without placeholder card data.

### Step 32: REG-008

- Write failing test: Reused the raw external lifecycle fixture added for `IT-010` across create, identical replay, changed-CID update, and delete.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15571/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/index -run '^TestCraftskyPost_FederatedImagesAndExternalLifecycleUsesImagesWin$' -count=1`.
- Confirmed failure: Not expected; the existing CID-guarded upsert and delete paths are independent of external union contents.
- Implement: No additional production change required.
- Run command: The focused regression test passes against real Postgres.
- Refactor: None required.
- Notes: External JSONB does not weaken replay idempotency or create/update/delete convergence.

### Step 33: UT-001

- Write failing test: Added pure Flutter candidate tests for scheme/host casing, default ports, bare domains, preserved path/query, duplicate source fragments, absent fragments, reorder/removal, and redirect-fragment precedence.
- Run command: `flutter test test/feed/composer/link_preview_candidate_test.dart`.
- Confirmed failure: The link-preview candidate value object and normalization/navigation helpers were absent.
- Implement: Added a pure `LinkPreviewCandidate` that separates normalized fragmentless identity/transport URI from occurrence-specific source fragment and final navigation composition.
- Run command: The focused test and focused analyzer run pass.
- Refactor: Kept normalization and navigation in one immutable value object with no provider or network dependency.
- Notes: AppView's final URI remains authoritative; Flutter adds a source fragment only when the final URI has none.

### Step 34: UT-002

- Write failing test: Added shared rich-text tests for unfinished end tokens, whitespace/punctuation completion, normalized duplicate occurrences, text order, and the four-candidate cap.
- Run command: `flutter test test/shared/rich_text/link_preview_candidates_test.dart`.
- Confirmed failure: The explicit completion-boundary predicate and candidate derivation helper were absent.
- Implement: Added a shared facet completion predicate, explicit-port support in link token syntax, and a pure candidate derivation helper using `LinkFacetToken` plus `UT-001` normalization.
- Run command: The focused candidate tests, existing facet-generator tests, `UT-001`, and focused analyzer run pass.
- Refactor: Candidate derivation remains a pure list transform with no controller or network dependency.
- Notes: End-of-text alone never completes a URL; only the first four distinct fragmentless identities are retained in text order.

### Step 35: UT-003

- Write failing test: Added generated Riverpod controller tests for immediate/sequential fetches, success/failure caching, selection/reorder fragments, suppression cancellation/restoration, dismissal/Undo expiry, stale completions, and disposal.
- Run command: `flutter test test/feed/composer/link_preview_controller_test.dart`.
- Confirmed failure: The preview repository, session state, and controller provider were absent.
- Implement: Added an auto-disposed composer/account family with one cancellable request, generation guards, session caches, synchronous state transitions, and AppView-only repository injection.
- Run command: The focused controller test, `UT-001`/`UT-002`, generated provider output, and focused analyzer run pass.
- Refactor: Kept candidate derivation pure and isolated API access behind `LinkPreviewRepository`.
- Notes: Canceled work is not cached; ordinary failures are cached silently; expired Undo cannot clear session dismissal.

### Step 36: IT-012

- Write failing test: Added an account-boundary integration case proving the old provider cancels its fragmentless request and cannot attach a stale completion to the new account.
- Run command: Focused `IT-012` controller test, then `flutter test test/feed/data/link_preview_api_client_test.dart` serially.
- Confirmed failure: Not expected after `UT-003` and `UT-014`; the first parallel Flutter attempt collided on shared iOS ephemeral cleanup and was rerun serially.
- Implement: No additional production change required.
- Run command: Both controller/account and singular API wire tests pass.
- Refactor: None required.
- Notes: Requests contain normalized fragmentless transport URLs; account disposal cancels old work and final/source fragment precedence remains client-only.

### Step 37: AT-001

- Write failing test: Added a localized composer carousel widget test for loading, deterministic position, navigation controls, dismissal, and final-host presentation.
- Run command: `flutter test test/feed/widgets/composer_link_preview_carousel_test.dart`.
- Confirmed failure: Carousel widget and localization keys were absent.
- Implement: Added the localized carousel, generated l10n, and bound composer text/account state to the controller with loading/success selection controls.
- Run command: Carousel/controller tests and the real `AT-001 real composer selection follows URL across edits` widget test pass after focused Riverpod/l10n generation.
- Refactor: Composer reads selected/available values from the controller while candidate and queue logic remain outside the widget.
- Notes: Text updates remain synchronous and no rendering path performs network work directly.

### Step 38: AT-002

- Write failing test: Reused focused image suppression/restoration and Undo-expiry controller scenarios, then added real standard-composer dismissal/snackbar/Undo and quote-suppression tests plus a project-composer no-request/no-card test.
- Run command: Focused controller tests and the concrete `AT-002` widget functions in `scheduled_post_submission_test.dart` and `project_composer_validation_test.dart`.
- Confirmed failure: The first real Undo interaction exposed an off-viewport widget-test hit target; invoking the actual `SnackBarAction.onPressed` callback proved the production action path without changing behavior.
- Implement: Bound image/quote state to controller suppression and added localized dismiss/Undo snackbar behavior in the standard composer.
- Run command: All focused acceptance selections pass.
- Refactor: None required.
- Notes: Suppression retains cache/selection; dismissal is session-wide and cannot resume after Undo expires.

### Step 39: IT-020

- Write failing test: Added an architecture scan over production preview files for direct socket/HTTP/image-fetch APIs and required the AppView `PostApiClient` boundary.
- Run command: Architecture scan plus controller and carousel suites.
- Confirmed failure: Not expected because the controller repository was designed around `postApiClientProvider`.
- Implement: No additional production change required.
- Run command: The architecture and platform-neutral composer/render selections pass without third-party access.
- Refactor: None required.
- Notes: Flutter preview metadata and rendering contain no direct third-party network path.

### Step 40: IT-013

- Write failing test: Added immediate materialization tests for metadata-only and thumbnail-bearing selected previews.
- Run command: Focused `composer_media_uploader_test.dart` selection.
- Confirmed failure: `materializeImmediateExternal` did not exist.
- Implement: Snapshotted the eligible selected preview synchronously in the composer, added guarded thumbnail materialization with ownership/timeout/cancellation/retry-cache behavior, and threaded `CreatePostExternal` through provider/repository/API layers.
- Run command: Focused uploader, API wire, create-provider, composer-facet suites and focused analyzer pass.
- Refactor: Reused the existing prepared-image upload callback and composer-owned successful-upload cache.
- Notes: Metadata-only previews perform no blob upload; failed upload/create leaves the controller session intact for retry.

### Step 41: AT-003

- Write failing test: Added create-provider payload coverage plus real-composer pending-submit/late-result and thumbnail-upload-failure state-retention scenarios.
- Run command: Focused provider test plus exact Go PDS-record shape/conflict test.
- Confirmed failure: The repository mutation surface did not accept `CreatePostExternal`.
- Implement: Forwarded the materialized external payload from standard composer submission to the existing external create wire model.
- Run command: Real composer acceptance tests, provider test, and `TestCreatePost_ExternalEmbed_WritesStandardShapeAndRejectsConflicts` pass.
- Refactor: Kept selection/session concerns out of `CreatePost`; it receives only an immutable wire payload.
- Notes: Only a completed, unsuppressed, undismissed synchronous selection is eligible at submit time.

### Step 42: REG-001

- Write failing test: Reused standard-composer mention/link/tag facet submission coverage with preview integration active.
- Run command: Focused `post_composer_sheet_facets_test.dart` selection.
- Confirmed failure: Not expected; preview derivation does not mutate the text or facet controller.
- Implement: No production change required.
- Run command: Existing UTF-8 facet submission scenario passes unchanged.
- Refactor: None required.
- Notes: External preview payload and rich-text facets remain independent fields on create.

### Step 43: UT-018

- Write failing test: Added Go and Flutter round-trip fixtures for metadata-only and thumbnail-bearing frozen scheduled external state plus standard/project eligibility.
- Run command: Focused scheduled payload/model tests.
- Confirmed failure: Scheduled payloads and Flutter models had no external/source/thumb-media representation.
- Implement: Added strict frozen external codecs with source identity, final URI, metadata, optional private thumbnail media ID, and project-external rejection.
- Run command: Focused Go tests and Flutter model tests pass.
- Refactor: Kept external thumbnail identity separate from ordinary image payload entries so images-win and project-photo invariants remain explicit.
- Notes: The frozen payload contains no thumbnail bytes and metadata-only state omits the private-media ID.

### Step 44: IT-015

- Write failing test: Added raw external-thumbnail staging coverage and standard/project scheduled-request fixtures.
- Run command: Focused Flutter scheduled model/submission tests and AppView scheduled request tests.
- Confirmed failure: No scheduled external materializer or private thumbnail association existed.
- Implement: Staged optional preview bytes unchanged through the existing private-media API before schedule create/update, persisted only the media ID, and rejected project external payloads.
- Run command: Thumbnail and metadata-only schedule flows, existing atomic image staging/failure tests, and project rejection pass.
- Refactor: Reused scheduled media timeout/cancellation and stable composer-owned UUID behavior.
- Notes: Create follows successful staging; metadata-only cards make no stage request and project production behavior remains unchanged.

### Step 45: IT-016

- Write failing test: Added frozen external publication-record coverage and carried metadata-only external state through the existing worker and all crash-recovery boundaries.
- Run command: Focused publication processor and recovery acceptance tests against required Postgres.
- Confirmed failure: Publication records did not project frozen external state or associate the optional predicted thumbnail blob.
- Implement: Counted external thumbnail media separately from image media, froze the exact standard external embed before effects, uploaded staged media through existing guarded effects, and reused frozen bytes across retries/recovery.
- Run command: Worker publication, thumbnail record construction, and four crash-boundary recovery scenarios pass.
- Refactor: Source identity remains private payload state and is intentionally omitted from the public PDS record.
- Notes: The publication processor has no metadata-fetch dependency, so publication/retry cannot refresh metadata.

### Step 46: AT-005

- Write failing test: Added widget acceptance for thumbnail and metadata-only scheduled cards, controller seeding without source refetch, and real scheduled-editor hydration/preserve/remove/replace/failure scenarios.
- Run command: Full scheduled submission widget suite, focused controller seed test, worker publication, and recovery tests.
- Confirmed failure: Reopened frozen state had no controller seed path and scheduled submission omitted external state.
- Implement: Seeded frozen scheduled metadata into a new preview session, retained it only while source identity remains, fetched only new candidates, and saved/published both optional-thumbnail variants.
- Run command: Both Flutter scenario variants and AppView frozen publication selections pass.
- Refactor: Existing frozen external maps are reused on edit when their source remains, preserving private thumbnail identity without restaging/refetching.
- Notes: Removing the source URL removes frozen selected state; ordinary new candidates remain independently eligible.

### Step 47: REG-005

- Write failing test: Reused the complete Flutter and AppView scheduled-post suites after adding external state.
- Run command: `flutter test test/scheduled_posts` and required-Postgres `go test ./internal/scheduledposts -count=1`.
- Confirmed failure: Not expected; this is the scheduled behavior preservation gate.
- Implement: No additional production change required.
- Run command: All 42 Flutter scheduled tests and the full Go scheduled-post package pass.
- Refactor: None required.
- Notes: Capacity, image staging/editing, publication retry, recovery, cleanup, project fields, and management behavior remain intact.

### Step 48: UT-019

- Write failing test: Added recursive manifest-key exclusions for external, cache, selection, dismissal, and suppression state while retaining URL text as ordinary draft content.
- Run command: Focused draft manifest codec test.
- Confirmed failure: Not expected; the existing whitelist already excludes preview session state.
- Implement: No production change required.
- Run command: Explicit preview-state exclusion canary passes.
- Refactor: None required.
- Notes: URLs in authored text remain draft content; derived/fetched preview state never enters the manifest.

### Step 49: IT-017

- Write failing test: Added account-scoped controller integration coverage for reopening the same URL under a new composer identity after the old session was dismissed.
- Run command: Focused controller test.
- Confirmed failure: Not expected because controller families are composer-session scoped.
- Implement: No additional production change required.
- Run command: New session is not dismissed, starts a fresh fragmentless request, and the old request is cancelled.
- Refactor: None required.
- Notes: No preview cache, failure, selection, suppression, or dismissal state crosses draft reopen.

### Step 50: AT-006

- Write failing test: Added a Drafts-page widget flow that reopens URL text and expects a freshly fetched composer carousel.
- Run command: Focused `drafts_page_test.dart` acceptance selection.
- Confirmed failure: Not expected after composer/controller integration; the test makes the product boundary explicit.
- Implement: No additional production change required.
- Run command: Reopened draft renders fresh preview metadata and sends exactly one fragmentless source request.
- Refactor: None required.
- Notes: Draft hydration supplies only authored text/languages/media; the composer derives a new preview session.

### Step 51: REG-006

- Write failing test: Reused the full local-draft suite after composer preview integration.
- Run command: `flutter test test/drafts`.
- Confirmed failure: Not expected; this is the draft schema/behavior preservation gate.
- Implement: No additional production change required.
- Run command: All 47 draft tests pass.
- Refactor: None required.
- Notes: Storage schema, privacy redaction, media recovery, retention, account isolation, save progress, and no-transfer boundaries remain unchanged.

### Step 52: UT-020

- Write failing test: Added AppView metric canaries using arbitrary URL-like stage/result/error values and Flutter failure-path import/reporting canaries.
- Run command: Focused Go observability and Flutter privacy tests.
- Confirmed failure: `ObserveLinkPreview` and a preview-specific bounded metric boundary were absent.
- Implement: Added closed stage/result/error allowlists plus status, redirect-count, byte, and non-negative duration buckets; Flutter silently caches failure without an analytics/error-reporting path.
- Run command: Go bounded-field/all-sink and Flutter reporter/event-boundary privacy tests pass.
- Refactor: Metric sinks receive only sanitized attributes; raw dependency errors and user-derived strings are never method parameters beyond the sanitizer boundary.
- Notes: Unknown strings collapse to `unknown`; production Sentry metrics and in-memory tests share the same sanitized map.

### Step 53: IT-018

- Write failing test: Added endpoint canaries, then extended correction coverage through the real route admission stack, auth/device/disabled/rate failures, scheduled thumbnail publication retry, and cleanup retry.
- Run command: Focused API, observability, and Flutter privacy/network-boundary suites.
- Confirmed failure: The preview endpoint was not connected to a dedicated privacy-safe observer.
- Implement: Wired route-owned observability into disabled, validation, policy, fetch failure, timeout/upstream, and success endpoint paths using fixed classes only.
- Run command: API/observer and Flutter privacy integrations pass.
- Refactor: Handler tests remain source-compatible through the optional observer argument; capture scans now prove no URL, metadata, identity, token, or device field enters preview telemetry.
- Notes: Scheduled publication and cleanup use real private payload/media/object fixtures and scan the observer output after failure/retry paths.

### Step 54: AT-009

- Write failing test: Added success-response canaries for final URL, metadata, and thumbnail bytes alongside failure canaries.
- Run command: Required-Postgres Go privacy selection across API/observability/scheduling and Flutter observability/draft/network-boundary selection.
- Confirmed failure: Success-specific preview telemetry coverage was absent.
- Implement: Success emits only complete/success/none, status class, byte bucket, redirect bucket, and duration; response content remains confined to the API response/model.
- Run command: All cross-pillar privacy selections pass, including serialized local logs, log sink attributes, metrics, Sentry errors, Sentry traces, Flutter reporter calls, and breadcrumb/event-source scans.
- Refactor: None required.
- Notes: No preview-specific logs, traces, breadcrumbs, client analytics, or error reports carry user intent; metrics expose only approved bounded values.

### Step 55: UT-015

- Write failing test: Added narrow/full and compact card tests for exact final-URI launch, bounded host/title, empty description, and absent thumbnail.
- Run command: Focused `external_card_test.dart`.
- Confirmed failure: Shared `ExternalCard` and localized card semantics were absent.
- Implement: Added one responsive full/compact card with bounded copy, optional fixed thumbnail frames, AppView cached image loading, localized semantics, and existing external launcher injection.
- Run command: Focused card tests and analyzer pass.
- Refactor: URI parsing/display uses the existing external-link helpers while the exact normalized final URI remains the launch target.
- Notes: Scheme/query/fragment are hidden from the host label but preserved for launch.

### Step 56: IT-014

- Write failing test: Added shared `PostCard` full-card and `PostSummaryData` compact quote integration tests with images-win fixtures.
- Run command: Focused post-card and post-summary tests after mapper generation.
- Confirmed failure: Neither shared rendering surface consumed external model data.
- Implement: Rendered eligible external state in `PostCard`, carried compact quote external through `PostSummaryData`, and applied images-win again at presentation.
- Run command: Full/compact integration tests pass.
- Refactor: Feed/profile/comment/detail keep composing the existing shared `PostCard`; no per-page renderer or network path was added.
- Notes: Hidden/unavailable quote states remain state-only and notification summaries still exclude external cards.

### Step 57: AT-004

- Write failing test: Added large-text thumbnail layout coverage, shared full/compact rendering, and concrete `FeedPage`, `ProfilePostsTab`, post-detail, comment, and reply surface fixtures.
- Run command: External card, full PostCard, profile tab, thread, comment section, and compact summary suites.
- Confirmed failure: Thumbnail/large-text acceptance coverage and card localization were absent.
- Implement: Added localized action/thumbnail semantics and adaptive narrow-column/wide-row/compact layouts.
- Run command: The corrected combined composer/scheduled/surface selection passes 133 tests; focused card tests include exact launch and large-text thumbnail behavior.
- Refactor: None required.
- Notes: Rendering performs no metadata fetch; optional thumbnails use only the AppView-provided blob URL and existing feed cache manager.

## Final Verification

- Consecutive `just lexgen` runs produced the same generated lexicon diff hash, `ff850ccf1b67b28d3984cb6fffa0b5755c0b991cf5da397374ddfe8c64d7b81c`; `just lexgen-check` remains the clean-tree pre-merge drift guard.
- `just fmt` passed, including `go vet ./...`.
- `just dev-d` built and started AppView, Postgres, MinIO, and Tap; required services report healthy where health checks are defined.
- `just test` passed with the race detector, real Postgres, and MinIO.
- Flutter generation completed with `build_runner` and `flutter gen-l10n`.
- `just app-analyze` passed with no issues.
- `just app-test` passed all 1,529 tests after the implementation-review corrections.
- `MAN-001` responsive/accessibility device smoke, `MAN-002` production direct-egress enablement, and `MAN-003` privacy-safe p95 measurement remain release-time gates.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] No database migration or redundant public external column added
- [x] No automated test uses public DNS or public websites
- [x] `just lexgen-check` ready for clean-tree pre-merge execution
- [x] `just fmt` passing
- [x] `just test` passing
- [x] `just app-analyze` passing
- [x] `just app-test` passing
- [x] Manual release gates recorded as pending or completed
- [x] Docs and per-test red/green evidence updated
- [x] Implementation review completed; ordered corrections tracked below

## Implementation Review Correction Order

The `06-implementation-review.md` verdict is Changes required. Corrections run
in reviewer order, one focused red-green-refactor loop at a time. The review
document remains immutable.

| Step | Review ID | Requirement / Criteria | Focused Missing Behavior |
|---|---|---|---|
| 58 | IR-001 | FR-016; AC-018; AT-005; IT-015 | Scheduled edit hydrates real thumbnail identity/bytes and preserves, removes, replaces, or fails staging without corrupting frozen state |
| 59 | IR-002 | FR-003, FR-006, FR-016; AC-003, AC-007, AC-018; IT-015, IT-016 | Save and freeze validate referenced external media role, MIME, and exact 1,000,000-byte limit |
| 60 | IR-003 | FR-012; AC-012; AT-003; IT-013 | Submit snapshots only completed selection, cancels/invalidate active and queued preview work, and ignores late completion after publish failure |
| 61 | IR-004 | FR-005, NFR-002; AC-004; UT-010 | Public-address policy rejects the complete special-use IPv4/IPv6 matrix and mapped forms |
| 62 | IR-005 | FR-004; AC-006; IT-004 | A response body stalled until context cancellation maps to the 504 timeout envelope |
| 63 | IR-006 | FR-003, FR-004; AC-003, AC-006; IT-004 | Image decode work has bounded concurrency and cannot affect results after the total deadline |
| 64 | IR-007 | FR-016; AC-018; IT-016; REG-005 | Publication snapshots retain exact media row IDs and recovery proves real external thumbnails across retry boundaries |
| 65 | IR-008 | AT-001..AT-005; IT-013, IT-014 | Real composer and shared-surface harnesses cover approved behavior and traceability names concrete evidence |
| 66 | IR-009 | NFR-004; AC-022; UT-020, IT-018, AT-009 | Cross-pillar capture harnesses prove unique privacy canaries never enter logs, traces, Sentry, middleware, schedule, publication, or cleanup sinks |
| 67 | IR-010 | FR-006; AC-007; IT-008 | Immediate and scheduled authoring construct external embeds through generated union types |
| 68 | IR-011 | FR-009; UT-001; TD-001 | Shared token parsing accepts uppercase HTTP(S) schemes without changing byte ranges |

## Correction Execution Evidence

### Step 58: IR-001
- Write failing test: Added real scheduled-editor coverage for unchanged frozen thumbnail hydration/identity, explicit dismissal, replacement under a new immutable ID, and replacement staging failure retention, one scenario at a time.
- Run command: `flutter test test/scheduled_posts/scheduled_post_edit_test.dart --plain-name 'AT-005 unchanged scheduled external retains thumbnail identity and bytes'`.
- Confirmed failure: The unchanged editor made no private-media request, so the frozen selection had no thumbnail bytes and save would drop its `thumbMediaId`. The first dismissal attempt also exposed and corrected a test hit-target setup issue before behavior was assessed.
- Implement: Hydrated existing external thumbnail bytes through the scheduled repository with byte/type/decode validation, seeded them into the frozen selection without refetching, returned an exact unchanged frozen map without restaging, and reserved a fresh UUID for replacements. Existing dismissed-state admission explicitly removes the card; replacement staging failures leave the editor/card intact and save nothing.
- Green/regression command: Each focused scenario passed; `flutter test test/scheduled_posts/scheduled_post_edit_test.dart test/scheduled_posts/scheduled_post_submission_test.dart` passed 22 tests.
- Refactor/notes: Existing IDs are never reused for replacements. Frozen source/URI/title/description and thumbnail presence must all match for no-op preservation.

### Step 59: IR-002
- Write failing test: Added public create-handler/store coverage for an actual 1,000,001-byte external media row, then exact-limit WebP, unsupported MIME, exact-reference mismatch, and publication-time metadata mutation cases separately.
- Run command: `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/api -run TestScheduledPostCreateRejectsOversizedExternalThumbnailMedia -count=1`.
- Confirmed failure: The real oversized ready row produced HTTP 201 because request validation used a one-byte JPEG placeholder. The freeze-time mutation test persisted 372 frozen record bytes before later upload validation failed.
- Implement: Canonical payload decoding now classifies exact ordered media references transactionally. Create/update lock and validate actual row MIME/size, external thumbnails allow JPEG/PNG/WebP through exactly 1,000,000 bytes, mismatched IDs fail closed, and publication repeats role/MIME/size validation before freeze.
- Green/regression command: Focused API/store/publication tests passed; `go test ./internal/api ./internal/scheduledposts -count=1` passed after stale lifecycle fixtures were corrected to include exact payload media references.
- Refactor/notes: Ordinary image limits remain independent. Invalid media receives the bounded `scheduled_media_invalid` envelope; no placeholder participates in durable media validation.

### Step 60: IR-003
- Write failing test: Added real `PostComposerSheet` coverage with two queued URLs, a pending first request, failed publication, and late preview completion.
- Run command: `flutter test test/scheduled_posts/scheduled_post_submission_test.dart --plain-name 'IT-013 submit invalidates pending preview queue after publish failure'`.
- Confirmed failure: The active preview cancel token remained uncancelled after submit began.
- Implement: Added synchronous `snapshotForSubmit`, which returns only an already successful eligible selection, invalidates the generation, cancels active work, and blocks queue advancement until text actually changes. The composer invokes it before facets, dialogs, uploads, or any other await.
- Green/regression command: Focused test passed; controller plus scheduled submit/edit selection passed 31 tests.
- Refactor/notes: Completed selections remain visible for retry after downstream failure; pending results and queued candidates cannot attach or start after submission invalidation.

### Step 61: IR-004
- Write failing test: Added `fec0::/10` first, then explicit cases for `2001:2::/48`, `2001:10::/28`, `2001:20::/28`, both ends of deprecated site-local space, and IPv4-mapped special-use forms.
- Run command: `go test ./internal/linkpreview -run TestValidateAddressesRejectsDeprecatedIPv6SiteLocal -count=1`.
- Confirmed failure: `fec0::1` was accepted with no error.
- Implement: Added deprecated IPv6 site-local `fec0::/10` to the fail-closed non-public table. Existing `2001::/23` already rejects all three cited IETF/benchmark/ORCHID ranges; explicit tests now prevent accidental narrowing. Unmapping continues to apply the complete IPv4 matrix to mapped addresses.
- Green/regression command: Focused test passed; `go test ./internal/linkpreview -count=1` passed.
- Refactor/notes: Resolve-and-pin, whole-answer-set rejection, and direct no-proxy transport are unchanged.

### Step 62: IR-005
- Write failing test: Added a real service/handler fetcher whose page returns HTTP 200 HTML headers and whose body blocks until the request context expires.
- Run command: `go test ./internal/api -run TestLinkPreviewHandlerMapsStalledPageBodyToTimeout -count=1`.
- Confirmed failure: The stalled body returned HTTP 422 `link_preview_unsupported` instead of 504 `link_preview_timeout`.
- Implement: Capture the page context error before explicit phase cancellation and preserve deadline/cancellation read errors; only non-context head decode failures map to unsupported content.
- Green/regression command: Focused handler test passed; focused link-preview service/API regressions passed.
- Refactor/notes: The test asserts the standard bounded envelope and request ID; it releases deterministically on context cancellation.

### Step 63: IR-006
- Write failing test: Added separate cancelled-before-work, blocked near-limit decode deadline, and five-request decode-concurrency tests. The deterministic decoder seam receives an exact 1,000,000-byte input and reports a 16 MP result without relying on machine-specific decode timing.
- Run command: `go test ./internal/linkpreview -run TestServiceThumbnailIgnoresDecodeResultAfterTotalDeadline -count=1`, followed by `go test ./internal/linkpreview -run TestServiceThumbnailDecodeConcurrencyIsBounded -count=1`.
- Confirmed failure: A blocked decode kept `fetchThumbnail` waiting beyond the total deadline. After deadline selection was added, three simultaneous decode calls started because there was no admission bound.
- Implement: Reject already-cancelled thumbnail work before fetch/decode, execute validation behind a two-slot per-service decode admission channel, retain each slot until the real decoder exits, and return immediately without consuming a late result when the total context expires.
- Green/regression command: All three focused resource-bound tests passed; `go test ./internal/linkpreview -count=1` and the focused `go test -race` selection passed.
- Refactor/notes: The buffered result channel lets a decoder finish after caller cancellation without blocking. Timed-out decode work still occupies admission capacity until it actually exits, preventing abandoned near-limit decodes from accumulating without bound.

### Step 64: IR-007
- Write failing test: Added a publication test that keeps media role/count constant while replacing the canonical payload UUID, then corrected recovery coverage for metadata-only external cards and real `thumbMediaId` schedules at every applicable upload/record crash boundary.
- Run command: `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/scheduledposts -run TestPublicationWorkerRejectsSameCountDifferentMediaIdentityBeforeFreeze -count=1`.
- Confirmed failure: The same-count UUID substitution caused one private upload and one record write because snapshot media rows carried no IDs and were assigned positionally.
- Implement: Publication snapshots now select and scan each ready media row UUID. The processor compares every exact ordered payload reference to the corresponding snapshot row before freeze or effects.
- Green/regression command: Exact-identity and parameterized recovery tests passed; `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/scheduledposts -count=1` passed.
- Refactor/notes: The recovery fixture now models the last media row as the external thumbnail, asserts the published external blob CID, covers partial upload with an ordinary image plus real thumbnail, and covers metadata-only/external-thumbnail variants before record write and after record success.

### Step 65: IR-008
- Write failing test: Added the previously missing real composer and page-harness scenarios for selection/reorder, dismissal snackbar/Undo, quote and project suppression, pending submit, thumbnail upload failure retention, scheduled external edit hydration, and every full post surface.
- Run command: Focused tests were run individually while added, followed by one combined eight-file Flutter selection.
- Confirmed failure: The real snackbar test initially missed Undo because its text center was below the synthetic 600 px test viewport; the real upload-failure test also corrected its assertion to the existing localized immediate-post failure copy. Neither exposed a production behavior defect after interaction was made deterministic.
- Implement: No production change was needed in this correction. Tests now exercise `PostComposerSheet`, `ProjectComposerSheet`, `FeedPage`, `ProfilePostsTab`, `PostThreadPage`, comment/reply cards, scheduled edit/submission, `PostCard`, and compact `PostSummary` directly.
- Green/regression command: The combined acceptance selection passed all 133 tests.
- Refactor/notes: Steps 37, 38, 41, 46, and 57 now name concrete evidence rather than claiming controller/carousel composition as end-to-end coverage.

### Step 66: IR-009
- Write failing test: Added distinct URL, host, query, redirect, title, description, thumbnail-byte, post-text, DID, bearer-token, device-ID, object-key, and dependency-error canaries across telemetry sinks and actual admission/scheduled lifecycle paths.
- Run command: Required-Postgres focused Go selection across `internal/observability`, `internal/routes`, `internal/api`, and `internal/scheduledposts`, plus `flutter test test/observability/link_preview_privacy_test.dart`.
- Confirmed failure: The expanded Flutter scan initially named a nonexistent standalone API file and was corrected to the production `post_api_client.dart`; no production telemetry leak was found by the completed capture harnesses.
- Implement: Added serialized capture assertions for local slog, the external log sink, bounded metrics, Sentry errors, Sentry transaction/span data, route auth/device/disabled/rate outcomes, successful/failing endpoint observation, scheduled external-thumbnail upload plus publication retry, cleanup failure/retry, and Flutter error/message/breadcrumb collectors.
- Green/regression command: All focused Go privacy packages and the Flutter privacy test pass.
- Refactor/notes: Traceability no longer relies on source assertions alone. Source scans remain only as evidence that preview UI/network files do not create client analytics, reports, or breadcrumbs; runtime collectors prove no calls occur on failure.

### Step 67: IR-010
- Write failing test: Added an authoring architecture test requiring both immediate and scheduled production paths to call one helper that constructs `craftsky.FeedPost_Embed`, `appbsky.EmbedExternal`, and `appbsky.EmbedExternal_External`.
- Run command: `go test ./internal/lexicon/craftsky -run TestIT008ProductionExternalAuthoringUsesGeneratedUnionTypes -count=1`.
- Confirmed failure: `post_create.go` still used a hand-built external map, and the scheduled path did the same.
- Implement: Added `internal/postrecord.ExternalEmbed`, which decodes the optional blob through `lexutil.LexBlob`, constructs the generated open-union variant and Indigo standard external objects, and emits the existing map wire shape. Both authoring paths now propagate generated-construction errors.
- Green/regression command: Architecture and focused immediate/scheduled external tests passed; required-Postgres `go test ./internal/postrecord ./internal/lexicon/craftsky ./internal/api ./internal/scheduledposts -count=1` passed.
- Refactor/notes: No lexicon schema changed. Malformed fake thumbnail CIDs in authoring tests were replaced with canonical CIDs because the generated standard blob type validates links; JSON-equivalent numeric size values are compared on their wire encoding.

### Step 68: IR-011
- Write failing test: Added end-to-end candidate derivation for completed uppercase `HTTP://` and `HTTPS://` values through the shared facet-token parser.
- Run command: `flutter test test/shared/rich_text/link_preview_candidates_test.dart --plain-name 'UT-002 deriveLinkPreviewCandidates derives uppercase HTTP schemes through shared facet tokens'`.
- Confirmed failure: Candidate derivation returned an empty list even though direct `LinkPreviewCandidate.parse` accepted the same values.
- Implement: Made the shared link regular expression case-insensitive and made `_linkUri` recognize existing schemes through a lowercase comparison while preserving the original visible token and byte range.
- Green/regression command: Candidate derivation, candidate normalization, and shared facet generation suites passed all 16 tests.
- Refactor/notes: Bare-domain behavior and emitted facet ranges are unchanged.

## Correction Completion Checklist

- [x] IR-001 complete
- [x] IR-002 complete
- [x] IR-003 complete
- [x] IR-004 complete
- [x] IR-005 complete
- [x] IR-006 complete
- [x] IR-007 complete
- [x] IR-008 complete
- [x] IR-009 complete
- [x] IR-010 complete
- [x] IR-011 complete
- [x] Generation and full repository gates pass
- [x] Final diff and `05-implementation-plan.md` readback complete

## Second Implementation Review Correction Order

The second-round findings in immutable `06-implementation-review.md` run in
reviewer order, one focused red-green-refactor loop per behavior.

| Step | Review ID | Requirement / Criteria | Focused Missing Behavior |
|---|---|---|---|
| 69 | IR-012 | FR-016; AC-018; AT-005; IT-015 | Delayed scheduled-thumbnail hydration atomically seeds the frozen selection and preserves its media identity and bytes |
| 70 | IR-013 | FR-011, FR-016, RULE-002; AC-011, AC-014, AC-018; AT-002, AT-005 | Scheduled save explicitly chooses preserve, attach, or remove for images, dismissal, source removal, and unchanged frozen state |
| 71 | IR-014 | FR-016; AC-018; IT-015 | A changed replacement after staged update failure receives a fresh immutable media ID |
| 72 | IR-015 | FR-005, NFR-002; AC-004, AC-005; UT-010, IT-002 | IPv6 admission is fail-closed to supported public allocation space and rejects current special-use/reserved ranges |
| 73 | IR-016 | AT-001, AT-002, AT-004, IT-012, IT-014 | Concrete tests or exact behavior mappings cover every remaining composer and rendering acceptance clause |
| 74 | IR-017 | NFR-004; AC-022; UT-020, IT-018, AT-009 | Actual lifecycle paths are captured at every applicable sink and impossible-by-construction sinks are documented and boundary-tested |

## Second Correction Execution Evidence

### Step 69: IR-012

- Write failing test: Added a real scheduled-editor test whose thumbnail `mediaBytes` future remains behind a controlled completer until the initial metadata-only frozen seed has run.
- Run command: `flutter test test/scheduled_posts/scheduled_post_edit_test.dart --plain-name 'IR-012 delayed scheduled thumbnail hydration reseeds frozen selection'`.
- Confirmed failure: After delayed hydration completed, the controller selection still had no thumbnail bytes, so unchanged save would no longer match the frozen external record.
- Implement: Completing scheduled-media hydration now clears the one-time seed guard in the same state transition that installs the thumbnail and enables save. The next composer frame reseeds the watched account-scoped controller from the complete frozen state.
- Green/regression command: The focused delayed-completer test passed; the complete `scheduled_post_edit_test.dart` suite passed all 19 tests.
- Refactor/notes: The correction avoids seeding a newly read auto-dispose provider before the composer watches it. No preview refetch or media restaging occurs.

### Step 70: IR-013

- Write failing test: Changed the scheduled dismissal scenario so the complete source URL remains in the middle of text, and added an existing-schedule images-win save with both frozen image media and an external card.
- Run command: Each exact widget test was run separately with `--plain-name` before implementation.
- Confirmed failure: Both saves retained the frozen `external` map. Dismissal and image suppression produced no selected preview, and the URL-presence fallback interpreted that null as preserve.
- Implement: Submission now computes an explicit scheduled external disposition before any await. Images, dismissal, or source removal select `remove`; an exact frozen selection selects `preserve`; a different completed selection selects `attach`. Save/publish materialization switches only on that disposition and no longer infers intent from a null selection.
- Green/regression command: The dismissal and images-win tests passed independently. An explicit source-removal regression passed, and combined scheduled edit/submission suites passed all 30 tests.
- Refactor/notes: Existing unchanged and replacement tests provide concrete preserve/attach coverage. Snapshot invalidation still runs for scheduled image/dismissed submissions even though no external is attached.

### Step 71: IR-014

- Write failing test: Added a real editor sequence where replacement A stages successfully, schedule update fails, the user changes to replacement B with different metadata/bytes, and update then succeeds.
- Run command: `flutter test test/scheduled_posts/scheduled_post_edit_test.dart --plain-name 'IR-014 changed replacement after failed update stages under a fresh ID'`.
- Confirmed failure: Both staged byte sets used the same editor-lifetime UUID, and the successful payload referenced that reused immutable media identity.
- Implement: New-schedule creation retains its stable operation-scoped external media ID for idempotent create retries. Every attach attempt on an existing schedule now allocates a fresh UUID, so a staged replacement whose update did not commit cannot collide with later replacement content.
- Green/regression command: The focused two-attempt test passed; the complete scheduled edit suite passed all 22 tests.
- Refactor/notes: Unchanged frozen state still uses `preserve` and never stages. Only the explicit `attach` branch allocates replacement identity.

### Step 72: IR-015

- Write failing test: Added explicit IPv6 rejection cases for IANA dummy `100:0:0:1::1`, the address immediately below supported space, `4000::1`, `8000::1`, and `c000::1`.
- Run command: `go test ./internal/linkpreview -run TestValidateAddressesRejectsIPv6OutsideSupportedPublicAllocation -count=1`.
- Confirmed failure: Every new reserved/unallocated representative was accepted because `netip.Addr.IsGlobalUnicast` intentionally classifies them as global unicast.
- Implement: IPv6 admission now fails closed unless the unmapped address is within IANA's currently allocated `2000::/3` global-unicast space, then applies the existing current special-purpose exclusions. IPv4 and mapped-IPv4 continue through the complete IPv4 policy.
- Green/regression command: The focused allocation test, complete `internal/linkpreview` package, and full package race run passed.
- Refactor/notes: Representative Cloudflare IPv6 remains admitted. Resolve-and-pin and whole-answer-set rejection are unchanged.

### Step 73: IR-016

- Write tests/mapping: Added a real five-link composer with four-request cap, sequential active-count assertion, silent failed candidate, ordered successful navigation, and default selection. Added real add/remove image actions against cached plus pending candidates, and real injected-duration Undo expiry followed by a fresh composer session. Upgraded every full post-surface harness and compact quote coverage from existence checks to content, bounded-layout/no-exception, exact final-URI launcher, images-win, and hidden/unavailable assertions.
- Run command: Each new composer scenario ran independently first, followed by the combined controller, composer, project, feed, profile, detail/comment/reply, compact-summary, and shared-card selection.
- Confirmed failure: Five-link sequencing and image suppression/restoration passed after deterministic real hit-target setup, confirming missing evidence rather than a production defect. Undo expiry failed because current Flutter keeps snackbars with actions persistent by default and the composer had no injected duration/explicit persistence policy. Existing surface tests also could not observe launcher calls before the shared injection seam.
- Implement: Added a four-second `linkPreviewUndoDurationProvider`, set the Undo snackbar to `persist: false`, and added a shared `externalCardLauncherProvider` used only when a card does not receive a direct launcher. No preview selection, rendering, or navigation contract changed.
- Green/regression command: All three concrete composer scenarios passed. The combined eight-file acceptance/nearby selection passed all 64 tests.
- Refactor/notes: `AT-001` maps to the new five-link test plus the existing real identity-following test and `IT-012` fragment-reorder test. `AT-002` maps to real image actions, real dismissal/Undo, real expiry/new-session, quote-composer, and project-composer tests. `IT-012` now labels immediate/sequential/cache, fragment precedence, suppression, dismissal/expiry, stale/disposal, and account-boundary tests explicitly. `AT-004`/`IT-014` map to exact launch/content/layout assertions on feed, profile, detail, comment, reply, shared narrow/large-text thumbnail, and compact visible/hidden/unavailable quote tests.

### Step 74: IR-017

- Write tests/boundaries: Wrapped the actual registered admission stack, real preview handler success/failure, and real scheduled-save failure in local slog, bounded metric, external-log, Sentry error, and Sentry transaction collectors with distinct lifecycle canaries. Extended the real scheduled composer failure test with an enabled client reporter collector. Added production-source boundary tests for the scheduled worker and Flutter composer paths.
- Run command: Required-Postgres `go test ./internal/observability ./internal/routes ./internal/api ./internal/middleware ./internal/scheduledposts -count=1`, followed serially by `flutter test test/observability/link_preview_privacy_test.dart test/scheduled_posts/scheduled_post_submission_test.dart`.
- Confirmed missing evidence: Existing lifecycle tests did not place actual route/fetch/save paths under every sink they can reach. Worker publication/retry/cleanup already exercised concrete bounded metrics, but there was no executable proof that those packages could not emit logs, traces, or Sentry events. The real Flutter scheduled-save failure likewise had no client-event collector.
- Implement/evidence: Actual admission, fetch, and save paths now prove local logs, HTTP metrics, transactions, and 5xx Sentry events remain free of URL/content/identity/dependency canaries. Their external observer-log collector remains empty because these lifecycle paths do not call `Observer.Log`. Publication, retry, stale-worker recovery, needs-attention exhaustion, cleanup success, and cleanup retry continue through the concrete `OperationalObserver` metric path with distinct canaries.
- Impossible-by-construction sinks: Scheduled worker production files may depend only on the four bounded `OperationalObserver` methods; a package-wide guard fails on direct slog, Sentry, capture, span, breadcrumb, or observer-log calls. Flutter preview and scheduled composer production files are guarded against Sentry, report, breadcrumb, and analytics calls; enabled runtime collectors remain empty during actual preview and scheduled-save failures. These boundaries make non-applicable sinks explicit instead of inferring them from sanitizer unit tests.
- Green/regression command: All five focused Go privacy packages passed with required Postgres. The combined Flutter privacy/submission selection passed all 13 tests. `git diff --check` passed.
- Refactor/notes: Production behavior is unchanged. Generic observer sanitizer tests remain defense in depth; lifecycle tests now supply the AC-022 evidence.

### Second Correction Final Verification

- Deterministic generation: Two consecutive `just lexgen` runs retained generated lexicon diff SHA-256 `ff850ccf1b67b28d3984cb6fffa0b5755c0b991cf5da397374ddfe8c64d7b81c`.
- Go formatting/static checks: `just fmt` passed, including `gofmt` and `go vet ./...`.
- Go tests: Required-Postgres race-enabled `just test` passed the complete AppView suite. The first invocation was terminated only by the 120-second command timeout; the identical rerun passed with a longer timeout.
- Flutter analysis: `just app-analyze` passed with no issues after mechanical lint cleanup in second-round files.
- Flutter tests: `just app-test` passed all 1,536 tests.
- Final audit: `git diff --check` passed, immutable `06-implementation-review.md` has no tracked diff, and no unrelated generated-file changes appear in status.

## Second Correction Completion Checklist

- [x] IR-012 complete
- [x] IR-013 complete
- [x] IR-014 complete
- [x] IR-015 complete
- [x] IR-016 complete
- [x] IR-017 complete
- [x] Generation and full repository gates pass after second corrections
- [x] Final diff and `05-implementation-plan.md` readback complete after second corrections

## Third Implementation Review Correction Order

The third-round findings in `06-implementation-review.md` run in reviewer order,
one focused red-green-refactor loop per behavior.

| Step | Review ID | Requirement / Criteria | Focused Missing Behavior |
|---|---|---|---|
| 75 | IR-018 | FR-016; AC-018; AT-005; IT-015 | Delayed hydration cannot restore a removed frozen source or override a deliberate alternate selection |
| 76 | IR-019 | FR-016; AC-018; AT-005 | A frozen card survives the create-save-reopen-save round trip when submission trimming removes its completion boundary |
| 77 | IR-020 | FR-016; AC-018; IT-015 | New schedules retain a staged-media ID only for identical-content retry and rotate it for changed content |
| 78 | IR-021 | FR-011; AC-011; AT-002; IT-012 | Undo expiry applies only to the dismissal generation that created the snackbar |
| 79 | IR-022 | FR-007; AC-008; UT-012; IT-007 | Malformed external thumbnail blob type/CID is rejected with 4xx before any PDS write |
| 80 | IR-023 | FR-016; AC-018; UT-018 | Scheduled `sourceUri` is a validated canonical fragmentless composer identity |
| 81 | IR-024 | NFR-004; AC-022; IT-018; AT-009 | Real fetch lifecycle paths prove privacy across DNS, redirect, parse, image, and failure stages |

## Third Correction Execution Evidence

### Step 75: IR-018

- Write failing test: Added delayed scheduled-thumbnail hydration where the member removes the frozen source and selects a replacement before private bytes finish loading.
- Confirmed failure: Hydration reseeded and selected the removed frozen card, overriding the member's newer state.
- Implement: Added `refreshSeed`, which updates hydrated frozen thumbnail data only while that identity remains a current cached candidate and never changes the current selection. Hydration no longer resets the one-time seed guard.
- Green/regression command: The focused stale-hydration and original delayed-hydration tests passed; nearby combined Flutter suites passed.
- Refactor/notes: Initial frozen metadata still seeds once. Late bytes may enrich that card but cannot resurrect it or select it.

### Step 76: IR-019

- Write failing test: Added a real create-save-reopen-save round trip using the trimmed text actually persisted by schedule creation.
- Confirmed failure: The reopened terminal URL had no completion boundary, so text synchronization removed the frozen external and the next save omitted it.
- Implement: Candidate derivation accepts a terminal incomplete token only when it matches the controller's seeded frozen identity. Ordinary terminal URLs remain ineligible and trigger no preview fetch.
- Green/regression command: The focused round trip and ordinary candidate-boundary suite passed.
- Refactor/notes: The exception is scoped to an already-frozen scheduled identity and preserves the original no-fetch-while-typing rule.

### Step 77: IR-020

- Write failing test: Added a new-schedule sequence where thumbnail A stages, identical retry stages again, creation fails twice, then the member selects thumbnail B.
- Confirmed failure: Thumbnail B reused thumbnail A's immutable scheduled-media UUID.
- Implement: New schedules retain the replacement UUID only while thumbnail bytes and MIME type are identical. Changed content allocates a new UUID; existing-schedule attach attempts continue to allocate a fresh UUID.
- Green/regression command: The focused new-schedule retry and existing-schedule replacement regressions passed.
- Refactor/notes: Idempotent identical-content retry remains intact without allowing changed bytes to collide with a staged object.

### Step 78: IR-021

- Write failing test: Added dismiss, Undo, dismiss again, then close the first snackbar while the second Undo action remains visible.
- Confirmed failure: The first snackbar's completion expired the second dismissal's Undo window.
- Implement: `dismiss` now returns a monotonically increasing dismissal generation and `expireUndo` acts only when the snackbar generation is still current.
- Green/regression command: The focused snackbar race and controller dismissal/expiry regressions passed.
- Refactor/notes: Undo behavior and session-long dismissal after the current window expires are unchanged.

### Step 79: IR-022

- Write failing tests: Added malformed external-thumbnail cases for missing/wrong `$type`, malformed/noncanonical CID, unknown blob fields, and unknown ref fields through the create handler.
- Confirmed failure: Malformed type/CID reached generated authoring and returned 500; noncanonical and unknown-field bodies could be accepted and written.
- Implement: Request validation now requires the exact blob/ref shape, `$type: blob`, a canonical CID, supported MIME, and bounded positive size before PDS work.
- Green/regression command: Focused handler tests passed with 4xx and zero PDS writes; `go test ./internal/api ./internal/linkpreview ./internal/scheduledposts -count=1`, focused race tests, and package vet passed.
- Refactor/notes: Generated `lexutil.LexBlob` construction remains defense in depth rather than the request-validation boundary.

### Step 80: IR-023

- Write failing tests: Added scheduled source identity cases for empty, userinfo, fragment, bare domain, unsupported scheme, malformed escaped host, and uppercase/default-port canonicalization.
- Confirmed failure: Invalid identities were accepted and noncanonical source values were persisted unchanged.
- Implement: Scheduled validation now parses an absolute HTTP(S) source without userinfo or fragment, lowercases scheme/host, removes default ports, preserves path/query, and stores the canonical fragmentless identity.
- Green/regression command: Focused source validation/canonicalization tests and nearby API/scheduled packages passed.
- Refactor/notes: The final navigation URI remains separately frozen and is not substituted for source identity.

### Step 81: IR-024

- Write failing evidence: Added a real `linkpreview.Service` lifecycle harness under actual handler logging, metrics, Sentry error, and transaction collectors.
- Confirmed missing evidence: The prior static service harness could not exercise DNS, pinned dialing, redirect, metadata parsing, image fallback/decode, DNS failure, or dial failure.
- Implement/evidence: An injected resolver and `net.Pipe` dialer script public answers and deterministic HTTP responses without public networking. The lifecycle covers redirect, parsed metadata, corrupt-image fallback, valid image decode, unsupported parse, DNS failure, and dial failure, then scans every applicable sink for URL/content/identity/dependency canaries.
- Green/regression command: Focused privacy tests, package tests, race tests, and the complete Go race suite passed.
- Refactor/notes: The test also asserts each required host/path was resolved/requested, preventing a false-positive empty-sink test.

### Step 82: Strict Blob / Scheduled Placeholder Interaction

- Write failing test: Added a unit-level scheduled external-thumbnail request that must pass post blob-shape validation and retain its media UUID.
- Confirmed failure: The internal scheduled placeholder lacked `$type` and used a non-CID link, so the stricter IR-022 validator rejected valid thumbnail schedules.
- Implement: The policy-only placeholder now uses the exact blob shape and a canonical CID before private media ownership/type/size validation runs.
- Green/regression command: The focused interaction test passed, and the complete Go race suite passed afterward.
- Refactor/notes: The placeholder is never published; publication still constructs the frozen record from the verified uploaded PDS blob.

### Third Correction Final Verification

- Deterministic generation: Two consecutive `just lexgen` runs produced identical generated `feedpost.go` SHA-256 `f5c666977e384757b4c069a2d7ab14b75fbc9e60c4a7ca5558a5c4f26ac8229a`.
- Go formatting/static checks: `just fmt` passed, including `gofmt` and `go vet ./...`.
- Go tests: Required-Postgres race-enabled `just test` passed the complete AppView suite.
- Flutter analysis: The first concurrent `just app-analyze` invocation contended with `just app-test` over Flutter's ephemeral iOS package directory. The isolated rerun passed with no issues.
- Flutter tests: `just app-test` passed all 1,540 tests.
- Independent analysis/audit: Dart MCP analysis reported no errors; `git diff --check` passed.
- Remaining release gates: `MAN-001`, `MAN-002`, and `MAN-003`; clean-tree `just lexgen-check` remains the pre-merge generated-drift gate.

## Third Correction Completion Checklist

- [x] IR-018 complete
- [x] IR-019 complete
- [x] IR-020 complete
- [x] IR-021 complete
- [x] IR-022 complete
- [x] IR-023 complete
- [x] IR-024 complete
- [x] Strict blob / scheduled placeholder interaction regression complete
- [x] Generation and full repository gates pass after third corrections
- [x] Final diff and `05-implementation-plan.md` readback complete after third corrections

## Post-Review UI Improvements

### Step 83: Link Preview Presentation and Interaction

- Write failing tests: Added widget regressions requiring the full published-card thumbnail above its copy, external-link confirmation before launch, an in-memory composer thumbnail above copy, an outlined composer surface without card elevation, and the shared themed snackbar payload for dismissal Undo.
- Confirmed failures: Full cards used a side thumbnail, card taps launched immediately, the composer ignored available thumbnail bytes and inherited `Card` shadow styling, and dismissal used a plain Material snackbar.
- Implement: Full published cards now use a bounded vertical image-first layout and call the shared `confirmAndLaunchExternalLink` flow. Compact quote cards retain their compact side-by-side variant. Composer previews render fetched bytes with a capped 16:9 image, neutral decode fallback, themed surface fill, visible outline, and no elevation. Dismissal uses `CraftskySnackBarContent`, info-surface coloring, and `MessageAction` while retaining generation-safe expiry.
- Green/regression commands: Focused card, composer, scheduling, feed, profile, quote, and thread suites passed. `flutter analyze` passed and the complete Flutter suite passed all 1,541 tests.
- Runtime evidence: Hot reload into the connected iPhone app succeeded and reported no runtime errors.

### Step 84: Scheduled Thumbnail Conflict Diagnostics

- Write failing tests: Required changed immutable-media retries to expose an allowlisted conflict stage and required the API warning to include that stage without owner DID, media UUID, object key, hash, or image content.
- Confirmed failure: AppView emitted only `error_class=conflict`, leaving reservation and identity/content mismatches indistinguishable.
- Implement: Scheduled-media conflicts now carry privacy-safe stages for invalid service input, digest/object-key construction, existing immutable-field mismatches, object-attempt reservation/identity, ready-blob mismatch, and media-row reservation. The handler logs `conflict_stage`; untyped or unlisted values become `unspecified`.
- Green/regression commands: Focused package tests, focused race tests, `go vet ./...`, `git diff --check`, and the complete race-enabled `just test` suite passed.
- Runtime evidence: Rebuilt the active AppView image through `scripts/compose-dev`; AppView is listening on the worktree's configured port with Firebase credentials mounted correctly.

### Step 85: Scheduled Object-Attempt Deadline Canonicalization

- Runtime diagnosis: A real thumbnail upload reported `object_attempt_identity`. The attempt was newly inserted in the same transaction, narrowing the mismatch to database-canonical attempt state.
- Root cause: The reservation compared a freshly computed nanosecond `remote_deadline` with PostgreSQL's microsecond-precision `timestamptz` value. The deadline is operational state, not immutable request identity.
- Implement: The stored deadline is now authoritative after attempt reservation. Remaining identity comparisons have distinct privacy-safe stages for owner, generation, media, object key, digest, and outcome mismatches.
- Regression evidence: The service test now uses `time.Now()` and requires the returned deadline to have database-canonical microsecond precision.
- Green/runtime commands: Focused AppView tests, focused race tests, `go vet ./...`, and `git diff --check` passed. The active AppView image was rebuilt and started successfully with the corrected service.
