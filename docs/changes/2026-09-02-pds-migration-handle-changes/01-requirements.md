# Requirements: PDS Migration And Handle Change Resilience

## 1. Initial Request

Ensure Craftsky continues to identify, display, index, and act for the correct AT Protocol account when a user keeps the same DID but changes handle, migrates to another PDS hosting provider, or does both. Document all required changes and fixes. Craftsky is not in production, so implementation may make breaking API, persistence, and client changes without compatibility shims.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/internal/auth/session_coordinator.go` is the shared boundary for persisted Indigo OAuth sessions and authenticated PDS effects, but currently validates endpoint safety rather than continued DID authority.
  - `appview/internal/app/federated.go` creates authenticated PDS clients from persisted session endpoints without comparing them with the DID's current authoritative PDS.
  - `appview/internal/auth/pds_errors.go` recognizes token/authentication failures but not endpoint movement as a reauthentication condition.
  - `appview/internal/auth/background_session_selector.go` selects background-write sessions by recent Craftsky activity without excluding sessions for obsolete PDS endpoints.
  - `appview/internal/ingestion/service.go` treats ordinary Tap identity events as handle-cache refresh hints and currently terminalizes an owner directly from a Tap `deleted` identity status.
  - `appview/internal/app/tap_repository.go` reconciles known ambiguous record sources, not the complete indexed repository.
  - `appview/internal/auth/initialize_profile.go` requests Tap repository tracking on a best-effort basis even though durable repository jobs exist.
  - `appview/internal/api/identity_cache_store.go` has an unconditional authoritative-handle upsert path outside the version-fenced refresh flow.
  - `appview/internal/api/identity_cache_refresh.go` retains a previously searchable handle when the DID is authoritatively found to have no valid handle.
  - `app/lib/auth/services/session_validation_coordinator.dart` validates the DID returned by `/v1/whoami` but discards its current handle.
  - `app/lib/auth/providers/active_account_identity_provider.dart` loads the signed-in account through the persisted handle rather than the DID or `/v1/profiles/me`.
  - `app/lib/shared/rich_text/facet_action_handler.dart` navigates mentions through visible historical handle text instead of the mention facet DID.
  - `app/lib/profile/pages/profile_page.dart` determines profile ownership through handle equality.
  - `app/lib/router/router.dart` and multiple profile-link call sites use mutable handles as internal route identity.
  - `app/lib/search/models/recent_search.dart` stores profile DIDs, but `app/lib/search/pages/blank_search_view.dart` reopens recent profiles through their stored handles.
  - `app/lib/auth/models/account_deletion.dart` stages deletion confirmation using the locally persisted handle while the server binds the intent to its independently resolved current handle.
  - `docker-compose.yml` pins Indigo Tap `0.1.10`, whose source has migration-era resync limitations, including no deleted-record reconciliation.
- Existing patterns:
  - Account ownership, Craftsky sessions, indexed records, relationships, drafts, notifications, and most lifecycle state are keyed by DID.
  - AT URIs created by Craftsky use DID authorities.
  - Registration callback completion verifies that the authorization server belongs to the current PDS, but ordinary login callback completion can combine newly resolved PDS location data with credentials issued by the authorization server selected before migration.
  - Security-sensitive handle-targeted mutations use an authoritative resolver.
  - The Flutter app talks only to the AppView and does not persist PDS credentials or PDS locations.
  - Tap identity events already durably enqueue identity refresh work.
- Current behavior:
  - A handle change can leave the Flutter session, profile routes, account switcher, recent searches, mention navigation, caches, and deletion confirmation using the old handle.
  - A PDS migration leaves existing OAuth credentials pointed at the old PDS until an operation happens to produce a recognized terminal authentication error or the user signs in again.
  - Some old-PDS responses remain generic upstream failures and can be retried indefinitely.
  - Tap is expected to follow the DID to its current PDS, but Craftsky has no end-to-end migration proof or complete downstream repair mechanism if Tap does not emit a sufficient record diff.
- Constraints discovered:
  - AT Protocol defines the DID as canonical identity. Handles are mutable, must be bidirectionally verified, and may be reassigned to another DID.
  - The DID document is authoritative for the current PDS endpoint, signing key, and claimed handle.
  - OAuth access and refresh credentials are bound to the resource server and authorization server that issued them. Credentials from the old PDS must not be redirected to a new PDS.
  - PDS migration may require user reauthentication. A temporary migration or identity-propagation window must not be treated as account deletion or DID loss.
  - `#identity` is a refresh hint for any DID-document change, including a PDS endpoint or signing-key change when the handle is unchanged.
  - Writes continue to go through the authoritative PDS; reads continue to come from the AppView.
  - Flutter must continue to hold only Craftsky session tokens.
  - No production compatibility or persisted-user migration constraints apply.
- Test/build commands discovered:
  - `go test ./internal/auth ./internal/api ./internal/ingestion ./internal/tap` from `appview/`.
  - `flutter test test/shared/rich_text/faceted_text_actions_test.dart test/auth` from `app/`.
  - Full project verification uses `just test` with the Compose PostgreSQL service.
  - The existing targeted Go and Flutter suites passed during discovery, but they do not simulate PDS migration or complete handle-change behavior.

## 3. Clarifying Questions And Decisions

### Q1: Must existing implementations and wire contracts remain backwards compatible?

Answer: No. The user confirmed the app is not in production.

Decision / implication: The implementation may replace handle-keyed client routes, change API response fields or error codes, revise persistence schemas, update dependencies, and remove obsolete behavior directly. It must not add compatibility shims solely for unshipped behavior.

### Q2: Should an old OAuth session be transparently reused against the new PDS?

Answer: No user choice is required because the AT Protocol security model determines this behavior.

Decision / implication: Craftsky must detect that the persisted OAuth authority is obsolete, stop using that OAuth parent, preserve DID-owned Craftsky data, and require a fresh OAuth authorization against the current authoritative PDS. Tokens and DPoP state issued by the old authority must never be sent to the new PDS.

### Q3: Is a handle a valid durable identity key?

Answer: No user choice is required because AT Protocol defines handles as mutable aliases.

Decision / implication: Durable identity, internal navigation, authorization, ownership checks, cache identity, and record references must use DID. Handles remain current presentation and user-input aliases.

### Q4: When and how should client sessions be validated?

Answer: The backend validates authority centrally and uncached before every authenticated PDS effect. Flutter may validate on cold start for earlier feedback but does not gate all write controls.

Decision / implication: A user may attempt one write before learning that reauthorization is required, but no credential reaches an obsolete PDS. Existing invalid-session handling applies without a migration-specific client state machine.

### Q5: What is the reconciliation boundary?

Answer: Tap remains the primary create/update ingestion and backfill path. Craftsky owns only a narrow DID-scoped repair sweep after same-DID reauthorization or existing synchronization uncertainty.

Decision / implication: The sweep fully fetches and verifies one authoritative repository before inferring deletes, then may apply repairs incrementally through existing indexers. Briefly mixed projections and existing backfill side-effect behavior are accepted; no double-buffered serving or reconciliation-specific notification mode is required.

### Q6: What happens to scheduled posts during migration?

Answer: They may resume automatically only within the existing 30-minute publication window after valid current-PDS authorization is restored.

