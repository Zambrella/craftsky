# Coding Plan: Multi-Account Sessions And Notification Routing

## 1. Inputs

- Requirements: `01-requirements.md` — Approved, High risk, 2026-07-18
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes, High risk, 2026-07-18
- Repository guidance: `AGENTS.md`
- Architecture reference: `atproto-craft-social-app-reference.md`
- Existing implementation inspected:
  - Flutter single-session secure storage, auth projection/controller, OAuth handoff, session and handoff Dio clients, global `401` interceptor, router redirects, onboarding state, shell navigation, account-sensitive repositories/providers, compose/edit discard flows, and provider logging
  - Flutter notification registration/runtime/open/effect flow, secure routing bindings, notification list/preferences/seen/new-count providers, foreground banner, and Firebase lifecycle boundary
  - AppView current-session logout and notification-subscription cleanup handlers/tests, including fail-closed cleanup behavior
- Approved clarification, 2026-07-18: the existing OAuth callback handoff credential in `craftsky:///auth/complete?token=...` is a narrow exception to the `NFR-002` URL prohibition. It remains forbidden from logs, analytics, crash context, UI text, and string representations. The exception does not apply to retained session-registry tokens, cleanup credentials, routing IDs, DIDs, handles, or retained-account data, and it does not authorize a new AppView route.
- Approval gate: coding planning is approved; source implementation remains blocked until the user explicitly approves or invokes `implement-tdd` because this is a High-risk change.

## 2. Implementation Strategy

Implement a client-first account boundary while retaining the existing AppView API:

1. Replace the one-key `StoredSession` blob with a versioned `SessionRegistry` persisted as a two-slot secure journal. The registry owns retained sessions keyed by DID, one active DID, monotonic session and recent-use generations, cached switcher identity, secure notification-routing bindings, and non-activatable pending cleanup credentials. A full verified snapshot is the commit unit.
2. Make `sessionRegistryProvider` the only mutable authentication source. Keep `authSessionProvider` as a token-free derived projection for router and UI compatibility. UI-facing account values use a redacted `AccountKey`; asynchronous authority is always a DID plus immutable session generation, and active UI work additionally captures an activation generation.
3. Split anonymous and authenticated networking. Each authenticated Dio is constructed from one captured account session and never consults a mutable global token during a request. Its `401` interceptor invalidates only the captured DID/session generation and coalesces cleanup per account generation. The existing active repository providers rebuild from the active account client; inactive validation, push registration, and count refresh use account-bound clients directly.
4. Centralize switching in `AccountActivationCoordinator`: consult the shared unsaved-work guard, expose a blocking transition overlay, atomically activate the retained account, invalidate/dispose account-sensitive state, reset navigation to Home, and reject completions carrying the old activation lease. Notification activation uses the same coordinator before destination inference/navigation.
5. Generalize notification registration and new-count work to every eligible retained account. Reverse-resolve `accountSubscriptionId` through the secure registry, intersect the result with the current retained-session generation, and never treat routing data as authorization. Foreground pushes resolve the recipient for banner identity and authoritative recipient-count refresh without activating until tapped.
6. Make sign-out account-scoped. Confirmed AppView logout removes only the active account. Unconfirmed logout atomically removes the usable session/binding and quarantines its token, invalidates the shared provider token, retries AppView deactivation, deletes the cleanup credential after success or authoritative `401`, and only then registers remaining accounts to a replacement provider token.
7. Add a shared switcher model and responsive presentation: compact long-press opens a modal bottom sheet; large layout opens an anchored menu. The Profile destination renders the active avatar or generic person fallback. Add account reuses OAuth through a signed-in root route and upserts the returned DID without disturbing existing accounts.
8. Add the mandatory AppView shared-installation logout contract test. No production Go route, migration, lexicon, PDS-token handling, dependency, or database change is planned.

