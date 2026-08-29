# Requirements: Business Profiles

## 1. Initial Request

Let a Craftsky member present their existing account as a craft-related business. The business profile supports optional public business details, featured external products, one primary external action, and business-appearance events without introducing Pro or Business subscriptions. This implementation slice ends at lexicons and the Go AppView; Flutter integration is deferred.

Discovery selected an AppView-authoritative scalar account type, a portable but non-activating public business declaration, and first-class public event records. Featured products route to external commerce rather than forming a native Craftsky catalog.

## 2. Current Codebase Findings

- Relevant files:
  - `lexicon/social/craftsky/actor/profile.json` defines the current Craftsky membership record.
  - `docs/superpowers/specs/2026-04-23-profile-onboarding-design.md` defines the split between `app.bsky.actor.profile` and the Craftsky profile.
  - `appview/internal/index/craftsky_profile.go` and `appview/internal/ingestion/service.go` couple Craftsky profile presence to membership lifecycle.
  - `appview/internal/app/deps.go` and `docker-compose.yml` register and forward supported record collections.
  - `appview/internal/api/profile*.go` provides the existing combined profile read/edit path.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` governs authenticated `/v1/` routes, pagination, errors, and URL conventions.
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md` requires camelCase `/v1/` JSON.
  - `appview/internal/accountdeletion/collections.go` controls the registered Craftsky collections removed during explicit permanent account deletion.
  - `https://lexicon.garden/lexicon/did:plc:mtr7qrqtcyseedx3jyr5o7db/community.lexicon.location.address/docs` defines the reused community address object.
- Existing patterns:
  - `app.bsky.actor.profile/self` owns display name, bio, avatar, and banner.
  - `social.craftsky.actor.profile/self` owns Craftsky preferences and signals current membership.
  - Independent content uses TID-keyed PDS records; bounded subordinate data may be embedded in a parent record.
  - Writes pass through the AppView to the PDS; reads come from AppView projections.
  - Projectors are idempotent by record URI/CID and retain public source data separately from hydrated API responses.
  - Existing AppView-mediated content creation establishes the event `createdAt` behavior for this slice: server-stamp on create, reject client input, and preserve on update.
- Current behavior:
  - Every current member has an ordinary combined profile; there is no account type, business declaration, featured-product data, event record, business API, or business UI.
  - Deleting `social.craftsky.actor.profile/self` means leaving Craftsky and removes serving projections.
  - Blocked profile responses are intentionally reduced and must not leak additional profile data.
- Constraints and established contracts:
  - All PDS record data is public, including locations, links, prices, and events.
  - Account type is authoritative private AppView state and is not duplicated on the PDS.
  - Existing image constraints are JPEG, PNG, or WebP up to 15 MiB, optional aspect ratio, and optional alt text bounded to 1000 graphemes and 1000 bytes.
  - Lexicon changes require an ADR, generated Go types, indexers, migrations, Tap forwarding, account-deletion registration, and tests.
  - External definition references must be pinned exactly and reproducibly for lexicon generation.
  - Lexicon evolution permits additive optional fields but does not safely permit later widening of array maxima for old consumers.
  - Business accounts must not receive algorithmic ranking or paid reach.
- Test/build commands discovered:
  - `just lexgen`
  - `just test`
  - `just fmt`

## 3. Clarifying Questions And Decisions

### Q1: What is a featured product?

Answer: An ordered external product card with title, destination URI, required image on Craftsky writes, and optional alt text and price.

Decision / implication: Products remain embedded in the business declaration. They have no native Craftsky identity, inventory, checkout, material taxonomy, or detail page. The lexicon minimum is title and URI so independently authored cards without images remain valid source records, but Craftsky writes require images.

### Q2: How are product limits handled?

Answer: The lexicon permits 20 cards and Craftsky writes permit four.

Decision / implication: Four is a code-level constant, not runtime configuration. Products retain authored order. Craftsky rejects exact duplicate product URIs without aggressive URI normalization.

### Q3: What does a business event represent?

Answer: A first-class record describing the publishing account's appearance in one or more roles: organizer, instructor, vendor, exhibitor, speaker, or demonstrator.

Decision / implication: The record has its own URI and lifecycle. It may link to an external event and registration destination but is not a globally canonical Craftsky event.

### Q4: How does business identity relate to personal identity?

Answer: The same account may set its authoritative AppView account type to `regular` or `business` and continues to use its existing identity profile.

Decision / implication: `regular` and `business` are the only supported account types; no reserved `pro` value exists. Any current member may set `business`, even when no declaration exists. Account type survives ordinary membership departure and is restored on rejoin; permanent Craftsky account deletion removes it.

### Q5: How is business classification represented?

Answer: Business types and offerings are unordered, distinct selections from canonical catalogs. They are display-only.

Decision / implication: Craftsky writes accept recognized values only and return them in canonical catalog order. Ingestion preserves unknown lexicon-valid values and APIs represent them safely. Taxonomies do not affect search, filtering, ranking, permissions, or event eligibility. `other-craft-business` is explained through tagline or bio; there is no custom taxonomy field.

### Q6: What location and operating-hours detail is needed?

Answer: Reuse the externally defined `community.lexicon.location.address`; project only canonical uppercase ISO 3166-1 alpha-2 `country` and optional `locality`. Hours are optional display-only free text.

Decision / implication: The external reference must be pinned exactly and reproducibly for lexgen. Craftsky mutations and responses include only `country` and `locality`. Independent extra address fields remain in raw source but are omitted from hydrated APIs. An oversized locality is omitted while a valid country remains; a non-alpha-2 country omits the entire location projection without suppressing the containing record.

### Q7: How does the profile answer where the business serves?

Answer: With one optional free-text `serviceArea`.

Decision / implication: `serviceArea` is display-only and is not interpreted. There is no structured worldwide flag, country list, or region expansion.

### Q8: How do calls to action and contact work?

Answer: There is one optional primary typed action and no secondary-link collection.

Decision / implication: Q14 fixes the exact shared web-destination and mailto grammars. Independently authored unsafe destinations remain in raw source but are omitted when hydrated.

### Q9: What does the business label mean?

Answer: It is an owner-selected, self-declared account classification.

Decision / implication: It does not imply subscription, legal verification, identity verification, recommendation, or endorsement. It does not change ranking, reach, search, filtering, or permissions.

### Q10: What happens when account type, membership, or declaration changes?

Answer: Public business classification and event eligibility derive from current membership plus authoritative account type `business`; declaration presence is irrelevant to both.

Decision / implication: Deleting the declaration removes only declaration-backed details. The account remains business and eligible events remain visible. Switching to `regular` updates and retains the authoritative private account-type row with value `regular`; it suppresses event serving but retains public PDS records. Leaving Craftsky suppresses business surfaces; rejoining restores the persisted account-type value and read-time eligibility. Permanent deletion removes events, declaration, account type, then membership.

### Q11: May regular members publish business records?

Answer: Yes. Any authenticated current member may create or edit their own declaration and event records as public PDS setup.

Decision / implication: AppView indexes records independently of membership, account type, and declaration, preserves raw source, and applies public eligibility only at read time. Regular accounts' declaration details and events are publicly suppressed by Craftsky until account type becomes `business`.

### Q12: How are events timed, listed, and moderated?

