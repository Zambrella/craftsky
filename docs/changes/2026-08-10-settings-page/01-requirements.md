# Requirements: Settings Page And Account Management

## 1. Initial Request

Flesh out the Settings page shown in the supplied iPhone simulator screenshot. Add trailing arrows to rows that open another surface, show the active account's username and handle at the top, place a Switch account row immediately below that identity, add entry points for the existing notification settings and new About and Account pages, move Clear image cache into About, show the app version, and render Sign out in the theme's error colour. The Account page initially contains Delete account; deletion requires confirmation and removes the user's CraftSky membership, private CraftSky data, and every `social.craftsky.*` record from their PDS, but does not delete their AT Protocol/PDS account or non-CraftSky records.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/settings/pages/settings_page.dart` renders the current Settings list.
  - `app/lib/settings/widgets/clear_image_cache_tile.dart` owns the existing cache-clear action and its success/error feedback.
  - `app/lib/settings/widgets/sign_out_tile.dart` signs out only the active retained account.
  - `app/lib/router/router.dart` and `app/lib/router/route_locations.dart` define Settings child routes and the existing `/notifications/settings` route.
  - `app/lib/notifications/pages/notification_settings_page.dart` is the existing notification settings destination.
  - `app/lib/auth/providers/active_account_identity_provider.dart` exposes the active account's profile, including display name and handle.
  - `app/lib/auth/widgets/account_switcher_content.dart`, `app/lib/auth/models/account_switcher_state.dart`, and `app/lib/router/app_shell.dart` implement the retained-account switcher used by compact and large layouts.
  - `app/lib/app_dependencies.dart` exposes `PackageInfo`, including version and build number.
  - `app/lib/router/app_shell.dart` already opens `https://craftsky.social/terms` and `https://craftsky.social/privacy` through the shared external-link path, reports link-launch failures, and formats package metadata through the localized `navigationBuildVersion(version, buildNumber)` label used by the drawer and navigation rail.
  - `appview/internal/index/craftsky_profile.go` defines CraftSky membership by the presence of the user's `social.craftsky.actor.profile` record and runs membership-owned cleanup when that record is deleted.
  - The post, like, repost, and profile indexers already handle delete events idempotently; applicable handlers remove/soft-delete indexed content and retract derived notifications. Most list/search/profile reads are membership-gated, while some direct reads can remain visible until their delete event is processed.
  - `appview/internal/app/instagram_lifecycle.go`, account-owned foreign keys in `appview/migrations/`, and the notification/scheduled-data deletion services provide part of the existing private-data cleanup lifecycle.
  - `appview/internal/auth/handlers_session.go` can revoke one CraftSky session or all CraftSky sessions for a DID, but there is no authenticated account-deletion API or Flutter deletion flow.
  - `lexicon/social/craftsky/` currently defines four PDS record collections: `social.craftsky.actor.profile`, `social.craftsky.feed.post`, `social.craftsky.feed.like`, and `social.craftsky.feed.repost`. Project posts, replies, quotes, and craft-specific project fields are shapes within `social.craftsky.feed.post`, not separate record collections.
- Existing patterns:
  - Settings and notification settings are lifted onto the authenticated shell navigator so compact layouts cover primary navigation while large layouts remain beside the navigation rail.
  - Manual account switching uses a modal bottom sheet on compact layouts and an anchored popover on large layouts; activation is lease-scoped and resets navigation to Home.
  - Sign out removes only the active retained account and activates the most recently used remaining account, if any.
  - External legal links, package version data, cache clearing, error messaging, confirmation dialogs, and error colours already have shared application seams.
  - Clear image cache and Sign out are currently immediate actions without confirmation.
- Current behavior:
  - The repository Settings page starts with Languages and contains Followers, Muted accounts, Blocked accounts, Following, Find people from Instagram, Clear image cache, and Sign out. It does not show the active identity, chevrons, Notifications, Account, or About.
  - The supplied screenshot also contains a Customisation row. That row may be coming from adjacent work not present in this checkout and must be preserved in the target Settings experience.
  - Clear image cache is a top-level action.
  - Sign out uses the default foreground colour.
  - Notification settings is currently reached from Notifications rather than Settings.
  - Terms, Privacy, and build version are already shown in shell navigation but not on an About page.
- Constraints discovered:
  - CraftSky must not delete or attempt to delete the user's DID, PDS account, general AT Protocol identity, or records outside the `social.craftsky.*` namespace.
  - Leaving CraftSky durably requires deletion of the membership-defining `social.craftsky.actor.profile/self` record and every other `social.craftsky.*` record in the user's repo. With the current schemas this includes CraftSky posts/projects/replies/quotes, likes, and reposts.
  - Lexicon files that contain only reusable definitions do not create independent PDS collections. Deletion targets record collections, including future `social.craftsky.*` record collections, rather than trying to delete schema definitions.
  - Deleting CraftSky records removes their blob references, but the PDS controls lifecycle/garbage collection of now-unreferenced blobs; the account-deletion flow must not delete blobs still referenced by non-CraftSky records.
  - Account deletion is broader than sign out: it must remove all current and future CraftSky-namespaced PDS records, clean up private CraftSky state, revoke every ordinary CraftSky session for that DID at job acceptance, clear local product data immediately on the confirming device, and retain only restricted deletion-status state until authoritative success.
  - The active account can change while asynchronous identity, cache, link, sign-out, or deletion work is in flight; completions must not mutate or report success for a different active account.
  - Existing Instagram membership handling retains or inactivates some data according to separate lifecycle/retention behavior. The account-deletion design must explicitly reconcile that behavior with the requested deletion of private CraftSky data.
  - Repository guidance normally prohibits CraftSky-initiated PDS deletion because the PDS is user-owned. The product owner has explicitly approved one narrow exception: after fresh owner reauthentication and exact-handle confirmation, AppView may delete only that DID's `social.craftsky.*` record collections. The implementation change must amend the repository guidance/reference to encode this exception before destructive code lands.
  - Revoking every ordinary CraftSky bearer session does not itself delete the underlying server-side OAuth session. The deletion job must bind the OAuth session created or refreshed by the successful reauthentication before revoking client bearer sessions, and ordinary background writers must not be able to select that job-bound authority.
  - Tap events expose the record URI, DID, collection, action, per-event ID, and repo revision. A deletion-convergence receipt can therefore be recorded only after the existing indexer successfully handles an expected delete event, without adding an eager-hide or second public-deletion path.
- Test/build commands discovered:
  - Focused Flutter tests from `app/`: `flutter test test/settings test/router/notification_settings_route_test.dart test/router/settings_routes_test.dart test/router/app_shell_account_switcher_test.dart`.
  - Focused Flutter analysis from `app/`: `dart analyze lib/settings lib/router lib/auth test/settings test/router`.
  - Flutter generation, if routes/providers/localizations change: `dart run build_runner build` and the repository's localization generation path.
  - AppView verification from the repository root with compose Postgres available: `just test`.

## 3. Clarifying Questions And Decisions

### Q1: What does Delete account delete?

Answer: It deletes the user's CraftSky membership and private CraftSky/AppView data, plus anything in the CraftSky Lexicon from their PDS. It definitely must not delete the user's entire PDS or AT Protocol account, and the confirmation message must explain this.

Decision / implication: Account deletion removes every record in the user's PDS whose collection is a `social.craftsky.*` record Lexicon, including the membership profile, CraftSky posts/projects, likes, and reposts, plus private CraftSky state and CraftSky sessions. It leaves the DID, PDS account, and records from other namespaces intact. The confirmation copy states this boundary before the destructive action can be confirmed.

### Q2: What happens to blobs referenced by deleted CraftSky records?

Answer: Delete the CraftSky records and their references, but do not promise immediate physical blob removal. Leave unreferenced-blob reclamation to the PDS and never delete a blob that is still referenced by retained non-CraftSky records.

Decision / implication: The deletion guarantee is record- and reference-level. PDS garbage collection timing is outside CraftSky's terminal-success promise.

### Q3: How strong must deletion confirmation be?

Answer: Require fresh PDS OAuth reauthentication, show a confirmation naming the active `@handle`, and then require the member to type that handle before submission. CraftSky must never collect a PDS password or emailed confirmation code itself.

Decision / implication: Typed-handle submission is the point of no return. Before submission the member may cancel without mutation; after submission the accepted deletion job cannot be canceled.

### Q4: How is deletion executed and recovered?

Answer: Use a durable, idempotent background deletion job whose state survives app closure and network loss. Revoke normal CraftSky sessions immediately when the job is accepted, give the confirming device only a narrowly scoped deletion-status credential, retry bounded transient failures automatically, and provide a manual Retry action plus support guidance when attention is required.

Decision / implication: The account never returns to ordinary use after job acceptance. The accepted job binds the fresh server-side OAuth session as deletion-only PDS authority before all ordinary CraftSky bearer sessions are revoked. That OAuth session is unavailable to Flutter and ordinary/background product writes, is deleted after the final PDS/convergence gate, and can be replaced only through fresh OAuth reauthentication from the status flow if it expires. The account remains in `Deleting…` or `Deletion needs attention` until terminal success.

### Q5: When may deletion report success?

Answer: Only after every `social.craftsky.*` PDS record is gone, private CraftSky data and sessions are gone, and the existing Tap/indexer pipeline has converged the AppView deletions. Temporary partial visibility during event lag is acceptable; no separate eager-hiding layer is required.

