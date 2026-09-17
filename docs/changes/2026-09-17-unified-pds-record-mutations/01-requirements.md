# Requirements: Tap-Authoritative PDS Data Lifecycle

## 1. Initial Request
Reconsider the complete Flutter App -> AppView -> PDS -> Tap -> AppView lifecycle before production. Use Tap's per-repository ordering and the PDS as the public source of truth to reduce mutation complexity, improve behavior when other clients write to a user's PDS, and make both AppView and Flutter robust during response loss, Tap lag, replay, resync, and conflicting external writes. A large rewrite is acceptable when it produces a simpler and more federated design.

## 2. Current Codebase Findings
- `appview/internal/tap/consumer.go` durably ingests or quarantines each Tap frame before acknowledging it. Unacknowledged frames are retried.
- `appview/internal/ingestion/store.go` retains the latest source version per URI using the repository revision, detects stale, duplicate, and conflicting versions, and queues projection work.
- Tap delivery is at least once. Live events are ordered barriers within one repository, historical backfill/resync events may be concurrent, and there is no ordering across repositories.
- `tap_projection_jobs` are claimed independently of repository order, so projection correctness already depends on revision fences and idempotence rather than callback order.
- `appview/internal/ingestion/store.go` currently lets Craftsky-originated effect matching influence projection eligibility. This couples command reliability to public-state interpretation.
- `appview/internal/index/bluesky_follow.go` collapses duplicate owner/subject follows by deleting prior projection rows.
- `appview/internal/index/craftsky_interaction.go` marks prior matching likes or reposts deleted when another source URI appears. Deleting the selected source can therefore report no interaction while an older record still exists on the PDS.
- Immediate post, event, like, and repost effect identities are derived from one HTTP request. A retry in a new HTTP request can select a new TID before Tap exposes the first write.
- Follow and block changes in this worktree add feature-specific durable intents. Scheduled publication has a separate durable identity and recovery workflow.
- Profile updates currently write the Bluesky and Craftsky `self` records separately and can return partial success.
- Flutter has multiple feature-specific optimistic and projection-overlay implementations. Scheduled posts already retain a stable client operation UUID; immediate post and event flows do not yet share one in-memory operation lifecycle.
- `com.atproto.repo.applyWrites` supports atomic same-repository create, update, and delete batches guarded by `swapCommit`.
- Ordinary Flutter writes must remain AppView-mediated. OAuth credentials and reusable PDS credentials remain server-side.
- Existing release gates are `just test` and `just appview-check`.

## 3. Clarifying Questions And Decisions
### Q1: What owns public truth?
Answer: The PDS owns public records, and Tap-derived source state owns AppView's interpretation of those records.
Decision / implication: A valid PDS record is considered without regard to which client or AppView created it. Craftsky command state must not decide whether public data exists.

### Q2: How should Tap ordering be used?
Answer: Use per-repository revisions as monotonic safety fences, but do not require indexer callbacks to execute in strict order.
Decision / implication: Projectors are state-based, idempotent, and rebuildable from retained current source state. They tolerate duplicate, stale, and historically reordered events. No cross-repository ordering is assumed.

### Q3: What write-side state remains necessary?
Answer: Retain a small durable command journal for request identity, authorization and lifecycle fences, selected record identities, CAS inputs, PDS dispatch outcomes, and idempotent response replay or ambiguous-outcome recovery through the original mutation endpoint.
Decision / implication: The journal solves response loss and retry ambiguity but is not a second public-state model.

### Q4: How are records from other clients handled?
Answer: Every semantically valid current PDS record participates equally in source facts and projections. Invalid records remain source evidence but do not affect product state.
Decision / implication: Projection behavior cannot depend on a matching Craftsky command or effect record.

### Q5: What does a set-like relationship or interaction mean when duplicates exist?
Answer: Follow, block, like, or repost is active while at least one valid matching source record exists.
Decision / implication: Retain every source URI. A deterministic representative may be selected for response metadata, but no winner replaces aggregate truth. Notifications follow aggregate `0 -> 1` and `1 -> 0` transitions.

### Q6: How does an explicit remove action clear duplicate set-like records?
Answer: AppView reads the authoritative PDS state, identifies every valid matching record present at that snapshot, and deletes them in an atomic `applyWrites` transaction guarded by the repository head when supported.
Decision / implication: An `InvalidSwap` causes bounded reread/retry. A genuinely later external create remains active. Indexers do not autonomously delete PDS records merely because they are duplicates.

### Q7: How are compound same-repository writes handled?
Answer: Use `applyWrites` when a user action must change multiple records in one repository, including the two profile records.
Decision / implication: Craftsky no longer intentionally exposes partial profile success for a newly initiated update.

### Q8: How are client retries identified?
Answer: Every ordinary Flutter-initiated public mutation carries one canonical UUID operation key. Within its authenticated owner and operation-kind scope, the key identifies one immutable command and is reused for unchanged retries while that operation remains in memory.
Decision / implication: Full successful responses remain replayable for 24 hours; a minimal owner/operation-kind-scoped key-hash tombstone remains until account purge so an old scoped key never becomes a new command. The same UUID does not couple unrelated owners or operation kinds.

### Q9: How does Flutter bridge accepted writes and Tap-derived reads?
Answer: Treat definite PDS acceptance as mutation success and use one shared local operation controller and optimistic overlay model rather than separate feature-specific retry state machines.
Decision / implication: Flutter retains operation identity, immutable dispatched input, and overlays only in memory. After definite acceptance, Flutter presents the accepted result optimistically and retires the overlay when an ordinary Tap-derived read agrees with it. A bounded overlay lifetime followed by a forced refresh prevents stale optimistic state from remaining indefinitely. For an ambiguous outcome, Flutter retries the original mutation endpoint with the same operation UUID and immutable input while the operation controller remains alive; it does not poll a separate status resource. App restart, process death, sign-out, or account switch discards this local state rather than restoring it.

### Q10: Which lifecycle state remains local?
Answer: Authentication, authorization, owner generations, deletion-pending/terminal policy, private data, and permanent account deletion remain durable AppView concerns.
Decision / implication: A public profile may provide PDS-derived membership evidence, but semantically invalid records cannot change membership and terminal local policy remains fail-closed.

### Q11: What remains outside the rewrite?
Answer: Private drafts, mutes, push tokens, scheduled-post domain scheduling, and permanent account deletion retain their current ownership boundaries.
Decision / implication: Scheduled publication adapts to the command contract without duplicating its publication state. Permanent account deletion remains the narrow documented exception for bulk PDS removal.

### Q12: How is protocol correctness verified?
Answer: Use authoritative Lexicon-driven in-process validation and real-PostgreSQL race-enabled tests in the release gate.
Decision / implication: A local PDS smoke suite is not required. Validator drift remains an accepted, monitored residual risk.

## 4. Candidate Approaches
### Option A: Tap-Only Mutation Lifecycle
Summary: Remove durable command state and wait for Tap to reveal every PDS write.
Pros:
- One public source of truth.
- Minimal write-side persistence.
Cons:
- Cannot distinguish a retry from a new append/create action after a lost HTTP response.
- Cannot provide reliable mutation status during Tap outages.
- Cannot replace pre-dispatch authorization, CAS, or terminal lifecycle fencing.
Risks:
- Blind retries create duplicates and ambiguous failures produce poor Flutter behavior.

### Option B: Intent-First Public State
Summary: Preserve feature-specific intents as the controlling lifecycle for follows, blocks, likes, reposts, and other mutations.
Pros:
- Strong immediate behavior for Craftsky-originated writes.
- Existing follow/block work can be extended.
Cons:
- Creates a second model of public truth.
- Requires generations, duplicate cleanup, and origin matching that do not naturally represent external clients.
- Keeps multiple Flutter mutation state machines.
Risks:
- Valid PDS data may be hidden, deleted, or misrepresented because Craftsky did not originate it.

