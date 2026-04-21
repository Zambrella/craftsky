# Flutter Auth (v1) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the stub bool `authStatusProvider` with a real atproto-OAuth-backed session flow: handle → system browser → PDS auth → custom-scheme deep link back into the app → persisted Craftsky bearer token → optimistic cold start with background `whoami` validation. Mobile-only (iOS + Android).

**Architecture:** Flutter client talks only to the AppView's `/v1/auth/login`, `/v1/auth/logout`, `/v1/whoami` endpoints via a `dio`-based `CraftskyApiClient`. An opaque bearer token minted by the AppView at the end of the OAuth callback is delivered back to the app via `craftsky://auth/complete?token=…`, intercepted by go_router's `AuthCompleteRoute`, and persisted to `flutter_secure_storage`. `AuthSession` exposes `AsyncValue<AuthState>` (sealed: `SignedOut | SignedIn`); `AuthController` owns signIn/completeFromDeepLink/signOut; a global dio interceptor centralises 401 → local sign-out.

**Tech Stack:** Flutter 3.x, Riverpod 3 (`@riverpod` codegen), `dio`, `flutter_secure_storage`, `url_launcher`, `dart_mappable`, `go_router`, `logging`, `http_mock_adapter` (test-only).

**Spec:** [docs/superpowers/specs/2026-04-21-flutter-auth-design.md](../specs/2026-04-21-flutter-auth-design.md)

---

## File Structure

### New files

```
app/lib/
├── shared/api/
│   ├── api_exception.dart                        # sealed ApiException + 4 subtypes
│   ├── craftsky_api_client.dart                  # login/whoami/logout wrappers
│   ├── models/
│   │   ├── login_response.dart                   # + .mapper.dart
│   │   └── whoami.dart                           # + .mapper.dart
│   └── providers/
│       ├── dio_provider.dart                     # + .g.dart — Dio w/ interceptors
│       ├── api_client_provider.dart              # + .g.dart
│       ├── auth_interceptor.dart                 # _AuthInterceptor (Bearer injection)
│       └── error_mapping_interceptor.dart        # _ErrorMappingInterceptor + 401 sign-out
├── auth/
│   ├── models/
│   │   ├── auth_error.dart                       # sealed AuthError + 7 subtypes
│   │   ├── auth_state.dart                       # sealed AuthState + SignedIn/SignedOut
│   │   ├── pending_auth.dart                     # + .mapper.dart
│   │   └── stored_session.dart                   # + .mapper.dart
│   ├── providers/
│   │   ├── auth_controller.dart                  # + .g.dart — signIn / completeFromDeepLink / signOut
│   │   ├── auth_session_provider.dart            # + .g.dart — AsyncValue<AuthState>
│   │   ├── in_flight_token_provider.dart         # + .g.dart
│   │   ├── pending_auth_provider.dart            # + .g.dart
│   │   └── secure_token_storage.dart             # + .g.dart — async read/write wrapper
│   ├── pages/
│   │   └── auth_complete_page.dart               # NEW — deep-link landing ("Signing in…")
│   └── widgets/
│       └── auth_error_snack_bar_content.dart     # NEW — error → user-facing string
├── settings/
│   └── widgets/
│       └── sign_out_tile.dart                    # NEW — settings row
└── router/
    └── onboarding_refresh_listener.dart          # NEW — re-attach per-DID onboarding sub
```

### Modified files

- `app/pubspec.yaml` — add deps
- `app/lib/bootstrap.dart` — register new `dart_mappable` mappers (`initializeMappers`)
- `app/lib/router/router.dart` — redirect uses `authSessionProvider`; new `AuthCompleteRoute`; refresh listenable
- `app/lib/router/route_locations.dart` — `authComplete = '/auth/complete'`
- `app/lib/auth/pages/welcome_page.dart` — drop dev toggle
- `app/lib/auth/pages/sign_in_page.dart` — wire to `AuthController`
- `app/lib/onboarding/providers/onboarding_status_provider.dart` — family keyed by DID, SharedPreferences-backed
- `app/lib/onboarding/pages/onboarding_page.dart` — call finish() with did
- `app/lib/settings/pages/settings_page.dart` — use new `SignOutTile`
- `app/ios/Runner/Info.plist` — `CFBundleURLTypes` for scheme `craftsky`
- `app/android/app/src/main/AndroidManifest.xml` — intent-filter for `craftsky://auth`
- `app/README.md` — Deep links + Dev setup
- `app/test/fakes/auth_status_fakes.dart` → **rename** to `auth_session_fakes.dart` and rewrite

### Deleted files

- `app/lib/auth/providers/auth_status_provider.dart` + `.g.dart`

---

## Chunk 1: Dependencies, API client scaffold, error types

The goal of this chunk is to get `dio` installed, `CraftskyApiClient` wired with its two interceptors and error mapping, and every branch tested against `http_mock_adapter`. No auth logic yet — the client just takes a `Dio` and exposes typed methods.

### Task 1: Add dependencies to pubspec

**Files:**
- Modify: `app/pubspec.yaml`

- [ ] **Step 1: Edit `app/pubspec.yaml`** — add these under `dependencies:`:

```yaml
  dio: ^5.7.0
  flutter_secure_storage: ^9.2.2
  url_launcher: ^6.3.1
```

and under `dev_dependencies:`:

```yaml
  http_mock_adapter: ^0.6.1
```

- [ ] **Step 2: Run `flutter pub get` and confirm clean**

```bash
cd app && flutter pub get
```

Expected: no warnings about version conflicts. New entries appear in `pubspec.lock`.

- [ ] **Step 3: Commit**

```bash
git add app/pubspec.yaml app/pubspec.lock
git commit -m "feat(app): add dio, flutter_secure_storage, url_launcher, http_mock_adapter"
```

---

### Task 2: `ApiException` sealed class

**Files:**
- Create: `app/lib/shared/api/api_exception.dart`
- Create: `app/test/shared/api/api_exception_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/shared/api/api_exception_test.dart`:

```dart
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ApiException', () {
    test('ApiUnauthorized carries a stable message', () {
      expect(const ApiUnauthorized().message, 'unauthorized');
    });

    test('ApiBadRequest exposes the server error code when present', () {
      expect(const ApiBadRequest('handle_required').code, 'handle_required');
      expect(const ApiBadRequest('handle_required').message, 'handle_required');
    });

    test('ApiBadRequest falls back to "bad_request" when code is null', () {
      expect(const ApiBadRequest(null).message, 'bad_request');
      expect(const ApiBadRequest(null).code, isNull);
    });

    test('ApiServerError preserves the provided message', () {
      expect(const ApiServerError('boom').message, 'boom');
    });

    test('ApiNetworkError preserves the provided message', () {
      expect(const ApiNetworkError('offline').message, 'offline');
    });

    test('ApiException is exhaustive via switch', () {
      const values = <ApiException>[
        ApiUnauthorized(),
        ApiBadRequest('x'),
        ApiServerError('y'),
        ApiNetworkError('z'),
      ];
      for (final e in values) {
        final kind = switch (e) {
          ApiUnauthorized() => 'unauth',
          ApiBadRequest() => 'bad',
          ApiServerError() => 'srv',
          ApiNetworkError() => 'net',
        };
        expect(kind, isNotEmpty);
      }
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/shared/api/api_exception_test.dart
```

Expected: FAIL — "Target of URI doesn't exist".

- [ ] **Step 3: Implement** — `app/lib/shared/api/api_exception.dart`:

```dart
/// Errors surfaced by [CraftskyApiClient]. Sealed so call sites can
/// exhaustively switch.
sealed class ApiException implements Exception {
  const ApiException(this.message);
  final String message;
}

/// HTTP 401. The global 401 handler in `_ErrorMappingInterceptor`
/// signs the user out before this is rethrown to the caller.
final class ApiUnauthorized extends ApiException {
  const ApiUnauthorized() : super('unauthorized');
}

/// HTTP 4xx (non-401). [code] comes from the server's `{"error": "…"}`
/// body when present, else null.
final class ApiBadRequest extends ApiException {
  const ApiBadRequest(this.code) : super(code ?? 'bad_request');
  final String? code;
}

/// HTTP 5xx or any non-mapped error response.
final class ApiServerError extends ApiException {
  const ApiServerError(super.message);
}

/// Timeout, connection failure, or socket error. Distinct from
/// [ApiServerError] so the background `whoami` validation can
/// tolerate offline launches without signing the user out.
final class ApiNetworkError extends ApiException {
  const ApiNetworkError(super.message);
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/shared/api/api_exception_test.dart
```

