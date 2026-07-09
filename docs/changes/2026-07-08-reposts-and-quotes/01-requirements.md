# Requirements: Reposts And Quote Posts

## 1. Initial Request
The user asked to implement reposts so users can share someone else's post or project into their followers' feed, either as a straight repost or as a repost with quote. Some straight-repost work already exists. The user prefers copying Bluesky's lexicon shape and UX where appropriate, and confirmed Option A: Bluesky semantics using Craftsky's existing schema. The user also confirmed that an ADR is not needed for lexicon work because Craftsky is still in development and the product is not live.

## 2. Current Codebase Findings
- Relevant files:
  - `lexicon/social/craftsky/feed/repost.json` defines `social.craftsky.feed.repost` with required `subject` and `createdAt`.
  - `lexicon/social/craftsky/feed/post.json` already supports quote posts through `embed` with local `#quoteEmbed`.
  - `appview/internal/index/craftsky_post.go` indexes `quote_uri` and `quote_cid` from quote embeds.
  - `appview/internal/index/craftsky_repost.go` indexes straight repost records into `craftsky_reposts`.
  - `appview/migrations/000010_craftsky_posts.up.sql` stores quote pointers on `craftsky_posts`.
  - `appview/migrations/000011_craftsky_interactions.up.sql` stores active repost records and enforces one active repost per `(did, subject_uri)`.
  - `appview/internal/api/post.go` exposes straight repost and unrepost handlers at `/v1/posts/{did}/{rkey}/reposts`.
  - `appview/internal/api/timeline_store.go` currently lists top-level posts from the viewer and followed accounts, but not repost activity as timeline items.
  - `craftsky_posts` already stores `reply_root_uri` and `reply_parent_uri`, so AppView can identify reply targets without lexicon changes.
  - `app/lib/feed/providers/toggle_repost_post_provider.dart` optimistically toggles straight reposts.
  - `app/lib/feed/widgets/post_composer_sheet.dart` supports replies but not quote-post composition.
- Existing patterns:
  - Writes go through AppView to the caller's PDS; reads come from AppView.
  - `/v1/*` API JSON uses camelCase.
  - Post responses currently expose `quote` as a strongRef only.
  - Timeline responses currently return `PostPage` with `items: [Post]`, not feed items with reasons.
  - Flutter mutation providers optimistically update live timeline/profile/project/comment caches.
- Current behavior:
  - Users can create general posts, project posts, replies, images, likes, and straight repost records.
  - Quote post record shape exists and can be accepted by `POST /v1/posts` as `embed.quote`, but the Flutter UI has no quote-post entry point and no quoted-card rendering.
  - Straight reposts affect `repostCount` and viewer state, but followed-user repost activity does not appear as a distinct timeline item with "reposted by" context.
  - Search and profile project surfaces intentionally exclude quote posts from project/search root-result scopes in several queries.
- Constraints discovered:
  - Product vision requires chronological feed behavior, no algorithmic ranking, no ads, user-owned data, and social basics done well.
  - Lexicons are load-bearing, but the user explicitly waived ADR overhead for this pre-live development change.
  - Existing `social.craftsky.feed.post` diverges from Bluesky by keeping images top-level; quote-with-images should preserve that Craftsky shape instead of moving to `app.bsky.embed.recordWithMedia`.
  - API changes must remain additive where possible and preserve existing post-shaped clients.
  - Quote and repost records are public PDS records.
- Test/build commands discovered:
  - Full dev stack: `just dev` or `just dev-d`.
  - Go tests: `just test` with compose Postgres running.
  - Flutter tests: `just app-test`.
  - Flutter analysis: `just app-analyze`.
  - Lexicon generation, if lexicon files change: `just lexgen`.

## 3. Clarifying Questions And Decisions
### Q1: Which approach should requirements use?
Answer: Option A sounds good.
Decision / implication: Requirements use Bluesky semantics with Craftsky schema: straight reposts remain `social.craftsky.feed.repost`; quote posts remain normal `social.craftsky.feed.post` records with `embed.quoteEmbed`.

### Q2: Is an ADR required if lexicon work is needed?
Answer: No ADR is needed because the product is still in development and not live.
Decision / implication: This requirements artifact records the process waiver. Later stages may still use the atproto lexicon checklist and `just lexgen`, but must not block on an ADR for this feature.

### Q3: How should timeline duplicate repost activity behave?
Answer: Show straight reposts as separate chronological feed items.
Decision / implication: Do not collapse multiple followed-user reposts server-side in this feature. Each authored post and each straight repost activity has its own deterministic timeline item.

### Q4: Should self-reposts and self-quotes be allowed?
Answer: Yes.
Decision / implication: Users may repost or quote their own eligible top-level posts or project posts. These actions follow the same record and feed semantics as reposting or quoting another user's post.

### Q5: How should repost and quote counts appear in the UI?
Answer: AppView should expose `repostCount` and `quoteCount` separately, but Flutter may add them together for one visible total on the repost/share control.
Decision / implication: The data contract preserves distinct semantics. The first UI does not need a separate quote-count affordance.

### Q6: Should tapping the combined repost/quote count open a list?
Answer: No.
Decision / implication: The repost/share control opens the action menu. No reposts/quotes list screen is in scope.