Decision / implication: Work outside that window transitions to `needs_attention` rather than publishing late or being discarded.

### Q7: What are the canonical handle and profile contracts?

Answer: Canonical Flutter profile URLs contain the DID. The existing backend identifier route remains the explicit handle/DID alias boundary, and old handle mappings are removed after authoritative change. An identity without a valid handle uses `handle.invalid` in existing non-null API fields.

Decision / implication: Flutter renders "handle unavailable" rather than the sentinel or old handle. Account deletion always confirms the full server-bound DID, including when a valid handle exists.

### Q8: How broad are migration tests and importer support?

Answer: Defer migration-specific standalone Instagram importer behavior. Automated migration coverage uses HTTP fakes rather than two real local PDS instances.

Decision / implication: Main-app tests still model PDS endpoint and signing-key changes, complete snapshots, missing live events, and same/changed-handle variants at protocol boundaries.

### Q9: What proves an OAuth authority mismatch, and how is it enforced?

Answer: At OAuth callback and immediately before each authenticated PDS effect, require successful uncached DID resolution and verification of current PDS and authorization-server metadata. Identity hints, handle changes, timeouts, DNS failures, and generic old-PDS errors alone do not force logout; existing conclusive token-error handling remains valid.

Decision / implication: Once proven at a foreground or background effect boundary, fence only the exact stale parent and its child Craftsky sessions through existing terminalization. Existing cleanup performs bounded best-effort revocation against the original authorization server; no proactive identity-event invalidation system is required.

### Q10: What does a missing Craftsky profile mean after migration?

Answer: If and only if a complete verified authoritative snapshot lacks the Craftsky profile record, apply the existing profile-deletion/departure policy.

Decision / implication: Stop member-only writes and scheduled work as ordinary profile deletion requires, but do not terminalize the DID as deleted. PDS movement by itself never implies departure.

### Q11: Can the design rely more heavily on Tap and accept reduced UX guarantees?

Answer: Yes. The user selected the lean hybrid approach and accepted all identified simplifications and minor UX detriments.

Decision / implication: Tap remains the primary create/update ingestion and backfill path. Craftsky adds only a narrow verified repository repair sweep for synchronization uncertainty and same-DID reauthorization, centralizes uncached authority verification at the shared OAuth session boundary, permits a failed first write instead of client-wide cold-start write gating, uses `handle.invalid` rather than a nullable/status pair, confirms deletion with the full DID, reuses existing revocation and scheduled-post machinery, permits incremental projection after complete snapshot verification, and defers importer-specific migration UX.

## 4. Candidate Approaches

### Option A: Lean Hybrid With Tap-First Ingestion

Summary: Keep Tap as the normal create/update ingestion and backfill path, add uncached authority verification at the one shared OAuth-session boundary, require same-DID reauthorization after verified migration, and use one narrow verified repository sweep to repair gaps and safely identify deletes.

Pros:
- Matches AT Protocol's identity and OAuth trust model.
- Prevents credentials from being sent to an unauthorized or obsolete host.
- Makes historical mentions, internal links, account data, and UI ownership stable across handle changes.
- Reuses existing coordinated PDS clients, exact-parent terminalization, revocation cleanup, repository jobs, indexers, and scheduled-post retries.
- Avoids authority-cache persistence, proactive client write gating, snapshot double buffering, and broad nullable-handle API changes.

Cons:
- Adds network latency and availability dependence to authenticated PDS effects.
- A migrated user may need to complete OAuth again.
- Readers may briefly see mixed old and repaired projections after a snapshot has been fully verified.

Risks:
- Incorrect authority-transition handling could revoke the wrong session or interrupt scheduled writes.
- Tap gaps still require the narrow AppView sweep because Tap resync does not discover deletes.

### Option B: Tap-Only Convergence

Summary: Add only OAuth authority safety and DID-first client identity, while relying exclusively on Tap for repository state.

Pros:
- Minimal implementation work.
- Avoids an AppView repository snapshot reader.

Cons:
- Tap `0.1.10` and current upstream do not emit deletes discovered during repository resync.
- Same-handle PDS/signing-key identity changes are not forwarded to consumers.
- A quiet repository can remain stale after missed relay events.

Risks:
- Deleted public records may remain surfaced indefinitely.
- Craftsky's portability guarantee would depend on known beta behavior outside its control.

### Option C: Full Defensive Reconciliation And UX Coordination

Summary: Add persisted authority freshness, proactive identity-event invalidation, cold-start write gating, repository-wide previous-state serving until reconciliation completes, explicit nullable handle status, and migration-specific UI/background state.

Pros:
- Provides the strongest migration UX and operational guarantees.
- Detects and communicates migration earlier on more devices.

Cons:
- Requires substantially more persistence, cache coherence, Flutter state, API surface, and concurrency control.
- Duplicates safety already provided by centralized write-time verification.

Risks:
- The larger implementation surface increases auth and synchronization regression risk.
- Complexity is disproportionate before production usage demonstrates the stronger UX is needed.

## 5. Recommended Direction

Recommended approach: Option A, lean hybrid with Tap-first ingestion.

Why: Tap is suitable as Craftsky's primary filtered, at-least-once ingestion transport, but its current resync cannot discover deleted records and does not forward every identity-authority change. A narrow authoritative sweep closes that correctness gap without duplicating Tap or requiring coordinated snapshot serving. Centralized uncached authority checks preserve OAuth safety while deliberately accepting extra write latency and a less polished first-failure experience.

## 6. Problem / Opportunity

Craftsky promises user-owned, portable AT Protocol accounts. Today, a user can keep the same DID and successfully migrate at the protocol level while Craftsky continues writing to the old PDS, displays an obsolete handle, routes a historical mention to the wrong person, or fails to converge its index. Closing these gaps turns the existing DID-first data model into an end-to-end portability guarantee rather than a storage convention.

## 7. Goals

- G-001: Preserve a user's Craftsky identity and AppView-owned data when the user's DID moves to another PDS.
- G-002: Ensure all future authenticated PDS effects target only the PDS currently authorized by the DID document.
- G-003: Converge AppView public data with the complete repository at the DID's authoritative PDS after migration.
- G-004: Ensure a handle change updates current presentation without changing account ownership or redirecting durable references.
- G-005: Give users an explicit, recoverable reauthentication experience when PDS migration invalidates an OAuth grant.
- G-006: Detect and diagnose migration convergence failures operationally.

## 8. Non-Goals

