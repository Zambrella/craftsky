# Business and subscription review

Reviewed 16 November 2026. Changes are local source changes, with the local
AppView rebuilt and the running Flutter app refreshed. No production deployment,
store release, new charge, subscription cancellation or provider configuration
change was made by this review.

## Scope

Business coverage included declarations and settings, public profile identity
and About sections, product cards and four-photo editors, event creation,
editing, display, lifecycle actions, sharing, record conflicts, image handling
and account boundaries. The Flutter and Svelte implementations were compared
against their AppView contracts and existing parity specifications.

Subscription coverage included per-account identity and store ownership,
catalogue/entitlement mappings, feature gates, access caching, promotional grants,
shared paywalls, purchase/restore reconciliation, pending operations, plan changes,
provider portals, RevenueCat response parsing, webhooks, environment isolation,
direct-checkout recovery, server enforcement and account-deletion guards.

## Tester subscription investigation

The logged-in CraftSkyTester1 account has an active **sandbox Stripe Business**
subscription through **15 December 2026 at 15:14 UTC**. Renewal is disabled, so
both clients correctly say **Access until**, rather than **Renews on**. The
expired Plus record ended on 15 November when that subscription was upgraded.

This was checked against the account's private RevenueCat customer, the local
AppView billing/access responses, both clients' displayed summary and the
authenticated Stripe sandbox portal opened from each client. Stripe also says
the Business service ends on 15 December. No expired Business-access grant was
reproduced for this account, and its valid access was not revoked.

