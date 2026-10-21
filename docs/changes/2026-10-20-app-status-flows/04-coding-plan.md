# Coding Plan: App Status Flows

## 1. Inputs

- Requirements: `01-requirements.md`, Reviewed; 22 Must requirements and AC-001–AC-023.
- Test design: `02-acceptance-tests.md`, reviewed with non-blocking notes; AT-001–AT-011, UT-001–UT-009, IT-001–IT-006, REG-001–REG-004 and MAN-001–MAN-003.
- Document review: `03-document-review.md`, Approved with notes. DR-001/002 are resolved by user-confirmed D-011/D-012. Carry DR-003 revision-lifecycle and DR-004 browser-harness notes into this design.
- Scope: create this design artifact only. No source/test/dependency changes, code generation, runtime checks, commits or live actions in this stage.

## 2. Implementation Strategy

Extend the existing service-status feature in place. Keep its anonymous independent repository, lifecycle controller and retained application host. Replace the exclusive schema-1 mode with a schema-2 document containing optional independent maintenance, minimum-version and message sections. Keep one atomic accepted snapshot; track maintenance freshness and update trust separately.

Continue using a Riverpod `Notifier` for deterministic policy/timer/persistence state. Add an independent runtime-version provider and small platform action adapters with injectable boundaries and standard trusted-host image loading. Use the existing full-page cover composition for the update experience rather than replacing the GoRouter or navigating away from retained editors. A visible status page has route semantics and consumes back; it does not need a new URL or destructive route-stack redirect.

Use installed `package_info_plus`, `url_launcher`, `web`, Dio and existing design-system components; no update-specific package or new app framework is required. D-013 adds an explicitly pinned Python publication-validator dependency. The installed `pub_semver` 2.2.1 source was inspected: its `compareTo` includes build metadata and its grammar accepts leading-zero components. Do not use that comparator directly as the status contract. Define a small strict, metadata-free status version value object and parity fixtures; keep the unrelated app-wide version provider unchanged.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| JSON and eligibility | `dart_mappable` models plus explicit validation | Schema 2, strict version/range/CTA validation, pure platform eligibility | BR-001, BR-002, FR-002, FR-007, FR-010, FR-012, RULE-002 | UT-001/002/007, AT-001/002/006, IT-003 |
| Controller/lifecycle | `ServiceStatusController`, injected clock/completers | Independent update trust, retained maintenance age, derived presentation, captured-revision effects | FR-001, FR-006, FR-008, FR-009, RULE-001, RULE-003 | UT-003–UT-006, UT-008, AT-001/007/010/011 |
| Blocking UI | Stable `ServiceStatusHost` stack and root back dispatcher | Full-page update cover, both blocking flows in ready/loading/error hosts, richer message | FR-003–FR-005, FR-011, NFR-001, NFR-002 | AT-002–AT-006, AT-009, REG-001–REG-003 |
| External actions and images | Launcher functions, independent native/web status fetch | Injected store/reload/HTTPS launch, named route mapping, trusted-host Image.network rendering | FR-004, FR-005, FR-010, FR-011, NFR-003 | AT-004/005/008/009, IT-001/004/005/006, MAN-001/002 |
| Operator workflow | Python validator/publisher, local R2 helper, Worker | New fields, preserve supplied message revision, explicit full clear and safe summaries | BR-001, FR-009, FR-012, FR-013 | AT-007, UT-007, IT-003, REG-004 |
| Verification/documentation | Existing Flutter, Python, Node and Playwright tooling | Parity corpus, race tests, device storage test, external reload harness, runbook | NFR-001–NFR-003, FR-005, FR-008, FR-013 | IT-002/005/006, REG-001–REG-004, MAN-001–MAN-003 |

## 4. Files And Modules

