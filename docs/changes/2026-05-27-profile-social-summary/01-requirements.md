# Requirements: Profile Social Summary

## 1. Initial Request
Improve social interactions in the app by removing follower and following counts from profile pages and replacing them with:

- When viewing someone else's profile, show a list of accounts the viewer follows who also follow the current profile (mutual followers).
- A stat that shows the account's age.
- Recent activity shown as `X posts in the last 7 days`, including all types of posts.
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
  - Project-count storage is not yet visible in the inspected code; the existing project stat is currently UI-only/hardcoded.
- Test/build commands discovered:
  - AppView Go tests: `just test` after `just dev-d` is running.
  - Flutter tests are present under `app/test/**`; likely command is `flutter test` from `app/`.

## 3. Clarifying Questions And Decisions
### Q1: Which direction should the requirements document use for this change?
Answer: Profile summary API (Recommended).

Decision / implication: Add profile summary fields to existing profile responses, add list endpoints for mutuals/followers/following, and update Flutter profile/settings UI. Recent activity counts all Craftsky post rows; account age derives from the existing Craftsky profile `createdAt` value.

## 4. Candidate Approaches
### Option A: Profile summary API (Recommended)
Summary: Extend existing profile responses with profile-summary fields and add paginated list endpoints for mutual followers, followers, and following. Update Flutter profile and settings UI to consume those AppView contracts.

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
- Ambiguity around whether repost interactions should count as posts if they are not represented as `craftsky_posts` rows.

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
- NG-004: Do not build project persistence or redesign project tabs as part of this change.
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
Profile pages no longer display follower or following counts. Visitor profiles display mutual followers: accounts that the viewer follows and that also follow the profile being viewed. Profile stats show account age, recent activity as posts in the last 7 days, total posts, and the existing projects stat. Settings gives the signed-in user entry points to full follower and following lists, and each list is ordered by recency.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Profile pages must de-emphasize popularity by not showing follower or following counts. | Aligns social experience with meaningful context over popularity metrics. | Prompt | AC-001 |
| BR-002 | Business | Must | Visitor profiles must show viewer-specific mutual-follower context. | Helps viewers understand shared community connections. | Prompt, Q1 | AC-002, AC-003 |
| BR-003 | Business | Must | Users must retain access to their complete follower and following lists from settings. | Removing profile counts must not remove graph visibility for the account owner. | Prompt | AC-008, AC-009 |
| FR-001 | Functional | Must | The profile page shall remove follower-count and following-count cells from the visible profile stats area. | Directly satisfies the requested profile-page change. | Prompt, codebase | AC-001 |
| FR-002 | Functional | Must | When the signed-in viewer opens someone else's profile, the profile page shall show a mutual-followers section listing accounts the viewer follows who also follow the current profile. | Replaces count-based social proof with relationship-based context. | Prompt, Q1 | AC-002, AC-003, AC-004 |
| FR-003 | Functional | Must | Mutual followers shall be computed by AppView from the indexed follow graph, not by the Flutter client querying PDS. | Preserves architecture rules and centralizes graph logic. | AGENTS.md, discovery | AC-003 |
| FR-004 | Functional | Must | The profile summary shall expose and render account age derived from the profile account creation timestamp available to AppView. | Provides requested account-age stat. | Prompt, Q1, codebase | AC-005 |
| FR-005 | Functional | Must | The profile summary shall expose and render recent activity as the number of Craftsky post rows authored by the account in the last 7 days. | Provides requested recent-activity stat across current post types. | Prompt, Q1, codebase | AC-006 |
| FR-006 | Functional | Must | The profile summary shall expose and render total posts alongside the existing projects stat. | Adds total post visibility without removing projects. | Prompt | AC-007 |
| FR-007 | Functional | Must | Settings shall include tappable follower and following entries for the signed-in user. | Provides the requested access point after counts are removed from profiles. | Prompt, codebase | AC-008 |
| FR-008 | Functional | Must | Tapping followers in settings shall show the signed-in user's followers ordered by most recent follow first. | Satisfies list order requirement. | Prompt, discovery | AC-009 |
| FR-009 | Functional | Must | Tapping following in settings shall show accounts the signed-in user follows ordered by most recent follow first. | Satisfies list order requirement. | Prompt, discovery | AC-010 |
| FR-010 | Functional | Must | Follower, following, and mutual-follower list responses shall return display-ready account summaries including stable DID, current handle, and available profile display fields. | Allows Flutter to render account lists without PDS reads. | Discovery | AC-011 |
| FR-011 | Functional | Should | Large graph lists should paginate using existing opaque cursor conventions. | Prevents unbounded responses and matches API design. | API architecture spec | AC-012 |
| FR-012 | Functional | Should | The API may continue returning old `followerCount` and `followingCount` fields temporarily for compatibility, but Flutter shall not display them on profile pages. | Avoids unnecessary breaking API work while satisfying UX. | Discovery | AC-001 |
| NFR-001 | Non-functional | Must | New or changed AppView API fields and endpoints must use `/v1/`, authenticated requests, camelCase JSON, error envelopes, and opaque cursor pagination for lists. | Maintains API consistency. | AGENTS.md, API spec | AC-012, AC-013 |
| NFR-002 | Non-functional | Should | Social graph and activity summary queries should use bounded result sizes and indexed order/filter columns where applicable. | Reduces performance risk for profiles with larger graphs. | Discovery risk | AC-014 |
| RULE-001 | Business rule | Must | Mutual followers are accounts where `viewer -> mutual` and `mutual -> profile` follow records both exist in AppView's active indexed follow graph. | Defines mutuals precisely. | Prompt, Q1 | AC-003 |
| RULE-002 | Business rule | Must | The account age stat is based on the profile's Craftsky account/profile creation timestamp known to AppView. | Uses existing available timestamp and avoids external account-age dependencies. | Q1, codebase | AC-005 |
| RULE-003 | Business rule | Must | Recent activity counts all authored rows in `craftsky_posts` within the trailing 7-day window, including root posts and replies/comments; likes and follows do not count. | Matches confirmed direction and current data model. | Prompt, Q1, codebase | AC-006 |
| RULE-004 | Business rule | Must | Followers list recency is based on the follow record's `created_at` where another account followed the signed-in user; following list recency is based on the signed-in user's follow record `created_at`. | Defines deterministic list ordering. | Prompt, discovery | AC-009, AC-010 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-012 | Given any profile page, when it is rendered, then follower and following counts are not visible in the profile stats area. |
| AC-002 | BR-002, FR-002 | Given a signed-in viewer opens another Craftsky profile with at least one mutual follower, when the profile loads, then the mutual-follower section displays those mutual account summaries. |
| AC-003 | BR-002, FR-002, FR-003, RULE-001 | Given the indexed follow graph contains `viewer -> mutual` and `mutual -> profile`, when AppView builds the mutual followers list, then that mutual account is included; accounts missing either relationship are excluded. |
| AC-004 | FR-002 | Given a signed-in viewer opens their own profile, when the profile loads, then the mutual-follower section is not shown. |
| AC-005 | FR-004, RULE-002 | Given a profile has a known Craftsky creation timestamp, when the profile renders, then an account-age stat derived from that timestamp is visible. |
| AC-006 | FR-005, RULE-003 | Given an account authored Craftsky post rows inside and outside the trailing 7-day window, when the profile summary is fetched, then `posts in the last 7 days` includes only rows within the trailing 7-day window and includes both root posts and replies/comments. |
| AC-007 | FR-006 | Given a profile summary has total post data, when the profile renders, then total posts are shown alongside the existing projects stat. |
| AC-008 | BR-003, FR-007 | Given the signed-in user opens settings, when the settings page renders, then followers and following entries are visible and tappable. |
| AC-009 | BR-003, FR-008, RULE-004 | Given the signed-in user has followers with different follow `created_at` values, when they tap the followers settings entry, then the follower list is ordered newest follow first. |
| AC-010 | FR-009, RULE-004 | Given the signed-in user follows accounts at different times, when they tap the following settings entry, then the following list is ordered newest follow first. |
| AC-011 | FR-010 | Given a follower, following, or mutual-follower list response contains accounts, when Flutter renders the list, then each row has enough AppView-provided account data to display identity without a PDS read. |
| AC-012 | FR-011, NFR-001 | Given a graph list contains more than one page of accounts, when the client requests pages with `limit` and returned cursor values, then pagination follows the existing opaque-cursor response convention. |
| AC-013 | NFR-001 | Given an unauthenticated or missing-device request reaches a new `/v1/` social graph endpoint, when AppView handles it, then it returns the existing authenticated-device error behavior. |
| AC-014 | NFR-002 | Given a profile has many followers/following records or posts, when AppView serves the new summary/list data, then responses are bounded and ordered/filterable without unbounded client-side filtering. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | No mutual followers | Visitor profile shows an empty or absent mutual-follower state without showing follower/following counts. | FR-001, FR-002 |
| EC-002 | Mutual account has no cached Bluesky profile row | List still includes stable identity if available, with missing optional display fields handled gracefully. | FR-010 |
| EC-003 | Non-Craftsky profile is viewed | Profile should not invent Craftsky-only stats if required indexed data is unavailable; UI uses a graceful unavailable/empty state. | FR-004, FR-005, FR-006 |
| EC-004 | Account has zero posts in last 7 days | Recent activity displays zero activity in the chosen copy format. | FR-005 |
| EC-005 | Account has posts exactly on the 7-day boundary | Boundary handling is deterministic and covered by tests against AppView's chosen time comparison. | RULE-003 |
| EC-006 | Follower/following list is empty | Settings list page shows an empty state, not an error. | FR-008, FR-009 |
| EC-007 | Handle resolution fails for a profile/list account | AppView follows existing identity error behavior for target resolution; list rows with unavailable optional handles are handled according to the final API design. | NFR-001, FR-010 |