Paid access continues to follow RevenueCat's authoritative `gives_access` flag;
period-end dates alone cannot revoke grace-period or delayed-renewal access.
Promotional grants have hard expiry dates. See the
[RevenueCat subscription model](https://www.revenuecat.com/docs/api-v2/subscription-data-model)
and [ADR 027](../../adr/027-per-did-subscriptions.md).

## Business fixes

| Finding | Change and reason |
| --- | --- |
| Event times used the viewer's timezone while displaying the event's zone. All-day dates used a fixed 24-hour subtraction. | Both clients format in the event's IANA timezone, handle the exclusive all-day end by calendar date, and show spanning dates/years. This prevents shifted dates, incorrect times and DST errors. Missing/invalid zones fall back to UTC. |
| Native event input silently moved nonexistent DST times. | Native keeps entered calendar fields independent of the device timezone and rejects times skipped by DST, matching web's explicit validation. |
| Native rejected valid multilingual photo descriptions. | Image validation now uses the shared 1,000-grapheme/10,000-byte bounds already used by the schema, API and web. |
| Accepted event-photo edits kept an old description. | Optimistic event overlays update the description while preserving remote image references or transient local preview pixels. |
| Conflict reloads could complete after switching accounts. | Native discards late results and failures using the existing account-operation fence. |
| Web product titles had no live bound; some validation errors targeted the wrong control. | Titles use the shared 150-character counter/required styling. Price, venue and event-link errors target their actual fields. Product and event destinations are trimmed before validation and publication. |
| Small selector/copy differences remained. | Business action labels/order, explicit No primary action behavior, empty initial currency and the subscription storefront glyph match native. The Business details settings icon remains its separate identification badge. |
| First-party API writes accepted blank required names. | Product titles and event names reject empty or whitespace-only values with 422 before any PDS effect. Portable schemas and independently authored records are unchanged. |

## Subscription fixes

| Finding | Change and reason |
| --- | --- |
| Shared feature access could stay cached indefinitely. | Both clients refresh at upcoming access boundaries, at a bounded five-minute interval, and after returning to the foreground. Dates trigger fresh server reads rather than locally revoking paid access. |
| Web feature pages could stay locked after a successful paywall purchase. | Settings, customisation, growth and business gates consume shared confirmed access reactively; content is loaded when permission is gained. |
| Billing reads could be discarded by concurrent shared reads, or old renewal metadata could remain indefinitely. | The billing page keeps its account-scoped result, respects newer shared access and independently refreshes billing metadata. A failed reactive reload is attempted once per shared snapshot instead of causing a retry loop; explicit retry remains available. |
| Cached promotional grants could remain displayed after expiry. | Both clients filter expired grants and refresh billing presentation. Promotions still allow an immediate paid subscription with the existing billing-starts-now wording. |
| Native treated paywall Restore as purchasing the displayed tier. | Restore has its own outcome and reconciliation intent. Restoring Plus from a Business paywall no longer waits for a Business purchase; pending restore recovery retains the correct operation across lifecycle/account changes. |
| Native trial/grace/provider states lost their Active presentation. | A valid provider access flag takes precedence over the limited status-label allowlist, while anomaly and payment-pending guards remain. |
| Pending payments without access could lack Manage. | A sole recognized pending subscription offers management for payment recovery. It does not receive an Active badge or advertise another plan change. Stripe purchases continue to use the authenticated Stripe portal. |
| Web asynchronous purchase preflight admitted overlapping attempts. | The paywall coordinator serializes attempts before asynchronous checks and releases its guard after success, failure or cancellation. |
| A retained direct checkout still advertised a new SDK purchase. | New-purchase controls remain hidden until the pending operation is resolved; cancellation/recovery remains available. |
| Direct checkout could create a payable session while another environment had an unresolved operation. | The shared owner checkout fence checks all remaining operations after any current-environment recovery, blocking a second new session across sandbox/production. |
| Duplicate provider rows or null paid product IDs could replace valid access with an anomaly/unsupported product. | RevenueCat parsing rejects the complete inconsistent enumeration. Previously confirmed access, pins and reconciliation generations remain intact until a valid refresh. Promotional null product IDs remain supported. |
| Uncertain mobile access refreshes could unmount a dirty Business form, and initial retry could seed an empty declaration. | Existing drafts remain mounted while permission is uncertain, with saving gated on confirmed access. A business form is not initialized before its first confirmed authorization, preserving the existing declaration on retry. Confirmed revocation retains the existing routed-page exit policy. |

## Live checks

- Compared the logged-in browser and iPhone simulator business settings, product
  editors and subscription cards. The existing product's four photos render in
  both editors; fields, limits, media management and primary actions align.
- Saved a temporary tagline through web. A stale native save produced the
  expected version conflict and retained the draft instead of overwriting the
  server. The temporary edit was restored to the original text; native edits
  were discarded. No other business record was intentionally changed.
- Both clients show the same current Business access/date, Active/Manage and
  alternative Downgrade action. Both Manage flows opened the account-specific
  Stripe sandbox portal, which confirmed the same cancellation date.
- Both event managers showed the same empty Upcoming and History state.

## Validation

- Native business/profile/route suite: 265 passed, two existing platform skips.
  Another 40 event tests passed with `TZ=America/Los_Angeles`.
- Web unit suite: 606 passed. The business/subscription/settings browser matrix
  covered 98 enabled cases plus the separately exercised disabled-billing case.
- Final native subscription/profile/shell suite: 371 passed, one existing skip.
  Native business and subscription/profile/shell static analysis were clean.
- Two additional browser regressions passed for failed-reload retry bounds and
  cancellation metadata changes that leave effective access unchanged. Web type
  checking reported zero errors/warnings; lint/build were clean.
- The full 35-case billing browser suite passed again after those final fixes.
- Backend checks ran with local PostgreSQL required and the race detector across
  subscription, RevenueCat, Stripe, business, routes, app wiring and relevant API
  tests. Additional real-database preservation cases cover malformed provider
  snapshots and partial pagination.

These checks establish local behavior and the inspected sandbox management
flows. New App Store/Google Play transactions, Android device UI, production
webhook delivery and production deployment were not exercised. The provider
catalogue/offerings were inspected read-only; no shared paywall was published.