- NG-001: Craftsky will not perform or orchestrate the user's PDS-to-PDS account migration.
- NG-002: Craftsky will not transfer OAuth tokens, DPoP keys, passwords, blobs, private PDS state, PLC rotation keys, or signing keys between providers.
- NG-003: Craftsky will not make Flutter contact a PDS directly or store PDS credentials or endpoints on-device.
- NG-004: Craftsky will not change the rule that public records are written through the PDS and read through the AppView.
- NG-005: Craftsky will not preserve unshipped handle-based route or API compatibility.
- NG-006: Craftsky will not make historical post text display a new handle; immutable text remains as authored while its mention target remains the facet DID.
- NG-007: Craftsky will not treat a PDS migration, temporary PDS outage, invalid handle, or missing handle as DID deletion.
- NG-008: Craftsky will not redesign unrelated identity parsing, media delivery, subscriptions, or account-migration tooling.
- NG-009: This change will not add migration-specific recovery UX to the standalone Instagram importer; it remains bound by standard OAuth authority rules and may be handled separately.
- NG-010: Repository repair will not provide repository-wide atomic visibility, previous-snapshot double buffering, or migration-specific notification/push/analytics suppression.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Migrating member | A Craftsky member whose DID document changes to a different PDS provider, with or without a handle change. | Their Craftsky account and data remain associated with the same identity; the app explains and completes required reauthentication. |
| Handle-changing member | A member whose DID stays on the same or a different PDS while their canonical handle changes. | Current UI shows the new handle and durable links still reach the same account. |
| Other member | A user viewing old posts, mentions, profiles, notifications, follows, or search history involving the changed identity. | Historical interactions continue to target the original DID, not a new owner of the old handle. |
| AppView | Craftsky's trusted backend and OAuth client. | It resolves current identity authority, fences stale credentials, and keeps indexed state convergent. |
| Flutter client | The user-facing application holding only Craftsky session credentials. | It receives actionable recovery states and uses DID for internal identity. |
| Tap | The upstream repository synchronization sidecar. | It follows authoritative identity changes and emits sufficient events, with Craftsky repair available if it does not. |
| Operator | A Craftsky maintainer monitoring federation health. | They can distinguish migration, reauthentication, identity resolution, Tap, and reconciliation failures without sensitive data exposure. |

## 10. Current Behavior

Craftsky stores most durable data under DID. Registration callbacks verify current PDS and authorization-server authority, but ordinary login callbacks can resolve a new PDS after exchanging with an old authorization server. Persisted sessions retain their original authority, and later operations check endpoint safety without checking continued DID authority. Only a subset of old-provider errors expire the session.

Tap receives identity hints and refreshes its own identity cache, but forwards an identity event only when the handle changes. Its resync repairs present creates and updates but does not emit deletes for paths absent from the fetched repository. Craftsky disables relay replay in local Compose and its existing reconciliation job repairs only known ambiguous event/effect sources. A Tap `deleted` identity status currently terminalizes the Craftsky owner without separate authoritative proof.

Flutter stores sessions by DID but also persists the login-time handle. Validation confirms only the DID and ignores the current handle returned by the AppView. Profile loading, many internal links, recent searches, mention taps, ownership checks, and account-deletion confirmation rely on handles. These paths can fail or target another DID when a handle changes or is reassigned.

## 11. Desired Behavior

Before each authenticated PDS effect and before an ordinary OAuth callback is accepted, Craftsky resolves the DID's current PDS and verifies the corresponding authorization server. Existing indexed records and private AppView data remain owned by the DID. Received identity hints refresh presentation identity but are not themselves logout or permanent-deletion proof.

If the current PDS or authorization server differs from the authority that issued a persisted OAuth session, that exact OAuth parent and its child Craftsky sessions become unusable through existing terminalization and revocation cleanup. The next affected request presents an explicit reauthentication flow against the current authoritative PDS. Once reauthentication succeeds, writes and scheduled work resume through the new grant.

