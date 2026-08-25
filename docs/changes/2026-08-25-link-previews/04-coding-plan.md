# Link Previews Coding Plan

Date: 2026-08-25
Status: Approved
Reviewer: User
Risk: High

## 1. Purpose

Implement safe website-card previews for image-free standard posts without changing the PDS/AppView trust boundary. Flutter derives candidates from existing link facet tokens and asks the authenticated AppView for metadata. AppView performs every third-party page and image request through a resolve-and-pin transport. Publishing reuses `app.bsky.embed.external`; public reads hydrate cards from the complete record already stored in `craftsky_posts.record`.

The implementation includes immediate standard posts, standard scheduled posts, canonical post reads, compact quote reads, and federated records. Project preview authoring and scheduling remain excluded because project posts remain photo-required.

## 2. Fixed Contracts And Decisions

- Authenticated route: `POST /v1/link-previews` with the standard session and device middleware.
- Request: one strict camelCase `{ "url": string }` no longer than 8,192 UTF-8 bytes. Flutter sends a normalized fragmentless transport URL. AppView ignores an input fragment for compatibility.
- Response: `{ "url", "title", "description", "thumbnail?" }`, where thumbnail is `{ "bytes", "mimeType", "width", "height" }`.
- Thumbnail bytes: standard padded RFC 4648 base64 representing at most 1,000,000 decoded bytes.
- Metadata limits: title 200 graphemes and 2,000 UTF-8 bytes; description 300 graphemes and 3,000 UTF-8 bytes. Satisfy both limits without splitting a grapheme.
- Image limits: fully decoded JPEG/PNG/WebP only, at most 1,000,000 bytes, each side at most 8,192 pixels, and total area at most 16 MP.
- Fetch limits: at most 2,000,000 raw page bytes, stop after a completed `</head>`, 6-second page phase, 10-second total request, at most three distinct image candidates, and at most five redirects per page or image resource.
- Candidate limit: first four distinct completed link tokens in text order, fetched sequentially and immediately without debounce.
- Embed conflicts: reject `quote + external`, `images + external`, and `project + external`. Preserve existing `images + quote` behavior.
- Scheduled storage: persist frozen external metadata plus normalized source identity in the versioned private payload. Reuse one existing `scheduled_post_media` row at ordinal zero for an optional external thumbnail. No migration or role column.
- Public storage: project the standard external object from `craftsky_posts.record JSONB`. No post column, migration, per-row lookup, or read-time metadata fetch.
- Feature flag: the route always remains registered. Development and test configurations enable it by default. Production requires explicit `LINK_PREVIEWS_ENABLED=true`; otherwise the handler returns `503 link_preview_unavailable` before calling the fetcher.
- User-Agent: fixed `CraftskyLinkPreview/1.0 (+https://craftsky.social)` on page and image requests.
- `Accept-Language`: omitted so requests do not vary by member or device locale.
- Transport: no cookie jar, ambient authorization, referer, generic proxy inheritance, or automatic redirect following. Preserve the original host for HTTP Host and TLS SNI while dialing a validated address.
- Dependencies: promote the repository's existing `golang.org/x/net v0.55.0` resolution to a direct dependency for HTML/charset handling, and add `github.com/rivo/uniseg` as a direct dependency for Unicode grapheme iteration.

### 2.1 Error Taxonomy

Every error uses `{error, message, requestId}`. Messages are destination-free and not displayed by Flutter. Flutter treats every non-success, including 429, as a silent failed candidate.

| HTTP | Code | Meaning |
|---|---|---|
| 400 | `invalid_request` | Malformed/unknown request fields, wrong type, missing URL, or URL over 8,192 bytes. |
| 422 | `link_preview_not_allowed` | Scheme, userinfo, port, address, DNS answer set, redirect, or destination violates outbound policy. |
| 422 | `link_preview_unsupported` | Bounded response cannot yield supported page metadata, such as non-HTML, oversized head, unsupported charset, or no usable fallback title. |
| 502 | `link_preview_upstream_failed` | Target resolution, connection, or non-2xx upstream response failed. |
| 504 | `link_preview_timeout` | Page or total request budget expired. Successful metadata with exhausted image work is still a 200 without thumbnail. |
| 503 | `link_preview_unavailable` | Production preview fetching is disabled. |
| 429 | Existing middleware `rate_limited` | Token or device preview bucket is exhausted. Preserve existing `Retry-After` behavior. |
| 500 | Existing `internal_error` | Unexpected local failure only. |

