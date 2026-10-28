# Coding Plan: Edit Published Posts

Date: 2026-10-26. Status: Approved by maintainer; timestamp edge clarified 2026-10-27. Risk: Medium.

This document records the approved implementation design. Execution and verification evidence is maintained in `05-implementation-plan.md`.

## 1. Inputs

- Requirements: [01-requirements.md](01-requirements.md), revised 2026-10-26; BR-001–BR-002, FR-001–FR-012, RULE-001–RULE-007, NFR-001–NFR-003 and AC-001–AC-026.
- Tests: [02-acceptance-tests.md](02-acceptance-tests.md), including the revised timestamp and publication-recipient outcome matrices and TD-010/TD-011 field/surface inventories.
- Document review: [03-document-review.md](03-document-review.md), verdict **Approved with notes**. It reviewed the preceding drafts; this plan responds to DR-001–DR-004 and the subsequently authorized revisions. It does not claim a fresh independent review occurred.
- Authorization: the maintainer approved this coding plan and subsequently requested TDD implementation. On 2026-10-27, the maintainer confirmed strict Lexicon rejection for malformed external edit timestamps. Commits, pushes and deployments remain unauthorized.
- Architecture: `atproto-craft-social-app-reference.md`, API architecture and wire-alignment specs, existing command/ingestion/notification boundaries, mandatory logging guide and project atproto-lexicon skill. Earlier product vision/discovery remain the product context.

## 2. Implementation Strategy

Reuse the current general/project composers, `PostRepository`, addressed PDS commands, Riverpod mutation controller and durable ingestion projections. Add a narrowly scoped owner edit seed and partial update contract. Keep raw unknown record data server-side and build the outgoing record from the current authoritative PDS record under an exact version precondition.

Add an optional portable `editedAt` record field, projected directly from the current record. Preserve original sorting and interaction identity. Establish original publication recipients before a source update overwrites the previous current record; edit notification processing retains that fixed original set rather than treating the edited record as another publication.

Important existing behaviors verified during planning:

1. `AddressedCommandService.putFenced` currently builds desired content before checking the expected CID and may accept equal content before that check. Edits require stricter first-attempt conflict behavior and exact-plan replay; opt in without changing other callers' existing semantics.
2. `PreparedCommand.CreatedAt` already provides a durable server-selected time. Use it for a successful logical edit, not `time.Now()` on each retry.
3. `IndigoPDSClient.ApplyWrites` discards its response into `any`, and `LatestCommit` returns only a CID. Preserve commit revision metadata through an additive capability for ordered edit reconciliation.
4. `postRecordProjection` currently includes only text/languages/sponsorship, and `applyPostRecordListOverlays` can insert a missing overlay at list position zero. Neither is sufficient for editing project/media fields without moving posts.
5. Durable source ingestion overwrites the current record and resets notification jobs. Freezing publication recipients only inside the later notification projector would be too late.
6. `notifications.Service.Activate` can reactivate a retracted event when source CID changes. An edit must not take that generic reactivation path.
7. The project draft adapter omits `patternSelfDrafted` from its saved-field list although the composer supports it. Follow TD-010 and the actual hydrator/payload, not that list alone.

No edit history, generic JSON Patch service, new OAuth handoff, new framework or unrelated draft repair is proposed.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Owner loading and writes | AppView handlers, typed identifiers, addressed commands | Owner-only edit seed and conditional partial update | BR-001, FR-001–FR-005, FR-009, FR-010, RULE-001–RULE-003, RULE-007, NFR-002 | AT-001, AT-002, AT-005–AT-008, UT-001–UT-004, UT-006, UT-008, UT-009, IT-001–IT-006, IT-008, IT-009, IT-016 |
| Edit metadata and projection | Current source/serving rows, generated Lexicon | Optional record timestamp, current edit state and revision-aware reads | BR-002, FR-006–FR-008, FR-011, FR-012, RULE-003, RULE-006 | AT-003, AT-004, AT-006, UT-003, UT-005, UT-007, IT-007, IT-010, IT-011, IT-014, REG-004–REG-006, REG-008 |
| Publication notifications | Transactional ingestion, deferred notification projector, outbox | Freeze original reason facts; update-only reconciliation; final dispatch recheck | RULE-004, RULE-005 | IT-012, IT-013, REG-007, REG-008 |
| Flutter editing and shared rendering | Existing composers, account leases, mutation controller, PostCard/PostSummary | Edit origin, existing media seeds, account-scoped edit notifier, accessible indicator | FR-001–FR-011, RULE-002, RULE-003, NFR-001 | AT-001–AT-008, UT-001–UT-009, IT-015, REG-001–REG-006, MAN-001, MAN-002 |
| Diagnostics and gates | Existing issue owners, safe contexts, local gates | Edit classifications and serialized sink assertions | NFR-003, NFR-002 | IT-016, IT-017, MAN-003 |

## 4. Files And Modules

