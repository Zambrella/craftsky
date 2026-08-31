# Coding Plan: Onboarding Experience

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)

## 2. Implementation Strategy
Implement the feature in dependency order with strict red-green-refactor loops: first make the already-fetched OAuth Bluesky profile available through the canonical indexer, then add permanent private completion storage and its authenticated API, then replace Flutter's local completion flag and startup routing, and finally build the flow UI and extract reusable Instagram sections.

The design keeps existing boundaries intact:

- OAuth receives a narrow projector interface; `auth` does not import `index`.
- Direct OAuth projection adapts the fetched record into the existing `BlueskyProfile.Handle` path so Tap and OAuth share one parser/upsert implementation.
- Completion is private AppView state derived only from authenticated context; Flutter stores no durable completion or pending marker.
- Flutter server state is session-lease scoped, while visible flow state is active-account-lease scoped.
- Profile saves use the existing API and server-side read-before-write/`ExpectedCID` behavior; no lexicon or profile route changes are required.
- Existing Instagram providers remain behavior owners. The work is a structural widget extraction, not a backend/provider redesign.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| OAuth initialization | Callback fetches Bluesky profile, initializes CraftSky profile, then creates handoff | Retain fetched record/CID and invoke a best-effort canonical projector before handoff | FR-024, NFR-004, RULE-007 | UT-009, AT-019, IT-009, REG-008 |
| Bluesky indexing | `index.BlueskyProfile.Handle(tap.Event)` owns parsing and idempotent DID upsert | Add an application adapter that synthesizes the same event from OAuth's fetched map and CID | NFR-004, RULE-007 | IT-009, REG-008 |
| Completion persistence | Per-DID `SharedPreferences` boolean | Add one private permanent Postgres row per completed DID and both deletion inventories | FR-018, RULE-003, RULE-004 | IT-001, IT-005, REG-007 |
| Completion API | No server contract | Add authenticated status read and idempotent completion creation under `/v1/` | FR-018, FR-019, FR-020 | IT-001, IT-002, IT-008 |
| Flutter completion state | Synchronous `OnboardingStatus(Did)` notifier | Replace with session-lease-scoped async server state plus optimistic, in-memory retry | FR-018, FR-019, RULE-003 | UT-004, IT-008, AT-018 |
| Startup/router | Router reads local boolean; separate refresh listener | Include onboarding status in active-account initialization and route from the resolved exact lease | FR-020, RULE-004 | UT-006, AT-013, REG-003, REG-006 |
| Notification readiness | Notification consumers synchronously read local completion flags | Consume resolved active/all-session server-backed statuses and treat unresolved state as ineligible | FR-018, NFR-003 | AT-015, REG-003, REG-007 |
| Flow state | Placeholder page has no draft/controller model | Add active-lease-scoped flow controller, steps, drafts, prefill retry, save state, and action derivation | FR-001–FR-007, FR-011, FR-017, FR-021, FR-022 | UT-001–UT-005, AT-001–AT-008, AT-016 |
| Profile editing | Edit dialog privately owns limits, picker, save, and cache publication | Share constraints/cache publication; add account-scoped profile and image dependencies for onboarding | FR-002, FR-003, FR-005, FR-014, FR-023, RULE-001, RULE-006 | AT-003, AT-004, AT-006, AT-017, UT-007, IT-003, REG-001, REG-004 |
| Crafts | Existing 22-item `EditProfileCraftsPicker` and `Craft.values` | Reuse picker/catalog and preserve unknown craft IDs in every onboarding profile payload | FR-004, RULE-001 | AT-005, AT-017, REG-005 |
| Instagram UI | One large settings page with private widgets | Extract reusable account, import-composer, and suggestion sections; structurally retain history/revoke as settings-only | FR-008, FR-013, FR-015, FR-016 | AT-009–AT-012, IT-007, REG-002 |
| Onboarding page | Static scaffold and Finish button | Compose responsive three-step page, progress, Back/Skip rules, persistent action, and shared sections | FR-001, FR-009, FR-010, FR-012, NFR-001, NFR-002 | AT-001, AT-007, AT-012, AT-014 |
| Localization/generated code | Hard-coded placeholder copy; generated l10n/Riverpod/router/mappers | Add ARB keys and regenerate outputs after source changes | FR-012, FR-013, NFR-002 | AT-014, REG-003 |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/auth/initialize_profile.go` | Change | Return optional fetched Bluesky record/CID internally and call a narrow best-effort projector | FR-024, RULE-007 | UT-009, AT-019 |
| `appview/internal/auth/handlers_oauth.go` | Change | Carry projector dependency through callback initialization before `CreateExchange` | FR-024 | UT-009, AT-019 |
| `appview/internal/app/oauth_bluesky_profile_projection.go` | Create | Marshal the fetched map, synthesize canonical `tap.Event`, and call `BlueskyProfile.Handle` | NFR-004 | IT-009, REG-008 |
| `appview/internal/app/deps_tap.go`, `deps.go`, `routes_adapter.go` | Change | Construct and expose the projector without adding an `auth` to `index` dependency | FR-024, NFR-004 | IT-009, REG-008 |
| `appview/internal/routes/dependencies.go`, `routes_public_auth.go` | Change | Thread the narrow projector into OAuth handlers | FR-024 | AT-019, REG-008 |
| `appview/internal/auth/initialize_profile_test.go`, `handlers_test.go` | Change | First-failing projection branching, redaction, and ordering tests | FR-024, RULE-007 | UT-009, AT-019 |
| `appview/internal/app/federated_real_flow_integration_test.go` | Change | Real canonical direct projection before recording handoff | FR-024, NFR-004 | IT-009 |
| `appview/internal/index/bluesky_profile_test.go`, `bluesky_backfiller_test.go` | Change/retain | Prove same-CID replay and fallback behavior remain correct | NFR-004, RULE-007 | IT-009, REG-008 |
| `appview/migrations/000062_account_onboarding_completion.{up,down}.sql` | Create | Add/drop private permanent completion rows | FR-018, RULE-004 | IT-001, IT-005 |
| `appview/internal/api/onboarding_store.go` | Create | Lifecycle-guarded status read and idempotent complete operation | FR-018, RULE-003, RULE-004 | IT-001, IT-005 |
| `appview/internal/api/onboarding.go` | Create | CamelCase response handlers using authenticated DID and canonical envelopes | FR-018, FR-020 | IT-001, IT-002 |
| `appview/internal/routes/routes_onboarding.go` | Create | Register onboarding status/completion routes | FR-018 | IT-002 |
| `appview/internal/routes/policy.go`, `routes.go`, `inventory_test.go` | Change | Add exact route policies and catalogue composition | FR-018, RULE-003 | IT-002 |
| `appview/internal/accountdeletion/private_cleanup.go` | Change | Remove completion during explicit CraftSky account deletion | FR-018 | IT-005 |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Register the plaintext DID column for terminal identity purge | FR-018, RULE-003 | IT-005 |
| `appview/internal/api/onboarding_test.go` | Create | Store/handler status, idempotency, isolation, guard, and envelope tests | FR-018, RULE-003, RULE-004 | IT-001 |
| `appview/internal/routes/onboarding_route_test.go` | Create | Auth/device/current-member/body/query policy tests | FR-018 | IT-002 |
| `appview/internal/accountdeletion/private_cleanup_test.go` and owner-lifecycle/migration suites | Change | Prove Alice-only cleanup, retry safety, inventory, up/down/reapply | FR-018, RULE-003 | IT-005 |
| `app/lib/onboarding/models/onboarding_completion.dart` | Create | Immutable server status model (`completed`, optional `completedAt`) | FR-018 | IT-008 |
| `app/lib/onboarding/data/onboarding_api_client.dart` | Create | `GET` status and `POST` completion HTTP calls | FR-018, FR-019 | IT-008 |
| `app/lib/onboarding/data/onboarding_repository.dart`, `api_onboarding_repository.dart` | Create | Testable completion repository boundary | FR-018, FR-019 | UT-004, IT-008 |
| `app/lib/onboarding/providers/onboarding_repository_provider.dart` | Create | Build repository from the owning `AccountSessionLease` account transport | FR-018, RULE-003 | UT-004, IT-008 |
| `app/lib/onboarding/providers/onboarding_status_provider.dart` | Change | Async status load, optimistic completion, finite in-memory retry, session-generation guard | FR-018, FR-019, FR-020, RULE-003 | UT-004, UT-006, AT-018, IT-008 |
| `app/lib/auth/models/active_account_initialization.dart` and provider/gate | Change | Carry resolved completion and expose retryable startup errors | FR-020, RULE-004 | UT-006, AT-013, REG-006 |
| `app/lib/router/router.dart`, `onboarding_refresh_listener.dart` | Change/Delete | Listen to active initialization directly and remove synchronous per-DID listener | FR-018, FR-020 | AT-013, REG-003, REG-007 |
| `app/lib/notifications/widgets/notification_effect_host.dart` | Change | Use active initialization completion readiness | FR-018, NFR-003 | AT-015, REG-007 |
| `app/lib/notifications/providers/notification_runtime_provider.dart` | Change | Watch session-scoped completion states for eligible stored accounts | FR-018, NFR-003 | AT-015, REG-007 |
| `app/lib/onboarding/models/onboarding_flow_state.dart`, `onboarding_action_state.dart` | Create | Step, draft, baseline, upload/save, error, and derived action state | FR-001–FR-007, FR-011, FR-017, FR-022 | UT-001, UT-002, UT-005 |
| `app/lib/onboarding/data/onboarding_profile_payload.dart` | Create | Compose complete editable profile snapshot and preserved unknown crafts | FR-005, RULE-001, RULE-006 | UT-007, AT-017 |
| `app/lib/onboarding/providers/onboarding_flow_provider.dart` | Create | Active-lease-scoped prefill, navigation, image upload, save, and stale-result controller | FR-002–FR-007, FR-011, FR-014, FR-021–FR-023 | UT-001, UT-003, UT-005, AT-003–AT-008, AT-016 |
| `app/lib/profile/data/profile_field_constraints.dart` | Create | Share display-name and bio limits | FR-003 | AT-004, REG-001 |
| `app/lib/profile/providers/profile_repository_provider.dart` | Change | Add active-lease-scoped profile repository for onboarding | NFR-003, RULE-003 | AT-015, IT-006 |
| `app/lib/profile/providers/profile_image_picker_provider.dart` | Change | Add active-lease/account transport variant while retaining gallery-only behavior | FR-003, FR-023, NFR-003 | AT-003, AT-004, MAN-001 |
| `app/lib/profile/providers/save_profile_provider.dart` plus a small shared cache helper | Change | Reuse authoritative profile cache publication without duplicating save logic | FR-005, RULE-001 | IT-003, REG-001 |
| `app/lib/onboarding/widgets/onboarding_progress.dart` | Create | Localized label, semantics, and non-interactive progress bar | FR-012, NFR-002 | UT-002, AT-001, AT-014 |
| `app/lib/onboarding/widgets/onboarding_profile_step.dart` | Create | Avatar, read-only handle, display name, and bio UI | FR-002, FR-003, FR-023 | AT-003, AT-004, MAN-001 |
| `app/lib/onboarding/widgets/onboarding_crafts_step.dart` | Create | Reuse `EditProfileCraftsPicker` | FR-004 | AT-005, REG-005 |
| `app/lib/onboarding/widgets/onboarding_instagram_step.dart` | Create | Optional/privacy copy and approved shared Instagram sections | FR-008, FR-013, FR-015, FR-016 | AT-009–AT-012 |
| `app/lib/onboarding/widgets/onboarding_bottom_action.dart` | Create | Render derived `Next`, `Save & next`, or `Finish` state | FR-005, FR-006, FR-009, FR-022 | UT-001, AT-006, AT-012 |
| `app/lib/onboarding/pages/onboarding_page.dart` | Change | Full responsive scaffold, progress, Skip, PopScope, scrolling, and step composition | FR-001, FR-009–FR-012, FR-017, NFR-001, NFR-002 | AT-001, AT-007, AT-008, AT-014 |
| `app/lib/instagram_migration/widgets/instagram_migration_sections.dart` | Create | Host five extracted public sections: three reusable and two settings-only | FR-008, FR-015, FR-016 | AT-009–AT-012, IT-007, REG-002 |
| `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Change | Recompose settings from extracted sections without behavior loss | FR-008 | IT-007, REG-002 |
| `app/lib/l10n/app_en.arb` and generated localization output | Change/Generate | Add all onboarding-visible and semantic strings | FR-012, FR-013, NFR-002 | AT-014 |
| `app/test/onboarding/**` and affected router/auth/profile/Instagram tests | Create/Change | Implement the acceptance specification at named targets | All Flutter requirements | AT-001–AT-018, UT-001–UT-008, IT-003–IT-008, REG-001–REG-007 |

