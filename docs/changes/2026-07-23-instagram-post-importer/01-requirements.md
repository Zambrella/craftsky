# Requirements: Instagram Historical Post Importer

## 1. Initial Request

Create a separately deployed web application that lets an existing CraftSky
member select an Instagram Data Export ZIP, review historical Instagram posts,
and import selected supported posts as ordinary
`social.craftsky.feed.post` records directly into the member's PDS. The raw
archive must remain on the device. The importer must use browser-native AT
Protocol OAuth rather than an application backend, process large ZIP files
incrementally off the main thread, upload only selected supported images, and
support safe resume and rollback.

This change is intentionally separate from the existing Flutter Instagram
account-verification and following-import feature. ZIP support for importing
the private Instagram following graph is out of scope.

## 2. Current Codebase Findings

- Relevant files:
  - `lexicon/social/craftsky/feed/post.json` defines CraftSky's public post
    record, including a `tid` record key, client-declared `createdAt`, up to four
    JPEG/PNG/WebP images, a 2,000-grapheme text limit, and a 15 MiB image-blob
    limit.
  - `appview/internal/index/craftsky_post.go` indexes
    `social.craftsky.feed.post`, drops post events from DIDs without an indexed
    `craftsky_profiles` row, materializes mentions, and activates post
    notifications.
  - `appview/internal/api/timeline_store.go` currently uses `indexed_at` as the
    activity time for authored posts in the home timeline.
  - `appview/internal/api/post_store.go` currently orders profile post lists by
    `indexed_at`.
  - `appview/migrations/000010_craftsky_posts.up.sql` defines the existing
    indexed post storage.
- Existing patterns:
  - Public user-owned content is written to PDS records and indexed by AppView.
  - Lexicon-derived Go types are generated and consumed by indexers.
  - AppView notification creation is centralized behind lifecycle logic.
  - Repository rules require an ADR before changing a Lexicon and regeneration
    through `just lexgen`.
- Current behavior:
  - Writing historical posts directly to a PDS would make them look newly
    active in the home timeline and at the top of the author's profile because
    AppView orders these surfaces by ingestion time.
  - Incoming post facets can create mention materializations and notification
    activity.
  - A post written before the author's CraftSky profile is indexed is dropped
    from AppView permanently under the current indexer behavior.
- Constraints discovered from the consented local July 2026 Instagram export:
  - A full-information export can contain unrelated messages, personal data,
    media, and relationship information alongside post data.
  - Post metadata can appear in more than one `posts*.json` representation and
    may describe the same post more than once.
  - A carousel can contain more than CraftSky's four-image maximum.
  - Captions can contain reversible UTF-8/Latin-1 mojibake.
  - Real exports may contain video even though the observed sample contains
    image media only.
  - The local export is compatibility evidence only. Raw content, filenames,
    handles, captions, media, and derived fixtures from it must not be
    committed.
- Project/tooling:
  - The repository currently has Go and Flutter applications but no JavaScript
    package, web build, or static web deployment configuration.
  - The web importer will therefore own its package manifest, lockfile, build,
    test, type-check, and static deployment configuration.

## 3. Clarifying Questions And Decisions

### Q1: Should the historical importer live in Flutter or in a separate web application?

Answer: Use a completely separate web application to avoid adding specialized
archive/media complexity to the existing Flutter app and to use the stronger
AT Protocol and browser-file tooling available in TypeScript.

Decision / implication: Add a separately built and deployed static web client
in this repository. Do not add historical post importing to Flutter.

### Q2: Should the importer have an application backend?

Answer: No. The browser should maintain the OAuth session, process the archive,
and write directly to the member's PDS.

Decision / implication: The deployed application consists of static assets and
OAuth client metadata only. It has no importer API, database, server-side OAuth
session, archive upload, or media proxy.

### Q3: How should imported posts affect CraftSky activity surfaces?

Answer: Imported posts should appear on the author's profile at their original
historical dates, but their original backfill events must be excluded from home
timelines and notifications.

Decision / implication: Add explicit import provenance to the public post
record and teach AppView to distinguish historical imports from ordinary new
posts. Do not infer imports merely from an old `createdAt`.

### Q4: Should imported posts show provenance?

Answer: Yes, show a subtle "Imported from Instagram" label.

Decision / implication: AppView and Flutter post models/presentation must carry
and render the explicit provenance without publishing an Instagram handle,
media identifier, archive identifier, filename, or other source detail.

### Q5: What happens to carousels containing more than four images?

Answer: Keep the first four images in Instagram's original order, discard the
remainder, and show a small warning before import.

Decision / implication: Do not split one Instagram post into multiple CraftSky
posts.

### Q6: What happens to video?

Answer: For mixed-media posts, import the supported images and ignore videos
with a warning. Skip video-only posts and list them as unsupported before
import.

Decision / implication: Do not add video support or generate still frames in
this change.

### Q7: What happens to captions over CraftSky's text limit?

Answer: Keep the first 2,000 graphemes, visibly warn the user, and allow the
result to be edited before import.

Decision / implication: Truncation is deterministic and previewed, never
silent. Facets are generated only after the final caption edit.

### Q8: Should common caption mojibake be repaired?

Answer: Yes, but only when the repair is confidently reversible.

Decision / implication: Apply a conservative UTF-8/Latin-1 repair, show the
result in review, and leave ambiguous text unchanged.

### Q9: Should the importer support rollback?

Answer: Yes.

Decision / implication: Request create and delete permission only for
`social.craftsky.feed.post`, track records created by an import locally, and
offer explicit rollback for those records. Rollback does not delete or modify
unrelated records.

### Q10: How should reruns and partial imports behave?

Answer: Use deterministic record keys. Skip existing imported records, never
overwrite records edited after import, allow a deleted record to be imported
again, and resume partially completed imports from the remaining posts.

Decision / implication: Use create semantics with an explicit deterministic
`tid` rkey, not update/upsert semantics. Persist only bounded progress metadata
and created AT URIs in browser storage.

### Q11: Is the private following-graph ZIP import part of this change?

Answer: No. Focus only on the historical-post web importer.

Decision / implication: The importer never calls Instagram migration AppView
routes and never extracts, displays, uploads, or retains follower/following
data.

### Q12: Is importer provenance trusted or self-asserted?

Answer: It is self-asserted. Any compatible PDS client may write the same
provenance field.

Decision / implication: Provenance grants no trust or additional visibility.
It only labels the record and reduces distribution of the original backfill
event. The label must not imply Instagram account ownership. It remains after
ordinary edits.

### Q13: How do imported posts behave after the backfill?

Answer: Imported posts participate in keyword and hashtag search. Later likes,
reposts, quotes, and replies behave normally.

Decision / implication: Suppress only the imported record's original authored
timeline item and notification activations. A later repost or quote may enter
home timelines, and later engagement may notify recipients according to their
settings.

### Q14: What is selected by default and how much review is required?

Answer: Select every importable post by default, including warned
transformations. Do not require the user to inspect every post.

Decision / implication: Use a virtualized, filterable review list with
aggregate warning/transformation/skipped counts, bulk and per-post selection,
and a final confirmation containing exact post and image counts before the
first network write.

### Q15: What happens when only some source media is usable?

Answer: Before final confirmation, keep the post selected with its remaining
valid images and show omissions. If no image remains, require explicit
confirmation for a non-empty text-only result and skip an empty result. If an
image selected in the final review fails during publication, fail that post
without silently changing it.

Decision / implication: Review-time loss is explicit and editable; runtime
loss never changes the confirmed public record shape.

