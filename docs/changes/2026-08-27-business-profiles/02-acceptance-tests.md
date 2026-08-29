# Acceptance Test Specification: Business Profiles

## 1. Test Strategy

This high-risk change is tested primarily in the Go AppView. Table-driven unit tests cover closed catalogs, text and URI bounds, money, event time rules, safe hydration, and eligibility. Real-Postgres integration tests cover migrations, account-type persistence, indexer replay/delete behavior, raw-source preservation, query ordering, membership lifecycle, moderation, and permanent deletion. HTTP acceptance tests use the repository's existing `httptest` style with fake PDS and identity boundaries, plus focused real-store tests where transaction or query behavior matters.

Lexicon contract tests validate record keys, required/optional fields, bounds, schema ceilings, generated types, and the pinned external address definition. A regeneration drift check is required. Regression tests protect ordinary profile ownership, block redaction, chronological ranking, permissions, and account deletion boundaries. Flutter tests are intentionally excluded because Flutter integration is outside this implementation slice.

The implementation should use a controllable clock and deterministic PDS/indexer fakes. No acceptance test may contact external product, action, event, registration, or email destinations. Manual checks are limited to ADR/schema review and telemetry inspection where a fully automated semantic assertion is not practical.

Risk level: **High**. Document review completed and the user explicitly approved this specification for implementation on 2026-08-28.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, AT-002, IT-001, IT-014 | Acceptance / Integration | Yes |
| BR-002 | AC-003 | AT-003, IT-004 | Acceptance / Integration | Yes |
| BR-003 | AC-004, AC-005 | AT-003, AT-005, IT-004 | Acceptance / Integration | Yes |
| BR-004 | AC-006 | AT-012, IT-017, IT-018, IT-019, REG-005, REG-006 | Acceptance / Integration / Regression | Yes |
| FR-001 | AC-001, AC-007 | AT-001, UT-001, IT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-001, AC-008 | AT-001, AT-004, IT-002 | Acceptance / Integration | Yes |
| FR-003 | AC-009 | UT-020, IT-003, IT-004, REG-001 | Unit / Integration / Regression | Yes |
| FR-004 | AC-010 | UT-002, UT-003, IT-004 | Unit / Integration | Yes |
| FR-005 | AC-003, AC-011 | AT-003, UT-005 | Acceptance / Unit | Yes |
| FR-006 | AC-012, AC-013 | AT-003, UT-006, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-014, AC-015 | AT-014, UT-008, UT-009, IT-021 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-004, AC-015, AC-016 | AT-003, AT-014, UT-005, UT-009, UT-010, IT-004, IT-021 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-017 | UT-011, IT-016 | Unit / Integration | Yes |
| FR-010 | AC-018 | UT-012, UT-013, IT-004 | Unit / Integration | Yes |
| FR-011 | AC-005, AC-019 | AT-005, UT-014, IT-005, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-020 | UT-005, UT-014, IT-008 | Unit / Integration | Yes |
| FR-013 | AC-021 | UT-015, UT-016, UT-019, IT-008 | Unit / Integration | Yes |
| FR-014 | AC-022, AC-023 | AT-007, UT-017, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-015, AC-024 | AT-014, UT-009, UT-018, IT-008, IT-021 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-025, AC-026 | AT-005, UT-019, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-002, AC-026, AC-027 | AT-002, AT-005, UT-019, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-028 | AT-006, UT-021, IT-009, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-019 | AC-023, AC-029 | AT-007, IT-010 | Acceptance / Integration | Yes |
| FR-020 | AC-030, AC-031 | AT-008, UT-019, IT-011, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-021 | AC-032, AC-033 | AT-009, IT-005, IT-006 | Acceptance / Integration | Yes |
| FR-022 | AC-034, AC-035 | AT-010, IT-014, IT-015, REG-007 | Acceptance / Integration / Regression | Yes |
| FR-023 | AC-036 | AT-011, UT-020, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-024 | AC-037 | AT-011, UT-022, IT-007, IT-012, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-025 | AC-038 | AT-005, UT-023, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-026 | AC-033, AC-039 | AT-009, UT-004, UT-020, IT-004, IT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-001, AC-002 | AT-001, AT-002, UT-019, IT-014 | Acceptance / Unit / Integration | Yes |
| RULE-002 | AC-008, AC-025 | AT-004, IT-002, IT-007, REG-004 | Acceptance / Integration / Regression | Yes |
| RULE-003 | AC-002, AC-027 | AT-002, UT-019, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-040 | AT-003, IT-020, REG-008 | Acceptance / Integration / Regression | Yes |
| RULE-005 | AC-013 | UT-007, IT-004 | Unit / Integration | Yes |
| RULE-006 | AC-034, AC-035 | AT-010, IT-014, IT-015, REG-007 | Acceptance / Integration / Regression | Yes |
| RULE-007 | AC-010 | UT-002 | Unit | Yes |
| RULE-008 | AC-010 | UT-003 | Unit | Yes |
| RULE-009 | AC-014 | UT-008 | Unit | Yes |
| RULE-010 | AC-006 | AT-012, UT-004, IT-017, IT-018, IT-019, REG-005, REG-006 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-011 | AC-019, AC-020 | UT-014 | Unit | Yes |
| NFR-001 | AC-041 | AT-013, IT-016, REG-001, MAN-001 | Acceptance / Integration / Regression / Manual | Partly |
| NFR-002 | AC-032, AC-033 | AT-009, IT-005, IT-006 | Acceptance / Integration | Yes |
| NFR-003 | AC-028, AC-037, AC-042 | AT-006, AT-011, UT-021, UT-022, IT-009, IT-012 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-029, AC-031 | AT-007, AT-008, IT-008, IT-010, IT-011, REG-002 | Acceptance / Integration / Regression | Yes |
| NFR-005 | AC-016, AC-024 | UT-010, UT-018, IT-004, IT-007 | Unit / Integration | Yes |
| NFR-006 | AC-043 | AT-014, IT-013, MAN-002 | Acceptance / Integration / Manual | Partly |
| NFR-007 | AC-015 | AT-014, UT-009, IT-021 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Select Account Type Without A Declaration
Requirement IDs: BR-001, FR-001, FR-002, RULE-001
Acceptance Criteria: AC-001, AC-007
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_account_type_acceptance_test.go`

```gherkin
Feature: Authoritative business classification
  Scenario: A current member changes classification without publishing a declaration
    Given a current member has no account-type row and no business declaration
    When the member reads their normally visible profile
    Then the profile reports accountType "regular"
    When the member puts accountType "business"
    Then the private account-type row stores "business"
    And every normally visible profile and author summary reports "business"
    And no PDS mutation is made
    When the member puts accountType "regular"
    Then the same row stores "regular"
    And normally visible summaries report "regular"
    When the member attempts to put accountType "pro"
    Then validation rejects the value before storage
