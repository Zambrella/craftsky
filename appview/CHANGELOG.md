# AppView Changelog

## 1.1.0 - 2026-11-01

- feat(chat): add direct messaging with consent and account-bound sync
- feat(chat): polish direct messaging experience
- feat(profile): add shared Bluesky profile links
- chore: Disabled Sentry Metrics in dev
- fix(appview): include projects in recent post count
- feat(quote): preserve selected image in quote previews
- feat(appview): auto-approve images in development
- feat(chat): refine messaging session and conversation flows
- feat(chat): improve direct messaging experience
- fix(search): filter projects by primary craft
- feat(chat): delete sent messages for self
- fix(auth): verify optional chat OAuth scopes

## 1.0.20 - 2026-10-28

- feat(notifications): highlight new activity until leaving the page
- fix(appview): show all languages in profile feeds
- feat(posts): add published post editing
- fix(render): enable link previews in production
- feat(post-edit): support editing published posts
- fix(observability): preserve safe Sentry diagnostics

## 1.0.19 - 2026-10-23

- test(app): add Patrol account post flow
- fix(auth): activate onboarding before issuing handoff
- test(e2e): support iOS account post flow
- feat(service-status): add versioned notice flows
- fix(observability): improve Sentry triage diagnostics
- fix(app): dismiss composer keyboard on post submission
- feat(notifications): show text previews in notifications
- test(observability): expand push privacy fixture schema
- refactor(profile): unify compact identity layouts
- fix(tests): isolate flaky integration and migration setup

## 1.0.18 - 2026-10-20

- feat(projects): add multi-craft discovery and search
- feat(posts): preserve reserved craft metadata
- fix(observability): preserve safe Sentry issue diagnostics
- fix(composer): restore additional craft selections
- fix(auth): hand off Android OAuth completion to the app

## 1.0.17 - 2026-10-19

- test(release): skip Flutter tests in app preflight
- test(app): prepare Rive libraries before test runs
- fix(justfile): create obfuscation map directory
- fix(notifications): open reply threads at their root
- chore(app): drop unsupported desktop platform targets
- chore(render): enable moderation admin configuration
- fix(operatorconsole): add staged OAuth failure diagnostics

## 1.0.16 - 2026-10-16

- feat(discovery): add desktop sidebar recommendations
- docs: add contributor guidance and issue templates
- feat(branding): update logos across apps and web
- fix: honor lexicon image limit during post ingestion
- feat(moderation): add operator console and sessions
- refactor(composer,ops): extract shared submission flows and add status C
- fix(ops): count due notification retries in status
- fix(instagram): stabilize suggestion follow acceptance
- feat(moderation-console): expand operator workspaces
- fix(observability): classify expected cancellations and auth failures
- feat(moderation): add version-aware contextual previews
- fix(observability): preserve provider failure diagnostics
- feat(text-inputs): enforce record text limits

## 1.0.15 - 2026-10-15

- feat: upload immediate images directly to PDS with scoped authorization
- feat(auth): request granular OAuth permissions
- fix(profile): enforce Bluesky profile image limits
- feat(media): allow up to eight photos per post

## 1.0.14 - 2026-10-13

- feat: add online safety launch readiness controls
- feat(safety): allow private beta with documented safety gaps
- feat(web): add versioned Cloudflare Worker deployments
- feat(service-status): add maintenance and announcement screens
- build(appview): update Go toolchain and check tools
- feat(search): improve relevance and typo matching
- test(api): load search migrations lazily
- feat(search): preserve representative hashtag spelling
- test(scheduledposts): stabilize concurrent reupload cleanup test
- fix(observability): suppress expected account cancellations
- feat(notifications): filter notifications by category
- feat(ingestion): project post notifications independently
- feat(projects): prioritize preferred crafts
- test(appview): verify RevenueCat provider failure logging
- fix(atproto): align protocol handling and PDS retries

## 1.0.13 - 2026-10-08

- chore(release): AppView 1.0.11
- fix: preserve onboarding session authority and add revocation CLI
- feat(appview): expose cached Tap telemetry in healthz
- fix(appview): embed timezone data for business events
- fix: avoid demo seed identity handle collisions
- fix: recover business profile saves and preserve existing data
- test: allow database setup in registration deadline regression
- feat: improve logging and error reporting with simpler privacy boundaries
- Use loopback URLs for local Android AppView access
- refactor: simplify diagnostics with native Sentry integrations
- fix: improve diagnostic context and retry severity
- fix: stabilize Sentry log test and clear staticcheck

## 1.0.12 - 2026-10-02

- fix(appview): upgrade OpenTelemetry to address GO-2026-6505
- chore(release): AppView 1.0.11

## 1.0.11 - 2026-10-02

- chore(render): expose Tap diagnostics on loopback
- fix(appview): reconcile missing profiles during OAuth sign-in

## 1.0.10 - 2026-10-01

- fix(appview): revalidate retained sources during repository repair

## 1.0.9 - 2026-10-01

- fix(app): repair iOS release preflight signing check
- fix(release): push tagged releases without confirmation prompt
- fix(appview): journal onboarding profile writes

## 1.0.8 - 2026-09-30

- fix(appview): run and verify production migrations before deploy

## 1.0.7 - 2026-09-30

- feat(appview): add account subscriptions
- feat: unify PDS record mutations
- feat: add profile reposts and harden PDS mutation flow
- fix: harden PDS mutation retries and source projection
- refactor: retire legacy set projection tables
- chore(dev): skip stale Tap cursor replay
- feat(app): add RevenueCat subscriptions and billing UI
- feat: gate subscription features for beta
- fix(appview): mark saved profiles as CraftSky profiles
- fix(app): unify profile account list rows
- feat(dev): run Flutter on physical Android devices
- fix(ci): restore test gates and object store pulls

## 1.0.6 - 2026-09-24

- fix: enable Render SSH shell

## 1.0.5 - 2026-09-23

- fix: reduce health observability bandwidth

## 1.0.4 - 2026-09-17

- refactor: simplify oauth completion styling
- chore: harden mobile release workflow
- test: consolidate Flutter test suite
- chore: add local release toolkit
- test: harden and consolidate Go suite
- fix: make unit checks portable in CI
- test: pin release fixture branch

Release history before local release automation is represented by Git tags.
