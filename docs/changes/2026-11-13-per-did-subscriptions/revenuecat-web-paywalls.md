# Shared RevenueCat web paywalls

15 November 2026. The maintainer accepts RevenueCat's ordinary Stripe Billing
fallback when Managed Payments is unavailable. This supersedes the earlier
strict-MoR requirement for new web purchases; retained direct Stripe sessions
keep their original recovery behavior.

The web client loads RevenueCat Web SDK 1.74.0 only when opening the Plus or
Business bottom sheet. Purchase selection opens the larger Stripe checkout
modal, and provider cancellation restores the sheet. It presents the same
published offering/paywall as Flutter and
uses the authenticated owner's private RevenueCat UUID. Sandbox and live Stripe
products are attached to the existing monthly packages without changing native
products. Environment-specific public SDK keys select the Stripe app/catalog.

A new authenticated `POST /v1/billing/web-paywall` accepts only `tier`, checks
fresh billing state, existing subscriptions and retained pending operations,
and returns no-store SDK configuration. Every SDK purchase rechecks server
state; account changes, disposal and concurrent cooperating browser tabs are
guarded. The pinned SDK's public purchase dispatch is covered by a contract
check because it is wrapped for asynchronous preflight.

Provider results request reconciliation and never directly grant access. The
SDK owns its Checkout sessions; AppView's durable direct-session fence, pending
flag and cancellation endpoint cannot track or expire those sessions. Cross-device
and uncertain-payment acceptance remains necessary before production activation.
Existing receipt-management fallback handles SDK-created subscriptions without
a direct-checkout customer mapping.

The maintainer explicitly approved publishing both corrected billing paragraphs.
Published on 15 November 2026: Plus revision 17 at 11:33:08 UTC and Business
revision 16 at 11:33:23 UTC. Published-component read-back verified that only
the English billing paragraph and revision changed for each paywall. Dynamic
prices, layout, components and all three Outfit font mappings were preserved.
The shared copy now describes account ownership, immediate monthly billing and
cancellation through the payment provider. No application deployment is included.

Validation: 61 focused web unit tests, web type checks, build and 19 subscription
browser tests. A live sandbox render confirmed Plus £2.49/month and Business
£8.99/month with their published features. Business proceeded to a sandbox
embedded Stripe Checkout within the modal; no final payment was submitted.
Go subscriptions, API and routes passed with the race detector and local database;
configuration-focused checks passed after isolating the new default. Documentation
index and catalog checks passed. RevenueCat sandbox settings were read back with
Managed Payments when available enabled for both sandbox and live apps.

Bottom-sheet follow-up validation: 8 focused paywall lifecycle tests and 21
subscription browser tests passed, including sheet geometry at 390px and 1280px,
checkout target separation, cancellation restoration and payment completion
reconciliation. Web build, type and lint checks passed. Live sandbox DOM read-back
confirmed a 720px sheet flush with the viewport bottom, followed by a 1000px
centered Stripe checkout with the paywall hidden. No payment was submitted.

Web UI cleanup: removed the outer sheet header, Close control and drag-handle
strip so the RevenueCat paywall fills the sheet. Removed the web Refresh status
button and Last checked timestamp; automatic load, restored-page and purchase
reconciliation remain in place.

Checkout-close correction: the larger Stripe checkout has a sticky Close checkout
control, while the bottom sheet retains only RevenueCat chrome. Closing disposes
the flow and restores focus without calling direct-session cancellation or
granting access. Desktop and mobile browser coverage checks this dismissal.

Desktop-checkout correction: increase the checkout cap from 1000px to 1200px.
The former cap left a 982px provider frame with a vertical scrollbar, below
Stripe Checkout's 991px desktop breakpoint. Wait for the Svelte layout update
before mounting the provider; narrow windows retain responsive mobile checkout.
Validation: 22 subscription browser tests cover 1280px, 1050px and 390px viewports,
including available checkout width with forced vertical overflow, closing and
focus restoration. Nine lifecycle tests, web build, type and lint checks passed.

The bottom sheet dismisses when a primary pointer starts and clicks on its
backdrop. Clicking content or dragging from content onto the backdrop preserves
the sheet. Checkout keeps its explicit Close checkout control. Browser coverage
checks dismissal and focus restoration at desktop, intermediate and phone widths.
