# Flutter Sentry Release Symbolication

Sentry upload credentials must come from the build environment. Do not commit `SENTRY_AUTH_TOKEN`, DSNs, org/project secrets, or generated symbol artifacts.

Required environment for symbol upload:

```sh
export SENTRY_AUTH_TOKEN=...
export SENTRY_ORG=...
export SENTRY_PROJECT=...
```

Android release smoke build:

```sh
cd app
flutter build apk --release --obfuscate --split-debug-info=build/debug-info \
  --extra-gen-snapshot-options=--save-obfuscation-map=build/app/obfuscation.map.json
dart run sentry_dart_plugin
```

iOS release smoke build:

```sh
cd app
flutter build ipa --release --obfuscate --split-debug-info=build/debug-info \
  --extra-gen-snapshot-options=--save-obfuscation-map=build/app/obfuscation.map.json
dart run sentry_dart_plugin
```

Web release smoke build:

```sh
cd app
flutter build web --release --source-maps
dart run sentry_dart_plugin
```

Manual check before production release: trigger a controlled reportable error in staging and confirm the Sentry event has the expected environment, release/dist, readable Dart frames, and no forbidden sensitive fields.