Expected: PASS, 5 tests.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/api/api_exception.dart app/test/shared/api/api_exception_test.dart
git commit -m "feat(app): add ApiException sealed class for shared API errors"
```

---

### Task 3: Response models — `LoginResponse` and `WhoAmI`

**Files:**
- Create: `app/lib/shared/api/models/login_response.dart`
- Create: `app/lib/shared/api/models/whoami.dart`
- Create: `app/test/shared/api/models/login_response_test.dart`
- Create: `app/test/shared/api/models/whoami_test.dart`
- Modify: `app/lib/bootstrap.dart` (register mappers)

- [ ] **Step 1: Write the failing tests**

`app/test/shared/api/models/login_response_test.dart`:

```dart
import 'dart:convert';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('parses snake_case JSON from the server', () {
    const json = '{"auth_url":"https://pds.example.com/authorize?request_uri=x"}';
    final parsed = LoginResponseMapper.fromJson(json);
    expect(parsed.authUrl, 'https://pds.example.com/authorize?request_uri=x');
  });

  test('serialises back to snake_case JSON', () {
    const original = LoginResponse(authUrl: 'https://pds.example.com/a');
    final roundTrip = jsonDecode(original.toJson()) as Map<String, dynamic>;
    expect(roundTrip.keys.single, 'auth_url');
    expect(roundTrip['auth_url'], 'https://pds.example.com/a');
  });
}
```

`app/test/shared/api/models/whoami_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('WhoAmI parses did + handle from JSON', () {
    const json = '{"did":"did:plc:alice","handle":"alice.bsky.social"}';
    final parsed = WhoAmIMapper.fromJson(json);
    expect(parsed.did, 'did:plc:alice');
    expect(parsed.handle, 'alice.bsky.social');
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd app && flutter test test/shared/api/models/
```

Expected: FAIL — import errors.

- [ ] **Step 3: Implement `LoginResponse`** — `app/lib/shared/api/models/login_response.dart`:

```dart
import 'package:dart_mappable/dart_mappable.dart';

part 'login_response.mapper.dart';

@MappableClass(caseStyle: CaseStyle.snakeCase)
class LoginResponse with LoginResponseMappable {
  const LoginResponse({required this.authUrl});

  final String authUrl;
}
```

- [ ] **Step 4: Implement `WhoAmI`** — `app/lib/shared/api/models/whoami.dart`:

```dart
import 'package:dart_mappable/dart_mappable.dart';

part 'whoami.mapper.dart';

@MappableClass()
class WhoAmI with WhoAmIMappable {
  const WhoAmI({required this.did, required this.handle});

  final String did;
  final String handle;
}
```

- [ ] **Step 5: Run build_runner** to generate mappers

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: `login_response.mapper.dart` and `whoami.mapper.dart` written. No errors.

- [ ] **Step 6: Register mappers in `bootstrap.dart`** — edit the `initializeMappers` function to add the new calls:

```dart
void initializeMappers() {
  AppDependenciesMapper.ensureInitialized();
  CraftskyDeviceInfoMapper.ensureInitialized();
  LoginResponseMapper.ensureInitialized();
  WhoAmIMapper.ensureInitialized();
}
```

And add the imports at the top of `bootstrap.dart`:

```dart
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd app && flutter test test/shared/api/models/
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add app/lib/shared/api/models app/test/shared/api/models app/lib/bootstrap.dart
git commit -m "feat(app): add LoginResponse / WhoAmI response models"
```

---

### Task 4: `_AuthInterceptor` — Bearer injection

The interceptor resolves the token in this order: `inFlightTokenProvider` (if set, i.e. mid-handoff) → token field on a `SignedIn` auth state. For Chunk 1 we build a version that only wires against the *current* auth/in-flight state using `Ref`; the providers it depends on will be stubbed in tests via overrides. When those providers land in Chunk 2/3 this file stays unchanged.

**Files:**
- Create: `app/lib/shared/api/providers/auth_interceptor.dart`
- Create: `app/test/shared/api/providers/auth_interceptor_test.dart`

- [ ] **Step 1: Write the failing test**

`app/test/shared/api/providers/auth_interceptor_test.dart`:

```dart
import 'package:craftsky_app/shared/api/providers/auth_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// Test-only providers that stand in for inFlightTokenProvider and
// authSessionProvider from later chunks. The interceptor reads these
// via the resolver callback injected by the constructor — keeping the
// production providers out of Chunk 1's dependency graph.
final fakeInFlightTokenProvider = StateProvider<String?>((_) => null);
final fakePersistedTokenProvider = StateProvider<String?>((_) => null);

String? _resolve(Ref ref) =>
    ref.read(fakeInFlightTokenProvider) ??
    ref.read(fakePersistedTokenProvider);

void main() {
  group('AuthInterceptor', () {
    late ProviderContainer container;
    late RequestOptions options;
    late _CapturingHandler handler;

    setUp(() {
      container = ProviderContainer();
      addTearDown(container.dispose);
      options = RequestOptions(path: '/v1/whoami');
      handler = _CapturingHandler();
    });

    test('adds Authorization header from persisted token', () {
      container.read(fakePersistedTokenProvider.notifier).state = 'tok-persisted';

      AuthInterceptor(container, _resolve).onRequest(options, handler);

      expect(options.headers['Authorization'], 'Bearer tok-persisted');
      expect(handler.continued, isTrue);
    });

    test('prefers in-flight token over persisted token', () {
      container.read(fakeInFlightTokenProvider.notifier).state = 'tok-handoff';
      container.read(fakePersistedTokenProvider.notifier).state = 'tok-persisted';

      AuthInterceptor(container, _resolve).onRequest(options, handler);

      expect(options.headers['Authorization'], 'Bearer tok-handoff');
    });

    test('omits Authorization header when no token is resolved', () {
      AuthInterceptor(container, _resolve).onRequest(options, handler);

      expect(options.headers.containsKey('Authorization'), isFalse);
    });

    test('skips Authorization for /v1/auth/login even when a token exists', () {
      container.read(fakePersistedTokenProvider.notifier).state = 'tok-persisted';
      options = RequestOptions(path: '/v1/auth/login');

      AuthInterceptor(container, _resolve).onRequest(options, handler);

      expect(options.headers.containsKey('Authorization'), isFalse);
    });
  });
}

class _CapturingHandler extends RequestInterceptorHandler {
  bool continued = false;

  @override
  void next(RequestOptions options) {
    continued = true;
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/shared/api/providers/auth_interceptor_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/shared/api/providers/auth_interceptor.dart`:

```dart
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Paths on which the Authorization header should never be attached.
/// Single constant today; revisit if another anonymous endpoint lands.
const _anonymousPaths = <String>{'/v1/auth/login'};

/// Resolves the current Bearer token for outgoing requests.
///
/// The resolver is injected via constructor so the interceptor can be
/// tested against test-local providers without depending on the real
/// auth providers (which are built in later chunks).
typedef TokenResolver = String? Function(Ref ref);

class AuthInterceptor extends Interceptor {
  AuthInterceptor(this._ref, this._resolve);

  final Ref _ref;
  final TokenResolver _resolve;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (_anonymousPaths.contains(options.path)) {
      handler.next(options);
      return;
    }
    final token = _resolve(_ref);
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/shared/api/providers/auth_interceptor_test.dart
```

Expected: PASS, 4 tests.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/api/providers/auth_interceptor.dart app/test/shared/api/providers/auth_interceptor_test.dart
git commit -m "feat(app): add AuthInterceptor that Bearer-injects via a resolver callback"
```

---

### Task 5: `_ErrorMappingInterceptor` — error mapping (401 sign-out deferred)

Chunk 5 adds the global 401 side effect. Here we only map `DioException` → `ApiException` so the error types land alongside the interceptor itself. The 401 sign-out behaviour is a constructor-injected callback so it can be a no-op in Chunk 1 tests.

**How the unwrap works.** `dio` rethrows the error its interceptor chain produces. We set `DioException.error` to an `ApiException` and let `dio.xxx()` throw the wrapping `DioException`; the thin unwrap helper in `CraftskyApiClient` (Task 6) catches that `DioException` and rethrows the contained `ApiException`. This keeps the interceptor pipeline idiomatic (no `handler.reject` side effects) and keeps every call site of the client dealing in `ApiException`.

**Files:**
- Create: `app/lib/shared/api/providers/error_mapping_interceptor.dart`
- Create: `app/test/shared/api/providers/error_mapping_interceptor_test.dart`

- [ ] **Step 1: Write the failing test**

`app/test/shared/api/providers/error_mapping_interceptor_test.dart`:

```dart
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

DioException _ex({int? status, DioExceptionType type = DioExceptionType.badResponse, dynamic data}) {
  final req = RequestOptions(path: '/v1/whoami');
  return DioException(
    requestOptions: req,
    type: type,
    response: status == null
        ? null
        : Response(requestOptions: req, statusCode: status, data: data),
  );
}

void main() {
  group('ErrorMappingInterceptor', () {
    late _CapturingHandler handler;
    late int onUnauthorizedCalls;

    setUp(() {
      handler = _CapturingHandler();
      onUnauthorizedCalls = 0;
    });

    ErrorMappingInterceptor build() => ErrorMappingInterceptor(
          onUnauthorized: (_) => onUnauthorizedCalls++,
        );

    test('401 → ApiUnauthorized and invokes onUnauthorized', () {
      build().onError(_ex(status: 401), handler);

      expect(handler.error, isA<ApiUnauthorized>());
      expect(onUnauthorizedCalls, 1);
    });

    test('400 with {"error": "handle_required"} → ApiBadRequest(code)', () {
      build().onError(
        _ex(status: 400, data: {'error': 'handle_required'}),
        handler,
      );

      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest).code, 'handle_required');
    });

    test('400 with no error field → ApiBadRequest(null)', () {
      build().onError(_ex(status: 400, data: {}), handler);

      expect(handler.error, isA<ApiBadRequest>());
      expect((handler.error as ApiBadRequest).code, isNull);
    });

    test('500 → ApiServerError', () {
      build().onError(_ex(status: 500), handler);

      expect(handler.error, isA<ApiServerError>());
    });

    test('timeout → ApiNetworkError', () {
      build().onError(_ex(type: DioExceptionType.connectionTimeout), handler);

      expect(handler.error, isA<ApiNetworkError>());
    });

    test('connection error → ApiNetworkError', () {
      build().onError(_ex(type: DioExceptionType.connectionError), handler);

      expect(handler.error, isA<ApiNetworkError>());
    });
  });
}

class _CapturingHandler extends ErrorInterceptorHandler {
  Object? error;

  @override
  void next(DioException err) {
    error = err.error;
  }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/shared/api/providers/error_mapping_interceptor_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/shared/api/providers/error_mapping_interceptor.dart`:

```dart
import 'package:dio/dio.dart';

import '../api_exception.dart';

/// Called when an authenticated request returns 401. The production
/// wiring in Chunk 5 signs the user out; tests pass a no-op or a
/// counting fake.
typedef OnUnauthorized = void Function(RequestOptions options);

class ErrorMappingInterceptor extends Interceptor {
  ErrorMappingInterceptor({required this.onUnauthorized});

  final OnUnauthorized onUnauthorized;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    final mapped = _mapDioError(err);
    if (mapped is ApiUnauthorized) {
      onUnauthorized(err.requestOptions);
    }
    handler.next(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: mapped,
        stackTrace: err.stackTrace,
      ),
    );
  }

  ApiException _mapDioError(DioException err) {
    switch (err.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.connectionError:
        return ApiNetworkError(err.message ?? err.type.name);
      case DioExceptionType.badResponse:
        final status = err.response?.statusCode ?? 0;
        if (status == 401) return const ApiUnauthorized();
        if (status >= 400 && status < 500) {
          final data = err.response?.data;
          final code = data is Map && data['error'] is String
              ? data['error'] as String
              : null;
          return ApiBadRequest(code);
        }
        return ApiServerError('http_$status');
      case DioExceptionType.cancel:
      case DioExceptionType.badCertificate:
      case DioExceptionType.unknown:
        // Treat socket-ish "unknown" as network; pass through anything
        // else as a server error with the raw message.
        if (err.error is Exception) {
          return ApiNetworkError(err.message ?? 'network_error');
        }
        return ApiServerError(err.message ?? 'server_error');
    }
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/shared/api/providers/error_mapping_interceptor_test.dart
```

Expected: PASS, 6 tests.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/api/providers/error_mapping_interceptor.dart app/test/shared/api/providers/error_mapping_interceptor_test.dart
git commit -m "feat(app): add ErrorMappingInterceptor that maps DioException → ApiException"
```

---

### Task 6: `CraftskyApiClient` — typed login/whoami/logout

**Files:**
- Create: `app/lib/shared/api/craftsky_api_client.dart`
- Create: `app/test/shared/api/craftsky_api_client_test.dart`

- [ ] **Step 1: Write the failing test**

`app/test/shared/api/craftsky_api_client_test.dart`:

```dart
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    dio.interceptors.add(ErrorMappingInterceptor(onUnauthorized: (_) {}));
    return dio;
  }

  group('CraftskyApiClient.login', () {
    test('POSTs /v1/auth/login with handle + deep_link handoff', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost(
        '/v1/auth/login',
        (server) => server.reply(200, {'auth_url': 'https://pds.example.com/auth?x=1'}),
        data: {'handle': 'alice.bsky.social', 'handoff_mode': 'deep_link'},
      );

      final res = await CraftskyApiClient(dio).login(handle: 'alice.bsky.social');

      expect(res.authUrl, 'https://pds.example.com/auth?x=1');
    });

    test('400 with handle_required surfaces as ApiBadRequest(handle_required)', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost(
        '/v1/auth/login',
        (server) => server.reply(400, {'error': 'handle_required'}),
      );

      await expectLater(
        () => CraftskyApiClient(dio).login(handle: ''),
        throwsA(isA<ApiBadRequest>().having((e) => e.code, 'code', 'handle_required')),
      );
    });
  });

  group('CraftskyApiClient.whoami', () {
    test('GETs /v1/whoami and parses did + handle', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onGet(
        '/v1/whoami',
        (server) => server.reply(200, {'did': 'did:plc:alice', 'handle': 'alice.bsky.social'}),
      );

      final res = await CraftskyApiClient(dio).whoami();

      expect(res.did, 'did:plc:alice');
      expect(res.handle, 'alice.bsky.social');
    });

    test('401 surfaces as ApiUnauthorized', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onGet('/v1/whoami', (server) => server.reply(401, {}));

      await expectLater(
        () => CraftskyApiClient(dio).whoami(),
        throwsA(isA<ApiUnauthorized>()),
      );
    });
  });

  group('CraftskyApiClient.logout', () {
    test('POSTs /v1/auth/logout and returns on 204', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio);
      adapter.onPost('/v1/auth/logout', (server) => server.reply(204, null));

      await CraftskyApiClient(dio).logout();
    });
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/shared/api/craftsky_api_client_test.dart
```

Expected: FAIL — `craftsky_api_client.dart` not found.

- [ ] **Step 3: Implement** — `app/lib/shared/api/craftsky_api_client.dart`:

```dart
import 'package:dio/dio.dart';

import 'api_exception.dart';
import 'models/login_response.dart';
import 'models/whoami.dart';

/// Thin typed wrapper around the three AppView endpoints this release
/// needs. All calls assume the attached [Dio] has the auth + error
/// interceptors installed (see [dioProvider] / Chunk 1 / Chunk 5).
///
/// Each method unwraps the `DioException` that `dio` throws and
/// rethrows the `ApiException` carried in its `.error` field — so
/// callers only ever deal in `ApiException` subtypes.
class CraftskyApiClient {
  const CraftskyApiClient(this._dio);

  final Dio _dio;

  /// POST /v1/auth/login — starts an OAuth flow for [handle], returns
  /// the authorization URL the caller opens in the system browser.
  /// The app-level handoff is always `deep_link` (mobile-only).
  Future<LoginResponse> login({required String handle}) => _unwrap(() async {
        final res = await _dio.post<Map<String, dynamic>>(
          '/v1/auth/login',
          data: {'handle': handle, 'handoff_mode': 'deep_link'},
        );
        return LoginResponseMapper.fromMap(res.data!);
      });

  /// GET /v1/whoami — resolves the caller's DID + handle. Requires an
  /// authenticated request (Bearer token attached by AuthInterceptor).
  Future<WhoAmI> whoami() => _unwrap(() async {
        final res = await _dio.get<Map<String, dynamic>>('/v1/whoami');
        return WhoAmIMapper.fromMap(res.data!);
      });

