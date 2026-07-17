# Coding Plan: Direct Push Notification Routing

## 1. Inputs

- Requirements: `01-requirements.md` — Approved, High risk, 2026-07-17
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with no findings, High risk, 2026-07-17
- Repository guidance: `AGENTS.md`
- Architecture references:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Existing implementation inspected:
  - AppView durable notification events, push dispatcher/sender/payload, route registry/policy, destination APIs, moderation filtering, and notification-resolution handler/store
  - Flutter Firebase adapter, provider-neutral notification runtime, pending-open readiness slot, secure DID-keyed routing storage, resolution repository/policy, effect host, typed GoRouter routes, destination providers/pages, shared API error mapping, localization, and notification-row behavior
  - Existing Go and Flutter unit, integration, widget, router, privacy, lifecycle, and route-policy tests
- Approval gate: coding planning is approved; source implementation remains blocked until the user explicitly invokes or approves `implement-tdd`.

## 2. Implementation Strategy

Make one coordinated pre-launch cutover across AppView and Flutter:

1. AppView extends the existing internal `push.SendRequest` with typed canonical actor/source/subject facts already stored on `notification_events`. The dispatcher selects those columns in the existing claim query. `BuildPayload` emits `payloadVersion=1`, `type`, `accountSubscriptionId`, and only the category-specific facts required by the approved matrix. Visible title/body, token/platform/TTL, claiming, fencing, retries, and delivery lifecycle remain unchanged.
2. Flutter replaces nullable all-or-nothing provider parsing with a provider-neutral `NotificationOpenAttempt`. The attempt contains an independently parsed nullable account-subscription binding and a sealed fact outcome (`valid`, `unknown`, or `invalid`). Missing/malformed facts therefore cannot erase a valid binding before the coordinator applies the account gate.
3. After readiness, `NotificationOpenCoordinator` loads the current DID's secure binding and rejects any absent/malformed/mismatched binding before consulting the fact outcome. A matching binding is followed by pure destination inference and synchronous effect emission; there is no notification-specific HTTP dependency.
4. A pure inference service maps validated facts to provider-neutral destinations. The existing navigation bridge alone constructs typed `UserProfileRoute`, `PostThreadRoute`, and `NotificationsRoute` objects. Reply facts carry the subject URI as the thread and the source URI as `focus`.
5. Post/profile destination providers remain the content authority. A small shared destination-error policy distinguishes named permanent `404 post_not_found` / `profile_not_found`, transient network/`5xx`/`502 identity_unavailable`, and `401` authentication loss. Post and profile pages use one shared localized error surface for permanent Back/View notifications actions and transient Retry.
6. Remove the AppView resolver handler/store/route/policy and the Flutter resolution API/interface/provider/model/policy together. Generic/unknown notification-feed rows become non-interactive; known hydrated rows and explicit unavailable rows retain their current behavior.

The first red test is UT-001 in `app/test/notifications/models/notification_open_event_test.dart`. It establishes the structured provider-neutral attempt before any routing, Firebase, AppView payload, UI, or resolver-removal work.

No database migration, lexicon change, PDS access change, new route, dependency addition, persisted open queue, receipt deduplication, retry scheduler, literal deep link, or identifier-bearing telemetry is introduced.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria IDs | Test IDs |
|---|---|---|---|---|---|
| AppView push projection | Dispatcher sends notification ID, category, binding, copy, token/platform/TTL | Select typed actor/source/subject facts and emit the exact version 1 category matrix without notification ID or route policy | FR-001, FR-002, FR-004, FR-014, FR-017, NFR-003, RULE-001 | AC-002, AC-004, AC-014, AC-016, AC-018 | UT-011, UT-012, IT-001, IT-002, REG-001, REG-002 |
| Flutter provider boundary | Nullable parser rejects the whole event when any common field is invalid | Return one structured open attempt with independent binding and fact validity; allowlist and type all accepted facts | FR-005, FR-011, FR-012, FR-014, FR-019, NFR-002, NFR-004 | AC-005, AC-011, AC-012, AC-014, AC-017, AC-019, AC-021 | UT-001, UT-002, UT-003, UT-010, UT-015, REG-007, REG-009 |
| Binding and readiness | Latest-only pending event, then secure binding, then HTTP resolution | Preserve latest-only readiness; make binding the mandatory gate before pure inference or fallback; clear on sign-in boundary | FR-006, FR-013, FR-018, RULE-003 | AC-006, AC-013, AC-022 | AT-002, AT-004, AT-008, UT-006, UT-008, IT-003, IT-004, REG-006, REG-010 |
| Destination inference/navigation | AppView resolution returns post/profile/list target; Flutter maps it to typed routes | Infer exact category destination locally and emit navigation immediately; add reply focus | BR-001, FR-003, FR-007, FR-016, NFR-001 | AC-001, AC-003, AC-007, AC-016 | AT-001, AT-003, AT-005, UT-004, UT-005, UT-007, UT-014, IT-003, IT-009 |
| Destination authorization/errors | Destination APIs authorize reads; post has generic Retry; profile has generic load error | Keep authenticated APIs; classify permanent/transient/auth failures; share localized unavailable/Retry surface | BR-002, FR-008, FR-009, FR-010, NFR-005, RULE-002, RULE-005 | AC-008, AC-009, AC-010 | AT-006, AT-007, UT-009, IT-005, IT-006, IT-007 |
| Resolver cutover | AppView GET resolver plus Flutter repository/model/policy and generic-row resolution | Delete all resolution-only surfaces and route policy; make generic/unknown rows inert | FR-001, FR-011, FR-014, FR-015 | AC-011, AC-014, AC-015 | AT-003, AT-009, IT-008, IT-011, REG-008 |
| Privacy/observability | Redacted identifier wrappers, safe endpoint categories, classification-only observers | Keep facts out of strings, logs, Sentry, analytics, metrics, feedback, and snapshots; add sentinel coverage | NFR-002, RULE-001 | AC-017 | UT-010, IT-010, REG-007 |
| Unchanged notification behavior | Durable eligibility/newness/list/seen/preferences and provider delivery suites | Run unchanged as regressions; only provider routing metadata and resolver behavior change | RULE-006 | AC-020 | REG-001, REG-002, REG-003, REG-004, REG-005, REG-006, REG-010 |
| Visible notification wording | OS copy hard-codes `post`; in-app rows hard-code `post`/`replied` | Classify root post/direct comment/nested reply from indexed or hydrated reply structure and use the same vocabulary on both surfaces | FR-020 | AC-023 | AT-010, UT-016, UT-017, IT-012 |
| In-app row context | Rows show only action copy and optional post text | Add a Bluesky-style actor column with avatar above the bold name, share outlined category icons with notification settings, add compact relative time, return a display-ready avatar, and retain a DID/CID compatibility fallback | FR-021 | AC-024 | UT-018, UT-020, UT-021 |
| Follow-row relationship action | Follow activity is navigational only and the response has no current viewer-to-actor relationship | Project `viewerIsFollowing` in the existing actor response and add a compact optimistic Follow/Unfollow control backed by the profile repository | FR-022 | AC-025 | UT-022, UT-023, IT-014 |