### Q7: Should optional `via` be added to the repost lexicon now?
Answer: No.
Decision / implication: Keep `social.craftsky.feed.repost` unchanged unless implementation uncovers a concrete need.

### Q8: Should quote posts support images?
Answer: Yes.
Decision / implication: Quote posts may use normal top-level Craftsky images plus `embed.quote`. Do not introduce Bluesky `recordWithMedia`.

### Q9: Should quote posts require non-empty commentary?
Answer: Yes.
Decision / implication: Quote-post creation requires valid non-empty text so an empty quote does not duplicate straight-repost semantics.

### Q10: What happens when quoted or reposted content is unavailable or hidden?
Answer: Quote posts remain visible with a quoted-content placeholder unless the quote post itself is hidden. Straight reposts of hidden or unavailable subjects are filtered out completely.
Decision / implication: Quote posts preserve the quote author's own commentary; straight reposts have no independent content to render safely.

### Q11: Should straight reposts appear on profiles?
Answer: No.
Decision / implication: Profiles remain authored-post surfaces in this feature. Quote posts may appear as authored posts; straight repost records appear only as home timeline activity.

### Q12: How optimistic should Flutter cache updates be?
Answer: Straight repost actions optimistically update button/count state only, not timeline insertion. Quote posts should follow the existing normal post-create cache behavior rather than special quote-specific insertion rules.
Decision / implication: Avoid speculative repost feed-item insertion and keep quote-post cache behavior consistent with the existing composer.

### Q13: What API shape should the home timeline use?
Answer: Change only the home timeline endpoint to return feed items with `{post, reason}`.
Decision / implication: Profile, search, thread, and post-detail responses remain post-shaped. Repost attribution belongs to home timeline feed items, not to the original post model.

### Q14: What should a straight repost reason contain?
Answer: Include a hydrated lightweight actor/profile summary plus repost record identity and timestamp.
Decision / implication: Flutter can render "reposted by" without extra profile lookups.

### Q15: What should quote preview hydration return?
Answer: Return a compact one-level quote preview model with explicit `visible`, `unavailable`, or `hidden` state.
Decision / implication: Do not recursively embed full `PostResponse` objects or hydrate nested quoted previews in this feature.

### Q16: How should counts interact with moderation visibility?
Answer: Counts should reflect active, visible records under the same policy used for the response.
Decision / implication: Counts should not leak hidden or moderated content where AppView can apply the visibility policy.

### Q17: Should reposting or quoting replies be allowed?
Answer: No for this first implementation.
Decision / implication: Only top-level posts and project posts are eligible repost/quote targets. AppView rejects reply targets, and Flutter hides repost/quote actions on reply cards. The restriction is intentionally easy to relax later because reply target status is already indexed.

### Q18: Should users be able to repost or quote blocked, muted, or moderated targets?
Answer: No.
Decision / implication: The UI should not offer amplification actions for non-actionable targets, and AppView should reject writes when the target is hidden by current visibility policy.

### Q19: Should quote posts create notifications?
Answer: No new quote notification type is in scope.
Decision / implication: Existing notification behavior may continue, but this feature does not add quote-specific notifications.

## 4. Candidate Approaches
### Option A: Bluesky Semantics, Craftsky Schema
Summary: Keep straight reposts as interaction records and quote posts as normal posts with an embedded strongRef. Add the missing timeline, quote hydration, counts, and Flutter UX.
Pros: Matches the user-selected direction, matches Bluesky's user model, uses existing Craftsky schema, avoids breaking migrations, and preserves quote-with-images support through top-level images.
Cons: Requires AppView timeline response changes and Flutter model/UI updates.
Risks: Feed response compatibility, duplicate timeline entries, quoted-post moderation, and cache consistency need focused tests.

### Option B: Literal Bluesky Embed Shape
Summary: Move quote embeds toward Bluesky's `app.bsky.embed.record` and `recordWithMedia` style.
Pros: Closer to exact Bluesky lexicon conventions.
Cons: Conflicts with Craftsky's existing top-level image design and creates unnecessary schema churn.
Risks: Higher lexicon and indexer complexity with little product benefit.

### Option C: New Quote/Repost Record Type
Summary: Model quote reposts as a new interaction-like record rather than as normal posts.
Pros: Keeps all share actions in one interaction family.
Cons: Diverges from Bluesky, duplicates post behavior, weakens search/profile/thread consistency, and complicates moderation.
Risks: Long-term interoperability and product semantics risk.

## 5. Recommended Direction
Recommended approach: Option A, with no ADR requirement for this pre-live change.

Why: Bluesky's important distinction is semantic: a straight repost is an interaction record; a quote post is a post that embeds another record. Craftsky already has that record model. The remaining work should complete the AppView and Flutter product experience rather than replacing the existing lexicon shape.

## 6. Problem / Opportunity
Craftsky users can currently interact with posts through likes and straight repost records, but reposts do not fully behave like a social feed feature. Followers should see posts reshared by people they follow, and users should be able to add commentary when sharing someone else's post or project. Completing this makes the chronological home feed more useful while preserving Craftsky's user-owned atproto data model.

