# Coding Plan: Post And Content Languages

## 1. Inputs

- Requirements: `01-requirements.md`
- Acceptance tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Document-review verdict: Approved
- Risk level: High
- Blocking issues: None
- Implementation approval: Required before source or test changes begin
- Lexicon impact: None planned. `social.craftsky.feed.post.langs` and generated `FeedPost.Langs` already have the required public shape.

## 2. Implementation Strategy

Implement the feature as four connected boundaries:

1. A versioned language domain shared by behavior, not storage: BCP-47 validation plus matching v1 base-language catalogues in Go and Flutter.
2. One private, per-DID AppView preference row, exposed through the fixed authenticated GET/PUT/initialize contract and removed by terminal account deletion.
3. Public post-language propagation through create, PDS write, firehose indexing, canonical responses, and Flutter models.
4. Authoritative SQL eligibility on exactly the eight approved browse/discovery endpoints, followed by account-keyed Flutter loading, cache invalidation, settings, and composer controls.

The implementation should proceed in this order:

1. Prove migration `000033` with IT-028 before adding stores or handlers.
2. Implement private preference validation, persistence, exact routes, concurrency, and terminal deletion.
3. Propagate `langs` through AppView post create, indexing, response hydration, and Flutter wire models.
4. Make IT-008 the first authoritative filtering slice by adding the executed Postgres predicate to the home timeline before its cursor and limit.
5. Reuse the same predicate and visibility corpus across the remaining seven endpoints, then prove pagination and existing safety policies.
6. Prove all deliberate/contextual exceptions independently.
7. Add the account-keyed Flutter preference repository/provider graph, locale initialisation, loading gate, and successful-update invalidation.
8. Add Languages settings and the shared composer selector, then require languages through every Flutter create-post seam.
9. Complete accessibility, redaction, query-plan, regression, generated-file, and full-suite gates.

Key implementation decisions:

- `appview/internal/languages.Store` owns private preferences and implements terminal identity deletion. API handlers never accept a DID; they obtain it only from authentication and call this store. (`FR-013`–`FR-016`, `NFR-003`, `NFR-005`; `IT-002`–`IT-004`, `IT-023`, `IT-024`, `IT-026`)
- `PUT /v1/languages/preferences` is full replacement only. `initialize` uses insert-on-conflict-do-nothing followed by a new authoritative read in the same transaction, so concurrent callers receive the row that actually won. (`FR-013`, `FR-015`; `IT-003`, `IT-023`)
- Filtered handlers load Content languages from the private store and pass that server-owned array to their query method. Client headers, bodies, and query parameters never become a language-policy input. A missing row on a filtered route fails closed as an internal invariant instead of silently substituting device defaults or show-all. (`FR-009`, `FR-017`, `NFR-005`; `REG-011`)
- One `languageVisibilityPredicate(alias, viewerParam, languagesParam)` emits only trusted static SQL fragments. Its contract is `(viewer owns post) OR (Content is empty) OR (post langs overlap Content)`. It is inserted before cursor predicates, ordering, and limit in every candidate query. (`FR-009`, `NFR-002`, `RULE-002`, `RULE-003`, `RULE-008`; `IT-008`–`IT-017`)
- Timeline repost rows apply the predicate to the subject `craftsky_posts` row. Quote rows apply it to the outer row; quote hydration remains unchanged so eligible outer quotes retain quoted context. (`RULE-004`, `RULE-007`; `AT-009`, `REG-008`)
- Direct post, thread, notification, quote-hydration, and Saved Posts queries do not receive the predicate or the preference reader. Their bypass is structural and protected by separate tests. (`RULE-007`; `IT-020`–`IT-022`)
- Flutter preferences are keyed by redacted `AccountKey` and use `accountDioProvider(account)`. An active-account projection selects the correct family; late work for a previous account cannot populate the new account. (`FR-017`, `NFR-005`; `UT-012`, `IT-019`)
- Flutter never optimistically changes the effective preference. A replacement keeps the last authoritative `AsyncData` until PUT succeeds, permits one replacement at a time, and returns a failure result without changing preference or list state. (`FR-011`, `RULE-005`; `IT-018`)
- All eight filtered Riverpod families await an active Content-policy readiness provider before their first request. Only a Content-language change invalidates them; a Primary-only update does not reset browse state. (`FR-011`, `FR-017`; `UT-013`, `IT-018`, `IT-019`)
- Every composer owns a local `PostLanguageSelection` initialised once from Primary. Changes to Primary or a previous composer never overwrite that local selection. The create repository and notifier require a non-empty language list so omissions become compile-time call-site work. (`FR-003`, `FR-005`, `FR-007`, `RULE-006`, `RULE-009`; `UT-006`, `UT-016`, `UT-017`, `IT-027`)
- Go post-language validation accepts zero to three distinct valid BCP-47 tags for backward compatibility, while Flutter composers require one to three. Preference validation is stricter: Primary and Content values must be members of the v1 selectable base catalogue, and Content may be empty. (`FR-005`, `FR-006`, `FR-016`, `NFR-001`; `UT-001`, `UT-011`)
- No local Primary/Content cache is introduced in v1. App language alone uses `SharedPreferences`; server preferences remain authoritative on every activation, which avoids a stale-cache fallback path. (`NFR-003`, `NFR-005`; `REG-010`)

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Database | `craftsky_posts` materialises tags but not languages; private preferences use DID-keyed Postgres rows. | Add `langs`, GIN index, and one `account_language_preferences` row per DID with timestamps. | FR-008, FR-013, FR-015, NFR-003 | IT-028 |
| Language domain | Generated Lexicon exposes `Langs`, but no reusable validator/catalogue exists. | Add BCP-47 validation and pinned v1 base-language catalogues in Go and Flutter. | FR-006, FR-012, FR-016, NFR-001, RULE-010 | UT-001, UT-004, UT-008, UT-009, UT-011 |
| Private preferences | Notification preferences are the closest private AppView precedent. | Add GET/full PUT/create-if-absent initialize with strict input and auth-derived DID. | FR-003, FR-004, FR-013–FR-016 | IT-002–IT-004, IT-023, IT-026 |
| Account deletion | Terminal Tap identity deletion composes idempotent private-data handlers. | Add language preference deletion to that terminal handler list. | NFR-003 | IT-024, IT-028 |
| Post write/index | Create and index paths omit existing Lexicon `langs`. | Validate, write, synthetically return, materialise, and update exact arrays. | FR-006–FR-008, RULE-001, RULE-010 | IT-005, IT-006, REG-009 |
| Post responses | `postSelectColumns`, scanners, and `PostResponse` expose tags but no languages. | Add non-null `langs` to canonical Go and Dart post models; legacy omission maps to `[]`. | FR-010 | UT-010, IT-007 |
| AppView filtering | Eight independent SQL list/search queries own ordering and pagination. | Apply one overlap/empty/owner predicate in each candidate query before cursor/limit. | FR-009, NFR-002, RULE-002–RULE-004, RULE-008 | IT-008–IT-017, IT-025 |
| Flutter preference state | Active features mostly use current-account providers; fixed-account Dio exists for race-safe work. | Add fixed-account repository families, active projection, device-locale initialisation, retry, and serialized replacement. | FR-003, FR-004, FR-011, FR-013–FR-017, NFR-005 | UT-002, UT-007, UT-012, UT-013, IT-018, IT-019 |
| Settings/navigation | Settings is a typed root-level route with list destinations. | Add Languages tile, typed child route, three-section page, local App language, and searchable selectors. | FR-001, FR-002, FR-012, NFR-004 | AT-001, AT-003, AT-014, IT-001 |
| Composers | General/reply/quote share one composer; projects use a separate submit adapter. | Add one shared one-to-three selector and require selected languages through both pipelines. | FR-003, FR-005, FR-007, RULE-006, RULE-009 | AT-004, UT-006, UT-016, UT-017, IT-027 |
| Filtered client caches | Each surface owns an opaque cursor provider family. | Gate first load on authoritative policy and invalidate only the eight affected families after successful Content replacement. | FR-011, FR-017 | AT-010, AT-011, UT-013, IT-018, IT-019 |
| Exceptions | Direct/thread, notification, and Saved Posts use separate APIs/providers. | Leave these paths independent and add explicit regression coverage. | RULE-007, RULE-008 | AT-012, IT-020–IT-022, REG-004–REG-008 |