Answer: Craftsky writes require status, mode, timezone, start, end, roles, and created-at behavior; public event listing is chronological and eligibility is computed at read time.

Decision / implication: `past` and `completed` are derived from `endsAt`, not authored statuses. Public direct reads keep normal-duration past, cancelled, and postponed events. Upcoming lists exclude ended, cancelled, and postponed events. Owner management includes all events and bounded suppression reason codes. Individual event reporting and moderation are in scope.

### Q13: What is the exact owner event-management contract?

Answer: `GET /v1/events` is the authenticated current owner's all-events collection. It defaults to 20 items, permits at most 50, orders by `startsAt DESC, URI DESC`, and uses an opaque seek cursor. The response is `{items, cursor?}`; every item is the management event view with non-null, distinct, canonical-order `publicSuppressionReasons` and `upcomingExclusionReasons` arrays.

Decision / implication: Public suppression reason values are exactly `owner-not-business`, `invalid-time-range`, `duration-exceeds-limit`, and `record-moderated`. Upcoming exclusion values are exactly `ended`, `cancelled`, `postponed`, `owner-not-business`, `invalid-time-range`, `duration-exceeds-limit`, and `record-moderated`, in that canonical order. Empty arrays serialize as `[]`. Event reports use the existing record-report pattern at `POST /v1/events/{did}/{rkey}/reports`.

### Q14: What validation applies to external destinations and country codes?

Answer: Every first-party web destination in an action, product, event, or registration link must parse with Go `url.Parse` as a non-opaque absolute URI, use case-insensitive `https`, have no userinfo, and be at most 2048 bytes; query and fragment are allowed. The host must be an ASCII DNS name of 2-253 bytes with at least two labels, no trailing dot, labels of 1-63 ASCII letters/digits/hyphens without leading/trailing hyphen, and an optional decimal port 1-65535. Raw Unicode/IDNA input, IP literals, single-label names, percent-encoded authority, and empty hosts are rejected; callers may provide an already ASCII `xn--` punycode label. Email is exactly lowercase `mailto:` followed by one ASCII dot-atom addr-spec with one `@` and the same DNS grammar without a port, at most 320 bytes for the address, with no whitespace, control characters, percent encoding, query, fragment, comma, or semicolon. Country must be an assigned ISO 3166-1 alpha-2 code in the pinned code catalog, accepted case-insensitively and returned uppercase.

Decision / implication: Product links now share the same no-credentials/HTTPS/host/length protection as event and non-email action links. The AppView never resolves, fetches, previews, or otherwise contacts any authored destination. The country catalog is generated from a checked-in ISO 3166-1 Online Browsing Platform snapshot retrieved 2026-08-28, stored under `appview/internal/business/catalogdata/` with retrieval metadata and SHA-256 sidecar. Production uses generated Go data; a drift test verifies the source digest and generated table. Updating the snapshot, digest, date, or generated catalog is a reviewed code change, not runtime configuration.

### Q15: What is the canonical price grammar and currency catalog?

Answer: The integer part is `0` or a nonzero digit followed by up to 11 digits. A fraction is optional only when the currency minor-unit scale is nonzero, contains one through that many digits, and ends in a nonzero digit. Therefore USD accepts `1`, `1.2`, and `1.23` but rejects `1.20`; zero is exactly `0`; a zero-scale currency rejects fractions.

Decision / implication: The versioned code catalog is generated from a checked-in SIX Group ISO 4217 Maintenance Agency List One XML snapshot retrieved 2026-08-28, stored under `appview/internal/business/catalogdata/` with retrieval metadata and SHA-256 sidecar. Active alphabetic entries with numeric minor-unit scales are supported; entries with minor unit `N.A.` and withdrawn List Three codes are unsupported. Production uses generated Go data; a drift test verifies the source digest and generated table. Catalog updates are reviewed code changes. The API returns only seller-authored amount/currency data and adds no disclaimer or commerce-guarantee fields.

### Q16: How do owner and visitor direct event reads differ?

Answer: On `GET /v1/events/{did}/{rkey}`, an authenticated current owner reading their own retained event receives the management event view and diagnostics even when account type, duration, lifecycle, or record moderation suppresses public serving. Other authenticated visitors receive `404 event_not_found` whenever the event is absent, blocked, moderated, over-duration, owned by a non-member, or owned by a regular account.

Decision / implication: Eligible normal-duration past, cancelled, and postponed events remain directly readable to visitors even though they are excluded from upcoming. A departed owner is no longer a current member and cannot use owner-management access. Blocked event lists are empty and blocked direct reads return `404 event_not_found`; no redacted event shell is returned.

### Q17: How are external schema pinning and out-of-order projection made reproducible?

Answer: The address schema record is retrieved by resolving its DID/PDS and calling `com.atproto.repo.getRecord` for the exact AT-URI, then verifying the returned CID is `bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq`. Its complete record value, including `$type`, is vendored at `appview/cmd/lexgen/external/community.lexicon.location.address.bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq.json`. The ADR records the retrieval command/output and canonical DAG-CBOR/CID verification procedure. A contract test re-encodes the vendored record with atproto DAG-CBOR rules and recomputes the CID; `just lexgen` maps the external NSID to only this local file and performs no network access.

Decision / implication: Projected business rows and deletion tombstones store the Tap repository revision TID. For one record URI, an event applies only when its revision is newer than the stored row or tombstone; equal/older replay, create, update, or delete events are no-ops. A newer delete leaves a revision tombstone so an older create/update cannot resurrect the record. Duplicate delivery at the same URI/CID/revision is idempotent.

## 4. Candidate Approaches

### Option A: Extend The Membership Profile

Summary: Add business fields, products, and events to `social.craftsky.actor.profile/self`.

Pros: One existing record and projection.

Cons: Couples business editing to membership lifecycle, rewrites membership for subordinate changes, and prevents independent event identity and pagination.

### Option B: Separate Declaration With Embedded Products And Events

Summary: Add `social.craftsky.business.profile/self` containing all business data as bounded arrays.

Pros: Separates public business details from membership.

Cons: Every event change rewrites the declaration; events cannot be independently addressed, moderated, or paginated.

### Option C: Authoritative Account Type, Optional Declaration, And First-Class Events

Summary: Keep AppView-authoritative `regular`/`business` scalar state; add `social.craftsky.business.profile/self` for optional portable details and ordered product cards; add `social.craftsky.business.event/{tid}` for appearances. Classification and event eligibility do not depend on declaration presence.

Pros:

- Separates classification, optional public details, and membership lifecycle.
- Allows business classification before a declaration exists and keeps event visibility after declaration deletion.
- Keeps deliberately small product curation simple.
- Gives events stable identity, independent CRUD, moderation, diagnostics, pagination, and date indexing.
- Indexes portable public records without turning them into permissions or activation signals.

Cons:

- Adds private account-type persistence and two complete PDS collection paths.
- Requires centralized read-time membership/type eligibility and careful redaction across summary shapes.

Risks: Out-of-order firehose events, unknown independent values, and unsafe independently authored fields require raw preservation plus safe hydration rather than destructive normalization.

### Option D: Reuse `community.lexicon.calendar.event`

Summary: Publish the generic community calendar record as the Craftsky business event.

Pros: Reuses an existing event NSID and general calendar vocabulary.

