# Coding Plan: Flutter Business Profiles

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, 2026-08-30)
- Backend source contract: `docs/changes/2026-08-27-business-profiles/01-requirements.md`
- API architecture: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Risk level: Medium
- Scope constraints: no lexicon, migration, authentication, permission, eligibility, persistence, or new-route change on AppView; no direct PDS access or PDS token in Flutter.

## 2. Implementation Strategy

Implement the approved slice in vertical TDD stages, fixing the three AppView read-contract prerequisites before adding Flutter consumers:

1. Add declaration CID and exact reusable image views to business profile/event responses.
2. Add independent, cutoff-bound owner Upcoming/History filters while preserving unfiltered behavior.
3. Add a dedicated Flutter `business/` feature for models, HTTP/repository access, account-scoped state, forms, managers, and presentation components.
4. Integrate account type into Account settings, business rows into Settings, business summary/tabs into Profile, and business-detail fields into the existing combined Edit Profile dialog.
5. Add product management, then event management/detail/reporting, followed by localization, accessibility, privacy, and regression passes.

The AppView will keep `internal/business` independent of `internal/api`. Business hydration will return validated source image metadata; API response builders will convert it to the existing `api.PostImageView` and synthesize display URLs with the owner DID. This avoids an import cycle and a duplicate wire format.

Flutter read models and write drafts will remain separate. Hydrated open values and display URLs must never leak into mutation bodies. All providers will use the existing repository-provider and active-account operation-lease patterns. Accepted declaration/event writes will additionally use CID-identity overlays so lagging AppView reads cannot visibly restore pre-write state.

The existing Edit Profile dialog remains the single ordinary/business detail editor. It will embed a bounded business-fields section when the authoritative account type is business and dispatch independent ordinary and declaration writes through one combined-save controller. Products remain on their own settings manager but are included unchanged in every declaration replacement.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView profile projection | `business.Store.ReadEligibleProfile` hydrates `ProfileView` from `raw_record` | Select indexed CID, attach it to the eligible view, and expose it only inside normal `business` projection | FR-021, NFR-004 | AT-013, IT-002, UT-001 |
| AppView image projection | Business views expose validated raw image JSON; posts expose `PostImageView` | Hydrate validated source metadata and project one `PostImageView` across product profile and event owner/public/detail surfaces | FR-020, FR-021 | AT-013, UT-014, IT-002 |
| AppView owner event list | One descending unfiltered SQL traversal and management cursor | Add exact `filter` admission, three owner query modes, independent filtered cutoffs, and kind-bound cursors | FR-014, FR-023 | AT-006, UT-007, IT-003, REG-006 |
| Flutter models | `dart_mappable` full `Profile` and generated mappers | Add nullable full-profile account type/business projection plus business/event/read-write models | FR-001, FR-008, FR-011, FR-015, FR-021 | UT-001, UT-004, UT-006, UT-008, UT-009, UT-014 |
| Flutter data layer | Dio client, repository interface, API implementation, Riverpod DI | Add exact account/declaration/event/report operations and query/header serialization | FR-002, FR-009, FR-012, FR-015, FR-016, FR-018, FR-019, FR-023 | IT-001, IT-008, IT-009 |
| Account-scoped state | Generated Riverpod providers plus `captureActiveAccountOperation` | Add business repository invalidation, list/detail controllers, combined save, account-type mutation, and generation-fenced overlays | FR-003, FR-010, FR-016, FR-022, NFR-001 | UT-005, UT-010, UT-011, UT-015, IT-004, IT-005, IT-010, IT-013 |
| Account/settings UI | Account deletion page; static settings sections and typed routes | Add account selector; conditionally add Business > Events/Products; guard owner routes | FR-002, FR-004 | AT-001, UT-018, IT-011 |
| Profile UI | Fixed five-tab `DefaultTabController`, summary/meta/About slivers | Derive one stable tab list, add business summary/About, product cards, and always-present business Upcoming Events | FR-005, FR-006, FR-007, FR-013, FR-017 | AT-002, AT-003, AT-009, UT-002, UT-003, REG-001, REG-005 |
| Combined editor | One ordinary `FormBuilder` and `SaveProfile` notifier | Embed business-detail fields; dispatch changed records independently; reconcile baselines per outcome | FR-008, FR-009, FR-010 | AT-004, UT-004, UT-005, IT-005, REG-002 |
| Product management | No Flutter surface; blob upload exists | Add ordered manager/editor, full declaration replacement, upload/preview, conflict reload | FR-011, FR-012, FR-020 | AT-005, UT-006, IT-006, IT-007 |
| Event management/detail | No Flutter surface; AppView routes exist | Add Upcoming/History manager, event form, detail route, external actions, deletion, and reporting | FR-014 through FR-019 | AT-006 through AT-010, UT-007 through UT-010, IT-008, IT-009 |
| Localization/time/country | English ARB and `intl`; no IANA database | Add generated English business copy, pure-Dart IANA conversion, and localized ISO country labels without adding another supported language | FR-015, NFR-002 | AT-007, AT-012, UT-003, UT-008, UT-013, UT-016 |
| Privacy/accessibility | Existing semantics, launcher adapter, Sentry redaction patterns | Localize external confirmation, handle launcher failure, add bounded screen/layout/semantics/privacy tests | RULE-004, NFR-003, NFR-005 | AT-012, AT-014, UT-017, REG-010, MAN-001, MAN-002 |

## 4. Files And Modules

