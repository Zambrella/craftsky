# TDD Implementation Plan: Edit Published Posts

Status: Automated review corrections complete, 2026-10-27. IR-001/IR-002 are corrected and verified; IR-003 awaits the maintainer's manual results. A new implementation review has not been run.

## Inputs

- Requirements: `01-requirements.md` (revised 2026-10-26).
- Tests: `02-acceptance-tests.md`.
- Document review: `03-document-review.md` (Approved with notes).
- Coding plan: `04-coding-plan.md`; explicitly approved by the maintainer.
- Current branch: `feat/edit-posts`; starting HEAD `e168266b` after merging `origin/main` at `7e7cd639`.
- Implementation authorized by the maintainer; commits, pushes and production actions are not authorized.

## Implementation Rules

- Work one test ID/behavior at a time; record a meaningful red result before implementation, then the same test green.
- Refactor only while green. Preserve requirement, acceptance and test IDs.
- Reuse existing owner/lifecycle/security controls under the approved plan; no new privileges or live changes.
- Keep unknown public record fields server-side and preserve fixed post identity/context.
- The repository explicitly requires the missing `writing-architecture-decision-records` skill for the ADR before a Lexicon change. Do not bypass that prerequisite without a maintainer override.

## Startup And Prerequisite Status

- Workflow contract loaded from all four current files before source edits.
- Current main already fixes `patternSelfDrafted` persistence; the earlier plan observation is stale and does not require a separate draft fix.
- Main also changes composer bottom actions and alt-text state; preserve those behaviors in the shared composer implementation and regressions.
- Missing ADR skill searched in project `.agents`, installed `.agents`/`.codex` skills, plugin caches, `.claude` and `/Users/douglastodd/dev/agent-skills`; no matching skill found.
- Maintainer decision requested: use existing repository ADR format instead, or provide the missing skill path. This is the approved plan CPQ-002 prerequisite, not a request to reapprove the feature.
- Resolved: maintainer explicitly authorized the existing ADR format. ADR 022 records the accepted contract. Implementation loops can proceed; no additional feature approval is required.

## Test Order

Order follows the first occurrence of IDs in the approved coding-plan section 9. AT-001 describes the same initial vertical behavior as IT-001; its UI portion is completed when composer wiring becomes available, not claimed as completed by a backend-only test.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State | Status |
|---|---|---|---|---|---|
| 1 | IT-001 | BR-001, RULE-003 | AC-001, AC-005 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 2 | AT-001 | BR-001, FR-001, FR-004, RULE-001 | AC-001, AC-002, AC-008 | Missing general composer edit origin; first slice now green | Automated core green; evidence and limits below |
| 3 | UT-001 | FR-002, FR-005 | AC-003, AC-009 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 4 | UT-002 | FR-003, FR-005 | AC-003, AC-009 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 5 | UT-003 | FR-006 | AC-006, AC-010 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 6 | UT-004 | FR-003, FR-010 | AC-004, AC-013 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 7 | UT-008 | RULE-003 | AC-018 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 8 | IT-006 | RULE-003, RULE-007 | AC-018, AC-023 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 9 | IT-009 | FR-005 | AC-009 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 10 | IT-002 | FR-001, RULE-001, RULE-002 | AC-002, AC-016 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 11 | IT-003 | RULE-002 | AC-017 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 12 | IT-004 | FR-009 | AC-012 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 13 | IT-005 | FR-010 | AC-013 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 14 | IT-016 | NFR-002 | AC-025 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 15 | UT-009 | NFR-002 | AC-025 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 16 | IT-010 | FR-006, RULE-006 | AC-006, AC-010, AC-022 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 17 | IT-014 | FR-012 | AC-019 | Missing behavior; executed subcases recorded below | Green: strict malformed rejection and valid observed-time fallback |
| 18 | IT-007 | FR-011, FR-012 | AC-014, AC-015 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 19 | IT-008 | FR-003, FR-005 | AC-003, AC-009 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 20 | IT-011 | FR-008, RULE-005 | AC-011, AC-021 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 21 | IT-013 | RULE-005 | AC-021 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 22 | REG-004 | BR-001, RULE-003 | AC-005 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 23 | REG-006 | FR-005, FR-012, RULE-003 | AC-009, AC-015, AC-018, AC-019 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 24 | REG-008 | RULE-006 | AC-022 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 25 | IT-012 | RULE-004 | AC-020 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 26 | REG-007 | RULE-004, RULE-005 | AC-020, AC-021 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 27 | AT-002 | FR-001, FR-002, FR-003, FR-005 | AC-002, AC-003, AC-009 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 28 | AT-008 | FR-003, FR-004, FR-010 | AC-004, AC-008, AC-013 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 29 | UT-006 | FR-004 | AC-008 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 30 | REG-001 | FR-004, NFR-001 | AC-008, AC-024 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 31 | REG-002 | FR-004, NFR-001 | AC-008, AC-024 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 32 | REG-003 | FR-004, NFR-001 | AC-008, AC-024 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 33 | AT-005 | FR-009, RULE-007 | AC-012, AC-023 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 34 | AT-006 | FR-010, FR-011 | AC-013, AC-014 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 35 | AT-007 | RULE-002 | AC-016, AC-017 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 36 | UT-007 | FR-011 | AC-014 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 37 | AT-003 | BR-002, FR-006, FR-007, NFR-001 | AC-006, AC-007, AC-010, AC-024 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 38 | AT-004 | FR-008 | AC-011 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 39 | UT-005 | BR-002, FR-007, NFR-001 | AC-006, AC-007 | Missing behavior; executed subcases recorded below | Green: interaction, regional locale and controlled DST cases |
| 40 | REG-005 | FR-001, FR-007, FR-008 | AC-002, AC-006, AC-007, AC-011 | Missing behavior; executed subcases recorded below | Automated core green; evidence and limits below |
| 41 | IT-017 | NFR-003 | AC-026 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Automated core green; evidence and limits below |
| 42 | IT-015 | NFR-001 | AC-024 | Missing behavior; executed subcases recorded below | Green: complete loopback suite on iOS and Android simulators |
| 43 | MAN-001 | BR-002, FR-007, NFR-001 | AC-006, AC-007, AC-024 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Not run: actual assistive technology and browser usability |
| 44 | MAN-002 | FR-003, FR-005, NFR-001 | AC-003, AC-009, AC-024 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Not run: native picker and actual browser interaction |
| 45 | MAN-003 | FR-008, FR-009, FR-011, FR-012 | AC-011, AC-012, AC-014, AC-019 | Not run; expected missing behavior (manual IDs require separate platform evidence) | Optional live proof; not authorized or run |

## Implementation Steps

### Startup: ADR prerequisite

- Confirmed failure: no test failure claimed. The missing required skill is a workflow prerequisite.
- Resolution: maintainer authorized existing ADR format; read project Lexicon skill and wrote `adr/022-published-post-editing.md`.
- First test after resolution: IT-001, proposed `TestEditPostUpdatesExistingRecord`, proving same-URI owner caption update and no create effect.
- Run command: focused `go test` with `TEST_DATABASE_REQUIRED=true` and the checked-in compose-derived environment, or root `just test` until focused setup is available; a skipped integration fixture is not red/green evidence.
- No generators, migrations, runtime tests or live services were run at startup.

## Completion Checklist

- [x] Approved workflow loaded and current main inspected.
- [x] Test-level progress and durable execution plan initialized.
- [x] ADR skill prerequisite resolved by explicit maintainer override.
- [x] All Must requirements mapped to executed evidence or explicit gaps below.
- [x] Planned automated Must tests passing; strict-validation clarification resolved.
- [x] Relevant regression tests and authoritative gates passing for the recorded scopes.
- [x] Platform/manual evidence and unexecuted checks recorded separately.
- [x] No unlinked behavior implemented.
- [x] Implementation notes and current-main adjustments finalized.
- [x] Implementation review explicitly deferred to the next workflow stage; not performed here.

## Evidence

### IT-001: initial addressed edit boundary (in progress)

- `TestEditPostUpdatesExistingRecord`: RED with HTTP 501 from the empty handler; GREEN after routing the owner request through the existing addressed command interface. Confirms same target URI/key, expected CID, unchanged exact publication timestamp and a preserved extension integer beyond JavaScript's safe integer range. This is a handler test with a command fake, not a durability/projection proof.
- `TestPostEditUsesDurableCommandTime`: real PostgreSQL migrated-schema test RED because the prepared-command callback was ignored and unchanged content was returned. GREEN after adding the optional `BuildPreparedRecord` callback. Confirms one update effect, one record, stable durable operation time and exact accepted replay after the clock advances.
- Extended initial handler test: RED because `editedAt` was missing; GREEN after using `PreparedCommand.CreatedAt` in the edit builder.
- `TestEditPostNoOpPreservesEditTime`: RED because unchanged text advanced the edit timestamp; GREEN after lossless canonical-value comparison before stamping a change.
- `TestPostEditTimestampContract`: RED because unknown `editedAt` accepted invalid timestamps/types. GREEN after the optional datetime under ADR 022 and `just lexgen`. Both schema and generated-type package suites passed; `just lexgen-check` passed.
- Commands: `GOTOOLCHAIN=go1.27.3 go test ./internal/api -run '^TestEditPost(UpdatesExistingRecord|NoOpPreservesEditTime)$' -count=1`; focused `./internal/pdscommands` test with required worktree PostgreSQL; focused `./internal/lexicon/schema` test. Go toolchain/build-cache access required sandbox escalation. Cache permission failures were setup issues, not meaningful RED results.
- Started only the worktree-local PostgreSQL and MinIO fixtures. No live service writes, commits or pushes.
- Remaining IT-001 coverage includes real HTTP/service wiring, projection, engagement and chronological-position preservation. The handler is not registered in production routes while its safety/validation contract remains incomplete.

### Command prerequisites: IT-004 / IT-005 (partial)

- Sequencing adjustment: while implementing IT-001's durable time callback, execute the backend strict-precondition and exact-recovery subcases before Flutter hydration. They are prerequisites for the edit command interface, not completion of the later conflict/error UI cases.
- `TestPostEditRejectsStaleBaseBeforeBuildingEvenIfContentMatches`: RED because stale content was accepted and the builder ran; GREEN after opt-in `StrictExpectedCID` checks the first submission before building. Existing business addressed-put integration test also passed unchanged.
- `TestPostEditLostResponseRecoversStoredBodyWithoutRebuilding`: RED with two builds after a lost response; GREEN after strict command recovery reads the immutable dispatched body from the existing command ledger. One PDS write and one build remain after recovery.
- Handler strict opt-in assertion: RED when missing; GREEN after enabling it.
- Further recovery coverage still needs stable head/revision observation and concurrent external-update cases; neither the whole IT-004 nor IT-005 is marked complete.
- IT-016 transport prerequisites: head-revision test RED because Indigo exposed no metadata capability; GREEN after additive typed head metadata. An initially invalid fake CID was corrected at the real syntax validator boundary; that fixture error is not a behavior RED.
- Write-commit test RED because the adapter discarded the write response; GREEN after optional `ApplyWritesWithCommit` retains the returned CID/revision with exactly one request. Existing error-only write protocol/outcome tests remain green.
- Coordinated metadata forwarding test RED because the outer client did not expose it; GREEN after forwarding both capabilities through `withClient`, retaining the existing owner/session fence. Full command revision observation and API reconciliation fields remain pending.

