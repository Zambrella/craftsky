# TDD Implementation Plan: Logged-out browsing

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md`
- Approved coding plan: `04-coding-plan.md`
- User approved the coding plan and TDD implementation on 6 November 2026, including its local schema, auth and privacy work. No commits, pushes or production changes authorized.

## Implementation Rules
- One test ID at a time: meaningful red, minimum implementation, same test green, then refactor.
- Preserve no-backfill, basic submitted-query guest search, no recent users/autocomplete/top hashtags; craft project feed filters/sorting remain.
- Preserve existing member flows and never expose member/private data to guests.
- Record local tests separately from device/live-provider proof.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Status |
|---|---|---|---|---|
| 1 | IT-001 | NFR-004, RULE-004 | AC-010, AC-017 | completed |
| 2 | IT-012 | RULE-004 | AC-017 | completed |
| 3 | UT-001 | FR-010, RULE-002 | AC-010 | completed |
| 4 | IT-006 | BR-002, FR-009, RULE-004 | AC-010, AC-017 | completed |
| 5 | UT-007 | FR-009 | AC-009 | completed |
| 6 | IT-005 | BR-002, FR-009, RULE-004 | AC-009 | completed |
| 7 | IT-007 | FR-008 | AC-008, AC-014 | completed |
| 8 | IT-008 | FR-002, FR-006, RULE-001 | AC-006 | completed |
| 9 | IT-009 | FR-001, NFR-002 | AC-012 | completed |
| 10 | UT-009 | NFR-002 | AC-012 | completed |
| 11 | IT-002 | FR-001, FR-010, RULE-002 | AC-001, AC-010, AC-012 | completed |
| 12 | IT-003 | FR-002, FR-010, RULE-002 | AC-003, AC-010, AC-012 | completed |
| 13 | IT-004 | FR-003, FR-010, RULE-002 | AC-004, AC-005, AC-010 | completed |
| 14 | IT-011 | RULE-003 | AC-016 | completed |
| 15 | UT-008 | FR-011, NFR-004 | AC-011 | completed |
| 16 | IT-010 | FR-010, FR-011, NFR-004 | AC-010, AC-011 | completed |
| 17 | AT-010 | FR-010, FR-011, NFR-004 | AC-010, AC-011 | completed |
| 18 | REG-003 | FR-011, NFR-004 | AC-011 | completed |
| 19 | UT-002 | FR-006, RULE-001 | AC-005, AC-006 | completed |
| 20 | UT-003 | FR-004, FR-005 | AC-002 | completed |
| 21 | UT-005 | FR-007, NFR-004 | AC-007 | completed |
| 22 | AT-002 | FR-004, FR-005, NFR-003 | AC-002, AC-015 | completed |
| 23 | AT-007 | FR-006, FR-007, NFR-004, RULE-001 | AC-006, AC-007 | completed |
| 24 | UT-004 | FR-002 | AC-003, AC-006 | completed |
| 25 | AT-001 | BR-001, FR-001 | AC-001 | completed |
| 26 | AT-003 | BR-001, FR-002 | AC-003, AC-006 | completed |
| 27 | AT-004 | BR-001, FR-003 | AC-004 | completed |
| 28 | AT-005 | FR-003, FR-006 | AC-005, AC-006 | completed |
| 29 | UT-006 | FR-008 | AC-008, AC-014 | completed |
| 30 | AT-008 | BR-002, FR-008, RULE-003 | AC-008, AC-014 | completed |
| 31 | AT-009 | BR-002, FR-009, FR-010, NFR-004, RULE-002, RULE-003, RULE-004 | AC-009, AC-010, AC-016, AC-017 | completed |
| 32 | AT-006 | FR-005, FR-007 | AC-007 | completed |
| 33 | REG-001 | FR-004, FR-007 | AC-002, AC-007 | completed |
| 34 | REG-002 | FR-007, NFR-004 | AC-007 | completed |
| 35 | AT-011 | FR-001, NFR-002 | AC-012 | completed |
| 36 | AT-012 | NFR-001, NFR-003 | AC-001, AC-003, AC-004, AC-006, AC-010, AC-012, AC-013, AC-015 | completed |
| 37 | REG-004 | FR-003, NFR-001 | AC-005, AC-013 | completed |
| 38 | REG-005 | RULE-003 | AC-016 | completed |
| 39 | REG-006 | FR-008 | AC-008, AC-014 | completed |
| 40 | REG-007 | FR-002, NFR-002 | AC-003, AC-012 | completed |
| 41 | MAN-001 | NFR-003 | AC-015 | not run - manual assistive technology |
| 42 | MAN-002 | FR-003, FR-006, NFR-003 | AC-005, AC-015 | partial - automated layouts only |
| 43 | MAN-003 | FR-008, FR-009, RULE-003 | AC-008, AC-009, AC-010, AC-016 | not run - optional live authorization absent |

## Implementation Steps
### IT-001 / RULE-004, NFR-004 / AC-017
- First test: `TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown`.
- Focused command: `./scripts/appview-test focused ./internal/db -run '^TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown$'`.
- Environment: worktree-local PostgreSQL/MinIO started with approved elevated local tooling. No production access.
- Red: focused test failed because migration 000114 was absent.
- Green: same focused test passed (real PostgreSQL, 0.413s). Migration adds only nullable evidence/hold columns and constraints; no population statements. Guest-detail policy integration follows in later slices.

### Order adjustment
- Move IT-012 after the guest read service exists: its startup/request-level no-repair evidence depends on those interfaces. UT-001 is active next, then IT-006. No scope or criteria are removed.

## Completion Checklist
- [x] All Must requirements covered by automated tests or the explicit manual gaps below
- [x] Automated Must tests passing; manual/live evidence is separately qualified below
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Initial implementation review completed (`06-implementation-review.md`); correction review remains the next stage

### UT-001 / FR-010, RULE-002 / AC-010
- Red: no production guestvisibility package/record policy existed.
- Green: TestGuestVisibilityRecordPreference passed (0.258s): absent/empty/other labels, opt-out, malformed/unknown unions and null records; only authoritative public state permits guests.

### IT-006 / FR-009, RULE-004 / AC-010, AC-017
- Red: TestBlueskyProfileGuestVisibilitySameCIDAndOrdering failed in both ordinary and image-safety paths: same-CID normal record left hidden/revision NULL.
- Implementing typed SQL projection and revision guard; green pending.

### IT-006 outcome
- Green: same-CID/order/delete test passed in both indexer paths (1.843s); actual normal duplicate ingress test passed (1.181s), with no repository jobs and no reopening completed projections.
- Nearby regression initially expected physical removal of invalid profile row. Updated it to verify cleared presentation plus retained unknown revision floor, as explicitly planned. Visibility/index/ingestion/db focused matrix passed (0.087s/2.630s/1.119s/0.389s).

### UT-007 / FR-009 / AC-009
- Red: SetPreference absent. Green: raw profile patch preserves other labels/extensions and 9007199254740993, removes target duplicates, rejects unknown union, is idempotent (0.198s).

### IT-005 / FR-009, RULE-004 / AC-009
- Red: no unchanged-record finalizer. Green: TestGuestVisibilityUnchangedRecordFinalizesAtomically passes (1.033s); injected failure rolls back local evidence/acceptance; retry accepts without PDS write, terminal replay does not rerun hook.
- Owner read/save command and delayed-source fencing slice in progress; production route/client integration remains pending.

### IT-005/007 owner command and contract evidence
- Owner save fences delayed public events (1.118s); PDS-success/local-failure retry reuses one write and clears its hold (1.223s).
- Verified profile absence can create the minimal standard profile (1.113s). GET/PUT handler tests enforce authenticated owner identity, strict fields and no-store (0.592s); actual route auth/inventory tests pass (0.523s).
- Nearby addressed/post-edit command regressions pass (12.457s). Further concurrent-intent and client recovery coverage remains open; these IDs are not yet complete.

### IT-008 public boundary slice
- Red: unknown, opted-out and pending-write profiles/posts/lists returned 200 with private canaries.
- Green: TestPublicPagesEnforceGuestVisibilityWithoutRepair (1.239s); all existing public-page tests with explicitly verified public fixtures pass (2.063s). Unknown/pending stays unavailable; known opt-out offers generic authentication-required. No new repair jobs.
- Expanded route/DTO coverage remains open.

### UT-009 public request bounds slice
- Red: excessive/invalid limits, duplicate and unsupported parameters reached the reader with 200.
- Green: existing public handlers reject them before dependencies; public-page matrix passes (2.281s). Expanded-route cursor binding and client transport error coverage remain open.

### IT-002/003 shared feed/search query slices
- Red: public craft-feed and submitted-query entry points absent. Green: production-schema feed test filters interleaved hidden/unknown owners before pagination, retaining primary-craft filtering (1.389s).
- Search's four categories now use explicit GuestAudience before ranking/aggregation. Public-search test passes (1.334s), including hashtag counts and eligible continuation pages. Fixture was corrected to use production-generated tag spellings rather than writing that generated column.
- UT-009 cursor red: a craft cursor was accepted for another filter. Green: feed/search cursors bind audience, resource and query/filter/sort scope; three-test matrix passes (2.898s).

### IT-004 thread/embed slice
- Red: no public thread/profile-comment entry points or quote presentation.
- Green: independently eligible replies remain readable under unavailable ancestors; hidden/unknown replies are excluded before limit; quote embeddings use a minimal unavailable state with no identity (1.471s). Public counts/media/facets are selected explicitly; member flags remain absent.

### IT-008 expanded route slice
- Red: craft-feed route returned 404.
- Added concrete feed, four basic submitted-query search, comments/reposts and thread routes under the existing shared anonymous admission budget. Owner/private and discovery routes remain protected or absent. Focused route/inventory green pending.

### Further backend evidence
- Expanded route/inventory slice passes (0.528s), including event, interaction and exact-tag registrations; all use the same anonymous limiter/slots.
- Public profile facts/customisation and page/thread matrix pass (3.366s). Business events and interaction identities pass (2.026s); canonical production source fixtures are required for aggregate foreign keys.
- Shared feed/search/member read regressions pass (82.230s). No deployment or live-provider checks performed.
- IT-005 deterministic overlap red: a retry prepared before acceptance reinstalled the hold. Green: lock the command state before installing a hold; all visibility command tests pass (4.584s).
- IT-005 conflicting observation red: returning an error rolled back unknown visibility. Green: persist fail-closed evidence, then return the recoverable conflict (0.974s).
- Exact-tag search extension passes (1.548s), with only eligible source content.

### Test order adjustment
- Backend public read slices (IT-002/003/004/008 and UT-009) were needed to exercise the shared privacy boundary before client wiring. Their focused evidence is recorded above; full surface/client coverage remains pending. Continue with SSR error mapping (UT-008), then client read/capability/runtime slices and their acceptance journeys. No test IDs or scope are removed.

### Backend regression gate and web transport
- Complete affected Go package suites pass for API (185.976s), business (3.573s), PDS commands (60.473s), index (96.427s), ingestion (12.723s), DB (36.951s), policy (0.217s).
- Routes exposed missing diagnostic/access/suspension inventory entries and a direct registration violating the existing capability-registrar convention. Updated those explicit inventories and extracted the settings registrar; the five failing route checks now pass (0.582s). Full route package rerun remains due.
- UT-008 red: upstream 403 became 503. Green: generic authentication-required mapping; 11 server-fetch tests pass.
- IT-008 web proxy red: no shared route allowlist/proxy existed. Green: 38 route/query cases plus proxy/error tests, 52 tests total. Only closed GET routes and approved query keys pass; credentials are omitted, bodies/time/redirects remain bounded.
- Local npm dependencies were installed from the lockfile. Vite's local runtime requires the approved elevated test runner for localhost/cache access. No hosting actions performed.
- UT-002/IT-008 guest client slice red: public client absent. Green: 3 tests prove read adaptation, credential omission, and local private/discovery/mutation rejection. Display-only member flags default inert locally; they never expand the anonymous wire contract.

### UT-003 / FR-004, FR-005 / AC-002
- Red: no guest welcome-session model. Green: 3 tests cover Explore dismissal across remount/reload in a tab, fresh-tab prompt, direct-link reading and blocked storage fallback.

### AT-002 web home/modal slice
- Red: `/feed` redirected to the old standalone sign-in page and had no dialog.
- Green: Chromium desktop journey passes (1.8s): existing shared Feed reads all public crafts, existing sign-in content is reused in a dialog with Explore, Escape/dismiss/reload obey tab-session behavior, and a persistent sign-in/register banner remains.
- Added the planned JS-enabled runner on ports 19850/19851 with desktop/mobile Chromium, mobile WebKit and tablet projects. Other viewports and the rest of the guest UI remain pending.
- Shared `MemberPost` now accepts an absent viewer identity; no fake guest account is created. Its remaining member-action controls will be gated in the next capability acceptance slice.
- Full routes suite rerun now passes (22.803s).

### AT-007 web read-only content slice
- Red: the existing shared post rendered a Like button for a guest.
- Green: two desktop journeys pass: guest actions become read-only engagement links plus Copy link; member mutation/chat/settings links disappear, and no member API requests are issued. The member branch retains its existing markup/behavior.

### AT-003 / UT-004 web submitted search slice
- Red: guest search route had no search input.
- Green: desktop browser journey passes (2.5s): blank query prompt only, no discovery/autocomplete/history effects while typing or clearing, explicit submit and category navigation read filtered public results. Existing shared Search and Tag UI is reused; guest read limits stay 20.
- Guest profile/thread enhanced UI, settings, history/auth return and Flutter work remain pending. This is not stage completion.
- Guest welcome/read-only controls pass across all four browser projects. Mobile search initially targeted the existing visually hidden submit button; the journey now uses the actual keyboard submission path. A separate initial-hydration race could discard early input; shared Search now waits for mount before enabling its field. Search and profile/thread journeys pass on desktop, mobile Chromium, mobile WebKit and tablet (8 checks, 10.4s). WebKit was installed locally; no production/browser-provider proof is implied.
- AT-004 red: public profile had no Comments tab. Green: shared non-owner Profile and Thread views render public comment tabs and multi-author comments with no fabricated session. Guest mutation controls are hidden; initial public SSR/OG branches are retained pending the broader SSR parity slice.

### Owner recovery refinements (IT-005/006/007)
- Owner observation revisions are repository-head bounds, not the profile write revision. A matching-CID normal source event may now project matching presentation without lowering that later floor. Red covered ordinary/image-safety paths; focused index/command/policy matrix passes (3.559s/4.910s/0.218s).
- Closing/reopening settings after PDS success/local acceptance failure must retain the original pending operation. The owner-only response now optionally includes `pending: {operationKey, expectedCid, hideFromLoggedOut}`; the immutable command retains the original expected CID. This is an additive recovery detail for FR-008/009, not a public field or a new repair worker. Reopen/retry test passes with one total PDS write (1.106s).
- A rejected stale-CID save previously released its hold against stale local visibility. It now observes/persists the current shared preference before releasing only its resolved terminal hold. Unavailable observation retains the hold; explicit owner reload recovers it (1.727s).
- Unsupported authoritative profile/label variants now persist unknown and return an error instead of retaining/assuming public state. Focused green pending; clients will display honest error/pending states.
- Unsupported owner-record fail-closed matrix and all visibility commands pass (6.983s).
- Retention red: command compaction could orphan a terminal privacy hold. Green: held command identities remain until explicit owner recovery, with a partial hold-key index and no data population; command/DB matrix passes (9.096s/0.495s). Ordinary terminal compaction remains available after the hold clears.
- Flutter locked dependencies are now resolved locally. Beginning its explicit viewer/capability and read-epoch slices; no native UI behavior has been changed yet.

### Native UT-002 / UT-005 / IT-008 foundations
- Red: no explicit browsing context/read guard/public adapter existed; an account client leaked its bearer and a supplied Cookie onto public paths.
- Green: seven context/adapter/late-read tests and eight session-auth tests pass. The read guard rejects loading and changed epochs; guest transport maps only closed GET routes and adapts inert display values. Account auth interception strips credentials on public paths before device/session lookup.
- Provider/UI integration remains in progress; this does not establish native journey completion.

### Native AT-001/002 and UT-003
- Red: actual signed-out `/feed` routed to the standalone welcome screen, with no Home underneath. Green: actual router journey reaches shared Home, opens reused Welcome content with Explore, dismisses to All crafts with persistent CTA and Home/Search-only navigation (2s).
- The checked-in `just app-test` runner installed its required checkout-local Rive test library. This is host test setup, not a native macOS build target.
- Guest session tests cover process-lifetime dismissal and closed public auth-return destinations. Shared content providers now select a closed public GET transport and use read audience guards. Further read-only controls, stale-state and native detail checks remain open.

### Native guest controls and UT-006 settings
- Native home/blank search/typing/shared post journeys pass together (3 checks, 1s); guest profile action variant passes separately. Read-only rendering bypasses reaction assets and private overlays; guest sharing excludes chat.
- UT-006 red: neither client's visibility settings model existed. Green: web model tests (2) and native controller test retain the last confirmed state and immutable command identity through pending/retry; owner reload recovers pending intent. Existing error-reporting Privacy regression passes with the owner-setting seam explicitly stubbed (2 checks, 1s).
- Both Privacy UIs include cross-app effect, open-network limits and cached-preview limitations. Browser component journey, full error matrix and native reopening/lease-transition checks remain pending.
- Web check is green with zero errors/warnings after these changes. No physical-device or live-PDS validation is claimed.

### Read compatibility / web route continuation
- Existing member post/profile/project provider regression matrix passes (11 checks) after isolated tests explicitly supply a member browsing context. Production loading remains fail-closed; the member operation guard was not relaxed.
- Guest engagement routes initially rendered a placeholder; they now use shared Thread, and filter routes use the existing filter page. The browser test found the persistent CTA covering its fixed Apply footer; guest layout now measures the banner and reserves its height. Full viewport verification remains in progress.
- Explicit web `/sign-in` now redirects to the feed modal, retaining authenticated Add account. The auth start handoff already persisted a server-validated return path; the guest modal now supplies its public reading destination. Browser checks are exposing and fixing early hydration timing; no completion is claimed yet.

### UT-005 cache boundary refinement / AT-004 permalink safety
- Red: a member row remained readable through the current list's retained AsyncValue while the anonymous read was pending. Riverpod intentionally retains previous values; assigning a plain loading state did not clear it.
- Green: guest-capable read families now include the immutable browsing audience in their cache key, and UI/mutation-refresh call sites select that audience explicitly. The selected cache has no previous member rows while the guest request is pending; the focused transition test passes. This follows the coding plan's read-lease/cache-isolation requirement, with no change to member mutation authority.
- Guest navigation mounts only its active public branch; member IndexedStack behavior is retained. Regenerated Riverpod/router outputs; removed unrelated mapper trailing-newline churn.
- Eligible comment permalink browser test initially lacked an unavailable-ancestor placeholder. The shared thread now keeps a guest on that eligible reply instead of redirecting to its excluded root; a generic placeholder reveals no identity. Focused desktop journey passes (2.4s).
- Desktop/mobile sign-in-return and engagement/filter checks now pass together (4 checks, 7.5s), after fixing both fixed filter controls and early hydration CTA timing. Remaining browser projects, history, SSR/business sections and native regression matrices are still being completed.


## Final implementation status — 6 November 2026

The approved feature is implemented locally across AppView, Flutter and the Svelte web client. Earlier “pending” notes above describe intermediate TDD checkpoints; this section and the test-order table record the final stage state. No commits, pushes, releases, deployments, production writes or historical backfills were performed.

### Delivered behavior
- Guests enter the existing craft project feeds, including All crafts, with retained project filters and sort controls. Profiles, ordinary/business tabs, projects/posts, comments/replies, engagement identities, quote previews, events, image/video presentation and guarded captions use the shared reading components.
- Search is explicitly submitted and uses the four basic result categories. Blank/typing/clear states do not fetch recent users, suggestions/autocomplete, top hashtags or member history; guest result sorting/refinements and account mutations are absent.
- The existing welcome content appears in a dismissible home modal with Explore CraftSky. Explicit sign-in entries can reopen it after exploration; persistent sign-in/register controls reserve their own layout space. Direct public links stay unobstructed. Public location/history state survives navigation and the validated sign-in handoff. Excluded-content recovery links and the banner use a home modal with a retained public destination; the banner also has a normal GET form fallback before hydration.
- Privacy edits the standard shared `app.bsky.actor.profile/self` self-label `!no-unauthenticated`. It preserves unrelated fields/labels, uses the existing addressed CAS/idempotency flow and exposes only confirmed state. Unknown reads show loading/error, pending saves retain their original intent/key, and successful owner reads recover an unresolved save after reopening.
- Nullable authoritative visibility, repository revision fences and a durable pre-dispatch hold prevent stale ingestion or incomplete local acceptance from re-exposing an owner. Legacy/unsupported/unknown states stay hidden. Only a normal authoritative event or an explicit owner settings operation establishes visibility.
- Public API reads and SSR/OG use the same conservative policy and explicit DTOs. Excluded embeds/ancestors carry no identity/content; eligible independent commentary remains readable. No public caller gains a viewer through cookies or bearer/device headers.

### Final red/green refinements
- Guest history: a settled Posts → Projects → Posts journey made a second posts request. Adding a guest-scoped instance of the existing PageHistory restored cached category content, Back state and scroll handling; the same journey then passed. Guest caches are disposed when their runtime leaves.
- Native cache lifetime: the selected Riverpod family initially retained a member canary while a guest read was pending. Audience-qualified family keys removed that retained value. A separate obsolete single-post audience test initially performed a read; it now rejects before repository IO. Every continuation validates its read guard before accessing Ref after an async gap.
- Native member editor: same-owner reactivation unmounted the profile form during a fresh read. Keeping its same-owner snapshot preserves unsaved text/accepted image state while the operation lease still cancels stale work. The original image/lifecycle regression tests pass.
- Owner pending state: both controllers initially called an unresolved acknowledgement “Saved”; they now reserve success for a settled response. The web checkbox initially retained its browser-toggled value through a 202; it now restores the confirmed value immediately and exposes the pending retry action. Browser/native UI tests verify the original command identity and independent reporting consent.
- Explicit native sign-in: after Explore, a legacy sign-in route did not reopen the modal because the mounted guest frame did not observe resolved route changes. The guest frame now observes the same router delegate used by the member shell, handles explicit entry independently of the automatic prompt and consumes the entry flag.
- WebKit focus: the final Escape assertion exposed that touch activation does not focus the banner button before native dialog opening. The guest runtime now records the invoking control and restores focus explicitly after closing; automatic entry uses the Home link as its fallback.
- Admission: a previously canceled request entered dependency work. The admission boundary now rejects it before acquisition; an in-flight cancellation test verifies the shared slot is released.
- Public captions: guest media could not reach the authenticated caption route. An explicit public caption route shares admission, eligibility, indexed membership and bounded WebVTT validation; both clients/proxy adapt its caption bytes/text without inventing a viewer.
- Browser error recovery: a sign-in return directed at an excluded SSR page could not open an interactive modal. Explicit sign-in now enters the readable home page, retains the validated public destination separately and consumes the request on mount/navigation events. A reactive URL-consumption attempt caused a feedback loop and was replaced with event handling before final verification. The persistent banner has a progressive GET fallback, preserving recovery even on the error page before hydration.
- Auth regression harness: home is now `/feed` before sign-in, so that URL can no longer prove authentication completed. The session helper now waits for the actual provider confirmation before asserting retained accounts; this corrects a test race without changing member auth behavior.
- Browser sign-in: modal-only control parameters initially survived into the authenticated return URL. The public destination now excludes those parameters, while confirmation-pending recovery remains available. A separate session-expiry regression retains the safe public profile tab through reauthentication.

### Acceptance evidence by test group
| Test IDs | Implemented evidence |
|---|---|
| IT-001, IT-012 | Exact migration up/down/up preserves nullable legacy state; repeated public-store composition and legacy browsing leave profile snapshots/job counts unchanged; normal duplicate delivery establishes only its received subject without reopening completed work. No new startup/worker repair path exists. |
| UT-001, UT-007, IT-005, IT-006, AT-009 | Standard-label parsing/raw patch tests; real migrated-Postgres source/observation/receipt ordering; both indexers; unchanged writes, absent profile, delayed events, conflicting/unsupported evidence, overlapping retry, PDS success/local failure and held-command retention/recovery. |
| IT-007, UT-006, AT-008, REG-006 | Authenticated GET/PUT route policy/body/owner fences; web authenticated proxy allowlist; controller pending/reopen cases; native/browser unknown and pending UI; error-reporting consent persistence remains independent. |
| IT-002–004, IT-008, UT-002, UT-004, AT-001, AT-003–005, AT-007 | Real-Postgres project/search/profile/thread/repost/interaction/business queries; pre-pagination exclusions; nested canaries; closed client/proxy read maps and credential removal; actual shared guest UI and prohibited-request spies. |
| IT-009, UT-009, AT-011, REG-007 | Aggregate quota/concurrency rejection with Retry-After; cancellation slot release; bounded page/query/cursor validation and audience/filter scopes; upstream streamed body/time/redirect/error controls; existing member search regressions. |
| IT-010–011, UT-008, AT-010, REG-003, REG-005 | Conservative membership/lifecycle/moderation/image predicates; public metadata and nested DTO allowlists; unknown/opt-out canaries; no-JS readable profile/post and business SSR, canonical/OG/error/accessibility checks; member restrictions still pass. |
| UT-003, UT-005, AT-002, AT-006, REG-001–002 | Modal lifetime and explicit reopen; safe auth return and session-expiry return; native audience transitions/late completion; existing OAuth/onboarding/add-account/session isolation suites. |
| AT-012, REG-004 | Shared post/project/profile/business/media presentation; phone/tablet/desktop browser projects, WebKit, accessible modal/read-only controls and measured banner offsets; full member visual/detail suites; native shared-widget suite and iOS fixture integration. Physical/manual limits are below. |

### Verification results
| Check | Result |
|---|---|
| Go affected packages: `api`, `routes`, `guestvisibility`, `index`, `ingestion`, `pdscommands`, `db`, `business` | All passed with local migrated PostgreSQL/MinIO. Package durations: 236.135s, 62.373s, 0.246s, 124.370s, 22.278s, 92.752s, 56.313s, 7.881s respectively. This was not the whole-repository race/release gate. |
| Flutter full suite: `flutter test --no-pub --reporter expanded` | 3,890 passed, 44 skipped, 6m10s. Final focused deltas separately verify the obsolete-audience guard, Privacy pending/unknown UI and explicit modal reopen. |
| Native final modal/router/welcome/Privacy/read delta | 61 passed after the final resolved-route fix. Native analysis reports zero issues. |
| Native final Privacy/lifecycle delta | 12 passed; includes unchanged reporting consent and exact pending-command retry. |
| Web units | 374 passed in 58 files, including safe expiry return, public proxy/cursors, pending settings and owner route allowlist. |
| Public no-JS browser suite | Preview: 10 passed. Local Worker: 10 passed. Includes safe metadata/canonical behavior and light/dark accessibility. |
| Authenticated browser suite | Preview: 326 passed, 5 skipped. Local Worker: 326 passed, 5 skipped. After the final error-recovery changes, the 10 session/Privacy journeys passed again in both runtimes (8.0s Worker, 6.7s preview). Skips are feature-off release variants and Sentry-enabled diagnostics, not enabled by this local configuration. |
| Guest browser matrix | Preview: 52 passed. Local Worker: 52 passed. Each runs desktop Chromium, phone Chromium, phone WebKit and tablet Chromium. |
| iOS simulator fixture integration | Three guest journeys passed again after the final native entry fix on iPhone 18 Pro, iOS 27.2 (59.2s incremental Xcode build, 4s tests). Real native runtime, isolated repositories; no real OAuth/PDS or physical-device claim. |
| Web/static generation/tooling | Svelte check and production build passed; private source maps prepared only, not uploaded. Pinned `just guest-visibility-sqlc` regeneration passed. Python dev/release/deploy tooling suites: 5, 7 and 8 passed; `just --list` parsed the updated recipes. |

### Project adaptations and operational boundaries
- Shared social content/API repositories are reused. The guest shells are deliberately small separate runtime wrappers so account/chat/private discovery providers do not mount for guests; this avoids a broad member navigation rewrite. Existing member behavior is covered by the full suites.
- Visibility DTO types live in the API layer. The browser and server proxy share one closed public path validator; the native Dio interceptor uses its corresponding explicit map. The member proxy remains authenticated.
- Public video captions are an additional public route required by the existing video reader. No custom `social.craftsky.*` Lexicon schema changed; the opt-out is the existing Bluesky profile label.
- Added `test:e2e:guest` and preview/Worker guest gates to the web quality workflow, including Chromium/WebKit installation. Added `just app-test-guest-integration DEVICE` and the pinned scoped sqlc recipe.
- Preview/Worker suites must run sequentially: concurrent preview instances contend on the checkout's local Miniflare SQLite files. The final qualifying runs were serial. Initial failing runs and incorrect test selectors were corrected and are not treated as passing evidence.
- Build-runner emitted unrelated mapper trailing-newline churn and a broad formatter expanded the generated project taxonomy; those unrelated changes were removed. Keep the actual generated model/provider/router changes.
- Backend schema/routes must ship before the new clients. No deployment was performed. Unknown historical rows intentionally remain unavailable until a normal event or owner setting establishes their shared preference; there is no population task to run.

### Explicit verification gaps
- **MAN-001:** Physical iOS VoiceOver/Android TalkBack and a human screen-reader/enlarged-text pass were not performed. Automated semantics/browser accessibility and responsive journeys are corroboration, not a substitute for that manual proof.
- **MAN-002:** Automated web phone/tablet layouts and iOS guest navigation ran. Manual native media/clipboard/share-cancel/keyboard/safe-area inspection across physical devices remains unverified. The shared guest menu keeps the existing Copy link action and omits chat sharing; no new OS share sheet was introduced.
- **Android emulator:** Device panel refused opening with “Android emulator video needs working ffmpeg with the libx264 encoder.” `/opt/homebrew/bin/ffmpeg` does expose libx264; the Device service nevertheless requests PATH/configuration repair and a local environment-server restart. That host runtime was left untouched. No Android simulator/device pass is claimed, and no alternative device-control channel was used.
- **MAN-003:** Optional real-PDS/Bluesky interoperation smoke was not run because no disposable live account/infrastructure write authorization was supplied. All shared-record write/order/retention assertions use mandatory isolated protocol/database fixtures. External cached link previews cannot be revoked by this implementation.
- Implementation review is the next workflow stage and has not been run. No claim is made for remote CI, a release build on physical devices, deployment, external crawler refresh timing or a live provider journey.


### Stage handoff
- Final native modal/router/welcome/Privacy/read run: 61 passed; final iOS fixture rerun: 3 passed; final native analysis: zero issues.
- Final guest runs include both excluded-content recovery links and progressive banner submission, preserved auth return, explicit focus restoration, craft requests and basic search: 52 passed on Worker and 52 passed on preview. The final core session/Privacy reruns passed 10 each after correcting the authentication-completion barrier.
- Final web units: 374 passed; formatting checks: 141 native source/test files unchanged by Dart format and all changed web source/tests accepted by Prettier.
- `git diff --check` is clean. Existing lockfiles and native platform project configuration were not changed. All work remains uncommitted in this worktree.
- TDD implementation is complete with the explicit manual/live gaps above. Per implement-tdd's stage exit gate, stop here and offer implementation review, a manual comment, or no further action. Do not start the review stage or publish changes automatically.

## Implementation-review corrections — 6 November 2026

The user chose Address required changes after `06-implementation-review.md` reported Changes required. This authorizes the correction work, including the existing local privacy/command boundary. Prior automated completion counts above are historical execution evidence; IR-001–004 remain unmet until the correction tests below pass. No new product behavior, backfill, live PDS operation, commit or deployment is included.

| Order | Finding / test IDs | Requirements / criteria | Status |
|---|---|---|---|
| C1 | IR-003 / IT-005, IT-007 | FR-008–009, RULE-004 / AC-008–010, AC-014 | completed |
| C2 | IR-001 / AT-004–005, REG-004 | FR-003 / AC-004–005 | completed |
| C3 | IR-002 / IT-010, AT-010, REG-003 | FR-003, FR-011, NFR-001 / AC-004, AC-011, AC-013 | completed |
| C4 | IR-004 / IT-004, REG-004 | FR-003, NFR-001 / AC-005, AC-013 | completed |
| C1 supplement | IT-005, IT-007 | FR-008–009, RULE-004 / AC-008–010, AC-014 | completed |
| C5 | REG-003, AT-006 | FR-007, NFR-004 / AC-007, AC-011 | completed |

Correction order follows the review handoff: privacy recovery first, then canonical route identity, SSR handoff and public provenance. Each behavior receives a permanent regression test before its implementation change. The original review remains an independent snapshot; correction evidence will be recorded here before another review.

### C1 / IT-005 / FR-008–009, RULE-004 / AC-009, AC-014
- Red: permanent `TestGuestVisibilityRecoversAfterExternalEdit` returned ambiguous in all four local-acceptance/response-loss × enable/disable cases. Corrected test raw-JSON CID encoding first; setup errors are not red evidence.
- Green: same four cases passed (3.770s package). Recovery accepts the current profile only for this preference operation, with stable head/revision advancement beyond its immutable dispatch. It never replays the old profile. A subsequent deliberate toggle preserves the external display name and precision-sensitive extension field. The fake PDS now increments its current revision rather than deriving it solely from AppView write count, so external commits retain monotonic revisions.
- Next loop: an external edit reversing the desired preference must resolve as conflict with current authoritative state, rather than remain pending.
- Second red: all four external-preference-reversal cases stayed ambiguous while the original four still passed.
- Second green: all eight cases passed (7.723s package). A newer contradictory preference resolves to 409 with current visibility persisted and the matching hold cleared atomically; unchanged desired preference resolves to its current CID. Reopen, terminal replay and a later owner toggle work in every case. Generic dispatch completion still rejects ambiguous-to-rejected recovery; only the private profile recovery path can perform it after proving the advanced head in its completion transaction.

### C2 / IR-001 / AT-004–005, REG-004 / FR-003 / AC-004–005
- Red: distinct-subject anchor navigation changed URL/title to Bob while Alice remained visible. The fixture first needed to normalize the client's supported `@did` form; that initial selector timeout was a setup issue, not the qualifying failure.
- Green: key the canonical shared profile/post components by subject identity; existing disposal invalidates their pending work. The permanent browser journey now covers profile and quoted-post anchor navigation plus Back/Forward and matching metadata/content (1 passed, 3.3s runner).
- Nearby privacy/command regressions passed: pdscommands 33.843s, guestvisibility 0.189s, API 26.148s, routes 0.522s with the GuestVisibility/PostEdit/Addressed filter.

### C3 / IR-002 / IT-010, REG-003 / FR-003, FR-011, NFR-001 / AC-004, AC-011, AC-013
- Red: browser API 503 removed the successfully rendered profile; the cursor fixture's second page was present in initial HTML and absent after enhancement. Both permanent JS regressions failed before implementation.
- Shared Profile now initializes identity/tab/page/events/continuation state through the existing public display adapter. Its route lifetime also includes tab/input cursor. Shared Thread initializes the loaded public post and fetches only the missing thread information on first mount; a retryable comments failure retains the root.
- Green: SSR failure retention, cursor/history and canonical navigation journeys all pass on desktop Chromium. No duplicate profile/root-post browser reads occur during the SSR handoff. No-JS markup and metadata remain derived from the same public loader data.

### C4 / IR-004 / IT-004, REG-004 / FR-003, NFR-001 / AC-005, AC-013
- Red: permanent public DTO test confirmed canonical provenance existed but `externalImport` was absent in guest JSON.
- Green: public allowlist/conversion now preserves the existing safe `ExternalImportResponse` (0.553s package). Native regression decodes the public wire through the real display adapter and renders the same imported label in member and guest shared-card contexts; card/transport suites passed 112 tests.
- No new provenance trust/verification claim or custom Lexicon field was introduced.

### Additional C1 recovery boundary checks
- During final verification, checking the same approved partial-success behavior for the already-supported absent standard profile and for insufficient observation evidence. This supplements IT-005/007; no new operation or background repair is introduced.
- Missing-profile red: response-loss recovery could not find the create dispatch because its immutable-body lookup accepted only update steps. Green: allow the existing single standard-profile create step only for the exact owner visibility operation; generic edit recovery remains update-only.
- Added fail-closed observation controls to the external-edit matrix: unchanged dispatch head or non-advanced revision cannot settle the command or release its hold. No arbitrary sleep/deadline replaces repository evidence.
- External-deletion red: a lost visibility-write response followed by deletion in another app remained ambiguous. Green: this adapter reuses its existing stable head/record/head absence observer for recovery, yielding a recoverable conflict without recreating the profile. Generic strict edit observation is unchanged. Final recovery/evidence suite passed (8.335s).
- Removed the two formatter-only immediate-image-upload diffs noted by the review; their feature behavior was untouched.

### Correction regression verification
- Full affected Go packages passed against local migrated PostgreSQL/MinIO: API 198.134s, pdscommands 65.607s, guestvisibility 0.234s, routes 35.400s. This is affected-package validation, not a whole-repository race/release gate.
- Native final card/guest-transport/Privacy run: 115 passed; native analysis: no issues. Initial analysis identified four lint issues in the new test; those were corrected before the qualifying run.
- Web check: zero errors/warnings; units: 374 passed in 58 files; production build passed, private source maps prepared only.
- Guest preview matrix: 68 passed (29.7s), including canonical profile/post navigation, history, failing browser requests after successful SSR, cursor pages and a barrier-controlled previous-profile pagination response.
- Initial broad guest execution exposed fixture-only issues (a quote duplicating existing comment text, and treating an empty cursor as a later page); these were corrected, with the quote moved to a dedicated permalink. An intervening fixture edit omitted the generic post fallback; that run was stopped, the fallback restored and the affected three journeys passed before the qualifying full matrix. None of those incomplete/failing runs counts as passing evidence.

### C5 / REG-003 / FR-007, NFR-004 / AC-007, AC-011
- Final no-JS regression was meaningfully red: the banner's progressive sign-in form copied an unrelated `code=PRIVATE_CANARY` query into its hidden return field. Existing canonical/metadata filtering did not sanitize this separate form value. Adding the smallest shared public-return query allowlist, while retaining supported craft/search/thread navigation context; this corrects an existing Must regression within the approved auth-return scope.
- Guest Worker matrix already passed 68 tests (33.3s); subsequent browser gates paused at the no-JS failure, rather than treating the remaining unexecuted gates as passes.
- C5 green: the existing no-JS suite now passes all 10 tests (5.0s). Public return targets use route-specific query allowlists, including repeated craft filters and supported profile/thread/search state; unrelated parameters are omitted in both the progressive form and interactive modal. Two new positive banner-return journeys preserve craft filters/sort and submitted search while dropping the canary (2 passed, 3.6s).
- The new positive feed-return browser test initially raced automatic modal opening when checking visibility without waiting. It now waits explicitly for the feed welcome modal before Explore; both new return cases pass on WebKit (4.4s runner). This was a harness readiness correction, not a product behavior change. The final full guest runs include these two extra cases on every project.

### Final correction status and handoff

IR-001–004 have implementation corrections and permanent passing regression coverage. The additional no-JS return-field regression (C5) is also corrected. `06-implementation-review.md` remains the original independent review snapshot; its verdict is not silently changed to approval by this implementation pass. A follow-up implementation review is the next stage.

| Final correction gate | Result | Evidence boundary |
|---|---|---|
| Full affected Go packages | API 198.134s; pdscommands 65.607s; guestvisibility 0.234s; routes 35.400s — all passed | Local migrated PostgreSQL/MinIO; not a whole-repository release gate |
| Race-enabled command/visibility checks | Focused `-race` checks for GuestVisibility and Addressed: passed, 71.261s / 1.231s | Includes recovery, overlapping retry and generic addressed-command controls; no production/PDS proof |
| Final focused recovery after green cleanup | Passed, 11.036s | External unrelated edits, opposing preference, both directions, local failure/response loss, invalid head/revision, profile creation and external deletion |
| Flutter card, guest transport and Privacy | 115 passed | Real widget/model/adapter tests; no new device journey claimed |
| Flutter analysis | No issues | Includes final new native test |
| Web check / units / build | Zero errors/warnings; 374 tests in 58 files; build passed | Private source maps prepared only; no upload/deployment |
| Final guest matrix | Preview 76 passed (35.8s); Worker 76 passed (53.2s) | Desktop Chromium, phone Chromium, phone WebKit and tablet Chromium, including the two new sanitized-return cases |
| No-JS public SSR | Preview 10 passed (5.0s); Worker 10 passed (7.9s) | Readability, metadata/canonical canaries, links, accessibility and progressive sign-in form |
| Relevant member browsers | Preview 53 passed (30.6s); Worker 53 passed (37.7s) | `authenticated-social-detail-parity`, `authenticated-social-regressions`, `authenticated-session`, `authenticated-guest-visibility`; this is the selected affected regression set, not another full 326-test run |
| Formatting / diff | Scoped Prettier check, Go formatting and `git diff --check` clean | The unrelated immediate-image formatter diffs were removed |

Race command: `./scripts/appview-test focused -race ./internal/pdscommands ./internal/guestvisibility -run 'GuestVisibility|Addressed'`.

All browser servers ran serially, preserving the local Miniflare constraint. Failing setup/readiness runs above were corrected and excluded from these final pass counts. Manual assistive technology, physical native share/media/safe-area inspection, Android runtime and optional real-PDS/Bluesky interoperability gaps remain exactly as documented in the initial implementation handoff; local mocks and simulator history do not fill them.

No custom Lexicon, OAuth scope, migration, background job, backfill, live account, production resource, commit, push or deployment was introduced by the correction pass. The public API adds only the existing safe imported-post display provenance; owner privacy writes still target the same standard profile and use server-side credentials and durable fencing. All changes remain local and uncommitted.

TDD corrections and relevant final checks are complete. Per implement-tdd, stop here and offer implementation review, a manual comment, or no further action. Do not start review or publish automatically.

## IR-005 correction — fresh same-profile loader data

The user chose Address required changes after the follow-up review. Scope: IT-010 / AT-004, AT-009–010, REG-003–004; FR-003, FR-010–011, NFR-001/004, RULE-002; AC-004, AC-010–011, AC-013. First test: deterministic fresh safe data after handle-to-DID navigation, with default-to-explicit Posts as the unchanged-effective-key control. Then retain the initial SSR/no-duplicate-read, pagination/history and member regression gates. No backend/native, PDS, backfill, migration, publication or commit work is included.

Status: completed. The original profile snapshot was keyed only by DID/tab/cursor; the shared component intentionally consumes that snapshot once per mount. The correction aligns its lifetime with newly accepted loader results.
- Red: both permanent fresh-loader cases failed on the updated-heading assertion, after verifying the response contained updated safe content and omitted the excluded quote. Same DID/tab/cursor retained the original component snapshot.
- Minimum implementation: key the guest ProfileView by its accepted loader result rather than the three effective location values. Each new result initializes one coherent shared-component snapshot; existing component disposal cancels publication from old work and drops its tab/page caches. No component-internal/member state logic or extra fetch was added.
- Green: both new desktop cases passed (4.0s runner), including updated identity/body, removed quote, fresh continuation cursor and cold reload. Web check has zero errors/warnings; 374 units in 58 files and build pass. Private source maps prepared only.
- After focused green, final gates ran (results below): guest preview/Worker matrices, no-JS public preview/Worker and member social/profile regressions. Unchanged backend/native suites are not repeated for this one guest route-lifetime change, per the review handoff.

### IR-005 final verification and handoff

| Gate | Result |
|---|---|
| New same-profile regression cases | Both failed meaningfully before the change; both passed afterwards (desktop runner 4.0s) |
| Full guest preview | 84 passed, 35.3s |
| Full guest local Worker | 84 passed, 36.7s |
| Public no-JS preview / Worker | 10 passed each, 4.9s / 5.3s |
| Member social/detail/profile preview / Worker | 43 passed each, 20.4s / 22.8s |
| Web static analysis | Zero errors and warnings |
| Web units | 374 passed in 58 files |
| Web build | Passed; private source maps prepared, no upload |
| Formatting and diff | Scoped Prettier and `git diff --check` clean |

The two new cases verify fresh responses and visible content agree for both handle-to-DID navigation and default-to-explicit Posts navigation. They check replacement of identity/body, removal of an excluded quote, the new continuation cursor, and cold reload. Existing failure-retention, no-duplicate-initial-read, cursor/history and delayed old-response tests remain green in every guest project. Browser servers ran serially to avoid Miniflare contention.

The product change is one guest route key: each accepted loader result now owns its ProfileView lifetime. The reused shared component, member rendering, backend, Flutter, credentials and PDS command/projection paths are unchanged in this pass. Test fixture state is restricted to an isolated synthetic profile and explicitly reset. No backfill, background refresh, public API, Lexicon, migration or live data operation was added. No commit, push or deployment was performed.

IR-005 is implemented with passing relevant checks; `06-implementation-review.md` remains the independent follow-up review snapshot pending another review. Manual/device/Android/live-PDS gaps recorded earlier remain unverified; no new native or external-provider proof is claimed. Prior backend/native results were not rerun or relabelled as current-pass execution.

Per implement-tdd, stop after reading this handoff and offer implementation review, a manual comment, or no further action. Do not start review or publish automatically.