The first red test is `UT-001` in `app/test/auth/models/session_registry_test.dart`. It fixes the registry model and two-slot recovery contract before providers, networking, notification routing, or UI depend on it.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Secure auth state | One `craftsky_session` JSON blob and one active session | Versioned DID-keyed registry in a verified two-slot secure journal; tolerant entry decoding, deterministic MRU fallback, secure routing bindings, and cleanup quarantine | BR-001, FR-001, FR-003, FR-016, FR-018, FR-026, RULE-001, RULE-004, NFR-002 | AT-001, AT-002, AT-008, AT-011, UT-001–UT-003, UT-012–UT-014, IT-001, IT-002, IT-007, IT-010 |
| Auth projection and startup | `AuthSession` owns storage and validates one global client | Registry notifier restores optimistically; token-free auth projection; active-first, bounded inactive validation scoped to session generations | FR-017, FR-024 | AT-010, UT-010, IT-009 |
| HTTP authorization | One keep-alive Dio reads the latest global token in `onRequest`; one global `401` sign-out | Anonymous Dio plus fixed-account Dio family; captured credential/lease; per-account `401` invalidation and coalescing | FR-008, FR-017, NFR-001 | AT-004, AT-008, UT-004, UT-005, IT-003, IT-008, REG-003 |
| Account boundary/state | Route and providers implicitly follow global auth | One activation coordinator, transition overlay, explicit account-state invalidator, account-operation leases, Home reset, and stale completion rejection | FR-007–FR-009, FR-019, FR-023, NFR-001, NFR-003 | AT-004, AT-009, UT-004, UT-015, UT-021, IT-003, IT-012, REG-004, REG-008 |
| OAuth/add account | Signed-in routes redirect away from Sign in; callback replaces the one stored session | Signed-in Add account route reuses sign-in UI; successful callback atomically upserts/activates; failure/cancel leaves the snapshot and route untouched; enforce five-DID limit | FR-003, RULE-001, RULE-004 | AT-002, UT-002, UT-020, UT-022, IT-002, REG-001 |
| Switcher/Profile identity | Static Profile person icon and no account switcher | Shared active-first/MRU switcher model, cached identity, compact sheet, large anchored menu, count badges, accessible actions, and active avatar/fallback | FR-004–FR-006, FR-020, FR-022, RULE-003–RULE-005, NFR-004 | AT-003, AT-006, UT-009, UT-016, UT-018, UT-019, IT-011, REG-002, REG-009 |
| Notification registration | One active DID/readiness and one active repository | Register every retained eligible account with fixed clients; retry independently; fence late saves; token refresh drains all accounts | FR-010, FR-011, RULE-002 | AT-007, UT-011, IT-004, REG-007 |
| Notification routing/open | Compare payload binding only with current DID; navigation under current account | Reverse exact binding lookup; classify malformed/ambiguous/removed; activate recipient before inference/navigation; carry latest open and DID/session generation | BR-002, FR-012–FR-015, FR-025 | AT-005, UT-006–UT-008, IT-005, REG-006 |
| Notification state and banner | Global keep-alive list/preferences/count/seen; foreground banner has actor copy only | DID-keyed repositories/providers; row owner lease; recipient-scoped authoritative count refresh; inactive-recipient cached identity line | FR-009, FR-020, FR-021, RULE-002 | AT-006, AT-012, UT-008, UT-009, UT-017, IT-006 |
| Sign-out/recovery | Logout and global `401` clear all secure auth; offline cleanup best-effort deletes provider token | Remove only selected account; atomic quarantine; ordered shared-token recovery; MRU fallback or signed-out only when empty | BR-003, FR-016–FR-018, FR-026, NFR-002 | AT-008, AT-011, UT-005, UT-012, UT-013, IT-007, IT-008, IT-010, IT-013 |
| Privacy/diagnostics | Auth values include DID/handle in strings; GoRouter diagnostics enabled; handoff token is a provider-family argument | Fully redacted account/auth/registry strings, redacted provider keys, disabled route diagnostics, bounded outcome-only logging, secret sentinel coverage, and narrow OAuth URL exception | NFR-002 | UT-001, UT-014, REG-010, MAN-003 |
| AppView contract | Existing logout deactivates authenticated installation subscription before revoking session | Extend handler tests for two accounts sharing one installation and retain fail-closed ordering; no production Go change expected | BR-003, FR-016, FR-026 | IT-013 |

## 4. Files And Modules