  /// POST /v1/auth/logout — revokes the current Craftsky session
  /// (single-device). Server responds 204.
  Future<void> logout() => _unwrap(() async {
        await _dio.post<void>('/v1/auth/logout');
      });

  /// Runs [body], translating any `DioException` whose `.error` is an
  /// `ApiException` into a direct throw of that `ApiException`. Other
  /// `DioException`s — theoretically unreachable because
  /// `ErrorMappingInterceptor` always sets `.error` — surface as
  /// `ApiServerError` with the underlying message.
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

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/shared/api/craftsky_api_client_test.dart
```

Expected: PASS, 5 tests.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/api/craftsky_api_client.dart app/test/shared/api/craftsky_api_client_test.dart
git commit -m "feat(app): add CraftskyApiClient with login/whoami/logout methods"
```

---

### Task 7: `dioProvider` and `craftskyApiClientProvider`

The `TokenResolver` is wired with a real provider hook in Chunk 5 (once `inFlightTokenProvider` and `authSessionProvider` exist). For now, we install the providers with a placeholder resolver that always returns null — tests in later chunks will override them.

Until Chunk 5 wires this up, any code that accidentally reads `craftskyApiClientProvider` in production will get an unauthenticated client (no Bearer on outgoing requests). That's acceptable between chunks because **no production caller of the API client exists until Chunk 3**, when `AuthSession` fires `whoami` during background validation.

**Files:**
- Create: `app/lib/shared/api/providers/dio_provider.dart`
- Create: `app/lib/shared/api/providers/api_client_provider.dart`
- Create: `app/test/shared/api/providers/dio_provider_test.dart`

- [ ] **Step 1: Create `dio_provider.dart`** — `app/lib/shared/api/providers/dio_provider.dart`:

```dart
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'auth_interceptor.dart';
import 'error_mapping_interceptor.dart';

part 'dio_provider.g.dart';

/// Android emulator maps the host machine to 10.0.2.2. iOS simulator
/// reaches the host directly via localhost. We default to the Android
/// footgun; iOS devs pass --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:8080.
const _devDefaultBaseUrl = 'http://10.0.2.2:8080';

const _baseUrl = String.fromEnvironment(
  'CRAFTSKY_API_BASE_URL',
  defaultValue: kDebugMode ? _devDefaultBaseUrl : '',
);

@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  if (_baseUrl.isEmpty) {
    throw StateError(
      'CRAFTSKY_API_BASE_URL must be set for non-debug builds. '
      'Pass it via --dart-define.',
    );
  }

  final dio = Dio(
    BaseOptions(
      baseUrl: _baseUrl,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 15),
      headers: {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.addAll([
    // Resolver returns null until Chunk 5 wires the real auth providers.
    AuthInterceptor(ref, (_) => null),
    ErrorMappingInterceptor(onUnauthorized: (_) {}),
  ]);

  return dio;
}
```

- [ ] **Step 2: Create `api_client_provider.dart`** — `app/lib/shared/api/providers/api_client_provider.dart`:

```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../craftsky_api_client.dart';
import 'dio_provider.dart';

part 'api_client_provider.g.dart';

@Riverpod(keepAlive: true)
CraftskyApiClient craftskyApiClient(Ref ref) =>
    CraftskyApiClient(ref.watch(dioProvider));
```

- [ ] **Step 3: Generate provider code**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: `dio_provider.g.dart` and `api_client_provider.g.dart` created.

- [ ] **Step 4: Write a smoke test for `dioProvider`** — `app/test/shared/api/providers/dio_provider_test.dart`:

```dart
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('dioProvider builds with the debug-default base URL', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    final dio = container.read(dioProvider);

    // Debug-mode default per the provider.
    expect(dio.options.baseUrl, 'http://10.0.2.2:8080');
    expect(dio.interceptors, hasLength(2));
  });
}
```

Note: this test runs in `kDebugMode == true` (the Flutter test harness is always debug), so the default URL is exercised. Release-mode behaviour (the `StateError` on empty base URL) is intrinsically compile-time-gated and not unit-testable — it's smoke-tested at release-build time via `flutter build --dart-define=` omission failing fast.

- [ ] **Step 5: Analyzer clean**

```bash
cd app && dart analyze lib/shared/api
```

Expected: `No issues found!`

- [ ] **Step 6: Run test**

```bash
cd app && flutter test test/shared/api/providers/dio_provider_test.dart
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add app/lib/shared/api/providers/dio_provider.dart app/lib/shared/api/providers/dio_provider.g.dart app/lib/shared/api/providers/api_client_provider.dart app/lib/shared/api/providers/api_client_provider.g.dart app/test/shared/api/providers/dio_provider_test.dart
git commit -m "feat(app): add dioProvider + craftskyApiClientProvider"
```

---

### End-of-chunk gate

- [ ] **Run full analyzer + test suite**

```bash
cd app && dart analyze lib test
cd app && flutter test
```

Expected: `No issues found!` and all tests green. Do not advance to Chunk 2 if either fails.

---

## Chunk 2: Auth models + secure storage + small providers

Goal: lay down the data types (`AuthState`, `StoredSession`, `PendingAuth`, `AuthError`) and the thin providers (`SecureTokenStorage`, `InFlightToken`, `PendingAuthProvider`) they depend on. No orchestration yet.

### Task 8: `AuthState` sealed class

**Files:**
- Create: `app/lib/auth/models/auth_state.dart`
- Create: `app/test/auth/models/auth_state_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/models/auth_state_test.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SignedOut is const-equal', () {
    expect(const SignedOut(), const SignedOut());
  });

  test('SignedIn carries did, handle, token', () {
    const s = SignedIn(did: 'did:plc:a', handle: 'a.bsky.social', token: 'tok');
    expect(s.did, 'did:plc:a');
    expect(s.handle, 'a.bsky.social');
    expect(s.token, 'tok');
  });

  test('AuthState pattern-matches exhaustively', () {
    const values = <AuthState>[
      SignedOut(),
      SignedIn(did: 'd', handle: 'h', token: 't'),
    ];
    for (final v in values) {
      final label = switch (v) {
        SignedOut() => 'out',
        SignedIn() => 'in',
      };
      expect(label, anyOf('out', 'in'));
    }
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/models/auth_state_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/models/auth_state.dart`:

```dart
/// High-level auth state exposed by `authSessionProvider`. Distinct
/// from the notifier class name (`AuthSession`) to avoid shadowing
/// inside `AuthSession.build()`.
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

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/auth/models/auth_state_test.dart
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/lib/auth/models/auth_state.dart app/test/auth/models/auth_state_test.dart
git commit -m "feat(app): add sealed AuthState with SignedIn/SignedOut variants"
```

---

### Task 9: `StoredSession` + `PendingAuth` models

**Files:**
- Create: `app/lib/auth/models/stored_session.dart`
- Create: `app/lib/auth/models/pending_auth.dart`
- Create: `app/test/auth/models/stored_session_test.dart`
- Create: `app/test/auth/models/pending_auth_test.dart`
- Modify: `app/lib/bootstrap.dart` (register mappers)

- [ ] **Step 1: Write the failing tests**

`app/test/auth/models/stored_session_test.dart`:

```dart
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('StoredSession round-trips through JSON', () {
    const original = StoredSession(token: 'tok', did: 'did:plc:a', handle: 'a.bsky.social');
    final roundTrip = StoredSessionMapper.fromJson(original.toJson());
    expect(roundTrip.token, 'tok');
    expect(roundTrip.did, 'did:plc:a');
    expect(roundTrip.handle, 'a.bsky.social');
  });
}
```

`app/test/auth/models/pending_auth_test.dart`:

```dart
import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('PendingAuth round-trips through JSON', () {
    final original = PendingAuth(
      handle: 'a.bsky.social',
      startedAt: DateTime.utc(2026, 4, 21, 12),
    );
    final roundTrip = PendingAuthMapper.fromJson(original.toJson());
    expect(roundTrip.handle, 'a.bsky.social');
    expect(roundTrip.startedAt, DateTime.utc(2026, 4, 21, 12));
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd app && flutter test test/auth/models/stored_session_test.dart test/auth/models/pending_auth_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement `StoredSession`** — `app/lib/auth/models/stored_session.dart`:

```dart
import 'package:dart_mappable/dart_mappable.dart';

part 'stored_session.mapper.dart';

/// Single JSON blob we persist to `flutter_secure_storage` under the
/// key `craftsky_session`. `did` and `handle` are cached so cold start
/// can render an optimistic `SignedIn(did, handle)` without waiting
/// for a `/whoami` round-trip — background validation reconciles them.
@MappableClass()
class StoredSession with StoredSessionMappable {
  const StoredSession({
    required this.token,
    required this.did,
    required this.handle,
  });

  final String token;
  final String did;
  final String handle;
}
```

- [ ] **Step 4: Implement `PendingAuth`** — `app/lib/auth/models/pending_auth.dart`:

```dart
import 'package:dart_mappable/dart_mappable.dart';

part 'pending_auth.mapper.dart';

/// Records that a sign-in flow is in progress. `startedAt` is used by
/// `AuthController.completeFromDeepLink` to reject stale deep links
/// (older than 10 minutes).
@MappableClass()
class PendingAuth with PendingAuthMappable {
  const PendingAuth({required this.handle, required this.startedAt});