## 4. Files And Modules

### AppView production files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000033_post_languages.up.sql` | Create | Add `craftsky_posts.langs TEXT[] NOT NULL DEFAULT '{}'`, `craftsky_posts_langs_gin`, and `account_language_preferences(account_did PRIMARY KEY, primary_language, content_languages, created_at, updated_at)`. | FR-008, FR-013, FR-015, NFR-003 | IT-028 |
| `appview/migrations/000033_post_languages.down.sql` | Create | Drop the preference table, GIN index, and materialised post column in reverse order. | FR-008, FR-013 | IT-028 |
| `appview/internal/languages/catalog.go` | Create | Define the pinned Bluesky-compatible v1 base tag set, supported-preference checks, and friendly catalogue metadata needed by Go tests. | FR-012, FR-016, NFR-001 | UT-004, UT-011 |
| `appview/internal/languages/validation.go` | Create | Validate BCP-47 post tags, max-three/distinct rules, and complete preference values without logging values. Use `golang.org/x/text/language` as a direct dependency and retain original valid tags. | FR-006, FR-016, NFR-001, RULE-001, RULE-010 | UT-001, UT-011, UT-014 |
| `appview/internal/languages/store.go` | Create | Implement get, full replace, concurrent-safe initialise, Content-policy read, and typed not-found/validation errors. | FR-013–FR-016, NFR-005 | IT-002–IT-004, IT-023 |
| `appview/internal/languages/account_data.go` | Create | Implement idempotent `HandleIdentityDeleted` for the terminal Tap identity-deletion lifecycle. | NFR-003 | IT-024 |
| `appview/internal/api/language_preferences.go` | Create | Decode strict complete bodies, reject all query parameters, map errors to the fixed envelopes/statuses, and encode camelCase responses. | FR-013, FR-016, NFR-003, NFR-005 | UT-011, IT-002, IT-004, IT-026 |
| `appview/internal/api/language_visibility.go` | Create | Define the narrow preference reader, fail-closed handler helper, and trusted shared SQL predicate builder. | FR-009, FR-017, NFR-002, NFR-005 | UT-003–UT-005, IT-008–IT-017 |
| `appview/internal/routes/policy.go` | Change | Register authenticated read/write/write policies for GET, PUT, and initialize. | FR-013 | IT-026 |
| `appview/internal/routes/routes.go` | Change | Wire the preference routes and pass the language store only to the eight filtered handler factories. | FR-009, FR-013, RULE-007 | IT-020–IT-022, IT-026 |
| `appview/internal/app/deps.go` | Change | Construct one language store, expose it on `Deps`, and append it to terminal identity deletion. | FR-013, NFR-003 | IT-024 |
| `appview/internal/api/post_request.go` | Change | Decode optional `langs` and return a standard `langs` field error for invalid, duplicate, or over-limit input. | FR-006, NFR-001, RULE-001 | UT-001, UT-015, IT-005 |
| `appview/internal/api/post.go` | Change | Add languages to the PDS Lexicon body and synthetic `PostRow`. | FR-006, FR-007, FR-010 | IT-005, IT-007 |
| `appview/internal/index/craftsky_post.go` | Change | Validate and materialise `rec.Langs` on create/update; normalise nil to empty; leave no partial row for malformed input. | FR-008, NFR-001, RULE-010 | IT-006, REG-009 |
| `appview/internal/api/post_store.go` | Change | Add `Langs` to `PostRow`, `postSelectColumns`, ordinary scanners, author-list signatures, and the three profile list queries. | FR-008–FR-010, RULE-008 | UT-010, IT-007, IT-013–IT-015 |
| `appview/internal/api/post_response.go` | Change | Expose `langs` on canonical post responses as a non-null array. | FR-010 | UT-010, IT-007 |
| `appview/internal/api/timeline.go` | Change | Load authoritative Content languages before invoking the timeline query. | FR-009, FR-017 | IT-008, IT-026 |
| `appview/internal/api/timeline_store.go` | Change | Accept Content languages and apply the shared predicate to authored posts and repost subjects before timeline cursor/limit. | FR-009, NFR-002, RULE-002–RULE-004, RULE-008 | IT-008, IT-016, IT-017 |
| `appview/internal/api/search.go` | Change | Load authoritative Content languages in Projects browse, post search, project search, and hashtag-post handlers only. | FR-009, FR-017 | IT-009–IT-012 |
| `appview/internal/api/search_store.go` | Change | Accept the server-loaded array, apply the predicate before ranking/cursor/limit, and add `langs` to scored scanners. | FR-009, FR-010, NFR-002, RULE-002, RULE-003, RULE-008 | IT-007, IT-009–IT-012, IT-016, IT-017 |
| `appview/go.mod`, `appview/go.sum` | Change | Promote `golang.org/x/text` from indirect to direct for BCP-47 parsing; no unrelated dependency updates. | NFR-001 | UT-001, UT-011 |

