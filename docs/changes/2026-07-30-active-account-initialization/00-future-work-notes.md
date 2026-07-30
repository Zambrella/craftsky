# Future Work: Active Account Initialization

## Status

- Date: 2026-07-30
- Status: Discussion note
- Decision state: Not yet approved for implementation
- Origin: Review of the content-language readiness barriers

This document records a possible architecture for account-scoped asynchronous
initialization. It is intended as a starting point for a future requirements and
test-design exercise, not as an implementation plan.

## Problem

Several content providers currently contain a readiness barrier such as:

```dart
await ref.watch(activeContentLanguagePolicyProvider.future);
```

The barrier has two purposes:

1. It prevents a provider from fetching content before the active account's
   language preferences have loaded.
2. It makes the content provider react when the active language policy changes.

This is correct for language filtering, because showing unfiltered content while
the preference is loading could briefly expose posts the user has chosen to
hide. However, repeating the barrier in every affected provider obscures the
application lifecycle and will become increasingly awkward if more
account-critical asynchronous state must be ready before signed-in features can
run.

## Current Architecture

- `appDependenciesProvider` is the process-level initialization gate used by
  `app/lib/app.dart`.
- Synchronous dependency accessors in `app/lib/app_dependencies.dart` use
  `AsyncValue.requireValue` after that gate has succeeded.
- Language preferences are private, account-scoped state loaded or initialized
  by `app/lib/languages/providers/language_preferences_provider.dart`.
- Login is not the only way an account becomes active. The app can restore a
  session on cold start and can switch between accounts through
  `app/lib/auth/providers/account_activation_coordinator.dart`.
- Content-language filtering is enforced by the AppView. The client waits for
  the active policy so that requests are made only after the AppView has the
  correct private preference state.

## Recommendation

Introduce a second initialization boundary specifically for the active account.
Keep it separate from both process-level app initialization and the login
operation.

| Lifetime | Examples | Should block signed-in content? |
|---|---|---|
| App-global initialization | Package/device information, local storage, date formatting | Yes, through the existing app gate |
| Active-account initialization | Content-language preferences and other future privacy- or correctness-critical account state | Yes, through a signed-in account gate |
| Background initialization | Push registration, analytics, non-critical profile prefetch | No |
| Feature-lazy initialization | Data needed only when a particular feature opens | No |

The active-account gate should run for every way the active session can change:

- cold session restoration;
- first login;
- account switching; and
- reactivation after the current session changes.

It should be keyed to the active session lease, rather than only the account DID,
so that a late result from an old session cannot make the new session appear
ready.

## Illustrative Provider Shape

The exact types and provider names should be designed during the future
requirements and coding-plan stages. Conceptually, the coordinator could look
like this:

```dart
@Riverpod(keepAlive: true)
Future<ActiveAccountInitialization?> activeAccountInitialization(
  Ref ref,
) async {
  final registry = await ref.watch(sessionRegistryProvider.future);
  final lease = registry.activeLease?.session;

  if (lease == null) {
    return null;
  }

  final preferences = await ref.watch(
    accountLanguagePreferencesProvider(lease.account).future,
  );

  return ActiveAccountInitialization(
    lease: lease,
    languagePreferences: preferences,
  );
}
```

A gate in or immediately below `MaterialApp.router` would watch this provider
for the signed-in application subtree. It would render:

- the signed-out experience when there is no active account;
- a loading state while account-critical dependencies initialize;
- an error state with recovery actions when initialization fails; or
- the signed-in application once initialization succeeds.

After that boundary, synchronous projections such as
`activeContentLanguagePolicyProvider` could use `requireValue`, and content
providers could watch the synchronous policy without individually awaiting its
future.

## Why This Should Not Be Part of App-Global Initialization

Account-scoped state has a different lifetime from process-scoped dependencies:

- there may be no signed-in account when the app starts;
- the active account may change without the app restarting;
- sign-out must remove the old account's state from the active dependency
  graph; and
- a process-level initialization provider should not have to restart whenever
  the user switches accounts.

