# AV-029 — Moderation restoration outbox

- **Included finding:** AV-029, Medium — Moderation persistence and restoration scheduling are not atomic
- **Priority/order:** Persistence/worker correctness; land after the release gate and coordinate its migration number with index maintenance
- **Status:** Planned
- **Audit source:** [AV-029](../2026-08-12-appview-code-audit.md#av-029--moderation-persistence-and-restoration-scheduling-are-not-atomic)

## Shared implementation strategy

Persist a qualifying moderation output and a durable outbox row in one PostgreSQL transaction. Relay pending outbox rows to `instagram_reconciliation_jobs` using a short database-only, `FOR UPDATE SKIP LOCKED` transaction; inserting the reconciliation job and marking the outbox processed happen atomically. The reconciliation worker then performs the idempotent eligibility restoration under its lease/retry/membership controls and emits the AV-007-approved downstream outcome: a private follow suggestion on the strict-safety branch, or automatic-follow reconciliation with an explicitly accepted delayed-commit residual on the residual-acceptance branch.

This separates two guarantees cleanly:

1. The HTTP write guarantees that output plus restoration intent are both durable or neither is.
2. The worker guarantees that each durable intent is promoted at most once and retried until a terminal queue outcome.

Because there are no production compatibility constraints, remove the current optional post-insert callback contract rather than supporting both atomic and non-atomic paths. Keep the successful HTTP response/envelope unchanged.

## Finding closure

### AV-029 — Moderation persistence and restoration scheduling are not atomic

`ModerationStore.InsertOutput` currently inserts through the pool, then calls `ReconciliationTrigger.EnqueueModerationRestoration`, which performs an independent write. If enqueue fails, the handler returns an error although the moderation negate is already committed; restoration is absent and a retry can create another output.

AV-029 closes when:

- Account `hide`/`takedown` negates persist the moderation output and one keyed restoration intent in the same transaction.
- Any definitely rolled-back insert/outbox/pre-commit failure leaves neither row visible; after an indeterminate commit result the database still exposes both rows or neither, and same-key retry resolves the outcome safely.
- A commit-result transport failure is treated as outcome-unknown rather than proof of rollback; retrying the same required idempotency key within the documented receipt window returns the one original response instead of creating another output.
- The handler returns success only after that transaction commits.
- A relay crash/failure cannot lose or duplicate promotion into reconciliation jobs.
- Non-qualifying post subjects, apply actions, and warnings do not create restoration intents.
- The reconciliation worker remains the only component that evaluates candidates and promotes the approved downstream outcome; the moderation transaction itself never assumes or performs a PDS write.

## Scope and design decisions

### In scope

- Transactional moderation-output persistence.
- A durable moderation-restoration outbox schema and database-only relay.
- App dependency/worker wiring, idempotency, leases/locking, cleanup, metrics, and fault tests.
- Reconciliation of qualifying pre-existing development outputs.

### Out of scope

- Changing moderation visibility policy, values, source trust, or expiry behavior.
- Performing PDS writes inside the moderation HTTP transaction.
- Generalizing this table into a cross-domain event bus.
- Choosing the AV-007 product branch or redesigning its Instagram matching policy. This plan consumes the already approved branch and adapts promotion to its downstream outcome.
- Adding public moderation routes; the current synthetic endpoint remains dev-only.

### Decisions

1. Add `moderation_restoration_outbox` with `moderation_output_id` as its primary/idempotency key and FK to `moderation_outputs(id) ON DELETE RESTRICT`. Lifecycle/retention code always locks and classifies the child before parent deletion: pending work is promoted or explicitly cancelled; processed/cancelled evidence is copied to a bounded non-FK `moderation_restoration_history` row and then the live child is deleted. The parent can then be removed without either silent loss or indefinite FK blocking. Store target DID only in the live outbox; history keeps output ID, bounded outcome, and timestamps but no DID or moderator reason.
2. A qualifying output always gets an outbox row even if no active/discoverable Instagram link exists at insert time. The relay resolves current link state and records `queued` or `no_work`; later link activation continues to use its existing reconciliation trigger.
3. The relay holds row locks only during database operations. It selects pending rows with `FOR UPDATE SKIP LOCKED`, inserts an `instagram_reconciliation_jobs` row when work exists, and marks the outbox processed in the same transaction.
4. Use a traceable, bounded reason such as `moderationCleared:<outputID>` for the promoted job. The outbox primary key and atomic promotion, not string parsing, provide idempotency.
5. Pre-parse/validate the target DID before beginning persistence so an invalid target cannot be committed and then reported as failure.
6. Remove `ModerationRestorationEnqueuer` from `api.ModerationStore` construction. The store owns only the transaction/outbox insert; the Instagram worker owns promotion/processing.
7. Require `Idempotency-Key` on the synthetic moderation POST: 16–128 printable ASCII characters, sent in a header and stored only as SHA-256 plus a SHA-256 fingerprint of the normalized request. Store that material and the minimal replay response in a separate, non-FK `moderation_idempotency_receipts` row committed with the output/outbox. The receipt survives ordinary output/history archival for a documented 24-hour retry window; same-key/same-fingerprint replay during that window returns the original 201 body, while different-fingerprint reuse returns canonical `409 idempotency_key_conflict`. After expiry, the key is new and may create a new output. Missing/malformed keys return canonical 400 errors. This bounded contract is required because an HTTP/database client cannot always distinguish rollback from a committed transaction whose response was lost; it does not imply indefinite retention.
8. Parse and classify source and subject DIDs before persistence. For any DID with an AppView lifecycle row, acquire its shared owner fence in canonical DID order and re-read lifecycle in the transaction before inserting output/outbox; an existing terminal tombstone always rejects the write. A trusted external moderation source with no lifecycle row is permitted only when its explicit source class/credential authorizes it—absence does not create an implicit CraftSky owner. A CraftSky source that should have lifecycle state must use `EnsureOnboardingOwner`/the declared source policy rather than bypassing terminal exclusion.

## Unified implementation plan

1. Reserve the next migration number and add `appview/migrations/0000NN_moderation_restoration_outbox.up.sql`/`.down.sql`.
2. Define the outbox with:
   - `moderation_output_id TEXT PRIMARY KEY REFERENCES moderation_outputs(id) ON DELETE RESTRICT` (processed retention may later be pruned explicitly in child-then-parent order)
   - `target_did TEXT NOT NULL`
   - `status` constrained to `pending`, `queued`, `no_work`, and explicit lifecycle-cancellation outcomes such as `cancelled_target_terminal`
   - `reconciliation_job_id UUID` nullable with an exact FK to `instagram_reconciliation_jobs(id) ON DELETE SET NULL`; `queued` plus `processed_at` records that promotion occurred even after downstream job retention/lifecycle cleanup removes the job
   - `moderation_restoration_outbox_reconciliation_job_id_idx` with `reconciliation_job_id` first and `WHERE reconciliation_job_id IS NOT NULL`, so batched reconciliation-job deletion can perform the FK referential action without scanning the outbox
   - `created_at`/`processed_at`, with consistency checks between status/job/timestamps
   - a partial claim index on `(created_at, moderation_output_id) WHERE status='pending'`
   Define `moderation_restoration_history` separately with output ID, bounded terminal outcome, and processed/cancelled timestamps, no FK and no target/source DID. Define `moderation_idempotency_receipts` separately with `request_key_hash BYTEA PRIMARY KEY`, `request_fingerprint BYTEA NOT NULL`, replayable `output_id`/`output_status`, `created_at`, and `expires_at`; it has no FK to `moderation_outputs`, no source/target DID, and no request/reason payload. Add indexes led by `expires_at` for sweeping and by `output_id` for deliberate lifecycle/privacy erasure, plus a check that expiry follows creation.
3. The pre-production migration deletes existing synthetic `moderation_outputs` (after applying the documented role-aware restoration/reconciliation reset) because they have no caller idempotency key. It creates the receipt table rather than inventing keys for old rows; no output/outbox/receipt backfill remains after that intentional reset. If preservation is later required, stop and design a reviewed legacy-key namespace rather than inventing externally replayable keys in this migration.
4. Refactor `ModerationStore.InsertOutput` in `appview/internal/api/moderation_store.go`: validate and type source/subject DIDs first, require/validate the idempotency key, acquire all applicable source/subject shared owner fences in canonical DID order, begin a transaction, lock/recheck lifecycle tombstones in the global owner-row order, insert the output and conditional outbox through `pgx.Tx`, and insert the receipt carrying the exact successful response before committing and releasing fences. A receipt-key uniqueness conflict rolls back any speculative output/outbox; after the winner commits, compare the stored request fingerprint and return its stored response or a conflict. Never generate a second output for a live receipt. Define trusted external-source authorization separately so a missing lifecycle row cannot be confused with a departed/terminal CraftSky owner.
5. Remove the optional variadic restoration dependency and its direct call. Update `appview/internal/app/deps.go`, store fakes, and constructor tests accordingly.
6. Add an outbox promotion capability in `appview/internal/instagram/restoration.go` or a focused sibling file. In one transaction, select a bounded pending set, resolve the current eligible account link, insert a reconciliation job where applicable, and update each outbox row to `queued`/`no_work`. Define the job FK as `ON DELETE SET NULL` and make processed status—not permanent job-row existence—the promotion record.
7. Integrate promotion at the beginning of `ReconciliationWorker.ProcessBatch` (or a narrowly composed database relay invoked by the same run loop) so no second external-effect worker contract is needed. Promotion errors fail the batch and leave rows pending via rollback.
8. Keep downstream reconciliation jobs under their existing retry, lease-token, membership, relationship, and eligibility checks. On the AV-007 strict branch they may create only private suggestions; on the residual branch any later automatic PDS effect follows AV-007's durable attempt/fence/reconciliation contract. Do not perform candidate matching or PDS work while holding outbox locks.
9. Add bounded observations for pending age/count and promotion outcome. Use output/job IDs only in local diagnostic logs when allowed; never emit target DIDs or moderator reasons to external telemetry.
10. Integrate owner-lifecycle purge by DID role. Before deleting a moderation output, lock its outbox row and atomically choose a durable outcome: a source/moderator deletion promotes the target restoration intent before source-owned parent deletion; a target/subject terminal deletion records `cancelled_target_terminal`. Once terminal, copy the bounded outcome to `moderation_restoration_history`, delete the live outbox child, delete any receipt selected by that `output_id`, and then delete the parent. Privacy/lifecycle erasure deliberately ends that receipt's replay guarantee; terminal source/subject guards still prevent a retried request from recreating the purged effect. Never infer cancellation from FK cascade.
11. Add retention behavior: live processed/cancelled outbox rows remain linked for an audit/retry window. At expiry, one transaction copies bounded evidence to history, deletes the child, then deletes the parent only when lifecycle/retention calls for it. Ordinary retention does not delete a still-live idempotency receipt, so a replay remains stable even when its output has been archived. A fake-clock sweeper deletes receipts at 24 hours using the expiry index; after that boundary, reuse is a new request. Link/import/owner cleanup may delete downstream reconciliation jobs; `ON DELETE SET NULL` preserves the live outcome until history archival. Pending rows are never archived/pruned.
12. Add required request-idempotency parsing/error behavior to the dev moderation handler and store contract. Record only hashes and the minimal response facts, cap length, never log keys/fingerprints, and document the exact 24-hour replay/conflict window.
13. Run migration up/down/up and the complete moderation/Instagram/account-deletion suite under PostgreSQL and the race detector.

## Migration, reconciliation, and operations plan

This update requires a schema migration. Coordinate its number with AV-028/AV-034 and other audit migrations; never edit `000014` or `000025` in place.

The up migration performs the intentional pre-production moderation reset described above, then installs the required receipt/outbox schema. Before reset, run a one-off dry-run report of qualifying negates and, if any matter to development fixtures, enqueue equivalent reconciliation through the normal worker after the reset; do not preserve ambiguous unkeyed HTTP outputs. Verify there are no unaccounted old pending intents before declaring the migration complete.

The down migration drops the outbox only after its FK/indexes. It does not delete moderation outputs or reconciliation jobs already promoted. Down/up testing uses isolated data; do not use rollback as an operational cancellation mechanism. Lifecycle purge must use the explicit role-aware child transition before parent deletion; direct parent deletion is expected to fail rather than lose pending work.

Operational dashboards/alerts should cover pending count, oldest pending age, promotion failures, queued/no-work results, and downstream reconciliation failures. A non-empty old pending queue is actionable; an isolated downstream provider outage does not roll back the original moderation decision.

## API/client/config/operations impact

- The successful synthetic moderation response remains HTTP 201 with the existing camelCase `outputId`/`status` shape, but the endpoint now requires the breaking `Idempotency-Key` header. For 24 hours from the first successful commit, same-key/same-request replay returns the identical 201 body and conflicting reuse returns 409 `idempotency_key_conflict`; after expiry, the key is treated as new. Missing/malformed keys return 400 `missing_idempotency_key`/`invalid_idempotency_key` in the canonical envelope. A deliberate lifecycle/privacy purge may end the window early and is documented as such.
- Validation errors and internal failures continue through the canonical `{error,message,requestId}` envelope.
- A known pre-commit transaction failure returns 500 with no committed output/outbox. An indeterminate commit result returns a retryable error; the caller retries with the same idempotency key and receives the one committed result if it exists. A downstream reconciliation outage no longer makes the original HTTP request fail after commit.
- No Flutter change is required.
- Worker polling/batch settings may reuse current Instagram reconciliation limits; add configuration only if independent tuning is demonstrated necessary, and validate it through AV-030.

## Security, failure, and race considerations

- No external call occurs inside the moderation transaction or outbox promotion transaction.
- Multiple replicas use `FOR UPDATE SKIP LOCKED`; output primary key plus atomic insert/status transition prevents duplicate promotion.
- Crash before commit leaves neither output nor intent. Crash after HTTP commit leaves a durable pending intent. Crash during promotion rolls back job and status together.
- Loss of the commit acknowledgement may leave output, intent, and receipt committed even though the handler reports uncertainty. Database atomicity guarantees co-visibility, not that the client knows the outcome; the unique live receipt makes retry safe during the documented window.
- Receipts contain only key/fingerprint hashes and the minimal 201 response facts, never DID, moderator reason, or request payload. Routine history archival cannot erase the retry proof prematurely; explicit lifecycle/privacy purge removes receipts by output ID and relies on the terminal tombstone to reject resurrection.
- A worker must not mark `queued` unless the job insert succeeded in the same transaction.
- Account/terminal purge cannot delete a parent moderation output while its restoration intent is pending. It must atomically promote/preserve source-owned intent or explicitly cancel target-owned intent according to the documented DID role.
- Moderation insert and lifecycle transition share source/subject owner fences. If insert linearizes first, terminal purge sees and classifies/deletes it; if terminal commits first, the insert's tombstone recheck rejects it. A trusted external source without an AppView owner row may create only the explicitly authorized source role, while any existing terminal tombstone is authoritative regardless of trust class.
- Membership departure, deletion, link removal, moderation reapplication, mute/block state, and conflict state are rechecked downstream; no outbox row authorizes a public write by itself.
- Internal moderator reasons, target identity, and record payloads remain out of Sentry metrics/log attributes.

## Unified test plan

1. **Store unit/integration:** Cover qualifying and non-qualifying inputs, CraftSky source/subject lifecycle states, explicitly trusted external source with no lifecycle row, missing unauthorized CraftSky source state, and terminal tombstones; assert output/outbox visibility only after commit.
2. **Atomic fault injection:** Fail output insert, outbox insert, receipt insert, and a definitely rolled-back commit separately; from another connection assert no partial combination persists. Separately simulate a commit that succeeds while its acknowledgement is lost, retry the same idempotency key, and assert one output/outbox/receipt and the identical 201 response with no duplicate.
3. **Promotion integration:** Seed pending rows with eligible link/no link; assert job plus `queued` or `no_work` transition is atomic and linked correctly.
4. **Concurrency:** Run two promoters over the same pending rows and assert exactly one job per moderation output.
5. **Crash/retry:** Cancel/fail after selection, after job insert, and before status update; rollback leaves the row pending and the retry produces one job.
6. **Downstream:** Verify reconciliation retry/lease/membership tests consume the promoted job into the selected AV-007 outcome without duplicating work. Strict-branch tests assert no PDS capability; residual-branch tests assert promotion alone does not bypass durable attempt/fence/reconciliation rules.
7. **Idempotency retention:** With a fake clock, replay the same and conflicting keys before 24 hours, archive/delete the live output through ordinary retention, and prove the receipt still returns the original response/conflict. Advance through expiry, sweep, and prove reuse follows documented new-request semantics. Separately run source/subject lifecycle privacy purge, prove the receipt is removed early, and prove a retry cannot recreate an effect through the terminal guard.
8. **Migration/lifecycle:** Test the intentional legacy reset/report, constraints/indexes, up/down/up, and role-aware account/terminal deletion. Barrier terminal transition against output+outbox+receipt insertion for both source and subject: insert-first is visible to purge; terminal-first makes insertion reject; no schedule recreates a post-purge moderation effect. Race source-DID deletion and subject-DID deletion against outbox promotion; assert source deletion preserves/promotes the target intent, subject terminal deletion records explicit cancellation, and neither path silently cascades pending work. Delete link/import/owner reconciliation jobs and assert `ON DELETE SET NULL` leaves a valid processed outbox row; catalog and representative-plan assertions must prove the referential action uses the leading `reconciliation_job_id` index rather than scanning the outbox.
9. **Race:** Run moderation and Instagram packages under `go test -race` with PostgreSQL.

## Traceability and acceptance criteria

### AV-029

- **Implementation seams:** next-numbered outbox migration, `internal/api/moderation_store.go`, `internal/instagram/restoration.go`/reconciliation worker, `internal/app/deps.go`.
- **Verification seams:** moderation store/handler tests, restoration/reconciliation concurrency tests, migration tests, and worker observations.

- [ ] Every qualifying output and restoration intent commit atomically.
- [ ] Every definitely rolled-back/pre-commit failure leaves neither row visible; an indeterminate commit exposes output and intent atomically and same-key retry resolves to the one result.
- [ ] Ambiguous commit acknowledgement plus same-key retry within the documented 24-hour receipt window returns one co-visible output/outbox/receipt and cannot create a duplicate; conflicting key reuse is rejected during that window.
- [ ] Non-qualifying outputs create no intent.
- [ ] Multiple replicas promote at most one reconciliation job per output.
- [ ] Promotion crash/failure leaves retryable pending work and cannot split job/status durability.
- [ ] Source/moderator and target/subject lifecycle purges classify pending intents atomically by role; no parent-row cascade can erase unprocessed restoration work.
- [ ] Source/subject lifecycle guards fence output+outbox insertion against terminal transition: terminal-first rejects and insert-first is removed/classified by purge, including deterministic multi-DID ordering tests.
- [ ] Trusted external sources with no AppView lifecycle have an explicit authenticated source class; an existing terminal tombstone always rejects, and missing CraftSky lifecycle state never silently bypasses the guard.
- [ ] Lifecycle/retention can remove the moderation parent after terminal child classification while bounded non-FK history retains required evidence without owner/target DID data; privacy purge removes the associated receipt and terminal guards prevent recreation.
- [ ] No active link produces a durable `no_work` outcome rather than an ambiguous missing intent.
- [ ] The required `Idempotency-Key` header, validation errors, 24-hour identical replay/409-conflict window, post-expiry new-request semantics, and early privacy-purge behavior are documented and fake-clock tested with the canonical envelope.
- [ ] Intentional legacy reset/report and migration up/down/up tests pass.
- [ ] Reconciliation-job lifecycle uses the documented indexed `ON DELETE SET NULL` behavior and cannot scan the outbox, block privacy purge, or make processed outbox state invalid.
- [ ] Downstream membership/eligibility/fencing tests still pass under race detection.
- [ ] Promotion targets the explicitly approved AV-007 branch and does not assume a PDS write: private suggestion on strict removal, or durable-attempt automatic reconciliation on residual acceptance.

## Dependencies and coordination

- Reserve migration numbering with **AV-028/AV-034**; include the new `reconciliation_job_id` FK-support index in that plan's catalog/cascade review even when the migrations land separately.
- Run all migration/fault/race tests through the **AV-012/AV-013/AV-033/AV-036** release gate.
- Preserve lifecycle fences from AV-003 and AV-007; an outbox intent never overrides departure/deletion.
- Finalize worker behavior only after AV-007's product-requirement branch is approved and its requirements/acceptance/design artifacts are revised; this plan must not silently restore retired automatic writes or silently replace approved automatic following.
- If **AV-037** moves moderation/Instagram capabilities, keep the transaction and relay boundaries intact rather than recreating an optional post-commit callback.

## References

- [Moderation flow coding plan](../changes/2026-05-30-moderation-flow-mvp/04-coding-plan.md)
- [Instagram post importer coding plan](../changes/2026-07-23-instagram-post-importer/04-coding-plan.md)
- [API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [PostgreSQL explicit locking](https://www.postgresql.org/docs/16/explicit-locking.html)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