## 4. Files And Modules

Generated `.g.dart` and localization files are regenerated from their source and are not hand-edited.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/push/sender.go` | Change | Remove provider-facing `NotificationID`; add typed canonical `ActorDID`, `SourceURI`, and `SubjectURI` routing facts to `SendRequest` | FR-001, FR-002, FR-014, FR-017 | UT-011, IT-001, REG-008 |
| `appview/internal/push/sender.go`, `dispatcher.go` | Change | Carry a bounded internal target-content role derived from indexed root/parent structure; do not add a provider data key | FR-020 | IT-012 |
| `appview/internal/push/dispatcher.go` | Change | Select canonical facts from `notification_events` during the existing claim and pass them to the sender without altering leases/retries/TTL | FR-002, FR-017, RULE-006 | IT-001, REG-002 |
| `appview/internal/push/payload.go` | Change | Build the exact version 1 flat data map and preserve generic visible copy | FR-001, FR-002, FR-004, FR-014, NFR-003, RULE-001 | UT-011, UT-012, IT-002, REG-001 |
| `appview/internal/push/payload.go`, `payload_test.go`, `firebase_sender_test.go` | Change | Select content-free role-aware visible copy for post/comment/reply targets while preserving the exact data contract | FR-020 | UT-017, IT-012 |
| `appview/internal/push/firebase_sender.go` | Change | Pass canonical facts to `BuildPayload`; keep provider result classification and platform TTL/sound behavior unchanged | FR-001, FR-002, RULE-006 | IT-002, REG-002 |
| `appview/internal/push/payload_test.go` | Change | Cover every category's exact key matrix, forbidden keys/copy, public-fact bounds, and maximum reply payload | FR-001, FR-002, FR-004, FR-014, FR-017, NFR-003, RULE-001 | UT-011, UT-012, IT-002, REG-001 |
| `appview/internal/push/dispatcher_test.go`, `firebase_sender_test.go` | Change | Assert durable fact projection, serialized FCM data, and unchanged delivery behavior | FR-002, FR-017, RULE-006 | IT-001, IT-002, REG-002 |
| `appview/internal/api/notification_resolution.go` | Delete | Remove the owner-scoped resolution handler, DTOs, interface, store query, and resolution-only visibility helpers | FR-015 | IT-008, REG-008 |
| `appview/internal/api/notification_resolution_test.go` | Delete | Remove obsolete resolver behavior tests; destination authorization remains in post/profile API suites | FR-008, FR-015 | IT-005, IT-008 |
| `appview/internal/api/notification_store.go`, `notifications.go`, `notifications_test.go` | Change | Select actor avatar MIME with the existing notification query and add the canonical display-ready avatar URL to the actor response | FR-021 | UT-020 |
| `appview/internal/api/notification_store.go`, `notifications.go`, `notifications_test.go`, `durable_notification_store_test.go` | Change | Select the viewer-to-actor follow relationship in the existing notification query and expose it as additive camelCase actor state | FR-022 | UT-022, IT-014 |
| `appview/internal/routes/routes.go`, `policy.go` | Change | Unregister `GET /v1/notifications/{notificationId}` and remove its route policy | FR-015 | IT-008, REG-008 |
| `appview/internal/routes/routes_test.go` | Change | Assert the former GET path is absent and no matching route policy remains | FR-015 | IT-008, REG-008 |
| `appview/internal/api/post_test.go`, `profile_test.go` | Change | Add/retain direct-identifier visibility, moderation, named-not-found, and unauthorized read assertions | BR-002, FR-008, RULE-002, RULE-005 | IT-005 |
| `appview/internal/observability/push_integration_test.go` | Change if needed | Extend identifier sentinels to actor/source/subject routing facts without exposing raw values | NFR-002, RULE-001 | IT-010 |
| `app/lib/notifications/models/notification_open_event.dart` | Change | Define non-null `NotificationOpenAttempt`, independent binding/fact results, typed facts, strict allowlist/version/category parsing, source, and redacted string output | FR-005, FR-011, FR-012, FR-014, FR-019, NFR-002 | UT-001, UT-002, UT-003, UT-010, REG-009 |
| `app/lib/notifications/models/notification_destination.dart` | Create | Hold provider-neutral Notifications/profile/post-with-optional-focus destinations and optional unable-to-open feedback | FR-003, FR-007, FR-012, FR-016 | UT-004, UT-005, UT-014 |
| `app/lib/notifications/models/foreground_notification_event.dart` | Change | Carry the structured open attempt and expose only safe outcome/source classes in diagnostics | FR-013, NFR-002 | AT-004, UT-010, IT-004 |
| `app/lib/notifications/services/firebase_notification_service.dart` | Change | Convert every foreground/background/initial callback to the same structured attempt; stop filtering malformed facts before binding policy | FR-013, NFR-004 | AT-004, IT-004, UT-015 |
| `app/lib/notifications/services/notification_service.dart` | Change | Update provider-neutral stream/initial-open types from event to attempt; retain the fakeable Firebase boundary | FR-013, NFR-004 | AT-004, UT-015 |
| `app/lib/notifications/services/notification_destination_inference.dart` | Create | Pure category/fact-to-destination mapping and invalid/unknown fallback classification | FR-002, FR-003, FR-011, FR-012, FR-016, FR-019, RULE-004 | AT-001, AT-003, AT-005, UT-004, UT-005 |
| `app/lib/notifications/services/notification_open_coordinator.dart` | Change | Apply secure current-DID binding before inference/fallback, then emit one outcome without resolver/network work | BR-001, BR-002, FR-006, FR-007, NFR-001, RULE-003 | AT-001, AT-002, UT-006, UT-007, IT-003, IT-004 |
| `app/lib/notifications/services/pending_notification_open.dart` | Change | Store the latest structured attempt during transient readiness and clear it on `requiresSignIn` | FR-018 | AT-008, UT-008, IT-003 |
| `app/lib/notifications/services/notification_runtime.dart` | Change | Remove resolution repository/error handling; wire binding loader, inference outcome, and existing effects directly | BR-001, FR-007, FR-013, FR-018, NFR-001 | AT-001, AT-004, AT-008, IT-003, IT-004, REG-010 |
| `app/lib/notifications/services/notification_navigation.dart` | Change | Construct typed profile/post/Notifications routes from inferred destinations; pass reply `focus` | FR-003, FR-007, FR-016 | AT-001, AT-005, UT-014, IT-009 |
| `app/lib/notifications/models/notification_effect.dart`, `app/lib/notifications/widgets/notification_effect_host.dart` | Change | Carry inferred outcomes and present quiet/feedback fallbacks through the existing single root host | FR-007, FR-012, FR-013 | AT-001, AT-003, AT-004, IT-003, IT-004 |
| `app/lib/notifications/providers/notification_runtime_provider.dart` | Change | Remove the resolution-provider dependency; retain one keep-alive runtime owner and effect stream | FR-007, FR-013, NFR-004 | IT-003, IT-004, UT-015 |
| `app/lib/notifications/data/notification_repository.dart`, `api_notification_repository.dart` | Change | Remove `NotificationResolutionRepository`, imports, and GET call; retain list/device/newness/preferences interfaces | FR-015 | IT-008, REG-004, REG-008 |
| `app/lib/notifications/providers/notification_repository_provider.dart`, generated `.g.dart` | Change / Generate | Remove the resolution provider and regenerate Riverpod output | FR-015 | IT-008, REG-008 |
| `app/lib/notifications/models/notification_resolution.dart`, `notification_id.dart` | Delete | Remove resolution-only wire models and notification-ID wrapper; durable feed row IDs remain strings in `CraftskyNotification` | FR-014, FR-015 | IT-008, REG-008 |
| `app/lib/notifications/services/notification_resolution_policy.dart` | Delete | Remove server-resolution destination/failure policy replaced by local fact inference and destination-page error handling | FR-003, FR-015 | UT-004, UT-005, IT-008 |
| `app/lib/notifications/widgets/notification_row.dart` | Change | Give generic/unknown rows `onTap: null`; preserve known typed navigation and unavailable-row feedback | FR-015, RULE-006 | AT-009, UT-013, IT-011, REG-005 |
| `app/lib/notifications/widgets/notification_row.dart`, `app/lib/l10n/app_en.arb` | Change / Generate | Classify hydrated targets as post/comment/reply and render localized role-aware like/repost/response copy | FR-020 | AT-010, UT-016 |
| `app/lib/notifications/widgets/notification_row.dart`, `app/lib/notifications/models/craftsky_notification.dart` | Change | Prefer the display-ready actor avatar, derive a canonical public DID/CID fallback, and render the avatar above the bold actor name with compact relative time | FR-021 | UT-018, UT-020, UT-021 |
| `app/lib/notifications/widgets/notification_row.dart`, `app/lib/notifications/models/craftsky_notification.dart`, generated mapper | Change / Generate | Decode initial follow state and render a compact follow-row-only button using the existing localized profile labels and profile repository mutation, with optimistic rollback and profile-cache invalidation | FR-022 | UT-023 |
| `app/lib/notifications/widgets/notification_category_icon.dart`, `app/lib/notifications/pages/notification_settings_page.dart` | Create / Change | Centralize the outlined category icon matrix and reuse it in both settings and notification rows so the two surfaces cannot drift | FR-021 | UT-018 |
| `app/lib/shared/time/relative_time_text.dart`, `app/lib/feed/widgets/post_card.dart` | Create / Change | Extract the post-card relative timestamp into a shared widget with the existing compact format and full localized tooltip | FR-021 | UT-018, REG-004 |
| `app/lib/shared/errors/notification_destination_error.dart` | Create | Purely classify permanent, transient, and authentication-loss destination errors without recording identifiers | FR-009, FR-010, NFR-002 | AT-006, AT-007, UT-009, IT-006, IT-007 |
| `app/lib/shared/widgets/notification_destination_error_state.dart` | Create | Shared localized/accessibility surface for permanent Back/View notifications and transient Retry; auth loss renders no notification-specific state | FR-009, FR-010, NFR-005 | AT-006, AT-007, IT-006, IT-007 |
| `app/lib/feed/pages/post_thread_page.dart` | Change | Pass the actual provider error to the shared state; make permanent errors take precedence over `_lastSection` so cached content is hidden; keep route/focus in place | FR-009, FR-010, RULE-005 | AT-006, AT-007, IT-006, IT-007, IT-009 |
| `app/lib/profile/widgets/profile_page_error.dart`, `app/lib/profile/pages/profile_page.dart` | Change | Use the shared permanent/transient/auth destination state while preserving the existing profile provider/API | FR-008, FR-009, FR-010, NFR-005 | AT-006, AT-007, IT-005, IT-006, IT-007 |
| `app/lib/l10n/app_en.arb`, generated localization files | Change / Generate | Add unable-to-open feedback, permanent-unavailable explanation, Back, and View notifications labels; reuse existing Retry where appropriate | FR-009, FR-010, FR-012, NFR-005 | AT-003, AT-006, AT-007, IT-006, IT-007 |
| `app/test/notifications/models/notification_open_event_test.dart` | Change | First structured-attempt red test plus exact parser/fact/bounds/privacy matrix | FR-001, FR-002, FR-005, FR-006, FR-011, FR-012, FR-014, FR-019, NFR-002 | UT-001, UT-002, UT-003, UT-010, REG-009 |
| `app/test/notifications/services/notification_destination_inference_test.dart` | Create | Cover every category, invalid/legacy feedback fallback, quiet unknown fallback, and ignored extras | FR-002, FR-003, FR-011, FR-012, FR-016, FR-019, RULE-004 | UT-004, UT-005, AT-003 |
| `app/test/notifications/providers/notification_open_coordinator_test.dart` | Change | Prove binding-first gating and immediate resolver-free outcome emission | BR-001, BR-002, FR-006, FR-007, NFR-001, RULE-003 | AT-002, UT-006, UT-007 |
| `app/test/notifications/notification_open_flow_test.dart`, `notification_effect_host_test.dart`, `services/pending_notification_open_test.dart`, `services/notification_runtime_lifecycle_test.dart` | Change | Exercise unified callback sources, readiness, latest-only/sign-in discard, effect presentation, and at-least-once behavior | FR-013, FR-018, RULE-006 | AT-001, AT-003, AT-004, AT-008, IT-003, IT-004, REG-010 |
| `app/test/router/notification_open_routing_test.dart` | Create | Verify exact typed route paths and reply focus with no arbitrary URL execution | FR-003, FR-007, FR-016 | AT-001, AT-005, UT-014, IT-009 |
| `app/test/shared/errors/notification_destination_error_test.dart` | Create | Cover named permanent 404s, transient failures, and 401 classification | FR-009, FR-010 | UT-009 |
| `app/test/feed/pages/post_thread_page_test.dart`, `app/test/profile/profile_page_test.dart` | Create / Change | Widget acceptance for permanent actions, transient Retry, 401, no stale content, and no redirect | BR-002, FR-008, FR-009, FR-010, NFR-005, RULE-002, RULE-005 | AT-006, AT-007, IT-005, IT-006, IT-007 |
| `app/test/notifications/notifications_page_test.dart` | Change | Assert generic/unknown rows have no tap semantics while known/unavailable behavior remains | FR-015, RULE-006 | AT-009, UT-013, IT-011, REG-005 |
| `app/test/notifications/data/api_notification_repository_test.dart`, `app/test/notifications/providers/notification_repository_provider_test.dart` | Change | Remove resolver request/provider cases and retain coverage for list/device/newness/preferences repository seams | FR-015, RULE-006 | IT-008, REG-004, REG-008 |
| `app/test/notifications/notification_architecture_test.dart`, `app/test/shared/errors/sentry_redaction_test.dart` | Change | Prove Firebase isolation, resolver absence, no notification ID in provider opens, no new persistence, and identifier-free diagnostics | FR-014, FR-015, NFR-002, NFR-004 | UT-010, UT-015, IT-008, IT-010, REG-007, REG-008, REG-010 |
| Obsolete Flutter resolution tests and fakes | Delete / Change | Remove `notification_resolution_policy_test.dart`, `notification_id_test.dart`, resolution repository cases/overrides, and resolution fakes | FR-014, FR-015 | IT-008, REG-008 |

## 5. Services, Interfaces, And Data Flow

### AppView payload projection

The database schema already stores every required canonical value, so no migration is needed. Interaction notification activation records the subject post's canonical root, and the claim query selects `n.actor_did`, `n.source_uri`, nullable `n.subject_uri`, and nullable `n.root_uri`; unrelated parent/quoted references are not passed to the provider builder.

```text
// Partial Go shapes only.
type RoutingFacts struct {
    ActorDID   syntax.DID
    SourceURI  syntax.ATURI
    SubjectURI syntax.ATURI // zero value when the category does not use it
    RootURI    syntax.ATURI // canonical thread root for like/repost
}

