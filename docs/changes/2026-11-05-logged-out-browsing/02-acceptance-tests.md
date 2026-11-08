# Acceptance Test Specification: Logged-out browsing

## 1. Test Strategy

Revision on 6 November 2026: basic submitted-query guest search only; no recent users, autocomplete/suggestions or top hashtags. Craft project feed sort/filter controls stay available. Existing IDs are preserved.

Design status: revised following document review DR-001–004 at the user’s request; **no test code has been created or run**. Existing test IDs and approved product scope are unchanged. `03-document-review.md` records the pre-revision review, not a fresh review of these edits. Requirements were approved on 5 November 2026, all assumptions confirmed, and backfill explicitly excluded. This specification uses the amended `01-requirements.md` and preserves its IDs.

Risk remains **High** because anonymous exposure, shared PDS profile mutations, nested content and metadata must agree. Verify real server decisions first, then client workflows. A passing browser mock suite proves client behavior against that contract; it does not prove AppView queries, live PDS OAuth or a production deployment.

Use:

- Go table-driven tests and `httptest` for policy, serialization, input limits and mutation adapters; real PostgreSQL integration with `testdb.WithSchema` and migrated-schema coverage for queries and projection behavior.
- Flutter `flutter_test`/Riverpod overrides for widget, router and provider transitions; `integration_test` for guest journeys on iOS/Android. Preserve the existing suite's fake repository conventions while adding server-backed corroboration where needed.
- Vitest for public upstream/auth adapters and guest state; Playwright for interactive guest journeys and initial HTML/OG. The existing public Playwright configuration disables JavaScript; retain it for SSR checks. Interactive cases need an explicitly JS-enabled project/context, not accidental execution in that configuration.
- Regression suites for member access, auth callbacks, account boundaries, search, disclosures and moderation. Update assertions that encode the intentionally replaced guest landing page, while retaining the behavior they protect.
- Limited manual checks for native assistive technology, platform share sheets, responsive visual quality and an authorized disposable-account interoperability smoke check.

No backfill means **no migration population from guessed defaults, startup/deploy sweep, historical replay, background catch-up or lazy guest-read repair**. Legacy unknown profiles remain hidden from guests until normal incoming profile events or an explicit owner settings operation establishes visibility. Tests must prove both the exclusion and absence of repair calls. Use in-memory/local fakes and ephemeral databases; do not modify production or real users' PDS records.

Automation targets below are existing suites/directories unless explicitly marked **proposed**. Exact new filenames/routes are deferred to the coding plan. Requirement coverage indicates planned coverage, never a passed result.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003, AC-004 | AT-001, AT-003, AT-004 | Acceptance | Yes |
| BR-002 | AC-008, AC-009, AC-010 | AT-008, AT-009, IT-005, IT-006 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-012 | AT-001, AT-011, IT-002, IT-009 | Acceptance / Integration | Yes |
| FR-002 | AC-003, AC-006 | AT-003, IT-003, IT-008, UT-004 | Acceptance / Integration / Unit | Yes |
| FR-003 | AC-004, AC-005 | AT-004, AT-005, IT-004 | Acceptance / Integration | Yes |
| FR-004 | AC-002 | AT-002, UT-003, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-005 | AC-002, AC-007 | AT-002, AT-006, UT-003 | Acceptance / Unit | Yes |
| FR-006 | AC-005, AC-006 | AT-005, AT-007, UT-002, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-007 | AT-006, AT-007, UT-005, REG-001, REG-002 | Acceptance / Unit / Regression | Yes |
| FR-008 | AC-008, AC-014 | AT-008, UT-006, IT-007, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-009, AC-010 | AT-009, UT-007, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-010, AC-011 | AT-009, AT-010, UT-001, IT-002, IT-003, IT-004, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-011 | AT-010, UT-008, IT-010, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-001 | AC-013 | AT-012, REG-004 | Acceptance / Regression; component review | Yes + review |
| NFR-002 | AC-012 | AT-011, UT-009, IT-009, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-003 | AC-015 | AT-002, AT-012, MAN-001, MAN-002 | Acceptance / Manual | Yes + manual |
| NFR-004 | AC-007, AC-010, AC-011 | AT-007, AT-009, AT-010, UT-005, IT-010 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-006 | AT-007, UT-002, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-010 | AT-009, UT-001, IT-002, IT-003, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-008, AC-016 | AT-008, AT-009, IT-011, REG-005, MAN-003 | Acceptance / Integration / Regression / Manual | Yes + manual |
| RULE-004 | AC-009, AC-010, AC-017 | AT-009, IT-005, IT-006, IT-012 | Acceptance / Integration | Yes |

## 3. Acceptance Scenarios

For all scenarios: use TD-001–TD-008 as applicable; assertions inspect visible behavior and network/persistence evidence. Run client scenarios on Flutter iOS/Android and Svelte responsive web;  Native mobile and web fixtures must represent the same eligibility states.

### AT-001: Explore chronological craft feeds

