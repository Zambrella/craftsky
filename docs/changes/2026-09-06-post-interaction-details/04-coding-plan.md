# Coding Plan: Post Interaction Details

## 1. Inputs

- Requirements: `01-requirements.md` (Reviewed, 2026-09-06)
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (Approved with notes, Medium risk)
- Governing API documents: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` and `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Architecture constraints: authenticated `/v1/*` JSON reads through AppView, camelCase bodies, standard error envelopes, opaque cursors, no PDS read or credential handoff, and no lexicon change

## 2. Implementation Strategy

Implement the feature from AppView storage outward so the first red-green cycles establish list membership and count policy before UI work depends on them.

AppView will keep the three public GET resources separate while sharing internal target resolution, eligibility predicates, and cursor validation. Likes and Reposts will use one account-interaction query path parameterized by a closed interaction kind. Quotes will use a separate post query and the existing post response hydration pipeline. The authoritative viewer-aware predicates used by these lists will also replace the narrower like, repost, and quote predicates inside engagement summaries so the detail model counts and list membership cannot drift for unchanged data. The cursor will carry interaction kind, target URI, `createdAt`, and interaction/post URI; stores will fetch `limit + 1`, trim to `limit`, and emit a cursor only when another row exists.

Account-page SQL will hydrate profiles, viewer relationship flags, customisation inputs, and cached current handles set-wise. A missing required handle fails the complete request as `identity_unavailable`; handlers will not perform one directory call per item. Quote rows will use `postSelectColumns`, batch engagement summaries, batch relationship state, one handle resolution per distinct author through the existing resolver/cache path, and `attachQuoteViews`. Blocked/hidden/language-ineligible quotes are omitted; muted quotes remain and are shaped as revealable `SurfaceQuote` placeholders.

Flutter will extend the existing post repository rather than add another client boundary. One generated `AsyncNotifier` family will serve Likes/Reposts through an enum mode and `ProfileAccountPage`; a separate generated `AsyncNotifier` family will serve Quotes through `PostPage`. Both use immutable cursor-accumulating state, active-account operation guards, refresh generation protection, retained-data load-more errors, and invalid-cursor restart behavior. Likes/Reposts share one page and extracted account-row widget; Quotes has its own page and normal `PostCard` composition. A dedicated summary widget remains owned by `PostThreadPage`. `PostCard` receives only an optional response-liker callback and renders it in a separate first menu group when the card is a liked reply.

