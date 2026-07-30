# TDD Implementation Plan: Post And Content Languages

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`)
- Coding plan: `04-coding-plan.md`
- Risk: High
- Implementation approval: Granted by the user on 2026-07-29 through `implement-tdd`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and executed commands updated below.
- Do not edit `lexicon/`; stop and return to planning if a Lexicon change becomes necessary.
- Do not add client-controlled language-filter inputs or filter hydrated pages.
- Do not commit or push unless the user explicitly asks.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-028 | FR-008, FR-013, FR-015, NFR-003 | AC-009, AC-016, AC-020 | Fails: migration is absent |
| 2 | IT-002 | FR-003, FR-004, FR-013, NFR-003, NFR-005 | AC-004, AC-005, AC-016 | Fails: preference store/API is absent |
| 3 | IT-003 | FR-015 | AC-020 | Fails: atomic initialise is absent |
| 4 | IT-004 | FR-013, FR-016, NFR-001 | AC-022 | Fails: strict validation is absent |
| 5 | IT-023 | FR-013, FR-015 | AC-020 | Fails: existing-row initialise is absent |
| 6 | IT-026 | FR-013, FR-016, NFR-005 | AC-016, AC-019, AC-020, AC-022 | Fails: routes are absent |
| 7 | UT-001 | FR-006, NFR-001, RULE-001 | AC-008 | Fails: post-tag validation is absent |
| 8 | UT-002 | FR-014 | AC-019 | Fails: locale derivation is absent |
| 9 | UT-003 | RULE-002, RULE-003 | AC-010, AC-011, AC-024 | Fails: visibility corpus is absent |
| 10 | UT-004 | NFR-001, RULE-010 | AC-028 | Fails: exact-match proof is absent |
| 11 | UT-005 | RULE-004 | AC-013 | Fails: repost/quote proof is absent |
| 12 | IT-005 | FR-006, FR-007, RULE-001 | AC-008, AC-009 | Fails: create path omits languages |
| 13 | IT-006 | FR-008, NFR-001, RULE-010 | AC-009, AC-028 | Fails: indexer omits languages |
| 14 | IT-007 | FR-010 | AC-015 | Fails: responses omit languages |
| 15 | IT-008 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-010, AC-011, AC-012, AC-024 | Fails: timeline SQL has no language predicate |
| 16 | IT-009 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: Projects browse is unfiltered |
| 17 | IT-010 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: post search is unfiltered |
| 18 | IT-011 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: project search is unfiltered |
| 19 | IT-012 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: hashtag posts are unfiltered |
| 20 | IT-013 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: profile posts are unfiltered |
| 21 | IT-014 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: profile projects are unfiltered |
| 22 | IT-015 | FR-009, NFR-002, RULE-002, RULE-003, RULE-008 | AC-011, AC-012, AC-024 | Fails: profile comments are unfiltered |
| 23 | IT-016 | FR-009, NFR-002 | AC-012, AC-024 | Fails: cross-adapter pagination proof is absent |
| 24 | IT-017 | FR-009 | AC-024 | Fails: policy-composition proof is absent |
| 25 | IT-020 | RULE-007 | AC-025 | Fails: direct/thread exception proof is absent |
| 26 | IT-021 | RULE-007 | AC-025 | Fails: notification exception proof is absent |
| 27 | IT-022 | RULE-007 | AC-025 | Fails: Saved Posts exception proof is absent |
| 28 | IT-024 | NFR-003 | AC-016, AC-018 | Fails: deletion/redaction proof is absent |
| 29 | IT-025 | NFR-002 | AC-012, AC-024 | Fails: representative query-plan proof is absent |
| 30 | UT-006 | FR-003, FR-005, RULE-001, RULE-009 | AC-006, AC-007, AC-026 | Fails: composer selection model is absent |
| 31 | UT-007 | FR-003, FR-004, RULE-005 | AC-004, AC-005 | Fails: preference model is absent |
| 32 | UT-008 | FR-012, NFR-001 | AC-027 | Fails: Flutter catalogue is absent |
| 33 | UT-009 | FR-012, NFR-001 | AC-017, AC-027 | Fails: fallback labels are absent |
| 34 | UT-010 | FR-010 | AC-015 | Fails: model mapping omits languages |
| 35 | UT-011 | FR-013, FR-016, NFR-001 | AC-022 | Fails: strict preference decoding is absent |
| 36 | UT-012 | FR-011, FR-017, NFR-005 | AC-014, AC-021, AC-023 | Fails: stale-completion guards are absent |
| 37 | UT-013 | FR-011 | AC-014, AC-024 | Fails: invalidation inventory is absent |
| 38 | UT-014 | NFR-003 | AC-018 | Fails: redaction proof is absent |
| 39 | UT-015 | FR-006 | AC-008 | Fails: `langs` envelope proof is absent |
| 40 | UT-016 | FR-005, FR-007 | AC-006, AC-009 | Fails: composer payloads omit languages |
| 41 | UT-017 | FR-003, RULE-006, RULE-009 | AC-006, AC-026 | Fails: open-composer stability is absent |
| 42 | IT-001 | FR-001, FR-002 | AC-002, AC-003 | Fails: Languages route/page is absent |
| 43 | IT-018 | FR-003, FR-004, FR-011, RULE-005 | AC-004, AC-005, AC-014, AC-016 | Fails: authoritative replacement/invalidation is absent |
| 44 | IT-019 | FR-014, FR-017, NFR-005 | AC-019, AC-021, AC-023 | Fails: account bootstrap/gating is absent |
| 45 | IT-027 | FR-003, FR-005, FR-007, RULE-006, RULE-009 | AC-006, AC-009, AC-026 | Fails: composer integration is absent |
| 46–60 | AT-001–AT-015 | See `02-acceptance-tests.md` | AC-001–AC-028 | Fails until vertical slices compose |
| 61–73 | REG-001–REG-013 | See `02-acceptance-tests.md` | See `02-acceptance-tests.md` | Existing behavior must remain green |

## Implementation Steps

### Step 1: IT-028

- Write failing test: inspect the migrated schema, defaults, preference uniqueness, terminal deletion behavior, and rollback.
- Run command: `go test ./internal/db -run LanguagesMigration`
- Confirmed failure: `000033_post_languages.up.sql` was absent.
- Implement: Added reversible `000033` migration with `craftsky_posts.langs`, its GIN index, and the private one-row-per-DID preference table.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/db -run '^TestLanguagesMigration$' -count=1 -v`
- Refactor: Formatted the focused test and ran the complete `internal/db` package against Postgres.
- Notes: Focused and nearby Postgres-backed tests pass. The table intentionally has no public-profile foreign key; terminal identity deletion is implemented through the approved private-data lifecycle in a later loop.

