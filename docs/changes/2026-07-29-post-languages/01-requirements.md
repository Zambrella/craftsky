# Requirements: Post And Content Languages

## 1. Initial Request

Add support for posting in different languages in CraftSky, based on Bluesky's language model:

- Each post can be tagged with up to three languages.
- Settings expose an App language, a Primary language, and any number of Content languages.
- CraftSky supports only English as an App language initially, while making future App languages visible as a planned capability.
- The Primary language is the default language selected when composing a post.
- Content languages determine which posts appear in the user's feed.
- Primary and Content language preferences are stored privately in the AppView, per CraftSky account.
- A user's first Primary and Content language preferences are seeded from the device's ordered locale information.
- The existing post Lexicon must be checked and changed only if required.

The supplied Bluesky Languages screenshot is the visual and terminology reference.

## 2. Current Codebase Findings

### Relevant files

- `lexicon/social/craftsky/feed/post.json`
  - Already defines optional `langs`.
  - `langs` is an array whose items use Lexicon string format `language`.
  - The array already has `maxLength: 3`.
- `appview/internal/lexicon/craftsky/feedpost.go`
  - Generated `FeedPost` already exposes `Langs []string`.
- `app/lib/feed/widgets/post_composer_sheet.dart`
  - General posts, replies, and quotes use the same composer.
  - The composer currently sends text, facets, images, reply data, and quote data, but no languages.
- `app/lib/projects/widgets/project_composer_sheet.dart` and `app/lib/projects/composer/`
  - Project posts have a separate composer pipeline and also create `social.craftsky.feed.post` records.
- `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`, and `app/lib/feed/providers/create_post_provider.dart`
  - The Flutter create-post seam has no language field.
- `appview/internal/api/post_request.go` and `appview/internal/api/post.go`
  - `POST /v1/posts` does not accept, validate, or write `langs`.
  - Synthetic create responses do not carry languages.
- `appview/internal/index/craftsky_post.go`
  - The indexer decodes the generated record type but does not materialise `rec.Langs`.
- `appview/migrations/000010_craftsky_posts.up.sql`
  - `craftsky_posts` has no materialised language column.
  - The full source record is retained in `record JSONB`.
- `appview/internal/api/timeline_store.go`
  - `GET /v1/feed/timeline` is filtered and paginated in Postgres before hydration.
  - Its current eligibility rules cover following, mutes, blocks, replies, imports, and moderation, but not languages.
- `appview/internal/routes/routes.go`
  - Current post-returning browse/discovery routes also include Projects browse, post/project/hashtag search, and profile post/project/comment lists.
  - Direct posts, post comments/replies, notifications, and Saved Posts have distinct routes and stores, so the language-policy exceptions must be explicit rather than accidental.
- `appview/internal/api/post_response.go` and `app/lib/feed/models/post.dart`
  - Canonical post responses and Flutter post models do not expose languages.
- `app/lib/settings/pages/settings_page.dart` and `app/lib/router/router.dart`
  - Settings has no Languages destination.
- `app/lib/app_dependencies.dart`
  - `SharedPreferences` is already available for the device-local App language and for an optional cache of server-owned preferences.
  - Flutter exposes the device's ordered locale list through `PlatformDispatcher.instance.locales`; the current bootstrap reads only `PlatformDispatcher.instance.locale` for date formatting.
- `app/lib/onboarding/providers/onboarding_status_provider.dart`
  - Provides an existing Riverpod pattern for preferences that survive app relaunch.
- `appview/internal/api/notification_preferences.go`, `appview/internal/notifications/preferences.go`, and `appview/migrations/000021_appview_notifications.up.sql`
  - Provide an existing account-DID-scoped pattern for private AppView preferences, validation, effective defaults, and atomic persistence.
- `app/lib/l10n/app_en.arb`
  - English is the only current app localisation.

### Existing patterns

- Published post metadata belongs in the public PDS record; the AppView indexes it for reads.
- UI-only preferences can remain device-local, while account behavior that must follow a user across devices can be stored privately in the AppView.
- The Flutter app reads feeds only from the AppView.
- Browse/discovery filtering belongs in each AppView candidate query so pagination remains correct.
- AppView-backed preferences are keyed by authenticated account DID, matching the notification-preference precedent.
- API request and response bodies use camelCase.
- List endpoints use opaque cursors.
- CraftSky supports multiple signed-in accounts on one device.

### Current behavior

- CraftSky posts are normally created without `langs`, even though the Lexicon permits them.
- Posts are returned without language metadata.
- The chronological home timeline includes eligible posts regardless of language.
- The AppView has no account language-preference table or API.
- Users cannot declare which languages they post in or can read.
- The app UI is English-only and has no visible App language setting.

### Constraints discovered

- No Lexicon edit is required for the requested post-language shape. Editing the existing `langs` definition merely to match Bluesky would create unnecessary Lexicon and ADR work.
- AppView storage, request validation, response models, and filtering still require changes.
- Private Primary and Content language persistence requires an account-scoped AppView migration and authenticated preference API.
- App language must remain available before sign-in, so it remains device-local even though Primary and Content languages are server-owned.
- Device-derived initialisation must be create-if-absent: signing in on another device must never overwrite an existing account preference.
- Filtering after a page is returned would create short pages, incorrect cursors, and transient display of mismatched posts.
- The cross-surface rule must be applied consistently to the home timeline, Projects browse, post/project/hashtag search, and other users' profile post/project/comment queries.
- Direct post/thread context, notifications, Saved Posts, and viewer-authored content must bypass language filtering intentionally.
- Every composer that writes `social.craftsky.feed.post` must use the same language-selection behavior.
- CraftSky is not in production and has no active users, so no production backfill is required. Untagged development records still need defined behavior.

### Test/build commands discovered

- Flutter tests: run focused tests with `flutter test <test paths>` from `app/`.
- Flutter static analysis: run `dart analyze` from `app/`.
- Go tests: run focused package tests from `appview/`.
- Canonical repository verification: run `just test` from the repository root against the compose Postgres.
- Lexicon generation: `just lexgen` is required only if a Lexicon file changes; this requirements slice does not currently call for such a change.
- Flutter acceptance tests use repository fakes and prove the client workflow, not a live AppView/PDS/Tap/Postgres round trip. Go/Postgres integration tests and any explicitly run compose-stack smoke test are separate verification evidence.

### Bluesky research findings

Research was checked against official Bluesky sources on 2026-07-29:

- [`app.bsky.feed.post`](https://github.com/bluesky-social/atproto/blob/d3bbeb5fe87f8c389c2f18abd2bc055ef916a63a/lexicons/app/bsky/feed/post.json) defines optional `langs` as at most three `language`-formatted strings.
- Bluesky's [Languages settings screen](https://github.com/bluesky-social/social-app/blob/27e4f84f3fb7429855a72377c307710eff910c76/src/screens/Settings/LanguageSettings.tsx) separates App language, Primary language, and Content languages. It states that selecting no Content languages shows all languages.
- Bluesky's [persisted language-preference schema](https://github.com/bluesky-social/social-app/blob/27e4f84f3fb7429855a72377c307710eff910c76/src/state/persisted/schema.ts) keeps these preferences locally, uses two-letter BCP-47 language codes for primary/content/post languages, and defaults from device languages.
- Bluesky's [post language selector](https://github.com/bluesky-social/social-app/blob/27e4f84f3fb7429855a72377c307710eff910c76/src/view/com/composer/select-language/PostLanguageSelect.tsx) lets users select up to three post languages.
- Bluesky's [post write pipeline](https://github.com/bluesky-social/social-app/blob/27e4f84f3fb7429855a72377c307710eff910c76/src/lib/api/index.ts) truncates the selected language list to three and writes it into each post record.
- Bluesky's [custom-feed client](https://github.com/bluesky-social/social-app/blob/27e4f84f3fb7429855a72377c307710eff910c76/src/lib/api/feed/custom.ts) sends Content languages via `Accept-Language`.
- Bluesky's separate Following timeline client does not currently send Content languages. CraftSky will copy the preference and record model while applying the user's stricter readability requirement consistently across CraftSky's post-browsing and discovery surfaces.
- Bluesky currently describes Primary language as the translation target and separately persists the current post language. CraftSky intentionally follows the user's requested simpler meaning: Primary language is the default post language. Translation is not part of this slice.
- Bluesky initialises Primary, Content, and posting languages from the device's ordered language codes. CraftSky will copy that first-use behaviour while intentionally storing Primary and Content languages privately per account in the AppView so they synchronise across CraftSky devices.

## 3. Clarifying Questions And Decisions

### Q1: Does Primary language mean translation target or default posting language?

Answer: Default posting language, per the initial request.

Decision / implication: New composers start with the Primary language selected. Translation behavior is out of scope.

### Q2: What happens when no Content languages are selected?

Answer: Follow the Bluesky behavior shown in the supplied reference.

Decision / implication: No selected Content languages means all languages are shown. The UI must explain this before the user clears the final selection.

### Q3: What happens to untagged posts?

Answer: The user confirmed that untagged posts are hidden while filtering is active and shown when Content languages is empty.

Decision / implication: When one or more Content languages are selected, untagged posts are excluded from filtered browse and discovery surfaces. When no Content languages are selected, untagged posts are included. All new posts created by the CraftSky app have at least one language tag, so untagged records are a legacy, import, or non-CraftSky-client case.

### Q4: Which surfaces are filtered?

Answer: The user confirmed that language-mismatched posts should be hidden throughout browsing and discovery, with explicit direct, contextual, notification, saved-content, and ownership exceptions.

Decision / implication: V1 filtering applies to the home timeline, Projects browse, post/project/hashtag search results, and other users' profile post/project/comment lists. Direct post and thread context, all notifications, Saved Posts, and the viewer's own content remain visible regardless of language.

### Q5: Where are language preferences stored?

Answer: The user confirmed that account language preferences are saved privately in the AppView.

Decision / implication: Primary and Content languages are private, per-DID AppView preferences and follow the account across devices. App language remains device-local because it controls the UI before sign-in and may reasonably differ by device. An optional local cache may improve startup behavior, but the AppView is authoritative. Public post tags remain portable in PDS records.

### Q6: How are an account's first language preferences chosen?

Answer: The user confirmed that CraftSky should use the device-provided locale information.

Decision / implication: When the authenticated account has no stored language preferences, Flutter reads the device's ordered locale list, converts it to the supported v1 base-language catalogue, removes duplicates while preserving order, and proposes the first valid language as Primary and the complete valid list as Content languages. If no supported language remains, both fall back to English. AppView initialisation is atomic and create-if-absent; an existing stored preference always wins.

### Q7: How do multilingual posts match Content languages?

Answer: The user confirmed any-match semantics.

Decision / implication: A post with multiple tags is eligible when at least one tag matches a selected Content language. The reader does not need to select every language attached to the post.

### Q8: How do reposts and quote posts match?

Answer: The user confirmed that reposts use the original post's languages and quote posts use the outer post's languages.

Decision / implication: A repost is excluded when its subject does not match. An eligible outer quote post remains visible with its quoted post shown as context even when the quoted post's languages do not match.

### Q9: What should a new composer select after a one-off language change, and what should replies use?

Answer: The user confirmed that every newly opened composer, including a reply composer, starts from Primary.

Decision / implication: One-off post-language selections are not remembered as a second posting default, and a reply does not inherit its parent post's languages. An already-open composer retains its explicit selection.

### Q10: Which language catalogue and tag granularity does v1 use?

Answer: The user confirmed Bluesky-compatible base languages for v1.

Decision / implication: The selector uses the full Bluesky-compatible post/content base-language catalogue rather than the much smaller App-language catalogue. Region and script variants are not selectable in v1; valid external BCP-47 tags remain preserved as record data but do not implicitly match a base selection until language-range matching is introduced.

## 4. Candidate Approaches

### Option A: Per-account AppView preferences with device-locale initialisation

Summary: Keep App language device-local; store Primary and Content languages privately by account DID in the AppView; seed an account's first values from its device's ordered locales; write one to three language tags into public post records; and apply stored Content preferences to every in-scope post-list query in Postgres.

Pros:

- Preserves Bluesky's user model and AT Protocol record shape while fitting CraftSky's multi-account architecture.
- Synchronises Primary and Content languages across devices.
- Keeps private preferences out of public PDS data.
- Prevents one signed-in account from inheriting another account's reading or posting preferences.
- Keeps language filtering before pagination across browse and discovery queries.
- Preserves public, portable language metadata on posts.
- Does not require a Lexicon change.

Cons:

- Requires private preference routes, a migration, and additional server state.
- First-use initialisation needs an explicit create-if-absent flow.
- Clients need a loading/error state while authoritative preferences are fetched.
- Every post-list query and corresponding Flutter cache needs consistent visibility handling.

Risks:

- Untagged posts disappear when filters are active.
- Concurrent first use on two devices could race unless the server initialisation is atomic.
- A stale local cache could briefly disagree with the AppView if it is treated as authoritative.
- A missed endpoint or cache could expose language-mismatched content inconsistently.

### Option B: Bluesky-style device-local preferences

Summary: Store App, Primary, and Content languages locally and send Content languages with every post-list request.

Pros:

- Closest to Bluesky's current client persistence.
- Avoids a server preference API and preference table.
- Can initialise immediately without a network round trip.

Cons:

- Does not synchronise across devices.
- Requires careful per-account local keying in CraftSky's multi-account client.
- The client must transmit Content languages on every affected list request.
- Reinstalling or clearing app data loses the preferences.

Risks:

- Account switching or incorrect local namespacing can leak one account's preferences into another.
- Browser or platform locale headers can be confused with explicit Content-language selections.

### Option C: Client-side surface filtering

Summary: Return normal post lists and remove non-matching posts in Flutter.

Pros:

- Smallest AppView change.
- No server-side language filter contract.

Cons:

- Breaks page-size and cursor semantics across paginated surfaces.
- Can show disallowed posts briefly from caches or optimistic updates.
- Requires over-fetching and repeated client filtering.
- Duplicates filtering logic across clients.

Risks:

- Users can receive empty or very short pages even when matching posts exist.
- Different clients and screens can disagree about the same post.

## 5. Recommended Direction

Recommended approach: Option A.

Why:

- It matches Bluesky's public `langs` record field, three-language composer limit, and device-locale defaults.
- It improves on device-global persistence for CraftSky by making Primary and Content languages private, per-account, and available on every device.
- It satisfies CraftSky's AppView architecture by filtering browse and discovery reads in authoritative AppView queries.
- It preserves each surface's existing ordering; language eligibility filters the candidate set but does not rank matching posts.
- It keeps public post metadata portable, account reading preferences private, and App language device-specific.
- It avoids an unnecessary Lexicon change.

The proposed API contract is:

- `POST /v1/posts` accepts optional camelCase `langs: string[]`.
- Canonical post responses expose `langs: string[]`.
- `GET /v1/languages/preferences` returns the authenticated account's Primary and Content languages.
- `PUT /v1/languages/preferences` completely and atomically replaces the authenticated account's Primary and Content languages.
- `POST /v1/languages/preferences/initialize` accepts the device-derived proposal, inserts it only when the authenticated account has no preference row, and returns the authoritative stored preferences whether it inserted or found an existing row.
- In-scope post-list endpoints read the authenticated viewer's stored Content languages; callers do not send `Accept-Language` as the preference source.
- Filtering covers the home timeline, Projects browse, post/project/hashtag search, and other users' profile post/project/comment lists.
- Direct post and thread context, notifications, Saved Posts, and the viewer's own posts are not language-filtered.
- An empty stored Content-language list means show all.

## 6. Problem / Opportunity

Crafting is international, but CraftSky currently neither records a post's language nor lets readers control which languages appear while browsing the app. Without language tags, multilingual authors cannot accurately describe their content and readers may receive posts they cannot understand.

The existing Lexicon already provides the interoperable AT Protocol field needed to solve this. Completing the client, API, indexing, persistence, and post-list filtering paths makes multilingual posting useful without changing the ordering policy of any CraftSky surface.

## 7. Goals

- G-001: Let an author declare one to three languages used in every post created by the CraftSky app.
- G-002: Default every new composer to the user's Primary language.
- G-003: Let a user select any number of Content languages for post-browsing and discovery surfaces.
- G-004: Exclude other people's non-matching and untagged posts from in-scope browse and discovery surfaces when Content-language filtering is active.
- G-005: Show all posts when no Content languages are selected.
- G-006: Expose the English-only App language now without requiring a settings redesign when more UI localisations are added.
- G-007: Preserve every surface's existing ordering, pagination correctness, and AT Protocol interoperability.
- G-008: Synchronise Primary and Content languages privately across devices without exposing them in the user's PDS.
- G-009: Produce sensible first-use preferences from the device's ordered locales without overwriting an existing account preference.
- G-010: Keep direct and contextual content, notifications, Saved Posts, and the viewer's own posts available regardless of language.

## 8. Non-Goals

- NG-001: Translating post text or project metadata.
- NG-002: Automatically detecting a post's language.
- NG-003: Suggesting a language based on typed text or the post being replied to.
- NG-004: Filtering direct post views, thread context, notifications, or Saved Posts.
- NG-005: Hiding or translating quote-preview content separately from its outer post.
- NG-006: Synchronising App language between devices.
- NG-007: Supporting more than English for CraftSky's UI in this slice.
- NG-008: Storing Primary or Content languages in public PDS records or an AT Protocol preference Lexicon.
- NG-009: Backfilling language tags onto existing PDS records.
- NG-010: Editing `lexicon/social/craftsky/feed/post.json` when its existing `langs` definition already meets the requirement.
- NG-011: Language-aware ranking or algorithmic feed ordering.
- NG-012: Detecting or backfilling language tags for imported Instagram posts; normal untagged-post and viewer-ownership rules apply.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Reader | A signed-in CraftSky user browsing posts, projects, search results, and profiles on one or more devices. | Discovery lists limited to their account's stored languages, with deliberate access to direct, contextual, saved, notification, and own content. |
| Author | A CraftSky user creating a general, project, reply, or quote post. | A cross-device Primary default and a way to tag a post with up to three languages. |
| Multilingual author | An author whose post genuinely uses more than one language. | The ability to select multiple distinct post languages without exceeding the Lexicon limit. |
| Future localised-app user | A user who may later choose a non-English CraftSky interface. | A visible App language model that can expand without changing Primary or Content-language semantics. |
| First-time account | An authenticated account with no stored language preferences. | Safe defaults derived once from the device's ordered locales. |
| AppView | The read and write mediator between Flutter, private account preferences, PDS records, and indexed post surfaces. | Atomic preference initialisation, validated language inputs, materialised post languages, and consistent filtering before pagination. |

## 10. Current Behavior

The settings page has no Languages destination. New posts contain no language tags even though the post Lexicon supports them. The AppView stores and returns no materialised language data, and post-browsing surfaces serve all otherwise-eligible posts regardless of language.

## 11. Desired Behavior

Settings includes a Languages page modelled on the supplied Bluesky screen:

- App language shows English and explains that additional App languages are coming.
- Primary language selects the account's cross-device default language for new posts.
- Content languages allows any number of unique per-account selections that follow the account across devices.
- The page explains that clearing all Content languages shows all languages.

When an account has no language preferences, the client derives an ordered, deduplicated set of supported base languages from the device locale list. The first language becomes the proposed Primary and the complete set becomes the proposed Content languages, with English fallbacks. The AppView creates these values only if the account is still uninitialised and returns the authoritative stored values.

Every newly opened CraftSky composer, including replies, starts with only the current Primary language selected and allows one to three post languages. One-off selections do not alter the next composer. On a successful create, those language tags travel through Flutter, the AppView API, the public PDS record, the firehose indexer, Postgres, and canonical post responses.

The home timeline, Projects browse, post/project/hashtag search, and other users' profile post/project/comment lists are filtered before pagination using the authenticated viewer's AppView preferences. With selected Content languages, a post is eligible when at least one of its tags matches at least one selected Content language. Untagged posts and posts with no matching tag are excluded. With no selected Content languages, language eligibility imposes no restriction.

Language filtering does not hide the viewer's own posts. Direct post views and complete thread context show posts as-is, including quoted context. Notifications and Saved Posts also remain visible regardless of language. Reposts use their subject post's languages for discovery eligibility; quote posts use the outer post's languages.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | CraftSky shall let authors identify the human language or languages used in a post. | Makes multilingual content explicit and portable. | Prompt; Bluesky research | AC-001, AC-006, AC-007, AC-009 |
| BR-002 | Business | Must | CraftSky shall let readers limit post-browsing and discovery surfaces to languages they selected. | Readers should not have to scan content they cannot read while browsing the app. | Prompt; grilling decision Q4 | AC-001, AC-010, AC-011, AC-012, AC-024 |
| BR-003 | Business | Must | CraftSky shall provide the App, Primary, and Content language settings as distinct concepts. | Matches the requested and familiar Bluesky model. | Prompt; screenshot; Bluesky research | AC-001, AC-002, AC-003, AC-004, AC-005 |
| BR-004 | Business | Should | The language model should be ready for additional App localisations without changing post or feed preference semantics. | Avoids a settings redesign when CraftSky is translated. | Prompt | AC-003 |
| FR-001 | Functional | Must | Settings shall contain a Languages destination and a dedicated Languages page. | Makes the feature discoverable. | Prompt; screenshot; codebase | AC-002 |
| FR-002 | Functional | Must | App language shall show English as the only available UI language and visibly state that more App languages are coming. | Sets accurate expectations while exposing the future setting. | Prompt | AC-003 |
| FR-003 | Functional | Must | Primary language shall allow exactly one supported language, persist privately for the authenticated account in the AppView, and be the sole initial selection in every newly opened composer. | Provides a reliable posting default on every device without turning one-off choices into a second default. | Prompt; decisions Q1, Q5, and Q9 | AC-004, AC-006, AC-016, AC-026 |
| FR-004 | Functional | Must | Content languages shall allow any number of unique supported languages, persist privately for the authenticated account in the AppView, and permit the user to clear all selections. | Represents all languages a reader can read while retaining Bluesky's show-all escape hatch across devices. | Prompt; Bluesky research; decisions Q2 and Q5 | AC-005, AC-011, AC-016 |
| FR-005 | Functional | Must | General, project, reply, and quote composers shall expose the same post-language selector and require one to three distinct languages before submission. | Every CraftSky post record uses the same Lexicon and must follow the same rule. | Codebase; Bluesky research | AC-006, AC-007 |
| FR-006 | Functional | Must | `POST /v1/posts` shall accept optional `langs`, validate each supplied value, reject more than three or duplicate values, and preserve valid values in the PDS write. | Enforces the public record contract at the server boundary. | Codebase; Lexicon; API spec | AC-008, AC-009 |
| FR-007 | Functional | Must | The CraftSky app shall include its selected one-to-three language values in every post create request. | New CraftSky-authored posts must not be untagged. | Prompt; decision Q3 | AC-009 |
| FR-008 | Functional | Must | The AppView indexer shall materialise post languages from `social.craftsky.feed.post.langs`, preserving an absent field as an untagged post. | Cross-surface filtering must be efficient and distinguish legacy/nonconforming records. | Codebase | AC-009, AC-010, AC-011 |
| FR-009 | Functional | Must | The AppView shall apply the authenticated viewer's stored Content-language eligibility before pagination to `GET /v1/feed/timeline`, Projects browse, post/project/hashtag search, and other users' profile post/project/comment lists. | Prevents incorrect pages and makes every browse/discovery surface use the same authoritative preference. | Prompt; architecture; decisions Q4 and Q5 | AC-010, AC-011, AC-012, AC-024 |
| FR-010 | Functional | Must | Canonical post responses shall expose `langs` as a camelCase JSON array, using an empty array for untagged posts. | Keeps Flutter models, synthetic responses, indexed responses, and optimistic caches consistent. | Codebase; API conventions | AC-015 |
| FR-011 | Functional | Must | After a Content-language update succeeds in the AppView, Flutter shall invalidate affected browse/discovery caches, discard incompatible loaded items, and restart paginated results with the stored preference; a failed update shall not change effective filtering. | Previously loaded rows and cursors belong to a different eligibility set on every affected surface. | Codebase; decisions Q4 and Q5 | AC-014, AC-024 |
| FR-012 | Functional | Should | Language selectors should use a versioned snapshot of the full base-language catalogue used by the cited Bluesky app revision, show user-friendly English names, support efficient lookup, and prevent duplicate selections. | Posting and reading languages must not be limited to CraftSky's currently available App localisations. | Screenshot; Bluesky research; decision Q10 | AC-005, AC-007, AC-017, AC-027 |
| FR-013 | Functional | Must | The AppView shall expose authenticated `GET /v1/languages/preferences`, `PUT /v1/languages/preferences`, and `POST /v1/languages/preferences/initialize` operations for the caller's Primary and Content language preferences. `PUT` is a complete atomic replacement; `initialize` is create-if-absent and always returns the authoritative row. All three operations derive the account DID only from authentication. | Makes the private server state accessible without permitting cross-account access or ambiguous update semantics. | User amendment; notification-preference precedent; document review DR-002 | AC-016, AC-019, AC-020, AC-022 |
| FR-014 | Functional | Must | For an account with no stored preferences, Flutter shall derive the proposed first preference set from `PlatformDispatcher.instance.locales` in device preference order, normalise to supported unique v1 base-language tags, use the first as Primary, use the full list as Content languages, and fall back to `en` and `["en"]` when none remain. | Gives a multilingual user useful first-run defaults without asking redundant questions. | User amendment; Bluesky research | AC-019 |
| FR-015 | Functional | Must | AppView preference initialisation shall atomically create values only when the account has no stored language preferences and shall always return the authoritative stored result. | Prevents a new device or concurrent first use from overwriting an existing choice. | User amendment; discovery | AC-020 |
| FR-016 | Functional | Must | The AppView shall validate preference initialisation and updates as one complete unit: Primary must be one supported base-language tag, Content languages must be supported and unique, and invalid input shall leave the stored preference unchanged. | Prevents malformed or partially applied private preferences from changing post visibility. | User amendment; API conventions | AC-022 |
| FR-017 | Functional | Must | Flutter shall load or initialise the active account's authoritative preferences before enabling post submission or any filtered browse/discovery surface. | Prevents temporary use of device defaults, stale cache data, or another account's settings. | User amendment; multi-account architecture | AC-023 |
| NFR-001 | Non-functional | Must | Post, Primary, and Content languages shall use valid BCP-47 language tags; the v1 UI catalogue shall use unique two-letter base language codes where available. | Matches the Lexicon `language` format and Bluesky's interoperable preference model. | Lexicon skill; Bluesky research | AC-008, AC-022, AC-027, AC-028 |
| NFR-002 | Non-functional | Must | Language filtering shall preserve each affected surface's existing ordering and pagination semantics among eligible items. | Language is an eligibility filter, not a ranking algorithm or reordering step. | Product vision; API architecture | AC-012, AC-024 |
| NFR-003 | Non-functional | Must | App language shall remain device-local; Primary and Content languages shall be private per-DID AppView data; server logs and errors shall not record their complete values. | Keeps UI choice device-specific while synchronising account behavior and minimising preference exposure. | User amendment; architecture; decision Q5 | AC-016, AC-018 |
| NFR-004 | Non-functional | Should | The Languages page and composer selector should be keyboard, screen-reader, and large-text accessible and identify selection state without relying on colour alone. | Language controls are core settings and must be usable accessibly. | Screenshot; Flutter conventions | AC-017 |
| NFR-005 | Non-functional | Must | Language preference reads, writes, caches, and in-flight results shall remain isolated by account DID during account switching and multi-device use. | Prevents one account's private preferences or visibility behavior from leaking into another. | User amendment; multi-account architecture | AC-016, AC-021, AC-022, AC-023 |
| RULE-001 | Business rule | Must | A post may contain no more than three distinct language tags. | Mirrors the existing CraftSky and Bluesky Lexicons. | Existing Lexicon; prompt | AC-007, AC-008 |
| RULE-002 | Business rule | Must | With one or more Content languages selected, an in-scope post is eligible when any post language matches any selected Content language; an untagged post is ineligible unless an explicit exception applies. | Defines deterministic multilingual and legacy behavior. | Prompt; decisions Q3 and Q7 | AC-010, AC-024 |
| RULE-003 | Business rule | Must | With no Content languages selected, all otherwise-eligible posts are language-eligible, including untagged posts. | Copies the explicit Bluesky show-all behavior. | Bluesky research; decision Q2 | AC-011 |
| RULE-004 | Business rule | Must | Repost discovery eligibility uses the reposted subject post's languages; quote-post eligibility uses the outer quote post's languages, and an eligible quote displays its quoted context as-is. | A repost adds no text, while the outer quote contains the author's commentary and the embedded record is context. | Existing timeline model; decision Q8 | AC-013 |
| RULE-005 | Business rule | Must | Primary language and Content languages are independent: changing one shall not silently add, remove, or replace values in the other. | Avoids surprising preference mutation and matches the three-setting model. | Screenshot; Bluesky research | AC-004, AC-005 |
| RULE-006 | Business rule | Must | Changing Primary language shall affect composers opened afterward but shall not silently replace an explicit selection in an already-open composer. | Protects in-progress author intent. | Prompt; discovery | AC-006 |
| RULE-007 | Business rule | Must | Direct post views, complete thread context, notifications, and Saved Posts shall remain visible regardless of Content languages. | These surfaces represent deliberate access or interaction context rather than discovery. | Grilling decisions Q4 | AC-025 |
| RULE-008 | Business rule | Must | The authenticated viewer's own posts shall remain visible on every surface regardless of Content languages. | Authors must be able to find, verify, and manage what they published. | Grilling decision Q4 | AC-024, AC-025 |
| RULE-009 | Business rule | Must | A one-off composer language selection shall not change the next composer's initial selection, and a reply shall not inherit the parent post's languages. | Keeps Primary as the single predictable posting default. | Grilling decision Q9 | AC-026 |
| RULE-010 | Business rule | Must | V1 Content-language matching shall use exact equality between selectable base-language tags; a valid external region/script tag is preserved but does not implicitly match its base language. | Avoids introducing unapproved BCP-47 language-range semantics while keeping external records lossless. | Grilling decision Q10 | AC-028 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, BR-002, BR-003 | Given a fresh CraftSky installation, when a user opens Languages settings and then creates and browses posts, then the app exposes distinct language settings, creates language-tagged posts, and applies the configured Content-language behavior consistently across in-scope discovery surfaces. |
| AC-002 | BR-003, FR-001 | Given a signed-in user on Settings, when they inspect the available destinations and open Languages, then a dedicated Languages page opens with App language, Primary language, and Content languages sections. |
| AC-003 | BR-003, BR-004, FR-002 | Given the current English-only build, when the Languages page is shown, then App language displays English as the selected and only available option and visibly communicates that more App languages are coming. |
| AC-004 | BR-003, FR-003, RULE-005 | Given English is Primary, when the user changes Primary to Spanish and reopens the page, then Spanish remains Primary, Content languages are unchanged, and the setting's description identifies it as the default language for new posts rather than a translation setting. |
| AC-005 | BR-003, FR-004, FR-012, RULE-005 | Given the Content-language selector, when the user adds and removes languages, then it accepts any number of unique supported languages, prevents duplicates, remains independent of Primary, and explains that selecting none shows all languages. |
| AC-006 | BR-001, FR-003, FR-005, RULE-006 | Given Spanish is Primary, when the user opens any general, project, reply, or quote composer, then Spanish is initially selected; if Primary changes while that composer remains open, its existing explicit language selection is not replaced. |
| AC-007 | BR-001, FR-005, FR-012, RULE-001 | Given an open composer, when the author edits post languages, then they can select one, two, or three distinct languages, cannot select a fourth, cannot submit with none, and can identify the current selection without relying on colour alone. |
| AC-008 | FR-006, NFR-001, RULE-001 | Given `POST /v1/posts`, when `langs` contains one to three distinct valid supported BCP-47 tags, the request passes language validation; when it contains an invalid tag, a duplicate, or more than three entries, the AppView returns the standard `400` validation error envelope with a `langs` field error. |
| AC-009 | BR-001, FR-006, FR-007, FR-008 | Given an author submits a post with English and French selected, when the AppView writes and later indexes the record, then the PDS record contains `langs: ["en", "fr"]` and the indexed row contains the same two values; the create response does not need to wait for firehose indexing. |
| AC-010 | BR-002, FR-008, FR-009, RULE-002 | Given the authenticated viewer's stored Content languages are English and Welsh, when the home timeline contains otherwise-eligible posts tagged English, Welsh, English plus French, French only, and untagged, then the response includes the English, Welsh, and English-plus-French posts and excludes the French-only and untagged posts. |
| AC-011 | BR-002, FR-004, FR-008, FR-009, RULE-003 | Given the authenticated viewer has an empty stored Content-language list, when any in-scope browse/discovery surface contains tagged and untagged otherwise-eligible posts, then all of them are returned using that surface's normal ordering. |
| AC-012 | BR-002, FR-009, NFR-002 | Given a paginated in-scope surface where non-matching posts occur between matching posts, when the user requests results, then filtering occurs before the page limit, each page contains up to the requested number of eligible items, matching posts are not skipped or duplicated across cursors, and the surface's existing order is preserved. |
| AC-013 | RULE-004 | Given a French post is reposted and an English quote post quotes French content, when Content languages contain only English, then the repost item is excluded from discovery and the English outer quote post is eligible with the quoted French post displayed as context. |
| AC-014 | FR-011 | Given affected surfaces have cached English results and active cursors, when the AppView successfully stores French as the account's Content language, then incompatible cached items are no longer displayed, old cursors are not reused, and subsequent results use the French preference; if the update fails, the English preference and cached surfaces remain effective. |
| AC-015 | FR-010 | Given tagged and untagged indexed posts plus a newly created synthetic post, when each is returned through a post-shaped endpoint, then tagged posts expose their exact `langs` values and untagged posts expose `langs: []` consistently in Go and Flutter models. |
| AC-016 | FR-003, FR-004, FR-013, NFR-003, NFR-005 | Given an account completely replaces Primary and Content languages through `PUT /v1/languages/preferences`, when the same account signs in on another device and reads `GET /v1/languages/preferences`, then the AppView values are loaded there; when App language differs between the devices, each device retains its own App language. |
| AC-017 | FR-012, NFR-004 | Given keyboard navigation, a screen reader, or enlarged text, when the user operates the Languages page and composer selector, then controls remain reachable, labels and selection state are announced, and content does not become unusably clipped. |
| AC-018 | NFR-003 | Given preference or post-list operations, when normal request, validation, and database logging occurs, then logs may record that language filtering was active, the surface, and the selection count but do not record the complete Primary or Content-language values. |
| AC-019 | FR-013, FR-014 | Given `GET /v1/languages/preferences` returns `404 language_preferences_not_found` for a new account on a device whose ordered locales are `fr-CA`, `en-GB`, `fr-FR`, and an unsupported locale, when first preferences are proposed for initialisation, then the supported deduplicated base list is `["fr", "en"]`, Primary is `fr`, and Content languages are `["fr", "en"]`; given no supported locale, Primary is `en` and Content languages are `["en"]`. |
| AC-020 | FR-013, FR-015 | Given an account already has stored preferences, when any device calls `POST /v1/languages/preferences/initialize`, then the stored values are returned unchanged; given two devices initialise an uninitialised account concurrently, then exactly one preference set is created and both receive the same authoritative result. |
| AC-021 | NFR-005 | Given two signed-in accounts with different Primary and Content languages, when the active account changes while a preference or filtered-list request is in flight, then the old account's result is discarded and neither preferences nor filtered items appear under the newly active account. |
| AC-022 | FR-013, FR-016, NFR-001, NFR-005 | Given a preference initialisation or replacement with an invalid Primary tag, unsupported Content tag, duplicate Content value, unknown JSON field such as `accountDid`, or any query parameter, when the AppView validates it, then the standard `400` error envelope identifies an invalid request or preference field, no other account is selected, and none of the authenticated account's stored values change. |
| AC-023 | FR-017, NFR-005 | Given a newly activated account whose authoritative preferences are loading or initialising, when the user reaches the composer or an in-scope filtered surface, then posting and filtered results remain unavailable with an explicit loading or retry state until the correct account preferences are available. |
| AC-024 | BR-002, FR-009, FR-011, NFR-002, RULE-002, RULE-008 | Given English is selected and otherwise-eligible English, French, untagged, and viewer-authored French posts exist, when the viewer opens the home timeline, Projects browse, post/project/hashtag search, or another user's profile post/project/comment list, then English and viewer-authored posts are included, other French and untagged posts are excluded, filtering occurs before pagination, and each surface retains its existing order. |
| AC-025 | RULE-007, RULE-008 | Given a language-mismatched post, when the viewer opens it directly, encounters it as thread or quoted context, receives any notification involving it, views it in Saved Posts, or is its author, then the post remains visible as-is without a language-hidden placeholder. |
| AC-026 | FR-003, RULE-009 | Given Primary is English, when the author publishes a French or multilingual post and later opens a new general, project, quote, or reply composer, then only English is initially selected; the reply composer does not inherit its parent's languages. |
| AC-027 | FR-012, NFR-001 | Given English is CraftSky's only App language, when the user opens a Primary, Content, or post-language selector, then the full Bluesky-compatible v1 base-language catalogue is available and is not restricted to App localisations. |
| AC-028 | NFR-001, RULE-010 | Given an external post is tagged `fr-CA` and the viewer selected base language `fr`, when the post is indexed and read, then its exact `fr-CA` tag is preserved, it does not match `fr` on filtered browse/discovery surfaces, and it remains available through the defined visibility exceptions. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | A fresh installation signs into an account with no saved preferences. | App language is English; Primary and Content languages are atomically initialised from the ordered supported device locales, falling back to English. | FR-002, FR-014, FR-015 |
| EC-002 | The user clears the final Content language. | The UI explains the consequence and refreshed browse/discovery surfaces show all otherwise-eligible languages, including untagged posts. | FR-004, RULE-003 |
| EC-003 | A post has multiple languages and only one matches. | The post is eligible; matching is any-to-any, not all-to-all. | RULE-002 |
| EC-004 | A record has no `langs` field or an empty list. | It is treated as untagged: hidden on filtered browse/discovery surfaces, shown when Content languages is empty, and shown under an explicit exception. | FR-008, RULE-002, RULE-003, RULE-007, RULE-008 |
| EC-005 | A non-CraftSky client writes more than three or malformed languages. | Normal PDS Lexicon validation is authoritative for the public record; the CraftSky API separately rejects invalid input at its boundary. Invalid records must not crash the indexer or post-list queries. | FR-006, NFR-001, RULE-001 |
| EC-006 | The author attempts to remove every composer language. | The composer prevents submission and retains or restores Primary as the selectable default. | FR-005 |
| EC-007 | Primary changes while a composer is open, or a previous post used a one-off selection. | The open draft keeps its selected languages; every subsequently opened composer starts with only the current Primary. | RULE-006, RULE-009 |
| EC-008 | Content languages change while any filtered list request is in flight. | After the AppView update succeeds, results belonging to the old preference set are discarded and cannot overwrite refreshed surface state. | FR-011, NFR-005 |
| EC-009 | A matching post is muted, blocked, moderated, a reply, or an excluded Instagram import. | Existing eligibility rules still exclude it; language matching does not override moderation or surface policy. | FR-009 |
| EC-010 | A repost actor's language differs from the subject post. | Only the subject post's language controls eligibility. | RULE-004 |
| EC-011 | A quote's outer text and quoted post have different languages. | Only the outer quote post controls discovery eligibility; an eligible quote displays its quoted content as-is. | RULE-004 |
| EC-012 | Content languages contain a value not in the current UI catalogue due to stored data from a newer build. | The app preserves a valid BCP-47 value and presents a safe fallback label rather than dropping it silently. | NFR-001 |
| EC-013 | Device locales contain repeated region variants, unsupported values, or an empty list. | Initialisation preserves first-seen order, collapses supported variants to unique base tags, discards unsupported values, and falls back to English only when no supported value remains. | FR-014 |
| EC-014 | The author creates a post in a language not currently selected as a Content language. | The PDS write succeeds and the author continues to see the post on every surface; other viewers apply their own normal eligibility rules. | FR-007, RULE-008 |
| EC-015 | A returning account signs in on a device with different locales. | Existing AppView preferences load unchanged; device locales are not reapplied. | FR-015 |
| EC-016 | Two devices initialise the same new account concurrently. | The database creates one account preference row and both devices receive the same stored result. | FR-015 |
| EC-017 | The active account changes while a preference read, update, initialisation, or filtered-list request is in flight. | The old result is discarded and cannot populate the new account's state. | NFR-005 |
| EC-018 | The AppView preference read, update, or initialisation operation is unavailable. | The app presents a retryable error and does not replace authoritative values with device-derived or stale cached values; posting and filtered-surface behavior must not silently use another account's preferences. | FR-013, NFR-005 |
| EC-019 | A user opens a direct link to a mismatched or untagged post. | The complete post and thread context are shown as-is without a placeholder. | RULE-007 |
| EC-020 | A reply, mention, quote, like, repost, or follow notification is associated with content outside the selected languages. | The notification remains present, and opening its destination shows the relevant content as-is. | RULE-007 |
| EC-021 | A saved post no longer matches after Content languages change. | It remains present in Saved Posts and its folder, and opening it shows the post as-is. | RULE-007 |
| EC-022 | An external record contains a valid region or script BCP-47 tag that is not in the v1 base-language catalogue. | The AppView preserves and returns the exact tag; base-language exact matching does not treat it as a selectable base tag in v1. | NFR-001, RULE-010 |
| EC-023 | A preference caller supplies `accountDid`, `did`, or another unexpected selector in the JSON body or query string. | The strict preference route returns the standard `400` error envelope, does not select the supplied account, and does not mutate any preference row. | FR-013, FR-016, NFR-005 |

## 15. Data / Persistence Impact

- New public fields:
  - None. `social.craftsky.feed.post.langs` already exists and is optional.
- Changed API fields:
  - `POST /v1/posts` accepts optional `langs: string[]`.
  - Canonical post responses include `langs: string[]`.
- AppView materialisation:
  - Add a language-array column to `craftsky_posts`, populated from `FeedPost.Langs`.
  - Preserve absence/empty as the untagged state.
  - Add only the indexing needed to keep the affected filtered list queries efficient; exact index design belongs in the coding plan.
- Private AppView preferences:
  - Store exactly one row per account DID containing Primary language, Content languages, and normal preference timestamps.
  - Primary is one supported base-language tag.
  - Content languages is a unique tag array; `[]` means show all.
  - Initial creation must use an atomic uniqueness constraint on account DID.
- Device-local preferences:
  - App language: one stored tag, initially `en`.
  - An optional per-DID cache of Primary and Content languages may be retained for rendering continuity, but it is not authoritative and must not initialise or overwrite server state.
- Migration required:
  - Yes, for AppView post-language materialisation and the private account-language-preferences table.
  - No PDS record rewrite or production backfill.
- Backwards compatibility:
  - Existing public records without `langs` remain valid.
  - Existing API clients may omit `langs`; those posts remain untagged.
  - Existing post records still validate because no Lexicon change is required.
  - Canonical response consumers must tolerate the new camelCase `langs` field.
  - Existing accounts have no preference row and are initialised once from the first updated CraftSky client's device locales.

## 16. UI / API / CLI Impact

- UI:
  - Add Languages to Settings.
  - Add a Languages page based on the supplied Bluesky structure.
  - App language displays English and a visible future-language message.
  - Primary language is a single-select control.
  - Content languages is an unlimited multi-select control with an Add more languages action.
  - General, project, reply, and quote composers expose a shared one-to-three-language selector.
  - The app loads the active account's authoritative Primary and Content preferences and shows retryable loading/error states.
  - Successful Content-language changes invalidate and refresh affected home, Projects, search, and profile-list state.
  - Direct posts, thread context, notifications, Saved Posts, and the viewer's own posts remain visible without language placeholders.
- API:
  - Add `GET /v1/languages/preferences`.
    - It returns `{"primaryLanguage":"<tag>","contentLanguages":["<tag>"]}` for the authenticated DID.
    - It returns the standard `404 language_preferences_not_found` envelope when the authenticated account has no preference row, allowing Flutter to derive and propose first-use values.
  - Add `PUT /v1/languages/preferences`.
    - It requires the complete `primaryLanguage` and `contentLanguages` body and atomically replaces both values for the authenticated DID.
    - It returns `404 language_preferences_not_found` when no row exists; first creation goes through the initialisation operation.
  - Add `POST /v1/languages/preferences/initialize`.
    - It accepts the same complete preference body, creates it only when absent, and returns `200` with the authoritative stored row whether this request created it or lost/found a concurrent race.
  - Preference bodies use strict JSON decoding. Unknown fields, including `did` and `accountDid`, and any query parameters return the standard `400 invalid_request` envelope.
  - None of the preference routes accepts an account identifier in its path, query, or body. App language is absent from all request and response bodies.
  - Extend `POST /v1/posts` with `langs`.
  - Extend post-shaped responses with `langs`.
  - Apply authenticated Content-language filtering to:
    - `GET /v1/feed/timeline`
    - `GET /v1/projects`
    - `GET /v1/search/posts`
    - `GET /v1/search/projects`
    - `GET /v1/search/hashtags/{tag}/posts`
    - `GET /v1/profiles/{handleOrDid}/posts`
    - `GET /v1/profiles/{handleOrDid}/projects`
    - `GET /v1/profiles/{handleOrDid}/comments`
  - Do not language-filter direct post/thread endpoints, `GET /v1/notifications`, `GET /v1/saved-posts`, or posts authored by the authenticated viewer.
  - Affected endpoints read stored Content languages without accepting an account DID or Content-language override from the client.
  - Preserve the standard CraftSky error envelope and opaque cursors.
- CLI:
  - None.
- Background jobs:
  - Firehose indexing materialises languages on post create and update events.
  - No backfill job is required.

## 17. Security / Privacy / Permissions

- Authentication:
  - Preference read, update, and initialisation operations require an active CraftSky session.
  - The AppView derives the account DID from the authenticated session rather than accepting it in the request body, query, or path.
- Authorization:
  - A caller may access or change only the language preferences belonging to the authenticated DID.
  - Language tags do not otherwise change who may create a post or access CraftSky.
- Sensitive data:
  - Post languages are deliberately public metadata in PDS records.
  - Primary and Content languages are private account data in the AppView and participate in normal account-data deletion.
  - App language remains device-local and is not sent to the AppView for persistence.
  - Device locale values are normalised on-device; only the proposed supported Primary and Content values are sent for initialisation.
- Abuse cases:
  - Authors can mis-tag content. Language tags are self-declared metadata, not a moderation guarantee.
  - An author can add a popular language tag to gain distribution. Reporting or automated language verification is out of scope.
  - Invalid or excessively large language arrays must fail at request validation and must not reach the PDS write.
  - Language filtering must not bypass existing mute, block, moderation, membership, or import rules.

## 18. Observability

- Events:
  - Optional privacy-conscious client event for Languages settings opened.
  - Optional client event for post-language selector used, recording only selection count.
- Logs:
  - Validation failures may identify `langs` as the invalid field without logging the complete submitted list.
  - Preference and post-list logs may record whether filtering is active, the surface, and selection counts, not complete Primary, Content, or raw device-locale lists.
  - Indexer errors should identify the post URI/CID and validation category without logging post text.
- Metrics:
  - Count of created posts with zero, one, two, or three language tags.
  - Count of post-list requests with filtering active versus show-all, grouped by surface.
  - Count of preference initialisations that created a row versus returned an existing row.
  - Preference read/update/initialisation latency and failure rate.
  - Filtered post-list query latency and returned page size, grouped by surface.
- Alerts:
  - None specific for v1. Existing AppView error and latency alerting applies.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Authors mis-tag or omit a language. | Readers can receive content they cannot read or miss content they could read. | Require at least one tag in CraftSky composers, make selection visible, and retain manual correction before posting. |
| RISK-002 | Active filters hide untagged legacy or third-party posts. | Some otherwise-eligible posts disappear. | Make show-all available via an empty Content-language selection and state the behavior clearly. |
| RISK-003 | Filtering is applied after pagination on any affected surface. | Short pages, skipped items, duplicates, or leaked mismatched content. | Enforce filtering in each AppView candidate query before limit/cursor and add database-backed pagination tests per surface. |
| RISK-004 | Affected Flutter caches survive a preference change. | Old-language content remains visible until restart or refresh. | Treat Content languages as provider input, invalidate all affected surfaces, clear incompatible state, and reset cursors. |
| RISK-005 | Concurrent first use on different devices races. | One device could overwrite the other or receive preferences that differ from the stored row. | Use a unique account-DID row and one atomic create-if-absent operation that always returns the authoritative result. |
| RISK-006 | A very large selectable language catalogue is hard to use or localise. | Users struggle to find the desired language. | Use searchable, deduplicated language names with stable BCP-47 values and accessible fallback labels. |
| RISK-007 | Device locale normalisation chooses an unexpected initial preference. | A new account starts with languages the user does not want. | Preserve device order, show the result in Settings, allow immediate changes, and never reapply device defaults after initialisation. |
| RISK-008 | Filtering changes query performance. | Browse, search, or profile-list latency increases as the indexed post volume grows. | Materialise languages, design appropriate indexes in the coding plan, and test each affected query plan with representative data. |
| RISK-009 | Primary language terminology differs from current Bluesky. | Users familiar with Bluesky may expect a translation target. | Use copy that explicitly says it is the default language for new posts; translation is not present in CraftSky v1. |
| RISK-010 | A stale local cache is treated as authoritative. | The wrong composer default or post-list visibility can appear after another device updates preferences. | Scope caches by DID, reconcile with every authoritative read, and never use cache data to overwrite the AppView. |
| RISK-011 | Preference service failure blocks otherwise-readable surfaces. | Users cannot confidently compose or know which visibility rule is active. | Provide explicit retry states and do not silently substitute device defaults or another account's cache. |
| RISK-012 | Private language preferences are overexposed in telemetry or support logs. | A user's language profile is retained beyond the functional need. | Log only operation type, active-filter flag, and counts; include preferences in account deletion. |
| RISK-013 | One post-list endpoint omits the shared eligibility rule. | Language-mismatched content appears inconsistently and undermines the user promise. | Inventory all post-returning routes, centralise eligibility where practical, and run a shared visibility corpus against every in-scope endpoint. |
| RISK-014 | An exception is applied too broadly or too narrowly. | Discovery leaks mismatched posts, or deliberate/contextual content unexpectedly disappears. | Test direct, thread, notification, saved, own-content, quote, and repost boundaries independently. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-010 | No Lexicon file changes are required because the current CraftSky `langs` field already matches the official Bluesky shape. | Any later Lexicon change requires the project Lexicon skill, an ADR, `just lexgen`, and regenerated-code review. |
| ASM-011 | The route inventory in section 16 covers every current post-returning browse/discovery endpoint. | A missed route would need to be added to scope before acceptance tests are approved. |

## 21. Open Questions

- [ ] Non-blocking: Should future support include BCP-47 region/script variants and language-range matching rather than base-language exact matching?
- [ ] Non-blocking: Should a future language detector suggest corrections when typed content does not match the selected post languages?

## 22. Review Status

Status: Approved

Risk level: High

Review recommended: Required

Reviewer: Codex

Date: 2026-07-29

Notes:

- The user confirmed private per-account AppView persistence, device-locale first-use initialisation, show-all for an empty selection, strict untagged handling, any-match multilingual eligibility, and the complete visibility boundary.
- Filtered surfaces are the home timeline, Projects browse, post/project/hashtag search, and other users' profile post/project/comment lists.
- Explicit exceptions are direct post and thread context, all notifications, Saved Posts, quoted context, and the viewer's own posts.
- The risk level is High because visibility is now cross-cutting across multiple independently paginated API queries and Flutter caches; requirements review is required before test design or implementation.
- No Lexicon change is proposed. If review introduces one, the repository's Lexicon-plus-ADR workflow must be followed.
- CraftSky's product vision remains intact: public post data is portable, private reading preferences stay in the AppView, App language stays local, and language filtering never changes a surface's ordering.
- Formal workflow review is recorded in `03-document-review.md`; DR-001 through DR-003 were resolved in the requirements and test specification on 2026-07-29.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`, `BR-003`
  - `FR-001` through `FR-011`, `FR-013` through `FR-017`
  - `NFR-001`, `NFR-002`, `NFR-003`, `NFR-005`
  - `RULE-001` through `RULE-010`
- Suggested test levels:
  - Flutter widget tests for Settings navigation, Languages page controls, accessibility, Primary-only composer defaults, the three-language limit, all post composer variants, and visibility exceptions.
  - Flutter provider/unit tests for ordered device-locale normalisation, English fallback, authoritative preference loading, local cache reconciliation, account switching, in-flight request ownership, and invalidation of every affected surface.
  - Flutter API/repository tests for `langs` mapping and account-language-preference read, update, and initialisation operations.
  - Go request tests for `langs` decoding, validation, error envelopes, PDS record assembly, and synthetic responses.
  - Go preference API and Postgres integration tests for per-DID isolation, atomic create-if-absent initialisation, concurrent devices, updates, and existing-value wins.
  - Go indexer unit and Postgres integration tests for tagged, untagged, updated, and malformed external records.
  - Go integration tests for stored preference lookup, any-match filtering, show-all, untagged behavior, viewer ownership, repost/quote semantics, existing moderation rules, account isolation, ordering, and cursor correctness on every in-scope list endpoint.
  - A shared endpoint visibility corpus covering the home timeline, Projects browse, post/project/hashtag search, and profile post/project/comment lists.
  - Explicit regression tests proving direct posts, complete thread context, notifications, Saved Posts, quoted context, and the viewer's own posts are never language-filtered.
  - Migration tests for the new materialised language column, private preference table, uniqueness constraints, account deletion, and rollback.
  - Regression tests proving each surface's ordering and page behavior are unchanged when Content languages are empty.
- TDD seam:
  - The first schema/persistence test is `IT-028`.
  - The first authoritative language-filter test is the database-backed timeline corpus in `IT-008`; a pure truth table may document semantics but cannot substitute for testing the executed SQL before limit/cursor.
- Verification boundary:
  - Flutter acceptance tests are composed client flows using fakes.
  - Go/Postgres tests verify API, persistence, indexing, and query behavior.
  - A compose-stack smoke test, if run, is reported separately as live cross-process evidence.
- Blocking open questions:
  - None. Formal document review is approved; implementation still requires explicit approval after the coding plan.