No migration is planned. Existing indexes are accepted only after the focused query-plan tests demonstrate indexed subject/quote scans. If they do not, implementation stops for a separately reviewed reversible migration rather than silently adding one.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView target policy | `ReadEligiblePostsByURI`, `RequiredContextStates`, moderation/block SQL fragments | Resolve a target as viewer-available; allow replies only for Likes and return indistinguishable `post_not_found` otherwise | FR-008, FR-009, FR-010 | IT-001, IT-006, IT-011 |
| AppView account interactions | Inline pgx stores and `ProfileAccountPage` response | Shared closed-kind Likes/Reposts list path with one eligibility seam, total, set-wise account/identity hydration, and newest-first seek pagination | FR-003, FR-004, FR-005, FR-010, NFR-001, NFR-002, RULE-001, RULE-002, RULE-003 | UT-002, UT-003, UT-008, IT-001, IT-002, IT-004, IT-008, IT-010, IT-013 |
| AppView quote posts | Existing post select columns, response builders, relationship shaping, quote-preview hydration | List quote posts by target URI, filter with normal post/block/language policy, retain muted placeholders, and return a plain `PostPage` shape | FR-006, FR-010, NFR-001, NFR-002, RULE-001, RULE-002, RULE-003 | IT-003, IT-004, IT-008, IT-010, IT-013, REG-003, REG-005 |
| Engagement summaries | `EngagementSummaries` calls separate active-only count helpers | Route all like/repost/quote counts through the same viewer-aware eligibility fragments as list membership; pass authoritative content languages where post hydration already occurs | FR-010, FR-011 | IT-004, IT-012, AT-008 |
| AppView HTTP and routes | stdlib handlers, envelope helpers, route bundles, policy catalogue/inventory | Add three method-distinct GET registrations and standard validation/error/observability behavior | FR-008, FR-009, NFR-004, NFR-005 | IT-005, IT-006, IT-007, IT-014, REG-004 |
| Flutter data boundary | `PostApiClient` -> `ApiPostRepository` -> `PostRepository` | Add typed list methods reusing `ProfileAccountPage` and `PostPage` | FR-003, FR-004, FR-005, FR-006, FR-008, NFR-004 | UT-005, UT-007, IT-009, REG-004 |
| Flutter list state | Generated Riverpod `AsyncNotifier` families and active-account guards | Add shared account-list and separate quote-list providers with retained-data pagination errors and refresh fencing | FR-007, NFR-002, NFR-004 | UT-004, AT-003, AT-004, AT-006 |
| Flutter destinations | Authenticated typed detail routes, `AutoPaginatedListView`, `CraftskyEmptyState`, profile modal, `PostCard` | Add account and quote pages, three routes, all loading/empty/error/pagination states, and existing navigation/actions | FR-002 through FR-007, NFR-003 | AT-002, AT-003, AT-004, AT-006, AT-007, AT-009 |
| Detail summary | Root `PostCard` in `PostThreadPage` | Add an immediately-following nonzero-only wrapping summary sourced directly from the rendered root `Post` | FR-001, FR-011, FR-012, RULE-004 | UT-001, AT-001, AT-002, AT-008, REG-001, REG-002, REG-006 |
| Response menu | `PostCard` grouped Craftsky context menu | Add optional `View likes` callback, guarded to positive-count replies, as its own first non-destructive group | FR-013, RULE-004, RULE-005 | UT-006, AT-005, AT-007, REG-002, REG-006 |
| Localisation and accessibility | English ARB, generated localisations, Material/Craftsky semantics | Add labels, pluralized count links, titles, empty/error copy, and semantic destinations | NFR-003 | AT-001, AT-005, AT-006, AT-007, MAN-001, MAN-002 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/post_interaction_lists.go` | Create | Read-handler interfaces, three handlers, request validation, response hydration, and error mapping | FR-008, FR-009, FR-010, NFR-004, NFR-005 | IT-005, IT-006, IT-014, REG-004 |
| `appview/internal/api/post_interaction_lists_store.go` | Create | Target resolution, account/quote list SQL, shared eligibility fragments, bound cursor encode/decode, and `limit + 1` exhaustion detection | FR-003, FR-004, FR-005, FR-006, FR-009, FR-010, NFR-001, NFR-002, RULE-001 through RULE-003 | UT-002, UT-003, UT-008, IT-001 through IT-004, IT-008, IT-010, IT-011, IT-013 |
| `appview/internal/api/post_store.go` | Change | Add interaction-list row/filter types and reusable eligibility contracts; keep typed atproto identifiers at handler boundaries | FR-003 through FR-006, FR-010 | IT-001 through IT-004 |
| `appview/internal/api/post_engagement_store.go` | Change | Replace active-only interaction counts with shared eligible counts and authoritative language-aware quote counts | FR-010, FR-011 | IT-004, IT-012, AT-008 |
| `appview/internal/api/post_read.go`, `post_conversation.go`, `post_author_feed.go`, `timeline.go`, `search.go`, `notifications.go`, `saved_post.go` and their narrow capability fakes | Change as required by the explicit engagement-summary signature | Supply the current viewer's authoritative content-language selection to post count hydration; preserve existing list ordering and presentation | FR-010, FR-012 | IT-004, IT-012, REG-003, REG-005, REG-006 |
| `appview/internal/routes/routes_scheduled_post.go` | Change | Register GET Likes/Reposts/Quotes beside existing method-distinct writes and pass language/identity dependencies | FR-008, NFR-004 | IT-007, REG-004 |
| `appview/internal/routes/policy.go`, `inventory_test.go`, `routes_test.go`, `architecture_test.go` | Change | Classify all three as current-member, read-rate, no-body routes and prove middleware/no-PDS behavior | FR-008, NFR-004 | IT-007, REG-004 |
| `appview/internal/api/post_interaction_lists_store_test.go` | Create | Cohesive cursor, membership, policy, pagination, count-alignment, hydration-bound, and query-plan suite | FR-003 through FR-006, FR-009, FR-010, NFR-001, NFR-002, RULE-001 through RULE-003 | UT-002, UT-003, UT-008, IT-001 through IT-004, IT-008, IT-010 through IT-013 |
| `appview/internal/api/post_interaction_lists_handler_test.go` | Create | Cohesive wire, validation, unavailable-target, identity-failure, and observability suite | FR-008, FR-009, NFR-005 | IT-005, IT-006, IT-014 |
| `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, `api_post_repository.dart` | Change | Add `listLikes`, `listReposts`, and `listQuotes` methods with exact paths and optional query parameters | FR-003 through FR-006, FR-008, NFR-004 | UT-005, IT-009, REG-004 |
| `app/lib/feed/models/post_interaction_list_state.dart` | Create | Immutable account and quote page accumulation state; no wire model duplication | FR-003 through FR-007 | UT-004, AT-003, AT-004, AT-006 |
| `app/lib/feed/providers/post_interaction_lists_provider.dart` plus generated `.g.dart` | Create / Generate | Shared account `AsyncNotifier` family and separate Quotes `AsyncNotifier` family | FR-003 through FR-007, NFR-002, NFR-004 | UT-004, AT-003, AT-004, AT-006 |
| `app/lib/feed/pages/post_interaction_accounts_page.dart` | Create | Likes/Reposts account destination, authoritative title total, profile presentation, and complete states | FR-003, FR-004, FR-005, FR-007 | AT-003, AT-006, AT-007 |
| `app/lib/feed/pages/post_quotes_page.dart` | Create | Quotes post destination with plain title and normal post-card callbacks | FR-006, FR-007, FR-012 | AT-004, AT-006, AT-007, REG-003, REG-005 |
| `app/lib/profile/widgets/profile_account_list_tile.dart` | Create | Extract existing account summary row/profile-opening presentation for Follow and interaction lists | BR-003, FR-005 | AT-003, AT-007 |
| `app/lib/settings/pages/follow_list_page.dart` | Change | Consume the extracted account row without behavior changes | BR-003 | REG-006 |
| `app/lib/feed/widgets/post_interaction_summary.dart` | Create | Fixed-order, nonzero-only, wrapping accessible summary links | FR-001, RULE-004, NFR-003 | UT-001, AT-001, AT-002, AT-007 |
| `app/lib/feed/widgets/post_card.dart` | Change | Add the response-only liker action group; do not alter mutation controls or other card layouts | FR-012, FR-013, RULE-004, RULE-005 | UT-006, AT-005, AT-008, REG-001, REG-002 |
| `app/lib/feed/pages/post_thread_page.dart` | Change | Insert root summary and pass response-specific liker callbacks using each response's own author DID/rkey | FR-001, FR-002, FR-011, FR-013 | AT-001, AT-002, AT-005, AT-008, REG-006 |
| `app/lib/router/route_locations.dart`, `router.dart` plus generated `router.g.dart` | Change / Generate | Add typed authenticated Likes/Reposts/Quotes detail routes without `$extra` | FR-002 | AT-002, AT-005, AT-009 |
| `app/lib/l10n/app_en.arb` plus generated localisations | Change / Generate | Add pluralized summary labels, menu label, page titles, empty states, unavailable state, and semantic hints | FR-001, FR-005, FR-006, FR-007, FR-013, NFR-003 | AT-001, AT-005 through AT-007 |
| `app/test/feed/fakes/fake_post_repository.dart` | Change | Add programmable callbacks for all three list methods | NFR-006 | UT-004, UT-005, AT-003, AT-004, AT-006 |
| Existing/new Flutter test targets from Section 9 | Create / Change | Cover data, provider, pages, routing, summary, menu scope, accessibility, and regressions | NFR-006 | AT-001 through AT-009, UT-001, UT-004 through UT-007, IT-009, REG-001 through REG-006 |