## 7. Goals
- G-001: Let a user straight-repost an eligible indexed top-level Craftsky post or project into their followers' chronological timeline.
- G-002: Let a user quote-post an eligible indexed top-level Craftsky post or project with their own non-empty text and optional allowed attachments.
- G-003: Render straight reposts and quote posts in a Bluesky-familiar way.
- G-004: Keep the home timeline chronological and non-algorithmic.
- G-005: Preserve existing Craftsky post, project, image, reply, moderation, and interaction contracts except where additive changes are required.

## 8. Non-Goals
- NG-001: No algorithmic ranking, recommendations, boosting, or paid reach.
- NG-002: No literal migration to Bluesky's `app.bsky.embed.record` or `app.bsky.embed.recordWithMedia` schema.
- NG-003: No ADR for lexicon changes in this pre-live feature scope.
- NG-004: No quote-detach, postgate, quote-disable, or anti-dogpile controls in this scope.
- NG-005: No support for quoting arbitrary non-Craftsky records unless they are already representable by existing indexed data and strongRef handling.
- NG-006: No reusable generic feed-generator or XRPC surface.
- NG-007: No push notifications for quotes or reposts unless already covered by existing notification behavior.
- NG-008: No private reposts or private quote posts; PDS records remain public.
- NG-009: No reposting or quoting reply posts in the first implementation.
- NG-010: No reposts/quotes list screen or tap-through count detail surface in this scope.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Viewer | Authenticated Craftsky user reading their home timeline. | See posts and reposts from followed accounts in chronological order with clear attribution. |
| Reposter | Authenticated Craftsky user sharing an eligible post or project without commentary, including their own eligible content. | Repost, undo repost, and see optimistic feedback. |
| Quote poster | Authenticated Craftsky user sharing an eligible post or project with commentary, including their own eligible content. | Compose a quote post with a preview of the quoted post and publish it as their own post. |
| Original author | Author of the post being reposted or quoted. | Keep attribution and engagement semantics clear; have moderation filtering still apply. |
| Moderator/AppView | Craftsky backend applying policy and serving read models. | Index, count, hydrate, and filter repost/quote content without bypassing existing moderation rules. |

## 10. Current Behavior
Straight repost records can be written and indexed, and post responses expose `repostCount` plus `viewerHasReposted`. The home timeline lists top-level posts from the viewer and followed accounts but does not include repost activity from followed accounts. Quote-capable record shape exists in `social.craftsky.feed.post`, and AppView can store quote strongRefs, but Flutter has no quote action, no quote composer mode, and no hydrated quoted-post card in feed rendering.

