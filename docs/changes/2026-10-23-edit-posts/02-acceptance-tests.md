# Acceptance Test Specification: Edit Published Posts

Date: 2026-10-26

Status: Simplifications approved 2026-10-27; renewed implementation review and manual acceptance pending. Risk: Medium.

Source: [01-requirements.md](01-requirements.md). The maintainer explicitly requested continuation to acceptance-test design. Implementation and simplifications 1–5 were subsequently authorized by the maintainer on 2026-10-27. Requirement and acceptance-criterion IDs remain unchanged; prior review applies to its historical scope.

## 1. Test Strategy

Design behavior-first tests around the existing general/project composers, owner write boundary and current-record projections. Acceptance scenarios describe observable journeys. Unit tests isolate field preservation, no-op detection, validation and presentation. Integration tests establish the API, conditional PDS transport, real PostgreSQL state, projection, safety and notification outcomes. Regression tests protect creation, drafts, scheduling and existing social behavior. Manual checks are limited to assistive technology, real media pickers, browser interaction and optional real federation verification.

No tests have been implemented or executed in this stage. All new filenames below are **proposed** targets, not existing capabilities. Existing suites are explicitly labeled. No live accounts or production services are needed to author or run the default automated matrix.

Existing test context:

- Flutter uses `flutter_test`, Riverpod overrides, controlled async completions, fake repositories and localized/theme harnesses. Relevant existing suites include `app/test/projects/widgets/project_composer_sheet_test.dart`, the adjacent project composer field/media/submit suites, `app/test/feed/widgets/post_card_test.dart`, `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/data/post_repository_test.dart` and `app/test/feed/composer/composer_submission_coordinator_test.dart`.
- Go uses `testing`, table-driven cases, `httptest`, fake PDS transports, and migrated PostgreSQL fixtures from `testdb.WithMigratedSchema`. Reuse patterns from `appview/internal/pdscommands/addressed_command_service_integration_test.go`, `appview/internal/api/post_writes_test.go`, `appview/internal/index/post_notification_projection_test.go` and `appview/internal/ingestion/image_safety_replay_integration_test.go`.
- The existing `app/integration_test/critical_journeys_test.dart` is an iOS/Android loopback-service suite. It proves running UI-to-socket behavior, not real PDS writes, and is not a web runner. Keep native picker/federation claims separate.
- Use controlled clocks and explicit projection barriers/async completions; do not use arbitrary sleeps as evidence of convergence. Assert both positive outcomes and forbidden side effects. A mocked endpoint response alone does not prove persistence, moderation, notification suppression or eventual consistency.

### Coverage dimensions

Apply these as named data-driven cases, not an indiscriminate Cartesian product:

| Dimension | Required cases | Main verification |
|---|---|---|
| Post subtype | General, each currently supported project craft variant, reply, quote-post, imported post with provenance | AT-001, AT-002, UT-001, IT-001, REG-006 |
| Content | Text/facets, sponsorship, languages; every existing editable project field; optional absent/present/cleared values | AT-002, UT-001, UT-002, IT-007 |
| Media/embed | No media, existing/reordered/removed/new images, alt text, existing/replacement video, external card and thumbnail, fixed quote target | AT-002, UT-001, IT-008 |
| Presentation | Home/feed, profile, search, saved posts, detail/thread, comments, repost, quote preview wherever post metadata/menu exists | AT-001, AT-003, AT-004, REG-005 |
| Ownership | Active owner, non-owner, unauthenticated, ineligible owner, switched account, invalidated lifecycle | AT-007, IT-002, IT-003 |
| Submission | Success, unchanged, invalid input, upload failure, write rejection, lost response, retry, concurrent edit/delete | AT-005, AT-006, AT-008, IT-004–IT-006 |
| Platform/input | iOS/Android touch, web pointer/keyboard, screen reader, localized dates | AT-003, IT-015, MAN-001, MAN-002 |

The concrete field/surface inventory below addresses DR-003. For each editable field assert its original value is visible, a changed value persists, and clearing persists where legal. Do not treat caption-only success as coverage of all project fields. Existing option catalogs and validators determine valid fixture values; unsupported data remains preserved rather than becoming new editor controls.

### Field fixture inventory (TD-010)

| Named fixture | Fields / values to exercise | Primary cases |
|---|---|---|
| General and shared content | Text, UTF-8 facets (mention/link/tag), sponsorship true→false and false→true, languages, image ordering/removal/addition, alt text, retained and replacement video, external link-preview retention/removal/replacement through the normal creation flow | AT-002, UT-001, UT-002, UT-004, IT-008 |
| Project common | `craftType`, `additionalCrafts`, `status`, `title`, `duration`, `materials`, `colors`, `designTags`; preserve other common extension data | AT-002, UT-001, UT-002, IT-007, IT-009 |
| Project pattern | `patternUrl`, `patternName`, `patternDifficulty`, `patternSelfDrafted`, `patternDesigner`, `patternPublisher`; existing facet annotations on supported text fields | AT-002, UT-001, UT-002, IT-009 |
| Sewing | `sewingProjectType`, `sewingProjectSubtype`, `sewingSizeMade`, `sewingFitNotes` | AT-002, UT-001, UT-002 |
| Knitting | `knittingProjectType`, `knittingProjectSubtype`, `knittingYarnWeight`, `knittingNeedleSize`, `knittingGaugeStitches`, `knittingGaugeRows`, `knittingGaugeMeasurement`, `knittingGaugeUnit`, `knittingFinishedSize` | AT-002, UT-001, UT-002 |
| Crochet | `crochetProjectType`, `crochetProjectSubtype`, `crochetYarnWeight`, `crochetHookSize`, `crochetGaugeStitches`, `crochetGaugeRows`, `crochetGaugeMeasurement`, `crochetGaugeUnit`, `crochetFinishedSize` | AT-002, UT-001, UT-002 |
| Quilting | `quiltingProjectType`, `quiltingProjectSubtype`, `quiltingSize`, `quiltingPiecingTechnique`, `quiltingMethod` | AT-002, UT-001, UT-002 |
| Embroidery and unknown craft-specific data | Exercise shared supported fields with embroidery selected; preserve additional raw data without inventing craft-specific controls | AT-002, IT-009, REG-006 |
| Fixed identity and context | DID/key/collection, original createdAt, general/project presence, reply root/parent and quote reference full data; absent→present, present→absent and target/CID replacement attempts | UT-008, IT-006 |
| Provenance and unsupported extensions | Imported source, unknown top-level/common/material/variant fields and unknown embed union values | UT-001, IT-009, REG-006 |