### Q16: When should OAuth authorization occur?

Answer: Parse and preview the archive locally before requesting OAuth.

Decision / implication: OAuth begins only when the user proceeds toward
publication. After authorization, bind the manifest to the authenticated DID,
verify the PDS profile record, and show a final account-and-count confirmation
before any blob or record write.

### Q17: What production resource limits apply?

Answer: Do not impose an overall ZIP-size limit. Use one easily configurable,
typed safety configuration with these initial production values:

- 100,000 ZIP entries.
- 64 MiB central directory.
- 32 MiB per candidate `posts*.json`.
- 128 MiB combined candidate post metadata.
- 25,000 normalized posts.
- 64 MiB per uncompressed source image entry.
- 25 megapixels per decoded image.
- 12,000 pixels maximum width or height.
- 200:1 maximum selected-entry decompression ratio.
- One image decode/re-encode at a time.
- 15 MiB maximum final image blob, inherited from the post Lexicon.

Decision / implication: Limits are named, versioned, documented, centrally
configurable, and covered one below/at/above each boundary. Exceeding archive
metadata limits stops locally with Posts-only export guidance. Exceeding media
limits omits that media during review. There is no cumulative selected-media
limit because media is processed and released incrementally.

### Q18: How long is local import history retained?

Answer: Retain bounded progress and rollback metadata until the user explicitly
clears it or successfully completes rollback.

Decision / implication: Signing out ends OAuth access but preserves
non-content import history. Clearing history permanently removes resume and
bulk-rollback capability. A later export is a separate session: overlapping
records are skipped, new records are eligible, and the new session never
claims rollback ownership of earlier records.

### Q19: How is CraftSky membership checked?

Answer: It is sufficient to read
`social.craftsky.actor.profile/self` from the authenticated user's PDS.

Decision / implication: Do not add an importer-specific AppView membership
endpoint. The existing-member scope accepts the narrow residual risk that a
delayed AppView profile event could cause subsequently published posts not to
be indexed.

### Q20: How are invalid timestamps and duplicate conflicts handled?

Answer: Accept parseable timestamps from 6 October 2010 through 24 hours after
the browser's current time. Skip anything outside that range without
substituting the current time. Merge equivalent duplicate representations, but
skip a materially conflicting duplicate rather than guessing.

Decision / implication: Timestamp and deduplication failures are visible local
warnings and never create records.

### Q21: What accessibility and image-format editing belongs in v1?

Answer: Leave imported image alt text empty. Do not offer an alt-text editor or
missing-alt warning in this bulk flow. Only JPEG, PNG, and WebP are supported.

Decision / implication: Do not auto-generate accessibility text and do not
convert HEIC, AVIF, GIF, animated images, or other unsupported formats.

### Q22: Where and how is the importer deployed?

Answer: Deploy at the dedicated `import.craftsky.social` origin in a separate
Cloudflare Pages project.

Decision / implication: Production has its own CSP, security headers, OAuth
metadata, storage, and deployment rollback. Ephemeral Pages previews use only
synthetic fixtures and mocked OAuth/PDS behavior. Real OAuth is enabled only
for localhost, the canonical production origin, and an optional stable staging
subdomain with separate metadata.

### Q23: How are handles and incompatible PDS permissions handled?

Answer: Use `https://bsky.social` as the v1 public handle resolver, disclose
that the entered handle is sent there, and permit DID/direct-PDS entry wherever
the OAuth library supports it. Never broaden permissions for an incompatible
PDS.

Decision / implication: A PDS that cannot grant the required granular
post-create, post-delete, and supported-image-blob permissions receives a
pre-publication compatibility error. There is no wildcard repository,
record-update, or account-management fallback.

## 4. Candidate Approaches

### Option A: Static TypeScript Importer With Explicit PDS Provenance

Summary: Build a static browser application using AT Protocol browser OAuth,
streaming ZIP access, Web Workers, and direct PDS writes. Add a narrowly scoped
post provenance field plus AppView and Flutter support for historical
presentation.

Pros:

- Raw exports and OAuth sessions stay off a new CraftSky backend.
- The archive can be inspected lazily without loading or extracting it in full.
- Public records remain user-owned and portable on the PDS.
- Explicit provenance gives AppView safe, deterministic activity behavior.
- The importer can be deployed independently from Flutter.

Cons:

- Requires coordinated web, Lexicon, AppView, migration, generated-code, and
  Flutter changes.
- Browser suspension and PDS rate limits require careful progress/resume UX.
- Frontend-held OAuth sessions cannot be centrally invalidated by the importer.

Risks:

- A compromised static application origin could use a live browser OAuth
  session.
- Unbounded or adversarial ZIP content can exhaust client resources unless
  every extraction boundary is enforced.

### Option B: Static Importer With No CraftSky Record Or AppView Changes

Summary: Write ordinary CraftSky records directly and rely only on historical
`createdAt`.

Pros:

- Smallest implementation.
- No Lexicon or AppView migration.

Cons:

- Floods timelines and profile tops as newly indexed content.
- Can create notification activity from a historical backfill.
- Has no trustworthy way to distinguish imports from clock skew, relay
  backfill, or an intentionally old timestamp.

Risks:

- Violates the confirmed product behavior and makes a large import disruptive
  to other members.

### Option C: Backend-Assisted Import Service

Summary: Upload the archive or an import manifest to a server that owns OAuth
sessions, processing, retries, and AppView coordination.

Pros:

- Reliable background execution independent of a browser tab.
- Centralized retry, monitoring, and rollback.

Cons:

- CraftSky receives highly sensitive full archives or expanded private data.
- Adds durable storage, deletion, breach, operational, and compliance burden.
- Conflicts with the requested client-only trust boundary.

Risks:

- A server-side archive pipeline materially expands privacy and security scope.

## 5. Recommended Direction

Recommended approach: Option A.

Why: It preserves the user's requested local-only archive processing and direct
PDS ownership while giving CraftSky an explicit, auditable way to suppress
backfill activity. The additional Lexicon/AppView work is justified because the
no-backend shortcut is otherwise observably incorrect.

Recommended web stack:

- Vite, React, and TypeScript for a static single-page application.
- `@atproto/oauth-client-browser` and `@atproto/api` for browser OAuth and PDS
  XRPC calls.
- Generated or validated CraftSky Lexicon types rather than hand-maintained
  wire shapes.
- `@zip.js/zip.js` for Blob-backed ZIP64 access, streaming extraction, and
  worker support.
- A dedicated application Web Worker with a small typed message protocol for
  archive discovery, parsing, hashing, and media processing.
- Zod or an equivalently strict runtime schema layer for untrusted Instagram
  JSON variants.
- IndexedDB, wrapped by Dexie or an equivalently narrow adapter, for resumable
  progress and rollback metadata.
- Browser image decoding plus canvas/OffscreenCanvas re-encoding to validate
  media, remove embedded metadata, and satisfy CraftSky blob constraints.
- Vitest for unit/component tests and Playwright for browser/OAuth/file-flow
  acceptance coverage.

The importer should live in the CraftSky monorepo so its generated types and
contract tests remain synchronized with the authoritative Lexicons, while
being independently built and deployed at the dedicated
`import.craftsky.social` origin.

## 6. Problem / Opportunity

Instagram members can obtain a portable export of their historical posts, but
CraftSky currently offers no safe way to turn that export into user-owned
CraftSky records. Manually recreating years of posts is impractical. Uploading a
full export to CraftSky would expose unrelated messages, relationships,
personal information, and media. A local browser importer can make migration
practical while preserving CraftSky's public-PDS/private-AppView boundary and
the user's control over publication.