Requirement IDs: BR-001, FR-001
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/integration_test/` guest journey (**proposed**), Flutter feed/router suites; `web-client/tests/` JS-enabled guest journey (**proposed**).

```gherkin
Scenario: Explore and switch craft feeds without an account
  Given a guest with no member credentials and eligible projects across crafts
  And some projects have a different primary craft from an additional craft
  When the guest opens home and selects Explore CraftSky
  Then the unpersonalized all-crafts view is readable
  When the guest selects each available craft and requests another page
  Then existing chronological ordering and primary-craft eligibility are preserved
  And no followed-account or saved member preference request is made
```

### AT-002: Welcome modal and persistent banner

Requirement IDs: FR-004, FR-005, NFR-003
Acceptance Criteria: AC-002, AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/welcome_page_test.dart`, router tests; JS-enabled web guest suite (**proposed**).

```gherkin
Scenario: Explore once and choose authentication later
  Given a fresh guest browsing session on home
  Then the modal contains the existing registration, handle sign-in, help and legal content
  And Explore CraftSky is available
  When the guest explores and navigates between feeds, search and a profile
  Then the modal stays dismissed for that browsing session
  And the persistent banner offers Sign in and Register
  When either banner action or a legacy sign-in entry link is used
  Then the equivalent authentication flow opens with the current destination retained
  And focus is contained in the modal and restored after accessible dismissal
```

Exercise a second fresh session to verify recurrence. Do not equate every route change with a new session. Banner controls remain reachable during scroll and content respects the banner's occupied area.

### AT-003: Basic public search without discovery or member history

Requirement IDs: BR-001, FR-002
Acceptance Criteria: AC-003, AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/search/`, guest journey; web guest suite (**proposed**) with request capture.

```gherkin
Scenario Outline: Search eligible public data
  Given eligible, opted-out and unknown authors matching a unique query
  When a guest explicitly submits a query for <kind>
  Then only eligible result content is displayed using the existing default order
  And result navigation and additional pages work
  And no recent users, autocomplete/suggestions or top hashtags are shown or requested
  And no advanced search filters or search sort controls are shown
  And member recent searches and saved filters are neither fetched nor written
  Examples:
    | kind     |
    | person   |
    | post     |
    | project  |
    | hashtag  |
```

Before submission, open blank search, type and clear a query, switch categories, navigate back and reopen search: no recent-user, autocomplete/suggestion, top-hashtag or query request may fire just from those empty/typing states. After submission, paginate and open results; hidden-only matches must not expose excluded snippets. Use the existing default result order; guest search has no advanced filter/sort controls. Verify craft project feed sort/filter controls separately under AT-001; those remain available.

### AT-004: Deep links and full public reading

Requirement IDs: BR-001, FR-003
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: Flutter router/profile/thread suites and device guest journey; `web-client/tests/public-pages.spec.ts` plus interactive counterpart.

```gherkin
Scenario Outline: Open public content directly
  Given no active session
  When an eligible <destination> is opened on cold start or web reload
  Then the requested destination is shown without the automatic welcome modal
  And the applicable public profile sections, project fields, disclosures and thread are readable
  And no private owner fields are present
  Examples:
    | destination       |
    | profile           |
    | project post      |
    | general post      |
    | individual reply  |
```

Cover handle and DID profile entry where supported, rich-text links, timestamps, profile customization, public business sections, reply policy and comments. Use an inventory of existing public sections rather than inventing new social graph screens.

### AT-005: Read-only controls and media

Requirement IDs: FR-003, FR-006
Acceptance Criteria: AC-005, AC-006
Priority: Must
Level: Acceptance
Automation Target: Flutter `post_card_test.dart`, profile action and thread suites; web guest suite (**proposed**).

```gherkin
Scenario: Read, open media and share without mutation controls
  Given a guest viewing an eligible post and profile
  When they open comments, nested eligible replies, galleries, alt text, video and public links
  Then those reading operations work
  And read-only engagement counts and disclosures remain visible
  And Copy link and platform sharing are available
  And compose, reply creation, like, repost, follow, save and account-only navigation controls are absent
```

Native share-sheet completion is corroborated by MAN-002. Opening comments must remain possible even when the reply-creation control is hidden.

### AT-006: Authenticate and return to context

Requirement IDs: FR-005, FR-007
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance
Automation Target: Flutter auth/router/device journey; `web-client/tests/authenticated-session.spec.ts` and guest suite (**proposed**).

```gherkin
Scenario Outline: Authenticate from guest content
  Given a guest on a profile, post or filtered discovery destination
  When they choose <flow> from the banner
  And complete the existing provider handoff and any required onboarding
  Then they return to the retained safe public destination
  And member navigation replaces the guest banner
  And no content mutation is automatically replayed
  Examples:
    | flow      |
    | Sign in   |
    | Register  |
```

Also cancel provider handoff and inject transport/provider errors: exploration remains available and safe context is retained. Reject external/open-redirect destinations using existing navigation conventions. Live provider completion is a separate evidence boundary (MAN-003).

### AT-007: Protected URLs and session transitions

Requirement IDs: FR-006, FR-007, RULE-001, NFR-004
Acceptance Criteria: AC-006, AC-007
Priority: Must
Level: Acceptance
Automation Target: Flutter router/account-boundary suites; authenticated-session and guest web suites.

```gherkin
Scenario: Transition safely from member to guest
  Given a member has viewed private notifications, bookmarks, drafts or chat
  When the last active session is logged out or expires
  Then private viewer data is cleared before guest content appears
  And a guest-readable or authentication destination is offered
  When a private URL is opened directly or through browser back navigation
  Then it offers authentication and exposes no retained private data