Paths below are relative to the repository root. New names are implementation targets, not files created by this stage. Existing test files may own multiple IDs; keep fixtures shared within the feature.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/service_status/models/service_status_document.dart` and `.mapper.dart` | Change / regenerate | Root schema-2 document and mapped maintenance/message/CTA types, explicit strict validator, safe summaries | BR-001, FR-007, FR-010, FR-012 | UT-007, AT-001/006, IT-003 |
| `app/lib/service_status/models/status_app_version.dart` | Create | Strict semantic-version value, metadata-free precedence and range helpers | FR-002, FR-007, RULE-002 | UT-001/002 |
| `app/lib/service_status/providers/status_runtime_provider.dart` | Create | Independent package-info read and supported platform selection, with injectable values | FR-001, FR-002, RULE-003 | UT-002/008, AT-002/010 |
| `app/lib/service_status/providers/service_status_controller.dart` | Change | Separate trust state, single presentation selection, revision-safe dismissal/activation state | FR-001, FR-006, FR-008, FR-009, RULE-001, RULE-003 | UT-003–UT-006, UT-008, AT-001/007/011 |
| `app/lib/service_status/data/announcement_dismissal_store.dart` | Change | Keep current single-revision API; new schema-2 key and explicit captured-revision writes | FR-008, FR-009 | UT-006, IT-002, AT-007 |
| `app/lib/service_status/data/status_actions.dart`, `status_reload_stub.dart`, `status_reload_web.dart` | Create | Typed action callbacks, official listings, existing external launcher reuse, conditional web reload | FR-004, FR-005, FR-010 | AT-004/005/008, IT-004/005 |
| `app/lib/service_status/widgets/status_message_image.dart` | Change | Standard trusted-host network image with bounded decorative display and failure fallback | FR-011, NFR-003, D-014 | AT-009, IT-006 |
| `app/lib/service_status/widgets/update_required_screen.dart` | Create | Full-page themed update action, pending/failure feedback, semantics and bounded scrolling | FR-003–FR-005, NFR-002 | AT-002/004/005/009, MAN-001/003 |
| `app/lib/service_status/widgets/announcement_modal.dart` | Change | Title/description/image/CTA plus retained explicit dismissal | FR-007, FR-008, FR-010, FR-011, NFR-002 | AT-006/008/009, REG-002 |
| `app/lib/service_status/widgets/service_status_host.dart`, `maintenance_screen.dart` | Change | One presentation branch; reusable full-page blocking/focus behavior; maintenance consumes nested section | FR-003, FR-006, NFR-001, NFR-002 | AT-001/003/009/011, REG-001–REG-003 |
| `app/lib/service_status/service_status_router_config.dart`, `app/lib/app.dart` | Change | Blocking predicate includes update; splash resolves for either cover; ready/loading/error parity; normal router for named CTA | FR-003, FR-010, NFR-001 | AT-002/003/008, IT-004, REG-003 |
| `app/lib/service_status/widgets/service_status_lifecycle_host.dart` | Retain; adjust only if required | Existing one-time start and foreground events remain sole lifecycle owner | FR-001 | UT-004, REG-001 |
| `app/lib/l10n/app_en.arb` and generated localization output | Change / regenerate | Update/reload/action-failure fixed copy; retain operator prose as plain text | FR-004, FR-005, FR-010, NFR-002 | AT-004/005/008/009 |
| `service-status/app.schema.json`, `service-status/requirements.txt`, project-local validator environment setup | Create | Formal schema and pinned offline publication validator | FR-012, FR-014 | UT-007/009, IT-003 |
| `scripts/service_status.py`, `scripts/service-status-dev`, `justfile` comments | Change | Schema 2 validation/clear, preserved message identity, no exclusive-mode assumptions | FR-009, FR-012, FR-013, FR-014 | UT-007/009, IT-003, REG-004 |
| `service-status/example.json`, `contract-fixtures.json`, `README.md`, `docs/operations/service-status.md` | Change | Examples/corpus, revision lifecycle, image CORS and clear/rollout/reload evidence instructions | FR-012, FR-013 | UT-007, IT-003, MAN-002 |
| `service-status/src/index.js`, `service-status/test/worker.test.js` | Mostly retain; update fixtures | Same read-only bounded JSON transport; no new routes/resources | FR-012, FR-013, NFR-003 | IT-001/003, REG-004 |
| `app/test/service_status/` existing suites; new `app_version_eligibility_test.dart`, `update_required_test.dart`, `message_cta_test.dart` | Change / create during TDD | Planned unit/widget/integration tests and schema-2 fixtures | All Must requirements | AT-001–AT-011, UT-001–UT-009, REG-001–REG-004 |
| `app/test/observability/service_status_diagnostics_test.dart` | Change | New operation/privacy/failure-path sink coverage | NFR-003 | IT-006, REG-004 |
| `app/integration_test/critical_journeys_test.dart` and status helper | Change | Actual preferences adapter recreation, native plugin and loopback schema-2 behavior | FR-001, FR-008, NFR-001 | IT-001/002, REG-003 |
| `web/test/service-status-reload.spec.js`, `web/test/service-status-reload-server.py`, `web/playwright.status.config.js` | Create | External Playwright runner survives target Flutter app reload; dedicated isolated loopback server | FR-005, NFR-001 | IT-005 |
| `app/test/service_status/support/status_reload_harness.dart` | Create | Compile-time-selected fixture entrypoint using production status host/controller/actions, deterministic app state and safe version marker | FR-003, FR-005, NFR-001 | AT-005, IT-005 |

No Go/AppView, lexicon, database migration, production Blueprint or public-site deployment changes are planned. Existing status repository transports remain intact apart from model fixtures/imports and any safe shared bounded-reader extraction justified by image loading.

## 5. Services, Interfaces, And Data Flow

### Formal JSON Schema and publication gate

Create `service-status/app.schema.json` using JSON Schema Draft 2020-12, with a stable versioned identifier such as `urn:craftsky:service-status:2` and local `$defs`/fragment references only. It formalizes the structure below: required fields, exact schema version, object/string types, revision/SemVer syntax, text bounds, optional/null behavior, named destinations and mutually exclusive CTA targets. Preserve the selected unknown-field policy rather than silently making additional properties invalid. Include descriptions for editor/tool support. This is the normative structural contract, checked against examples and the Flutter decoder through shared fixtures.

Use Python `jsonschema.Draft202012Validator` in the existing Python tooling, with an explicitly configured format checker for declared formats: `format` is not automatically an assertion in every validator configuration. Retain explicit URL/calendar validation and version comparison. Byte size, surrogate rejection, minimum ≤ maximum and publication identity rules remain semantic checks after formal schema validation. Do not add custom schema keywords that generic validators would silently ignore.

Pin the validator and required format dependencies in `service-status/requirements.txt`, with reproducible project-local environment setup documented in the runbook. Select exact package versions during implementation. Both Python commands and the deterministic status-test recipe use that configured interpreter. Validation/publication must not install packages or retrieve schemas. Missing dependencies/schema, invalid schema or validator failure exits safely before upload, with no fallback that bypasses the schema gate. All references resolve locally; disable network resolution.

Validation order is bounded bytes → JSON decode → formal schema → semantic checks → canonical recognized fields. Validate drafts as applicable and always the exact final encoded payload after root-revision generation, before writing. This gate applies to hosted publish, local publish, setup and clear. Preserve environment selection, confirmation and public verification. Tests meta-validate the schema itself and inject invalid final payloads to prove zero writes. Flutter keeps its typed parser without a runtime JSON Schema dependency; shared positive/negative fixtures prevent drift.

References: [JSON Schema Draft 2020-12 validation](https://json-schema.org/draft/2020-12/json-schema-validation) and [Python jsonschema validator API](https://python-jsonschema.readthedocs.io/en/v4.17.0/validate/). Pin the implementation dependency rather than assuming an implicit latest package.

This addition updates planning only; the formal schema and validation code are created during TDD, beginning with UT-009 alongside UT-007.

### Schema 2 wire contract

All keys are camelCase. Root has required integer `schemaVersion: 2` and required document `revision`. Optional sections are absent when disabled; supplied null sections/fields are invalid. An empty object is not a maintenance/message section. Unknown fields are ignored and stripped by operator canonicalization, preserving the existing forward-field policy; recognized fields are strictly validated before the whole snapshot is accepted.

```json
{
  "schemaVersion": 2,
  "revision": "document-draft",
  "maintenance": {
    "title": "CraftSky is taking a short break",
    "description": "We are carrying out maintenance.",
    "estimatedRecoveryAt": "2026-10-20T18:00:00Z"
  },
  "minimumVersions": {
    "ios": "1.8.0",
    "android": "1.8.0",
    "web": "1.8.0"
  },
  "message": {
    "revision": "message-example-a",
    "minVersion": "1.8.0",
    "maxVersion": "1.9.0",
    "title": "A CraftSky update",
    "description": "Read about the latest changes.",
    "imageUrl": "https://public.example.test/craftsky-update.png",
    "cta": {"text": "Open settings", "destination": "settings"}
  }
}
```

The illustrative image URL is a fixture placeholder, not a new hosted resource. Sections coexist. Full clear is `{"schemaVersion":2,"revision":"draft"}` before publisher revision replacement. `minimumVersions: {}` is valid and imposes no minimum. A platform key is optional; supplied values must be valid versions.

Maintenance requires `title` and `description`, with the current optional RFC3339 recovery estimate. Message requires revision, minVersion, title and description. `maxVersion`, `imageUrl` and `cta` are optional. Message CTA is either `{text, destination}` or `{text, url}`; exactly one recognized destination field must be present. URLs must be absolute HTTPS with nonempty host and no userinfo; do not add schemes, trim/rewrite destinations or accept arbitrary route paths. Reject control characters and malformed URIs. Image URLs use the same URL syntax boundary. Destination tokens are the exact enum names `feed`, `projects`, `search`, `notifications`, `profile`, `settings`.

Retain 16,384 bytes per UTF-8 document; document/message revisions use the existing ASCII `[A-Za-z0-9_-]{1,64}` contract. Title limit stays 160 Unicode scalars, description 2,400; CTA text uses the same 160-scalar short-text limit. Required text must be nonblank and contain no unpaired surrogate scalars. Optional fields, if present, must meet their constraints. Preserve timestamp validation and real-calendar/zone constraints. No schedules or automatic clearing are introduced.

Schema 1 is not accepted by the new parser/publisher: old-mode records fail open according to the new failure policy. Keep old-client/new-document rollout limitations explicit. Do not silently map exclusive-mode JSON into independent settings. Coordinate eventual client release and operator publication separately; publication cannot retrofit the new flow into old installed builds.

### Versions and eligibility

Updated by D-015 (2026-10-21): keep `StatusAppVersion` as a small adapter over the existing `pub_semver` dependency. Validate strict SemVer syntax (three core components, no leading-zero numeric core/prerelease identifiers, legal prerelease/build identifiers), then strip `+build` before `Version.parse`. Delegate precedence to `Version.compareTo`; do not use Pub version-range semantics. Core and numeric prerelease identifiers have at most 15 decimal digits, keeping all compared integers exactly representable on both native and web. Nonnumeric prerelease identifiers retain ordinary lexical precedence. Build identifiers are ignored and have no numeric limit beyond the document byte bound. The formal schema and Python validator enforce the same numeric grammar; the Python comparator remains for operator range validation. Shared fixtures and native/Chrome eligibility tests cover boundaries, strict rejection, prereleases, metadata omission and inclusive ranges. No dependency was added or upgraded.

`minVersion <= current <= maxVersion` for messages; absent maximum is unbounded. Required update is `current < minimumVersions[currentPlatform]`. Invalid runtime version returns unknown; it does not invalidate valid maintenance. The runtime version provider calls `PackageInfo.fromPlatform()` independently of device-info, preferences, session or `appDependenciesProvider`. It can reuse the plugin's internal cache but must not await unrelated initialization. A package-info failure produces unknown version and existing protected diagnostics, not a thrown status-provider construction failure. Detect web before `defaultTargetPlatform`; native host tests override the supported-platform provider instead of adding desktop app support.

### Action boundaries and destinations

Illustrative signatures only:

```text
StatusAppVersion? readRuntimeVersion()
StatusPresentation selectPresentation(snapshot, runtime, trust, dismissal)
Future<bool> openOfficialStore(StatusPlatform platform)
Future<bool> openMessageUrl(Uri validatedUrl)
void reloadCurrentWebApp()
Future<Uint8List> loadPublicMessageImage(Uri validatedUrl)
Future<bool> activateMessage(revision, invokeDestination)
```

Use existing `launchExternalLink`/`url_launcher` primitive via an injectable `statusExternalLauncherProvider`; do not reuse void-returning confirmation helpers that swallow outcomes. The message itself supplies the deliberate CTA; no extra confirmation flow is added. Invoke browser launch synchronously from the user gesture before awaiting persistence so popup blocking is not introduced by saving first. Official app links are fixed app configuration: iOS `https://apps.apple.com/app/id6812428632`, Android `https://play.google.com/store/apps/details?id=social.craftsky.app` (present in existing publication tests; actual listings still need MAN-001). No remote store URL override is needed.

