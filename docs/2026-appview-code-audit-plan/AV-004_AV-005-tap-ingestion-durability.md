# AV-004 / AV-005 — Tap ingestion durability and order-independent projection

- **Findings:** AV-004, “Tap ACKs transient indexer failures permanently after six deliveries”; AV-005, “Order-dependent index gates permanently lose posts and interactions”
- **Severity:** High / High
- **Priority/order:** 1 — establish the durable ingestion contract before adding indexers or re-running repository backfills
- **Status:** Planned
- **Source:** [AV-004](../2026-08-12-appview-code-audit.md#av-004--tap-acks-transient-indexer-failures-permanently-after-six-deliveries) and [AV-005](../2026-08-12-appview-code-audit.md#av-005--order-dependent-index-gates-permanently-lose-posts-and-interactions)

## Shared implementation strategy

Replace “invoke a projection and ACK if it returns `nil`” with a durable, replayable ingestion pipeline. A Tap event is acknowledged only after AppView has committed one of two durable outcomes:

1. a valid record/ordinary identity event, its current source or tombstone state, and the work needed to project it;
2. a permanently invalid event and enough bounded evidence to diagnose and replay it from quarantine.

Projection into serving tables is a separate, idempotent state transition. A missing membership profile, subject post, cross-repository record, or backfill result is a blocked dependency—not successful handling and not permanent failure. Arrival or reconciliation of that dependency wakes the blocked work. Temporary database, object-store, PDS, context, and internal errors remain retryable regardless of how many deliveries have occurred.

Security-revoking lifecycle events have a stricter synchronous durability outcome. Before ACK, ingestion must acquire the owner transition fence and commit the authoritative denial:

- a terminal identity event commits the irreversible terminal lifecycle tombstone, auth/effect denial, an exhaustive fixed component/role purge ledger, every required private-object cleanup intent, and the authoritative terminal predicate used by every serving/projection/effect query; and
- deletion of `social.craftsky.actor.profile` commits the departed lifecycle generation, ordinary-session/work invalidation, and membership-source tombstone before serving-table cleanup is allowed to continue asynchronously.

For profile create/update/delete, the source-state comparison and lifecycle transition are one owner-fenced database decision. `ActivateOwner` or `DepartOwner` runs only when the incoming revision/CID/action wins the same compare-and-set that installs the current profile source row/tombstone. A duplicate or stale event is durably recognized and may be ACKed without changing lifecycle/auth state or enqueueing a superseded projection. If Tap does not expose a trustworthy total order for the competing repository operations, ingestion marks the profile source uncertain and reconciles the authoritative PDS state under the fence before applying either lifecycle transition; it never guesses from delivery order.

Physical purge of public, recipient-referenced, and owner-private rows plus object cleanup may continue asynchronously in bounded keyset batches because owner cardinality is not bounded by configuration. Authentication/effect checks deny the terminal/departed owner immediately, and systematic terminal anti-joins/guards make every retained public/search/feed/relationship/moderation/recipient-notification row invisible or ineffective at ACK. No cleanup obligation exists only in Tap memory. Failure to acquire the fence or commit this fixed-size lifecycle/ledger outcome remains unacknowledged and retryable. Profile creation/rejoin activates only through the lifecycle service and cannot override a terminal tombstone.

This update deliberately replaces the current retry-count and order-sensitive contracts. The app is pre-production, so remove `TAP_MAX_RETRIES`, remove tests that enshrine dropping after six attempts, and rebuild the public projections from authoritative repository data rather than preserving incomplete local rows. Public records remain on PDSes; private/AppView data remains in Postgres. No reconciliation tool may write, mutate, or delete a user’s PDS record.

## Finding closure

### AV-004 — Tap ACKs transient indexer failures permanently after six deliveries

The shared pipeline closes AV-004 by making ACK depend on durable classification rather than an untyped retry counter:

- **Retryable failure:** return without ACK. Track attempts, age, and the last bounded error category for observability, but never convert the event to permanent solely because a count was reached.
- **Permanent input failure:** write a quarantine row in a successful transaction, then ACK. Invalid DID/NSID/AT URI/record-key syntax, malformed supported records, and unsupported action shapes need explicit reason codes.
- **Valid record/ordinary identity event:** commit source state and projection work idempotently, then ACK. A process crash between commit and ACK is harmless because redelivery converges on the same keys.
- **Security-revoking lifecycle event:** commit the fenced terminal or membership-departure outcome defined above, then ACK. Merely persisting source state and enqueueing generic projection work is not sufficient.
- **Classification or quarantine-storage failure:** do not ACK. Unknown errors and recovered panics default to retryable until an operator or a narrower classifier proves otherwise.

Malformed record and identity envelopes must pass through the same classifier. They must not bypass both ACK and attempt tracking as they do today. Quarantined events need operator-visible replay and reconciliation commands; quarantine is not a write-only dead letter.

### AV-005 — Order-dependent index gates permanently lose posts and interactions

The shared pipeline closes AV-005 by retaining valid source records before applying membership and subject visibility rules:

- A `social.craftsky.feed.post` arriving before `social.craftsky.actor.profile` is stored and its projection is marked blocked on actor membership. Profile projection wakes all relevant work for that DID.
- A like or repost arriving before its actor is a member, or before the subject post is projected, is stored and blocked on the precise missing dependency. Membership/post arrival wakes it.
- A delete/tombstone supersedes pending create/update work and prevents an older redelivery from resurrecting the record.
- A source URI/version matching an outcome-unknown AppView PDS-effect attempt is joined to its remote-boundary disposition. It remains non-serving while the owner is departed and the call is never repeated. If reconciliation shows authoritative still-current PDS state accepted before transition, it may project under the established same-DID rejoin rule; a proved-not-accepted call/job cannot create state, and terminal state never reactivates.
- Like/repost counts and notification activation occur in the same idempotent projection transaction. Replays do not double-count or duplicate notifications.
- Bluesky profile repair and other repository backfills become durable jobs with leases, retries, and reconciliation instead of one-shot calls whose errors are logged and discarded.

Serving tables may remain membership-filtered. The durable source layer exists to remove ingestion order from the visibility decision; it does not make private data public or bypass moderation/membership policy at read time.

## Desired outcome and invariants

- Every ACK corresponds to a committed source event, committed quarantine record, or committed fenced security-revoking lifecycle outcome.
- A retryable outage can last beyond six deliveries without causing permanent data loss.
- Valid records converge to the same serving projection for every legal interleaving across repositories.
- Source receipt, tombstones, projection jobs, and quarantine writes are idempotent under at-least-once delivery.
- The newest known repository state wins; stale replay cannot resurrect a deleted or superseded record.
- Projection side effects, including counts and notifications, are exactly-once in database state even when attempts repeat.
- Blocked work is bounded, observable, wakeable, and repairable through the same production code path used for live ingestion.
- AppView does not write to PDSes during ingestion repair or reconciliation.
- Terminal ACK work is bounded by the finite component catalogue rather than owner row count; terminal visibility predicates remain active until durable keyset purge completes.
- Projection eligibility carries owner-generation/remote-boundary disposition, so dependency wake-up cannot repeat or invent an old call; departure hides it, legitimate rejoin may restore accepted/current public PDS state, and terminal state always denies it.

## Scope

### In scope

- Tap event classification and ACK behavior in `appview/internal/tap/consumer.go`.
- Durable source, projection-job, quarantine, and repository-reconciliation storage plus sqlc queries.
- A transaction-aware projection runner and idempotent domain projectors under `appview/internal/index/`.
- Dependency wake-up for CraftSky membership, posts, likes, reposts, Bluesky profiles, and any related notification projections.
- Operator commands for backlog inspection, quarantine replay, and repository/projection reconciliation.
- Removal of retry-count drop configuration and controlled non-production rebootstrap.

### Out of scope

- Changing Tap’s wire protocol or forking Tap.
- Treating the AppView source layer as the authority over PDS records.
- Moving private drafts, mutes, push tokens, or moderation state onto a PDS.
- Adding new Lexicon record types or changing any schema under `lexicon/`.
- Generic account deletion or retention policy, except honoring observed source tombstones.

## Design decisions

1. **Persist before project.** The ACK boundary is durable receipt, not successful visibility in every derived table.
2. **Keep source and serving concerns distinct.** Raw public record state is retained in an internal source table; existing product queries continue to read purpose-built projections.
3. **Classify narrowly and fail safely.** Only deterministic payload defects are permanent. Dependency absence and all unknown/infrastructure failures are retryable or blocked.
4. **Use explicit dependency keys.** Jobs record a dependency type plus canonical key, such as member DID or subject AT URI, so insertion can wake work without broad scans.
5. **Use database leases, not process memory.** Projection and backfill work survives restart and supports multiple AppView instances.
6. **Make projection transactional.** Serving-row changes, dependency-state changes, count maintenance, and notification activation commit together.
7. **Use typed atproto identifiers at boundaries.** Parse DID, NSID, AT URI, record key, and action once; hand typed identifiers to storage and projectors.
8. **Prefer a clean rebuild.** Because there are no production users, migrate the schema and rebootstrap source/projections instead of attempting to infer records that were previously ACKed and lost.
9. **Separate retention from eligibility.** Retaining an authoritative PDS source record does not require serving it while membership is absent. Terminal tombstones are permanent projection denial; departed-generation effect dispositions prevent retries and distinguish accepted/current public state that may follow same-DID rejoin policy from calls that never crossed remote acceptance.

## Unified implementation plan

1. Add failure-first tests around the current consumer for a transient failure lasting more than six deliveries, a malformed event, a crash after durable commit but before ACK, and post/profile/interaction interleavings.
2. Define typed outcomes shared by Tap and index code: `Applied`, `Blocked`, `PermanentInvalid`, and `Retryable`. Give permanent reasons a closed set of stable codes; preserve wrapped errors for logs without persisting unbounded or sensitive text.
3. Add migrations and sqlc queries for current source records/tombstones, durable projection jobs, event quarantine, repository reconciliation jobs, and projection disposition keyed to the lifecycle plan's deterministic ordinary PDS-effect attempt. Use unique natural keys and lease indexes; add foreign keys only where they cannot recreate arrival-order coupling.
4. Introduce an ingestion service, likely under `internal/ingestion/`, that validates the envelope and identifiers, stores current source state, resolves terminal state and any matching URI/CID/content-fingerprint effect disposition, and enqueues/upserts only eligible projection work in one transaction. Store bounded JSON needed for replay, not Go error strings or secrets. Terminal source remains permanently non-serving; departed source is blocked, with accepted/current pre-transition public state eligible only after a legitimate same-DID rejoin.
5. Change `internal/tap/consumer.go` to call the ingestion service. ACK only after its transaction reports durable receipt or durable quarantine. Remove `shouldDrop` and the semantic use of `MaxRetries`; retain retry metrics outside correctness decisions.
6. Route malformed record and identity messages through the same outcome model. Define which identity actions are supported and quarantine unsupported/malformed actions without blocking a repository forever. Give terminal identity actions a dedicated ingestion branch that calls the owner-lifecycle `TerminalPurge` service under its exclusive fence and reports `Applied` only after the tombstone, auth/effect denial, exhaustive fixed component/role ledger, object-cleanup intents, and serving/projection terminal predicate commit. It must not enumerate an unbounded owner's rows under the Tap deadline. Give CraftSky profile create/update/delete a dedicated owner-fenced ingestion branch: compare/install source revision/CID/action and run `ActivateOwner`/`DepartOwner` in the same transaction only if that source CAS wins. Stale/duplicate events record no lifecycle/auth change; unordered competing events trigger PDS reconciliation before transition. A winning delete commits the departed generation, session/work invalidation, membership-source tombstone, and non-repeatable outcome-unknown disposition for unresolved old-generation effects before ordinary profile projection work is queued.
7. Add a database-backed projection worker with `FOR UPDATE SKIP LOCKED` leases, positive lease/backoff settings, attempt timestamps, and recovery of expired leases. A worker crash must make work eligible again.
8. Refactor `craftsky_post.go` and `craftsky_interaction.go` into transaction-aware projectors. Replace membership/subject `return nil` gates with `Blocked` outcomes carrying canonical dependency keys. Before materializing, recheck lifecycle and projection disposition: terminal is permanent denial; departed is blocked; accepted/current public PDS state may become eligible after legitimate same-DID rejoin; proved-not-accepted work never does.
9. On membership/profile and post projection, atomically wake matching eligible jobs. Ensure a missing dependency discovered after wake returns to `blocked` without busy-looping. Rejoin wakes authoritative accepted/current public source under existing policy but never repeats old effect work or wakes terminal/proved-not-accepted dispositions.
10. Reconcile likes, reposts, counts, and notifications inside the successful projection transaction. Use source URI/CID and notification uniqueness as idempotency keys; reverse derived state when a current tombstone is projected.
11. Convert the one-shot Bluesky profile backfill in `craftsky_profile.go` into a durable reconciliation job. Persist AddRepo/fetch failures, retry with bounded backoff, and expose stale/failed jobs operationally.
12. Add CLI subcommands, likely under `cmd/cli`, to list quarantine/backlogs, replay selected quarantine rows after code or data correction, enqueue a DID/repository reconciliation, and report blocked dependencies. Every replay re-enters the normal classifier and projector.
13. Remove `TAP_MAX_RETRIES` from `internal/app/config.go`, Compose, examples, and tests. Add validated positive lease, poll, batch, and backoff settings, coordinating with AV-030.
14. Document and execute the non-production cutover: stop AppView ingestion, apply migrations, clear/rebuild public source and projection state, request fresh repository ingestion/backfill despite the current `TAP_NO_REPLAY` setting, verify zero unexpected blocked/quarantined items, then resume serving.
15. Add bounded metrics and alerts for oldest unprojected age, jobs by state/reason, repeated retryable failures, quarantine insertions, replay results, and repository backfill age. Do not label metrics with DID, URI, CID, or raw error.

Likely files include `appview/internal/tap/consumer.go`, `appview/internal/index/*.go`, `appview/internal/app/deps.go`, `appview/internal/app/config.go`, `appview/internal/notifications/`, `appview/cmd/cli/`, `appview/migrations/`, `appview/queries/`, generated sqlc files, `docker-compose.yml`, and their focused tests.

## Data, schema, migration, and reconciliation plan

Add forward and rollback migrations for tables equivalent to:

- `tap_source_records`: canonical URI key, DID, collection, record key, current CID/revision/action, bounded record JSON or tombstone state, source timestamps, projection version/state, and durable eligibility/disposition metadata referencing the matching owner generation/effect attempt where applicable.
- `tap_projection_jobs`: unique source/projection key, state (`pending`, `blocked`, `processing`, `complete`), dependency kind/key, attempts, next-attempt time, and lease owner/expiry.
- `tap_quarantined_events`: stable event fingerprint, bounded original envelope/record, permanent reason code, first/last seen, occurrence count, and replay state/lease.
- `tap_repository_jobs`: DID/repository job kind, durable state, lease/backoff fields, and last successful reconciliation metadata.

Use compare-and-set semantics based on repository revision/CID/action so an older delivery cannot replace newer state. For the membership profile, execute that CAS and its owner lifecycle/auth transition atomically under the owner fence; projector order is not an authorization boundary. Departure also freezes unresolved AppView PDS-effect attempts from the closing generation as outcome-unknown and non-repeatable. Validate the exact ordering token supplied by Tap before selecting the comparison rule; where no total order exists, persist uncertainty and reconcile against the PDS under the same transition discipline rather than guessing. Reconciliation may attach an observed CID/version and classify accepted/current public state for ordinary rejoin, but cannot revive an old job or fabricate acceptance.

The current database cannot reveal events already ACKed and discarded. Because the deployment is pre-production, do not fabricate a lossy SQL backfill. Recreate public source/projection tables and re-ingest authoritative repositories. Preserve AppView-private tables unless their foreign-key contracts require an explicitly reviewed reset. Verify Tap 0.1.10 replay/AddRepo behavior and the operational effect of `TAP_NO_REPLAY`; if necessary, recreate the non-production Tap store or re-register repositories in a controlled procedure. Take a database snapshot before the reset even though rollback compatibility is not a product requirement.

Reconciliation must be safe to run repeatedly and resumable after interruption. Its dry-run/report mode lists counts and reason categories only. An apply run requires explicit scope, uses the same ingestion/projection code as live traffic, and never performs PDS writes.

## API, client, configuration, and operations impact

- No `/v1/*` success payload or client request shape needs to change. Public reads may become more complete after reconciliation.
- During rebootstrap, return the standard retryable JSON error envelope or hold AppView unready rather than serving a silently partial projection.
- Remove `TAP_MAX_RETRIES`. Add positive, bounded worker lease/poll/batch/backoff configuration with secure defaults; invalid values fail startup.
- Add readiness checks for migration compatibility and optionally for an explicitly configured maximum projection lag. Do not make one poison event permanently fail readiness after it has been quarantined.
- Operators need runbooks for Tap pause/resume, projection drain, quarantine review/replay, repository reconciliation, and the clean non-production rebuild.
- Logs may include a redacted event fingerprint and closed reason code. Raw record bodies, private table data, credentials, and unbounded remote errors stay out of logs and metric labels.

## Security, failure, and race considerations

- ACK must occur only after the relevant database commit returns success. If the ACK write itself fails, redelivery must be harmless.
- A quarantine transaction failure leaves the event unacknowledged; otherwise a database outage could recreate AV-004 through the error path.
- Do not classify context cancellation, deadline exceeded, serialization failure, connection loss, object-store/PDS failure, or panic as permanent input failure.
- Bound quarantined envelopes and sanitize stored error details to prevent a malformed event from causing storage or log exhaustion.
- Lease acquisition and completion use compare-and-set predicates. A stale worker cannot complete or overwrite work after losing its lease.
- Source create/update/delete races resolve using verified repository ordering and tombstones. Dependency wake-up is idempotent and safe if it races with a worker claim.
- Profile source CAS and owner lifecycle transition cannot split. A stale create after a winning delete cannot reactivate the owner, and a stale delete after a winning recreate cannot revoke the new generation; reconciliation supplies the decision when ordering is not provable locally.
- Projection and notification writes use database uniqueness, not in-memory deduplication.
- Cap attempts per polling interval and apply jittered backoff so a shared outage or newly arrived dependency does not create a retry storm.

## Unified test plan

### Unit tests

- Outcome classification for every supported record/identity shape and representative malformed, unsupported, retryable, and unknown failures.
- Retry counts never change a retryable outcome into permanent invalid.
- Source comparison rules for duplicate, stale, newer, and tombstone events.
- Dependency-key generation for member DID and subject AT URI.
- Quarantine redaction, size caps, reason codes, and replay state transitions.

### PostgreSQL and integration tests

- A database failure persists for more than six Tap deliveries, then recovers: no failure delivery is ACKed and the recovered event projects once.
- Permanent malformed input commits quarantine before ACK; a quarantine database failure produces no ACK.
- Commit succeeds and the process fails before ACK; redelivery changes neither source nor derived counts/notifications.
- A terminal identity event cannot be ACKed after only generic source/job persistence. Inject failure before terminal tombstone, auth/effect denial, fixed exhaustive component/role ledger, terminal query predicate, and cleanup-intent commit; each attempt remains unacknowledged. After commit, pause every purge worker while physical rows remain and prove other-account profile/feed/post/search/relationship/moderation/notification reads expose nothing and no effect may begin. Crash/redelivery resumes the same idempotent keyset/object purge obligations without restoring access.
- Configure the fixed terminal fence/tombstone/epoch/ledger commit at the validated `TAP_ACK_TIMEOUT` budget and one step beyond it. In-budget work commits and ACKs; injected over-budget work rolls back and remains unacknowledged. Seed an owner far beyond one purge batch and prove ACK latency/work is independent of row count, serving remains terminal-gated, and durable batches eventually converge without a deterministic timeout loop.
- A winning CraftSky profile delete cannot be ACKed while the owner remains active. Pause the later serving-table projector and prove the committed source tombstone already makes stale bearer authentication/effect checks fail; inject lifecycle-commit failure and prove Tap does not ACK. Deliver create → delete → stale create and delete → recreate → stale delete with barriers: only the source-CAS winner changes lifecycle/auth/effect permission and serving projection. Repeat with no trusted total order and prove authoritative PDS reconciliation decides before transition.
- Post-before-profile, like/repost-before-actor-membership, interaction-before-subject, and cross-repository concurrent delivery all converge after dependencies arrive.
- Let an AppView-originated record call be accepted, crash before outcome persistence, depart the owner, and ingest the resulting source while membership is absent. Prove it stays non-serving and is never repeated. On legitimate same-DID rejoin, prove authoritative still-current PDS state follows the approved rejoin projection policy. Pair it with a proved-not-accepted old call/job and a terminal owner, neither of which can create/reactivate state; duplicate/reordered delivery cannot exchange dispositions.
- Delete-before-projection and stale-create-after-delete do not resurrect records.
- Projection failure midway rolls back serving state, dependency state, counts, and notifications together.
- Expired leases are reclaimed; stale owners cannot complete them; multiple workers safely use `SKIP LOCKED`.
- Bluesky profile/AddRepo backfill fails, survives restart, retries, and eventually converges.

### Fault, concurrency, and end-to-end tests

- Run the consumer and multiple projection workers under `go test -race` with duplicate deliveries and dependency insertion barriers.
- Kill/restart between source commit, ACK, job claim, projection commit, and job completion.
- Exercise backlog/quarantine list and replay commands against a real test database, including interruption and resumption.
- Rebootstrap a disposable Compose stack from PDS/Tap fixtures and compare canonical posts, interactions, counts, and notifications with the expected repository state.
- Verify bounded metrics/alerts and confirm no record body, DID, URI, credential, or raw error appears in labels.

## Per-ID traceability and acceptance criteria

### AV-004

- [ ] `shouldDrop`/`MaxRetries` no longer controls ACK or permanent loss, and obsolete configuration/tests are removed.
- [ ] Retryable failures remain unacknowledged beyond six deliveries and succeed exactly once after recovery.
- [ ] Every deterministic invalid event is durably quarantined before ACK and can be replayed through supported tooling.
- [ ] Malformed record and identity envelopes cannot retry forever outside classification/tracking.
- [ ] A crash after durable receipt but before ACK is proven idempotent by an integration test.
- [ ] Terminal identity ACK occurs only after its fenced tombstone, auth/effect denial, exhaustive fixed component/role ledger, terminal serving/projection predicate, and cleanup-intent commit; physical purge is durable and batched.
- [ ] Terminal pre-ACK fence/tombstone/ledger geometry fits inside `TAP_ACK_TIMEOUT` with margin and is independent of owner row cardinality; timeout rolls back and never ACKs partial security state.
- [ ] CraftSky profile deletion ACK occurs only after fenced departure generation, ordinary-session/work invalidation, and membership-source tombstone commit; authorization does not wait for asynchronous serving projection.
- [ ] Profile `ActivateOwner`/`DepartOwner` is atomic with the winning source revision/CID/action CAS; stale create/delete deliveries ACK without changing lifecycle/auth, and unordered conflicts reconcile the PDS before either transition.
- [ ] Oldest-retry and quarantine conditions are observable and alertable without sensitive labels.

### AV-005

- [ ] Valid posts and interactions are durably retained even when membership or subject dependencies are absent.
- [ ] Profile/member/post arrival wakes precise blocked work without broad unbounded scans.
- [ ] Historical and cross-repository interleavings converge to identical posts, interactions, counts, and notifications.
- [ ] Deletes and stale redeliveries cannot resurrect source or projection state.
- [ ] Profile create/delete/stale-replay barriers keep source, lifecycle generation/auth permission, and serving projection on the same authoritative winner.
- [ ] An AppView PDS record accepted before departure remains source-retained/non-serving while departed, is never repeated, and follows approved same-DID rejoin projection policy only if still authoritative; proved-not-accepted work and terminal state never project.
- [ ] Bluesky profile/repository backfill is durable, retryable, restart-safe, and operationally visible.
- [ ] A clean non-production rebootstrap recovers authoritative public state, with unexpected blocked/quarantined work reviewed before rollout completes.

## Dependencies and coordination

- **AV-002 / AV-003:** terminal deletion and membership departure share the owner lifecycle. Both security-revoking transitions commit their fixed authorization/visibility outcome before ACK. Terminal physical purge of public, private, relationship, moderation, and recipient-notification rows may run afterward only because every read/effect/projector applies the committed terminal predicate and the exhaustive component ledger survives restart. Ordinary departed-profile cleanup may also run after a non-terminal profile-delete ACK, but lifecycle denial and non-repeatable outcome tracking may not; existing same-DID rejoin semantics for accepted/current public PDS state remain intact.
- **AV-012:** fail-closed migrations are required before the new durable tables can be trusted at startup.
- **AV-013:** complete dependency/toolchain remediation before treating fault and race results as a release gate.
- **AV-025 / AV-029:** reuse the project’s lease and atomic-side-effect patterns where applicable; do not invent incompatible worker semantics.
- **AV-028 / AV-034:** review every new foreign key, lease scan, dependency wake-up, and uniqueness constraint for supporting and duplicate indexes.
- **AV-030:** validate every lease, backoff, timeout, and poll duration as positive and check their relationships.
- **AV-033 / AV-036:** the PostgreSQL, fault, race, formatting, static-analysis, and vulnerability suites must become required gates.

## References

- [Tap integration design](../superpowers/specs/2026-04-17-tap-integration-design.md)
- [Feed post indexing design](../superpowers/specs/2026-05-04-feed-post-indexing-design.md)
- [Profile onboarding design](../superpowers/specs/2026-04-23-profile-onboarding-design.md)
- [CraftSky AT Protocol architecture reference](../../atproto-craft-social-app-reference.md)
- [Indigo Tap package documentation](https://pkg.go.dev/github.com/bluesky-social/indigo/cmd/tap)
- [PostgreSQL explicit locking](https://www.postgresql.org/docs/current/explicit-locking.html)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
