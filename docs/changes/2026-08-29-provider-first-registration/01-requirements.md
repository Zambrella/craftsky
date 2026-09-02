# Requirements: Provider-First AT Protocol Registration

## 1. Initial Request
Add a registration journey for people who do not yet have an AT Protocol account. For this pass, Craftsky will use the Bluesky PDS service as the only account provider. The design must leave a clear path to support another provider, such as EuroSky, later without changing the user/session model.

## 2. Current Codebase Findings
- Relevant files: `app/lib/auth/pages/welcome_page.dart`, `app/lib/auth/pages/sign_in_page.dart`, `app/lib/auth/providers/auth_controller.dart`, `app/lib/auth/data/auth_api_client.dart`, `appview/internal/auth/handlers_session.go`, `appview/internal/auth/oauth_flow.go`, `appview/internal/auth/store.go`, `appview/internal/auth/initialize_profile.go`, and `appview/internal/routes/routes_public_auth.go`.
- Existing patterns: Flutter asks the AppView to start OAuth, opens the returned authorization URL in the system browser, and completes a code-only device handoff after the AppView OAuth callback. PDS credentials remain server-side.
- Current behavior: both welcome-page actions route to handle sign-in. `POST /v1/auth/login` requires a valid existing handle, resolves it to a DID and PDS, sends that DID as `login_hint`, and binds durable OAuth request state to an existing owner lifecycle before browser authorization.
- Constraints discovered: provider-first OAuth starts from an authorization service rather than an account identifier, so the owner DID is unknown until token exchange. Current `oauth_auth_requests` persistence requires a non-null owner DID, generation, and auth epoch with an owner foreign key. The callback currently verifies token `sub` against the owner DID known before authorization.
- Protocol behavior: omitting `login_hint` does not force registration. It allows the authorization server to offer its own account creation, sign-in, or account-selection interface. A server-first flow must verify after token exchange that the returned DID resolves to a PDS whose discovered authorization server is the issuer selected before authorization.
- Existing coverage: the real federated OAuth integration test covers distinct PDS and authorization-server origins, PDS reads/writes, and token/session behavior, but injects an already-known identity.
- Test/build commands discovered: `just appview-check`, `just appview-test-unit`, `just app-test`, and `just app-analyze`.

## 3. Clarifying Questions And Decisions
### Q1: Does omitting `login_hint` mean the provider must register an account?
Answer: No. It starts an OAuth flow not bound to a preselected account. The provider may offer account creation, sign-in, or account selection.
Decision / implication: Craftsky will describe the action as creating an account, but it will accept any valid account the Bluesky authorization interface returns after the user completes the flow and Craftsky verifies its authority.

### Q2: Which account provider is in scope?
Answer: The Bluesky PDS service only for this pass, while preserving a future path to providers such as EuroSky.
Decision / implication: Users will not enter or select an arbitrary provider URL in this release. The Bluesky service origin is product-controlled rather than client-controlled, and provider-specific policy must not be spread through owner/session behavior.

### Q3: Should Craftsky collect account credentials or call account-creation XRPC directly?
Answer: No. Account creation should occur in the provider's browser-based authorization interface.
Decision / implication: Craftsky never receives the user's provider email, password, invite code, or anti-abuse answers.

### Q4: What happens if provider-first OAuth selects an existing account?
Answer: Accept it.
Decision / implication: The flow is create-or-continue. A valid existing account completes sign-in after the same authority and lifecycle checks; Craftsky does not attempt to prove that the DID was newly created.

### Q5: How does registration launch and explain the provider relationship?
Answer: The action is labelled “Create an account” and launches the external browser immediately. The welcome and Add Account surfaces say: “Bluesky hosts your portable account, which you can use with Craftsky.”
Decision / implication: There is no registration interstitial or confirmation dialog. The provider disclosure must be visible before the action is tapped.

### Q6: How is the initial provider selected?
Answer: A validated server-side configuration selects one provider and defaults production to `https://bsky.social`.
Decision / implication: Flutter cannot submit or override the provider origin. Controlled development and test deployments may configure a safe fixture origin, and a future approved provider does not require an API-body change.

### Q7: Is registration available while adding another account?
Answer: Yes.
Decision / implication: The authenticated Add Account flow also offers “Create an account,” subject to the existing retained-account limit. A new flow cannot start at the limit. If a flow started below the limit and the limit is reached before handoff, an already-retained DID may replace/activate that account's local session without consuming a slot; a new DID receives the existing account-limit outcome.

### Q8: What happens if the browser is closed without a callback?
Answer: Craftsky remains non-blocking and retryable.
Decision / implication: App resume does not infer cancellation. The user can start again, and abandoned server state expires through the existing bounded cleanup path.

### Q9: Where are callback failures shown?
Answer: Explicit provider denial and technical failures reached through trusted registration state return to Flutter with bounded, non-secret error codes. Unknown or untrusted state remains on generic browser error HTML.
Decision / implication: Flutter distinguishes cancellation, temporary provider unavailability, and registration that could not be verified or completed according to the bounded outcome table in Section 12. Explicit `access_denied` is canceled; retryable upstream transport/timeout/429/5xx failures are provider unavailable; validation, authority, lifecycle, onboarding, and local launch failures are registration incomplete. Detailed security causes remain internal.

