# Implementation Review: Edit Published Posts

## Verdict

Status: Changes required

Reviewer: Codex, direct source, regression and reproduction review

Date: 2026-10-27

Reviewed commit: `947b7f24` (`feat/edit-posts`, after merging `origin/main`)

Risk level: Medium

## Summary

This is a renewed review of the merged implementation, not only a recheck of the two earlier corrections. One new required code correction was identified: **IR-004, a pending material edit can silently delete the published material when the author leaves the detail page and saves**. The exact composer interaction was reproduced in a temporary test without changing repository source or tests.

IR-001 and IR-002 remain resolved. IR-003 remains open because the required real accessibility, browser and native-picker evidence has not been supplied. The maintainer has said they will provide those results; this is neither a pass nor a waiver.

The review compared the feature diff against the approved requirements, acceptance specification, historical document review, coding plan, implementation record and ADR 022. It traced the owner seed and conditional PDS update, lossless nested merge, project/material hydration and patches, retained/replacement media, immutable retry and recovery, current edit-time projection, chronology, quote/repost hydration, client reconciliation, publication-notification restrictions and dispatch fences, diagnostics, and terminal cleanup. It also examined the merged PostCard header and material-editing interaction. No other actionable defect was identified in those reviewed paths. This is not a claim of exhaustive verification of every interleaving or supported physical platform.

Only this review artifact was changed. Source, tests, dependencies, migrations and other workflow documents remain unchanged. No commit, push, deployment or live-provider mutation was performed.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-004 | Important | Behavior / Tests | **Pending material edits can become unintended published deletions.** Open an existing project containing Wool, open common details, choose Edit material, type Merino wool, then go Back without pressing Add and select Save changes. `_edit` immediately removes the original entry from the FormBuilder value; `_syncDetailForm` copies that removal to the parent. The pending text and `_editingMaterial` exist only in the detail widget and disappear when its route closes. Saving then sends `project.common.materials: null` for the single-material case, so the existing server merge deletes the material. The merge fix preserves `existingIndex` only when Add is pressed and does not cover this exit path. | FR-002–FR-005, FR-010; AC-003, AC-009, AC-013; AT-002, UT-001/UT-002, REG-001–REG-003. `app/lib/projects/widgets/project_materials_field.dart:179–187`; `app/lib/projects/widgets/project_composer_sheet.dart:1279–1288`, `1372–1394`, `2180–2224`; temporary failing reproduction `/tmp/craftsky_edit_review_probe_test.dart`, output `/tmp/craftsky-edit-review-material-probe.log`. | Make material editing preserve the committed entry until an explicit commit/removal, or prevent leaving/submitting with a pending edit while preserving recoverable input. Back followed by Save must not silently publish a removal. Add a permanent actual-composer regression for this sequence, plus unchanged-edit exit, explicit Add/Remove and cancellation cases. Retain source-entry identity/unknown fields when committing. Check shared create/draft/scheduled material flows because the field is reused. |
| IR-003 | Important | Tests | **Required platform validation remains pending.** Widget semantics/input tests and native loopback journeys do not establish actual VoiceOver/TalkBack, browser screen-reader/keyboard/pointer behavior or real native picker operation. | FR-007, NFR-001; AC-003, AC-007, AC-024; MAN-001/MAN-002 and GAP-003 in `02-acceptance-tests.md`; manual limits in `05-implementation-plan.md`. | Record and assess the MAN-001/MAN-002 results promised by the maintainer, including platforms, steps and outcomes. Route actual behavioral failures through TDD. No waiver has been given. MAN-003 remains optional and requires separate live authorization. |

### Resolved findings

| ID | Resolution reviewed | Evidence | Status |
|---|---|---|---|
| IR-001 | Published editing faithfully seeds absent/empty languages and allows removing the final language. Ordinary creation still retains its existing language constraint. Unchanged or reverted empty/absent input emits no patch. | `PostLanguageSelection.forPublishedEdit`, both composer baselines and optional selector controls; 14 actual-composer language scenarios. These cases passed again in this review's focused Flutter run. | Resolved; FR-002/FR-003/FR-005, AC-003/AC-009, AT-002/UT-002. |
| IR-002 | Owner-seed identity failures call the existing `ReportRequestFailure` owner with the original cause before returning the safe 502. | `TestEditPostSeedIdentityFailureDiagnostics` checks original typed cause/stack, one issue, correlation and serialized private-value exclusion. Passed again in this review's backend edit tests. | Resolved; NFR-003, AC-026, IT-017. |

Review history: the initial review identified IR-001–IR-003; the correction follow-up closed IR-001/IR-002. This renewed review after the main merge adds IR-004 and retains IR-003. Finding IDs are stable.

