# Web client visual and interaction review

Reviewed on 2–3 November 2026 against the Flutter source in this checkout. The
existing public AppView reads and public SSR routes are preserved. No API,
lexicon, billing provider or production infrastructure changes are part of this
visual pass. Purchases remain in the mobile apps.

## Reference method

The current Flutter app was built and launched on the iOS simulator against this
worktree's Docker environment. The test account completed provider authorization
with the account owner's assistance. The owner subsequently completed local
onboarding, including the age/policy declaration. Live Home, profile, Settings,
Appearance, navigation and disabled-messaging screens were then inspected.
The populated live Home exposed missing feed/author craft glyphs, numbered unread
badges and carousel counter/dots/description controls; these now have dedicated
browser regressions. The same eight-photo quilt was located in the web client,
where selecting the last photo updated the counter to 8/8, its description opened,
and the author crafts and narrow-layout bounds were verified. The disabled chat
gate now uses the native heading, action visibility and three disclosures without
implicitly enabling messaging or requesting broader permissions. Browser
preview screenshots timed out, so fixture screenshots remain the repeatable web
visual artifacts; live browser DOM checks do not substitute for those captures.

For repeatable, populated comparisons, actual Flutter widgets were rendered at
402 × 874 logical pixels with the bundled Outfit, DM Serif Display and Phosphor
fonts, using iOS theme/platform behavior. Eighteen capture tests produced 21
reference PNGs across settings, social, composer, drafts and chat. The welcome
screen was also captured from the running simulator. These are review artifacts,
not golden baselines; fixture content differs between some native and web shots.
The chat fixture's emoji fallback and unavailable idle Rive preload do not stand
in for live device media or animation verification.

The web audit captures healthy, populated AppView fixtures at the same logical
size. It waits for account initialization, requests and fonts, rejects API errors
and loading/error placeholders, checks horizontal overflow and navigation
visibility, and runs WCAG A/AA accessibility checks. Screenshots are attached to
each Playwright result. The fixture illustrations are explicitly labelled
CraftSky artwork; they are not simulated photographs of real projects.

## Screen coverage

| Area | Native references | Alignment and browser checks |
|---|---|---|
| Welcome and onboarding | Running iOS welcome; onboarding/profile pages | Native welcome hierarchy, craft illustrations, outlined card, registration/sign-in order, three onboarding steps and independent age/policy declaration. Onboarding hides signed-in navigation. |
| Navigation and accounts | `AppShell`, `AppShellDrawerScope`, `FormFactor`, `NavigationBar`, post composer chooser | Compact layout at 900px, icon-only bottom bar on primary branches, drawer on secondary branches, desktop rail, identity/account picker, own-profile FAB and explicit Regular/Project choice. Drawer/account focus trapping, Escape dismissal, startup failure recovery and expired-account switching are exercised. |
| Feed and posts | `FeedPage`, `PostCard`, `_ExpandablePostBody` | Outlined cards, native spacing, exact icons, compact reaction/share/save/overflow groups, protected states, loading skeletons and empty states. Body taps open the thread; long bodies expand after 300 graphemes without cutting facets. Tests retain publication, edit CID, ambiguous-write retry, reporting and stale/protected-content assertions. |
| Search and hashtags | `SearchPage`, `TagSearchPage` | Compact search, clear/cancel, recent searches, grouped trending tags, new makers, native category defaults and contextual filters. Populated search/tag routes and filter query restoration are covered. |
| Profiles and profile editing | `ProfileSliverAppBar`, profile tabs/actions/stats, edit-profile widgets | Native band/overlapping avatar, serif initials, craft chips, joined/weekly/project stats, business-first tabs and owner/visitor actions. Avatar camera picker retains image review/crop/remove. Other profiles expose Back; own DID and handle routes retain navigation. |
| Threads, projects and media | `PostPage`, project fields, image carousel/gallery | Flat comment hierarchy and contextual reply actions, craft details and rich annotations. Image selection/gallery, keyboard gallery navigation, captions, crop/rotate and video paths retain their existing guards. |
| Composer and project details | Standard/project composer and `ProjectComposerDetailPage` | Native app bar, required fields, language/sponsorship/schedule options, horizontal attachment picker and persistent Post action. Separate detail views preserve edits and return focus. Tests save/reopen project values, private image bytes, cleared optional status and schedules. |
| Drafts and scheduled posts | `DraftsPage`, scheduled-post views | Compact thumbnail/title/story/date rows and contextual edit/delete controls. Device-scoped drafts, pending-write recovery and schedule publication semantics remain covered. |
| Saved posts and folders | `SavedPostsPage` | Flat saved cards with saved date and overflow, folder overview, native ordering, folder create/rename/delete and move/remove controls. Folder views have Back without a duplicate drawer button. |
| Activity | `NotificationRow`, notification category control | Native narrative text, avatar/reaction badges, follow actions, category ordering, timestamps and protected previews. Populated rows and category controls are captured. |
| Messaging | Inbox, request, conversation and request-conversation widgets | Compact headers, peer identity, unread badges, native time sections and white/lavender bubbles, bottom-anchored history/input and request acceptance below history. Browser-visible action menus preserve report/copy/delete access. Read/sync, history anchoring, stale-account suppression and retry safeguards have focused tests. |
| Settings and preferences | Settings index, appearance, languages, notifications, customisation widgets | Flat ordered groups, identity/craft icons, subscription callout, native radio/switch rows, immediate language saves, category cards, palette/background previews and save states. All preference pages are captured, including dark desktop views. |
| Connections, account, business and subscriptions | Corresponding native settings pages and shared row/card controls | Growth/count/trend/period views, relationship lists, standing/history, account deletion, About/version/legal rows, native product/event forms, verified Instagram import modes and existing licence assignments. Browser tests preserve owner binding, payloads, CID preconditions and cancellation behavior. |