Generated mapper, Riverpod, GoRouter, and localization outputs are regenerated from source and are not hand-edited.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/auth/models/session_registry.dart` | Create | Define the versioned registry, DID-keyed sessions, active/MRU rules, global next-generation counters, cached identity, routing bindings, pending cleanup records, five-account rule, tolerant decoder, and fully redacted diagnostics | FR-001, FR-003, FR-005, FR-016, FR-018, FR-026, RULE-001, RULE-004, NFR-002 | UT-001–UT-003, UT-012–UT-014, UT-016 |
| `app/lib/auth/models/account_key.dart`, `account_session_lease.dart` | Create | Provide redacted family keys and immutable DID/session/activation ownership tokens without exposing credentials in provider names or strings | FR-007–FR-009, FR-017, NFR-001, NFR-002 | UT-004, UT-005, UT-010, UT-014 |
| `app/lib/auth/models/account_switcher_state.dart` | Create | Build active-first rows, identity fallbacks, count presentation inputs, Add enabled/full state, and allowed actions | FR-004, FR-005, FR-020, RULE-003–RULE-005 | UT-009, UT-016, UT-019 |
| `app/lib/auth/models/stored_session.dart`, `auth_state.dart` | Change | Add session generation/recent-use/cached identity to stored entries; remove token from `SignedIn`; fully redact identity and credentials from `toString` | FR-001, FR-005, FR-008, NFR-002 | UT-001, UT-003, UT-014, REG-003 |
| `app/lib/auth/providers/secure_token_storage.dart` | Change/Rename | Replace single-key storage with `SecureSessionRegistryStorage`, two journal slots, read-back verification, bounded outcome logging, and fakeable slot backend | FR-001, FR-026, NFR-002 | UT-001, IT-001, IT-010 |
| `app/lib/auth/providers/session_registry_provider.dart` | Create | Sole mutable source; serialize journal mutations; expose safe summaries/leases; upsert, activate, update identity/binding, remove, quarantine, and terminal-cleanup methods | FR-001, FR-003, FR-007, FR-016–FR-018, FR-026 | UT-002–UT-005, UT-010, UT-012, UT-013, IT-001–IT-003, IT-007–IT-010 |
| `app/lib/auth/providers/auth_session_provider.dart` | Change | Become a token-free derived active-session projection and launch active-first background validation after registry restore | FR-017, FR-024 | UT-010, IT-001, IT-009 |
| `app/lib/auth/services/session_validation_coordinator.dart` | Create | Validate active account first, then inactive accounts with concurrency two; retain transient failures; apply authoritative results only to unchanged leases | FR-017, FR-024 | UT-010, IT-009 |
| `app/lib/auth/providers/account_activation_coordinator.dart` | Create | Serialize switch requests, consult discard guard, expose transition, activate atomically, invalidate account state, reset Home, and fence stale work | FR-007, FR-009, FR-019, FR-023, NFR-001, NFR-003 | UT-004, UT-015, UT-021, IT-003, IT-012 |
| `app/lib/auth/providers/unsaved_work_guard_provider.dart` | Create | Register the current dirty compose/edit owner and reuse its existing confirmation for manual and notification activation | FR-023 | UT-015, IT-012, REG-008 |
| `app/lib/auth/providers/auth_controller.dart`, `pending_auth_provider.dart` | Change | Use anonymous login and one-shot handoff client; atomically upsert only after successful `whoami`; make Sign out account-scoped; preserve registry on all Add failures | FR-003, FR-016, FR-018, FR-026, RULE-004 | UT-002, UT-012, UT-020, UT-022, IT-002, IT-007, IT-010 |
| `app/lib/auth/providers/auth_api_client_provider.dart`, `handoff_api_client_provider.dart` | Change | Split anonymous/account auth clients; wrap the exempt callback credential in a redacted argument or construct its one-shot client directly | FR-003, FR-008, NFR-002 | UT-005, UT-014, UT-020, UT-022, REG-003 |
| `app/lib/shared/api/providers/dio_provider.dart` | Change | Provide anonymous Dio, fixed `accountDioProvider(AccountKey)`, and active alias; capture token/generation at construction; close clients on disposal | FR-008, FR-017, NFR-001 | UT-005, IT-003, IT-008, REG-003 |
| `app/lib/shared/api/providers/session_auth_interceptor.dart` | Change | Attach one captured bearer plus device ID; never read mutable global auth; keep login anonymous | FR-008 | UT-005, REG-003 |
| `app/lib/shared/api/providers/sign_out_on_401_interceptor.dart` | Change | Carry account/session lease and invalidate only that unchanged lease; coalesce by lease rather than globally | FR-017, FR-018, FR-026 | UT-005, IT-008 |
| Active feature repository providers under `feed/`, `profile/`, `projects/`, `search/`, and `shared/media/` | Change | Rebuild from active fixed Dio; capture repository and account lease at mutation start | FR-008, FR-009, NFR-001 | UT-004, IT-003, REG-004 |
| Authenticated async providers under `feed/providers/`, `profile/providers/`, `projects/providers/`, and `search/providers/` | Change | Reject late data/error/optimistic rollback unless the captured lease is current; participate in boundary invalidation | FR-009, NFR-001 | UT-004, IT-003, REG-004 |
| `app/lib/auth/services/account_state_invalidator.dart` | Create | Centralize invalidation of feed/post/comment/profile/project/search/recent-search/notification/seen/preferences/count/mutation/composer state on activation/removal | FR-009, NFR-001 | UT-004, IT-003, REG-004, REG-005 |
| `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/projects/widgets/project_composer_sheet.dart`, `app/lib/profile/pages/edit_profile_dialog.dart` | Change | Register dirty ownership/confirmation and originating account lease; unregister on disposal; preserve existing PopScope behavior | FR-023 | UT-015, IT-012, REG-008 |
| `app/lib/auth/widgets/account_switcher.dart`, `account_switcher_sheet.dart`, `account_switcher_menu.dart`, `account_transition_overlay.dart` | Create | Shared content plus compact/large surfaces, account rows, count badges, Add/full state, semantics, and non-interactive transition | FR-004, FR-005, FR-007, FR-020, NFR-003 | UT-016, IT-011, MAN-001 |
| `app/lib/auth/widgets/account_avatar.dart` | Create | Render cached avatar with a generic person fallback and selected treatment appropriate to navigation/switcher/banner | FR-005, FR-021, FR-022 | UT-017, UT-018, IT-011 |
| `app/lib/router/app_shell.dart` | Change | Use active identity in Profile destination; normal tap unchanged; long-press/semantics action opens sheet; large action anchors menu; show account counts | FR-004, FR-006, FR-020, FR-022, NFR-004 | UT-009, UT-018, UT-019, IT-011, REG-002 |
| `app/lib/router/router.dart`, `route_locations.dart` | Change | Add authenticated `/add-account`, allow its OAuth flow, route successful new active account by onboarding, reset Home on activation, and disable raw route diagnostics | FR-003, FR-019, NFR-002 | UT-020, UT-021, IT-002, IT-003, REG-001 |
| `app/lib/auth/pages/sign_in_page.dart`, `auth_complete_page.dart` | Change | Reuse sign-in UI in Add mode, surface maximum/storage/handoff failures safely, and keep exempt callback token invisible/redacted | FR-003, RULE-004, NFR-002 | UT-014, UT-020, UT-022, REG-001 |
| `app/lib/profile/pages/profile_page.dart` or `app/lib/auth/providers/account_identity_provider.dart` | Change/Create | Refresh the active account's own profile and persist display name/avatar only if its account lease remains current | FR-005, FR-022 | UT-003, UT-018, IT-011 |
| `app/lib/notifications/services/notification_routing_storage.dart` | Change | Become a secure-registry adapter; write bindings against unchanged leases and reverse-resolve exact/ambiguous/unbound outcomes | FR-010, FR-012, FR-014, FR-025, NFR-002 | UT-006, UT-014, IT-004, IT-005 |
| `app/lib/notifications/services/notification_registration_coordinator.dart` | Change | Accept a retained-account readiness set; register with fixed account clients; queue revisions; isolate failures; reject late removed-account bindings | FR-010, FR-011 | UT-011, IT-004, REG-007 |
| `app/lib/notifications/services/notification_open_coordinator.dart`, `pending_notification_open.dart` | Change | Resolve recipient before inference, activate via shared coordinator, scope latest pending work to session generation, and distinguish invalid from removed | FR-012–FR-015, FR-025 | UT-006, UT-007, IT-005, REG-006 |
| `app/lib/notifications/services/notification_runtime.dart`, `providers/notification_runtime_provider.dart` | Change | Observe all retained readiness, route opens by resolved account, resolve foreground recipient identity, and refresh recipient-scoped list/count | FR-010–FR-015, FR-020, FR-021 | UT-007, UT-011, UT-017, IT-004, IT-005 |
| `app/lib/notifications/models/foreground_notification_event.dart`, `notification_effect.dart` | Change | Carry a redacted recipient lease/identity view for in-app banners without modifying OS-visible provider copy | FR-021, NFR-002 | UT-014, UT-017, IT-005, REG-007 |
| `app/lib/notifications/widgets/notification_effect_host.dart` | Change | Render `For @handle` and avatar for inactive recipients; re-resolve/guard on tap; show exact removed-account message | FR-013, FR-021, FR-025 | UT-017, IT-005 |
| Notification repository/providers (`notification_repository_provider.dart`, `notifications_provider.dart`, `notification_preferences_provider.dart`, `notification_seen_provider.dart`, `notification_new_count_provider.dart`) | Change | Key account-sensitive clients/state by redacted `AccountKey`; retain installation-wide permission/service; add active aliases where useful | FR-009, FR-020, RULE-002 | UT-008, UT-009, IT-006, REG-005 |
| `app/lib/notifications/models/notifications_state.dart` and row/open flow | Change | Attach producing account/session generation to list state/row actions and reject stale row taps | FR-012 | UT-008, IT-005 |
| `app/lib/notifications/services/notification_sign_out_cleanup.dart` | Replace | Split confirmed cleanup from persisted `NotificationSignOutRecovery`; enforce provider-token invalidation, terminal logout, credential deletion, then re-registration | FR-016, FR-026, NFR-002 | UT-012, UT-013, IT-010 |
| `app/lib/notifications/providers/notification_lifecycle_provider.dart` | Change | Wire routing adapter and keep-alive recovery coordinator; retry at startup/resume/connectivity-relevant triggers | FR-010, FR-011, FR-016, FR-026 | UT-011–UT-013, IT-004, IT-010 |
| `app/lib/app.dart`, `bootstrap.dart` | Change | Mount transition host/overlay, initialize new mappers, probe anonymous networking, and prevent provider/route diagnostics from stringifying account data | NFR-001, NFR-002, NFR-003 | UT-004, UT-014, REG-010 |
| `app/lib/l10n/app_en.arb` and generated localization | Change/Generate | Add switcher, limit, transition, Add-account, recipient, removed-account, and safe failure/accessibility strings | FR-004, FR-020, FR-021, FR-025, RULE-004 | AT-003, AT-005, AT-012, IT-011 |
| Flutter tests named in `02-acceptance-tests.md` | Create/Change | Implement `UT-001`–`UT-022`, `IT-001`–`IT-012`, and regression coverage with fake journal slots, account clients, controlled completers, operation recorder, and responsive harnesses | All client requirements | UT-001–UT-022, IT-001–IT-012, REG-001–REG-010 |
| `appview/internal/auth/handlers_test.go` | Change | Seed A/B sessions and subscriptions on one device; prove A logout changes only A; retain cleanup-failure/no-revocation assertion | BR-003, FR-016, FR-026 | IT-013 |

## 5. Services, Interfaces, And Data Flow

### Secure registry and journal protocol

The persisted snapshot is complete and self-contained. Routing bindings and cleanup credentials live in the same secure commit as usable sessions so removal/quarantine cannot expose a half-active account.

```text
// Partial shapes only.
final class SessionRegistry {
  final int schemaVersion;          // v1
  final int revision;               // journal winner
  final int nextSessionGeneration;  // never reused after remove/re-add
  final int nextUseOrdinal;         // stable MRU ordering
  final int activationGeneration;   // active UI boundary
  final Did? activeDid;
  final Map<Did, StoredSession> sessions;
  final Map<Did, String> routingBindings;        // private/redacted
  final Map<String, PendingSessionCleanup> cleanupQueue;