Inventory sources: `app/lib/projects/composer/project_composer_fields.dart`, `project_composer_hydrator.dart`, `project_composer_payload.dart`, `project_composer_submit_adapter.dart` and the existing general/project composers. The current draft adapter's `_knownFields` set omits `patternSelfDrafted` even though the field is in the composer, hydrator and payload: published-edit coverage must include it and must not use that set alone as a completeness oracle. This is a preservation requirement for this feature, not authorization for an unrelated draft fix.

### Surface fixture inventory (TD-011)

| Named fixture | Existing UI path | Expected coverage |
|---|---|---|
| Home/feed and repost | `app/lib/feed/pages/feed_page.dart` → `PostCard` | Eligible owner's existing menu, edited original metadata, current reposted content; share metadata remains independent |
| Profile posts/projects/reposts | `app/lib/profile/widgets/profile_tabs/profile_post_feed_slivers.dart` → `PostCard` | Menu and indicator through the shared card; project details retained |
| Profile comments | `app/lib/profile/widgets/profile_tabs/profile_comments_tab.dart` → `PostCard` | Eligible author's reply edit entry and current metadata |
| Search / tag search | `app/lib/search/pages/search_page.dart`, `tag_search_page.dart` → `PostCard` | Menu/indicator plus refreshed discovery results after projection |
| Thread original / comments / replies | `app/lib/feed/pages/post_thread_page.dart` → `PostCard` in each presentation | Correct owner post edited; root/parent fixed; each edited row exposes its own timestamp |
| Quote list and embedded quote | `app/lib/feed/pages/post_quotes_page.dart`; `PostCard` → `_QuotePreviewCard` | Quoting post's menu/metadata stays separate from original embedded metadata; embedded old-CID reference resolves latest visible original |
| Saved posts / folder | `app/lib/saved_posts/widgets/saved_post_row.dart` → `PostSummary`; `saved_post_row_actions.dart` | Indicator in existing summary metadata; Edit only where an existing owner post context menu exists; do not add one to unrelated folder/save controls |
| Project rendering | `app/lib/projects/widgets/project_card.dart` and corresponding post detail/card surfaces | Updated structured content remains a project; post edit icon belongs to existing post metadata, not every project field |

REG-005 must instantiate these named fixtures with edited/unedited state, owner/non-owner context and appropriate general/project data. Shared widgets need focused interaction tests; route-level fixtures prove each caller supplies the correct edit identity. No new embedded-target menu is required where the original preview has none.

## 2. Requirement Coverage Matrix

Every row refers to actual cases specified below. All acceptance criteria have a proposed verification path; design-dependent assertions are called out in section 9.

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-005 | AT-001, IT-001, REG-004 | Acceptance / Integration / Regression | Yes |
| BR-002 | AC-006, AC-007 | AT-003, UT-005, MAN-001 | Acceptance / Unit / Manual | Yes + assistive checks |
| FR-001 | AC-001, AC-002 | AT-001, AT-002, IT-002, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-002 | AC-002, AC-003 | AT-002, UT-001 | Acceptance / Unit | Yes |
| FR-003 | AC-003, AC-004 | AT-002, AT-008, UT-002, UT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-001, AC-008 | AT-001, AT-008, UT-006, REG-001, REG-002, REG-003 | Acceptance / Unit / Regression | Yes |
| FR-005 | AC-003, AC-009 | AT-002, UT-001, UT-002, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-006, AC-010 | AT-003, UT-003, IT-010, IT-014 | Acceptance / Unit / Integration | Yes; metadata representation chosen in coding plan |
| FR-007 | AC-006, AC-007 | AT-003, UT-005, IT-014, REG-005, MAN-001 | Acceptance / Unit / Regression / Manual | Yes + assistive checks |
| FR-008 | AC-011 | AT-004, IT-011 | Acceptance / Integration | Yes |
| FR-009 | AC-012 | AT-005, IT-004 | Acceptance / Integration | Yes |
| FR-010 | AC-004, AC-013 | AT-006, AT-008, UT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-014 | AT-006, UT-007, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-015, AC-019 | IT-007, IT-014, REG-006 | Integration / Regression | Yes; current-record timestamp contract in requirements section 12 |
| RULE-001 | AC-001, AC-016 | AT-001, IT-002 | Acceptance / Integration | Yes |
| RULE-002 | AC-016, AC-017 | AT-007, IT-002, IT-003 | Acceptance / Integration | Yes |
| RULE-003 | AC-005, AC-018 | UT-008, IT-001, IT-006, REG-004 | Unit / Integration / Regression | Yes |
| RULE-004 | AC-020 | IT-012, REG-007 | Integration / Regression | Yes |
| RULE-005 | AC-021 | IT-013, IT-011 | Integration | Yes |
| RULE-006 | AC-022 | IT-010, REG-008 | Integration / Regression | Yes + schema/design review |
| RULE-007 | AC-023 | AT-005, IT-006 | Acceptance / Integration | Yes |
| NFR-001 | AC-007, AC-024 | AT-003, IT-015, REG-001, REG-002, REG-003, MAN-001, MAN-002 | Acceptance / Integration / Regression / Manual | Mixed by platform |
| NFR-002 | AC-025 | UT-009, IT-016 | Unit / Integration | Yes |
| NFR-003 | AC-026 | IT-017 | Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Edit an existing post from its context menu at any age