Internal typed errors may carry unsafe causes transiently for control flow, but handlers and observers map them to these fixed classes without logging or exporting raw values.

## 3. Architecture And Data Flow

### 3.1 Composer Candidate Session

1. Add `hasFacetTokenTrailingBoundary(text, tokenEnd)` to `facet_syntax.dart`. It returns true only when a token ends before the draft end and the next character is shared whitespace, sentence punctuation, or an unmatched closing delimiter recognized by link trimming.
2. Candidate derivation calls `detectSupportedFacetTokens`, keeps completed `LinkFacetToken`s, normalizes fragmentless identity/transport URLs, deduplicates by identity, and retains the first four in current text order.
3. Normalization adds HTTPS for bare domains, lowercases scheme/host, removes default ports and fragments, and otherwise preserves path/query semantics. It does not sort queries, rewrite paths, or strip tracking values.
4. The controller starts the first unattempted request synchronously with the state update and starts each later request only after the prior one settles. Success and failure are cached by identity for the composer session.
5. A session generation plus Dio `CancelToken` prevents stale results after removal, account change, disposal, dismissal, or submit from attaching.
6. Successful cards appear in current text order. The earliest success is initially selected. Left/right controls move selection; a position indicator reflects successful cards. Selection follows normalized identity across reorder.
7. Duplicate occurrences share fetched metadata. AppView's final redirect URL remains the destination base. Its fragment wins; otherwise the first current equivalent source occurrence supplies the fragment, including its absence. Reorder/removal recalculates this client fragment without refetching.
8. Removing the selected source identity drops that card and selects the earliest remaining success.
9. Image or quote state temporarily suppresses the session: cancel current work and hide cards, but retain candidate success/failure caches, queue state, and selection. Removing all images restores cached cards and resumes eligible queued work. Project composer never creates a preview session.
10. Dismissal is separate from suppression. It cancels work, hides cards, and blocks new fetches for the session. A short `Link previews hidden` snackbar offers Undo; Undo restores cached/queued work. Expiry leaves dismissal active until a new composer session.
11. Starting submit snapshots only an already-successful selected card. Pending/failed/absent/dismissed state publishes immediately without external, increments the generation, cancels active work, and ignores late results.

Local draft persistence does not gain fields. Reopening a draft creates a new controller/session from restored text and refetches completed candidates.

### 3.2 AppView Fetch Pipeline

1. Middleware admits session, device, dedicated rate budgets, and route policy before handler work.
2. The handler returns disabled production 503 before service work, then strictly decodes the singular request.
3. `linkpreview.Service` parses the fragmentless transport URL and delegates to an injected safe fetcher.
4. Before each page/image hop, the transport validates HTTP/HTTPS, no userinfo, port 80/443, resolves the complete hostname answer set, rejects the set if any address is forbidden, and dials one retained validated address.
5. Redirect handling is manual. Every hop repeats syntax, DNS, and pinning checks; a resource follows no more than five redirects.
6. The transport has no proxy, cookie jar, forwarded credentials, referer, or user-derived headers. It sends only fixed request headers, including the fixed User-Agent and resource-appropriate `Accept`; it omits `Accept-Language`.
7. Page fetching accepts only 2xx HTML/XHTML, supports declared/sniffed UTF-8 and tested legacy charsets, reads at most 2,000,000 raw bytes, and can stop at a completed head.
8. Metadata extraction independently chooses title, description, and image candidates: non-empty Open Graph, then Twitter, then HTML fallback. Title finally falls back to the validated final hostname; description may be empty. Ignore canonical and `og:url` destinations.
9. Collapse whitespace and clamp metadata by both grapheme and byte limits.
10. Track the first valid `<base href>` and resolve relative image candidates against it, otherwise against the final page URL. Try at most three distinct candidates in priority/document order through the same safe transport.
11. Bound the byte stream first, use `image.DecodeConfig` to derive format/dimensions and reject side/area overflow before allocating a full image, then fully decode to reject corrupt/truncated data. Derive MIME and dimensions from bytes. Any image failure or image-phase timeout produces metadata without a thumbnail.
12. Return the validated final redirect URL, including only a redirect-supplied fragment, plus bounded metadata and optional padded-base64 thumbnail.

