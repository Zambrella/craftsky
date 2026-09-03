# Acceptance Test Specification: PDS Migration And Handle Change Resilience

## 1. Test Strategy

This is a high-risk change spanning OAuth authority, repository synchronization, account lifecycle, API contracts, and Flutter identity routing. The primary acceptance path uses deterministic HTTP fakes for DID documents, OAuth metadata, PDS A, PDS B, Tap administration, and complete `com.atproto.sync.getRepo` snapshots. Real PostgreSQL integration tests verify exact-parent fencing, durable jobs, repair idempotency, ownership preservation, and process-restart recovery. Flutter unit and widget tests verify DID-first session, routing, ownership, mention, search, deletion, and unavailable-handle behavior.

Security assertions capture every outbound request so old credentials can be proven absent from the new provider. Repair tests distinguish a fully verified repository from truncated, malformed, incorrectly signed, or wrong-root input before allowing inferred deletes. Existing suites provide regression coverage for the no-migration path. Manual checks are limited to release-level UI presentation and operational alerts that are impractical to prove completely in local automated tests.

All acceptance, unit, integration, contract, and regression cases are intended to be automated. The high risk level from `01-requirements.md` is unchanged, and document review plus explicit approval is required before implementation.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-017 | AT-001, IT-006, IT-009, REG-001 | Acceptance / Integration / Regression | Yes |
| BR-002 | AC-003, AC-004, AC-005, AC-006 | AT-002, AT-003, UT-008, UT-009, UT-010 | Acceptance / Unit | Yes |
| BR-003 | AC-007, AC-008 | AT-001, AT-004, IT-002 | Acceptance / Integration | Yes |
| FR-001 | AC-007, AC-009, AC-010, AC-036 | AT-004, AT-005, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-007, AC-011, AC-037 | AT-004, UT-003, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-012, AC-052, AC-055 | AT-004, AT-009, IT-004, IT-016 | Acceptance / Integration | Yes |
| FR-004 | AC-013, AC-017 | AT-005, IT-005, IT-009 | Acceptance / Integration | Yes |
| FR-005 | AC-007, AC-008, AC-038 | AT-004, UT-004, IT-002, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-001, AC-008, AC-011 | AT-001, IT-003, IT-009 | Acceptance / Integration | Yes |
| FR-007 | AC-009, AC-014, AC-053 | AT-011, UT-005, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-014, AC-015, AC-039 | AT-001, IT-006, IT-007 | Acceptance / Integration | Yes |
| FR-009 | AC-015, AC-016, AC-017, AC-040, AC-050 | AT-001, AT-010, UT-006, IT-006, IT-008, IT-009, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-018 | IT-007 | Integration | Yes |
| FR-011 | AC-014, AC-015, AC-016, AC-041 | AT-001, IT-011 | Acceptance / Integration | Yes |
| FR-012 | AC-003, AC-019 | AT-002, UT-007 | Acceptance / Unit | Yes |
| FR-013 | AC-019, AC-020 | AT-002, UT-008 | Acceptance / Unit | Yes |
| FR-014 | AC-004, AC-020, AC-021 | AT-003, UT-008, UT-009, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-015 | AC-005 | AT-003, UT-010 | Acceptance / Unit | Yes |
| FR-016 | AC-006 | AT-003, UT-009 | Acceptance / Unit | Yes |
| FR-017 | AC-022, AC-042 | AT-007, UT-013, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-023 | UT-011 | Unit | Yes |
| FR-019 | AC-024 | UT-012, IT-012 | Unit / Integration | Yes |
| FR-020 | AC-025, AC-043 | AT-006, UT-014, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-021, AC-044 | AT-003, UT-008, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-023 | AC-045 | AT-012, IT-015 | Acceptance / Integration | Yes |
| FR-025 | AC-048 | AT-008, UT-015, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-027 | AC-050, AC-051 | AT-010, IT-010 | Acceptance / Integration | Yes |
| FR-028 | AC-052 | IT-004 | Integration | Yes |
| FR-029 | AC-053 | AT-011, IT-014 | Acceptance / Integration | Yes |
| FR-030 | AC-054 | AT-011, REG-006 | Acceptance / Regression | Yes |
| FR-031 | AC-055 | AT-009, UT-002, IT-016 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-015, AC-018, AC-024, AC-027 | UT-003, UT-012, IT-002, IT-006, IT-007, IT-008, IT-012 | Unit / Integration | Yes |
| NFR-002 | AC-010, AC-028 | UT-001, IT-001, IT-005 | Unit / Integration | Yes |
| NFR-003 | AC-016, AC-029 | AT-010, IT-006, IT-008 | Acceptance / Integration | Yes |
| NFR-005 | AC-031 | UT-016, IT-017, REG-008 | Unit / Integration / Regression | Yes |
| NFR-006 | AC-032 | REG-001 through REG-008 | Regression | Yes |
| RULE-001 | AC-001, AC-004, AC-017 | AT-001, AT-003, IT-009 | Acceptance / Integration | Yes |
| RULE-002 | AC-009, AC-012 | AT-004, UT-001, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-010, AC-033 | AT-001, IT-001, REG-002 | Acceptance / Integration / Regression | Yes |
| RULE-004 | AC-034 | UT-017, REG-008 | Unit / Regression | Yes |
| RULE-005 | AC-005, AC-006, AC-023 | AT-003, UT-009, UT-010, UT-011 | Acceptance / Unit | Yes |
| RULE-006 | AC-013, AC-017, AC-053 | AT-005, AT-011, IT-009, IT-014 | Acceptance / Integration | Yes |
| RULE-007 | AC-035 | UT-018, REG-004 | Unit / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Same-Handle Migration Preserves One Account And Converges
Requirement IDs: BR-001, BR-003, FR-006, FR-008, FR-009, FR-011, RULE-001, RULE-003
Acceptance Criteria: AC-001, AC-002, AC-008, AC-014, AC-015, AC-016, AC-017, AC-033, AC-041
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/app/pds_migration_acceptance_test.go`

```gherkin
Feature: Same-DID PDS migration
  Scenario: A member migrates with an unchanged handle
    Given DID D has an active Craftsky account, private data, relationships, indexed records, and an OAuth parent for PDS A
    And HTTP fakes authoritatively resolve D with the same handle to PDS B with a rotated signing key
    And PDS B's complete verified repository contains one new, one updated, and omits one previously indexed Craftsky record
    When a protected effect detects stale authority and D reauthorizes at PDS B
    And the durable repair sweep completes
    Then Craftsky exposes one account keyed by D with all private data and ownership preserved
    And creates, updates, and inferred deletes converge exactly once to PDS B's verified repository
    And subsequent writes go to PDS B while reads continue through the AppView
