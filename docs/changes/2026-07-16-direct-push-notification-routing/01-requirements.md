# Requirements: Direct Push Notification Routing

## 1. Initial Request

Replace the normal notification-open resolution round trip with minimal canonical notification facts carried in the push data payload. After validating that the push belongs to the currently authenticated account on this installation, Flutter should infer and construct a typed internal route immediately and let the destination screen handle deleted, hidden, unavailable, malformed, and offline cases gracefully.

The user confirmed that exposing public DIDs and AT-URIs in the provider data payload is acceptable. The earlier prohibition on those public identifiers is intentionally relaxed for this notification-fact use case because the latency and simpler normal open path are more valuable.

The app is not live. No production clients or previously delivered notification data require compatibility or preservation, so this change may make a clean cutover and remove the old notification-resolution behavior instead of carrying a legacy path.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/internal/push/payload.go` builds a combined notification-and-data FCM message containing `notificationId`, `type`, and `accountSubscriptionId`.
  - `appview/internal/push/dispatcher.go` loads notification and subscription data before handing a send request to the provider sender.
  - `appview/internal/api/notification_resolution.go` implements owner-scoped `GET /v1/notifications/{notificationId}` and selects a currently visible post, profile, or Notifications fallback.
  - `app/lib/notifications/models/notification_open_event.dart` parses the current provider-data allowlist into a provider-neutral open event.
  - `app/lib/notifications/services/notification_open_coordinator.dart` validates the current DID's locally stored account-subscription binding, then waits for AppView resolution before navigation.
  - `app/lib/notifications/services/notification_navigation.dart` converts resolved post/profile targets into typed GoRouter routes.
  - `app/lib/feed/pages/post_thread_page.dart` loads routed post content through AppView but currently presents one generic retry state for every load error.
- Existing patterns:
  - Provider data is untrusted and parsed through an allowlist before it enters domain code.
  - One keep-alive notification runtime owns foreground, background-open, and terminated-open handling.
  - The locally stored `accountSubscriptionId` is keyed by DID and is the cross-account/stale-push gate.
  - Post and profile routes load their data through authenticated AppView APIs; route parameters do not themselves authorize access to content.
  - Notification rows already navigate from hydrated DIDs and AT-URIs without performing notification-ID resolution for known categories.
- Current behavior:
  - A valid push open does not navigate until `GET /v1/notifications/{notificationId}` succeeds or produces a safe fallback.
  - AppView resolution verifies notification ownership and current target visibility, and may select alternate post/profile/list destinations.
  - Network latency for resolution is serial with the subsequent destination-page content request.
  - Retracted and unknown resolutions are mapped to Notifications by the Flutter policy even if the server retained other tombstone references.
- Constraints discovered:
  - The push is a flat string-to-string provider data map and must remain bounded.
  - Background and terminated opens must use the same provider-neutral domain path as foreground banner opens.
  - A push accepted by FCM cannot be recalled or rewritten after its source or destination changes.
  - The app is not live, so there are no supported old clients, accepted production pushes, or legacy notification-open data to preserve.
  - DIDs and AT-URIs are public identifiers, but their values and full payloads must still not be written to logs, Sentry, analytics, metrics labels, or user-facing diagnostics.
- Test/build commands discovered:
  - Flutter focused tests run from `app/` with `flutter test test/notifications` and linked router/feed/profile tests.
  - Flutter static analysis runs from `app/` with `dart analyze` or from the repository root with `just app-analyze`.
  - Focused AppView tests run from `appview/` with `go test ./internal/push ./internal/api`.
  - Canonical repository verification includes `just app-test`, `just app-analyze`, `just test`, and `git diff --check`.

## 3. Clarifying Questions And Decisions

### Q1: May a push data payload expose the destination DID or AT-URI?

Answer: Yes. The user confirmed that these identifiers are public data and that the previous restriction is too strict.

Decision / implication: A DID or AT-URI may appear in typed, allowlisted FCM data fields used for routing. This permission does not extend to post text, project data, image URLs, device tokens, session material, or identifier-bearing telemetry.

### Q2: Which layer owns destination inference?

Answer: Flutter.

Decision / implication: AppView owns notification classification and supplies canonical notification facts. Flutter owns the category-to-destination mapping, typed route construction, and navigation behavior. AppView does not send `routeKind` or other client-navigation policy. Provider data remains untrusted, the local account-subscription binding gates navigation, and destination content still loads through authenticated AppView APIs.

### Q3: Should the payload contain a literal deep-link URL?

Answer: No. Use a small versioned notification-fact payload and construct typed application routes locally.

Decision / implication: Version 1 uses `payloadVersion`, `type`, `accountSubscriptionId`, and the minimum category-specific canonical references. Flutter validates those facts and applies its own route policy; it never executes or passes through an arbitrary URL supplied by provider data.

### Q4: What are the version 1 fact fields and Flutter category mappings?

Answer:

- Common fields are `payloadVersion`, `type`, and `accountSubscriptionId`; `notificationId` is no longer needed in the provider open contract.
- Version 1 sends only the minimum category-specific facts:
  - `follow`: `actorDid`
  - `like`, `repost`: `subjectUri`
  - `mention`, `quote`: `sourceUri`
  - `reply`: `subjectUri` and `sourceUri`
  - `everythingElse`: no destination reference
- Flutter mapping is:
  - `follow` -> actor profile using `actorDid`
  - `like`, `repost` -> the interacted-with post using `subjectUri`
  - `mention` -> the post containing the mention using `sourceUri`
  - `quote` -> the actor's quoting post using `sourceUri`
  - `reply` -> the parent/subject thread using `subjectUri`, focused on the reply identified by `sourceUri`
  - `everythingElse` -> Notifications

Decision / implication: AppView projects canonical notification facts but does not select a client route. Changing a category's navigation may require a payload-contract revision if Flutter later needs a fact not present in this minimal version.

### Q5: How should stale, deleted, hidden, and offline destinations behave?

Answer: Navigate immediately from valid notification facts after Flutter infers the route, then let the destination's normal AppView read determine whether content is available.

Decision / implication:

- Explicit `404 post_not_found` and `404 profile_not_found` responses are permanent unavailable outcomes.
- Permanent unavailable destinations remain visible and show localized explanatory UI with Back and View notifications actions; they do not auto-redirect.
- Network failures, `5xx`, and `502 identity_unavailable` are transient and remain on the intended destination with Retry.
- `401` follows the existing global sign-out flow and does not leave a notification-specific unavailable or retry state behind.
- No outcome creates a persisted or automatic notification-open retry.

### Q6: How should old payloads and old clients be handled?

Answer: They do not need to be handled. The app is not live, and old notification data and client behavior do not need preservation.

Decision / implication: Make a coordinated clean cutover. Flutter requires a supported payload version and valid category facts, AppView stops sending `notificationId` in provider data, the notification-resolution endpoint and resolution-only client path are removed, and any stale development payload may fall back safely without being resolved. No compatibility rollout, data migration, or backfill is required.

### Q7: How strict is category-specific payload validation?

Answer: Require the fields needed by the known category and ignore all extras.

Decision / implication: A missing or malformed required field makes a known-category payload unroutable. Extra fields never affect routing and do not invalidate an otherwise valid payload. This permits harmless additive facts without letting them widen navigation authority.

### Q8: How do unknown types and malformed payloads differ?

Answer: Unknown types use a quiet fallback; malformed known types and unsupported payload versions use feedback.

Decision / implication: An unknown but syntactically valid `type` opens Notifications without an error message. A supported known type with missing/malformed required facts, or an unsupported/malformed `payloadVersion`, opens Notifications with brief Unable to open this notification feedback after binding validation.

### Q9: How should notification opens behave across readiness and sign-in boundaries?

Answer: Retain only the latest open through transient readiness and discard it when a new sign-in is required.

Decision / implication: Restoration or onboarding readiness for the already authenticated account may hold one in-memory open; a newer open replaces it. Requiring actual sign-in clears the pending open permanently, even if the member later signs into the same account.

### Q10: Are generic or unknown rows in the Notifications page interactive?

Answer: No.

Decision / implication: Generic and unknown feed rows are informational and non-tappable because the user is already on the only safe destination. Known rows with unavailable hydrated content retain their explicit unavailable behavior.

## 4. Candidate Approaches

### Option A: Versioned Minimal Notification Facts With Flutter Routing

Summary: AppView sends a versioned, category-specific minimum set of canonical DIDs/AT-URIs. Flutter validates the local account binding, infers the destination from notification type and facts, constructs a typed route immediately, and lets the destination perform its normal authenticated AppView read.

Pros:

- Removes a serial network request from the normal open path.
- Navigates immediately to a meaningful loading surface.
- Reuses the same AppView content authorization and moderation boundary as ordinary in-app navigation.
- Makes known notification opens resemble known notification-row navigation.
- Removes the notification-resolution API and client plumbing instead of retaining a parallel legacy path.
- Keeps client navigation policy in Flutter rather than coupling AppView to GoRouter concepts.

Cons:

- FCM/APNs can observe the public destination identifier and associate it with a delivery.
- Destination pages need explicit unavailable-versus-retry UX.
- A future navigation change may require a payload-version update if Flutter needs facts omitted by the minimal contract.

Risks:

- Loose category/reference parsing could turn provider data into an unintended navigation surface.
- Stale notification facts can identify content that changed after provider acceptance.

### Option B: Retain Mandatory Notification-ID Resolution

Summary: Keep the current minimal opaque payload and wait for AppView to authorize and select every destination before navigation.

Pros:

- Preserves the strictest provider-data privacy boundary.
- Selects a current server-side fallback before the app changes route.
- Requires no push contract change.

Cons:

- Retains the serial resolution round trip and slower perceived response.
- Duplicates an AppView check immediately before the destination's own AppView read.
- Keeps notification-specific navigation policy split between server resolution and destination screens.

Risks:

- A slow or unavailable resolution endpoint prevents immediate navigation even when the destination could render a useful loading or retry state.

### Option C: Encrypted Per-Installation Route Capsule

Summary: Encrypt destination metadata for a per-installation or per-account-subscription key so providers see only ciphertext and the app can decrypt locally.

Pros:

- Removes the resolution round trip without exposing destination identifiers to the provider.

Cons:

- Adds key generation, secure storage, rotation, rebinding, versioning, cryptographic failure handling, and server-side encryption concerns.
- Increases complexity beyond the current resolver design.

Risks:

- Key lifecycle mistakes could make notifications unroutable or weaken account isolation.

## 5. Recommended Direction

Recommended approach: Option A, versioned minimal public notification facts with strict typed parsing, Flutter-owned destination inference, local account-binding validation, immediate typed navigation, and normal AppView authorization at the destination.

Why: The user explicitly accepts provider exposure of public DIDs and AT-URIs. This approach removes the avoidable serial resolution request while preserving the meaningful security boundary: AppView still decides whether the signed-in member may receive the target content. AppView remains responsible for notification classification and canonical facts, while Flutter owns presentation/navigation policy. A versioned fact contract is safer than a literal arbitrary URL, and its minimal category-specific fields keep provider data bounded. Because the app is not live, the obsolete resolver contract can be removed cleanly instead of preserving parallel behavior.

## 6. Problem / Opportunity

Notification taps currently wait for a notification-specific AppView request before navigation and then issue another AppView request to load the destination. This adds latency and couples ordinary post/profile navigation to a separate resolver even though route identifiers are public and the destination API already enforces authenticated visibility. Direct navigation inferred from notification facts can make the app respond immediately while consolidating unavailable and retry behavior at the destination screens where those states also matter for ordinary deep links.

## 7. Goals

- G-001: Remove notification-ID resolution from the normal open path for newly generated notification-fact payloads.
- G-002: Navigate immediately after readiness and local account-binding validation without awaiting a network request.
- G-003: Preserve AppView as the authority for content availability, moderation, and authenticated reads.
- G-004: Handle stale, deleted, hidden, malformed, unknown, cross-account, pre-cutover, and offline opens predictably.
- G-005: Keep the provider-neutral notification architecture and one unified open path across foreground, background, and terminated app states.
- G-006: Remove obsolete notification-resolution payload, API, model, repository, policy, and routing behavior that exists only for the pre-launch implementation.

## 8. Non-Goals

- NG-001: Authorize content access from push payload data.
- NG-002: Read post or profile content directly from a PDS.
- NG-003: Add external universal-link/app-link registration or a public `craftsky://` scheme.
- NG-004: Put post text, project fields, mention text, image URLs, handles, tokens, credentials, or private account data into the push payload.
- NG-005: Add encrypted push payloads or per-installation cryptographic key management.
- NG-006: Persist or automatically retry failed notification opens.
- NG-007: Change notification eligibility, preference, coalescing, newness, seen, TTL, sound, badge, or delivery-retry behavior.
- NG-008: Preserve compatibility with pre-cutover provider payloads, development data, or client behavior.
- NG-009: Change ordinary known notification-row navigation except where shared unavailable/retry UI is intentionally improved; generic/unknown rows deliberately become non-interactive.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky member | Signed-in member opening a foreground banner or OS notification. | Immediate, correct navigation with understandable unavailable and retry states. |
| Multi-account installation | One installation that may hold routing bindings for different account DIDs over time. | A stale or different account's push must never navigate under the current account. |
| AppView | Trusted server that indexes records, creates notification intent, sends pushes, and serves content. | A stable notification-fact contract while remaining authoritative for content reads. |
| Flutter client | Provider-neutral consumer of notification events and typed routes. | Strict parsing, account routing validation, clean cutover, and safe UI fallbacks. |
| Push provider | FCM/APNs delivery infrastructure. | A bounded string data map and ordinary visible notification fields. |
| Operator | Maintains delivery and client health. | Privacy-safe outcome telemetry without identifiers or raw payloads. |

