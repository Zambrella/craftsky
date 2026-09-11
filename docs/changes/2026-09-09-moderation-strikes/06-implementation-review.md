# Implementation Review: Moderation Cases, Strikes, Appeals, And Suspension

## Verdict

Status: Approved
Reviewer: OpenCode
Date: 2026-09-10
Risk level: High

## Summary

The final correction pass resolves IR-008 through IR-010. Owner history now offers appeals only for unappealed decision events, presents immutable effect actions as historical facts, and exercises stored suspension through `AddRoutes`, actual post handlers, and the recording PDS executor boundary. The complete PostgreSQL-required AppView suite, all 2,336 Flutter tests, static analysis, and the release-equivalent AppView gate pass. No remaining automated merge blocker was identified.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: The migration, moderation domain, admin and owner APIs, standing enforcement, notifications, Flutter navigation, appeal behavior, and observability map to the approved requirement set. No lexicon or unauthorized PDS-owned record behavior changed.
- Tests implemented: The final widget matrix covers an eligible decision, expiry, restoration, and an existing appeal. Historical effect coverage renders apply, expire, negate, and restore actions together. The final IT-014 integration uses PostgreSQL-backed standing and lifecycle state, production `AddRoutes` composition, actual retained-delete and denied-create handlers, and a recording `NewPDSEffects` executor.
- Unplanned behavior: Direct effect-change commands emit the same bounded operation telemetry as other approved moderation commands; this satisfies the approved observability requirement without changing an external contract.
- Remaining gaps: MAN-001 through MAN-004 remain externally blocked as documented. They require Retool, representative provider/device delivery, real email-client behavior, and physical accessibility environments.

## Test Evidence

- Commands reviewed: Focused Flutter model/widget tests; focused PostgreSQL-required route and moderation packages; serial PostgreSQL-required `go test -p 1 ./... -count=1`; `go vet ./...`; full `flutter test`; `flutter analyze`; Dart MCP analysis; `git diff --check`; and `just appview-check`.
- Passing evidence: Focused corrections passed. The serial AppView suite passed without database skips. All 2,336 Flutter tests passed. Flutter analysis, Dart MCP analysis, `go vet ./...`, and `git diff --check` passed. `just appview-check` passed every release gate; artifact: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.ImGJUeKP8m`.
- Failing or skipped tests: No automated test failed or skipped. MAN-001 through MAN-004 were not run because their external systems and devices are unavailable. The release gate continues to report the accepted `GO-2026-5932` advisory for the unmaintained transitive `golang.org/x/crypto/openpgp` package; no fixed version is available and the existing gate permits it.

## Risk Review

- Risk level: High.
- Risk notes: Moderation authority, private evidence, suspension, and PDS-effect boundaries remain intrinsically high risk. Automated coverage now includes atomic transitions, deterministic concurrency, privacy-safe representations, exhaustive route classification, production observability hooks, and the combined real-handler PDS boundary that was previously missing.
- Approval notes: Ready for merge or handoff based on automated evidence. Complete MAN-001 through MAN-004 when the required external environments are available.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The owner history UI follows existing responsive and localization patterns, and the misleading action wording and appeal eligibility defects are corrected. A device pass would still provide useful non-blocking evidence for large text, screen readers, and email handoff.
- Suggested polish notes: Inspect chronology, large-text wrapping, focus semantics, and appeal copy on representative mobile devices as part of MAN-002 and MAN-004.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None; no additional correction stage is required.
- Verification to rerun: Rerun the standard release gates if implementation files change after this review.
