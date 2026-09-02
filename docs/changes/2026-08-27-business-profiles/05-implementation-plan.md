# TDD Implementation Plan: Business Profiles

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (Approved, 2026-08-28)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and red/green evidence updated after every test ID.
- Create ADR 010 and invoke the `atproto-lexicon` skill before editing `lexicon/`.
- Preserve the approved high-risk controls for migrations, authentication, privacy, moderation, and permanent deletion.
- Do not create a stage commit unless the user explicitly enables commits.

Sections before **Package-Backed Catalog Simplification** preserve the original snapshot-generator implementation history. That later section and the amended requirements, tests, coding plan, and ADR define the current catalog contract.

## Test Order

The order initially mirrored `04-coding-plan.md` section 9. Each unique test ID appears once at its first implementation point; the final acceptance gate reruns AT-001 through AT-014 together. During the original implementation, UT-006 and UT-007 were deferred until the then-required official source snapshot was acquired; the later package-backed simplification supersedes that implementation constraint.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001 | AC-007 | Fails: account-type domain does not exist |
| 2 | IT-001 | BR-001, FR-001 | AC-001, AC-007 | Fails: migration/store do not exist |
| 3 | UT-002 | FR-004, RULE-007 | AC-010 | Fails: business-type catalog absent |
| 4 | UT-003 | FR-004, RULE-008 | AC-010 | Fails: offering catalog absent |
| 5 | UT-004 | FR-026, RULE-010 | AC-006, AC-039 | Fails: open-value policy absent |
| 6 | UT-005 | FR-005, FR-008, FR-012 | AC-003, AC-011, AC-016, AC-020 | Fails: text validation absent |
| 7 | UT-006 | FR-006 | AC-012 | Fails: location validation absent |
| 8 | UT-007 | FR-006, RULE-005 | AC-013 | Fails: safe location hydration absent |
| 9 | UT-008 | FR-007, RULE-009 | AC-014 | Fails: action validation absent |
| 10 | UT-009 | FR-007, FR-008, FR-015, NFR-007 | AC-015 | Fails: destination grammar absent |
| 11 | UT-010 | FR-008, NFR-005 | AC-016 | Fails: product image policy absent |
| 12 | UT-011 | FR-009 | AC-017 | Fails: product collection policy absent |
| 13 | UT-012 | FR-010 | AC-018 | Fails: money catalog/validator absent |
| 14 | UT-013 | FR-010 | AC-018 | Fails: safe money hydration absent |
| 15 | UT-014 | FR-011, FR-012, RULE-011 | AC-019, AC-020 | Fails: event catalogs/defaults absent |
| 16 | UT-015 | FR-013 | AC-021 | Fails: event instant validation absent |
| 17 | UT-016 | FR-013 | AC-021 | Fails: all-day/DST policy absent |
| 18 | UT-017 | FR-014 | AC-022 | Fails: duration policy absent |
| 19 | UT-018 | FR-015, NFR-005 | AC-024 | Fails: event link/image policy absent |
| 20 | UT-019 | FR-013, FR-016, FR-017, RULE-001, RULE-003 | AC-002, AC-021, AC-025, AC-026, AC-027, AC-031 | Fails: eligibility policy absent |
| 21 | UT-020 | FR-003, FR-023, FR-026 | AC-009, AC-033, AC-036 | Fails: replacement merge absent |
| 22 | UT-021 | FR-018, NFR-003 | AC-028, AC-042 | Fails: cursor helper absent |
| 23 | UT-022 | FR-024, NFR-003 | AC-037, AC-042 | Fails: If-Match helper absent |
| 24 | UT-023 | FR-025 | AC-038 | Fails: createdAt policy absent |
| 25 | IT-016 | FR-009, NFR-001 | AC-017, AC-041 | Fails: ADR/schemas/provenance absent |
| 26 | AT-013 | NFR-001 | AC-041 | Fails: reproducible generation absent |
| 27 | REG-001 | FR-003, NFR-001 | AC-009, AC-041 | Fails: business schema regression guard absent |
| 28 | IT-004 | BR-002, BR-003, FR-003, FR-004, FR-006, FR-008, FR-010, FR-026, RULE-005, NFR-005 | AC-003, AC-004, AC-009, AC-010, AC-013, AC-016, AC-018, AC-033 | Fails: declaration projection absent |
| 29 | IT-005 | FR-011, FR-021, NFR-002 | AC-005, AC-032, AC-033 | Fails: event projection absent |
| 30 | IT-006 | FR-021, FR-026, NFR-002 | AC-032, AC-033 | Fails: dispatch/ingestion wiring absent |
| 31 | AT-009 | FR-021, FR-026, NFR-002 | AC-032, AC-033 | Fails: federated convergence absent |
| 32 | IT-002 | FR-001, FR-002, RULE-002 | AC-007, AC-008 | Fails: account-type HTTP mutation absent |
| 33 | AT-001 | BR-001, FR-001, FR-002, RULE-001 | AC-001, AC-007 | Fails: classification acceptance path absent |
| 34 | IT-003 | FR-003, FR-023 | AC-009, AC-036 | Fails: declaration mutation absent |
| 35 | AT-002 | BR-001, FR-017, RULE-001, RULE-003 | AC-002, AC-027 | Fails: declaration-independent eligibility not integrated |
| 36 | AT-003 | BR-002, BR-003, FR-005, FR-006, FR-008, RULE-004 | AC-003, AC-004, AC-012, AC-040 | Fails: declaration presentation not integrated |
| 37 | AT-004 | FR-002, RULE-002 | AC-008 | Fails: ownership path not integrated |
| 38 | AT-011 | FR-023, FR-024, NFR-003 | AC-036, AC-037 | Fails: route/CID contracts incomplete |
| 39 | IT-007 | FR-011, FR-016, FR-024, FR-025, RULE-002, NFR-005 | AC-005, AC-024, AC-025, AC-037, AC-038 | Fails: event CRUD absent |
| 40 | AT-005 | BR-003, FR-011, FR-016, FR-017, FR-025 | AC-005, AC-025, AC-026, AC-038 | Fails: event lifecycle not integrated |
| 41 | REG-004 | FR-024, RULE-002 | AC-025, AC-037 | Fails: new PDS paths unguarded |
| 42 | IT-008 | FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-026, RULE-003, NFR-004 | AC-020, AC-021, AC-023, AC-024, AC-025, AC-026, AC-027, AC-031, AC-033 | Fails: caller-aware serving absent |
| 43 | IT-009 | FR-018, NFR-003 | AC-028, AC-042 | Fails: event pagination query absent |
| 44 | AT-006 | FR-018, NFR-003 | AC-028 | Fails: pagination acceptance path absent |
| 45 | REG-003 | FR-018 | AC-028 | Fails: profile collection guard absent |
| 46 | IT-010 | FR-019, NFR-004 | AC-023, AC-029 | Fails: management/reporting integration absent |
| 47 | AT-007 | FR-014, FR-019, NFR-004 | AC-023, AC-029 | Fails: diagnostics acceptance path absent |
| 48 | IT-011 | FR-020, NFR-004 | AC-030, AC-031 | Fails: summary hydration/redaction absent |
| 49 | AT-008 | FR-020, NFR-004 | AC-030, AC-031 | Fails: summary acceptance path absent |
| 50 | REG-002 | FR-020, NFR-004 | AC-031 | Fails: blocked shapes unguarded for business fields |
| 51 | IT-012 | FR-024, NFR-003 | AC-037, AC-042 | Fails: routes not registered |
| 52 | IT-014 | BR-001, FR-022, RULE-001, RULE-006 | AC-002, AC-034 | Fails: retained lifecycle not demonstrated |
| 53 | AT-010 | FR-022, RULE-006 | AC-034, AC-035 | Fails: lifecycle acceptance path absent |
| 54 | IT-015 | FR-022, RULE-006 | AC-035 | Fails: deletion stages absent |
| 55 | REG-007 | FR-022, RULE-006 | AC-035 | Fails: deletion registry guard incomplete |
| 56 | IT-013 | NFR-006 | AC-043 | Fails: business observability paths unverified |
| 57 | AT-014 | FR-007, FR-008, FR-015, NFR-006, NFR-007 | AC-015, AC-043 | Fails: security acceptance path absent |
| 58 | IT-017 | BR-004, RULE-010 | AC-006 | Fails: feed neutrality unverified |
| 59 | IT-018 | BR-004, RULE-010 | AC-006 | Fails: search neutrality unverified |
| 60 | IT-019 | BR-004, RULE-010 | AC-006 | Fails: policy neutrality unverified |
| 61 | AT-012 | BR-004, RULE-010 | AC-006 | Fails: non-entitlement acceptance path absent |
| 62 | REG-005 | BR-004, RULE-010 | AC-006 | Fails: ranking neutrality guard absent |
| 63 | REG-006 | BR-004, RULE-010 | AC-006 | Fails: authorization neutrality guard absent |
| 64 | IT-020 | RULE-004 | AC-040 | Fails: price wire contract unverified |
| 65 | REG-008 | RULE-004 | AC-040 | Fails: commerce-semantics guard absent |
| 66 | IT-021 | FR-007, FR-008, FR-015, NFR-007 | AC-015 | Fails: no-fetch property unverified |