## 7. Goals

- G-001: Let an existing CraftSky member review and import supported historical
  Instagram posts from a ZIP export.
- G-002: Keep raw archive content and unrelated Instagram data on the user's
  device.
- G-003: Publish imported posts and selected images directly to the member's
  PDS as valid CraftSky records.
- G-004: Preserve original post dates and image order within CraftSky's limits.
- G-005: Prevent the original historical backfill from becoming new timeline
  or notification activity while preserving ordinary later engagement.
- G-006: Make long-running imports resumable, idempotent, cancellable, and
  explicitly reversible.
- G-007: Give members a clear review of transformations, omissions, warnings,
  and failures before publication.

## 8. Non-Goals

- NG-001: Import Instagram followers, following accounts, contacts, likes,
  comments, messages, stories, searches, profile data, or advertising data.
- NG-002: Extend the Flutter Instagram migration settings flow to parse ZIPs.
- NG-003: Upload, proxy, store, log, or inspect raw Instagram archives on
  CraftSky infrastructure.
- NG-004: Support video, audio, reels, live video, or generated video stills.
- NG-005: Convert Instagram posts into CraftSky project posts or infer craft
  metadata.
- NG-006: Publish `app.bsky.feed.post` records or cross-post into Bluesky.
- NG-007: Onboard non-members or create a CraftSky profile from the importer.
- NG-008: Prove ownership of the Instagram account from archive possession.
- NG-009: Restore Instagram likes, comments, mentions, accessibility text, or
  engagement counts.
- NG-010: Automatically map Instagram `@handles` to AT Protocol identities.
- NG-011: Overwrite or synchronize an already imported or subsequently edited
  CraftSky post.
- NG-012: Continue an import in a server background job after the browser is
  closed.
- NG-013: Guarantee compatibility with every past or future undocumented
  Instagram export shape.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Existing CraftSky member | An AT Protocol account with an indexed CraftSky profile and an Instagram ZIP export | Private local parsing, clear review, direct ownership, progress, retry, and rollback |
| Browser importer | Separately deployed static web application | Narrow OAuth authority, bounded archive processing, deterministic publication |
| Member's PDS | AT Protocol Personal Data Server selected through OAuth | Valid scoped requests, supported blobs, valid CraftSky records |
| CraftSky AppView | Firehose consumer and read API | Self-asserted provenance, historical profile ordering, original-backfill timeline exclusion and notification suppression |
| CraftSky Flutter app | Primary post-reading client | Render imported posts normally with subtle provenance |

## 10. Current Behavior

CraftSky has no historical archive importer. Its normal post write surface uses
AppView-mediated PDS writes and server-stamped current timestamps. Direct PDS
records are indexed only for known CraftSky members. Profile lists and home
timeline authored-post activity are ordered by indexing time, and the post
indexer can materialize mentions and activate notifications. The post Lexicon
has no external-import provenance.

## 11. Desired Behavior

1. The member opens a separately deployed CraftSky Instagram importer.
2. The application explains that selected posts and images will become public
   PDS data while the raw archive remains local.
3. The member selects an Instagram Data Export ZIP. The UI recommends
   requesting only Instagram Posts but tolerates a full-information export.
4. A worker inspects the archive directory and extracts only bounded supported
   post metadata. It builds a deduplicated local review manifest without
   expanding the full archive.
5. The importer opens a virtualized, filterable review with all importable
   posts selected by default. The member may use bulk or per-post selection,
   edit captions, and inspect exact warnings, omissions, and aggregate counts
   without being required to open every post.
6. When the member proceeds, they authenticate with an AT Protocol handle,
   DID, or compatible PDS using browser OAuth and grant only the displayed
   post-create, post-delete, and supported-image-upload permissions.
7. The importer binds the manifest to the authenticated DID, confirms that
   DID's PDS contains `social.craftsky.actor.profile/self`, and displays the
   destination account plus exact selected post/image counts for final
   confirmation.
8. For each finally confirmed post, the importer lazily extracts and sanitizes
   at most four supported images, uploads them directly to the PDS, and creates
   a valid CraftSky post with its original timestamp, deterministic rkey, and
   Instagram import provenance.
9. Progress is saved locally after every durable record result. The member can
   pause, retry failed items, reselect the same archive after a reload, or
   resume a partial import without duplicating successful posts.
10. Imported posts are visible and searchable as normal CraftSky posts, appear
    on the author's profile according to their original dates, and show subtle
    provenance. The original backfill items never enter home timelines or
    activate notifications; later shares and interactions behave normally.
11. The member can explicitly roll back the current locally tracked import.
    Only the created CraftSky post records are deleted; temporary/unreferenced
    blob cleanup remains the PDS's responsibility.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Existing CraftSky members shall be able to migrate supported historical Instagram posts into user-owned CraftSky PDS records without manually recreating each post. | Makes historical portability practical. | Initial request | AC-001, AC-011 |
