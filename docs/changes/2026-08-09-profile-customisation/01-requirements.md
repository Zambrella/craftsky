# Requirements: Public Profile Customisation

## 1. Initial Request

Complete Craftsky's currently presentation-only profile customisation so each member can choose a profile colour, one of three profile-picture border thicknesses, and a bundled profile-background texture. The selected border and its colour must appear everywhere that member's profile picture appears, while the background must appear in both compact and full profile views. Customisation is public profile data returned by the AppView with profile/actor information, but its authoritative state is stored only in the AppView against the member's DID and is never written to the PDS. Members manage the fixed, locally rendered choices from a page under Settings. The contract and client architecture must make future customisation fields additive.

The existing Flutter customisation widgets are discovery evidence only. They are not a compatibility contract and may be replaced.

## 2. Current Codebase Findings

- Relevant files:
  - Product and storage boundaries: `AGENTS.md`, `atproto-craft-social-app-reference.md`, and `docs/roadmap.md`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` and `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
  - Existing profile-only presentation experiments: `app/lib/profile/widgets/profile_customisation_theme.dart`, `profile_framed_avatar.dart`, `profile_header_background.dart`, `profile_card.dart`, `profile_sliver_app_bar.dart`, `profile_route_presentation.dart`, and `app/lib/profile/pages/profile_page.dart`.
  - Shared avatar seam: `app/lib/profile/widgets/profile_avatar.dart`.
  - Current avatar consumers include posts, replies/quotes through shared post widgets, notifications, search, profile headers/cards, post summaries, editing previews, and app-shell/account presentation.
  - Flutter profile contract and repository: `app/lib/profile/models/profile.dart`, `app/lib/profile/data/profile_api_client.dart`, and `api_profile_repository.dart`.
  - Settings routing and page: `app/lib/router/route_locations.dart`, `app/lib/router/router.dart`, and `app/lib/settings/pages/settings_page.dart`.
  - AppView profile contract and persistence seam: `appview/internal/api/profile_response.go`, `profile_request.go`, `profile.go`, `profile_store.go`, `appview/migrations/000008_craftsky_profiles.up.sql`, and `000009_bluesky_profiles.up.sql`.
  - Other public identity summaries: `PostAuthor` in `appview/internal/api/post_response.go`, notification actor responses, `ProfileAccountSummary` in `profile_response.go`, and `ProfileSearchSummary` in `search_response.go`.
- Existing patterns:
  - `ProfileAvatar` is intended to be the common avatar primitive, but custom framing currently uses a separate large-profile-only wrapper. Most avatar surfaces therefore cannot render profile customisation.
  - Compact and full profile routes already share one loaded `Profile` model, but colour, background, and frame values are separately injected through route/presentation arguments rather than read from that model.
  - Profile colour currently creates a profile-local theme, the existing background is a custom-painted illustration, and the existing frame choices are decorative styles. These do not match the requested fixed texture background and three inside-border thicknesses.
  - Public list responses embed small actor/author/profile summaries so Flutter can render identity without an N+1 profile request. Customisation must follow that same embedding model.
  - `PUT /v1/profiles/me` currently writes Bluesky and Craftsky profile records to the PDS. AppView-only customisation must not be added to that PDS record body or wait for firehose convergence.
  - Private member-owned AppView tables use the authenticated DID as owner, reference `craftsky_profiles(did)`, and cascade when membership is permanently removed.
- Current behavior:
  - No customisation value exists in AppView persistence or any public API response.
  - Full and compact profile pages can render manually supplied colour/background/frame values, but ordinary profile navigation does not derive them from the fetched profile.
  - Post, reply, quote, notification, search, relationship-list, navigation, and account-switcher avatars render the standard ink border only.
  - There is no customisation settings route, catalogue, mutation, account-scoped state, or cross-device persistence.
- Constraints discovered:
  - AppView-owned data requires no lexicon, PDS record, Tap indexer, or PDS write.
  - New `/v1/*` routes must use existing authentication/device middleware, camelCase JSON, route/body policies, and the `{error, message, requestId}` error envelope.
  - Public does not mean unauthenticated in the current API: it means every authenticated viewer who receives an otherwise-visible profile identity receives the same customisation values.
  - Current block/moderation response shells intentionally restrict identity fields. Customisation may accompany an avatar that remains visible, but must not make a hidden identity or content object visible.
  - The background source describes its images as free to use, says the creator made them, and encourages edits/recolours. Selected assets still need recorded provenance and any required attribution in the repository; the app must bundle them rather than hotlink them.
  - The current highest migration is `000035`; implementation must re-check before allocating a migration number.
- Test/build commands discovered:
  - AppView tests: `just test` with the checkout's compose services available.
  - Go formatting/vetting: `just fmt`.
  - Flutter tests: `just app-test` or focused `flutter test` from `app/`.
  - Flutter analysis: `just app-analyze`.

## 3. Clarifying Questions And Decisions

### Q1: Is the existing Flutter implementation a contract?

Answer: No. The prompt explicitly says it is half-baked and may need to be rewritten.

Decision / implication: Existing widgets identify affected surfaces and reusable seams only. Requirements describe the target behavior, not preservation of current enum names, painters, route extras, or widget structure.

### Q2: Where is customisation authoritative?

Answer: In the AppView, against the relevant member, and not on the PDS.

Decision / implication: Saving customisation uses a dedicated AppView-owned mutation and durable Postgres state keyed by the authenticated member DID. It performs no PDS call, lexicon write, or firehose round trip.

### Q3: What is public and where must it be returned?

Answer: Customisation is public and must be included whenever profile information is fetched so the app can render it.

Decision / implication: Every AppView response object that exposes a visible member identity/avatar includes the member's effective `customisation` object. This includes full profiles and embedded post authors, quote/reply authors, notification actors, profile search results, and account/relationship summaries. Flutter does not issue a separate profile lookup for each avatar.

### Q4: What is the extensible wire shape?

Answer: The three current choices are simple values, and more customisation choices are expected later.

Decision / implication: Public and mutation contracts use one nested `customisation` object whose current camelCase keys are `colour`, `profileBorder`, and `profileBackground`. Each value is a stable string key from a server/client catalogue. Future fields can be added inside the object without flattening more fields across every identity response.

### Q5: How do border colour and thickness relate?

