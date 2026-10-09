# Requirements: Service Status and Maintenance Messaging

## 1. Initial Request

Allow CraftSky to show a maintenance screen and a custom message when downtime occurs because of planned maintenance or unexpected errors. Explain the available approaches and common platform patterns, then define requirements for the selected approach.

The user confirmed a hosted JSON document containing the necessary information and selected Cloudflare because it already hosts the CraftSky landing page. This artifact defines requirements only; it does not authorize a production deployment.

## 2. Current Codebase Findings

- Relevant files: `app/lib/app.dart`, `app/lib/router/router.dart`, `app/lib/auth/widgets/active_account_initialization_gate.dart`, `app/lib/initialization_error_screen.dart`, `app/lib/shared/errors/app_error_mapper.dart`, `app/lib/shared/api/api_exception.dart`, `app/lib/shared/api/providers/sign_out_on_401_interceptor.dart`, `appview/internal/middleware/auth.go`, `web/wrangler.json`, `web/README.md`, `scripts/cloudflare_pages_build.py`, and `justfile`.
- Existing patterns: Flutter uses Riverpod, GoRouter, localized UI and shared error mapping. Account initialization gates signed-in UI. AppView errors follow the `/v1/` JSON error envelope `{error, message, requestId}`.
- Current behavior: API server errors map to `serviceUnavailable`, while connection failures/timeouts map to `networkUnavailable`. The sign-out interceptor invalidates the captured session lease only on HTTP 401. Authentication dependency failures return HTTP 503 `authentication_unavailable` with `Retry-After: 5`. Startup has loading/error screens and retry actions, but no reviewed service-wide status controller or operator-authored outage message was found in the inspected paths.
- Cloudflare: the landing site uses a Worker with static assets from `web/dist`, not a backend-dependent Render page. The build explicitly allowlists public files. The selected status document will have independent publication rather than being added to the landing-site artifact. `web/README.md` documents version previews, verification, controlled production promotion and rollback. Production website deployment currently requires a clean checkout matching `origin/main`.
- Constraints: ordinary app reads and writes remain AppView-mediated; OAuth credentials stay server-side. Status communication must not require AppView, Postgres, authentication, or successful account initialization. The API architecture spec and `docs/development/logging-error-reporting.md` apply to API and diagnostic changes. Production change controls remain in force.
- Test/build commands discovered: `just app-test`, `just app-analyze`, `just test` (host Go tests against the running compose stack), `just web-build`, and `just web-check`. None were run during this requirements-only stage.

## 3. Clarifying Questions And Decisions

The user confirmed the following during the grilling interview and approved the complete scope on 2026-10-09. Interview question numbers are retained for traceability.

| Question | Confirmed decision | Implication |
|---|---|---|
| Q1 | Whole-app maintenance plus a non-blocking general announcement. | No individual feature notices or restrictions. |
| Q2 | Communication and Flutter screen only. | No AppView enforcement, backend pause or worker control in this feature. Existing maintenance/deployment procedures control the backend. |
| Q3 | Manual activation and clearing. | No scheduled activation; estimated recovery is informational only. |
| Q4 | Independent local JSON publishing command. | No website release or Git commit required for each message change; website deployment/rollback must not alter incident state. |
| Q5 | English custom messages. | Localize built-in screen text and controls; defer translated custom messages. |
| Q6 | Unknown failures stay on affected screens; startup can show a connection error. | Only explicit JSON maintenance triggers the custom whole-app screen. |
| Q7 | Cached maintenance is authoritative for at most five minutes after the last successful fetch. | A successful validated fetch of unchanged maintenance renews freshness. After expiry, allow ordinary access attempts without claiming recovery. |
| Q8 | Check at launch, foreground resume, manual retry, and every 60 seconds while foregrounded, including normal use; three-second timeout. | Startup proceeds concurrently; the timeout is not a mandatory launch delay. |
| Q9 | Immediately cover the current UI with maintenance, preserving the editor underneath. | Existing in-flight requests keep their existing rules; no automatic resubmission. |
| Q10 | Dismiss announcements until revised/replaced, across restarts. | A revised announcement can resurface; maintenance cannot be dismissed. |
| Q11 | Optional estimated recovery time; no external information link initially. | Clearly label estimates; defer links and a public human-readable page. |
| Q12 | Editable JSON with validate, publish and clear commands. | Publish shows the intended change, asks for confirmation, then verifies the public response. Clear publishes normal without manual JSON editing. |
| Q13 | `https://status.craftsky.social/app.json`. | Dedicated Cloudflare address with publication independent of the landing page; JSON only initially. |
| Q14 | One current status: `normal`, `announcement`, or `maintenance`. | Each publication replaces the previous status; no concurrent notices or incident history. |

