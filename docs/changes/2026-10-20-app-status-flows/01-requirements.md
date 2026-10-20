# Requirements: App Status Flows

## 1. Initial Request

Expand the existing public status document into three independent flows: a mandatory app update, ongoing maintenance with existing behavior, and a version-targeted message with title, description, optional image and optional CTA. Mobile updates open the appropriate store; web updates reload the app.

## 2. Current Codebase Findings

- `app/lib/service_status/models/service_status_document.dart`: schema 1 currently selects one of `normal`, `announcement` or `maintenance`. Documents are limited to 16,384 UTF-8 bytes; notice titles to 160 Unicode scalars and body text to 2,400.
- `app/lib/service_status/providers/service_status_controller.dart`: launch/resume refresh, polling every 60 foreground seconds, a three-second overall request deadline, overlapping-request coalescing and protection against late results. Accepted maintenance expires five minutes after the last valid fetch; failed reads do not renew it. Maintenance is not restored after process restart.
- `app/lib/service_status/widgets/service_status_host.dart`, `announcement_modal.dart` and `service_status_router_config.dart`: retained underlying app, CraftSky-themed dismissible notice, maintenance cover, focus/semantics isolation and root back handling.
- `app/lib/service_status/data/announcement_dismissal_store.dart`: local persisted dismissal revision, independent of account identity.
- `app/lib/app_dependencies.dart` and `app/pubspec.yaml`: installed app version is already available through `package_info_plus`; `url_launcher` is already a dependency. Current manifest version is `1.8.0+13`.
- `app/lib/router/route_locations.dart`: canonical application destinations. Authentication and account-initialization gates already control access to protected screens.
- `service-status/src/index.js`, `scripts/service_status.py`, `scripts/service-status-dev`, `service-status/contract-fixtures.json`: independent Worker/R2 transport, operator validation/publication, local development and shared contract fixtures. Publication currently regenerates the document revision unconditionally.
- `docs/operations/service-status.md`: existing operator workflow, status transport and maintenance semantics. Checked-in documentation does not establish current hosted provisioning or deployment state.
- Relevant checks: `just service-status-test`; `just app-test --no-pub test/service_status test/observability/service_status_diagnostics_test.dart`; the existing Chrome status transport test; `just app-analyze`.
- Repository constraints: iOS, Android and web only; public configuration stays independent of AppView/authentication. No PDS records, lexicon changes or backend enforcement are needed.

## 3. Clarifying Questions And Decisions

| Source | Question | Confirmed answer / implication |
|---|---|---|
| D-001 | Independent settings or one exclusive mode? | Independent settings in one JSON document. Display priority: maintenance, update required, message. |
| D-002 | Message version targeting? | Required minimum and optional maximum; both inclusive. |
| D-003 | Message dismissal? | Remember dismissal per installation until a new message revision is published. |
| D-004 | Platform-specific update thresholds? | Separate minimum required versions for iOS, Android and web. |
| D-005 | Failure to fetch on launch? | Allow access; do not enforce a persisted minimum offline. |
| D-006 | Message CTA destinations? | Either an HTTPS URL or a predefined, supported in-app destination. |
| D-007 | Version comparison? | Semantic versions only; ignore build metadata such as `+13`. |
| D-008 | Retain timing and maintenance behavior? | Retain launch/resume/60-second foreground checks and current maintenance behavior. |
| D-009 | Number of messages? | One active message at a time. |
| D-010 | Direction and next stage? | User confirmed the summarized behavior and requested requirements. |
| D-011 | Failed refresh after accepting an update policy? | Release an existing update gate on failed, invalid or timed-out refresh. Maintenance retains its existing freshness rule. A later valid policy can enforce again. |
| D-012 | Does successful message CTA activation dismiss the message? | Yes, for the captured revision. Failed activation leaves the message visible with retryable feedback. If saving dismissal fails, hide it for the current session; it may reappear after restart. |

Additional user decision D-013: create a formal, checked-in JSON Schema and validate every status document against it before publication.

## 4. Candidate Approaches

### Option A: Independent sections in the existing status document — selected