Paths marked Create are proposed. Migration numbers and ADR number must use the next free values at implementation time. Test filenames from the test specification remain proposed unless already present.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `adr/<next>-published-post-editing.md` | Create later | Record optional timestamp, live quote semantics, current-only metadata and publication eligibility decisions before schema mutation | FR-006, FR-008, RULE-004, RULE-006 | IT-010–IT-014, REG-008 |
| `lexicon/social/craftsky/feed/post.json`; generated `appview/internal/lexicon/craftsky/` and schema output | Change later | Optional `editedAt` datetime; regenerate rather than hand-edit Go types | FR-006, FR-012, RULE-006 | UT-003, IT-010, IT-014 |
| `appview/migrations/<next>_post_editing.{up,down}.sql`; `appview/queries/` | Create/change later | Current edit state, immutable publication eligibility facts, necessary indexes and current-read joins | FR-006, FR-011, FR-012, RULE-003–RULE-006 | IT-001, IT-007, IT-010–IT-014, REG-004, REG-008 |
| `appview/internal/postrecord/edit.go` | Create | Raw-field preservation, supported change application, fixed-structure checks and content fingerprint excluding edit metadata | FR-002, FR-003, FR-005, RULE-003 | UT-001–UT-004, UT-008, IT-008, IT-009 |
| `appview/internal/api/post_edit.go`, `post_edit_request.go`, `post_edit_commands.go` | Create | Owner seed, strict input decode, validation, addressed command adaptation and response | FR-001–FR-006, FR-009, FR-010, RULE-001–RULE-003, RULE-007, NFR-002 | AT-001, AT-002, AT-005, AT-008, IT-001–IT-006, IT-008–IT-010, IT-016 |
| `appview/internal/api/post_store.go`, `post_response.go`, `post_response_shape.go`; read stores including search/saved-post queries | Change | Current metadata, revision and complete edit seed; quote/saved projections carry edited state with current visibility | FR-002, FR-006–FR-008, FR-011, FR-012 | AT-003, AT-004, IT-007, IT-011, IT-014, REG-005 |
| `appview/internal/pdscommands/addressed_command_service.go`, `store.go`, `pds_transport.go` | Change | Opt-in strict edit mode, prepared-time builder, exact-plan replay, atomic accepted-state hook and revision evidence | FR-005, FR-006, FR-009–FR-011, RULE-002, RULE-007 | IT-001, IT-003–IT-006, IT-009, IT-010, IT-016 |
| `appview/internal/auth/pds_client.go`, `pds_client_indigo.go`, `coordinated_pds_client.go` | Change | Additive typed head/write-result capability retaining returned revision; keep purpose-bound auth and old adapters intact | FR-010, FR-011, NFR-002 | IT-004, IT-005, IT-016 |
| `appview/internal/ingestion/store.go`, `reconciliation.go`, `post_publication.go` | Change/create | Capture original publication facts before coalescing; project current-record timestamp and exact revision without transition state | FR-006, FR-011, FR-012, RULE-004 | IT-007, IT-010, IT-012, IT-014 |
| `appview/internal/index/craftsky_post.go`, `transactional_projectors.go`, `post_notification_projection.go`, `image_safety_post.go` | Change | Stable sort fields, CID-bound metadata, current project/search data and no edit reactivation of notifications | FR-006, FR-008, FR-012, RULE-003–RULE-005 | IT-001, IT-007, IT-011–IT-014, REG-004, REG-007 |
| `appview/internal/notifications/post_publication.go`; `service.go`/`lifecycle.go` | Create/change | Original-reason facts and insert-once publication activation; separate current visibility/delivery reconciliation from activation | RULE-004, RULE-005, RULE-006 | IT-012, IT-013, REG-007, REG-008 |
| `appview/internal/push/dispatcher.go`; existing lifecycle/cleanup paths | Change | Current-source reason eligibility recheck and consistent per-source dispatch fence; cleanup of new current state | RULE-002, RULE-004–RULE-006 | IT-003, IT-012, IT-013, REG-007, REG-008 |
| `appview/internal/routes/routes_scheduled_post.go`, `routes.go`, `policy.go`; `appview/internal/app/` DI | Change | Wire edit handlers and command dependencies into existing complete capability matrices | RULE-002, FR-001, NFR-002 | IT-002, IT-003, IT-016 |
| `app/lib/feed/models/post_edit.dart`; `post.dart` and generated mapper outputs | Create/change | Edit seed/changes/result models and typed edit source/time/revision; map unknown data only on server | FR-002, FR-005–FR-008, FR-011 | UT-001, UT-005, UT-009, IT-015, REG-005 |
| `app/lib/feed/data/post_repository.dart`, `api_post_repository.dart`, `post_api_client.dart` | Change | `fetchEdit` and conditional `edit`; existing mutation response handling | FR-002, FR-004, FR-009–FR-011, NFR-002 | UT-009, IT-015, IT-016 |
| `app/lib/feed/providers/post_edit_seed_provider.dart`, `edit_post_provider.dart` | Create | Account-bound loading and mutation state, conflicts, immutable retries and accepted result | FR-002, FR-009–FR-011, RULE-002 | AT-005–AT-007, UT-007, IT-015 |
| `app/lib/feed/providers/post_record_overlay.dart`; shared mutation/reconciliation controller | Change | Replace-in-place edit overlays, complete controlled content and ordered reconciliation; preserve create/delete behavior | FR-011, RULE-003 | AT-006, UT-007, REG-001, REG-004–REG-006 |
| `app/lib/feed/composer/post_edit_origin.dart`, `post_edit_hydrator.dart`; existing image/video/link preview state | Create/change | Published edit origin, editable base/index identities, retained remote media and supported leaf changes | FR-002–FR-005, FR-009 | AT-002, AT-005, AT-008, UT-001–UT-004, UT-006, IT-008 |
| `app/lib/feed/widgets/post_composer_sheet.dart`; `app/lib/projects/widgets/project_composer_sheet.dart`; project hydrator/payload/submit adapter | Change | Reuse forms in edit mode; fixed context, correct CTA, no ordinary draft/schedule side effects | FR-001–FR-005, RULE-003, NFR-001 | AT-001, AT-002, AT-005–AT-008, REG-001–REG-003 |
| `app/lib/feed/widgets/post_card.dart`, new `post_edit_indicator.dart`; `app/lib/shared/widgets/post_summary.dart`; TD-011 callers | Change/create | Owner menu callback and shared accessible indicator; independent embedded-original metadata | FR-001, FR-007, FR-008, NFR-001 | AT-001, AT-003, AT-004, REG-005, MAN-001, MAN-002 |
| `app/lib/l10n/`; existing test keys and safe diagnostic classification locations | Change | Localized labels/time source/conflict/error states, test selectors and existing error owners | FR-007, FR-009, NFR-001, NFR-003 | AT-003, AT-005, IT-017, MAN-001 |
| Existing/new tests under `app/test/`, `appview/internal/`, `app/integration_test/` | Create/change during TDD only | Implement planned cases using current fakes and disposable DB helpers | All Must requirements | All AT/UT/IT/REG IDs; MAN-001–MAN-003 remain separate evidence |