## Manual Checks

| Step | Test ID | Requirement IDs | Acceptance Criteria | Status |
|---|---|---|---|---|
| M1 | MAN-001 | NFR-001 | AC-041 | Completed 2026-08-29 after deterministic generation and the release gate passed |
| M2 | MAN-002 | NFR-006 | AC-043 | Completed 2026-08-29 after IT-013, AT-014, and IT-021 passed |

## Implementation Steps

Each completed loop is recorded below before moving to the next test ID.

### Step 1: UT-001
- Write failing test: Added exact parse/default coverage in `appview/internal/business/account_type_test.go`.
- Run command: `go test ./internal/business -run TestAccountType`
- Confirmed failure: Build failed only on the missing `AccountType`, constants, parser, resolver, and sentinel error.
- Implement: Added the minimum closed scalar API in `appview/internal/business/account_type.go`.
- Run command: `go test ./internal/business -run TestAccountType -count=1` passed.
- Refactor: None; implementation is already the minimal existing closed-parser pattern.
- Notes: Empty, whitespace-padded, mixed-case, and `pro` values remain invalid. Missing storage state is represented only by `nil`, not by an empty string.

### Step 2: IT-001
- Write failing test: Added `TestBusinessAccountTypesMigration`, which failed because migration 000061 was absent.
- Run command: `go test ./internal/db -run TestBusinessAccountTypesMigration -count=1`
- Confirmed failure: The exact approved migration files did not exist.
- Implement: Added the private table, primary key, and closed database check without a membership foreign key.
- Run command: Real-Postgres migration test passed with `TEST_DATABASE_REQUIRED=true`.
- Write failing test: Added `TestStoreAccountType`, which then failed only because `NewStore` was absent.
- Implement: Added missing-row default reads, validated upserts, and typed DID parameters.
- Run command: Real-Postgres `TestAccountType|TestStoreAccountType` and migration tests passed.
- Refactor: None; HTTP authorization and lifecycle guarding remain assigned to IT-002.
- Notes: Reads do not persist default rows. Direct SQL invalid values are rejected, and account-type rows survive membership deletion.

### Steps 3-5: UT-002, UT-003, UT-004
- UT-002 red: Business-type catalog and validator symbols were absent.
- UT-002 green: Exact RULE-007 catalog, maximum five, uniqueness, exact matching, and canonical order pass.
- UT-003 red: Offering catalog and validator symbols were absent.
- UT-003 green: Exact RULE-008 catalog, maximum ten, uniqueness, exact matching, and canonical order pass.
- UT-004 red: Open-value classification and closed first-party validation symbols were absent.
- UT-004 green: Independent strings are preserved with explicit known/unknown classification; first-party unknown values fail without adding entitlement behavior.
- Run command: `go test ./internal/business -run 'Test(BusinessTypeCatalog|OfferingCatalog|OpenCatalogValueClassification)$' -count=1`
- Refactor: None; shared catalog membership is one small helper and catalog-specific limits/errors remain explicit.

### Step 6: UT-005
- Write failing test: Added field-rule coverage for declaration, product, and event text with ASCII, combining graphemes, and multi-code-point emoji.
- Confirmed failure: Text field types, bounds, typed errors, and validator were absent.
- Implement: Added exact grapheme/byte rules using `uniseg` and UTF-8 validation.
- Run command: `go test ./internal/business -run TestBusinessTextValidation -count=1` passed.
- Refactor: None.
- Notes: UT-006 and UT-007 were moved to the provenance slice after the official OBP UI exposed only 25/249 rows to noninteractive retrieval and Cloudflare blocked the interactive full-list control. No third-party or runtime catalog was substituted.