Cons: Its optional timing and generic event meaning do not encode Craftsky's required appearance lifecycle, roles, limits, and owner-management semantics.

## 5. Recommended Direction

Recommended approach: Option C.

Why: It preserves membership and ordinary identity ownership, makes account classification explicit and reversible, keeps declaration fields fully optional, permits portable setup by regular members, and matches the different lifecycle needs of product cards and dated appearances without introducing commerce, Flutter, subscriptions, or declaration-based entitlement.

## 6. Problem / Opportunity

Craft businesses currently use ordinary profiles that cannot consistently identify a self-declared business, describe optional business details, feature external products, or publish independently addressable appearances. A private authoritative classification plus portable public details can add transparency while keeping transactions external and avoiding ranking or permission effects.

## 7. Goals

- G-001: Let any current member select `regular` or `business` independently of declaration creation.
- G-002: Report `regular` or `business` consistently on normally visible profiles and author summaries.
- G-003: Let business accounts display optional types, offerings, tagline, hours note, service area, general location, and one external action.
- G-004: Display up to four ordered external product cards on Craftsky while allowing a schema ceiling of 20.
- G-005: Let current members publish and manage independently addressable business appearances, with public event serving limited to current business accounts.
- G-006: Preserve portable public records and unknown valid source fields while keeping authoritative account type private.
- G-007: Preserve chronological, display-only, non-pay-to-play product principles.
- G-008: Support individual event reporting, moderation, and owner diagnostics.

## 8. Non-Goals

- NG-001: Pro or Business subscriptions, entitlements, billing, trials, feature gates, or a `pro` account type.
- NG-002: Legal, trader, identity, or platform verification.
- NG-003: Native catalogs, variants, inventory, checkout, orders, tax, shipping, price synchronization, or custom taxonomy values.
- NG-004: Native messaging, lead forms, booking, registration, ticketing, attendee management, or private event instructions.
- NG-005: Reviews, ratings, recommendations, advertising, boosted reach, algorithmic ranking, or taxonomy-driven search/filter behavior.
- NG-006: Separate business names, represented brands, teams, or production-partner disclosures.
- NG-007: Street-address hydration, coordinates, maps, multiple locations, structured hours, interpreted service regions, or a business directory.
- NG-008: Event recurrence, canonical shared Craftsky events, global event discovery, booth maps, capacity, waitlists, refunds, accessibility notes, or `onlineUri`.
- NG-009: Rich product fields such as materials, dimensions, care, lead time, or availability.
- NG-010: Changes to ordinary identity ownership or the PDS membership record shape.
- NG-011: Flutter models, repositories, providers, routes, screens, widgets, navigation, or client-side publication warnings.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Member/business owner | Current Craftsky member preparing or presenting business data | Select account type and manage public details, products, and events without another account |
| Profile visitor | Authenticated member viewing a normally visible profile | See authoritative classification and eligible, safely hydrated business data |
| AppView | Authenticated write mediator and public-data index | Validate first-party writes, preserve independent source, enforce read eligibility, and redact blocked shells |
| Moderator/reporter | Authenticated actor using existing moderation boundaries | Report and moderate an individual event |
| Independent AT Protocol client | Third-party author or reader | Publish conforming records with unknown values and retain fields Craftsky does not hydrate |

## 10. Current Behavior

All current members have the same ordinary profile shape. Profiles expose identity, bio, crafts, counts, relationship state, and customisation, but no authoritative account classification or business data. There is no business event collection or profile-scoped event endpoint.

## 11. Desired Behavior

Any current member can set authoritative account type to `regular` or `business`, even without a declaration. Every normally visible profile and supported author/account summary reports one of those values. A blocked shell omits account type and every business field.

Any current member can prepare a valid, possibly empty business declaration and valid events through Craftsky. Records are indexed and preserved independently of membership/type/declaration state. Public declaration details and events are hydrated only for current members whose authoritative type is `business`; event eligibility never requires a declaration. Deleting the declaration removes only declaration-backed details. Switching to `regular` persists `regular` in the private account-type row. Ordinary departure retains the row's current value and public PDS records; rejoin restores eligibility from that persisted value. Permanent account deletion removes owned events, declaration, private account type, then membership.