## Populated-account follow-up on 3 November

The owner switched the simulator to an account with projects, replies, reposts,
relationships and notifications. Computer use produced 57 native captures while
walking Home and craft feeds, filter/search controls, quote/share/overflow/report
sheets, a project/thread/likes list, other and owner profiles, all four owner tabs,
profile/link editing, notifications/categories, the account picker/drawer,
saved/scheduled/drafts, reachable preferences and connections, Instagram discovery,
and regular/project composers including their language, sponsorship and optional
detail screens. The temporary captures are in
`/private/tmp/craftsky-active-parity/native/`.

This follow-up aligns quote identity/content/image layout and independent author
navigation, in-card repost attribution, compact action sheets and two-step reports.
Bookmarks now confirm before their first write. Thread replies load initially,
can be hidden and reopened, and reconcile deletion. Explicitly unavailable
notification content stays suppressed; Instagram matches navigate to their actor.
Profile links use compact rows, context menus and an edit dialog while retaining
owned record references until the profile is saved.

The project filters and project/language/business multiselects now use searchable
fields with outlined removable chips above a separate search row. Filter Cancel discards unapplied choices, parent
changes prune dependent subtypes, and Clear all waits for Apply. Material controls
preserve the identity, annotations and metadata of unchanged items. Composer
language search, sponsorship, schedule selection, crop and selected-image quoting
follow the native controls. Save closes after durable persistence; closing edited
content offers Save, Discard or Keep editing. A serialized, account-bound recovery
queue prevents an earlier autosave from recreating a discarded or published draft.
Discard restores an existing draft's original timestamp and private media bytes.
Completed body URLs now generate bounded, cancellable AppView previews automatically,
with candidate navigation, dismissal/Undo and media suppression. The empty manual
preview accordion is gone. Required labels and grapheme counters follow the native
layout. Shared fields use the native quiet outline at rest and ink/shadow on focus.

The profile editor uses Close/Edit profile/Save, native counters and Links before
flat selectable craft rows. The first selected craft is marked Default; reorder,
drag and keyboard actions retain the saved craft identifiers. Successful saves
refresh shell name/avatar/customisation through an owner-bound event and an
AppView reread; stale or foreign-account events cannot replace that identity.

The same pass covers failed preference rollback, disabled messaging reads, native
relationship rows, business field limits/currencies, billing-state fallbacks and
Instagram privacy/revocation presentation. Purchases remain mobile-only.

Live messaging is disabled on this account, Instagram verification is unavailable,
and several subscription features are marked coming soon in the native build.
Their enabled states were checked through source and controlled fixtures. Simulator
scroll/drag automation returned `noWindowsAvailable`, so lower areas inaccessible
through the accessibility controls retain source/fixture evidence rather than a
claim of live traversal. Browser preview screenshots timed out twice; live browser
DOM checks verified the populated account, loaded images and narrow bounds, while
repeatable browser captures provide visual evidence.

