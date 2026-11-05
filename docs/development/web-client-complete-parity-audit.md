# Web client parity audit — 3 November 2026

This continues the route and component work recorded in
[visual parity](web-client-visual-parity.md) and [feature parity](web-client-parity.md).
The references are this checkout's Flutter source, the running iPhone 18 Pro and
iPad Pro 13-inch, and [CraftSky in Figma](https://www.figma.com/design/PZzIL0qkhtkRAnvxNgSmF9/CraftSky-%C2%B7-Flutter-Design-System?node-id=69-2).

The release configuration matches the simulators: subscriptions are disabled by
default, promotions are hidden, and Growth, Customisation, scheduling and folders
show their native coming-soon gates. The enabled implementation remains available
behind the explicit release flag and is independently regression-tested.

## Coverage and evidence

Every domain below has been reviewed. “Source/fixture” means that enabled state
was checked against Flutter and controlled browser responses; it does **not** mean
that a live provider or production workflow was exercised. The earlier 49-route
layout/accessibility inventory and native widget captures remain part of the
coverage, with the following fresh simulator and Figma checks.

| Domain | Native evidence and resulting web checks |
|---|---|
| Foundations and adaptive layout | Figma color, typography, metrics, surfaces, pointer, skeleton and light/dark boards; native phone/tablet layouts. Native hover/press displacement, 256px rail, 304px drawer, 800px content bounds and discovery sidebar above 1200px. Reduced-motion shimmer regression and narrow/dark accessibility coverage. |
| Navigation and accounts | Both simulator shells, drawer, own-profile switch control and account bottom sheet. Native footer/order/selected icon pill and headerless account sheet. Escape/focus trapping, account-switch confirmation and startup recovery regressions. |
| Welcome, sign-in, registration, onboarding and session failures | Native Add account/sign-in inspected; Create account opens the provider. Existing native welcome/widget references and browser fixtures cover onboarding, restricted/expired accounts, callback and recovery states. No existing account was signed out or registered again. |
| Following and craft feeds | Both live Home layouts and preferred craft order. Complete native craft tabs, short-key URL normalization, pagination/filter/reselection fixtures, independent discovery/sidebar retry. |
| Search and hashtags | Native trending, hashtag results and result sorting; fixtures cover recent searches, result categories, filters and query restoration. Shared native compact/wide sort menus replace platform selects. |
| Post/project detail, replies and lists | Existing populated native post/thread/likes traversal plus fresh project metadata comparison. Pattern facets, designer/publisher, size, status and edited indicators corrected; reply branch/deletion/protected-content fixtures retained. |
| Post actions | Native share/overflow/save menus and own/visitor actions. Compact 56px rows in rounded sheets, native quote-image switch and description, wide anchored popups, bookmark confirmation, report/block/delete confirmations, selected-image quote CID, retry and owner safeguards. |
| Media | Live phone eight-photo quilt and tablet single-photo gallery. Fullscreen black gallery, count, dots, collapsed/expanded alt text, swiping and zoom. Feed photo opens the post first; CID carries selection into detail before opening fullscreen. Browser tests cover keyboard/drag/zoom/reset/focus, image transforms and video/caption paths. Live provider video processing remains untested. |
| Profiles and editing | Both accounts' own/visitor tabs, profile menus, crafts, links and editor/avatar choices inspected. Native menus, searchable fields, link controls and confirmation dialogs; authoritative identity refresh and unchanged-link ownership fixtures. No iPad profile edits saved. |
| Regular/project/reply/quote/edit composers | Native regular composer and every primary craft: Sewing, Knitting, Crochet, Quilting, Embroidery. Native project types/subtypes, actual tool-size catalogs, gauge/size order, optional clears, counters and limits. Knitting draft saved/reopened/deleted on the test iPhone; remaining temporary unsaved composer discarded. Reply/quote/edit/media recovery covered by source and existing browser regressions. |
| Composer secondary controls | Native language selection/add-language, sponsorship and schedule gate. Browser crop/media review, previews, draft save/discard/recovery, unknown-write handling and composition tests. Text limits preserve graphemes, UTF-8 bounds and in-progress IME text. |
| Drafts, scheduled and saved | Live empty pages and native whole-folder gate. Fixtures cover populated private drafts, save/reopen, scheduling, folders and ambiguous-write recovery with the enabled flag. |
| Activity | Populated iPad notifications, nine native categories and empty comments/replies category. Figma notification variants; filled badges, avatar geometry, unread tint/dot and protected/system rows. Category menu and notification preferences covered. |
| Messaging | Native gate and disclosures; test-account enable returned “Messaging is temporarily unavailable. Try again.” Refresh confirmed it remained disabled. Inbox, requests, conversations, menus, anchoring and stale-account handling are source/fixture coverage. No chat messages were sent. |
| Preferences | Every native settings preference page inspected read-only, including appearance, languages, notifications and privacy. Shared searchable selectors, immediate-save/rollback fixtures, native confirmation and success snackbar presentation. |
| Connections, account, standing and About | Populated iPad followers/following, empty muted/blocked pages, account options, good standing and About inspected. Growth gate verified. Relationship mutations, account deletion and failed-state recovery are fixture-only; no live deletion or relationship changes. |
| Business, products and events | Flutter/native widget references plus browser route, field, image, payload, cancellation and ownership fixtures. Shared selectors and external/remove confirmations applied. No live business declaration, product/event creation or billing changes. |
| Subscriptions and Instagram | User-selected disabled release; Growth/Customisation coming soon and promotions absent. Native Instagram verification unavailable and export help inspected. Enabled subscription/licence/import/revocation paths retain source/fixture checks; no purchase, assignment or external verification performed. |
| Public pages and service states | Public SSR metadata, association/callback bytes, accessibility, narrow/light/dark layouts, privacy, errors and offline/service-status browser regressions. Skeleton variants reviewed against Figma and Flutter. |

## Shared implementation changes

- Shared switch styling now applies across settings, project and share controls;
  settings no longer override shared field corners or icon spacing.
- `SingleSelect` supplies native searchable selectors, optional clear, required
  selection, anchored menus, keyboard navigation and dismissal across settings,
  project forms, business management and saved folders. `MultiSelect` closes on
  selection and supports Enter selection.
- Shared context menus handle compact sheets, wide popups, outside/Escape close,
  focus restoration and overlapping scroll locks. `ChoiceMenu` provides the same
  behavior to feed/search/saved/comment ordering.
- `PromptDialog` replaces generic JavaScript confirmations for internal actions
  and external links. Editor navigation protects unsaved changes, including
  browser Back; full document unloads retain browser-owned protection.
- `ImageGallery` implements the native fullscreen presentation and touch/keyboard
  interaction. Image selection between feed and detail is identified by CID.
- Native notification glyphs, project metadata, loading skeletons, field counters,
  text limits and four-second success snackbars are shared across relevant views.

## Figma coverage

Reviewed foundations `4:9`, `4:95`, `4:152`; buttons `7:431`; fields `9:513`;
surfaces `11:474`; dialogs `12:697`; navigation `11:473`; composer `51:1309`;
profile `72:155`; notifications `75:304`; skeleton guidance `66:2` and posts
`64:2`; pointer states `90:1961`; adaptive light/dark `28:842`; post action sheets
`104:425`. Inventoried foundations/components, adaptive examples, skeletons,
full showcase `69:2`, pointer states and context-menu pages.

## Validation

Final results on the completed implementation:

| Check | Result |
|---|---|
| Svelte diagnostics | 0 errors, 0 warnings |
| Production build | Passed |
| Unit tests | 276 passed across 43 files |
| Local launcher tests | 5 passed |
| Authenticated Chromium, Node preview | 166 passed |
| Authenticated Chromium, Cloudflare Worker | 166 passed |
| Public SSR Chromium, Node preview | 9 passed |
| Public SSR Chromium, Cloudflare Worker | 9 passed |
| Subscriptions-disabled release, Node preview | 1 passed |
| Subscriptions-disabled release, Cloudflare Worker | 1 passed |
| Whitespace/diff check | Passed |

The main authenticated runs enable subscriptions to retain coverage of the
complete implementation and skip the single beta-specific test. That test then
runs separately with `WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false` on each runtime.
The application itself defaults to disabled subscriptions.

Run the normal gates with `just web-client-check`, then run the disabled-release
checks from `web-client/`:

```sh
WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false npm run test:e2e:app -- authenticated-beta-release.spec.ts
WEB_CLIENT_WORKER=1 WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false npm run test:e2e:app -- authenticated-beta-release.spec.ts
```

Browser fixtures are repeatable regression evidence, not proof of live
third-party operation or pixel identity in every dynamic state.

## Live-account boundaries

The iPad was read-only throughout. Only the disposable iPhone draft was saved and
removed; the last temporary composer was discarded and the phone returned Home.
The iPad gallery was closed without changing its account. The shared live browser
account was used only for reading/navigation. Live browser DOM checks confirmed
feed-to-post photo navigation, selected CID, loaded fullsize image, fullscreen
402 × 874 bounds, count and alt text; repeated preview screenshot
requests timed out, so fixture screenshots provide the repeatable visual artifacts.

No API, lexicon, production infrastructure or deployment changes. No commit or
push was made. Provider OAuth, purchases, live messaging, Instagram verification,
video processing and destructive account lifecycle cannot be claimed as live
verified from this audit. Native-only platform chrome and browser-owned unload
prompts retain platform behavior.
