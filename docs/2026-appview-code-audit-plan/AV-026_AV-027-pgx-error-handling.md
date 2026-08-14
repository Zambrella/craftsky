# AV-026 and AV-027 — pgx error handling

- **Included findings:** AV-026, Medium — Search query errors are shadowed and can become request panics; AV-027, Medium — Two transactional row loops omit terminal iterator errors
- **Priority/order:** Persistence/API correctness; land early with the first static-analysis and PostgreSQL test gate
- **Status:** Planned
- **Audit sources:** [AV-026](../2026-08-12-appview-code-audit.md#av-026--search-query-errors-are-shadowed-and-can-become-request-panics), [AV-027](../2026-08-12-appview-code-audit.md#av-027--two-transactional-row-loops-omit-terminal-iterator-errors)

## Shared implementation strategy

Adopt one explicit pgx rule across the affected paths: check query acquisition at the call site, consume rows only after successful acquisition, check `rows.Err()` after every loop and before any dependent mutation or commit, then preserve the underlying error with concise operation context. Extract only the small row/query capabilities necessary for deterministic fault tests; do not build a repository-wide database abstraction.

Use pre-production freedom to normalize both search branches and both transactional iterators to the same readable shape rather than applying isolated syntax patches. Make staticcheck and database fault tests enforce the pattern thereafter.

## Finding closure

### AV-026 — Search query errors are shadowed and can become request panics

`SearchStore.searchPosts` declares an outer `err`, but cursor decoding with `:=` creates branch-local variables. Each branch's `pool.Query` then assigns that local variable, leaving the checked outer error nil. A failed acquisition can reach `rows.Close()`/iteration with invalid rows.

The update closes AV-026 when:

- Cursor decode failures return before querying.
- Each popular/chronological `pool.Query` result is checked in its own branch.
- No row method is called without successful acquisition.
- Both branches wrap storage failures consistently, preserve error identity, and continue mapping through the existing `search_unavailable` JSON envelope.
- Staticcheck reports no `SA4006`.

### AV-027 — Two transactional row loops omit terminal iterator errors

`push.Dispatcher.claim` and `PostStore.deactivateSubscriptions` interpret `Next() == false` as clean exhaustion without checking `rows.Err()`. A late connection/cancellation/protocol error can therefore commit operations based on only a prefix of the selected rows.

The update closes AV-027 when:

- Both loops check terminal iterator errors before the first dependent update or commit.
- Scan or terminal errors close rows and roll back the transaction.
- Push never leases an incompletely enumerated selection.
- Subscription cleanup never deactivates/cancels only a scanned prefix.

## Scope and design decisions

### In scope

- `SearchStore.searchPosts`, `Dispatcher.claim`, and `PostStore.deactivateSubscriptions`.
- Narrow `Query/Rows` test seams or unexported row-scanner helpers.
- Search-handler envelope tests, transaction rollback tests, and staticcheck enforcement.
- A bounded audit of other `for rows.Next()` loops while establishing the rule.

### Out of scope

- Search ranking, SQL meaning, cursor format, or notification/push business semantics.
- Replacing pgx, introducing an ORM, or creating a generic repository framework.
- Broad search/post package restructuring, which belongs to AV-037.

### Decisions

1. Use `decodeErr` and `queryErr` (or immediate returns) rather than a shared mutable `err` across branches.
2. Keep common row consumption only after a successful branch assigns a valid iterator.
3. Check `rows.Err()` immediately after iteration. `Close()` is resource cleanup, not error validation.
4. In transactional enumeration, collect/validate the full result set before applying dependent writes; any enumeration failure rolls back all work.
5. Add an unexported minimal iterator interface (`Next`, `Scan`, `Err`, `Close`) only where needed to inject prefix-then-error behavior.

## Unified implementation plan

1. Refactor `SearchStore.searchPosts` in `appview/internal/api/search_store.go`: decode with distinct variables, execute each query with an immediately checked result, and assign only successful rows to the common consumer.
2. Wrap acquisition failures with one lower-case operation label while preserving `%w`; keep invalid cursors typed and storage details server-side.
3. Retain the successful scan loop, `rows.Err()` check, result truncation, and opaque cursor generation.
4. In `appview/internal/push/dispatcher.go`, extract or normalize the selected-delivery scanner. Check terminal error before lease tokens are generated/persisted. If AV-025 replaces the multi-row query, apply this invariant to the new one-row claim path.
5. In `appview/internal/api/notification_devices.go`, collect subscription IDs, check terminal error, close rows, then perform deactivation/delivery cancellation. Never update using a partial prefix.
6. Return wrapped scan/iteration errors and let deferred `tx.Rollback` preserve all-or-nothing behavior.
7. Add deterministic fake iterators that yield zero/prefix rows followed by a sentinel error, plus scan-error and clean-exhaustion cases.
8. Add transaction-level PostgreSQL tests proving no leases or subscription changes commit after injected/cancelled enumeration failure.
9. Add handler contract tests proving search storage failure returns HTTP 500, `application/json`, and the canonical `{error,message,requestId}` envelope with `search_unavailable`.
10. Search all AppView `rows.Next()` loops. Fix equivalent omissions if the change is mechanical and behavior-preserving; otherwise open a separately reviewed follow-up rather than silently broadening this update.
11. Enforce `SA4006` and the chosen staticcheck set through the shared AV-036 release gate.

## Migration, reconciliation, and operations plan

No schema migration is required.

Historical partial effects are unlikely and cannot be inferred safely. Reconciliation is targeted and idempotent: re-run explicit account/installation subscription deactivation if an interrupted operation is known, and let expired push leases return through the fenced retry path. Do not rewrite succeeded/permanent push state without evidence.

Operationally, normal pgx failures now become visible store/batch failures rather than panics or false successes. Logs must use bounded operation/stage fields and keep SQL, parameters, tokens, and viewer identifiers out of external telemetry.

## API, client, configuration, and operational impact

- Successful search, push, and notification-device behavior does not change.
- Search acquisition failure reliably returns the existing `search_unavailable` canonical JSON envelope instead of risking a panic/connection abort.
- Device/account deactivation may return a retryable server failure rather than falsely succeeding after partial enumeration; callers can safely retry the idempotent operation.
- No Flutter, schema, or environment change is required.
- Storage/batch error observations become more accurate because terminal iterator failures are retained.

## Security, failure, and race considerations

- A query error must never be replaced by a nil-row panic, which obscures root cause and may abort the response.
- A terminal iterator error is not successful exhaustion; using the prefix can leave privacy-sensitive push subscriptions active.
- Preserve context cancellation and `errors.Is` through wrapping.
- Close row iterators before issuing dependent statements on the same transaction, but capture/check `rows.Err()` first.
- Run AV-025 concurrency changes against these rollback invariants so a refactor cannot reintroduce partial claims.

## Unified test plan

1. **Search unit/fault:** Table-test both sort modes for cursor decode failure and query acquisition failure; assert no panic and no row calls after failure.
2. **Iterator unit:** Fake zero-plus-error, prefix-plus-error, scan-error, and clean-exhaustion results for both transactional scanners.
3. **Database integration:** Run successful search, push claim, and subscription deactivation against PostgreSQL, then inject cancellation/connection failure and assert rollback from a separate connection.
4. **HTTP contract:** Assert `search_unavailable`, JSON content type, non-empty `requestId`, and no SQL/storage details.
5. **Race:** Run search, push, and notification-device tests under `go test -race`.
6. **Static analysis:** Run pinned staticcheck and require no `SA4006`.
7. **Regression:** Run existing search cursor/ranking/language/moderation tests and push/device lifecycle tests unchanged.

## Traceability and acceptance criteria

### AV-026

- **Implementation seam:** `appview/internal/api/search_store.go`.
- **Verification seams:** search store/response/handler tests and pinned staticcheck.

- [ ] Both search branches check acquisition errors at the call site.
- [ ] Cursor decode variables cannot shadow query errors.
- [ ] Failed acquisition cannot reach `Close`, `Next`, or `Scan`.
- [ ] Both sort modes have deterministic error/no-panic tests.
- [ ] `/v1/search/*` returns the canonical `search_unavailable` envelope for store failure.
- [ ] Staticcheck reports no `SA4006`.

### AV-027

- **Implementation seams:** `appview/internal/push/dispatcher.go`, `appview/internal/api/notification_devices.go`.
- **Verification seams:** push dispatcher and notification-device transaction tests.

- [ ] Both affected loops check `rows.Err()` before dependent writes/commit.
- [ ] Scan and terminal iterator failures roll back.
- [ ] Prefix-plus-error tests prove no partial leases or deactivations commit.
- [ ] Clean exhaustion and idempotent retry behavior remain unchanged.
- [ ] PostgreSQL-backed and race tests pass.

## Dependencies and coordination

- Coordinate the push query shape with **AV-025**.
- Run PostgreSQL failures through **AV-033** and static analysis through **AV-036** in the grouped release-gate plan.
- If **AV-037** moves search capabilities first, retain these tests at the new query boundary rather than postponing correctness.
- Lifecycle fixes in AV-002/AV-003/AV-010/AV-021 should rely on the corrected all-or-nothing subscription cleanup.

## References

- [Search foundation coding plan](../changes/2026-06-19-appview-search-foundation/04-coding-plan.md)
- [API error contract](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [pgx `Rows` documentation](https://pkg.go.dev/github.com/jackc/pgx/v5#Rows)