  SessionRegistry upsertAndActivate(...);
  SessionRegistry activate(AccountSessionLease target);
  SessionRegistry removeConfirmed(AccountSessionLease target);
  SessionRegistry quarantineForCleanup(AccountSessionLease target);
}

final class StoredSession {
  final String token;
  final Did did;
  final Handle handle;
  final int sessionGeneration;
  final int lastUsedOrdinal;
  final String? cachedDisplayName;
  final String? cachedAvatarUrl;
}
```

Secure keys are `craftsky_session_registry_a` and `craftsky_session_registry_b`. There is no legacy single-session migration because the approved scope states there are no signed-in installations to migrate.

Read protocol:

1. Read both slots independently; a platform read failure is a bounded storage outcome and never logs the exception value or stored data.
2. A slot is top-level valid only when JSON, schema version, revision, counters, and collection containers are valid. Unknown versions are invalid slots.
3. Choose the top-level-valid slot with the highest revision; ties resolve to slot A. Never choose an older revision merely because one entry in the newest slot is corrupt, because doing so could resurrect an intentionally removed account.
4. Decode session, binding, and cleanup entries independently from the winning slot. Drop malformed entries. Require a session map key to match its parsed DID; discard bindings without a decodable DID key; never make cleanup entries activatable.
5. If `activeDid` is absent from valid sessions, choose the highest `lastUsedOrdinal`, with a stable DID-order tie-break. If no valid sessions remain, project `SignedOut` while retaining independently valid cleanup work.
6. If neither slot is top-level valid, recover as signed out and report only a bounded `allSlotsInvalid` classification.

Write protocol:

1. Serialize mutations through `sessionRegistryProvider`; concurrent storage writers are forbidden.
2. Derive a full next snapshot at `revision + 1` without changing provider state.
3. Write it to the non-winning/older slot, leaving the current winner untouched.
4. Read that slot back, decode it through the production decoder, and require the expected revision plus canonical snapshot equality.
5. Only after verification publish the new in-memory state. A throw, partial write, corrupt read-back, or process interruption before verification leaves the prior winning slot and provider state authoritative.
6. The following successful mutation alternates slots. No separate active-pointer key exists, so there is no pointer commit window.

`TD-002` covers: interruption before write, platform throw, truncated target slot, target read-back mismatch, valid newest slot with one corrupt entry, corrupt active DID, unknown newest version with valid older slot, and both slots invalid.

### Account ownership and fixed clients

```text
final class AccountKey {
  final Did did; // available to trusted code; toString is fully redacted
}

final class AccountSessionLease {
  final AccountKey account;
  final int sessionGeneration;
}

final class ActiveAccountLease {
  final AccountSessionLease session;
  final int activationGeneration;
}