## 5. Services, Interfaces, And Data Flow

### OAuth Projection

Keep the auth-facing interface narrow and typed:

```text
BlueskyProfileProjector.ProjectBlueskyProfile(
  context,
  did: syntax.DID,
  cid: syntax.CID,
  record: map[string]any,
) -> error
```

Delete the redundant exported `InitializeProfile(...)` function. It has no independent production caller; direct callers are tests, while production already uses `InitializeProfileAndIdentityCache`. Refactor the underlying work into one private result-producing helper consumed by `InitializeProfileAndIdentityCache`, and update direct tests to exercise that sole production entry point (or the private helper from package-internal tests where narrower coverage is useful).

```text
OAuth callback
  -> fetch optional app.bsky.actor.profile and CID once
  -> validate/create mandatory CraftSky profile
  -> if Bluesky profile exists:
       run projector under a child deadline bounded by callback context
       warn safely and continue on projector failure
  -> retain repository AddRepo tracking
  -> retain identity-cache update
  -> CreateExchange handoff
```

The application adapter JSON-encodes the fetched map and builds the same create/update-shaped `tap.Event` used by the existing backfiller: DID, CID, `app.bsky.actor.profile`, `self`, canonical AT URI, and raw record. It calls `BlueskyProfile.Handle`, not transactional `Project`, because OAuth may precede projected CraftSky membership and has no ingestion receipt/generation. No fetched record, DID, CID, token, or raw error is logged.