Business events are fetched from a dedicated profile events endpoint rather than embedded in the main profile. Visitors get a chronological upcoming list whose cursor freezes the eligibility cutoff `asOf`; concurrent record mutations follow normal seek-pagination semantics. Normal-duration past, cancelled, and postponed events remain available by direct read. Owners get a bounded all-events management view with suppression reason codes. Individual event reporting and moderation use existing policy boundaries.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A current member shall be able to select or remove public business presentation on the same account without creating a declaration or another account. | Classification is optional platform state. | Initial request, User interview | AC-001, AC-002 |
| BR-002 | Business | Must | Business details shall help visitors understand what the business does, its general location/service area, and its primary external action. | Supports transparent presentation without directory semantics. | Initial request, User interview | AC-003 |
| BR-003 | Business | Must | Members shall be able to feature external products and publish independently addressable event appearances. | Delivers the requested profile enhancements. | Initial request | AC-004, AC-005 |
| BR-004 | Business | Must | Business classification and records shall not introduce paid access, verification, ranking, search, filtering, reach, or permission advantages. | Preserves product principles. | Initial request, Product vision | AC-006 |
| FR-001 | Functional | Must | AppView shall persist a separate authoritative scalar `accountType` with exactly `regular` and `business`; missing state defaults to `regular`, and no public PDS account-type field or `pro` value shall be introduced. | Avoids duplicated authority and reserved behavior. | User interview, Codebase | AC-001, AC-007 |
| FR-002 | Functional | Must | Any authenticated current member may set `accountType` to either supported value before, after, or without a declaration. | Declaration is not activation or entitlement. | User interview | AC-001, AC-008 |
| FR-003 | Functional | Must | `social.craftsky.business.profile/self` shall be a public singleton whose schema fields are all optional; an independently authored empty declaration is valid and it shall not duplicate display name, bio, avatar, or banner. Craftsky publicly hydrates declaration-backed details only for a current member with type `business`. | Keeps portable details independent and additive. | User interview, Codebase | AC-009 |
| FR-004 | Functional | Must | Business types shall be unordered and distinct, with Craftsky maximum 5 and schema maximum 20; offerings shall be unordered and distinct, with Craftsky maximum 10 and schema maximum 20. Craftsky writes shall accept recognized values only and return selections in canonical catalog order. | Provides bounded deterministic taxonomies. | User interview, Lexicon rules | AC-010 |
| FR-005 | Functional | Must | The declaration may contain `tagline` up to 100 graphemes/1000 bytes, display-only `hoursNote` up to 300/3000, and display-only uninterpreted `serviceArea` up to 200/2000. | Bounds public free text without inferred semantics. | User interview | AC-003, AC-011 |
| FR-006 | Functional | Must | Declaration location shall reference the CID-pinned `community.lexicon.location.address`; Craftsky mutation/response shape contains only country and locality, accepts only assigned ISO 3166-1 alpha-2 codes from the versioned catalog, canonicalizes case-insensitive input to uppercase, and bounds locality to 100 graphemes/1000 bytes. Events have no structured location field. | Reuses the community shape while minimizing hydrated data. | User interview, Lexicon rules, Document review | AC-012, AC-013 |
| FR-007 | Functional | Must | The declaration shall have at most one primary action. Recognized non-email actions use the Q14 common web-destination rule. Email requires lowercase `mailto:` plus one ASCII dot-atom addr-spec with one `@` and the Q14 DNS hostname grammar without a port, at most 320 address bytes, and no whitespace, controls, percent encoding, query, fragment, comma, or semicolon. | Constrains untrusted outbound destinations. | User interview, Security review, Document review | AC-014, AC-015 |
| FR-008 | Functional | Must | Featured products shall preserve authored order. The lexicon minimum is title plus URI with optional image, alt, and price; Craftsky writes additionally require a valid image and apply the common web-destination rule to URI. Title is bounded to 150 graphemes/1500 bytes. | Balances interoperable ingestion with first-party card quality. | Initial request, User interview, Document review | AC-004, AC-015, AC-016 |
| FR-009 | Functional | Must | Product arrays shall have schema maximum 20 and Craftsky code-level maximum 4. Craftsky writes reject exact duplicate product URIs without aggressive normalization. | Establishes deterministic product limits. | User interview, Lexicon rules | AC-017 |
| FR-010 | Functional | Must | Optional product price shall use uppercase currency from the versioned active ISO 4217 List One catalog and a canonical nonnegative decimal: integer `0` or 1-12 digits without a leading zero; optional fraction of 1 through the currency's numeric minor-unit scale ending nonzero; no fraction for zero-scale currencies. Signs, exponents, separators, trailing fractional zeros, `N.A.`-scale/withdrawn/unknown currency, excess precision, and noncanonical zero are invalid. | Makes displayed prices exact, bounded, and reproducible. | User interview, Document review | AC-018 |
| FR-011 | Functional | Must | Each event shall be a public TID-keyed `social.craftsky.business.event` record. Lexicon-required fields are name, `startsAt`, `endsAt`, nonempty roles, and `createdAt`; mode, authored status, and `timeZone` are schema-optional but required on Craftsky writes. | Supports independent event identity and broader ingestion. | User interview, Lexicon rules | AC-005, AC-019 |
| FR-012 | Functional | Must | Event name is bounded to 200 graphemes/2000 bytes; optional public-appearance summary to 1000/10000; optional `venueName` to 200/2000. Omitted `isAllDay` is represented and hydrated as `false`. Roles are a nonempty set with Craftsky maximum 4/schema maximum 10. | Bounds event content. | User interview, Lexicon rules, Document review | AC-020 |
| FR-013 | Functional | Must | Craftsky event writes require end after start, canonical UTC whole-second `Z` timestamps, and an IANA timezone name or `UTC`. All-day writes require local-midnight boundaries in that timezone and an exclusive end. Independently authored equal/reversed ranges remain indexed/raw but are omitted from visitor direct/upcoming reads and appear to the current owner with `invalid-time-range` diagnostics. | Makes temporal behavior reproducible. | User interview, Document review | AC-021 |
| FR-014 | Functional | Must | Craftsky writes and public serving allow event duration of at most 31 days. New Craftsky events may be ongoing but cannot already be ended; existing past events remain editable. Independently authored longer records are indexed and preserved but suppressed from public direct/list reads with owner diagnostics. | Bounds public events without losing source. | User interview | AC-022, AC-023 |
| FR-015 | Functional | Must | Events may contain distinct optional `eventUri` and `registrationUri` using the common web-destination rule: absolute credential-free HTTPS, nonempty host, at most 2048 bytes, query/fragment allowed. Craftsky rejects exact duplicates, while independent duplicates hydrate once. Events may contain an optional image using the existing image contract. No `onlineUri` exists. | Keeps links safe and non-redundant. | User interview, Codebase, Document review | AC-015, AC-024 |
| FR-016 | Functional | Must | Owners shall have authenticated event CRUD through AppView-mediated PDS writes. On item GET, a current owner receives the retained management view plus diagnostics even when public serving is suppressed; other callers receive only publicly eligible events and get `404 event_not_found` for absent, blocked, or suppressed records. Authored statuses are exactly `scheduled`, `cancelled`, and `postponed`; omitted independent status safely means scheduled, while past/completed state is derived from `endsAt`. | Defines lifecycle without destructive completion writes. | User interview, API architecture, Document review | AC-025, AC-026 |
| FR-017 | Functional | Must | Public event serving requires current membership and authoritative `business` type, never declaration presence. Public direct reads retain eligible normal-duration past, cancelled, and postponed events; upcoming excludes ended, cancelled, and postponed events. | Separates classification from details and list semantics. | User interview | AC-002, AC-026, AC-027 |
| FR-018 | Functional | Must | `GET /v1/profiles/{handleOrDid}/events` shall provide events separately from the main profile, ordered by `startsAt ASC, URI ASC`, default limit 10 and maximum 50, using an opaque cursor that freezes `asOf` from the first page. | Provides seek pagination with a consistent time-eligibility cutoff. | User interview, API architecture | AC-028 |
| FR-019 | Functional | Must | `GET /v1/events` shall provide the authenticated current owner's all-events management view, ordered by `startsAt DESC, URI DESC`, default limit 20 and maximum 50, with opaque seek pagination. It includes past, cancelled, postponed, and suppressed records. Every item exposes distinct canonical-order `publicSuppressionReasons` and `upcomingExclusionReasons` arrays using only the closed codes defined in Q13. `POST /v1/events/{did}/{rkey}/reports` and existing record-level moderation support individual events. | Makes suppression understandable and abuse actionable. | User interview, Moderation policy, Document review | AC-023, AC-029 |
| FR-020 | Functional | Must | Every normally visible profile/account/author summary shall include authoritative `regular` or `business`, including business accounts without declarations. List hydration shall use a set-based join or one bounded batch lookup whose query count is independent of result count. Blocked shells shall omit `accountType` and all business data; blocked event lists are empty and blocked direct event reads return `404 event_not_found`. | Ensures consistent labels without leaks or N+1 reads. | User interview, Codebase, Document review | AC-030, AC-031 |
| FR-021 | Functional | Must | Declaration and event indexers shall process create/update/replay/delete independently of membership, account type, and declaration while preserving raw source and unknown top-level fields. Rows and deletion tombstones store Tap repository revision TID; only a strictly newer revision for a URI may mutate projection, equal/older operations are no-ops, and duplicate URI/CID/revision delivery is idempotent. Eligibility is evaluated at read time. | Handles out-of-order federation safely. | User interview, Codebase, Document review | AC-032, AC-033 |
| FR-022 | Functional | Must | The persisted account-type row and its current `regular` or `business` value shall survive ordinary membership departure and resume on rejoin. Permanent deletion shall remove owned event records, declaration, private account type, then membership. | Distinguishes departure from permanent deletion. | User interview, Codebase | AC-034, AC-035 |
| FR-023 | Functional | Must | Account type mutation shall be `PUT /v1/profiles/me/account-type` with body `{accountType}`. Declaration mutation shall be `PUT` and `DELETE /v1/profiles/me/business`; PUT full-replaces known fields while merging unknown top-level source extensions. | Fixes owner API contracts without erasing extensions. | User interview, API architecture | AC-036 |
| FR-024 | Functional | Must | Event API shall be `POST /v1/events` plus `GET`, `PUT`, and `DELETE` on `/v1/events/{did}/{rkey}`. Declaration creation requires `If-Match: *` and succeeds only when the singleton is absent; declaration replacement and every business/event PUT or DELETE of an existing record require its expected canonical CID in `If-Match`. Failed absence/version preconditions return `409 pds_record_conflict`, and successful record responses expose CID. | Prevents lost PDS updates while allowing conditional singleton creation. | User interview, API architecture, implementation clarification 2026-08-29 | AC-037 |
| FR-025 | Functional | Must | On Craftsky event creation, AppView shall reject client-supplied `createdAt` and server-stamp the required field. Event update bodies must omit `createdAt`; AppView rejects any supplied value, including the stored value, and preserves the stored field. | Matches existing mediated-write behavior for events. | User interview, Codebase, Document review | AC-038 |
| FR-026 | Functional | Must | Craftsky write catalogs are closed to recognized values, while ingestion preserves unknown lexicon-valid taxonomy/action/role/mode/status values and APIs represent them safely. Unknown or unsafe independent subordinate fields are omitted when necessary without suppressing the containing record. | Supports federation without weakening first-party validation. | User interview, Lexicon rules | AC-033, AC-039 |
| RULE-001 | Business rule | Must | Public business classification requires current membership plus authoritative type `business`; declaration presence and contents do not activate or deactivate classification. | Keeps the scalar authoritative. | User interview | AC-001, AC-002 |
| RULE-002 | Business rule | Must | Any authenticated current member, including `regular`, may mutate their own declaration/events; no caller may mutate another DID's records. | Allows public setup without cross-owner writes. | User interview, API architecture | AC-008, AC-025 |
| RULE-003 | Business rule | Must | Event serving requires current membership plus type `business`, not a declaration. Deleting a declaration removes only declaration-backed details and leaves classification/event eligibility unchanged. | Prevents declaration from becoming entitlement state. | User interview | AC-002, AC-027 |
| RULE-004 | Business rule | Must | Product prices are seller-authored amount/currency display data. AppView adds no disclaimer field and no inventory, availability, synchronization, shipping, tax, checkout, or accuracy field or claim. | External prices can become stale; future Flutter presentation is separate. | User interview, Document review | AC-040 |
| RULE-005 | Business rule | Must | Independent address fields beyond country/locality are preserved raw and omitted hydrated. Oversized locality is omitted while valid assigned country remains. Non-alpha-2 or unassigned country omits the whole location projection while the containing declaration remains served if otherwise eligible. | Safely projects broad external records. | User interview, Lexicon rules, Document review | AC-013 |
| RULE-006 | Business rule | Must | Permanent deletion order is owned events, declaration, private account type, then membership; ordinary departure deletes none of the first three. | Preserves explicit deletion guarantees. | User interview, Codebase | AC-034, AC-035 |
| RULE-007 | Business rule | Must | Recognized business types, in canonical order, are `dyer`, `fiber-producer`, `fiber-processor`, `yarn-shop`, `fabric-shop`, `craft-supply-shop`, `pattern-designer`, `finished-goods-maker`, `tool-maker`, `teacher`, `craft-studio`, `repair-service`, `technical-editor`, `photographer`, `publisher`, `other-craft-business`. | Defines the exact first-party catalog. | User interview | AC-010 |
| RULE-008 | Business rule | Must | Recognized offerings, in canonical order, are `yarn`, `fiber`, `fabric`, `patterns`, `kits`, `notions`, `tools`, `finished-goods`, `custom-work`, `repairs`, `classes`, `studio-hire`, `wholesale`, `digital-products`, `technical-editing`, `photography-services`, `fiber-processing`. | Defines the exact first-party catalog. | User interview | AC-010 |
| RULE-009 | Business rule | Must | Recognized actions are exactly `shop`, `browse-patterns`, `request-custom-order`, `book-class`, `book-appointment`, `view-event-calendar`, `email`, `visit-website`, `wholesale-enquiries`. | Defines the exact first-party CTA catalog. | User interview | AC-014 |
| RULE-010 | Business rule | Must | Taxonomies and business state are display-only and shall not affect feed order, search/filter/ranking, moderation priority, reach, or permissions. | Preserves product principles. | User interview, Product vision | AC-006 |
| RULE-011 | Business rule | Must | Recognized event roles are exactly `organizer`, `instructor`, `vendor`, `exhibitor`, `speaker`, `demonstrator`; modes are exactly `in-person`, `online`, `hybrid` and impose no field dependencies; statuses are exactly `scheduled`, `cancelled`, `postponed`. | Defines uniform first-party catalogs and descriptive modes. | User interview | AC-019, AC-020 |
| NFR-001 | Non-functional | Must | Lexicons shall use the confirmed bounds and optionality, valid record keys, and ADR-approved evolution strategy. The complete address record shall be vendored at the exact CID-named Q17 path; a test shall recompute its atproto DAG-CBOR CID and compare it to Q17; `just lexgen` shall resolve the NSID only from that local file without network access and generated types shall be reproducible. Country/currency source snapshots, retrieval metadata, SHA-256 sidecars, and generated Go catalogs shall also be drift-tested. | Public schemas and validation catalogs are load-bearing. | Lexicon rules, Document review | AC-041 |
| NFR-002 | Non-functional | Must | Projection shall be idempotent and preserve raw public source sufficient for unknown valid values, omitted hydration, extensions, replay, and later re-projection. | Protects federated source fidelity. | Codebase, Lexicon rules | AC-032, AC-033 |
| NFR-003 | Non-functional | Must | `/v1/` JSON shall use camelCase, standard error envelopes and request IDs, existing authenticated-device policy, and opaque event cursors. | Maintains API architecture. | API architecture | AC-028, AC-037, AC-042 |
| NFR-004 | Non-functional | Must | Existing block and moderation policy shall cover business details, products, summaries, and events; blocked shells expose neither account type nor business data. | Prevents alternate-shape leakage. | Codebase, Moderation policy | AC-029, AC-031 |
| NFR-005 | Non-functional | Must | Product/event images shall accept JPEG, PNG, or WebP up to 15 MiB, optional aspect ratio, and optional alt text up to 1000 graphemes/1000 bytes. | Reuses the established image contract. | Codebase | AC-016, AC-024 |
| NFR-006 | Non-functional | Must | Logs, metrics, and traces shall not record full URIs, email destinations, free-text notes, titles, prices, or locations as telemetry attributes. | Public data remains sensitive and high-cardinality. | Security review | AC-043 |
| NFR-007 | Non-functional | Must | AppView validation and hydration shall never fetch authored product, action, event, registration, or email destinations. | Prevents SSRF and privacy leaks. | Security review | AC-015 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002, RULE-001 | Given any current member with or without a declaration, setting `business` succeeds and every normally visible profile reports `business`; setting `regular` restores ordinary classification without PDS mutation. |
| AC-002 | BR-001, FR-017, RULE-001, RULE-003 | Given a current business member with events, deleting or never creating a declaration does not change `business` classification or event eligibility; changing type to `regular` suppresses public events but not source records. |
| AC-003 | BR-002, FR-005 | Given bounded authored details, eligible profile hydration accurately returns tagline, hours note, service area, and general location without interpreting the free text. |
| AC-004 | BR-003, FR-008 | Given valid ordered product cards, eligible hydration preserves order and returns title, URI, required first-party image, and optional alt/price when supported. |
| AC-005 | BR-003, FR-011 | Given a valid event, it receives an independently addressable TID-keyed PDS record and can be read independently. |
| AC-006 | BR-004, RULE-010 | Business type, taxonomies, products, and events provide no paid, verified, search, filter, ranking, reach, moderation-priority, or permission effect. |
| AC-007 | FR-001 | Missing account-type state reads as `regular`; values other than `regular` and `business`, including `pro`, are rejected and absent from storage/API contracts. |
| AC-008 | FR-002, RULE-002 | An authenticated current regular member can create/edit own declaration/events and set account type; unauthenticated, non-current, or different-DID mutation is rejected. |
| AC-009 | FR-003 | An independently authored empty declaration is accepted; declaration schema omits ordinary identity fields; Craftsky hydrates its details only for current business accounts; deleting it leaves membership and account type intact. |
| AC-010 | FR-004, RULE-007, RULE-008 | Craftsky accepts only exact recognized type/offering values, rejects duplicates and counts above 5/10, permits schema counts through 20 for independent records, and returns recognized selections in the listed canonical order. |
| AC-011 | FR-005 | Inputs exceeding any tagline, hours-note, or service-area grapheme or byte bound are rejected with field-specific validation. |
| AC-012 | FR-006 | Lower/mixed-case assigned alpha-2 country input is stored/returned uppercase; unassigned/non-alpha-2 first-party country or oversized locality is rejected; only country/locality appear in declaration mutation and response shapes, and events expose no structured location. |
| AC-013 | FR-006, RULE-005 | An independent declaration location with valid country plus extra fields and oversized locality keeps raw source, hydrates country only, and omits extras; non-alpha-2 or unassigned country omits location while the containing declaration remains otherwise servable. |
| AC-014 | FR-007, RULE-009 | Craftsky accepts zero or one action with an exact recognized type and compliant destination, and rejects unknown first-party action types or a second action. |
| AC-015 | FR-007, FR-008, FR-015, NFR-007 | Every first-party web destination satisfies the exact Q14 parser/DNS/port grammar and 2048-byte bound; query/fragment are allowed. Email accepts only lowercase `mailto:` plus one ASCII dot-atom addr-spec with the same no-port DNS grammar up to 320 bytes. HTTP/custom schemes, userinfo, Unicode hosts, IP literals, single-label/trailing-dot/invalid-label hosts, invalid ports, percent-encoded authority/email, whitespace, controls, query/fragment/list email forms fail. No destination is fetched or resolved. |
| AC-016 | FR-008, NFR-005 | Craftsky rejects a product without image, with invalid title/image, or with a product URI that violates the common web-destination rule; an independent schema-valid title+URI card remains indexed, and valid optional image alt round-trips within 1000/1000. |
| AC-017 | FR-009 | Craftsky accepts four ordered products, rejects five, and rejects exact duplicate URI strings while not collapsing distinct URI strings through extra normalization; schema permits up to 20. |
| AC-018 | FR-010 | The checked-in SIX List One snapshot/digest generates the active numeric-scale catalog. Price accepts exact `0` and canonical variable-scale values such as USD `1`, `1.2`, `1.23`, rejects USD `1.20`, zero-scale fractions, leading-zero integers, signs, exponents, separators, lowercase/unknown/withdrawn/`N.A.`-scale currency, over 12 integer digits, and excess precision. Unsupported independent price is omitted while its product card remains. |
| AC-019 | FR-011, RULE-011 | Lexicon accepts required minimum fields with optional mode/status/timezone, while Craftsky writes require exact recognized mode/status, IANA timezone or UTC, and at least one recognized role. Omitted independent status behaves as scheduled; omitted mode/timezone is safely unspecified. |
| AC-020 | FR-012, RULE-011 | Event name/summary/venue bounds are enforced; omitted `isAllDay` hydrates as false; roles reject empty, duplicates, unknown first-party values, and more than four while schema permits ten; modes impose no location/link dependencies. |
| AC-021 | FR-013 | Craftsky rejects end at/before start, non-whole-second/non-UTC writes, invalid timezone, or all-day boundaries not at local midnight with exclusive end; valid DST-crossing instants retain correct ordering. Independent equal/reversed ranges remain raw-indexed, are absent from visitor direct/upcoming reads, and appear to the current owner with `invalid-time-range`. |
| AC-022 | FR-014 | Craftsky rejects duration over 31 days and a newly created already-ended event, accepts an ongoing event, and permits edits to an existing past event. |
| AC-023 | FR-014, FR-019 | An independent event over 31 days remains indexed/raw but is absent from public direct/list reads and appears to its owner with a bounded suppression reason. |
| AC-024 | FR-015, NFR-005 | Craftsky accepts distinct optional event/registration URIs that satisfy the common credential-free HTTPS/host/2048-byte rule and a valid optional event image, rejects exact duplicate links and invalid image/link data, exposes no `onlineUri`, and hydrates duplicate independent links once. |
| AC-025 | FR-016, RULE-002 | An authenticated current owner can create/read/update/delete an event through AppView/PDS convergence; another actor cannot mutate it. The current owner may directly read their retained suppressed event with management diagnostics; other callers receive `404 event_not_found` for absent, blocked, or publicly suppressed events. |
| AC-026 | FR-016, FR-017 | Scheduled future/ongoing events appear upcoming; cancelled, postponed, or ended events do not, but remain directly readable when owner membership/type and duration policy permit; derived past/completed is based on `endsAt`. |
| AC-027 | FR-017, RULE-003 | A current business member's eligible events are publicly served without a declaration and remain served after declaration deletion; a regular/departed owner's events are suppressed. |
| AC-028 | FR-018, NFR-003 | Profile events use the dedicated route, omit events from the main profile payload, order by start then URI, and default to 10/cap at 50. For an unchanged record set, opaque seek-cursor traversal has no duplicates or omissions; first-page `asOf` freezes time-based eligibility, while concurrent record mutations retain normal seek-pagination semantics. |
| AC-029 | FR-019, NFR-004 | `GET /v1/events` defaults to 20/caps at 50, orders by `startsAt DESC, URI DESC`, and returns every owner lifecycle/suppression state with non-null distinct canonical-order reason arrays using only Q13 codes. `POST /v1/events/{did}/{rkey}/reports` reports an individual event and moderation can suppress it under existing record policy. |
| AC-030 | FR-020 | Every normally visible profile/search/relationship/embedded-author summary includes authoritative `regular` or `business`, including `business` without declaration. Each list uses a set-based join or one bounded batch lookup, with account-type query count independent of row count. |
| AC-031 | FR-020, NFR-004 | A blocked profile shell and every blocked summary omit `accountType`, declaration details, location, action, and products; blocked event lists are empty and blocked direct event reads return `404 event_not_found`. |
| AC-032 | FR-021, NFR-002 | Duplicate delivery for one URI/CID/revision is idempotent; a strictly newer create/update/delete revision wins; equal or older operations are ignored; delete tombstones prevent stale resurrection; and records arriving before membership/type/declaration are retained without dependency retries. |
| AC-033 | FR-021, FR-026, NFR-002 | Unknown lexicon-valid values/extensions and unsafe independent subordinate fields survive raw projection; APIs safely represent unknown values and omit only unsafe subordinate data rather than the containing record. |
| AC-034 | FR-022, RULE-006 | Ordinary membership departure suppresses serving but retains the account-type row with its current `regular` or `business` value, declaration, and events; rejoin restores that persisted value and recalculated eligibility. |
| AC-035 | FR-022, RULE-006 | Approved permanent deletion observably orders owned events → declaration → private account type → membership, including retries and already-absent membership, with no reusable generic PDS-delete behavior. |
| AC-036 | FR-023 | Exact account/declaration routes and body are implemented; declaration PUT full-replaces known fields, merges unknown top-level source extensions, and DELETE removes only the declaration. |
| AC-037 | FR-024, NFR-003 | Exact event routes are implemented; stale/missing `If-Match` on required PUT/DELETE yields standard `409 pds_record_conflict`, while successful record responses expose CID in camelCase contracts. |
| AC-038 | FR-025 | Event creation rejects client `createdAt` and server-stamps the required field; update requires the field to be absent, rejects even the unchanged stored value when supplied, and preserves the stored value. |
| AC-039 | FR-026 | Craftsky rejects unknown taxonomy/action/role/mode/status writes, while valid independent unknown values do not invalidate the record and are safely represented. |
| AC-040 | RULE-004 | Product/profile records and AppView responses expose only seller-authored amount/currency and contain no disclaimer, inventory, availability, synchronization, shipping, tax, checkout, or accuracy field or claim. |
| AC-041 | NFR-001 | ADR/schema review confirms all exact bounds, optionality, keys, catalogs, and the Q17 address AT-URI/CID/path. Tests recompute the vendored record's atproto DAG-CBOR CID, verify country/currency snapshot SHA-256 metadata and generated catalogs, run lexgen offline from the local external mapping, and repeated generation produces stable output. |
| AC-042 | NFR-003 | Invalid input/cursor returns the standard camelCase error envelope with request ID and appropriate 4xx status under existing device authentication policy. |
| AC-043 | NFR-006 | Request/projector observability records only bounded operation/result metadata and none of the prohibited authored values. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Declaration/event arrives before membership or account-type state | Index and preserve immediately; evaluate membership/type only when serving. | FR-021 |
| EC-002 | Business account has no declaration or deletes it | Keep `business` in visible summaries and continue serving otherwise eligible events; omit declaration-backed details. | FR-017, FR-020, RULE-003 |
| EC-003 | Member leaves and later rejoins | Suppress while absent; retain the account-type row's current value and public source; restore classification/eligibility from that value on rejoin. | FR-022 |
| EC-004 | Fifth product or exact duplicate URI | Reject Craftsky write; do not normalize merely similar URIs into duplicates. | FR-009 |
| EC-005 | Product price becomes stale or independently uses unsupported currency | Keep supported seller-authored price without guarantees; omit unsupported price but retain card/source. | FR-010, RULE-004 |
| EC-006 | Event is postponed without replacement dates | Keep last valid required times and authored `postponed`; exclude upcoming but permit direct read. | FR-016, FR-017 |
| EC-007 | All-day event crosses DST | Validate each boundary as local midnight in the named timezone and compare canonical instants; end remains exclusive. | FR-013 |
| EC-008 | Independent unsafe action/product/event destination | Preserve raw destination, omit it hydrated, retain containing declaration/product/event, and never fetch or resolve it. | FR-026, NFR-007 |
| EC-009 | Unknown valid taxonomy/role/mode/status | Preserve raw, represent safely, and do not infer search, permission, or field dependencies. | FR-026, RULE-010 |
| EC-010 | Address has valid country, oversized locality, and street | Hydrate uppercase country only; retain locality/street raw. | RULE-005 |
| EC-011 | Address country is non-alpha-2 or unassigned | Omit the entire location projection but continue serving the eligible containing declaration. | RULE-005 |
| EC-012 | Events have equal startsAt but different URIs | Order by URI ascending after startsAt. With an unchanged record set, seek traversal has no duplicates/omissions; `asOf` freezes time eligibility but does not snapshot concurrent edits. | FR-018 |
| EC-013 | New event started but has not ended | Accept it when all other rules pass and include it in upcoming; reject if already ended. | FR-014, FR-017 |
| EC-014 | Independent event exceeds 31 days | Retain/index, suppress from public direct/list, expose bounded owner reason, and allow moderation/reporting paths as policy permits. | FR-014, FR-019 |
| EC-015 | Independent event omits status/mode/timezone | Treat status as scheduled and mode/timezone as safely unspecified; apply timing and eligibility without inventing source fields. | FR-011, FR-016 |
| EC-016 | Independent event repeats event/registration URI | Hydrate the destination once without rewriting raw source. | FR-015 |
| EC-017 | Permanent deletion starts with retained records while membership is absent | Delete registered event/declaration records and private account type before final membership cleanup. | FR-022, RULE-006 |
| EC-018 | Stale update arrives after a newer update or delete | Compare Tap repository revision; ignore equal/older operation and retain the newer row or deletion tombstone. | FR-021, NFR-002 |
| EC-019 | Independent event has equal or reversed start/end | Retain raw source, suppress visitor direct/upcoming reads, and expose `invalid-time-range` to the current owner. | FR-013, FR-019 |