Dio buildAccountDio(StoredSession capturedSession) {
  // Captured Bearer + device header + error mapping
  // + Account401Interceptor(captured lease).
}
```

An account Dio never asks `authSessionProvider` for a token. A repository/mutation captures its client and lease before its first await. UI state applies success, error, rollback, cache prepend, or navigation only when `isCurrent(lease)` remains true. A late A request may complete against A on the server, but it cannot write B's provider state or navigate B's UI.

Anonymous login uses `anonymousDioProvider`; active feature repositories use the active account alias; inactive validation/registration/count work uses `accountDioProvider(AccountKey)`. Reauthentication of the same DID receives a never-reused session generation and disposes the prior client.

### Activation boundary

```text
Future<AccountActivationResult> activate(
  AccountSessionLease target, {
  required AccountActivationSource source,
})
```

Activation sequence:

1. Reject missing/stale target lease and coalesce duplicate requests.
2. Ask `UnsavedWorkGuard` to discard/leave the active dirty flow. Cancellation returns `canceled` and permanently consumes that activation/open attempt.
3. Publish `AccountTransition(targetIdentity)` so a modal barrier blocks authenticated interaction and identifies the selected account.
4. Commit registry activation, advancing MRU and `activationGeneration` without changing any session generation.
5. Dispose/rebuild the active Dio and repository graph; invalidate every account-sensitive state provider and optimistic mutation listed in `AccountStateInvalidator`.
6. Clear root and branch navigation and go to Home. Manual switching completes locally without waiting for network.
7. Remove the barrier only after the Home shell observes the new activation generation. Notification activation then performs destination inference/navigation as a separate next step.

Confirmed sign-out and authoritative invalidation use the same boundary from step 3 onward when the removed account was active, selecting the registry's MRU fallback. Removing an inactive account never changes active routing (inactive direct removal is not exposed in v1).

### Add account and identity refresh

`AddAccountRoute` reuses `SignInPage` in an explicit mode and is allowed only while signed in. The existing callback credential remains in the callback URL under the approved exception, but route diagnostics are disabled and every wrapper/string remains redacted. Handoff `whoami` must succeed before any registry mutation. The controller then performs one journal upsert that:

- replaces an existing DID's token/identity with a new session generation, even when five accounts are retained;
- rejects a sixth distinct DID without changing the current snapshot;
- adds a new DID below the limit;
- makes the returned DID active and advances MRU/activation generation.

OAuth cancellation, browser-launch failure, callback timeout, handoff rejection, storage verification failure, or other completion failure leaves the entire previous snapshot and route untouched. After success, router onboarding logic uses the newly active DID. The active own-profile response may later update cached display name/avatar only against the unchanged lease.

### Notification routing and registration

```text
provider payload
  -> parse redacted NotificationOpenAttempt
  -> reverseResolve(accountSubscriptionId)
       missing/malformed/ambiguous -> generic unavailable, no activation
       well-formed unbound/non-retained -> signed-out-account message
       exact retained lease -> continue
  -> AccountActivationCoordinator.activate(target) when inactive
  -> confirm target lease still current
  -> existing pure destination inference
  -> existing typed GoRouter navigation under target client
```

The reverse index returns only a redacted account lease to downstream code. A routing ID selects local context but never authorizes content; typed destination reads still use the selected account's authenticated AppView client.

For foreground delivery, runtime resolves the recipient immediately. It refreshes `accountNotificationNewCountProvider(recipient)` from the server and invalidates the recipient list if instantiated; it never increments a local counter. An inactive-recipient banner receives only cached display identity and a redacted lease. Tapping re-runs resolution/currentness and normal activation, so a removed or reauthenticated recipient cannot reuse the stale banner.

Registration receives a snapshot of all retained accounts whose DID-keyed onboarding flag is complete. OS permission remains installation-wide. On authorization/resume/token refresh, it obtains the latest provider token once and drains accounts through fixed clients. Each saved binding requires the same session generation still be retained. Failures remain retryable per account and do not block other registrations.

### Sign-out and offline recovery

Confirmed path:

```text
active fixed client POST /v1/auth/logout
  -> AppView deactivates only that DID + installation subscription
  -> AppView revokes only that CraftSky session
  -> one secure snapshot removes usable session + binding
  -> invalidate that account state
  -> MRU fallback Home, or SignedOut when empty
```

Unconfirmed path:

```text
logout network/server failure or account-scoped 401 invalidation
  -> one secure snapshot removes usable session + binding
     and adds non-activatable PendingSessionCleanup(token, lease)
  -> reject stale opens and activate MRU fallback locally
  -> invalidate/delete shared provider token; block all replacement registration
  -> retry cleanup with quarantined fixed credential
       204 success or authoritative 401 = terminal
       transient failure = retain queue and retry later
  -> secure snapshot deletes terminal cleanup credential
  -> after all pending cleanup is terminal, obtain replacement provider token
  -> register every remaining eligible account
```

Multiple pending cleanups drain before any replacement registration. Provider-token refresh events received while cleanup is pending are retained only as a retry signal, not registered. Recovery is retried on startup, app resume, and existing registration retry triggers. Logs expose only bounded stage/outcome enums and queue length is not logged.

## 6. State, Providers, Controllers, Or DI

```text
secureSessionRegistryStorageProvider
  -> sessionRegistryProvider (keepAlive AsyncNotifier; sole mutable source)
       -> authSessionProvider (token-free AsyncValue<AuthState>)
       -> activeAccountLeaseProvider
       -> retainedAccountSummariesProvider
       -> sessionValidationCoordinatorProvider
       -> accountDioProvider(AccountKey) [autoDispose family]
            -> account repository families
       -> dioProvider (active alias for existing feature repositories)
       -> accountActivationCoordinatorProvider
            -> unsavedWorkGuardProvider
            -> accountStateInvalidatorProvider
            -> accountTransitionProvider
            -> goRouterProvider