### UT-002: scalar presence (partial)

- `TestEditPostPresenceChanges`: RED with unsupported-field HTTP 400 for sponsorship false, empty languages and explicit languages/facets removal. GREEN after supporting these bounded presence-aware fields. Full nested project/media presence coverage remains pending.
- `TestEditPostRejectsNullSponsorship`: RED with HTTP 200/false when null was supplied; GREEN after validating the merged raw record against the Lexicon before typed checks.
- `TestEditPostChangedTextDoesNotRetainOldFacets`: RED retaining offsets from the old caption; GREEN after discarding those facets when text changes and replacement facets are not supplied.
- `TestEditPostProjectLeafChangesPreserveUnknownSiblings`: RED with unsupported project edit; GREEN after supported nested common/pattern leaf merging. Covers null title, false self-drafted flag and retained unknown project/common/pattern fields including a large integer.
- `TestEditPostReordersMaterialsWithUnknownSiblings`: RED because replacement material entries lacked their authoritative siblings; GREEN with bounded retained-entry indices and raw copying. Source indices are not published.
- `TestEditPostDetailsLeafPreservesUnknownSiblings`: RED with rejected sewing-detail patch; GREEN after retaining the known variant and applying supported leaves.
- `TestEditPostCraftDetailsChanges`: RED with rejected knitting/crochet/quilting patches; GREEN after bounded per-variant inventories and nested gauge merging. Includes unchanged gauge rows/measurement/unit/extensions alongside changed stitches. Complete hydration, media/field and facet checks remain pending.
- `TestEditPostChangedPatternTextDoesNotRetainOldFacets`: RED retaining old name annotations; GREEN after dropping stale text-paired facets for supported nested pattern/material strings.
- `TestEditPostValidatesUTF8FacetRanges`: RED accepting partial-emoji/out-of-bounds/empty ranges; GREEN after checking byte ranges and UTF-8 boundaries. Valid multibyte annotations remain accepted.
- `TestEditPostValidationReturnsActionableFields`: RED with generic invalid-request envelope; GREEN after retaining typed field validation in stored command rejection.
- `TestEditPostTextLimits`: RED with lost field feedback at over-limit values; exact grapheme/byte limits already accepted. GREEN after typed policy validation precedes complete merged-record schema validation.

### IT-010 / IT-014: pure current timestamp state (partial)

- Sequencing: timestamp-source value logic is being proved in neutral `postrecord` before database/source/command integration, following the approved dependency chain. These pure tests do not establish durable ingestion behavior.
- `TestObserveEditFirstSnapshotHasNoInferredEdit`: RED with missing current identity/digest; GREEN after canonical AT-data content digest (excluding only editedAt), with no discovery-based timestamp.
- `TestObserveEditFirstDeclaredSnapshotUsesReportedTime`: RED with absent declared time; GREEN after usable author declaration is labeled reported. Non-default offset is compared by instant.
- `TestObserveEditKnownChangeUsesObservedFallback`: RED with absent edit time on known changes; GREEN after missing/removed/future declarations use the observation instant/source.
- `TestObserveEditMetadataOnlyKeepsKnownTime`: RED with lost/advanced time or downgraded confirmation; GREEN after unchanged content digest carries the known instant/qualifier onto the new current version. Large integers and reordered JSON keys remain semantically stable.
- `TestObserveEditStaleEventCannotRollBackCurrentState`: RED with old content/time replacing current confirmation; GREEN after revision-order guard. CID names intentionally sort opposite the source revision.
- Remaining: duplicate/rebuild/unknown-content transition cases, accepted-write confirmation correlation, durable migrated state, ingestion/reconciliation integration, serving-CID association and API/Flutter fields.

### IT-010 / IT-014: durable current timestamp state (partial)

- `TestPostEditIngestionPersistsObservedTimeAcrossRestart`: RED because no current edit state existed. GREEN after additive migration 102 and transactional capture before source replacement, persisting one current row. Reconstructed store, duplicate event and stale event retain the same observed instant and source version.
- Metadata rows reference source URI with cascade deletion; no old bodies or per-version archive are introduced. Migration ran only through disposable migrated test schemas. No developer database or production migration was applied.
- Ingestion's partial-schema fixture helpers now apply the new migration. Full ingestion package passed once without race after this addition (5.737s); log `/tmp/craftsky-edit-ingestion-tests.log`.
- `TestPostEditReconciliationPersistsObservedTransition`: RED because the repository reconciliation path retained old metadata after replacing source content. GREEN after shared before/after transition hooks. Both live/reconciliation focused migrated-schema tests pass together.
- Remaining: schema/cleanup/invalid-first-seen/no-op matrices in durable paths, accepted exact-write confirmation correlation, serving-CID association, ordered command recovery and API/Flutter exposure.
- Third stable-tree race run passed all eight touched packages with required PostgreSQL and compose-derived MinIO: postrecord (1.297s), ingestion (9.188s), auth (20.422s), pdscommands (19.625s), api (122.056s), index (28.602s), lexicon/schema (1.458s), lexicon/craftsky (1.325s). Log `/tmp/craftsky-edit-backend-tests-round3.log`. Full release gate, Flutter, platform/manual cases and feature completion are still pending.

### IT-001: projection chronology subcase

- `TestPostEditPreservesPublicationPositionAndRelationships`: RED because every replacement recalculated `profile_sort_at`, including the clamp for future-dated imports. GREEN after retaining the original sort anchor on conflict. Both ordinary/imported cases keep one post, its publication timestamp, save and pin.
- Existing import provenance regression exposed the old expectation that removing provenance repositions a post at current indexing time. Revised that expectation to the approved stable-publication-position rule; provenance classification still tracks the current record. Focused new/existing tests pass together.
- Race-enabled broader run passed command and API packages, then the index package build raced a file move/addition during ongoing edits (`mergePostProjectEdit` not found). That run is not an overall pass. Rerun all touched packages from a stable source tree after the current loop; do not change source while that gate is running. No Flutter or live-platform validation is claimed.
- Fresh stable-tree rerun passed all five touched packages with `-race -count=1`: pdscommands (18.679s), api (108.983s), index (34.292s), lexicon/schema (1.409s), lexicon/craftsky (1.283s). Log: `/tmp/craftsky-edit-backend-tests-stable.log`. This is a backend slice regression pass, not feature completion or the full release gate.
- Read-ordering follow-up: `TestPostEditFutureDatePreservesTimelineActivity`, `...SearchPosition`, `...CommentPosition`, `...ProjectSearchPosition` each produced meaningful REDs showing a future-dated edited post moving after `indexed_at` advanced. GREEN after timeline, chronological/popular/hashtag/project/relevance search, conversation ordering and cursor calculations also bound by the preserved `profile_sort_at` anchor. Fresh migrated-schema fixtures include active owner and current-CID safety clearance. Missing lifecycle/safety/query setup failures were corrected before treating ordering failures as RED evidence. Broader regression pass above predates these read-query changes and must be repeated.
- Second stable-tree race run passed all six touched packages with compose-derived PostgreSQL/MinIO environment: auth (20.371s), pdscommands (20.900s), api (123.945s), index (27.514s), lexicon/schema (1.459s), lexicon/craftsky (1.329s). Log: `/tmp/craftsky-edit-backend-tests-round2.log`. Index identity/relationship fixture now uses the exact migrated schema. Further focused TDD work after this run requires final broader verification again.

### IT-007 / IT-010 / IT-011: serving metadata and live quotes (partial)

- Unsupported detail preservation: RED when null/replacement removed an unfamiliar variant; GREEN after protecting unsupported details while allowing supported unrelated edits. Additional preservation characterization passes for unknown detail/embed variants and large integer extensions.
- Atomic command characterization: the edit fixture now models native PDS update existence and record-CID preconditions, and preserves integers on reads. Concurrent replacement/deletion between record read and head read is rejected without applying a write or recreating the record. Existing command transport already supplied the correct conditional update; no new production behavior was needed for this additional proof.
- `TestPostEditProjectionExposesCurrentMetadata`: after resolver fixture corrections, meaningful RED with missing metadata through real ingestion, media-safe projection and GET handler. GREEN after migration 103, CID-matched serving metadata copy, typed read-row/wire fields and normal response mapping.
- Metadata projection is a separate update in the source projection transaction and does not advance indexing time. Replacing content clears older metadata before copying the exact matching CID; legacy direct/unversioned indexer fixtures carry no invented source revision.
- `TestPostEditMetadataStaysWithServingCIDWhileSourceIsAhead`: passes against the serving-copy design. Holding projection leaves old body/time/revision together; draining advances all three to the new version.
- Quote-preview metadata test: RED with omitted embedded metadata although old-CID references correctly hydrate current URI content. GREEN after compact-preview fields. Own quote commentary/edit state remains independent.
- Remaining: full serving/saved/safety regression matrix, local confirmation and revision evidence, owner loading/update routing, notification rules and Flutter implementation.
- Actual-search query-plan regression first exposed a missed dedicated search row scanner after the shared wire columns grew; added its three scan destinations. It then produced a meaningful RED showing a sequential hot-post scan and sort because the old chronological index matched the earlier expression. GREEN after migration 104 updates the existing index to the stable three-input clamp. The test captures and explains the real public store query on representative cardinality; no mocked or copied query is used.
- Fourth stable-tree race run passed all eight touched packages: postrecord (1.239s), ingestion (9.886s), auth (21.420s), pdscommands (24.020s), api (136.467s), index (32.854s), lexicon/schema (1.456s), lexicon/craftsky (1.308s). Required PostgreSQL and compose-derived MinIO enabled; log `/tmp/craftsky-edit-backend-tests-round4.log`.

### IT-005 / IT-016: exact write receipt evidence (in progress)

- Internal optional write metadata capability now returns a `RepositoryWriteReceipt` containing the actual commit plus per-write record results. This extends the approved commit-evidence design to handle an acknowledged write followed immediately by another external version; it avoids substituting a later read's identity.
- Refactored the new capability while focused adapter/fence tests were green; legacy error-only write methods are unchanged.
- `TestIndigoPDSClientPreservesReturnedRecordIdentity`: RED because record results were discarded. GREEN after typed URI/CID result capture.
- Edit PDS fixture upgraded while focused tests were green to generate content-addressed DAG-CBOR CIDs and deterministic repository revisions/receipts, while preserving native conditional update behavior. This is fixture fidelity, not a new product path.
- `TestPostEditOperationKeyCannotTargetAnotherURI`: RED when a key reused against a different URI with identical base CID/changes silently replayed the first result. Added target URI to strict-edit fingerprint conditions; focused GREEN and existing addressed-command regressions pass. Non-strict callers retain their original fingerprint shape.
- Remaining: forward receipt through command transport, persist exact commit evidence, use stable observation brackets after lost responses, atomic current edit confirmation, real owner edit service/route and client behavior. No write has been routed to this incomplete feature.