Requirement IDs: BR-001, FR-001, FR-004, RULE-001

Acceptance Criteria: AC-001, AC-002, AC-008

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_flow_test.dart` and `app/test/projects/widgets/project_edit_flow_test.dart`; persistence verified by IT-001.

```gherkin
Scenario Outline: Update a post without republishing it
  Given I own an eligible published <type> post with an existing context menu
  And it was published <age> ago and already has <edits> edits
  When I choose Edit, change its caption and choose Save changes
  Then the existing post displays my changed caption after reconciliation
  And the write targets the same post URI with its base version
  And no new post, scheduled item or ordinary draft is created
  And no product edit-window or edit-count error is shown

  Examples:
    | type    | age       | edits |
    | general | 1 minute  | 0     |
    | project | 1 year    | 0     |
    | general | 1 year    | 12    |
```

Repeat context-menu entry assertions across the named presentation inventory in REG-005. Existing service rate limits still apply; this scenario does not claim infinite requests are accepted.

### AT-002: Prefill and edit the complete existing content

Requirement IDs: FR-001, FR-002, FR-003, FR-005

Acceptance Criteria: AC-002, AC-003, AC-009

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_prefill_test.dart` and `app/test/projects/widgets/project_edit_prefill_test.dart`, extending existing field/media composer fixtures.

```gherkin
Scenario: Edit supported content without losing unrelated data
  Given I own a post containing saved content, media and optional fields
  When I choose Edit
  Then the corresponding composer shows every existing supported value
  And the editor can open and save immediately after publication before Tap has ingested or projected the post
  And existing attachments and their alt text are already present
  And reply or quote context is shown without a target-changing control
  And a general/project conversion control is not offered
  And adding, removing or replacing reply/quote context is not offered
  When I change supported values, clear optional values and save
  Then the requested changes and removals persist
  And untouched values, provenance and unknown fields survive
  And unchanged media is not uploaded again
```

Run named cases for all subtype/content/media dimensions in section 1, including supported optional nested project data. Use an untouched unknown open-union value to prove preservation without claiming the editor can render or edit an unsupported variant.

### AT-003: Disclose only the latest edit time accessibly

Requirement IDs: BR-002, FR-006, FR-007, NFR-001

Acceptance Criteria: AC-006, AC-007, AC-010, AC-024

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_indicator_test.dart`, existing `app/test/feed/widgets/post_card_test.dart`, plus MAN-001.

```gherkin
Scenario: Show the latest edit time without a history interface
  Given an unedited post without known edit metadata
  Then its metadata has no edit icon
  When a content edit succeeds at T1
  Then the post metadata shows an edit icon
  And hover, keyboard focus or mobile tap exposes localized T1
  And assistive technology can discover the edit time
  When another content edit succeeds at T2
  Then the indicator exposes T2 rather than T1
  And it does not open revision history
  When the unchanged post is submitted
  Then the edit time remains T2
```

Assert all named metadata surfaces. Freeze locale/time zone and clock; test another locale and a daylight-saving boundary. For external updates assert one localized Edited label for declared editedAt; absent metadata produces no icon even on a known transition. These assertions establish consistent formatting, not a new date-format convention.

### AT-004: Reposts and quotes display the latest visible original

Requirement IDs: FR-008

Acceptance Criteria: AC-011

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_share_refresh_test.dart`; IT-011 proves backend hydration.

```gherkin
Scenario: Update a quote preview even though its reference has an older CID
  Given Bob reposted Alice's post and Carol quoted its original version
  And Carol's commentary discusses the original wording
  When Alice edits the wording and image and the update converges
  Then Bob's repost and Carol's quote preview show Alice's latest visible content
  And Alice's embedded metadata shows its latest edit indicator
  And Carol's commentary, publication time and quote-post identity are unchanged
  And Bob's share identity and time are unchanged
  And no historical-version warning appears solely because the CID differs
```

### AT-005: Preserve work on concurrent edit or deletion

Requirement IDs: FR-009, RULE-007

Acceptance Criteria: AC-012, AC-023

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_conflict_test.dart`; IT-004 and IT-006 prove the server boundary.

```gherkin
Scenario: Another client saves first
  Given I opened version V1 and typed unsaved changes
  And another client has saved V2
  When I save my V1-based edit
  Then I see that the post has changed
  And V2 is not silently overwritten or merged
  And my unsaved input remains recoverable while I review the current content
  And any deliberate resubmission uses the reviewed current version

Scenario: The post was deleted while I was editing
  Given I opened a published post and typed unsaved changes
  And that post is now deleted
  When I save
  Then I see that the post is unavailable
  And my unsaved text remains recoverable in this session
  And the post is not recreated
