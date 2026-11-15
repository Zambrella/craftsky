# Subscription-page presentation and web prices

The web subscription page now follows the existing Flutter presentation: a
compact Current access card, the same account-plan explanation, tinted Plus and
Business cards, the CraftSky Plus and storefront icons, monthly prices below
the tier names, matching benefits and View details actions. Business uses the
same gold accent. Complimentary purchase timing uses the same wording.
Provider-specific checkout, restore and management actions retain their existing
platform behavior. Flutter presentation and its RevenueCat prices are unchanged.

Web prices come from the exact server-configured Stripe Checkout catalog rather
than hard-coded client amounts or native-store prices. The new current-member
GET /v1/billing/web-prices endpoint is independent of account access, owner
registration and reconciliation. It exposes tier, integer minor-unit amount,
currency and tax behavior only. The Stripe boundary validates the actual merchant,
mode, active fixed monthly licensed price and rejects unsupported billing shapes.
A five-minute per-client cache bounds successful reads; unavailable prices are
shown explicitly and can be retried without blocking existing access.

The open local page returned verified sandbox prices of GBP 2.49 for Plus and
GBP 8.99 for Business, both tax-inclusive. Browser regression checks cover the
matching copy, actual catalog amounts, mobile width, unavailable-price recovery,
promotional access and purchase ownership protections. Unit checks cover currency
formatting, catalog validation and safe API failure responses. These checks do
not submit a payment or establish production purchase acceptance.

Validation completed: the route, API, subscription and Stripe packages passed
with the race detector against the local Compose PostgreSQL (including the full
API package). Web Svelte checks, ESLint, production build, 52 focused unit/proxy
tests and 22 distinct enabled browser regressions passed; the release-disabled
feature case was intentionally skipped in that enabled run. The open T3 Code
web page was refreshed and confirmed to display both sandbox catalog prices.

## Whole-card activation

Plus and Business cards now activate their existing plan action across the whole
card in Flutter and web. Native cards use an InkWell with a single accessible
card action; the existing visible action retains its callback and is excluded
from duplicate keyboard/screen-reader focus. Loading and unavailable actions
remain inactive. Web uses the existing native button with a stretched hit area,
so pointer and Enter/Space activation retain the checkout guard; keyboard focus
outlines the entire card. No extra navigation or checkout handler is introduced.

Validation: 10 Flutter subscription presentation/accessibility widget checks and
18 web subscription browser regressions passed, including card-body activation,
correct tier, single dispatch, keyboard support and disabled states. Flutter
analysis, Svelte checks, web lint and build passed. The running Flutter app was
hot-reloaded successfully. No payment was submitted.