## 5. Services, Interfaces, And Data Flow

### 5.1 Owner edit contract

Choose two additive endpoints, governed by existing `/v1/` routing, camelCase JSON and error envelope:

- `GET /v1/posts/{did}/{rkey}/edit`: owner-only, `Cache-Control: private, no-store`. Load the authoritative owner record from the PDS through the existing AppView session/lifecycle-fenced read capability; return private `post`, `baseCid`, and typed `editable` values without waiting for Tap. Include `recordRevision` only if exact record-revision evidence is available; never substitute a head revision. Keep current owner/account/post restrictions and strict record validation, with missing/dependency failures surfaced safely; never seed a partial/empty form. See section 13 for the approved refinement. Return no raw extension object, credentials or previous version.
- `PATCH /v1/posts/{did}/{rkey}`: requires `If-Match` with the concrete base CID, existing `Idempotency-Key`, session/device headers and owner lifecycle generation. Body is `{changes: ...}` containing only supported leaf changes; omitted means unchanged and explicit null removes an optional field. A complete editor form is not permission to drop unspecified record fields.
- `200` accepted result: `{changed, acceptedCid, editedAt?, recordRevision?, observationCid?, observationRevision?}`. `recordRevision` is ordered evidence for the accepted/verified version (see 5.4). `changed:false` returns unchanged acceptance metadata with no record mutation. `202 {status: "ambiguous"}` and `Retry-After` retain the existing command convention. Do not conflate uncertain PDS outcome with accepted-but-not-indexed state.
- Standard existing error envelope for validation, forbidden, unavailable and conflict. Missing or wildcard `If-Match` is rejected using the existing concrete-CID precondition policy. Use `409 pds_record_conflict` for stale/missing-at-save records; owner UI explains unavailable versus changed after reloading the edit seed. Missing initial load is `404`; malformed input is `400`; content validation uses existing `422` conventions; existing lifecycle/auth/rate-limit outcomes remain.
- Reject supplied identity, general/project classification, reply/quote context, provenance and editedAt fields, including relationship insertion/removal and CID-only changes. A supported project craft/details discriminator is editable within the project and is not this fixed general/project classification. Do not silently ignore a malicious protected-field request.
- Record identity and original creation/sort values always come from the server base. Current read APIs continue using AppView, not direct Flutter PDS reads. Authoritative PDS reads remain server-side inside owner editor loading and command preparation/reconciliation.

Partial interface sketches (names proposed; no production bodies):

```text
PostRepository.fetchEdit(Did, RecordKey) -> PostEditSeed
PostRepository.edit(Did, RecordKey, changes, expectedCid, operationKey)
  -> PostEditResult  // existing ambiguous/rate-limit response handling

PostEditSeed { post, baseCid, recordRevision?, editable }
PostEditResult { changed, acceptedCid, editedAt?, recordRevision?, observationCid?, observationRevision? }
PostEditChanges { presence-aware text/facets/langs/sponsored/images/embed/project changes }

AddressedPutCommandRequest:
  existing BuildRecord retained for current callers
  optional BuildPreparedRecord(AuthoritativeRecord, PreparedCommand) -> raw desired record
  opt-in StrictExpectedCID for post.edit
  original-create acceptance callback only; edits need no notification/provenance callback
```

### 5.2 Lossless supported changes and media

Keep `json.RawMessage`/`recordjson.Unmarshal` at the authoritative server boundary. Do not unmarshal a post into generated `FeedPost` and marshal it back as the entire replacement: generated types do not preserve all extension keys. Use generated types/schema for validation and inspect/apply narrowly scoped changes to the raw field map. Preserve exact timestamp strings and integer values in untouched extensions, including integers beyond JavaScript's safe range; Flutter never round-trips those unknown values.