No changes are planned under `lexicon/`, `appview/migrations/`, dependency manifests, CLI modules, or background jobs.

## 5. Services, Interfaces, And Data Flow

### AppView read contracts

Use closed internal kinds rather than accepting arbitrary table names from handlers. Dynamic SQL selection is limited to constants selected by the closed kind.

```text
type PostInteractionKind = likes | reposts | quotes

type PostInteractionTarget {
  URI: syntax.ATURI
  Author: syntax.DID
  IsReply: bool
}

type PostInteractionCursor {
  Kind: PostInteractionKind
  Target: syntax.ATURI
  CreatedAt: time.Time
  URI: syntax.ATURI
}

ResolveInteractionTarget(ctx, viewer, owner, rkey, kind) -> target | ErrPostNotFound
ListInteractionAccounts(ctx, viewer, target, kind, limit, cursor) -> rows, nextCursor, total
ListQuotePosts(ctx, viewer, target, contentLanguages, limit, cursor) -> rows, nextCursor
EligibleEngagementSummaries(ctx, viewer, contentLanguages, postURIs) -> summaries
```

`ResolveInteractionTarget` uses the existing current-member, terminal-owner, moderation, symmetric viewer/author block, and reply-context seams. A missing or ineligible target is always `ErrPostNotFound`. Likes accepts a root, comment, or nested reply. Reposts/Quotes additionally require `target.IsReply == false`; failure remains `ErrPostNotFound`, not validation success with an empty page.