  final String handle;
  final DateTime startedAt;
}
```

- [ ] **Step 5: Generate mappers**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: two new `.mapper.dart` files.

- [ ] **Step 6: Register mappers in `bootstrap.dart`**

Add imports:

```dart
import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
```

Add to `initializeMappers`:

```dart
  StoredSessionMapper.ensureInitialized();
  PendingAuthMapper.ensureInitialized();
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd app && flutter test test/auth/models/
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add app/lib/auth/models app/test/auth/models app/lib/bootstrap.dart
git commit -m "feat(app): add StoredSession and PendingAuth models"
```

---

### Task 10: `AuthError` sealed class

**Files:**
- Create: `app/lib/auth/models/auth_error.dart`
- Create: `app/test/auth/models/auth_error_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/models/auth_error_test.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AuthError pattern-matches exhaustively', () {
    final values = <AuthError>[
      const HandleRequired(),
      const InvalidHandle(),
      const ServerUnavailable(),
      const BrowserLaunchFailed(),
      const NoPendingSignIn(),
      const SignInTimedOut(),
      StorageFailure(Exception('oops')),
    ];
    for (final e in values) {
      final label = switch (e) {
        HandleRequired() => 'handle_required',
        InvalidHandle() => 'invalid_handle',
        ServerUnavailable() => 'server_unavailable',
        BrowserLaunchFailed() => 'browser_launch_failed',
        NoPendingSignIn() => 'no_pending',
        SignInTimedOut() => 'timed_out',
        StorageFailure() => 'storage',
      };
      expect(label, isNotEmpty);
    }
  });

  test('StorageFailure preserves its cause', () {
    final cause = Exception('keystore down');
    expect(StorageFailure(cause).cause, same(cause));
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/models/auth_error_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/models/auth_error.dart`:

```dart
/// User-actionable auth errors surfaced by `AuthController`. Sealed so
/// call sites can exhaustively switch on them.
sealed class AuthError implements Exception {
  const AuthError();
}

/// User submitted an empty handle.
final class HandleRequired extends AuthError {
  const HandleRequired();
}

/// Server rejected the handle (e.g. malformed). Mapped from any
/// non-specific 4xx from `/v1/auth/login`.
final class InvalidHandle extends AuthError {
  const InvalidHandle();
}

/// AppView is unreachable or returned 5xx, or the device is offline.
final class ServerUnavailable extends AuthError {
  const ServerUnavailable();
}

/// `url_launcher` failed to open the system browser.
final class BrowserLaunchFailed extends AuthError {
  const BrowserLaunchFailed();
}

/// A deep link arrived but no sign-in is in progress.
final class NoPendingSignIn extends AuthError {
  const NoPendingSignIn();
}

/// A deep link arrived more than 10 minutes after the user started
/// the sign-in.
final class SignInTimedOut extends AuthError {
  const SignInTimedOut();
}

/// `flutter_secure_storage` read/write failed (Android keystore issues,
/// platform quirks).
final class StorageFailure extends AuthError {
  const StorageFailure(this.cause);

  final Object cause;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/auth/models/auth_error_test.dart
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/lib/auth/models/auth_error.dart app/test/auth/models/auth_error_test.dart
git commit -m "feat(app): add sealed AuthError with seven user-actionable variants"
```

---

### Task 11: `SecureTokenStorage`

**Files:**
- Create: `app/lib/auth/providers/secure_token_storage.dart`
- Create: `app/test/auth/providers/secure_token_storage_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/providers/secure_token_storage_test.dart`:

```dart
import 'dart:convert';

import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
  });

  test('read returns null when storage is empty', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());
    expect(await storage.read(), isNull);
  });

  test('write then read round-trips a session', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    await storage.write(
      const StoredSession(token: 'tok', did: 'did:plc:a', handle: 'a.bsky.social'),
    );

    final session = await storage.read();
    expect(session, isNotNull);
    expect(session!.token, 'tok');
    expect(session.did, 'did:plc:a');
  });

  test('clear removes the stored session', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    await storage.write(
      const StoredSession(token: 't', did: 'd', handle: 'h'),
    );
    await storage.clear();

    expect(await storage.read(), isNull);
  });

  test('read returns null on corrupt blob and logs a warning', () async {
    FlutterSecureStorage.setMockInitialValues(
      {'craftsky_session': 'not-valid-json'},
    );
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    expect(await storage.read(), isNull);
  });

  test('read gives back well-formed JSON that matches the blob shape', () async {
    FlutterSecureStorage.setMockInitialValues(
      {'craftsky_session': jsonEncode({'token': 't', 'did': 'd', 'handle': 'h'})},
    );
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    final session = await storage.read();
    expect(session?.token, 't');
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/providers/secure_token_storage_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/providers/secure_token_storage.dart`:

```dart
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../models/stored_session.dart';

part 'secure_token_storage.g.dart';

final _log = Logger('SecureTokenStorage');

/// Thin async wrapper around [FlutterSecureStorage] for the Craftsky
/// session blob. All platform errors are swallowed and logged so the
/// app can always fall back to a `SignedOut` state rather than
/// crashing on startup.
class SecureTokenStorage {
  SecureTokenStorage(this._fss);

  final FlutterSecureStorage _fss;

  static const _key = 'craftsky_session';

  Future<StoredSession?> read() async {
    try {
      final raw = await _fss.read(key: _key);
      if (raw == null) return null;
      return StoredSessionMapper.fromJson(raw);
    } on PlatformException catch (e, st) {
      _log.warning('read failed; treating as unsigned-in', e, st);
      return null;
    } catch (e, st) {
      // FormatException (malformed JSON) or MapperException (missing
      // required fields from dart_mappable) — both mean the blob on
      // disk is garbage we can't use. Delete it so subsequent writes
      // aren't fighting a corrupt value.
      _log.warning('corrupt blob; clearing', e, st);
      try {
        await _fss.delete(key: _key);
      } catch (deleteErr, deleteSt) {
        _log.warning('delete-after-corrupt also failed', deleteErr, deleteSt);
      }
      return null;
    }
  }

  Future<void> write(StoredSession session) =>
      _fss.write(key: _key, value: session.toJson());

  Future<void> clear() => _fss.delete(key: _key);
}

@Riverpod(keepAlive: true)
SecureTokenStorage secureTokenStorage(Ref ref) =>
    SecureTokenStorage(const FlutterSecureStorage());
```

- [ ] **Step 4: Generate provider code**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: `secure_token_storage.g.dart` created.

- [ ] **Step 5: Run test to verify it passes**

```bash
cd app && flutter test test/auth/providers/secure_token_storage_test.dart
```

Expected: PASS, 5 tests.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/providers/secure_token_storage.dart app/lib/auth/providers/secure_token_storage.g.dart app/test/auth/providers/secure_token_storage_test.dart
git commit -m "feat(app): add SecureTokenStorage async wrapper around flutter_secure_storage"
```

---

### Task 12: `InFlightTokenProvider`

**Files:**
- Create: `app/lib/auth/providers/in_flight_token_provider.dart`
- Create: `app/test/auth/providers/in_flight_token_provider_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/providers/in_flight_token_provider_test.dart`:

```dart
import 'package:craftsky_app/auth/providers/in_flight_token_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late ProviderContainer container;

  setUp(() {
    container = ProviderContainer();
    addTearDown(container.dispose);
  });

  test('starts null', () {
    expect(container.read(inFlightTokenProvider), isNull);
  });

  test('setToken updates state', () {
    container.read(inFlightTokenProvider.notifier).setToken('tok');
    expect(container.read(inFlightTokenProvider), 'tok');
  });

  test('clear resets to null', () {
    container.read(inFlightTokenProvider.notifier).setToken('tok');
    container.read(inFlightTokenProvider.notifier).clear();
    expect(container.read(inFlightTokenProvider), isNull);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/providers/in_flight_token_provider_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/providers/in_flight_token_provider.dart`:

```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'in_flight_token_provider.g.dart';

/// Holds the bearer token between the deep-link arriving in
/// `AuthController.completeFromDeepLink` and the follow-up `whoami`
/// call resolving. Read by the Dio auth interceptor as a fallback
/// before secure storage so the whoami call can be authenticated
/// without persisting a half-written session.
@Riverpod(keepAlive: true)
class InFlightToken extends _$InFlightToken {
  @override
  String? build() => null;

  void setToken(String token) => state = token;
  void clear() => state = null;
}
```

- [ ] **Step 4: Generate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd app && flutter test test/auth/providers/in_flight_token_provider_test.dart
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/providers/in_flight_token_provider.dart app/lib/auth/providers/in_flight_token_provider.g.dart app/test/auth/providers/in_flight_token_provider_test.dart
git commit -m "feat(app): add InFlightTokenProvider"
```

---

### Task 13: `PendingAuth` notifier (`pendingAuthProvider`)

The notifier class name is `PendingAuth`, but the value type is the `PendingAuth` data class from `auth/models/pending_auth.dart`. To avoid the name collision inside the notifier, we import the model with the prefix `model`. The generated provider is `pendingAuthProvider` (from the notifier class name). Callers consume `pendingAuthProvider` (the provider) and see `model.PendingAuth?` values.

**Files:**
- Create: `app/lib/auth/providers/pending_auth_provider.dart`
- Create: `app/test/auth/providers/pending_auth_provider_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/providers/pending_auth_provider_test.dart`:

```dart
import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late ProviderContainer container;

  setUp(() {
    container = ProviderContainer();
    addTearDown(container.dispose);
  });

  test('starts null', () {
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('start records handle + current time', () {
    final before = DateTime.now();
    container.read(pendingAuthProvider.notifier).start('alice.bsky.social');
    final pending = container.read(pendingAuthProvider);

    expect(pending, isA<model.PendingAuth>());
    expect(pending!.handle, 'alice.bsky.social');
    expect(pending.startedAt.isBefore(before), isFalse);
  });

  test('clear resets to null', () {
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');
    container.read(pendingAuthProvider.notifier).clear();
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('start overwrites any prior pending auth', () {
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');
    container.read(pendingAuthProvider.notifier).start('b.bsky.social');
    expect(container.read(pendingAuthProvider)!.handle, 'b.bsky.social');
  });

  test('debugSet directly replaces state (for aging in other tests)', () {
    final aged = model.PendingAuth(
      handle: 'x.bsky.social',
      startedAt: DateTime.now().subtract(const Duration(minutes: 15)),
    );
    container.read(pendingAuthProvider.notifier).debugSet(aged);

    expect(container.read(pendingAuthProvider)!.startedAt, aged.startedAt);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/providers/pending_auth_provider_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/providers/pending_auth_provider.dart`:

```dart
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../models/pending_auth.dart' as model;

part 'pending_auth_provider.g.dart';

/// Tracks the in-flight sign-in attempt. Lets
/// `AuthController.completeFromDeepLink` reject deep links that
/// arrive without a prior `signIn()` or later than the 10-minute
/// staleness window.
///
/// The notifier class is named `PendingAuth` — same identifier as
/// the data class it holds, imported under the `model` prefix to
/// dodge the collision inside this file. The generated provider is
/// `pendingAuthProvider`.
@Riverpod(keepAlive: true)
class PendingAuth extends _$PendingAuth {
  @override
  model.PendingAuth? build() => null;

  void start(String handle) => state = model.PendingAuth(
        handle: handle,
        startedAt: DateTime.now(),
      );

  void clear() => state = null;

  /// Direct state setter — used by tests that need to age the
  /// `startedAt` without real clock manipulation (see
  /// `auth_controller_test.dart` for the stale-pending scenario).
  @visibleForTesting
  void debugSet(model.PendingAuth value) => state = value;
}
```

- [ ] **Step 4: Generate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd app && flutter test test/auth/providers/pending_auth_provider_test.dart
```

Expected: PASS, 5 tests.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/providers/pending_auth_provider.dart app/lib/auth/providers/pending_auth_provider.g.dart app/test/auth/providers/pending_auth_provider_test.dart
git commit -m "feat(app): add PendingAuth notifier tracking in-flight sign-in attempts"
```

---

### End-of-chunk gate

- [ ] **Run full analyzer + test suite**

```bash
cd app && dart analyze lib test
cd app && flutter test
```

Expected: `No issues found!` and all tests green.

---

## Chunk 3: `AuthSession`, `AuthController`, `OnboardingStatus` rewrite

Goal: the core state machine. `AuthSession` exposes `AsyncValue<AuthState>` with optimistic cold start + background validation. `AuthController` drives signIn/completeFromDeepLink/signOut. `OnboardingStatus` is rewritten as a family keyed by DID, backed by `SharedPreferences`.

### Task 14: `AuthSession` — optimistic cold start + background validation

**Files:**
- Create: `app/lib/auth/providers/auth_session_provider.dart`
- Create: `app/test/auth/providers/auth_session_provider_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/providers/auth_session_provider_test.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/api_client_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// --- fakes ----------------------------------------------------------

// `implements` (not `extends`) avoids needing to construct a real
// `FlutterSecureStorage` just to satisfy the superclass — we only
// need the three methods the production code calls.
class _FakeStorage implements SecureTokenStorage {
  _FakeStorage({StoredSession? initial}) : _value = initial;
  StoredSession? _value;
  @override
  Future<StoredSession?> read() async => _value;
  @override
  Future<void> write(StoredSession s) async => _value = s;
  @override
  Future<void> clear() async => _value = null;
}

class _FakeApi implements CraftskyApiClient {
  _FakeApi({required this.onWhoami});
  final Future<WhoAmI> Function() onWhoami;
  @override
  Future<WhoAmI> whoami() => onWhoami();
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

ProviderContainer _container({
  required SecureTokenStorage storage,
  CraftskyApiClient? api,
}) =>
    ProviderContainer(
      overrides: [
        secureTokenStorageProvider.overrideWithValue(storage),
        if (api != null) craftskyApiClientProvider.overrideWithValue(api),
      ],
    );

/// `_validateInBackground` is `unawaited` inside `AuthSession.build`; the chain
/// performs up to three awaits (whoami → storage.{clear|write} → state =),
/// each yielding one microtask. Flush the event loop a few times to settle.
Future<void> _flushBackgroundValidation() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}

void main() {
  setUpAll(initializeMappers);

  test('resolves to SignedOut when storage is empty', () async {
    final container = _container(storage: _FakeStorage());
    addTearDown(container.dispose);

    final state = await container.read(authSessionProvider.future);
    expect(state, isA<SignedOut>());
  });

  test('resolves to SignedIn when storage has a session', () async {
    final container = _container(
      storage: _FakeStorage(
        initial: const StoredSession(token: 't', did: 'd', handle: 'h'),
      ),
      api: _FakeApi(
        onWhoami: () async => const WhoAmI(did: 'd', handle: 'h'),
      ),
    );
    addTearDown(container.dispose);

    final state = await container.read(authSessionProvider.future);
    expect(state, isA<SignedIn>());
    final signed = state as SignedIn;
    expect(signed.token, 't');
  });

  test('whoami 401 clears storage and flips to SignedOut', () async {
    final storage = _FakeStorage(
      initial: const StoredSession(token: 't', did: 'd', handle: 'h'),
    );
    final container = _container(
      storage: storage,
      api: _FakeApi(onWhoami: () async => throw const ApiUnauthorized()),
    );
    addTearDown(container.dispose);

    // Resolve build: optimistic SignedIn.
    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    expect(container.read(authSessionProvider).value, isA<SignedOut>());
    expect(await storage.read(), isNull);
  });

  test('whoami DID mismatch clears and flips to SignedOut', () async {
    final storage = _FakeStorage(
      initial: const StoredSession(token: 't', did: 'd-old', handle: 'h'),
    );
    final container = _container(
      storage: storage,
      api: _FakeApi(
        onWhoami: () async => const WhoAmI(did: 'd-new', handle: 'h'),
      ),
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    expect(container.read(authSessionProvider).value, isA<SignedOut>());
    expect(await storage.read(), isNull);
  });

  test('whoami handle drift updates cache + state, keeps SignedIn', () async {
    final storage = _FakeStorage(
      initial: const StoredSession(token: 't', did: 'd', handle: 'old.bsky.social'),
    );
    final container = _container(
      storage: storage,
      api: _FakeApi(
        onWhoami: () async => const WhoAmI(did: 'd', handle: 'new.bsky.social'),
      ),
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    final signed = container.read(authSessionProvider).value as SignedIn;
    expect(signed.handle, 'new.bsky.social');
    expect((await storage.read())!.handle, 'new.bsky.social');
  });

  test('whoami network error keeps cached SignedIn (offline tolerance)', () async {
    final storage = _FakeStorage(
      initial: const StoredSession(token: 't', did: 'd', handle: 'h'),
    );
    final container = _container(
      storage: storage,
      api: _FakeApi(onWhoami: () async => throw const ApiNetworkError('offline')),
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    expect(container.read(authSessionProvider).value, isA<SignedIn>());
    expect(await storage.read(), isNotNull);
  });

  test('setSignedIn/setSignedOut set state imperatively', () async {
    final container = _container(storage: _FakeStorage());
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    container.read(authSessionProvider.notifier).setSignedIn(
          const SignedIn(did: 'd', handle: 'h', token: 't'),
        );
    expect(container.read(authSessionProvider).value, isA<SignedIn>());

    container.read(authSessionProvider.notifier).setSignedOut();
    expect(container.read(authSessionProvider).value, isA<SignedOut>());
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd app && flutter test test/auth/providers/auth_session_provider_test.dart
```

Expected: FAIL — `authSessionProvider` doesn't exist yet.

- [ ] **Step 3: Implement** — `app/lib/auth/providers/auth_session_provider.dart`:

```dart
import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../shared/api/api_exception.dart';
import '../../shared/api/providers/api_client_provider.dart';
import '../models/auth_state.dart';
import '../models/stored_session.dart';
import 'secure_token_storage.dart';

part 'auth_session_provider.g.dart';

/// Sole source of truth for the app's auth state. Cold start reads
/// secure storage once and emits an optimistic `SignedIn` immediately
/// (if a session exists), then background-validates via `/whoami`.
/// Later updates come through `setSignedIn` / `setSignedOut`, called
/// by `AuthController` and the global 401 interceptor.
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  @override
  Future<AuthState> build() async {
    final storage = ref.watch(secureTokenStorageProvider);
    final stored = await storage.read();
    if (stored == null) return const SignedOut();

    // Unawaited — we return SignedIn now and validate in parallel.
    unawaited(_validateInBackground(stored));

    return SignedIn(did: stored.did, handle: stored.handle, token: stored.token);
  }

  Future<void> _validateInBackground(StoredSession stored) async {
    try {
      final api = ref.read(craftskyApiClientProvider);
      final who = await api.whoami();
      if (!ref.mounted) return;

      if (who.did != stored.did) {
        await _clearLocalState();
        return;
      }
      if (who.handle != stored.handle) {
        final updated = StoredSession(
          token: stored.token,
          did: who.did,
          handle: who.handle,
        );
        await ref.read(secureTokenStorageProvider).write(updated);
        if (!ref.mounted) return;
        state = AsyncData(SignedIn(
          did: who.did,
          handle: who.handle,
          token: stored.token,
        ));
      }
      // else: handles match; nothing to do.
    } on ApiUnauthorized {
      await _clearLocalState();
    } on ApiNetworkError {
      // Offline; keep cached SignedIn. Next cold start revalidates.
    }
  }

  Future<void> _clearLocalState() async {
    await ref.read(secureTokenStorageProvider).clear();
    if (!ref.mounted) return;
    state = const AsyncData(SignedOut());
  }

  void setSignedIn(SignedIn signedIn) => state = AsyncData(signedIn);

  void setSignedOut() => state = const AsyncData(SignedOut());
}
```

- [ ] **Step 4: Generate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd app && flutter test test/auth/providers/auth_session_provider_test.dart
```

Expected: PASS, 7 tests.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/providers/auth_session_provider.dart app/lib/auth/providers/auth_session_provider.g.dart app/test/auth/providers/auth_session_provider_test.dart
git commit -m "feat(app): add AuthSession w/ optimistic cold start + whoami background validation"
```

---

### Task 15: `AuthController` — signIn / completeFromDeepLink / signOut

**Files:**
- Create: `app/lib/auth/providers/auth_controller.dart`
- Create: `app/test/auth/providers/auth_controller_test.dart`

- [ ] **Step 1: Write the failing test** — `app/test/auth/providers/auth_controller_test.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/in_flight_token_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/api_client_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeStorage implements SecureTokenStorage {
  StoredSession? _v;
  @override
  Future<StoredSession?> read() async => _v;
  @override
  Future<void> write(StoredSession s) async => _v = s;
  @override
  Future<void> clear() async => _v = null;
}

class _FakeApi implements CraftskyApiClient {
  _FakeApi({this.onLogin, this.onWhoami, this.onLogout});
  final Future<LoginResponse> Function(String)? onLogin;
  final Future<WhoAmI> Function()? onWhoami;
  final Future<void> Function()? onLogout;

  @override
  Future<LoginResponse> login({required String handle}) =>
      onLogin?.call(handle) ?? Future.error(UnimplementedError());
  @override
  Future<WhoAmI> whoami() =>
      onWhoami?.call() ?? Future.error(UnimplementedError());
  @override
  Future<void> logout() => onLogout?.call() ?? Future.value();
}

// Records calls instead of touching real url_launcher.
class _LaunchRecorder {
  final List<Uri> launched = [];
  bool nextResult = true;
  Future<bool> launch(Uri uri) async {
    launched.add(uri);
    return nextResult;
  }
}

ProviderContainer _container({
  required _FakeStorage storage,
  required _FakeApi api,
  required _LaunchRecorder launch,
}) =>
    ProviderContainer(
      overrides: [
        secureTokenStorageProvider.overrideWithValue(storage),
        craftskyApiClientProvider.overrideWithValue(api),
        launchAuthUrlProvider.overrideWithValue(launch.launch),
      ],
    );

void main() {
  setUpAll(initializeMappers);

  test('signIn trims handle + @ prefix and posts to /login', () async {
    final launch = _LaunchRecorder();
    final api = _FakeApi(
      onLogin: (h) async {
        expect(h, 'alice.bsky.social');
        return const LoginResponse(authUrl: 'https://pds.example.com/a?b=1');
      },
    );
    final container = _container(
      storage: _FakeStorage(),
      api: api,
      launch: launch,
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier)
        .signIn(handle: '  @alice.bsky.social  ');

    expect(launch.launched, hasLength(1));
    expect(launch.launched.single.toString(),
        'https://pds.example.com/a?b=1');
  });

  test('signIn with empty handle surfaces HandleRequired', () async {
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier).signIn(handle: '');
    final err = container.read(authControllerProvider).error;
    expect(err, isA<HandleRequired>());
  });

  test('signIn maps ApiBadRequest(handle_required) → HandleRequired', () async {
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(onLogin: (_) async => throw const ApiBadRequest('handle_required')),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier).signIn(handle: 'a.bsky.social');
    expect(container.read(authControllerProvider).error, isA<HandleRequired>());
  });

  test('signIn maps ApiNetworkError → ServerUnavailable', () async {
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(onLogin: (_) async => throw const ApiNetworkError('offline')),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier).signIn(handle: 'a.bsky.social');
    expect(container.read(authControllerProvider).error, isA<ServerUnavailable>());
  });

  test('signIn surfaces BrowserLaunchFailed when launchUrl returns false', () async {
    final launch = _LaunchRecorder()..nextResult = false;
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(onLogin: (_) async => const LoginResponse(authUrl: 'https://x')),
      launch: launch,
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier).signIn(handle: 'a.bsky.social');
    expect(container.read(authControllerProvider).error, isA<BrowserLaunchFailed>());
  });

  test('completeFromDeepLink with no pending surfaces NoPendingSignIn', () async {
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier)
        .completeFromDeepLink('tok');
    expect(container.read(authControllerProvider).error, isA<NoPendingSignIn>());
  });

  test('completeFromDeepLink stale pending surfaces SignInTimedOut', () async {
    final container = _container(
      storage: _FakeStorage(),
      api: _FakeApi(),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    // Use debugSet (defined on the PendingAuth notifier in Task 13) to
    // install an aged record directly, without real clock manipulation.
    container.read(pendingAuthProvider.notifier).debugSet(
          model.PendingAuth(
            handle: 'a.bsky.social',
            startedAt: DateTime.now().subtract(const Duration(minutes: 15)),
          ),
        );

    await container.read(authControllerProvider.notifier)
        .completeFromDeepLink('tok');
    expect(container.read(authControllerProvider).error, isA<SignInTimedOut>());
  });

  test('completeFromDeepLink happy path writes storage + flips SignedIn', () async {
    final storage = _FakeStorage();
    final container = _container(
      storage: storage,
      api: _FakeApi(
        onWhoami: () async => const WhoAmI(did: 'did:plc:a', handle: 'a.bsky.social'),
      ),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    // Seed AuthSession build so setSignedIn lands on ready state.
    await container.read(authSessionProvider.future);
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');

    await container.read(authControllerProvider.notifier)
        .completeFromDeepLink('tok');

    final state = container.read(authSessionProvider).value;
    expect(state, isA<SignedIn>());
    expect((state as SignedIn).did, 'did:plc:a');

    final stored = await storage.read();
    expect(stored?.token, 'tok');
    expect(stored?.did, 'did:plc:a');

    expect(container.read(inFlightTokenProvider), isNull);
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('completeFromDeepLink whoami failure clears in-flight + pending, does NOT write storage', () async {
    final storage = _FakeStorage();
    final container = _container(
      storage: storage,
      api: _FakeApi(onWhoami: () async => throw const ApiUnauthorized()),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    container.read(pendingAuthProvider.notifier).start('a.bsky.social');

    await container.read(authControllerProvider.notifier)
        .completeFromDeepLink('tok');

    expect(await storage.read(), isNull);
    expect(container.read(inFlightTokenProvider), isNull);
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('signOut clears storage + flips SignedOut even on server failure', () async {
    final storage = _FakeStorage();
    await storage.write(const StoredSession(token: 't', did: 'd', handle: 'h'));
    final container = _container(
      storage: storage,
      api: _FakeApi(onLogout: () async => throw const ApiNetworkError('offline')),
      launch: _LaunchRecorder(),
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    await container.read(authControllerProvider.notifier).signOut();

    expect(await storage.read(), isNull);
    expect(container.read(authSessionProvider).value, isA<SignedOut>());
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/providers/auth_controller_test.dart
```

Expected: FAIL — `authControllerProvider` / `launchAuthUrlProvider` not found.

- [ ] **Step 3: Implement** — `app/lib/auth/providers/auth_controller.dart`:

```dart
import 'dart:async';

import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

import '../../shared/api/api_exception.dart';
import '../../shared/api/models/login_response.dart';
import '../../shared/api/providers/api_client_provider.dart';
import '../models/auth_error.dart';
import '../models/auth_state.dart';
import '../models/stored_session.dart';
import 'auth_session_provider.dart';
import 'in_flight_token_provider.dart';
import 'pending_auth_provider.dart';
import 'secure_token_storage.dart';

part 'auth_controller.g.dart';

final _log = Logger('AuthController');

/// The URL-launch function. Overridable in tests so we don't trigger
/// the real system browser.
typedef AuthUrlLauncher = Future<bool> Function(Uri uri);

@Riverpod(keepAlive: true)
AuthUrlLauncher launchAuthUrl(Ref ref) {
  return (Uri uri) => url_launcher.launchUrl(
        uri,
        mode: url_launcher.LaunchMode.externalApplication,
      );
}

/// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
/// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
///
/// Tests that need to simulate a stale `PendingAuth` do so via
/// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
/// `PendingAuth` notifier in Task 13), not through this controller.
@Riverpod(keepAlive: true)
class AuthController extends _$AuthController {
  @override
  FutureOr<void> build() => null;

  Future<void> signIn({required String handle}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final trimmed = handle.trim().replaceFirst(RegExp(r'^@'), '');
      if (trimmed.isEmpty) throw const HandleRequired();

      final api = ref.read(craftskyApiClientProvider);
      final LoginResponse response;
      try {
        response = await api.login(handle: trimmed);
      } on ApiBadRequest catch (e) {
        throw e.code == 'handle_required'
            ? const HandleRequired()
            : const InvalidHandle();
      } on ApiNetworkError {
        throw const ServerUnavailable();
      } on ApiServerError {
        throw const ServerUnavailable();
      } on ApiUnauthorized {
        // Login is anonymous; a 401 here is defensive-only.
        throw const ServerUnavailable();
      }

      if (!ref.mounted) return;
      ref.read(pendingAuthProvider.notifier).start(trimmed);

      final launched =
          await ref.read(launchAuthUrlProvider)(Uri.parse(response.authUrl));
      if (!launched) {
        if (ref.mounted) ref.read(pendingAuthProvider.notifier).clear();
        throw const BrowserLaunchFailed();
      }
    });
  }

  Future<void> completeFromDeepLink(String token) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final pending = ref.read(pendingAuthProvider);
      if (pending == null) throw const NoPendingSignIn();
      if (DateTime.now().difference(pending.startedAt) >
          const Duration(minutes: 10)) {
        ref.read(pendingAuthProvider.notifier).clear();
        throw const SignInTimedOut();
      }

      ref.read(inFlightTokenProvider.notifier).setToken(token);
      try {
        final api = ref.read(craftskyApiClientProvider);
        final who = await api.whoami();
        if (!ref.mounted) return;

        final storage = ref.read(secureTokenStorageProvider);
        try {
          await storage.write(
            StoredSession(token: token, did: who.did, handle: who.handle),
          );
        } catch (e, st) {
          _log.warning('SecureTokenStorage.write failed', e, st);
          throw StorageFailure(e);
        }
        if (!ref.mounted) return;

        ref.read(authSessionProvider.notifier).setSignedIn(
              SignedIn(did: who.did, handle: who.handle, token: token),
            );
      } finally {
        if (ref.mounted) {
          ref.read(inFlightTokenProvider.notifier).clear();
          ref.read(pendingAuthProvider.notifier).clear();
        }
      }
    });
  }

  Future<void> signOut() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      try {
        await ref.read(craftskyApiClientProvider).logout();
      } on ApiException catch (e, st) {
        _log.warning('logout network/server error; clearing locally', e, st);
      }
      if (!ref.mounted) return;
      await ref.read(secureTokenStorageProvider).clear();
      if (!ref.mounted) return;
      ref.read(authSessionProvider.notifier).setSignedOut();
    });
  }
}
```

- [ ] **Step 4: Generate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd app && flutter test test/auth/providers/auth_controller_test.dart
```

Expected: PASS, 10 tests.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/providers/auth_controller.dart app/lib/auth/providers/auth_controller.g.dart app/test/auth/providers/auth_controller_test.dart
git commit -m "feat(app): add AuthController w/ signIn, completeFromDeepLink, signOut"
```

---

### Task 16: Rewrite `OnboardingStatus` as a family keyed by DID

**Files:**
- Modify: `app/lib/onboarding/providers/onboarding_status_provider.dart`
- Create: `app/test/onboarding/providers/onboarding_status_provider_test.dart`
- Modify: `app/lib/onboarding/pages/onboarding_page.dart`

- [ ] **Step 1: Write the failing test** — `app/test/onboarding/providers/onboarding_status_provider_test.dart`:

```dart
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

ProviderContainer _container(SharedPreferences prefs) => ProviderContainer(
      overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
    );

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  test('build returns false when no flag stored for this DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    addTearDown(container.dispose);

    expect(container.read(onboardingStatusProvider('did:plc:a')), isFalse);
  });

  test('build returns true when prefs has flag', () async {
    SharedPreferences.setMockInitialValues({'flutter.onboarded_did:plc:a': true});
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    addTearDown(container.dispose);

    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
  });

  test('finish writes flag and flips state for the DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    addTearDown(container.dispose);

    await container.read(onboardingStatusProvider('did:plc:a').notifier).finish();

    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
    expect(prefs.getBool('onboarded_did:plc:a'), isTrue);
  });

  test('finish for one DID does not affect a different DID', () async {
    final prefs = await SharedPreferences.getInstance();
    final container = _container(prefs);
    addTearDown(container.dispose);

    await container.read(onboardingStatusProvider('did:plc:a').notifier).finish();

    expect(container.read(onboardingStatusProvider('did:plc:a')), isTrue);
    expect(container.read(onboardingStatusProvider('did:plc:b')), isFalse);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/onboarding/providers/onboarding_status_provider_test.dart
```

Expected: FAIL — current provider doesn't take an arg.

- [ ] **Step 3: Rewrite `onboarding_status_provider.dart`**:

```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../app_dependencies.dart';

part 'onboarding_status_provider.g.dart';

/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  bool build(String did) {
    final prefs = ref.watch(sharedPreferencesProvider);
    return prefs.getBool(_keyFor(did)) ?? false;
  }

  Future<void> finish() async {
    final prefs = ref.read(sharedPreferencesProvider);
    await prefs.setBool(_keyFor(did), true);
    if (!ref.mounted) return;
    state = true;
  }

  static String _keyFor(String did) => 'onboarded_$did';
}
```

- [ ] **Step 4: Update `onboarding_page.dart`** — `app/lib/onboarding/pages/onboarding_page.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class OnboardingPage extends ConsumerWidget {
  const OnboardingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Onboarding')),
      body: const Center(child: OnboardingPageBody()),
    );
  }
}

class OnboardingPageBody extends ConsumerWidget {
  const OnboardingPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider).valueOrNull;
    final did = switch (auth) {
      SignedIn(:final did) => did,
      _ => null,
    };

    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Onboarding'),
        const SizedBox(height: 24),
        ChunkyButton(
          onPressed: did == null
              ? null
              : () => ref.read(onboardingStatusProvider(did).notifier).finish(),
          child: const Text('Finish'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 5: Generate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 6: Run tests**

```bash
cd app && flutter test test/onboarding/providers/onboarding_status_provider_test.dart
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add app/lib/onboarding app/test/onboarding
git commit -m "refactor(app): key OnboardingStatus by DID via SharedPreferences"
```

---

### End-of-chunk gate

- [ ] **Run full analyzer + test suite**

```bash
cd app && dart analyze lib test
cd app && flutter test
```

Expected: `No issues found!` and all tests green. The old widget/router tests that depend on the deleted stub `authStatusProvider` will still fail here — that's fixed in Chunk 4 (Tasks 19–22). If any failures are **not** in `test/fakes/`, `test/router/`, `test/auth/welcome_page_test.dart`, or `test/auth/sign_in_page_test.dart`, stop and diagnose before advancing.

---

## Chunk 4: Router, AuthCompletePage, page rewires, platform config

Goal: swap the old bool-based `authStatusProvider` out of the router, add the `/auth/complete` route, wire `SignInPage` / `SettingsPage` / `WelcomePage` to the new controller, and land the iOS + Android custom-scheme intent filters.

### Task 17: Add `/auth/complete` route location

**Files:**
- Modify: `app/lib/router/route_locations.dart`

- [ ] **Step 1: Add the constant** — `app/lib/router/route_locations.dart`:

```dart
class RouteLocations {
  RouteLocations._();

  static const welcome = '/welcome';
  static const signIn = '/sign-in';
  static const authComplete = '/auth/complete';
  static const onboarding = '/onboarding';
  // ... (rest unchanged)
}
```

- [ ] **Step 2: Commit**

```bash
git add app/lib/router/route_locations.dart
git commit -m "feat(app): add authComplete route constant"
```

---

### Task 18: Build `AuthCompletePage`

**Files:**
- Create: `app/lib/auth/pages/auth_complete_page.dart`
- Create: `app/test/auth/pages/auth_complete_page_test.dart`

- [ ] **Step 1: Write the failing widget test** — `app/test/auth/pages/auth_complete_page_test.dart`:

```dart
import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  _FakeAuthController({required this.onComplete});
  final Future<void> Function(String token) onComplete;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> completeFromDeepLink(String token) => onComplete(token);
}

void main() {
  testWidgets('calls completeFromDeepLink with the token on init', (tester) async {
    final seen = <String>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(() => _FakeAuthController(
                onComplete: (t) async => seen.add(t),
              )),
        ],
        child: const MaterialApp(
          home: AuthCompletePage(token: 'tok-123'),
        ),
      ),
    );
    await tester.pump(); // one frame for addPostFrameCallback
    expect(seen, ['tok-123']);
  });

  testWidgets('renders spinner by default', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(onComplete: (_) async {}),
          ),
        ],
        child: const MaterialApp(home: AuthCompletePage(token: 't')),
      ),
    );
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('Signing in…'), findsOneWidget);
  });

  testWidgets('renders retry text on AuthError', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (_) async => throw const NoPendingSignIn(),
            ),
          ),
        ],
        child: const MaterialApp(home: AuthCompletePage(token: 't')),
      ),
    );
    await tester.pump();
    await tester.pump(); // allow AsyncError to propagate
    expect(find.textContaining('sign in again', findRichText: true), findsOneWidget);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && flutter test test/auth/pages/auth_complete_page_test.dart
```

Expected: FAIL.

- [ ] **Step 3: Implement** — `app/lib/auth/pages/auth_complete_page.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AuthCompletePage extends ConsumerStatefulWidget {
  const AuthCompletePage({required this.token, super.key});

  final String token;

  @override
  ConsumerState<AuthCompletePage> createState() => _AuthCompletePageState();
}

class _AuthCompletePageState extends ConsumerState<AuthCompletePage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(authControllerProvider.notifier).completeFromDeepLink(widget.token);
    });
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(authControllerProvider);

    return Scaffold(
      body: Center(
        child: switch (state) {
          AsyncError(:final error) when error is AuthError =>
            _AuthCompleteError(error: error),
          AsyncError(:final error) =>
            _AuthCompleteError(error: GenericAuthError(error)),
          _ => const _AuthCompleteLoading(),
        },
      ),
    );
  }
}

class _AuthCompleteLoading extends StatelessWidget {
  const _AuthCompleteLoading();

  @override
  Widget build(BuildContext context) {
    return const Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        CircularProgressIndicator(),
        SizedBox(height: 16),
        Text('Signing in…'),
      ],
    );
  }
}

class _AuthCompleteError extends StatelessWidget {
  const _AuthCompleteError({required this.error});

  final Object error;

  @override
  Widget build(BuildContext context) {
    final message = switch (error) {
      SignInTimedOut() => 'That sign-in link expired. Please sign in again.',
      NoPendingSignIn() => 'No sign-in is in progress. Please sign in again.',
      StorageFailure() =>
        "Couldn't save your session securely. Please sign in again.",
      _ => "Couldn't complete sign-in. Please sign in again.",
    };

    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(message, textAlign: TextAlign.center),
        ],
      ),
    );
  }
}

/// Stand-in for non-AuthError failures so the switch stays exhaustive.
class GenericAuthError implements Exception {
  const GenericAuthError(this.cause);
  final Object cause;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd app && flutter test test/auth/pages/auth_complete_page_test.dart
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/lib/auth/pages/auth_complete_page.dart app/test/auth/pages/auth_complete_page_test.dart
git commit -m "feat(app): add AuthCompletePage deep-link handoff screen"
```

---

### Task 19: Rewrite router — redirect + AuthCompleteRoute + refresh listenable

This is the biggest single edit in the plan. Takes three steps: add the route class, rewrite the redirect function, wire the `refreshListenable`.

**Files:**
- Modify: `app/lib/router/router.dart`
- Create: `app/lib/router/onboarding_refresh_listener.dart`
- Modify: `app/test/router/router_redirect_test.dart`
- Delete: `app/test/fakes/auth_status_fakes.dart`
- Create: `app/test/fakes/auth_session_fakes.dart`

- [ ] **Step 1: Write test fakes** — `app/test/fakes/auth_session_fakes.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';

class SignedOutAuthSession extends AuthSession {
  @override
  Future<AuthState> build() async => const SignedOut();
}

class SignedInAuthSession extends AuthSession {
  SignedInAuthSession({this.did = 'did:plc:test'});
  final String did;

  @override
  Future<AuthState> build() async =>
      SignedIn(did: did, handle: 'test.bsky.social', token: 'tok');
}

class PendingOnboardingStatus extends OnboardingStatus {
  @override
  bool build(String did) => false;
}

class CompletedOnboardingStatus extends OnboardingStatus {
  @override
  bool build(String did) => true;
}
```

- [ ] **Step 2: Delete the old `auth_status_fakes.dart`**

```bash
git rm app/test/fakes/auth_status_fakes.dart
```

- [ ] **Step 3: Rewrite the redirect test** — `app/test/router/router_redirect_test.dart`:

```dart
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';

Future<void> _pumpRouter(
  WidgetTester tester,
  ProviderContainer container, {
  String initialLocation = RouteLocations.welcome,
}) async {
  final router = container.read(goRouterProvider);
  // Drive the router to a specific initial location before pumping
  // the app, so deep-link-style tests can start on /auth/complete.
  router.go(initialLocation);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        routerConfig: router,
        builder: (context, child) =>
            FormFactorWidget(child: child ?? const SizedBox.shrink()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  group('router redirect', () {
    testWidgets('SignedOut + /feed → WelcomePage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
        ],
      );
      await _pumpRouter(tester, container, initialLocation: RouteLocations.feed);
      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets('SignedOut + /auth/complete stays on AuthCompletePage',
        (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '${RouteLocations.authComplete}?token=t',
      );
      expect(find.byType(AuthCompletePage), findsOneWidget);
    });

    testWidgets('SignedIn + not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(PendingOnboardingStatus.new),
        ],
      );
      await _pumpRouter(tester, container, initialLocation: RouteLocations.feed);
      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('SignedIn + onboarded + /welcome → FeedPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(CompletedOnboardingStatus.new),
        ],
      );
      await _pumpRouter(tester, container);
      expect(find.byType(FeedPage), findsOneWidget);
    });

    testWidgets('SignedIn + onboarded + /auth/complete → FeedPage',
        (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(CompletedOnboardingStatus.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '${RouteLocations.authComplete}?token=t',
      );
      expect(find.byType(FeedPage), findsOneWidget);
    });

    testWidgets('SignedIn + !onboarded + /auth/complete → OnboardingPage',
        (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(PendingOnboardingStatus.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: '${RouteLocations.authComplete}?token=t',
      );
      expect(find.byType(OnboardingPage), findsOneWidget);
    });
  });
}
```

- [ ] **Step 4: Create onboarding refresh helper** — `app/lib/router/onboarding_refresh_listener.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Attaches a per-DID `ref.listen` on `onboardingStatusProvider(did)`
/// so the router's `refreshListenable` fires when onboarding flips.
/// Call [update] whenever the `authSessionProvider` transitions; it
/// closes the previous subscription (if any) and reattaches to the
/// new DID, or tears down on sign-out.
///
/// Caller owns the lifecycle: invoke [close] from `ref.onDispose`.
class OnboardingRefreshListener {
  OnboardingRefreshListener({required this.ref, required this.onChange});

  final Ref ref;
  final VoidCallback onChange;

  ProviderSubscription<bool>? _sub;
  String? _currentDid;

  void update(AuthState? auth) {
    final newDid = switch (auth) {
      SignedIn(:final did) => did,
      _ => null,
    };
    if (newDid == _currentDid) return;

    _sub?.close();
    _sub = null;
    _currentDid = newDid;

    if (newDid != null) {
      _sub = ref.listen<bool>(
        onboardingStatusProvider(newDid),
        (_, __) => onChange(),
      );
    }
  }

  void close() {
    _sub?.close();
    _sub = null;
  }
}
```

- [ ] **Step 5: Rewrite router.dart's `goRouter` provider** — replace the entire `@riverpod GoRouter goRouter(Ref ref) { ... }` function (the top of `router.dart`, above the `// --- Shell route --` comment) with:

```dart
@riverpod
GoRouter goRouter(Ref ref) {
  final refresh = ChangeNotifier();
  final onboardingListener = OnboardingRefreshListener(
    ref: ref,
    onChange: refresh.notifyListeners,
  );

  ref.onDispose(() {
    onboardingListener.close();
    refresh.dispose();
  });

  ref.listen(authSessionProvider, (_, next) {
    refresh.notifyListeners();
    onboardingListener.update(next.valueOrNull);
  });

  return GoRouter(
    initialLocation: RouteLocations.welcome,
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    refreshListenable: refresh,
    redirect: (context, state) {
      final loc = state.matchedLocation;
      const unauthenticatedRoutes = [
        RouteLocations.welcome,
        RouteLocations.signIn,
      ];

      final auth = ref.read(authSessionProvider).valueOrNull;
      if (auth == null) return null; // transient AsyncLoading

      switch (auth) {
        case SignedOut():
          if (loc == RouteLocations.authComplete) return null;
          return unauthenticatedRoutes.contains(loc)
              ? null
              : RouteLocations.welcome;
        case SignedIn(:final did):
          final onboarded = ref.read(onboardingStatusProvider(did));
          if (loc == RouteLocations.authComplete) {
            return onboarded ? RouteLocations.home : RouteLocations.onboarding;
          }
          if (!onboarded && loc != RouteLocations.onboarding) {
            return RouteLocations.onboarding;
          }
          if (onboarded &&
              (unauthenticatedRoutes.contains(loc) ||
                  loc == RouteLocations.onboarding)) {
            return RouteLocations.home;
          }
          return null;
      }
    },
    routes: $appRoutes,
    errorBuilder: (context, state) =>
        ErrorScreen(error: state.error ?? 'Unknown routing error'),
  );
}
```

Add imports at the top:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/onboarding_refresh_listener.dart';
import 'package:flutter/foundation.dart';
```

Remove:

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
```

(the second import stays via the new import above; remove any duplicate)

- [ ] **Step 6: Add `AuthCompleteRoute` class** — add to `router.dart` alongside `WelcomeRoute` / `SignInRoute`:

```dart
@TypedGoRoute<AuthCompleteRoute>(
  path: RouteLocations.authComplete,
  name: 'auth-complete',
)
class AuthCompleteRoute extends GoRouteData with $AuthCompleteRoute {
  const AuthCompleteRoute({required this.token});

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  final String token;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      AuthCompletePage(token: token);
}
```

- [ ] **Step 7: Regenerate router code**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 8: Run tests**

```bash
cd app && flutter test test/router/router_redirect_test.dart
```

Expected: PASS — all new scenarios covered.

- [ ] **Step 9: Commit**

```bash
git add app/lib/router app/test/router app/test/fakes
git commit -m "feat(app): rewrite router redirect + AuthCompleteRoute for real auth flow"
```

---

### Task 20: Rewire `SignInPage` and `WelcomePage`

**Files:**
- Modify: `app/lib/auth/pages/welcome_page.dart`
- Modify: `app/lib/auth/pages/sign_in_page.dart`
- Modify: `app/test/auth/welcome_page_test.dart`
- Modify: `app/test/auth/sign_in_page_test.dart`

- [ ] **Step 1: Rewrite `welcome_page.dart`**

```dart
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Welcome')),
      body: const Center(child: WelcomePageBody()),
    );
  }
}

class WelcomePageBody extends ConsumerWidget {
  const WelcomePageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Welcome'),
        const SizedBox(height: 24),
        ChunkyButton(
          onPressed: () => const SignInRoute().go(context),
          child: const Text('Sign in'),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () => const SignInRoute().go(context),
          child: const Text('Create account on a PDS'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 2: Rewrite `sign_in_page.dart`**

```dart
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignInPage extends ConsumerStatefulWidget {
  const SignInPage({super.key});

  @override
  ConsumerState<SignInPage> createState() => _SignInPageState();
}

class _SignInPageState extends ConsumerState<SignInPage> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(authControllerProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncError(:final error)):
          final message = _messageFor(error);
          ScaffoldMessenger.of(context)
            ..clearSnackBars()
            ..showSnackBar(SnackBar(content: Text(message)));
        case _:
          break;
      }
    });

    final state = ref.watch(authControllerProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Sign in')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BrandTextField(
              label: 'Handle',
              hintText: 'alice.bsky.social',
              controller: _controller,
              onSubmitted: (_) => _submit(),
            ),
            const SizedBox(height: 24),
            ChunkyButton(
              onPressed: state is AsyncLoading ? null : _submit,
              child: state is AsyncLoading
                  ? const SizedBox(
                      width: 20, height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Continue'),
            ),
          ],
        ),
      ),
    );
  }

  void _submit() {
    ref.read(authControllerProvider.notifier).signIn(handle: _controller.text);
  }

  String _messageFor(Object? error) => switch (error) {
        HandleRequired() => 'Please enter a handle.',
        InvalidHandle() => "We couldn't recognise that handle.",
        ServerUnavailable() => "Couldn't reach the server. Please try again.",
        BrowserLaunchFailed() =>
          "Couldn't open the browser. Check that you have one installed.",
        _ => 'Something went wrong. Please try again.',
      };
}
```

Note: `BrandTextField` may need a `controller` and `onSubmitted` parameter if it doesn't already have them — inspect `app/lib/theme/brand_text_field.dart` first and add if missing (simple passthrough to the underlying `TextField`).

- [ ] **Step 3: Rewrite `welcome_page_test.dart`** — drop the dev-toggle assertion since that button is removed:

```dart
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('WelcomePage renders Welcome + Sign in + Create account', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const WelcomePage(),
        ),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(find.widgetWithText(ChunkyButton, 'Sign in'), findsOneWidget);
    expect(find.text('Create account on a PDS'), findsOneWidget);
  });
}
```

- [ ] **Step 4: Rewrite `sign_in_page_test.dart`** — assert the handle-submit dispatches `signIn`:

```dart
import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingAuthController extends AuthController {
  final List<String> signInCalls = [];

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> signIn({required String handle}) async {
    signInCalls.add(handle);
  }
}

