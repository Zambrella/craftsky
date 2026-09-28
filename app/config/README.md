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

`SENTRY_DSN` is public client configuration once the app is shipped, but keep it
out of committed examples. `SENTRY_AUTH_TOKEN`, `SENTRY_ORG`, and
`SENTRY_PROJECT` are build/upload credentials for Sentry symbolication and must
come from CI secrets or your shell environment, not from these app config files.

`REVENUECAT_IOS_PUBLIC_KEY` and `REVENUECAT_ANDROID_PUBLIC_KEY` are public SDK
keys, not RevenueCat secret API keys. Leave them blank to disable billing safely
for that platform. Never place a RevenueCat secret server key in app config.

Local native debug builds may opt into RevenueCat Test Store by setting
`REVENUECAT_TEST_STORE_PUBLIC_KEY` and `REVENUECAT_USE_TEST_STORE=true`. Release
builds ignore that opt-in and always select the matching Apple or Google public
key. Test Store keys must never be used for a release build.
