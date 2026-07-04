# Document Review: Flutter Error Handling And Sentry Reporting

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-07-03
Risk level: Medium

## Summary
The requirements and acceptance test specification are aligned and ready for coding-plan work. The recommended direction is preserved: central typed error UX, app-owned Sentry reporting, privacy-bounded diagnostics, localized safe user messages, and global Riverpod retry disablement. Every Must business, functional, non-functional, and business-rule requirement has linked acceptance criteria and test coverage.

The remaining findings are non-blocking coding-plan notes. They should be handled while designing implementation boundaries, static checks, and release-symbolication tasks.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk | Release symbolication is intentionally broad and includes platform/build choices that are not yet pinned to exact implementation steps. | `01-requirements.md` FR-017, AC-015; `02-acceptance-tests.md` IT-007, REG-011, MAN-001 | In the coding plan, choose the minimal supported Sentry Dart Plugin setup, document platform-specific build commands, and make an explicit source-context decision. |
| DR-002 | Suggestion | Tests | Static regression checks are specified, but the exact enforcement mechanism is still open. | `02-acceptance-tests.md` REG-006, REG-007, REG-008, REG-011 | In the coding plan, decide whether these are Dart tests, scripts invoked by Flutter tests, or CI-only checks, and keep them runnable without real Sentry credentials. |
| DR-003 | Suggestion | Requirements | The reviewed requirements document still labels itself `Status: Draft`, even though it has no blocking open questions. | `01-requirements.md` section 22 | If the workflow tracks document states strictly, update the status during the next requirements revision or as a manual note. This does not block coding planning. |

## Traceability Review
- Planning to requirements: No separate `00-initial-prompt.md` exists in this folder, but `01-requirements.md` captures the initial request, codebase findings, decisions, candidate approaches, recommended direction, goals, non-goals, risks, assumptions, and open questions.
- Requirements to acceptance criteria: All Must `BR`, `FR`, `NFR`, and `RULE` entries link to one or more acceptance criteria. Should requirements also have traceable criteria where relevant.
- Acceptance criteria to tests: `02-acceptance-tests.md` maps each requirement and acceptance criterion to concrete acceptance, unit, integration, regression, or manual checks. Manual-only areas are justified for real Sentry ingestion and native crash capture.

## Coverage Review
- Must requirements covered: `BR-001` through `BR-004`, Must `FR` entries, `NFR-001` through `NFR-003`, and `RULE-001` through `RULE-005` are covered by acceptance criteria and tests.
- Missing or weak coverage: None blocking. The weakest areas are release symbolication and static policy enforcement because their exact implementation mechanisms are deferred to the coding plan.
- Manual-only coverage: Real staging Sentry delivery and native crash capture are manual or partly manual by design, with fake/no-op reporters covering the automated contract.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Implementation should receive focused review around startup/error-handler ordering, Sentry redaction allowlists, reportability classification, direct Sentry import boundaries, localization coverage, and Riverpod retry behavior.
- Approval notes: The document set is safe to use for coding planning. The privacy and no-raw-error requirements are explicit enough to prevent accidental broad Sentry capture or user-facing diagnostic leakage.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Start with `UT-001` in `app/test/observability/sentry_options_test.dart`, then add the app-owned reporter/no-op abstraction and disabled-feature guard tests before wiring startup handlers.
- Blocking issues: None.

## Notes For Next Stage
- Keep Sentry imports behind the central reporting implementation; plan a static check for this before feature call sites are added.
- Treat redaction and reportability as core domain behavior with unit tests before integrating provider, Dio/AppView, and log bridges.
- Plan l10n generation and generated-file handling explicitly because user-facing error cases depend on generated `AppLocalizations`.
- Preserve existing debug diagnostics while adding Sentry capture; startup handler ordering should be designed and tested rather than patched opportunistically.
