# Requirements: Profile Social Summary

## 1. Initial Request
Improve social interactions in the app by removing follower and following counts from profile pages and replacing them with:

- When viewing someone else's profile, show clickable text such as `12 mutual followers`; tapping it opens a 90%-height bottom sheet with the paginated list of mutuals.
- A stat that shows the account's age.
- Recent activity shown as `X posts in the last 7 days`, counting top-level authored posts only and excluding replies/comments, repost interactions, likes, and follows.
- Total number of posts alongside the existing total number of projects.
- Users can still view all their followers and followings from the settings page; tapping those settings entries shows lists ordered by recency.

## 2. Current Codebase Findings
- Relevant files:
  - Flutter profile model/client/UI: `app/lib/profile/models/profile.dart`, `app/lib/profile/data/profile_api_client.dart`, `app/lib/profile/widgets/profile_meta_section.dart`, `app/lib/profile/widgets/profile_stats.dart`, `app/lib/profile/pages/profile_page.dart`.
  - Flutter settings UI: `app/lib/settings/pages/settings_page.dart`.
  - AppView profile handlers/store: `appview/internal/api/profile.go`, `appview/internal/api/profile_response.go`, `appview/internal/api/profile_store.go`.
  - AppView routes: `appview/internal/routes/routes.go`.
  - Social graph persistence: `appview/migrations/000012_atproto_follows.up.sql`.
  - Post persistence: `appview/migrations/000010_craftsky_posts.up.sql`, `appview/internal/api/post_store.go`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Existing patterns:
  - Flutter app reads profile data from AppView JSON/HTTP endpoints, not from PDS directly.
  - `/v1/profiles/@{handleOrDid}` and `/v1/profiles/me` return profile summary data via one `ProfileResponse` shape.
  - List endpoints use `/v1/`, auth headers, JSON bodies, camelCase keys, and opaque cursor pagination.
  - Follow/unfollow writes already use `/v1/profiles/@{handleOrDid}/follows`.
- Current behavior:
  - Profile pages show following count, follower count, and a hardcoded project count in `ProfileStats`.
  - `ProfileResponse` exposes `followingCount`, `followerCount`, `createdAt`, and `viewerIsFollowing`.
  - AppView currently calculates follower/following counts from `atproto_follows`, restricted to Craftsky profiles for count totals.
  - Settings page only renders clear-image-cache and sign-out tiles.
  - Posts and comments are both indexed in `craftsky_posts`; root profile posts and authored comments are currently listed through separate endpoints.
- Constraints discovered:
  - The Flutter app must continue to read from AppView; it must not query PDS directly for graph or post summaries.
  - The Flutter app must never hold PDS tokens.
  - New API routes must follow the existing `/v1/` REST conventions, authenticated-device requirement, camelCase JSON, and opaque-cursor pagination.
  - No lexicon change is required for this change unless future implementation changes record schemas, which is out of scope here.
  - Project-count storage is not yet visible in the inspected code; the existing project stat is currently UI-only/hardcoded and should be replaced by a real response field/value, even if that value is `0` until project persistence exists.
- Test/build commands discovered:
  - AppView Go tests: `just test` after `just dev-d` is running.
  - Flutter tests are present under `app/test/**`; likely command is `flutter test` from `app/`.

## 3. Clarifying Questions And Decisions
### Q1: Which direction should the requirements document use for this change?
Answer: Profile summary API (Recommended).

Decision / implication: Add profile summary fields to existing profile responses, add list endpoints for mutuals/followers/following, and update Flutter profile/settings UI. Recent activity and total posts count top-level authored Craftsky posts only; account age is computed in Flutter from the existing Craftsky profile `createdAt` value.

### Q2: Follow-up requirement decisions from requirements review
Answer: Mutuals should render as clickable uncapped text such as `12 mutual followers`; zero mutuals render nothing. The mutuals list opens in a 90%-height bottom sheet and is loaded from a separate paginated endpoint. Profile responses keep `followerCount` and `followingCount`, and add only `mutualFollowerCount` for mutuals, not a preview array. Follower/following list page counts appear in the app bar title, not in settings entries. Account age uses Craftsky `createdAt` and is hidden for non-Craftsky profiles. Recent activity and total posts count top-level posts only. Follow graph staleness from indexing lag is acceptable. Blocks/mutes are out of scope. Empty following copy can be `You are not following anyone`; empty followers copy can be `No one follows you yet`; no discovery CTA is included in this slice. Project count should be implemented as a real value now, even if it is always `0` until project persistence exists.