### 3.3 Immediate Publish

1. Convert the successful selected card into `CreatePostExternal` using the current navigation URI.
2. For a thumbnail, `ComposerMediaUploader.materializeExternal` uploads the already-validated bytes through the existing AppView image blob endpoint, with current transfer budget, ownership checks, and a composer-ID/content-digest success cache.
3. Upload failure uses existing post failure feedback, creates no post, and retains text/card state. Metadata-only external proceeds without upload.
4. Thread external through `CreatePost`, `PostRepository.create`, `ApiPostRepository`, and `PostApiClient.createPost` as `embed.external`.
5. AppView validates metadata/blob limits and all conflicts before PDS work. It does not require an exact link facet match.
6. `lexiconRecordBody` uses generated standard lexicon types. `syntheticPostRow` stores equivalent raw embed JSON so the create response includes external before Tap catches up.

### 3.4 Scheduled Publish

The private payload gains a frozen source identity distinct from the final navigation URI:

```go
type PayloadExternal struct {
    SourceURL   string                    `json:"sourceUrl"`
    URI         string                    `json:"uri"`
    Title       string                    `json:"title"`
    Description string                    `json:"description"`
    Thumbnail   *PayloadExternalThumbnail `json:"thumbnail,omitempty"`
}

type PayloadExternalThumbnail struct {
    ID       string `json:"id"`
    MIMEType string `json:"mimeType"`
    Width    int    `json:"width"`
    Height   int    `json:"height"`
}
```

1. `SourceURL` is the normalized fragmentless composer identity. `URI` is the frozen final navigation URI after redirect/source-fragment rules.
2. Flutter stages selected thumbnail bytes through the existing private scheduled-media endpoint before schedule create/update and writes the returned media ID into the frozen payload.
3. On reopen, hydrate bytes from private media and seed one successful cached result only when a current completed candidate matches `SourceURL`. Do not refresh it. Other/new candidates may enter the normal sequential queue.
4. If the source identity no longer appears, drop the frozen selection rather than attaching it to unrelated text.
5. `Payload.Media` remains top-level images. `Payload.External` is standard-only and cannot coexist with media, project data, or quote state.
6. Existing `scheduled_post_media` rows remain sufficient: image schedules attach ordered image IDs; external schedules attach zero or one thumbnail ID at ordinal zero.
7. Add row ID to publication snapshots and verify ordered attached IDs against payload references before upload.
8. The worker predicts blobs, freezes the exact record, uploads staged bytes, verifies returned blobs, and uses existing write/retry/recovery/cleanup behavior without any preview fetch.

### 3.5 Indexing And Reads

1. Keep indexer persistence unchanged: the complete record body is already stored in `craftsky_posts.record` on create/update/replay.
2. Append `p.record -> 'embed' AS raw_embed` to `postSelectColumns`; scan it into `PostRow.RawEmbed` before query-specific destinations.
3. Update the manual timeline and search scanners for the appended common column. Other stores using `scanPostRowWithExtra` inherit it automatically.
4. `BuildPostResponse` inspects `$type`, decodes only `app.bsky.embed.external`, and returns `external {uri,title,description,thumb? {cid,mime,size,url}}`. Thumbnail URL synthesis reuses post-image CDN policy.
5. Visible compact quote responses expose the identical external shape. Hidden/unavailable quote policy remains unchanged.
6. If top-level images and external coexist in a federated raw record, retain/index both but omit external from every response. Unknown embed variants remain safely omitted.
7. Notifications remain intentionally unchanged because notification subjects are text-only and notification behavior is out of scope.

## 4. AppView Change Map

### 4.1 Lexicon Governance