## 11. Desired Behavior
The post action row should offer a Bluesky-style share choice on eligible top-level posts and project posts: straight repost or quote. A straight repost writes or deletes a `social.craftsky.feed.repost` record and appears in followers' timelines as the original post with a "reposted by" reason. A quote creates a normal `social.craftsky.feed.post` containing non-empty commentary and `embed.quote`, appears as the quoting user's post, and renders a compact quoted-post preview card. The home timeline returns feed items, remains reverse-chronological by authored post or repost activity time, and has no algorithmic scoring. Reply cards do not expose repost or quote actions in this first implementation.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky users must be able to share an eligible top-level Craftsky post or project to their followers without commentary. | Straight reposts are a social basic in the product vision. | Prompt, vision doc, discovery, user grilling | AC-001, AC-003 |
| BR-002 | Business | Must | Craftsky users must be able to share an eligible top-level Craftsky post or project with their own commentary. | Quote posts make resharing conversational while preserving attribution. | Prompt, Bluesky research, discovery, user grilling | AC-002, AC-004 |
| BR-003 | Business | Must | Repost and quote behavior must preserve a chronological followed-account feed and must not introduce ranking. | Product principle: no algorithmic feed. | Vision doc, AGENTS.md | AC-005 |
| BR-004 | Business | Should | The user experience should feel familiar to Bluesky users for repost versus quote actions. | The user explicitly requested copying Bluesky's approach. | Prompt, user decision, Bluesky research | AC-006 |
| FR-001 | Functional | Must | The system shall keep straight reposts represented as `social.craftsky.feed.repost` records with a strongRef subject. | Matches current schema and Bluesky semantics. | Codebase, Bluesky repost lexicon | AC-001, AC-007 |
| FR-002 | Functional | Must | The system shall keep quote posts represented as `social.craftsky.feed.post` records with `embed.quote` translated to `social.craftsky.feed.post#quoteEmbed`. | Matches current Craftsky schema and Bluesky quote semantics. | Codebase, Bluesky post docs | AC-002, AC-008 |
| FR-003 | Functional | Must | The Flutter post action UI shall let the user choose between straight repost and quote when activating the repost/share affordance on an eligible top-level post or project post, and shall not offer those actions on replies. | Users need both choices from one familiar entry point, while reply sharing is intentionally out of scope for v1. | Recommended direction, user grilling | AC-006, AC-027 |
| FR-004 | Functional | Must | The quote composer shall show a preview of the quoted post and submit a quote post with non-empty text plus the quoted post's `{uri,cid}` strongRef. | Prevents accidental context loss, avoids empty quote posts duplicating repost semantics, and writes the correct atproto reference. | Recommended direction, current composer pattern, user grilling | AC-002, AC-004, AC-028 |
| FR-005 | Functional | Must | The home timeline API shall include eligible straight repost activity from followed accounts as distinct feed items carrying the original post plus repost attribution. | Followers must see straight reposts in their feed, and each repost activity remains chronological and attributable. | Prompt, Bluesky feed defs, codebase gap, user grilling | AC-003, AC-009, AC-029 |
| FR-006 | Functional | Must | The timeline API shall include quote posts as normal authored posts from followed accounts. | Quote posts are posts, not interaction records. | Bluesky semantics, codebase | AC-004, AC-010 |
| FR-007 | Functional | Must | Feed items that represent straight reposts shall expose a hydrated lightweight reposter actor summary, repost record URI, repost CID when available, and repost indexed/created time. | Flutter needs attribution and stable ordering/context without extra profile lookups. | Bluesky `reasonRepost`, codebase, user grilling | AC-009 |
| FR-008 | Functional | Must | Post-shaped responses that contain a quote strongRef shall expose a compact one-level quoted-post preview for Flutter to render when the quoted post is indexed and visible. | A strongRef alone is insufficient for the UX, while recursive full-post hydration is unnecessary and risky. | Codebase gap, Bluesky embed view, user grilling | AC-011, AC-030 |
| FR-009 | Functional | Must | When a quoted post is missing, deleted, unindexed, hidden, or blocked by moderation policy, the response shall provide a stable placeholder state rather than failing the containing quote post. | Quote posts must remain renderable when embedded content is unavailable. | Bluesky embed view states, moderation docs | AC-012 |
| FR-010 | Functional | Must | Engagement summaries shall expose straight repost count and quote count as separate values, while Flutter may render a single combined repost/share count in the action row. | Bluesky separates reposts and quotes in data, and the selected UI keeps one simple visible count for now. | Bluesky research, user selected Option A, user grilling | AC-013, AC-031 |
| FR-011 | Functional | Must | `viewerHasReposted` shall continue to refer only to the viewer's active straight repost record for the subject post. | Quote posts are authored posts and should not toggle straight repost state. | Existing contract, Bluesky semantics | AC-014 |
| FR-012 | Functional | Must | Unreposting shall delete only the viewer's active straight repost record and shall not delete or alter quote posts authored by the viewer. | Prevents destructive confusion between repost and quote post. | Existing handler behavior, recommended direction | AC-015 |
| FR-013 | Functional | Should | Profile post lists shall continue to list authored posts, including quote posts, but shall not list straight repost interaction records as authored posts. | Reposts are feed activity, not authored posts. | Bluesky semantics, current profile pattern | AC-016 |
| FR-014 | Functional | Should | Search result surfaces shall continue to treat quote posts consistently with existing search scope decisions unless explicitly expanded by later search requirements. | Avoids broad search behavior changes in a repost feature. | Codebase search filters | AC-017 |
| FR-015 | Functional | Must | Existing project-post constraints shall remain intact: project posts shall not become replies or quote posts unless a separate product decision changes that rule. | Current validation rejects project quote/reply combinations; changing it is separate scope. | Codebase | AC-018 |
| FR-016 | Functional | Should | The repost lexicon should remain unchanged for this feature unless implementation uncovers a concrete need; any lexicon changes shall be backward-compatible optional additions only. | Avoids unnecessary schema churn while preserving future evolution room. | User ADR waiver, atproto lexicon guidance, Bluesky repost lexicon, user grilling | AC-019 |
| FR-017 | Functional | Must | The home timeline endpoint shall return feed items with `{post, reason}` semantics, and profile, search, thread, and post-detail surfaces shall remain post-shaped. | Repost attribution belongs to home timeline activity, not to every post response. | User grilling, codebase | AC-032 |
| FR-018 | Functional | Must | The repost/share control's combined count shall be passive feedback in this feature and shall not open a reposts/quotes list. | A list surface is separate product scope. | User grilling | AC-033 |
| FR-019 | Functional | Must | Straight-repost optimistic UI shall update only the visible action state/count and shall not insert a repost feed item into the local timeline cache before refresh/indexing. | Avoids cursor, rollback, and duplicate reconciliation complexity. | User grilling, current provider pattern | AC-034 |
| FR-020 | Functional | Should | Quote-post creation shall follow the existing normal post-create cache behavior rather than adding quote-specific optimistic insertion rules. | Quote posts are normal posts. | User grilling, current composer pattern | AC-035 |
| FR-021 | Functional | Must | The feature shall not add a new quote-specific notification type. | Notifications are a separate product surface with abuse and moderation implications. | User grilling | AC-036 |
| NFR-001 | Non-functional | Must | The home timeline shall remain reverse-chronological by post/repost activity timestamp and deterministic for cursor pagination. | Product principle and pagination correctness. | Vision doc, API architecture | AC-005, AC-020 |
| NFR-002 | Non-functional | Must | The implementation shall preserve `/v1/*` camelCase JSON and standard error envelope conventions. | Maintains API consistency. | AGENTS.md, API specs | AC-021 |
| NFR-003 | Non-functional | Should | Timeline and quote hydration should avoid N+1 profile/post lookups for normal page sizes. | Prevents feed performance regressions. | Codebase patterns | AC-022 |
| NFR-004 | Non-functional | Must | The feature shall preserve PDS-token privacy: Flutter shall not receive or store PDS OAuth tokens. | Architectural rule. | AGENTS.md | AC-023 |
| NFR-005 | Non-functional | Should | Flutter optimistic cache updates should remain reversible on API failure for repost and quote actions. | Current mutation UX pattern and user trust. | Codebase | AC-024, AC-034, AC-035 |
| RULE-001 | Business rule | Must | A straight repost and a quote post are distinct actions: a straight repost is an interaction record; a quote post is an authored post. | Core Bluesky semantic distinction. | User decision, Bluesky research | AC-001, AC-002, AC-014, AC-015 |
| RULE-002 | Business rule | Must | A user may have at most one active straight repost per subject post. | Existing DB invariant and expected toggle behavior. | Codebase | AC-007 |
| RULE-003 | Business rule | Must | A user may author multiple quote posts for the same subject unless normal post creation limits reject them. | Quote posts are normal posts, not toggles. | Bluesky semantics | AC-025 |
| RULE-004 | Business rule | Must | Straight repost feed items must not transfer authorship; the original post remains authored by the original author, with repost attribution shown separately. | Prevents attribution confusion. | Prompt, Bluesky feed defs | AC-009 |
| RULE-005 | Business rule | Must | Quote posts must attribute both the quote author and the quoted post author when the quoted post is visible. | Preserves creator attribution. | Prompt, Bluesky embed view | AC-011 |
| RULE-006 | Business rule | Must | Moderation visibility rules must apply to original posts, quote posts, and quoted previews before serving them to Flutter. | Prevents repost/quote paths from bypassing AppView policy. | AGENTS.md, moderation docs | AC-012, AC-026 |
| RULE-007 | Business rule | Must | No ADR is required for lexicon edits in this feature because Craftsky is pre-live, but lexicon style/evolution checks and regeneration still apply when lexicon files change. | Captures explicit user process decision while preserving technical hygiene. | User answer, atproto lexicon skill | AC-019 |
| RULE-008 | Business rule | Must | Only top-level posts and project posts are eligible repost/quote targets in this implementation; reply posts are not eligible targets. | Keeps the first UX simpler while preserving an easy future relaxation path. | User grilling, codebase | AC-027, AC-037 |
| RULE-009 | Business rule | Must | Users may straight-repost or quote their own eligible top-level posts and project posts. | Resurfacing and updating one's own work is valid social behavior. | User grilling | AC-038 |
| RULE-010 | Business rule | Must | Straight reposts of unavailable or hidden subjects shall be filtered out completely; quote posts shall remain visible with a placeholder when only the quoted target is unavailable or hidden. | Straight reposts have no independent content, while quote posts contain the quote author's own commentary. | User grilling | AC-012, AC-026, AC-039 |
| RULE-011 | Business rule | Must | Repost and quote counts shall not reveal records hidden by the viewer's applicable moderation or visibility policy where AppView can apply that policy. | Counts should not leak hidden content. | User grilling, moderation requirements | AC-040 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-001, RULE-008 | Given an authenticated user chooses straight repost for an eligible indexed top-level post or project post, when the request succeeds, then AppView writes a `social.craftsky.feed.repost` record with the target post strongRef as `subject`. |
| AC-002 | BR-002, FR-002, FR-004, RULE-001, RULE-008 | Given an authenticated user chooses quote for an eligible indexed top-level post or project post and enters valid non-empty text, when the request succeeds, then AppView writes a `social.craftsky.feed.post` record whose embed is `social.craftsky.feed.post#quoteEmbed` referencing the target `{uri,cid}`. |
| AC-003 | BR-001, FR-005 | Given Alice follows Bob and Bob straight-reposts Carol's visible post, when Alice loads her timeline, then Carol's post appears as a feed item attributed as reposted by Bob. |
| AC-004 | BR-002, FR-004, FR-006 | Given Alice follows Bob and Bob quote-posts Carol's visible post, when Alice loads her timeline, then Bob's quote post appears as Bob's authored post with Carol's quoted post preview. |
| AC-005 | BR-003, NFR-001 | Given timeline items include authored posts and straight repost activity, when the timeline is requested, then items are ordered reverse-chronologically by the relevant activity timestamp with deterministic tie-breakers and no ranking score. |
| AC-006 | BR-004, FR-003 | Given a user activates the repost/share control on an eligible top-level post or project post, then the UI offers separate straight repost and quote actions, and selecting quote opens the quote composer. |
| AC-007 | FR-001, RULE-002 | Given a user already has an active straight repost for a subject post, when they attempt to straight-repost again, then no duplicate active repost is created and the existing active repost state is returned or preserved. |
| AC-008 | FR-002 | Given AppView receives a quote create request, then the PDS write body uses Craftsky's current quote embed lexicon shape rather than Bluesky's `app.bsky.embed.record` shape. |
| AC-009 | FR-005, FR-007, RULE-004 | Given a timeline item represents a straight repost, then the response includes the original post data plus repost reason data containing a lightweight reposter actor summary and repost record identity/timestamp. |
| AC-010 | FR-006 | Given a quote post is indexed as a top-level post, when timeline eligibility is evaluated, then it is included or excluded by the same followed-author rules as other top-level authored posts. |
| AC-011 | FR-008, RULE-005 | Given a post response has a visible indexed quote target, then the response includes compact quoted-post preview data with quoted author attribution and renderable text/project/image summary fields. |
| AC-012 | FR-009, RULE-006 | Given a quote target is unavailable or hidden by policy, when the quote post is served, then the quote post still renders and the quoted preview is represented by a stable unavailable/hidden placeholder. |
| AC-013 | FR-010 | Given a post has active straight reposts and quote posts referencing it, when a post-shaped response is returned, then straight repost count and quote count are exposed separately in the API/model. |
| AC-014 | FR-011, RULE-001 | Given the viewer has authored a quote post for a subject but has no active straight repost record, when the subject post is returned, then `viewerHasReposted` is false. |
| AC-015 | FR-012, RULE-001 | Given the viewer has both a straight repost and one or more quote posts for a subject, when the viewer unreposts the subject, then only the straight repost record is deleted and the quote posts remain. |
| AC-016 | FR-013 | Given a profile posts tab is loaded, then authored quote posts may appear as posts but straight repost records do not appear as authored posts. |
| AC-017 | FR-014 | Given existing search filters exclude quote posts from specific root-result scopes, then this feature does not change those search semantics unless later requirements explicitly do so. |
| AC-018 | FR-015 | Given a project post create request includes a quote or reply, then existing validation continues to reject that request. |
| AC-019 | FR-016, RULE-007 | Given implementation changes a lexicon file for this feature, then the change is backward-compatible and `just lexgen` is run, while no ADR artifact is required. |
| AC-020 | NFR-001 | Given a paginated timeline containing mixed authored and repost items, when the client follows cursors across pages, then no eligible item is skipped or duplicated because of ordering ties. |
| AC-021 | NFR-002 | Given any new or changed `/v1/*` request or response for this feature, then JSON keys are camelCase and errors use `{error, message, requestId}` with optional `fields`. |
| AC-022 | NFR-003 | Given a normal timeline page with posts, reposts, and quote previews, then AppView resolves authors and quoted previews using batched or bounded queries rather than one unbounded query per item. |
| AC-023 | NFR-004 | Given Flutter performs straight repost, unrepost, or quote-post actions, then Flutter sends only Craftsky session credentials to AppView and never receives PDS OAuth tokens. |
| AC-024 | NFR-005 | Given a repost or quote action fails after optimistic UI update, then affected Flutter caches revert to the prior visible state and a user-facing error is emitted through existing messaging patterns. |
| AC-025 | RULE-003 | Given a user has already quote-posted a subject, when they compose another valid quote post for the same subject, then normal post creation may succeed and no toggle-style deduplication is applied. |
| AC-026 | RULE-006, RULE-010 | Given an original post or quoted target is hidden/taken down by moderation, when timeline or post detail is served, then straight reposts of hidden subjects are filtered and quote previews are filtered or represented according to the same moderation policy used for existing post surfaces. |
| AC-027 | FR-003, RULE-008 | Given a post card represents a reply, when Flutter renders its action row, then repost and quote actions are not offered. |
| AC-028 | FR-004 | Given a user attempts to submit a quote post with empty or whitespace-only commentary, then the request is rejected with a validation error and no PDS record is written through AppView. |
| AC-029 | FR-005 | Given multiple followed accounts straight-repost the same visible original post, when the home timeline is requested, then each repost appears as its own deterministic chronological feed item with its own attribution. |
| AC-030 | FR-008 | Given a quoted post is itself a quote post, when it is returned as a quote preview, then the preview does not recursively hydrate the nested quoted target. |
| AC-031 | FR-010 | Given Flutter renders the repost/share control for a post with separate `repostCount` and `quoteCount`, then the visible count may be the sum of both values while the model preserves both values separately. |
| AC-032 | FR-017 | Given the home timeline endpoint is called, then response items use a feed-item shape with `post` and optional `reason`; given profile, search, thread, or post-detail surfaces are called, then those surfaces remain post-shaped. |
| AC-033 | FR-018 | Given the user taps the repost/share control or its combined count, then the action menu opens and no reposts/quotes list screen is required. |
| AC-034 | FR-019, NFR-005 | Given a straight repost action succeeds or fails, then Flutter updates or rolls back the action state/count but does not optimistically insert a repost feed item into the timeline list. |
| AC-035 | FR-020, NFR-005 | Given a quote post is created, then Flutter applies the same cache insertion or refresh behavior used by normal post creation, with no quote-specific timeline insertion path. |
| AC-036 | FR-021 | Given a user quote-posts another user's post, then this feature does not create a new quote-specific notification type. |
| AC-037 | RULE-008 | Given AppView receives a straight repost or quote-post create request targeting a reply post, then the request is rejected with a validation error and no PDS record is written through AppView. |
| AC-038 | RULE-009 | Given a user straight-reposts or quote-posts their own eligible top-level post or project post, when the request is valid, then the action succeeds using the same semantics as sharing another user's post. |
| AC-039 | RULE-010 | Given a straight repost subject is unavailable, hidden, or unindexed when a timeline is served, then the repost feed item is omitted completely. |
| AC-040 | RULE-011 | Given counts are returned for a viewer, then `repostCount` and `quoteCount` do not include records hidden from that viewer by applicable AppView moderation or visibility policy where that policy can be evaluated. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Repost target is not indexed. | Straight repost and quote actions return `post_not_found` or an equivalent validation error; no PDS record is written through AppView. | FR-001, FR-002 |
| EC-002 | Quote target becomes unavailable after quote post is created. | Quote post remains visible if otherwise eligible; quoted preview renders unavailable/hidden state. | FR-009 |
| EC-003 | Viewer reposts their own eligible top-level post or project post. | Allowed; it behaves like any straight repost and does not duplicate authored post identity. | RULE-001, RULE-004, RULE-009 |
| EC-004 | Viewer quotes their own eligible top-level post or project post. | Allowed; it is an authored quote post and does not toggle straight-repost state. | RULE-003, RULE-009 |
| EC-005 | Same original appears because followed author posted it and another followed user reposted it. | Both activities may appear as separate deterministic feed items; attribution must not be lost. | FR-005, NFR-001 |
| EC-006 | Multiple followed users repost the same original. | Each repost appears as a distinct deterministic feed item with its own reposter attribution. | FR-005, FR-007 |
| EC-007 | Repost record delete arrives before create during firehose replay. | Indexer remains idempotent and active repost state is eventually correct. | FR-001, RULE-002 |
| EC-008 | Quote post has images. | Quote post may use existing top-level `images` plus `embed.quote`; no `recordWithMedia` wrapper is required. | FR-002 |
| EC-009 | Quoted target is a project post. | Preview renders project-summary information when available and does not require converting the quote post itself into a project post. | FR-008, FR-015 |
| EC-010 | Unrepost is requested when no active straight repost exists. | Existing idempotent 204 behavior is preserved. | FR-012 |
| EC-011 | Repost or quote target is a reply post. | Flutter does not offer the action on reply cards, and AppView rejects direct API attempts without writing to the PDS. | FR-003, RULE-008 |
| EC-012 | Quote post text is empty or whitespace only. | AppView rejects the request with a validation error and no PDS record is written. | FR-004 |
| EC-013 | Repost or quote target is hidden by moderation or visibility policy. | Amplification is not offered in the UI, and AppView rejects write attempts when the target is not visible/actionable to the caller. | RULE-006, RULE-010 |
| EC-014 | Quoted target is itself a quote post. | The first-level quote preview renders the quoted post summary, but nested quote previews are not hydrated. | FR-008 |

