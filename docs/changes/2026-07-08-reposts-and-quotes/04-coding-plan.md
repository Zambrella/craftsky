# Coding Plan: Reposts And Quote Posts

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Reference docs read for this plan:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`

## 2. Implementation Strategy
Keep the existing atproto record model: straight reposts are `social.craftsky.feed.repost` interaction records and quote posts are normal `social.craftsky.feed.post` records with Craftsky `#quoteEmbed`. Implement the missing read model and Flutter UX in layers:

1. Change only the home timeline from post-list shape to feed-item shape.
2. Add compact quote-preview hydration and `quoteCount` to post-shaped responses.
3. Tighten write validation for repost/quote eligibility without changing lexicons.
4. Update Flutter models/providers/widgets so home timeline handles feed items while profile/search/thread/post-detail keep post-shaped models.

This fits the current codebase because the AppView already has `PostStore`, `PostResponse`, interaction rows, indexed quote pointers, route middleware, and Riverpod mutation providers. The plan extends those seams rather than adding a separate feed system or new serialization format.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView timeline store | `PostStore.ListTimeline` returns `[]*PostRow` ordered by `(indexed_at, uri)` | Return mixed `[]TimelineFeedItemRow` containing authored posts and followed straight repost activity with stable item identity. | FR-005, FR-006, FR-007, FR-017, NFR-001 | AT-003, AT-004, AT-005, IT-001, IT-004, IT-005 |
| AppView timeline handler | `TimelinePage.items` is `[]*PostResponse` | Return `{items:[{post, reason}], cursor}` where `reason` is nullable and only straight reposts use `reason.type == "repost"`. | FR-005, FR-007, FR-017, NFR-002 | AT-003, IT-005, UT-004, IT-012 |
| AppView post response | `PostResponse` has `quote` strongRef and no `quoteCount` | Add `quoteCount` and `quoteView` compact preview state while preserving existing `quote` strongRef. | FR-008, FR-009, FR-010, RULE-005, RULE-010 | AT-004, AT-007, AT-008, UT-003, UT-004, IT-003 |
| AppView write validation | Create validates strongRef shape; repost resolves target identity only | Validate share targets are visible/actionable top-level or project posts, reject replies and hidden targets, require quote commentary via existing non-empty text validation. | FR-003, FR-004, RULE-006, RULE-008, RULE-009 | AT-001, AT-002, AT-006, AT-011, IT-006, IT-007, IT-015 |
| Engagement counts | Active likes/reposts and reply counts are batched by URI | Add visible quote counts by `craftsky_posts.quote_uri`; make count filtering follow the same visibility policy where evaluable. | FR-010, RULE-011 | AT-008, IT-016 |
| Flutter timeline models | `PostPage`/`TimelineState` use `List<Post>` | Add timeline-specific `TimelinePage` and `TimelineItem`; keep `PostPage` for non-home surfaces. | FR-017 | AT-003, AT-005, IT-005, UT-006 |
| Flutter post model/rendering | `Post.quote` is a strongRef; action row count is `repostCount` | Add `quoteCount` and `QuoteView`; render quoted cards/placeholders and combined repost/share count. | FR-008, FR-009, FR-010 | AT-004, AT-007, AT-008, UT-006, UT-008 |
| Flutter repost/quote UX | Repost button directly toggles repost | Replace direct action with a share menu offering repost/unrepost and quote for eligible non-reply posts; open quote composer with preview. | FR-003, FR-004, FR-018, RULE-008 | AT-006, UT-007, MAN-001 |
| Flutter cache updates | Repost provider patches `Post` in live post caches and timeline | Patch matching posts inside timeline feed items, never insert repost feed items optimistically; quote create uses existing normal create path. | FR-019, FR-020, NFR-005 | AT-009, AT-010, UT-009, UT-010 |
| Notifications/search/profile regressions | Existing surfaces are post-shaped or notification-specific | Do not add quote notifications; keep profile/search semantics scoped to existing requirements. | FR-013, FR-014, FR-021 | REG-001, REG-002, REG-005, REG-006 |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/timeline_store.go` | Change | Replace post-only timeline query with mixed authored/repost feed-item query and cursor. | FR-005, FR-006, FR-007, NFR-001 | IT-001, IT-004 |
| `appview/internal/api/timeline.go` | Change | Define `TimelineFeedItemResponse`, `TimelineReasonRepost`, and handler hydration. | FR-005, FR-007, FR-017, NFR-002 | AT-003, IT-005, IT-012 |
| `appview/internal/api/post_store.go` | Change | Add target eligibility resolution, quote-count query, quote-preview batch hydration, and feed-item row support. | FR-008, FR-009, FR-010, RULE-006, RULE-008, RULE-011 | UT-003, IT-003, IT-007, IT-016 |
| `appview/internal/api/post_response.go` | Change | Add `QuoteCount`, `QuoteView`, `TimelineFeedItemResponse`, and response builders. | FR-008, FR-009, FR-010, FR-017 | UT-003, UT-004, IT-005 |
| `appview/internal/api/post.go` | Change | Enforce quote/repost target eligibility and build quote create request body using existing `embed.quote` translation. | FR-001, FR-002, FR-004, RULE-008, RULE-009 | AT-001, AT-002, IT-006, IT-007, IT-015 |
| `appview/internal/api/timeline_store_test.go` | Change | First failing timeline tests for followed repost activity, feed item identity, mixed pagination, hidden repost filtering. | FR-005, NFR-001, RULE-010 | IT-001, IT-004, IT-010 |
| `appview/internal/api/timeline_test.go` | Change | Handler contract tests for `{post, reason}` and home-only shape. | FR-017, NFR-002 | IT-005, IT-012 |
| `appview/internal/api/post_response_test.go` | Change | Unit tests for `quoteCount`, quote preview visible/unavailable/hidden states, no nested hydration. | FR-008, FR-009, FR-010 | UT-003, UT-004 |
| `appview/internal/api/post_test.go` | Change | Handler tests for quote request validation, reply/hidden target rejection, self-share, unrepost separation. | FR-001, FR-002, FR-004, FR-012, RULE-008, RULE-009 | IT-006, IT-007, IT-008, IT-015 |
| `appview/internal/api/post_store_test.go` | Change | Store tests for quote counts, quote hydration, moderation-aware counts. | FR-008, FR-010, RULE-011 | IT-003, IT-016 |
| `appview/migrations/000021_*.sql` | Create only if needed | Add supporting indexes only if query plans/tests show current `craftsky_reposts_*` and `craftsky_posts_quote_uri` indexes are insufficient. | NFR-003 | IT-013 |
| `app/lib/feed/models/post.dart` | Change | Add `quoteCount` and compact `QuoteView` models. | FR-008, FR-009, FR-010 | UT-006 |
| `app/lib/feed/models/timeline_item.dart` | Create | Timeline item, repost reason, and stable item key models. | FR-005, FR-007, FR-017 | AT-003, IT-005 |
| `app/lib/feed/models/timeline_page.dart` | Create | Home timeline page model with `List<TimelineItem>`. | FR-017 | IT-005 |
| `app/lib/feed/models/timeline_state.dart` | Change | Store `List<TimelineItem>` and dedupe by item key. | FR-017, FR-019 | AT-005, AT-009 |
| `app/lib/feed/data/post_api_client.dart` | Change | `listTimeline` decodes `TimelinePage`; `createPost` accepts optional quote ref. | FR-002, FR-017 | AT-002, IT-005 |
| `app/lib/feed/data/post_repository.dart` | Change | Repository returns `TimelinePage` for home timeline and accepts quote ref on create. | FR-002, FR-017 | UT-010 |
| `app/lib/feed/providers/timeline_provider.dart` | Change | Page, append, replace, remove, and update posts inside feed items by stable item key and post URI. | FR-017, FR-019 | AT-005, AT-009 |
| `app/lib/feed/providers/create_post_provider.dart` | Change | Pass quote ref and keep normal post-create cache behavior. | FR-020 | AT-010, UT-010 |
| `app/lib/feed/providers/toggle_repost_post_provider.dart` | Change | Patch matching posts inside timeline items without inserting feed items. | FR-019, NFR-005 | AT-009, UT-009 |
| `app/lib/feed/widgets/post_card.dart` | Change | Add quote preview rendering, combined repost/share count, and share menu callback shape. | FR-003, FR-008, FR-010, FR-018 | AT-006, AT-007, AT-008 |
| `app/lib/feed/widgets/timeline_item_card.dart` | Create | Render optional "reposted by" attribution around `PostCard`. | FR-005, FR-007 | AT-003 |
| `app/lib/feed/widgets/post_composer_sheet.dart` | Change | Add quote composer mode with quoted-post preview and non-empty text enforcement. | FR-004 | AT-002, AT-006 |
| `app/lib/feed/pages/feed_page.dart` | Change | Render `TimelineItemCard` and route action menu results to repost/unrepost or quote composer. | FR-003, FR-005, FR-017 | AT-003, AT-006 |
| `app/lib/l10n/app_en.arb` and generated localization files | Change | Add labels for quote action, share menu, repost attribution, and quote placeholders. | BR-004, FR-009 | AT-006, AT-007, MAN-001 |
| `app/test/feed/*` target files from `02-acceptance-tests.md` | Change/Create | Flutter model, API client, provider, and widget tests. | Multiple | UT-006 through UT-010, AT-006 through AT-010 |

## 5. Services, Interfaces, And Data Flow
### AppView Data Contracts
Use explicit feed-item responses for the home timeline only:

```text
type TimelineFeedItemResponse struct {
    ItemKey string                 `json:"itemKey"`
    Post    *PostResponse          `json:"post"`
    Reason  *TimelineReasonRepost  `json:"reason,omitempty"`
}

type TimelineReasonRepost struct {
    Type      string     `json:"type"` // "repost"
    By        PostAuthor `json:"by"`
    URI       string     `json:"uri"`
    CID       string     `json:"cid,omitempty"`
    IndexedAt time.Time  `json:"indexedAt"`
    CreatedAt time.Time  `json:"createdAt"`
}
```

Keep `PostResponse.quote` as the existing strongRef for compatibility and add a separate compact preview field:

```text
type PostResponse struct {
    ...
    RepostCount int        `json:"repostCount"`
    QuoteCount  int        `json:"quoteCount"`
    Quote       *StrongRef `json:"quote"`
    QuoteView   *QuoteView `json:"quoteView,omitempty"`
}

type QuoteView struct {
    State string             `json:"state"` // "visible", "unavailable", "hidden"
    Post  *QuotePreviewPost  `json:"post,omitempty"`
}

type QuotePreviewPost struct {
    URI string `json:"uri"`
    CID string `json:"cid"`
    Text string `json:"text"`
    Author PostAuthor `json:"author"`
    Images []PostImageView `json:"images,omitempty"`
    Project *Project `json:"project,omitempty"`
    CreatedAt time.Time `json:"createdAt"`
}
```

`QuotePreviewPost` intentionally does not include another `QuoteView`; quote-of-quote targets stop at one level.

### Timeline Store Flow
Introduce a store row that carries item metadata plus the original post:

```text
type TimelineFeedItemRow struct {
    ItemKind string // "post" or "repost"
    ItemKey string
    ActivityAt time.Time
    ActivityID string
    Post *PostRow
    Repost *RepostReasonRow
}
```

Use `itemKey` as a stable API/client identity:

- Authored post item: `post:<post-uri>`
- Straight repost item: `repost:<repost-uri>`

Use cursor keys:

- `activityAt`: post indexed time for authored posts, repost indexed time for repost items.
- `itemKey`: stable tie-breaker.

Query shape:

```text
WITH eligible_authors AS (
  SELECT $viewerDID AS did
  UNION
  SELECT subject_did FROM atproto_follows WHERE did = $viewerDID
), feed AS (
  SELECT 'post' AS item_kind, 'post:' || p.uri AS item_key,
         p.indexed_at AS activity_at, p.uri AS activity_id, p.uri AS post_uri,
         NULL... repost fields
  FROM craftsky_posts p
  JOIN eligible_authors a ON a.did = p.did
  WHERE top-level visible post predicate

  UNION ALL

  SELECT 'repost', 'repost:' || r.uri,
         r.indexed_at, r.uri, r.subject_uri,
         r.uri, r.cid, r.did, r.created_at, r.indexed_at
  FROM craftsky_reposts r
  JOIN eligible_authors a ON a.did = r.did
  JOIN craftsky_posts p ON p.uri = r.subject_uri
  WHERE r.deleted_at IS NULL
    AND p is visible top-level target
)
SELECT feed metadata + postSelectColumns + reposter profile columns
WHERE cursor seek on (activity_at, item_key)
ORDER BY activity_at DESC, item_key DESC
LIMIT limit + 1
```

This preserves separate chronological repost activity and allows the same original post to appear once as authored content and multiple times as separate repost items.

### Quote Hydration Flow
Add a batch method instead of one lookup per quote:

```text
func (s *PostStore) QuoteViews(ctx, viewerDID string, refs []ResponseStrongRef) (map[string]*QuoteView, error)
```

The key can be quote URI. The method should:

- Initialize every requested URI to `State: "unavailable"`.
- Query visible `craftsky_posts` rows for requested quote URIs using `postVisibleModerationPredicate`.
- Hydrate authors using existing joined `bluesky_profiles` columns and `HandleResolver` only if needed by current response-building pattern.
- Return `State: "visible"` for visible rows.
- Return `State: "hidden"` only when the store can distinguish an indexed row hidden by moderation from a missing/unindexed row without leaking extra detail beyond approved policy; otherwise `unavailable` is acceptable for missing/unindexed and `hidden` for explicit moderation hits.
- Do not hydrate nested quote targets.

The handler should collect quote refs from all post responses in a page/detail response, call this once, and attach the results.

### Quote Counts
Add:

```text
func (s *PostStore) CountVisibleQuotes(ctx context.Context, viewerDID string, postURIs []string) (map[string]int, error)
```

Base query counts root/top-level quote posts where `quote_uri = ANY($1)` and the quote post itself passes `postVisibleModerationPredicate`. This aligns with "counts should not reveal hidden records where AppView can apply policy." Add the result to `EngagementSummary.QuoteCount`.

### Write Eligibility
Replace `ResolvePostTarget` usage for repost and quote writes with an eligibility-aware method for share actions:

```text
type ShareTargetRef struct {
    URI string
    CID string
    IsReply bool
    IsProject bool
}

func (s *PostStore) ResolveShareTarget(ctx, viewerDID, did, rkey string) (*ShareTargetRef, error)
```

The query should use current visibility moderation predicate and return only visible indexed targets. The handler should reject:

- Missing/unindexed target: `404 post_not_found` or existing equivalent.
- Reply target: `422 validation_failed`, field `target`.
- Hidden/non-actionable target: `404 post_not_found` if hidden by visibility predicate, or `422 validation_failed` if the implementation can safely distinguish actionable policy without leaking.

Apply this method in:

- `RepostPostHandler`
- `CreatePostHandler` when `req.Embed.Quote != nil`

Keep project post creation rejecting `project + quote`; normal quote posts may target project posts.

## 6. State, Providers, Controllers, Or DI
### Flutter Model And Provider Graph
Keep `PostPage` for non-home lists. Add timeline-specific page/state:

```text
PostApiClient.listTimeline -> TimelinePage
ApiPostRepository.listTimeline -> TimelinePage
timelineProvider -> TimelineState(items: List<TimelineItem>)
FeedPage -> TimelineItemCard -> PostCard
```

Model sketch:

```text
class TimelinePage {
  final List<TimelineItem> items;
  final String? cursor;
}

class TimelineItem {
  final String itemKey;
  final Post post;
  final RepostReason? reason;
}

class RepostReason {
  final String type; // "repost"
  final PostAuthor by;
  final AtUri uri;
  final Cid? cid;
  final DateTime indexedAt;
  final DateTime createdAt;
}
```

`TimelineState` should dedupe and append by `itemKey`, not `post.uri`, so multiple reposts of the same original remain distinct. Cache update helpers should patch every item whose `post.uri` matches the updated post:

```text
replacePostInTimeline(post):
  items = [item.post.uri == post.uri ? item.copyWith(post: post) : item]

prependLiveTimelineCache(post):
  only for normal post create
  itemKey = "post:${post.uri}"
```

`createPostProvider` should accept `quoteTarget` or `quote` and call `repo.create(..., quote: ref)`. For quote posts, it should still use the normal top-level post cache path because the created quote is a post.

`toggleRepostPostProvider` should keep the existing optimistic state/count patch and rollback, but the timeline helper must not create new repost `TimelineItem`s.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces
### PostCard And Timeline Rendering
Change `PostCard` so the share control opens a menu callback instead of direct repost-only behavior. A simple API is:

```text
PostCard(
  post: post,
  onShare: eligible ? () => showShareMenu(post) : null,
  showRepostAction: post.reply == null,
)
```

The share menu should offer:

- `Repost` or `Unrepost`, depending on `viewerHasReposted`.
- `Quote`.

The combined visible count on the share control should be `post.repostCount + post.quoteCount`; selection color remains driven by `viewerHasReposted`.

Create `TimelineItemCard`:

```text
if item.reason?.type == "repost":
  render compact "reposted by <display name/handle>" row
render PostCard(post: item.post, ...)
```

### Quote Preview UI
Add a reusable compact quote preview widget, likely inside `post_card.dart` first unless it grows:

- Visible state: author, handle, short text, optional image/project summary.
- Hidden/unavailable state: a stable placeholder string from l10n.
- No nested card rendering.

Use existing `ProjectCard` with a compact/summary variant for project previews if it fits; otherwise render only project title/craft summary to avoid card nesting.

### Quote Composer
Extend `showPostComposerSheet` and `PostComposerSheet`:

```text
Future<Post?> showPostComposerSheet(context, {Post? replyTarget, Post? quoteTarget})
```

Rules:

- `replyTarget` and `quoteTarget` are mutually exclusive.
- Quote mode shows `_QuoteTargetPreview`.
- Quote mode allows normal top-level images.
- Quote mode requires non-empty text through the existing `trimmedText.isNotEmpty` gate.
- Submit passes `quote: PostRef(uri: quoteTarget.uri, cid: quoteTarget.cid)`.

### Routes
No new named route is required. Quote composer opens as the existing fullscreen dialog. Post thread route remains unchanged.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Invalid timeline cursor | Keep standard `400 invalid_cursor` envelope. | NFR-001, NFR-002 | IT-004, IT-012 |
| Followed user reposts hidden/unavailable subject | Omit the repost feed item entirely. | FR-009, RULE-010 | AT-007, IT-010 |
| Quote target hidden/unavailable after quote exists | Keep quote post visible if otherwise eligible; attach `quoteView.state` placeholder. | FR-009, RULE-010 | AT-007, IT-003, IT-010 |
| Quote target is itself a quote | Hydrate only first-level preview; do not set nested `quoteView`. | FR-008 | AT-004, UT-003, IT-003 |
| User tries to repost/quote reply | Hide action in Flutter; reject direct API write with validation error and no PDS write. | FR-003, RULE-008 | AT-006, IT-007 |
| User tries to quote with empty text | Existing create validation rejects empty/whitespace commentary before PDS write. Trim handling should be explicit in backend validation if not already. | FR-004 | AT-002, UT-001 |
| User shares own eligible post/project | Allow same as any other eligible target. | RULE-009 | AT-011, IT-015 |
| Repost request duplicates active repost | Preserve existing idempotent response. | RULE-002 | AT-001, IT-006 |
| Unrepost with quote posts present | Delete only active straight repost. Quote posts remain indexed. | FR-012 | AT-008, IT-008 |
| Repost optimistic action fails | Roll back post fields in all live caches and emit existing error messaging. | FR-019, NFR-005 | AT-009, UT-009 |
| Quote create fails | Use existing create-post error state and messaging; no quote-specific optimistic item remains. | FR-020, NFR-005 | AT-010, UT-010 |
| No quote-specific notification | Leave notification type set unchanged. | FR-021 | REG-005 |

## 9. Test Implementation Plan
| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `appview/internal/api/timeline_store_test.go` | Alice follows Bob; Bob reposts Carol; seed reposter profile and original post. | `ListTimeline` returns only posts or has no repost reason/item type. |
| 2 | IT-004 | `appview/internal/api/timeline_store_test.go` | Mixed authored posts/reposts with tied `indexed_at`. | Cursor still expects `indexedAt/uri` and cannot page mixed item keys. |
| 3 | IT-005 | `appview/internal/api/timeline_test.go` | Fake timeline store returns authored and repost item rows. | Handler JSON items are bare posts, not `{post, reason}`. |
| 4 | UT-004 | `appview/internal/api/post_response_test.go` | Build post and repost reason responses with counts. | `quoteCount` and reason response fields are absent. |
| 5 | UT-006 | `app/test/feed/models/post_test.dart` | Decode post JSON with `quoteCount` and quote preview. | `PostMapper` rejects or drops new fields. |
| 6 | UT-012 / IT-012 | `appview/internal/api/*_test.go` | Marshal new response structs and validation errors. | New fields may be missing camelCase tags or error paths. |
| 7 | IT-003 | `appview/internal/api/post_store_test.go` | Quote rows for visible, hidden, missing, quote-of-quote targets. | Store only returns strongRef, no preview state. |
| 8 | IT-010 | `appview/internal/api/timeline_store_test.go` | Hidden repost subject and hidden quote target. | Repost may leak hidden subject or quote preview may fail containing post. |
| 9 | IT-016 | `appview/internal/api/post_store_test.go` | Visible and hidden reposts/quotes for one subject. | Counts include hidden quote/repost activity. |
| 10 | IT-006 | `appview/internal/api/post_test.go` | Existing repost handler fake PDS/store. | Eligibility method not wired or duplicate behavior regresses. |
| 11 | IT-007 | `appview/internal/api/post_test.go` | Quote create targeting top-level, project, reply, hidden, missing. | Create handler accepts reply/hidden target or writes before validation. |
| 12 | IT-008 | `appview/internal/api/post_test.go` | Viewer has quote and active repost for same subject. | Unrepost may not prove quote rows remain independent. |
| 13 | IT-009 | `appview/internal/api/post_test.go` | User creates multiple quote posts for same subject. | Handler/store might accidentally dedupe quotes. |
| 14 | IT-015 | `appview/internal/api/post_test.go` | Viewer targets own eligible post/project. | Eligibility logic may reject self-share. |
| 15 | UT-007 | `app/test/feed/widgets/post_card_test.dart` | Top-level and reply cards, share control taps. | PostCard directly calls repost or shows actions on replies. |
| 16 | UT-008 | `app/test/feed/widgets/post_card_test.dart` | `repostCount: 2`, `quoteCount: 3`. | UI shows only repost count. |
| 17 | UT-009 | `app/test/feed/providers/toggle_post_interactions_provider_test.dart` | Timeline with multiple items for same post URI. | Cache helper dedupes by post URI or inserts/removes rows incorrectly. |
| 18 | UT-010 | `app/test/feed/providers/create_post_provider_test.dart` | Quote create through provider. | Provider lacks quote argument or special-cases timeline insertion incorrectly. |
| 19 | AT-006 through AT-010 | Flutter widget/provider/page tests | Feed page, quote composer, quote preview, optimistic failure fixtures. | End-to-end UI flows are missing. |
| 20 | REG-001 through REG-007 | Existing regression suites | Profile/search/project/notification/route fixtures. | Accidental surface changes outside home timeline. |

Focused commands:

```sh
just dev-d
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimeline|TestBuildPostResponse|TestRepostPost|TestCreatePost|TestPostStore' ./internal/api
cd app && flutter test test/feed/models/post_test.dart test/feed/data/post_api_client_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/widgets/post_card_test.dart
```

Full verification remains:

```sh
just test
just app-test
just app-analyze
```

## 10. Sequencing And Guardrails
- First TDD step: Write `IT-001` in `appview/internal/api/timeline_store_test.go` for a followed straight repost feed item with reposter attribution.
- Dependencies between work items:
  - Backend feed-item store shape before timeline handler JSON shape.
  - Backend response contract before Flutter timeline models.
  - Quote preview/quote count backend before Flutter quoted-card rendering.
  - Flutter timeline model migration before provider/page cache updates.
  - Repost/quote menu after action callbacks can route to repost provider or quote composer.
- Guardrails:
  - Do not change lexicon files unless implementation uncovers a concrete need.
  - Do not add an ADR for this feature; the requirements explicitly waive it for pre-live work.
  - Do not change profile, search, thread, or post-detail response shape to feed items.
  - Do not store or expose PDS OAuth tokens in Flutter.
  - Do not optimistically insert straight repost feed items.
  - Do not recursively hydrate quote previews.
  - Keep `/v1/*` JSON camelCase and error envelopes standard.
- Out of scope:
  - Repost/quote list screens.
  - Reply reposting/quoting.
  - Quote-detach/postgate/anti-dogpile controls.
  - Algorithmic ranking or deduplication of duplicate repost activity.
  - New quote-specific notification type.
  - Private reposts or private quote posts.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact quote placeholder distinction between `hidden` and `unavailable` depends on whether the store can distinguish moderated rows without leaking extra detail. | Flutter needs stable states; product wants both hidden/unavailable semantics. | Prefer `hidden` for explicit moderation hits and `unavailable` for missing/unindexed/deleted; document fallback in tests if the store cannot safely distinguish. |
| CPQ-002 | Non-blocking | N+1 prevention lacks known query-count instrumentation. | Performance regression could slip through if implementation hydrates one quote/profile per item. | Use batch store methods and add bounded-query tests where practical; otherwise make query count visible in test helper or document query plan in implementation review. |
| CPQ-003 | Non-blocking | Generated Dart mapper/localization files must stay in sync with model and ARB changes. | Flutter tests or analysis will fail if generation is skipped. | Implementation stage should run the repo's established build_runner/gen-l10n command if required by local workflow, then `just app-test` and `just app-analyze`. |
| CPQ-004 | Non-blocking | Existing notification behavior includes repost notifications; quote-specific notifications are disallowed, but quote posts may still pass through generic post-create paths. | Accidental new quote notification type would violate scope. | Keep notification enum unchanged and add `REG-005`. |

## 12. Handoff To TDD Builder
- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001` in `appview/internal/api/timeline_store_test.go`.
- Focused command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore' ./internal/api`
- Notes:
  - Start backend-first because Flutter depends on the new timeline and quote-preview wire contracts.
  - Keep the first implementation thin: store row, response type, handler shape, then Flutter model decode.
  - Revisit migrations only after query plans prove current indexes are insufficient.
  - Carry document-review findings DR-001 through DR-003 into the implementation plan.