- Invoke and record the `atproto-lexicon` review.
- Create `adr/009-standard-external-embed.md` following existing ADR structure.
- Add `app.bsky.embed.external` to the open union in `lexicon/social/craftsky/feed/post.json`.
- Keep the existing `app.bsky` package mapping in `appview/cmd/lexgen/build.json`; add Indigo's `app/bsky/embed/external.json` to the `--external-lexicons` inputs in `justfile`.
- Run `just lexgen`; commit generated `appview/internal/lexicon/craftsky/` output rather than hand-editing it.
- Add generated JSON/CBOR fixtures proving standard external, local quote, and absent embed compatibility.

### 4.2 Link Preview Package

Create `appview/internal/linkpreview/`:

- `linkpreview.go`: bounded `Preview`/`Thumbnail`, typed outcome classes, service orchestration, phase budgets, and safe observer interface.
- `transport.go`: URL checks, resolver/dialer interfaces, complete-answer validation, pinned dialing, fixed headers, direct transport, redirect loop, body limits, and cleanup.
- `addresses.go`: explicit IPv4/IPv6 public-routability classification including mapped forms and all acceptance matrix ranges.
- `metadata.go`: bounded head extraction, charset decode, independent fallback chains, first valid base URL, grapheme/byte clamp, and candidate ordering.
- `image.go`: bounded read, `image.DecodeConfig` dimension/area guard, then full JPEG/PNG/WebP decode and validation.
- Focused tests with fake resolver/dialer/streams/clocks only; never call public DNS or websites.

Central constants match Section 2 exactly. Use `golang.org/x/net/html` plus `html/charset` for bounded document decoding and `github.com/rivo/uniseg` for grapheme iteration; do not implement unsafe byte slicing for grapheme limits.

### 4.3 API, Configuration, Routes, And Limits

- Add `appview/internal/api/link_preview.go` and request/handler/privacy tests.
- Add a narrow preview service dependency to `appview/internal/app/deps.go` and `appview/internal/routes/deps.go`.
- Add validated `LinkPreviewsEnabled` config in `appview/internal/app/config.go`: dev/test default true, production default false and explicit true to enable.
- Document `LINK_PREVIEWS_ENABLED` in `appview/environments/dev.env` and `prod.env.example`.
- Always register `POST /v1/link-previews` in `appview/internal/routes/routes.go` and always include it in `baseV1RoutePolicies()`.
- Assign authenticated/device policy and a new `link_preview` rate class in `policy.go`.
- Extend `rate_limit.go` and `DefaultRateLimitConfig()` with independent 60/hour/token and 120/hour/device buckets. Both must pass before handler work; write/upload counters are untouched.
- Construct preview HTTP transport directly, with `Proxy: nil`; never use `http.DefaultTransport` or `ProxyFromEnvironment`.

### 4.4 Safe Observability

- Extend `appview/internal/observability/metric_recorder.go`, Sentry/no-op implementations, and `Observer` with a preview method accepting only closed enums and bounded numeric buckets.
- Allowed dimensions: stage, result/error class, upstream status class, redirect-count bucket, byte bucket, and duration.
- Forbidden everywhere: page/base/image/redirect URL, hostname, query, fragments, metadata, thumbnail bytes, post text, DID/handle, token, device/session identity, request/response bodies, and raw dependency errors.
- Add privacy-canary capture tests across auth/rate/disabled, fetch, write, scheduling, retry, and Flutter event/report boundaries. Do not add client analytics or breadcrumbs.

### 4.5 Post Write/Read

- `appview/internal/api/post_request.go`: strict `ExternalEmbedRequest`, thumbnail blob validation, and quote/image/project conflict validation.
- `appview/internal/api/post.go`: generated lexicon mapping and synthetic raw embed.
- `appview/internal/api/post_store.go`: `RawEmbed`, common JSONB projection, and common scanner.
- `appview/internal/api/timeline_store.go` and `search_store.go`: destination order updates.
- `appview/internal/api/post_response.go`: shared external response, complete thumb metadata, CDN URL, quote hydration, and images-win.
- No migration and no indexer production change. Extend indexer fixtures to prove raw-record retention and lifecycle idempotency.

### 4.6 Scheduled Posts