On 2026-10-09, the user also approved all three simplification-review proposals: allow the maintenance status cache to remain in memory without cross-process persistence; remove mandatory publication time and use one revision for announcement dismissal; and reference the timeout/freshness requirements rather than repeat their numerical rules. This explicitly authorizes losing the cached custom maintenance message after an offline process restart. Persistent announcement dismissal remains required.

The initial draft included broader enforcement, feature scope and scheduling. These are superseded by the decisions above. FR-010, FR-011, AC-013 and AC-014 are withdrawn; their IDs are reserved and will not be reused.

## 4. Candidate Approaches

### Option A: Independent Cloudflare-hosted JSON — selected

Summary: Flutter reads one public status at the dedicated Cloudflare address. A local command publishes it independently of website releases.

Pros: Communicates during AppView outages; fits the selected host; requires no configuration SDK or app release for message changes.

Cons: Needs validation, bounded freshness, polling and a publishing workflow.

Risks: Stale cached instructions and accidental status replacement; mitigate through freshness and independent publication.

### Option B: Firebase Remote Config

Summary: Manage mode/message through remote app parameters. Offers console editing but adds dependency and fetch/activation semantics. Not selected.

### Option C: AppView-only status or inferred maintenance

Summary: Use API responses to supply or infer status. Cannot deliver a new custom message when AppView is unreachable, and failed requests do not establish global maintenance. Existing error handling remains the fallback, without inferred custom maintenance.

### Option D: Hosted incident-management service

Summary: Provides history, subscriptions and richer communication, with another service/workflow. Deferred.

## 5. Recommended Direction

Serve one versioned public JSON document at `https://status.craftsky.social/app.json`, independently of landing-page deployment and rollback. Modes are `normal`, `announcement` and `maintenance`, manually published and cleared.

Flutter renders a non-blocking announcement or an immediate whole-app maintenance screen. It preserves account/navigation/editor state, checks status at the agreed intervals, and stops trusting cached maintenance five minutes after the last successful fetch. Existing localized failure handling covers unknown outages.

Provide local validate, publish and clear commands with confirmation and public-response verification for live changes. This feature communicates status and controls client presentation; it does not enforce backend maintenance.

## 6. Problem / Opportunity

Generic startup and request errors give users little indication of what is happening. An independently delivered message explains confirmed downtime while preserving sessions and unsent work. A general announcement communicates information without interrupting normal use.

## 7. Goals

- G-001: Explain confirmed whole-app downtime with an operator-authored public message.
- G-002: Keep communication available when AppView or its database is unavailable.
- G-003: Distinguish explicit maintenance, non-blocking announcements and unknown connection failures.
- G-004: Recover without unnecessary sign-in, navigation loss or automatic write replay.
- G-005: Publish, revise and clear status independently of website/app/backend releases.

## 8. Non-Goals

