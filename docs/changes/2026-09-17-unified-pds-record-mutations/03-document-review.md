# Document Review: Tap-Authoritative PDS Data Lifecycle

## Verdict
Status: Approved
Reviewer: OpenCode document review
Date: 2026-09-22
Risk level: High

## Summary
The requirements and acceptance-test specification are consistent, complete, and ready for coding-plan work. The revision resolves the prior scope contradiction for scheduled publication, defines mutation-specific Flutter reconciliation, promotes the binding Q13-Q19 decisions into FR-043 through FR-049 and AC-051 through AC-057, and provides exact command retention, PDS fallback, migration, retry, table-boundary, production-wiring, delivery-order, and legacy-removal evidence.

Mechanical rereview found 66 Must requirements and one Should requirement. Every Must requirement links to acceptance criteria and detailed tests, every AC-001 through AC-057 appears in detailed tests, all coverage-matrix test references resolve, and detailed test rows include the complete requirement ownership of every claimed acceptance criterion.

## Findings
None identified.

## Traceability Review
- Planning to requirements: The Tap-authoritative source model, command journal, aggregate set semantics, atomic PDS transactions, in-memory Flutter policy, and approved vertical-slice order are preserved. Scheduled final publication uses only the AppView command journal and remains outside the Flutter controller.
- Requirements to acceptance criteria: All 66 Must requirements link bidirectionally to at least one acceptance criterion. NFR-005 remains explicitly Should-cover guidance and shares AC-044 with mandatory NFR-003 redaction.
- Acceptance criteria to tests: AC-001 through AC-057 all appear in detailed acceptance, unit, integration, regression, or justified manual tests. Detailed rows carry the complete requirements assigned to their claimed criteria.

## Coverage Review
- Must requirements covered: All 66 Must requirements have automated test coverage. FR-049 additionally uses MAN-003 to gate delivery phase order while automated conformance proves the final architecture.
- Missing or weak coverage: None identified. Exact retry/status oracles, generated signed-CAR verification, guarded single-write fallback, invalid-record preservation, command compaction, sorted aggregate locking, overlay clock origin, real Indigo/coordinated-client wiring, real Flutter client/provider adoption, and final legacy removal are explicit.
- Manual-only coverage: None for required behavioral correctness. MAN-003 verifies implementation phase ordering that cannot be reconstructed reliably from final runtime behavior; final FR-049 conformance remains automated. MAN-001 reviews the operational usefulness of Should-level telemetry while redaction and bounded labels remain automated.

## Risk And Approval Review
- Risk level: High because the change spans ingestion, database projections, PDS transactions, mutation APIs, scheduled publication, and Flutter state.
- Review requirement: Combined document review completed. Coding planning may proceed; implementation still requires the specified test-first sequence and release gates.
- Approval notes: Product approval for test design was granted on 2026-09-22. No blocking product or coding-design questions remain.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Design the first narrow IT-013 red test that proves migration 73 exists and establishes the command/source/aggregate ownership baseline, then expand it incrementally to the full schema and legacy-path oracle.
- Blocking issues: None.

## Notes For Next Stage
- Preserve the exact contracts in FR-043 through FR-049 and AC-051 through AC-057 rather than reopening resolved Q13-Q19 decisions.
- Follow the FR-049 sequence: foundations, complete like/unlike reference slice, remaining set actions, posts, business records, personal profiles, then scheduled final publication.
- Extend both `appview/internal/auth/pds_client_indigo_test.go` and `appview/internal/auth/coordinated_pds_client_test.go` for production PDS wiring.
- Require each migrated Flutter feature provider to delegate keyed mutation state and overlay reconciliation to the shared controller.
- Extract a shared generated signed-CAR fixture helper from existing repository-repair tests rather than maintaining divergent snapshot builders.
- Keep `just test`, `just appview-check`, and `just app-test` as release gates. Treat `just app-analyze` as recommended non-gating evidence.