## 15. Data / Persistence Impact
- New fields:
  - AppView response additions: home timeline feed item reason data for straight reposts; compact quote preview or quote-state data on post-shaped responses; `quoteCount` on post-shaped responses.
  - No repost lexicon additions are planned for this feature.
- Changed fields:
  - Home timeline response shape changes from a plain `PostPage` item list to a feed-item shape that can represent repost reasons while preserving post data.
  - Profile, search, thread, and post-detail responses remain post-shaped.
  - `PostResponse` may gain additive optional fields for `quoteCount` and compact quote preview state.
- Migration required:
  - Possibly new indexes on `craftsky_reposts` and/or `craftsky_posts.quote_uri` depending on selected timeline/quote-count query plans.
  - No migration is expected for reply-target validation because `craftsky_posts` already stores reply pointers.
  - No destructive migration expected.
- Backwards compatibility:
  - API changes should be additive where possible, but changing the home timeline list item shape requires coordinated Flutter updates before release.
  - PDS lexicon changes, if any, must be optional/backward-compatible.

## 16. UI / API / CLI Impact
- UI:
  - Repost/share affordance becomes a choice between straight repost and quote.
  - Reply cards do not show repost or quote actions in this first implementation.
  - Quote composer mode shows quoted-post preview.
  - Quote composer requires non-empty commentary and may include normal top-level images.
  - Post cards render quoted-post preview cards.
  - Timeline items render "reposted by" attribution for straight repost feed items.
  - Action row may render one combined repost/share count computed from separate `repostCount` and `quoteCount` values.
  - Tapping the repost/share control opens the action menu; no reposts/quotes list screen is added.
  - Straight repost success/failure updates the action state/count only; it does not optimistically insert a repost feed item.
