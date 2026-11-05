# Requirements: Logged-out browsing

## 1. Initial Request

Allow people to explore CraftSky before registering or signing in on mobile and web. Reuse the existing UI for craft feeds, search, profiles, posts, comments and public information, hiding account-only elements. Replace the standalone sign-in landing page with a modal containing its existing content and an **Explore CraftSky** option. Show a persistent sign-in/register banner while logged out. Consolidate public browsing and web Open Graph (OG) data where practical. Add a Privacy setting matching Bluesky's request to hide an account from logged-out people, and research its PDS representation and preview consequences.

This document is the requirements stage only. It does not authorize implementation, migrations, publishing or deployment.

## 2. Current Codebase Findings

Inspected on 5 November 2026. Findings describe this checkout, not a live production verification.

### Repository evidence

- `app/lib/router/router.dart`: initial route is welcome; signed-out users are redirected away from content routes to welcome. Auth completion, onboarding, account eligibility and retained-account initialization have separate routing rules.
- `app/lib/auth/pages/welcome_page.dart` and `sign_in_page.dart`: existing registration, handle-based sign-in, provider handoff, validation, help and legal content. Sign-in also supports adding an account; this use must survive removing the guest landing page.
- `app/lib/settings/pages/privacy_page.dart`: currently contains error-reporting consent.
- `web-client/src/routes/+page.server.ts`: root redirects to `/feed` for selected accounts and `/sign-in` otherwise.
- `web-client/src/lib/server/auth/page.ts`: authenticated page loader sends guests and expired sessions to sign-in.
- `web-client/src/routes/profiles/[handleOrDid]/` and `posts/[did]/[rkey]/`: existing server-rendered public pages alongside an authenticated rendering branch. The public rendering uses separate `PostCard.svelte`, profile markup and `lib/public.ts` types; it does not offer the full member experience.
- `web-client/src/lib/server/api.ts`: public server fetches use `/v1/public/*`, omit credentials, constrain upstream origin/redirects/time/body size, and map unavailable resources safely.
- `web-client/src/lib/Metadata.svelte` and the public route pages generate titles, descriptions, canonical paths and OG images from public data.
- `web-client/src/hooks.server.ts`: content responses use `no-store`; noncanonical hosts and error responses use `noindex, nofollow`.
- `appview/internal/routes/routes_public_pages.go`: four existing guest resources: profile, profile posts, profile projects and individual post. Aggregate rate/concurrency limits bound anonymous work.
- `appview/internal/api/public_pages.go`: explicit public response allowlists intentionally omit member-only fields. Public routes do not become member views when cookies/tokens are supplied.
- `appview/internal/api/public_pages_store.go`: active CraftSky membership/lifecycle, moderation and image-safety checks apply. Warning-labelled content is conservatively excluded. Public post reads currently exclude replies; no guest thread, home-feed or search contract exists here.
- `appview/internal/routes/policy.go`: timeline, projects, search and related member resources have explicit access policies. Making the UI guest-capable alone is insufficient.
- `appview/internal/index/bluesky_profile.go`: both current profile projection paths omit persistence of the logged-out visibility self-label.
- `appview/internal/lexicon/schema/standard/app/bsky/actor/profile.json` and `com/atproto/label/defs.json`: vendored standard schemas already describe profile `labels` and `!no-unauthenticated`.
- `appview/internal/auth/scopes.go`: profile create/update authority already exists. No broader OAuth authority is implied by this feature.

### Constraints and principles

Read `atproto-craft-social-app-reference.md`, the API architecture spec at `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`, and the [product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit). Preserve user-owned data, AppView-mediated writes, server-side OAuth credentials, chronological feeds, business/sponsorship disclosure and no paid visibility advantage. Older reference/spec descriptions are historical where current repository behavior has superseded them.

### Bluesky research and observed behavior