## 10. Current Behavior

AppView sends generic visible copy plus `notificationId`, `type`, and an opaque installation-specific `accountSubscriptionId`. Flutter validates the current DID's stored binding, calls owner-scoped `GET /v1/notifications/{notificationId}`, maps the returned resolution to post/profile/Notifications, and only then navigates. AppView resolution applies current visibility checks and may choose an alternate target. The destination screen subsequently performs its own authenticated AppView read. Network or resolution failures route to Notifications rather than immediately showing the intended destination's loading/retry surface.

## 11. Desired Behavior

AppView sends version 1 minimal notification facts with each push and no longer sends a notification ID for open resolution. Flutter parses the facts into provider-neutral typed domain values, waits for the existing authenticated/onboarded readiness boundary, validates the payload's account-subscription ID against secure DID-keyed local storage, infers the category destination, and navigates without calling notification resolution. The destination screen performs its normal authenticated AppView read. Explicit post/profile not-found outcomes remain on the destination with unavailable UI and Back/View notifications actions; transient failures remain retryable, while `401` follows global sign-out. Unknown types quietly open Notifications, whereas malformed known types or unsupported versions open Notifications with brief feedback. Only the latest open survives transient readiness, and none survives an actual sign-in boundary. Resolution-only server and client code is removed as part of the clean pre-launch cutover.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A member opening a newly generated notification shall reach the intended in-app destination without waiting for a notification-specific AppView resolution request. | Reduces tap-to-navigation latency and simplifies the normal open path. | Initial request / User answer | AC-001, AC-007 |
| BR-002 | Business | Must | Direct notification routing shall preserve account isolation and shall not bypass AppView content availability or moderation enforcement. | Latency must not trade away the actual authorization boundary. | Discovery / Architectural rules | AC-006, AC-008 |
| FR-001 | Functional | Must | AppView shall replace notification-ID provider routing with a versioned minimal notification-fact contract in FCM data payloads containing common `payloadVersion`, `type`, and `accountSubscriptionId` fields. | Provides explicit schema selection and account context without an obsolete resolution identifier. | Recommended direction / Grilling decision | AC-002, AC-014 |
| FR-002 | Functional | Must | Payload version 1 shall contain only the minimum category-specific facts defined in Q4: `actorDid` for follow; `subjectUri` for like/repost; `sourceUri` for mention/quote; `subjectUri` plus `sourceUri` for reply; and no destination reference for everything-else. | Keeps payloads bounded while supplying the facts Flutter needs. | Grilling decision | AC-002, AC-003, AC-018 |
| FR-003 | Functional | Must | Flutter shall own category-to-destination inference according to Q4; AppView shall supply canonical facts but shall not send `routeKind`, final client paths, or other navigation policy. | Keeps UI navigation logic in the app and avoids coupling AppView to Flutter routes. | Grilling decision | AC-003, AC-016 |
| FR-004 | Functional | Must | Push-visible title/body copy shall remain generic, and only the data portion may add public target DIDs and AT-URIs; post text, project data, mention text, image URLs, handles, tokens, credentials, and session data shall remain absent. | Narrows the privacy change to public routing identifiers. | User answer / Existing privacy contract | AC-004 |
| FR-005 | Functional | Must | Flutter shall parse provider data through an allowlist into provider-neutral typed values, validating `payloadVersion`, bounded `type`, account-subscription ID, DID, and Craftsky post AT-URIs before use. | Treats all provider data as untrusted input. | Existing pattern / Security analysis | AC-005 |
| FR-006 | Functional | Must | Before any notification navigation or safe fallback, Flutter shall require the payload `accountSubscriptionId` to equal the securely stored binding for the current authenticated DID; absent or mismatched bindings shall not navigate. | Preserves stale/cross-account isolation. | Existing contract | AC-006 |
| FR-007 | Functional | Must | For valid version 1 facts with a matching binding and known category, Flutter shall infer, construct, and open the corresponding typed GoRouter route without calling `GET /v1/notifications/{notificationId}` first. | Removes the serial resolution round trip while keeping navigation policy client-side. | Initial request / Grilling decision | AC-001, AC-007 |
| FR-008 | Functional | Must | Post and profile destinations inferred from notification facts shall load through their existing authenticated AppView APIs and shall apply the same server visibility/moderation behavior as ordinary in-app navigation. | Keeps payload facts as navigation input rather than authorization. | Architectural rules / Discovery | AC-008 |
| FR-009 | Functional | Must | Explicit `404 post_not_found` and `404 profile_not_found` destination responses shall remain on the intended route and show localized permanent-unavailable UI with Back and View notifications actions, without displaying stale payload-derived content or auto-redirecting. | Handles deleted, hidden, and taken-down destinations safely and understandably. | Codebase finding / Grilling decision | AC-009 |
| FR-010 | Functional | Must | Network failures, `5xx`, and `502 identity_unavailable` shall retain the intended destination and offer Retry; `401` shall follow existing global sign-out without leaving notification-specific error UI; no failure shall persist or automatically schedule a notification-open retry. | Distinguishes recoverable connectivity, authentication loss, and permanent unavailability. | Codebase finding / Grilling decision | AC-010 |
| FR-011 | Functional | Must | Flutter shall not preserve a notification-ID resolution path for pre-cutover payloads without `payloadVersion`; after binding validation, such payloads shall open Notifications with brief unable-to-open feedback. | The app is not live and legacy behavior would retain unnecessary complexity. | User clarification / Grilling decision | AC-011 |
| FR-012 | Functional | Must | An unsupported/malformed `payloadVersion`, malformed `type`, or known category with missing/malformed required facts shall open Notifications with brief unable-to-open feedback after binding validation; an unknown but syntactically valid `type` shall open Notifications quietly without inferred routing. | Distinguishes corrupt/version-drift input from valid future generic activity. | Grilling decision | AC-012, AC-021 |
| FR-013 | Functional | Must | Foreground banner taps, background opens, and terminated initial opens shall all use the same provider-neutral fact parsing, readiness, binding, inference, navigation, and fallback policy. | Avoids app-state-specific routing drift. | Existing architecture / Grilling decision | AC-013 |
| FR-014 | Functional | Must | Version 1 provider payloads and provider-neutral open events shall not require or carry `notificationId`; durable notification IDs remain internal to the in-app notification feed where otherwise needed. | Removes a field whose only open-path purpose was server resolution. | User clarification / Simplification goal | AC-014 |
| FR-015 | Functional | Must | AppView shall remove `GET /v1/notifications/{notificationId}`, and Flutter shall remove resolution-only API, repository, model, policy, and coordinator behavior; generic or unknown notification feed rows shall be informational and non-interactive. | Completes the clean cutover and avoids meaningless navigation while already on Notifications. | User clarification / Grilling decision | AC-015 |
| FR-016 | Functional | Must | Flutter shall open reply notifications at the parent/subject thread identified by `subjectUri` and focus the reply identified by `sourceUri`. | Preserves conversational context and aligns push taps with known notification-row behavior. | Grilling decision | AC-003, AC-016 |
| FR-017 | Functional | Must | Payload production shall copy only the category's required canonical actor/source/subject facts from the durable notification event, never deriving a final route from localized copy or accepting a client-supplied destination. | Keeps server facts deterministic while leaving route policy in Flutter. | Codebase finding / Grilling decision | AC-003, AC-016 |
| FR-018 | Functional | Must | During transient restoration/onboarding readiness, Flutter shall retain only the latest in-memory notification open; requiring actual sign-in shall discard it permanently, even if the same account later signs in. | Preserves current intent without carrying navigation across an authentication boundary or adding a queue. | Grilling decision | AC-022 |
| FR-019 | Functional | Must | For a known category, Flutter shall require and use only that category's required facts, ignore all extra fields, and never use extras to infer or alter a destination. | Allows harmless payload additions without widening navigation authority. | Grilling decision | AC-005, AC-021 |
| NFR-001 | Non-functional | Must | For valid version 1 opens, no network operation shall be awaited between successful readiness/binding validation and route navigation. | Makes the latency improvement measurable and regression-testable. | Initial request | AC-007 |
| NFR-002 | Non-functional | Must | FCM tokens, account-subscription IDs, notification IDs, route DIDs, AT-URIs, focus URIs, full payloads, and credentials shall not appear in logs, Sentry contexts, analytics, metrics labels, or user-facing diagnostics. | Public identifiers are permitted in transit, not in telemetry. | Existing security model / User answer | AC-017 |
| NFR-003 | Non-functional | Must | The notification-fact payload shall remain a bounded flat string map and shall not add unbounded content or a second serialization format. | Preserves provider limits and the existing FCM integration shape. | Provider constraint / Architectural rules | AC-018 |
| NFR-004 | Non-functional | Should | Fact parsing, destination inference, and navigation shall remain provider-neutral outside the Firebase adapter and shall be fully testable without Firebase, FCM, OS permission, or a physical device. | Preserves deterministic testability and adapter isolation. | Existing architecture | AC-019 |
| NFR-005 | Non-functional | Should | Unavailable and retry states introduced or refined by this change should use localized, accessible copy and controls. | Keeps edge-case handling understandable and consistent with the app. | Existing UI conventions | AC-009, AC-010 |
| RULE-001 | Business rule | Must | Public DIDs and AT-URIs may be included only in allowlisted push notification-fact fields; this exception does not make arbitrary payload or content fields acceptable. | Records the confirmed relaxation without discarding the wider privacy boundary. | User answer | AC-004, AC-017 |
| RULE-002 | Business rule | Must | Notification facts inform Flutter's requested destination but never authorize content access; authenticated AppView destination reads remain authoritative. | Prevents a payload from becoming a capability. | Discovery decision | AC-008 |
| RULE-003 | Business rule | Must | The local DID-keyed account-subscription binding is mandatory before direct navigation, including when the route target would otherwise be public. | Prevents stale pushes opening under the wrong signed-in account. | Existing privacy model | AC-006 |
| RULE-004 | Business rule | Must | An unknown but syntactically valid notification type shall be treated as generic activity and open Notifications quietly; Flutter shall not infer a destination from unknown extra fields. | Preserves forward compatibility without presenting valid future activity as an error. | Grilling decision | AC-012 |
| RULE-005 | Business rule | Must | Delivered notification facts are snapshots and may be stale; current destination availability is determined only when the destination loads through AppView. | A provider-accepted push cannot be recalled or rewritten. | Provider constraint / Discovery | AC-008, AC-009 |
| RULE-006 | Business rule | Must | The change shall not alter notification eligibility, preference snapshots, coalescing, newness, seen acknowledgement, TTL, sound, badge, or delivery retry semantics. | Keeps this slice focused on open routing. | Scope boundary | AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-007 | Given a valid version 1 push for the ready current account, when the member opens it, then the app begins typed destination navigation without first waiting for notification-ID resolution. |
| AC-002 | FR-001, FR-002 | Given each current category, when AppView builds a payload, then it includes `payloadVersion=1`, `type`, `accountSubscriptionId`, exactly the category's minimum required `actorDid`/`subjectUri`/`sourceUri` facts, and no `notificationId`, `routeKind`, final path, or unnecessary canonical references. |
| AC-003 | FR-002, FR-003, FR-016, FR-017 | Given valid facts for each current category, when Flutter infers the destination, then follow opens the actor DID profile; like/repost open the subject post; mention opens the source post; quote opens the quoting source post; reply opens the subject thread focused on the source reply; and everything-else opens Notifications. |
| AC-004 | FR-004, RULE-001 | Given a sent push, when its visible and data fields are inspected, then only the approved generic copy, category, account-subscription binding, version metadata, and minimum public notification facts are present, with no notification ID, prohibited content, token, credential, handle, or session data. |
| AC-005 | FR-005, FR-019 | Given valid and invalid payload versions, types, account-subscription IDs, DIDs, AT-URIs, category-required facts, extra fields, and arbitrary URLs, when Flutter parses provider data, then only validated provider-neutral facts enter the domain event, required fields are enforced, and extras are ignored for routing. |
| AC-006 | BR-002, FR-006, RULE-003 | Given a missing, stale, or different account-subscription binding, when a notification is opened, then Flutter neither navigates nor calls notification resolution and shows only generic unavailable feedback after readiness. |
| AC-007 | BR-001, FR-007, NFR-001 | Given valid category facts and a matching binding, when readiness completes, then Flutter-inferred typed navigation is emitted before any destination network result and the notification-resolution repository is not called. |
| AC-008 | BR-002, FR-008, RULE-002, RULE-005 | Given a valid direct post/profile route, when the destination loads, then it uses the authenticated AppView API and hidden, taken-down, or unauthorized content is not returned merely because its identifier came from a push. |
| AC-009 | FR-009, NFR-005, RULE-005 | Given a destination returning explicit `404 post_not_found` or `404 profile_not_found`, when the page renders, then it remains on that route, shows localized permanent-unavailable explanation with accessible Back and View notifications actions, does not auto-redirect, and shows no stale payload-derived content. |
| AC-010 | FR-010, NFR-005 | Given a destination network failure, `5xx`, or `502 identity_unavailable`, when the error is shown, then the intended route remains visible with localized accessible Retry and no persisted/automatic open retry; given `401`, normal global sign-out occurs without notification-specific error UI. |
| AC-011 | FR-011 | Given a stale pre-cutover payload without `payloadVersion`, when the current app opens it with a matching binding, then it opens Notifications with brief unable-to-open feedback without calling notification resolution or reconstructing a target from old fields. |
| AC-012 | FR-012, RULE-004 | Given an unsupported/malformed version, malformed type, or malformed known-category facts, when opened with a valid binding, then Flutter opens Notifications with brief unable-to-open feedback; given an unknown but syntactically valid type, it opens Notifications quietly and ignores unknown extra fields. |
| AC-013 | FR-013 | Given equivalent valid fact events from a foreground banner, background notification tap, and terminated initial message, when processed, then all three traverse the same validation/inference policy and produce the same navigation outcome exactly once per callback. |
| AC-014 | FR-001, FR-014 | Given a valid version 1 push and provider-neutral open event, when their fields are inspected, then neither requires or carries `notificationId`, while durable in-app notification rows may retain their own IDs. |
| AC-015 | FR-015 | Given the clean cutover is complete, when AppView routes and Flutter notification-open/feed-row code are inspected, then the notification-resolution endpoint and resolution-only client types/calls are absent, while generic or unknown feed rows have no tap action. |
| AC-016 | FR-003, FR-016, FR-017 | Given each durable notification event, when AppView builds provider data, then it copies only the category's canonical facts and no Flutter route policy; reply data preserves subject thread plus source reply so Flutter can focus it. |
| AC-017 | NFR-002, RULE-001 | Given sentinel route identifiers, common IDs, tokens, and payload data across parse, success, fallback, and failure paths, when logs, Sentry, analytics, metrics, and user-facing diagnostics are inspected, then no sentinel identifier or raw payload appears. |
| AC-018 | FR-002, NFR-003 | Given the largest supported version 1 fact payload, when the provider message is built, then data remains a bounded flat string map containing only common and minimum category-specific fields, with no embedded object/list serialization or content body. |
| AC-019 | NFR-004 | Given automated direct-routing tests, when they run, then provider-neutral fakes cover fact parsing, inference, readiness, binding, and navigation without initializing Firebase, contacting FCM, requesting OS permission, or requiring a device. |
| AC-020 | RULE-006 | Given direct push routing is introduced, when notification preference, eligibility, delivery, TTL, sound, badge, list, new-count, and seen regression suites run, then their existing behavior remains unchanged except for the replacement push data contract and removal of notification-resolution behavior. |
| AC-021 | FR-012, FR-019 | Given a known category with all required valid facts plus unexpected extras, when opened, then Flutter ignores the extras and routes normally; when a required fact is absent/malformed, it opens Notifications with brief unable-to-open feedback. |
| AC-022 | FR-018 | Given multiple opens during transient readiness, when readiness becomes ready, then only the latest open is processed; when readiness requires sign-in, all pending opens are discarded and do not reappear after sign-in. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Push targets an account other than the current locally authenticated DID. | Binding mismatch prevents navigation; generic unavailable feedback appears after readiness. | FR-006, RULE-003 |
| EC-002 | Push arrives while the existing account is restoring or onboarding is incomplete. | Retain only the latest pending open through transient readiness, then infer and route it once ready. | FR-013, FR-018 |
| EC-003 | Push would require a new sign-in. | Discard it before destination navigation so it cannot carry into another account. | FR-006, FR-018 |
| EC-004 | `payloadVersion` is missing. | After binding validation, open Notifications with brief unable-to-open feedback and do not use pre-cutover fields. | FR-011, FR-012 |
| EC-005 | `payloadVersion` is newer than the client. | Open Notifications with brief unable-to-open feedback; do not infer a target from category or unknown fields. | FR-012 |
| EC-006 | A known category is missing its required fact. | Open Notifications with brief unable-to-open feedback. | FR-002, FR-005, FR-012 |
| EC-007 | `sourceUri` or `subjectUri` is a valid AT-URI for a non-post collection. | Treat the required fact as malformed and open Notifications with brief feedback. | FR-005, FR-012 |
| EC-008 | A follow payload contains an invalid `actorDid`. | Treat the required fact as malformed and open Notifications with brief feedback. | FR-005, FR-012 |
| EC-009 | Unknown future category has a syntactically valid type and extra facts. | Render generic activity and open Notifications quietly; ignore all unknown facts. | FR-012, RULE-004 |
| EC-010 | Source/destination is deleted after FCM accepts the push. | Navigate from the snapshot hint, then render permanent unavailable state when AppView refuses the destination. | FR-009, RULE-005 |
| EC-011 | Destination is temporarily unreachable. | Keep the destination route and show Retry rather than redirecting to Notifications as if content were permanently unavailable. | FR-010 |
| EC-012 | Actor account/repository is permanently deleted after delivery. | Profile/post AppView read returns unavailable/not-found and the destination offers a path back to Notifications without exposing stale content. | FR-008, FR-009 |
| EC-013 | Reply source remains but its subject/parent target becomes unavailable. | The thread route remains visible and presents permanent unavailable UI; it does not reconstruct unprovided fallback identifiers. | FR-009, FR-016 |
| EC-014 | Extra destination-like keys accompany an otherwise valid known-category payload. | Parser ignores them; only the category's required validated facts influence Flutter routing. | FR-019 |
| EC-015 | Duplicate provider callbacks carry the same notification facts. | Each callback follows existing at-least-once behavior; this change adds no receipt deduplication store. | RULE-006 |
| EC-016 | Current app opens a stale development payload from before the cutover. | Matching binding is validated, then the app opens Notifications without notification-ID resolution; preserving the old destination is not required. | FR-011 |
| EC-017 | A pre-cutover development client receives the replacement payload. | Compatibility is not provided because the app is not live; developers must update/reset their local build and data. | FR-015 |
| EC-018 | Two or more opens arrive during transient readiness. | Retain only the most recent open and process it once when ready. | FR-018 |
| EC-019 | An open is pending when readiness changes to requires-sign-in. | Discard it permanently; later sign-in, including to the same DID, does not revive it. | FR-018 |