### Option C: Tap-Authoritative Source Facts With A Command Journal
Summary: Retain every current PDS source, validate and project it independently of origin, derive semantic aggregates from source facts, and keep only the command state required for reliable API and Flutter behavior.
Pros:
- Aligns with federation and treats external clients as normal.
- Makes projections rebuildable and robust to duplicate and reordered delivery.
- Removes winner generations and autonomous duplicate cleanup.
- Unifies Flutter retry and optimistic behavior.
- Enables atomic same-repository changes through `applyWrites`.
Cons:
- Requires a cross-cutting AppView schema/indexer rewrite and Flutter state refactor.
- Stores more source-level rows for set-like records.
- Requires idempotent ambiguous-outcome recovery on every mutation endpoint and shared Flutter overlay reconciliation against ordinary reads.
Risks:
- Incorrect source aggregation or overlay retirement could temporarily misstate user-visible data.

## 5. Recommended Direction
Recommended approach: Option C, Tap-authoritative source facts with a minimal durable command journal and a shared Flutter operation controller.

Why: This separates three concerns that are currently intertwined: PDS public truth, API command reliability, and client experience during eventual consistency. It handles valid writes from any client, removes feature-specific intent generations, preserves safe response-loss behavior while the in-memory operation survives, and uses PDS transaction primitives rather than reconstructing transactionality in AppView. The app has no production compatibility obligations, making this the lowest-risk point for the rewrite.

## 6. Problem / Opportunity
Craftsky's current mutation reliability is strongest when AppView originated and can recognize a record. Federation means that assumption is incomplete: another authorized client can create, update, duplicate, or delete valid records directly in the same PDS repository. Current duplicate-collapsing projectors can then report state that differs from the PDS. Meanwhile, feature-specific mutation intents and Flutter overlays create substantial lifecycle complexity.

The opportunity is to make current PDS source state independently rebuildable, derive user-facing semantics from all valid records, and constrain durable command state to the problems Tap cannot solve: idempotent client requests, ambiguous network outcomes, CAS, authorization, and local lifecycle policy.

## 7. Goals
- G-001: Retain every current PDS record as authoritative source evidence and make public serving state a faithful semantic projection of every valid, locally eligible source, regardless of origin.
- G-002: Make every ordinary Flutter public mutation safely retryable with the same in-memory operation identity during response loss and Tap delay, without requiring recovery across app restarts, process death, sign-out, or account switches.
- G-003: Make projectors deterministic, state-based, idempotent, and rebuildable without command history.
- G-004: Represent duplicate set-like records accurately and remove them only through explicit authorized user commands.
- G-005: Use atomic PDS repository transactions for compound same-repository user actions where supported.
- G-006: Replace feature-specific Flutter mutation lifecycles with one account-fenced operation and overlay model.
- G-007: Preserve security, private-data ownership, and permanent account-deletion boundaries.

## 8. Non-Goals
- NG-001: Changing Craftsky or Bluesky Lexicon record shapes.
- NG-002: Allowing ordinary Flutter writes directly to a PDS or exposing reusable PDS credentials to Flutter.
- NG-003: Writing public projection rows directly from an API mutation response.
- NG-004: Assuming global ordering across repositories or atomic projection of every multi-record PDS commit.
- NG-005: Deduplicating distinct posts or business events based on equal content.
- NG-006: Automatically deleting PDS records from an indexer because they are duplicate, malformed, old, or non-representative.
- NG-007: Moving drafts, mutes, scheduled-publication domain state, push tokens, moderation state, or other private data onto the PDS.
- NG-008: Replacing the owner lifecycle fence, terminal-owner policy, OAuth TMB boundary, or permanent account-deletion job.
- NG-009: Providing compatibility for unreleased database schemas or Flutter clients.
- NG-010: Requiring a local PDS process in the normal release gate.
- NG-011: Persisting ordinary Flutter mutation operations or optimistic overlays across restart, process death, sign-out, account removal, or account switch.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Craftsky member | Uses Craftsky and may also use other authorized AT Protocol clients | Complete source handling, policy-consistent serving state, and mutations that do not duplicate after retries |
| Flutter client | Initiates AppView-mediated commands and displays eventual projections | One operation model, clear accepted/ambiguous/failed states, optimistic accepted results, and safe account switching |
| AppView API | Authorizes commands and writes to the PDS | Minimal durable command state, atomic writes, CAS, and deterministic recovery |
| Tap ingestion service | Delivers verified repository events at least once | Durable acknowledgement and revision-safe source installation |
| AppView projector | Validates and derives read models from current source state | Idempotent source facts, aggregate semantics, and dependency retries |
| External client/AppView | May validly change the same user's PDS records | Equal treatment of valid records without requiring Craftsky command metadata |
| Operator | Diagnoses delayed, invalid, ambiguous, or divergent state | Bounded telemetry and authoritative repair paths |

## 10. Current Behavior
Tap ingestion already retains one latest source version per URI and durably queues projection work, but public projection eligibility may depend on lifecycle/effect-origin matching. Feature indexers generally consume event deltas directly. Follow and interaction projectors collapse multiple logical duplicates by deleting or soft-deleting earlier projection rows, which loses actual PDS-presence information.

AppView mutation endpoints use several identity models: request-local effects for immediate writes, relationship-specific intents in the current worktree, scheduled-publication state, and URI/CID addressing for some updates and deletes. Flutter similarly has feature-specific optimistic providers and overlays. Profiles are written as two independent effects.

## 11. Desired Behavior
Tap ingestion first stores the latest observed version of every relevant URI by repository revision. Semantic processing validates the current version and materializes one source fact per structurally and semantically valid PDS record. Projection eligibility is a separate state that accounts for unresolved cross-repository dependencies and existing local membership/terminal policy. Invalid current versions remain diagnosable source evidence but contribute no public fact. Projectors derive serving rows and logical aggregates from the complete current eligible source-fact set, not from command origin or event callback order.

Set-like state is active when at least one matching valid, projection-eligible fact exists. Every source URI remains represented. An explicit remove command authoritatively reads the PDS, atomically deletes all matching records present at that repository snapshot with `applyWrites` and `swapCommit`, and retries after a bounded conflict reread. A later external create is a newer valid action and becomes active when projection-eligible.

Every ordinary Flutter public mutation has one operation UUID and immutable request fingerprint. AppView durably records the command before dispatch, returns definite PDS acceptance without waiting for Tap, and reconciles ambiguous dispatch. Definite acceptance is terminal command success; Tap source observation and serving-projection convergence proceed independently. Flutter uses one account-fenced, in-memory operation controller to retain retry identity while the process and account session remain active, retry the original mutation endpoint with the same operation UUID and immutable input after an ambiguous response, and apply ordered logical-scope overlays after definite acceptance. AppView replays the accepted or rejected result when known, or returns `202` with retry guidance while reconciliation remains incomplete. An accepted overlay retires when an ordinary read agrees with it, after a bounded lifetime and forced refresh, or when the in-memory controller is reset by restart, process death, sign-out, or account switch. No device persistence, separate operation-status resource, or projection-convergence polling is required. A later retry after local state is discarded is a new operation and may duplicate an append create whose earlier response was lost.

The ordinary public mutation matrix is:

| Mutation class | Flutter operation key | AppView/PDS behavior |
|---|---|---|
| Immediate post create and delete | Required | Append create or addressed delete through the command journal |
| Business-event create, update, and delete | Required | Append create or addressed CID-guarded mutation through the command journal |
| Follow/unfollow, block/unblock, like/unlike, repost/unrepost | Required | Authoritative set read followed by guarded no-op, create, or delete transaction |
| Instagram suggestion acceptance that creates a follow | Required | Uses the same authoritative set-like follow command rather than a separate mutation lifecycle |
| Personal profile create/update | Required | Guarded compound write of the applicable Bluesky and Craftsky fixed-key profile records |
| Business-profile create/update/delete | Required | Addressed fixed-key business-profile command with guarded update/delete semantics |
| Scheduled final publication | Existing scheduled operation identity | Worker adapts the frozen publication to the command journal; schedule CRUD remains private state |
| Image upload and direct video blob upload | Excluded | Existing AppView image and purpose-bound video upload paths remain unchanged; final post creation is included |
| Mutes, drafts, profile customisation/account type, saves/pins, schedule editing, push tokens, moderation state | Excluded | Private PostgreSQL/client workflows remain outside the public command contract |
| Permanent account deletion | Excluded | Restricted terminal deletion job remains separate |

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | AppView shall retain every current PDS record within Craftsky's indexed collections as source evidence and shall reflect every semantically valid, locally eligible source in public serving state, regardless of which authorized client created it. | Federation makes the PDS authoritative while existing membership and terminal policy still control Craftsky serving eligibility. | User request and confirmed direction | AC-001, AC-002, AC-034 |
| BR-002 | Business | Must | One Flutter mutation operation shall not create a second incompatible PDS effect when its response is lost or Tap is delayed, provided Flutter still retains that operation in memory and retries with the same key and immutable request. Cross-restart and cross-account recovery are not required. | In-session reliable retries require identity outside Tap without adding device persistence. | Confirmed direction and user clarification | AC-003, AC-016, AC-021 |
| BR-003 | Business | Must | Flutter shall display a coherent accepted, ambiguous, or failed mutation state without treating an unchanged Tap-derived read as a failed PDS write. | Definite PDS acceptance is sufficient for optimistic success while ordinary reads remain eventually consistent. | User request and confirmed direction | AC-004, AC-035, AC-036 |
| BR-004 | Business | Must | The public data lifecycle shall remain correct under duplicate delivery, historical reordering, external writes, projection retries, and repository resynchronization. | These are normal federated-system conditions. | Tap contract and user request | AC-005, AC-006, AC-010, AC-045 |
| FR-001 | Functional | Must | Durable ingestion shall retain the latest observed source version for every relevant URI using its repository revision, action, CID, record, owner, collection, and record key. | Current source state is the rebuildable public-data foundation. | Recommended direction | AC-001, AC-006 |
| FR-002 | Functional | Must | Ingestion and projection shall tolerate at-least-once delivery, stale versions, duplicate versions, and arbitrary ordering among historical events without requiring globally ordered processing. | Tap's guarantee is per repository and historical delivery may be concurrent. | Tap contract | AC-005, AC-006 |
| FR-003 | Functional | Must | The current version of each source shall be validated against authoritative Lexicon shape, record-key policy, identifier syntax, and collection-specific semantic rules before it contributes a source fact. Structural/semantic validity and projection eligibility shall be represented separately. | Arbitrary PDSes may store unknown records, while valid records may still await dependencies or fail local serving policy. | Confirmed direction | AC-002, AC-007, AC-008, AC-015 |
| FR-004 | Functional | Must | If the latest version of a previously valid URI becomes semantically invalid, its prior source fact shall stop contributing to public state; a later valid version may restore it. | The projection must represent current PDS content, not the last valid historical content. | Consequence of source authority | AC-009 |
| FR-005 | Functional | Must | A valid source's projection eligibility shall not depend on a matching Craftsky command, effect attempt, idempotency key, or origin classification. Existing collection-specific membership and terminal policy shall be applied deterministically and identically to all origins. | External writes are first-class public data without bypassing local serving policy. | Confirmed direction | AC-001, AC-023, AC-034 |
| FR-006 | Functional | Must | Projectors shall derive output from retained current source state and be capable of rebuilding serving projections without command-journal history. | State-based projection is robust to retries, reordering, and command cleanup. | Recommended direction | AC-010, AC-045 |
| FR-007 | Functional | Must | Follows, blocks, likes, and reposts shall retain one source fact per semantically valid PDS URI rather than deleting, soft-deleting, or overwriting another PDS-present source as a duplicate. | Every valid source must remain representable. | Codebase finding and confirmed direction | AC-011 |
| FR-008 | Functional | Must | Creating, updating, invalidating, retargeting, or deleting a set-like source shall transactionally recompute every affected old and new logical scope. | Updates can move a source between aggregates. | State-based projection design | AC-012, AC-041 |
| FR-009 | Functional | Must | A set-like logical scope shall be active when at least one valid, projection-eligible current source fact matches it; any deterministic representative URI shall be metadata only and shall not define activity. | Aggregate truth handles valid duplicates while excluding unresolved or policy-ineligible facts. | Confirmed direction | AC-013, AC-015 |
| FR-010 | Functional | Must | Set-like notifications and other edge-triggered side effects shall activate only on eligible aggregate transition `0 -> 1` and retract only on `1 -> 0`. Representative churn during `1 -> N` or `N -> 1` shall not reset activity time, newness, or delivery state. | Source churn must not duplicate, resurface, or prematurely retract logical notifications. | Confirmed direction | AC-014, AC-050 |
| FR-011 | Functional | Must | A source whose projection dependency is not yet available shall remain durably blocked and be retried when that dependency changes, without blocking Tap acknowledgement. | There is no cross-repository ordering guarantee. | Tap contract | AC-015 |
| FR-012 | Functional | Must | Before dispatching an ordinary public mutation, AppView shall durably prepare one command containing the owner, operation key, immutable request fingerprint, operation kind, selected URI/record key where applicable, CAS or repository-head inputs, owner/target lifecycle fences, and command steps. | A crash must not erase mutation identity or safety inputs. | Recommended direction | AC-003, AC-016 |
| FR-013 | Functional | Must | Every Flutter-initiated mutation listed as requiring a key in the mutation matrix shall provide a canonical UUID in `Idempotency-Key`; unchanged retries shall reuse it while the operation remains in memory, and intentional new commands shall use a new key. Command identity is scoped by authenticated owner, operation kind, and key. Restart, process death, sign-out, and account switch may discard the key. | One enumerable client contract supports in-session idempotent replay and recovery without accidental cross-owner coupling or device persistence. | Confirmed direction and user clarification | AC-003, AC-017 |
| FR-014 | Functional | Must | Within one authenticated owner and operation kind, reuse of an operation key with a different lifecycle generation or immutable request fingerprint shall fail before another PDS write. The same UUID used by another owner or operation kind is a distinct scoped identity. | A stable scoped key cannot alias incompatible intent, while unrelated accounts remain isolated. | Confirmed direction | AC-018 |
| FR-015 | Functional | Must | A terminal command's full response shall remain replayable for 24 hours, after which a minimal owner/operation-kind-scoped key-hash tombstone shall reject reuse until account purge. | Late retries must not silently become new commands. | Prior confirmed decision retained | AC-019, AC-046 |
| FR-016 | Functional | Must | Commands shall retain durable prepared, dispatching, accepted, ambiguous, and rejected outcomes. A retry of the original authenticated mutation endpoint with the same operation key and immutable request shall replay accepted or rejected, resume safe reconciliation, or return `202` with retry guidance while still ambiguous. No separate operation-status API shall be required. Accepted and rejected are terminal command outcomes; source observation, projection convergence, and later external writes are not command lifecycle states. | One idempotent mutation path is sufficient for response replay and ambiguous recovery, while asynchronous projection progress belongs to the Tap-derived read path. | Recommended direction and user clarification | AC-020, AC-024 |
| FR-017 | Functional | Must | An ambiguous PDS outcome shall be reconciled by authoritative reads of the exact URI, expected content/CID, command steps, or repository state before redispatch; AppView shall not blindly repeat it. | Tap may be delayed and network failure does not prove rejection. | Recommended direction | AC-021 |
| FR-018 | Functional | Must | A definite PDS acceptance shall complete through the mutation endpoint's normal success response without waiting for Tap or requiring later status polling; successful DELETE responses shall preserve the existing `204` empty-body contract. A genuinely ambiguous result shall return `202` with bounded retry guidance such as `Retry-After`; Flutter shall retry the same mutation endpoint with the same operation key and immutable request rather than follow an operation-status reference. | API availability and client success must not depend on Tap latency, and the client already owns the operation key needed for idempotent retries. | Confirmed direction, user clarification, and API architecture | AC-022, AC-035 |
| FR-019 | Functional | Must | Durable source ingestion and serving projection shall proceed independently after command acceptance and shall not advance, supersede, or otherwise alter the terminal accepted command outcome. Command identity may be used for bounded diagnostics, but correlation shall not allow, deny, modify, or remove the public source projection. | PDS acceptance is sufficient for command success, and Tap-derived public truth remains independent of command state. | User clarification | AC-023, AC-024 |
| FR-020 | Functional | Must | Updates and deletes of an addressed record shall use its exact URI/key and expected CID or stronger repository-head CAS; already-matching or already-absent outcomes shall reconcile idempotently. | Existing record identity and version define safe mutation. | Prior decision and recommended direction | AC-021, AC-025 |
| FR-021 | Functional | Must | A user action that changes multiple records in one repository shall use one atomic `applyWrites` transaction guarded by `swapCommit` when the target PDS supports the standard operation. Profile updates shall use this path for the Bluesky and Craftsky `self` records. | The PDS can provide transactionality that AppView should not emulate with partial writes. | Confirmed direction | AC-026 |
| FR-022 | Functional | Must | Removing a set-like logical action shall capture an authoritative repository head, enumerate every semantically valid matching record from a verified repository snapshot or head-guarded complete collection read, and delete that snapshot in one `applyWrites` transaction guarded by the same head. | Pagination or Tap state alone does not prove a complete repository snapshot. | Confirmed direction | AC-027 |
| FR-023 | Functional | Must | If a guarded compound or set-like mutation fails because the repository head changed, AppView shall perform a bounded authoritative reread, rebuild the command steps, and retry without changing the operation identity. | Repository concurrency should be resolved from current truth. | Confirmed direction | AC-028 |
| FR-024 | Functional | Must | A valid record created after a successful set-like removal snapshot shall be treated as a newer active action rather than absorbed into an old delete lifecycle. | Later repository state is authoritative and removes generation complexity. | Confirmed direction | AC-029 |
| FR-025 | Functional | Must | Indexers and reconciliation workers shall not autonomously delete PDS records solely because they are duplicate, malformed, old, invalid, or non-representative. | Public user data must not be normalized destructively without an explicit authorized command. | Architectural rule | AC-030 |
| FR-026 | Functional | Must | Scheduled publication shall retain its existing private schedule, frozen publication version, leases, media state, and recovery behavior while using the shared command identity and outcome contract for its final PDS publication. | Scheduling is a private domain workflow, not a Tap projection. | Codebase finding | AC-031 |
| FR-027 | Functional | Must | Fixed-key records shall use their Lexicon-defined key, participate in the command journal, and reconcile through exact-record reads and CAS without allocating a TID. | Fixed identity differs from append records but still benefits from unified command recovery. | Confirmed direction | AC-032 |
| FR-028 | Functional | Must | Existing authentication, authorization, owner generation, expected-owner/target fencing, deletion-pending policy, terminal-owner policy, and PDS credential boundaries shall be enforced before dispatch and during command recovery. | Public source authority does not replace local security policy. | Architecture constraints | AC-033, AC-042 |
| FR-029 | Functional | Must | A Craftsky profile source shall pass semantic validation before it can activate, depart, or otherwise transition local membership lifecycle; terminal local policy shall continue to override later public writes. | Invalid external records must not change authorization state. | Codebase finding and confirmed direction | AC-002, AC-034 |
| FR-030 | Functional | Must | Flutter shall use one account-fenced, in-memory operation controller for the mutation matrix, retain the operation key and immutable dispatched input while the PDS outcome is unresolved and the controller remains alive, retry an ambiguous command through its original mutation endpoint, and expose accepted, ambiguous, and failed state to feature UI. Definite acceptance ends mutation retries. No ordinary mutation operation or overlay shall require device persistence. | Replaces inconsistent feature-specific retry state without adding persistent client infrastructure, a polling resource, or coupling normal client success to projection progress. | Confirmed direction and user clarification | AC-004, AC-035, AC-037, AC-048 |
| FR-031 | Functional | Must | After definite PDS acceptance, Flutter shall apply a local optimistic overlay and reconcile it against ordinary AppView reads rather than server-side operation or projection status. It shall retire the overlay when an ordinary read agrees with the accepted result. If agreement does not arrive within a bounded lifetime, Flutter shall retire the overlay and force an ordinary refresh so optimistic state cannot mask authoritative state indefinitely. | PDS acceptance proves mutation success; the overlay only bridges eventual read-model lag. | User clarification | AC-035 |
| FR-032 | Functional | Must | While an ambiguous dispatched command remains in memory, Flutter shall freeze edits, reconcile using the same operation key, and rotate the key only after definitive failure plus an intentional changed command. If restart, process death, sign-out, or account switch discards the operation, Flutter shall not restore or automatically resume it. | Editing an active ambiguous operation under the old key can publish two incompatible records, while cross-session recovery is intentionally out of scope. | Prior confirmed decision adapted by user clarification | AC-036, AC-048 |
| FR-033 | Functional | Must | Flutter shall bind in-memory operations and overlays to the active account generation, clear them on restart, process reinitialization, sign-out, account removal, or account switch, and discard late results after sign-out, account switch, or generation change. Switching back shall not restore discarded operations or overlays; authoritative reads remain available and server command retention is unaffected. | Cross-account state leakage is a security and correctness risk, and ordinary mutation state intentionally does not survive controller or account lifetime. | Existing Flutter pattern and user clarification | AC-037, AC-048 |
| FR-034 | Functional | Must | Ordinary AppView reads shall remain derived only from Tap projections; command state may be returned through original or retried mutation responses but shall not be silently merged into public read models. | Preserves one public projection authority. | Confirmed direction | AC-038 |
| FR-035 | Functional | Must | Permanent account deletion shall remain outside the ordinary command/projection model and continue through its restricted, freshly reauthenticated deletion job. | Terminal bulk cleanup has different authority and scope. | Architecture rule | AC-039 |
| FR-036 | Functional | Must | The undeployed migration 73 shall be replaced directly with the final command-journal, source-fact, set-aggregate, and projection schema changes, with no relationship-intent compatibility layer. | There are no production migration obligations. | User instruction and codebase state | AC-040 |
| FR-037 | Functional | Must | Administrative repository repair shall obtain a complete verified repository snapshot at a known head, reconcile retained sources including records absent from that snapshot, and then reconstruct facts, aggregates, notifications, and serving rows without command records. Tap event replay or omission alone shall not be treated as proof of deletion. | Commands may be compacted, external records have no command, and an event stream has no omission evidence. | Recommended direction and existing repair path | AC-010, AC-045 |
| FR-038 | Functional | Must | Command cleanup shall preserve unresolved dispatch and reconciliation state, compact terminal replay payloads separately, and remove command/tombstone state during owner purge. | Cleanup must not reopen retry ambiguity or retain account-scoped data after purge. | Prior confirmed decision adapted | AC-019, AC-046 |
| FR-039 | Functional | Must | Flutter shall assign each operation one or more logical scope keys and an account-local monotonic sequence. When operations overlap a scope, only the newest active operation or accepted overlay controls local presentation; older original or retry responses and ordinary-read reconciliation cannot retire or overwrite a newer overlay. | Rapid like/unlike, follow/unfollow, and repeated profile edits can complete or project out of order. | Review finding adapted to user clarification | AC-047 |
| FR-040 | Functional | Must | Ordinary Flutter mutation operations and overlays shall remain in memory only and shall not be written to durable device storage. Restart, process death, sign-out, account removal, account switch, and permanent-deletion completion shall begin with no restored ordinary mutation state. | The accepted simplicity tradeoff avoids local persistence and cross-session recovery machinery. | User clarification | AC-048 |
| FR-041 | Functional | Must | Creating a set-like action shall inspect matching records from a verified repository snapshot or a complete read tied to a captured head. If a valid match exists and the head is confirmed unchanged, the command shall complete as an idempotent no-op using deterministic representative metadata; otherwise creation shall use a head-guarded write so concurrent change causes reread rather than an avoidable duplicate. | Tap lag and mixed pagination must not cause Craftsky itself to create logical duplicates or report a stale no-op. | Existing API contract and review finding | AC-049 |
| FR-042 | Functional | Must | Notification identity and delivery state for a set-like scope shall be keyed to the logical aggregate rather than its representative source URI. Representative changes may update bounded metadata but shall not create new activity or delivery. | Representative churn must not resurface an existing notification. | Review finding | AC-050 |
| NFR-001 | Non-functional | Must | Source installation, fact replacement, affected-scope aggregation, notification transitions, and projection-job completion shall be transactionally safe and idempotent. | Crashes and duplicate delivery are expected. | Architecture constraint | AC-041 |
| NFR-002 | Non-functional | Must | The design shall assume no ordering across repositories and no strict callback ordering for historical events or projection jobs. | Cross-owner records and asynchronous workers cannot rely on stronger ordering. | Tap contract | AC-005, AC-015 |
| NFR-003 | Non-functional | Must | Raw operation keys, record bodies, OAuth tokens, DPoP material, and PDS credentials shall not appear in logs or metrics; server-persisted command data shall be limited to reconciliation needs, and ordinary Flutter mutation state shall remain in memory. | The rewrite must preserve privacy and credential boundaries. | Architecture constraints and user clarification | AC-042, AC-044 |
| NFR-004 | Non-functional | Must | Release validation shall include `just test` and `just appview-check` for AppView, plus `just app-test` for Flutter in-memory reset, account-switch, and overlay tests. It shall cover real-PostgreSQL race interleavings, authoritative Lexicon validation, and PDS command contracts for every class in the mutation matrix. | Fake sequential tests and backend-only gates will miss the targeted failures. | Confirmed direction and repository workflow | AC-043 |
| NFR-005 | Non-functional | Should | Telemetry should expose bounded collection, command kind, source validation state, projection stage, dependency, PDS outcome, retry count, aggregate transition, and convergence latency. | Operators need to distinguish command, source, and projection failures. | Recommended direction | AC-044 |
| NFR-006 | Non-functional | Must | Source and projection recovery shall remain correct if Tap redelivers or historically reorders events, initiates repository resynchronization, or changes historical delivery concurrency within its documented contract. AppView shall use verified-snapshot repair whenever completeness or absence must be established. | Tap is beta, event omission is not deletion proof, and transport implementation details may evolve. | Tap contract | AC-005, AC-045 |
| RULE-001 | Business rule | Must | PDS repository state is authoritative for source existence. Tap-derived validated and locally eligible source state is authoritative for AppView public serving projections. | Establishes one source truth while preserving explicit local serving policy. | Confirmed direction | AC-001, AC-034, AC-038 |
| RULE-002 | Business rule | Must | A valid record from another client has the same projection authority as a valid record written through Craftsky. | Federation cannot depend on origin. | Confirmed direction | AC-001 |
| RULE-003 | Business rule | Must | A set-like scope is active if and only if at least one valid, projection-eligible current source fact matches it. | Defines duplicate-safe semantics without counting unresolved or locally hidden facts. | Confirmed direction | AC-013, AC-015, AC-029 |
| RULE-004 | Business rule | Must | Tap projectors are the only ordinary path that inserts, updates, or deletes public serving projections. | Prevents split-brain reads. | Architecture rule | AC-038 |
| RULE-005 | Business rule | Must | Command state describes requested effects and delivery progress but does not constitute public record state. | Keeps the journal from becoming a second AppView. | Confirmed direction | AC-023, AC-038 |
| RULE-006 | Business rule | Must | Repository revisions order versions within one repository; the system shall not derive causal order between different repositories from Tap arrival time. | Prevents invalid cross-owner ordering assumptions. | Tap contract | AC-006, AC-015 |
| RULE-007 | Business rule | Must | Within its authenticated owner and operation-kind scope, one operation key identifies one immutable command and shall never be reused for a changed or intentional new command. | Defines the API and Flutter idempotency contract without coupling unrelated owners. | Confirmed direction | AC-017, AC-018, AC-019 |
| RULE-008 | Business rule | Must | Destructive PDS cleanup requires an explicit authorized user command or the separately authorized permanent account-deletion job. | Indexing must not become an implicit data-deletion authority. | Architecture rule | AC-030, AC-039 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-005, RULE-001, RULE-002 | Given equivalent valid records written through Craftsky and another authorized client, when Tap ingests them, then both become current source facts and, when locally eligible, participate equally in public serving projections. |
| AC-002 | BR-001, FR-003, FR-029 | Given a malformed or semantically invalid Craftsky record on a PDS, when Tap ingests it, then source evidence is retained but it cannot affect public projection or membership lifecycle. |
| AC-003 | BR-002, FR-012, FR-013 | Given a prepared create command whose PDS response is lost while the Flutter operation remains in memory, when Flutter retries with the same operation key and immutable request, then AppView reconciles the original command and does not create another record. |
| AC-004 | BR-003, FR-030 | Given a mutation is definitely accepted by the PDS but not yet projected, when Flutter renders the affected feature, then it presents the accepted result optimistically, treats the command as complete, and does not treat the old read model as failure. |
| AC-005 | BR-004, FR-002, NFR-002, NFR-006 | Given duplicate delivery, stale delivery, concurrent historical delivery, and independently scheduled projection jobs, when processing completes, then current public state is identical to a clean rebuild from current sources. |
| AC-006 | BR-004, FR-001, FR-002, RULE-006 | Given several versions of one URI arrive out of order, when ingestion compares repository revisions, then only the latest version controls current source state and equal conflicting versions fail closed for authoritative repair. |
| AC-007 | FR-003, NFR-004 | Given every indexed collection and supported action, when validation fixtures run, then valid records pass and invalid Lexicon shape, key policy, identifier, and semantic cases cannot enter source facts. |
| AC-008 | FR-003 | Given a PDS stores a custom record without validating its unknown Lexicon, when Craftsky ingests it, then Craftsky's own authoritative validator determines projection eligibility. |
| AC-009 | FR-004 | Given a URI previously held a valid source and its latest update is invalid, when that update is processed, then the previous fact is removed from affected projections; a later valid update restores current state. |
| AC-010 | BR-004, FR-006, FR-037 | Given command tables are empty, when serving projections are rebuilt from retained current sources, then the rebuilt results match normal projection output. |
| AC-011 | FR-007 | Given two valid matching follows, blocks, likes, or reposts with different URIs, when both are indexed, then both remain represented as current PDS-present facts. |
| AC-012 | FR-008 | Given a source update changes its logical target, when projected, then the old aggregate is decremented and the new aggregate is incremented transactionally. |
| AC-013 | FR-009, RULE-003 | Given two matching eligible set-like facts, when one is deleted, then the logical action remains active; when the last eligible fact is deleted or becomes ineligible, it becomes inactive. |
| AC-014 | FR-010 | Given an eligible set-like aggregate changes `0 -> 1 -> 2 -> 1 -> 0`, when notifications are processed, then exactly one activation and one final retraction occur. |
| AC-015 | FR-003, FR-009, FR-011, NFR-002, RULE-003, RULE-006 | Given a structurally valid like arrives before its post or another cross-repository dependency, when ingested, then it remains a valid but projection-ineligible blocked fact and does not count as active; when the dependency becomes available, it becomes eligible and projects without replaying the original Tap frame. |
| AC-016 | BR-002, FR-012 | Given AppView crashes before, during, or after PDS dispatch, when the same command resumes, then its operation identity, selected records, CAS inputs, and lifecycle fences remain recoverable. |
| AC-017 | FR-013, RULE-007 | Given an ordinary Flutter public mutation lacks a canonical UUID `Idempotency-Key`, when it reaches AppView, then it is rejected without a PDS write; distinct intentional commands use distinct keys, and Flutter is not required to restore a discarded key after its in-memory operation lifetime ends. |
| AC-018 | FR-014, RULE-007 | Given an owner and operation kind already use a key for one immutable command, when that scoped key is reused with changed lifecycle or payload data, then AppView returns a conflict without another PDS effect; the same UUID under another owner or operation kind does not collide. |
| AC-019 | FR-015, FR-038, RULE-007 | Given a terminal command is retried within 24 hours, then its full response is replayed; after compaction, its tombstone rejects reuse until owner purge removes it. |
| AC-020 | FR-016 | Given a command is prepared, dispatching, accepted, ambiguous, or rejected, when its authenticated owner retries the original mutation endpoint with the same operation key and immutable request, then AppView replays accepted or rejected, safely resumes reconciliation, or returns `202` with retry guidance while still ambiguous; no separate status request is needed. |
| AC-021 | BR-002, FR-017, FR-020 | Given a PDS connection fails after dispatch, when the command is recovered, then AppView reads the exact addressed state and either confirms the effect, confirms rejection/absence, detects conflict, or leaves it ambiguous without blind replay. |
| AC-022 | FR-018 | Given a PDS definitely accepts a mutation before Tap convergence, when AppView responds, then create/update returns the accepted result, DELETE returns `204` with no body, and neither requires later polling; a genuinely ambiguous result returns `202` with bounded retry guidance for the same mutation endpoint and operation key. |
| AC-023 | FR-005, FR-019, RULE-005 | Given a valid Tap source has no Craftsky command or mismatches an unrelated command, when projected, then its public eligibility is unchanged while only legitimate command correlation is updated. |
| AC-024 | FR-016, FR-019 | Given a command is definitely accepted, when Tap later observes, projects, or supersedes the corresponding source state, then the command remains accepted while ordinary reads independently converge to current authoritative state. |
| AC-025 | FR-020 | Given an addressed update/delete is retried after an ambiguous result, when its expected content is already present or the intended deletion is already absent, then it completes idempotently; a genuine CID/head conflict returns conflict. |
| AC-026 | FR-021 | Given a profile update changes both `self` records, when the PDS accepts the guarded `applyWrites`, then both changes share one repository commit; if the guard fails, neither requested change is applied by that transaction. |
| AC-027 | FR-022 | Given several valid PDS records match one set-like scope, when the user removes the action, then AppView binds enumeration to a known repository head, targets every match from that complete snapshot in one transaction guarded by the same head, and does not infer absence from Tap or mixed pagination. |
| AC-028 | FR-023 | Given `applyWrites` returns `InvalidSwap`, when the retry budget remains, then AppView rereads repository truth, rebuilds steps under the same operation key, and retries without deleting a record introduced after the refreshed snapshot. |
| AC-029 | FR-024, RULE-003 | Given set-like removal succeeds and another client creates a matching record in a later repository commit, when Tap projects it, then the logical action becomes active again without an old delete generation absorbing it. |
| AC-030 | FR-025, RULE-008 | Given Tap observes duplicate, invalid, old, or non-representative records without an explicit remove command, when indexers and repair workers run, then they issue no PDS deletion. |
| AC-031 | FR-026 | Given scheduled publication retries after restart or ambiguous dispatch, when final publication resumes, then its existing frozen publication identity is preserved and its outcome is exposed through the common command contract without duplicate scheduling state. |
| AC-032 | FR-027 | Given a fixed-key profile command retries, when the exact `self` record already contains the canonical desired value, then it reconciles successfully without a TID or duplicate record. |
| AC-033 | FR-028 | Given an unauthorized, stale-generation, terminal-owner, or unexpected-target command, when dispatch or recovery is attempted, then it is rejected before an unauthorized PDS effect. |
| AC-034 | FR-029 | Given an invalid profile create/delete event and a valid profile create/delete event, when each is ingested, then only the valid event may transition non-terminal membership while terminal local state remains unchanged by either. |
| AC-035 | BR-003, FR-018, FR-030, FR-031 | Given definite PDS acceptance and delayed serving projection, when Flutter receives stale ordinary reads, then its accepted overlay continues to present the intended result without any status request; when an ordinary read agrees, Flutter retires the overlay, and when the bounded lifetime expires without agreement, Flutter retires it and forces an ordinary refresh. |
| AC-036 | BR-003, FR-032 | Given a dispatched command has an ambiguous outcome and remains in memory, when the user attempts to edit or resubmit it, then Flutter reconciles the frozen command under the same key before allowing a changed command with a new key. |
| AC-037 | FR-030, FR-033 | Given an operation is in flight when the user signs out or switches account, when the outgoing state is cleared and a late response or projection refresh completes, then it cannot alter the new account's operation state or UI. |
| AC-038 | FR-034, RULE-001, RULE-004, RULE-005 | Given a command is accepted but Tap has not projected it, when another client calls an ordinary read endpoint, then the response contains only Tap-derived public state and no command-journal overlay. |
| AC-039 | FR-035, RULE-008 | Given permanent account deletion removes registered Craftsky records, when it runs, then it continues through the restricted deletion job rather than ordinary mutation commands or indexer cleanup. |
| AC-040 | FR-036 | Given a fresh development database, when migrations run, then the final source-fact, aggregate, command, and projection schema exists with no relationship-intent compatibility table. |
| AC-041 | FR-008, NFR-001 | Given interruption or duplicate execution at any source/fact/aggregate/notification transition, when processing resumes, then source presence, logical activity, side effects, and job completion converge exactly once. |
| AC-042 | FR-028, NFR-003 | Given mutation dispatch and recovery paths, when security tests run, then authorization and lifecycle fences remain enforced and no reusable PDS credential reaches Flutter or telemetry. |
| AC-043 | NFR-004 | Given `just test`, `just appview-check`, and `just app-test`, when release validation runs, then real-PostgreSQL race cases, protocol validation, PDS command contracts, and Flutter in-memory reset/account-switch/overlay scenarios cover every class in the mutation matrix. |
| AC-044 | NFR-003, NFR-005 | Given command, source, and projection success or failure, when telemetry is emitted, then it distinguishes the bounded stage/outcome and excludes raw keys, record bodies, tokens, and credentials. |
| AC-045 | BR-004, FR-006, FR-037, NFR-006 | Given source-order uncertainty, repository resync, or projection divergence, when repair obtains a complete verified repository snapshot and rebuilds, then omitted records are removed and projections equal current valid, eligible repository sources without relying on event omission, original order, or command history. |
| AC-046 | FR-015, FR-038 | Given terminal receipt compaction, unresolved command recovery, and owner purge, when cleanup runs, then unresolved data survives, replay payloads compact safely, tombstones prevent reuse, and all owner command data is removed at purge. |
| AC-047 | FR-039 | Given like then unlike, follow then unfollow, or two profile edits overlap before ordinary reads converge, when older original or retry responses or read results arrive after the newer operation, then only the newest active operation or accepted overlay for each logical scope controls local presentation. |
| AC-048 | FR-030, FR-032, FR-033, FR-040 | Given in-memory operation or overlay state exists, when the app restarts, its process/controller is reinitialized, the user signs out, removes or switches account, or completes permanent deletion, then no ordinary mutation state is restored and late results cannot affect a later account session. |
| AC-049 | FR-041 | Given Tap has not projected an existing external follow, block, like, or repost, when Craftsky receives a create command for the same logical scope, then a complete head-bound read returns the existing representative as a no-op only if the head remains unchanged; otherwise reread occurs rather than an avoidable duplicate or stale no-op. |
| AC-050 | FR-010, FR-042 | Given a set aggregate remains active while its representative URI changes, when projection and notification reconciliation run, then notification activity time, newness, and delivery state are not reset or redelivered. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | PDS commits a create but the HTTP connection closes | Reconcile the selected URI and immutable payload under the same operation key; do not choose another rkey. | FR-012, FR-017 |
| EC-002 | Tap is unavailable after definite PDS acceptance | Return success immediately and let Flutter present its accepted overlay until an ordinary read agrees or the bounded overlay lifetime expires and triggers a refresh. | FR-018, FR-031 |
| EC-003 | Tap redelivers one event | Source installation and projection are idempotent. | FR-002, NFR-001 |
| EC-004 | Historical versions arrive newest-first | Repository revision preserves the newest current source and stale jobs cannot restore older state. | FR-001, FR-002 |
| EC-005 | Equal repository revision carries conflicting content | Mark ordering uncertain, block projection, and request authoritative repository repair. | FR-001, FR-002 |
| EC-006 | External client creates duplicate likes | Preserve every URI and keep one active logical aggregate. | FR-007, FR-009 |
| EC-007 | External client deletes only one duplicate | Remove only that source fact; keep the logical action active while another fact remains. | FR-008, FR-009 |
| EC-008 | External client updates a like to target another post | Recompute old and new scopes atomically. | FR-008 |
| EC-009 | Latest record version becomes malformed | Retain invalid source evidence and retract its previous valid fact without falling back to historical content. | FR-003, FR-004 |
| EC-010 | Like arrives before referenced post | Keep a blocked dependency and retry when the post changes. | FR-011 |
| EC-011 | User removes a set action while another client writes | `swapCommit` detects the race; bounded reread decides which records precede the successful remove commit. | FR-022, FR-023 |
| EC-012 | External client recreates after successful removal | Treat the later record as active current truth. | FR-024 |
| EC-013 | `applyWrites` is unsupported or the atomic batch exceeds a PDS limit | Fail safely with an explicit unsupported/too-large outcome and no partial mutation. An implementation fallback is permitted only if it preserves the same atomic external contract. | FR-021, FR-022 |
| EC-014 | Profile compound update loses its response | Reconcile both fixed URIs and the guarded command before redispatch. | FR-017, FR-021 |
| EC-015 | An accepted command is immediately superseded by a newer external write | Preserve the latest public state. Flutter may temporarily retain its accepted overlay, but bounded expiry and an ordinary refresh must reveal authoritative state without requiring command or projection status. | FR-019, FR-031 |
| EC-016 | Idempotency key is reused after response compaction | Reject from the tombstone; do not dispatch. | FR-015 |
| EC-017 | App restarts with an ambiguous operation | Discard the in-memory operation and do not automatically resume it. A later user retry uses a new operation key and may duplicate an append create if the earlier PDS write succeeded despite its lost response. | FR-030, FR-032, FR-040 |
| EC-018 | Account switches while ambiguous-operation reconciliation is active | Discard the old account's in-memory operation and overlay, ignore late results, and do not restore them when switching back. | FR-033, FR-040 |
| EC-019 | Tap replay or resync does not emit a formerly present URI | Do not infer deletion from omission. Verified-snapshot repair proves absence, removes the retained source, and recomputes affected projections without command history. | FR-037 |
| EC-020 | Terminal owner later receives valid external records | Retain source evidence as required for repair/audit, but mark it locally ineligible so terminal policy prevents serving visibility and authorization restoration. | FR-005, FR-028, FR-029 |
| EC-021 | Invalid duplicate is present during explicit set removal | It contributes no active fact and is not autonomously deleted merely for being invalid. | FR-003, FR-025 |
| EC-022 | Two operations concurrently reuse one key | Transactional uniqueness selects one immutable command; incompatible reuse conflicts. | FR-012, FR-014 |
| EC-023 | Older Flutter operation finishes after a newer operation on the same scope | Record the older outcome without changing or retiring the newer operation's overlay. | FR-039 |
| EC-024 | Source observation precedes projection completion | The command remains accepted without further client requests, and Flutter keeps the overlay until an ordinary read agrees or bounded expiry forces a refresh. | FR-016, FR-019, FR-031 |
| EC-025 | Set-like create races an unprojected external create | Guard creation by repository head and reread after conflict so Craftsky does not knowingly add another logical duplicate. | FR-041 |

