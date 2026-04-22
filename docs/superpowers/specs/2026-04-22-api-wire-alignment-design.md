# API Wire Alignment (v1) — design

**Date:** 2026-04-22
**Status:** proposed
**Scope:** Align the HTTP wire protocol between the Flutter client and the AppView so the OAuth sign-in flow works end-to-end. Codifies camelCase JSON, fixes `/v1/whoami`, and resolves the `X-Craftsky-Device-Id` contract divergence.

## Summary

The Flutter client and the AppView server were specified separately ([2026-04-18-appview-oauth-bff-design.md](./2026-04-18-appview-oauth-bff-design.md), [2026-04-21-flutter-auth-design.md](./2026-04-21-flutter-auth-design.md), [2026-04-21-appview-api-architecture-design.md](./2026-04-21-appview-api-architecture-design.md)) and have drifted in three places that together prevent sign-in from completing. This spec closes every client/server contract divergence on the v1 surface:

1. Codifies **camelCase** as the project-wide JSON key convention and stands up `appview/internal/api/envelope/` with helpers that enforce it.
2. Changes `/v1/whoami` to return `{did, handle}`; handle is resolved on every call via the indigo identity directory; directory failures return 502 `identity_unavailable`.
3. Implements `X-Craftsky-Device-Id` in full — server middleware enforces the header on authenticated `/v1/*` routes (400 on missing/malformed), client generates + persists a UUID and attaches it on every call.
4. Defines an end-to-end smoke-test checklist that, when it passes against `just dev`, is the acceptance bar.

## Goals

1. Make the client-facing JSON contract internally consistent so the Flutter app's existing `LoginResponse` / `WhoAmI` / `ApiException` code stops needing compensating annotations.
2. Unblock the Flutter OAuth flow by having `/v1/whoami` return the handle the client already expects.
3. Close the "contract exists on paper only" gap for `X-Craftsky-Device-Id` so future features (active-sessions UI, push notifications, per-device rate limiting) don't require a retrofit.
4. Give implementers a concrete acceptance bar so "done" is not subjective.

## Non-goals

- **XRPC or `/oauth/*` renaming.** Those surfaces are governed by the atproto OAuth spec; we do not touch their JSON shapes.
- **Rate limiting, CORS, observability, blob upload, success-response envelope.** Already tracked as future work in the API architecture spec; this spec does not change their status.
- **Token rotation / TMB upgrade.** Owned by the OAuth BFF spec's §6.
- **`flutter_drive` integration tests.** Covered by manual smoke test (§5).
- **Server-side handle caching or fallback.** `/v1/whoami` does a fresh directory lookup on every call; no stored-handle column, no "last known good" fallback.
- **Active-sessions UI, per-device rate limiting, push token registration.** `X-Craftsky-Device-Id` is plumbing-only in v1 — the column is populated but no feature consumes it yet.
- **Device-ID validation stricter than "ASCII identifier chars, ≤128 length."** No UUID-format pinning; no IP/user-agent fingerprinting.

## 1. camelCase convention + envelope package

### 1.1 The rule

Every JSON body the AppView emits or consumes under `/v1/*` uses camelCase keys. No exceptions within the Craftsky API surface. The `/oauth/*` surface keeps whatever the atproto OAuth spec dictates (we do not own those names). Struct tags in Go source carry the convention (`json:"authUrl"`); enforcement is by review + tests, not by a runtime field-name transform.

The API architecture spec ([2026-04-21-appview-api-architecture-design.md](./2026-04-21-appview-api-architecture-design.md)) gets a short errata note at the top of §6 pinning the casing explicitly. AGENTS.md's "Coding Conventions" block grows one line pointing at this spec.

### 1.2 `appview/internal/api/envelope/` — new package

Four files:

- **`envelope.go`** — the two public write helpers. These are the *only* sanctioned way for a `/v1/*` handler to emit a JSON response:
  ```go
  // WriteJSON encodes v as JSON with the standard Content-Type header.
  // v's struct tags must already be camelCase — WriteJSON does not
  // transform field names.
  func WriteJSON(w http.ResponseWriter, status int, v any)

  // WriteError writes the standard error envelope:
  // {"error": code, "message": msg, "requestId": rid}
  // rid is pulled from request context; falls back to a fresh ULID if
  // the context has no request-ID (the middleware should always have
  // populated it, but WriteError must be safe to call regardless).
  func WriteError(w http.ResponseWriter, r *http.Request, status int, code, msg string)

  // WriteValidationError adds a "fields" sibling (map[string]string)
  // to the error envelope for 422s.
  func WriteValidationError(w http.ResponseWriter, r *http.Request, msg string, fields map[string]string)
  ```

- **`request_id.go`** — request-ID middleware and context helpers:
  ```go
  func WithRequestID(next http.Handler) http.Handler  // middleware
  func RequestIDFrom(ctx context.Context) string       // "" if absent
  ```
  The middleware generates a ULID via `github.com/oklog/ulid/v2`, writes it into request context via a package-internal key, and sets the `X-Request-Id` response header. If a `X-Request-Id` header is already present on the incoming request, it is preserved (future-work for client-supplied trace IDs); otherwise a fresh ULID is generated.

- **`envelope_test.go`** — unit tests for both helpers.

- **`doc.go`** — package comment describing the contract and pointing at this spec.

### 1.3 Refactor existing handlers

All existing handlers that emit JSON under `/v1/*` switch to the envelope helpers:

- `appview/internal/auth/handlers_session.go`:
  - `writeJSONError` (the current local helper) is deleted; callers switch to `envelope.WriteError`.
  - `loginResponse` struct tag becomes `AuthURL string \`json:"authUrl"\``.
  - `loginRequest` struct tags become `HandoffMode \`json:"handoffMode"\``, `LoopbackRedirectURI \`json:"loopbackRedirectUri"\``. `Handle` is already camelCase-equivalent (single word).
- `appview/internal/api/whoami.go` — switches to `envelope.WriteJSON(w, 200, WhoAmIResponse{...})` (the handle-resolution change is §2).
- OAuth HTML error page in `handlers_render.go` is untouched (it is `text/html`, not JSON).
- `recordHandoff` / `loadHandoff` SQL is **unaffected** — the `handoff_mode` column name stays snake_case (SQL convention); only the wire JSON renames.
- `oauth_sessions.data` / `oauth_auth_requests.data` JSONB blobs are indigo-owned and opaque; no change.

### 1.4 Client-side alignment

With the server emitting camelCase:

- `app/lib/shared/api/models/login_response.dart` drops the `@MappableClass(caseStyle: CaseStyle.snakeCase)` override. Plain `@MappableClass()` is correct once the server emits `authUrl`.
- `app/lib/shared/api/craftsky_api_client.dart` `login` sends `{handle, handoffMode: 'deep_link'}` (instead of `handoff_mode`).
- `WhoAmI` model is already camelCase; no change.
- `ApiBadRequest` in `app/lib/shared/api/api_exception.dart` continues to read `response.data['error']` — already camelCase-equivalent. The `requestId` field in the response envelope is **not** wired into `ApiException` in v1 (no subclass gains a `requestId` field; the UI does not surface it). Adding it is a trivial future change when the first support workflow needs it; leaving it unwired now keeps the `ApiException` hierarchy narrow.

### 1.5 Tests

- `envelope/envelope_test.go` — status code, Content-Type, body shape (including `requestId` presence and camelCase keys); request-ID fallback when context is empty.
- `envelope/request_id_test.go` — ULID generated when absent; existing incoming `X-Request-Id` preserved; response header set.
- Existing `appview/internal/auth/handlers_test.go` assertions update field names (`authUrl`, `handoffMode`).
- **Contract test** — one new test in `appview/internal/routes/routes_test.go` (or a new file alongside) that hits a representative set of `/v1/*` error-emitting routes and asserts every response body is exactly the envelope shape (`error`, `message`, `requestId`, optional `fields`) with camelCase keys. Uses `json.RawMessage` decode + key inspection rather than a strongly-typed decode so unknown keys fail the test.