| BR-002 | Business | Must | Raw Instagram exports and unrelated export data shall remain on the member's device. | Full exports contain sensitive unrelated information. | Initial request / Discovery | AC-002, AC-017 |
| BR-003 | Business | Must | The original historical backfill shall not create home-timeline or notification activity, while later user-initiated engagement with imported posts shall behave normally. | A migration is backfill, not a burst of new publication, but imported posts remain ordinary social content afterward. | User answer / Requirements grilling | AC-012, AC-013 |
| BR-004 | Business | Must | Members shall review material transformations and retain control over selection, continuation, retry, and rollback. | Bulk public publication must remain deliberate and recoverable. | User answers | AC-006, AC-014, AC-015 |
| FR-001 | Functional | Must | The repository shall add an independently built and deployed static web importer; historical post import shall not be added to Flutter. | Isolates specialized tooling from the primary app. | User answer | AC-001, AC-020 |
| FR-002 | Functional | Must | After local archive parsing and review but before any network publication, the importer shall authenticate through AT Protocol browser OAuth using a handle, DID, or compatible PDS/entryway and shall not collect a password or app password. | Lets the member verify archive compatibility before granting public-write authority. | Discovery / Requirements grilling | AC-003 |
| FR-003 | Functional | Must | OAuth shall request only base identity, create/delete access for `social.craftsky.feed.post`, and blob upload access for JPEG, PNG, and WebP; it shall not request record update, wildcard repo, account-management, identity-management, or unrelated RPC permission, and an incompatible PDS shall fail without a broader fallback. | Applies least privilege and enables the approved rollback. | User answer / Discovery / Requirements grilling | AC-003, AC-018 |
| FR-004 | Functional | Must | After OAuth and before final publication confirmation, the importer shall verify through the authenticated PDS that the DID owns `social.craftsky.actor.profile/self` and shall block import with guidance if the record is absent. | Limits importing to existing CraftSky members without adding an importer-specific AppView route. | Codebase / Requirements grilling | AC-004 |
| FR-005 | Functional | Must | The importer shall accept ZIP and ZIP64 files of any overall compressed size through browser file selection, recommend an export containing only Instagram Posts, and tolerate unrelated files in a full-information export without parsing them, subject to the configured bounded metadata envelope. | Supports likely user behavior and very large full exports while minimizing exposure. | Initial request / Discovery / Requirements grilling | AC-002, AC-005, AC-016 |
| FR-006 | Functional | Must | Archive directory reading, target extraction, JSON parsing, fingerprinting, and media processing shall execute outside the UI main thread using a dedicated worker and streaming/Blob-backed archive access. | Large archives must not block UI or require whole-file memory. | Initial request | AC-005, AC-016 |
| FR-007 | Functional | Must | The parser shall recognize only explicitly supported, versioned Instagram post metadata paths and shapes, including an optional single top-level export directory, and shall fail locally with guidance for absent, ambiguous, encrypted, malformed, or unsupported inputs. | Instagram's export shape is undocumented and untrusted. | Discovery | AC-005, AC-019 |
| FR-008 | Functional | Must | The parser shall merge equivalent supported `posts*.json` representations and deterministically deduplicate them using normalized timestamps and media identities, without treating caption equality alone as identity; materially conflicting representations shall produce one skipped ambiguity item rather than an automatic preference. | The observed export can repeat one post across shapes, and conflict must not be guessed. | Local sample observation / Requirements grilling | AC-005, AC-019 |
| FR-009 | Functional | Must | The parser shall extract only post timestamp, caption, ordered supported media references, media type, and bounded warning metadata required for review; unrelated JSON fields and archive entries shall not cross the worker result boundary. | Enforces data minimization structurally. | Discovery | AC-002, AC-017 |
| FR-010 | Functional | Must | The review UI shall virtualize and filter every importable or skipped post; select all importable posts by default, including warned transformations; expose bulk and per-post selection, original date, final editable caption, selected supported media, warning reasons, and aggregate selected/skipped/transformed counts; and require a final destination-account confirmation with exact post/image counts before publication. | Makes very large bulk publication practical, understandable, and deliberate. | User answers / Requirements grilling | AC-006 |
| FR-011 | Functional | Must | For a post containing more than four supported images, the importer shall retain the first four in source order, omit the remainder, and show a warning before import. | Matches the CraftSky post limit without splitting source posts. | User answer | AC-007 |
| FR-012 | Functional | Must | For mixed image/video posts, the importer shall retain supported images and omit video with a warning; it shall skip video-only posts and list them as unsupported. | CraftSky has no video post support. | User answer | AC-007 |
| FR-013 | Functional | Must | Captions over 2,000 graphemes shall be truncated to the first 2,000 graphemes, visibly warned, and editable before publication; no truncation shall occur without review. | Meets the Lexicon limit without silent loss. | User answer | AC-008 |
| FR-014 | Functional | Must | The importer shall repair only confidently reversible UTF-8/Latin-1 mojibake, show the repaired text in review, and leave ambiguous text unchanged. | Repairs the observed export defect without speculative rewriting. | User answer | AC-008 |
| FR-015 | Functional | Must | After the final caption edit, the importer shall generate valid facets for supported hashtags and URLs; Instagram `@handles` shall remain plain text and shall not be resolved or encoded as AT Protocol mention facets. | Avoids false identity links and notifications. | Discovery | AC-009, AC-013 |
| FR-016 | Functional | Must | Before upload, each selected JPEG, PNG, or WebP image shall be signature/MIME validated, decoded within the centrally configured resource limits, stripped of embedded metadata by re-encoding, and brought within the post Lexicon's supported MIME, blob-size, and aspect-ratio constraints; unsupported source formats shall not be converted. | Export media is untrusted and may expose metadata; browser-dependent conversion would be unreliable. | Discovery / Codebase / Requirements grilling | AC-010, AC-016 |
| FR-017 | Functional | Must | The importer shall upload only media belonging to a finally confirmed post, directly to the authenticated PDS, and shall create the record only after every image retained at final confirmation uploads successfully. Review-time unusable media shall be omitted with warning while valid images remain selected; a non-empty text-only result requires explicit confirmation; an empty result is skipped; and a runtime media failure shall fail the post without silently publishing a different shape. | Prevents unrelated transfer, partial records, and post-confirmation content drift. | Initial request / Discovery / Requirements grilling | AC-002, AC-010 |
| FR-018 | Functional | Must | Each imported record shall be a valid `social.craftsky.feed.post` with final text, final facets, retained images, original Instagram creation time as `createdAt`, deterministic `tid` rkey, and explicit Instagram external-import provenance. | Preserves history and provides an explicit AppView classification signal. | User answers / Codebase | AC-011, AC-019 |
| FR-019 | Functional | Must | The post Lexicon shall gain minimal self-asserted external-import provenance that identifies Instagram as the source but contains no Instagram handle, source media/post ID, filename, archive name, archive fingerprint, caption hash, or external URL and grants no trust or elevated behavior. | Enables transparent behavior without publishing unnecessary linkage or implying ownership verification. | User answer / Privacy boundary / Requirements grilling | AC-011, AC-017 |
| FR-020 | Functional | Must | AppView shall index imported posts for existing CraftSky members, persist their provenance, expose it through post responses, place them on author profile lists according to original `createdAt`, and include them in ordinary keyword/hashtag search using existing relevance and original-date chronology while preserving existing behavior for ordinary posts. | Imported history must appear in the correct profile chronology and remain discoverable without import-time ranking. | User answer / Codebase / Requirements grilling | AC-011, AC-012 |
| FR-021 | Functional | Must | AppView shall exclude the original authored imported record from every home-timeline page for the author and followers, regardless of import time, caption, facets, or original date, while allowing a later repost or quote of that record to enter timelines under ordinary rules. | Implements silent backfill without permanently preventing intentional sharing. | User answer / Requirements grilling | AC-012 |
| FR-022 | Functional | Must | AppView shall suppress notification activation whose source is the original imported record, including its mention-derived events, while later likes, reposts, quotes, and replies involving the imported post shall create normal notifications according to existing preferences. | Implements silent backfill without making future engagement second-class. | User answer / Requirements grilling | AC-013 |
| FR-023 | Functional | Must | CraftSky post responses and Flutter presentation surfaces shall render a subtle localized "Imported from Instagram" label for imported posts and no label for ordinary posts; ordinary post edits shall preserve the label and provenance. | Provides durable origin context without implying current text is unchanged. | User answer / Requirements grilling | AC-011 |
| FR-024 | Functional | Must | The importer shall derive stable deterministic record keys for a normalized source post, use create rather than update/upsert semantics, treat a matching existing imported record as already imported, and never overwrite an existing record. | Makes retry idempotent without update permission. | User answer | AC-014, AC-019 |
| FR-025 | Functional | Must | The importer shall persist bounded progress keyed to the authenticated DID and a locally derived archive-manifest fingerprint, including successful AT URIs and safe status/error codes, but not raw archive bytes, captions, image bytes, OAuth tokens, or unrelated source metadata; this history shall survive sign-out and remain until explicit clearing or successful rollback. | Supports resume and rollback without retaining sensitive content. | User answer / Privacy boundary / Requirements grilling | AC-014, AC-017 |
| FR-026 | Functional | Must | A reload or partial failure shall allow the member to reselect the same archive and resume remaining items; a previously deleted deterministic record may be created again, while an existing or edited record is skipped and never overwritten. A newer archive shall form a separate session that skips overlapping existing records, imports eligible new records, and never claims rollback ownership of an earlier session's records. | Implements the approved rerun and later-export behavior. | User answer / Requirements grilling | AC-014 |
| FR-027 | Functional | Must | Cancel shall stop scheduling new uploads without deleting successful posts; explicit rollback shall delete only the locally tracked post records created by the selected import session and shall report per-record success/failure. | Separates pause from destructive recovery. | User answer | AC-015 |
| FR-028 | Functional | Must | The importer shall expose per-post and aggregate progress, bounded retry with backoff for transient PDS/rate-limit failures, pause, retry-failed, completion, sign-out, and explicit clear-local-history actions; clearing shall explain that resume and bulk rollback become unavailable. | Long imports must be understandable and recoverable. | Discovery / Requirements grilling | AC-014, AC-015 |
| NFR-001 | Non-functional | Must | Selecting a large archive shall not copy the complete ZIP into JavaScript memory, IndexedDB, cache storage, logs, telemetry, or a network request; only the central directory and explicitly needed bounded entries may be read. | Core privacy and scalability boundary. | Initial request | AC-002, AC-016, AC-017 |
| NFR-002 | Non-functional | Must | One named, versioned, typed, centrally configurable production safety envelope shall initially limit archives to 100,000 entries, a 64 MiB central directory, 32 MiB per candidate post JSON, 128 MiB combined candidate metadata, 25,000 normalized posts, 64 MiB per uncompressed selected image, 25 decoded megapixels, 12,000 pixels per dimension, a 200:1 selected-entry decompression ratio, one concurrent image decode/re-encode, and the Lexicon's 15 MiB final blob size; there shall be no overall ZIP-size or cumulative selected-media limit. | Defends against ZIP/media bombs while supporting very large local full exports and easy future tuning. | Discovery / Requirements grilling | AC-016, AC-019 |
| NFR-003 | Non-functional | Must | Archive and media work shall be cancellable, process/release selected media incrementally with only one decode/re-encode at a time, and leave the main UI responsive enough to update progress and accept cancellation throughout supported workloads. | Required for large-file usability and bounded memory. | Initial request / Requirements grilling | AC-016 |
| NFR-004 | Non-functional | Must | The importer shall deploy at `https://import.craftsky.social` in a separate Cloudflare Pages project, self-host production scripts/assets, use a restrictive CSP and secure headers, contain no advertising/analytics/session-replay scripts, and make network calls only to chosen OAuth/PDS infrastructure and the disclosed `https://bsky.social` handle resolver. Ephemeral preview deployments shall use only synthetic fixtures and mocked OAuth/PDS behavior; real OAuth is allowed only on localhost, the canonical production origin, and an optional stable staging origin with separate metadata. | Protects highly sensitive local data and live OAuth authority through origin and deployment isolation. | Discovery / Requirements grilling | AC-018, AC-020 |
| NFR-005 | Non-functional | Must | Logs, exceptions, browser diagnostics intentionally emitted by the app, test snapshots, URLs, and error UI shall not contain archive names, entry paths, captions, handles, media bytes, OAuth artifacts, DIDs, AT URIs, or source fingerprints; production errors shall use bounded safe codes. | Prevents secondary disclosure. | Privacy boundary | AC-017 |
| NFR-006 | Non-functional | Must | The implementation shall use generated/validated Lexicon contracts, add the required ADR before modifying `lexicon/`, run `just lexgen`, and keep generated Go and web types synchronized. | Lexicons are load-bearing public contracts. | Repository rules | AC-019 |
| NFR-007 | Non-functional | Should | The importer shall support current desktop releases of Chrome, Edge, Firefox, and Safari, including ZIP64 and worker-based processing, with a clear compatibility error where a required API is absent. | Large archives are primarily a desktop workflow. | Discovery | AC-020 |
| NFR-008 | Non-functional | Must | Ordinary CraftSky post indexing, profile ordering, timeline inclusion, notification creation, deletion, moderation, and pagination shall remain unchanged except where explicit imported-post behavior applies. | Prevents a migration feature from changing normal social behavior. | Codebase | AC-012, AC-013, AC-021 |
| RULE-001 | Business rule | Must | Only a DID whose authenticated PDS contains `social.craftsky.actor.profile/self` may proceed to final publication. | Enforces the approved existing-member preflight without adding an AppView endpoint. | Codebase / Requirements grilling | AC-004 |
| RULE-002 | Business rule | Must | Archive possession is not proof of Instagram-account ownership; provenance means only that the record was created by this importer. | Avoids a false verification claim. | Discovery | AC-011 |
| RULE-003 | Business rule | Must | Imported posts are public PDS records and selected uploaded images are public blobs; the member must acknowledge this before OAuth publication begins. | Ensures informed publication. | Architecture / Discovery | AC-001, AC-003 |
| RULE-004 | Business rule | Must | Imported posts may be shown on profiles, direct post views, and relevant search surfaces; only their original backfill activity is excluded from home timelines and notification activation, while later shares and engagement follow ordinary rules. | Defines the approved visibility and later-engagement boundary. | User answer / Requirements grilling | AC-012, AC-013 |
| RULE-005 | Business rule | Must | No more than four supported source images may be attached to one imported post, and unsupported media must never be uploaded. | Enforces the approved loss policy and Lexicon bounds. | User answer | AC-007, AC-010 |
| RULE-006 | Business rule | Must | Rollback authority and behavior apply only to post records created and locally tracked by the selected import session; the importer never bulk-deletes untracked posts or blobs. | Bounds destructive behavior. | User answer | AC-015 |
| RULE-007 | Business rule | Must | Rerunning an import never updates an existing PDS record. | Protects edits and prevents surprising synchronization. | User answer | AC-014 |
| RULE-008 | Business rule | Must | Instagram data outside supported historical post metadata and selected post images must be ignored, not retained, and not transmitted. | Keeps the post importer from becoming a general archive processor. | User scope decision | AC-002, AC-017 |
| RULE-009 | Business rule | Must | A source timestamp is eligible only when it parses and falls between 2010-10-06T00:00:00Z and 24 hours after the browser's current time; invalid timestamps are skipped and never replaced with the current time. | Prevents malformed or implausible source chronology. | Requirements grilling | AC-022 |
| RULE-010 | Business rule | Must | Imported image alt text is always empty in v1, no alt-text editor or missing-alt warning is shown, and non-JPEG/PNG/WebP source formats are omitted rather than converted. | Keeps bulk review usable and format behavior deterministic across browsers. | Requirements grilling | AC-010, AC-023 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-003 | Given an existing CraftSky member using the standalone importer, when they acknowledge public publication and complete a supported import, then selected posts exist as `social.craftsky.feed.post` records owned by their DID without using the Flutter importer UI. |
| AC-002 | BR-002, FR-005, FR-009, FR-017, NFR-001, RULE-008 | Given a full-information ZIP containing synthetic post, following, follower, message, profile, advertising, and unrelated media canaries, when it is reviewed and imported, then only supported post metadata is returned from the worker and only selected sanitized image bytes plus final post records reach the PDS; every unrelated canary remains local and absent from storage/network captures. |
| AC-003 | FR-002, FR-003, RULE-003 | Given a locally parsed and reviewed archive plus a compatible handle, DID, or PDS, when the member proceeds toward publication, then browser OAuth begins only at that point and shows only base identity, create/delete CraftSky-post, and supported-image blob permissions; no password is collected, and denied/cancelled/unsupported or non-granular authorization returns safely to local review with no writes or broader-scope fallback. |
| AC-004 | FR-004, RULE-001 | Given OAuth succeeds for a DID whose PDS lacks `social.craftsky.actor.profile/self`, when the importer performs its authenticated PDS preflight, then final publication remains disabled with guidance; given the record exists, the manifest is bound to that DID and the member may continue without an importer-specific AppView membership call. |
| AC-005 | FR-005, FR-006, FR-007, FR-008 | Given synthetic ZIP, ZIP64, arbitrarily large sparse-file, optional-wrapper-directory, equivalent duplicate-shape, materially conflicting duplicate-shape, unrelated-entry, missing-post, ambiguous-post-path, encrypted, malformed, and unsupported fixtures, when the worker inspects them, then supported post variants produce one deterministic manifest independent of total ZIP size, equivalent duplicates merge, material conflicts become one skipped ambiguity item, and every unsupported variant fails or warns locally with no full extraction or network request. |
| AC-006 | BR-004, FR-010 | Given a parsed manifest containing thousands of valid, transformed, skipped, and unsupported posts, when review opens, then all importable items are selected by default, the virtualized list remains filterable without requiring every item to open, bulk/per-item selection and each applicable detail are available, aggregate counts remain exact, and the first network write is blocked until a final destination-account confirmation states exact selected post and image counts. |
| AC-007 | FR-011, FR-012, RULE-005 | Given a nine-image post, a mixed image/video post, and a video-only post, when reviewed, then the first post retains exactly its first four images with an omission warning, the mixed post retains only supported images with a video warning, and the video-only post is skipped without uploading video. |
| AC-008 | FR-013, FR-014 | Given captions at 2,000/2,001 graphemes, confidently reversible mojibake, and ambiguous byte-like text, when parsed and edited, then the limit boundary is exact, overlong text is visibly truncated and editable, reversible text is repaired, ambiguous text is unchanged, and the final preview equals the published text. |
| AC-009 | FR-015 | Given final captions containing hashtags, URLs, valid-looking Instagram `@handles`, Unicode, and byte-boundary cases, when facets are generated, then hashtag/link facets use correct UTF-8 byte ranges and no AT Protocol mention facet is created from an Instagram handle. |
| AC-010 | FR-016, FR-017, RULE-005, RULE-010 | Given valid JPEG/PNG/WebP images plus spoofed MIME, metadata-bearing, oversized, excessive-pixel, corrupt, missing, runtime-upload-failing, HEIC, AVIF, GIF, and other unsupported fixtures, when media is reviewed and published, then only sanitized supported images within configured/Lexicon limits upload; unsupported or review-time unusable media is omitted with warning while valid images remain selected; a non-empty text-only result requires explicit confirmation; an empty result is skipped; every image has empty alt text; and failure of a finally retained image prevents creation rather than silently changing the confirmed post. |
| AC-011 | BR-001, FR-018, FR-019, FR-020, FR-023, RULE-002 | Given a valid selected historical post, when imported, indexed, and later edited through an ordinary client, then its PDS record has original `createdAt`, deterministic `tid`, minimal self-asserted Instagram provenance and no source identifiers; AppView returns the provenance, profile chronology uses the original date, and Flutter retains a subtle localized label without implying verified ownership or unchanged content. |
| AC-012 | BR-003, FR-020, FR-021, NFR-008, RULE-004 | Given imported posts plus ordinary posts and later repost/quote activity, when author/follower timelines, profile pages, and keyword/hashtag searches paginate, then the original imports appear on the author's original-date profile chronology and applicable search results but never as original authored home-timeline items; later reposts/quotes may appear normally; and ordinary ordering/pagination remains unchanged. |
| AC-013 | BR-003, FR-015, FR-022, NFR-008, RULE-004 | Given an imported record deliberately containing valid mention facets, later likes/reposts/quotes/replies involving that record, and equivalent ordinary controls, when AppView indexes them, then the imported source record creates no notification lifecycle event or delivery while later engagement and ordinary controls create normal preference-governed notifications. |
| AC-014 | BR-004, FR-024, FR-025, FR-026, FR-028, RULE-007 | Given success, interruption, reload, sign-out/sign-in, transient failure, already-imported, edited-existing, deleted-previous, deterministic-rkey-collision, and a later overlapping export, when manifests are resumed or started, then safe progress survives sign-out until explicit clearing/successful rollback, successful records are not duplicated or overwritten, remaining/failed/new records can continue, a deleted record can be recreated, a newer session never claims earlier rollback targets, collisions fail safely, and clearing explains loss of resume/bulk rollback. |
| AC-015 | BR-004, FR-027, FR-028, RULE-006 | Given a partial import with successful, failed, and unrelated posts, when the member cancels, no new work starts and existing posts remain; when they explicitly roll back, only tracked successful records are deleted, per-record failures are reported, and unrelated records/blobs are never targeted. |
| AC-016 | FR-005, FR-006, FR-016, NFR-001, NFR-002, NFR-003 | Given archives of different total sizes and values one below/at/above the configured 100,000-entry, 64 MiB central-directory, 32 MiB candidate-JSON, 128 MiB combined-metadata, 25,000-post, 64 MiB source-image, 25-megapixel, 12,000-pixel-dimension, 200:1-ratio, one-decode-concurrency, and 15 MiB output boundaries, when worker processing runs, then overall ZIP size alone never rejects an archive; at/below supported bounds remain responsive, incremental, and cancellable; archive metadata overflow stops with Posts-only guidance; media overflow is omitted in review; and no case causes unbounded memory, main-thread parsing, partial publication, or content-bearing errors. |
| AC-017 | BR-002, FR-009, FR-019, FR-025, NFR-001, NFR-005, RULE-008 | Given synthetic canaries in every raw archive category and OAuth/progress value, when parsing, review, failure, resume, publication, diagnostics, and rollback execute, then raw/source canaries are absent from PDS records except explicitly selected final public content, and absent from app persistence, logs, telemetry, URLs, snapshots, errors, and stringification. |
| AC-018 | FR-003, NFR-004 | Given localhost, production, stable-staging, and ephemeral-preview builds plus their OAuth metadata, when origins, dependencies, headers, network requests, resolver disclosure, and scopes are inspected, then production is isolated at `import.craftsky.social` in its own Cloudflare Pages project, assets are self-hosted, CSP/headers are restrictive, no analytics/session replay exists, `bsky.social` handle disclosure is visible, network destinations are limited, previews cannot use real OAuth/PDS writes, and requested authority contains no wildcard/update/account/identity/unrelated permission or compatibility fallback. |
| AC-019 | FR-007, FR-008, FR-018, FR-024, NFR-002, NFR-006 | Given generated Lexicon contracts, valid/invalid provenance, deterministic-rkey vectors, ambiguous duplicate shapes, and hostile ZIP metadata, when validation and build gates run, then records and types agree across JSON/TypeScript/Go, invalid provenance/keys/inputs fail, an ADR documents the public schema decision, and no hand-maintained contract silently diverges. |
| AC-020 | FR-001, NFR-004, NFR-007 | Given canonical production, localhost, optional stable staging, and ephemeral preview builds on the supported desktop-browser matrix, when OAuth, ZIP64 selection, worker review, import, resume, and rollback smoke tests run, then authorized stable origins complete the real flow, previews remain mock-only, and each browser either succeeds or reports a pre-write compatibility error without freezing or leaking content. |
| AC-021 | NFR-008 | Given the existing ordinary post, profile, timeline, notification, moderation, deletion, and pagination regression suites, when the import changes are present, then their non-import behavior remains green. |
| AC-022 | RULE-009 | Given timestamps immediately before, at, and after 2010-10-06T00:00:00Z plus immediately before, at, and after browser-now plus 24 hours, when the manifest is built under a fixed clock, then only values inside the inclusive window are eligible and invalid values are skipped without current-time substitution. |
| AC-023 | RULE-010 | Given retained supported images and HEIC, AVIF, GIF, animated, and other unsupported source formats, when review and record generation run, then supported images carry empty alt text without an editor or missing-alt warning, while unsupported formats are omitted and never converted or uploaded. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User selects a non-ZIP file renamed `.zip` | Reject by content before JSON parsing or OAuth write. | FR-005, FR-007 |
| EC-002 | ZIP has a single wrapping export directory | Normalize and accept exact supported paths beneath that wrapper. | FR-007 |
| EC-003 | ZIP has traversal, absolute, duplicate, case-conflicting, or Unicode-confusable target paths | Never write to disk; reject ambiguous target metadata and ignore unrelated entries. | FR-007, NFR-002 |
| EC-004 | ZIP is encrypted or uses an unsupported compression method | Fail locally with safe guidance and no partial import. | FR-007 |
| EC-005 | Multiple `posts*.json` files describe the same post differently | Merge only equivalent representations; collapse material disagreement into one skipped ambiguity item without choosing a preferred variant. | FR-008 |
| EC-006 | Caption is empty | Permit an empty text value when the record otherwise satisfies the Lexicon. | FR-013, FR-018 |
| EC-007 | Timestamp is absent, malformed, before Instagram's launch date, or more than 24 hours beyond browser-now | Skip the item with a warning; do not substitute the current time. | FR-018, FR-024, RULE-009 |
| EC-008 | Several posts share one timestamp | Derive stable distinct valid TIDs using deterministic source ordering/disambiguation. | FR-024 |
| EC-009 | Two normalized source posts resolve to the same deterministic rkey | Do not overwrite; surface a collision and require the affected item to remain skipped. | FR-024 |
| EC-010 | Referenced media entry is missing or corrupt during review | Keep valid remaining images selected with an omission warning; require explicit confirmation for a non-empty text-only result; skip an empty result. | FR-010, FR-017 |
| EC-011 | Selected media is exactly at/over configured image or PDS limits | Re-encode within limits when safe; otherwise omit the media with a review warning before publication. A runtime failure after final confirmation fails the post rather than changing its shape. | FR-016, FR-017, NFR-002 |
| EC-012 | PDS accepts blobs but post creation fails | Record a safe retry state; do not claim success. Let the PDS garbage-collect unreferenced temporary blobs. | FR-017, FR-028 |
| EC-013 | PDS rate-limits, expires OAuth, or cannot grant granular permissions | Pause and back off/restore compatible authorization while retaining safe progress; fail closed without broader permissions when scopes are unsupported; never restart successful posts. | FR-002, FR-003, FR-028 |
| EC-014 | Browser reloads, suspends the worker, or signs out | Require archive reselection if the File is unavailable, verify the manifest fingerprint, preserve content-free history across sign-out, and resume remaining items only after the same DID reauthorizes. | FR-025, FR-026 |
| EC-015 | Active DID changes between parsing and publication | Fence the manifest/progress to the original DID and require explicit review under the new account. | FR-002, FR-025 |
| EC-016 | Rollback is interrupted or some records were independently deleted | Continue idempotently over tracked records and report already-absent/failure outcomes without widening targets. | FR-027 |
| EC-017 | Existing deterministic rkey contains an ordinary or different-source post | Treat as collision, never overwrite/delete it, and exclude it from rollback. | FR-024, FR-027 |
| EC-018 | Imported record is later edited by another CraftSky client | Reruns recognize it as existing and never overwrite it. | FR-024, RULE-007 |
| EC-019 | Import provenance is forged by another client | AppView still applies safe backfill semantics; the label does not claim verified Instagram ownership. | FR-019, RULE-002 |
| EC-020 | Authenticated PDS lacks the CraftSky profile self record | The importer blocks final confirmation and writes, with existing-member guidance; it does not call an importer-specific AppView membership endpoint. | FR-004, RULE-001 |
| EC-021 | Later like, repost, quote, or reply targets an imported post | Apply ordinary timeline and preference-governed notification behavior to the later activity while retaining the original post's provenance label. | FR-021, FR-022, RULE-004 |
| EC-022 | Later export overlaps an earlier import | Create a separate session, skip existing deterministic records, import eligible new records, and never add earlier records to the new rollback target set. | FR-025, FR-026, RULE-007 |
| EC-023 | Ephemeral Cloudflare Pages preview is opened | Permit synthetic parsing and mocked flows only; do not expose production OAuth metadata or real PDS write capability. | NFR-004 |
| EC-024 | Selected source image is HEIC, AVIF, GIF, animated, or otherwise unsupported | Omit it with a warning; do not convert or upload it. Supported imported images carry empty alt text. | FR-016, RULE-010 |