- API:
  - Home timeline endpoint must support feed items with repost reason data.
  - Post-shaped responses must expose compact quote preview state and quote count.
  - Existing straight repost create/delete endpoints remain.
  - Existing create-post endpoint continues to accept `embed.quote`.
  - AppView validates that repost/quote targets are eligible top-level posts or project posts and are visible/actionable to the caller.
  - No new quote-specific notification API or notification type is added.
- CLI:
  - No user-facing CLI impact expected.
- Background jobs:
  - Existing Tap indexers continue indexing post and repost records.
  - No new background worker expected unless implementation chooses materialized feed rows.

## 17. Security / Privacy / Permissions
- Authentication:
  - All repost/quote write actions require existing authenticated `/v1/*` session and device ID.
- Authorization:
  - Users can only write repost and quote records to their own PDS repo through AppView.
  - Deleting a straight repost deletes only the caller's own repost record.
  - Repost and quote writes are allowed for the caller's own eligible top-level posts and project posts.
  - Repost and quote writes are rejected for reply targets and for targets hidden by current AppView visibility policy.
- Sensitive data:
  - No private data is written to PDS; repost and quote records are public atproto records.
  - Flutter never receives PDS tokens.
- Abuse cases:
  - Quote posts can be used for dogpiling. Quote-detach/postgate protections are out of scope but recorded as a future risk.
  - Repost/quote feed visibility must honor existing moderation filtering.
  - Counts should not reveal moderated or otherwise hidden repost/quote activity where AppView can evaluate the viewer's visibility policy.