Decision / implication: Before each PDS delete, the job durably records the expected record URI. After the existing indexer successfully processes the matching owner-scoped Tap delete event, the consumer records a receipt containing only job/URI identity plus the Tap event ID and repo revision before acknowledging the event. Convergence means every expected URI has such a receipt, the indexed CraftSky rows and derived notification effects owned by those URIs are absent/retracted, and a final PDS rescan finds no `social.craftsky.*` records. Duplicate and reordered events are idempotent; missing receipts, Tap unavailability, or records found by the final rescan keep the job non-terminal and eventually attention-required under the bounded retry policy. This observation contract adds no eager hiding or duplicate public-deletion logic.

### Q6: What data may remain after deletion?

Answer: Retain only a minimal deletion audit from terminal success until `terminalSuccessAt + 30 days`: DID, deletion job ID, timestamps, and coarse outcome. Do not retain the handle, tokens, content, relationships, preferences, or settings. Explicit account deletion overrides Instagram migration retention, hard-deletes Instagram links/imports/suggestions/verification/private imported data, and releases any username claim.

Decision / implication: The audit expires automatically when the clock reaches `terminalSuccessAt + 30 days`, does not keep the membership active, and does not prevent the same AT account from joining CraftSky again after terminal success.

### Q7: How does deletion behave across devices and retained accounts?

Answer: Immediately erase the confirming device's drafts, staged media, caches, ordinary session, and other account-local data when the job is accepted. Other offline devices erase their local data when they next launch or learn about the deletion; confirmation copy must not promise instantaneous offline-device erasure. If another retained account exists, activate the most recently used one immediately and keep the deleting account as a disabled status row in the switcher. If none remains, show the deletion-status screen directly.

Decision / implication: A pending row reads `Deleting…`; a failed row reads `Deletion needs attention` and opens status/Retry. The row is removed after terminal success.

### Q8: Can the same AT account use CraftSky again?

Answer: Only after terminal deletion success. Authentication while deletion is pending reopens the deletion-status experience. A later rejoin creates a fresh membership and does not restore deleted data.

Decision / implication: The 30-day audit is not a tombstone and must not block re-enrolment.

### Q9: How is active identity presented?

Answer: Show the existing account avatar, display name, and `@handle`. If the display name is absent, show `@handle` once as the primary identity and omit the duplicate secondary line. Use the existing avatar fallback; never display a raw DID or `No username`.

Decision / implication: The identity block materially identifies which account the Settings actions affect without introducing a new identity source.

### Q10: How is the Settings page organized?

Answer: Use titled sections in this order: Preferences (Customisation, Languages, Notifications), Connections (Followers, Following, Muted accounts, Blocked accounts), Discovery (Find people from Instagram), and General (Account, About). Keep Sign out as a separate final action. Switch account remains immediately below the identity block and reuses the existing account-switcher surface.

Decision / implication: Followers and Following are adjacent; maintenance and destructive destinations remain near the bottom.

### Q11: Which rows get trailing icons?

Answer: In-app disclosure rows, including Switch account, use a direction-aware chevron. Terms and Privacy policy use an external-link icon because they open the device's external browser. Clear image cache, Delete account, and Sign out are direct actions without chevrons; app version is read-only without a trailing icon.

Decision / implication: Trailing icons describe the actual interaction rather than decorating every visual row identically.

### Q12: What existing behavior is preserved for Sign out, legal links, cache clearing, and version text?

Answer: Sign out remains immediate with no confirmation and uses the theme error colour. Terms and Privacy use the existing external URL launcher. Clear image cache remains immediate with its existing success/error feedback. About reuses the same localized build label source and formatting already shown in the drawer and navigation rail, for example `1.2.3 (123)`.

Decision / implication: These are relocations or presentation changes, not new behavioral implementations.

## 4. Candidate Approaches

### Option A: Settings hub with reused destinations and account lifecycle

Summary: Make Settings a clear identity-led, sectioned hub, preserve its existing destinations, reuse the existing account-switcher presentation, and add focused About and Account child pages. Add a durable server-owned deletion job that coordinates deletion of all `social.craftsky.*` PDS records, private cleanup, session revocation, and AppView convergence while exposing a restricted status/retry flow.

Pros:

- Matches the requested information hierarchy and keeps direct destructive/maintenance actions off the main Settings page.
- Reuses the tested notification, account-switcher, legal-link, cache, version, session, record-deletion, and membership-lifecycle seams.
- Keeps account deletion authoritative, durable, and retry-safe instead of asking the client to coordinate partial cleanup calls.
- Scales to future Account and About options without expanding the main list indefinitely.

Cons:

- Adds Settings child routes and an authenticated AppView mutation.
- Requires explicit lifecycle coverage across PDS, AppView, durable job/audit state, restricted status credentials, secure local storage, multiple devices, and multiple locally retained accounts.

Risks:

- A partially coordinated deletion could remove only some CraftSky PDS collections, remove access without fully cleaning private data, or clean local state before the server operation is confirmed.

### Option B: Keep one flat page and coordinate deletion in Flutter

Summary: Append the new content and actions to the existing Settings list and have Flutter issue separate membership, cleanup, and logout operations.

Pros:

- Fewer new pages and route definitions.
- Superficially smaller initial UI change.

Cons:

- Conflicts with the requested About and Account pages.
- Leaves a long, mixed-purpose Settings list.
- Duplicates account-switching and lifecycle coordination logic in the client.
- Makes partial failure and safe retry materially harder.

Risks:

- The client can lose connectivity or change active account between destructive steps, leaving ambiguous account state.

## 5. Recommended Direction

Recommended approach: Option A.

Why: The requested page structure is naturally a sectioned Settings hub with nested About and Account destinations. Existing reusable seams already cover every non-destructive behavior. Account deletion is the only new cross-system operation and needs a durable, server-authoritative boundary so app closure, network loss, session revocation, or Tap lag cannot produce false success or restore ordinary account access after the point of no return.

### Approved Architecture Decisions For Account Deletion

- **PDS authority exception:** The product owner's confirmed request approves a narrow exception to the general “do not delete from a user's PDS” rule. It applies only to a freshly reauthenticated owner deleting their own `social.craftsky.*` record collections through this non-cancelable flow. It does not authorize whole-PDS account deletion, other namespaces, direct blob deletion, or a reusable generic PDS-delete facility. The implementation change must update the repository guidance/reference to state this exception before enabling the destructive route.
- **Deletion-only OAuth lease:** Successful fresh reauthentication supplies the exact owner-scoped OAuth session ID that is durably bound to the deletion job before acceptance completes. Acceptance revokes and hard-deletes ordinary CraftSky bearer sessions and deletes other unused OAuth sessions for the DID, but retains this one server-side OAuth session only as deletion-job authority. The worker resumes it directly by job binding; it is excluded from ordinary background-session selection, cannot authorize ordinary APIs, never reaches Flutter, and is deleted after the terminal PDS/convergence checks. If it becomes unusable earlier, the job becomes attention-required and Retry requires another fresh OAuth redirect that replaces only this job binding without restoring ordinary access.
- **Indexer-owned convergence:** Existing indexers remain solely responsible for public AppView deletion and derived-notification retraction. The job records an expected URI before issuing each PDS delete. After an indexer successfully handles a matching owner-scoped Tap delete, the consumer records an idempotent receipt with job ID, URI, collection, Tap event ID, and repo revision before acknowledging the event. Success requires receipts for the complete expected set, absence/retraction of indexed and derived effects for those URIs, and a final empty PDS rescan. A receipt failure causes Tap retry; duplicate/reordered events converge; missing receipts, Tap outage, or newly discovered records keep the job non-terminal and follow bounded retry/attention behavior. Receipts observe existing deletion—they do not hide or purge data themselves.
- **Retention clock:** The 30-day audit period begins at the stored terminal-success timestamp and expires when the injected/server clock reaches `terminalSuccessAt + 30 days`. Non-terminal minimized job, receipt, OAuth-lease, and status state may exist only while needed to complete or recover deletion; after terminal success only the defined audit remains.

## 6. Problem / Opportunity

The current Settings page exposes a growing flat list without identifying which signed-in account is being configured or consistently signalling which rows navigate. Important controls are split across unrelated surfaces, legal/version information is not grouped, and permanent CraftSky membership deletion is unavailable. This creates ambiguity in a multi-account app and makes routine, maintenance, and destructive actions harder to understand.

## 7. Goals

- G-001: Make the active account unmistakable at the top of Settings.
- G-002: Make navigational rows consistent and discoverable.
- G-003: Give Settings a scalable Account and About information architecture.
- G-004: Reuse the existing notification settings and account-switching behavior.
- G-005: Provide an explicit, safe way to delete a CraftSky membership and all CraftSky-Lexicon records without deleting the user's AT Protocol account or other apps' records.
- G-006: Visually distinguish Sign out as a destructive/session-ending action.
- G-007: Make permanent deletion durable and recoverable across app closure, connectivity loss, worker restart, multiple devices, and Tap/indexer lag without restoring ordinary account access.
- G-008: Minimize retained deletion data and communicate the exact PDS, blob, AppView, and offline-device boundaries honestly.

## 8. Non-Goals

