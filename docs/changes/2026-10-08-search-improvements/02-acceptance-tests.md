# Acceptance Test Specification: First Search Improvements

## 1. Test Strategy

Design status: Revised following document review, addressing DR-001 through DR-004 at the user’s request. `03-document-review.md` records the pre-revision review; it has not been rewritten. Risk: Medium, carried forward from requirements. This specification describes future tests; none have been implemented or run in this stage.

Use Go table-driven unit tests for matching rules and bounds; real PostgreSQL integration tests for retrieval, ranking, cursors, visibility and projection convergence; authenticated handler acceptance tests for useful results; and Flutter widget/client regressions for the unchanged UI contract. The relevance benchmark checks result identities and order only. No performance measurement is included.

Existing foundations: `appview/internal/api/search_store_test.go` uses `testing`, `testdb.WithSchema`, `seedMember`, `seedPost`, `seedSearchProject`, `seedProjectDetails`, fixed timestamps and URI assertions. Handler tests use `httptest` and viewer context. `testdb.WithMigratedSchema` applies production migrations on isolated PostgreSQL 16 schemas; use it for lifecycle and migration evidence because focused schemas may substitute all-active lifecycle predicates. Indexer tests in `appview/internal/index/craftsky_post_test.go` use real PostgreSQL and Tap events. Flutter search tests use `flutter_test`, provider overrides, `FakeSearchRepository` and mocked HTTP responses.

Targets below are proposed extensions or new files in these suites, not files created by this stage. Prefix new Go test names with `TestSearchImprovements` and include this document's test IDs in subtest names. IDs are local to this change folder; older suites may contain the same numeric IDs.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001 | AT-001 | Acceptance | Planned yes |
| FR-001 | AC-002, AC-003 | AT-002, UT-001, UT-002, IT-001 | Acceptance / Unit / Integration | Planned yes |
| FR-002 | AC-004 | AT-003, UT-003, IT-002 | Acceptance / Unit / Integration | Planned yes |
| FR-003 | AC-005, AC-006 | AT-004, UT-004, IT-003 | Acceptance / Unit / Integration | Planned yes |
| FR-004 | AC-007, AC-008 | AT-003, IT-002, IT-004 | Acceptance / Integration | Planned yes |
| FR-005 | AC-009 | UT-005, IT-005 | Unit / Integration | Planned yes |
| FR-006 | AC-010 | AT-005, UT-006, IT-006 | Acceptance / Unit / Integration | Planned yes |
| NFR-001 | AC-011 | IT-007, IT-008 | Integration | Planned yes |
| NFR-002 | AC-012 | UT-007, IT-009 | Unit / Integration | Planned yes |
| RULE-001 | AC-014 | AT-006, REG-001, REG-002 | Acceptance / Regression | Planned yes |
| RULE-002 | AC-015 | AT-006, IT-010, REG-003 | Acceptance / Integration / Regression | Planned yes |
| RULE-003 | AC-016 | IT-008, IT-011, IT-012 | Integration | Planned yes |
| RULE-004 | AC-017 | AT-002, AT-007, IT-013, REG-004, REG-005 | Acceptance / Integration / Regression | Planned yes |

NFR-003 and AC-013 are retired and have no test obligations. All 13 active Must requirements and 16 active acceptance criteria have planned coverage.

## 3. Acceptance Scenarios

### AT-001: Realistic useful results in both tabs

Requirement IDs: BR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Useful submitted craft search
  Scenario: Retrieve useful records for every explicit benchmark case
    Given the reviewed TD-001 benchmark corpus and an eligible authenticated viewer
    When the viewer submits each independently defined TD-001 query and tab case
    Then each designated useful URI is among the first five results
    And no designated negative URI appears across the complete result traversal
    And existing exact-name baseline matches remain retrievable

  Scenario: Corrected result is displaced beyond five by reliable matches
    Given six eligible reliable matches and one designated useful corrected match
    When the viewer searches with a page limit of two and follows every cursor
    Then all six reliable matches precede the corrected match
    And the corrected URI is retrieved exactly once after those matches
```

Implement the scenario as table-driven cases using every TD-001 row, including both tabs and distractors. The corrected-match exception asserts order and eventual retrieval instead of top-five placement. Assert URIs, not an exposed score or elapsed time.

### AT-002: Natural wording remains topic search

Requirement IDs: FR-001, RULE-004
Acceptance Criteria: AC-002, AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Keyword search with English word forms
  Scenario Outline: Find a showcase through ordinary wording
    Given a declared-English showcase record saying "Finished knitting socks"
    And the record contains no instructions
    When the viewer searches for <query> in its result tab
    Then the showcase record is retrieved
    And instructional content is not required for eligibility

    Examples:
      | query             |
      | sock              |
      | knit              |
      | how to knit socks |
```

