# AppView code-audit remediation plans

- Audit source: [AppView code audit](../2026-08-12-appview-code-audit.md)
- Audit snapshot: `7615d1774fef9e601e5024693573fdd93b3181d5`
- Plan status: Core remediation implemented; automated gates pass; final
  implementation review is Approved with notes; explicitly external/destructive
  release gates remain
- Scope: Grouped implementation plans covering every audit finding, AV-001 through AV-037

## How to use this directory

Findings that share an implementation boundary are deliberately grouped into one update document. This keeps one contract, migration, replay strategy, and test matrix together instead of prescribing overlapping changes in several files. Every AV ID still has explicit traceability and acceptance criteria inside its grouped plan.

A grouped plan is complete only when every included finding's acceptance criteria and test evidence are satisfied; changing code without the required replay, migration, fault test, or client update does not close the finding.

The grouped documents retain their original design-time sequencing and
acceptance matrices. The current execution record is
[`05-implementation-plan.md`](05-implementation-plan.md), and the final
`Approved with notes` verdict is recorded in
[`06-implementation-review.md`](06-implementation-review.md). Where a
design-time checkbox or status line was intentionally left as historical
planning context, those two execution artifacts are authoritative for the
implemented tree and its remaining release gates.

The app has no production users, so these plans deliberately prefer correcting contracts and rebuilding local state over compatibility layers. That freedom does not relax correctness: migrations must still support up/down/up verification, public records remain PDS-owned, private data remains AppView-owned, and destructive PDS writes stay limited to the explicit owner-authorized CraftSky deletion boundary.

Before implementing a plan:

1. Re-check its cited seams against the current branch because the audit is tied to the snapshot above.
2. Confirm that prerequisite AV plans have landed or combine the work in one deliberately scoped change.
3. Write the failure-focused tests first for security, ordering, concurrency, and lifecycle defects.
4. Implement the smallest coherent contract correction, including schema, client, configuration, and operations changes where named.
5. Run the plan-specific checks plus the repository-wide quality gate established by AV-033 and AV-036.
6. Record verification evidence in the eventual implementation artifact or pull request; do not mark a checkbox from code inspection alone when the criterion calls for an executed test.

## Cross-cutting implementation rules

- Use one hardened outbound HTTP stack for AV-001 and AV-017; do not create endpoint-specific SSRF or timeout variants.
- Treat membership departure, terminal DID deletion, OAuth handoff/finalization, bearer revocation, and external-effect fencing as one coordinated lifecycle model across AV-002, AV-003, AV-006, AV-007, AV-008, AV-009, AV-010, AV-011, AV-018, AV-019, AV-020, AV-021, and AV-035. The three grouped documents land together because each consumes the same owner/session transition boundary.
- Design AV-004 and AV-005 together so retry classification, durable quarantine, replay, and order-independent indexing form one recovery story.
- Keep authentication errors distinct from dependency failures across AV-010, AV-011, AV-019, AV-020, and AV-021.
- Make AV-033 and AV-036 early enabling work even though their audit severity is lower; every other fix benefits from a required database, race, formatting, static-analysis, and vulnerability gate.
- Prefer capability-based package boundaries from AV-037 while touching large files, but avoid mixing broad moves with behavioral fixes unless the move is required to expose a testable seam.
- The product owner resolved both lifecycle contract conflicts before
  implementation: AV-007 uses strict background-write removal with private
  suggestions and explicit current-member acceptance, and AV-006 permits
  minimal exact-key non-secret safety tombstones until uncertain effects
  converge. The lifecycle plan and product workflow artifacts record those
  approved branches.

## Recommended sequence