The account eligibility fragment is authoritative for both list rows and like/repost counts:

```text
active interaction
+ actor has current craftsky_profiles row
+ actor and target author are non-terminal
+ actor has no active account hide/takedown
+ no viewer <-> actor block
+ no actor <-> target-author block
= eligible account interaction

viewer mute and warning state are selected for presentation, not exclusion.
```

The quote eligibility fragment is authoritative for both quote rows and quote counts:

```text
active indexed top-level post with quote_uri == target
+ author/target/viewer owners non-terminal
+ no account/post hide or takedown
+ no viewer <-> quote-author block
+ no quote-author <-> target-author block
+ authored-content language predicate
= eligible quote post

viewer mute is retained and later shaped as a revealable SurfaceQuote placeholder.
```

Likes/Reposts use one eligibility CTE per request so page selection and `totalCount` consume identical predicates, including when the first page is empty. The query joins `craftsky_profiles`, `bluesky_profiles`, profile customisation data used by `BuildProfileAccountSummary`, and `atproto_identity_cache` set-wise. Missing/invalid handles become a typed store error mapped to `502 identity_unavailable`; rows are never silently dropped after total calculation.

Quotes fetch `limit + 1` `PostRow` values with `postSelectColumns`, trim the sentinel, then hydrate the retained page in bounded batches:

```text
rows
 -> EligibleEngagementSummaries(rows.uri)
 -> RelationshipStates(distinct rows.did)
 -> resolve handles once per distinct DID through existing cached resolver
 -> buildPostResponse + applyEngagementSummary
 -> ApplyPostRelationshipPolicy(..., SurfaceQuote) for muted authors
 -> attachQuoteViews(all items)
 -> PostPage-compatible {items, cursor?}
```

The cursor encoder/decoder validates all fields and exact kind/target equality before SQL runs. Seek comparison and ordering are exactly `(created_at, uri) < (cursor.createdAt, cursor.uri)` and `ORDER BY created_at DESC, uri DESC`. Cursors are omitted after trimming when no sentinel row exists.

### HTTP contracts

```text
GET /v1/posts/{did}/{rkey}/likes?limit=&cursor=
 -> {items: ProfileAccountSummary[], totalCount: int, cursor?: string}

GET /v1/posts/{did}/{rkey}/reposts?limit=&cursor=
 -> {items: ProfileAccountSummary[], totalCount: int, cursor?: string}

GET /v1/posts/{did}/{rkey}/quotes?limit=&cursor=
 -> {items: Post[], cursor?: string}
```

Handlers parse `syntax.DID` and `syntax.RecordKey` once at the boundary, read the current viewer DID from middleware, use existing bounded `parseLimit`, reject invalid cursors as `400 invalid_cursor`, map unavailable targets to `404 post_not_found`, map identity hydration failure to `502 identity_unavailable`, and map unexpected failures to `500 internal_error`. Responses are bare camelCase JSON; final `cursor` is omitted. Request logs/metrics include only bounded operation name, result, limit, row count, and `has_cursor`, never target/actor identifiers or cursor contents.

### Flutter repository contracts

```text
PostRepository.listLikes(Did, RecordKey, {String? cursor, int? limit})
  -> Future<ProfileAccountPage>
PostRepository.listReposts(Did, RecordKey, {String? cursor, int? limit})
  -> Future<ProfileAccountPage>
PostRepository.listQuotes(Did, RecordKey, {String? cursor, int? limit})
  -> Future<PostPage>
```

`PostApiClient` uses Dio's existing authenticated interceptors and `unwrapApi`; it sends only `/v1/` GETs and omits absent query parameters. No interaction-record DTO is introduced.

## 6. State, Providers, Controllers, Or DI

### Provider graph

```text
postApiClientProvider
 -> postRepositoryProvider (keepAlive)
    -> postInteractionAccountsProvider(did, rkey, kind)
       kind: PostInteractionAccountKind.likes | reposts
    -> postQuotesProvider(did, rkey)

authSessionProvider/sessionRegistryProvider
 -> account operation guard captured before loadMore/refresh

activeContentLanguagePolicyProvider
 -> postQuotesProvider invalidation/refetch
```

`PostInteractionAccounts` is an `AsyncNotifier<PostInteractionAccountsState>` family keyed by `Did`, `RecordKey`, and the closed account kind. State contains immutable `items`, `cursor`, and authoritative `totalCount`. The initial title reads only the route kind; after `AsyncData`, it includes `totalCount`, including zero.