### Steps 9-10: UT-008, UT-009
- Red: Focused action and destination tests failed on missing catalogs, cardinality policy, and destination parsers.
- Green: Added the exact action catalog, zero-or-one action rule, common credential-free HTTPS grammar, and strict lowercase simple `mailto:` grammar.
- Run commands: `go test ./internal/business -run TestActionValidation -count=1` and `go test ./internal/business -run TestDestinationValidation -count=1` passed.
- Notes: Validation is syntax-only and receives no HTTP client or resolver.

### Steps 11-14: UT-010, UT-011, UT-012, UT-013
- Red: Product/image, collection, currency-scale, canonical-price, and independent hydration symbols were absent.
- Green: Added the shared 15 MiB JPEG/PNG/WebP image policy, product requirements/order/duplicate checks, generated active ISO 4217 scales, canonical decimal validation, and subordinate-only unsafe price omission.
- Provenance: Pinned SIX List One XML, metadata, and SHA-256 `838dfb991648cf36df939edd5fe3811737962b75a32252847d239cedd1e291c9`; added the deterministic offline currency generator.
- Run commands: Focused `TestProduct`, `TestMoney`, and `TestHydrateIndependentPrice` tests passed.
- Refactor: None; schema maximum 20 remains separate from Craftsky write maximum 4.

### Steps 15-20: UT-014 through UT-019
- Red: Event catalogs/defaults, canonical time parsing, all-day/DST checks, duration policy, media rules, and centralized eligibility were absent.
- Green: Added open independent defaults, closed first-party catalogs, whole-second UTC instant validation, timezone-aware all-day boundaries, 31-day/new-event policy, safe media hydration, and canonical public/upcoming diagnostic ordering.
- Run commands: Focused event validation, event time, and `TestEventEligibility` tests passed.
- Notes: Independent invalid records remain representable; suppression is read-time policy and declaration presence is not an eligibility input.

### Step 21: UT-020
- Red: Known-field replacement merge and safe independent profile hydration were absent.
- Green: Added extension-preserving replacement and subordinate-only omission for unsafe independent values.
- Run command: `go test ./internal/business -run TestProfileReplacementMergeAndSafeHydration -count=1` passed.

### Step 22: UT-021
- Red: `TestBusinessEventCursorAndLimits` failed only on missing event cursor and limit helpers.
- Green: Added opaque event cursors with separate upcoming/management kinds, frozen upcoming `asOf`, seek `startsAt`/URI values, malformed/wrong-kind rejection, defaults 10/20, and cap 50.
- Run command: `go test ./internal/api -run TestBusinessEventCursorAndLimits -count=1` passed.

### Step 23: UT-022
- Red: `TestBusinessRecordIfMatch` failed only on missing canonical CID precondition helpers and conflict response.
- Green: Added canonical bare-CID parsing, current-CID comparison, and `409 pds_record_conflict` standard envelope output.
- Run command: `go test ./internal/api -run TestBusinessRecordIfMatch -count=1` passed.

### Step 24: UT-023
- Red: `TestEventCreatedAtIsServerOwned` failed only on missing event authoring helpers.
- Green: Create rejects any supplied `createdAt` and stamps UTC whole seconds; update rejects supplied identical/different values and preserves the stored value when omitted.
- Run command: `go test ./internal/business -run TestEventCreatedAtIsServerOwned -count=1` passed.
- Package verification: `go test ./internal/business ./internal/api` passed after UT-023.

### Lexicon prerequisite
- Invoked the `atproto-lexicon` skill and added accepted ADR 010 before any file under `lexicon/` was changed.
- Resolved `did:plc:mtr7qrqtcyseedx3jyr5o7db` to `https://eurosky.social`, retrieved the exact Q17 record, verified the returned AT-URI/CID, and vendored its complete value at the approved CID-named path.
- Initial noninteractive and headless-browser access to the official ISO 3166-1 OBP source was blocked by Cloudflare; no substitute source was used. The resolution and resulting tests are recorded below.

### Steps 7-8: UT-006, UT-007
- Red: Focused location tests failed only on missing assigned-country normalization, locality bounds, and independent hydration helpers.
- Source recovery: The user completed the official OBP Cloudflare challenge in headed Chromium. The rendered Officially assigned codes result contained 249 rows/1,245 cells at page size 300 and was captured as the approved native HTML snapshot with truthful retrieval metadata and SHA-256 `f33ff970874a35c9b0d8a535f00d1ad130e4ef9ecc6d0f7052f4d3bf694984c1`.
- Green: Added deterministic generation of the exact 249 assigned alpha-2 codes, case-insensitive normalization to uppercase, 100-grapheme/1000-byte locality validation, whole-location omission for invalid country, and locality-only omission when oversized.
- Run command: `go test ./internal/business -run 'Test(LocationValidation|IndependentLocationHydration)$' -count=1` passed.
- Notes: `XK`, `ZZ`, malformed codes, and whitespace variants are rejected; no third-party/runtime catalog is used.

### Steps 25-27: IT-016, AT-013, REG-001
- IT-016 red: `TestBusinessLexiconContract` failed because the three business lexicons did not exist; initial `just lexgen` then failed on the unresolved community address reference.
- Implement: Added accepted ADR 010, `business.defs`, singleton declaration, TID event schema, exact CID-named external address value, offline build mapping, generated Craftsky/community JSON and CBOR types, catalog provenance contracts, and generator sidecar/check support.
- Contract evidence: Recomputed the external value's canonical DAG-CBOR CID as `bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq`; schema validation accepts 20 products and rejects 21; exact keys/required fields/maxima/refs/forbidden fields pass.
- AT-013 green: `just lexgen` succeeded repeatedly from local inputs; SHA-256 of every generated Craftsky/community Go file was unchanged across regeneration. `go run ./cmd/businesscataloggen -check` passed twice without writes.
- REG-001 green: The existing actor-profile schema retains SHA-256 `29b3167edab98bb360e2713a1f55861c75180ba551c2adf052fcf639c2589f4f`; the business declaration remains a separate all-optional singleton.
- Run commands: `go test ./internal/lexicon/craftsky ./internal/lexicon/community`, `go test ./internal/business -run TestBusinessCatalogProvenance`, repeated `just lexgen`, and repeated catalog check all passed.

