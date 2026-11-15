# Direct paywalls from paid features

15 November 2026. Tapping a locked paid feature now opens its RevenueCat paywall
in place on both clients, replacing the explanation and Learn about subscriptions
step. Existing paid affordances (scheduling, folders, Pin, Customisation and
Growth) require Plus and open the `plus` offering. Business includes Plus access.
The host accepts tier selection so a Business-specific entry can use `business`.

Web uses an account-scoped host in AppShell with the same RevenueCat sheet and
Stripe checkout component used by billing settings. It reconciles AppView state,
obtains the identified paywall configuration and repeats preflight before SDK
purchase. It disposes on account changes or navigation, ignores late work and
restores the originating control's focus on dismissal. Nested Save dialogs remain
open underneath. Only reconciled server access updates the shared feature gates.

Flutter feature entry uses SubscriptionPage's existing purchase controller in
modal mode: load/setup and pending/ownership checks precede the native paywall;
no subscription screen is shown for an eligible purchase. Cancelled and completed
flows return to the feature's origin. Errors or blocked billing retain the
subscription recovery UI; unconfirmed work clears on disposal and confirmed work
retains its reconciliation state. Account-switch checks apply before dispatch.

Unknown access still shows an unavailable/retry prompt. A disabled release flag
still shows Coming soon. Owned features execute normally without a paywall. Entry rechecks fresh access
and skips the provider if a paid or promotional grant now covers the feature.
No new HTTP route, schema, catalog edit or production deployment is included.

Validation: web build, type/lint and 31 browser tests passed (one disabled-flag
case skipped in the enabled run). Browser coverage includes direct saved/settings/
post entry, nested-dialog restoration, paid access bypass, unavailable state,
provider identity, mismatched access rejection, fresh-access bypass and subscription reconciliation. Flutter targeted analysis and
34 widget/controller tests passed (one opposite-flag case skipped), including
actual feature entry, provider cancellation, simulated purchased-state access
refresh and disposal while the offering is pending. Hot reload succeeded. Disabled-flag checks passed separately (one web browser
case and two Flutter cases), with opposite-flag cases intentionally skipped.

Live local previews verified Create folder opening the published Plus paywall on
web without leaving `/saved`, and Customisation opening the native Plus paywall
in the iOS simulator directly from Settings. Both were dismissed without payment.
The running native SDK displayed older cached footer copy; the dashboard publish
read-back from the preceding change remains the catalog evidence. Android UI and
real store purchases were not exercised in this pass.

## Plus affordances and mobile saving follow-up

Web Settings (Customisation and Growth) and the composer scheduling control now
use the CraftSky needle-plus SVG instead of a text plus. Shared folder/Pin action
badges use the same small, unbacked bottom-right mark as mobile.

The mobile Save post dialog now loads and listens for folders only after the
active account has confirmed Plus access. Its controller no longer starts a
folder request merely by opening the dialog. Free accounts can save without a
folder, without the paid endpoint's 403 appearing as “Folders couldn't load”.
New folder uses the existing Plus-badged folder icon and directly opens the Plus
paywall for a known free account; unavailable access retains the retry prompt.
Paid accounts retain folder creation, pagination, genuine failure presentation
and selected-folder clearing after deletion.

Validation: 12 enabled Flutter widget/controller tests passed, including free
save, direct Plus entry, paid folder creation/errors and deletion selection.
The disabled-subscriptions save test passed separately. Targeted Dart analysis
was clean and hot reload succeeded. The iOS simulator verified a free Save post
dialog without a folder error, the badge, New folder opening the native Plus
paywall and dismissal returning to the original save dialog. No purchase or save
was submitted in that manual check. Web build, type/lint and the paid-feature
browser suite cover Settings, scheduling, folders and Pin. Android UI was not
exercised.

The web scheduling icon follow-up isolates the shared Plus overlay from the
site-wide `.badge` pill style and composer-wide SVG sizing. The clock remains
visible with a transparent 12px mark at its bottom-right. Browser regressions
assert actual dimensions, no pill border/padding/margin and placement within the
button for both post and project composers, alongside paywall opening/dismissal.
All 11 enabled paid-feature browser cases passed (one disabled-flag case skipped),
web build/type checks passed, and the user's open post composer was visually
verified after hot update.
