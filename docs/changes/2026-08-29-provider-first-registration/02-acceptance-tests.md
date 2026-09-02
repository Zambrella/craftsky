# Acceptance Test Specification: Provider-First AT Protocol Registration

## 1. Test Strategy
Provider-first registration is a high-risk authentication change. Coverage therefore starts at the database authority boundary, proceeds through AppView provider discovery/PAR/callback/lifecycle behavior, and finishes with Flutter API, controller, routing, and widget behavior.

Automated tests use controlled authorization-server, PDS, DID, and network fixtures. Real PostgreSQL tests verify purpose-aware constraints, atomic owner binding, request-state transitions, capacity, expiry, lifecycle fencing, and restart durability. The existing real federated-flow fixture verifies protected-resource discovery, authorization-server discovery, PAR, token exchange, authoritative DID/PDS checks, onboarding PDS operations, and handoff creation without contacting a live provider.

Flutter unit and widget tests verify both registration entry points, exact API requests, immediate external launch, retained-account behavior, retry after browser abandonment, and safe error categories. Existing handle-first login, account deletion, OAuth handoff, SSRF controls, and multi-account tests remain mandatory regressions.

One production-like manual smoke test against Bluesky is required before release. CI must not create live Bluesky accounts.

Risk level: High. Automated and manual evidence must pass, followed by the workflow document review and explicit implementation approval.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-010 | AT-001, AT-004, IT-011, MAN-001 | Acceptance / Integration / Manual | Yes, except live provider |
| BR-002 | AC-002, AC-018 | AT-002, AT-008, REG-001, REG-002, REG-003 | Acceptance / Regression | Yes |
| BR-003 | AC-017 | IT-015, REG-009 | Integration / Regression | Yes |
| FR-001 | AC-001, AC-002, AC-020 | AT-001, AT-002, AT-005, AT-009, UT-010, REG-008 | Acceptance / Unit / Regression | Yes |
| FR-002 | AC-003 | IT-001, UT-001 | Integration / Unit | Yes |
| FR-003 | AC-003, AC-004, AC-023 | IT-002, IT-003, IT-018, UT-003, UT-004, MAN-001 | Integration / Unit / Manual | Yes, except live provider |
| FR-004 | AC-005, AC-016, AC-027 | IT-004, IT-005, IT-016, IT-021, REG-006 | Integration / Regression | Yes; real PostgreSQL |
| FR-005 | AC-005, AC-006, AC-016, AC-029 | UT-013, IT-004, IT-006, IT-016, IT-020, REG-006 | Unit / Integration / Regression | Yes |
| FR-006 | AC-007, AC-015, AC-021 | AT-001, AT-003, UT-006, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-007 | AC-008, AC-011, AC-012 | IT-007, IT-010, IT-019, IT-020 | Integration | Yes |
| FR-008 | AC-008, AC-009 | IT-007, IT-008, IT-015 | Integration | Yes |
| FR-009 | AC-009, AC-011, AC-013 | AT-005, IT-009, IT-012, IT-021, IT-022 | Acceptance / Integration | Yes; real PostgreSQL |
| FR-010 | AC-010 | AT-004, IT-011, REG-005, REG-008 | Acceptance / Integration / Regression | Yes |
| FR-011 | AC-011, AC-012, AC-014, AC-025 | AT-006, AT-007, UT-014, IT-008, IT-010, IT-013, IT-014, IT-019, IT-020, IT-022 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-019, AC-021 | AT-001, AT-002, UT-012 | Acceptance / Unit | Yes |
| FR-013 | AC-003, AC-022 | IT-001, UT-001 | Integration / Unit | Yes |
| FR-014 | AC-024 | AT-003, UT-009 | Acceptance / Unit | Yes |
| FR-015 | AC-026 | AT-006, UT-004, UT-007, UT-008, IT-003, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-028 | MAN-001, REG-010 | Manual / Regression | Partially; live flow manual |
| NFR-001 | AC-005, AC-008, AC-009, AC-011, AC-012, AC-016 | IT-004, IT-007, IT-008, IT-009, IT-010, IT-016, IT-020, IT-021, IT-022, REG-007 | Integration / Regression | Yes |
| NFR-002 | AC-015 | IT-017, UT-011, REG-004, REG-007 | Integration / Unit / Regression | Yes |
| NFR-003 | AC-017, AC-022 | IT-001, IT-015, UT-002, REG-009 | Integration / Unit / Regression | Yes |
| NFR-004 | AC-014 | AT-006, IT-001, IT-013, UT-011 | Acceptance / Integration / Unit | Yes |
| RULE-001 | AC-003, AC-004, AC-022 | IT-001, IT-002, IT-015, UT-002 | Integration / Unit | Yes |
| RULE-002 | AC-015 | IT-017, UT-001, UT-011 | Integration / Unit | Yes |
| RULE-003 | AC-004, AC-013 | AT-005, IT-002, IT-012 | Acceptance / Integration | Yes |
| RULE-004 | AC-008, AC-009, AC-011 | IT-007, IT-008, IT-009, IT-021 | Integration | Yes |
| RULE-005 | AC-023 | IT-003, IT-018, UT-003, UT-004, MAN-001 | Integration / Unit / Manual | Yes, except live provider |
| RULE-006 | AC-025 | AT-007, UT-005, UT-014, IT-013, IT-014 | Acceptance / Integration / Unit | Yes |
| RULE-007 | AC-006, AC-029 | IT-006, REG-006 | Integration / Regression | Yes |

