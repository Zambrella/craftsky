# AV-009, AV-010, AV-011, AV-020, AV-021, and AV-035 — Session integrity and revocation

- Findings: AV-009, AV-010, AV-011, AV-020, AV-021, AV-035
- Severities: High (AV-009, AV-010, AV-011); Medium (AV-020, AV-021); Low (AV-035)
- Priority/order: Authentication and session integrity; land after the verification foundation and coordinate with the OAuth handoff lifecycle schema
- Status: Planned
- Sources: [AV-009](../2026-08-12-appview-code-audit.md#av-009--concurrent-token-refresh-can-revoke-a-newly-valid-oauth-session), [AV-010](../2026-08-12-appview-code-audit.md#av-010--logout-leaves-the-bearer-valid-when-auxiliary-cleanup-fails), [AV-011](../2026-08-12-appview-code-audit.md#av-011--authentication-store-outages-are-returned-as-401-and-erase-client-sessions), [AV-020](../2026-08-12-appview-code-audit.md#av-020--configured-oauth-expiry-does-not-expire-bearer-only-access), [AV-021](../2026-08-12-appview-code-audit.md#av-021--all-devices-logout-leaves-other-parent-oauth-credentials-active), [AV-035](../2026-08-12-appview-code-audit.md#av-035--session-throttle-maps-grow-for-the-lifetime-of-the-process)

## Shared implementation strategy

Make the parent OAuth session and its child CraftSky bearers one explicit server-owned lifecycle, enforced by a single session service. Authentication, authenticated PDS calls, refresh persistence, expiration, single-device logout, all-device logout, auxiliary cleanup, and background revocation must all use that service rather than issuing unrelated SQL calls in handlers and dependency closures.

The shared design has four parts:

1. **Authoritative database state.** Parent rows have a constrained lifecycle and monotonically increasing version. The owner lifecycle also has a monotonically increasing `auth_epoch`, copied onto every auth request, parent, child, exchange, and receipt. A bearer lookup succeeds only when child/parent state is active, their epoch equals the owner epoch, and all absolute/inactivity limits pass. Expiration and revocation are compare-and-set transitions.
2. **Serialized PDS use.** Every authenticated PDS operation for one `(DID, OAuth session ID)` runs through a coordinator that prevents two Indigo session instances from refreshing the same rotating token concurrently. The database remains the cross-process authority; a version check prevents a stale `invalid_grant` from deleting newer credentials.
3. **Local revocation first.** Logout and expiry make local access unusable in a short transaction before any push-store or authorization-server call. Slow/failing auxiliary work is represented durably and processed with bounded retries.
4. **Bounded activity updates.** Replace process-lifetime throttle maps with conditional SQL timestamp updates. Correct expiry behavior no longer depends on which AppView process last saw a token.

This is a breaking session-contract correction. Existing local sessions can be invalidated and users can sign in again; there is no value in preserving ambiguous expiry or partial logout semantics before launch.

## Problem to solve

The current implementation treats several security decisions as incidental side effects:

- Indigo's per-instance mutex cannot coordinate the separate sessions constructed for concurrent requests.
- A losing refresh can classify `invalid_grant` as terminal and delete credentials already rotated by another request.
- Logout performs fallible push and upstream work before the local bearer is revoked.
- Middleware maps both invalid credentials and database failures to 401, while Flutter intentionally deletes local session state on 401.
- Bearer authentication checks only the child row's token hash and `revoked_at`; it does not enforce the parent OAuth lifetime.
- `all=true` handles only the presented parent even when several parent credentials exist for the DID.
- in-memory activity-throttle maps retain every observed token/session for the process lifetime and do not coordinate across replicas.

These defects share the same root cause: there is no authoritative lifecycle service spanning the parent/child relationship.

## Desired outcome and invariants

- At most one AppView operation may refresh a specific parent OAuth session at a time across goroutines and replicas.
- A stale refresh failure cannot revoke or delete a parent whose version advanced after the caller loaded it.
- CraftSky authentication succeeds only for an unrevoked child of an `active`, unexpired parent and an unexpired child inactivity window.
- A database/dependency failure is never presented as invalid credentials. Only missing, malformed, revoked, expired, or otherwise intentionally invalid credentials return 401.
- Once local logout commits, the affected bearer(s) cannot authenticate even if push cleanup or upstream token revocation is slow, unavailable, or permanently fails.
- `all=true` takes the exclusive owner fence, increments the DID's auth epoch, and invalidates every active or pending login parent/child, auth request, exchange, and receipt before returning success. A childless `deletion_only` parent is exempt only while it is the exact credential generation bound to an accepted operation whose owner lifecycle is `deleting` and whose worker state is active or retrying; the same transaction rebases only that parent to the new auth epoch.
- Authorization-server revocation is attempted for every affected parent with bounded I/O, durable retries, idempotency, and observable exhaustion.
- Single-device logout does not revoke sibling children or their shared parent.
- Activity timestamps are updated with bounded database writes and no unbounded process map.
- No path logs bearer tokens, access/refresh tokens, DPoP keys, refresh-lock keys, or raw cleanup-job payloads.

## Scope

### In scope

- Owner auth epoch plus auth-request/exchange/receipt and parent/child session lifecycle, versioning, expiry, and query behavior.
- Cross-goroutine and cross-replica coordination for authenticated Indigo/PDS operations.
- Stale-refresh detection and retry/terminal classification.
- Single-device and all-device local revocation ordering.
- Durable upstream OAuth and push-subscription cleanup processing.
- Middleware status/error classification and Flutter's 401-only invalidation contract.
- Replacement of `lastSeenMemory` and `deviceIDMemory` with database-conditional updates.
- Proactive expiry/revocation sweeps and operational metrics.

### Out of scope

- The bearer-to-one-time-code OAuth handoff itself, owned by AV-008/018/019.
- OAuth/PDS SSRF, response-size, and timeout transport implementation, owned by AV-001/017; this plan consumes that client.
- Membership departure/account deletion revocation triggers, owned by AV-002/003/006/007, although those triggers must call the same session service.
- An end-user active-sessions management UI. The schema/API may support it later, but this update only implements logout semantics.
- Persisting PDS credentials on the Flutter device; this remains forbidden.

## Design decisions

### One lifecycle model

Use the `oauth_sessions.lifecycle_state` introduced by the grouped OAuth-handoff plan, or introduce it here if this work lands first. At minimum, support `pending_handoff`, `active`, `deletion_only`, and `revocation_pending`. Ordinary session resume requires `active`; only the typed, matching-job `ResumeDeletionSession` capability may load `deletion_only`. Do not create a second Boolean such as `disabled` whose combinations could contradict lifecycle state.

Add a monotonic `row_version BIGINT NOT NULL DEFAULT 1`; every credential rotation or lifecycle transition increments it. Use the `revocation_pending` parent row itself as the upstream-revocation queue, with explicit `cleanup_attempts`, `cleanup_next_attempt_at`, `cleanup_lease_token`, `cleanup_lease_expires_at`, and bounded last-result category columns. Use a separate `auth_auxiliary_cleanup_jobs` table for installation/account push cleanup because single-child logout must not transition or retain a parent solely to represent that work. The version is security state, not an optimistic UI value.

Add a positive monotonic `auth_epoch` to the owner lifecycle row. Auth start snapshots it into the atomically inserted authorization request; initial parent persistence, handoff exchange/receipt creation, child issuance, and confirmation all copy and recheck it. Ordinary authentication requires owner, parent, and child epochs to agree. Incrementing the epoch is therefore the DID-wide linearization point even for a callback or pending child whose row was not in an earlier query snapshot.

### DID-wide logout and the deletion-only exception

HTTP `all=true` calls a typed `RevokeAllForDID(UserLogout)` operation that acquires the exclusive owner-effect fence and every existing parent-session advisory lock in canonical `(DID, sessionID)` order. Its transaction increments `auth_epoch`, consumes/cancels every owner login or unaccepted deletion authorization request, invalidates every exchange and sealed receipt, revokes every pending or active child, and transitions every pending or active ordinary parent to `revocation_pending`. Existing `revocation_pending` parents stay queued. This covers rows created before the fence was acquired and makes any old-epoch callback/finalizer fail even if a cleanup query missed it. Lifecycle departure and terminal purge use distinct typed reasons that permit no deletion-only exemption; the preservation choice is internal and cannot come from an HTTP flag.

The only exemption is a childless `deletion_only` parent whose `(owner, sessionID, deletionCredentialGeneration)` exactly matches an already accepted operation, while the owner lifecycle is `deleting` and the operation worker state is `active` or `retrying`. The logout transaction updates only that verified parent to the new owner auth epoch, so it remains usable solely through the narrow deletion capability without creating an epoch bypass. It cannot authenticate an HTTP request, mint a handoff child, or resume through the ordinary PDS factory. An intent, pending confirmation, canceled/expired operation, mismatched generation, `reauth_required` job, or terminal owner receives no exemption. Logout still invalidates all related auth requests/exchanges/receipts, and completion or terminal failure queues the exempt parent for revocation.

Cancellation or expiry of an unaccepted deletion intent clears its proof/binding and transitions any bound `deletion_only` parent to `revocation_pending` under the owner and parent locks. An accepted worker refreshes with expected `row_version`; terminal refresh changes the job to `reauth_required` and makes the old parent unusable. A later public fresh-sign-in resolves the same deleting DID and is server-derived—not client-selected—into a replacement-deletion callback for that exact job; it creates no child or membership, binds the current owner epoch and a higher credential generation, and queues the old parent. Completion or terminal failure cleans the current deletion-only parent. No recovery credential/status route is introduced.

### Explicit lifetime semantics

Use the parent's `created_at` for the configured absolute OAuth lifetime. Enforce inactivity per child bearer using `craftsky_sessions.last_seen_at`, so activity on one device does not keep a stolen bearer on another device alive. Rename the configuration if necessary to make this semantic honest (for example, `CRAFTSKY_SESSION_INACTIVITY` rather than overloading `OAUTH_SESSION_INACTIVITY`); breaking environment changes are acceptable before launch.

Store an explicit parent `expires_at` and child `idle_expires_at`, or calculate from validated immutable-at-issuance durations. Explicit timestamps are preferred because a configuration change must not silently extend already issued credentials. Conditional activity updates advance only the child idle deadline up to the parent's absolute deadline.

An expired parent transitions to `revocation_pending` and revokes all children. An expired child revokes only that child. Both outcomes are invalid credentials (401), while any database error during the decision is a dependency failure (503).

### Database-backed serialization with stale-version defense

Do not rely solely on a process-local keyed mutex; the code already anticipates scaling and a second replica would bypass it. Wrap every authenticated PDS method in a per-parent coordinator that obtains a PostgreSQL advisory lock (or an equivalently durable lease row) for the full Indigo operation, including any transparent refresh and `SaveSession` callback. Use a deterministic collision-resistant lock key derived from `(DID, sessionID)` without logging the raw value. Ensure all web and worker PDS paths use the wrapper.

The operation records the loaded `row_version`. If it receives `invalid_grant`, it re-reads the parent while still serialized:

- If the version advanced and the row remains active, rebuild from the new data and retry the operation once.
- If the version is unchanged, classify the refresh as terminal, transition the parent to `revocation_pending`, and revoke its children.
- If state changed to revocation pending/expired, do not retry.

The lock has a bounded acquisition context. If advisory locks are used, hold them on a dedicated acquired `pgx.Conn` and always release/close on cancellation. A fixed shard-lock array may reduce same-process duplicate work but cannot replace the database lock. Coordinate with the account-lifecycle plan's global order: owner-effect fence, object/work fence when present, this parent-session refresh lock, then a short transaction. The refresh coordinator must never call back into code that acquires an owner/object fence.

### Version-aware session persistence

Indigo's persistence callback currently cannot express an expected version. Add an AppView-owned adapter around the session operation so refreshed `ClientSessionData` is saved with `UPDATE ... WHERE row_version = expected RETURNING row_version`. A zero-row result is a stale writer, not permission to overwrite. If the pinned Indigo API cannot support this safely, replace only the thin session-resume/refresh integration with an atproto-specific implementation that preserves DPoP; do not adopt a generic OAuth library.

### Revocation as a local transaction plus durable work

Create a session-lifecycle service with transaction-capable store methods:

- **Single-device logout:** revoke the presented child and enqueue installation-scoped push cleanup in one transaction. Leave its parent and sibling children active.
- **All-device logout:** under the exclusive owner fence and sorted parent locks, increment the DID auth epoch, invalidate all active/pending ordinary parents and children plus authorization requests, exchanges, and receipts, enqueue account-scoped push cleanup, and make each non-exempt credential parent claimable by the upstream-revocation worker in one transaction. Rebase only the exact accepted-job deletion parent described above to the new epoch.
- **Expiry/terminal refresh:** transition the affected parent and revoke its children with the same primitive used by all-device logout.

The HTTP response depends only on the local transaction. It returns 204 after commit even while auxiliary work is pending. If the transaction cannot commit, return a retryable server failure and do not claim logout succeeded.

Retain parent credential JSON while upstream revocation is pending, but make it unavailable to ordinary resume. The worker uses a privileged load method, attempts revocation through the hardened bounded client, then deletes the parent; the foreign key removes any retained children. Cap retries/backoff and define when local deletion proceeds despite an unreachable AS so sensitive credentials are not retained indefinitely.

### SQL throttling instead of maps

After successful authorization, update activity with a conditional statement such as `UPDATE ... SET last_seen_at=$now, idle_expires_at=$deadline WHERE token_hash=$hash AND last_seen_at <= $now-$throttle`. Updating `last_device_id` should similarly occur when the value changes or a persisted `last_device_seen_at` is older than the throttle. This works across processes and naturally disappears with row deletion. Remove `lastSeenMemory`, `deviceIDMemory`, their mutex, and related comments/tests.

## How the shared update closes each finding

### AV-009 — Concurrent token refresh can revoke a newly valid OAuth session

All authenticated PDS operations for a parent are serialized across AppView processes, and credential persistence uses an expected row version. `invalid_grant` is terminal only when a re-read proves no newer rotation landed. The losing request can no longer delete the winner's credentials.

### AV-010 — Logout leaves the bearer valid when auxiliary cleanup fails

Local child/parent revocation is the first and only synchronous security gate. Push cleanup and AS revocation are durable jobs after local invalidation. Their failures affect observability/retry state, not bearer validity or the logout response after commit.

### AV-011 — Authentication-store outages are returned as 401 and erase client sessions

The store and service return an invalid-token sentinel only for authoritative invalid/expired state. Middleware checks `errors.Is(err, auth.ErrAuthTokenInvalid)` for 401 and maps all infrastructure errors to a standard 503 envelope with a bounded `Retry-After`. Flutter continues to invalidate only on 401, so a database outage preserves the local session for retry.

### AV-020 — Configured OAuth expiry does not expire bearer-only access

The bearer lookup joins its parent and evaluates parent state/absolute expiry plus child inactivity on every authentication. A proactive sweeper uses the same transition, but correctness does not depend on the sweeper or a future PDS call.

### AV-021 — “All devices” logout leaves other parent OAuth credentials active

The all-device transition is DID-wide rather than presented-parent-wide: its owner-fenced auth-epoch increment immediately makes every pre-existing login artifact unusable, and the same transaction explicitly invalidates active/pending parents, children, authorization requests, exchanges, and receipts. The revocation worker independently processes each non-exempt credential. Only the exact childless credential for an already accepted deletion job survives locally, rebased to the new auth epoch and behind `ResumeDeletionSession`; cancellation, expiry, `reauth_required`, completion, replacement, or terminal owner state removes that exception.

### AV-035 — Session throttle maps grow for the lifetime of the process

Conditional SQL activity updates replace both maps. Memory use no longer scales with historical tokens, and activity/expiry decisions are consistent across replicas and restarts.

## Unified implementation plan

1. **Write failure-first tests.** Add concurrent rotating-refresh tests, logout cleanup-failure tests with the corrected local outcome, bearer-only absolute/inactivity expiry tests, multi-parent all-device tests including callback/redemption barriers and the accepted-deletion exception, authentication-store outage tests, and high-cardinality activity tests that prove there is no process map.
2. **Finalize the lifecycle schema contract.** Coordinate with AV-008/018/019 on `oauth_sessions.lifecycle_state`. Add owner `auth_epoch`; copy it onto authorization requests, parents, children, and handoff exchanges/receipts; add parent `row_version`, `absolute_expires_at`, deletion credential generation, and upstream-revocation claim/retry fields; add child `lifecycle_state`, `idle_expires_at`, `last_seen_at`, `last_device_id`, and `last_device_seen_at`; and create `auth_auxiliary_cleanup_jobs` for installation/account push cleanup. Add consistency constraints, uniqueness/idempotency keys, and claim/sweep indexes. Update every focused auth schema fixture.
3. **Build transaction-capable stores.** In `appview/internal/auth/store.go` and `craftsky_session.go`, add narrow `pgx.Tx` methods for authoritative lookup, conditional activity touch, parent transition, child revocation, all-DID enumeration, version-aware credential save, and cleanup claim/finalization. Return typed invalid-state versus infrastructure errors.
4. **Introduce `SessionLifecycleService`.** Put orchestration in a focused auth file/package rather than expanding handlers. It owns authenticate, single logout, DID-wide epoch logout, parent expiry, terminal refresh, deletion-only cleanup/replacement, and worker claim state. Route middleware, OAuth finalization, account deletion, and handlers call this service. It exposes the exact accepted-job exception and later fresh-sign-in replacement transition as typed operations rather than flags accepted from HTTP input; departure/terminal reasons statically select the no-exemption branch.
5. **Serialize authenticated PDS operations.** Add a session-operation coordinator in `appview/internal/auth/`; after its caller has acquired any required owner/object fence, acquire the database lock/lease, load versioned session data through the ordinary-active or narrow deletion-only capability, construct Indigo once for that operation, run the PDS method, persist rotations with expected version, classify stale `invalid_grant`, and release. Wire ordinary paths through the owner-effect executor and give account deletion only its matching-job `ResumeDeletionSession` entrypoint; neither path can bypass refresh serialization.
6. **Make terminal classification version-aware.** Replace the unconditional `TranslatePDSError`/`expirePDSSession` deletion path in `appview/internal/app/deps.go`. A terminal result calls the lifecycle service only after the coordinator proves the version unchanged. Delete direct `OAuthStore.DeleteSession` calls from request paths.
7. **Make authentication authoritative.** Replace `CraftskySessionStore.Lookup` with one query/transaction that joins `oauth_sessions`, checks lifecycle and deadlines, and classifies child-expired versus parent-expired state. Apply conditional activity touch only after authorization. Wire `CraftskyAuthService` to translate only those authoritative invalid states to `ErrAuthTokenInvalid`.
8. **Correct middleware errors.** In `appview/internal/middleware/auth.go`, return 401 only for `ErrAuthTokenInvalid`; log a safe infrastructure category and return the v1 JSON 503 envelope otherwise. Set a conservative numeric `Retry-After`. Keep missing/malformed Authorization headers as 401 and malformed `X-Dev-DID` as 400.
9. **Reorder and fence logout.** In `appview/internal/auth/handlers_session.go`, call lifecycle `RevokeOne` or owner-fenced `RevokeAllForDID` first. For `all=true`, acquire sorted parent locks, increment auth epoch, invalidate all login artifacts, and rebase only the verified accepted-job deletion parent using the exact exception above. Remove synchronous notification cleanup and `oauth.Logout` from the handler. Return 204 only after the local transaction commits; use the standard retryable envelope for transaction failure.
10. **Add cleanup processors.** Claim `revocation_pending` parents and auxiliary push-cleanup jobs with leases/`SKIP LOCKED`. Revoke each AS session through the AV-001/017 client, delete parent rows idempotently, invoke push deactivation idempotently, and use capped retry/backoff plus age limits. Wire processors in `deps.go`/`main.go` with graceful shutdown.
11. **Add an expiry sweep.** Periodically transition expired parents/children using the same store functions used at request time. Sweep in bounded batches with indexes; a sweep failure must not weaken request-time enforcement.
12. **Remove in-memory throttles.** Simplify `CraftskySessionStore`, implement conditional SQL touches, update constructor/wiring, and delete tests/debug access tied to map keys. Do not replace the maps with another unbounded lock/cache map in the refresh coordinator.
13. **Verify the Flutter boundary.** Keep `SignOutOn401Interceptor` limited to 401 and add tests proving 503 is forwarded/retried without invalidating `AccountSessionLease`. Ensure API error mapping distinguishes temporary unavailability from unauthorized state.
14. **Update configuration and documentation.** Make the breaking names explicit: replace `OAUTH_SESSION_EXPIRY` with `OAUTH_SESSION_ABSOLUTE_LIFETIME`, replace `OAUTH_SESSION_INACTIVITY` with `CRAFTSKY_SESSION_INACTIVITY`, and replace `CRAFTSKY_SESSION_LAST_SEEN_THROTTLE` with `CRAFTSKY_SESSION_ACTIVITY_WRITE_INTERVAL`. Store absolute/idle deadlines at issuance/touch time so later configuration changes do not extend existing credentials. Document retry/backoff/retention and single/all logout semantics, and update `prod.env.example`, `dev.env`, OAuth BFF design, and operational alerts.
15. **Delete obsolete behavior.** Remove the test that expects push-cleanup failure to retain a valid token, the direct presented-parent-only all-logout path, any callback/finalizer that ignores owner auth epoch, lazy-only parent cleanup as an authorization mechanism, and comments saying lifetime-growing maps are acceptable.

## Data, schema, migration, and reconciliation plan

- Add a reversible migration for any lifecycle columns not already created by AV-008/018/019, owner `auth_epoch`, epoch columns on every login artifact, parent `row_version`/absolute deadline/deletion-credential generation/upstream-revocation fields, child lifecycle/idle/device activity fields, and `auth_auxiliary_cleanup_jobs`.
- Use check constraints for lifecycle values, positive auth epoch/version/deletion generation, deadline ordering, job kind/state, attempts, and lease-field consistency. Add indexes for owner-scoped auth requests/exchanges, parent/child epoch lookup, DID-wide active-or-pending transition, expired sweeps, and cleanup claims.
- Preserve the existing parent-child foreign key and `ON DELETE CASCADE`. Do not delete parent rows synchronously before upstream revocation state can be processed.
- Pre-production migration policy: invalidate and remove all existing local OAuth/CraftSky sessions, then require fresh sign-in. Do not guess expiry timestamps or lifecycle for opaque development credentials. The implementation migration may remain structurally non-destructive while the documented local reset performs the purge.
- If both grouped auth plans land together, use one migration series and one lifecycle enum/check. If they land separately, the later migration must extend rather than replace the earlier constraints.
- No public record replay/reindex is required. Session reset affects only AppView-private credentials and child bearers.
- Validate up/down/up and representative `EXPLAIN` plans for token lookup, DID-wide epoch logout across every artifact table, expiry sweep, and cleanup claim.

## API, client, configuration, and operations impact

- `POST /v1/auth/logout` retains its route. `all=true` becomes a strict DID-auth-epoch contract covering active/pending parents, children, auth requests, exchanges, and receipts, with only the accepted-job deletion credential exception. After local commit, auxiliary cleanup status does not change the 204 response.
- Authentication infrastructure failures return a standard 503 error (for example `authentication_unavailable`) plus `Retry-After`; invalid credentials remain 401. Update the API architecture/error catalog and client mappings.
- Define positive bounded values for absolute lifetime, per-child inactivity, activity write throttle, revocation worker lease, provider timeout, retry/backoff, and maximum credential-retention age. AV-030 validates relationships such as provider timeout being shorter than the worker lease.
- Alert on refresh-lock wait/timeout, stale version conflicts, parent transitions by reason, oldest revocation-pending age, exhausted AS/push cleanup, expiry sweep failures, and auth 503 rate. Never label metrics with DID/session/token.
- Document that all-device logout guarantees immediate local invalidation and eventual best-effort AS revocation. Operators need a bounded cleanup/backlog runbook, not a synchronous upstream dependency.

## Security, failure, and race considerations

- Never hold a database transaction open across an outbound AS/PDS request. A dedicated advisory-lock connection may span the operation, but lifecycle/credential updates are short transactions. The cross-plan advisory order is: every owner-effect fence in canonical DID order; optional object/work fence; every parent-session lock in canonical `(DID, sessionID)` order; then begin the transaction. The exact database row-lock order inside it is: owner lifecycle/auth-epoch row; authorization-request rows by primary key; account-deletion operation if relevant; OAuth parent rows by session ID; CraftSky child rows by primary key; handoff exchange/receipt rows by primary key; cleanup/outbox rows by primary key. A path locks only the subsets it needs, but never a later class before an earlier one. Discovery queries may find IDs without locking and must revalidate after the ordered locks.
- Use context-bounded lock acquisition and operations. Cancellation must release advisory locks/leases and connections.
- Hash lock identities with domain separation. Lock-key collision handling must remain safe; a collision may serialize unrelated sessions but must never merge state.
- A process crash while holding a PostgreSQL advisory lock releases it with the connection. A lease implementation must have owner tokens, expiries, and compare-and-set finalization.
- Refresh persistence must not silently ignore a failed `SaveSession`; the caller must know whether the new rotating refresh token became durable before reporting the PDS operation as successful. Initial callback persistence separately uses AV-019's durable `exchange_started`/ambiguous-attempt protocol because a token response can exist before the first local parent row.
- If the upstream accepted a write but credential persistence failed, surface an indeterminate classified result and reconcile the public record through Tap; do not blindly retry a non-idempotent operation.
- Logout, revocation, callback, redemption, confirmation, expiry, and deletion-credential replacement must use the advisory and database row order above. Foreign-key cascades do not authorize skipping explicit parent-before-child/receipt locks in race-sensitive transitions.
- Cleanup jobs are idempotent. Duplicate AS revocation or push deactivation must be harmless, and stale workers must be fenced from deleting reactivated/newer rows.
- Return the same 401 envelope for unknown/revoked/expired tokens; do not expose session existence or reason. Store/log the safe internal reason separately.
- A DB outage returns 503 and must not run the dev fallback unless the real result is specifically `ErrAuthTokenInvalid`.
- Conditional activity writes occur only after all authorization checks. An attacker cannot keep an expired token alive by presenting it.

## Unified test plan

### Unit tests

- Error taxonomy: missing/revoked/expired map to invalid; pool/query/cancellation errors remain infrastructure failures.
- Middleware response matrix for missing header, malformed header, invalid token, expired token, and store outage; assert JSON envelope and `Retry-After` where applicable.
- Version comparison and `invalid_grant` decision table, including advanced version, unchanged version, revoked state, and retry limit.
- Auth-epoch decision table for start, callback save/finalization, redemption, confirmation, ordinary authentication, logout-all, exact accepted-job `deletion_only` rebase, and later fresh-sign-in replacement.
- Advisory-lock and database row-lock planners return canonical owner/parent/row order and reject or make inverse acquisition unrepresentable.
- SQL activity deadline calculations cap at parent expiry and update only after throttle.
- Cleanup retry/backoff/terminal-retention policy and safe logging/redaction.

### Database and integration tests

- Two real concurrent requests encounter an expired access token against a fake rotating AS: only one refresh token is submitted at a time, the rotation persists, both operations resolve safely, and the parent/children remain active.
- Run the same test through two independently constructed services/pools to prove cross-instance coordination.
- Inject a stale `invalid_grant` after another transaction advances `row_version`; assert no revocation/deletion and one reload/retry.
- Single logout commits child revocation even when push cleanup and AS are blocked. Sibling child and parent remain active.
- All logout with several active/pending parents and children, live authorization requests, unconfirmed exchanges/receipts, and one accepted-job deletion credential increments the owner epoch and atomically invalidates every ordinary artifact before workers run. Only the exact childless bound deletion credential is rebased to the new epoch and remains usable through its narrow capability; canceled, expired, `reauth_required`, mismatched, or unaccepted deletion credentials do not.
- Repeat the same fixture through lifecycle departure and terminal-purge reasons; neither preserves or rebases `deletion_only`, and terminal leaves no credential-bearing auth artifact.
- Cancel and expire unaccepted deletion intents and prove their bound parents are locally invalidated and queued for cleanup. For an accepted job, force terminal refresh, restart without an ordinary bearer, run later normal fresh sign-in, and prove the server derives the replacement purpose from the same deleting owner/job, creates no child or membership, binds a higher credential generation/current epoch, and fences the old parent and worker.
- Barrier callback start, initial save, finalization, redemption, and confirmation independently against `all=true`. If auth work linearizes first, logout waits and invalidates it; if logout commits first, the epoch mismatch prevents persistence/activation. No old-epoch parent, child, code, or receipt becomes usable.
- Parent absolute expiry on a bearer-only read returns invalid and queues revocation. Child inactivity expires only that child. Boundary timestamps use database/server time deterministically.
- Pool/query/commit failures return infrastructure errors and never mutate Flutter state in the end-to-end client test.
- Conditional last-seen/device updates work across two store instances, respect throttle, record a changed device, and create no process-memory entry.
- Expiry/revocation workers recover expired leases, fence stale claims, survive restart, and process bounded batches.
- Run concurrency/deadlock scenarios under `go test -race` with short PostgreSQL `lock_timeout` and barriers around logout, callback, refresh, redemption, confirmation, expiry, deletion credential replace/cancel, and lifecycle departure. Exercise reverse parent/input ordering and assert no deadlock, timeout leak, or forbidden terminal state.

### Flutter tests

- `SignOutOn401Interceptor` invalidates exactly the captured account lease on 401.
- 503, timeout, cancellation, 429, and other 5xx errors never invalidate a session.
- A later 401 for a stale account lease cannot sign out the newly active account.
- Logout UI treats a committed 204 as locally complete without waiting for provider cleanup.

### Fault and operational tests

- AS token/revocation timeout, malformed response, DNS failure, PDS 401, database disconnect, process crash during refresh, worker crash after provider success, and duplicate job delivery.
- Verify no credential/lock key appears in logs, metrics, traces, or formatted structs.
- Load-test high-cardinality authenticated tokens and verify stable heap use after removing maps; separately observe bounded database write rate.

### Regression commands

- Run PostgreSQL-backed `internal/auth`, `internal/middleware`, `internal/routes`, affected PDS writer/worker packages, and Flutter auth/provider tests.
- Run the full database-backed race suite plus formatting, vet, staticcheck, and vulnerability gates from AV-033/036.

## Per-ID traceability and acceptance criteria

### AV-009

- [ ] Authenticated PDS operations serialize refresh per parent across two service instances.
- [ ] Credential persistence increments/checks `row_version`; stale writers cannot overwrite new refresh tokens.
- [ ] `invalid_grant` causes terminal revocation only after an unchanged-version re-read.
- [ ] A rotating-refresh concurrency integration test passes under the race detector.

### AV-010

- [ ] Single/all logout invalidates local bearer access before auxiliary calls can begin.
- [ ] Push/AS failure after local commit does not restore or retain usable affected bearers and does not turn the committed logout into an HTTP failure.
- [ ] Auxiliary cleanup is durable, idempotent, bounded, and observable.
- [ ] Transaction failure is reported as retryable and never falsely returns 204.

### AV-011

- [ ] Only `ErrAuthTokenInvalid` produces 401 from authenticated middleware.
- [ ] Database/query/cancellation failures produce a standard retryable 503 and safe logs.
- [ ] Flutter tests prove 503 does not invalidate local account/session state.
- [ ] Dev fallback occurs only after authoritative invalid-token state, never after store failure.

### AV-020

- [ ] Every bearer lookup enforces active parent state, parent absolute expiry, and per-child inactivity without waiting for a PDS call or sweeper.
- [ ] Expiry boundary tests cover exact timestamp behavior and server/database clock use.
- [ ] Parent expiry revokes all children and queues bounded upstream cleanup; child idle expiry affects only that child.
- [ ] Configuration docs state immutable issuance and inactivity semantics unambiguously.

### AV-021

- [ ] `all=true` increments auth epoch and locally disables every non-exempt parent, child, authorization request, exchange, and receipt for the DID in one transaction.
- [ ] The owner-fenced auth-epoch increment also invalidates pending parents/children, login and unaccepted-deletion auth requests, handoff exchanges, and receipts; callbacks/finalizers require a matching epoch.
- [ ] Every non-exempt parent is independently claimed for AS revocation and locally deleted. The sole exception is the exact childless `deletion_only` credential bound to an accepted deleting/active-or-retrying job, and that parent alone is rebased to the new epoch.
- [ ] Multi-parent and callback-race tests prove no non-presented or old-epoch ordinary credential becomes active after the response.
- [ ] A failed/unavailable AS cannot delay immediate local invalidation.
- [ ] Canonical advisory/row-lock barrier tests cover logout, callback, refresh, confirmation, expiry, and deletion credential replacement without deadlock.
- [ ] Unaccepted deletion cancel/expiry and accepted completion/terminal failure queue credential cleanup; `reauth_required` replacement is reachable only through later fresh sign-in for the same deleting DID/job and never restores a bearer or membership.

### AV-035

- [ ] `CraftskySessionStore` contains no lifetime-growing token/session throttle maps or equivalent unbounded cache.
- [ ] Conditional SQL updates preserve the configured write throttle across replicas and restarts.
- [ ] Revoked/deleted sessions leave no retained process-memory key.
- [ ] A high-cardinality test demonstrates bounded heap behavior and expected database write volume.

## Dependencies and coordination

- **AV-008/018/019 grouped OAuth-handoff update:** defines pending/active lifecycle and transaction-capable issuance. Both plans must share one schema/state machine.
- **AV-001/017:** provides the hardened, bounded client for refresh, PDS calls, and revocation. The worker must not use `http.DefaultClient`.
- **AV-002/003/006/007 grouped account-lifecycle update:** membership departure and deletion must call `RevokeAllForDID` and share parent/effect locks rather than write session tables directly.
- **AV-015:** supplies trusted-IP/global admission and may supply a shared persistent limiter; cleanup/auth bursts must be rate/capacity bounded.
- **AV-030 grouped configuration hardening:** validates all session, lock, lease, provider, retry, and expiry durations and their relationships.
- **AV-025:** use the same lease-fencing conventions for cleanup work and ensure push cleanup does not race provider delivery incorrectly.
- **AV-033/036:** required PostgreSQL, race, static-analysis, formatting, and vulnerability gates are completion prerequisites.

## References

- [AppView OAuth BFF design](../superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
- [Flutter authentication design](../superpowers/specs/2026-04-21-flutter-auth-design.md)
- [AppView API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [Settings and permanent-deletion requirements](../changes/2026-08-10-settings-page/01-requirements.md)
- [AT Protocol architecture reference](../../atproto-craft-social-app-reference.md#authentication)
- [AT Protocol OAuth specification](https://atproto.com/specs/oauth)
- [Go `errors.Is`](https://pkg.go.dev/errors#Is)
- [PostgreSQL advisory locks](https://www.postgresql.org/docs/16/explicit-locking.html#ADVISORY-LOCKS)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