```

### AT-002: Handle Change Refreshes Presentation Without Replacing Identity
Requirement IDs: BR-002, FR-012, FR-013
Acceptance Criteria: AC-003, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/handle_change_acceptance_test.dart`

```gherkin
Feature: Active-account handle refresh
  Scenario: The app cold-starts with a stale stored handle
    Given secure storage contains DID D, a Craftsky token, lease generation, unrelated account fields, and handle old.example
    And whoami returns DID D with handle new.example
    When session validation and own-profile loading complete
    Then the app persists and reactively displays new.example
    And it retains D, the token, lease generation, routing binding, and unrelated fields unchanged
    And it loads the signed-in profile through me or D rather than old.example
```

### AT-003: Durable Profile Targets Survive Handle Reassignment
Requirement IDs: BR-002, FR-014, FR-015, FR-016, FR-021, RULE-001, RULE-005
Acceptance Criteria: AC-004, AC-005, AC-006, AC-020, AC-021, AC-044
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/did_first_identity_acceptance_test.dart`

```gherkin
Feature: DID-first profile navigation
  Scenario: An old handle is reassigned after D changes handle
    Given a historical mention displays @old.example with facet DID D
    And a recent profile, internal profile link, and canonical profile URL target D
    And old.example now resolves to another DID while D uses new.example
    When the viewer opens each stored destination and taps the mention
    Then every durable destination opens D using a DID-keyed route and cache entry
    And the historical post still displays @old.example
    And D's own profile is recognized by DID and does not show self-targeting actions
    And only an explicit current alias lookup may resolve old.example to its new owner
```

### AT-004: Stale Parent Is Fenced Before Any Credential-Bearing Effect
Requirement IDs: BR-003, FR-001, FR-002, FR-003, FR-005, RULE-002
Acceptance Criteria: AC-007, AC-009, AC-012, AC-036, AC-037, AC-038
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/pds_migration_auth_acceptance_test.go`

```gherkin
Feature: Write-time OAuth authority enforcement
  Scenario: A protected effect selects an OAuth parent from the old PDS
    Given parent P and its child Craftsky session were issued by PDS A for DID D
    And uncached authoritative resolution proves D now belongs to PDS B and its authorization server
    And another independently valid parent exists
    When the child requests an authenticated PDS effect
    Then P and only P's children are fenced through the existing terminalization path
    And no access token, refresh token, DPoP material, revocation request, or effect from P reaches PDS B
    And the API returns 401 with error pds_session_expired, a message, and a requestId
    And the independently valid parent remains usable
```

### AT-005: Transient Federation Failure Does Not Delete Or Log Out The Account
Requirement IDs: FR-004, NFR-002, RULE-006
Acceptance Criteria: AC-013, AC-028, AC-036
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/authority_failure_acceptance_test.go`

```gherkin
Feature: Retryable migration propagation
  Scenario Outline: Authority cannot be conclusively verified
    Given DID D owns active Craftsky data and an active OAuth parent
    When authority validation encounters <failure>
    Then the protected effect is not sent
    And the result remains retryable rather than pds_session_expired
    And neither the OAuth parent nor owner is terminalized
    And D-owned data remains present

    Examples:
      | failure |
      | DID resolution timeout |
      | DNS failure |
      | current PDS outage |
      | generic old-PDS upstream failure |
      | outbound-policy rejection |