All acceptance criteria `AC-001` through `AC-029` have at least one verification path. No Must requirement depends solely on a manual check except `FR-016`, which explicitly requires live manual release evidence.

## 3. Acceptance Scenarios
### AT-001: New User Starts Registration From Welcome
Requirement IDs: BR-001, FR-001, FR-006, FR-012
Acceptance Criteria: AC-001, AC-019, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/welcome_page_test.dart`

```gherkin
Feature: Provider-first account registration
  Scenario: Start account creation from the signed-out welcome page
    Given the user is signed out
    And the welcome page explains in one sentence that Bluesky hosts a portable account used with Craftsky
    When the user taps "Create an account"
    Then Craftsky starts provider-first registration without asking for a handle
    And the returned authorization URL opens immediately in the external browser
    And no registration interstitial or confirmation dialog is shown
```

### AT-002: Registration And Sign-In Remain Distinct
Requirement IDs: BR-002, FR-001, FR-012
Acceptance Criteria: AC-002, AC-020, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/welcome_page_test.dart`, `app/test/auth/sign_in_page_test.dart`, `app/test/router/app_shell_account_switcher_test.dart`

```gherkin
Feature: Distinct account entry paths
  Scenario: Existing user chooses handle-first sign-in
    Given the user is signed out
    When the user taps "Sign in"
    Then Craftsky displays the existing handle form
    And submitting a handle starts handle-first OAuth

  Scenario: Signed-in user creates another account
    Given the user is signed in
    And fewer than the maximum accounts are retained
    When the user opens Add Account
    Then existing-handle sign-in remains available
    And "Create an account" is available with the Bluesky hosting explanation

  Scenario: Account limit prevents another new account
    Given the maximum number of distinct accounts are retained
    When the user opens the account switcher
    Then Add Account and provider-first registration are unavailable under the existing limit behavior
```

### AT-003: Browser Abandonment Remains Retryable
Requirement IDs: FR-006, FR-014
Acceptance Criteria: AC-007, AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/providers/auth_controller_test.dart`, `app/test/auth/welcome_page_test.dart`

```gherkin
Feature: Retry abandoned registration
  Scenario: User returns without an OAuth callback
    Given provider-first registration opened the external browser
    And no OAuth callback was received
    When the app resumes
    Then Craftsky does not treat app resume as provider cancellation
    And the registration UI remains non-blocking
    And the user can start registration again
```

### AT-004: Verified New Account Completes Craftsky Onboarding
Requirement IDs: BR-001, FR-010
Acceptance Criteria: AC-010
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/app/federated_real_flow_integration_test.go`, `app/test/auth/providers/auth_controller_test.dart`, `app/test/router/router_redirect_test.dart`

```gherkin
Feature: Complete provider-first registration
  Scenario: Newly created account is verified and handed to Flutter
    Given the provider-first request is durable and unbound to an owner
    And the provider returns a new DID whose PDS discovers the stored issuer
    When OAuth callback, profile initialization, and handoff confirmation complete
    Then the owner and OAuth session become active only after authority verification
    And the Craftsky profile is initialized
    And repository tracking and identity-cache update are requested
    And Flutter receives only approved handoff material
    And the account enters normal first-account onboarding
```

### AT-005: Existing Provider Account Is Accepted
Requirement IDs: FR-001, FR-009, RULE-003
Acceptance Criteria: AC-013, AC-020
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/app/federated_real_flow_integration_test.go`, `app/test/auth/models/session_registry_test.dart`

```gherkin
Feature: Create-or-continue provider flow
  Scenario: Provider selects an existing valid account
    Given the user started "Create an account"
    And Bluesky selected an existing account
    When its DID passes issuer authority and lifecycle checks
    Then Craftsky completes sign-in for that account
    And does not require evidence that the DID was newly created
    And if the DID is already retained locally its session is replaced and activated without consuming another account slot
```

### AT-006: Registration Errors Are Safe And Actionable
Requirement IDs: FR-011, FR-015, NFR-004
Acceptance Criteria: AC-014, AC-026
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/pages/auth_complete_page_test.dart`, `app/test/auth/providers/auth_controller_test.dart`, `app/test/auth/models/auth_error_test.dart`