Repeat with an ordinary post and a project. The searchable filler-only input is covered by IT-001 rather than treated as a browse-all query.

### AT-003: Approved vocabulary and all concepts across authored fields

Requirement IDs: FR-002, FR-004
Acceptance Criteria: AC-004, AC-007, AC-008
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Craft concepts across published fields
  Scenario Outline: Retrieve the approved equivalent while requiring socks
    Given a record containing "<source> socks" in its own searchable fields
    And a negative record containing "<source> hat" with no socks concept
    When the viewer searches for "<query> socks" in <tab>
    Then the socks record is retrieved and the hat record is excluded

    Examples:
      | source          | query           | tab      |
      | WIP             | work in progress | Posts   |
      | work in progress | WIP            | Posts    |
      | jumper          | sweater         | Posts    |
      | sweater         | jumper          | Posts    |
      | stockinette     | stocking stitch | Posts    |
      | stocking stitch | stockinette     | Posts    |
      | WIP             | work in progress | Projects |
      | work in progress | WIP            | Projects |
      | jumper          | sweater         | Projects |
      | sweater         | jumper          | Projects |
      | stockinette     | stocking stitch | Projects |
      | stocking stitch | stockinette     | Projects |

  Scenario: A project satisfies concepts across fields
    Given a project with "blue" in its caption, "wool" in materials and "socks" in title
    And a project with only blue and socks
    When the viewer searches for "blue wool socks"
    Then the first project qualifies and the second does not

  Scenario: An ordinary post satisfies concepts across fields
    Given a post with "blue" in its caption, an authored wool hashtag and "socks" in image alt text
    When the viewer searches for "blue wool socks"
    Then the post qualifies without requiring a new author-entered field
```

Run each equivalent pair in both directions and both tabs. IT-002 checks non-equivalences and phrase alternatives explicitly.

### AT-004: A strong single typo follows every reliable match

Requirement IDs: FR-003
Acceptance Criteria: AC-005, AC-006
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Conservative typo matching
  Scenario Outline: Retrieve a correction with or without reliable records
    Given a published "crochet blanket" record and <reliable_count> reliable "crochett blanket" records
    And a record containing only "crochet"
    When the viewer searches for "crochett blanket"
    Then the crochet blanket record is returned after every reliable record
    And the crochet-only record is excluded

    Examples:
      | reliable_count |
      | 0              |
      | 2              |

  Scenario: Do not rescue unrelated or multiply misspelled queries
    Given the protected-code and typo fixtures in TD-003
    When the viewer submits an unrelated query or one requiring two word corrections
    Then only independently qualifying reliable records or an empty page are returned
    And no result qualifies by correcting two query words
```

Repeat in both tabs; include a pattern-name typo for Projects. Short-code literal retrieval and non-correction are asserted in UT-004 and IT-003.

### AT-005: Pagination covers both tiers once

Requirement IDs: FR-006
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Complete submitted-search pagination
  Scenario Outline: Traverse an unchanged result set
    Given TD-004 data and unchanged matching rules
    When the viewer searches in <tab> with page limits of one, two and three
    And follows cursors until exhausted
    Then the concatenated URIs equal the expected ordered eligible set
    And each URI occurs exactly once including records qualifying through both tiers
    And reliable records precede corrected records across page boundaries

    Examples:
      | tab      |
      | Posts    |
      | Projects |

  Scenario: Reject a cursor used for a different search
    Given a cursor from a submitted search
    When it is reused with another normalized query or result type
    Then the existing invalid-cursor validation response is returned
```

Exercise reliable-only, synonym-only, typo-only and mixed sets. No continuity assertion spans a data change or deployment changing matching rules.

### AT-006: Separate tabs and visibility precede candidate selection

Requirement IDs: RULE-001, RULE-002
Acceptance Criteria: AC-014, AC-015
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_improvements_acceptance_test.go` (proposed)

```gherkin
Feature: Eligible separate search results
  Scenario: Matching content remains in its proper tab
    Given ordinary posts, projects and replies sharing a searchable topic
    When the viewer searches the Posts and Projects tabs
    Then Posts returns only ordinary top-level posts
    And Projects returns only projects
    And replies appear in neither tab

  Scenario: Hidden matches cannot crowd out visible results
    Given more hidden reliable and corrected candidates than the page limit and candidate bound
    And eligible reliable and corrected records for the same query
    When the viewer searches and follows all pages
    Then no hidden record is returned
    And hidden candidates do not displace eligible records or prevent the visible correction
```

IT-010 defines the visibility matrix and uses actual migrated lifecycle predicates.

### AT-007: The entered typo query and existing client workflow remain intact

