# Document Review: Edit Published Posts

## Verdict

Status: Approved with notes

Reviewer: Codex, document consistency and test-design review

Date: 2026-10-26

Risk level: Medium

## Summary

The document set is ready for coding-plan work. It reflects the maintainer's confirmed direction: owner editing at any age through the context menu, fully prefilled corresponding composer, `Save changes`, latest edit indicator/time, no feature edit history, latest content in reposts/quotes, and no general/project conversion. The accepted safeguards for identity, conversation, media preservation, conflicts and moderation are retained.

All 24 Must requirements have reciprocal acceptance-criterion links and planned test coverage. All 26 acceptance criteria appear in the test coverage matrix. There are no missing Must-coverage blockers. The remaining notes concern decisions and concrete test detail the coding plan must settle, rather than a need to change the approved product direction.

This verdict approves readiness for **coding planning**, not implementation, a specific schema, the final code or a release. Earlier artifacts remain unchanged, including their Draft statuses and labeled assumptions. No tests, source changes, schema changes, migrations, commits or live actions were performed during this review.

Reviewed inputs:

- Conversation discovery, UX choices and explicit maintainer confirmations; no separate `00-initial-prompt.md` exists or is required for this workflow.
- [01-requirements.md](01-requirements.md), including discovery, BR/FR/NFR/RULE/AC/EC IDs, risks and assumptions.
- [02-acceptance-tests.md](02-acceptance-tests.md), including coverage matrix, scenarios, test levels, fixtures, gaps and handoff.
- Narrow verification of checked-in test commands in `justfile` and `scripts/appview-test`; prior inspected test structures remain appropriate targets.

## Findings

Findings below are non-blocking for writing a coding plan. Important design notes must be closed in that plan before affected implementation is approved. They are not a license to silently choose new product behavior.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Data / Test precision | The documents deliberately defer timestamp storage/authority and external-update fallback. The core app-edited T1/T2 behavior is testable, but external edits with absent, removed or untrustworthy metadata and first-seen snapshots lack a finalized observable timestamp contract. This is expected design work, not missing core editing scope. | Requirements FR-006, FR-007, FR-012, AC-006, AC-019, sections 15/21; Tests IT-010, IT-014, GAP-001 | In the coding plan, specify latest edit metadata source, persistence, API presentation and behavior for known external updates versus first-seen snapshots. Include missing/malformed/future metadata, no-op/replay and restart/rebuild cases. Never present first-observed time as a proven exact author edit time. Finalize test expectations against that contract. Follow the Lexicon/ADR workflow if public record metadata changes. |
| DR-002 | Important | Notification races | Tests identify but do not fully enumerate outcomes when the first publication's notification work is still pending and an edit adds/removes mentions. "Preserve existing publication semantics" must not accidentally become permission to notify recipients introduced only by the edit. | Requirements RULE-004, AC-020, ASM-003; Tests IT-012, TD-008, GAP-005, REG-007 | The coding plan must define how publication and update are distinguished without introducing edit-history storage. Add named cases for initial work pending, original recipient already notified, newly added recipient, removed recipient, duplicate update and retry. Assert no notification caused by the edit and no reactivation/reset of existing notification state. If an original pending delivery's treatment needs a product decision, identify that narrowly before implementation. |
| DR-003 | Suggestion | Test specificity / Scope | The per-field and surface dimensions are comprehensive, but the exact field inventory and entry-point fixtures remain delegated to implementation. Fixed structural relationships also need absent→present and present→absent cases, not only replacement with a different target. | Requirements FR-002–FR-005, RULE-003, AC-003, AC-009, AC-018; Tests section 1, AT-002, UT-001, UT-002, UT-008, IT-006, REG-005 | In coding planning, map current composer fields and shared presentation/menu entry points to named fixtures. Include adding/removing reply/quote context, imported provenance, each supported project variant and optional-field clears. Preserve fixed absence/presence as well as existing target identity. Avoid a full Cartesian product or a new form/schema framework solely for testing. |
| DR-004 | Suggestion | Validation instructions | Requirements section 2 says `just app-test` includes analysis; the recipe runs Rive setup and Flutter tests only. The acceptance-test handoff already gives the correct separate analysis command. | Requirements section 2, final bullet; Tests section 11; `justfile` recipes `app-test` and `app-analyze` | Use `just app-test` plus `just app-analyze` as separate checks in the coding plan. Correct the requirements' command description during an authorized earlier-document revision. This wording mismatch does not affect behavior or coverage. |

## Traceability Review