### Steps 28-30: IT-004, IT-005, IT-006
- IT-004 red: Focused tests failed on the absent declaration projector and eligible profile-store read.
- IT-004 green: Added raw declaration projection with materialized owner/URI/CID/revision fields, revision tombstones, safe hydration, and a membership plus authoritative-business read gate. Unknown source survives; unsafe subordinate action/price/location fields are narrowed without dropping otherwise safe cards; regular/non-member reads return no declaration while retaining the projection.
- IT-005 red: `TestBusinessEventProjectionConvergesByRevision` failed on the absent event projector.
- IT-005 green: R1 duplicate create, R2 update/new CID, stale R1, R3 delete, stale R2, and R4 recreate converge by source revision. Delete tombstones prevent stale resurrection, newer recreates clear the tombstone, and raw unknown extensions survive.
- IT-006 red: Business records were rejected as unsupported, missing membership blocked projection, and the production dispatcher had no business registrations.
- IT-006 green: Both generated record types validate and receive the source revision, business projection is independent of membership while retaining terminal/generation fences, production wiring registers both projectors, and Compose forwards both NSIDs through Tap.
- Run commands: focused real-Postgres tests for `TestBusinessProfileProjectionAndSafeHydration`, `TestBusinessProfileStoreEligibility`, `TestBusinessEventProjectionConvergesByRevision`, and `TestNewIndexerDispatcherRegistersBusinessRecordsWithoutMembership`; then `go test ./internal/index ./internal/app ./internal/api ./internal/business -count=1` with `TEST_DATABASE_REQUIRED=true` passed. Static analysis reported no errors.

### Steps 31-32: AT-009, IT-002
- AT-009 confirmation: The specified acceptance target passed on its first run after the lower-level projection slices. Both records arrive through the transactional dispatcher before local membership/type state, retain R2 raw extensions, safely narrow unsupported profile subfields, and reject stale resurrection after R3 tombstones.
- IT-002 red: `TestBusinessAccountTypeMutationAuthenticationAndOwnership` failed only on the absent HTTP handler.
- IT-002 green: Added strict camelCase scalar decoding, authenticated-DID-only ownership, standard validation/error envelopes, and an AppView-private response. The real store write now runs under the active owner-generation fence; unauthenticated, departed, unsupported-value, and attempted different-owner requests do not mutate state. The handler constructor has no PDS dependency.
- Run commands: real-Postgres focused `TestBusinessProjectionPreservesFederatedSourceAndConverges`, `TestBusinessAccountTypeMutationAuthenticationAndOwnership`, and updated `TestStoreAccountType`; then `go test ./internal/index ./internal/app ./internal/api ./internal/business -count=1` with `TEST_DATABASE_REQUIRED=true` passed. Static analysis reported no errors.

### Steps 33-34: AT-001, IT-003
- AT-001 red: The account-type batch response hydrator and store batch lookup were absent.
- AT-001 green: Missing rows decorate normally visible profile/author identities as `regular`; the owner-only mutation changes all occurrences to `business` and back in one deduplicated batch without creating a declaration or invoking a PDS capability.
- Contract clarification: The user selected `If-Match: *` as conditional declaration creation when the singleton is absent. Existing declaration/event mutations continue to require a canonical CID. UT-022 was extended and passed for wildcard absence/presence semantics.
- IT-003 red: Declaration PUT/DELETE handlers were absent; the initial PDS boundary test also exposed that create-only writes needed `swapRecord: null` rather than a literal wildcard CID.
- IT-003 green: Strict declaration decoding/validation, conditional singleton creation, authoritative CID replacement/delete, unknown-extension-preserving known-field replacement, lifecycle-fenced effects, conflict envelopes, and CID responses passed. The PDS effect canonical body and Indigo client now encode the wildcard precondition as an absent-record swap.
- Run commands: focused AT-001, UT-022, and IT-003 tests; `go test ./internal/api ./internal/auth ./internal/pdseffects -count=1` passed.

### Step 36: AT-003
- Red: The full-profile handler had no business reader and `ProfileResponse` had no typed `accountType` or declaration-backed `business` fields.
- Green: Eligible full profiles expose authoritative classification and safely hydrated bounded text, canonical country/locality, action, ordered product/image/price cards; blocked profiles skip business reads and omit both fields. Events and commerce guarantee fields remain absent.
- Run command: real-Postgres `go test ./internal/api -run TestBusinessDeclarationPresentation -count=1` passed; neighboring API/business/routes/app packages passed after constructor updates.

### Steps 35, 37, 39, 41-42: AT-002, AT-004, IT-007, REG-004, IT-008
- IT-007 red: Event POST/PUT/DELETE constructors were absent.
- IT-007 green: Strict event decoding, TID creation, fixed-clock server `createdAt`, media/link/time validation, owner-scoped authoritative reads, immutable update timestamps, canonical CID swaps, and conditional deletes passed with the existing durable effect boundary.
- REG-004 green: Cross-owner and stale/missing/wildcard event update/delete requests are rejected before unintended PDS mutation.
- IT-008 red: Caller-aware event hydration/store interfaces were absent; an intermediate red exposed nil diagnostic arrays.
- IT-008 green: Real-Postgres direct/upcoming reads use centralized `EvaluateEvent` policy for membership, type, reciprocal blocks, moderation, range, duration, and status. Owners retain suppressed management reads; eligible past/cancelled/postponed events remain visitor-direct-readable; independent unsafe/unknown source is narrowed without raw mutation.
- AT-002 confirmation: Classification and event eligibility remain active without a declaration, declaration create/delete changes neither, regular type suppresses visitor serving while retaining raw rows, and switching back to business restores serving.
- AT-004 confirmation: A regular current member can create/edit only their own declaration/event through PDS effects while public hydration remains suppressed; unauthenticated, non-current, and different-DID callers are rejected before the effect factory runs.
- Run commands: focused event CRUD and real-Postgres event store/AT-002/AT-004 tests passed; API/auth/pdseffects/business package tests passed.