Answer: The prompt identifies one profile colour and a profile-picture border with its associated colour, while limiting the border to three thickness levels.

Decision / implication: `colour` supplies the border colour and the profile accent colour; there is no separate border-colour picker. `profileBorder` has exactly three thickness values: thin, medium, and thick. Each stroke is rendered inside the avatar's existing circular bounds and therefore does not change avatar layout size.

### Q6: What happens for members without saved state?

Answer: Existing members and older responses need deterministic rendering even though no customisation row exists.

Decision / implication: AppView and Flutter share effective defaults. Reads always expose a complete effective object for a current member, while Flutter also tolerates an absent object during a rolling deployment. The confirmed defaults are Craftsky cobalt (`#1535D6`), medium border, and no texture.

### Q7: How are background resources loaded?

Answer: The prompt says values are simple and the app renders what is needed, with no network resource loading.

Decision / implication: `profileBackground` is a stable asset key. Six confirmed transparent tiled textures are checked into the app bundle, mapped locally, and drawn over the selected profile colour only in the header region of compact and full profile views. `none` is the default catalogue value. No response contains an asset URL, filename, or arbitrary CSS/image data.

### Q8: What save interaction is confirmed?

Answer: The user confirmed an explicit Save action during requirements grilling.

Decision / implication: The page has a live preview plus an explicit Save action. Selection changes remain local until Save succeeds; failures preserve the selections for retry and do not replace the last confirmed profile state. Back with a dirty draft requires branded discard confirmation. A successful save keeps the owner on the page, marks the draft confirmed, and shows exact feedback `Profile customisation saved`.

### Q9: What remains to be selected?

Answer: Requirements grilling fixed the palette derivation, texture subset/display names, and raw border-width table. The original six colour bundles, their texture treatment, and the save-failure copy were finalized on 2026-08-10 when the user authorized implementation through completion. The user subsequently approved a seventh `ink` bundle using the theme's Ink colour.

Decision / implication: The approved colour constants in Q11 are committed catalogue inputs rather than runtime colour generation. Texture and border entries are fixed below. Exact texture and feedback assertions use the recorded values below.

### Q10: How does the custom border relate to existing avatar styling?

Answer: The user confirmed exactly three always-on thickness levels, with no borderless state. The coloured custom border replaces the current ink border rather than adding a second ring.

Decision / implication: Every avatar has one inside circular stroke using the selected profile colour. The default is medium. Existing hard-offset ink shadows remain unchanged as structural Craftsky styling. Widths are audited per current avatar size: 36 px uses 1.5/2.5/4 px, 48 px uses 2/3.5/5 px, and 96 px uses 3/5/8 px for thin/medium/thick respectively.

### Q11: How is the colour palette derived and where does it apply?

Answer: The user confirmed seven colours: cobalt plus five high-saturation colours initially generated by approximately 60-degree hue rotations and then manually tuned, plus Ink using the theme's Ink colour. Values are generated once and committed, not calculated at runtime. Each key maps locally to an audited theme bundle.

Decision / implication: A theme bundle includes the saturated base, readable foreground, hover/pressed tone, and soft container tone. The selected bundle themes everything in the compact profile view, including buttons and links. In the full profile it themes everything above the tab bar. The tab bar itself and all content below it keep the normal Craftsky theme.

Approved fixed bundles (2026-08-10):

| Key | Base | Foreground | Hover | Pressed | Soft container | Texture tint | Opacity |
|---|---|---|---|---|---|---|---|
| `cobalt` | `#1535D6` | `#FFFFFF` | `#122EBA` | `#0F279E` | `#D8DDF9` | `#FFFFFF` | 18% |
| `orchid` | `#B615D6` | `#FFFFFF` | `#9E12BA` | `#860F9E` | `#F3D8F9` | `#FFFFFF` | 18% |
| `rose` | `#D61535` | `#FFFFFF` | `#BA122E` | `#9E0F27` | `#F9D8DD` | `#FFFFFF` | 18% |
| `amber` | `#766200` | `#FFFFFF` | `#655300` | `#544500` | `#F9F3D8` | `#FFFFFF` | 18% |
| `lime` | `#23770F` | `#FFFFFF` | `#1D650C` | `#175309` | `#DDF9D8` | `#FFFFFF` | 18% |
| `teal` | `#007663` | `#FFFFFF` | `#006454` | `#005146` | `#D8F9F3` | `#FFFFFF` | 18% |
| `ink` | `#161210` | `#FFFFFF` | `#3E3733` | `#0B0908` | `#EFE7D6` | `#FFFFFF` | 18% |

The foreground applies to the base, hover, and pressed tones. Every such pair has a contrast ratio of at least 5.16:1. The base, hover, and pressed tones also maintain at least 4.5:1 contrast when used as link/accent text on the Craftsky paper and white surfaces. Soft containers use Craftsky ink `#111318` as their readable content colour. Amber, lime, and teal were darkened after simulator review on 2026-08-10 so their link and accent uses meet this surface-contrast requirement. The stable `lime` key is shown to members as `Green`; persisted values and API payloads remain `lime`.

### Q12: Which background catalogue and placement are approved?

Answer: The user approved `none` plus six Ribo source tiles: `bayerdark`, `cubedark`, `dotcrossdark`, `scallopdark`, `skewdark`, and `x2`.

Decision / implication: User-facing names are `Dither`, `Grid`, `Cross stitch`, `Scallops`, `Diagonal weave`, and `Crosshatch`. Each texture is treated as a transparent tiled mask over the selected colour, using the bundle foreground as its tint at 18% opacity. Texture is confined to the header region in both compact and full views.

### Q13: What is the confirmed editing lifecycle?

Answer: Explicit Save, dirty-draft discard confirmation, remain-on-page success, and exact success copy were confirmed.

Decision / implication: Saves publish the complete combination atomically. Pending Save prevents duplicates. Success keeps the page open, updates confirmed state, and shows `Profile customisation saved`. Back only prompts when the draft differs from confirmed state. Failure retains the draft and last confirmed public state and shows `Couldn't save your profile customisation.`

## 4. Candidate Approaches

### Option A: Dedicated AppView customisation resource with nested public identity data

Summary: Persist one AppView-owned customisation record per member DID, expose a dedicated self-mutation, and embed one nested effective `customisation` object in every public profile/actor/author response. Flutter maps stable keys through one local catalogue and renders borders through the shared avatar primitive.