```gherkin
Feature: Registration error presentation
  Scenario Outline: Flutter presents a bounded registration outcome
    Given a registration callback returns <result>
    When Flutter handles the result
    Then it displays <category>
    And it does not display issuer, DID-resolution, token, or internal lifecycle details
    And it offers only the retry behavior assigned to that condition

    Examples:
      | result                         | category                                      |
      | trusted access_denied                            | canceled                                      |
      | transport, timeout, 429, or 5xx upstream failure | provider temporarily unavailable              |
      | malformed, authority, lifecycle, or onboarding   | registration could not be verified/completed  |
```

### AT-007: Only Trusted Callback State May Reopen Flutter
Requirement IDs: FR-011, RULE-006
Acceptance Criteria: AC-025
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/handlers_render_test.go`, `appview/internal/auth/account_deletion_reauth_test.go`

```gherkin
Feature: Safe callback error routing
  Scenario: Trusted registration state identifies the handoff
    Given a provider denial or technical callback failure
    And durable registration state is valid and trusted
    When the callback is rendered
    Then a bounded non-secret error uses the stored approved handoff destination and mode

  Scenario: Callback state is untrusted
    Given callback state is missing, malformed, unknown, expired, or consumed
    When the callback fails
    Then Craftsky renders generic browser error HTML
    And no app deep link is constructed from callback input
```

### AT-008: Existing Handle Login Is Unchanged
Requirement IDs: BR-002
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance regression
Automation Target: existing AppView and Flutter auth suites

```gherkin
Feature: Handle-first login regression
  Scenario: Existing user signs in by handle
    Given provider-first registration is enabled
    When the user submits a valid existing handle through Sign in
    Then Craftsky resolves the handle and DID before OAuth
    And sends the expected account login hint
    And completes the existing lifecycle, onboarding, and handoff flow unchanged
```

### AT-009: Account Limit Race Does Not Displace An Account
Requirement IDs: FR-001
Acceptance Criteria: AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/auth/models/session_registry_test.dart`, `app/test/auth/providers/auth_controller_test.dart`

