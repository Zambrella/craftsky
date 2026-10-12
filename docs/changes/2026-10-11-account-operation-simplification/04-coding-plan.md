# Plan Outline: Account Operation Simplification

## 1. Purpose And Inputs

Reduce the number of places responsible for account ownership during asynchronous work, while preserving existing behavior. Capture account authority and dependencies once, reuse shared operation handling, and isolate results by session where that simplifies the code.

This is a planning outline requested on 2026-10-11. It does not authorize further implementation or claim that a new requirements, acceptance-test, or document-review stage has been completed. The stages below include design and validation decisions that must be resolved before each migration.

Existing contracts:

- [Multi-account requirements](../2026-07-17-multi-account-sessions/01-requirements.md), especially FR-007, FR-008, FR-009 and NFR-001: account-bound requests, isolated state, and rejection of stale completions.
- [Multi-account acceptance tests](../2026-07-17-multi-account-sessions/02-acceptance-tests.md) and [approved document review](../2026-07-17-multi-account-sessions/03-document-review.md).
- [PDS mutation requirements](../2026-09-17-unified-pds-record-mutations/01-requirements.md), [acceptance tests](../2026-09-17-unified-pds-record-mutations/02-acceptance-tests.md), and [approved document review](../2026-09-17-unified-pds-record-mutations/03-document-review.md): immutable keyed retries, in-memory operation state, and reconciliation through the shared controller.
- [Architecture reference](../../../atproto-craft-social-app-reference.md) and [logging/error-reporting guide](../../development/logging-error-reporting.md).

The current worktree already contains an uncommitted profile-save retry consolidation and its regression tests from this conversation. Treat those as a starting point to review, rather than repeating the change or assuming it completes this plan.

## 2. Implementation Strategy

Build on the existing `account_operation_guard.dart`, fixed-token Dio clients, `runPdsMutation`, and `PdsRecordOperationController`. Avoid a second general retry framework or one context containing every feature repository.

The initial inventory found 91 calls to `isActiveAccountOperationCurrent` across 37 files, excluding the helper declaration. Other checks exist in widgets, composers, Instagram migration, account-scoped storage, onboarding, notifications, and billing. This is a search baseline, not a count of removable checks.

Classify each check by the authority it protects:

| Check | Meaning | Refactor rule |
|---|---|---|
| `AccountSessionLease` validity | The same stored session generation still exists for this account. | Keep for work that legitimately targets an inactive retained account. |
| `ActiveAccountLease` validity | The same session and activation generation still own the active flow. | Keep for active-account operations, including Alice → Bob → Alice. |
| Provider/widget lifetime | The original state owner is still usable. | Preserve `ref.mounted` and `context.mounted` protections where required. |
| Request or mutation revision | A newer action, refresh, or mutation has superseded this work. | Preserve independently of account checks. |
| Domain-specific authority | Billing ownership, deletion authorization, notification recipient, or another feature invariant. | Review separately; do not replace with a generic active-account check. |

Binding requests to an account and suppressing obsolete results solve different problems. Capturing a repository must not allow an operation to continue after an account switch where it currently stops. Likewise, making a provider account-scoped does not make every late completion safe.

## 3. Behavior To Preserve

Use these local planning IDs for traceability; they are not replacements for the existing requirement IDs.

| ID | Required invariant | Verification |
|---|---|---|
| INV-01 | Subsequent requests and retries use the initiating session; never silently acquire another account or a replacement session. | V-01, V-02 |
| INV-02 | Existing active-operation cancellation points remain effective after switching, switching back, sign-out, session replacement, or disposal. | V-02, V-03 |
| INV-03 | Obsolete successes and failures cannot update another account's state, navigation, dialogs, or caches. | V-03, V-05 |
| INV-04 | Retry budgets, keys, immutable inputs, optimistic behavior, overlays, and overlapping-operation ordering remain unchanged. | V-04, V-05 |
| INV-05 | Loading, partial-save results, error types, return values, and diagnostic ownership remain unchanged. | V-03, V-06 |
| INV-06 | Inactive-account work and specialized ownership rules retain their existing semantics. | V-07 |