### 4.1 AppView

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/business/profile.go` | Change | Add `CID` to `ProfileView`; replace product raw image with nullable validated `HydratedImage`; preserve declaration merge behavior | FR-009, FR-021 | IT-002, IT-014, REG-011 |
| `appview/internal/business/event.go` | Change | Replace event raw image with nullable `HydratedImage`; parse safe blob CID/MIME/size/alt/aspect once | FR-021 | IT-002, UT-014 |
| `appview/internal/business/product.go` | Change | Define/reuse `HydratedImage` source metadata and retain approved validation | FR-020, FR-021 | UT-014, IT-002 |
| `appview/internal/business/store.go` | Change | Select declaration CID; add owner event filter input and direction-specific SQL queries | FR-014, FR-021, FR-023 | UT-007, IT-002, IT-003, REG-006 |
| `appview/internal/api/post_response.go` | Change | Keep `PostImageView` as the one wire object; make required metadata serialize at zero values where contract requires; expose a small validated source-to-view builder | FR-021 | AT-013, IT-002, existing post image regressions |
| `appview/internal/api/profile_response.go` | Change | Add API business/product response projection and build product image URLs with profile DID | FR-005, FR-021 | IT-002 |
| `appview/internal/api/profile.go` | Change | Convert eligible domain business view into API response view before writing profile JSON | FR-021, NFR-004 | IT-002, IT-012 |
| `appview/internal/api/business_event.go` | Change | Parse exact filter contract; map event domain views to API views on detail and both lists | FR-014, FR-021, FR-023 | AT-006, AT-013, IT-002, IT-003 |
| `appview/internal/api/business_event_cursor.go` | Change | Add owner-upcoming/history cursor kinds with required `asOf`; keep owner-unfiltered and public-upcoming kinds isolated | FR-023 | IT-003, REG-006 |
| `appview/internal/api/business_profile_acceptance_test.go` | Change | Assert declaration CID and exact product image JSON/omission | FR-021 | IT-002, AT-013 |
| `appview/internal/api/business_event_acceptance_test.go` | Change | Assert exact image view on public list and detail | FR-021 | IT-002, AT-013 |
| `appview/internal/api/business_event_management_acceptance_test.go` | Change | Assert owner image projection and real-Postgres independent filtered traversals | FR-014, FR-021, FR-023 | AT-006, IT-002, IT-003 |
| `appview/internal/api/business_event_pagination_test.go` | Change | Cover cutoff freezing, later-page limits, all filter/cursor admission combinations, and unknown params | FR-023 | IT-003 |
| `appview/internal/api/business_event_cursor_test.go` | Change | Round-trip all cursor kinds and reject cross-kind/scope/tampered payloads | FR-023 | IT-003 |
| `appview/internal/api/business_profile_test.go` | Change | Prove unknown nested extensions/large integers survive detail-only and product-only replacements | FR-009, FR-012 | IT-014, REG-011 |
| `appview/internal/business/profile_merge_test.go` | Change | Add focused merge cases and adapt hydrated-image assertions | FR-009, FR-021 | IT-014, REG-011 |
| Existing post/profile/event route and response tests | Change only as needed | Protect post image JSON, blocked shells, unfiltered owner events, and public eligibility | FR-021, FR-023, NFR-004 | REG-004, REG-006, REG-007 |

No migration, query generation, route registration, indexer, lexicon, or generated lexicon file changes are planned. Business event SQL is currently handwritten in `internal/business/store.go`; keep it there for this slice.

### 4.2 Flutter Models And Data

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/profile/models/profile.dart` | Change | Add nullable `AccountType` and `BusinessProfile`; omission remains distinct from regular | FR-001 | UT-001, IT-012 |
| `app/lib/business/models/business_profile.dart` | Create | Account enum, open value, declaration, location, action, product, price, and normalized image read models | FR-001, FR-005, FR-006, FR-011, FR-021 | UT-001, UT-003, UT-006, UT-014 |
| `app/lib/business/models/business_event.dart` | Create | Event/event-page models with typed DID/rkey/URI/CID and diagnostic arrays | FR-014 through FR-019 | UT-001, UT-007, UT-009, UT-010 |
| `app/lib/business/models/business_drafts.dart` | Create | Declaration/product/event write drafts, validation, complete known-field serialization, and image mutation reconstruction | FR-008, FR-009, FR-011, FR-015, FR-020 | UT-004, UT-006, UT-008, UT-009, UT-014 |
| `app/lib/business/models/business_state.dart` | Create | Page state, management filter, record keys, accepted overlays, and combined-save outcomes/baselines | FR-010, FR-014, FR-022 | UT-005, UT-010, UT-015 |
| `app/lib/business/data/business_api_client.dart` | Create | Exact Dio methods for all approved routes, headers, query parameters, and errors | FR-002, FR-009, FR-012, FR-015, FR-016, FR-019, FR-023 | IT-001, IT-008, IT-009 |
| `app/lib/business/data/business_repository.dart` | Create | Mockable read/write interface using typed models/drafts | All data FRs | IT-004 through IT-010 |
| `app/lib/business/data/api_business_repository.dart` | Create | Delegate interface to API client without UI/state logic | All data FRs | API repository tests |
| `app/lib/business/providers/business_repository_provider.dart` | Create | Construct Dio client/repository with existing providers | NFR-001 | IT-010 |
| `app/lib/bootstrap.dart` | Change | Register new root mappers | FR-001, FR-021 | UT-001 |
| Generated `*.mapper.dart`, `*.g.dart` | Generate | `dart_mappable`, Riverpod, and typed-route outputs | All Flutter FRs | Build/analyze regression |