## 15. Data / Persistence Impact
- Retained source state: Evolve `tap_source_records` to remain the latest per-URI repository evidence while removing public eligibility dependence on Craftsky effect origin.
- Validated source facts: Add or adapt collection-specific source-level rows carrying URI, CID, repository revision, structural/semantic validity, normalized logical scope, dependency state, and local projection eligibility.
- Set aggregates: Materialize eligible logical activity and deterministic representative metadata separately from source presence and validity.
- Command journal: Add or adapt one owner/operation-kind-scoped command table with immutable fingerprint, operation key hash, lifecycle fences, selected identities, dispatch state, PDS outcome, replay expiry, and terminal tombstone. It does not need source-observed, projection-converged, or superseded command states.
- Command steps: Represent every addressed write in compound `applyWrites`, set removal, profile update, and scheduled final publication.
- Projection jobs: Continue durable source-first jobs, adding affected-scope recomputation and dependency wake-up where needed.
- Flutter persistence: None for ordinary mutation operations or overlays. Keep account generation, operation UUID, logical scope keys, monotonic sequence, immutable dispatched input or fingerprint, and accepted overlays only in Riverpod/in-memory controller state for the active process and account session.
- Migration required: Yes. Replace undeployed migration 73 directly and adjust existing source/projection schema as needed. No compatibility layer is required because there are no production users.
- Cleanup: Separate terminal response compaction from unresolved server command state. Server owner purge removes server commands, steps, receipts, and tombstones. Flutter has no durable ordinary-mutation state to purge; resetting the controller or account session clears its in-memory state.