```gherkin
Feature: Registration account-limit race
  Scenario: Final account slot fills while provider OAuth is open
    Given provider-first registration started below the retained-account limit
    And another account fills the final slot before handoff
    When the returned DID is already retained
    Then its local session is replaced and activated without increasing account count

    When the returned DID is new
    Then Craftsky returns the existing account-limit outcome
    And no retained account is displaced
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, FR-013, RULE-002 | AC-003, AC-015, AC-022 | Registration API client serializes the exact wire request. | Verified-link, loopback, and dev handoff modes. | `POST /v1/auth/registrations`; camelCase handoff fields only; no handle, DID, provider, credentials, or token fields; parses `{authUrl}`. | `app/test/auth/data/auth_api_client_test.dart` |
| UT-002 | NFR-003, RULE-001 | AC-017, AC-022 | Registration provider configuration is validated and server-owned. | Missing production value, default production config, valid public HTTPS fixture origin, private/HTTP/path/query/credential origins. | Production defaults to `https://bsky.social`; valid controlled fixture accepted; unsafe/non-origin values rejected at startup. | `appview/internal/app/config_test.go` |
| UT-003 | FR-003, RULE-005 | AC-023 | Create-prompt decision follows metadata capability. | Metadata with create supported, absent, duplicate, malformed, or unrelated prompt values. | Create prompt selected only for valid advertised support; ordinary unbound request otherwise. | `appview/internal/auth/oauth_test.go` or focused new registration-flow test |
| UT-004 | FR-003, FR-015, RULE-005 | AC-023, AC-026 | Failed prompt PAR cannot downgrade. | Generic invalid request, provider error descriptions claiming unsupported prompt, DPoP failure, scope rejection, timeout, 429, and 5xx. | Exactly one PAR attempt; no provider text is parsed and no prompt-free retry occurs; bounded outcome follows the failure table. | `appview/internal/auth/oauth_test.go` or focused new registration-flow test |
| UT-005 | RULE-006 | AC-025 | Error-handoff eligibility requires trusted stored state. | Valid registration metadata for each handoff mode; unknown/expired/consumed/wrong-purpose metadata; callback-supplied destination. | Only valid trusted registration metadata can select a completion destination; callback destination input is ignored/rejected. | `appview/internal/auth/handlers_render_test.go` |
| UT-006 | FR-006 | AC-007, AC-021 | Flutter registration controller opens the returned URL immediately. | Successful `{authUrl}`, launcher success/failure. | No route/interstitial is pushed; external launcher called once; launch failure maps to recoverable error and clears unusable pending state. | `app/test/auth/providers/auth_controller_test.dart` |
| UT-007 | FR-015 | AC-026 | AppView/API registration errors map to the bounded failure table. | Trusted `access_denied`; metadata/PAR/token transport, timeout, 429, 5xx; advertised-prompt rejection; invalid metadata/endpoint; ordinary PAR OAuth/client/scope/DPoP/4xx rejection; malformed callback/token; authority/lifecycle/onboarding/browser-launch failure; untrusted state; handoff transport failure. | Exact category/retry result for each row; untrusted state produces no Flutter result; handoff retains existing receipt retry; unknown errors expose no raw body. | `app/test/auth/models/auth_error_test.dart`, `app/test/auth/providers/auth_controller_test.dart` |
| UT-008 | FR-015 | AC-026 | Auth completion page renders approved messages and retry affordances. | Each bounded registration code and handoff-recovery state. | Correct localized category appears; canceled/incomplete/provider-unavailable permit only explicit fresh start, no automatic OAuth retry; durable handoff recovery retains its existing exact-receipt retry; no technical values render. | `app/test/auth/pages/auth_complete_page_test.dart` |
| UT-009 | FR-014 | AC-024 | Pending registration state does not infer cancellation on lifecycle resume. | Registration started; app pause/resume; no callback. | State remains non-blocking/retryable; a second start is permitted; no session is created. | `app/test/auth/providers/pending_auth_provider_test.dart`, `app/test/auth/providers/auth_controller_test.dart` |
| UT-010 | FR-001 | AC-020 | Existing registry limit and duplicate-DID semantics apply before start and under a handoff race. | Four/five retained DIDs; start eligibility; flow started at four then registry reaches five; handoff for new/existing DID. | Start disabled at five; raced new sixth DID rejected without displacement; raced existing DID replaces/activates its session without increasing count. | `app/test/auth/models/session_registry_test.dart`, `app/test/auth/providers/auth_controller_test.dart` |
| UT-011 | NFR-002, NFR-004, RULE-002 | AC-014, AC-015 | Structured auth logs and errors redact registration secrets. | Authorization code, PAR URI, access/refresh token, DPoP key, handoff code, provider error text. | Captured logs/envelopes omit all secrets and raw provider text while retaining safe stage/reason and request ID. | focused AppView auth logging tests; existing `authLog*` test patterns |
| UT-012 | FR-012 | AC-019, AC-021 | Registration copy is localized and bounded to one clear sentence. | English localization and rendered Welcome/Add Account widgets. | Both surfaces render exactly “Bluesky hosts your portable account, which you can use with Craftsky.” before the registration action. | `app/test/auth/welcome_page_test.dart`, `app/test/auth/sign_in_page_test.dart` |
| UT-013 | FR-005 | AC-006 | Registration request validation mirrors existing handoff validation without handle validation. | Missing/invalid device ID, handoff mode, loopback URI, unknown JSON fields, valid builds. | Safe standard envelope; no OAuth service call on invalid admission; valid request proceeds without handle. | `appview/internal/auth/handlers_test.go` |
| UT-014 | FR-011, RULE-006 | AC-025 | Trusted failure code builder permits only bounded registration error values. | Approved codes, arbitrary provider text, URL/control characters, oversized strings. | Only approved enum-like codes serialize; arbitrary values cannot enter completion URLs. | `appview/internal/auth/handlers_render_test.go` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-002, FR-013, NFR-003, RULE-001 | AC-003, AC-022 | Registration route enforces exact API and server-owned provider contract. | Route bundle with recording registration flow and standard middleware. | POST valid/invalid bodies to `/v1/auth/registrations`; try handle/DID/provider fields and missing device ID. | Valid handoff-only request returns no-store `{authUrl}`; unsupported fields cannot select identity/provider; standard envelope/request ID on failure; no unversioned route. | `appview/internal/auth/handlers_test.go`, `appview/internal/routes/routes_test.go`, `appview/internal/routes/inventory_test.go` |
| IT-002 | FR-003, RULE-001, RULE-003 | AC-003, AC-004 | Server-first discovery and PAR omit account binding. | Real federated fixture with configured public provider origin, protected-resource and authorization-server metadata, no create prompt. | Start provider-first registration. | Metadata is validated; PAR contains required client/scopes/state/PKCE/DPoP and no `login_hint`; returned redirect uses discovered authorization endpoint. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-003 | FR-003, FR-015, RULE-005 | AC-023, AC-026 | Advertised create prompt is honored without downgrade. | Federated fixture advertises create; scenarios accept or reject the prompted PAR with generic/provider-text OAuth errors. | Start registration. | Prompt is sent when advertised; success persists usable request state; rejection produces one bounded provider-unavailable result after exactly one PAR and no prompt-free retry. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-004 | FR-004, FR-005, NFR-001 | AC-005 | Ownerless request is durable before redirect and restart-safe. | Real PostgreSQL; registration PAR succeeds. | Save request, reconstruct AppView/store, load callback metadata. | Ready row exists before response, has registration purpose/provider/issuer/handoff/device data, has no owner authority, retains PKCE/DPoP request data, and completes after restart within expiry. | `appview/internal/auth/store_test.go` |
| IT-005 | FR-004 | AC-027 | Migration enforces purpose-aware authority shapes through up/down/up. | Real PostgreSQL migrated through the new version. | Insert/update login, deletion, unbound registration, partially bound registration, and fully bound registration rows; migrate down/up. | Login/deletion require all owner fields; initial registration requires all absent; partial triples rejected; valid atomic binding accepted; indexes/constraints survive round trip. | new `appview/internal/db/provider_registration_migration_test.go` or `owner_lifecycle_migration_test.go` |
| IT-006 | FR-005, RULE-007 | AC-006, AC-029 | Registration shares auth route limiting/admission and validates handoff/device input. | Route with real `RateClassAuth` limiter plus real PostgreSQL at capacity and handler/service fixtures. | Start valid, invalid, repeated same-device, concurrent, and capacity-exhausted registrations; while registration capacity is reserved, race login and deletion request insertion. | Middleware rejects requests exceeding the configured same-device limit before provider work with bounded 429/`Retry-After`; invalid requests do not call provider; one unified reservation/request count prevents every purpose from exceeding shared hard capacity; 503 `authentication_capacity_exhausted` matches login. | `appview/internal/routes/routes_test.go`, `appview/internal/auth/auth_request_admission_test.go`, `appview/internal/auth/auth_request_capacity_handler_test.go` |
| IT-007 | FR-007, FR-008, NFR-001, RULE-004 | AC-008 | Callback accepts a DID whose PDS discovers the stored issuer. | Federated fixture returns token `sub`; authoritative DID resolves to separate PDS origin whose protected-resource metadata points to stored authorization-server origin. | Complete callback. | Exact state/`iss`/code/token checks pass; PDS and issuer need not share hostname; candidate accepted only after round-trip authority proof. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-008 | FR-008, FR-011, NFR-001, RULE-004 | AC-009 | Invalid token/identity and post-exchange fault boundaries fail closed and converge. | Table cases: missing access/refresh token, missing mandatory scope, malformed DID, DID lookup failure, missing/private PDS, mismatched authorization server, mixed DNS, timeout; fault injection immediately after exchange begin, after token receipt, during quarantine persistence, and by racing cleanup against an in-progress callback before eligibility. | Complete callback and run reconciliation/cleanup through retry/restart. | No owner/session activation; every ordinary exit leaves failed, cleanup-pending, or ambiguous state; each credential retained by a surviving callback is durably quarantined before authority work or immediately revoked if persistence fails; cleanup cannot claim before the callback deadline; hard process death after provider response decode but before quarantine commit yields bounded ambiguous evidence because lost token material cannot be revoked; stale attempts release capacity after the bounded registration interval; safe error only. | `appview/internal/app/federated_real_flow_integration_test.go`, `appview/internal/auth/registration_flow_test.go`, `appview/internal/auth/auth_request_sweeper_test.go`, `appview/internal/auth/session_cleanup_processor_test.go`, `appview/internal/federatedhttp/oauth_metadata_test.go` |
| IT-009 | FR-009, NFR-001, RULE-004 | AC-011 | Verified DID binding obeys lifecycle eligibility and owner fence. | Real PostgreSQL owners absent, departed, active, deletion-pending, deleting, terminal; concurrent lifecycle transition. | Bind verified registration after token response. | Absent owner safely created departed; departed/active admitted; deletion/terminal states rejected; no transition crosses the exclusive fence. | `appview/internal/ownerlifecycle/store_integration_test.go`, focused registration flow test |
| IT-010 | FR-007, FR-011, NFR-001 | AC-011, AC-012 | Callback state and code are one-time under retries/races. | Real PostgreSQL ready, expired, consumed, exchange-started, failed, ambiguous rows; two concurrent callbacks. | Submit callback variants and race duplicate code. | At most one exchange starts; stale/replayed/wrong-issuer/missing-code attempts fail; no duplicate active parent/child session. | `appview/internal/auth/store_test.go`, focused registration callback integration test |
| IT-011 | BR-001, FR-010 | AC-010 | Successful registration reuses onboarding and handoff finalization. | Verified new DID, PDS profile fixtures, identity cache and repository tracker recorders, real PostgreSQL handoff store. | Finalize callback, exchange handoff, persist locally, confirm. | Bluesky profile read remains optional; Craftsky profile validated/created; tracking/cache called; parent and child activate atomically only on confirmation. | `appview/internal/app/federated_real_flow_integration_test.go`, `appview/internal/auth/handoff_test.go` |
| IT-012 | FR-009, RULE-003 | AC-013 | Existing account selected through registration completes normal sign-in. | Verified active/departed Craftsky owner returned by provider-first token response. | Complete callback and handoff. | Existing owner policy applies; session activates; no newness check or duplicate owner; existing retained Flutter DID replaces/activates session. | focused registration callback integration test, `app/test/auth/models/session_registry_test.dart` |
| IT-013 | FR-011, NFR-004, RULE-006 | AC-014, AC-025 | Trusted-state denial/technical failures return bounded app errors. | Valid stored registration metadata for verified-link, loopback, and allowed dev mode; provider denial and post-state technical failures. | Invoke callback. | Approved destination comes only from stored metadata; bounded error contains no code/token/provider text; Flutter completion URL/loopback response follows mode. | `appview/internal/auth/handlers_render_test.go`, focused callback handler test |
| IT-014 | FR-011, RULE-006 | AC-025 | Untrusted callback state cannot create a deep link. | Missing, malformed, unknown, expired, consumed, wrong-purpose states with attacker-controlled URL values. | Invoke callback failure. | Generic no-store browser HTML only; CSP remains strict; no location/deep-link/loopback response derived from attacker input. | `appview/internal/auth/handlers_render_test.go`, focused callback handler test |
| IT-015 | BR-003, FR-008, NFR-003 | AC-017 | Provider fixture can change without changing completion semantics. | Run the same federated tests with Bluesky-like entryway/PDS names and a second approved fixture origin. | Start and complete provider-first flow for each configured origin. | Only discovery origin/label differs; DID proof, lifecycle, session, onboarding, and handoff assertions are identical. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-016 | FR-004, FR-005, NFR-001 | AC-016 | Abandoned registration expires and is swept under shared limits. | Real PostgreSQL ownerless ready rows and reservations before/after cutoff plus registration exchange-started/ambiguous evidence and existing login/deletion ambiguous evidence. | Run admission reclamation, stale-exchange reconciliation, and sweeper through restart. | Expired ready rows/reservations cannot callback or consume capacity; stale registration exchanges become cleanup-pending or ambiguous; registration ambiguity releases capacity after its bounded interval while evidence reaches terminal retention; login/deletion ambiguity retention is unchanged; no owner/session orphan. | `appview/internal/auth/auth_request_admission_test.go`, `appview/internal/auth/auth_request_sweeper_test.go`, `appview/cmd/appview/auth_request_sweeper_test.go` |
| IT-017 | NFR-002, RULE-002 | AC-015 | Browser, Flutter, persistence, and logs preserve credential boundary. | Successful/failing fixture flows with sentinel provider password/email strings, access/refresh tokens, DPoP key, auth code, PAR URI, handoff code. | Inspect HTTP bodies/URLs, Flutter models/storage, callback HTML, structured logs, and database purpose columns. | Provider credentials never enter Craftsky request model; PDS secrets remain server-side; callback/client get approved handoff/error only; logs omit all sentinels. | AppView auth integration tests plus Flutter API/model tests |
| IT-018 | FR-003, FR-015, RULE-005 | AC-023, AC-026 | PAR failures never fallback. | Prompt advertised/absent plus table of provider error text, generic invalid request, DPoP, scope, client auth, invalid metadata/endpoint, timeout, 429, and 5xx failures. | Start registration. | One PAR attempt only; no prompt-free downgrade or provider-text parsing; advertised-prompt rejection and transient upstream failures map to provider unavailable; invalid metadata/endpoint and ordinary non-transient no-prompt PAR failures map to registration incomplete. | `appview/internal/app/federated_real_flow_integration_test.go` |
| IT-019 | FR-007, FR-011 | AC-012, AC-025 | Explicit provider denial consumes/fails request safely. | Valid trusted registration request and callback `error`; variants with/without safe description. | Invoke callback twice. | No token exchange/session; first result returns bounded canceled error through stored handoff; provider description not echoed; replay cannot produce another usable result. | focused registration callback integration test |
| IT-020 | FR-005, FR-007, FR-011, NFR-001 | AC-006, AC-012 | Registration start/callback timeouts cancel work and fail closed. | Blocking metadata/PAR/token/DID/PDS discovery fakes with configured operation deadlines. | Start registration or complete callback past deadline. | Context cancellation reaches dependency; no late persistence/activation; trusted callback gets safe failure where available. | `appview/internal/auth/oauth_flow_timeout_test.go` |
| IT-021 | FR-004, FR-009, NFR-001, RULE-004 | AC-011, AC-027 | Verified authority attaches atomically under race. | Real PostgreSQL unbound exchange-started row, verified DID, concurrent callback/lifecycle/deletion operations. | Bind owner authority and create pending OAuth session concurrently. | Row never exposes partial owner triple; owner generation/auth epoch match locked lifecycle; stale/racing operation loses; session references exactly the attached authority. | focused registration store/flow integration test |
| IT-022 | FR-009, FR-011, NFR-001 | AC-011 | Fatal failure after verified owner binding leaves only permitted durable state. | Real PostgreSQL verified new/existing owners; injected failures at profile initialization, pending-session persistence, and handoff preparation; revocation success/ambiguous cases. | Complete callback through each fatal failure boundary. | New verified owner remains at most `departed`; an existing owner's prior lifecycle and sessions remain unchanged; no OAuth/Craftsky session created by the failed attempt becomes active; pending session is abandoned; credentials are revoked or retained only in existing ambiguous/revocation cleanup; auth-request evidence follows terminal retention. Identity-cache and repository-tracking failures remain nonfatal warnings under existing shared onboarding behavior. | focused registration callback/store integration test, `appview/internal/auth/initialize_profile_effect_test.go` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Handle-first `/v1/auth/login` request and handle/DID resolution remain unchanged. | BR-002 | AC-002, AC-018 | Run existing handler, Flutter API/controller, and federated login tests; assert login still requires handle, sends `login_hint`, and does not use configured registration provider directly. |
| REG-002 | Handle-first callback still requires token `sub` to equal the pre-resolved DID. | BR-002 | AC-018 | Existing and focused OAuth-flow tests reject mismatched `sub` without using provider-first authority rules. |
| REG-003 | Account-deletion OAuth remains owner-bound and exact-purpose. | BR-002 | AC-018 | Existing account deletion reauth, lifecycle, migration, and callback tests pass with non-null authority constraints and unchanged privileged flow. |
| REG-004 | Verified-link, loopback, and debug-only dev-scheme handoffs remain code-only. | FR-006, NFR-002 | AC-007, AC-015 | Run `handlers_render_test.go`, `dev_scheme_test.go`, Flutter verified-link configuration, and route secret-inventory tests. |
| REG-005 | Existing profile initialization and onboarding failure cleanup remain unchanged. | FR-010 | AC-010 | Run `initialize_profile_test.go`, `initialize_profile_effect_test.go`, handoff tests, and profile onboarding regression paths; assert profile failures remain fatal while identity-cache and repository-tracking failures remain warning-only. |
| REG-006 | Shared auth rate class, pending-auth capacity, expiry, and terminal retention remain bounded. | FR-004, FR-005, RULE-007 | AC-005, AC-006, AC-016, AC-029 | Existing route policy/limiter, admission, sweeper, cleanup, and config geometry tests include ownerless registration rows without changing limits. |
| REG-007 | Federated outbound SSRF, redirect, same-origin, DNS rebinding, and endpoint policies remain fail-closed. | NFR-001, NFR-002 | AC-008, AC-009, AC-011, AC-012, AC-015, AC-016 | Run `internal/federatedhttp` policy/real-listener tests and malicious metadata cases in the real-flow fixture. |
| REG-008 | Multi-account session staging, confirmation recovery, and active-account switching remain durable. | FR-001, FR-010 | AC-010, AC-020 | Existing registry/controller/account-boundary suites pass, including five-account limit and lost confirmation response recovery. |
| REG-009 | Provider-specific configuration does not enter provider-independent session records or Flutter request bodies. | BR-003, NFR-003 | AC-017, AC-022 | Serialization and two-origin fixture tests assert origin/label remain at AppView start/config boundary while OAuth/Craftsky session and handoff contracts stay provider-neutral. |
| REG-010 | Release gates do not make live Bluesky calls or create accounts. | FR-016 | AC-028 | CI/release test inventory contains only fixture hosts; full automated suite passes offline from Bluesky. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Approved production provider | Origin `https://bsky.social`, label `Bluesky`, no client-supplied override. | UT-002, IT-001, MAN-001 |
| TD-002 | Controlled provider fixture | Public synthetic origins such as `https://pds.registration.test` and `https://auth.registration.test`, routed only by the test dialer. | IT-002, IT-003, IT-007, IT-015 |
| TD-003 | Provider metadata variants | Protected-resource metadata; authorization metadata with create prompt advertised/absent/malformed; valid PAR/token/revocation endpoints. | UT-003, IT-002, IT-003, IT-018 |
| TD-004 | PAR failure classifications | Advertised-prompt rejection, provider error text claiming unsupported prompt, generic invalid request, invalid client assertion, invalid DPoP, invalid scope, timeout, 429, and 5xx. | UT-004, IT-003, IT-018 |
| TD-005 | Candidate identities | Valid DID with separate PDS/issuer, malformed DID, unresolved DID, missing PDS service, mismatched issuer, private/mixed-DNS PDS. | IT-007, IT-008 |
| TD-006 | Owner lifecycle states | Absent, departed, active, deletion-pending, deleting, and terminal owners with explicit generation/auth epoch. | IT-009, IT-012, IT-021 |
| TD-007 | Auth-request purpose shapes | Owner-bound login, owner-bound deletion, ownerless registration, partially bound registration, fully verified registration. | IT-004, IT-005, IT-021 |
| TD-008 | Callback states | Ready, expired, exchange-started, failed, ambiguous, consumed, revoked, unknown, wrong purpose, and concurrent duplicate state. | IT-010, IT-013, IT-014, IT-019 |
| TD-009 | Handoff variants | Verified-link, loopback, allowed dev scheme, wrong device, expired exchange, duplicate confirmation. | UT-005, IT-011, IT-013 |
| TD-010 | Secret sentinels | Unique provider email/password text, authorization code, PAR URI, access token, refresh token, DPoP private key, handoff code. | UT-011, IT-017, REG-004 |
| TD-011 | Flutter account registries | Zero, four, five accounts; existing-DID and new-DID pending handoffs. | AT-002, AT-005, UT-010, REG-008 |
| TD-012 | Safe registration outcomes | `canceled`, `providerUnavailable`, `registrationIncomplete`; arbitrary/oversized/provider-supplied strings. | AT-006, AT-007, UT-007, UT-008, UT-014 |