### Flutter production files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/languages/models/language_preferences.dart` | Create | Immutable Primary/Content response and replacement model with value equality and safe unknown-tag preservation. | FR-003, FR-004, FR-013, NFR-005 | UT-007, UT-009, IT-018 |
| `app/lib/languages/models/language_preferences.mapper.dart` | Create generated | Decode/encode/copy camelCase preference values. | FR-013, NFR-005 | IT-018, IT-019 |
| `app/lib/languages/models/post_language_selection.dart` | Create | Model ordered one-to-three local composer selections and explicit add/remove validation. | FR-005, RULE-001, RULE-006, RULE-009 | UT-006, UT-017 |
| `app/lib/languages/data/language_catalogue.dart` | Create | Pin the cited Bluesky base-language snapshot, friendly English labels, value lookup, search options, and fallback labels for valid unknown tags. | FR-012, NFR-001 | UT-008, UT-009 |
| `app/lib/languages/data/device_locale_languages.dart` | Create | Convert injected ordered `Locale` values to supported unique base tags and English fallback. | FR-014 | UT-002 |
| `app/lib/languages/data/language_preferences_api_client.dart` | Create | Implement exact GET, complete PUT, and initialize calls without DID, App language, or query parameters. | FR-013, FR-016, NFR-005 | IT-018, IT-019 |
| `app/lib/languages/data/language_preferences_repository.dart` | Create | Expose typed load, replace, and initialize methods; identify only the not-found bootstrap case. | FR-013–FR-016 | IT-018, IT-019 |
| `app/lib/languages/data/api_language_preferences_repository.dart` | Create | Delegate the repository contract to the fixed-account API client. | FR-013, NFR-005 | IT-018, IT-019 |
| `app/lib/languages/providers/language_preferences_repository_provider.dart` | Create | Build account-keyed API/repository instances from `accountDioProvider(account)`. | FR-013, NFR-005 | IT-019 |
| `app/lib/languages/providers/device_locale_provider.dart` | Create | Provide an injectable ordered locale source backed by `PlatformDispatcher.instance.locales`. | FR-014 | UT-002, IT-019 |
| `app/lib/languages/providers/app_language_provider.dart` | Create | Persist device-local App language under a namespaced SharedPreferences key, defaulting to `en`. | FR-002, NFR-003 | REG-010 |
| `app/lib/languages/providers/language_preferences_provider.dart` | Create | Own each account's authoritative load-or-initialize flow, active projection, retry, serialized full replacement, and stale-completion guards. | FR-003, FR-004, FR-011, FR-013–FR-017, NFR-005 | UT-012, IT-018, IT-019 |
| `app/lib/languages/providers/content_language_invalidation.dart` | Create | Enumerate exactly timeline, Projects browse, post/project/hashtag-post search, and profile post/project/comment provider families. | FR-011 | UT-013, IT-018 |
| `app/lib/languages/providers/*.g.dart` | Create generated | Riverpod generated outputs for the new providers. | FR-013, NFR-005 | IT-018, IT-019 |
| `app/lib/languages/widgets/post_language_selector.dart` | Create | Wrap the existing searchable multi-select with ordered values, max three, non-colour semantics, limit error, and no-submit-with-zero state. | FR-005, FR-012, NFR-004, RULE-001 | AT-004, AT-014, UT-006 |
| `app/lib/languages/pages/languages_page.dart` | Create | Render App, Primary, and Content sections with loading/retry, single in-flight replacement, show-all explanation, and safe catalogue fallback labels. | FR-001–FR-004, FR-012, FR-017, NFR-004 | AT-001, AT-003, AT-014, IT-001, IT-018 |
| `app/lib/app.dart` | Change | Watch the device-local App language and set `MaterialApp.router.locale`; current supported locales still contain English only. | FR-002, BR-004, NFR-003 | REG-010 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add the Languages destination and typed navigation. | FR-001 | AT-001, IT-001 |
| `app/lib/router/route_locations.dart` | Change | Add the `languages` child under `/profile/settings`. | FR-001 | IT-001 |
| `app/lib/router/router.dart` | Change | Add a typed, root-navigator `LanguagesRoute` returning to Settings on Back. | FR-001 | IT-001 |
| `app/lib/router/router.g.dart` | Change generated | Regenerate typed route output. | FR-001 | IT-001 |
| `app/lib/feed/models/post.dart` | Change | Add `langs`, default legacy omission/null to `[]`, and preserve exact external tags. | FR-010, NFR-001, RULE-010 | UT-009, UT-010, IT-007 |
| `app/lib/feed/models/post.mapper.dart` | Change generated | Generate mapping/copy support for `langs`. | FR-010 | UT-010, IT-007 |
| `app/lib/feed/data/post_api_client.dart` | Change | Require and serialize ordered `langs` for Flutter-created posts. | FR-007 | UT-016, IT-027 |
| `app/lib/feed/data/post_repository.dart` | Change | Add required `langs` to the create contract so every implementation/fake/caller is updated. | FR-007 | UT-016 |
| `app/lib/feed/data/api_post_repository.dart` | Change | Forward the required language list unchanged. | FR-007 | UT-016 |
| `app/lib/feed/providers/create_post_provider.dart` | Change | Require languages, pass them to the repository, and preserve them on optimistic/synthetic copies. | FR-007, FR-010, RULE-008 | UT-016, IT-027 |
| `app/lib/feed/widgets/post_composer_sheet.dart` | Change | Gate on authoritative preferences, initialise local selection once from Primary, render the shared selector, and submit languages for general/reply/quote posts. | FR-003, FR-005, FR-007, FR-017, RULE-006, RULE-009 | AT-004, AT-011, IT-027 |
| `app/lib/projects/composer/project_composer_submit_adapter.dart` | Change | Carry the selected ordered language list in project submit arguments. | FR-005, FR-007 | UT-016, IT-027 |
| `app/lib/projects/widgets/project_composer_sheet.dart` | Change | Gate on preferences, initialise local Primary once, render the shared selector, and submit it with the project. | FR-003, FR-005, FR-007, FR-017, RULE-006, RULE-009 | AT-004, AT-011, IT-027 |
| `app/lib/feed/providers/timeline_provider.dart` | Change | Await active Content-policy readiness before first/load-more requests. | FR-017 | AT-011, IT-019 |
| `app/lib/projects/providers/project_feed_provider.dart` | Change | Gate Projects browse on the active policy. | FR-017 | AT-011, IT-019 |
| `app/lib/search/providers/post_search_provider.dart` | Change | Gate post search on the active policy. | FR-017 | AT-011, IT-019 |
| `app/lib/search/providers/project_search_provider.dart` | Change | Gate project search on the active policy. | FR-017 | AT-011, IT-019 |
| `app/lib/search/providers/hashtag_search_provider.dart` | Change | Gate hashtag post results on the active policy; do not gate hashtag-name search. | FR-017 | AT-011, IT-019 |
| `app/lib/feed/providers/user_posts_provider.dart` | Change | Gate profile post lists on the active policy. | FR-017, RULE-008 | AT-011, IT-019 |
| `app/lib/projects/providers/user_projects_provider.dart` | Change | Gate profile project lists on the active policy. | FR-017, RULE-008 | AT-011, IT-019 |
| `app/lib/feed/providers/user_comments_provider.dart` | Change | Gate profile comment lists on the active policy. | FR-017, RULE-008 | AT-011, IT-019 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate language repositories/preferences plus every affected filtered family during account activation/removal. | NFR-005 | UT-012, IT-019 |
| `app/lib/shared/api/providers/error_mapping_interceptor.dart` | Change | Classify the three fixed preference endpoints without values or identifiers. | NFR-003 | UT-014 |
| `app/lib/l10n/app_en.arb` | Change | Add Settings, three sections, future-language, Primary default, show-all, selector, limit, loading/retry, save-failure, and accessibility copy. | FR-001–FR-005, FR-012, NFR-004 | AT-001, AT-003, AT-004, AT-014 |
| `app/lib/l10n/generated/app_localizations*.dart` | Change generated | Regenerate checked-in English localisations. | Same as ARB | Same as ARB |