- General scalar/facet/language changes use explicit presence flags. Regenerate affected UTF-8 facets when text changes; retain original facets when the corresponding text is untouched. Validate both grapheme and byte limits.
- Project changes mirror the existing supported field names from TD-010, mapped to `common`, pattern and the selected supported details variant. Nested omitted keys mean unchanged; null clears only optional supported leaves. Reuse existing project JSON codecs/ExtraFields and server validation where appropriate. Avoid a generic user-supplied JSON-pointer/patch language.
- For editable ordered image/material/additional-craft entries, include a bounded editor-only `existingIndex` identifying the element in the exact base version. Retained elements are copied from that authoritative base, supported changed leaves applied, and unknown siblings preserved. Missing index denotes a new validated entry. Reject duplicated/out-of-range source indices. Omitted elements in an explicitly supplied list are intentional removals; an omitted list is unchanged. These indices never go into PDS records.
- Switching a supported project craft variant is allowed while remaining a project; validate compatible known details and treat removed/replaced known variant content as an explicit change. Preserve unknown project/common siblings. For an unsupported details/embed variant, permit edits to other supported content while leaving that raw variant untouched; do not invent controls or strip it.
- Retained image/video/external blobs are referenced by existing record values. `editable` supplies supported metadata and secure existing display URLs, not original bytes. Extend composer media state with a retained published reference variant alongside local selection. Existing image inputs omit size in display projections, so build retained references from the owner seed, not `PostImage` alone.
- Reuse the approved new-media upload/verifier flow for replacements. A retained video does not require its old job ID or re-transcoding; a newly replaced video does. An explicit external-card removal also removes its thumbnail reference. Do not fetch old media just to save unchanged attachments.
- Build candidate content without editedAt first. Validate the complete merged record under existing rules, compare canonical content excluding editedAt, and return a no-op before assigning a new timestamp if unchanged. Preserve all fixed fields and full strong references.

### 5.3 Durable conditional write and recovery

Adapt existing `AddressedCommandService.Put`, scoped to operation kind `post.edit` and the existing owner/lifecycle boundary. Never pass expected CID `*`. Fingerprint normalized changes, target, expected CID, owner generation and blob references using the existing command fingerprint—not a retry-generated timestamp.

1. Prepare/replay the command using the stable operation key; `PreparedCommand.CreatedAt` is the server-selected edit time. Format UTC RFC3339 with the existing supported precision, consistently across DB, JSON and record.
2. For an undispatched strict edit, load authoritative PDS record; absence rejects, and CID mismatch rejects **before** content-equivalence acceptance/build. Existing non-edit callers keep their previous behavior through the opt-in branch.
3. Apply supported changes to that exact record, preserve unknowns/fixed data, validate and detect no-op. On a real change add `editedAt = prepared.CreatedAt` and persist the exact desired body in the existing dispatch plan before the remote effect.
4. Dispatch one `update` using the existing head-bound atomic write and record-CID condition; never delete/create, fall back to unconditional Put, or rebuild against a different base. Known repository-head conflicts may retry under the existing bounded policy only while the record still matches the expected CID.
5. An uncertain dispatch resumes by reading the **stored exact plan**, not by merging the request into whichever record is current now. If authoritative current record exactly matches that plan, confirm it; otherwise stay unresolved/read-only under existing command recovery. Never resend an ambiguous effect with a changed body or a new timestamp. A later external update cannot be silently overwritten.
6. Extend accepted finalization with a small transaction hook receiving record/revision evidence: complete the command, update current edit confirmation and restrict pending notification applicability atomically. Do not write feature metadata inside a callback that can run outside command finalization and then fail independently. No-op acceptance leaves edit state unchanged.
7. Keep existing command retention/tombstone policy. No new command-history endpoint or previous-body archive; only current metadata persists beyond ordinary operational retention.

The current service's equal-content-before-precondition behavior must have a regression test proving existing callers are unchanged and edits enforce FR-009. Replay equality must use stored desired content so unknown fields and prepared time remain stable.

### 5.4 Current-record timestamps and ordered reconciliation (approved 2026-10-27)

Keep optional public editedAt and stable command-prepared timestamps, no-op detection and strict schema validation. Serving projection copies editedAt from the exact current source record. Remove digest/transition state and timestamp provenance correlation. Keep authoritative record revisions and observation bounds for stale-read reconciliation; these are independent of timestamp provenance.

### 5.5 Fixed original publication recipients (approved 2026-10-27)

Preserve bounded original recipient facts from proven creation before coalescing. Edits cannot expand them or reset existing deliveries. Do not withdraw original mention recipients due to edits, intersect with current mentions, or serialize edit acceptance with push dispatch for that purpose. Retain independent owner, block, moderation, deletion and delivery controls. Unknown first edited snapshots cannot establish notification recipients.

### 5.6 Sorting, safety, persistence and cleanup

- In `craftsky_post.go`, stop resetting `profile_sort_at`/original sort anchor to `now()` on content update. Preserve original createdAt and first-seen chronological anchor for the edited post, including imported sort semantics and pagination clamps. If needed keep a separate first-indexed anchor rather than overloading last projection time; test future-createdAt records as well as ordinary posts.
- Keep all current project/tag/mention/language materialisations transactional with serving state. Do not mutate likes/reposts/replies/saves/pins as part of editing. Quotes continue resolving current visible original by URI; original strong references remain untouched.
- New media goes through existing CID-bound safety gates. Retained blobs reuse existing scan results under the existing scanner/policy identity; a stale clear result cannot clear a replacement CID. Accepted synthetic responses must not bypass safety by exposing newly uploaded unapproved media: use existing visibility policy or an accepted/pending presentation until cleared.
- Migrations add current metadata and publication facts only; do not backfill invented edit dates or notification candidates. Add indexes for source URI/current CID and recipient lookup as query plans require. SQL changes follow existing migration/sqlc conventions.
- Include new rows in explicit post/terminal-owner cleanup and retention paths. Operational reason facts follow existing account/moderation/privacy controls. Down migration removes only the new structures/fields; full migration rehearsal runs only in the disposable backend gate.