Tap remains the normal create/update ingestion and backfill path. On synchronization uncertainty or same-DID reauthorization, AppView fetches and fully verifies the authoritative repository before applying an idempotent repair through existing indexers; deletes are never inferred from an incomplete snapshot. Projection may proceed incrementally after verification. Flutter uses DID for internal identity, ownership, navigation, cache, and deletion confirmation, while handles remain presentation aliases and `handle.invalid` represents no current valid handle.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A member who migrates their account to another PDS while retaining the same DID shall retain the same Craftsky account, private AppView data, social relationships, and indexed content ownership. | Account portability is a core AT Protocol and Craftsky product promise. | Initial request; project vision; discovery | AC-001, AC-002, AC-017 |
| BR-002 | Business | Must | A member who changes handle while retaining the same DID shall remain the same Craftsky identity throughout the product. | Handles are mutable aliases and may be reassigned. | Initial request; AT Protocol identity findings | AC-003, AC-004, AC-005, AC-006 |
| BR-003 | Business | Must | Craftsky shall provide a clear and recoverable reauthentication path when migration makes an existing PDS OAuth grant obsolete. | Reusing old-provider credentials at a new provider is invalid and unsafe. | AT Protocol OAuth findings; discovery | AC-007, AC-008 |
| FR-001 | Functional | Must | At the shared OAuth-session coordinator, before each authenticated PDS effect uses a persisted parent, the AppView shall perform uncached DID resolution and verify that the parent's PDS and authorization server remain current. | This provides one simple mandatory safety boundary without authority-cache state or per-handler checks. | Backend audit; user answer | AC-007, AC-009, AC-010, AC-036 |
| FR-002 | Functional | Must | When the centralized check proves an OAuth parent obsolete, the AppView shall fence that exact parent and its child Craftsky sessions through the existing terminalization path, preserve independently valid parents, and require fresh authorization. No proactive background invalidation system is required. | Old tokens and DPoP state are authority-bound; write-time fencing is sufficient for safety. | Backend audit; AT Protocol OAuth; user answer | AC-007, AC-011, AC-037 |
| FR-003 | Functional | Must | The AppView shall not send an old OAuth parent's access token, refresh token, DPoP proof, revocation request, or authenticated PDS request to the newly resolved PDS or authorization server. | Prevents credential disclosure and authority confusion. | AT Protocol OAuth | AC-012 |
| FR-004 | Functional | Must | Temporary DID-resolution failures, PDS outages, migration propagation windows, and unclassified upstream failures shall remain retryable and shall not by themselves mark the DID deleted or purge DID-owned Craftsky data. | The protocol permits temporary unavailability during migration. | AT Protocol DID/account guidance | AC-013, AC-017 |
| FR-005 | Functional | Must | A terminalized or authority-stale OAuth parent shall produce HTTP `401` with error code `pds_session_expired`; Flutter shall use its existing invalid-session handling and direct the user to authorize again instead of indefinitely retrying. No migration-specific one-time reason UI is required. | The existing client flow needs only one actionable distinction from transient federation failure. | Backend and Flutter audits; user answer | AC-007, AC-008, AC-038 |
| FR-006 | Functional | Must | After fresh OAuth succeeds for the same DID at its current PDS, Craftsky shall preserve the account boundary, activate the new OAuth parent, and allow foreground and background effects to resume without creating a second Craftsky identity. | OAuth grants change; DID ownership does not. | Initial request; architecture | AC-001, AC-008, AC-011 |
| FR-007 | Functional | Must | A Tap identity event received by Craftsky shall invalidate and refresh current handle metadata, but shall be treated only as a hint and shall not by itself expire OAuth or terminalize an owner. | Tap does not forward every authority change and is not permanent-deletion proof. | AT Protocol Sync; Tap audit; user answer | AC-009, AC-014, AC-053 |
| FR-008 | Functional | Must | Same-DID OAuth reauthorization and existing synchronization-uncertainty handling shall durably enqueue the repository repair sweep. Craftsky need not persist signing-key fingerprints or classify handle-only identity events. | These existing boundaries provide recovery without a new identity-change state machine. | Tap and repository-job audit; user answer | AC-014, AC-015, AC-039 |
| FR-009 | Functional | Must | The repair sweep shall fetch `com.atproto.sync.getRepo` from the authoritative PDS, fully read and verify the repository commit/signature/root before inferring absence, and compare every Craftsky-indexed collection from one shared registry. After verification it may apply idempotent creates, updates, and deletes incrementally through existing indexers without repository-wide atomic visibility. | Tap remains primary ingestion, while the sweep closes its missed-delete and convergence gaps with minimal duplicate machinery. | Tap source review; AT Protocol Sync; user answer | AC-015, AC-016, AC-017, AC-040, AC-050 |
| FR-010 | Functional | Must | Onboarding, sign-in, and reauthentication shall durably enqueue Tap repository tracking so a transient Tap administration failure is retried after process restart. | Current direct `AddRepo` failure is warning-only. | Backend audit | AC-018 |
| FR-011 | Functional | Must | The selected Tap version and Craftsky ingestion path shall pass automated HTTP-fake migration scenarios covering PDS endpoint and signing-key changes, complete snapshots, missed live events, and same-handle and changed-handle variants before migration support is considered complete. Two real local PDS instances are not required. | Current Tap behavior is beta and migration-specific gaps are not covered. | Dependency audit; user answer | AC-014, AC-015, AC-016, AC-041 |
| FR-012 | Functional | Must | Flutter shall reconcile the current handle returned for the active DID and persist it without replacing the DID, Craftsky token, lease generation, account routing binding, or unrelated cached identity fields. | Current validation discards the new handle. | Flutter audit | AC-003, AC-019 |
| FR-013 | Functional | Must | The signed-in account's profile shall be loaded through the authenticated `me` resource or stable DID rather than the persisted handle. | A cold start must not depend on an obsolete handle. | Flutter audit | AC-019, AC-020 |
| FR-014 | Functional | Must | Internal profile navigation, durable profile links, recent-profile destinations, and profile provider/cache identity shall use DID whenever the DID is known. Handle input may resolve to a DID only at the user-input or external-alias boundary. | Prevents stale or reassigned handles from changing destination identity. | Flutter audit; AT URI guidance | AC-004, AC-020, AC-021 |
| FR-015 | Functional | Must | Profile ownership and authorization-sensitive UI shall compare the loaded profile DID with the viewer DID, never handles. | DID deep links currently expose visitor actions for the viewer's own account. | Flutter audit | AC-005 |
| FR-016 | Functional | Must | Tapping a mention shall navigate to the DID stored in the mention facet while preserving the historical visible mention text. | The facet DID is stable; visible text is historical presentation. | Flutter audit; AT Protocol facets | AC-006 |
| FR-017 | Functional | Must | Account deletion shall always bind, display, persist, and require exact confirmation of the full authenticated owner DID rather than any handle. | One immutable confirmation value removes handle refresh, nullability, and race branches. | Account-deletion audit; user answer | AC-022, AC-042 |
| FR-018 | Functional | Must | Profile search pagination and identity collections shall deduplicate accounts by DID only, not by handle. | A handle can move between DIDs while results are paginated. | Flutter search audit | AC-023 |
| FR-019 | Functional | Must | All authoritative identity-cache writes shall use one version-fenced update path that supersedes older refresh work and invalidates process-local entries for the DID and affected old/new handles after commit. | Prevents an older refresh from overwriting a newly resolved handle. | Backend identity-cache audit | AC-024 |
| FR-020 | Functional | Must | When authoritative resolution reports no bidirectionally valid handle, Craftsky shall use the `handle.invalid` sentinel in existing non-null handle fields, remove the prior current/searchable mapping, retain DID-owned content, and render "handle unavailable" instead of presenting the sentinel or old handle as current. | The protocol sentinel avoids a cross-API nullable/status migration. | AT Protocol Handle; cache audit; user answer | AC-025, AC-043 |
| FR-021 | Functional | Must | Canonical profile URLs shall contain the DID. Handle-based external aliases shall resolve bidirectionally to a DID and canonicalize to that route; after an authoritative handle change, Craftsky shall remove the old alias rather than retaining a permanent or timed redirect. | Durable URLs must survive handle changes without hijacking handles later assigned to another DID. | Recommended direction; user answer | AC-021, AC-044 |
| FR-023 | Functional | Must | Flutter may validate at cold start to refresh identity and prompt reauthorization, but shall not implement a client-wide write-gating state machine; any attempted write remains protected by the backend authority check. | Accepting one failed attempt materially reduces client coordination without weakening credential safety. | User answer | AC-045 |
| FR-025 | Functional | Must | Stale-authority failures for scheduled posts shall use the existing retry and 30-minute cutoff machinery; no migration-specific pause/resume state is required. | Reusing current behavior avoids a second scheduler lifecycle. | User answer; scheduled-post audit | AC-048 |
| FR-027 | Functional | Must | After the complete repository has been verified, repair projections may become visible incrementally. If the verified repository lacks the Craftsky profile record, Craftsky shall apply the existing profile-deletion/departure policy without terminalizing the DID as deleted. | Briefly mixed projections are an accepted simplification; authoritative membership deletion must still converge. | User answer | AC-050, AC-051 |
| FR-028 | Functional | Must | Proven-stale parents shall use the existing bounded revocation cleanup path against only their original verified authorization server; logout and reauthorization shall not wait for cleanup. | Existing machinery already provides safe provider-side cleanup. | User answer; OAuth rules | AC-052 |
| FR-029 | Functional | Must | Tap `deleted`, inactive, suspended, or takendown identity/account statuses shall not directly terminalize or purge a Craftsky owner. Permanent terminalization remains exclusive to the authenticated Craftsky account-deletion flow. | Tap status is a synchronization hint and can be missed, delayed, or observed during lifecycle propagation. | Tap audit; project deletion rule; user answer | AC-053 |
| FR-030 | Functional | Must | Supported Tap deployments shall resume from their durable relay cursor rather than intentionally reconnecting at the live head; `TAP_NO_REPLAY=true` shall not be part of the supported migration configuration. | Replay reduces gaps and lets Tap remain the primary ingestion path. | Tap source audit; user answer | AC-054 |
| FR-031 | Functional | Must | Before persisting an ordinary OAuth callback session, Craftsky shall uncached-resolve the subject DID's current PDS and authorization server and require them to match the resource/issuer used for the authorization and token exchange. On mismatch, it shall not persist the mixed-authority session and shall clean up the credential only through its original issuer. | Migration can occur between OAuth initiation and callback; the current ordinary login path can otherwise combine old credentials with a new host. | OAuth callback audit; user answer | AC-055 |
| NFR-001 | Non-functional | Must | Migration detection, session transitions, Tap tracking, repository reconciliation, and identity-cache updates shall be idempotent and safe under duplicate events, retries, concurrent requests, and process restarts. | Tap delivery is at least once and migration produces multiple related events. | Existing architecture; discovery | AC-015, AC-018, AC-024, AC-027 |
| NFR-002 | Non-functional | Must | Authoritative identity and PDS operations shall remain bounded by existing outbound-network safety controls, deadlines, and owner/session lifecycle fences. | Migration introduces dynamic destinations and must not weaken SSRF or lifecycle controls. | Architecture constraints | AC-010, AC-028 |
| NFR-003 | Non-functional | Must | Migration reconciliation shall not surface a partially reconciled repository as successfully complete; failed work shall remain durably retryable with bounded backoff. | Silent partial convergence is worse than a visible retry state. | Tap audit | AC-016, AC-029 |
| NFR-005 | Non-functional | Must | Logs, metrics, and API errors shall not expose OAuth tokens, DPoP private keys/proofs, Craftsky bearer tokens, deletion-confirmation hashes, or raw secret-bearing session JSON. | Identity recovery paths process sensitive credentials. | Security architecture | AC-031 |
| NFR-006 | Non-functional | Should | Existing non-migration behavior and DID-keyed content operations shall remain covered by regression tests after routes and caches become DID-first. | Broad identity refactoring can regress normal use. | Discovery | AC-032 |
| RULE-001 | Business rule | Must | DID is the sole durable identity and ownership key; handle and PDS endpoint are mutable metadata. | This is the AT Protocol identity model. | Initial request; AT Protocol | AC-001, AC-004, AC-017 |
| RULE-002 | Business rule | Must | OAuth credentials may be used only with the resource server and authorization-server authority for which they were issued and verified. | Prevents confused-deputy and credential disclosure failures. | AT Protocol OAuth | AC-009, AC-012 |
| RULE-003 | Business rule | Must | Public record writes shall go through the user's current authoritative PDS, while normal reads shall continue to come from the AppView. | Preserves the established Craftsky PDS/AppView split. | `AGENTS.md`; architecture | AC-010, AC-033 |
| RULE-004 | Business rule | Must | Flutter shall hold only Craftsky session credentials and shall not resolve, persist, refresh, or use PDS OAuth credentials. | Keeps PDS secrets server-side. | `AGENTS.md`; architecture | AC-034 |
| RULE-005 | Business rule | Must | A handle shall not authorize an action, establish profile ownership, deduplicate an account, or serve as a durable target when a DID is available. | Handles are mutable and reusable. | Handle-change audit | AC-005, AC-006, AC-023 |
| RULE-006 | Business rule | Must | Migration or handle loss shall not be interpreted as account deletion; only the existing authenticated permanent-deletion flow may terminalize and purge the Craftsky owner. | Protects user data during transient federation changes. | Project deletion rules; discovery | AC-013, AC-017 |
| RULE-007 | Business rule | Must | Because Craftsky is not in production, the implementation shall prefer direct replacement of obsolete identity behavior over compatibility branches for existing handle-based routes or stale persisted shapes. | Avoids unnecessary complexity for unshipped behavior. | User answer | AC-035 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-006, RULE-001 | Given a member is known as DID D on PDS A, when D migrates to PDS B and completes required Craftsky reauthentication, then Craftsky still exposes one account for D with the same private AppView data and ownership. |
| AC-002 | BR-001 | Given DID D has follows, drafts, saved state, notifications, moderation state, scheduled work, and indexed records before migration, when migration is detected, then no item is reassigned to another DID or discarded merely because the PDS endpoint changed. |
| AC-003 | BR-002, FR-012 | Given an active session stores `old.example` for DID D and `/v1/whoami` returns D with `new.example`, when validation succeeds, then the persisted and reactive presentation handle becomes `new.example` while the account key remains D. |
| AC-004 | BR-002, FR-014, RULE-001 | Given a profile destination was created when DID D used `old.example`, when that handle changes or is assigned to another DID, then the destination still opens D. |
| AC-005 | BR-002, FR-015, RULE-005 | Given the viewer DID is D, when the viewer opens D through any supported profile route, then the UI recognizes it as the viewer's own profile and does not expose follow, report, mute, or block actions against self. |
| AC-006 | BR-002, FR-016, RULE-005 | Given an old post displays `@old.example` with a mention facet for DID D, when the mention is tapped after the handle changes or is reassigned, then Craftsky opens D while the post still displays `@old.example`. |
| AC-007 | BR-003, FR-001, FR-002, FR-005 | Given a Craftsky session refers to an OAuth parent issued for PDS A and DID D now authoritatively points to PDS B, when an authenticated effect is requested, then no effect is sent through the stale parent and Flutter receives an explicit reauthentication-required result. |
| AC-008 | BR-003, FR-005, FR-006 | Given a migrated member receives reauthentication-required, when they authorize Craftsky at the current PDS for the same DID, then their existing Craftsky account resumes without registration as a new identity. |
| AC-009 | FR-001, FR-007, RULE-002 | Given Craftsky receives an identity hint, when it refreshes D's presentation metadata, then it does not treat the hint as OAuth proof. When an authenticated PDS effect is later attempted, the centralized uncached check compares the parent with D's current PDS and authorization server even if D's handle is unchanged. |
| AC-010 | FR-001, NFR-002, RULE-003 | Given the authoritative PDS still matches the OAuth parent, when a write is requested, then it proceeds through the existing bounded outbound client and lifecycle fence. |
| AC-011 | FR-002, FR-006 | Given stale and newly authorized OAuth parents both exist for DID D, when foreground or scheduled work selects a parent, then only a non-stale parent for the current authority is eligible. |
| AC-012 | FR-003, RULE-002 | Given a token and DPoP key were issued by PDS A, when DID D moves to PDS B, then captured outbound requests prove that those credentials are never sent to B or B's authorization server. |
| AC-013 | FR-004, RULE-006 | Given DID or PDS resolution temporarily times out during migration, when Craftsky handles the failure, then the operation is retryable, the owner remains non-terminal, and DID-owned data remains present. |
| AC-014 | FR-007, FR-008, FR-011 | Given a same-handle migration rotates the PDS endpoint and signing key, when Tap processes migration events and D reauthorizes or synchronization becomes uncertain, then Tap remains the primary ingestion path and a durable AppView repair sweep is scheduled without requiring Craftsky to receive or classify a same-handle identity event. |
| AC-015 | FR-008, FR-009, FR-011, NFR-001 | Given PDS B's relevant repository contains a record absent from AppView and an updated record with a new CID, when migration reconciliation completes, then AppView contains both authoritative versions exactly once. |
| AC-016 | FR-009, FR-011, NFR-003 | Given AppView contains a relevant record that is absent from the complete authoritative repository on PDS B, when reconciliation completes, then the record is removed from public AppView projections without deleting anything from the PDS. |
| AC-017 | BR-001, FR-004, FR-009, RULE-001, RULE-006 | Given a complete migration reconciliation for DID D, when records are added, updated, or removed, then all changes remain attributed to D and the owner lifecycle is not terminalized by migration. |
| AC-018 | FR-010, NFR-001 | Given Tap `AddRepo` is unavailable during onboarding, ordinary sign-in, or reauthentication, when the process restarts and Tap recovers, then each entry point's durable job retries and successfully tracks D without another user write. |
| AC-019 | FR-012, FR-013 | Given secure storage contains DID D with an old handle, when the app cold-starts after a handle change, then validation and own-profile loading succeed using D/current identity and update the stored handle. |
| AC-020 | FR-013, FR-014 | Given the app has a profile DID available from a post, notification, search result, follow list, relationship list, suggestion, or own account, when profile navigation occurs, then the request is keyed by that DID. |
| AC-021 | FR-014, FR-021 | Given a user enters or opens a valid current handle alias, when it resolves bidirectionally to DID D, then subsequent route state and cache identity use D rather than retaining the handle as ownership identity. |
| AC-022 | FR-017 | Given Flutter stores any current or stale handle, when the server creates a deletion intent, then the confirmation screen displays, persists, and submits the full server-bound owner DID and the server accepts only an exact DID match. |
| AC-023 | FR-018, RULE-005 | Given one DID changes handles between pages or one handle is reassigned between DIDs, when profile results are merged, then uniqueness is determined only by DID and no distinct DID is suppressed by handle equality. |
| AC-024 | FR-019, NFR-001 | Given an old refresh is in flight when a newer authoritative handle update commits, when the old refresh completes, then it cannot overwrite the newer value and all affected local identity-cache keys are invalidated. |
| AC-025 | FR-020 | Given authoritative bidirectional validation returns no current valid handle for DID D, when refresh completes, then the old handle is not returned as a current search mapping, D's content remains visible by DID, existing API handle fields contain `handle.invalid`, and user-facing UI renders "handle unavailable" rather than the sentinel or old handle. |
| AC-027 | NFR-001 | Given duplicate and out-of-order migration-related identity/account hints and repeated reconciliation jobs, when all work settles, then final session, identity, and repository state equals one correct processing of the newest authoritative state. |
| AC-028 | NFR-002 | Given a resolved PDS or authorization endpoint violates Craftsky's outbound destination policy or exceeds a deadline, when validation runs, then no request reaches the destination and the failure respects existing lifecycle fences. |
| AC-029 | NFR-003 | Given repository reconciliation fails after only part of the repository was inspected, when the attempt ends, then it is not marked complete and durable retry resumes safely without presenting partial success as convergence. |
| AC-031 | NFR-005 | Given migration success and failure paths are exercised, when logs, metrics, traces, and API errors are inspected, then none contains tokens, DPoP secrets/proofs, bearer credentials, confirmation hashes, or raw OAuth session JSON. |
| AC-032 | NFR-006 | Given no PDS or handle change has occurred, when existing profile, post, follow, relationship, notification, search, scheduled-write, and account-switching regression suites run, then their intended behavior remains correct under DID-first identity. |
| AC-033 | RULE-003 | Given a member reads and writes Craftsky public records after migration, when traffic is observed, then reads come from the AppView and writes go through the current PDS. |
| AC-034 | RULE-004 | Given Flutter storage and outbound requests are inspected before, during, and after migration recovery, then they contain only Craftsky session credentials and no PDS OAuth credentials or DPoP key material. |
| AC-035 | RULE-007 | Given obsolete unshipped handle-based route, cache, or persistence behavior conflicts with DID-first identity, when implementation is reviewed, then that behavior is replaced directly rather than retained through a compatibility branch without another concrete requirement. |
| AC-036 | FR-001 | Given a Tap identity hint, handle change, timeout, DNS failure, or generic old-PDS endpoint error, when Craftsky evaluates a parent, then that signal alone does not force logout. Given an authenticated PDS effect performs uncached DID resolution and verifies a different current PDS/authorization-server chain, then the mismatch is established before any credential is sent. Conclusive token errors may still expire the issuing parent through existing behavior. |
| AC-037 | FR-002 | Given several OAuth parents exist for DID D and the centralized check for an attempted foreground or background effect proves only parent P stale, when the transition commits, then P and its child Craftsky sessions are fenced while independently valid parents remain usable. |
| AC-038 | FR-005 | Given an affected device calls Craftsky after its parent is fenced, when the API responds, then it receives HTTP `401` with `{error: "pds_session_expired", message, requestId}` and follows the existing invalid-session path to fresh authorization without migration-specific reason state. |
| AC-039 | FR-008 | Given D completes same-DID OAuth reauthorization or an existing source-order check reports repository uncertainty, when that transaction commits, then the existing durable repository-job system coalesces a repair sweep without storing identity-change classifications or signing-key fingerprints. |
| AC-040 | FR-009 | Given one shared registry declares all indexed record collections, when reconciliation enumerates D's authoritative repository, then every registered collection participates in the same complete-snapshot comparison and no separate collection list silently omits indexed data. |
| AC-041 | FR-011 | Given HTTP fakes model PDS A, PDS B, DID/OAuth metadata, signing-key rotation, complete snapshots, and missed events, when migration contract tests run, then they verify convergence for unchanged and changed handles without requiring two real local PDS instances. |
| AC-042 | FR-017 | Given DID D has a valid, stale, or invalid handle, when the server creates an account-deletion intent, then the confirmation value is always the full server-bound DID and deletion proceeds only after an exact DID match. |
| AC-043 | FR-020 | Given D has an invalid handle, when identity UI renders an existing non-null API model, then it recognizes `handle.invalid`, shows "handle unavailable" with available display identity, and never presents the sentinel or last valid handle as current. |
| AC-044 | FR-021 | Given D changes from `old.example` to `new.example`, when the change is authoritatively committed, then the canonical route remains DID-based, `new.example` may resolve to D after bidirectional verification, and Craftsky no longer resolves `old.example` to D. |
| AC-045 | FR-023 | Given cold-start validation has not completed, when Flutter initializes, then reads and write controls may load normally. If the user attempts a write, the backend performs the mandatory authority check and either proceeds safely or returns the existing actionable reauthorization response without contacting an obsolete PDS. |
| AC-048 | FR-025 | Given a scheduled post becomes due while its OAuth parent is stale, when current-PDS authorization is available for an eligible attempt through the final attempt at exactly 30 minutes after the due time, then publication may resume automatically. If that final attempt fails, or authorization returns after 30 minutes, the post is `needs_attention` and is not automatically published. |
| AC-050 | FR-009, FR-027 | Given a repository download is incomplete or fails commit/signature/root verification, when the attempt ends, then no absence is applied as a delete. Given verification succeeds, repair may update visible projections incrementally in idempotent batches without repository-wide double buffering. |
| AC-051 | FR-027 | Given a complete verified authoritative snapshot lacks D's Craftsky profile record, when reconciliation commits, then the existing departure policy stops member-only writes and scheduled work, but D is not marked deleted and PDS migration alone is not recorded as the cause of departure. |
| AC-052 | FR-028, FR-003 | Given parent P is proven stale and its original authorization server is A, when terminalization occurs, then local fencing completes without waiting, bounded revocation may send P's revocation credential only to A, failures retry without blocking reauthorization, and no credential from P reaches the new provider B. |
| AC-053 | FR-007, FR-029, RULE-006 | Given Tap reports D as deleted, inactive, suspended, or takendown, when Craftsky ingests the identity event, then it records/refreshes synchronization state as applicable but does not terminalize or purge D. Only the authenticated permanent Craftsky deletion flow may make the owner terminal. |
| AC-054 | FR-030 | Given Tap has a durable relay cursor and reconnects after interruption, when the supported configuration starts it, then Tap requests replay from that cursor rather than intentionally starting at the live head. |
| AC-055 | FR-031, FR-003 | Given login begins against PDS A and D moves to PDS B before callback completion, when A returns tokens for D, then Craftsky resolves B and its authorization server, rejects the mixed-authority session before persistence, sends no A credential to B, and cleans up only through A. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | PDS changes but handle does not. | Tap handles normal repository ingestion; the next authenticated effect rejects stale OAuth, and same-DID reauthorization schedules repair despite the equal handle. | FR-001, FR-008 |
| EC-002 | Handle changes but PDS does not. | OAuth remains usable if its authority is still current; presentation and DID-first navigation update. | FR-001, FR-012, FR-014 |
| EC-003 | PDS and handle change together. | One DID-owned account remains; both authority recovery and handle refresh complete. | BR-001, BR-002, FR-006 |
| EC-004 | Old handle is reassigned to another DID. | Historical mentions, recent profiles, internal links, and caches continue to target the original DID. Current handle lookup may correctly resolve the new owner only at an explicit alias-input boundary. | FR-014, FR-016, FR-018 |
| EC-005 | DID has no valid handle or reports `handle.invalid`. | Content remains addressable by DID; the old handle is not presented as current or searchable; handle-dependent user input fails clearly. | FR-020, RULE-006 |
| EC-006 | DID resolution succeeds but current PDS is temporarily unavailable. | No owner deletion occurs; effects remain retryable; reauthentication is requested only when authority staleness is established. | FR-004, FR-005 |
| EC-007 | Old PDS remains online and accepts login after the DID points elsewhere. | Craftsky rejects it as non-authoritative even if it can issue tokens or serve a repository. | FR-001, RULE-002 |
| EC-008 | Migration rotates signing key and resets the repository commit chain. | Tap/Craftsky refresh identity, accept only correctly signed authoritative state, and reconcile without treating the chain reset as record deletion or DID replacement. | FR-007, FR-009, FR-011 |
| EC-009 | AppView missed all migration-era live record events. | Complete reconciliation repairs creates, updates, and deletes from the current PDS. | FR-009, NFR-003 |
| EC-010 | Several devices retain different OAuth parents for one DID. | Each stale parent is independently fenced; a valid current parent can serve eligible work; one device's failure does not corrupt another account. | FR-002, FR-006, NFR-001 |
| EC-011 | Scheduled publication becomes due during migration. | It does not write to the old PDS; it resumes after valid current-PDS authorization only within the 30-minute window, otherwise it moves to `needs_attention`. | FR-002, FR-006, FR-025 |
| EC-012 | Identity refresh races with login or another identity event. | The newest authoritative refresh wins and stale completion cannot reclaim the old handle. | FR-019 |
| EC-013 | Deletion starts immediately after a handle change. | The exact confirmation value comes from the server-created intent's authoritative identity result. | FR-017 |
| EC-014 | Reconciliation is repeated after process failure. | Reprocessing is idempotent and converges to the same final state. | NFR-001, NFR-003 |
| EC-015 | Handle alias cannot be bidirectionally verified. | It is not accepted as an identity target even if one resolution direction returns a DID. | FR-014, FR-020 |
| EC-016 | Identity hint arrives before the new PDS and OAuth metadata are consistently available. | The hint may refresh presentation identity but does not force logout; a later authenticated effect performs its own uncached authority check. | FR-001, FR-004, FR-007 |
| EC-017 | Repair discovers old records after migration. | Existing indexer behavior applies; migration-specific notification, push, analytics, and snapshot-serving modes are not required. | FR-009, FR-027 |
| EC-018 | Complete new-PDS snapshot omits the Craftsky profile. | Existing departure behavior applies without treating the DID as deleted. | FR-027 |
| EC-019 | Scheduled post authorization returns after the late-publication window. | The post moves to `needs_attention` and is not automatically published. | FR-025 |
| EC-020 | Old handle is later claimed by another DID. | No retained Craftsky alias redirects the old handle to its former owner; current bidirectional resolution may identify the new owner. | FR-021 |

