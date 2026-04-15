# app

The Craftsky Flutter client. Ships as **app.craftsky.social**.

Uses [`atproto.dart`](https://github.com/myConsciousness/atproto.dart) for atproto primitives, but the happy path for data is:

- **Reads:** `app` → `appview` HTTP API → Postgres index
- **Writes:** `app` → `appview` (which holds PDS OAuth tokens) → user's PDS → Relay → firehose → `appview` indexes it

The app never holds PDS access or refresh tokens. It holds a Craftsky session token issued by the App View.

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

The `bluesky` package is Bluesky-specific and likely not needed here — Craftsky uses its own `social.craftsky.*` lexicon.

## Platform IDs

Native bundle IDs are rooted at `social.craftsky` (reverse of `craftsky.social`), e.g. `social.craftsky.craftsky_app` on Android/iOS.
