# App Changelog

## 1.11.1+17 - 2026-11-01

- fix(chat): refresh unread state after reads and pushes
- fix(app): keep dialogs above the keyboard
- fix(navigation): hide drawer button outside drawer scope

## 1.11.0+16 - 2026-11-01

- feat(chat): add direct messaging with consent and account-bound sync
- feat(chat): polish direct messaging experience
- feat(profile): add shared Bluesky profile links
- feat(profile): improve profile link editing and display
- fix(profile): confirm external link opens
- feat(time): show compact dates for older posts
- fix(search): disable autocorrect and suggestions
- fix(projects): hide unavailable embroidery subtype filter
- fix(account-deletion): report failures and allow expired intent retries
- fix(feed): preserve optimistic interactions across project navigation
- fix(feed): keep rapid post interactions independent
- fix(feed): show full reply target preview
- feat(image): add basic photo editing across composer and profile flows
- test(patrol): verify photo rotation in live smoke journey
- feat(quote): preserve selected image in quote previews
- feat(onboarding): support profile links
- feat(chat): refine messaging session and conversation flows
- feat(projects): label knitting finished size as size made
- test(projects): assert renamed size field ordering
- feat(chat): improve direct messaging experience
- feat(chat): delete sent messages for self
- feat(chat): add link sharing and session-wide sync
- fix(chat): preserve inbox state when returning from conversations
- feat(select-inputs): add clear actions for optional selections
- fix(app): keep autocomplete suggestions above keyboard
- test(select-inputs): account for optional clear controls
- test(feed): include chat sharing in post menu
- feat(home): unify following and project feeds

## 1.10.0+15 - 2026-10-28

- fix(projects): restore saved status in project drafts
- feat(feed): show an alt text badge on described images
- feat(notifications): highlight new activity until leaving the page
- fix(composer): preserve alt text editing state
- fix(projects): persist self-drafted pattern toggle
- test(router): expect notification list refresh after return
- test(notifications): expect visit refreshes across settings route
- feat(composer): add keyboard dismissal action
- feat(composer): morph submit action into keyboard dismiss button
- test: find scheduled submit button inside animated wrapper
- fix(projects): clarify design labels
- fix(profile): keep craft reorder controls from toggling selection
- fix(router): keep hashtag search in authenticated shell
- feat(profile): add back-to-top scrolling
- feat(projects): add editing for selected materials
- refactor(profile): remove compact profile presentation
- feat(profile): add avatar hero navigation
- feat(profile): animate identity details from compact profile
- feat(profile): fade profile details in reading order
- fix(projects): retain independent craft tab scroll state
- fix(router): make top-level pages navigation branches
- feat(images): preload adjacent carousel images
- test(router): provide scope for image gallery navigation
- feat(posts): add published post editing
- fix(app): preserve pending material edits on navigation
- feat(notifications): add event badges and reply context
- fix(notifications): retain list state during profile navigation
- feat(post-edit): support editing published posts
- fix(projects): restore typed style selections and pattern names
- fix(observability): preserve full logs in iOS console
- test(patrol): cover post editing in live smoke journey
- fix(observability): preserve safe Sentry diagnostics

## 1.9.0+14 - 2026-10-23

- test(app): add Patrol account post flow
- test(e2e): support iOS account post flow
- fix(drafts): support the configured image limit
- feat(service-status): add versioned notice flows
- fix(media): accept images with appended data
- docs(media): clarify JPEG trailing-data validation
- fix(observability): improve Sentry triage diagnostics
- fix(instagram-import): replace Android export picker with file_picker
- fix(app): dismiss composer keyboard on post submission
- feat(projects): add duration field to project composer
- feat(service-status): gate messages on account readiness
- feat(app): let users reorder preferred crafts
- fix(service-status): navigate from router builder
- fix(feed): keep like feedback centered in visible post
- test(app): assert preferred crafts follow profile order
- feat(notifications): show text previews in notifications
- fix(onboarding): keep avatar change action available
- fix(profile): place avatar border outside the photo
- fix(languages): prefer English as primary when available
- fix(feed): refresh on Home reselection
- fix(notifications): sync follow buttons with profile state
- feat(feed): explain post types on first use
- feat(feed): limit fullscreen alt text with expansion
- refactor(profile): unify compact identity layouts
- fix(profile): constrain identity badge width
- feat(notifications): show unified actor identity
- feat(feed): open post comments from comment actions
- fix(tests): isolate flaky integration and migration setup

## 1.8.0+13 - 2026-10-20

- feat(projects): add multi-craft discovery and search
- fix(feed): preserve scroll position after image pinch
- fix(observability): preserve safe Sentry issue diagnostics
- fix(composer): restore additional craft selections
- feat(settings): add feedback link
- fix(android): match Craftsky HTTPS links without path filters
- test(android): accept all verified HTTPS link paths