### Step 43: IT-009
- Red: Added `business_event_pagination_test.go`; the focused build failed only because `GetProfileBusinessEventsHandler` was absent.
- Green: Added handle/DID resolution, strict limit/cursor errors, default 10/max 50, first-page frozen `asOf`, opaque next cursors, and the `(starts_at ASC, uri ASC)` seek query with `limit + 1`.
- Real-query coverage: More than 50 equal fractional `startsAt` values traverse completely without duplicates or omissions; advancing the injected clock retains first-page eligibility; insert-before, insert-after, unseen delete, and ordering-key mutation exhibit documented non-snapshot seek behavior.
- Run commands: Required real-Postgres focused test and race test passed; all `BusinessEvent` API tests, full `internal/api`, `internal/business`, and static analysis passed.
- Scope: No route registration, owner management, reporting, or product documentation was added.

### Steps 56-57: IT-013, AT-014
- IT-013 red: The focused business observability test showed both business NSIDs collapsing through the Tap fallback and all profile/event PDS read/write/delete operations collapsing to `unknown`.
- IT-013 green: Added only the two business NSIDs and six business PDS operation values to the existing closed registries/mappers. Captured log, metric, and sanitized trace-attribute output excludes unique authored URI, email, free text, title, price, and locality canaries, plus the full owner DID and profile/event AT-URIs. Every emitted metric call passes `observability.ValidateMetricCall`.
- AT-014 confirmation: The security acceptance target drives valid HTTPS query/fragment/port/punycode and lowercase ASCII dot-atom mailto forms through profile/event handlers. Representative HTTP, custom-scheme, credentialed, hostless, single-label, trailing-dot, IP, Unicode, percent-authority, invalid-port, oversized, uppercase-mailto, whitespace, control, percent-email, query, fragment, comma, and semicolon forms are rejected before PDS effects. Product, action, event, registration, and email destinations are represented.
- Telemetry evidence: Capturing request logs, in-memory HTTP metrics, and Sentry transaction/span events contain none of the authored canaries, full DID, or full AT-URI. Metric labels remain bounded route/method/status and approved Tap/PDS operation/result/reason values.
- Run commands: Focused `TestBusinessObservabilityExcludesAuthoredValuesAndUsesBoundedAttributes`, `TestBusinessSecurityAcceptance`, Tap NSID registry, PDS operation registry, and Sentry import-boundary tests passed.

### Step 66: IT-021
- Test setup: Installed fail-fast counting replacements for `http.DefaultTransport` and `net.DefaultResolver` for the duration of the focused test, restoring both with test cleanup.
- Green: First-party validation and independent profile/event hydration processed valid and malicious action, product, event, registration, and mailto destinations with exactly zero HTTP transport calls and zero DNS resolver calls.
- Run command: `go test ./internal/business -run TestBusinessDestinationProcessingDoesNotFetchOrResolve -count=1` passed.

### Manual Check M2: MAN-002
- Review timing: Performed only after IT-013, AT-014, IT-021, all focused reruns, and the complete affected observability/API/business package suites passed.
- Reviewed artifacts: Captured JSON slog output, in-memory metric calls, Sentry transaction/span events, `SanitizeEventContext` output, Tap NSID labels, and PDS operation mappings exercised by the automated canary tests.
- Outcome: Pass. No authored destination URI, email, free text, title, price, or location canary appears in logs, metric attributes, or trace attributes/events. No full DID or AT-URI appears. Metrics expose only bounded technical labels and every captured call passes `ValidateMetricCall`; unsupported identifiers and operations retain bounded fallbacks.
- Network review: Destination validators and hydrators remain syntax-only and have no HTTP client or resolver dependency; IT-021 independently proves the process defaults are not consulted.
- Affected suites: `go test ./internal/observability -count=1`, `go test ./internal/api -count=1`, and `go test ./internal/business -count=1` passed.

### Manual Check M1: MAN-001
- Review timing: Performed after lexicon contract tests, deterministic `just lexgen-check`, catalog digest checks, and the release-equivalent gate passed.
- Durable schema review: ADR 010, `business.defs`, the `literal:self` declaration, and the TID-keyed event match the approved required/optional fields, open `knownValues`, schema ceilings, string/blob bounds, shared refs, and forbidden commerce/location fields. The actor-profile digest guard confirms this is additive rather than a membership-schema change.
- External dependency review: The complete Q17 value is stored at the CID-named local path; its AT-URI, CID, resolved PDS, and retrieval method are explicit. Contract tests losslessly recompute the canonical DAG-CBOR CID. `cmd/lexgen/build.json` plus `just lexgen` resolve the address NSID only from the checked-in file.
- Catalog review: ISO 3166 and SIX ISO 4217 snapshots have source URLs, retrieval timestamps, content metadata, SHA-256 sidecars, and matching metadata digests. The offline generator verifies the digests and deterministically emits assigned countries and active numeric-scale currencies only.
- Evolution review: All declaration fields remain optional; existing required fields, keys, types, and maxima may not be tightened; recognized open values and new optional fields are additive; incompatible evolution requires a versioned NSID.
- Outcome: Pass. Schemas, external input, catalogs, generated types, and evolution policy are explicit, cryptographically checked, offline-reproducible, and approved for handoff.