void main() {
  testWidgets('renders a handle field and a Continue button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const SignInPage(),
        ),
      ),
    );
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Continue'), findsOneWidget);
  });

  testWidgets('tapping Continue dispatches AuthController.signIn with text', (tester) async {
    final fake = _RecordingAuthController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [authControllerProvider.overrideWith(() => fake)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const SignInPage(),
        ),
      ),
    );
    await tester.enterText(find.byType(BrandTextField), '  @alice.bsky.social ');
    await tester.tap(find.widgetWithText(ChunkyButton, 'Continue'));
    await tester.pump();

    expect(fake.signInCalls, ['  @alice.bsky.social ']);
    // (Controller trims — that's unit-tested in auth_controller_test.dart.)
  });
}
```

- [ ] **Step 5: Run all tests**

```bash
cd app && flutter test test/auth
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add app/lib/auth/pages app/test/auth
git commit -m "feat(app): wire SignInPage to AuthController; strip dev toggle from WelcomePage"
```

---

### Task 21: Rewire `SettingsPage` with `SignOutTile`

**Files:**
- Create: `app/lib/settings/widgets/sign_out_tile.dart`
- Modify: `app/lib/settings/pages/settings_page.dart`
- Modify: `app/test/settings/` (existing test, if any) or add one

- [ ] **Step 1: Create `SignOutTile`** — `app/lib/settings/widgets/sign_out_tile.dart`:

```dart
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignOutTile extends ConsumerWidget {
  const SignOutTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(authControllerProvider);
    return ListTile(
      leading: const Icon(Icons.logout),
      title: const Text('Sign out'),
      enabled: state is! AsyncLoading,
      onTap: () => ref.read(authControllerProvider.notifier).signOut(),
    );
  }
}
```

- [ ] **Step 2: Replace the dev toggle in `settings_page.dart`**

```dart
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: const SettingsPageBody(),
    );
  }
}