## 15. Data / Persistence Impact

- Public PDS record:
  - Add a minimal optional external-import provenance object to
    `social.craftsky.feed.post`.
  - The only initial supported source token is Instagram.
  - No Instagram identity or archive-specific source identifier is public.
- AppView:
  - Persist whether a post is an external historical import and its source
    token.
  - Preserve independent ingestion/index timestamps.
  - Add the profile-ordering representation needed to place imported posts by
    original `createdAt` without changing ordinary-post behavior.
  - Exclude imported rows from home timeline selection and notification
    activation only for the original backfill record; later interaction
    activity remains ordinary.
- Flutter:
  - Extend post wire/model state with minimal provenance for presentation and
    preserve it through ordinary post edits.
- Browser:
  - OAuth library-managed session state in IndexedDB.
  - Separate bounded importer progress containing DID binding, manifest
    fingerprint, deterministic item/rkey state, safe status/error codes, and
    created AT URIs.
  - Preserve content-free progress across sign-out until explicit clearing or
    successful rollback.
  - No archive, caption, source media, thumbnail, or raw JSON persistence.
- Migration required:
  - Yes, for AppView post provenance/profile-ordering storage and relevant
    indexes.
- Backwards compatibility:
  - Existing post records without provenance remain ordinary posts.
  - Existing API consumers must tolerate the new optional response field.
  - The repository is not in production, but the Lexicon change still requires
    an ADR and generated-type synchronization.