### Test and support files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/db/languages_migration_test.go` | Create | First red test: up/down schema, empty post default, DID uniqueness, timestamps, GIN index, and terminal deletion support. | FR-008, FR-013, FR-015, NFR-003 | IT-028 |
| `appview/internal/languages/validation_test.go` | Create | BCP-47, max-three, distinct post tags, supported preference catalogue, empty Content, and redacted errors. | FR-006, FR-016, NFR-001, RULE-001 | UT-001, UT-011, UT-014 |
| `appview/internal/languages/store_test.go` | Create | Per-DID reads/replacements, absent errors, concurrent initialize, existing-row winner, and deletion. | FR-013–FR-016, NFR-003, NFR-005 | IT-002, IT-003, IT-023, IT-024 |
| `appview/internal/api/language_preferences_request_test.go` | Create | Strict complete JSON, trailing JSON, unknown fields, duplicate Content, invalid tags, and no partial request. | FR-016, NFR-001 | UT-011 |
| `appview/internal/api/language_preferences_test.go` | Create | Handler/store integration for GET/PUT/initialize, atomicity, isolation, and camelCase responses. | FR-013–FR-016, NFR-005 | IT-002, IT-004, IT-023 |
| `appview/internal/routes/language_preferences_test.go` | Create | Exact route/policy/auth matrix, absent GET/PUT 404, initialize 200, selector/query rejection, and no cross-account access. | FR-013, FR-016, NFR-005 | IT-026 |
| `appview/internal/api/post_request_test.go` | Change | Add post-language decoder, validation, envelope, and no-PDS-call cases. | FR-006, NFR-001, RULE-001 | UT-001, UT-015, IT-005 |
| `appview/internal/api/post_test.go` | Change | Assert PDS body, synthetic response, and updated fake method signatures. | FR-006, FR-007, FR-010 | IT-005, IT-007 |
| `appview/internal/index/craftsky_post_test.go` | Change | Cover create/update/absent/external/malformed index behavior. | FR-008, NFR-001, RULE-010 | IT-006 |
| `appview/internal/api/post_response_test.go` | Change | Assert exact/non-null `langs` across ordinary and protected-compatible response shapes. | FR-010 | UT-010, IT-007 |
| `appview/internal/api/language_visibility_test.go` | Create | Define the test-only truth table and one reusable SQL corpus adapter for all eight endpoints, ownership, reposts/quotes, and existing exclusions. | FR-009, NFR-002, RULE-002–RULE-004, RULE-008 | UT-003–UT-005, IT-008–IT-015, IT-017 |
| `appview/internal/api/language_visibility_pagination_test.go` | Create | Traverse all cursors with interleaved and tied rows on every adapter. | FR-009, NFR-002 | IT-016 |
| `appview/internal/api/language_visibility_exceptions_test.go` | Create | Prove direct/thread, notification, Saved Posts/folders, quote context, and own-content bypasses independently. | RULE-007, RULE-008 | IT-020–IT-022, REG-004–REG-008 |
| `appview/internal/api/language_query_plan_test.go` | Create | Assert preference PK lookup, GIN overlap path, SQL-side eligibility, and bounded representative plans. | NFR-002 | IT-025 |
| `appview/internal/observability/language_redaction_test.go` | Create | Capture distinctive values and prove only surface, active flag, and count can escape. | NFR-003 | UT-014, IT-024 |
| Existing route, timeline, search, author-list, and fake tests | Change | Supply stored policies/new method arguments while retaining every prior expected URI/order/cursor. | FR-009, NFR-002 | REG-002, REG-003, REG-012 |
| `app/test/languages/fakes/fake_language_preferences_repository.dart` | Create | Programmable fixed-account fake with delayed reads/writes/initialise, call capture, and safe diagnostics. | FR-013–FR-017, NFR-005 | UT-012, IT-018, IT-019 |
| `app/test/languages/device_locale_languages_test.dart` | Create | Deterministic ordered locale normalisation and English fallback. | FR-014 | UT-002 |
| `app/test/languages/language_catalogue_test.dart` | Create | Pinned snapshot uniqueness, base tags, English labels, searchability, and breadth beyond App locales. | FR-012, NFR-001 | UT-008 |
| `app/test/languages/language_label_test.dart` | Create | Known/external/future valid fallback labels without data loss. | NFR-001 | UT-009 |
| `app/test/languages/models/language_preferences_test.dart` | Create | Independent Primary/Content copying and value equality. | FR-003, FR-004, RULE-005 | UT-007 |
| `app/test/languages/post_language_selection_test.dart` | Create | Initial Primary, ordered add/remove, zero/four/duplicate rejection, and open-composer stability. | FR-003, FR-005, RULE-001, RULE-006, RULE-009 | UT-006, UT-017 |
| `app/test/languages/providers/language_preferences_provider_test.dart` | Create | Successful/failed full replacement, one in-flight mutation, Primary-only behavior, Content invalidation, and authoritative reconciliation. | FR-003, FR-004, FR-011 | IT-018 |
| `app/test/languages/providers/account_language_preferences_test.dart` | Create | Load-or-initialize, retry, two accounts, fixed clients, account switch, and stale future rejection. | FR-014, FR-017, NFR-005 | UT-012, IT-019 |
| `app/test/languages/providers/content_language_invalidation_test.dart` | Create | Enumerate the eight affected families and prove direct, notification, Saved Posts, and unrelated search state remain. | FR-011 | UT-013 |
| `app/test/languages/post_language_selector_test.dart` | Create | Search, selection semantics, one-to-three behavior, keyboard actions, limit error, and constrained text layout. | FR-005, FR-012, NFR-004 | AT-004, AT-014 |
| `app/test/settings/languages_page_test.dart` | Create | Three sections, English-only App language, future copy, independent selectors, show-all copy, retry, and save failure. | FR-001–FR-004, NFR-004 | AT-001, AT-003, IT-001, IT-018 |
| `app/test/router/settings_routes_test.dart` | Change | Assert canonical typed Languages route, root navigator, and Back to Settings. | FR-001 | IT-001 |
| `app/test/feed/models/post_test.dart` | Change | Decode/copy exact, empty, missing, and null `langs`. | FR-010 | UT-010 |
| `app/test/feed/data/post_api_client_test.dart` | Change | Assert create payload and canonical response mapping for languages. | FR-007, FR-010 | IT-007 |
| `app/test/feed/providers/create_post_provider_test.dart` | Change | Require and forward selected languages in every mutation test. | FR-007 | UT-016 |
| `app/test/feed/widgets/post_composer_languages_test.dart` | Create | General/reply/quote defaults, one-off selection, Primary change, submission, reopen, and loading gate. | FR-003, FR-005, FR-007, FR-017, RULE-006, RULE-009 | IT-027 |
| `app/test/projects/project_composer_payload_test.dart` | Change | Add language submit arguments. | FR-005, FR-007 | UT-016 |
| `app/test/projects/widgets/project_composer_languages_test.dart` | Create | Project default, shared selector, submit, reopen, and loading gate. | FR-003, FR-005, FR-007, FR-017, RULE-006, RULE-009 | IT-027 |
| Existing filtered provider tests | Change | Prove authoritative loading/error gates and reset cursors after Content change. | FR-011, FR-017 | AT-010, AT-011, IT-018, IT-019 |
| Existing notification and Saved Posts tests | Change | Add mismatched language fixtures without changing visibility/navigation. | RULE-007 | IT-021, IT-022, REG-005, REG-006 |
| `app/test/shared/errors/language_redaction_test.dart` | Create | Prove providers, errors, and endpoint classification do not expose complete values. | NFR-003 | UT-014 |