- NG-001: Delete, deactivate, or modify the user's DID, PDS account, or general AT Protocol identity.
- NG-002: Delete any PDS record outside the `social.craftsky.*` namespace, including `app.bsky.*` profile, post, follow, like, repost, or block records. All `social.craftsky.*` records are explicitly in deletion scope.
- NG-003: Add account editing, handle changing, password management, data export, active-session management, or a sign-out-all control to the Account page.
- NG-004: Redesign the existing notification preference controls or account switcher.
- NG-005: Change the behavior of Customisation, Languages, relationship lists, moderation lists, or Instagram migration during ordinary use; the explicitly specified permanent-deletion cleanup is the sole Instagram-lifecycle change in scope.
- NG-006: Draft or host new Terms or Privacy content; this change links the existing canonical URLs.
- NG-007: Make app version information interactive or add update checking.
- NG-008: Add a generic master settings toggle or new notification categories.
- NG-009: Delete blobs directly, wait for PDS blob garbage collection, or promise immediate physical blob erasure.
- NG-010: Add an eager AppView hiding/purge layer in parallel with the existing Tap/indexer delete handlers.
- NG-011: Offer deletion-job cancellation, temporary CraftSky deactivation, or restoration after typed-handle submission.
- NG-012: Guarantee immediate erasure of account-local data from a device that is offline and cannot receive the deletion state.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in member | A person using one active CraftSky membership | Identify the active account, find settings, maintain the app, sign out, or delete the CraftSky membership safely. |
| Multi-account member | A person with two or more retained CraftSky accounts on the device | Know which account Settings applies to and open the existing switcher without leaving Settings ambiguously. |
| AppView | CraftSky's authenticated backend and membership authority | Coordinate deletion of all CraftSky PDS records, private-data cleanup, and CraftSky-session revocation without touching other namespaces. |
| User's PDS | The user's independent AT Protocol data server | Delete every authorized `social.craftsky.*` record in the user's repo and retain the account plus records from other namespaces. |

## 10. Current Behavior

Settings is a flat scrollable list. It does not display the active identity or trailing chevrons. The screenshot contains Customisation, Languages, Followers, Muted accounts, Blocked accounts, Following, Find people from Instagram, Clear image cache, and Sign out. The current checkout contains the same list except Customisation. Notification settings exists elsewhere; Terms, Privacy, and the build version appear in shell navigation; the account switcher is opened from profile navigation. There is no Account page or CraftSky membership-deletion operation.

## 11. Desired Behavior

Settings opens with the active account's avatar, display name, and `@handle`, followed immediately by Switch account. A missing display name falls back to one non-duplicated `@handle`, never a DID. The main rows are arranged into Preferences, Connections, Discovery, and General sections in the agreed order, with Sign out separated at the bottom. In-app disclosure rows use direction-aware chevrons; external legal links use an external-link icon; direct actions and read-only information do not use misleading chevrons. Notifications opens the existing notification settings. Account contains Delete account. About contains externally opened Terms and Privacy links, immediate Clear image cache, and the same read-only version/build label used by shell navigation. Sign out remains immediate and uses the error colour.

Delete account first requires fresh PDS OAuth reauthentication, then a two-step confirmation that names the active `@handle` and requires the member to type it. Accepted submission is the non-cancelable point of no return. It creates a durable deletion job for only that CraftSky account, binds the fresh server OAuth session as deletion-only authority, immediately revokes ordinary CraftSky sessions, clears local data on the confirming device, and permits the device only deletion status/reauthentication/retry access. The job deletes every `social.craftsky.*` PDS record, including membership/profile, posts and project posts, replies and quotes, likes, and reposts; hard-deletes private CraftSky and Instagram-migration data; releases username claims; and waits for expected Tap deletes to be indexed and receipted. It does not delete the AT Protocol/PDS account, non-CraftSky records, or shared blobs, and does not promise immediate PDS garbage collection or instantaneous erasure from offline devices.

