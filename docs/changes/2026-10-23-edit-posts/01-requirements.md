# Requirements: Edit Published Posts

## 1. Initial Request

Users want to edit published CraftSky posts. The maintainer requested exploration of AT Protocol constraints before requirements, followed by a comparison of other platforms' editing UX. The confirmed direction is editing at any time through the post context menu, using the familiar composer prefilled with existing information and a different save action. Edited content displays an edit icon with a timestamp tooltip. Reposts and quote-reposts display the latest content. No edit history is wanted. General posts and project posts cannot be converted into each other.

This document specifies published-post editing. The maintainer authorized implementation and the five simplifications on 2026-10-27; production actions remain separate.

## 2. Discovery Baseline (before implementation)

- Relevant files (paths relative to the repository root):
  - `lexicon/social/craftsky/feed/post.json`: general posts, project posts, replies, images, external/video/quote embeds, languages, sponsorship and provenance share one record type. There is no edit timestamp in the current schema. Quote and reply references contain a URI and CID.
  - `lexicon/social/craftsky/feed/like.json` and `repost.json`: interactions also contain URI/CID strong references.
  - `app/lib/feed/widgets/post_composer_sheet.dart` and `app/lib/projects/widgets/project_composer_sheet.dart`: existing composers integrate local-draft restoration, media, scheduled publishing and account ownership.
  - `app/lib/feed/composer/composer_submission_coordinator.dart`: shared submission coordination.
  - `appview/internal/api/post_create.go`, `post_request.go`, `post_response.go`: existing publication validation and post presentation.
  - `appview/internal/routes/routes_scheduled_post.go`: published-post creation, reads, interactions and deletion are registered; no published-post edit route exists.
  - `appview/internal/pdscommands/addressed_command_service.go`: addressed updates carry an expected CID, build from the authoritative record and use existing command lifecycle handling.
  - `appview/internal/auth/pds_client_indigo.go`: conditional PDS updates via `swapRecord` are supported.
  - `appview/internal/index/craftsky_post.go`: create and update events upsert the current post and refresh project materialisation, tags and mentions.
  - `appview/internal/index/set_source_projector.go`: like/repost aggregates are scoped by subject URI.
  - `appview/internal/api/post_conversation_store.go`: comment relationships use parent/root URIs.
  - `appview/internal/api/post_engagement_store.go`: quote hydration loads the latest visible post by URI, without requiring the quoted CID to equal the current CID.
  - `appview/internal/index/image_safety_post.go` and `post_notification_projection.go`: safety and notification projections are version-sensitive and run on updates.