`PostQuotes` is an `AsyncNotifier<PostQuotesState>` family keyed by `Did` and `RecordKey`. State contains immutable `items` and `cursor`; no total is added. It watches active content-language policy and supports `replace(Post)`/`remove(AtUri)` only as needed to preserve normal visible card mutation/delete behavior.

Both notifiers:

- Fetch the first page in `build`.
- Set `AsyncLoading` while retaining previous data for continuation requests.
- Capture the exact cursor before load-more and retry the same cursor after failure.
- Deduplicate appended items by stable DID for account lists and URI for quotes as a defensive UI guard; server traversal remains authoritative.
- Restart from page one after `invalid_cursor` and discard the old traversal only after restart succeeds.
- Use `captureActiveAccountOperation` and discard stale completions after account switches.
- Use a request generation for refresh so a pending continuation cannot overwrite refreshed state.
- Let `post_not_found` remain in `AsyncError` for the page to classify as permanent and non-retryable.

Dependency injection remains Riverpod overrides of `postRepositoryProvider`; `FakePostRepository` gains only the three corresponding callbacks. No new package or service locator is introduced.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Routes

Add sibling authenticated detail routes under the existing authenticated shell:

```text
/posts/:did/:rkey/likes   -> PostInteractionAccountsPage(kind: likes)
/posts/:did/:rkey/reposts -> PostInteractionAccountsPage(kind: reposts)
/posts/:did/:rkey/quotes  -> PostQuotesPage
```

Each typed route constructor carries only validated DID/rkey path values. No `$extra` is required, so direct links survive process start. `push` from the detail summary or response menu preserves the originating navigator history; signed-out redirects continue through the existing top-level router guard.

### Root detail summary

`PostInteractionSummary` is placed in the same root sliver immediately after the top-level `PostCard` and before sort/reply content. It receives the current rendered `Post`, derives entries on every build, and owns no count state.

```text
Wrap(alignment: start)
  if likeCount > 0: localized Likes(count) link -> Likes route
  if repostCount > 0: localized Reposts(count) link -> Reposts route
  if quoteCount > 0: localized Quotes(count) link -> Quotes route
```

Use `Wrap` with theme spacing, no punctuation separators, fixed order, independent focus/tap targets, minimum touch target, and explicit semantic label/hint containing exact value, type, and destination. Return `SizedBox.shrink` when all counts are zero. Existing `PostCard` like and combined share controls remain untouched.

### Response more-actions menu

Add `VoidCallback? onViewLikes` to `PostCard`. `_PostCardMenu` prepends a separate `CraftskyContextMenuGroup` only when `post.reply != null`, `post.likeCount > 0`, and the callback is non-null. Its first and only item is localized `View likes`, normal style, and a likes/people icon. Existing ownership, relationship, report, pin, and delete items remain in their following group.

`PostThreadPage` passes the callback for each visible comment and nested reply using that response's `post.author.did` and `post.rkey`; it does not pass it to the root. Zero-count responses receive no callback. This keeps the route identity local to the rendered response and prevents root identity capture.

### Likes and Reposts page

`PostInteractionAccountsPage` watches the account provider and renders:

- App bar title `Likes`/`Reposts` while initial loading or initial error.
- App bar title with authoritative total after any successful first page, including zero.
- Full-page `StitchProgressIndicator` for initial pending.
- Type-specific `CraftskyEmptyState` after successful empty data.
- Shared post-unavailable state with Back and no Retry for `404 post_not_found`.
- Localized retry state for transient initial failures.
- `AutoPaginatedListView` for content, automatic continuation, retained rows, bottom progress, and retry.
- Extracted `ProfileAccountListTile` using display name, handle, avatar/customisation, relationship state as already supported, and `showUserProfileCard` on tap.

### Quotes page

`PostQuotesPage` keeps a plain localized `Quotes` title in every state. It uses the same loading/error/empty/pagination shell and `AutoPaginatedListView`, with `PostCard` rows in repository order. It wires the existing post navigation, author/quote-preview defaults, reply composer, like/repost toggles, quote composer, report, and owner delete callbacks. Mutation completions replace the matching quote-provider item from the returned post model; successful deletion removes it. No interaction summary is rendered inside quote rows.

### Responsive shell