## 5. Services, Interfaces, And Data Flow

### 5.1 Language catalogue and validation

The cited Bluesky revision is pinned in both implementations. Updating either catalogue is a deliberate versioned change and must update parity fixtures in both test suites.

```text
Go
  languages.ValidatePostTags([]string) error
    - zero to three
    - no exact duplicates
    - every value parses as BCP-47
    - retain original valid spelling/value

  languages.ValidatePreferences(Preferences) error
    - Primary is exactly one supported v1 base tag
    - Content is any number of unique supported v1 base tags
    - empty Content is valid

Flutter
  LanguageCatalogue.v1
    - stable base tag -> English display name
    - ordered searchable options
    - fallbackLabel(validUnknownTag) never drops the value

  deriveInitialLanguages(deviceLocales)
    - read Locale.languageCode in device order
    - retain supported bases only
    - first-seen dedupe
    - first item is Primary; all items are Content
    - no retained item -> en / [en]
```

Post matching remains Postgres exact array overlap. No range matching, canonicalisation, translation, language detection, or variant selector is introduced.

### 5.2 Private preference contract

```text
GET /v1/languages/preferences
  body: none
  response 200: {"primaryLanguage":"en","contentLanguages":["en"]}
  absent: 404 language_preferences_not_found

PUT /v1/languages/preferences
  body: {"primaryLanguage":"fr","contentLanguages":["fr","en"]}
  effect: complete atomic replacement for authenticated DID
  response 200: authoritative stored row
  absent: 404 language_preferences_not_found

POST /v1/languages/preferences/initialize
  body: same complete shape
  effect: insert only if absent
  response 200: authoritative stored row, whether inserted or already present
```

All three reject any query parameter with `400 invalid_request`. PUT and initialize use one strict JSON decoder, reject trailing values and unknown fields such as `did` or `accountDid`, validate the complete value before beginning mutation, and never echo submitted arrays in errors.

The store flow is:

```text
Get(did)
  SELECT by account_did
  no row -> ErrPreferencesNotFound

Replace(did, completeValue)
  validate complete value
  UPDATE ... WHERE account_did = did
  zero rows -> ErrPreferencesNotFound
  RETURNING authoritative value

Initialize(did, proposal)
  validate complete proposal
  BEGIN
  INSERT ... ON CONFLICT (account_did) DO NOTHING
  SELECT authoritative row in a subsequent statement
  COMMIT
  return selected row

HandleIdentityDeleted(did)
  DELETE WHERE account_did = did
  zero rows is success
```

### 5.3 Post create, index, and response flow

```text
PostLanguageSelection (Flutter, 1..3)
  -> CreatePost.create(langs: required)
  -> PostRepository.create(langs: required)
  -> POST /v1/posts {"langs":[...]}
  -> ValidatePostCreate
  -> lexiconRecordBody includes exact langs
  -> PDS social.craftsky.feed.post record

PDS response
  -> syntheticPostRow.Langs
  -> PostResponse.langs
  -> Flutter Post.langs

Tap create/update
  -> generated FeedPost.Langs
  -> validate external record safely
  -> craftsky_posts.langs (nil becomes empty array)
  -> postSelectColumns / scored scanners
  -> PostResponse.langs
```

The create handler does not wait for Tap. Omitted or explicitly empty API arrays remain valid untagged compatibility cases; the updated Flutter composers cannot produce them.

### 5.4 Authoritative filtering

Each filtered handler performs:

```text
authenticated DID
  -> languageStore.Get(DID)
  -> Content languages
  -> existing list/search store method(viewerDID, contentLanguages, request)
  -> SQL candidate eligibility before cursor/order/limit
  -> existing hydration, engagement, quote, and response flow
```

The trusted SQL fragment is logically:

```sql
AND (
  p.did = $viewerDID
  OR cardinality($contentLanguages::text[]) = 0
  OR p.langs && $contentLanguages::text[]
)
```

Each caller supplies its actual post alias and parameter positions. The helper never accepts request text as an alias or parameter expression. For timeline repost candidates, `p` is the repost subject. For quotes, `p` is the outer post. Existing moderation, mute, block, membership, reply, import, filters, ranking, and stable tie-breakers remain separate predicates and continue to win.

