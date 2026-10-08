# AppView Changelog

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
