# Coding Plan: Public Profile Customisation

## 1. Status And Inputs

Status: Ready for TDD implementation after approval\
Date: 2026-08-09\
Risk: Medium\
Workflow inputs:

- `01-requirements.md`
- `02-acceptance-tests.md`
- `03-document-review.md` (`Approved with notes`)
- `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- `atproto-craft-social-app-reference.md`

This plan implements the approved AppView-owned public customisation resource, additive identity response enrichment, authenticated full-replacement mutation, shared Flutter rendering, and Settings editing flow. It does not change a Lexicon, PDS record, Tap indexer, blob, atproto profile mutation, or public interaction record.

Three design inputs remain deliberately gated rather than being invented during implementation:

1. **Palette gate (GAP-001 / DR-005):** approve and record stable keys plus audited base, foreground, hover, pressed, and soft-container values for the five non-cobalt colour bundles before the first exact-value, contrast, or palette-sensitive red test.
2. **Texture-style gate (GAP-002 / DR-005):** approve and record the tint and opacity for every colour bundle before the first exact texture painter/style test or visual baseline.
3. **Failure-copy gate (GAP-003 / DR-005):** approve and record the themed save-failure message before adding its exact string assertion. State, retention, and retry behavior can be implemented earlier.

These gates do not block persistence, API contracts, response hydration, tolerant decoding, draft state, avatar geometry, local asset plumbing, or structural theme-boundary work.

## 2. Implementation Decisions

### 2.1 AppView persistence

Add one dedicated AppView table with explicit value columns. The current migration head is `000035_profile_pins`, so the expected files are `000036_profile_customisation.up.sql` and `.down.sql`; implementation must re-check the head before creating them.

```text
profile_customisations
- owner_did          TEXT PRIMARY KEY
    -> craftsky_profiles(did) ON DELETE CASCADE
- colour             TEXT NOT NULL
- profile_border     TEXT NOT NULL
- profile_background TEXT NOT NULL
- created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
- updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
```

Use one row and one `INSERT ... ON CONFLICT (owner_did) DO UPDATE` statement so every successful write replaces a complete combination atomically. Do not add catalogue `CHECK` constraints: the server request validator owns the current write allow-list, while read-time per-field fallback must keep a retired stored key renderable after catalogue evolution. The primary key is also the index used by single-owner and `owner_did = ANY($1)` batch reads. The membership foreign key gives permanent deletion/rejoin semantics without a separate cleanup worker.

No row represents the complete defaults. Reads must not eagerly create default rows. Timestamps are internal and never enter public responses or telemetry labels.

### 2.2 Canonical value and catalogues

Define a small server wire/domain value with exactly these current fields:

```text
ProfileCustomisation
- colour
- profileBorder
- profileBackground
```

The server catalogue and Flutter catalogue independently assert the same stable wire keys. Defaults are `cobalt`, `medium`, and `none`; cobalt resolves to `#1535D6`. Border keys are exactly `thin`, `medium`, and `thick`. Background keys are exactly `none`, `bayerdark`, `cubedark`, `dotcrossdark`, `scallopdark`, `skewdark`, and `x2`.

Keep mutation parsing stricter than public response parsing:

- `PUT` accepts exactly the three current camelCase keys, all strings, with no missing or unknown keys.
- Unsupported catalogue values produce field-specific `422 validation_failed` details.
- Malformed JSON, duplicate keys according to the repository's strict-decoder policy, extra resource data, and oversized bodies are rejected before persistence.
- Public responses remain additive: future nested response fields may be ignored by current Flutter, while the current mutation still rejects them.
- Stored and received unknown values fall back independently by field, preserving valid siblings.

### 2.3 Centralized response hydration

Do not add customisation joins independently to every existing hot query and do not perform a database read from a response builder. Introduce one `IdentityCustomisationHydrator` backed by a batch reader:

```text
ProfileCustomisationReader
- ReadEffective(ctx, ownerDID) -> effective value
- ReadEffectiveMany(ctx, ownerDIDs) -> map[DID]effective value

IdentityCustomisationHydrator
- Profile(response)
- Profiles(responses)
- Posts(responses), recursively including replies/comments and quote previews
- Notifications(items), including available actors and subject posts
- SearchProfiles(items)
- AccountSummaries(items)
```