- NG-001: Implement code, tests, infrastructure or deployments during this requirements update.
- NG-002: Add incident history, subscriptions, push/email announcements, an operator dashboard or a human-readable status page.
- NG-003: Add feature-specific restrictions, a generic feature-flag system, forced updates or arbitrary remote UI.
- NG-004: Add offline capability, device-persisted drafts or new durability guarantees for in-memory work.
- NG-005: Infer custom whole-app maintenance from failures or automatically publish incidents from monitoring.
- NG-006: Enforce backend restrictions, drain requests/deployments, pause workers or perform database/PDS maintenance. Existing operational procedures remain responsible for backend control.
- NG-007: Change lexicons, PDS access, OAuth credentials, account deletion, moderation or account eligibility.
- NG-008: Schedule activation/clearing, support simultaneous notices, translate custom messages or include external information links initially.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| App user | Signed-in or signed-out user on supported Flutter platforms | Clear explanation, safe retry, retained state and recovery |
| Maintainer/operator | Authorized status publisher | Validate, publish and clear messages while Render is unavailable |
| Flutter client | Anonymous status consumer | Validated public data independent of account initialization |
| Cloudflare status host | Serves the current document | Publication independent of landing-page releases and backend availability |

## 10. Current Behavior

Startup waits for dependency/auth/account readiness and displays existing loading or error UI on failure. Individual API failures flow through shared error mapping. There is no custom Cloudflare-delivered maintenance announcement in the inspected app paths. The website publication process packages only explicitly selected public assets and deploys a complete verified site version.

## 11. Desired Behavior

One fresh status determines normal UI, a dismissible announcement or a non-dismissible whole-app maintenance screen. English custom content uses native branded UI with localized controls and fallback messages. An optional recovery time is clearly an estimate, not a scheduled transition.

At cold start, status retrieval runs concurrently with initialization. During a session, maintenance immediately covers the current UI while retaining editor/navigation state underneath. Existing in-flight work follows existing rules. Clearing maintenance restores appropriate UI through existing account/policy gates without automatically retrying writes.

The last validated status may be retained in memory; it need not survive process restarts. Each process launch fetches anew and uses existing error handling if no valid status is available. Announcement dismissal separately persists by revision. A valid successful maintenance fetch renews a five-minute trust period. If status cannot be validated/fetched and that period expires, the client removes custom blocking and allows normal access attempts. Expiry does not claim that service recovered. Failed requests use localized errors on the affected screen; startup failure can use a full-screen connection error. API failures alone never create the custom maintenance screen.

## 12. Requirements