Web reload adapter imports `package:web` only behind conditional export, calls the current document's location reload once on tap, and retains path/query/fragment without inventing a cache-busting URL. Native/stub reload is unavailable and never selected on supported mobile. Do not automatically reload on policy observation.

Map named destinations to existing `FeedRoute`, `ProjectsRoute`, `SearchRoute`, `NotificationsRoute`, `ProfileRoute` and `SettingsRoute`, with default parameters and normal `.go(context)` semantics. Do not construct destination pages directly or bypass the router/account/policy gates. Router acceptance/redirect counts as successful in-app activation; it does not claim the user reached protected content while signed out. Only enable in-app actions when a ready router exists; under loading/error hosts retain the message and show a retryable unavailable-action state if needed. Recheck revision/current presentation immediately before invoking an action.

### Public image loading

Updated by user decision D-014 on 2026-10-21: use `Image.network` for optional images from a trusted public HTTPS host. Keep the image decorative, bounded to 200 logical pixels high with `BoxFit.contain`, and fail-soft with an error builder. No AppView client or authorization/account/device headers are supplied. The standard Flutter image provider owns request/caching behavior; remove custom native/web byte loaders, auto-disposed providers and their transport-specific tests. Cookies/referrer/redirects, fetch size/deadline and cancellation follow platform defaults. Require a host compatible with Flutter web CORS; no custom HTML-image fallback is introduced.

