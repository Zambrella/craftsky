# Document Review: Provider-First AT Protocol Registration

## Verdict
Status: Approved
Reviewer: OpenCode document review
Date: 2026-08-29
Risk level: High

## Summary
The revised requirements and acceptance-test specification are consistent, testable, and ready for coding-plan work. They preserve the approved provider-first direction while resolving the unsafe create-prompt fallback, pre-owner lifecycle boundaries, public error taxonomy, account-limit race, abuse-control decision, evidence requirements, and reverse traceability gaps found in the initial review.

The implementation remains high risk because it changes unauthenticated OAuth admission, durable callback state, identity authority verification, owner lifecycle binding, credential cleanup, and trusted error handoff. The document set now defines those behaviors without requiring the coding planner to invent product or security policy.

## Findings
None identified.

## Traceability Review
- Planning to requirements: The Bluesky-only provider-first flow, create-or-continue semantics, server-owned provider configuration, credential boundary, immediate browser launch, welcome/Add Account entry points, trusted callback failures, browser-abandonment behavior, manual release smoke check, and future-provider seam are preserved.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` links to at least one objective acceptance criterion. Stable IDs are retained; `AC-029` adds explicit auth-route limiting and shared-capacity evidence.
- Acceptance criteria to tests: Every acceptance criterion `AC-001` through `AC-029` has a verification path. Every declared `AT`, `UT`, `IT`, `REG`, and `MAN` case references valid requirement and acceptance-criteria IDs and appears in the applicable coverage-matrix row.

## Coverage Review
- Must requirements covered: All Must requirements have automated coverage except the intentionally manual live-provider portion of `FR-016`. Database, AppView, federated-network, Flutter, lifecycle, handoff, security, and regression levels are represented.
- Missing or weak coverage: None blocking. `GAP-003` is a known coding-design constraint around the pinned Indigo API, not an unresolved behavior. `GAP-001`, `GAP-002`, and `GAP-004` correctly retain live OS/provider/propagation limitations as explicit manual or future concerns.
- Manual-only coverage: `MAN-001` is justified because CI must not create live Bluesky accounts and the repository has no real-OS browser-to-app integration harness. Its evidence fields and destination are now fixed by `AC-028`.

## Risk And Approval Review
- Risk level: High.
- Review requirement: Satisfied for coding-plan work. Implementation still requires the planned TDD and implementation-review gates, and release requires `MAN-001`.
- Approval notes: The original product direction was approved on 2026-08-29. Review-driven revisions remove automatic prompt downgrade; distinguish newly created owners from pre-existing owner lifecycle/session state; reuse and explicitly bound current auth admission controls; define account-limit race behavior; map trusted failures deterministically; and lock user-facing registration copy and smoke-test evidence.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Design the purpose-aware `oauth_auth_requests` migration and store transitions around `IT-005`, then select the narrowest safe create-prompt integration strategy compatible with the pinned Indigo behavior before planning OAuth start code.
- Blocking issues: None.

## Notes For Next Stage
- Do not plan against Indigo `WithPrompt` or `AuthServerMetadata.PromptValuesSupported`; neither exists in pinned `v0.0.0-20260417172304-7da09df6081d`, and upstream PR #1411 is unmerged as of 2026-08-29.
- Choose either a stable Indigo release that includes the required metadata/request support or a narrow project-local adapter that preserves Indigo's OAuth, PAR, PKCE, and DPoP protections. Do not use an unmerged fork.
- Never parse provider error descriptions or generic `invalid_request` to authorize a prompt-free downgrade. An advertised prompted PAR gets one attempt; an unadvertised prompt is omitted.
- Preserve one shared auth-request state machine with purpose-aware database constraints. Login and deletion remain strictly owner-bound; initial registration is ownerless; verified binding is atomic.
- Preserve the returned-DID -> authoritative PDS -> discovered authorization-server proof before owner/session activation.
- A failed attempt may leave a newly verified owner `departed`; it must not alter an existing owner's prior lifecycle or sessions, and no session created by the failed attempt may become active.
- Apply the existing per-device `RateClassAuth` middleware and shared durable pending-auth capacity. Device-ID rotation remains an explicitly accepted residual risk for this pass.
- Preserve existing exact-receipt handoff recovery independently from fresh-registration retry behavior.
- Keep live Bluesky account creation out of CI; `MAN-001` remains a release blocker.
- Preserve existing onboarding effect severity: profile read/validation/write failures are fatal, while identity-cache and repository-tracking failures remain warning-only for login and registration.