type SendRequest struct {
    Token                 string
    Category              notifications.Category
    AccountSubscriptionID string
    RoutingFacts          RoutingFacts
    ActorDisplayName      string
    Platform              string
    TTL                   time.Duration
}

func BuildPayload(
    category notifications.Category,
    routingID string,
    actorDisplayName string,
    facts RoutingFacts,
) Payload
```

`BuildPayload` starts with only the common map and then adds exactly one category branch:

| Category | Added fact keys |
|---|---|
| `follow` | `actorDid` |
| `like`, `repost` | `subjectUri`, `rootUri` |
| `mention`, `quote` | `sourceUri` |
| `reply` | `subjectUri`, `sourceUri` |
| `everythingElse` | None |

Common keys are exactly `payloadVersion=1`, `type`, and `accountSubscriptionId`. The builder never accepts a final route/path or arbitrary data map. `notification_id` remains in the durable `push_deliveries` foreign key but leaves the in-memory provider request and FCM data contract.

The Flutter and Go contract use the same declared bounds in tests: existing 64-character `type`, existing 128-character account-subscription ID, and at most 1024 ASCII bytes for each public DID/AT-URI fact. The maximum reply data map is asserted below the provider's existing message-data budget. A future bound change is a payload-contract change and must update both suites.

### Flutter provider-neutral attempt

The parser always returns a structured attempt for a provider open. It never returns `null` merely because binding or fact data is malformed.

```text
// Partial Dart shapes only.
final class NotificationOpenAttempt {
  final AccountSubscriptionId? accountSubscriptionId; // null = invalid binding
  final NotificationFactOutcome facts;
  final NotificationOpenSource source;

