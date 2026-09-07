# Implementation Review: PDS Migration And Handle Change Resilience

## Verdict

Status: Approved
Reviewer: OpenCode
Date: 2026-09-04
Risk level: High

## Summary

The migration, DID-first identity, repository verification, scheduling, deletion, client credential, and effect-time OAuth metadata-cache boundaries have broad automated coverage, and all repository release gates pass. The correction passes resolve `IR-001`, `IR-002`, and `IR-003`; the phase 24 review additionally found and corrected constructor-level TTL enforcement, abandoned-loader capacity accounting, production OAuth-flow evidence, and complete cache-outcome telemetry coverage. Final re-review found no findings. The implementation is ready for handoff, with the documented device and external staging checks still required before release.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: BR-001 through BR-003; FR-001 through FR-032 as scoped by the approved plan; NFR-001 through NFR-006; RULE-001 through RULE-007.
- Tests implemented: Planned unit, acceptance, integration, and regression coverage through phase 24 is present across Go and Flutter, including signed-CAR trust failures, exact-parent fencing, DID ownership, handle changes, deletion identity, scheduler cutoff, replay, credential boundaries, production-path cross-sink secret scanning, and operations-only OAuth metadata caching with fresh OAuth-flow discovery.
- Unplanned behavior: None identified. The repository lease and alert defaults match CPQ-001.
- Remaining gaps: `MAN-001` and external dashboard/alert delivery in `MAN-002` remain explicitly pending for pre-release validation because no running device/DTD or staging observability backend is available in this workspace.

## Test Evidence

- Commands reviewed: Focused and repeated race-enabled cache/verifier/real-flow/observability tests; PostgreSQL-backed Go auth/app integration tests; `go vet`; `just appview-check`; `just app-test`; `just app-analyze`; `just test`; `git diff --check`.
- Passing evidence: The cache invariant corrections and production OAuth-flow test passed ten times under `-race`; the full affected package sweep passed under `-race`; final `just appview-check` passed every release gate; all 1,978 Flutter tests passed; Flutter analysis reported no issues; the full serialized Go race suite passed against PostgreSQL and MinIO; `git diff --check` passed.
- Failing or skipped tests: The first phase 24 `appview-check` encountered timing-sensitive push lifecycle and auth fixture failures. Each failed test passed ten times in isolated `-race` runs without code changes, and the final complete `appview-check` passed. The first new production-flow test run exposed a duplicate fake PAR request URI; assigning distinct fixture URIs resolved it without a production change. No product test currently fails. Device-level `MAN-001` and external staging dashboard/alert delivery in `MAN-002` were not available locally and remain documented as pending.

## Risk Review

- Risk level: High
- Risk notes: The core migration touches OAuth credential authority, durable repository repair, private deletion state, owner lifecycle, and many Flutter identity surfaces. The operations cache remains process-local, positive-only, capacity-bounded, and limited to a five-minute fixed insertion TTL; DID resolution and OAuth start/registration/callback checks remain fresh. The accepted residual risk is that an issuer-only change at an unchanged PDS may remain unseen for at most five minutes, while PDS movement remains immediately detectable. The production-path cross-sink harness, exact-parent fencing tests, signed-CAR verification, and full release gates substantially reduce data-loss and credential-forwarding risk. `appview-check` reports `GO-2026-5932` for the unmaintained transitive `x/crypto/openpgp` surface with no available fixed version; the configured release gate passes and the approved exact Indigo/Tap pins remain unchanged.
- Approval notes: Approved for handoff. Complete `MAN-001` and external staging portions of `MAN-002` before release.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: The UI changes are identity-contract and error-state changes covered by focused widget tests and the full Flutter suite. No polish-only defect was identified in this review.
- Suggested polish notes: Complete `MAN-001` on a supported device before release; treat any visible sentinel, stale handle, or unclear reauthorization action as a behavioral correction rather than optional polish.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None; all review findings are resolved.
- Verification to rerun: None for handoff. Run the documented device and staging checks before release.