Sources: **Prompt** = initial request; **Q1–Q14** = confirmed interview decisions in section 3; **Codebase** = section 2. All active requirements below are Must requirements. Withdrawn IDs FR-010 and FR-011 are reserved.

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Users shall receive a custom explanation of explicitly published whole-app maintenance and non-blocking announcements. | Explain service status clearly. | Prompt, Q1 | AC-001, AC-003, AC-004 |
| BR-002 | Business | Must | Maintainers shall publish, revise and clear status without an app release, website release, Git commit for each message, or functioning Render AppView/database. | Communicate during outages. | Q4, Q12 | AC-002, AC-015 |
| FR-001 | Functional | Must | Cloudflare shall serve the current public JSON at `https://status.craftsky.social/app.json` without account, AppView or database dependencies and independently of landing-page releases/rollbacks. | Independent delivery and ownership. | Q13 | AC-001, AC-002, AC-015 |
| FR-002 | Functional | Must | The versioned camelCase JSON contract shall represent one current mode (`normal`, `announcement`, `maintenance`), a revision, English plain-text title/message for announcement/maintenance, and optional estimated recovery time. Each publication replaces the previous status; announcement dismissal applies to its revision. Publication time is not required. There shall be no scheduled activation, feature scope or external information link. | Bound the first-version contract. | Q3, Q5, Q11, Q14 | AC-003, AC-004, AC-005 |
| FR-003 | Functional | Must | Publication and consumers shall reject malformed/oversized data, unsupported schema/mode, missing required text or an invalid estimated recovery timestamp when supplied. Additive unknown optional fields shall be tolerated within a supported schema without enabling unsupported behavior. Invalid fetches shall not overwrite valid cache or renew its trust period. | Avoid unsafe interpretation and accidental blocking. | Codebase, Q7 | AC-005 |
| FR-004 | Functional | Must | Flutter shall fetch status at launch, foreground resume and manual retry, and every 60 seconds while foregrounded in all modes. Each fetch shall time out within three seconds. Initialization shall proceed concurrently; duplicate triggers shall coalesce rather than start overlapping requests. | Detect changes during ordinary use without mandatory startup delay. | Q8 | AC-006, AC-007 |
| FR-005 | Functional | Must | Fresh maintenance mode shall immediately show a branded, non-dismissible whole-app screen with custom title/message and Try again, regardless of sign-in/account initialization. Optional recovery time shall be labelled as an estimate. | Cover cold start and active sessions. | Q1, Q9, Q11 | AC-001, AC-003, AC-008 |
| FR-006 | Functional | Must | Announcement mode shall show a non-blocking dismissible general notice. Dismissal shall persist across restarts for the same revision and cease to apply when that announcement is revised/replaced. Announcement dismissal shall never dismiss maintenance. | Inform without interrupting use. | Q1, Q10, Q14 | AC-004, AC-009 |
| FR-007 | Functional | Must | Without fresh JSON maintenance, failed requests shall retain localized errors/retry on affected screens; startup failure may show a full-screen connection/service error. Failed API/status requests or normal JSON shall not trigger the custom maintenance screen. | Distinguish unknown failures from explicit status. | Q6 | AC-010 |
| FR-008 | Functional | Must | The client may retain the last validated status in memory; cross-process persistence is not required. Each process launch shall fetch status anew and use existing error handling when no valid status is available. Cached maintenance shall stop blocking no later than five minutes after its last successful validated fetch. A validated successful fetch of unchanged maintenance renews that period. New normal/announcement status clears maintenance immediately upon acceptance. Expired cache permits ordinary access attempts without asserting recovery; failed/invalid fetches do not refresh freshness. | Prevent indefinite cached lockout. | Q7, Q14 | AC-007, AC-011 |
| FR-009 | Functional | Must | Maintenance shall cover rather than discard existing editor/navigation state, preserve the session/current account and unsent in-memory work, and allow in-flight requests to follow existing rules. Clearing/expiry shall restore appropriate UI through required initialization/policy gates without automatic write resubmission. | Preserve work and account isolation. | Q9, Q6, Codebase | AC-008, AC-012 |
| FR-012 | Functional | Must | Provide local validate, publish and clear commands. Publish shall validate the editable JSON, display the intended live change, obtain confirmation and verify the public response. Clear shall publish normal without manual JSON editing, with confirmation and verification. Invalid/cancelled changes shall not alter live state; unverified publication shall not be reported as successful. | Verifiable operator workflow. | Q4, Q12 | AC-002, AC-015 |
| NFR-001 | Non-functional | Must | Status retrieval shall not cause indefinite loading or prevent otherwise available operations when status is unknown. Apply the timeout and freshness rules in FR-004 and FR-008. Document/text bounds shall be finite and specified during implementation design. | Status infrastructure must not become an unbounded startup dependency. | Q7, Q8 | AC-005, AC-006, AC-010, AC-011 |
| NFR-002 | Non-functional | Must | Cloudflare/browser caching shall support the 60-second foreground refresh behavior without serving a superseded response as newly fresh indefinitely. JSON content type and web CORS shall permit anonymous supported clients to read the document. Public verification shall check the intended live document. | Make refresh/freshness meaningful. | Q8, Q12 | AC-016 |
| NFR-003 | Non-functional | Must | Use CraftSky themes, accessible layouts, screen-reader support and supported text scaling/form factors. Built-in controls/fallbacks shall be localized; custom text shall be English initially. Recovery estimates shall use the user's local time and never automatically change mode. | Clear accessible communication. | Q5, Q11 | AC-017 |
| NFR-004 | Non-functional | Must | Follow the diagnostic guide: bounded classification/transition context, original available typed causes/stacks, one issue owner, no serialized remote message/document or protected data, and no repeated actionable issues from expected maintenance/offline polling. | Useful privacy-bounded diagnostics. | Codebase | AC-018 |
| RULE-001 | Business rule | Must | Only validated JSON maintenance that remains fresh under FR-008 shall authorize the custom whole-app maintenance screen. Estimates, announcements, health probes and API failures shall not activate it. | Single understandable presentation authority. | Q6, Q7, Q14 | AC-003, AC-010, AC-011 |
| RULE-002 | Business rule | Must | Status retrieval shall send no CraftSky bearer token, device identifier, DID, OAuth credential or account-specific data. The document shall contain public operational information only; publication credentials stay outside the app/repository. | Separate public retrieval from authenticated API traffic. | Codebase | AC-019 |
| RULE-003 | Business rule | Must | Remote content shall render as bounded plain text, never executable HTML/code, API destinations or external links. Redirect handling shall not carry authenticated headers or enable arbitrary remote destinations. | Bound the remote control surface. | Q11, Codebase | AC-005, AC-019 |
| RULE-004 | Business rule | Must | Status/unavailability shall not invalidate sessions as if it were 401 or bypass existing account isolation/auth/eligibility/moderation checks. This feature shall not change AppView admission, worker behavior or existing request outcomes. | Communication-only scope with existing safeguards. | Q2, Q9, Codebase | AC-008, AC-012 |