## 6. State, Providers, Controllers, Or DI

Backend DI: reuse `routes`' existing addressed command dependency and PostStore, adding a narrow `PostEditService` adapter for edit-specific builder/finalization. Pure raw-record logic remains in `postrecord`; storage/notification fact helpers must not import `api` or `index`, preventing cycles. Inject a clock, command store, existing media verifiers and observer through existing constructors.

Flutter graph (proposed Riverpod names):

```text
active account lease
  -> postRepositoryProvider -> PostApiClient -> existing session Dio
  -> postEditSeedProvider(account session, post URI) [autoDispose FutureProvider]
       -> PostEditOrigin(baseCid, recordRevision, supported base values, media indices)
       -> existing general/project composer, isolated composerId
  -> editPostProvider(account session, post URI) [autoDispose AsyncNotifier]
       -> pdsRecordOperationControllerProvider [existing application lifetime]
       -> existing immutable operation key/body, retry policy and account checks
       -> accepted current-post overlay + loaded-list notifications and bounded single-record reads
```

- `PostEditOrigin` is distinct from draft and scheduled origin; assert they cannot coexist. Capture `ActiveAccountLease` at open/load/save boundaries. Seed callbacks and async media completions must still match the captured lease and editor identity.
- Session state holds base supported values, immutable attempted payload/token, dirty fields, retained media, conflict/current seed and current save outcome. It is not persisted as an ordinary local draft. Cancellation uses existing unsaved-change protection with discard/stay; hide save-draft/schedule controls in published-edit mode.
- Reuse the shared submission lifecycle for progress/screen-awake/ownership. Give origin-specific callbacks no-op draft persistence/cleanup for published edits. Do not route through `createPostProvider`; call the edit notifier and preserve meaningful failures before any content-free UI failure wrapper.
- Use the existing mutation controller's scoped identity `post:<URI>` so delete/edit cannot run incompatible unresolved operations for the same owner post. Uncertain save freezes the attempted payload until resolved; further edits do not mutate an in-flight operation. A different deliberate edit after resolution gets a new operation key.
- Extend post reconciliation's controlled content to include all edited supported leaves/media/project information. For an edit, require accepted CID or verified later authoritative `recordRevision`; matching caption alone cannot clear a media/project overlay. Do not compare CIDs lexically or use client time as version order.
- Edit overlays replace existing rows in place. An immediate edit may inherit only the lists that actually displayed its pending create. Notify loaded lists in place, preserving cursor/depth; normal raw public reads reconcile versions. Do not invalidate whole accumulated collections. Filtering changes are reconciled by real read results, not by guessing feed positions.
- Preserve newer projected versions over an older accepted overlay; deleted/hidden/blocked results continue through current policy. Apply source-post overlays to embedded quotes and summaries only while visibility is confirmed; do not restore suppressed content from cached edit state.
- Account switch clears visible edit state and ignores late completions; it does not undo an already accepted owner write. Background retry ownership remains with the captured session, never the newly active account.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

