# Requirements: Settings Page And Lean Account Deletion

## 1. Initial Request

Expand Settings with identity, account switching, disclosure chevrons, Notifications, About, Account, moved image-cache clearing, version information, and error-coloured Sign out. Delete account permanently removes the member's CraftSky membership, private CraftSky data, and every record in registered `social.craftsky.*` PDS collections without deleting their DID/PDS account, other namespaces, or blobs directly.

After reviewing the first implementation's size, the product owner approved a lean asynchronous deletion contract on 2026-08-11: keep fresh OAuth reauthentication, exact-handle confirmation, a durable worker, private cleanup, namespace boundaries, and a final empty PDS scan; remove indexer receipts/convergence gates, deletion-status credentials and UI, manual Retry, cleanup checkpoints, the deletion audit, and detailed deletion metrics.

## 2. Current Codebase Findings

- Settings is a Flutter/Riverpod surface using typed GoRouter routes and the shared account switcher.
- AppView owns OAuth/PDS credentials; Flutter holds only CraftSky bearer sessions.
- Existing Tap/indexer delete handlers already remove public indexed CraftSky records and derived effects idempotently.
- The PDS deletion loop can safely converge without per-record persistence by repeatedly scanning registered CraftSky collections from the beginning, treating missing records as success, and deleting the membership record last.
- Private account data spans Postgres, Instagram migration storage, scheduled-post state, and scheduled-media cleanup jobs. These deletion operations are or shall be idempotent so the complete cleanup can be replayed without per-component checkpoints.
- The first implementation added eight deletion tables and client/server status machinery. The approved simplification retains one operational job table plus OAuth request metadata only.

## 3. Clarifying Questions And Decisions

### Q1: What does Delete account delete?
Answer: The CraftSky membership, private CraftSky/AppView data, and every PDS record in a registered `social.craftsky.*` record collection.

Decision / implication: It never deletes/deactivates the DID or PDS account, another namespace, or a blob directly.

### Q2: Is fresh owner proof required?
Answer: Yes. Use PDS OAuth reauthentication, name the active `@handle`, and require that exact handle to be typed. CraftSky never collects a PDS password or email code.

Decision / implication: The fresh server OAuth session is bound to the durable job and never reaches Flutter.

### Q3: Must deletion wait for AppView indexers?
Answer: No. Existing indexers are trusted to process PDS delete events eventually.

Decision / implication: Terminal deletion does not wait for per-record receipts or inspect public indexed tables. Temporary stale AppView visibility is acceptable.

### Q4: What happens after acceptance?
Answer: The confirming client removes the account locally and returns to another retained account or Sign in. There is no progress screen, deleting switcher row, status credential, manual Retry, or user-visible phase model.

Decision / implication: The server retries automatically with capped backoff. Persistent failures are operationally visible through ordinary redacted logs/error monitoring. A later login while the job is pending must not mint an ordinary CraftSky session.

### Q5: Are checkpoints, an audit, or detailed metrics required?
Answer: No.

Decision / implication: Private cleanup replays idempotently, terminal finalization removes the operational job, and only ordinary redacted success/failure logging remains.

## 4. Candidate Approaches

### Option A: Lean Durable Deletion — Approved
Keep destructive boundaries and a restartable job, but let indexers converge independently and remove user-facing recovery/status infrastructure.

Pros: materially smaller schema, backend, and Flutter surface; preserves secure confirmation and safe PDS scope; robust against network/process interruption.

Cons: the member cannot monitor progress; AppView may briefly show stale indexed content; persistent failures require operational intervention or a later sign-in to refresh OAuth authority.

### Option B: Fully Synchronous Deletion — Rejected
Delete everything inside the acceptance request.

Pros: smallest apparent API.

Cons: unsafe for pagination, network interruption, large repos, and external cleanup; the client could time out after partial irreversible work.

### Option C: Receipt-Backed Observable Deletion — Superseded
The previously implemented durable status, receipt, checkpoint, audit, and recovery design.

Pros: strongest observable terminal guarantee and user recovery experience.