The existing app gate remains useful, but it should initialize only dependencies
whose lifetime is the application process.

## Why This Should Not Be Tied Only to Login

Login is only one account-activation path. A login-only initializer would miss:

- a session restored during cold startup;
- switching to an account that is already registered on the device; and
- future activation paths that do not perform a fresh login.

Account initialization should therefore react to the active session source of
truth, while login and account switching may optionally wait for that
initialization before declaring their transitions complete.

## `requireValue` Contract

Riverpod's `AsyncValue.requireValue` is an assertion, not a waiting mechanism.
It returns the existing value or throws when the provider is loading or has
failed.

It is appropriate only where the widget or provider is guaranteed to be below a
successful initialization gate. Code that can run outside that subtree—such as
background work, isolated provider tests, or startup coordination—must establish
the same invariant or continue awaiting the asynchronous provider.

This makes the initialization boundary an explicit architectural contract:
signed-in features can consume initialized account dependencies synchronously,
while the gate owns loading and failure handling.

## What May Block the Gate

Only work required to preserve privacy or the correctness of the initial
signed-in screen should block the gate. Content-language preferences meet that
standard because feeds and discovery must not issue requests using an unknown
policy.

Future dependencies should be evaluated individually. The coordinator should
not become a general list of everything the app would like to preload.

The following should normally remain non-blocking:

- push-notification registration;
- analytics initialization;
- speculative cache warming;
- profile prefetching; and
- data needed only by a feature the user has not opened.

## Failure and Recovery

The gate must not trap the user behind an unrecoverable loading or error screen.
Its error state should provide:

- retry;
- sign out; and
- account switching, where another account is available.

A signed-out state should be considered successfully initialized with no active
account. When the active account changes during initialization, completion from
the previous session must be ignored.

## Possible Migration

1. Write and approve requirements and acceptance tests for the account
   initialization lifecycle.
2. Add the active-account initialization result model and coordinator provider.
3. Add a signed-in initialization gate with loading, failure, retry, sign-out,
   and account-switch behavior.
4. Initially retain the existing language readiness barriers while proving the
   new lifecycle with tests.
5. Change the active language policy into a synchronous projection of the
   initialized account state.
6. Replace the repeated `.future` barriers in content providers with ordinary
   watches of that synchronous policy.
7. Reassess explicit content-provider invalidation only after tests prove that
   the reactive dependency refreshes every required surface.
8. Manually verify cold start, login, sign-out, retry, and account switching.

## Test Inventory

A future implementation should cover at least:

- signed-out cold startup;
- cold startup with a restored session;
- first login;
- switching between initialized accounts;
- switching accounts while the first account is still initializing;
- initialization failure and retry;
- access to sign-out and account switching after a failure;
- rejection of a stale completion from the previous session;
- a content-language preference change causing all filtered surfaces to
  refetch; and
- code outside the gated widget subtree not incorrectly relying on
  `requireValue`.

## Open Questions

- Should the gate cover the entire signed-in application or only content-bearing
  routes inside the application shell?
- Should login and account-switch UI wait for account initialization before
  completing their transitions?
- Which future account dependencies genuinely meet the blocking threshold?
- How should the error screen expose account switching without depending on the
  failed initialized state?
- Can the current explicit content-provider invalidator be removed, or does it
  remain useful as a defensive refresh boundary?
- How should onboarding interact with initialization for a newly created
  account?
- What exact session-lease identity should fence stale asynchronous results?

## Non-goals

This note does not propose:

- changing AppView language-preference storage or API behavior;
- moving private preferences onto the PDS;
- changing the current language-filtering semantics;
- blocking startup on push, analytics, or other non-critical work; or
- implementing the refactor as part of the current language-page polish.

## References

- [Riverpod: Eager initialization of providers](https://riverpod.dev/docs/how_to/eager_initialization)
- `app/lib/app.dart`
- `app/lib/app_dependencies.dart`
- `app/lib/languages/providers/language_preferences_provider.dart`
- `app/lib/auth/providers/account_activation_coordinator.dart`
- `app/lib/feed/providers/timeline_provider.dart`
