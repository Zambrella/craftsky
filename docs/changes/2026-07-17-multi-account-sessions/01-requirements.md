# Requirements: Multi-Account Sessions And Notification Routing

## 1. Initial Request

Allow one CraftSky installation to remain signed in to multiple CraftSky accounts at the same time. Members can add or switch accounts by pressing and holding the Profile destination in the bottom navigation, following the familiar Bluesky/Instagram interaction. Push notifications and in-app notification opens must identify and activate the correct recipient account before opening their destination. Signing out or losing a session removes only that account; no "sign out of all accounts" action is required.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/auth/models/stored_session.dart`, `app/lib/auth/providers/secure_token_storage.dart`, and `app/lib/auth/providers/auth_session_provider.dart` persist and expose one CraftSky session.
  - `app/lib/auth/providers/auth_controller.dart` completes OAuth by replacing that one session and signs it out globally.
  - `app/lib/shared/api/providers/session_auth_interceptor.dart` supplies the current session token to authenticated AppView requests, while `sign_out_on_401_interceptor.dart` clears the current global auth state on an unauthorized response.
  - `app/lib/router/app_shell.dart` owns the bottom navigation and navigation rail. The Profile destination is currently a generic person icon with tap behavior only.
  - `app/lib/notifications/services/notification_registration_coordinator.dart` registers the current ready account for push.
  - `app/lib/notifications/services/notification_routing_storage.dart` already stores notification routing bindings as a DID-keyed map.
  - `app/lib/notifications/services/notification_open_coordinator.dart` currently validates a push routing ID only against the current DID; it cannot locate and activate a different stored account.
  - `app/lib/notifications/services/notification_sign_out_cleanup.dart` deletes the installation-wide provider token when logout is unconfirmed. Multi-account sign-out must coordinate rotation and re-registration so this shared cleanup does not strand retained accounts.
  - `appview/migrations/000002_oauth_tables.up.sql` and the auth handlers already allow multiple independent CraftSky sessions.
  - `appview/migrations/000021_appview_notifications.up.sql` models one installation with multiple account subscriptions, and `appview/internal/api/notification_devices.go` registers or removes them independently.
- Existing patterns:
  - The Flutter app stores CraftSky bearer tokens in platform secure storage and keeps PDS credentials server-side.
  - Authenticated state is Riverpod-driven and consumed by API interceptors, router redirects, onboarding, profile, feeds, and notification lifecycle code.
  - Push payloads contain an opaque `accountSubscriptionId`; secure local bindings associate that ID with an account DID.
  - Notification destinations are provider-neutral typed values and are opened through one runtime path for foreground, background, and terminated launches.
- Current behavior:
  - Secure storage contains one `craftsky_session` JSON blob and cold start restores one optimistic signed-in identity.
  - Completing another sign-in overwrites the prior session.
  - Signing out, background validation failure, or a `401` transitions the whole app to `SignedOut`.
  - Push permission and registration readiness follow only the active account.
  - A push whose routing binding does not match the current DID is rejected rather than switching accounts.
- Constraints discovered:
  - Account switching changes the token used by every authenticated request and must not allow stale providers, responses, navigation state, or cached viewer-specific data to cross account boundaries.
  - Device notification permission is installation-scoped, while notification preferences, subscriptions, new counts, and notification lists are account-scoped.
  - Notification payload facts are untrusted navigation input. AppView remains authoritative for content visibility and all destination reads.
  - The existing AppView data model supports this feature without grouping accounts together or exposing one account's sessions to another account.
  - There are no existing signed-in installations, so the single-session storage format can be replaced without a legacy-session migration.
  - A well-formed notification routing ID with no retained local binding can safely use the signed-out-account message without retaining a removed binding; malformed provider data still uses the generic unavailable fallback.
  - Registering a new FCM token for the same `device_id` updates the shared `push_installations` row used by every still-active account subscription. After unconfirmed sign-out, the removed account must be deactivated before remaining accounts are registered to the replacement token, or the removed subscription will follow the rotation.
- Test/build commands discovered:
  - Focused Flutter tests from `app/`: `flutter test test/auth test/notifications test/router`.
  - Static analysis from `app/`: `dart analyze`.
  - Code generation from `app/`: `dart run build_runner build`.
  - Server verification, if AppView behavior changes: `just test` from the repository root with the compose Postgres available.

## 3. Clarifying Questions And Decisions

### Q1: What should happen when a member signs out or one stored session expires?

Answer: Remove only that account. If other accounts remain, automatically switch to the most recently used valid account and land on its Home page. There does not need to be a "sign out of all accounts" action.

Decision / implication: Session validity, unauthorized handling, push-subscription cleanup, and secure-storage removal are account-scoped. The most recently used remaining valid account becomes active; the app becomes fully signed out only when none remain.

### Q2: Where should a manual account switch land?

Answer: The selected account's Home page.

Decision / implication: A manual switch resets shell navigation to the selected account's Home root rather than preserving a separate navigation location per account. Notification-triggered switches still open their resolved notification destination after activation.

### Q3: How should activity for inactive accounts appear in app?

Answer: Show how many new notifications each inactive account has in the account switcher. A foreground in-app notification for an inactive account must also have a visual indication that it belongs to a different account.

Decision / implication: The switcher displays account-scoped numeric new-count badges for inactive accounts, using the existing visual `99+` cap. Foreground banners identify the recipient account from the secure routing binding and cached account identity before a tap switches to it.

### Q4: How should the account switcher and active identity appear?

Answer: Use a modal bottom sheet on phones and an anchored popover/menu beside Profile on large layouts. Order the active account first and inactive accounts by most recent use. Show the active account's avatar in the Profile navigation destination, with the generic person icon as fallback. At five accounts, keep Add account visible but disabled with `Maximum of 5 accounts` helper text. Inactive accounts cannot be removed directly from the switcher; the member must switch to one and use Sign out.

Decision / implication: Compact and large layouts share the same ordering, counts, disabled-full state, and accessibility semantics while using the presentation pattern appropriate to each layout.

### Q5: How should switching behave around loading and unsaved work?

Answer: Manual switching works offline from the retained secure session and lands on Home, where ordinary offline/retry UI may appear. A newly added account follows onboarding when incomplete and otherwise lands on Home. If switching would abandon unsaved edits or an in-progress compose flow, ask for confirmation. Canceling keeps the current account and discards any notification-open attempt rather than switching later.

Decision / implication: Account activation is local and does not depend on a successful initial content request, but existing unsaved-work protections remain authoritative.

### Q6: How should retained sessions be validated at startup?

Answer: Restore the active account immediately, validate it first with authenticated `whoami`, then validate inactive accounts opportunistically in the background with bounded concurrency. Network or server availability errors retain cached sessions; only an authoritative unauthorized response removes an account.

Decision / implication: Multi-account validation does not block startup and one inactive account's failure cannot disrupt another account.

### Q7: What notification behavior is required beyond correct routing?

Answer: Open the switcher immediately with cached or blank counts and refresh accounts independently. A foreground notification refreshes the recipient account's authoritative new count rather than incrementing locally. An inactive-account banner keeps the existing title/body and adds the account avatar plus `For @handle`. OS-visible notifications do not add recipient identity in this slice. If a notification's account has been removed, keep the current account unchanged and show `This notification belongs to an account that is no longer signed in`, with no automatic sign-in prompt or navigation.

Decision / implication: Duplicate push delivery cannot inflate local counts, provider payload privacy remains unchanged, and stale notification opens fail visibly without crossing accounts.

### Q8: What happens when account-scoped sign-out cannot reach AppView?

Answer: Remove the selected account from usable local sessions immediately, retain all other sessions, rotate the shared provider token, and re-register the remaining accounts when connectivity returns. Reject stale opens for the removed account during the transition.

Decision / implication: Offline sign-out favors immediate local removal and account isolation. The removed token is quarantined in secure storage solely as a pending cleanup credential and can never reactivate the account. When online, AppView logout/deactivation for that account must complete or return authoritative unauthorized before the replacement provider token is registered for remaining accounts; the cleanup credential is then deleted. Remaining accounts may have a temporary push-delivery gap, but the existing single-account behavior of deleting a shared provider token without coordinated re-registration must not be reused unchanged.

## 4. Candidate Approaches

### Option A: Device-Local Session Registry With One Active Account

Summary: Store independent CraftSky sessions in secure storage keyed by DID, persist an active-account pointer and recent-use order, and make all authenticated Flutter dependencies derive from the active session. Reuse the existing AppView session and per-installation push-subscription contracts.

Pros:

- Matches the requested simultaneous sign-in and fast switching behavior.
- Keeps bearer tokens in secure storage and PDS credentials in AppView.
- Uses the existing server model for independent sessions and per-account push subscriptions.
- Keeps account membership private to the device rather than creating a server-visible account group.

Cons:

- Requires careful invalidation or account-keying across authenticated providers and caches.
- Requires background registration and validation work for inactive stored accounts.
- Requires replacing the current single-session storage model before launch.

Risks:

- A stale request or provider can expose one account's viewer-specific state after a switch unless account boundaries are explicit.
- Platform secure-storage writes must preserve the registry and active pointer atomically enough to recover safely after interruption.

### Option B: AppView-Managed Linked Account Group

Summary: Add a server-side account group that returns or brokers multiple account sessions through one device-level identity.

Pros:

- Could centralize account discovery and switching across installations.
- Could reduce the amount of local session orchestration.

Cons:

- Introduces server-visible links between otherwise independent identities.
- Expands authentication, authorization, privacy, API, and migration scope.
- Does not remove the need to isolate active-account state in Flutter.

Risks:

- A server authorization defect could expose or activate the wrong linked account.
- The approach conflicts with the minimum-data design when device-local knowledge is sufficient.

## 5. Recommended Direction

Recommended approach: Option A, a secure device-local session registry with exactly one active account at a time.

Why: It satisfies simultaneous sign-in, familiar switching, and correct notification routing while preserving independent AppView sessions and existing architectural boundaries. AppView already supports multiple sessions and multiple account subscriptions on one installation; no cross-account server relationship is needed.

## 6. Problem / Opportunity

Members who participate through more than one CraftSky identity currently have to replace and later recreate their session whenever they change accounts. Notification delivery is also tied to the currently active account, so an installation cannot safely receive and open notifications for multiple retained accounts. Multi-account support removes that friction without weakening account isolation.

## 7. Goals

- G-001: Retain multiple independent CraftSky sessions securely on one installation.
- G-002: Make adding and switching accounts quick and familiar from the Profile navigation destination.
- G-003: Receive push notifications for every eligible retained account.
- G-004: Activate the notification's recipient account before opening any foreground, in-app, background, or terminated notification destination.
- G-005: Scope sign-out, session expiry, preferences, badges, caches, and authenticated data to the relevant account.
- G-006: Make new activity for inactive accounts visible without merging account notification surfaces.

## 8. Non-Goals

- NG-001: Link accounts together on AppView or expose a member's account collection to other installations.
- NG-002: Add a "sign out of all accounts" action.
- NG-003: Merge notifications, feeds, drafts, preferences, or saved content from multiple accounts into one combined surface.
- NG-004: Allow simultaneous side-by-side use of more than one active account in the UI.
- NG-005: Synchronize the active-account choice or retained-account list between devices.
- NG-006: Change atproto OAuth, store PDS tokens on-device, or read CraftSky content directly from a PDS.
- NG-007: Change notification eligibility, category, payload-fact, visibility, moderation, or delivery-retry policy.
- NG-008: Retain more than five distinct CraftSky accounts on one installation.
- NG-009: Add recipient-account identity to background or terminated OS-visible notification copy.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Multi-account member | A member with two or more CraftSky identities on one installation. | Add, retain, identify, switch, and remove accounts without repeatedly signing in. |
| Single-account member | A member who uses one CraftSky account. | Keep sign-in and ordinary navigation simple without requiring use of the switcher. |
| Notification recipient | A retained account receiving activity while active or inactive. | Receive the alert and open it under the intended identity. |
| Flutter client | Holds CraftSky session tokens and active UI state. | Secure storage, deterministic active-account selection, and strict account isolation. |
| AppView | Issues independent CraftSky sessions and serves account-authorized data. | Continue authorizing every request by the selected session without linked-account trust. |
| Push provider | Delivers installation-level FCM/APNs messages. | Continue using bounded opaque routing data without receiving session credentials. |

## 10. Current Behavior

The app restores one secure session, sends its token with every authenticated request, and registers the installation's push token for that account. A new OAuth completion replaces the existing session. A sign-out, validation failure, or `401` clears global authentication. Notification opens validate the opaque routing ID against only the current account, so a notification for a different account cannot be opened.

## 11. Desired Behavior

The app securely retains independent sessions keyed by DID and persists one active DID. Long-pressing the Profile navigation destination opens an account switcher that identifies retained accounts, shows each inactive account's numeric new-notification count, and offers Add account. Choosing an account activates its session, resets navigation to that account's Home root, and refreshes all account-scoped state without affecting other sessions. The installation maintains push subscriptions and secure routing bindings for every eligible retained account. A foreground notification for an inactive account visibly identifies its recipient account. When a notification is opened, the routing binding identifies its recipient DID; the app activates that stored session first and then follows the existing typed destination flow. Signing out or invalidating one account removes only that session and subscription, selecting the most recently used remaining account when available.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A member shall be able to remain signed in to multiple independent CraftSky accounts on one installation. | Eliminates repeated authentication for multi-identity participation. | Initial request | AC-001, AC-003 |
| BR-002 | Business | Must | A notification open shall use the notification recipient's retained account and shall never open under another account. | Prevents incorrect actions and cross-account disclosure. | Initial request | AC-010, AC-011, AC-012, AC-020, AC-024, AC-028 |
| BR-003 | Business | Must | Loss or removal of one account shall not sign out other retained accounts. | Preserves independent sessions. | User answer | AC-015, AC-016, AC-029 |
| FR-001 | Functional | Must | The app shall securely persist a DID-keyed collection of CraftSky sessions, one active DID, and enough recent-use ordering to choose a remaining account deterministically. | Provides durable independent sessions and account-scoped fallback. | Recommended direction / User answer | AC-001, AC-015, AC-018 |
| FR-003 | Functional | Must | Completing Add account OAuth shall add or refresh the returned DID's session without deleting other stored sessions and shall make the newly completed account active; an incomplete account shall continue through onboarding, while an onboarded account shall land on Home. | Converts existing replacement behavior into additive sign-in while preserving account readiness. | Initial request / Existing OAuth flow / Grilling decision | AC-003, AC-004 |
| RULE-001 | Business rule | Must | At most one stored session may exist for a DID; signing in again to an already retained DID shall replace that DID's token and cached identity rather than create a duplicate switcher row. | A DID is the stable account identity. | Architectural convention | AC-004 |
| FR-004 | Functional | Must | Pressing and holding the Profile destination shall open an account switcher as a modal bottom sheet on compact layouts and an anchored popover/menu on large layouts; both presentations shall provide equivalent account actions, ordering, badges, full-state messaging, semantics, and keyboard/accessibility behavior. | Implements the requested familiar entry point using existing layout patterns. | Initial request / Codebase / Grilling decision | AC-005, AC-006 |
| FR-005 | Functional | Must | The account switcher shall identify each retained account using available avatar, display name, and handle, place the active account first, order inactive accounts by most recent use, distinguish the active account, and provide Add account; missing profile metadata shall fall back to the cached handle without blocking switching. | Members must be able to identify accounts and understand fallback order reliably. | Initial request / UX analysis / Grilling decision | AC-005, AC-007, AC-018 |
| FR-006 | Functional | Must | A normal tap on the Profile destination shall retain its existing behavior and shall not open the account switcher. | Avoids regressing primary navigation. | Existing behavior | AC-006 |
| FR-007 | Functional | Must | Selecting a retained account shall make its session active, update recent-use order, and transition authenticated routing and UI to that account without revoking or removing any other session. | Defines switching rather than reauthentication. | Recommended direction | AC-008, AC-018 |
| NFR-001 | Non-functional | Must | Account activation shall prevent stale authenticated requests, responses, provider state, navigation state, and viewer-specific caches from being presented or mutated as the newly active account. | Cross-account state leakage is the highest implementation risk. | Security analysis | AC-008, AC-009 |
| FR-008 | Functional | Must | Every authenticated AppView request shall use the session selected for that request's account context; active-account UI requests shall use only the active session. | A global token swap alone is unsafe during concurrent work. | Architectural rules / Codebase | AC-009, AC-016 |
| FR-009 | Functional | Must | Account-scoped data, including onboarding state, notification list, notification new count, notification preferences, feeds, profiles with viewer state, drafts, saved data, and optimistic mutations, shall be isolated by DID or invalidated at an account boundary. | These values may differ by viewer. | Codebase analysis | AC-008, AC-009 |
| FR-010 | Functional | Must | With device notification permission authorized, the app shall register the current installation token and retain an active account subscription and secure routing binding for every signed-in, notification-eligible account, including inactive accounts. | Enables simultaneous delivery. | Initial request / Existing server contract | AC-010, AC-014 |
| FR-011 | Functional | Must | FCM token refresh and relevant lifecycle retry shall update registration for every retained eligible account without requiring each account to become manually active. | Provider tokens can rotate while accounts are inactive. | Push lifecycle constraint | AC-010, AC-014 |
| RULE-002 | Business rule | Must | OS notification permission shall remain installation-scoped, while notification preferences, subscriptions, notification lists, seen state, and new counts remain account-scoped. | Preserves current server semantics. | Existing contract | AC-010, AC-013 |
| FR-012 | Functional | Must | For provider-delivered foreground banners, background opens, and terminated-launch opens, the app shall resolve the opaque `accountSubscriptionId` to exactly one retained DID before destination inference or navigation; an in-app notification-list row shall remain bound to the active DID whose authenticated list response produced it. | Creates an account-safe rule for both provider opens and account-scoped in-app rows. | Initial request / Existing routing contract | AC-011, AC-012, AC-020 |
| FR-013 | Functional | Must | When a valid notification maps to an inactive retained account, the app shall activate that account, complete the account-boundary transition, and then open the existing typed notification destination using that account's session. | Fulfills correct-account routing. | Initial request | AC-011 |
| FR-014 | Functional | Must | A notification with a missing, malformed, or ambiguous routing binding shall not switch accounts or open content under the current account and shall use the generic safe unavailable fallback. A well-formed binding that has no retained local account shall follow FR-025. | Treats provider data as untrusted and distinguishes invalid payloads from signed-out accounts without retaining removed bindings. | Security analysis / Grilling decision | AC-012, AC-028 |
| FR-015 | Functional | Must | If notification readiness is delayed during cold start or account activation, only the latest pending open shall continue once the identified account is ready; no pending open shall cross removal or reauthentication of that account. | Preserves current latest-open behavior across a new account boundary. | Existing notification contract | AC-011, AC-012 |
| FR-016 | Functional | Must | Signing out shall revoke and remove only the active selected account's session, deactivate only that account's installation subscription, remove only its routing binding and account-scoped local state, and retain all other accounts. If server sign-out cannot be confirmed, local removal shall still complete and shared push-token recovery shall follow FR-026. | Implements the confirmed account-scoped sign-out decision. | User answer / Grilling decision | AC-015, AC-029 |
| FR-017 | Functional | Must | A `401` or failed background session validation shall invalidate only the session used by that request; it shall not clear another active or inactive session. | Unauthorized status is session-specific. | User answer / Codebase constraint | AC-016 |
| FR-018 | Functional | Must | After the active account is removed or invalidated, the app shall activate the most recently used remaining valid account and land on its Home root; it shall enter signed-out routing only when no sessions remain. | Provides deterministic continuity. | User answer / Grilling decision | AC-015, AC-016, AC-018 |
| FR-019 | Functional | Must | After a manual account selection completes its account-boundary transition, the app shall navigate to the selected account's Home root rather than restore the previous account's route or maintain per-account navigation locations. Switching shall complete from secure local state while offline; destination content may show its normal offline/retry state and shall not cause a rollback to the prior account. | Gives manual switching a predictable, resilient, state-isolated destination. | User answer / Grilling decision | AC-022 |
| FR-020 | Functional | Must | The switcher shall open immediately with cached or blank account-scoped new-count badges, refresh inactive accounts independently using each account's session, hide zero counts, and visually cap counts above 99 as `99+`; one count failure shall not block the switcher or account selection. A foreground notification shall refresh the recipient account's authoritative count rather than increment a local value. | Makes inactive-account activity visible without blocking switching or overcounting duplicate pushes. | User answer / Existing new-count contract / Grilling decision | AC-023, AC-031 |
| FR-021 | Functional | Must | A foreground in-app notification mapped to an inactive retained account shall keep the existing title/body and add a compact recipient line with the account avatar and `For @handle`; cached handle and the generic avatar shall be fallbacks. Tapping the banner shall follow the normal account activation and destination flow. | Distinguishes the recipient from the actor without making the banner noisy. | User answer / Grilling decision | AC-024 |
| FR-022 | Functional | Must | The Profile destination in compact and large navigation shall display the active account's avatar with a clear selected-state treatment and shall use the existing generic person icon when no avatar is available. | Makes the current identity visible before switching. | Grilling decision | AC-025 |
| FR-023 | Functional | Must | If a manual switch or notification-triggered switch would leave unsaved edits or an in-progress compose flow, the app shall require the existing unsaved-changes confirmation before changing accounts. Canceling shall keep the current account, discard the open attempt, and never switch later; any retained draft remains scoped to its originating DID. | Prevents accidental work loss and delayed surprise switching. | Grilling decision | AC-026 |
| FR-024 | Functional | Must | Startup shall restore the active cached session without waiting for network validation, validate the active account first through authenticated `whoami`, and validate inactive sessions opportunistically with bounded concurrency. Network or server availability failures shall retain cached sessions; only authoritative unauthorized or identity-mismatch results shall remove the affected account. | Keeps startup responsive while detecting revoked sessions account-by-account. | Existing behavior / Grilling decision | AC-027 |
| FR-025 | Functional | Must | When a notification open maps to an account that is no longer retained, the app shall keep the current account unchanged, perform no destination navigation or automatic sign-in, and show `This notification belongs to an account that is no longer signed in`. | Gives a clear stale-open outcome without crossing account boundaries. | Grilling decision | AC-028 |
| FR-026 | Functional | Must | If account-scoped sign-out cannot be confirmed with AppView, the app shall remove the account from the usable registry immediately, quarantine its token in secure storage solely as a non-activatable pending cleanup credential, invalidate/rotate the shared provider token, and reject stale opens. When connectivity returns, the app shall use the quarantined token to complete AppView logout/deactivation (treating authoritative unauthorized as already complete), delete the cleanup credential, and only then register the replacement provider token for every remaining eligible account. | Re-registering first would move the removed account's still-active subscription onto the replacement token because the installation row is shared. | Codebase finding / Grilling decision | AC-029 |
| RULE-003 | Business rule | Must | The product shall not expose a "sign out of all accounts" action in this feature. | Explicit user decision. | User answer | AC-017 |
| RULE-004 | Business rule | Must | One installation shall retain no more than five distinct account DIDs. At five retained accounts, Add account shall remain visible but disabled with `Maximum of 5 accounts` helper text. | Establishes a discoverable supported account limit. | User answer / Grilling decision | AC-021 |
| RULE-005 | Business rule | Must | An inactive account shall not expose a direct remove or sign-out action in the switcher. To remove it, the member must first switch to that account and then use the existing Sign out action. | Keeps account removal explicit and within the selected account's settings context. | Grilling decision | AC-030 |
| NFR-002 | Non-functional | Must | Session tokens, cleanup credentials, and the local DID-to-`accountSubscriptionId` binding map shall remain in platform secure storage. The opaque `accountSubscriptionId` may appear only where required by the existing routing contract: the authenticated AppView registration response and the provider notification payload. Tokens, cleanup credentials, and routing IDs shall never be included in logs, analytics, crash context, URLs, UI text, or string representations. DIDs, handles, raw provider payloads, and retained-account lists shall never be included in logs, analytics, crash context, or diagnostic string representations; cached handles and avatars may appear only on the account-identification UI required by FR-005, FR-021, and FR-022. An account's usable token and local routing binding shall be deleted when it is removed; FR-026 may retain only a non-activatable cleanup credential until revocation/deactivation succeeds or is authoritatively unnecessary, after which it shall be deleted. | Maintains credential confidentiality while allowing the opaque, non-authorizing routing ID to select the correct local account context, allowing intentional account-identification UI, and allowing safe recovery from unconfirmed sign-out. | Existing security pattern / Codebase constraint | AC-019, AC-029 |
| NFR-003 | Non-functional | Should | Manual switching should show an immediate transition state and make the selected identity evident before account-authorized content becomes interactive. | Reduces accidental actions during account transition. | UX risk | AC-008 |
| NFR-004 | Non-functional | Should | Single-account users should experience no additional required steps, prompts, or visible switcher during ordinary tap navigation. | Keeps the common path unchanged. | Product fit | AC-006 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given two valid accounts have completed sign-in on one installation, when the app restarts, then both sessions remain retained and the previously active account is restored. |
| AC-003 | BR-001, FR-003 | Given account A is retained, when Add account completes successfully for account B, then A and B remain signed in, B becomes active, and B enters onboarding if incomplete or Home if already onboarded. |
| AC-004 | FR-003, RULE-001 | Given account A is already retained, when sign-in completes again for account A, then its credentials and cached identity are refreshed and only one account A entry exists. |
| AC-005 | FR-004, FR-005 | Given a signed-in member on a compact layout, when they press and hold the Profile destination, then a modal bottom-sheet switcher identifies every retained account, places the active account first, orders inactive accounts by recent use, and offers Add account without navigating to Profile first. |
| AC-006 | FR-004, FR-006, NFR-004 | Given any supported layout, when the member normally activates the Profile destination, then Profile navigation works as before; when they invoke the account-switcher action, a bottom sheet opens on compact layouts or an anchored popover/menu opens on large layouts with equivalent behavior and accessibility. |
| AC-007 | FR-005 | Given a retained account has unavailable avatar, display name, or freshly fetched profile data, when the switcher opens, then its cached handle still identifies a selectable account. |
| AC-008 | FR-007, FR-009, NFR-001, NFR-003 | Given account A is active with account-scoped UI state and account B is retained, when B is selected, then a visible transition occurs, B becomes clearly active, and A's viewer-specific state is not presented or mutated as B. |
| AC-009 | NFR-001, FR-008, FR-009 | Given an account A request is in flight while the app activates B, when A's response completes, then it cannot populate or mutate B's account context, and subsequent B UI requests carry only B's session. |
| AC-010 | BR-002, FR-010, RULE-002 | Given notification permission is authorized and accounts A and B are retained and eligible, when registration settles, then the installation has independent active push subscriptions and DID-keyed routing bindings for both accounts even though only one is active. |
| AC-011 | BR-002, FR-012, FR-013, FR-015 | Given A is active and a foreground, background, or terminated notification open maps uniquely to retained account B, when the open is handled, then B becomes active before the typed destination opens and all destination reads use B's session. |
| AC-012 | BR-002, FR-012, FR-014, FR-015 | Given a notification routing ID is missing, malformed, or ambiguous, when it is opened, then the app neither changes accounts nor opens the destination under another account and instead uses the generic safe unavailable fallback. |
| AC-013 | RULE-002 | Given two retained accounts have different notification preferences and new counts, when the member switches between them, then each account shows and mutates only its own preferences, notification list, seen state, and new count. |
| AC-014 | FR-010, FR-011 | Given the provider rotates the installation token while multiple eligible accounts are retained, when registration retry completes, then every retained eligible account is registered to the latest token without manual switching. |
| AC-015 | BR-003, FR-016, FR-018 | Given A is active and B is the most recently used remaining valid account, when A signs out, then only A's session, routing binding, subscription, and local account state are removed, B remains signed in, becomes active, and lands on Home. |
| AC-016 | BR-003, FR-008, FR-017, FR-018 | Given a request made with one retained account's session returns `401` or validation rejects that session, when invalidation completes, then only that account is removed; another valid retained account remains available and, if activation is necessary, the most recently used one becomes active on Home. |
| AC-017 | RULE-003 | Given the account switcher and settings are inspected, then no "sign out of all accounts" action is exposed. |
| AC-018 | FR-001, FR-007 | Given several accounts have been used, when the active one is removed, then the valid remaining account with the most recent prior activation is selected deterministically. |
| AC-019 | NFR-002 | Given storage, registration, provider payload, logging, analytics, crash reporting, URL, UI, string-output, and cleanup paths are inspected and exercised, then session tokens and cleanup credentials are stored only in secure storage, session tokens travel only in authorized request headers, and the DID-to-routing-ID map exists only in secure storage; the opaque `accountSubscriptionId` appears only in the authenticated registration response and provider payload; credentials and routing IDs are absent from diagnostic and user-visible output; DIDs, handles, raw provider payloads, and retained-account lists are absent from diagnostics while intended cached handle/avatar identity appears only in the switcher, Profile destination, and inactive-recipient banner; removed accounts have no usable token or local binding; and any quarantined cleanup credential is non-activatable and deleted after terminal cleanup. |
| AC-020 | BR-002, FR-012 | Given account A's in-app notification list is visible, when the member taps one of its rows, then the destination opens under A; if an account transition has made that row stale, it cannot open under the newly active account. |
| AC-021 | RULE-004 | Given five distinct accounts are retained, when the account switcher opens, then Add account remains visible but disabled with `Maximum of 5 accounts` and a sixth distinct account cannot be added; after one account is signed out, Add account becomes enabled. |
| AC-022 | FR-019 | Given account A is active on any route and account B is retained, when the member manually selects B while online or offline, then B becomes active and the app lands on B's Home root without showing A's route or restoring a separate prior route for B; an initial content failure shows normal retry UI and does not reactivate A. |
| AC-023 | FR-020 | Given inactive accounts B and C have cached or fetched new counts of 3 and 120, when the switcher opens, then it is immediately usable, B shows `3`, C shows `99+`, a zero or not-yet-known count has no badge, refreshes update independently, and any one account's count failure does not prevent account selection. |
| AC-024 | BR-002, FR-021 | Given account A is active and a foreground notification maps to inactive account B, when the in-app banner appears, then it keeps the existing notification title/body and shows B's avatar plus `For @handle` with defined fallbacks; when tapped, B becomes active before the notification destination opens. |
| AC-025 | FR-022 | Given an active account with an avatar, when compact or large navigation renders, then the Profile destination shows that avatar with a clear selected state; when no avatar is available, it shows the generic person icon. |
| AC-026 | FR-023 | Given switching from A would leave unsaved edits, when a manual selection or notification tap requests B, then the existing confirmation appears before activation; confirming switches normally, while canceling keeps A active, retains A's scoped draft where applicable, discards the open attempt, and never switches later. |
| AC-027 | FR-024 | Given active account A and inactive accounts B and C are cached at startup, when the app launches, then A is restored without waiting for network and validated first, B and C validate later with bounded concurrency, network failures retain their sessions, and an authoritative unauthorized result removes only its account. |
| AC-028 | FR-025 | Given A is active and a tapped notification belongs to a removed account, when the open is handled, then A remains active, no destination or sign-in flow opens, and the app shows `This notification belongs to an account that is no longer signed in`. |
| AC-029 | FR-016, FR-026, NFR-002 | Given A and B are retained and A signs out while AppView is unreachable, when local cleanup completes, then A cannot be activated, B remains signed in, stale A opens are rejected, A's token is quarantined only for cleanup, and the shared provider token enters rotation recovery; when connectivity returns, A is logged out/deactivated before the cleanup credential is deleted and B is registered to the replacement token, so A's subscription cannot follow the rotation. |
| AC-030 | RULE-005 | Given A is active and B is inactive, when the switcher is inspected, then B has no direct remove or sign-out action; removing B requires selecting B and using Sign out from B's account context. |
| AC-031 | FR-020 | Given a foreground push is received for inactive account B, when notification state refreshes, then the app fetches B's authoritative new count and does not increment a local counter, so duplicate provider delivery cannot inflate the badge. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-002 | Registry or active-account pointer is corrupt or refers to a missing DID. | Preserve any independently decodable sessions, select the most recently used valid account, and fail signed out only when none are recoverable. | FR-001, NFR-002 |
| EC-003 | Same DID is added again. | Refresh that DID's session and identity without duplicating it. | RULE-001 |
| EC-004 | Add-account OAuth is cancelled or fails. | Keep the prior active account and all retained sessions unchanged. | FR-003 |
| EC-005 | Account switch occurs with requests or optimistic mutations in flight. | Old-account results stay scoped to the old DID and cannot affect the new account. | NFR-001, FR-009 |
| EC-006 | Inactive account session expires. | Remove only that account and its subscription/binding; do not interrupt the active account. | FR-017, FR-018 |
| EC-007 | Active account session expires. | Remove it and activate the most recently used remaining valid account on Home, or show signed-out flow if none remains. | FR-017, FR-018 |
| EC-008 | Notification arrives for an inactive retained account. | Receive it normally; on open, activate that account before navigation. | FR-010, FR-013 |
| EC-009 | Notification arrives for an account removed after provider delivery. | Keep the current account, do not navigate or prompt for sign-in, and show the signed-out-account message. | FR-025 |
| EC-010 | Two notification opens arrive while startup/account transition is pending. | Only the latest eligible open continues. | FR-015 |
| EC-011 | Push token rotates while an inactive account cannot reach AppView. | Retain retry eligibility for that account without preventing other accounts from updating. | FR-011 |
| EC-012 | Account profile image or display name changes. | Switching remains possible using cached handle; refreshed metadata may update the presentation later. | FR-005 |
| EC-013 | Device notification permission is denied. | Accounts remain retained and switchable; per-account server preferences remain editable, but device push registration does not proceed until permission is authorized. | RULE-002 |
| EC-014 | A fifth account is retained and the member tries to add another distinct DID. | Do not create a sixth registry entry; explain that an existing account must be removed first without disturbing any retained session. | RULE-004 |
| EC-015 | An inactive account's new-count request fails or its session expires while the switcher is open. | Keep all other account indicators and switching usable; omit or show an unavailable state for that count and apply account-scoped invalidation if unauthorized. | FR-017, FR-020 |
| EC-016 | A foreground notification maps to an inactive account whose avatar or display name is unavailable. | Identify the account with its cached handle and preserve the correct-account tap behavior. | FR-021 |
| EC-017 | Manual or notification-triggered switching would abandon unsaved work. | Ask for confirmation; canceling keeps the current account and discards the switch/open attempt. | FR-023 |
| EC-018 | App starts offline with several cached sessions. | Restore the active account immediately, retain all cached sessions, and defer validation without blocking use. | FR-024 |
| EC-019 | OS notification is opened after its account was signed out. | Keep the current account, show the signed-out-account message, and do not prompt or navigate. | FR-025 |
| EC-020 | Offline sign-out cannot revoke the account or remove its server subscription. | Remove it from usable sessions, quarantine its token for cleanup only, reject stale opens, invalidate the shared provider token, deactivate the removed account first when online, then delete the cleanup credential and register remaining accounts to the replacement token. | FR-016, FR-026, NFR-002 |

## 15. Data / Persistence Impact

- New fields:
  - Flutter secure storage requires a versioned session registry keyed by DID, an active DID, and recent-use metadata.
  - Each retained entry contains the existing CraftSky token, DID, cached handle, and any minimum switcher identity metadata chosen during design.
  - The registry enforces a maximum of five distinct DIDs.
  - Push recovery requires a secure, non-activatable pending-cleanup record containing the minimum credential and state needed to deactivate the removed account, rotate the installation token, and re-register remaining eligible accounts after an unconfirmed offline sign-out.
- Changed fields:
  - The current pre-launch `craftsky_session` blob is replaced by the versioned registry.
  - Existing DID-keyed notification routing bindings remain compatible but must support reverse lookup from opaque routing ID to DID.
- Migration required:
  - No client data migration is required because there are no existing signed-in installations.
  - No AppView database migration is expected from current findings.
- Backwards compatibility:
  - Preserve a simple single-account UX for members who do not add another account.
  - AppView's independent session tokens and installation/account subscription schema remain authoritative.

## 16. UI / API / CLI Impact

- UI:
  - Add a long-press modal bottom-sheet account switcher to compact navigation and an anchored popover/menu beside Profile on large layouts.
  - Show the active account first and inactive accounts by most recent use, with retained account identity, active state, and Add account.
  - Use the active account avatar as the Profile destination icon, with a generic-person fallback and clear selected state.
  - Open the switcher immediately with cached or blank numeric new-notification badges for inactive accounts, capped visually at `99+`, then refresh independently.
  - At five accounts, keep Add account visible but disabled with `Maximum of 5 accounts` helper text.
  - Do not expose direct inactive-account removal; members switch to the account and use Sign out.
  - Show a non-interactive transition boundary while changing active accounts.
  - Land manual switches on the selected account's Home root.
  - Preserve unsaved-work confirmation before account changes and discard a canceled notification-open attempt.
  - Identify an inactive recipient on foreground banners with its avatar and `For @handle`; leave OS-visible recipient copy unchanged.
  - Show the explicit signed-out-account message for stale notification opens without changing accounts.
  - Keep ordinary Profile tap navigation unchanged.
- API:
  - No new AppView route is expected. Existing login, handoff, whoami, current-session logout, profile, notification-device, and notification endpoints are reused under the appropriate retained session.
  - The API architecture contract and camelCase wire format remain unchanged.
- CLI:
  - None identified.
- Background jobs:
  - Client lifecycle/token-refresh registration must cover all retained eligible accounts.
  - Inactive-account new counts refresh with their own sessions when the switcher needs them and when relevant foreground/resume activity makes cached counts stale.
  - Startup restores the active cached session immediately, validates it first, and then validates inactive sessions opportunistically with bounded concurrency.
  - Unconfirmed offline sign-out rotates the shared provider token and schedules re-registration of every remaining eligible account when connectivity returns.
  - Existing server push delivery remains per account subscription.

## 17. Security / Privacy / Permissions

- Authentication:
  - Each account retains an independent CraftSky bearer token. Account switching selects a token; it never exchanges or derives one token from another.
  - OAuth completion is additive and no account can authorize access to another retained account.
- Authorization:
  - AppView continues to authorize every request from that request's session token.
  - Notification routing selects an account context but does not authorize destination content.
- Sensitive data:
  - Session tokens, cleanup credentials, and the local DID-to-`accountSubscriptionId` map remain secure-storage-only. The opaque routing ID may travel in its authenticated registration response and provider payload, but it is untrusted, non-authorizing, and redacted everywhere else. Cached handles and avatars may appear only on the intended account-identification UI; identity and retained-account data remain absent from diagnostics.
  - The retained-account collection is local private data and is not uploaded as a group.
  - An offline-removed account may leave a quarantined cleanup credential in secure storage, but it is excluded from the account registry, cannot authenticate ordinary app requests, and is deleted immediately after terminal server cleanup.
- Abuse cases:
  - Reject unknown or ambiguous notification routing IDs instead of guessing from the active account.
  - Prevent stale asynchronous work from crossing a DID boundary.
  - Clear only the compromised/expired session and its account-bound local data.

## 18. Observability

- Events:
  - Privacy-safe outcomes for account add success/failure, manual switch success/failure/cancel, account-scoped invalidation, notification-triggered switch success/failure/cancel, shared-token recovery, and routing-binding mismatch.
- Logs:
  - Include operation and outcome class, but never tokens, routing IDs, raw provider payloads, handles, DIDs, or retained-account lists.
- Metrics:
  - Account-switch failures, account-scoped `401` invalidations, notification opens requiring account activation, removed-account opens, unknown-binding opens, inactive-account new-count refresh failures, session-validation outcomes, provider-token recovery, and per-account push registration outcomes.
- Alerts:
  - Consider an alert for sustained increases in push registration or notification account-routing failures; thresholds are an operational follow-up.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Stale providers, caches, in-flight responses, or optimistic mutations cross an account boundary. | Private or viewer-specific state could appear under the wrong identity, or an action could use the wrong account. | Key state by DID where it must survive switching; otherwise invalidate/dispose it, cancel work where possible, and reject stale completions by account generation. |
| RISK-002 | A global `401` interceptor clears the wrong account after a concurrent switch. | Valid sessions could be removed or the app could sign out unexpectedly. | Bind unauthorized handling to the session token/account that made the request. |
| RISK-003 | A partial secure-storage write corrupts the retained session registry. | Member must reauthenticate to several accounts. | Use versioned validation, preserve independently recoverable entries, and test interrupted and corrupt storage writes. |
| RISK-004 | Token refresh updates only the active account. | Inactive accounts silently stop receiving pushes. | Re-register every eligible retained account and track retry per DID. |
| RISK-005 | Notification routing switches by untrusted or stale provider data. | Destination could open under the wrong account. | Require an exact unique match against secure local bindings before switching. |
| RISK-006 | Retained inactive sessions expand credential exposure on a lost device. | More accounts are accessible until sessions are revoked or device storage is locked. | Use platform secure storage, preserve server revocation, delete account credentials promptly, and document normal device-lock expectations. |
| RISK-007 | The five-account limit is enforced only in the switcher UI and bypassed by another sign-in completion path. | A sixth account could be retained or an existing registry could become inconsistent. | Enforce the distinct-DID limit in the session-registry domain layer as well as the UI, while allowing replacement of an existing DID. |
| RISK-008 | Fetching inactive-account counts uses the active token or globally handles one account's failure. | Counts could leak across identities or one expired inactive session could disrupt the active account. | Bind every count request and unauthorized outcome to its target DID, isolate failures, and cap concurrent work to the five retained accounts. |
| RISK-009 | Offline sign-out reuses global provider-token deletion or re-registers remaining accounts before deactivating the removed account. | Every remaining account could stop receiving push notifications, or the removed subscription could follow the replacement token and continue producing visible stale alerts. | Quarantine the removed credential for cleanup only, reject stale opens, deactivate the removed account first, delete the cleanup credential, and then re-register all remaining eligible accounts to the replacement token. |
| RISK-010 | Account activation bypasses unsaved-work guards or leaves a canceled notification open pending. | Work could be lost or the app could switch accounts unexpectedly later. | Route manual and notification activation through one guarded coordinator and destroy the open attempt on cancellation. |
| RISK-011 | Concurrent startup validation removes or mutates the wrong retained account. | One revoked token could disrupt valid sessions. | Bind each validation result to its original DID/token generation, validate the active account first, and bound inactive concurrency. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-004 | The existing AppView session and push-subscription APIs can be called independently with each retained session without a new route or migration. | Server API or database work would need to be added to the coding plan. |

## 21. Open Questions

None identified.

## 22. Review Status

Status: Reviewed
Risk level: High
Review recommended: Required
Reviewer: Codex
Date: 2026-07-18
Notes: Grilling completed on 2026-07-17 with all product questions resolved. The requirements and acceptance tests were jointly reviewed and corrected on 2026-07-18. Because the work remains High risk, explicit user approval is required before implementation begins.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-003`, `FR-001`, `FR-003` through `FR-026`, `RULE-001` through `RULE-005`, `NFR-001`, and `NFR-002`.
- Suggested test levels:
  - Unit: registry recovery, recent-use ordering/fallback, five-account limit enforcement, duplicate-DID replacement at the limit, reverse binding lookup, inactive-account new-count isolation, notification account selection, validation result scoping, shared-token recovery state, and account-scoped unauthorized handling.
  - Provider/controller: additive OAuth completion with onboarding routing, online/offline activation to Home, unsaved-work confirmation/cancellation, active-first session validation, per-account sign-out, per-account count fetching, authoritative foreground count refresh, per-account push registration, token rotation/recovery, and latest pending notification open.
  - Widget/router: bottom-sheet and anchored switchers, active-avatar Profile destination, MRU ordering, numeric inactive-account indicators, disabled five-account Add state, inactive-account banner `For @handle`, signed-out-account feedback, ordinary Profile tap, identity fallbacks, transition state, Home-root switching, and absence of inactive direct removal or bulk sign-out.
  - Integration: two real sessions on one installation, online/offline cold start, inactive validation, inactive-account count display, push receipt for both accounts, visually identified inactive-account foreground notification, notification open from foreground/background/terminated states, canceled guarded switch, offline sign-out plus shared-token recovery, and one-account expiry.
  - Regression: single-account sign-in, onboarding routing, unsaved-edit protection, OS notification visible copy, notification preferences/new-count seen behavior, API bearer selection, and sign-out cleanup.
- Blocking open questions: None.