Cons: disproportionate complexity for the current product stage.

## 5. Recommended Direction

Use Option A. Acceptance creates or resumes one owner-scoped job and revokes ordinary CraftSky access. The worker idempotently purges private data, repeatedly deletes registered CraftSky PDS collections until empty, performs a final empty scan, removes retained OAuth authority, and deletes its operational row. Tap/indexers converge independently.

## 6. Problem / Opportunity

The Settings page needs the requested account-management structure, while permanent CraftSky deletion must remain secure and restartable without carrying a production-scale progress, receipt, audit, and recovery subsystem before the app has users.

## 7. Goals

- G-001: Deliver the requested identity-led Settings, About, and Account surfaces.
- G-002: Permanently remove the current CraftSky membership, private data, and registered CraftSky PDS records after explicit fresh owner confirmation.
- G-003: Preserve the DID/PDS account, non-CraftSky records, and PDS-owned blob lifecycle.
- G-004: Make server deletion safely replayable across failures while keeping the client flow immediate and simple.

## 8. Non-Goals

- NG-001: Delete or deactivate the user's AT Protocol account or DID.
- NG-002: Delete PDS records outside registered `social.craftsky.*` record collections.
- NG-003: Delete blobs directly or wait for PDS garbage collection.
- NG-004: Wait for, inspect, receipt, or otherwise coordinate Tap/indexer convergence.
- NG-005: Provide deletion progress, a disabled switcher row, status credentials, manual Retry, or replacement-reauthentication status flows.
- NG-006: Retain a deletion audit or implement deletion-specific metrics.
- NG-007: Persist per-record delete markers or per-component cleanup checkpoints.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Member | Signed-in CraftSky user | Clear Settings and an explicit permanent-deletion boundary. |
| AppView | CraftSky backend and OAuth/PDS authority holder | A minimal durable job that cannot escape owner/namespace scope. |
| Tap/indexers | Existing public-data projection pipeline | Continue processing ordinary delete events independently. |
| Operator | Maintains the pre-production service | Redacted failure logs for jobs that cannot complete automatically. |

## 10. Current Behavior

The branch contains the requested Settings UI plus an over-specified deletion implementation with status/recovery credentials, per-record expected/receipt state, convergence checks, cleanup checkpoints/artifacts, a retained audit/sweeper, detailed metrics, Flutter status persistence/polling, and manual recovery screens.

## 11. Desired Behavior