AT-009 retains rendering, pending/failure fallback, layout, dismissal/CTA usability and cover-removal coverage through the real NetworkImage path with disposable HTTP fixtures. IT-006 retains status transport isolation and image diagnostic sink coverage. The earlier 5 MiB/ten-second/explicit-omit/abort requirements are superseded for images only; status JSON transport is unchanged.

### Operator and persistence contracts

Keep current CLI names and explicit environment/confirmation/public-verification behavior. `publish` and local publish regenerate only the root revision; preserve the supplied `message.revision` exactly. Update all mode-based summaries/clear logic in both Python commands. Preserve draft files, verified/unverified receipts, bounded reads, no retry/rollback, and fixed endpoint selection. Removing maintenance means publishing the complete document with only that section omitted, leaving minima/message intact; full `clear` visibly removes everything.

Resolve DR-003 by retaining bounded existing semantics: persist only the most recently dismissed message revision, not a history ledger. Operators must use a fresh message revision for each new campaign or intended resurfacing; do not reuse an older revision after a different campaign. Revisions may remain identical through unrelated document publications. Document and test A dismissed → unrelated republish A stays hidden; A dismissed → B dismissed → reused A can resurface under the bounded contract, with operator instructions forbidding that reuse. This narrows ambiguous historical wording to existing current-message behavior without adding cross-campaign history.

