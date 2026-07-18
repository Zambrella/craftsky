# TDD Implementation Plan: Multi-Account Sessions And Notification Routing

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes; no blocking findings
- Coding plan: `04-coding-plan.md`
- Implementation approval: the user invoked `implement-tdd` on 2026-07-18, satisfying the explicit High-risk source-implementation gate in `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and the execution notes below updated after every loop.
- Use fixed captured account sessions for authenticated requests; never read a mutable global token during a request.
- Fence asynchronous work by DID and session generation; active UI work also checks activation generation.
- Publish registry mutations only after two-slot journal read-back verification.
- Do not log or stringify tokens, cleanup credentials, routing IDs, DIDs, handles, payloads, or retained-account lists.
- Do not add bulk logout, inactive direct removal, linked-account APIs, migrations, lexicon changes, or per-account navigation history.
- Do not commit, push, or open a pull request unless separately requested.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | BR-001, FR-001, NFR-002 | AC-001, AC-018, AC-019 | Fails: no registry/journal domain model exists |
| 2 | UT-002, UT-003 | FR-001, FR-003, FR-005, FR-007, FR-018, RULE-001, RULE-004 | AC-003, AC-004, AC-018, AC-021 | Fails: no additive upsert, limit, generations, or MRU fallback |
| 3 | IT-001 | BR-001, FR-001, NFR-002 | AC-001, AC-019 | Fails: storage persists only one session blob |
| 4 | UT-005, REG-003 | FR-008, FR-017, NFR-001 | AC-009, AC-016 | Fails: interceptors use and clear global auth |
| 5 | UT-004, IT-003 | FR-007, FR-008, FR-009, FR-019, NFR-001, NFR-003 | AC-008, AC-009, AC-022 | Fails: no activation boundary or stale-completion fence |
| 6 | IT-002, UT-020, UT-022 | BR-001, FR-003, RULE-001, RULE-004 | AC-003, AC-004, AC-021 | Fails: OAuth replaces the existing session and signed-in Add route is absent |
| 7 | UT-010, IT-009 | FR-017, FR-024 | AC-016, AC-027 | Fails: startup validates one global session |
| 8 | UT-012, IT-007 | BR-003, FR-016, FR-018 | AC-015, AC-018 | Fails: sign-out clears all auth state |
| 9 | IT-013 | BR-003, FR-016, FR-026 | AC-015, AC-029 | Fails: shared-installation account isolation is not directly covered |
| 10 | UT-006–UT-008, IT-005 | BR-002, FR-012–FR-015, FR-021, FR-025 | AC-011, AC-012, AC-020, AC-024, AC-028 | Fails: opens compare only against the current DID |
| 11 | UT-011, IT-004 | FR-010, FR-011, RULE-002 | AC-010, AC-014 | Fails: registration owns one active account/client |
| 12 | UT-009, IT-006 | FR-020, RULE-002 | AC-013, AC-023, AC-031 | Fails: notification state is global |
| 13 | UT-015, IT-012 | FR-023 | AC-026 | Fails: activation bypasses shared unsaved-work protection |
| 14 | UT-013, IT-010 | BR-003, FR-016, FR-026, NFR-002 | AC-029 | Fails: offline cleanup is best-effort and cannot preserve required ordering |
| 15 | UT-016–UT-019, IT-011 | FR-004–FR-006, FR-021, FR-022, RULE-003–RULE-005 | AC-005–AC-007, AC-017, AC-021, AC-024, AC-025, AC-030 | Fails: switcher, account avatar, actions, and responsive surfaces do not exist |
| 16 | UT-014, REG-010 | NFR-002 | AC-019, AC-029 | Fails: existing diagnostic strings and router diagnostics expose identity or URL data |
| 17 | REG-001–REG-009 | FR-003, FR-004, FR-006, FR-008, FR-009, FR-010, FR-011, FR-012, FR-016, FR-023, RULE-002, RULE-003, NFR-001, NFR-004 | AC-003, AC-006, AC-008–AC-017, AC-020, AC-024, AC-026 | Pending regression verification |
| 18 | MAN-001–MAN-003 | BR-002, FR-004, FR-010–FR-013, FR-021, FR-022, FR-026, NFR-002, NFR-003 | AC-005, AC-006, AC-008, AC-010, AC-011, AC-014, AC-019, AC-024, AC-025, AC-029 | Requires supported layouts and physical devices/platform inspection |

## Implementation Steps

### Step 1: UT-001

- Write failing test: versioned two-account registry round-trip; missing/corrupt active-DID MRU repair; independently corrupt entry recovery; unsupported/all-invalid slot outcomes; redacted strings.
- Run command: from `app/`, `flutter test test/auth/models/session_registry_test.dart`.
- Confirmed failure: Yes. The focused test failed to compile because `SessionRegistry` and the session generation/MRU fields did not exist. The recovery test then failed on the corrupt entry, and the journal-selection test failed because no two-slot recovery API existed.
- Implement: Added the explicit v1 registry codec, session generation/MRU/cached identity fields, independent entry recovery, active-DID MRU repair, newest-supported-slot selection with slot-A tie behavior, empty signed-out recovery, and fully redacted registry/session strings.
- Run command: `flutter test test/auth/models/session_registry_test.dart` — passed (3 tests).
- Refactor: Reordered declarations and constructor parameters to satisfy project analysis rules while green.
- Notes: Focused `dart analyze` reports no issues. The decoder never falls back to an older valid revision merely to restore a corrupt/missing entry from the newer supported revision.

### Step 2: UT-002, UT-003

- Write failing test: additive upsert, same-DID replacement, five-account atomic limit, never-reused generation, active-first/MRU order, deterministic fallback.
- Run command: focused registry suite.
- Confirmed failure: Yes. Additive-upsert first failed because no mutation existed; the account-limit test then failed because no bounded exception/rule existed; the ordering/fallback test failed because neither ordered projection nor removal fallback existed.
- Implement: Added immutable additive/replacement upsert with monotonic session/use/activation generations, the five-distinct-DID guard with existing-DID refresh at capacity, active-first/MRU ordering with stable DID tie-break, and active removal fallback without reusing session generations.
- Run command: `flutter test test/auth/models/session_registry_test.dart` — passed (6 tests).
- Refactor: Kept registry collections unmodifiable and diagnostics bounded/redacted; focused analysis remains clean.
- Notes: A rejected sixth DID throws before constructing a new snapshot, leaving the original immutable registry unchanged.

### Step 3: IT-001

- Write failing test: two-slot read/write/restart and every TD-002 interruption point through the production decoder.
- Run command: focused secure-storage and auth-provider suites.
- Confirmed failure: Yes. The first persistence test failed because no registry storage existed; the interruption test failed because storage was not fakeable; the provider test failed because no durable registry source or verified-write mutation API existed.
- Implement: Added alternating secure slots `craftsky_session_registry_a`/`_b`, independent reads, newest-supported recovery, production-decoder read-back and canonical verification, bounded storage errors, a fakeable backend, and a keep-alive serialized registry notifier that publishes only after storage succeeds.
- Run command: focused registry model/storage/provider suites — passed (15 tests total).
- Refactor: Introduced narrow storage interfaces for deterministic interruption tests and kept the existing single-session storage temporarily for compatibility until all callers migrate in later approved steps.
- Notes: Corrupt target read-back leaves the previous winner authoritative. No legacy single-session migration was added, per approved scope. Focused analysis reports no issues.

### Step 4: UT-005, REG-003

- Write failing test: fixed A/B bearer capture, anonymous request behavior, and lease-scoped coalesced `401` invalidation.
- Run command: focused interceptor suites.
- Confirmed failure: Yes. The captured-bearer test failed because no fixed interceptor constructor existed; the scoped-`401` test failed because account keys, session leases, and lease-bound invalidation did not exist.
- Implement: Added fully redacted immutable `AccountKey`/`AccountSessionLease`, fixed and anonymous auth interceptors, an anonymous Dio provider, an auto-disposed fixed-account Dio family, and generation-checked registry invalidation. A fixed client's `401` closes over only its captured lease.
- Run command: focused session-auth, `401`, and registry-provider suites — passed (12 tests in the latest grouped run).
- Refactor: Removed the temporary dummy `SignedIn` used by the fixed interceptor; it now reads a captured token closure directly. Focused analysis is clean after using Flutter's immutable annotation.
- Notes: The legacy active `dioProvider` remains as a compatibility alias until the activation-boundary loop migrates active repositories; inactive/account-specific work can now use fixed clients without reading mutable auth.

### Step 5: UT-004, IT-003

- Write failing test: transition barrier, local/offline activation, Home reset, invalidation, and stale success/error/rollback/navigation rejection.
- Run command: focused activation and routing suites.
- Confirmed failure: Yes. The boundary test failed because the registry had no lease-aware activation and the coordinator/transition/result types did not exist. The duplicate-request test then failed with two commits instead of one.
- Implement: Added active leases carrying activation generation, lease lookup/currentness checks, immutable local activation with MRU advance, a transition-first coordinator, stale/already-active results, duplicate-target coalescing, verified registry activation mutation, account-state invalidation and Home-reset hooks.
- Run command: focused activation and account-switch routing suites — passed (3 tests).
- Refactor: Extracted the in-flight operation cleanup and used bounded/redacted transition diagnostics; focused analysis reports no issues.
- Notes: Offline activation contains no network await and remains selected after later content failure. Concrete provider invalidation inventory and router/overlay wiring use these hooks in the later UI/integration loops.

### Step 6: IT-002, UT-020, UT-022

- Write failing test: additive/repeated OAuth, onboarding route, five-account limit, and complete failure/cancellation preservation.
- Run command: focused auth-controller and router suites.
- Confirmed failure: Yes. The additive completion test initially left only A in the registry because the controller still wrote the legacy blob. The route test failed because `/add-account` did not exist.
- Implement: Handoff `whoami` now precedes a single verified registry upsert/activation; successful new or repeated DIDs preserve all other sessions. Added signed-in root `/add-account`, explicit Add-account page mode, and disabled raw GoRouter diagnostics. Storage/mutation failure maps safely and leaves the previous snapshot unpublished.
- Run command: full auth-controller suite — passed (13 tests); focused signed-in Add-account router test — passed.
- Refactor: Removed legacy single-session writes from OAuth completion while retaining the legacy storage only for not-yet-migrated sign-out/startup compatibility.
- Notes: Existing cancellation, browser-launch, timeout, and handoff-failure tests prove mutation is never reached; the new storage-failure test compares the complete serialized snapshot before/after. Onboarding/Home routing continues to derive from the newly published active DID.

### Step 7: UT-010, IT-009

- Write failing test: immediate restore, active-first validation, inactive concurrency two, transient retention, authoritative lease-scoped removal.
- Run command: focused auth-session validation suite.
- Confirmed failure: Yes. The scheduler test failed because no validation coordinator existed; the token-free auth projection test failed because `SignedIn` still required a token and startup read the legacy blob.
- Implement: Added active-first validation followed by an inactive worker pool of two, account-bound `whoami`, transient/authoritative result classification, generation-checked invalidation, immediate token-free auth projection from the secure registry, and background validation launch after cached restore.
- Run command: focused validation service and auth-session suites — passed (4 tests); token-free auth-state and fixed/anonymous interceptor suites — passed (6 tests).
- Refactor: Authenticated interceptors now read captured registry credentials instead of UI auth state. Account Dio captures device ID before construction and checks `ref.mounted` after async build gaps, preventing disposed-Ref use.
- Notes: Network/server/canceled validation retains sessions; unauthorized or identity mismatch removes only the unchanged lease. Authenticated UI/router state no longer contains bearer tokens.

### Step 8: UT-012, IT-007

- Write failing test: confirmed account-only sign-out, account state/binding cleanup, MRU Home fallback, last-account SignedOut.
- Run command: focused auth-controller, settings, and cleanup suites.
- Confirmed failure: Yes. The multi-account test showed both A and B remained because sign-out still cleared only the legacy blob; it also used the mutable/global auth client.
- Implement: Confirmed sign-out captures the active lease, uses its fixed account Auth API client, removes only that routing binding, commits lease-checked registry removal, selects the registry's deterministic fallback, and projects SignedOut only when empty. Anonymous login now uses the anonymous Dio provider.
- Run command: focused confirmed multi-account and last-account sign-out tests — passed (2 tests).
- Refactor: Split anonymous and account-bound Auth API providers and removed auth-controller dependence on bearer-bearing UI state/legacy storage for confirmed sign-out.
- Notes: The shared activation/Home-reset UI effect is wired through the activation coordinator in the later router/UI loop; the registry and auth projection already switch to the MRU fallback atomically.

### Step 9: IT-013

- Write failing test: A/B sessions and subscriptions on one installation, A-only success cleanup, and fail-closed cleanup failure.
- Run command: focused AppView logout test with test Postgres available.
- Confirmed failure: Yes, at test-fixture level: the auth-only schema did not include notification tables, so the new shared-installation contract could not exercise the real cleanup store. No production defect was exposed.
- Implement: Added the mandatory AppView contract fixture with two sessions and two subscriptions sharing one installation, real `PostStore` cleanup, A-only logout, and assertions that B's session/subscription remain active. Existing fail-closed cleanup coverage remains adjacent.
- Run command: `TEST_DATABASE_URL=... go test ./internal/auth -run 'TestLogout'` — passed.
- Refactor: Kept notification DDL scoped to the fresh auth test schema; no production Go, API, migration, or lexicon change was needed.
- Notes: The first sandboxed runs were blocked by Go cache permissions; the approved unsandboxed focused run completed against local Postgres.

### Step 10: UT-006–UT-008, IT-005

- Write failing tests one at a time: routing classification; pending-open generation; row ownership; end-to-end recipient activation and banner identity.
- Run command: smallest relevant notification suite per test ID.
- Confirmed failure: Yes. Routing tests first failed because the app had only a second DID-keyed secure blob and current-DID comparison; no exact/ambiguous/removed result or recipient lease existed. Pending-open and row tests then failed because work carried no session generation and notification rows had no producing-account owner.
- Implement: Moved routing bindings into the journaled session registry; added lease-checked binding mutations and exact/invalid/removed reverse resolution; pinned resolved recipient leases and latest pending work; fenced removal/reauthentication; activated exact inactive recipients through the shared coordinator before destination inference; added the distinct removed-account effect/message; bound notification rows to their producing active lease; and attached redacted inactive-recipient identity to foreground banner effects.
- Run command: focused registry/routing/open/pending/runtime/effect-host suites — passed (22 tests); `flutter analyze` — no issues.
- Refactor: Removed the obsolete independent secure-routing backend, kept resolution and effect strings bounded/redacted, preserved at-least-once handling for already-ready active-account opens, and limited latest-only suppression to delayed/transitioning work.
- Notes: Runtime provider wiring now saves bindings only against unchanged registry leases and uses account activation before navigation. The foreground effect retains existing title/body and exposes cached handle/avatar presentation data; the current messenger renders the required `For @handle` line, while the richer avatar presentation remains part of the planned UI surface work in Step 15.

### Step 11: UT-011, IT-004

- Write failing test: every eligible account registers with its fixed client; permission, retry, token refresh, removal fencing.
- Run command: focused registration suites.
- Confirmed failure: Yes. The first two-account test failed to compile because registration accepted only one mutable current DID and one active repository; it had no account lease, retained-account set, or lease-bound save callback.
- Implement: Replaced current-DID registration with an eligible lease set, installation-scoped permission/token gating, independent fixed-account registration clients, per-lease settled-token tracking, isolated retryable failures, all-account token refresh, serialized revision draining, and removal/reauthentication checks before binding saves. Runtime readiness now supplies every retained onboarded account without manual activation.
- Run command: focused registration/device/runtime/open/effect-host suites — passed (15 tests); `flutter analyze` — no issues.
- Refactor: Added an auto-disposed account-keyed device repository backed by `accountDioProvider`, preserved one installation-scoped notification service, and prevented successful accounts from being retried when only one lease failed.
- Notes: Provider token changes re-register every eligible lease; a late response for a removed or replaced generation cannot write a routing binding. Permission denial or provider failure leaves accounts signed in and retryable.

### Step 12: UT-009, IT-006

- Write failing test: account-keyed list/preferences/seen/count, badge formatting, isolated errors, authoritative foreground refresh.
- Run command: focused notification provider suites.
- Confirmed failure: Yes. The first count test failed to compile because only one global new-count repository/provider existed; the list/preferences/seen integration fixtures likewise had no `AccountKey` families or fixed-client repositories.
- Implement: Added redacted account-keyed repository families backed by `accountDioProvider`, durable per-account count caches, and account families for list/pagination, preferences/optimistic edits, seen render-token consumption, and new counts. Active notification pages/settings/shell/resume select the active family, while compatibility aliases preserve existing single-account tests. Foreground runtime callbacks target the resolved recipient account.
- Run command: focused account count/list/preferences/seen/open/page/settings/shell/effect-host suites — passed (33 tests); `flutter analyze` — no issues.
- Refactor: Shared the fixed account API repository across all notification interfaces, preserved row-producing leases through pagination, capped badges through the existing `NotificationBadge`, and kept one account's AsyncError independent of every other family instance.
- Notes: Duplicate foreground delivery invokes recipient-scoped authoritative refresh for each delivery; no code increments a cached count locally. Zero/unknown remains hidden and 100+ remains `99+`.

### Step 13: UT-015, IT-012

- Write failing test: clean, confirm, cancel, repeated manual and notification activation attempts across dirty flows.
- Run command: focused activation, router, and composer discard suites.
- Confirmed failure: Yes. Activation tests failed to compile because the coordinator had no pre-transition guard or cancelled result; composer integration then showed the existing PopScope confirmation was local to back/close actions and unregistered guard state initially accessed Riverpod unsafely during disposal.
- Implement: Added a shared owner-lease `UnsavedWorkGuard` with clean fast-path, overlapping-confirmation coalescing, lifecycle registration, and confirm-and-close callbacks. Activation now returns a terminal `cancelled` result before transition/commit, rechecks leases after confirmation, and notification opens silently consume cancelled attempts. Post/project composers and Edit Profile register their existing dirty checks and localized discard dialogs; runtime notification activation uses the shared guard.
- Run command: focused guard/activation/post/project/profile/open suites — passed (26 tests); `flutter analyze` — no issues.
- Refactor: Stored the guard in each ConsumerState during `initState` for disposal safety, centralized replace/unregister behavior, and reused each surface's existing confirmation copy instead of introducing a second dialog contract.
- Notes: Cancel preserves the active account and originating draft and leaves no delayed activation. Confirm closes the dirty flow before registry activation; clean flows switch without prompting. Manual switcher activation will use the same coordinator in Step 15.

### Step 14: UT-013, IT-010

- Write failing test: quarantine, stale-open rejection, restart/retry, terminal `204`/`401`, credential deletion, replacement registration ordering.
- Run command: focused sign-out recovery suite.
- Confirmed failure: Yes. The focused test failed to compile because the registry had no non-activatable cleanup queue or atomic quarantine operation and no recovery coordinator existed; the controller still reused best-effort token deletion followed by ordinary removal.
- Implement: Added redacted `PendingSessionCleanup` entries to the secure registry snapshot, atomic lease-checked quarantine/removal, terminal credential deletion, and a serialized recovery coordinator. Unconfirmed sign-out now quarantines first, clears registration eligibility, invalidates the shared provider token, and retries logout with a fixed cleanup-only credential; `204` and authoritative `401` drain the credential before remaining registration resumes. Pending work retries on startup and resume, and token refresh cannot register while the queue is non-empty.
- Run command: focused registry/storage, auth-controller, recovery, registration, and runtime suites — passed (35 tests); `flutter analyze` — no issues.
- Refactor: Reused the fixed-token auth/error interceptor stack for cleanup requests, made runtime resume refresh eligibility before registration, serialized overlapping recovery retries, and kept all cleanup diagnostics redacted.
- Notes: Secure-journal round-trip coverage includes cleanup credentials and verifies their diagnostic representation omits the credential and identity. Provider-token invalidation failures retain the queue and retry before any server cleanup or replacement registration.

### Step 15: UT-016–UT-019, IT-011

- Write failing tests one at a time: switcher model/actions; inactive banner identity; Profile avatar/fallback; no bulk sign-out; compact/large interaction and semantics.
- Run command: smallest relevant model/widget suite per test ID.
- Confirmed failure: Yes. The model test failed because no account-switcher state existed; the content test failed because there was no shared switcher widget; the Add-account router test showed the reused form still presented ordinary Sign-in copy.
- Implement: Added an active-first redacted switcher model, shared account rows with cached identity and capped per-account badges, five-account Add gating, generic-fallback `AccountAvatar`, compact Profile long-press sheet, accessible large-rail anchored menu trigger, guarded manual activation, and a blocking identity transition overlay. Profile loads now lease-check and persist cached display/avatar metadata. Add-account has distinct localized title/copy, and neither switcher nor Settings exposes bulk or inactive removal actions.
- Run command: focused registry/model/overlay/shell/router/settings suites — passed (26 tests); `flutter analyze` — no issues.
- Refactor: Both form factors render `AccountSwitcherContent`; manual selection reuses `AccountActivationCoordinator` and the shared unsaved-work guard. Account-switch invalidation clears active repositories/caches before Home reset, while normal Profile destination selection remains the existing `goBranch` path.
- Notes: Compact fallback, current-account semantics, badges, disabled-limit helper, real local switching, large anchored-menu rendering, transition interaction blocking, Add routing/copy, and absence of sign-out-all semantics are automated. Image load failure and physical-device sizing remain in Step 18 manual coverage.

### Step 16: UT-014, REG-010

- Write failing test: sentinel scan and bounded diagnostic output, with only the approved OAuth callback transport exception and intended identity UI.
- Run command: focused observability, routing-event, registry, and shell suites.
- Confirmed failure: Yes. The handoff provider's generated diagnostic string exposed its raw token/device-ID family arguments, and the provider-observer test captured arbitrary exception text containing token, routing, DID, and handle sentinels in both logs and the reporting boundary.
- Implement: Wrapped handoff credentials in an equality-preserving `HandoffClientKey` with a constant redacted string, changed provider observation to use only provider names and bounded error types, and report a redacted provider-failure wrapper while retaining safe mapped API metadata. Added sentinel coverage across registry/session/cleanup/switcher/transition/routing/open/effect models and confirmed route diagnostics remain disabled.
- Run command: focused secret-scan, provider-logger, open-event, registry, auth-controller, and shell suites — passed (38 tests); `flutter analyze` — no issues.
- Refactor: Provider logs no longer stringify family arguments, mutations, arbitrary errors, collection contents, or account model values. The OAuth callback still uses the approved URL transport, but its one-shot client provider key is non-diagnostic.
- Notes: The opaque routing ID remains parsed from provider data and authenticated registration responses while all tested string/log/report surfaces redact it. Intended cached identity continues to render only in the switcher, Profile destination, inactive banner, and transition overlay.

### Step 17: REG-001–REG-009

- Write/update regression test: one behavior at a time where existing coverage is insufficient.
- Run command: focused auth/router/notification/composer/feature suites.
- Confirmed failure: The broad suites initially exposed no failure, but final behavior audit found the confirmed sign-out test did not prove account-boundary invalidation or Home reset for an MRU fallback. Tightening that assertion failed until both effects were wired. A subsequent complete Flutter run exposed two more deterministic gaps: notification readiness dereferenced an asynchronously loading registry, and the existing root-post repost regression test stopped at the share menu instead of selecting its Repost action.
- Implement: Centralized account-state invalidation for manual/notification activation and sign-out; made active-account sign-out, authoritative startup validation failure, and account-bound `401` reset retained-account fallback navigation to Home after lease-checked durable removal; treated a loading registry as having no notification-registration candidates; and corrected the repost regression interaction to exercise the existing share-menu flow.
- Run command: full `test/auth test/notifications test/router` group — passed (174 tests); shared auth-interceptor, Settings, post/project discard groups — all 27 existing tests passed; Edit Profile groups — passed (11 tests); focused account-boundary/validation/`401` tests — passed (6 tests); complete `flutter test --reporter compact` — passed (908 tests).
- Refactor: Replaced duplicate shell/runtime invalidation lists with one account-boundary provider and retained the requirement-linked implementations already green in their focused suites.
- Notes: One aggregate command named a non-existent `edit_profile_dialog_discard_test.dart`; this was a command-path error, not a test failure. The actual Edit Profile suites were located and passed. Inactive and stale authoritative invalidations do not disturb the active UI boundary. Single-account auth/routing, fixed bearer selection, notification copy/lifecycle/preferences/seen/open behavior, and existing discard confirmations remain green.

### Step 18: MAN-001–MAN-003

- Check: responsive visual/keyboard/screen-reader behavior, physical-device push/recovery behavior, and platform secure-storage/redaction behavior.
- Run command: manual; no hermetic command can prove these checks.
- Outcome: Not run; blocked by the current environment. No physical iOS/Android device, provider-delivery session, platform secure-storage inspection session, or active screen-reader session is attached to this workspace.
- Notes: Automated compact/large-layout rendering, semantics, interaction blocking, account switching, provider-token recovery ordering, secure-journal recovery, and diagnostic sentinel tests all pass. MAN-001–MAN-003 still require supported physical devices and assistive-technology/platform inspection before release sign-off.

## Final Verification

- From `app/`: `dart run build_runner build`.
- From `app/`: `dart analyze`.
- From `app/`: `flutter test test/auth test/notifications test/router`.
- From `app/`: `flutter test test/shared/api/providers/session_auth_interceptor_test.dart test/shared/api/providers/sign_out_on_401_interceptor_test.dart`.
- From `app/`: `flutter test test/settings test/feed/widgets/post_composer_sheet_discard_test.dart test/projects/widgets/project_composer_discard_test.dart`.
- From repository root: `just test`.
- Inspect `git diff`, map every source change to requirement/test IDs, and preserve unrelated worktree changes.

Results:

- `dart run build_runner build` — passed; generated Riverpod/router/mappable outputs are current.
- `dart analyze` — passed with no issues.
- Complete `flutter test --reporter compact` — passed (908 tests).
- Focused auth, notification, router, interceptor, Settings, composer, Edit Profile, boundary, startup, and repost regression suites — passed.
- Repository `just test` — passed, including the race-enabled Go suite and shared-installation logout contract.
- `git diff --check` — passed.
- No lexicon files changed. No commit, push, or pull request was created.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Generated files and localization updated
- [x] Manual checks completed or explicitly blocked with environment reason
- [x] Docs and execution notes updated
- [x] Implementation review completed; `06-implementation-review.md` required the correction pass below
- [x] No commit, push, or pull request created without separate authorization

## Review Correction Pass

### Inputs

- Implementation review: `06-implementation-review.md` — Changes required, High risk, 2026-07-18
- Correction authorization: the user selected `Address required changes` on 2026-07-18

### Correction Test Order

| Step | Finding / Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| C1 | IR-001 / UT-004, UT-005, IT-003, IT-008 | FR-008, FR-009, FR-017, NFR-001 | AC-008, AC-009, AC-016 | Fails: the production active feature Dio still reads mutable global auth and globally signs out on `401` |
| C2 | IR-002 / IT-002, IT-003 | FR-003, FR-007, FR-009, NFR-001 | AC-003, AC-008, AC-009 | Fails: successful Add-account OAuth activates B without invalidating A's account-scoped state |
| C3 | IR-003 / UT-018, IT-011 | FR-004, FR-006, FR-022, NFR-004 | AC-005, AC-006, AC-025 | Fails: large navigation uses a separate switch button and its Profile destination ignores the active avatar/switch action |
| C4 | IR-004 / UT-001, IT-001 | FR-001, FR-018, NFR-001, NFR-002 | AC-001, AC-018, AC-019 | Fails: malformed active-pointer types reject the newest recoverable slot and inconsistent counters can reuse ownership generations |

### Correction Steps

#### C1: IR-001 / UT-004, UT-005, IT-003, IT-008

- Write failing test: exercise the production `dioProvider` under active A, switch to B, and deliver A's delayed `401`; assert fixed bearer ownership and A-only invalidation.
- Run command: `flutter test test/shared/api/providers/dio_provider_test.dart`.
- Confirmed failure: Yes. The production provider returned the same mutable Dio after B activation, so the test failed before bearer/`401` assertions with `dioB` identical to `dioA`.
- Implement: Made `dioProvider` rebuild from the active registry session, capture its bearer and immutable lease, use lease-scoped `401` invalidation, close the old client on disposal, and use an anonymous client only when no active session is available. Removed the production mutable-token and global-sign-out constructors. Expanded account-boundary invalidation to active mutation providers that read repositories imperatively.
- Run command: focused `dio_provider_test.dart` — passed (2 tests); full `test/shared/api/providers` — passed (16 tests).
- Refactor: Kept the existing synchronous active-repository API while making its underlying client immutable per active session. Account-family clients remain unchanged and anonymous login remains bearer-free.
- Notes: The focused test dispatches under A before activation, proves the rebuilt active client carries B, then completes A's delayed `401` and verifies A-only removal with B still active.

#### C2: IR-002 / IT-002, IT-003

- Write failing test: complete B Add-account OAuth from retained A and assert the account-state boundary runs after durable upsert.
- Run command: `flutter test test/auth/providers/auth_controller_test.dart`.
- Confirmed failure: Yes. The additive completion test activated and persisted B but observed no account-state invalidation.
- Implement: Added a `beforePublish` hook to the serialized registry upsert. Add-account completion now verifies and writes the full journal snapshot, invalidates A-scoped state only after durable success, and publishes B only after that boundary completes. First-account sign-in remains direct.
- Run command: focused additive completion test — passed; full auth-controller and registry-provider suites — passed (17 tests).
- Refactor: Kept invalidation inside the durable mutation's pre-publication window so a storage failure preserves both the prior registry and prior account state.
- Notes: Existing router observation of the newly published active account continues to select onboarding or Home; no premature Home reset was added.

#### C3: IR-003 / UT-018, IT-011

- Write failing test: on a large layout, render the active Profile avatar and open the anchored menu from the Profile destination's long-press action while normal tap remains Profile navigation.
- Run command: `flutter test test/router/app_shell_account_switcher_test.dart`.
- Confirmed failure: Yes. The large-layout test found no `AccountAvatar` in the navigation rail because Profile rendered the generic icon and the account switcher lived on a separate trailing button.
- Implement: Removed the separate rail switch button, attached avatar/fallback plus long-press/keyboard/semantics switching to the Profile destination icon, and anchored the menu to the Profile destination label.
- Run command: focused large-layout test — passed; full account-switcher widget suite — passed (4 tests).
- Refactor: Reused `_DestinationIcon` for compact and large layouts so both surfaces share the same account-switch action and active-avatar behavior while normal rail destination selection remains owned by `NavigationRail`.
- Notes: The large test now asserts the cached avatar URL, absence of the separate switch icon, the Profile long-press semantics action, anchored content, and real local account activation.

#### C4: IR-004 / UT-001, IT-001

- Write failing test: recover the newest supported slot with a non-string active pointer and inconsistent next-generation/MRU counters; preserve sessions, repair active by MRU, and advance counters past retained ownership.
- Run command: `flutter test test/auth/models/session_registry_test.dart`.
- Confirmed failure: Yes. Recovery rejected revision 5 because `activeDid: 42` failed the direct cast and selected the older revision 1 snapshot.
- Implement: Treats a non-string active pointer as missing repairable metadata, requires positive entry generations/ordinals, and advances decoded next-generation and next-use counters beyond every retained session and pending cleanup generation.
- Run command: focused corrupt-metadata test — passed; registry model, secure-storage, and registry-provider suites — passed (18 tests).
- Refactor: Counter repair occurs only inside the newest supported top-level snapshot; it never selects an older revision to recover discarded metadata.
- Notes: The test also performs the next additive upsert and proves the repaired generation and MRU ordinal are the values assigned to the new account.

### Correction Completion Checklist

- [x] IR-001 fixed with production-shape active-client coverage
- [x] IR-002 fixed with additive OAuth boundary coverage
- [x] IR-003 fixed with large Profile destination coverage
- [x] IR-004 fixed with corrupt-journal recovery coverage
- [x] Generated outputs current
- [x] Static analysis and full Flutter/Go regressions passing
- [x] `05-implementation-plan.md` read back after final update
- [x] Ready for implementation re-review

### Correction Final Verification

- `dart run build_runner build` — passed and regenerated current Riverpod outputs. The existing non-failing analyzer language-version warning remains.
- `dart analyze` — passed with no issues.
- Focused correction suites — passed: active fixed-client/lease-scoped `401`, additive OAuth boundary, compact/large switcher, registry/journal recovery, auth controller/provider, and shared API provider coverage.
- Complete `flutter test --reporter compact` — passed (908 tests).
- Repository `just test` — passed, including the race-enabled Go suite and shared-installation logout contract.
- `git diff --check` — passed.
- No lexicon files changed. No commit, push, or pull request was created.
- `MAN-001`–`MAN-003` remain not run for the environment reasons already recorded in Step 18; they remain a physical-device/platform pre-release gate rather than a correction-code blocker.

## Second Review Correction Pass

### Inputs

- Re-review: `06-implementation-review.md` — Changes required, High risk, 2026-07-18
- Correction authorization: the user selected `Address required changes` on 2026-07-18

### Correction Test Order

| Step | Finding / Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| R1 | IR-005 / UT-004, IT-003, REG-004 | FR-007, FR-008, FR-009, NFR-001 | AC-008, AC-009 | Fails: no production-shape feature-provider test releases late A read success/error and optimistic rollback after B activation |
| R2 | IR-006 / AT-003, UT-018, IT-011, REG-002 | FR-004, FR-006, FR-022, NFR-004 | AC-006, AC-025 | Fails: shell coverage omits normal Profile tap, keyboard switching, selected avatar treatment, and missing/failed avatar variants |

### Correction Steps

#### R1: IR-005 / UT-004, IT-003, REG-004

- Write failing test: Start controlled timeline pagination, user-post read, and optimistic like work under A; activate B through the real registry/coordinator/invalidator; release A success, error, and rollback completions; require only B state to remain.
- Run command: `flutter test test/auth/providers/account_boundary_provider_test.dart --plain-name 'UT-004 IT-003 late A reads errors and rollback cannot publish as B' --reporter expanded`.
- Confirmed failure: Yes. The first precise failure left the A user-post family in `AsyncLoading` after B activation. After adding the missing family invalidations, the test exposed the deeper race: A's delayed timeline `loadMore` appended `account-a` and `account-a-late` into B's rebuilt notifier because `ref.mounted` remained true across the rebuild.
- Implement: Expanded the approved account-boundary inventory to live feed/comment/profile/project/search families and mutation notifiers. Added an active-operation ownership guard that captures DID/session/activation generation before the first await and rejects completion after activation changes. Applied it to pagination, optimistic rollback, create/delete/report, profile/follow/save, project/search, and recent-search async mutations that previously relied on mounted-only checks.
- Run command: Focused boundary test — passed; boundary plus timeline and post-interaction suites — passed (24 tests).
- Refactor: Centralized the generation check in `account_operation_guard.dart`; isolated provider tests without a registry retain their existing mounted behavior, while production authenticated operations require the captured active lease to remain current.
- Notes: The production Dio test from the first correction pass already proves the subsequent B client carries only B's bearer. This loop adds the previously missing real feature-provider success/error/rollback evidence and uncovered/fixed a real same-notifier Riverpod rebuild race.

#### R2: IR-006 / AT-003, UT-018, IT-011, REG-002

- Write failing test: Keep the shell router alive, normally activate Profile, assert no switcher and selected avatar state, focus the Profile switch action and invoke Alt+Down on large layout, and cover failed/missing avatar fallbacks in unselected and selected states.
- Run command: `flutter test test/router/app_shell_account_switcher_test.dart --reporter expanded`.
- Confirmed failure: Yes. Once the router fixture retained the mounted auto-dispose router, tapping the Profile avatar left the rail on Feed because the nested switch-action surface did not forward normal activation. After that was corrected, the selected avatar semantics assertion failed because its semantics node exposed no selected state.
- Implement: Forwarded Profile avatar taps to the existing destination callback, gave the shared Profile switch surface a lifecycle-owned focus node so Alt+Down invokes the same compact/large switch action, and exposed `selected` on `AccountAvatar` semantics. No separate switch button or ordinary-navigation prompt was added.
- Run command: Full account-switcher widget suite — passed (5 tests).
- Refactor: Kept normal destination selection and account-switch activation as separate callbacks on the same Profile surface. Updated the test fixture to retain the mounted auto-dispose router and use a deterministic profile repository.
- Notes: The large test now proves ordinary Profile activation leaves the switcher closed, selected avatar treatment, long-press semantics availability, keyboard-opened anchored content, and real account selection. Compact coverage proves failed-image fallback; the added large fixture proves null-image fallback in unselected and selected states.

### Second Correction Completion Checklist

- [x] IR-005 covered with production-shape late read/mutation completion evidence
- [x] IR-006 covered with normal tap, keyboard, selected avatar, and fallback evidence
- [x] Generated outputs current
- [x] Static analysis and full Flutter/Go regressions passing
- [x] `05-implementation-plan.md` read back after final update
- [x] Ready for implementation re-review

### Second Correction Final Verification

- `dart run build_runner build` — passed and regenerated 97 current outputs. The existing non-failing analyzer language-version warning remains.
- `dart analyze` — passed with no issues.
- Focused IR-005 test — passed; boundary plus timeline and post-interaction suites — passed (24 tests).
- Account-switcher widget suite — passed (5 tests).
- Expanded feature-provider and router regression set — passed (108 tests).
- Complete `flutter test --reporter compact` — passed (910 tests).
- Repository `just test` — passed, including the race-enabled Go suite.
- `git diff --check` — passed.
- No lexicon files changed. No commit, push, or pull request was created.
- `MAN-001`–`MAN-003` remain not run for the environment reasons recorded in Step 18; they remain a physical-device/platform pre-release gate rather than a correction-code blocker.

## Post-Approval Field Correction Pass

### Inputs

- Field report, 2026-07-18: after first sign-in, the Profile navigation avatar remained generic until Profile was opened.
- Field report, 2026-07-18: a signed-in Add-account OAuth callback returned to the old account's Home without retaining the new account.

### F1: Initial Active-Account Identity Hydration

- Write failing test: Pump the signed-in shell with an active session that has no cached avatar and an own-profile response that contains one; require the Profile destination and durable registry identity to update without navigating to Profile.
- Confirmed failure: Yes. The Profile destination's `AccountAvatar.avatarUrl` remained null because the only cached-identity listener lived inside `ProfilePage`.
- Implement: Added an auto-disposed active-account identity provider. The shell loads the active account's own `userProfileProvider` immediately, renders the returned avatar, and persists display/avatar metadata only while the captured session lease remains current. Cached identity remains the fallback while loading or on error.
- Focused result: `initial signed-in shell hydrates the active Profile avatar` — passed.

### F2: Signed-In Add-Account OAuth Callback

- Write failing test: Enter `/auth/complete` while A is already signed in; process B through the real `AuthController`; require A and B to remain in the registry, B to become active, and the app to return Home.
- Confirmed failure: Yes. The router redirected the signed-in callback to A's Home before `AuthCompletePage` mounted, so `completeFromDeepLink` never consumed B's handoff token. After allowing the callback page to mount, the integration test then proved successful completion needed an explicit post-success navigation.
- Implement: Signed-in callbacks now remain on `AuthCompletePage` while OAuth completion runs. A successful `AsyncLoading` to `AsyncData` transition navigates to Home, where the existing redirect sends incomplete accounts to onboarding. Error states remain on the completion page. The existing verified registry upsert continues to retain A and activate B.
- Focused result: `Add account callback retains A, activates B, and returns Home` — passed.

### Field Correction Final Verification

- `dart run build_runner build` — passed and wrote 97 current outputs. The existing non-failing analyzer language-version warning remains.
- `dart analyze` — passed with no issues.
- Complete auth suite — passed (73 tests).
- Complete router suite — passed (21 tests).
- Complete `flutter test --reporter compact` — passed (912 tests).
- Repository `just test` — passed, including the race-enabled Go suite.
- `git diff --check` — passed.
- No dependency, migration, or lexicon files changed. No commit, push, or pull request was created.
- `MAN-001`–`MAN-003` remain the documented physical-device/platform pre-release gate.

## Approved Complexity Simplification Pass

### Inputs

- Complexity review: `06-implementation-review.md` — Approved with notes, IR-007 through IR-012.
- Requirement/test/coding-plan amendments: approved by the user on 2026-07-18.
- Scope: single fail-closed snapshot, online-confirmed sign-out, lazy inactive validation, no inactive switcher badges, switcher-local activation progress, and obsolete compatibility/boilerplate removal.

### Simplification Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| P1 | SIM-UT-001 | SIM-FR-001, NFR-002 | SIM-AC-001 | Fails: storage and codec still implement a tolerant two-slot journal and pending cleanup entries |
| P2 | SIM-UT-002 | SIM-FR-002, FR-016, FR-018 | SIM-AC-002 | Fails: transient logout removes/quarantines the active account and starts recovery |
| P3 | SIM-UT-003 | SIM-FR-003, FR-017 | SIM-AC-003 | Fails: startup schedules inactive `whoami` calls through a worker pool and launch guard |
| P4 | SIM-UT-004 | SIM-FR-004, RULE-002 | SIM-AC-004 | Fails: switcher rows/watchers still expose and fetch inactive badges |
| P5 | SIM-IT-005 | SIM-FR-005, FR-007, FR-009, NFR-001 | SIM-AC-005 | Fails: manual activation closes the switcher immediately and uses a global transition overlay |
| P6 | SIM-REG-006 | SIM-NFR-001, SIM-FR-001, SIM-FR-005 | SIM-AC-006 | Fails: legacy storage/mapper, unused action/source state, recovery files, and duplicated mutation bodies remain |

### Simplification Steps

#### P1: SIM-UT-001

- Write failing test: require one secure-storage key, strict whole-snapshot decoding, signed-out fallback for corrupt reads, and surfaced write failures.
- Run command: focused `secure_token_storage_test.dart` and `session_registry_test.dart` cases.
- Confirmed failure: Yes. The single-key API did not exist and the codec salvaged valid accounts from a partially corrupt registry.
- Implement: Replaced the alternating journal with `craftsky_session_registry`; reads fail closed to `SessionRegistry.empty()`, writes target only that key, and the codec rejects any malformed entry or counter.
- Run command: both focused SIM-UT-001 tests — passed.
- Refactor: Removed registry revision/recovery state and consolidated registry reconstruction behind `_copyWith`.
- Notes: A corrupt snapshot now requires signing in again by design; a failed write is never published to Riverpod state.

#### P2: SIM-UT-002

- Write failing test: make logout fail with `ApiNetworkError` and require the exact active registry snapshot and UI boundary to remain unchanged.
- Run command: focused AuthController SIM-UT-002 case.
- Confirmed failure: Yes. The controller quarantined the active account and created a pending cleanup credential.
- Implement: Only a completed logout or `ApiUnauthorized` permits local removal. Other API failures are retained and surfaced for retry.
- Run command: focused test and complete AuthController suite — passed.
- Refactor: Deleted pending cleanup storage, recovery coordination/provider, and startup/resume retry hooks.
- Notes: Account state is invalidated before publishing a confirmed fallback account, preserving the hard account boundary without a global overlay.

#### P3: SIM-UT-003

- Write failing test: require startup to launch validation for Alice only, then launch Bob only after Bob becomes active.
- Run command: focused auth-session-provider SIM-UT-003 case.
- Confirmed failure: Yes. The ownership launch guard prevented activation from launching Bob's lazy validation.
- Implement: AuthSession now tracks the last active lease and schedules `whoami` only for a newly active lease.
- Run command: focused test and complete auth-session-provider suite — passed.
- Refactor: Removed the inactive worker pool, concurrency setting, ownership-map launch guard, and coordinator test.
- Notes: Invalid inactive credentials remain retained until that account is activated.

#### P4: SIM-UT-004

- Write failing test: open the switcher with an inactive unread count and require no badge or inactive count fetch.
- Run command: focused account-switcher SIM-UT-004 cases.
- Confirmed failure: Yes. The row rendered `7` and opening the switcher subscribed to Bob's count provider.
- Implement: Switcher rows contain identity/selection only; the shell retains the active navigation badge.
- Run command: focused widget cases and complete account-switcher suite — passed.
- Refactor: Removed notification-count inputs, row badge state, and live inactive count watches.
- Notes: No notification repository or routing semantics changed.

#### P5: SIM-IT-005

- Write failing test: require the target row to show an inline progress indicator while all switcher actions are disabled.
- Run command: focused SIM-IT-005 widget case.
- Confirmed failure: Yes. `AccountSwitcherContent` had no local activation state and the caller closed it immediately.
- Implement: `_LiveAccountSwitcherContent` now owns the activation future, keeps the surface open while busy, and closes it after activation succeeds.
- Run command: focused test and complete account-switcher suite — passed.
- Refactor: Removed the global transition provider/widget and activation source/transition models.
- Notes: Account state is invalidated before activation publishes the target registry, so removing the global visual shield does not weaken account isolation.

#### P6: SIM-REG-006

- Write/update regression check: search production/tests for journal keys, pending cleanups, legacy storage/mappers, transition models, action enums, and inactive notification inputs.
- Run command: `rg` over `app/lib` and `app/test` for the removed symbols.
- Confirmed failure: Yes. Every obsolete family still had production and test references at the start of the pass.
- Implement/refactor: Removed the legacy `SecureTokenStorage`, StoredSession mapper, pending-cleanup/recovery and unused notification-cleanup files, transition files, switcher action/helper state, and duplicated registry-provider mutation bodies.
- Run command: final `rg` returned no matches; generated output, analysis, complete Flutter tests, Go race tests, and `git diff --check` passed.
- Notes: Fixed account clients, leases, exact notification routing, account-scoped invalidation, and stale-result fences remain intact.

### Simplification Completion Checklist

- [x] Amended Must requirements covered by focused tests
- [x] All simplification tests passing
- [x] Fixed clients, account leases, exact notification routing, and stale-result fences preserved
- [x] Generated outputs and localization current
- [x] Relevant focused and complete Flutter tests passing
- [x] AppView Go tests passing
- [x] `git diff --check` passing
- [x] `05-implementation-plan.md` read back after final update
- [x] No commit, push, or pull request created without separate authorization

### Simplification Final Verification

- `dart run build_runner build` — passed and wrote 101 current outputs. The existing non-failing analyzer language-version warning remains.
- `flutter analyze` — passed with no issues.
- Complete `flutter test --reporter compact` — passed (904 tests).
- Repository `just test` — passed, including the race-enabled Go suite.
- Removed-symbol regression search — passed with no matches.
- `git diff --check` — passed.
- Net implementation/test change removes substantially more code than it adds; no dependency, migration, API, or lexicon changes were required.
- No commit, push, or pull request was created.

## Account-Switch Request-Storm Correction Pass

### Inputs

- Field report, 2026-07-18: switching retained profiles can leave the previous profile image visible in the selector/navigation surface.
- Correlated Flutter, AppView, and PostgreSQL logs: registry-driven client rebuilds repeatedly cancelled profile/timeline queries, retried account reads, and eventually exhausted the AppView per-token read limit with `429 rate_limited`.
- Correction authorization: the user explicitly requested the required fixes on 2026-07-18.

### Correction Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| S1 | IT-003, REG-003, REG-004 | FR-008, FR-009, NFR-001 | AC-008, AC-009 | Fails: active and account-scoped Dio clients rebuild and force-cancel requests for metadata-only registry writes |
| S2 | UT-010, IT-009 | FR-017, FR-024 | AC-016, AC-027 | Fails: cached identity and routing metadata writes relaunch whole-registry session validation |
| S3 | UT-009, UT-018, IT-006, IT-011 | FR-005, FR-020, FR-022 | AC-007, AC-023, AC-025 | Fails: the shell may render a previous account identity while the new profile loads and eagerly fetches every inactive count before the switcher opens |

### Correction Steps

#### S1: IT-003, REG-003, REG-004

- Write failing test: update cached identity for a retained lease while active and account-scoped clients are mounted; require both client instances and captured bearer ownership to remain stable.
- Run command: `flutter test test/shared/api/providers/dio_provider_test.dart --reporter expanded`.
- Confirmed failure: Yes. A cached-avatar write disposed the active Dio provider and constructed a new client even though the active token, DID, and session generation were unchanged.
- Implement: Selected only the target account's token, DID, and session generation from registry state. Active and account-scoped clients now survive metadata, MRU, and unrelated-account writes, while a real ownership change still closes the old client and creates a correctly owned replacement.
- Run command: Focused stability test — passed; complete Dio-provider suite — passed (3 tests).
- Refactor: Centralized the minimal client target and account-selection records in `dio_provider.dart`; initial registry loading remains awaited without subscribing each client to the whole registry value.
- Notes: Bearer tokens remain internal to the client target and are not exposed through provider diagnostics. The existing token-rotation test continues to prove that changed session ownership replaces the client.

#### S2: UT-010, IT-009

- Write failing test: mutate cached identity without changing session membership, token, generation, or active ownership; require no second validation launch.
- Run command: `flutter test test/auth/providers/auth_session_provider_test.dart --reporter expanded`.
- Confirmed failure: Yes. Switching between two already-retained accounts launched session validation twice because every registry revision retriggered validation.
- Implement: Added a validation launch guard keyed by each retained account's session generation. Activation, MRU, cached identity, and routing metadata changes no longer relaunch validation; adding, removing, or rotating a session still does.
- Run command: Focused retained-account switch test — passed; complete auth-session-provider suite — passed (4 tests).
- Refactor: Kept the comparison in `SessionValidationLaunchGuard`, with a redacted diagnostic representation, so the auth provider only coordinates validation when session ownership changes.
- Notes: The guard deliberately ignores `activeAccount`, because activating a session that has already been validated does not change its server-side validity.

#### S3: UT-009, UT-018, IT-006, IT-011

- Write failing test: switch from a loaded A avatar to B while B's profile remains pending; render B's cached avatar immediately, never A's previous value, and do not fetch B's inactive count until the switcher opens.
- Run command: `flutter test test/router/app_shell_account_switcher_test.dart --reporter expanded`.
- Confirmed failure: Yes. While B's profile request was pending, the shell rendered A's previously loaded avatar. The same fixture also showed B's inactive notification count was fetched before the switcher opened.
- Implement: Bound hydrated profile identity to the exact active session lease and require that lease at render and persistence time. The shell now uses B's cached identity immediately during B's load. Moved inactive-account count watches into consumer content that exists only while the compact sheet or large anchored menu is open.
- Run command: Both focused regressions — passed; complete account-switcher widget suite — passed (8 tests).
- Refactor: Added `_LiveAccountSwitcherContent` as the single live switcher boundary; the always-mounted shell watches only the active account's badge count.
- Notes: Closing the switcher disposes inactive count subscriptions. Reopening it obtains current values without keeping every account's notification provider live throughout normal navigation.

### Correction Completion Checklist

- [x] Metadata-only registry writes preserve fixed network clients
- [x] Session validation is keyed to session ownership rather than registry revision
- [x] Active identity rendering is fenced by the current lease
- [x] Inactive notification counts load only while the switcher needs them
- [x] Focused and broad regressions pass
- [x] `05-implementation-plan.md` read back after final update

### Request-Storm Correction Final Verification

- `dart run build_runner build` — passed and wrote 97 current outputs. The existing non-failing analyzer language-version warning remains.
- `dart analyze` — passed with no issues.
- Complete Dio-provider suite — passed (3 tests).
- Complete auth-session-provider suite — passed (4 tests).
- Complete account-switcher widget suite — passed (8 tests).
- Auth, notification, router, and shared-API regression set — passed (199 tests).
- Complete `flutter test --reporter compact` — passed (916 tests).
- Repository `just test` — passed, including the race-enabled Go suite.
- `git diff --check` — passed.
- No dependency, migration, API, rate-limit, or lexicon changes were required. No commit, push, or pull request was created.
- `MAN-001`–`MAN-003` remain the documented physical-device/platform pre-release gate.

## Sign-Out Confirmation Correction Pass

### Inputs

- Field request, 2026-07-18: show a success message after manual account sign-out.
- If no retained account remains, confirm sign-out only. If fallback activation keeps another account signed in, identify that account in the confirmation.
- Requirement linkage: `FR-016`, `FR-018`; acceptance criterion `AC-015`.

### Correction Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| L1 | UT-023 | FR-016 | AC-015 | Fails: last-account sign-out completes without user feedback |
| L2 | IT-014 | FR-016, FR-018 | AC-015 | Fails: fallback activation completes without identifying the newly active account |

### Correction Steps

#### L1: UT-023

- Write failing test: complete manual sign-out with no remaining account and require one transient informational success message.
- Run command: `flutter test test/settings/sign_out_tile_test.dart --plain-name 'UT-023 last-account sign-out shows success' --reporter expanded`.
- Confirmed failure: Yes. Manual sign-out completed with no message dispatched.
- Implement: Added a redacted `SignOutResult`, returned it only after the account-scoped sign-out and any Home reset complete successfully, and dispatched a localized transient informational confirmation through the shared app messenger.
- Run command: Focused last-account widget test — passed.
- Refactor: The widget captures the messenger and localized copy before awaiting sign-out, so the confirmation remains safe when the settings route is removed by the signed-out redirect.
- Notes: A failed controller result remains null and cannot emit success feedback.

#### L2: IT-014

- Write failing test: complete manual sign-out with an MRU fallback and require the same confirmation to identify the newly active account by handle.
- Run command: `flutter test test/settings/sign_out_tile_test.dart --plain-name 'IT-014 fallback sign-out identifies the active account' --reporter expanded`.
- Confirmed failure: Yes. The fallback result still displayed the last-account-only confirmation and omitted the newly active account.
- Implement: Added localized parameterized copy using the fallback session's canonical handle: `Signed out successfully. Now signed in as @<handle>.` The controller returns that token-free identity only after fallback activation and Home reset succeed.
- Run command: Focused fallback widget test — passed; sign-out widget plus controller suites — passed (17 tests).
- Refactor: Kept outcome selection in the controller and presentation/localization in the widget. Added a nearby regression proving a null/failed result emits no success message.
- Notes: The result's diagnostic string is fully redacted; it never contains the handle or any credential.

### Sign-Out Confirmation Completion Checklist

- [x] Last-account sign-out shows one success confirmation
- [x] Fallback sign-out identifies the newly active retained account
- [x] Failed sign-out does not show success feedback
- [x] Localized generated outputs are current
- [x] Focused and broad regressions pass
- [x] `05-implementation-plan.md` read back after final update

### Sign-Out Confirmation Final Verification

- `flutter gen-l10n` — passed; localized sign-out confirmation outputs are current.
- `dart run build_runner build` — passed and wrote 97 current outputs. The existing non-failing analyzer language-version warning remains.
- `dart analyze` — passed with no issues.
- Sign-out widget and AuthController suites — passed (18 tests), including the real fallback handle result and last-account result.
- Complete `flutter test --reporter compact` — passed (919 tests).
- Repository `just test` — passed, including the race-enabled Go suite.
- `git diff --check` — passed.
- No dependency, migration, API, rate-limit, or lexicon changes were required. No commit, push, or pull request was created.
- `MAN-001`–`MAN-003` remain the documented physical-device/platform pre-release gate.
