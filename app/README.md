# app

The CraftSky Flutter client. Ships as **app.craftsky.social**.

Uses [`atproto.dart`](https://github.com/myConsciousness/atproto.dart) for atproto primitives, but the happy path for data is:

- **Reads:** `app` → `appview` HTTP API → Postgres index
- **Writes:** `app` → `appview` (which holds PDS OAuth tokens) → user's PDS → Relay → firehose → `appview` indexes it

The app never holds PDS access or refresh tokens. It holds a CraftSky session token issued by the App View.

## Development

```bash
cd app
flutter pub get
flutter run
```

Point the app at a local App View (`http://localhost:8080` by default) once `appview/` is running.

## Key Packages

- [`atproto`](https://pub.dev/packages/atproto) — core `com.atproto.*` operations
- [`bluesky_text`](https://pub.dev/packages/bluesky_text) — text parsing, facets, mentions, links (reusable for any atproto app)

The `bluesky` package is Bluesky-specific and likely not needed here — CraftSky uses its own `social.craftsky.*` lexicon.

## Platform IDs

Native bundle IDs are rooted at `social.craftsky` (reverse of `craftsky.social`), e.g. `social.craftsky.craftsky_app` on Android/iOS.

## Dev setup

### Base URL

The app talks to the AppView via `CRAFTSKY_API_BASE_URL`. In debug builds the
default is `http://10.0.2.2:18080` (Android emulator → host). Chrome, macOS,
and iOS simulator runs use `localhost` instead.

Initialize local app config once:

```bash
just app-env-init
```

Then run from the repo root:

```bash
just app-run-ios
just app-run-android
just app-run-chrome
```

Under the hood these recipes call Flutter with
`--dart-define-from-file=app/config/<env>.env`. Release builds **require** a
config file with `CRAFTSKY_API_BASE_URL`; the app throws on first API call if
it's missing.

Sentry runtime config uses the same files:

```env
SENTRY_DSN=
SENTRY_ENVIRONMENT=development
SENTRY_RELEASE=
SENTRY_DIST=
SENTRY_LOCAL_OPT_IN=false
```

Keep Sentry upload credentials (`SENTRY_AUTH_TOKEN`, `SENTRY_ORG`,
`SENTRY_PROJECT`) in CI secrets or your shell environment, not in app config.

## Deep links

The app registers `craftsky://` as a custom URL scheme. The OAuth flow lands
on `craftsky:///auth/complete?token=…` (triple slash — empty host, path
`/auth/complete`) after the user authenticates at their PDS. Smoke tests:

```bash
# iOS simulator
xcrun simctl openurl booted 'craftsky:///auth/complete?token=testtoken'

# Android emulator (replace the package name if applicationId differs)
adb shell am start -W -a android.intent.action.VIEW \
  -d 'craftsky:///auth/complete?token=testtoken' \
  social.craftsky.app
```

Both should land on the "Signing in…" screen and surface a `NoPendingSignIn`
error (since no sign-in is in progress — correct behaviour for a bare link).
