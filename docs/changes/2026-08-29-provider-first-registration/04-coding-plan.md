# Coding Plan: Provider-First AT Protocol Registration

## 1. Inputs
- Requirements: `01-requirements.md` (Approved, High risk)
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (Approved)
- API contract: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Architecture reference: `atproto-craft-social-app-reference.md`
- Pinned OAuth dependency: Indigo `v0.0.0-20260417172304-7da09df6081d`

## 2. Implementation Strategy
Extend the existing OAuth request, owner lifecycle, onboarding, handoff, and Flutter session paths instead of creating a parallel registration stack.

The AppView will add a `registration` OAuth purpose whose request starts without owner authority. It will reserve shared admission before provider work, discover the configured provider, persist ownerless callback state, exchange the callback code once, and durably quarantine the returned server-side credential before identity-network work. It will then verify the returned DID -> PDS -> authorization-server relationship and enter the existing exclusive onboarding owner fence. Owner authority and the pending OAuth parent session will be persisted atomically from that quarantine before the existing profile and handoff finalization runs. Failed/uninterrupted authority verification converges through bounded durable revocation without creating an owner.

Keep the pinned Indigo version. A registration-scoped adapter will decode `prompt_values_supported`, decorate only validated PAR form requests with `prompt=create`, and sanitize failed PAR/token responses before Indigo can log provider-controlled bodies. Indigo continues to generate state, PKCE, DPoP keys/proofs, client assertions, nonce retries, and `AuthRequestData`; no OAuth cryptography is reimplemented.

