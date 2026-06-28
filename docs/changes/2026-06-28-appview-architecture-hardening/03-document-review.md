# Document Review: AppView Architecture Hardening

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document-reviewer
Date: 2026-06-28
Risk level: High

## Summary
The requirements and acceptance-test specification are consistent and ready for coding planning. The selected direction, Option B launch-ready route-class hardening with bare v1 success bodies, is carried through requirements, acceptance criteria, tests, risks, and handoff guidance. Must requirements have acceptance criteria and test coverage. The remaining issues are non-blocking planning notes around auth-route identity wording, CORS credential terminology, and deployment/config documentation precision.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests / Risk | Auth/login rate limiting is intentionally per-device only in the current decisions, but `AT-005` phrases the auth example under a scenario that says requests include a token. This is not blocking because the example row uses `token = none` and requirements explicitly cover pre-auth device limiting. | `01-requirements.md` Q5, Q16, EC-005, Open Questions; `02-acceptance-tests.md` AT-005 | During coding planning, ensure auth-route limiter tests model unauthenticated requests explicitly and do not require a Craftsky token for the auth class. |
| DR-002 | Suggestion | Requirements / Tests | CORS guidance correctly says `Access-Control-Allow-Credentials` should not be enabled, but some wording refers to “credentialed APIs” and “credentialed cookie CORS.” This could be misread as disallowing `Authorization` headers, even though the requirements explicitly require `Authorization` support. | `01-requirements.md` Q9, FR-010, RULE-003; `02-acceptance-tests.md` AT-006, IT-007 | In coding planning and implementation notes, distinguish cookie credentials from bearer-token `Authorization` headers. |
| DR-003 | Suggestion | Risk / Readiness | `RULE-008` and `AC-021` require process-local limiter deployment guidance, and tests mention inspecting “docs generated for dev/prod.” The exact artifact for this guidance is not defined. This is acceptable for planning, but implementation should decide where the warning lives. | `01-requirements.md` RULE-008, AC-021, RISK-008; `02-acceptance-tests.md` IT-009, MAN-002 | Coding plan should identify the concrete config/deployment documentation or startup log/config validation surface that satisfies AC-021. |

## Traceability Review
- Planning to requirements: Confirmed decisions Q1-Q17 are reflected in goals, non-goals, requirements, acceptance criteria, risks, and assumptions. Option B is consistently selected and Option C is explicitly rejected by NG-003, BR-001, and RULE-001.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` has linked acceptance criteria. Should requirements `FR-011`, `NFR-003`, and `NFR-004` also have acceptance criteria and tests.
- Acceptance criteria to tests: Every acceptance criterion AC-001 through AC-021 is represented in acceptance, unit, integration, regression, or manual tests. Manual coverage is limited to review-style properties that are difficult to prove fully through automation.

## Coverage Review
- Must requirements covered: Yes. Must requirements across success shape, error envelopes, device ID, body limits, rate limits, CORS, no-body routes, preflight ordering, no-IP keys, upload attempts, and single-instance constraints all map to tests.
- Missing or weak coverage: No blocking gaps. Non-blocking weak spots are the auth-class no-token phrasing in AT-005, real-world suitability of numeric rate defaults, production logger sink coverage, and multi-replica behavior intentionally left out of scope.
- Manual-only coverage: No Must requirement is manual-only. `MAN-001` and `MAN-002` supplement automated tests for middleware ordering/log redaction review and deployment/no-IP-key review.

## Risk And Approval Review
- Risk level: High, appropriately called out in both documents because the change affects launch API contracts, CORS/security posture, abuse controls, middleware ordering, logging safety, and deployment constraints.
- Review requirement: Explicit review before coding planning is satisfied by this document.
- Approval notes: Proceed with coding planning, but preserve the documented guardrails: bare success bodies, enveloped errors, exact production CORS, no cookie credentials, no AppView IP limiter keys, process-local only under single-instance deployment, and explicit route policy classification.

## Coding Plan Readiness
- Ready for coding planning: Yes
- Recommended first step: Start with `UT-006` / `IT-004`, the route policy classification contract requiring every `/v1/*` route to declare rate-limit class and body policy. This creates the guardrail for subsequent body-limit, rate-limit, and CORS middleware work.
- Blocking issues: None.

## Notes For Next Stage
- Treat `01-requirements.md` and `02-acceptance-tests.md` as the source of truth.
- Plan implementation around explicit route policy metadata before implementing individual middleware behaviors.
- Keep auth/pre-auth route-class tests clear: auth/login limits are per-device without requiring a session token.
- Specify where AC-021 deployment guidance will live before implementation sign-off.
- Preserve existing v1 bare success response shapes and standard error envelope behavior while adding hardening.