1. **Verification and release gates:** implement the complete AV-012/AV-013/AV-033/AV-036 plan, then establish a clean reproducible baseline.
2. **Configuration and outbound network boundaries:** implement the complete AV-022/AV-023/AV-024/AV-030, AV-001/AV-017, and AV-016 plans as one coordinated security phase. Share configuration primitives without splitting any grouped plan's completion contract.
3. **Authentication and owner-lifecycle foundation:** first approve and document AV-007's strict-removal or accepted-residual branch and explicit deletion's temporary-safety-tombstone or verified-settlement/staged branch, then jointly implement the complete AV-002/AV-003/AV-006/AV-007, AV-008/AV-018/AV-019, and AV-009/AV-010/AV-011/AV-020/AV-021/AV-035 plans. One coordinated landing owns `owner_lifecycles`, canonical owner-effect fences, parent/child session states, callback/handoff finalization, lifecycle-bound deletion capability, and the explicit route access classes; no one of these three grouped contracts is considered complete against a placeholder owned by another.
4. **Inbound HTTP admission and routing:** implement the complete AV-014/AV-015/AV-031/AV-032 plan against the explicit access classes established in phase 3, while retaining the configuration validation and timeout geometry from phase 2.
5. **Projection durability:** implement the complete AV-004/AV-005 plan, followed by its controlled non-production rebootstrap and reconciliation.
6. **Persistence and worker correctness:** implement the complete AV-026/AV-027, AV-025, AV-028/AV-034, and AV-029 plans, coordinating migration numbers and shared lease conventions.
7. **Structural cleanup:** implement AV-037 only after the corrected behavioral contracts and required quality gate are stable.

This ordering is a dependency guide. A grouped document describes one coherent update and should normally be implemented as a unit when splitting it would create competing intermediate contracts.

## Plan index

