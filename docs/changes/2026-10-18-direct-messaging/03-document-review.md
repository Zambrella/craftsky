# Document Review: Direct Messaging

## Verdict

Status: Approved with notes  
Reviewer: Codex, document-set review  
Date: 2026-10-19  
Risk level: High

## Summary

The requirements and acceptance-test specification are ready for coding-plan work. All 35 Must requirements link to acceptance criteria and planned tests, and all 29 acceptance criteria have a corresponding acceptance scenario. No blocking product question, contradictory authoritative requirement or missing Must coverage was identified.

The review covers `01-requirements.md` and `02-acceptance-tests.md` together. No `00-initial-prompt.md` exists in this workflow folder; requirements sections 1 and 3 preserve the initial brief and confirmed decisions. Requirements sections 8, 12 and 13 define acceptance scope; SIM-01–SIM-04 supersede historical alternatives. The source audit in section 24 supports planning but is neither fresh live-service evidence nor an additional acceptance contract.

This verdict approves coding planning. It does not authorize implementation, a dependency change, live messaging/reporting, a migration or a production action. High-risk implementation approval remains explicit. Tests and live checks are proposed, not executed.

## Findings

All findings below are non-blocking notes for coding planning and executable test development. They do not require rewriting the reviewed documents before planning.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests | Consent coverage asserts the shared history/settings, hosting and reporting disclosures, but does not explicitly check the prohibited privacy/portability claims stated in BR-003. The requirement is clear and has linked coverage; its negative assertions could be more explicit. | `01-requirements.md` §12 BR-003; `02-acceptance-tests.md` AT-002, AT-010 | In the coding plan's consent/copy tests, explicitly check that reviewed enablement/help copy does not promise end-to-end encryption, PDS message portability or a CraftSky-only inbox. Check reporting copy names Bluesky as the destination. Keep these assertions tied to BR-003 and AC-002/AC-010 rather than inventing new requirements. |
| DR-002 | Suggestion | Tests | Most acceptance scenarios share proposed umbrella Flutter/Go file targets, including scenarios with mobile/manual portions. This is sufficient for test design but does not yet assign each assertion to an executable suite; AT-023's generic target in particular cannot establish real OS push or web usability. The strategy and manual sections already disclose these limits correctly. | `02-acceptance-tests.md` §1, AT-001–AT-029, especially AT-023; IT-013, IT-017; MAN-001–MAN-004; GAP-001, GAP-003 | Allocate each scenario's assertions to the actual Go, Flutter widget, mobile journey, browser/platform or controlled live check in the coding plan. Use the lower-level cases as the starting point; do not duplicate every scenario in both umbrella files or report host tests as device/live proof. Preserve stable test IDs. |
| DR-003 | Suggestion | Risk | Safe retention and recovery representations remain intentionally undesigned. A planner must make the relationship between minimal retained metadata, same-operation send fencing, request lifetimes and recovery identities concrete before persistence tests can assert cleanup correctly. Unknown-event decoding also needs envelope/revision handling that does not become a raw private-payload archive. This is an acknowledged design task, not unresolved product intent. | `01-requirements.md` FR-006, FR-012, FR-013, NFR-002; `02-acceptance-tests.md` IT-006, IT-009, IT-010, IT-012, IT-018; GAP-004 | Specify persisted fields, transaction boundaries, operation/recovery identities, cleanup rules and local draft isolation in the coding plan. Show how disconnect/re-enable preserves required deduplication and unresolved-send safety, how unknown events advance safely without archiving payloads, and how deletion removes CraftSky state. Exercise these rules with the existing planned crash/cleanup tests. |
| DR-004 | Suggestion | Risk | Live validation is correctly retained within full implementation, but report submission, restricted-account eligibility and service-dependent cursor/leave/request transitions may need fixtures or service cooperation unavailable to deterministic suites. MAN-001 explicitly leaves unexercised cases unverified; this evidence boundary must survive implementation reporting. | `01-requirements.md` NFR-007, AC-024, ASM-006; `02-acceptance-tests.md` MAN-001, GAP-002 | Include the controlled live evidence checklist and required test-account/environment prerequisites in the coding plan. Record material contract differences and resolve them against requirements; do not treat mocks as proof, silently expand authority, drop controls or add a separate spike/pre-design gate. Leave unavailable checks explicitly open until genuine evidence is obtained. |