- `TestPostEditAcknowledgedWriteKeepsReceiptWhenCurrentRecordChanges`: RED when a later external read made an acknowledged save ambiguous. GREEN after strict edits consume the exact write receipt rather than a later record read. A second RED exposed discarded commit revision in durable dispatch evidence; GREEN after migration 105 and typed completion fields persist the returned revision separately from the base head.
- `TestPostEditLostResponseRecoversStoredBodyWithoutRebuilding`: extended RED for missing observation evidence, then GREEN after bounded head/record/head recovery. Exact commit fields stay empty. Extended durable assertion RED because only the command became accepted; GREEN after atomically accepting the dispatch and persisting its separate observation CID/revision.
- `TestPostEditLostResponseWaitsForStableObservation`: characterization GREEN: three racing brackets remain ambiguous without another write; a later stable bracket accepts. The builder is never rerun after uncertain dispatch.
- `TestPostEditMissingReceiptRemainsRecoverable`: RED accepting a response with no ordering metadata; GREEN after incomplete acknowledgements become ambiguous and recover through stable observation.
- `TestPostEditFinalizationRollsBackAndRecoversWithoutAnotherWrite`: RED because the optional feature finalizer was ignored. GREEN after running it inside the dispatch/terminal-command transaction, including read-only recovery. A simulated failure rolls back feature and acceptance state; retry accepts without another PDS effect; terminal replay does not repeat finalization.
- These are backend command subcases. Owner endpoint, full confirmation correlation, notification safety, client editing, complete acceptance matrix and final gates remain incomplete.

### IT-010: source confirmation correlation (partial)

- `TestPostEditSourceBeforeAcknowledgementIsConfirmed`: RED because a real source update arriving inside the PDS effect/acknowledgement gap was labeled reported. GREEN after correlating the current raw source with the immutable active dispatch body, server-prepared timestamp and revision window in a neutral storage helper.
- Pending/ambiguous dispatches may prove a matching source before acknowledgement. Accepted exact receipts require the exact returned revision; recovered receipts allow only revisions inside their stable observation bound. A later external restore of an old accepted body is characterized as reported, not confirmed.
- Matching metadata-only transitions retain existing current confirmation under the approved digest rule. No history table or old-body archive is added. Original dispatch evidence follows existing operational retention.
- Shared edit/addressed mutation focused regressions passed after receipt, bracket and atomic-finalization changes (7.988s). A fresh stable-tree broader race run follows; no source changes during that run.

- Fifth stable-tree backend race run passed: postrecord (1.244s), ingestion (9.793s), auth (21.891s), pdscommands (27.059s), api (135.413s), index (33.689s), lexicon/schema (1.430s) and generated craftsky (1.308s). The new posteditstate helper is exercised by the command/ingestion integration test. Log: `/tmp/craftsky-edit-backend-tests-round5.log`. This remains a slice regression pass, not the release gate or feature completion.

### IT-012 / REG-007: publication notification safety (in progress)

- Dependency sequencing: begin the required notification guard before exposing owner edit routes or client controls, as required by coding-plan section 9. Current receipt/source prerequisites are green; full earlier-ID/UI matrices are still pending.
- First focused behavior: repeat publication projection with a new CID must preserve notification identity, activity/newness/read state and cannot reactivate a retracted publication event. Generic set/follow activation keeps existing behavior.

- `TestPostPublicationActivationIsInsertOnce`: after correcting setup to use the real account-level seen revision (there is no per-event read_at), meaningful RED showed a changed CID advancing activity/newness and making a seen alert new. GREEN after explicit publication activation skips reactivation. An extended reason-switch RED exposed a second event; GREEN after preserving one publication event per recipient across reply/mention/quote reasons. Existing generic set activation/newness behavior stays green.
- `TestPostEditBeforeInitialProjectionKeepsOriginalRecipients`: RED delivered an alert to the edit-added mention. GREEN after migration 106 and capture of original publication facts before source replacement, followed by current/original mention intersection. Replay/stale versions do not expand the set. Facts include bounded original timestamp/strong references and mention DIDs, no previous body.
- `TestPostEditOnlySnapshotDoesNotInventReplyNotification`: RED guessed a reply recipient from first-seen update; GREEN after unknown original eligibility suppresses speculative catch-up. Backfill-only snapshot suppression is characterized separately.
- `TestPostEditCannotReplaceOriginalMentionWithNewReplyReason`: RED allowed a newly introduced structural reason after removal of the original mention. GREEN after requiring complete original/current parent, root and quote references to match before evaluating those reasons.
- Initial-publication/dependency fixtures now explicitly mark their sources live; the old helper silently used backfill. Renamed the dependency-order regression accordingly. Its former changed-reference catch-up expectation now requires no new quote recipient, as the approved original-reason rule requires. Normal live reply/quote/mention membership/dependency recovery remains tested, not globally disabled.
- Focused index/notification/ingestion regressions passed after those changes: index 16.516s, notifications 0.209s, ingestion 2.000s. Log `/tmp/craftsky-edit-notification-focused-round2.log`.
- `TestPostEditPendingNotificationKeepsOriginalActivityTime`: RED used an externally changed current creation timestamp for delayed work. GREEN using the captured original publication time.
- `TestPostEditRemovedMentionSuppressesPendingDelivery`: RED left an outbox row pending. GREEN after existing-event retraction cancels pending work; later re-addition leaves the same event retracted and the one delivery cancelled. Provider dispatch cutoff/fencing, accepted-command early applicability, local-create evidence, cleanup and the remaining notification matrix are still pending.

- `TestPostEditRemovalBeforeProjectionSuppressesPush`: after using a complete migrated profile fixture and a due delivery, meaningful RED recorded one provider send despite a committed source removal. GREEN after the final `currentDeliveryStatus` check consults original/current source applicability, independently of background projection. Existing pre-feature events without any publication marker retain their previous policy; an explicit unknown marker cannot produce speculative catch-up.
- Push package regressions passed (4.778s). The new post-source behavior uses an exact migrated-schema test; existing preview/delivery fixtures now include the required profile CID and empty publication tables.
- `TestPostEditRemovalWaitsForInFlightPublicationPush`: RED let source removal commit through an already in-flight provider attempt. GREEN with shared per-post transaction/session fences held through the bounded provider attempt/finalization. Source ingest locks before source-version rows; notification/serving projection locks after actor lifecycle fencing and before source/job rows; reconciliation does the same; changed strict-command finalization takes the post fence before feature-state callbacks. No PDS call holds a source/serving transaction.
- `TestPostPublicationFenceReusesOwnerConnectionAtCapacity`: RED deadlocked when the nested post fence acquired a second pool connection. GREEN after reusing the existing owner-fenced connection, including a competing-lock probe during/after callback. Production push adapter exposes this capability; test-only/permissive adapters use the independent pool path.
- Focused edit/fence/publication suites passed together: push 2.290s, ownerlifecycle 0.157s, pdscommands 9.188s, index 17.394s, ingestion 2.337s, notifications 1.477s, app 0.825s. Log `/tmp/craftsky-edit-fence-focused.log`.
- Still pending before routing Edit: complete lock failure/cancellation/ordering coverage, accepted-command early applicability, original local-create proof/acceptance seeding, cleanup inventory, remaining notification matrix, full media/service/client work and authoritative gates.

### REG-008 / RULE-006: lifecycle cleanup (partial)

- Full owner-lifecycle package exposed a meaningful schema-catalog RED: publication owner and mention-recipient DID roles were missing from terminal inventory. GREEN after explicit delete roles, scoped indexes and bounded publication-child drain classification.
- `TestTerminalPurgeDrainsPublicationFactsBeforeSource`: RED when deleting one source cascaded through a publication parent and five mention children outside the row budget. GREEN after source cleanup waits for the publication component; that component drains mentions under the configured row limit before deleting its locked parent.
- Extended cleanup assertion RED showed the current edit-state row silently cascading with the source. GREEN after explicitly draining/counting that child before source deletion. No PDS records are deleted by these AppView cleanup paths.
- Lock cleanup was refactored while focused tests were green to discard a possibly locked session if callback unwinding prevents a confirmed release. Callback-failure release is characterized through the owner-fenced connection; same URI can be locked afterward. Broader regression verification follows.

- Sixth stable-tree race run passed postrecord (1.253s), ingestion (9.283s), auth (19.172s), pdscommands (23.705s), notifications (2.216s), push (7.546s), ownerlifecycle (21.003s), api (117.989s), index (33.685s), schema (1.470s) and generated craftsky (1.333s). The app package failed its legacy custom video fixture, missing serving metadata columns/current edit-state table. This is not an overall passing run. Log `/tmp/craftsky-edit-backend-tests-round6.log`.
- Applied exact migrations 45, 102 and 103 to the video convergence fixture rather than changing production behavior; focused rerun follows. Fresh whole-app/final broader verification is still required after the remaining implementation.
- Next required matrix subcase: original A removed and re-added before any initial notification work runs. A final original/current intersection alone forgets that intervening removal and could issue prohibited catch-up. Prove with a RED, then retain monotonic current applicability (original reason may retire, never expand/re-enable) and restrict pending work inside authoritative source commits and local accepted-edit finalization. This is operational delivery state, not post edit history. Another remaining original reason must continue to permit the original recipient once.

### IT-012: durable retirement of removed original reasons (partial)

- `TestPostEditRemovedThenReaddedBeforeProjectionHasNoCatchup`: RED delivered a catch-up alert after both source updates were coalesced. GREEN with a monotonic eligible flag on original mention candidates, restricted inside authoritative source/reconciliation commits. Current mention rendering still includes the re-added mention; its retired original delivery reason stays disabled.
- `TestPostEditRemovedThenReaddedReferenceHasNoCatchup`: REDs for external reply and quote remove/re-add; GREEN with current eligibility flags on their original strong-reference facts. Full original refs remain fixed evidence; eligibility can retire but never expand/re-enable.
- `TestPostEditRemovingMentionKeepsOriginalReply`: characterization GREEN, proving the remaining original reply still notifies once even when its original mention reason retires. Positive initial notification behavior is retained.
- Focused original-reason/notification/push/source suite passed: index 17.667s, push 2.050s, ingestion 1.782s. Log `/tmp/craftsky-edit-retired-reasons.log`.
- Extended `TestPostEditRemovalBeforeProjectionSuppressesPush`: meaningful RED left a pending row after source commit. GREEN after retirement atomically retracts inapplicable existing events and cancels pending/retry/leased deliveries in that same transaction, under the post fence. A later push batch claims no cancelled work. Source capture persists neither prior bodies nor a sequence of edits.
- Video fixture correction is focused GREEN (0.691s); round 6 remains a partial pass until fresh whole-app/broader verification.

### IT-012: local acceptance and original-create evidence (partial)