```

### AT-002: Declaration Does Not Control Classification Or Event Eligibility
Requirement IDs: BR-001, FR-017, RULE-001, RULE-003
Acceptance Criteria: AC-002, AC-027
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_eligibility_acceptance_test.go`

```gherkin
Feature: Read-time business eligibility
  Scenario: A business member has no declaration
    Given a current member has accountType "business" and an eligible event
    And the member has never created a business declaration
    When a visitor reads the member profile and upcoming events
    Then the profile reports accountType "business"
    And the event is served
    And no declaration-backed fields are present

  Scenario: A declaration is deleted
    Given a current business member has a declaration and an eligible event
    When the owner deletes the declaration with its current CID
    Then declaration-backed details disappear
    And accountType remains "business"
    And the event remains publicly eligible

  Scenario: The owner changes to regular
    Given the event source record still exists
    When the owner changes accountType to "regular"
    Then public event reads are suppressed
    And the source record remains indexed
```

### AT-003: Present Business Details And Ordered Products
Requirement IDs: BR-002, BR-003, FR-005, FR-006, FR-008, RULE-004
Acceptance Criteria: AC-003, AC-004, AC-012, AC-040
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_profile_acceptance_test.go`

```gherkin
Feature: Business declaration presentation
  Scenario: An eligible declaration is hydrated without reinterpretation
    Given a current business member has bounded tagline, hours note, service area, lowercase country, locality, primary action, and four valid product cards
    When a visitor reads the profile
    Then the free text is returned unchanged
    And country is returned in uppercase
    And only country and locality are exposed from the address
    And products retain authored order
    And each product includes its external URI and image
    And supported optional alt text and price are returned
    And the response exposes only seller-authored amount and currency for price
    And no disclaimer, inventory, availability, synchronization, tax, shipping, checkout, or accuracy field or claim is present

  Scenario: First-party location input is constrained to the declaration shape
    Given a current business member submits an assigned mixed-case alpha-2 country and bounded locality
    When the declaration is written and read
    Then country is stored and returned uppercase
    And only country and locality are accepted and returned
    When the member submits an unassigned or non-alpha-2 country, oversized locality, or structured event location
    Then the write is rejected with a field-specific validation error
```

### AT-004: A Regular Member Can Prepare Only Their Own Business Records
Requirement IDs: FR-002, RULE-002
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_ownership_acceptance_test.go`

```gherkin
Feature: Business record ownership
  Scenario: A regular current member prepares business data
    Given an authenticated current member has accountType "regular"
    When the member creates or edits their own declaration and event
    Then each AppView-mediated PDS mutation succeeds
    And public business hydration remains suppressed while the type is regular

  Scenario Outline: An ineligible caller attempts a mutation
    Given the caller is <caller_state>
    When the caller attempts to mutate the member's business record
    Then the request is rejected before a PDS mutation occurs

    Examples:
      | caller_state |
      | unauthenticated |
      | not a current member |
      | authenticated as a different DID |
```

### AT-005: Create And Manage An Independently Addressable Event
Requirement IDs: BR-003, FR-011, FR-016, FR-017, FR-025
Acceptance Criteria: AC-005, AC-025, AC-026, AC-038
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_event_acceptance_test.go`

```gherkin
Feature: Business event lifecycle
  Scenario: An owner creates, updates, reads, and deletes an event
    Given an authenticated current member submits a valid scheduled future event without createdAt
    When AppView creates the event
    Then AppView server-stamps a canonical createdAt
    And the PDS record uses social.craftsky.business.event with a TID record key
    And the response exposes its DID, rkey, URI, and CID
    When the owner reads the event by DID and rkey
    Then the same independently addressable event is returned
    When the owner updates it with the current CID and without changing createdAt
    Then the stored createdAt is preserved
    And the new CID is returned
    When the owner deletes it with the new CID
    Then PDS and AppView projection converge on absence

  Scenario: A client attempts to author createdAt
    Given an authenticated current owner has an existing event with a stored createdAt
    When the owner supplies createdAt on create
    Then the create is rejected before a PDS mutation
    When the owner supplies either the same or a different createdAt on update
    Then the update is rejected before a PDS mutation
    And the stored createdAt remains unchanged

  Scenario: Lifecycle state controls the upcoming list
    Given otherwise eligible scheduled, ongoing, ended, cancelled, and postponed events exist
    When a visitor lists upcoming events
    Then only scheduled future and ongoing events are returned
    And ended, cancelled, and postponed events remain directly readable
    And past or completed state is derived from endsAt

  Scenario: The current owner directly reads a suppressed event
    Given the current owner has a retained over-duration or moderated event
    When the owner gets /v1/events/{did}/{rkey}
    Then the management event view and diagnostics are returned
    When another authenticated visitor requests the same event
    Then the response is 404 event_not_found
```

### AT-006: Paginate Upcoming Events With Frozen Time Eligibility
Requirement IDs: FR-018, NFR-003
Acceptance Criteria: AC-028
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_event_pagination_test.go`

```gherkin
Feature: Upcoming event pagination
  Scenario: Traverse an unchanged event set
    Given more than ten eligible events include equal startsAt values
    When a visitor traverses GET /v1/profiles/{handleOrDid}/events using returned cursors
    Then the default page size is ten
    And events are ordered by startsAt ascending and URI ascending
    And traversal has no duplicates or omissions
    And the cursor is opaque
    And every later page uses the first page asOf for time eligibility
    And requesting more than fifty is capped at fifty
    And the main profile response contains no events collection
```

