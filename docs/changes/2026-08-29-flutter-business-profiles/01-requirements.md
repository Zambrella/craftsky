# Requirements: Flutter Business Profiles

## 1. Initial Request

Implement the Flutter portion of Craftsky business profiles on top of the completed AppView work. Members need to switch between regular and business account types, present business information alongside their existing profile, show upcoming events and featured products in profile tabs, manage events and products from business-only settings, and edit business details within the existing profile-edit flow.

Discovery confirmed an integrated Flutter slice with the smallest client-enabling AppView API changes. The feature does not change the approved lexicons, persistence model, eligibility rules, or mutation routes in `docs/changes/2026-08-27-business-profiles/01-requirements.md`.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/profile/models/profile.dart`, `app/lib/profile/data/profile_api_client.dart`, and `app/lib/profile/providers/` implement profile models, AppView access, and state.
  - `app/lib/profile/pages/profile_page.dart`, `app/lib/profile/widgets/profile_meta_section.dart`, `app/lib/profile/widgets/profile_tab_bar.dart`, and `app/lib/profile/widgets/profile_tabs/` implement the shared owner/visitor profile and its fixed five-tab layout.
  - `app/lib/profile/pages/edit_profile_dialog.dart` implements a full-screen, dirty-tracked profile editor with image upload, unsaved-work protection, and one Save action.
  - `app/lib/settings/pages/settings_page.dart`, `app/lib/settings/pages/account_page.dart`, `app/lib/settings/models/settings_row.dart`, and `app/lib/router/router.dart` implement nested settings pages and typed routes.
  - `app/lib/shared/media/blob_api_client.dart` and the existing profile/composer image flows provide reusable authenticated image upload behavior.
  - `appview/internal/api/profile_response.go`, `appview/internal/business/profile.go`, and `appview/internal/business/event.go` define the implemented profile, product, and event response shapes.
  - `appview/internal/api/business_profile.go`, `appview/internal/api/business_event.go`, and `appview/internal/routes/routes_business.go` implement the approved business mutation, event list/detail, owner-management, and report routes.
  - `docs/changes/2026-08-27-business-profiles/01-requirements.md` is the approved source of truth for business data, validation, public eligibility, moderation, and API behavior.
- Existing patterns:
  - Flutter uses Dio API clients, Riverpod providers, `dart_mappable` models, typed `go_router` routes, generated localization, responsive modal navigation, and account-scoped invalidation.
  - The same profile screen serves owners and visitors. Tabs use a `DefaultTabController`, `NestedScrollView`, and sliver content with retained per-tab scroll positions.
  - Settings use labeled sections and disclosure rows. Account settings currently contain permanent account deletion.
  - PDS writes are mediated by AppView; Flutter holds only the Craftsky session and must not read a PDS directly.
  - Record replacement and deletion use current CID preconditions and return `409 pds_record_conflict` when stale.
- Current behavior:
  - Flutter ignores `accountType` and `business` fields and has no business models, controls, presentation, products, events, or business settings.
  - Profiles always have five tabs: Projects, Posts, Comments, Reposts, and About.
  - Profile editing updates ordinary identity/profile fields only.
- Constraints discovered:
  - Account type is AppView-authoritative and exactly `regular` or `business`; switching to regular suppresses business presentation without deleting declaration, products, or events.
  - Business details and ordered products share one singleton declaration record. Updating either must preserve the other known declaration fields.
  - Upcoming events are fetched separately from the main profile. Owner event management uses a different all-events endpoint with suppression diagnostics.
  - Business declaration replacement requires its current CID, but the implemented profile response does not expose that CID and no declaration GET route exists.
  - Implemented product/event image responses retain raw record image data rather than the normalized, display-ready image view Flutter already consumes elsewhere.
  - Ordinary profile and business declaration updates are separate PDS records and cannot be atomically saved.
  - Blocked profile shells omit account type and business data; blocked event lists are empty and direct event reads return not found.
- Test/build commands discovered:
  - `just app-test`
  - `just app-analyze`
  - From `app/`: `dart run build_runner build --delete-conflicting-outputs`
  - Relevant focused suites: `flutter test test/profile test/settings test/router`

## 3. Clarifying Questions And Decisions

### Q1: When are Products and Upcoming Events tabs visible?

Answer: Both business tabs are always visible on normally visible business profiles.

Decision / implication: Every normally visible business profile shows Products and Upcoming Events regardless of ownership or content. Owners receive useful empty states linking to the corresponding settings manager; visitors receive explanatory empty states without management controls. Regular profiles retain the existing tab set, while blocked profiles retain the existing blocked-shell presentation. Both business tabs therefore have stable loading, error, retry, empty, and content surfaces where applicable.

### Q2: Where is business information presented?

Answer: Summary plus About.

Decision / implication: A self-declared Business label, optional tagline, and optional primary action appear near the existing identity/profile summary. Full business types, offerings, location, service area, and hours appear in About without duplicating ordinary identity fields.

### Q3: What happens when switching from business to regular?

Answer: Retain without warning.

Decision / implication: The account type changes immediately after a successful request. Business details, products, and events are retained but hidden by AppView eligibility. The business settings section disappears and returns with the retained data if the member switches back. The app does not offer deletion or a confirmation prompt during this switch.

### Q4: What is the visitor event experience?

Answer: In-app event detail.

Decision / implication: Upcoming event cards open a dedicated detail screen that presents the complete hydrated event, external event/registration actions, and the existing event report flow.

### Q5: How does the combined profile editor save ordinary and business fields?

Answer: One Save action with partial-success reconciliation.

Decision / implication: Only changed records are mutated. If one record succeeds and the other fails, the successful result remains saved, the editor updates its baseline for that portion, clearly identifies what failed, and permits retry without resubmitting or reverting the successful portion.

### Q6: Where does account type selection live?

Answer: Account settings.

Decision / implication: The existing Account page exposes the regular/business selector. The main Settings page conditionally shows a Business section containing Events and Products only while the authoritative account type is business.

### Q7: Is an additional public-PDS warning required?

Answer: No.

Decision / implication: Business publication uses the ordinary editing/management interaction model. No first-use confirmation or additional inline public-data warning is introduced in this slice.

### Q8: May the Flutter workflow include AppView contract fixes?

Answer: Yes, only the minimum required for the client.

Decision / implication: The profile's eligible business projection exposes its current declaration CID, and product/event images use a normalized AppView image view containing display URLs and source metadata sufficient for unchanged mutation round-trips. No new route, persistence, lexicon, or eligibility behavior is introduced.

### Q9: Which overall implementation direction is confirmed?

Answer: One integrated profile and settings slice.

Decision / implication: Account type, profile presentation/editing, product management/presentation, event management/presentation/detail, and minimal AppView API alignment are specified together so every exposed journey is complete.

### Q10: How should owner events be divided for management?

Answer: The Events settings screen has Upcoming and History views, backed by a basic owner-endpoint filter.

Decision / implication: Upcoming contains events whose effective status is scheduled and whose end is after that view's server cutoff, including publicly suppressed events so owners retain diagnostics and a path to correct them. History contains ended, cancelled, postponed, unknown-status, or otherwise non-active events relative to its own cutoff. `GET /v1/events` accepts optional `filter=upcoming|history`; omitting the filter preserves the approved all-events contract. Upcoming is nearest-first, History is most-recent-first, and each view independently binds its cursors to its own filter and first-page cutoff. The views are not a shared point-in-time snapshot: an event may move between them as time passes or state changes, and refreshing either view discards only that view's accumulated pages and starts it with a new cutoff.

## 4. Candidate Approaches

### Option A: Integrated Profile And Settings Slice

Summary: Deliver the full Flutter experience with minimal AppView API alignment in one coherent change.

Pros: Completes every owner and visitor journey; avoids shipping unmanageable profile surfaces; centralizes model, cache, conflict, and navigation behavior.

Cons: Broad profile/settings test surface and multiple forms are included in one workflow.

Risks: Partial multi-record saves, projection lag, dynamic profile tabs, and account-switch cache leakage require explicit behavior and regression coverage.

### Option B: Phased Flutter Delivery

Summary: Add account type and read-only presentation first, then products, then events.

Pros: Smaller implementation increments and narrower reviews.

Cons: Temporarily exposes incomplete business profiles and repeatedly changes the same profile/settings architecture.

Risks: Interim states may be mistaken for finished product behavior and require temporary code or tests.

### Option C: Dedicated Business Hub

Summary: Manage all business content in a new dashboard while keeping ordinary profile editing separate.

Pros: One management entry point and less complexity in the current profile editor.

Cons: Conflicts with the confirmed standard edit-profile flow and adds a new top-level navigation concept.

Risks: Duplicated profile concepts and weaker discoverability from existing settings patterns.

## 5. Recommended Direction

Recommended approach: Option A, the integrated profile and settings slice.

Why: Business info, products, and events form one public profile experience but have distinct existing backend lifecycles. Integrating them through the current profile and settings patterns gives owners complete maintenance paths, gives visitors coherent presentation, and preserves the approved AppView boundaries without introducing another navigation model.

## 6. Problem / Opportunity

The AppView can classify business accounts and serve business declarations, products, and events, but Flutter cannot display or manage them. Members need a transparent way to present a craft business on their existing identity, and visitors need usable product and upcoming-event experiences. The Flutter client also needs conflict-safe, display-ready API data to use the implemented backend without reading PDS records directly.

## 7. Goals

- G-001: Let a member switch the active account between regular and business from Account settings.
- G-002: Present self-declared business identity and details as part of the existing profile experience.
- G-003: Let business owners curate up to four externally linked featured products and control their order.
- G-004: Let business owners create, inspect, edit, cancel/postpone, and delete event appearances through separate Upcoming and History management views.
- G-005: Let visitors browse featured products and upcoming events, open event details, launch external actions, and report events.
- G-006: Preserve business records when account type changes and preserve successful writes during a partial combined-profile save.
- G-007: Maintain account isolation, accessibility, localization, responsive layout, block/moderation boundaries, and external-commerce-only behavior.

## 8. Non-Goals

- NG-001: Any lexicon, migration, account-type, eligibility, taxonomy, validation, moderation, or persistence change beyond the approved AppView contract.
- NG-002: New AppView routes, direct PDS reads, PDS tokens on device, or client-side bypass of AppView validation.
- NG-003: Subscription, verification, endorsement, paid reach, advertising, ranking, search/filter advantages, or a `pro` account type.
- NG-004: Native product detail pages, inventory, variants, checkout, orders, availability, tax, shipping, price synchronization, or commerce guarantees.
- NG-005: Global event discovery, recurrence, attendee/registration management, ticketing, maps, reminders, calendar export, or push notifications for events.
- NG-006: Editing business records while the account is regular in the first-party UI, even though the backend permits portable setup.
- NG-007: Deleting declaration, products, or events as a side effect of switching to regular.
- NG-008: A first-use or inline warning specifically about public PDS business records.
- NG-009: A visible business marker rollout across every feed, search, notification, and relationship summary; models must tolerate the existing field, but this slice requires presentation on the full profile.
- NG-010: Separate business name, avatar, banner, represented brand, team member, or delegated business management.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Regular member | A signed-in current member using an ordinary account | Understand and select business account type without losing the existing identity |
| Business owner | A current member whose authoritative account type is business | Edit business details, manage products/events, understand conflicts and event suppression, and preview public results |
| Profile visitor | An authenticated member viewing a business profile | Recognize self-declared business status, understand the business, browse products/events, and safely launch external actions |
| Event reporter | An authenticated visitor reporting an individual event | Open event detail and use the established report reasons and feedback behavior |
| AppView | Craftsky's authenticated read/write mediator | Supply conflict-safe client projections and enforce all approved validation, eligibility, block, and moderation rules |

## 10. Current Behavior

Flutter renders all current accounts as ordinary profiles with the same fixed tabs and settings. The Account page exposes deletion only. Profile editing updates existing identity and Craftsky profile fields, while business fields in profile responses are unmodeled. There are no product/event models, repositories, providers, routes, forms, cards, tabs, detail screens, or management screens.

The AppView routes and records exist, but an existing declaration cannot be replaced by Flutter because its CID is unavailable. Product and event image projections are not in the client-ready normalized form used by existing Flutter media widgets.

## 11. Desired Behavior

Account settings displays an authoritative regular/business selector. Successful changes immediately update account-scoped UI and invalidate or reconcile affected profile/event state. Switching to regular has no confirmation and deletes nothing. Switching back restores retained AppView-served data.

A normally visible business profile shows a non-verification Business label, optional tagline, and primary action near existing profile information. About shows complete declaration details. Every business owner and visitor sees Products and Upcoming Events tabs, including appropriate empty states. Products launch their external destinations. Upcoming event cards open an in-app detail screen with external actions and reporting.

The main Settings page shows Events and Products under a Business heading only for a business account. Products are edited as an ordered part of the declaration. Events have paginated Upcoming and History management views with lifecycle and suppression information. The standard profile editor conditionally adds business detail fields and saves changed ordinary/business records through one action with accurate partial-success reconciliation.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A current member shall be able to select regular or business presentation for the same active account without creating another identity. | Business status augments the existing member identity. | Initial request, Q6 | AC-001 |
| BR-002 | Business | Must | A business profile shall transparently identify itself as self-declared and present useful business details alongside existing profile information without implying verification or endorsement. | Supports transparent business participation. | Initial request, approved AppView requirements, Q2 | AC-002, AC-003 |
| BR-003 | Business | Must | Business owners shall be able to feature external products and manage public event appearances from business-only settings. | Provides the requested owner maintenance paths. | Initial request | AC-004, AC-005 |
| BR-004 | Business | Must | Visitors shall be able to browse available products and upcoming events without introducing native commerce or event registration. | Keeps transactions and registration external. | Initial request, approved AppView requirements | AC-006, AC-007 |
| FR-001 | Functional | Must | The Flutter full-profile response model shall decode `accountType` as `regular` or `business`, tolerate its required omission on blocked shells, and decode the optional eligible business projection without failing on omitted optional fields or AppView-safe unknown catalog values. This slice does not add `accountType` to post-author, feed, search, notification, relationship, or other compact identity models; their existing decoders shall continue to ignore unknown response fields. | Matches normal and redacted full-profile shapes while bounding the client-model rollout. | Codebase, approved AppView requirements, NG-009 | AC-008 |
| FR-002 | Functional | Must | Account settings shall display the authoritative account type and allow a current member to set either supported value through `PUT /v1/profiles/me/account-type`, with pending state preventing duplicate mutation and errors leaving the prior state visible. | Provides a reliable account control. | Initial request, Q6 | AC-001, AC-009 |
| FR-003 | Functional | Must | After a successful account-type mutation, the app shall reconcile the active account's profile/settings state immediately. Switching to regular shall show no confirmation, remove business-only profile/settings surfaces, and delete no declaration, product, or event data; switching back shall restore retained eligible data after refresh. | Implements the confirmed reversible behavior. | Q3 | AC-010 |
| FR-004 | Functional | Must | The main Settings page shall show an Events row and Products row under a Business heading only when the active account's authoritative type is business. Direct navigation to owner management screens while regular shall not expose management controls and shall return the member to an appropriate settings surface. | Enforces the first-party UI scope consistently. | Initial request, Q6, NG-006 | AC-011 |
| FR-005 | Functional | Must | A normally visible business profile shall show a Business text label without a verification icon, optional tagline, and optional primary action near existing profile information; action labels shall map from the approved action catalog and launch only the hydrated destination. | Creates concise transparent presentation. | Q2, approved AppView requirements | AC-002, AC-012 |
| FR-006 | Functional | Must | The About tab of an eligible business profile shall present available business types, offerings, locality/country, service area, and hours note with localized known-value labels and safe fallback labels for unknown values. It shall omit absent sections without placeholder clutter. | Presents complete optional business context. | Q2, approved AppView requirements | AC-003, AC-013 |
| FR-007 | Functional | Must | Profile tabs shall be derived from profile/account context while retaining the existing ordinary tabs and stable per-tab navigation. Every normally visible business profile shall receive Products and Upcoming Events tabs regardless of ownership or content. Owners receive setup empty states; visitors receive explanatory empty states without management controls. Upcoming Events retains loading, retryable error, empty, and content states. Regular and blocked profiles receive neither business tab. | Gives every business profile a simple stable tab structure without changing ordinary profiles. | Q1 | AC-014, AC-015 |
| FR-008 | Functional | Must | The existing profile-edit flow shall add business types, offerings, tagline, hours note, service area, locality/country, and primary-action fields only when the active account is business, prefilled from the current declaration and constrained to the approved AppView catalogs and limits. | Keeps identity and business details in one familiar flow. | Initial request, Q5 | AC-016 |
| FR-009 | Functional | Must | One profile-editor Save action shall submit only changed ordinary and business records. Business replacement shall include all current known declaration fields, including products, and use the current declaration CID or `If-Match: *` for creation so editing details cannot erase products or unknown server-preserved extensions. | Prevents atomic-record data loss. | Codebase, Q5, Q8 | AC-017, AC-018 |
| FR-010 | Functional | Must | If a combined profile Save partially succeeds, the app shall retain and display the successful portion, update its dirty/baseline state, identify the failed portion, remain open, and allow retry of only the failed portion. A full success shall refresh/reconcile profile state and close through the existing successful-save behavior. | Separate PDS records cannot be committed atomically. | Q5, Codebase | AC-019 |
| FR-011 | Functional | Must | The Products settings page shall list the declaration's products in authored order and support adding, editing, removing, and reordering cards up to the approved maximum of four. Each first-party product requires title, HTTPS destination, and image, with optional alt text and optional canonical amount/currency. | Provides complete product curation. | Initial request, approved AppView requirements | AC-004, AC-020 |
| FR-012 | Functional | Must | Product changes shall replace the declaration with its current CID while preserving every other current known declaration field. A `409 pds_record_conflict` shall not overwrite newer state; the app shall explain that the data changed and offer reload/retry. | Products and details share one atomic declaration. | Codebase, Q8 | AC-018, AC-021 |
| FR-013 | Functional | Must | The Products profile tab shall display ordered product cards with image, title, optional formatted seller-authored price, and an external-link affordance. Activating a card shall launch its hydrated HTTPS URI outside native Craftsky commerce and surface launch failure without navigating to a nonexistent native product page. | Implements featured external products honestly. | Initial request, approved AppView requirements | AC-006, AC-022 |
| FR-014 | Functional | Must | The Events settings page shall provide separate Upcoming and History views, paginate each view independently, and expose server-provided suppression/upcoming-exclusion reasons in understandable owner-facing text. Upcoming shall include active scheduled events even when publicly suppressed and order by `startsAt ASC, URI ASC`; History shall include ended, cancelled, postponed, unknown-status, and otherwise non-active events and order by `startsAt DESC, URI DESC`. Each view shall retain its own confirmed rows during incremental failure, and refresh or a successful lifecycle mutation shall discard affected accumulated pages and restart affected views from page one with independent new cutoffs. No cross-view shared-snapshot guarantee is made. | Owners need a useful active/archive split without losing complete management or diagnostics. | Initial request, approved AppView requirements, review feedback Q10 | AC-005, AC-023, AC-041 |
| FR-015 | Functional | Must | A business owner shall be able to create and edit an event with name, start/end, roles, mode, status, timezone, all-day state, and optional summary, venue, event URI, registration URI, and image. The form shall enforce approved first-party required fields and bounds, represent local date/time choices as server-required UTC whole-second instants, and preserve server-owned `createdAt` on update by omitting it from mutation bodies. | Provides valid event authoring against the existing contract. | Approved AppView requirements | AC-024, AC-025 |
| FR-016 | Functional | Must | Event update and deletion shall use the event's current CID. Destructive deletion shall require confirmation. A conflict shall preserve the newer server record and offer reload/retry; successful mutation shall reconcile owner-management, profile-upcoming, and direct-detail state without exposing a stale duplicate. | Prevents lost updates and accidental deletion. | Codebase, approved AppView requirements | AC-026, AC-027 |
| FR-017 | Functional | Must | The Upcoming Events profile tab shall page through the dedicated profile events endpoint in ascending server order, display every AppView-returned event without client lifecycle/status re-filtering, show useful date/time, role/mode, venue, and image summaries, and provide initial loading, owner and visitor empty, retryable initial and incremental error, pagination, and end-of-list states. An initial failure retains the tab and exposes Retry; a refresh failure retains the last confirmed items and exposes retry feedback without replacing them with an empty state. | Provides a complete public event list and stable visitor recovery. | Initial request, Q1 | AC-007, AC-028 |
| FR-018 | Functional | Must | Activating an upcoming event shall open an in-app detail route identified by owner DID and record key. The detail shall show the complete hydrated event and offer available external event and registration actions; missing optional values shall not render empty controls. | Implements the confirmed detail experience. | Q4 | AC-029 |
| FR-019 | Functional | Must | A non-owner event detail shall expose the existing report flow for that event. Successful reports, validation failures, unavailable events, block/moderation not-found responses, and external-launch failures shall use established app feedback/error patterns without leaking suppressed data. | Extends existing moderation UX safely. | Q4, approved AppView requirements | AC-030 |
| FR-020 | Functional | Must | Product and event image selection shall reuse the authenticated image upload path and existing MIME/size/alt behavior. Existing images shall remain unchanged when not edited, and removed/replaced images shall produce the correct full-record mutation payload. | Avoids duplicate media behavior and accidental image loss. | Codebase, approved AppView requirements | AC-020, AC-025, AC-031 |
| FR-021 | Functional | Must | AppView's eligible business projection shall expose the current declaration `cid`. Every safe product image on the profile response and event image on owner-list, public-list, and detail responses shall use the reusable `PostImageView` wire object: required nonempty `cid` string, required nonempty `mime` string limited to `image/jpeg`, `image/png`, or `image/webp`, required nonnegative integer `size`, required `alt` string (which may be empty), optional `aspectRatio: {width: positive integer, height: positive integer}`, and required nonempty `thumb` and `fullsize` strings. The image object is omitted when the source image is missing, malformed, unsupported, or otherwise unsafe. `cid`, `mime`, `size`, `alt`, and `aspectRatio` reconstruct the unchanged mutation image as blob `$type`, `ref: {$link: cid}`, `mimeType: mime`, `size`, authored `alt`, and optional `aspectRatio`; display URLs are never sent in mutations. | Flutter needs one exact concurrency-safe and renderable image contract without PDS access. | Codebase gap, Q8, existing `api.PostImageView` | AC-032, AC-033 |
| FR-022 | Functional | Must | After a successful business/product/event mutation, the client shall compose the accepted local view from the submitted payload and mutation response, keyed by account, record identity, pre-write CID (or absence for create), accepted CID, and request generation. While AppView reads still return the exact pre-write CID, or absence after create, the accepted overlay remains visible. A read with the accepted CID clears the overlay and becomes authoritative. A read with any different CID is treated as concurrent authoritative divergence, clears the overlay, and replaces it; CIDs are never ordered. A successful delete keeps the record locally absent while reads return the deleted CID, settles when AppView returns absence, and adopts any different-CID record as divergence. Refresh failure retains the overlay with retry feedback; explicit reload discards it only after warning that accepted local state may be replaced. Late results from another account or superseded generation are ignored. | PDS projection may lag the mutation response and CIDs are opaque. | Codebase, Discovery, document review | AC-034 |
| FR-023 | Functional | Must | `GET /v1/events` shall accept optional `filter=upcoming|history` alongside `limit` and `cursor`. `upcoming` shall return events whose effective status is scheduled and whose `endsAt` is after that view's first-page server cutoff, regardless of public suppression, ordered by `startsAt ASC, URI ASC`; `history` shall return every other retained owner event relative to its own cutoff, ordered by `startsAt DESC, URI DESC`. Omitting `filter` shall preserve the approved all-events behavior and ordering. Each filtered cursor shall freeze and bind its filter, cutoff, and ordering; `limit` may change on later pages. Unknown, empty, or repeated `filter` values return standard `400 invalid_filter`; malformed cursors, an unfiltered cursor used with a filter, a filtered cursor used without its exact filter, or a cursor from the opposite filter return standard `400 invalid_cursor`. Unknown query parameters retain the endpoint's existing ignored behavior. | Server-side filtering avoids loading complete event history and defines exact admission and independent stable-pagination semantics. | Review feedback Q10, approved AppView API | AC-041 |
| RULE-001 | Business rule | Must | Flutter shall treat account type as presentation state, not verification, subscription, entitlement, ranking, reach, moderation priority, or permission. | Preserves approved product policy. | Approved AppView requirements | AC-002, AC-035 |
| RULE-002 | Business rule | Must | Switching to regular shall retain but hide business declaration, products, and events; it shall never issue declaration/event deletion requests. | Implements confirmed retention behavior. | Q3 | AC-010 |
| RULE-003 | Business rule | Must | Product prices are seller-authored display values; the UI shall make no inventory, availability, synchronization, tax, shipping, checkout, or accuracy claim. | Avoids implied native commerce guarantees. | Approved AppView requirements | AC-022 |
| RULE-004 | Business rule | Must | Flutter shall use only AppView-hydrated outbound destinations and shall not fetch, preview, resolve, normalize, or construct destinations from raw omitted data. | Preserves SSRF/phishing/privacy boundaries. | Approved AppView requirements | AC-012, AC-022, AC-029 |
| RULE-005 | Business rule | Must | Owner management UI is shown only for the active account and only while that account is business; visitor event detail/reporting remains governed by AppView visibility rather than owner-management UI state. | Separates client discoverability from backend eligibility. | Q6, NG-006 | AC-011, AC-030 |
| NFR-001 | Non-functional | Must | Business models, providers, drafts, pagination, and optimistic/reconciled state shall be scoped by the active account and target identity so switching accounts cannot display or mutate another account's business data. | Craftsky supports multiple signed-in accounts. | Codebase | AC-036 |
| NFR-002 | Non-functional | Must | All new user-visible text, catalog labels, validation, lifecycle/suppression reasons, dates, prices, empty states, and accessibility labels shall use generated localization; date, time, and currency display shall respect the active locale while mutation values retain canonical wire forms. | Maintains internationalization and wire correctness. | Codebase | AC-037 |
| NFR-003 | Non-functional | Must | New screens, forms, tabs, cards, controls, and dialogs shall be keyboard/screen-reader accessible, expose meaningful semantics and focus order, meet existing touch-target conventions, and adapt without overflow on supported mobile and wider layouts. | Maintains accessible responsive UI quality. | Codebase, frontend constraints | AC-038 |
| NFR-004 | Non-functional | Must | Business UI shall preserve blocked-shell, moderation, authentication/device, error-envelope, and request-ID behavior and shall not reveal account type, business data, event existence, or owner controls beyond AppView responses. | Prevents alternate client leakage. | Approved AppView requirements | AC-008, AC-030, AC-039 |
| NFR-005 | Non-functional | Must | The feature shall add no runtime fetch of authored destinations and no telemetry containing full destinations, email addresses, free text, product/event titles, prices, or locations. Client error reporting shall follow existing redaction patterns. | Preserves approved security and observability constraints. | Approved AppView requirements | AC-040 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002 | Given a current member on Account settings, when they select business or regular and the AppView succeeds, then the control and active account state show the returned authoritative type; duplicate taps cannot issue concurrent changes. |
| AC-002 | BR-002, FR-005, RULE-001 | Given a normally visible business profile, then it has a textual Business label without verified/subscribed/endorsed treatment and shows available tagline/action near ordinary profile information. |
| AC-003 | BR-002, FR-006 | Given eligible declaration details, then About displays each available detail with localized known values and safe unknown fallbacks while absent details consume no empty section space. |
| AC-004 | BR-003, FR-011 | Given a business owner, then Settings > Business > Products supports add/edit/remove/reorder and persists up to four valid cards in authored order. |
| AC-005 | BR-003, FR-014 | Given a business owner, then Settings > Business > Events provides Upcoming and History views, retains the selected view while managing events, and permits management of every server-returned lifecycle/suppression state with understandable diagnostics. |
| AC-006 | BR-004, FR-013 | Given a visitor-visible product, activating its card launches its hydrated external URI and no native catalog, checkout, or product-detail route is offered. |
| AC-007 | BR-004, FR-017 | Given visitor-visible upcoming events, the tab renders them in server order and can traverse all pages without client reordering. |
| AC-008 | FR-001, NFR-004 | Normal regular/business profiles, optional/partial business declarations, unknown safe values, and blocked shells decode without crash; blocked shells render no business label/details/tabs. |
| AC-009 | FR-002 | Given an account-type request failure, the prior type remains selected, the control becomes usable again, and established error feedback is shown. |
| AC-010 | FR-003, RULE-002 | Given a business account with retained declaration/products/events, switching to regular succeeds without confirmation or delete requests and removes business UI; switching back and refreshing restores eligible retained data. |
| AC-011 | FR-004, RULE-005 | Business settings rows and owner controls appear only for the active business account; a regular account reaching an owner-management deep link cannot manage records and returns to settings. |
| AC-012 | FR-005, RULE-004 | Each supported primary action has a localized action label, launches only its hydrated HTTPS or mailto destination, and surfaces launcher failure without fetching or rewriting the destination. |
| AC-013 | FR-006 | Location shows only hydrated locality/country; unknown business types/offerings remain readable but receive no unsupported inferred behavior. |
| AC-014 | FR-007 | A regular profile keeps exactly the existing ordinary tabs; every normally visible business profile has Products and Upcoming Events even when either is empty; blocked profiles show neither. Owner empty states may link to management, while visitor empty states expose no owner control. |
| AC-015 | FR-007 | Tab selection and per-tab scroll state remain stable across ordinary navigation and account-type-based tab construction, and blocked profiles never initialize business tab content. |
| AC-016 | FR-008 | Opening Edit Profile as regular shows only existing fields; opening it as business shows every approved business-detail field prefilled from current state with client guidance/validation matching server limits. |
| AC-017 | FR-009 | Saving only ordinary changes sends no business mutation; saving only business changes sends no ordinary mutation; saving both sends one mutation for each changed record. |
| AC-018 | FR-009, FR-012 | Business detail or product saves send the current complete known declaration and correct CID/create wildcard so changing one area preserves the other area and server-preserved unknown extensions. |
| AC-019 | FR-010 | If ordinary save succeeds and business save fails, or vice versa, the successful values become the new baseline, only failed values remain dirty, an accurate partial-failure message is shown, and retry resubmits only the failed record. |
| AC-020 | FR-011, FR-020 | Product editing rejects a fifth card and missing/invalid required values, uploads a valid image through the existing path, preserves an untouched image, and supports optional alt and canonical price input. |
| AC-021 | FR-012 | A stale declaration CID returns conflict feedback, performs no blind overwrite, and lets the owner reload current details/products before retrying. |
| AC-022 | FR-013, RULE-003, RULE-004 | Product cards preserve authored order, show normalized images/title and locale-formatted optional authored price, make no commerce guarantees, and use only hydrated external links. |
| AC-023 | FR-014 | At each view's own frozen first-page cutoff, Upcoming pagination includes active scheduled events nearest-first, including publicly suppressed events with diagnostics, and History includes ended, cancelled, postponed, unknown-status, and otherwise non-active events most-recent-first. No cross-view snapshot or exact-union guarantee is asserted across independently loaded cutoffs. Refresh and successful lifecycle mutations restart affected views from page one, and all approved suppression/exclusion reason codes map to bounded localized copy without becoming editable fields. |
| AC-024 | FR-015 | Event creation collects every required first-party field, omits client `createdAt`, validates end after start and duration/all-day/timezone constraints, and sends canonical whole-second UTC instants. |
| AC-025 | FR-015, FR-020 | Event editing prefills the current event, omits `createdAt` from the update, preserves an untouched optional image, and can replace or remove optional fields without dropping required fields. |
| AC-026 | FR-016 | Event update/delete sends the current CID; stale CID produces reload/retry conflict UX and never silently overwrites newer state. |
| AC-027 | FR-016 | Event deletion requires explicit confirmation; cancellation/postponement remains an edit, and successful changes remove/update the event consistently across manager, upcoming tab, and detail caches. |
| AC-028 | FR-017 | Upcoming Events is present for every normally visible business profile and handles initial loading, owner-empty setup CTA, visitor-empty text, retryable initial failure, incremental failure with confirmed rows retained, page loading, refresh failure with confirmed rows retained, and end of list. Flutter renders the AppView-returned public set without lifecycle/status re-filtering. |
| AC-029 | FR-018, RULE-004 | Selecting an event opens the DID/rkey detail, renders all available hydrated fields, omits absent optional controls, and launches only available hydrated event/registration URIs. |
| AC-030 | FR-019, RULE-005, NFR-004 | A visitor can submit the existing report reasons for a visible event and receives established success/failure feedback; a 404 due to absence/block/moderation shows an unavailable state and no event data. |
| AC-031 | FR-020 | Product/event image upload enforces existing image constraints and exposes progress/retry; cancelling or failing upload cannot accidentally replace the saved image. |
| AC-032 | FR-021 | A normally visible eligible business projection includes its declaration CID; no CID is added to blocked shells or a missing declaration. Flutter can create with `*` and replace with the returned current CID. |
| AC-033 | FR-021 | Profile product images and owner-list, public-list, and detail event images match the exact `PostImageView` JSON contract. Safe JPEG/PNG/WebP include required `cid`, `mime`, nonnegative `size`, `alt`, `thumb`, `fullsize`, and optional valid `aspectRatio`; unsafe, unsupported, malformed, or missing images omit the object; source fields reconstruct an unchanged mutation blob without URLs or PDS reads. |
| AC-034 | FR-022 | Following a successful create/update/delete, CID-identity reconciliation retains accepted local state for reads at the exact pre-write CID, settles on the accepted CID or absence, adopts any different CID as concurrent authoritative divergence, retains accepted state on refresh failure, and ignores superseded/account-mismatched completions. No CID chronology is inferred. |
| AC-035 | RULE-001 | Account type and business data produce no client feature entitlement, ranking, reach, verification, or moderation-priority behavior beyond the specified presentation/management UI. |
| AC-036 | NFR-001 | Switching between two signed-in accounts with different types/drafts/pages never displays, submits, or restores the previous account's business data, CID, event cursor, or pending mutation. |
| AC-037 | NFR-002 | All new visible copy and semantic labels are localized; dates/times/prices follow the active locale, while API bodies retain canonical status/catalog/currency/timestamp values. |
| AC-038 | NFR-003 | Widget tests and manual checks at narrow and wide constraints find no overflow; keyboard focus, screen-reader labels, buttons, tab semantics, validation errors, images, and destructive confirmations are understandable and operable. |
| AC-039 | NFR-004 | Unauthenticated/non-current requests retain existing routing/API handling, standard AppView errors remain consumable, and business UI never manufactures content omitted by eligibility or moderation. |
| AC-040 | NFR-005 | Tests or inspection confirm no client network request resolves/previews authored destinations and captured logs/errors contain none of the prohibited authored values. |
| AC-041 | FR-014, FR-023 | Each filtered traversal uses its own first-page cutoff: `filter=upcoming` returns `startsAt ASC, URI ASC` pages and `filter=history` returns `startsAt DESC, URI DESC` pages, with every record classified exactly once for that traversal's cutoff. Omitted filter retains approved all-events behavior. Later pages retain their traversal's cutoff/filter/order and may change `limit`. Unknown/empty/repeated filters return `400 invalid_filter`; malformed or incompatible filtered/unfiltered/opposite-filter cursors return `400 invalid_cursor`; unknown query parameters remain ignored. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Business account has no declaration | Show Business label and owner business tabs/settings; business detail editor starts empty and first declaration save uses `If-Match: *`. | FR-005, FR-007, FR-009 |
| EC-002 | Business account has declaration but no products/events | Owner sees both setup empty states; visitors see both explanatory empty states without management controls. | FR-007 |
| EC-003 | Visitor event request fails | Keep Upcoming Events visible. Initial failure shows retryable error rather than empty; refresh or incremental failure retains confirmed items and offers retry. | FR-007, FR-017 |
| EC-004 | Account changes while a business form has unsaved work | Existing unsaved-work guard applies before activation; drafts and CIDs never transfer to the newly active account. | NFR-001 |
| EC-005 | One half of combined profile save conflicts and the other succeeds | Keep the success, reload only the conflicted record on owner request, and do not revert or duplicate the successful request. | FR-010 |
| EC-006 | Product manager and profile editor concurrently change the declaration | The stale CID loses; no last-write-wins overwrite occurs, and reload presents the current complete declaration. | FR-012 |
| EC-007 | Primary action or product/event URI is absent from hydrated response | Omit the action; never reconstruct it from raw source or offer a broken control. | RULE-004 |
| EC-008 | Product/event image has unsupported MIME, malformed source, or cannot produce both display URLs | Omit the image object, keep the containing product/event safe and usable where its remaining required projection is valid, and never ask Flutter to contact a PDS. | FR-021 |
| EC-009 | Event is ongoing | Include it in Upcoming Events according to AppView response and present start/end accurately. | FR-017 |
| EC-010 | Event is cancelled or postponed | Exclude it from public upcoming through AppView, place it in owner History, and let the owner edit it back to scheduled with valid future/ongoing dates so it moves to owner Upcoming. | FR-014, FR-015, FR-023 |
| EC-011 | Event becomes unavailable while detail is open | A refresh/action receiving not found replaces content with an unavailable state and offers no report/external action for stale data. | FR-019 |
| EC-012 | Last product is removed or every business field is cleared | Persist a valid declaration with the remaining known fields (including an empty declaration when appropriate); do not change account type or events. | FR-009, FR-012 |
| EC-013 | External application cannot handle a hydrated URI | Keep the app stable and show established action-failed feedback. | FR-013, FR-018 |
| EC-014 | All-day event crosses daylight-saving time | Form conversion retains local-midnight boundaries in the selected IANA timezone and sends the correct exclusive UTC end. | FR-015 |
| EC-015 | Unknown safe catalog value arrives from an independent client | Display a safe fallback label, preserve it during unrelated declaration updates where the API contract permits, and never offer it as a first-party selectable value. | FR-001, FR-006, FR-009 |
| EC-016 | Future scheduled event is publicly suppressed | Keep it in owner Upcoming with suppression diagnostics rather than misclassifying it as historical. | FR-014, FR-023 |
| EC-017 | Event changes management partition after edit or time passes | A successful lifecycle mutation invalidates and restarts affected views; passive time movement appears after that view is refreshed. Independently loaded views may transiently overlap or omit the event because they do not share a cutoff; no single-view traversal duplicates it. | FR-016, FR-022, FR-023 |

## 15. Data / Persistence Impact

- New Flutter models/state: account type, business profile/declaration, open catalog value, action, location, money, product, event, event page, event diagnostics, and normalized business image projection.
- New local durable persistence: None identified. Drafts remain in-memory under existing unsaved-work behavior unless coding design finds an established draft mechanism that must be reused.
- AppView response changes:
  - Eligible `business` profile projection includes current declaration `cid`.
  - Product and event images use the exact reusable `PostImageView` display/write-round-trip object defined in FR-021.
- AppView owner-list query change: `GET /v1/events` accepts optional `filter=upcoming|history`; the unfiltered all-events contract remains available.
- Changed mutation behavior: None. Existing account-type, declaration, event, image-upload, and report routes remain authoritative.
- Migration required: No.
- Lexicon change required: No.
- Backwards compatibility: The app is not in production. AppView changes remain restricted to client-enabling response projection and an additive owner-list filter; they do not alter source records. Blocked-shell omission and unfiltered owner-list behavior remain unchanged.

## 16. UI / API / CLI Impact

- UI:
  - Account settings gains a regular/business selector.
  - Main Settings conditionally gains Business > Events and Products.
  - Edit Profile conditionally gains business detail fields under one Save action.
  - Business profile summary/About gain business presentation.
  - Profile tab composition gains always-present Products and Upcoming Events for normally visible business profiles.
  - New Products manager, two-view Events manager/editor, product cards, event cards, and event detail/report surfaces.
- API:
  - Flutter integrates the approved existing routes: account type, declaration PUT, profile upcoming events, owner events, event CRUD/detail/report, and image upload.
  - Existing eligible business response adds declaration CID.
  - Existing product/event image responses become normalized client views with display URLs and round-trip metadata.
  - Existing owner event collection gains optional `filter=upcoming|history` with filter-bound cursors; no filter retains all-events behavior.
  - No new route is required.
- CLI: None identified.
- Background jobs: None identified.

## 17. Security / Privacy / Permissions

- Authentication: All reads/writes retain existing Craftsky session, device, and current-member policy.
- Authorization: The client exposes owner management only for the active business account. AppView remains authoritative and rejects cross-owner mutation.
- Sensitive data: Business records are public under the approved backend design. Per Q7, this slice adds no special publication warning. Client logs/error reports must still redact authored destinations, email, text, prices, and locations.
- Outbound links: Launch only AppView-hydrated destinations. Do not prefetch, resolve, preview, canonicalize, or contact them in the background.
- Block/moderation: Omitted profile fields, empty event lists, and event not-found responses remain information boundaries, not recoverable client errors containing hidden data.
- Abuse cases: Misleading business claims, phishing links, impersonation, event spam, and stale prices remain handled through self-declared labeling, safe AppView hydration, external launching, and event reporting/moderation.

## 18. Observability

- Events: Existing provider/API error reporting may record bounded operation/result classes for account-type, declaration, product, event, report, and external-launch outcomes.
- Logs: Do not include authored free text, titles, locations, price values, email addresses, or full external destinations.
- Metrics: No new product metric is required. Existing request/error/performance instrumentation may distinguish bounded screen/operation names without authored-value labels.
- Alerts: None identified beyond existing AppView/client error monitoring.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | One Save spans independently versioned ordinary and business records. | A generic failure could mislead the owner or cause duplicate retries. | Changed-record dispatch, per-record outcomes, baseline reconciliation, and partial-success tests. |
| RISK-002 | Products and business details share one declaration. | Editing one surface could erase the other or unknown portable values. | Full known-field replacement, current CID, server extension merge, and concurrent-editor conflict tests. |
| RISK-003 | AppView projection lags a successful PDS mutation. | Saved data may flicker back or appear lost. | CID-identity overlays, account/request-generation fencing, explicit divergence handling, and no CID ordering. |
| RISK-004 | Dynamic tabs change controller length or selection. | Crashes, tab jumps, or lost scroll state. | Derive one stable tab list per resolved state and test owner/visitor/content transitions. |
| RISK-005 | Event date/time/timezone conversion is incorrect. | Events publish at wrong instants or fail server validation, especially around DST/all-day boundaries. | Canonical conversion layer and focused timezone/DST/widget tests. |
| RISK-006 | Multi-account caches retain declaration CIDs or drafts. | One member could see or attempt to mutate another active account's data. | Account-key every provider/draft and use existing account-boundary invalidation/unsaved-work guard. |
| RISK-007 | Business label resembles verification. | Visitors may infer endorsement. | Plain localized Business text, no verification icon, no verified wording or privilege. |
| RISK-008 | External links or seller-authored prices are misleading. | Phishing or commerce confusion. | Hydrated destinations only, external-link affordances, report flow for events, and no commerce guarantees. |
| RISK-009 | Minimal AppView image alignment accidentally changes source/mutation shape. | Existing validation or record round-trip could regress. | Normalize response only, retain sufficient blob metadata, and add AppView response/mutation contract tests. |
| RISK-010 | Integrated scope is broad. | Regression risk in mature profile/settings/navigation code. | TDD coverage across models, APIs, providers, widgets, routes, account switching, and existing profile/settings suites. |
| RISK-011 | Independently loaded Upcoming and History views use different time cutoffs. | An event crossing the boundary can temporarily appear in both views or neither until refresh. | Make the non-snapshot behavior explicit, guarantee only each single traversal, and restart affected views after lifecycle mutation or user refresh. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The approved AppView routes and validation behavior are complete except for declaration CID, normalized image response data, and the filtered owner-event views specified here. | Additional backend prerequisite requirements would be needed. |
| ASM-002 | Existing image upload can be reused for product and event images without a new media endpoint. | A separate media contract and API change would expand scope. |
| ASM-003 | One normalized AppView image response shape can support both rendering and reconstruction of unchanged mutation payloads. | Separate owner/public image shapes or an owner record GET may be needed. |
| ASM-004 | The active account's profile is the authoritative client source for account type and eligible declaration details. | A dedicated owner business GET route may be required. |
| ASM-005 | A country selector, known catalog selectors, and an IANA-timezone-capable event form can be implemented within normal Flutter coding design without product-level choices. | Requirements would need a follow-up product decision about free text, catalogs, or timezone limitations. |
| ASM-006 | Event deletion confirmation follows existing Craftsky destructive-action style; product removal does not require a separate confirmation because the manager Save remains explicit. | Product removal confirmation behavior would need a UX decision. |
| ASM-007 | No visible business badge outside the full profile is required in this slice even though AppView summaries already include `accountType`. | Feed/search/notification widgets would add substantial presentation and regression scope. |

## 21. Open Questions

- None blocking requirements design.
- Non-blocking coding-design detail: choose the timezone data/provider and selector interaction that can satisfy FR-015 and DST acceptance coverage.
- Non-blocking visual-design detail: choose card density, image aspect treatment, and responsive form grouping while preserving the specified content, semantics, and tab rules.

## 22. Review Status

Status: Draft

Risk level: Medium

Review recommended: Yes

Reviewer:

Date: 2026-08-29

Notes: Review is recommended because the slice adds broad public CRUD UI, independently versioned combined saves, dynamic tabs, date/time conversion, external actions, and minimal AppView API changes. It does not change lexicons, persistence, authentication, permissions, or public eligibility, so explicit high-risk approval is not required before test design.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-08-29-flutter-business-profiles/01-requirements.md`
- Next test specification: `docs/changes/2026-08-29-flutter-business-profiles/02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001 through BR-004; FR-001 through FR-023; RULE-001 through RULE-005; NFR-001 through NFR-005.
- Suggested test levels: Dart model/serialization unit tests; API-client contract tests; provider/controller unit tests; widget tests for settings, profile tabs/presentation, editors, two-view event management, managers, cards, detail, errors, and accessibility; router tests; AppView Go response/contract tests for CID, normalized images, owner-event filters, and filter-bound cursors; multi-account and partial-save integration-style widget/provider tests; regression suites for existing profile/settings/router behavior.
- Blocking open questions: None.
