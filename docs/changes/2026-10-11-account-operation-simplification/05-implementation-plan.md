# Implementation: Account Operation Simplification

## Inputs And Authorization

The user approved [04-coding-plan.md](04-coding-plan.md) and requested implementation on 2026-10-11. Its INV-01–06 and V-01–07 are the implementation contract, alongside the linked multi-account and PDS mutation requirements/tests. This approval includes the client account-authority work described in the plan. Source changes remain local and uncommitted.

## Progress

| Step | Contract | Status | Work |
|---|---|---|---|
| Inventory | INV-01–06 | completed | All 80 baseline files classified in [06-check-inventory.md](06-check-inventory.md), with supplementary direct-check review. |
| Foundation | V-01 / INV-01–03 | completed | Captured ActiveAccountOperation; guarded acquisition/steps and synchronous publication. Existing fixed-session Dio and disposal behavior reused. |
| Reference mutation | V-02–05 / INV-01–04 | completed | Like/unlike retains initiating repository across retries; shared runner and optimistic behavior retained. |
| Ordinary mutations | V-02–06 / INV-01–05 | completed | Posts, relationships, reports, profile/business records use shared authority/retry mechanics and explicit outcome adapters. |
| Multi-step workflows | V-02–06 / INV-01–05 | completed | Standard/project composers, uploader/coordinator assertion, drafts, scheduling and Instagram migrated. |
| Read/state review | V-03, V-05–06 / INV-02–05 | completed | Feed/search/business/relationship-list guards consolidated; existing private families, epochs, revisions and cleanup retained. |
| Specialized review | V-07 / INV-06 | completed | Onboarding compatible mechanics migrated; auth, notification recipient, billing and deletion fences reviewed and retained. |
| Automated verification | V-01–07 | completed | Final complete Flutter suite: 2,850 passed, 38 skipped. Analysis and diff checks clean. |
| Manual device verification | V-02, V-06 | not performed | Slow upload/account-switch and retry interaction on a signed-in device remain a manual validation gap. Automated completer/widget coverage ran instead. |

## Implementation Decisions

- The context stores an existing ActiveAccountLease and validity predicate, with no credentials, feature repository registry, client cache or keep-alive registration. Ref/WidgetRef lifetime and null-ownership test behavior remain supported.
- Feature dependencies are resolved once and captured for retries. Synchronous mutation repositories use lazy final capture at the original first-read location inside the error boundary, so a dependency failure still follows existing loading/error behavior.
- Async account-keyed acquisition uses before/after authority checks. Session replacement changes the captured generation and cannot pass acquisition. Existing fixed-token clients, client disposal and lease-scoped 401 invalidation remain unchanged; no new session-keyed client adapter was necessary.
- PdsMutationRetry shares ambiguous marking, elapsed budget, injected clock/jitter/delay and token validation with runPdsMutation and existing feature adapters. Features retain their original null/false/discard outcomes, raw ambiguous exhaustion versus unresolved errors, independent profile-record outcomes and diagnostics.
- The existing adapters differ in whether a failed markAmbiguous immediately makes a retry obsolete. That distinction is explicit rather than silently normalized.
- Awaiting a shared helper introduces a continuation boundary. Callers retain authority checks before later dispatch, state/cache changes and navigation; publish guards synchronous completion side effects. Guarded steps preserve original failed causes/stacks.
- Provider families were not broadly converted: existing session families and central invalidation/boundary epochs already establish the required isolation. Saved-post revisions, private cleanup, request sequences and mutation-token fences are independent and remain in place.
- Widget and composer adapters retain originating-flow identity, mounted checks, submission order, media reuse, cancellation and cleanup. The shared composer assertion preserves its existing DiagnosticStateError text and issue owner.
- No backend, wire API, lexicon, dependencies, storage schema or generated-provider declaration changed. No Riverpod regeneration was required.

## Test Execution And Evidence

The planned first slice was exact-session acquisition, followed by the like/unlike reference mutation and shared retry behavior. Existing preservation suites were reused for the broader migrations.