## 15. Data / Persistence Impact

- New public records:
  - Singleton `social.craftsky.business.profile/self`, valid even when empty.
  - TID-keyed `social.craftsky.business.event/{tid}`.
- Existing `social.craftsky.actor.profile/self` remains unchanged; account type is never published to the PDS.
- New private authoritative AppView scalar per DID: exactly `regular` or `business`, defaulting absent state to `regular`; it is not tied by destructive membership foreign-key cascade.
- Reused external definition: vendored offline from `at://did:plc:mtr7qrqtcyseedx3jyr5o7db/com.atproto.lexicon.schema/community.lexicon.location.address` at CID `bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq`; raw independent fields remain stored, hydrated declaration shape is country/locality only.
- Likely shared definitions: money, primary action, product card, image reuse, and open known-value fields. There is no service-reach object, secondary link, custom taxonomy, `onlineUri`, or public account-type definition.
- Projection stores one declaration row per DID, zero-to-many event rows, CID/URI/raw source/source repository revision, deletion revision tombstones, and materialized query columns. It must accept records before membership/type/declaration dependencies and ignore stale/equal revisions per URI.
- Product write maximum is a code constant of four, not configuration.
- Migration and indexes are required; exact migration number is deferred to coding design.
- Unknown top-level declaration extensions survive Craftsky PUT merges; unknown and unsafe independent fields survive raw projection.
- Ordinary departure retains account type and PDS records. Permanent deletion order is event records, declaration, account type, membership.