- `TestEditPostAcceptanceCancelsRemovedMentionBeforeTap`: HTTP handler/real migrated-state RED left an original delivery pending while Tap still held the old version. GREEN after the handler supplies the feature finalizer, running monotonic restriction in the command's terminal transaction. The fake boundary executes that real hook transaction; separate real command tests prove atomic rollback/recovery/no second PDS effect.
- Source-atomic cancellation/remaining-reason focused regressions passed: index 17.955s, ingestion 1.871s, push 2.018s. Log `/tmp/craftsky-edit-source-restriction.log`.
- `TestPostEditFirstSnapshotUsesAcceptedOriginalCreateEvidence`: RED suppressed the original A along with edit-added B when only an updated source snapshot arrived. GREEN after recovering original facts exclusively from the accepted `post.create` exact step/result, bound to owner/URI/active plan. Edited snapshot content cannot seed original recipients.
- Ingestion/auth full regressions exposed one legacy repository-repair fixture without command tables; auth passed (16.640s). Changed only that failed repair fixture to `WithMigratedSchema`; focused repair rerun is GREEN (2.080s). No production schema-check fallback or mirrored command-table DDL was added. Log `/tmp/craftsky-edit-create-evidence-regressions.log` remains a failed overall run until fresh verification.
- `TestPostEditLateOriginalProofRespectsObservedApplicability`: positive kept-reason RED showed unknown markers could never become established. GREEN now promotes only on accepted original proof and intersects it with a bounded, shrinking applicability set retained while origin is unknown. The remove/re-add case stays suppressed. Unknown candidates alone cannot activate a notification and can never expand; edit-only candidates are discarded when original proof arrives. This stores only current operational eligibility facts/flags, no previous text/media/body or sequence of edits.
- Remaining: seed original facts atomically at ordinary append-command acceptance (including before Tap), final service/owner-read/media/result contract, full notification matrix and diagnostics, all Flutter work, full gates and manual evidence.

### IT-012 / REG-007: original append acceptance seeding (partial)

- `TestPostCreateAcceptanceSeedsOriginalFactsBeforeTap`: acknowledged/lost-response REDs showed append acceptance ignored its feature finalizer. GREEN after an optional append finalizer participates in normal dispatch acceptance and read-only recovery transactions. Replay runs neither the remote effect nor the finalizer again.
- Original eligibility roots can precede Tap, so migration 106 deliberately does not reference tap_source_records. They remain keyed by URI, with explicit owner/recipient cleanup roles, scoped indexes and bounded mention-child drain. Current edit metadata (migration 102) still references the current source. No source row is fabricated and no earlier post body is retained by publication facts.
- `TestCreatePostAcceptanceSeedsPublicationBeforeTap`: HTTP wiring RED left both original-fact/mention counts zero. GREEN after ordinary create supplies the append finalizer. The test proves one established marker/mention and zero Tap sources.
- `TestPostCreateProofWakesWaitingOriginalPublication`: RED left already-completed unknown work asleep after original proof arrived. GREEN after accepted seeding wakes only current valid serving-source notification work. Repeated seed preserves the original notification count/newness.
- The focused publication/proof/shared suite before these last append additions passed: ingestion 5.498s, index 24.428s, api 10.528s, push 2.982s, notifications 1.586s, pdscommands 9.753s, ownerlifecycle 1.627s, app 0.947s. Log `/tmp/craftsky-edit-publication-proof-focused.log`. Further verification is required after the append additions.

### REG-008 / RULE-006: deleted-post operational cleanup

- `TestPostDeletionClearsCurrentEditingFacts`: RED retained original recipient/ref/time facts after deletion. GREEN after clearing current edit state and publication recipients/references/dates, keeping only an ineligible current deletion marker.
- `TestLateCreateProofCannotRestoreDeletedPublicationFacts`: delete-first/known-post REDs showed a late create finalizer could promote a cleared unknown marker and restore old facts. GREEN after the explicit deleted state blocks both accepted seeding and source-proof promotion. Deletion received without an earlier source creates the same minimal marker. Source deletion payload is ignored when restricting applicability.
- Focused combined cleanup/permanence rerun passed (2.184s). No old post body or history is retained in these markers.
- Latest focused append/publication/edit/shared regression pass: ingestion 5.541s, index 25.330s, api 11.579s, push 2.941s, notifications 1.616s, pdscommands 12.952s, ownerlifecycle 1.675s, app 0.881s. Log `/tmp/craftsky-edit-append-seeding-focused.log`; it precedes the final deletion-state changes and must be followed by fresh broader verification.

### IT-002 / UT-001: private owner edit seed (in progress)

- Seventh stable-tree backend race run passed all 15 selected packages, including the full app package (43.106s), index (41.881s) and API (124.033s). Log `/tmp/craftsky-edit-backend-tests-round7.log`. This follows the append/deletion changes, but precedes the new edit-seed implementation. It is a slice regression gate, not feature completion.
- `TestPostEditSeedLoadsCurrentOwnerVersion`: missing-handler RED, GREEN after the private owner boundary and single-statement source/serving snapshot. Carries current CID/revision, supported shared values and private/no-store cache policy.
- `TestPostEditSeedIncludesSupportedProjectFields`: meaningful RED omitted the project. GREEN after generated read types expose supported common/pattern/details fields, including selfDrafted, with retained list indices. Unknown extensions remain in the authoritative server record, outside editable values; unsupported unions are flagged and omitted from controls.
- `TestPostEditSeedWaitsForValidatedCurrentSource`: meaningful RED exposed an invalidated source. GREEN after both source validation statuses join the matching-CID, current-generation and authoritative-order readiness conditions. No public route is registered yet.
- Remaining: source-lag/delete/eligibility cases, full media seed coverage, complete PATCH media/result/policy and routing, all Flutter work and final gates.

- `TestPostEditSeedWaitsForInitialProjection`: RED returned 404 before first serving projection. GREEN with a repeatable-read source/serving snapshot and same-snapshot initial-projection classification. The first seed implementation's single statement was insufficient for that distinction; both reads now share one transaction snapshot.
- `TestPostEditSeedEligibilityAndVersionBoundaries`: characterization GREEN for non-owner/anonymous, source CID ahead, uncertain order, stale projection generation, absent membership, departed owner and deleted source. No form content returned by these outcomes. Owner seed suite passes (7.801s).

### IT-001 / IT-007 / UT-004: accepted result and media (in progress)

- Upgraded initial `TestEditPostUpdatesExistingRecord` assertion to the approved accepted/pending contract: RED returned a synthetic post with invented zero counts. GREEN after a replay-safe result containing no post body, acceptedCid, changed, edit time/source and projectionPending. Exact recordRevision and separate observation CID/revision preserve receipt-versus-observation evidence. These additive identity fields allow ordering even with post:null; the pending route is not an ambiguous PDS outcome.
- `TestEditPostReordersRetainedImagesWithoutLosingAuthoredFields`: RED rejected images; GREEN with bounded retained entry indices and supported image leaves, preserving unknown siblings/wide integers. Whole merged Lexicon and image policies still validate.
- `TestEditPostRetainsVideoWithoutOriginalJob`: RED interpreted a published blob as a create request's upload descriptor. GREEN after separately mapping known public embed variants into existing validation shapes. Retained published video requires no old job and the raw embed stays unchanged.
- `TestEditPostUpdatesExternalCardWithoutLosingThumb`: RED rejected the embed field. GREEN with supported media leaf merging, preserving retained external/thumb/unknown siblings. Quote/unknown embed changes remain forbidden.
- `TestEditPostCannotAttachUnverifiedVideo`: RED accepted a new video through retained-video validation. GREEN now rejects changed video blobs pending completion verification. Replacement verification is the next loop; no route exposes this partial implementation.
- Field validation now follows approved HTTP 422; the prior draft text/facet tests expected 400. The adjacent run identified precisely those outdated assertions; updated only their status expectations. Focused edit command suite passes (1.060s). Malformed/protected-field requests retain 400.

- `TestEditPostVerifiesReplacementVideo`: missing-option RED, GREEN with the existing owner-bound completion verifier. Replacement blobs must match its receipt; jobId is stripped before storing the public desired record. Retained blobs bypass re-verification.
- `TestPostEditRetriesUnavailablePreparationWithoutRemoteEffect`: meaningful RED terminally rejected a temporary preparation outage. GREEN leaves strict edits prepared when a builder dependency is unavailable; same-key retry builds again only before dispatch, then performs one remote effect. Lost-response recovery still never rebuilds a dispatched body.
- `TestEditPostValidatesNestedUTF8FacetRanges`: RED accepted a byte range inside an emoji in a pattern name. GREEN validates caption, material and all supported pattern rich-text ranges against their own corresponding text.
- `TestEditPostHydratesAcceptedResultFromCurrentEligibleView`: RED always returned pending. GREEN hydrates only a currently eligible matching-CID post with real engagement; durable terminal results remain content-free. No-op hydration restores current metadata/source rather than inventing confirmation. Broader replay/hiding/error-sink coverage remains.
- Extended `TestEditPostAcceptanceCancelsRemovedMentionBeforeTap` with pre-feature records: RED found no original eligibility marker. GREEN prepares bounded facts from the prior source in the same accepted transaction before restricting current reasons.
- `TestEditPostRejectsCurrentlyIneligiblePost` and `TestEditPostCannotAddMentionAcrossBlock`: REDs accepted an ineligible current post and an added blocked mention. GREEN checks the current eligible snapshot and existing directed-interaction policy during preparation, before a remote write. Retained references are preserved; edits cannot use these checks to retarget them.
- Latest focused API edit suite passes (1.626s). These are boundary/subcase results, not a complete UI-to-PDS vertical acceptance claim.

### IT-002 / IT-016: route wiring (in progress)

- `TestPostEditRoutesHaveCurrentMemberPolicies`: RED missing policies, GREEN adds current-member, age-restricted-denied and suspension-denied editor routes. The private edit-seed GET is an explicit suspension exception to ordinary read availability because it grants editor prefill, while ordinary post reads remain available.
- `TestPostEditHandlersAreRegistered`: RED missing handlers, GREEN wires GET edit seed and PATCH through existing middleware, the production addressed command service and media verifier/feature configuration.
- First full route run failed exactly four tests: diagnostic metadata/inventory/suspension catalogues needed the additive routes; the interaction-read legacy fixture lacked the three serving edit columns. Updated the catalogues explicitly and that fixture only. All four focused reruns pass (0.528s). The first full run is not a passing gate; fresh full routes and release gate remain required.
- Flutter skills read: dart-add-unit-test and flutter-add-widget-test. The next UT-005 metadata test is running first red before adding client fields; no client implementation claimed yet.

- Fresh full route suite passes (20.635s), log `/tmp/craftsky-edit-routes-round2.log`. This covers all existing route tests after the explicit catalogue/fixture changes; it is not the complete backend release gate.

### UT-005 / UT-009: Flutter metadata and wire contracts (partial)

- `post_edit_metadata_test.dart`: missing-field RED, GREEN after nullable current edit time/source/revision fields on Post and QuotePreviewPost, typed source enum with unknown fallback and regenerated mapper. A quote's source/time remains independent of its commentary.
- Generation completed using pinned FVM. Removed only generator-created trailing whitespace/hash-only churn in 87 unrelated outputs; retained the necessary post mapper. No unrelated source was reverted.
- Conditional PATCH test: missing-model/method RED, GREEN with immutable presence-aware changes, typed acceptance result, concrete If-Match and stable operation key. Recovery observation revision remains separate from exact recordRevision; pending accepted content is not treated as ambiguous. Corrected an initial test-filter invocation that split words into paths; the recorded focused RED uses `--name=UT-009` from app/.
- Owner seed client test: missing-method RED, GREEN GET edit seed keeps immutable editable values and matching concrete base/post CID, with no replacement of real engagement state.
- Production repository boundary: RED after moving the conditional-wire test through ApiPostRepository, GREEN after adding fetchEdit/edit to the production interface/adapter and programmable test fake. Both touched Flutter API/metadata suites pass together: 51 tests, log `/tmp/craftsky-edit-flutter-wire-regressions.log`.
- All Flutter composer/menu/indicator/media/overlay/account race behavior remains pending; these model/wire tests do not prove the editing UI or any device/live PDS path.