### Implementation Review Correction Pass
- IR-001: Added missing module-content checksums through `go mod tidy`; `go mod tidy -diff` is empty. Reworked `lexgen-check` to snapshot current generated outputs before regeneration, making drift detection valid both before commit and in CI. The release gate now serializes Go packages to avoid disposable PostgreSQL shared-lock exhaustion without reducing within-package or race coverage.
- IR-002: Projection now enforces `self`/TID keys and validates business records against the embedded reviewed lexicons before generated-type decoding. Invalid required fields, bounds, and nested values are permanently rejected.
- IR-003: Product hydration reuses the independent image safety boundary and omits malformed, unsupported, oversized, invalid-alt, and invalid-aspect images without dropping products.
- IR-004: Event cursors are HMAC-SHA256 authenticated, kind/profile scoped, and wired through production dependencies/routes. Tampered and cross-context cursors receive the standard invalid-cursor response.
- IR-005: PDS record reads support raw JSON and declaration replacement preserves nested integers beyond exact `float64` range.
- IR-006: Production profile, search, relationship, post/reply/quote, and notification paths now have visible/blocked account-type assertions. Real SQL tracing proves one set-based `ANY($1)` lookup for both 1 and 50 DIDs.
- IR-007: The lifecycle test now drives production owner transitions and Craftsky membership projection through active, departed with no membership row, and active rejoin; retained business rows are suppressed then restored.
- IR-008: A real ingestion-service test first failed because business NSIDs were unsupported and departed-gated. Durable source/reconciliation policy now treats both collections as membership-independent while retaining deletion/terminal fences; both ingest before membership with pending, dependency-free projection jobs.
- IR-009: Missing evidence rows were reconciled below, MAN-001 completed, migration 000063 documented, and completion checks closed.
- Migration deviation: The coding plan grouped event moderation with migration 000062. Implementation uses 000062 for new projection/tombstone storage and 000063 for changes to existing moderation constraints. This keeps creation of new storage separate from compatibility changes to existing tables and gives the latter an isolated rollback; it does not alter the planned ordering or behavior.
- IR-010: Not required for approval. Existing release migration up/down/up checks, ordered permanent-deletion acceptance coverage, collection registry guards, and lifecycle retry suites cover the identified risk; no additional feature behavior was added.
- Final release evidence: `just appview-check` passed all module, generation, catalog, formatting, vet/static analysis, required real-Postgres/MinIO, race, migration, image, vulnerability-policy, startup, and media-memory gates. Artifact: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.2TSYLmBHFh`.

### Repeat Review Correction Plan
- IR-011 / REG-002 / IT-011 (`FR-020`, `NFR-004`, `AC-031`): First add a production branch-replies test where a visible reply points to a blocked business parent outside the returned page. Confirm the flattened `replyingTo` leak, then omit that alternate summary for either block direction.
- IR-011 / REG-002 / IT-011 (`FR-020`, `NFR-004`, `AC-031`): Add the equivalent focused-comment nested-reply test. Confirm the focused path leaks before applying the same relationship shaping to its nested reply page.
- IR-012 / IT-012 (`NFR-003`, `AC-042`): Obtain a valid management cursor through the registered application route, modify its authenticated value, and require the routed device-authenticated request to return the standard `400 invalid_cursor` camelCase envelope with `requestId`.
- Verification: Run each smallest focused test red then green, the neighboring post/routes packages, `just test`, `just appview-check`, `go vet ./...`, `go mod tidy -diff`, `docker compose config --quiet`, and `git diff --check`.

### Repeat Review Correction Execution
- IR-011 branch red: `TestListCommentReplies_OmitsBlockedParentOutsidePage` failed for both reciprocal block directions because the visible child's flattened parent serialized with `accountType: business`.
- IR-011 branch green: Separately loaded parent rows now participate in the same deduplicated relationship lookup. `shapeReplyItems` removes only `replyingTo` when that parent has either block direction, preserving the visible child and its own authoritative account type.
- IR-011 focused red: `TestGetPostComments_OmitsBlockedParentOutsideFocusedPage` reproduced the same `replyingTo.accountType` leak for both block directions in a focused nested page.
- IR-011 focused green: Focused nested reply pages now pass through the existing reply relationship shaper after construction. Blocked parent metadata is omitted while ordinary visible flattened-parent tests remain unchanged.
- IR-012 evidence: `TestBusinessRouteRejectsTamperedValidCursor` passed on its first run because HMAC rejection was already implemented. The test creates 21 owner events, obtains a valid cursor from the registered device-authenticated `GET /v1/events` route, changes its signature, and verifies routed `400 invalid_cursor` with `error`, `message`, and `requestId`.
- Focused commands: Both new red tests failed for the intended privacy behavior before implementation; both then passed with neighboring visible-parent regressions. `go test ./internal/api ./internal/routes -count=1` passed, static diagnostics were clean, and `git diff --check` passed.
- Broad commands: `just test`, `go vet ./...`, `go mod tidy -diff`, and `docker compose config --quiet` passed.
- Final release evidence: `just appview-check` passed all generation, catalog, module, formatting, static-analysis, required real-Postgres/MinIO, race, migration, image, vulnerability-policy, startup, and media-memory gates. The existing approved `GO-2026-5932` exception is unchanged. Artifact: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.kTQTz6uzOw`.

## Execution Log