## 16. AppView / API / Client Impact

- Flutter: no changes in this slice.
- Profiles and summaries:
  - Normally visible shapes always include `accountType: regular|business`.
  - Business hydration is declaration-backed but classification is not.
  - Main profile does not embed events.
  - Blocked shells omit account type and all business data.
- Exact owner APIs:
  - `PUT /v1/profiles/me/account-type` with camelCase `{accountType}`.
  - `PUT|DELETE /v1/profiles/me/business`.
  - `POST /v1/events`.
  - `GET|PUT|DELETE /v1/events/{did}/{rkey}`.
  - `GET /v1/profiles/{handleOrDid}/events` for dedicated paginated public events.
  - `GET /v1/events?limit&cursor` is the owner all-events management route: default 20, max 50, `startsAt DESC, URI DESC`, `{items,cursor?}`, and exact diagnostic arrays/codes from Q13.
  - `POST /v1/events/{did}/{rkey}/reports` extends the existing reporting/moderation surface to individual events.
- Business/event PUT and DELETE require expected CID via `If-Match`; conflicts return `409 pds_record_conflict`; record responses expose CID.
- Declaration PUT full-replaces known fields while preserving unknown top-level source extensions.
- All routes maintain existing authentication/device policy, camelCase JSON, and standard errors.
- No scheduler is required. Upcoming, past/completed, `asOf`, and eligibility are computed at read time.