### Completion API And Store

Use these wire operations:

```text
GET  /v1/onboarding/status       -> { completed, completedAt? }
POST /v1/onboarding/completion   -> { completed: true, completedAt }
```

Both routes use `AccessCurrentMember`, `BodyNoBody`, the existing bearer/device middleware, and no DID/body/query selector. The POST is idempotent and returns the original completion timestamp.

Persistence shape:

```text
account_onboarding_completions
  account_did   TEXT PRIMARY KEY
  completed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
```

Missing row means incomplete. `Complete` starts a transaction, calls `ownerlifecycle.GuardPrivateMutationTx`, inserts with `ON CONFLICT DO NOTHING`, selects the authoritative row, and commits. There is no unset/version/update operation. Avoid a foreign-key cascade so the existing explicit private and terminal cleanup systems remain the visible deletion authority.

Suggested internal contract:

```text
OnboardingStatusStore.Status(context, syntax.DID) -> OnboardingStatus
OnboardingStatusStore.Complete(context, syntax.DID) -> OnboardingStatus
```

Handlers use `middleware.GetDID`, `envelope.WriteJSON`, and redacted canonical errors. Route-local store construction in `AddRoutes` is sufficient; do not widen application dependency structs solely for this store.

### Flutter Completion Data Flow

```text
AccountSessionLease
  -> accountDioProvider(lease.account)
  -> OnboardingApiClient
  -> ApiOnboardingRepository
  -> onboardingStatusProvider(lease)
  -> activeAccountInitializationProvider for active lease
  -> initialization gate + router + notification readiness
```