class SettingsPageBody extends ConsumerWidget {
  const SettingsPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const Column(
      children: [
        SignOutTile(),
      ],
    );
  }
}
```

- [ ] **Step 3: Widget test** — `app/test/settings/sign_out_tile_test.dart`:

```dart
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'dart:async';

import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  int signOutCalls = 0;
  @override
  FutureOr<void> build() => null;
  @override
  Future<void> signOut() async {
    signOutCalls++;
  }
}

void main() {
  testWidgets('tapping the tile calls AuthController.signOut', (tester) async {
    final fake = _FakeAuthController();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [authControllerProvider.overrideWith(() => fake)],
        child: const MaterialApp(home: Material(child: SignOutTile())),
      ),
    );
    await tester.tap(find.byType(SignOutTile));
    await tester.pump();
    expect(fake.signOutCalls, 1);
  });
}
```

- [ ] **Step 4: Run tests**

```bash
cd app && flutter test test/settings
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add app/lib/settings app/test/settings
git commit -m "feat(app): add SignOutTile + wire SettingsPage to AuthController"
```

---

### Task 22: Delete the old `authStatusProvider`

**Files:**
- Delete: `app/lib/auth/providers/auth_status_provider.dart`
- Delete: `app/lib/auth/providers/auth_status_provider.g.dart`

- [ ] **Step 1: Confirm no remaining references**

ripgrep's regex engine doesn't support lookaround, so run two narrower searches and visually check the second:

```bash
cd app && rg 'authStatusProvider|auth_status_provider' lib test
cd app && rg '\bAuthStatus\b' lib test
```

Expected: the first command returns zero matches. The second command should only match lines in `app/lib/auth/providers/auth_status_provider.dart` / `.g.dart` (about to be deleted) and nothing else.

- [ ] **Step 2: Delete**

```bash
git rm app/lib/auth/providers/auth_status_provider.dart app/lib/auth/providers/auth_status_provider.g.dart
```

- [ ] **Step 3: Run full analyze**

```bash
cd app && dart analyze lib test
```

Expected: `No issues found!`

- [ ] **Step 4: Run all tests**

```bash
cd app && flutter test
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git commit -m "chore(app): delete stub authStatusProvider"
```

---

### Task 23: iOS custom URL scheme

**Files:**
- Modify: `app/ios/Runner/Info.plist`

- [ ] **Step 1: Add `CFBundleURLTypes`** — append inside the top-level `<dict>` in `app/ios/Runner/Info.plist`:

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

- [ ] **Step 2: Smoke test on iOS simulator**

```bash
# In another terminal, with the app running on simulator:
xcrun simctl openurl booted 'craftsky://auth/complete?token=testtoken'
```

Expected: app foregrounds, lands on the `AuthCompletePage`, surfaces `NoPendingSignIn` (correct — no sign-in in progress).

- [ ] **Step 3: Commit**

```bash
git add app/ios/Runner/Info.plist
git commit -m "feat(app): register craftsky:// custom URL scheme on iOS"
```

---

### Task 24: Android custom-scheme intent filter

**Files:**
- Modify: `app/android/app/src/main/AndroidManifest.xml`

- [ ] **Step 1: Add intent-filter** — inside the existing `<activity android:name=".MainActivity">` block, alongside the existing `MAIN`/`LAUNCHER` filter:

```xml
<intent-filter>
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.DEFAULT" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data android:scheme="craftsky" android:host="auth" />
</intent-filter>
```

- [ ] **Step 2: Smoke test on Android emulator** (after running the app)

```bash
adb shell am start -W -a android.intent.action.VIEW \
  -d 'craftsky://auth/complete?token=testtoken' \
  social.craftsky.app
