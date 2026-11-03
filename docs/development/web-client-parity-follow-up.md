# Web parity follow-up — 3 November 2026

The earlier audit established route coverage, not complete visual or interaction
parity. This follow-up tracks the user's concrete discrepancies against live
Flutter behaviour and browser reproduction. No production deployment is implied.

## User acceptance checklist

- [x] Remove app-bar refresh icons.
- [x] Suppress outer post hover while interacting with embedded content.
- [x] Confirm external links consistently.
- [x] Unify social-action hover states.
- [x] Centre images where carousel ratios differ.
- [x] Swipe inline carousels and provide arrow controls.
- [x] Show fullscreen image alt text at the bottom.
- [x] Match project feed sort/filter styling and app-bar placement; compact filter FAB.
- [x] Make the project post surface clickable.
- [x] Match project detail styling.
- [x] Use cobalt-based light-mode action hovers.
- [x] Repair project filter input selection.
- [x] Match selected/focused search fields.
- [x] Remove blue avatar outlines from skeletons.
- [x] Return from notification-linked posts to Activity.
- [x] Match account/profile switcher.
- [x] Use link images in profile links.
- [x] Add scroll-to-top controls to long lists.
- [x] Repair reorderable profile links and crafts.
- [x] Remove cover-photo concepts.
- [x] Repair Saved error.
- [x] Remove Scheduled tab bar.
- [x] Remove Drafts tab bar.
- [x] Match follower/following row spacing.
- [x] Use native new-post choice dialog.
- [x] Move photo addition/review to its own native-style page.
- [x] Match selected/focused project composer fields.

## Additional findings from the renewed native comparison

Implemented and regression-checked:

- Settings shares the shell account sheet; disabled release no longer displays Plus badges.
- Settings identity uses the shared compact identity; native app-bar/title and row insets.
- Appearance radio layout/system icon; no extra success snackbar.
- Shared selects retain non-searchable selected values while open, flip disclosure,
  support language-code descriptions/search, use native helper gaps and focus lift.
- Language and notification card spacing; outline-variant divider colour.
- Privacy permits storing local consent independently of SDK configuration, like Flutter.
  Existing configured-and-consented transmission gate remains unchanged.
- Account removes the extra business link and uses the native first deletion confirmation.
  Native confirmation was opened and cancelled; deletion was not continued.
- Native bold icon variants and dialog typography/spacing.

Platform distinction: About's native image-cache clearing has no equivalent in
this web client, whose service worker has no fetch cache. Browser HTTP cache is
managed by browser/site controls. Do not add a misleading success-only action.

## Verification evidence

The checked items above mean implemented or verified in the stated paths. They do
not certify pixel identity in every possible account, provider or loading state.

| Area | Evidence from this follow-up |
|---|---|
| Feed and posts | Live populated iPad project detail compared with the same web post; Flutter `project_card.dart` metadata order, chip typography and title treatment. Figma project card `71:118` and shared design references. Browser regressions cover whole-card navigation, nested hover, external confirmation, social menus, carousels and fullscreen alt text. |
| Filters and selection | Label-click regression reproduces the project-filter focus race, then verifies selection and URL/query persistence. All craft detail suites pass, including optional clear, selected values, keyboard dismissal and reopening drafts. |
| Navigation | Notification post Back returns to Activity. Browser Back closes Photo details and retains descriptions. Home reselect refresh retains protected-content and late-mutation safeguards after removing app-bar refresh buttons. |
| Profiles and lists | Live browser profile link images loaded; no cover-photo control or display. Live Following rows measure 56px, use cobalt avatar borders, and scroll-to-top returns to zero. Native iPad Following and account popup inspected; browser tests cover desktop popup and compact account sheet. Existing profile save/reorder/cancel regressions pass. |
| Composer and media | Native iPhone photo picker/details inspected without publishing. Browser selection opens its own page; descriptions survive reorder, Back, reopen and removal. Crop handles support pointer dragging after rotation, keyboard input, export and cancel. Photo details screenshot and crop editor screenshot captured by the focused tests. |
| Settings and release gates | Native settings, appearance, languages, privacy, account confirmation and about inspected. Confirmation cancelled. Subscription-disabled release checks cover hidden promotions, Growth/Customisation and unavailable scheduling/folders. |

### Local checks

- Svelte check: 0 errors, 0 warnings.
- Production build: passed.
- Unit tests: 277 passed across 43 files.
- Launcher tests: 5 passed.
- Authenticated Chromium on Node preview: 174 passed across the full run and the
  corrected scroll-readiness test rerun. The beta-only test is separately gated.
- Authenticated Chromium on Cloudflare Worker: 174 passed in one full run.
- Public SSR Chromium: 9 passed on Node preview and 9 on Cloudflare Worker.
- Subscriptions-disabled release and photo workflow/accessibility: 2 passed on
  Node preview and 2 on Cloudflare Worker.
- `git diff --check`: passed.

The authenticated suite includes the 49-route layout/accessibility inventory,
narrow and dark layouts, session/account isolation, pending-write recovery,
composer persistence, moderation and messaging regressions. Fixture success is
not proof of live third-party OAuth, messaging, uploads, Instagram verification,
purchases or destructive account lifecycle.

### Deliberate platform/accessibility differences

Muted text retains the slightly darker accessible web shade `#736a64` instead of
native `#7a716b`, which fails 4.5:1 contrast on paper. Small destructive/status
text uses accessible shades for the same reason. Browser file selection and full
document unload prompts remain browser-owned. About's native image-cache action
has no equivalent in the current web service worker and remains absent.

### Account and delivery boundaries

The iPad and populated live web profile remained read-only. The iPhone photo was
selected and removed in an unsaved composer, then the composer was closed and the
phone returned Home. No new live posts, profile edits or messages were sent.
No API schema, lexicon, backend or production infrastructure changes; nothing
committed, pushed or deployed. The existing local app remains available at
`http://127.0.0.1:25319/app/home`.