The routes remain siblings of `PostThreadRoute` under `AuthenticatedShellRoute`, so compact full-screen and large rail/detail behavior comes from the existing shell. Lists use always-scrollable, safe-area-aware vertical layouts. The summary uses wrapping rather than horizontal scrolling. Tests exercise 390x844 and 1200x800 at text scales 1.0 and 2.0.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| All root counts zero | Omit the whole summary | FR-001, RULE-004 | UT-001, AT-001 |
| Mixed root counts | Omit zero entries independently; preserve Likes/Reposts/Quotes order | FR-001, RULE-004 | UT-001, AT-001 |
| Response likes zero | Do not create the menu group or callback | FR-013, RULE-004, RULE-005 | UT-006, AT-005 |
| Initial request pending | Full-page progress; Likes/Reposts title has no pending `(0)` | FR-005, FR-007 | UT-004, AT-006 |
| Successful empty page | Type-specific scroll-safe empty state; account title shows authoritative zero | FR-005, FR-007 | AT-006 |
| Target missing, hidden, blocked, terminal, or unsupported response share target | `404 post_not_found`; localized Post unavailable with Back only and no retained target content | FR-007, FR-009, FR-010 | IT-006, IT-011, AT-006 |
| Initial transient/API/network failure | Localized retry; invalidate/refetch first page | FR-007 | UT-004, AT-006 |
| Load-more pending | Keep rows and show bottom progress | FR-007 | UT-004, AT-006 |
| Load-more failure | Keep rows, retain exact cursor, show bottom Retry, append once after success | FR-007, NFR-002 | UT-004, AT-006 |
| Refresh races load-more | Generation guard discards stale continuation result | FR-007, NFR-004 | UT-004 |
| Active account changes in flight | Account operation guard discards old completion; provider rebuilds in new scope | NFR-004 | UT-004, IT-009 |
| Invalid/mismatched cursor | AppView `400 invalid_cursor`; Flutter restarts traversal from first page | FR-009, NFR-002 | UT-002, UT-005, IT-006, IT-010 |
| Exact final page size | `limit + 1` sentinel determines exhaustion; no cursor unless a further eligible row exists | FR-008, NFR-002 | IT-005, IT-010 |
| Tied timestamps | URI descending tie-break and tuple seek preserve fixed-data traversal | FR-003, FR-004, FR-006, NFR-002 | UT-008, IT-010 |
| Deleted interaction between pages | Deleted row is omitted; current traversal remains stable and refresh restarts | RULE-001, NFR-002 | IT-001, IT-002, IT-010 |
| Same author quotes repeatedly | Preserve every eligible quote post by URI | RULE-003 | UT-003, IT-003, AT-004 |
| Actor reposts and quotes | Treat records independently; no inferred straight repost | RULE-002 | UT-003, IT-002, IT-003, REG-003 |
| Muted actor/account | Keep account row and relationship flag; keep quote as revealable muted placeholder | FR-010 | IT-004, IT-012 |
| Hidden/blocked/former/terminal actor | Exclude item and corresponding total/count without leaking identity | FR-010 | IT-004, IT-011, IT-012 |
| Language-ineligible quote | Exclude quote row and viewer-specific `quoteCount` through the shared language predicate | FR-010 | IT-004, IT-012 |
| Handle unavailable | Fail account request as retryable `502 identity_unavailable`; do not omit row or alter total | FR-005, FR-007, FR-010 | IT-005, IT-013, AT-006 |
| Nested/unavailable quote preview | Reuse one-level `attachQuoteViews`; no recursive list hydration | FR-006, FR-010 | IT-003, REG-005 |
| Like/repost mutation completes | Root summary rebuilds from replaced post model; quote-list item replacement uses provider state | FR-011, FR-012 | AT-008, REG-001 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `appview/internal/api/post_interaction_lists_store_test.go` | TD-002/TD-003/TD-004; roots, comments, replies, active/deleted likes, tied times | `PostStore` has no liker-list method or eligible total |
| 2 | IT-002, UT-003 | Same store suite | Repost-only, quote-only, dual actor, deleted repost | Straight-repost membership/kind classifier does not exist |
| 3 | IT-003 | Same store suite | Repeated quote author, hidden/deleted quotes, quote-of-quote | Quote list query/hydration does not exist |
| 4 | UT-002, UT-008, IT-010 | Same store suite, cursor-focused test names | TD-006/TD-007 with ties, cross-kind/target cursors, exact-size final pages | Generic cursor accepts unbound payload or traversal lacks URI tie-break/sentinel |
| 5 | IT-004, IT-011, IT-012 | Same store suite and existing `post_store_test.go` engagement cases | Full viewer/actor/target moderation, relationship, membership, mute, warning, language matrix | Existing counts include actors/posts the lists exclude |
| 6 | IT-013 | Same store suite with counting resolver/query observer | Full account and quote pages | Hydration performs per-item identity/query work |
| 7 | IT-008 | Same store suite using existing EXPLAIN conventions | Representative rows with sequential scans disabled where appropriate | New query shape is absent or fails to use subject/quote indexes |
| 8 | IT-005, IT-006 | `appview/internal/api/post_interaction_lists_handler_test.go` | Fake list reader; valid/invalid identifiers, limits, cursors, targets, response targets, missing handles | GET handlers/contracts do not exist |
| 9 | IT-014 | Same handler suite plus existing observer recorders | Success, validation, unavailable, and internal failure | New operations are unobserved or expose high-cardinality fields |
| 10 | IT-007, REG-004 | `appview/internal/routes/routes_test.go`, `inventory_test.go`, `architecture_test.go` | Production mux, missing auth/device/body, PDS panic recorder | GET routes/policies are absent |
| 11 | UT-007 | Existing `app/test/profile/models/profile_test.dart` and `feed/models/post_page_test.dart` | Account page total/cursor and post page without pin metadata | Reused shape expectations are not pinned for these endpoints |
| 12 | UT-005, IT-009, REG-004 | Existing `post_api_client_test.dart`, `post_repository_test.dart` | Dio adapter, all paths, optional params, 400/404, response-like identity | Client/repository methods do not exist |
| 13 | UT-004 | `app/test/feed/providers/post_interaction_lists_provider_test.dart` | TD-006/TD-010, completers, errors, refresh and account-switch races | Providers/state do not exist |
| 14 | AT-003, AT-006 | `app/test/feed/pages/post_interaction_accounts_page_test.dart` | Hydrated accounts, totals, empty/loading/permanent/transient/incremental states | Account destination does not exist |
| 15 | AT-004 | `app/test/feed/pages/post_quotes_page_test.dart` | Repeated author and normal post affordances | Quotes destination does not exist |
| 16 | UT-001 | `app/test/feed/widgets/post_interaction_summary_test.dart` | TD-001 count combinations and semantics | Summary widget does not exist |
| 17 | AT-001, AT-002, AT-008 | Extend `post_thread_page_test.dart` and `post_card_test.dart` | Mixed counts and completed like/repost model replacement | Detail has no links; mutation separation is unproven |
| 18 | UT-006, AT-005 | Extend `post_card_test.dart` and `post_thread_page_test.dart` | Zero/positive comment/reply likes with distinct identities | Response menu has no separate liker group |
| 19 | AT-009 | `app/test/router/post_interaction_routes_test.dart`, existing `router_redirect_test.dart` | Production router, direct locations, push/back, signed-out cases | Typed routes are absent |
| 20 | AT-007 | `app/test/feed/pages/post_interaction_accessibility_test.dart` | TD-009 semantics, long text, four size/scale combinations | New controls/pages lack verified semantics/layout |
| 21 | REG-001, REG-002 | Existing `post_card_test.dart`, `toggle_post_interactions_provider_test.dart`, thread tests | All card surfaces and mutation callbacks | Shared-card changes may leak or replace controls |
| 22 | REG-003, REG-005 | Existing repost/quote and post response tests plus Quotes page test | Dual interaction actor, repeated/nested quotes, moderation placeholders | New list hydration may conflate or recurse |
| 23 | REG-006 | Existing thread/comment/router suites | Sorting, replies, pin/report/delete, compact/large shell | New root/menu composition may regress thread behavior |
| 24 | MAN-001 | Manual responsive smoke | Seeded zero/mixed/positive data on narrow and large layouts | Visual quality not established by automated tests alone |
| 25 | MAN-002 | Manual assistive-technology smoke | Record mobile screen reader and desktop keyboard/platform used | Real spoken phrasing/focus restoration remains unverified |