## Traceability Review

- Planning to requirements: the embedded brief and confirmed decisions align with the selected optional Bluesky-backed one-to-one inbox. The explicit decisions for shared upstream state, non-CraftSky peers, server-held OAuth, Bluesky-only reports, local drafts and selected `flutter_chat_ui` remain intact. The missing standalone initial-prompt file does not conceal an unresolved decision because the brief and decisions are embedded in the requirements artifact.
- Scope precedence: SIM-01 preserves native accepted/request unread aggregates and 99+ caps; SIM-02 preserves full integration with four-second foreground polling and removes latency/spike gates; SIM-03 permits only the first supported link with no later fallback; SIM-04 prevents provenance sections from adding acceptance obligations. The test document follows all four.
- Requirements to acceptance criteria: 3 business requirements, 19 functional requirements, 5 business rules and 8 non-functional requirements are all Must priority and have acceptance-criteria links. A read-only structural check compared every requirement's criterion set with the coverage matrix and found exact agreement for all 35 rows.
- Acceptance criteria to tests: AC-001–AC-029 each has the corresponding AT-001–AT-029 scenario. Scenario requirement references match the criterion links in the requirements table. All 10 unit, 20 integration, 6 regression and 4 manual cases include requirement and acceptance-criteria references. Test IDs, data needs and automation boundaries are explicit.
- Supporting risks and edge cases: ambiguous sends, per-device versus account revocation, current dispatch checks, unsupported/group events, cursor recovery, public declaration merging, first-link failures and stale-owner responses are carried into concrete test paths. No endpoint-count constraint or live-proven behavior was inferred from the reference audit.

## Coverage Review

- Must requirements covered: 35 of 35 have planned test coverage. All 29 acceptance criteria have explicit scenarios. Coverage includes happy paths and failure/race/recovery paths; this is design coverage, not passing executable evidence.
- High-risk coverage:
  - Optional grant semantics, normal login/refresh and multi-session selection: UT-001, IT-005, IT-015, REG-001.
  - Stable send operations and ambiguous acceptance across crashes: AT-007, IT-006, including full transport retry-stack verification.
  - Correct direct/request pagination, safe history and native unread/read-watermark semantics: IT-001, IT-002, IT-007, UT-003, UT-004.
  - Crash-consistent cursor/outbox progress and durable dedupe: IT-009, IT-010; once-per-request lifetime and bounded catch-up: IT-012.
  - Account-safe navigation and stale work/private UI fencing: IT-005, IT-013, IT-017; dispatch-time read/mute/block/leave/preference/subscription checks: IT-011, IT-014.
  - Authorized Bluesky reporting and conditional shared policy updates: IT-004, IT-008, with live complements in MAN-001.
  - Serialized privacy canaries and positive diagnostic controls, minimal retention and deletion boundaries: IT-018, REG-005, REG-006.
  - First-supported-link previews, no fallback/composer/request fetches and owner-switch races: UT-008, IT-016.
  - Local drafts and no automatic offline replay: UT-009, IT-017.
  - Shared four-second foreground polling, bounded faults and healthy-account progress: UT-006, IT-020. No message-arrival benchmark is required.
