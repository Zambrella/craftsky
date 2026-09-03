# Implementation Review: Image Upload Limits

## Verdict
Status: Approved
Reviewer: OpenCode
Date: 2026-09-03
Risk level: Medium

## Summary
The implementation and correction pass satisfy the approved image-upload policy across Flutter, AppView, the Instagram importer, and Lexicon-derived contracts. The previous review findings are resolved: source inspection is header-first, memory-heavy preparation is serialized, the shared 20:1 output limit is enforced, importer output is bounded to 4000 pixels, and ADR 011 accurately describes the direct-PDS validation boundary. No blocking or important findings remain.

## Findings
None identified.

## Requirement And Test Traceability
- Requirements implemented: IL-001 through IL-008 and correction steps IL-C01 through IL-C05 are complete.
- Tests implemented: Flutter source-reader, media-service, provider, state, configuration, and scheduled-post tests; AppView blob request, handler, validator, configuration, and post request tests; importer safety and sanitizer boundary tests; Lexicon generation checks.
- Unplanned behavior: None identified in the changed implementation. The scheduled-worker retry-classification concern found during review predates this branch; this branch changes only its default media byte limit and does not expand that existing issue.
- Remaining gaps: Physical lowest-supported-device peak-memory and elapsed-time measurements remain a release gate. The repository change folder contains `05-implementation-plan.md` but not repository-local `01` through `04` artifacts; the approved Plannotator plans remain the authoritative requirements/design source.

## Test Evidence
- Commands reviewed: `flutter test`; Dart MCP static analysis; `go test ./...`; `npm test`; `npm run generate:check`; `npm run typecheck`; `npm run lint`; `just lexgen-check`; `git diff --check`; simulator hot restart and runtime-error inspection.
- Passing evidence: 1,925 Flutter tests passed; Dart analysis reported no diagnostics; all AppView Go packages passed; all 174 importer tests passed; importer lint, type, and generation checks passed; Lexicon generation checks passed; simulator hot restart completed without runtime errors; diff whitespace check passed.
- Failing or skipped tests: No automated failures were reported. Physical lowest-supported Android/iOS peak-memory and elapsed-time measurements were skipped because suitable devices were unavailable.

## Risk Review
- Risk level: Medium
- Risk notes: Automated coverage is broad, processing concurrency is bounded, and upload policy is consistent across first-party paths. Residual risk is limited to unmeasured peak memory and timing on the lowest-supported physical devices and the documented inability of Lexicon metadata alone to validate decoded direct-PDS blobs.
- Approval notes: Approved for merge. Complete physical-device measurements before release and handle any future direct-PDS blob quarantine through a separately reviewed architecture change.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: The blocking issues are processing, validation, and architecture concerns. No review finding is limited to copy, spacing, styling, accessibility, or visual presentation.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None.
- Verification to rerun: Physical low-end Android/iOS peak-memory and elapsed-time measurements before release.
