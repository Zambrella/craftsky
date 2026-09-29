# Flutter App Config

Flutter app runtime configuration is passed at build time with
`--dart-define-from-file`. The Dart code reads these values with
`String.fromEnvironment` and `bool.fromEnvironment`.

Local `.env` files are ignored by git. Start with:

```bash
just app-env-init
```

That creates:

- `app/config/local.env` for Chrome, macOS, and iOS simulator.
- `app/config/local-android.env` for the Android emulator.

Build config files must be created explicitly from their examples:

```bash
cp app/config/staging.env.example app/config/staging.env
cp app/config/production.env.example app/config/production.env
```

`CRAFTSKY_ENABLE_VIDEO_UPLOADS` controls whether post composers offer video
selection. It defaults to `false` when omitted; set it to `true` only when the
configured video infrastructure is dependable.

`SENTRY_DSN` is public client configuration once the app is shipped, but keep it
out of committed examples. `SENTRY_AUTH_TOKEN`, `SENTRY_ORG`, and
`SENTRY_PROJECT` are build/upload credentials for Sentry symbolication and must
come from CI secrets or your shell environment, not from these app config files.

`REVENUECAT_IOS_PUBLIC_KEY` and `REVENUECAT_ANDROID_PUBLIC_KEY` are public SDK
keys, not RevenueCat secret API keys. Leave them blank to disable billing safely
for that platform. Never place a RevenueCat secret server key in app config.

`SUBSCRIPTIONS_ENABLED` defaults to `false`. Beta builds hide the subscription
callout and route, show a muted “Coming soon” prompt for Plus actions, and do
not configure RevenueCat even if public keys are supplied. When billing is
ready to launch, set it to `true` in the build config and verify purchases and
paid-feature access on both native platforms. AppView remains the authority for
paid-feature permissions; this build flag controls the Flutter experience.

Local native debug builds may opt into RevenueCat Test Store by setting
`REVENUECAT_TEST_STORE_PUBLIC_KEY` and `REVENUECAT_USE_TEST_STORE=true`. Release
builds ignore that opt-in and always select the matching Apple or Google public
key. Test Store keys must never be used for a release build.

The `app-build-ios`, `app-build-ipa`, `app-build-apk`, and
`app-build-appbundle` recipes require Sentry to be enabled in the selected app
config and require all three upload credentials. They build with obfuscation and
split debug information, then upload native symbols, Dart symbols, the
obfuscation map, and source context to Sentry. The recipe fails if either the
build or upload fails.
