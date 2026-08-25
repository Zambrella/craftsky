# Coding Plan: Follower Growth Metrics

## 1. Inputs

- Requirements: `docs/changes/2026-08-25-follower-growth-metrics/01-requirements.md`
- Tests: `docs/changes/2026-08-25-follower-growth-metrics/02-acceptance-tests.md`
- Document review: `docs/changes/2026-08-25-follower-growth-metrics/03-document-review.md`
- Review status: `03-document-review.md` records `Changes required`; DR-001 through DR-005 were subsequently resolved in `01-requirements.md` and `02-acceptance-tests.md`. The user explicitly skipped another document-review pass and directed work to coding planning.
- Architecture reference: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Risk level: Medium

## 2. Implementation Strategy

Implement the feature database-outward so every later layer depends on a tested, deterministic snapshot contract.

1. Add a reversible migration containing a canonical follower-count view, private daily snapshots, and a date-level successful-run ledger.
2. Add a focused `internal/followergrowth` package for period math, series shaping, atomic capture, bounded reads, and the daily worker. Keep it independent from HTTP and Flutter concerns.
3. Reuse the canonical count view in existing profile reads so current profile counts and captured counts cannot drift semantically.
4. Coordinate capture with a transaction-scoped PostgreSQL advisory lock and a successful-run row. The snapshot rows and run row commit together, including a successful zero-member run.
5. Run capture immediately at AppView startup, then at each UTC midnight. Retry failed current-date runs after five minutes. The completed-run ledger makes restarts and concurrent processes cheap and duplicate-safe.
6. Add one current-member read route, `GET /v1/profiles/me/follower-growth?period=7d|30d|1y`, with a bare camelCase response and canonical error envelope.
7. Add account-keyed Flutter data access and a typed Settings route. Keep period selection local to the page because the first release deliberately does not persist it.
8. Render the line graph with `fl_chart`'s `LineChart`, `LayoutBuilder`, and an explicit `Semantics` wrapper. Use `FlSpot.nullSpot` for honest gaps and keep textual summaries outside the package renderer for RTL-independent chronology, large text, and accessibility.