Requirement IDs: RULE-004
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/search_page_test.dart`, `app/test/search/data/search_api_client_test.dart`

```gherkin
Feature: Unchanged submitted-search interface
  Scenario: Display corrected results without rewriting the query
    Given the viewer has entered "crochett blanket"
    When the viewer submits search and opens Posts then Projects
    Then the entered query remains "crochett blanket"
    And the existing cards and loading, empty, error and load-more states remain available
    And no correction explanation, additional control or merged results tab appears
    And any existing explicit recent-search save retains the entered wording
```

Capture repository/client calls to distinguish the existing explicit save workflow from result fetching; do not require the UI to stop existing authorized saves. Backend no-write coverage is IT-013.

## 4. Unit Test Cases

The proposed engineering defaults below make bounded behavior testable; they do not prescribe SQL operators, a scoring formula or cursor encoding. Document review/coding plan may refine these defaults with updated boundary tests while retaining the approved product rules.

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001 | AC-002, AC-003 | English forms and filler handling | Declared English; sock/socks, knit/knitting; `how to knit socks`; filler-only `how to` | Documented forms match; how/to are filler in these examples; sock/knit remain meaningful; no surviving concepts yields empty eligibility, never match-all | `appview/internal/api/search_matching_test.go` (proposed); use SQL integration assertions if normalization lives in PostgreSQL |
| UT-002 | FR-001 | AC-003 | Language boundary and exact-name path | Metadata en, non-English, missing, mixed including en; exact `Flax Light`, `K2`, `DK`; case/punctuation | English forms only when metadata declares English, including mixed metadata with en; assert this on the reliable normalizer/path separately from typo matching; missing/non-English retain literal names and may independently qualify for a typo match; no content-language inference; matching policy does not alter visibility policy | Same proposed matching suite |
| UT-003 | FR-002 | AC-004 | Finite equivalent concepts | Three approved bidirectional groups; multiword terms; repeated terms; wool/yarn; US/UK dc/sc; unlisted pair | Only approved groups expand, multiword alternative satisfies one concept, every other concept stays required; no recursive or arbitrary expansion | Same proposed matching suite |
| UT-004 | FR-003 | AC-005, AC-006 | Conservative correction boundary | `crochett`/crochet; `flaxx`/Flax; two wrong words; DK/K2/KAL; unrelated input | Proposed default: at most one insertion, deletion, substitution or adjacent transposition for one query word of at least four characters; protected short codes unchanged; other concepts required; deterministic confident candidate or no correction | Same proposed matching suite |
| UT-005 | FR-005 | AC-009 | Ordering and dedup invariants | Reliable and corrected rank tuples, equal score/time, repeated tag identity, changed popularity | Reliable tier first; relevance then descending time/URI; repeated tag identity contributes once; likes/reposts add no boost. Do not assert literal above approved synonyms | `appview/internal/api/search_ranking_test.go`; SQL-only ranking belongs in IT-005 |
| UT-006 | FR-006 | AC-010 | Opaque cursor validation | Correct tuple/query/kind, malformed cursor, changed query/kind | Round-trip internal tier/ordering state; normalized query/kind mismatch and malformed cursor rejected through existing error; no required version field | `appview/internal/api/search_cursor_test.go` |
| UT-007 | NFR-002 | AC-012 | Finite nonrecursive expansion and correction | Maximum accepted input, repeated equivalents, many eligible candidates, adversarial punctuation | Vocabulary fixed at three initial groups, each containing two alternatives; one expansion pass with deduplication and no recursive expansion; at most eight deterministic correction candidates per eligible query word, at most one replaced word per result, no retry loop; candidate generation stays finite for request size | Proposed matching suite; test pure behavior or exposed matching plan, not a snapshot of implementation syntax |

For UT-007, with n meaningful query words the correction candidate combinations are at most 1 + 8n before finite vocabulary alternatives. Preserve the existing 256-character query validation and maximum 100-result page limit; verify the actual parser boundaries for both routes in IT-009. The eight-candidate cap and edit rule are proposed engineering choices, not additional product commitments. Candidates must be selected from eligible content so hidden words cannot consume this cap. The proposed deterministic policy is specified below. If the implementation uses a different confidence mechanism, revise the documented defaults and tests during coding-plan review; do not silently broaden matching.

### Proposed correction policy for coding-plan review

- Normalize candidate word identity consistently with literal matching, deduplicate it, and protect query words shorter than four characters, including DK, K2 and KAL.
- Accept only one insertion, deletion, substitution or adjacent transposition in one query word. This is the proposed confidence boundary; no lower-confidence retry or second corrected word is permitted. Reliable matches remain eligible independently.
- Derive candidate words only from records eligible for the viewer/tab and capable of satisfying every other meaningful query concept. Hidden records and words that cannot complete the query cannot consume the cap.
- Sort qualifying distinct candidate words by edit distance ascending, then normalized word ascending; take at most eight per query word. Stable lexical ties avoid dependence on database row order, popularity or hidden-record frequency.
- Expand the three approved vocabulary groups once, deduplicate equivalent concepts and do not recursively re-expand. Each corrected record must satisfy the complete query with at most one replaced word.
- Assign a record qualifying through both paths to the reliable tier once. Record order remains relevance, descending time, then URI within each tier. For unchanged data/rules, candidate selection and page traversal must reproduce the same eligible set.
- Add boundary cases for four versus three characters, one versus two edits, eight versus nine candidates, shuffled seed order and hidden candidates preceding visible words lexically. Check candidate identities and output, not execution duration.

These defaults resolve the test-design proposal, not the final SQL/index design. The coding plan must confirm them or document revised deterministic confidence and finite bounds with corresponding boundary tests. Product rules and performance exclusions remain unchanged.

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001 | AC-002, AC-003 | PostgreSQL language-aware matching | TD-001/TD-002, otherwise identical English/non-English/unknown records; viewer allows each fixture language | Search forms, exact names, filler-only and literal non-English tokens in both tabs | Only declared-English records gain reliable word-form expansion; assert the reliable path directly or use a word-form contrast outside the selected typo boundary (for the proposed one-edit rule, knitting/knit); unknown/non-English sock/socks may independently qualify through typo matching; literal names remain available; filler-only query returns empty; showcase qualifies without instructions | `appview/internal/api/search_store_test.go` |
| IT-002 | FR-002, FR-004 | AC-004, AC-007, AC-008 | Equivalent and cross-field concepts | TD-001 pairs; TD-002 project and ordinary post with split concepts, missing-concept contrasts and phrase contrasts | Search each pair both ways; `blue wool socks`; `work in progress socks`; negative equivalences | All concepts can match across scoped fields; phrases act as approved alternatives, not individually optional words; missing concept, wool/yarn, ambiguous abbreviations and arbitrary association negatives excluded | Same existing store suite |
| IT-003 | FR-003 | AC-005, AC-006 | Typos with negatives and protected codes | TD-003 reliable/corrected pairs and pattern-name cases; literal DK/K2/KAL records | Search one typo, two typos, excessive/unrelated misspellings and protected codes | Strong single correction eligible with/without reliable results; no missing-concept admission; no short-code correction; reliable literal short-code matches preserved; no two-word correction | Same existing store suite |
| IT-004 | FR-004 | AC-007, AC-008 | Every scoped field and optional metadata | One exclusive match per caption, hashtag, alt text, project title, pattern name, material, project tag, design tag, materialized authored tag; null/empty images/tags/alt | Query unique marker in each field for its tab | Each field works independently; absent optional fields do not error; no generated description or new tag field needed | Same existing store suite, using TD-002 |
| IT-005 | FR-005 | AC-009 | Database ranking invariants | TD-004 comparable records differing only in primary versus supporting field, tier, time, URI, duplicate tags or popularity | Query; then add duplicate same tag or likes/reposts and rerun | Every reliable match ahead of every corrected match; comparable caption/title/pattern matches ahead of tag/alt-only within each tier; ties descending time/URI; duplicate tags/popularity do not change ordering | Same existing store suite |
| IT-006 | FR-006 | AC-010 | Cursor transition, ties and mismatches | TD-004 with reliable-only, synonym, typo-only and mixed results; dual-qualified record | Traverse at limits 1/2/3; repeat traversal; reuse cursor with different query/tab through handlers | Exact expected ordered URI set once; dual-qualified URI in reliable tier only; final cursor exhausted; wrong query/tab gets existing invalidCursor envelope | Store suite and proposed acceptance suite |
| IT-007 | NFR-001 | AC-011 | Existing projection receives improvements | Populate pre-change production schema with authored records including langs/tags/images/project fields; retain original record/CID values | Apply required local upgrade/backfill in isolated test DB, then search; if no migration required, run new query against pre-existing rows | Old eligible rows obtain new matching without author edits; record JSON/CID unchanged; no PDS interaction. Validate real migration SQL, not only hand-maintained test DDL | `appview/internal/db/search_improvements_migration_test.go` (proposed) with API retrieval helper; exact setup finalized after schema choice |
| IT-008 | NFR-001, RULE-003 | AC-011, AC-016 | Local retrieval and no external/PDS side effects | Real local DB, injected outbound request recorder and PDS write spies where applicable; TD-005 | Execute reliable, synonym, typo, added-field and empty-result searches | No external model/search/PDS request or write; no published-record mutation; local PostgreSQL sufficient. Review dependency wiring alongside runtime assertions so an unobserved direct client cannot bypass the recorder | Proposed acceptance suite using existing capability interfaces |
| IT-009 | NFR-002 | AC-012 | Request/result and candidate bounds | Max-length valid query, one over boundary, limit 100/101, repeated equivalences, more candidates than cap | Parse both routes and execute accepted requests | Existing validation/defaults retained; accepted query plan obeys UT-007 finite caps and one-word correction; returned page never exceeds accepted limit; no elapsed-time assertion | `search_request_test.go` and store suite |
| IT-010 | RULE-002 | AC-015 | Visibility before all candidate limits | TD-005 matrix in migrated schema: mute, block both directions, existing moderation exclusion states, terminal owner, viewer language mismatch, own-post exception, no viewer-language preference; hidden candidates exceed limits/cap | Run reliable, synonym and typo queries in both tabs, with small pages and traversal | Apply each existing policy unchanged before candidate eligibility/selection; no hidden URI, no visible displacement or correction suppression; own-post/no-preference exceptions remain as existing policy specifies | Store suite plus migrated-schema acceptance suite |
| IT-011 | RULE-003 | AC-016 | Private and quote boundaries | Unique markers only in draft/schedule/private evidence; eligible original quote target, quoting post with unrelated own fields, quoting post with matching own fields | Search each marker and quote topic in both tabs | Private-only markers create no result; quote target can match independently; quote body never makes quoting post match; ordinary top-level quote matching own fields remains eligible | Proposed acceptance suite using TD-005 |
| IT-012 | RULE-003 | AC-016 | Protected values absent from serialized diagnostics | Unique query/correction/private-content canaries; captured slog, supported Sentry transport and outbound recorder; successful, empty, validation and DB-error paths | Perform searches, serialize actual emitted sink output, flush supported transport | No raw query/correction/private canary in messages, attributes, errors, spans or outbound requests; empty result remains normal; permitted operation/error context survives per logging guide. Check sink contents, not just pre-sanitized objects; no new metrics required | `appview/internal/api/search_improvements_privacy_test.go` (proposed), existing observer/test transport patterns |
| IT-013 | RULE-004 | AC-017 | Authenticated API and no history side effect | TD-005 viewer saves/likes/embed state; recent-search rows; handler/capability spies | Request both routes with valid/absent auth, malformed cursor and invalid bounds; fetch multiple pages | Existing auth status and error/message/requestId envelope, camelCase page shape, opaque cursor and hydrated viewer/embed state preserved; no additional history-write call or DB row change from fetching | Existing `search_capability_test.go`, `search_response_test.go`, store handler tests; extend for both tabs |
| IT-014 | NFR-001, RULE-002 | AC-011, AC-015 | Index update/delete convergence | Existing indexed record with old caption/tag/alt markers and project fields in migrated schema | Deliver valid update, replay same URI/CID event, search old/new markers, then deliver deletion and search again | New markers searchable, obsolete markers absent, replay introduces no duplicate projection/result, deleted record absent from both tiers; source record not rewritten by search | `appview/internal/index/search_improvements_projection_test.go` (proposed), existing Tap/indexer fixture conventions |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test / Automation Target |
|---|---|---|---|---|
| REG-001 | Disjoint result types and replies excluded | RULE-001 | AC-014 | Retain `TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs`; add own-field quote versus reply fixtures in `search_store_test.go` |
| REG-002 | Other search and feed surfaces retain their behavior | RULE-001 | AC-014 | Run existing exact-hashtag equality/order, project browse/filter, profile rank, suggestions/typeahead tests in `search_store_test.go` and Flutter search suites; retain chronological timeline ordering tests in `appview/internal/api/timeline_store_test.go`. Approved lexical equivalents/typos must not alter exact hashtag equality |
| REG-003 | Existing visibility exceptions and lifecycle | RULE-002 | AC-015 | Retain post/project language-before-pagination tests, moderation-before-ranking tests and `appview/internal/index/projection_lifecycle_integration_test.go`; extend new matching paths through IT-010 |
| REG-004 | Client request/response and unchanged search states | RULE-004 | AC-017 | Retain `app/test/search/data/search_api_client_test.dart`, error suite, provider contract/pagination tests and `search_page_test.dart`; capture original typo query in requests and explicit saved-history payloads; verify separate tab results and loading/empty/error/load-more |
| REG-005 | Recent searches remain explicitly saved, separate from fetching | RULE-004 | AC-017 | Retain `search_recent_store_test.go` and Flutter recent-search provider/model tests; pair with IT-013 asserting no new writes during backend fetch. Existing normalization/casing rules remain; original spelling is not replaced by corrected wording |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Reviewable relevance corpus | Query table below; stable synthetic public URIs, tab, authored fields, declared language, usefulness rationale and explicit excluded URIs; fixed viewer/time; use the six concrete family distractors below in each independent case, including missing-concept near-matches | AT-001, AT-002, AT-003, IT-001, IT-002 |
| TD-002 | Fields and language boundary | One record per scoped field using an exclusive marker; null/empty optional fields; English/non-English/unknown/mixed langs; author/viewer combinations; split-concept post/project; exact Flax Light pattern and non-English literal `chaussettes` | UT-001, UT-002, IT-001, IT-002, IT-004 |
| TD-003 | Correction and precision | `crochett blanket` reliable and `crochet blanket` corrected; Flaxx Light/Flax Light pattern; `crochett blankett` requiring two changes; `zzzzq blanket`; DK, K2, KAL literal and near-code negatives; crochet-only missing blanket | AT-004, UT-004, IT-003 |
| TD-004 | Rank and pagination | Six reliable and three corrected matches; identical content across different times and equal-time different URIs; comparable primary/support-only records in each tier; dual-qualified URI; duplicate tags and high/low like/repost counts | AT-005, UT-005, UT-006, IT-005, IT-006 |
| TD-005 | Security/contracts/projection | Unique private/query/correction canaries; quote records; visibility matrix; real terminal lifecycle state; recent-history rows; viewer save/like state and images/quotes; before/after ingestion and pre-upgrade projection records | AT-006, AT-007, IT-007, IT-008, IT-010, IT-011, IT-012, IT-013, IT-014 |
| TD-006 | Finite bounds | 256-character input and over-boundary counterpart according to existing parser counting; repeated approved phrases, maximum supported terms, >8 correction candidates per eligible word, hidden candidates exceeding cap, page limit 100/101 | UT-003, UT-007, IT-009, IT-010 |

TD-001 seed records use `at://did:plc:searchfixture/social.craftsky.feed.post/<key>`. Each key below is a URI suffix, not a new record schema. Use `post-` prefixes for Posts fixtures and `project-` prefixes for Projects fixtures, with top-level published records and English metadata unless noted. Each row records query, URI/tab and usefulness rationale; preserve this metadata in the eventual checked-in fixture representation.