Settings retains the requested expanded hierarchy. Delete account still requires fresh OAuth and exact-handle confirmation. A successful acceptance response means the server durably owns the request; the client immediately removes the account and its local product data. The backend retries a small owner-scoped operation until private cleanup is complete and a final scan proves every registered CraftSky PDS collection empty. Completion does not wait for AppView projections and retains no deletion audit.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Settings shall present the active identity and requested account/application destinations clearly. | Fulfils the original Settings request. | Prompt | AC-001–AC-014 |
| BR-002 | Business | Must | Existing Settings destinations and Notifications behavior shall remain available. | Avoids regression while reorganizing the page. | Prompt / codebase | AC-003–AC-006 |
| BR-003 | Business | Must | A member shall be able to permanently delete their CraftSky membership, private CraftSky data, and registered CraftSky PDS records without deleting their AT/PDS account or unrelated records. | Provides user control within the federated boundary. | User decision | AC-015–AC-018 |
| BR-004 | Business | Must | About and Account shall be separate child pages. | Keeps Settings extensible and readable. | Prompt | AC-005, AC-007, AC-011 |
| FR-001 | Functional | Must | Settings shall show the active account's avatar, display name when present, and normalized `@handle` above its rows. | Makes account scope explicit. | Prompt | AC-001 |
| FR-002 | Functional | Must | Switch account shall be the first row beneath identity and open the existing shared account switcher. | Reuses established multi-account behavior. | Prompt | AC-002 |
| FR-003 | Functional | Must | Every row that opens another in-app page or switcher shall show a trailing chevron. | Requested disclosure affordance. | Prompt | AC-003 |
| FR-004 | Functional | Must | Settings shall contain Customisation, Languages, Notifications, Followers, Following, Muted accounts, Blocked accounts, Find people from Instagram, Account, About, and Sign out in the approved order/grouping. | Defines the expanded hierarchy. | Prompt / codebase | AC-004, AC-005 |
| FR-005 | Functional | Must | Every existing destination shall preserve typed navigation and Back behavior. | Prevents navigation regression. | Codebase | AC-005 |
| FR-006 | Functional | Must | Notifications shall open the existing notification settings page unchanged. | Avoids duplication. | Prompt | AC-006 |
| FR-007 | Functional | Must | About shall contain Terms, Privacy policy, Clear image cache, and app version/build information. | Groups requested application information. | Prompt | AC-007 |
| FR-008 | Functional | Must | Terms shall open `https://craftsky.social/terms` externally. | Uses the canonical destination. | Prompt | AC-008 |
| FR-009 | Functional | Must | Privacy policy shall open `https://craftsky.social/privacy` externally. | Uses the canonical destination. | Prompt | AC-008 |
| FR-010 | Functional | Must | Clear image cache shall exist only on About and preserve its existing two-cache busy/success/error behavior. | Moves rather than changes the action. | Prompt / codebase | AC-007, AC-009 |
| FR-011 | Functional | Must | About shall reuse the existing localized version/build formatter. | Keeps application metadata consistent. | Prompt / codebase | AC-010 |
| FR-012 | Functional | Must | Account shall initially contain only Delete account. | Requested narrow page. | Prompt | AC-011 |
| FR-013 | Functional | Must | Delete account shall require fresh PDS OAuth reauthentication, a destructive explanation naming the captured active `@handle`, and exact typing of that handle before acceptance. | Reduces wrong-account deletion. | User decision | AC-012, AC-013, AC-025, AC-035 |
| FR-014 | Functional | Must | Confirmation copy shall state that deletion is permanent and asynchronous; removes CraftSky membership, private data, and CraftSky PDS records; signs the account out; may leave temporarily stale AppView projections; and does not delete the AT/PDS account, other-app records, or blobs directly. | Communicates the lean terminal boundary accurately. | User decision | AC-012, AC-036 |
| FR-015 | Functional | Must | The worker shall idempotently delete all current private CraftSky data and repeatedly list/delete every registered CraftSky record collection for the owner until a final scan is empty. | Provides durable deletion without per-record/checkpoint state. | User decision / architecture | AC-015–AC-017, AC-037 |
| FR-016 | Functional | Must | After acceptance, the client shall remove the deleting account and its local product data immediately, activate the MRU retained account when present, otherwise show Sign in, and retain no deletion-status row. | Replaces the complex status UX. | User decision | AC-018, AC-038, AC-043 |
| FR-017 | Functional | Must | The server shall automatically retry transient deletion failures with capped backoff. It shall expose no manual Retry or attention workflow; persistent failures are operationally logged without restoring ordinary access. | Keeps recovery server-owned and simple. | User decision | AC-023, AC-039 |
| FR-018 | Functional | Must | Sign out shall remain immediate, account-scoped, non-destructive, and use the theme error colour for icon and label. | Distinguishes reversible sign out. | Prompt | AC-014, AC-024 |
| FR-019 | Functional | Must | Fresh proof shall use OAuth redirects and server-held credentials; CraftSky shall never collect or expose a PDS password/code/token in Flutter. | Preserves the BFF security boundary. | User decision / architecture | AC-035, AC-040 |
| FR-020 | Functional | Must | Acceptance shall create or resume one durable owner-scoped operation and bind its exact fresh OAuth session before revoking ordinary CraftSky sessions. The operation shall survive process/network interruption and duplicate submission. | Prevents a foreground request from owning irreversible work. | User decision | AC-023, AC-037, AC-041 |
| FR-021 | Functional | Must | Acceptance shall revoke/remove ordinary CraftSky bearer sessions and unbound OAuth sessions. Only the job-bound server OAuth session may remain until completion; no client status/recovery credential shall be created. | Keeps the account unusable without a second credential system. | User decision | AC-016, AC-040, AC-042 |
| FR-022 | Functional | Must | The confirming device shall clear drafts, staged media, caches, session, and account state after acceptance. Other devices shall clear through normal invalid-session handling on next contact. | Reuses established session invalidation. | User decision | AC-036, AC-043 |
| FR-023 | Functional | Must | Existing Tap/indexers shall process PDS delete events independently. The deletion worker shall not register expected URIs, store receipts, inspect indexed rows, block Tap acknowledgement, or wait for AppView convergence. | Removes the largest non-essential subsystem. | User decision | AC-044 |
| FR-024 | Functional | Must | Authentication for a DID with an active deletion operation shall not mint an ordinary CraftSky session and shall return a coarse deletion-in-progress outcome. After the operation is gone, the DID may join as a fresh membership. | Prevents recreation during deletion without status infrastructure. | User decision | AC-045 |
| FR-025 | Functional | Must | Terminal success shall remove the operation and deletion-bound OAuth session. CraftSky shall not retain a deletion audit; ordinary redacted infrastructure logs remain subject to existing retention. | Avoids a deletion-specific retained subsystem. | User decision | AC-046 |
| FR-026 | Functional | Must | Explicit deletion shall hard-delete account-owned Instagram migration data and release username claims, overriding ordinary migration retention. | Private deletion must include imported data. | User decision / codebase | AC-047 |
| FR-027 | Functional | Must | The user-visible flow shall expose only pre-acceptance validation and a successful acceptance acknowledgement; it shall not expose worker phases, counts, status polling, attention, or Retry. | Defines the simplified UX. | User decision | AC-038, AC-039 |
| NFR-001 | Non-functional | Must | Identity and destructive client actions shall remain fenced to the captured active-account lease. | Prevents deleting the wrong retained account. | Codebase | AC-021, AC-025 |
| NFR-002 | Non-functional | Must | The owner-scoped operation, private cleanup, collection scanning, and finalization shall be idempotent or safely convergent across worker restarts and partial side effects. | Makes asynchronous deletion safe. | Architecture | AC-017, AC-023, AC-037, AC-041 |
| NFR-003 | Non-functional | Must | Changed controls shall preserve semantics, accessibility, focus, and minimum target sizes. | Maintains UI quality. | Codebase | AC-022, AC-027 |
| NFR-004 | Non-functional | Must | Visible Settings and deletion copy shall be localizable. | Maintains product conventions. | Codebase | AC-028 |
| NFR-005 | Non-functional | Should | Settings child pages should preserve CraftSky theming and responsive layout. | Maintains visual coherence. | Codebase | AC-029 |
| NFR-006 | Non-functional | Must | The operation and logs shall contain only identifiers, timestamps, retry scheduling, and coarse error categories—never record content, handles, tokens, or unrelated account data. | Prevents operational data becoming a secondary private store. | User decision | AC-040, AC-046 |
| RULE-001 | Business rule | Must | Deletion applies only to the account lease captured at confirmation. | Prevents cross-account deletion. | Codebase | AC-025 |
| RULE-002 | Business rule | Must | Deletion shall never call an AT account-deletion API or delete outside registered `social.craftsky.*` record collections. | Preserves federated ownership. | User decision | AC-015, AC-017 |
| RULE-003 | Business rule | Must | Chevrons denote in-app disclosure; external links use an external-link icon; actions/read-only rows use neither. | Keeps affordances accurate. | Prompt | AC-003, AC-033 |
| RULE-004 | Business rule | Must | Sign out and Delete account remain distinct reversible/destructive actions. | Prevents accidental scope expansion. | Prompt | AC-014, AC-024 |
| RULE-005 | Business rule | Must | The deletion collection inventory shall exactly cover every primary PDS record Lexicon under `social.craftsky.*`, with membership last. | Prevents collection drift. | Architecture | AC-015, AC-030 |
| RULE-006 | Business rule | Must | Exact-handle submission is the non-cancelable point of no return. | Work may become irreversible immediately. | User decision | AC-013, AC-039 |
| RULE-007 | Business rule | Must | Terminal success requires private cleanup, ordinary-session removal, a final empty scan of registered CraftSky PDS collections, and deletion-bound OAuth removal; indexer convergence and blob GC are not gates. | Defines the lean completion boundary. | User decision | AC-037, AC-044, AC-048 |
| RULE-008 | Business rule | Must | Temporary public AppView staleness is acceptable and no eager-hide/purge layer shall be added. | Leaves public projection ownership with indexers. | User decision | AC-044 |
| RULE-009 | Business rule | Must | CraftSky deletes record references only and never directly deletes or waits for physical PDS blob garbage collection. | Blob lifecycle belongs to the PDS. | User decision | AC-036, AC-048 |
| RULE-010 | Business rule | Must | Only the minimal active operation and one bound OAuth session may survive acceptance; both are removed on success and no deletion audit/checkpoint/receipt/status state remains. | Enforces the approved simplification and minimization. | User decision | AC-042, AC-046 |
| RULE-011 | Business rule | Must | The same DID may rejoin only after no active deletion operation remains, as a fresh membership with no data restoration. | Separates deletion from re-enrolment. | User decision | AC-045 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Settings shows the captured active avatar/name/handle above rows. |
| AC-002 | FR-002 | Switch account is immediately below identity and opens the shared switcher. |
| AC-003 | FR-003, RULE-003 | In-app disclosure rows alone show chevrons. |
| AC-004 | FR-004 | The complete approved Settings inventory renders in order. |
| AC-005 | BR-002, BR-004, FR-004, FR-005 | Every destination opens through its canonical typed route and Back behaves normally. |
| AC-006 | FR-006 | Notifications opens the existing page unchanged. |
| AC-007 | FR-007, FR-010 | About shows legal links, cache action, and version; cache is absent from top-level Settings. |
| AC-008 | FR-008, FR-009 | Legal rows launch the exact external HTTPS URLs and fail safely. |
| AC-009 | FR-010 | Cache clearing preserves both-cache busy/success/error behavior. |
| AC-010 | FR-011 | About version matches the shared formatter. |
| AC-011 | FR-012 | Account initially contains only Delete account. |
| AC-012 | FR-013, FR-014 | The warning names the captured handle and states the complete permanent asynchronous boundary. |
| AC-013 | FR-013, RULE-006 | Acceptance is disabled until exact handle typing; cancellation before submission mutates nothing. |
| AC-014 | FR-018, RULE-004 | Sign out remains immediate/non-destructive and uses error colour. |
| AC-015 | BR-003, FR-015, RULE-002, RULE-005 | Deletion empties every registered CraftSky collection for the owner while preserving other namespaces and the AT/PDS account. |
| AC-016 | BR-003, FR-015, FR-021 | Private CraftSky/Instagram data and ordinary sessions are removed while the one worker OAuth session remains only until finalization. |
| AC-017 | BR-003, FR-015, NFR-002, RULE-002 | Partial/repeated deletion converges without touching another owner, namespace, or blob. |
| AC-018 | FR-016 | After 202 acceptance the client removes the account, activates MRU fallback or Sign in, and shows no deletion row/status page. |
| AC-019 | FR-008–FR-010 | External/cache action failures use existing safe feedback. |
| AC-020 | FR-011 | Version/build formatting is shared. |
| AC-021 | NFR-001 | Stale identity work never replaces the active account's Settings identity. |
| AC-022 | NFR-003 | New controls expose correct semantics and targets. |
| AC-023 | FR-017, FR-020, NFR-002 | An accepted job survives restart/duplicate submission and retries transient failure automatically with capped backoff. |
| AC-024 | FR-018, RULE-004 | Sign out never deletes membership/data. |
| AC-025 | FR-013, NFR-001, RULE-001 | Reauth/confirmation for one lease cannot delete a later active account. |
| AC-026 | NFR-002 | API errors use existing `/v1/*` envelopes. |
| AC-027 | NFR-003 | Layout remains accessible across supported sizes/text scales. |
| AC-028 | NFR-004 | Visible copy comes from localization resources. |
| AC-029 | NFR-005 | Pages preserve CraftSky responsive theming. |
| AC-030 | RULE-005 | A drift test fails when a record Lexicon and compiled collection inventory differ. |
| AC-031 | FR-004 | No requested Settings destination is lost. |
| AC-032 | BR-002 | Existing destinations preserve behavior. |
| AC-033 | RULE-003 | External/actions/read-only rows use the correct non-chevron affordance. |
| AC-034 | FR-011 | Version remains read-only. |
| AC-035 | FR-013, FR-019 | Fresh OAuth precedes acceptance without collecting PDS credentials. |
| AC-036 | FR-014, FR-022, RULE-009 | Copy and cleanup avoid promising immediate offline-device/blob/indexer erasure. |
| AC-037 | FR-015, FR-020, NFR-002, RULE-007 | Restarted work reaches success only after private cleanup and an empty final PDS scan, then removes OAuth/job state. |
| AC-038 | FR-016, FR-027 | No status registry, deleting switcher row, progress screen, or polling remains. |
| AC-039 | FR-017, FR-027, RULE-006 | No cancel/manual Retry exists after acceptance; automatic retry and redacted operational failure handling remain. |
| AC-040 | FR-019, FR-021, NFR-006 | OAuth authority remains server-only and deletion state/logs expose no sensitive values. |
| AC-041 | FR-020, NFR-002 | Job/OAuth binding is durable before ordinary-session revocation. |
| AC-042 | FR-021, RULE-010 | No status/recovery credential is created; only the operation and bound OAuth session remain. |
| AC-043 | FR-016, FR-022 | Confirming-device cleanup is immediate; other devices use normal invalid-session cleanup. |
| AC-044 | FR-023, RULE-007, RULE-008 | Terminal success does not inspect or wait for AppView/Tap; existing indexers still process deletion events independently. |
| AC-045 | FR-024, RULE-011 | Pending login mints no ordinary session; rejoin after operation removal is fresh. |
| AC-046 | FR-025, NFR-006, RULE-010 | Terminal finalization leaves no operation, audit, status, receipt, expected-record, or checkpoint state. |
| AC-047 | FR-026 | Account-owned Instagram migration/private data and claims are hard-deleted. |
| AC-048 | RULE-007, RULE-009 | Blob deletion/GC is neither invoked nor a completion gate. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Display name absent | Show normalized handle once. | FR-001 |
| EC-002 | External launch fails | Stay on About and show safe feedback. | FR-008, FR-009 |
| EC-003 | Confirmation handle mismatch | Keep acceptance disabled. | FR-013 |
| EC-004 | Active account changes during reauth | Reject stale completion without mutation. | NFR-001, RULE-001 |
| EC-005 | PDS record already missing | Treat delete as successful and continue. | FR-015, NFR-002 |
| EC-006 | PDS contains more than one page | Repeated scans/batches converge to empty. | FR-015, NFR-002 |
| EC-007 | Worker stops after a side effect | Restart replays idempotent cleanup and scanning. | FR-017, FR-020 |
| EC-008 | AppView delete events lag | Job may complete; indexers catch up independently. | FR-023, RULE-008 |
| EC-009 | Another device uses a revoked session | Existing invalid-session behavior signs it out and clears local account state. | FR-021, FR-022 |
| EC-010 | Same DID logs in while deletion exists | Do not mint ordinary access; report coarse deletion-in-progress. | FR-024 |
| EC-011 | Bound OAuth becomes unusable | Keep the minimal job, retry/log safely, and permit a later fresh sign-in to refresh authority without restoring membership. | FR-017, FR-024 |
| EC-012 | New CraftSky record Lexicon is added | Registry drift test fails until included. | RULE-005 |