`OnboardingCompletion` maps `completed` and nullable `completedAt`. Repository methods are `readStatus()` and `complete()`.

The status notifier is keyed by `AccountSessionLease`, not bare DID or activation generation. This ensures requests and optimistic state belong to one stored session generation, survive an ordinary account switch in the same process, and are discarded if that session is removed or replaced. `completeOptimistically()` publishes complete before awaiting POST, retains the notifier while retrying, and uses an injectable finite exponential retry policy. Each attempt first verifies that `SessionRegistry.leaseFor(account)` still equals the captured session lease. Failure is silent in UI; exhausted retry logs contain no account data. No `SharedPreferences` key or durable pending marker remains.

### Profile Save Data Flow

```text
active AccountSessionLease + ActiveAccountLease
  -> account-scoped ProfileRepository and blob upload client
  -> bounded fetchMe prefill
  -> OnboardingFlow drafts/baseline
  -> complete editable payload
  -> updateMe
  -> authoritative Profile response
  -> shared user-profile cache publication
  -> baseline reset + one-step advance
```

Profile payload composition is explicit:

```text
displayName = current identity draft
description = current identity draft
crafts = visible selected IDs in catalogue order + preserved unknown IDs
avatar = newly uploaded blob only when replaced
clearAvatar = false
banner/clearBanner = omitted/false
```