Pros:

- Matches the required AppView-only boundary.
- Keeps customisation independent of atomic PDS profile records and firehose timing.
- Makes future response fields additive inside one object.
- Lets list endpoints render without per-avatar requests.
- Centralises validation, defaults, rendering, and local asset mapping.

Cons:

- Adds a migration, route, response hydration across several query shapes, generated Flutter model changes, and broad widget regression coverage.
- Requires a coordinated catalogue shared by server validation and client rendering.

Risks:

- Missing one embedded identity shape could produce inconsistent avatars.
- Duplicated client border implementations could drift at different sizes.

### Option B: Add flat customisation columns to existing profile responses and reuse `PUT /v1/profiles/me`

Summary: Add three top-level fields to every identity response and extend the existing profile mutation to split AppView fields from fields written to PDS.

Pros:

- Reuses the existing profile edit endpoint.
- Could store values directly beside `craftsky_profiles` membership data.

Cons:

- Couples an AppView-only mutation to a handler whose documented purpose is PDS profile replacement.
- Makes partial PDS failure semantics unnecessarily relevant to a local-only save.
- Repeats flat fields across every identity shape and scales poorly as options grow.
- Makes it easier to accidentally include customisation in a PDS record body.

Risks:

- A local save could fail because a PDS is unavailable or could trigger unintended public repository writes.
- Future options could cause repeated wire and handler churn.

### Option C: Keep customisation client-only

Summary: Store selections locally and pass them through route/presentation arguments as the current experiment does.

Pros:

- Requires no AppView migration or API work.

Cons:

- Other viewers and devices cannot see the customisation.
- Embedded authors in posts, replies, notifications, and search cannot render it reliably.
- Directly contradicts the public, durable, AppView-owned requirements.

Risks:

- Presentation varies by device/cache and disappears on reinstall or account changes.

## 5. Recommended Direction

Recommended approach: Option A — a dedicated AppView customisation resource, nested public identity contract, and central Flutter catalogue/rendering seam.

Why: It cleanly separates AppView-owned customisation from PDS-owned profile records, returns all rendering inputs with the identity objects that already carry avatars, avoids N+1 reads, and provides one additive namespace for future options. A dedicated persistence resource and save endpoint also make authorization, validation, deployment compatibility, and failure behavior independently testable.

## 6. Problem / Opportunity

Craftsky currently has visual experiments that suggest profile customisation but no durable or public product behavior. A member cannot manage their choices in Settings, another viewer cannot receive those choices, and the same avatar has different presentation depending on the surface. Completing the feature gives members a recognisable visual identity across the social app while keeping resource loading deterministic and retaining Craftsky's AppView/PDS privacy boundary.

## 7. Goals

- G-001: Give each current member a durable, public profile colour, avatar-border thickness, and profile-background texture selection.
- G-002: Render the selected inside border and colour consistently everywhere that member's avatar is shown.
- G-003: Render the selected local background texture in both compact and full profile views.
- G-004: Let a signed-in member preview and save fixed choices from Settings.
- G-005: Return effective customisation with existing identity/profile payloads without per-item network requests.
- G-006: Keep customisation persistence and mutation entirely in the AppView while making future fields additive.

## 8. Non-Goals

- NG-001: Do not add or modify an atproto lexicon, PDS profile field, PDS write, Tap event, or federated/portable customisation record.
- NG-002: Do not support an arbitrary colour picker, arbitrary hex/RGB values, user-uploaded backgrounds, remote background URLs, custom CSS, animation, video, or generated textures.
- NG-003: Do not add decorative frame shapes; the border choice is thickness only.
- NG-004: Do not add separate border colour, background colour, per-surface choices, or independent light/dark selections in this slice.
- NG-005: Do not redesign the profile information architecture, avatar image upload, banner image behavior, account switcher, post card, notifications, search ranking, or moderation policy beyond carrying and rendering the customisation.
- NG-006: Do not expose another member's mutation surface or customisation edit controls.
- NG-007: Do not support user-created or runtime-generated colour bundles, expand beyond the confirmed seven-colour palette, or expand beyond `none` plus the six confirmed texture backgrounds in this slice.
- NG-008: Do not promise customisation interoperability with other atproto clients or AppViews.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Profile owner | Authenticated current member editing their own appearance. | See current choices, preview fixed options, save once, and receive reliable feedback. |
| Profile viewer | Authenticated member viewing another member anywhere in the app. | See the same public colour/border identity consistently without extra loading. |
| AppView | Trusted storage, validation, and public response boundary. | Persist one owner-scoped selection, validate catalogue keys, and hydrate identity responses efficiently without PDS writes. |
| Flutter client | Consumer and renderer of simple customisation values. | Decode defaults safely, map keys to local colours/assets/widths, and render through shared seams. |
| Product/design maintainer | Owner of the curated catalogue. | Add or retire choices deliberately, with stable keys, accessibility review, and asset provenance. |

## 10. Current Behavior

Flutter can apply an optional colour theme, one of three custom-painted background illustrations, and one of three decorative large-avatar frames when values are manually passed into profile route widgets. Those values do not come from a profile model, are not editable under Settings, and are not stored or returned by the AppView. Ordinary avatars on posts, replies, notifications, search, summaries, and app navigation use only the base `ProfileAvatar` styling. The AppView profile and embedded identity contracts contain avatar URLs but no customisation data.

## 11. Desired Behavior

Every current member has an effective customisation made from one of seven fixed colours—six high-saturation hues plus theme Ink—one of three inside-border thicknesses, and either no background or one of six curated local textures. Profiles without saved state use cobalt (`#1535D6`), medium border, and no texture. A member opens `/profile/settings/customisation`, sees the last confirmed choices and a representative live profile/avatar preview, changes values through finite accessible controls, and explicitly saves. AppView authenticates the member, validates every key against its supported catalogue, atomically upserts the DID-scoped record, and returns the complete effective object. Flutter then updates/refetches account-scoped profile and identity state so the confirmed appearance is visible immediately. Success keeps the member on the page, marks the draft confirmed, and shows `Profile customisation saved`. Failure leaves the last confirmed public state unchanged and preserves the draft selections for retry. Back with unsaved changes requires branded discard confirmation.