Use a new preferences key `service_status.dismissed_message_revision` for schema 2. Do not migrate the old `service_status.dismissed_revision` into unrelated schema-2 identities; leave the old key untouched. Interface shape stays one `readRevision`/`writeRevision`; changing names is unnecessary churn. No maintenance/update persistence and no account identifiers are stored.

## 6. State, Providers, Controllers, Or DI

Keep handwritten feature providers and `NotifierProvider`; do not turn polling/mutations into `AsyncNotifier` retries. Existing repository, reporter, dismissal-store and clock providers remain override seams.

```text
statusPlatformProvider + statusRuntimeVersionProvider (FutureProvider)
                       ↓
serviceStatusRepositoryProvider → ServiceStatusController ← dismissalStore + clock
                       ↓ immutable ServiceStatusState
                one derived presentation
                       ↓
              ServiceStatusHost + back dispatcher
                       ↓ user tap
       statusExternalLauncherProvider / reload adapter / typed router action
                       ↓ completion captured by revision
           session suppression + serialized persistence

visible message imageUrl → Image.network → Flutter image cache / platform transport
```

State additions: accepted `document`, existing fetch/dismissal flags, `updatePolicyTrusted`, current-session dismissed revision, action-in-flight revision and bounded UI failure status. Keep the persisted dismissal result distinct from session suppression; no mutable sentinel hidden in `copyWith`. Support explicit field clearing rather than `value ?? oldValue` for nullable state transitions. Maintain the existing maintenance accepted-at clock sample/timer independently of update trust.

Transitions:

1. Valid current-epoch fetch: atomically replace document, set update trust true, reset/renew maintenance accepted-at only for that valid snapshot, reevaluate presentation. Missing sections truly clear. Unknown version suppresses only version-dependent flows.
2. Fetch failure/timeout/synchronous start or repository-construction failure: set update trust false in the same finish path; retain the previous snapshot for still-fresh maintenance and existing message behavior; do not renew maintenance. A late timed-out success must not restore trust. Diagnostic failure cannot prevent invalidation.
3. Maintenance expiry: reevaluate once from monotonic/wall-safe existing logic. It must not set update trust true. Foreground resume checks expiry and fetches without extending trust before success.
4. Presentation: fresh maintenance → trusted applicable minimum → eligible undismissed message (after dismissal load) → none. Use a single selector/getter everywhere so host, splash and back handling agree. Watch state and async runtime changes so a resolved version triggers reevaluation without restarting/canceling in-flight status requests. Do not watch an async dependency in a way that rebuilds and resets the Notifier's timers.
5. Explicit dismissal: retain existing awaited-write behavior/failure recovery; no accidental broad persistence behavior change. Capture message identity before await; an older completion cannot clear a new message's state. Serialize writes so final disk state cannot be overwritten by an older outstanding write.
6. CTA: capture the visible revision, fence duplicate taps, invoke the destination, and only on successful activation mark that captured revision session-dismissed before attempting persistence. Persist success updates the durable marker; persist failure leaves session suppression and reports safely without reopening/relaunching. Keep a small current-session set only for failed CTA persistence if multiple campaigns are followed in one process, so repeat reads/account switches cannot resurrect them; no durable historical ledger is added. Never overwrite a newer message's pending/error state with an older completion.

Use the provider's process lifetime (not account scope) for session suppression. No status action signs out, replays writes, or cancels accepted remote work. Epoch/disposed checks and typed causes/stacks remain at the current operation owner.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