## 16. UI / API / CLI Impact

- Web UI:
  - New standalone sign-in, privacy acknowledgement, file selection, archive
    processing, review/edit, progress, completion, retry, resume, rollback, and
    local-data clearing flows.
  - Archive parsing and review precede OAuth; all importable posts are selected
    by default in a virtualized/filterable list, and a final account/count
    confirmation gates publication.
  - Clear warnings for omitted images/video, skipped posts, caption repair and
    truncation, unsupported variants, and public publication.
  - No alt-text editor or missing-alt warning in the bulk importer.
- Flutter UI:
  - Subtle localized imported-from-Instagram label on applicable post
    presentation surfaces.
  - No archive selection or historical importer workflow.
- AppView API:
  - Existing post response shapes gain optional minimal provenance.
  - No new importer upload endpoint.
- PDS XRPC:
  - Browser OAuth, `com.atproto.repo.getRecord`,
    `com.atproto.repo.uploadBlob`, `com.atproto.repo.createRecord`, and
    `com.atproto.repo.deleteRecord`.
- CLI:
  - None.
- Background jobs:
  - None. Browser work stops when the client is closed or suspended.

## 17. Security / Privacy / Permissions

- Authentication:
  - Use `@atproto/oauth-client-browser` with DPoP/PKCE and browser-managed
    non-extractable keys/session storage.
  - Do not accept passwords or app passwords.
  - Production client metadata is hosted only at
    `https://import.craftsky.social`; ephemeral previews are mock-only.
  - Use and disclose `https://bsky.social` for handle resolution, with
    DID/direct-PDS input wherever supported.