```

### AT-006: Invalid Handle Uses Sentinel Only On The Wire
Requirement IDs: FR-020
Acceptance Criteria: AC-025, AC-043
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/invalid_handle_acceptance_test.dart`

```gherkin
Feature: Identity without a valid handle
  Scenario: Authoritative bidirectional validation rejects the current handle
    Given DID D previously used old.example and still owns indexed content
    When identity refresh reports no bidirectionally valid handle
    Then API non-null handle fields contain handle.invalid
    And old.example is removed from current search and alias mappings
    And D's content remains accessible by DID
    And Flutter shows handle unavailable with available display identity
    And Flutter never presents handle.invalid or old.example as D's current handle
```

### AT-007: Account Deletion Always Confirms The Full DID
Requirement IDs: FR-017
Acceptance Criteria: AC-022, AC-042
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/account_deletion_did_acceptance_test.dart`

```gherkin
Feature: Stable account-deletion confirmation
  Scenario Outline: A deletion intent is created regardless of handle state
    Given authenticated owner DID D has a <handle_state> handle
    When the server creates a deletion intent and Flutter displays and submits it
    Then the confirmation value is the full server-bound DID D
    And only an exact confirmation of D is accepted

    Examples:
      | handle_state |
      | valid |
      | stale |
      | invalid |
```

### AT-008: Scheduled Publication Respects Existing Late Cutoff
Requirement IDs: FR-025
Acceptance Criteria: AC-048
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/scheduledposts/pds_migration_acceptance_test.go`

```gherkin
Feature: Scheduled posts during PDS migration
  Scenario Outline: Current-PDS authorization returns around the publication cutoff
    Given a scheduled post becomes due while its selected OAuth parent is stale
    When valid authorization at the current PDS returns <delay> after the due time
    Then the post <outcome>
    And no write is sent to the old PDS

    Examples:
      | delay | outcome |
      | 29 minutes 59 seconds | may publish automatically through the current PDS |
      | exactly 30 minutes with authorization available | may publish automatically through the current PDS on the final eligible attempt |
      | exactly 30 minutes with the final attempt failing | becomes needs_attention and is not automatically published |
      | more than 30 minutes | remains needs_attention and is not automatically published |
```

### AT-009: OAuth Callback Rejects Mixed Authority
Requirement IDs: FR-003, FR-031
Acceptance Criteria: AC-055
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/oauth_migration_callback_acceptance_test.go`

```gherkin
Feature: OAuth callback authority verification
  Scenario: DID migration completes between login initiation and callback
    Given login starts with PDS A and its authorization server for DID D
    And D moves authoritatively to PDS B before A returns tokens
    When Craftsky processes A's callback
    Then it uncached-resolves D, PDS B, and B's authorization server
    And it rejects the mixed-authority session before persistence
    And it sends no A credential to B
    And cleanup is bounded and directed only to A
```

### AT-010: Incomplete Or Unverified Snapshot Cannot Cause Deletes
Requirement IDs: FR-009, FR-027, NFR-003
Acceptance Criteria: AC-016, AC-029, AC-050, AC-051
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/ingestion/repository_repair_acceptance_test.go`

```gherkin
Feature: Verified repository repair
  Scenario Outline: A repository snapshot cannot be trusted as complete
    Given AppView has Craftsky records for DID D
    When getRepo <failure>
    Then no absence is applied as a delete
    And the job is not marked complete
    And durable retry remains scheduled with bounded backoff

    Examples:
      | failure |
      | terminates before EOF |
      | has an invalid commit signature |
      | has an unexpected root |
      | changes authoritative source during the attempt |

  Scenario: A verified complete snapshot omits the Craftsky profile
    Given the entire authoritative repository is read and commit, signature, and root verification succeeds
    And the Craftsky profile record is absent
    When repair applies the verified comparison
    Then the existing departure policy stops member-only writes and scheduled work
    And DID D is not terminalized or recorded as deleted because of migration
```

### AT-011: Tap Status Is A Hint And Replay Resumes From The Cursor
Requirement IDs: FR-007, FR-029, FR-030, RULE-006
Acceptance Criteria: AC-053, AC-054
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/tap/migration_identity_acceptance_test.go`

```gherkin
Feature: Tap migration resilience
  Scenario Outline: Tap reports a non-active account status
    Given DID D has active Craftsky ownership and a durable relay cursor
    When Tap reports D as <status>
    Then Craftsky records or refreshes synchronization state as applicable
    And it does not terminalize or purge D
    And a reconnect requests replay from the durable cursor rather than the live head

    Examples:
      | status |
      | deleted |
      | inactive |
      | suspended |
      | takendown |
```

### AT-012: Cold Start Does Not Add Client-Wide Write Gating
Requirement IDs: FR-023
Acceptance Criteria: AC-045
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/cold_start_write_behavior_test.dart`

