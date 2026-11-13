# Per-DID subscription implementation and validation

Date: 2026-11-13. Local implementation; not deployed.

## Behaviour

- One stable, private RevenueCat customer UUID per DID, recovered through
  authenticated AppView. No billing-holder selection or licence assignment.
- Plus and Business are alternative tiers. Native purchases, restores and
  management retain the current customer's SDK identity through serialized
  operations and active-session checks.
- Native read-only ownership checks fail closed for conflicting or unverifiable
  store transactions. A known DID/handle is displayed only by matching verified
  store ownership to another retained account's private local customer mapping.
- Native plan changes use explicit Google replacement information and a
  StoreKit same-group check on iOS. Current separate Apple groups are refused.
- AppView exposes the provider-confirmed scheduled tier separately from current
  paid access, so a deferred downgrade does not remove Business benefits early.
- Web checkout uses server-created Stripe Managed Payments sessions, authorized
  for the current DID by AppView. Existing/current/pending/unknown or stale
  provider state blocks checkout. Durable pending operations and provider
  idempotency protect concurrent requests and uncertain retries. Management
  uses a server-selected customer-specific Stripe portal session. Web tier
  changes remain deferred; mobile contains no web purchase promotion.
- Additive `BillingState.webCheckoutPending` reflects retained web-checkout work
  across environments and blocks native repurchase before RevenueCat indexing.
  Disabling provider configuration does not clear it. Unresolved operations also
  block account deletion; deletion reconciliation reads known sessions without
  creating unknown-response sessions.
- API writes retain authentication, suspended-account restrictions, error
  envelope and no-store handling. Browser requests remain bound to the page's
  explicit account header rather than another tab's active-account cookie.
- Migration 000124 refuses ambiguous historical assignments/concurrent paid
  subscriptions, preserves customer UUIDs/history, auto-binds the internal
  entitlement projection and enforces DID ownership in PostgreSQL.

## Initial per-DID checks completed before Stripe replacement

These checks covered the initial per-DID implementation, including its disabled
Paddle flow. They are historical baseline evidence and do not substitute for
validation of the subsequent Stripe implementation.

### AppView

Passed real local PostgreSQL integration suites for subscriptions, RevenueCat
provider handling, API, routes, app configuration/wiring, Business and account
deletion. Passed subscription/provider race tests, all-package compilation,
`go vet`, module verification, tidy diff, gofmt, project catalog and whitespace
checks. This is affected-package validation, not a complete appview-check,
release, Docker image, production migration or deployment.

After adding the pending-tier field, focused billing route/app/API checks and
subscription/provider race tests passed again. The provider-to-Postgres-to-JSON
regression covers scheduled downgrade, unknown and wrong-app pending products,
cleared changes and the final effective downgrade.

### Svelte web

Passed full Vitest suite: 529 tests across 76 files. Passed Svelte check with
zero errors/warnings, ESLint, formatting and production build. Passed six
subscription browser fixture tests against SSR preview and the same six against
Cloudflare Worker fixtures. These are local fixture tests, not live Paddle.

After adding the scheduled native-tier display, Svelte check and ESLint passed
again. Neither checkout return parameters nor a browser fixture grant access.

### Flutter and native

The focused subscription/authentication regression suite passed 161 tests with
five pre-existing skips, including native replacement parameters, wrong-group
refusal, account-switch fences, Customer Center ownership, same-ID upgrades and
provider-confirmed scheduled downgrades. Flutter analysis passed with no
issues. The iOS Simulator Runner.app and Android debug APK built successfully,
including the native store preflight and explicit plan-change adapter. No live
Dart Tooling Daemon application was available for hot reload; no store purchase
was performed.

## Independent review

Read-only reviews covered AppView/Svelte ownership, migration, lifecycle and
checkout; Flutter SDK serialization, native ownership, plan replacement and
Customer Center. Findings addressed during implementation:

- Block cross-provider native purchases so Apple cannot duplicate a current
  Google/web subscription, and vice versa.
- Apply ownership preflight to Customer Center's embedded restore/purchase
  actions as well as explicit purchase/restore controls.