The existing AppView read-before-write merge preserves applicable image fields and uses `ExpectedCID`. The client does not add a CID, pre-save fetch, merge prompt, or conflict UI.

### Instagram Data Flow

Keep existing `ActiveAccountLease` family providers and operation guards unchanged. Extract public widgets, not a new aggregate controller:

```text
InstagramAccountSection
  verification + linked status + discoverability + reactivation

InstagramImportComposerSection
  export picker/parser + manual input + create import

InstagramSuggestionsSection(onSuggestionTap?)
  follow + dismiss + load-more; navigation only when callback supplied

InstagramImportHistorySection             // settings only
InstagramRevokeVerificationAction         // settings only
```

The import composer must watch/retain `instagramImportsProvider(lease)` itself and wait for provider readiness, because onboarding intentionally omits the history section that currently initializes that provider. Settings supplies a suggestion-route callback; onboarding supplies none and never imports router code into the shared section.

## 6. State, Providers, Controllers, Or DI

### Provider Graph

```text
sessionRegistryProvider
  -> onboardingRepositoryProvider(AccountSessionLease)
  -> onboardingStatusProvider(AccountSessionLease) [keep alive during retry]
  -> activeAccountInitializationProvider
       + accountLanguagePreferencesProvider(ActiveAccountLease)
       + onboardingStatusProvider(lease.session)
  -> ActiveAccountInitialization(lease, languages, onboardingComplete)

activeAccountInitializationProvider
  -> ActiveAccountInitializationGate
  -> goRouterProvider refresh/redirect
  -> NotificationEffectHost active readiness

sessionRegistryProvider.sessions
  -> notificationRuntimeProvider watches each session's onboarding status
  -> only AsyncData(completed: true) sessions are notification eligible

activeAccountInitializationProvider.requireValue.lease
  -> onboardingFlowProvider(ActiveAccountLease)
       + accountProfileRepositoryProvider(lease)
       + accountProfileImagePickerProvider(lease)
       + onboardingStatusProvider(lease.session)
       + existing Instagram provider families on step 3
```

### Flow Controller

Use a generated Riverpod `AsyncNotifier` family keyed by `ActiveAccountLease`. `build` performs the guarded initial profile read and bounded prefill sequence. State contains:

- `OnboardingStep` enum: profile, crafts, Instagram.
- Authoritative loaded/saved `Profile` baseline.
- Identity draft and known-craft draft.
- Preserved unknown craft IDs.
- Avatar preview bytes, uploaded blob, uploading/failed status.
- Save busy/error status.
- Boolean recording whether the one bounded prefill sequence has run.

Methods remain focused: update identity, toggle craft, pick avatar, next/save-and-next, previous, skip, and finish. Every async completion verifies the exact active lease before changing visible state. Skip/Finish delegate only completion to the session-scoped status notifier; they do not persist flow drafts.

Use a pure `deriveOnboardingActionState` function rather than embedding action-label and enabled-state branching in widgets. Do not introduce `useMemo`/`useCallback` or unrelated state packages.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

`OnboardingPage` remains the typed `/onboarding` route target but becomes a full-screen `ConsumerWidget`/`ConsumerStatefulWidget` host for the exact active initialization lease.

Widget outline:

```text
PopScope
  Scaffold
    AppBar
      progress title/label
      Skip text action
    SafeArea
      Column
        OnboardingProgress
        Expanded
          scrollable constrained content (max width approximately 720)
            switch step
              OnboardingProfileStep
              OnboardingCraftsStep
              OnboardingInstagramStep
        OnboardingBottomAction
```

Behavioral rules:

- Progress is semantics-enabled but has no tap callback.
- System/app-bar Back moves 3 to 2 or 2 to 1; step 1 intercepts Back.
- Submitted profile save disables Back, Skip, and duplicate submit.
- Avatar upload blocks save but not Skip; late upload completion after exit/account change is discarded.
- Instagram busy state remains local to its controls and never disables Finish or Back.
- Skip immediately discards flow state, preserves prior writes, optimistically completes, and exits without a dialog.
- Finish behaves the same for completion and does not invoke cancellation on Instagram operations.
- Persistent action and scroll padding respect safe areas, software keyboard, large text, and long Instagram content.
- Step 1 shows gallery replacement and read-only `@handle`; excluded controls are not built.
- Step 2 directly reuses `EditProfileCraftsPicker` and localized `Craft.values` order.
- Step 3 structurally omits history, revocation, and suggestion navigation.

The initialization gate remains outside the route content. Loading does not guess a route; status failure shows the existing retry surface and invalidates the exact onboarding status/repository dependency together with other account-critical state.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| OAuth profile absent | Skip direct projection; continue initialization, tracking, and handoff | RULE-007 | UT-009, AT-019 |
| OAuth projection fails/times out | Emit non-sensitive warning; continue tracking/cache/handoff | RULE-007 | UT-009, IT-009 |
| Duplicate direct/Tap CID | Existing CID-aware upsert leaves one correct row | NFR-004 | IT-009, REG-008 |
| Completion row absent | Return `200` with `completed:false` | FR-018 | IT-001 |
| Repeated/concurrent completion | Return original timestamp; retain one row | FR-018, RULE-004 | IT-001 |
| Departed/stale owner writes completion | Lifecycle guard prevents commit; canonical route error | RULE-003 | IT-001, IT-002 |
| Cold-start status loading | Initialization gate remains visible; router does not choose flow/main shell | FR-020 | UT-006, AT-013 |
| Cold-start status error | Show retry gate and invalidate exact status read on Retry | FR-020 | UT-006, AT-013 |
| Optimistic completion write fails | Stay in main app; retry silently for captured session lease | FR-019 | UT-004, AT-018 |
| Retry lease removed/replaced | Stop retries and ignore late results | NFR-003, RULE-003 | UT-004, AT-015 |
| All retries exhausted/process ends | No durable marker; later cold start follows server state | FR-019 | UT-004, AT-018 |
| Profile read error | Show retryable initialization error; never expose blank writable snapshot | FR-014 | AT-016, IT-004 |
| Bluesky identity fields absent | Retry successful reads for at most five seconds once, then show optional empties | FR-021 | UT-003, AT-016 |
| Invalid identity/upload pending/upload failed | Show localized feedback and disable save | FR-003 | AT-004 |
| Profile save fails | Keep step/draft, restore applicable controls, expose Retry/error | FR-007 | AT-006 |
| Profile save/account activation changes | Reject stale result and do not advance/update new account | NFR-003 | AT-015, IT-006 |
| Unknown craft IDs | Hide from catalogue but append unchanged to every payload | RULE-001 | UT-007, AT-017 |
| No optional values | Clean Next/Finish remain enabled; no profile write | RULE-002 | AT-002 |
| Instagram unavailable/load error | Preserve shared error/retry state while Finish/Skip remain available | FR-008, FR-009 | AT-009, AT-012 |
| Instagram action in flight | Disable only affected Instagram controls; Back/Finish remain independent | FR-009 | AT-012 |
| Import composer without history | Retain/await imports provider itself before create | FR-015 | AT-009, IT-007 |
| Suggestion row in onboarding | Inline actions work; identity area has no route callback | FR-016 | AT-011, UT-008 |
| Compact/large-text/keyboard content | One scroll surface, safe padding, reachable persistent action, no overflow | NFR-001, NFR-002 | AT-014, MAN-002 |