## 16. UI / API / CLI Impact
- UI: Features use a shared in-memory operation controller for progress, idempotent retry of the original mutation, optimistic accepted state, ambiguous recovery, definitive failure, ordinary-read reconciliation, and bounded overlay expiry. Existing feature-specific optimistic behavior is adapted behind that contract; no ordinary mutation state is restored after controller or account reset.
- API: Every mutation marked as requiring a key in the mutation matrix requires `Idempotency-Key`. Definite PDS acceptance completes through the endpoint's normal success contract without requiring projection metadata or later polling. Successful DELETE preserves `204` with no body. Ambiguous work returns `202` with bounded retry guidance such as `Retry-After`; the client retries the same endpoint with the same key and immutable request.
- API: No general operation-status resource is added. Each mutation endpoint owns replay and ambiguous-outcome recovery for its command.
- API: Use conflict responses for immutable-key mismatch, stale CID, and exhausted repository-head conflicts; validation responses for malformed input; unavailable responses only when dispatch cannot begin safely.
- API: Ordinary reads remain Tap-only and do not merge command state.
- CLI: Administrative repair/rebuild tooling may expose source, projection, and command status separately; no end-user CLI is required.
- Background jobs: Projection workers recompute current state and dependencies. Command recovery handles ambiguous commands. Scheduled publication retains its worker. Indexers do not dispatch destructive PDS cleanup.