Read and write types must remain separate:

```text
BusinessImageView
  cid, mime, size, alt, aspectRatio, thumb, fullsize

BusinessImageDraft.toJson()
  image: {$type: blob, ref: {$link: cid}, mimeType: mime, size: size}
  alt, aspectRatio
  // never thumb/fullsize

BusinessProfileDraft.toJson()
  all known declaration fields, including unchanged products

BusinessEventDraft.toCreateJson()/toUpdateJson()
  all writable event fields
  // never did/rkey/uri/cid/createdAt/past/diagnostics
```

### 4.3 Flutter State And UI

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/business/providers/account_type_controller.dart` | Create | Pending/success/failure selector mutation and cache reconciliation | FR-002, FR-003 | AT-001, UT-011, IT-004 |
| `app/lib/business/providers/business_projection_overlay_provider.dart` | Create | Account/record-keyed CID-identity overlay registry for declarations/events | FR-022, NFR-001 | UT-015, IT-010, IT-013 |
| `app/lib/business/providers/profile_business_events_provider.dart` | Create | Public paginated upcoming state keyed by viewer account and target identity | FR-017 | AT-009, UT-010, IT-009 |
| `app/lib/business/providers/owner_business_events_provider.dart` | Create | Independent owner Upcoming/History states and cursors | FR-014, FR-023 | AT-006, UT-010, IT-003 |
| `app/lib/business/providers/business_event_detail_provider.dart` | Create | Read one DID/rkey event and apply accepted/deleted overlay | FR-018, FR-019, FR-022 | AT-009, AT-010, IT-009, IT-013 |
| `app/lib/business/providers/business_event_mutation_controller.dart` | Create | Create/update/delete with conflicts, overlays, and targeted invalidation | FR-015, FR-016, FR-022 | AT-007, AT-008, IT-008, IT-013 |
| `app/lib/business/providers/products_controller.dart` | Create | Full declaration replacement, conflict reload, and accepted declaration overlay | FR-011, FR-012, FR-020, FR-022 | AT-005, IT-006, IT-007, IT-013 |
| `app/lib/business/providers/report_business_event_provider.dart` | Create | Account-guarded event report mutation | FR-019 | AT-010, IT-009 |
| `app/lib/profile/providers/save_profile_provider.dart` | Change | Dispatch ordinary/business changes independently and return `CombinedProfileSaveResult` | FR-009, FR-010, FR-022 | AT-004, UT-005, IT-005 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Advance overlay generation and invalidate all business repositories/providers/controllers | NFR-001 | AT-011, IT-010, REG-008 |
| `app/lib/settings/pages/account_page.dart` | Change | Add authoritative regular/business selector above deletion | FR-002, FR-003 | AT-001 |
| `app/lib/settings/pages/settings_page.dart` | Change | Conditionally render Business heading with Events and Products rows | FR-004 | AT-001, REG-003 |
| `app/lib/settings/models/settings_row.dart` | Change | Add stable Events/Products row IDs | FR-004 | AT-001, REG-003 |
| `app/lib/profile/pages/edit_profile_dialog.dart` | Change | Embed business fields, track two baselines, and render partial-save feedback | FR-008 through FR-010 | AT-004, REG-002 |
| `app/lib/business/widgets/business_profile_fields.dart` | Create | Bounded reusable declaration detail fields, unknown-value preservation, and validation | FR-008, NFR-002, NFR-003 | AT-004, UT-003 |
| `app/lib/business/pages/products_settings_page.dart` | Create | Ordered product manager with loading/error/conflict/empty states | FR-011, FR-012 | AT-005, IT-006, IT-007 |
| `app/lib/business/widgets/product_editor.dart` | Create | Add/edit/remove/reorder up to four products with image upload | FR-011, FR-020 | AT-005, UT-006, IT-007 |
| `app/lib/business/pages/events_settings_page.dart` | Create | Upcoming/History owner views, diagnostics, pagination, and create action | FR-014 | AT-006, UT-010 |
| `app/lib/business/pages/event_editor_dialog.dart` | Create | Full-screen modal create/edit form with unsaved-work guard | FR-015, FR-020 | AT-007, IT-008 |
| `app/lib/business/widgets/event_form.dart` | Create | Timed/all-day/timezone/status/role/location/action fields and validation | FR-015 | AT-007, UT-008, UT-009 |
| `app/lib/business/pages/event_detail_page.dart` | Create | Complete event detail, owner actions, visitor report, unavailable state | FR-018, FR-019 | AT-009, AT-010 |
| `app/lib/business/widgets/product_card.dart` | Create | Ordered external-commerce product presentation | FR-013 | AT-003, UT-012, UT-013 |
| `app/lib/business/widgets/event_card.dart` | Create | Public/owner event summary with mode/role/date/image/diagnostics variants | FR-014, FR-017 | AT-006, AT-009 |
| `app/lib/business/widgets/business_network_image.dart` | Create | Cached image, alt semantics, aspect handling, placeholder/error, optional gallery | FR-013, FR-017, FR-020 | AT-003, AT-009, AT-012 |
| `app/lib/profile/widgets/profile_tab_bar.dart` | Change | Add Products/Upcoming Events enum values and accept one explicit ordered list | FR-007 | UT-002, REG-001, REG-005 |
| `app/lib/profile/pages/profile_page.dart` | Change | Derive tabs once; wire product/event slivers; clamp selection across changes | FR-007, FR-013, FR-017 | AT-003, AT-009, REG-005 |
| `app/lib/profile/widgets/profile_meta_section.dart` | Change | Add plain Business label, tagline, and optional primary action | FR-005 | AT-002 |
| `app/lib/profile/widgets/profile_tabs/profile_about_tab.dart` | Change | Add optional full business sections | FR-006 | AT-002, UT-003 |
| `app/lib/profile/widgets/profile_tabs/profile_products_tab.dart` | Create | Owner/visitor product tab and owner setup state | FR-007, FR-013 | AT-003 |
| `app/lib/profile/widgets/profile_tabs/profile_events_tab.dart` | Create | Stable loading/error/empty/content/pagination sliver | FR-007, FR-017 | AT-009, UT-010 |
| `app/lib/moderation/widgets/report_subject_sheet.dart` | Change | Add event subject title/type | FR-019 | AT-010 |
| `app/lib/moderation/widgets/report_flow.dart` | Change | Add event report modal using event provider | FR-019 | AT-010, IT-009 |
| `app/lib/shared/link/external_link.dart` | Change | Localize confirmation and surface false/throw launch failures without destination rewriting | FR-005, FR-013, FR-018, RULE-004 | UT-012, UT-017, MAN-002 |
| `app/lib/router/route_locations.dart` | Change | Add business products/events child paths and `/events/:did/:rkey` | FR-004, FR-018 | UT-018, IT-011 |
| `app/lib/router/router.dart` | Change | Add typed settings routes and authenticated event detail route | FR-004, FR-018 | UT-018, IT-011 |
| `app/lib/l10n/app_en.arb` | Change | Add all business copy and remove hard-coded external-link copy | NFR-002 | AT-012 |
| `app/pubspec.yaml`, `app/pubspec.lock` | Change | Add `timezone ^0.11.1` and `l10n_countries ^2.0.3` | FR-015, NFR-002 | UT-003, UT-008, UT-013 |

The first-party image picker may continue producing JPEG/PNG only; WebP remains fully supported for hydrated independent records. Reuse `BlobApiClient.uploadImage` and the existing image preparation path, but add a business picker wrapper with the business 1000-character alt limit and aspect-ratio output rather than reusing the profile/composer 300-character policy.

## 5. Services, Interfaces, And Data Flow

### 5.1 AppView Response Projection

```text
PDS/indexed raw image
  -> business.parseHydratedImage(raw)
       validates blob/CID, MIME, size, alt, aspect ratio
       returns HydratedImage? (no display URL)
  -> api.buildBusinessImageView(ownerDID, HydratedImage)
       returns PostImageView with thumb/fullsize, or nil
  -> profile/event API response builder
  -> JSON