### Q10: How is pre-owner OAuth state persisted?
Answer: Extend the existing `oauth_auth_requests` state machine with purpose-aware nullable owner fields, then atomically bind the verified DID and lifecycle authority after token validation.
Decision / implication: Login and deletion rows remain strictly owner-bound through database constraints. Registration reuses existing capacity, expiry, replay, exchange, and cleanup machinery instead of duplicating it.

### Q11: What is the registration-start API?
Answer: `POST /v1/auth/registrations`.
Decision / implication: The request reuses handoff fields, contains no handle/DID/provider origin, and returns the existing `{authUrl}` response shape. `/v1/auth/login` remains unchanged.

### Q12: Should Craftsky request direct account creation when supported?
Answer: Yes, but only when authorization-server metadata advertises a supported create-account prompt.
Decision / implication: If a PAR containing an advertised create prompt is rejected, Craftsky fails safely and does not retry without the prompt. OAuth has no reliable machine-readable signal that distinguishes an unsupported prompt from another `invalid_request`, so parsing provider error text or downgrading a generic PAR failure is forbidden. Ordinary provider-first OAuth continues without a prompt when metadata does not advertise one.

### Q13: What live-provider evidence is required?
Answer: A manual real-Bluesky smoke test is required before release.
Decision / implication: Automated tests use controlled federated fixtures and do not create live accounts in CI. Release evidence includes one successful real create-and-return journey.

## 4. Candidate Approaches
### Option A: Standalone Bluesky account page
Summary: Open `https://bsky.social/account`, require the user to return manually, enter the new handle, and then run existing handle-first OAuth.
Pros: Small implementation; does not change OAuth persistence or callback authority checks.
Cons: No reliable callback, no automatic handle discovery, duplicated browser journey, and a high chance of onboarding abandonment.
Risks: Users may not return to Craftsky or may enter the wrong handle.

### Option B: Provider-first OAuth
Summary: Start OAuth from the Bluesky service without `login_hint`; request account creation only when metadata advertises the capability; let Bluesky provide registration/sign-in/account selection; receive the selected DID at token exchange; verify that DID against the preselected issuer; then continue through Craftsky onboarding and handoff.
Pros: Seamless browser return, credentials remain with the provider, and the resulting account is a normal portable AT Protocol identity.
Cons: Requires a pre-owner durable OAuth state and a different callback authority proof.
Risks: Authentication and owner-lifecycle changes are security-sensitive; provider UI behavior can vary with registration availability and existing browser sessions.

## 5. Recommended Direction
Recommended approach: Option B, provider-first OAuth with the Bluesky service as the sole product-selected provider.

Why: It gives users a continuous create-and-authorize journey without exposing PDS credentials to Craftsky. It follows the AT Protocol server-first OAuth model and preserves the existing AppView-mediated token and device-handoff architecture. The provider is selected by validated server configuration, defaulting production to Bluesky, so a future approved provider can be supplied without changing the Flutter contract or the returned-DID, owner-lifecycle, OAuth-session, and Craftsky-session semantics.

## 6. Problem / Opportunity
Someone without an AT Protocol identity cannot currently join Craftsky. The welcome page advertises account creation but routes to a form that only accepts an already-resolvable handle. A provider-first flow can make Craftsky usable as a person's entry point into the AT Protocol network without making Craftsky an account host or credential processor.

## 7. Goals
- G-001: Let a person without an AT Protocol account begin account creation from Craftsky and return signed in after successful provider authorization.
- G-002: Keep account credentials, provider terms, and anti-abuse checks entirely within the provider's browser interface.
- G-003: Prove that the DID returned after an unbound OAuth flow is controlled by the authorization server selected at flow start.
- G-004: Preserve existing handle-first sign-in, onboarding, token storage, and device handoff behavior.
- G-005: Isolate the initial Bluesky provider choice so another approved provider can be added later without changing account identity or session semantics.

## 8. Non-Goals
- NG-001: Running a Craftsky PDS or hosting AT Protocol accounts.
- NG-002: Supporting a provider chooser, EuroSky, arbitrary PDS host input, or automatic provider discovery in this pass.
- NG-003: Guaranteeing that the provider displays registration rather than sign-in or account selection.
- NG-004: Collecting provider email addresses, passwords, invite codes, CAPTCHA responses, or other account-creation credentials in Flutter or the AppView.
- NG-005: Calling `com.atproto.server.createAccount` from Craftsky.
- NG-006: Changing existing handle-first login, account migration, account deletion, or PDS token mediation rules.
- NG-007: Modifying lexicons or record shapes.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| New Craftsky user | A person who does not yet have an AT Protocol account | A clear way to create an account and return to Craftsky without manually transferring a handle |
| Existing AT Protocol user | A person who already has an account and may encounter the provider interface | Existing handle sign-in must remain available; a valid account selected in provider-first OAuth must be handled safely |
| Bluesky PDS service | The only approved account provider for this pass | Standards-compliant OAuth requests and control of its own account creation and authentication UI |
| Craftsky AppView | Confidential OAuth client and session authority | Durable pre-owner request state, issuer/DID verification, server-side tokens, and owner lifecycle safety |
| Craftsky Flutter app | Native client holding only Craftsky session credentials | A registration action, external browser launch, code-only return, and clear success/failure states |