## 15. Data / Persistence Impact

- New fields:
  - No persisted OAuth-authority cache, signing-key fingerprint, handle-status field, or migration-specific client state is required.
  - Existing durable repository-job fields should be reused; add repair progress only if required to resume bounded snapshot work safely.
- Changed fields:
  - Flutter stored-session handle becomes mutable presentation metadata refreshed by DID.
  - Client route/provider/cache keys change from handle-or-DID strings to canonical DID where identity is known.
  - Existing non-null API handle fields use `handle.invalid` after definitive handle validation failure.
  - Account-deletion intent confirmation is always the full owner DID.
- Migration required:
  - No authority-cache migration is planned. A database migration is needed only if the current repository-job schema cannot safely resume the narrow repair sweep.
  - Flutter secure-storage schema may be changed directly or reset because there are no production users.
  - Existing handle-keyed recent-search and route persistence may be replaced directly; no legacy conversion is required unless retained by implementation choice.
- Backwards compatibility:
  - Not required for unshipped API routes, error envelopes, Flutter storage, provider keys, or local development data.
  - AT Protocol records and lexicons are not expected to change. Any later-discovered lexicon change requires the separate lexicon/ADR workflow.

## 16. UI / API / CLI Impact

- UI:
  - Show a specific PDS reauthentication state rather than a generic action failure or silent repeated retry.
  - Do not add global cold-start write gating; an attempted write may be the first operation to discover migration.
  - Refresh account switcher, settings identity, own profile, and other current-handle presentation when the handle changes.
  - Recognize `handle.invalid` centrally and show "handle unavailable" without presenting the sentinel or stale handles.
  - Profile and mention navigation become DID-first.
  - Account deletion always displays the full server-bound DID confirmation value.