```

Repeat with another retained valid account: preserve existing switch rules and isolate account generations. Complete a delayed old-account request after the transition and verify it cannot repopulate guest state.

### AT-008: Shared Privacy preference and honest recovery

Requirement IDs: BR-002, FR-008, RULE-003
Acceptance Criteria: AC-008, AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/privacy_page_test.dart`; `web-client/tests/authenticated-settings-detail-parity.spec.ts` or feature-focused extension.

```gherkin
Scenario: Save and reopen the shared visibility setting
  Given an authenticated owner with a known current standard profile preference
  When Privacy is opened and the logged-out visibility toggle is changed
  Then the setting explains its Bluesky effect, public-network limits and cached preview limits
  And a successful save survives reopening and another client read
  When settings reads, saves or required reauthorization fail
  Then no false success or public default is displayed
  And retry or recovery remains available
```

Cover both enabled and disabled starting states, repeated submissions, conflict recovery and preserved error-reporting consent controls. Do not expose the screenshot's algorithmic recommendation setting.

### AT-009: Opt-out across public surfaces and normal-event convergence

Requirement IDs: BR-002, FR-009, FR-010, NFR-004, RULE-002, RULE-003, RULE-004
Acceptance Criteria: AC-009, AC-010, AC-016, AC-017
Priority: Must
Level: Acceptance
Automation Target: server-backed fixture journey combining AppView integration and guest clients; fixture-driven UI cases supplement it.

```gherkin
Scenario: Shared preference changes apply to every guest representation
  Given an eligible author whose content appears in guest feeds, search, a thread and an embed
  When the owner successfully enables the shared opt-out
  Then the next guest read already excludes the account and authored presentation
  And direct access has a safe authentication-required or unavailable state
  And eligible neighbours remain readable without excluded author content
  And authenticated viewers retain otherwise permitted access
  When a normal incoming profile event disables the opt-out from another app
  Then eligible content becomes readable after that event is applied
```

Repeat for an unknown legacy profile: hidden until a normal incoming event or explicit owner operation establishes visibility; no repair activity is triggered by visits. Shared profile updates preserve unrelated fields/labels. State assertions are tied to completed ingestion/local acknowledgment, not arbitrary sleeps or an invented cross-network SLA.

### AT-010: SSR and previews obey the same guest boundary

Requirement IDs: FR-010, FR-011, NFR-004
Acceptance Criteria: AC-010, AC-011
Priority: Must
Level: Acceptance
Automation Target: `web-client/tests/public-pages.spec.ts` (JS disabled), HTTP-response assertions and JS-enabled hydration counterpart.

```gherkin
Scenario Outline: Render eligible or excluded public permalinks
  Given a post or profile with <visibility>
  When an anonymous browser or crawler requests its URL
  Then the initial HTML and page metadata use <presentation>
  And hydration, structured data, preload URLs and subsequent public data agree
  Examples:
    | visibility      | presentation                                  |
    | verified public | readable content and content-specific OG data  |
    | opted out       | generic CraftSky sign-in messaging, noindex    |
    | unknown         | safe unavailable content without protected data |
```

Use distinct canaries for restricted name/bio/text/project fields/avatar/media. On an excluded direct URL, identifiers already in the requested path may remain; no additional identity may be introduced. Capture raw response bytes, not only hidden DOM. Fetch after a visibility transition and verify no-store plus absence of stale content. Missing/deleted resources retain appropriate unavailable/error behavior.

### AT-011: Empty states, pagination and overload

Requirement IDs: FR-001, NFR-002
Acceptance Criteria: AC-012
Priority: Must
Level: Acceptance
Automation Target: Flutter feed/search provider suites; web guest suite; AppView public/search integration.

```gherkin
Scenario: Continue discovery through excluded candidates and retryable errors
  Given enough eligible records for multiple pages interleaved with hidden and unknown authors
  When a guest pages through a feed or search result
  Then each eligible record appears once in the existing order
  And hidden candidates do not create a premature end of results
  When no eligible records remain or a dependency, throttle or capacity failure occurs
  Then a genuine empty/end state or a usable retry state is shown
  And cursor errors do not crash or expose diagnostic payloads
```

Prove API bounds separately; do not require a particular number of frontend network calls or an unbounded scan to fill a page.

### AT-012: Shared presentation and accessible layouts

Requirement IDs: NFR-001, NFR-003
Acceptance Criteria: AC-013, AC-015
Priority: Must
Level: Acceptance
Automation Target: Flutter semantics/layout/component tests; Playwright role/focus/axe checks and responsive guest project (**proposed**).

