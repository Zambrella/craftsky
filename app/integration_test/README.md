# Flutter integration tests

This directory owns device/process coverage. The fast `app-test` recipe still
runs only `test/`; use `just app-test-integration <device-id>` locally and
`just app-test-integration-ci <device-id>` in CI.

`critical_journeys_test.dart` runs the production app, router, providers,
repositories, API clients, socket transport, and mobile plugin lifecycle. It
uses real package-info, device-info, shared-preferences, and wakelock plugins.
External state is deterministic: retained sessions are in memory, parsed push
opens are injected at the `NotificationRuntime` boundary, and AppView responses
come from a loopback HTTP server. No production endpoint or credential is read.

Covered journeys:

- Authenticated timeline read over HTTP, followed by a UI-driven retained
  account switch and an account-fenced reload with the second bearer token.
- Regular post composition and publication through the production
  `PostApiClient`/repository, including the mobile wakelock lifecycle and live
  timeline update.
- Notification provider payload parsing, recipient resolution, runtime effect,
  and production deep-link routing to a profile.
- Mobile startup dependency resolution through package-info, device-info, and
  shared-preferences plugins.

Not responsibly automated yet:

- Initial OAuth login/add-account requires a disposable PDS identity and a
  browser redirect service. The suite covers retained-account activation, not
  external OAuth ownership.
- APNs/FCM delivery and OS notification taps require signed platform setup and
  provider credentials. The suite starts at the `NotificationRuntime` boundary.
- Camera, photo-library, file-picker, and video-upload journeys require OS
  permission orchestration plus stable media fixtures. Faking those platform
  channels would reduce them to widget tests, so they remain out of scope.
- A real AppView/PDS write is not used because it would require shared mutable
  infrastructure and credentials. The loopback server verifies the running
  app's complete UI-to-socket request path and production client contract.

The suite currently targets Android and iOS. The app intentionally rejects
desktop dependency initialization, and the loopback server uses `dart:io`, so
macOS and web are not valid targets.

No `test_driver` entry point or `enableFlutterDriverExtension` is needed for
the current `flutter test integration_test/... -d <device>` runner.