```

Replace `social.craftsky.app` with the actual package name if different — check `app/android/app/build.gradle.kts` for `applicationId`.

Expected: app launches or foregrounds, lands on `AuthCompletePage`.

- [ ] **Step 3: Commit**

```bash
git add app/android/app/src/main/AndroidManifest.xml
git commit -m "feat(app): register craftsky://auth intent-filter on Android"
```

---

### End-of-chunk gate

- [ ] **Run full analyzer + test suite**

```bash
cd app && dart analyze lib test
cd app && flutter test
```

Expected: `No issues found!` and all tests green (all widget + router tests now target the new providers; the stub is deleted).

---

## Chunk 5: Wire interceptor to real auth providers + final integration

Goal: close the loop between the Dio interceptor (Chunk 1 stub) and the live `authSessionProvider`/`inFlightTokenProvider`, including the central 401 sign-out behaviour.

### Task 25: Real `TokenResolver` in `dioProvider`

**Files:**
- Modify: `app/lib/shared/api/providers/dio_provider.dart`

_(`auth_interceptor_test.dart` and `error_mapping_interceptor_test.dart` from Chunk 1 continue to pass unchanged — they target the interceptors directly via injected resolver / onUnauthorized callbacks, independent of which providers we wire up here.)_

- [ ] **Step 1: Replace the stub resolver** — update `dio_provider.dart`:

```dart
import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../auth/models/auth_state.dart';
import '../../../auth/providers/auth_session_provider.dart';
import '../../../auth/providers/in_flight_token_provider.dart';
import '../../../auth/providers/secure_token_storage.dart';
import 'auth_interceptor.dart';
import 'error_mapping_interceptor.dart';