## 13. Acceptance Criteria

AC-013 and AC-014 (backend enforcement/coordination) are withdrawn and their IDs reserved.

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-005 | Given AppView/database are unavailable, when a signed-in or signed-out client launches and obtains maintenance JSON from the selected Cloudflare address, then it shows the custom screen without successful account initialization or backend access. |
| AC-002 | BR-002, FR-001, FR-012 | Given Render is unavailable, when an authorized maintainer validates, publishes/revises and clears status, then verified public responses reflect the changes without app/site/backend releases or a Git commit for each message. Clear requires no manual JSON edit. |
| AC-003 | BR-001, FR-002, FR-005, RULE-001 | Given fresh maintenance, when rendered, then the English title/message, localized Try again and optional local-time estimated recovery appear, with no dismiss/link action. Passing the estimate never changes mode. |
| AC-004 | BR-001, FR-002, FR-006 | Given announcement mode, when accepted, then a general notice appears without blocking normal operations. Publishing another mode replaces it; there is no simultaneous notice or scheduled activation. |
| AC-005 | FR-002, FR-003, NFR-001, RULE-003 | Given malformed JSON/HTML, oversized data, unsupported schema/mode, missing required text or invalid optional estimated recovery timestamp, when validated, then publication is rejected or the client ignores it without replacing valid cache/renewing freshness. Supported-schema optional unknown fields are ignored without activating links or other behavior. |
| AC-006 | FR-004, NFR-001 | Given launch, resume, retry or foreground polling in any mode, when triggered, then one coalesced fetch runs with a three-second timeout and periodic checks run every 60 seconds. Startup proceeds concurrently and does not wait an unconditional three seconds; background polling stops. |
| AC-007 | FR-004, FR-008 | Given foreground maintenance UI, when normal or announcement is published and fetched, then custom blocking clears without relaunch. Late overlapping results cannot reinstate superseded state. |
| AC-008 | FR-005, FR-009, RULE-004 | Given unsent editor text/media, navigation state and a request in flight, when maintenance arrives, then it covers the UI immediately while retaining existing state. The request follows existing outcome rules; neither entry nor exit changes backend admission or auto-submits/replays work. |
| AC-009 | FR-006 | Given a dismissed announcement, when the app restarts and fetches the same revision, then it stays dismissed. A revised/replaced announcement with a new revision resurfaces; entering maintenance is unaffected by dismissal. |
| AC-010 | FR-007, NFR-001, RULE-001 | Given no fresh maintenance and an unreachable status endpoint, offline device or API failure, when errors appear, then affected screens retain localized errors/retry and startup may show a connection/service error. No custom maintenance is inferred; otherwise available operations remain usable. |
| AC-011 | FR-008, NFR-001, RULE-001 | Given cached maintenance and no subsequent successful fetch, when five minutes have elapsed since its last validated fetch, then custom blocking ends and ordinary access attempts are allowed. API failures use ordinary error UI without claiming recovery. A successful unchanged maintenance fetch before expiry resets the five-minute period; invalid/failed responses do not. After a process restart, a client with no valid fetched status uses existing error handling rather than requiring restoration of the previous maintenance cache. |
| AC-012 | FR-009, RULE-004 | Given a retained session/current account, when maintenance clears/expires, then existing editor/navigation is restored within account boundaries and required initialization/policy gates still apply. No outage-triggered sign-out, cross-account restoration, policy bypass or automatic write replay occurs. |
| AC-015 | BR-002, FR-001, FR-012 | Given a publish/clear command, when executed, then it shows the intended live change, obtains confirmation and verifies the public document. Invalid input/cancellation leaves live state unchanged; verification failure reports an unverified result rather than success. An unrelated landing-site deployment/rollback leaves status unchanged. |
| AC-016 | NFR-002 | Given supported native/web clients, when foreground refresh checks follow publication, then anonymous requests receive JSON with required CORS and can observe the new document; edge/browser cache policy does not indefinitely renew superseded maintenance. Publication verification reads the intended live response. |
| AC-017 | NFR-003 | Given supported themes, locales, form factors, screen readers and text scaling, when status UI renders, then actions/text remain accessible without overflow, built-in UI is localized, custom text remains English, and any recovery time is a labelled local-time estimate without automatic mode transition. |
| AC-018 | NFR-004 | Given expected maintenance/offline polling and an unexpected consumed failure, when serialized diagnostics are inspected, then approved bounded positive fields and available original cause/stack are retained, remote prose/protected canaries are absent, and expected polling does not repeatedly create issues. |
| AC-019 | RULE-002, RULE-003 | Given a signed-in account and malicious remote text/unknown link fields or redirects, when status is retrieved/rendered, then no account/auth/device data leaves on status requests, content stays plain text and no remote code, external link or credential-bearing redirect is enabled. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Status unavailable on first launch | Initialization proceeds concurrently; existing startup failure UI as appropriate, no custom maintenance inference. | FR-004, FR-007 |
| EC-002 | Cached maintenance becomes stale offline | Stop blocking five minutes after last validated fetch; ordinary errors if access fails, no recovery claim. | FR-008 |
| EC-003 | Resume after clear | Fetch status, remove maintenance and restore appropriate UI through any required initialization. | FR-004, FR-009 |
| EC-004 | JSON normal but backend still fails | Existing affected-screen/startup error handling; no custom maintenance from API response. | FR-007 |
| EC-005 | Proxy HTML/502/503 | Defensive parsing and localized errors, no remote HTML rendering or inferred global screen. | FR-003, FR-007 |
| EC-006 | Maintenance during upload/publish | Preserve current state and existing request semantics; never blindly replay. | FR-009 |
| EC-007 | Account switch/OAuth completion races with maintenance | Keep existing account boundaries/protocol behavior; restore only appropriate current-account UI. | FR-009, RULE-004 |
| EC-008 | Late fetch result | Do not reinstate superseded state; coalesce triggers. | FR-004, FR-008 |
| EC-009 | Unknown schema/mode or optional field | Reject unsupported required semantics; tolerate optional fields without activating unsupported features. | FR-003 |
| EC-010 | Landing-site deployment/rollback | Current independently published status remains unchanged. | FR-001, FR-012 |
| EC-011 | Process restart without a valid fetched status | Fetch anew; existing error handling applies if unavailable. Restoring previous maintenance status is not required. Announcement dismissal still persists by revision. | FR-004, FR-006, FR-008 |
| EC-012 | Process killed with unsent work | Existing durability only; no new promise to persist in-memory editor work. | FR-009 |
| EC-013 | Unchanged maintenance fetched successfully | Renew freshness; do not treat failed/invalid fetch as renewal. | FR-003, FR-008 |
| EC-014 | Cancelled publish or failed public verification | Cancellation leaves live state unchanged. Verification failure states that publication may already be live and remains unverified; no false success or automatic rollback. | FR-012 |
| EC-015 | Estimated recovery passes | Continue explicit maintenance until clear or fetch freshness expires; do not auto-clear at the estimate. | FR-002, RULE-001 |