### AT-003 / REG-005: shared edit disclosure (partial)

- Indicator widget missing-file RED, GREEN with localized confirmed/reported/observed/unknown tooltip and matching accessible semantics. Null edit time renders no icon. Corrected the test's semantics-handle disposal before Flutter end-of-test verification; no production change was needed for that test-harness failure.
- PostCard missing-icon RED, GREEN after the shared indicator sits beside the original creation timestamp. It does not replace or advance that timestamp.
- Compact-summary missing-icon RED, GREEN after fromPost/savedPost/fromQuoteView/notificationSubject carry edit metadata into the shared summary indicator. The quote retains its own source/time. Single focused card and compact summary tests pass; full touched widget regressions follow the image-state addition.

### UT-004 / FR-003: retained published images (in progress)

- `published images submit retained indices without bytes or reupload`: missing-phase/method RED, focused GREEN inside the image-state suite after the new PublishedImageReady origin. It is submit-ready for an edit, has no local bytes, cannot save as an ordinary draft/create payload, and serializes retained index plus current alt text.
- Filtered build_runner generation removed other tracked generated outputs, including a required create-image mapper. The first post-generation run therefore failed setup rather than behavior. Re-ran the complete generator graph, restored all required outputs, retained substantive feature mappers and removed only unrelated whitespace/hash churn. Use complete generation for later changes, not build-filter.
- First combined image/widget run passed all nine image-state tests and the indicator but exposed one additional exhaustive switch in scheduled media staging. Added explicit rejection of PublishedImageReady there; it cannot become scheduled private media. Fresh combined rerun is running; prior combined run is not a pass.
- Still pending: actual uploader mixing retained/new images, edit-seed image hydration, full general/project composers and save flow, account/conflict/retry/overlay behavior, remaining acceptance matrix and final backend/Flutter gates.

- Fresh combined disclosure/image/widget run passes all 107 tests, log `/tmp/craftsky-edit-disclosure-image-regressions2.log`. The earlier setup/exhaustiveness failures remain failed runs, not passing evidence.
- `materializePostEdit` missing-method RED, GREEN reuses the existing owner-guarded immediate uploader for only new images and combines retained indices in exact display order. Full uploader suite passes eight tests; combined image-prefill/uploader suite passes nine (`/tmp/craftsky-edit-media-hydrator-green.log`).
- `hydratePublishedPostImages` missing-helper RED, GREEN pre-fills original alt, secure current display URL, ratio and exact retained index into a published-only phase. No remote bytes are fetched by the hydrator and incomplete image/presentation counts fail instead of seeding an empty attachment list.

### IT-015 / UT-007: account-bound mutation and accepted overlays (partial)

- Edit notifier missing-provider RED, GREEN saves the seed's exact owner/key/base through PostRepository with a stable operation key and shared retry controller. Original exceptions/stacks use existing diagnostic ownership; pending accepted results carry no invented post content.
- Replace-in-place RED retained the old body. GREEN with a distinct PostEditOverlay: it replaces complete controlled content on existing visible rows, preserves newer engagement and original list position, and is excluded from create-overlay insertion into absent/filtered lists.
- Later-version RED let the accepted overlay overwrite a later external version. GREEN adds optional source revision to the shared read projection and a post-edit predicate requiring exact accepted CID or a valid later TID than exact acceptance/recovery observation bound. It deliberately does not use text-only content equality for media/project acknowledgement. The recovery bound remains separate from exact recordRevision.
- First three provider/overlay tests pass (`/tmp/craftsky-edit-ordering-green.log`). Account-bound seed provider missing-file RED is followed by generated implementation keyed by the captured activation and guarded before/after the read. Its account-switch test and shared controller/reconciliation suite are running; not yet claimed green.
- Remaining client work includes complete composer origins/prefill/save/cancel, retained video/external controls, material identities/full field diffs, menu entry, accepted-but-indexing UI, all shared surfaces/cache invalidation, conflict/account/retry matrices, diagnostics, final gates and separate manual evidence.

- Account seed first follow-up run found a missing Riverpod selection-extension import; corrected it. The next cancellation test timed out because ProviderContainer.test defaulted to automatic retry unlike production (`appProviderRetry` disables it). Aligned this fixture's retry policy with production; the complete provider/account plus shared controller/reconciliation run now passes 17 tests (`/tmp/craftsky-edit-provider-account-green3.log`). The two earlier follow-ups are not passing runs.

### AT-001 / FR-002 / FR-004: first visible general-post edit slice (partial)

- New `post_edit_flow_test.dart` starts from a six-year-old published post, opens the existing composer with a distinct PostEditOrigin, verifies caption and Save changes, changes the caption and saves through edit only. Draft/scheduling controls are absent, original creation identity stays in the seed, and success returns the edited post with Changes saved feedback.
- Missing-origin/parameter RED, then an initialization failure exposed premature editor closure while the account registry was still loading. The editor now waits for registry initialization, disables saving until its captured activation matches, and closes if established authority changes. The same focused widget test is GREEN (`/tmp/craftsky-edit-flow-green2.log`).
- The edit branch reuses the existing submission lifecycle, sends presence-aware changed shared fields and retained/new image changes, keeps unchanged text/facets/languages omitted, and cannot invoke ordinary create/schedule/draft submission. It guards final asynchronous route removal with the captured owner.
- Accepted-but-indexing currently remains in the editor with a saved/updating state and disabled repeat save. Its coherent read/retry completion remains a required next loop before exposing the Edit menu. External/video prefill/change controls and fixed reply/quote context also remain incomplete.
- Fresh shared standard composer/facet/discard/language/scheduled-edit regression run started; no result yet. This widget slice uses the programmable repository, not a live PDS or a full routed API-to-PDS journey.

- Shared standard composer/facet/discard/language/scheduled-edit regression run passes 45 tests with two existing feature-configuration skips (`/tmp/craftsky-edit-standard-composer-regressions.log`). No change to those skip conditions.
- `refreshAccepted` missing-method RED, GREEN performs only current eligible seed reads, returns no older version, and recognizes exact accepted CID or later source revision. Its test proves one write and two completion reads; provider suite passes five tests (`/tmp/craftsky-edit-pending-read-green.log`). Read failures retain original cause/stack at the explicit refresh diagnostic owner, independent of the already-final acceptance.
- Pending widget RED found no completion read. GREEN resolves an immediately available accepted version. A separate lagging-read RED then required Refresh post and locked fields; GREEN now allows read-only refresh, preserves saved form content, enables closing without a discard prompt once accepted, and never repeats the PDS command. Three general flow cases pass (`/tmp/craftsky-edit-pending-refresh-green.log`).
- Saving-progress RED caught reuse of the publishing copy. Shared submission overlay now has edit/refresh kinds with Saving changes/Updating post copy while preserving existing post/comment/reply and video stages. Focused three-case rerun follows; do not claim this last run until checked.
- Still required before menu exposure: project composer/full-field changes, media preview/replacement controls, fixed-context rendering, coherent cache/surface updates, conflict/unavailable/account/ambiguous retry and diagnostics matrices, real routed API/service integration, broader gates and separate manual evidence.

### UT-001 / UT-002 / FR-002 / FR-005: project hydration and presence-aware changes (partial)

- Published project hydration retains supported common/detail values, self-drafted patterns, bounded material indices and additional-craft identities. Normalized form payloads recover retained identities without copying opaque fields into the editor patch.
- Pattern-last-field RED emitted `pattern: null`; GREEN sends only supported leaf clears. Same-craft detail-last-field RED emitted `details: null`; GREEN preserves the detail discriminator and opaque siblings with leaf clears. Changing craft can still remove the prior supported detail variant.
- Clearing self-drafted RED lost `false` when ordinary payload normalization omitted an empty pattern; GREEN restores explicit false when the original pattern contains the checkbox field.
- Combined project hydration, patch and existing payload regressions pass 28 tests (`/tmp/craftsky-edit-project-normalization-green.log`). This is helper evidence, not full composer/media coverage.
- Project flow missing-origin RED confirmed. ProjectComposerSheet now has a separate published origin, supported prefill, retained images, owner-bound conditional save and leaf project diff; the first widget journey is running and is not yet claimed green. Pending acceptance/refresh and full media/context/error matrices remain incomplete.
- Project journey first follow-ups exposed a fixture's unresolved profile-options read and an edit notifier that was not watched by the project widget. Aligned the craft-options fixture with the existing project tests and retained the edit provider for the editor lifetime. Immediate project journey is GREEN (`/tmp/craftsky-edit-project-flow-green4.log`).
- Project accepted-pending RED demonstrated missing completion reads/refresh action. GREEN adds eligible read-only completion, saved/pending form locking and Refresh post with no repeated write. All six general/project immediate, immediately indexed and lagging-index cases pass (`/tmp/craftsky-edit-project-pending-green.log`). Full shared composer regressions and all-field/media/error/account coverage remain required.
- Shared retained-image preview RED showed a missing URL; GREEN resolves PublishedImageReady display URLs in summary and details, while preserving caller-supplied URL behavior. Full attachment widget suite passes nine tests (`/tmp/craftsky-edit-image-preview-green.log`).
- Fresh broader project, general composer and scheduled-edit regressions pass 221 tests with two existing feature-configuration skips (`/tmp/craftsky-edit-project-composer-regressions.log`). This covers existing creation/draft/scheduled behavior after published project wiring; it is not full Flutter analysis, the complete suite or device proof.

### UT-007 / REG-005 / FR-011: detail and thread read reconciliation (partial)

- Detail stale-response RED returned the old caption after an accepted edit; GREEN applies the existing accepted-content overlay while retaining fresh engagement.
- Thread comment stale-response RED reproduced the same rollback. The first shared-helper fix applied the overlay twice on detail reads and prematurely acknowledged its own optimistic CID; that failed run is not passing evidence. GREEN applies content once at each actual raw read boundary. Detail, thread and edit-provider suites pass 11 tests (`/tmp/craftsky-edit-thread-overlay-green2.log`).
- Full feed/profile/search/saved/quote invalidation, embedded quote overlay, later-version/cross-response ordering and remaining visibility matrices still require coverage.
- Active-feed invalidation RED kept the timeline repository read count at one. GREEN refreshes detail/thread/quotes, timeline/profile/project lists, post/project/tag search and saved-post provider families after acceptance and completion reads. The combined provider/composer run passes 17 tests (`/tmp/craftsky-edit-read-invalidation-green.log`); per-family stale-result coverage remains partial.
- Visible embedded quote RED retained the original caption. GREEN overlays accepted original content only for a visible preview, preserves the quote reference CID and uses exact/later revision evidence to leave newer previews alone. Provider/detail/thread run passes 11 tests (`/tmp/craftsky-edit-quote-overlay-green.log`). Hidden-preview and complete quote-surface matrices remain required.
- Search stale-row RED returned pre-edit text after invalidation. GREEN applies replace-only record overlays to post/project/hashtag search pages and incoming pagination rows, without inserting absent publications. Edit provider plus complete search-provider suite passes 30 tests (`/tmp/craftsky-edit-search-overlay-green.log`).
- Saved-row stale-content RED likewise returned the previous caption. GREEN applies record overlays to saved rows while keeping savedAt/folder metadata and raw saved-state reconciliation. Edit provider plus saved-provider suites pass 15 tests with two existing skips (`/tmp/craftsky-edit-saved-overlay-green.log`).
- Fixed-context composer RED found neither published reply-parent nor quote preview. GREEN renders fixed context through shared visibility-aware summaries, with reply parents read through AppView. The first follow-up needed the correct progress-indicator import. All six composer journeys now pass (`/tmp/craftsky-edit-context-green2.log`), including general reply and project quote contexts. Complete unavailable/reveal/account/context cases remain required.