notificationServiceProvider (installation scoped)
  -> notificationEligibleAccountsProvider
  -> notificationRuntimeProvider (keepAlive)
       -> notificationRegistrationCoordinator
       -> notificationRoutingStorageProvider (registry adapter)
       -> accountNotificationRepositoryProvider(AccountKey)
       -> accountNotificationNewCountProvider(AccountKey)
       -> accountActivationCoordinatorProvider
       -> notificationEffectStreamProvider
  -> notificationSignOutRecoveryProvider (keepAlive)

active notification pages
  -> activeAccountLeaseProvider
  -> notificationsProvider(AccountKey)
  -> notificationPreferencesProvider(AccountKey)
  -> notificationSeenProvider(AccountKey)
  -> accountNotificationNewCountProvider(AccountKey)
```

Provider rules:

- Raw `Did`, `Handle`, token, routing ID, or retained account list is never a provider diagnostic argument. Families use `AccountKey`, whose `toString` is constant and redacted.
- `SessionRegistry`, `StoredSession`, `PendingSessionCleanup`, account leases, `SignedIn`, routing outcomes, and foreground effects have explicit diagnostic strings that contain only type/bounded enum information.
- `ProviderLogger` continues to log values only through those redacted strings. GoRouter `debugLogDiagnostics` is disabled because the exempt OAuth callback URL contains a credential that is still forbidden from logs.
- Active account state invalidation includes timeline/post/thread/comments/user-posts/user-comments/create-delete-report-like-repost, profile/viewer/follow/save/report, project feeds/user projects/composer work, search results/suggestions/top hashtags/recent searches, notification list/preferences/seen/count, and any future saved-data provider registered with the invalidator.
- Onboarding remains the existing DID-keyed SharedPreferences family. It is selected using the active account key and never logged.
- Installation-scoped providers remain shared: app dependencies, device ID, OS notification permission/service, theme, and public image cache. Image URLs may be reused, but viewer-specific profile objects are invalidated.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Account switcher

Both responsive presentations render one `AccountSwitcherContent`:

- active account first, then inactive accounts by descending `lastUsedOrdinal`;
- avatar, display name when cached, and handle fallback;
- current-account semantic/state treatment;
- per-account new-count badge hidden for null/zero and capped with existing `NotificationBadge` as `99+`;
- selectable inactive rows only—no direct remove/sign-out action;
- Add account always visible; disabled at five distinct accounts with `Maximum of 5 accounts`;
- no sign-out-all command, label, semantics action, or hidden menu item.

Compact Profile long-press/semantics action invokes `showModalBottomSheet`; large layout uses an anchor attached to the Profile rail destination and an accessible menu/popover. Normal Profile tap continues `goBranch` unchanged and single-account users see no extra ordinary-navigation step. Keyboard activation invokes the same account-switch action, not a separate behavior.

### Profile destination and transition

The Profile destination uses a small `AccountAvatar` only when a cached avatar URL is available; load failure/missing data uses the generic person icon required by `FR-022`. Selected state retains an explicit visual border/background and selected semantics. It does not reuse `ProfileAvatar`'s initial fallback because the approved Profile fallback is the generic person glyph.

`AccountTransitionOverlay` is mounted above router content in `MaterialApp.router.builder`. It displays the selected cached identity/fallback, blocks pointer and semantics interaction with the old account, and remains until the selected Home shell is mounted. Offline content failure appears only after the switch and uses existing retry UI.

### Add account and routing

- `/add-account` is a root-navigator route available only to a signed-in, onboarded active account.
- It reuses the existing handle form with Add-account title/copy and returns to the unchanged account on cancel/failure.
- `/auth/complete?token=...` remains the existing external callback transport under the explicit exception; no other route may carry a credential.
- Successful completion activates the returned account. Existing redirect rules send incomplete accounts to Onboarding and complete accounts to Home.
- Manual switching and sign-out fallback always clear stacks and land at Home; no per-account navigation history is stored.

### Unsaved work

Post compose, project compose, and edit profile register their dirty owner and existing localized discard confirmation. A notification or manual account change requests that same guard. Cancel leaves the current account, route, local draft, and provider state unchanged and destroys the request. Confirm closes/disposes the originating flow before the transition starts. Draft/provider identifiers include the originating `AccountKey` where state can outlive a widget.

### Notification surfaces

- OS-visible notification title/body remains unchanged and gains no recipient identity.
- A foreground notification for the active account retains current in-app copy.
- An inactive exact recipient adds a compact avatar plus `For @handle` line beneath existing title/body, using cached handle and generic avatar fallbacks.
- A malformed/missing/ambiguous routing ID uses the existing generic unavailable feedback without switching.
- A well-formed unbound or no-longer-retained recipient shows exactly `This notification belongs to an account that is no longer signed in` without switching, signing in, or navigating.
- Notification rows carry their producing lease. A stale row cannot navigate after an account transition; current rows keep existing typed destination behavior.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Both secure journal slots absent | Empty registry and `SignedOut`; cleanup queue empty | FR-001 | UT-001, IT-001 |
| Target slot write interrupted/corrupt | Keep prior verified winner and in-memory state; next mutation retries alternate slot | FR-001, NFR-002 | UT-001, IT-001 |
| Winning slot has one corrupt session | Drop only that entry; never resurrect older revision; repair active by MRU | FR-001, FR-018 | UT-001, UT-003 |
| Unknown newest schema | Use older valid supported slot; if none, signed out with bounded classification | FR-001 | UT-001 |
| Sixth distinct account | Reject atomically; keep Add visible disabled at five; existing-DID refresh still allowed | FR-003, RULE-001, RULE-004 | UT-002, UT-016, IT-011 |
| Add account canceled/failed | Clear only pending OAuth state; registry, active account, MRU, route, and providers unchanged | FR-003 | UT-022 |
| Storage verification fails after OAuth | Surface safe storage error; do not publish partial account | FR-003, NFR-002 | UT-001, UT-022 |
| Manual switch offline | Commit locally, reset Home, keep selected account; normal content retry may render | FR-019 | UT-021, IT-003 |
| Dirty work, activation canceled | Preserve account and draft; discard activation/open permanently | FR-023 | UT-015, IT-012 |
| Late A result after B activation | Fixed A request may finish, but lease check/disposal prevents B state/error/rollback/navigation | FR-008, FR-009, NFR-001 | UT-004, UT-005, IT-003 |
| `401` from inactive or formerly active A | Invalidate/quarantine only unchanged A lease; B remains; no global clear | FR-017, FR-026 | UT-005, IT-008 |
| Startup network/server failure | Retain cached session; retry opportunistically | FR-024 | UT-010, IT-009 |
| Startup unauthorized/identity mismatch | Remove only unchanged target lease; choose MRU fallback only if active | FR-017, FR-024 | UT-010, IT-009 |
| One inactive count/registration failure | Preserve other rows/accounts; retain null/cached count and retry that account | FR-010, FR-011, FR-020 | UT-009, UT-011, IT-004, IT-006 |
| Notification binding malformed/missing/ambiguous | Generic unavailable, current account unchanged, no destination | FR-014 | UT-006, IT-005 |
| Notification binding well-formed but removed | Exact signed-out-account message, no activation/navigation | FR-025 | UT-006, IT-005 |
| Recipient removed/reauthed during open | Lease mismatch consumes open; safe unavailable/removed outcome; never navigate under replacement session | FR-015, FR-025 | UT-007, IT-005 |
| Foreground duplicate delivery | Fetch recipient's authoritative count; never local increment | FR-020 | UT-009, IT-006 |
| Confirmed logout | Remove selected session/binding/state; MRU Home or signed out if last | FR-016, FR-018 | UT-012, IT-007, IT-013 |
| Offline/unconfirmed logout | Remove usable session immediately; persist cleanup-only credential; rotate provider token; block registration until terminal cleanup | FR-016, FR-026 | UT-013, IT-010 |
| Recovery logout returns `401` | Treat cleanup as terminal, delete credential, then permit replacement registration | FR-026 | UT-013, IT-010 |
| Recovery retryable failure/restart | Keep non-activatable queue in secure journal and retry before any replacement registration | FR-026, NFR-002 | UT-013, IT-010 |
| OAuth callback URL handling | Existing handoff URL is allowed only by explicit exception; disable route logs and redact every other output | NFR-002 clarification | UT-014, REG-010, MAN-003 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `app/test/auth/models/session_registry_test.dart` | TD-001/TD-002 registry snapshots and redaction sentinels | No multi-entry registry, tolerant decoder, MRU repair, or journal model exists |
| 2 | UT-002, UT-003 | Same registry suite | Five-account, duplicate DID, generation, and tied-MRU fixtures | Current model cannot additive-upsert, limit, or choose fallback |
| 3 | IT-001 | `secure_token_storage_test.dart`, `auth_session_provider_test.dart` | Fake A/B slots with interruption at each write step | Current one-key storage loses or replaces sessions |
| 4 | UT-005, REG-003 | Existing interceptor suites | Fixed A/B sessions, device ID, concurrent completers, repeated `401`s | Interceptors still read/clear global auth |
| 5 | UT-004, IT-003 | `account_activation_coordinator_test.dart`, `account_switch_routing_test.dart` | Controlled A providers/mutations, retained B, transition harness | No hard activation boundary or stale-result fence exists |
| 6 | IT-002, UT-020, UT-022 | Auth controller and router redirect suites | Retained A; new/repeated B; canceled/failed OAuth variants | Callback overwrites A and signed-in Add route is unavailable |
| 7 | UT-010, IT-009 | Auth session/validation suite | Active A plus B/C/D with mixed delayed outcomes | Only one global session validates and results are not account-scoped |
| 8 | UT-012, IT-007 | Auth controller/sign-out cleanup/settings suites | A active; B/C MRU; success and last-account cases | Sign out clears all secure state |
| 9 | IT-013 | `appview/internal/auth/handlers_test.go` | Two sessions/subscriptions sharing installation plus failing cleaner | Existing test does not directly prove B remains active on A logout |
| 10 | UT-006–UT-008, IT-005 | Routing storage/open/pending/runtime/row flow suites | Exact, ambiguous, malformed, unbound, removed, stale-row, and generation fixtures | Opens compare only with current DID and cannot activate recipient safely |
| 11 | UT-011, IT-004 | Registration coordinator/device registration suites | Authorized permission, A/B fixed clients, rotation, failure/removal | Registration owns one active DID/client |
| 12 | UT-009, IT-006 | Account new-count/list/preferences/seen suites | A/B distinct state and counts 0/3/99/100/120 | Notification state is global rather than account-keyed |
| 13 | UT-015, IT-012 | Activation guard, router, and existing discard suites | Clean/dirty post/project/profile flows; manual/push requests | Account activation bypasses local PopScope confirmations |
| 14 | UT-013, IT-010 | `notification_sign_out_recovery_test.dart` | Ordered recorder, A cleanup token, B usable session, restart/connectivity | Existing cleanup is best-effort and cannot persist required ordering |
| 15 | UT-016–UT-019, IT-011 | Switcher model/shell widget suites | Compact/large TD-010, five accounts, missing identity, semantics | No switcher/avatar/limit/action model exists |
| 16 | UT-014, REG-010 | Secret scan, router, registry, open-event, shell suites | TD-009 sentinels including exempt callback credential | Current auth/account strings and route diagnostics expose identity/URL data |
| 17 | REG-001–REG-009 | Existing auth/router/notification/composer/feature suites | One-account registry plus representative viewer caches | Compatibility and state-isolation regressions remain unproven |
| 18 | MAN-001–MAN-003 | Supported phone/large layout and physical device | Two test accounts, provider delivery, token rotation, offline sign-out | Hermetic tests cannot prove OS/provider/secure-platform behavior |

Focused red command for step 1, from `app/`:

```sh
flutter test test/auth/models/session_registry_test.dart
```

Focused AppView command for `IT-013`, from `appview/` after `just dev-d` at repository root if the test database is not already running:

```sh
TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/auth -run 'TestLogout'
```

Feature verification commands:

```sh
# from app/
dart run build_runner build
dart analyze
flutter test test/auth test/notifications test/router
flutter test test/shared/api/providers/session_auth_interceptor_test.dart test/shared/api/providers/sign_out_on_401_interceptor_test.dart
flutter test test/settings test/feed/widgets/post_composer_sheet_discard_test.dart test/projects/widgets/project_composer_discard_test.dart

