# Document Review: Service Status and Maintenance Messaging

## Verdict

Status: Approved with notes

Reviewer: Codex document review

Date: 2026-10-09

Risk level: Medium

## Summary

The requirements and acceptance test specification are consistent with the confirmed Cloudflare-hosted JSON approach and subsequent simplifications. They are ready for coding-plan work. No blocking contradiction, unresolved product question or missing Must coverage was identified.

Reviewed `01-requirements.md` and `02-acceptance-tests.md` together. No separate `00-initial-prompt.md` exists; the initial request and confirmed decisions are embedded in requirements sections 1 and 3. Read-only repository inspection confirmed the supported-platform distinction in `app/lib/app_dependencies.dart` and the mobile-only integration harness described in `app/integration_test/README.md`. Structural checks verified reciprocal requirement/criterion links, test references and coverage-matrix completeness.

The findings below are non-blocking design notes. This verdict approves moving to coding planning; it is not implementation evidence or authorization to provision, publish or deploy production resources. Earlier workflow documents were not changed during review.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests / Traceability | Client tests prove that a new announcement revision resurfaces, but the publishing tests do not yet specify how editing an announcement obtains a new revision. If revised prose retains a dismissed revision, it can remain invisible to those users. Product intent is clear; responsibility for enforcing it remains a design choice. | `01-requirements.md` §12 FR-002/FR-006, AC-009; `02-acceptance-tests.md` UT-005, IT-003, GAP-001/GAP-003 | In the coding plan, define revision ownership and the operator workflow for revised/replacement announcements. Extend the publisher case family to show that a changed announcement receives or requires a new revision before a successful publication; link that evidence to FR-006/AC-009 without introducing mandatory publication time or a second revision field. |
| DR-002 | Suggestion | Tests / Risk | Hosted cache checks have the correct behavioral aim, but “observable via required refresh policy” and “without indefinitely renewing” need a finite pass/fail observation rule once the hosting/cache strategy is selected. The current document explicitly defers that strategy and runner. | `01-requirements.md` NFR-002, AC-016, RISK-001; `02-acceptance-tests.md` IT-007, GAP-002 | In the coding plan, specify edge/browser cache behavior and a bounded hosted assertion: after verified publication, the next qualifying foreground refresh must observe the intended document under the chosen policy. Cover native and web paths, warmed caches and unchanged-response validation if conditional requests are used. Distinguish client freshness from origin revalidation so a stale cache hit cannot indefinitely renew maintenance. |
| DR-003 | Suggestion | Tests | Recreating app/provider instances with actual preferences is a practical persistence test, but a preferences plugin cache could accidentally make a test pass without demonstrating a persisted write/read. The specification already distinguishes this from a true OS process kill. | `01-requirements.md` FR-006/FR-008, AC-009/AC-011; `02-acceptance-tests.md` AT-007, IT-002, TD-005 and §7 restart note | Define IT-002's storage oracle in the coding plan: await the dismissal write, discard feature state and reload preferences from persisted storage using the chosen API, then fetch the same revision. Avoid reusing only a populated preferences object as proof. Keep maintenance-cache restoration and durable editor storage out of scope. |

## Traceability Review

- Planning to requirements: the embedded Q1–Q14 decisions and approved simplifications are carried through. Modes are one current `normal`, `announcement` or `maintenance`; publication is manual and independent of landing-site release/rollback; custom prose is English; backend enforcement and scheduling remain excluded.
- Requirements to acceptance criteria: all **20 active Must requirements** have criteria, and all requirement-to-criterion links are reciprocal. All **17 active criteria** refer to valid requirements. Withdrawn FR-010/FR-011 and AC-013/AC-014 remain reserved.
- Acceptance criteria to tests: all **17 active criteria** have planned tests across **28 case families**: 9 acceptance, 7 unit, 7 integration, 4 regression and 1 manual. Every case links to valid requirements and criteria. Each coverage-matrix row links to cases that reference that requirement and collectively cover its listed criteria.
- Structural coverage is evidence of document completeness, not evidence that any future test passes. Design dependencies and unexecuted external checks are explicitly recorded as GAP-001–GAP-005.

## Coverage Review