## 15. Data / Persistence Impact
- New fields:
  - Expected additive profile summary fields include total post count and posts-in-last-7-days count. Exact JSON names should use camelCase.
  - Account age may be derived client-side from existing `createdAt` or exposed as an additive summary field; the source timestamp remains AppView-known Craftsky profile creation time.
  - Graph list account summaries should include DID, handle, and optional display fields.
- Changed fields:
  - Flutter no longer displays `followerCount` or `followingCount` on profile pages.
  - Existing API fields may remain temporarily for compatibility but are no longer profile-page UI requirements.
- Migration required:
  - No new durable table is expected from requirements discovery.
  - Implementation may add indexes if performance review shows existing indexes are insufficient for mutuals or recency ordering.
- Backwards compatibility:
  - Additive API response fields and routes are preferred under `/v1/`.
  - Removing old response fields is not required for this change and would be a breaking API decision.

## 16. UI / API / CLI Impact
- UI:
  - Profile stats area changes from follower/following/projects to account age, recent activity, total posts, and projects.
  - Visitor profiles add a mutual-follower section.
  - Settings gains followers/following entries and list pages.
- API:
  - Extend existing profile summary responses with scalar social/activity stats.
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
| RISK-002 | Project count has no discovered durable source and is hardcoded in current Flutter UI. | Requirements could be misread as requiring project persistence. | Keep project persistence out of scope; preserve existing projects stat behavior unless a project source already exists elsewhere. |
| RISK-003 | "All types of posts" could be interpreted to include repost interaction records. | Recent activity could disagree with product expectations. | Requirements define current scope as all `craftsky_posts` rows; record repost counting as an assumption/open question for future product decision. |
| RISK-004 | Removing visible follower/following counts while API still returns them could confuse implementation review. | UI/API expectations drift. | Acceptance criteria focus on UI non-display and allow temporary API compatibility. |
| RISK-005 | New settings list routes/pages may expand Flutter navigation scope. | More test surface and potential UX edge cases. | Keep routes focused on settings-owned followers/following list views only. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Account age means Craftsky account/profile age based on AppView's `createdAt`, not DID/PDS creation age. | AppView may need a different data source not currently discovered. |
| ASM-002 | Recent activity counts root posts and replies/comments in `craftsky_posts`; likes/follows do not count. | Requirements must change if product wants likes, follows, or repost interactions included. |
| ASM-003 | Mutual-follower section can be absent or empty when there are no mutuals. | UI acceptance criteria may need explicit empty-state copy if product wants always-visible messaging. |
| ASM-004 | Followers/following list access requested from settings is for the signed-in user's own graph lists. | API/UI scope expands if users must browse any account's full follower/following lists. |
| ASM-005 | Existing projects stat remains as-is for this change; no new project persistence is required. | Implementation scope expands if an accurate project count must be created now. |