### UT-002 / AT-002 / IT-008: retained and replacement published video (partial)

- Retained-video state missing-helper RED, GREEN tracks supported alt/removal without forwarding unchanged blob data. Reverting alt to its original value produces no patch (`/tmp/craftsky-edit-video-state-green.log`).
- Composer retained-video RED found no prefilled alt field. GREEN extends the existing video attachment card with a retained display origin and uses a leaf embed patch in both composers. Retained video satisfies existing project media requirements. Eight general/project journeys pass (`/tmp/craftsky-edit-video-flow-green.log`).
- Selected-video replacement RED showed no completion-proof consumption. GREEN consumes the prepared video proof and sends its exact blob/job metadata through conditional edit; production connects the existing upload/processing coordinator. Ten composer journeys pass (`/tmp/craftsky-edit-video-replacement-green.log`). This uses the existing proof seam, not real picker/service/PDS proof. Missing-blob recovery, upload failure/cancel/account matrices and full regression verification remain required.
- Fresh shared project/general/video/card/scheduled regression run passes 226 tests with two existing configuration skips (`/tmp/craftsky-edit-video-composer-regressions.log`). Full feature error/account/media/surface matrices, final complete Flutter/analysis and backend gates remain outstanding.

### AT-002 / UT-002 / FR-002–FR-005: published external-card edits (partial)

- Supported external state missing-getter RED, GREEN sends only changed URL/title/description/thumbnail leaves, preserves explicit empty description and supports card removal. Both external/video state cases pass (`/tmp/craftsky-edit-external-state-green.log`).
- External composer missing-controls RED, GREEN pre-fills localized metadata controls and supports thumbnail/card removal. The first follow-up missed the thumbnail button while scrolling/keyboard animation was unsettled; fixing the test interaction produces GREEN (`/tmp/craftsky-edit-external-flow-green2.log`).
- Explicit preview-refresh missing-action RED, GREEN reuses AppView preview fetching with captured-owner/cancellation guards, updates supported metadata, and uses the existing new-thumbnail uploader only at save. Original media is not fetched for preservation. Both manual-edit and refresh-without-thumbnail journeys pass (`/tmp/craftsky-edit-external-refresh-green.log`). Replacement-thumbnail and nearby composer/uploader regressions are running; not yet claimed green.
- New preview refresh diagnostics have an explicit original-cause/stack owner with bounded stage and initiating public-record references; serialized sink/canary and account/cancel failure tests are still required under IT-017.
- Replacement-thumbnail regression is GREEN: refresh with a new thumbnail performs exactly one upload and submits that returned blob, while manual edits and thumbnail-free refreshes perform zero uploads. Combined external/video composer, embed-state and uploader regressions pass 23 tests (`/tmp/craftsky-edit-external-thumbnail-regressions.log`). Actual preview service, native picker, direct PDS upload and runtime assistive checks remain separate evidence.
- Unsupported project-detail UI RED exposed a known-craft details action for an opaque variant. GREEN hides only that unsupported details action while shared fields continue through the same editor. Ten composer journeys pass (`/tmp/craftsky-edit-opaque-project-ui-green.log`).
- First full Flutter static-analysis pass is running before further error/retry/account and menu work. This is an intermediate check, not final feature completion.

### Intermediate complete-matrix checks (in progress)

- First full Flutter analysis found 175 feature-local issues (imports, explicit JSON types, constructor/style rules and async widget-context guards). Reviewed `dart fix --dry-run` and applied its 83 mechanical fixes across the 31 listed feature files; manually tightened remaining typed maps, context lifetime guards and test descriptions. Analysis is being rerun; no final analysis pass claimed yet.
- First whole-backend `go test -p 1 -race ./...` exposed a test import cycle: observability tests import push → postpublication → Tap → observability. Interrupted that already-failed run; `/tmp/craftsky-edit-backend-tests-round8.log` is not passing evidence.
- Minimum dependency fix replaces the publication helper's Tap Event parameter with a bounded, transport-independent typed Source value; both ingestion and accepted-edit callers adapt at their boundary. Focused observability/ingestion/API publication, edit and redaction tests pass under race (`/tmp/craftsky-edit-import-cycle-green.log`: 1.725s / 4.172s / 10.183s). A fresh full backend race run is underway in `/tmp/craftsky-edit-backend-tests-round9.log`.
- Full Flutter analysis is now clean (`/tmp/craftsky-edit-flutter-analysis-round4.log`: no issues). Fresh affected suites across feed, projects, search providers, saved posts, shared mutations and summaries pass 1,029 tests with 12 existing skips (`/tmp/craftsky-edit-flutter-affected-round1.log`). This is the affected matrix, not the entire Flutter suite or native/web runtime evidence.
- Next vertical work: a fresh account-bound published-editor loading route, usable by the owner menu and explicit conflict review. Each opening must fetch a fresh eligible seed without invalidating or replacing the underlying editor's recoverable unsaved form. Conflicts must not rebase retained-media indices onto a new base.

### AT-002 / AT-006 / AT-007: fresh owner-bound loading route (partial)

- Missing entry-route RED, GREEN opens the matching composer only after an eligible owner seed. A per-opening request ID separates new/review reads from a live editor's seed; invalidating a review cannot replace recoverable working input. Complete generator graph rerun succeeded; removed 89 unrelated whitespace/hash-only generated diffs and retained substantive feature outputs.
- Ten composer journeys plus edit-provider cases pass 15 tests through the new route (`/tmp/craftsky-edit-editor-loader-green.log`). Load failure/retry and late seed after account switch additionally pass with those journeys: 12 tests (`/tmp/craftsky-edit-editor-loading-regressions2.log`). The first loading-test run had an ambiguous SessionRegistry import; that setup failure is not passing evidence.

### AT-005 / FR-009 / RULE-007: explicit conflict review (partial)