The eight adapters are:

| Adapter | Existing store method | Planned signature input |
|---|---|---|
| Home timeline | `PostStore.ListTimeline` | viewer DID + Content array |
| Projects browse | `SearchStore.ListProjects` | existing request + Content array |
| Post search | `SearchStore.SearchPosts` | existing request + Content array |
| Project search | `SearchStore.SearchProjects` | existing request + Content array |
| Hashtag posts | `SearchStore.SearchHashtagPosts` | existing args + Content array |
| Profile posts | `PostStore.ListByAuthor` | viewer DID + author DID + Content array |
| Profile projects | `PostStore.ListProjectsByAuthor` | viewer DID + author DID + Content array |
| Profile comments | `PostStore.ListCommentsByAuthor` | viewer DID + author DID + Content array |

No post-hydration filter is added in Go or Flutter. An empty Content array is passed normally and makes the SQL condition show all. A non-empty array excludes an empty `p.langs` unless ownership applies.

### 5.5 Structural exception flow

The following paths remain preference-independent:

```text
GET /v1/posts/{did}/{rkey}
post thread/comment/reply readers
QuoteViewRows hydration
GET /v1/notifications and destination resolution
GET /v1/saved-posts and folder scopes
```

They continue to return canonical `langs`, but neither handler nor store receives Content languages. This prevents a later refactor from confusing “post carries language metadata” with “surface applies discovery eligibility.”

## 6. State, Providers, Controllers, Or DI

### 6.1 Account-keyed preference graph

```text
sessionRegistryProvider
  -> active AccountKey
  -> accountLanguagePreferencesProvider(AccountKey)
       -> accountLanguagePreferencesRepositoryProvider(AccountKey)
            -> accountDioProvider(AccountKey)
       -> GET preferences
            200 -> authoritative AsyncData
            404 -> deviceLocaleProvider
                   -> derive proposal
                   -> POST initialize
                   -> authoritative AsyncData
            other error -> retryable AsyncError

activeLanguagePreferencesProvider
  -> projects only current AccountKey family

activeContentLanguagePolicyProvider
  -> value-equal Content-only readiness projection
  -> watched by exactly eight filtered list families
```

The account family holds no App language and uses redacted keys/diagnostics. The locale proposal is computed only after a 404 and is never persisted locally as authority. An initialize retry may recompute the same proposal; AppView create-if-absent remains authoritative.

### 6.2 Replacement flow

```text
replacePrimary(newPrimary)
  current = authoritative value
  candidate = current.copy(primaryLanguage: newPrimary)
  PUT complete candidate
  success -> publish returned value
  failure -> retain current, return false

replaceContent(newContent)
  current = authoritative value
  candidate = current.copy(contentLanguages: newContent)
  PUT complete candidate
  success -> publish returned value
             invalidate eight filtered families
  failure -> retain current and every existing cursor/item, return false
```

Only one PUT may be in flight per account; the page disables both selectors during it. The notifier checks its account lease/generation before publishing or invalidating. A stale completion may finish its HTTP call but cannot mutate active state.

### 6.3 Filtered provider gate and invalidation

The following providers await `activeContentLanguagePolicyProvider.future` before any first-page or load-more request:

- `timelineProvider`
- `projectFeedProvider`
- `postSearchProvider`
- `projectSearchProvider`
- `hashtagSearchProvider` (post results)
- `userPostsProvider`
- `userProjectsProvider`
- `userCommentsProvider`

Existing page widgets already render their providers' loading/error states; update copy or retry wiring only where a surface currently lacks an explicit retry action. Direct post, post thread, notifications, Saved Posts, profile data, profile search, hashtag-name search, suggestions, and recent search providers do not watch this gate.

`invalidateContentLanguageSurfaces(ref)` invalidates the eight provider families after a successful Content replacement. Family invalidation discards all loaded items and opaque cursors, including non-visible family instances. It does not mutate cached direct posts, notification pages, Saved Posts, or open composer selections.

### 6.4 Composer state

Each composer stores a local immutable `PostLanguageSelection`:

```text
preferences loading/error
  -> editor submission unavailable
  -> explicit progress or Retry

first authoritative preferences value
  -> local selection = [Primary]
  -> mark selection initialized

user edits selection
  -> local ordered list changes
  -> provider preference changes do not overwrite it

submit
  -> require 1..3
  -> pass exact ordered values

close/reopen any composer
  -> new local state starts from current Primary only
```

General, reply, and quote modes use the same `PostComposerSheet` state. Projects own the same model in `ProjectComposerSheet` and pass it through `ProjectComposerSubmitArguments`.

### 6.5 Device-local App language

`appLanguageProvider` reads and writes `SharedPreferences` key `app_language`. The only valid v1 value is `en`; invalid stored values recover to `en`. `_ReadyApp` supplies `Locale('en')` to `MaterialApp.router`. The provider is not keyed by account and is not invalidated on account switching.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Settings route

- Add a Languages `ListTile` to Settings with a language icon.
- Add `/profile/settings/languages` as a typed child of `SettingsRoute`.
- Lift it to the root navigator like the other full-screen Settings children.
- Back returns to Settings without resetting the profile shell.

### 7.2 Languages page

Use a scrollable page with the supplied Bluesky screen as the hierarchy reference:

1. App language
   - Description explains this controls the interface.
   - English is selected and the only option.
   - Supporting copy visibly says more App languages are coming.
2. Primary language
   - Description says it is the default for new posts.
   - Reuse `CraftskySingleSelectInput` with the complete searchable language catalogue.
3. Content languages
   - Description says these languages appear in browse/discovery.
   - Reuse `CraftskySearchableMultiSelectInput` with no maximum.
   - Empty selection is allowed and persistent copy explains that none means all languages.

The server-owned sections show a page-level loading state on first activation, a retryable error when load/initialize fails, and disabled controls plus progress during a replacement. A failed save retains visible values and shows localized feedback. Selection state is conveyed by labels, check/radio semantics, and semantic values, not colour alone.

### 7.3 Shared post-language selector

- Place the selector adjacent to the post body area and before submission in both composer layouts.
- Reuse the searchable multi-select and set `maxSelected: 3`.
- Display the current languages with friendly labels and announced selected state.
- Reject a fourth value with localized inline feedback.
- If the author removes the last value, keep the zero state visible for correction but disable submit and announce the required error after an attempted submission.
- The full post/content catalogue is available even though App language has only English.

### 7.4 Filtered surface states

No language-hidden placeholder is added. While authoritative preferences are unavailable, the eight provider-backed surfaces use their normal loading or retry presentation and do not show stale content. Once preferences are ready, server results render normally. Direct/contextual surfaces never enter a language-specific gate.