Already-dispatched server work can still complete after local cancellation. This refactor does not roll it back or strengthen the guarantee into remote cancellation.

## 4. Affected Areas And Candidate Modules

Paths below are relative to `app/`. Proposed names are illustrative until the foundation design is verified.

| Area / Existing path | Planned work | Invariants / Checks |
|---|---|---|
| `lib/auth/providers/account_operation_guard.dart`; `lib/auth/models/account_session_lease.dart` | Introduce a small captured operation context, using existing lease types and validity rules. | INV-01–03; V-01–03 |
| `lib/shared/api/providers/dio_provider.dart`; feature API/repository providers | Resolve and retain a dependency belonging to the exact captured session. Reuse fixed-token clients and lease-scoped 401 invalidation. | INV-01–02; V-01–02 |
| `lib/shared/mutations/pds_record_operation_controller.dart` | Integrate captured ownership with the existing runner; retain token revision checks and reconciliation. | INV-02, INV-04–05; V-02, V-04–06 |
| `lib/feed/providers/`; `lib/profile/providers/`; `lib/business/providers/` | Migrate ordinary mutations in small feature groups. | INV-01–05; V-01–06 |
| `lib/feed/composer/`; standard/project composer widgets; drafts and scheduled-post providers | Replace repeated ownership plumbing across multi-step workflows without changing submission order or cleanup. | INV-01–05; V-02–06 |
| Feed/search/business read providers; account-scoped saved-post and Instagram providers | Selectively use session-scoped state and consolidate completion handling. Preserve request revisions. | INV-02–05; V-03, V-05–06 |
| Auth/onboarding, notifications, subscriptions, settings account deletion | Complete a separate review of session, activation, and domain authority. Adopt only compatible primitives. | INV-02–03, INV-05–06; V-02–03, V-06–07 |

New code should live near the current guards and shared mutation infrastructure. Add feature-specific account repository families only where needed; avoid copying every repository/provider solely to support this refactor.

## 5. Operation Context, Interfaces, And Data Flow

The context should carry the captured lease, expose named validity checks, and support resolving a feature dependency once. It should not contain raw credentials, own global navigation, or import every feature.

Illustrative responsibilities, not production signatures:

```text
capture active operation -> captured ActiveAccountLease
resolve feature repository -> exact captured AccountSessionLease
guarded step -> check required authority, run with captured dependency
publish completion -> check authority and original state-owner lifetime
```

Dependency resolution is itself asynchronous. Check ownership before acquisition and after acquisition, and verify that the returned dependency belongs to the captured session generation. Existing `accountDioProvider(AccountKey)` is account-keyed; the key alone does not prove that a replacement session was not acquired while awaiting it.

Choose the smallest existing-provider extension or lease-keyed adapter that establishes that guarantee. Keep the dependency alive for the operation using the normal Riverpod lifecycle, and release any retention on completion or disposal. Do not create unbounded client caches, extend obsolete client lifetimes, or weaken current close/cancel behavior without evidence that behavior is preserved.

Resolve feature dependencies once, then close over them for retries and subsequent steps. An operation must not reread a global active repository after an `await`.

Guard helpers should cover clearly defined steps and result publication. Do not add checks after every local `await` by convention. Preserve each flow's existing cancellation points first; remove a caller check only after the shared boundary covers the same continuation.

## 6. State, Providers, And Dependency Injection

Keep Riverpod and current repository interfaces. A typical dependency path becomes:

```text
session registry -> captured operation authority
captured session -> fixed-token client -> feature repository
repository + existing mutation token -> shared retry runner
result -> guarded feature state / guarded navigation
```

Use provider families keyed by `AccountSessionLease` where they eliminate shared mutable account state. Identity-only keys can remain appropriate for public data; viewer-specific and private data also need a session or boundary guarantee. Keep active selectors small and consistent with account initialization and routing.

