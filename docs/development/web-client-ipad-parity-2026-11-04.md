# iPad and web parity follow-up — 4 November 2026

This follow-up uses the open iPad Pro 13-inch (1376 × 1032 logical pixels)
and this worktree's signed-in Svelte web client. It corrects concrete differences
missed by the [previous audit](web-client-complete-parity-audit.md). Previous
passing route tests did not establish visual identity.

## Changes

- Inbox rows use the shared compact profile identity, including CraftSky craft
  icons enriched from the authenticated AppView profile read. Late profile
  responses are discarded after a peer/account change. Row padding and avatar
  sizes follow the native inbox.
- New message is a 56px pill pinned to the content column's bottom edge, with
  compact-navigation clearance, instead of following the last inbox row.
- Conversation headers retain a page heading and show the compact peer identity.
  Bubbles have the native radius and width allowance. Time headers have the
  native bottom spacing. Profile previews live inside their message bubble and
  show the avatar, handle and crafts. Their layout uses a distinct class to avoid
  inheriting the full profile page's centered/overlapping identity CSS.
- Resize observation keeps the latest message anchored while previews, fonts or
  viewport dimensions change. User scrolling away from the bottom preserves the
  older-history reading position. Pointer/keyboard message actions remain
  available without permanently drawing action dots beside every desktop bubble.
- Saved, Drafts, Scheduled and empty relationship lists use the shared centered
  empty state. Draft/Scheduled headings and descriptions match the iPad.
- Search trending craft headings include craft glyphs and native typography.
  Discovery rows use native spacing. Post cards retain 16px body typography,
  native inline Show more colour and neutral photo-frame backgrounds.
- The web composer now follows Flutter's opt-in
  `CRAFTSKY_ENABLE_VIDEO_UPLOADS` release switch. Its default is false; the normal
  picker offers photos. Existing video rendering and draft recovery remain
  available. Main browser fixtures explicitly enable video to retain coverage;
  a separate disabled-release test checks the native default.
- Secondary text retains the web's accessible colour: copying native outline
  ink exactly produced 4.16:1 contrast and failed the accessibility audit.

## Evidence boundaries

Live native inspection covered Home and craft tabs; Search and trending;
notifications; own profile and the unsaved profile editor; inbox, an existing
conversation and its action menu; Saved, Scheduled and Drafts empty states;
settings navigation, appearance, languages, notification preferences, privacy,
direct-message preferences, muted/blocked empty states, Instagram's unavailable
state, account options, About and the coming-soon dialogs; regular/project
composers, the post-language menu and project pattern details.

The browser suite covers the full route inventory, light/compact and dark/wide
layouts, profile/relationship handling, filters, post/thread/media navigation,
composer fields and private draft recovery, messaging requests and menus,
settings, business forms, account recovery and protected states. These fixture
checks are separate from live iPad inspection. Followers/following captures
initially reached loading states; those populated states remain fixture coverage
in this follow-up. Signup/onboarding, business/billing mutations, destructive
account actions and provider failures were not exercised on the live account.
No message, post, report, relationship change, profile edit or purchase was
submitted. The iPad was returned Home without saving temporary editors.

Browser snapshot images intermittently timed out. Successful snapshots, DOM
geometry, the T3 window screenshot and fixture screenshots were used alongside
native device screenshots. This is not a claim of pixel identity for every
possible account state, viewport or interaction.

## Validation

- `npm run check`: zero errors and warnings.
- `npm run build`: passed.
- `npm test`: 292 tests passed.
- Full authenticated Node browser suite: 244 passed; five conditional skips.
- Full authenticated Cloudflare Worker suite: 244 passed; five conditional skips.
- Final focused Node checks after the last preview/header/frame adjustments:
  13 passed; one conditional skip. Includes carousel and profile navigation.
- Native-default release (subscriptions and video uploads disabled): six passed,
  including the photo-only picker and late-preview scroll preservation.
- `git diff --check`: passed.

No API routes, wire contracts, lexicons or production infrastructure changed.
No deployment, commit or push was performed.


## Messaging feedback follow-up

The user identified six remaining messaging differences after the initial pass.
They are now covered by `authenticated-messaging-feedback.spec.ts`:

- Read acknowledgements no longer replace checked send authority with the
  response's default `canSend: false`. They update only unread metadata. Mute
  acknowledgements likewise update mute state, and accepting a request rereads
  checked conversation details before presenting sending controls. Controller
  tests reproduce the read failure and cover all three acknowledgement paths.
- Conversation and inbox skeletons begin immediately below the app bar and fill
  the page height. Message settings use a circular loading spinner, including
  when the initial status request is delayed.
- Requests has an app-bar Back link to the inbox.
- Conversation actions open an in-place context menu rather than an extra
  conversation-details page. Old details URLs render the conversation.
- Message actions use the shared context-menu implementation: consistent
  left-aligned full-width rows, viewport-clamped placement, compact bottom
  sheets, Escape dismissal/focus return, and scrolling for long menus.

The live existing conversation was checked without sending or selecting a
relationship action. Its header menu stayed on the conversation URL, showed all
four native actions and remained inside the viewport. Controlled browser tests
also verified the message menu's actual hit target escapes the scroll container
at phone and iPad widths, with no horizontal page overflow.

Feedback validation: Svelte check reports zero errors/warnings, production build
passes, and all 294 unit tests pass. The reported messaging states passed seven
browser cases on each runtime; the final Node shared-interaction regression run
passed 43 cases. Before the synchronous menu-opening adjustment, the broader
route/accessibility run passed 149 cases. Final phone/iPad menu screenshots were
visually inspected. No backend change, deployment or live message was required.