### Steps 2–6: Private Preferences

- Executed IT-002, IT-003, IT-004, IT-023, and IT-026 individually.
- Added the store and exact authenticated HTTP contract only after each focused red test.
- Notes:
  - IT-002 red: the `languages.Store` and GET/PUT handler constructors were absent.
  - IT-002 green: per-DID get and complete replacement plus authenticated-DID GET/PUT response behavior pass against Postgres; another account's row remains unchanged and App language is absent from the wire response.
  - IT-003 proves concurrent first-account initialization creates one authoritative row.
  - IT-004 proves strict, atomic validation and rejection of selectors, unknown fields, malformed JSON, invalid catalogue values, variants, and duplicates.
  - IT-023 proves initialization never overwrites an existing row.
  - IT-026 proves the three authenticated `/v1/languages/preferences*` routes, their methods, and absence/validation envelopes.

### Steps 7–14: Language Domain And Post Propagation

- UT-001 and IT-005–IT-007 are complete. UT-003–UT-005 moved alongside IT-008 so their truth tables exercise the production SQL predicate rather than an isolated duplicate. UT-002 remains in the Flutter locale-bootstrap slice.
- Keep post values lossless; preference values use the pinned selectable base catalogue.
- Notes:
  - UT-001 proves zero-to-three exact BCP 47 post tags, preserves variants, and rejects malformed and duplicate values.
  - IT-005 proves valid `langs` reach the PDS record and immediate response unchanged, while invalid values cause no PDS write.
  - IT-006 proves create/update indexing persists exact values and rejects malformed external updates without corrupting the prior indexed row.
  - IT-007 proves canonical post reads expose a non-null `langs` array.