- Authorization:
  - Scope is limited to base identity, create/delete CraftSky posts, and
    JPEG/PNG/WebP blob upload.
  - No update permission is needed because existing posts are never
    overwritten.
  - Incompatible granular-permission support fails closed; no broad fallback is
    permitted.
  - Rollback target resolution comes only from the locally tracked selected
    import.
- Sensitive data:
  - Full Instagram exports are highly sensitive even though imported posts are
    intentionally public.
  - Raw ZIP/JSON, unrelated fields, captions before selection, source media,
    filenames, archive metadata, source identifiers, and manifest fingerprints
    remain local and content-free in diagnostics.
  - The UI explicitly explains which final text/images become public.
- Network:
  - No importer backend or archive proxy.
  - Only OAuth/PDS endpoints and the disclosed `https://bsky.social` handle
    resolver are allowed.
  - The entered handle's disclosure to Bluesky must be explained; a DID or
    direct PDS/entryway remains usable where the OAuth library supports it.
- Abuse cases:
  - ZIP bombs, decompression bombs, path ambiguity, excessive entries,
    oversized JSON, media decode bombs, spoofed MIME, malformed Unicode,
    malicious facets, deterministic-key collision, OAuth account switch,
    rollback target widening, and forged provenance.
- Destructive behavior:
  - Rollback is explicit, scoped, reviewable, resumable, and never uses a
    wildcard or source-derived unverified target.