## 10. Current Behavior
The welcome page has “Sign in” and “Create account on a PDS” actions, but both navigate to the same handle form. Login cannot start until the submitted handle resolves to an existing DID and PDS. The AppView binds the authorization request to that owner, sends the DID as `login_hint`, and later requires token `sub` to match it. A person without an account therefore cannot enter the OAuth flow.

## 11. Desired Behavior
The welcome page and authenticated Add Account flow offer separate existing-account and account-creation actions. One sentence explains Bluesky's hosting role before “Create an account” immediately starts the browser journey. Flutter asks the AppView to start OAuth against the validated server-selected provider without an account-specific `login_hint`. When metadata advertises a create-account prompt, the AppView requests it without an unsafe prompt-free downgrade. The system browser displays Bluesky's provider-controlled interface, where the person can create, sign in to, or select an account. On callback, the AppView exchanges the code, validates the original state and issuer, resolves the returned DID authoritatively, proves its PDS discovers the same authorization server, and only then creates or resumes owner/session state. The existing profile initialization and code-only Flutter handoff complete the journey. Trusted-state failures return safely to Flutter; abandoned browser requests remain retryable and expire normally.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A person without an AT Protocol account shall be able to begin an account-creation journey from Craftsky and return authenticated after completing provider authorization. | Removes the current prerequisite to join another AT Protocol app first. | Initial request | AC-001, AC-010 |
| BR-002 | Business | Must | Existing users shall retain the current handle-first sign-in journey without behavioral regression. | Registration must be additive. | Codebase findings | AC-002, AC-018 |
| BR-003 | Business | Should | The provider integration should permit a future approved provider to replace or accompany Bluesky without changing DID-based account identity, owner lifecycle, OAuth session, Craftsky session, or handoff semantics. | The user explicitly anticipates providers such as EuroSky. | User direction | AC-017 |
| FR-001 | Functional | Must | The welcome experience and authenticated Add Account flow shall expose distinct actions for signing in with an existing account and creating an account, subject to the existing retained-account limit. | The current duplicate navigation is misleading, and account creation should be available wherever an identity can be added. | Codebase findings / User answer | AC-001, AC-002, AC-020 |
| FR-002 | Functional | Must | Starting registration shall request a provider-first OAuth flow from the AppView without requiring a handle or DID from Flutter. | A new account has no resolvable identity yet. | Discovery | AC-003 |
| FR-003 | Functional | Must | The AppView shall discover and validate OAuth metadata for the configured Bluesky service and create a PAR request without an account-specific `login_hint`; when metadata advertises a supported create-account prompt, it shall request that prompt. | The authorization server must control registration/sign-in/account selection while allowing a direct creation path when explicitly supported. | Confirmed direction / User answer | AC-003, AC-004, AC-023 |
| FR-004 | Functional | Must | The AppView shall durably persist bounded, expiring registration OAuth state in the existing auth-request state machine before browser redirection even though no owner DID is known. | Callback safety and retries cannot depend on process memory, and duplicate security machinery should be avoided. | Codebase findings / User answer | AC-005, AC-016, AC-027 |
| FR-005 | Functional | Must | Registration authorization requests shall preserve the existing handoff mode, device binding, PKCE, DPoP, state, shared request-capacity, timeout, and `RateClassAuth` protections. | Provider-first initiation must not weaken existing OAuth controls. | Existing architecture | AC-005, AC-006, AC-016, AC-029 |
| FR-006 | Functional | Must | Flutter shall immediately open the returned provider authorization URL in the system browser and retain the existing code-only verified-link, loopback, or allowed development handoff behavior. | Reuses the secure browser and app-return architecture without an extra interstitial. | Existing architecture / User answer | AC-007, AC-015, AC-021 |
| FR-007 | Functional | Must | The callback shall validate the stored request state, exact expected issuer, authorization code, one-time exchange state, and token response before accepting a returned account. | Prevents request substitution and callback replay. | Security discovery | AC-008, AC-011, AC-012 |
| FR-008 | Functional | Must | After token exchange, the AppView shall parse token `sub` as the candidate account DID, resolve that DID authoritatively, discover the authorization server for its declared PDS, and require it to match the issuer bound at registration start. | An unbound authorization server must not be trusted to claim an unrelated DID. | AT Protocol server-first OAuth rules | AC-008, AC-009 |
| FR-009 | Functional | Must | The AppView shall create or resume owner lifecycle and OAuth session state only after FR-007 and FR-008 succeed, while enforcing the same ineligible-owner rules as ordinary login. If later onboarding or handoff preparation fails, a newly created verified owner may remain `departed`; an existing eligible owner's prior lifecycle and sessions remain unchanged; no OAuth or Craftsky session created by the failed attempt may become active. | Avoids unauthorized active owners and preserves deletion/terminal fences while retaining verified onboarding identity. | Codebase findings / Review | AC-009, AC-011, AC-013 |
| FR-010 | Functional | Must | A successfully verified registration shall run the existing Craftsky profile initialization, identity-cache update, repository tracking request, pending session creation, and device handoff confirmation flow. | A registered account must enter Craftsky in the same valid state as a signed-in account. | Existing onboarding flow | AC-010 |
| FR-011 | Functional | Must | Cancellation, provider denial, malformed callbacks, metadata failures, token failures, authority mismatches, and onboarding failures shall not create an active Craftsky session and shall produce a user-safe error path. Trusted registration state shall return the bounded result defined in the outcome table; unknown or untrusted state shall remain on generic browser error HTML. After verified owner binding, failure may retain a newly created owner only as `departed`, shall leave an existing owner's prior lifecycle/sessions unchanged, and may retain terminal auth-request evidence; exchanged credentials shall be revoked or retained solely in the existing ambiguous/revocation cleanup state. | Partial authentication must fail closed without using untrusted state to choose an app destination. | Discovery / User answer / Review | AC-011, AC-012, AC-014, AC-025 |
| FR-012 | Functional | Should | One clear sentence beside each registration action shall identify Bluesky as the account hosting provider and explain that the portable account works with Craftsky. | Avoids conflating Craftsky membership with Bluesky app usage without adding an interstitial or protocol lesson. | Product discussion / User answer | AC-019, AC-021 |
| FR-013 | Functional | Must | Flutter shall start provider-first registration through `POST /v1/auth/registrations`, sending only the existing handoff mode and any mode-required loopback redirect URI; the response shall use the existing `{authUrl}` shape. | Gives registration a distinct durable-attempt resource while leaving handle login unchanged. | User answer / API architecture | AC-003, AC-022 |
| FR-014 | Functional | Must | Returning to Craftsky without a callback shall leave registration non-blocking and retryable; app resume alone shall not mark the attempt canceled. | The app cannot reliably distinguish browser abandonment from an in-progress provider flow. | User answer | AC-024 |
| FR-015 | Functional | Must | Flutter shall present exactly the three safe registration failure categories and retry behavior defined in the Section 12 outcome table. | Users need actionable distinctions without exposure of security-sensitive causes. | User answer / Review | AC-026 |
| FR-016 | Functional | Must | Release verification shall include a successful manual real-Bluesky account create-and-return smoke test; automated CI shall use controlled fixtures rather than creating live provider accounts. | Confirms deployed provider behavior without coupling CI to anti-abuse controls or provider availability. | User answer | AC-028 |
| NFR-001 | Non-functional | Must | Provider-first registration shall fail closed under network errors, process interruption, duplicate callbacks, stale requests, issuer mismatch, DID resolution failure, or authority verification failure. | This flow establishes account and credential authority. | Security discovery | AC-005, AC-008, AC-009, AC-011, AC-012, AC-016 |
| NFR-002 | Non-functional | Must | Provider credentials and PDS OAuth access/refresh tokens shall never be returned to or persisted by Flutter, included in callback URLs, or written to application logs. | Preserves the Token Mediating Backend boundary. | Architectural rule | AC-015 |
| NFR-003 | Non-functional | Should | Provider-specific service selection and presentation should be isolated from provider-independent OAuth completion, owner lifecycle, onboarding, and handoff behavior; the selected origin shall come from validated AppView configuration rather than Flutter input. | Keeps future provider support incremental and prevents arbitrary federated destinations. | User direction / User answer | AC-017, AC-022 |
| NFR-004 | Non-functional | Should | Registration-start and callback errors should use the existing camelCase `/v1/` API conventions, error envelope, request ID, and safe structured logging conventions. | Keeps the API and operations surface consistent. | API architecture | AC-014 |
| RULE-001 | Business rule | Must | The only registration provider in this pass shall be the server-configured Bluesky PDS service, defaulting production to `https://bsky.social`; the client shall not be permitted to supply or override a provider origin. | Meets scope, supports controlled fixtures/future rollout, and avoids an arbitrary federated request/SSRF surface. | User direction / User answer | AC-003, AC-004, AC-022 |
| RULE-002 | Business rule | Must | Craftsky shall not collect, proxy, or persist provider account-creation credentials or directly create the PDS account. | The user must have a direct security relationship with the host. | Confirmed direction | AC-015 |
| RULE-003 | Business rule | Must | Absence of `login_hint` shall be treated as provider-controlled account selection, not as a guarantee that a new account will be created. A valid existing account selected by the user may complete the same flow. | This is the actual authorization-server behavior. | Clarifying answer | AC-004, AC-013 |
| RULE-004 | Business rule | Must | No owner, OAuth session, or Craftsky session may become active until the returned DID is proven authoritative for the issuer selected at registration start. | Prevents issuer account-claim attacks. | Security discovery | AC-008, AC-009, AC-011 |
| RULE-005 | Business rule | Must | A create-account prompt shall be sent only when advertised by authorization-server metadata. Any PAR rejection after adding that prompt shall fail without a prompt-free retry; Craftsky shall not classify generic OAuth errors or provider error descriptions as permission to downgrade. | Improves the direct creation journey without masking unrelated OAuth or security failures. | User answer / Review | AC-023 |
| RULE-006 | Business rule | Must | Callback errors may return to Flutter only when trusted stored registration state identifies the approved handoff destination and mode. | Prevents untrusted callback input from selecting a native-app destination. | User answer / Security discovery | AC-025 |
| RULE-007 | Business rule | Must | The unauthenticated registration route shall use the same per-device `RateClassAuth` limit and shared durable pending-request capacity as login. No new source-IP or distributed limiter is required in this pass; device-ID rotation is an explicitly accepted residual risk. | Bounds routine and single-device abuse without silently expanding this feature into a new rate-limiting architecture. | Codebase findings / Review | AC-006, AC-029 |