When any allowed viewer loads a full profile, compact profile, feed post, reply/comment, quoted-post summary, notification, profile search result, relationship/account list, post summary, or navigation/account surface that shows the member's avatar, the relevant API identity object already contains the same effective customisation. The shared Flutter avatar renderer replaces the current ink border with one chosen-colour inside circular stroke at the selected thin, medium, or thick level, including while showing an initial/loading/error fallback. Existing ink drop shadows remain unchanged. The outside avatar dimensions and surrounding layout do not change. Compact and full profile header regions draw the selected bundled tiled texture over the chosen profile colour; `none` draws only the profile colour.

Each colour key maps locally to an audited theme bundle containing its high-saturation base, readable foreground, interaction tones, and soft container tone. The selected bundle themes everything in the compact profile view, including buttons and links. In the full profile it themes only the region above the tab bar. The tab bar itself and every tab-content surface retain the normal Craftsky theme.

Members with no persisted row receive catalogue defaults. New Flutter tolerates an older AppView that omits `customisation`; a new AppView emits the nested object additively so older Flutter clients ignore it. Unknown or retired stored keys fall back per field without preventing the rest of the profile from rendering. Customisation follows the DID across devices and sessions, stays isolated across accounts, and disappears only with permanent membership deletion according to AppView lifecycle policy. No render path fetches a texture, stylesheet, or customisation record separately over the network.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall give every current member one durable public profile appearance composed of a fixed colour, one of three avatar-border thicknesses, and a fixed local profile background. | Completes the requested member identity feature. | Prompt | AC-001, AC-003, AC-016 |
| BR-002 | Business | Must | A member's avatar border and associated colour shall appear consistently on every app surface that displays that member's avatar. | Recognisable identity is only useful when presentation is global. | Prompt | AC-002, AC-008, AC-009, AC-010 |
| BR-003 | Business | Must | A member's selected profile background shall appear in both compact and full profile presentations. | Preserves custom identity across the responsive profile experience. | Prompt | AC-011 |
| BR-004 | Business | Must | Members shall manage their own customisation from a page under Settings, and confirmed choices shall persist across devices and sessions. | Makes the feature usable rather than presentation-only. | Prompt | AC-003, AC-006, AC-007, AC-014 |
| BR-005 | Business | Must | Customisation shall be public through the Craftsky AppView while its authoritative data remains only in AppView persistence, not the PDS. | Matches the requested visibility and ownership boundary. | Prompt; Architecture | AC-001, AC-004, AC-016 |
| FR-001 | Functional | Must | Every AppView response object that exposes an otherwise-visible Craftsky member identity/avatar shall include a complete nested `customisation` object with the string fields `colour`, `profileBorder`, and `profileBackground`. This includes full/self profile responses, post and quote/reply author objects, available notification actors, search profile summaries, and relationship/account summaries. | Gives every rendering surface the necessary values without secondary reads. | Prompt; Codebase; Recommended direction | AC-001, AC-002, AC-015 |
| FR-002 | Functional | Must | The AppView shall expose an authenticated, device-bound `PUT /v1/profiles/me/customisation` full-replacement mutation accepting exactly the three current keys and returning the complete authoritative effective `customisation` object on success. The route shall use camelCase JSON, existing route/body/rate policies, and the standard error envelope. | Separates local customisation from the PDS-backed profile mutation and gives Flutter authoritative confirmation. | API architecture; Recommended direction | AC-003, AC-004, AC-005, AC-017 |
| FR-003 | Functional | Must | AppView shall atomically create or replace the authenticated member's complete customisation record, validate each submitted value against its server catalogue, treat an identical retry as idempotent success, reject unknown/missing/malformed fields without changing confirmed state, and return field-specific `422 validation_failed` details for unsupported catalogue values. | Prevents partial or unrenderable saved combinations and supports safe retries. | Prompt; API conventions; User grilling decision | AC-003, AC-005, AC-013 |
| FR-004 | Functional | Must | Customisation shall be stored as one AppView-owned logical record keyed by the member's canonical DID, linked to current Craftsky membership, durable across sessions/devices, isolated between accounts, and removed when membership is permanently removed. | Implements the required AppView ownership and lifecycle. | Prompt; Codebase pattern | AC-003, AC-004, AC-014, AC-016 |
| FR-005 | Functional | Must | `/profile/settings` shall include a Customisation entry leading to `/profile/settings/customisation`. The page shall load the active member's confirmed values, present only fixed catalogue choices for all three fields, show a representative live preview, expose an explicit Save action, disable duplicate saves while pending, and support Back navigation to Settings. Back with a dirty draft shall show the existing branded discard confirmation; a clean draft shall return directly. | Provides a discoverable, bounded editing flow under the existing router hierarchy without silently losing a composed draft. | Prompt; Codebase; User grilling decision | AC-006, AC-007, AC-018 |
| FR-006 | Functional | Must | On successful save, Flutter shall reconcile the initiating account's confirmed customisation from the authoritative response, invalidate/update affected profile, embedded-identity, and cached account presentation, remain on the customisation page, mark the draft confirmed, and show exact feedback `Profile customisation saved`. On failure it shall preserve confirmed public state, retain the draft selections for retry, re-enable Save, and show exact feedback `Couldn't save your profile customisation.` | Gives immediate, recoverable behavior while protecting account-scoped state and the confirmed editing lifecycle. | Existing provider/messaging patterns; User grilling decision | AC-007, AC-014 |
| FR-007 | Functional | Must | The shared avatar rendering seam shall accept effective customisation and replace the current ink border with exactly one circular stroke inside the existing avatar bounds. It shall use the selected colour and thin/medium/thick width table: 36 px avatars use 1.5/2.5/4 px, 48 px avatars use 2/3.5/5 px, and 96 px avatars use 3/5/8 px. The same rule applies to every image state; external dimensions and existing hard-offset ink shadows remain unchanged, and no second ring or decorative frame is drawn. | Implements exact border behavior consistently without layout drift or loss of the paper-cutout shadow. | Prompt; User grilling decision | AC-008, AC-009, AC-010 |
| FR-008 | Functional | Must | Every avatar-bearing Flutter surface shall pass the identity's effective customisation to the shared renderer, including feed posts, post-thread roots/replies/comments, quote previews, notifications with actors, profile compact/full headers, search profile results, relationship/account lists, post summaries, editing/customisation previews, and app navigation/account switching. A future avatar surface shall use the same seam rather than reimplementing border logic. | Covers the requested examples and current consumers. | Prompt; Codebase | AC-002, AC-008, AC-010 |
| FR-009 | Functional | Must | Compact and full profile header regions shall render the selected `profileBackground` as a transparent tiled mask over the selected profile colour, using audited tint/opacity from the colour theme bundle. The texture shall not extend beyond the header region in either view. The `none` key shall render the colour without a texture. Both presentations shall derive values from their loaded `Profile`, not route extras or caller-supplied styling. | Ensures responsive consistency, protects body-content legibility, and removes presentation-only data injection. | Prompt; Codebase; User grilling decision | AC-011, AC-012 |
| FR-010 | Functional | Must | The selected colour theme bundle shall apply to everything in the compact profile view, including buttons and links, and to everything above—but not including—the tab bar in the full profile view. The full-profile tab bar and all content below it shall retain the normal Craftsky theme. The custom colour shall not alter app-wide navigation, dialogs outside the compact profile, another member's surface, or post/tab content. | Defines the exact custom-colour boundary and prevents theme leakage. | Prompt; User grilling decision | AC-012, AC-014 |
| FR-011 | Functional | Must | Flutter shall map every supported colour, border, and background key through one versioned local catalogue. The colour catalogue shall contain seven fixed committed entries: cobalt plus five high-saturation colours initially generated at approximately 60-degree hue rotations and then manually tuned, plus Ink using theme Ink `#161210`. Each key shall resolve to an audited bundle containing base, readable foreground, hover/pressed, and soft-container tones; values shall not be generated at runtime. Background rendering shall use checked-in assets only; no API response or render path shall request a remote asset, custom CSS, or arbitrary resource URL. | Keeps values simple, deterministic, accessible, offline-capable, and extensible. | Prompt; User grilling decision | AC-005, AC-011, AC-012, AC-019 |
| FR-012 | Functional | Must | For a member with no saved record, AppView shall return a complete default object. Flutter shall also apply the same defaults when an older AppView omits the object, and shall fall back independently for an unknown/retired field while preserving valid fields. Defaults are Craftsky cobalt (`#1535D6`), `medium` border, and `none` background. | Supports existing accounts, rolling deployments, and catalogue evolution with confirmed defaults. | Codebase; User grilling decision | AC-013, AC-017 |
| FR-013 | Functional | Must | Existing moderation, block, and content-availability policy shall remain authoritative. If a response shell still exposes an avatar, it shall carry enough effective customisation to render that avatar consistently; if policy removes the identity/avatar object, customisation shall not reintroduce or disclose it. | Preserves safety boundaries while meeting all-avatar consistency. | Codebase constraint | AC-015 |
| NFR-001 | Non-functional | Must | AppView shall hydrate customisation for paginated identity collections with bounded, indexed, set-based access and shall not perform one database or HTTP lookup per returned avatar/profile. | Posts, notifications, search, and relationship lists are hot paginated paths. | Codebase; Recommended direction | AC-002, AC-019 |
| NFR-002 | Non-functional | Must | The customisation migration shall be reversible and shall preserve all existing profile, post, notification, search, relationship, moderation, and account data. | The feature adds durable cross-cutting state. | Workflow quality standard | AC-020 |
| NFR-003 | Non-functional | Must | Customisation controls, previews, borders, and backgrounds shall remain understandable with screen readers, keyboard/focus navigation, supported text scaling, light/dark themes, missing images, and colour-vision differences. Catalogue colours must maintain readable foreground contrast, and border thickness or labels—not colour alone—must distinguish choices. | Fixed visual choices must remain accessible. | UI quality standard | AC-018 |
| NFR-004 | Non-functional | Must | The nested response addition shall be backwards compatible: old clients can ignore it, new clients tolerate its absence, and existing required fields, blocked shells, endpoint statuses, and error semantics remain unchanged except for the additive mutation route. | Supports non-atomic client/server rollout. | API architecture | AC-013, AC-015, AC-017 |
| NFR-005 | Non-functional | Should | Customisation reads and mutations should emit bounded operation/result/error-class telemetry and actionable logs without using DID, colour key, border key, background key, or asset filename as metric labels. No new alert is required while the app has no production users. | Supports diagnostics without high-cardinality or preference-labelled telemetry. | Existing observability pattern; Project status | AC-019 |
| NFR-006 | Non-functional | Must | The feature shall have migration, store, handler, route-contract, response-shape, validation, query-count/plan, Flutter model/repository/provider, widget, accessibility, account-switch, and affected-surface regression coverage. Generated serialization/router outputs shall be updated through existing project commands. | The broad identity contract needs traceable regression protection. | Workflow quality standard | AC-020 |
| RULE-001 | Business rule | Must | `colour` shall be one stable key from the seven-entry approved palette: cobalt (`#1535D6`), five fixed manually tuned high-saturation hue shifts, or Ink (`ink`, theme Ink `#161210`). The API and UI shall not accept arbitrary colour values, and the same selected bundle supplies the scoped profile theme and avatar border colour. | Enforces fixed choices, deterministic rendering, and one associated colour. | Prompt; User grilling decision | AC-005, AC-012 |
| RULE-002 | Business rule | Must | `profileBorder` shall have exactly three always-on thickness levels—`thin`, `medium`, and `thick`—with `medium` as the default, no borderless state, no decorative shape, and no separately selected colour. Widths shall follow the confirmed size table in FR-007. | Implements the explicitly bounded border object. | Prompt; User grilling decision | AC-005, AC-009 |
| RULE-003 | Business rule | Must | `profileBackground` shall be `none` (the default) or one of six stable bundled Ribo texture keys: `bayerdark` (`Dither`), `cubedark` (`Grid`), `dotcrossdark` (`Cross stitch`), `scallopdark` (`Scallops`), `skewdark` (`Diagonal weave`), or `x2` (`Crosshatch`). Selected source files shall be stored locally with provenance and any required attribution; hotlinking is forbidden. | Implements the confirmed local catalogue and responsible asset use. | Prompt; Referenced asset source; User grilling decision | AC-005, AC-011, AC-019 |
| RULE-004 | Business rule | Must | Only the authenticated current member may mutate their own customisation. Every authenticated viewer who is otherwise permitted to receive that member identity receives the same effective public values; there are no viewer-specific customisation variants. | Separates owner-only control from public presentation. | Prompt; Auth architecture | AC-001, AC-004, AC-015 |
| RULE-005 | Business rule | Must | Saving customisation shall change only AppView-owned customisation state and dependent AppView/Flutter caches. It shall not write a PDS record, alter avatar/banner blobs, modify an atproto profile, emit a Tap event, or wait for indexing convergence. | Protects the requested storage boundary. | Prompt | AC-003, AC-016 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, BR-005, FR-001, RULE-004 | Given a current member has saved valid choices, when the owner or another allowed authenticated viewer fetches that member's full profile, then the response contains the same complete nested `customisation` object with `colour`, `profileBorder`, and `profileBackground` string keys. |
| AC-002 | BR-002, FR-001, FR-008, NFR-001 | Given posts, replies/comments, quote previews, available social notifications, profile search results, and relationship/account-list items by a customised member, when each paginated endpoint is fetched, then its existing embedded author/actor/profile object contains the same effective customisation and Flutter needs no follow-up profile/customisation request per item. |
| AC-003 | BR-001, BR-004, FR-002, FR-003, FR-004, RULE-005 | Given a signed-in current member submits one supported value for all three keys, when `PUT /v1/profiles/me/customisation` succeeds, then one atomic DID-scoped AppView record is created/replaced, the complete authoritative effective object is returned, an identical retry succeeds without duplicate state, and the result remains after sign-out, another device sign-in, and AppView restart. |
| AC-004 | BR-005, FR-002, FR-004, RULE-004 | Given no valid session/device context, another member's DID, or a removed/non-current member, when customisation mutation is attempted, then existing authentication/current-membership error policy rejects it, no member's record changes, and no route permits selecting a mutation target other than `me`. |
| AC-005 | FR-003, FR-011, RULE-001, RULE-002, RULE-003 | Given a request has an unknown field, omits one of the three required fields, supplies a non-string value, arbitrary colour, unsupported border/background key, or extra resource data, when it is submitted, then AppView rejects it using the standard envelope (with field-specific `422 validation_failed` for unsupported catalogue values), confirmed state remains unchanged, and no supplied URL/resource is loaded. |
| AC-006 | BR-004, FR-005 | Given a signed-in member is on `/profile/settings`, when they activate Customisation, then `/profile/settings/customisation` opens and the page shows the active account's confirmed colour, border, and background choices plus a representative preview. Given a clean draft, Back returns directly to Settings; given a dirty draft, Back first shows branded discard confirmation. |
| AC-007 | FR-005, FR-006 | Given the member changes one or more selections, when Save is pending, then duplicate saves are disabled while navigation/focus remains coherent. On success, the page remains open, the authoritative choices become confirmed, newly rendered surfaces reflect them without restart, and exact feedback `Profile customisation saved` is shown. On failure, prior confirmed public state remains, draft selections remain available for retry, Save is re-enabled, and exact feedback `Couldn't save your profile customisation.` is shown. |
| AC-008 | BR-002, FR-007, FR-008 | Given a customised member appears in each current avatar-bearing surface, when the surface renders, then the same shared avatar seam replaces the old ink border with exactly one selected-colour inside border on feed posts, thread roots/replies/comments, quote previews, notifications, compact/full profiles, search, relationship/account lists, post summaries, previews, and navigation/account presentation, while retaining the surface's existing ink shadow behavior. |
| AC-009 | BR-002, FR-007, RULE-002 | Given identical 36 px, 48 px, and 96 px avatars, when `thin`, `medium`, and `thick` are rendered, then their widths are respectively 1.5/2.5/4 px, 2/3.5/5 px, and 3/5/8 px; all strokes stay inside the circle, image/fallback content remains clipped, external dimensions and surrounding layout are unchanged, and no borderless, second-ring, or decorative-frame state appears. |
| AC-010 | BR-002, FR-007, FR-008 | Given avatar URL is absent, loading, or fails, when the initial fallback renders, then the selected inside border thickness and colour remain visible and the fallback/semantics match the existing avatar behavior. |
| AC-011 | BR-003, FR-009, FR-011, RULE-003 | Given `profileBackground` is one of Dither, Grid, Cross stitch, Scallops, Diagonal weave, or Crosshatch, when the same profile opens compact and full screen, then both header regions use the matching bundled transparent tiled mask over the selected colour with audited tint/opacity and no network request; the texture does not extend beyond either header. Given `none`, both render the colour without a texture. |
| AC-012 | FR-009, FR-010, FR-011, RULE-001 | Given two profiles use different supported colour bundles, when each compact profile renders, then its entire view—including buttons and links—uses its own audited base/foreground/interaction/container tones. When each full profile renders, the same theme ends immediately before the tab bar; the tab bar and content below it use the normal Craftsky theme. Avatars use the selected base colour, and app-wide chrome/other profiles are not recoloured. |
| AC-013 | FR-003, FR-012, NFR-004 | Given an existing member has no row, a new Flutter client receives a response without `customisation`, or one persisted/received field is unknown while the others are valid, when the profile renders, then defaults are applied only where needed—cobalt (`#1535D6`), `medium`, and `none`—without rejecting the profile or discarding valid fields. |
| AC-014 | BR-004, FR-004, FR-006, FR-010 | Given two accounts have different confirmed choices and the active account changes while a save is in flight, when the request settles, then the completion updates only the initiating account's record/caches, cannot recolour the newly active account, and each account recovers its own values when selected. |
| AC-015 | FR-001, FR-013, NFR-004, RULE-004 | Given an identity is muted, blocked, blocking, unavailable, warned, or hidden, when existing response policy is applied, then customisation accompanies an avatar only where that shell already exposes the avatar, does not restore stripped fields/content, and cannot bypass moderation or relationship visibility. System notifications without an actor remain actor-free. |
| AC-016 | BR-001, BR-005, FR-004, RULE-005 | Given save, read, account switch, session expiry, and permanent membership-deletion cases, when storage and outbound activity are inspected, then only AppView customisation state and dependent caches change; no lexicon/PDS/Tap/blob operation occurs, state survives session/device changes, and permanent membership deletion removes only that owner's record. |
| AC-017 | FR-002, FR-012, NFR-004 | Given API contract tests with old/new response fixtures and the customisation mutation, then existing endpoints retain their statuses/error shapes, customisation is an additive nested object, older clients can ignore it, newer clients tolerate its absence, and the new route is authenticated/device-bound, camelCase, full-replacement, idempotent, and returns its authoritative object. |
| AC-018 | FR-005, NFR-003 | Given touch, keyboard, screen reader, supported text scales, and light/dark themes, when the settings controls and preview are used, then every choice has a readable label and selected state, focus order/Back/Save/discard confirmation are operable, no option depends on colour/texture alone, all audited colour-bundle foregrounds and interactions remain readable, and content does not clip. |
| AC-019 | FR-011, NFR-001, NFR-005, RULE-003 | Given multi-item profile-bearing endpoints and successful/failed mutations, when queries, network calls, repository assets, logs, and telemetry are inspected, then customisation hydration is bounded/indexed/set-based, no per-item or remote-asset request occurs, selected textures have provenance/attribution records, and telemetry uses bounded operation/result/error classes without preference values or identifiers as metric labels. |
| AC-020 | NFR-002, NFR-006 | Given the feature verification suite runs, then it covers migration up/down/up; default/upsert/member-deletion persistence; auth and validation; all public identity response shapes; query count/plan behavior; old/new wire compatibility; active-account fencing; settings loading/save/failure/navigation/accessibility; three inside-border levels at all avatar sizes/states/surfaces; compact/full local backgrounds; colour scoping; blocked/moderated shells; no-PDS/no-network guarantees; and existing profile/feed/thread/notification/search/relationship regressions. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Existing member has no customisation row. | Return and render the complete approved defaults without eagerly requiring a row to be created. | FR-012 |
| EC-002 | One stored field contains a retired key after a catalogue change. | Fall back for that field, preserve other valid values, log a bounded diagnostic, and keep the profile renderable. | FR-012, NFR-005 |
| EC-003 | Older AppView omits the nested object. | New Flutter renders defaults and keeps profile/avatar surfaces usable. | FR-012, NFR-004 |
| EC-004 | Older Flutter receives the nested object. | It ignores the additive field and continues decoding all existing identity responses. | NFR-004 |
| EC-005 | A list contains repeated appearances of one author. | Every appearance renders the same embedded effective customisation without additional HTTP/profile calls. | FR-001, NFR-001 |
| EC-006 | Avatar image is null, loading, corrupt, or evicted. | Fallback initial remains visible with the chosen border and unchanged dimensions. | FR-007 |
| EC-007 | Very small avatar with thick border. | A 36 px avatar uses the confirmed 4 px inside stroke, leaving fallback/image content legible, external dimensions unchanged, and the existing shadow unaffected. | FR-007, NFR-003 |
| EC-008 | Texture has transparency and profile colour changes. | The local texture remains the same selection, layers over the new profile colour using the new bundle's audited tint/opacity, and stays within the header region in both profile modes. | FR-009, FR-010 |
| EC-009 | Save fails after draft preview changed. | Last confirmed public appearance remains authoritative; draft preview/selections remain for retry and do not leak into unrelated surfaces. | FR-006 |
| EC-010 | Two devices save different valid combinations concurrently. | Each complete replacement is atomic; last committed write wins and subsequent reads return one complete combination, never mixed fields. | FR-003, FR-004 |
| EC-011 | Active account switches while loading or saving. | Lease/account fencing prevents the previous account's completion from changing the new active account's page, cache, feedback, or theme. | FR-006, FR-010 |
| EC-012 | Blocked profile shell still shows an avatar. | Include/render the border customisation with that avatar while keeping all other stripped profile fields and content stripped. | FR-013 |
| EC-013 | Notification has no available actor. | Do not synthesize a customisation or avatar object; preserve the existing unavailable/system-notification presentation. | FR-013 |
| EC-014 | Member is permanently removed and later rejoins. | Cascaded old customisation is gone; the rejoined member begins with current defaults unless they save new choices. | FR-004 |
| EC-015 | Asset key is valid server-side but missing in the installed client catalogue. | Treat only that background as unknown and render `none`; do not crash, fetch remotely, or lose the valid colour/border. | FR-011, FR-012 |
| EC-016 | Rapid repeat Save taps or retry after timeout. | Only one request is initiated while pending; an identical retry is idempotent and returns the authoritative complete object. | FR-003, FR-005 |
| EC-017 | Owner presses Back after editing, after saving, or after reverting the draft to confirmed values. | Show branded discard confirmation only while the draft differs from confirmed state; return directly once Save succeeds or the draft again equals confirmed state. | FR-005 |