Automatic approval review rejected opening the live Save control because it could
persist a bookmark. The safe alternative uses the real native `SavePostDialog` with
a fake repository: opening and cancelling produces no save or folder creation.
The opt-in capture is retained in
`app/test/saved_posts/widgets/save_post_dialog_test.dart`; its temporary PNG is in
`/private/tmp/craftsky-active-parity/native-fixtures/`.

## Completed validation

- Type checking: **0 errors, 0 warnings**; production and Cloudflare Worker build passed.
- Unit tests: **273 passed across 42 files**.
- Authenticated Chromium: **156 passed on local production preview and the same 156 passed on the local Cloudflare Worker**. This includes 49 route/layout/accessibility audits, compact/dark desktop checks, and 43 new populated-account regressions. Coverage includes confirmations before writes, reply branches, selected-photo animation/fullscreen consistency, profile/header/craft/link editing, identity refresh, automatic link previews and stale results, private draft save/discard/media recovery, shared searchable fields, failed preference rollback, business payloads and owner/CID/idempotency safeguards.
- Public SSR Chromium: **9 passed on local production preview and the same 9 passed on the local Worker**. Initial HTML, metadata, association/callback handling, privacy, accessibility and narrow layouts remain covered.
- Native reference capture: the earlier 18 domain capture cases produced 21 reference PNGs. This follow-up added 57 live simulator captures and a real Save-dialog fixture. Its two enabled tests passed; eight existing subscription-enabled cases were skipped by the native beta configuration. Opening/cancelling the fixture proves no bookmark or folder mutation occurs.
- `git diff --check` and Dart formatting passed. The running local AppView health endpoint and web sign-in page both returned HTTP 200.

## Reproduce the checks

Start development from the repository root with `just dev-d`, then run
`just web-run`. The frontend command starts only the web client and uses the
current worktree's environment; it does not start the backend.

From `web-client/`:

```sh
npm run check
npm test
npm run build
npm run test:e2e:app
WEB_CLIENT_WORKER=1 npm run test:e2e:app
npm run test:e2e
WEB_CLIENT_WORKER=1 npm run test:e2e
```

Run build/check before browser tests, rather than modifying `.svelte-kit` while a
production preview is using it. To additionally collect web PNGs in a stable
review directory:

```sh
CRAFTSKY_VISUAL_ARTIFACT_DIR="$TMPDIR/craftsky-parity/web" npm run test:e2e:app
```

The reusable opt-in native settings captures can be regenerated from `app/`:

```sh
fvm flutter test --dart-define=CAPTURE_NATIVE_REFERENCES=true test/settings/native_visual_reference_test.dart
```

The save-confirmation capture uses the same opt-in helper:

```sh
fvm flutter test --dart-define=CAPTURE_NATIVE_REFERENCES=true test/saved_posts/widgets/save_post_dialog_test.dart
```

They default to the system temporary directory. Set
`--dart-define=NATIVE_REFERENCE_DIRECTORY=/absolute/review/directory` to override
the destination. Ordinary Flutter test runs skip these artifact captures.

Web icons use the exact outlines from the pinned Flutter Phosphor font, with its
MIT licence retained. To regenerate after an intentional icon change, first
populate `app/.dart_tool/package_config.json`, then run from the repository root:

```sh
uv run --with fonttools python web-client/scripts/generate-native-icons.py
```

## Boundaries

This is a source-backed and rendered comparison, not a claim of pixel identity
or exhaustive live account parity. Profile headers collapse to a compact identity;
the native continuous transition retains a browser-specific rendering. Browser
selects and unsaved-change confirmations retain platform controls. OS share
sheets, photo/library permissions, text selection, gesture interactions, media
codecs and keyboard geometry retain browser/platform differences. Browser crop
controls are explicit controls; native uses touch gestures. Device drafts remain
local to each client. Native image-cache management is represented by browser
site controls, and new purchases/restore remain mobile-only by request.

The browser fixtures make no real posts, messages, follows, account deletions or
subscription changes. Live PDS media processing, Bluesky delivery, configured
push delivery, Instagram verification and store reconciliation require separate
external verification. This pass did not deploy a release or run remote CI.