### Registration Failure Outcomes
| Trusted condition | Flutter category | Retry behavior |
|---|---|---|
| Provider callback returns `access_denied`. | `canceled` | No automatic retry; the user may explicitly start a fresh registration. |
| Provider metadata/discovery, PAR, or token exchange fails because of transport failure, timeout, HTTP 429, or HTTP 5xx; or a PAR containing an advertised create prompt is rejected. | `providerUnavailable` | No protocol operation is automatically retried. The user may explicitly start a fresh registration; an authorization code is never reused. |
| Provider configuration or metadata is invalid; a discovered endpoint is invalid; or an ordinary no-prompt PAR is rejected with a non-transient OAuth/client/scope/DPoP/HTTP 4xx error. | `registrationIncomplete` | No automatic retry or downgrade occurs. The user may leave the failure state or explicitly start a fresh registration; raw provider errors are not shown. |
| Callback/token shape is malformed; state/issuer/DID/PDS authority validation fails; lifecycle rejects the owner; onboarding or browser launch fails; or another bounded non-retryable completion error occurs. | `registrationIncomplete` | The failed OAuth attempt is not resumed. The user may explicitly start a fresh registration after any existing cleanup requirement is satisfied. |
| Handoff exchange or confirmation has a transport/timeout/5xx failure after valid handoff material exists. | Existing handoff error/recovery behavior | Preserve retry of the exact durable handoff receipt; do not restart OAuth or translate it into a new registration callback result. |
| Callback state is missing, malformed, unknown, expired, consumed, or otherwise untrusted. | None | Render generic browser error HTML; do not open Flutter. |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given an unauthenticated person on the welcome page, when they choose account creation, then Craftsky starts the dedicated registration journey without displaying the existing handle form as a prerequisite. |
| AC-002 | BR-002, FR-001 | Given an unauthenticated person with an existing account, when they choose sign-in, then the current handle-first login journey remains available and requires their handle. |
| AC-003 | FR-002, FR-003, RULE-001 | Given registration is started, when Flutter calls the AppView, then no handle, DID, or client-selected provider origin is required or accepted and the AppView starts from the approved Bluesky service. |
| AC-004 | FR-003, RULE-001, RULE-003 | Given valid Bluesky authorization metadata, when the AppView creates the PAR request, then it uses the configured Craftsky client, required OAuth protections and scopes, and no account-specific `login_hint`. |
| AC-005 | FR-004, FR-005, NFR-001 | Given a registration PAR succeeds, when the AppView returns the browser URL, then the request state needed for callback verification and handoff is already durably stored without an owner DID and survives AppView restart. |
| AC-006 | FR-005 | Given a registration-start request with a missing/invalid device ID, unavailable handoff mode, invalid loopback URI, exhausted request capacity, or operation timeout, when it is processed, then OAuth does not start and the corresponding safe API error is returned. |
| AC-007 | FR-006 | Given a successful registration-start response, when Flutter receives the authorization URL, then it records pending authentication state and opens the URL in the external system browser using the build-appropriate handoff mode. |
| AC-008 | FR-007, FR-008, NFR-001, RULE-004 | Given a callback whose token response contains a DID that resolves to a PDS whose discovered authorization server matches the exact stored issuer, when all OAuth checks pass, then the account is accepted as authoritative. |
| AC-009 | FR-008, FR-009, NFR-001, RULE-004 | Given a callback whose returned DID is malformed, unresolvable, missing a PDS, or discovers a different authorization server, when callback processing runs, then it fails closed before owner/session activation and the exchanged credentials enter the existing safe cleanup/revocation path. |
| AC-010 | BR-001, FR-010 | Given a newly created provider account passes authority verification, when callback and handoff complete, then profile initialization and tracking run, Flutter receives only the short-lived handoff material, confirms a Craftsky session, and routes through normal first-account onboarding. |
| AC-011 | FR-007, FR-009, FR-011, NFR-001, RULE-004 | Given an unknown, stale, replayed, consumed, or concurrently exchanged state, an ineligible owner, or a failure after verified owner binding, when callback completion is attempted, then no session from the failed attempt becomes active and lifecycle fences hold. A verified new owner may remain only `departed`; an existing eligible owner's prior lifecycle and sessions remain unchanged; pending session state from the attempt is abandoned; credentials are revoked or enter existing ambiguous/revocation cleanup; and auth-request evidence follows existing terminal retention. |
| AC-012 | FR-007, FR-011, NFR-001 | Given the callback contains provider denial, missing code, wrong `iss`, failed token exchange, invalid token response, or a duplicate callback, when processed, then it fails closed and does not expose credentials or create an active Craftsky session. |
| AC-013 | FR-009, RULE-003 | Given the user selects an existing valid Bluesky-hosted account in the provider-first interface, when its DID passes authority and lifecycle validation, then the flow signs that account into Craftsky rather than requiring that a new DID was created. |
| AC-014 | FR-011, NFR-004 | Given registration start or completion fails, when an HTTP API error is surfaced, then it uses the standard camelCase error envelope and request ID; when trusted callback state permits Flutter return, only a Section 12 bounded result is sent; and structured logs identify the safe stage/reason without secrets, raw credentials, or provider error text. |
| AC-015 | FR-006, NFR-002, RULE-002 | Given successful, denied, malformed, network-failed, authority-failed, onboarding-failed, and handoff-failed registration fixtures, when registration request/response bodies, authorization redirects, callback URLs/HTML, handoff payloads, Flutter models/storage, AppView auth persistence, and structured logs are inspected, then provider passwords/account-creation inputs and PDS access/refresh tokens are absent from all Flutter/browser/log surfaces; only approved short-lived handoff values may reach Flutter. |
| AC-016 | FR-004, FR-005, NFR-001 | Given an abandoned registration request, when its expiry or cleanup window passes, then it no longer permits callback completion and is reclaimed under bounded authentication-state retention without an orphan active owner/session. |
| AC-017 | BR-003, NFR-003 | Given two controlled configured provider origins run through the same registration fixture, when contracts and persisted records are inspected, then only provider configuration, metadata discovery, and PAR initiation vary; authority verification, owner/OAuth/Craftsky session records, onboarding, and handoff contracts remain provider-neutral and produce the same outcomes. |
| AC-018 | BR-002 | Given the provider-first registration change is present, when existing handle-first OAuth integration and Flutter auth regression suites run, then their successful, denied, timeout, lifecycle, and handoff behaviors remain unchanged. |
| AC-019 | FR-012 | Given the registration entry UI is displayed, when a user reads the provider explanation, then it says “Bluesky hosts your portable account, which you can use with Craftsky.” |
| AC-020 | FR-001 | Given a signed-in user is below the retained-account limit, when they open Add Account, then both existing-handle sign-in and “Create an account” are available; at the limit, a new registration cannot start. Given a flow started below the limit and another account fills the final slot before handoff, an already-retained returned DID replaces/activates its session without increasing count, while a new DID receives the existing account-limit outcome. |
| AC-021 | FR-006, FR-012 | Given a registration action is visible, when the user reviews the screen and taps “Create an account,” then one clear provider/portability sentence was visible before the tap and the external browser launches without an interstitial or confirmation dialog. |
| AC-022 | FR-013, NFR-003, RULE-001 | Given Flutter starts registration, when the request is inspected, then it targets `POST /v1/auth/registrations`, contains the required handoff fields but no handle, DID, or provider origin, and the AppView uses its validated configured provider and returns `{authUrl}`. |
| AC-023 | FR-003, RULE-005 | Given metadata advertises a supported create-account prompt, when registration PAR is sent, then the prompt is included; if that PAR fails for any reason, no prompt-free retry occurs. Given metadata does not advertise the prompt, the AppView sends one ordinary provider-first PAR without it. |
| AC-024 | FR-014 | Given the user closes or leaves the provider browser without a callback and later resumes Craftsky, when they revisit the registration action, then the UI is usable and can retry without treating resume as authoritative cancellation; abandoned server state expires normally. |
| AC-025 | FR-011, RULE-006 | Given a provider denial or technical callback failure with trusted stored registration state, when the callback is handled, then a bounded non-secret error returns through the approved handoff to Flutter; given unknown or untrusted state, no deep link is produced and generic browser error HTML is shown. |
| AC-026 | FR-015 | Given each trusted condition in the Section 12 outcome table, when the result is handled, then Flutter presents the specified category and retry behavior without internal issuer/DID/token details; untrusted callback state never produces a Flutter category, and handoff transport recovery remains unchanged. |
| AC-027 | FR-004 | Given the migrated auth-request schema, when rows for login, deletion, and provider-first registration are validated, then login/deletion rows require owner DID/generation/auth epoch, initial registration rows require those fields to be absent, and a verified registration can atomically bind valid lifecycle authority without changing the shared exchange state. |
| AC-028 | FR-016 | Given automated fixture suites pass, when release verification is performed, then a tester completes one real Bluesky account create-and-return journey and records date, build/version, platform, provider metadata/prompt behavior, callback mode, outcome, secret inspection, and disposable-account cleanup disposition in `docs/changes/2026-08-29-provider-first-registration/release-smoke-evidence.md`; CI does not create or retain live Bluesky accounts. |
| AC-029 | FR-005, RULE-007 | Given repeated unauthenticated registration requests, when route and admission controls are inspected, then each request passes through the existing `RateClassAuth` per-device limit before provider work and shares the existing durable global pending-auth capacity; limit failures return bounded 429/503 responses with `Retry-After`. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User already has an authenticated Bluesky browser session | The provider may select or offer that existing account; Craftsky accepts it only after normal callback authority and lifecycle checks. | RULE-003, FR-008, FR-009 |
| EC-002 | Provider registration is disabled, invite-only, or unavailable | The provider controls its UI; Craftsky does not promise account creation and surfaces cancellation/failure without creating a session. | FR-011, RULE-003 |
| EC-003 | Returned DID points to a concrete Bluesky PDS while OAuth issuer is the Bluesky entryway | Craftsky follows PDS protected-resource discovery and accepts the relationship only when it resolves back to the stored issuer; it does not require equal PDS/issuer hostnames. | FR-008 |
| EC-004 | Authorization server returns a DID hosted by an unrelated PDS/provider | Craftsky rejects the flow and safely disposes of exchanged credentials. | FR-008, RULE-004 |
| EC-005 | Existing Craftsky account is selected through registration | Craftsky treats it as a login subject to current owner eligibility and multi-account/device session rules. | FR-009, RULE-003 |
| EC-006 | User backs out of the browser or denies authorization | Pending state expires; no active owner/session is created; the user can retry. | FR-011, FR-004 |
| EC-007 | AppView restarts between PAR and callback | Durable pre-owner request state allows valid completion after restart within the normal expiry window. | FR-004 |
| EC-008 | Two callbacks race for the same state | At most one exchange attempt proceeds; the other fails without creating another active OAuth/Craftsky session. | FR-007, NFR-001 |
| EC-009 | Authority verification fails after a successful token exchange | Credentials are not activated and enter the same explicit revocation/ambiguous cleanup discipline used by existing OAuth failures. | FR-008, FR-011 |
| EC-010 | Flutter browser launch fails | Pending client state is cleared or made safely retryable and no Craftsky session is created. | FR-006, FR-011 |
| EC-011 | PAR containing an advertised create prompt is rejected | AppView does not parse provider error text or retry without the prompt; it returns the bounded provider-unavailable result. | FR-003, FR-015, RULE-005 |
| EC-012 | User resumes Craftsky while provider flow remains open | Craftsky remains non-blocking and permits a later callback or explicit retry; resume itself does not cancel server state. | FR-014 |
| EC-013 | Callback error has trusted state but no successful token exchange | Craftsky returns only a bounded error result through the stored approved handoff; no session or credential is created. | FR-011, RULE-006 |
| EC-014 | Callback state is missing, unknown, malformed, or expired | Craftsky renders generic browser error HTML and does not construct an app deep link from callback input. | FR-011, RULE-006 |
| EC-015 | Account limit is reached after browser launch | A returned DID already retained locally may replace/activate that session; a new DID receives the existing account-limit outcome without displacing another account. | FR-001 |
| EC-016 | Onboarding fails after verified new-owner binding | The owner may remain `departed`; no active OAuth/Craftsky session remains; credentials and auth-request evidence follow the defined cleanup/retention paths. | FR-009, FR-011 |