  static NotificationOpenAttempt fromProviderData(
    Map<String, Object?> data, {
    required NotificationOpenSource source,
  });
}

sealed class NotificationFactOutcome {}
final class ValidNotificationFacts extends NotificationFactOutcome {
  // Private/category factories preserve the required-field matrix.
}
final class UnknownNotificationFacts extends NotificationFactOutcome {}
final class InvalidNotificationFacts extends NotificationFactOutcome {
  final NotificationFactFailureClass failureClass; // bounded enum only
}
```

Parsing rules:

- Read only `payloadVersion`, `type`, `accountSubscriptionId`, `actorDid`, `subjectUri`, `rootUri`, and `sourceUri`; ignore every other key.
- Parse the account-subscription ID independently, even when version/type/facts are invalid.
- Require exact version string `1` and the existing bounded type identifier syntax.
- For known categories, parse only required facts. A valid DID uses the existing atproto DID validator. A post fact must be an `at://` URI in `social.craftsky.feed.post` with a valid DID authority and record key. Enforce the contract bounds before constructing typed values.
- An unknown syntactically valid type becomes `UnknownNotificationFacts`; its extra fields are never parsed or used.
- Missing/unsupported version, malformed type, and malformed/missing known-category facts become `InvalidNotificationFacts` with a bounded class, never an exception containing raw input.
- `notificationId` is ignored as an unknown extra and never enters the attempt.

