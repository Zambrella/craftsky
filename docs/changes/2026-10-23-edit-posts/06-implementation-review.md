# Implementation Review: Edit Published Posts

## Verdict

Status: Changes required

Reviewer: Codex, direct source, test and evidence review

Date: 2026-10-27

Risk level: Medium

## Summary

The two code findings from the initial review are resolved. Published editing now represents absent/empty languages faithfully, permits explicit assignment/clearing and preserves untouched/reverted values. Editor-seed identity failures retain their original cause at the existing safe diagnostic owner. No new code defect was identified in this follow-up.

The overall verdict remains **Changes required solely for IR-003: required manual validation evidence is pending**. The maintainer has said they will provide MAN-001/MAN-002 results; that is not a waiver or evidence of passing. No further source correction is requested at this checkpoint.

This follow-up inspected the current correction source and tests against FR-002/FR-003/FR-005, NFR-003, the acceptance specification, approved coding plan and updated implementation record. It also read the fresh correction-run logs. The earlier broader architectural review remains the basis for unchanged feature paths; this is not a claim that every line was independently re-reviewed or every gate rerun. Only this review artifact changed; no source edits, tests, commits, pushes, deployments or live actions were performed during this review.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-003 | Important | Tests | **Complete the explicitly required platform validation before final acceptance.** MAN-001 and MAN-002 remain unexecuted. Widget semantics/input tests and native loopback journeys are useful evidence but do not establish actual VoiceOver/TalkBack, browser screen-reader/keyboard/pointer behavior or real picker operation. The implementation record discloses this correctly; it does not close AC-007/AC-024's platform evidence. | FR-007, NFR-001; AC-003, AC-007, AC-024; `02-acceptance-tests.md` section 8, MAN-001/MAN-002 and GAP-003; `05-implementation-plan.md` completion and manual limits. | Execute and record MAN-001/MAN-002 against local/test data, including supported web interaction and real native picker/assistive checks. Record platform, steps and outcomes; route any behavioral failure back through TDD. The maintainer will provide the results; record those results before closing this finding. No explicit waiver has been given. MAN-003 remains optional and requires its separate live authorization. |

### Resolved findings

| ID | Resolution reviewed | Evidence | Status |
|---|---|---|---|
| IR-001 | `PostLanguageSelection.forPublishedEdit` permits zero tags while ordinary factories retain their one-to-three constraint. Both composers seed the exact existing language list and baseline. Adding/removing retains the edit mode; metadata controls permit removing the final language only in that mode. Unchanged or reverted empty/absent input emits no patch, preserving its original server representation. | 14 actual-composer cases in `app/test/feed/widgets/published_post_editor_test.dart`: both composers × absent/empty prefill/no-op, preferred-English assignment/reversion, and final-language clearing. Backend `TestEditPostPresenceChanges` already covers empty/removal. All 69 focused language/editor regressions and the full Flutter suite pass. | Resolved; FR-002/FR-003/FR-005, AC-003/AC-009, AT-002/UT-002. |
| IR-002 | `GetPostEditHandler` now calls `ReportRequestFailure` with the original identity error before returning the unchanged safe 502. It uses the existing request owner rather than a new reporting mechanism. | `TestEditPostSeedIdentityFailureDiagnostics` verifies one original-cause issue, `net.DNSError` chain, available original stack, request/operation/stage context and private payload/credential/DNS/session canary exclusion in serialized sinks. Focused and complete API/observability/route race suites pass. | Resolved; NFR-003, AC-026, IT-017. |

Review history: the first review on 2026-10-27 identified IR-001, IR-002 and IR-003. The maintainer authorized corrections; this follow-up closes only the first two. IR-003 remains open pending human results. Existing finding IDs are retained.

## Requirement And Test Traceability

