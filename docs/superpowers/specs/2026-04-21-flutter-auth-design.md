# Flutter Auth (v1) — design

**Date:** 2026-04-21
**Status:** proposed
**Scope:** Wire atproto OAuth into the Flutter app against the existing AppView BFF endpoints. Mobile-only (iOS + Android). Replaces the stub `authStatusProvider` bool with a real session-backed flow.

## Summary

The AppView already implements atproto OAuth as a confidential BFF client and serves `/auth/login`, `/oauth/callback`, `/auth/logout`, `/whoami`, plus the two `/oauth/*` discovery endpoints (see [2026-04-18-appview-oauth-bff-design.md](2026-04-18-appview-oauth-bff-design.md)). The AppView issues an opaque Craftsky bearer token on successful sign-in; every authenticated call the app makes presents that token.

This spec covers **the Flutter side of that contract**: how the app starts the flow, catches the callback, persists the token, validates it on cold start, signs out, and exposes auth state to the router. It also covers the minimal `dio`-based API client that every future feature will lean on, and the platform configuration required for the deep-link handoff.

## Goals

1. Replace `authStatusProvider` (a `kDebugMode`-gated bool) with a real `AsyncValue<AuthState>` notifier backed by the AppView.
2. Sign-in flow: user enters handle → system browser → PDS auth → deep link returns → app lands on feed.
3. Persistent sign-in via `flutter_secure_storage`; optimistic cold start with background `/whoami` validation.
4. Single-device logout from the Settings page.
5. Per-DID onboarding flag persisted in `shared_preferences`, replacing the debug-mode ternary.
6. A minimal `CraftskyApiClient` (three methods: `login`, `whoami`, `logout`) that Bearer-injects from secure storage and maps errors to a sealed `ApiException`.
7. Custom URL scheme (`craftsky://auth/complete`) for the OAuth handoff — no domain-association infra.

## Non-goals

- **HTTPS Universal Links / App Links.** Deferred to a follow-up spec that introduces content-sharing deep links (`/profile/:handle`, `/post/:uri`) — at which point brand-domain hosting + AASA / `assetlinks.json` become worth standing up. The app can support both schemes simultaneously during any future transition; custom-scheme tokens don't need migrating.
- **Web or desktop platforms.** `app_dependencies.dart` explicitly restricts to iOS / Android / Web, and Web isn't a current target. Desktop throws `UnsupportedError`. All three are follow-up specs if and when they're prioritised.
- **Sign-out-everywhere (`?all=true`).** The server supports it; the Flutter UI doesn't expose it in v1. Needs a confirmation dialog + a second settings row; defer until the profile/settings screen grows.
- **Server-side onboarding state.** No server endpoint for "has this user completed onboarding" exists yet. Onboarding stays local for v1.
- **Write-proxy calls.** Creating `social.craftsky.feed.post` records through the AppView is its own spec. This one only guarantees the app can authenticate and identify itself.
- **E2E tests against a real PDS.** Library correctness (indigo, dio) is upstream's problem; we unit / widget-test our glue.
- **Deferred-deep-link fallback (app not installed).** Handled by the AppView's existing callback HTML page rendering a "Don't have the app yet? [App Store] [Play Store]" block. Small future-work server-side template tweak, not a Flutter concern.
- **Rate-limiting / retry / circuit-breaker** behaviour on the client. Out of scope; future work.

## 1. Architecture

Three logical slices, all inside `app/lib/`:

1. **`shared/api/`** — `dio`-based `CraftskyApiClient` with two interceptors (Bearer auth, error mapping). Three typed methods for v1: `login(handle)`, `whoami()`, `logout()`. Exposed via a keep-alive provider.
2. **`auth/`** — sealed `AuthState`, `SecureTokenStorage`, `AuthStateProvider` (optimistic cold start + background validation), `AuthController` (state machine: `signIn` / `completeFromDeepLink` / `signOut`). Existing `welcome_page.dart` and `sign_in_page.dart` get rewired; a new `auth_complete_page.dart` is the deep-link landing screen. The old `authStatusProvider` is deleted.
3. **`onboarding/`** — rewritten `OnboardingStatus` notifier as a family keyed by DID, backed by `SharedPreferences`. The debug-mode ternary goes away.

The router in `lib/router/router.dart` grows:
- One new root-navigator route: `AuthCompleteRoute` at `/auth/complete`.
- The redirect function reads the new `AsyncValue<AuthState>` with `ref.read` and uses a `ChangeNotifier` owned by the `goRouter` provider to trigger re-evaluation on auth / onboarding state changes.

```
┌──────────────────────────────────────────────────────────────┐
│                       Flutter App                            │
│                                                              │
│  ┌──────────────────┐       ┌──────────────────────────┐     │
│  │  SignInPage /    │──────▶│  AuthController          │     │
│  │  SettingsPage    │       │  .signIn / .signOut /    │     │
│  └──────────────────┘       │  .completeFromDeepLink   │     │
│                             └───────────┬──────────────┘     │
│                                         │                    │
│                ┌────────────────────────┼────────────────┐   │
│                ▼                        ▼                ▼   │
│     ┌────────────────┐      ┌──────────────────┐   ┌───────┐ │
│     │ SecureToken-   │      │ CraftskyApi-     │   │ Auth- │ │
│     │ Storage        │      │ Client (dio)     │   │ State │ │
│     └────────────────┘      └────────┬─────────┘   └───────┘ │
│             │                        │                       │
│             │       Bearer injection │                       │
│             └────────────────────────┘                       │
│                                      │                       │
│                                      │ HTTPS JSON            │
│                                      ▼                       │
└──────────────────────────────────────┼───────────────────────┘
                                       │
                  ┌────────────────────┴────────────────────┐
                  │   AppView (BFF — already implemented)   │
                  │   POST /v1/auth/login                   │
                  │   GET  /v1/whoami                       │
                  │   POST /v1/auth/logout                  │
                  └─────────────────────────────────────────┘
```

### Endpoint paths (`/v1/...`)

A concurrent AppView work-stream is moving client-facing endpoints behind `/v1/...` (`/v1/auth/login`, `/v1/auth/logout`, `/v1/whoami`). The `/oauth/*` endpoints (client-metadata, jwks, callback) are atproto-spec-facing and stay unversioned.

The Flutter client targets `/v1/...` from day one. If this spec lands before the server migration, a small PR renames three path constants. If the server migration lands first, the app targets correct paths natively.

**`CraftskyApiClient` takes the bare host URL** (`https://appview.craftsky.social`) and composes `/v1/auth/login` etc. internally — `/v1` is **not** baked into the base URL. This leaves room for future non-`/v1` paths (health checks, unversioned atproto-style XRPC endpoints) without a base-URL split.

## 2. Data model

Three pieces of state, each with one owner.

### 2.1 `AuthState` — sealed class