## 1.7.5+12 - 2026-10-19

- fix(notifications): open reply threads at their root
- feat(feed): add quote action to thread repost menu
- fix(app): handle profile image upload failures
- chore(app): drop unsupported desktop platform targets
- fix(profile): report unexpected photo upload failures

## 1.7.4+11 - 2026-10-16

- fix(justfile): create obfuscation map directory

## 1.7.3+10 - 2026-10-16

- docs(release): separate tests from store builds
- fix(ios): preserve Rive symbols and use Flutter renderer
- chore(release): app 1.7.2+9
- test(app): prepare Rive libraries before test runs

## 1.7.2+9 - 2026-10-16

- fix(app): use Rive renderer for yarn refresh animation
- fix(app): use supported font weights
- fix(app): fall back to Flutter renderer in headless tests

## 1.7.1+8 - 2026-10-16

- build(android): apply Kotlin plugin to app

## 1.7.0+7 - 2026-10-16

- feat(discovery): add desktop sidebar recommendations
- docs: add contributor guidance and issue templates
- feat(app): add animated publishing indicator
- feat(branding): update logos across apps and web
- feat(feed): animate post likes with a stitched heart
- refactor(composer,ops): extract shared submission flows and add status C
- feat(profile): celebrate successful follows with Rive animation
- feat(app): add yarn pull-to-refresh animation
- feat(refresh): add a theme-aware curved yarn strand
- fix(observability): retain full sanitized Flutter stacks
- fix(instagram): stabilize suggestion follow acceptance
- feat(settings): add a dedicated privacy settings page
- fix(app): align summary media layout with post previews
- fix(observability): classify expected cancellations and auth failures
- feat(text-inputs): enforce record text limits
- test(profile): cover over-limit name editing behavior

## 1.6.0+6 - 2026-10-14

- fix(profile): avoid mutating cached profiles after save
- fix(profile): retain accepted fields during refresh

## 1.5.0+5 - 2026-10-14

- feat(feed): add discovery actions to empty feed
- feat: upload immediate images directly to PDS with scoped authorization
- feat(instagram-migration): add export guide
- chore(ios): update Firebase package pins
- feat(auth): request granular OAuth permissions
- fix(onboarding): retain image picker during avatar upload
- fix(profile): enforce Bluesky profile image limits
- fix(profile): preserve cached fields after profile saves
- feat(media): allow up to eight photos per post
- fix(app): show hover state on select inputs

## 1.4.0+4 - 2026-10-13

- feat: add online safety launch readiness controls
- feat(safety): allow private beta with documented safety gaps
- chore(release): app 1.3.0+3
- feat(web): add versioned Cloudflare Worker deployments
- chore(ios): declare no non-exempt encryption
- feat(service-status): add maintenance and announcement screens
- feat(search): improve relevance and typo matching
- feat(search): preserve representative hashtag spelling
- feat(profile): show mutual followers on a full page
- fix(profile): fit long display names in identity
- feat(profile): add compact new post button
- feat(app): show profile photos in mention suggestions
- feat(app): show progress on async actions
- fix(app): handle expected provider lifecycle cancellations
- fix(observability): suppress expected account cancellations
- feat(observability): add optional error reporting consent
- feat(notifications): filter notifications by category
- fix(profile): add bottom safe-area padding to tabs
- feat(app): show sort option icons with checkmarks
- fix(search): select results tab from query prefix
- feat(sharing): add copy links for posts, profiles, and events
- test(feed): cover copy link context menu item
- refactor(auth): centralize account operation guards
- feat(projects): prioritize preferred crafts
- feat(composer): add photo details route
- chore(app): upgrade Flutter and Dart dependencies
- fix(atproto): align protocol handling and PDS retries
- fix(observability): lower known cancellation log severity
- feat(projects): gate scheduling behind Plus access
- test(projects): expect scheduling subscription lock

## 1.3.0+3 - 2026-10-08

- fix(release): push tagged releases without confirmation prompt
- chore: ios config
- fix(app): inherit themed settings icon colours
- fix(app): preserve notification follow providers during requests
- fix: recover business profile saves and preserve existing data
- feat: improve logging and error reporting with simpler privacy boundaries
- Use loopback URLs for local Android AppView access
- refactor: simplify diagnostics with native Sentry integrations
- feat(auth): add login handle suggestions with avatar overlay
- fix: improve diagnostic context and retry severity
- fix: stabilize Sentry log test and clear staticcheck

## 1.1.0+2 - 2026-09-30

- Share photos, projects, and posts with other textile crafters; browse projects by craft, materials, and pattern details.
- Follow makers, join conversations, and discover posts through search, profiles, and a chronological feed.
- Save posts in folders, customize your profile, and manage notifications and account settings.
- Improved sign-in, navigation, image viewing, and post interactions, with fixes for feed refresh and comment threads.

Release history before local release automation is represented by Git tags.
