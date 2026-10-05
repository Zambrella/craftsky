# Document Review: Useful Logging and Error Reporting

## Verdict

Status: Approved with notes  
Reviewer: Codex, document consistency and traceability review  
Date: 2026-10-05  
Risk level: High

## Summary

The requirements and acceptance-test specification are ready for coding-plan work. The confirmed public/private boundary, actual incoming request paths, cause and stack retention, independent log export, capture ownership, release-platform fallback and repository guidance are consistently represented.

All 24 Must requirements and the one Should requirement have linked test designs. All 24 acceptance criteria have coverage across 49 proposed acceptance, unit, integration, regression and manual cases. Automated document checks found no broken requirement/criterion reciprocity, missing Must coverage or undefined references among those cases. This establishes document traceability, not implemented or passing behavior.

No blocking product decisions or contradictions were identified. The notes below concern decisions that the coding plan must make within the approved requirements. Earlier documents were reviewed without modification. No source or test code was edited, application tests run, production inspected, or stage commit created.

## Findings

All findings are non-blocking for coding planning. Their required actions belong in the plan and subsequent implementation validation; none relaxes a Must requirement.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk / Implementation design | The documents distinguish known-safe, published, private and unknown provenance, but deliberately leave its representation and assignment to implementation. A caller-controlled claim that an arbitrary exception is safe would undermine the private-text protection even if every regex test passes. | `01-requirements.md` FR-001, FR-010, FR-011, RULE-003; `02-acceptance-tests.md` UT-001, UT-003, UT-005, IT-011, GAP-005 | In the coding plan define how supported entry points establish provenance from the operation/source, which structured fields and safe error sources are approved, and the default for unclassified sources. Preserve separate positive known-safe and negative opaque-private-text fixtures. Do not treat prior publication of a target as proof that its associated exception or workflow payload is public. |
| DR-002 | Suggestion | Implementation design / Tests | Arrival must already contain a safely classified actual path while a handler may never complete. The test design covers this outcome but does not prescribe how route classification becomes available before arrival emission. Completion-only route metadata cannot satisfy arrival privacy and usefulness. | `01-requirements.md` FR-004, RULE-004, AC-004, AC-020; `02-acceptance-tests.md` AT-004, UT-007, IT-001, IT-008, GAP-006 | Plan route-aware classification before arrival emission, with safe unmatched defaults and the same policy for companion context. Specify how classifications stay aligned with registered routes and method-specific private actions. Run the held-handler assertion against the actual middleware wiring, rather than only a standalone sanitizer or handler-supplied route pattern. |
| DR-003 | Suggestion | Tests / Sink integration | TD-005 specifies independently varied error/log/tracing gates, while the concrete export cases mostly name the Logs and tracing gates. Their intended independence should become explicit executable configurations to avoid export being accidentally coupled to issue capture. | `01-requirements.md` FR-005, FR-015; `02-acceptance-tests.md` IT-005, IT-013, TD-005, GAP-001 | Instantiate the matrix with Logs enabled and issue capture disabled, tracing disabled and zero-rate tracing, plus Logs disabled with local logging retained. Count log records separately from issue events and batched envelopes. Include the configured direct-slog path and Flutter root logger, not only Observer/reporter helper calls. |
| DR-004 | Suggestion | Implementation design / Risk | Bounds and repeated-error suppression are intentionally implementation decisions. The tests specify boundary behavior, but concrete budgets, suppression scope and terminal-error bypass must be selected before those cases can be made executable. | `01-requirements.md` NFR-002, ASM-003; `02-acceptance-tests.md` UT-013, UT-016, IT-016, GAP-002 | Put sink-specific field/event/depth/cause limits and deterministic suppression policy in the coding plan or an explicit implementation task. Preserve core diagnostics before excerpts, sanitize before truncation, and retain terminal causes and suppression counts. Avoid introducing any diagnostic-only fetch or changing business retries. |
| DR-005 | Suggestion | Validation / Handoff | Manual checks are justified, but their evidence locations and completion ownership are not yet specified. Adapter tests cannot establish physical release-device output, and local transports cannot establish production retrieval. | `01-requirements.md` FR-014, FR-016, FR-017; `02-acceptance-tests.md` IT-014, IT-015, MAN-001–MAN-004, GAP-003, GAP-004, GAP-008 | The coding plan should name the validation artifact location, who runs/reviews device and investigation checks, and how pending checks are reported. Keep production verification separately pending until authorized execution. Verify guide examples against implemented entry points and record human usability review without creating a second policy source. |

## Traceability Review

- Planning to requirements: No separate `00-initial-prompt.md` exists in this workflow folder. The original request, audit findings, confirmed clarification and both grilling rounds are embedded in `01-requirements.md` sections 1–5. The reviewed requirements preserve those decisions and the non-goals in section 8.
- Requirements to acceptance criteria: All 25 requirements reference at least one criterion. Requirement-table and acceptance-criterion references are reciprocal. Must requirements span business outcomes, functional behavior, non-functional constraints and privacy rules; FR-012 remains Should.
- Acceptance criteria to tests: Every AC-001–AC-024 has at least one linked case. All 49 AT/UT/IT/REG/MAN cases have valid requirement and criterion references. The coverage matrix contains every requirement. Designed coverage was inspected for expected outcomes as well as reference presence.
- Audit findings to implementation scope: F-01–F-20 feed the shared contract and feature coverage under FR-009/FR-010/FR-015. IT-006, IT-010, IT-011 and GAP-005 explicitly require a call-site inventory; representative cases do not imply every caller is already covered.