## 15. Data / Persistence Impact

- New fields:
  - No database fields.
  - Replacement FCM fact keys: common `payloadVersion`, `type`, and `accountSubscriptionId`, plus category-specific `actorDid`, `sourceUri`, and `subjectUri` as defined in Q4.
- Changed fields:
  - `type` and `accountSubscriptionId` meanings remain unchanged.
  - `notificationId` is removed from provider payloads and provider-neutral open events; durable feed rows may retain their internal IDs.
  - Existing durable actor/source/subject references become inputs to minimal push fact construction.
- Migration required: No.
- Local persistence:
  - No new storage. Existing secure DID-to-`accountSubscriptionId` bindings remain the routing gate.
  - No pending deep link, receipt, notification payload, or retry is persisted.
- Backwards compatibility:
  - None required. The app is not live, no old client behavior is supported, and no accepted old push needs to remain routable.
  - No backfill, transformation, compatibility deployment sequence, or legacy data preservation is required.
  - Existing local development payloads/data may be discarded or reset.

## 16. UI / API / CLI Impact

- UI:
  - Notification taps navigate immediately to Flutter-inferred typed post/profile/Notifications routes after local validation.
  - Post and profile destinations distinguish permanent unavailable/not-found outcomes from transient retryable failures.
  - Permanent unavailable UI remains on the destination and includes Back and View notifications actions.
  - Generic/unknown feed rows are informational and non-interactive.