While deletion runs, the account is non-usable and represented as `Deleting…`; failures use bounded automatic retry, then `Deletion needs attention`, manual Retry, and support guidance without restoring ordinary access. An expired deletion OAuth session requires fresh reauthentication from status before Retry. Terminal success occurs only after the final PDS rescan, private-data/session cleanup, indexed-effect/receipt convergence, and deletion OAuth removal checks pass. A remaining local account becomes active immediately; otherwise the confirming device shows deletion status. The deleted account's switcher row disappears after success. The same AT account may rejoin only after terminal success as a fresh membership with no restoration. A minimal DID/job/timestamp/outcome audit remains until `terminalSuccessAt + 30 days` and then expires.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Settings shall identify the active CraftSky account before presenting account-scoped options. | Prevents changes being made under the wrong identity in a multi-account app. | Prompt, codebase | AC-001, AC-002 |
| BR-002 | Business | Must | Settings shall provide discoverable entry points for Notifications, Account, and About while preserving the existing settings destinations shown in the supplied screenshot. | Completes the requested Settings information architecture without regressing existing access. | Prompt, screenshot | AC-003, AC-004, AC-005 |
| BR-003 | Business | Must | A member shall be able to permanently delete their current CraftSky membership, every `social.craftsky.*` record in their PDS repo, and private CraftSky data without deleting their AT Protocol/PDS account or non-CraftSky records. | Provides user control while respecting federated data ownership. | User answer, architecture | AC-015, AC-016, AC-017 |
| BR-004 | Business | Must | The product shall clearly distinguish navigation, maintenance actions, session sign-out, and permanent membership deletion. | Reduces accidental destructive actions and misleading affordances. | Prompt, discovery | AC-003, AC-010, AC-012, AC-014 |
| FR-001 | Functional | Must | The Settings identity header shall display the active account's existing avatar, display name, and current handle in `@handle` form. If display name is unavailable, it shall show `@handle` once as the primary identity and omit the duplicate secondary line; the avatar shall use its existing fallback and the header shall never show a raw DID or `No username`. | Makes the active account recognizable and provides a non-sensitive fallback. | Prompt, grilling, codebase | AC-001, AC-020, AC-031 |
| FR-002 | Functional | Must | A Switch account row with a trailing direction-aware chevron shall appear immediately below the displayed identity and open the existing account-switcher presentation for the current form factor. | Makes switching discoverable without creating a second switching flow. | Prompt, grilling, codebase | AC-002, AC-021 |
| FR-003 | Functional | Must | The main Settings page shall use titled sections in this order: Preferences (Customisation, Languages, Notifications), Connections (Followers, Following, Muted accounts, Blocked accounts), Discovery (Find people from Instagram), and General (Account, About), followed by a separately presented Sign out action. | Preserves every requested destination while making the expanded page scannable. | Prompt, screenshot, grilling | AC-003, AC-004, AC-005, AC-032 |
| FR-004 | Functional | Must | Every enabled row that opens another in-app surface shall show a trailing directional chevron appropriate to the current text direction; Terms and Privacy policy shall instead show an external-link icon; direct actions and read-only information shall show neither. | Signals the actual interaction and supports RTL layouts. | Prompt, grilling | AC-003, AC-007, AC-022, AC-033 |
| FR-005 | Functional | Must | Each Settings child page shall use the established authenticated-shell navigation behavior, and Back shall return to the immediately preceding Settings surface without changing the active account. | Preserves responsive navigation and predictable Back behavior. | Codebase | AC-006, AC-021 |
| FR-006 | Functional | Must | The Notifications row shall open the existing notification settings page and preserve its current account-wide preferences, permission guidance, retry, and error behavior. | Avoids duplicating or changing an implemented settings feature. | Prompt, codebase | AC-004, AC-006 |
| FR-007 | Functional | Must | The About row shall open a new About page containing Terms, Privacy policy, Clear image cache, and app version information. | Groups legal, maintenance, and application metadata as requested. | Prompt | AC-005, AC-007 |
| FR-008 | Functional | Must | Terms shall open `https://craftsky.social/terms` in the device's external browser through the existing external-link behavior. | Reuses the canonical Terms destination without adding an in-app browser. | Prompt, grilling, codebase | AC-008, AC-019 |
| FR-009 | Functional | Must | Privacy policy shall open `https://craftsky.social/privacy` in the device's external browser through the existing external-link behavior. | Reuses the canonical Privacy destination without adding an in-app browser. | Prompt, grilling, codebase | AC-008, AC-019 |
| FR-010 | Functional | Must | Clear image cache shall be removed from the main Settings page and exposed on About as an immediate action with no confirmation, preserving its existing busy state, cache-clearing scope, success message, and mapped error behavior. | Moves rather than duplicates a reversible maintenance action. | Prompt, grilling, codebase | AC-007, AC-009, AC-019 |
| FR-011 | Functional | Must | About shall display the installed app version and build number as read-only information by reusing the same localized source and formatting already used by the drawer and navigation rail, for example `1.2.3 (123)`. | Provides accurate, consistent support information without a new data source or formatter. | Prompt, grilling, codebase | AC-010, AC-020, AC-034 |
| FR-012 | Functional | Must | The Account row shall open a new Account page whose only initial option is Delete account. | Provides the requested extensible destination with deliberately narrow initial scope. | Prompt | AC-005, AC-011 |
| FR-013 | Functional | Must | Selecting Delete account shall require fresh PDS OAuth reauthentication and then show a two-step confirmation: first a destructive explanation naming the active `@handle`, then a field requiring the member to type that exact handle before submission. No deletion job shall be accepted before both steps complete. | Reduces wrong-account deletion without collecting PDS credentials directly. | Prompt, grilling | AC-012, AC-013, AC-035 |
| FR-014 | Functional | Must | The confirmation shall explain that deletion permanently removes the named account's CraftSky membership, private CraftSky data, and all CraftSky records from its PDS; signs it out of CraftSky on every device; cannot be recovered; does not delete the AT Protocol/PDS account or records from other apps; does not promise immediate PDS blob reclamation; and removes local data from offline devices when they next reconnect. | Makes the complete agreed deletion boundary explicit without over-promising physical or offline-device erasure. | User answers, grilling | AC-012, AC-036 |
| FR-015 | Functional | Must | An accepted account-deletion job shall enumerate and delete every record in the authenticated DID's PDS repo whose collection is a record Lexicon under `social.craftsky.*`, remove that DID's private CraftSky/AppView data except the defined non-terminal operational state and 30-day deletion audit, and revoke then remove all ordinary CraftSky sessions and notification subscriptions for that DID. | Removes all CraftSky-owned public and private product data while retaining the AT Protocol account and other namespaces. | User answers, architecture, codebase, lexicon inventory | AC-015, AC-016, AC-017, AC-023, AC-037 |
| FR-016 | Functional | Must | When deletion is accepted, the most recently used remaining retained account shall become active immediately if one exists; the deleting account shall remain as a disabled `Deleting…` row in the account switcher that opens deletion status, change to `Deletion needs attention` with Retry when required, and be removed only after terminal success. If no other account remains, the confirming device shall show deletion status directly. | Keeps the deleting account non-usable while preserving progress and multi-account continuity. | Grilling, codebase | AC-018, AC-038 |
| FR-017 | Functional | Must | Before point-of-no-return submission, cancellation shall cause no mutation. After job acceptance, failure shall never restore ordinary account access: bounded transient retries run automatically, and exhausted or non-transient failure shall show `Deletion needs attention`, a manual Retry action, and support guidance. | Prevents ambiguous recovery or accidental reactivation after destructive work begins. | Grilling | AC-013, AC-018, AC-023, AC-039 |
| FR-018 | Functional | Must | Sign out shall remain an immediate current-session/account action with no confirmation, shall not perform membership deletion, and shall render its icon and label using the theme's error colour. | Preserves existing reversible sign-out semantics while visually distinguishing it. | Prompt, grilling, codebase | AC-014, AC-024 |
| FR-019 | Functional | Must | Fresh PDS OAuth reauthentication shall use the established OAuth redirect flow immediately before destructive confirmation; CraftSky shall not request, render, transmit, or store the member's PDS password, email confirmation code, or equivalent credential. | Preserves the token-mediating security boundary. | Grilling, architecture | AC-035, AC-040 |
| FR-020 | Functional | Must | Typed-handle submission shall create or resume one durable, idempotent deletion job whose state survives app closure, process restart, network loss, duplicate submission, and client reconnection. Before acceptance completes, the job shall durably bind the exact fresh owner OAuth session used for deletion-only PDS authority. | Cross-system deletion cannot depend on one foreground request or on an ordinary client bearer session that is revoked at acceptance. | Grilling, architecture review | AC-023, AC-037, AC-041 |
| FR-021 | Functional | Must | Job acceptance shall immediately revoke and remove all ordinary CraftSky bearer sessions for the DID and remove any server OAuth session not bound to the deletion job. The confirming device may retain only a separate narrowly scoped, revocable credential that can read deletion status, begin fresh reauthentication when required, and request Retry for that job; it cannot access ordinary CraftSky APIs, PDS APIs, or another account's job. The job-bound OAuth session shall remain server-only, deletion-only, unavailable to ordinary background selectors, replaceable only by fresh owner reauthentication, and removed after the final PDS/convergence gate. | Prevents use of an account undergoing irreversible deletion while preserving the minimum PDS authority and recovery visibility required to finish. | Grilling, architecture review, OAuth BFF | AC-016, AC-040, AC-042 |
| FR-022 | Functional | Must | On job acceptance, the confirming device shall erase the deleting account's drafts, staged media, caches, ordinary session, and other device-local product data, retaining only the minimal job binding and display identity needed for deletion status until terminal success. Other devices shall perform the same cleanup when they next launch or otherwise learn deletion was accepted. | Removes accessible local product data while preserving agreed status visibility and acknowledging offline-device limits. | Grilling | AC-036, AC-043 |
| FR-023 | Functional | Must | The deletion job shall rely on the existing idempotent Tap/indexer delete handlers to remove indexed public AppView data. Before each PDS delete it shall durably register the expected URI; after the matching owner-scoped Tap delete is successfully indexed, an observation layer shall durably record the job/URI/collection, Tap event ID, and repo revision before acknowledgement. AppView convergence passes only when every expected URI has a receipt, the indexed rows and derived notification effects for those URIs are absent/retracted, and a final PDS rescan finds no `social.craftsky.*` records. Receipt failure shall retry the Tap event; duplicate/reordered events shall be idempotent; missing receipts, Tap unavailability, or newly discovered records shall keep the job non-terminal under bounded retry/attention behavior. No additional eager-hide or duplicate eager-purge layer is permitted. | Defines observable terminal success while leaving deletion and retraction in the existing indexers. | Grilling, codebase, architecture review | AC-044 |
| FR-024 | Functional | Must | Authentication with the same AT account while deletion is pending shall open only the deletion-status experience. After terminal success, the same account may join CraftSky again as a fresh membership, and no deleted content, relationships, preferences, imports, or settings shall be restored. | Defines a safe lifecycle without permanently banning the independent AT identity. | Grilling | AC-045 |
| FR-025 | Functional | Must | CraftSky shall retain a deletion audit from the stored terminal-success timestamp until, but not beyond, `terminalSuccessAt + 30 days`, containing only DID, deletion job ID, timestamps, and coarse outcome, then automatically expire it at that boundary. The audit shall contain no handle, token, content, relationship, preference, import, or settings data and shall not keep membership active or block rejoining. | Provides minimal operational evidence with a precise bounded privacy impact. | Grilling, document review | AC-046 |
| FR-026 | Functional | Must | Explicit account deletion shall hard-delete all account-owned Instagram migration data, including links, imports, suggestions, verification state, and private imported data, and shall release any username claim regardless of ordinary migration retention behavior. | A deliberate account deletion must override product-level migration retention. | Grilling, codebase | AC-047 |
| FR-027 | Functional | Must | Deletion status shall use phase-level progress rather than record counts and shall distinguish active deletion, waiting for AppView convergence, bounded automatic retry, attention required, and terminal success without exposing private record details. | Gives useful progress despite eventual event processing and changing collection sizes. | Grilling | AC-038, AC-039, AC-044 |
| NFR-001 | Non-functional | Must | Identity loading and every asynchronous action shall be fenced to the active account lease so stale completions cannot update, delete, navigate, or report success for another account. | Account changes can occur while work is in flight. | Codebase, discovery | AC-021, AC-025 |
| NFR-002 | Non-functional | Must | The account-deletion service and job shall be authenticated, owner-scoped, durable, idempotent or safely convergent on retry across pagination/batches and worker restarts, and shall use the standard `/v1/*` camelCase error envelope. | A destructive cross-system mutation must be secure and recoverable even when a repo contains many CraftSky records. | API architecture, grilling | AC-017, AC-023, AC-026, AC-041 |
| NFR-003 | Non-functional | Must | New and changed controls shall expose correct button/link/selected/destructive semantics, readable labels, keyboard focus behavior, and at least a 48x48 logical-pixel target. | Maintains accessibility across mobile and large layouts. | Design conventions | AC-022, AC-027 |
| NFR-004 | Non-functional | Must | New visible copy, including Settings labels, deletion confirmation, errors, and accessibility labels, shall be localizable. | Settings is user-facing application chrome. | Codebase convention | AC-028 |
| NFR-005 | Non-functional | Should | Settings, About, Account, and confirmation content should preserve the existing CraftSky theme, spacing, maximum content width, and compact/large responsive behavior. | Keeps the new surfaces visually coherent with existing settings pages. | Screenshot, codebase | AC-029 |
| NFR-006 | Non-functional | Must | Deletion job, audit, status-credential, retry, and observability data shall be minimized, access-controlled, and free of record content or unrelated account data. | Deletion infrastructure must not become a secondary store of deleted private data. | Grilling, privacy | AC-040, AC-042, AC-046 |
| RULE-001 | Business rule | Must | Delete account always applies to the account identified by the lease captured when the user confirms; it shall never follow a later active-account change. | Prevents cross-account deletion. | Discovery | AC-025 |
| RULE-002 | Business rule | Must | CraftSky account deletion shall not call an AT Protocol account-deletion API or delete any PDS record whose collection is outside the `social.craftsky.*` record namespace. | Enforces the user-confirmed federated ownership boundary. | User answer, architecture | AC-015, AC-017 |
| RULE-003 | Business rule | Must | A trailing chevron denotes disclosure to another in-app surface, including the existing account switcher; external-browser rows use an external-link icon; direct actions and read-only information, including Sign out, Delete account, Clear image cache, and version, display neither. | Keeps the requested affordances semantically accurate. | Prompt, grilling | AC-003, AC-007, AC-010, AC-011, AC-014, AC-033 |
| RULE-004 | Business rule | Must | Sign out and Delete account are distinct actions: sign out preserves membership, `social.craftsky.*` PDS records, and private CraftSky data, while Delete account removes them and revokes CraftSky sessions on every device. | Prevents a destructive behavior change to the existing sign-out control. | Prompt, user answer, codebase | AC-014, AC-016, AC-024 |
| RULE-005 | Business rule | Must | The deletion collection inventory shall include all current and future Lexicons under `social.craftsky.*` whose primary definition is a PDS `record`; referenced object/defs Lexicons are not independent collections. | Prevents new CraftSky record types from escaping account deletion while avoiding meaningless schema-def deletion attempts. | User clarification, lexicon inventory | AC-015, AC-030 |
| RULE-006 | Business rule | Must | Typed-handle submission is the point of no return; an accepted deletion job cannot be canceled, paused, or converted back into an ordinary active membership. | Later cleanup stages may already have irreversibly deleted data. | Grilling | AC-039 |
| RULE-007 | Business rule | Must | Terminal deletion success requires a final empty rescan of all `social.craftsky.*` PDS record collections, completion of private-data cleanup and ordinary-session revocation, an indexer receipt plus absent/retracted indexed effects for every job-expected URI, and removal of the deletion-only OAuth session. | Prevents partial cleanup, event lag, or retained PDS authority from being reported as completion. | Grilling, architecture review | AC-037, AC-044 |
| RULE-008 | Business rule | Must | Temporary partial public visibility during Tap/indexer lag is acceptable while status remains `Deleting…`; CraftSky shall not add a separate immediate-hiding layer for this change. | Avoids duplicating existing indexer deletion behavior and complexity. | Grilling, codebase | AC-044 |
| RULE-009 | Business rule | Must | CraftSky deletes records and their blob references but does not promise immediate physical blob deletion, directly delete shared blobs, or include PDS garbage-collection completion in terminal success. | Blob lifecycle belongs to the PDS and blobs may be shared across namespaces. | Grilling, architecture | AC-036, AC-048 |
| RULE-010 | Business rule | Must | While an accepted deletion job is non-terminal, only minimized job, expected-URI/receipt, deletion-only OAuth, and status state needed to complete or recover it may remain. At terminal success that operational state and all other ordinary product data shall be removed so only the 30-day minimal deletion audit remains. | Matches the member's deliberate permanent-deletion intent without deleting state required to finish an in-progress job. | Grilling, document review | AC-046, AC-047 |
| RULE-011 | Business rule | Must | Rejoining with the same AT identity is permitted only after terminal deletion success and always creates a fresh CraftSky membership without restoration. | Separates deletion-job recovery from later voluntary re-enrolment. | Grilling | AC-045 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given a signed-in member opens Settings, when active identity data is available, then the top of the page shows that account's avatar, display name, and `@handle` before any settings rows. |
| AC-002 | BR-001, FR-002 | Given Settings is open, when the identity header renders, then Switch account is the first row immediately beneath it and opens the established account switcher. |
| AC-003 | BR-002, BR-004, FR-003, FR-004, RULE-003 | Given main Settings is rendered, then all preserved and newly added in-app disclosure rows are present with one trailing directional chevron, while direct actions do not show a trailing navigation icon. |
| AC-004 | BR-002, FR-003, FR-006 | Given the member taps Notifications in Settings, then the existing notification settings page opens with its current controls and behavior intact. |
| AC-005 | BR-002, FR-003, FR-007, FR-012 | Given Settings is rendered, then distinct Account and About rows are present and each opens its corresponding new child page. |
| AC-006 | FR-005, FR-006 | Given a Settings child page is open on compact or large layout, when Back is invoked, then the user returns to the preceding Settings surface with the same active account and correct shell selection. |
| AC-007 | FR-004, FR-007, FR-010, RULE-003 | Given About is open, then Terms, Privacy policy, Clear image cache, and version information are present; both legal links show an external-link icon, Clear image cache and version show no trailing navigation icon, and Clear image cache is absent from main Settings. |
| AC-008 | FR-008, FR-009 | Given About is open, when Terms or Privacy policy is selected, then the corresponding canonical HTTPS URL is passed to the shared launcher and opens in the device's external browser. |
| AC-009 | FR-010 | Given About is open, when Clear image cache is selected, then no confirmation is shown, both existing image-cache scopes are cleared once, re-entry is disabled while busy, and the existing success or mapped error feedback is shown. |
| AC-010 | BR-004, FR-011, RULE-003 | Given About is open, then the installed version and build number use the same localized label shown in the drawer/navigation rail, such as `1.2.3 (123)`, are read-only, and are not presented as a navigational row. |
| AC-011 | FR-012, RULE-003 | Given Account is open, then Delete account is the only account option, is visibly destructive, and has no page chevron. |
| AC-012 | BR-004, FR-013, FR-014 | Given the member selects Delete account, then no mutation occurs until fresh PDS reauthentication succeeds, the first confirmation names the active `@handle` and displays the full deletion boundary, and the member types that exact handle in the second step. |
| AC-013 | FR-013, FR-017 | Given either deletion-confirmation step is open before typed-handle submission, when the member selects Cancel or dismisses it, then no deletion job is accepted and the current account remains active. |
| AC-014 | BR-004, FR-018, RULE-003, RULE-004 | Given main Settings is rendered, then Sign out's icon and label use the current theme error colour, it has no chevron, and activating it immediately follows the existing active-account sign-out behavior without another confirmation. |
| AC-015 | BR-003, FR-015, RULE-002, RULE-005 | Given a deletion job reaches terminal success, then no record remains in the user's repo under any current `social.craftsky.*` record collection, while the DID/PDS account and records under every other namespace remain intact. |
| AC-016 | BR-003, FR-015, FR-016, RULE-004 | Given deletion succeeds, then private account-owned CraftSky state and all CraftSky sessions/subscriptions for that DID are no longer usable on any device. |
| AC-017 | BR-003, FR-015, RULE-002, NFR-002 | Given the deletion request is authenticated as one DID, then it cannot delete another DID's membership or private state and does not invoke whole-PDS/AT-account deletion. |
| AC-018 | FR-016, FR-017 | Given a deletion job is accepted, then any most-recently-used remaining account becomes active immediately; otherwise deletion status is shown. The deleting account cannot be activated for ordinary use, remains available only as a status row until success, and exposes attention/Retry state after exhausted automatic retries. |
| AC-019 | FR-008, FR-009, FR-010 | Given an external link or cache action fails, then the user receives the existing safe, localized error treatment without losing navigation or changing active account. |
| AC-020 | FR-001, FR-011 | Given display name is unavailable, then the active `@handle` appears once as primary identity with the avatar fallback and no secondary duplicate, raw DID, or `No username`; given package metadata is incomplete, About degrades without malformed punctuation or a crash. |
| AC-021 | FR-002, FR-005, NFR-001 | Given the member opens the switcher from Settings and activates another retained account, then activation uses the existing unsaved-work guard and lease boundary, closes the old Settings flow, and lands on the selected account's Home page. |
| AC-022 | FR-004, NFR-003 | Given left-to-right or right-to-left text direction, then page chevrons point in the forward direction and each row exposes an accessible label, role, focus target, and minimum touch target. |
| AC-023 | FR-015, FR-017, FR-020, NFR-002 | Given a deletion phase is retried or a dependency fails, then the same owner-scoped durable job converges safely, transitions to an actionable status when bounded retries are exhausted, never reports partial cleanup as success, and does not affect another DID. |
| AC-024 | FR-018, RULE-004 | Given the member signs out instead of deleting, then CraftSky membership, all `social.craftsky.*` PDS records, and private account data remain and only the established active-account session lifecycle runs. |
| AC-025 | NFR-001, RULE-001 | Given account A confirms deletion and account B becomes active before the response completes, then the completion cannot delete, remove, navigate, or show success as account B; the operation remains bound to A. |
| AC-026 | NFR-002 | Given an unauthenticated, malformed, or unauthorized account-deletion request, then AppView rejects it with the standard `/v1/*` status and camelCase error envelope without performing cleanup. |
| AC-027 | NFR-003 | Given keyboard, assistive technology, or touch input, then Switch account, page destinations, legal links, cache clearing, Sign out, typed-handle confirmation, deletion status/Retry, and Delete account can each be identified and operated with correct semantics. |
| AC-028 | NFR-004 | Given a supported locale is active, then every new visible label, confirmation sentence, error, and semantic label resolves through localization resources rather than a production hard-coded string. |
| AC-029 | NFR-005 | Given compact mobile, tablet, or desktop layout, then Settings and its new child pages remain scrollable, avoid clipped content, preserve the established content-width behavior, and use the CraftSky theme and spacing system. |
| AC-030 | RULE-005 | Given the CraftSky lexicon inventory contains profile, post, like, repost, referenced defs, and any future schemas, then account deletion targets every schema whose primary definition is `record` and does not treat referenced defs-only schemas as repo collections. |
| AC-031 | FR-001 | Given the identity header renders with an available avatar or fallback, then it reuses the established account-avatar presentation and identifies the same account as the displayed handle. |
| AC-032 | FR-003 | Given main Settings renders, then its titled sections and rows appear in the exact agreed order, Followers is adjacent to Following, and Sign out is separated after General. |
| AC-033 | FR-004, RULE-003 | Given Settings or About renders, then in-app disclosure rows including Switch account show direction-aware chevrons, Terms and Privacy show external-link icons, and direct/read-only rows show neither. |
| AC-034 | FR-011 | Given a build label is visible simultaneously in About and shell navigation, then both are produced from the same package metadata and localized formatter and display the same version/build value. |
| AC-035 | FR-013, FR-019 | Given fresh PDS OAuth reauthentication is canceled or fails, then typed-handle confirmation is unavailable and no deletion job is accepted; given it succeeds, the exact active `@handle` is required before submission is enabled. |
| AC-036 | FR-014, FR-022, RULE-009 | Given destructive confirmation is displayed, then it accurately states the AT-account/non-CraftSky preservation boundary, unrecoverability, PDS blob-GC limitation, and next-contact erasure behavior for offline devices without claiming instantaneous erasure everywhere. |
| AC-037 | FR-015, FR-020, RULE-007 | Given a job is accepted, then it cannot reach terminal success until a final PDS rescan is empty, all required private stores and ordinary sessions are cleared, every expected delete has an indexed receipt with absent/retracted indexed effects, and the deletion-only OAuth session is removed. |
| AC-038 | FR-016, FR-027 | Given deletion is pending with other accounts retained, then the deleting account appears disabled as `Deleting…` and opens phase-level job status; with no other account, status is the primary signed-out experience; after terminal success, the pending row disappears. |
| AC-039 | FR-017, FR-027, RULE-006 | Given typed-handle submission has been accepted, then no cancellation control is available, ordinary access cannot be restored, transient failures receive bounded automatic retries, and an unresolved failure becomes `Deletion needs attention` with manual Retry and support guidance. |
| AC-040 | FR-019, FR-021, NFR-006 | Given deletion reauthentication and job acceptance, then CraftSky uses OAuth redirects and server-held tokens, never collects a PDS password/email code, no ordinary session remains usable, and only the job-bound server OAuth session can authorize deletion PDS calls until it is replaced or removed. |
| AC-041 | FR-020, NFR-002 | Given the app closes, the service restarts, connectivity is lost, or the submission is duplicated after acceptance, then reconnecting resolves the same deletion job and its bound deletion-only OAuth authority, and resumes or reports durable state without duplicating destructive work. |
| AC-042 | FR-021, NFR-006 | Given the confirming device holds deletion-status access, then it can read only its job's status, begin fresh OAuth reauthentication when the bound authority is unusable, and request Retry; it cannot call ordinary CraftSky or PDS endpoints, inspect another job, or receive the server OAuth session, and loses access when revoked or no longer needed. |
| AC-043 | FR-022 | Given job acceptance on the confirming device, then that account's drafts, staged media, caches, ordinary session, and other local product data are erased immediately except the minimal deletion-status binding/identity; given another device was offline, it performs equivalent cleanup on next launch/contact; after success, the remaining status data is erased. |
| AC-044 | FR-023, FR-027, RULE-007, RULE-008 | Given expected PDS deletions have occurred but Tap/indexer events or receipts are pending, then status remains in a deletion/convergence phase, temporary partial AppView visibility is tolerated, existing idempotent indexers process the events, receipts are stored only after successful handling and before acknowledgement, and terminal success waits for receipts plus absent/retracted indexed effects and a final empty PDS rescan without an eager-hide layer. |
| AC-045 | FR-024, RULE-011 | Given the same AT identity authenticates while deletion is pending, then only deletion status is available; given terminal success has occurred, normal onboarding may create a fresh membership and no deleted data is restored. |
| AC-046 | FR-025, NFR-006, RULE-010 | Given deletion data is inspected after terminal success, then only DID, job ID, timestamps, and coarse outcome exist in the deletion audit; no prohibited product or operational deletion data is present, the audit does not block rejoin, it exists immediately before `terminalSuccessAt + 30 days`, and it is absent at and after that boundary. |
| AC-047 | FR-026, RULE-010 | Given an account with Instagram migration data and a username claim reaches terminal deletion success, then links, imports, suggestions, verification/private imported data, and the claim are gone regardless of ordinary migration retention behavior. |
| AC-048 | RULE-009 | Given deleted CraftSky records referenced blobs, then CraftSky does not directly delete a blob still referenced by retained records and does not wait for PDS garbage collection before terminal success. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Profile display name is null, blank, or still loading. | Show the active `@handle` once as primary identity with the existing avatar fallback and no secondary duplicate; never show a blank header, raw DID, bearer token, `No username`, or another account's cached name. | FR-001, NFR-001 |
| EC-002 | The active account changes while its profile identity is loading. | Discard the stale identity completion and render only the new lease's identity. | FR-001, NFR-001 |
| EC-003 | Only one account is retained. | Switch account still opens the existing switcher so Add account remains discoverable; the current row remains selected. | FR-002 |
| EC-004 | Account limit is reached. | The reused switcher preserves its existing disabled Add account state and helper text. | FR-002 |
| EC-005 | External URL launcher returns false or throws. | Stay on About and show the shared safe link-open error. | FR-008, FR-009 |
| EC-006 | Clear image cache is tapped repeatedly. | Only one clear operation runs while the action is busy. | FR-010 |
| EC-007 | Package build number is empty on a supported target. | Show the available version without malformed punctuation; do not make the row interactive. | FR-011 |
| EC-008 | Delete confirmation is canceled or dismissed. | Make no server or local mutation. | FR-013 |
| EC-009 | Deletion loses network connectivity, AppView fails, or the job-bound OAuth session becomes unusable after acceptance. | Preserve the durable job and non-usable status account, retry within bounds, show attention/Retry when necessary, require replacement fresh OAuth reauthentication for an unusable PDS grant, and do not claim deletion succeeded or restore ordinary access. | FR-017, FR-020, FR-021, NFR-002 |
| EC-010 | Some or all CraftSky records are already absent during a retried deletion. | Treat missing records idempotently and converge deletion of every remaining `social.craftsky.*` record plus private cleanup/session revocation without touching other namespaces. | FR-015, NFR-002, RULE-002 |
| EC-011 | Private cleanup succeeds for some stores but fails for another. | Do not return success or restore ordinary access; retries must be safe, durable, owner-scoped, and resumable from completed phases. | FR-015, FR-017, FR-020, NFR-002 |
| EC-012 | Another local account remains when deletion is accepted. | Clear the deleting account's local data, activate the most recently used retained account immediately, and keep only a disabled deletion-status row for the deleting account until success. | FR-016, FR-022 |
| EC-013 | No other local account remains when deletion is accepted. | Clear ordinary local account data and session, retain only restricted deletion-status access, and show the status screen directly. | FR-016, FR-021, FR-022 |
| EC-014 | Account A confirms deletion and account B is activated while A's request is in flight. | Keep the request and cleanup scoped to A; never delete or report the result as B. | NFR-001, RULE-001 |
| EC-015 | The user authenticates with the same still-existing AT Protocol account. | While deletion is pending, reopen only deletion status; after terminal success, allow normal join/onboarding as a fresh membership without restoring deleted data. | BR-003, FR-024, RULE-011 |
| EC-016 | The screenshot's Customisation route is merged from adjacent work after this document. | Preserve that destination and apply the same main-page chevron and navigation rules without redefining its feature behavior. | FR-003, FR-004 |
| EC-017 | The user has more CraftSky records than one PDS list/delete batch can hold. | Page through each CraftSky record collection and safely continue/retry batches until every record is deleted before reporting terminal success. | FR-015, NFR-002 |
| EC-018 | A new `social.craftsky.*` record Lexicon is added after this feature ships. | The maintained deletion inventory/test fails until the new record collection is included; successful deletion cannot silently omit it. | RULE-005 |
| EC-019 | A CraftSky image blob is referenced by a CraftSky record and by a non-CraftSky record. | Delete the CraftSky record/reference but do not directly delete a blob still referenced outside CraftSky; leave blob lifecycle to supported PDS behavior. | FR-015, RULE-002 |
| EC-020 | Fresh PDS OAuth reauthentication is canceled, expires, or returns for a different account. | Do not show an enabled typed-handle submission or accept a job; keep Settings scoped to the original active-account lease. | FR-013, FR-019, NFR-001 |
| EC-021 | The typed handle differs by any character from the active handle named in confirmation. | Keep submission disabled or reject it without mutation; do not accept aliases, display names, DIDs, or another retained account's handle. | FR-013 |
| EC-022 | The app closes immediately after typed-handle submission and before receiving the response. | On reconnection, resolve the accepted owner-scoped job and its deletion-only OAuth binding idempotently and show status; never create a second job or restore ordinary access. | FR-020, FR-021, RULE-006 |
| EC-023 | The PDS deletion completes but Tap/AppView is delayed, a receipt write fails, events are duplicated/reordered, or Tap is temporarily unavailable. | Remain in the convergence phase and allow temporary partial indexed visibility; acknowledge an expected delete only after both indexer handling and its idempotent receipt succeed; retry without eager purge logic; require all receipts, absent/retracted effects, and a final empty PDS rescan before success. | FR-023, FR-027, RULE-007, RULE-008 |
| EC-024 | An offline secondary device still has local drafts or cached content. | It cannot be promised immediate erasure; on next launch/contact it learns deletion state, blocks ordinary access, and erases that account's local data. | FR-014, FR-022 |
| EC-025 | The clock reaches `terminalSuccessAt + 30 days` while the same AT identity has already rejoined. | Expire the old audit exactly at the boundary without affecting the fresh membership or restoring/altering any product data. | FR-024, FR-025 |