- General/project conflict RED showed only generic save failure. GREEN reports a changed post and offers Review current version. Review opens a fresh current-version editor above the preserved form; closing review returns to the original unsaved input. A later deliberate save uses the reviewed base and fresh retained-media identities, without rebasing the old form or recreating a post.
- Both general/project conflict journeys pass (`/tmp/craftsky-edit-conflict-review-green2.log`); the first follow-up needed the actual fullscreen Close control rather than WidgetTester.pageBack. Known-deletion feedback and remaining failure/account/ambiguity matrices are still being implemented.
- Full backend race round 9 is still running and has exposed two stale fixture contracts so far: dispatch-column inventory omits the three new evidence fields, and the legacy observability push fixture lacks publication-state tables. Do not claim the whole run as passing or change backend source while it remains active.
- Known-deletion RED retained generic feedback and allowed another save. GREEN shows specific unavailable feedback, retains text/form values and disables resubmission for post_not_found. All six loading/account/conflict/deletion entry-route tests pass (`/tmp/craftsky-edit-deleted-ui-green.log`).
- Full backend race round 9 completed with three failures: the two stale fixture contracts above and an uncreated local MinIO bucket. The other packages passed (some unchanged packages used Go's cache). Updated only the dispatch-column inventory and the observability fixture's exact migration list; bootstrapped only this worktree's checked-in local private MinIO buckets. The three failed cases are rerunning under race; no complete backend pass claimed yet.
- Focused repairs for the three backend failures pass under race (db 2.220s, observability 1.811s, MinIO scheduledposts 1.470s; `/tmp/craftsky-edit-backend-matrix-repairs-green.log`). Whole-backend race round 10 then passes (`/tmp/craftsky-edit-backend-tests-round10.log`), with Go cache used for unchanged successful packages. The authoritative appview-check release gate still remains required.
- Unresolved edit RED lost the original cause/stack at retry exhaustion. GREEN preserves the caught ambiguous wrapper and stack using the shared retry contract. Six edit-provider tests pass (`/tmp/craftsky-edit-ambiguous-cause-green.log`), including seven bounded attempts followed by an explicit eighth retry with one operation key/base/body. Serialized sink coverage and the locked exact-attempt retry UI are still pending.
- Unresolved-save UI RED showed generic rejection copy and editable fields after the retry budget ended. GREEN distinguishes uncertain saving, locks submitted form values, and provides Retry save. The retry closure captures the final immutable PostEditChanges after facet/media preparation and calls only the original keyed save; it does not regenerate facets or upload again. Both general/project cases pass (`/tmp/craftsky-edit-unresolved-ui-green2.log`). The first attempted edit script made no source changes because its formatter-sensitive anchor failed; that follow-up remains a failed test run, not passing evidence.
- Combined loading/conflict/deletion/uncertain-save, general/project/media and edit-provider regressions are running, including an explicit once-only facet-generation assertion. Remaining work still includes complete new/removed external-card selection behavior, owner-menu surfaces, account/lifecycle and upload/validation cases, serialized diagnostic canaries, routed API-to-real-command integration, final complete gates and separate native/web/manual evidence.
- Combined retry/conflict/loading/deletion, general/project/media and edit-provider regressions pass 27 tests (`/tmp/craftsky-edit-retry-conflict-regressions.log`), including the once-only facet-generation assertion on explicit uncertain-save retry. This is local fake-repository/widget proof, not real PDS or native service proof.
- Replacement-video recovery missing-parameter RED, GREEN lets the existing bounded video coordinator receive the original missing-blob error/stack, while explicit uncertain-save retries use the frozen payload without starting another upload. Edit-provider, coordinator and general/project editor regressions pass 38 tests (`/tmp/craftsky-edit-video-recovery-green.log`); a subsequent readback caught and corrected the two composer call sites forwarding the recovery flag. Final rerun and real coordinator-to-widget wiring evidence remain required.
- Backend fresh/resumed replacement-video BlobNotFound RED returned generic 503. GREEN returns the definitive video_blob_missing outcome only for immutable replacement intent; it also works when recovery resumes the stored plan without running the builder again. Retained videos do not request fresh-upload recovery. Focused API race tests are running (`/tmp/craftsky-edit-video-missing-api-green.log`); the whole-backend round 10 pass predates this additive response mapping.
- Replacement-video API race cases pass (1.643s; `/tmp/craftsky-edit-video-missing-api-green.log`).
- Image/video aspect-ratio edit RED dropped an unknown nested wide-integer sibling. GREEN merges only supported width/height leaves into the raw ratio object. Five nearby media API race cases pass (1.623s; `/tmp/craftsky-edit-ratio-preservation-green.log`).
- Accepted-result refresh diagnostics RED consumed an unexpected DB cause without an issue. GREEN uses the existing request failure owner while preserving the already-accepted pending response. The first follow-up's missing public refs came from a fixture without the classified logging boundary; corrected it to use the actual route catalogue. Serialized local/native event sinks now retain SQLSTATE, typed cause and approved public refs, emit one issue, and exclude private edit/session/credential/database canaries. Expected pending/unavailable emit no issue. Focused API race tests pass (1.689s; `/tmp/craftsky-edit-refresh-diagnostics-green2.log`). Identity/engagement/quote failures also use the owner; their full sink matrix remains required.
- AT-007 direct-composer late-save RED reproduced Alice's dirty form remaining rendered after Bob activation. GREEN immediately removes a lost-owner form before route closure and suppresses completion feedback when the captured owner no longer matches. Four general/project late success/failure cases pass (`/tmp/craftsky-edit-late-account-ui-green.log`). The already-submitted Alice write is not retracted; its late UI outcome is discarded, with one original request.
- A first attempt to add those tests made no file changes because a formatter-sensitive fixture anchor failed; the resulting no-matching-tests invocation is not evidence. The actual RED is `/tmp/craftsky-edit-late-account-ui-red2.log`.
- Existing valid future craft values RED were rejected by creation's supported-craft inventory during unrelated edits. GREEN keeps all ordinary edit validation, but retains unchanged nonblank primary craft and existing additional craft values while still rejecting newly introduced unsupported values and duplicates. Focused unsupported/invalid edit API race cases pass (1.682s; `/tmp/craftsky-edit-future-craft-green.log`). Additional nonblank-negative and UI regression coverage follows; this is compatibility with an existing public record, not new craft creation support.

- Future-craft follow-up GREEN: existing blank additional values remain rejected (focused API race 1.647s, `/tmp/craftsky-edit-future-craft-green2.log`); unrelated project caption edits preserve an existing future craft through the UI (`/tmp/craftsky-edit-future-craft-ui-check.log`).
- AT-002 new external selection RED omitted the visible selected card from PATCH. GREEN snapshots the account-bound selected preview and materializes only its new thumbnail. Four external journeys pass (`/tmp/craftsky-edit-new-external-green.log`). The first attempt lacked the existing URL completion boundary; that setup failure is not missing-behavior evidence.
- Explicit external removal RED made the caption URL's preview immediately reappear. GREEN dismisses the preview session when the author removes the retained card. All five external journeys pass (`/tmp/craftsky-edit-remove-external-green.log`), including keeping the URL while sending embed:null.
- AT-001 / FR-001 owner menu RED found no Edit entry. Shared PostCard wiring now opens the fresh owner-bound editor without an age gate; focused GREEN verification is running. No feature completion or full surface matrix claimed yet.

- Owner-menu GREEN opens a fresh prefilled editor for a 2020 publication (`/tmp/craftsky-edit-owner-menu-green2.log`); the first follow-up needed the existing initialized-language fixture. Shared card/editor regressions pass 120 cases (`/tmp/craftsky-edit-menu-editor-regressions.log`). Eight general/project/reply/quote owner/non-owner menu cases pass (`/tmp/craftsky-edit-menu-variants.log`). Full named route-fixture coverage remains separate.
- Untouched-caption URL RED manufactured an external card on a no-op save. GREEN starts automatic preview candidates only after caption changes in published mode, including clearing candidates when reverting the caption. Six external save/remove/refresh/no-op journeys pass (`/tmp/craftsky-edit-untouched-external-green.log`).
- Real socket integration of production owner-seed/PATCH handlers, durable addressed commands, conditional PDS fixture and Tap projection passes under race (`/tmp/craftsky-edit-http-command-check2.log`, 4.418s). It proves pending acceptance, wide-integer preservation, projection hydration, exact replay, changed-request rejection, no-op timestamp and stale-base rejection. The fixture supplies authenticated context; route policy tests remain separate. First attempt lacked the transport's required fixture Lister capability and was not a product failure.
- IT-015 native loopback journey passes on iPhone 18 Pro iOS 27 simulator (`/tmp/craftsky-edit-ios-loopback-round3.log`) and Android 17/API37 emulator (`/tmp/craftsky-edit-android-loopback-round4.log`). It enters through the owner menu, edits, verifies same-post conditional PATCH/bearer/key/body, receives a 409 without losing input, and cancels without another update/create/schedule/draft write. No live PDS, OAuth, picker or assistive-technology proof is claimed.
- Initial native runs exposed fixture issues: forced 1x viewport/native inset mismatch, tapping menus/actions before native animation/keyboard completion, and omission of the production ErrorMappingInterceptor. Corrected the existing harness and interactions; no production error mapping or global Snackbar behavior changed. Failed native logs remain nonpassing evidence.
- New-video option RED was hidden because published edits are unschedulable. GREEN separates media eligibility from scheduling; published general posts expose the enabled existing video picker (`/tmp/craftsky-edit-add-video-green.log`).
- Published-reply image-prefill RED found no attachment controls despite supported existing images. GREEN shows those controls for published replies, preserving ordinary reply-creation behavior (`/tmp/craftsky-edit-reply-images-green.log`).
- Authoritative appview-check is running with isolated local artifacts under `/tmp/craftsky-edit-appview-gate`; no completed gate result is claimed yet.

- IT-017 Flutter serialized sink coverage passes for PDS-write/provider ownership and consumed accepted-read failures (`/tmp/craftsky-edit-flutter-diagnostics-green.log`). Both retain original error identity/stack, approved request/status and public record refs, emit exactly one native issue, and exclude private body/response/credential/capability canaries. Initial fixture attempts omitted Flutter binding/session initialization and expected an unapproved arbitrary platform code; those are not product failures.
- Actual explicit preview-refresh failure now has serialized local/SDK proof (`/tmp/craftsky-edit-preview-diagnostics-green2.log`): one original-cause/stack owner, public post refs, old metadata retained, no write/upload, private URL/error canaries excluded. The first widget fixture exposed an SDK fake-timer shutdown issue; logs are disabled for this SDK fixture and closure occurs outside the fake timer zone. Combined retained/new/removed external and diagnostics/privacy regressions pass ten tests (`/tmp/craftsky-edit-diagnostic-regressions.log`). The older automatic-preview source guard now permits only ErrorReporter.captureException in the shared composer, keeping direct SDK/analytics and automatic preview reporting forbidden.
- Media-exclusivity RED exposed image controls while a retained external card was present. GREEN offers those controls after explicit card removal, preventing invalid mixed link/photo submissions. All six external cases pass (`/tmp/craftsky-edit-external-media-choice-green.log`); the green scenario refactor replaces boolean tuples with named cases.
- Unsupported-embed caption RED exposed an automatic card that could not safely replace the opaque embed. GREEN gates both immediate text-change and rebuild candidate paths, suppressing invisible fetches as well as the preview. First follow-up fixed rendering but still fetched through the immediate callback; the second run passes all 23 editor/external cases (`/tmp/craftsky-edit-unsupported-embed-ui-green2.log`). Retained external/video cards use the same candidate guard.

- AT-003 tap/keyboard RED found no visible tooltip for a normal tap. GREEN makes the shared disclosure focusable and tap-triggered, retains its accessible timestamp/source label, and uses a 24px hit area around the existing 14px pencil. Touch, keyboard focus, indicator, card and summary regressions pass 108 tests (`/tmp/craftsky-edit-indicator-input-green.log`). Actual assistive technology remains MAN-001, not inferred from widget semantics.
- Full Flutter generator pass succeeds (`/tmp/craftsky-edit-generation-round3.log`, 19s, 246 outputs). Restored 89 unrelated whitespace/hash-only generated files after content comparison; retained the three substantive mapper files and new edit provider outputs. Complete Flutter analysis and the repository app-test recipe are running.

### Verification before strict-validation clarification (historical)

- Complete Flutter app-test round 1 passes 3,360 tests with 38 existing skips (`/tmp/craftsky-edit-flutter-full-round1.log`); full analysis round 8 is clean. Later field-feedback behavior and added fixtures are undergoing a fresh complete round 2 and analysis.
- TD-011 actual caller fixtures pass all 22 owner/non-owner cases (`/tmp/craftsky-edit-surface-fixtures-check2.log`): profile posts/projects/comments/reposts, quote list, search/tag search, thread root/comment/loaded reply, and saved summary/control boundary. The initial fixture run omitted FormFactorWidget and an active repost flag; corrected those fixture contracts without production changes. Native feed entry supplies the remaining named Home/feed caller proof.
- General/project production video coordinator and service-client wiring passes four cases (`/tmp/craftsky-edit-video-recovery-widget-check.log`), including one definitive missing-blob recovery. The adapter consumes actual synthetic bytes; fresh deduplication bypass is bounded to one second upload, a new verified proof and a new command key, retaining the original post/base. No external video service is contacted. This extends the earlier completed-proof seam.
- All five existing critical-journey loopback tests pass on both iOS and Android (`/tmp/craftsky-edit-ios-loopback-full.log`, `/tmp/craftsky-edit-android-loopback-full.log`), including read/account switch, edit/conflict/cancel, create, notification routing and native preferences.
- Authoritative appview-check completes successfully, exit 0 (`/tmp/craftsky-edit-appview-gate.log`, artifacts `/tmp/craftsky-edit-appview-gate`). Includes normal/race tests, Lexicon checks, migration rehearsal, pinned scanners, production Docker/media checks, packaged console/auth boundary and pinned Tap handshake. The gate itself records that Tap provides no safe deterministic injection API; it does not prove live federation or production deployment.
- Expanded accepted-refresh diagnostics query/identity/engagement/quote matrix passes under race (1.810s; `/tmp/craftsky-edit-refresh-diagnostic-matrix.log`) after the gate, with production source unchanged. Expected pending/missing cases emit no issue, unexpected failures retain one typed SQL cause owner and exclude all protected canaries.
- AT-008 server-field-feedback RED showed only a generic save error. GREEN maps known field paths to localized control labels, retains input and allows a corrected save with the same base and a new operation key in both editors (`/tmp/craftsky-edit-field-feedback-green.log`). Server prose and raw schema paths are not shown as labels.
- UT-006 save/discard versus unrelated local-draft regression passes four cases (`/tmp/craftsky-edit-unrelated-draft-check.log`): general/project, save/cancel; no create call, draft save or draft delete; original draft object/revision/content/future schedule intent remains unchanged.
- **Pending maintainer decision:** requirements 01 section 12 requests an observed-time fallback for malformed external edit metadata. ADR 022 declares a Lexicon datetime, and AGENTS.md's “Don't bypass the lexicon” rule requires schema-invalid records to be dropped. Real ingestion/projection RED (`TestPostEditMalformedExternalDeclarationUsesObservedTime`, `/tmp/craftsky-edit-malformed-time-contract-red.log`) proves malformed editedAt yields invalid/invalid source validation and no serving row. No schema/validation bypass has been implemented. The earlier passing pure timestamp fallback test did not establish this ingestion boundary. Asked whether to retain strict validation and revise this edge case (recommended), or revise the metadata schema/design. The required answer was pending at this checkpoint and is resolved in the 2026-10-27 note below.

### Remaining validation limits and stage status

- The earlier IT-014 blocker is resolved by the approved strict-validation clarification below. Schema validation remains unchanged.
- Valid absent/future declaration projection cases pass under race (4.391s; `/tmp/craftsky-edit-valid-time-fallback-check.log`) and use the exact source observation timestamp. The blocking issue is confined to schema-invalid malformed declarations.
- UT-005 interaction/source formatting and regional locale/controlled London DST characterization are green; the final evidence is recorded below. Actual assistive technology remains a separate manual check.
- MAN-001 actual screen-reader/assistive-technology assessment and MAN-002 real native picker and actual browser pointer/keyboard assessment have not been performed. Widget semantics/touch/keyboard tests and iOS/Android loopback tests do not substitute for those human/platform checks.
- MAN-003 is optional real-PDS/federation proof and has not been authorized or executed. No production data or infrastructure changes, remote CI claims, commits or pushes have been made.
- Route policy, credential-fence, real socket/command and projection proofs use their specified separate fixtures. The HTTP convergence fixture supplies authenticated context and uses a conditional PDS fixture; it is not real OAuth or a third-party PDS session.
- Full Flutter round 2 and final analysis completed successfully; final counts and scope are recorded below.

- UT-005 characterization gap is now closed: four indicator tests pass with `TZ=Europe/London` (`/tmp/craftsky-edit-indicator-locale-dst-check.log`). en-US/en-GB dates honor regional ordering and 24-hour preferences; two explicit-offset instants in the repeated 01:30 hour remain distinct as BST/GMT. Portable checks also run without assuming the host time zone. Actual assistive-technology assessment remains MAN-001.
- Final client analysis round 11 passed with no issues after field feedback and caller/video/draft additions. A final analysis pass includes the newly added locale/DST tests. No subsequent backend production changes were made after the authoritative gate.

### Independent verification checkpoint before IT-014 decision (historical)

- Complete repository Flutter app-test round 2 passes **3,392 tests, 38 existing skips**, exit 0 (`/tmp/craftsky-edit-flutter-full-round2.log`). This includes the final validation-feedback behavior, caller fixtures, coordinator recovery and unrelated draft cases. The two later locale/DST cases are additionally covered by the four-test controlled-TZ indicator run; no production code changed for those additions.
- Complete final Flutter static analysis passes with **no issues**, exit 0 (`/tmp/craftsky-edit-flutter-analysis-final.log`). Full generator, localization generation and `git diff --check` are clean for their documented scopes.
- Backend release gate passed before the final malformed-declaration contract probe. After it, the expanded diagnostic matrix and valid external fallback projection cases pass under race, with backend production source unchanged. **At that checkpoint the working tree contained one deliberately red contract test**, `TestPostEditMalformedExternalDeclarationUsesObservedTime`, pending the maintainer's answer. Do not represent the current complete backend test set or the TDD stage as all green.
- No commits, pushes, deployments, production mutations or live PDS/video-service writes were performed. Manual checks remain explicitly unexecuted. TDD stage exit and implementation review have not been invoked because the required timestamp decision is still pending.

### 2026-10-27: strict-validation decision resolved

- Maintainer selected “Keep strict validation and revise this requirements edge case.” Updated requirements section 12, the IT-010/IT-014 acceptance matrix and clarification, coding-plan edge-state row and approval status, and ADR 022. The acceptance specification already contained the invalid-record exception; the requirements row and the newly added probe had not made it explicit.
- Renamed/revised the integration probe to `TestPostEditMalformedExternalDeclarationIsNotProjected`, covering malformed datetime strings and wrong field types. It asserts invalid source validation, no serving row and no current edit metadata assigned to the invalid version. Existing production validation is unchanged.
- Strict malformed rejection plus valid absent/future fallback tests pass under race (4.736s; `/tmp/craftsky-edit-strict-time-resolution.log`). There is no remaining product-policy blocker. The broader edit-related regression result is recorded below.

### TDD stage completion, 2026-10-27

- Fresh broader editing regression matrix passes under race, exit 0 (`/tmp/craftsky-edit-final-backend-regressions.log`): API 10.746s, index 26.002s, ingestion 3.019s, PDS commands 9.475s, pure timestamp values 1.263s, notifications 1.981s and push 3.145s; posteditstate compiles and has no separate test files. The run includes strict malformed timestamp rejection and schema-valid observed-time fallback. No intentionally red contract test remains.
- This final clarification changes documents and test expectations only. Production validation and the Lexicon datetime schema are unchanged. The earlier authoritative release gate and complete client/native checks remain the recorded broader evidence; they have not been relabeled as fresh runs on this date.
- All automated implementation steps are complete. MAN-001 and MAN-002 are explicitly deferred/unexecuted human/platform checks, and MAN-003 remains optional and unauthorized. Those limitations remain visible for review and release planning.
- Implementation review has not been run or silently skipped; it is offered as the next workflow stage. No commit, push, deployment or live service mutation was made.

### Review correction pass, 2026-10-27

- Input: `06-implementation-review.md`, Changes required; maintainer selected Address required changes. The review is historical input and will not be silently rewritten as approval.
- Order: IR-001 / AT-002, UT-002 / FR-002, FR-003, FR-005 / AC-003, AC-009: first reproduce absent/empty language prefill in both composers; then explicit preferred-language assignment, clearing and no-op/revert preservation. Preserve creation/draft/schedule behavior.
- Next: IR-002 / IT-017 / NFR-003 / AC-026: reproduce missing original seed identity failure diagnostics with serialized sink/canary assertions, then use the existing request owner.
- IR-003 / MAN-001, MAN-002 / NFR-001: assess available real platform interaction/assistive capabilities; execute supported local checks and record any remaining inability without substituting widget evidence or claiming a maintainer waiver. MAN-003 stays optional and unauthorized.
- Final verification: focused correction suites, relevant language/composer/diagnostic regressions, Flutter analysis and appropriate broader gates. No commits, pushes, deployments or live service writes are authorized.

#### IR-001 / AT-002, UT-002: optional published languages

- RED: all four general/project × absent/empty cases exposed accessible language value `English` instead of empty (`/tmp/craftsky-edit-ir001-prefill-red2.log`). The first test draft looked for InputChips, but production uses a context-menu control; its passing result was not a reproduction. Switched to the actual accessibility value before implementation. A subsequent green attempt exposed test semantics-handle cleanup, corrected without changing product code.
- GREEN: an explicit `PostLanguageSelection.forPublishedEdit` permits zero tags and preserves this mode through changes. Both composers seed its exact supported values and original baseline; new/draft/scheduled composers retain their ordinary one-to-three behavior. Four prefill/no-op cases pass (`/tmp/craftsky-edit-ir001-prefill-green2.log`).
- RED: the final-language menu action remained disabled in both composers, leaving the context menu over the save button and preventing submission (`/tmp/craftsky-edit-ir001-clear-red.log`). GREEN: language controls allow final removal only for an optional published selection; both clearing cases pass (`/tmp/craftsky-edit-ir001-clear-green.log`). Added an explicit menu-dismissal assertion before saving to make failure localization clearer.
- Added assignment/reversion coverage: both composers, omitted/empty originals, explicitly choose preferred English and either save that assignment or remove it before save. Reverted or untouched originals emit no language patch; clearing an existing language emits an empty array. No record replacement/fallback value is invented.
- Relevant editor/metadata/all-language regressions pass **69 tests**, exit 0 (`/tmp/craftsky-edit-ir001-regressions.log`), including ordinary selection constraints and creation behavior. IR-001 code correction is complete pending final broader verification and review.

#### IR-002 / IT-017: owner-seed identity diagnostics

- RED: `TestEditPostSeedIdentityFailureDiagnostics` received a generic HTTP 502 issue (`operation=http.server`, `failure_stage=unexpected`) instead of the original identity failure (`/tmp/craftsky-edit-ir002-red.log`). Response behavior itself was unchanged; missing cause/stage was the defect.
- GREEN: the seed handler now reports the original error through `ReportRequestFailure` before emitting the existing safe 502. The serialized local/Sentry test passes under race, 1.808s (`/tmp/craftsky-edit-ir002-green.log`). It asserts one issue, original `net.DNSError` chain, an available original stack, request correlation, operation/identity stage and public actor, and excludes private content, DNS/host, credential and session canaries. IR-002 code correction is complete pending broader verification and review.

#### IR-003 / MAN-001, MAN-002: manual evidence remains pending

- Device discovery returned `Agent device access is turned off for this environment.` No actual native picker, VoiceOver or TalkBack check was performed.
- Tried a disposable local Flutter web fixture using production cards/composers with synthetic in-memory records and loopback-only API configuration. Initial debug-server import-path errors were fixture setup failures, not product failures. After correcting the fixture imports, it compiled and served locally, but the preview browser twice returned `Preview automation snapshot timed out after 15000ms`. The fallback computer-use connection exposed no connected browser tabs; native browser connection yielded no editing interaction evidence. No browser save/cancel/conflict, pointer/keyboard or screen-reader outcome is claimed.
- On 2026-10-27 the maintainer answered “I'll provide manual test results.” MAN-001/MAN-002 and IR-003 therefore remain pending receipt and recording of those results; this is not a waiver or a passing result. MAN-003 remains optional and unauthorized.
- Closed the agent-created preview tab, stopped the temporary local web server and removed the temporary fixture from the checkout. Its original disposable source remains `/tmp/craftsky_edit_web_probe.dart`; the attempt logs are `/tmp/craftsky-edit-ir-web-server*.log`. No live account or service writes occurred.

#### Review-correction final verification

- Full `fvm exec just app-test` passes **3,408 tests, 38 existing skips**, exit 0 (`/tmp/craftsky-edit-ir-full-flutter.log`). Includes the 14 new published-language editor cases and the previously separate locale/DST cases. Ordinary creation, drafts, scheduled posts and shared language controls remain covered.
- `fvm flutter analyze` from `app/` passes with no issues, exit 0 (`/tmp/craftsky-edit-ir-analysis.log`). No final production/test source changed after those checks; only documentation and removal of the disposable fixture followed.
- Complete API, observability and route suites pass with required local database/media fixtures and `GOTOOLCHAIN=go1.27.3 go test -p 1 -race ./internal/api ./internal/observability ./internal/routes -count=1`, exit 0 (`/tmp/craftsky-edit-ir-backend-regressions.log`): API 274.657s, observability 14.627s, routes 33.152s. This includes the new seed diagnostic case and existing edit/validation/route cases.
- Scoped Dart formatting, Go formatting and `git diff --check` pass. No schema, dependency, migration, permission or PDS-command behavior was changed in this correction pass. The earlier full AppView release gate and native loopback runs remain historical evidence; they were not rerun or relabeled as fresh results for this pass.
- IR-001/IR-002 code corrections are ready for another review. IR-003 remains explicitly open for human evidence. `06-implementation-review.md` remains the unchanged correction input; a follow-up review is offered as the next stage, not silently performed here. No commit, push, deployment, production mutation or live PDS/video-service write was made.
