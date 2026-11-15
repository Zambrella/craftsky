# Web subscription feature prompts

The explanation/Learn more flow below was superseded on 15 November 2026 by
[direct paywall entry on both clients](feature-paywall-entry.md).

The web client uses one subscription-feature prompt for scheduling, Customisation,
Growth, Pin and saved-post folders. The copy matches Flutter: explain the required
tier, offer Cancel and Learn about subscriptions, and open the web subscription
page only after the user chooses to learn more. Unknown access shows an unavailable
message instead of treating an account as Free. A disabled subscriptions release
flag keeps the existing coming-soon dialog.

The Saved posts app bar always exposes its folder button. Free accounts see the
CraftSky Plus badge and the upgrade prompt; Plus and Business accounts open folder
creation directly. Locked New folder and Move to folder actions remain discoverable
from posts. Business and promotional access use the server-confirmed effective tier.
Prompts use the existing focus-restoring dialog and cancel when their component is
disposed. Server entitlement checks remain authoritative.

Validation: Svelte check, ESLint and the web production build passed. Browser
fixtures cover free prompts, cancellation/focus restoration, navigation to billing,
Plus and Business folder creation, unavailable access, nested Save dialogs, Pin
and Move to folder. Release-disabled feature and beta-gating checks passed. Existing subscription checkout, promotional access, live access
refresh and authenticated audit tests also passed. These are local fixture checks,
not payment-provider or production acceptance.