`ServiceStatusHost` retains `widget.child` at the same stack position with stable keys and existing pointer/focus/semantics exclusion. Select one presentation branch. Both maintenance and update are full-page `Material`/SafeArea/scrollable covers with labelled route semantics and focused actions; message remains `CraftskyDialog`. No recreation of router or underlying editor due solely to status changes. Incoming navigation may still be processed through normal routing, but it cannot expose interactive covered content; do not invent a deep-link queue/replay policy.

Change `serviceStatusRouterConfig` from a maintenance-only predicate to a shared blocking predicate. In ready app, root dispatcher consumes both blocking states before message dismissal. Add a common `PopScope`/back guard suitable for the loading/error MaterialApp hosts too, and test their system-back paths; those hosts currently do not use the ready router wrapper. Route/deep-link changes do not remove a cover. Extend the splash-removal condition in `App` to either blocking presentation so an independent update page is not hidden behind native splash while account dependencies are pending/erroring.

Update page copy is fixed localized text with an Update button on mobile and Reload on web. Include pending launch prevention and inline localized failure feedback so it works even without a ready app messenger. Retain polling; do not clear the gate because launching a store succeeded. Status failure or valid minimum removal controls its removal. Do not add a new loading blocker before the initial status result: existing app loading/navigation can proceed, and a valid applicable policy covers it as soon as accepted.

Message includes required plain text, optional image area, optional CTA and existing Dismiss. CTA busy state is associated with the captured revision; disable duplicate activation without freezing incoming status refreshes. Required update and maintenance have no dismiss action. Covering a message removes its image widget; Flutter owns the image request/cache lifetime under D-014. It does not cancel an already-submitted external action; a pending completion cannot lower the new cover. Focus restoration uses existing retained-focus validity checks, including when transitions go directly between blocking flows.

No AppView route/API/error envelope changes are planned. Typed existing routes are consumers, not new `/v1/*` endpoints.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Startup unknown/offline/invalid | No restored update/maintenance; bounded three-second check then ordinary access attempts with existing account gates | RULE-003, FR-001 | AT-010, UT-004/008 |
| Unknown runtime version or unsupported host platform fixture | Maintenance still works; no update/message; override platform in host tests | FR-002, RULE-003 | UT-002/008 |
| Accepted policy then failed refresh | Invalidate update trust immediately, preserve unrenewed maintenance until deadline; no resurrection on expiry | RULE-001, RULE-003, FR-006 | AT-011, UT-005/008 |
| New valid snapshot removes a section | Replace atomically and reevaluate; no `copyWith` null-retention bug | BR-001, RULE-001 | AT-001, UT-003 |
| Store/message URL false/throws | Localized actionable feedback, retain applicable page/message; no assumed update/dismissal | FR-004, FR-008, FR-010 | AT-004/008, IT-004 |
| CTA succeeds, dismissal write fails | Session-hide captured revision, permit recurrence only after process restart; never replay action | FR-008, FR-010 | AT-008, IT-002/004 |
| Replacement/cover while CTA awaits | Completion affects captured revision only; no new message dismissal or cover removal | FR-008, RULE-001 | AT-008, UT-006 |
| Image slow/broken/non-CORS/oversized | Omit/fallback media; keep text/dismiss/CTA reachable; standard Flutter request lifetime (D-014) | FR-011, NFR-003 | AT-009, IT-006 |
| Web reload gets same build | Accepted still-applicable policy gates again; no auto loop; deployed cache evidence remains manual | FR-005 | AT-005, IT-005, MAN-002 |
| Persisted-dismissal read fails | Resolve readiness without permanent loading, show eligible current message and allow recovery; report original cause safely | FR-008, NFR-003 | UT-006, IT-002/006 |
| Diagnostics throw | Outcome/freshness/eligibility unaffected; expected offline/validation stays expected, no duplicate issue owner | NFR-003 | IT-006, REG-004 |

## 9. Test Implementation Plan