```

Partial signatures:

```go
type HydratedImage struct {
    CID         syntax.CID
    MIME        string
    Size        int64
    Alt         string
    AspectRatio *AspectRatio
}

func BuildBusinessProfileResponse(owner syntax.DID, view *business.ProfileView) *BusinessProfileResponse
func BuildBusinessEventResponse(view business.EventView) BusinessEventResponse
func buildBusinessImageView(owner syntax.DID, source *business.HydratedImage) *PostImageView
```

`PostImageView.Size` must serialize `0`; safe business images always have nonempty CID/MIME/thumb/fullsize. Existing post behavior for unsupported MIME remains unchanged, so business omission is enforced before the shared view is constructed.

### 5.2 Owner Event Filter Flow

```text
GET /v1/events
  -> inspect Query()["filter"] for presence/count/value
  -> normalize limit
  -> choose cursor kind: owner-unfiltered / owner-upcoming / owner-history
  -> decode cursor with owner-DID HMAC scope
  -> first filtered page: cutoff = now.UTC
     later filtered page: cutoff = cursor.asOf
  -> Store.ListOwnerEvents(filter, cutoff, seek, limit+1)
  -> hydrate + owner diagnostics
  -> API event projection
  -> encode next cursor with same kind/cutoff/order
```

Store query predicates:

```text
effectiveScheduled = COALESCE(event.status, 'scheduled') = 'scheduled'
upcoming = effectiveScheduled AND event.ends_at > cutoff
history = NOT (effectiveScheduled AND event.ends_at > cutoff)

upcoming seek/order: (starts_at, uri) > seek; ASC, ASC
history seek/order:  (starts_at, uri) < seek; DESC, DESC
unfiltered: preserve existing DESC query exactly
```

Public suppression is not part of owner partitioning. Every filtered traversal classifies against its own cutoff. Refresh and successful lifecycle mutations invalidate accumulated pages/cursors for affected views.

### 5.3 Flutter Repository Contract

```text
Future<AccountType> updateAccountType(AccountType value)
Future<RecordMutationResult> putBusinessProfile(BusinessProfileDraft draft, Cid? expectedCid)
Future<BusinessEventPage> listProfileEvents(AtIdentifier owner, {String? cursor, int limit = 10})
Future<BusinessEventPage> listOwnerEvents(OwnerEventFilter filter, {String? cursor, int limit = 20})
Future<BusinessEvent> getEvent(Did owner, RecordKey rkey)
Future<RecordMutationResult> createEvent(BusinessEventDraft draft)
Future<RecordMutationResult> updateEvent(Did owner, RecordKey rkey, Cid expectedCid, BusinessEventDraft draft)
Future<RecordMutationResult> deleteEvent(Did owner, RecordKey rkey, Cid expectedCid)
Future<ReportResult> reportEvent(Did owner, RecordKey rkey, ReportSubmission report)
```

API details:

- `PUT /v1/profiles/me/account-type`: `{accountType}`.
- `PUT /v1/profiles/me/business`: complete known declaration; `If-Match: *` for create or current CID for replace.
- Event create/update bodies omit `createdAt` and all response-only fields.
- Event update/delete use current CID precondition.
- List cursors are opaque strings and never parsed by Flutter.
- Only AppView-hydrated `https` or `mailto` destinations reach launcher adapters.
- Every method goes through existing Dio/session/device/error-envelope behavior; no PDS URL or token is constructed.

### 5.4 Combined Profile Save

The editor creates independent ordinary and business plans from current values and mutable baselines. Do not use fail-fast `Future.wait`.

```text
ordinaryPlan? ----> ProfileRepository.updateMe -----\
                                                   +-> CombinedProfileSaveResult