### Steps 15–24: Authoritative Browse/Discovery Filtering

- IT-008–IT-017 are complete.
- Language eligibility must be in executed SQL before cursor/order/limit.
- Notes:
  - IT-008 store red: no language-aware timeline method existed.
  - IT-008 store green: exact overlap, empty-preference show-all, own-post inclusion, untagged hiding, and filtering before pagination pass against Postgres.
  - IT-008 handler loading is green and fails closed when the authoritative private policy cannot be loaded.
  - IT-009–IT-012 apply the same predicate to Projects browse, post search, project search, and hashtag posts across chronological, popularity, and relevance query branches.
  - IT-013–IT-015 apply it to profile posts, projects, and comments while preserving viewer-owned records.
  - IT-016 runs one interleaved pagination corpus across all eight adapters and proves no eligible row is skipped when mismatched rows occur before a page boundary.
  - IT-017 proves the language predicate composes with existing moderation and relationship eligibility instead of replacing it.
  - Existing search and profile-list calls retain show-all wrappers for focused legacy tests; production routes pass the private preference store.

### Steps 25–29: Exceptions, Privacy, And Query Plans

- IT-020–IT-025 are complete.
- Direct/contextual paths must remain structurally independent of preference loading.
- Notes:
  - IT-024 store deletion is green and the language store is appended to the ordered, retry-safe terminal identity-deletion lifecycle.
  - IT-020 proves direct reads and complete thread context return exact mismatched `fr-CA` posts without loading Content preferences or rendering a placeholder.
  - IT-021 proves notifications involving exact mismatched `fr-CA` posts remain visible and open their destination unchanged.
  - IT-022 proves Saved Posts and folder scopes retain exact mismatched `fr-CA` posts.
  - IT-025 proves the representative active-filter query uses the `craftsky_posts.langs` GIN index.

### Steps 30–45: Flutter Domain, State, Settings, And Composers

- UT-006–UT-017 and IT-001, IT-018, IT-019, and IT-027 are complete.
- Server preferences remain authoritative and account-keyed; App language remains device-local.
- Notes:
  - UT-006 proves one-to-three ordered, unique post-language selection and prevents removal of the final tag.
  - UT-007 proves strict immutable preference parsing and replacement.
  - UT-008–UT-009 prove the pinned selectable catalogue and readable fallback labels.
  - UT-010 proves exact post-language mapping plus legacy omitted/null normalization to an empty list.
  - UT-011 proves strict private preference API decoding.
  - UT-015 proves the Flutter create envelope sends ordered language tags unchanged.
  - UT-016 proves both ordinary and Project composer submission paths preserve the selected tags through their public boundaries.
  - UT-012 proves account and generation leases reject stale preference loads and replacements.
  - UT-013 enumerates exactly the eight filtered cache families and excludes direct, notification, and Saved state.
  - UT-014 proves language preference API error classification never retains submitted language values.
  - UT-017 proves an open composer retains its one-off selection when Primary changes, while the next composer uses the new Primary.
  - IT-001 proves typed Settings navigation, the three distinct language sections, English-only App language, searchable Primary/Content catalogues, and empty-Content show-all copy.
  - IT-018 proves serialized authoritative replacement, Content-only invalidation, disabled selectors during saving, retained values on failure, and localized failure feedback.
  - IT-019 proves one-time ordered device-locale initialization, returning-account behavior, account isolation, stale-work rejection, and first-list gating on the authoritative policy.
  - IT-027 proves general, reply, quote, and Project composers initialize from Primary, remain locally stable, enforce one-to-three tags, and submit exact ordered tags.

### Steps 46–73: Acceptance And Regression

