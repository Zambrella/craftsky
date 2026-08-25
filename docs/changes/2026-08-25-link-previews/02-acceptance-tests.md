# Acceptance Test Specification: Link Previews (Website Card Embeds)

Status: Approved
Reviewer: User
Date: 2026-08-25

## 1. Test Strategy

Risk level: **High**. This feature changes a load-bearing post-record union, adds an SSRF-bearing outbound HTTP path, introduces asynchronous multi-link composer state, and extends private scheduled-media publication. Document review and explicit approval are required before implementation continues.

- Go unit tests cover URL policy, address classification, DNS pinning, redirect handling, metadata extraction, charset decoding, grapheme/byte clamps, image validation, request validation, response projection, configuration, and safe telemetry fields.
- Go integration tests use `httptest`, injected resolvers/dialers/transports, in-memory telemetry observers, and `internal/testdb.WithSchema`. Tests must never depend on public DNS or attempt real connections to forbidden addresses.
- Flutter unit/provider/widget tests use `ProviderContainer.test`, controllable completers, `DioAdapter`, existing composer harnesses, and fake launchers. They cover explicit URL boundaries, sequential work, stale-result races, selection, suppression, submission, drafts, schedules, rendering, and semantics.
- Lexicon and contract tests verify the standard `app.bsky.embed.external` record shape, generated union output, existing quote/no-embed compatibility, authoritative JSONB reads, and images-win shaping.
- Regression tests preserve link facets, images-plus-quote, posts without external embeds, existing scheduled media, local draft shape, and independent write/upload rate budgets.
- Manual checks are limited to deployed direct-egress enablement, real-device presentation/launching, and performance measurements that repository tests cannot establish reliably.

All timeouts, cancellation, clocks, DNS answers, socket destinations, and HTTP response streams must be injectable. Security tests assert both the returned result and whether any dial occurred. Async Flutter tests must use deterministic completers or fake time rather than wall-clock delays.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-013 | AT-001, AT-004, IT-001, IT-014 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-017 | AT-007, UT-012, UT-014, IT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002, AC-023 | UT-004, UT-005, UT-006, IT-004 | Unit / Integration | Yes |
| FR-003 | AC-003 | UT-007, UT-008, IT-004 | Unit / Integration | Yes |
| FR-004 | AC-006, AC-023 | UT-006, IT-004, MAN-003 | Unit / Integration / Manual | Yes, except deployed p95 measurement |
| FR-005 | AC-004, AC-005 | AT-007, UT-009, UT-010, UT-011, IT-002, IT-003, IT-019 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-007 | IT-008, REG-003 | Integration / Regression | Yes |
| FR-007 | AC-008, AC-014 | AT-003, UT-012, IT-007, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-008 | AC-009, AC-016, AC-021 | AT-004, UT-013, IT-009, IT-011, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-010 | AT-001, UT-001, UT-002, UT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-010 | AT-001, UT-001, UT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-011 | AT-002, UT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-012 | AT-003, IT-013, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-013 | AC-013 | AT-004, UT-015, IT-014, MAN-001 | Acceptance / Unit / Integration / Manual | Yes |
| FR-014 | AC-016 | AT-004, IT-011, MAN-001 | Acceptance / Integration / Manual | Yes |
| FR-015 | AC-015 | AT-007, UT-017, IT-006, IT-012, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-016 | AC-018 | AT-005, UT-018, IT-015, IT-016, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-017 | AC-019 | AT-006, UT-019, IT-017, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| FR-018 | AC-017 | AT-007, UT-016, IT-005, IT-019, MAN-002 | Acceptance / Unit / Integration / Manual | Yes |
| FR-019 | AC-020 | AT-008, UT-013, IT-010 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-006, AC-010 | UT-003, IT-004, IT-012, MAN-003 | Unit / Integration / Manual | Partial: responsiveness automated; deployed p95 is tracked as GAP-001 and measured manually |
| NFR-002 | AC-004, AC-005 | AT-007, UT-009, UT-010, UT-011, IT-002, IT-003, IT-019 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-010, AC-013 | AT-001, AT-004, IT-014, IT-020, MAN-001 | Acceptance / Integration / Manual | Yes, plus real-device smoke check |
| NFR-004 | AC-022 | AT-009, UT-020, IT-018 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-008, AC-014 | AT-002, UT-012, IT-007, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-002 | AC-014, AC-020 | AT-002, AT-008, UT-003, UT-012, UT-013, IT-007, IT-010, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-017 | AT-007, IT-005 | Acceptance / Integration | Yes |
| RULE-004 | AC-001, AC-005, AC-010 | AT-001, UT-001, UT-011, IT-001, IT-003, IT-012 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-008, AC-010 | AT-001, UT-003, IT-007, IT-012 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-005, AC-017 | AT-007, UT-016, IT-019, MAN-002 | Acceptance / Unit / Integration / Manual | Yes |

## 3. Acceptance Scenarios

### AT-001: Preview and select completed links

Requirement IDs: BR-001, FR-009, FR-010, NFR-001, NFR-003, RULE-004, RULE-005
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance / Widget
Automation Target: `app/test/feed/widgets/post_composer_sheet_link_previews_test.dart`