## 15. Data / Persistence Impact

- New fields: current public status contract with one revision and no required publication time; optional in-memory last-valid status/fetch-age tracking; persisted dismissed-announcement revision. Maintenance status persistence across process restarts is not required.
- Changed fields: no existing user record changes.
- Migration required: none expected. No backend enforcement storage is introduced.
- Backwards compatibility: shipped-app compatibility is not required by repository guidance; version the JSON contract for future evolution. No lexicon/PDS changes. Clients that ignore status remain outside the presentation control; this is not a backend safety gate.

## 16. UI / API / CLI Impact

- UI: maintenance cover, non-blocking general announcement, localized existing failure guidance, retry and optional estimate. Preserve underlying editor/navigation and expose the custom screen before account initialization succeeds.
- API: public anonymous Cloudflare JSON contract at the confirmed address. No AppView routes, maintenance error code, admission policy or health-probe changes.
- CLI: local validate/publish/clear workflow with editable JSON, intended-change display, confirmation and public verification. Status is independently published, outside landing-site release/rollback.
- Background jobs: none added. Client polling occurs only while foregrounded; backend workers/in-flight work retain existing rules.

## 17. Security / Privacy / Permissions

- Authentication: dedicated anonymous status retrieval without authenticated API headers, account-specific data or credentials.
- Authorization: production updates require maintainer Cloudflare permissions; publishing tools protect credentials and require confirmation. Preview/dev configuration must not affect production clients.
- Sensitive data: only public English operational information; no private drafts, moderation detail, account identifiers or infrastructure secrets in JSON/diagnostics.
- Abuse cases: malformed/oversized content, HTML/code injection, credential-bearing redirects, unauthorized publication and stale cached blocking. Apply bounded validation, anonymous requests, plain-text rendering and five-minute freshness. No external information link initially.