Each response DTO/builder starts with a complete default object. At the handler boundary, after all post/quote/actor/account objects for a response page have been assembled, the hydrator collects and deduplicates every retained DID, performs at most one indexed batch customisation query, resolves missing/retired fields, and decorates every occurrence before JSON encoding. This keeps builders pure, makes no-row unit fixtures valid by default, and bounds hydration independently of page length or repeated authors.

The handler must fail the response with the existing internal-error envelope if authoritative batch hydration fails; it must not silently emit defaults for a database outage. Defaults are for a successful missing-row result, not for an unreadable store. Retired stored values use defaults for only those fields and emit a bounded diagnostic.

Moderation policy runs before or alongside DID collection:

- An ordinary retained identity gets the complete object.
- A blocked/muted shell gets customisation only when its existing JSON shell retains the avatar.
- A stripped post author, blocked quote, hidden identity, or actor-free system notification stays stripped/actor-free.
- Hydration never restores display name, avatar, post content, or relationship fields removed by policy.

### 2.4 Flutter value and confirmed-state boundary

Add one immutable `ProfileCustomisation` value and local catalogue policy. Its decoder must tolerate a missing object and unknown values per field; do not use a strict generated enum decoder that can throw away the whole identity. Generated parent mappers may call a custom mapping hook/converter for the nested object. The value owns equality, defaults, `copyWith`, wire serialization for the mutation, and field-specific `fromWire` fallbacks.

Add the value to every Flutter identity model that corresponds to an enriched AppView shape:

- `Profile`
- `PostAuthor` (therefore posts, replies/comments, and quote previews)
- `NotificationActor`
- `ProfileSearchResult` and any profile suggestion/recent-search payload that exposes an avatar
- `ProfileAccountSummary` (therefore followers, following, muted, blocked, and other account lists)

The editor's draft is local to the Settings page/provider. It may drive only that page's preview. Public profile/avatar/theme state changes only after the authoritative `PUT` response succeeds. A failed request leaves the old confirmed value globally visible and the draft available for retry.

### 2.5 Shared avatar seam

Extend `ProfileAvatar` with an effective customisation input defaulting to the canonical defaults. Keep its existing circular clipping, initial/loading/error behavior, hard-offset ink shadow, semantics, and external `36`, `48`, or `96` dimensions. Replace only the old ink border with one inward-painted selected-colour stroke:

| Avatar size | Thin | Medium | Thick |
|---|---:|---:|---:|
| 36 px | 1.5 px | 2.5 px | 4 px |
| 48 px | 2 px | 3.5 px | 5 px |
| 96 px | 3 px | 5 px | 8 px |

Remove the decorative `ProfileAvatarFrame` rendering path. Where the full/compact header currently reserves a larger positioning box, center the ordinary 96 px `ProfileAvatar` in that box without drawing a rim or second ring.

`AccountAvatar` is a legacy parallel seam and uses a non-catalogue 32 px size. Replace its navigation/account-switcher uses with the standard 36 px `ProfileAvatar`, or make it a thin adapter around that exact size; do not invent a fourth width table. Verify surrounding navigation and switcher layout in regression tests.

### 2.6 Profile theme and texture boundaries

Replace runtime `ColorScheme.fromSeed` and route-supplied colours with a fixed `ProfileColourThemeBundle` resolved from `profile.customisation.colour`. Each approved bundle contains base, readable foreground, hover, pressed, soft container, and the approved texture tint/opacity. `ProfileCustomisationTheme` consumes that bundle and supplies Material state styles for text links, icon buttons, primary/secondary buttons, and containers.

Scope it as follows:

- Compact presentation: wrap the complete `ProfileCard` surface so all header, metadata, buttons, and links use the selected bundle.
- Full presentation: theme only the header slivers above the tab bar. `ProfileSliverAppBar`, `ProfileMetaSection`, relationship annotations, and actions opt into the profile theme; `ProfileTabBar` and its entire `TabBarView` remain under the normal Craftsky theme.
- Loading/error shells and app-wide navigation remain under the normal app theme unless they are part of an already-loaded compact profile.