## 18. Observability
- Events:
  - Existing PDS operation events for repost create/delete should remain.
  - Add or reuse post-create observability for quote-post create.
- Logs:
  - Log repost and quote write failures with existing request ID and PDS operation patterns.
  - Avoid logging raw PDS tokens or sensitive session values.
- Metrics:
  - Existing route and DB observation should cover new/changed timeline and create paths.
  - Consider separate DB operation labels for timeline feed item assembly and quote hydration.
- Alerts:
  - None required specifically for this feature.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Timeline response shape changes from posts to feed items. | Flutter models and cache code may break. | Make the contract explicit in tests and update client/server together before release. |
| RISK-002 | Straight reposts duplicate original posts confusingly in the timeline. | Poor feed UX and unclear attribution. | Define deterministic ordering/deduplication behavior in acceptance tests. |
| RISK-003 | Quote previews create N+1 lookups. | Timeline performance degrades. | Use batched quote and author hydration; add store tests for query behavior where practical. |
| RISK-004 | Quote posts bypass moderation of embedded content. | Hidden/taken-down content could be resurfaced. | Apply existing moderation policy to quoted previews and original repost subjects. |
| RISK-005 | Counts become semantically confusing. | Users cannot tell straight reposts from quote posts. | Expose `repostCount` and `quoteCount` separately. |
| RISK-006 | Lexicon changes are made casually because ADR was waived. | Future record compatibility could still be harmed before launch. | Limit lexicon work to optional backward-compatible additions and still run lexicon checklist/`just lexgen`. |
| RISK-007 | Optimistic repost cache updates conflict with later timeline refresh/indexing. | UI may show stale counts or duplicate visible state after failure. | Limit optimistic repost behavior to action state/count and extend provider tests for success/failure rollback. |
| RISK-008 | Quote abuse controls are deferred. | Product may need later anti-dogpile controls. | Record quote-detach/postgate as explicit non-goal and future work. |
| RISK-009 | Reply sharing is disabled now and later product expectations may change. | Future users may expect Bluesky-like reposting or quoting from threads. | Keep the rule in AppView validation/UI gating only, with no lexicon constraint, so it can be relaxed later. |
| RISK-010 | Combined UI count hides the distinction between reposts and quotes. | Users may not know how many quote posts exist. | Preserve separate API/model counts and defer quote-list UX until a dedicated surface is designed. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The product remains pre-live, so ADR waiver for lexicon work is acceptable. | If public PDS data already exists, lexicon process and migration risk must be revisited. |
| ASM-002 | The existing `social.craftsky.feed.post#quoteEmbed` shape is the desired Craftsky quote schema. | Requirements must change if literal Bluesky embed schema becomes required. |
| ASM-003 | Quote posts should be authored posts and not toggle the straight repost state. | Counts, UI, and delete behavior would need redesign. |
| ASM-004 | Project posts should remain unable to quote other posts in this scope, while normal quote posts may target project posts. | Validation and project profile/search semantics would need broader changes if project posts themselves become quote posts. |
| ASM-005 | Timeline can be assembled from existing `craftsky_posts`, `craftsky_reposts`, and `atproto_follows` without materialized feed tables for this scope. | Data model and performance requirements may need expansion. |
| ASM-006 | Existing moderation/visibility data is sufficient to reject non-actionable repost/quote targets and hide unavailable straight reposts. | Additional moderation state or query joins may be required before write validation can fully enforce the intended policy. |