Migrate state families only when they reduce complexity without changing retention behavior. Existing private-state cleanup, account-boundary invalidation, controller resets, and cache clearing remain authoritative. Do not retain old-account private data simply because a family prevents it being displayed.

Request sequence, refresh, entity-revision, and mutation-token checks remain separate. Session identity does not prevent an older response from overwriting a newer response within the same account.

Preserve the current null-ownership allowance used by isolated provider tests while migrating fixtures. Production entry points must still capture authenticated authority. Do not accidentally require secure storage in formerly isolated tests or introduce an implicit anonymous production fallback.

## 7. UI, Widgets, And Routes

No visual redesign or routing change is planned. Keep manual switching to Home, notification recipient activation, unsaved-work confirmation, composer overlays, and existing loading/error presentation.

Widgets should pass captured operation authority into controllers or coordinators rather than rebuild validity closures at every step. Navigation, snackbars, dialogs, and draft cleanup still require a valid originating flow and a live widget/context where applicable.

Account scoping alone does not authorize a stale result to close a new screen or delete another flow's draft. Preserve standard and project composer behavior together so the shared workflow does not diverge.

## 8. Error, Loading, And Edge States

| Case | Planned handling |
|---|---|
| Account changes during dependency acquisition | Discard acquisition for the old operation; never bind its later steps to the new session. |
| Account changes between steps or during a retry delay | Stop at the same boundary as today; leave already-dispatched remote effects alone. |
| Same account logs in again; Alice → Bob → Alice | Reject old session/activation generations even though the DID matches. |
| Late success or failure after disposal | Preserve return and reporting behavior without updating disposed state or another account's UI. |
| Ambiguous outcome or exhausted retry budget | Preserve the existing feature contract. Profile-save loops currently differ from `runPdsMutation` in exhaustion and superseded-error handling; adapt explicitly before consolidating them. |
| Partial profile save or conflicting operations | Keep independent outcomes, typed causes, operation keys, immutable inputs, and controller revision checks. |
| Local discard with diagnostic obligations | Use content-free internal signals, preserve original failure causes/stacks and initiating identity, and retain one issue owner per occurrence. |

A shared runner must not turn previously returned `null`, ignored completions, or per-record failures into a newly visible generic exception. Keep feature adapters where needed instead of forcing uniform behavior.

## 9. Verification Plan

Use existing fake repositories, registry storage fixtures, injected clocks/delays, and completers to pause at meaningful boundaries. Assert requests and state transitions, not just calls to a guard helper. The IDs below describe planned checks, not tests already implemented.

| ID | Scenario | Primary targets |
|---|---|---|
| V-01 | Pause client/repository acquisition, replace the session, then resume. No wrong-generation dependency is dispatched. Inspect fixed-token request headers. | New operation-context suite under `test/auth/providers/`; existing `test/shared/api/providers/` suites. |
| V-02 | Switch during a request, between steps, during a retry delay, switch back, or replace the same account's session. No later guarded write occurs. | Mutation provider suites; account activation tests; profile-save regression cases. |
| V-03 | Late success/failure and provider/widget disposal do not publish stale state or navigate; preserve original return/error behavior. | Provider tests and account-switch routing/widget tests. |
| V-04 | Retry uses the same key and input, preserves budget/delay/exhaustion semantics, and never switches repository ownership. | `test/shared/mutations/pds_record_operation_controller_test.dart`; migrated mutation suites. |
| V-05 | Newer same-account operations/refreshes win; overlays reconcile and expire as before; account reset discards mutation state. | Feed/business read and mutation suites; saved-post revision tests. |
| V-06 | Loading, optimistic actions, separate save outcomes, composer cleanup, and serialized diagnostics retain their behavior. | Profile-save tests; standard/project composer tests; relevant observability suites. |
| V-07 | Inactive-account notification work, billing-owner rules, onboarding, and deletion continue to use their intended authority. | Existing auth/onboarding/notifications/subscriptions/settings suites. |

