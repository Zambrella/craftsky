# Coding Plan: Logged-out browsing

## 1. Inputs

Revision on 6 November 2026: basic submitted-query guest search; no recent users, autocomplete/suggestions or top hashtags. Craft project feed filters/sort controls remain. This supersedes the corresponding earlier search/discovery design without changing IDs or no-backfill.

- Requirements: `01-requirements.md`, approved with no-backfill amendment; subsequently clarified at the user's request for DR-001–004.
- Tests: `02-acceptance-tests.md`, revised for the same review notes; 43 planned tests/checks, not executed.
- Document review: `03-document-review.md`, **Approved with notes**. It records the pre-revision review; this plan resolves its remaining design handoffs.
- Authorization: user requested coding planning. Implementation, test code, migrations, commits, pushes and live actions are not authorized by this document.
- Constraints: checked-in `AGENTS.md`, previously read architecture reference/product vision, `/v1/` API architecture and mandatory logging/error-reporting guide. Flutter targets iOS/Android only. No custom Lexicon change is planned.
- Read-only inspection: profile/index and transactional projection, ingestion source ordering/duplicate handling, PDS addressed/compound commands and metadata transport, public route/store contracts, member query/hydration boundaries, Flutter repositories/providers/account boundaries/home shell, and Svelte route loaders/shell/API/social views/test configs.

## 2. Implementation Strategy

Extend the current anonymous `/v1/public/*` API, sharing underlying read/query/hydration logic with member reads through an explicit audience policy. Keep member routes protected. Public responses remain explicit allowlists, including nested objects. Both native Flutter and Svelte consume that public contract; Svelte SSR/OG uses the same contract without forwarding credentials.

Reuse existing home, search, profile, post, project, thread and media components. Extract audience-neutral presentation from member runtime initialization where necessary. A guest is an explicit viewer mode with read capability, never a synthetic account or empty-string owner. Member mutations continue through the existing command/repository paths.

Store a nullable, revision-fenced guest-visibility projection alongside the existing Bluesky profile row. Null/missing/invalid is unknown and guest-hidden. Populate it only from normally delivered authoritative profile events or explicit owner Privacy operations. There is **no backfill**, startup/deploy population, historical replay, projection-version sweep, PDS profile repair on guest reads or new catch-up worker.

Use the standard profile `!no-unauthenticated` self-label. Privacy saves use the durable addressed-command service for this one record, with compare-and-swap, safe read/modify/write, commit/observation evidence and transactional local acceptance. Do not use a two-record compound profile update merely to save one visibility label.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Visibility projection | Bluesky profile row and two indexer paths; transactional ingestion | Nullable visibility and revision floor; same-CID unknown initialization; normal-event ordering and duplicate delivery | BR-002, FR-009, RULE-004 | IT-001, IT-006, IT-012 |
| Owner settings | Fenced PDS commands, record merge and typed API errors | Read/save shared label; durable acceptance/unchanged-observation hook; partial-success handling | BR-002, FR-008, FR-009, RULE-004 | AT-008, AT-009, UT-006, UT-007, IT-005, IT-007, REG-006 |
| Anonymous read services | Public DTOs, conservative visibility and admission controls | Expand safe reads and share audience-aware query/hydration helpers | BR-001, FR-001, FR-002, FR-003, FR-010, RULE-001, RULE-002, RULE-003 | UT-001, IT-002, IT-003, IT-004, IT-008, IT-009, IT-011 |
| Flutter browsing | Riverpod repositories, account-operation guards, shared widgets and GoRouter | Guest read context/guard, guest repository selection, common chrome/modal/banner and controls | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, NFR-001, NFR-003, NFR-004 | AT-001, AT-002, AT-003, AT-004, AT-005, AT-006, AT-007, AT-012, UT-002, UT-003, UT-004, UT-005 |
| Svelte browsing and SSR | AppShell, social views, public server fetch and metadata | Separate member runtime from common shell; guest API; retained SSR/OG; safe auth return | FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-010, FR-011, NFR-001, NFR-004 | AT-001, AT-002, AT-003, AT-004, AT-005, AT-006, AT-007, AT-010, UT-008, IT-010, REG-003 |
| Bounds, regressions and evidence | Go tests/Postgres, Flutter tests, Vitest/Playwright | Explicit JS-enabled guest runner alongside no-JS SSR; native/manual validation | NFR-001, NFR-002, NFR-003, NFR-004, RULE-003 | AT-011, AT-012, UT-009, REG-001, REG-002, REG-004, REG-005, REG-007, MAN-001, MAN-002, MAN-003 |

## 4. Files And Modules