Extend the existing endpoint and publication workflow with independent maintenance, update-policy and message sections. Client eligibility and priority determine presentation.

Pros: preserves the existing independent transport and operational workflow; maintenance changes do not remove version requirements; one fetched snapshot is internally consistent.

Cons/risks: requires contract changes across Flutter and operator tooling, plus a message revision independent of the publication revision.

### Option B: Extend the exclusive mode enum

Add update-required and richer announcement modes to the current schema.

Pros: smaller conceptual extension to the current model. Cons/risks: publishing maintenance would replace update requirements; cannot express the confirmed independent controls. Rejected during discovery.

## 5. Recommended Direction

Use Option A. Preserve the current public endpoint, transport isolation, bounded polling and operator confirmation workflow. Define a versioned JSON contract with independent controls and message identity. Specify observable behavior here; exact wire field names, route mechanics and dependency choices belong in coding planning.

## 6. Problem / Opportunity

The exclusive status mode cannot retain a minimum app version while communicating maintenance or selectively announcing a change. Users need an actionable update screen and relevant messages without losing the current maintenance recovery behavior.

## 7. Goals

- G-001: Direct unsupported clients to an available update through a blocking page.
- G-002: Keep maintenance independently controllable and preserve existing recovery semantics.
- G-003: Show one relevant, actionable message to a selected app-version range without repeatedly showing a dismissed revision.
- G-004: Keep status failures from causing an offline startup lockout.

## 8. Non-Goals

- NG-001: Backend minimum-version enforcement, feature flags, per-account targeting, push notifications, multiple-message campaigns or analytics.
- NG-002: Scheduled activation/expiry, automatic store-version discovery, automatic installation or automatic web reload.
- NG-003: Arbitrary remotely supplied routes, executable content, HTML/Markdown rendering or custom URL schemes in messages.
- NG-004: New administration UI, image-upload service, PDS/lexicon changes or desktop app targets.
- NG-005: Hosting changes, publication, deployment, dependency installation or implementation during this requirements stage.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| App user | Signed-in or signed-out user on a supported platform | Relevant notice, clear recovery action, accessible presentation. |
| Operator | Maintainer publishing the public status document | Independent controls, validation and deliberate publication to the selected environment. |
| Status service | Existing independent public JSON transport | Continue serving bounded anonymous configuration during AppView outages. |

## 10. Current Behavior

One document mode controls maintenance or an announcement. Maintenance covers the retained app and consumes back navigation; it has retry and an optional recovery estimate. Announcements use a dismissible modal and persist dismissal by document revision. Every publication receives a fresh document revision, so republishing currently resurfaces the announcement. Unknown/unavailable startup status permits ordinary access attempts.

## 11. Desired Behavior

The document can simultaneously define all three controls. The app evaluates platform and version, then shows only the highest-priority applicable flow. Clearing maintenance reveals an applicable update page; clearing an update requirement reveals an eligible undismissed message. Suppression by a higher-priority flow never counts as message dismissal. Absence of an applicable flow leaves ordinary app navigation in control.