- AT-001–AT-015 and the language-related REG-001–REG-013 coverage are complete through the focused Go, Postgres, Flutter unit/widget/provider, and composed-flow suites.
- AT-014 adds deterministic selected-state semantics and a 2x-text narrow-layout check. Physical VoiceOver/TalkBack, keyboard, and maximum platform text scaling remain MAN-001 and MAN-002 release gates.
- AT-015 composes locale bootstrap, authoritative preference replacement, Content-cache invalidation, exact tagged publishing, and browse-policy readiness using fakes; it does not claim a live PDS/Tap round trip.
- MAN-003 remains a release gate for production-scale latency. IT-025 provides the automated representative index-plan proof.
- Broad verification:
  - `go test ./... -count=1` passed from `appview/` against the development Postgres.
  - The focused Flutter language/model/provider/composer suite passed all 143 tests.
  - `dart analyze` passed with no issues.
  - The complete `flutter test` run has one unrelated existing failure: `instagram_migration_page_test.dart` expects a `Notification settings` `InkWell`, while the unchanged production notice uses a `TextSpan` recognizer. The language-related provider regressions exposed by the first run were fixed and their focused suites pass.
  - `git diff --check` passed and `lexicon/` is unchanged.

## Execution Log

| Test ID | Red command and meaningful failure | Green command and result | Implementation / refactor notes |
|---|---|---|---|
| IT-028 | Missing migration file; focused test failed before schema setup. | Focused test passed against Postgres; `go test ./internal/db -count=1` also passed. | Added reversible schema, default empty post languages, GIN index, private DID primary key, and timestamps. |
| IT-002 | Store types and GET/PUT handlers were undefined. | `go test ./internal/languages ./internal/api -run 'Test(StoreGetAndReplaceAreIsolatedByDID|LanguagePreferencesHandlersReadAndReplaceAuthenticatedAccount)$' -count=1 -v` passed against Postgres. | Added the narrow store and authenticated-DID handlers; full validation remains assigned to IT-004. |
| IT-003 | `Store.Initialize` was undefined. | Concurrent initialization test passed repeatedly with one row and one authoritative result. | Added transactional insert-if-absent followed by an authoritative read. |
| IT-004 | Invalid values mutated state and malformed/selector HTTP inputs were not rejected correctly. | Focused store and handler validation tests passed. | Added pinned-catalogue preference validation, atomic replacement, strict JSON decoding, and selector rejection. |
| IT-023 | Initialize HTTP behavior did not exist. | Focused existing-row initialize test passed. | Existing preferences are returned unchanged. |
| IT-026 | Dependency wiring, policies, and routes were absent. | Focused route/policy/API tests passed. | Added authenticated GET, PUT, and POST initialize routes with camelCase envelopes. |
| UT-001 | Post request had no `langs`; permissive parsing accepted malformed wire values. | Focused request validation corpus passed. | Added zero-to-three exact tags, strict wire-shape validation, BCP 47 parsing, duplicate rejection, and lossless preservation. |
| IT-005 | Create payload and synthetic response omitted languages. | Focused create-handler tests passed. | PDS records and immediate responses now preserve valid `langs`; invalid language input makes no write. |
| IT-006 | Indexed rows dropped languages. | Focused index create/update/malformed-update tests passed. | Indexer persists exact arrays and rejects invalid updates atomically. |
| IT-007 | Canonical reads returned no languages. | Focused post-store read test passed. | Shared post selection/scanning and response mapping now include a non-null `langs` array. |
| IT-008 | `ListTimelineWithLanguages` was undefined. | Store-level timeline visibility and handler preference-loading tests passed. | Added the SQL predicate before cursor/order/limit in authored and repost branches and wired authoritative private Content preferences into the production route. |
| UT-003 | The reusable visibility truth table was absent. | `TestLanguageVisibilityCorpusTruthTable` passed. | Added the test-only matching/empty/untagged/ownership corpus used to describe the authoritative SQL cases. |
| UT-004 | Exact base-vs-variant matching was not independently proved. | Focused Postgres eligibility test passed. | Proved selected `fr` does not match preserved `fr-CA`. |
| UT-005 | Repost-subject and quote-outer eligibility lacked a focused proof. | Focused Postgres eligibility test passed. | Proved reposts use the subject, quotes use the outer record, and quoted context does not determine eligibility. |
| IT-009 | `SearchProjectsWithLanguages` was undefined. | Focused Projects browse pagination test passed against Postgres. | Added language eligibility to project candidates before cursor/order/limit; fixed the scored scanner to include `langs`. |
| IT-010 | `SearchPostsWithLanguages` was undefined. | Focused post-search pagination test passed against Postgres. | Added explicit viewer/Content inputs and predicates to relevance, chronological, and popularity branches. |
| IT-011 | Project-search coverage began green after the shared IT-009 implementation already covered the relevance branch. | Focused project-search pagination test passed against Postgres. | Confirmed relevance results remove ineligible rows without changing ordering. |
| IT-012 | Hashtag coverage began green after the shared IT-010 implementation established the common post-search query. | Focused hashtag pagination test passed against Postgres. | Confirmed exact hashtag selection composes with server-owned language eligibility. |
| IT-013 | `ListByAuthorWithLanguages` was undefined. | Focused profile-post test passed against Postgres. | Added pre-cursor SQL filtering and the viewer-owned exception. |
| IT-014 | Coverage began green after the three author-list variants were added in the IT-013 production loop. | Focused profile-project test passed against Postgres. | Confirmed project membership rules and ownership compose with language eligibility. |
| IT-015 | Coverage began green after the three author-list variants were added in the IT-013 production loop. | Focused profile-comment test passed against Postgres. | Confirmed reply membership rules and ownership compose with language eligibility. |
| IT-024 | `Store.HandleIdentityDeleted` was undefined. | Focused idempotent deletion test passed against Postgres. | Added private-row deletion and wired the store into terminal identity deletion. |
| UT-015 | Existing create-client tests failed to compile once `langs` became required. | Focused ordered create-envelope test passed. | Made language tags required at the Flutter API boundary and proved exact ordered response mapping. |
| UT-016 | Composer and repository tests failed to compile because callers omitted the new required payload. | Focused ordinary-create and Project submit-adapter tests passed. | Propagated required tags through repositories, mutation state, both composer submit paths, and test fakes without optional fallbacks. |
| IT-016 | Cross-adapter interleaved pagination proof was absent. | Combined eight-surface pagination corpus passed against Postgres. | Proved language filtering precedes cursor/order/limit consistently. |
| IT-017 | Existing policy composition was not explicitly covered. | Focused moderation-composition test passed. | Proved matching language cannot restore an otherwise-hidden row. |
| IT-020–IT-022 | Exact mismatched exception records were not represented. | Focused direct/thread, notification, and Saved tests passed with `fr-CA`. | Kept exception stores and handlers structurally independent of preference loading. |
| IT-025 | No representative plan assertion existed. | Focused Postgres query-plan test passed. | Proved the active overlap predicate uses the GIN language index. |
| UT-012–UT-014 | Stale-generation, exact invalidation inventory, and redaction proofs were absent. | Focused provider and error-mapping tests passed. | Added generation leases, eight-family invalidation, and value-free error categories. |
| UT-017 | Open-composer default stability was not proved. | Focused selection and composer tests passed. | Composer state initializes once and remains local. |
| IT-001 | Languages route, page, and selectors were absent. | Focused page and typed-route tests passed. | Added localized App, searchable Primary, and searchable multi-select Content settings. |
| IT-018 | Preferences had no authoritative client mutation lifecycle. | Focused replacement tests passed, including failed-save retention and feedback. | Serialized complete replacements; only successful Content changes invalidate filtered caches. |
| IT-019 | Account bootstrap and first-list readiness were absent. | Focused bootstrap, account-race, and timeline-gating tests passed. | Device locales propose values once; AppView results remain authoritative per DID. |
| IT-027 | Composers did not load or submit language selections. | Focused general/reply/quote and Project widget tests passed. | Shared one-to-three selector, Primary defaults, retry/loading gates, and exact submissions are green. |
| AT-014 | Deterministic semantics and enlarged layout coverage was absent. | `flutter test test/languages/language_accessibility_test.dart` passed. | Added explicit semantics container, selected chip state, limit state, and 2x-text narrow layout coverage. |
| AT-015 | No single client-composition proof existed. | `flutter test test/languages/post_languages_flow_test.dart` passed. | Composed bootstrap, replacement, invalidation, tagged create, and browse readiness with fakes. |
| Broad Go | N/A | `go test ./... -count=1` passed. | All AppView packages passed against development Postgres. |
| Broad Flutter | First run exposed canonical empty-language fixture omissions, three language-policy harness regressions, and one unrelated Instagram failure. | The focused language/model/provider/composer suite passed all 143 tests; `dart analyze` is clean. Full `flutter test` retains only the unchanged Instagram `InkWell`/`TextSpan` mismatch. | Canonical post round trips now include `langs: []`; active policy projections retain their authoritative value for list consumers; legacy provider harnesses explicitly use deterministic English preferences or policy. Unrelated Instagram behavior remains untouched. |

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] `05-implementation-plan.md` read back
- [ ] Review completed or explicitly skipped
- [x] Manual device checks recorded or left as explicit release gates
- [x] Representative query performance check recorded or left as an explicit release gate