```

Do not prescribe a merge editor, automatic rebasing or a new durable draft mechanism.

### AT-006: Handle failures and delayed indexing without false rollback

Requirement IDs: FR-010, FR-011

Acceptance Criteria: AC-013, AC-014

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_submission_test.dart`, `app/test/feed/providers/post_edit_reconciliation_test.dart`; IT-005 and IT-007 prove persistence/projection.

```gherkin
Scenario: Accepted save has not yet been indexed
  Given I edited a post and its write has been accepted
  And an older AppView response arrives before projection completes
  When I return to a loaded copy or refresh
  Then accepted or pending state is represented coherently
  And the older response does not become a false final rollback
  When projection completes
  Then current representations converge to the accepted version

Scenario: The write response is lost
  Given I have unsaved input and an edit request has an uncertain outcome
  When I retry or tap Save changes again
  Then the app reconciles the existing attempt without creating another post
  And it does not report failure as success or discard recoverable input
  And the same accepted operation does not receive a second edit timestamp
```

Separately exercise load failure before usable prefill, upload failure before write, known rejection and terminal dependency failure. Loading failure must not open an empty form that can overwrite the post.

### AT-007: Keep editing within the original owner's session

Requirement IDs: RULE-002

Acceptance Criteria: AC-016, AC-017

Priority: Must. Level: Acceptance.

Automation Target: proposed `app/test/feed/widgets/post_edit_account_fencing_test.dart`; IT-002 and IT-003.

```gherkin
Scenario: Switch accounts during editing
  Given I opened Alice's edit session and started an asynchronous media or save operation
  When I switch to Bob before that operation completes
  Then Bob cannot submit Alice's edit session or see Alice's unsaved input
  And completion does not overwrite Bob's current post or composer state
  And any already-submitted write remains attributed to Alice under existing lifecycle rules
```

Do not assert that switching accounts retracts an already accepted remote write. Also cover lifecycle invalidation before the server effect begins.

### AT-008: Validate and cancel without publication side effects

Requirement IDs: FR-003, FR-004, FR-010

Acceptance Criteria: AC-004, AC-008, AC-013

Priority: Must. Level: Acceptance.

Automation Target: proposed edit-flow widget suites alongside existing general/project validation and discard suites.

```gherkin
Scenario: Correct invalid input before saving
  Given an edit session with an invalid required value or content over an existing limit
  When I choose Save changes
  Then I see the applicable validation feedback and retain my input
  And no invalid record replaces the published post
  When I correct the value and save
  Then only the existing published post is updated

Scenario: Cancel a changed edit
  Given a published edit session with unsaved changes and an unrelated saved draft
  When I close the composer and confirm discarding using the existing close protection
  Then the published post and unrelated draft are unchanged
  And no scheduled item or ordinary new-post draft is created
```

## 4. Unit Test Cases

Targets are proposed unless marked existing. These tests verify meaningful value/side-effect contracts, not private method call order.

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, FR-005 | AC-003, AC-009 | Hydrate every supported field while preserving untouched data | TD-001, TD-002, TD-004; non-default values; unknown nested fields/open union | Prefill faithful; unchanged data remains semantically identical; persisted timestamps/unknown data are not lossy-normalized | Proposed `app/test/feed/composer/post_edit_hydration_test.dart`; project counterpart |
| UT-002 | FR-003, FR-005 | AC-003, AC-009 | Distinguish absent, untouched, cleared and false/empty values | Optional project fields, sponsorship true→false, language changes, image removals, unknown sibling fields | Explicit clears persist where valid; no original-value fallback; unrelated data survives; facet offsets match changed UTF-8 text | Proposed edit payload tests in `app/test/feed/composer/` and `appview/internal/api/` |
| UT-003 | FR-006 | AC-006, AC-010 | Detect effective content change | Unchanged prefill, reverted edits, actual text/alt/order/sponsorship change | No-op leaves edit metadata untouched; real supported changes count as edits | Proposed `appview/internal/api/post_edit_request_test.go`; Flutter dirty-state counterpart |
| UT-004 | FR-003, FR-010 | AC-004, AC-013 | Existing validation boundaries in edit mode | Exact and over-limit graphemes/bytes, invalid fields, invalid media combinations, multilingual facets | Existing policies apply; failures identify actionable fields; no invalid effect or input loss | Proposed request tests; existing project validation suites extended |
| UT-005 | BR-002, FR-007, NFR-001 | AC-006, AC-007 | Indicator date/semantics presentation | No metadata, T1/T2, non-default locale/time zone, DST boundary | No icon for unedited; latest localized instant for edited; focus/touch/semantics expose same time | Proposed `app/test/feed/widgets/post_edit_indicator_test.dart` |
| UT-006 | FR-004 | AC-008 | Edit submission selects the update operation | General/project edit origin, unrelated draft and schedule intent | Save label is correct; create/schedule/draft-consumption effects absent; cancellation makes no write | Existing `app/test/feed/composer/composer_submission_coordinator_test.dart` extended |
| UT-007 | FR-011 | AC-014 | Reconcile stale responses against confirmed edit state | Deferred old read, accepted V2, subsequent V2/V3 read, failure | No final rollback; pending resolves; newer authoritative V3 can supersede V2 | Proposed `app/test/feed/providers/post_edit_reconciliation_test.dart` |
| UT-008 | RULE-003 | AC-018 | Fixed structure rejects malicious mutation | General↔project, changed DID/key/time, quote target and reply root/parent: absent→present, present→absent, different URI, different CID and malformed/partial reference; omitted protected fields | Reject structural changes, preserve fixed data, allow legitimate content changes | Proposed `appview/internal/api/post_edit_request_test.go` |
| UT-009 | NFR-002 | AC-025 | Client encodes edit version/operation contract and decodes outcomes | Current contract fixtures; validation/conflict/pending responses | CamelCase body, required existing auth/device headers, retained version/operation identity on retry, decoded errors; no PDS OAuth credentials | Existing post API/repository tests extended after route design |