```gherkin
Feature: Link previews in the composer
  Scenario: Select among completed link candidates
    Given an image-free original-post composer is open
    When the member enters five links with explicit trailing boundaries
    Then the first eligible request starts without advancing a debounce timer
    And only the first four distinct normalized links are requested
    And no more than one request is active at once
    And failed candidates are omitted without an error
    And successful cards appear in text order
    And the first successful card is selected by default
    When the member selects another card and reorders the text
    Then selection follows the normalized URL rather than the carousel index
    When the selected source link is removed
    Then its card is removed and the earliest remaining success is selected

  Scenario: Equivalent fragment occurrences share metadata
    Given two occurrences normalize to one identity but have different fragments
    Then only one metadata request is made
    And the metadata request URL is fragmentless
    And AppView's validated final redirect URL remains the destination base
    And Flutter applies the first current source fragment only when that redirect URL has none
    When those occurrences are reordered or the first is removed
    Then the selected identity is retained and only the inherited fragment follows the new first occurrence without refetching
```

### AT-002: Suppress or dismiss preview work

Requirement IDs: FR-011, RULE-001, RULE-002
Acceptance Criteria: AC-011, AC-014
Priority: Must
Level: Acceptance / Widget
Automation Target: `app/test/feed/widgets/post_composer_sheet_link_previews_test.dart`, `app/test/projects/widgets/project_composer_validation_test.dart`

```gherkin
Feature: Link-preview suppression
  Scenario: Images suppress previews and removal restores them
    Given the composer has cached and queued link candidates
    When the member adds a post image
    Then visible previews disappear and pending preview work is cancelled
    When the member removes all post images
    Then cached cards and eligible queued work return

  Scenario: Dismiss previews for the composer session
    Given one or more preview cards are visible
    When the member dismisses the carousel
    Then pending work is cancelled and no preview is attached or newly fetched
    And a short "Link previews hidden" snackbar offers Undo
    When Undo is activated before expiry
    Then eligible cached and queued previews are restored

  Scenario: Undo expires
    Given the member dismissed previews and did not activate Undo
    When the injected snackbar duration expires
    Then dismissal remains in force for that composer session
    And opening a new composer session restores normal preview eligibility

  Scenario: Quote composition suppresses external previews
    Given a quote composer contains a completed URL
    Then no link-preview request or external embed is offered

  Scenario: Project composition remains photo-required
    Given a project composer contains a completed URL and its required photo
    Then no link-preview request or external embed is offered
    And removing every photo leaves project submission invalid rather than enabling previews
```

### AT-003: Submit without waiting and preserve thumbnail failures

Requirement IDs: FR-007, FR-012
Acceptance Criteria: AC-008, AC-012, AC-014
Priority: Must
Level: Acceptance / Widget
Automation Target: `app/test/feed/widgets/post_composer_link_preview_submission_test.dart`, `app/test/feed/composer/composer_submission_coordinator_test.dart`

```gherkin
Feature: Publish a post with an optional external card
  Scenario: Submit while preview work is pending
    Given a preview request is still pending
    When the member submits the post
    Then post creation starts immediately without an external embed
    And a late preview response is ignored

  Scenario: Submit a selected card with a thumbnail
    Given a successful selected card has a thumbnail
    When the member submits the post
    Then the thumbnail is uploaded before post creation
    And the create request contains one standard external embed with the uploaded blob

  Scenario: Selected thumbnail upload fails
    Given a successful selected card has a thumbnail
    When its thumbnail upload fails
    Then no post is created
    And the composer text and selected card remain intact
    And the existing post failure feedback is shown
```

### AT-004: Read and open external cards on every post surface

Requirement IDs: BR-001, FR-008, FR-013, FR-014, NFR-003
Acceptance Criteria: AC-009, AC-013, AC-016, AC-021
Priority: Must
Level: Acceptance / Widget
Automation Target: `app/test/feed/widgets/external_card_test.dart`, `app/test/feed/widgets/post_card_test.dart`, `app/test/shared/widgets/post_summary_test.dart`

```gherkin
Feature: External-card reading
  Scenario Outline: Render a full external card
    Given a visible <surface> post has an external embed and no images
    When the post renders at narrow and wide widths
    Then its fixed-frame optional thumbnail, title, description, and host are legible
    And tapping the card asks the existing launcher to open the final URI

    Examples:
      | surface            |
      | feed               |
      | profile posts      |
      | comment or reply   |
      | post detail        |

  Scenario: Render an external card inside a visible quote
    Given a quoted post has an external embed and no images
    Then its compact quote view renders the external card
    And existing hidden or unavailable quote behavior is unchanged
```

### AT-005: Freeze and publish a scheduled external card

Requirement IDs: FR-016
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/scheduled_posts/scheduled_post_link_preview_test.dart`, `appview/internal/scheduledposts/worker_acceptance_test.go`

```gherkin
Feature: Scheduled link previews
  Scenario Outline: Save, edit, and publish a standard scheduled external card
    Given a standard scheduled post has selected frozen external metadata with <thumbnail>
    When the member saves the schedule
    Then an optional thumbnail is staged privately before schedule creation when present
    And the scheduled payload retains the frozen external metadata
    When the schedule is reopened while its source URL remains
    Then the selected card is restored without refreshing it
    And only other or newly eligible candidates may be fetched
    When publication becomes due
    Then the worker uploads the staged thumbnail when present and writes the same frozen standard embed
    And no metadata refetch occurs during publication or retry

    Examples:
      | thumbnail |
      | present   |
      | absent    |

  Scenario: Project schedules remain image-required
    Given the project composer is scheduling a post
    Then its existing required-photo validation remains active
    And no external-card field is accepted or staged