| ID | Severity | Finding | Plan |
|---|---|---|---|
| AV-001 | Critical | Public OAuth login permits SSRF through discovered endpoints | [Federated HTTP boundary](AV-001_AV-017-federated-http-boundary.md) |
| AV-002 | High | Terminal DID deletion leaves a ghost account and private/public data | [Account lifecycle](AV-002_AV-003_AV-006_AV-007-account-lifecycle.md) |
| AV-003 | High | Departed members retain usable sessions and member-write access | [Account lifecycle](AV-002_AV-003_AV-006_AV-007-account-lifecycle.md) |
| AV-004 | High | Tap ACKs transient indexer failures permanently after six deliveries | [Tap ingestion durability](AV-004_AV-005-tap-ingestion-durability.md) |
| AV-005 | High | Order-dependent index gates permanently lose posts and interactions | [Tap ingestion durability](AV-004_AV-005-tap-ingestion-durability.md) |
| AV-006 | High | Account deletion can leave an untracked private-media object | [Account lifecycle](AV-002_AV-003_AV-006_AV-007-account-lifecycle.md) |
| AV-007 | High | Automatic-follow can write publicly after departure or deletion | [Account lifecycle](AV-002_AV-003_AV-006_AV-007-account-lifecycle.md) |
| AV-008 | High | A long-lived bearer crosses a claimable custom URL scheme | [OAuth handoff](AV-008_AV-018_AV-019-oauth-handoff.md) |
| AV-009 | High | Concurrent token refresh can revoke a newly valid OAuth session | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-010 | High | Logout leaves the bearer valid when auxiliary cleanup fails | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-011 | High | Authentication-store outages are returned as 401 and erase client sessions | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-012 | High | Missing or unreadable migrations are treated as successful migration | [Verification and release gates](AV-012_AV-013_AV-033_AV-036-verification-release-gates.md) |
| AV-013 | High | The build contains 19 reachable known vulnerabilities | [Verification and release gates](AV-012_AV-013_AV-033_AV-036-verification-release-gates.md) |
| AV-014 | High | HTTP admission permits slow and large unauthenticated resource exhaustion | [HTTP admission and routing](AV-014_AV-015_AV-031_AV-032-http-admission-routing.md) |
| AV-015 | High | Rate limiting is bypassable and grows memory and auth rows without bound | [HTTP admission and routing](AV-014_AV-015_AV-031_AV-032-http-admission-routing.md) |
| AV-016 | High | Scheduled image validation permits decompression-bomb OOM | [Media decode safety](AV-016-media-decode-safety.md) |
| AV-017 | High | Untrusted OAuth/PDS responses have no overall deadline or size cap | [Federated HTTP boundary](AV-001_AV-017-federated-http-boundary.md) |
| AV-018 | Medium | Loopback OAuth handoff is deterministically lost | [OAuth handoff](AV-008_AV-018_AV-019-oauth-handoff.md) |
| AV-019 | Medium | Partial OAuth callback failure retains unreachable upstream credentials | [OAuth handoff](AV-008_AV-018_AV-019-oauth-handoff.md) |
| AV-020 | Medium | Configured OAuth expiry does not expire bearer-only access | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-021 | Medium | “All devices” logout leaves other parent OAuth credentials active | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-022 | Medium | OAuth JWKS metadata is derived from the request Host header | [Configuration hardening](AV-022_AV-023_AV-024_AV-030-configuration-hardening.md) |
| AV-023 | Medium | Production OAuth configuration fails open to localhost mode | [Configuration hardening](AV-022_AV-023_AV-024_AV-030-configuration-hardening.md) |
| AV-024 | Medium | Default development deployment exposes DID impersonation to the LAN | [Configuration hardening](AV-022_AV-023_AV-024_AV-030-configuration-hardening.md) |
| AV-025 | Medium | Push lease geometry allows duplicate provider sends | [Push delivery leases](AV-025-push-delivery-leases.md) |
| AV-026 | Medium | Search query errors are shadowed and can become request panics | [pgx error handling](AV-026_AV-027-pgx-error-handling.md) |
| AV-027 | Medium | Two transactional row loops omit terminal iterator errors | [pgx error handling](AV-026_AV-027-pgx-error-handling.md) |
| AV-028 | Medium | Cascading foreign keys lack usable supporting indexes | [Index maintenance](AV-028_AV-034-index-maintenance.md) |
| AV-029 | Medium | Moderation persistence and restoration scheduling are not atomic | [Moderation outbox](AV-029-moderation-outbox.md) |
| AV-030 | Medium | Operational duration settings accept zero and negative values | [Configuration hardening](AV-022_AV-023_AV-024_AV-030-configuration-hardening.md) |
| AV-031 | Medium | Browser PATCH calls fail CORS preflight | [HTTP admission and routing](AV-014_AV-015_AV-031_AV-032-http-admission-routing.md) |
| AV-032 | Medium | Unknown routes and method mismatches violate the JSON error contract | [HTTP admission and routing](AV-014_AV-015_AV-031_AV-032-http-admission-routing.md) |
| AV-033 | Medium | The default test path skips the database suite, while the DB-enabled suite fails | [Verification and release gates](AV-012_AV-013_AV-033_AV-036-verification-release-gates.md) |
| AV-034 | Low | Several indexes exactly duplicate uniqueness indexes | [Index maintenance](AV-028_AV-034-index-maintenance.md) |
| AV-035 | Low | Session throttle maps grow for the lifetime of the process | [Session integrity](AV-009_AV-010_AV-011_AV-020_AV-021_AV-035-session-integrity.md) |
| AV-036 | Low | Formatting and static-analysis hygiene is not enforced | [Verification and release gates](AV-012_AV-013_AV-033_AV-036-verification-release-gates.md) |
| AV-037 | Low | Core API and storage files have grown beyond clear ownership boundaries | [Capability boundaries](AV-037-capability-boundaries.md) |

## Shared completion gate

After all plans are complete, perform a clean local database/object-store rebuild, run migrations up/down/up, rebootstrap Tap membership, replay or reindex public records, and verify that private data starts empty unless a plan explicitly seeds it. Then run the full database-backed race suite, `go vet`, the pinned staticcheck policy, `gofmt` verification, and `govulncheck` against the same Go toolchain and dependency graph used by the release image.
