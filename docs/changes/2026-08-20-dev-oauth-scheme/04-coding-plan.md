# Development OAuth Scheme Coding Plan

1. Add a constrained `dev_scheme` handoff mode and migration without accepting a client-provided callback URL.
2. Add `APPVIEW_ENABLE_DEV_OAUTH_SCHEME`, default false; allow it only in development and thread the resulting capability to OAuth handlers.
3. Render fixed `craftsky-dev` login and deletion callback URLs using existing escaped query construction and security headers.
4. Add a pure Flutter handoff-mode policy gated by `kDebugMode` plus `CRAFTSKY_DEV_OAUTH_SCHEME`; keep the existing exchange/confirm flow unchanged.
5. Register `craftsky-dev` in Android's debug source set and iOS's Debug-only plist/build setting. Leave main/Profile/Release configuration unchanged.
6. Preserve release tests proving the production artifacts contain only verified HTTPS associations.

## Guardrails

- Never place a bearer in a URL.
- Never accept an arbitrary custom callback URI from Flutter.
- Never enable the scheme solely from a Dart define or solely from server environment.
- Never add the custom scheme to Android main/release or the iOS release plist.
- Do not change production verified-link behavior.