### Binding-first inference and navigation

```text
provider callback
  -> NotificationOpenAttempt.fromProviderData
  -> PendingNotificationOpen (latest only while transient)
  -> NotificationOpenCoordinator
       -> load secure binding for current DID
       -> invalid/missing/mismatch: generic unavailable effect, stop
       -> match: NotificationDestinationInference.forFacts(attempt.facts)
  -> NotificationNavigationEffect
  -> NotificationEffectHost
  -> typed GoRouter route
  -> authenticated AppView destination provider
```

```text
// Partial Dart signatures only.
final class NotificationOpenCoordinator {
  Future<void> open(NotificationOpenAttempt attempt);
}

abstract final class NotificationDestinationInference {
  static NotificationOpenOutcome forFacts(NotificationFactOutcome facts);
}

sealed class NotificationDestination {}
final class NotificationsDestination extends NotificationDestination {}
final class ProfileDestination extends NotificationDestination {
  final Did did;
}
final class PostDestination extends NotificationDestination {
  final AtUri subjectUri;
  final AtUri? focusUri;
}
```

Inference is pure and has no repository, Dio, Firebase, context, or GoRouter dependency. `everythingElse` returns Notifications quietly. `UnknownNotificationFacts` also returns Notifications quietly. `InvalidNotificationFacts` returns Notifications plus `unableToOpen`. Only the effect host presents feedback and executes typed routes.

### Destination error boundary

The push facts choose what the app asks to open; they do not authorize or hydrate content. Existing `postCommentSectionProvider` and `userProfileProvider` still call the authenticated AppView repositories.

```text
enum NotificationDestinationErrorKind {
  permanentUnavailable,
  retryable,
  authenticationLost,
}

NotificationDestinationErrorKind classifyNotificationDestinationError(
  Object error,
);
```

- `ApiBadRequest` with `post_not_found` or `profile_not_found` is permanent.
- `ApiNetworkError`, `ApiServerError` (including `502 identity_unavailable`), and unexpected load failures are retryable in place.
- `ApiUnauthorized` is authentication loss. The existing global interceptor owns sign-out; the destination renders no notification-specific unavailable/retry state while router/auth redirection completes.

The classifier and shared widget receive only exception classes and bounded AppView error codes. They never stringify route identifiers or raw payloads.

### Resolver removal

The cutover deletes rather than deprecates:

- AppView resolver DTO/interface/handler/store query/visibility helpers, route registration, route policy, and resolver tests.
- Flutter resolution DTOs, notification ID wrapper, repository interface/method/provider, resolution policy, runtime dependency, generic-row resolution, tests, and fakes.

Historical workflow documents are not rewritten. The current approved change folder and resulting code/tests become the source of truth for the replacement contract.

## 6. State, Providers, Controllers, Or DI

No new Riverpod provider is needed. Parsing, inference, and error classification remain pure services/functions.

```text
notificationServiceProvider (Firebase adapter behind interface)
  -> notificationRuntimeProvider (keepAlive, one owner)
       -> notificationRoutingStorageProvider
       -> NotificationRegistrationCoordinator (unchanged)
       -> PendingNotificationOpen
       -> NotificationOpenCoordinator (constructed per ready open)
       -> _notificationEffectControllerProvider
  -> notificationEffectStreamProvider
       -> one NotificationEffectHost
       -> typed route navigation / messenger feedback
```

Provider changes:

- Remove `notificationResolutionRepositoryProvider` and regenerate `notification_repository_provider.g.dart`.
- Remove the resolution repository argument from `NotificationRuntime` and `notificationRuntimeProvider`.
- Keep `notificationRuntimeProvider`, effect controller/stream, service provider, routing storage provider, list provider, new-count provider, preferences providers, and registration wiring otherwise unchanged.
- Keep `PendingNotificationOpen` as a plain in-memory object owned by the runtime, not a provider or persisted store.
- Keep destination providers unchanged so normal route entry and notification route entry share the same authenticated reads/cache behavior.

Readiness behavior stays:

- `requiresSignIn`: clear the pending attempt and process nothing.
- `transient`: retain/replace one attempt.
- `ready`: process the current attempt immediately or release the latest pending attempt once.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Notification-open route mapping

| Fact outcome | Typed route behavior |
|---|---|
| Follow `actorDid` | `UserProfileRoute(handle: actorDid.toString()).push` |
| Like/repost `subjectUri` + `rootUri` | Root DID/rkey in `PostThreadRoute`; differing subject URI in `focus` |
| Mention/quote `sourceUri` | Parse DID/rkey and `PostThreadRoute(...).push` |
| Reply `subjectUri` + `sourceUri` | Subject DID/rkey in `PostThreadRoute`; source URI in `focus` |
| Everything else | `NotificationsRoute().go` |
| Unknown valid type | `NotificationsRoute().go`, no feedback |
| Invalid/legacy facts after binding match | `NotificationsRoute().go`, brief unable-to-open feedback |
| Invalid/mismatched binding | No route; generic unavailable feedback only |

