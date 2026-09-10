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