## 15. Data / Persistence Impact

- New fields: None identified for Settings, About, or Account UI. Durable server-side deletion-job state, retry/phase state, restricted status-credential state, and expiring deletion-audit state are required; exact schemas are coding-design decisions.
- Changed fields: None identified.
- Deleted data on confirmed account deletion:
  - Every PDS record in a `social.craftsky.*` record collection. The current inventory is `social.craftsky.actor.profile`, `social.craftsky.feed.post` (including standard posts, projects, replies, and quotes), `social.craftsky.feed.like`, and `social.craftsky.feed.repost`.
  - All ordinary CraftSky sessions and notification subscriptions for the deleted DID, immediately upon job acceptance.
  - Account-owned private AppView state, including applicable notification state/preferences, saves/folders, mutes, pins, scheduled posts/private media, language preferences, customisation, and every other private membership-owned record.
    - The current mandatory server manifest covers auth/session state (`craftsky_sessions`, ordinary/unbound `oauth_sessions`, and deletion-related OAuth request state); private activity (`craftsky_recent_searches`, `actor_mutes`, `saved_post_folders`, `saved_posts`, `account_language_preferences`, and `profile_pins`); notification/device state (`notification_events` received by the DID, `notification_preferences`, `notification_seen_state`, `push_account_subscriptions`, dependent `push_deliveries`, and an installation only when no other account still uses it); scheduled publication state (`scheduled_posts`, `scheduled_post_media`, `scheduled_post_publication_tombstones`, associated object-store media, and cleanup jobs for those object keys); and moderation rows whose reporter, source, or subject DID is the deleting DID (`moderation_reports`, `moderation_outputs`).
    - Public indexed CraftSky tables (`craftsky_profiles`, `craftsky_posts`, `craftsky_project_posts`, `craftsky_post_mentions`, `craftsky_likes`, and `craftsky_reposts`) and notification effects sourced by their URIs remain indexer-owned and are covered by the convergence contract rather than direct private cleanup.
    - Shared/public caches and non-CraftSky indexes, including `atproto_identity_cache`, `bluesky_profiles`, `atproto_follows`, and `atproto_blocks`, are not owner-private deletion targets merely because they mention the DID; CraftSky content references and derived effects are still removed through their owning indexers.
  - Every account-owned Instagram migration link, import, suggestion, verification record, private imported datum, and username claim, overriding ordinary migration retention.
    - The current Instagram manifest covers `instagram_verification_attempts`, `instagram_account_links`, `instagram_identity_claims`, link conflicts and webhook work reachable from those links/claims, `instagram_graph_imports`, imported graph handles that are no longer referenced by another owner, `instagram_follow_suggestions`, `instagram_suggestion_sources`, `instagram_reconciliation_jobs`, `pds_follow_operations`, owner-derived rate-limit buckets, `instagram_audit_events`, `instagram_notification_suggestions`, and their owner notification effects. Non-CraftSky follow records already written to the PDS are retained under RULE-002.
  - Account-owned device-local drafts, staged media, caches, ordinary sessions, and product state on the confirming device at acceptance and on other devices at next launch/contact. A minimal job binding/display identity may remain only until terminal success so the agreed status row/screen can render.