- In a fresh incognito visit to [Bluesky home](https://bsky.app/), a Discover feed loaded behind a welcome prompt with Create account, Explore the app and Sign in. Selecting Explore dismissed the prompt, leaving public content and registration/sign-in controls.
- A public post permalink displayed its images, engagement counts, reply policy and a multi-author comment thread without signing in. Desktop registration/sign-in controls remained available. This is a behavioral reference, not a requirement to copy Bluesky's algorithmic Discover feed or every visible action.
- Subsequent like-button interaction and snapshot checks timed out (`Preview automation click timed out after 15000ms`; `Preview automation snapshot timed out after 15000ms`). Search interaction, mobile behavior and every authentication prompt were not verified live. No Bluesky account was signed into or changed. The inspection tab was closed.
- The [standard profile schema](https://github.com/bluesky-social/atproto/blob/main/lexicons/app/bsky/actor/profile.json) supports account self-labels on `app.bsky.actor.profile`, record key `self`. The [official moderation guide](https://github.com/bluesky-social/bsky-docs/blob/main/docs/advanced-guides/moderation.md) defines `!no-unauthenticated` as a global self-label for applications that respect logged-out hiding.
- Representation: `at://<did>/app.bsky.actor.profile/self`, with `labels: { "$type": "com.atproto.label.defs#selfLabels", "values": [{ "val": "!no-unauthenticated" }] }`. Other labels and profile fields must be preserved.
- Bluesky's [current web renderer](https://github.com/bluesky-social/social-app/blob/main/bskyweb/cmd/bskyweb/server.go) checks the author label before populating post content into metadata. Its restricted branch omits post text/media, supplies minimal auth-required preview data and sets noindex/nofollow; it can retain author naming. This is source inspection, not proof that every third-party preview provider has refreshed its cache.
- The label requests presentation restrictions; it does not make records/blobs private, revoke previously downloaded content, or force other clients to comply. CraftSky should enforce its own guest boundary on the server as well as in UI.

### Existing verification entry points (not run for this document)

- AppView: `just test` against the compose development dependencies; focused public-page, route-policy, profile/index and search suites.
- Flutter: `just app-analyze`, `just app-test`; existing router, welcome, privacy, feed and search tests; device journeys for iOS/Android.
- Svelte web: from `web-client/`, `npm run check`, `npm test`, `npm run build`, `npm run test:e2e`, `npm run test:e2e:app`. Existing public-page tests include SSR/metadata behavior.

## 3. Clarifying Questions And Decisions

### Q1: Should the visibility setting be shared with Bluesky or CraftSky-only?

Answer: **Use the shared Bluesky preference.**

Decision / implication: CraftSky reads and updates the same account self-label on the PDS. A change can affect Bluesky and other apps that honour it. Changes made elsewhere must propagate into CraftSky. No new CraftSky Lexicon field is proposed.

### Confirmed prompt decisions

- Both mobile and web support guest browsing, including comments and search.
- Reuse existing UI and hide unavailable account features.
- Home shows craft feeds, with existing sign-in content in an optional-exploration modal.
- Logged-out users retain a persistent sign-in/register CTA banner.

The user confirmed all section 20 assumptions and approved these requirements on 5 November 2026, with one scope amendment: **no backfill**. This excludes startup/deploy sweeps, historical replay, background catch-up, lazy guest-read repair and dedicated profile-refresh jobs for this feature. Normal incoming profile events and explicit owner settings operations may establish visibility; unknown existing profiles remain unavailable to guests until then.

## 4. Candidate Approaches

### Option A: Extend the existing clients and share AppView read policy

Summary: Adapt existing content/navigation components to anonymous viewer capabilities. Extend or consolidate current public read services, using one guest visibility policy for mobile, web and SSR/OG generation.

Pros: Meets the requested UI reuse; keeps moderation and visibility consistent; preserves server-rendered sharing and existing authentication flows.

Cons: Existing components/providers assume a member; public data contracts need fuller safe projections.

Risks: Accidental private-field exposure or hidden authors leaking through nested content if reuse is implemented without explicit guest contracts.

### Option B: Expand the separate public website and build separate mobile guest screens

Summary: Continue independent public components/endpoints and duplicate the needed mobile views.

Pros: Smaller initial changes to member UI assumptions.

Cons: Duplicates profiles, threads, project fields and search; likely divergence from the requested experience.

Risks: Repeated privacy fixes and inconsistent OG/client behavior.

## 5. Recommended Direction

Use Option A, matching the user's UI preference. Reuse rendering and public read logic while retaining explicit anonymous capabilities and field allowlists. AppView owns guest eligibility; web SSR and preview generation consume that same eligibility decision. Reusing services does not require giving anonymous requests access to member response payloads.

Keep exact route organization as a coding-plan decision: extending `/v1/public/*` or consolidating read handlers is acceptable if all criteria below hold. Do not discard existing public safety controls or SSR/OG support during consolidation. Use the standard shared profile self-label, with an indexed projection populated by normal incoming profile events and explicit owner settings operations. No backfill or read-triggered repair is included.

## 6. Problem / Opportunity

An account wall prevents prospective crafters from evaluating the community and following shared links naturally. Separate public views currently provide only part of the content experience. Opening browsing must also give authors a meaningful, consistently honoured choice about logged-out presentation.

## 7. Goals

- G-001: Let guests discover crafts and read conversations on supported mobile and web clients.
- G-002: Offer clear registration/sign-in paths without repeatedly obstructing exploration.
- G-003: Honour the shared author visibility preference across every CraftSky guest surface and metadata response.
- G-004: Reduce duplicated public rendering/data policy while preserving privacy and existing member behavior.

## 8. Non-Goals

- NG-001: Anonymous publishing, likes, follows, reposts, bookmarks, messaging or account changes.
- NG-002: Private PDS records, private accounts, authenticated scraping prevention or removing content already retained elsewhere.
- NG-003: Algorithmic feeds, ads, monetization changes or expansion to non-CraftSky content collections.
- NG-004: New desktop-native clients, a broad UI redesign, custom visibility Lexicons, new reporting/moderation systems or production deployment.
- NG-005: New public features absent from the current product solely because Bluesky has them.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Guest | Person without an active CraftSky session | Discover, search, read, share and choose whether to join |
| Author/member | Authenticated CraftSky account owner | Control logged-out presentation and retain existing account functionality |
| Returning member | Person signing in from guest content | Complete existing OAuth/onboarding and return to context |
| Preview/search crawler | Anonymous consumer of web pages | Safe eligible metadata without account credentials |
| Other AT Protocol app | Changes or reads the shared profile | Interoperate through the existing standard self-label |

## 10. Current Behavior

Mobile redirects guests to welcome. Web home redirects guests to sign-in; public profile/post permalinks use reduced separate views and public responses. Guest comments, craft feeds and search are not provided by those public contracts. The shared visibility label is not retained by the inspected profile indexer, and Privacy does not expose it.

## 11. Desired Behavior

A guest opens home into the craft-feed shell and can choose Explore CraftSky from the welcome/sign-in modal. Afterwards they browse chronological craft feeds, search, profiles, posts, project details, media and comments using familiar UI. A persistent banner offers registration/sign-in. Public permalinks open the requested content directly. Account-only navigation/actions are hidden, with protected deep links offering authentication.

An authenticated owner can ask apps and sites to hide their account from logged-out people. CraftSky honours that shared preference in listings, detail views, nested content and link metadata; it continues to show eligible content to signed-in viewers subject to normal rules.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | People shall explore eligible CraftSky content without an account on iOS, Android and web. | Evaluate community before joining | Prompt | AC-001, AC-003, AC-004 |
| BR-002 | Business | Must | Account owners shall control logged-out presentation through the shared Bluesky preference. | User choice and interoperability | Q1 | AC-008, AC-009, AC-010 |
| FR-001 | Functional | Must | Guest home shall show selectable craft feeds using existing feed ordering and primary-craft eligibility, with loading, empty, retry and pagination states. It shall not require followed accounts or saved member feed preferences. | Useful entry point | Prompt; codebase | AC-001, AC-012 |
| FR-002 | Functional | Must | Guests shall search public people, posts, projects and hashtags using existing applicable filters, suggestions and result navigation, with existing sort semantics. Member recent searches and saved filters shall not be fetched or written. | Public discovery | Prompt; codebase | AC-003, AC-006 |
| FR-003 | Functional | Must | Guests shall read eligible profiles, posts, project fields, comments/replies and media, including existing public disclosures, timestamps, links, counts and read-only profile presentation. Supported public detail links shall work on cold start and reload. | Full public reading experience | Prompt | AC-004, AC-005 |
| FR-004 | Functional | Must | Guest home shall offer a modal containing the existing welcome/sign-in content and flows plus Explore CraftSky; exploring dismisses it without authentication. The standalone guest sign-in landing page shall be replaced by this flow, with old auth entry links opening the equivalent modal. | Requested entry experience | Prompt | AC-002 |
| FR-005 | Functional | Must | A persistent banner shall offer Sign in and Register throughout guest browsing, opening the same authentication flow without discarding the current public destination. | Join when ready | Prompt | AC-002, AC-007 |
| FR-006 | Functional | Must | Account-only navigation and mutation controls shall be hidden for guests. Public reading, opening threads/media, copying links and platform sharing remain available. Protected URLs shall offer authentication without exposing member data. | Reuse UI safely | Prompt | AC-005, AC-006 |
| FR-007 | Functional | Must | Successful sign-in shall restore the public destination after any required onboarding; cancellation/failure leaves exploration available. Logout/session expiry shall clear member state and establish a guest or retained-account context under existing account-switch rules. | Consistent auth transitions | Codebase; inferred UX need | AC-007 |
| FR-008 | Functional | Must | Privacy settings on both clients shall read and update the shared opt-out and explain cross-app effects, public-network limits and preview consequences. Failed saves shall not appear successful. | Informed author control | Prompt; Q1 | AC-008, AC-014 |
| FR-009 | Functional | Must | Adding/removing only `!no-unauthenticated` in the owner's standard PDS profile shall preserve unrelated fields, other labels and concurrent edits. Standard profile changes arriving from other apps shall update guest eligibility. | Interoperability without data loss | Q1; standard schema | AC-009, AC-010 |
| FR-010 | Functional | Must | Guest eligibility shall be applied consistently to home, search/suggestions, profiles, posts, replies, nested quotes/reposts, public relationship/activity views where present, and all public web/OG data. | Avoid alternate disclosure paths | Prompt; privacy research | AC-010, AC-011 |
| FR-011 | Functional | Must | Eligible public profile/post pages shall retain server-rendered readable content and content-specific OG metadata. Opted-out content shall produce a generic sign-in-required page/preview without author biography, avatar, post/project text or media, and shall be noindex. | Preserve sharing while honouring preference | Prompt; renderer research; proposed conservative default | AC-011 |
| NFR-001 | Non-functional | Must | Guest screens shall reuse existing client content components/presentation and shared backend eligibility/data services where practical, with explicit guest-safe response contracts. | Prevent divergent public implementations | Prompt | AC-013 |
| NFR-002 | Non-functional | Must | Anonymous work shall remain bounded by server-side rate/concurrency, query/result/body and pagination limits, with safe 429/503 handling and the existing `/v1/` JSON/error/cursor conventions. | Public-service reliability | Existing public API; API spec | AC-012 |
| NFR-003 | Non-functional | Must | Mobile/web guest navigation, modal and banner shall support screen readers, keyboard/focus where applicable, safe areas, text scaling, responsive layouts and existing reduced-motion behavior. | Usable shared experience | Existing UI conventions | AC-015 |
| NFR-004 | Non-functional | Must | Guest and member caches/state shall remain isolated. New responses shall not retain excluded content in CraftSky-controlled caches after the eligibility change has been applied. SSR/hydration payloads shall obey guest rules. | Avoid stale or cross-account disclosure | Privacy; existing no-store | AC-007, AC-010, AC-011 |
| RULE-001 | Business rule | Must | No active session grants guest read capability only; no synthetic member identity or OAuth credentials shall be required for browsing. Writes/private resources shall retain existing authentication/authorization. | Preserve trust boundary | Architecture | AC-006 |
| RULE-002 | Business rule | Must | An indexed opt-out shall exclude the account and its authored content from guest discovery and substitute a safe auth-required/unavailable state on direct access. Every nested author is evaluated separately; visible neighbours may remain. | Account-wide presentation choice | Prompt; Q1 | AC-010 |
| RULE-003 | Business rule | Must | Opt-out is a presentation request, not private storage. Authenticated visibility remains subject to existing moderation, lifecycle, block and eligibility rules; guests cannot override those rules. | Accurate privacy promise | Official moderation docs; codebase | AC-008, AC-016 |
| RULE-004 | Business rule | Must | No backfill, historical replay, catch-up sweep or guest-read repair shall be added or run. Unverified or unreadable preferences, including legacy profiles, shall stay unavailable to guests until normal incoming profile ingestion or an explicit owner settings operation establishes visibility. Own successful saves shall apply locally before success is acknowledged; delayed pre-save events shall not undo that newer decision. Normal incoming authoritative records shall establish unknown visibility even when their CID already matches the stored legacy profile; repeated established events remain idempotent. External edits converge after normal ingestion. | Safe rollout and propagation | Indexer finding | AC-009, AC-010, AC-017 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given no active session on each supported client, opening home and choosing Explore displays craft feeds; switching crafts and loading later pages preserves existing order/eligibility and requires no account. |
| AC-002 | FR-004, FR-005 | Given a new guest browsing session, home offers current welcome/sign-in content plus Explore CraftSky. Explore dismisses the modal. Navigating within that session does not reopen it automatically. The persistent Sign in/Register banner remains available, including on direct content entry, and legacy sign-in links open the equivalent flow. |
| AC-003 | BR-001, FR-002 | Given eligible and excluded fixture authors/content, guest searches for a person, keyword, project and hashtag return only eligible results; applicable filters, suggestions, sort selection and pagination work without member search-history requests. |
| AC-004 | BR-001, FR-003 | Given an eligible profile/post/reply permalink, a signed-out cold start or web reload shows the requested public profile, project fields or thread, including eligible comments and media. Public business/sponsorship labels remain visible. The welcome modal does not automatically obscure direct content entry. |
| AC-005 | FR-003, FR-006 | Given a guest reading content, opening a comment thread, public profile section, image/gallery/video, external link, Copy link or platform share works. Account mutation controls and member-only navigation are absent; public counts remain read-only. |
| AC-006 | FR-002, FR-006, RULE-001 | Given direct anonymous API calls or protected client URLs, writes/private data remain protected; no drafts, saved searches/filters, bookmarks, chat, notifications, owner-only fields or credentials enter guest responses/state. Guest browsing works without a member token. |
| AC-007 | FR-005, FR-007, NFR-004 | Given a guest on a public destination, sign-in/register completes through existing OAuth/onboarding and returns there. Cancellation/error permits continued browsing. Logout/expiry/account switching clears prior viewer data before guest rendering and never reveals previous-account caches. No attempted mutation is silently replayed after sign-in. |
| AC-008 | BR-002, FR-008, RULE-003 | Given authenticated Privacy settings, both clients display the current shared preference, cross-app effect and public-network explanation; turning it on/off saves and survives reopening. Text explains that content remains public on the network and previews may remain cached elsewhere. |
| AC-009 | BR-002, FR-009, RULE-004 | Given an existing standard profile with other labels/unknown fields, enabling adds only the target label and disabling removes only it. A competing profile edit is preserved or produces a recoverable conflict. PDS/local persistence failure does not display completed success. Successful enablement already suppresses local guest reads when acknowledged; delivering a delayed pre-save profile event afterwards does not restore guest visibility. |
| AC-010 | BR-002, FR-009, FR-010, NFR-004, RULE-002, RULE-004 | Given an opt-out applied locally or received through profile ingestion, guest lists, suggestions, detail data, nested replies/quotes/reposts and existing public relationship/activity surfaces omit the author's protected presentation. Other eligible participants remain readable without embedded excluded content. Disabling restores only otherwise eligible content after convergence. New requests and refreshed pages use the updated decision. |
| AC-011 | FR-010, FR-011, NFR-004 | Given eligible versus opted-out post/profile URLs, requests without JavaScript and crawler requests obey the same eligibility. Eligible pages retain readable SSR and specific OG data; excluded pages expose no protected content in HTML, JSON/hydration, descriptions, images, structured data or preload URLs, show only generic authentication messaging/branding, and use noindex. Anonymous requests never inherit member privileges from forwarded credentials. |
| AC-012 | FR-001, NFR-002 | Given empty data, hidden-only pages, malformed/expired cursors, dependency failure, throttling and concurrent load, clients provide usable empty/retry/pagination behavior and bounded API responses; filtering occurs before page limits and cannot truncate discovery prematurely or bypass limits. |
| AC-013 | NFR-001 | Given equivalent eligible content in member and guest modes, shared components preserve project fields, disclosure and media presentation. Review demonstrates shared eligibility for clients and metadata and explicit guest field allowlists; no parallel full public UI implementation is introduced. |
| AC-014 | FR-008 | Given unavailable settings reads, failed/conflicting saves or a required OAuth reauthorization, the setting is not falsely shown as saved or defaulted to public; the user receives retry/recovery behavior and other Privacy settings continue to work. |
| AC-015 | NFR-003 | Given keyboard/screen reader access, enlarged text, reduced motion and narrow/wide viewports, modal focus is contained/restored, dismissal is accessible, banner CTAs remain reachable, and neither banner nor modal hides required reading/navigation controls. Validate iOS, Android and responsive web. |
| AC-016 | RULE-003 | Given suspended/deleting/nonmember or moderation-restricted fixtures, guest feeds, search, threads, profiles and previews preserve the current conservative public exclusions and image-safety policy. A missing member identity cannot enable an owner bypass. Signing in does not override ordinary member eligibility rules. |
| AC-017 | RULE-004 | Given pre-existing profiles whose labels were never indexed, migration/startup/guest reads perform no backfill, replay or repair requests and leave them unavailable to guests. Normal incoming profile events or explicit owner settings operations can establish visibility. For both profile projection paths, a normally delivered authoritative event establishes unknown visibility even if its CID equals the stored legacy CID, without fetching or replaying records. Repeated established events remain idempotent; replacement/deletion and unavailable source reads leave a deterministic safe state. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Opted-out commenter under an eligible root | Omit protected reply presentation; keep eligible replies readable without leaking parent excerpts | FR-010, RULE-002 |
| EC-002 | Eligible quote/repost references an opted-out author | Eligible outer commentary may remain; protected embed/author/media are unavailable | FR-010, RULE-002 |
| EC-003 | Opted-out root with eligible replies | Direct root uses safe auth-required state; independently addressed eligible replies may render with unavailable ancestor context | FR-003, RULE-002 |
| EC-004 | All candidates on a result page are hidden | Continue bounded eligible pagination or return a genuine safe empty/end state | FR-001, FR-002, NFR-002 |
| EC-005 | External profile edit or rapid enable/disable | Apply authoritative record ordering under existing ingestion rules; stale events must not undo newer visibility | FR-009, RULE-004 |
| EC-006 | Missing/deleted/malformed profile preference | Preserve account lifecycle rules; use verified-absence versus unknown distinction; unknown fails closed | RULE-003, RULE-004 |
| EC-007 | Existing sign-in/add-account/callback deep link | Reach correct authentication flow; preserve callback security and retained-account behavior | FR-004, FR-007 |
| EC-008 | Third-party cached preview or direct PDS/blob URL | Explain that CraftSky cannot revoke external copies or make public blobs private | FR-008, RULE-003 |
| EC-009 | Session expires while private screen is open | Clear private viewer data and offer safe guest/authentication destination; do not render stale member screen | FR-007, NFR-004 |
| EC-010 | Search suggestion/count derived only from hidden content | Do not use excluded content to expose guest snippets, identities or result suggestions; aggregate count policy is recorded below | FR-002, FR-010 |

## 15. Data / Persistence Impact

- New fields: indexed verified visibility state/provenance for the standard profile preference; exact storage design deferred.
- Changed fields: standard PDS `app.bsky.actor.profile/self.labels` only for explicit owner updates; no new `social.craftsky.*` record shape.
- Migration required: likely an AppView schema migration to represent unknown versus established visibility. **No backfill**, including SQL-derived inferred opt-in, historical replay, startup/background sweeps or lazy guest-read repair. Existing unknown profiles remain unavailable to guests until normal incoming profile events or explicit owner settings operations establish visibility. No migrations are run in this stage.
- Backwards compatibility: shipped app compatibility is not required under AGENTS.md, but existing public URLs, OAuth callbacks, owner data and operational safeguards remain acceptance constraints. No production release actions are authorized here.
- Concurrent edits: preserve unknown fields and other labels; conflict handling must use the existing PDS mutation conventions. The coding plan must identify the actual owner-save/ingestion ordering boundary and partial-success recovery. CIDs identify content and must not be treated as sortable versions. A delayed pre-save event cannot undo an acknowledged newer local visibility decision (DR-002).
- Normal-event initialization: both profile projection paths must distinguish an unknown projection from an established one even when the stored CID is unchanged. Establishing visibility from a normally delivered authoritative record is permitted; requesting historical delivery or fetching profiles to fill unknown state is excluded (DR-001).

## 16. UI / API / CLI Impact

- UI: mobile and Svelte guest shell, craft selection/search, shared content/thread/profile rendering, auth modal, persistent banner and Privacy toggle. Flutter web remains a supported app target; parity coverage includes its guest routing if shipped alongside Svelte.
- API: safe anonymous feed/search/thread/public-detail data; shared eligibility for content and SSR/OG. Exact endpoints/DTOs deferred to coding plan. Private/write access policies stay explicit. Keep `/v1/`, camelCase, standard errors and opaque bounded cursors.
- CLI: no new CLI or backfill mechanism is required or authorized.
- Background jobs: existing profile ingestion must retain labels and converge external changes. New recurring jobs are not assumed.

### Public reading inventory (DR-004)

This inventory makes the approved “all public information” scope explicit. It records current reading surfaces, not new guest endpoint names. Existing member routes are evidence of behavior, not permission to expose their payloads unchanged. The coding plan maps each row to a guest-safe contract, shared components and tests.

| Existing surface / evidence | Guest behavior and boundary | Requirement / AC IDs |
|---|---|---|
| Ordinary profile tabs: Projects, Posts, Comments, Reposts; Flutter `ProfileTabPolicy`, Svelte `social/Profile.svelte` | Retain each eligible public tab and pagination; independently filter authors in comments/reposts/embeds. No owner management controls. | FR-003, FR-010; AC-004–005, AC-010 |
| Business tabs: Products, Upcoming events, About, plus ordinary tabs | Preserve existing public business details and conditional tab availability; empty owner-only management states do not become guest controls. | FR-003, FR-006; AC-004–005 |
| Profile identity, links, customization, disclosures and public counts | Preserve public presentation; opted-out profiles use the approved generic state. Exclude viewer relationship flags, mutual-follower personalization and owner analytics. Existing public identity lists, where present, filter excluded authors; allowed aggregate totals can still include opted-out participants (ASM-007). Do not add new graph screens. | FR-003, FR-006, FR-010; AC-005–006, AC-010 |
| General/project detail, comment thread, quoted/reposted content; `social/MemberPost.svelte`, `Thread.svelte`, `composer/ProjectDetail.svelte` and Flutter equivalents | Keep public structured fields, rich text, reply policy, media/alt text and disclosures. Hide creation/interaction controls while retaining thread reading, copy and platform share. Evaluate every nested author. | FR-003, FR-006, FR-010; AC-004–006, AC-010 |
| Search categories: people, posts, projects, hashtags; `social/Search.svelte` | Preserve query/results, suggestions, clear, pagination and category navigation without viewer history. Apply existing public filters/sorts where that client exposes them; do not invent new controls to equalize clients. | FR-002; AC-003, AC-006 |
| Blank search/recent profiles/top hashtags; existing Search load path | Show eligible public discovery data. Omit member recent-search fetch/save/delete and saved-filter requests, including effects triggered just by typing or opening a result. | FR-002, FR-010; AC-003, AC-006, AC-010 |
| Craft/project filter screen and hashtag results; `social/ProjectFiltersPage.svelte`, API search/browse methods, Flutter search equivalents | Retain applicable existing transient craft filters and sort semantics; no account-specific saved preferences. Filter excluded content before result bounds. | FR-001–002, NFR-002; AC-001, AC-003, AC-012 |
| Private/owner surfaces: saved posts/folders, drafts, chat, notifications, recent searches, follower-growth and mutual-follow personalization | Remain authenticated; hide guest entry controls and protect direct/API access. No inferred public access merely because the surrounding profile is public. | FR-006, RULE-001; AC-006 |

Detailed field/route mappings and the reusable component dependency audit remain coding-plan deliverables. This table adds no new product surfaces.

### Proposed Privacy copy

Label: **Ask apps and sites not to show my account to logged-out users**

Explanation: “CraftSky will hide your account and posts from people who aren't signed in. This uses the same setting as Bluesky, so changing it here also changes your Bluesky setting. Other apps may not honour this request. It doesn't make your account private.”

Network note: “CraftSky is part of the Atmosphere, an open public network. This setting asks apps and websites to limit your visibility, but they can choose not to. Your public content may still appear elsewhere.”

Preview help: “CraftSky won't include your profile or post content in new link previews while this is on. Other sites may keep previews they already saved.”

This adapts the user-supplied screenshot, with explicit shared-setting consequences. Only the logged-out visibility control is in scope, not the screenshot's algorithmic-recommendation or other settings.

## 17. Security / Privacy / Permissions

- Authentication: guest reading requires no OAuth or member identity; existing authenticated PDS write mediation remains mandatory.
- Authorization: opt-out and public moderation/lifecycle filtering are server decisions, applied to every nested subject and metadata serialization, not just hidden controls.
- Sensitive data: explicit public response allowlists; no member relationship flags, private intent, search history, notifications, drafts, chat, credentials or capabilities in guest payloads.
- Abuse cases: pagination/filter bypass, suggestions, quoted content, alternate route paths, spoofed viewer headers, SSR/JSON/preload leakage, uninitialized legacy profiles and stale viewer caches are test targets.
- Scope: shared profile write capability already exists. Implementation must verify its actual supported read/modify/write path without broadening scopes by default.
- No privacy guarantee is made about independent PDS readers or downloaded media. No PDS records are deleted to enforce guest hiding.

## 18. Observability

No new analytics events, alerts or tracking requirements are introduced. Use existing classified route failures, bounded dependency/ingestion diagnostics and rate-limit signals to diagnose guest failures. Any diagnostic implementation must follow `docs/development/logging-error-reporting.md`, including privacy selection at source, one issue owner, preserved typed causes and serialized sink canaries. Never log profile record bodies, search text, tokens or private viewer state to demonstrate this feature.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Legacy profiles have no indexed preference and no backfill is permitted | Legacy content remains unavailable to guests until a normal event or explicit owner settings operation establishes visibility | Preserve unknown state and fail closed; index both paths without adding catch-up jobs |
| RISK-002 | Hidden content leaks through nested entities or OG/SSR | Preference violated despite hidden UI | Shared eligibility, safe serialization and canary tests across all representations |
| RISK-003 | Shared toggle surprises Bluesky users or overwrites profile edits | Cross-app behavior/data loss | Explicit copy; preserve unrelated data/labels; concurrency handling |
| RISK-004 | Member-centric state reused in guests | Private state or owner privileges exposed | Explicit capability contracts, isolation and auth-transition tests |
| RISK-005 | External previews/copies remain | Users infer privacy guarantees the system cannot provide | Honest explanation; suppress new CraftSky content previews; preserve no-store |
| RISK-006 | Public search/feed load bypasses current bounded routes | Capacity or availability degradation | Retain admission limits and bounded queries; verify with representative pagination/load |
| RISK-007 | Guest availability conflicts with current age/content safety gates | Unsafe content becomes newly reachable | Preserve conservative existing public policy; review all guest admission routes before implementation |

## 20. Assumptions

All assumptions ASM-001–ASM-009 were confirmed by the user on 5 November 2026. The later no-backfill instruction is incorporated in RULE-004 and AC-017.

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Web means the current Svelte client; the supported Flutter web app must also retain guest-capable behavior if distributed. | Adjust platform test matrix before implementation |
| ASM-002 | Automatically offer the welcome modal once per guest browsing session; Explore prevents repeat prompts in that session. Direct profile/post links open unobstructed. | Change modal recurrence/entry rules and AC-002/004 |
| ASM-003 | Guest default is the existing unpersonalized all-crafts view, with all available craft feeds selectable; no new ranking or account-based recommendations. | Specify initial craft/order before test design |
| ASM-004 | “All public information” includes existing public profile sections, project details, media, reply policy and read-only social counts/relationships where the product exposes them; it excludes private-by-intent and account-only features. New social graph screens are not required. | Expand or reduce route/field inventory before test design |
| ASM-005 | Guest controls for account mutations are hidden, as requested. Read-only counts can remain; authentication is offered through banner/protected URLs. No automatic mutation replay. | Change action-to-sign-in behavior |
| ASM-006 | Restricted direct pages/previews use generic CraftSky branding without author identity, except identifiers already in the requested URL. This is more conservative than the inspected Bluesky preview naming. | Decide whether minimal handle/display name may be shown |
| ASM-007 | Aggregate public engagement totals may include opted-out participants, but their identities, snippets and authored content are excluded. This setting does not rewrite historical public totals or third-party text mentioning a person. | Recompute totals or add identity/text-redaction rules |
| ASM-008 | Existing conservative guest moderation policy remains the baseline; this change does not introduce click-through warning or age-verification policies. | Separate product/safety decisions required |
| ASM-009 | External preference updates take effect after normal ingestion; no instantaneous cross-network propagation or live erasure of already rendered pages is promised. New requests/refreshes after application use current eligibility. | Define stronger propagation SLA/live refresh behavior |

## 21. Open Questions

No unresolved blocking product questions. Q1 and all assumptions are confirmed; requirements are approved with backfill explicitly excluded.

- Non-blocking design follow-up: complete the field/route/component mappings for the public reading inventory in section 16; keep all genuinely public existing reading paths available (DR-004).
- Non-blocking design follow-up: choose route consolidation and normal-event visibility projection, including same-CID unknown state in both indexer paths, owner-save/ingestion ordering, missing-profile semantics and partial-success recovery (DR-001–002). Do not introduce a backfill mechanism.
- Non-blocking validation follow-up: adopt a JS-enabled guest test project alongside the existing no-JS SSR checks, with explicit runner selection and fixture/browser coverage as specified in the revised test design (DR-003).
- Non-blocking validation follow-up: complete live Bluesky search/action/mobile comparisons if useful during UI design; current browser observations are limited as recorded above.
- Non-blocking operational follow-up: define measurable external-ingestion convergence monitoring based on existing pipeline behavior; do not invent a latency guarantee at requirements stage.

## 22. Review Status

Status: **Approved — with no-backfill amendment**

Risk level: **High** (public exposure, shared cross-app preference and metadata/caching boundaries).

Review recommended: **Required; requirements approval received.** Test design is authorized. Implementation is not yet authorized.

Reviewer: User (product requirements approval); Codex (document review: Approved with notes).

Date: 2026-11-05.

Notes: User confirmed the shared preference and all assumptions, explicitly excluded backfill, approved requirements and requested test design. Following document review, the user authorized revisions to both documents as needed. DR-001–004 are incorporated as clarifications and coding-plan handoffs, with all existing IDs and product scope preserved. The review artifact records the pre-revision review; these edits are not a new independent review. No code/tests/dependencies/migrations/deployments changed or executed for this artifact.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-11-05-logged-out-browsing/01-requirements.md`.
- Next test specification: `02-acceptance-tests.md`, using `write-acceptance-tests` after explicit requirements approval.
- Must-cover requirement IDs: BR-001–002, FR-001–011, NFR-001–004, RULE-001–004; acceptance criteria AC-001–017.
- Suggested levels: policy/serialization units; real-Postgres feed/search/thread/index integration and no-backfill verification; PDS mutation/conflict fixtures; Flutter widget/router/provider tests; Svelte SSR/metadata and browser guest/member transitions; mobile device journeys and accessibility checks.
- Include positive eligible-content cases and protected-value canaries in JSON, HTML, metadata and caches. Cover cross-app opt-out ingestion, duplicates/reordering, preserved labels, unknown legacy state, nested authors and failures.
- Existing member behavior must retain coverage, especially multi-account isolation, onboarding/callbacks, moderation, project detail/media and Privacy consent.
- Blocking open questions: none. Requirements approval was received and test design is authorized; implementation has not started.