- API:
  - No new JSON/HTTP route is required.
  - Existing destination routes remain authoritative.
  - Remove `GET /v1/notifications/{notificationId}` and its owner-scoped resolution store surface.
- Push provider contract:
  - Existing combined notification-and-data messages replace `notificationId` routing with versioned minimal notification facts.
  - Visible notification copy remains generic and unchanged.
- CLI: None identified.
- Background jobs:
  - The push dispatcher reads only the category's durable canonical facts required by payload version 1.
  - Claiming, fencing, retry classification, TTL, cancellation, and provider sending semantics remain unchanged.

## 17. Security / Privacy / Permissions

- Authentication:
  - Direct routes become actionable only after the existing authenticated/onboarded readiness boundary.
  - Destination reads continue using the Craftsky session and device-ID middleware.
- Authorization:
  - The payload does not authorize content.
  - The local account-subscription match prevents cross-account opens.
  - AppView post/profile reads enforce current content visibility and moderation.
- Sensitive data:
  - Public DIDs and AT-URIs are explicitly permitted in allowlisted FCM notification-fact fields.
  - Their presence in telemetry, diagnostics, or visible copy remains prohibited.
  - Tokens, credentials, sessions, private database data, post text, project content, images, mention text, and handles remain excluded from payload data.
- Abuse cases:
  - Arbitrary URL injection is prevented by category-specific required fields and typed DID/AT-URI parsing.
  - A forged or stale payload cannot select a different local account because binding validation uses secure DID-keyed local state.
  - A forged public identifier cannot reveal moderated content because the destination still reads through AppView.
  - Extra or future fields do not widen routing authority.

