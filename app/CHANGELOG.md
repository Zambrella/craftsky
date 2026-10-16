# App Changelog

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