| Query | Useful URI suffix / tab | Authored content / usefulness rationale | Designated negative / reason |
|---|---|---|---|
| `Flax Light` | `post-flax`, `project-flax` / both | Caption or pattern name Flax Light; exact technical name | `post-flax-other`, `project-flax-other`: Flax without Light |
| `sock` | `post-socks`, `project-socks` / both | Finished knitting socks; English singular/plural topic match | `post-hat`, `project-hat`: knitting hats, no socks |
| `knit` | `post-socks`, `project-socks` / both | Finished knitting socks; English inflection topic match | `post-weaving`, `project-weaving`: weaving towels, no knitting; knitting hats are allowed |
| `how to knit socks` | `post-socks`, `project-socks` / both | Finished knitting socks; topic showcase, no instructional promise | `post-hat`, `project-hat`: knitting hats, no socks |
| `jumper` / `sweater` | `post-sweater` / `post-jumper`, `project-sweater` / `project-jumper` / both | Each spelling retrieves the other approved equivalent | `post-cardigan`, `project-cardigan`: cardigan-only, unlisted association |
| `WIP socks` / `work in progress socks` | `post-progress` / `post-wip`, `project-progress` / `project-wip` / both | Approved phrase equivalence with required socks concept | `post-progress-hat`, `project-progress-hat`: missing socks |
| `stockinette socks` / `stocking stitch socks` | `post-stocking` / `post-stockinette`, `project-stocking` / `project-stockinette` / both | Approved stitch-name equivalent | `post-garter`, `project-garter`: garter-only |
| `crochett blanket` | `post-crochet`, `project-crochet` / both | One minor misspelling, blanket concept retained | `post-crochet-hat`, `project-crochet-hat`: no blanket; two-correction record |
| `Flaxx Light` | `project-flax` / Projects | One pattern-name typo without dropping Light | `project-flax-other`: no Light |
| `blue wool socks` | `post-cross`, `project-cross` / both | Concepts collectively satisfied across scoped fields as AT-003 describes | `post-yarn`, `project-yarn`: blue yarn socks without wool |
| `shawl` | `post-alt`, `project-alt` / both | Own authored image alt text describes shawl; terse caption | `post-quote-only`, `project-unrelated`: term not in own scoped fields |
| `lace` | `post-tag`, `project-tag` / both | Authored hashtag/materialized authored tag contributes topic | `post-private-only`, `project-private-only`: only private marker context |
| `chaussettes` | `post-fr`, `project-fr` / both | Declared French literal word, viewer permits French | `post-unrelated-fr`, `project-unrelated-fr`: unrelated French record |

