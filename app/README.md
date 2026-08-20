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

The Android application ID and iOS bundle ID are both `social.craftsky.app`.

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
`--dart-define-from-file=app/config/<env>.env`. They discover the current
worktree stack's published AppView port and override `CRAFTSKY_API_BASE_URL`
for that run; the remaining local settings still come from the config file.
Release builds **require** a config file with `CRAFTSKY_API_BASE_URL`; the app
throws on first API call if it's missing.

`just app-run-android` also installs an ADB reverse mapping from the emulator's
loopback address to the same worktree-specific host port. The app uses
`10.0.2.2:<port>` for normal API requests, but atproto's localhost OAuth client
requires its browser callback to use `127.0.0.1:<port>`. The reverse mapping
lets that callback reach the matching local AppView from Android.

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

OAuth completion uses verified HTTPS links on `app.craftsky.social`; the app no
longer registers a production custom `craftsky://` scheme. The accepted routes are:

- `https://app.craftsky.social/auth/complete?code=…` for login. The URL carries
  only a short-lived, single-use handoff code; it never carries a session
  bearer.
- `https://app.craftsky.social/account-deletion/reauth-complete?job-id=…&proof=…`
  for account-deletion reauthentication. The proof is single use and the URL
  never carries the ordinary CraftSky session bearer.

Smoke tests (synthetic values should open the app and then fail closed):

```bash
# iOS simulator
xcrun simctl openurl booted \
  'https://app.craftsky.social/auth/complete?code=test-code'

# Android emulator
adb shell am start -W -a android.intent.action.VIEW \
  -d 'https://app.craftsky.social/auth/complete?code=test-code' \
  social.craftsky.app
```

Publishing the domain-association files is a release gate:

- `https://app.craftsky.social/.well-known/assetlinks.json` must authorize
  `social.craftsky.app` with the SHA-256 fingerprint of the real Android release
  signing certificate. The release fingerprint is not stored in this repo and
  must not be replaced with a debug-keystore fingerprint.
- `https://app.craftsky.social/.well-known/apple-app-site-association` must
  authorize application ID `B6YZZCUZWS.social.craftsky.app` for exactly
  `/auth/complete` and `/account-deletion/reauth-complete`. The Apple App ID
  and release provisioning profile must also enable Associated Domains.

The files must be served directly by the canonical HTTPS host with the content
types required by Android and Apple. Confirm ownership/deployment of
`app.craftsky.social`, publish both files, and verify them on release-signed
physical devices before release.

### Local development OAuth

Debug builds register the code-only `craftsky-dev` scheme without adding it to
Profile or Release artifacts. Local AppView enables the matching server
capability through `APPVIEW_ENABLE_DEV_OAUTH_SCHEME=true`. Opt the Flutter debug
build in explicitly:

```bash
flutter run --dart-define=CRAFTSKY_DEV_OAUTH_SCHEME=true
```

Login then returns through
`craftsky-dev:///auth/complete?code=...`; account-deletion reauthentication
returns through
`craftsky-dev:///account-deletion/reauth-complete?job-id=...&proof=...`.
Neither URL carries a CraftSky session bearer. Without the Dart define, debug
builds continue to request verified HTTPS links. AppView rejects the server
flag in production, and the release platform manifests contain no custom OAuth
scheme.