```

### AT-006: Reopen a local draft as a new preview session

Requirement IDs: FR-017
Acceptance Criteria: AC-019
Priority: Must
Level: Acceptance / Widget
Automation Target: `app/test/drafts/link_preview_draft_test.dart`, `app/test/drafts/privacy/draft_privacy_test.dart`

```gherkin
Feature: Link previews and local drafts
  Scenario: Reopen a draft containing completed URLs
    Given a composer session has cached, selected, or dismissed previews
    When the post is saved as a local draft and the composer closes
    Then no preview metadata, bytes, selection, cache, or dismissal is persisted
    When the draft is reopened
    Then its text is restored
    And a new preview session derives and refetches eligible candidates
```

### AT-007: Enforce the authenticated bounded egress route

Requirement IDs: FR-001, FR-005, FR-015, FR-018, NFR-002, RULE-003, RULE-006
Acceptance Criteria: AC-004, AC-005, AC-015, AC-017
Priority: Must
Level: Acceptance / Integration
Automation Target: `appview/internal/api/link_preview_test.go`, `appview/internal/routes/routes_test.go`

```gherkin
Feature: Secure link-preview endpoint
  Scenario Outline: Reject before outbound work
    Given the preview fetcher records every outbound attempt
    When a request is rejected for <reason>
    Then the standard error envelope is returned
    And no outbound dial occurs

    Examples:
      | reason                         |
      | missing or invalid session     |
      | missing or invalid device ID   |
      | production feature disabled    |
      | token hourly budget exhausted  |
      | device hourly budget exhausted |
      | forbidden URL or address       |

  Scenario: Fetch an eligible public page
    Given previews are enabled and auth, device, and dedicated budgets are valid
    When a public HTTP or HTTPS URL is submitted
    Then AppView validates and pins each connection and redirect
    And returns bounded metadata through the singular camelCase contract
```

### AT-008: Apply images-win to federated records

Requirement IDs: FR-019, RULE-002
Acceptance Criteria: AC-020
Priority: Must
Level: Acceptance / Integration
Automation Target: `appview/internal/index/craftsky_post_test.go`, `appview/internal/api/post_response_test.go`

```gherkin
Feature: Federated images and external embeds
  Scenario: Index a record containing both
    Given Tap delivers a CraftSky post record with top-level images and a standard external embed
    When the indexer handles the event
    Then the authoritative record is retained and indexed
    And every CraftSky read view returns the images
    And every CraftSky read view omits the external card