## 18. Observability

- Events: bounded fetch/validation outcome, presentation transition and recovery outcome; use existing approved diagnostic context.
- Logs: preserve available original typed causes/stacks with existing entry points; no full document, custom prose, private work or credentials.
- Metrics: existing low-cardinality outcomes if appropriate; no message/account/revision metric labels and no new monitoring service required.
- Alerts: expected maintenance/offline polling does not repeatedly create actionable issues. Publishing verifies live delivery; automatic incident publication is deferred.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Stale edge/browser/client status | Delayed detection/clearing or prolonged blocking. | Foreground polling, verified cache policy and five-minute client freshness. |
| RISK-002 | Normal JSON while backend remains unavailable | Attempts still fail after the screen clears. | Ordinary localized errors; never present normal/expiry as proof of recovery. |
| RISK-003 | Maintenance replaces editor or gates initialization indefinitely | Work loss or startup lockout. | Retain underlying UI, concurrent startup and bounded fetch/freshness. |
| RISK-004 | Individual failures inferred as global maintenance | Unnecessary app blocking. | Only explicit fresh JSON maintenance triggers custom screen. |
| RISK-005 | Remote content or credentials mishandled | Disclosure or remote-code/link abuse. | Public-only contract, anonymous retrieval, plain text and protected publishing. |
| RISK-006 | Website rollback replaces incident state | Maintenance changes unexpectedly. | Independent publication and regression verification. |
| RISK-007 | Late responses or incorrectly tracked in-memory age | Superseded maintenance reappears or trust extends indefinitely. | Coalesced requests, supersession checks and FR-008 freshness handling. |
| RISK-008 | Operator assumes screen pauses backend | API callers/workers/in-flight requests continue. | Explicit communication-only scope; backend control remains with existing procedures. |