```gherkin
Scenario: Read equivalent content with accessible guest chrome
  Given the same eligible content in guest and member modes
  When it is rendered at narrow and wide widths with enlarged text and reduced motion
  Then shared project, disclosure and media presentation remains equivalent
  And guest-only omissions match the capability rules
  And banner and modal controls are named, reachable and do not overlap reading/navigation
  And dismissing a modal restores focus to its invoking control or an appropriate home control
```

Include iOS/Android safe areas, long handle/copy and keyboard viewport changes. Component/service reuse itself is verified during review; brittle tests of import strings do not establish architecture quality.

### Reviewed surface coverage (DR-004)

Use the requirements section 16 inventory as the contract checklist, not only one representative profile/post. Expand existing test IDs with parameterized rows:

| Surface / request path category | Cases and evidence | Test IDs / AC IDs |
|---|---|---|
| Ordinary profile tabs | Projects, Posts, Comments, Reposts; eligible/opted-out/unknown author combinations; pagination and nested authors | AT-004–005, IT-004, IT-008; AC-004–006, AC-010 |
| Business public tabs/details | Products, Upcoming events, About and ordinary tabs; current non-owner conditional tab availability; public links and disclosures; no edit/manage actions | AT-004–005, AT-012, IT-004; AC-004–005, AC-013 |
| Identity/counts | Public profile identity/customization, post/reply counts; identity lists exclude hidden participants while permitted aggregate totals can include them; no mutual-follower/owner analytics fields | AT-009, IT-004, IT-008; AC-006, AC-010 |
| Search categories and query effects | People, posts, projects, hashtags; explicit submission, blank/typing/clear states, category switch, result open, back navigation and additional pages; no recent users/autocomplete/top hashtags or their requests | AT-003, UT-004, IT-003, IT-008; AC-003, AC-006, AC-010 |
| Filters/sorts and craft/tag routes | Craft project feeds retain existing filters/sort controls and transient selections; basic guest search/tag results use default order without advanced controls; recent-user/autocomplete/top-hashtag/saved-filter/history request spies remain empty | AT-001, AT-003, IT-002–003; AC-001, AC-003, AC-006, AC-012 |
| Thread/embed/media | Hidden intermediate ancestor, eligible descendant, excluded root permalink, mixed quote/repost authors, structured project fields, links, gallery/video/alt text and read-only reply policy | AT-004–005, AT-009–010, IT-004, IT-010; AC-004–006, AC-010–011 |

