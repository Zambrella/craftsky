# Web Client Changelog

## 1.15.0 - 2026-11-25

- feat(projects): expand craft-specific project details and filters
- test(web-client): update project detail expectations

## 1.14.0 - 2026-11-22

- feat(web): add a generated project schema explorer
- feat(video): add custom thumbnail selection and upload
- fix(chat): preserve failed message recovery across clients
- test(web-client): update share link guidance assertion
- fix(web-client): preserve feed tab scroll position
- fix(profile-links): resolve streamed icons and generate valid rkeys
- feat(profile): default to posts tab
- fix(notifications): harden cross-client notification handling
- perf(chat): preload conversations and bound sync refreshes
- fix(observability): preserve safe failure details across clients
- fix(notifications): fence stale opens and push deliveries
- test(web): wait for relationship loading hydration

## 1.13.2 - 2026-11-18

- fix(web-client): reject stale notification badge polls

## 1.13.1 - 2026-11-18

- test(web-client): unmount profile page before resizing

## 1.13.0 - 2026-11-17

- fix(release): verify guest landing redirects
- feat(social): link plain text profile bios
- feat(profile): add visitor follower and following lists
- fix(instagram-migration): show export help after verification
- fix(web-client): align connection pages with settings layout
- fix(posts): preserve selected carousel image in details
- fix(media): preserve image framing across clients
- feat(catalog): add navy, turquoise, and teal colors
- fix(ci): upgrade Go to 1.27.4 for release gate
- fix(composer): trim submitted text before generating facets
- feat(web-client): add profile share cards
- feat(web-client): add interactive foil profile share cards
- feat(composer): limit posts to ten hashtags
- fix(feed): preserve scroll position during refresh
- fix: preserve loaded content during background refreshes
- test(web): scope connection pagination assertions to main
- feat(subscriptions): add custom Plus tier icon
- fix(web): preserve composer return destination while loading
- test(web): wait for post before holding editor request
- fix(subscriptions): clarify tier benefit summaries
- feat(subscriptions): refine active billing and assignment UI
- feat(settings): show account hosting provider details
- feat(profile): add colour options and shader backgrounds
- fix(settings): repair provider accessibility and fixture inventory
- feat(profile): add reference-inspired shader backgrounds
- chore(repo): optimize checks and client performance
- chore(app): harden Riverpod lifecycles and enable linting
- feat(subscriptions): support independent billing accounts
- fix(observability): report consumed failures across app flows
- feat(docs): add searchable change archive
- fix(subscriptions): guard purchases by store ownership
- feat(web-client): add automatic feed pagination
- feat(subscriptions): add complimentary access grants
- feat(subscriptions): add embedded checkout and web paywalls
- feat(subscriptions): open paywalls from locked features
- feat(subscriptions): expand cross-client subscription and business parit
- feat(business): add product photo carousels
- feat(profile): show business identity icons across clients
- fix(subscriptions): harden billing lifecycle and access handling
- feat(profile): retire textures and decorate recent searches
- style(web-client): format image upload test call
- test(ci): stabilize business and subscription regression checks
- fix(app): improve form and composer keyboard actions

## 1.12.0 - 2026-11-10

- fix(web-client): provide safe social share images
- fix(web-client): recover autocomplete and handle missing crafts
- feat(video): add Bunny uploads and hosted web playback
- fix(web-client): handle errors consistently across features
- fix(web-client): harden recovery and restore release checks
- fix(web-client): keep onboarding next action clickable
- test(web-client): wait for draft navigation before opening tab
- feat(stats): add public CraftSky statistics page
- fix(video): show previews in quotes and saved posts
- feat(composer): enable local video previews
- fix(video): extend upload finalization deadlines
- feat(video): cancel Bunny uploads safely
- feat(guest-browsing): enable public content exploration
- feat(feed): add extended network timeline
- test(web): wait for refreshed network feed after navigation
- feat(browsing): expand logged-out content exploration
- feat(moderation): allow guests to submit content reports
- style(web-client): unify profile people rows
- fix(web-client): add filter trigger hover and active states
- fix(web-client): keep report radio options in a row
- fix(web-client): separate notification profile and row links
- fix(web-client): always offer onboarding error reporting consent
- chore(app): remove Flutter web support
- fix(search): align recent and hashtag result rows
- feat(instagram-import): style help links as buttons
- feat(web-client): add consent-gated YouTube embeds
- fix(profile): overlay stats for accepted publications
- test(web): update shared hashtag row assertions
- feat(profile): add likes links to comment menus
- fix(links): route canonical CraftSky links internally
- fix(web-client): clear focus when opening auth dialog
- feat(profile): show avatar in navigation
- fix(web-client): confirm project links and underline on hover
- feat(video): enforce PDS blob limits for Bunny uploads
- feat(auth): show Atmosphere logos beside handle fields
- feat(auth): display full-color Atmosphere icons
- fix(web-client): highlight reposted posts
- feat(web-client): show branded startup loading screen
- test(web-client): await filter route navigation
- test(web): update startup loading selectors
- feat(safety): add go-live safety and moderation controls
- fix(web-client): repair launch readiness CI checks
- chore(config): centralize AppView secret management
- feat(config): separate local credentials from app settings
- refactor(config): move local env files into component config dirs
- feat(config): isolate client settings by build mode
- chore(docs): reorganize project documentation
- fix(app): stabilize chat and feed journeys
- fix(social): preserve replies and improve feed media labels
- test(web): assert sponsored post tooltip copy
- test(web-client): assert carousel image URL
- style(web-client): remove craft icons from sign-in screen
- fix(web-client): keep report dialog interactions on the feed
- fix(web-client): preserve filter button hover styles
- fix(web-client): align logged-out visibility with privacy settings

## 1.11.4 - 2026-11-05

- fix(web-client): smooth feed navigation and profile actions (#311)

## 1.11.3 - 2026-11-05

- fix(web-client): preserve timeline identities across repost overlays (#310)

## 1.11.2 - 2026-11-05

- Introduce the Svelte web client with public profiles, post and project pages, and share metadata.
- Add browser sign-in and authenticated feeds, discovery, messaging, notifications, profile editing and account settings.
- Support publishing posts and projects, media uploads, drafts, scheduling and recoverable edits.
- Include responsive layouts, keyboard access, reduced-motion support and browser error-reporting consent.
- Add tagged web releases with checked Cloudflare previews and promotion of the reviewed Worker version.

Release history before local release automation is represented by Git tags.