| Test ID | Red evidence | Green evidence | Refactor / notes |
|---|---|---|---|
| UT-001 | Missing approved domain symbols caused focused build failure. | `go test ./internal/business -run TestAccountType -count=1` | Minimal exact parser and nil resolver; no storage/API behavior. |
| IT-001 | Missing migration pair, then missing store constructor. | Real-Postgres focused business and migration tests with `TEST_DATABASE_REQUIRED=true`. | No membership FK/cascade; no PDS behavior. |
| UT-002 | Missing business-type catalog API. | Focused catalog suite passed. | Exact RULE-007 order; first-party max 5. |
| UT-003 | Missing offering catalog API. | Focused catalog suite passed. | Exact RULE-008 order; first-party max 10. |
| UT-004 | Missing open-value policy API. | Focused catalog suite passed. | Unknown independent values remain data, not authority. |
| UT-005 | Missing field-specific grapheme/byte validator. | `go test ./internal/business -run TestBusinessTextValidation -count=1` | Uses extended grapheme clusters, not rune counts. |
| UT-006 | Missing assigned-country normalizer and locality bounds. | Focused location tests passed. | Generated only from 249-row official OBP snapshot. |
| UT-007 | Missing independent location hydration. | Focused location hydration test passed. | Invalid country omits location; unsafe locality omits only locality. |
| UT-008 | Missing action catalog/cardinality policy. | Focused action test passed. | Exact RULE-009 values; at most one action. |
| UT-009 | Missing web/mail destination grammar. | Focused destination test passed. | Pure parsing; no DNS or network access. |
| UT-010 | Missing product/image validator. | Focused product test passed. | Existing image media/size/alt/aspect contract reused. |
| UT-011 | Missing product collection policy. | Focused product collection test passed. | Authored order retained; exact URI duplicate rejection. |
| UT-012 | Missing pinned currency scales and amount validator. | Focused money test passed. | SIX source, metadata, digest, and generated table checked in. |
| UT-013 | Missing safe independent price hydration. | Focused money hydration test passed. | Unsafe price omitted without dropping product. |
| UT-014 | Missing event catalogs/defaults. | Focused event validation test passed. | Independent omissions hydrate safely. |
| UT-015 | Missing canonical instant/range validation. | Focused event time test passed. | UTC `Z`, whole seconds, end after start. |
| UT-016 | Missing all-day timezone/DST policy. | Focused event time test passed. | Local midnight with exclusive end. |
| UT-017 | Missing event duration/new-event policy. | Focused temporal policy test passed. | Maximum 31 days; already-ended creation rejected. |
| UT-018 | Missing event destination/image policy. | Focused event media test passed. | Duplicate links rejected on writes and deduplicated on hydration. |
| UT-019 | Missing centralized event eligibility. | `go test ./internal/business -run TestEventEligibility -count=1` | Canonical distinct diagnostics; no declaration dependency. |
| UT-020 | Missing extension merge/safe profile hydration. | `go test ./internal/business -run TestProfileReplacementMergeAndSafeHydration -count=1` | Unknown top-level source fields retained. |
| UT-021 | Missing event cursor/limit helpers. | `go test ./internal/api -run TestBusinessEventCursorAndLimits -count=1` | Upcoming cursor freezes `asOf`; wrong-kind cursor rejected. |
| UT-022 | Missing CID precondition/conflict helpers. | `go test ./internal/api -run TestBusinessRecordIfMatch -count=1` | Missing, malformed, quoted, and stale values conflict. |
| UT-023 | Missing server-owned timestamp policy. | `go test ./internal/business -run TestEventCreatedAtIsServerOwned -count=1` | Presence is rejected even when value equals stored timestamp. |
| IT-016 | Missing schemas, offline external mapping, generated types, and complete country provenance. | Business lexicon and catalog provenance suites passed. | CID, schema maxima, digest, generated output, and forbidden-field contracts fixed. |
| AT-013 | Lexgen initially failed on unresolved external ref and then CBOR bootstrap. | Repeated local regeneration produced identical file hashes; catalog check passed twice. | No network access in generation. |
| REG-001 | No business-schema separation guard. | Actor profile digest and all-optional separate declaration assertions passed. | Existing membership record remains unchanged. |
| IT-004 | Declaration projector and eligible store read were absent. | Both focused real-Postgres declaration projection/hydration tests passed. | Read-time suppression never deletes independent source. |
| IT-005 | Event projector was absent. | R1-R4 real-Postgres convergence test passed. | Equal/older revisions are no-ops across rows and tombstones. |
| IT-006 | Business records were unsupported, membership-blocked, and unregistered. | Dispatcher policy/unit and production wiring integration tests passed. | Tap Compose filters now include both business NSIDs. |
| AT-009 | Cross-cutting acceptance target was absent. | Real-Postgres dispatcher convergence and safe-hydration scenario passed. | Lower-level IT-004 through IT-006 made the acceptance test green immediately. |
| IT-002 | Account-type HTTP mutation handler was absent. | Auth/current-member/ownership real-store integration test passed. | Private write is generation-fenced and has no PDS dependency. |
| AT-001 | Account-type response hydration and batch lookup were absent. | Real-store classification acceptance scenario passed. | One lookup decorates duplicate visible identities; missing state defaults regular. |
| IT-003 | Declaration mutation handlers and absent-record swap behavior were absent. | Conditional create/replace/delete effect tests and PDS boundary tests passed. | Unknown top-level source extensions survive known-field replacement. |
| AT-003 | Full profile response lacked declaration presentation. | Real-Postgres profile presentation acceptance test passed. | Blocked shells do not query or expose business state; events remain separate. |
| IT-007 | Event mutation handlers were absent. | Focused POST/PUT/DELETE PDS-effect tests passed. | TID identity, server timestamp, media validation, and CID swaps are enforced. |
| REG-004 | Event PDS ownership/CID paths lacked regression coverage. | Cross-owner and stale-precondition tests passed. | Rejected paths do not construct or invoke unintended effects. |
| IT-008 | Event hydration and caller-aware store reads were absent. | Real-Postgres eligibility matrix passed. | One domain policy owns visitor/owner diagnostics and suppression. |
| AT-002 | Declaration-independent classification/event behavior lacked cross-layer evidence. | Real-Postgres acceptance scenario passed. | Type, not declaration, controls public business serving. |
| AT-004 | Regular-owner preparation and ineligible caller behavior lacked cross-layer evidence. | Mutation/serving acceptance scenario passed. | Regular owners may prepare public PDS source without activating serving. |
| IT-009 | Focused build failed on the absent public profile-events handler. | Required real-Postgres pagination test, focused race run, neighboring business-event tests, and full API package passed. | Cursor freezes only time eligibility; concurrent record changes retain strict seek, not snapshot, semantics. |
| IT-010 | Event management/report/moderation integration was absent. | Management pagination and moderation suppress/restore suites passed in the full API and release gates. | Reports retain exact record subjects and lifecycle diagnostics. |
| AT-005 | Cross-layer event lifecycle behavior was absent. | Event management acceptance coverage passed through create, update, direct/list reads, suppression, moderation, and delete. | Server-owned timestamps and CID preconditions remain enforced. |
| AT-006 | Public pagination acceptance behavior was absent. | Real-Postgres traversal, frozen-time, tamper, and concurrent-mutation scenarios passed. | Seek semantics are documented rather than represented as snapshot isolation. |
| AT-007 | Management diagnostics and moderation acceptance behavior was absent. | Event management and moderation API acceptance suites passed. | Canonical reason arrays are owner-only; visitor suppression remains not-found/empty. |
| IT-011 | Production summary and bounded-query evidence was incomplete. | Actual profile/search/relationship/post/reply/quote/notification tests plus one-query SQL tracing for 1 and 50 DIDs passed. | Blocked/reduced shapes omit both account type and business data. |
| AT-008 | Cross-surface summary hydration was synthetic only. | Production response fixtures passed for regular, business, missing, and blocked identities. | No surface performs declaration/event hydration implicitly. |
| REG-002 | Blocked shapes lacked business-field assertions. | Existing response-shape tests now assert `accountType` and `business` omission in every reduced shape. | Redaction occurs before business reads where applicable. |
| REG-003 | Full profile collection separation lacked recorded evidence. | `TestBusinessDeclarationPresentation` passed and asserts no `events` key. | Events remain exclusively on the paginated endpoint. |
| IT-012 | Business route registration evidence was omitted. | App/router route inventory and exact/near-miss business route suites passed. | Device auth, request IDs, camelCase errors, and exact methods remain enforced. |
| AT-011 | Route/CID cross-layer evidence was omitted. | Declaration/event exact-route, conditional-create, canonical-CID conflict, and near-miss acceptance suites passed. | Successful responses use camelCase CID contracts; rejected requests do not dispatch effects. |
| IT-014 | Lifecycle coverage bypassed production departure/rejoin. | Real-Postgres production transition/projector test passed for active to departed/absent to active. | Account type, declaration, and events are retained; serving alone is suppressed. |
| AT-010 | Ordinary/permanent lifecycle acceptance evidence was omitted. | Retention/rejoin and ordered permanent-deletion acceptance suites passed. | Ordinary departure never deletes independent business state. |
| IT-015 | Permanent deletion evidence was omitted. | Ordered event, declaration, account-type, membership cleanup and replay/absence tests passed. | Namespace, blob, DID, and PDS-account boundaries remain closed. |
| REG-007 | Deletion registry evidence was omitted. | Collection registry and permanent deletion boundary suites passed. | Only registered `social.craftsky.*` records are eligible for deletion. |
| IT-013 | Business Tap labels became fallback values and business PDS operations became `unknown`. | Focused business privacy test and affected observability/business suites passed. | Added only closed business NSID and profile/event get/put/delete labels; canaries, full DID, and full AT-URI are absent and every metric call validates. |
| AT-014 | Cross-layer destination and telemetry acceptance evidence was absent. | `go test ./internal/api -run TestBusinessSecurityAcceptance -count=1` and full API suite passed. | Actual mutation handlers accept/reject representative Q14 forms before effects and captured logs/metrics/traces remain bounded. |
| IT-021 | No fail-fast proof guarded against hidden process-default networking. | `go test ./internal/business -run TestBusinessDestinationProcessingDoesNotFetchOrResolve -count=1` passed with HTTP/DNS counts 0/0. | Test restores both process defaults; production validators/hydrators remain dependency-free. |
| IT-017 | Feed-neutrality integration evidence was omitted. | Chronological feed fixtures passed before/after business-state changes in focused and release gates. | No business join affects ordering or inclusion. |
| IT-018 | Search-neutrality integration evidence was omitted. | Real search fixtures passed before/after type/taxonomy changes. | Search has no business boost or filter. |
| IT-019 | Permission/moderation neutrality evidence was omitted. | Policy-neutrality matrices passed for regular and business actors. | Classification grants no entitlement or moderation priority. |
| AT-012 | Cross-cutting non-entitlement evidence was omitted. | Feed, search, policy, and business non-entitlement acceptance suites passed. | Business state changes presentation only. |
| REG-005 | Ranking neutrality evidence was omitted. | Feed/search ordered-identity regression fixtures passed. | Chronological/relevance behavior is unchanged. |
| REG-006 | Authorization neutrality evidence was omitted. | Existing permission matrices plus business policy regressions passed. | Business classification is not an authorization input. |
| IT-020 | Price wire-contract evidence was omitted. | Product contract tests passed against schemas, generated records, and API output. | Price exposes authored amount/currency only. |
| REG-008 | Commerce-semantics regression evidence was omitted. | Contract scans reject disclaimer, inventory, availability, synchronization, checkout, shipping, tax, and accuracy semantics. | No native-commerce claim was introduced. |
| MAN-002 | Pending until automated security evidence passed. | Manual review completed after all focused and affected suites passed. | Captured telemetry contains no prohibited authored value, full DID, or AT-URI; bounded metric calls validate. |
| MAN-001 | Pending until schema/provenance implementation and gates passed. | Manual ADR/schema/external-CID/catalog/generated-output review completed after deterministic and release checks. | Durable evolution is additive; all inputs are pinned and offline-reproducible. |