- Must requirements covered: 20 of 20 in planned coverage. No missing Must coverage was found.
- Missing or weak coverage: no blocking omission. DR-001 strengthens the publication-to-dismissal link; DR-002 makes hosted cache evidence measurable; DR-003 strengthens the persistence oracle. These can be resolved in coding planning without reopening product scope.
- Timing and freshness: UT-003/UT-004 own the three-second timeout, concurrent startup, all-mode 60-second foreground polling, trigger coalescing, five-minute trust boundary, unchanged-success renewal, failed/invalid non-renewal and stale-result protection. UI cases cross-reference those tests rather than repeating clock suites.
- State and fallback: AT-001/AT-004–AT-007 and REG-001–REG-003 cover account-independent startup maintenance, preserved editor/navigation/session state, current-account recovery gates, existing in-flight outcomes, no replay, unknown failures and offline restart without restored maintenance status.
- Public/operator boundaries: IT-001/IT-003–IT-007 cover anonymous HTTP, bounded validation, confirmation before mutation, honest public verification, website independence, credential separation and serialized diagnostics. Real hosting and web CORS/cache evidence remains pending under GAP-002.
- Manual-only coverage: real screen-reader interaction is MAN-001. Layout, semantics, localization, text scaling and estimate formatting also have automation targets. Manual effort is appropriately limited to behavior that widget assertions cannot fully demonstrate.
- Test practicality: proposed Flutter tests fit existing widget/Riverpod fixtures; device integration fits the existing Android/iOS suite; Python command tests fit script-test conventions. The mobile suite is not misrepresented as web coverage. The hosted browser runner must be selected during planning.

## Risk And Approval Review

- Risk level: Medium, unchanged from requirements and test design. State retention, stale blocking, publication independence and public-content handling have concrete verification paths.
- Review requirement: the recommended document review has been completed. No newly identified High-risk scope or blocking product ambiguity requires further approval before coding planning.
- Approval notes: the user approved requirements and simplifications and explicitly selected document review after test design. That authorizes this review artifact. It does not automatically advance to coding planning or implementation.
- Provisioning and release evidence: Cloudflare permissions/resource availability are unverified. GAP-002 and GAP-004 must be tracked as incomplete hosted/accessibility evidence until executed. Any production resource or publishing action remains subject to repository change controls and explicit authorization.
- Scope protection: no AppView admission changes, worker controls, migrations, lexicon/PDS changes, new offline durability or external-link behavior is implied. Existing diagnostic guidance and account isolation remain mandatory.

## Coding Plan Readiness

- Ready for coding planning: **Yes**.
- Recommended first step: choose the JSON schema/revision representation and finite body/text bounds, then plan UT-001 as the first failing test accepting supported maintenance without publication time. Specify the revision workflow at the same time (DR-001).
- Blocking issues: **None for coding planning**. GAP-001/GAP-003 are design outputs the plan must resolve; GAP-002/GAP-004 are execution/evidence dependencies, not unresolved product decisions. GAP-005 preserves existing external OAuth/media test limitations.
- Implementation readiness: not established by this review. A concrete coding plan and the workflow's next-stage selection are still required before implementation.

## Notes For Next Stage

- Create `04-coding-plan.md` using `write-coding-plan` only after the user selects that stage. Preserve all requirement, criterion, test and finding IDs.
- Resolve DR-001–DR-003 and record their design decisions/test refinements in the coding plan. Earlier documents need no blocking rewrite; revise them only if the user requests it or approved scope changes.
- Choose the independent Cloudflare resource/publication adapter, credential source, explicit redirect policy, cache headers/revalidation behavior, exact CLI interface and isolated environment strategy. Do not silently bundle live status into website assets.
- Describe how status fetch/presentation lives above failed startup/account initialization while preserving the existing router/editor subtree and account gates. Keep retrieval concurrent and anonymous.
- Reuse existing test fixtures and recipes. The device recipe runs `critical_journeys_test.dart` specifically; extend it or explicitly plan the runner change if introducing another suite. Select a separate web/hosted harness under GAP-002.
- Keep the test order from `02-acceptance-tests.md` §11, with timing logic tested once and external verification clearly separated from mocks.
- Record hosted cache/CORS and screen-reader checks as pending until observed. Do not claim a provisioned status endpoint, successful live publication or production readiness from document or mocked-test evidence.
- No source/test implementation, dependency changes, infrastructure mutations, commits or pushes were performed in this stage.