New names below are planned. Reuse existing module boundaries; split files by responsibility, not one file per endpoint. Generated Riverpod/router files are regenerated only during authorized implementation, never edited by hand.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/<next>_profile_guest_visibility.{up,down}.sql` | Create | Nullable columns/checks only; no data population; verify current next number | RULE-004, NFR-004 | IT-001, IT-012 |
| `appview/internal/guestvisibility/` and `appview/queries/guest_visibility.sql` | Create | Label extraction, explicit evidence type, ordering reducer, SQL persistence and guest eligibility | FR-009, FR-010, RULE-002, RULE-004 | UT-001, UT-007, IT-005, IT-006 |
| `appview/internal/index/bluesky_profile.go`, `transactional_projectors.go`; `appview/internal/ingestion/store.go` | Change | Both projection paths; source revision evidence; narrow duplicate-delivery path for unknown visibility | FR-009, RULE-004 | IT-006, IT-012 |
| `appview/internal/api/guest_visibility.go` | Create | Authenticated owner settings GET/PUT and command adapter | FR-008, FR-009 | AT-008, UT-006, IT-005, IT-007 |
| `appview/internal/pdscommands/addressed_command_service.go`, relevant store completion helpers | Change, narrowly | Transactional finalization on authoritative unchanged observations as well as write/recovery; keep default behavior for other commands | FR-009, RULE-004 | IT-005, REG-001 |
| `appview/internal/api/public_pages.go`, `public_pages_store.go`; new `public_reads.go`, `read_audience.go` | Change / Create | Safe public shapes and shared service/policy, thread/list/search/business reading | FR-001, FR-002, FR-003, FR-010, RULE-001, RULE-002 | IT-002, IT-003, IT-004, IT-008, IT-011 |
| Existing `search*_store.go`, `post_store.go`, thread/profile/business stores and response hydrators | Change | Factor audience predicates and safe nested hydration before limits; preserve member signatures through wrappers | FR-001, FR-002, FR-003, FR-010, NFR-001, NFR-002 | IT-002, IT-003, IT-004, IT-009, REG-004, REG-005, REG-007 |
| `appview/internal/routes/routes_public_pages.go`, `policy.go`, route inventory/tests; `appview/internal/app/deps.go` and adjacent composition | Change | Register expanded anonymous and owner settings routes, shared admission/services | FR-008, FR-010, RULE-001, NFR-002 | IT-007, IT-008, IT-009, IT-012 |
| `app/lib/shared/browsing/` | Create | Viewer/read lease, capabilities, common auth prompt/banner and public read adapters | FR-004, FR-005, FR-006, FR-007, NFR-004 | UT-002, UT-003, UT-005, AT-002, AT-007 |
| `app/lib/shared/api/providers/`, feature repository/API client providers under `feed/`, `profile/`, `projects/`, `search/` | Change | Credential-free guest transport and audience-aware read routing; preserve member mutation clients | FR-001, FR-002, FR-003, RULE-001 | UT-004, IT-008, AT-001, AT-003, AT-004 |
| Existing content providers; `app/lib/auth/providers/account_boundary_provider.dart` | Change | Read-lease guards instead of member-only captures on guest-capable reads; isolate caches/late results | FR-007, NFR-004 | UT-005, AT-007, REG-002 |
| `app/lib/router/{router.dart,app_shell.dart,app_shell_drawer.dart}`; `home/`, `feed/`, `profile/`, `search/` widgets | Change | Public route classification, all-crafts guest source, reusable content and hidden account-only chrome/actions | FR-001, FR-003, FR-004, FR-006, NFR-001, NFR-003 | AT-001, AT-004, AT-005, AT-012, REG-004 |
| `app/lib/auth/pages/{welcome_page.dart,sign_in_page.dart}` and auth-return state | Change | Extract reusable auth content; modal/Explore; retain add-account/callback handling | FR-004, FR-005, FR-007 | AT-002, AT-006, REG-001 |
| `app/lib/settings/pages/privacy_page.dart`; new settings API/repository/controller; l10n ARBs | Change / Create | Shared owner toggle and truthful pending/error states/copy | FR-008 | AT-008, UT-006, IT-007, REG-006 |
| `web-client/src/lib/api/{client.ts,types.ts}`; new `public-client.ts` and `browsing.ts` | Change / Create | Read-only client adapter, safe public types, discriminated viewer/capabilities | FR-003, FR-006, RULE-001 | UT-002, UT-004, IT-008 |
| `web-client/src/lib/features/shell/AppShell.svelte`; new `BrowsingShell.svelte`, `GuestRuntime.svelte` | Change / Create | Shared chrome; member runtime retained; guest avoids account initialization/side effects | FR-004, FR-005, FR-007, NFR-001, NFR-004 | AT-002, AT-007, AT-012 |
| `web-client/src/lib/features/social/{Feed,Search,Profile,Thread,MemberPost}.svelte`, business/event/detail components | Change | Shared reads/presentation with explicit capabilities, optional member context and no guest history writes | FR-001, FR-002, FR-003, FR-006, FR-010 | AT-001, AT-003, AT-004, AT-005, REG-004 |
| New reusable auth content/modal/banner; existing `routes/sign-in/+page.svelte`; Privacy settings component | Change / Create | Existing auth form and registered handoffs; Explore; owner toggle | FR-004, FR-005, FR-007, FR-008 | AT-002, AT-006, AT-008 |
| `web-client/src/routes/`, `lib/routes.ts`, route matcher, `lib/server/api.ts`, auth loader/start/complete/session logic | Change | Public route dispatch; bounded guest proxy; safe auth returns; public SSR data and metadata | FR-003, FR-007, FR-010, FR-011, NFR-004 | AT-004, AT-006, AT-010, IT-010, REG-003 |
| `web-client/tests/playwright-guest.config.ts`, `guest-*.spec.ts`, mock fixture utilities; existing no-JS/member suites | Create / Change | Runnable guest browser coverage; existing SSR/member regressions | NFR-001, NFR-003, NFR-004 | AT-001, AT-002, AT-003, AT-004, AT-005, AT-006, AT-007, AT-008, AT-009, AT-010, AT-011, AT-012 |

## 5. Services, Interfaces, And Data Flow

### 5.1 Visibility persistence and evidence (DR-001–002)

Add columns to `bluesky_profiles`, rather than a new owner-scoped table:

- `guest_visibility_hidden BOOLEAN NULL`: true=opted out, false=verified label absence, null=unknown. No default false and no UPDATE of legacy rows.
- `guest_visibility_revision TEXT NULL` and `guest_visibility_record_cid TEXT NULL`: ordering floor and associated record identity; revisions are parsed repository TIDs.
- `guest_visibility_evidence TEXT NULL`: closed provenance `source`, `writeReceipt`, `observation`; database constraints prevent incomplete known state.
- `guest_visibility_hold_key UUID NULL`: durable fail-closed hold for an explicitly pending owner settings operation. Its key is an existing operation UUID, not a capability or new job.

Guest eligibility requires a known false flag, no pending hold, active CraftSky ownership and all existing public moderation/image/lifecycle rules. Missing profile row is unknown. Reuse existing profile-row purge behavior; do not introduce a new DID inventory component. Preserve a revision floor through a profile delete by retaining a minimal unknown visibility row (clear presentation/links/media) until existing owner lifecycle purge removes it. A delayed older record cannot recreate guest visibility. Test existing profile-deletion behavior for member reads against that minimal row; it must still behave as no public profile presentation.

Evidence is an explicit input to the shared reducer, not inferred from a CID or wall clock:

```text
VisibilityEvidence = Source(recordCID, repoRevision)
                   | WriteReceipt(recordCID, commitCID, repoRevision)
                   | Observation(recordCID, observedHeadCID, observedHeadRevision)