- Retained data:
  - The user's DID and PDS/AT Protocol account.
  - Records outside `social.craftsky.*`, including the Bluesky profile and any `app.bsky.*` posts, follows, likes, reposts, blocks, or other records.
  - Blobs are not record collections. Account deletion removes CraftSky record references; reclamation of unreferenced blobs remains PDS-owned, and blobs referenced by retained non-CraftSky records must not be deleted by this flow.
  - A minimal deletion audit containing only DID, job ID, timestamps, and coarse outcome from terminal success until `terminalSuccessAt + 30 days`. It contains no handle, tokens, content, relationships, preferences, imports, settings, expected URIs, Tap receipts, or credential state; it does not keep membership active or block a fresh rejoin; and it expires at that exact boundary.
  - While a job is non-terminal only, minimized job/status state, expected-URI and convergence-receipt state, and the one server-side deletion OAuth session may remain as needed to complete or recover deletion. They must obey the same data-minimization and access-control boundary and are removed before terminal success.
- Migration required: Yes. Coding design must add durable deletion job/audit/status/OAuth-binding/expected-URI/receipt support. One maintained deletion-coverage manifest shall name every private owner-DID table/service and every indirect/shared-resource cleanup rule above; a schema/completeness test must fail when a new owner-private store is added without an explicit delete or retain policy.
- Backwards compatibility: The app is not in production and has no active users, but deletion must still be safe across multiple retained accounts and multiple CraftSky sessions/devices.