## 5. Integration Test Cases

Each PostgreSQL case uses a disposable migrated schema. Fake PDS transports must model conditional writes at the remote boundary, rather than only checking an AppView cached CID. Use valid generated CID/TID values when exercising actual validators or hashing.

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, RULE-003 | AC-001, AC-005 | Update identity and engagement in place | TD-001, TD-005; normal and imported chronological positions; surrounding posts and valid pin | Apply accepted edit then project it | Same record URI/key and createdAt; unchanged likes/reposts/replies/saves/pin and original sort position; one current post | Proposed `appview/internal/api/post_edit_commands_test.go`, existing post/store/index suites |
| IT-002 | FR-001, RULE-001, RULE-002 | AC-002, AC-016 | Ownership, eligibility and unlimited-age boundary | TD-001, TD-006; owner/non-owner/anonymous; current eligibility matrix | Submit old/high-edit-count and unauthorized edits through routed middleware | Eligible owner accepted without product time/count cap; denied cases make no remote mutation; status/error follows existing API policy | Proposed `appview/internal/api/post_edit_http_test.go`, route capability suites |
| IT-003 | RULE-002 | AC-017 | Lifecycle race at effect boundary | Owner A generation changes while request is gated; request session for B | Release request into existing owner fencing | Invalid generation/session denied before effect; already accepted A result not exposed as B state | Existing addressed-command lifecycle tests extended; proposed post-edit tests |
| IT-004 | FR-009 | AC-012 | Atomic stale-write rejection | Two editors at V1; advance PDS to V2 after AppView validation but before conditional write | Submit V1-based write using actual fake transport condition | PDS refuses stale write; V2 remains; conflict returned; no automatic overwrite; explicit later request against V2 may succeed | Existing `addressed_command_service_integration_test.go`, proposed `post_edit_commands_test.go` |
| IT-005 | FR-010 | AC-013 | Lost response, replay and failures | TD-007; accepted remote write with response loss; same operation and separately conflicting reuse | Retry, replay after process/service reconstruction, inject pre-write rejection | Exactly one logical accepted edit/time; reconciliation reads authoritative record; changed operation intent cannot hijack accepted replay; no new record | Existing addressed command/PDS transport integration patterns extended |
| IT-006 | RULE-003, RULE-007 | AC-018, AC-023 | Reject conversion, retargeting and recreation | General/project/reply/quote records; delete after authoritative read before write | Crafted requests and save/delete races including retry | Fixed fields rejected including relationship insertion/removal and reference CID-only mutation; absent target stays absent; conditional update cannot fall back to create | Proposed post-edit API tests plus addressed command integration tests |
| IT-007 | FR-011, FR-012 | AC-014, AC-015 | End-to-end controlled projection convergence | Accepted V2 held before indexing; TD-002, TD-005; cached V1; removed tags/fields | Release ingestion/projection and read feeds, project detail, search, languages and mentions | V2 current and derived data correct; removed values absent; pending distinguishable; no stale final state | Existing migrated-schema post/index/search suites; proposed edit integration cases |
| IT-008 | FR-003, FR-005 | AC-003, AC-009 | Preserve/rewrite media under existing upload rules | TD-003; existing video without its original upload job; new verified replacement video | Caption/alt edit, reorder/remove/add images, change external card, retain/replace video | Existing refs reused; new refs validated; original video not needlessly reprocessed; no invalid replacement publication | Proposed API edit-media tests with existing video/image verifiers |
| IT-009 | FR-005 | AC-009 | Preserve authoritative extension/provenance fields | TD-004 at PDS; narrower AppView projection; supported edit payload | Update from authoritative record | Untouched unknown top-level/nested/open-union/provenance data survive; no replacement from a lossy UI projection | Proposed post-edit command integration tests |
| IT-010 | FR-006, RULE-006 | AC-006, AC-010, AC-022 | Latest timestamp persists without history | TD-001; deterministic T1/T2; normal app/server restart | Edit twice, no-op, restart/read | Server-confirmed current-version time persists and renders T2; no-op/replay leaves it unchanged; one accepted operation retains one time; no new history collection or public history API; operational/moderation storage preserved | Proposed post-edit persistence/API tests; design review for no-history architecture |
| IT-011 | FR-008, RULE-005 | AC-011, AC-021 | Hydrate old-CID shares using latest visible original | TD-005 with old CID; current original visible, hidden, blocked and deleted cases | Read quote/repost representations after edit | Visible current original returned with edit metadata; surrounding share text/time fixed; old CID alone causes no warning; existing hidden/blocked/deleted policies still enforced | Existing `appview/internal/api/post_store_test.go` and read/quote suites extended |
| IT-012 | RULE-004 | AC-020 | No social notifications caused by edits | TD-008; existing unread/read notifications and pending push deliveries; added/removed mentions | Project edit, retry/replay it, complete prior pending work | No new edit-caused notification/outbox/push; existing read/activity state not reset; old reply/quote notifications not reactivated; pending original eligible reasons deliver at most once and newly added/removed recipients follow the explicit matrix below | Existing `post_notification_projection_test.go` and notification lifecycle/push integration suites extended |
| IT-013 | RULE-005 | AC-021 | Safety and evidence survive new versions | TD-009; report/hold/enforcement; cached clear scan for same image and new pending/rejected image | Edit content/media; deliver stale scan result; read direct/share views | Current-version gates apply; stale approval cannot clear new content; identical-image reuse follows current policy; reports/evidence not erased; existing enforcement persists | Existing ingestion image-safety replay and moderation test suites extended |
| IT-014 | FR-012 | AC-019 | Valid external updates and ordering | Previously indexed V1; V2/V3 external events; duplicate, stale and first-seen snapshot cases | Ingest with authoritative ordering and backfill | Latest valid content wins without losing derived data; stale/duplicate events cannot regress it; current-record and first-seen behavior follows requirements section 12 and the explicit matrix below | Existing `appview/internal/ingestion/` and index suites; current-record timestamp behavior specified below; persistence/wire mechanism selected in coding plan |
| IT-015 | NFR-001 | AC-024 | Native running-app edit/save/cancel/error path | TD-001 served by existing critical-journey loopback service; iOS/Android runner | Edit via menu, save, receive conflict and cancel another edit | Running UI/client sends an update contract; correct outcome displayed; no create/schedule request | Existing `app/integration_test/critical_journeys_test.dart` extended; no native picker/federation proof claimed |
| IT-016 | NFR-002 | AC-025 | API and credential boundary | Authenticated routed request, recording client socket and PDS transport; credential canaries | Load/save edit and decode every response state | `/v1/`, camelCase and error envelope preserved; only AppView performs record write; Flutter never receives/persists OAuth or DPoP material; approved media exceptions unchanged | Existing post API client, routed HTTP and auth transport suites extended |
| IT-017 | NFR-003 | AC-026 | Serialized safe diagnostics with one issue owner | Credential/private payload canaries and typed cause/stack; consent/sink variants | Fail load/upload/conditional write/projection and serialize configured local/SDK sinks | Permitted operation references and original cause retained; canaries absent; one capture owner; routine conflicts not spuriously escalated beyond existing policy | Existing Go testlog/observability and Flutter reporter suites, extended for edit paths |