### Independent benchmark cases and concrete distractors

Each listed query is a separate case, and each “both” row produces separate Posts and Projects cases. Slash-separated queries produce one case per direction: `jumper` targets sweater, `sweater` targets jumper; `WIP socks` targets work-in-progress socks and the reverse; `stockinette socks` targets stocking-stitch socks and the reverse. Seed only that case's useful record, designated negatives and its six family distractors, so records from other cases cannot accidentally change expectations.

Useful records use the authored content in the query table: caption for ordinary posts; project title for structured-name examples, pattern name for Flax Light, and the expressly listed fields for cross-field/tag/alt examples. Every unlisted field is empty. All declare English except the literal French case, which declares `fr`; the viewer permits the case's language. Use a fixed timestamp and eligible author. Every negative must lack the required concept in all scoped fields, not merely its caption. These rules give each case explicit authored fields, language, tab and expected URI sets.

The six numbered items in each row are six separate top-level authored records. Put each item's text only in caption for Posts or project title for Projects, leaving other scoped fields empty. Their keys are `post-<family>-d1` through `post-<family>-d6`, or `project-<family>-d1` through `project-<family>-d6`, in listed order. They are designated negatives for the stated query only. Use English metadata except the French family. No family distractor receives a typo spelling that would independently satisfy its missing concepts.

