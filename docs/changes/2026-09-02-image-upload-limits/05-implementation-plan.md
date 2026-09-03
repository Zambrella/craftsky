# TDD Implementation Plan: Image Upload Limits

Status: Approved

## Inputs
- Approved plan: `/Users/douglastodd/.plannotator/plans/image-upload-limits-plan-2026-09-02-approved.md`
- Approved correction plan: `/Users/douglastodd/.plannotator/plans/image-upload-limits-correction-2026-09-02-approved.md`
- Implementation review: `06-implementation-review.md`
- Repository architecture and Lexicon rules: `AGENTS.md`

## Implementation Rules
- Implement only behavior approved by the image upload limits plan.
- Write or update a failing focused test before each behavior change.
- Run the smallest relevant test first and refactor only while green.
- Keep Flutter preparation and AppView enforcement independent and testable.

## Test Order
| Step | Test ID | Plan Requirement | Expected Initial State |
|---|---|---|---|
| 1 | IL-001 | Exact shared byte and geometry policy | Fails |
| 2 | IL-002 | Bounded Flutter source admission and safe geometry | Fails |
| 3 | IL-003 | Bounded target-size encoding | Fails |
| 4 | IL-004 | Scheduled media reuses canonical prepared bytes | Fails |
| 5 | IL-005 | AppView exact byte and dimension boundaries | Fails |
| 6 | IL-006 | Immediate uploads are decoded before PDS forwarding | Fails |
| 7 | IL-007 | Record blob metadata matches upload policy | Fails |
| 8 | IL-008 | Lexicons encode the 2,000,000-byte limit | Fails |

## Implementation Steps
### IL-001
- Write failing tests for exact Flutter and AppView defaults.
- Implement central policy constants.
- Result: Passed. Defaults are 2,000,000 encoded bytes, 4000 x 4000 uploaded pixels, and separate 50,000,000-byte/8192-side/16-megapixel source admission.

### IL-002
- Write failing tests for 50,000,000-byte source admission and source geometry guards.
- Implement bounded reads and header-first geometry validation.
- Result: Passed. Picker-reported and streamed byte lengths are independently bounded; decoder configuration is checked before frame decode.

### IL-003
- Write failing tests for 4000-pixel resize, byte target, transparency, and hard output checks.
- Implement bounded JPEG quality search and measured-size resolution fallback.
- Result: Passed. JPEG uses explicit 4:2:0 encoding and bounded quality/resolution attempts; transparent PNG remains lossless.

### IL-004
- Write a failing scheduled-media identity test.
- Stage canonical prepared bytes without a second encode.
- Result: Passed for both ready and already-uploaded composer states.

### IL-005
- Write failing AppView byte/config/validator boundary tests.
- Set exact limits and retain decode admission controls.
- Result: Passed. Width and height hard ceilings are 4000; pixel, aspect, semaphore, and wait limits remain fail-closed.

### IL-006
- Write a failing handler test proving invalid images never reach PDS effects.
- Share the bounded image validator between immediate and scheduled handlers.
- Result: Passed. Validation failure occurs before PDS effect construction.

### IL-007
- Write failing post/profile/business metadata boundary tests.
- Apply MIME and size validation consistently.
- Result: Passed for post, profile, product, event, and scheduled defaults through shared policy constants and validators.

### IL-008
- Add ADR 011, tighten both canonical Lexicons, and regenerate derived outputs.
- Result: Passed. Go schema copy and importer-generated contract were regenerated; `just lexgen-check` and importer generation checks pass.

## Verification Evidence
- `flutter test`: 1,925 tests passed after the final image-dimension parser migration.
- Dart MCP static analysis: no diagnostics.
- `go test ./...`: all AppView packages passed.
- `npm test`: all 174 importer tests passed.
- `npm run generate:check`: passed.
- `npm run typecheck`: passed.
- `npm run lint`: passed.
- `just lexgen-check`: passed.
- `git diff --check`: passed.
- Device memory/time measurements remain a release task because no physical lowest-supported devices were attached during implementation.

## Completion Checklist
- [x] All approved hard limits have passing boundary tests
- [x] Flutter source and output processing is bounded
- [x] Immediate and scheduled AppView validation is authoritative
- [x] Relevant regression tests pass
- [x] Lexicon and generated outputs are current
- [x] Static analysis passes
- [x] Implementation review completed and approved

## Correction Pass

### Test Order
| Step | Test ID | Review Finding | Expected Initial State |
|---|---|---|---|
| 1 | IL-C01 | IR-001 header-only preview inspection | Fails |
| 2 | IL-C02 | IR-001 bounded preparation concurrency and byte ownership | Fails |
| 3 | IL-C03 | IR-002 shared 20:1 Flutter output policy | Fails |
| 4 | IL-C04 | IR-003 importer 4000 x 4000 final-output policy | Fails |
| 5 | IL-C05 | IR-004 accurate direct-PDS enforcement boundary | Documentation gap |

### IL-C01
- Write a focused inspection test whose image configuration is readable but whose pixel data cannot be decoded.
- Require preview inspection to return dimensions while full preparation fails.
- Result: Passed. JPEG/PNG dimensions are parsed and bounded before decoder startup, JPEG preview orientation is read from EXIF without pixel decoding, and a pre-SOS JPEG header succeeds inspection while failing full preparation as expected.

### IL-C02
- Write a provider test proving no more than one full preparation runs concurrently while four attachments remain supported.
- Serialize the composer pipeline and remove avoidable explicit byte copies without changing retry, cancellation, preview, or ready-state behavior.
- Result: Passed. Four-image selection remains supported, overlapping add pipeline invocations share one provider-wide tail, no state observation contains more than one preparing image, and source/prepared buffers are reused instead of explicitly copied between provider stages.

### IL-C03
- Write exact 20:1 and over-20:1 prepared-output tests in both orientations.
- Put the maximum ratio in shared Flutter configuration and enforce it for every canonical prepared result.
- Result: Passed. Shared configuration is exactly 20:1; both orientations accept the boundary and reject the next ratio, and canonical preparation rejects extreme input before upload.

### IL-C04
- Write importer sanitizer tests proving a 4001-pixel side is resized before the first successful return, including a compressible 5000 x 1000 regression case.
- Separate importer source dimensions from final dimensions and validate encoded output before return.
- Result: Passed for JPEG, PNG, and WebP at exact 4000, 4001 landscape/portrait, and 5000 x 1000 first-pass regression cases. Wider source admission remains separate.

### IL-C05
- Correct ADR and implementation evidence so Lexicon metadata constraints are not described as decoded-blob validation.
- Keep asynchronous direct-PDS blob quarantine outside this correction pass.
- Result: Completed. ADR 011 now scopes decoded-byte authority to AppView-mediated uploads and records direct-PDS validation/quarantine as separate architecture work.
