# Coding Plan: Service Status and Maintenance Messaging

## 1. Inputs

- Requirements: `01-requirements.md`, approved scope and simplifications dated 2026-10-09.
- Tests: `02-acceptance-tests.md`, 20 Must requirements, 17 active criteria and 28 planned case families.
- Document review: `03-document-review.md`, **Approved with notes**, Medium risk. Address DR-001–DR-003 below.
- Repository context inspected: `app/lib/app.dart`, `bootstrap.dart`, `main.dart`, `app_dependencies.dart`, the API Dio providers, app-language/theme providers, diagnostic entry points/guide, app test and integration harnesses, `web/wrangler.json`, `web/package.json`, `web/playwright.config.js`, `scripts/web-deploy`, `justfile`, and path-aware PR checks. The AT Protocol reference and AppView API architecture were consulted; no AppView/PDS route or authentication change is planned.
- This artifact is implementation design only. No source, executable tests, generated files, dependencies, infrastructure or commits are changed in this stage.

## 2. Implementation Strategy

Use a **dedicated Cloudflare Worker reading one private R2 object**. Its public read endpoint is `https://status.craftsky.social/app.json`. The operator command replaces the R2 object directly; routine message publication deploys neither Worker code nor website assets. This separates mutable incident content from both code release systems. Do not use Workers KV or a public cached R2 domain for the current document.