## 17. Security / Privacy / Permissions

- All `/v1/` surfaces remain authenticated under existing device policy.
- Any current member may manage only their own account type/declaration/events. Account type is classification, not permission or entitlement.
- Every lexicon field is public. Private authoritative account type is exposed only through normally visible API classification, never on the PDS or blocked shells.
- Craftsky accepts only country/locality and safe recognized outbound destinations. Country uses the versioned assigned ISO alpha-2 catalog. Independent extra/unsafe values remain raw and are omitted hydrated.
- Every web destination uses absolute credential-free HTTPS with nonempty host and a 2048-byte maximum; query/fragment are allowed. Email uses the exact Q14 ASCII mailto grammar. AppView never resolves or fetches an authored destination.
- Product/event images reuse validated MIME, size, aspect-ratio, and alt bounds.
- Individual events can be reported and moderated. Business data does not bypass profile blocks, account takedowns, or record moderation.
- Abuse cases include misleading business claims, phishing, impersonation, spam events, stale prices, and location/email oversharing; business labels never imply verification.

## 18. Observability

- Logs record bounded operation, collection, validation class, eligibility result, suppression reason code, and outcome without authored contents.
- Metrics count projection/API outcomes, moderation/reporting outcomes, and latency with bounded labels.
- Traces omit full external destinations, email, text, titles, prices, and locations.
- Existing ingestion freshness/failure alerting covers both new collections; repeated projector failures and unusual suppression rates should be visible.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Durable lexicon fields or bounds prove unsuitable. | Versioned NSID and migration may be needed. | ADR, exact bounds/optionality review, generated-schema tests, and conservative schema maxima. |
| RISK-002 | Public location/email surprises an owner. | Privacy or safety harm. | Minimal hydration, API documentation, explicit deletion, and future Flutter publication warnings. |
| RISK-003 | External price becomes stale. | Visitor confusion. | Seller-authored display framing and no synchronization/availability claim. |
| RISK-004 | Malicious outbound links are independently published. | Phishing or abuse. | Strict first-party validation, safe omission during hydration, no-fetch behavior, reporting/moderation. |
| RISK-005 | Business state resembles verification. | False trust. | Self-declared semantics and no verified/subscribed/endorsed language or behavior. |
| RISK-006 | Membership, account type, declaration, and event delivery occur in any order. | Data loss or early/late serving. | Independent raw indexing and centralized read-time eligibility. |
| RISK-007 | Business data leaks through embedded identity shapes. | Block/privacy regression. | Central account-type hydration/redaction and contract tests for every summary shape. |
| RISK-008 | Broad taxonomies are exclusionary. | Poor presentation. | Exact catalog includes `other-craft-business`; explanation uses tagline/bio; schema-valid unknown values remain preserved. |
| RISK-009 | AppView account-type state is missing/stale. | Incorrect labels or event serving. | Default missing to regular, centralized mutation/read policy, departure/rejoin tests, permanent deletion ordering. |
| RISK-010 | External address schema changes or carries more detail. | Lexgen drift or privacy exposure. | Exact reproducible pin, raw preservation, country/locality allowlist, ADR dependency analysis. |
| RISK-011 | Seek pagination and mutable event records interact. | Concurrent inserts, deletes, or ordering-key edits may cause normal seek-pagination duplicates or omissions even though time eligibility is frozen. | Freeze first-page `asOf` only for time eligibility; order and seek by `(startsAt, URI)`; guarantee complete traversal only for an unchanged record set; test and document normal concurrent-mutation semantics. |
| RISK-012 | Currency rules drift. | Rejected legitimate prices or inconsistent displays. | Versioned known ISO 4217/minor-unit table, canonical validation tests, omit unsupported independent prices only. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | One exact optional nonnegative product amount and currency is sufficient. | Ranges, sales, or “from” pricing require additive/versioned design. |
| ASM-002 | Free-text `serviceArea` is sufficient and should never drive filtering. | Structured service discovery needs a separate future design. |
| ASM-003 | One optional locality/country location is sufficient. | Multiple locations need first-class records later. |
| ASM-004 | Event owner and represented business are the same DID. | Delegation needs represented-business references and authorization. |
| ASM-005 | Read-time event eligibility and derived temporal state are operationally sufficient. | High volume may later need materialization without changing authored semantics. |
| ASM-006 | Existing image constraints suit products and events. | Different image needs require a separate contract. |
| ASM-007 | AppView-owned account type need not be portable to another AppView. | Portability would require a separate protocol convention, not declaration activation. |
| ASM-008 | The pinned community address dependency remains acceptable. | Craftsky would need a restricted owned definition and migration. |
| ASM-009 | The implementation has or can maintain a known ISO 4217 minor-unit catalog. | Price validation cannot meet the confirmed canonical contract. |