## 16. UI / API / CLI Impact

- UI:
  - Restructure the Settings landing page around active identity and clear destination rows.
  - Add Account and About pages.
  - Add a Settings entry to the existing notification settings page.
  - Move Clear image cache into About.
  - Apply the error colour to Sign out and destructive deletion controls.
  - Add localized confirmation/error/accessibility copy.
  - Add fresh OAuth reauthentication, typed-handle confirmation, deletion status/Retry states, disabled deleting-account switcher rows, and the no-remaining-account status flow.
- API:
  - Add authenticated, owner-scoped `/v1/*` operations to accept permanent CraftSky membership deletion and to read/retry the resulting job through a restricted status credential.
  - Acceptance must require server-verified fresh PDS OAuth reauthentication and exact active-handle confirmation without accepting PDS passwords or email codes.
  - The job must coordinate discovery and deletion of every current/future `social.craftsky.*` PDS record collection, private AppView/Instagram cleanup, username-claim release, immediate ordinary-session/notification revocation, the deletion-only OAuth binding/replacement/removal lifecycle, pagination/batching, bounded retry, receipt-backed indexer convergence, audit expiry, and standard error envelopes.
  - Exact REST paths, response bodies/statuses, job identifiers, status-credential format, fresh-reauth proof/window, retry counts, and schema layout are coding-design decisions. They must implement the approved OAuth and convergence contracts above. Route names must communicate account-wide CraftSky deletion rather than imply that only the profile record is removed.
- CLI: None.
- Background jobs:
  - A durable worker owns the non-cancelable deletion lifecycle after acceptance and resumes safely after worker/service restart.
  - Terminal success requires a final empty CraftSky PDS rescan, private cleanup, ordinary-session revocation/removal, a receipt and absent/retracted effects for every expected URI, and deletion-only OAuth removal; PDS blob garbage collection is not a terminal gate.
  - Existing Tap replay for every deleted CraftSky record, including the membership record, remains the sole public AppView deletion mechanism and must stay idempotent across duplicates and ordering differences. The receipt observer runs after successful indexer handling and before Tap acknowledgement, and never mutates public indexed state.
  - A retention task expires the minimal deletion audit at `terminalSuccessAt + 30 days`.

## 17. Security / Privacy / Permissions

- Authentication: Starting deletion requires a valid ordinary CraftSky bearer token/device ID plus fresh PDS OAuth reauthentication. Before acceptance completes, the fresh server OAuth session is bound to the job. After acceptance, all ordinary bearer sessions are revoked/removed; the client may hold only narrowly scoped deletion-status access, while the server retains only the deletion-bound OAuth session until completion or replacement.
- Authorization: The server derives the deletion DID exclusively from authenticated context. The client does not supply a target DID capable of selecting another account.
- Reconfirmation: The user must complete fresh OAuth reauthentication, read a confirmation naming the active `@handle`, and type that exact handle. The operation must not be triggered by opening Account, selecting the initial row, or completing only the first confirmation step.
- Sensitive data: No bearer token, PDS token, raw DID, private record contents, or external-link details are shown in errors or logs.
- PDS authority: Use the server-held OAuth session to enumerate and delete all `social.craftsky.*` records in the authenticated user's repo. Never invoke whole-account deletion or enumerate/delete records from another namespace.
- Status authorization: A deletion-status credential is job- and owner-scoped, permits status, initiation of replacement fresh OAuth reauthentication, and Retry only; it cannot access ordinary APIs, PDS APIs, the server OAuth session, or another job, and is revocable/expiring.
- Deletion-worker authorization: The worker resumes only the OAuth session ID durably bound to the matching job. That session is never selected by ordinary background writers after bearer revocation, can call only the PDS list/delete operations required by this job, and is removed before terminal success. An unusable grant produces attention-required status and can be replaced only through fresh owner reauthentication.
- Abuse cases: Reject unauthenticated, stale-reauth, handle-mismatched, replayed cross-account, or cross-job requests; rate-limit destructive/retry endpoints under the established write policy; make repeated same-owner acceptance safe.
- Privacy: Private product state must not remain active after job acceptance or survive terminal deletion. Non-terminal operational state is limited to the approved job/status/OAuth-binding/expected-URI/receipt fields. Only the agreed audit may remain from terminal success until `terminalSuccessAt + 30 days`; job/status/log data must not copy deleted content or unrelated account data.

