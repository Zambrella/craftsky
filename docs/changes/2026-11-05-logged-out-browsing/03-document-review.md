# Document Review: Logged-out browsing

Scope revision on 6 November 2026: the user subsequently limited guest search to basic submitted-query results, excluding recent users, autocomplete/suggestions and top hashtags. Craft project feed sort controls stay available. Requirements, test design and coding plan have been updated; discovery-related references below describe the historical review and are superseded by that revision.

## Verdict

Status: **Approved with notes**

Reviewer: Codex, document consistency and coverage review.

Date: 2026-11-05.

Risk level: **High**.

## Summary

The requirements and acceptance test specification are ready for coding-plan work. No blocking contradiction, uncovered Must requirement or unresolved product decision was identified. All 21 Must requirements have acceptance criteria and planned test coverage; all 17 acceptance criteria have concrete verification paths across 43 planned tests/checks.

The user approved the requirements and all assumptions, explicitly excluded backfill, and requested test design followed by this review. Both documents consistently retain that amendment: unknown legacy visibility stays unavailable to guests until normal incoming profile processing or an explicit owner settings operation establishes it. Neither a catch-up job nor guest-triggered repair is permitted.

This is a planning review, not implementation approval or test execution evidence. Only this review artifact was created in this stage; the reviewed documents were not rewritten. No source, tests, dependencies, migrations, credentials or live resources were changed. No commit or push was requested or performed.

## Findings

All findings below are non-blocking coding-plan notes. They identify concrete implementation/validation details within the approved behavior; they do not add product scope.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk / Tests | Existing profiles can already have the same record CID as a normally delivered event while their new visibility projection remains unknown. Both inspected indexer upserts currently skip an unchanged CID. Blindly retaining that condition could leave authoritative normal events unable to establish visibility. | Requirements RULE-004, AC-017, section 15; test design IT-001, IT-006, IT-012; `appview/internal/index/bluesky_profile.go` upserts | In the coding plan, specify how normal-event projection establishes an unknown state even for an unchanged CID, in both indexer paths, while repeated established events remain idempotent. Include that concrete legacy-row/same-CID fixture under IT-006. Do not enqueue replay, fetch missing profiles or run a population sweep. |
| DR-002 | Suggestion | Risk / Tests | Owner saves must suppress guests before success is acknowledged, while external updates converge through normal ingestion. The plan must reconcile these paths so a delayed pre-save event cannot undo a newer local decision. A CID is an identity, not an ordering sequence. | Requirements FR-009, RULE-004, AC-009–010, EC-005; test design IT-005–006 and their ordering-boundary note | Identify the actual mutation/ingestion ordering boundary and partial-success recovery in the coding plan. Place deterministic delayed-event/conflict tests there. Do not assume standalone indexers provide ordering or invent an externally instantaneous propagation guarantee. |
| DR-003 | Suggestion | Tests / Coding-plan readiness | The current public Playwright configuration matches only `public-pages.spec.ts` and disables JavaScript. Interactive scenarios have appropriate proposed targets but no concrete runnable project yet; adding a spec alone would not make it run. | Test design sections 1, 3 and 11, GAP-001; `web-client/playwright.config.ts` | Name the JS-enabled guest project/command, fixture additions and browser/device matrix in the coding plan. Preserve no-JS public permalink/metadata checks and include the new guest project in validation explicitly. Reuse existing fixture infrastructure where practical. |
| DR-004 | Suggestion | Requirements / Tests | Public profile sections and applicable search controls are intentionally described through existing behavior. A concrete route/field inventory is still needed to prevent accidentally dropping public reading paths or exposing member-only state while reusing components. | Requirements FR-002–003, FR-006, FR-010, ASM-004–007; test design AT-003–005, IT-003–004, IT-008, GAP-003 | Inventory existing profile tabs, counts/identity lists, search filters/suggestions, thread/embed types and shared components during coding planning. Map each to its guest contract and tests. Preserve permitted aggregate totals while excluding opted-out identities; do not require new social graph screens or redact independent third-party mentions. |

## Traceability Review

- **Planning to requirements:** The original conversation, attached privacy screenshot, shared-preference answer and subsequent approval/no-backfill instruction were reviewed against requirements sections 1–5, 8 and 20–22. A separate `00-initial-prompt.md` is absent; the conversation and section 1 provide the initial request. No additional prompt artifact is needed.
- **Requirements to acceptance criteria:** All 21 Must requirements (BR-001–002, FR-001–011, NFR-001–004, RULE-001–004) have valid AC references. Each forward requirement/criterion reference is reflected in the criterion's requirement IDs. No missing or dangling IDs were found.
- **Acceptance criteria to tests:** All AC-001–017 appear in planned test/check definitions. The requirement coverage matrix covers all 21 requirements, its test IDs resolve, and its listed criteria are present in the linked test cases. The count is 12 acceptance scenarios, 9 unit cases, 12 integration cases, 7 regression cases and 3 manual checks.
- **Scope amendment:** RULE-004 and AC-017 were amended without renumbering. IT-001/006/012 and AT-009 test unknown state, normal-event establishment and absence of backfill/repair. Old requirements for a backfill mechanism have been removed.
- **Automated document checks:** Parsed both Markdown artifacts to verify requirement/AC linkage, matrix completeness, test ID resolution and coverage of each matrix criterion. These checks validate document structure and traceability, not application behavior.

