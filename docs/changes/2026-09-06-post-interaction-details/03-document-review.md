# Document Review: Post Interaction Details

## Verdict

Status: Approved with notes

Reviewer: OpenCode

Date: 2026-09-06

Risk level: Medium

## Summary

The reviewed requirements and acceptance-test specification consistently describe three authenticated, cursor-paginated AppView list endpoints; nonzero-only top-level detail links; comment/reply liker access through the existing more-actions menu; account lists for likes/reposts; post lists for quotes; and preservation of current mutation behavior. Every Must requirement links to acceptance criteria and automated tests, all 24 acceptance criteria have coverage, and no blocking product or architecture question remains.

The documents are ready for coding planning. Post-review grilling resolved the exact ordering and detailed eligibility policy. The planner should now map those decisions onto shared query predicates and consolidate proposed tests into the smallest practical set of files.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Resolved | Requirements / Tests | Post-review grilling selected `(created_at DESC, uri DESC)` for likes, reposts, and quote posts. | `01-requirements.md` Q5, FR-003, FR-004, FR-006, NFR-002; `02-acceptance-tests.md` UT-008, IT-010 | Ensure SQL ordering, seek comparisons, cursor encoding, and tests use this exact tuple and direction. |
| DR-002 | Important | Risk / Traceability | Count/list policy is now explicit: current members only for account lists; terminal, viewer-actor block, actor-author block, and hide/takedown exclusions; mute and warnings included; quote-language filtering; revealable muted quotes included. Implementation could still drift if counts and lists duplicate these joins independently. | `01-requirements.md` Q7-Q8, FR-010, AC-012, AC-017, RISK-001; `02-acceptance-tests.md` IT-004, IT-011, IT-012, GAP-004 | Identify authoritative helpers/query predicates or define one shared interaction eligibility seam so count and list paths use identical policy. |
| DR-003 | Suggestion | Tests | The automation targets propose several new narrowly named Go and Flutter files. This is traceable but may fragment closely related fixtures and make the TDD sequence heavier than necessary. | `02-acceptance-tests.md` UT-002 through UT-008, IT-001 through IT-014, handoff test order | During coding planning, consolidate cursor/store/policy/hydration cases into existing or a small number of cohesive suites when package boundaries permit. Preserve test IDs in test names/comments or implementation tracking even if file targets change. |
| DR-004 | Suggestion | Tests / Accessibility | Core accessibility behavior is automated, while final VoiceOver/TalkBack phrasing and platform-menu focus behavior remain manual. This is appropriately disclosed and non-blocking, but it needs explicit completion evidence before merge. | `01-requirements.md` NFR-003, AC-020; `02-acceptance-tests.md` AT-007, MAN-002, GAP-001 | Include MAN-002 in the coding plan's final verification checklist and record the platform/device used. Do not treat widget semantics tests alone as completion of the manual assistive-technology check. |

## Traceability Review

- Planning to requirements: The selected detail-summary/dedicated-route approach is preserved. User decisions now also pin exact ordering, mute/member/block/language eligibility, Quotes title/contract, loading and unavailable states, summary wrapping, and response-menu grouping.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` references at least one acceptance criterion. Stable IDs were preserved when review feedback added BR-004, FR-013, RULE-005, and AC-024.
- Acceptance criteria to tests: AC-001 through AC-024 each appear in at least one acceptance, unit, integration, regression, or justified manual test. Core behavior is automated; only real platform assistive-technology behavior is manual-only in part.

## Coverage Review

- Must requirements covered: BR-001, BR-002, BR-004; FR-001 through FR-013; NFR-001 through NFR-004 and NFR-006; RULE-001 through RULE-005.
- Should requirements covered: BR-003 is covered by AT-003, AT-004, and AT-006. NFR-005 is covered by IT-014.
- Missing or weak coverage: None blocking. Exact SQL/cursor ordering is resolved; mutable cross-request count/list equality remains correctly limited to unchanged data and recorded in GAP-004.
- Manual-only coverage: MAN-001 covers final responsive visual quality; MAN-002 covers real screen-reader and keyboard/platform-menu behavior. Automated widget semantics and layout tests cover the deterministic portions first.

## Risk And Approval Review

- Risk level: Medium, unchanged from requirements and test design.
- Review requirement: Review was recommended rather than mandatory. The user reviewed `01-requirements.md`, supplied five annotations, and those decisions are reflected in both reviewed documents.
- Approval notes: Coding planning may proceed. DR-001 is resolved. DR-002 must become an explicit shared-query/predicate decision in the coding plan but requires no further product answer.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Map current post hydration, engagement counts, profile summaries, moderation/relationship/language predicates, and cursor helpers into one AppView read design using the approved `(created_at DESC, uri DESC)` order.
- Blocking issues: None.

## Notes For Next Stage

- Keep the three public GET route shapes exactly as reviewed; GET must coexist with existing POST/DELETE likes/reposts routes without changing mutation contracts.
- Make the Likes read path accept top-level posts, comments, and nested replies. Keep Reposts and Quotes top-level-only.
- Build comment/reply Likes navigation from the selected response's own DID/rkey, never the root post identity.
- Keep zero-count visibility as a presentation rule: omit the top-level link or response menu action, while direct routes still support valid empty states after concurrency/policy changes.
- Prefer one reusable account-list page/provider mode for Likes and Reposts and a separate post-list path for Quotes; do not force both item types into one polymorphic abstraction.
- Reuse existing `ProfileAccountSummary`, `ProfileAccountPage`, `Post`, `PostPage`, profile presentation, `PostCard`, `AutoPaginatedListView`, API envelope, route policy, and authenticated-shell conventions.
- Include query-plan verification only if the selected SQL/index path needs it; the requirements permit no migration when existing active-subject and quote URI indexes are sufficient.
- Preserve requirement, acceptance-criterion, and test IDs in the coding and implementation plans for traceability.
