# Implementation Review: Business Profiles

## Verdict
Status: Approved
Reviewer: OpenCode
Date: 2026-08-29
Risk level: High

## Summary

The implementation satisfies the amended Business Profiles AppView requirements and is ready for merge or handoff. IR-001 through IR-009, IR-011, and IR-012 are resolved; IR-010 was reviewed as covered by existing release migration, deletion-order, registry, and lifecycle-retry suites. The post-review catalog simplification delegates country and currency data to the already pinned `golang.org/x/text` module, removes the custom generator and snapshots, and passes the complete release-equivalent gate.

## Findings

None identified.

## Requirement And Test Traceability

- Requirements implemented: All amended Must requirements have corresponding implementation and passing automated or approved manual evidence. Private account type, public records, package-backed country/currency validation, read-time eligibility, block/moderation policy, projection ordering, deletion ordering, route contracts, and neutrality constraints match the documents.
- Tests implemented: Source tests exist for UT-001 through UT-023, IT-001 through IT-021, AT-001 through AT-014, and REG-001 through REG-008. MAN-002 is recorded complete.
- Unplanned behavior: Migration `000063_business_event_moderation` splits moderation work assigned to `000062` in the coding plan; the implementation plan now documents and justifies the deviation.
- Remaining gaps: None blocking. The approved full-real-PDS acceptance gap remains covered by contract-faithful PDS effect fakes plus real-Postgres projection tests.

## Test Evidence

- Commands reviewed: Focused `go test` for business/schema/index/ingestion/API packages; `just test`; `just lexgen-check`; `go vet ./...`; `go mod tidy -diff`; `docker compose config --quiet`; `git diff --check`; `just appview-check`.
- Passing evidence: Focused privacy tests first reproduced and then closed the ordinary and focused flattened-parent leak for both block directions. The routed cursor test obtains a valid management cursor through `AddRoutes`, tampers it, and verifies the authenticated standard error envelope. Package-backed validator tests cover ordinary and exceptional country/currency behavior plus explicit exclusions. Focused packages, `just test`, and all static/module/config/diff checks passed. `just appview-check` passed all release gates. Current artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.yUEXehEmwm`.
- Failing or skipped tests: None. The gate reports only the existing policy-approved `GO-2026-5932` exception.

## Risk Review

- Risk level: High.
- Risk notes: The feature remains High risk because it combines private classification, broad response hydration, durable federated records, moderation, and permanent deletion. Country/currency behavior now follows reviewed `golang.org/x/text` upgrades rather than first-party snapshots; representative tests protect Craftsky-specific casing, exclusions, and decimal grammar.
- Approval notes: Approved for merge or handoff. IR-001 through IR-012 are resolved or explicitly closed as already covered; no blocking or non-blocking implementation finding remains.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: Flutter and other user-facing UI changes are outside this implementation slice.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None.
- Verification to rerun: None before handoff. Run the normal required CI/release checks again if the implementation changes after this review.