- API:
  - Return HTTP `401` with `pds_session_expired` for confirmed obsolete PDS authorization.
  - Return the full DID confirmation value from account-deletion intent creation; no handle/DID discriminator is needed.
  - Keep existing non-null handle shapes and use `handle.invalid`; do not add nullable `handle` or `handleStatus`.
  - Keep the existing backend profile identifier route as the explicit alias boundary. Flutter canonical URLs and known-identity requests use DID.
  - All `/v1/*` changes must retain camelCase JSON and the standard `{error, message, requestId}` error envelope.
- CLI:
  - No end-user CLI feature is required.
  - An operator-only reconciliation trigger/status command may be added if needed for verification and recovery, but must remain DID-scoped and must not become a generic PDS mutation facility.
- Background jobs:
  - Durable Tap tracking after onboarding/reauthentication.
  - Durable DID-scoped repository repair after same-DID reauthorization or existing synchronization uncertainty.
  - Identity-event refresh of presentation metadata only; no proactive OAuth invalidation worker.
  - Existing scheduled-post retries and stale-session handling remain the only scheduler path.
  - Existing OAuth cleanup performs bounded revocation against the original verified authorization server.

## 17. Security / Privacy / Permissions

- Authentication:
  - Reauthentication must start from authoritative DID-to-PDS-to-authorization-server discovery and verify the callback subject is the expected DID.
  - Ordinary login callback completion and every authenticated PDS effect must independently verify current authority.
  - Old credentials must use existing exact-parent terminalization once authority mismatch is established.
  - Identity hints and transient failures alone do not establish mismatch.