ApplyTx(ctx, tx, ownerDID, parsedPreference, evidence) -> applied | stale | conflict
```

Compare repository TIDs for the same owner/record. Lower revisions cannot alter established state. Equal revision/equal content is idempotent and may initialize unknown state; equal revision/conflicting identity/content fails closed. A later revision can change the preference. An observation revision is a lower-bound fence against pre-observation events, **not** the original record write revision. Never sort CIDs, use Tap envelope IDs as repo revisions, or fabricate a source event from a local read.

Both `BlueskyProfile.Handle` paths extract the same standard self-label and use the reducer, with the transactional `Project` path passing `SourceRecord.Revision`. Do not gate the visibility update solely on the existing profile `record_cid IS DISTINCT FROM` clause. Separate visibility-column update from presentation/image reconciliation so the same CID can establish an unknown projection without churning image jobs.

Direct OAuth/onboarding `Handle` calls currently have no revision. They may update their existing presentation behavior, but must not overwrite established visibility using unversioned data. Default unknown remains hidden until a normally delivered versioned event or explicit owner Privacy operation. Do not add OAuth-time repair to populate guest visibility.

**Ingress duplicate delivery:** `ingestion.Store` currently returns a receipt for duplicate sources without scheduling projection. For a normally received, validated `app.bsky.actor.profile/self` duplicate whose visibility remains unknown, permit one bounded, subject-only visibility projection from that received source under existing transaction/lifecycle fences. Do not scan retained sources, bump global projection versions or enqueue historical delivery. Validate that source revision/identity still wins; stale/conflicting events never initialize public state. Guard this narrow path for absent/unknown visibility and nonterminal eligible source; after establishment ordinary duplicates remain no-ops. Extend IT-006 to the ingress boundary, not just direct indexer calls.

### 5.2 Owner Privacy API and command acceptance

Add authenticated current-member routes:

| Route | Contract | Access / rules |
|---|---|---|
| `GET /v1/preferences/logged-out-visibility` | `{ hideFromLoggedOut: boolean, expectedCid: string }` | Read exactly the owner's standard profile through the fenced PDS boundary; stable head/record/head observation. Can establish local visibility from explicit owner settings access. Unknown/failure is an error, not false. |
| `PUT /v1/preferences/logged-out-visibility` | `{ hideFromLoggedOut: boolean, expectedCid: string }`; returns same saved shape | Existing `Idempotency-Key` convention and addressed CAS command; owner/session comes only from auth. A 200 means both authoritative PDS outcome and local visibility fence are persisted. |

Use existing command errors/statuses, including conflicts, 202 outcome-unknown, reauth and rate limits; retain/retry the same immutable intent/key on ambiguous outcomes. Reject unknown request fields and malformed bodies under existing conventions. Verified record absence is handled deliberately: GET may return empty expected CID and false only after a stable absence observation; PUT creates the minimal standard profile with the target labels as appropriate, preserving any concurrent creation through CAS. Missing/unsupported authoritative evidence fails safely.

Create `GuestVisibilitySettingsService` in API/composition backed by `AddressedCommandService`, the owner lifecycle fence and the visibility reducer. Use `recordjson` and raw fields/precision-safe merging; add/remove only the target label, preserve all other fields/labels, and write only `app.bsky.actor.profile/self`. Existing general profile saves must preserve this label; their actual versioned events can update projection later.

Before dispatching the explicit operation, persist a matching hold under the existing owner/command transaction discipline. Guest reads fail closed while it is pending. Serialize conflicting settings intents; do not overwrite another operation's hold. A definite pre-dispatch/terminal rejection may clear only its own hold after current authoritative state is safely retained; ambiguous PDS/local failures keep it until retry or explicit owner observation resolves that operation. No new worker or periodic repair is introduced.

Use `StrictExpectedCID` and actual PDS write metadata. `AuthoritativeRecord` already separates `Revision/CommitCID` from `ObservationRevision/ObservationCID`. `FinalizeAccepted` runs in the terminal acceptance transaction for changed/recovered records. It must persist the visibility evidence and clear only the matching hold before accepting a 200 result.

**Unchanged-record acceptance seam:** the current `acceptRecord` path does not run that finalizer. Add an opt-in `FinalizeObserved` hook for this feature, executing in the terminal transaction after a stable authoritative observation, including verified absence. Keep other addressed commands' current defaults. This covers setting an already-enabled label on a legacy unknown row without forcing a no-op PDS write. Persist projection and acceptance atomically; terminal-result replay is allowed only after that transaction. No GET side effect or `Accepted` response-builder callback substitutes for transactional finalization.

Failure after a PDS write rolls back local acceptance and reports a retryable/ambiguous result; the durable hold prevents guest exposure. Retrying the same command uses existing authoritative recovery, not an unbounded background loop. A newer normal external edit may advance the established revision; finalization must never replace that newer state with an older response. Return current authoritative state or recoverable conflict under the existing command semantics rather than falsely claim the desired toggle is current.

### 5.3 Shared anonymous read policy and query reuse

Introduce `ReadAudience` as an internal explicit union (`Member(viewerDID, policy)` / `Guest`). Existing member store methods remain wrappers over audience-aware helpers. New `PublicReadService` selects only Guest. Do not silently convert a missing/invalid member bearer to guest access on protected member routes.

Factor common filter/sort/cursor/query helpers and renderable record hydration from existing post/search/profile/business stores. Add the guest owner eligibility predicate to base queries **before LIMIT and cursor generation**, and to independently hydrated quote/repost/parent/reply subjects. Query eligible rows in the database; do not fetch a member page and remove entries afterwards. Guest search exposes submitted-query/category results only, using existing default ordering. Do not add guest recent-user, autocomplete/suggestion or top-hashtag aggregates/endpoints. Public hashtag query results and tag detail still filter eligible source content. Reject advanced guest search filter/sort parameters; keep existing filters and sort controls on craft project feeds and preserve primary-craft feed eligibility.

Use explicit `PublicAuthor`, `PublicProfile`, `PublicPost` and public thread/list/business shapes. Extend them with approved public presentation fields: stable URI/CID where needed, structured project data/facets, public counts, reply policy, safe images/video/alt text, profile customization/links/business facts. Omit owner analytics, viewer interaction/saved/reply flags, mutes/blocks, mutual-follower personalization and private payloads. Existing clients may adapt absent member flags to inert local display defaults; never serialize invented member state on the guest wire.

Nested hydration receives the audience explicitly. Batch visibility decisions and record reads; no per-card PDS fetch. Ineligible embeddings use a minimal unavailable discriminant with no author/media/snippet; eligible outer commentary may remain. Independently addressed eligible replies may render with unavailable ancestor context. Direct eligible quotes and all existing public profile tabs follow the same policy. Aggregate engagement totals may retain opted-out participants, while identity lists exclude them.

### 5.4 Anonymous route inventory and bounded contracts (DR-004)

Keep the four current public URLs and expand within `/v1/public/`:

| Proposed route group | Shared implementation / presentation | Response / query policy |
|---|---|---|
| `GET /v1/public/projects` | Existing project browsing queries and home craft/filter rendering | Public post page; omitted `craftType` means all crafts, chronological default; existing public filter/sort vocabulary |
| `GET /v1/public/search/{profiles,posts,projects,hashtags}` | Corresponding existing search methods | Category-specific public pages; required submitted `q`, bounded cursor/limit; existing default result order. No advanced filter/sort parameters |
| `GET /v1/public/search/hashtags/{tag}/posts` | Existing tag result reading service | Eligible public post page in existing default order, bounded pagination; no autocomplete or top-hashtag endpoints |
| `GET /v1/public/profiles/{handleOrDid}` | Shared profile fields/business/customization hydration | Public profile; ordinary Projects/Posts/Comments/Reposts tabs; business Products/Upcoming events/About under existing non-owner tab policy |
| `GET /v1/public/profiles/{handleOrDid}/{posts,projects,comments,reposts,events}` | Existing author tab queries and business event list | Guest-safe paginated content; products/about live in public business profile projection, without management fields |
| `GET /v1/public/posts/{did}/{rkey}`, `/posts/{did}/{rkey}/thread` | Shared detail/thread pipeline | Public post, including replies as detail; bounded tree/section response matching existing reading UI |
| `GET /v1/public/posts/{did}/{rkey}/{likes,reposts,quotes}` | Existing interaction-list reading surfaces | Safe filtered identities/posts and existing totals where exposed; no mutations |
| `GET /v1/public/events/{did}/{rkey}` | Existing business event detail | Safe public event plus eligible author presentation |

These are concrete registrations, not a single unrestricted wildcard handler. No guest `/profiles/recent`, `/search/suggestions` or `/search/hashtags/top` route is registered or proxied. Explicitly reject reserved discovery paths before dynamic identifiers so `/profiles/recent` cannot accidentally resolve as a public profile request. Existing member discovery routes remain protected and unchanged. Do not add guest `/profiles/me`, mutual-followers, account follower-growth, search/recent, saved, notifications, messaging, drafts, scheduled or mutation routes. Current follower/following list methods are owner `/me` surfaces; public counts remain readable, but no new generic follower browser is introduced.

Reuse the aggregate public limiter/concurrency admission and timeout/no-store controls for **all** expanded reads. Retain current public detail/list default bounds (20 rows) and constrain new page requests to max 20; cap thread root/reply sections to the existing bounded section contract and max 20 returned posts per request with a continuation cursor. Reuse current filter validation limits and enforce the existing public 2048-byte cursor cap; bind new cursor versions to audience/resource/filter/sort so member cursors cannot cross into guest queries. Preserve legacy four-route cursors where compatible or reject safely; do not misinterpret them.

For known opt-out direct resources, return `403 authentication_required` with only generic message/requestId. Unknown/missing/deleted or other policy-hidden resources return generic 404; invalid input 400, throttle 429, unavailable 503. No identifying details are carried in excluded bodies. Svelte maps known opt-out to a generic sign-in-required presentation and `noindex`, never a retry loop. No auth requirement exists for eligible reads, including native app use.

## 6. State, Providers, Controllers, Or DI

### 6.1 Flutter/Riverpod

Keep existing repositories and content widgets; isolate guest reads behind a small shared abstraction rather than clone every page.

```text
sessionRegistryProvider + authSessionProvider
    -> browsingContextProvider (Provider; loading | Guest(sessionEpoch) | Member(lease))