businessPlan? ----> BusinessRepository.putProfile -/

for each requested portion:
  success -> update that baseline and clear only that dirty portion
  failure -> preserve that draft/dirty portion and classify conflict/general error

both successful -> apply caches/overlay, close
partial success -> apply successful cache/overlay, keep dialog open, show exact failed portion
both failed -> retain both drafts/baselines and show combined failure
retry -> build plans only from still-dirty portions
```

The business plan always starts from the current complete known declaration and replaces only detail fields, retaining products and unknown open catalog values that the first-party UI cannot select. Unknown top-level extension fields remain AppView's merge responsibility.

### 5.5 Time, Country, Price, And Destinations

- Add pure-Dart `timezone` data initialization during app bootstrap and inject a `BusinessTimeZoneService` for tests.
- New events default to `UTC`; editing preserves the stored IANA zone. The user selects an IANA zone explicitly. Do not treat platform abbreviations as IANA IDs.
- Build timed instants and all-day local-midnight/exclusive-end boundaries as `TZDateTime`, then serialize UTC whole-second RFC3339 values.
- Use `l10n_countries` with the active locale for ISO 3166 country labels; preserve canonical alpha-2 codes in drafts/writes.
- Keep money amount as an authored canonical string. Format a separate display value with `intl`; never round-trip a formatted or binary-floating value.
- External actions pass the exact hydrated URI to confirmation/launcher. Business form validation requires credential-free HTTPS (or approved `mailto` for email action) without invoking network resolution.

## 6. State, Providers, Controllers, Or DI

### 6.1 Provider Graph

```text
dioProvider
  -> businessApiClientProvider
    -> businessRepositoryProvider

auth/session + userProfileProvider(me)
  -> accountTypeControllerProvider
  -> saveProfileProvider (ordinary + business details)
  -> productsControllerProvider
  -> ownerBusinessEventsProvider(filter)
  -> businessEventMutationControllerProvider
  -> reportBusinessEventProvider

userProfileProvider(target)
  -> profile tab specification
  -> profileBusinessEventsProvider(target)

businessEventDetailProvider(did, rkey)
ownerBusinessEventsProvider(filter)
userProfileProvider(target)
  <- businessProjectionOverlayProvider(account, recordKey)
```

### 6.2 Provider Choices

- `businessRepositoryProvider`: generated `Provider`, account-invalidated with the shared Dio dependency.
- `accountTypeControllerProvider`: generated `AsyncNotifier<AccountType?>`; idle value is null, duplicate selections are blocked while loading.
- `profileBusinessEventsProvider(target)`: generated `AsyncNotifier<BusinessEventListState>` family keyed by a typed target containing active viewer account and target identifier.
- `ownerBusinessEventsProvider(filter)`: generated `AsyncNotifier<BusinessEventListState>` family; Upcoming and History are independent instances.
- `businessEventDetailProvider(key)`: generated `FutureProvider` or `AsyncNotifier` family. Prefer `AsyncNotifier` if refresh/unavailable transition needs retained data; do not add a controller solely for naming symmetry.
- Mutation/report controllers: generated `AsyncNotifier` with idle nullable result, matching existing save/report patterns.
- `businessProjectionOverlayProvider(recordKey)`: generated `Notifier` family whose key includes active `AccountKey` plus declaration owner or event DID/rkey. It stores at most one current accepted generation per record.
- `saveProfileProvider`: retain the existing provider name and ordinary-only call compatibility; change result to a combined outcome model and add optional business plan.

### 6.3 CID-Identity Overlay State Machine

```text
accepted update/create:
  store {generation, preCid/preAbsent, acceptedCid, acceptedView}

read at exact preCid (or absent after create): keep acceptedView
read at acceptedCid: publish read and clear overlay
read at any third CID: publish read as divergence and clear overlay
read failure: retain acceptedView and expose retry metadata
explicit reload: warn, then clear overlay and fetch if confirmed

accepted delete:
  store tombstone {generation, deletedCid}
read at deletedCid: keep absent
read not-found: settle and clear tombstone
read at different CID: publish recreation/divergence and clear tombstone