## 15. Data / Persistence Impact

- New fields:
  - One AppView-owned logical customisation record per current member DID.
  - Current values: `colour`, `profile_border`, and `profile_background` (physical SQL naming/schema is finalized in coding design).
- Changed fields:
  - Additive nested `customisation` on public profile and embedded identity response shapes.
  - Add corresponding optional/tolerant Flutter fields/value model and generated mappings.
- Migration required:
  - Yes. Add reversible owner-keyed persistence with membership lifecycle and efficient joins/lookups. Re-check the migration sequence after current `000035` before implementation.
- Backwards compatibility:
  - No backfill is required; missing rows resolve to defaults.
  - New AppView responses are additive for old clients.
  - New Flutter treats a missing object or unknown individual key as defaults during rolling deployment/catalogue evolution.
  - Physical storage should permit future fields without changing the public meaning of existing stable keys; the coding plan will choose explicit columns versus a validated JSON object.

## 16. UI / API / CLI Impact

- UI:
  - Add a Customisation tile under Settings and `/profile/settings/customisation` page.
  - Add fixed controls and live preview for the seven-colour theme catalogue, thin/medium/thick border, and `none`/six confirmed texture backgrounds.
  - Use explicit Save, branded dirty-draft discard confirmation, remain-on-page success, and exact success feedback `Profile customisation saved`.
  - Replace presentation-only route extras with values from the loaded profile/customisation state.
  - Extend the shared avatar seam with the confirmed inside-width table, replace the ink stroke, preserve ink shadows, and audit every current avatar consumer.
  - Apply the colour theme to the entire compact profile and only above the full-profile tab bar; keep the tab bar/content on the normal theme.
  - Confine local texture rendering to the header region in both profile modes.
  - Bundle the approved texture subset and record provenance/attribution.