### 1.6 What this section does not do

- Does not add a struct-tag linter. Convention is enforced by review + the contract test.
- Does not version-bump the API. The API is pre-release; we change the shape and move on.
- Does not standardize success-response wrappers. Still bare bodies per the API architecture spec's open question #1.

## 2. `/v1/whoami` returns `{did, handle}`

### 2.1 Contract

```
GET /v1/whoami
Authorization: Bearer <craftsky-token>
X-Craftsky-Device-Id: <uuid>

200 OK
{"did": "did:plc:xyz...", "handle": "alice.craftsky.social"}

502 Bad Gateway
{"error": "identity_unavailable", "message": "...", "requestId": "..."}
```

The handle is resolved on every call via the indigo identity directory. No server-side caching (beyond whatever indigo's directory provides internally). No fallback to a stored handle — there is no stored handle.

### 2.2 Implementation

**Handler signature:**

```go
type WhoAmIResponse struct {
    DID    string `json:"did"`
    Handle string `json:"handle"`
}

type HandleResolver interface {
    ResolveHandle(ctx context.Context, did string) (string, error)
}

func WhoAmIHandler(resolver HandleResolver) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        did, ok := middleware.GetDID(r.Context())
        if !ok {
            envelope.WriteError(w, r, http.StatusInternalServerError, "internal_error", "no did in context")
            return
        }
        handle, err := resolver.ResolveHandle(r.Context(), did)
        if err != nil {
            envelope.WriteError(w, r, http.StatusBadGateway, "identity_unavailable", "could not resolve handle for did")
            return
        }
        envelope.WriteJSON(w, http.StatusOK, WhoAmIResponse{DID: did, Handle: handle})
    })
}
```

**`HandleResolver` implementation** lives next to `whoami.go` as `handle_resolver.go`. It wraps an `identity.Directory`:

1. Parse the DID via `syntax.ParseDID`.
2. Call `Directory.LookupDID(ctx, parsedDID)`.
3. Return `identity.Handle.String()` if non-empty; error otherwise.

The `identity.Directory` instance is hoisted into `Deps` from wherever the OAuth wiring currently constructs it, or constructed fresh alongside OAuth setup if it is currently encapsulated inside indigo's `oauth.ClientApp`. Implementation-plan decision.

**Timeout.** The resolver call inherits the request context; if no global handler timeout exists, the resolver wraps the directory call in `context.WithTimeout(ctx, 5*time.Second)`. Verify during implementation — the concern is `LookupDID` blocking indefinitely on PLC directory outage, not latency budget.

### 2.3 Error-path details

All failure modes collapse to 502 `identity_unavailable`:

- **DID doesn't exist at the PLC directory** — from the client's perspective, indistinguishable from "directory is down." Treating them identically keeps client logic simple.
- **DID exists but has no handle** (deactivated / in-migration state) — same reasoning; no leaked distinction in v1.
- **Timeout or network error** — same.

A **500 `internal_error`** is reserved for the routing-bug case where the middleware didn't populate the DID in context. This should be unreachable. The `internal_error` code is the generic bucket for unreachable-but-must-return states across the surface.

### 2.4 Client-side

No code change. `WhoAmI` model is already `{did, handle}`; `CraftskyApiClient.whoami` already decodes that shape; `_validateInBackground` already tolerates 502 by keeping cached `SignedIn` state (per Flutter auth spec §3.1). The client starts working as soon as the server ships.

### 2.5 Tests

- **`whoami_test.go`:**
  - Happy path: resolver returns handle → 200 + camelCase `{did, handle}`.
  - Resolver returns error → 502 + `{error: "identity_unavailable", ...}` with `requestId` present.
  - No DID in context → 500 `internal`.
- **`handle_resolver_test.go`:** unit tests against a fake `identity.Directory`:
  - Known DID → returns handle.
  - Unknown DID → returns error.
  - DID with empty handle → returns error.
  - Timeout → returns error (fake blocks until context expires).

## 3. `X-Craftsky-Device-Id` enforcement

### 3.1 Why

The API architecture spec §3.1 made the device-ID header required; neither side currently enforces it. With no app build in the wild, there is no coordinated-release cost to strict enforcement — server and client can ship together, and the "false positive locks users out" concern is hypothetical. Strict is what the spec says; strict is what we implement.

### 3.2 Server — middleware + migration

**Migration.** `ALTER TABLE craftsky_sessions ADD COLUMN last_device_id TEXT;` (up) / `DROP COLUMN last_device_id;` (down). Number is whichever is next in `appview/migrations/` at implementation time. The API architecture spec §7.3 already describes this migration; this spec is where it actually lands.

**Middleware** `appview/internal/middleware/device_id.go`:

```go
const deviceIDHeader = "X-Craftsky-Device-Id"

var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func DeviceID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get(deviceIDHeader)
        if id == "" {
            envelope.WriteError(w, r, http.StatusBadRequest, "missing_device_id", "X-Craftsky-Device-Id header is required")
            return
        }
        if !deviceIDPattern.MatchString(id) {
            envelope.WriteError(w, r, http.StatusBadRequest, "invalid_device_id", "X-Craftsky-Device-Id is malformed")
            return
        }
        r = r.WithContext(ctxkeys.WithDeviceID(r.Context(), id))
        next.ServeHTTP(w, r)
    })
}
```

The regex is deliberately permissive — accepts UUIDs (with or without dashes), ULIDs, any ASCII identifier up to 128 chars. Does not pin UUIDv4.

**ctxkeys** grows `WithDeviceID(ctx, string) ctx` / `DeviceIDFrom(ctx) (string, bool)`, mirroring the existing DID helper pattern.

**Session-store update.** `CraftskySessionStore` grows `UpdateLastDeviceID(ctx, tokenHash []byte, deviceID string) error`. Called opportunistically alongside the existing `last_seen_at` update, sharing the same 5-minute throttle — no double write-rate. The existing throttle tracks `last_seen_at` only; it extends to cover both columns together (one UPDATE writes both). This means a device-ID change *between* throttle windows is not persisted until the next window. Acceptable for v1 — the only observable consequence is a small lag on the (not-yet-built) active-sessions UI.

**Route wiring.** In `routes.go`, authenticated `/v1/*` routes compose `deviceID(authN(handler))`. Order matters: `authN` first so we know the session exists before we record a device-ID update; `deviceID` wraps outside so the 400 for missing header fires before we bother loading the session.

**Exemptions.** The middleware applies to **authenticated `/v1/*` routes only**:
- `POST /v1/auth/login` is exempt — pre-auth, and a future CLI or curl-based tool should be able to hit it without synthesizing a device-ID.
- `/oauth/*` routes are exempt — atproto-spec-governed, not our surface.
- `/health`, `/healthz` are exempt — ops surface.

### 3.3 Client — generation, persistence, attachment

**Device-ID provider** `app/lib/shared/device/device_id_provider.dart`:

```dart
@Riverpod(keepAlive: true)
Future<String> deviceId(Ref ref) async {
  final storage = ref.watch(secureStorageProvider);
  const key = 'craftsky_device_id';
  final existing = await storage.read(key: key);
  if (existing != null && existing.isNotEmpty) return existing;
  final fresh = const Uuid().v4();
  await storage.write(key: key, value: fresh);
  return fresh;
}
```

- Stored under a **separate** secure-storage key from `craftsky_session`. Sign-out clears the session; device-ID survives.
- Generated via the `uuid` package (new dep in `pubspec.yaml`).
- `secureStorageProvider` is a small keep-alive `Provider` that returns the shared `FlutterSecureStorage` instance. If one already exists (it's referenced by `SecureTokenStorage`), we reuse it; otherwise we extract it into its own provider so both `SecureTokenStorage` and the device-ID provider can share.

**Interceptor.** `_SessionAuthInterceptor` grows a second header attachment. Async read of the device-ID is fine — Dio supports async interceptors, and the keep-alive provider caches after first resolution:

```dart
@override
Future<void> onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
  final deviceId = await _ref.read(deviceIdProvider.future);
  options.headers['X-Craftsky-Device-Id'] = deviceId;

  if (!_anonymousPaths.contains(options.path)) {
    final auth = _ref.read(authSessionProvider).value;
    if (auth is SignedIn) {
      options.headers['Authorization'] = 'Bearer ${auth.token}';
    }
  }
  handler.next(options);
}
```

Device-ID is attached on **every** request including anonymous paths (`/v1/auth/login`). Rationale: the device-ID is an identity-of-the-app-install signal, not identity-of-the-user. Sending on anonymous calls is free and lets future rate-limiting key on `(device_id, endpoint)` without a subsequent protocol change. The server does not enforce device-ID on `/v1/auth/login` (§3.2), but sending it is harmless.

**`HandoffApiClient`.** The handoff Dio bakes Bearer + device-ID into `BaseOptions.headers`. To keep the handoff provider synchronous, `AuthController.completeFromDeepLink` pre-resolves the device-ID via `await ref.read(deviceIdProvider.future)` and passes it into the provider family as a second parameter:

```dart
@riverpod
HandoffApiClient handoffApiClient(Ref ref, String token, String deviceId) {
  final dio = Dio(_baseOptions().copyWith(
    headers: {
      ..._baseOptions().headers ?? const <String, dynamic>{},
      'Authorization': 'Bearer $token',
      'X-Craftsky-Device-Id': deviceId,
    },
  ));
  dio.interceptors.add(_ErrorMappingInterceptor());
  return HandoffApiClient(dio);
}
```

The two-parameter family adds a small type-level safety property: you cannot construct a handoff client without explicitly passing a device-ID.

**Bootstrap ordering.** `bootstrap.dart` touches `deviceIdProvider` before `runApp`, alongside the existing `dioProvider` touch. This is **correctness-critical under strict enforcement**: the first `/v1/*` call must not fire before the device-ID future resolves, or the server will 400. Eagerly awaiting the provider in bootstrap guarantees this.

Note this does **not** contradict the Flutter auth spec's §3.2 "bootstrap.dart requires no special handling" remark — that was scoped to `SecureTokenStorage` for the session blob, whose read is deferred into `authSessionProvider.build` and can resolve lazily. The device-ID provider is a new bootstrap-time dependency specific to this spec; the existing secure-storage minimalism for the session blob is unchanged.

### 3.4 Tests

**Server:**
- Middleware test: missing header → 400 `missing_device_id`; empty → 400 `missing_device_id`; malformed (spaces, non-ASCII, too long) → 400 `invalid_device_id`; valid → request proceeds, context populated.
- Integration: authenticated `/v1/whoami` call with valid device-ID → 200, `craftsky_sessions.last_device_id` populated (respecting throttle).
- Route-wiring test: `/v1/auth/login` accepts requests without device-ID header.

**Client:**
- Device-ID provider: missing in storage → generates UUIDv4, writes, returns; present → returns existing, no write.
- Interceptor: attaches `X-Craftsky-Device-Id` on both authenticated and anonymous paths.
- Handoff client: both Bearer and device-ID baked into `BaseOptions`.
- Sign-out: `SecureTokenStorage.clear()` does not touch `craftsky_device_id` key. Assert by reading the key after sign-out.
- Bootstrap ordering: device-ID provider is resolved before the first `authSessionProvider` validation fires. (Widget-test level; overrides the provider and asserts the read happened before any mock API call.)

### 3.5 What this section does NOT do

- Does not expose device-ID to feature handlers. It is in context; nothing reads it in v1. Plumbing-only.
- Does not add an index on `last_device_id`. Wait until active-sessions UI lands.
- Does not define `device_label` population. That column exists from the OAuth BFF spec; populating it is future work.
- Does not implement per-device rate limiting, active-sessions UI, or push routing. All future specs.

## 4. Data model changes

Single migration, summarizing §3.2:

```
appview/migrations/NNNNNN_craftsky_sessions_device_id.up.sql
appview/migrations/NNNNNN_craftsky_sessions_device_id.down.sql
```

```sql
-- up
ALTER TABLE craftsky_sessions
  ADD COLUMN last_device_id TEXT;

-- down
ALTER TABLE craftsky_sessions
  DROP COLUMN last_device_id;
```

Implementer verifies the current highest migration number before committing and numbers accordingly. No other schema changes.

## 5. Acceptance — end-to-end smoke test

### 5.1 Pre-conditions

1. `just dev` brings up postgres + migrate + tap + appview cleanly.
2. Flutter app builds on at least one of (iOS simulator, Android emulator). Android emulator is preferred — uses the existing `http://10.0.2.2:8080` default, no `--dart-define` needed.
3. A test atproto account on a reachable PDS (public Bluesky works).
4. No existing `craftsky_sessions` row for the test DID (or the test starts with `DELETE FROM craftsky_sessions WHERE account_did = $1`).

### 5.2 Smoke test steps

Each step has an explicit pass criterion.

1. **Fresh sign-in.** Launch → `/welcome` → enter handle → "Continue" → browser PDS auth → `craftsky://auth/complete?token=...` → lands on `/onboarding` or `/feed`. **Pass:** post-auth screen, no snackbar, no crash.

2. **Server log inspection during step 1.** **Pass:** `POST /v1/auth/login` returns 200 (not 400); `GET /v1/whoami` returns 200 (not 400 or 502). Asserted by inspecting the appview container's request log lines (status + path). If the request log format does not currently include status codes, the smoke test equivalently passes if no `envelope.WriteError` callsite is logged during the sign-in flow — verify the log format at implementation time and adjust the assertion wording.

3. **JSON casing via curl.** `curl -v http://localhost:8080/v1/auth/login -H 'Content-Type: application/json' -H 'X-Craftsky-Device-Id: smoke-test' -d '{"handle":"","handoffMode":"deep_link"}'`. **Pass:** response body is exactly `{"error":"handle_required","message":"...","requestId":"..."}` with all four camelCase keys; `requestId` non-empty; Content-Type `application/json`.

4. **`whoami` shape.** `curl http://localhost:8080/v1/whoami -H 'Authorization: Bearer <token-from-step-1>' -H 'X-Craftsky-Device-Id: smoke-test'`. **Pass:** 200 with `{"did":"did:plc:...","handle":"..."}`; handle matches test account's PDS handle.

5. **`whoami` directory failure.** Covered by unit tests (§2.5), not manually smoked.

6. **Device-ID enforcement.**
   - `curl http://localhost:8080/v1/whoami -H 'Authorization: Bearer <token>'` (no device-ID). **Pass:** 400 `missing_device_id`.
   - Same with `-H 'X-Craftsky-Device-Id: '` (empty). **Pass:** 400 `missing_device_id`.
   - Same with `-H 'X-Craftsky-Device-Id: has spaces'` (malformed — contains disallowed chars). **Pass:** 400 `invalid_device_id`.
   - Same with `-H 'X-Craftsky-Device-Id: <129 consecutive a's>'` (malformed — exceeds max length). **Pass:** 400 `invalid_device_id`.

7. **Persistent sign-in.** Kill app. Relaunch. **Pass:** lands on `/feed` or `/onboarding`, not `/welcome`.

8. **Background validation.** Step 7 + tail appview logs. **Pass:** exactly one `GET /v1/whoami` fires during cold start; `X-Craftsky-Device-Id` present on the call.

9. **Sign out.** Settings → "Sign out". **Pass:** returns to `/welcome`. Relaunch → still `/welcome`.

10. **Mid-session revocation.** Sign back in. `UPDATE craftsky_sessions SET revoked_at = now() WHERE account_did = '<test-did>';` Trigger any authenticated call. **Pass:** app transitions to `/welcome` without crashing; `SecureTokenStorage` cleared.

11. **Device-ID persistence across sign-out.** After step 9, inspect secure storage. **Pass:** `craftsky_session` cleared; `craftsky_device_id` retained. Re-sign-in reuses the same device-ID (assert via `last_device_id` column value in Postgres).

### 5.3 Done = all of:

- All 11 smoke-test steps pass against `just dev`.
- `just test` green (Go unit + integration).
- `flutter test` green.
- `dart analyze` clean.
- `dart run build_runner build --delete-conflicting-outputs` clean.
- AGENTS.md line pointing at this spec's camelCase rule is added.
- Errata note in API architecture spec §6 pinning camelCase is added.
- Single PR (or small stack) across `appview/`, `app/`, migrations, spec errata.

### 5.4 Automated coverage gaps accepted

- No `flutter drive` integration test. Manual smoke covers.
- No automated full-surface contract test. Envelope helpers + per-handler tests + the limited contract test in §1.5 are sufficient; full surface-walking waits for OpenAPI.
- No load test of the `whoami` directory lookup. At N=1 smoke is fine; production latency/rate-limit behaviour is future concern.

### 5.5 Roll-back

Pre-release, rollback is a revert PR. Migration has a `.down.sql`. No user data at risk.

Once the Flutter app ships to users, a server revert breaks any live build on the new client (snake_case server vs camelCase client, or strict-vs-tolerant device-ID enforcement). This is the normal coordinated-release property — not specific to this spec, and not a blocker pre-release.

## 6. Implementation map

### 6.1 Files touched

**Server — `appview/`:**

```
internal/api/envelope/              (new package)
├── envelope.go
├── request_id.go
├── envelope_test.go
├── request_id_test.go
└── doc.go

internal/api/
├── whoami.go                       (rewritten — handle + resolver)
├── whoami_test.go                  (updated assertions + 502 path)
├── handle_resolver.go              (new)
└── handle_resolver_test.go         (new)

internal/auth/
├── handlers_session.go             (snake → camel tags; delete writeJSONError)
├── handlers_render.go              (unchanged — HTML only)
├── handlers_test.go                (updated assertions)
└── craftsky_session.go             (UpdateLastDeviceID method)

internal/middleware/
├── device_id.go                    (new)
└── device_id_test.go               (new)

internal/ctxkeys/
└── ctxkeys.go                      (WithDeviceID / DeviceIDFrom)

internal/routes/
├── routes.go                       (wire envelope.WithRequestID + deviceID middleware)
└── routes_test.go                  (add contract test; update field-name assertions)

internal/app/
└── deps.go                         (hoist identity.Directory; inject resolver + device-id store method)

migrations/
├── NNNNNN_craftsky_sessions_device_id.up.sql
└── NNNNNN_craftsky_sessions_device_id.down.sql

queries/
└── craftsky_sessions.sql           (sqlc query for UpdateLastDeviceID; regenerate)
```

**Client — `app/`:**

```
lib/shared/api/
├── models/login_response.dart      (drop snakeCase override)
└── craftsky_api_client.dart        (handoff_mode → handoffMode)

lib/shared/api/providers/
├── session_auth_interceptor.dart   (attach X-Craftsky-Device-Id)
└── api_client_provider.dart        (handoff family: (token, deviceId))

lib/shared/device/                  (new)
├── device_id_provider.dart
├── device_id_provider.g.dart
└── secure_storage_provider.dart    (extract shared FlutterSecureStorage if not already)

lib/auth/providers/
└── auth_controller.dart            (pre-resolve deviceId before handoff client)

lib/bootstrap.dart                  (touch deviceIdProvider before runApp)

test/shared/api/
├── craftsky_api_client_test.dart   (device-ID header assertions; camelCase body)
├── handoff_api_client_test.dart    (device-ID baked into BaseOptions)
└── session_auth_interceptor_test.dart

test/shared/device/
└── device_id_provider_test.dart    (new)

test/auth/providers/
└── auth_controller_test.dart       (update handoff construction)

pubspec.yaml                        (add uuid)
```

**Docs:**

```
AGENTS.md                           (camelCase convention line)
docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md
                                    (§6 errata note pinning camelCase)
```

### 6.2 Dependency additions

**Go:**
- `github.com/oklog/ulid/v2` — for request-ID generation. If already transitively present, no `go mod tidy` churn.

**Dart:**
- `uuid` — for device-ID generation. Latest stable at implementation time.

### 6.3 Order of operations

The changes can land in one PR or a small stack. Suggested ordering for reviewability:

1. Envelope package + request-ID middleware + contract test (no behavioural change).
2. Refactor existing auth handlers to use envelope; flip JSON tags to camelCase.
3. Flutter: drop `snakeCase` override on `LoginResponse`, update `handoff_mode` → `handoffMode`. At this checkpoint, the client and server agree on casing.
4. `whoami` handle resolution (server + handler tests). No client change needed; client was already expecting this shape.
5. Migration + device-ID middleware + session-store update + route wiring.
6. Flutter: device-ID provider + interceptor + bootstrap ordering.
7. Smoke test.

Steps 1–4 on their own are sufficient for sign-in to work end-to-end (device-ID is strict, so step 5–6 must land together, not on top of a working-but-incomplete server). If staged as a stack, 1–4 land first, then 5+6 together.

## 7. Risks

1. **Directory-lookup latency on `/v1/whoami`.** Every background validation on cold start pays one PLC lookup. If indigo's directory has no internal cache, or the cache is cold, this is tens to low-hundreds of ms per launch. Tolerable pre-release; add a measurement during smoke test and revisit if it shows up as a real cold-start delay. Mitigation path: per-process LRU cache in `HandleResolver` with a short TTL (60s) — small PR.

2. **Directory outage.** PLC directory down → every `/v1/whoami` returns 502. Signed-in users keep working (their session token is valid); only background validation fails. Flutter client tolerates this correctly by design. Mitigation not needed.

3. **Device-ID secure-storage failure on Android.** Keystore-backed reads can fail (device-lock removed, OTA corruption). The existing `SecureTokenStorage` handles this by catching `PlatformException`; the device-ID provider must do the same. On failure we generate a new UUID in memory and attempt to persist; if persistence fails, we send the in-memory value and accept that a future launch may generate a different one. Non-fatal — device-ID is not a security primitive.

4. **Bootstrap ordering regression.** If a future refactor moves Dio construction or first API call ahead of the device-ID resolution, the very first authenticated request after install fails with 400. Mitigation: the bootstrap-ordering widget test asserts the order.

5. **Migration conflict.** Numbering clash if another migration lands between spec and implementation. Trivial to resolve — renumber. Implementer checks the current max before committing.

6. **Spec/implementation drift in the API architecture spec.** This spec adds an errata to the API architecture spec rather than rewriting it. Future readers must notice the errata. Mitigation: errata is at the top of the affected section (§6), not buried.

## 8. Future work

Explicitly out of scope. Discoverable here.

1. **OpenAPI / typed client generation.** Once the `/v1/*` surface stabilises. Replaces hand-maintained Dart models with generated ones.
2. **Struct-tag linter.** If camelCase drift becomes a repeated review issue, a small `go vet`-style check that rejects snake_case tags in `/v1/*` handlers. Probably not needed.
3. **Per-process handle cache** in `HandleResolver`. See §7.1.
4. **Active-sessions UI.** Reads `last_device_id` + `device_label`. Own spec.
5. **Per-device rate limiting.** Reads device-ID from context. Own spec.
6. **Push notification registration.** Attaches push token to `(did, device_id)`. Own spec.
7. **Success-response envelope.** Open question in API architecture spec §8.1.
8. **`/v1/auth/login` device-ID enforcement.** Currently exempted; revisit if unauth-rate-limiting becomes a goal.
9. **Full-surface contract test.** Walk every route; assert envelope shape on every error. Waits for route enumeration to be automatic (OpenAPI).