R2 documents strong consistency for object reads/writes and notes that Worker binding access bypasses the public-domain cache. Use that binding directly, with no Worker Cache API and `Cache-Control: no-store` responses. This is the design basis for observing the next published state; still verify it on the selected hosted configuration. [Cloudflare R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/), [Worker R2 bindings](https://developers.cloudflare.com/r2/api/workers/workers-api-reference/).

Flutter uses a dedicated anonymous transport and a keep-alive Riverpod `Notifier` for last-valid status, polling and freshness. It starts alongside the existing `App` initialization and never watches account/auth/device providers. A presentation host inside each existing MaterialApp covers the retained child tree rather than replacing routes or editors. Only dismissed-announcement revision is persisted.

Use existing Python stdlib command-test patterns, pinned Wrangler from `web/node_modules`, Node's built-in runner for the small Worker, and the already pinned Playwright package for web/hosted smoke checks. No new runtime Flutter or Node package is anticipated. Installing the existing lockfile is tooling setup, not a website release.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| Public host | Landing Worker serves a complete static site | Separate read-only Worker and private R2 bucket; one fixed object; no shared release asset | BR-002, FR-001, FR-012, NFR-002 | AC-001, AC-002, AC-015, AC-016 | AT-008, IT-005, IT-007, REG-004 |
| Contract | Explicit JSON/HTTP boundaries | Shared literal fixture corpus and bounded Dart/Python validators | FR-002, FR-003, NFR-001, RULE-003 | AC-003, AC-004, AC-005, AC-019 | UT-001, UT-002, UT-007, IT-003 |
| Retrieval | Existing anonymous Dio still attaches device ID | Independent native/web transport without API interceptors, redirects or credentials | FR-001, FR-004, RULE-002, RULE-003 | AC-001, AC-006, AC-019 | IT-001, UT-003 |
| State/lifecycle | Riverpod keep-alive providers and app initialization | In-memory status controller, coalesced refresh, bounded freshness, foreground observer | FR-004, FR-007, FR-008, NFR-001, RULE-001 | AC-006, AC-007, AC-010, AC-011 | UT-003, UT-004, AT-005–AT-007 |
| Preferences | SharedPreferences and generated providers | Independent announcement store, one persisted revision, fresh storage read | FR-006, FR-008 | AC-009, AC-011 | UT-005, AT-007, IT-002 |
| Presentation | Separate loading/error MaterialApps and ready router builder | Common status host, maintenance cover, announcement notice; splash resolution on visible maintenance | BR-001, FR-005, FR-006, FR-009, NFR-003, RULE-004 | AC-001, AC-003, AC-004, AC-008, AC-009, AC-012, AC-017 | AT-001–AT-004, AT-006, AT-009, UT-006, REG-001–REG-003, MAN-001 |
| Operator CLI | Python deploy scripts, pinned Wrangler, public verification | Validate/publish/clear object commands with confirmation, generated revision and honest result | BR-002, FR-012, RULE-002 | AC-002, AC-009, AC-015, AC-019 | AT-008, IT-003, IT-004 |
| Diagnostics | ProviderLogger, Logger, ErrorReporter and filtered serialized sinks | Bounded status summaries, explicit consumed-failure owner, no remote prose | NFR-004, RULE-002 | AC-018, AC-019 | IT-006 |
| Validation | Flutter/Python suites; mobile process harness; path-aware CI | Focused recipes and a local status job; separate authorized hosted checks | FR-001, FR-004, FR-009, FR-012, NFR-002, NFR-003, NFR-004 | AC-006, AC-008, AC-015, AC-016, AC-017, AC-018 | UT-003, IT-005–IT-007, REG-003, REG-004, MAN-001 |

## 4. Files And Modules

Paths are proposed implementation targets; they are not created by this plan. Generated outputs are regenerated only during implementation and only for affected sources.

| Path / Module | Create / Change | Purpose | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| `service-status/wrangler.json`, `service-status/package.json` | Create | Separate Worker/bucket binding and preview environment; private module using existing tool dependencies | FR-001, NFR-002 | AC-001, AC-015, AC-016 | IT-005, IT-007 |
| `service-status/src/index.js` | Create | Fixed-path read handler, CORS/no-store headers, no public mutation route | FR-001, NFR-002, RULE-002, RULE-003 | AC-001, AC-016, AC-019 | IT-001, IT-007 |
| `service-status/contract-fixtures.json`, `service-status/example.json` | Create | Literal valid/invalid corpus and editable example, never the live state asset | FR-002, FR-003, NFR-001 | AC-005 | UT-001, UT-002, IT-003 |
| `scripts/service-status`, `scripts/service_status.py` | Create | Executable wrapper, validation, fresh-revision generation, R2 adapter, verification and command results | BR-002, FR-006, FR-012, RULE-002 | AC-002, AC-009, AC-015, AC-019 | AT-008, IT-003, IT-004 |
| `app/lib/service_status/models/service_status_document.dart`, `service_status_state.dart` | Create | Bounded decoding, DTO/state with safe diagnostic summaries | FR-002, FR-003, FR-008, RULE-003, NFR-004 | AC-005, AC-007, AC-011, AC-018, AC-019 | UT-001, UT-002, UT-004, UT-007, IT-006 |
| `app/lib/service_status/data/service_status_transport.dart`, `service_status_transport_native.dart`, `service_status_transport_web.dart`, `service_status_transport_stub.dart` | Create | Cancellable bounded read contract and conditional platform adapters | FR-001, FR-003, FR-004, RULE-002, RULE-003 | AC-001, AC-005, AC-006, AC-019 | IT-001, UT-002, UT-003 |
| `app/lib/service_status/data/service_status_repository.dart`, `announcement_dismissal_store.dart` | Create | Decode response and isolate preference storage from app/account readiness | FR-003, FR-006, FR-008 | AC-005, AC-009, AC-011 | UT-002, UT-005, IT-002 |
| `app/lib/service_status/providers/service_status_providers.dart`, `service_status_controller.dart` | Create | Keep-alive DI providers/controller, timeout/poll/expiry timers and revision dismissal | FR-004, FR-006, FR-007, FR-008, NFR-001, RULE-001 | AC-006, AC-007, AC-009, AC-010, AC-011 | UT-003–UT-005, AT-005–AT-007 |
| `app/lib/service_status/widgets/service_status_host.dart`, `maintenance_screen.dart`, `announcement_banner.dart`, `service_status_lifecycle_host.dart` | Create | Retained-tree cover/banner, accessible focus/semantics, global lifecycle wiring | BR-001, FR-005, FR-006, FR-009, NFR-003, RULE-004 | AC-001, AC-003, AC-004, AC-008, AC-009, AC-012, AC-017 | AT-001–AT-004, AT-006, AT-009, REG-002, REG-003, MAN-001 |
| `app/lib/service_status/service_status_router_config.dart`, `service_status_estimate_formatter.dart` | Create | Stable root back-dispatch wrapper; localized local-time estimate formatting | FR-005, FR-009, NFR-003, RULE-001 | AC-003, AC-008, AC-012, AC-017 | AT-002, AT-006, UT-006, AT-009 |
| `app/lib/app.dart` | Change | Start status alongside initialization, add host to all root variants, reveal maintenance under splash | FR-004, FR-005, FR-009, NFR-001 | AC-001, AC-006, AC-008, AC-012 | AT-001, AT-004, AT-006, UT-003 |
| `app/lib/bootstrap.dart` | Change | Override only the new status reporter provider with existing bootstrap reporter in both root scopes | NFR-004 | AC-018 | IT-006 |
| `app/lib/l10n/app_en.arb`, affected localization/provider generated outputs | Change / Generate | Built-in status controls/estimate strings and annotation-generated DI | FR-005, NFR-003 | AC-003, AC-017 | AT-002, AT-009, UT-006 |
| `app/test/service_status/`, `app/test/observability/service_status_diagnostics_test.dart` | Create | Test targets detailed in section 9 | FR-001–FR-009, NFR-001–NFR-004, RULE-001–RULE-004 | AC-001, AC-003–AC-012, AC-016–AC-019 | AT-001–AT-007, AT-009, UT-001–UT-007, IT-001, IT-002, IT-006 |
| `scripts/test_service_status_publish.py`, `service-status/test/worker.test.js` | Create | Python operator tests and Node handler tests with fake binding | BR-002, FR-001, FR-003, FR-006, FR-012, NFR-002, RULE-002 | AC-001, AC-002, AC-005, AC-009, AC-015, AC-016, AC-019 | AT-008, IT-003–IT-005, IT-007 |
| Existing app integration/auth/router/composer suites; `scripts/test_web_deploy.py`, `scripts/test_cloudflare_pages_build.py` | Change narrowly where needed | Extend reusable fixtures and protect existing boundaries | FR-001, FR-006, FR-007, FR-008, FR-009, FR-012, RULE-004 | AC-008–AC-012, AC-015 | IT-002, IT-005, REG-001–REG-004 |
| `service-status/playwright.config.cjs`, `service-status/test/hosted-status.spec.cjs`, `scripts/service-status-check` | Create | Opt-in native/web hosted read/cache smoke; isolated publication setup and bounded test budget | FR-001, FR-012, NFR-002, RULE-002 | AC-001, AC-015, AC-016, AC-019 | IT-007 |
| `justfile`, `.github/workflows/backend-ci.yml` | Change | Local deterministic status recipes and path-aware PR status-test job, no automatic publishing | FR-001, FR-003, FR-004, FR-012, NFR-002 | AC-005, AC-006, AC-015, AC-016 | UT-001–UT-004, IT-003–IT-005, IT-007, REG-004 |
| `docs/operations/service-status.md`, `service-status/README.md` | Create | Setup/change controls, publishing/clear commands, credentials, privacy and verification evidence | BR-002, FR-001, FR-012, NFR-002, RULE-002 | AC-002, AC-015, AC-016, AC-019 | AT-008, IT-003, IT-004, IT-007 |

No changes to `appview/`, `lexicon/`, Render configuration, website public-file allowlist or website runtime are planned. Website scripts receive test assertions only if needed; independent resource separation is primarily enforced by the new command's fixed target.

## 5. Services, Interfaces, And Data Flow

### Public contract and validation

Choose these implementation defaults to close GAP-001. They are engineering bounds selected by this plan, not a claim that the earlier illustrative numbers were approved product limits.

| Field / Bound | Design |
|---|---|
| `schemaVersion` | Integer `1`; reject other versions and wrong types |
| `mode` | Exact string `normal`, `announcement` or `maintenance` |
| `revision` | Nonempty ASCII token matching `[A-Za-z0-9_-]{1,64}`; publishers generate a UUID v4 string for each publication |
| `title`, `message` | Required nonblank strings for announcement/maintenance; title at most 160 Unicode scalar values, message at most 2,400; valid optional supplied text on normal must also be bounded, but is not rendered |
| `estimatedRecoveryAt` | Optional zoned RFC3339 string, whole seconds or 1–6 fractional digits; real Gregorian date/time, explicit `Z` or signed `HH:MM` offset; reject impossible dates, invalid offsets, leap-second values and supplied null/wrong types; no activation or expiry semantics |
| Whole document | UTF-8 JSON object, at most 16,384 bytes including unknown fields; enforce before parsing/while reading; non-200 or non-JSON response is not an accepted fetch |
| Unknown fields | Ignore within schema 1, including unsupported link/scope/schedule fields; do not turn them into widget or network controls |
| Publication time | Absent from required contract and examples; no secondary revision or document freshness timestamp |

Pin literal cross-language corpus fixtures with expected accept/reject results. Python measures code points and Dart measures runes; reject invalid Unicode representations consistently. Validate timestamps strictly rather than relying on permissive date parsers that normalize impossible dates. Store accepted estimates as an instant, format with `intl` and device local time, and omit the estimate if absent. No HTML/markdown/link parser is used.

The editable input is the same complete document schema, including a valid draft revision. `validate` checks that file without changing it. `publish` first validates input, then **replaces its draft revision with a new UUID**, validates/serializes the final document and shows that exact intended document before confirmation. This avoids a second draft format and makes every revision/replacement visible after a previous dismissal. `clear` generates a minimal valid normal document with a fresh UUID. The file itself is not rewritten. This resolves DR-001, including copied or reused draft revisions.

### Dedicated host

- Resource names proposed: Worker `craftsky-status`, private bucket `craftsky-service-status`, object key `app.json`, binding `STATUS_DOCUMENTS`. Preview has a separately named Worker/bucket and never uses the production binding. These are intended names, not verified remote resources/IDs.
- Production route is a Worker custom domain for `status.craftsky.social`. The bucket has no public R2 custom domain or `r2.dev` access. Routine publishing operates only on its fixed object key.
- Handler accepts anonymous `GET /app.json` and `HEAD /app.json`; answers matching CORS OPTIONS without auth. Other paths/methods return bounded 404/405 and never write. No operator/admin route is exposed.
- Fetch via `env.STATUS_DOCUMENTS.get('app.json')` on each read, with no public cache fetch or Cache API. Bound object size before returning it. Missing object is an ordinary non-200 unknown-status response, never an invented normal/maintenance document.
- Success headers: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`, `Access-Control-Allow-Origin: *`, and `X-Content-Type-Options: nosniff`. Do not set `Access-Control-Allow-Credentials`. CORS/no-store also applies to bounded errors. Do not emit cookies or reflect arbitrary request data.
- Worker runtime catches expected storage failures into a generic bounded error response, with static classification if logging is needed. Do not log document text, request headers, secrets or account data. No Sentry SDK/dependency is added to this minimal public host.
- Initial provision and DNS/binding deployment are separate operator actions after authorization, not a side effect of `publish`. Start with normal only during that separately authorized setup. A Worker code deploy must leave the R2 object untouched.

### Operator commands and verification

Public CLI surface:

```text
scripts/service-status validate <json-file>
scripts/service-status publish <json-file> --environment production|preview
scripts/service-status clear --environment production|preview
scripts/service-status-check --environment preview
```

Environment selection is mandatory for mutations. Checked-in configuration contains only non-secret names/origins; account context is resolved from the existing Cloudflare account configuration or explicit operator environment, never a credential copied into files. Read-only discovery must establish the intended account/bucket before confirmation. No Git-clean/main-branch requirement, website build or Render probe belongs to message updates.

The operator sees target account/environment/bucket/public URL and intended public title/message/mode/revision. Explicitly label content as public. Confirmation precedes the first mutation; decline/EOF/invalid file performs no upload. No implicit `--yes` mode in the initial command. Use subprocess argument arrays, a frozen temporary JSON file and bounded sanitized output; do not interpolate credentials/JSON into shell text or print raw Wrangler environment/error payloads.

`WranglerStatusObjectStore.put(document)` invokes the pinned Wrangler `r2 object put <bucket>/app.json --file <temp-file> --content-type application/json --cache-control no-store --remote` with explicit account/config context. The supported object command is documented by Cloudflare; inspect the pinned CLI help during implementation before relying on optional flags. Credentials remain in Wrangler's authorized login or operator environment and outside the app/repository. [Wrangler R2 commands](https://developers.cloudflare.com/workers/wrangler/commands/r2/).

After attempted mutation, fetch the **fixed public URL** anonymously with redirects disabled, finite size/type/time bounds, and compare all intended recognized fields including the fresh revision. Three verification attempts, each at most three seconds, separated by at most one second; final result is bounded. Success requires the intended live document and expected no-store/JSON/CORS headers. A matching upload result or private bucket read is insufficient. Later concurrent publication with a different revision yields an unverified result, not false success. Current scope remains last-write-wins with no operator lock/history.

Results: verified success exits 0; cancellation/validation/setup failure exits nonzero without mutation; mutation/verification failure exits nonzero and conservatively says live state may have changed if upload was attempted. No rollback, destructive cleanup or write retry is automatic. Restoring another mode requires a new confirmed publication. Keep receipts limited to environment, operation, revision and verified/unverified outcome; do not persist credentials or full content in logs.

### Interfaces and flow sketch

```text
ServiceStatusTransport.start(uri) -> StatusRequest { result, cancel() }
ServiceStatusRepository.fetch() -> cancellable validated document result
AnnouncementDismissalStore.readRevision() / writeRevision(revision)
ServiceStatusClock.now() -> in-process age samples
ServiceStatusController.refresh(trigger) / setForeground(bool) / dismissAnnouncement()
StatusObjectStore.put(finalDocument) / PublicStatusVerifier.verify(intendedDocument)

operator file -> Python validator -> fresh revision -> confirmation
             -> R2 object replacement -> public Worker verification

app lifecycle -> controller -> isolated transport -> bounded validator
              -> accepted in-memory state -> retained-tree status host
announcement dismissal <-> independent local preferences
```

This is an interface sketch, not production implementation. Storage/public verifier/confirmation/clock/UUID generator are injectable boundaries in Python tests. Flutter tests replace repository/transport/clock/store with deterministic fakes.

## 6. State, Providers, Controllers, Or DI

Use annotation-generated keep-alive providers where this feature follows existing Riverpod conventions. Use `Notifier<ServiceStatusState>`, not `AsyncNotifier` state that replaces the whole UI with fetch loading/errors. Expose fetch-in-progress separately from last-valid data; request failure does not erase accepted content or block available app UI.

```text
serviceStatusEndpointProvider (Provider<Uri>)
serviceStatusClockProvider (Provider<clock>)
serviceStatusTransportProvider (Provider<transport>, onDispose cancels/closes)
    -> serviceStatusRepositoryProvider (Provider<repository>)
announcementDismissalStoreProvider (FutureProvider<store>, keepAlive)
serviceStatusReporterProvider (Provider<ErrorReporter>, bootstrap override)
    -> serviceStatusControllerProvider (keepAlive Notifier<ServiceStatusState>)
    -> serviceStatusPresentationProvider (derived Provider)
serviceStatusRouterConfigProvider <- existing goRouterProvider
    reads current maintenance predicate without reconstructing router on mode changes
```

- Endpoint defaults to `https://status.craftsky.social/app.json`. A compile-time `CRAFTSKY_STATUS_URL` permits loopback/isolated test builds; production configurations must omit it or use the exact production HTTPS endpoint. Reject credentials/fragments and unsupported schemes. Debug/test loopback is explicit. Remote JSON cannot change the endpoint. No endpoint depends on the API base URL or active account.
- Controller starts once from `ServiceStatusLifecycleHost` around `App`, alongside dependency/account initialization. No eager awaited status request is added to `bootstrap`.
- `ServiceStatusState` contains last accepted document, successful-fetch age sample, foreground flag, fetch flag and dismissal readiness/revision. Its `toString()` exposes only bounded mode/loading/outcome, never prose, document, revision or private context. DTOs/store result wrappers also have safe summaries because ProviderLogger formats provider values in debug logs. Do not expose a bare persisted revision string as a logged provider value.
- Launch/resume/retry refresh immediately; a single 60-second timer runs while foregrounded in all modes. Cancel polling outside foreground. Overlapping triggers share the active request/future. Request timeout is an overall three-second deadline covering transport/body/validation, not merely separate connect/read timeouts. Abort on deadline/disposal and reject late completions by request epoch before accepting data.
- On accepted maintenance, record successful fetch age and schedule expiry for five minutes. An unchanged successful validated document renews it. Invalid/network/non-200 failures leave document/age untouched. Accepted normal/announcement clears maintenance immediately. A stale request cannot override a later accepted result, including one after cancellation.
- Inject an age clock with a running `Stopwatch` and UTC wall sample, retaining only in memory. Use the larger nonnegative elapsed duration so foreground monotonic elapsed and long suspended intervals both expire promptly on resume. Backwards wall time invalidates trust conservatively rather than extending it. Evaluate freshness at build/refresh/resume as well as the expiry callback; never depend solely on a suspended timer. No persisted clock reconstruction.
- Expired maintenance may remain as last accepted data internally but no longer authorizes a cover. UI permits normal attempts and makes no recovery claim. No API health/auth error feeds maintenance state.
- Announcement storage uses `SharedPreferences.getInstance()` independently of `appDependenciesProvider`/its synchronous accessor. Use one key `service_status.dismissed_revision` under the existing platform namespace; no account ID. Do not change global prefix initialization in this feature. Readiness of dismissal storage does not delay status fetching/maintenance; suppress an announcement briefly while the initial dismissal read is pending to avoid same-revision flash. Storage failure falls back to an available notice with normal app access and bounded diagnostics, never global loading.
- Await a successful preference write when dismissing; return/report failure without claiming durable success. During that write, keep the current notice visible or mark only its dismiss action busy; maintenance is unaffected. After success, mark that captured revision dismissed, not a newer revision that arrived meanwhile. Do not delete an old dismissal simply because normal/maintenance was observed; a newly published notice has a new UUID.
- Resolve DR-003 by disposing feature instances, calling the selected preference API's reload/read boundary after the completed write, and verifying the revision from persisted storage before recreating UI/controller in IT-002. Mock preferences alone are not the device persistence evidence.
- Existing native bootstrap already awaits platform setup before `runApp`; this feature adds no delay there and does not overhaul unrelated plugin startup. App/account failure after `runApp` cannot prevent the independent status host.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Root composition and native splash

Keep `_ReadyApp`, `_LoadingApp` and `_ErrorApp` chosen by their existing dependency/auth conditions, never by status mode. Add `ServiceStatusHost` to the builders of **all three** MaterialApps. Place it outside `NotificationEffectHost` and `ActiveAccountInitializationGate` in the ready builder, so status can cover the root navigator, dialogs and failed account gates while they remain mounted. Reuse `TextScaleFactorClamper`/`FormFactorWidget` around the presentation as appropriate for each root.

```text
App lifecycle owner (starts independent status controller)
  existing loading/error/ready MaterialApp choice
    localized/themed builder
      ServiceStatusHost (stable wrappers and child position)
        normal app content, retained in tree
        foreground maintenance cover OR announcement notice
```

Normal content remains in a stable `Stack`/layout slot with `IgnorePointer`, excluded focus and excluded semantics toggled during maintenance. Do not substitute a status MaterialApp for a ready app, conditionally unmount the router, push/pop a maintenance route, or reset composer/provider/account state. Use an opaque, full-viewport accessible maintenance surface above all in-app overlays. Announcement occupies a stable notice slot above the ordinary content and keeps operations available; use bounded scrollable layout rather than text clipping.

When fresh maintenance becomes visible, call the existing one-shot `_scheduleInitializationResolved()` to remove the native splash even if `_coldStartComplete` is false. **Do not mark account initialization complete** to accomplish this. Without maintenance, retain the current splash removal conditions. Add this assertion to AT-001/UT-003 app wiring coverage.

Theme/locale accessors currently require resolved app dependencies. Loading/error roots use `AppTheme` and supported device/system fallback locale/theme without watching those synchronous accessors; ready UI uses existing `themeModeProvider`/`appLanguageProvider`. Only English is currently supported, so localization checks cover the existing locale plus device fallback, not invented translated locales.

### Navigation, focus and operations

- Keep the existing GoRouter instance/route information provider/parser/delegate. Build one stable `RouterConfig` adapter for the ready MaterialApp that delegates these existing objects and substitutes a root back-button dispatcher which consumes internal back actions while maintenance is active and otherwise delegates existing behavior. Do not recreate the adapter/router for every status update. A `PopScope` outside the Router alone is insufficient because it has no active ModalRoute.
- Test Android back/predictive-back and web keyboard/back integration at the available harness boundary. Browser URL/account/auth events may continue according to existing router rules; they cannot dismiss the visible cover or reveal covered controls. This feature does not add OAuth cancellation or backend admission restrictions.
- Maintenance entry saves the appropriate focus target, moves focus into the status surface and prevents focus/semantics access to underlying controls. On exit, restore focus only when that target is still mounted, belongs to the permitted current UI and is not hidden by an account gate; otherwise use normal current-screen focus behavior.
- Keep notification/account initialization/in-flight request effects running under existing rules. Unsent editor text/media stays in existing providers/widgets. A success outcome can still clear a submitted editor as it did before; failures/cancellation retain their existing semantics. No status transition invalidates sessions, retries uploads, resubmits posts/drafts or bypasses eligibility/moderation.

### Components and localized strings

`MaintenanceScreen` uses existing themes/icons and `ChunkyButton` for Try again, plain `Text` title/message, a scrollable bounded-width body and optional local-time estimate. There is no dismiss/link action. Retry requests status only; clearing reveals the existing screen's own retry actions if initialization/API work failed.

`AnnouncementBanner` uses a general plain-text notice, dismiss action and the same informational estimate treatment when supplied. It never opens a link or blocks ordinary controls. Maintain screen-reader labels/focus order and large text reachability.

Add English ARB entries for maintenance retry, announcement dismissal and estimated recovery label; reuse existing compatible localized strings where appropriate. Regenerate localization via `flutter gen-l10n`. Formatter converts the accepted instant with `toLocal()` and `intl`, including the locale-appropriate date/time; ensure formatting is available in loading/error roots independently of `appDependenciesProvider` (local formatting initialization can be owned by a separate UI formatter future with fallback, never the status controller). Do not delay the maintenance title/message/retry if estimate formatting initialization is pending. Estimates never schedule a transition.

### Diagnostics

Controller/repository own consumed status failures: expected offline/timeouts/validation/expiry remain bounded supporting logs, not repeated issues. A genuinely unexpected consumed implementation failure is captured once via the existing bootstrap-provided `ErrorReporter`; keep original typed cause and available caught stack. Failures propagated to Riverpod/framework remain owned there, with no second capture. New `serviceStatusReporterProvider` gets the existing reporter in both bootstrap root scopes, without changing other feature reporter wiring.

Use existing `Logger`, `DiagnosticMessage`/`ReportContext` and guards; no Sentry import in feature code. Classify fetch, validate, presentation transition and operator verification with finite static outcome vocabulary. Never put title/message/raw JSON, revision, credentials, drafts/account/device IDs, raw response/error prose or arbitrary destinations in logs or metric labels. DTO/state safe summaries must survive ProviderLogger's opt-in debug path. Test serialized local and SDK output with positive fields, protected canaries, expected polling repetitions and reporter failure. Worker/operator logs follow the same exclusions even though they do not use Flutter sinks.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|
| Initial fetch pending, dependencies ready | Continue ordinary app; no fixed startup wait | FR-004, NFR-001 | AC-006 | UT-003 |
| Maintenance with failed/pending account/dependencies | Show status using independent DI; remove splash on visible cover; keep initialization state intact | FR-001, FR-005 | AC-001 | AT-001 |
| Valid announcement while dismissal read pending | Briefly hold only notice rendering; normal UI usable; maintenance never waits | FR-006, NFR-001 | AC-004, AC-009 | AT-003, AT-007, IT-002 |
| Invalid/oversized/proxy/non-200/redirect response | Reject before acceptance; preserve valid status/age; existing errors when no trusted maintenance | FR-003, FR-007, RULE-003 | AC-005, AC-010, AC-019 | UT-002, UT-004, AT-005, IT-001 |
| Hung read or duplicate triggers | Overall three-second abort, shared pending future, safe epoch invalidation | FR-004, NFR-001 | AC-006 | UT-003 |
| Maintenance ages out or app resumes after expiry | Remove cover by five-minute rule without recovery claim; ordinary failures remain | FR-008, RULE-001 | AC-011 | UT-004, AT-006 |
| Unchanged valid maintenance/failed renewal | Success renews age, failure does not; no periodic issue spam | FR-008, NFR-004 | AC-011, AC-018 | UT-004, IT-006 |
| Clear/announcement replaces maintenance | Immediately reveal appropriate retained current-account UI/gates | FR-008, FR-009, RULE-004 | AC-007, AC-012 | AT-006, REG-002 |
| Passing estimated time | Render informative local estimate; remain explicit mode while fresh | FR-002, NFR-003, RULE-001 | AC-003, AC-017 | AT-002, UT-006 |
| Account/OAuth/operation completion under cover | Existing boundaries/outcomes; no sign-out or replay attributable to status | FR-009, RULE-004 | AC-008, AC-012 | AT-004, REG-001–REG-003 |
| New process offline | Fetch anew; no maintenance restoration/durable-editor promise; retain persisted notice dismissal | FR-006, FR-007, FR-008 | AC-009, AC-010, AC-011 | AT-007, IT-002 |
| Cancelled/invalid command | Nonzero outcome, zero object mutation | FR-012 | AC-015 | IT-003 |
| Upload attempted, public verification fails or another operator supersedes it | Unverified, may already be live, no automatic rollback or retry write | FR-012, NFR-002 | AC-015, AC-016 | IT-004 |
| Website/Worker code rollback | Separate bucket object untouched; status code still uses same schema/binding | FR-001, FR-012 | AC-015 | IT-005, REG-004, IT-007 |

## 9. Test Implementation Plan

Every existing case family has a concrete target. Expand subcases within stable IDs; do not renumber requirements or reserve withdrawn IDs for new behavior. First writes are executable tests during `implement-tdd`, not during this plan.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `app/test/service_status/service_status_document_test.dart` | Minimal literal schema 1 maintenance, valid token revision, no publication time; shared corpus | Missing DTO/decoder or unsupported valid contract |
| 2 | UT-002 | Same Dart suite and `scripts/test_service_status_publish.py` validator group | Literal 16,384/16,385-byte bodies, 160/161 title and 2,400/2,401 message scalar boundaries, impossible zoned timestamps, streaming oversize | Unsafe/mismatched acceptance or unbounded decode |
| 3 | UT-003 | `app/test/service_status/service_status_controller_test.dart`; startup wiring in `app_test.dart` | Fake clock/repository and delayed dependency init; all modes and lifecycle triggers | No polling/coalescing/deadline or blocked startup |
| 4 | UT-004 | Same controller suite | Exact freshness boundaries/renewal, suspension/resume, wall rollback and cancelled late result | Indefinite cover or superseded acceptance |
| 5 | IT-001 | `app/test/service_status/service_status_http_test.dart`; mobile socket smoke in `critical_journeys_test.dart` | Separate status/AppView servers, captured synthetic auth/device values, redirects; web adapter mocked fetch + hosted checks | API headers reused, redirects followed or endpoint depends on account |
| 6 | AT-001 | `app/test/service_status/service_status_app_test.dart` | Signed-in/out and failed/pending dependencies/account; splash callback recorder | No custom cover before initialization or splash remains |
| 7 | AT-002 | `app/test/service_status/service_status_widgets_test.dart` | Plain/malicious prose, estimates, retry recorder and internal back event | Wrong content/actions, hidden screen interaction or missing retry |
| 8 | AT-005 | Same app suite and existing error fixtures | Status unavailable, API 503, HTML/non-200, announcement/health probe | Global maintenance inferred or usable UI blocked |
| 9 | UT-005 | `app/test/service_status/announcement_dismissal_test.dart` | Literal revisions A/B, delayed dismissal completion with newer status | Same revision resurfaces or new revision suppressed |
| 10 | IT-002 | `app/test/service_status/announcement_persistence_test.dart`; `critical_journeys_test.dart` | Mocked adapter for fast tests; real preferences write/reload, fresh instances for device evidence | Dismissal lost, plugin-only cache masks missing storage, or maintenance reused |
| 11 | AT-003 | `service_status_app_test.dart` | Current announcement, usable navigation/operations, normal/maintenance replacement | Notice blocks app or simultaneous modes visible |
| 12 | AT-007 | `app/test/service_status/service_status_restart_test.dart` | Shared only preference storage across recreated feature/app instances; offline restart | Same-revision flash, missing new fetch or mandatory restored cache |
| 13 | AT-004 | `app/test/service_status/service_status_editor_test.dart` | Existing post/project editor fixtures with text/media and delayed operation | Editor unmounted, navigation lost or writes replayed |
| 14 | AT-006 | `app/test/service_status/service_status_recovery_test.dart` | Inject normal/announcement/expiry and current-account gates | Relaunch required, wrong-account UI or policy bypass |
| 15 | REG-001 | Existing sign-out interceptor/error mapper tests | Genuine 401 control, 503/offline/status outcomes | Session invalidated by status failure |
| 16 | REG-002 | Existing account-switch/OAuth handoff/restricted routing/gate suites | Delayed old-account completion and handoff beneath cover | Current-account fencing or eligibility bypassed |
| 17 | REG-003 | Existing composer/video/draft suites and mobile critical journey | Original operation counts/outcomes with maintenance transition | Existing completion cleanup changes or extra submission |
| 18 | UT-006 | `app/test/service_status/service_status_estimate_test.dart` | Explicit offset instants/DST zone test context, absent/past estimate | Incorrect local display or automatic mode transition |
| 19 | UT-007 | Document/widgets suites | Markup, URL prose and unsupported control fields | Code/link/control behavior enabled |
| 20 | AT-009 | `app/test/service_status/service_status_accessibility_test.dart` | Existing English locale/device fallback, light/dark, supported small/web sizes and maximum text scaling | Overflow, inaccessible retry/dismissal, exposed covered semantics |
| 21 | IT-006 | `app/test/observability/service_status_diagnostics_test.dart`; Python/Node output checks | Serialized transport, debug ProviderLogger, original cause/stack and prose/secret canaries | Leak, missing positive evidence, duplicate issues or reporter affects behavior |
| 22 | AT-008 | `scripts/test_service_status_publish.py` command workflow | Injectable R2/public verifier/confirmation; Render unavailable; no Git/site release | Independent validate/revise/clear workflow unavailable |
| 23 | IT-003 | Same Python suite | Invalid/declined/EOF inputs, target discovery, fixed temporary payload; reused draft revision | Mutation before confirmation; unchanged revision used for new announcement; credentials exposed |
| 24 | IT-004 | Same Python suite | Exact/stale/unreachable/wrong-type public responses and concurrent newer revision | False verified result, unlimited wait, write replay or rollback |
| 25 | IT-005 | Python command/website suites and Node Worker tests | Separate fake stores/resources and two website code versions | Status overwritten by unrelated release or Worker deploy mutates object |
| 26 | REG-004 | Existing `test_web_deploy.py`, `test_cloudflare_pages_build.py` | Status resource separation configuration; existing website allowlist/promotion fixtures | Status bundled or website release behavior broken |
| 27 | IT-007 | `service-status/test/worker.test.js`, `hosted-status.spec.cjs`, `scripts/service-status-check` | Local fake R2 for handler assertions, then separately authorized preview R2/Worker and real browser/native reads | Wrong CORS/cache/type or stale next refresh; mock pass cannot satisfy hosted evidence |
| 28 | MAN-001 | Manual recorded evidence | Supported Android/iOS/web screen readers, long text/large scaling, underlying editor | Reading/focus/accessibility fails despite widget semantics |

### Transport, clock and automation detail

Native adapter uses new Dio with no interceptors, `followRedirects: false`, no auth/cookies/device headers, `ResponseType.stream`, capped byte reads and a CancelToken. Do not reuse `anonymousDioProvider` or `baseDioOptions()`. Status requests send only needed public headers such as `Accept: application/json`.

Web uses a conditional `package:web` Fetch adapter with `credentials: omit`, `redirect: error`, `cache: no-store`, AbortController and capped response-stream reads. This avoids relying on Dio's browser/XHR adapter to enforce a redirect policy it cannot fully control. No new package is needed: `web` is already a dependency. Do not use default ambient cookies. Test failures/cancellation and CORS with actual browser Fetch, not native-only mocks.

Controller fakes provide cancellable pending requests, fake age clock/timers and literal results; widget tests use Riverpod overrides plus existing app/editor fakes. Do not run real five-minute sleeps. Test known DTO/state `toString()` output through ProviderLogger with debug enabled, not just explicitly emitted feature logs.

Node Worker tests import the small handler with a fake `STATUS_DOCUMENTS` binding and assert fixed routing, no mutation calls, no-store/CORS headers and bounded errors. Python tests use temporary files and mock subprocesses/public sockets; assert exact immutable final payload, no secret output, no unconfirmed calls and fresh revision across repeated publications of the same draft. Add FR-006/AC-009 traceability to that extension of IT-003 in implementation evidence (DR-001); preserve IT-003's ID.

### Hosted verification rule (DR-002)

`service-status/playwright.config.cjs` is separate from the website consent config and resolves the existing pinned Playwright package. Ordinary CI runs local Node/Python/Flutter tests without Cloudflare credentials. Hosted smoke is opt-in against explicit preview resources, with the operator confirmation preceding the test's planned object replacements.

After publication B is **verified** at the preview public URL, reuse a browser context that read A, and issue the next anonymous Fetch with `cache: no-store`; the equivalent native read uses the public URL. Both must return B in a single successful fetch within the application's three-second deadline. Repeat B → normal. Confirm no-store/JSON/CORS and no credential headers; no cache-busting nonce or fresh browser context can substitute for warmed-browser evidence. Unchanged maintenance reads are accepted only from the uncached binding path; no conditional requests/304 handling in v1.

Keep deployment propagation retries in the publisher's finite verification budget; do not broaden the post-verification delivery assertion with indefinite eventual-success polling. Any zone rule or proxy that overrides no-store and causes stale content fails IT-007 and blocks readiness for rollout. Equivalent production route/binding/cache configuration still needs read-only review and an authorized smoke check; preview evidence alone does not prove production configuration. Landing release/rollback checks use a disposable landing resource, never production website mutations.

## 10. Sequencing And Guardrails

- First TDD step: write UT-001 with a literal minimal maintenance JSON fixture and no publication time; run it red before adding the decoder.
- Resolve contract corpus first, then controller deadline/freshness, anonymous adapters, root/splash presentation, dismissal persistence, state/account regressions, accessibility/diagnostics and operator/host integration. Follow section 9 red-green-refactor order; do not implement the whole feature ahead of its tests.
- Worker/CLI implementation can use mocked/local R2 before any Cloudflare access is available. Infrastructure provision, custom domain and live publication remain an explicitly authorized later operation. Ordinary publish/clear never provisions resources or deploys Worker/website code.
- Preserve current GoRouter/controller/editor identities; guard back/focus at the root presentation boundary and test it early. Retain existing current-account/initialization/moderation/auth semantics.
- No status dependency on device/account/API readiness. No credentials, arbitrary redirects/links, remote code or private data in status transport, public document or diagnostics.
- Keep one polling/expiry suite; cross-link evidence rather than duplicating timer tests. Shared contract fixtures are independent literals, not expectations generated from the implementation.
- Update local recipes and PR path classification for `service-status/**`, `scripts/service-status*`, `scripts/service_status.py`, `scripts/test_service_status_publish.py` and relevant pinned-tool/config changes. A status check job runs Python/Node deterministic tests. Reuse pinned checkout/Node setup conventions; no secrets or automatic Cloudflare writes in CI. Broad existing tests run once after focused cases pass.
- Check commands during authorized implementation: `just app-test test/service_status/`, focused auth/router/composer/observability suites, `just app-test`, `just app-analyze`, `just app-test-integration <device-id>`, Python script tests, Node Worker tests, and `just web-check` for the website regressions. Go tests are unnecessary unless scope later changes AppView.
- Generate affected Riverpod code with existing `dart run build_runner build --delete-conflicting-outputs` from `app/` and localization with `flutter gen-l10n`; inspect generated changes for unrelated drift. Do not add dart_mappable generation for this small manually validated contract.
- Out of scope: backend admission/draining/worker control, Render/lexicon/PDS changes, feature flags, schedules, multiple notices/history, external links, translated custom prose, persistent maintenance cache, new offline editor durability and automatic write replay.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking for local TDD; blocking for hosted verification | Account permissions, R2 availability, bucket and Worker/domain resources unverified (ASM-008/GAP-002) | Cannot claim live JSON availability or execute IT-007 | Use fake/local adapters first. During separately authorized setup, verify target account/resource names, credential scope and DNS; create preview before production. Do not embed resource IDs/secrets in source. |
| CPQ-002 | Resolved design | Revision ownership and finite contract bounds (GAP-001/DR-001) | Changed notices could stay dismissed; inconsistent validator behavior | Section 5 pins schema/fields/bounds; fresh UUID every publish/clear; same editable schema; shared literal corpus and expanded IT-003. No publication time or secondary revision. |
| CPQ-003 | Resolved design; deployment evidence pending | Edge/browser cache behavior and measurable hosted observation (DR-002) | Superseded maintenance could be treated as newly fresh | Direct R2 binding, no Cache API, no-store responses/web Fetch, no conditional requests; next warmed-client read after verified publication must match within three seconds. Verify zone overrides before rollout. |
| CPQ-004 | Resolved design; device evidence pending | Preferences cache masquerading as persisted dismissal (DR-003) | Restart test can falsely pass | Await write, dispose feature state, reload actual storage and recreate instances in IT-002; no durable maintenance/editor additions. |
| CPQ-005 | Non-blocking for coding plan; early implementation verification | Stable router back dispatcher and native splash/focus integration | Cover might be hidden by splash or bypassed with back/focus | AT-001/AT-002 test root wiring before expanding UI; preserve router delegate/identity and consume internal back while covered. If a framework constraint requires a different adapter, keep behavior and record the narrow plan adjustment. |
| CPQ-006 | Non-blocking for automated TDD; blocking for complete accessibility evidence | Real assistive technology and external OAuth/media limitations (GAP-004/GAP-005) | Cannot infer device usability/external flow coverage from widgets | Execute MAN-001 and supported mobile tests; retain existing limited OAuth/media harness boundaries; no synthetic production failures or new external-flow automation scope. |
| CPQ-007 | Non-blocking operational risk | Anonymous uncached reads incur Worker/R2 traffic, including 60-second foreground polling | Hosting capacity/plan limits may affect delivery | Review existing Cloudflare plan/usage during authorized setup; keep document small and no backend dependency. No CDN cache that violates freshness, new monitoring service or unrequested infrastructure spend. |
| CPQ-008 | Non-blocking operational risk | Concurrent operator publications are last-write-wins | A confirmed change may be replaced before verification | Match intended UUID/content publicly; report unverified if superseded. Initial runbook assumes maintainer-coordinated updates; no lock/history service added. |

No unresolved product or architectural question blocks a useful implementation plan. External resource/access and manual evidence dependencies remain visible. Risk stays Medium; implementation initiation requires the user's next-stage selection, and production mutations require separate explicit scope.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-08-service-status-maintenance/04-coding-plan.md`.
- TDD execution plan: `05-implementation-plan.md` in the same folder, created by `implement-tdd` after the user selects implementation.
- Start with test: **UT-001** at `app/test/service_status/service_status_document_test.dart`, minimal schema 1 maintenance with no publication time, targeting `ServiceStatusDocument` decoder.
- Focused command: `just app-test test/service_status/service_status_document_test.dart` once the first test is written.
- Proposed additional local commands: `python3 -m unittest discover -s scripts -p 'test_service_status_publish.py'`, `node --test service-status/test/worker.test.js`, then existing Flutter/website recipes. Hosted command uses only explicitly configured preview resources; credentials and mutations are not required for the local sequence.
- Review notes DR-001–DR-003 have explicit design resolutions in sections 5, 6 and 9. Preserve their IDs and record test/evidence links rather than rewriting the earlier documents during implementation without authorization.
- Keep pending hosted configuration, device/browser and manual checks in validation evidence. Do not mark them complete from mocks or this design document.
- No stage commit is enabled. This plan remains uncommitted; no implementation or Cloudflare/Render mutation has occurred.
