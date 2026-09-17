# Flutter Sentry Release Symbolication

Sentry upload credentials must come from the build environment. Do not commit `SENTRY_AUTH_TOKEN`, DSNs, org/project secrets, or generated symbol artifacts.

Required environment for symbol upload:

```sh
export SENTRY_AUTH_TOKEN=...
export SENTRY_ORG=...
export SENTRY_PROJECT=...
```

Create the local app release commit and `app-vX.Y.Z+N` tag before building store
artifacts. See [`../../operations/releases.md`](../../operations/releases.md)
for the explicit version, changelog, build, and push sequence.

Android release build and symbol upload:

```sh
just app-build-appbundle production
```

iOS release build and symbol upload:

```sh
just app-build-ipa production
```

The production IPA, APK, and app-bundle recipes first require clean `main` at the
matching annotated app tag and run Flutter analysis and tests. The unsigned iOS
variant (`app-build-ios`) remains a smoke build, not a releasable artifact. The
selected `app/config/<environment>.env` must contain a non-empty `SENTRY_DSN`
and enable Sentry for that environment. `SENTRY_RELEASE` and `SENTRY_DIST`, when
set there, are also used for the upload.

Internally, the recipes build with `--obfuscate`, `--split-debug-info`, and an
obfuscation map, then run `dart run sentry_dart_plugin` against those exact
artifacts. Web source-map upload is disabled for these mobile-only builds. After
upload, releasable recipes print the artifact SHA-256 and matching symbol paths;
retain those values with the release record. The releasable artifacts are
`app/build/ios/ipa/*.ipa` and
`app/build/app/outputs/bundle/release/app-release.aab`; matching debug data is
under `app/build/debug-info/<target>` and
`app/build/app/obfuscation-<target>.map.json`.

Web release smoke build:

```sh
cd app
flutter build web --release --source-maps
dart run sentry_dart_plugin
```

Manual check before production release: trigger a controlled reportable error in staging and confirm the Sentry event has the expected environment, release/dist, readable Dart frames, and no forbidden sensitive fields.
