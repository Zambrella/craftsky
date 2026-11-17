# Embedded web checkout and pending-purchase recovery

The account-plan wording now says: Your subscription belongs to this CraftSky
account. Business includes everything in Plus.

The reported mobile card failure was reproduced on the running iOS simulator:
the account was Free with no paid subscriptions, but its unfinished Stripe
checkout kept `webCheckoutPending` true. The native duplicate-purchase guard was
working; the page silently omitted actions and provided no recovery explanation.
Both cards now open a pending-purchase explanation with Cancel pending purchase,
and the page exposes the same recovery action. The ordinary native paywall flow
and receipt/customer ownership guards remain intact.

Web plan selection opens Stripe Embedded Checkout inside a large accessible
modal, full-screen on narrow viewports. The server returns a matching-mode
publishable key and ephemeral client secret after verifying Managed Payments,
Stripe tax liability and the original account UUID. It retains the exact
original provider request and idempotency key, including presentation mode.
Migration 000127 adds that frozen mode and refuses rollback while embedded
operations remain. Hosted sessions are never put inside a generic iframe.

Closing an unpaid modal uses a new authenticated cancel operation. It never
accepts a caller-selected provider session. Only provider-confirmed expiration
releases the pending flag; a concurrent completed payment or unknown outcome
remains protected. Completion refreshes AppView/RevenueCat access, not client
state alone. SDK instances are destroyed on close, disposal and account changes;
capabilities are never persisted, logged or included in navigation URLs.

Local validation: 19 web browser fixture regressions, 106 web unit/proxy tests
and 15 native presentation/API/accessibility checks passed. The running iOS app
was hot-reloaded and its card was tapped to verify the new pending explanation.
Provider sandbox activation still needs the publishable key. Expiring the user's
existing unpaid sandbox checkout requires the separately requested approval;
automatic approval review rejected that tap and it was not retried.

The affected subscription, Stripe, API, route, account-deletion, database and
application suites also passed with the race detector against local PostgreSQL.
The additive migration was applied to the local Compose stack, and the backend
was rebuilt. No production deploy or configuration change was performed.

After explicit maintainer approval, the pending unpaid sandbox checkout was
expired through the native recovery action. The authenticated billing response
confirmed `webCheckoutPending: false` and no paid subscriptions. Both Plus and
Business RevenueCat paywalls were then observed opening on the running iOS
simulator. They were closed without submitting purchases. The remaining actual
web-overlay check still requires the sandbox publishable key.

The provider paywalls displayed historical independently assignable licence
copy in their footers. That dashboard-managed copy was observed but not edited
as part of this cancellation verification.
