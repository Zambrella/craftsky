# Local subscription development

Development defaults enable subscriptions, with RevenueCat's synthetic Test
Store on Flutter and sandbox access in AppView. The Test Store uses its public
SDK key and does not require an Apple/Google sandbox account. Its Plus and
Business monthly products remain attached to their existing offerings.

Web uses real Stripe sandbox Checkout with Managed Payments required, rather
than a simulated browser purchase. Sandbox account, price and portal settings
are server-owned; the secret is saved only in ignored AppView configuration.
HTTP return URLs are allowed only for sandbox loopback addresses. Production
configuration, release flags, store catalogs and provider permissions are not
enabled by this change.

## Start and test

From the repository root:

```sh
just config-check
just dev-d
just web-run
# In another terminal, choose an iOS simulator or Android emulator/device:
just app-run
```

The launchers discover this checkout's ports. At setup verification the web app
was `http://127.0.0.1:25029`, with AppView on port 18110. Sign in to CraftSky,
complete onboarding and open its subscription screen. Mobile purchases open
the Test Store dialog; choose a successful/failed/cancelled test outcome. Web
purchases open Stripe's sandbox; use card `4242 4242 4242 4242`, any future expiry
and three-digit CVC. Check RevenueCat sandbox records and AppView access before
calling a purchase successful. Relaunch mobile after build-profile changes.

## Local webhook delivery

For immediate callbacks, keep a webhook-only proxy/tunnel running:

```sh
./scripts/revenuecat-webhook-dev --upstream-port 18110 \
  --state-file /private/tmp/craftsky-sandbox-webhook-tunnel.json
```

Use the AppView port discovered by your launcher, not another worktree's port.
The existing receiver named `Local sandbox t3-4951d617` was updated to the new
endpoint with its sandbox filter retained. Authorization and HMAC verification
remain enabled. The proxy forwards only POST requests to the exact webhook
path, preserving signed bytes; ordinary API/auth routes are never forwarded.
The private state file contains the endpoint for dashboard/CLI configuration.

Quick-tunnel URLs change when restarted: update only the intended sandbox
receiver's URL and keep its credentials and environment filter. The tunnel must
stay running for callbacks. Development reconciliation also runs every minute
as a fallback, and purchase/refresh actions request reconciliation explicitly.
No public tunnel exposes the full local AppView API.

## Setup verification

- The supplied Stripe test key matched the existing CraftSky sandbox account;
  active GBP 2.49/8.99 monthly prices and a cancel-only sandbox portal were
  verified through read-only provider requests.
- Running AppView configuration validates; its public health endpoint reports
  a healthy database and connected Tap. The local web app serves the sign-in
  route. Signed test notifications returned 200 locally and through the newly
  configured public webhook-only tunnel. The saved RevenueCat server key also
  authenticated a read-only subscription request for a nonexistent customer.
- Native configuration summaries select enabled subscriptions/Test Store for
  development and disabled subscriptions for the production example.
- Regression checks passed: 17 native activation/ownership tests, 31 web
  billing/identity tests, five Node development-mode/profile tests, 18
  configuration tests, four mobile launch-summary tests, and the webhook-only
  proxy regression. Checkout/configuration/Stripe tests passed under the race
  detector, including loopback-only sandbox return URLs and production denial.
  Svelte check, lint, production build, formatting, configuration validation and
  archive checks passed. The complete AppView configuration package passed
  under the race detector after its environment-isolation fixture was updated
  to clear RevenueCat settings and supply credentials only where needed.

These checks establish configuration and callback connectivity. They do not
claim a completed purchase through a device or authenticated browser session.