guestSessionProvider (Notifier; process-local Explore state + return destination)
    -> contentCapabilitiesProvider (Provider; read/share vs member permissions)
browsingContextProvider -> contentReadGuardProvider (Provider; epoch/lease identity)
publicDioProvider (Provider, anonymous interceptors, same AppView origin)
    -> PublicReadApiClient -> guest Post/Profile/Project/Search read adapters
member context -> existing account-bound repository/client paths
    -> existing read providers, parameterized/invalidated by browsing context
owner lease -> guestVisibilitySettingsRepositoryProvider
    -> loggedOutVisibilityProvider (AsyncNotifier family keyed by ActiveAccountLease)
```

Existing content repository providers select a guest read adapter or their existing member implementation. If an interface combines reads/writes, the guest adapter rejects mutation methods locally as a defense; shared UI still hides those controls and server policy remains authoritative. Do not weaken `ActiveAccountOperation` globally. Only guest-capable read providers (`post`, thread/section loaders, profile/tabs, project feed and public search) use the new content-read guard. Member mutations, onboarding and private providers retain their current lease guard.

The new read guard captures an immutable guest epoch/member lease before awaits and discards late results on transitions. Add its dependencies to current account-boundary invalidation. Guest caches are keyed by epoch and cannot reuse member overlays. Bypass member publication/like/block overlays, member language preferences and saved craft preferences during guest reads; use existing device locale/public default policy with explicit transient filters. Do not initialize push/chat/billing/notification/account runtimes in guest mode.

`publicDioProvider` uses existing API error mapping and anonymous transport; no bearer, DPoP, PDS header or cookie credentials. Tighten `SessionAuthInterceptor` so `/v1/public/` never attaches a member token even if accidentally called on a member client; guest settings use only account-bound Dio. Device identity is optional to guest authorization, never a synthetic account or trusted admission key.

`loggedOutVisibilityProvider` loads only for the authenticated owner Privacy route. Keep last confirmed value during a save with explicit saving/pending status; an unknown read is loading/error, not an off switch. A command timeout/202 retains the operation key and desired value for retry. On account change, cancel/dispose and prevent stale publication. Use existing Riverpod public refresh lifecycle and accepted-result publication conventions rather than mutating unrelated cached profiles.

### 6.2 Svelte state and clients

Add discriminated `BrowsingContext` (`guest` or `member`, with session identity only on member) and typed read/capability contracts. Convert social component `did: string` requirements into explicit optional viewer/context input; never pass an invented guest DID. Keep mutation clients/session helpers unavailable unless member capability exists.

Extract `BrowsingShell.svelte` for common layout/navigation slots, toast/history host and content sizing. `AppShell.svelte` stays the member runtime wrapper for account initialization, Publications, push, languages, subscriptions and onboarding. `GuestRuntime.svelte` supplies public read API, epoch-scoped PageHistory and nonmember capabilities. Existing optional publication contexts must remain optional in guest rendering; no mutation subscription is started merely by rendering a post card.

Build `PublicCraftskyAPI` over the shared API transport/types and an explicit read-only method map to `/v1/public/*`. Expose a bounded same-origin `GET /api/public/[...path]` proxy for browser requests, with a closed route/query allowlist and no cookie/bearer forwarding; use the configured AppView origin and current body/redirect/time limits from `lib/server/api.ts`. Do not open the existing member `/api/appview/*` proxy anonymously. SSR calls the shared public helper directly. Request credentials are omitted, and all guest responses are no-store.

Key guest runtime/history/request generations by guest session and member runtime by active session ID. Navigation between modes destroys old contexts, cancels pending work and clears private snapshots. Retain existing `Work` generation/disposal protection and extend where context can change without remounting. Search skips **all** recent-user, autocomplete/suggestion, top-hashtag and recent-history effects for guests, including blank query, typing, clear and profile/result clicks. Blank search renders only the query entry/prompt. Submit commits query state; pagination/category navigation operates only on that submitted query. Guest advanced search filters/sort controls are hidden; craft project feed sort/filter controls stay active. Public-client/proxy allowlists contain no discovery methods, and malicious advanced guest-search query parameters are rejected rather than forwarded.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Entry, authentication and session definition

- Home `/` opens the existing home/feed shell. Guest default is an **All crafts** project browse query without craft filter, followed by all supported craft tabs in catalogue order. Existing Following remains member-only. Add a guest all-crafts feed definition to the current `HomeFeedDefinition`/web craft-feed model; do not reinterpret Following as global social ranking.
- Modal recurrence: Flutter guest session lasts for the app process/guest epoch, retained across background/foreground; a new process starts a new guest session. Svelte uses per-tab `sessionStorage` for Explore state, retained across reloads but not a new tab session. If storage is unavailable, fall back to runtime memory. Auth/guest transitions start a new epoch. Direct public content routes suppress automatic prompting; explicit banner/legacy auth entry always works.
- Extract current welcome/auth form into reusable content. Put it in existing responsive modal/dialog primitives with **Explore CraftSky**; preserve handle validation, help, legal links, registration handoff and add-account semantics. Legacy welcome/sign-in paths route to home with an explicit modal request; OAuth callbacks and errors still reach the correct auth/recovery state.
- Persistent banner is part of common shell layout, not a floating overlay covering reading/navigation. Both CTAs open the same modal and retain the internal public destination (including query/filter context). Registration remains existing provider onboarding. Only safe same-origin allowlisted public paths are accepted as return targets; private/auth/recursive/open-redirect targets fall back to home.
- Reuse current browser auth start/complete/confirm and native handoff flow; persist pending return intent under the existing pending-auth lifecycle, not in provider callback query values. Restore after required onboarding, never silently replay a mutation. Cancel/error leaves guest reading available.

### 7.2 Shared content and protected navigation

Flutter GoRouter allows home/search/profile/post/reply/event public routes and keeps private routes authenticated. Preserve auth-complete, eligibility, initialization and account-switch behavior. During unresolved session loading, render neutral initialization rather than briefly treating a real member as guest. Update bottom navigation, drawer and rail consistently: guest Home/Search and public reading routes only; no profile-self/messages/notifications/compose/account shortcuts. Authenticated settings continue as before.

On Svelte, split public-capable route classification from the current `authenticatedRoute` matcher. Keep public profile/post explicit loaders. Make the existing catch-all loader dispatch guest public home/search/tag/event/interaction routes to a `publicPage` loader and leave private routes on `authenticatedPage`. Root selects `/feed` for either viewer mode; guests no longer redirect to standalone sign-in. `+layout.svelte` chooses GuestRuntime/BrowsingShell or member AppShell by trusted loader/session state.

Feed/Search/Profile/Thread/MemberPost and Flutter equivalents render the same content sections with explicit capabilities. Read-only counts and thread-opening targets survive hiding mutation buttons. Guest menus keep copy/platform-share and safe external links, while hiding like/follow/repost/quote creation/bookmark/report/mute/block/edit/delete/account actions. Preserve public reporting-information links; do not invent anonymous report submission. Business Products/Upcoming events/About retain current non-owner availability. Media interaction, alt text and reduced-motion semantics remain in shared components.

### 7.3 SSR/OG preservation

Refactor existing public profile/post branches to render shared components with public loader data as their initial state. Guest components must not immediately refetch and erase that state. Keep native anchors/GET forms or progressive links for profile tabs and older pages so no-JS content navigation continues working. Avoid loading the entire authenticated runtime just to reuse markup.

Derive `Metadata.svelte` exclusively from the eligible public projection even when the browser user is authenticated. Authenticated page data cannot populate publicly rendered OG fallback. For excluded responses render only generic CraftSky branding/message, optional exact requested canonical path, and noindex; no avatar/name/bio/media/snippets in HTML, hydration, preload or structured data. Respect existing noncanonical-host noindex rules and no-store/referrer protections. Do not change robots/crawler policy globally or require third-party preview eviction.

Privacy UI in both clients uses the approved copy and shared setting; it coexists with error-reporting consent. It does not add the screenshot's unrelated algorithmic-recommendation preference.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Unknown legacy/malformed preference | Guest-hidden, no inferred public default or repair fetch; normal authoritative evidence may establish state | RULE-004 | IT-001, IT-006, IT-012 |
| Same CID, unknown projection | Source reducer initializes without repeating image work; bounded duplicate ingress initialization only for that received authoritative subject | FR-009, RULE-004 | IT-006 |
| Delayed old event after local save | Revision floor prevents rollback; source conflict fails closed; hold survives ambiguous acceptance | FR-009, RULE-004 | IT-005, IT-006 |
| Settings read/save/reauth/202 failure | Confirmed value distinct from pending intent; retry exact key; useful existing error mapping; no false success | FR-008, FR-009 | AT-008, UT-006, IT-005, IT-007 |
| Hidden-only result candidates | Filter in SQL before page size; genuine safe end state; no unbounded client fill loop | FR-001, FR-002, NFR-002 | AT-011, IT-002, IT-003, IT-009 |
| Hidden ancestor/quote/reposter | Independent eligibility per subject; minimal unavailable nested shape; eligible neighbours/outer commentary remain | FR-003, FR-010, RULE-002 | AT-009, IT-004 |
| Direct opt-out vs unknown/missing | Generic auth-required state for known opt-out; generic unavailable for unknown/missing; no identity returned | FR-011, RULE-002 | AT-010, IT-010 |
| 400/429/503 and cancelled reads | Existing safe envelopes, retry states, Retry-After and released admission slots; no raw diagnostic payload | NFR-002 | AT-011, UT-009, IT-009 |
| Logout/expiry/account switch | Existing account rules plus epoch/lease disposal; private cache cleared; stale responses ignored before guest display | FR-007, NFR-004 | AT-007, UT-005, REG-002 |
| Keyboard/text scaling/reduced motion | Shared accessible modal, focus restore, safe-area banner allocation and existing media behavior | NFR-003 | AT-002, AT-012, MAN-001, MAN-002 |

## 9. Test Implementation Plan

Each row is a test-first slice: write the smallest listed case, observe the relevant failure, implement the narrow behavior and rerun. Existing IDs are preserved. Manual cases are validation steps, not red-green source tests.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001, IT-012 | New `internal/db/guest_visibility_migration_test.go`; API/composition tests | Legacy rows at prior schema, instrumented outbound/job adapters | No unknown-state schema/policy; legacy content not excluded by visibility |
| 2 | UT-001, IT-006 | New `internal/guestvisibility/*_test.go`; existing index/ingestion integration | TD-003/005; both paths; same CID; received duplicate; delete/stale revision | No label projection, revision floor or duplicate unknown initialization |
| 3 | UT-007, IT-005, IT-007 | New `api/guest_visibility_test.go`, addressed-command tests | CAS, unknown fields, hold, write receipts, no-op observations, delayed source and partial failure | Settings route absent; no transactional visibility acceptance |
| 4 | IT-008, IT-009, UT-009 | Routes public/policy tests and public handler/store tests | Public/private route inventory, canaries, quota/slot barriers, malformed cursor | Missing routes/limits or member-only read assumptions |
| 5 | IT-002, IT-003 | Public/search/project store tests with real PostgreSQL | TD-001/002/003/007; craft-feed filters/sorts, basic submitted search, forbidden discovery calls, hidden-only hashtag and paging ties | Guest audience missing before limits or prohibited discovery requests still execute |
| 6 | IT-004, IT-011 | Public thread/profile/business/interaction integration | TD-004 plus suspension/deletion/warn/image state | Missing guest tabs/threads or nested-policy leaks |
| 7 | UT-008, IT-010, AT-010, REG-003 | Existing public adapter/Vitest and JS-disabled public pages | Same public contract; raw SSR/OG/preload canaries and transition | Existing reduced public DTO/metadata does not enforce new policy |
| 8 | UT-002, UT-003, UT-005, AT-002, AT-007 | Flutter router/auth/browsing tests; Svelte shell/state tests | Fresh session, Explore, private URL, stale async result and mode switch | Signed-out redirect and member-only runtime assumptions |
| 9 | UT-004, AT-001, AT-003, AT-004, AT-005 | Flutter home/search/profile/thread suites; guest browser browsing spec | Per-surface inventory fixtures; private-history request spies | Member-only guards/queries/actions still render or execute |
| 10 | UT-006, AT-008, AT-009 | Privacy widget/provider and guest-privacy browser integration | Two owners, same label across clients, saves/reauth/normal external events | Toggle absent or false-success behavior |
| 11 | AT-006, REG-001, REG-002 | Auth/router/device and guest-auth-transitions browser spec | Safe/hostile returns, onboarding, add-account, retained accounts and expiry | Destination dropped or guest/member state mixed |
| 12 | AT-011, AT-012, REG-004, REG-005, REG-006, REG-007 | Guest accessibility/browsing; existing member/safety/Privacy suites | Bounds/retry states, component parity, text scale/reduced motion and non-owner business tabs | New guest path loses parity or violates private/moderation constraints |
| 13 | MAN-001, MAN-002, MAN-003 | Native device/manual evidence and optional disposable-provider smoke | Approved test account only where needed; defined platform matrix | Record unverified/live-limited behavior separately; never label mocks as device/provider proof |

### Acceptance-criterion mapping

| Acceptance criteria | Plan sections / test slices |
|---|---|
| AC-001, AC-003, AC-004, AC-005 | Shared public read contracts/query policy (5.3–5.4), guest repositories/providers (6), shared UI/SSR (7); slices 5–9 |
| AC-002, AC-007 | Viewer/session boundaries (6), modal/banner/return flow (7.1–7.2); slices 8 and 11 |
| AC-006 | Explicit access/DTO/capability allowlists (5.3–5.4, 6–7); slices 4 and 9 |
| AC-008, AC-009, AC-014 | Owner settings and durable finalization (5.2), owner AsyncNotifier/UI (6.1, 7.3); slices 3 and 10 |
| AC-010, AC-017 | Evidence/normal-event/no-backfill reducer (5.1), owner revision fence (5.2), nested read policy (5.3); slices 1–6 and 10 |
| AC-011 | Shared public metadata and raw SSR contract (5.4, 7.3); slice 7 |
| AC-012 | Database filtering before bounds and shared admission/cursors (5.3–5.4, 8); slices 4–5 and 12 |
| AC-013, AC-015 | Shared presentation and capability-sensitive layout (6–7); slices 9 and 12–13 |
| AC-016 | Existing public/member lifecycle/moderation and image policy (5.3, 8, 10); slices 6 and 12 |

### Runnable web guest harness (DR-003)

Implement the proposed `tests/playwright-guest.config.ts` and four `guest-*.spec.ts` files from the test specification. Use isolated ports 19850 (web preview/Worker) and 19851 (mock AppView), rejecting reuse of unrelated running servers. Factor fixture utilities from existing public/authenticated mock servers so guest/member transitions share one deterministic contract. Add private-request counters, visibility transitions, same-CID/unknown fixtures, business tabs, safe nested states, pagination and failure controls. Fail guest calls to recent-user/autocomplete/top-hashtag endpoints; assert blank/typing search issues no query requests, and craft-feed sorting remains available.

Projects: Chromium desktop (1280×900), touch Chromium phone (390×844), mobile WebKit (390×844); add a 768×1024 tablet layout case in accessibility specs. Install/use supported Playwright engines through existing tooling only during implementation. Keep JavaScript enabled for the guest config and disabled in the existing public config. Reuse authenticated runner coverage for unaffected member behavior rather than duplicating all member scenarios.

From `web-client/`, after build:

```sh
npx playwright test --config tests/playwright-guest.config.ts --list
npx playwright test --config tests/playwright-guest.config.ts
WEB_CLIENT_WORKER=1 npx playwright test --config tests/playwright-guest.config.ts
npm run test:e2e
npm run test:e2e:app
```

Guest test discovery must name all four intended specs and all configured projects; zero tests is not a pass. Worker mode has the same fixture expectations. Add the explicit guest runs to `public-app-check` in `justfile`; the inspected `web-client-check` recipe aliases that aggregate gate. Recheck its current contents before editing. Do not assume the existing commands select the new files.

### Focused backend/Flutter and corroboration

First proposed failing test name: `TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown`. From repo root, with existing compose test dependencies available:

```sh
./scripts/appview-test focused ./internal/db -run '^TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown$'
./scripts/appview-test focused ./internal/guestvisibility ./internal/index ./internal/ingestion -run 'GuestVisibility|BlueskyProfile'
./scripts/appview-test focused ./internal/api ./internal/routes -run 'Guest|Public|LoggedOutVisibility'
just app-test test/router/router_redirect_test.dart test/auth/welcome_page_test.dart test/settings/privacy_page_test.dart
```

Extend filters/files to include the new tests actually created. Full required validation later includes `just test`, `just app-analyze`, appropriate full Flutter suites, web check/unit/build plus public/member/guest Node and Worker matrices. Run diagnostic sink/canary checks if diagnostics change. All commands are future validation, not executed results.

Add `app/integration_test/guest_browsing_test.dart` reusing existing integration support and a local guest AppView fixture. Run with `cd app && flutter test integration_test/guest_browsing_test.dart -d <selected-device>` after repository-native library setup. Native UI interaction uses device tools; no native desktop targets. Include iOS and Android checks. Extend `just app-test-integration` only if it gains explicit feature-file selection; the current recipe names the existing critical-journeys test and does not automatically run the new file.

Corroborate the browser mocks with a local compose-AppView-backed HTTP journey exercising real API/Postgres projection and fetching Svelte initial HTML before/after applied opt-out. Reuse a local test fixture process, never production data. Keep mocked contract tests deterministic and distinct from this corroboration and optional live-provider smoke.

## 10. Sequencing And Guardrails

- **First TDD step:** IT-001 with a legacy profile at previous schema; migration adds unknown-state columns without updating old rows or triggering any network/job activity. Pair with the first guest-detail hidden-state assertion before declaring the policy slice complete.
- **Dependencies:** projection/ordering and durable owner acceptance precede broad guest exposure. Public safe DTO/query contract precedes client adapters/SSR consolidation. Client mode isolation precedes enabling guest route access. Run tests per slice, then the required aggregate gates once the behavior is complete.
- **Release readiness:** nullable defaults make legacy accounts guest-hidden by design; some feeds can be empty until normal events arrive. Do not “fix” that with population scripts or a false default. This plan includes no release, Blueprint sync or production migration execution.
- **No backfill:** no historical source replay, retained-source scans, profile fetch on guest reads, startup repair or visibility-driven schedule/worker. A bounded operation on an actually received duplicate is normal-event processing, not a request for historical data. Existing unrelated repository/lifecycle recovery remains intact; do not enlist it to initialize this feature.
- **Concurrency:** visibility writes/reads use existing owner fences and transaction lock ordering. Never hold an ordinary database transaction across PDS network IO. Durable hold installation, network dispatch and terminal acceptance are separate framework-coordinated phases. Source projection and owner acceptance compare real revision/observation evidence inside transactions.
- **Auth and schemas:** standard profile self-label only; no OAuth tokens or PDS reads in clients, no broader grant, no generic PDS endpoint. Preserve existing account deletion/lifecycle restrictions. No changes under `lexicon/` or generated custom record types are expected.
- **Scope:** no algorithmic feed, paid ranking, anonymous writes, new follow browser, new privacy guarantees or added native desktop target. Reuse the public content UI and shared services; do not create parallel full guest implementations.
- **Diagnostics:** use existing reporters/classification and typed causes; no profile bodies, search text, credentials, recipient routing or hold keys in logs. Apply the mandatory diagnostic guide and sink-canary tests if adding diagnostic paths.
- **Documentation/approval:** implementation starts only after the user approves this concrete plan; no stage commit is enabled. Review the final implementation separately under the workflow after tests.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking implementation verification | Exact next migration number and generated sqlc wiring can drift before implementation | Naming/generated output changes | Choose current next number at the first TDD slice; no hard-coded historic migration ID. Columns stay on the existing owner-purged row. |
| CPQ-002 | Non-blocking implementation verification | Existing owner-fence/command completion helper must support unchanged-observation finalization and hold rollback safely | False success or delayed visibility rollback | Implement explicit opt-in hook and terminal transaction tests before exposing the setting; fail closed if receipt/observation evidence is unavailable. Do not bypass command storage. |
| CPQ-003 | Non-blocking implementation verification | Ingress duplicate fast path and image-safety retry path may bypass naive projection initialization | Unknown state never resolves or repeated image work | Subject-only received-event handling plus separate visibility update; test both constructors and actual ingestion duplicates. No projection sweep. |
| CPQ-004 | Non-blocking implementation verification | Shared components have implicit member contexts and required DTO fields | Guest rendering could trigger private work or lose SSR | Audit the section 4 components as they are adapted; explicit capabilities/read interfaces and private-request spies. Keep positive no-JS SSR cases. |
| CPQ-005 | Non-blocking environment | Device/WebKit/Worker and disposable live account availability are not established in this planning stage | Limits validation evidence | Discover at implementation validation time; record missing proof separately. Local contract/Postgres tests remain mandatory; no invented fallback live claims. |
| CPQ-006 | Non-blocking operational consequence | No backfill means legacy content may remain unavailable to guests indefinitely absent normal events/owner settings access | Empty/reduced guest discovery | Approved constraint, not a reason to add repair. Use honest empty states and preserve verified-public fixtures. |

No unresolved product question blocks implementation design. These are bounded implementation checks, with a concrete safe choice specified above. Risk remains High. If existing command/ordering evidence cannot support the prescribed invariant, stop that slice and amend this plan with evidence before weakening visibility or adding synchronization infrastructure.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-11-05-logged-out-browsing/04-coding-plan.md`.
- TDD execution plan: `05-implementation-plan.md`, to be created by `implement-tdd` only after user authorization.
- Start with test: IT-001 / `TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown`, followed by guest-detail exclusion and IT-006 same-CID unknown initialization in both projection paths and ingestion duplicate delivery.
- Focused command: `./scripts/appview-test focused ./internal/db -run '^TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown$'` from repository root with compose test dependencies running.
- DR-001: resolved by nullable projection, independent same-CID reducer and narrowly scoped duplicate ingress initialization.
- DR-002: resolved by PDS receipt/observation revision floors, durable pending hold, transactional acceptance including unchanged observation, and delayed-event/partial-failure tests.
- DR-003: resolved by named JS-enabled guest config/specs, isolated fixtures/ports, explicit discovery/Node/Worker commands and desktop/mobile/tablet matrix.
- DR-004: resolved by route/public-field inventory, shared read audience/hydration and explicit guest capability mapping across profile/business/search/thread/media views.
- Status: **Coding plan complete, awaiting user approval for implementation.** No implementation, tests, migrations or live operations have been executed. Only this artifact is created in this stage; requirements, test design and historical review remain unchanged.