Fixture adapters must also fail on guest recent-user/autocomplete/top-hashtag requests, including existing member endpoints. Guest blank/typing search must not run the current recent-profile or top-hashtag loaders. Fixture adapters must reject unexpected private requests; hiding a recent-search panel while still fetching/saving history does not pass. In particular, guest entry must bypass the existing blank-search history fetch and query/result-triggered history-save effects. Map actual anonymous endpoints and response fields during coding planning; the inventory does not expose current member routes unchanged.

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-010, RULE-002 | AC-010 | Guest eligibility and nested subject policy | Known public, opted-out, unknown, malformed label and mixed-author quote/reply/repost fixtures | Only established public subjects render; excluded embeds/ancestors cannot leak through an eligible outer item; unrelated eligible commentary remains | Feature policy tests in `appview/internal/api/` (**proposed**) |
| UT-002 | FR-006, RULE-001 | AC-005, AC-006 | Capability-to-control and route decisions | Signed-out/loading/signed-in states; public/private destination/action inventory | Reading actions available, mutation controls absent, protected routes offer auth; loading is not an owner bypass | Flutter router/shared action suites; Svelte shell/state unit tests (**proposed**) |
| UT-003 | FR-004, FR-005 | AC-002 | Modal recurrence and retained destination | Fresh session, Explore, navigation, CTA reopen, legacy link, second session | Automatic prompt once per session; CTA reopens; current destination retained; no prompt on direct public content | Flutter welcome/router; Svelte guest state tests (**proposed**) |
| UT-004 | FR-002 | AC-003, AC-006 | Guest search request shaping | Each search category; blank/typing/submitted/cleared states; forbidden discovery and member-history request spies | Only submitted-query/page/category-result requests with allowed basic parameters; no autocomplete/recent-user/top-hashtag/history requests or advanced search controls | `app/test/search/`; web search tests in `src/lib/features/` |
| UT-005 | FR-007, NFR-004 | AC-007 | Account-boundary and late response handling | Guest to member, member to guest, retained-account switch, stale outstanding request | Prior generations/caches discarded; late result ignored; safe destination survives; no mutation replay | `app/test/router/`, auth/provider tests; web session state tests |
| UT-006 | FR-008 | AC-008, AC-014 | Settings UI state and failures | Unknown read, known true/false, saving, conflict, transport failure, reauth required | No false public default/success; accessible retry; cross-app/privacy copy present | Flutter Privacy widget; web settings Vitest/component suite |
| UT-007 | FR-009 | AC-009 | Standard profile label patch construction | Multiple other labels, target already present/absent/duplicated, unknown profile fields and precision-sensitive values | Add/remove target idempotently; preserve other fields/labels and current authoritative record | Existing profile command tests in `appview/internal/api/profile_test.go` or feature-focused extension |
| UT-008 | FR-011, NFR-004 | AC-011 | Safe public adapter/metadata mapping | Eligible vs excluded/error responses; hostile URL/query and credential canaries; oversized/redirected upstream responses | Specific metadata only for eligible content; generic excluded metadata; no sensitive query reflection, credential forwarding or response-size bypass | `web-client/src/lib/server/api.test.ts`, public helpers/metadata tests |
| UT-009 | NFR-002 | AC-012 | Error, cursor and bound validation | Invalid/oversized/reused cursor, excessive limit, timeout, throttle, non-2xx | Existing error envelope/status conventions and bounded work; safe retry state; no raw protected error string | Go public/search handler tests; Flutter API tests; web upstream tests |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | NFR-004, RULE-004 | AC-010, AC-017 | Migration and unknown-state semantics without backfill | Local database at prior schema with legacy profile/content rows and unrecorded labels | Apply candidate schema in test database; initialize app under instrumented adapters; request legacy profile | Schema permits unknown state; no guessed public value or existing-profile population; guest-hidden; zero sweep/replay/read-repair calls | Migrated-schema AppView tests (**proposed**), `public_pages_test.go` |
| IT-002 | FR-001, FR-010, RULE-002 | AC-001, AC-010, AC-012 | Actual anonymous feed queries | TD-001/002/003 across primary/additional crafts with deterministic ties | Read all-crafts and craft pages under anonymous policy | Existing order/primary-craft eligibility; excluded/unknown authors filtered before limits; no duplication/skipping of eligible rows | Go feed/public store tests using real Postgres |
| IT-003 | FR-002, FR-010, RULE-002 | AC-003, AC-010, AC-012 | Basic search and prohibited discovery isolation | Unique terms in eligible and excluded people/posts/projects/hashtags | Submit category queries and later pages; probe excluded guest-discovery endpoints and unsupported advanced query parameters | Eligible results/snippets only in default order; no anonymous recent-user/autocomplete/top-hashtag route; unsupported advanced search parameters rejected; no member history persistence | `search_improvements_acceptance_test.go`, `search_cursor_test.go`, public/search extension |
| IT-004 | FR-003, FR-010, RULE-002 | AC-004, AC-005, AC-010 | Thread, profile sections and nested serialization | Mixed eligible/excluded reply chain, opted-out root, quotes/reposts; rich public profile/project | Read direct detail, thread, profile tabs and existing public relationship/activity endpoints | Every nested subject checked; safe ancestor/embed states; eligible neighbours and public disclosures retained; identity lists exclude hidden subjects while allowed aggregate totals may remain | `public_pages_test.go`, thread/profile store tests (**feature extensions**) |
| IT-005 | BR-002, FR-009, RULE-004 | AC-009 | Owner PDS write/local visibility acknowledgment | Fake PDS with real AppView command boundary, unknown fields/labels, record CID; concurrent edit barrier | Enable/disable; inject stale revision, hold/release a pre-save event after acknowledgment, PDS failure, local projection failure and retry | Delayed pre-save event cannot restore guests after acknowledged opt-out; correct owner record only; other data preserved; conflict recovered or reported; successful enablement already suppresses guests; partial failure recoverable without false completed success | Go profile command adapter integration with instrumented fake PDS |
| IT-006 | BR-002, FR-009, RULE-004 | AC-010, AC-017 | Normal profile ingestion in both projection paths | Table-driven constructors for ordinary and image-safety profile indexers; known/unknown states including unknown legacy row at the incoming CID | Deliver normal create/update/delete, same-CID unknown initialization, duplicate established CID, replacement and stale event sequences via supported ingestion ordering boundary | Same-CID normal evidence establishes unknown visibility with target label present/absent; target label/verified absence retained; established duplicates idempotent; stale events cannot override newer effective state; failures remain safe; unknown becomes public only on authoritative evidence | `appview/internal/index/bluesky_profile_test.go` plus dispatcher/order harness as needed |
| IT-007 | FR-008 | AC-008, AC-014 | Settings contract and owner authorization | Two local owners and instrumented session/PDS adapters | Read/save/reopen per client contract; try another owner and unauthenticated access; inject failures | Same shared preference, no cross-owner mutation or false success; useful errors and retry/reauth behavior | API/route contract tests plus Flutter/Svelte repository adapters |
| IT-008 | FR-002, FR-006, RULE-001 | AC-006 | Anonymous route and DTO allowlists | TD-006 private canaries; no credentials, malformed credentials and spoofed viewer headers | Call each allowed guest read and representative write/private routes | Public contract cannot acquire member/owner privileges; writes/private reads rejected; cookies/tokens not reflected; no private canaries in serialized output or history writes | Route inventory/policy tests, `routes_public_pages_test.go`, public serialization tests |
| IT-009 | FR-001, NFR-002 | AC-012 | Bounded capacity and query work | Controlled limiter/concurrency and dependency barriers; many excluded rows | Saturate slots, exceed quota, abort requests, send bad bounds and retry | Bounded query/result/body behavior; expected 429/503 and Retry-After; slots released on cancel/error; no unbounded hidden-row scan; pagination remains usable | Public route admission, handler and real-Postgres pagination tests |
| IT-010 | FR-010, FR-011, NFR-004 | AC-010, AC-011 | Web SSR/OG shares eligibility with guest data | Same fixture state exposed via real guest contract; separate cookies/browser contexts; canary-rich author | Fetch initial HTML/data before and after applied opt-out; hydrate; request with crawler UA | No excluded canaries/media/preloads in any representation; generic restricted preview/noindex; no-store; eligible positive controls remain; forwarded identity cannot lift guest boundary | `public-pages.spec.ts`, public adapter tests; local AppView-backed HTTP corroboration (**proposed**) |
| IT-011 | RULE-003 | AC-016 | Existing public moderation/lifecycle/image policy | Suspended/deleting/nonmember authors, account/post warn/hide/takedown, uncleared/stale-safe images | Read all expanded guest surfaces and compare member normal policy | Guest conservative exclusions preserved throughout; no owner bypass; opt-out does not bypass/alter independent member moderation | Migrated-schema Go public/search/thread integration, existing safety fixtures |
| IT-012 | RULE-004 | AC-017 | No backfill or lazy repair during normal operation | Legacy unknown profiles; outbound PDS/resolver, replay and job adapters instrumented; no new profile events | Start/restart app, browse home/search/detail repeatedly, advance local job clock; then deliver one normal profile event | No added profile-fetch, replay, catch-up job or legacy mutation occurs; existing necessary handle resolution is distinguishable from prohibited profile repair; normal event updates only its subject under ordinary processing | AppView composition/startup test seam and API/index integration (**proposed**) |