completion with wrong account/session lease or superseded generation: ignore
```

No timeout may silently discard accepted state. If a background retry policy is added during implementation, it may only trigger reads; it may not change these authority transitions.

### 6.4 Invalidation Matrix

| Mutation | Immediate state | Invalidated / reconciled state |
|---|---|---|
| Account type | Patch active full-profile cache to returned type; regular hides business projection | Active profile aliases, Settings, public/owner event families; switch back refetches retained eligible data |
| Declaration details | Accepted declaration overlay with submitted details and unchanged products | Full-profile aliases and Products manager |
| Products | Accepted declaration overlay with submitted ordered products and unchanged details | Full-profile aliases and combined editor baseline |
| Event create/update | Accepted event overlay | Owner Upcoming/History, matching public profile events, matching detail |
| Event delete | Tombstone overlay | Owner Upcoming/History, matching public list/detail |
| Event report | No event content mutation | Report controller only; unavailable response clears stale detail through normal not-found path |
| Account switch/session invalidation | Advance business generation first | Repository, overlays, lists, details, mutations, reports, drafts, combined save, and routes return to account-safe state |

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Route Structure

Add typed routes without adding AppView routes:

```text
/profile/settings/products       BusinessProductsRoute
/profile/settings/events         BusinessEventsRoute
/events/:did/:rkey               BusinessEventRoute
```

- Products and Events settings routes are children of `SettingsRoute` and use `_NavigatorKeys.authenticatedShellNavigatorKey`, preserving compact/wide settings behavior.
- `BusinessEventRoute` is an authenticated-shell route parallel to `PostThreadRoute`, with constructor parsing/validation through existing DID/rkey mappers.
- Event create/edit remain temporary full-screen modal routes via `responsiveModalNavigator`, not named URLs.
- Owner route widgets re-check the current full-profile account type. A regular deep link renders no manager and returns to Settings; AppView remains the final authorization authority.

### 7.2 Profile Composition

Use one `List<ProfileTab>` for controller length, labels, children, keys, and selection mapping.

```text
regular:
  Projects, Posts, Comments, Reposts, About

business:
  Projects, Posts, Comments, Reposts, Products, Upcoming Events, About