## 21. Open Questions
- [ ] Non-blocking: Should repost interaction records count toward "posts in the last 7 days" in a future product model, or only post records authored in `craftsky_posts`?
- [ ] Non-blocking: What exact copy/format should account age use (for example, `Joined 2y ago`, `2 years on Craftsky`, or `Account age: 2y`)?
- [ ] Non-blocking: Should the mutual-follower profile section show all mutuals through pagination, a capped preview, or a capped preview with a "view all" action?

## 22. Review Status
Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date:
Notes: This is a user-visible full-stack change touching AppView API shape, graph queries, Flutter profile UI, Flutter settings navigation, and tests. Plannotator review is recommended before test design but not strictly required.

## 23. Handoff To Test Design
- Requirements file: `docs/changes/2026-05-27-profile-social-summary/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - BR-001, BR-002, BR-003
  - FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010
  - NFR-001
  - RULE-001, RULE-002, RULE-003, RULE-004
- Suggested test levels:
  - AppView unit/integration tests for profile summary counts, mutual-follower query behavior, follower/following list ordering, pagination, and auth/device enforcement.
  - Flutter model/API client tests for new response fields and list endpoints.
  - Flutter widget tests for profile stats, mutual-follower section, settings entries, list empty states, and absence of follower/following counts on profile pages.
  - Regression tests for follow/unfollow profile state and existing posts/comments tabs.
- Blocking open questions: None.