No literal URL is executed. Post URIs are decomposed into typed DID/record-key/focus arguments before GoRouter sees them. The existing root effect host remains the only UI navigation owner for provider opens.

### Permanent and transient destination states

The shared `NotificationDestinationErrorState` is embedded in both the post-thread and profile scaffolds so the intended route and app bar remain visible.

- Permanent state: localized title/body, explicit Back action, explicit View notifications action, no Retry, no stale post/profile/payload content, and no automatic redirect.
- Transient state: localized safe message plus Retry, no persisted/automatic notification-open retry, and no route change.
- Authentication loss: no notification-specific state; existing global sign-out/router behavior takes over.
- In the post thread, a permanent error takes precedence over any `_lastSection` cache. A transient refresh may preserve already authenticated server content, but its Retry still re-runs only the destination provider.
- Back uses the current navigator when possible and falls back to the normal app home when there is no prior route. View notifications uses the typed `NotificationsRoute`.
- Buttons expose localized text labels and standard accessible button semantics/tap targets.

### Notification feed rows

- Known hydrated follow/post/reply rows keep their existing typed routes and reply focus behavior.
- `GenericNotification`, including unknown categories, renders its informational copy with `ListTile.onTap == null` and no tap semantics.
- `UnavailableNotification` retains its explicit warning feedback; it is not silently converted into a generic row.

No route path is added or changed. Only the AppView resolution route is removed.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Valid known facts, matching binding, ready app | Emit inferred typed navigation immediately; destination provider may still be loading | BR-001, FR-007, NFR-001 | AT-001, UT-007, IT-003 |
| Missing/malformed payload binding | Preserve fact result if parseable, but binding gate emits generic unavailable feedback and no route/network call | FR-006, RULE-003 | AT-002, UT-001, UT-006, IT-004 |
| Missing local or mismatched/stale binding | Same binding rejection; Notifications fallback is also forbidden | BR-002, FR-006, RULE-003 | AT-002, UT-006, IT-004, MAN-004 |
| Missing/unsupported/malformed version with valid binding | Open Notifications with unable-to-open feedback; ignore legacy notification ID | FR-011, FR-012 | AT-003, UT-001, UT-005, IT-003 |
| Malformed known category fact with valid binding | Open Notifications with unable-to-open feedback | FR-005, FR-012, FR-019 | AT-003, UT-002, UT-003, UT-005 |
| Unknown valid bounded type with extras | Ignore extras and open Notifications quietly | FR-012, RULE-004 | AT-003, UT-002, UT-005 |
| Valid known facts with extras | Ignore extras and use only the category-required facts | FR-019 | AT-003, UT-002, REG-009 |
| Multiple opens during transient readiness | Replace pending attempt; release only the latest once ready | FR-018 | AT-008, UT-008, IT-003 |
| Pending open reaches `requiresSignIn` | Clear permanently; later same-account sign-in does not revive it | FR-018 | AT-008, UT-008, IT-003 |
| Duplicate provider callbacks | Process each callback once under existing at-least-once behavior; add no dedupe store | FR-013, RULE-006 | AT-004, REG-010 |
| `404 post_not_found` / `profile_not_found` | Stay on intended route; permanent unavailable UI with Back/View notifications; no stale content/redirect | FR-009, NFR-005, RULE-005 | AT-006, UT-009, IT-006, MAN-003 |
| Network, `5xx`, or `502 identity_unavailable` | Stay on route and show Retry; retry only the destination provider when tapped | FR-010, NFR-005 | AT-007, UT-009, IT-007 |
| `401` | Existing global sign-out runs; no notification-specific unavailable/retry UI remains | FR-010 | AT-007, UT-009, IT-007, REG-006 |
| Generic/unknown feed row | Informational `ListTile` with no tap callback or semantics action | FR-015 | AT-009, UT-013, IT-011 |
| Known unavailable feed row | Preserve explicit unavailable warning behavior | RULE-006 | IT-011, REG-005 |
| Over-bound DID/AT-URI or non-post URI/arbitrary URL | Invalid facts; after a valid binding, safe Notifications fallback with feedback | FR-005, FR-012, NFR-003 | UT-003, UT-005, REG-009 |
| Fact identifiers in errors/telemetry | Emit only bounded outcome/category classes; never stringify payload values | NFR-002, RULE-001 | UT-010, IT-010, REG-007 |
| Old development AppView/Flutter pairing | No compatibility path; malformed side falls back safely and developers update/reset both components | FR-011, FR-015 | IT-008, REG-008 |

## 9. Test Implementation Plan