Decision / implication: Requirements should specify mutual count as an uncapped clickable count, not a capped preview; profile responses should contain counts but list contents should come from paginated endpoints; profile-page account age and post activity stats apply to Craftsky profiles; project count must no longer be hardcoded in Flutter.

## 4. Candidate Approaches
### Option A: Profile summary API (Recommended)
Summary: Extend existing profile responses with profile-summary fields and add paginated list endpoints for mutual followers, followers, and following. Update Flutter profile and settings UI to consume those AppView contracts, including clickable mutual-follower count text that opens a 90%-height bottom sheet for the paginated mutuals list.

Pros:
- Fits existing AppView-read / Flutter-render architecture.
- Minimizes profile-page request complexity for scalar stats.
- Keeps list data paginated where it can grow large.
- Preserves API convention consistency.

Cons:
- Requires coordinated AppView and Flutter changes.
- Existing `Profile` model and profile response tests must change.
- Requires careful compatibility around removing old count presentation while old fields may still exist temporarily.

Risks:
- Mutual-follower queries can become expensive without appropriate indexes or limits.

### Option B: Separate social endpoints only
Summary: Keep profile responses lean and fetch mutuals, age, recent activity, and counts from separate dedicated endpoints.

Pros:
- Smaller profile response contract.
- Each social summary can load/fail independently.
- Easier to cache high-cost list/stat queries separately.

Cons:
- More client requests and loading states on profile pages.
- More route surface than necessary for scalar summary data.
- Less aligned with current profile-summary pattern.

Risks:
- Profile page could feel slower or visually fragmented unless the client adds skeleton/loading states.

### Option C: UI-only first slice
Summary: Remove follower/following counts and rearrange only currently available fields, deferring mutuals, list pages, and new stats.

Pros:
- Smallest immediate change.
- Lower implementation risk.

Cons:
- Does not satisfy the requested mutuals, account age, recent activity, total posts, or settings list behavior.
- Leaves social-interaction improvements incomplete.

Risks:
- Users lose follower/following counts without receiving the requested replacement context.

## 5. Recommended Direction
Recommended approach: Option A — Profile summary API.

Why: The request is a coherent profile/social-summary change that spans AppView summary data, social graph lists, and Flutter UI. Extending the existing profile summary response for scalar values while adding paginated list endpoints for mutuals/followers/following best matches existing code patterns and keeps large data off the main profile response.

## 6. Problem / Opportunity
Follower/following counts emphasize popularity metrics on profile pages. Craftsky should instead surface social context and crafting activity: who the viewer knows in common with the profile, how established the account is, and how active the account has been recently.

## 7. Goals
- G-001: Reduce popularity-count prominence on profile pages.
- G-002: Provide viewer-specific mutual-follower context on visitor profiles.
- G-003: Show account age, recent activity, total posts, and projects in a concise profile summary.
- G-004: Preserve access to full follower/following lists through settings, ordered by recency.
- G-005: Keep all profile/social reads routed through the AppView.

## 8. Non-Goals
- NG-001: Do not change atproto lexicon record schemas.
- NG-002: Do not add algorithmic ranking or recommendation logic.
- NG-003: Do not expose follower/following counts on profile pages under a different label.
- NG-004: Do not build project persistence or redesign project tabs as part of this change; project count should be a real API/model field and may be `0` until project persistence exists.
- NG-005: Do not change follow/unfollow write semantics.
- NG-006: Do not add public unauthenticated access to social graph lists.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Viewer | Signed-in user viewing another account's profile. | See meaningful social context without popularity counts. |
| Profile owner | Signed-in user viewing their own profile and settings. | See their own profile stats and access follower/following lists from settings. |
| AppView | Backend read model and API layer. | Serve profile summary and graph lists from indexed public data. |
| Flutter client | Mobile app UI. | Render new profile and settings experiences from AppView responses. |

## 10. Current Behavior
Profile pages display follower and following counts directly. The AppView profile response includes `followerCount` and `followingCount`. The profile page also shows a projects stat, but the inspected Flutter UI currently hardcodes the project count. Full follower/following list views are not present in settings.