### IT-010 / IT-014 current-record timestamp outcomes (approved 2026-10-27)

FR-006/FR-012; AC-006/AC-010/AC-019. Verify: CraftSky assigns one stable editedAt for a changed operation; no-op/failure/retry does not advance it; external declarations are displayed using one Edited label; missing metadata has no icon even after a known transition; future schema-valid declarations remain declarations; malformed/wrong-type records remain rejected; stale events cannot restore older metadata. No content digest, observation fallback or confirmation-evidence correlation is required.

### IT-012 original publication notifications (approved 2026-10-27)

RULE-004; AC-020. Hold original work pending. Original recipients may receive once after edits, including removed mentions. Edit-added recipients never receive a notification. Replays and removal/re-addition neither create a delivery nor reset activity/read state. Original eligibility must remain bounded before source coalescing; unknown edited snapshots cannot establish recipients. Existing deletion/blocking/moderation controls remain independently tested. Notification retraction purely because an edit removes a mention is no longer required.


## 6. Regression Tests

All automated regression cases include acceptance-criterion references even where the protected behavior also predates editing.

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | General/project creation and validation | FR-004, NFR-001 | AC-008, AC-024 | Existing composer submission/validation suites: new-post CTA and create semantics remain; no edit version required for creation; both composer types publish normally |
| REG-002 | Local draft restoration and close protection | FR-004, NFR-001 | AC-008, AC-024 | Existing `app/test/drafts/` and project discard/hydration suites: drafts retain identity/content; closing published edit neither consumes an unrelated draft nor creates a new ordinary draft |
| REG-003 | Scheduled-post editing/publication and media staging | FR-004, NFR-001 | AC-008, AC-024 | Existing `app/test/scheduled_posts/` and backend scheduled suites: scheduled edits use their existing resource/CTA, preserve intended schedule and do not become immediate published edits |
| REG-004 | Engagement, saved items, pins and chronology | BR-001, RULE-003 | AC-005 | Existing API post/timeline/profile-pin/saved-post suites: URI-linked interactions survive edit; later like/unlike/repost/unrepost still work; pagination does not move edited posts or duplicate them |
| REG-005 | Shared metadata/menu/rendering surfaces | FR-001, FR-007, FR-008 | AC-002, AC-006, AC-007, AC-011 | Parameterized widget/provider suites: each existing menu exposes Edit only for eligible owner; each metadata surface shows latest edit state; quote icon does not replace the quoting author's metadata; no menu added where none exists |
| REG-006 | Imported/project/extension handling and external ordering | FR-005, FR-012, RULE-003 | AC-009, AC-015, AC-018, AC-019 | Existing project evolution/crafts/import/ingestion tests: provenance and original import sorting survive; project variants retain fields; stale replay cannot roll back current state |
| REG-007 | Initial notifications, blocks and moderation | RULE-004, RULE-005 | AC-020, AC-021 | Existing notification/post-read/safety suites: new posts still generate eligible notifications; editing suppresses new ones without disabling initial delivery; hidden/blocked media remains gated |
| REG-008 | Existing operational/moderation retention | RULE-006 | AC-022 | Existing command retention and moderation evidence tests: repeated edits add no feature history archive, while required existing recovery/evidence records retain their policy-defined lifecycle |

## 7. Test Data