## 17. Security / Privacy / Permissions
- Authentication: Existing member-session authentication remains required for original and retried mutation requests.
- Authorization: Owner lifecycle, expected owner/target, directed-interaction, CID/head CAS, and terminal-policy checks run before dispatch and recovery.
- Credential boundary: OAuth access/refresh tokens, DPoP keys, and generic PDS credentials remain AppView-only. The existing narrow video-upload service JWT exception is unchanged.
- Sensitive data: Command records retain only data needed for immutable fingerprinting, dispatch, and reconciliation. Tombstones retain scoped hashes rather than raw operation keys.
- External records: Valid, locally eligible public records are projected regardless of origin, but they do not grant API authorization or override terminal local policy.
- Destructive actions: Only explicit authenticated commands and the separately authorized permanent-deletion workflow may delete PDS records.

## 18. Observability
- Events: Command prepared, dispatch started, PDS accepted/rejected/ambiguous, source observed, projection converged, source valid/invalid/uncertain, dependency blocked/woken, aggregate transition, repository repair, and overlay reconciliation by ordinary-read agreement or bounded expiry.
- Logs: Bounded owner-safe identifiers, collection, operation kind, stage, reason, retry count, and request ID. Exclude raw operation keys, record bodies, tokens, and credentials.
- Metrics: Command latency by dispatch stage, ambiguous age, source-validation outcomes, projection queue age, blocked dependencies, eligible aggregate widths, external-origin ratio where safely inferable, Tap convergence delay, swap conflicts, repair/rebuild duration, and Flutter operation recovery outcomes.
- Alerts: Growing ambiguous-command age, source-order uncertainty, repeated swap exhaustion, blocked-dependency growth, projection divergence after rebuild, protocol-validation failures, or Tap convergence delay.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Source facts and aggregate projections diverge. | Public state can disagree with the PDS. | Transactional affected-scope recomputation, invariant tests, and rebuild comparison. |
| RISK-002 | Semantic validation rejects records accepted by common PDS implementations. | Valid user data may be hidden. | Generate validation from authoritative Lexicons, quarantine with diagnostics, and monitor rejection reasons. |
| RISK-003 | Semantic validation is too permissive. | Malformed external data can corrupt serving projections or lifecycle. | Negative fixtures for every collection and validate profiles before lifecycle transitions. |
| RISK-004 | Command state grows into a second public-state model. | Complexity and origin-dependent behavior return. | Prohibit command-dependent projection eligibility and require rebuilds without command history. |
| RISK-005 | `applyWrites` support or batch limits vary across PDS implementations. | Compound profile or set removal cannot meet atomic guarantees everywhere. | Detect capabilities/errors, return explicit no-partial-mutation failure, permit only contract-equivalent reviewed fallbacks, and monitor unsupported cases. |
| RISK-006 | Authoritative set removal requires expensive collection scans. | High mutation latency for accounts with large collections. | Use bounded pagination, source hints only as an optimization, repository-head CAS, and operation progress. |
| RISK-007 | Flutter overlay persists after a newer external action. | UI temporarily displays stale intended state. | Account and sequence fences, a bounded overlay lifetime, and a forced ordinary refresh on expiry. |
| RISK-008 | Cross-repository dependencies never wake. | Valid interactions remain absent. | Durable dependency indexes, wake-up tests, and stale-blocked alerts. |
| RISK-009 | Notifications fire per source rather than aggregate edge. | Duplicate notifications or premature retractions. | Make aggregate transition and notification mutation one transaction. |
| RISK-010 | Terminal local policy conflicts with later PDS records. | Public source and Craftsky visibility intentionally differ. | Keep source evidence, document terminal override, and test fail-closed behavior. |
| RISK-011 | Tap beta behavior changes. | Ordering or recovery assumptions become invalid. | Depend only on documented at-least-once/per-repo revision semantics and retain authoritative rebuild. |
| RISK-012 | The rewrite touches ingestion, projection, API, Flutter, and migrations together. | Regression surface is high. | Phase behind invariant tests, preserve source-first ingestion, and require high-risk review before test design and implementation. |
| RISK-013 | Flutter retires an accepted overlay merely because an ordinary read is still stale. | The UI briefly reverts despite definite PDS acceptance. | Retire on an agreeing ordinary read or bounded expiry, never on a disagreeing stale read alone; test delayed projectors. |
| RISK-014 | Older overlapping Flutter operations alter a newer overlay. | Rapid toggles or edits visibly revert. | Scope and sequence operations and make only the newest active operation or accepted overlay authoritative for local presentation. |
| RISK-015 | Flutter loses an unresolved operation on restart, process death, sign-out, or account switch after the PDS accepted a write but before Flutter received the response. | A later retry with a new operation key may duplicate an append create such as a post or business event. | Accept this simplicity tradeoff explicitly; rely on Tap-derived reads to reveal the original write and retain in-session idempotency for normal retries. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Every current PDS record version delivered by Tap includes a repository revision, URI identity, action, and CID/record data sufficient for latest-source selection. | Source storage would require more authoritative PDS reads. |
| ASM-002 | Standard target PDS implementations support `com.atproto.repo.applyWrites` and `swapCommit`. | Compound operations need a separately reviewed fallback contract. |
| ASM-003 | Follow, block, like, and repost product semantics are active-if-any-valid-and-eligible-source, not exactly-one-physical-record. | The aggregate model would need a different user-visible policy. |
| ASM-004 | There are no production users or released clients requiring compatibility with migration 73 or existing mutation request contracts. | A staged schema/API migration would be required. |
| ASM-005 | Product behavior accepts that ordinary Flutter mutation identity, ambiguous state, and overlays are in-memory only and disappear on restart, process death, sign-out, or account switch. | Durable client persistence and cross-session recovery would need to be reintroduced. |
| ASM-006 | Ordinary reads may remain eventually consistent while mutation responses and local overlays show accepted state. | AppView would need an explicit read-your-writes API mode, increasing complexity. |
| ASM-007 | A deterministic representative is sufficient wherever an API needs one URI/CID for an active set aggregate. | APIs may need to expose all source identities instead. |
| ASM-008 | Projection rebuilds can be executed from retained current sources without requiring full historical event order. | More repository history or periodic PDS snapshots would be required. |