The update page cannot be dismissed or escaped through app navigation. Its primary action opens the platform's CraftSky store listing or, on web, explicitly reloads the current app. The message remains a dismissible CraftSky dialog with optional image and CTA. Published prose is plain text.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Operators shall independently control maintenance, required updates and one version-targeted message in a single public JSON document. | Avoid replacing unrelated controls. | Prompt, D-001, D-009 | AC-001, AC-002 |
| BR-002 | Business | Must | Users below the platform minimum shall receive an actionable mandatory-update page when a valid update policy is accepted. | Move unsupported clients to a supported version. | Prompt, D-004 | AC-003, AC-004 |
| FR-001 | Functional | Must | The app shall keep anonymous status checks independent of AppView and account initialization; check on launch, foreground resume and every 60 foreground seconds, retaining the three-second request deadline, request coalescing and late-result protection. | Preserve existing availability and freshness behavior. | Codebase, D-008 | AC-005 |
| FR-002 | Functional | Must | Configuration shall allow a separate optional minimum supported version for iOS, Android and web. An absent platform minimum shall impose no update requirement for that platform. | Accommodate independent platform releases. | D-004 | AC-003 |
| FR-003 | Functional | Must | When update-required applies, the app shall show a full-page blocking experience that cannot be dismissed by back, Escape, a barrier, a deep link or ordinary app navigation. It shall remain effective regardless of sign-in state while applicable. | Enforce the confirmed client-side gate. | Prompt, BR-002 | AC-004, AC-006 |
| FR-004 | Functional | Must | The update primary CTA shall open the configured official CraftSky App Store listing on iOS and Google Play listing on Android. Launch failure shall leave the page usable with retryable feedback rather than claiming an update occurred. | Provide the correct update path. | Prompt, existing launcher dependency | AC-007 |
| FR-005 | Functional | Must | On web the update primary CTA shall explicitly reload the app at its current URL. The deployed caching arrangement shall allow this action to obtain the newly deployed version; there shall be no automatic reload loop or automatic removal of the gate without reevaluation. | Make web recovery effective. | Prompt | AC-008 |
| FR-006 | Functional | Must | Maintenance shall preserve existing title/body, optional recovery estimate, manual retry, five-minute freshness, foreground polling, retained app state and back-blocking behavior. Failed reads shall not renew maintenance freshness, and a new process shall not restore cached maintenance. | Preserve the agreed existing flow. | Prompt, D-008, codebase | AC-009 |
| FR-007 | Functional | Must | A message shall require a nonblank title, description, message revision and minimum app version, with an optional inclusive maximum. It may include an optional public HTTPS image and one optional CTA. It shall be eligible only when the running app version is within its range. | Relevant, bounded messages. | Prompt, D-002, D-009 | AC-010, AC-011 |
| FR-008 | Functional | Must | Messages shall retain explicit, barrier, Escape and back dismissal, storing dismissal locally per installation/browser storage scope and message revision. A successfully persisted dismissal shall stay effective after restart and account changes; a different revision shall be eligible again. Successful CTA activation shall dismiss the captured revision; failed activation shall retain the message with retryable feedback. If saving a CTA dismissal fails, the message shall remain hidden for the current session but may reappear after restart. Late activation or persistence completion shall not dismiss a replacement message or bypass a newly active blocking flow. | Avoid repeated messages. | D-003, D-012, codebase | AC-012, AC-022 |
| FR-009 | Functional | Must | Message revision shall be independent of document publication revision. Publishing unrelated maintenance or update changes shall not reset dismissal of an unchanged message revision. | Independent controls require independent identity. | D-001, D-003, publisher findings | AC-013 |
| FR-010 | Functional | Must | A CTA shall have nonblank text and exactly one destination: an HTTPS URL or a named supported in-app destination. Validation shall reject incomplete, ambiguous or unsupported destinations. In-app actions shall use normal authentication/account/policy gates. | Prevent broken navigation or bypasses. | D-006, D-012, repository constraints | AC-014, AC-022 |
| FR-011 | Functional | Must | Image absence, loading or failure shall not prevent reading or dismissing the message or using its CTA; image layout shall remain bounded and accessible. | Optional media must remain optional. | Prompt | AC-015 |
| FR-012 | Functional | Must | App and operator tooling shall validate a shared versioned contract, retaining the existing 16 KiB document limit, title/body limits and timestamp validation. Invalid documents shall not become authoritative status; no partial acceptance of a malformed snapshot. | Keep publication and client interpretation consistent. | Codebase | AC-016 |
| FR-013 | Functional | Must | Operator publish, validation, local-development, examples and clear workflows shall support independent sections and message identity. Publication shall preserve an explicitly supplied unchanged message revision; operators shall be able to remove maintenance without changing other controls. A full clear shall explicitly remove all controls and preview that result before publication. | Make the feature operable without accidental resets. | Existing tooling, D-001, FR-009 | AC-013, AC-017 |
| FR-014 | Functional | Must | Provide a formal versioned JSON Schema for status documents. Validate drafts and the final publication payload against it and semantic rules before every local or hosted upload, including setup/clear. Missing or broken schema/validator shall prevent upload. Validation shall work offline without retrieving remote schemas. | Executable document contract and publication gate. | D-013 | AC-023 |
| RULE-001 | Business rule | Must | Display priority shall be fresh maintenance, then applicable update-required, then eligible undismissed message. Only one shall display; suppression shall not dismiss a message, and valid configuration changes shall reevaluate the next flow without relaunch. | Deterministic coexistence. | D-001, D-011 | AC-002, AC-021 |
| RULE-002 | Business rule | Must | Version comparisons shall use semantic-version precedence, ignore build metadata and compare message boundaries inclusively. Invalid versions or reversed message ranges shall fail contract validation. | Avoid lexical comparisons and ambiguous targeting. | D-002, D-007 | AC-003, AC-010, AC-016 |
| RULE-003 | Business rule | Must | Unavailable, invalid or timed-out startup configuration shall permit ordinary access attempts after the bounded check, without enforcing a persisted minimum or inventing maintenance. Any failed, invalid or timed-out refresh shall release update enforcement from the last accepted policy for the current process; maintenance retains its independent five-minute trust rule. A later valid snapshot shall reevaluate update enforcement. Missing/invalid local version information shall not cause an update lockout or a version-targeted message. | Fail open on unavailable status. | D-005, D-011; ASM-002 | AC-018, AC-021 |
| NFR-001 | Non-functional | Must | Status presentation shall preserve the underlying navigation, editor data, media selections and sessions, shall not replay writes, and shall retain existing account/policy gates. Explicit web reload remains a user-initiated page reload and may discard in-memory state. | Avoid status-driven data loss and auth regressions. | Codebase, prompt web CTA | AC-006, AC-008 |
| NFR-002 | Non-functional | Must | Blocking pages and messages shall work at narrow/wide supported layouts, in light/dark themes and supported large text scales, with scrollable long content, accessible labels, hidden covered controls and appropriate focus restoration. | Preserve usability and accessibility. | Codebase | AC-019 |
| NFR-003 | Non-functional | Must | Status and optional image fetches shall not carry AppView credentials or account/device identifiers. Public content shall not change API destinations or credential handling. Diagnostics shall follow the mandatory logging/error-reporting guide, preserve typed causes/stacks and remain independent of feature behavior. | Preserve transport privacy and diagnostics boundaries. | Repository guide, current transport | AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001 | Given one valid document with all three sections, when accepted, then all controls remain independently represented; omission/removal of a section disables only that control. |
| AC-002 | BR-001, RULE-001 | Given applicable maintenance, update and message, only maintenance displays; removing it reveals update; removing the update requirement reveals the undismissed message. Higher-priority suppression does not persist a dismissal. |
| AC-003 | BR-002, FR-002, RULE-002 | Given iOS minimum `1.8.0`, iOS `1.7.9` is blocked, `1.8.0+13` and `1.8.1` are not; Android/web use their own values, and an omitted platform minimum does not block. `1.10.0` sorts after `1.9.0`. |
| AC-004 | BR-002, FR-003 | Given accepted applicable update policy, signed-in and signed-out clients see the full-page update experience and cannot dismiss it through back, Escape or surrounding UI. |
| AC-005 | FR-001 | Given launch, foreground resume, foreground polling, concurrent refreshes and a request exceeding three seconds, existing timing/coalescing behavior holds; a late response cannot overwrite newer accepted state. AppView/auth failure does not prevent the independent status check. |
| AC-006 | FR-003, NFR-001 | Given a retained editor/back stack and an active blocking flow, navigation/deep links do not expose interactive covered content; removing the gate preserves retained state and normal auth/account/policy checks. In-flight writes are neither replayed nor treated as canceled by the status presentation. |
| AC-007 | FR-004 | Given an update CTA tap on each mobile platform, the appropriate official listing opens. A simulated launch failure keeps the page actionable and reports failure; merely returning from the store does not clear an applicable requirement. |
| AC-008 | FR-005, NFR-001 | Given a deployed newer web version and an older running tab, one CTA tap reloads the current URL and loads the new version under the actual hosting cache configuration. If still below the minimum, the page remains gated after a successful status check, without an automatic reload loop. |
| AC-009 | FR-006 | Given accepted maintenance, unchanged valid reads renew five-minute trust; failed reads do not; expiry releases maintenance without asserting recovery. Manual retry, recovery estimate, normal startup on unavailable status and no maintenance persistence across restart remain intact. |
| AC-010 | FR-007, RULE-002 | Given message range `1.8.0` to `1.9.0`, versions below/above are excluded and both endpoints included regardless of build metadata. With no maximum, all versions at/above minimum are eligible. Standard prerelease precedence is respected. |
| AC-011 | FR-007 | Given an eligible message with only required fields, its title/description and dismissal appear without image or CTA. Adding optional image/CTA does not create a second message. |
| AC-012 | FR-008 | Given a successfully persisted dismissal through each supported path, restarting on the same installation/storage scope or switching account keeps the revision dismissed. A new revision becomes eligible. Delayed dismissal of an older revision does not dismiss a replacement. |
| AC-013 | FR-009, FR-013 | Given dismissed message revision A, publishing changed maintenance or minimum versions with a new document revision and unchanged message A does not resurface it; publishing message revision B does. |
| AC-014 | FR-010 | Given valid CTA text and either supported destination type, activation opens that destination through normal gates. Missing text/target, two targets, unsupported destination names and non-HTTPS URLs fail validation. |
| AC-015 | FR-011 | Given absent, slow, broken or oversized-display image content, the message text, CTA and dismissal remain usable without overflow or blocked interaction. |
| AC-016 | FR-012, RULE-002 | Given valid/invalid shared fixtures, app and operator validation agree on recognized fields, schema, text/body bounds, versions, ranges, CTA structure and estimates; malformed/oversized snapshots never partially change authoritative controls. |
| AC-017 | FR-013 | Given local validation/publication/clear workflows, operators can review the intended complete document and selected environment, preserve unaffected sections/message revision, and deliberately clear all controls. Invalid input, declined confirmation or EOF performs no remote write. Existing verified/unverified publication outcome semantics remain. |
| AC-018 | RULE-003 | Given offline, timeout, malformed or unsupported-schema status at startup, ordinary app access attempts are permitted without a cached update or maintenance gate. Missing/invalid local version data cannot lock the user out or show a targeted message. |
| AC-019 | NFR-002 | Given long content, both themes, narrow/wide layouts, large text, keyboard and screen reader navigation, visible controls remain reachable and labelled; covered controls are inaccessible; dismissal/uncovering restores appropriate focus. Verify physical/platform back behavior separately. |
| AC-020 | NFR-003 | Given status/image requests and diagnostic failures, inspect emitted requests and serialized sink output: no account/device credentials, protected canaries or arbitrary remote prose/URLs are exported, original typed failure/stack handling is preserved, and reporter failure does not change presentation/recovery behavior. |
| AC-021 | RULE-001, RULE-003 | Given an accepted update policy, when any subsequent refresh fails, times out or is invalid, then its update gate releases. Fresh maintenance remains until its original five-minute deadline without renewal from failure; after expiry, the invalidated update policy does not reappear. A later valid snapshot can enforce the minimum again. |
| AC-022 | FR-008, FR-010 | Given a message CTA, successful destination activation dismisses only the captured revision and attempts persistence; activation failure retains the message with retryable feedback. If persistence fails after success, the message stays hidden for the current session, including repeat reads/account switches, but may reappear after restart. Replacement messages remain eligible; a newly active maintenance/update flow remains effective when an older activation completes. |
| AC-023 | FR-014 | Given the checked-in schema, meta-schema validation succeeds, valid examples pass, and structurally invalid documents fail. Every upload path validates the exact final payload before writing, including after root-revision generation; invalid input or missing/broken schema/validator causes zero writes. Structurally valid reversed ranges fail semantic validation. Validation needs no network and shared app/tooling fixtures agree on contract outcomes. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Maintenance expires while an accepted update requirement exists | Reevaluate update policy only if it has not been invalidated by a failed refresh. An invalidated policy must not reappear when maintenance expires; expiry does not assert recovery. | FR-006, RULE-001 |
| EC-002 | Message range excludes current app or message was dismissed | No message; neither condition weakens maintenance/update handling. | FR-007, FR-008, RULE-001 |
| EC-003 | Failure refreshing an already visible update gate | Release update enforcement immediately on failed, invalid or timed-out refresh; maintenance retains its separate five-minute rule. A later valid policy may enforce again (D-011, AC-021). | RULE-003 |
| EC-004 | Operator lowers/removes minimum after mistaken publication | A subsequent successful refresh releases the update page without relaunch. | RULE-001 |
| EC-005 | App version remains too old after returning from store/reload | Reevaluate rather than assuming success; keep gating only when policy is authoritative. | FR-003, FR-004, FR-005 |
| EC-006 | Message changes while its older revision is being dismissed | Older completion cannot mark the new revision dismissed. | FR-008 |
| EC-007 | Missing platform minimum, no message, no maintenance | Ordinary app flow; no invented notice. | FR-002, RULE-001 |
| EC-008 | Browser storage cleared or app data removed | Prior dismissal may be lost; no cross-device/account synchronization is implied. | FR-008 |