### 7.5 Localization and accessibility

- All new visible copy and semantics live in `app_en.arb`.
- Search controls remain keyboard reachable; Escape closes, Enter selects, and Tab advances using the existing select-input behavior.
- At large text sizes, sections and selected chips wrap vertically rather than using a fixed page height.
- Unknown but valid stored tags render a safe non-empty tag-based label.

## 8. Error, Loading, Empty, And Edge States

| State | Required behavior | Requirement / Test |
|---|---|---|
| Preference GET absent | Flutter derives ordered device proposal and calls initialize; direct GET remains 404. | FR-014, FR-015; AT-002, IT-026 |
| Concurrent initialize | One row wins; both callers receive the same row; no overwrite. | FR-015; IT-003 |
| Returning account on different device | Existing row wins; device locales are not applied. | FR-015; IT-023 |
| Preference network/server failure | Composer and filtered surfaces show retry; no device/stale fallback becomes effective. | FR-017, NFR-005; AT-011, IT-019 |
| PUT failure | Primary/Content state and all list items/cursors remain at the prior authoritative policy. | FR-011; AT-010, IT-018 |
| Account switch in flight | Old family may complete privately but active projection and surface state reject it. | NFR-005; AC-021, UT-012 |
| Empty Content | SQL shows all otherwise-eligible tagged and untagged rows in existing order. | RULE-003; AT-007, REG-002 |
| Active Content + untagged | Hidden on filtered surfaces; visible through explicit exceptions/ownership. | RULE-002, RULE-007, RULE-008; AT-006, AT-012 |
| Multilingual post | Any exact overlap is eligible. | RULE-002; UT-003 |
| External `fr-CA` with selected `fr` | Preserve exact tag; do not match; direct/contextual paths still show it. | RULE-010; AT-013 |
| Repost actor matches but subject does not | Repost excluded because subject languages govern. | RULE-004; AT-009 |
| Outer quote matches, quoted post does not | Outer quote and quoted context both render. | RULE-004, RULE-007; AT-009, REG-008 |
| Viewer authors mismatched post | Remains eligible on every filtered surface that can contain it. | RULE-008; REG-007 |
| API post languages omitted/empty | Valid untagged compatibility record; response returns `[]`. | FR-006, FR-010; REG-009 |
| Malformed external record | Indexer returns a bounded validation error and commits no partial language update; process does not panic. | EC-005; IT-006 |
| Unknown preference JSON/query selector | `400 invalid_request`; no row read or mutation for supplied DID. | FR-013, FR-016; IT-004, IT-026 |
| Complete values in logs/errors | Prohibited; log only operation, surface, active flag, and count. | NFR-003; UT-014, IT-024 |

## 9. Test Implementation Plan

### Slice 1: Schema and private storage

Red:

- Add IT-028 in `languages_migration_test.go`.
- Add store tests for absent reads, per-DID isolation, full replacement, concurrent initialise, existing-row winner, timestamps, and idempotent deletion.

Green:

- Add migration `000033`.
- Add language catalogue/validation and `languages.Store`.
- Add terminal deletion implementation and dependency wiring.

Refactor:

- Keep SQL in the language package.
- Keep typed errors value-free.
- Run:
  - `go test ./internal/db ./internal/languages`

### Slice 2: Exact preference HTTP contract

Red:

- Add UT-011 and IT-002, IT-004, IT-023, IT-026.
- Cover strict fields, trailing JSON, every query parameter, authentication, absent GET/PUT, initialize 200, and cross-account attempts.

Green:

- Add handler, route policy, and route wiring.
- Map not-found and invalid errors exactly.

Refactor:

- Share one complete-body decoder between PUT and initialize.
- Keep GET bodyless and App language absent.
- Run:
  - `go test ./internal/api ./internal/routes`

### Slice 3: Post-language propagation

Red:

- Add UT-001, UT-010, UT-015 and IT-005–IT-007.
- Start with valid/invalid create body and no-PDS-call assertions, then index create/update/untagged/external cases.

Green:

- Add request field/validation, PDS body, synthetic row, migration column use, indexer materialisation, select/scanner fields, and response field.
- Add Flutter `Post.langs` mapping after Go response behavior is green.

Refactor:

- Use one Go validator for handler and indexer.
- Preserve exact valid tag values.
- Run:
  - `go test ./internal/api ./internal/index`
  - `flutter test test/feed/models/post_test.dart test/feed/data/post_api_client_test.dart`

### Slice 4: First authoritative filtering seam

Red:

- Add the test-only truth table UT-003.
- Add IT-008 as the first executed-SQL filter test: matching, multilingual, mismatched, untagged, owner, repost subject, quote outer/context, show-all, and cursor traversal.

Green:

- Add preference lookup to `ListTimelineHandler`.
- Add Content array to `TimelineReader`.
- Add the shared SQL predicate before timeline cursor/order/limit.
- Apply it to both authored and repost subject candidates.

Refactor:

- Keep the truth table test-only; production authority remains SQL.
- Run:
  - `go test ./internal/api -run 'LanguageVisibility|Timeline'`

### Slice 5: Remaining seven filtered adapters

Red:

- Extend the shared corpus for IT-009–IT-015.
- Add IT-016 tied-page cursor traversal and IT-017 existing-policy composition.

Green:

- Update Projects browse, post search, project search, hashtag posts, profile posts, profile projects, and profile comments.
- Pass viewer DID explicitly to author-list queries for ownership.
- Update scored scanners for `langs`.

Refactor:

- Keep one corpus builder and one expected eligibility function in test support.
- Preserve each adapter's existing sort keys and cursor encoder.
- Run:
  - `go test ./internal/api -run 'LanguageVisibility|LanguagePagination'`
  - existing focused timeline/search/post-store tests

### Slice 6: Exception and performance boundaries

Red:

- Add IT-020–IT-025 and REG-004–REG-008, REG-011, REG-012.
- Seed distinctive preference values for redaction.

Green:

- Confirm exception routes need no production predicate.
- Add terminal deletion and observability coverage if not already complete.
- Finalize the GIN index only after representative plans are observed.

Automated query-plan contract for IT-025:

- Seed 20,000 otherwise-eligible root posts with 5% exact-language matches and run `ANALYZE`.
- Preference lookup must use the `account_language_preferences` primary-key index.
- A non-empty overlap probe must use `craftsky_posts_langs_gin` and must not use `Seq Scan on craftsky_posts`.
- Every full adapter test must demonstrate the language predicate inside executed SQL before its `LIMIT`; no Go/Flutter filtering may compensate.