- Missing or weak coverage: no missing Must coverage. DR-001 identifies a narrow copy-assertion improvement, and DR-002 identifies the remaining suite-allocation task. GAP-001 correctly states that implementation and test execution have not started.
- Manual-only coverage: actual external OAuth/PDS/chat/moderation behavior (MAN-001), APNs/FCM plus real OS notification taps (MAN-002), assistive-technology and actual platform interaction (MAN-003), and package planning/platform fit review (MAN-004). Their external/device/review boundaries justify manual complements; business rules and fault/crash paths remain automated.
- Test-level suitability: pure mapping/validation/policy functions use unit tests; persistence, adapter/API boundaries and lifecycle races use integration tests; user journeys use acceptance scenarios with lower-level automation. REG-001–REG-006 protect existing auth, social notification, routing, link, diagnostic and deletion behavior. UT-003/UT-010 contain controller/widget assertions and should use the appropriate Flutter test runner rather than be interpreted as pure Dart-only tests.
- Test data and commands: 11 explicit data sets cover identities, Unicode boundaries, counts, declarations, protected-value canaries, durable failures, request lifetime, permission/route tampering, preview races and local drafts. Discovered recipes distinguish unit-only, real persistence, backend release gates, host Flutter and mobile journey evidence. The mobile harness's loopback/injection and web limitations are disclosed. No commands were executed to validate implementation in this review.

## Risk And Approval Review

- Risk level: High; unchanged. Private content, optional OAuth authority, shared settings, non-idempotent upstream writes and background delivery justify this classification.
- Review requirement: the requested document review is complete. No high-risk source change or live action was performed. Explicit owner approval remains required before implementation begins, as recorded in the test-design artifact and its originating skill.
- Approval notes: the owner selected “Move to document review.” That authorizes this review artifact, not implementation. A later selection of coding planning authorizes the next design stage. This verdict approves the document set for that stage and must not be interpreted as owner authorization to write feature code.
- External assumptions: ASM-006 and service-dependent behavior remain integration evidence to obtain during full implementation, not blocking product questions or prerequisites to coding planning. The live acceptance obligation remains; absence of a separate spike does not waive it.
- Verification boundaries: at-most-one logical job/subscription is distinct from exactly-once provider delivery. Lifecycle checks cannot retract already-issued network requests. Source/solver/mocked results are distinct from actual service, platform and device results. Both documents preserve those boundaries.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: design optional capability evaluation and owner/session fencing, beginning with the proposed UT-001/IT-015 failure case: base-only login/refresh succeeds with optional messaging advertised, while a chat operation without its required effective authority is denied.
- Blocking issues: none for coding planning. DR-001–DR-004 are non-blocking notes. Implementation approval and actual integration/platform evidence remain future gates appropriate to their stages.
- First-failing-test order: capability/owner boundaries → text/report validation and fixed adapters/API contracts → durable send uncertainty → inbox/history/request/read/actions and conditional declaration merge → atomic sync and notification policy → dispatch/privacy/lifecycle → Flutter adapters/drafts/navigation/previews → bounded scheduling/regression → controlled live/platform verification.

## Notes For Next Stage

- Produce `04-coding-plan.md` from the two reviewed documents and this review; do not redesign confirmed product choices. Carry forward DR-001–DR-004 and GAP-001–GAP-007 with concrete implementation/test tasks.
- Define final routes and schemas, finite scheduling/resource budgets, durable operation/recovery identities, minimal retention/cleanup and private local draft storage. These are design choices constrained by the existing requirements, not reasons to request new product decisions by default.
- Inspect actual implementation seams before finalizing file paths. Proposed test files are targets, not existing suites. Reuse session coordination, owner fencing, access checks, durable delivery and notification-routing patterns without inheriting the social profile-mute policy for DMs.
- Preserve upstream native count semantics, explicit request acceptance, separate chat/report audiences, conditional declaration updates and first-supported-link-only previews. Keep `flutter_chat_ui` selected and record concrete package compatibility evidence and any blockers.
- Consult the repository architecture reference before architectural decisions and the API architecture before route design. No local Lexicon changes are proposed; any later decision to change `lexicon/` must follow the repository's skill/ADR requirements.
- Keep functional live checks in the full integration plan. Do not introduce a separate spike, pre-design service gate, latency target or performance benchmark.
- Earlier documents were reviewed without modification. This stage creates only `03-document-review.md`; no source/tests/dependencies/migrations/generated files changed, no commit or push, and no production/live action occurred.