- Add `onEdit` alongside existing owner actions on `PostCard`. Wire TD-011 callers to an `openPostEditor` coordinator that loads `postEditSeedProvider`, verifies ownership, then selects the existing project or general composer. No extra top-level navigation route or new editor layout is needed.
- Initial load has progress, retry and unavailable/denied feedback. Do not open a blank composer after seed failure or rely on an old feed card as the complete base.
- Prefill through shared/edit-specific hydration adapters that cover TD-010, including self-drafted patterns, non-default craft/status, duration, materials, optional fields, facets and retained attachments. Seed fixed reply/quote presentation separately from new-post target selection.
- Composer layout and controls stay familiar; CTA is localized `Save changes`. Disable during submission, expose existing validation feedback, and prevent general/project conversion and adding/removing/replacing reply/quote context. Existing ordinary creation/draft/scheduled entry points retain their CTA and behavior.
- Conflict UI retains the user's working values and offers loading/reviewing the current post. Do not replace the base CID behind the scenes. A deliberate resubmission occurs only after reviewing the new base; no merge editor or automatic overwrite is added.
- Add shared `PostEditIndicator` using the project's icon/Tooltip/time/localization/semantics widgets. A keyboard-focusable, touch-activatable icon exposes the current record date/time; no history navigation. Suppress for no known edit metadata.
- Extend `PostSummaryData` and its normal/quote/saved constructors with edit metadata, not just PostCard. Keep quoted-original metadata separate from the quoting author's metadata. Existing notification summaries without author/date metadata need no new unsolicited timestamp row.
- Route/call-site inventory is TD-011: home/feed/repost, profile posts/projects/reposts/comments, search/tag search, detail/thread/comment/reply, quote list/preview, saved summaries and project cards. Add Edit only to existing eligible owner post menus; do not invent a menu on an embedded preview or save-folder controls.
- Use the existing design system and mobile/web modal conventions; no desktop native builds. Add localized save/conflict/pending/load/error/time-source labels and stable test keys without exposing CID/protocol jargon in product copy.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Seed loading/failure | Progress/retry; no partial composer or overwrite from empty defaults | FR-002, FR-010 | AT-002, AT-006, IT-002, IT-015 |
| Invalid supported edit | Existing field validation and limits; retain input; no invalid PDS effect | FR-003, FR-010 | AT-008, UT-004 |
| No effective content change | Return existing result with changed=false; no timestamp/write/notification | FR-006 | AT-003, UT-003, IT-010 |
| Stale base / target deleted | Strict conflict, retain working input, fetch/review current seed; no recreation | FR-009, RULE-007 | AT-005, IT-004, IT-006 |
| Lost response / rate limit | Existing immutable keyed retry and Retry-After; exact-plan read-only recovery when ambiguous | FR-010, FR-011 | AT-006, IT-005 |
| Accepted, source/safety projection pending | Keep confirmed operation separate from visibility; avoid stale final rollback or revealing unapproved media | FR-011, RULE-005 | AT-006, IT-007, IT-013 |
| External absent/future/malformed metadata | Absent editedAt produces no indicator; schema-valid future declarations are displayed as declarations; malformed datetime or wrong-type declarations follow strict invalid-record rejection (confirmed 2026-10-27) | FR-006, FR-012 | IT-010, IT-014 |
| Retained video/no original job | Preserve raw existing ref and metadata; verify only replacement media | FR-003, FR-005 | AT-002, IT-008 |
| Unsupported nested fields/variant | Preserve server raw data; edit supported unrelated leaves only | FR-005 | UT-001, IT-009 |
| Account changes/lifecycle invalidated | Existing owner fences; no cross-account seed, UI completion or submission | RULE-002 | AT-007, IT-003 |
| New or removed mentions before initial delivery | Fixed original-recipient facts; edits do not withdraw pending original reasons | RULE-004 | IT-012, REG-007 |
| Old-CID quote / hidden original | Latest URI-hydrated original with indicator if visible; existing placeholders if unavailable/blocked | FR-008, RULE-005 | AT-004, IT-011, IT-013 |
| Diagnostics failure paths | Existing single issue owner, original causes/stacks and safe bounded refs; no private edit payloads/credentials | NFR-003 | IT-017 |

## 9. Test Implementation Plan

Implement named test cases in red-green-refactor slices; do not create a large passing-only test batch after implementation. Proposed files can be consolidated into existing adjacent suites if that improves fixture reuse without losing test IDs.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001, AT-001 | Proposed `appview/internal/api/post_edit_commands_test.go`; later edit-flow widget suites | Eligible owner, base record/engagement, stable prepared time | No supported edit command/route; cannot update same URI with no create effect |
| 2 | UT-001–UT-004, UT-008; IT-006, IT-009 | `postrecord/edit_test.go`, API request tests, existing raw JSON/project evolution fixtures | TD-002/TD-004/TD-010; unknown large integer, nullable clears and fixed context presence | Missing lossless supported-change builder/no-op/structural validation |
| 3 | IT-002–IT-005, IT-016; UT-009 | Routed edit HTTP, addressed command, auth transport and Dart API/repository suites | TD-006/TD-007; concurrent PDS write, deletion, lost response, exact-plan and same-key mismatch | No strict edit preconditions/commit metadata recovery/owner endpoint |
| 4 | IT-010, IT-014; UT-003 | Schema validation, migrated current-state and ingestion tests | T1/T2, duplicate/stale/metadata-only/first-seen/rebuild matrices | No direct current-record timestamp projection or ordered read fields |
| 5 | IT-007, IT-008, IT-011, IT-013; REG-004, REG-006, REG-008 | Post/index/search/saved/pin/image-safety/current-state suites | TD-003/TD-005/TD-009; future createdAt and coalesced ingestion | Sort movement, stale derived fields, missing media preservation/current quote state |
| 6 | IT-012; REG-007 | Ingestion publication facts, notification lifecycle and push dispatcher integration suites | TD-008; fixed original-recipient/delivery matrix cases, blocked dependencies and dispatch barriers | Edit reactivates events or first snapshot creates alerts for new recipients |
| 7 | AT-001, AT-002, AT-008; UT-001, UT-002, UT-006; REG-001–REG-003 | General/project edit prefill/flow widgets and existing composer/draft/schedule suites | TD-010 all supported fields and retained media, unrelated ordinary draft | No Edit menu/prefilled mode/Save changes, accidental create/draft/schedule effects |
| 8 | AT-005–AT-007; UT-007 | Edit conflict/submission/account widget/provider suites and shared mutation tests | Deferred seed/read/save completions, second session and V3 after accepted V2 | Silent overwrite, cross-account completion, partial content reconciliation or edit inserted at list head |
| 9 | AT-003, AT-004; UT-005; REG-005 | Indicator/PostCard/PostSummary and route fixture tests | TD-011 edited/unedited/owner/non-owner, time zone/locale, old-CID quoted original | Missing/incorrect tooltip semantics or embedded edit metadata |
| 10 | IT-017 | Existing Go testlog/Flutter reporter sink tests extended for edits | TD-009 protected canaries, original cause/stack, captured safe output | Lost cause, duplicated issue owner or protected payload leakage |
| 11 | IT-015 | Extend `app/integration_test/critical_journeys_test.dart` loopback fixture | iOS/Android running app, edit/save/conflict/cancel | Complete UI-to-update socket path missing |
| 12 | MAN-001–MAN-003 | Platform/browser evidence and optional separately authorized disposable PDS | Real accessibility/picker and propagation checks | Record as unexecuted until actually performed; no mock-based claim |

