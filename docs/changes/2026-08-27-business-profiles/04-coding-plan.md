# Coding Plan: Business Profiles

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` is approved. The revised requirements and tests resolve the prior contract gaps, and the user explicitly approved the full revised contract on 2026-08-28.
- Approval status: Approved for TDD implementation on 2026-08-28.
- Architecture references: `atproto-craft-social-app-reference.md`, `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`, `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`, and `docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md`.
- Scope boundary: Go AppView, durable lexicons, ADR, generated code, persistence, Tap projection, API, moderation, and permanent deletion only. Flutter remains excluded by NG-011.

## 2. Implementation Strategy

Implement the feature as four deliberately separate authorities that converge through existing boundaries:

1. `internal/business` owns closed first-party catalogs, validation, safe independent-record hydration, account-type defaults, event eligibility, diagnostics, cursors' domain tuples, and the Postgres store.
2. PDS records remain the public source of truth for `social.craftsky.business.profile/self` and TID-keyed `social.craftsky.business.event` records. AppView mutation handlers use the existing lifecycle-fenced `pdseffects.EffectExecutor`; account type never invokes the PDS.
3. Transactional indexers preserve complete raw records and apply only strictly newer Tap repository revisions. Business projection bypasses the ordinary active-membership prerequisite but retains repository-generation and terminal-owner fences. Public visibility is computed at read time from current membership, authoritative account type, block state, moderation, and event policy.
4. API handlers expose the approved `/v1/` routes. A single batch account-type response hydrator decorates visible identity objects across existing response shapes without N+1 queries; the main profile handler separately attaches declaration-backed business details. Blocked shapes are excluded from both paths.

Use two migrations so the first TDD slice can add only account-type persistence before record projection and moderation schema are introduced. Use hand-written pgx stores, matching the current repository; `appview/queries/` contains no sqlc queries.

Before any lexicon file is created, add ADR 010 and verify the address record and catalog provenance described by Q14, Q15, and Q17. Durable schema work then uses one shared non-record definitions lexicon plus the two public record lexicons. `just lexgen` and catalog generation remain offline and deterministic.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Domain policy | Feature-local Go packages with table-driven validators | Add `internal/business` for account types, catalogs, validation, hydration, eligibility, diagnostics, and store contracts | FR-001–FR-020, FR-023–FR-026, RULE-001–RULE-011 | UT-001–UT-023 |
| Private classification | AppView-private pgx tables guarded by current-member middleware | Add durable `craftsky_account_types`; absent row reads `regular`; no membership FK/cascade | FR-001, FR-002, FR-022 | AT-001, IT-001, IT-002, IT-014, IT-015 |
| Public schemas | JSON lexicons plus generated Go/CBOR types | Add shared business defs, singleton declaration, TID event, and exact external address pin | FR-003–FR-015, NFR-001, NFR-005 | AT-013, IT-016, REG-001 |
| Catalog provenance | Checked-in generated language data and generation drift gates | Add pinned ISO 3166-1/4217 source snapshots, metadata, SHA-256 sidecars, generator, and generated Go catalogs | FR-006, FR-010, NFR-001 | UT-006, UT-012, IT-016, MAN-001 |
| Projection storage | Transactional pgx projectors with durable Tap source rows | Add raw declaration/event projections plus revision-bearing delete tombstones and query indexes | FR-021, FR-026, NFR-002 | AT-009, IT-004–IT-006 |
| Projection lifecycle | `TransactionalDispatcher` enforces owner lifecycle before projectors | Add an explicit collection policy allowing business source preservation before/after membership while retaining generation and terminal fences | FR-021, FR-022 | IT-005, IT-006, IT-014 |
| Tap wiring | Explicit dispatcher registrations and Compose collection filters | Register both NSIDs and forward them from Tap | FR-021, NFR-002 | IT-006 |
| PDS writes | Strict request DTOs and lifecycle-fenced `EffectExecutor` | Add declaration/event create, replace, and delete flows with TID keys, CID preconditions, declaration extension merge, and immutable event `createdAt` | FR-023–FR-025, RULE-002 | AT-004, AT-005, AT-011, IT-002, IT-003, IT-007, IT-012 |
| Profile reads | `ProfileResponse`, profile store, and global identity response hydration | Add business declaration view to full profiles and one bounded account-type batch query to all visible identity shapes | FR-003, FR-020, NFR-004 | AT-003, AT-008, IT-004, IT-011, REG-002, REG-003 |
| Event reads | Opaque seek cursors and feature-local stores/handlers | Add visitor direct read, frozen-`asOf` upcoming list, and owner management list | FR-016–FR-019 | AT-005–AT-007, IT-008–IT-010 |
| Reports/moderation | Record reports and closed post/account subject constraints | Add event record subjects and use event eligibility/moderation in serving and diagnostics | FR-019, NFR-004 | AT-007, IT-010 |
| Permanent deletion | Closed collection registry, PDS deleter, private cleaner, terminal DID inventory | Stage events, declaration, account type, then membership; preserve ordinary departure behavior | FR-022, RULE-006 | AT-010, IT-014, IT-015, REG-007 |
| Routes/DI | Capability route bundles, route policy catalogue, `contentDependencies`, adapters | Add a business/event bundle and inject store, clock, resolver, effects, report services, and hydrator | FR-018, FR-019, FR-023, FR-024, NFR-003 | AT-006, AT-007, AT-011, IT-009, IT-010, IT-012 |
| Observability/security | Bounded operation/result attributes and no authored content | Add business operation/reason constants; validation and hydration remain pure and never resolve/fetch authored destinations | NFR-006, NFR-007 | AT-014, IT-013, IT-021, MAN-002 |
| Product neutrality | Chronological feed/search and existing authorization policy | Do not add business inputs to ranking, search, permissions, or moderation priority | BR-004, RULE-010 | AT-012, IT-017–IT-019, REG-005, REG-006 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `adr/010-business-profile-event-lexicons-and-pinned-data.md` | Create | Record durable schema shapes, shared defs, external CID verification, source provenance, maxima/evolution, and unchanged actor profile | NFR-001 | AT-013, IT-016, MAN-001 |
| `lexicon/social/craftsky/business/defs.json` | Create | Shared image/aspect-ratio, price, action, and product definitions; open `knownValues` where required | FR-004, FR-007–FR-010, NFR-005 | UT-002–UT-013, IT-016 |
| `lexicon/social/craftsky/business/profile.json` | Create | Optional singleton declaration referencing shared defs and pinned community address | FR-003–FR-010 | AT-003, IT-003, IT-004, IT-016 |
| `lexicon/social/craftsky/business/event.json` | Create | TID-keyed event with exact required/optional fields and shared image ref | FR-011–FR-015 | AT-005, IT-005, IT-016 |
| `lexicon/social/craftsky/actor/profile.json` | No change | Protected membership/identity ownership contract | FR-003, NFR-001 | REG-001 |
| `lexicon/README.md` | Change | Document the new NSIDs and exact external dependency pin | NFR-001 | MAN-001 |
| `appview/cmd/lexgen/external/community.lexicon.location.address.<CID>.json` | Create | Complete Q17 PDS record value, including `$type`, used only from disk | FR-006, NFR-001 | AT-013, IT-016 |
| `appview/cmd/lexgen/build.json`, `appview/cmd/lexgen/cborgen/main.go` | Change | Map/generate local `community` types and Craftsky business JSON/CBOR types offline | NFR-001 | AT-013, IT-016 |
| `appview/internal/lexicon/community/*.go` | Create/generated | Local generated type and CBOR methods for the pinned address definition | FR-006, NFR-001 | IT-016 |
| `appview/internal/lexicon/craftsky/business*.go`, `cbor_gen.go` | Create/change/generated | Generated shared defs, profile, event, and CBOR methods | NFR-001 | IT-016 |
| `appview/internal/lexicon/craftsky/business_contract_test.go` | Create | Schema, CID, generated-type, maxima, forbidden-field, and actor-profile regression contract | FR-003–FR-015, NFR-001, NFR-005 | AT-013, IT-016, REG-001 |
| `appview/internal/business/catalogdata/iso-3166-1-obp-2026-08-28.{html,metadata.json,sha256}` | Create | Native checked-in assigned-country snapshot and provenance | FR-006, NFR-001 | UT-006, IT-016 |
| `appview/internal/business/catalogdata/iso-4217-list-one-2026-08-28.{xml,metadata.json,sha256}` | Create | Native checked-in SIX List One snapshot and provenance | FR-010, NFR-001 | UT-012, IT-016 |
| `appview/cmd/businesscataloggen/main.go` | Create | Offline parser/digest verifier and deterministic country/currency Go generator with check mode | FR-006, FR-010, NFR-001 | IT-016 |
| `appview/internal/business/countries_generated.go`, `currencies_generated.go` | Create/generated | Runtime assigned-country and active numeric-scale currency catalogs | FR-006, FR-010 | UT-006, UT-012, UT-013 |
| `appview/internal/business/account_type.go`, `catalog.go`, `validation.go`, `destination.go`, `money.go`, `event_time.go`, `eligibility.go`, `models.go` | Create | Domain types, constants, validation, safe hydration, canonical order, and eligibility/diagnostic policy | FR-001–FR-020, FR-025, FR-026 | UT-001–UT-020, UT-023 |
| `appview/internal/business/store.go` | Create | Account-type persistence, raw projection operations, batch identity lookup, profile hydration, event direct/list queries | FR-001, FR-003, FR-016–FR-022 | IT-001, IT-004, IT-008–IT-011, IT-014 |
| `appview/internal/business/*_test.go` | Create | Unit and real-store tests named by the acceptance specification | FR-001–FR-026 | UT-001–UT-020, UT-023, IT-001, IT-004, IT-008, IT-013, IT-014, IT-016, IT-021 |
| `appview/migrations/000061_business_account_types.{up,down}.sql` | Create | Private no-cascade scalar table with exact check constraint | FR-001, FR-022 | IT-001, IT-014, IT-015 |
| `appview/migrations/000062_business_records.{up,down}.sql` | Create | Declaration/event projections, tombstones, indexes, and event moderation subject constraints | FR-019, FR-021, FR-022, NFR-002, NFR-004 | IT-004–IT-006, IT-009–IT-011, IT-015 |
| `appview/internal/index/craftsky_business_profile.go`, `craftsky_business_event.go` | Create | Generated-type decoding, raw preservation, materialized query fields, and revision-aware projection | FR-021, FR-026 | AT-009, IT-004, IT-005 |
| `appview/internal/index/transactional_dispatcher.go`, `transactional_projectors.go` | Change | Validate/register new records and apply the explicit membership-independent business projection policy | FR-021, FR-022 | IT-005, IT-006 |
| `appview/internal/index/business_*_test.go` | Create | Projection, dispatch, replay, tombstone, unknown-source, and dependency-order tests | FR-021, FR-026, NFR-002 | AT-009, IT-004–IT-006 |
| `appview/internal/ingestion/service.go`, business ingestion tests | Change | Permit durable business projection without a missing-member retry while retaining terminal fences | FR-021 | IT-006 |
| `appview/internal/app/deps.go`, `deps_tap.go`, `deps_content.go`, `routes_adapter.go` | Change | Construct/register business projectors/store and pass API dependencies | FR-016–FR-024 | IT-006, IT-012 |
| `docker-compose.yml` | Change | Add both collections to `TAP_COLLECTION_FILTERS` | FR-021 | IT-006 |
| `appview/internal/api/business_account_type.go`, `business_profile.go`, `business_event.go` | Create | Exact mutation/read handlers and response DTOs | FR-002, FR-016–FR-019, FR-023–FR-025 | AT-001–AT-007, AT-011, IT-002, IT-003, IT-007–IT-010 |
| `appview/internal/api/business_*_request.go`, `business_event_cursor.go` | Create | Strict camelCase decoding, `If-Match`, createdAt exclusion, limits, and opaque cursors | FR-018, FR-019, FR-023–FR-025, NFR-003 | UT-021–UT-023, IT-009, IT-012 |
| `appview/internal/api/business_*_test.go` and acceptance test files from `02-acceptance-tests.md` | Create | Handler, store, cursor, route, security, neutrality, and end-to-end acceptance tests | All in-scope requirements | AT-001–AT-014, IT-002, IT-003, IT-007–IT-013, IT-017–IT-021 |
| `appview/internal/api/profile_response.go`, `profile.go` | Change | Add nullable account/business fields to ordinary responses; blocked marshal omits both; full profile loads eligible declaration details | FR-003, FR-020, NFR-004 | AT-003, AT-008, IT-004, IT-011, REG-002, REG-003 |
| `appview/internal/api/profile_account_type_hydrator.go` | Create | One deduplicated batch lookup to decorate all visible `{did,handle}` identity objects, defaulting missing rows to regular | FR-020 | AT-001, AT-008, IT-011 |
| `appview/internal/routes/routes_business.go`, `policy.go`, `catalogue.go`, route tests | Create/change | Capability bundle and exact route/policy inventory | FR-018, FR-019, FR-023, FR-024, NFR-003 | AT-011, IT-012 |
| `appview/internal/routes/dependencies.go`, `routes.go`, `routes_profile_notification.go` | Change | Inject business store/hydrator and pass business profile reader to profile handlers | FR-003, FR-020 | IT-004, IT-011, IT-012 |
| `appview/internal/api/report.go`, `report_store.go`, `moderation_*.go` | Change | Add closed `event` record subject support and exact event URI visibility/moderation behavior | FR-019, NFR-004 | AT-007, IT-010 |
| `appview/internal/accountdeletion/collections.go`, `pds_deleter.go`, `lifecycle.go`, `private_cleanup.go` | Change | Enforce observable events → declaration → account type → membership stages with retry safety | FR-022, RULE-006 | AT-010, IT-015, REG-007 |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Register every DID role in account type, projection, and tombstone tables | FR-022, NFR-002 | IT-015 |
| `.gitattributes`, `justfile`, `scripts/appview-check` | Change | Mark generated files and add `business-cataloggen[-check]` plus release-gate drift checks | NFR-001 | AT-013, IT-016 |

## 5. Services, Interfaces, And Data Flow

### Domain and store contracts

`internal/business` is the only package that decides whether independently authored public source is safe to expose. API and index packages pass typed records and policy inputs to it rather than duplicating catalogs or reason ordering.

```text
type AccountType string // exact regular | business

func ParseAccountType(string) (AccountType, error)
func ResolveAccountType(row *AccountType) AccountType // nil => regular

type Store struct { pool *pgxpool.Pool }
func (s *Store) ReadAccountType(ctx, did) (AccountType, error)
func (s *Store) ReadAccountTypes(ctx, dids) (map[DID]AccountType, error)
func (s *Store) PutAccountType(ctx, did, AccountType) error
func (s *Store) DeleteAccountType(ctx, did) error

func ValidateProfileWrite(ProfileWrite) fieldErrors
func HydrateProfile(raw json.RawMessage) ProfileView
func MergeProfileReplacement(existingRaw, replacementKnown) json.RawMessage

func ValidateEventCreate(EventWrite, now) fieldErrors
func ValidateEventUpdate(EventWrite, storedCreatedAt, now) fieldErrors
func HydrateEvent(raw json.RawMessage) EventView
func EvaluateEvent(EligibilityInput) EligibilityResult
```

`EligibilityResult` always builds reason arrays by iterating fixed canonical slices, so values are distinct, bounded, non-null, and order-stable. Declaration presence is not an input. Owner access requires the caller to be the current owner; visitor direct/list access additionally applies blocks, current membership, business account type, valid range, duration, and record moderation. Upcoming adds ended/cancelled/postponed exclusions. Eligible past/cancelled/postponed records remain visitor-direct-readable.

### Projection revision algorithm

Each projection row stores `source_revision`; each delete leaves a corresponding tombstone with the same key and revision. Revision values are canonical Tap repository TIDs and compare in their canonical lexical order.

```text
Project(event):
  lock current row and tombstone for event URI
  newest = max(row.sourceRevision, tombstone.sourceRevision)
  if event.rev <= newest: return applied/no-op
  if delete:
    delete projection row
    upsert tombstone(event.uri, event.rev)
  else:
    decode generated lexicon type for schema validity
    preserve complete event.Record as raw JSON
    materialize only safe query columns
    upsert projection row with event.rev and CID
    delete older tombstone
```

The dispatcher gets a closed collection policy, not a blanket lifecycle bypass. For the two business NSIDs it still locks owner lifecycle state, rejects terminal owners, and requires matching projection generation when available, but it does not require a membership row or active membership for create/update. Existing collections retain their current behavior unchanged.

### Mutation flow

Account type:

```text
current-member middleware -> strict {accountType} decode -> ParseAccountType
  -> Store.PutAccountType -> 200 {accountType}
  -> no EffectExecutor and no PDS request
```

Declaration PUT/DELETE and event PUT/DELETE:

```text
current-member middleware -> verify path DID equals authenticated DID where present
  -> parse required If-Match canonical CID
  -> EffectExecutor.ReadRecord for authoritative CID/raw source
  -> reject mismatch as 409 pds_record_conflict
  -> declaration: full-replace known fields and merge unknown top-level source fields
  -> event: build the approved event shape and preserve stored createdAt
  -> EffectExecutor.PutRecord/DeleteRecord with ExpectedCID
  -> return record view including CID without waiting for Tap
```

Event POST allocates `newImmediateRecordKey()`, rejects any client `createdAt`, stamps `now().UTC().Truncate(time.Second).Format(time.RFC3339)`, writes with `social.craftsky.business.event`, and returns DID/rkey/URI/CID. Event update rejects `createdAt` whenever the key is present, including an identical value.

No validator receives an HTTP client or DNS resolver. Destination validation performs syntax-only parsing and ASCII checks.

### Profile and summary reads

The existing full-profile read obtains the normal profile row first. If blocked, it returns the existing reduced shell. Otherwise it asks `business.Store` for account type and, only when current membership plus `business` is true, safely hydrates the declaration into an optional `business` object. Events are never embedded.

All route responses continue through the existing customisation hydrator and a new account-type hydrator. The account-type hydrator recursively collects deduplicated visible identity objects, skips objects with `blocking` or `blockedBy` true and reduced placeholders, performs exactly one batch query, and writes `accountType` for every retained identity, using `regular` for absent rows. Tests compare query counts for one and fifty identities.

### Event queries

Public profile events query materialized rows by `(starts_at ASC, uri ASC)`, applies owner membership/type, block, moderation, duration, status, and the cursor's frozen `asOf`. The first page captures `asOf = now.UTC()`; subsequent cursors carry `kind`, `asOf`, `startsAt`, and `uri`. Owner management uses `(starts_at DESC, uri DESC)` and a separate cursor kind carrying `startsAt` and `uri`. Both fetch `limit + 1` to decide cursor presence.

Direct event reads resolve `{did}/{rkey}` to one row, then choose owner-management or visitor policy. Suppressed visitors always receive `404 event_not_found`; no shell is emitted.

### Moderation and reports

Extend the existing closed report/moderation subject type with `event`. Database checks require collection/rkey/URI for both post and event record subjects. Event report target resolution uses the same visitor visibility policy as direct event GET, snapshots the current event CID, and persists the exact event URI. Active hide/takedown outputs feed `record-moderated` in event policy; negate restores normal evaluation.

### Permanent deletion

The current lifecycle runs broad private cleanup before PDS deletion, so account type must not be added to that early generic list. Add explicit resumable stages to the deletion processor:

```text
1. Converge existing non-business, non-membership Craftsky collections to empty.
2. Converge social.craftsky.business.event to empty.
3. Converge social.craftsky.business.profile to empty.
4. Delete the craftsky_account_types row.
5. Converge social.craftsky.actor.profile membership to empty.
6. Prove the complete closed registry empty and finalize existing cleanup.
```

`LifecycleProcessor` invokes these idempotent stages in this order on every attempt; no new stage ledger is needed. A retry safely replays already-empty collection scans and the account-type delete before proceeding, so no later stage can run ahead of an earlier stage. The existing operation lease/failure state remains the retry authority, and the exact safety-record path remains collection-allowlisted. Ordinary departure does not invoke these stages and deletes no business data.

## 6. State, Providers, Controllers, Or DI

No Flutter/Riverpod state is in scope.

Go dependency graph:

```text
pgxpool.Pool
  -> business.Store
  -> index.TransactionalCraftskyBusinessProfile
  -> index.TransactionalCraftskyBusinessEvent
  -> api.IdentityAccountTypeHydrator

pdseffects.ExecutorFactory + business.Store + HandleResolver + ReportStore
  -> routes.businessEventRouteBundle
  -> account-type/profile/event handlers

business.Store
  -> profile handlers (declaration-backed view)
  -> v1 middleware (batch accountType hydration)
  -> account deletion staged account-type cleanup
```

Add `*business.Store` to `contentDependencies`, `app.Deps`, `routes.Dependencies`, and the route adapter. Add `accountTypeHydrator` beside the existing customisation hydrator in `v1Middleware`; both decorate only successful JSON and preserve headers/status. Construct projectors from the same pool in `newTransactionalIndexerDispatcherWithActorDeletion`.

Clock injection is required at domain/handler boundaries that evaluate `createdAt`, event creation eligibility, derived temporal state, and cursor `asOf`. Production passes `time.Now`; tests pass a fixed clock.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No Flutter widgets, screens, providers, or navigation changes.

Register this exact current-member/device-authenticated surface in a dedicated `routes_business.go` bundle and in the closed route policy catalogue:

| Method and path | Handler responsibility |
|---|---|
| `PUT /v1/profiles/me/account-type` | Validate/upsert private scalar; return `{accountType}` |
| `PUT /v1/profiles/me/business` | Require `If-Match`; full-replace known declaration fields while preserving unknown top-level extensions |
| `DELETE /v1/profiles/me/business` | Require `If-Match`; delete only declaration |
| `GET /v1/profiles/{handleOrDid}/events` | Resolve profile, enforce block/public eligibility, paginate upcoming events |
| `POST /v1/events` | Validate, stamp `createdAt`, allocate TID, and create event |
| `GET /v1/events` | Paginate all current-owner events with both diagnostic arrays |
| `GET /v1/events/{did}/{rkey}` | Return owner management view or publicly eligible visitor view |
| `PUT /v1/events/{did}/{rkey}` | Owner-only conditional replacement preserving server-owned `createdAt` |
| `DELETE /v1/events/{did}/{rkey}` | Owner-only conditional deletion |
| `POST /v1/events/{did}/{rkey}/reports` | Report a publicly visible event record |

All request/response DTOs use explicit camelCase tags and bare success bodies. Profile responses add `accountType` to normally visible identity shapes and an optional declaration-backed `business` object only on full profiles. Blocked marshaling omits both. Event page bodies are `{items, cursor?}` with absent cursor on completion and `[]`, never `null`, for diagnostic arrays.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing account-type row | Resolve to `regular` without inserting on read | FR-001 | UT-001, IT-001, AT-001 |
| Unsupported/mixed-case account type | Field validation; no storage/PDS effect | FR-001, FR-002 | UT-001, IT-002 |
| Empty declaration | Preserve/index as valid; expose no declaration-backed values | FR-003 | IT-003, IT-004 |
| Regular/departed owner | Preserve source; suppress declaration details and public events | FR-003, FR-017, FR-022 | AT-002, IT-004, IT-008, IT-014 |
| Block relation | Existing reduced profile shell; omit account type/business; empty event list; direct event 404 | FR-020, NFR-004 | AT-008, IT-008, IT-011, REG-002 |
| Unknown independent catalog value | Preserve raw; represent safely without entitlement; omit only unsafe subordinate values | FR-026 | UT-004, UT-020, IT-004, IT-006, IT-008 |
| Invalid independent location | Keep declaration; project valid country only or omit location per RULE-005 | FR-006, RULE-005 | UT-007, IT-004 |
| Unsafe independent destination/price | Keep raw record/card/event; omit unsafe hydrated field; never fetch/resolve | FR-026, NFR-007 | UT-009, UT-013, IT-004, IT-008, IT-021 |
| Invalid/reversed event range | First-party write 422; independent raw retained; owner diagnostic; visitor 404/list omission | FR-013, FR-016 | UT-015, UT-019, IT-008 |
| Event over 31 days | First-party write 422; independent raw retained; owner diagnostic; public suppression | FR-014, FR-019 | UT-017, IT-008, IT-010 |
| Ended/cancelled/postponed event | Exclude upcoming; retain eligible visitor direct read and owner management | FR-016, FR-017 | AT-005, UT-019, IT-008 |
| Omitted independent status/mode/timezone/isAllDay | Scheduled behavior; mode/timezone unspecified; `isAllDay: false` | FR-011, FR-012 | UT-014, IT-008 |
| Suppressed visitor direct read | Standard `404 event_not_found`; no redacted event | FR-016 | AT-005, IT-008 |
| Malformed/tampered cursor | Standard camelCase 4xx envelope with request ID | FR-018, FR-019, NFR-003 | UT-021, IT-009, IT-012 |
| Missing/malformed/stale `If-Match` | `409 pds_record_conflict`; no mutation | FR-024 | UT-022, IT-003, IT-007, REG-004 |
| Client-supplied `createdAt` | Reject on create/update before PDS mutation | FR-025 | UT-023, AT-005, IT-007 |
| Equal/older projection revision | Idempotent no-op; newer row/tombstone remains | FR-021 | AT-009, IT-005, IT-006 |
| Record before membership/type | Persist immediately without dependency retry; suppress until read-time eligible | FR-021 | AT-009, IT-005, IT-006 |
| No event results | Return `{items: []}` with no cursor | FR-018, FR-019 | IT-009, IT-010 |
| Downstream PDS/session failure | Existing bounded PDS error mapping; authored values excluded from telemetry | NFR-003, NFR-006 | IT-007, IT-013 |
| Permanent deletion retry/absent membership | Resume fixed stages idempotently and preserve exact order | FR-022, RULE-006 | AT-010, IT-015, REG-007 |

There are no client-side loading states in this implementation slice.

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `internal/business/account_type_test.go` | Missing/exact/invalid scalar table | Domain type/default does not exist |
| 2 | IT-001 | `internal/db/business_profiles_migration_test.go`, `internal/business/store_test.go` | Real Postgres through migration 000061 | Table, constraint, and store do not exist |
| 3 | UT-002, UT-003, UT-004 | `internal/business/catalog_test.go` | TD-002 exact and unknown catalogs | Catalog validation/order policy absent |
| 4 | UT-005, UT-006, UT-007 | `internal/business/text_validation_test.go`, `location_test.go` | TD-003/TD-005 plus generated country catalog | Bounds and safe location projection absent |
| 5 | UT-008, UT-009 | `internal/business/action_test.go`, `destination_test.go` | TD-004 destination corpus and no-network sentinels | Action/destination grammar absent |
| 6 | UT-010, UT-011 | `internal/business/product_test.go` | TD-006/TD-011 product and image corpus | Product/image/count/order rules absent |
| 7 | UT-012, UT-013 | `internal/business/money_test.go` | TD-007 and pinned SIX catalog | Canonical money validator/hydrator absent |
| 8 | UT-014, UT-015, UT-016, UT-017, UT-018 | `internal/business/event_validation_test.go`, `event_time_test.go` | TD-008/TD-011 fixed clock and timezone corpus | Event validation/default/time/image policy absent |
| 9 | UT-019 | `internal/business/eligibility_test.go` | Full actor/block/lifecycle/moderation/time matrix | Central eligibility and diagnostics absent |
| 10 | UT-020 | `internal/business/profile_merge_test.go` | Raw extension and unsafe subordinate fixtures | Extension-preserving replacement absent |
| 11 | UT-021, UT-022, UT-023 | `internal/api/business_event_cursor_test.go`, `business_record_request_test.go`, `internal/business/event_authoring_test.go` | Fixed clock, cursor tuples, CID and createdAt cases | API parsing/authoring helpers absent |
| 12 | IT-016, AT-013, REG-001 | Lexicon/business contract tests and `just lexgen-check` | ADR, pinned address, TD-015, catalog snapshots, clean generated baseline | Schemas/generated/provenance artifacts absent |
| 13 | IT-004, IT-005, IT-006, AT-009 | Index/business store and ingestion tests | Migration 000062, real Postgres, R1–R4 revisions, TD-010 | Raw projections, tombstones, registrations absent |
| 14 | IT-002, AT-001 | `internal/api/business_account_type_test.go` and acceptance test | Current/departed/unauthenticated actors, PDS call recorder | Account-type route/handler absent |
| 15 | IT-003, AT-002, AT-003, AT-004, AT-011 | Business profile/eligibility/ownership/routes tests | Fake PDS with extension-bearing singleton and CIDs | Declaration APIs and profile hydration absent |
| 16 | IT-007, AT-005, REG-004 | `internal/api/business_event_test.go` and acceptance test | Fake mutable PDS, fixed clock, media corpus, projector fixture | Event CRUD/CID/createdAt contract absent |
| 17 | IT-008, AT-002, AT-005 | `internal/api/business_event_store_test.go` | Real store eligibility matrix | Direct/upcoming/owner policy not wired |
| 18 | IT-009, AT-006, REG-003 | `internal/api/business_event_pagination_test.go` | More than 50 events, equal starts, frozen clock, concurrent mutations | Dedicated list/cursors absent |
| 19 | IT-010, AT-007 | Event management and moderation tests | All suppression states, report store, apply/negate moderation | Diagnostics/event report subject absent |
| 20 | IT-011, AT-008, REG-002 | Summary contract/query-plan tests | TD-012 with one and fifty identities and blocks | Account type absent or query count grows |
| 21 | IT-012 | `internal/app/business_routes_test.go` | Full router, auth/device policy, exact and near-miss routes | Route catalogue/bundle absent |
| 22 | IT-014, AT-010 | Membership lifecycle tests | Business member departure/rejoin with retained source | State retention/restore not demonstrated |
| 23 | IT-015, REG-007 | Account deletion acceptance/collection tests | TD-013 effect recorder, interruption at each stage | Required four-stage order absent |
| 24 | IT-013, AT-014, MAN-002 | Observability/security tests and manual review | TD-004/TD-014 canaries, capturing logger/metrics/traces | New paths may leak authored values |
| 25 | IT-017, IT-018, IT-019, AT-012, REG-005, REG-006 | Feed/search/policy neutrality tests | Equivalent regular/business fixtures | Neutrality has no explicit regression evidence |
| 26 | IT-020, REG-008 | Product wire/schema contract tests | Reflection/schema enumeration | Forbidden commerce semantics not guarded |
| 27 | IT-021 | `internal/business/no_fetch_test.go` | Fail-fast HTTP transport/DNS resolver around all paths | No-fetch property not demonstrated |
| 28 | AT-001–AT-014 | Acceptance targets named in `02-acceptance-tests.md` | Compose Postgres, deterministic fakes, fixed clocks | Cross-layer approved scenarios not yet proven together |
| 29 | MAN-001 | ADR/schema/provenance review | ADR 010, generated diff, digest/CID evidence | Human durable-schema approval not recorded |
| 30 | Full gate | Existing and new suites | Dev stack and isolated release check | Integration/regression drift may remain |

Focused commands by slice:

```text
go test ./internal/business -run TestAccountType
just appview-test-unit
just test
just business-cataloggen-check
just lexgen-check
just appview-check
just fmt
```

Run `go test` from `appview/`. `just test` requires the Compose Postgres described in `AGENTS.md`. `just fmt` is a final formatting action, not an initial validation command.

## 10. Sequencing And Guardrails

- First TDD step: Add failing UT-001 for exact `regular`/`business`, invalid values, and missing-row default; implement only the domain scalar/default needed to pass.
- Second TDD step: Add failing IT-001; create migration 000061 and the minimal account-type store/upsert needed to pass.
- Dependencies between work items: Domain catalogs/validators precede API decoders. ADR 010 and provenance verification precede any lexicon edit. Lexicons and generated types precede projectors. Migration 000062 precedes projector/store integration. Projection precedes read APIs. Eligibility precedes all event read handlers. CID/request helpers precede mutation handlers. Staged deletion lands only after both collections and account-type persistence exist.
- Lexgen bootstrap: Generate JSON types first; add temporary local CBOR method stubs only as described by the existing codegen design; run cborgen for Craftsky and community packages; delete all temporary stubs before the slice is considered green.
- Guardrail: Account type remains private AppView state and never appears in a PDS record or lexicon.
- Guardrail: Business projection bypasses only active-membership presence, not terminal-owner or repository-generation safety.
- Guardrail: Store raw public source before hydration decisions; never rewrite source to remove unknown or unsafe independent values.
- Guardrail: One URI is mutated only by a strictly newer repository revision; deletes always retain a tombstone.
- Guardrail: Public event eligibility has one domain implementation. SQL may prefilter candidates but must not independently redefine reason semantics.
- Guardrail: Blocked responses are fail-closed. Generic response hydration must explicitly skip blocked/reduced identities.
- Guardrail: PDS replacement uses authoritative `ReadRecord`, required `If-Match`, and `ExpectedCID`; no last-write-wins fallback.
- Guardrail: Authored destinations are parsed only. Do not inject or call DNS, HTTP, preview, link-expansion, or email services.
- Guardrail: Logs/metrics/traces use bounded operation/result/reason constants and never raw authored values.
- Guardrail: Account type is not added to ordinary departure cleanup. Only the approved permanent deletion stages remove it.
- Guardrail: Do not add account type/taxonomy/business joins to feed ordering, search ranking/filtering, permissions, or moderation priority.
- Out of scope: All Flutter work; subscriptions/verification; native commerce; search/filter/ranking changes; structured event location; recurrence; global event discovery; booking/registration; secondary links; `onlineUri`; runtime catalog updates; real destination checks; generic PDS deletion.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | The earlier `Changes required` verdict needed reconciliation after the revised contract was approved. | Workflow history could appear inconsistent. | `03-document-review.md` now records the approved re-review outcome while preserving prior findings as history. |
| CPQ-002 | Non-blocking | The official ISO 3166-1 OBP snapshot's native media type must match the checked-in file extension. | A converted file would weaken provenance. | Save the exact retrieved bytes under the planned date-stamped native extension; if not HTML, adjust only the extension and metadata together before generating data. Never silently convert it. |
| CPQ-003 | Non-blocking | Shared business defs add a third schema file even though there are only two public record types. | Generated names/imports differ from embedding defs in the profile schema. | Use `social.craftsky.business.defs`; it is not a record and follows the lexicon shared-definition convention. ADR 010 records this choice. |
| CPQ-004 | Non-blocking | Current deletion performs broad private cleanup before PDS collection deletion. | Adding account type to current cleanup would violate required ordering. | Keep account type out of generic private cleanup; have `LifecycleProcessor` replay the fixed idempotent collection/account-type sequence on every retry. |
| CPQ-005 | Non-blocking | Generic JSON response decoration can accidentally label blocked identity shells. | Privacy leak across many summary shapes. | Skip maps with block flags/reduced shape, retain explicit blocked marshaling, and require the one-versus-fifty query-count/redaction contract tests before route rollout. |
| CPQ-006 | Non-blocking | Seek pagination cannot snapshot concurrent event edits. | Concurrent ordering-key changes can cause normal seek duplicates/omissions. | Freeze only `asOf` time eligibility; document/test complete traversal only for unchanged record sets, exactly as approved. |
| CPQ-007 | Non-blocking | Independent records can carry unknown values and broad address fields. | Narrow typed projections could destroy source fidelity. | Store complete raw JSON and use generated types only for schema decoding/materialized safe fields; contract tests compare raw retention. |
| CPQ-008 | Non-blocking | Full AppView-to-real-PDS acceptance infrastructure is absent. | Mutation convergence is split across boundaries in tests. | Use contract-faithful fake PDS effect tests plus real-Postgres projection tests, as approved in GAP-003; no new unstable real-PDS harness. |

No blocking implementation question remains.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` in `appview/internal/business/account_type_test.go`
- First focused command: `go test ./internal/business -run TestAccountType`
- Follow with: `IT-001` using real Postgres, migration `000061_business_account_types`, and `internal/business/store_test.go`.
- Notes: Invoke `implement-tdd` to create/update `05-implementation-plan.md` and execute strict red-green-refactor slices. Invoke `atproto-lexicon` again before the lexicon slice, create ADR 010 before editing `lexicon/`, and preserve all generated artifacts with their source schemas/snapshots.