```dart
sealed class AuthState {
  const AuthState();
}

final class SignedOut extends AuthState {
  const SignedOut();
}

final class SignedIn extends AuthState {
  const SignedIn({required this.did, required this.handle, required this.token});
  final String did;
  final String handle;
  final String token;
}
```

Exposed as `AsyncValue<AuthState>` via `authSessionProvider` (the notifier class is `AuthSession` — named distinctly from the sealed `AuthState` type to avoid shadowing inside the notifier's `build()`). `AsyncLoading` is used transiently on cold start (while secure storage resolves), during `signOut`, and during background re-validation.

### 2.2 Secure-storage payload

Single JSON blob under `flutter_secure_storage` key `craftsky_session`:

```json
{
  "token": "<opaque bearer token>",
  "did": "did:plc:...",
  "handle": "alice.bsky.social"
}
```

Stored as one value (atomic read / write). Represented in Dart as a `@MappableClass` record, matching the project's `dart_mappable` convention.

Rationale for storing `did` + `handle` alongside the token: lets the optimistic cold start emit `SignedIn(did, handle)` immediately without a `/whoami` round-trip. The cached values are treated as hints — the background `/whoami` call is authoritative and overwrites them if the server returns different values (e.g., user changed their handle on the PDS).

### 2.3 Pending-auth state

Owned internally by `AuthController`, not exposed via its own provider:

```dart
@MappableClass()
class PendingAuth with PendingAuthMappable {
  PendingAuth({required this.handle, required this.startedAt});
  final String handle;
  final DateTime startedAt;
}
```

Held in a `pendingAuthProvider` (a plain `@riverpod` class exposing `start(handle)` / `clear()`). Used by `completeFromDeepLink` to (a) confirm a sign-in is in progress and (b) reject stale deep links (older than 10 minutes).

### 2.4 Onboarding state

Family-keyed by DID, backed by `SharedPreferences`:

```dart
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  bool build(String did) {
    final prefs = ref.watch(sharedPreferencesProvider);
    return prefs.getBool('onboarded_$did') ?? false;
  }

  Future<void> finish() async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setBool('onboarded_$did', true);
    if (!ref.mounted) return;
    state = true;
  }
}
```

The existing `kDebugMode ? true : false` ternary is removed. First-run is per-DID, survives reinstall only if `SharedPreferences` data survives (it does on iOS by default; on Android it survives unless the user clears app data).

## 3. Providers & state machine

### 3.1 `AuthSession` — optimistic cold start + background validation

```dart
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  @override
  Future<AuthState> build() async {
    final storage = ref.watch(secureTokenStorageProvider);
    final stored = await storage.read();

    if (stored == null) {
      return const SignedOut();
    }

    // Optimistic: emit SignedIn from cached values immediately;
    // background-validate via /whoami. The validation runs unawaited
    // so `build` resolves without blocking on the network — the
    // router lands the user on /feed using the cached DID, and the
    // background call flips to SignedOut if the server rejects the
    // token (e.g. revoked by another device).
    unawaited(_validateInBackground(stored));

    return SignedIn(did: stored.did, handle: stored.handle, token: stored.token);
  }

  Future<void> _validateInBackground(StoredSession stored) async {
    try {
      final api = ref.read(apiClientProvider);
      final who = await api.whoami();
      if (!ref.mounted) return;

      if (who.did != stored.did) {
        // Server says this token belongs to a different account.
        // Treat as stale; clear.
        await _clearLocalState();
        return;
      }
      if (who.handle != stored.handle) {
        // Handle renamed on the PDS; update cache.
        await ref.read(secureTokenStorageProvider).write(
          StoredSession(token: stored.token, did: who.did, handle: who.handle),
        );
        if (!ref.mounted) return;
        state = AsyncData(SignedIn(did: who.did, handle: who.handle, token: stored.token));
      }
      // else: handles match; nothing to do.
    } on ApiUnauthorized {
      await _clearLocalState();
    } on ApiNetworkError {
      // Offline; keep cached SignedIn state. Next launch will re-validate.
    }
  }

  Future<void> _clearLocalState() async {
    await ref.read(secureTokenStorageProvider).clear();
    if (!ref.mounted) return;
    state = const AsyncData(SignedOut());
  }

  // Called by AuthController after a successful sign-in handoff.
  void setSignedIn(SignedIn signedIn) => state = AsyncData(signedIn);

  // Called by AuthController on sign-out and by the global
  // 401-unauthorized interceptor (see §5.3) when a mid-session
  // request is rejected by the server.
  void setSignedOut() => state = const AsyncData(SignedOut());
}
```

Key properties:

- **Async `build()`** — returns `Future<AuthState>`. Initial consumers see one brief `AsyncLoading` state while secure storage resolves (typically sub-10 ms on iOS Keychain / Android Keystore); the router redirect already holds position on `valueOrNull == null`, so this produces no visible flicker. This matches how `appDependenciesProvider` already gates the rest of the app.
- **DID mismatch handling.** Background validation compares `who.did` against the stored DID. A mismatch means the token authenticates a different account than we thought (pathological case, but possible if storage is manually corrupted or sessions are swapped); we treat it as stale and sign out.
- **Handle drift handling.** A handle change on the PDS updates the cache. The token is unaffected.
- **Offline tolerance.** Network failure during validation keeps the user signed in. 401 (definitely invalid) signs them out. Server error (5xx) is treated like network error — we don't sign out on transient server issues.
- **Mid-session revocation.** Any 401 on an authenticated API call — from background validation, a feed fetch, any future write-proxy call — funnels through `_ErrorMappingInterceptor`'s 401 handler (§5.3), which calls `setSignedOut()` and clears storage. There is no token refresh to attempt (see §3.4); the only path back is through `/welcome` and a fresh sign-in.

### 3.2 `SecureTokenStorage`

A plain async read/write wrapper around `flutter_secure_storage`. No bootstrap pre-load, no synchronous cache — `AuthSession.build` is async, so a regular `Future<StoredSession?> read()` is enough.

```dart
class SecureTokenStorage {
  SecureTokenStorage(this._fss);
  final FlutterSecureStorage _fss;

  static const _key = 'craftsky_session';

  /// Reads the current session from secure storage.
  ///
  /// A keystore failure on Android (e.g., device-lock removed, OTA
  /// corruption) throws `PlatformException` — we catch it and return
  /// null so the app falls back to SignedOut rather than crashing.
  /// The cause is logged; no user-facing error. Corrupt-blob
  /// (FormatException) clears the row so subsequent writes don't
  /// collide.
  Future<StoredSession?> read() async {
    try {
      final raw = await _fss.read(key: _key);
      return raw == null ? null : StoredSessionMapper.fromJson(raw);
    } on PlatformException catch (e, st) {
      _log.warning('SecureTokenStorage.read failed; treating as unsigned-in', e, st);
      return null;
    } on FormatException catch (e, st) {
      _log.warning('SecureTokenStorage.read: corrupt blob; clearing', e, st);
      await _fss.delete(key: _key).catchError((_) {});
      return null;
    }
  }

  Future<void> write(StoredSession session) =>
      _fss.write(key: _key, value: session.toJson());

  Future<void> clear() => _fss.delete(key: _key);
}
```

Exposed via `@Riverpod(keepAlive: true) SecureTokenStorage secureTokenStorage(Ref ref) => ...;` — the `FlutterSecureStorage` instance is constructed inside the provider with platform-default `AndroidOptions` / `IOSOptions`. `bootstrap.dart` requires no special handling: the first `ref.watch(authSessionProvider)` in the router's build resolves the storage read naturally.

### 3.3 Craftsky token lifecycle

The opaque Craftsky bearer token has **no client-side refresh mechanism** in v1. It is valid until one of the following happens server-side (see the OAuth BFF spec's `craftsky_sessions` schema):

1. **Explicit single-device logout** (`POST /v1/auth/logout`) sets `revoked_at`; future requests with the token return 401.
2. **Sign-out-everywhere** (`POST /v1/auth/logout?all=true`) deletes the parent `oauth_sessions` row, cascading all `craftsky_sessions` rows for the DID. Future requests return 401.
3. **The underlying atproto OAuth refresh token fails** — the user revoked app access at the PDS, or the atproto-spec 180-day refresh-token cap elapsed. Once the AppView can no longer refresh, the `oauth_sessions` row dies and the cascade removes the craftsky token.

From the Flutter client's perspective, all three are indistinguishable: the server returns 401 and we sign out locally. There is nothing to "refresh" or "rotate" — the next sign-in runs the full OAuth flow again to mint a fresh token.

**Implication for the API client.** Any authenticated call can return 401 at any time. The `_ErrorMappingInterceptor` handles this centrally (§5.3) so every feature doesn't implement its own 401 recovery.

**If the server ever introduces token rotation** (e.g. as part of the future TMB upgrade, which hands short-lived access tokens + DPoP material to clients), the `AuthController` grows a `refresh()` method and the interceptor grows refresh-on-401 retry logic. That's genuinely new work; no shim is needed in v1.

### 3.4 `AuthController` — the state machine

```dart
@riverpod
class AuthController extends _$AuthController {
  @override
  FutureOr<void> build() => null; // idle; no initial load

  Future<void> signIn({required String handle}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final trimmed = handle.trim().replaceFirst(RegExp(r'^@'), '');
      if (trimmed.isEmpty) throw const HandleRequired();

      final api = ref.read(apiClientProvider);
      final response = await api.login(handle: trimmed);

      if (!ref.mounted) return;
      ref.read(pendingAuthProvider.notifier).start(trimmed);

      final launched = await launchUrl(
        Uri.parse(response.authUrl),
        mode: LaunchMode.externalApplication,
      );
      if (!launched) throw const BrowserLaunchFailed();
    });
  }

  Future<void> completeFromDeepLink(String token) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final pending = ref.read(pendingAuthProvider);
      if (pending == null) throw const NoPendingSignIn();
      if (DateTime.now().difference(pending.startedAt) > const Duration(minutes: 10)) {
        ref.read(pendingAuthProvider.notifier).clear();
        throw const SignInTimedOut();
      }

      // Use a dedicated HandoffApiClient that carries the just-minted
      // token without touching SecureTokenStorage or AuthSession — see
      // §5 for why this is a separate Dio instance. If the app is
      // killed mid-flow, cold start sees no stored session and the
      // flow restarts cleanly rather than loading a half-written blob.
      final handoff = ref.read(handoffApiClientProvider(token));
      try {
        final who = await handoff.whoami();
        if (!ref.mounted) return;

        // Single write: only persist after we know token → DID resolves.
        await ref.read(secureTokenStorageProvider).write(
          StoredSession(token: token, did: who.did, handle: who.handle),
        );
        if (!ref.mounted) return;

        ref.read(authSessionProvider.notifier).setSignedIn(
          SignedIn(did: who.did, handle: who.handle, token: token),
        );
      } finally {
        if (ref.mounted) {
          ref.read(pendingAuthProvider.notifier).clear();
        }
      }
    });
  }

  Future<void> signOut() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(apiClientProvider);
      try {
        await api.logout();
      } on ApiException {
        // Local sign-out wins: even if server is unreachable or returns
        // an error, clear locally. The token row is lazy-cleaned server-side.
      }
      if (!ref.mounted) return;
      await ref.read(secureTokenStorageProvider).clear();
      if (!ref.mounted) return;
      ref.read(authSessionProvider.notifier).setSignedOut();
    });
  }
}
```

**`AuthError`** is a sealed class covering the user-actionable errors:

```dart
sealed class AuthError implements Exception {
  const AuthError();
}
final class HandleRequired extends AuthError { const HandleRequired(); }
final class InvalidHandle extends AuthError { const InvalidHandle(); }
final class ServerUnavailable extends AuthError { const ServerUnavailable(); }
final class BrowserLaunchFailed extends AuthError { const BrowserLaunchFailed(); }
final class NoPendingSignIn extends AuthError { const NoPendingSignIn(); }
final class SignInTimedOut extends AuthError { const SignInTimedOut(); }
final class StorageFailure extends AuthError {
  const StorageFailure(this.cause);
  final Object cause;
}
```

Pages consume `authControllerProvider`'s `AsyncValue<void>` via `ref.listen` for transition-based error surfacing (the Riverpod rules' preferred pattern). `AsyncError(:final error)` cases pattern-match on `AuthError` for UI messaging; anything else maps to a generic "Something went wrong" with the raw error logged via the `logging` package.

## 4. OAuth flow & deep link

### 4.1 Sign-in flow

```
┌─────────────┐     ┌──────────────┐      ┌─────────────┐     ┌──────┐
│ Flutter app │     │   AppView    │      │   Browser   │     │ PDS  │
└──────┬──────┘     └───────┬──────┘      └──────┬──────┘     └──┬───┘
       │                    │                    │               │
       │ signIn("alice.bsky.social")              │               │
       │                    │                    │               │
       │ POST /v1/auth/login                      │               │
       │ {handle, handoff_mode: "deep_link"}      │               │
       ├───────────────────▶│                    │               │
       │◀── 200 {auth_url} ─┤                    │               │
       │                    │                    │               │
       │ pendingAuth.start("alice.bsky.social")   │               │
       │                    │                    │               │
       │ url_launcher(authUrl, externalApplication)               │
       ├───────────────────────────────────────▶│               │
       │                    │                    │ user auths    │
       │                    │                    ├──────────────▶│
       │                    │                    │◀─ redirect ───┤
       │                    │ GET /oauth/callback?code=…&state=… │
       │                    │◀───────────────────┤               │
       │                    │ code → tokens (server-to-server)   │
       │                    ├───────────────────────────────────▶│
       │                    │◀─ tokens ──────────────────────────┤
       │                    │ mints Craftsky token + HTML page   │
       │                    │ with craftsky://auth/complete?token=…
       │                    ├───────────────────▶│               │
       │                    │                    │ meta-refresh  │
       │                    │                    │ craftsky://…  │
       │◀── deep link ──────────────────────────┤               │
       │                    │                    │               │
       │ AuthCompletePage initState → completeFromDeepLink(token) │
       │   ├─ validate pendingAuth freshness (<10min)             │
       │   ├─ build HandoffApiClient(token) — Bearer baked in     │
       │   │                                                      │
       │ GET /v1/whoami     │                    │               │
       ├───────────────────▶│                    │               │
       │◀── 200 {did, handle} ─┤                 │               │
       │   ├─ SINGLE write: storage ← {token, did, handle}        │
       │   ├─ authSession.setSignedIn(...)                        │
       │   └─ finally: pendingAuth.clear()                        │
       │                                                          │
       │ router refresh fires → redirect re-evaluates            │
       │   → /feed (or /onboarding if first-run for this DID)    │
```

### 4.2 Deep-link receipt via `go_router`

A new root-navigator route catches the deep link:

```dart
@TypedGoRoute<AuthCompleteRoute>(
  path: RouteLocations.authComplete,   // '/auth/complete'
  name: 'auth-complete',
)
class AuthCompleteRoute extends GoRouteData with $AuthCompleteRoute {
  const AuthCompleteRoute({required this.token});

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  final String token; // from ?token=...

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const AuthCompletePage();
}
```

`AuthCompletePage` is a consumer widget that:
- Fires `AuthController.completeFromDeepLink(token)` from `initState` via `addPostFrameCallback` (per Riverpod rules for async notifier calls in initState).
- Renders a `CircularProgressIndicator` + "Signing in…" text.
- Uses `ref.listen(authControllerProvider, ...)` with the transition-based `(prev, state)` pattern to detect completion and surface errors via snackbar / retry UI.

**Why a page, not just a `redirect` hook.** The `completeFromDeepLink` call can take several hundred ms (one `whoami` round-trip); surfacing that as a visible "Signing in…" screen is better UX than a blank bounce. Errors have a place to render. And it gives the top-level `authSessionProvider` redirect a single, well-defined location to route away from once the controller flips state.

go_router receives custom-scheme URIs via the same platform channel as HTTPS URIs; matching happens on path + query, so `craftsky://auth/complete?token=xxx` and `https://.../auth/complete?token=xxx` match the same route. This means the migration to HTTPS Universal Links (if/when it happens) is a platform-config change only — no Flutter code churn.

### 4.3 Router-level redirect & refresh

`goRouter`'s `redirect` uses `ref.read` exclusively; a `ChangeNotifier` owned by the provider is kicked via `ref.listen` when auth state changes. Onboarding refreshes use `ref.listen` on the specific family instance for the currently-signed-in DID:

```dart
@riverpod
GoRouter goRouter(Ref ref) {
  final refresh = ChangeNotifier();
  ref.onDispose(refresh.dispose);

  ref.listen(authSessionProvider, (_, next) {
    refresh.notifyListeners();

    // When auth state becomes SignedIn, start listening to that DID's
    // onboarding status too. Riverpod family providers are only
    // listenable at a specific instance (onboardingStatusProvider(did)),
    // not at the family level — so we attach/detach the listener as the
    // signed-in DID changes. A nullable subscription holds the current
    // one; `ref.onDispose` cleans it up alongside the ChangeNotifier.
    _reattachOnboardingListener(ref, next.valueOrNull, refresh);
  });

  return GoRouter(
    initialLocation: RouteLocations.welcome,
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    refreshListenable: refresh,
    redirect: (context, state) {
      final loc = state.matchedLocation;

      const unauthenticatedRoutes = [
        RouteLocations.welcome,
        RouteLocations.signIn,
      ];

      final auth = ref.read(authSessionProvider).valueOrNull;
      if (auth == null) return null; // transient AsyncLoading — hold position

      switch (auth) {
        case SignedOut():
          // While signed-out, /auth/complete is allowed (it's where a
          // deep link lands before completeFromDeepLink updates state).
          if (loc == RouteLocations.authComplete) return null;
          return unauthenticatedRoutes.contains(loc)
              ? null
              : RouteLocations.welcome;

        case SignedIn(:final did):
          // Once signed in, /auth/complete has served its purpose —
          // the top-level redirect is what moves the user onward.
          // Onboarding state is only keyed by a real DID; no empty-string
          // keys reach SharedPreferences.
          final isOnboarded = ref.read(onboardingStatusProvider(did));
          if (loc == RouteLocations.authComplete) {
            return isOnboarded ? RouteLocations.home : RouteLocations.onboarding;
          }
          if (!isOnboarded && loc != RouteLocations.onboarding) {
            return RouteLocations.onboarding;
          }
          if (isOnboarded &&
              (unauthenticatedRoutes.contains(loc) ||
                  loc == RouteLocations.onboarding)) {
            return RouteLocations.home;
          }
          return null;
      }
    },
    routes: $appRoutes,
    errorBuilder: ...,
  );
}
```

Notes:
- `/auth/complete` is allowed-to-stay only when `SignedOut`. Once `AuthController.completeFromDeepLink` flips state to `SignedIn` (or back to `SignedOut` on failure), the refresh fires and the redirect routes the user onward — to `/feed` or `/onboarding` on success; to `/welcome` on failure (the `AuthCompletePage` surfaces the error via snackbar before the redirect catches it). This closes the gap the review flagged.
- `_reattachOnboardingListener` is a small helper (co-located in the same file) that manages a single `ProviderSubscription<bool>` against `onboardingStatusProvider(did)` for whatever DID is currently `SignedIn`. The subscription handle is held in a closure-captured `ProviderSubscription<bool>? onboardingSub` variable owned by the `goRouter` provider's build body, next to the `ChangeNotifier`. The helper's contract: on `SignedIn(newDid)` with no existing sub or a different DID, close the old sub (if any) and `ref.listen(onboardingStatusProvider(newDid), (_, __) => refresh.notifyListeners())`; on `SignedOut`, close and null the sub. `ref.onDispose(() => onboardingSub?.close())` tears it down with the rest of the router. This is the concrete mechanism for "onboarding flips → redirect re-evaluates" — no family-level listen required.
- The redirect uses `ref.read` throughout, never `ref.watch`. go_router does not auto-re-evaluate on watched providers; the `refreshListenable` pattern is how we drive re-evaluation explicitly.
- No `GoRouterRefreshStream` or stream-bridge — just a plain `ChangeNotifier` driven by `ref.listen`.

## 5. API client

Two Dio instances, each with a single, narrow job:

- **`dioProvider`** — the **session Dio**. Reads the Bearer token from `authSessionProvider`'s current state. Used for every authenticated call from feature code (`/v1/whoami` background validation, future feed/post calls). Handles mid-session 401 by globally signing the user out.
- **`handoffApiClientProvider(token)`** — the **handoff Dio**, constructed with a specific token baked into its default headers. Built on demand by `AuthController.completeFromDeepLink`, used for exactly one `whoami` call, disposed immediately. 401 on this Dio does NOT sign the user out — it surfaces as an `ApiUnauthorized` and `AuthController` maps it to an `AuthError`.

Splitting them removes the "is this a handoff request?" branching that would otherwise live inside interceptors. The session Dio never carries a token that isn't in `authSessionProvider`; the handoff Dio never outlives a single call.

### 5.1 `CraftskyApiClient` — session-scoped

```dart
class CraftskyApiClient {
  const CraftskyApiClient(this._dio);
  final Dio _dio;

  /// POST /v1/auth/login — anonymous, no Bearer header required.
  Future<LoginResponse> login({required String handle}) => _unwrap(() async {
        final res = await _dio.post<Map<String, dynamic>>(
          '/v1/auth/login',
          data: {'handle': handle, 'handoff_mode': 'deep_link'},
        );
        return LoginResponseMapper.fromMap(res.data!);
      });

  /// GET /v1/whoami — authenticated via the session Dio.
  Future<WhoAmI> whoami() => _unwrap(() async {
        final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
        return WhoAmIMapper.fromMap(res.data!);
      });

  /// POST /v1/auth/logout — authenticated via the session Dio.
  Future<void> logout() => _unwrap(() async {
        await _dio.post<void>('/v1/auth/logout');
      });

  /// Translates `DioException` carrying an `ApiException` in `.error`
  /// into a direct `ApiException` throw. See §5.4.
  Future<T> _unwrap<T>(Future<T> Function() body) async {
    try {
      return await body();
    } on DioException catch (e) {
      final err = e.error;
      if (err is ApiException) throw err;
      throw ApiServerError(e.message ?? 'server_error');
    }
  }
}
```

### 5.2 `HandoffApiClient` — token-scoped, single-use

```dart
class HandoffApiClient {
  const HandoffApiClient(this._dio);
  final Dio _dio;

  Future<WhoAmI> whoami() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
      return WhoAmIMapper.fromMap(res.data!);
    } on DioException catch (e) {
      final err = e.error;
      if (err is ApiException) throw err;
      throw ApiServerError(e.message ?? 'server_error');
    }
  }
}
```

Only `whoami` is exposed — this client has no other purpose. It does not carry the 401 sign-out behaviour (its own interceptor set leaves 401 alone).

Response shapes (`dart_mappable`):

```dart
@MappableClass()
class LoginResponse with LoginResponseMappable {
  LoginResponse({required this.authUrl});
  final String authUrl;
}

@MappableClass()
class WhoAmI with WhoAmIMappable {
  WhoAmI({required this.did, required this.handle});
  final String did;
  final String handle;
}
```

The `handoff_mode: "deep_link"` constant is baked in — loopback mode is CLI-only, not an app concern.

### 5.3 Dio construction — two providers

```dart
// --- Shared base options / error interceptor -----------------------

const _devDefaultBaseUrl = 'http://10.0.2.2:8080'; // Android-emulator default.
const _baseUrl = String.fromEnvironment(
  'CRAFTSKY_API_BASE_URL',
  defaultValue: kDebugMode ? _devDefaultBaseUrl : '',
);

BaseOptions _baseOptions() {
  if (_baseUrl.isEmpty) {
    throw StateError(
      'CRAFTSKY_API_BASE_URL must be set for non-debug builds. '
      'Pass it via --dart-define.',
    );
  }
  return BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    headers: {'Content-Type': 'application/json'},
  );
}

// --- Session Dio ---------------------------------------------------

@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  final dio = Dio(_baseOptions());
  dio.interceptors.addAll([
    _SessionAuthInterceptor(ref),
    _SignOutOn401Interceptor(ref),
    _ErrorMappingInterceptor(),
  ]);
  return dio;
}

@Riverpod(keepAlive: true)
CraftskyApiClient craftskyApiClient(Ref ref) =>
    CraftskyApiClient(ref.watch(dioProvider));

// --- Handoff Dio (family — one instance per token) -----------------

@riverpod
HandoffApiClient handoffApiClient(Ref ref, String token) {
  final dio = Dio(_baseOptions().copyWith(
    headers: {
      ..._baseOptions().headers ?? const <String, dynamic>{},
      'Authorization': 'Bearer $token',
    },
  ));
  dio.interceptors.add(_ErrorMappingInterceptor());
  return HandoffApiClient(dio);
}
```

The handoff provider is **not** keep-alive — the `family` parameter (the token) is unique per sign-in, so the instance gets auto-disposed when no one watches it. `AuthController.completeFromDeepLink` reads it exactly once.

### 5.4 Interceptors

**`_SessionAuthInterceptor`** — attaches the Bearer token read from `authSessionProvider`. Tests control its behaviour by overriding `authSessionProvider` (no injected resolver callback needed).

```dart
const _anonymousPaths = <String>{'/v1/auth/login'};

class _SessionAuthInterceptor extends Interceptor {
  _SessionAuthInterceptor(this._ref);
  final Ref _ref;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (_anonymousPaths.contains(options.path)) {
      handler.next(options);
      return;
    }
    final auth = _ref.read(authSessionProvider).value;
    if (auth is SignedIn) {
      options.headers['Authorization'] = 'Bearer ${auth.token}';
    }
    handler.next(options);
  }
}
```

**`_SignOutOn401Interceptor`** — on 401 from the session Dio, signs the user out. Because the handoff Dio has its own (simpler) interceptor stack, this one never fires for handoff calls.

```dart
class _SignOutOn401Interceptor extends Interceptor {
  _SignOutOn401Interceptor(this._ref);
  final Ref _ref;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    final status = err.response?.statusCode;
    if (status == 401) {
      // Fire-and-forget storage clear; synchronous state flip.
      unawaited(_ref.read(secureTokenStorageProvider).clear());
      _ref.read(authSessionProvider.notifier).setSignedOut();
    }
    handler.next(err);
  }
}
```

**`_ErrorMappingInterceptor`** — the same for both Dios. Converts `DioException` → `ApiException`:

- `connectionTimeout` / `sendTimeout` / `receiveTimeout` / `connectionError` (or socket-ish unknowns) → `ApiNetworkError`.
- `badResponse` with `statusCode == 401` → `ApiUnauthorized`.
- `badResponse` with `statusCode ∈ [400, 499]` → `ApiBadRequest(code: response.data['error'] as String?)`. Server's error JSON shape: `{"error": "handle_required"}` (see `appview/internal/auth/handlers_session.go:writeJSONError`).
- `badResponse` with `statusCode >= 500` → `ApiServerError`.
- Anything else → generic `ApiServerError`.

Exhaustive switch in the mapper uses merged cases where behaviour is identical:

```dart
ApiException _mapDioError(DioException err) {
  return switch (err.type) {
    DioExceptionType.connectionTimeout ||
    DioExceptionType.sendTimeout ||
    DioExceptionType.receiveTimeout ||
    DioExceptionType.connectionError =>
      ApiNetworkError(err.message ?? err.type.name),
    DioExceptionType.badResponse => _mapBadResponse(err),
    DioExceptionType.cancel ||
    DioExceptionType.badCertificate ||
    DioExceptionType.unknown =>
      err.error is Exception
          ? ApiNetworkError(err.message ?? 'network_error')
          : ApiServerError(err.message ?? 'server_error'),
  };
}
```

```dart
sealed class ApiException implements Exception {
  const ApiException(this.message);
  final String message;
}
final class ApiUnauthorized extends ApiException { const ApiUnauthorized() : super('unauthorized'); }
final class ApiBadRequest extends ApiException {
  const ApiBadRequest(this.code) : super(code ?? 'bad_request');
  final String? code;
}
final class ApiServerError extends ApiException { const ApiServerError(super.message); }
final class ApiNetworkError extends ApiException { const ApiNetworkError(super.message); }
```

### 5.5 Error → UI mapping

`AuthController` converts `ApiException` to `AuthError` in the `signIn` path using merged switch cases where behaviour is shared:

```dart
AuthError _mapSignInError(ApiException e) => switch (e) {
      ApiBadRequest(code: 'handle_required') => const HandleRequired(),
      ApiBadRequest() => const InvalidHandle(),
      // Login is anonymous; a 401 here is defensive-only, treated like a
      // transient server failure so the user can retry.
      ApiNetworkError() || ApiServerError() || ApiUnauthorized() =>
        const ServerUnavailable(),
    };
```

`SignInPage` listens to `authControllerProvider` and maps `AuthError` to a user-facing message via `Theme.of(context).textTheme` / `ScaffoldMessenger`. Per the flutter.md rule, the snackbar content widget is a separate `AuthErrorSnackBarContent` widget class, not a helper method.

## 6. Package layout & file changes

### 6.1 New files

```
app/lib/
├── shared/
│   └── api/
│       ├── api_exception.dart
│       ├── craftsky_api_client.dart
│       ├── handoff_api_client.dart
│       ├── models/
│       │   ├── login_response.dart           # + .mapper.dart
│       │   └── whoami.dart                   # + .mapper.dart
│       └── providers/
│           ├── api_client_provider.dart      # + .g.dart (craftskyApiClient, handoffApiClient)
│           └── dio_provider.dart             # + .g.dart (session Dio + shared interceptors)
├── auth/
│   ├── models/
│   │   ├── auth_error.dart
│   │   ├── auth_state.dart
│   │   ├── pending_auth.dart                 # + .mapper.dart
│   │   └── stored_session.dart               # + .mapper.dart
│   ├── providers/
│   │   ├── auth_controller.dart              # + .g.dart
│   │   ├── auth_session_provider.dart        # + .g.dart  (notifier: AuthSession)
│   │   ├── pending_auth_provider.dart        # + .g.dart
│   │   └── secure_token_storage.dart         # + .g.dart
│   └── pages/
│       └── auth_complete_page.dart
```

### 6.2 Modified files

| File | Change |
|---|---|
| `app/pubspec.yaml` | Add `dio`, `flutter_secure_storage`, `url_launcher`; add `http_mock_adapter` to dev deps. |
| `app/lib/bootstrap.dart` | Reads `dioProvider` once before `runApp` to fail-fast on release builds missing `--dart-define CRAFTSKY_API_BASE_URL`. No secure-storage preload is needed — `authSessionProvider.build` is async and reads storage itself (see §3.1 / §3.2). |
| `app/lib/app_dependencies.dart` | Unchanged in v1. `SecureTokenStorage` is auth-specific and lives under `auth/providers/`, not alongside generic device/package-info deps. |
| `app/lib/router/router.dart` | Redirect rewritten per §4.3: `ref.read` throughout, `refreshListenable: ChangeNotifier` driven by `ref.listen` on `authSessionProvider` + the re-attaching onboarding listener. New `AuthCompleteRoute` root-navigator route. Imports updated: old `authStatusProvider` → new `authSessionProvider`; `onboardingStatusProvider` now takes a `did` argument. |
| `app/lib/router/route_locations.dart` | Add `authComplete = '/auth/complete'` constant. |
| `app/lib/auth/pages/welcome_page.dart` | Remove the "Dev: toggle auth" `OutlinedButton`. "Sign in" and "Create account on a PDS" both continue to `SignInRoute` (account creation is a separate future spec). |
| `app/lib/auth/pages/sign_in_page.dart` | Wire the `BrandTextField` to a controller; "Continue" button calls `ref.read(authControllerProvider.notifier).signIn(handle: text)`. Listen to `authControllerProvider` for loading / error states. |
| `app/lib/onboarding/providers/onboarding_status_provider.dart` | Rewrite as a family keyed by DID; `build(String did)` reads `SharedPreferences`; `finish()` writes it. Remove `kDebugMode` ternary. |
| `app/lib/onboarding/pages/onboarding_page.dart` | `finish()` call unchanged in shape; the provider's `did` argument is obtained by watching `authSessionProvider` and pattern-matching on `SignedIn`. |
| `app/lib/settings/pages/settings_page.dart` | Add `SignOutTile` widget class (its own file or inlined as a class per flutter.md rule) that calls `AuthController.signOut` and listens for completion. |
| `ios/Runner/Info.plist` | Add `CFBundleURLTypes` for scheme `craftsky` (see §7.1). |
| `android/app/src/main/AndroidManifest.xml` | Add a second `intent-filter` on `MainActivity` for `craftsky://auth` (see §7.2). |
| `app/README.md` | Add "Deep links" + "Dev setup" sections covering `--dart-define`, Android-vs-iOS localhost, and the smoke-test URLs. |

### 6.3 Deleted files

- `app/lib/auth/providers/auth_status_provider.dart` (and its generated `.g.dart`) — replaced by `auth/providers/auth_session_provider.dart`.
- Any `part '…'` directive referencing `auth_status_provider.g.dart` must be removed from all call sites before re-running `dart run build_runner build --delete-conflicting-outputs` — otherwise the build will fail on a missing `part of` target.

### 6.4 Dependency additions

```yaml
dependencies:
  dio: ^5.7.0
  flutter_secure_storage: ^9.2.2
  url_launcher: ^6.3.1

dev_dependencies:
  http_mock_adapter: ^0.6.1
```

Versions are representative (latest majors at time of writing); the implementation plan will pin concrete latest-at-implementation values.

## 7. Platform configuration

### 7.1 iOS — custom URL scheme

**`ios/Runner/Info.plist`:**

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLName</key>
    <string>social.craftsky.auth</string>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>craftsky</string>
    </array>
  </dict>
</array>
```

No entitlements, no Associated Domains capability, no `apple-app-site-association`.

**Smoke test:** `xcrun simctl openurl booted 'craftsky://auth/complete?token=testtoken'` on a booted simulator should wake the app and land it on the `AuthCompletePage`.

### 7.2 Android — custom scheme intent filter

**`android/app/src/main/AndroidManifest.xml`** (added alongside the existing `MAIN`/`LAUNCHER` intent-filter on `MainActivity`, not replacing it):

```xml
<intent-filter>
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.DEFAULT" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data android:scheme="craftsky" android:host="auth" />
</intent-filter>
```

No `android:autoVerify`, no SHA256 fingerprint, no `assetlinks.json`. Works identically on debug and release builds.

**Smoke test:** `adb shell am start -W -a android.intent.action.VIEW -d 'craftsky://auth/complete?token=testtoken' social.craftsky.app`.

### 7.3 Dev setup notes (captured in `app/README.md`)

- `flutter run` with no `--dart-define` uses `http://10.0.2.2:8080` (Android-emulator-friendly default).
- iOS simulator users should pass `--dart-define CRAFTSKY_API_BASE_URL=http://localhost:8080`.
- `just run-app` (or similar) recipes may be added to the root `justfile` wrapping the right defines per platform.

## 8. Testing

Three test layers, matching the existing `app/test/` convention.

Tests follow the Riverpod 3 idioms codified in `.claude/rules/riverpod.md` (Testing section): `ProviderContainer.test()` per test body, `overrideWith(FakeNotifier.new)` for notifier seams, `overrideWith((ref) => fake)` or `overrideWithValue` for plain providers, and service-level fakes rather than notifier mocks wherever possible.

### 8.1 API client unit tests — `test/shared/api/`

Using `http_mock_adapter`:

**`craftsky_api_client_test.dart`** (session client):
- `login(handle)` → `POST /v1/auth/login` with body `{handle, handoff_mode: "deep_link"}`; no Bearer header (anonymous path).
- `whoami()` → `GET /v1/whoami` with `Authorization: Bearer <token>` derived from an overridden `authSessionProvider` (seeded to `SignedIn`).
- `logout()` → `POST /v1/auth/logout` with Bearer.
- `_SignOutOn401Interceptor`: 401 on `/v1/whoami` signs out — `authSessionProvider` transitions to `SignedOut`, `SecureTokenStorage.clear()` is called.
- `_ErrorMappingInterceptor` maps the four branches (`ApiUnauthorized`, `ApiBadRequest(code)`, `ApiServerError`, `ApiNetworkError`).
- `ApiBadRequest.code` defensively handles missing / non-string `error` field.

**`handoff_api_client_test.dart`**:
- `whoami()` composes `GET /v1/whoami` with `Authorization: Bearer <token>` baked into `BaseOptions.headers` (no interceptor dependency).
- 401 surfaces as `ApiUnauthorized` and **does NOT** trigger sign-out (the handoff Dio's interceptor stack contains only `_ErrorMappingInterceptor`, not `_SignOutOn401Interceptor`). Assert by wiring it up inside a `ProviderContainer.test()` and checking `authSessionProvider` stays unchanged.

### 8.2 Provider unit tests — `test/auth/providers/`

Using `ProviderContainer.test()`:

- `AuthSession.build` resolves to `SignedOut` when storage is empty.
- `AuthSession.build` resolves to `SignedIn` when storage has a stored session; `_validateInBackground` fires unawaited.
- Background validation: `ApiUnauthorized` → clear storage, state → `SignedOut`.
- Background validation: DID mismatch → clear + `SignedOut`.
- Background validation: handle drift → update storage + state.
- Background validation: network error → state unchanged.
- `AuthController.signIn` happy path (browser launch mocked by overriding `launchAuthUrlProvider`), pending-auth stamped, state transitions `idle → loading → data`.
- `AuthController.signIn` trims whitespace + leading `@` from handle.
- `AuthController.signIn` empty handle → `AsyncError(HandleRequired)`.
- `AuthController.completeFromDeepLink` happy path: builds `handoffApiClientProvider(token)` (overridden in the test with a fake), calls `whoami`, writes storage ONCE with full `{token, did, handle}`, transitions auth state, clears `pendingAuthProvider` in the finally block.
- `AuthController.completeFromDeepLink` with no pending auth → `AsyncError(NoPendingSignIn)`.
- `AuthController.completeFromDeepLink` with stale pending (>10min) → `AsyncError(SignInTimedOut)`, pending cleared.
- `AuthController.completeFromDeepLink` with `whoami` failure → storage NOT written; no session leaks to `SecureTokenStorage`.
- Kill-during-sign-in simulation: run `completeFromDeepLink` against a handoff client that never completes; assert `SecureTokenStorage.read()` still returns `null`. Next cold start → `SignedOut`.
- `AuthController.signOut` happy path.
- `AuthController.signOut` on network failure still clears local state.
- `OnboardingStatus(did).build` reads `SharedPreferences`.
- `OnboardingStatus(did).finish` writes `SharedPreferences` + updates state.

### 8.3 Widget tests — `test/auth/pages/`

- `SignInPage`: entering a handle + tapping "Continue" dispatches `AuthController.signIn` with the trimmed handle. Error states render mapped messages (`HandleRequired`, `ServerUnavailable`, generic).
- `AuthCompletePage`: on mount, calls `completeFromDeepLink` with the route's `token` param. `AsyncError(AuthError)` renders a retry tile. `AsyncData(null)` after the transition completes relies on router redirect to move user off the page — widget test asserts the call happened; router integration is asserted at the router-test level.
- `SignOutTile` (in Settings): tap dispatches `signOut`. Asserts the call happened; router behavior is not in scope here.

### 8.4 Router tests — `test/router/router_test.dart`

- Redirect: `SignedOut` + location `/feed` → redirects to `/welcome`.
- Redirect: `SignedIn` + `!onboarded` + location `/feed` → redirects to `/onboarding`.
- Redirect: `SignedIn` + `onboarded` + location `/welcome` → redirects to `/feed`.
- Redirect: `/auth/complete` + `SignedOut` → no redirect (user stays on spinner while controller works).
- Redirect: `/auth/complete` + `SignedIn` + onboarded → `/feed`.
- Redirect: `/auth/complete` + `SignedIn` + not onboarded → `/onboarding`.
- Refresh listenable fires on `authSessionProvider` transitions.
- Refresh listenable fires when onboarding flips for the currently-signed-in DID (re-attach-on-DID-change helper).

### 8.5 Not in scope

- Integration tests via `flutter drive` (deep-link injection, real browser launch). Deferred until a CI harness exists. Manual smoke tests in §9 cover the gap for v1.
- End-to-end tests against a real PDS.

## 9. Acceptance

The spec is implemented when:

1. **Manual smoke tests pass on both iOS simulator and Android emulator:**
   - Fresh install → enter handle → complete sign-in in browser → deep link returns → feed loads.
   - Kill app → relaunch → still signed in.
   - Airplane mode → relaunch → still signed in (optimistic; `whoami` fails quietly in background).
   - Settings → Sign out → welcome page. Relaunch → still signed out.
   - Mid-session revocation: while signed in, manually `UPDATE craftsky_sessions SET revoked_at = now()` in the dev Postgres. Trigger any authenticated call (e.g. pull-to-refresh). App transitions to `/welcome` cleanly without a crash.
   - Enter an empty handle → `HandleRequired` message.
   - Force a server 502 (e.g., docker-compose down) → `ServerUnavailable` message.
   - `xcrun simctl openurl` / `adb shell am start` fire a manual deep link → lands on `AuthCompletePage` (which shows `NoPendingSignIn` since no sign-in is in progress, correctly).
2. **Automated tests pass:**
   - `flutter test` green with the new test files.
   - `dart analyze` clean.
   - `dart run build_runner build --delete-conflicting-outputs` clean.
3. **Code health:**
   - Both `kDebugMode ? true : false` ternaries are removed.
   - `authStatusProvider` file is deleted (not kept as a compatibility shim).
   - No `ref.watch` inside the router redirect.
   - No `GoRouterRefreshStream` or similar stream-bridge; the refresh mechanism is the `ChangeNotifier` owned by the `goRouter` provider.
   - No `print` / `debugPrint` — all diagnostics go through `logging`.
4. **Documentation:**
   - `app/README.md` has "Deep links" + "Dev setup" sections.
   - `AGENTS.md` rule #2 already describes the Flutter-never-holds-PDS-tokens invariant correctly; no rewrite needed.

## 10. Risks

1. **Token-in-URL exposure.** `craftsky://auth/complete?token=...` lands in any process-level logging of incoming intents. **Mitigation:** the `logging` package call-sites in `AuthCompletePage.initState` / `AuthController.completeFromDeepLink` must redact the `token` query parameter — log the URI with the query string replaced by `<redacted>`. Unit-assertable via the `logging` package's `Logger.root.onRecord` stream.

2. **Pending-auth staleness.** A user could kick off sign-in, background the app for hours, then the deep link fires late. The server's `oauth_auth_requests` row (30-min TTL) may be gone; the Craftsky token would still be valid, but the UX is confusing. **Mitigation:** 10-minute client-side window enforced in `completeFromDeepLink` (§3.4). User sees `SignInTimedOut` → "Please sign in again."

3. **Release build with no base URL.** `kDebugMode` is false in release, and the default becomes `''`. The `StateError` throws at first API call, not startup. **Mitigation:** `bootstrap.dart` touches `dioProvider` once before `runApp`, failing fast with a clear error screen.

4. **`flutter_secure_storage` edge cases.** Android: user removes device lock screen → keystore-backed reads can fail; OTA updates can corrupt the keystore. iOS: no known issues. **Mitigation:** `SecureTokenStorage.load` catches `PlatformException` and treats it as "no session" (→ `SignedOut`). Write failures bubble up through `AuthController.completeFromDeepLink` and surface as `StorageFailure`.

5. **`/v1` migration timing.** If the server migration doesn't land before this, the Flutter client's hardcoded `/v1/...` paths 404. **Mitigation:** Cross-reference note in both specs; whoever lands second does the rename (server-side is three route lines; client-side is three path constants). No compat shim — both sides flip together.

6. **DID spoofing via storage corruption.** An attacker with filesystem access could swap the stored `did` for another user's. On relaunch the app optimistically trusts it, shows their profile / feed until `whoami` returns. **Mitigation:** the DID-mismatch check in `_validateInBackground` clears state if the token → DID mapping disagrees. The window is one round-trip on app launch — tolerable for a crafting app. App-layer encryption of the storage blob is future work if the threat model tightens. The separate handoff Dio pattern in `completeFromDeepLink` (§3.4 + §5.2) closes a related footgun: we never persist a `StoredSession` until `whoami` has resolved the real DID, so there's no "half-written session" attack surface from an app-kill during the handoff window.

## 11. Future work

Explicitly out of scope for v1. Discoverable here.

1. **HTTPS Universal Links / App Links** at `app.craftsky.social` (or whichever brand domain is chosen). Introduces AASA + `assetlinks.json` hosting. Motivated by content-sharing deep links (`/profile/:handle`, `/post/:uri`). Auth handoff can migrate concurrently or remain custom-scheme indefinitely — no coupling.
2. **"Sign out everywhere."** UI toggle → `POST /v1/auth/logout?all=true`. Needs a confirmation dialog.
3. **"Active sessions" listing** (profile/settings → "This device + 2 others, sign out from…"). Depends on a new server endpoint + UI.
4. **Server-side onboarding state.** Once onboarding collects real data, move the flag server-side so it survives reinstalls cleanly.
5. **Account creation (`createAccount`)** via the AppView against Bluesky's entryway or a craftsky-owned PDS. The welcome page's "Create account on a PDS" currently falls through to sign-in.
6. **Deferred deep link fallback page.** AppView callback template can render "Download Craftsky — [App Store] [Play Store]" when the OS fails to open `craftsky://`. Small server template change.
7. **Web platform.** Requires either a webview-based OAuth flow or a different handoff (postMessage / cookie-based session). Its own spec.
8. **Desktop platform.** Would use the existing `loopback` handoff mode the server already supports (same path the CLI will use). Its own spec.
9. **Token rotation / refresh.** The server currently issues one bearer token per device that lives until explicitly revoked or the underlying OAuth session dies (see §3.3). If a short-lived-token + refresh pattern is introduced later (most likely alongside the TMB upgrade in the OAuth BFF spec), `AuthController` grows a `refresh()` method and `_ErrorMappingInterceptor` grows refresh-on-401-then-retry logic. Until then, 401 means "sign in again" — no client-side refresh to add.
10. **Write-proxy wiring.** Once the `POST /v1/xrpc/com.atproto.repo.createRecord` endpoint exists, add a typed method to `CraftskyApiClient`. Its own spec.
11. **Integration testing** of the deep-link round-trip via `flutter drive` + simulator URL injection.