Test files are intentionally consolidated compared with `02-acceptance-tests.md`: one cohesive Go store suite, one Go handler suite, one Flutter provider suite, and behavior-specific widget/page suites. Test names retain the IDs above for traceability.

## 10. Sequencing And Guardrails

- First TDD step: Add failing `IT-001` cases for active liker accounts on a top-level post, comment, and nested reply, including newest-first tied ordering, exact eligible `totalCount`, and final cursor omission. Implement only enough target/account store contract to make those cases green.
- Dependencies between work items: account eligibility and cursor primitives precede Reposts; quote eligibility then reuses cursor binding and post hydration; viewer-aware engagement counts follow the list predicates; handlers/routes follow stores; Flutter data follows wire contracts; providers follow data; pages/routes/widgets follow providers.
- Generated artifacts: after annotated providers/routes and ARB changes, run `flutter gen-l10n` and `dart run build_runner build --delete-conflicting-outputs` from `app/`; include generated localisation, provider, and router files in implementation changes.
- AppView focused verification: from `appview/`, run `go test ./internal/api -run 'TestPostStore_ListPostInteractionAccounts_Likes'`, then `go test ./internal/api ./internal/routes`.
- Flutter focused verification: from `app/`, run `flutter test test/feed/providers/post_interaction_lists_provider_test.dart test/feed/pages test/feed/widgets/post_interaction_summary_test.dart test/router/post_interaction_routes_test.dart`.
- Final automated gates: `just fmt`, `just appview-check`, `just app-analyze`, and `just app-test`.
- Final manual gates: complete MAN-001 and MAN-002 and record viewport/text scale plus the screen-reader/keyboard platform used.
- Guardrails: parse DID/rkey only at HTTP/router boundaries; never interpolate user input as SQL identifiers; bind cursor kind and exact target; use `limit + 1`; keep account and quote paths non-polymorphic publicly; share eligibility fragments with counts; use bounded batch hydration; fail the complete request rather than silently omit an unavailable identity and leave the total inconsistent; do not log identifiers/cursors/items; preserve muted reveal behavior.
- Guardrails: do not change POST/DELETE like/repost semantics, combined share count/action behavior, quote creation, response interaction support, shell redirect policy, existing post response wire fields, or profile-row actions.
- Out of scope: tabs, filters/search/sort controls, follow/moderation actions on rows, interaction record metadata, notifications/analytics, frozen pagination snapshots, direct PDS access, OAuth changes, lexicons, migrations without query-plan evidence, dependencies, CLI work, and background jobs.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Existing engagement count helpers are less restrictive than the approved list policy. | Hidden/former/blocked activity could leak through root summary counts. | Replace them with shared viewer-aware account/quote eligibility fragments before exposing UI; prove unchanged-data equality in IT-004/IT-012. |
| CPQ-002 | Non-blocking | Existing profile graph handlers resolve handles in a loop; this feature forbids normal per-item identity work. | Interaction pages could become latency/N+1 paths. | Join the bounded local identity cache in the account list query and fail the whole request with `identity_unavailable` if a required handle is absent; cover with IT-013. Do not broaden this task into refactoring existing graph lists. |
| CPQ-003 | Non-blocking | Exact-page exhaustion cannot be inferred from `len(rows) == limit`. | A cursor could be emitted that leads only to an empty page. | Fetch `limit + 1`, trim the sentinel, and encode from the final returned row only when the sentinel exists. |
| CPQ-004 | Non-blocking | Quote hydration can accidentally omit muted posts or recursively hydrate nested quotes. | Policy or response-size regression. | Keep muted rows in SQL, apply `SurfaceQuote` relationship shaping, and call existing one-level `attachQuoteViews` once for the page. |
| CPQ-005 | Non-blocking | The current active-subject/quote indexes may not satisfy the exact policy/order plan selected by PostgreSQL. | Slow scans at larger data volumes. | Run IT-008 after query shape stabilizes. If inadequate, stop and request review of a reversible migration; no migration is part of this plan. |
| CPQ-006 | Non-blocking | Independent requests cannot hold detail counts and later pages in one snapshot. | Counts may naturally differ after concurrent activity. | Keep deterministic fixed-data traversal, refresh from page one, and do not add snapshot tokens, per approved GAP-002/GAP-004. |
| CPQ-007 | Non-blocking | Widget semantics cannot prove real VoiceOver/TalkBack phrasing or platform menu focus. | Accessibility could appear green in automation but fail on-device. | MAN-002 is a required handoff check; record platform/device and keyboard path before merge. |
| CPQ-008 | Non-blocking | `PostCard` is shared across many surfaces. | Liker navigation or summary could leak outside approved thread responses. | Keep summary outside `PostCard`; require reply + positive count + callback for the menu group; run REG-001/REG-002/REG-006. |

Blocking questions: None.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001`, named `TestPostStore_ListPostInteractionAccounts_Likes`, covering root/comment/reply targets, active/deleted likes, tied `(created_at DESC, uri DESC)` order, exact eligible total, and omitted final cursor.
- Focused command: from `appview/`, `go test ./internal/api -run 'TestPostStore_ListPostInteractionAccounts_Likes'`
- Notes: Preserve all requirement, acceptance-criterion, and test IDs in `05-implementation-plan.md`. Keep red-green-refactor evidence per test group. Do not create a migration unless IT-008 provides evidence and the migration is separately reviewed. Complete and record MAN-001/MAN-002 after automated gates.