part 'dio_provider.g.dart';

const _devDefaultBaseUrl = 'http://10.0.2.2:8080';
const _baseUrl = String.fromEnvironment(
  'CRAFTSKY_API_BASE_URL',
  defaultValue: kDebugMode ? _devDefaultBaseUrl : '',
);

@Riverpod(keepAlive: true)
Dio dio(Ref ref) {
  if (_baseUrl.isEmpty) {
    throw StateError(
      'CRAFTSKY_API_BASE_URL must be set for non-debug builds. '
      'Pass it via --dart-define.',
    );
  }

  final dio = Dio(BaseOptions(
    baseUrl: _baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 15),
    headers: {'Content-Type': 'application/json'},
  ));

  dio.interceptors.addAll([
    AuthInterceptor(ref, _resolveToken),
    ErrorMappingInterceptor(
      onUnauthorized: (options) => _handleUnauthorized(ref, options),
    ),
  ]);

  return dio;
}

String? _resolveToken(Ref ref) {
  final inFlight = ref.read(inFlightTokenProvider);
  if (inFlight != null) return inFlight;

  final auth = ref.read(authSessionProvider).value;
  return switch (auth) {
    SignedIn(:final token) => token,
    _ => null,
  };
}

void _handleUnauthorized(Ref ref, RequestOptions options) {
  // Carve-out: a 401 during the handoff `whoami` is a sign-in failure,
  // not a session-expiry. AuthController surfaces it as an AuthError.
  if (options.path == '/v1/whoami' &&
      ref.read(inFlightTokenProvider) != null) {
    return;
  }
  // Fire-and-forget: don't block the error-response pipeline.
  unawaited(ref.read(secureTokenStorageProvider).clear());
  ref.read(authSessionProvider.notifier).setSignedOut();
}
```

- [ ] **Step 2: Regenerate**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 3: Run all tests**

```bash
cd app && flutter test
```

Expected: all PASS (existing tests keep working, behaviour is now wired).

- [ ] **Step 4: Commit**

```bash
git add app/lib/shared/api/providers/dio_provider.dart app/lib/shared/api/providers/dio_provider.g.dart
git commit -m "feat(app): wire dio token resolver + global 401 sign-out to real auth providers"
```

---

### Task 26: Integration test — 401 on authenticated call signs user out

**Files:**
- Create: `app/test/shared/api/providers/dio_unauthorized_test.dart`

- [ ] **Step 1: Write the test**

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/in_flight_token_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

class _FakeStorage implements SecureTokenStorage {
  StoredSession? _v = const StoredSession(token: 't', did: 'd', handle: 'h');
  @override
  Future<StoredSession?> read() async => _v;
  @override
  Future<void> write(StoredSession s) async => _v = s;
  @override
  Future<void> clear() async => _v = null;
}

void main() {
  setUpAll(initializeMappers);

  test('401 on /v1/whoami WITHOUT in-flight token signs user out', () async {
    final storage = _FakeStorage();
    final container = ProviderContainer(
      overrides: [secureTokenStorageProvider.overrideWithValue(storage)],
    );
    addTearDown(container.dispose);

    // Resolve build so setSignedOut can mutate state.
    await container.read(authSessionProvider.future);
    container.read(authSessionProvider.notifier).setSignedIn(
          const SignedIn(did: 'd', handle: 'h', token: 't'),
        );

    final dio = container.read(dioProvider);
    final adapter = DioAdapter(dio: dio);
    adapter.onGet('/v1/whoami', (s) => s.reply(401, {}));

    await expectLater(
      () => dio.get<dynamic>('/v1/whoami'),
      throwsA(isA<ApiUnauthorized>()),
    );

    expect(container.read(authSessionProvider).value, isA<SignedOut>());
    expect(await storage.read(), isNull);
  });

  test('401 on /v1/whoami WITH in-flight token does NOT sign user out', () async {
    final storage = _FakeStorage();
    final container = ProviderContainer(
      overrides: [secureTokenStorageProvider.overrideWithValue(storage)],
    );
    addTearDown(container.dispose);

    await container.read(authSessionProvider.future);
    container.read(authSessionProvider.notifier).setSignedIn(
          const SignedIn(did: 'd', handle: 'h', token: 't'),
        );
    container.read(inFlightTokenProvider.notifier).setToken('handoff-tok');

    final dio = container.read(dioProvider);
    final adapter = DioAdapter(dio: dio);
    adapter.onGet('/v1/whoami', (s) => s.reply(401, {}));

    await expectLater(
      () => dio.get<dynamic>('/v1/whoami'),
      throwsA(isA<ApiUnauthorized>()),
    );

    expect(container.read(authSessionProvider).value, isA<SignedIn>());
    expect(await storage.read(), isNotNull);
  });
}
```

- [ ] **Step 2: Run**

```bash
cd app && flutter test test/shared/api/providers/dio_unauthorized_test.dart
```

Expected: PASS, 2 tests.

- [ ] **Step 3: Commit**

```bash
git add app/test/shared/api/providers/dio_unauthorized_test.dart
git commit -m "test(app): cover global 401 sign-out + handoff carve-out"
```

---

### Task 27: Update `app/README.md` — Deep links + Dev setup

_(No separate "bootstrap fail-fast" task: `dioProvider`'s `StateError` surfaces on first read, which happens during router build at app start — the failure mode is effectively startup-time already. Document it in the README below.)_

**Files:**
- Modify: `app/README.md`

- [ ] **Step 1: Add sections**:

```markdown
## Dev setup

### Base URL

The app talks to the AppView via `CRAFTSKY_API_BASE_URL`. In debug builds the
default is `http://10.0.2.2:8080` (Android emulator → host). iOS simulator
runs need an override:

```bash
flutter run --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:8080
```

Release builds **require** `--dart-define`; the app throws on first API call
if it's missing.

## Deep links

The app registers `craftsky://` as a custom URL scheme. The OAuth flow lands
on `craftsky://auth/complete?token=…` after the user authenticates at their
PDS. Smoke tests:

```bash
# iOS simulator
xcrun simctl openurl booted 'craftsky://auth/complete?token=testtoken'

# Android emulator (replace the package name if applicationId differs)
adb shell am start -W -a android.intent.action.VIEW \
  -d 'craftsky://auth/complete?token=testtoken' \
  social.craftsky.app
```

Both should land on the "Signing in…" screen and surface a `NoPendingSignIn`
error (since no sign-in is in progress — correct behaviour for a bare link).
```

- [ ] **Step 2: Commit**

```bash
git add app/README.md
git commit -m "docs(app): add Dev setup + Deep links sections"
```

---

### Task 28: Final smoke test run — full manual verification

- [ ] **Step 1: Boot docker-compose stack**

```bash
cd / && just dev
```

Expected: postgres + appview up at `http://localhost:8080`.

- [ ] **Step 2: Run app on iOS simulator**

```bash
cd app && flutter run -d 'iPhone' \
  --dart-define=CRAFTSKY_API_BASE_URL=http://localhost:8080
```

- [ ] **Step 3: Sign in with a test handle**

Enter a valid test handle. Browser opens, complete PDS sign-in, returns to app via `craftsky://auth/complete?token=…`. Assert lands on `/feed`.

- [ ] **Step 4: Cold relaunch — still signed in**

Kill the app, reopen. Assert feed loads without re-auth.

- [ ] **Step 5: Airplane mode relaunch**

Enable airplane mode, kill + relaunch. Assert still signed in (background `whoami` fails quietly).

- [ ] **Step 6: Settings → Sign out → welcome page**

Navigate to settings, tap Sign out. Assert lands on `/welcome`. Relaunch → still signed out.

- [ ] **Step 7: Error paths**

Enter empty handle → `HandleRequired` message. Stop docker-compose, try sign-in → `ServerUnavailable`.

- [ ] **Step 8: Mid-session revocation**

Sign in. In a separate shell: `just psql` then `UPDATE craftsky_sessions SET revoked_at = now() WHERE ...;`. Trigger a refresh in the app. Assert cleanly transitions to `/welcome`.

- [ ] **Step 9: Android emulator — repeat all of the above**

- [ ] **Step 10: Final analyzer + test run**

```bash
cd app && dart analyze lib test && flutter test
```

Expected: `No issues found!` + all tests green.

- [ ] **Step 11: Create PR**

Create a pull request to main with the full diff. Include the smoke-test checklist above as PR description.

---

## Done

After Task 28 passes, the spec's acceptance criteria are met. The branch is ready to merge.