## Coverage Review

- Must requirements covered: 24 of 24 by proposed tests or justified manual checks. The Should requirement FR-012 also has UT-014 coverage.
- Acceptance criteria covered: 24 of 24. The cases comprise 7 acceptance scenarios, 16 unit cases, 16 integration cases, 6 regressions and 4 manual checks.
- Missing or weak coverage: No missing Must or acceptance-criterion design coverage. The final Flutter SDK emission harness is a planned addition, not an existing demonstrated capability. The export configurations in DR-003 should be concrete in executable cases. Numerical bounds and suppression policy remain implementation tasks, as approved.
- Positive usefulness: Known-safe type/message/cause retention, supplied/recovery frames, distinct public actor/target references, unchanged request IDs and meaningful log messages have positive assertions. Tests do not equate absence of secrets with useful diagnostics.
- Privacy: Published versus draft provenance, private-target context reintroduction, unknown free text, credentials/capability URLs, SDK fields, local/debug/platform output and sanitization before truncation are all represented. Neither a public identifier nor a public target makes its private activity public.
- Test levels: Unit cases address classifiers, sanitizers, mapping and bounds; integration cases exercise middleware, real callers and final sink payloads. Acceptance scenarios express operator-visible outcomes and are backed by component harnesses without adding a Gherkin runner. Regression cases protect API/UI and worker behavior.
- Practical automation: Targets reuse Go `testing`/`httptest`/MockTransport, Flutter `flutter_test`/Riverpod/Dio and existing package-specific fixtures. Proposed files are identified as future work. Real PostgreSQL/MinIO suites remain separate from the explicitly incomplete Go unit path. Focused and full-suite commands are documented; no command result is claimed.
- Manual-only coverage: FR-017 uses MAN-003 local/test investigation evidence and MAN-004 production-checklist review. FR-014 combines automated guide/example checks with MAN-002 usability review. FR-016 combines platform-adapter tests with MAN-001 actual release-device output. These manual checks address operational or platform observations that unit tests cannot establish.

## Risk And Approval Review

- Risk level: High remains appropriate. Expanded diagnostics can disclose embedded credentials or private activity through paths, causes, context and final SDK payloads.
- Review requirement: Requirements record the user-approved grilling decisions on 2026-10-05. The user requested acceptance-test design and then selected document review. This review establishes readiness for coding planning.
- Approval notes: This verdict approves the document set for coding-plan work; it does not authorize implementation, deployment, production configuration changes or live production verification. Explicit approval is still required before implementation under the High-risk workflow policy. Selecting the next coding-plan stage is authorization to write that plan only.
- Protection obligations: Final serialized sink checks, vetted unknown/private exception handling, private-path/context tests and retained useful diagnostics remain completion conditions. Logging failure must preserve business behavior and local output without recursion.
- Operational assumptions: Existing access/retention arrangements and effective production gates remain unverified. Opaque workflow IDs must be non-capability identifiers. These assumptions are recorded and have follow-up checks; they do not block local implementation planning.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Plan the diagnostic/provenance contract and its approved source/field classifications; start TDD with UT-003 paired with UT-001, as recommended by the test specification.
- Test order: provenance/cause/privacy → path/context/bounds → final sink harnesses → lifecycle/correlation/export → ownership/expected conditions → route/envelope mapping → feature and private worker coverage → behavior regressions → guide and manual validation evidence.
- Blocking issues: None for coding planning. Implementing code still requires the explicit approval described above. Missing harnesses and selected numerical limits are scoped implementation work; release-device and production evidence stay visible as separate verification statuses.
- Scope completeness: The plan must cover all Must requirements, including contributor guidance and validation evidence. Prioritizing F-01–F-13 cannot silently defer F-14–F-20 or leave private worker failures generic.

## Notes For Next Stage

- Read `01-requirements.md`, `02-acceptance-tests.md` and this review together; carry DR-001–DR-005 into the plan's decisions/tasks and verification order.
- Define supported Go/Flutter entry points and capture owners before changing call sites; classify safe structured fields and enforce equivalent protection at local and remote emission boundaries.
- Use one canonical route/privacy policy where practical, and explicitly plan how AppView route definitions and Flutter categories remain aligned. Actual sanitized paths and route patterns serve different diagnostic purposes.
- Document the sink matrix, independent gates, available correlation and disabled/failed-export behavior. Keep ordinary health noise below INFO and preserve failed probe severities.
- Keep exception messages and safe API diagnostic fields separate from localized user-facing text and public response disclosure. Do not expand the API envelope to carry internal stacks or provider bodies.
- Replace obsolete blanket-public-redaction tests with positive context assertions while retaining credential/private regression tests. Do not use deleted tests or source scans as substitutes for final emitted-payload validation.
- Implement the concise mandatory AGENTS.md policy and canonical guide using final interfaces; compile equivalent examples and perform the guide usability review.
- Record local/test investigation evidence and pending release/production checks separately. No production mutation or remote synthetic-error publication follows from this review.
- No automatic stage commit is enabled. The acceptance-test artifact remains uncommitted from the prior stage; this review adds only `03-document-review.md`.