```gherkin
Feature: Cold-start migration discovery
  Scenario: Validation is still pending when a write control loads
    Given Flutter has started and cold-start validation has not completed
    When the user reaches and activates a write control
    Then normal reads and write controls are not globally disabled by client migration state
    And the backend authority boundary either performs the write safely or returns actionable reauthorization
    And no obsolete PDS is contacted
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, NFR-002, RULE-002 | AC-009, AC-010, AC-028, AC-036 | Classify uncached DID/PDS/authorization-server checks without trusting hints or cached identity. | Matching authority; mismatched PDS; mismatched issuer; timeout; DNS error; unsafe endpoint; unchanged handle with changed authority. | Only a conclusive current-authority mismatch is stale; matching authority proceeds through bounded clients; transient/policy failures do not invoke the effect. | `appview/internal/auth/session_coordinator_test.go` |
| UT-002 | FR-031 | AC-055 | Validate callback authority against the resource and issuer used for authorization/token exchange. | Callback subject D plus matching and mismatching current PDS/issuer metadata. | Matching callback is eligible for persistence; mismatch is rejected before persistence with original-issuer cleanup metadata. | `appview/internal/auth/oauth_test.go` |
| UT-003 | FR-002, NFR-001 | AC-011, AC-027, AC-037 | Select and fence OAuth parents by exact DID, parent ID, row version, lifecycle generation, and auth epoch. | Multiple current/stale parents, concurrent duplicate fencing, corrected row version. | Only the proven stale parent and children are fenced; current parent remains eligible; repeats are idempotent. | `appview/internal/auth/background_session_selector_test.go`, `appview/internal/auth/session_coordinator_test.go` |
| UT-004 | FR-005 | AC-007, AC-038 | Map confirmed stale-session errors to the standard API envelope. | `ErrPDSSessionExpired` or equivalent coordinator result. | HTTP 401 with camelCase `{error: "pds_session_expired", message, requestId}`; transient failures do not map to this code. | `appview/internal/api/profile_test.go`, `appview/internal/api/error_test.go` |
| UT-005 | FR-007 | AC-009, AC-014, AC-053 | Classify Tap identity events as identity-refresh/synchronization hints only. | Ordinary, deleted, inactive, suspended, and takendown identity events; equal and changed handles. | Refresh work may be requested, but no OAuth or owner terminalization decision is produced. | `appview/internal/ingestion/identity_event_policy_test.go` |
| UT-006 | FR-009 | AC-040, AC-050 | Build the repair comparison from the shared indexer registry only after snapshot verification. | Registered Craftsky NSIDs, unknown NSID, verified/unverified snapshot marker. | Every registered collection participates once; unknown collections are ignored; absence comparison is unavailable before verification. | `appview/internal/app/indexer_wiring_test.go`, `appview/internal/ingestion/repository_repair_test.go` |
| UT-007 | FR-012 | AC-003, AC-019 | Reconcile whoami handle into an existing stored account. | Same DID, old/new handles, existing token, lease generation, route binding, and unrelated fields. | Only presentation handle changes and reactive observers receive the update. | `app/test/auth/services/session_validation_coordinator_test.dart` |
| UT-008 | FR-013, FR-014, FR-021 | AC-004, AC-019, AC-020, AC-021, AC-044 | Produce own-profile and known-profile requests/routes keyed by DID and canonicalize valid aliases. | Own account, post, notification, search, follow, relationship, suggestion, recent search, valid alias. | Requests, provider keys, cache keys, and canonical URLs use DID; alias is retained only as input/presentation. | `app/test/profile/providers/profile_identity_test.dart`, `app/test/router/profile_routes_test.dart` |
| UT-009 | FR-014, FR-016, RULE-005 | AC-004, AC-006, AC-020 | Dispatch a mention and recent-profile destination using stored DID rather than visible/stored handle. | Historical text `@old.example`, facet DID D, recent search with DID D and stale handle. | Navigation targets D while visible historical text is unchanged. | `app/test/shared/rich_text/faceted_text_actions_test.dart`, `app/test/search/models/recent_search_test.dart` |
| UT-010 | FR-015, RULE-005 | AC-005 | Determine profile ownership and authorization-sensitive actions by DID. | Same DID/different handles; different DIDs/same handle. | Same DID is self and suppresses follow/report/mute/block; same handle does not confer ownership. | `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_actions_test.dart` |
| UT-011 | FR-018, RULE-005 | AC-023 | Merge paginated profile results by DID. | Same DID under two handles; same handle under two DIDs; duplicate DID on adjacent pages. | One result per DID and distinct DIDs are never suppressed by handle equality. | `app/test/search/providers/search_pagination_merge_test.dart` |
| UT-012 | FR-019, NFR-001 | AC-024, AC-027 | Version-fence competing authoritative identity refresh completions and calculate cache invalidations. | Older refresh completes after newer commit; duplicate event; old/new/sentinel handles. | Newest authoritative value wins; DID plus affected old/new handle keys are invalidated after commit; repeats converge. | `appview/internal/api/identity_cache_refresh_test.go`, `appview/internal/api/identity_cache_store_test.go` |
| UT-013 | FR-017 | AC-022, AC-042 | Model and submit deletion confirmation exclusively as full DID. | Valid, stale, invalid, and absent handles with server-bound DID D; near-match DID input. | UI/persistence submits D; exact D succeeds; all non-exact values fail. | `app/test/settings/account_deletion_controller_test.dart`, `app/test/settings/account_deletion_repository_test.dart` |
| UT-014 | FR-020 | AC-025, AC-043 | Format `handle.invalid` centrally for identity presentation and alias input. | Sentinel, valid handle, stale local handle, display name present/absent. | Sentinel is never displayed; UI says `handle unavailable`; alias input cannot use stale/unverified value. | `app/test/profile/models/profile_handle_test.dart`, `app/test/profile/widgets/profile_card_test.dart` |
| UT-015 | FR-025 | AC-048 | Apply the existing scheduled-post retry cutoff around stale authority. | Authorization at 29:59, exactly 30:00, and after 30:00 from due time; success and failure at the final attempt. | Authorization available at exactly 30:00 may publish on the final eligible attempt. A failed final attempt transitions to `needs_attention`; authorization after 30:00 cannot automatically publish. | `appview/internal/scheduledposts/retry_test.go`, `appview/internal/scheduledposts/failure_acceptance_test.go` |
| UT-016 | NFR-005 | AC-031 | Redact migration/auth/reconciliation fields from structured errors and telemetry. | Access/refresh tokens, DPoP key/proof, bearer token, confirmation hash, raw session JSON embedded in errors. | Output contains bounded reason codes and IDs but none of the supplied secrets. | `appview/internal/observability/secret_scan_test.go`, `app/test/observability/secret_scan_test.dart` |
| UT-017 | RULE-004 | AC-034 | Enforce Flutter's credential model in storage and request interceptors. | Craftsky token plus attempted PDS token, endpoint, refresh token, and DPoP fields. | Only opaque Craftsky session credentials are represented, stored, or attached to AppView requests. | `app/test/auth/providers/secure_token_storage_test.dart`, `app/test/shared/api/providers/session_auth_interceptor_test.dart` |
| UT-018 | RULE-007 | AC-035 | Inventory source route/provider/cache identity APIs for obsolete handle-keyed compatibility branches and require focused coverage for each known identity surface. | Flutter route declarations plus own profile, post, notification, search, follow list, relationship list, suggestion, recent search, mention, account switcher, and deletion call sites. | Canonical known-identity contracts require DID; no fallback branch preserves an obsolete handle-based identity path without a requirement; every inventoried surface is paired with a focused provider or widget test. | `app/test/router/router_usage_test.dart`, `app/test/profile/did_first_identity_inventory_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, NFR-002, RULE-003 | AC-009, AC-010, AC-028, AC-033 | Exercise coordinator authority validation and a real protected-effect adapter through HTTP fakes. | Active parent, lifecycle fences, fake DID/OAuth metadata and PDS, outbound policy/deadlines. | Request an effect with matching, mismatching, unsafe, and timing-out destinations. | Each attempt resolves uncached; matching write reaches only current PDS; reads remain AppView; unsafe/transient cases send no credential-bearing effect. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-002 | BR-003, FR-002, FR-005, NFR-001 | AC-007, AC-008, AC-027, AC-037, AC-038 | Persist exact-parent terminalization and API recovery under retries/concurrency. | PostgreSQL owner with several parents and children, one proven stale. | Concurrent foreground requests fence the stale parent and call authenticated endpoints. | One transition fences only the stale parent/children; affected calls receive exact 401 envelope; valid parent and owner data survive. | `appview/internal/auth/session_coordinator_test.go`, `appview/internal/api/pds_migration_auth_integration_test.go` |
| IT-003 | FR-002, FR-006, FR-025 | AC-001, AC-008, AC-011, AC-048 | Verify foreground/background selection after same-DID reauthorization and scheduler cutoff behavior. | Stale old parent, valid new parent, scheduled posts inside/outside 30-minute window. | Select sessions and run scheduled work after reauthorization. | Only current parent is selected; account remains D; in-window work may publish and late work becomes `needs_attention`. | `appview/internal/auth/background_session_selector_test.go`, `appview/internal/scheduledposts/failure_acceptance_test.go` |
| IT-004 | FR-003, FR-028, RULE-002 | AC-012, AC-052 | Capture terminalization cleanup traffic. | Parent P issued by authorization server A; current provider B; fake endpoints record headers/bodies. | Prove P stale and run bounded cleanup with success and retryable failure. | Local fencing does not wait; P credentials are sent only to A; no P credential or proof reaches B; cleanup retry does not block reauthorization. | `appview/internal/auth/session_cleanup_processor_test.go` |
| IT-005 | FR-004, NFR-002 | AC-013, AC-028, AC-036 | Preserve lifecycle and data on transient resolution/PDS failures. | Active owner with representative private/indexed rows and parent. | Inject timeout, DNS, outage, generic upstream error, and policy denial. | Error remains retryable, owner/parent stay non-terminal, data remains, and forbidden destinations receive no request. | `appview/internal/auth/session_coordinator_test.go`, `appview/internal/ownerlifecycle/store_integration_test.go` |
| IT-006 | BR-001, FR-008, FR-009, NFR-001, NFR-003 | AC-001, AC-002, AC-015, AC-016, AC-017, AC-027, AC-029, AC-039, AC-040, AC-050 | Reconcile a complete signed CAR into PostgreSQL through the shared indexers. | DID-owned records across every registered collection with new/updated/missing records and a fake authoritative signed repository. | Run duplicate and interrupted repair jobs, then retry to completion. | Creates/updates/deletes converge exactly once only after full verification; all ownership remains D; incomplete work is not complete and resumes safely. | `appview/internal/ingestion/repository_repair_integration_test.go` |
| IT-007 | FR-008, FR-010, NFR-001 | AC-014, AC-018, AC-027, AC-039 | Durably coalesce Tap tracking and repair requests across commit/restart. | Table-driven onboarding, ordinary sign-in, and reauthentication cases with Tap `AddRepo` initially unavailable; same-DID reauthorization and source-order uncertainty also request repair. | Commit each entry point, fail the first remote attempt, restart the worker/process, recover fake Tap, and redeliver duplicate triggers. | Every entry point persists one effective tracking job that completes without another user write; repair requests coalesce; duplicate triggers do not duplicate effects. | `appview/internal/ingestion/repository_jobs_integration_test.go`, `appview/internal/auth/initialize_profile_effect_test.go`, `appview/internal/auth/oauth_test.go` |
| IT-008 | FR-009, NFR-001, NFR-003 | AC-015, AC-016, AC-027, AC-029, AC-050 | Fail repository download/verification at every trust boundary. | Existing AppView records plus truncated CAR, bad signature, wrong root, source change, and valid CAR variants. | Run and retry repair. | Invalid attempts infer no deletes and remain retryable; valid retry converges; repeated valid work is idempotent. | `appview/internal/ingestion/repository_repair_integration_test.go` |
| IT-009 | BR-001, FR-004, FR-006, FR-009, RULE-001, RULE-006 | AC-001, AC-002, AC-013, AC-017 | Preserve DID-owned account boundary and representative data through migration. | Seed owner lifecycle, drafts, saved state, notifications, moderation state, relationships, scheduled work, and records under D. | Detect migration, fence stale parent, reauthorize same DID, and repair. | No row changes owner DID or disappears solely from endpoint movement; owner remains one non-terminal identity. | `appview/internal/app/pds_migration_ownership_integration_test.go` |
| IT-010 | FR-009, FR-027 | AC-050, AC-051 | Apply profile absence only from a verified complete repository. | Active member D with scheduled/member-only work; snapshots with and without profile, verified and unverified. | Run repair. | Unverified absence changes nothing; verified absence applies existing departure policy without DID terminalization; projection may update incrementally only after verification. | `appview/internal/ingestion/repository_repair_profile_integration_test.go` |
| IT-011 | FR-011 | AC-014, AC-015, AC-016, AC-041 | Run pinned-version HTTP-fake migration contract matrix. | Exact selected Indigo/Tap versions; PDS A/B, DID/OAuth metadata, signing-key rotation, reset commit chain, complete snapshots, missing live events. | Run same-handle and changed-handle migration cases. | Both variants converge creates/updates/deletes without two real PDS instances and expose dependency behavior changes on upgrade. | `appview/internal/tap/pds_migration_contract_test.go` |
| IT-012 | FR-019, FR-020, FR-021, NFR-001 | AC-024, AC-025, AC-027, AC-043, AC-044 | Commit authoritative handle changes atomically with alias ownership and local-cache invalidation. | D old/new handles, reassigned old handle, concurrent old refresh, and no-valid-handle result. | Commit newer refresh, then complete stale refresh; repeat events. | Newest value wins; old alias is removed; new alias requires bidirectional proof; sentinel is stored on wire path; affected local keys invalidate after commit. | `appview/internal/api/identity_cache_refresh_test.go`, `appview/internal/api/identity_cache_store_test.go` |
| IT-013 | FR-017 | AC-022, AC-042 | Enforce full-DID deletion intent contract server-side. | Authenticated D with valid/stale/invalid handle variants. | Create intent and submit exact and non-exact confirmations. | Intent always returns full D; only exact D advances deletion; camelCase wire contract remains valid. | `appview/internal/api/account_deletion_identity_test.go`, `appview/internal/auth/account_deletion_reauth_test.go` |
| IT-014 | FR-007, FR-029, RULE-006 | AC-009, AC-053 | Persist and redeliver all Tap identity/account status variants after pure classification by UT-005. | Active owner D and duplicate/out-of-order ordinary/deleted/inactive/suspended/takendown events. | Ingest, restart, and redeliver events. | Refresh/sync work is transactional and idempotent; no event terminalizes or purges D; permanent deletion remains exclusive to authenticated deletion. | `appview/internal/ingestion/identity_refresh_trigger_integration_test.go`, `appview/internal/ingestion/lifecycle_integration_test.go` |
| IT-015 | FR-005, FR-023 | AC-038, AC-045 | Verify backend protection when Flutter has not completed validation. | Flutter/API harness with pending cold-start validation and stale or current parent. | Invoke a write endpoint. | Current authority proceeds; stale authority returns standard 401 and existing invalid-session recovery; no client-wide migration gate or obsolete-PDS request occurs. | `app/test/auth/cold_start_write_behavior_test.dart`, `appview/internal/api/profile_test.go` |
| IT-016 | FR-003, FR-031 | AC-012, AC-055 | Persist no session when migration occurs during ordinary OAuth callback. | OAuth begins at A; A returns valid tokens for D after uncached authority moves to B. | Complete callback and run cleanup. | No mixed session/child is committed, no A credential reaches B, and cleanup targets only A. | `appview/internal/auth/oauth_test.go`, `appview/internal/auth/handoff_handlers_test.go` |
| IT-017 | NFR-005 | AC-031 | Inspect emitted logs, metrics labels, traces, and API errors across migration outcomes. | Secret-bearing test values injected into success, stale, transient, repair, and cleanup paths. | Exercise all paths and scan captured telemetry. | Secrets and raw session JSON are absent; bounded operation/reason/result labels and request/run IDs remain. | `appview/internal/observability/pds_migration_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Ordinary sign-in, account switching, and DID-keyed private/indexed ownership without migration. | BR-001, NFR-006 | AC-001, AC-002, AC-032 | Run existing auth, account-switcher, owner-lifecycle, and persistence suites with unchanged authority and assert no second account or data movement. |
| REG-002 | Public reads come from AppView and writes use the authorized PDS. | RULE-003, NFR-006 | AC-010, AC-032, AC-033 | Run existing profile/post/follow/business effect tests and assert the established read/write split. |
| REG-003 | Existing profile, follow, relationship, notification, search, and account-switching behavior. | NFR-006 | AC-032 | Run corresponding Go and Flutter suites with stable DIDs/handles. |
| REG-004 | Normal profile aliases resolve while canonical known-identity routes and caches are DID-first. | FR-014, RULE-007, NFR-006 | AC-020, AC-021, AC-032, AC-035 | Run profile API/router/provider tests; assert current external handle input resolves once and canonicalizes without compatibility fallback. |
| REG-005 | Existing scheduled post publication and failure lifecycle. | FR-025, NFR-006 | AC-032, AC-048 | Run `internal/scheduledposts` and scheduled API suites, including the existing retry schedule and `needs_attention` transition. |
| REG-006 | Tap reconnect/replay and ordinary at-least-once ingestion. | FR-030, NFR-006 | AC-032, AC-054 | Run Tap replay, consumer, ingestion lease/replay, and repository-job suites with replay enabled and durable cursor assertions. |
| REG-007 | Authenticated permanent deletion remains the only terminal purge path. | FR-029, RULE-006, NFR-006 | AC-032, AC-053 | Run account-deletion and terminal-purge acceptance suites; verify explicit deletion still terminalizes while federation status does not. |
| REG-008 | Existing security boundary and secret redaction. | NFR-005, RULE-004, NFR-006 | AC-031, AC-032, AC-034 | Run Go/Flutter secret scans, federated HTTP policy tests, secure storage, and auth interceptor tests. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Stable migrated identity | DID `did:plc:migrating`, handles `old.example` and `new.example`, PDS A/B and authorization-server A/B URLs, signing keys A/B. | AT-001, AT-004, AT-009, IT-001 through IT-005, IT-009, IT-011, IT-016 |
| TD-002 | Independent parent/device selection | Two stale parents and one current parent for the same DID, each with separate child Craftsky sessions, row versions, auth epochs, and last-seen times. | AT-004, UT-003, IT-002, IT-003 |
| TD-003 | Complete authoritative repository | Deterministic signed CAR with valid commit/root, every registered Craftsky collection, unchanged/new/updated/missing records, and optional missing profile. | AT-001, AT-010, UT-006, IT-006, IT-008, IT-010, IT-011 |
| TD-004 | Invalid repository variants | Truncated stream, malformed block, bad signature, wrong root, source changed during fetch, and commit-chain reset signed by current key. | AT-010, IT-008, IT-011 |
| TD-005 | DID-owned account state | Follows, drafts, saved items, notifications, moderation state, scheduled posts, owner lifecycle, indexed records, and profile all keyed by D. | AT-001, AT-005, IT-005, IT-009 |
| TD-006 | Handle reassignment identities | D originally owns `old.example` then `new.example`; E later validly owns `old.example`; F has `handle.invalid`; include one-way-invalid alias. | AT-002, AT-003, AT-006, UT-007 through UT-012, UT-014, IT-012 |
| TD-007 | Historical client destinations | Mention text with facet DID D, recent-profile entry with DID D and stale handle, post/notification/follow/relationship/suggestion profile references. | AT-003, UT-008, UT-009 |
| TD-008 | Scheduled cutoff times | Due instant plus authorization return at 29:59, 30:00, and 30:01, using a controllable clock. | AT-008, UT-015, IT-003 |
| TD-009 | Sensitive canary values | Unique access/refresh tokens, DPoP private key/proof, Craftsky bearer token, deletion hash, and raw session JSON markers. | AT-004, UT-016, UT-017, IT-004, IT-017, REG-008 |
| TD-010 | Tap lifecycle and replay | Durable relay cursor plus duplicate/out-of-order ordinary, deleted, inactive, suspended, and takendown events. | AT-011, UT-005, IT-014, REG-006 |
| TD-011 | Deletion identities | Full DID D plus valid, stale, invalid, and absent handle states and exact/non-exact confirmation strings. | AT-007, UT-013, IT-013 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-005, FR-020 | AC-038, AC-043 | Device-level reauthorization and unavailable-handle presentation. | Run the app on one supported mobile device; use fake/local responses to enter `pds_session_expired` and `handle.invalid` states; inspect account switcher, settings, own profile, profile card, and recovery navigation. | Recovery is actionable without migration-specific reason state; all identity surfaces say `handle unavailable` where appropriate and never show the sentinel or stale handle. |
| MAN-002 | NFR-003, NFR-005 | AC-029, AC-031 | Operational dashboards and alerts are usable and secret-free. | In a staging-like environment, generate successful, retrying, stale-parent, Tap failure, and aged repair-backlog signals; inspect logs, metrics, dashboards, and configured alerts. | Operators can distinguish bounded outcomes and see backlog age/retry count; sustained backlog, stale-parent selection, and Tap resync failures alert; no sensitive canary appears. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Acceptance Criteria | Reason | Follow-Up |
|---|---|---|---|---|---|
| GAP-001 | No test uses two real local PDS instances. | FR-011 | AC-041 | Explicitly deferred in requirements; deterministic HTTP fakes are the approved boundary. | Keep protocol fixtures realistic and rerun the contract matrix whenever Indigo or Tap is upgraded. |
| GAP-002 | CAR size, batch, lease, retry, and alert thresholds are not fixed. | FR-009, NFR-003 | AC-029, AC-050 | Requirements leave these non-blocking values to coding/operations design. | Define limits in `04-coding-plan.md`; add boundary and retry tests once selected. |
| GAP-003 | Full end-to-end alert delivery depends on deployment infrastructure not represented by local tests. | NFR-003, NFR-005 | AC-029, AC-031 | Unit/integration tests can prove signal emission but not every external alert route. | Retain MAN-002 and document staging evidence before release. |
| GAP-004 | Pinned Tap behavior may change when the image is upgraded. | FR-011, FR-030 | AC-041, AC-054 | Dependency behavior is beta and external. | Treat IT-011 and REG-006 as upgrade gates; update assumptions only after reviewed evidence. |
| GAP-005 | Broad Flutter identity call-site inventory may miss dynamically constructed navigation. | FR-014, NFR-006 | AC-020, AC-032 | Static inventory tests cannot prove every future runtime path. | UT-018 now requires each inventoried identity surface to have a focused provider/widget test. Retain code review and add a regression whenever a new identity surface is introduced. |

## 10. Out Of Scope

- Tests that orchestrate or perform PDS-to-PDS account migration.
- Transfer of OAuth credentials, DPoP keys, passwords, blobs, private PDS state, PLC rotation keys, or signing keys.
- Migration-specific standalone Instagram importer recovery behavior beyond ordinary OAuth authority regression coverage.
- Repository-wide atomic serving, previous-snapshot double buffering, or migration-specific notification, push, and analytics suppression.
- Backward-compatibility tests for unshipped handle-based routes, stale Flutter persistence, or superseded API shapes.
- Lexicon migration or record-shape tests because no lexicon change is expected.
- Generic or cross-DID PDS mutation through reconciliation; repair is read-only against the PDS and DID-scoped in AppView.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-09-02-pds-migration-handle-changes/`
- Risk level: High. Document review and explicit approval are required before implementation.
- Recommended first failing test for implementation: UT-001 in `appview/internal/auth/session_coordinator_test.go`, proving that an uncached authoritative PDS/authorization-server mismatch prevents the callback effect before any stale credential-bearing request and distinguishes transient resolution failure.
- Suggested test order for implementation: UT-001, UT-002, UT-003, UT-004; IT-001 through IT-005 and IT-016; IT-007; UT-006 and IT-006/IT-008/IT-010/IT-011; UT-012 and IT-012; UT-005 and IT-014; UT-007 through UT-011, UT-013, UT-014, UT-017, UT-018 and Flutter acceptance scenarios; UT-015 and IT-003; IT-017; regression suites; manual checks.
- Commands discovered: `cd appview && go test ./internal/auth ./internal/api ./internal/ingestion ./internal/tap`; `cd appview && go test ./internal/scheduledposts ./internal/app`; `cd app && flutter test test/shared/rich_text/faceted_text_actions_test.dart test/auth test/profile test/search test/router test/settings`; `just app-test`; `just appview-test-unit`; `just test` with the Compose PostgreSQL/MinIO services; release-equivalent AppView evidence through `just appview-check`.
- Blocking gaps: None for test design. GAP-002 thresholds must be resolved during coding/operations design before their boundary tests can be finalized.
- Review gate: Because risk is high, do not continue to implementation without completed document review and explicit approval.