Concrete DR-001 fixtures for **IT-006**: in each ordinary/image-safety indexer path, start with an unknown legacy visibility projection and stored CID X. Deliver a normal authoritative profile record with CID X, once with the target label and once without it. Visibility must become established hidden/public respectively. Deliver the established record again and verify idempotence. Keep an untouched legacy control author unknown and assert zero profile-repair/replay calls. No historical delivery is requested to set up runtime behavior.

Concrete DR-002 sequence for **IT-005/006**: hold a pre-save public event, acknowledge a newer owner opt-out save, release the held event, and request guest data. The author must remain excluded. Also simulate PDS write success followed by local projection failure: report recoverable incomplete success, retain/recover safe guest visibility and allow retry without clobbering concurrent fields. Subsequent valid external edits still converge normally. Apply the same ordering checks to rapid enable/disable sequences.

Do not assume standalone indexers provide ordering if that responsibility belongs upstream. The coding plan must locate the real owner-save/ingestion ordering boundary and its existing version/sequence evidence; CID lexical order is not such evidence. Exercise that boundary using barriers/controlled completions, not sleeps. Unknown-until-event behavior is intentional; do not silently reintroduce backfill to make legacy fixtures pass.

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | OAuth handoff, callbacks, onboarding, add-account and eligibility routes | FR-004, FR-007 | AC-002, AC-007 | Extend Flutter router/OAuth tests and web authenticated-session/auth tests for modal entry while preserving callback validation, registered links, initialization and retained-account flows. |
| REG-002 | Member/guest and retained-account isolation | FR-007, NFR-004 | AC-007 | Reuse Flutter account-boundary fixtures and web session tests with owner-only data and a delayed request crossing logout/switch. Restore member capabilities only for the new authenticated context. |
| REG-003 | Public readable SSR, safe canonical/OG and provider fallback pages | FR-011, NFR-004 | AC-011 | Retain `public-pages.spec.ts` no-JS positive/error cases, canonical/query-canary and exact verified-link callback/association-file assertions; add excluded metadata cases. |
| REG-004 | Member content/project/media and business disclosure parity | NFR-001, FR-003 | AC-005, AC-013 | Run existing post-card/profile/project suites and web authenticated detail/social parity suites; compare identical eligible guest content through shared components, allowing deliberate action omissions. Review shared policy/component reuse without source-string tests. |
| REG-005 | Independent moderation, blocks, lifecycle, image safety and account eligibility | RULE-003 | AC-016 | Existing member restrictions remain; new guest paths have no owner bypass. Allow opt-out authors for eligible signed-in viewers while retaining all ordinary restrictions. |
| REG-006 | Privacy error-reporting consent remains independent | FR-008 | AC-008, AC-014 | Retain `privacy_page_test.dart` persistence check and web settings diagnostics/consent coverage; visibility changes neither toggle reporting nor overwrite its preference. |
| REG-007 | Existing search and public admission semantics | NFR-002, FR-002 | AC-003, AC-012 | Preserve member search sorting/refinements/cursors and guest body/time/origin limits, safe errors and slot release; member recent/saved features still work for members. |

## 7. Test Data