- `appview/internal/scheduledposts/payload.go`: frozen external/source/thumbnail payload types.
- `appview/internal/api/scheduled_post_request.go`: strict decoding, canonicalization, source-retention validation, conflict checks, and media-ID derivation.
- `appview/internal/scheduledposts/validation.go`: standard-only external eligibility.
- `appview/internal/api/scheduled_publication_policy.go`: final current-policy validation.
- `appview/internal/scheduledposts/store_queries.go` and `publication_store.go`: select/scan attachment IDs.
- `appview/internal/scheduledposts/publication_processor.go`: verify payload/media identity and construct exact frozen external records.
- Do not change schema, leases, retry budgets, Needs-attention behavior, cleanup, or object ownership.

## 5. Flutter Change Map

### 5.1 Models And API

- Add `app/lib/feed/models/link_preview.dart` for response metadata, thumbnail bytes/MIME/dimensions, normalized source identity, and final URL.
- Add `app/lib/feed/models/create_post_external.dart` for create metadata and optional uploaded `CreatePostBlob`.
- Add `fetchLinkPreview(url, cancelToken)` and external create encoding to `post_api_client.dart` or a narrowly separate client if test isolation is materially clearer.
- Thread nullable external through `PostRepository`, `ApiPostRepository`, `CreatePost`, fakes, and tests.
- Extend `Post` and compact quote models with nullable external `{uri,title,description,thumb}` and regenerate dart_mappable/Riverpod outputs using existing tooling.
- Decode padded base64 strictly and reject malformed or decoded-overflow success bodies at the API boundary; the controller converts that to a silent failure.

### 5.2 Candidate Controller

- Add pure candidate helpers under `app/lib/feed/composer/` and the shared trailing-boundary helper in `shared/rich_text/facet_syntax.dart`.
- Add an auto-disposed Riverpod link-preview controller containing current text candidates, session cache, one in-flight identity/token, selected identity, suppression state, dismissal state, and generation.
- Keep `textCandidates` separate from `temporarilyEligible`; image/quote suppression must not erase cache, queue, or selection.
- Cache failure as well as success, so ordinary edits do not retry an identity during the same session.
- Resume unattempted current identities after suppression/Undo. Never resume after dismissal expiry unless Undo occurred.
- On submit, return the selected success only if already complete, then cancel/invalidate remaining work without awaiting it.

### 5.3 Composer, Upload, Schedule, And Drafts

- `post_composer_sheet.dart`: bind text/image/quote changes, show loading/success carousel, left/right controls, position indicator, dismiss action, Undo snackbar, and pass only a completed selected card to submission.
- `composer_media_uploader.dart`: add external thumbnail materialization using existing upload, timeout, ownership, and digest-cache behavior.
- Keep `composer_submission_coordinator.dart` structure; perform optional thumbnail materialization inside its current operation before repository create.
- Do not change project composer production behavior. Add tests proving links never trigger previews and required-photo validation remains.
- `scheduled_post_api_client.dart`, `scheduled_post.dart`, and `scheduled_composer_media.dart`: encode/decode frozen external/source state, stage one private thumbnail without recompression, hydrate it on edit, and seed the controller only while source identity remains.
- Do not add preview fields to local draft models/codecs. Add privacy/regression tests around existing serialization.

### 5.4 Rendering And Localization

- Add `app/lib/feed/widgets/external_card.dart` as a shared responsive widget with full and compact modes, a fixed optional thumbnail frame, bounded title/description/host, and semantic tap/action labels.
- Render full cards from `PostCard`; all feed/profile/comment/detail surfaces already compose that shared card or receive the same model.
- Carry/render compact external data through `PostSummaryData.fromQuoteView`; preserve hidden/unavailable behavior and text-only notification subjects.
- Apply images-win before presentation as defense in depth.
- Use the existing external URL launcher and pass the exact final URI. Add no webview and perform no rendering-time fetch.
- Add English localization strings for carousel position, previous/next, dismiss, `Link previews hidden`, Undo, loading, and card semantics; regenerate localization output.

## 6. Test-First Implementation Sequence

Use strict red-green-refactor. Preserve every test ID and automation target from `02-acceptance-tests.md`; do not renumber or reinterpret them.

### Step 1: Lexicon Contract