## 18. Observability

- Events:
  - Record privacy-safe classes for payload parse outcome, payload-version support, binding match/mismatch, known/unknown category, inferred destination class, safe Notifications fallback, permanent unavailable outcome, and transient destination failure.
- Logs:
  - Log only operation/outcome classes and safe route kind/version categories.
  - Never log full payloads, notification IDs, account-subscription IDs, DIDs, AT-URIs, focus URIs, tokens, or destination URLs.
- Metrics:
  - Optional bounded counters may distinguish direct navigation, safe Notifications fallback, binding mismatch, permanent unavailable, and transient failure.
  - No identifier may be used as a label.
- Alerts:
  - None required for the initial change. Existing push delivery health remains unchanged.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Provider data is accidentally treated as authorization. | Hidden, taken-down, or unavailable content could be exposed. | Keep destination reads on authenticated AppView APIs and test moderation/not-found outcomes. |
| RISK-002 | DID/AT-URI exposure through providers is broader than intended. | Provider can associate a delivery with a public account/content identifier. | Record explicit user acceptance, limit identifiers to minimum notification-fact fields, and keep content/telemetry exclusions. |
| RISK-003 | Arbitrary or malformed payload fields become navigation input. | Unexpected routes, crashes, or unsafe fallback behavior. | Use a versioned allowlist, category-specific required facts, typed parsing, extras-ignore policy, and Notifications fallback. |
| RISK-004 | A stale push routes to deleted or moderated content. | Confusing or broken destination experience. | Treat hints as snapshots and implement dedicated unavailable UI backed by current AppView reads. |
| RISK-005 | Network failures are mistaken for permanent deletion. | Users cannot retry content that is still available. | Classify permanent versus transient AppView errors and keep Retry on the intended route. |
| RISK-006 | Account binding validation is skipped on the faster path. | A stale notification may open under the wrong account. | Centralize direct navigation and safe fallback behind the same mandatory binding gate with negative tests. |
| RISK-007 | A developer runs mismatched pre-cutover AppView and Flutter builds. | Local notification opens may fall back or fail during development. | Treat the change as a coordinated cutover and require developers to update/reset both local components; production compatibility is not required. |
| RISK-008 | Obsolete resolver code remains partially wired after the cutover. | Complexity persists and generic/unknown paths may behave inconsistently. | Remove the endpoint and resolution-only client/server surfaces together, with structural regression tests. |
| RISK-009 | Flutter push inference and known notification-row navigation drift. | The same activity opens different post/profile contexts depending on entry point. | Reuse shared destination types/mapping where practical and add cross-entry regression tests. |
| RISK-010 | Route identifiers leak through errors or observability. | Identifier exposure outside the approved provider transit boundary. | Preserve sentinel redaction tests across parse, navigation, and destination failure paths. |
| RISK-011 | Reply subject/source semantics do not match thread routing. | Reply notifications open the wrong context or cannot focus the reply. | Specify subject thread plus source focus and cover root/nested reply cases in test design. |
| RISK-012 | Direct navigation improves route transition but destination loading still dominates perceived latency. | Expected performance gain may be smaller than anticipated. | Define success as removing the serial resolver and instrument only bounded outcome/timing classes if measurement is later required. |
| RISK-013 | Minimal facts are insufficient for a future navigation choice. | Flutter cannot adopt the new behavior without a payload change. | Increment `payloadVersion` when a category needs additional canonical facts; unknown versions fall back safely. |
| RISK-014 | Multiple startup opens create a navigation queue or cross a sign-in boundary. | Users see stale sequential navigation or an open under the wrong session. | Retain latest only during transient readiness and discard all pending state on requires-sign-in. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | FCM/APNs deliver the replacement notification-fact fields to foreground, background-open, and terminated-open callbacks in the same situations as the current data fields. | Native delivery handling or payload shape would need adjustment and physical-device verification. |
| ASM-002 | Current post and profile AppView routes apply the required visibility and moderation policy independently of notification resolution. | Direct routing would need a server authorization adjustment before it is safe. |
| ASM-003 | The app is not live, no production client depends on the current payload/resolution contract, and no old notification-open data must be preserved. | A compatibility design and staged rollout would be required before implementation. |
| ASM-004 | The durable notification event contains the actor/source/subject facts required by the minimal version 1 contract. | Some known categories would fall back with feedback or need additional durable data. |
| ASM-005 | Public DID/AT-URI exposure in provider data is acceptable under Craftsky's product privacy policy as confirmed by the user. | The design would need to retain resolution or adopt an encrypted route capsule. |
| ASM-006 | A dedicated unavailable destination state with Back and View notifications actions is an acceptable replacement for server-selected alternate fallback routing after delivery. | The payload would need additional fallback facts or the resolver would need to remain on selected categories. |
| ASM-007 | The current six-hour provider TTL remains sufficient to bound stale push behavior during development and after launch. | Staleness handling would require a different delivery policy, which is outside this change. |