## 15. Data / Persistence Impact

- New public data: independent maintenance configuration, per-platform minimum versions, message version range/revision/content and optional image/CTA.
- Changed fields: current exclusive mode contract becomes a versioned independent-section contract. Exact wire schema is deferred to coding planning.
- Persistence: retain local message dismissal keyed to message revision. Do not persist update policy for offline enforcement or restore maintenance across process starts.
- Migration: no database/PDS migration. Existing announcement-dismissal storage needs a deliberate preserve/reset choice during coding planning; see ASM-005.
- Backwards compatibility: shipped-app behavior compatibility is not required under repository policy. Production status publication still needs a deliberate rollout because clients with unsupported schemas fail open; it cannot update already-installed code to understand new fields.

## 16. UI / API / CLI Impact

- UI: blocking update page; existing maintenance UI; richer existing message dialog. Fixed UI controls localized through the app's current localization mechanism; operator content remains plain English as today.
- API: existing independent public `/app.json` contract changes; no AppView `/v1/*`, OAuth or lexicon changes.
- CLI: validators, examples, shared fixtures, local setup/publish/clear and operator instructions must reflect the new contract and independent message identity.
- Background jobs: none; retain foreground lifecycle polling only.
- No package requirement is imposed. Existing `url_launcher` can be evaluated before adding an update-specific dependency.