## 15. Data / Persistence Impact

- Retain OAuth request purpose/owner/job metadata for atomic fresh-reauth binding.
- Replace the eight-table deletion schema with one `account_deletion_operations` table containing owner/job identity, intent/active state, bound OAuth session, minimized confirmation proof hashes, retry scheduling, lease fields, timestamps, and coarse error category.
- Remove status/recovery credential, expected-record, index-receipt, cleanup-step, cleanup-artifact, and audit tables.
- Terminal finalization deletes the operation rather than inserting an audit.
- Migration remains `000037` because the branch is not deployed and the app has no users.

## 16. UI / API / Background Impact

- Keep Settings, About, Account, fresh-reauth completion, and exact-handle confirmation.
- Remove deletion status/pending-status routes, account-switcher deletion rows, secure status registry, status polling, manual Retry, and status-specific 401 recovery.
- Acceptance returns `202` with no client status capability. The client immediately reuses ordinary account removal/local cleanup behavior.
- Remove status/Retry/recovery routes. Pending ordinary login returns a coarse standard error/outcome and no bearer.
- Keep one background worker with capped automatic retry and a small phase/state surface internal to the server.

## 17. Security / Privacy / Permissions

- Fresh OAuth, captured lease, exact-handle confirmation, owner DID validation, namespace allowlist, and server-only PDS authority remain mandatory.
- No generic PDS delete API, Flutter PDS token, direct blob deletion, or other-namespace operation is permitted.
- No status capability exists. Ordinary sessions are revoked at acceptance and cannot authorize pending deletion.
- Logs contain job/phase/category only, never DID, handle, token, record URI/content, or private settings.

