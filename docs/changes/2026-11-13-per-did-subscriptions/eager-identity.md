# Account activation and RevenueCat identity

Date: 2026-11-13
Scope: local implementation; no deployment or provider mutation

## Behavior

- With the existing integration flag enabled, eligible signed-in accounts that
  completed onboarding begin optional identity setup when they become active.
  This includes sign-in, switching and restored sessions on app launch.
- AppView creates or recovers the account's permanent private customer UUID and
  durably queues initial RevenueCat registration. The existing reconciler
  creates the exact customer if absent; there is no provider call in sign-in.
- Completed accounts are not requeued on every activation. Pending registration,
  lease and retry backoff survive repeated setup calls. An unreconciled account
  with no outstanding request can be queued again.
- Flutter attempts native SDK identification through the existing serialized
  ownership guard. First configuration carries the UUID explicitly; later
  switches use direct login. Guests never get an anonymous customer.
- Svelte registers the customer through authenticated AppView and refreshes its
  shared access projection; it does not add a RevenueCat browser SDK.
- Identity setup is separate from account readiness. Failure or a native store
  conflict does not prevent signing in, switching or using AppView access.
  Current-session and disposal checks discard late results from old accounts.
- Billing-screen retries and operation-level identity/ownership checks remain
  in place. Existing enablement flags and provider configuration are unchanged.
- Promotional entitlements and the proposed first-1,000 campaign are separate
  work; this change prepares customer identity only.

## Verification

Passed on final source:

- Real PostgreSQL race suites: complete subscriptions (9.497s) and RevenueCat
  integration (4.891s), plus subscription/billing/webhook/owner-refresh routes
  (2.090s). New coverage proves atomic queue creation, stable UUIDs, concurrent
  repair, retained leases/backoff and exact-UUID registration after provider
  failure/retry without an SDK.
- Flutter: 213 passed, five existing skips across subscriptions, active-account
  initialization, its readiness gate and AppShell switching. Fourteen new cases
  cover sign-in, switching, cold restoration, deduplication, flags/guests,
  onboarding/eligibility, failures, store conflict, delayed ownership, stale
  sessions, SDK serialization, shared access and retained HTTP-client lifetime.
  Full Flutter analysis reported no issues.
- Web: 12 focused unit tests, Svelte check with zero errors/warnings, lint,
  formatting and production build. Fourteen browser fixtures passed (six eager
  identity cases plus eight existing billing flows), and the disabled-flag case
  also passed separately. Live authorization, disposal, account and revision
  checks reject late responses.
- Cross-component review found no blocking issue. It confirmed setup remains
  outside readiness and prompted a live-eligibility/expired-state check before
  web follow-up requests. The native repository is watched during setup so its
  account HTTP client cannot be disposed in the middle of the request.
- Final whitespace checks passed. No routes, migrations, dependencies, native
  bridges, lockfiles, provider configuration or enablement flags were changed
  for this identity lifecycle update.

Commands:

```sh
./scripts/appview-test focused -p=1 -race ./internal/subscriptions ./internal/integrations/revenuecat
./scripts/appview-test focused -p=1 -race ./internal/routes -run 'Subscription|Billing|RevenueCat|OwnerRefresh'
cd app
flutter test --no-pub --concurrency=4 test/subscriptions test/auth/providers/active_account_initialization_provider_test.dart test/auth/widgets/active_account_initialization_gate_test.dart test/router/app_shell_account_switcher_test.dart
flutter analyze --no-pub
```

The normal browser fixture port was occupied by an unidentified listener. The
same fixture suites ran through temporary configuration copies on ports
19856/19857; those copies were removed and the existing listener was untouched.
The browser command was `npx playwright test --config <temporary-config>`;
the disabled case additionally used `WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false`
and `--grep 'respects the subscriptions release flag'`.

These are local tests with fixture provider responses and SDK adapters, not
RevenueCat dashboard or real-store verification. A DTD daemon was available for
another checkout only, so no matching app was hot reloaded and the other app
was left untouched. No deployment, commits or provider mutations were performed.