| Requirement group | Implementation and evidence reviewed | Assessment |
|---|---|---|
| BR-001; FR-001, FR-004; RULE-001, RULE-007 | Shared PostCard menu, per-opening owner seed, matching composers, conditional addressed command; owner/surface, draft-isolation, conflict/deletion and native loopback tests | Main behavior covered; no feature age/count limit or create-on-edit path identified. |
| FR-002, FR-003, FR-005 | Typed editable projection; server raw-field merges; retained indices/media; general/project hydration, variant/clear/embed/video tests | Broad coverage; IR-001 language-state correction and its regression cases now pass review. |
| BR-002; FR-006, FR-007, FR-012 | Optional datetime schema and generated types; current digest/CID/revision state; confirmed/reported/observed metadata; shared localized indicator; source/project/search projection | Strict malformed timestamp rejection now agrees with the maintainer's decision. Absent/future schema-valid fallback and locale/DST tests are green. Actual platform access remains IR-003. |
| FR-008, FR-011; RULE-003 | Current-by-URI quote reads, stable chronology, accepted-version hydration, edit overlays that replace existing rows without inserting publications; exact-CID/later-revision reconciliation | No additional correctness defect identified in reviewed paths. Stored strong references and quote commentary remain unchanged. |
| FR-009, FR-010; RULE-002; NFR-002 | Exact expected CID before build/no-op, immutable dispatch-body recovery, stable operation time/key, owner/session fencing, pending read-only refresh, client account-change guards | Supported by command, API, provider/widget and loopback tests. No new credential handoff identified. |
| RULE-004, RULE-005, RULE-006 | Original-publication facts before coalescing; monotonically shrinking applicability; insert-once activation; final dispatch fence; current safety/visibility and terminal cleanup inventory | No new history archive identified. New persistence contains current metadata, bounded eligibility facts and evidence within the existing command retention mechanism. Notification/safety race evidence reviewed. |
| NFR-001, NFR-003 | Reused localized forms, tooltip semantics and interaction tests; request/provider diagnostic owners and serialized canary tests | IR-002 seed identity diagnostic gap is resolved; IR-003 records remaining real-platform evidence. |

- Requirements implemented: all 24 Must requirements have corresponding code and planned evidence; the findings prevent declaring every acceptance criterion complete.
- Tests implemented: AT-001–AT-008, UT-001–UT-009, IT-001–IT-017 and REG-001–REG-008 are mapped in `05-implementation-plan.md` to concrete automated suites and recorded execution. IDs describe scenario groups, not one test method each. The correction pass adds the previously missing language-less editor cases and seed identity diagnostic case within those groups.
- Unplanned behavior: no unrelated product feature, general/project conversion, history interface, additional OAuth authority or production action identified. Helper placement differs from proposed filenames in the coding plan without changing the architecture. Current-main draft fixes were retained rather than reimplemented.
- Document changes: the approved strict malformed-datetime clarification is reflected in requirements section 12, acceptance matrix, coding plan and ADR 022. `03-document-review.md` is correctly retained as a historical review, not represented as a fresh review of later revisions.
- Remaining gaps: IR-003 only. No outstanding source correction identified. Optional live federation verification is explicitly separate; it is not a new prerequisite imposed by this review.

## Test Evidence

Commands/results reviewed from the implementation and correction logs, **not rerun in this review**:

| Scope | Recorded result / evidence |
|---|---|
| Authoritative backend gate, `./scripts/appview-check` | Passed; `/tmp/craftsky-edit-appview-gate.log`, artifacts `/tmp/craftsky-edit-appview-gate`. Includes normal/race suites, generation/schema checks, static checks, migration rehearsal and builds. Predates the final timestamp test clarification; backend production source did not change in that clarification. Pinned Tap handshake passed; deterministic event injection was unavailable and was not claimed. |
| Final editing backend regression matrix | `GOTOOLCHAIN=go1.27.3 go test -p 1 -race ./internal/api ./internal/index ./internal/ingestion ./internal/pdscommands ./internal/postrecord ./internal/posteditstate ./internal/notifications ./internal/push -run 'PostEdit\|EditPost\|ObserveEdit\|PostPublicationActivation' -count=1`, with the recorded required local database environment. Passed; `/tmp/craftsky-edit-final-backend-regressions.log`. Includes malformed timestamp rejection and valid observed fallback. `posteditstate` compiled but has no standalone tests; its correlation behavior is exercised through integration paths. |
| Complete correction-pass Flutter suite, `fvm exec just app-test` | **3,408 passed, 38 existing skips**; `/tmp/craftsky-edit-ir-full-flutter.log`. Includes all 14 new language cases and the prior locale/DST additions. Replaces the earlier 3,392-test run as the latest complete client evidence. |
| Correction-pass Flutter analysis, `fvm flutter analyze` from `app/` | No issues; `/tmp/craftsky-edit-ir-analysis.log`. |
| Correction-pass complete backend API/observability/route suites | `GOTOOLCHAIN=go1.27.3 go test -p 1 -race ./internal/api ./internal/observability ./internal/routes -count=1` with recorded required local database/media environment: passed, API 274.657s, observability 14.627s, routes 33.152s; `/tmp/craftsky-edit-ir-backend-regressions.log`. Includes the new seed diagnostic test. |
| Focused correction evidence | 69 editor/metadata/all-language tests passed (`/tmp/craftsky-edit-ir001-regressions.log`); seed identity diagnostic test passed under race (`/tmp/craftsky-edit-ir002-green.log`). Recorded meaningful red results precede the corrections. |
| Controlled locale/DST indicator suite | `TZ=Europe/London fvm flutter test test/feed/widgets/post_edit_indicator_test.dart`: 4 passed; `/tmp/craftsky-edit-indicator-locale-dst-check.log`. |
| Complete iOS/Android loopback critical journeys | 5 passed on each simulator; `/tmp/craftsky-edit-ios-loopback-full.log`, `/tmp/craftsky-edit-android-loopback-full.log`. Synthetic services/auth; no live PDS or picker/screen-reader proof. |
| Additional focused contract evidence | Real-socket handler→command→PDS fixture→projection, current surface callers, bounded video recovery, unrelated drafts and serialized diagnostics are recorded individually in `05-implementation-plan.md`; their tests were inspected where relevant to the review findings. |