```

- Stable enum/value keys preserve each surviving tab's `PageStorage` scroll state.
- Products and Upcoming Events never disappear during empty/nonempty or loading/error transitions for a normally visible business profile.
- If account type or block state removes the selected business tab, remap to About; do not index-clamp against separately computed lists.
- Blocked profiles retain the existing blocked-shell presentation and initialize no tab content. Do not initialize product/event providers for regular or blocked profiles.
- Keep Business as plain localized text with no verification icon or privilege styling.
- Put tagline and primary action in `ProfileMetaSection`; put types, offerings, location, service area, and hours in About.

### 7.3 Account And Settings

- Add a localized two-choice regular/business control to Account above the destructive section.
- Disable both choices while mutation is pending; retain the last confirmed selection on error.
- Main Settings watches the active full profile and inserts a Business heading with Events and Products only for authoritative business type.
- Switching to regular has no warning and sends no record deletion.

### 7.4 Combined Profile Editor

- Keep avatar/banner/display name/bio/crafts unchanged.
- For business accounts, append a `BusinessProfileFields` section for types, offerings, tagline, hours, service area, country/locality, and primary action.
- Products are not editable in this form, but the draft carries current products into replacement.
- Save enablement is `(ordinaryDirty || businessDirty) && allVisibleFieldsValid && noUpload && !saving`.
- On partial success, update only the successful baseline, leave failed fields/draft visibly unchanged, announce the failed portion, and keep the route open.
- Unsaved-work registration remains owned by the combined dialog and reflects both portions.

### 7.5 Products

- Render authored order using stable draft IDs that never enter wire JSON.
- Support drag reorder plus explicit Move up/Move down semantic actions.
- Enforce four-card maximum and required title, credential-free HTTPS URI, and image.
- Preserve untouched normalized source metadata; new upload overlay may use local preview bytes until AppView projection catches up.
- Product cards show image/title/optional locale-formatted authored price and an external affordance; no native detail route exists.

### 7.6 Events

- Events settings starts on Upcoming and retains selected view while opening/closing editors.
- Each view owns confirmed rows, loading-more status, cursor, incremental error, and refresh generation.
- Owner cards show lifecycle plus the two diagnostic categories without editing diagnostic codes.
- Event form supports create/update, local timed/all-day boundaries, IANA zone, roles, mode, status, optional content/destinations/image, and complete replacement.
- Delete uses localized destructive confirmation. Cancelled/Postponed are normal edits.
- Public Upcoming Events renders every server-returned row in server order without status filtering.
- Event detail shows only available hydrated fields/actions; owner gets edit/delete, non-owner gets Report.
- Any `event_not_found` replaces stale detail with unavailable and removes actions/data.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Account type pending/failure | Disable duplicate selection; failure restores prior confirmed value and localized error | FR-002 | AT-001, UT-011 |
| Regular owner-management deep link | Render no manager, schedule typed return to Settings, do not issue management request | FR-004 | AT-001, UT-018, IT-011 |
| Blocked profile | Existing blocked shell only; no business label/tab/provider fetch | FR-001, NFR-004 | AT-002, IT-012, REG-004 |
| Empty declaration | Business label/tabs/settings still derive from account type; editor starts empty and create uses `*` | FR-005, FR-007, FR-009 | AT-002, AT-004 |
| Combined save partial success | Commit successful baseline/cache; retain failed dirty portion and targeted retry | FR-010 | AT-004, UT-005, IT-005 |
| Declaration conflict | Keep draft; show changed-elsewhere action; reload complete current declaration before retry | FR-012 | AT-005, IT-006 |
| Product upload cancelled/failed | Preserve saved image and local draft; expose bounded retry | FR-020 | AT-005, IT-007 |
| Visitor upcoming initial load | Stable tab with centered progress state | FR-007, FR-017 | AT-009, UT-002 |
| Visitor upcoming initial failure | Stable tab with localized error and Retry; never show false empty | FR-017 | AT-009, UT-010 |
| Visitor/owner upcoming empty | Visitor explanatory empty; owner setup action to Events settings | FR-007, FR-017 | AT-009 |
| Incremental/refresh failure | Retain confirmed rows, show inline retry, retain cursor until explicit retry/refresh | FR-014, FR-017 | AT-006, AT-009, UT-010 |
| `invalid_cursor` | Discard only affected traversal, perform one cursorless restart, prevent retry loop | FR-023 | UT-010, IT-003 |
| Event crosses view boundary | Passive change appears on refresh; mutation restarts affected owner views; no shared-snapshot claim | FR-014, FR-023 | AT-006, IT-003 |
| Event conflict | Preserve draft/new server record boundary; offer Reload/Retry without blind overwrite | FR-016 | AT-008, IT-008 |
| Event delete projection lag | Tombstone hides exact deleted CID; settle on not-found; third CID appears as recreation | FR-016, FR-022 | AT-008, UT-015, IT-013 |
| Event unavailable/moderated/blocked | Replace detail with unavailable; omit report/external actions and hidden reasons | FR-019, NFR-004 | AT-010, IT-012 |
| Unknown catalog value | Safe readable fallback; preserve on unrelated declaration write; never offer as selectable first-party value | FR-001, FR-006, FR-009 | AT-002, UT-003, UT-004 |
| Unsafe/missing image | Omit image widget/object; keep containing eligible content stable | FR-021 | AT-013, IT-002 |
| External launcher false/throw | Keep current screen, show localized action failure, make no fetch/rewrite | RULE-004 | AT-003, AT-009, UT-012, MAN-002 |
| Account changes during async work | Operation lease/generation rejects all late state, message, navigation, CID, cursor, and draft publication | NFR-001 | AT-011, IT-010 |

## 9. Test Implementation Plan

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-002, AT-013 | AppView business profile/event response suites | CID-bearing declaration; JPEG/PNG/WebP/zero-size/aspect and unsafe images on every surface | CID absent and images are raw lexicon JSON instead of exact `PostImageView` |
| 2 | UT-007, IT-003, AT-006, REG-006 | AppView event filter/store/cursor/handler suites | Deterministic advancing clock; all statuses/reasons; tied starts; multiple pages; invalid combinations | Owner endpoint ignores filter and management cursor lacks cutoff/kind binding |
| 3 | IT-014, REG-011 | AppView profile merge/handler tests | Detail-only and product-only complete-known replacements with nested extension and large integer | Existing test does not prove both replacement modes |
| 4 | UT-001, IT-001 | Flutter business/profile model and API client tests | Exact normal/blocked/image JSON; Dio adapter for every route/header/query | Flutter drops business data and has no business client |
| 5 | UT-011, IT-004, AT-001 | Account type controller and Account/Settings widgets | Recording repository; current regular/business profiles; controlled failures | No selector/controller/conditional rows |
| 6 | UT-002, UT-003, AT-002, AT-003, REG-001, REG-005 | Profile tab policy, summary/About, product card widgets | Regular/business/blocked matrices, owner/visitor empty states, and unknown catalogs | Fixed five-tab enum and no business presentation |
| 7 | UT-004, UT-005, IT-005, AT-004, REG-002 | Declaration drafts, combined-save reducer/provider, edit dialog | Every changed-record/outcome ordering and unknown open values | Existing save provider fails fast as one ordinary mutation |
| 8 | UT-006, UT-014, IT-006, IT-007, AT-005 | Product draft/image/controller/manager tests | 0/4/5 products; reorder; upload outcomes; conflicts; unchanged image | No product manager or full declaration replacement |
| 9 | UT-010, IT-009, AT-009 | Public event list provider/tab/detail/router tests | Initial/next/error/empty/unusual-status pages; DID/rkey detail | No event models/providers/tab/detail route |
| 10 | UT-008, UT-009, IT-008, AT-007 | Event time/draft/API/editor tests | UTC, London/New York DST, Tokyo, timed/all-day, create/update/image | No IANA conversion or event editor |
| 11 | UT-016, AT-006, AT-008 | Owner list diagnostics, lifecycle mutation, manager tests | Closed reason catalog; independent page states; mutation movement/delete | No owner management state or diagnostics copy |
| 12 | UT-015, IT-013 | Projection overlay provider/integration tests | Create/update/delete identity sequences, third CID, failure, reload | Delayed AppView reads can overwrite accepted state |
| 13 | AT-010, IT-009, IT-012 | Event report and unavailable boundary tests | Owner/non-owner; report outcomes; 404/block/moderation | Report flow supports only post/profile |
| 14 | AT-011, IT-010, REG-008 | Account-boundary suites | Alice/Bob controlled reads/uploads/mutations/drafts/cursors | Business providers not invalidated/fenced |
| 15 | UT-012, UT-013, UT-017, AT-012, AT-014, REG-009, REG-010 | Localization/formatting/launcher/privacy/layout/semantics suites | Generated English copy, English formatter variants, exact viewports/scales, and all sink canaries | Hard-coded link dialog, missing generated business copy, unbounded quality checks |
| 16 | REG-003, REG-004, REG-007 plus full suites | Existing Settings/profile/AppView regressions | Existing fixtures unchanged | Broad integration may alter old rows, blocked shells, or event behavior |

Planned concrete Flutter test files follow the acceptance specification targets, with these consolidation choices:

- Keep wire decoding in `app/test/business/models/business_wire_models_test.dart` and existing `app/test/profile/models/profile_test.dart`.
- Keep declaration/product/event draft unit tests under `app/test/business/models/`.
- Use one `app/test/business/fakes/fake_business_repository.dart` with call recording and controlled completers.
- Keep provider tests under `app/test/business/providers/`; do not mock generated Riverpod classes.
- Extend existing profile/settings/router/report tests where preserving behavior is the purpose; create business-specific widget/page tests for new surfaces.
- Run AppView real-Postgres tests with `TEST_DATABASE_URL`; a skip is not completion evidence.

Focused commands by stage:

```text
cd appview && go test ./internal/api/... ./internal/business/...
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter test test/business test/profile test/settings test/router test/moderation
just app-analyze
just test
just app-test
```

## 10. Sequencing And Guardrails

- First TDD step: add the failing AppView IT-002 assertions for declaration CID and exact product/event image JSON across profile, owner list, public list, and detail.
- First focused command: `cd appview && go test ./internal/api/... -run 'Business(Profile|Event).*Image|BusinessProfile.*CID'` (adjust regex to the final test names without broadening scope).
- AppView image projection must be green before Flutter image models are frozen.
- AppView filter/cursor tests must be green before owner list providers are implemented.
- Flutter wire/client tests precede providers; provider tests precede pages/widgets; regression/quality tests close each vertical stage rather than being deferred entirely.
- Run code generation after each model/provider/router/localization batch and commit generated outputs with source during implementation.
- Use `gofmt` for Go and repository Dart formatting/analyze commands for Flutter.
- Do not alter lexicons, generated lexicon Go types, migrations, indexers, mutation routes, public eligibility, or unfiltered owner-event semantics.
- Do not add a dedicated business declaration GET route; the approved full-profile projection is authoritative.
- Do not split business details into a separate editor; preserve the approved one-Save partial-success flow.
- Do not parse, compare, sort, or infer chronology from CIDs.
- Do not place products in a native detail/commerce route.
- Do not add client status filtering to public upcoming events.
- Do not use `DateTime.timeZoneName` as an IANA identifier or perform manual DST offset arithmetic.
- Do not serialize hydrated display URLs, diagnostics, `createdAt`, or response identities into PDS mutation bodies.
- Do not log model `toString()` values for business records; operation/error reporting uses bounded enums and request IDs only.
- Do not fetch, preview, resolve, or validate destinations over the network.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | `PostImageView.Size` currently uses `omitempty`, which violates required `size: 0` business JSON. | A shared tag change could affect post response snapshots. | Make size required on the shared type and update exact post regressions; business builder still rejects incomplete source metadata. |
| CPQ-003 | Non-blocking | IANA time conversion and localized country names are unavailable in current dependencies. | Event DST/all-day and location display cannot satisfy requirements with SDK APIs alone. | Add pure-Dart `timezone ^0.11.1` and `l10n_countries ^2.0.3`; initialize/inject them and pin through `pubspec.lock`. |
| CPQ-004 | Non-blocking | New uploaded images have no AppView display URL until projection. | Accepted UI could briefly lose its image. | Accepted overlay stores local preview bytes separately from wire draft; settle to AppView URLs only under FR-022 identity rules. |
| CPQ-005 | Non-blocking | Existing external-link confirmation is hard-coded and does not handle false/throw. | New surfaces would violate localization/error requirements. | Strengthen the shared helper under regression tests before business actions use it. |
| CPQ-006 | Non-blocking | Dynamic tab removal can invalidate controller index and retained scroll state. | Runtime assertion or wrong selected tab. | Derive one tab list, key by enum, and explicitly remap selected logical tab on profile changes. |
| CPQ-007 | Non-blocking | Independent event cutoffs permit temporary cross-view overlap/omission. | Owner may see movement around end time. | Preserve the approved per-traversal contract; refresh/mutations restart only affected views and UI makes no snapshot claim. |
| CPQ-008 | Non-blocking | Combined save has two independently completed writes and projection paths. | Duplicate retry or false rollback. | Use explicit per-record outcomes, mutable per-section baselines, accepted overlay, and controlled-completion tests in both orders. |
| CPQ-009 | Non-blocking | Product/business edits can race on one declaration. | One surface could erase another. | Complete-known replacement plus current CID; conflict requires reload, never blind retry. |
| CPQ-010 | Non-blocking | Exact API response types currently serialize business domain structs directly. | Adding normalized images can cause an `api`/`business` import cycle. | Keep validated metadata in `business`; add API response builders/types in `api`; never import `api` from `business`. |

Blocking questions: None.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-08-29-flutter-business-profiles/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-08-29-flutter-business-profiles/05-implementation-plan.md`
- Start with test: IT-002, declaration CID and exact business image response contract.
- First target: `appview/internal/api/business_profile_acceptance_test.go` and `appview/internal/api/business_event_acceptance_test.go`.
- Focused command: `cd appview && go test ./internal/api/... -run 'Business(Profile|Event).*Image|BusinessProfile.*CID'`.
- Next prerequisite: IT-003 owner event filter/cursor contract before Flutter owner event state.
- TDD order: follow Section 9 exactly unless a red test proves an inspected extension point differs; record any necessary plan deviation in `05-implementation-plan.md` before implementing it.
- Generated artifacts expected during implementation: Flutter mappers, Riverpod providers, typed router, localization output, and `pubspec.lock`; no lexicon generation is expected.
- Verification gate: focused tests per stage, then `just test`, `just app-test`, `just app-analyze`, `git diff --check`, MAN-001, and MAN-002.
