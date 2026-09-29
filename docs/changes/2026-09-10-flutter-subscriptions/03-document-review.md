# Document Review: Flutter Account Subscriptions

## Verdict

Status: Approved

Reviewer: OpenCode

Date: 2026-09-10

Risk level: High

## Summary

The requirements and acceptance tests consistently define an
AppView-authoritative native subscription flow with one permanently reserved
billing-owner DID, crash-safe UUID completion, explicit payer/beneficiary
separation, safe lapsed repurchase, race-safe reconciliation, and explicit
assignment. The narrow
`BillingLicense.subscriptionId` response enhancement supplies the relationship
Flutter needs without changing billing behavior.

All findings from the prior reviews are resolved. Every Must requirement and
acceptance criterion has appropriate automated or justified manual coverage, and
the documents are ready for coding planning. Production activation dependencies
remain explicit and do not block deterministic implementation.

## Findings

None identified.

### Resolution Verification

| Finding | Resolution |
|---|---|
| DR-001 | Lapsed repurchase eligibility uses access, payment, exact renewal, anomaly, and reconciliation state rather than retained-row presence. |
| DR-002 | Owner DID and optional UUID remain reserved after setup failure or session loss; another DID cannot replace them and RevenueCat is not logged out. |
| DR-003 | FR-027 adds internal `BillingLicense.subscriptionId` and prohibits array-order or provider-ID correlation. |
| DR-004 | FR-017 defines baseline-generation reconciliation with separate expected-state verification. |
| DR-005 | Same-subscription recovery preserves assignment while a new subscription creates an unassigned license. |
| DR-006 | FR-026 and AT-017 define the complete active-owner call boundary and stale-lease fencing. |
| DR-007 | Target races map to `assignment_target_ineligible`; owner 401 uses normal invalidation. |
| DR-008 | Tier parsing remains closed while provider-owned strings use safe fallback presentation. |
| DR-009 | Wrong-owner MAN-003 evidence is explicitly blocked until its provider/recovery prerequisites exist. |
| DR-010 | `PUT /v1/billing/account` requires a durable same-DID reservation with no UUID; after UUID persistence every recovery is GET-only and remains locked on 404/mismatch. |
| DR-011 | Direct paywall results exclude `notPresented`; missing named offerings/packages fail before RevenueCatUI, while attached-paywall failures are mapped from presentation. |
| DR-012 | Owner guards explicitly cover reservation writes, owner GET/PUT, offerings, identity, paywall, restore, Customer Center, reconciliation request/polls, assignment, and unassignment. |
| DR-013 | Exact assignment errors are tested on applicable routes; unassignment does not expect cooldown. |
| DR-014 | Nullable, empty, known non-terminal, and unknown renewal/anomaly values all have fail-closed purchase tests. |
| DR-015 | Coverage-matrix associations match detailed acceptance, unit, integration, and regression declarations. |
| DR-016 | First setup durably reserves the confirming DID before PUT and persists the returned UUID before RevenueCat identity; interruption tests cover both barriers. |
| DR-017 | Release-safe RevenueCat logging and provider exception sanitization are explicit and covered by canary tests. |
| DR-018 | Every configured paywall version requires manual proof of an operable close affordance because `displayCloseButton` is not sufficient for all templates. |

## Traceability Review

- Planning to requirements: The accepted payer/beneficiary split,
  AppView-authoritative access model, native-only purchase scope, no-gating
  boundary, DID-first reserved-owner lifecycle, narrow response enhancement, and
  client reconciliation algorithm are represented without reopening decided
  scope.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE`
  links to at least one externally verifiable acceptance criterion, including
  FR-027 and AC-029.
- Acceptance criteria to tests: AC-001 through AC-029 are represented in the
  coverage matrix and detailed tests. Test declarations and matrix associations
  are synchronized.

## Coverage Review

- Must requirements covered: All Must requirements have automated coverage or a
  justified native/provider manual check.
- Missing or weak coverage: None that blocks coding planning. Real native store,
  RevenueCatUI, actual paywall close affordances, restore ownership, Customer
  Center, and lifecycle behavior remain appropriately manual.
- Manual-only coverage: MAN-001 through MAN-005 cover iOS/Android sandbox
  transactions, provider restore/management, native lifecycle, and accessibility.
  The wrong-owner MAN-003 branch remains explicitly blocked until RevenueCat
  restore ownership and recovery/support procedures are configured.

## Risk And Approval Review

- Risk level: High.
- Review requirement: Satisfied for coding planning. Implementation review and
  native sandbox evidence remain required before production activation.
- Approval notes: The product owner approved the initial requirements and chose
  the pinned original owner, explicit subscription/license relationship, and
  client baseline-generation reconciliation decisions. The product owner then
  selected DID-first durable reservation for crash-safe setup. Subsequent
  revisions preserved those decisions while closing all review findings.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Begin with `UT-001`, `UT-012`, `IT-001`, and `IT-013`
  to establish AppView models, wire contracts, and the explicit internal
  subscription/license relationship before owner identity or RevenueCat flows.
- Blocking issues: None for coding planning or deterministic implementation.
  Production activation dependencies remain documented in the requirements and
  acceptance-test gaps.

## Notes For Next Stage

- Implement the narrow AppView `BillingLicense.subscriptionId` response change
  and its contract test before Flutter correlation logic.
- Persist the confirming DID before first ensure and the returned UUID before
  RevenueCat identity; permit PUT retry only for the same DID while UUID is absent,
  then enforce GET-only recovery.
- Keep AppView self-access, not RevenueCat `CustomerInfo`, authoritative.
- Guard the complete owner call boundary and fence every asynchronous completion
  by the captured account lease.
- Model direct RevenueCat paywall results separately from observable
  pre-presentation offering/package unavailability and presentation-time
  attached-paywall failure.
- Configure release-safe RevenueCat logging, never forward raw SDK/platform
  payloads, and manually verify every paywall version's close affordance.
- Preserve exact assignment error mapping and fail-closed purchase eligibility
  for unknown provider state.
- Do not activate production until catalog, credentials, paywall close
  affordances, Customer Center, sandbox evidence, and recovery prerequisites are
  complete.