- Authorization:
  - DID remains the owner and viewer authorization boundary.
  - Handle equality must never grant own-profile controls or authorize a mutation.
  - Reconciliation is read-only against the PDS and may mutate only AppView projections for the same DID.
- Sensitive data:
  - OAuth access/refresh tokens and DPoP keys remain server-side.
  - Flutter stores only opaque Craftsky credentials.
  - Logs and metrics use bounded reason codes and must not include secret-bearing session payloads.
- Abuse cases:
  - A malicious old PDS must not remain authoritative merely because it still responds or accepts login.
  - An attacker claiming another user's old handle must not acquire historical links, mentions, private Craftsky state, or authorization.
  - A malicious DID document endpoint must remain subject to outbound boundary/SSRF validation.
  - Duplicate or forged migration hints must not override independently resolved identity authority.

## 18. Observability

- Events:
  - Reuse existing state transitions for OAuth parent terminalization/revocation, Tap tracking, and repository repair requested/completed/retried.
- Logs:
  - Emit structured, secret-free logs with operation name, bounded reason code, request/run ID where applicable, and whether the result was retryable.
  - Avoid logging raw handles unless existing privacy policy explicitly permits them; DID logging should follow the repository's established observability policy.
- Metrics:
  - Count authority-check outcomes, stale OAuth parents, reauthentication-required responses, Tap tracking retries, and repair outcomes using existing metric patterns.
  - Track repair backlog age and retry count.