- No intentionally failing automated test remains after the strict timestamp decision. Earlier red/setup-failure entries are historical and are not passing evidence.
- MAN-001 and MAN-002 have not run. MAN-003 has not run and remains optional/unauthorized. Remote CI, production and live third-party behavior are not established by local gates.
- IR-001/IR-002 now have focused regression coverage and passing broader evidence. No further automated rerun was needed for this documentation-only review. The full AppView release gate and native loopback results predate the correction pass and are retained as historical evidence; they are not claimed as fresh correction runs.
- IR-003 remains pending. The browser fixture compiled, but preview automation timed out and produced no usable interaction evidence; device interaction was disabled. These attempts are not manual test passes. The maintainer will provide the actual results, as recorded in `05-implementation-plan.md`.

## Risk Review

- Risk level: Medium. The feature changes public record updates, shared composers, current projection, notification eligibility and migration/cleanup paths.
- Conditional PDS writes and read-only uncertain-outcome recovery preserve the existing authority boundary. A response-only eligible refresh avoids persisting replayable post content in the new terminal response.
- The optional Lexicon field has an Accepted ADR and generated outputs. Strict validation remains intact; malformed external versions are hidden rather than being accepted through a fallback.
- New tables do not implement edit history. Existing command plans/recovery and moderation retention still need their normal lifecycle; this review does not recommend deleting them.
- IR-001/IR-002 corrections remain within existing editor and diagnostic boundaries. No new schema, migration, dependency, permission or PDS-command behavior was introduced by the correction pass.
- Remaining risk is unverified real browser/assistive/native-picker operation under IR-003. Automated semantics and simulator/fixture evidence do not close that risk.
- Approval notes: no additional source correction is requested. Required manual evidence must be supplied and reviewed before full acceptance. This review does not authorize commits, pushes, deployment, production migration or live service writes.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: No independent polish-only change is required by the reviewed implementation. Existing forms, localized feedback and shared edit disclosure provide a coherent UI contract at the source/widget level. No fresh visual/browser inspection was performed, so this is not a visual sign-off.
- Suggested polish notes: none. Real accessibility operability remains a validation issue, not optional polish. The language-state correctness finding has been resolved.

## Handoff Back To TDD Builder

- Required source fixes: none identified in this follow-up. IR-001 and IR-002 are resolved.
- Required validation: IR-003. Await the maintainer's MAN-001/MAN-002 results, including platform/browser, actual steps and outcomes for assistive access, indicator interaction, real picker retain/replace/alt/cancel, and web edit/save/conflict/keyboard operation. Do not treat the promise to provide results as a passing result or waiver.
- Suggested next failing test: none unless manual validation exposes a behavioral defect. Translate any such defect into the smallest appropriate regression test and correction loop under the existing requirements.
- Verification to rerun: no source changed in this review, so no redundant automated rerun is requested. If manual results expose a code defect, rerun its focused regression and relevant broader checks after correction. MAN-003 remains optional and separately authorized.
- Next review action: record and assess the promised manual results, then update IR-003 and the overall verdict. Do not start another code correction loop solely because this verdict still says Changes required for missing human evidence.