Rewrite `ProfileHeaderBackground` as a bounded, clipped local texture layer over the selected base colour. The six texture options map to bundled transparent assets and tile without `NetworkImage`, HTTP, CSS, or arbitrary paths. `none` paints only the base colour. Both compact and full headers derive the choice from the loaded `Profile`; route extras no longer carry colour, background, or avatar-frame values.

## 3. Files And Modules

### 3.1 AppView

| Path / module | Change | Purpose | Requirement / AC IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000036_profile_customisation.up.sql` and `.down.sql` | Create after re-checking head | Add/drop the owner-DID row, membership cascade, complete value columns, and timestamps without changing existing data. | BR-001, FR-004, NFR-002 | IT-001, REG-001 |
| `appview/internal/db/profile_customisation_migration_test.go` and `migration_files_test.go` | Create/extend | Prove up/down/up, PK/FK/index/timestamps, member deletion/rejoin, and preservation of prior data. | FR-004, NFR-002, NFR-006 | IT-001, REG-001 |
| `appview/internal/api/profile_customisation.go` | Create | Define wire value, defaults, catalogues, per-field effective fallback, exact request decoding, handler interfaces, and mutation handler/error mapping. | FR-002, FR-003, FR-011, FR-012, RULE-001–RULE-004 | UT-001, UT-002, IT-002, IT-003 |
| `appview/internal/api/profile_customisation_store.go` | Create | Implement missing-row defaults, batch read, retired-key fallback, and atomic full upsert using typed `syntax.DID` boundaries. | FR-003, FR-004, FR-012, NFR-001 | IT-002, IT-005 |
| `appview/internal/api/profile_customisation_hydrator.go` | Create | Deduplicate retained identities and decorate full/nested response graphs with one batch read. | FR-001, FR-013, NFR-001, NFR-004 | AT-001, IT-004–IT-006 |
| `appview/internal/api/profile_response.go` and tests | Change | Add complete nested object to `ProfileResponse` and `ProfileAccountSummary`; preserve blocked-shell JSON policy. | FR-001, FR-013 | AT-001, IT-004, IT-006, REG-002, REG-006 |
| `appview/internal/api/post_response.go` and post/timeline/saved-post response tests | Change | Add the object to `PostAuthor`; hydrate roots, replies/comments, quote previews, saved posts, and timelines once per response. | FR-001, NFR-001 | AT-001, IT-004, IT-005, REG-002 |
| `appview/internal/api/notifications.go` and tests | Change | Add actor customisation only for available/retained actors and hydrate subject posts in the same batch. | FR-001, FR-013 | AT-001, IT-004, IT-006, REG-006 |
| `appview/internal/api/search_response.go`, `search.go`, and tests | Change | Enrich profile results/suggestions and post results; include recent profile payloads only where the current payload retains identity/avatar data. | FR-001, NFR-001 | AT-001, IT-004, IT-005, REG-002 |
| Relationship/profile-account handlers and response tests | Change | Hydrate followers, following, muted, blocked, and other `ProfileAccountSummary` collections once per page. | FR-001, FR-013, NFR-001 | AT-001, IT-004–IT-006 |
| `appview/internal/api/profile_customisation_request_test.go` | Create | Lock strict keys/types/catalogues/body size and field-specific validation without writes. | FR-003, RULE-001–RULE-003 | UT-002, AT-003 |
| `appview/internal/api/profile_customisation_store_test.go` and `profile_customisation_test.go` | Create | Real-Postgres defaults, replacement, retries, devices, isolation, concurrency, lifecycle, auth, and exact response tests. | FR-002–FR-004, RULE-004 | IT-002, IT-003, AT-003 |
| `appview/internal/api/profile_customisation_query_plan_test.go` | Create | Assert PK/index plan and fixed customisation statement count as page size/repeated appearances grow. | NFR-001 | IT-005 |
| `appview/internal/api/profile_customisation_boundary_test.go` | Create | Assert zero PDS, profile/blob writer, Tap/indexer, and indexing-wait calls. | BR-005, RULE-005 | IT-007, REG-009 |
| `appview/internal/api/profile_customisation_observability_test.go` and observer recorder files | Create/extend | Record bounded operation/result/error class and duration; reject DID, choice keys, and filenames as metric labels. | NFR-005 | IT-009 |
| `appview/internal/app/deps.go` | Change | Construct/share the customisation store and hydrator from the existing Postgres/observer dependencies. | FR-004, NFR-001 | IT-002, IT-005 |
| `appview/internal/routes/routes.go`, `policy.go`, and `routes_test.go` | Change | Register `PUT /v1/profiles/me/customisation` with authenticated, device-bound, current-member, JSON body/rate policy and inject hydration into affected handlers. | FR-002, RULE-004 | IT-003, IT-008, REG-002 |

No file under `lexicon/`, generated Lexicon code, Tap registration, or PDS profile writing is changed.

### 3.2 Flutter models, API, and account state

| Path / module | Change | Purpose | Requirement / AC IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/profile/models/profile_customisation.dart` | Create | Immutable value, defaults, tolerant field-by-field decode, strict mutation encode, catalogue keys, theme/background lookup types, equality/copy. | FR-003, FR-011, FR-012, RULE-001–RULE-003 | UT-001, UT-003 |
| `app/lib/profile/models/profile.dart`, `profile_account_summary.dart`, generated mappers | Change/regenerate | Decode additive customisation with default/unknown tolerance. | FR-001, FR-012, NFR-004 | UT-003, IT-008 |
| `app/lib/feed/models/post.dart`, notification/search/recent-search models, generated mappers | Change/regenerate | Carry the same value through every embedded avatar identity. | FR-001, FR-008, NFR-004 | AT-001, AT-002, UT-003, IT-008 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Add authenticated `PUT /v1/profiles/me/customisation`, sending exactly three string fields and decoding the authoritative object. | FR-002, FR-006 | UT-009, IT-008 |
| `app/lib/profile/data/profile_repository.dart`, `api_profile_repository.dart`, fakes | Change | Expose the full-replacement mutation through the existing profile repository boundary with controllable deferred test responses. | FR-006 | UT-008–UT-010 |
| `app/lib/profile/providers/profile_customisation_provider.dart` plus generated provider | Create | Own account-lease-scoped confirmed/draft state inside `AsyncValue`, live preview, save suppression, authoritative reconciliation, retained-value error handling, and stale-completion fencing. | FR-005, FR-006, FR-010 | AT-006, AT-007, UT-008–UT-010 |
| `app/lib/profile/providers/user_profile_provider.dart` | Change | On success, replace/invalidate alive handle/DID self-profile caches without applying a draft optimistically. | FR-006, FR-010 | UT-009, REG-008 |
| `app/lib/auth/models/stored_session.dart`, `session_registry.dart`, providers, tests | Change | Persist a tolerant `cachedCustomisation` alongside cached avatar/display name and update only the initiating account after success/profile load. | FR-006, FR-010, FR-012 | AT-007, UT-010, REG-008 |
| App shell/account switcher view models and widgets | Change | Pass the correct account's cached/effective customisation to navigation presentation immediately after startup/switch. | FR-008, FR-010 | AT-002, REG-004, REG-008 |

