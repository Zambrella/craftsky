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

## Second follow-up: interaction and page-state corrections

The second user report is implemented in the local web client:

- Detail posts remain reading surfaces; feed cards retain hover feedback and a pointer.
- Likes/reposts use contiguous 56px rows with centered identity/avatar and row hover.
- Profile links and selected crafts lift the whole row while dragging, animate nearby
  rows, auto-scroll, and support Escape cancellation. Keyboard/menu alternatives
  remain available; reordering writes only when the profile is saved.
- Profile and link-editor fields use the ink focus border and paper lift without the
  cobalt focus halo.
- Drafts and Scheduled share the standard sticky app bar.
- Scroll-to-top matches the native circular outlined control, bold icon and elevation.
  The light fill was measured from the running iPad as `#ededed`; the dark fill is
  the Flutter theme's `#24201d`.
- Trending hashtags, profile rows, tabs, action icons, row handles and account choices
  receive cobalt hover feedback; primary buttons retain their primary hover color.
- Closing a comment composer returns through history, so the next post Back returns
  to the originating feed rather than reopening the composer.
- Toasts follow the actual main-content bounds at desktop and compact widths.
- Sponsored options anchor to their trigger on desktop, retain the compact sheet,
  and no longer have the stray scrollbar shown in the report.
- Mention suggestions include avatars; hashtag suggestions include post counts.
- Pattern, materials/style and craft detail editors open as main-content pages with
  browser history, Back/Escape, focus and scroll restoration. Unsaved material confirmation
  also works with browser Back. Shallow page changes no longer reinitialize the
  composer and discard its in-memory state. Detail-field Enter cannot submit the
  parent composer, and dismissing the account menu keeps the detail page open.

Evidence: the supplied live Likes URL now measures 56px contiguous rows, with each
41px avatar centered vertically. The populated iPad was used read-only for Scheduled
and scroll-control comparisons. Flutter theme, profile-account tile, scroll control
and composer source supplied the behavior/layout references. Local browser fixture
screenshots cover the lifted row, ink field focus, sponsored menu and project detail
page. No populated-account content was created, changed or deleted.

The full authenticated Node-preview suite passed 179 tests (the subscription-disabled
case is gated separately), including the existing route/layout/accessibility inventory.
All 277 unit tests passed; Svelte check reports zero errors and warnings.

Final verification:

- Cloudflare authenticated inventory: 178 passed in the full run. The remaining
  drag test held its pointer near the viewport edge during screenshot capture;
  auto-scroll moved the next test target offscreen. Centering the fixture rows
  removed that timing dependency. Its corrected cancellation/reorder/save test
  passed on Cloudflare, completing all 179 authenticated scenarios across the run
  and rerun.
- The final project-page keyboard, browser history, field/scroll preservation,
  desktop sizing, account-menu dismissal and accessibility cases passed in the
  same targeted Cloudflare rerun (3 tests including drag).
- Subscription-disabled Cloudflare release check: 1 passed. Promotions remain
  hidden and unavailable subscription actions stay local.
- Public SSR browser checks: 9 passed.
- Production build and `git diff --check`: passed.

These are local fixtures and read-only live UI comparisons, not production
publication, payment, OAuth-provider or destructive lifecycle verification.
No API, lexicon, backend or infrastructure changes; no commit, push or deployment.

## Third feedback pass — 3 November 2026

- [x] Home, Search, Profile and business tab strips support mouse drag scrolling;
  a completed drag suppresses navigation, while ordinary clicks and touch scrolling
  keep working.
- [x] Photo management is a content-column page with shallow browser history,
  matching the project detail editors. Back/Forward retain media and descriptions;
  leaving the editor still uses its unsaved-work confirmation. Image editing and
  export remain guarded while work is in progress.
- [x] Primary craft and its required note share one line. Text-field titles retain
  accessible labels without pointer hover/click activation. Text inputs, including
  sign-in, use the ink focus treatment without the cobalt outer outline.
- [x] Mention and hashtag suggestions follow the textarea caret. The keyboard-help
  footer is removed; loading/failure feedback remains. Facet text is painted in
  the primary colour through an inert mirror, retaining native textarea selection,
  composition, keyboard editing and scroll behavior.
- [x] Sponsored uses the shared switch with one thumb. Draft deletion uses the
  native 360px branded confirmation, white/surface fill, 14px corners, ink border,
  offset shadow and filled destructive action.
- [x] Sign-in uses Flutter's public Waow actor typeahead, with up to five deduplicated
  accounts, avatars, keyboard selection, cancellation and manual-entry fallback.
  Requests omit credentials and reject redirects. The live service's CORS
  preflight permits the web origin and `X-Client` header; a credential-free live
  query returned actors with the expected handle and avatar fields.