- Alerts:
  - Alert on sustained repair backlog, repeated stale-parent selection attempts, and elevated Tap repository resync failures.
  - Thresholds may be set during implementation/operations design; the emitted signals are required before release.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Uncached authority resolution adds latency or is unavailable when a protected effect starts. | Writes can be slower or fail transiently. | Centralize the bounded check, retain explicit retryable errors, and add caching later only if measurements justify it. |
| RISK-002 | Existing OAuth code terminalizes all sessions for a DID rather than only stale parents. | Valid devices or newly authorized sessions could be signed out. | Fence transitions by DID, parent session ID, row version, and auth epoch; test multiple parents/devices. |
| RISK-003 | Tap does not emit complete migration diffs. | AppView can retain missing, stale, or deleted records indefinitely. | Keep Tap primary, enable relay replay, and retain the narrow verified Craftsky repair sweep. |
| RISK-004 | Repository repair overloads a PDS or AppView. | Federation latency and worker backlog increase. | Trigger only from reauthorization/uncertainty, reuse DID-scoped durable jobs, fully verify one bounded CAR, and apply idempotent batches. |
| RISK-005 | Handle reassignment races with identity-cache refresh. | Search or alias lookup temporarily maps a handle to the wrong DID. | Bidirectional authoritative resolution, version fencing, unique ownership updates, and invalidation of old/new aliases. |
| RISK-006 | Broad DID-first Flutter refactoring misses a handle-keyed surface. | One UI path can still fail or target the wrong account after a handle change. | Maintain an inventory test for profile navigation/cache call sites and add cross-feature acceptance coverage. |
| RISK-007 | Write-time migration discovery is less polished than proactive client gating. | The first post-migration write may fail and require retry after reauthorization. | Return the explicit `pds_session_expired` recovery response and rely on the backend to prevent unsafe delivery. |
| RISK-008 | Repository repair applies deletion from an incomplete snapshot. | Valid AppView content could disappear. | Fully consume and verify the repository commit/signature/root before inferring absence; incremental projection starts only afterward. |
| RISK-009 | Indigo OAuth or Tap behavior changes across upgrades. | Craftsky assumptions and tests become invalid. | Pin exact versions/images and rerun HTTP-fake migration contract tests before upgrades. |
| RISK-010 | Requirements still touch auth, synchronization, deletion, and routing. | Regression and security risk remain high despite reduced scope. | Require document review, focused acceptance tests, and HTTP-fake migration fixtures before merge. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The DID remains unchanged during the supported migration. | A new DID is a new identity and requires separate account-linking/product requirements. |
| ASM-002 | The current DID document is the authority for PDS endpoint and signing key, and handles are accepted only after bidirectional verification. | The entire trust and resolution design would need revision. |
| ASM-003 | A migrated account can complete a fresh OAuth flow at the current authoritative PDS after migration finalization. | Craftsky can preserve data but cannot resume authenticated writes until the provider supports valid OAuth. |
| ASM-004 | Relevant public repository state can be fetched completely and read-only from the authoritative PDS with `com.atproto.sync.getRepo`. | A relay- or provider-specific repair source and trust design would be required. |
| ASM-005 | Craftsky's existing DID-keyed AppView/private data does not embed hidden PDS-host ownership outside the audited OAuth session data. | Additional schema and cache migrations would be required. |
| ASM-006 | No lexicon record shape must change to support migration or handle changes. | A lexicon skill review, ADR, regeneration, and PDS record-evolution plan would become mandatory. |
| ASM-007 | There are no production users or compatibility commitments. | Storage and route migrations would need explicit backwards-compatibility and rollout requirements. |
| ASM-008 | Indigo Tap issue 1407 remains unconfirmed upstream, but the pinned source's missing delete reconciliation is sufficient reason to retain a Craftsky-owned reconciliation safety net alongside normal Tap ingestion. | If upstream behavior differs, the safety net and tests still provide deterministic convergence but implementation integration points may change. |

## 21. Open Questions

- [ ] Non-blocking: Set repository-repair CAR size, batch, lease, retry, and alert thresholds during coding/operations design.

## 22. Review Status

Status: Draft

Risk level: High

Review recommended: Required

Reviewer:

Date: 2026-09-02

Notes: The requirements were simplified to the approved lean hybrid. Explicit approval is still required before test design or implementation because this work changes OAuth authority handling, repository synchronization, account deletion, and navigation. No blocking product question remains.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-003`; `FR-001` through `FR-021`; `FR-023`, `FR-025`, `FR-027` through `FR-031`; `NFR-001`, `NFR-002`, `NFR-003`, `NFR-005`; `RULE-001` through `RULE-007`.
- Suggested test levels:
  - Acceptance: same-handle migration, changed-handle migration, combined migration, write-time reauthentication, old mention, old profile link, scheduled-write cutoff, `handle.invalid` presentation, and DID deletion confirmation.
  - Unit: OAuth callback/current-authority proof, centralized session authority checks, PDS error translation, exact-parent session eligibility, handle reconciliation, DID-based ownership, mention dispatch, profile deduplication, and identity refresh version fencing.
  - Integration: persisted-session transition through existing revocation cleanup, durable Tap tracking retry, non-terminal Tap account statuses, verified CAR repair with safe deletes, lifecycle preservation, and multi-parent background selection.
  - Contract: exact `pds_session_expired` envelope, non-null `handle.invalid`, full-DID deletion intent, existing profile identifier route, DID-first Flutter routes, and Flutter credential boundary.
  - Dependency/end-to-end: HTTP-fake PDS A to PDS B scenarios against exact pinned Tap/Indigo versions, including signing-key rotation, commit-chain reset, same/changed handle, missed live events, and process restart.
  - Regression: ordinary sign-in, profile reads, posts, follows, relationships, search, notifications, account switching, scheduled effects, Tap replay, and permanent deletion.
  - Security: stale-token non-forwarding, malicious old PDS, invalid/bidirectionally unverified handle, SSRF boundary, log redaction, and lifecycle fencing.
- Blocking open questions: None. High-risk review and explicit requirements approval are required before proceeding.