Each row is a red-green-refactor slice or a small related group. Use the existing test ID references; do not create tests that merely mirror model fields or mock the selector being tested.

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-007/009, IT-003 | Schema/meta-schema, Dart document suite, Python contract corpus | One literal valid schema-2 all-controls fixture, empty clear, invalid sibling section | Current parsers reject schema 2 and no formal-schema gate exists. Prove structural/final-payload rejection causes zero writes; missing test dependencies are setup failures, not meaningful red |
| 2 | UT-001/002, AT-002 | New eligibility suite | Independently expected strict SemVer/range/platform fixtures; metadata variants and prereleases, 15-digit numeric boundaries and oversized rejection | Current model lacks strict independent eligibility; raw pub_semver build comparisons would disagree |
| 3 | UT-003/004/005/008, AT-001/010/011 | Controller/app/restart suites | Injected clock, controlled completers, runtime-version override, combined snapshot then failure and expiry | Existing controller has one exclusive mode, no update-trust invalidation or independent priority |
| 4 | UT-006, AT-006/007, IT-002 | Dismissal/persistence suites and native journey | Same A across document revisions, B replacement, delayed writes, fresh preferences instances; reuse-sequence test per DR-003 | Existing root revision controls dismissal; key/identity and captured completion semantics differ |
| 5 | AT-003, REG-001/002/003 | Host/router/editor/recovery/back tests | Real retained editor/account gates and ready/loading/error app hosts | No update cover/back/splash integration; ensure current maintenance regressions stay protected |
| 6 | AT-004/005, IT-004, MAN-001 | Update screen/action adapter tests | Fake launcher returning false/throw/success, platform listing capture, injected reload counter | No correct-platform update action/reload or failure UI |
| 7 | AT-008, IT-002/004 | Message CTA suite | Actual typed router plus account gates; launcher/activation and persistence completers; new cover/revision during await | Current dialog lacks CTA and session-only fallback/race handling |
| 8 | AT-009, IT-006, MAN-003 | Image widget/accessibility and sink suites | NetworkImage pending/valid/broken fixtures, protected diagnostic canaries, light/dark/narrow/wide/scaled layouts | Verify optional-media presentation and serialized sink positives/exclusions; standard trusted-host network loading per D-014 |
| 9 | IT-003, REG-004 | Python/Node/local tooling | Publish all controls then omit maintenance; unchanged message revision; clear/decline/EOF/unverified outcome | Current publisher/local helper assumes mode and regenerates root-only announcement identity |
| 10 | IT-001/002/006, REG-001–REG-004 | Transport/device/regression suites | Existing native/browser fetch harness, loopback status, real storage plugin | Update existing schema-1 fixtures; no artificial expected failure needed for unchanged protective regressions |
| 11 | IT-005, AT-005, MAN-002 | Dedicated external browser reload spec | Two compiled safe harness builds and separate loopback status server, described below | Current feature has no reload adapter/CTA; same running tab cannot recover to build B |

### Browser harness for DR-004

Reuse the pinned `@playwright/test` already in `web/package.json`, with a dedicated config/testMatch/server so public-site tests and their server remain separate. Do not run a reload inside the Flutter test-runner document. Build `status_reload_harness.dart` twice in debug mode into temporary directories (the existing status endpoint validator permits loopback overrides only in debug), explicitly assigning version A `1.8.0+1` and B `1.9.0+2`; actual compiled version metadata must be distinguishable through a harmless visible/semantics marker. Use the production package-info runtime provider, status host/controller, actions and reload adapter. Enable Flutter semantics in the harness so Playwright can observe labelled actions and the harmless version marker without relying on canvas coordinates. Override only unrelated app/session dependencies and debug loopback status origin, not the reload implementation or version comparison.

The external Python server serves an atomically switchable build root at a fixed loopback origin with SPA fallback for a non-root path plus query and proper asset MIME types, no-store on unversioned entry/bootstrap/version resources, and the independent schema-2 status endpoint. Playwright owns a separate process/page and survives reload. Open A, retain that same page, switch server root to B and set minimum `1.9.0`, wait for an accepted update page, tap Reload once, assert a real document navigation and compiled B marker/current URL. Assert no repeated navigation after recovery. Separate cases keep A served (gate returns) and make startup status unavailable (fail open). Capture safe results/request headers, not credentials. Do not manually unregister a worker or clear the entire browser cache to force success.

This harness proves the adapter and local build transition only. Actual production/preview bootstrap, browser and CDN behavior still requires MAN-002. Existing checked-in `scripts/web-deploy` deploys the public website; it is not evidence of a Flutter app deploy/cache path. No modification or deployment of that unrelated site is planned. Discover the real Flutter hosting/cache configuration before any separately authorized release work; if cache fixes are needed, prepare a scoped concrete change then.

### Focused commands

Run from repository root during implementation unless a `cd app` command is shown. Commands are planned, not executed here.

```sh
just service-status-test
just app-test --no-pub test/service_status/service_status_document_test.dart
just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart
just app-test --no-pub --platform chrome test/service_status/service_status_repository_web.browser.dart
just app-test-integration DEVICE --no-pub
just app-analyze
```

For the external harness, build both artifacts with `flutter build web --debug --target test/service_status/support/status_reload_harness.dart --build-name ... --build-number ... --output ...` from `app/`, passing only the local harness/status defines needed. Use two explicit temporary output paths, then `npm exec --prefix web -- playwright test --config web/playwright.status.config.js` from root with those paths supplied via task-specific environment names. Finalize CLI flag support against the installed SDK help before execution; do not install or upgrade packages merely to run this test. Existing pinned tools may need normal environment setup, which must be reported if unavailable.