## 21. Open Questions
- [ ] Non-blocking coding design: What exact `202` response body, `Retry-After` policy, and bounded Flutter retry backoff fit the existing mutation API contracts without introducing a general success envelope?
- [ ] Non-blocking coding design: Should the command journal evolve `owner_effect_attempts` or replace it with a narrower schema?
- [ ] Non-blocking coding design: What canonical fingerprint encoding covers blobs, derived timestamps, and compound command steps?
- [ ] Non-blocking coding design: What source-fact and aggregate table boundaries minimize duplication while preserving rebuildability?
- [ ] Non-blocking coding design: What bounded swap retry count, verified-snapshot reader, and contract-equivalent fallback implementation should be used?
- [ ] Non-blocking coding design: What bounded in-memory overlay lifetime and refresh trigger should apply after acceptance, and which existing feature overlays can be consolidated immediately?
- [ ] Non-blocking delivery planning: What vertical slice order best proves the architecture before broad migration?

## 22. Review Status
Status: Draft
Risk level: High
Review recommended: Required
Reviewer:
Date: 2026-09-17
Notes: This replaces the prior intent-heavy mutation design. It changes public source interpretation, set-like semantics, command persistence, profile transactions, API mutation contracts, and Flutter operation state. Explicit approval is required before acceptance-test design or implementation.

## 23. Handoff To Test Design
- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-004`, `FR-001` through `FR-042`, `NFR-001` through `NFR-006`, and `RULE-001` through `RULE-008`
- Suggested test levels: source-ingestion unit tests; Lexicon/semantic validation fixtures; source-fact and aggregate property tests; real-PostgreSQL race and crash-boundary integration tests; command-journal idempotency tests; PDS `applyWrites`/`swapCommit` contract tests; Tap replay/resync/reorder tests; projection rebuild equivalence tests; notification edge tests; API contract tests; Flutter in-memory operation-controller reset, ordinary-read overlay reconciliation, bounded expiry, overlapping operations, and account-switch tests.
- Blocking open questions: None for requirements review. Non-blocking coding-design questions must be resolved in coding planning before implementation. Explicit high-risk approval is required before test design.