## 15. Data / Persistence Impact
- New fields: Durable registration OAuth state must represent the configured provider/issuer binding and registration purpose before an owner DID exists. Exact field names are deferred to design.
- Changed fields: Existing `oauth_auth_requests.owner_did`, `owner_generation`, and `auth_epoch` become purpose-aware nullable fields. Database constraints must require all three for login/deletion, require all three absent for unbound registration, and permit an atomic verified transition to all three present.
- Migration required: Yes. Extend `oauth_auth_requests` and its constraints/indexes so pre-owner registration shares exact issuer binding, handoff/device metadata, capacity, expiry, callback exchange states, terminal retention, and cleanup without weakening owner-bound login and deletion rows.
- Backwards compatibility: Existing handle-first auth requests and OAuth/Craftsky sessions must retain their semantics. No lexicon or PDS record migration is required.

## 16. UI / API / CLI Impact
- UI: Replace the duplicate welcome-page navigation with distinct sign-in and “Create an account” actions; add the same registration action to Add Account under the existing account limit. Show one provider/portability sentence, launch immediately, remain retryable after browser abandonment, and present the three approved failure categories.
- API: Add unauthenticated `POST /v1/auth/registrations` with existing camelCase handoff fields and `{authUrl}` response. It accepts no handle, DID, or provider origin. Existing `/v1/auth/login`, `/oauth/callback`, and handoff endpoint contracts remain compatible; callback internals gain a registration purpose and trusted-state error handoff.
- CLI: No user-facing CLI registration flow is required. Existing loopback handoff compatibility must not regress.
- Background jobs: Existing auth-request expiry, ambiguous exchange handling, token revocation, and cleanup behavior must cover or be safely extended to pre-owner registration requests.