## Package-Backed Catalog Simplification

After the approved implementation was checkpointed in commit `1bf5e3a0`, the user deliberately loosened FR-006, FR-010, and NFR-001 to reduce catalog maintenance.

- Red: package-policy tests failed because the generated snapshot catalogs rejected CLDR exceptional region behavior and the recognized `BGN` currency.
- Green: country validation now uses `golang.org/x/text/language.ParseRegion`, `Region.IsCountry`, and private-use rejection. Tests cover ordinary countries, the CLDR exceptional `AC` region, unknown/private regions, canonical casing, and safe independent hydration.
- Green: currency validation now uses `golang.org/x/text/currency.ParseISO` and `currency.Standard.Rounding`. Uppercase package-recognized codes are accepted, including `BGN` and `XAU`; `XXX`, `XTS`, lowercase, and unknown codes remain rejected.
- Removed: the `businesscataloggen` command, dated ISO/SIX source snapshots and sidecars, generated country/currency maps, provenance test, generated-file attributes, Just recipes, and release-gate drift check.
- Documentation: requirements, acceptance tests, coding plan, and ADR 010 now make the pinned module version the maintenance boundary and explicitly accept broader CLDR/ISO package semantics.
- Verification: focused business/API tests, `go test ./internal/business`, `go test ./internal/lexicon/craftsky`, `just test`, `go vet ./...`, `go mod tidy -diff`, `docker compose config --quiet`, `git diff --check`, and `just appview-check` passed. Release artifact: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.yUEXehEmwm`.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] ADR, lexicons, external CID, package-backed catalog policy, and generated lexicon artifacts reviewed
- [x] Telemetry configuration manually reviewed
- [x] Docs updated and this file read back
- [x] Implementation review completed or explicitly skipped

## Stage Notes
- Commits are disabled unless the user explicitly requests one.
- No blocking contract gaps were found at startup.
- The worktree was clean at startup.
- Historical note: execution originally deferred UT-006 and UT-007 to the catalog-provenance slice. The later package-backed simplification above supersedes that source-snapshot requirement.
- ADR 010, the exact CID-pinned external address record, reviewed business lexicons, and generated packages are present.
- Historical note: the user completed the official OBP challenge in headed Chromium on 2026-08-29. Those captured catalog artifacts were subsequently removed by the package-backed simplification.
- On 2026-08-29 the user resolved the declaration-create precondition ambiguity: `If-Match: *` means conditional creation only when the singleton is absent; existing-record mutation continues to require its canonical CID.
- AT-002 is deferred from step 35 until after IT-008 because its approved scenario requires declaration-independent event serving and regular-account suppression; implementing only its profile half would not satisfy the test ID.
- AT-004 and AT-011 are deferred until event CRUD/routes exist because both approved scenarios include event behavior in addition to declaration behavior.
- IT-013, AT-014, IT-021, and MAN-002 were executed sequentially on 2026-08-29. MAN-002 was not recorded until all focused and affected automated checks were green.