## 9. Test Implementation Plan
| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-009 | `appview/internal/auth/initialize_profile_test.go` | PDS fake returning profile/CID, recording projector/logger/handoff | Fetched record and CID are currently discarded; no projector call occurs |
| 2 | IT-009, AT-019, REG-008 | App real-flow and index tests | Postgres `bluesky_profiles`, full blob record, recording handoff, duplicate Tap event | No direct row exists before handoff |
| 3 | IT-001 | `appview/internal/api/onboarding_test.go` | Isolated Postgres, Alice/Bob lifecycle contexts | Store/table/handlers do not exist |
| 4 | IT-002 | `appview/internal/routes/onboarding_route_test.go` | Full policy mux, bearer/device/member cases | Routes and policies are absent |
| 5 | IT-005 | migration, private cleanup, terminal inventory suites | Seed Alice/Bob completions; up/down/reapply | New table is absent from cleanup/inventory |
| 6 | UT-004, IT-008, AT-018 | Flutter onboarding repository/status tests | Fake Dio, fake retry delays, session replacement/disposal | Status is local synchronous `SharedPreferences` state |
| 7 | UT-006, AT-013, REG-003, REG-006, REG-007 | initialization/router tests | Async incomplete/complete/error statuses and no local prefs | Initialization/router do not wait for server status |
| 8 | UT-001, UT-002, UT-005 | action/progress/flow-state tests | Pure state fixtures and controller reconstruction | No step/action/draft model exists |
| 9 | UT-003, AT-016, IT-004 | profile prefill tests | Empty then populated profile responses, fake clock, true read error | No bounded retry or retryable profile gate exists |
| 10 | UT-007, AT-017, IT-003, REG-001, REG-004 | profile payload/save suites | Full profile with image fields and unknown craft; external edit case | No onboarding snapshot composer exists |
| 11 | AT-003, AT-004, MAN-001 | profile-step/widget tests | Existing/empty profile, image picker upload states | Placeholder page has no identity controls |
| 12 | AT-005, REG-005 | craft-step/widget tests | Known/unknown crafts and stable catalogue | Placeholder page has no craft controls |
| 13 | AT-006, AT-007, AT-008 | save/navigation/completion widget tests | Deferred save, failure, drafts, Skip | Placeholder has no required navigation contract |
| 14 | AT-009–AT-012, UT-008, IT-007, REG-002 | onboarding/settings Instagram tests | Unlinked/linked/inactive, imports, paginated suggestions, deferred operations | Required widgets are private to settings; onboarding has none |
| 15 | AT-001, AT-002, AT-014, AT-015, IT-006, MAN-002 | full flow/layout/account-isolation tests | Exact leases, viewport/text scale matrix, deferred operations | Full flow, semantics, and stale-result guards are absent |

Focused commands during red-green work:

```text
go test ./internal/auth -run 'TestInitializeProfile|TestOAuth.*Projection'
just appview-test-unit
just test
just app-test test/onboarding
just app-test test/router test/auth
just app-test test/instagram_migration test/profile
just app-analyze
```

Final AppView release evidence is `just appview-check`, not `just appview-test-unit`, because database tests, migration down-to-zero/reapply, race coverage, and cleanup/inventory checks must run. Final Flutter evidence is `just app-test` plus `just app-analyze`. Run MAN-001 and MAN-002 on supported targets before release evidence is considered complete.

