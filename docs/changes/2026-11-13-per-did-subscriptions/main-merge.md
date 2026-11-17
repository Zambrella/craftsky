# Main integration, 14 November 2026

Integrated `origin/main` at `71efd3a0` into the subscription branch based on
`c7d0dcf8`. The merge keeps account-owned subscriptions, promotional access,
guarded native purchase recovery and optional identity synchronization on account
activation while incorporating main's lifecycle, diagnostics, refresh,
pagination, hosting-settings and repository changes.

## Conflict resolutions

- Kept the removed cross-DID assignment controller, orchestration and backend
  assignment test removed. Ported main's relevant original-cause reporting and
  lifecycle checks into the current owner-only purchase, restore, management and
  setup paths instead of restoring the retired flow.
- Kept RevenueCat customer registration after an initial 404, complete grant
  decoding/pagination and expiry validation. Integrated main's typed provider
  errors into the shared fetch path, nested grant pages and customer creation;
  malformed raw product/expiry values preserve typed JSON causes safely.
- Updated subscription HTTP error callers to retain request context and main's
  failure ownership. Coverage uses current access, checkout and management
  handlers rather than removed assignment handlers.
- Retained shader profile backgrounds alongside main's profile pagination and
  refresh changes. Auto-merged shell changes retain account/session/disposal
  fencing for identity and shared access. Optional web billing setup now also
  reports consumed startup failures through main's privacy-filtered diagnostics;
  late-account and reporting failures cannot affect readiness.
- Regenerated native provider outputs and the configuration reference from the
  combined sources. Migrations 124–126 have no incoming numbering collision.
- Added this package to main's new change archive and linked the historical
  assignment packages to the current account-owned decision.

## Validation

- All AppView packages compile. Full subscription, RevenueCat and Stripe race
  suites pass, together with focused API, route, migration, deletion, Business
  and scheduling checks. Full runtime, observability, middleware and testdb
  suites also pass. New tests cover current billing-handler diagnostics and
  cause preservation through nested grant pages and customer creation.
- Flutter: the full suite passes 4,259 tests with 44 skips. The final
  subscription suite passes 222 tests with five existing skips; whole-app
  analysis including Riverpod lint is clean. Native bridge files are unchanged,
  so this integration did not repeat iOS/Android native builds or store-device
  purchases.
- Web: Svelte check has no errors/warnings; lint, production build and all 578
  unit tests pass. Authenticated browser checks pass 61 billing, promotion,
  identity, settings and retained-profile/feed cases plus the disabled-release-
  flag case. The final startup-reporting adaptation passes 23 focused tests,
  check and lint. Temporary browser fixtures used isolated ports 19866/19867 and
  were removed afterward.
- Generated configuration validation and the change-archive checks pass. The
  archive test suite passes 19 tests. Both regenerated conflicted native provider
  outputs match main; core subscription models, migrations and billing/grant
  rules remain unchanged from the subscription branch.

No running debug application was available for hot reload. This is a local
branch integration, without a push, deployment, flag activation, provider change
or real purchase.