Tests: `IT-008`, `REG-003`; supports `AC-007`.

1. Add failing generated-record fixtures and drift/governance checks.
2. Write the ADR with recorded lexicon-skill evidence and update schema/build inputs.
3. Run `just lexgen` and make quote/no-embed/external fixtures pass.

### Step 2: Metadata, Images, And Safe Transport

Completed tests: `UT-004` through `UT-011`, `IT-002` through `IT-004`, `IT-019`. This supplies the outbound-policy portion of `AT-007` and privacy inputs used when `AT-009` completes in Step 7.

1. Implement metadata fallback, Unicode grapheme/byte clamps, charset/head streaming, base URL, image candidate/decode limits.
2. Implement syntax/address matrices, mixed-answer rejection, resolve-and-pin dialing, redirects, fragment rules, direct no-proxy transport, fixed safe headers, and all phase/total budgets.
3. Assert dial history and body closure in every rejection/timeout branch.

### Step 3: Endpoint, Flag, Limits, And Telemetry

Completed tests: preview-request cases in `UT-012`, `UT-016`, `UT-017`, AppView fetch/route portions of `UT-020` and `IT-018`, `IT-001`, `IT-005`, `IT-006`, `AT-007`, and `REG-007`. The create-post half of `UT-012` completes in Step 4; cross-pillar privacy `UT-020`, `IT-018`, and `AT-009` complete in Step 7.

1. Add strict wire/base64 and standard envelope tests.
2. Add route-always-registered and disabled-503-before-fetch tests.
3. Add independent token/device boundary and reset tests.
4. Add telemetry canaries and only then wire config, dependencies, middleware, handler, and observer.

### Step 4: Post Record Write And Read

Completed tests: create-post half of `UT-012`, `UT-013`, AppView portions of `UT-014`, `IT-007` through `IT-011`, `AT-008`, `REG-002`, `REG-004`, and `REG-008`. This supplies server/model prerequisites for `AT-003`, which completes in Step 6, and `AT-004`, which completes in Step 8.

1. Test valid metadata-only/thumb create and all conflicts with zero PDS writes.
2. Test JSONB common projection/scanners, complete thumb response, synthetic create response, compact quote policy, and unknown embeds.
3. Test federated images+external create/update/replay/delete retention and images-win reads.

### Step 5: Flutter Candidate Session

Completed tests: `UT-001` through `UT-003`, Flutter preview-response portion of `UT-014`, `IT-012`, `AT-001`, `AT-002`, and `IT-020`.

1. Add explicit trailing-boundary, normalization, first-four, duplicate-fragment, reorder/removal, and final-URL fragment tests.
2. Add immediate/no-debounce, one-at-a-time, success/failure cache, stale/account/disposal races, temporary suppression/restore, dismissal/Undo/expiry, selection controls, and silent failure/429 tests.
3. Bind the controller to the standard composer and prove Flutter imports only AppView preview abstractions.

### Step 6: Immediate Submission

Completed tests: Flutter create-request portion of `UT-014`, `IT-013`, `AT-003`, and `REG-001`. Add immediate-write privacy canaries to the still-open cross-pillar `UT-020`/`IT-018` fixtures.

1. Test pending submit starts immediately without external and ignores late completion.
2. Test metadata-only submit, thumbnail upload-before-create, upload failure retention/feedback, and exact forwarded external.
3. Add uploader/repository threading without changing ordinary facet behavior.

### Step 7: Scheduling And Draft Privacy

Completed tests: `UT-018` through `UT-020`, `IT-015` through `IT-018`, `AT-005`, `AT-006`, `AT-009`, `REG-005`, and `REG-006`. This step closes the cross-pillar privacy tests after adding schedule/publication and Flutter canaries to the earlier fetch/write coverage.

1. Test source identity retention/removal, frozen metadata and thumbnail round-trip, private atomic staging, and project rejection.
2. Test due publication/retry/recovery at each side-effect boundary with a fetcher call counter fixed at zero.
3. Test draft disk data excludes preview/cache/selection/dismissal and reopened text starts a new session.

### Step 8: Shared Rendering And Manual Gates