## 11. Desired Behavior
Profile pages no longer display follower or following counts. Visitor profiles display mutual followers as uncapped clickable text, such as `12 mutual followers`, that opens a 90%-height bottom sheet with the paginated mutuals list. Profile stats show Craftsky account age using `Joined <age> ago` copy, recent activity as top-level posts in the last 7 days, total top-level posts, and total projects from response data. Non-Craftsky profiles do not show account age. Settings gives the signed-in user entry points to follower and following list pages without showing the counts on the settings page itself; each list page is ordered by recency and shows its count in the app bar title.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Profile pages must de-emphasize popularity by not showing follower or following counts. | Aligns social experience with meaningful context over popularity metrics. | Prompt | AC-001 |
| BR-002 | Business | Must | Visitor profiles must show viewer-specific mutual-follower context through uncapped clickable mutual-count text that opens a mutuals list bottom sheet. | Helps viewers understand shared community connections without restoring general follower/following counts. | Prompt, Q1, Q2, review feedback | AC-002, AC-003, AC-004, AC-015, AC-019 |
| BR-003 | Business | Must | Users must retain access to their complete follower and following lists from settings. | Removing profile counts must not remove graph visibility for the account owner. | Prompt | AC-008, AC-009, AC-010 |
| FR-001 | Functional | Must | The profile page shall remove follower-count and following-count cells from the visible profile stats area. | Directly satisfies the requested profile-page change. | Prompt, codebase | AC-001 |
| FR-002 | Functional | Must | When the signed-in viewer opens someone else's profile with one or more mutual followers, the profile page shall show uncapped clickable text for accounts the viewer follows who also follow the current profile, such as `12 mutual followers`. | Replaces public follower counts with viewer-specific relationship context. | Prompt, Q1, Q2, review feedback | AC-002, AC-003, AC-004, AC-015 |
| FR-003 | Functional | Must | Mutual followers shall be computed by AppView from the indexed follow graph, not by the Flutter client querying PDS. | Preserves architecture rules and centralizes graph logic. | AGENTS.md, discovery | AC-003 |
| FR-004 | Functional | Must | For Craftsky profiles, the profile summary shall provide the account creation timestamp, and Flutter shall compute and render account age from that timestamp using `Joined <age> ago` copy. | Provides requested account-age stat while keeping age formatting a frontend concern. | Prompt, Q1, Q2, codebase, review feedback | AC-005, AC-018 |
| FR-005 | Functional | Must | The profile summary shall expose and render recent activity as the number of top-level Craftsky posts authored by the account in the last 7 days. | Provides requested recent-activity stat while matching the product meaning of posts. | Prompt, Q1, Q2, codebase, review feedback | AC-006 |
| FR-006 | Functional | Must | The profile summary shall expose and render total top-level posts alongside total projects; the project count shall come from response/model data rather than a Flutter hardcoded value. | Adds total post visibility and makes the existing projects stat data-driven. | Prompt, Q2 | AC-007, AC-020 |
| FR-007 | Functional | Must | Settings shall include tappable follower and following entries for the signed-in user without showing follower or following counts on the settings page. | Provides the requested access point after counts are removed from profiles while keeping counts one tap deeper. | Prompt, codebase, review feedback | AC-008 |
| FR-008 | Functional | Must | Tapping followers in settings shall show the signed-in user's followers ordered by most recent follow first, with the follower count shown in the list page app bar title. | Satisfies list order requirement and keeps counts one tap deeper than settings. | Prompt, Q2, discovery | AC-009 |
| FR-009 | Functional | Must | Tapping following in settings shall show accounts the signed-in user follows ordered by most recent follow first, with the following count shown in the list page app bar title. | Satisfies list order requirement and keeps counts one tap deeper than settings. | Prompt, Q2, discovery | AC-010 |
| FR-010 | Functional | Must | Follower, following, and mutual-follower list responses shall return display-ready account summaries including stable DID, current handle, and available profile display fields. | Allows Flutter to render account lists without PDS reads. | Discovery | AC-011 |
| FR-011 | Functional | Should | Large graph lists should paginate using existing opaque cursor conventions. | Prevents unbounded responses and matches API design. | API architecture spec | AC-012 |
| FR-012 | Functional | Must | The profile response shall continue making `followerCount` and `followingCount` available, but Flutter shall not display those counts on profile pages or on the settings entry page. | Keeps graph totals available to clients while satisfying the UX change. | Q2, discovery, review feedback | AC-001, AC-008, AC-016 |
| FR-013 | Functional | Must | Tapping the mutual-follower count text on a visitor profile shall open a 90%-height bottom sheet containing the mutual followers list. | Makes mutual context explorable without crowding the profile header. | Q2, review feedback | AC-015 |
| FR-014 | Functional | Must | Empty follower and following list pages shall show a plain empty-state message; zero mutual followers shall not render a mutual-follower section. | Handles empty graph states without adding premature discovery CTAs. | Review feedback | AC-017 |
| FR-015 | Functional | Must | The profile response shall include `mutualFollowerCount` for visitor-profile mutuals and shall not embed a mutual-follower preview array. | Keeps scalar summary data on the profile response and list data behind paginated endpoints. | Q2 | AC-019 |
| FR-016 | Functional | Must | The full mutual followers list shall be fetched from a separate authenticated paginated endpoint. | Keeps profile response small and supports large mutual lists. | Q2 | AC-012, AC-015, AC-019 |
| FR-017 | Functional | Must | Non-Craftsky profiles shall not show account age. | Avoids implying a Craftsky account age for external profiles. | Q2 | AC-018 |
| NFR-001 | Non-functional | Must | New or changed AppView API fields and endpoints must use `/v1/`, authenticated requests, camelCase JSON, error envelopes, and opaque cursor pagination for lists. | Maintains API consistency. | AGENTS.md, API spec | AC-012, AC-013 |
| NFR-002 | Non-functional | Should | Social graph and activity summary queries should use bounded result sizes and indexed order/filter columns where applicable. | Reduces performance risk for profiles with larger graphs. | Discovery risk | AC-014 |
| RULE-001 | Business rule | Must | Mutual followers are accounts where `viewer -> mutual` and `mutual -> profile` follow records both exist in AppView's active indexed follow graph. | Defines mutuals precisely. | Prompt, Q1 | AC-003 |
| RULE-002 | Business rule | Must | The account age stat is based on the profile's Craftsky account/profile creation timestamp known to AppView and formatted by Flutter. | Uses existing available timestamp and avoids external account-age dependencies. | Q1, codebase, review feedback | AC-005 |
| RULE-003 | Business rule | Must | Recent activity counts top-level authored rows in `craftsky_posts` within the trailing 7-day window; replies/comments, repost interactions, likes, and follows do not count. | Matches confirmed direction, review feedback, and current data model. | Prompt, Q1, Q2, codebase, review feedback | AC-006 |
| RULE-004 | Business rule | Must | Followers list recency is based on the follow record's `created_at` where another account followed the signed-in user; following list recency is based on the signed-in user's follow record `created_at`. | Defines deterministic list ordering. | Prompt, discovery | AC-009, AC-010 |
| RULE-005 | Business rule | Must | Total posts count top-level authored `craftsky_posts` rows only. | Aligns total post count with product meaning of posts. | Q2 | AC-007 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-012 | Given any profile page, when it is rendered, then follower and following counts are not visible in the profile stats area. |
| AC-002 | BR-002, FR-002 | Given a signed-in viewer opens another Craftsky profile with one or more mutual followers, when the profile loads, then the mutual-follower section displays uncapped clickable text such as `12 mutual followers`. |
| AC-003 | BR-002, FR-002, FR-003, RULE-001 | Given the indexed follow graph contains `viewer -> mutual` and `mutual -> profile`, when AppView builds the mutual followers list, then that mutual account is included; accounts missing either relationship are excluded. |
| AC-004 | FR-002 | Given a signed-in viewer opens their own profile, when the profile loads, then the mutual-follower section is not shown. |
| AC-005 | FR-004, RULE-002 | Given a Craftsky profile response includes a creation timestamp, when the profile renders, then Flutter computes the age and displays it using `Joined <age> ago` copy. |
| AC-006 | FR-005, RULE-003 | Given an account authored top-level posts, replies/comments, and repost interactions inside and outside the trailing 7-day window, when the profile summary is fetched, then `posts in the last 7 days` includes only top-level authored `craftsky_posts` rows within the trailing 7-day window. |
| AC-007 | FR-006, RULE-005 | Given a profile summary has total post data, when the profile renders, then total top-level posts are shown alongside the projects stat. |
| AC-008 | BR-003, FR-007, FR-012 | Given the signed-in user opens settings, when the settings page renders, then followers and following entries are visible and tappable without showing follower or following counts on the settings page. |
| AC-009 | BR-003, FR-008, RULE-004 | Given the signed-in user has followers with different follow `created_at` values, when they tap the followers settings entry, then the follower list is ordered newest follow first and the follower count appears in the app bar title. |
| AC-010 | BR-003, FR-009, RULE-004 | Given the signed-in user follows accounts at different times, when they tap the following settings entry, then the following list is ordered newest follow first and the following count appears in the app bar title. |
| AC-011 | FR-010 | Given a follower, following, or mutual-follower list response contains accounts, when Flutter renders the list, then each row has enough AppView-provided account data to display identity without a PDS read. |
| AC-012 | FR-011, NFR-001 | Given a graph list contains more than one page of accounts, when the client requests pages with `limit` and returned cursor values, then pagination follows the existing opaque-cursor response convention. |
| AC-013 | NFR-001 | Given an unauthenticated or missing-device request reaches a new `/v1/` social graph endpoint, when AppView handles it, then it returns the existing authenticated-device error behavior. |
| AC-014 | NFR-002 | Given a profile has many followers/following records or posts, when AppView serves the new summary/list data, then responses are bounded and ordered/filterable without unbounded client-side filtering. |
| AC-015 | BR-002, FR-002, FR-013, FR-016 | Given a visitor profile shows mutual-follower count text, when the viewer taps it, then a 90%-height bottom sheet opens and loads the mutual followers list from the separate paginated endpoint. |
| AC-016 | FR-012 | Given a client requests the profile response, when AppView returns the response, then follower and following counts remain available in the API contract even though the Flutter profile/settings entry UI does not display them. |
| AC-017 | FR-014 | Given a user opens an empty following list page, when the page renders, then it shows a simple empty-state message such as `You are not following anyone`; given a user opens an empty followers list page, then it shows `No one follows you yet`; given a visitor profile has zero mutual followers, then no mutual-follower section is shown. |
| AC-018 | FR-004, FR-017 | Given a non-Craftsky profile is rendered, when profile metadata is displayed, then account age is not shown. |
| AC-019 | BR-002, FR-015, FR-016 | Given AppView returns a visitor profile response, then the response includes `mutualFollowerCount` and does not include embedded mutual account preview items; given the client needs mutual account rows, it fetches them from the separate paginated mutual followers endpoint. |
| AC-020 | FR-006 | Given the Flutter profile page renders the projects stat, then it uses profile response/model data for project count rather than a hardcoded Flutter value, even when the value is `0`. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | No mutual followers | Visitor profile renders no mutual-follower section (`SizedBox.shrink()` equivalent) and does not show follower/following counts. | FR-001, FR-002, FR-014 |
| EC-002 | Mutual account has no cached Bluesky profile row | List still includes stable identity if available, with missing optional display fields handled gracefully. | FR-010 |
| EC-003 | Non-Craftsky profile is viewed | Account age is not shown; Craftsky-only stats should not be invented if required indexed data is unavailable. | FR-004, FR-005, FR-006, FR-017 |
| EC-004 | Account has zero posts in last 7 days | Recent activity displays zero activity in the chosen copy format. | FR-005 |
| EC-005 | Account has posts exactly on the 7-day boundary | Boundary handling is deterministic and covered by tests against AppView's chosen time comparison. | RULE-003 |
| EC-006 | Follower/following list is empty | Settings list page shows the appropriate simple empty-state message, not an error or discovery CTA. | FR-008, FR-009, FR-014 |
| EC-007 | Handle resolution fails for a profile/list account | AppView follows existing identity error behavior for target resolution; list rows with unavailable optional handles are handled according to the final API design. | NFR-001, FR-010 |