Every AT/UT/IT/REG/MAN ID from the specification appears above. TD-010/TD-011 define concrete field/entry-point coverage; retain their case names in tests for traceability. Test schema evolution as optional-only and regenerate deterministically. Mocked PDS content must satisfy actual hashing/CID/identifier validators where those paths are exercised.

Focused implementation commands (not run in planning):

- First backend slice: from the root, use the existing local compose stack and `just test` until the checked-in wrapper's local-service environment has been supplied securely to a focused invocation. Then, from `appview/`, use `TEST_DATABASE_REQUIRED=true go test -count=1 -race ./internal/api -run '^TestEditPostUpdatesExistingRecord$'` for the proposed first test. Require an executed test, not a missing-test pass or a skipped PostgreSQL fixture. The current wrapper has no package-filter argument; do not pretend it sets environment in the parent shell. Do not guess or print a database URL. An incomplete unit-only run is not backend gate evidence.
- First Flutter slice: root `fvm exec just app-test test/feed/widgets/post_edit_flow_test.dart` after the proposed file exists. Run `fvm exec just app-analyze` separately; app-test is tests plus host Rive setup, not analysis. Direct FVM Flutter/Dart commands run from `app/`.
- Broader backend completion: root `just appview-check` with disposable isolated services, including migration up/down and required normal/race artifacts. Do not point migration rehearsal at developer/shared/production data.
- Generation after approved schema/model changes: root `just lexgen` and `just lexgen-check`; use the app's current Dart mapper/Riverpod build_runner generation convention from `app/`, and check generated diffs. No dependencies are added just for this feature.
- Native loopback journey: root `just app-test-integration <device-id>` for iOS/Android. Web does not use that dart:io runner; retain MAN-001/MAN-002 browser checks unless an existing suitable browser harness is available at implementation time.

## 10. Sequencing And Guardrails

- First TDD step: write the narrow owner-caption-edit integration case IT-001 with fixed URI/expected CID and an asserted absence of create calls. Add only the missing boundary needed to get a meaningful failing test. Follow immediately with no-op, strict stale precondition and unknown-field preservation cases before broader editor wiring.
- Design prerequisites: approve the intended metadata/recipient behavior and this plan; author the required ADR before optional Lexicon changes. Generation/migration work is future implementation work, not completed by this document.
- Dependencies: lossless builder and strict/replayable command semantics → portable/current metadata and revision evidence → source/publication transition capture and safety/read projections → Flutter model/repository/state → prefill/menus/indicator → regressions and runtime evidence. Notifications/ordering/safety are required completion work, not optional follow-ups after exposing Edit.
- DR-001 response: section 5.4 selects optional current-record editedAt and ordered commit evidence; wire fields and restart/rebuild handling are explicit.
- DR-002 response: section 5.5 selects original publication facts captured before coalescing, insert-once activation, current reason intersection and final fenced dispatch cutoff.
- DR-003 response: sections 5.2/6/7 use TD-010/TD-011, including patternSelfDrafted and fixed relationship presence/CID. Retain numbered tests; no full Cartesian matrix or general patch framework.
- DR-004 response: test and analysis commands are separate in section 9.
- Guardrails: typed DID/URI/key/CID/revision at boundaries; current JSON casing/errors; owner/lifecycle gates; no credentials in Flutter/logs; current-version safety; no PDS deletes or generic record write API. Do not use profile/notification refactors to expand scope.
- Out of scope: edit history, preserved old media versions, new progress announcements, scheduled edits, general/project conversion, conversation retargeting, new edit alerts, durable local edit drafts, generic merge UI, desktop-native targets and production actions.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved 2026-10-27 | Maintainer approved simplifications 1–5: single timestamp label, fixed original recipients, confirmation-only save, shared orchestration and normal previews | Supersedes prior proposal | Verify the revised implementation; no further product decision pending |
| CPQ-002 | Implementation prerequisite | Optional public editedAt requires an ADR, schema generation and project Lexicon workflow | Unreviewed schema evolution would violate repository rules | atproto-lexicon skill read/applied for this plan. The separately named `writing-architecture-decision-records` skill was not found in the checked project/skill roots; locate an applicable copy or resolve that workflow prerequisite before authoring the ADR/schema. It does not block writing this plan |
| CPQ-003 | Non-blocking design risk | Some PDS write adapters may omit revision/atomic-write capability | Cannot safely claim ordered reconciliation from CIDs alone | Additive typed adapters preserve available commit metadata; use stable authoritative recovery bracket and fail unresolved safely when evidence is unavailable. Test capability failures explicitly; no unconditional write fallback |
| CPQ-004 | Non-blocking design risk | Long provider calls and multiple lock domains can cause source-dispatch contention | Removal/send race or deadlocks | Fixed owner→post→delivery order, bounded provider timeout, all-exit cleanup and deterministic concurrency/race tests; no network call inside source projection transaction |
| CPQ-005 | Non-blocking design risk | Reuse of lossy draft fields or partial mutation projections can drop content or clear overlays early | Silent data loss or stale final UI | Server-only raw base, TD-010 complete inventory, retained entry indices and exact/revision-aware reconciliation |
| CPQ-006 | Non-blocking limitation | External clients can mutate their own PDS data outside CraftSky's editor restrictions | CraftSky cannot enforce editor policy on arbitrary third-party writes | API enforces fixed context/type; ingestion continues existing semantic validation/visibility and stable AppView sorting. No claim to prevent owners writing directly elsewhere or guarantee old external copies disappear |
| CPQ-007 | Non-blocking validation boundary | Local tests cannot prove live PDS timing, real push, screen-reader or picker behavior | Overstated release confidence | Keep MAN-001–MAN-003 and actual platform evidence separate from hermetic automated passes |