# from repository root
just test
```

## 10. Sequencing And Guardrails

- First TDD step: write `UT-001` for a two-account v1 registry round-trip, missing/corrupt active DID fallback, and fully redacted strings; confirm it fails before adding the model.
- Dependencies between work items:
  1. Registry domain and journal protocol before any provider migration.
  2. Registry provider/auth projection before fixed clients.
  3. Fixed clients and scoped `401` before activation/provider invalidation.
  4. Activation boundary before additive OAuth, startup validation, notification routing, or switcher UI.
  5. Confirmed sign-out and AppView contract before notification routing/registration work.
  6. Routing, registration, and account notification state before foreground banner/switcher counts.
  7. Unsaved-work guard before any notification-triggered activation ships.
  8. Offline recovery state machine before final switcher/UI polish and full regression.
- Guardrails:
  - Every authenticated request uses a fixed captured session; no request-time global token lookup.
  - Every asynchronous local write is fenced by DID plus session generation; active UI work also checks activation generation.
  - A registry mutation is visible only after journal read-back verification.
  - Never recover an older registry revision merely to regain an entry missing/corrupt in the newest valid revision.
  - Routing IDs choose context only and remain redacted; destination APIs authorize content.
  - Provider registration remains blocked while any cleanup-only credential exists.
  - AppView logout/deactivation must be terminal before cleanup credential deletion and replacement-token registration.
  - No bulk logout, inactive-row removal, per-account route history, OS-visible recipient copy, or combined-account surface.
  - Disable GoRouter diagnostics; never stringify account keys, leases, registry values, auth identity, tokens, bindings, raw provider payloads, or retained-account lists.
  - The OAuth callback URL exception is exact and non-generalizable; tests must still prove the callback credential is absent from logs, diagnostics, crash data, analytics, UI, and string output.
  - Run `IT-013` and full `just test` even when production Go code remains unchanged.
- Out of scope:
  - New AppView routes, linked-account APIs, database migrations, lexicon changes, PDS-token storage, or server-visible account grouping.
  - Legacy migration from `craftsky_session`.
  - Sign out all, inactive direct removal, per-account saved navigation locations, new alert thresholds, or recipient identity in provider-visible copy.
  - Replacing the existing OAuth callback transport; it is covered only by the approved narrow exception.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | Existing OAuth callback carries its handoff credential in a URL while `NFR-002` otherwise prohibits credential URLs | Without a decision, implementation required an out-of-scope exchange route | User explicitly exempted only the existing callback transport on 2026-07-18; route logging remains disabled and all other output restrictions remain |
| CPQ-002 | Non-blocking | Platform secure-storage atomicity cannot be assumed | Interrupted writes could lose all sessions | Two alternating full snapshots, prior-slot preservation, production decode/read-back verification, and TD-002 cover every application-visible interruption point |
| CPQ-003 | Non-blocking | Flutter provider dependency rebuilds alone may retain previous `AsyncValue` data or permit late mutations | Could display A as B | Transition barrier, explicit invalidator inventory, fixed clients, provider disposal, and lease checks are all mandatory; no single mechanism is treated as sufficient |
| CPQ-004 | Non-blocking | Real FCM/APNs background/terminated behavior and provider-token deletion are not hermetic | Runtime ordering could differ from fakes | Keep lifecycle simulations and require MAN-002 on physical devices before release |
| CPQ-005 | Non-blocking | Platform keychain/keystore encryption cannot be proven by Dart tests | NFR-002 at-rest assurance remains partly manual | Use `flutter_secure_storage`, inspect platform configuration/keys without values, and complete MAN-003 |
| CPQ-006 | Non-blocking | Registration/routing/recovery operational alert thresholds remain undefined | Production monitoring is incomplete | Keep bounded outcome hooks only; threshold/dashboard work remains a pre-production follow-up and does not block TDD |
| CPQ-007 | Non-blocking | The affected provider inventory is broad and future providers may be missed | A future viewer cache could cross an account boundary | Central invalidator has an architecture test/inventory assertion and new authenticated providers must register or be explicitly keyed by `AccountKey` |

No blocking implementation question remains. High-risk implementation still requires explicit user approval.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` in `app/test/auth/models/session_registry_test.dart`
- Focused command: from `app/`, `flutter test test/auth/models/session_registry_test.dart`
- First implementation target: `app/lib/auth/models/session_registry.dart`; do not touch providers or UI until the v1 model, two-slot recovery cases, redaction, additive upsert, account limit, and MRU behavior are green.
- Preserve test order: registry/MRU and journal; fixed clients/`401`; activation boundary; additive OAuth/failure preservation; startup validation; confirmed sign-out plus `IT-013`; notification routing; registration; counts/state; unsaved-work guard; offline recovery; switcher UI; privacy/regression/manual checks.
- Required verification: generated files, `dart analyze`, focused Flutter suites, `just test`, MAN-001–MAN-003 before release.
- Do not implement, commit, push, or open a PR until separately authorized.