All data is synthetic and local. Use valid typed DID/handle/rkey/CID values where parser/real-record validation matters; do not copy production accounts. Existing lightweight fixtures may remain only where identifier validation is outside the tested boundary.

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Positive public reading | Verified-public regular and business authors; general post, rich project with all supported field categories, links/facets, timestamps, disclosures, media/alt text; multiple crafts and equal-time sort ties | AT-001–005, AT-010–012, IT-002–004 |
| TD-002 | Opt-out canaries | Opted-out author with distinct restricted display name, bio, text, project title, avatar/banner/media URLs and image alt text; only-requested URL identifiers tracked separately | AT-003, AT-009–010, UT-001, IT-002–004, IT-010 |
| TD-003 | Legacy/unknown and malformed states | Existing profile lacking visibility projection with stored CID equal to a normally incoming record, tested with target label present/absent in both paths; untouched unknown control author; separate verified-absence record; failed/unsupported label read; deleted profile; no natural update initially | AT-009–011, IT-001, IT-006, IT-012 |
| TD-004 | Mixed nested graph | Eligible root plus opted-out and eligible reply authors, hidden intermediate parent with eligible descendants, hidden root, eligible outer quote/repost referencing hidden content, counts including hidden participants | AT-004–005, AT-009, IT-004 |
| TD-005 | Profile mutation/concurrency | Standard self-label union with target, unrelated labels, unknown extension fields, precision-sensitive JSON; revisions before/after a concurrent edit; delayed/repeated events and partial failures | AT-008–009, UT-007, IT-005–007 |
| TD-006 | Auth/private boundaries | Two member accounts and guest; owner-only draft, chat, bookmark, notification, search-history and token canaries; delayed outstanding responses; safe and hostile return destinations | AT-006–007, UT-004–005, IT-008, REG-001–002 |
| TD-007 | Public moderation/capacity | Suspension/deletion/nonmembership, warn/hide/takedown, image state variants, hidden-only windows, many interleaved rows, failure/timeout/slot barriers | AT-011, IT-009, IT-011 |
| TD-008 | Layout/accessibility | Long handles, translated/long copy, empty/many images, text scaling, narrow/wide viewports, reduced motion, keyboard open and mobile safe areas | AT-002, AT-012, MAN-001–002 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-003 | AC-015 | Native assistive-technology usability | On iOS VoiceOver and Android TalkBack, explore modal, banner, feed, search and thread at enlarged text; use keyboard/screen reader on web | Logical reading/focus order; labelled actions; no focus trap after dismissal or blocked reading area; reduced motion respected |
| MAN-002 | FR-003, FR-006, NFR-003 | AC-005, AC-015 | Native sharing and visual safe-area quality | Copy/share a guest post, cancel native share, open media and keyboard; inspect phone/tablet/web widths | Correct public URL, cancellation returns to content, no hidden mutation controls or overlapping banner/navigation/media |
| MAN-003 | FR-008, FR-009, RULE-003 | AC-008, AC-009, AC-010, AC-016 | Optional authorized disposable-account interoperation smoke | With an explicitly authorized test account, complete real OAuth; change preference in CraftSky and observe shared profile/Bluesky; change externally and await normal ingestion; restore starting state | Same label is used; other profile data retained; guest suppression and eligible member behavior converge; no claim of external preview cache revocation |

Manual/live work is not performed during test design. Use device tools for any later simulator/emulator interaction so it is visible in the Device panel. MAN-003 needs authorized disposable infrastructure/account access; mocked adapter checks remain mandatory regardless of live access.

### Proposed executable web guest matrix (DR-003)

The following is a **planned test harness contract**, not files/scripts already present:

- Config: `web-client/tests/playwright-guest.config.ts`; `testDir: '.'`, `testMatch: 'guest-*.spec.ts'`, JavaScript enabled. The coding plan may share utilities with the existing runner, but must preserve this explicit suite selection or document an equivalent command.
- Interactive files: `guest-browsing.spec.ts`, `guest-auth-transitions.spec.ts`, `guest-privacy.spec.ts` and `guest-accessibility.spec.ts` under `web-client/tests/` (**proposed**). Map AT-001–012 to these suites rather than duplicating each scenario in every file.
- Reuse/extend `tests/mock-appview.mjs` and existing auth-fixture utilities where practical. Add verified-public/opted-out/unknown authors, ordinary/business tab data, mixed nested content, pagination, session transitions and injected settings failures. Private request spies fail on guest-history or owner-only reads/writes. Use separate contexts and deterministic fixture resets; isolate server ports from other runners.
- Interactive automated projects: Chromium desktop, Chromium narrow mobile viewport with touch, and WebKit mobile viewport; include a tablet-width layout case. Native iOS/Android guest journeys remain separate and cannot be proven by browser emulation.
- Keep `playwright.config.ts` and its JS-disabled `public-pages.spec.ts` SSR assertions. Run the guest config separately for Node preview and Worker modes, with explicit project/test discovery output. A green public/member suite does not count as execution of guest tests.
- The later coding plan must map these proposed files to concrete fixtures, runtime/browser dependencies, ports and CI/maintainer validation commands. No package or config changes are made at this document stage.

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Current web public runner is JS-disabled; member mock runner assumes member routes | FR-004–007, NFR-003–004 | AC-002, AC-005–007, AC-015 | Existing runner alone cannot establish interactive guest coverage | The proposed config, spec pattern, fixtures, browser matrix and commands in section 8 define the intended runner; coding plan must wire them explicitly and retain the no-JS SSR suite separately |
| GAP-002 | Client mocks do not establish real AppView policy or cross-network OAuth | FR-009–011, RULE-001–004 | AC-006, AC-009–011, AC-016–017 | Different evidence boundaries | Pair mocked journeys with actual Postgres/API/index tests and local server-backed corroboration; label any unavailable live smoke separately |
| GAP-003 | Exact new route/DTO/policy seam and ordering owner not chosen yet | FR-010, NFR-001, RULE-004 | AC-010, AC-013, AC-017 | Deferred architecture detail, not missing behavior | Requirements section 16 and test section 3 now enumerate surfaces; coding plan completes actual guest route/field/component mappings and same-CID/order/no-backfill seams. No new product decision required |
| GAP-004 | External preview caches and arbitrary PDS readers cannot be controlled | FR-008, FR-011, RULE-003 | AC-008, AC-011 | Platform limitation, outside CraftSky's authority | Assert truthful copy and safe new CraftSky responses, never require deletion of external cached copies |
| GAP-005 | Real assistive technology/share sheets and live PDS access not validated in this stage | NFR-003, FR-003, FR-009 | AC-005, AC-009, AC-015 | Planning-only stage and environment-dependent access | Record device/browser matrix and unexecuted manual/live checks when implementation is validated |