```

### AT-009: Keep preview intent out of telemetry

Requirement IDs: NFR-004
Acceptance Criteria: AC-022
Priority: Must
Level: Acceptance / Integration
Automation Target: `appview/internal/api/link_preview_privacy_test.go`, `app/test/observability/link_preview_privacy_test.dart`

```gherkin
Feature: Link-preview privacy
  Scenario: Exercise success and failure with privacy canaries
    Given unique canaries appear in URL, hostname, query, redirect, metadata, thumbnail bytes, post text, DID, token, and device identity
    When preview success, validation, auth, rate, timeout, upstream, scheduling, and publication paths run
    Then logs, metrics, traces, Sentry events, breadcrumbs, and client analytics contain none of the canaries
    And emitted operational fields use only approved bounded categories and buckets
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-009, FR-010, RULE-004 | AC-001, AC-010 | Normalize candidate identity/transport URL separately from deterministic navigation URI. | Scheme/host casing, default ports, paths/queries, bare domains, duplicate occurrences with distinct/absent fragments before/after reorder/removal, and AppView final URLs with/without redirect fragments. | Identity/transport URL adds HTTPS when absent, lowercases scheme/host, removes default port/fragment, and preserves path/query. AppView's final URL remains the base; its fragment wins when present, otherwise Flutter applies the first current source fragment and updates it without a metadata refetch. | `app/test/feed/composer/link_preview_candidate_test.dart` |
| UT-002 | FR-009 | AC-010 | Derive only completed ordered candidates from shared link tokens. | End-of-draft URL, whitespace/punctuation boundaries, duplicates, and five URLs. | Final unfinished token is excluded; first four distinct completed normalized URLs remain in text order. | `app/test/shared/rich_text/link_preview_candidates_test.dart` |
| UT-003 | FR-009, FR-010, FR-011, NFR-001, RULE-002, RULE-005 | AC-010, AC-011, AC-014 | Drive deterministic preview-session state, timing, and races. | Fake time, controllable fetch completers, reorder/remove/add edits, failures, image toggles, dismiss/Undo/expiry, disposal/account change. | First eligible fetch is invoked without advancing time; one request remains active; stale results are ignored; success/failure is cached; selection follows identity; Undo expiry leaves dismissal active; suppression/cancellation/restoration and input updates are synchronous. | `app/test/feed/composer/link_preview_controller_test.dart` |
| UT-004 | FR-002 | AC-002 | Select title, description, and image candidates independently. | Mixed non-empty/empty OG, Twitter, HTML, and hostname metadata in varied order. | Each field follows its own fallback chain; canonical and `og:url` do not affect destination. | `appview/internal/linkpreview/metadata_test.go` |
| UT-005 | FR-002 | AC-002 | Normalize and clamp Unicode metadata safely. | Repeated whitespace, combining marks, emoji graphemes, values crossing grapheme and byte limits. | Whitespace collapses; title and description satisfy both limits without splitting graphemes or producing invalid UTF-8. | `appview/internal/linkpreview/metadata_test.go` |
| UT-006 | FR-002, FR-004 | AC-006, AC-023 | Decode bounded HTML/XHTML head metadata. | UTF-8 and supported legacy charsets, malformed declarations, early `</head>`, no close tag, and >2 MB streams. | Supported text decodes; parsing stops at completed head; raw reads never exceed 2 MB; oversize/unsupported input fails safely. | `appview/internal/linkpreview/metadata_test.go` |
| UT-007 | FR-003 | AC-003 | Resolve and order thumbnail candidates. | First valid/invalid `<base>`, relative/cross-origin/query-signed OG and Twitter candidates, duplicates, four candidates. | First valid base is honored; at most three distinct candidates are returned in priority/document order without mutation. | `appview/internal/linkpreview/metadata_test.go` |
| UT-008 | FR-003 | AC-003 | Validate thumbnail bytes from decoded content. | Valid JPEG/PNG/WebP; spoofed MIME, corrupt/truncated, GIF/AVIF/SVG, >1,000,000 bytes, 8,193 px side, >16 MP. | Only fully decoded supported images within every limit pass; MIME and dimensions come from bytes; failures omit thumbnail. | `appview/internal/linkpreview/image_test.go` |
| UT-009 | FR-005, NFR-002 | AC-004 | Validate URL syntax before DNS or dialing. | HTTP/HTTPS, FTP/file/data, userinfo, ports 80/443/default/other, malformed host, public/non-public literals. | Only HTTP(S) URLs without credentials, with permitted ports and public literals pass; rejected inputs cause no resolution or dial. | `appview/internal/linkpreview/transport_test.go` |
| UT-010 | FR-005, NFR-002 | AC-004, AC-005 | Classify complete IPv4/IPv6 answer sets. | Public, loopback, private, link-local, multicast, unspecified, documentation, benchmark, carrier-grade, reserved, IPv4-mapped IPv6, and mixed sets. | Every special/non-public or mixed set is rejected; all-public sets retain validated addresses for pinning. | `appview/internal/linkpreview/addresses_test.go` |
| UT-011 | FR-005, RULE-004 | AC-001, AC-005 | Apply redirect limits and fragment destination rules. | No redirect, source fragment, redirect with/without fragment, five and six hops, forbidden next hop. | Every hop is revalidated; at most five redirects; source fragment survives unless redirect supplies one; final destination never comes from page metadata. | `appview/internal/linkpreview/fetcher_test.go` |
| UT-012 | FR-001, FR-007, RULE-001, RULE-002 | AC-008, AC-014, AC-017 | Strictly decode and validate preview/create-post request shapes. | Missing/extra/wrong-type URL, >8,192-byte URL, valid/oversized metadata, valid/invalid thumb, external+quote, external+images, external+project. | Strict camelCase validation accepts valid singular preview/external values and rejects all invalid/conflicting combinations before external or PDS work. | `appview/internal/api/link_preview_request_test.go`, `appview/internal/api/post_request_test.go` |
| UT-013 | FR-008, FR-019, RULE-002 | AC-009, AC-016, AC-020, AC-021 | Shape external data from authoritative records. | Metadata-only, thumbnail, compact quote, absent external, and federated images+external records. | Shared responses expose the agreed camelCase external shape and synthesized thumb URL; images-win omits external; no extra query is required. | `appview/internal/api/post_response_test.go` |
| UT-014 | FR-001, FR-007, FR-008 | AC-001, AC-008, AC-009 | Encode/decode Flutter preview, external API, and model shapes. | Preview response with valid padded RFC 4648 base64 at decoded-size boundaries, invalid base64/overflow, full/compact post response, metadata-only card, and uploaded thumbnail. | Valid base64 decodes to exact bytes at or below 1,000,000 bytes; malformed or oversized decoded data fails safely; other camelCase fields map without loss. | `app/test/feed/data/link_preview_api_client_test.dart`, `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/models/post_test.dart` |
| UT-015 | FR-013 | AC-013 | Derive safe external-card presentation. | Final URI with host casing/port/fragment, empty description, no thumbnail, long text. | Display host is bounded; final URI remains tap destination; optional regions collapse without changing card semantics. | `app/test/feed/widgets/external_card_test.dart` |
| UT-016 | FR-018, RULE-006 | AC-017 | Parse link-preview enablement and direct-egress configuration. | Dev/test/prod environments; unset/true/false flag; proxy environment variables. | Dev/test default enabled; production requires explicit true; generic proxy settings are never selected for preview transport. | `appview/internal/app/config_test.go`, `appview/internal/linkpreview/transport_test.go` |
| UT-017 | FR-015 | AC-015 | Define dedicated hourly token/device budgets. | Boundary counts 59/60/61 and 119/120/121 with injected clock rollover. | Limits are 60/hour token and 120/hour device, reset at the correct boundary, and use the `link_preview` class only. | `appview/internal/middleware/rate_limit_test.go` |
| UT-018 | FR-016 | AC-018 | Round-trip frozen external state for standard schedules and reject it for project schedules. | Standard payloads with metadata-only, thumbnail media, absent external, and edits where source remains/changes; project payload with external. | Standard codec preserves exact frozen metadata/private media and retains selected state only while its source remains; project external is rejected without weakening existing project fields or image requirements. | `appview/internal/scheduledposts/payload_test.go`, `appview/internal/scheduledposts/validation_test.go`, `app/test/scheduled_posts/scheduled_post_model_test.dart` |
| UT-019 | FR-017 | AC-019 | Exclude preview session state from local draft serialization. | Preview metadata/bytes/cache/selection/dismissal alongside normal draft text and media. | Draft codec persists only existing draft fields; reopened state is initialized from text as a new session. | `app/test/drafts/privacy/draft_privacy_test.dart` |
| UT-020 | NFR-004 | AC-022 | Restrict preview telemetry to bounded fields. | Safe stage/result/error/status/redirect-count/byte/duration values plus forbidden URL/content/identity attributes. | Only approved bounded values survive validation; dependency errors and user-derived strings cannot become telemetry attributes. | `appview/internal/observability/link_preview_test.go`, `app/test/observability/link_preview_privacy_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, RULE-004 | AC-001 | Serve the singular authenticated preview contract. | Authenticated `httptest` handler with scripted safe fetcher and valid HTML/image fixtures. | POST one `{url}` containing a source fragment, follow redirects with/without their own fragments, then decode optional thumbnail JSON. | Fetch target omits the input fragment; response uses camelCase and contains only the validated final redirect URL/fragment, normalized metadata, padded RFC 4648 base64 that round-trips exact thumbnail bytes within the decoded cap, MIME/dimensions, and standard request ID behavior. | `appview/internal/api/link_preview_test.go` |
| IT-002 | FR-005, NFR-002 | AC-004, AC-005 | Pin an allowed connection and reject forbidden answer sets without dialing. | Injected resolver and recording dialer with public, forbidden, mixed, and rebinding scripts. | Fetch each URL and inspect resolution/dial sequence. | Rejections perform zero forbidden dials; allowed requests dial an exact prevalidated IP and original host remains available for TLS/HTTP semantics. | `appview/internal/linkpreview/transport_integration_test.go` |
| IT-003 | FR-005, NFR-002, RULE-004 | AC-001, AC-005 | Revalidate redirects and DNS changes at every resource hop. | Scripted page/image redirect servers behind injected resolver/dialer; fragment variants and six-hop chain. | Fetch page and thumbnail chains including public-to-private and changed-answer attacks. | Every hop is independently validated/pinned, forbidden and over-cap chains fail, and valid final URI obeys fragment rules. | `appview/internal/linkpreview/transport_integration_test.go` |
| IT-004 | FR-002, FR-003, FR-004, NFR-001 | AC-002, AC-003, AC-006, AC-023 | Enforce response type, stream, phase, total-budget, charset, and image fallback behavior together. | Fake clock/context plus scripted chunked page and up to three image responses. | Exercise non-2xx/non-HTML responses, HTML/XHTML, early head, >2 MB stream, slow page, slow/corrupt/oversized images, and legacy charsets. | Only 2xx HTML/XHTML pages are parsed; page bounds fail safely; successful metadata survives thumbnail exhaustion; no fourth image is fetched; request respects the 10-second total budget. | `appview/internal/linkpreview/fetcher_test.go` |
| IT-005 | FR-001, FR-018, RULE-003 | AC-017 | Register route policy and reject auth/device/disabled cases before fetch. | Full route registry/mux and fetcher call counter across environment configs. | Call route with missing/invalid auth/device, disabled production, wrong method/body, and valid identity. | Standard envelopes and 503 `link_preview_unavailable` are returned as applicable; fetcher runs only for fully admitted requests. | `appview/internal/routes/routes_test.go`, `appview/internal/api/link_preview_test.go` |
| IT-006 | FR-015 | AC-015 | Enforce independent token and device preview budgets. | Middleware with fake clock, two tokens/devices, write/upload counters, and handler call counter. | Cross each preview limit and advance the hour boundary. | Standard 429/`Retry-After` occurs before handler work; token/device isolation holds; write/upload budgets are unchanged. | `appview/internal/middleware/rate_limit_test.go`, `appview/internal/routes/routes_test.go` |
| IT-007 | FR-007, RULE-001, RULE-002, RULE-005 | AC-008, AC-014 | Write valid standard external records and reject conflicts without PDS writes. | Authenticated create-post handler and recording fake PDS. | Submit metadata-only/thumb external, no matching facet, quote+external, images+external, and project+external. | Valid standard image-free request writes exact `$type`/external/blob shape; no facet coupling is required; conflicts return 4xx with zero PDS writes. | `appview/internal/api/post_test.go`, `appview/internal/api/post_request_test.go` |
| IT-008 | FR-006 | AC-007 | Record governance, generate, and validate the external union variant. | Numbered ADR under `adr/`, recorded `atproto-lexicon` review, updated lexicon/build inputs, and generated package. | Verify the ADR/review evidence, run generation/drift checks, and decode/validate standard external, quote, and absent embed fixtures. | Governance artifacts identify standard external reuse and review; `app.bsky.embed.external` maps to the pinned indigo type; generated code is stable; existing records remain valid. | `adr/009-*.md`, `just lexgen`, `just lexgen-check`, `appview/internal/lexicon/craftsky/feedpost_test.go` |
| IT-009 | FR-008 | AC-009, AC-021 | Project external data from authoritative JSONB on canonical reads. | `testdb.WithSchema` posts with metadata-only and thumbnail external records. | Query timeline, profile, detail, and comment/reply post-shaped stores. | Shared projection/scan returns the same external shape without a redundant column, migration, or per-row lookup. | `appview/internal/api/post_store_test.go`, `appview/internal/api/timeline_test.go` |
| IT-010 | FR-019, RULE-002 | AC-020 | Preserve federated external records while applying images-win. | Real Postgres indexer fixture with create/update/replay/delete events containing images+external. | Handle events and read canonical surfaces. | Raw record retains both; reads expose images only; replay is idempotent; update/delete convergence remains intact. | `appview/internal/index/craftsky_post_test.go`, `appview/internal/index/craftsky_post_import_test.go` |
| IT-011 | FR-008, FR-014 | AC-016 | Hydrate compact quote external data without changing visibility policy. | Visible, hidden, unavailable, metadata-only, thumbnail, and images+external quote fixtures. | Build timeline/detail quote views and decode in Flutter. | Visible eligible quote includes compact external; hidden/unavailable behavior is unchanged; images-win applies. | `appview/internal/api/post_response_test.go`, `appview/internal/api/timeline_test.go`, `app/test/feed/models/post_test.dart` |
| IT-012 | FR-009, FR-010, FR-011, FR-015, NFR-001, RULE-004, RULE-005 | AC-010, AC-011, AC-015 | Connect composer state to singular fragmentless API calls under edit races. | Riverpod container, fake time, fake preview repository with controllable completions/cancellations, AppView final-URL fragment fixtures, and account/session changes. | Type an explicit completion boundary, reorder/remove duplicate source fragments, suppress, dismiss/undo/expire, submit, and dispose while requests are pending. | First eligible fragmentless call starts without advancing fake time; calls remain sequential and bounded; Flutter applies only an inherited source fragment from the first occurrence while AppView redirect-fragment precedence holds; stale/account-old results never attach; failures/429 are silent; no input or scroll operation waits on network work. | `app/test/feed/composer/link_preview_controller_test.dart`, `app/test/feed/data/link_preview_api_client_test.dart` |
| IT-013 | FR-012 | AC-012 | Integrate thumbnail upload with immediate post submission. | Composer coordinator, fake media uploader/repository, selected metadata-only/thumb card, pending request, and injected failures. | Submit each state and retry after upload failure. | Pending/failed/absent cards omit external; metadata-only needs no upload; thumb uploads first; failure creates no post and retains state; success forwards exact external. | `app/test/feed/composer/composer_submission_coordinator_test.dart`, `app/test/feed/composer/composer_media_uploader_test.dart` |
| IT-014 | BR-001, FR-013, NFR-003 | AC-013 | Render external cards through all full post surfaces. | Existing post harnesses, narrow/wide constraints, large text, metadata-only/thumb cards, fake launcher. | Pump feed/profile/comment/detail surfaces and tap cards. | Shared responsive card renders without overflow, exposes semantics, and launches the exact final URI; no direct third-party fetch occurs. | `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/widgets/profile_posts_tab_test.dart`, `app/test/feed/pages/post_thread_page_test.dart`, `app/test/feed/pages/post_comment_section_page_test.dart` |
| IT-015 | FR-016 | AC-018 | Stage standard scheduled thumbnails atomically and reject project external fields. | Flutter scheduled repository/API fakes and AppView private-media handler/store. | Save standard metadata-only/thumbnail schedules with injected stage/create failures, then submit a project schedule carrying external. | Standard thumbnail is owner-private and claimed only by successful schedule creation; failure creates no schedule and preserves composer/card; project external is rejected before staging while ordinary project scheduling remains valid. | `app/test/scheduled_posts/scheduled_post_link_preview_test.dart`, `app/test/projects/widgets/project_composer_validation_test.dart`, `appview/internal/api/scheduled_media_test.go`, `appview/internal/api/scheduled_post_test.go` |
| IT-016 | FR-016 | AC-018 | Publish exact frozen standard external cards through retries and recovery. | Durable standard schedules with/without thumbnails, fake private store/PDS, side-effect barriers, and fetcher call counter. | Publish each standard variant, fail/retry, and recover at each external side-effect boundary. | Metadata never refreshes; staged thumbnails upload safely; frozen record identity/body remains identical; no preview fetch occurs; existing retry/Needs-attention and cleanup behavior applies. | `appview/internal/scheduledposts/publication_processor_test.go`, `worker_acceptance_test.go`, `recovery_acceptance_test.go`, `failure_acceptance_test.go` |
| IT-017 | FR-017 | AC-019 | Reopen local drafts without persisted preview state. | Existing draft codec/repository and fake preview repository. | Save/reopen drafts after success, failure, selection, and dismissal states. | Disk representation contains none of those states; restored text starts a new session and causes eligible refetches only after explicit boundaries. | `app/test/drafts/link_preview_draft_test.dart`, `app/test/drafts/privacy/draft_privacy_test.dart` |
| IT-018 | NFR-004 | AC-022 | Scan cross-pillar diagnostics for preview privacy canaries. | In-memory slog, metrics, trace/Sentry transports, Flutter report/event collectors, and canary inputs for every sensitive category. | Exercise auth/rate/disabled, validation, DNS, redirect, parse, image, success, post upload, schedule, worker retry, and cleanup paths. | No canary or identity appears; only approved stage/result/error/status classes and count/byte/duration buckets appear. | `appview/internal/api/link_preview_privacy_test.go`, `appview/internal/observability/link_preview_test.go`, `app/test/observability/link_preview_privacy_test.dart` |
| IT-019 | FR-005, FR-018, NFR-002, RULE-006 | AC-005, AC-017 | Prove direct pinned transport ignores generic proxy configuration and forwards no sensitive headers. | Set `HTTP_PROXY`/`HTTPS_PROXY`; recording proxy and direct dialers; incoming auth/cookie/referer/user-content headers. | Fetch an eligible page/image. | Proxy receives no request; pinned direct dialer is used; outbound headers contain no credentials, cookies, referer, or user content and use only fixed client headers. | `appview/internal/linkpreview/transport_integration_test.go` |
| IT-020 | NFR-003 | AC-010, AC-013 | Guard the client-only/AppView-only network boundary. | Repository import/dependency scan plus Flutter tests with only injected preview API repository. | Scan production Flutter preview code and run composer/render suites with third-party network disabled. | Flutter references only AppView API abstractions for preview metadata; renderer performs no fetch; all platform-neutral tests pass without external access. | `app/test/architecture/link_preview_network_boundary_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | URLs remain ordinary tappable facets when previews are absent, pending, failed, dismissed, rate-limited, or omitted at submit. | FR-012 | AC-012 | Extend facet/composer/action tests to assert link facets and external launching remain unchanged without an attached card. |
| REG-002 | Existing quote embeds remain valid, images may still coexist with quote embeds, and project posts remain photo-required; create conflicts produce no PDS write. | FR-007, RULE-001, RULE-002 | AC-008, AC-014 | Run quote/image/project request and PDS-write tests; add explicit images+quote accepted and quote/images/project+external rejected cases. |
| REG-003 | Existing quote and no-embed post records remain valid after lexicon generation. | FR-006 | AC-007 | Keep generated union decode/CBOR/JSON fixtures for absent and local quote variants and run `just lexgen-check`. |
| REG-004 | Canonical read surfaces without external retain their current JSON shape and query behavior. | FR-008 | AC-009, AC-016, AC-021 | Run existing post/timeline/profile/comment/detail response/store suites with absent external and compare unchanged optional-field behavior. |
| REG-005 | Existing scheduled text/image/project publication, retries, recovery, and cleanup remain unchanged. | FR-016 | AC-018 | Run all current `internal/scheduledposts` and Flutter scheduled-post suites; external absence follows the old payload/media path. |
| REG-006 | Local draft format and existing text/image restoration remain unchanged. | FR-017 | AC-019 | Run existing draft codec/privacy tests and assert no schema expansion is required for preview state. |
| REG-007 | Preview traffic does not consume or alter write/upload quotas. | FR-015 | AC-015 | Exercise preview, write, and upload classes together at each boundary and assert independent counters and existing defaults. |
| REG-008 | Index create/update/replay/delete remains idempotent with external data in raw record JSONB. | FR-008, FR-019 | AC-009, AC-020, AC-021 | Extend `craftsky_post_test.go` and import tests with external fixtures through full lifecycle events. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Candidate normalization and ordering. | HTTP/HTTPS/bare-domain URLs varying host/scheme case, default ports, fragments, path/query, punctuation boundaries, duplicates, and five distinct links. | AT-001, UT-001, UT-002, UT-003, IT-012 |
| TD-002 | Metadata fallback and Unicode clamps. | HTML mixing OG/Twitter/title/description, empty tags, canonical/`og:url`, whitespace, combining marks, emoji, and values crossing both grapheme and byte limits. | UT-004, UT-005, IT-004 |
| TD-003 | Charset and bounded-head streams. | UTF-8, Windows-1252, and another supported legacy charset HTML/XHTML; early `</head>`; chunked no-head-close stream exceeding 2 MB. | UT-006, IT-004 |
| TD-004 | Thumbnail corpus and JSON wire representation. | Generated valid JPEG/PNG/WebP; corrupt/truncated bytes; spoofed MIME; GIF/AVIF/SVG; exact and over byte/dimension/area boundaries; valid padded RFC 4648 base64, malformed base64, and decoded-size overflow. | UT-007, UT-008, UT-014, IT-001, IT-004 |
| TD-005 | SSRF address matrix. | Public IPv4/IPv6 plus loopback, private, link-local, multicast, unspecified, documentation, benchmark, carrier-grade, reserved, IPv4-mapped, mixed, and rebinding answer scripts. | AT-007, UT-009, UT-010, IT-002, IT-003 |
| TD-006 | Redirect and timeout scripts. | Zero/five/six hops; fragment/no-fragment redirects; public-to-private redirect; slow headers/body/image; cancellation at each phase. | UT-011, IT-003, IT-004 |
| TD-007 | Standard external record fixtures. | Metadata-only and thumb records with `$type: app.bsky.embed.external`; local quote; no embed; deliberately federated images+external raw record. | IT-007 through IT-011, REG-002 through REG-004, REG-008 |
| TD-008 | Composer race and occurrence controls. | Fake time, completers for four queued successes/failures/429s, duplicate identities with distinct fragments, cancellation recorders, image/quote toggles, Undo expiry, account switch, submit, and disposal. | AT-001 through AT-003, UT-001, UT-003, IT-012, IT-013 |
| TD-009 | Scheduled external fixtures. | Standard frozen metadata-only/thumbnail payloads, project payload with forbidden external, owner-private media IDs/bytes, stable TID/createdAt, retry and crash barriers. | AT-005, UT-018, IT-015, IT-016, REG-005 |
| TD-010 | Privacy canaries. | Unique URL, hostname, query, redirect, title, description, thumbnail bytes, post text, DID, handle, bearer token, device ID, and dependency error strings. | AT-009, UT-020, IT-018 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-013, FR-014, NFR-003 | AC-013, AC-016 | Real-device card presentation, accessibility, and launch smoke check. | On supported mobile/desktop/web targets, inspect full and compact cards at narrow width, large text, light/dark themes, keyboard/screen-reader navigation, then tap metadata-only and thumbnail cards. | No clipping or inaccessible controls; carousel/card semantics are meaningful; final URI opens externally or fails through existing safe launcher behavior. |
| MAN-002 | FR-018, RULE-006 | AC-005, AC-017 | Production direct-egress release gate. | In a production-like environment, verify previews are disabled by default, enable `LINK_PREVIEWS_ENABLED=true` only with reviewed direct pinned egress, and attempt a known public page plus blocked targets. | Disabled route returns 503 before egress; enabled route reaches only validated public 80/443 destinations and does not traverse a generic proxy. |
| MAN-003 | FR-004, NFR-001 | AC-006 | Measure reachable-page p95 in a controlled staging workload. | Run a representative bounded corpus through the production-like direct-egress path while recording only privacy-safe durations/outcomes. | Reachable page phases complete within 5 seconds at p95; any misses are investigated without logging target identity or content. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | The 5-second page-phase p95 cannot be established by deterministic repository tests. | NFR-001 | Fake-clock tests prove bounds and non-blocking UI, not production DNS/network latency distributions. | Treat MAN-003 as a production-enablement check and retain privacy-safe duration metrics. |
| GAP-002 | Exact typed error codes for all preview failures remain a coding-plan detail. | FR-001, FR-005, FR-018 | Requirements fix standard envelopes, base64 behavior, and known disabled/rate behavior but intentionally leave the complete taxonomy open. | Finalize a bounded taxonomy during coding planning and parameterize IT-001/IT-005 without changing acceptance semantics. |
| GAP-003 | Exact fixed User-Agent and `Accept-Language` policy remain open. | FR-005, NFR-004, RULE-006 | These values do not change the required no-user-data/direct-egress behavior but affect exact header assertions. | Choose a stable CraftSky User-Agent/contact and omit or fix `Accept-Language` during coding planning; update IT-019 fixtures. |
| GAP-004 | Real DNS rebinding is intentionally not exercised against the public internet. | FR-005, NFR-002 | Public-network tests are flaky and could create unsafe traffic. | Use injected resolver answer changes and exact dial assertions in IT-002/IT-003; inspect production transport wiring during implementation review. |
| GAP-005 | Full assistive-technology behavior and platform URL-launch integration are not completely represented by widget tests. | FR-013, FR-014, NFR-003 | Flutter widget semantics and launcher fakes cannot emulate every target shell. | Run MAN-001 before release and retain automated semantics/layout tests as the merge gate. |