## 17. Security / Privacy / Permissions
- Authentication: Registration is unauthenticated at start, but device-bound admission, OAuth state, PKCE, DPoP, exact issuer checks, token response validation, authoritative DID/PDS resolution, and one-time callback exchange are mandatory.
- Authorization: A returned DID gains owner/session authority only after its discovered authorization server matches the issuer selected at flow start and owner lifecycle policy permits activation.
- Sensitive data: Provider credentials never pass through Craftsky. PDS tokens and DPoP keys remain encrypted/server-side according to existing OAuth storage rules. Flutter receives only Craftsky session/handoff material.
- Abuse cases: Client-supplied provider URLs are forbidden in this pass. Registration shares login's per-device `RateClassAuth` limiter and the durable global pending-auth capacity/expiry. Device-ID rotation can bypass the local per-device rate bucket; adding source-IP or distributed limiting is explicitly outside this pass and remains a monitored residual risk. Callback replay, issuer substitution, arbitrary DID claims, owner lifecycle bypass, and logs containing OAuth secrets must fail closed.

## 18. Observability
- Events: Registration start accepted/rejected, create prompt attempted/rejected, provider redirect produced, callback exchange started, authority verification passed/failed, owner/session activation passed/failed, error handoff produced, successful handoff produced/confirmed, and abandoned-state cleanup.
- Logs: Use request/run IDs and safe stage/reason classifications. Include whether the flow is provider-first and the approved provider identity where safe. Do not log authorization codes, PAR request URIs, tokens, DPoP private material, handoff secrets, or provider credentials.
- Metrics: Count registration starts, provider redirects, callbacks, successful authority verification, completed handoffs, provider denials/cancellations, verification failures, lifecycle rejections, capacity rejections, and expirations. Exact metric names are deferred.
- Alerts: Existing OAuth ambiguous-exchange, cleanup/revocation, capacity, and upstream-failure alerts should include registration flows. New alert thresholds are deferred to implementation planning.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Pre-owner OAuth state weakens existing owner lifecycle fencing | Unauthorized or orphan sessions could be created. | Separate pre-owner state semantics; create/attach owner only after authority proof; preserve transactional state transitions and negative integration tests. |
| RISK-002 | Craftsky trusts token `sub` without proving issuer authority | A malicious or compromised authorization server could claim another DID. | Authoritatively resolve DID to PDS, discover its authorization server, and require an exact match to stored `iss` before activation. |
| RISK-003 | Bluesky's unbound authorization UI does not display registration in some contexts | The “Create account” action may show account selection or sign-in, confusing users. | Use accurate copy, do not promise forced registration, accept valid selected accounts, and surface provider-controlled outcomes. |
| RISK-004 | Bluesky-specific assumptions become coupled to lifecycle/session code | Adding EuroSky or another provider becomes risky and expensive. | Keep provider selection/presentation separate from provider-independent OAuth completion and authority checks; include architecture acceptance review. |
| RISK-005 | Schema changes make owner-bound login/deletion requests nullable without adequate constraints | Existing callback and deletion security could regress. | Use purpose-aware database constraints in the shared auth-request table; retain migration and regression tests for every request purpose. |
| RISK-006 | Successful token exchange followed by authority/onboarding failure leaves live credentials | Credentials may remain valid without a usable Craftsky session. | Reuse explicit revocation-pending/ambiguous cleanup and verify failure paths. |
| RISK-007 | Provider service outage or anti-abuse policy blocks signup | New users cannot complete registration. | Return safe retryable errors; do not fall back to unsafe direct account creation; future approved providers remain possible. |
| RISK-008 | Pinned Indigo lacks metadata/request-helper support for the advertised create prompt | An unsafe adapter or unmerged dependency could weaken PAR behavior or create maintenance drift. | Do not rely on unmerged PR #1411; coding design must choose a stable dependency upgrade or narrow project-local adapter, preserving Indigo's OAuth/DPoP protections and covering it with fixture tests. Never downgrade failed prompt PAR from error text. |
| RISK-009 | Error deep links are constructed from untrusted callback input | An attacker could trigger or redirect native-app error flows. | Produce error handoffs only from trusted durable state and use bounded non-secret error codes. |
| RISK-010 | An attacker rotates device IDs to bypass the local auth rate bucket | Provider PAR load or shared pending capacity could be exhausted. | Retain the shared hard pending-request cap, timeout, expiry, and `RateClassAuth` limit; monitor capacity/provider failures. Accept this residual risk for the first Bluesky-only pass and revisit source-level/distributed admission before broad provider rollout. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The Bluesky authorization service continues to support server-first OAuth without `login_hint` and can offer account creation from its authorization interface. | The seamless registration journey would need a provider-specific documented initiation mechanism or fall back to manual account creation. |
| ASM-002 | A successful server-first token response includes an atproto DID in `sub`, and Bluesky protected-resource/authorization-server discovery proves entryway authority for that DID's PDS. | Authority verification rules or the selected provider integration would need revision before implementation. |
| ASM-003 | A missing `login_hint` does not guarantee or force registration, while an advertised create-account prompt can request but does not change base authority verification. | If Bluesky's advertised prompt semantics differ, the direct-create optimization must be disabled while ordinary server-first OAuth remains available. |
| ASM-004 | Existing profile initialization can onboard a newly created Bluesky-hosted account immediately after token exchange. | Provider propagation delays may require bounded retry or a dedicated pending-onboarding state. |
| ASM-005 | The approved Bluesky service identity can be selected through validated server configuration while remaining isolated behind a provider boundary. | A multi-provider registry or user-selectable provider model would need additional requirements. |