First test: V-01, proving exact-session dependency acquisition and rejection after session replacement. Then V-02/V-03 for a single ordinary mutation before broad adoption.

Run focused suites after each slice. Initial existing targets include:

```sh
just app-test test/shared/api/providers test/shared/mutations
just app-test test/feed/providers/toggle_post_interactions_provider_test.dart
just app-test test/profile/providers/save_profile_provider_test.dart
just app-test test/feed/providers/create_post_provider_test.dart
```

Expand to affected composer, private-state, and specialized suites as they migrate. Before completion, run `just app-test`, `just app-analyze`, and `git diff --check`; regenerate Riverpod output with the existing build-runner setup when declarations change. Manually exercise account switching during a slow composer upload and a retrying mutation using existing development tools.

No backend changes are planned. If a proposed slice requires an API, persistence, lexicon, or backend change, stop treating it as this client-only refactor and review that additional scope separately.

## 10. Sequencing And Guardrails

1. **Inventory and baseline.** Record every account ownership check and classify it using section 2. Identify its protected side effect, invalidation behavior, and regression coverage. Establish the expected behavior before moving it.
2. **Foundation and one reference slice.** Define the context and exact-session dependency lifecycle. Integrate ownership with the existing retry runner. Migrate like/unlike as a representative keyed optimistic mutation, preserving its visible state and overlays. Proceed only when V-01–05 pass.
3. **Remaining ordinary mutations.** Migrate repost, follow/block, reports and deletion, then create-post, profile, and business writes in small groups. Keep operation-specific adapters for different outcome/error contracts. Remove duplicate plumbing only after each group passes focused tests.
4. **Composers and multi-step workflows.** Migrate standard/project submissions, image/video upload, link thumbnails, drafts, scheduled-post client actions, and Instagram migration. Preserve step ordering, media reuse, timeout/cancel behavior, and successful-origin cleanup. Server-side scheduled publication is unchanged.
5. **Read/state consolidation.** Review feeds, search, relationship lists, business records, and private-state providers. Introduce session-scoped families selectively; retain same-account request revisions, reconciliation, and private-data cleanup.
6. **Specialized authority review.** Review auth/onboarding, notifications, billing, and account deletion separately. Reuse compatible mechanics but retain domain-specific guards. Document why remaining checks are required; these areas are reviewed, not silently exempted.
7. **Final audit.** Re-run the inventory, remove unused helpers and imports, validate every remaining direct check, run the completion checks, and record migrated areas and intentionally retained guards.

Each slice must be reviewable and independently verified. Do not remove all old guards first, change account-switch availability, persist operation contexts, or change retry/cancellation behavior to make the abstraction easier.

## 11. Risks And Design Questions

| Question / Risk | Decision needed |
|---|---|
| Exact-session dependency acquisition and auto-disposal | Resolve in the foundation stage before callers migrate. AccountKey alone is insufficient. |
| Different retry/error contracts | Keep explicit adapters and characterization tests; do not normalize behavior as part of consolidation. |
| Provider-family retention changes | Migrate selectively and verify cleanup; leave an existing boundary in place when a family adds complexity. |
| An overly broad context or callback framework | Keep authority small and feature dependencies explicit. Pilot one slice before fixing the interface app-wide. |
| Count reduction mistaken for success | Measure duplicated lifecycle code and feature complexity, while retaining necessary checks and revision fences. |

No new product decision is required for this outline. Concrete interface and lifecycle choices remain implementation design work, with the preservation constraints above.

## 12. Completion And Handoff

The refactor is complete when all inventoried areas have been migrated or have a documented reason to retain their current guards; subsequent writes use captured dependencies; repeated lifecycle handling lives in shared infrastructure; and existing behavior passes the relevant regression matrix.

Record validation and remaining limitations alongside this document. A later implementation stage can use `05-implementation-plan.md` for slice tracking and evidence. Start with exact-session acquisition tests, then the like/unlike reference slice. This document alone does not advance that stage.