## 10. Out Of Scope

- oEmbed, video players, PDF/JSON/image-document previews, favicons, JavaScript execution, robots.txt, and social-platform-specific embeds.
- Client-side cross-origin page or thumbnail fetching.
- Persistent/cross-user preview caching and metadata refresh at read or publication time.
- Metadata editing, image resizing/transcoding, and unsupported-image recovery.
- URL-shortener special behavior, canonical/`og:url` destination overrides, and generic proxy support.
- New analytics, moderation, search, notification, or account-deletion behavior.
- Public database migration tests because authoritative external data remains in existing `craftsky_posts.record JSONB`.
- Live third-party website conformance as a merge-gating automated suite.
- Creating implementation or executable test files during this test-design stage.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-08-25-link-previews/01-requirements.md`
- Test specification: `docs/changes/2026-08-25-link-previews/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-08-25-link-previews/03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-25-link-previews/`
- Recommended first failing test for implementation: `UT-009` in `appview/internal/linkpreview/transport_test.go`, starting with rejection of forbidden schemes, userinfo, nonstandard ports, and non-public literals before resolution or dialing.
- Suggested test order for implementation:
  1. `UT-009`, `UT-010`, `UT-011`, `IT-002`, `IT-003`, and `IT-019` for the secure outbound boundary.
  2. `UT-004` through `UT-008` and `IT-004` for bounded extraction and thumbnail handling.
  3. `UT-016`, `UT-017`, `IT-001`, `IT-005`, `IT-006`, and `AT-007` for config, route, auth/device, and rate policy.
  4. `IT-008`, `UT-012` through `UT-014`, and `IT-007` through `IT-011` for lexicon, write, index, and read contracts.
  5. `UT-001` through `UT-003`, `IT-012`, `AT-001`, and `AT-002` for Flutter candidate/session behavior.
  6. `IT-013`, `AT-003`, `UT-015`, `IT-014`, and `AT-004` for immediate submission and rendering.
  7. `UT-018`, `IT-015`, `IT-016`, and `AT-005` for scheduled publication.
  8. `UT-019`, `IT-017`, and `AT-006` for local drafts.
  9. `UT-020`, `IT-018`, `AT-009`, then `REG-001` through `REG-008` for privacy and regression closure.
- Commands discovered:
  - Start required services from repo root: `just dev-d`.
  - Run the complete Go suite with race detection, real Postgres, and MinIO: `just test`.
  - Run focused Go packages from `appview/`: `go test ./internal/linkpreview ./internal/api ./internal/routes ./internal/middleware ./internal/index ./internal/scheduledposts -count=1`.
  - Regenerate and check lexicon output from repo root: `just lexgen`; `just lexgen-check`.
  - Run all Flutter tests from repo root: `just app-test`.
  - Run focused Flutter tests from repo root: `just app-test test/feed/composer test/feed/widgets test/scheduled_posts test/drafts`.
  - Run Flutter static analysis from repo root: `just app-analyze`.
- Blocking gaps: None for document review. GAP-002 and GAP-003 must be resolved in the coding plan before their exact contract/header assertions are implemented.
