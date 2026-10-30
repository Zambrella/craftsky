# Implementation Review: Edit Published Posts

## Verdict

Status: Changes required

Reviewer: Codex
Date: 2026-10-27
Risk level: Medium

## Summary

The code corrections are accepted by this review: IR-001/IR-002 and IR-004–IR-011 are resolved, with no new code defect identified. The five approved simplifications remain consistent with the current requirements and coding plan. The IR-011 change is narrow: fresh same-CID quote observations advance ordering evidence, and partial-to-full handoff retains the maximum known revision for identical content. Cached rendering cannot promote evidence, and no complete post is fabricated from partial quote content.

The verdict remains Changes required solely because required manual platform validation IR-003 / MAN-001 / MAN-002 is still pending. This is a validation gap, not a request for another code-refactoring loop. The maintainer has already committed to supplying results; no repeat commitment or waiver is requested.

Reviewed the current requirements, acceptance specification, historical document review, coding-plan refinements, implementation record and current source/diff, with emphasis on the client-only IR-011 correction. Only this review artifact was edited. No source, repository tests, dependencies, migrations, generated files or other workflow documents were changed during review; no commit, push, deployment or real PDS mutation was performed.

## Findings

No open code findings identified.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-003 | Important | Tests | **Required platform validation remains pending.** Automated semantics/input and loopback tests do not establish real VoiceOver/TalkBack, browser screen-reader/keyboard/pointer or native-picker behavior. | FR-003/FR-005/FR-007/NFR-001; AC-003/AC-007/AC-009/AC-024; MAN-001/MAN-002, GAP-003. | Record and assess the maintainer's promised results when supplied. Keep pending; do not ask for the same commitment again. Optional live federation MAN-003 remains separate. |

### Previous finding status

| ID | Status | Evidence reviewed |
|---|---|---|
| IR-001 | Resolved | Absent/empty language prefill, assignment, clearing and reversion remain covered by passing editor regressions. |
| IR-002 | Resolved | Existing owner-seed/PDS diagnostics preserve typed causes, stacks and safe context; obsolete accepted-refresh paths were deliberately removed. |
| IR-004 | Resolved | Pending project-material input navigation guards remain in passing published-editor tests. |
| IR-005 | Resolved for its full-post interleaving | Full-post observations advance retained presentation; matching detail cannot release a lagging timeline. |
| IR-006 | Resolved | Loaded page-two items, continuation requests, list metadata and cursors remain protected across edit reprojection. |
| IR-007 | Resolved | Immediate create → edit preserves previously established publication-list membership without unrelated insertion or changed publication time. |
| IR-008 | Resolved | Dispatch independently checks authoritative, non-deleted, structurally/semantically valid current source. Permanent malformed/semantic-invalid/uncertain/deleted/missing cases coexist with positive removed-original-mention delivery. |
| IR-009 | Original reproduction resolved | A fresh different-CID quote V3 now advances retained partial preview state, updates loaded quotes and resists delayed V1; a full V4 supersedes the partial preview. No full record is fabricated. The additional same-CID boundary is now resolved under IR-011. |
| IR-010 | Resolved | Retained preview presentation accepts raw URI text without requiring HTTP fetch eligibility or successful generic URI parsing. Caption edits preserve the exact retained embed, including unknown fields; new unsupported preview URIs remain rejected. Client tests cover retained AT URI/no exception/no fetch and caption changes. |
| IR-003 | Open | Promised manual platform results have not been recorded. |
| IR-011 | Resolved | Fresh same-CID quotes advance ordering evidence. Older/missing quote and matching full-post revisions preserve the maximum known bound; permanent provider regression passes. Fresh review-only cached/fresh/later-full interleavings also pass. |

IDs remain stable. No approved product tradeoff is reopened.

## Requirement And Test Traceability

- **BR-001; FR-001–FR-005; RULE-001–RULE-003/RULE-007:** authoritative owner prefill without Tap, corresponding composers, fixed identity/type/context, exact-CID writes, explicit clears, retained media and raw-field preservation remain represented. Shared submission preserves frozen retries, account fences and video recovery.
- **BR-002; FR-006/FR-007/FR-012:** current-record editedAt, one localized Edited tooltip, external missing/future declarations, strict malformed rejection and no-op/retry stability follow the revised contract. Removed provenance/history storage is not required.
- **FR-008/FR-011/FR-012; AC-011/AC-014/AC-015; AT-004/UT-007/IT-011:** latest-by-URI hydration, confirmed immediate dismissal and loaded-list overlays remain. IR-011 now covers same-CID restoration, stale intermediate reads and older/revisionless matching quote/full responses while preserving loaded presentations. Fresh quote evidence remains partial until a complete record arrives.
- **FR-009/FR-010; NFR-002/NFR-003:** strict conflicts, immutable keyed recovery, session/lifecycle controls, confirmation-only responses and bounded diagnostics remain represented. No new PDS capability or client credential access is added.
- **RULE-004/RULE-005:** fixed original recipients remain distinct from authoritative current-source validation, deletion and existing block/moderation/delivery controls. IR-008's positive and negative dispatch cases remain covered.
- **RULE-006:** no edit-history UI/API or feature body archive. Original publication facts and ordinary operational/moderation retention retain their existing roles. IR-011 adds no storage or migration.
- **NFR-001:** automated semantics, localization, focus/touch and prior native loopback evidence are documented. Required actual platform checks remain pending under IR-003.
- **Unplanned behavior:** none identified. The implementation plan accurately records the two IR-011 red-green loops and client-only scope. Earlier design sections superseded by approved sections 13–15 are treated as historical, not contradictory new requirements.