### AT-007: Diagnose, Report, And Moderate An Event
Requirement IDs: FR-014, FR-019, NFR-004
Acceptance Criteria: AC-023, AC-029
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_event_management_acceptance_test.go`

```gherkin
Feature: Event management and moderation
  Scenario: Owner sees lifecycle and suppression diagnostics
    Given the owner has future, past, cancelled, postponed, invalid-time-range, over-duration, and moderated events
    When the owner traverses GET /v1/events
    Then the default page contains at most 20 events and a request is capped at 50
    And events are ordered by startsAt descending and URI descending
    And every event is returned across the opaque cursor traversal
    And each item has non-null distinct canonical-order publicSuppressionReasons and upcomingExclusionReasons arrays
    And the arrays contain only the closed Q13 reason codes
    And the over-duration event remains raw-indexed but is absent from visitor direct and upcoming reads

  Scenario: A user reports an event and moderation suppresses it
    Given an authenticated user can see an individual event
    When the user reports that event through POST /v1/events/{did}/{rkey}/reports
    And record moderation applies a hide outcome
    Then the report identifies the event URI
    And the event is suppressed from public serving
    And owner diagnostics reflect a bounded moderation reason
```

### AT-008: Include Account Type Without Leaking Through Blocked Shapes
Requirement IDs: FR-020, NFR-004
Acceptance Criteria: AC-030, AC-031
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_summary_redaction_acceptance_test.go`

```gherkin
Feature: Profile and author summary classification
  Scenario: Normally visible summaries include authoritative account type
    Given regular and business members exist and one business member has no declaration
    When profiles, search results, relationship lists, and embedded authors are hydrated
    Then each normally visible shape includes accountType "regular" or "business"
    And summary list queries do not issue one account-type query per row

  Scenario: A blocked profile is reduced to a shell
    Given the viewer is blocked by a business member
    When any profile or summary shape for that member is returned
    Then accountType is omitted
    And declaration, location, action, and products are omitted
    When the viewer lists that member's events
    Then the event list is empty
    When the viewer directly reads one of that member's events
    Then the response is 404 event_not_found
```

### AT-009: Preserve Federated Source While Hydrating Safely
Requirement IDs: FR-021, FR-026, NFR-002
Acceptance Criteria: AC-032, AC-033
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/business_projection_acceptance_test.go`

```gherkin
Feature: Independent business record projection
  Scenario: Records arrive before local dependencies
    Given a declaration and event arrive before membership, account type, or each other
    When a create at revision R1 is projected twice with the same URI and CID
    And an update with a distinct CID at revision R2 is projected
    And the older R1 create is delivered again
    Then raw records and unknown top-level fields from R2 are retained once
    And projection is idempotent
    And no dependency retry is required
    And unsafe subordinate destinations or prices are omitted only from hydration
    And unknown lexicon-valid values are represented safely
    When a delete at revision R3 arrives
    Then the matching projection converges on absence and retains an R3 tombstone
    When an older create or update is delivered after the delete
    Then it is ignored and cannot resurrect the record
```

### AT-010: Distinguish Departure From Permanent Deletion
Requirement IDs: FR-022, RULE-006
Acceptance Criteria: AC-034, AC-035
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/accountdeletion/business_records_acceptance_test.go`

```gherkin
Feature: Business data lifecycle
  Scenario: A member leaves and rejoins Craftsky
    Given a business member has an account-type row, declaration, and events
    When ordinary membership departure completes
    Then business serving is suppressed
    And account type, declaration, and events remain stored
    When the same DID rejoins
    Then the persisted business account type and recalculated event eligibility resume

  Scenario: An owner permanently deletes their Craftsky account
    Given retained business records exist even if membership is absent
    When the approved permanent deletion worker completes
    Then owned event PDS records are deleted before the declaration
    And the declaration is deleted before the private account-type row
    And the private account-type row is removed before membership cleanup
    And no other namespace, DID, PDS account, or blob is directly deleted
    When the worker is interrupted after any completed stage and retries
    Then completed stages are safely replayed or skipped
    And the same four-stage order is preserved through completion
```

### AT-011: Enforce Exact Routes, CIDs, And Error Contracts
Requirement IDs: FR-023, FR-024, NFR-003
Acceptance Criteria: AC-036, AC-037
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_routes_acceptance_test.go`

```gherkin
Feature: Business API contracts
  Scenario: Exact mutation routes use camelCase contracts
    Given an authenticated current owner
    When the owner uses PUT /v1/profiles/me/account-type, PUT or DELETE /v1/profiles/me/business, and POST or item routes under /v1/events
    Then only the specified methods and paths are routed
    And successful record responses expose cid
    And JSON keys use camelCase

  Scenario: Declaration replacement retains independent extensions
    Given the current declaration has known fields, unknown top-level extensions, and a current CID
    When the owner puts replacement known fields with that CID
    Then omitted known fields are cleared
    And supplied known fields replace their prior values
    And unknown top-level source extensions are preserved
    And deleting the declaration leaves membership and account type intact

  Scenario: A record update is not based on the current PDS version
    Given a business or event record has a current CID
    When PUT or DELETE omits If-Match or supplies a stale CID
    Then the response is 409 pds_record_conflict
    And the error envelope contains error, message, and requestId
    And no lost update occurs
```

### AT-012: Business State Has No Product Or Permission Advantage
Requirement IDs: BR-004, RULE-010
Acceptance Criteria: AC-006
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_non_entitlement_acceptance_test.go`

```gherkin
Feature: Self-declared business state
  Scenario: Equivalent regular and business members use existing product surfaces
    Given otherwise equivalent regular and business members
    When feed order, search results, relationship permissions, moderation priority, and record mutation authorization are evaluated
    Then account type, taxonomy, products, and events do not alter order, filtering, reach, moderation priority, or permission
    And no subscription, verification, endorsement, or paid-reach field is exposed
```

### AT-013: Generate Stable Lexicon Contracts
Requirement IDs: NFR-001
Acceptance Criteria: AC-041
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/lexicon/craftsky/business_contract_test.go` and `just lexgen-check`

```gherkin
Feature: Reproducible business lexicons
  Scenario: Generate code from reviewed schemas
    Given the ADR-approved business profile and event schemas
    And the address snapshot is vendored under the Q17 CID from the exact Q17 AT-URI
    And country and currency source snapshots have checked-in retrieval metadata and SHA-256 sidecars
    When lexicon generation runs twice offline from a clean generated baseline
    Then generated Go and CBOR output is stable
    And record keys, required and optional fields, catalogs, bounds, and schema maxima match the reviewed contract
    And the address DAG-CBOR CID and both catalog digests are recomputed and match
    And generated country and currency tables match their pinned snapshots
    And social.craftsky.actor.profile remains unchanged
```

### AT-014: Do Not Fetch Or Expose Authored Values In Telemetry
Requirement IDs: FR-007, FR-008, FR-015, NFR-006, NFR-007
Acceptance Criteria: AC-015, AC-043
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/business_security_acceptance_test.go`

