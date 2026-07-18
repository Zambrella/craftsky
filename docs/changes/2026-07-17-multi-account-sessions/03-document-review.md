# Document Review: Multi-Account Sessions And Notification Routing

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-18
Risk level: High

## Summary

The corrected requirements and acceptance tests are consistent, traceable, and ready for coding-plan work. They select a device-local, DID-keyed session registry with one active account, preserve the project's AppView/PDS and credential boundaries, and specify the account isolation and recovery behavior needed for High-risk implementation planning.

The prior blocking findings are resolved. `NFR-002` and `AC-019` now distinguish the opaque `accountSubscriptionId` carried by the authenticated registration response and provider payload from the secure local DID-to-ID map. `UT-022` covers canceled and failed Add account flows without partial registry mutation. `IT-013` directly protects AppView shared-installation logout isolation and is part of normal verification even when production server code is unchanged.

All 35 Must requirements link to acceptance criteria and tests, all 30 defined acceptance criteria appear in the test specification, and no unresolved product question or document contradiction remains. The remaining notes concern coding-plan precision and physical-platform verification, not product intent.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Requirements / Security / Traceability | Resolved: the security contract now separates the opaque, non-authorizing provider routing ID from the secure DID-to-ID binding map and credentials. The ID is permitted only in the authenticated registration response and provider payload and remains redacted from diagnostic and user-visible output. Cached handles/avatars are permitted only on the designated switcher, Profile, and inactive-recipient UI and remain absent from diagnostics. | `01-requirements.md` `FR-005`, `FR-012`, `FR-021`, `FR-022`, `NFR-002`, `AC-019`; `02-acceptance-tests.md` `UT-014`, `REG-010`, `MAN-003` | No further document change. The coding plan must preserve the transport, redaction, and intentional identity-UI boundaries exactly and must not treat the routing ID as authorization. |
| DR-002 | Suggestion | Tests / Risk | Resolved: `IT-013` now requires an AppView test with A and B on one installation, proving logout deactivates only A and fails closed before session revocation if notification cleanup fails. AppView verification is mandatory for the feature. | `01-requirements.md` `FR-016`, `FR-026`, `AC-015`, `AC-029`, `ASM-004`; `02-acceptance-tests.md` `IT-013`, `GAP-003`, Commands discovered | Implement `IT-013` and include the focused/full AppView contract tests in the coding plan even if no production Go file changes. |
| DR-003 | Suggestion | Tests / Coverage | Resolved: `UT-022` explicitly covers OAuth cancellation, browser-launch failure, handoff failure, rejected completion, and partial-result cleanup while preserving A's registry, active pointer, MRU order, route, and account-scoped state. | `01-requirements.md` `FR-003`, `EC-004`; `02-acceptance-tests.md` `UT-022` | Implement `UT-022` alongside additive OAuth completion tests without renumbering existing IDs. |

## Traceability Review

- Planning to requirements: The embedded discovery, eight confirmed decisions, candidate approaches, and recommended direction consistently lead to Option A: a secure device-local session registry with one active DID. Goals and non-goals preserve independent AppView sessions, no server-visible account group, no PDS-token storage, no combined account surfaces, no bulk logout, and a maximum of five accounts. No `00-initial-prompt.md` is present, so this review uses the initial request and discovery context embedded in `01-requirements.md`.
- Requirements to acceptance criteria: All 35 Must requirements link to at least one acceptance criterion. The 30 defined criteria cover storage/restoration, additive OAuth, responsive switching, account-bound requests/state, multi-account push, routing, sign-out/invalidation, MRU fallback, confidentiality, limits, offline switching, counts, banners, unsaved work, validation, removed-account opens, offline recovery, and switcher action constraints. The corrected `AC-019` is externally testable and no longer conflicts with notification routing.
- Acceptance criteria to tests: Every defined acceptance criterion appears in the coverage matrix and at least one test case. Every acceptance, unit, integration, regression, and manual test row carries requirement and acceptance-criterion IDs. `UT-022` covers `EC-004`, and `IT-013` adds direct AppView depth for `AC-015`/`AC-029`.

## Coverage Review

- Must requirements covered: 35 of 35 have linked tests or explicit partial/manual gaps. The two Should requirements, `NFR-003` and `NFR-004`, are also covered.
- Missing or weak coverage: No blocking coverage gap remains. `GAP-005` appropriately defers the exact partial-write protocol to coding planning, where every recoverable failure point must be enumerated against `TD-002`. `GAP-003` is addressed by `IT-013` and mandatory AppView verification.
- Manual-only coverage: No Must requirement is manual-only. `MAN-001` supplements automated widget/semantics tests; `MAN-002` supplements runtime simulations with physical-device provider behavior; `MAN-003` supplements mock storage and secret scanning with platform inspection. These manual checks are justified.

## Risk And Approval Review

- Risk level: High. Authentication credentials, account-scoped authorization, concurrent provider state, notification routing, and shared-token recovery can cause cross-account disclosure or actions under the wrong identity.
- Review requirement: Satisfied for coding planning. The corrected documents are coherent and have concrete verification paths for every high-risk behavior.
- Approval notes: Coding planning may proceed. Because the feature remains High risk, explicit user approval is still required before implementation begins; selecting coding planning does not itself authorize implementation.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start with `UT-001` in proposed `app/test/auth/models/session_registry_test.dart`, specifying the versioned registry representation and recovery behavior before providers or UI depend on it.
- Blocking issues: None.

## Notes For Next Stage

- Preserve the test order in `02-acceptance-tests.md`: registry/MRU, storage restoration, request-token and `401` scoping, activation boundary, additive OAuth plus failure preservation, startup validation, confirmed sign-out plus `IT-013`, notification routing, registration/counts, unsaved-work guard, offline recovery, and switcher UI.
- Treat account identity as DID and asynchronous ownership as DID plus session/account generation. Enumerate which providers are DID-keyed, which are disposed or invalidated at activation, and how late completions are rejected.
- Specify the registry write/recovery protocol and every interruption point covered by `TD-002`; do not leave “atomic enough” as an implementation judgment.
- Preserve the `NFR-002` boundary: tokens, cleanup credentials, and the DID-to-routing-ID map remain in secure storage; the opaque ID is allowed only in its authenticated registration response and provider payload; routing selects context but never authorizes destination content; credentials and routing values remain absent from diagnostic and user-visible output; identity values remain absent from diagnostics and appear only on the designated switcher, Profile, and inactive-recipient UI.
- The current AppView API surface is sufficient: registration returns an opaque routing ID, logout deactivates subscriptions for the authenticated DID and installation, and notification-device deletion is account/device scoped. No new route or migration is indicated, but `IT-013` and normal server verification remain required.
- Preserve the exact offline recovery ordering: removed account becomes non-activatable locally; AppView logout/deactivation completes or returns authoritative unauthorized; cleanup credential is deleted; only then are remaining eligible accounts registered to the replacement provider token.
- Do not add recipient identity to OS-visible notification copy, a bulk logout action, direct inactive-account removal, or per-account saved navigation locations.