Completed tests: `UT-015`, `IT-014`, remaining Flutter `IT-011`, and `AT-004`. Manual release evidence remains `MAN-001` through `MAN-003`.

1. Add widget tests for full/compact, metadata-only/thumb, narrow/wide, large text, semantics, exact launch URI, and images-win.
2. Run real-device/web presentation and accessibility smoke checks.
3. Treat production direct-egress enablement and privacy-safe p95 measurement as release gates, not merge-time public-network tests.

## 7. Verification

Use focused Go/Flutter tests during each red-green cycle, then run:

```bash
just lexgen-check
just fmt
just dev-d
just test
```

Regenerate Flutter outputs from `app/`:

```bash
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n
```

Then run the root Flutter gates:

```bash
just app-analyze
just app-test
```

`just test` requires the Compose Postgres and MinIO services. No automated preview test may depend on external DNS or public websites.

## 8. Completion Checklist

- `AT-001` through `AT-009`, `UT-001` through `UT-020`, `IT-001` through `IT-020`, and `REG-001` through `REG-008` have the evidence assigned in `02-acceptance-tests.md` or an approved exception.
- Every `AC-001` through `AC-023` is traceable to passing automated/manual evidence.
- `just lexgen-check`, Go race tests, Flutter analysis, and Flutter tests pass.
- The route remains registered while disabled and returns 503 before egress.
- Feature enablement defaults match dev/test/prod requirements and direct transport ignores proxy environment variables.
- No database migration or redundant external post column exists.
- Flutter never fetches third-party preview pages/images and never holds PDS tokens.
- Telemetry and client reporting pass all privacy-canary scans.
- Preview failures and 429 remain silent; pending submit never waits.
- Image suppression restores cache/queue; dismissal survives Undo expiry for the session.
- Project composer/schedules remain photo-required and reject external.
- Immediate and scheduled records use generated `app.bsky.embed.external` types.
- Scheduled edit/publication uses frozen metadata and never refetches the selected card.
- Canonical and compact reads expose complete thumb metadata and apply federated images-win.
- Manual responsive/accessibility, direct-egress, and p95 checks are recorded before release.

## 9. Traceability Summary

| Area | Requirements / Criteria | Test IDs |
|---|---|---|
| Product value and full cards | BR-001, FR-013, AC-001, AC-013 | AT-001, AT-004, IT-001, IT-014, MAN-001 |
| Candidate identity, queue, selection, suppression | FR-009..FR-012, RULE-004, RULE-005, AC-010..AC-012 | AT-001..AT-003, UT-001..UT-003, IT-012, IT-013, REG-001 |
| Metadata, images, bounds | FR-001..FR-004, AC-001..AC-003, AC-006, AC-023 | UT-004..UT-008, UT-014, IT-001, IT-004 |
| SSRF, auth, direct egress, feature flag | FR-005, FR-018, NFR-002, RULE-003, RULE-004, RULE-006, AC-004, AC-005, AC-017 | AT-007, UT-009..UT-012, UT-016, IT-002, IT-003, IT-005, IT-019, MAN-002 |
| Lexicon, create API, and embed conflicts | FR-006, FR-007, RULE-001, RULE-002, RULE-005, AC-007, AC-008, AC-014 | AT-002, AT-003, UT-012, IT-007, IT-008, REG-002, REG-003 |
| Canonical/quote reads and images-win | FR-008, FR-014, FR-019, RULE-002, AC-009, AC-016, AC-020, AC-021 | AT-004, AT-008, UT-013..UT-015, IT-009..IT-011, IT-014, REG-004, REG-008 |
| Rate limits | FR-015, AC-015 | AT-007, UT-017, IT-006, IT-012, REG-007 |
| Scheduling and drafts | FR-016, FR-017, AC-018, AC-019 | AT-005, AT-006, UT-018, UT-019, IT-015..IT-017, REG-005, REG-006 |
| Privacy and platform boundary | NFR-003, NFR-004, AC-022 | AT-009, UT-020, IT-018, IT-020, MAN-001 |

## 10. Approval

The user explicitly approved `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this plan on 2026-08-25. Implementation may proceed using the documented red-green-refactor sequence. The High risk classification, full verification requirements, implementation review, and production direct-egress release gate remain in force.