| Family / query cases | Six concrete distractor texts, in key order | Why excluded |
|---|---|---|
| flax / `Flax Light`, `Flaxx Light` | 1. Flax pullover; 2. Light cardigan; 3. Harvest pullover; 4. Cabled cardigan; 5. Linen mittens; 6. Wool scarf | No record satisfies both Flax and Light, even after the permitted one-word correction |
| sock / `sock`, `how to knit socks` | 1. Finished knitting hats; 2. Blue knitted cardigan; 3. Knitted wool scarf; 4. Crochet blanket; 5. Woven linen towel; 6. Quilted cushion | No socks concept; question wording also requires knitting |
| knit / `knit` | 1. Woven linen towel; 2. Crochet blanket; 3. Quilted cotton cushion; 4. Embroidered sampler; 5. Felted wool bowl; 6. Cross stitch bookmark | No knit/knitting concept; knitting hats are deliberately not negatives |
| jumper / `jumper`, `sweater` | 1. Crochet blanket; 2. Woven scarf; 3. Quilted cushion; 4. Knitted mittens; 5. Felted bowl; 6. Linen trousers | Neither member of the approved jumper/sweater concept |
| progress / `WIP socks`, `work in progress socks` | 1. WIP hat; 2. Work in progress cardigan; 3. Finished socks; 4. Wool socks; 5. Progress on a quilt; 6. New sock yarn | Each lacks either the approved WIP concept or socks; progress alone is not the full approved phrase |
| stitch / `stockinette socks`, `stocking stitch socks` | 1. Garter stitch socks; 2. Ribbed socks; 3. Stockinette hat; 4. Stocking stitch scarf; 5. Cable blanket; 6. Moss stitch cardigan | Each lacks either socks or the approved stitch concept |
| crochet / `crochett blanket` | 1. Crochet hat; 2. Crochett scarf; 3. Woven blanket; 4. Knitted blanket; 5. Quilted blanket; 6. Crochet basket | Each lacks crochet/crochett or blanket; none can complete both concepts with the proposed single correction |
| cross / `blue wool socks` | 1. Blue yarn socks; 2. Red wool socks; 3. Blue wool hat; 4. White cotton socks; 5. Blue linen scarf; 6. Wool mittens | Each misses at least one of blue, wool, socks; yarn is not equivalent to wool |
| alt / `shawl` | 1. Knitted hat; 2. Crochet blanket; 3. Linen towel; 4. Wool mittens; 5. Cotton socks; 6. Quilted cushion | No shawl concept in any own field |
| tag / `lace` | 1. Cable cardigan; 2. Ribbed hat; 3. Wool mittens; 4. Linen towel; 5. Garter scarf; 6. Quilted cushion | No lace concept in any own field |
| french / `chaussettes` | 1. Bonnet en laine; 2. Écharpe bleue; 3. Couverture crochetée; 4. Serviette en lin; 5. Gants rouges; 6. Coussin brodé | Literal French query absent; no language detection or English word-form expansion |