Tests run in strict red-green-refactor order. Provider-neutral Flutter tests use fakes and never initialize Firebase; AppView projection tests use the existing recording sender and Postgres fixture patterns.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | UT-001 | `app/test/notifications/models/notification_open_event_test.dart` | Common fields with valid/invalid independent binding and fact states; legacy ID sentinel | Current nullable parser discards the whole event and still requires notification ID |
| 2 | UT-002, UT-003, REG-009 | Same parser suite | Exact category matrix, extras, valid/invalid DID and post AT-URI, arbitrary URLs, bounds | Current event has no versioned facts or typed category invariants |
| 3 | UT-006, AT-002 | `notification_open_coordinator_test.dart` | Matching/missing/malformed/local-missing/stale bindings with route/network spies | Coordinator assumes a valid binding object and proceeds to resolver |
| 4 | UT-004, UT-005, AT-003 | New destination-inference suite | All valid facts, invalid/legacy outcomes, quiet unknown, extras | No Flutter-owned inference or post-binding fallback policy exists |
| 5 | UT-008, AT-008 | `pending_notification_open_test.dart`, open-flow suite | Ordered attempts across transient/ready/requires-sign-in/same-DID sign-in | Pending slot stores old event shape; flow does not prove permanent sign-in discard with new outcomes |
| 6 | UT-011, UT-012 | `appview/internal/push/payload_test.go` | One distinct sentinel fixture per category plus maximum reply values | Payload still emits notification ID and no canonical facts/version |
| 7 | IT-001 | `appview/internal/push/dispatcher_test.go` | Durable event rows with distinct actor/source/subject/unrelated refs and recording sender | Claim/send request does not carry canonical routing facts |
| 8 | IT-002, REG-001 | `payload_test.go`, `firebase_sender_test.go` | Captured serialized provider messages for all categories | FCM data shape is old and exact-minimum/size assertions fail |
| 9 | UT-007, IT-003, AT-001 | Coordinator/runtime/open-flow suites | Matching valid attempt, route spy, incomplete destination future, no resolver | Runtime is serially coupled to resolver HTTP |
| 10 | IT-004, AT-004, REG-010 | Open-flow/effect-host/runtime lifecycle suites | Equivalent foreground/background/initial attempts, duplicates, matching/mismatched bindings | Callback sources still depend on nullable parsing/resolution fakes |
| 11 | UT-014, AT-005, IT-009 | New router suite plus comment-section suite | Subject post URI and source reply focus URI | Destination has no focus and no local reply mapping |
| 12 | UT-009 | New shared destination-error suite | Named 404s, network, representative 5xx, 502 identity unavailable, 401 | Existing generic error mapping does not express the required three-way destination policy |
| 13 | IT-005 | AppView post/profile API suites | Visible, hidden, taken-down, unauthorized, and deleted identifiers | Any missing direct-identifier moderation/not-found coverage is exposed |
| 14 | AT-006, IT-006 | New post-thread widget suite and profile suite | Permanent named errors, route harness, action spies | Post only retries; profile has generic Retry; neither has shared permanent actions |
| 15 | AT-007, IT-007, REG-006 | Destination widget suites and 401 interceptor suite | Transient-then-success repositories and 401 auth flow | Error pages do not share explicit transient/auth semantics |
| 16 | UT-013, AT-009, IT-011, REG-005 | `notifications_page_test.dart` | All known categories, generic, unknown, unavailable | Generic rows still invoke resolution and expose tap semantics |
| 17 | IT-008, REG-008 | AppView routes plus Flutter architecture/repository suites | Former GET path, policy registry, static source/provider scan | Resolver route/client types/providers/calls remain present |
| 18 | UT-010, IT-010, REG-007 | Parser, push observability, architecture, and Sentry-redaction suites | Unique token/binding/notification/DID/URI/focus/payload/provider-error sentinels | New fact fields are not yet covered by redaction guards |
| 19 | UT-015 | Flutter notification architecture suite | Import/source scan and provider-neutral fakes | New parser/inference seams may accidentally import Firebase/UI/network types |
| 20 | REG-002 | Existing dispatcher/retry/firebase sender suites | Current retry, fencing, cancellation, invalid-token, TTL fixtures | Any accidental delivery-semantic drift fails existing assertions |
| 21 | REG-003 | Existing notification lifecycle/index suites | Existing eligibility, preferences, coalescing, snapshot fixtures | Any unintended server lifecycle change fails |
| 22 | REG-004 | AppView list/newness and Flutter list/count/seen/badge suites | Existing durable/list/render/newness fixtures | Any state regression outside open routing fails |
| 23 | MAN-001, MAN-002, MAN-003, MAN-004, MAN-005 | Physical Android/iOS and debug request trace | Real eligible events, stale targets, multi-account retained OS notification | Provider/OS lifecycle and qualitative request order remain unproven by host tests |
| 24 | UT-016 | `app/test/notifications/notifications_page_test.dart` | Hydrated root post, direct comment, and nested reply fixtures | Current row copy calls every like/repost target a post and every response a reply to a post |
| 25 | UT-017 | `appview/internal/push/payload_test.go` | Category and target-role visible-copy matrix | Current OS body is selected only from category and hard-codes post wording |
| 26 | IT-012 | `appview/internal/push/dispatcher_test.go` | Durable events whose target rows have root/direct/nested reply structure | Dispatcher does not project a target role into the send request |

Focused commands by phase:

```text
# First TDD step, from app/
flutter test test/notifications/models/notification_open_event_test.dart

# Provider-neutral notification loop, from app/
flutter test test/notifications

# Notification routes and destination UI, from app/
flutter test test/notifications test/feed/pages/post_thread_page_test.dart test/feed/pages/post_comment_section_page_test.dart test/profile/profile_page_test.dart test/router

# Generate changed Riverpod/localization outputs, from app/
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n

# Focused analysis, from app/
dart analyze lib/notifications lib/feed/pages/post_thread_page.dart lib/profile lib/shared/errors lib/shared/widgets test/notifications test/feed/pages/post_thread_page_test.dart test/feed/pages/post_comment_section_page_test.dart test/profile/profile_page_test.dart test/router

# AppView cutover, from appview/
go test ./internal/push ./internal/api ./internal/routes -count=1

# Canonical repository verification, from repository root
just app-test
just app-analyze
just test
git diff --check
```

Manual checks start only after automated verification is green and real FCM/APNs configuration is available.

## 10. Sequencing And Guardrails

- First TDD step: write UT-001 for the structured open-attempt boundary, proving that valid binding state survives invalid/legacy facts and invalid binding state survives otherwise valid facts.
- Dependencies between work items:
  - Structured parsing precedes binding policy, inference, Firebase callback updates, and runtime wiring.
  - Binding-gate tests precede every route/fallback test.
  - Pure inference precedes typed router integration.
  - Latest-only readiness behavior is adapted before callback-source integration.
  - AppView payload unit tests precede dispatcher/sender changes.
  - Both new payload producer and new Flutter consumer are green before deleting the resolver path.
  - Error classification precedes post/profile widget changes.
  - Generic-row behavior changes in the same resolver-removal slice.
  - Privacy/architecture/regression suites run after the behavioral slices, then canonical verification, then manual checks.