- API:
  - Add `PUT /v1/profiles/me/customisation` with full-replacement validation and authoritative response.
  - Add nested effective `customisation` to every relevant full and embedded identity response.
  - No new public per-user read route is required because public values travel with identity payloads and self values are available from `GET /v1/profiles/me`.
- CLI:
  - None identified.
- Background jobs:
  - None required. No backfill or firehose convergence is needed.

## 17. Security / Privacy / Permissions

- Authentication:
  - Mutation requires the existing Craftsky session and device context.
  - Existing profile/collection read authentication remains unchanged.
- Authorization:
  - Mutation target is always the authenticated member (`me`); clients cannot supply an owner DID.
  - Current membership is required and membership deletion owns persistence cleanup.
- Sensitive data:
  - The three effective choices are intentionally public within Craftsky identity responses.
  - No PDS token, private profile state, local asset path, arbitrary URL, or raw request body belongs in responses or telemetry.
- Abuse cases:
  - Server allowlists prevent arbitrary colours, oversized/untrusted assets, CSS injection, URL fetching, or malformed rendering keys.
  - Existing moderation/block shells remain authoritative; customisation cannot make hidden identity/content visible.

## 18. Observability

- Events:
  - Bounded mutation attempt/success/failure and default/fallback diagnostics where existing conventions support them.