## Coverage Review

- **Must requirements covered:** All 21 have planned coverage. Guest craft feeds, search, public detail and comments, modal/banner, hidden account controls, sign-in return, shared Privacy preference, metadata, accessibility and member-state isolation are represented.
- **Privacy coverage:** Positive eligible content plus distinct protected-value canaries are specified for raw API JSON, initial HTML, hydration, OG/structured data and preload URLs. Mixed-author threads/quotes/reposts, unknown legacy state, public moderation/lifecycle/image checks and member credential spoofing have concrete cases.
- **Persistence/mutation coverage:** Tests cover both profile projection paths, preserved fields/labels, concurrent edits, PDS/local failure, duplicate events, delayed ordering and owner acknowledgment. DR-001/002 make the key fixtures/boundaries explicit for planning.
- **No-backfill coverage:** Migration/startup/repeated guest reads must leave legacy unknown state hidden without profile repair traffic, replay or sweep. Natural incoming events and explicit owner operations remain allowed; existing necessary handle resolution must be distinguishable from prohibited profile repair. No unrelated existing ingestion machinery needs to be removed solely for this feature.
- **Missing or weak coverage:** No requirement-level gap blocks planning. Concrete routes, field inventory, runner commands and ordering seams remain coding-plan work already acknowledged by GAP-001–003 and DR-001–004. Component reuse requires implementation review as well as behavioral parity; the test design correctly avoids brittle import-string tests as proof.
- **Manual-only coverage:** Native screen reader usability, OS share-sheet behavior and final visual safe-area quality reasonably supplement automated semantics/layout tests. Optional real-PDS/Bluesky interoperability smoke corroborates mandatory local contract/integration tests. It does not replace them or authorize live account mutations in this stage.
- **Evidence boundaries:** No tests have run. Browser mocks do not prove server SQL policy, local PostgreSQL tests do not prove production, and no-store/safe metadata does not revoke third-party cached previews. The documents consistently distinguish these limits.

## Risk And Approval Review

- **Risk level:** High, unchanged. Key risks are anonymous data exposure, cross-app PDS profile writes, nested presentation leaks and stale viewer/projection state.
- **Review requirement:** The requirements-stage high-risk approval was supplied by the user. This review establishes readiness for coding planning; implementation still requires explicit authorization under the staged workflow.
- **Approval notes:** Shared Bluesky preference and ASM-001–009 are confirmed. No-backfill is a direct user constraint and takes precedence over the original backfill proposal. The approved failure behavior is to hide unknown legacy content until authoritative normal processing establishes visibility, without inventing a refresh job.
- **Operational boundary:** No production changes, broader OAuth scopes, new custom Lexicons, account record deletion or release/deploy action is authorized. If the chosen design later changes a schema under `lexicon/`, the repository's Lexicon skill/ADR rules still apply; the approved direction currently needs no custom Lexicon change.
- **Diagnostics:** If implementation changes diagnostics, the mandatory repository logging/error-reporting guide still governs cause preservation, privacy context and serialized-sink/canary verification. The planning documents introduce no new analytics requirement.

## Coding Plan Readiness

- **Ready for coding planning:** Yes.
- **Recommended first step:** Resolve DR-001–004 into concrete component, API and persistence boundaries while preserving the approved requirements. Start the later test-first implementation with IT-001's unknown/no-backfill case, then IT-006's ordinary-event projection for both indexer paths, including same-CID unknown legacy state.
- **Blocking issues:** None identified for coding planning. This verdict does not authorize implementation, migration execution or deployment.
- **Reviewed documents:** `01-requirements.md` and `02-acceptance-tests.md` in this folder. Neither was modified during review.

## Notes For Next Stage

- Use `write-coding-plan` after the user chooses to proceed. Map each planned slice to the existing requirement, AC and test IDs; keep them stable.
- Prefer shared existing client rendering and shared AppView eligibility/read logic with explicit guest-safe response fields. Exact route consolidation is a design choice, not an instruction to expose all member read endpoints.
- Preserve the public SSR/no-JS profile/post path while adding the interactive shell, and document how metadata consumes the same eligibility result as clients.
- Choose and document a concrete guest-session lifetime for modal recurrence using existing client lifecycle conventions. Navigation within a session must not reprompt; direct public links remain unobstructed. This is a routine implementation definition within confirmed ASM-002, not a reason to reopen the approved product decision.
- Keep the no-backfill constraint visible in storage defaults, composition/startup, migration review, test fixtures and operational notes. Do not label unknown as public to make a legacy fixture or empty feed pass.
- Preserve account-switch/onboarding/callback rules and eliminate stale private data on guest transitions. Retain member regression coverage.
- Record local automated, device/manual and live-provider results separately when those stages are executed. No current check result should be reported as a passed product test.