## 20. Assumptions

Confirmed interview choices are decisions in section 3, not assumptions. Earlier draft assumptions ASM-001–ASM-003 and ASM-005–ASM-007 are resolved/superseded and their IDs reserved.

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-004 | Document/text bounds are finite, small implementation choices. Initial draft examples (16 KiB body, 120-character title, 2,000-character message) are not confirmed product limits. No document expiry/start-time policy is required; freshness is based on successful fetch age. | Select consistent bounds in contract/test design without expanding product scope. |
| ASM-008 | Existing Cloudflare access can provision the confirmed status address and independent publishing resource. This has not been verified live and no provisioning is authorized by this document update. | Resolve DNS/resource/permissions during authorized implementation; retain Cloudflare and independent publication requirements. |

## 21. Open Questions

Blocking product questions: none; the user confirmed the complete scope.

Implementation design choices remain: Cloudflare resource/storage and credentials, exact contract field names/revision representation, finite validation bounds, edge/browser caching, in-memory freshness tracking, and command syntax. These must satisfy the agreed behavior; they are not pending choices about feature restrictions, scheduling, languages, links or backend enforcement.

## 22. Review Status

Status: Approved scope — including simplifications confirmed on 2026-10-09

Risk level: Medium

Review recommended: Yes, for subsequent test/implementation design

Reviewer: Maintainer/user

Date: 2026-10-09

Notes: The user answered Q1–Q14 and confirmed the complete shared-understanding summary. The cache-persistence and contract simplifications were subsequently reopened for review; the user explicitly approved all three proposals before this update, so this approval includes those revised guarantees rather than relying on the earlier approval alone. Approval covers requirements scope, not production provisioning/deployment. FR-010/FR-011 and AC-013/AC-014 are withdrawn without ID reuse. No implementation, tests, infrastructure changes or commits are part of this stage.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-08-service-status-maintenance/01-requirements.md`.
- Next specification: `02-acceptance-tests.md`, using `write-acceptance-tests`.
- Must-cover IDs: BR-001–BR-002; FR-001–FR-009 and FR-012; NFR-001–NFR-004; RULE-001–RULE-004. Active criteria: AC-001–AC-012 and AC-015–AC-019. Do not generate tests for withdrawn backend-enforcement IDs.
- Suggested test levels: contract/validation/freshness unit tests with controlled clocks; Flutter cold-start/lifecycle/editor/navigation/account recovery and dismissal persistence tests; anonymous-request/privacy/diagnostic tests; publication validation/confirmation/clear/public-verification tests; independent website rollback regression checks; manual supported-device/browser and Cloudflare preview checks.
- Test timeout/freshness guarantees once and link the resulting evidence to dependent NFR-001/RULE-001 criteria; do not duplicate suites for repeated references. Include offline process restart without a restored maintenance cache, persistent dismissal by revision, and optional estimated recovery timestamp validation. Do not require publication time or persisted-cache clock reconciliation.
- Required boundary scenarios: all-mode 60-second foreground checks, three-second timeout with concurrent startup, five-minute freshness and renewal, expiry without a recovery claim, no inferred custom maintenance, preserved in-flight outcomes and no automatic write replay.
- Blocking open questions: none at product level. Resolve implementation bounds/contract/cache details in test design. Production checks require separate authorized execution.