## Requirement And Test Traceability

| Requirement group | Implementation and evidence reviewed | Assessment |
|---|---|---|
| BR-001; FR-001, FR-004; RULE-001, RULE-007 | Shared owner menu, fresh per-opening seed, corresponding composers, exact-base addressed update; owner/surface, draft-isolation, conflict/deletion suites | Stable URI and owner-only editing are represented; no edit age/count limit or create-on-edit path identified. |
| FR-002, FR-003, FR-005 | Generated supported projection; raw server merges; retained entry indices/media; project variant, clear, embed and video paths | IR-001 remains resolved. **IR-004 exposes a missing pending-material exit case despite passing existing preservation tests.** |
| BR-002; FR-006, FR-007, FR-012 | Optional datetime and generated JSON/CBOR schema types; current digest/CID/revision state; confirmed/reported/observed indicator and localized semantics | Additive schema agrees with ADR 022 and the approved strict-validation clarification. Source/serving metadata association and no-op behavior covered. Actual platform access remains IR-003. |
| FR-008, FR-011; RULE-003 | Current-by-URI quote reads, stable chronology, accepted-version hydration, replacement-only edit overlays and exact-CID/later-revision reconciliation | No additional defect identified. Stored reply/quote strong references remain fixed; overlays do not insert edited posts as new publications. |
| FR-009, FR-010; RULE-002; NFR-002 | Strict expected CID before build/no-op; stored immutable dispatch recovery; stable operation time/key; owner/session gates; read-only accepted refresh | Command/API/provider tests cover stale bases, uncertain outcomes, deletion and account changes. No new OAuth handoff identified. Pending material text is the distinct recovery gap in IR-004. |
| RULE-004, RULE-005, RULE-006 | Original-publication facts before coalescing, shrinking eligibility, insert-once activation, final dispatch fence, safety/visibility and terminal inventory | Targeted race regressions passed. No new feature history archive identified; existing bounded command/moderation evidence remains under its existing retention policy. |
| NFR-001, NFR-003 | Reused forms and localized disclosure/error feedback; existing diagnostic owners and serialized canary tests | No new diagnostic finding. Required manual evidence remains open. |

- Automated scenario groups AT-001–AT-008, UT-001–UT-009, IT-001–IT-017 and REG-001–REG-008 are mapped to implementation evidence in `05-implementation-plan.md`. Those mappings do not imply exhaustive edge-case coverage: IR-004 needs a permanent regression within the project-edit groups.
- Merge resolutions preserve main's avatar/profile hero wrapper alongside the edit indicator and preserve material identity after explicit Add. The latter is incomplete for leaving a pending material edit.
- No unrelated product scope, general/project conversion, history UI, broader credential authority or production operation was identified.
- Requirements/test-design status prose and `03-document-review.md` retain historical stage context; implementation authorization comes from the recorded maintainer approvals. The strict malformed-datetime decision is aligned across requirements, acceptance tests, coding plan and ADR 022.
- Remaining gaps: IR-004 code correction/regression and IR-003 human evidence. Optional live federation validation remains separate.

## Test Evidence

### Fresh execution in this review

| Scope | Result / evidence |
|---|---|
| Existing focused Flutter editor/provider/project-material tests | **80 passed**, exit 0. From `app/`: `fvm flutter test test/feed/widgets/published_post_editor_test.dart test/feed/widgets/post_edit_flow_test.dart test/feed/widgets/published_post_external_edit_test.dart test/feed/providers/edit_post_provider_test.dart test/feed/composer/post_edit_changes_test.dart test/projects/widgets/project_materials_field_test.dart`. Log: `/tmp/craftsky-edit-review-rerun-flutter.log`. |
| IR-004 temporary actual-composer reproduction | **Failed as expected**, exit 1. Existing test harness copied to `/tmp` with one review-only interaction case; repository source/tests untouched. `fvm flutter test /tmp/craftsky_edit_review_probe_test.dart --plain-name 'REVIEW pending material edit'`. Actual saved project patch was `{'common': {'materials': null}}`. Log: `/tmp/craftsky-edit-review-material-probe.log`. This confirms the outgoing patch defect; no actual PDS write was made. |
| Edit/publication backend regression matrix | **Passed**, exit 0, with required database tests enabled against this worktree's existing local PostgreSQL/MinIO test services. From `appview/`: `GOTOOLCHAIN=go1.27.3 go test -p 1 -race ./internal/api ./internal/auth ./internal/index ./internal/ingestion ./internal/pdscommands ./internal/notifications ./internal/push ./internal/ownerlifecycle ./internal/postrecord ./internal/routes -run 'Edit|Publication' -count=1`. Log: `/tmp/craftsky-edit-review-backend.log`. The auth package had no matching test names and is covered by the separate invocation below. |
| PDS metadata/session adapter contracts | **Passed**, exit 0. `GOTOOLCHAIN=go1.27.3 go test -race ./internal/auth -run 'PreservesRepositoryHeadRevision|PreservesReturnedWriteCommit|PreservesReturnedRecordIdentity|KeepsCommitEvidenceInsideSessionFence' -count=1`. Log: `/tmp/craftsky-edit-review-auth.log`. |
| Schema and edit-time unit contracts | **Passed**, exit 0. `GOTOOLCHAIN=go1.27.3 go test -race ./internal/lexicon/schema ./internal/postrecord -count=1`. Log: `/tmp/craftsky-edit-review-schema.log`. |