## 15. Data / Persistence Impact
- New fields:
  - Expected additive profile summary fields include total top-level post count, posts-in-last-7-days top-level count, project count, and mutual-follower count. Exact JSON names should use camelCase.
  - Profile responses keep `followerCount` and `followingCount` and add `mutualFollowerCount`; profile responses do not include a mutual-follower preview array.
  - Account age is derived client-side from existing `createdAt`; AppView must provide the account creation timestamp but does not need to compute a formatted age.
  - Graph list account summaries should include DID, handle, and optional display fields.
- Changed fields:
  - Flutter no longer displays `followerCount` or `followingCount` on profile pages.
  - Flutter does not display follower/following counts on the settings entry page.
  - Existing API count fields remain available as a formal API requirement.
- Migration required:
  - No new durable table is expected from requirements discovery.
  - Implementation may add indexes if performance review shows existing indexes are insufficient for mutuals or recency ordering.
- Backwards compatibility:
  - Additive API response fields and routes are preferred under `/v1/`.
  - Removing old response fields is not required for this change and would be a breaking API decision.

## 16. UI / API / CLI Impact
- UI:
  - Profile stats area changes from follower/following/projects to account age, recent activity, total posts, and projects.
  - Visitor profiles add clickable mutual-follower count text that opens a 90%-height bottom sheet.
  - Settings gains followers/following entries and list pages, but the settings entry page does not show follower/following counts.