## 21. Open Questions

- None blocking requirements design.
- Non-blocking ADR/coding detail: choose exact schema object layouts while preserving every confirmed field, bound, key, optionality decision, and the Q17 external-reference pin. The ADR must record retrieval/CID verification before lexicon implementation.
- Owner all-events route, pagination, ordering, diagnostics arrays/codes, direct-read policy, and report route are resolved in Q13 and Q16.
- Non-blocking product-design detail: choose optional image aspect ratios and visual treatments; this must not change the established MIME/size/alt contract.

## 22. Review Status

Status: Approved
Risk level: High
Review: Completed in `03-document-review.md`
Approver: User
Date: 2026-08-28
Notes: Approved after revision resolved owner-management, pinning, destination, money, direct-read, temporal, projection-order, and test-traceability findings. Risk remains high because this publishes durable lexicons, pins an externally controlled definition, introduces private authoritative account-type state, adds independently indexed collections, changes summary/redaction contracts, and adds event reporting, moderation, pagination, CID conflict, and deletion-order behavior.

## 23. Approved Handoff

- Requirements file: `docs/changes/2026-08-27-business-profiles/01-requirements.md`
- Status: Approved for coding planning and TDD implementation on 2026-08-28.
- Test specification: `docs/changes/2026-08-27-business-profiles/02-acceptance-tests.md`; approved with these resolved contracts.
- Test design must trace every Must row to its listed AC and each AC back to every listed requirement.
- Required coverage includes scalar account type without declaration, regular-member setup, departure/rejoin/permanent deletion, blocked-shell omission, exact catalogs/bounds, raw preservation/safe hydration, location pin/projection, URI/email safety, products/money/images, temporal/all-day/duration rules, direct/upcoming/owner event views, seek pagination with frozen time eligibility, event reporting/moderation, exact routes, CID conflicts, server-stamped immutable event `createdAt`, and no Flutter changes.
