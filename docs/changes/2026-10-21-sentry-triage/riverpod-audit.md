# Riverpod lifetime follow-up

The deeper review found real ownership mistakes in the affected and adjacent mutation paths. Mounted checks alone were not sufficient for those paths. This audit covers the Sentry-identified feed/project/thread providers, like/repost/follow/mute mutations, onboarding, profile image dependencies and relationship rendering. It is not a claim that every provider in the application has been exhaustively audited.

## Confirmed misuse and fixes

1. **Like, repost and follow writes depended on a UI listener.** Their generated providers are auto-disposed. Closing the last listener while a submitted write was pending disposed the notifier; account-operation guards then discarded successful completion, so the shared command controller never received the accepted overlay. Three new tests failed before the fix. Each mutation now holds an operation-scoped `ref.keepAlive()` link and closes it in `finally`, matching the existing delete/block pattern. Explicit invalidation still disposes the old instance; account changes retain their existing authority checks. Successful completion releases retention, allowing normal disposal.

2. **Mute/unmute acquired a resource-owning dependency with `read` only.** The persistent relationship controller read `accountRelationshipRepositoryProvider(account).future` without retaining that auto-disposed provider. The repository owns a dependency on `accountDioProvider`, whose disposal closes its HTTP client. A new test confirmed the repository disposed while the request was still pending. The controller now listens to the repository for the duration of the operation, closes that subscription in `finally`, and checks authority after acquisition before dispatch. A persistent parent does not retain a dependency merely by reading it.

3. **Onboarding assumed `keepAlive` also protected against invalidation.** A completion retry could wake after explicit invalidation and read the disposed `Ref`. A fourth test in the initial batch reproduced that error. `_ownsLease()` now checks `ref.mounted` before reading the registry; completion rechecks ownership after asynchronous repository acquisition. The profile-save continuation also checks its active ownership before accessing mutation dependencies. Retention does not override account/session invalidation or provider recomputation.

4. **Widget builds initialized shared relationship state through deferred microtasks.** Post cards, repost headers, notification rows and the profile body scheduled provider writes from rendering. Those callbacks could outlive the consumer and caused the reported widget-ref failures. The deferred initialization has been removed entirely, superseding the earlier mounted-only patch. Rendering now reads an existing overlay or uses the supplied server fallback. When the user chooses mute/unmute, the controller accepts that server baseline at the action boundary, owns the optimistic transition and restores the baseline on failure. A new widget test verifies rendering does not initialize shared relationship state; a controller test verifies rollback without prior widget initialization.

## Expected disposal, verified

Project feeds, authored comments, repost lists and comment sections have active `watch` consumers. Their repository/API dependencies are watched, and ordinary post/project repositories are persistent. No missing listener was found in those read chains. Eight tests exercise both keeping a listener during a delayed fetch and removing it: each read remains alive while observed and rejects its obsolete continuation with `ApiCanceled` after its listener leaves. Keeping those reads alive indefinitely would retain query-specific state after navigation without resolving an ownership defect.

The existing `ActiveAccountOperation` checks remain appropriate for these reads: they prevent post-await overlay reads/publication after the original provider or account activation has ended. The timeline already has equivalent disposal regression coverage.

Onboarding avatar selection already retains its account-specific picker with an operation-scoped subscription. That picker watches its account HTTP dependency and owns upload-client cancellation. The profile-editor picker has a persistent parent that watches its upload client. Existing avatar lifecycle tests cover picker retention during native selection, cancellation after leaving onboarding and the profile-editor path. No blanket provider-retention change was added to image selection.

The relationship overlay cache remains deliberately account/target scoped and explicitly invalidated at account boundaries. This is shared mutation/reconciliation state, rather than a replacement for per-screen read-provider lifetimes.

## Validation

- Five newly added lifetime regressions failed before their fixes: three last-listener mutation cases, onboarding retry invalidation, and mute repository disposal.
- The affected provider, project, profile, post-card, notification and onboarding suites passed: **400 tests**.
- Added coverage includes mutation-retention release after success, explicit invalidation despite retention, repository-retention release, baseline rollback, pure rendering and eight read-lifetime cases.
- Final Flutter analyzer and diff-check results are recorded with the follow-up in [validation.md](validation.md).
- These are local tests. No live Sentry recurrence, physical-device navigation or deployment was performed.

## Riverpod guidance checked

The changes follow the current guidance on [automatic disposal and recomputation](https://riverpod.dev/docs/concepts2/auto_dispose) and [avoiding provider initialization from widgets](https://riverpod.dev/docs/root/do_dont). The installed Riverpod 3.4.3 source was also inspected for `Ref.mounted`, `Ref.read`, subscription ownership and the generated providers' `isAutoDispose` configuration. Riverpod's [experimental Mutation API](https://riverpod.dev/docs/concepts2/mutations) addresses this class of operation lifetime, but no migration was necessary to correct ownership in the existing command architecture.
