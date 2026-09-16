# Flutter Sentry Release Symbolication

Sentry upload credentials must come from the build environment. Do not commit `SENTRY_AUTH_TOKEN`, DSNs, org/project secrets, or generated symbol artifacts.

Required environment for symbol upload:

```sh
export SENTRY_AUTH_TOKEN=...
export SENTRY_ORG=...
export SENTRY_PROJECT=...
```

Android release build and symbol upload:

```sh
just app-build-appbundle production
```

iOS release build and symbol upload:

```sh
just app-build-ipa production
```

The APK and unsigned iOS smoke-build variants (`app-build-apk` and
`app-build-ios`) use the same fail-closed Sentry checks and upload flow. The
selected `app/config/<environment>.env` must contain a non-empty `SENTRY_DSN`
and enable Sentry for that environment. `SENTRY_RELEASE` and `SENTRY_DIST`, when
set there, are also used for the upload.

Internally, the recipes build with `--obfuscate`, `--split-debug-info`, and an
obfuscation map, then run `dart run sentry_dart_plugin` against those exact
artifacts. Web source-map upload is disabled for these mobile-only builds.

Web release smoke build:

```sh
cd app
flutter build web --release --source-maps
dart run sentry_dart_plugin
```

Manual check before production release: trigger a controlled reportable error in staging and confirm the Sentry event has the expected environment, release/dist, readable Dart frames, and no forbidden sensitive fields.