- Logs:
  - Request ID, operation, result/error class, and safe validation field names. Avoid full request bodies, DIDs, and catalogue preference values in ordinary logs.
- Metrics:
  - Mutation result/error class and optionally read-fallback count with bounded labels. Never use DID or selected option/asset as a label.
- Alerts:
  - None required while the project has no production users. Revisit with production readiness.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | One or more embedded identity response shapes omit customisation. | Avatars differ by surface or require N+1 repair requests. | Inventory all response structs and avatar consumers; add shared contract fixtures and surface matrix tests. |
| RISK-002 | Server and client catalogues drift during deployment. | Valid saved keys may be unrenderable or new saves may be rejected. | Stable versioned keys, tolerant per-field client fallback, additive rollout, and contract fixtures shared in test data. |
| RISK-003 | Border width is implemented outside the avatar or independently per widget. | Layout shifts, clipping, and inconsistent thickness. | One shared inside-stroke renderer with golden/widget coverage at every supported avatar size and state. |
| RISK-004 | Customisation becomes coupled to the existing PDS profile save. | A local appearance save may contact/fail with the PDS or accidentally become federated. | Dedicated AppView route/service/store and explicit no-PDS integration tests. |
| RISK-005 | Adding customisation joins to multiple hot list queries causes N+1 access or regressions. | Feed, notification, search, or relationship latency grows with page size. | Indexed owner key, set-based joins/hydration, query-count tests, and relevant query-plan review. |
| RISK-006 | The five shifted high-saturation colours or transparent textures reduce text/button/link/avatar legibility. | Profiles become inaccessible or hard to scan. | Commit audited per-colour foreground/interaction/container tones, tune texture tint/opacity per bundle, review both profile modes, and add contrast/preview/accessibility tests. |
| RISK-007 | Source texture usage/provenance is not documented. | Future maintainers cannot verify reuse/attribution rights. | Check selected assets into the repository only after recording source URL, creator statement, retrieval date, modifications, and any attribution requirement. |
| RISK-008 | Active-account async completions leak presentation across accounts. | Wrong profile appearance or feedback is shown after switching accounts. | Reuse active-account lease/fencing and account-keyed caches; test in-flight account switching. |
| RISK-009 | The broad current Flutter experiment is preserved piecemeal. | Old decorative frames/illustrations and new thickness/textures conflict. | Treat existing enums/painters/route extras as replaceable and remove obsolete paths in the coding plan with regression coverage. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-005 | “Public” means consistent for all authenticated API viewers allowed to receive the identity under current policy, not a new unauthenticated endpoint. | Read-route authentication and threat analysis would expand. |
| ASM-007 | A dedicated AppView persistence resource is preferred over placing lifecycle-independent preference columns directly on `craftsky_profiles`. | Physical migration/store design changes, but public and UI requirements remain the same. |