## 21. Open Questions

- [ ] Non-blocking: Should privacy-safe latency timing be added in this slice, or is structural proof that no network call precedes navigation sufficient for the first release?

## 22. Review Status

Status: Approved

Risk level: High

Review recommended: Required

Reviewer: User

Date: 2026-07-17

Notes: This changes a previously explicit privacy and authorization-adjacent push contract, moves destination inference into Flutter, removes the notification-resolution surface, and requires destination error-state work. Grilling confirmed the minimal versioned fact contract, exact category mappings, required-fields-only validation, account binding, failure classification, latest-only readiness behavior, quiet unknown-type fallback, and non-interactive generic rows. The user approved public DID/AT-URI exposure, confirmed that the app is not live and no compatibility is required, and explicitly advanced the approved requirements into acceptance-test design and document review. Separate explicit approval remains required before implementation begins.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`-`BR-002`
  - `FR-001`-`FR-019`
  - `NFR-001`-`NFR-003`
  - `RULE-001`-`RULE-006`
- Suggested test levels:
  - Unit: AppView minimal canonical-fact projection; Flutter payload allowlist, category-specific required-fact validation, extras-ignore policy, destination inference, unknown-type fallback, error classification, and no-network-before-navigation policy.
  - Integration: dispatcher/payload facts from durable notification references; matching/mismatched account binding; safe fallback for pre-cutover payloads; authenticated destination moderation/not-found behavior; reply subject/source focus routing.
  - Widget/router: foreground/background/initial open equivalence; latest-only transient pending behavior; sign-in discard; immediate post/profile navigation; permanent unavailable Back/View notifications UI; transient Retry; quiet unknown-type fallback; generic-row non-interactivity; accessibility/localization.
  - Cutover/regression: absence of `notificationId`/route policy in provider opens; removal of the resolution endpoint/client path; safe stale development-payload fallback; notification preferences, list, count, seen, badge, sound, TTL, delivery retry, and sign-out isolation.
  - Static/privacy: Firebase import boundary; minimal payload field matrix; bounded flat map; sentinel scans across logs, errors, Sentry, analytics, and metrics.
  - Manual: physical Android/iOS background and terminated delivery with notification facts; cold-start timing observation; deleted/hidden target behavior; multi-account/stale-binding open.
- Blocking open questions: None.
- Approval gate: Requirements approval is recorded. High-risk workflow document review and separate explicit user approval remain required before implementation.