### 3.3 Flutter rendering, Settings, assets, and routes

| Path / module | Change | Purpose | Requirement / AC IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/profile/widgets/profile_avatar.dart` | Change | Centralize exact inward border-width policy and preserve dimensions, clipping, fallback, semantics, and shadow. | FR-007, FR-008, RULE-002 | AT-002, AT-004, UT-004, UT-005 |
| `post_card.dart`, `notification_row.dart`, `post_summary.dart`, search/profile/account-list/avatar callers | Change | Pass the embedded identity customisation into the shared avatar renderer. | FR-008 | AT-002, REG-004 |
| `app/lib/auth/widgets/account_avatar.dart` and navigation/account-switcher callers | Change/remove adapter | Use standard 36 px `ProfileAvatar` with cached identity customisation; avoid a separate 32 px policy. | FR-008 | AT-002, REG-004, REG-008 |
| `profile_framed_avatar.dart` and callers | Remove/replace | Eliminate decorative frames and center the standard 96 px avatar in existing header positioning. | FR-007, RULE-002 | AT-004, REG-004 |
| `profile_customisation_theme.dart` | Rewrite | Resolve fixed approved bundles; style buttons, links, interactions, and containers without runtime hue generation. | FR-010, FR-011, RULE-001 | AT-005, UT-006 |
| `profile_header_background.dart` | Rewrite | Paint only local tiled texture masks, selected base, approved tint/opacity, and exact header clipping. | FR-009, FR-011, RULE-003 | AT-005, UT-007, IT-010 |
| `profile_card.dart`, `profile_sliver_app_bar.dart`, `profile_meta_section.dart`, `profile_page.dart` | Change | Derive from loaded profile; theme all compact content and only full header content before the tab bar. | FR-009, FR-010 | AT-005, UT-006, UT-007, REG-005 |
| `profile_presentation_page.dart`, `profile_route_presentation.dart`, `profile_card_modal.dart`, router call sites | Change | Remove route-extra colour/background/frame injection while preserving compact-to-full transitions. | FR-009 | AT-005, REG-005 |
| `app/lib/settings/pages/profile_customisation_page.dart` and focused widgets | Create | Render loading/error, fixed accessible controls, representative live draft preview, explicit Save, branded discard dialog, and retryable failure. | FR-005, FR-006, NFR-003 | AT-006, AT-010, UT-008, UT-011, MAN-001 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add localized Customisation entry. | FR-005 | AT-006, REG-003 |
| `app/lib/router/route_locations.dart`, `router.dart`, `router.g.dart` | Change/regenerate | Add typed `/profile/settings/customisation` child route on the authenticated shell navigator. | FR-005 | AT-006, REG-003, REG-007 |
| `app/assets/profile_backgrounds/*`, provenance README, `app/pubspec.yaml` | Create/change | Check in the six approved Ribo texture files, record source/license/attribution, and expose only a bundled asset directory. | FR-011, RULE-003 | UT-007, IT-010, MAN-002 |
| `app/lib/l10n/app_en.arb` and generated localization files | Change/regenerate | Add control labels, semantics, discard copy, exact success copy, and the later approved failure copy. | FR-005, FR-006, NFR-003 | AT-006, AT-010, UT-009, UT-011 |

## 4. API, Services, And Error Semantics

### 4.1 Mutation flow

```text
Settings draft
  -> ProfileRepository.updateMyCustomisation(complete draft)
  -> PUT /v1/profiles/me/customisation
  -> auth + device + current-member middleware
  -> strict body/catalogue validation
  -> atomic AppView upsert by authenticated DID
  -> 200 complete authoritative customisation
  -> initiating Flutter lease reconciles confirmed caches and page state
```

The route has no DID path/body target, so another member cannot be selected. Use the standard error envelope and request ID. Preserve the repository's existing authentication/current-membership statuses. Proposed feature mappings are:

| Case | Result | State effect |
|---|---|---|
| Complete supported request | `200` authoritative object | Complete atomic create/replace |
| Identical retry | `200` same effective object | No duplicate row; timestamp behavior may remain an internal implementation detail |
| Malformed/partial/extra/non-string/oversized body | Existing `400` request/validation envelope | None |
| Unsupported catalogue value | `422 validation_failed` with field details | None |
| Missing/expired auth, device mismatch, removed member | Existing middleware error | None |
| Store failure | `500 internal_error` | Transaction rolled back |

Concurrency tests coordinate transactions with locks/barriers, not sleeps. Each request writes one complete row; the last committed write wins and no mixed-field state is observable.

### 4.2 Public read flow

```text
existing profile/post/notification/search/relationship query
  -> existing response builders and moderation policy
  -> collect retained identity DIDs recursively
  -> one ProfileCustomisationStore.ReadEffectiveMany call
  -> apply saved values/defaults/per-field retired-key fallback
  -> encode existing response plus nested customisation
```

Full/self profile reads use the same effective-value policy. Collection hydration remains one bounded customisation statement per response even as page size increases. The plan intentionally does not add a public standalone customisation GET: the effective value arrives with profile/identity data, and the editor seeds itself from the authenticated self-profile response.

### 4.3 Observability

Follow the current observer/structured-log pattern. Record only bounded dimensions such as operation (`read_one`, `read_many`, `replace`), result (`success`, `default`, `validation_error`, `store_error`), error class, and duration. DIDs, handles, catalogue selections, raw request bodies, asset names, and URLs are not metric labels. A retired-key structured log may name the field category and bounded reason, but not the owner or retired raw value. No new alert is required.

## 5. Riverpod Provider Graph And State Transitions

```text
sessionRegistryProvider
  -> activeLease (account + generation)
  -> account-scoped Dio/ProfileRepository
  -> profileCustomisationProvider(activeLease)
       -> GET self Profile seeds confirmed + draft
       -> setters change draft only
       -> PUT complete draft on Save
       -> authoritative response changes confirmed + draft
       -> targeted userProfile/session identity cache update

userProfileProvider(handle/DID)
  -> Profile.customisation
  -> full/compact header theme + texture
  -> ProfileAvatar

embedded PostAuthor / NotificationActor / ProfileAccountSummary
  -> ProfileAvatar directly; no profile lookup
```

Suggested editor state:

```text
ProfileCustomisationEditorState
- confirmed: ProfileCustomisation
- draft: ProfileCustomisation
- isDirty = draft != confirmed
```

`AsyncValue<ProfileCustomisationEditorState>` owns the asynchronous transport state instead of duplicating it inside the value:

- initial load: `isLoading && !hasValue`
- loaded/idle: `hasValue && !isLoading && !hasError`
- save pending with the draft retained: `isLoading && hasValue`
- initial-load failure: `hasError && !hasValue`
- save failure with confirmed/draft retained: `hasError && hasValue`

Riverpod preserves the previous value when an `AsyncNotifier` transitions from data to loading/error. Use the public `isLoading`, `hasValue`, `hasError`, `value`, and `error` surface; do not call Riverpod's internal `copyWithPrevious` API. The page can listen for loading-with-value to data/error transitions to emit one success/failure message, so the domain state does not need an outcome revision or error field.

Provider behavior:

1. Key the provider by `ActiveAccountLease`, not a mutable global current account.
2. Load self profile through the initiating account's repository; a stale load completion is discarded.
3. Field setters synchronously update `draft`, producing a live preview and value-based dirty state.
4. Save is disabled when `AsyncValue.isLoading`, clean, or not yet loaded. The notifier also guards duplicate programmatic calls.
5. Transition the notifier to `AsyncLoading` before the request. Riverpod retains the previous editor value, so the page keeps rendering the confirmed/draft controls and derives save-pending as `isLoading && hasValue` rather than replacing the whole state with a blank loading screen.
6. On success, verify the lease is still current for active-page feedback. Persist/update caches addressed to the initiating account even if another retained account is now active, but only while that initiating session/generation still exists; logout/removal must not recreate it. Never recolour the newly active account.
7. On success, publish `AsyncData` whose `confirmed` and `draft` both equal the authoritative response, remain on the page, and show `Profile customisation saved` only for the active initiating page.
8. On failure, publish `AsyncError`; Riverpod retains the prior editor value, so `confirmed` and `draft` remain available and Save can be retried. Expose the approved themed failure feedback only if the lease/page is still current.
9. Register the existing unsaved-work/account-switch guard while dirty or saving. Back shows the branded discard dialog only when `draft != confirmed`; reverting or successful saving makes Back direct.

## 6. UI Composition And Accessibility

The Settings page uses three labelled finite-choice groups plus a representative preview. Controls expose semantic group labels, option labels, selected state, and actions; colour swatches also show text labels, borders show names/thickness previews, and textures show names so no choice relies on hue or pattern recognition alone. Keyboard traversal follows preview, colour group, border group, background group, then Save in a stable order. After dismissing discard confirmation, focus returns to the invoking control/page. Pending state disables only duplicate mutation initiation and preserves coherent navigation/focus.

Test supported text scales, light/dark app themes, narrow phone and wider layouts, missing/error avatar images, and 36 px thick borders. Contrast assertions use the approved fixed bundles after the palette gate. Manual VoiceOver/TalkBack, colour-vision, and texture-balance checks supplement rather than replace widget/contrast/bounds tests.

## 7. TDD Implementation Sequence

Every slice starts with the named smallest failing test, adds only enough production code to pass, refactors behind the green contract, and then runs the affected regression group.

### Slice 1: Effective value, defaults, and catalogue structure

- Start with `UT-001` in Go and Dart.
- Create the canonical server value/default policy and tolerant Flutter value.
- Lock cobalt, border keys/width vocabulary, background keys/display mapping, uniqueness, and independent fallback.
- For the five non-cobalt entries, test only six-slot structure until the palette gate is approved; do not invent final names or values.
- Extend parent model fixtures for old responses and per-field unknown values (`UT-003`, `IT-008`).

### Slice 2: Strict mutation request policy

- Start with `UT-002`.
- Implement exact JSON key/type/body-size parsing and server allow-list validation.
- Prove every invalid request is a no-op before connecting a store.

### Slice 3: Durable AppView persistence

- Start with `IT-001`, create the next verified migration, and prove up/down/up plus cascade/preservation.
- Add `IT-002` store tests for no-row defaults, complete upsert, identical retry, account/device isolation, retired field fallback, fresh-store durability, and deletion/rejoin.
- Add transaction-barrier concurrency coverage for complete last-commit-wins behavior.

### Slice 4: Authenticated route

- Start with `IT-003` route/handler tests.
- Register route policy and handler, wire the store, preserve standard envelopes, and return the authoritative object.
- Add boundary spies for `IT-007`/REG-009 before considering the slice complete.

### Slice 5: Public response enrichment

- Start with `IT-004` on full profile and `PostAuthor`, then add each response family to the hydrator inventory.
- Cover profile, post root/reply/comment/quote, timeline, saved post, notification actor/subject, search result/suggestion/recent identity, and relationship/account summaries.
- Add `IT-006` moderation fixtures as each custom marshal/availability shell changes.
- Finish with `IT-005`: repeated authors, growing page sizes, bounded statement count, and PK/`ANY` plan assertions.

### Slice 6: Flutter repository and editor lifecycle

- Add API-client/repository fixtures for exact request/authoritative response (`IT-008`).
- Start `UT-008` for confirmed/draft equality, load, revert, pending, success, failure, and retry.
- Start `UT-009`/`UT-010` for duplicate-save suppression, targeted cache reconciliation, lease fencing, logout, switch, and disposal.
- Build the Settings route/page and AT-006 navigation/save/discard scenarios.
- Keep failure behavior assertions semantic until the failure-copy gate closes, then add the exact localized string assertion.

### Slice 7: Shared avatar propagation

- Start `UT-004` with all nine size/thickness combinations.
- Extend `profile_avatar_test.dart` for all image/fallback states, clipping, one inward stroke, selected colour, dimensions, shadow, and semantics (`UT-005`).
- Remove decorative frame/second-ring paths.
- Drive the complete TD-006 surface matrix through `ProfileAvatar`, including navigation/account switching, and run REG-004.

### Slice 8: Local textures and scoped profile theming

- Acquire/check in the six exact source assets and provenance/license notes before asset-dependent implementation. If the source site cannot be automated, pause only this asset step for a reviewed local acquisition; do not substitute unrelated textures.
- Add structural local-asset mapping/no-network/tiling/clipping tests (`UT-007`, `IT-010`) that do not depend on final tint.
- Resolve route extras out of compact/full presentation and establish the structural theme boundary.
- Close the palette gate, record the approved bundle constants in `01-requirements.md` or a linked approved design artifact, then write exact catalogue/contrast/material-state tests (`UT-001`, `UT-006`).
- Close the texture-style gate, record per-colour tint/opacity, then write the exact texture style/baseline assertions (`UT-007`, AT-005).
- Run REG-005 to prove compact scope is complete and full scope ends before the tab bar.

### Slice 9: Accessibility, observability, and full regression

- Complete `UT-011`, AT-010, IT-009, and the REG-001–REG-009 matrix.
- Run MAN-001–MAN-003 after automated semantics, contrast, bounds, and local-resource tests pass.
- Perform a scoped diff check proving no Lexicon/PDS/Tap/blob implementation changed.

## 8. Generated Outputs And Verification Commands

Generated files are committed. After relevant source edits:

```text
cd app
flutter gen-l10n
dart run build_runner build
dart format lib test
dart analyze
flutter test <focused target>
```

Inspect generated mapper/provider/router/localization diffs and avoid unrelated churn. If the current generator reports conflicts, use the repository's established conflict-resolution flag only after confirming the conflicted files are generated outputs for this feature.

AppView focused loops run from `appview/` with `go test` for the affected package. Full project verification runs from the repository root with the Compose Postgres available:

```text
just test
just app-test
just app-analyze
git diff --check
```

Report only commands that actually complete. Query-plan and migration tests require the repository's real-Postgres test setup; do not replace them with mocks. No dependency addition is planned.

## 9. Guardrails, Risks, And Closure Checks

- **Response completeness:** before finishing Slice 5, inventory every JSON type with `avatar`, `avatarCid`, `ProfileAccountSummary`, `PostAuthor`, or `NotificationActor` and either hydrate it or document why moderation/system policy removes it.
- **Query growth:** never call `ReadEffective` in a per-item loop or widget/provider. One response page gets at most one customisation batch query.
- **Fallback correctness:** absence is not an error; database failure is. Unknown fields fall back independently on both server and client.
- **Theme leakage:** do not wrap the whole full `ProfilePage` in the profile theme. Test the tab bar and post/tab body under the normal Craftsky theme.
- **Draft leakage:** a preview draft never enters `userProfileProvider`, embedded identity state, or session cache before server confirmation.
- **Account fencing:** stale continuations must not emit feedback or change the active page/theme. Cache updates must be addressed to the initiating `AccountKey`.
- **Avatar geometry:** use only 36/48/96 sizes and the approved table. Preserve the hard ink shadow and outside layout; remove decorative frame code rather than layering it with the new stroke.
- **Asset provenance:** retain original source URL, local filename, retrieval date, license/usage text, and required attribution. No hotlink or runtime remote fallback.
- **Catalogue retirement:** do not use database catalogue checks that turn a future retired key into a migration outage. Server writes remain closed; reads remain tolerant.
- **API compatibility:** keep existing status/error semantics and blocked custom marshals unchanged except for the additive nested field where an avatar remains.
- **Scope:** no Lexicon, PDS OAuth client, Tap dispatcher, profile blob, algorithm/ranking, notification creation, or global theme work.

## 10. Requirement And Test Closure Matrix

| Area | Requirements | Acceptance criteria | Primary tests/slices |
|---|---|---|---|
| AppView ownership, mutation, lifecycle | BR-001, BR-004, BR-005, FR-002–FR-004, RULE-004, RULE-005 | AC-003–AC-005, AC-014, AC-016, AC-017 | AT-003, UT-002, IT-001–IT-003, IT-007–IT-009; Slices 2–4 |
| Public identity enrichment and moderation | FR-001, FR-012, FR-013, NFR-001, NFR-004 | AC-001, AC-002, AC-013, AC-015, AC-017, AC-019 | AT-001, AT-008, AT-009, IT-004–IT-006, IT-008; Slices 1 and 5 |
| Shared avatar and all surfaces | BR-002, FR-007, FR-008, RULE-002 | AC-002, AC-008–AC-010 | AT-002, AT-004, UT-004, UT-005, REG-004; Slice 7 |
| Compact/full colour and backgrounds | BR-003, FR-009–FR-011, RULE-001, RULE-003 | AC-005, AC-011, AC-012, AC-019 | AT-005, UT-001, UT-006, UT-007, IT-010, REG-005; Slice 8 after gates |
| Settings, save, account fencing | FR-005, FR-006, FR-010 | AC-006, AC-007, AC-014 | AT-006, AT-007, UT-008–UT-010, REG-003, REG-007, REG-008; Slice 6 |
| Accessibility and quality | NFR-002, NFR-003, NFR-005, NFR-006 | AC-018, AC-020 | AT-010, UT-011, IT-001, IT-005, IT-009, REG-001–REG-009, MAN-001–MAN-003; Slice 9 |

All 28 Must requirements and AC-001 through AC-020 have an implementation owner and planned verification. NFR-005 remains Should priority but is included in Slice 9.

## 11. TDD Handoff

The first implementation action is `UT-001`: add failing Go and Dart tests for the effective value/default/catalogue structure, then implement only that policy. Continue in the slice order above. Do not begin exact non-cobalt theme values, exact texture styling, or exact failure-copy assertions until their named approval gates have been closed and recorded.

After all slices and manual supplements, run `$review-implementation` against `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this plan. The review must explicitly check response-surface inventory, bounded hydration evidence, no-PDS/no-network boundaries, account fencing, exact avatar geometry, theme clipping, asset provenance, accessibility, and the commands that actually ran.