## 21. Open Questions
- [x] Resolved by document re-review: The registration sentence is “Bluesky hosts your portable account, which you can use with Craftsky.”
- [x] Resolved by document review: The pinned Indigo version lacks `prompt_values_supported` and a prompt option on `SendAuthRequest`; OAuth provides no reliable unsupported-prompt fallback signal. Coding design must use a stable upgrade or narrow adapter and must never rely on unmerged PR #1411 or provider error text.

## 22. Review Status
Status: Approved
Risk level: High
Review recommended: Required
Reviewer: Product owner (original direction); OpenCode (document re-review)
Date: 2026-08-29
Notes: Original direction was explicitly approved on 2026-08-29. Revisions resolve document-review findings DR-001 through DR-008 by removing unsafe prompt fallback, defining durable failure state, explicitly accepting current abuse-control limits, resolving the account-limit race, bounding public outcomes, and making acceptance evidence objective. The high-risk document re-review approved coding-plan readiness on 2026-08-29.

## 23. Handoff To Test Design
- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001, BR-002, FR-001 through FR-011, FR-013 through FR-016, NFR-001, NFR-002, RULE-001 through RULE-007.
- Suggested test levels: Flutter widget/provider/API-client tests; AppView handler and OAuth-flow unit tests; real-Postgres purpose-constraint/store/migration tests; federated integration tests with separate PDS and authorization-server origins; advertised create-prompt/no-downgrade tests; trusted/untrusted callback error-handoff tests; post-binding cleanup tests; callback replay/race/failure tests; required manual real-Bluesky create-and-return smoke test; existing auth regression suite.
- Blocking open questions: None. The high-risk document re-review approved coding-plan readiness on 2026-08-29.