## 18. Observability

- Events: No new product analytics are required.
- Logs: Record coarse job/phase and result categories without DIDs, handles, tokens, private record content, relationship data, settings, or full URLs. Distinguish reauthentication/validation, acceptance/session revocation, PDS collection deletion, private/Instagram cleanup, indexer convergence, automatic/manual retry, terminal success, and audit expiry.
- Metrics: Count accepted jobs, terminal successes, phase latency, bounded automatic retries, manual retries, attention-required jobs, indexer-convergence delay, and categorized failures if the existing metrics abstraction supports this without adding a new observability system.
- Alerts: None required for the UI slice. Jobs that exhaust retry bounds, remain non-terminal beyond the expected deletion window, fail PDS/private cleanup repeatedly, or cannot converge AppView should be visible through existing server error monitoring.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Account deletion spans multiple PDS record collections, AppView private stores, sessions, notifications, local secure state, and eventual indexer events. | Partial completion could leave CraftSky records/membership/data divergent or produce ambiguous UX. | Use one durable, owner-scoped, safely retryable job; block ordinary access at acceptance; expose durable status; and require every terminal-success condition. |
| RISK-002 | Existing membership deletion and retention services do not yet cover every private data store uniformly. | Private data could survive contrary to the user's expectation, or required retained data could be deleted incorrectly. | Inventory every owner-DID table/service during coding design, document retention exceptions, and add store-level plus end-to-end deletion tests. |
| RISK-003 | Deleting many CraftSky PDS records emits many firehose/Tap events, including membership deletion. | Content may briefly remain visible, handlers may replay, receipt persistence may fail, or AppView convergence may be delayed. | Register expected URIs before deletion; record idempotent receipts only after indexer success and before acknowledgement; require absent/retracted effects plus a final empty rescan; never duplicate eager purge logic. |
| RISK-004 | Active account can change during identity load or destructive work. | Wrong-account identity, navigation, feedback, or deletion could occur. | Capture and validate the active account lease across every await and derive deletion ownership server-side. |
| RISK-005 | The screenshot contains Customisation but this checkout does not. | Later integration could drop or duplicate that row. | Treat screenshot behavior as required and preserve the adjacent route without redefining its feature logic. |
| RISK-006 | A chevron on direct actions would imply navigation that does not occur. | Misleading interaction and accessibility semantics. | Apply directional chevrons only to page-opening rows; style actions according to their true semantics. |
| RISK-007 | Deletion copy could over-promise blob garbage collection or instantaneous erasure from offline devices. | Loss of user trust and an unverifiable privacy promise. | State record/reference deletion and next-contact local erasure explicitly; exclude PDS GC timing from terminal success. |
| RISK-008 | A future CraftSky record Lexicon is not added to the deletion collection inventory. | Delete account would leave CraftSky records on the user's PDS contrary to the confirmed promise. | Derive or maintain one auditable record-collection registry and add a completeness test against `lexicon/social/craftsky/`. |
| RISK-009 | PDS pagination, rate limits, or per-write limits interrupt deletion of a large repo. | Some CraftSky records may remain after a partial request. | Use bounded batches, explicit progress/terminal semantics, safe retry, and failure-injection coverage before reporting success. |
| RISK-010 | Revoking all ordinary sessions at acceptance removes the usual authorization path while the worker still needs PDS authority. | The member could lose status/Retry access, or a generic background writer could misuse retained OAuth authority. | Separate the status credential from the job-bound deletion OAuth session; revoke ordinary bearers first, exclude the bound session from ordinary selectors, permit replacement only through fresh reauthentication, and remove it before success. |
| RISK-011 | Fresh reauthentication or typed-handle state binds to the wrong retained account during switching. | The wrong membership could be irreversibly deleted. | Capture the active-account lease before reauthentication, bind the proof and expected handle server-side, and reject stale or mismatched completion. |
| RISK-012 | Durable job/OAuth-binding/receipt/audit tables or logs retain copied product data. | Deleted private data or credentials could survive in operational infrastructure. | Store only minimized non-terminal identifiers and the explicitly permitted audit fields; remove operational state at success; add schema/log privacy tests and exact terminal-success-anchored expiry coverage. |
| RISK-013 | An offline device cannot be erased or notified at job acceptance. | Old local drafts or cache may remain physically present until that device reconnects. | Block ordinary access and erase on next launch/contact; make this limitation explicit in confirmation rather than promising remote instantaneous wipe. |
| RISK-014 | Existing project guidance normally treats the PDS as user-owned and warns against CraftSky-initiated deletion. | Implementation could silently conflict with the repository's architectural boundary. | Treat this explicitly confirmed account-deletion flow as a narrow user-authorized exception requiring architecture review; never delete outside `social.craftsky.*` or invoke whole-account deletion. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The existing active-account identity provider remains the authoritative source for avatar, display name, and handle. | A different identity source would need equivalent lease fencing and fallback rules. |
| ASM-002 | The existing account-switcher bottom-sheet/popover can be extended with disabled deletion-status rows without replacing its form-factor-specific presentation. | A new dedicated switcher route or materially different interaction would require revised navigation criteria. |
| ASM-003 | Terms and Privacy policy continue to use `https://craftsky.social/terms` and `https://craftsky.social/privacy`. | Canonical URLs and tests would need updating. |
| ASM-004 | The design system has or can provide a direction-aware chevron and external-link icon with the required accessibility semantics. | A shared icon treatment would need to be added before consistent row rendering. |
| ASM-005 | The existing localized navigation build-label formatter remains valid for About and tolerates an empty build number. | A formatter fix would need to be shared by shell navigation and About. |
| ASM-006 | Existing OAuth infrastructure can produce a server-verifiable fresh-reauth result and OAuth session ID bound to the active DID without the Flutter app receiving PDS credentials. | The deletion security design would need a compliant reauthentication extension before implementation. |
| ASM-007 | Existing authenticated AT identity can be safely mapped to its pending deletion job so later authentication opens status rather than onboarding. | A separate recovery identifier or support-mediated path would be required. |
| ASM-008 | Supported PDS APIs allow the AppView to list and delete every user-authorized `social.craftsky.*` record while leaving blob garbage collection PDS-owned. | The deletion promise or PDS authorization design would need revision. |
| ASM-009 | Tap's event ID/repo revision and ack-after-handler contract remain stable enough to support the approved expected-URI receipt design. | The convergence receipt contract would need an equivalent durable event-boundary signal before implementation. |

## 21. Open Questions

- None blocking for coding planning after document re-review.
- Non-blocking coding-design follow-up: choose exact REST paths, fresh-reauth proof/window, restricted status-credential format, schema layout for the approved OAuth binding and convergence receipts, PDS collection registry implementation, pagination/batching/order, durable phase model, retry counts/delays, and audit-expiry scheduler.
- Non-blocking content-design follow-up: select the existing support destination and finalize localized deletion/status copy while preserving every required boundary above.

## 22. Review Status

Status: Revised after document review
Risk level: High
Review recommended: Required
Reviewer:
Date:
Notes: The product owner explicitly approved the narrow owner-reauthenticated `social.craftsky.*` PDS deletion exception. Coding planning may proceed after the paired acceptance tests and document review approve the OAuth-lease, convergence-receipt, retention, and private-store inventory contracts. Implementation must update the repository guidance/reference before enabling deletion code.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001 through BR-004, FR-001 through FR-027, NFR-001 through NFR-004, NFR-006, and RULE-001 through RULE-011.
- Suggested test levels:
  - Flutter widget tests for avatar/name/handle fallbacks, section order, row icons, error colours, About/Account content, exact version formatter reuse, immediate cache/sign-out behavior, two-step confirmation, status/Retry states, localization, and accessibility.
  - Router integration tests for Settings child routes, external legal launching, notification reuse, responsive shell ownership, Back behavior, no-remaining-account deletion status, and pending-job authentication recovery.
  - Provider/controller tests for account lease fencing, fresh-reauth binding, typed-handle matching, immediate local cleanup, status-only credentials, MRU remaining-account activation, disabled switcher status rows, other-device next-contact cleanup, and fresh rejoin behavior.
  - AppView handler/job/store tests for authentication, owner scope, durable acceptance/idempotency, restart/network recovery, deletion-only OAuth binding/replacement/removal, restricted status/reauth-start/Retry authorization, complete current/future CraftSky collection inventory, paginated/batched PDS deletion, non-CraftSky/blob preservation, immediate ordinary-session revocation/removal, maintained private/Instagram cleanup manifests, username-claim release, bounded retries, error envelopes, and terminal-success-anchored 30-day audit expiry.
  - Integration tests covering expected-URI registration, job-driven deletion of profile/post/like/repost records, duplicate/replayed Tap deletions in different orders, post-handler/pre-ack receipt failure, temporary indexed visibility, absent/retracted effects, a final empty PDS rescan, and deletion OAuth removal before terminal success.
  - Regression tests for existing Settings destinations, notification preferences, account switching, cache clearing, sign out, drawer/navigation-rail version labels, and preservation of the AT Protocol account plus all non-CraftSky PDS records.
  - Manual visual/accessibility checks on compact iOS/Android and large tablet/desktop layouts.
- Blocking open questions: None after the revised document set passes re-review. The repository guidance/reference amendment remains an implementation prerequisite, not an unresolved product decision.