- API:
  - Extend existing profile summary responses with scalar social/activity stats, `mutualFollowerCount`, and data-driven `projectCount`; keep `followerCount` and `followingCount` on the profile response.
  - Add authenticated paginated list endpoints for mutual followers, followers, and following, following `/v1/` conventions.
- CLI:
  - None expected.
- Background jobs:
  - None expected beyond existing firehose indexing.

## 17. Security / Privacy / Permissions
- Authentication:
  - All new `/v1/` endpoints require the existing authenticated session and device ID behavior.
- Authorization:
  - Mutual followers are viewer-specific and require the viewer DID from auth context.
  - Settings follower/following lists are scoped to the signed-in user.
- Sensitive data:
  - Follow graph, profile data, and posts are public atproto/AppView-indexed data, but viewer-specific mutuals should still only be computed for authenticated viewers.
- Abuse cases:
  - List endpoints should remain paginated/bounded to reduce scraping and load risks.

## 18. Observability
- Events:
  - None required by requirements.
- Logs:
  - New handlers should follow existing AppView request/run ID logging conventions for failures.
- Metrics:
  - Could add latency/error metrics for new graph list endpoints if metrics infrastructure exists.
- Alerts:
  - None required by requirements.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Mutual-follower queries may become expensive on large follow graphs. | Slow profile loads or database load. | Use bounded list responses, pagination, and indexes/orderings appropriate to query shape. |