## 8. Manual Checks
| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-003, FR-016, RULE-005 | AC-001, AC-010, AC-023, AC-028 | Real Bluesky create-and-return release smoke test. | 1. Use a production-like signed Flutter build and AppView deployment with registration provider `https://bsky.social`. 2. Start signed out with no relevant Bluesky browser account if practical. 3. Verify the welcome copy and tap “Create an account.” 4. Confirm the external Bluesky interface offers account creation or its advertised create path. 5. Create a disposable test account while entering all credentials only on Bluesky-controlled pages. 6. Approve Craftsky. 7. Confirm the verified/app link returns to Flutter. 8. Confirm Craftsky onboarding completes and profile reads/writes work. 9. Inspect AppView logs and the device callback URL for secrets. 10. Record date, build/version, platform, provider metadata/prompt behavior, callback mode, outcome, secret inspection, and cleanup disposition in `release-smoke-evidence.md` in this workflow folder. | The complete journey succeeds without manual handle entry; the resulting DID is verified and signed in; only handoff material reaches Flutter; no provider credentials/tokens appear in URLs or logs. If Bluesky does not advertise/offer creation, or rejects an advertised prompt, record the incompatibility and block release rather than bypassing OAuth. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No automated real-OS browser-to-app registration test. | FR-006, FR-016 | The repository has widget/controller tests and static App/Universal Link checks but no Flutter `integration_test/` browser harness. | Keep MAN-001 mandatory. Consider a permanent integration test only after the project establishes a native integration-test environment. |
| GAP-002 | Live Bluesky registration UI and anti-abuse behavior are outside Craftsky control. | BR-001, FR-003, RULE-005 | Provider metadata, account availability, CAPTCHA, invite policy, and UI can change independently. | Fixture-test every protocol contract; record MAN-001 evidence for each release candidate; fail safely without direct XRPC fallback. |
| GAP-003 | Pinned Indigo does not expose advertised create-prompt metadata or a prompt option on `SendAuthRequest`. | FR-003, RULE-005 | Verified against `v0.0.0-20260417172304-7da09df6081d`; upstream PR #1411 remains unmerged. | Coding design must choose a stable dependency upgrade or narrow project-local adapter that preserves Indigo OAuth/DPoP behavior. UT-003/UT-004 and IT-003/IT-018 must cover it. Do not use an unmerged fork, parse provider text, or weaken no-downgrade behavior. |
| GAP-004 | Newly created DID/PDS metadata may have short propagation delays in the live service. | FR-008, FR-010 | Controlled fixtures are immediately consistent, while live identity infrastructure may not be. | Observe MAN-001. If reproduced, add bounded retry requirements before implementation rather than silently broadening callback timeouts. |