## 21. Open Questions
- None. Grilling resolved the prior non-blocking questions for this feature scope.

## 22. Review Status
Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-07-08
Notes: User confirmed Option A, waived ADR requirement for pre-live lexicon work, and completed grilling decisions through reply-target scope, timeline item shape, count display, optimistic behavior, quote preview depth, and notification scope. Review is recommended because this is a user-visible full-stack API/UI/data-shape change, but it is not required before test design unless the user wants it.

## 23. Handoff To Test Design
- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - BR-001, BR-002, BR-003
  - FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-017, FR-018, FR-019, FR-021
  - NFR-001, NFR-002, NFR-004, NFR-005
  - RULE-001, RULE-002, RULE-004, RULE-005, RULE-006, RULE-007, RULE-008, RULE-009, RULE-010, RULE-011
- Suggested test levels:
  - Go unit/store tests for timeline feed item assembly, repost counts, quote counts, quote hydration, unavailable quote states, and moderation filtering.
  - Go handler tests for straight repost idempotency, quote-post create shape, quote non-empty validation, reply-target rejection, hidden-target rejection, timeline response shape, and API casing/error contracts.
  - Indexer tests for quote pointer and repost idempotency regressions.
  - Flutter model/API client tests for feed-item and quote-preview decoding.
  - Flutter widget/provider tests for repost/quote action choice, hidden actions on replies, quote composer preview, post-card quote rendering, combined count display, and optimistic cache rollback.
- Blocking open questions:
  - None.
