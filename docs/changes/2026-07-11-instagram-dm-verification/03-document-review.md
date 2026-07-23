# Document Review: Instagram DM Ownership Verification And Follow Discovery

## Verdict

Status: Approved with notes  
Reviewer: Codex workflow review plus independent re-review  
Date: 2026-07-23
Risk level: High

## Summary

The requirements and acceptance tests remain approved after the verified-link
lifetime simplification. The direct Meta integration, verified-only local
archive parsing, explicit same-DID confirmation, public-by-default
discoverability choice, exact safety-filtered suggestions, explicit PDS follow
acceptance, unlink cleanup, and first-class actorless notification remain
faithful to the approved design. Every Must requirement and all 48 acceptance
criteria have automated coverage. Real Meta behavior and current export shapes
remain correctly isolated as release gates rather than implementation blockers.

The initial review found thirteen issues. Requirements and tests were revised,
then independently re-reviewed. A narrow second pass found four additional
contract seams; those were also resolved and re-reviewed as approved.

## Initial Findings And Resolutions

| ID | Severity | Area | Resolution |
|---|---|---|---|
| DR-001 | Critical | Tests / Privacy | `AC-039`, `UT-015`, and test data now permit controlled wholly synthetic or explicitly approved redacted inputs while prohibiting real/user-derived fixtures and leakage outside intended private fields. |
| DR-002 | Critical | Permissions / Membership | `FR-030` and `AC-048` define one current-member guard for every authenticated route and worker transition, with `404 profile_not_found` and reversible inactivation. |
| DR-003 | Critical | Safety / Matching | `InstagramSuggestionEligibilityPolicy` explicitly covers membership, active/current verified link, discovery, exact imported username, self/follow, hide/takedown, blocks both ways, importer mute, fail-closed unavailable safety data, and final pre-PDS revalidation. The later following-only simplification removes direction from the trusted data model. |
| DR-004 | Critical | Data lifecycle | Reversible membership loss, explicit link/import deletion, terminal identity deletion, and future whole-account deletion are distinct. Instagram owner rows do not broadly cascade from `craftsky_profiles`. |
| DR-005 | Critical | API / State | §12.1 fixes public states, transitions, request/response bodies, status/error codes, pagination, idempotent results, conflict/unavailable shapes, and shared Go/Flutter golden contracts. |
| DR-006 | Important | Webhook / Privacy | §12.2 defines the exact minimal durable work item and explicitly excludes raw body, message text, plaintext challenge, signature, and unrelated payload data. |
| DR-007 | Important | Availability / Imports | Meta outage affects only new verification/profile/reply work. Verified-link imports are additive, listable, inspectable, independently deletable, retained without expiry, and multi-source suggestions preserve remaining support. |
| DR-008 | Important | Limits / Retention | §12.4 and §15 define fixed production maxima, rate/worker/provider boundaries, shared enforcement, trusted proxy behavior, and retention for every private record class. |
| DR-009 | Important | Notifications | §12.3 defines a checked `kind: social | system` union, exact actorless JSON, fixed five-minute coalescing, count cap, newness, triggers, retraction, and one-push behavior. |
| DR-010 | Important | Challenge | The design and requirements use a 30-symbol alphabet, 13 random symbols, approximately 63.8 bits, canonical grouping, exact-message grammar, and ASCII-case/outer-whitespace normalization. |
| DR-011 | Important | Traceability | Business requirements link all relevant criteria; automated audit confirms every Must row and `AC-001` through `AC-048` are covered. |
| DR-012 | Suggestion | Consistency | All confirmation language consistently uses the same authenticated DID, not the same session token/device. |
| DR-013 | Suggestion | Test order | Fail-closed config/shared limits and migration seams precede dependent routes; Go and Flutter share a synthetic golden wire corpus. |

## Re-review Findings And Resolutions

| ID | Severity | Area | Resolution |
|---|---|---|---|
| RR-001 | Critical | Membership restoration | Import-only members can explicitly reactivate each paused `membershipInactive` import after rejoin through PATCH. Link and import reactivation remain separate and never silently restore discovery. |
| RR-002 | Critical | DELETE semantics | Attempt, account, import, and suggestion DELETE operations always return privacy-preserving `204` for owned, foreign, absent, or purged identifiers and mutate only caller-owned state, satisfying permanent idempotence without an existence oracle. |
| RR-003 | Critical | Webhook backpressure | Trusted-IP and post-signature global ingress excess return generic `429` plus bounded `Retry-After` with no partial persistence; per-IGSID invalid excess is terminally deduplicated/cleared and acknowledged `200` without lookup; worker pressure defers durable work with `200`. |
| RR-004 | Important | Verified-link import lifetime | Import creation requires an active verified link; matched and unmatched following handles remain without renewal until per-import deletion or unlink; unlink deletes owner imports and unfinished dependent state. Retention/direction/follower fields are no longer accepted or stored. |

## Traceability And Coverage

- Planning to requirements: Approved direction and privacy/product boundaries are preserved; no lexicon change is introduced.
- Requirements to criteria: 50 Must rows each link to at least one criterion.
- Criteria to tests: All 48 criterion IDs appear in the acceptance test specification.
- Cross-language contract: `IT-021` and `TD-011` define shared synthetic Go/Flutter request, response, error, state, DELETE, pagination, and notification fixtures.
- Regression posture: Existing social notifications, follows, membership/moderation boundaries, auth/device/body/error policies, observability, cancellation, and multi-account isolation are covered.

## Risk And Approval Review

- Risk remains high because the change links identities across networks and stores private social-graph data.
- The user explicitly approved formalization and feasible implementation of the design.
- Approval does not authorize commit, push, production enablement, Meta dashboard mutation, or real/user-derived fixture use.
- Production/release gates remain: live unrelated-sender DM capability, Meta access/token/reply behavior, trusted edge/replica validation, approved current export-shape observation, device push lifecycle, and final accessibility/platform inspection.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Blocking document issues: None.
- Required implementation posture: strict TDD from `UT-001`, fail-closed Meta configuration, shared current-member and eligibility policies, private pgx persistence without profile cascades, stable deterministic PDS follow writes, fixed-account Flutter operations, and no production enablement before manual gates.

## Notes For Next Stage

- Keep `01-requirements.md` and `02-acceptance-tests.md` authoritative when a code-level choice differs from the older design-plan sketch.
- The coding plan may choose concrete storage types and package/file grouping but must preserve the exact wire, state, lifecycle, privacy, limit, and retention contracts.
- Current repository practice uses direct pgx stores and has no active sqlc configuration; following that local pattern is acceptable for this feature and should be recorded rather than bootstrapping unrelated repository-wide tooling.