- Existing patterns: AppView-mediated writes, owner/account fencing, conditional writes, durable command outcomes and firehose projection. Existing composers provide the interaction model to reuse, not a mandate to reuse creation side effects.
- Current behavior: posts can be created and deleted through the app; incoming record updates can change indexed content. Supported author editing, edit indicators and public history are not currently provided.
- Constraints discovered: updating the same record preserves its URI but changes its CID; other users' strong references remain unchanged. Existing interactions can continue to follow the URI. A CID does not guarantee retrievable history. A successful PDS update and visible AppView projection are separate states.
- Architecture references: `atproto-craft-social-app-reference.md`; `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`; `docs/development/logging-error-reporting.md`; [product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit).
- Protocol references checked during discovery: [putRecord and conditional updates](https://raw.githubusercontent.com/bluesky-social/atproto/main/lexicons/com/atproto/repo/putRecord.json), [AT URI identity](https://atproto.com/specs/at-uri-scheme), [strong references](https://raw.githubusercontent.com/bluesky-social/atproto/main/lexicons/com/atproto/repo/strongRef.json), [repository state and diffs](https://atproto.com/specs/repository), [media lifecycle](https://atproto.com/specs/blob), [update distribution](https://atproto.com/specs/sync).
- Test/build commands discovered: `just app-test` (Rive host setup and Flutter tests), `just app-analyze` (separate static analysis; direct FVM Flutter commands run from `app/`), `just test` (host Go tests against compose Postgres), and `just appview-check` (full backend gate). Run `just lexgen` and the repository's Lexicon checks if a schema changes. Device/browser validation must cover iOS, Android and web. No tests were run during requirements authoring.

## 3. Clarifying Questions And Decisions

### Q1: What can be edited, and when?

Answer: "anything can be edited at any time."

Decision / implication: all author-editable content supported for the existing post type is editable, without a product time window or edit-count cap. Authorship, record identity, original publication time and conversation relationships stay fixed. Existing validation and account policies still apply.

### Q2: What should the editing flow look like?

Answer: context menu entry; like editing a draft, with all information prefilled; the CTA and save logic differ.

Decision / implication: use the corresponding familiar composer and a `Save changes` CTA. Saving updates the published record rather than creating a post, scheduling publication or consuming an unrelated draft.

### Q3: How should edits be disclosed?

Answer: an edit icon with the timestamp as a tooltip; no edit history. The maintainer agreed mobile users must also be able to expose the timestamp.

Decision / implication: show the most recent edit time with an accessible indicator; no revision browser or history storage introduced for this feature.

### Q4: What happens to reposts and quote-reposts?

Answer: display the latest version. The maintainer accepted that surrounding commentary can refer to earlier content.

Decision / implication: retain existing interaction records and resolve the original post's current visible content. Do not require an old CID to match before displaying a quote, and do not add a historical-version placeholder merely because the content changed.

### Q5: May general posts become project posts, or vice versa?

Answer: explicitly no.

Decision / implication: post type remains fixed throughout editing.

### Q6: Are the discovery safeguards accepted?

Answer: "Yes agreed with all your explicit points."

Decision / implication: retain URL, engagement and conversation; preserve unknown fields and existing media; handle conflicts without losing work; preserve feed position; do not republish or resend ordinary notifications; maintain moderation controls. Operational and moderation evidence retention remains separate from public edit history.

## 4. Candidate Approaches

### Option A: Update the existing record, showing its latest content — selected

Summary: preserve post identity and relationships, replace author-editable content conditionally, and expose the latest edit time.

Pros: matches the approved UX; preserves engagement and links; fits current URI-based reads and update indexing; no history feature.

Cons: quotes and comments can discuss content that is subsequently changed.

Risks: lost updates, stale app views, accidental field loss, notification replay and moderation bypass unless handled explicitly.

### Option B: Retain immutable revisions with public history — rejected

Summary: retain earlier versions and let readers inspect or resolve them.

Pros: preserves the exact context of earlier interactions.

Cons: contradicts the decision not to maintain edit history; adds retention, portability and media complexity.

Risks: historical sensitive content remains accessible and requires its own moderation policy.

### Option C: Publish a replacement post — rejected

Summary: create another post and link it to the original.

Pros: keeps the original discussion attached to unchanged content.

Cons: changes post identity and fragments engagement; does not meet the requested editing experience.

Risks: duplicates, broken links and confusing conversations.

## 5. Recommended Direction

Recommended approach: Option A, as confirmed by the maintainer. Edit the existing published record through AppView, preserve its identity and structural relationships, and serve its latest approved content everywhere it appears. Use the familiar composer with complete prefill and `Save changes`. Show only the latest edit timestamp, with no public revision history.

Why: this meets the requested UX, preserves the social conversation and fits existing AT Protocol and CraftSky update mechanisms. The product accepts that a quote is a live view, not a historical snapshot.

## 6. Problem / Opportunity

Authors need to correct mistakes, improve accessibility, update project information and change attachments without deleting their post or losing its conversation and engagement. Project information can need correction long after publication. Readers need a clear but unobtrusive indication that the current content has been edited.

## 7. Goals

- G-001: let authors maintain their published content at any age.
- G-002: make editing familiar through complete prefill and existing composer controls.
- G-003: preserve identity, relationships, engagement and chronological placement.
- G-004: show latest visible content consistently, with a discoverable edit time.
- G-005: prevent lost updates, data loss, unauthorized changes and moderation bypass.

## 8. Non-Goals

- NG-001: public or author-facing edit history, revision records, diffs, undo-to-previous-version or historical media archives.
- NG-002: converting general posts into project posts or project posts into general posts.
- NG-003: changing authorship, publication time, record identity, reply parent/root or quoted target.
- NG-004: announcing project progress, bumping a post in feeds, republishing or scheduling an edit.
- NG-005: rewriting other users' likes, reposts, replies or quote records.
- NG-006: guaranteeing immediate propagation or removal of previous public copies across third-party services.
- NG-007: redesigning creation, local-draft or scheduled-post workflows; introducing durable edit drafts or a merge editor.
- NG-008: weakening moderation evidence retention or adding administrator editing of another author's post.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Author | Authenticated owner of a published post | Correct and maintain content without losing engagement or work |
| Reader | Person viewing a post, reply, repost or quote | Latest visible content and an accessible edit timestamp |
| Reposter / quoting author | Person who previously shared the post | Existing interaction remains attached to the original identity |
| Moderator | Existing moderation operator | Existing reports, evidence and restrictions remain meaningful |
| External client | Another client editing the owner's PDS record | Valid updates converge through the existing ingestion boundary |

## 10. Discovery Behavior (before implementation)

Flutter provides creation composers and private draft/scheduled-post editing. Published-post editing has no supported app flow or API route. The backend can already ingest post updates and replace current materialised content. Engagement and conversation queries largely use stable post URIs; quote previews hydrate the latest visible target. No published edit timestamp is currently specified.

## 11. Desired Behavior

An eligible author chooses `Edit` from their post's context menu on any existing post surface. The corresponding composer opens with the complete existing content and fixed reply/quote context. The author changes permitted fields and selects `Save changes`. The system validates the change and conditionally updates the same published record. The app reconciles the result without creating another post or showing older cached content as the final result. Readers see the latest visible content and an edit icon exposing the most recent edit time. Cancelling leaves the published post unchanged. Conflict and failure states retain the author's unsaved work.

## 12. Requirements

### Timestamp interpretation (approved simplification, 2026-10-27)

CraftSky assigns the public current-record `editedAt` on a successful content change. Failed, unchanged and retried accepted operations do not advance it. The icon and localized `Edited <date/time>` tooltip display that value without claiming verified provenance. Valid external records display their declared `editedAt`, including future declarations; external records without it have no indicator even if a content change was observed. Malformed declarations remain subject to strict Lexicon validation. Do not infer edit times from observation, backfill or repository revisions. Current record metadata survives normal projection/rebuild without separate transition/provenance storage.

### Publication notifications versus edits (approved simplification, 2026-10-27)

Original publication establishes a fixed recipient set. Edits must not add recipients, replay deliveries, change activity time or reset unread state. Original pending deliveries may still be sent after an edit removes a mention; edits do not recall or retract original notifications. Existing deletion, moderation, blocking and delivery eligibility policies still apply. Unknown original eligibility cannot be inferred from an edited snapshot. Retain bounded original publication facts, not prior bodies or an edit history.

### Link previews and acceptance (approved simplification, 2026-10-27)

Existing link cards may be retained, removed or replaced through the normal link-preview flow. Titles, descriptions and thumbnails are generated from the linked page; no manual metadata/thumbnail editor is required. Unchanged cards retain their original data.

A save returns acceptance metadata only, independent of indexed post availability. The app dismisses after definite acceptance, projects frozen submitted content, and reconciles through normal public reads. There is no accepted-but-waiting composer or manual accepted-result refresh action. The two composers share submission orchestration; their field preparation remains separate.

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Authors can correct and maintain published posts without losing their identity or accumulated conversation and engagement. | Core user need | U1, U2 | AC-001, AC-005 |
| BR-002 | Business | Must | Readers can identify edited content and discover when its latest edit occurred, without a revision-history feature. | Transparent, unobtrusive editing | U2 | AC-006, AC-007 |
| FR-001 | Functional | Must | Offer `Edit` in the eligible owner's existing post context menus wherever those menus appear, including replies and quote-posts. | Consistent entry point | U2, A/ASM-001 | AC-001, AC-002 |
| FR-002 | Functional | Must | Open the corresponding general/project composer prefilled with every existing value supported for that post, including caption, rich-text annotations, project fields, sponsorship, languages, attachments, alt text and supported embed metadata. Display fixed reply/quote context. Permit opening and saving immediately after successful publication without waiting for Tap ingestion or public projection; AppView loads the authoritative owner record server-side. | Faithful draft-like editing | U2, U3 | AC-002, AC-003 |
| FR-003 | Functional | Must | Permit changes and explicit clearing/removal of all author-editable fields supported for the existing post type, subject to existing content/media validation. This includes supported image/video changes and normal link-preview replacement/removal, not changes to a quote's target. | "Anything" editable within fixed structure | U2, U3, U4 | AC-003, AC-004 |
| FR-004 | Functional | Must | Use `Save changes` for editing. Saving updates the existing published post; creation, scheduling and unrelated draft side effects must not run. Cancelling must not change the published post. | Distinguish editing from publication | U2, U3 | AC-001, AC-008 |
| FR-005 | Functional | Must | Preserve fields outside the editor's supported changes, including unknown extension data and provenance. Retain existing media references without re-uploading unchanged files. | Avoid silent record and media loss | U3, D | AC-003, AC-009 |
| FR-006 | Functional | Must | Record and display the latest successful content-edit time. Advance it on each subsequent content edit; do not mark an unchanged submission as a new edit. Previously unedited posts have no edit indicator. Use the current-record timestamp contract in section 12. | Accurate current edit state | U2, A/ASM-002 | AC-006, AC-010 |
| FR-007 | Functional | Must | Show the edit icon wherever a post's own metadata is rendered, including detail, feed/profile/search/saved-post presentations, comments, reposts and quoted previews. Expose a localized date/time by hover tooltip, keyboard focus, mobile tap and assistive technology. | Equivalent access on all platforms | U2, U3 | AC-006, AC-007 |
| FR-008 | Functional | Must | Reposts and quote-reposts resolve the original post's latest visible content and edited state while preserving their own author text, identity and timestamps. An old referenced CID alone must not make current content unavailable. | Confirmed live-view semantics | U2, U3, C | AC-011 |
| FR-009 | Functional | Must | Reject stale saves when the current post version differs from the editor's base version. Explain the conflict and retain unsaved input so the author can review current content before deliberately submitting against it. Do not silently overwrite or merge. | Prevent lost updates | U3, D | AC-012 |
| FR-010 | Functional | Must | Handle loading, validation failure, media failure, write failure and uncertain submission outcomes without falsely confirming success or discarding unsaved input. Repeated save/retry must not create a second post or apply a different timestamp for the same accepted operation. | Reliable editing | U3, C | AC-004, AC-013 |
| FR-011 | Functional | Must | On PDS-confirmed success, dismiss the composer to its originating screen and optimistically update loaded app representations without waiting for indexing. Reconcile subsequent reads without replacing confirmed edited content with stale cached content as a final state. | PDS and AppView converge asynchronously | U3, D | AC-014 |
| FR-012 | Functional | Must | Refresh derived project, tag, mention, language and discovery data after edits. Incoming valid external-client updates converge under the same visibility rules and expose the edit indicator only when current-record editedAt is declared. | Correct reads beyond the main card | U3, C, D | AC-015, AC-019 |
| RULE-001 | Business rule | Must | No feature-specific edit-age window or edit-count cap applies. Existing account eligibility, validation and service rate limits remain applicable. | Unlimited maintenance without bypassing policy | U2, U3 | AC-001, AC-016 |
| RULE-002 | Business rule | Must | Only the authenticated post owner may edit, under existing account/write eligibility and lifecycle rules. Account changes must not submit or expose another account's edit session. | Existing ownership boundary | U3, C | AC-016, AC-017 |
| RULE-003 | Business rule | Must | Keep author DID, collection/key/URI, original publication time, general/project type, and the presence and full references of reply parent/root and quote target fixed. Preserve original feed/profile ordering and existing valid pins. | Same post and conversation | U3, U4 | AC-005, AC-018 |
| RULE-004 | Business rule | Must | Edits must not send new social notifications or replay existing notifications, including reply, quote, repost and mention notifications. Keep the original recipient set fixed under section 12 even when initial work is pending. | Editing is not republishing | U3, A/ASM-003 | AC-020 |
| RULE-005 | Business rule | Must | Apply existing moderation, blocking and media safety controls to the current version and all quoted/reposted views. Editing must not clear applicable enforcement, discard report evidence or expose unapproved replacement media. | Preserve safety boundaries | U3, C | AC-021 |
| RULE-006 | Business rule | Must | Do not introduce storage of previous post bodies/media for an edit-history feature or expose a history UI/API. Retain only latest edit metadata for this feature; existing operational and moderation retention continues. | Explicit no-history decision | U2, U3 | AC-022 |
| RULE-007 | Business rule | Must | If the target is deleted before save, reject the edit without recreating it or changing existing replies/reposts/quotes. | An edit cannot become a create | U3, D | AC-023 |
| NFR-001 | Non-functional | Must | Provide equivalent edit, save, cancellation, error and timestamp access on iOS, Android and web, with localized text and accessible controls. | Supported platforms and accessibility | U3, C | AC-007, AC-024 |
| NFR-002 | Non-functional | Must | Keep record writes and OAuth credentials server-side under existing AppView/PDS boundaries, with existing approved media-upload exceptions only. Preserve the API's `/v1/`, camelCase JSON and error-envelope conventions. | Repository architecture contracts | C, D | AC-025 |
| NFR-003 | Non-functional | Must | Report edit failures through existing diagnostic owners with bounded safe operation context and original causes, without logging composer payloads, credentials or retained historical bodies. | Existing mandatory logging policy | C | AC-026 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-004, RULE-001 | Given eligible owners of recent and old published general/project posts, when they choose Edit, change content and save repeatedly as separate edits, then each save updates the same post with no product age/count restriction and no duplicate publication. |
| AC-002 | FR-001, FR-002 | Given owner posts displayed on each surface with an existing context menu, including replies/quote-posts, when Edit is selected, then the correct composer opens with complete current supported content and fixed context; non-owner menus do not offer editing. |
| AC-003 | FR-002, FR-003, FR-005 | Given posts exercising supported optional/project fields, rich text, media and embed variants, when opened and edited, then values are faithfully prefilled, intentional changes/removals persist, untouched values remain and existing attachments need no re-upload. |
| AC-004 | FR-003, FR-010 | Given edits exceeding existing text/media/project limits or producing invalid required content, when saved, then existing validation is enforced with actionable errors, unsaved input is retained and the published record is not replaced with invalid content. |
| AC-005 | BR-001, RULE-003 | Given a post with likes, comments, reposts, saves and a valid profile pin, when edited, then its URI/key, author, original publication time, conversation relationships, engagement, saves, valid pin and chronological placement are retained. |
| AC-006 | BR-002, FR-006, FR-007 | Given an unedited post, then no edit icon appears; after a successful content edit, its presentations show an edit icon whose tooltip gives the latest edit date/time; a later edit replaces that timestamp without exposing earlier versions. |
| AC-007 | BR-002, FR-007, NFR-001 | Given mouse, keyboard, touch and screen-reader users, when they interact with the edit indicator, then the localized latest edit date/time is available through hover, focus, mobile tap and accessible text, without requiring hover on touch devices. |
| AC-008 | FR-004 | Given a published edit session, when cancelled or closed, then published content remains unchanged; when saved, no new post, scheduled item or unrelated local-draft mutation results. Existing create/draft/schedule flows continue to work independently. |
| AC-009 | FR-005 | Given unknown record fields, provenance and unsupported optional nested data, when a supported field is edited, then unrelated data survives and unchanged media references remain identical. |
| AC-010 | FR-006 | Given a post opened without changes, or changed then restored to its initial values, when submission is attempted, then it does not create a new content edit, change the latest edit timestamp or add an edit icon. |
| AC-011 | FR-008 | Given existing reposts and quote-reposts referencing an earlier CID, when the original is edited and current reads converge, then each displays its latest visible content and edit indicator; quoting commentary and share metadata remain unchanged and no historical-version warning/placeholder is added solely for the CID mismatch. |
| AC-012 | FR-009 | Given the same post opened on two clients, when one saves and the other submits its older base version, then the second save is rejected as a conflict, preserves input and offers review of current content before a deliberate resubmission; the newer record is not overwritten silently. |
| AC-013 | FR-010 | Given an edit encountering load/upload/write errors, a timeout after a possible write, repeated taps or a retry after acceptance, then no false success is shown, recoverable unsaved input is retained, and the operation resolves without duplicate publication or repeated application of an accepted edit. |
| AC-014 | FR-011 | Given an accepted edit before its indexing completes, when the author revisits loaded instances or refreshes, then pending/accepted state is handled coherently; after convergence all relevant app reads show current content without a stale final rollback. |
| AC-015 | FR-012 | Given changes to project details, tags, mentions or languages, when indexing completes, then project detail, search/filter results and applicable visibility predicates reflect the edited record and removed values do not persist as stale derived data. |
| AC-016 | RULE-001, RULE-002 | Given unauthorized callers or accounts barred from existing write operations, when an edit is attempted, then it is denied under existing policy without changing the target; an eligible owner's old post remains editable. |
| AC-017 | RULE-002 | Given an edit opened under account A, when the active account changes or A's lifecycle invalidates writes before save, then account B cannot submit A's session, the wrong account receives no edit data and no unauthorized write is performed. |
| AC-018 | RULE-003 | Given general, project, reply and quote-post records, when structural changes are attempted through UI or crafted API input, then general/project conversion, changed authorship/publication time, and adding/removing/retargeting reply or quote relationships are rejected; normal content edits still succeed. |
| AC-019 | FR-012 | Given a previously indexed post updated by another client, when a valid newer authoritative event arrives, then latest content is projected and the indicator displays its declared editedAt when present and no indicator otherwise; stale events cannot roll back current content or metadata. |
| AC-020 | RULE-004 | Given an ordinary post, reply or quote with existing notifications, when edited, including addition/removal of mentions and event replay, then no new social notification or repeated push is sent because of that edit. Pending original work excludes newly added recipients, may deliver to removed original mentions once, and never resets existing read/activity state (section 12). |
| AC-021 | RULE-005 | Given restricted content or replacement media awaiting/rejected by safety checks, when edited and read directly or via a quote/repost, then existing enforcement and new-version safety gates apply, report evidence remains available under existing policy and unsafe content is not exposed. |
| AC-022 | RULE-006 | Given repeated edits, when current post data and supported product endpoints are inspected, then only current content/latest edit metadata is provided, no history feature or new historical-content archive exists, and existing operational/moderation retention is preserved. |
| AC-023 | RULE-007 | Given a post deleted after the editor opened, when the author saves or retries, then the target stays deleted, the UI explains it is unavailable, unsaved text remains recoverable in the session and no replacement record is created. |
| AC-024 | NFR-001 | Given iOS, Android and web with their supported accessibility/localization modes, when completing edit/save/cancel/failure flows, then controls remain operable and existing creation/draft/scheduling behavior passes regression checks. |
| AC-025 | NFR-002 | Given an edit request and its media/PDS interactions, when observing the client/server contract, then the route and JSON/errors follow existing API conventions, final record updates go through AppView and no OAuth credentials reach or persist in Flutter. |
| AC-026 | NFR-003 | Given edit failures carrying protected payload/credential canaries, when serialized diagnostic sinks are inspected, then safe operation references and original causes are retained by the existing issue owner and protected values are absent. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | An old quote refers to a CID preceding several edits | Display latest visible original and its latest edit time; keep quoting commentary unchanged | FR-008 |
| EC-002 | Two devices, or an external client, edit concurrently | Reject stale save and preserve input for review | FR-009 |
| EC-003 | Delete races with save | Never recreate the target | RULE-007 |
| EC-004 | Optional project field is emptied or attachment removed | Persist explicit clearing; do not restore the original through fallback hydration | FR-003, FR-005 |
| EC-005 | Unchanged submission or changes reverted before save | Do not count as a content edit | FR-006 |
| EC-006 | Save times out after the PDS accepted it | Reconcile the existing operation; do not create a new post or blindly overwrite | FR-010, FR-011 |
| EC-007 | Account switches while media upload/save is in progress | Enforce captured owner and lifecycle boundaries | RULE-002 |
| EC-008 | New mentions are added, old mentions removed | Refresh mention data and visibility rules without sending social notifications | FR-012, RULE-004 |
| EC-009 | Changed image or external-card thumbnail awaits a scan | Keep current-version safety gates; never show unapproved media via embedded views | RULE-005 |
| EC-010 | Video remains unchanged versus being replaced | Preserve an existing video without requiring its original upload job; verify replacements under the existing approved flow | FR-003, FR-005 |
| EC-011 | Unsupported extension fields exist in a record | Preserve fields outside explicit supported edits; do not fabricate an editable representation | FR-005 |
| EC-012 | Content is edited repeatedly while caches/projections lag | Final reads converge to the latest authoritative version; edit timestamps do not regress on stale events | FR-006, FR-011, FR-012 |

## 15. Data / Persistence Impact

- New fields: latest edit metadata in the post read contract and an appropriate persistent representation. It must survive normal app/server restarts. The current-record timestamp contract is in section 12; exact wire/schema names and storage mechanisms belong to coding design.
- Changed fields: current published content and CID; current derived project/search/mention/language/media-safety state. Original publication time and fixed structural data do not change.
- Migration required: expected for edit metadata/indexed presentation. If metadata is added to the public record, follow the project Lexicon skill, ADR, generation and validation requirements. No migration or schema change is authorized at this stage.
- Backwards compatibility: absence of edit metadata on legacy posts means no known edit time, not an invented historical timestamp. There are no active app users requiring shipped-client compatibility, but existing PDS records, extension fields and production change controls remain relevant.
- History: no revision collection or historical body/media archive for this feature. Existing current-source ingestion, command recovery and moderation records remain governed by their existing retention policies.
- Public edit timestamps are author-declared metadata, not tamper-proof evidence. Project the current record value; maintain no separate timestamp provenance/observation state.

## 16. UI / API / CLI Impact

Placement refinement approved 2026-10-28: edit disclosure is an icon without visible Edited text, part of the relative post time (for example `[edit icon] 2h`, with no separator dot), with a single tooltip containing publication and latest-edit timestamps. Full post cards use the header time; quoted/saved summaries merge their disclosure wherever they display post time. This supersedes separate footer and adjacent-indicator placements. Tap, long press, pointer hover, keyboard focus and accessible text expose both timestamps together.


- UI: eligible-owner context menus; general/project composer edit mode; fixed reply/quote context; save/loading/conflict/error states; accessible localized edit indicator. Use existing unsaved-change protection, adapted so closing an edit cannot accidentally publish or create an ordinary new-post draft. No new durable edit-draft feature is required.
- API: authenticated owner edit operation with explicit version precondition, fixed-structure validation, safe retry/outcome handling and current edit metadata on post/quote representations. Loading comes through AppView; server-side PDS reads for authoritative write preparation remain permitted. Exact route/body design is deferred under the API spec.
- CLI: none identified.
- Background jobs: existing ingestion, current-version media checks, derived projections and cache invalidation must support edits. Notification projection must distinguish edits from initial publication. No scheduled edit or history worker.

## 17. Security / Privacy / Permissions

- Authentication: existing CraftSky session and device contracts.
- Authorization: owner-only, preserving current account eligibility/lifecycle and operation fencing. "Any time" grants no bypass of suspension, deletion or other existing write restrictions.
- Sensitive data: unsaved text/media remain under existing private composer handling; no credentials or complete record bodies in diagnostics. Publicly edited posts remain public under existing policies.
- Abuse cases: unauthorized edits, stale overwrite, recreation after deletion, retargeted conversations, sponsorship changes hiding disclosures, replacement media bypassing checks, mention notification spam and editing reported content to evade enforcement.
- Mitigation: ownership checks, version preconditions, fixed structure, visible edit time, existing safety controls and preserved moderation evidence. No new sponsorship-specific edit ban is introduced; the existing disclosure field remains editable.

## 18. Observability

- Events: distinguish edit accepted, conflict, denied, validation failure, dependency failure and projection pending using existing command/request outcomes. No product analytics expansion is required.
- Logs: follow `docs/development/logging-error-reporting.md`; preserve original typed causes, permitted operation/request references and one issue owner. Never serialize unsaved content, OAuth/service tokens, private capabilities or historical post bodies.
- Metrics: reuse existing write, command and projection metrics; new edit-specific metrics are optional design work only if necessary to distinguish outcomes.
- Alerts: no new alerting policy required. Existing write/projection/safety failure monitoring applies.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Quotes/comments discuss content later changed | Reader context can become misleading | Explicitly accepted latest-view policy and visible edit time; no historical snapshot promise |
| RISK-002 | Whole-record save drops unknown or absent editor fields | Public data loss | Authoritative base, explicit editable fields and preservation criteria |
| RISK-003 | Concurrent or uncertain writes overwrite content | Lost work or accidental recreation | Version preconditions, command reconciliation and deletion protection |
| RISK-004 | Edit triggers old publication side effects | Duplicate posts, pushes or drafts | Distinct edit submission semantics and notification regression coverage |
| RISK-005 | Media/content edits evade enforcement | Unsafe content or destroyed evidence | Apply existing controls to every current version and retain moderation evidence |
| RISK-006 | Timestamp unavailable after external edits or rebuild | Incorrect transparency indicator | Display current-record declarations only; external records without editedAt have no indicator |
| RISK-007 | Shared composer and post presentation changes affect many surfaces | Creation, drafts, scheduling or embeds regress | Explicit surface matrix and regression tests |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Published replies and quote-posts count as editable posts; their structural target stays fixed. Imported posts are editable under the same rules, preserving provenance. | Scope and menu/composer coverage need revision if any subtype should be excluded |
| ASM-002 | CraftSky assigns editedAt for content changes; retries/no-ops preserve it. All indicators use one Edited label. External missing metadata produces no indicator; there is no verified provenance claim. | Relaxation explicitly approved 2026-10-27 |
| ASM-003 | Edits add no social notifications. Fixed original recipients may still receive their pending publication notification after mention removal. Existing safety/deletion/delivery policies remain. | Relaxation explicitly approved 2026-10-27 |
| ASM-004 | The draft-like request describes composer familiarity and prefill, not saving published edits as ordinary local/scheduled drafts. Existing cancellation protection is sufficient; durable recovery across process termination is not added. | Durable edit-draft storage would require separate scope |
| ASM-005 | The same eligibility/access rules used for current post interactions and owner writes determine whether an unavailable/restricted post can be edited; no new moderation-remediation workflow is introduced. | Restricted-author UX may need additional requirements |

## 21. Open Questions

- Blocking product questions: none identified. The confirmed direction is sufficient to proceed after requirements review.
- Non-blocking review note: none outstanding for these simplifications; platform manual results remain pending.
- Design handoff: implement the section 12 current-record timestamp and fixed original-publication recipient contracts; choose persistent/wire representations before coding-plan approval. ASM-002/ASM-003 simplifications are approved. No public schema name or storage mechanism is fixed by these requirements.

## 22. Review Status

Status: Approved simplification scope (2026-10-27); manual acceptance pending

Risk level: Medium

Review recommended: Yes

Reviewer: Maintainer review pending

Date: 2026-10-26 (revised after document review)

Notes: The product direction is confirmed; the five simplifications are explicitly approved; renewed implementation review is required. Risk comes from persistent public writes, shared UI, concurrency, notification and moderation integration; no new privilege model is proposed. Review of the revised clarifications is recommended before coding-plan approval. Revised on 2026-10-26 with maintainer authorization to address DR-001–DR-004; the original review report remains a historical review of the prior drafts. Current-record timestamp and fixed-recipient simplifications supersede prior timestamp-source/pending-withdrawal details. No source changes, test implementation, schema changes, commits or production actions were performed.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-23-edit-posts/01-requirements.md`.
- Next test specification: `02-acceptance-tests.md` in the same folder.
- Must-cover requirement IDs: BR-001–BR-002, FR-001–FR-012, RULE-001–RULE-007, NFR-001–NFR-003; all map to AC-001–AC-026 above.
- Suggested test levels: unit tests for preservation, fixed structure, no-op detection and validation; Flutter widget tests for prefill/save/error/timestamp access and account fencing; backend integration tests for ownership, conditional updates, retries, deletes, ingestion/derived data, safety and notification suppression; regression tests across general/project creation, drafts, scheduling, replies, quotes and shared post surfaces; device/browser journeys on iOS, Android and web.
- Include a surface/subtype matrix rather than assuming one successful composer path proves all entry points. Test historical CID references without requiring a historical record fetch.
- Blocking open questions: none at the product requirements stage. Resolve the timestamp storage/wire representation and notification classification mechanism in coding design; the observable contract is in section 12.
- Evidence boundary: local tests and mocks do not establish live PDS propagation, production deployment or physical-device behavior. Report those checks separately if performed later.

Maintainer approved simplifications 1–5 on 2026-10-27. These supersede earlier source-provenance, pending-mention withdrawal, manual link-card metadata and accepted-waiting decisions. Stable requirement/acceptance IDs remain; implementation review must assess the revised scope.
