# Document Review: Settings Page And Account Management

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-08-10
Risk level: High

## Summary

The revised requirements and acceptance tests are ready for coding planning. The Settings, About, Account, Notifications, switching, cache, version, and Sign out behavior remains clear and fully traceable. The destructive deletion scope remains limited to the confirmed CraftSky membership/private-data boundary and the owner's `social.craftsky.*` PDS record collections; it continues to preserve the AT Protocol/PDS account, other namespaces, shared blobs, and PDS-owned garbage collection.

The prior blocking findings are resolved. The documents now record the approved narrow PDS-authority exception, separate ordinary client sessions from a server-only deletion OAuth binding, define receipt-backed AppView convergence without adding eager deletion, anchor audit expiry to terminal success, distinguish non-terminal operational state from post-success retention, and enumerate the current private-data cleanup manifest with a future completeness gate. Remaining items are implementation-design or controlled manual-release work, not unresolved product decisions.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk | The narrow owner-reauthenticated PDS-deletion exception is now explicitly approved and bounded in the requirements, but the repository guidance/reference still contains the older blanket prohibition. | `01-requirements.md` Approved Architecture Decisions, BR-003, RULE-002, RULE-005, RULE-009, Review Status; `02-acceptance-tests.md` GAP-008, REG-009, MAN-003 | Coding planning must place the repository guidance/reference amendment before destructive route enablement. The amendment must preserve the whole-account, other-namespace, and direct-blob prohibitions. This does not block planning. |
| DR-002 | Suggestion | Risk | The credential and convergence contracts are approved, but their exact transaction, schema, and interface layout intentionally remains a coding-design choice. | FR-020, FR-021, FR-023, RULE-007; AC-037, AC-040 through AC-044; AT-006, AT-008, AT-011, AT-012; GAP-001, GAP-002 | The coding plan must preserve bind-before-revoke, worker-only OAuth resume, expected-URI-before-delete, post-handler/pre-ack receipts, final empty rescan, absent/retracted effects, and operational-state removal. Do not substitute eager hiding or a generic background OAuth selector. |
| DR-003 | Important | Tests | The current private-store and Instagram manifests are explicit, but future migrations can add new owner-private or indirect shared-resource surfaces. | FR-015, FR-026, RULE-010, Data / Persistence Impact; `02-acceptance-tests.md` IT-012, TD-008, TD-009, GAP-004 | Design one maintained deletion-coverage manifest and the IT-012 schema/completeness test so a new owner-private store cannot ship without an explicit delete or retain policy. This is required before implementation completes, but does not block coding planning. |
| DR-004 | Suggestion | Risk | The required Customisation row remains an adjacent-work dependency absent from this checkout. | FR-003, EC-016, RISK-005; `02-acceptance-tests.md` REG-001, GAP-007 | Record the merge dependency in the coding plan and run REG-001 once the route is present. Preserve the stable row requirement meanwhile. |

## Traceability Review

- Planning to requirements: Complete. The initial request, supplied screenshot, all twelve clarified decisions, chosen Settings hierarchy, destructive-operation boundaries, approved architecture decisions, data inventory, security model, risks, assumptions, and non-goals are represented in `01-requirements.md`.
- Requirements to acceptance criteria: Complete. There are 48 structured requirements: 47 Must requirements and one Should requirement. Every requirement links to at least one valid acceptance criterion, and all 48 acceptance criteria link back to valid requirements.
- Acceptance criteria to tests: Complete. All 48 acceptance criteria are referenced by defined tests; all requirement, acceptance-criterion, and test references resolve; no duplicate test IDs exist; and all 83 defined tests are referenced.

## Coverage Review

- Must requirements covered: All 47 Must requirements have acceptance, unit, integration, regression, or justified manual coverage. The specification defines 15 acceptance scenarios, 24 unit tests, 28 integration tests, 12 regression tests, and 4 manual checks.
- Missing or weak coverage: None blocking. Exact route/payload/schema/retry literals, the approved receipt/OAuth interface placement, final support copy, Customisation merge order, and future private-manifest enforcement are correctly identified as coding-plan inputs. Provider variation and unreachable-device storage remain controlled manual boundaries.
- Manual-only coverage: No Must requirement relies only on manual validation. Manual checks complement automation for final responsive visuals, real assistive technology, disposable real OAuth/PDS/Tap behavior, and offline secondary-device cleanup.

## Risk And Approval Review

- Risk level: High. Permanent deletion crosses client state, ordinary/status/deletion-worker authorization, durable jobs, multiple PDS collections, private and shared stores, Instagram state, Tap/indexers, multiple devices, and timed retention.
- Review requirement: Satisfied for coding planning. A later implementation review and disposable real-stack release check remain required.
- Approval notes: The product owner approved the narrow `social.craftsky.*` PDS-deletion exception. The revised documents prove that this is not whole-account deletion, do not create a generic PDS-delete facility, keep PDS credentials out of Flutter, retain only job-bound deletion authority while non-terminal, and leave public AppView deletion in existing indexers. The checked-in repository guidance must be reconciled before the destructive implementation is enabled.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Keep `UT-009` as the first failing implementation test to establish the complete `social.craftsky.*` record-collection boundary before destructive worker code. The coding plan should then order the OAuth-binding and authorization tests before PDS deletion, followed by durable job/private manifest work and receipt-backed convergence.
- Blocking issues: None for coding planning. Findings DR-001 through DR-004 are mandatory plan/release notes, not product blockers.

## Notes For Next Stage

- Preserve all existing requirement, acceptance-criterion, and test IDs in the coding plan.
- Model acceptance as one durable boundary: validate fresh reauthentication and exact handle, create/reuse the job, bind the exact OAuth session ID, revoke/remove ordinary bearer and unbound OAuth sessions, then issue separate status access.
- The worker may resume only the OAuth session bound to its job and only for the required PDS list/delete operations. Replacement requires fresh OAuth reauthentication from status and must never create ordinary access.
- Register each expected URI before its PDS delete. Record an idempotent receipt only after the existing indexer succeeds and before Tap acknowledgement; a handler or receipt failure must leave the event replayable.
- Terminal success requires the complete receipt set, absent/retracted indexed effects, a final empty CraftSky PDS rescan, private-manifest cleanup, ordinary-session removal, and deletion OAuth removal. Then remove operational job/status/expected-URI/receipt state and retain only the minimized audit.
- Anchor audit expiry to `terminalSuccessAt + 30 days`, including exact before/at/after boundary tests.
- Put the repository guidance/reference amendment and the Customisation merge dependency explicitly in `04-coding-plan.md`.