Manual MAN-003 threshold:

- Seed 100,000 posts with 10% matching, warm each query five times, and compare median `EXPLAIN (ANALYZE, BUFFERS)` execution.
- Active filtering must remain under 250 ms and no more than 2.0 times the same adapter's show-all median.
- If either threshold fails, inspect/revise indexes and query shape before implementation approval is considered complete; do not waive it as local-machine noise without recording the plan and buffer evidence.

Run:

- `go test ./internal/api ./internal/observability`

### Slice 7: Flutter preference domain and account state

Red:

- Add UT-002, UT-007–UT-009, UT-012, UT-013, IT-018, and IT-019.
- Use injected locale lists and delayed fixed-account repositories.

Green:

- Add catalogue, device derivation, models, API/repository, account family, active projection, replacement serialization, policy gate, and invalidation helper.
- Register language state in the account boundary.

Refactor:

- Ensure `AccountKey`, provider diagnostics, and errors are redacted.
- Ensure only the Content projection triggers list invalidation.
- Run:
  - `flutter test test/languages/models test/languages/providers test/languages/device_locale_languages_test.dart test/languages/language_catalogue_test.dart test/languages/language_label_test.dart`

### Slice 8: Languages settings

Red:

- Add AT-001, AT-003, AT-014, and IT-001.
- Cover route/back behavior, three concepts, future copy, independent edits, show-all copy, search, semantics, retry, and narrow/large-text layouts.

Green:

- Add route, page, App-language provider, Settings tile, ARB copy, and generated files.
- Wire `MaterialApp.router.locale`.

Refactor:

- Reuse existing select controls instead of creating a second overlay implementation.
- Run:
  - `flutter test test/settings/languages_page_test.dart test/router/settings_routes_test.dart test/languages/post_language_selector_test.dart`

### Slice 9: Composer plumbing

Red:

- Add UT-006, UT-016, UT-017, IT-027, and AT-004.
- Cover general, reply, quote, and project composers plus reopen behavior.

Green:

- Make `langs` required through API/repository/notifier seams.
- Add shared selector and local selection to both composer implementations.
- Add preference loading/retry gating.

Refactor:

- Keep selection logic out of widget-specific code.
- Update every fake/call site explicitly rather than adding a silent default.
- Run:
  - `flutter test test/languages/post_language_selection_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_languages_test.dart test/projects/project_composer_payload_test.dart test/projects/widgets/project_composer_languages_test.dart`

### Slice 10: Filtered provider gates and complete regressions

Red:

- Add AT-010, AT-011, AT-012, AT-015 and client-side REG-004–REG-006, REG-010, REG-013.

Green:

- Gate the eight filtered providers.
- Confirm successful Content invalidation resets pages/cursors.
- Confirm direct/thread, notification, and Saved Posts providers remain untouched.

Refactor:

- Remove duplicate surface lists; keep the invalidation helper as the single client inventory.
- Run focused provider, notification, Saved Posts, settings, composer, and post-model suites.

### Final verification gates

After all focused slices are green:

1. Run formatting/code generation required by changed Go, Riverpod, mapper, router, and localization sources.
2. Run `go test ./...` from `appview/`.
3. Run `dart analyze` from `app/`.
4. Run `flutter test` from `app/`.
5. Run `just test` from the repository root against compose Postgres.
6. Run `git diff --check`.
7. Confirm no Lexicon diff and no regenerated Lexicon package.
8. Complete MAN-001 and MAN-002 on physical iOS/Android before release.
9. Complete MAN-003 with recorded dataset, plans, buffers, medians, and thresholds.
10. Report Flutter fake-based acceptance, Go/Postgres integration, optional compose smoke, and manual-device evidence as separate layers.

## 10. Sequencing And Guardrails

1. Do not edit `lexicon/`. If implementation reveals a real schema mismatch, stop and invoke the project Lexicon-plus-ADR workflow before continuing.
2. Do not start with a pure in-memory eligibility helper. IT-028 is first overall and database-backed IT-008 is the first filtering implementation.
3. Do not add `Accept-Language`, `did`, `accountDid`, Content arrays, or preference versions to filtered client requests.
4. Do not filter hydrated pages in Go or Flutter.
5. Do not put the shared predicate into direct, thread, notification, Saved Posts, or quote-hydration queries.
6. Do not treat absence of a preference row as show-all on filtered routes; Flutter must initialize first and the AppView must fail closed if the invariant is broken.
7. Do not locally cache Primary/Content as authority or use cached values to initialize/overwrite AppView state.
8. Do not optimistically publish preference replacements. A failed PUT must preserve the prior policy and all loaded cursors.
9. Do not reset an open composer's explicit language selection when Primary changes.
10. Do not let a Primary-only update invalidate browse/discovery state.
11. Do not broaden hashtag filtering to hashtag-name search, suggestions, top hashtags, profile search, or recent searches.
12. Do not log Primary, Content, post-language arrays, or raw device locale lists.
13. Preserve `langs: []` as a non-null canonical response for untagged posts.
14. Preserve exact valid external tags; do not silently reduce `fr-CA` to `fr` in indexed records or responses.
15. Preserve existing ordering, tie-breakers, moderation, relationship, membership, reply, import, and cursor behavior.
16. Keep migrations reversible and scoped to `000033`; no public-record backfill or PDS rewrite.
17. Review dependency changes so `golang.org/x/text` only moves to direct use and no Flutter package is added unnecessarily.
18. Keep generated files checked in according to existing repository conventions.

## 11. Risks And Open Questions

### Resolved implementation risks

- Catalogue drift: pin the cited revision in both Go and Flutter and make both catalogue tests assert the same TD-001 anchors and total snapshot size.
- Concurrent first use: use the unique DID key plus insert-do-nothing and authoritative read.
- Query drift: one SQL fragment and one shared executed corpus cover all eight adapters.
- Cache leakage: one explicit invalidation inventory plus fixed-account repository families.
- Exception overreach: preference dependencies are absent from exception handlers/stores.
- Local stale authority: omit the optional Primary/Content cache in v1.

### Non-blocking future questions

- Region/script selection and language-range matching remain future work.
- Automatic language detection or mismatch suggestions remain future work.

Neither question blocks this exact base-language implementation.

## 12. Handoff To TDD Builder

- Requirements, tests, and document review are approved.
- This coding plan is ready for user review.
- Risk remains High.
- Implementation must not begin until the user explicitly approves moving to `implement-tdd`.
- On approval, start with `appview/internal/db/languages_migration_test.go` (IT-028), not production code.
- Stop and return to planning if a Lexicon change, a ninth filtered endpoint, a new public preference input, or a different visibility exception becomes necessary.