| Contract | Evidence |
|---|---|
| V-01 / INV-01–02 | New account-operation suite pauses the real accountDio acquisition on device identity, replaces Alice's session, resumes and observes AccountOperationDiscarded. Existing API provider tests verify fixed-token headers and 401/session binding. The initial new context API was absent; the foundation implementation made this test pass. |
| V-02 / INV-02–03 | New switch-away/back test rejects dispatch and publication from the original activation. Profile-save regression cases pause both record retries, then switch or replace the same DID's session; neither repository dispatches again and no overlay is accepted. |
| V-03 / INV-02–03 | New tests cover old Ref disposal/rebuild and a completed step followed by an ownership change before publication. Existing business/composer/Instagram widget tests cover late uploads/results and disposal. Riverpod invalidation alone does not replace an unobserved Ref immediately; the disposal fixture explicitly rebuilds the provider. |
| V-04 / INV-01, INV-04 | New like test replaces the repository provider during an ambiguous retry and verifies both requests use the original repository. It failed with the original read-on-every-retry behavior; capture fixed it. Shared retry test pauses the injected delay and rejects obsolete authority on resume. Existing controller tests cover immutable input/key, budget, revision, exhaustion and reset. |
| V-05 / INV-03–04 | Existing feed/business/search/relationship and saved-post suites exercise overlays, revisions, pagination/restarts and account boundaries. Those fences were reviewed independently of active authority. |
| V-06 / INV-05 | New guarded-step failure test preserves original cause and stack after disposal. New like dependency-failure test preserves the mutation error state and Riverpod's ProviderException containing the original cause. Existing tests cover partial profile saves, optimistic state, composer cleanup and diagnostic output. |
| V-07 / INV-06 | Existing auth/onboarding, notification, subscriptions and settings/account-deletion suites run in the complete Flutter suite. Domain guards retained as recorded in the inventory. |

Test setup corrections were separated from production behavior: the retry replacement fixture uses a mutable provider selection plus invalidate; disposal requires rebuilding the Ref; Riverpod exposes a dependency failure as ProviderException; the continuation test explicitly changes authority in a completion continuation. Missing imports and lint failures found during migration were corrected before final verification.

Validation runs:

- Foundation plus API binding tests: 5 passed.
- Shared retries/reference mutation slice: 39 passed.
- Posts/profile/business/onboarding slice: 82 passed.
- Broad affected-feature regression run: 534 passed, 9 skipped.
- Final retry/boundary adapter run: 174 passed.
- Final context/reference-mutation regression run: 26 passed.
- Complete Flutter suite before the final boundary refinements: 2,848 passed, 38 skipped.
- Final complete Flutter suite after all source/test refinements: 2,850 passed, 38 skipped (exit 0).
- Final Flutter analysis: no issues found.
- git diff --check: passed.

Commands run directly from app/ are equivalent to the checked-in just recipes, with dependency resolution skipped because dependencies were already installed:

```sh
flutter test --no-pub
flutter analyze --no-pub
```

The focused runs covered test/auth/providers/account_operation_guard_test.dart, test/shared/api/providers, test/shared/mutations, affected feed/profile/business/search/project providers, drafts, scheduled posts, Instagram and onboarding. The complete suite additionally covers the retained specialized areas and observability behavior.

## Limitations And Handoff

The planned manual slow-upload and retry-switch checks were not run on a signed-in device. Deterministic upload/retry/account-boundary tests cover these races, but they do not constitute physical-device verification. The suite's skips are reported, not treated as passes. Already-dispatched server work may still complete; local discarding does not undo remote effects.

Implementation review is a separate next stage and has not been invoked. Implementation and automated verification are complete; the work is ready for implementation review. No commit, push or deployment was performed.

## Completion Checklist

- [x] Every inventoried area migrated or reviewed with a retained-guard rationale.
- [x] Shared context and retry behavior verified across session/activation changes.
- [x] Focused regression suites pass.
- [x] Final complete Flutter tests pass (2,850 passed, 38 skipped).
- [x] Final static analysis and diff checks pass.
- [x] Implementation evidence and manual validation gap recorded.
- [x] Independent implementation-review stage left for user selection.