## 10. Sequencing And Guardrails
- First TDD step: Add UT-009 proving a fetched Bluesky record/CID reaches a best-effort projector before handoff while missing/profile-projection-failure cases still complete OAuth.
- Dependencies between work items: OAuth projection is independent and lands first; AppView completion migration/store precedes Flutter repository; Flutter repository/status precedes startup/router and notifications; flow state precedes widgets; Instagram extraction precedes onboarding step 3 composition.
- Guardrail: Do not modify `lexicon/`, profile record shapes, existing profile routes, OAuth wire shapes, or Tap/backfill registration.
- Guardrail: Do not call transactional Bluesky projection from OAuth or manufacture ingestion receipts/generations.
- Guardrail: Do not log profile records, image references, DID/CID, tokens, handles, Instagram input, or raw secrets.
- Guardrail: Derive completion DID exclusively from middleware context and use `GuardPrivateMutationTx`.
- Guardrail: Register completion data in explicit account-deletion cleanup and terminal DID inventory in the same migration step.
- Guardrail: Preserve the first completion timestamp and provide no reset/version endpoint.
- Guardrail: Do not write `onboarded_<did>` or any durable Flutter pending marker.
- Guardrail: Use `AccountSessionLease` for server completion ownership and `ActiveAccountLease` for visible flow operations; reject stale generations at every async boundary.
- Guardrail: Keep the existing server-side profile `ExpectedCID`; add no client concurrency field or UX.
- Guardrail: Preserve unknown crafts and all non-onboarding profile fields through explicit payload composition plus existing AppView merge behavior.
- Guardrail: Structurally omit Instagram history/revocation/navigation from onboarding rather than hiding them behind one broad mode flag.
- Guardrail: Do not couple onboarding Finish/Back to Instagram provider busy states.
- Guardrail: Do not hand-edit `.g.dart`, `.mapper.dart`, generated localization, or generated router files; run project generators.
- Out of scope: analytics, camera/avatar removal/banner editing, resumable drafts/steps, durable retries, versioned completion, profile conflict UI, Instagram backend changes, and Tap/backfill removal.

Generation after source/model/provider changes:

```text
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n
```

Run from `app/`, then verify generated diffs contain only expected Riverpod, mapper, router, and localization changes.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | OAuth eager projection bypasses durable ingestion receipts and can consume callback time. | Handoff could be delayed, or the derived row could exist after a later callback failure. | Limit it to `bluesky_profiles`, use a bounded child context, keep it warning-only, and retain Tap tracking/backfill. |
| CPQ-002 | Non-blocking | Same-CID upsert is idempotent, but different-CID event ordering is not a revision-ordering guarantee. | A delayed older Tap event could temporarily replace the eager row before convergence. | Preserve existing semantics; do not invent revision ordering in this feature. Cover only approved same-CID idempotency. |
| CPQ-003 | Non-blocking | The checkout's stores use embedded pgx SQL and currently have no sqlc config/generated queries despite repository guidance mentioning sqlc. | Bootstrapping sqlc would substantially broaden this feature. | Follow the established store pattern for the two completion queries and record the tooling discrepancy for a separate infrastructure decision. |
| CPQ-004 | Non-blocking | The older profile-onboarding design derives completion from crafts and rejects an `is_onboarded` flag. | Following it would contradict the approved account-wide completion contract. | Treat `01-requirements.md` as the superseding decision for this change; do not derive completion from profile fields. |
| CPQ-005 | Non-blocking | Notification runtime currently expects synchronous completion for every stored account. | A naive async conversion could register incomplete/unresolved accounts or stop multi-account notification behavior. | Watch one session-scoped status per stored session and include only resolved complete accounts; add regression coverage. |
| CPQ-006 | Non-blocking | The import composer currently relies on the settings history widget watching its provider. | Onboarding imports could race provider initialization or auto-dispose. | Make the extracted composer retain/await the imports provider and test it without history mounted. |
| CPQ-007 | Non-blocking | Finite silent retry timing is an engineering policy not exposed to users. | Brittle timing can slow tests or create excessive traffic. | Inject the delay policy; use a small exponential schedule in production and fake time in UT-004. No durable state is introduced. |
| CPQ-008 | Non-blocking | Real gallery permissions and broad locale/layout combinations exceed widget-test fidelity. | Automated success does not prove platform and visual usability. | Complete MAN-001/MAN-002 and run implementation review before merge, as required by `03-document-review.md`. |

No blocking questions remain.

## 12. Handoff To TDD Builder
- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: UT-009 in `appview/internal/auth/initialize_profile_test.go`.
- Focused command: from `appview/`, `go test ./internal/auth -run 'TestInitializeProfile|TestOAuth.*Projection'`.
- Notes: Keep each red-green-refactor slice aligned with section 9, record commands/results in `05-implementation-plan.md`, run required generators only after source annotations/models change, and finish with `just appview-check`, `just app-test`, `just app-analyze`, MAN-001/MAN-002 evidence, and the formal implementation-review stage.