No unresolved implementation-shape question prevents starting the specified TDD work after plan approval and the applicable ADR prerequisite. The implementation must prove safe concurrency and recovery rather than simplifying away the prescribed cases. This document does not approve its own implementation.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-23-edit-posts/04-coding-plan.md`.
- TDD execution plan: future `05-implementation-plan.md` in this folder, following `implement-tdd` after the maintainer requests implementation.
- Start with test: IT-001 / AT-001 owner caption edit with same URI/base CID and no create effect; next strict stale/no-op/preservation tests.
- Focused command: root `just test` under the existing compose test setup for the first backend slice; proposed Flutter `fvm exec just app-test test/feed/widgets/post_edit_flow_test.dart` plus separate analysis when that slice is introduced.
- Notes: preserve requirement/AC/test IDs, resolve the ADR skill prerequisite before schema changes, and keep CPQ-001's proposed policy details visible in approval. Existing source, tests, generated output, migrations and earlier workflow documents are unchanged in this stage.
- Status: plan written only; no runtime tests, generators, migrations, commits, pushes or live actions performed. Leave uncommitted. Next maintainer choice: implement with TDD, add a manual note, or stop for review.

## 13. Approved immediate-edit refinement, 2026-10-27

The maintainer explicitly approved loading the owner editor through AppView from the authoritative PDS record, without a Tap prerequisite. This supersedes section 5.1's indexed-seed selection and any initial-load/source-pending gate in this plan. Ordinary public reads and accepted-result hydration remain projection/safety-gated.

- Existing owner-only GET retains its URL, no-store policy, account/write eligibility and session/lifecycle fences. Inject a narrow post-record read capability implemented with the existing fenced `pdseffects.ReadRecord`; no raw PDS factory is exposed to routes and no credentials or direct PDS reads reach Flutter.
- Validate the authoritative record using the existing schema/semantic validator. Build the private owner prefill from that same raw record/CID, preserving all supported fields and existing attachment refs. Keep raw extensions server-side. A record revision absent from getRecord is omitted, never fabricated from repository head time. Exact-CID save preconditions remain authoritative.
- Local membership/lifecycle and current post/account moderation restrictions apply independently of post/source-row existence. Private editor previews may show retained media before a scan is complete; known unsafe matches/restrictions remain denied. Public reads, quotes/reposts and accepted overlays retain their existing safety gates.
- Remove the indexed-source prerequisite from save preparation; preserve all authoritative-CID, immutable retry, deletion and new-media verification rules. Original publication facts already seeded by accepted create remain the notification source before Tap arrives.
- Accepted-result refresh must use a normal public AppView post read, not the private authoritative editor seed. An owner prefill is not evidence of public visibility or projection convergence.
- Cover immediate general/project/media load and save, authoritative content ahead of Tap, absent/malformed records, ownership/session/lifecycle denial, moderation/matched-media denial, failures with safe original diagnostics, lossless fields and public-refresh isolation. No schema or migration change is planned.

## 14. Approved confirmed-save navigation refinement, 2026-10-27

The maintainer explicitly requests that a PDS-confirmed edit immediately dismiss the composer to its originating route and optimistically update loaded representations. This supersedes the composer-held accepted/pending refresh flow. FR-011/AC-014 and AT-001/AT-006/UT-007 apply.

- Definite accepted responses close both general and project editors without waiting for another AppView read. Ambiguous/conflicting/failed saves remain in the composer with recoverable input and existing exact-request retries.
- Build the optimistic presentation from the frozen supported editor seed and submitted changes, stamped only with returned accepted CID/edit metadata. It is local presentation, never raw extension storage or proof that public indexing/safety has completed.
- Register the existing URI-scoped/account-scoped edit overlay before closing. Replace loaded visible rows/quotes/reposts without inserting a post into unrelated feeds, changing engagement, overriding protected responses or altering fixed context/order. Matching or demonstrably newer AppView versions reconcile it. Confirmed content must not revert merely because the normal transient overlay timeout elapsed.
- No API, Lexicon, migration, direct client PDS reads or new write authority is introduced. New/retained media and project leaves must render coherently from the same frozen intent; no accepted command is resent to wait for projection.

## 15. Approved simplifications 1–5, 2026-10-27

This section supersedes earlier incompatible implementation details: use current-record editedAt with a single tooltip; no timestamp-source field/state/correlation. Return acceptance metadata only, remove public accepted-result hydration/ReadEditSnapshot and unused client refreshAccepted/waiting messages. Keep exact CID/revision evidence, immediate owner load and optimistic loaded-list behavior. Freeze original publication recipients without edit-driven withdrawals. Extract common published-edit submission/video-recovery/retry completion orchestration, retaining composer-specific patch construction. Replace bespoke manual link-card metadata editing with normal preview retention/removal/replacement. Do not relax field preservation or add a general framework. Existing migrations remain immutable; schema cleanup is additive. Review must use this revised scope.