All fixture identifiers/media/credentials are synthetic or disposable. Use schema-valid records, actual supported project option catalogs and valid media fixtures; never copy private production content. Valid CID fixtures should represent the serialized content when a test verifies hashing.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Base posts and time behavior | Eligible A/B accounts; general and project posts at recent/one-year ages; stable URI/key; original creation time; no metadata/T1/T2; version V1/V2/V3 | AT-001, AT-003, AT-005, IT-001, IT-002, IT-010, IT-015 |
| TD-002 | Complete field round-trip | Every current supported project variant/field with non-default values, optional clears, multilingual text/emoji/combining marks, valid UTF-8 facet offsets, sponsorship and languages | AT-002, AT-008, UT-001, UT-002, UT-004, IT-007 |
| TD-003 | Media | Existing image blob refs with alt/aspect metadata; reordered/new/removed image cases; existing/replacement video and verification result; external-card thumbnail | AT-002, IT-008, IT-013, MAN-002 |
| TD-004 | Preservation | Unknown top-level/nested project keys, optional unknown open-union variant, import provenance; exact original timestamp string | AT-002, UT-001, UT-002, IT-009, REG-006 |
| TD-005 | Social graph and surfaces | Likes/reposts holding V1 CID; quote with independent commentary/time; root/parent replies; saved-post folder, valid pin and surrounding chronological posts | AT-004, IT-001, IT-007, IT-011, REG-004, REG-005 |
| TD-006 | Eligibility/fencing | Existing anonymous/non-owner/suspended/deletion/lifecycle fixture states; two retained accounts; deferred media/read/write completions | AT-007, IT-002, IT-003 |
| TD-007 | Failure and race injection | Lost response after apply, timeout before apply, retryable/terminal dependency failures, version change or deletion between read and conditional write, old deferred read | AT-005, AT-006, IT-004, IT-005, IT-006 |
| TD-008 | Notification baselines | Initial reply/quote/mention events, read/unread state and push baselines, new/removed mentions; first publication held before notification projection | IT-012, REG-007 |
| TD-009 | Safety and diagnostics | Clear/pending/rejected image states and stale result; existing moderation report/hold with evidence; protected canaries and original typed causes | IT-013, IT-017, REG-008 |
| TD-010 | Explicit editable/fixed field inventory | Named per-field fixtures in section 1, including self-drafted pattern flag and structural presence transitions; valid existing catalog values | AT-002, UT-001, UT-002, UT-008, IT-006, IT-009 |
| TD-011 | Explicit presentation/entry-point inventory | Named route/shared-widget fixtures in section 1; edited and unedited metadata; owner/non-owner; general/project/reply/quote and saved summary | AT-001, AT-003, AT-004, REG-005 |

## 8. Manual Checks

Manual verification supplements the automated contract; it is not evidence of automated test execution. Device/browser steps must use a local/test environment and controlled data. Production mutation is outside this stage.

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-002, FR-007, NFR-001 | AC-006, AC-007, AC-024 | Actual assistive technology and tooltip usability | On iOS VoiceOver, Android TalkBack and web keyboard/screen reader, open post/quote metadata; discover edit icon, tap/focus/hover; repeat at larger text size and alternate locale | Latest date/time is understandable, reachable and localized; touch does not depend on hover; focus not trapped; tooltip does not unintentionally navigate/open a composer |
| MAN-002 | FR-003, FR-005, NFR-001 | AC-003, AC-009, AC-024 | Native media picker and browser interaction | With disposable/local posts on iOS/Android use real picker to replace an image, retain a video, edit alt text and cancel; on web exercise edit/save/conflict with keyboard and touch-sized viewport | Existing attachments are prefilled; retain/replace/cancel works without data loss; no unsupported desktop-native target; web flow works independently of native loopback suite |
| MAN-003 | FR-008, FR-009, FR-011, FR-012 | AC-011, AC-012, AC-014, AC-019 | Optional real PDS propagation | Under separately authorized disposable-account validation, edit a post, observe PDS→ingestion→AppView version, read an old-CID quote, then race two sessions | Latest content and correct conflict observed; record URI stable; report measured propagation without asserting instantaneous federation or complete third-party cache invalidation |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Observable timestamp behavior specified; persistence/wire design remains | FR-006, FR-007, FR-012, RULE-006 | Revised requirements section 12 and IT-010/IT-014 define current-record and first-seen outcomes; exact schema/source association is not selected | Coding plan chooses storage, optional public metadata representation, validation and current-version linkage. No unresolved test expectation is delegated to implementation; the single-label declaration contract is approved under ASM-002 |
| GAP-002 | Assumptions remain labeled, not separately approved | FR-001, RULE-004, RULE-002 | Requirements ASM-001–ASM-005 include replies/imports, suppression of newly added mention alerts, removed pending reasons, timestamp qualifiers, restricted posts and no durable edit drafts | Carry into document review; revise test expectations only with explicit product decisions, not by changing requirements during this stage |
| GAP-003 | Web and real OS interactions lack a shared existing E2E runner | NFR-001, FR-007 | Critical-journeys suite uses dart:io and native targets; widget tests do not prove real screen-reader/picker/browser behavior | Automate widget contracts plus iOS/Android loopback journey; perform MAN-001/MAN-002. A future browser runner can replace repetitive manual web checks |
| GAP-004 | Fake PDS/scan/push boundaries do not prove live provider behavior | FR-008, FR-010, FR-011, RULE-005 | Local integration proves transport/state logic, not remote replication timing, real scanners or push delivery | Keep MAN-003 optional and evidence separate; default automated suite remains hermetic with disposable Postgres |
| GAP-005 | Observable delayed-publication outcomes specified; classification mechanism remains | RULE-004 | Revised requirements section 12 and IT-012 distinguish original/new/removed recipients and already delivered work | Coding plan must show feasible bounded operational eligibility/dispatch handling without retaining earlier bodies as feature history; fixed original-recipient policy is approved under ASM-003 |
| GAP-006 | "No history" cannot be proved by a UI assertion alone | RULE-006 | Existing operational/evidence records legitimately retain data, and tests cannot prove absence of every possible future archive | Pair IT-010/REG-008 with design/schema review; prohibit new feature archives without deleting existing required evidence |