## 17. Security / Privacy / Permissions

- Status remains public and anonymous; maintenance/update/message rules apply independently of account sign-in.
- Operator publication retains explicit environment selection, review of intended public content and confirmation. This artifact authorizes no live writes or deployment.
- Message content, images and links must be public; no credentials, signed capability URLs or private payloads belong in the JSON.
- HTTPS message URLs do not grant generic navigation authority: in-app destinations come from a supported named set and retain existing access gates.
- Optional images must not inherit API authorization. Operators are responsible for choosing an appropriate public image host; no new upload/hosting facility is in scope.
- Store destinations are official application configuration, not arbitrary message links. Route blocking is a client experience, not server authorization or a security barrier.

## 18. Observability

- Retain bounded classifications for expected unavailable/invalid status and unexpected failures using existing diagnostic entry points; no new analytics, metrics service or alerts.
- Any added version/CTA/image/dismissal diagnostics must comply with `docs/development/logging-error-reporting.md`, retain original typed causes and available stacks, and exclude document prose, full destination/image URLs, credentials and account/device identifiers.
- Failures in reporting must never affect eligibility, navigation, freshness, retry or dismissal. Sink-output verification uses positive expected fields and protected-value canaries.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Minimum raised before a platform update is available | Users cannot obtain the required version. | Independent platform thresholds; operator availability check and preview before deliberate publication; easy removal via later refresh. |
| RISK-002 | Web reload returns cached old code | User remains below minimum. | Validate the deployed cache path with an older running tab; do not claim ordinary reload is sufficient without evidence. |
| RISK-003 | Document publication regenerates message identity | Unrelated changes repeatedly show dismissed messages. | Separate document/message revisions and regression coverage. |
| RISK-004 | Fail-open policy permits unsupported clients | Client-side update requirement is bypassable during status failure. | Explicit accepted availability tradeoff; no backend enforcement claim. |
| RISK-005 | Rich message content breaks layout or leaks request context | Unusable dialog or privacy regression. | Plain text, bounded layout, optional-image fallback and isolated public fetches. |
| RISK-006 | Old clients do not understand the new schema | Previously installed versions cannot be forced to upgrade through the new flow. | Document rollout limitation; coordinate client release and status publication. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Resolved by user decision D-011: failed refresh releases the current update gate; maintenance freshness is unchanged. | No longer an open assumption; normative coverage is RULE-003 and AC-021. |
| ASM-002 | If the running version cannot be read/parsed, update/message eligibility fails open; maintenance still works. | An alternative needs explicit fallback semantics. |
| ASM-003 | Initial named CTA destinations cover stable top-level destinations such as Feed, Projects, Search, Notifications, Profile and Settings; exact supported names are finalized in coding planning. No parameterized post/profile target is required initially. | Additional parameterized targets need bounded contract and access-gate tests. |
| ASM-004 | Resolved by user decision D-012: successful CTA activation dismisses; activation failure retains it; persistence failure after success hides it for the session but allows recurrence after restart. | No longer an open assumption; normative coverage is FR-008/010 and AC-022. |
| ASM-005 | Legacy announcement dismissal can be reset for a newly identified message; no cross-version preservation guarantee is required. | Reusing old identities would need a migration mapping. |
| ASM-006 | Message images are decorative supplements to the required description, so no essential information exists only in the image. | Essential-image content would need an explicit alternative-text field and validation. |
| ASM-007 | Semantic-version prerelease precedence follows the standard; release thresholds normally use `major.minor.patch`. Build metadata never affects eligibility. | Restricting configuration to stable versions instead changes validation fixtures. |