## 10. Out Of Scope
- Automated creation or retention of live Bluesky accounts in CI.
- Provider chooser, EuroSky-specific behavior, arbitrary provider input, or provider catalogue tests.
- Direct `com.atproto.server.createAccount` tests.
- Craftsky-hosted PDS account creation or credentials handling.
- Lexicon or PDS record-shape migration tests; no lexicon change is allowed.
- Guaranteeing provider UI wording or that registration is offered when provider policy disables it.
- Account migration, password recovery, email verification, CAPTCHA, invite-code, provider terms, and account-management tests.
- Detailed visual polish and responsive-layout checks beyond the presence, accessibility, and action behavior of registration controls and copy.

## 11. Handoff To Document Review
- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-08-29-provider-first-registration/`
- Recommended first failing test for implementation: `IT-005`, the real-PostgreSQL migration test proving purpose-aware owner authority constraints for login, deletion, and ownerless registration rows.
- Suggested test order for implementation: `IT-005` -> `IT-004`/`IT-021` -> `UT-002` -> `IT-001`/`IT-006` -> `UT-003`/`UT-004` -> `IT-002`/`IT-003`/`IT-018` -> `IT-007`/`IT-008`/`IT-009`/`IT-010`/`IT-022` -> `IT-011`/`IT-012` -> `UT-005`/`IT-013`/`IT-014`/`IT-019` -> Flutter `UT-001`/`UT-006` through `UT-010` -> `AT-001` through `AT-009` -> regressions -> `MAN-001`.
- Focused Flutter command: `just app-test test/auth test/router/router_redirect_test.dart test/router/oauth_handoff_route_test.dart test/router/app_shell_account_switcher_test.dart test/router/account_switch_routing_test.dart`
- Full Flutter commands: `just app-test` and `just app-analyze`.
- Focused AppView command after `just dev-d`: `cd appview && TEST_DATABASE_URL=<dev-postgres-url> TEST_DATABASE_REQUIRED=true GOTOOLCHAIN=go1.26.6 go test -race ./internal/auth ./internal/app ./internal/federatedhttp ./internal/db ./internal/ownerlifecycle ./cmd/appview`.
- Full AppView release command: `just appview-check`.
- Blocking gaps: None for document re-review. GAP-003 is a known coding-design constraint rather than an unresolved behavior; `MAN-001` blocks release, not coding.