- Guardrails:
  - Parse provider data once at the Firebase-to-domain boundary; trust only the resulting typed values internally.
  - Preserve account-subscription binding validity independently from fact validity.
  - Require the current DID's exact secure binding before every navigation, Notifications fallback, or fact-feedback outcome.
  - Never await notification-specific or destination network work between a successful binding match and navigation effect emission.
  - AppView destination reads remain authenticated and authoritative for visibility/moderation; payload facts are never capabilities.
  - Only allow `payloadVersion`, bounded type/binding, and the exact category-required fact keys. Ignore all extras and never accept a literal route or URL.
  - Keep route facts to ASCII and the documented bounds; keep the data map flat and below the tested provider budget.
  - Use typed `syntax.DID`/`syntax.ATURI` in new Go internal fields and typed `Did`/validated Craftsky post `AtUri` in Flutter.
  - Never log, report, analyze, label metrics with, stringify, or show tokens, account-subscription IDs, notification IDs, DIDs, AT-URIs, focus URIs, raw payloads, credentials, or provider errors.
  - Keep Firebase imports/listeners confined to the existing adapter. The runtime, parser, inference, coordinator, navigation outcome, and tests stay provider-neutral.
  - Keep exactly one runtime owner and one root effect host; do not add another listener, queue, receipt store, timer, or dedupe layer.
  - Preserve callback at-least-once behavior and latest-only transient readiness semantics.
  - Keep visible push copy content-free and bounded; only FR-020's post/comment/reply wording may change. Preserve preference/eligibility/coalescing/newness/seen/list/badge/sound/TTL/retry/invalid-token behavior.
  - Delete resolver producer and consumer surfaces together; do not leave a compatibility route or hidden fallback call.
  - Do not delete `notification_events.id`, `push_deliveries.notification_id`, or durable notification-feed row IDs; only remove the provider-open resolution identifier.
  - Do not modify migrations, lexicons, PDS writes, route paths other than removing the resolver, dependencies, native projects, or provider credentials.
  - Regenerate Riverpod and localization outputs; never hand-edit generated files.
  - Preserve unrelated dirty-worktree changes. This coding-plan stage edits only `04-coding-plan.md`.
- Out of scope:
  - Old-client/provider compatibility, staged rollout, data backfill, or migration.
  - Universal/App Links, public custom URL schemes, literal deep links, encrypted route capsules, or key management.
  - Persisted opens, automatic open retries, notification receipt deduplication, or numeric latency instrumentation.
  - Eligibility, preferences, coalescing, newness, seen, badge, sound, TTL, provider retry, token registration, or Firebase native setup changes.
  - PDS-direct reads, payload authorization, moderation-policy changes, lexicon changes, or new AppView endpoints.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Blocking workflow gate | High-risk source implementation has not yet been explicitly approved | `implement-tdd` must not edit source/tests until the user gives the implementation go-signal | Ask whether to continue with `implement-tdd` after this plan |
| CPQ-002 | Resolved | Binding and fact parsing previously failed as one nullable result | Could bypass required post-binding fallback or block binding-negative tests | Use one attempt with nullable parsed binding plus independent sealed fact outcome; UT-001 is first |
| CPQ-003 | Resolved | Exact internal fact transport from durable events | Could tempt a new table/query/API or leak unused references | Extend the existing dispatcher claim projection with actor/source/subject only; no migration |
| CPQ-004 | Resolved | Public fact bounds were required but not numerically specified | Unbounded values could exceed provider limits; inconsistent clients could disagree | Use 1024 ASCII bytes per DID/AT-URI fact plus the existing type/binding bounds and assert maximum reply map size on both sides |
| CPQ-005 | Resolved | Profile typed route parameter is named `handle` while push carries a DID | Could introduce a client identity lookup or new route | Pass the typed DID string to the existing route/API, which already accepts handle-or-DID |
| CPQ-006 | Resolved | Reply `subjectUri` currently represents the parent/subject post, not necessarily the root | Wrong interpretation could focus the reply in a different thread | Preserve the approved durable semantics: subject URI selects the thread target and source URI supplies focus; IT-009 asserts repository arguments |
| CPQ-007 | Residual | Physical FCM/APNs callback order and a truly stale accepted push cannot be proven in host tests | Automated green does not prove background/terminated device behavior | Complete MAN-001–MAN-005 before release readiness |
| CPQ-008 | Residual | Pre-cutover local AppView and Flutter builds become incompatible | Development taps may fall back until both components update | Intentional clean cutover; structural tests reject compatibility code and developers update/reset local builds |
| CPQ-009 | Residual | Direct navigation removes the server-selected alternate fallback | A deleted/hidden target now shows unavailable UI instead of another target | This is approved behavior; keep route visible with Back/View notifications and verify authorization at destination |
| CPQ-010 | Non-blocking | Existing generated Riverpod/localization outputs will change after provider/copy removal/addition | Source and generated files can drift if generation is skipped | Run build_runner and `flutter gen-l10n`, then analysis and drift checks during TDD |
| CPQ-011 | Non-blocking product follow-up | Numeric latency telemetry remains undecided | No percentile measurement for the improvement | Structural no-resolver proof plus MAN-005 is sufficient for this slice; add privacy-safe timing only in a separate approved change |

No unresolved product or architecture question blocks implementation. CPQ-001 is the explicit workflow approval gate.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md` will be created and maintained by `implement-tdd` after explicit approval.
- Start with test: UT-001 in `app/test/notifications/models/notification_open_event_test.dart`
- Focused command: from `app/`, `flutter test test/notifications/models/notification_open_event_test.dart`
- First production target after the red test: `app/lib/notifications/models/notification_open_event.dart`
- Next tests: UT-002/UT-003 parser matrix, then UT-006/AT-002 binding-first coordinator behavior.
- Notes:
  - Keep strict red-green-refactor order from Section 9.
  - Keep AppView and Flutter contract changes coordinated, but do not delete the resolver until the replacement producer/consumer tests are green.
  - Treat manual device checks as release-readiness evidence, not a substitute for automated gates.
  - Stop before source implementation unless the user explicitly continues with `implement-tdd`.