## 18. Observability

- Use existing structured logging/error monitoring for accepted, retrying, failed, and completed jobs.
- Do not add deletion-specific metrics, audit retention, convergence timing, or an audit sweeper.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | AppView events lag after job completion. | Other users may briefly see stale projections. | Explicitly accept eventual indexer convergence; do not claim immediate projection erasure. |
| RISK-002 | A future private store is omitted because the executable manifest is removed. | Private data could survive. | Keep cleanup code explicit, review schema changes, and retain focused owner-isolation integration fixtures for current stores. |
| RISK-003 | Persistent OAuth/PDS failure has no user Retry UI. | Job may remain pending until operator action or later sign-in. | Capped automatic retry, redacted monitoring, and pending-login OAuth refresh. |
| RISK-004 | Replaying uncheckpointed external cleanup fails partway. | More repeated calls/work. | Require idempotent cleanup adapters and repeat the whole cleanup safely. |
| RISK-005 | Simplification accidentally weakens PDS scope. | Irreversible unrelated deletion. | Preserve typed owner/NSID parsing, exact collection registry, final scan, and cross-owner/namespace/blob tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Existing Tap/indexers reliably and idempotently process PDS delete events eventually. | A separate projection repair tool may be needed, but deletion completion remains PDS/private scoped. |
| ASM-002 | Private cleanup adapters can be called repeatedly. | A minimal idempotency fix is required before removing checkpoints. |
| ASM-003 | Ordinary invalid-session handling removes another device's local account state sufficiently. | A separate local cleanup hook may be required, but not a deletion status system. |
| ASM-004 | The app remains pre-production with no active users, so rewriting migration `000037` is safe. | A follow-up migration would be required if deployment occurred. |

## 21. Open Questions

None identified.

## 22. Review Status

Status: Approved
Risk level: High
Review recommended: Required
Reviewer: Product owner
Date: 2026-08-11
Notes: The product owner explicitly approved the lean asynchronous/no-status contract and authorized implementation after reviewing its tradeoffs.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover: FR-013–FR-027, NFR-001, NFR-002, NFR-006, RULE-001, RULE-002, RULE-005–RULE-011 plus unchanged Settings requirements.
- Suggested levels: Flutter widget/controller tests; AppView migration/store/worker/PDS/private-cleanup integration tests; auth/route regression tests; no receipt/status/audit tests.
- Blocking questions: None.
