# AV-016 — Scheduled image validation permits decompression-bomb OOM

- **Severity:** High
- **Priority/order:** 1 — land with the inbound body/admission work before scheduled media is exposed outside controlled development
- **Status:** Planned
- **Source:** [AV-016](../2026-08-12-appview-code-audit.md#av-016--scheduled-image-validation-permits-decompression-bomb-oom)

## Shared implementation strategy

Replace the direct `image.Decode` call with a two-phase, resource-budgeted image inspector owned by the scheduled-media boundary:

1. decode only the format and header metadata with `image.DecodeConfig`;
2. reject invalid, unsupported, MIME-mismatched, zero/negative, over-dimension, over-pixel, or extreme-aspect metadata using overflow-safe arithmetic; and
3. only for accepted metadata, perform the full decode needed to reject truncated/corrupt payloads, while holding a process-wide concurrency permit.

The compressed-byte ceiling is not an allocation ceiling after decompression. Treat byte count, dimensions, total pixels, aspect ratio, decode concurrency, and decode duration as separate controls. Put hard non-disableable maximums in code and allow validated configuration only to lower them. Return content-free validation/overload errors and never log the image, member, object key, dimensions, or decoder error text.

Use an injected `ImageValidator` rather than package-global functions so limits, concurrency, clocks/contexts, and a test decoder seam are explicit. Keep JPEG, PNG, and WebP as the only registered formats. This is scheduled-private media and stays in AppView/object storage until publication; the change neither writes it early to a PDS nor alters the ordinary public-record boundary.

Because the app is pre-production, make the new limits a breaking policy. Do not grandfather staged media that cannot be shown to have passed the current validator. The Flutter media preparation path should resize unsupported large camera images before upload instead of asking AppView to allocate their native pixel buffers.

## Finding closure

### AV-016 — Scheduled image validation permits decompression-bomb OOM

The update closes AV-016 at four layers:

- **Header inspection before allocation:** `DecodeConfig` reads the encoded dimensions and format without constructing the complete image. Full decode is never called for a header that violates policy.
- **Per-image memory bound:** maximum width, maximum height, and maximum total pixels cap the decoded surface independently. The pixel test uses division after positive checks, not `width * height`, so attacker-controlled integers cannot overflow into acceptance.
- **Process memory bound:** one shared semaphore caps simultaneous validations across all handlers. Its capacity is derived from the AppView container memory budget, not from request rate limits; saturation returns promptly without queuing unbounded payloads/goroutines.
- **Corruption check after bounds:** an accepted header is fully decoded to preserve the existing requirement that truncated or corrupt JPEG/PNG/WebP bodies never become durable private media. The decoded format and bounds must agree with the header inspection and declared canonical MIME type.

Initial code ceilings to validate are 8,192 pixels on either axis, 16,000,000 total pixels, and a 20:1 maximum long-to-short aspect ratio, with one concurrent full decode per AppView process. These values cover normal 12-megapixel images while bounding a four-byte-per-pixel surface to roughly 64 MiB before codec/runtime overhead. They are starting security ceilings, not claims about exact heap use: implementation cannot ship until measured worst-case decoder RSS plus AppView baseline and a documented safety margin fit the deployment memory limit. If that cannot be demonstrated for every accepted codec, move the validator to a separately memory/CPU-constrained worker and keep the same API contract.

## Desired outcome and invariants

- No full image decode begins until format, declared MIME, width, height, pixel count, and aspect ratio pass explicit limits.
- Arithmetic on attacker-controlled dimensions cannot overflow or divide by zero.
- At most the configured number of full decodes can allocate concurrently, and queued validation work is itself bounded.
- Empty, malformed, truncated, unsupported, MIME-mismatched, and policy-oversized media never reaches `scheduledMediaService.Put` or object storage.
- A valid image is decoded once for content validation and stored byte-for-byte; AppView does not transcode, silently crop, or alter member-authored media.
- Errors use the standard `/v1/*` envelope and do not expose unpublished content or high-cardinality identifiers.
- Public/PDS and private/AppView ownership rules are unchanged.

## Scope

### In scope

- A scheduled-media `ImageValidator` and explicit `ImageDecodeLimits` in `appview/internal/api/` or a narrow `internal/media/` package.
- `PutScheduledMediaHandler` dependency injection and error translation.
- Validated configuration, concurrency admission, metrics, and shutdown/cancellation behavior.
- Flutter-side resize/validation for images exceeding the new breaking policy.
- Focused unit, fuzz, concurrency, handler, and memory-budget tests for JPEG, PNG, and WebP.
- Removal of pre-existing non-production staged media that cannot prove current validation.

### Out of scope

- Server-side transcoding, optimization, EXIF normalization, thumbnails, or content moderation.
- Expanding the MIME allow-list or adding animated formats.
- Decoding ordinary immediate PDS uploads, which remain validated/pass-through unless separately redesigned.
- Changing the image Lexicon or public post-record shape.
- Fixing the scheduled object upload/deletion fence, covered by AV-006.

## Design decisions

1. **Inspect, cap, then decode.** `DecodeConfig` is a guard, not a complete validity check; the second phase remains necessary for truncated-content rejection.
2. **Use encoded dimensions for the security decision.** EXIF orientation affects display, not the decoder’s underlying allocation. No EXIF parser is required here.
3. **Enforce all three geometry limits.** Axis limits stop pathological scanlines, pixel limits bound surface area, and aspect limits bound degenerate panoramas.
4. **Use overflow-safe comparisons.** After confirming positive dimensions and axis ceilings, compare `uint64(width) > maxPixels/uint64(height)`; perform aspect multiplication only after the small axis ceiling makes it safe.
5. **Centralize concurrency.** Construct one validator in `internal/app/deps.go` and share it across handlers. Per-request validators would defeat the process memory bound.
6. **Bound waiting as well as work.** Semaphore acquisition observes request cancellation and a short admission timeout. Do not retain a 15 MiB payload in an unbounded queue.
7. **Fail closed on configuration.** Zero, negative, above-code-ceiling, inconsistent, or memory-budget-incompatible values fail startup; zero never disables a limit.
8. **Keep validation content-free.** Stable result codes and format categories are sufficient for operations; raw decoder messages and image metadata are not telemetry.
9. **Require measured headroom.** In-process decoding is accepted only with a reproducible worst-case memory result. Otherwise use a constrained worker rather than weakening limits or relying on rate limiting.

## Unified implementation plan

1. Add failure-first tests that construct small JPEG/PNG/WebP headers declaring excessive dimensions and prove the current handler attempts a full decode or lacks the required guard. Include malformed and truncated fixtures that must continue to fail.
2. Define `ImageDecodeLimits` with width, height, total-pixel, aspect-ratio, concurrent-decode, and admission-wait fields. Add hard code ceilings and `Validate` logic; use explicit names/units and keep upload bytes in the existing `MediaLimits`.
3. Add an `ImageValidator` interface and implementation with a shared bounded semaphore. It accepts context, canonical declared MIME, and the already byte-bounded payload, and returns safe metadata (`format`, `width`, `height`) only after both phases succeed.
4. Implement phase one with `image.DecodeConfig(bytes.NewReader(payload))`. Map only `jpeg`, `png`, and `webp` to their canonical MIME types; reject every other or unregistered format before full decode.
5. Validate dimensions in this order: positive values, per-axis maxima, overflow-safe total-pixel ceiling, then aspect ceiling. Keep the pure geometry predicate separately table-testable.
6. Acquire the decode permit before calling any codec parser, or no later than immediately before `DecodeConfig`, so a header/CPU flood cannot run unbounded concurrently. Release it with `defer` on every path. Use context cancellation/admission timeout for waiting.
7. Run `image.Decode` only after phase-one acceptance. Require the returned format to match both phase one and the declared MIME, and require decoded `Bounds().Dx()/Dy()` to match inspected dimensions and remain within policy. Discard the decoded image immediately after validation.
8. Replace `validateScheduledImageContent` in `scheduled_media.go` with the injected validator. Ensure `scheduledMediaService.Put` is never called on validation or saturation failure and no partial database/object reservation exists.
9. Add stable error mapping: invalid/corrupt/MIME/geometry failures remain `422 scheduled_media_invalid` through the standard envelope; cancelled client work stops quietly according to the HTTP layer; decode-capacity saturation returns retryable `503` or `429` with bounded `Retry-After`. Do not echo dimensions or decoder text.
10. Construct one validator in `internal/app/deps.go` and thread it through route construction. Add lower-only environment configuration in `internal/app/config.go`; coordinate its positive-duration validation with AV-030 and its inbound queue/body behavior with AV-014/AV-015.
11. Add content-free metrics for validation result category, codec, duration, current in-flight count, and admission saturation. Do not record DID, media/schedule ID, object key, byte hash, dimensions, raw MIME, or error text.
12. Update Flutter’s scheduled-media preparation service to read local dimensions and resize/re-encode images that exceed the server policy before upload. Preserve aspect ratio and alt text; expose a clear validation message if preparation cannot produce an accepted image.
13. Add a reproducible memory test/benchmark that decodes valid worst-case fixtures for each accepted codec at the configured concurrency. Record AppView baseline, peak RSS/heap, container limit, and safety margin in the verification evidence. If any codec exceeds budget, implement the same validator in a separately constrained worker before closing the finding.
14. Purge pre-existing non-production scheduled posts/media through the existing private-media cleanup lifecycle, verify object cleanup, and restart those schedules rather than grandfathering old validation.
15. Run focused API tests, scheduled-post PostgreSQL tests, race tests, fuzzing for a time-bounded corpus, the full static/vulnerability gates, and a container-level concurrent upload test.

Likely files include `appview/internal/api/scheduled_media.go`, a new `appview/internal/api/image_validator.go` (or `internal/media/validator.go`), `appview/internal/api/scheduled_media_test.go`, `appview/internal/app/config.go`, `appview/internal/app/deps.go`, route construction tests, scheduled-media Flutter preparation/tests, and deployment environment documentation.

## Data, schema, migration, and reconciliation plan

**Schema migration: None.** AV-016 is closed at the pre-reservation request boundary, and a staged object is not decoded again during ordinary publication. Persisting width/height is not necessary to bound the vulnerable allocation and would create a second metadata source without changing the immutable bytes.

The new limits are nevertheless a breaking media policy. Before rollout, use the existing scheduled deletion/cleanup services to remove every non-production schedule and staged-media row created under the old validator, drain `scheduled_post_cleanup_jobs`, and verify the private object bucket contains no corresponding orphan. Do not truncate rows independently of object cleanup and do not touch already-public PDS blobs or records.

If the product later requires dimension-policy changes to revalidate already staged media without decoding it again, introduce a separately reviewed `validation_policy_version` plus stored inspected dimensions. That future schema must never mark legacy rows current by default. It is not required for this finding because the chosen pre-production cutover removes legacy private media.

## API, client, configuration, and operations impact

- `PUT /v1/scheduled-post-media/{mediaId}` keeps its success response shape and existing private owner authorization.
- Newly rejected media returns the existing standard validation envelope. Saturation is explicitly retryable; Flutter keeps the local source and offers retry rather than losing composer state.
- The scheduled Flutter path must resize large native camera images locally. Immediate public PDS uploads retain their current contract until a separate product decision aligns them.
- Add lower-only settings for operational decode limits only if deployment tuning is required. Code ceilings remain enforced even when environment values are absent or malformed.
- AppView/container memory limits must be explicit in deployment configuration. Alert on sustained validation saturation or abnormal duration using content-free, low-cardinality metrics.
- Rollout requires a private scheduled-state cleanup and real JPEG/PNG/WebP smoke upload; it does not require PDS reindexing or Lexicon changes.

## Security, failure, and race considerations

- Acquire one shared permit before decoder work and release it on error/panic/cancellation. Wrap only the narrow decoder boundary with panic recovery so a codec panic becomes a safe failure while still releasing the permit; continue to rely on the outer HTTP recovery as defense in depth.
- Go’s image decoders do not provide general mid-decode context cancellation. Byte/pixel/concurrency caps provide the in-process bound; a hard CPU/time isolation requirement requires the constrained-worker fallback.
- Check both configuration and decoded image bounds. Do not trust `DecodeConfig` metadata merely because a later decoder returns success.
- Avoid multiplying attacker-controlled dimensions before caps make multiplication safe. Test architecture-independent behavior on 32- and 64-bit integer boundaries in the pure predicate.
- Keep the payload byte-bounded even after AV-014 removes duplicate buffering. An unknown-length/chunked request must not bypass the 15 MiB compressed ceiling.
- A saturated validator must not create a goroutine per waiter or retain unlimited request bodies. The outer request/concurrency limits and short acquisition budget bound waiters.
- Decoder failures are user-input validation failures, not logs at error level and not retryable storage failures. Infrastructure/saturation/cancellation remain distinguishable.
- Do not log or trace unpublished image bytes, hashes, exact dimensions, object keys, owner identifiers, or raw codec errors.

## Unified test plan

### Unit tests

- Geometry tables for zero/negative, exactly-at-axis, one-over-axis, exactly-at-pixels, one-over-pixels, wide/tall aspect boundaries, and integer-overflow-sized inputs.
- Format/MIME tables for JPEG, PNG, WebP, unsupported formats, empty data, misleading signatures, and canonical MIME mismatch.
- A decoder spy proves full decode is never called after `DecodeConfig` reports over-limit geometry or unsupported/mismatched format.
- Full decode remains required for truncated headers/bodies; config-success/decode-failure returns the safe validation error.
- Header/decode dimension or format disagreement fails closed.
- Config validation rejects disabled, inconsistent, or above-hard-ceiling limits.

### Handler and integration tests

- For every invalid geometry/format/corruption case, assert `422`, standard envelope/content type/request ID, zero service calls, and zero durable/object-store writes.
- Exactly-at-limit valid JPEG/PNG/WebP fixtures succeed and are stored byte-identically.
- One-over-limit fixtures have tiny compressed bodies but are rejected before full allocation.
- Fill the decode semaphore, submit additional authenticated requests, and assert bounded retryable responses, prompt cancellation, no growing waiter/goroutine count, and recovery when the permit is released.
- Combine chunked compressed-size overflow with extreme dimensions to prove both byte and pixel controls remain active after the AV-014 body refactor.

### Fuzz, fault, concurrency, and operations tests

- Fuzz the pure geometry predicate and `DecodeConfig`/format classifier with a bounded corpus for JPEG, PNG, and WebP; assert no panic, overflow acceptance, or service call.
- Run concurrent validator/handler tests under `go test -race`, including cancellation and injected decoder panic barriers.
- Measure valid worst-case fixtures for all codecs at configured concurrency inside the release container. Peak memory plus baseline and safety margin must remain below its hard memory limit.
- Exercise the private-media cleanup cutover and verify database rows, cleanup jobs, and bucket objects converge without touching public PDS data.
- Inspect logs, traces, metrics, and error bodies with canary content/identifiers and assert none leak.

## Per-ID traceability and acceptance criteria

### AV-016

- [ ] `image.DecodeConfig` runs before every full scheduled-image decode.
- [ ] Width, height, total pixels, and long-to-short aspect ratio have explicit non-disableable ceilings enforced with overflow-safe arithmetic for JPEG, PNG, and WebP.
- [ ] A test seam proves no full decode occurs for a compact header declaring over-limit geometry.
- [ ] Full decoding of accepted geometry still rejects truncated/corrupt streams and verifies MIME/format/bounds consistency.
- [ ] One process-wide concurrency gate and bounded admission wait cap aggregate decoder memory; cancellation and panic always release the permit.
- [ ] Worst-case per-codec container measurements demonstrate baseline + peak decoder use + safety margin below the deployment limit, or decoding is moved to a constrained worker.
- [ ] Invalid or saturated requests create no scheduled-media reservation/object and use safe standard error envelopes.
- [ ] Legacy non-production staged private media is removed through durable cleanup, without deleting any PDS record/blob.

## Dependencies and coordination

- **AV-006:** keep media validation before reservation, while the upload/delete update supplies the separate object-effect fence.
- **AV-013:** upgrade the Go toolchain and `golang.org/x/image`, rerun `govulncheck`, and retain parser fuzz/regression tests after upgrades.
- **AV-014 / AV-015:** share compressed-body, in-flight, timeout, and authenticated abuse controls; route rate limiting does not replace the decode semaphore.
- **AV-030:** all admission-wait and related duration configuration must be positive and relationally valid.
- **AV-033 / AV-036:** require focused PostgreSQL/race/fuzz/static/vulnerability gates and retain the container memory check as explicit release evidence.

## References

- [Scheduled-post requirements](../changes/2026-07-31-scheduled-posts/01-requirements.md)
- [Scheduled-post acceptance tests](../changes/2026-07-31-scheduled-posts/02-acceptance-tests.md)
- [Scheduled-post implementation review](../changes/2026-07-31-scheduled-posts/05-implementation-plan.md)
- [Go `image.DecodeConfig` documentation](https://pkg.go.dev/image#DecodeConfig)
- [Go `image.Decode` documentation](https://pkg.go.dev/image#Decode)
- [`golang.org/x/image/webp` documentation](https://pkg.go.dev/golang.org/x/image/webp)
- [Go fuzzing guidance](https://go.dev/doc/security/fuzz/)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