## 21. Open Questions

- [x] Pre-visual-implementation gate closed 2026-08-10: Approved and recorded the `cobalt`, `orchid`, `rose`, `amber`, `lime`, `teal`, and `ink` base/foreground/hover/pressed/soft-container constants in Q11.
- [x] Pre-visual-implementation gate closed 2026-08-10: Use each bundle foreground as the texture tint at 18% opacity.
- [x] Pre-copy-test gate closed 2026-08-10: Use exact failure feedback `Couldn't save your profile customisation.`

## 22. Review Status

Status: Approved
Risk level: Medium
Review recommended: Yes
Reviewer: User
Date: 2026-08-10
Notes: Requirements grilling confirmed effective defaults; exactly three always-on inside borders and their per-size widths; one replacement colour stroke with preserved ink shadow; cobalt default; six-colour hue-shift palette design plus the later approved theme-Ink choice; audited colour theme bundles and exact theme boundaries; `none` plus six named Ribo textures; header-only texture placement; and explicit Save/discard/success/failure behavior. All implementation-input gates are closed.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001 through BR-005; FR-001 through FR-013; NFR-001 through NFR-004 and NFR-006; RULE-001 through RULE-005.
- Suggested test levels:
  - AppView migration up/down/up and membership cascade integration tests.
  - Store atomic upsert, defaults, concurrency/idempotency, and query-count/query-plan tests.
  - Handler/route validation, auth/device, no-PDS, error-envelope, and additive response contract tests.
  - Fixture tests for full profile, post/quote/reply authors, notification actors, search summaries, relationship/account summaries, and blocked/unavailable shells.
  - Flutter model/repository/provider tests for absent/unknown fields, save reconciliation/failure, active-account fencing, and cache invalidation.
  - Widget/golden/semantics tests for settings Save/discard/success behavior, the exact border-width table at all sizes/states, preserved ink shadows, the six compact/full header backgrounds, exact colour-theme boundaries, light/dark themes, and every avatar-bearing surface.
  - Asset/provenance and no-remote-resource checks.
  - Existing profile/feed/thread/notification/search/relationship/navigation regression suites.
- Blocking open questions: None. All catalogue, texture-style, and feedback-copy inputs are recorded above.