Generate model/localization output only after the relevant red/green slice requires it: `dart run build_runner build` and `flutter gen-l10n` from `app/`. Inspect `git status` immediately; avoid filtered generation that can delete unrelated generated files. Do not recreate desktop platforms. Run required focused gates once after final changes; expand only for new failures/risks.

## 10. Sequencing And Guardrails

- First TDD step: add formal-schema/meta-validation and publication-gate tests (UT-009) alongside the single literal schema-2 all-controls fixture and a meaningful expected-acceptance assertion in UT-007; demonstrate current rejection before changing model/parser. Pair Dart and Python acceptance/rejection before operator integration.
- Dependencies: contract/version rules → controller trust/priority → dismissal identity → blocking presentation → actions/media → operator workflow → integration/regression/platform evidence. Keep diagnostics independent of these decisions.
- Preserve the existing three-second status deadline and five-minute maintenance clock behavior. The ten-second optional-image budget is a separate media resource limit and never holds status processing or interaction open.
- No status policy persisted offline. No AppView credential/client use for public status/images. No PDS/lexicon/backend API changes, migration, feature flag platform or generic remote route executor.
- Capture typed causes and caught stacks using supported logging/reporter entry points and positive/protected serialized sink tests. Never interpolate documents, URL capabilities, recipients, account/device identifiers or dependency prose into diagnostics.
- Avoid resetting the mounted app/router for policy changes. Do not “fix” async races by ignoring state watches or awaiting diagnostics. No replay/cancellation of already-submitted writes by status UI.
- Full-document publication is last-write-wins as before; keep target preview and explicit confirmation, never automatically publish as part of app build/deploy. No live state changed by TDD fixtures.
- Preserve local-versus-live evidence. Failed store opening, unavailable CORS images, unrun physical-reader checks and unverified hosted cache behavior stay explicit in validation evidence rather than reported as passing.
- Out of scope: automatic installation/store lookup/reload, server enforcement, scheduling, historical campaign ledger, multiple messages, new upload hosting/admin UI, commits/push/deploy without separate authorization.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking design decision resolved | DR-003 bounded dismissal history | Reusing A after B dismissal may resurface A | Retain single durable revision; require fresh campaign identities, document/test the reuse sequence, no history ledger. |
| CPQ-002 | Non-blocking implementation verification | External browser harness requires valid pinned SDK build flags and installed Playwright browser | Local reload proof may be unavailable until normal tooling setup | Use separate runner and compiled A/B artifacts; inspect installed help, report environmental blockers without replacing proof with mocks. |
| CPQ-003 | Release verification dependency | Actual Flutter hosting/cache path is not established by the public-site deploy script | Reload can fetch stale assets under hosted caching | MAN-002 before claiming full AC-008 verification. Inspect real hosting after separate scoped authorization; no speculative public-site changes. |
| CPQ-004 | Release verification dependency | Official listing availability and physical platform/back/reader behavior | Widget fake success is insufficient | MAN-001/003 on iOS/Android/web with safe local/authorized preview configuration. |
| CPQ-005 | Non-blocking resource choice | Public image origin needs anonymous CORS access on web | Some supplied URLs render only a fallback | Standard fail-soft Image.network loading per D-014; document trusted-host/web CORS requirements; no proxy or custom fallback. |
| CPQ-006 | Non-blocking rollout dependency | New schema is not understood by old installed clients; new client fails open on schema 1 | Mandatory-update capability starts only after compatible app release/config publication | Keep rollout limitation in runbook, release app and publish separately, do not claim backward enforcement. |

No blocking product questions remain. If implementation exposes a need to alter the accepted flows, return that specific decision rather than silently changing requirements. Routine naming/harness details within this design do not require another approval loop.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-20-app-status-flows/04-coding-plan.md`.
- Execution evidence/plan to create at implementation stage: `05-implementation-plan.md` using `implement-tdd`.
- Start with UT-007/009's literal schema-2 acceptance and formal-schema gate, then UT-001/002 before state/UI implementation. Record actual red failure, smallest green change and refactor per approved slice.
- First focused commands: `just service-status-test` and `just app-test --no-pub test/service_status/service_status_document_test.dart`.
- All Must test IDs have targets above; real-platform/manual portions remain release checks, not silently skipped passes. Preserve AC-021/022 failure/race coverage through refactoring.
- `01`/`02`/`03` remain the behavioral authority; this plan supplies schema/adapter/state/test architecture and resolves their routine design notes. No earlier artifact was rewritten in this stage.
- Status: Ready for user review before implementation. Only this document was added; no commit requested or created.