The six negative distractors check precision, not the complete ranking contract. Controlled competing eligible records and field/tier priorities are tested separately through TD-004/IT-005. Keep the six-reliable corrected-result exception in its own case. Negative quote/private markers use actual quote/private setup from TD-005 rather than copying private text into published fields.

For typo fixtures, top-five is required when fewer than five reliable records precede the target. A separate six-reliable fixture asserts the explicit tier exception. Do not impose an exact full ranking on all benchmark distractors; only assert agreed tiers, useful top-five membership, designated exclusions and the controlled IT-005 comparisons. Keep negative assertions query-specific: a hat may qualify for knit while correctly failing socks.

## 8. Manual Checks

None identified. Retrieval, UI contract, privacy and lifecycle behavior have practical automation targets. Fixture relevance review belongs to document review rather than an extra manual release gate.

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Proposed numerical correction defaults need technical review | FR-003, NFR-002 | Edit distance, protected-word length and eight-candidate cap are engineering choices, not a selected implementation | A deterministic proposal is now documented below UT-007 (DR-003). Confirm it during coding plan or document revised boundaries and tests. No blocking product question |
| GAP-002 | Migration/backfill harness depends on projection design | NFR-001 | No migration or derived-vector strategy exists for this slice yet | Finalize IT-007 pre-upgrade schema cutoff and exact migration sequence in coding plan; use existing rows and unchanged record/CID assertions. If no migration is needed, retain pre-existing-row coverage |
| GAP-003 | Simplified schema fixtures can conceal lifecycle failures | RULE-002 | WithSchema may install all-active owner predicates | Run IT-010 and IT-014 against migrated production schemas with real lifecycle state; do not treat focused-schema results alone as lifecycle evidence |
| GAP-004 | Synthetic relevance examples may overfit | BR-001 | No actual user query dataset is available | Use the explicit independent cases and six concrete contrasting distractors per family in TD-001 (DR-004); verify fixture usefulness during implementation and incorporate user-provided examples if supplied; no query collection or analytics required |
| GAP-005 | Automated tests are designed, not implemented | All active requirements | This stage writes documentation only | Implement linked tests in the later TDD stage; report PostgreSQL skips as missing runtime evidence, never as successful integration coverage |