The first sandboxed Flutter launch could not write the SDK cache; the authorized rerun above executed and passed. It is not counted as a product test failure. The failing temporary reproduction is deliberate review evidence and remains unfixed at this stage.

### Earlier evidence reviewed, not rerun here

- Latest merged-tree complete Flutter run: **3,432 passed, 38 existing skips**, `/tmp/craftsky-edit-main-merge-full-final.log`; merged-tree analysis: **No issues**, `/tmp/craftsky-edit-main-merge-analysis-final.log`. These were run during the preceding merge task, not this review. The review confirms their logs but does not treat them as coverage of IR-004.
- Full backend `./scripts/appview-check`: passed, `/tmp/craftsky-edit-appview-gate.log`, artifacts `/tmp/craftsky-edit-appview-gate`. Includes normal/race suites, generation/schema checks, migration rehearsal and builds. Predates the final diagnostic correction; this is historical release-gate evidence, not a fresh merged-tree gate. Pinned Tap handshake passed; deterministic event injection was unavailable.
- Correction-pass complete API/observability/route race suites passed: `/tmp/craftsky-edit-ir-backend-regressions.log`. Serialized original-cause/privacy tests and language correction evidence remain recorded in the implementation plan.
- iOS and Android loopback critical journeys: 5 passed on each simulator, `/tmp/craftsky-edit-ios-loopback-full.log` and `/tmp/craftsky-edit-android-loopback-full.log`. Synthetic auth/services; no real picker, assistive technology or live PDS proof.
- MAN-001/MAN-002 remain unexecuted. Previous browser automation attempts produced no usable interaction evidence. MAN-003 remains optional and unrun. Remote CI, production deployment and live provider behavior were not verified.

## Risk Review

- Risk level: Medium overall; IR-004 is a blocking public-content data-loss risk on a normal editor exit sequence.
- Conditional writes, immutable operation retries and eligible response-only hydration maintain the existing PDS/AppView authority boundary. The review found no new credential exposure or generic write capability.
- The optional `editedAt` field has an Accepted ADR and generated counterparts. Strict invalid-record rejection remains intact. Current-only metadata and publication eligibility are distinct from historical body storage.
- The material pending-input problem exists in the shared field. Its published-edit consequence is confirmed; additional create/draft/scheduled consequences were not independently reproduced and must be regression-checked during correction.
- IR-003 remains a separate platform-confidence limit. Automated and simulated evidence does not establish real assistive/picker/browser behavior.
- Approval notes: correct IR-004 and record the promised manual results before full acceptance. This review authorizes no commit, push, deployment, production migration or live service write.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: No independent polish-only issue was identified. IR-004 changes state/persistence behavior and requires implementation correction; IR-003 is required validation. Neither should be deferred to UI polish.
- Suggested polish notes: none. No fresh visual/browser inspection was performed, so this is not a visual sign-off.

## Handoff Back To TDD Builder

- Required code correction: IR-004. Start with the exact project composer sequence in the temporary reproduction and add a permanent failing regression. Choose a handling approach that preserves committed material data and recoverable pending input, while keeping explicit Add/Remove behavior coherent.
- Suggested next failing tests: pending edit → Back → Save must not emit unintended removal; unchanged Edit → Back must not become a content edit; returning to details must not silently lose pending input; explicit Add retains the correct source index/unknown server fields; deliberate Remove still clears; cancelling the published editor performs no write.
- Verification after correction: actual project-editor/material/patch tests, shared create/draft/scheduled material regressions, complete Flutter tests and analysis. Broaden backend checks if the correction changes backend behavior; this review does not request a backend rewrite for a client state defect.
- Required human validation: IR-003. Await and assess the maintainer's MAN-001/MAN-002 results with platform, steps and outcome detail. Do not interpret the promise as a pass or waiver. MAN-003 remains optional and separately authorized.
- Stage boundary: source fixes are deferred to the next explicitly selected TDD correction stage. Only this artifact is modified by this review.
