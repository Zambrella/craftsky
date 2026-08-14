# AV-008, AV-018, and AV-019 — OAuth handoff and callback finalization

- Findings: AV-008, AV-018, AV-019
- Severities: High (AV-008); Medium (AV-018, AV-019)
- Priority/order: Authentication and session integrity; implement after the hardened OAuth transport in AV-001/AV-017 and before treating the mobile sign-in flow as release-ready
- Status: Planned
- Sources: [AV-008](../2026-08-12-appview-code-audit.md#av-008--a-long-lived-bearer-crosses-a-claimable-custom-url-scheme), [AV-018](../2026-08-12-appview-code-audit.md#av-018--loopback-oauth-handoff-is-deterministically-lost), [AV-019](../2026-08-12-appview-code-audit.md#av-019--partial-oauth-callback-failure-retains-unreachable-upstream-credentials)

## Shared implementation strategy

Replace the callback's bearer-in-URL contract with a server-owned, recoverable handoff state machine. The OAuth callback will retain validated handoff metadata before Indigo consumes the authorization request, finalize the upstream OAuth session into an explicit pending state, initialize the account, and issue only a short-lived handoff code. Flutter or the loopback client redeems that code over a direct HTTP request bound to the initiating device. Redemption creates one inactive child bearer and a short-lived encrypted receipt; only client confirmation after secure persistence atomically activates parent and child and consumes the receipt. Until confirmation, retrying the same code from the same device returns the same receipt and bearer, so a lost HTTP response cannot strand an active credential.

This is deliberately a breaking redesign. There are no production users, so the implementation must remove the old `?token=` deep link and loopback bearer body outright rather than support parallel insecure contracts. The shared update should also make unfinished logins durable and observable. Once the first parent session is durably saved, it remains either redeemable for a short bounded interval or eligible for idempotent revocation and deletion by a cleanup worker. A token response lost in the unavoidable pre-save crash window is represented honestly as a non-secret `exchange_ambiguous` attempt, not falsely described as recoverable or revoked.

The target state machine is:

1. Login start resolves the canonical owner DID, acquires that owner's fence, calls `EnsureOnboardingOwner` to insert a first-login `departed` lifecycle row at auth epoch 1 if absent and re-read it, then passes typed purpose, handoff mode, loopback URI, device ID, owner auth epoch, and owner DID through a generalized `AuthRequestMetadataFromContext`. `PostgresAuthStore.SaveAuthRequestInfo` inserts that metadata atomically with Indigo's state/data; there is no post-insert `recordHandoff` update. Start Indigo with the already resolved DID where its `AtIdentifier` input permits; otherwise the callback must compare the token response's canonical `AccountDID` with the persisted owner before the first `SaveSession`.
2. The callback reads and validates the complete row, acquires the owner fence, rechecks epoch/lifecycle, and durably advances the request from `ready` to `exchange_started` with a unique attempt ID before calling the token endpoint. Indigo's logical request deletion consumes it without erasing the attempt record.
3. The callback adapter captures a successful token response and attempts the first `SaveSession` under the owner fence and new parent-session lock. Success creates `pending_handoff`; persistence failure triggers immediate bounded best-effort revocation while token material remains in memory and records an honest `exchange_ambiguous` residual if revocation cannot be confirmed.
4. AppView initializes profile/identity state and creates a hashed, expiring exchange-code row.
5. The browser hands only the exchange code to a verified app/universal link or to the validated loopback listener.
6. Redemption locks the exchange row, checks device, purpose, and expiry, creates one inactive child, and stores its bearer only as short-lived authenticated ciphertext in a device-bound receipt. A repeat redemption returns that receipt; it does not create another child.
7. After Flutter durably stores the bearer, authenticated confirmation activates parent and child and destroys the receipt ciphertext/code hash in one transaction. A lost confirmation response is harmless because confirmation is idempotent and the bearer is already stored.
8. Any callback failure after a parent exists, unconfirmed-receipt expiry, or explicit client abandonment transitions the parent to `revocation_pending`; a bounded worker revokes upstream credentials when possible and always removes local parent, child, and receipt state after its retry policy is exhausted. An `exchange_ambiguous` attempt with no local token is retained and alerted until a conservative provider credential-residual horizon passes; it is never mislabeled as revoked.

Account-deletion reauthentication takes a separate branch: its callback stores the parent as `deletion_only`, emits no ordinary handoff code or bearer, and returns through the verified deletion-complete link. Only the accepted matching deletion job can resume that parent through a typed privileged store method.

## Problem to solve

The current callback crosses three boundaries without a coherent transaction:

- It puts a long-lived bearer in a custom-scheme URL and, in development, in callback HTML.
- It asks Indigo to delete the OAuth request row before AppView reads the loopback routing data stored on that row.
- It lets Indigo persist live access/refresh tokens before profile initialization and CraftSky bearer creation, with no durable compensation when those later steps fail.

Treating these as separate patches would leave intermediate unsafe states. The handoff channel, callback ordering, and credential lifecycle therefore need one shared contract.

## Desired outcome and invariants

- No CraftSky bearer, PDS access token, refresh token, or DPoP private key appears in a URL, browser history, callback HTML, application log, analytics event, or loopback payload.
- A handoff code has at least 256 bits of cryptographic entropy, is stored only as a hash, expires after a short fixed interval, and is bound to the initiating `X-Craftsky-Device-Id`.
- One code creates at most one pending child. Until a short confirmation deadline, same-code/same-device retries return the same AEAD-sealed receipt; other devices and purposes cannot replay it.
- Callback routing metadata is loaded before Indigo invokes its destructive authorization-request consume/delete callback; AppView logically consumes the row while retaining immutable non-secret request/attempt evidence, so loopback mode never silently degrades to deep-link mode.
- The authorization request is never visible without validated purpose/handoff/device/canonical-owner/epoch metadata. Callback racing login-start either observes the complete atomic insert or no request.
- A parent and child cannot be used by ordinary AppView/PDS operations until the client confirms successful secure storage and one transaction makes both active.
- Once an upstream credential is durably represented by a saved parent, it reaches exactly one of two terminal outcomes: an active CraftSky session or locally deleted credentials with upstream revocation attempted through the bounded cleanup path.
- If the token endpoint may have succeeded but initial local persistence/revocation cannot be proven, AppView records `exchange_ambiguous`, refuses code replay, alerts on the residual, and retains the non-secret attempt until the configured conservative provider expiry horizon. It does not claim the unknown credential was revoked.
- Callback, redemption, and cleanup races are resolved by row locks and compare-and-set state transitions, not by timing assumptions.
- Account-deletion reauthentication remains a distinct purpose. It mints neither an ordinary login exchange nor a CraftSky bearer, and its `deletion_only` parent is usable only by the matching accepted deletion job.
- Profile initialization uses only a callback-bound `ResumePendingOnboardingSession(attemptID, owner, sessionID, epoch)` capability while the owner and parent locks are held. Ordinary resume continues to reject `pending_handoff`.
- Callback finalization, redemption, and confirmation participate in the owner lifecycle fence. A committed terminal lifecycle can never be followed by a recreated OAuth parent, child, exchange, profile initialization, or PDS capability.
- Every callback HTML response, including errors, carries `Cache-Control: no-store`, `Pragma: no-cache`, `Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, and a restrictive nonce- or hash-based CSP.
- App and AppView diagnostics redact exchange codes and all session credentials.

## Scope

### In scope

- AppView OAuth authorization-request metadata lifecycle and callback ordering.
- A new handoff exchange/receipt store, unauthenticated-but-code-protected redemption endpoint, and authenticated confirmation endpoint under `/v1/auth/`.
- Transactional pending-child creation, idempotent receipt replay, and parent/child confirmation.
- Durable cleanup of abandoned or partially finalized OAuth sessions.
- Flutter route, controller, API-client, secure-storage flow, and generated router changes needed to redeem a code instead of accepting a bearer.
- CLI loopback payload and receiver changes.
- Verified Android App Links and iOS Universal Links, including the repository/deployment association files and release verification steps.
- Callback response security headers and secret-redaction tests.

### Out of scope

- Changing atproto OAuth, DPoP, scopes, or the PDS/AppView ownership boundary.
- Giving PDS credentials to Flutter.
- Generic OAuth libraries; the implementation continues to use Indigo behind AppView-owned lifecycle and transport controls.
- General account-deletion worker/purge semantics. This plan does own the callback's `deletion_only` credential state and verified deletion-complete link because changing either without the other breaks fresh reauthentication.
- General session expiry, refresh serialization, logout-all semantics, and authentication error classification; those are implemented by the grouped session-integrity plan for AV-009/010/011/020/021/035.
- General outbound SSRF and timeout policy, which belongs to AV-001/AV-017, although this plan depends on it for revocation and callback exchange calls.

## Design decisions

### One-time browser handoff, replayable bounded receipt

The callback URL parameter and loopback JSON field will be renamed to `code`. The server will not accept a bearer in that field and will not retain `token` as an alias. This uses the pre-production window to eliminate the vulnerable protocol instead of creating a permanent compatibility path.

The proposed endpoint is `POST /v1/auth/handoffs/exchange`, with a camelCase JSON body such as `{"code":"..."}` and the initiating `X-Craftsky-Device-Id` header. It returns a pending CraftSky bearer, opaque `receiptId`, confirmation deadline, and the minimum canonical identity needed for persistence. The client writes bearer, account, and receipt to secure storage, then calls `POST /v1/auth/handoffs/confirm` with that bearer, receipt, and device ID. Only confirmation activates ordinary API use.

Redemption is retryable only while the receipt is unconfirmed and unexpired. The exchange-row lock ensures the first call creates one pending child; a same-code/same-device retry decrypts and returns the same bearer/receipt. This avoids stranding an active credential when an HTTP response is lost or secure storage fails. Expiry without confirmation revokes/deletes the pending child and schedules parent revocation.

### Hash and bind the exchange secret

Generate the exchange code from 32 random bytes using `crypto/rand`, expose it as unpadded base64url, and store only SHA-256(code). Store the initiating device ID copied from the authorization request. Store the temporary bearer response only as AEAD ciphertext under a dedicated handoff-receipt key, with exchange, parent, child, and device identifiers as associated data. Plaintext exists only while serving redemption and is destroyed from durable state on confirmation or expiry. Do not make device ID the secret; it is an additional binding only.

### Explicit parent-session lifecycle

Add a lifecycle column to `oauth_sessions`, with constrained values such as `pending_handoff`, `active`, `deletion_only`, and `revocation_pending`. Indigo's first insert defaults to `pending_handoff`; subsequent refresh upserts preserve lifecycle rather than resetting it. Ordinary `GetSession`/`ResumeSession` paths accept only `active`. The callback alone may call `ResumePendingOnboardingSession(attemptID, owner, sessionID, epoch)` after acquiring the owner fence and parent-session lock; it verifies the login attempt, matching pending parent and epoch, and returns a narrowly typed onboarding client limited to the profile read/create operations required by `InitializeProfile`. The method is not exposed through `deps.NewPDSClient` or worker interfaces.

The pending child also needs a constrained state so its bearer authenticates only at confirmation/abandon endpoints. Confirmation locks receipt, parent, and child in the global order and moves parent/child to `active` in one transaction. Account-deletion callback completion instead moves its parent directly from `pending_handoff` to `deletion_only`, binds it to the deletion operation and a positive deletion-credential generation, and never creates a child.

This schema is also a coordination seam for the session-integrity plan. That plan should extend the same lifecycle instead of adding a second status model.

Callback processing, redemption, and confirmation also use `ownerlifecycle`. Persist the resolved canonical owner DID and current auth epoch with the authorization request when login starts. The callback loads them before Indigo processing, acquires the exclusive owner transition fence, re-reads lifecycle/purpose/epoch, and retains the fence across token exchange, initial `SaveSession`, callback-only onboarding resume, exchange creation or deletion-only binding, and local finalization. `terminal` therefore rejects before any new credential row can be saved; `departed` may proceed only through explicit login/onboarding; `deletion_pending` may proceed only for the matching deletion-purpose operation. Redemption and confirmation repeat epoch/lifecycle checks under the owner fence. DID-wide logout takes the same exclusive fence, increments auth epoch, and invalidates pending artifacts; a callback either finishes before logout and is revoked by it or observes the new epoch and cannot persist/activate.

Auth start must not treat a missing lifecycle row as an implicit epoch. Under the resolved-DID fence, `EnsureOnboardingOwner` inserts `(owner_did, departed, generation 1, auth_epoch 1)` if absent, then re-reads the authoritative row and rejects `terminal` or a superseding transition. This makes brand-new-DID login and logout/callback epoch checks use the same authority. The identifier passed to Indigo must remain bound to this row: prefer the parsed resolved `syntax.DID`; if Indigo must receive the original handle, require constant canonical equality between the first token response's `AccountDID` and the persisted owner before `SaveSession`, and best-effort revoke/fail closed on mismatch.

The exact cross-plan lock order is: every owner-effect fence in canonical DID order; optional object/work fence; every parent-session lock in canonical `(DID, sessionID)` order; then begin the short transaction. The exact database row-lock order inside it is: owner lifecycle/auth-epoch row; authorization-request rows by primary key; account-deletion operation if relevant; OAuth parent rows by session ID; CraftSky child rows by primary key; handoff exchange/receipt rows by primary key; cleanup/outbox rows by primary key. A path locks only the subsets it needs, but never a later class before an earlier one. Discovery queries may find IDs without locking and must revalidate after the ordered locks. Logout, callback, redemption, confirmation, cleanup, expiry, and deletion credential replacement share this order.

Make the fenced precondition explicit in the callback store adapter: initial `SaveSession` is callable only with the owner-fence plus callback-attempt capability/context created by the callback service, while ordinary refresh persistence uses the separately fenced session coordinator. A raw unfenced callback store must not remain constructible in dependency wiring. Do not acquire the owner fence from inside `SaveSession` after a parent-session/transaction lock is already held.

### Stage the token exchange honestly

Do not treat “`ProcessCallback` returned an error” as proof that no upstream credential was issued. Before the token request, atomically mark the complete auth-request row `exchange_started`, set `consumed_at`, and record a random attempt ID/start time without secrets. The callback integration must expose the parsed token response to an AppView adapter before first persistence. If `SaveSession` fails, use that in-memory token response for an immediate, bounded revocation attempt through the hardened client.

There is an unavoidable crash window after the authorization server returns tokens and before AppView durably saves them. The non-secret attempt marker makes this visible but cannot reconstruct a lost refresh token. On restart, `exchange_started` without a parent becomes `exchange_ambiguous`; the authorization code is never retried. Keep and alert that marker until a conservative configured residual horizon derived from provider guarantees has elapsed. If no finite refresh-token guarantee exists, retain a terminal security event and require operator/provider follow-up rather than claiming eventual revocation. Once `SaveSession` succeeds, normal `pending_handoff` compensation owns cleanup.

### Clean up and rotate deletion-only credentials

A deletion-purpose callback binds `deletion_only` only to its exact owner, operation ID, and credential generation. Intent cancellation or expiry acquires the exclusive owner fence, locks the operation and bound parent in global order, clears proof/binding, transitions the parent to `revocation_pending`, and enqueues bounded upstream cleanup. It cannot silently promote that parent to ordinary `active`.

An accepted deletion worker calls `ResumeDeletionSession(jobID, owner, sessionID, credentialGeneration)` under the owner fence and parent lock. Transparent access-token refresh persists with expected `row_version`. If refresh is terminal, the job becomes `reauth_required`; a new deletion-purpose callback atomically binds a higher credential generation, moves the old parent to `revocation_pending`, and fences stale worker leases from using it. Completion, terminal failure, or terminal DID state also queues the current deletion-only parent for cleanup. An accepted deletion job is not cancelable through the intent-cancel path.

Acceptance may already have revoked ordinary access, so replacement must use the existing public fresh-sign-in entry point rather than invent a status/recovery credential. After resolving the entered handle to the owner DID and taking its fence, login start recognizes the exact `deleting` owner with an accepted `reauth_required` operation and forces a purpose-bound replacement-deletion OAuth request for that operation. It cannot be selected by a client flag alone. The callback mints no bearer/child, atomically binds the higher credential generation, returns through the verified deletion-complete link, and never restores membership. All other deleting-owner login attempts fail closed. This preserves the account-deletion requirement that no status/recovery credential or route exists while still allowing later fresh PDS sign-in to refresh authority.

### Durable compensation over deferred best effort

Do not use a handler-local `defer` as the only cleanup mechanism. A callback can be canceled or the process can crash after Indigo persists credentials. Record `pending_handoff` and its deadline durably. On local finalization failure, mark the row `revocation_pending` in a short transaction. A worker claims such rows, calls upstream revocation with the AV-001/AV-017 bounded client, then deletes the local parent row. When upstream revocation is unavailable, retry to a documented ceiling and delete locally after the token's remaining useful window; surface the residual upstream risk in metrics. This durable parent cleanup is distinct from the pre-parent `exchange_ambiguous` residual above.

### Read handoff metadata before destructive processing

Generalize the existing `AuthRequestMetadataFromContext` seam so login and deletion starts pass one validated typed metadata value into `StartAuthFlow`. Resolve the canonical owner DID first; under its fence call `EnsureOnboardingOwner`, re-read lifecycle, and snapshot its real auth epoch. Validate mode/device/loopback/purpose, then let `PostgresAuthStore.SaveAuthRequestInfo` insert all metadata with Indigo state/data in one statement. Pass the resolved DID to Indigo when possible; otherwise enforce token-response `AccountDID == owner_did` before any credential save. Remove `recordHandoff`, the `request_uri` follow-up lookup/update, and compatibility fallback. `CallbackHandler` reads the complete row by state before processing; missing or inconsistent metadata fails closed.

### Verified links are the production mobile entry point

Replace the unverified `craftsky://` filters with an HTTPS origin controlled by CraftSky and verified by Android's `assetlinks.json` and Apple's `apple-app-site-association`. Both `/auth/complete` and `/account-deletion/reauth-complete` must be associated, routed, and tested before removing `craftsky:///auth/complete` and `craftsky:///account-deletion/reauth-complete`. Keep loopback HTTP only for the local CLI and only on `127.0.0.1` with a valid port. A local-development-only custom scheme may be retained only if it carries no bearer, is compile-time/environment gated, and cannot ship in release manifests; the preferred approach is to test verified HTTPS links in dev builds too.

## How the shared update closes each finding

### AV-008 — A long-lived bearer crosses a claimable custom URL scheme

The browser receives a short-lived code rather than a bearer. Device-bound redemption over the configured HTTPS AppView origin means interception alone is insufficient from a different installation; receipt replay is limited to the same device, one pending child, and a short confirmation window. Verified App/Universal Links remove the claimable production scheme for both login and deletion reauthentication. Security headers prevent caching, referrer propagation, MIME sniffing, and uncontrolled callback-page script execution.

### AV-018 — Loopback OAuth handoff is deterministically lost

The callback reads `handoff_mode`, `loopback_redirect_uri`, `device_id`, and purpose before Indigo invokes its destructive consume/delete callback. AppView maps that callback to a logical consume so immutable metadata and attempt evidence remain available. It treats missing data as an error instead of defaulting. The loopback page posts the handoff code to the retained, revalidated URI, so the CLI can then redeem it directly. A real-store integration test must exercise Indigo's destructive callback while proving AppView retains the logical record, rather than rely on a mock that leaves physical deletion semantics untested.

### AV-019 — Partial OAuth callback failure retains unreachable upstream credentials

The first upstream credential insert is explicitly `pending_handoff`, never implicitly active. Profile initialization failure, exchange-row creation failure, response-render failure, expiration, and process restart all converge through `revocation_pending` and the durable cleanup worker. A lost redemption response returns the same sealed receipt on retry; secure-storage failure leaves parent/child pending until expiry cleanup. Successful client confirmation is the only transaction that activates them. Deletion-purpose credentials instead become `deletion_only` and remain unreachable to ordinary login/session APIs.

## Unified implementation plan

1. **Write failure-first contract tests.** In `appview/internal/auth/handlers_test.go`, add tests for atomic auth-request metadata, `exchange_started`/ambiguous persistence, initial-save revocation compensation, callback-only onboarding resume, lost-response replay, secure-storage failure/expiry cleanup, confirmation, deletion-only cancel/replace/resume, logout epoch races, and pending-session cleanup. Update Flutter tests to expect `code`, not `token`.
2. **Add the lifecycle, attempt, and exchange schema.** Create the next paired migration in `appview/migrations/` to persist canonical owner DID/auth epoch and typed metadata on the authorization request; add its `ready`, `exchange_started`, `exchange_ambiguous`, and finalized/cleanup fields; add constrained parent/child lifecycle, row/credential versions, pending/cleanup timestamps and claim indexes; and add `oauth_handoff_exchanges` with code hash, epoch, parent/child keys, device, state, opaque receipt ID, AEAD ciphertext/nonce/key version, deadlines, and timestamps. Use foreign keys with `ON DELETE CASCADE`, unique attempt/code/receipt IDs, state-shape/deadline constraints, and claim indexes. Update every focused auth/account-deletion fixture.
3. **Make the OAuth store lifecycle-aware and fence-capable.** In `appview/internal/auth/store.go`, make auth-request deletion logical/consuming so exchange-attempt evidence survives; preserve lifecycle on refresh `SaveSession`; reject non-active rows from ordinary `GetSession`; and add narrow methods for pending onboarding, deletion-only resume, lifecycle transitions, ambiguous attempts, cleanup, and deletion credential replacement. Initial `SaveSession` requires matching owner-fence/callback-attempt/epoch context; refresh saves use the parent coordinator and expected version. Do not expose an unfenced callback store or raw credential JSON beyond auth.
4. **Add transaction-capable pending-bearer and receipt primitives.** Refactor `appview/internal/auth/craftsky_session.go` so secure token generation/insertion can run on `pgx.Tx`. The first redemption creates one pending child and seals its response; retry returns that child/receipt. A separate confirmation transaction activates parent/child and deletes ciphertext/code hash. Keep AEAD operations behind a typed sealer with key-version support and redacted errors.
5. **Insert auth-request metadata atomically.** Replace `recordHandoff` and the post-`StartAuthFlow` `request_uri` lookup/update. Resolve the canonical login owner before starting OAuth; under its fence call `EnsureOnboardingOwner`, reject terminal/superseded state, and snapshot its real epoch. Build generalized typed metadata in context (purpose, owner, epoch, mode, device, validated loopback destination and deletion job when applicable), pass the resolved DID to Indigo where supported, and make `PostgresAuthStore.SaveAuthRequestInfo` insert it with Indigo state/data. A real-store barrier invokes callback after insert but before `StartAuthFlow` returns and proves it sees either the complete row or no row, never partial metadata.
6. **Stage, bind, fence, and compensate callback processing.** In `appview/internal/auth/handlers_oauth.go`, load complete metadata, acquire the exclusive owner fence, and re-read lifecycle/purpose/epoch. Durably mark `exchange_started` before the token call; retain the fence through token response, parent lock, initial `SaveSession`, callback-only `ResumePendingOnboardingSession`, profile initialization, exchange creation or deletion-only binding, and finalization. Before `SaveSession`, require the returned canonical `AccountDID` to equal the persisted owner (even if Indigo was started with a DID), and best-effort revoke/fail closed on mismatch. Capture token data before the first save so save failure can attempt immediate bounded revocation. Record unconfirmed revocation or crash-without-parent as `exchange_ambiguous`; never retry the authorization code or claim cleanup succeeded. Terminal/superseded/old-epoch state never calls the token endpoint.
7. **Implement redemption and confirmation.** Add handlers/store/service in `appview/internal/auth/`, register exact exchange/confirm routes and policies in `appview/internal/routes/policy.go` and `routes.go`, and apply AV-015 abuse limits. Exchange requires code plus device but no bearer; confirmation requires pending bearer, receipt ID, and device. Both use the owner lifecycle fence and recheck terminal/purpose/generation before creating or activating state. Collapse unknown, expired, wrong-device, wrong-purpose, terminal-owner, and unavailable receipts to safe envelopes without consuming a valid row on a mismatch.
8. **Implement cleanup and deletion-credential rotation.** Add bounded processors under `appview/internal/auth/`, wire them from `deps.go`/`main.go`, and use leases or `FOR UPDATE SKIP LOCKED`, deadlines, capped backoff, idempotency, and secret-safe logs. Clean pending/revocation parents and expired receipts normally; retain/alert non-secret `exchange_ambiguous` attempts for the configured provider residual horizon. Account-deletion cancel/expiry revokes its unaccepted `deletion_only` parent. Accepted-job refresh uses expected version; terminal refresh moves the job to `reauth_required`. Teach the public fresh-sign-in start to derive—never trust from client input—the replacement-deletion purpose when the resolved owner is deleting with that exact accepted operation. A successful callback mints no child, atomically replaces/increments credential generation, revokes the old parent, and leaves lifecycle deleting.
9. **Harden callback HTML.** In `appview/internal/auth/handlers_render.go`, add a shared response-header helper for success and error pages, remove `Token` and the dev-token display, carry only `Code`, and use a nonce/hash CSP or an external static script. The loopback payload becomes `{"code":"..."}`. Rendering errors after the code is made durable leave it to expire/cleanup.
10. **Change Flutter to redeem, persist, then confirm.** Replace `token` with `code` in `app/lib/router/router.dart`, generated routes, `auth_complete_page.dart`, and related tests. `AuthController.completeFromDeepLink` resolves the stable device ID, redeems with a credential-free client, atomically stores bearer/account/receipt, then confirms with the pending bearer. Lost exchange responses retry the code; storage failure does not confirm and lets bounded cleanup revoke it; restart with a stored pending receipt resumes confirmation. Add the verified deletion-complete route without passing an ordinary bearer.
11. **Change CLI loopback behavior.** Update the loopback receiver and request client under `appview/cmd/cli/` to accept a code, redeem it using the initiating device ID, durably store the pending bearer/receipt, call `/v1/auth/handoffs/confirm`, and treat/display the bearer as usable only after confirmation succeeds. Persist enough pending state to retry confirmation safely after timeout or restart. Reject a legacy `token` field.
12. **Configure verified links.** Update `app/android/app/src/main/AndroidManifest.xml` with HTTPS `android:autoVerify="true"` handling for the exact login-complete and account-deletion-reauth-complete paths, then remove release custom schemes. Update iOS entitlements/project configuration and router coverage for both paths before removing `CFBundleURLTypes`. Add or document both paths in `/.well-known/assetlinks.json` and `/.well-known/apple-app-site-association`, including correct package/team identifiers and no redirects.
13. **Update contracts and operational docs.** Amend the OAuth BFF and Flutter auth design documents, environment examples for the canonical link origin and exchange/cleanup durations, and runbooks for cleanup backlog alerts. Call out the intentional breaking wire change.
14. **Remove obsolete paths.** Delete token-bearing template fields, custom-scheme production handling, bearer-keyed handoff client construction, post-insert `recordHandoff`/`request_uri` mutation, fallback-to-deep-link behavior, ordinary resume of pending/deletion parents, and tests that assert them. Search for `auth/complete?token`, `craftsky:///auth/complete`, `Dev-mode token`, loopback `token` JSON, and raw `ResumeSession`/`SaveSession` bypasses before closing.

## Data, schema, migration, and reconciliation plan

- Add one reversible migration for auth-request typed metadata/state/attempt fields, owner auth epoch propagation, parent/child lifecycle and versions, deletion credential generation, and `oauth_handoff_exchanges`. The down migration drops receipt/attempt dependencies before lifecycle/epoch columns and constraints.
- Because there are no production users, do not invent a compatibility backfill for existing credentials. For local databases, delete existing `craftsky_sessions`, `oauth_sessions`, and `oauth_auth_requests` during the development reset and require fresh login after migration. If the migration must preserve test fixtures, set existing parent rows to `active` only in fixture setup, not as a production assumption.
- Exchange rows contain no plaintext secret. During the unconfirmed window they contain only an AEAD-sealed bearer response and key version; the code remains hash-only. Confirmation destroys ciphertext/code hash, while expiry atomically disables the pending child and queues parent revocation. Supply the handoff key through secret storage, never log it, support rotation by version, and fail closed when it is absent outside explicit unit-test construction.
- Authorization requests are logically consumed rather than immediately erased. Retain only non-secret callback attempt state after finalization/cleanup. `exchange_ambiguous` stores owner, epoch, provider/issuer identity, attempt timestamps, safe failure category, and residual-horizon status—never access/refresh tokens, authorization code, DPoP key, or raw callback query.
- Index all owner/epoch/state predicates used by logout-all, callback claim, ambiguous-attempt sweep, and deletion-credential replacement. The accepted deletion operation owns the current session ID plus positive credential generation; replacing it makes an old worker's generation stale.
- Coordinate with AV-015's normalized `request_uri` column and auth-request sweeper so a single migration series owns indexes and cleanup without duplicating tables.
- Validate migration up/down/up against the PostgreSQL version used by the full test gate. Verify foreign-key cascades and claim indexes with representative query plans.
- No public PDS record reindex is required. This update changes credentials and private handoff state only.

## API, client, configuration, and operations impact

- Breaking API addition/change: `POST /v1/auth/handoffs/exchange` consumes `{code}` plus `X-Craftsky-Device-Id` and returns `{token,did,handle,receiptId,confirmBy}` for a pending child. `POST /v1/auth/handoffs/confirm` consumes `{receiptId}` with that pending bearer/device and activates it. The previous deep-link/loopback token contract is removed.
- The public login-start request does not gain a client-selected deletion purpose. After canonical DID resolution, AppView alone maps a deleting owner with the exact accepted `reauth_required` operation to replacement-deletion OAuth; its callback returns through the deletion-complete verified link and emits no handoff code or child bearer. No deletion status/recovery credential or route is introduced.
- Login-complete and deletion-reauth-complete URIs use exact HTTPS verified-link paths. Their canonical origin comes from validated configuration shared with AV-022/023, never from request `Host`.
- Add positive, bounded configuration for exchange TTL, shorter confirmation TTL, handoff-receipt encryption key/version, pending-login cleanup grace, cleanup lease, retry count/backoff, and per-operation timeout. The AV-030 parser validates their relationships and rejects a missing/invalid key outside explicit unit-test construction.
- The callback response must be marked non-cacheable at AppView and at any reverse proxy/CDN. The auth-complete path must not be logged with query strings.
- Alert on oldest `pending_handoff`, `revocation_pending`, `exchange_started`, and `exchange_ambiguous`; provider residual-horizon breaches; cleanup/revocation exhaustion; deletion jobs awaiting reauthentication; exchange failures by safe class; and attempted legacy token handoff.
- Publish Android/iOS association files from the canonical HTTPS origin before enabling release link filters; verify them from external networks and both platforms' diagnostic tooling.

## Security, failure, and race considerations

- Use constant-time hash comparison through indexed hash lookup semantics and never log raw codes. Redact request bodies on this endpoint.
- Rate-limit login starts and exchange attempts using AV-015's trusted client-IP/global boundary. Device ID alone is attacker-controlled and cannot be the outer abuse key.
- `SELECT ... FOR UPDATE` (or one conditional `UPDATE ... RETURNING`) serializes redemptions. Only the first creates a pending child; same-device retries decrypt the same receipt and cannot extend its original deadline.
- Bind exchange TTL to server time. Client clock affects only UI timeout messaging, not authorization.
- A wrong-device attempt must not consume a valid code; it must be indistinguishable externally from other invalid-code cases and remain rate limited.
- Distinguish three initial exchange outcomes. If the token call fails before a response, mark the attempt failed. If tokens are parsed but `SaveSession` fails, attempt immediate bounded revocation and record either confirmed revocation or `exchange_ambiguous`. If `SaveSession` commits, a crash before exchange creation leaves a purgeable `pending_handoff` parent. A crash between token response and first save leaves `exchange_started` without a parent; recovery marks it ambiguous and never claims or retries the code. Because token exchange/save ran under the owner fence and epoch, none of these can recreate a parent after terminal/logout commit.
- Secure-storage failure leaves no active bearer. Confirmation activates parent/child only after persistence; unconfirmed expiry cleanup revokes/deletes them. A lost confirmation response is safe because activation is idempotent and the bearer is already durably stored.
- Cleanup, redemption, and confirmation lock rows in one documented order. Cleanup cannot revoke an already confirmed parent; redemption/confirmation cannot revive `revocation_pending` state.
- Terminal owner transitions share that fence. Callback finalization, redemption, and confirmation cannot persist or activate state after terminal commit; terminal purge waits for any earlier already-linearized auth transition and then removes it.
- Logout-all shares the exclusive owner fence and increments auth epoch. It invalidates active/pending login artifacts and all auth requests/exchanges/receipts; only the exact accepted-job deletion credential may survive, childless and unusable outside `ResumeDeletionSession`.
- `ResumePendingOnboardingSession` requires the live callback attempt capability, matching epoch/owner/session, owner fence, and parent lock. It exposes only the profile initialization surface and cannot be retained after callback finalization. Ordinary resume and every worker reject pending parents.
- Canceling or expiring an unaccepted deletion intent and replacing/completing an accepted deletion credential use the common lock order and local-first revocation. A stale credential generation or worker lease cannot refresh or perform a PDS delete. Replacement authority comes only from a later fresh PDS sign-in whose resolved DID maps under the fence to the exact accepted `reauth_required` operation; no status/recovery credential or route is added.
- Redirect and callback URL construction must use the canonical configured origin and validated paths. Do not reintroduce `r.Host` or unrestricted redirect input.
- The cleanup worker's upstream revocation uses the hardened transport and bounded operation context from AV-001/017. Local credential deletion cannot be blocked forever by an unavailable AS.
- Account-deletion callback states remain purpose-bound. A state created for deletion cannot be exchanged/confirmed at login endpoints, and its `deletion_only` parent cannot pass ordinary resume or mint a child bearer.

## Unified test plan

### Unit tests

- Secure code generation shape, hash-only persistence through confirmation, receipt AEAD round-trip/AAD/key-version behavior, TTL validation, device comparison, and redacted string/error formatting.
- Callback header helper covers success and error pages; CSP test proves the template has no unsafe inline secret interpolation.
- Typed auth-request metadata validation rejects missing owner/epoch/mode/device, invalid loopback URI, wrong purpose, and inconsistent deletion job; there is no follow-up update API. `EnsureOnboardingOwner` covers first-login insert, existing row, terminal race, and real epoch snapshot.
- `ResumePendingOnboardingSession` accepts only a live matching callback attempt under the owner/parent capabilities and returns the limited profile client; ordinary resume rejects pending/deletion-only parents.
- Exchange attempt decision table covers no token response, token response plus save failure/revocation success, revocation failure, crash ambiguity, saved pending parent, and residual-horizon handling without secret persistence.
- Deletion-only decision table covers intent cancel/expiry, accepted-job exemption, refresh success, terminal refresh to `reauth_required`, later fresh-sign-in replacement, credential generation, stale worker, completion, and terminal owner.
- Flutter models/router parse `code` and reject/ignore `token`; controller maps expired, confirmed, or unavailable handoffs without persisting partial state.

### Database and integration tests

- Use the real `PostgresAuthStore` and real Indigo callback semantics: typed handoff/owner/epoch metadata is inserted atomically, logical consumption preserves attempt evidence, and loopback callback succeeds without a post-start update.
- Pause `StartAuthFlow` immediately after `SaveAuthRequestInfo`, run callback concurrently, and prove it sees the complete atomic metadata row. Also invoke callback before insert and prove safe not-found; no schedule observes state/data without owner/device/purpose/epoch.
- Start login for a brand-new DID and prove `EnsureOnboardingOwner` creates epoch 1 under the owner fence. Race first insertion against a terminal tombstone and prove the re-read rejects auth. Rebind a handle between discovery and PAR/callback; prove Indigo receives the resolved DID or the token response's mismatched `AccountDID` is rejected before `SaveSession`, with bounded best-effort revocation and no parent.
- First redemption creates exactly one pending child/receipt and returns canonical identity; confirmation atomically activates parent/child and destroys recoverable receipt secrets.
- Concurrent same-device redemptions from many goroutines return the same bearer/receipt and create one child under `go test -race`; other-device attempts safely fail.
- Wrong-device, expired, already-confirmed, unknown, and wrong-purpose codes cannot create a new child session.
- Inject failure/crash after `exchange_started`, after token response, during first `SaveSession`, after save, during pending-onboarding resume, at exchange insert, pending-child/receipt insert, receipt sealing, confirmation activation, and each transaction commit. Assert a pending parent, confirmed revocation, or explicit non-secret ambiguous residual—never an incorrectly “clean” or active unreachable credential.
- Restart between `ProcessCallback` and finalization, then run cleanup; assert the pending parent is claimed, upstream revocation is attempted once per retry policy, and local credentials are deleted.
- Race cleanup against redemption/confirmation with barriers and prove only active-and-reachable or deleted terminal outcomes.
- Add three callback/terminal barriers: terminal commits before `ProcessCallback`; process crashes after fenced `SaveSession` but before finalization and terminal then commits; and terminal races confirmation before activation. Assert terminal either prevents the initial save or waits for/re-purges the earlier save, and no parent, child, exchange, profile initialization, or PDS capability survives terminal completion.
- Complete account-deletion OAuth, assert the parent is `deletion_only`, prove ordinary/pending resume and exchange reject it, and prove only the matching accepted job/generation can resume. Cancel and expire unaccepted intents and assert local invalidation plus cleanup. Force terminal refresh, restart the app without an ordinary bearer, enter the normal fresh-sign-in flow, and prove the server derives the replacement-deletion purpose only after resolving the same deleting DID and accepted `reauth_required` job. Complete the verified-link callback with a higher credential generation; prove no bearer/membership is restored, the worker resumes, and the old parent/stale worker cannot refresh or delete.
- Run callback, redemption, confirmation, logout-all, expiry, cleanup, and deletion replacement concurrently using reverse input order, a short PostgreSQL `lock_timeout`, and barriers at every ordered row class. Assert no deadlocks and only allowed final states.

### Client and platform tests

- Flutter widget/controller tests cover successful exchange/persistence/confirmation, lost exchange response and same-code retry, secure-storage failure that leaves the session pending until bounded expiry cleanup, restart with a stored pending receipt, lost confirmation response and idempotent retry, add-account flow, and redaction.
- CLI end-to-end test starts a real loopback listener, completes callback through the real store, receives a code, redeems and durably records the pending receipt, confirms it, and asserts active parent/child state while proving no bearer crossed the browser-to-loopback request. Drop the first confirmation response and prove restart/retry confirms idempotently without creating a second child.
- Android release build verifies only the intended HTTPS host and exact login/deletion paths; an unrelated app claiming `craftsky://` cannot receive either production completion.
- iOS release build verifies the Associated Domain and both Universal Link paths. Test each web fallback when the app is absent.

### Security and fault tests

- Secret scan callback HTML, `Location`/link URLs, request logs, Sentry attributes, loopback payloads, and Flutter provider diagnostics for known bearer/access/refresh/code fixtures.
- Replay the same code concurrently, from a wrong device, after confirmation, and after expiry; only same-device unconfirmed retries return the existing receipt.
- Simulate callback disconnect/template write failure, token-response/initial-save crash, AS revocation timeout, DNS failure, ambiguous residual expiry, and cleanup-worker restart. Verify logs/metrics never describe an unconfirmed revocation as successful.
- Verify callback and exchange responses are not cached and do not leak query strings through referrers.

### Regression commands

- Run focused Go tests for `internal/auth`, `internal/routes`, and `cmd/cli` with the PostgreSQL fixture enabled.
- Run focused Flutter auth/router/provider tests and regenerate the router/provider outputs.
- Run the repository database-backed race suite and the AV-033/036 quality gates after platform/config updates.

## Per-ID traceability and acceptance criteria

### AV-008

- [ ] No callback, redirect, loopback body, manifest, log, or test fixture carries a long-lived bearer outside the direct exchange response.
- [ ] The handoff code is random, hash-only, device-bound, and short-lived; its bounded retry window creates one pending child and never exposes a bearer through the browser.
- [ ] Android App Links and iOS Universal Links are verified for exact login-complete and deletion-reauth-complete paths before production custom-scheme handling is removed.
- [ ] Success and error callback pages send no-store/no-referrer/nosniff and restrictive CSP headers.
- [ ] Automated secret-scanning and replay tests pass.

### AV-018

- [ ] Callback code loads required handoff metadata before invoking the destructive Indigo callback.
- [ ] Login/deletion start resolves the canonical owner and atomically inserts owner/epoch/purpose/device/handoff metadata with Indigo state/data; `recordHandoff` and `request_uri` follow-up mutation are removed.
- [ ] A callback-vs-start real-store barrier proves partial auth-request metadata is never observable.
- [ ] A real-store loopback integration test receives and redeems the code after Indigo invokes its destructive request-consumption callback, while AppView's immutable metadata/attempt record remains available through logical consumption.
- [ ] Missing or invalid handoff metadata fails closed; there is no implicit deep-link fallback.
- [ ] Loopback destination validation is applied at ingress and immediately before egress.

### AV-019

- [ ] Ordinary session lookup cannot use a `pending_handoff`, `deletion_only`, or `revocation_pending` parent, nor an unconfirmed child.
- [ ] Every failure after the parent is durably saved schedules cleanup, and expired abandoned handoffs are swept without another callback request.
- [ ] First redemption atomically creates one pending child plus sealed receipt exactly once while retaining the code hash for same-device retries; confirmation atomically activates parent/child and destroys the code hash and receipt ciphertext.
- [ ] Lost redemption responses are recoverable with the same code/device/receipt; secure-storage failure leaves no active session and expiry cleanup removes the pending credential.
- [ ] Crash/fault/race tests demonstrate that each durably saved parent becomes active and reachable or locally deleted with bounded revocation handling.
- [ ] A token response followed by pre-parent process death converges from `exchange_started` to `exchange_ambiguous`: the authorization code is never replayed, no credential secret is retained, the marker remains through the provider residual horizon/operator alert policy, and no revocation is falsely claimed. Initial-save errors attempt immediate revocation only while token material is available.
- [ ] Callback profile initialization can resume `pending_handoff` only through the attempt-bound onboarding capability under owner/parent locks; ordinary resume rejects it.
- [ ] Cancel/expiry cleans an unaccepted `deletion_only` parent. After accepted-job terminal refresh, a later normal fresh PDS sign-in is server-derived as replacement authority for the exact deleting owner/job, mints no bearer or membership, increments credential generation, and fences the prior parent and stale workers; no status/recovery credential or route is added.
- [ ] Auth request, operation, parent, child, exchange/receipt, and cleanup row locks follow the shared exact order; logout/callback/refresh/confirmation/deletion-replace barriers do not deadlock.
- [ ] Cleanup backlog and exhaustion are observable without exposing credentials.
- [ ] A terminal lifecycle committed between upstream exchange, local finalization, redemption, or confirmation always wins; barrier tests leave no recreated parent, child, exchange, profile, or PDS capability.

## Dependencies and coordination

- **AV-001 and AV-017:** must supply the hardened, bounded OAuth/PDS client used for token exchange and revocation. Do not ship the cleanup worker on `http.DefaultClient`.
- **AV-009/010/011/020/021/035 grouped session-integrity update:** must reuse parent/child lifecycle and owner auth epoch, define active and narrow pending/deletion resume consistently, own DID-wide epoch revocation/outbox behavior, and share the exact advisory/row-lock order.
- **AV-002/003/006/007 grouped account lifecycle:** supplies the owner fence and terminal/departed/deletion-pending rules used by auth start, callback exchange/save/finalization, redemption, confirmation, deletion credential rotation, and compensation. Auth cannot create state for a terminal or old-epoch owner.
- **AV-015:** should provide the outer trusted-IP/global limiter, normalized `request_uri`, capacity ceiling, and independent auth-request cleanup. Coordinate migration numbering and auth endpoint policy.
- **AV-022/023/024/030 grouped configuration-hardening update:** supplies the canonical validated public origin, verified-link host, positive TTL/deadline parsing, and safe dev exposure defaults.
- **AV-033 and AV-036:** provide the required PostgreSQL/race/static/vulnerability gates used as the completion gate.
- **Account-deletion flow:** purpose checks, exact-handle confirmation, the narrow `social.craftsky.*` deletion capability, and its dedicated callback are load-bearing. Include an end-to-end callback-to-worker acceptance test whenever callback/store/link schema changes.

## References

- [AppView OAuth BFF design](../superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [Flutter authentication design](../superpowers/specs/2026-04-21-flutter-auth-design.md)
- [AppView API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [Settings and permanent-deletion requirements](../changes/2026-08-10-settings-page/01-requirements.md)
- [AT Protocol architecture reference](../../atproto-craft-social-app-reference.md#authentication)
- [AT Protocol OAuth specification](https://atproto.com/specs/oauth)
- [Android App Links](https://developer.android.com/training/app-links/about)
- [Apple Universal Links](https://developer.apple.com/library/archive/documentation/General/Conceptual/AppSearch/UniversalLinks.html)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
