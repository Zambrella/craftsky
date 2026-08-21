# AV-037 — Capability boundaries

- **Included finding:** AV-037, Low — Core API and storage files have grown beyond clear ownership boundaries
- **Priority/order:** Structural cleanup after behavioral contracts and the release gate are stable; adopt opportunistically at touched seams without mixing unrelated moves into fixes
- **Status:** Planned
- **Audit source:** [AV-037](../2026-08-12-appview-code-audit.md#av-037--core-api-and-storage-files-have-grown-beyond-clear-ownership-boundaries)

## Shared implementation strategy

Split by cohesive capability and invariant, not by line-count targets. First lock current behavior with route-inventory, wire-contract, SQL integration, transaction, and startup-wiring tests. Then introduce narrow consumer-owned interfaces and move code in small, behavior-neutral stages. Keep functions that share a transaction or state-machine invariant together even if the resulting file is larger than average.

The target is clearer ownership and test seams, not a new framework. Initially keep cohesive files in existing packages where moving packages would create import cycles; create a subpackage only after its dependency direction is one-way and its API is smaller than the implementation it hides. The public API, SQL/schema, configuration, status/error envelopes, cursor encoding, route policies, worker timing, and side-effect order do not change in this update.

Pre-production freedom permits breaking internal constructors and interfaces. Use it to remove service-locator-style concrete dependencies, not to change observable behavior.

## Finding closure

### AV-037 — Core API and storage files have grown beyond clear ownership boundaries

The audit found `api/post.go` at 2,158 lines, `api/post_store.go` at 1,681, `scheduledposts/store.go` at 976, `api/search_store.go` at 923, and `app/deps.go` at 750. Each combines multiple endpoint/query/lifecycle concerns, which hides small control-flow defects and makes focused testing/ownership difficult.

AV-037 closes when:

- Post create/read/comment/interaction/profile-list responsibilities have named, narrow handler/store capabilities.
- Search profile/hashtag/post/project/top-hashtag responsibilities are isolated behind focused interfaces with shared hydration kept explicit.
- Scheduled-post draft/resource mutations and publication state/effect transitions are separated without splitting a transaction invariant.
- Dependency construction is composed from named capability constructors with deterministic cleanup and no hidden global/service locator.
- Route registration is grouped by capability while the central policy table remains the admission source of truth.
- Large aggregate facades are removed or reduced to thin composition; package tests compile against the narrow interfaces they consume.
- Characterization tests prove no observable behavior, SQL meaning, state transition, or side-effect order changed.

## Proposed capability map

### HTTP and post capabilities

Keep handler-facing interfaces next to consumers in `internal/api`:

- **Post composition:** create-post validation, quote-target validation, lexicon body construction, synthetic response shaping.
- **Post lookup:** read by URI/DID/rkey, report/saved/share target resolution, quote hydration.
- **Conversation reads:** root comments, branch replies, around-focus reads, reply/comment shaping.
- **Interactions:** like/unlike/repost/unrepost authorization, active interaction lookup, write response shaping.
- **Author/profile feeds:** posts/projects/comments by author, pin promotion/exclusion, seek cursor wrapping.
- **Engagement hydration:** counts, viewer interaction/reply/saved state, quote views, batch summaries.

Likely behavior-neutral file destinations are `post_create.go`, `post_read.go`, `post_conversation.go`, `post_interactions.go`, `post_author_feed.go`, `post_response_shape.go`, with corresponding `*_store.go` files. File names are secondary to the capability boundaries.

### Search capabilities

- **Profile search** with its cursor/ranking projection.
- **Hashtag discovery** and top hashtags.
- **Post search** for chronological/popular/relevance modes.
- **Project search** and project-filter projection.
- **Shared search post hydration** via narrow `EngagementReader`, `QuoteReader`, and relationship capabilities rather than an embedded concrete `PostStore`.

Keep shared SQL predicates/scanners in one explicitly internal search-query support file only if two or more capabilities genuinely share them. Do not centralize unrelated queries merely to avoid duplication.

### Scheduled-post capabilities

- **Draft/resource repository:** create, list, get, update, delete, media/resource membership.
- **Publication preparation:** manual publication checks, frozen record/rkey allocation.
- **Publication state machine:** due claims, retry/failure/final state transitions.
- **Effect guard:** acquire/release and external-effect fencing.

`PrepareManualPublication`, `SaveFrozenRecord`, `AcquirePublishingEffect`, and `ClaimDue` must remain grouped according to their shared state/transaction invariants even if exposed through separate narrow interfaces.

### Dependency and route composition

- Split `newDeps` into named constructors for base infrastructure, auth/identity, content/read models, Instagram integration, scheduled posts, account deletion, push, and Tap/indexers.
- Each constructor returns a small capability bundle plus cleanup, and accepts explicit dependencies. A top-level composer owns cleanup order and all cross-capability wiring.
- Split `routes.AddRoutes` into registration functions for public/OAuth, migration, profile/relationship, search, notifications, scheduled posts, and posts. Each consumes only the bundle it needs.
- Keep `V1RoutePolicies` central and retain a mechanical route/policy coverage test so registration cannot bypass admission rules.

## Scope and design decisions

### In scope

- Behavior-neutral file/type moves, narrow interfaces, constructor decomposition, route registrars, and characterization/architecture tests.
- Removal of obsolete aggregate interfaces/fields after all consumers move.
- Package comments documenting ownership, invariants, and dependency direction.

### Out of scope

- SQL rewrites, new indexes/migrations, API/cursor/error changes, new configuration, performance tuning, or business-logic fixes.
- Moving generated lexicon types or editing `lexicon/`.
- Arbitrary maximum file-size linting.
- Introducing dependency-injection frameworks, generic repositories, ORMs, or mocks for every concrete type.

### Decisions

1. Consumers own interfaces; production stores satisfy them implicitly, with compile-time assertions only where useful.
2. Prefer typed AT Protocol identifiers across new seams and parse only at existing HTTP/Tap/auth boundaries.
3. Preserve `pgx`/SQL conventions. Transactional methods accept/own the transaction at the capability that defines the invariant; do not split a transaction across unrelated interface calls.
4. Preserve route patterns, middleware order, body/rate/auth/member policy, error codes/messages, JSON casing, and response status exactly.
5. No new package may import `internal/app`; composition points depend inward on capabilities, never back on the root wiring package.
6. Avoid a flag day. Land capability slices with old facade delegation temporarily, then delete the facade after call sites move. Do not maintain both indefinitely.

## Unified implementation plan

1. Capture a baseline from the release gate: full PostgreSQL/MinIO race suite, route-policy inventory, migration version, staticcheck, and wire-contract tests.
2. Add characterization tests for every exported behavior being moved: status/body/content type, opaque cursor round trips, SQL result ordering, moderation/relationship filtering, transaction rollback, worker state transitions, and startup cleanup.
3. Write a dependency map for the five audited files. Identify state/transaction invariants that must remain atomic and mark them as non-splittable units.
4. Define the narrow consumer-owned interfaces listed above without changing concrete implementations. Replace handlers' broad `*PostStore`/`*SearchStore` dependencies one capability at a time.
5. Split `api/post.go` by handler capability using pure moves first. Run focused tests after each move; do not combine a move with error-message or business-logic edits.
6. Split `api/post_store.go` by query capability. Keep shared row models/scanners/predicates in one documented support seam, then reduce sharing where it obscures ownership.
7. Replace `SearchStore`'s embedded concrete `PostStore` with explicit engagement/quote/relationship capabilities; split profile, hashtag, post, and project query implementations into cohesive files/types. Carry the AV-026 tests to the post-search query seam.
8. Decompose `scheduledposts.Store` behind draft/resource, publication, and effect-guard interfaces. Move code only after state-transition/lease/effect tests characterize ordering and rollback.
9. Introduce capability-specific dependency bundles and constructors in `internal/app` sibling files (for example auth, content, integrations, workers). The top-level composer remains the sole owner of database pool/logger/observer lifecycle and cleanup order.
10. Split `routes.AddRoutes` into registrars consuming narrow bundles. Retain one `AddRoutes` composer and exact `V1RoutePolicies` coverage; registration order must not change `ServeMux` matching.
11. Add architecture tests or a lightweight dependency check preventing capability packages from importing `internal/app`, handlers from depending on broad concrete stores, and route registrars from adding `/v1/*` routes without policy coverage.
12. Delete temporary facades, unused fields/functions, and forwarding methods once all call sites move. Use staticcheck to prove no abandoned path remains.
13. Run the full release gate and compare the baseline: route inventory, API golden/contract outputs, migration/schema fingerprint, SQL behavior, worker observations, and startup/shutdown smoke behavior must match.
14. Document the new ownership map and extension instructions in package comments/`appview/README.md`, including where a new endpoint, query, worker transition, or dependency constructor belongs.

## Migration, reconciliation, and operations plan

None. AV-037 is explicitly behavior- and schema-neutral. No migrations, data backfills, Tap replay, PDS reconciliation, token invalidation, or configuration changes are permitted in this update.

If a move appears to require schema or API behavior change, stop and implement that change under its own AV plan first. Because there are no production users, internal constructor breakage is acceptable, but runtime observations and operational controls remain identical.

Deploy as ordinary AppView builds after the full gate. Compare startup dependency creation/cleanup logs and worker health before/after; do not run old/new implementations in parallel against the same queues merely to validate a code move.

## API, client, configuration, and operational impact

- None by contract: route patterns, statuses, JSON casing/envelopes, cursors, Flutter behavior, configuration keys/defaults, schema, and SQL results remain identical.
- Internal Go constructors/interfaces may break freely; all call sites move in the same reviewed slice.
- Startup/shutdown logs and worker health should remain observationally equivalent except for bounded capability names added to diagnostics.
- Any proposed external/config/schema change is removed from this update and handled under its behavioral AV plan first.

## Security, failure, and race considerations

- Preserve authentication/member/rate/body middleware order and canonical envelopes while splitting routes.
- Preserve transaction ownership and lease/effect fencing; never replace an atomic store operation with multiple interface calls from a handler.
- Cleanup order must remain reverse dependency order and execute exactly once on partial-construction failure.
- Narrow interfaces must not expose raw OAuth/PDS credentials, device tokens, private media, or logger payloads to capabilities that did not already own them.
- Concurrent worker tests must prove moves did not change claim ordering, cancellation, retry, or stale-owner fencing.
- Pure moves can still change Go initialization/import behavior; run startup and race tests after every slice.

## Unified test plan

1. **Characterization:** Record route patterns, policies, statuses, content types, JSON bodies/error codes, cursors, and deterministic store results before moving code.
2. **Interface/unit:** Compile focused fake capabilities against each consumer-owned interface; assert handlers need no unrelated methods.
3. **Database integration:** Run post, search, scheduled-post, moderation, notification, and lifecycle stores against PostgreSQL with result/order/rollback parity.
4. **Worker/concurrency:** Run scheduled, push, Instagram, account-deletion, and Tap-related race tests where dependency construction or shared stores move.
5. **Route architecture:** Assert every `/v1/*` registration has exactly one policy and that unknown/method behavior remains governed by the existing contract work.
6. **Startup/shutdown:** Inject constructor failures at each dependency stage and assert pool/client/worker cleanup order and no goroutine leak.
7. **Static architecture:** Check forbidden imports/dependency direction and require staticcheck/gofmt/vet cleanliness after facade deletion.
8. **End to end:** Start the full Compose stack, migrate/bootstrap Tap, exercise representative auth/search/post/scheduled/notification flows, and compare wire behavior to baseline.

## Traceability and acceptance criteria

### AV-037

- **Implementation seams:** `internal/api/post*.go`, `internal/api/search*.go`, `internal/scheduledposts/store*.go`, `internal/app/deps*.go`, and route registration files.
- **Verification seams:** existing per-capability tests plus new route inventory, dependency construction/cleanup, interface, and architecture tests.

- [ ] Each named post/search/scheduled/dependency capability has one documented owner and a narrow interface.
- [ ] Handlers and route registrars no longer depend on broad stores/bundles for unrelated methods.
- [ ] Transaction/state-machine/effect invariants remain inside one cohesive capability.
- [ ] `newDeps` is composed from named constructors with deterministic partial-failure cleanup.
- [ ] Every `/v1/*` route remains covered by the central policy table.
- [ ] No schema, SQL meaning, API wire shape, cursor, configuration, worker timing, or side-effect order changes.
- [ ] Temporary facades and dead forwarding methods are removed.
- [ ] Full PostgreSQL/MinIO race, static-analysis, migration, and Compose smoke gates match the baseline.
- [ ] Package/README ownership guidance explains where future capabilities belong.

## Dependencies and coordination

- Establish the **AV-012/AV-013/AV-033/AV-036** release gate before structural moves.
- Carry the corrected query and iterator seams from **AV-026/AV-027** into the final capability owners.
- Align AV-025's push claim/process/send extraction and AV-029's moderation outbox transaction with this map, but do not combine unrelated behavior changes into AV-037 commits.
- Complete API-contract changes such as AV-031/AV-032 before freezing the route characterization baseline, or baseline the intended corrected contract explicitly.
- Lexicon files are untouched; if later capability work requires a lexicon change, it is a separate ADR/skill-governed update.

## References

- [API architecture](../superpowers/specs/2026-04-21-appview-api-architecture-design.md)
- [API wire alignment](../superpowers/specs/2026-04-22-api-wire-alignment-design.md)
- [AppView architecture hardening coding plan](../changes/2026-06-28-appview-architecture-hardening/04-coding-plan.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Go package names](https://go.dev/blog/package-names)