No blocking product question prevents coding planning. DR-001/DR-002 now have explicit proposed observable outcomes and named test cases; GAP-001/GAP-005 retain only implementation design dependencies and labeled review assumptions. The gaps above are explicit limitations, not claims of passing coverage. Overall risk remains Medium; document review is recommended.

## 10. Out Of Scope

- Implementing or executing tests, production mutations, deployments, migrations, schema changes or commits in this stage.
- Public history/diffs/rollback or retained historical media retrieval; old-CID quote tests must not depend on fetching previous content.
- General/project conversion, conversation retargeting, scheduled edits, progress announcements or additional notifications.
- A new durable edit-draft store, merge editor or broad composer redesign.
- Universal third-party federation timing, revoking already delivered notification text or erasing copies outside CraftSky.
- Native macOS/Windows/Linux app builds. Desktop hosts remain valid for tests and browser use.
- A full Cartesian product of fields × surfaces × platforms. Use the named per-field preservation matrix and representative shared integration paths, then expand only where failures or distinct implementations justify it.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-10-23-edit-posts/01-requirements.md` (revised alongside this specification on 2026-10-26).
- Test specification: `docs/changes/2026-10-23-edit-posts/02-acceptance-tests.md`.
- Next review artifact: `03-document-review.md` in the same folder.
- External Plannotator review, if initiated by the maintainer outside this skill: `docs/changes/2026-10-23-edit-posts/`.
- Recommended first failing test: the general-post vertical slice of AT-001 with a real API/command integration case from IT-001: an owner edits the caption using the same URI and expected version, observes the edited result, and no create effect occurs. Implement its smallest meaningful failing boundary first; include the no-op and confirmed timestamp expectations defined in requirements section 12; coding planning must select their storage/wire representation first.
- Suggested implementation test order: (1) ownership/fixed-structure/conditional update and field preservation; (2) successful general-post composer/save; (3) all project/subtype/media cases; (4) conflict/delete/retry/account races; (5) metadata, projection and live quote/repost reads; (6) notification/safety and diagnostics; (7) shared workflow regression, device and accessibility evidence. Run each new test red before implementing its behavior.
- Commands discovered (not executed in this stage):
  - Root: `just app-test test/feed/widgets/post_edit_flow_test.dart` for the proposed focused Flutter suite, or `just app-test` for the full suite. This recipe refreshes host Rive native libraries before `flutter test`.
  - Root: `just app-analyze` separately; `just app-test` does **not** itself run analysis. With FVM, use the checkout's established `fvm exec just ...` invocation; direct FVM Flutter commands run from `app/`.
  - Root: `just test` runs Go tests with race detection against compose Postgres; requires the existing local stack. Use the project wrapper/environment setup, not a guessed database URL.
  - Root: `just appview-check` is the release-equivalent isolated disposable backend gate; a unit-only run is incomplete evidence.
  - Root: `just app-test-integration <device-id>` runs the existing iOS/Android loopback critical journeys after extension. It is not a web or real PDS runner.
  - Root: `just lexgen-check` if design introduces public record schema changes; follow existing Lexicon generation/validation workflow. This is a later implementation check, not an instruction to change schemas now.
- Blocking gaps: none for writing a coding plan. The plan must implement the explicit timestamp/notification outcome matrices, select persistence/classification mechanisms and carry ASM-002/ASM-003 proposed clarifications into maintainer review. See GAP-001, GAP-002 and GAP-005.
- Review response: DR-001 is addressed by current-record timestamp rules and IT-010/IT-014 matrix; DR-002 by original-recipient rules and IT-012 matrix; DR-003 by TD-010/TD-011 and explicit structural insertion/removal/CID-mutation cases; DR-004 by corrected requirements/test command descriptions. `03-document-review.md` remains the unchanged historical report, not a fresh review of these revisions.
- Status/evidence: all tests are planned, none implemented or executed. Leave both documents uncommitted unless the maintainer requests a commit. Continue only after the maintainer chooses document review, explicitly skips review for coding planning, adds a note or stops.

### Confirmed-save navigation refinement, 2026-10-27

AT-001/FR-011/AC-014: for both general and project posts, a confirmed PDS save with no hydrated AppView post immediately closes to the caller and presents the frozen edited version. No blocking follow-up read or Refresh post step. Check text, project leaves and retained/replaced attachments, exact accepted metadata, existing-row/quote overlays and unchanged engagement/order. AT-006 uncertain/conflict failures retain existing input/retry behavior. UT-007 confirmed overlays survive slow indexing beyond the transient timeout and reconcile to matching or demonstrably newer safe reads; protected/missing rows are never resurrected.

### Approved simplification regression scope, 2026-10-27

AT-002/IT-008 retain unchanged cards and replace/remove them through normal preview controls; no manual title/description/thumbnail fields. UT-007 verifies confirmation-only responses with no public hydration dependency. Shared general/project submission tests retain success, frozen ambiguous retry, media recovery, account fencing and dismissal behavior. Previous timestamp-source and removed-mention withdrawal assertions are superseded by the matrices above. MAN-001/MAN-002 remain pending maintainer results.