- [x] All three onboarding steps were compared with the running iPhone and Flutter
  source: app bar, continuous progress indicator, headings, avatar change action,
  signed-in handle, guidelines and fixed Next / Save & next / Finish action.
  One reused profile editor preserves edits across steps; unchanged profile links
  are omitted from unrelated saves. The existing diagnostics consent is retained.

### Evidence and limits for this pass

The iPhone was inspected through all three onboarding pages, without submitting
policies or altering its profile, and returned to step one. The populated iPad was
not changed. The live web onboarding screen was inspected through native browser
access after two preview automation snapshot timeouts.

Figma file access and the component-showcase inventory were confirmed. The MCP's
mandatory design-to-code guidance was unavailable (`get_figma_skill` was not
exposed, and no local Figma skill was installed), so this pass's exact styling
references are the running Flutter screens and checked-in theme/widget source.
No designs were written to Figma.

Regression screenshots are produced by `authenticated-input-parity.spec.ts` and
`authenticated-composer-detail-parity.spec.ts` for onboarding, facets, sign-in,
sponsored, photo details, and draft confirmation in both themes. Browser suites
use the local fixture AppView; they do not prove live OAuth or production writes.

Final validation for this pass:

- Svelte check: zero errors and warnings. Production build: passed.
- Unit suite: 278 tests across 43 files passed. Public browser suite: 9 passed.
- Authenticated Node preview suite: 184 passed, one compact-quote assertion
  timed out, and the subscription-disabled case was skipped by configuration.
  The unchanged compact-quote test passed in isolation and again on Cloudflare;
  its original failure snapshot already contained the expected text.
- Cloudflare with subscriptions disabled: 10 distinct focused scenarios passed
  across two runs, including the beta release configuration, input/onboarding
  parity, and photo-page selection/history. Together with the Node suite and
  unchanged rerun, all 186 authenticated scenarios passed across runs.
- The new draft-dialog test initially navigated before its asynchronous save
  completed. Waiting for the normal save-and-return navigation fixed that test
  race; both light and dark confirmations then passed on Cloudflare.
- `git diff --check`: passed. No commit, push or deployment.

## Fourth feedback pass — 3 November 2026

- [x] Account selection switches immediately, without a second confirmation.
  Existing per-account credentials, startup error recovery and language setup
  remain covered by the authenticated session tests.
- [x] Hashtag Back uses the actual navigation history, with Search only as the
  fallback for a directly opened hashtag page.
- [x] Returning to a post list restores its scroll position after asynchronous
  content arrives. Feed and thread pagination depth is retained per history
  entry and refreshed from the API before restoration. State is bounded,
  account-scoped and kept in memory; scrolling input cancels pending restoration.
- [x] The self-drafted project switch sits at the trailing end of its label row.
  Pattern text retains exactly what was typed, including a lone hash, a blank
  value or a plain name; clearing it no longer inserts another hash.
- [x] Project filters open at `/app/home/filters` in the main content column,
  with their own app bar and Back action. The feed stays mounted while filtering.
  Apply returns to the feed and updates its filters without a duplicate load.
- [x] Comments were compared directly with the open iPad and web views. The web
  now uses the native filled Comment button, primary-colour interaction totals,
  flat top-level comments without separators, muted handles, green reply controls,
  and a rounded paper-2 reply branch. Comment surfaces no longer advertise a
  redundant navigation action; their profile, facet and social controls work.

The iPad was inspected read-only and its profile/content were not changed. The
web comparison used native browser access after two preview snapshot timeouts.
No API, lexicon, infrastructure or diagnostic reporting changes in this pass.

History testing also reproduced a rapid Back/Forward race: an interrupted Back
transition could leave the app's navigation counter ahead of the browser, causing
Apply filters to retain an extra feed entry. History entries now carry their own
navigation index, and return actions wait for the active transition to settle.

Final validation for this pass:

- Svelte check: zero errors and warnings. Cloudflare production build: passed.
- Unit suite: 278 tests across 43 files passed.
- Final affected authenticated browser suites: all 54 tests passed, covering
  account isolation, filters, pattern draft restoration, comments in both themes,
  hashtag navigation and delayed, paginated feed scroll restoration.
- Cloudflare with subscriptions disabled: all six final history regression tests
  passed, including rapid filter Back/Forward, photo-page Back, profile edit
  cancellation, notification return and closing the comment composer. Earlier
  focused Worker checks also passed for the beta configuration, pattern drafts
  and comment/reply controls in both themes.
- Browser tests use the fixture AppView; they do not prove production writes or
  live OAuth. The iPad comparison was read-only.
- `git diff --check`: passed. No commit, push or deployment.