- Planning to requirements: consistent. The comparison with other platforms informs familiar UX without importing their time limits, paywalls or revision-history features. The maintainer explicitly accepted latest-view quote semantics and fixed structural relationships, then prohibited general/project conversion.
- Requirements to acceptance criteria: all 24 Must requirements link to one or more criteria; all 26 criteria reference existing requirements. A structural check confirmed reciprocal links. No unreferenced Must requirement was found.
- Acceptance criteria to tests: all 26 criteria are represented in the matrix; every Must requirement has defined test IDs. Acceptance scenarios and unit/integration/regression/manual rows carry requirement and AC references. All referenced test/data/gap IDs resolve to definitions.
- IDs reviewed: 8 acceptance scenarios, 9 unit cases, 17 integration cases, 8 regression cases, 3 manual checks, 9 data fixtures and 6 gaps. Counts describe planned cases, not executions or demonstrated coverage.
- Assumptions: ASM-001–ASM-005 are labeled consistently rather than disguised as separately confirmed answers. Replies/imported posts, no alerts for new mentions and no durable edit-draft feature remain visible for maintainer review. The current artifact set can be planned as written without further broad discovery; any proposed departure must be explicit.
- Scope boundaries: no history archive, conversation retargeting, republishing, scheduled edits, extra edit alerts or unsupported native desktop platforms have entered scope. Operational/moderation retention remains separate from the unwanted product history feature.

## Coverage Review

- Must requirements covered: BR-001–BR-002, FR-001–FR-012, RULE-001–RULE-007 and NFR-001–NFR-003.
- Main user journeys: entry from existing owner menus, complete prefill, field/media changes and clears, save/cancel, edit disclosure, latest quote/repost content, stale-save rejection and error recovery.
- High-consequence boundaries have concrete paths: remote conditional-write race (IT-004), accepted write with lost response (IT-005), deletion/recreation prevention (IT-006), account/lifecycle fencing (IT-003), unknown-field preservation (IT-009), scan/enforcement/evidence preservation (IT-013) and actual serialized diagnostic canaries (IT-017).
- Missing or weak coverage: no entire Must requirement lacks coverage. Timestamp edge expectations and delayed-publication recipient cases need the refinements in DR-001/DR-002. Named field/surface fixtures should be finalized under DR-003.
- Manual-only coverage: real VoiceOver/TalkBack/browser assistive behavior and native picker interaction have justified manual checks. Widget assertions cover basic interaction/semantics contracts; the native loopback runner covers UI-to-socket behavior. It cannot claim web, real PDS, real scanning or push-provider proof.
- No-history verification: tests correctly pair product/API checks with design/schema inspection and retain existing command/moderation evidence policies. The review does not promise earlier public versions cannot exist elsewhere.
- Automation practicality: proposed Flutter suites follow existing Riverpod/widget patterns; Go suites use existing fake PDS and disposable migrated PostgreSQL patterns. No additional external service or test framework is mandated.
- First failing test: AT-001's general-post caption edit supported by IT-001 is a coherent vertical slice. Test fixed identity/conditional update and absence of create side effects before expanding to media and shared surfaces. Resolve the timestamp contract before implementing its persistence assertions.

## Risk And Approval Review

- Risk level: Medium. The change spans persistent public record updates, shared composer/rendering paths, asynchronous projection, moderation and notifications. It reuses existing authority boundaries rather than introducing a new privilege model.
- Review requirement: document review was recommended and has now been performed at the maintainer's request. No delegated-agent review or Plannotator review was requested or performed.
- Approval notes: the maintainer confirmed the product direction and authorized requirements, acceptance-test design and this review. This report does not authorize source implementation, production changes or commits. There is no extra permission requirement inferred solely from using existing authentication/moderation controls.
- Next-stage gate: coding planning can proceed if the maintainer chooses it. The coding plan must explicitly close DR-001 and DR-002's design work before implementation approval; these are not blockers to writing that plan.
- Evidence boundary: structural document checks passed. No runtime, device, live federation or production evidence was produced by document review.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: define the edit read/write/version contract and latest edit metadata representation, then map it onto existing addressed commands, current-source projections and composer hydration. Include delayed notification classification in the same design pass.
- Blocking issues for coding planning: none identified.
- Design work before implementation: DR-001 timestamp/external-update contract and DR-002 concrete notification-race outcomes; finalize affected test expectations while preserving IDs.
- Other notes: use DR-003's explicit field/surface inventory to prevent omissions and DR-004's corrected validation commands. Do not start coding or run migrations as part of coding planning.

## Notes For Next Stage

- Produce `04-coding-plan.md` in this folder only after the maintainer requests continuation. Carry these four findings forward and show how each is addressed or remains an explicit dependency.
- Preserve `01-requirements.md` and `02-acceptance-tests.md` unless the maintainer explicitly chooses an earlier-document revision. This review intentionally changes neither.
- Keep the plan proportional: reuse the existing composers, conditional commands, owner lifecycle fences and ingestion machinery; avoid an edit-history system, generic patch engine or unrelated architecture refactor.
- Explain how unchanged remote media, unsupported extension fields and original timestamp strings survive round trips. Model fixed quote/reply context and general/project type on both client and server boundaries.
- Decide how accepted writes and subsequent reads reconcile without stale rollback, and how successful content saves receive stable latest-edit metadata across retries.
- Separate iOS/Android integration, Flutter widget/analysis, backend gate and browser/device checks in validation evidence. `just app-test` alone does not include static analysis.
- No stage commit was requested. The review artifact is left uncommitted alongside the existing workflow documents.