## 18. Observability

- Production events:
  - No remote analytics or content telemetry.
  - Local-only aggregate progress and safe error codes may be displayed.
- Logs:
  - Production logging is disabled or restricted to non-content safe codes.
  - Development logging and tests use wholly synthetic fixtures.
- Metrics:
  - None sent remotely by the importer.
  - Existing AppView aggregate metrics may count indexed posts but must not add
    source identifiers, captions, DIDs, or AT URIs as labels.
- Alerts:
  - Existing AppView/PDS operational alerts only; no importer-specific content
    alerting.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Frontend OAuth session is usable by code running at the importer origin | Compromise could create/delete CraftSky posts during the grant | Self-host assets, strict CSP, no third-party scripts, narrow scopes, sign-out/clear action, security review |
| RISK-002 | Instagram changes undocumented export shapes | Valid archives may fail or be misparsed | Versioned strict adapters, review manifest, fail closed, synthetic fixtures derived from consented observations only |
| RISK-003 | Very large or malicious ZIP/media exhausts browser resources | Freeze, crash, or denial of service | Blob-backed reads, worker isolation, fixed multi-layer limits, cancellation, lazy selected-media extraction |
| RISK-004 | A direct historical import floods activity surfaces | Other members see disruptive old content and notifications | Explicit provenance, original-backfill timeline exclusion and notification suppression, regression tests |
| RISK-005 | Partial PDS operations leave temporary blobs or incomplete history | Storage waste or confusing partial migration | Per-post sequencing, PDS blob GC, durable local progress, retry, explicit completion report |
| RISK-006 | Deterministic key derivation collides or drifts between versions | Duplicate, skipped, or overwritten records | Versioned canonicalization, golden vectors, create-only writes, collision fail-closed, never update |
| RISK-007 | Profile chronology changes break pagination for ordinary posts | Missing/duplicate posts or reordered normal content | Import-specific ordering state, opaque cursor update, real-Postgres pagination regression tests |
| RISK-008 | Provenance is mistaken for ownership verification | Misleading trust signal | Copy and schema define importer provenance only; no handle/source ID; subtle label; no verification claim |
| RISK-009 | Browser suspension makes long imports appear stalled | Poor reliability and abandoned imports | Visible pause/resume state, per-item persistence, archive reselection, bounded retries |
| RISK-010 | Rollback local state is lost | Importer cannot rediscover every rollback target safely | Explain local recovery boundary; never widen deletion; posts remain individually deletable through CraftSky |
| RISK-011 | The PDS profile record exists but AppView has not indexed membership yet | Imported firehose events may be dropped permanently | Scope the tool to existing members, check the PDS profile self record, disclose/retry failed indexing during live validation, and retain this accepted residual risk |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Target PDS implementations support current AT Protocol browser OAuth, granular repository permissions, and scoped blob uploads. | Some PDSs require a compatibility error or a later server-assisted path. |
| ASM-002 | Current desktop browsers allow a selected large `File`/`Blob` to be read incrementally from a worker without copying the complete archive. | Browser support must narrow or the app needs an optional file-system-handle path. |
| ASM-003 | Instagram post exports retain stable enough timestamps and media paths across supported variants for deterministic deduplication/rkeys. | More variants need explicit adapters or ambiguous items must remain skipped. |
| ASM-004 | For the intended existing-member flow, finding `social.craftsky.actor.profile/self` on the authenticated PDS is a sufficient preflight even though it does not prove AppView has already indexed the membership row. | A rare AppView lag can drop imported posts; the product owner explicitly accepted the PDS-only check and residual risk. |
| ASM-005 | PDSs garbage-collect unreferenced temporary blobs after failed record creation. | Importer documentation must warn about temporary quota use or add a provider-specific cleanup path. |
| ASM-006 | The source post's first four supported images are an acceptable deterministic subset. | The review UI would need per-image selection. |
| ASM-007 | Imported post provenance can be added before any production CraftSky post schema is immutable in practice. | A sidecar record or AppView-private import registration would be needed instead. |

## 21. Open Questions

- [ ] Non-blocking for implementation, blocking for release confidence: obtain
  consented observations from additional current Instagram post exports and
  convert only their structural shapes into wholly synthetic committed
  fixtures.
- [ ] Non-blocking for implementation, blocking for live release: run browser
  OAuth, large ZIP64, rate-limit/resume, imported-post indexing, timeline,
  notification, label, and rollback tests against a real compatible PDS and the
  deployed origin.

## 22. Review Status

Status: Approved
Risk level: High
Review recommended: Required
Reviewer: Product owner
Date: 2026-07-23
Notes:

- Requirements were stress-tested one decision at a time through the
  `grilling` workflow and explicitly approved before test design.
- The approved direction is a static client-only importer with explicit public
  provenance and AppView handling. A server-side archive processor is not an
  acceptable fallback without a new privacy decision.
- Configurable safety limits, OAuth timing/authority, deployment isolation,
  PDS-only membership preflight, review defaults, media loss, retention,
  provenance trust, and later activity semantics are resolved.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`, `BR-003`, `BR-004`
  - `FR-001` through `FR-028`
  - `NFR-001` through `NFR-006`, `NFR-008`
  - `RULE-001` through `RULE-010`
- Suggested test levels:
  - Acceptance/widget: complete static UI flow, review transformations,
    progress/resume/rollback, Flutter provenance label.
  - Unit: supported JSON adapters, deduplication, Unicode repair/truncation,
    facet byte ranges, deterministic TIDs, safe errors, progress state machine,
    OAuth scopes.
  - Worker integration: ZIP/ZIP64 streaming, target-only extraction,
    cancellation, limits, image validation/sanitization, privacy canaries.
  - PDS integration: OAuth session, blob upload, deterministic create,
    already-exists, delete/rollback, rate-limit/expiry behavior.
  - AppView/Postgres integration: provenance indexing, historical profile
    pagination, original-backfill timeline/notification suppression, normal
    later engagement, and delete lifecycle.
  - Cross-language contract: Lexicon JSON, generated TypeScript/Go, AppView
    response, Flutter model.
  - Regression: ordinary post indexing/profile/timeline/notification/moderation
    behavior.
  - Manual: deployed-origin OAuth, multiple PDSs, current Instagram exports,
    large archive, supported browser matrix, long import interruption.
- Blocking open questions:
  - None for acceptance-test design.
  - Additional consented export shapes and live deployed-origin/PDS validation
    remain release gates.