Flutter will extend `AuthController`, `AuthApiClient`, and `PendingAuth`; reuse the existing external-browser launcher, `/auth/complete` route, handoff clients, and `SessionRegistry`; and add one shared registration action used by Welcome and Add Account.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Auth persistence | One durable request table and parent-session cleanup | Add pre-PAR reservation, ownerless/bound registration shapes, unverified credential quarantine, and atomic authority/session binding | FR-004, FR-005, FR-009, FR-011, RULE-004 | IT-004, IT-005, IT-008, IT-016, IT-021, IT-022, REG-006 |
| Provider configuration | Validated environment-backed config | Add server-owned registration provider origin, production default `https://bsky.social` | RULE-001, NFR-003 | UT-002, IT-001, IT-015 |
| OAuth discovery/PAR | Indigo resolver and `SendAuthRequest` over hardened clients | Add extended metadata decode and registration-only prompt/error transport | FR-002, FR-003, RULE-005 | UT-003, UT-004, IT-002, IT-003, IT-018 |
| OAuth callback | Known-owner fenced callback | Branch ownerless registration, verify returned authority, then acquire owner fence | FR-007, FR-008, FR-009, NFR-001 | IT-007 through IT-010, IT-021, IT-022 |
| Onboarding/handoff | Login-only purpose checks | Admit login or registration through the same profile and handoff semantics | FR-010, RULE-003 | AT-004, AT-005, IT-011, IT-012, REG-005 |
| Failure handoff | Success code or generic browser HTML | Return three bounded errors only from trusted registration state | FR-011, FR-015, RULE-006 | AT-006, AT-007, UT-014, IT-013, IT-014, IT-019 |
| HTTP route | Anonymous login under route policy/middleware | Add strict `POST /v1/auth/registrations` under `RateClassAuth` | FR-005, FR-013, RULE-007 | UT-013, IT-001, IT-006, REG-006 |
| Flutter orchestration | One Riverpod auth controller and browser seam | Add registration start and bounded errors without a second controller | FR-006, FR-014, FR-015 | AT-003, AT-006, UT-006 through UT-009 |
| Flutter entry UI | Welcome and Add Account use sign-in route | Add direct registration action and exact provider copy | FR-001, FR-012 | AT-001, AT-002, UT-012 |
| Local account limit | `SessionRegistry.stageHandoff` protects capacity | Add pre-start UI/controller check; retain race-safe staging behavior | FR-001 | AT-009, UT-010, REG-008 |
| Credential boundary | PDS tokens remain AppView-only | Inspect all new paths and redact provider failures | NFR-002, RULE-002 | UT-001, UT-011, IT-017, REG-004, REG-007 |
| Observability | Bounded slog and `observability.MetricRecorder` | Add registration stage/result/category logs and metrics without identity/provider data | NFR-004 | UT-011, IT-017 |
| Release evidence | Fixture automation | Keep live Bluesky create/return manual and outside CI | FR-016 | MAN-001, REG-010 |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000061_provider_first_registration.{up,down}.sql` | Create | Add registration purpose, nullable authority, admission reservations, and unverified credential quarantine | FR-004, FR-005, FR-011, RULE-004 | IT-005, IT-008 |
| `appview/internal/db/provider_registration_migration_test.go` | Create | Verify up/down/up and every valid/invalid purpose shape | FR-004 | IT-005 |
| `appview/internal/auth/account_deletion_reauth.go` | Change | Add `RegistrationOAuthPurpose`, provider/issuer metadata, and purpose-aware validation helpers | FR-004, FR-009 | IT-004, IT-021 |
| `appview/internal/auth/store.go` | Change | Reserve capacity, save ownerless requests, enforce expiry, quarantine tokens, atomically bind authority/session | FR-004, FR-005, FR-009, FR-011 | IT-004, IT-008 through IT-010, IT-016, IT-021, IT-022 |
| `appview/internal/auth/registration_oauth.go` | Create | Extended metadata, prompt selection, PAR decoration, response redaction, typed stage errors | FR-003, FR-015, NFR-002, RULE-005 | UT-003, UT-004, IT-003, IT-017, IT-018 |
| `appview/internal/auth/registration_oauth_test.go` | Create | Contract-test adapter against pinned Indigo | FR-003, RULE-005 | UT-003, UT-004 |
| `appview/internal/auth/oauth_flow.go` | Change | Start registration and process ownerless callback/authority proof | FR-002, FR-007 through FR-011 | IT-002, IT-007 through IT-012, IT-020 through IT-022 |
| `appview/internal/auth/registration_flow_test.go` | Create | Registration start/callback/lifecycle tests not suited to the full federated fixture | FR-007 through FR-011 | IT-009, IT-010, IT-012, IT-021, IT-022 |
| `appview/internal/auth/session_cleanup_processor.go`, `session_cleanup_processor_test.go` | Change | Claim/revoke bounded unverified registration credentials alongside owner-bound parents | FR-011, NFR-001 | IT-008, IT-022 |
| `appview/internal/auth/auth_request_sweeper.go`, `auth_request_sweeper_test.go` | Change | Reconcile stale registration exchanges/reservations and release capacity while retaining bounded evidence | FR-005, FR-011, NFR-001 | IT-006, IT-008, IT-016 |
| `appview/internal/auth/registration_security_test.go` | Create | Capture logs, metrics, envelopes, URLs, and persistence for secret sentinels | FR-011, NFR-002, NFR-004 | UT-011, IT-017 |
| `appview/internal/auth/callback_attempt.go` | Change | Admit registration as ordinary onboarding authority after binding | FR-009 | IT-021 |
| `appview/internal/auth/initialize_profile.go` | Change | Permit registration purpose; retain fatal profile and warning-only cache/tracker behavior | FR-010 | IT-011, IT-022, REG-005 |
| `appview/internal/auth/handoff.go` | Change | Permit bound registration purpose without broadening deletion | FR-010 | IT-011, IT-012 |
| `appview/internal/auth/handlers_session.go` | Change | Add strict registration request handler and safe start error envelopes | FR-013, FR-015 | UT-013, IT-001 |
| `appview/internal/auth/handlers_oauth.go` | Change | Finalize registration and render trusted bounded callback failures | FR-010, FR-011, RULE-006 | IT-011, IT-013, IT-014, IT-019 |
| `appview/internal/auth/handlers_render.go` | Change | Render mutually exclusive code/error deep-link or loopback payloads | FR-006, FR-011 | AT-007, IT-013, IT-014 |
| `appview/internal/app/config.go`, `config_test.go` | Change | Parse validated `OAUTH_REGISTRATION_PROVIDER_ORIGIN` | RULE-001, NFR-003 | UT-002 |
| `appview/internal/app/deps_auth.go`, `routes_adapter.go` | Change | Construct/inject registration adapter and provider configuration | NFR-003 | IT-001, IT-015 |
| `appview/internal/app/deps_workers.go`, tests | Change | Inject quarantine cleanup/reconciliation dependencies and observer into production workers | FR-011, NFR-004 | IT-008, IT-017, IT-022 |
| `appview/internal/routes/policy.go`, `routes_public_auth.go`, route inventories/tests | Change | Register anonymous rate-limited route | FR-005, FR-013, RULE-007 | IT-001, IT-006, REG-006 |
| `appview/internal/auth/auth_request_capacity_handler_test.go`, `account_deletion_reauth_test.go` | Change | Extend coordinator fakes and preserve existing capacity/deletion contracts | BR-002, FR-005 | IT-006, REG-002, REG-003 |
| `appview/internal/app/federated_real_flow_integration_test.go` | Change | Add full provider-first fixture paths and two-provider-neutrality run | BR-001, BR-003, FR-003, FR-008 | IT-002, IT-003, IT-007, IT-008, IT-015, IT-018 |
| `appview/internal/federatedhttp/oauth_metadata_test.go` | Change | Retain destination/metadata fail-closed coverage | FR-003, FR-008, NFR-001 | IT-008, REG-007 |
| `appview/internal/observability/metric_recorder.go`, `observer.go`, tests | Change | Add bounded registration operation metrics and in-memory/Sentry implementations | NFR-004 | UT-011, IT-017 |
| `appview/environments/*.env*`, `appview/README.md` | Change | Document production default and controlled override | RULE-001 | UT-002 |
| `app/lib/auth/data/auth_api_client.dart` | Change | Add `register()` using existing `{authUrl}` model | FR-013 | UT-001 |
| `app/lib/auth/models/pending_auth.dart`, generated mapper | Change/regenerate | Represent sign-in versus registration pending hints | FR-014 | UT-009 |
| `app/lib/auth/models/auth_error.dart` | Change | Add strict three-value registration failure model/parser | FR-015 | UT-007, UT-014 |
| `app/lib/auth/providers/auth_controller.dart` | Change/regenerate | Start registration, map errors, launch browser, precheck capacity | FR-006, FR-014, FR-015 | UT-006 through UT-010 |
| `app/lib/auth/providers/pending_auth_provider.dart` | Change/regenerate | Add explicit `startSignIn` and `startRegistration` transitions | FR-014 | UT-009 |
| `app/lib/auth/widgets/registration_action.dart` | Create | Share exact disclosure/action presentation across two entry surfaces | FR-001, FR-012 | AT-001, AT-002, UT-012 |
| `app/lib/auth/pages/welcome_page.dart` | Change | Keep sign-in route; start registration directly | FR-001, FR-006 | AT-001 |
| `app/lib/auth/pages/sign_in_page.dart` | Change | Add registration only in Add Account mode and enforce limit state | FR-001, FR-012 | AT-002, UT-010, UT-012 |
| `app/lib/auth/pages/auth_complete_page.dart` | Change | Parse bounded callback errors; separate fresh OAuth retry from handoff retry | FR-011, FR-015 | AT-006, AT-007, UT-008 |
| `app/lib/shared/api/providers/session_auth_interceptor.dart` | Change | Defensively keep registration start anonymous while retaining device ID | NFR-002 | IT-017, REG-004 |
| `app/lib/shared/api/providers/error_mapping_interceptor.dart` | Change | Add safe registration endpoint category | NFR-004 | UT-011 |
| `app/test/shared/api/providers/session_auth_interceptor_test.dart`, `error_mapping_interceptor_test.dart` | Change | Verify anonymous device header and safe endpoint/error categorization | NFR-002, NFR-004 | UT-011, IT-017 |
| `app/lib/l10n/app_en.arb`, generated localizations | Change/regenerate | Add exact disclosure and bounded failure copy | FR-012, FR-015 | UT-008, UT-012 |

## 5. Services, Interfaces, And Data Flow

### Provider Configuration
Add `OAuthRegistrationProviderOrigin url.URL` to `app.Config`. Parse `OAUTH_REGISTRATION_PROVIDER_ORIGIN`, defaulting to `https://bsky.social`, through a generalized form of `parseCanonicalPublicOrigin`. Reject credentials, path/query/fragment, ports, IP literals, non-HTTPS schemes, and non-public DNS names at startup. Runtime DNS and endpoint checks still go through `federatedhttp.Boundary`.

Flutter never receives or submits this origin.

### Registration OAuth Adapter
Use a focused interface so OAuth flow tests can inject deterministic behavior:

```text
type RegistrationOAuth interface {
    ResolveMetadata(ctx, issuer) (RegistrationAuthMetadata, error)
    SendAuthorizationRequest(ctx, metadata, scopes) (oauth.AuthRequestData, error)
    SendInitialTokenRequest(ctx, code, request) (oauth.TokenResponse, error)
}

type RegistrationAuthMetadata struct {
    OAuth                  oauth.AuthServerMetadata
    PromptValuesSupported []string
}
```

Implementation details:
- Use Indigo `Resolver.ResolveAuthServerURL` with the hardened metadata client to resolve the configured provider service to an issuer.
- Fetch authorization metadata once through the same hardened client into an anonymous embedding of `oauth.AuthServerMetadata` plus `prompt_values_supported`; reject malformed field types and call Indigo `Validate(issuer)`.
- Exact, case-sensitive `create` membership selects the prompt. Missing/empty/unrelated values omit it.
- Shallow-copy `oauth.ClientApp` and its `http.Client`; wrap only the copy's transport. Never mutate the shared app/client.
- On the validated PAR endpoint, require form POST, absence of `login_hint`, and absence of a preexisting prompt; add only internal constant `prompt=create` when selected. Replace body, `GetBody`, and content length while preserving headers, including `DPoP`.
- DPoP and client assertions remain valid because pinned Indigo binds DPoP to method/URL and the client assertion to client/audience, not the form body. Contract tests lock this assumption to the pinned version.
- Apply the same registration-scoped redaction transport to initial token exchange. For non-success responses, consume the bounded body and return a typed error containing only stage/status/transience/prompted facts.
- Preserve Indigo's one DPoP nonce replay by passing back only sanitized `{"error":"use_dpop_nonce"}` when HTTP 400 has a `DPoP-Nonce` and the exact top-level error code. The replay retains the same prompt. No other automatic PAR/token retry is added.
- Never inspect `error_description`, HTML, or prose. Never retry a failed prompted PAR without the prompt.

Do not upgrade Indigo in this change. Current upstream still lacks complete metadata/request support, and a broad pseudo-version update would not remove the response-redaction requirement.

### Auth-Request Persistence
Migration `000061` will:
- Add `registration` to the purpose constraint.
- Make `owner_did`, `owner_generation`, and `auth_epoch` nullable.
- Add nullable `registration_provider_origin` and `registration_issuer` columns.
- Enforce the owner authority triple as all-null or all-present/positive.
- Require login rows to have owner authority and no deletion/registration metadata.
- Require account-deletion rows to keep complete owner/deletion authority and no registration metadata.
- Require initial registration rows to have provider/issuer and no owner/deletion authority; permit a bound registration row only with the complete owner triple.
- Keep the nullable owner foreign key and make owner-dependent indexes partial where appropriate.
- Add `oauth_auth_request_reservations` with an opaque ID and creation/expiry timestamps only. Active reservations and pending requests share one admission count.
- Add `oauth_unverified_credentials`, keyed by request state, with `oauth.ClientSessionData` under the same protected server-side storage controls as `oauth_sessions`, bounded status/lease/attempt/next-attempt fields, and expiry. It deliberately has no owner foreign key because it exists before authority proof.
- Add registration-only `cleanup_pending` to the auth-request state constraint. It means a quarantine exists and revocation owns the attempt; login/deletion cannot enter it.
- Delete registration request rows before the down migration restores old non-null/old-purpose constraints. Provider-neutral owner/session rows need no rollback transformation.

Extend `AuthRequestMetadata` with provider and issuer values plus purpose-aware methods such as `validReady()`, `isUnboundRegistration()`, and `hasOwnerAuthority()`. Do not treat zero-value DID/generation/epoch as valid bound authority.

Add `ReserveAuthRequestCapacity`, `ReleaseAuthRequestCapacity`, and a reservation-aware registration save path:
- Reservation acquires the existing advisory admission lock, reclaims expiry, and counts active reservations plus the same pending request states under the same hard capacity.
- `StartRegistration` reserves before discovery/PAR. Every error before request persistence releases idempotently; abandoned reservations expire after the configured start-operation timeout plus one sweeper interval, bounded well below `AuthRequestExpiry`.
- Saving a registration request consumes its opaque reservation and inserts the request in one transaction. No unreserved registration request can be inserted.
- Every login, deletion, and registration insertion path acquires the same advisory lock and runs the same unified admission query over active reservations plus pending request rows. Existing login/deletion callers retain their owner fence and `WithAuthTransaction` path, but cannot consume a slot reserved by registration. Reservation expiry reclamation runs under that lock before every purpose's admission decision.

The purpose-aware coordinator surface is explicit rather than adding zero-value behavior to owner-required methods:

```text
ReserveAuthRequestCapacity(ctx) (AuthRequestReservation, error)
ReleaseAuthRequestCapacity(ctx, reservationID) error
SaveRegistrationAuthRequest(ctx, reservationID, request) error
BeginRegistrationExchange(ctx, state) (AuthRequestInfo, error)
QuarantineRegistrationCredential(ctx, state, sessionData, eligibleAt) error
MarkRegistrationCredentialForCleanup(ctx, state, category) error
BindRegistrationAuthority(ctx, tx, binding) error
FinishRegistrationExchange(ctx, state, attemptID, outcome) error
ReconcileStaleRegistrationExchanges(ctx, batch) (stats, error)
```

Reservation release, quarantine cleanup eligibility, and cleanup completion are idempotent. The quarantine write and request association are one transaction. Authority binding consumes only a `held` quarantine associated with the same state/attempt and exact stored issuer.

The state model is explicit:
- `ready -> exchange_started` is the one-time callback claim.
- Successful fenced authority/session binding leaves the request `exchange_started`; `exchange_started -> consumed` occurs only when existing handoff preparation commits.
- `exchange_started -> exchange_failed` covers definite no-token failures.
- `exchange_started -> cleanup_pending` atomically accompanies a quarantine becoming revocation-eligible.
- `exchange_started -> exchange_ambiguous` covers indeterminate issuance with no stored credential.
- `cleanup_pending -> revoked` follows successful cleanup; retry remains `cleanup_pending`; bounded cleanup exhaustion becomes `exchange_ambiguous` with `exchange_finished_at`.

Admission counts unexpired `ready`, all `exchange_started`, all `cleanup_pending`, and `exchange_ambiguous` registration rows still inside their bounded ambiguity interval. Terminal retention deletes registration failed/revoked/ambiguous evidence only after the configured terminal cutoff. A request and quarantine are locked in consistent request-then-quarantine order for binding, cleanup claim, and reconciliation CAS operations.

Add an ownerless `BeginRegistrationExchange` using a pool transaction. Add expiry predicates to metadata trust and both exchange transitions so stale ready rows cannot be consumed. Login/deletion continue using their existing fenced `BeginExchange` semantics.

### Registration Callback And Atomic Binding
Extend the coordinator without changing existing callers:

```text
type OAuthFlowCoordinator interface {
    StartLogin(...)
    StartRegistration(ctx, mode, loopbackURI, deviceID) (authURL, error)
    CompleteCallback(ctx, params, finalizer) error
}
```

`StartRegistration`:
1. Validate handoff/device input and configured provider.
2. Reserve shared pending-request capacity before provider work.
3. Resolve provider service -> issuer -> extended metadata.
4. Send one server-first PAR with no `login_hint`; add `prompt=create` only when advertised.
5. Consume the reservation while saving `AuthRequestData` with registration purpose, provider, exact issuer, handoff, and device metadata but no owner authority.
6. Return the authorization endpoint plus `client_id` and `request_uri` only after persistence succeeds.
7. Release the reservation on every failure before step 5; defer-based cleanup makes this true for all ordinary error paths.

`CompleteCallback` dispatches by trusted stored purpose. Login and deletion retain their current known-owner fence and subject checks. Registration:
1. Load unexpired ready metadata before trusting any handoff destination.
2. Atomically enter `exchange_started`. From this point, a deferred completion guard finishes every ordinary return path as bound, failed, cleanup-pending, or ambiguous; it is disarmed only by a successful compare-and-set transition.
3. Explicit `access_denied`, wrong/missing issuer, and missing code perform no token exchange, transition to `exchange_failed`, and return the applicable bounded result.
4. Send one token exchange. A definite protocol rejection transitions to `exchange_failed`; timeout, connection loss, malformed success, or any response for which token issuance cannot be disproved transitions to `exchange_ambiguous` unless a returned credential is durably quarantined.
5. Validate the complete token response before authority use: parse a nonempty DID `sub`; require nonempty access token, refresh token, and approved scope; require the approved scope set to include every mandatory requested scope, including `atproto`; and reject blank, duplicate, or malformed scope elements. A malformed response is never a usable session.
6. If a malformed response contains any revocable token, persist it to the cleanup quarantine before returning failure. Never log or echo token-response fields.
7. For a valid response, first persist `oauth.ClientSessionData` to protected server-side `oauth_unverified_credentials` with status `held`. Use the configured protected-resource origin as temporary `HostURL`, retain the validated authorization-server endpoints, DPoP key, and all tokens, and set `eligible_at` to the callback start plus the configured callback-operation timeout and one sweeper interval. The worker cannot claim `held` rows before that timestamp.
8. If quarantine persistence fails after token receipt, retry the short local database write within the callback deadline. If it still fails, attempt immediate provider revocation from the in-memory credential, mark the request failed when revocation is confirmed, otherwise mark it ambiguous and emit a bounded critical cleanup signal. No owner/session may be created from an unquarantined credential.
9. Resolve the DID through `authoritativeDirectory`, require its PDS endpoint, discover that PDS's authorization server, and compare it exactly to the stored issuer. PDS and issuer origins may differ.
10. On any authority failure, atomically change the quarantine from `held` to `pending`, set it immediately eligible, and change the request from `exchange_started` to `cleanup_pending`. Do not create an owner just to store an unverified credential.
11. Enter `Owners.WithOnboardingAuth(candidateDID)`, require `departed` or `active`, and create a bound `CallbackAttempt` from the fenced lifecycle generation/epoch.
12. In one fenced transaction, lock/read the quarantine, replace its temporary host with the verified PDS, change the request from ownerless to the complete owner triple, insert the `pending_handoff` OAuth parent from the stored credential, and delete the quarantine. The compare-and-set requires registration purpose, exact attempt ID, `exchange_started`, matching issuer, and an all-null prior authority triple.
13. Run the shared profile/onboarding and handoff finalizer inside the fence. Fatal failure abandons only the new pending parent; a new owner may remain `departed`, while an existing owner's prior lifecycle and sessions remain unchanged.

`session_cleanup_processor.go` also claims `pending` rows, or expired `held` rows while atomically moving their request to `cleanup_pending`, using the existing bounded lease/backoff/retention policy. Successful provider revocation deletes the quarantine and changes the request to `revoked`. Retry keeps both durable states. Exhaustion changes the request to `exchange_ambiguous`, records `exchange_finished_at`, emits bounded failure telemetry, and applies the existing retention policy to the protected credential. Binding and cleanup both lock request then quarantine, so only one can consume a row; a callback operating within its deadline cannot race cleanup eligibility.

The auth-request sweeper additionally reconciles registration-only `exchange_started` rows older than the configured callback-operation timeout plus one sweeper interval. If a `held` quarantine exists it atomically changes quarantine/request to `pending`/`cleanup_pending`; otherwise it transitions the request to `exchange_ambiguous` and emits the irrecoverable-exchange signal. Registration ambiguous rows count against admission for one `AuthRequestExpiry` interval from `exchange_finished_at`, then stop consuming capacity while their non-secret evidence remains until `AuthRequestTerminalRetention`. This does not change login/deletion ambiguity retention. Therefore every known callback exit and stale process interruption converges. The unavoidable untracked-token window includes provider issuance where the AppView never receives the response and hard process death after response decode but before quarantine commit; both leave stale attempt evidence but no token material that could be revoked.

`CallbackAttempt.validFor`, profile initialization, pending PDS client creation, and handoff creation should use one narrow `permitsOrdinaryOnboarding()` predicate for login/registration. Deletion remains excluded.

Profile read/validation/write failures remain fatal. Identity-cache and repository-tracking failures remain warning-only, matching existing login behavior and `REG-005`.

### Trusted Failure Result
Define a bounded internal value:

```text
type RegistrationFailureCode string
const canceled, providerUnavailable, registrationIncomplete

type TrustedRegistrationFailure struct {
    Metadata AuthRequestMetadata
    Code RegistrationFailureCode
    Cause error // internal only; Error() is redacted
}
```

Only registration processing after a valid unexpired ready-state load may construct it. The handler uses stored handoff mode/destination; callback input cannot supply a destination. Missing, malformed, unknown, expired, wrong-purpose, replayed, or consumed state always gets generic no-store browser HTML.

`callbackPageData` gains a bounded error field and validates exactly one of code/error. Verified-link and dev-scheme URLs use `?error=<boundedValue>`; loopback posts exactly `{"error":"<boundedValue>"}`. Success remains code-only. Preserve CSP, contextual escaping, exact loopback origin, and no-store headers.

### API Handler
Add `RegistrationHandler()` with request fields `handoffMode` and optional `loopbackRedirectUri`. Use existing `decodeSingleJSON` so unknown fields and trailing JSON fail. Share handoff validation logic without changing login's accepted body behavior.

Programmatic `/v1/` errors keep snake_case codes:
- `registration_provider_unavailable` for transient provider failures and rejected advertised prompt PAR.
- `registration_incomplete` for invalid metadata/endpoints and non-transient ordinary PAR failures.
- Existing `authentication_capacity_exhausted`, validation, and rate-limit envelopes remain unchanged.

Callback/deep-link values remain the approved camelCase enum values because they are not `/v1/` error-envelope codes.

## 6. State, Providers, Controllers, Or DI

### AppView DI
`newAuthDependencies` constructs the registration adapter from the existing hardened metadata/OAuth clients and injects it plus the configured provider origin into `OAuthFlowServiceOptions`. The same `OAuthFlowService` remains the callback coordinator and gains registration start. Route adapters add no second session store or owner service.

`deps_workers.go` constructs the quarantine-capable cleanup processor and stale-exchange reconciler from the same auth store, revoker, lifecycle timings, logger, and `observability.Observer` already owned by app dependencies. The existing worker group starts/stops them with the application context; no callback-spawned goroutine owns cleanup. Worker-construction tests prove quarantine rows are processed in the production graph, not only in store tests.

### Flutter Provider Graph
No new Riverpod provider is needed:

```text
WelcomePage / SignInPage(addAccount) / AuthCompletePage retry
  -> authControllerProvider.startRegistration()
     -> sessionRegistryProvider.future (pre-start limit)
     -> authApiClientProvider.register()
        -> anonymousDioProvider + stable device header
     -> pendingAuthProvider.startRegistration()
     -> authUrlLauncherProvider(externalApplication)

/auth/complete?code=...
  -> existing completeFromDeepLink -> handoff exchange/stage/confirm

/auth/complete?error=...
  -> strict RegistrationFailure parser -> fresh start only on explicit user action
```

Add `PendingAuthPurpose { signIn, registration }`, named model constructors, nullable handle only for registration, and explicit notifier methods `startSignIn`/`startRegistration`. `startedAt` remains a UI hint; app lifecycle resume performs no cancellation.

`AuthController.startRegistration()`:
- Await the registry and reject before API work at five retained accounts.
- Call `AuthApiClient.register()` with handoff mode only.
- Map snake_case start errors/network failures into the three local categories without exposing server messages.
- Set pending registration immediately before launch.
- Launch through `authUrlLauncherProvider` with `externalApplication`.
- Clear pending and return `registrationIncomplete` if launch returns false/throws.
- Return to non-loading state after launch; do not wait for callback and allow explicit restart.

Keep the existing `SessionRegistry.stageHandoff` race authority: at five accounts it rejects a new DID without mutation but permits replacement/activation of an already-retained DID. Do not change the registry algorithm.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces
- `WelcomePage`: “Sign in” still navigates to the handle form. The registration action displays exactly “Bluesky hosts your portable account, which you can use with Craftsky.” and invokes `startRegistration` directly with no route/dialog/interstitial.
- `SignInPage(addAccount)`: retain handle sign-in, add the shared registration action, watch registry capacity, and disable both account-add paths at the limit. Ordinary `SignInMode.signIn` remains unchanged.
- `RegistrationAction`: presentation-only shared widget for the disclosure, action, and loading/disabled state; controller access remains in parent callbacks.
- `AuthCompletePage`: preserve `account_deletion_pending`; strictly parse the three approved errors; never exchange an unknown/error callback; show bounded localized copy; clear recognized pending UI state; and let explicit retry call `startRegistration` for a fresh OAuth request.
- Guard the existing loading -> data Feed navigation so it runs only after a nonempty handoff code completion. A successful fresh browser launch from an error page must not navigate to Feed.
- Preserve exact-code/receipt retry for handoff network/storage failures. Never translate durable handoff recovery into fresh registration.
- No `/register` route and no native link configuration change are needed. Existing `AuthCompleteRoute(code, error)` supports both first-account and Add Account returns.
- Add localization keys for the exact provider sentence, three bounded outcomes, and create/retry actions; regenerate localization output.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Invalid body/device/handoff | Standard 400 envelope; no provider call | FR-005, FR-013 | UT-013, IT-001, IT-006 |
| Rate/capacity rejection | Existing middleware 429 or store 503 with `Retry-After` before provider work | RULE-007 | IT-006, REG-006 |
| Metadata/endpoint invalid or ordinary PAR 4xx | `registration_incomplete` API error; no retry/downgrade | FR-015, RULE-005 | UT-004, IT-018 |
| Transport/timeout/429/5xx or prompted PAR rejection | `registration_provider_unavailable`; fresh user retry only | FR-015 | AT-006, UT-007, IT-003, IT-018, IT-020 |
| Browser abandonment/app resume | UI remains enabled; server request expires normally | FR-014 | AT-003, UT-009, IT-016 |
| Browser launch failure | Clear pending UI state; `registrationIncomplete` | FR-006, FR-015 | UT-006, UT-007 |
| Trusted `access_denied` | Consume/fail attempt and return bounded `canceled` | FR-011, FR-015 | AT-006, IT-019 |
| Unknown/stale/replayed callback state | Generic browser HTML; no Flutter destination | RULE-006 | AT-007, IT-010, IT-014 |
| Token response malformed | Reject before authority lookup; durably quarantine any returned revocable token; no owner/session activation | FR-008, FR-011 | IT-008, IT-017 |
| Token subject/PDS/issuer invalid | Make durable quarantine cleanup-eligible; no owner/session activation; bounded incomplete result | FR-008, FR-011 | IT-008, IT-022 |
| Process/binding failure after exchange | Grace expiry or immediate eligibility lets cleanup worker revoke the durable quarantine | FR-011, NFR-001 | IT-008, IT-022 |
| Quarantine write failure | Bounded local retry, then immediate in-memory revocation; failed revocation leaves ambiguous evidence/critical signal and no owner | FR-011, NFR-001 | IT-008 fault case C |
| Stale exchange without quarantine | Registration-only reconciliation marks ambiguity, emits signal, and releases capacity after bounded interval | FR-005, FR-011 | IT-008 fault cases A-B |
| Ineligible owner | Preserve lifecycle fence; no attempt session activation | FR-009 | IT-009, IT-021 |
| Fatal failure after binding | Abandon attempt parent; new owner may remain departed; existing owner/sessions unchanged | FR-009, FR-011 | IT-022 |
| Cache/tracker failure | Log bounded warning and continue shared onboarding | FR-010 | REG-005 |
| Limit reached before start | Disable UI/controller rejects before API call | FR-001 | AT-002, UT-010 |
| Limit reached during browser flow | Existing DID replaces session; new DID rejected without displacement/confirmation | FR-001 | AT-009, UT-010, REG-008 |
| Handoff transport/storage failure | Preserve existing exact receipt/code recovery | FR-006, FR-015 | UT-008, REG-004, REG-008 |

### Observability And Privacy
Add one bounded registration observation operation to `observability.MetricRecorder` and expose it through `Observer`. The auth service records only fixed enums:
- stage: `admission`, `metadata`, `par`, `callback`, `token`, `authority`, `binding`, `cleanup`, `handoff`;
- result: `success`, `failure`, `retry`, `rejected`;
- category: the approved failure/capacity/validation categories, never provider text.

The same bounded values may appear in structured `slog` records and metric attributes. Durations and counts are allowed. DID, handle, device ID, provider/PDS/issuer host, callback code, request URI, token fields, DPoP material, handoff values, response bodies, and raw errors are forbidden. `registration_security_test.go` injects unique sentinels across request, provider response, token response, callback, and storage data, then inspects API/browser output, redirect URLs, local/external logs, metric calls, captured error events, and persisted non-secret columns. Cleanup failure/retry metrics provide the alert input for unverified credentials approaching retention exhaustion.

## 9. Test Implementation Plan
| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-005, IT-008 | `internal/db/provider_registration_migration_test.go` | Real PostgreSQL up/down/up at migration 61 | Registration authority/reservation/quarantine shapes absent |
| 2 | IT-004, IT-006, IT-016, IT-021, REG-006 | `internal/auth/store_test.go`, admission/sweeper suites | Unified reservations plus login/deletion/registration insertion, expiry, atomic bind races | Capacity is consumed only after PAR and current admission cannot count reservations |
| 3 | UT-002 | `internal/app/config_test.go` | Production default and invalid/fixture origins | No registration provider config exists |
| 4 | UT-003, UT-004 | `internal/auth/registration_oauth_test.go` | Pinned Indigo app with recording transports and sentinel errors | Metadata prompt ignored; no decorator/redaction seam |
| 5 | IT-001, IT-006, UT-013 | Handler/route policy/inventory suites | Recording flow, strict bodies, real limiter/capacity | Route and coordinator method absent |
| 6 | IT-002, IT-003, IT-018 | Federated real-flow fixture | Advertised/absent prompt, nonce, status/error tables | Current flow requires handle/login hint |
| 7 | IT-007 through IT-010, IT-020, IT-021 | Registration flow, sweeper, cleanup processor, and federated fixtures | `IT-008` fault cases after begin, after token receipt, and during quarantine write; token shape table; durable quarantine; returned authority variants | Callback requires prebound owner and has no pre-authority credential cleanup/reconciliation |
| 8 | IT-011, IT-012, IT-022, REG-005 | Profile/handoff/registration flow tests | New/existing owner, fatal effects, warning-only effects | Login-only purpose guards reject registration |
| 9 | UT-005, UT-014, IT-013, IT-014, IT-019 | Render/callback handler suites | All handoff modes and trusted/untrusted errors | Callback can render only success code/generic HTML |
| 10 | UT-001, UT-011, IT-017, REG-004, REG-007 | Flutter API/interceptor plus AppView sentinel/observability tests | Exact body/header/log/URL/persistence/metric inventory | Registration request/model and bounded telemetry coverage absent |
| 11 | UT-007, UT-009 | Flutter model/provider tests | Bounded error values and pending purposes | Models are sign-in-only |
| 12 | UT-006, UT-008, UT-010 | `auth_controller_test.dart`, registry tests | Launcher, capacity, race, fresh versus receipt retry | Controller has no registration operation |
| 13 | AT-001, AT-002, AT-003, UT-012 | Welcome/SignIn widget tests | Four/five accounts, exact copy, launcher fake | Both Welcome actions still navigate to sign-in |
| 14 | AT-006, AT-007 | AuthComplete widget/router tests | Three bounded errors, unknown value, valid code | Error callbacks collapse to missing-code/generic state |
| 15 | AT-004, AT-005, AT-008, AT-009, REG-001 through REG-003, REG-008, REG-009 | End-to-end fixture and regression suites | New/existing accounts, handle login/deletion, two providers, limit race | Full additive behavior not yet integrated |
| 16 | REG-010 | Offline release inventory/full suites | Network trap/fixture-only provider hosts | Any accidental live Bluesky dependency is detected |
| Release | MAN-001 | `release-smoke-evidence.md` | Signed production-like build and disposable Bluesky account | Manual gate intentionally cannot run in CI |

Focused commands during implementation:

```text
just dev-d
cd appview && TEST_DATABASE_URL=<dev-postgres-url> TEST_DATABASE_REQUIRED=true GOTOOLCHAIN=go1.26.6 go test -race ./internal/db ./internal/auth ./internal/app ./internal/federatedhttp ./internal/routes ./internal/ownerlifecycle ./cmd/appview
just app-test test/auth test/shared/api/providers test/router/router_redirect_test.dart test/router/oauth_handoff_route_test.dart test/router/app_shell_account_switcher_test.dart test/router/account_switch_routing_test.dart
```

Generation and full gates:

```text
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter gen-l10n
just appview-check
just app-test
just app-analyze
```

## 10. Sequencing And Guardrails
- First TDD step: Write `IT-005` for migration 61, observe current schema reject ownerless registration, then implement only the migration needed to pass it.
- Dependencies: schema -> capacity reservation/quarantine/cleanup -> store atomic bind -> provider config -> OAuth adapter -> route/start -> authority callback -> onboarding/handoff -> trusted failures/observability -> Flutter API/state -> Flutter UI -> full regressions.
- Keep database and owner-fence lock order unchanged: owner fence before owner lifecycle/request/session rows once a verified DID exists; ownerless admission uses only the existing global auth-request admission lock.
- Keep remote metadata/token/DID/PDS work outside short database transactions. Hold the owner fence only after returned authority is verified, matching existing bounded callback behavior.
- Persist each returned credential to protected server-side quarantine before DID/PDS network work. Successful authority binding must consume that exact stored credential; no in-memory-only credential may become a parent session.
- Do not call Indigo `ProcessCallback`; retain the current explicit token/session persistence sequence so authority is verified before owner binding.
- Do not copy Indigo's PAR implementation, implement OAuth cryptography, use a generic OAuth library, depend on PR #1411, or upgrade to an unrelated pseudo-version.
- Do not parse provider descriptions, echo callback errors, or log authorization codes, request URIs, tokens, DPoP material, handoff secrets, provider credentials, or provider response bodies.
- Do not change handle-first login or deletion request contracts. Purpose predicates must name registration explicitly rather than treating every non-deletion purpose as ordinary.
- Do not activate an owner/session before returned authority proof and handoff confirmation. Do not alter an existing owner's previous sessions when a new attempt fails.
- Do not make identity-cache or repository-tracking failures fatal; preserve provider-neutral onboarding behavior.
- Do not add a provider body field, provider chooser, registration route/interstitial, direct account XRPC, PDS token storage in Flutter, lexicon change, or live-provider CI call.
- Generated Dart/localization files change only through their generators.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | PAR form decoration depends on pinned Indigo's form encoding and DPoP body-independence | Future Indigo changes could invalidate the adapter | Pin behavior with transport contract tests; re-evaluate adapter on every Indigo upgrade |
| CPQ-002 | Non-blocking | Indigo logs provider bodies on failed PAR/token responses | Secret/provider text exposure if a registration request bypasses the wrapper | Registration app clone wraps both endpoints; sentinel log tests fail on leakage |
| CPQ-003 | Non-blocking | DPoP nonce is an allowed protocol replay despite the one-PAR product rule | Tests may miscount logical versus HTTP attempts | Define one logical PAR with at most Indigo's same-body nonce replay; prompt remains unchanged |
| CPQ-004 | Non-blocking | Newly created DID/PDS discovery may be briefly inconsistent | Live signup can fail after token exchange | Fail closed now; observe MAN-001 before adding any bounded retry requirement |
| CPQ-005 | Accepted risk | Device-ID rotation bypasses local per-device auth limit | Shared capacity/provider load can be exhausted | Preserve hard shared capacity and monitoring; no new source/distributed limiter in this pass |
| CPQ-006 | Release blocker | Bluesky metadata/UI/anti-abuse behavior is external | Fixture success cannot prove live creation/return | MAN-001 and recorded evidence are mandatory before release |
| CPQ-007 | Accepted risk | A process can fail after provider token issuance but before quarantine commit, including after local response decode | The AppView cannot revoke token material lost with the process | Enter `exchange_started` before exchange, quarantine as the first surviving post-response action, reconcile stale state to bounded ambiguity, and emit cleanup alerts; no owner/session activation is possible |

Blocking implementation questions: None.

## 12. Handoff To TDD Builder
- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-005`, migration 61 authority/reservation/quarantine shapes.
- First focused target: `appview/internal/db/provider_registration_migration_test.go`.
- First focused command: run that package against the isolated/required PostgreSQL test database, then use `just appview-check` after each completed AppView slice.
- Implementation order: migration/store first, OAuth adapter and server flow second, Flutter third, full regressions and manual release evidence last.
- No lexicon files are affected; the atproto lexicon/ADR workflow is not invoked.
