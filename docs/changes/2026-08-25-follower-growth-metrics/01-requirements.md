# Requirements: Follower Growth Metrics

## 1. Initial Request

Give Craftsky members an owner-only graph showing how their follower count has changed over the last 7 days, 30 days, and year.

The original draft proposed an immutable count-change ledger. Product review simplified the feature to one canonical follower-count snapshot per UTC day. Growth data may lag by up to 24 hours and does not need a live current count.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/migrations/000012_atproto_follows.up.sql` defines the active follow projection.
  - `appview/internal/api/profile_store.go` calculates the canonical current Craftsky follower count.
  - `appview/cmd/appview/main.go` contains existing in-process worker lifecycles.
  - `appview/internal/routes/routes.go` and `appview/internal/routes/policy.go` govern AppView routes and access policy.
  - `app/lib/settings/pages/settings_page.dart` provides the owner settings hierarchy.
  - `app/lib/profile/data/profile_api_client.dart`, `profile_repository.dart`, and `api_profile_repository.dart` provide the existing Flutter profile data path.
  - `app/lib/router/router.dart` and `route_locations.dart` define typed settings routes.
- Existing patterns:
  - Public follow records are written to PDSes and indexed into the AppView through Tap.
  - Flutter reads social data from authenticated `/v1/` AppView endpoints.
  - DIDs, not handles, are durable account identifiers.
  - AppView JSON uses camelCase and the standard `{error, message, requestId}` error envelope.
  - In-process AppView workers run immediately, poll or wait, respect cancellation, and use Postgres for duplicate safety.
  - Riverpod providers carry account-bound Flutter state, and typed `go_router` routes provide navigation.
- Current behavior:
  - Current follower counts include active follows where both accounts are current Craftsky members.
  - A member leaving or rejoining Craftsky can change counts even if their PDS follow remains unchanged.
  - No historical follower snapshots, growth API, chart dependency, or Growth page exists.
- Constraints discovered:
  - Active follows cannot reconstruct historical counts after the fact.
  - Product design intentionally de-emphasises popularity metrics on public profiles.
  - The indexed graph is Craftsky-scoped, not a complete global AT Protocol follower graph.
  - Daily snapshots capture observed balances, not the individual gains and losses between observations.
  - A missed historical snapshot cannot be reconstructed exactly without event history.
  - No lexicon or PDS record change is required because snapshots are derived AppView data.
- Test/build commands discovered:
  - `just test`
  - `just app-test`
  - `just app-analyze`

## 3. Clarifying Questions And Decisions

### Q1: What persistence model should provide history?

Answer: Store daily canonical follower-count snapshots.

Decision / implication: Do not maintain a follow/unfollow or count-change ledger. Store at most one observed follower count per profile per UTC date.

### Q2: How fresh must the graph be?

Answer: It may lag by up to 24 hours.

Decision / implication: The Growth endpoint returns the latest completed snapshot and does not calculate or overlay a live count from the active graph.

### Q3: Which timezone defines a snapshot date?

Answer: UTC.

Decision / implication: Use one canonical daily boundary. Do not accept a device timezone, re-bucket history, or add account timezone preferences.

### Q4: What follower-count definition applies?

Answer: Use the same canonical count as profile reads: active follows where both accounts are current Craftsky members.

Decision / implication: Membership changes are reflected automatically in the next snapshot without event-driven fan-out or historical membership markers.

### Q5: How long is history retained?

Answer: Indefinitely during ordinary operation.

Decision / implication: Do not prune snapshots through ordinary retention. Permanent Craftsky account deletion remains an exception and removes the owner's private growth snapshots.

### Q6: What happens when less history exists than the selected period?

Answer: Show the available partial range.

Decision / implication: Keep all period selectors available, identify when observations begin, and do not fabricate earlier points.

### Q7: How should the metric be labelled?

Answer: Use “Followers” with explanatory scope and freshness copy.

Decision / implication: Explain that metrics reflect Craftsky members and update daily.

### Q8: Which period is selected initially?

Answer: 30 days.

Decision / implication: Growth opens on the trailing 30-day view. The first release does not persist the selection.

## 4. Candidate Approaches

### Option A: Daily Canonical Snapshots

Summary: Once per UTC date, calculate every current member's canonical follower count with set-based SQL and store one observation per profile.

Pros:

- Matches the current profile count definition at capture time.
- Membership changes require no special event handling.
- No changes to follow or profile indexers.
- Idempotency is a simple unique profile/date constraint.
- Bounded API queries and straightforward Flutter models.

Cons:

- Data can lag by up to 24 hours.
- Intraday changes are not retained.
- A missed snapshot date cannot be reconstructed exactly.

Risks:

- A failed daily run creates a visible data gap unless operations detect and address it promptly.

### Option B: Lazy Snapshots

Summary: Record a count only when the member opens Growth or another selected authenticated surface.

Pros:

- No scheduled worker.
- Minimal writes for inactive accounts.

Cons:

- New or infrequent visitors receive sparse and unreliable history.
- Observation frequency depends on unrelated user behaviour.
- Missing dates cannot distinguish inactivity from operational failure.

Risks:

- Users may interpret sparse observations as complete daily history.

### Option C: Immutable Count-Change Ledger

Summary: Record every count-changing follow and membership transition and derive daily balances.

Pros:

- Preserves exact transitions and supports later gained/lost analytics.
- Can derive balances in different timezones.

Cons:

- Requires indexer transaction changes, membership fan-out, event idempotency, baselines, gaps, and reconciliation.
- Considerably more implementation and operational complexity than a daily graph requires.

Risks:

- Missed or duplicate transitions can permanently drift historical balances.

## 5. Recommended Direction

Recommended approach: Option A, daily canonical snapshots.

Why: The requested UI is a daily count-over-time graph and the user accepts a 24-hour delay. A daily set-based snapshot captures exactly the value the product displays, including membership effects, without retaining event history or changing firehose indexers. This is the smallest approach that gives every member dependable history regardless of whether they visit the Growth page.

## 6. Problem / Opportunity

Members cannot currently understand whether their Craftsky audience is generally growing, stable, or shrinking. A daily observation is sufficient to show that trend without introducing behavioural tracking, advertising analytics, or a complex relationship-event ledger.

## 7. Goals

- G-001: Let a current member view their follower-count trend for the last 7 days, 30 days, and year.
- G-002: Capture one dependable canonical observation per current member per UTC date.
- G-003: Keep snapshot counts consistent with Craftsky's current member-scoped follower-count definition at capture time.
- G-004: Present freshness, partial history, and missing observations honestly.
- G-005: Present the metric accessibly on compact and large Flutter layouts.

## 8. Non-Goals

- NG-001: Live or near-real-time follower metrics.
- NG-002: Exact follow/unfollow event history or intraday movement.
- NG-003: Global AT Protocol or Bluesky follower metrics.
- NG-004: Public analytics for another member's profile.
- NG-005: Algorithmic ranking, recommendations, reach scoring, advertising, or targeting.
- NG-006: Premium or business-account entitlement.
- NG-007: Historical reconstruction before snapshot activation or for a missed snapshot date.
- NG-008: Device-local, saved-timezone, hourly, or minute-level buckets.
- NG-009: Gained, lost, churn, or conversion summaries.
- NG-010: A new lexicon, PDS record, or client-side analytics vendor.
- NG-011: Browsing previous windows or choosing a custom date range.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Current Craftsky member | Authenticated owner of the metrics | View their own daily follower trend and understand its scope and freshness |
| Snapshot worker | AppView background process | Capture every current member's canonical count once per UTC date without duplicates |
| Flutter client | Authenticated Craftsky application | Request bounded owner metrics and present loading, missing-history, and error states |
| AppView operator | Operates snapshot and API services | Detect failed or delayed snapshot runs without exposing per-account data |

## 10. Current Behavior

The AppView stores active follow relationships and calculates a current Craftsky-scoped follower count during profile reads. There is no historical count persistence, owner growth endpoint, Settings destination, or chart.

## 11. Desired Behavior

An AppView worker attempts to capture one canonical follower-count observation for every current member on each UTC date. An authenticated owner endpoint returns the observations for an allowlisted trailing period without overlaying live graph data. Flutter exposes the series through a dedicated Settings page that defaults to 30 days, explains that values update daily, and handles partial or missing history without fabricating points.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Current Craftsky members shall be able to view how their own follower count changed over trailing 7-day, 30-day, and 1-year periods. | Delivers the requested owner growth insight. | Initial request | AC-001, AC-002 |
| BR-002 | Business | Must | Growth metrics shall describe Craftsky-member followers and shall not claim global AT Protocol coverage. | The AppView does not index the complete global graph. | Discovery | AC-003 |
| BR-003 | Business | Must | Growth shall show only trustworthy observations and shall not fabricate data before activation or on a missed snapshot date. | Exact past counts cannot be reconstructed from active state. | Discovery | AC-004, AC-005 |
| FR-001 | Functional | Must | The AppView shall persist at most one follower-count snapshot per profile DID per UTC date. | Provides bounded daily history with simple idempotency. | User answers Q1 and Q3 | AC-006, AC-007 |
| FR-002 | Functional | Must | Each snapshot shall contain the profile DID, UTC snapshot date, non-negative canonical follower count, and capture timestamp. | Defines the minimum trustworthy observation. | Discovery | AC-006 |
| FR-003 | Functional | Must | A background worker shall attempt snapshot capture immediately on startup and continue checking for the current UTC date, while inserting no more than one snapshot for a profile/date. | Handles restarts without requiring exact process start timing. | Codebase pattern | AC-007, AC-008 |
| FR-004 | Functional | Must | Each snapshot run shall calculate counts for current members with a set-based query using the same membership eligibility rule as profile reads. | Avoids per-profile queries and semantic drift. | User answer Q4 | AC-009, AC-010 |
| FR-005 | Functional | Must | A snapshot run shall commit its member observations atomically; a failed run shall not leave a partially captured date. | Partial daily coverage would be misleading. | Discovery | AC-011 |
| FR-006 | Functional | Must | Snapshot capture shall be duplicate-safe across retries, restarts, and concurrent AppView processes. | Future deployment changes must not create multiple daily observations. | Codebase constraint | AC-007, AC-012 |
| FR-007 | Functional | Must | A profile that joins after the day's successful snapshot may have no observation until the next UTC-date capture. | Up to 24 hours of lag is accepted. | User answer Q2 | AC-013 |
| FR-008 | Functional | Must | Membership departure, rejoin, follow, and unfollow effects shall appear only through the canonical count observed by a later snapshot; they shall not create event-driven analytics writes. | Keeps indexers and analytics decoupled. | User answers Q1, Q2, and Q4 | AC-014 |
| FR-009 | Functional | Must | The AppView shall expose an authenticated, current-member, owner-only `/v1/` endpoint accepting only `7d`, `30d`, or `1y`. | Provides a bounded API and avoids public-profile privacy rules. | Discovery | AC-015, AC-016 |
| FR-010 | Functional | Must | A valid owner request shall succeed even when no snapshots exist. The camelCase response shall always contain the selected period, UTC range, `availableFrom`, latest snapshot date and capture timestamp, latest follower count, signed net change, and chronological daily points. `availableFrom` and all latest/change fields shall be `null` when no snapshots exist; net change shall also be `null` when fewer than two observations exist. | Gives Flutter one deterministic contract for populated and no-history states. | Discovery | AC-015, AC-017 |
| FR-011 | Functional | Must | The 7-day and 30-day periods shall include the current UTC date and the preceding 6 or 29 UTC dates. The 1-year period shall begin on the same month and day one UTC calendar year earlier and include the current date; when the current date is 29 February and the prior year has no 29 February, the start shall clamp to 28 February. | Defines deterministic bounded ranges, including leap years. | User answer Q3 | AC-018 |
| FR-012 | Functional | Must | The endpoint shall return a non-null point only when a snapshot exists for that profile/date and shall return `count: null` for an in-range date without an observation. | Avoids fabricating carry-forward or interpolated counts. | Discovery | AC-004, AC-019 |
| FR-013 | Functional | Must | The latest follower count shall come from the latest persisted snapshot and shall not be replaced or compared with a live active-graph count during the request. | A 24-hour lag is accepted and removes live-data complexity. | User answer Q2 | AC-020 |
| FR-014 | Functional | Must | Within the selected UTC response range, with at least two non-null observations, net change shall equal the latest count minus the earliest count; with fewer than two, net change shall be unavailable rather than zero. `availableFrom` outside the selected range shall not affect this calculation. | Provides a deterministic period summary without claiming no change when no comparison exists. | Discovery | AC-021 |
| FR-015 | Functional | Must | `availableFrom` shall be the owner's earliest persisted snapshot date across all retained history, regardless of the selected period, or `null` when the owner has no snapshots. If less history exists than the selected period, the endpoint and UI shall show the trustworthy partial range from that date without fabricating earlier values. A leading in-range gap for an owner whose `availableFrom` predates the range remains an explicit missing observation, not a new availability date. | New accounts should still use all selectors and distinguish first-ever history from later gaps. | User answer Q6 | AC-022 |
| FR-016 | Functional | Must | Flutter shall add an owner Growth destination under Settings and preserve typed compact and large-screen navigation behavior. | Fits the existing information architecture. | Discovery | AC-023 |
| FR-017 | Functional | Must | The Growth page shall default to 30 days and show, when available, the latest observed follower count and signed net change or insufficient-history label, plus a daily line graph and controls for 7 days, 30 days, and 1 year. | Delivers the requested experience with a useful initial range while supporting no history. | Initial request, user answer Q8 | AC-001, AC-024, AC-025 |
| FR-018 | Functional | Must | The Growth page shall provide loading, retryable error, no-history, single-point, partial-history, missing-date, and populated-history states. | New members and failed snapshot dates are expected. | Discovery | AC-026 |
| FR-019 | Functional | Must | The Growth page shall label the metric “Followers” and display nearby copy that it reflects Craftsky members and updates daily. When a latest snapshot exists, the page shall also show its date or capture time; the no-history state shall not fabricate freshness metadata. | Keeps the label concise without overstating scope or freshness. | User answer Q7 | AC-003, AC-027 |
| FR-020 | Functional | Must | Growth data shall be loaded through account-bound Flutter state that cannot publish one account's response after the active account changes. | Metrics are owner-private in a multi-account app. | Discovery | AC-028 |
| RULE-001 | Business rule | Must | A snapshot count shall include a relationship only when both its follower and subject are current Craftsky members at capture time. | Must match the existing profile count. | Existing behavior | AC-009, AC-014 |
| RULE-002 | Business rule | Must | Snapshot dates shall use UTC and shall not be re-bucketed for device or account timezones. | One canonical daily observation minimises complexity. | User answer Q3 | AC-018, AC-029 |
| RULE-003 | Business rule | Must | Snapshots shall be retained indefinitely during ordinary operation. | Preserves available history and future range options. | User answer Q5 | AC-030 |
| RULE-004 | Business rule | Must | Permanent Craftsky account deletion shall remove the owner's private follower-growth snapshots. | Existing deletion policy removes owner-private AppView data. | Architecture constraint | AC-031 |
| RULE-005 | Business rule | Must | Growth metrics shall not affect feed ordering, recommendations, discovery visibility, moderation outcomes, or advertising. | Preserves Craftsky product principles. | Product vision | AC-032 |
| NFR-001 | Non-functional | Must | Snapshot writes shall be idempotent under repeated and concurrent capture attempts. | Restarts and future horizontal scaling must not duplicate data. | Discovery | AC-007, AC-012 |
| NFR-002 | Non-functional | Must | Growth data may lag the active graph by up to 24 hours under normal operation. | The user explicitly accepted daily freshness. | User answer Q2 | AC-020, AC-033 |
| NFR-003 | Non-functional | Must | The endpoint shall return no more than 367 daily points and shall not require pagination. | Supported ranges are intrinsically bounded. | Discovery | AC-018, AC-034 |
| NFR-004 | Non-functional | Must | The chart shall adapt without overflow at supported compact and large layouts and at the app's tested large-text scale. | The app supports multiple form factors and accessible text. | Discovery | AC-035 |
| NFR-005 | Non-functional | Must | The page shall expose a meaningful textual summary and chart semantics and shall not communicate trend or missing data through colour alone. | A visual chart alone is inaccessible. | Discovery | AC-036 |
| NFR-006 | Non-functional | Should | Flutter should format UTC snapshot dates and counts with the current locale without changing their UTC date identity. | Keeps storage deterministic and presentation readable. | Discovery | AC-037 |
| NFR-007 | Non-functional | Must | Operational logs and metrics shall not include DIDs, handles, per-account counts, or snapshot rows as metric dimensions. | Existing observability policy forbids sensitive and high-cardinality attributes. | Codebase constraint | AC-038 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-017 | Given a current member opens Growth, when any supported period is selected, then the available follower snapshot trend for that period is displayed. |
| AC-002 | BR-001 | Given the member selects 7 days, 30 days, or 1 year, when the request succeeds, then the selected period and displayed observations correspond. |
| AC-003 | BR-002, FR-019 | Given Growth is displayed, then it uses “Followers” and explains that counts reflect Craftsky members. |
| AC-004 | BR-003, FR-012 | Given no snapshot exists for an in-range date, then the response does not infer, interpolate, or carry forward a count for that date. |
| AC-005 | BR-003 | Given no trustworthy snapshot exists before activation, then the system does not fabricate prior history. |
| AC-006 | FR-001, FR-002 | Given a profile is captured, then one row contains its DID, UTC date, non-negative canonical count, and capture timestamp. |
| AC-007 | FR-001, FR-003, FR-006, NFR-001 | Given repeated or concurrent capture attempts for the same profile/date, then at most one snapshot row exists. |
| AC-008 | FR-003 | Given the AppView starts and the current UTC date has not been captured, then the worker attempts capture without waiting for the next process restart. |
| AC-009 | FR-004, RULE-001 | Given active follows involving current and non-current accounts, when a snapshot runs, then each count exactly matches the canonical profile-count eligibility rule. |
| AC-010 | FR-004 | Given multiple current profiles, when capture runs, then their observations are produced through bounded set-based database work rather than one count query per profile. |
| AC-011 | FR-005 | Given snapshot persistence fails during a run, then no subset of member observations from that run remains committed. |
| AC-012 | FR-006, NFR-001 | Given two AppView processes attempt the same UTC-date capture, then database coordination and uniqueness produce one complete logical result. |
| AC-013 | FR-007 | Given a profile joins after the day's capture, then Growth may show no observation for that date and begins no later than the next successful UTC-date capture. |
| AC-014 | FR-008, RULE-001 | Given follows or memberships change between captures, then no analytics write occurs at event time and the next successful snapshot reflects the resulting canonical count. |
| AC-015 | FR-009, FR-010 | Given an authenticated current member requests their own growth with `7d`, `30d`, or `1y`, then the endpoint returns success whether the owner has populated, partial, or no snapshot history. |
| AC-016 | FR-009 | Given an unauthenticated, non-current, arbitrary-profile, or unsupported-period request, then access is denied or validation fails through the standard error envelope. |
| AC-017 | FR-010 | Given a successful request, then the camelCase body always contains period, UTC range, nullable `availableFrom`, nullable latest snapshot date, nullable latest capture timestamp, nullable latest count, nullable signed change, and chronological `{date, count}` points; given no snapshots, all nullable metadata fields are `null` and every in-range point has `count: null`. |
| AC-018 | FR-011, RULE-002, NFR-003 | Given each supported period, then UTC boundaries and maximum point counts are deterministic; specifically, a `1y` request on 2028-02-29 starts on 2027-02-28 and returns 367 inclusive daily points. |
| AC-019 | FR-012 | Given an in-range UTC date has no snapshot, then that date is present with `count: null` so Flutter can show an explicit gap. |
| AC-020 | FR-013, NFR-002 | Given the active graph changes after the latest snapshot, when Growth is requested, then the endpoint continues to return the latest persisted count without a live overlay. |
| AC-021 | FR-014 | Given at least two non-null points inside the selected UTC response range, then signed change equals that range's latest count minus its earliest count; given fewer than two, then signed change is null; an `availableFrom` date before the selected range does not enter the calculation. |
| AC-022 | FR-015 | Given snapshot history exists, then `availableFrom` equals the owner's earliest persisted snapshot date across all retained history regardless of selected period; given less history than the selected period, the selector remains usable and earlier dates are null; given no history, `availableFrom` is null. |
| AC-023 | FR-016 | Given compact or large navigation, when the member opens Growth from Settings and navigates back, then typed route, shell, and back-stack behavior are preserved. |
| AC-024 | FR-017 | Given the member changes the period, then the graph, summary, labels, and selected state update to the corresponding snapshot response. |
| AC-025 | FR-017 | Given the member opens Growth without selecting a period, then the 30-day control and corresponding data are selected. |
| AC-026 | FR-018 | Given loading, retryable failure, no history, one point, partial history, missing dates, or populated history, then the page renders the defined state without stale data or uncaught error. |
| AC-027 | FR-019 | Given Growth is displayed, then nearby text states that counts update daily; when a latest snapshot exists, its date or capture time is shown, and when no history exists no snapshot date or capture time is fabricated. |
| AC-028 | FR-020 | Given account A has an outstanding request and the app switches to account B, then account A's response is never rendered for account B. |
| AC-029 | RULE-002 | Given the same snapshot is viewed from devices in different timezones, then its API date identity remains the same UTC date. |
| AC-030 | RULE-003 | Given ordinary retention processing runs, then follower-growth snapshots are not pruned. |
| AC-031 | RULE-004 | Given permanent account deletion completes, then the deleted owner's growth snapshots can no longer be retrieved or found. |
| AC-032 | RULE-005 | Given snapshot data changes, then feed order, recommendations, discovery visibility, moderation, and advertising behavior remain unchanged. |
| AC-033 | NFR-002 | Given normal worker operation, then the latest available observation is no more than 24 hours behind the current time, allowing normal scheduling tolerance at the date boundary. |
| AC-034 | NFR-003 | Given the largest supported range, then the response is bounded and returned without a cursor or pagination flow. |
| AC-035 | NFR-004 | Given supported compact and large widths and tested large-text scaling, then controls, summary, and chart render without clipping or overflow. |
| AC-036 | NFR-005 | Given a screen reader or a user unable to distinguish chart colours, then the selected period, latest count, signed trend, range, freshness, and missing dates remain understandable. |
| AC-037 | NFR-006 | Given a non-default locale, then visible dates and counts use locale-aware formatting without changing UTC bucket identity. |
| AC-038 | NFR-007 | Given snapshot worker or API telemetry is emitted, then logs and metric dimensions satisfy existing identifier and cardinality privacy rules. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Worker retries the same UTC date | Existing profile/date rows are not duplicated or silently replaced. | FR-001, FR-006 |
| EC-002 | Two AppView processes capture concurrently | One complete logical daily result commits. | FR-005, FR-006 |
| EC-003 | Capture fails partway | The date receives no partial member result and failure telemetry is emitted. | FR-005, NFR-007 |
| EC-004 | AppView is down for an entire UTC date | That date remains `count: null`; the system does not reconstruct it later. | BR-003, FR-012 |
| EC-005 | Profile joins after daily capture | The profile may have no point until the next successful date. | FR-007 |
| EC-006 | Profile is not a member on capture date | No snapshot is created for that profile/date. | FR-004, RULE-001 |
| EC-007 | Profile rejoins later | Snapshotting resumes on the next successful date; prior missing dates remain null. | FR-007, FR-012 |
| EC-008 | Gain and loss occur between captures | Only the net canonical count observed at the next snapshot is retained. | FR-008 |
| EC-009 | Follower count does not change | Consecutive successful dates contain equal non-null counts. | FR-004 |
| EC-010 | Account has one snapshot | The latest count is shown, change is unavailable rather than zero, and the single-point state is accessible. | FR-014, FR-018 |
| EC-011 | Current date is 29 February | The 1-year range clamps its start to 28 February of the prior non-leap year and contains 367 inclusive daily points. | FR-011, NFR-003 |
| EC-012 | Latest snapshot is from yesterday | Growth displays that persisted value and its freshness without querying a live replacement. | FR-013, FR-019 |
| EC-013 | Account switches during request | Late data from the previous account is discarded. | FR-020 |
| EC-014 | Permanent account deletion | Owner snapshots are removed without deleting public follow records for analytics cleanup. | RULE-004 |

## 15. Data / Persistence Impact

- New table: Daily follower snapshots containing profile DID, UTC snapshot date, non-negative follower count, and capture timestamp.
- Constraints: Unique or primary key on `(profile_did, snapshot_date)` and an index supporting descending profile/date range reads.
- Changed fields: None required in `atproto_follows`; it remains the active projection.
- Migration required: Yes. The migration creates snapshot storage. It may seed one activation observation for existing current members but shall not fabricate earlier dates.
- Backwards compatibility: No production-user history exists. Existing APIs and follow semantics remain unchanged; the new endpoint and Flutter route are additive.
- Retention: Indefinite during ordinary operation, subject to permanent owner account deletion.
- Source of truth: PDS follows remain authoritative for relationships. `atproto_follows` and current membership determine the canonical count at capture time. Snapshots are authoritative only for historical observations.

## 16. UI / API / CLI Impact

- UI: Add an owner-only Settings row, Growth page, period selector, latest-observation summary, responsive line graph, daily-update copy, and complete asynchronous/history states.
- API: Add an authenticated owner-only route, expected to be `GET /v1/profiles/me/follower-growth?period=7d|30d|1y`, with bounded camelCase output and standard errors.
- CLI: None identified.
- Background jobs: Add one in-process, duplicate-safe daily snapshot worker using set-based Postgres capture.
- Flutter data flow: Add dedicated growth models and account-bound provider state through the existing profile API client/repository pattern rather than enlarging `Profile`.
- Dependencies: Chart-library selection is deferred to coding design; accessibility, maintenance, and platform support must be evaluated before adding a package.

## 17. Security / Privacy / Permissions

- Authentication: A valid Craftsky session token and device ID are required under existing `/v1/` middleware.
- Authorization: Only the authenticated current member may retrieve their own growth history. The API derives the owner DID from authentication and accepts no arbitrary target DID.
- Sensitive data: Growth history is owner-private derived AppView data. Responses and telemetry must not expose another account's snapshots or identifiers.
- Account boundaries: Cached/provider state must be isolated by account and invalidated or discarded across switches and logout.
- Abuse cases: Period values are allowlisted and response size is intrinsically bounded. The route uses the existing read rate class.
- Deletion: Permanent account deletion removes owner-private snapshots but does not introduce a generic PDS deletion path.

## 18. Observability

- Events: No client product-analytics event is required.
- Logs: Record bounded snapshot-run and API outcomes without DIDs, handles, counts, or raw request data.
- Metrics: Operational metrics may include run outcome, run duration, captured-profile count in a non-dimensional value, latest successful snapshot age, and query latency. Per-account values must not be sent to Sentry.
- Alerts: Production operations should alert when the latest successful snapshot age exceeds the accepted daily window. Exact thresholds belong in coding/operations design.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | A daily run fails or the AppView is down for a full UTC date. | Every affected profile has a permanent graph gap for that date. | Atomic runs, immediate startup attempts, bounded telemetry, and snapshot-age alerting. |
| RISK-002 | Snapshot SQL scans or writes poorly at scale. | The daily run may exceed its window or burden Postgres. | Use one set-based query, appropriate indexes, performance tests, and monitor duration. |
| RISK-003 | “Followers” is interpreted as global AT Protocol followers. | Users misunderstand coverage. | Show persistent Craftsky-scope copy. |
| RISK-004 | A daily value is interpreted as live. | The Growth count appears inconsistent with a newer profile count. | Label it as updated daily and show latest snapshot metadata. |
| RISK-005 | Indefinite snapshots are lost with the AppView database. | History cannot be reconstructed from PDS current state. | Include snapshot tables in production backups and avoid promising portable analytics history. |
| RISK-006 | Chart rendering is inaccessible or overflows. | Members cannot use the feature reliably. | Require semantics, textual summaries, responsive constraints, and widget accessibility tests. |
| RISK-007 | Owner-private state survives an account switch. | One account's metrics may be shown to another account on the device. | Key state by account and test late-response isolation. |
| RISK-008 | A single atomic transaction becomes too large at future scale. | Capture may hold locks or fail as membership grows. | Keep the first implementation set-based; revisit run partitioning with explicit completeness tracking when measured scale requires it. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Craftsky launches with snapshot capture enabled before active production use. | Earlier history remains unavailable and must be represented as partial. |
| ASM-002 | The current Craftsky-member-only follower-count definition remains authoritative. | Snapshot SQL and product copy require redesign. |
| ASM-003 | One daily UTC observation is sufficient for the requested trend. | Event history, more frequent snapshots, or timezone-specific data would be needed. |
| ASM-004 | Growth remains available to all current members rather than a paid/business entitlement. | Authorization and UI visibility require entitlement rules. |
| ASM-005 | A missing daily point is acceptable and preferable to reconstructed or carried-forward data. | A more complex event ledger or checkpoint strategy would be required. |
| ASM-006 | Indefinite daily snapshot volume is acceptable at expected launch scale. | Retention or downsampling requirements would be needed. |
| ASM-007 | Trailing periods ending on the current UTC date are sufficient. | Date navigation and additional API parameters would be needed. |

## 21. Open Questions

- None blocking.
- Non-blocking for coding design: choose the exact daily worker cadence and polling interval while preserving at most one observation per profile/UTC date.
- Non-blocking for coding design: decide whether a successful empty run needs a separate run-status row for operations.
- Non-blocking for coding design: select a Flutter chart implementation after evaluating accessibility, maintenance, RTL, and supported platforms.

## 22. Review Status

Status: Draft

Risk level: Medium

Review recommended: Yes

Reviewer:

Date:

Notes: This revision supersedes the earlier ledger-specific draft. Review should focus on snapshot cadence, the 24-hour freshness statement, atomic all-member capture, permanent missing-date behavior, and latest-count copy.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-08-25-follower-growth-metrics/01-requirements.md`
- Next test specification: `docs/changes/2026-08-25-follower-growth-metrics/02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-003`, `FR-001` through `FR-020`, `RULE-001` through `RULE-005`, and `NFR-001` through `NFR-005` plus `NFR-007`.
- Suggested test levels: snapshot store/worker unit tests, set-based SQL integration and transaction tests, worker lifecycle tests, API handler/route policy tests, Flutter model/repository/provider tests, route tests, and responsive/accessibility widget tests.
- Blocking open questions: None.