## Test Evidence

| Check | Result and scope |
|---|---|
| Fresh focused review matrix | **48 passed**, exit 0: `fvm flutter test test/feed/providers/edit_post_provider_test.dart test/feed/providers/post_edit_pagination_test.dart test/shared/mutations`. Includes the permanent extended IR-011 regression. Log `/tmp/edit-ir011-review-regressions.log`. |
| Fresh independent review-only ordering check | **1 passed**, exit 0: `fvm flutter test /tmp/edit-ir011-review-ordering_test.dart --plain-name 'IR-011 newer quoted originals'`. Adds a cached same-CID future revision, a genuine fresh intermediate revision, a later complete record and a delayed older quote to the production-provider fixture. The cache cannot suppress the genuine update; the complete record supersedes the partial preview. Log `/tmp/edit-ir011-review-ordering2.log`. |
| Review-only fixture correction | The first independent probe used invalid TID digits 8/9 and failed to accept the supposed newer version. The rerun uses valid ordered TIDs ending in a/b/c. `/tmp/edit-ir011-review-ordering.log` is a fixture failure, not a product regression. Repository source/tests were unchanged. |
| Latest complete client implementation evidence inspected | **3,467 passed, 38 skipped**, `/tmp/edit-ir011-full-flutter.log`; analysis has no issues, `/tmp/edit-ir011-analysis.log`. Completed after the final source correction. Not relabeled as fresh complete review runs. |
| Previous broader review evidence inspected | **127 Flutter tests passed**, `/tmp/edit-corrections-review-flutter.log`; selected required-PostgreSQL race tests passed API/index/push/PDS-command packages, `/tmp/edit-corrections-review-backend.log`. These predate the client-only IR-011 fix and are not fresh runs for this review. |
| Authoritative backend gate inspected | `/tmp/edit-corrections-appview-check.log` ends with all release gates passed; artifacts `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.Ynt7bKtobW`. Normal/race, module/generation/static/build/dependency/migration/startup and pinned Tap handshake checks passed. Backend source is unchanged by IR-011, so no additional backend gate was run for this review. No deterministic Tap event injection is claimed. |
| Native/manual evidence boundary | Implementation record documents prior iOS simulator/Android emulator IT-015 loopback passes. They are not real VoiceOver/TalkBack, native-picker, browser or live federation proof. MAN-001/MAN-002 remain pending; MAN-003 remains optional and unrun. |
| Working-tree checks | `git diff --check` passes. Only the review artifact was edited by this review; no implementation change or production action. |

## Risk Review

Risk level: Medium. No open code defect or new high-risk scope was identified. Residual acceptance risk is actual assistive-technology, browser interaction and native media-picker behavior, which automated host tests cannot certify.

IR-011 now preserves ordered evidence across matching CIDs and partial/full handoff without extending API, storage, write authority or history retention. Existing safety/account/cache boundaries and loaded-list behavior remain. Keep the five simplifications. Do not reopen completed code work merely because manual evidence is pending.

Approval note: code corrections are acceptable; final feature acceptance remains blocked only by IR-003 until the promised results are recorded and assessed. A failing manual result would require a targeted finding and correction at that time, not speculative changes now.

## UI Polish Recommendation

Recommendation: Not needed.

The reviewed correction changes data reconciliation, not UI layout or copy. Existing composer/preview/indicator controls are coherent in automated checks. Manual accessibility validation remains a required test gap rather than optional polish.

## Handoff Back To TDD Builder

- Required code fixes: none identified. Do not start another code correction loop for IR-011.
- Remaining required action: record and assess promised MAN-001/MAN-002 results under IR-003 when available. Keep pending without repeating the request or marking them passed.
- Suggested next failing test: none unless manual results reveal a concrete defect.
- Verification to rerun: tests appropriate to any subsequent actual change; no additional full backend/client rerun is required for this documentation-only review.
- No stage commit, push or production action was requested. This artifact remains uncommitted.
