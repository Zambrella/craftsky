# Flutter 3.47 upgrade

The app and CI use Flutter 3.47.7 with Dart 3.13.5. The FVM pin is in
the repository-root `.fvmrc`; the application language version is Dart 3.13.

Sources reviewed:

- [What's new in Flutter 3.47](https://flutter.dev/blog/whats-new-in-flutter-3-47)
- [Announcing Dart 3.13](https://dart.dev/blog/announcing-dart-3-13)
- [Standalone design package migration](https://docs.flutter.dev/release/breaking-changes/material-ui-and-cupertino-ui)
- [Accessibility heading migration](https://docs.flutter.dev/release/breaking-changes/semantics-header-heading-level)
- [Built-in Kotlin migration](https://docs.flutter.dev/release/breaking-changes/migrate-to-built-in-kotlin)
- [Sentry 10 migration](https://docs.sentry.io/platforms/dart/guides/flutter/migration/v9-to-v10/)

## Applied changes

- Migrated application, test, and preview imports to `material_ui` and
  `cupertino_ui`. The upgraded theme, font, form, image, and routing packages
  expose those packages' types.
- Added `app/lib/l10n/app_localizations.dart` as the application localization
  entry point. It exports generated translations and provides the standalone
  widget delegates. Flutter 3.47's localization generator still emits legacy
  delegates, so application widgets use `appLocalizationsDelegates` instead of
  the generated `AppLocalizations.localizationsDelegates` list.
- Wrapped the application's three MaterialApp roots with the official
  compatibility bridge for remaining SDK-based media and billing widgets.
  The bridge is deliberately temporary and is already marked deprecated by
  the package; remove it once the remaining plugins migrate their design-state
  reads. It cannot convert types in plugin API signatures.
- Replaced the three accessibility `header: true` uses with heading levels:
  maintenance page title at level 1, account-standing summary and saved-post
  sections at level 2.
- Adopted the Dart 3.13 concise in-body constructor syntax and the latest
  Very Good Analysis rules. Added `await` to returned futures, including those
  whose failures need to remain inside surrounding `try`/`catch` blocks.
  Reformatted with the new formatter and regenerated Riverpod, route, and
  model code. Existing trailing-comma preservation remains configured.
- Updated every direct Dart dependency to its latest stable resolvable version
  and refreshed the transitive lockfile. Sentry logging remains pinned to
  10.0.0 to match the Sentry Flutter SDK. The exact media-kit and visibility
  detector pins already match their latest stable releases.
- Adapted Sentry 10's always-enabled logs and new callback signatures. Log
  forwarding can still be disabled independently through the logging
  integration threshold. Metrics are dropped, and transaction tracing keeps
  its existing filtering path. Production tracing, replay, request capture,
  PII collection, and opt-in consent retain their existing configuration.
- Added Dio's transformation timeout and Firebase's permanent permission
  denial to the existing application error/permission mappings.
- Rebuild filtered request headers before attaching the device ID and bearer.
  The old remove-and-reinsert path reproduces a custom-hash-map type error
  under Dart 3.13.5. The existing credential-boundary tests verify that pending
  handoff bearers survive and PDS credentials remain excluded.
- Keep block/unblock and delete mutation providers alive only while their
  command is pending. Widget regression tests reproduced disposal before
  completion when no widget subscribed to those providers. Account-authority
  checks still fence obsolete completions, and automatic disposal resumes
  after the command finishes.

## Native targets

| Setting | Version |
| --- | --- |
| Android minimum API | 26 (required by Sentry 10) |
| Android compile/target API | Flutter SDK defaults (36) |
| Java source/target | 17 |
| Android Gradle Plugin | 9.1.0 |
| Gradle | 9.3.1, binary distribution |
| Kotlin toolchain/classpath | 2.4.0 |
| Google Services plugin | 4.5.0 |
| iOS minimum | 15.0 (already configured) |
| macOS minimum | 12.0 |

Swift Package Manager is explicitly enabled in `pubspec.yaml`. Both Apple
projects already had SwiftPM integration; iOS already used SceneDelegate.
The Apple dependency lockfiles were refreshed by the native builds.

Android uses AGP's built-in Kotlin with `android.builtInKotlin=true`. The app
does not apply the standalone Kotlin Android plugin. Its version remains on
the plugin classpath to select the supported Kotlin toolchain. The latest
app-settings, Firebase Core, and RevenueCat UI plugins guard their legacy
plugin application when built-in Kotlin is active. Flutter's static scan still
reports their legacy branches as KGP usage.

`android.newDsl=false` remains the Flutter-supported compatibility setting.
Current media-kit Apple plugins still need Flutter's CocoaPods fallback.
Recheck upstream support before removing that setting or the Podfiles.

## Features requiring no application migration

Desktop Impeller is an SDK default; the app has no renderer opt-out. Its web
transport code already uses `dart:js_interop` and `package:web`. WebAssembly
compilation is validated, while the normal release command stays unchanged.
Experimental deferred loading and multi-window APIs are not enabled. The app
has no custom native code-asset hooks requiring `@RecordUse`, and it is a
single Dart package, so the new Pub workspace commands require no changes.

## Dependency limits

`flutter pub outdated --json` reports no outdated direct or development
dependencies. Eight transitive packages have newer versions that the current
SDK or upstream packages do not resolve:

| Package | Resolved | Newer release |
| --- | --- | --- |
| cross_file | 0.3.5+5 | 0.4.0 |
| equatable | 2.1.0 | 3.0.0 |
| injector | 3.0.0 | 4.0.0 |
| jni | 1.0.3 | 1.1.0 |
| jni_flutter | 1.0.3 | 1.0.4+1 |
| material_color_utilities | 0.13.0 | 0.13.1 |
| test_api | 0.7.12 | 0.7.15 |
| xml | 7.0.1 | 7.1.0 |

No dependency overrides are used.

## Validation

Validated locally on October 12, 2026:

- `flutter analyze --no-pub`: no issues.
- `flutter test --no-pub --reporter expanded`: 2,908 passed, 38 skipped.
- `dart run build_runner build`: completed successfully.
- `dart format --output=none --set-exit-if-changed lib test integration_test`:
  1,412 files checked, no changes required.
- `flutter build web --release --wasm --no-pub`: WebAssembly and JavaScript
  compilation passed.
- `flutter build ios --simulator --no-codesign --no-pub`: passed on Xcode 27.
- `flutter build macos --release --no-pub`: passed on Xcode 27.
- `flutter build apk --debug --no-pub`: passed with built-in Kotlin enabled.
- `actionlint .github/workflows/backend-ci.yml`, Apple project `plutil -lint`,
  mirrored SwiftPM lockfile comparisons, and `git diff --check`: passed.
- Local web preview: welcome page rendered and the Atmosphere explanation
  expanded. The preview recorded optional font socket failures and service-status
  DNS failures, so this is a limited UI smoke check.

Native builds are compilation checks; store signing, device purchase/video/push
flows, and production deployment are separate release checks.

### Integration with the AT Protocol audit fixes

The upgrade branch at `cc5b94d8` was merged into the audit branch at `6cf80fe1`.
The two conflicts were the mutation exception constructor and scheduler typedef.
Both use the new Dart syntax while retaining the audit's rate-limit classification,
server cooldown, and frozen operation-key behavior. The upgrade's compound-write
fix also remains: unchanged records are omitted from `applyWrites` so Tap does
not reject commits with identical old and new CIDs.

Post-merge validation on Flutter 3.47.7 / Dart 3.13.5:

- Dependency resolution passed with `--enforce-lockfile` and no lockfile changes.
- Full Flutter suite: 2,912 passed, 38 skipped; analysis reported no issues.
- Formatting checked 1,412 files. Code generation reproduced the checked-in code
  apart from extra trailing blank lines in generated model files, which were
  removed to avoid unrelated formatting changes.
- Release web build passed for JavaScript and WebAssembly.
- The complete `scripts/appview-check` gate passed, including generated-code
  checks, Go analysis, full PostgreSQL-backed tests and race tests, source and
  binary vulnerability scans, Docker builds, source-free media probes,
  migration rollback/reapply and failure handling, and the pinned Tap handshake.
- Workflow lint, Apple project syntax, mirrored SwiftPM lockfiles, and merge
  whitespace checks passed. Native builds above were performed on the upgrade
  branch before integration; they were not repeated for this merge.