No acceptance criterion is intentionally left without a planned verification path. These are execution/design follow-ups, not claims of missing approved requirements. No blocker prevents document review. Implementation remains unapproved.

## 10. Out Of Scope

- Test implementation/execution in this stage; source/dependency/config changes, migration execution, commits, pushes and deployment.
- Any backfill, inferred opt-in population, historical replay or read-triggered repair; their **absence** is in scope for verification.
- Algorithmic feed parity, anonymous mutations, private PDS records, new social graph features, new warning click-through/age policies or desktop-native applications.
- Real production changes, third-party cache eviction, universal scraping prevention or guaranteed immediate external propagation.
- Pixel-perfect Bluesky copying or brittle import/source-text assertions. Protect observable behavior and review architecture directly.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-11-05-logged-out-browsing/01-requirements.md` (approved, amended to exclude backfill).
- Test specification: `docs/changes/2026-11-05-logged-out-browsing/02-acceptance-tests.md` (revised per DR-001–004 at user request; not executed).
- Next review artifact: `03-document-review.md`, via `review-workflow-documents`, unless the user explicitly skips document review and requests `write-coding-plan`.
- External Plannotator review, if initiated by the user: this workflow folder.
- Recommended first failing test: **IT-001**, with a legacy profile row and protected canaries. Assert migration/startup does not mark it public or fetch its PDS record and guest detail exposes no content. Add **IT-006** next to establish verified opt-out/absence through normal incoming events in both projection paths.
- Suggested order: unknown/no-backfill boundary → label projection and owner mutation/concurrency → real guest feed/search/thread/serialization → SSR/OG and cache boundary → Flutter/Svelte guest workflows and auth transitions → accessibility and member regression checks. Keep red/green slices small.
- Review follow-up: DR-001 is incorporated in AC-017, IT-006 and TD-003; DR-002 in AC-009 and deterministic IT-005/006 sequences; DR-003 in the proposed guest runner/matrix and commands; DR-004 in the requirements section 16 inventory and parameterized surface coverage above. Concrete implementation choices remain coding-plan deliverables, not unresolved product questions.
- Risk: **High**. User approval exists for requirements/test design only; explicit approval is required before implementation.
- Blocking gaps: none for document review; map proposed automation seams and record available live/device validation during coding planning.

### Commands discovered (future implementation validation; not run)

From the repository root, with compose development dependencies running:

```sh
./scripts/appview-test focused ./internal/api -run 'TestPublic'
./scripts/appview-test focused ./internal/index -run 'TestBlueskyProfile'
./scripts/appview-test focused ./internal/routes -run 'TestPublicPage'
just test
just app-analyze
just app-test test/router/router_redirect_test.dart test/auth/welcome_page_test.dart test/settings/privacy_page_test.dart
```

Focused patterns must be extended to include the actual feature test names created later. `just test` enables required real PostgreSQL/MinIO integration with race checks; unit-only paths are incomplete evidence. `just app-test` runs the checkout's Rive setup first.

From `web-client/`:

```sh
npm run check
npm test
npm run build
npm run test:e2e
npm run test:e2e:app
WEB_CLIENT_WORKER=1 npm run test:e2e
WEB_CLIENT_WORKER=1 npm run test:e2e:app
```

Proposed guest commands from `web-client/` (config/spec files will be created only during authorized implementation):

```sh
npx playwright test --config tests/playwright-guest.config.ts --list
npx playwright test --config tests/playwright-guest.config.ts
WEB_CLIENT_WORKER=1 npx playwright test --config tests/playwright-guest.config.ts
```

Run discovery first to establish that the guest specs and intended projects are selected; an empty discovery is not a pass. Build before preview/Worker runs as required by the existing harness. Existing public/member commands alone are not proof guest tests ran. Device integration uses `just app-test-integration DEVICE` with the selected device ID and the checked-in harness; extend that harness for the new journey or document a feature-specific runner in the coding plan. No device or third-party validation is claimed here.
