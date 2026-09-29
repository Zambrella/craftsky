# Implementation Review: Pre-Launch Online Safety Readiness

## Verdict

Status: Changes required
Reviewer: OpenCode
Date: 22 September 2026
Risk level: High

## Summary

The bounded repository implementation is complete and its automated verification is
green. The implementation covers the planned image-safety, restricted incident,
evidence, hold, retention, reporting, age-eligibility, video shutdown, policy, and
release-gate behavior without adding the prohibited production IWF adapter or Ozone
dependency.

This is not approval to merge into an auto-deploying production branch or to launch.
The policy artifacts are intentionally blocked, the production release command exits
non-zero, and the external/manual gates remain open. Cloudflare Pages must be verified
to execute `npm run build` before a merge can be considered safe because the previous
empty build command would publish the staged draft files without running the new
repository gate.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | Risk | The repository now supplies a fail-closed Cloudflare build command, but the external Cloudflare Pages project configuration has not been verified. If it still uses an empty build command, merging to `main` can publish blocked draft policies without evaluating readiness. | BR-001, BR-006, AT-001, AT-012, MAN-014; `web/package.json`; `scripts/cloudflare_pages_build.py`; `web/README.md` | Before merge, configure and verify the production Pages build command as `npm run build`, preserve `web` as the root, and capture evidence that a blocked manifest prevents deployment. Alternatively, keep the draft artifacts outside the deployable tree until approval. |
| IR-002 | Important | Risk | Public launch remains correctly blocked by unresolved approvals, production scanner evidence, provider and operational exercises, accessibility evidence, and every open P0 item. Automated success does not close these gates. | Step 26; AT-018, AT-019; MAN-001 through MAN-014; DR-001 through DR-008; GAP-001 through GAP-012; `web/policy-publication-manifest.json`; `online-safety/registers/compliance-gap-register.md` | Keep readiness false and complete the named external/manual evidence before approving policy publication or public launch. |
| IR-003 | Suggestion | Traceability | The execution summary in `05-implementation-plan.md` predates the final remediation: it records 2,379 Flutter tests and six readiness tests rather than the final 2,383 and nine, and its implementation-review checkbox remains open. | `05-implementation-plan.md:465-485` | After accepting this review, update the evidence counts and mark the review checklist item complete without changing the blocked launch status. |

## Requirement And Test Traceability

- Requirements implemented: Repository-owned behavior for BR-001 through BR-006,
  FR-001 through FR-038, applicable NFRs, and RULE-001 through RULE-010 is represented
  by implementation and automated tests or is explicitly bounded by the approved
  external/manual gaps.
- Tests implemented: Planned unit, integration, acceptance, and regression coverage
  through Step 25, including the final moderator age-eligibility command, retained
  restricted-account controls, hold-aware evidence deletion, durable retention,
  production video route shutdown, and Cloudflare publication gate.
- Unplanned behavior: None identified. The private S3 evidence store, database-backed
  operator authentication, retention lifecycle, and command-route composition are
  called for by `04-coding-plan.md` sections 3 through 5.
- Remaining gaps: Step 26 and Finding IR-001. Manual/external evidence is not replaced
  by repository tests.

## Test Evidence

- Commands reviewed: `CGO_ENABLED=0 go test -p 1 ./...`, `CGO_ENABLED=0 go vet ./...`,
  `flutter test`, `flutter analyze`, Dart MCP analysis, `npm test` in `web`,
  `python3 -m unittest discover -s scripts -p 'test_*.py'`, preview-branch
  `npm run build`, production `scripts/cloudflare_pages_build.py`, and
  `git diff --check`.
- Passing evidence: Full Go suite; Go vet; 2,383 Flutter tests; Flutter analyzer and
  Dart MCP with no issues; 10 Playwright tests; nine Python release/build-gate tests;
  preview build; whitespace validation.
- Failing or skipped tests: The production Cloudflare build command intentionally
  exits non-zero and lists blocked policy metadata, missing control evidence,
  unresolved scanner/video evidence, and open P0 register entries. AT-018, AT-019,
  and MAN-001 through MAN-014 remain pending external/manual evidence by design.

## Risk Review

- Risk level: High
- Risk notes: This change governs illegal-content detection, restricted evidence,
  legal holds, child eligibility, privacy, and public legal-policy deployment. A
  configuration bypass or unsupported launch claim has material safety and legal
  consequences. Production scanner integration is deliberately absent and video is
  deliberately disabled.
- Approval notes: Repository code is suitable for continued review and controlled
  non-production testing. It is not approved for auto-deploying merge or launch until
  IR-001 is verified and the Step 26 gates are closed.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: Restricted users now have functional access to standing/appeal, muted and
  blocked accounts, privacy, and account deletion. The controls use existing design
  components and localization, and automated layout/regression tests pass.
- Suggested polish notes: Conduct the planned supported-device accessibility pass and
  inspect the restricted-account action hierarchy at compact sizes; do not change
  behavior during polish.

## Handoff Back To TDD Builder

- Required fixes: Verify the Cloudflare production build configuration and preserve
  the fail-closed command; complete the external/manual launch evidence without
  marking unresolved controls approved.
- Suggested next failing test: Deployment-level evidence that a production-branch
  build with the current blocked manifest cannot publish, followed by MAN-014 atomic
  publication and rollback evidence when approvals are complete.
- Verification to rerun: `just public-release-check`, the production Cloudflare build,
  full Go/Flutter/web/Python suites, and `git diff --check` after any remediation.