```gherkin
Feature: Business destination and telemetry safety
  Scenario: Process authored public business data
    Given records contain unique canary values in destinations, email, notes, titles, prices, and locations
    And web destinations include valid credential-free HTTPS plus HTTP, custom-scheme, credentialed, hostless, and oversized values
    And email destinations include one valid lowercase ASCII dot-atom mailto plus uppercase-scheme, whitespace, control, percent-encoded, queried, fragmented, comma, semicolon, and oversized values
    When AppView validates, projects, hydrates, lists, and rejects those records
    Then only destinations satisfying the exact Q14 grammar are accepted on Craftsky writes
    And no outbound network request or DNS resolution is made for any authored destination
    And logs and trace attributes contain none of the canary values
    And metric labels contain only bounded operation, result, and reason values
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001 | AC-007 | Validate account type and missing-state default. | Missing row; `regular`; `business`; `pro`; empty; mixed case. | Missing maps to `regular`; exact supported values pass; all others fail before storage. | `appview/internal/business/account_type_test.go` |
| UT-002 | FR-004, RULE-007 | AC-010 | Validate business-type catalog, uniqueness, count, and canonical response order. | Every catalog value; unknown/mixed-case; duplicates; 5 and 6 values; independently sourced 20 values. | First-party exact values through 5 pass and sort by catalog; invalid/duplicate/6 fail; safe independent values are preserved. | `appview/internal/business/catalog_test.go` |
| UT-003 | FR-004, RULE-008 | AC-010 | Validate offering catalog, uniqueness, count, and canonical response order. | Every catalog value; unknown; duplicates; 10 and 11 values; independently sourced 20 values. | First-party exact values through 10 pass and sort by catalog; invalid/duplicate/11 fail; safe independent values are preserved. | `appview/internal/business/catalog_test.go` |
| UT-004 | FR-026, RULE-010 | AC-006, AC-039 | Classify known and unknown open values without granting behavior. | Known and unknown taxonomy/action/role/mode/status values from first-party and independent paths. | First-party unknowns fail; independent unknowns remain representable and never activate permissions, ranking, or field dependencies. | `appview/internal/business/catalog_test.go` |
| UT-005 | FR-005, FR-008, FR-012 | AC-003, AC-011, AC-016, AC-020 | Enforce grapheme and byte bounds for all business/event text fields. | Boundary and over-bound ASCII, multibyte, combining-mark, and emoji strings for tagline, hoursNote, serviceArea, product title, event name, summary, and venue. | Exact boundaries pass; either exceeded dimension produces a field-specific validation error. | `appview/internal/business/text_validation_test.go` |
| UT-006 | FR-006 | AC-012 | Validate and canonicalize first-party declaration location. | Assigned lower/mixed/upper alpha-2 country; unassigned/non-alpha-2 values; locality at and above both bounds; extra mutation fields; event request with structured location. | Assigned country becomes uppercase; unassigned/invalid country or locality fails; declaration extras and any structured event location are rejected by request shape. | `appview/internal/business/location_test.go` |
| UT-007 | FR-006, RULE-005 | AC-013 | Safely hydrate independent broad declaration addresses. | Assigned country with street/region/postal code and oversized locality; unassigned/non-alpha-2 country with otherwise valid declaration. | Raw input remains intact; hydration exposes country only in first case and omits location in second without dropping the declaration. | `appview/internal/business/location_test.go` |
| UT-008 | FR-007, RULE-009 | AC-014 | Validate the exact action catalog and single-action shape. | Every recognized type; unknown/mixed-case type; zero, one, and two actions. | Zero/one recognized action passes; unknown or second action fails. | `appview/internal/business/action_test.go` |
| UT-009 | FR-007, FR-008, FR-015, NFR-007 | AC-015 | Validate the Q14 common web-host/port rule and exact mailto grammar. | HTTPS with query/fragment and 2048/2049 bytes; userinfo; empty/single-label/trailing-dot/Unicode/IP/punycode/percent-encoded hosts; valid/invalid DNS labels; absent, boundary, and invalid ports; HTTP/custom scheme; lowercase/uppercase mailto; ASCII dot-atom/DNS address at 320/321 bytes; whitespace, control, percent encoding, query, fragment, comma, semicolon. | Only values satisfying every Q14 parser, DNS, port, and mail rule pass; validation performs no DNS resolution or fetch. | `appview/internal/business/destination_test.go` |
| UT-010 | FR-008, NFR-005 | AC-016 | Validate first-party and independent product card image rules. | Title+URI with no image; JPEG/PNG/WebP at/over 15 MiB; unsupported MIME; alt at/over both bounds; independent no-image card. | First-party requires valid image; independent minimum remains valid/indexable; image and alt boundaries match existing contract. | `appview/internal/business/product_test.go` |
| UT-011 | FR-009 | AC-017 | Validate product count, order, and exact duplicate URI handling. | Four/five cards; exact duplicate URI; case/path/query variants. | Four pass in authored order; five and exact duplicates fail; merely distinct strings remain distinct. | `appview/internal/business/product_test.go` |
| UT-012 | FR-010 | AC-018 | Validate the generated active currency catalog and canonical variable-scale decimals. | Verified SIX snapshot/digest; representative active, withdrawn, and `N.A.` entries; `0`; USD `1`, `1.2`, `1.23`, `1.20`; JPY fraction; KWD and four-scale boundaries; 12/13 integer digits; leading zero; signs, exponent, separators. | Generated entries/scales match the pinned source; only active uppercase numeric-scale codes with canonical grammar pass; fractional trailing zeros and noncanonical zero fail. | `appview/internal/business/money_test.go` |
| UT-013 | FR-010 | AC-018 | Hydrate unsupported independent money safely. | Product with valid card and unknown currency, excess scale, or noncanonical amount. | Price is omitted; product and raw source remain. | `appview/internal/business/money_test.go` |
| UT-014 | FR-011, FR-012, RULE-011 | AC-019, AC-020 | Validate event catalogs, required write fields, role set, and independent defaults. | All roles/modes/statuses; missing mode/status/timezone/isAllDay; empty/duplicate/5 roles; independent omitted optional values and up to 10 roles. | First-party exact required values and 1-4 distinct roles pass; invalid writes fail; omitted isAllDay hydrates false; independent omitted status behaves scheduled and omitted mode/timezone remain unspecified. | `appview/internal/business/event_validation_test.go` |
| UT-015 | FR-013 | AC-021 | Validate canonical event instants and timezone names. | Whole-second UTC Z; offsets/fractions; end equal/before start; UTC and valid/invalid IANA zones. | Only ordered whole-second `Z` instants with valid timezone pass first-party writes. | `appview/internal/business/event_time_test.go` |
| UT-016 | FR-013 | AC-021 | Validate all-day local-midnight and DST behavior. | Normal day and spring/fall DST crossings in representative IANA zones; non-midnight boundaries; inclusive/same boundary. | Each boundary must be local midnight, end is exclusive and after start by instant, and DST crossings retain correct ordering. | `appview/internal/business/event_time_test.go` |
| UT-017 | FR-014 | AC-022 | Validate duration and create-versus-update temporal policy. | 31 days; over 31 days; future, ongoing, ended create; ended existing update. | Up to 31 days and ongoing/future create pass; over-duration and ended create fail; existing past update may pass. | `appview/internal/business/event_validation_test.go` |
| UT-018 | FR-015, NFR-005 | AC-024 | Validate event links, absent onlineUri, and event image. | Zero/one/two distinct links satisfying the common web rule; exact duplicates; hostless/credentialed/oversized/unsafe links; valid/invalid image; request containing onlineUri. | Distinct safe links and valid image pass; duplicate/unsafe/image-invalid/onlineUri input fails; independent duplicate hydrates once. | `appview/internal/business/event_validation_test.go` |
| UT-019 | FR-013, FR-016, FR-017, RULE-001, RULE-003 | AC-002, AC-021, AC-025, AC-026, AC-027, AC-031 | Evaluate classification, visitor direct-read, upcoming, blocked, invalid-time, and owner-management eligibility from one policy. | Caller owner/visitor; membership/type/declaration/block matrix; scheduled/future/ongoing/ended/equal/reversed/cancelled/postponed/over-duration/moderated records. | Current owner receives retained management view; invalid ranges carry invalid-time-range; ineligible/blocked visitors get event_not_found; list/direct outcomes match criteria; declaration never affects classification/events; valid temporal state derives from endsAt. | `appview/internal/business/eligibility_test.go` |
| UT-020 | FR-003, FR-023, FR-026 | AC-009, AC-033, AC-036 | Merge declaration replacement with unknown source extensions and safe subordinate omission. | Existing raw top-level extension plus old known fields; replacement known fields; unsafe independent subordinate fields. | Known fields full-replace; unknown top-level extensions survive; unsafe subordinate fields are omitted only from hydrated output. | `appview/internal/business/profile_merge_test.go` |
| UT-021 | FR-018, NFR-003 | AC-028, AC-042 | Encode/decode opaque event cursor and enforce limits. | First-page tuple and asOf; valid cursor; malformed/tampered cursor; omitted/10/50/51 limit. | Cursor round-trips seek tuple+asOf without transparent contract; invalid cursor gives validation error; limits default/cap correctly. | `appview/internal/api/business_event_cursor_test.go` |
| UT-022 | FR-024, NFR-003 | AC-037, AC-042 | Parse required If-Match and map conflicts. | Missing, `*`, malformed, stale, and current CID; declaration presence/absence; validation errors. | `*` is accepted only for conditional declaration creation and conflicts when the singleton exists; a current canonical CID proceeds for existing-record mutation; missing/malformed/stale preconditions map to camelCase `409 pds_record_conflict` with requestId. | `appview/internal/api/business_record_request_test.go` |
| UT-023 | FR-025 | AC-038 | Enforce server-owned createdAt. | Create absent/present createdAt; update absent/same/different createdAt; stored value. | Create present is rejected and absent is stamped; update omission preserves stored value; supplying either the same or a different value is rejected. | `appview/internal/business/event_authoring_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001 | AC-001, AC-007 | Account-type migration and store default/upsert behavior. | Apply migration to real Postgres; seed member without row. | Read, set business, set regular, attempt unsupported value. | Missing reads regular; one durable row upserts exact values; DB constraint rejects unsupported values; no PDS account-type field exists. | `appview/internal/db/business_profiles_migration_test.go`, `appview/internal/business/store_test.go` |
| IT-002 | FR-001, FR-002, RULE-002 | AC-007, AC-008 | Account-type mutation authentication and ownership. | HTTP handler with real store and fake PDS that records calls. | Exercise authenticated current, unauthenticated, departed, and different-DID requests. | Only current caller's own supported scalar changes; no PDS call occurs; failures use standard errors. | `appview/internal/api/business_account_type_test.go` |
| IT-003 | FR-003, FR-023, FR-024 | AC-009, AC-036, AC-037 | Declaration conditional creation, PUT/DELETE contract, and extension-preserving replacement. | Absent singleton, then an existing PDS declaration with CID, known fields, and unknown top-level extension. | PUT creation with `If-Match: *`; PUT replacement with current CID; then DELETE with returned CID. | Creation succeeds only while absent; replacement full-replaces known fields, preserves extensions, exposes CID, and deletion removes only declaration data. | `appview/internal/api/business_profile_test.go` |
| IT-004 | BR-002, BR-003, FR-003, FR-004, FR-006, FR-008, FR-010, FR-026, RULE-005, NFR-005 | AC-003, AC-004, AC-009, AC-010, AC-013, AC-016, AC-018, AC-033 | Independent declaration projection and eligible hydration preserve source while narrowing responses. | Real Postgres; independent record with extra/unassigned address fields, unknown values/extensions, no-image product, unsupported price, and unsafe action; current business member. | Project record and read profile; repeat as regular/non-member. | Raw JSON remains exact enough for re-projection; eligible response returns safe supported details in canonical order; independent no-image card remains; unsupported price and unsafe subordinate fields are omitted; regular/non-member details are suppressed. | `appview/internal/index/business_profile_test.go`, `appview/internal/api/business_profile_store_test.go` |
| IT-005 | FR-011, FR-021, NFR-002 | AC-005, AC-032, AC-033 | Event indexer converges by repository revision and preserves independent source. | Real Postgres without membership/type/declaration dependencies; fixed revisions R1-R4. | Deliver R1 create twice, R2 update with new CID, stale R1 create, R3 delete, stale R2 update, and R4 recreate with unknown source extensions. | Newest revision wins; replay/stale operations do not change row/tombstone; delete prevents stale resurrection; R4 recreates; raw source survives. | `appview/internal/index/business_event_test.go` |
| IT-006 | FR-021, FR-026, NFR-002 | AC-032, AC-033 | Dispatch and Tap ingestion register both collections independently and carry revision TIDs. | Dispatcher and ingestion fixtures with records arriving before dependencies and in revised order. | Deliver profile/event before membership/type, duplicate and stale revisions, and unknown valid fields. | Correct indexer receives each collection with revision; records persist without dependency retries; stale delivery is ignored; unknown source survives. | `appview/internal/index/business_dispatch_test.go`, `appview/internal/ingestion/business_records_integration_test.go` |
| IT-007 | FR-011, FR-016, FR-024, FR-025, RULE-002, NFR-005 | AC-005, AC-024, AC-025, AC-037, AC-038 | Event HTTP/PDS CRUD, image contract, owner reads, and optimistic concurrency. | Authenticated owner, fake PDS with mutable record/CID, valid/invalid event image, projector convergence fixture, fixed clock. | POST, owner GET of eligible/suppressed records, valid PUT/DELETE, stale/missing If-Match, createdAt same/different injection, image failures, cross-owner mutation. | Exact lifecycle, image, direct-owner, and CID contracts hold; createdAt must be omitted; unauthorized/conflicting/invalid attempts make no unintended mutation. | `appview/internal/api/business_event_test.go` |
| IT-008 | FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-026, RULE-003, NFR-004 | AC-020, AC-021, AC-023, AC-024, AC-025, AC-026, AC-027, AC-031, AC-033 | Direct and list hydration apply caller-aware eligibility, defaults, and safe suppression. | Matrix of owner/visitor, current/departed, business/regular, blocked/unblocked, declaration/no declaration, lifecycle states, equal/reversed range, omitted isAllDay, long duration, moderation, and unsafe/unknown subordinate values. | Read each event directly and through upcoming/owner query. | Omitted isAllDay serializes as false; invalid range is raw-retained with invalid-time-range and hidden from visitors; current owner gets retained management diagnostics; ineligible/blocked visitors get event_not_found/empty lists; eligible past/cancelled/postponed remain visitor-readable; declaration has no effect. | `appview/internal/api/business_event_store_test.go` |
| IT-009 | FR-018, NFR-003 | AC-028, AC-042 | Real query pagination orders and freezes asOf. | Real Postgres with more than 50 events, equal startsAt values, and events ending around fixed asOf. | Traverse unchanged pages; alter clock; separately insert/delete/change an ordering key between pages; send bad cursor. | Unchanged traversal is complete and ordered; first asOf governs time eligibility; concurrent mutations exhibit documented seek semantics rather than snapshot guarantees; bad cursor is standard 4xx. | `appview/internal/api/business_event_pagination_test.go` |
| IT-010 | FR-019, NFR-004 | AC-023, AC-029 | Exact owner management, reporting, and moderation contracts integrate with record subjects. | More than 50 events across all lifecycle/suppression states and existing report/moderation stores. | Traverse `GET /v1/events` at default/max limits, report through `POST /v1/events/{did}/{rkey}/reports`, apply/negate record moderation. | Traversal orders startsAt/URI descending without unchanged-set gaps; every item has exact canonical reason arrays; report targets exact URI; moderation suppresses/restores and updates diagnostics. | `appview/internal/api/business_event_management_test.go`, `appview/internal/api/moderation_business_event_test.go` |
| IT-011 | FR-020, NFR-004 | AC-030, AC-031 | All profile/author query shapes hydrate account type in a constant number of queries and redact blocked shapes. | Regular/business/missing rows across profile, search, relationship, post-author, reply/quote, and notification fixtures; block relationships; instrumented query counter. | Execute each response query with 1 and 50 rows and blocked event routes. | Visible shapes contain resolved scalar; account-type query count is identical for 1 and 50 rows; blocked summaries omit business data, blocked event list is empty, and blocked direct GET is event_not_found. | `appview/internal/api/business_summary_contract_test.go`, `appview/internal/api/business_summary_query_plan_test.go` |
| IT-012 | FR-024, NFR-003 | AC-037, AC-042 | Router and middleware expose only exact methods/paths under device auth. | Construct application router with test dependencies. | Exercise exact and near-miss routes, methods, auth states, malformed bodies/cursors. | Exact routes dispatch; near misses fail; errors are camelCase envelopes with requestId and appropriate 4xx. | `appview/internal/app/business_routes_test.go` |
| IT-013 | NFR-006 | AC-043 | Observability excludes authored canaries and high-cardinality labels. | Capturing slog handler, metric reader, trace exporter; records with unique canaries. | Run successful and rejected mutation/projection/read/report flows. | No prohibited value appears in logs/labels/spans; bounded operation/result/reason metadata remains. | `appview/internal/business/observability_test.go` |
| IT-014 | BR-001, FR-022, RULE-001, RULE-006 | AC-002, AC-034 | Ordinary departure retains business state and rejoin restores it. | Real Postgres member with business row, projected declaration/events, and fixed visibility. | Process membership departure and later rejoin. | Serving is suppressed while absent; none of business row/declaration/events is deleted; rejoin restores business classification and recalculated eligibility. | `appview/internal/relationships/business_membership_lifecycle_test.go` |
| IT-015 | FR-022, RULE-006 | AC-035 | Permanent deletion registers collections and observes the complete cleanup order. | Deletion worker/PDS fake with call recorder, multiple event records, singleton declaration, account-type row, present/absent membership, unrelated records/blobs. | Run approved deletion through retry/replay and already-absent membership. | Recorder proves events → declaration → account type → membership; unrelated namespace, blobs, DID, and PDS account remain; retries preserve order and are safe. | `appview/internal/accountdeletion/business_records_acceptance_test.go`, `appview/internal/accountdeletion/collections_test.go` |
| IT-016 | FR-009, NFR-001 | AC-017, AC-041 | Lexicon schemas, 20-product ceiling, cryptographic address pin, catalog snapshots, and generated output match reviewed contracts. | Load lexicon JSON, complete CID-named address record, offline mapping, country/currency snapshots+metadata+SHA-256 sidecars, generated catalogs, and generated Go package without network. | Recompute address DAG-CBOR CID; verify snapshot digests and generated catalogs; validate 20 products/reject 21; assert schemas/refs/bounds/catalog forms and forbidden fields; run generation drift twice with network disabled. | CID equals Q17, source digests/catalogs match, lexgen resolves only the local mapping, contracts match ADR/requirements, and repeated output is stable. | `appview/internal/lexicon/craftsky/business_contract_test.go`, `appview/internal/business/catalog_provenance_test.go`, `just lexgen-check` |
| IT-017 | BR-004, RULE-010 | AC-006 | Feed remains chronological and business-neutral. | Equivalent posts from regular/business accounts with taxonomies/products/events. | Query timeline before and after changing business state. | Feed order and inclusion are unchanged. | `appview/internal/api/business_feed_neutrality_test.go` |
| IT-018 | BR-004, RULE-010 | AC-006 | Search remains business-neutral. | Equivalent searchable profiles with differing account type/taxonomy. | Query existing search and inspect ranking/filter inputs. | Existing relevance/order is unchanged; no business filter or boost exists. | `appview/internal/api/business_search_neutrality_test.go` |
| IT-019 | BR-004, RULE-010 | AC-006 | Permissions and moderation priority remain business-neutral. | Equivalent actors with regular/business type. | Exercise mutation authorization and moderation queue/output policy. | Decisions do not differ because of business state. | `appview/internal/api/business_policy_neutrality_test.go` |
| IT-020 | RULE-004 | AC-040 | Price wire/schema contracts contain only authored amount and currency. | Marshal record and AppView declaration/profile responses; load feature contract fixtures. | Enumerate price-related JSON fields and descriptions. | Only amount/currency exist; no disclaimer, inventory, availability, synchronization, shipping, tax, checkout, or accuracy field/claim is emitted. | `appview/internal/api/business_product_contract_test.go` |
| IT-021 | FR-007, FR-008, FR-015, NFR-007 | AC-015 | Destination processing has no DNS or HTTP dependency. | Inject fail-fast/counting HTTP transport and DNS resolver around validation/hydration paths. | Process valid and malicious action/product/event/registration/mailto destinations. | Resolver and transport call counts remain zero for all paths. | `appview/internal/business/no_fetch_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | `app.bsky.actor.profile` remains the owner of display name, bio, avatar, and banner; Craftsky membership shape is unchanged. | FR-003, NFR-001 | AC-009, AC-041 | Extend lexicon/profile contract tests to reject identity fields in the business declaration and detect changes to `social.craftsky.actor.profile`. |
| REG-002 | Blocked profile shells remain reduced across all existing response shapes. | FR-020, NFR-004 | AC-031 | Extend existing profile, search, relationship, post-author, reply/quote, and notification redaction tests with account/business assertions. |
| REG-003 | Main profile payload does not embed independently paginated collections. | FR-018 | AC-028 | Marshal a full profile and assert no events key while the dedicated endpoint remains available. |
| REG-004 | Existing PDS-mediated writes preserve optimistic concurrency and owner boundaries. | FR-024, RULE-002 | AC-025, AC-037 | Reuse handler/PDS effect tests to verify stale conflicts and cross-owner attempts do not dispatch writes. |
| REG-005 | Chronological feed and existing search ranking remain unchanged. | BR-004, RULE-010 | AC-006 | Run feed/search fixtures before and after account-type/taxonomy changes and compare ordered identities. |
| REG-006 | Business classification does not become authorization or entitlement. | BR-004, RULE-010 | AC-006 | Run existing relationship/content permission matrices for regular and business callers and require identical decisions. |
| REG-007 | Account deletion remains namespace-limited and does not delete blobs, DID, or PDS account. | FR-022, RULE-006 | AC-035 | Extend collection and blob-boundary tests with both new NSIDs and assert unrelated records remain. |
| REG-008 | Product prices remain schema-only seller-authored display data. | RULE-004 | AC-040 | Contract test asserts only amount/currency exist and no disclaimer, inventory, availability, synchronization, checkout, shipping, tax, or accuracy semantics are added. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Membership/type/declaration eligibility matrix | Current/departed/non-member DIDs; missing/regular/business rows; present/empty/absent declaration. | AT-001, AT-002, UT-019, IT-004, IT-008, IT-014 |
| TD-002 | Exact taxonomy catalogs | All business types, offerings, actions, roles, modes, statuses plus unknown and mixed-case sentinels. | UT-002, UT-003, UT-004, UT-008, UT-014, IT-016 |
| TD-003 | Unicode boundary corpus | ASCII, multibyte, combining graphemes, and emoji at and over each grapheme/byte boundary. | UT-005, UT-006, UT-010 |
| TD-004 | Destination safety corpus | HTTPS with query/fragment and 2048/2049 bytes; userinfo; empty, single-label, trailing-dot, Unicode, IP-literal, valid punycode, percent-encoded, and label-boundary hosts; absent/valid/zero/over-range/nondigit ports; HTTP/custom scheme; lowercase/uppercase mailto; valid/invalid ASCII dot-atom and DNS forms; whitespace, control, percent-encoded, queried, fragmented, comma/semicolon, 320/321-byte email; unique no-fetch canaries. | UT-009, UT-018, AT-014, IT-021 |
| TD-005 | Address projection corpus | Assigned and unassigned mixed-case alpha-2 values from a pinned ISO catalog; non-alpha-2 forms; bounded/oversized locality; street, region, postal code, coordinates, and unknown extension fields. | UT-006, UT-007, IT-004 |
| TD-006 | Product corpus | Ordered 4/5/20-card sets, exact and near duplicate URIs, valid/invalid images and alt text, no-image independent cards. | AT-003, UT-010, UT-011, IT-004 |
| TD-007 | Money corpus | Exact zero; USD variable-scale and trailing-zero forms; zero-, three-, and four-scale catalog entries; 12/13 integer digits; signs, exponents, separators, leading zeros; lowercase, unknown, withdrawn, and `N.A.`-scale currencies. | UT-012, UT-013, IT-004 |
| TD-008 | Event time corpus | Future, ongoing, ended, equal/reversed boundaries, 31-day/over-duration, fractional/offset timestamps, valid/invalid zones, DST spring/fall all-day events. | AT-005, UT-015, UT-016, UT-017, IT-008, IT-009 |
| TD-009 | Event listing corpus | More than 50 events, equal startsAt/different URIs, lifecycle states, boundary endsAt around fixed asOf, concurrent insert/delete/order-key update. | AT-006, IT-009 |
| TD-010 | Federated raw records | Unknown valid top-level extensions/catalog values, broad address, unsafe subordinate URI, unsupported price, duplicate event links, omitted optional event values. | AT-009, UT-007, UT-013, UT-020, IT-004, IT-005, IT-006, IT-008 |
| TD-011 | Image corpus | Minimal JPEG/PNG/WebP, unsupported MIME, exact/over 15 MiB metadata, optional aspect ratio, alt boundary strings. | UT-010, UT-018, IT-004, IT-007 |
| TD-012 | Summary/redaction corpus | Profiles, search rows, follower/following/mutual rows, post/reply/quote authors, notifications, blocked shells. | AT-008, IT-011, REG-002 |
| TD-013 | Deletion corpus | Multiple event rkeys, declaration/self, account-type row, present/absent membership, unrelated Craftsky and foreign-namespace records, blobs. | AT-010, IT-015, REG-007 |
| TD-014 | Telemetry canaries | Unique safe strings for URI, email, notes, title, price, locality, and country that are easy to search in captured output. | AT-014, IT-013 |
| TD-015 | External provenance fixtures | Complete Q17 address record; expected DAG-CBOR CID; ISO 3166-1 and SIX List One snapshots retrieved 2026-08-28; retrieval metadata and SHA-256 sidecars; representative assigned/unassigned and active/withdrawn/`N.A.` entries. | AT-013, UT-006, UT-012, IT-016, MAN-001 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-001 | AC-041 | ADR, schema evolution, external CID, and catalog provenance review. | Review both lexicons, Q17 retrieval/CID recomputation, offline mapping, country/currency source metadata+digests, old-consumer array maxima, generated catalogs/types, and ADR against every bound/optional field. | Reviewer confirms durable schemas, external record, validation catalogs, and generation are explicit, cryptographically verified, reproducible, additive where required, and approved before implementation merges. |
| MAN-002 | NFR-006 | AC-043 | Telemetry dashboard/configuration review. | Inspect configured log fields, metric labels, trace attributes, alerts, and sample captured output from integration tests. | Only bounded operation/result metadata appears; no authored destinations, email, text, title, price, or location is present. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | Resolved: owner all-events route and diagnostic codes. | FR-019 | AC-023, AC-029 | Q13 now fixes `GET /v1/events`, limits, ordering, pagination shape, diagnostic arrays, canonical codes, and report route. | Approved contract; no implementation guess remains. |
| GAP-002 | Resolved contract; ADR artifact still required for the external address pin and catalog provenance. | NFR-001 | AC-041 | Q14/Q15/Q17 fix source snapshots, metadata/digests, exact record path, DAG-CBOR CID recomputation, offline mapping, and drift behavior, but project rules still require the ADR before lexicon implementation. | Create the ADR as the first lexicon-design artifact and execute the specified retrieval/digest/CID verification before adding vendored inputs. |
| GAP-003 | No full AppView-to-real-PDS acceptance harness was identified. | FR-016, FR-024, FR-025 | AC-025, AC-037, AC-038 | Existing handler tests use fake PDS boundaries; firehose convergence is tested separately through indexer/ingestion fixtures. | Use contract-faithful fake PDS tests plus real Postgres projector tests; add a real local PDS smoke test only if the coding plan identifies a stable harness. |
| GAP-004 | “No ranking/reach effect” spans negative behavior across several existing systems. | BR-004, RULE-010 | AC-006 | Absence of future coupling cannot be proven globally by one test. | Add explicit neutrality regressions at feed, search, authorization, and moderation boundaries and retain code review as a required defense. |
| GAP-005 | Telemetry privacy cannot be exhaustively proven through unit output capture alone. | NFR-006 | AC-043 | Infrastructure exporters and future instrumentation may add attributes outside the focused test paths. | Combine canary integration tests, bounded-label helpers, manual configuration review, and review-time checks. |
| GAP-006 | Resolved: revised high-risk documents were reviewed and explicitly approved. | NFR-001, NFR-003, NFR-004 | AC-041, AC-042, AC-031 | Requirements and tests were revised from DR-001 through DR-017 and approved on 2026-08-28. | No approval gate remains; retain the High risk controls during implementation. |

## 10. Out Of Scope

- Flutter models, repositories, providers, routes, screens, widgets, navigation, publication warnings, and Flutter integration tests are excluded by NG-011.
- Native commerce, inventory, checkout, order, tax, shipping, price synchronization, booking, registration, ticketing, and attendee tests are excluded by NG-003 and NG-004.
- Business subscription, verification, advertising, boosted reach, and recommendation feature tests are excluded because the required behavior is their absence; neutrality regressions cover accidental coupling.
- Global event discovery, recurrence, canonical shared events, capacity, waitlists, accessibility, maps, coordinates, street-address hydration, and `onlineUri` are excluded by NG-007 and NG-008.
- Visual product-card and business-profile UI checks are deferred with Flutter integration. Image aspect-ratio visual treatment is not part of this AppView slice.
- Real external destination availability/content checks are prohibited; authored destinations must never be fetched by validation or hydration.
- Cross-AppView portability of the private authoritative account type is excluded by ASM-007.

## 11. Approved Handoff

- Requirements file: `docs/changes/2026-08-27-business-profiles/01-requirements.md`
- Test specification: `docs/changes/2026-08-27-business-profiles/02-acceptance-tests.md`
- Completed review artifact: `docs/changes/2026-08-27-business-profiles/03-document-review.md`
- Approved coding plan: `docs/changes/2026-08-27-business-profiles/04-coding-plan.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-27-business-profiles/`
- Recommended first failing test for implementation: `UT-001` in `appview/internal/business/account_type_test.go`, followed by `IT-001` for the private persistence default and constraint.
- Suggested test order for implementation: `UT-001` and `IT-001`; catalog/text/location/destination/money validators (`UT-002` through `UT-013`); event validation and eligibility (`UT-014` through `UT-019`); migration and indexers (`IT-004` through `IT-006`, `IT-016`); owner mutations and CID/createdAt behavior (`IT-002`, `IT-003`, `IT-007`); public/direct/owner queries and pagination (`IT-008` through `IT-010`); summary/redaction and routing (`IT-011`, `IT-012`); lifecycle/deletion (`IT-014`, `IT-015`); neutrality, no-fetch, and observability (`IT-013`, `IT-017` through `IT-021`); then acceptance scenarios and regressions.
- Commands discovered: `just appview-test-unit` for a fast incomplete unit path; `just test` with the dev stack for race-enabled Go tests; `just appview-check` for the isolated release-equivalent gate; `just lexgen`; `just lexgen-check`; `just fmt`.
- Blocking gaps: None. `GAP-001` and `GAP-006` are resolved. `GAP-002` has an approved exact contract and requires its mandated ADR before the lexicon implementation slice.
