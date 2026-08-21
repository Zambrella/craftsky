# AV-028 and AV-034 — Index maintenance

- **Included findings:** AV-028, Medium — Cascading foreign keys lack usable supporting indexes; AV-034, Low — Several indexes exactly duplicate uniqueness indexes
- **Priority/order:** Persistence phase; complete before realistic-volume lifecycle/deletion tests
- **Status:** Planned
- **Audit sources:** [AV-028](../2026-08-12-appview-code-audit.md#av-028--cascading-foreign-keys-lack-usable-supporting-indexes), [AV-034](../2026-08-12-appview-code-audit.md#av-034--several-indexes-exactly-duplicate-uniqueness-indexes)

## Shared implementation strategy

Treat both findings as one reviewed index inventory. Add the leading-column, non-partial indexes required for foreign-key cascades while removing only indexes whose key order, predicate, uniqueness semantics, and operator classes add no distinct access path. Validate the final set from PostgreSQL catalogs and representative plans, not from names alone.

Use a single next-numbered migration so the database never passes through a state that adds avoidable write cost before a later cleanup. The app is pre-production, so prefer a clean final schema and a local rebuild over compatibility aliases or retaining provably redundant indexes.

## Finding closure

### AV-028 — Cascading foreign keys lack usable supporting indexes

PostgreSQL does not automatically index referencing columns. Existing saved-post, push-delivery, and active-interaction indexes do not give cascades a complete leading-column path, especially for soft-deleted likes/reposts excluded by partial predicates.

AV-028 closes when the final schema contains usable leading indexes for:

- `saved_posts(post_uri)`
- `push_deliveries(account_subscription_id)`
- `craftsky_likes(subject_uri)` including soft-deleted rows
- `craftsky_reposts(subject_uri)` including soft-deleted rows

Catalog tests must prove exact definitions, and representative parent-deletion plans must use the intended child lookup.

### AV-034 — Several indexes exactly duplicate uniqueness indexes

The current schema maintains non-unique indexes identical to indexes already created by uniqueness enforcement:

- Active likes/reposts each have a unique partial `(did, subject_uri)` index and an identical non-unique partial index.
- `atproto_follows` has `UNIQUE (did, subject_did)` plus an explicit identical non-unique index.

AV-034 closes when those redundant non-unique indexes are removed after confirming no query depends on a different predicate/order/operator class, and the new AV-028 indexes do not recreate a duplicate.

## Scope and design decisions

### In scope

- Four FK-supporting indexes and the three audited exact duplicates.
- Migration up/down behavior, catalog assertions, query plans, cascade correctness, and write-cost checks.
- A reusable review rule for future cascading FKs/unique indexes.

### Out of scope

- Changing cascade semantics, soft-delete behavior, or PDS deletion policy.
- General query/index tuning beyond the audited tables.
- Dropping read-oriented partial indexes that retain a distinct access path without plan evidence.

### Decisions

1. Add ordinary B-tree indexes with FK columns first: `saved_posts_post_uri_idx`, `push_deliveries_account_subscription_id_idx`, `craftsky_likes_subject_uri_idx`, and `craftsky_reposts_subject_uri_idx` (final names may follow current convention).
2. Interaction FK indexes are non-partial; cascades must find active and soft-deleted rows.
3. Drop `craftsky_likes_active_did_subject_uri`, `craftsky_reposts_active_did_subject_uri`, and `atproto_follows_did_subject_did_idx` only after catalog/plan confirmation of exact redundancy.
4. Keep the unique constraints/indexes; they encode correctness as well as access paths.
5. Use standard transactional `CREATE INDEX` for pre-production datasets. Do not introduce `CONCURRENTLY` unless migration tooling is intentionally changed to support non-transactional files.

## Unified implementation plan

1. Inventory relevant FKs, constraints, indexes, predicates, column order, operator classes, and current usage with `pg_constraint`, `pg_index`, `pg_indexes`, and representative `EXPLAIN` output.
2. Reserve the repository's next migration number once. Create `appview/migrations/0000NN_index_maintenance.up.sql` and matching `.down.sql`; do not edit historical migrations.
3. In the up migration, create the four AV-028 supporting indexes and drop the three AV-034 redundant non-unique indexes.
4. In the down migration, recreate only the three removed historical indexes and drop only the four indexes introduced by this update. Make round-trip definitions exact.
5. Add `appview/internal/db/index_maintenance_migration_test.go` (or a cohesive existing migration test) that inspects `pg_get_indexdef`, predicates, uniqueness, and leading columns.
6. Seed active and soft-deleted interactions, saved posts, and push deliveries. Delete referenced posts/subscriptions and assert every child row cascades.
7. At representative cardinality, capture `EXPLAIN (ANALYZE, BUFFERS)` for child lookup/parent deletion and key read queries. Verify cascade paths use supporting indexes and read plans do not regress after duplicate removal.
8. Measure/index-report storage and write amplification before/after using a deterministic development fixture; document expected trade-offs rather than enforcing unstable timing thresholds in CI.
9. Run the full migration chain up/down/up and affected saved-post, push, interaction, follow, timeline, and account-deletion tests.
10. Add a contributor/test checklist: every new FK with delete/update action gets a referencing-index decision; every explicit index overlapping a unique constraint gets a distinct-plan justification.
11. When AV-029 lands, extend the same catalog/cascade review to its `moderation_restoration_outbox(reconciliation_job_id)` support index. Prove PostgreSQL uses that leading-column index for `instagram_reconciliation_jobs` deletion and `ON DELETE SET NULL`; do not fold it into this plan's four historical AV-028 indexes or create a duplicate in both migrations.

## Migration, reconciliation, and operations plan

This update requires one schema migration but no data backfill.

For shared development data:

1. Record baseline table sizes and representative plans.
2. Apply during a normal restart/maintenance window; standard index creation can hold locks proportional to table size.
3. Run `ANALYZE` on the affected tables if statistics are stale.
4. Re-run catalog and plan checks, then exercise popular-post/subscription deletion.

No logical reconciliation is required. If parallel remediation branches conflict on migration numbering, renumber before merge and rerun the complete chain. A destructive local reset is acceptable because there are no production users.

## API, client, configuration, and operational impact

- No HTTP API, JSON envelope, Flutter, or environment configuration changes.
- The schema intentionally changes only its physical access paths; uniqueness and cascade semantics stay identical.
- Four supporting indexes add some storage/write cost, offset by removing three proven duplicates.
- Operators should capture baseline/final sizes and plans, refresh statistics, and watch lock duration during the first realistic-volume deletion exercise.

## Security, failure, and race considerations

- Missing FK indexes are an availability/lock-amplification risk; exercise concurrent readers/writers during representative deletes.
- Partial active indexes cannot satisfy cascades for excluded rows; tests must inspect predicates, not infer from names.
- Do not drop unique indexes or constraints while removing duplicate non-unique indexes.
- Down migrations must not drop/recreate unrelated similarly named indexes.
- All migration identifiers are static SQL; no runtime interpolation.

## Unified test plan

1. **Migration:** Apply the prior version, seed data, migrate up, inspect definitions, migrate down, inspect restored definitions, and migrate up again.
2. **Correctness:** Delete referenced posts/subscriptions and assert all active/soft-deleted child rows cascade.
3. **Plans:** Verify intended indexes on representative cardinality with `EXPLAIN (ANALYZE, BUFFERS)` and guard key read plans after duplicate removal.
4. **Concurrency:** Run parent deletion with representative concurrent reads/writes and ensure no table-wide scan/lock amplification.
5. **Regression:** Run saved posts, push/notifications, likes/reposts, follows, timeline/profile, account deletion, and all migration tests.
6. **Catalog:** Assert no remaining exact duplicate across keys, predicates, uniqueness purpose, and operator classes in the audited set.

## Traceability and acceptance criteria

### AV-028

- **Implementation seam:** next-numbered AppView migration.
- **Verification seams:** migration catalog tests, cascade tests, and representative explain plans.

- [ ] All four audited FK columns have a usable leading-column index.
- [ ] Likes/reposts FK indexes include soft-deleted rows.
- [ ] Cascades remove every matching child row.
- [ ] Representative plans use the intended supporting indexes.
- [ ] Up/down/up passes on the supported PostgreSQL version.

### AV-034

- **Implementation seam:** the same index-maintenance migration.
- **Verification seams:** catalog equality checks and key read-query plans.

- [ ] The two redundant interaction `(did, subject_uri)` indexes are removed while unique partial indexes remain.
- [ ] The redundant follow `(did, subject_did)` index is removed while uniqueness remains enforced.
- [ ] No key read plan regresses because an allegedly redundant index had a distinct role.
- [ ] The final audited index set contains no new exact duplicate.

## Dependencies and coordination

- Reserve migration numbering with **AV-029** and every other schema-bearing remediation.
- AV-029 owns its new outbox FK-support index; this plan owns the reusable catalog and referential-action review that verifies it does not recreate the missing-index class.
- Run migration and plan tests through **AV-033** in the grouped release-gate plan.
- Lifecycle/deletion work in AV-002 and push work in AV-025 benefit from the improved cascade paths.
- If AV-037 moves store code, keep this migration independent of Go package moves.

## References

- [PostgreSQL foreign-key guidance](https://www.postgresql.org/docs/16/ddl-constraints.html)
- [PostgreSQL `EXPLAIN`](https://www.postgresql.org/docs/16/sql-explain.html)
- [PostgreSQL index documentation](https://www.postgresql.org/docs/16/indexes.html)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