## Implementation Review Correction Pass: 2026-07-30

Inputs:

- Implementation review: `06-implementation-review.md`
- Verdict: `Changes required`
- Findings in scope: IR-001 and IR-002

Correction order:

| Step | Finding / Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IR-001 / IT-025 | NFR-002 | AC-012, AC-024 | Fails against PostgreSQL because the assertion names a nonexistent index and the approved plan contract is incomplete |
| 2 | IR-002 / UT-008, UT-009 | FR-012, NFR-001 | AC-005, AC-007, AC-017, AC-027 | Fails once every selectable tag is required to have exact pinned friendly metadata |

Execution notes:

- IT-025 red:
  - Strengthened `TestIT025RepresentativeLanguagePlansUsePreferencePKAndGINIndexes` to seed 20,000 representative posts, exercise the production visibility predicate, and require both the private preference primary key and post-language GIN index.
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api -run '^TestIT025RepresentativeLanguagePlansUsePreferencePKAndGINIndexes$' -count=1 -v` failed because the assertion still named nonexistent `craftsky_posts_langs_gin_idx`; PostgreSQL's plan showed the real `craftsky_posts_langs_gin`.
- IT-025 green:
  - Corrected the assertion to `craftsky_posts_langs_gin`.
  - The focused PostgreSQL test passes and proves `account_language_preferences_pkey` is used for the private preference lookup, the production own-post-or-overlap predicate uses `craftsky_posts_langs_gin`, and the representative post query does not fall back to a sequential scan.
- UT-008 / UT-009 red:
  - Replaced loose membership checks with an exact 184-tag snapshot fingerprint and required every selectable tag to have a friendly English label.
  - `flutter test test/languages/language_catalogue_test.dart --reporter compact` failed at `aa`, which still rendered as the raw code.
- UT-008 / UT-009 green:
  - Added exact English labels for every selectable tag from the pinned official Bluesky language catalogue at commit `27e4f84f3fb7429855a72377c307710eff910c76`.
  - Flutter and Go now assert the same sorted 184-tag FNV-1a fingerprint, `5a751f77a5ee754c`; unknown external or future post tags retain their lossless raw fallback.
  - The focused Flutter catalogue suite and `go test ./internal/languages -run '^TestUT008V1BaseLanguageCatalogueMatchesPinnedSnapshot$' -count=1 -v` pass.
- Full-suite regression corrections:
  - The first complete Flutter run red-tested three legacy signed-account harnesses that did not provide the new required language dependency. The account-switcher, add-account redirect, and discard-confirmation harnesses now supply deterministic English policy/preferences; their focused suites pass.
  - The saved-post DTO round-trip expectation now includes canonical `langs: []`, preserving the Saved Posts exception while testing the current post wire shape.
- Broad verification:
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./... -count=1` passed from `appview/`.
  - The focused Flutter language/model/provider/composer command passed all 143 tests.
  - `dart analyze` passed with no issues.
  - The final complete `flutter test --reporter compact` run passed 1,138 tests and retained exactly one unrelated existing failure: `instagram_migration_page_test.dart` expects a `Notification settings` `InkWell`, while unchanged production uses a `TextSpan` recognizer.
  - `git diff --check` passed and `git diff --name-only -- lexicon` returned no paths.