## 21. Open Questions

- Blocking for requirements/test design: none. The agreed direction is captured; assumptions above are explicit review points.
- D-011 and D-012 resolve the former update-refresh and CTA-dismissal blockers. Finalize the named destination set, exact schema and bounded revision-reuse policy (DR-003) in the coding plan.
- Release verification: establish actual store listing availability and web reload/cache behavior separately; neither is proven by this document or local tests.

## 22. Review Status

Status: Reviewed

Risk level: Medium — user-visible blocking/navigation behavior and a public configuration contract change; no new auth authority, payments or destructive data operations.

Review: Completed in `03-document-review.md`; approved with non-blocking notes after user-confirmed D-011/D-012 revisions.

Reviewer: Codex document review. The user confirmed the discovery direction and the DR-001/DR-002 behavior resolutions; coding and deployment remain separate stages.

Date: 2026-10-20

Notes: Requirements stage only. No implementation, tests, dependencies, commits, publication or deployment performed.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-20-app-status-flows/01-requirements.md`.
- Next artifact: `02-acceptance-tests.md`, using `write-acceptance-tests` after review or explicit review skip.
- Must-cover IDs: BR-001–BR-002, FR-001–FR-014, RULE-001–RULE-003 and NFR-001–NFR-003; AC-001–AC-023 provide the coverage contract.
- Unit: semantic-version boundaries, platform selection, validation parity, independent revision/dismissal, stale/late responses and priority transitions with an injected clock.
- Widget: blocked back/deep-link presentation, retained state/account gates, message eligibility/dismissal/CTA/image fallback and accessibility.
- Integration: local Worker/publisher/clear behavior, persistence across restart, native store launch failure/success and anonymous browser/native transport.
- Manual/release: official listing behavior on physical iOS/Android, web update from a stale open tab under deployed caches, VoiceOver/TalkBack/web reader and platform back behavior.
- Regression: current maintenance lifecycle/freshness/retry, status transport privacy/deadline, account/policy navigation and diagnostic sink output.
- Blocking open questions: none for test design; preserve assumptions explicitly in test cases rather than silently treating them as user-confirmed decisions.