No blocking product gaps identified. Risk remains Medium. Document review is complete with an Approved with notes verdict; these revisions address its four notes, while proposed engineering defaults and the migration harness remain coding-plan follow-ups. Operational performance evaluation is intentionally excluded, not an unfilled test gap.

## 10. Out Of Scope

- Performance metrics, latency assertions, query-plan measurement, load tests and numerical performance gates; NFR-003/AC-013 remain retired. Existing instrumentation requires no change.
- Guaranteeing literal matches outrank approved synonyms; retain field weights and reliable-before-typo only.
- Matching-configuration versioning and snapshot/cross-deployment pagination continuity. Changed rules may require restarting search.
- Semantic search, external models/search services, generated descriptions, Google-style answers, instructional intent, automatic language detection and multiword typo correction.
- New UI controls/snippets, replies in results, merged tabs, new search-history behavior, changes to other discovery/feed surfaces, lexicon/PDS changes and production actions.
- Source/test-file creation, dependency changes, running tests or migrations, commits and deployment during this test-design stage.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-10-08-search-improvements/01-requirements.md`.
- Test specification: `docs/changes/2026-10-08-search-improvements/02-acceptance-tests.md`.
- Review artifact: `03-document-review.md`, created using `review-workflow-documents`; it records the pre-revision review.
- External Plannotator review, if initiated by the user outside this skill: `docs/changes/2026-10-08-search-improvements/`. No Plannotator session has been opened.
- Recommended first failing test: IT-001, declared-English `sock` retrieving an existing `socks` record in each submitted tab, with the language-boundary contrast tested on the reliable path directly, or using knitting/knit outside the proposed one-edit typo boundary. Do not assert that unknown-language socks is absent from a sock search: it may qualify independently through typo matching. This exposes the current simple-language matching limitation without introducing synonym or typo complexity.
- Suggested implementation test order: IT-001/UT-001/UT-002; UT-003 and IT-002/IT-004; UT-004/UT-007 and IT-003/IT-009; IT-005/UT-005; UT-006/IT-006/AT-005; IT-010/IT-011/IT-012/IT-013; IT-007/IT-008/IT-014; full AT-001 corpus and remaining acceptance/regression suites. Add tests before each implementation increment and keep visibility checks in every retrieval increment.
- Commands discovered: `just test` invokes `./scripts/appview-test full`; PostgreSQL-backed tests run on the host against the development compose database. `just dev-d` starts the dev stack when needed at implementation time. Focused host command after configuring the existing test database environment: `cd appview && TEST_DATABASE_REQUIRED=true go test ./internal/api -run '^TestSearchImprovements' -count=1`; analogous package runs for `./internal/index` and `./internal/db`. `just app-test test/search` runs Flutter search tests. Do not use unit-only `just appview-test-unit` as PostgreSQL acceptance evidence. Credentials must remain out of command output and documentation.
- Commands executed in this stage: Read-only inspection and document validation only; no runtime tests or migrations.
- Blocking gaps: None at product/test-design level. GAP-001 and GAP-002 must be resolved in the coding plan before claiming complete implemented coverage.
- Commit state: No stage commit requested or created.
- Review follow-up: DR-001 is addressed by separating reliable language normalization from typo eligibility; DR-002 by explicit scenarios and bound equivalence examples; DR-003 by the deterministic proposed policy; DR-004 by independent query/tab cases and enumerated distractors. Requirements and acceptance tests were revised together at the user’s request; the existing review artifact remains a record of the prior versions.
- Recommended next step: Coding plan under the existing Approved with notes verdict, carrying forward confirmation of proposed defaults and the migration harness. Implementation is not authorized by this document.