- Recheck the active session after asynchronous plan/group lookup immediately
  before dispatching a native purchase.
- Recognize same-ID tier changes and provider-confirmed scheduled downgrades
  during reconciliation without granting the future tier early.

Cross-tab browser review verified the existing explicit account-cookie selection
already prevents a stale page A from checking out for active cookie B.

## Stripe replacement validation

### AppView

Ran the complete PostgreSQL/MinIO Go race suite. It found three outdated test
fixtures: the exact suspension list omitted the new management route, while
Business membership and scheduled-publication tests still attached licences
from a different billing DID. Both ownership fixtures now use the subscribing
DID; the scheduled fixture also includes the pending-tier column needed by
snapshot application. All three failed tests passed focused race reruns. The
other packages passed the full run; a second complete run was not performed.

After the final checkout fixes, the complete subscriptions package passed with
the race detector. Final filtered API, routes, app configuration, account
deletion, Stripe integration and database migration checks also passed with
the race detector.
Coverage includes durable uncertain retries, competing tiers, exact merchant
and environment, explicit Managed Payments and Stripe tax liability, private
customer metadata, customer-specific management, single-connection operation,
concurrent suspension and post-cancellation reconciliation generations.
The migration rollback regression passed with real PostgreSQL: a nonempty
operation table retains recovery and the native purchase guard, while an empty
down/up cycle preserves the customer UUID. Go vet and the project catalog check
passed; the development configuration tests passed all 18 cases.
The completed provider sandbox purchases are recorded separately in
[provider setup](provider-setup.md).

### Svelte web

Passed 99 focused Vitest tests for API, settings and billing behavior, Svelte
check with zero errors/warnings, ESLint, formatting and a production build.
Six authenticated subscription Playwright fixture tests passed. After adding
the optional pending-checkout field, its final 13-test eligibility/navigation
subset passed again, including same-tier resume while pending. These checks
use local provider fixtures; they do not submit a Stripe payment.

### Flutter

The final subscription and account-switching suite passed 165 tests with five
pre-existing skips. It covers pending-web-checkout gating on both native stores,
strict decoding of the pending flag, Stripe subscription eligibility and the
absence of mobile web-purchase promotion. Flutter analysis passed with no
issues. Mapper/provider generation completed and generated files were retained.
No running Dart Tooling Daemon application was found; hot reload could not run.
No native store purchase was performed.

### Integrated review

Reviewed server-owned identity, merchant/environment checks, explicit Managed
Payments, URL capability handling, frozen retries, concurrent tiers and the
deletion/suspension boundaries. Account deletion cannot discard any unresolved
checkout, including another environment's retained operation. Suspended
reconciliation can inspect a known session without dispatching a new checkout.
Checkout, resume, management and cleanup reuse one pooled connection, verified
with a single-connection pool. Suspension before any remote dispatch clears only
the newly inserted attempt; known and uncertain provider attempts remain retained.
Native pending-state protection does not make independently started native and
web payment flows atomic; real store/application sandbox validation remains a
release gate.

## External work and limitations

See [provider setup](provider-setup.md). Live Stripe verification, catalog and
Managed Payments readiness, RevenueCat live app/entitlement mappings and the
customer portal were inspected or configured. After the maintainer completed
RevenueCat password verification, its live Stripe signing secret was saved and
verified after a reload, and the existing Stripe webhook was enabled and read
back. Live event delivery remains untested. The separate RevenueCat-to-AppView
production receiver and development receiver filters still require configuration
before live purchases. Production activation
and application checkout/management validation remain separate, alongside the
Apple single-group transition, missing Google products and real native-store
sandbox tests. The three documented
Stripe-to-RevenueCat sandbox purchases verified provider processing and customer
isolation; they did not exercise the application's DID mapping or Checkout
service. No production database migration or release was performed. No live
payment or cancellation was tested.

The current conservative web deletion gate can require the paid period to end
after cancellation at period end, because its completed checkout recovery row
remains retained until Stripe cancellation and non-blocking RevenueCat state
are established. Include that lifecycle in application sandbox acceptance.