No lexicon, PDS record, firehose indexer, pagination, cache, or client analytics vendor is needed. Flutter adds `fl_chart` as the sole feature dependency.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Persistence | Numbered reversible PostgreSQL migrations | Add canonical count view, snapshot table, and completed-run table in migration `000060` | FR-001, FR-002, FR-004, FR-005, FR-006, RULE-003 | IT-001, IT-002, IT-004, IT-010 |
| Canonical counts | Profile read embeds member-scoped follower SQL | Read profile follower count and all-member snapshot counts from one database view | FR-004, RULE-001 | IT-002, REG-001 |
| Snapshot domain | No historical follower module | Add period, store, series, and worker types under `internal/followergrowth` | FR-003 through FR-015, NFR-001 through NFR-003 | UT-001 through UT-005, IT-001 through IT-009 |
| Worker runtime | In-process cancellable workers in `cmd/appview/main.go` | Wire one immediate/midnight worker into dependencies and bounded shutdown | FR-003, NFR-002, NFR-007 | AT-007, UT-005, IT-005, IT-013 |
| API | Capability bundles, policy catalogue, narrow handler interfaces | Add owner-only current-member read handler and route policy | FR-009 through FR-015 | AT-004, IT-007 through IT-009 |
| Owner deletion | Exhaustive terminal DID inventory and bounded purge | Register snapshot owner DID for deletion; keep the fail-closed schema scan limited to persisted base tables because the canonical view stores no DID | RULE-004 | IT-011, REG-005 |
| Observability | `Observer` plus bounded `MetricRecorder` attributes | Record run outcome/duration, captured count as a value, and completed-run age without account dimensions | NFR-002, NFR-007 | UT-012, IT-013 |
| Flutter data | `ProfileApiClient`, repository interface, account-Dio family providers | Add growth model/method and account-keyed read provider | FR-010, FR-020 | UT-006, UT-008, AT-005 |
| Flutter navigation | Typed `go_router` Settings children | Add `/profile/settings/growth` route and Settings disclosure row | FR-016 | AT-006, IT-012 |
| Flutter UI | Localized Material pages, Riverpod `AsyncValue`, responsive tests | Add Growth page, period control, summaries, states, and an `fl_chart` line chart | BR-001 through BR-003, FR-017 through FR-019, NFR-004 through NFR-006 | AT-001 through AT-003, AT-006, UT-007, UT-009 through UT-011 |
| Architecture boundaries | Executable route/import inventories | Assert follower growth is absent from feed, ranking, discovery, moderation, advertising, and public profiles | RULE-005 | REG-003, REG-004 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000060_follower_growth_snapshots.up.sql` | Create | Create canonical count view, snapshots, completed-run ledger, constraints, and indexes. | FR-001, FR-002, FR-004, FR-005, NFR-001 | IT-001 through IT-004 |
| `appview/migrations/000060_follower_growth_snapshots.down.sql` | Create | Drop growth objects in dependency-safe order. | FR-001 | IT-001 |
| `appview/internal/db/follower_growth_migration_test.go` | Create | Exercise up/down/reapply and preserve unrelated data. | FR-001, FR-002 | IT-001 |
| `appview/internal/followergrowth/period.go` | Create | Parse allowlisted periods and calculate inclusive UTC ranges, including February 29 clamping. | FR-011, RULE-002, NFR-003 | UT-001 |
| `appview/internal/followergrowth/series.go` | Create | Fill daily points, calculate global availability metadata and selected-range net change. | BR-003, FR-010, FR-012, FR-014, FR-015 | UT-002, UT-003 |
| `appview/internal/followergrowth/store.go` | Create | Atomically capture a date and read bounded owner history. | FR-001 through FR-015, NFR-001 through NFR-003 | IT-001 through IT-004, IT-006, IT-008 |
| `appview/internal/followergrowth/worker.go` | Create | Immediate capture, next-midnight scheduling, five-minute failure retry, cancellation, and observations. | FR-003, NFR-002, NFR-007 | AT-007, UT-005, IT-005, IT-013 |
| `appview/internal/followergrowth/*_test.go` | Create | Unit, real-Postgres, concurrency, query-shape, retention, and privacy coverage. | FR-001 through FR-015, RULE-001, RULE-003, NFR-001 through NFR-003, NFR-007 | UT-001 through UT-005, UT-012, IT-001 through IT-006, IT-008, IT-010, IT-013 |
| `appview/internal/api/profile_store.go` | Change | Replace embedded follower-count subquery with the canonical view while preserving response behavior. | FR-004, RULE-001 | IT-002, REG-001 |
| `appview/internal/api/follower_growth.go` | Create | Define response DTOs, strict period validation, owner DID extraction, response encoding, and safe errors. | FR-009 through FR-015, NFR-003, NFR-007 | UT-004, IT-007 through IT-009 |
| `appview/internal/api/follower_growth_*_test.go` | Create | Response, handler, and observability unit coverage. | FR-009, FR-010, NFR-007 | UT-004, UT-012 |
| `appview/internal/routes/policy.go` | Change | Add current-member, read-rate, no-body route policy. | FR-009 | AT-004, IT-007 |
| `appview/internal/routes/routes_profile_notification.go` | Change | Register the growth handler through the profile capability bundle. | FR-009 | IT-007 |
| `appview/internal/routes/dependencies.go` | Change | Expose the narrow growth store dependency to route composition. | FR-009 | IT-007 |
| `appview/internal/routes/routes.go` | Change | Pass the wired growth store into the profile route bundle. | FR-009 | IT-007 |
| `appview/internal/routes/follower_growth_route_test.go` | Create | Test policy, auth, owner scope, no-history success, periods, live-overlay exclusion, and errors. | FR-009, FR-010, FR-013 | AT-004, IT-007, IT-009, REG-004 |
| `appview/internal/routes/follower_growth_architecture_test.go` | Create | Enforce private read boundaries and non-interference inventory. | RULE-005 | REG-003, REG-004 |
| `appview/internal/app/deps_content.go` | Change | Construct the growth store with other AppView read models. | FR-003, FR-009 | IT-005, IT-007 |
| `appview/internal/app/deps.go` | Change | Add growth store/worker fields and wire runtime dependencies. | FR-003, FR-009 | IT-005, IT-007 |
| `appview/cmd/appview/main.go` | Change | Start and await the growth worker using the common process context. | FR-003, NFR-002 | IT-005 |
| `appview/cmd/appview/follower_growth_worker_test.go` | Create | Prove startup, midnight rollover, retry, and shutdown wiring. | FR-003, NFR-002 | IT-005 |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Add `follower_growth_snapshots.profile_did` as an owner-delete role. | RULE-004 | IT-011, REG-005 |
| `appview/internal/ownerlifecycle/terminal_inventory_integration_test.go` | Change | Restrict persisted-DID discovery to base tables so the derived canonical count view is not misclassified as purgeable storage. | RULE-004 | IT-011, REG-005 |
| `appview/internal/ownerlifecycle/follower_growth_purge_integration_test.go` | Create | Prove permanent owner deletion removes snapshots only. | RULE-004 | IT-011, REG-005 |
| `appview/internal/observability/follower_growth.go` | Create | Add bounded worker observations. | NFR-002, NFR-007 | UT-012, IT-013 |
| `appview/internal/observability/metric_recorder.go` | Change | Add run duration/count/age recording to noop, in-memory, and Sentry recorders. | NFR-007 | UT-012, IT-013 |
| `app/lib/profile/models/follower_growth.dart` | Create | Define period, date-only points, nullable metadata, range, and strict `fromMap`. | FR-010 through FR-015, RULE-002, NFR-006 | UT-006, UT-011 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Send period query and decode the bare growth response. | FR-009, FR-010 | IT-007, UT-006 |
| `app/lib/profile/data/profile_repository.dart` | Change | Add one read-only `fetchFollowerGrowth` method. | FR-020 | UT-008 |
| `app/lib/profile/data/api_profile_repository.dart` | Change | Delegate growth reads to `ProfileApiClient`. | FR-020 | UT-008 |
| `app/lib/profile/providers/profile_repository_provider.dart` | Change | Add an account-keyed growth repository binding using `accountDioProvider`. | FR-020 | AT-005, UT-008 |
| `app/lib/profile/providers/follower_growth_provider.dart` | Create | Add auto-disposed account-and-period `FutureProvider` family. | FR-020 | AT-005, UT-008 |
| `app/test/profile/fakes/fake_profile_repository.dart` | Change | Add programmable growth callback for provider/page tests. | FR-020 | UT-007, UT-008 |
| `app/lib/settings/pages/follower_growth_page.dart` | Create | Own local period selection and render all async/history states. | BR-001 through BR-003, FR-017 through FR-020 | AT-001 through AT-003, AT-005, UT-007 |
| `app/lib/settings/widgets/follower_growth_chart.dart` | Create | Configure `fl_chart` line spots/dots, explicit gaps, sampled labels, and one semantic summary. | FR-017, FR-018, NFR-004 through NFR-006 | AT-006, UT-009 through UT-011 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add the Growth disclosure under Connections. | FR-016 | IT-012 |
| `app/lib/settings/models/settings_row.dart` | Change | Add stable `growth` row identity and section descriptor entry. | FR-016 | IT-012 |
| `app/lib/router/route_locations.dart` | Change | Add `growthChild = 'growth'`. | FR-016 | IT-012 |
| `app/lib/router/router.dart` | Change | Add typed `FollowerGrowthRoute` under Settings on the authenticated shell navigator. | FR-016 | AT-006, IT-012 |
| `app/lib/router/router.g.dart` | Regenerate | Commit generated typed-route changes. | FR-016 | IT-012 |
| `app/lib/profile/providers/*.g.dart` | Regenerate | Commit generated account repository and growth provider code. | FR-020 | UT-008 |
| `app/lib/l10n/app_en.arb` | Change | Add Growth labels, scope/freshness copy, state copy, semantics, and formatting phrases. | BR-002, FR-018, FR-019, NFR-005 | AT-002, AT-003, UT-007, UT-010 |
| `app/lib/l10n/generated/*` | Regenerate | Commit Flutter localization output. | FR-019, NFR-006 | UT-011 |
| `app/test/profile/models/follower_growth*_test.dart` | Create | Test wire decoding, UTC identity, range validation, and locale formatting. | FR-010 through FR-015, RULE-002, NFR-006 | UT-006, UT-011 |
| `app/test/profile/providers/follower_growth_provider_test.dart` | Create | Test account-keyed loading and late-response isolation. | FR-020 | AT-005, UT-008 |
| `app/test/settings/follower_growth_page_test.dart` | Create | Test periods, all states, retry, copy, responsive layout, and graph updates. | BR-001 through BR-003, FR-017 through FR-019, NFR-004 | AT-001 through AT-003, UT-007, UT-009 |
| `app/test/settings/follower_growth_accessibility_test.dart` | Create | Test semantics, non-colour gap communication, and large text. | NFR-005 | AT-006, UT-010 |
| `app/test/router/settings_routes_test.dart` | Change | Add typed Growth navigation on compact and large layouts. | FR-016 | IT-012 |
| `app/test/profile/data/profile_api_client_test.dart` | Change | Assert exact growth path/query and decode populated/no-history payloads. | FR-009, FR-010 | IT-007, UT-006 |
| `app/test/settings/settings_page_test.dart` | Change | Assert the Growth row is present in Connections. | FR-016 | IT-012 |
| `app/pubspec.yaml` | Change | Add `fl_chart: ^1.2.0`; continue using existing `intl` for visible labels and semantics. | NFR-004 through NFR-006 | UT-009 through UT-011 |
| `app/pubspec.lock` | Regenerate | Lock the resolved `fl_chart` dependency graph. | NFR-004 through NFR-006 | UT-009 through UT-011 |

## 5. Services, Interfaces, And Data Flow

### Persistence Contract

Migration `000060` should create:

```text
VIEW craftsky_profile_follower_counts
  profile_did TEXT
  follower_count BIGINT

TABLE follower_growth_snapshots
  profile_did TEXT NOT NULL
  snapshot_date DATE NOT NULL
  follower_count BIGINT NOT NULL CHECK follower_count >= 0
  captured_at TIMESTAMPTZ NOT NULL
  PRIMARY KEY (profile_did, snapshot_date)

TABLE follower_growth_snapshot_runs
  snapshot_date DATE PRIMARY KEY
  completed_at TIMESTAMPTZ NOT NULL
  captured_profile_count BIGINT NOT NULL CHECK captured_profile_count >= 0
```

The view must return one row for every current `craftsky_profiles` member, including zero followers. Eligible follow rows join both subject and follower to current Craftsky membership and exclude terminal owners exactly as current profile counts do. `ProfileStore.Read` should select the requested profile's `follower_count` from this view.

The snapshot primary key supports owner/date range scans in either direction; add another index only if the real query-plan test demonstrates a need. Add a role-leading owner index if terminal-inventory tests require an explicit index beyond the primary key.

### Atomic Capture

```text
Store.Capture(ctx, snapshotDate, capturedAt) -> CaptureResult
  begin transaction
  acquire one transaction advisory lock for follower-growth capture
  if follower_growth_snapshot_runs contains snapshotDate:
    commit/read existing completion and return AlreadyCompleted
  INSERT INTO follower_growth_snapshots (...)
    SELECT profile_did, snapshotDate, follower_count, capturedAt
    FROM craftsky_profile_follower_counts
  INSERT follower_growth_snapshot_runs(snapshotDate, capturedAt, insertedRowCount)
  commit
```

Use one `INSERT ... SELECT` for all current members. Do not loop over DIDs. The advisory lock serializes process-equivalent runs; the snapshot/run primary keys remain defense-in-depth. Keeping the run row in the same transaction proves that a successful date is complete and records successful zero-member dates. A failure rolls back snapshots and run status together.

The lock is system-scoped and date-independent because only the current date is captured. Do not derive lock keys from DIDs or expose a generic locking facility.

### Period And Read Contract

```text
type Period string // 7d, 30d, 1y
ParsePeriod(raw string) (Period, error)
Period.Range(currentUTCDate) DateRange

Store.Read(ctx, ownerDID, DateRange) -> History
BuildSeries(History, DateRange) -> Growth
```

`Period.Range` works with UTC date-only values:

- `7d`: current date minus 6 days through current date.
- `30d`: current date minus 29 days through current date.
- `1y`: same month/day in the previous year through current date; clamp February 29 to February 28 when needed.

Use one bounded SQL statement with CTEs to return:

- Earliest snapshot date across all retained owner history (`availableFrom`).
- Latest persisted owner snapshot metadata/count across all retained history.
- Existing points inside the selected range ordered ascending.

Fill absent in-range dates in Go with null counts. Calculate net change only from the first and last non-null selected-range points. Global metadata outside the selected range never enters net change.

### HTTP Contract

The success body is bare JSON:

```json
{
  "period": "30d",
  "rangeStart": "2026-07-27",
  "rangeEnd": "2026-08-25",
  "availableFrom": "2026-06-01",
  "latestSnapshotDate": "2026-08-25",
  "latestCapturedAt": "2026-08-25T00:00:02Z",
  "latestFollowerCount": 42,
  "netChange": 5,
  "points": [
    {"date": "2026-07-27", "count": 37},
    {"date": "2026-07-28", "count": null}
  ]
}
```

For an owner with no snapshots, keep all keys present, set `availableFrom`, all `latest*` fields, and `netChange` to null, and return every requested date with `count: null`.

Handler outline:

```text
GET /v1/profiles/me/follower-growth
  require exactly one period query value
  parse only 7d | 30d | 1y
  derive owner DID from authenticated middleware context
  calculate range from injected/current UTC time
  read and shape owner history
  encode 200 response
```

Use `400 invalid_period` for missing, repeated, or unsupported period values. Use existing middleware errors for auth/device/current-member failures and `500 internal_error` for store failures. Do not accept a DID, handle, timezone, cursor, or range parameter.

### Worker And Observability

```text
Worker.Run(ctx)
  attempt current UTC date immediately
  on success/already-complete: wait until next UTC midnight
  on failure: record bounded error category and retry in 5 minutes
  stop promptly on context cancellation
```

Inject `Now` and timer creation through worker options for deterministic tests. Production defaults use `time.Now` and `time.NewTimer`.

The run ledger is the source for latest successful run age, including empty runs. Extend `MetricRecorder` with one follower-growth method that records:

- Low-cardinality attributes: `result` (`success`, `already_complete`, `error`) and bounded `error_category`.
- Values, never dimensions: duration, captured profile count, latest successful run age.

Logs use only component, operation, result, and category. They must not contain DIDs, handles, per-owner counts, dates tied to owners, or snapshot rows. Existing HTTP metrics cover route latency.

### Deletion And Retention

Add `follower_growth_snapshots.profile_did` to `terminalDIDInventory` as `deleteDID(..., "owner", "profile_did", "snapshot_date")`. The existing fail-closed schema test queries `information_schema.columns`, which also exposes ordinary views; join `information_schema.tables` and require `table_type = 'BASE TABLE'` so it continues inventorying persisted plaintext DID roles without treating the derived `craftsky_profile_follower_counts` view as deletable storage. Do not add the owner-free completed-run table or the derived view to the inventory. Ordinary retention code must not reference either table. Terminal purge removes the target owner's snapshots while unrelated owner snapshots and public follow projections retain existing behavior.

## 6. State, Providers, Controllers, Or DI

Use existing Riverpod code generation and account-specific Dio bindings.

```text
accountFollowerGrowthRepositoryProvider(AccountKey account)
  watches accountDioProvider(account)
  returns ApiProfileRepository(ProfileApiClient(accountDio))

followerGrowthProvider(AccountKey account, FollowerGrowthPeriod period)
  auto-disposed FutureProvider family
  watches accountFollowerGrowthRepositoryProvider(account)
  calls repository.fetchFollowerGrowth(period)

FollowerGrowthPage local State<FollowerGrowthPeriod>
  defaults to thirtyDays
  does not persist selection
```

The page derives `AccountKey` from `sessionRegistryProvider.value?.activeLease?.session.account` and watches only that account/period provider instance. A late account A future can complete only inside account A's provider key; once the page watches account B, it cannot publish A's value into B's widget state. Auto-disposal cancels presentation ownership when no listener remains.

Add `accountFollowerGrowthRepositoryProvider` and `followerGrowthProvider` to `accountStateInvalidatorProvider` so switch/logout invalidation removes private state even though family keys already isolate it. Do not use a global unkeyed fallback for Growth.

Repository signatures:

```text
ProfileApiClient.getFollowerGrowth(FollowerGrowthPeriod period)
ProfileRepository.fetchFollowerGrowth(FollowerGrowthPeriod period)
ApiProfileRepository.fetchFollowerGrowth(FollowerGrowthPeriod period)
```

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Navigation

- Add `SettingsRowId.growth` under the Connections section before Followers.
- Label the row `Growth`; use a neutral trend icon such as `Icons.show_chart`.
- Add `RouteLocations.growthChild = 'growth'`.
- Add typed `FollowerGrowthRoute` beneath `SettingsRoute` with the authenticated-shell parent navigator key.
- Canonical location: `/profile/settings/growth`.
- Back navigation returns to Settings on compact and large layouts without adding another rail/bar.

### Page Composition

```text
FollowerGrowthPage
  Scaffold
    AppBar("Growth")
    account/period AsyncValue switch
      loading -> centered StitchProgressIndicator
      error -> safe localized message + Retry button
      data -> scrollable content
        heading/metric label: Followers
        Craftsky scope + daily freshness copy
        SegmentedButton: 7 days | 30 days | 1 year
        latest observation summary or no-history summary
        net change or insufficient-history label
        FollowerGrowthChart
        partial/gap explanation when applicable
```

The latest summary uses global latest persisted metadata. The chart and net change use only the selected range. If all selected points are null but older global metadata exists, show “No observations in this period” rather than claiming the owner has no history.

### Chart

`FollowerGrowthChart` should:

- Use `LayoutBuilder` for available width and a bounded fixed chart height that does not scale with text.
- Build one `LineChartData` with explicit `minX = 0` and `maxX = points.length - 1` so leading and trailing missing dates retain their position.
- Map observed counts to `FlSpot(dayIndex, count)` and missing counts to `FlSpot.nullSpot`; this creates disconnected segments without inventing values across gaps.
- Set explicit `minY` and `maxY` from observed counts; pad a flat series equally above and below so it renders as a centered horizontal line.
- Configure `FlDotData` to show observed points, including isolated/single observations, and use line style plus gaps/dots rather than colour alone.
- Configure `FlTitlesData`/`SideTitles` with a small responsive set of sampled date and count labels. Build labels with localized `intl` formatters and reserve enough axis space at the tested text scale.
- Disable chart touch interaction and implicit animation for the first release; the chart is informational and period changes should render deterministically.
- Keep chronological plotting left-to-right even in RTL because time-series direction is data identity; constrain only the `LineChart` plot to LTR and mirror surrounding controls/text normally.
- Wrap the chart in one semantic node describing selected period, latest selected observation, signed/insufficient trend, UTC range as localized dates, freshness, and missing-observation count.
- Exclude `LineChart` renderer internals from semantics to avoid noisy or package-version-dependent nodes. The surrounding textual summary remains the accessible source of truth.
- Render the established empty/no-observations state instead of constructing `LineChartData` when every point is null.

Use existing `intl` `NumberFormat` and `DateFormat` for visible counts/dates. Parse API date-only strings into `DateTime.utc` and never call `toLocal()` when determining bucket identity.

### Localization

Add English ARB entries for:

- Settings row and page title.
- `Followers`, three period labels, latest count, signed change, insufficient history.
- Craftsky-member scope and daily update copy.
- Available-since, latest snapshot, no history, no observations in period, partial history, and missing-date count.
- Retryable load error and retry action.
- Chart semantics for populated, flat, single-point, and gapped states.

Regenerate localization output rather than editing generated files manually.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Worker starts before today's run | Attempt immediately, commit snapshots and run row together. | FR-003 | AT-007, UT-005, IT-005 |
| Today's run already completed | Return `already_complete`, emit bounded observation, wait until next midnight. | FR-003, FR-006, NFR-001 | IT-004, IT-005 |
| Two processes capture concurrently | Advisory transaction lock serializes; second observes completed run; unique keys defend against defects. | FR-005, FR-006, NFR-001 | IT-004 |
| Capture statement fails | Roll back snapshots and run row; log category only; retry after five minutes. | FR-005, NFR-007 | IT-004, IT-013 |
| Successful zero-member run | Commit run row with count zero so the date is complete and observable. | FR-003, NFR-007 | IT-005, IT-013 |
| AppView misses an entire date | Never backfill it; API emits null for that in-range date. | BR-003, FR-012 | AT-003, UT-002, IT-008 |
| Member joins after completed run | No row until next UTC date; no same-day event write. | FR-007, FR-008 | IT-006 |
| Missing/repeated/unsupported period | Return canonical `400 invalid_period` envelope. | FR-009 | AT-004, IT-007 |
| No owner snapshots globally | Return 200 with nullable metadata null and all requested points null; page shows no-history state. | FR-010, FR-018 | AT-003, UT-004, UT-007, IT-007 |
| No selected-range points but older history | Preserve global latest/`availableFrom`; chart shows no observations in period; net change null. | FR-014, FR-015, FR-018 | UT-002, UT-003, UT-007, IT-008 |
| One selected point | Show count/dot, insufficient-history change, and accessible single-point summary. | FR-014, FR-018 | AT-003, UT-003, UT-007, UT-010 |
| Interior missing dates | Break line segments, include null points, and show textual/semantic missing count. | FR-012, FR-018, NFR-005 | AT-003, UT-002, UT-010 |
| Active graph changes after snapshot | API continues returning persisted latest metadata/count. | FR-013 | IT-009 |
| February 29 current date | Clamp 1-year start to prior February 28; return 367 points. | FR-011, NFR-003 | UT-001, IT-008 |
| Account switch during request | Page switches provider key; late old-account result is not rendered; invalidator clears private state. | FR-020 | AT-005, UT-008 |
| API/load failure | Preserve no stale cross-period data; show localized retry action that invalidates current account/period provider. | FR-018, FR-020 | UT-007, UT-008 |
| Large text/compact width | Scroll page content; wrap textual labels; keep chart bounded and controls reachable. | NFR-004 | AT-006, UT-009 |
| Permanent owner deletion | Terminal inventory deletes owner snapshots; endpoint later fails at current-member middleware. | RULE-004 | IT-011, REG-005 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `internal/db/follower_growth_migration_test.go`, `internal/followergrowth/store_integration_test.go` | Apply migration in isolated Postgres; insert valid/invalid/duplicate rows. | Migration/tables/constraints do not exist. |
| 2 | IT-002, REG-001 | `internal/followergrowth/store_integration_test.go`, `internal/api/profile_store_test.go` | TD-001 mixed current/external graph and zero-follower member. | Canonical view/capture store do not exist. |
| 3 | IT-003 | `internal/followergrowth/query_plan_test.go` | Record database statements and inspect set-based SQL plan. | Capture implementation has not been written. |
| 4 | IT-004, NFR-001 evidence | `internal/followergrowth/store_integration_test.go` | Failure trigger and two independent pools racing one date. | Atomic run ledger/advisory coordination are absent. |
| 5 | UT-001 | `internal/followergrowth/period_test.go` | TD-003 ordinary/leap UTC dates. | Period parser/range type are absent. |
| 6 | UT-002, UT-003 | `internal/followergrowth/series_test.go` | TD-002 sparse/global history fixtures. | Series fill, global availability, and selected-range change are absent. |
| 7 | UT-005, IT-005 | `internal/followergrowth/worker_test.go`, `cmd/appview/follower_growth_worker_test.go` | Fake clock/timer, completed and failed runs, cancellation. | Worker/runtime wiring are absent. |
| 8 | IT-006, REG-002 | `internal/followergrowth/store_integration_test.go`, `internal/index/follower_growth_boundary_test.go` | Mutate follows/membership between fixed capture dates. | Later-snapshot-only behavior is not implemented/protected. |
| 9 | UT-004 | `internal/api/follower_growth_response_test.go` | Populated and all-null no-history DTOs. | Response DTO/JSON contract is absent. |
| 10 | AT-004, IT-007 | `internal/routes/follower_growth_route_test.go` | TD-004 auth, owner histories, invalid period forms. | Policy/handler/route are absent. |
| 11 | IT-008 | `internal/followergrowth/store_integration_test.go` | Sparse periods, global earliest row, leap day, no-history owner. | Bounded read and deterministic series are absent. |
| 12 | IT-009 | `internal/routes/follower_growth_route_test.go` | Persist snapshot then mutate active follow graph. | Endpoint does not exist. |
| 13 | IT-010 | `internal/followergrowth/retention_test.go` | Old snapshots plus ordinary retention inventories. | Retention non-participation is not executable. |
| 14 | IT-011, REG-005 | `internal/ownerlifecycle/follower_growth_purge_integration_test.go`, `internal/ownerlifecycle/terminal_inventory_integration_test.go` | TD-008 terminal owner and unrelated/public rows; migrated base-table/view schema. | Terminal DID inventory does not include snapshots, and the persisted-role scan misclassifies the new derived view. |
| 15 | UT-012, IT-013 | `internal/followergrowth/observability_test.go`, `internal/api/follower_growth_observability_test.go` | TD-009 capturing logs/metrics over success/failure/API paths. | Follower-growth observations are absent. |
| 16 | REG-003, REG-004 | `internal/routes/follower_growth_architecture_test.go`, existing profile response tests | Approved dependency inventory and public response fixtures. | Boundary inventory does not know the new feature. |
| 17 | UT-006 | `app/test/profile/models/follower_growth_test.dart`, API-client test | Populated/no-history camelCase maps and UTC zone variants. | Flutter model/client method are absent. |
| 18 | AT-005, UT-008 | `app/test/profile/providers/follower_growth_provider_test.dart` | TD-007 delayed A and immediate B repositories. | Account-keyed provider is absent. |
| 19 | IT-012 | `app/test/router/settings_routes_test.dart`, Settings page test | Production router harness at compact/large widths. | Growth row/typed route are absent. |
| 20 | AT-001, AT-002, UT-007 | `app/test/settings/follower_growth_page_test.dart` | TD-006 period-specific responses and all async/history states. | Growth page is absent. |
| 21 | AT-003 | Flutter page test plus Go sparse-series tests | No, one, partial, and gapped histories. | Honest state rendering is absent. |
| 22 | AT-006, UT-009, UT-010 | Page/accessibility widget tests | Compact/large, text scale 2, semantics, light/dark, and `LineChartData` containing `FlSpot.nullSpot` gaps. | `fl_chart` rendering, responsive configuration, and semantic summary are absent. |
| 23 | UT-011 | `app/test/profile/models/follower_growth_format_test.dart` | Fixed UTC dates/counts under default and non-default locales. | Locale formatting helpers/copy are absent. |
| 24 | REG-001 through REG-005 full pass | Focused Go regression targets | Existing API/indexer/feed/public/deletion fixtures. | Any accidental coupling or semantic drift is exposed. |
| 25 | MAN-001, MAN-002 | Release-candidate device checks | VoiceOver/TalkBack, grayscale, compact/large, RTL, 200% text. | Manual evidence remains outstanding until a runnable UI exists. |

Focused red-green commands should use package/file targets during implementation, followed by full gates:

```text
cd appview && TEST_DATABASE_URL=... go test ./internal/followergrowth -run TestName
just test
just appview-check
just app-test test/settings/follower_growth_page_test.dart
just app-test
just app-analyze
```

The real-Postgres tests require `just dev-d` before `just test`; `just appview-test-unit` is useful but cannot serve as evidence for persistence/concurrency acceptance.

## 10. Sequencing And Guardrails

- First TDD step: Write IT-001 against migration `000060` and prove it fails because snapshot storage does not exist; then add only enough migration SQL to pass schema invariants.
- Dependencies between work items: migration before store; canonical view before capture/profile alignment; store before worker/API; API model before Flutter client; account provider before page; page before chart polish/manual checks.
- Keep each red-green-refactor loop scoped to one listed test or tightly related group. Do not create all production modules before the first test passes.
- Run code generation only after annotated Riverpod/routes or ARB inputs change; commit generated outputs with their source changes during implementation if commits are requested then.
- Use typed `syntax.DID` at Go boundaries and inside domain/store signatures. PostgreSQL receives typed DIDs directly.
- Keep capture SQL set-based and one-statement. Do not hide a per-profile query behind a helper or goroutine fan-out.
- Use one statement snapshot for canonical counts and one transaction for rows plus completion ledger.
- Do not replace historical rows on conflict. A completed date is immutable.
- Do not reconstruct a missed date, capture a past date, or write analytics from follow/profile indexers.
- Do not expose arbitrary owner identifiers in the API, Flutter route, logs, metrics, or provider keys visible in diagnostics.
- Keep success JSON bare and camelCase; all errors use `{error,message,requestId}`.
- Do not add pagination to the bounded response.
- Add only `fl_chart` for chart rendering; do not add other chart, analytics, timezone, or persistence dependencies to Flutter.
- Preserve all unrelated worktree changes and the separate link-preview workflow directory.
- Out of scope: live counts, gained/lost events, custom ranges, saved period selection, global AT Protocol counts, public analytics, PDS/lexicon changes, ranking/recommendation inputs, and client product analytics.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | How is a successful empty date distinguished from an unattempted date? | Without a marker, the worker repeatedly scans and freshness metrics fail with zero members. | Add `follower_growth_snapshot_runs` and commit it atomically with snapshots. |
| CPQ-002 | Resolved | How are concurrent AppView processes coordinated? | Uniqueness alone can permit mixed logical runs under changing membership. | Use one transaction advisory lock, then check/insert the completed-run row; retain unique constraints. |
| CPQ-003 | Resolved | What is the exact worker cadence? | A loose poll could exceed daily freshness or repeatedly scan all members. | Attempt at startup, schedule next UTC midnight after success, retry current date every five minutes after failure. |
| CPQ-004 | Resolved | What powers latest successful age when no snapshots exist? | Snapshot-row age cannot represent a successful empty run. | Read completed-run ledger; report age as a metric value. |
| CPQ-005 | Resolved | Which chart package should be added? | A package can increase maintenance/accessibility risk, but avoids maintaining a bespoke renderer. | Use `fl_chart` `^1.2.0`, selected by user review. Keep accessibility in package-independent textual and semantic wrappers, pin through `pubspec.lock`, and cover gaps/responsiveness with widget tests. |
| CPQ-006 | Non-blocking | The all-member view and atomic insert may become expensive at future scale. | Long transactions or database load. | Protect set-based shape with IT-003, inspect benchmark/query plans, emit duration/count/age metrics, and revisit partitioned completeness only from measured evidence. |
| CPQ-007 | Non-blocking | Real screen-reader output varies by platform. | Widget semantics tests cannot prove spoken output everywhere. | Complete MAN-001 on one supported mobile platform before release. |
| CPQ-008 | Non-blocking | `03-document-review.md` retains its pre-revision verdict. | The workflow folder has a stale review artifact. | User explicitly skipped re-review. Treat revised `01`/`02` plus this instruction as the implementation contract; do not silently rewrite `03`. |

Blocking open questions: None.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-08-25-follower-growth-metrics/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-08-25-follower-growth-metrics/05-implementation-plan.md`
- Start with test: IT-001, migration and Postgres schema invariants for one non-negative snapshot per profile DID/UTC date plus a date-level run ledger.
- Focused command: start the disposable/dev Postgres with `just dev-d`, then run the new package test through `just test` until a narrower environment command is captured in `05-implementation-plan.md`.
- Full AppView evidence: `just appview-check`.
- Full Flutter evidence: `just app-test` and `just app-analyze`.
- Implementation mode: strict red-green-refactor using `implement-tdd`; do not batch untested production layers.
- Notes: Create no lexicon ADR, add only the planned `fl_chart` dependency, preserve owner-private account boundaries, and include generated Flutter files only when their annotated sources change.