| RISK-002 | Project count has no discovered durable source and is hardcoded in current Flutter UI. | Requirements could be misread as requiring full project persistence. | Keep project persistence out of scope, but replace the Flutter hardcoded value with a real response/model field that can be `0` until project persistence exists. |
| RISK-003 | "Posts" could be interpreted differently by future contributors. | Recent activity or total posts could accidentally include replies/comments or repost interactions. | Requirements explicitly count top-level authored `craftsky_posts` rows only. |
| RISK-004 | Removing visible follower/following counts while API still returns them could confuse implementation review. | UI/API expectations drift. | Acceptance criteria focus on UI non-display and allow temporary API compatibility. |
| RISK-005 | New settings list routes/pages may expand Flutter navigation scope. | More test surface and potential UX edge cases. | Keep routes focused on settings-owned followers/following list views only. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Account age means Craftsky account/profile age based on AppView's `createdAt`, not DID/PDS creation age, and the formatted age is computed by Flutter. | AppView may need a different data source or API field if product wants a different account-age definition. |
| ASM-002 | Recent activity and total posts count top-level/root posts in `craftsky_posts`; replies/comments, repost interactions, likes, and follows do not count. | Requirements must change if product later wants replies, reposts, or other interactions included. |
| ASM-003 | Mutual-follower section is absent when there are no mutuals; when mutuals exist, the profile shows uncapped clickable count text. | UI requirements must change if product later wants an explicit zero-mutuals message. |
| ASM-004 | Followers/following list access requested from settings is for the signed-in user's own graph lists. | API/UI scope expands if users must browse any account's full follower/following lists. |
| ASM-005 | Project count should be data-driven now and can be `0` until project persistence exists. | Implementation scope expands if an accurate project persistence source must be created now. |
| ASM-006 | Firehose/indexing lag can make graph and post counts eventually consistent. | Product expectations must change if immediate strong consistency is required. |

## 21. Open Questions
None.

## 22. Review Status
Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date:
Notes: Plannotator and follow-up grilling feedback have been incorporated. This is a user-visible full-stack change touching AppView API shape, graph queries, Flutter profile UI, Flutter settings navigation, and tests. A second review is optional before test design.

## 23. Handoff To Test Design
- Requirements file: `docs/changes/2026-05-27-profile-social-summary/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - BR-001, BR-002, BR-003
  - FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017
  - NFR-001
  - RULE-001, RULE-002, RULE-003, RULE-004, RULE-005
- Suggested test levels:
  - AppView unit/integration tests for profile summary counts, top-level post counting, project count field behavior, mutual-follower count and list query behavior, follower/following list ordering, pagination, and auth/device enforcement.
  - Flutter model/API client tests for new response fields and list endpoints.
  - Flutter widget tests for profile stats, mutual-follower clickable count and 90%-height bottom sheet, settings entries without counts, app-bar counts on list pages, list empty states, non-Craftsky age hiding, and absence of follower/following counts on profile pages.
  - Regression tests for follow/unfollow profile state and existing posts/comments tabs.
- Blocking open questions: None.
