import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/api_client_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// --- Fakes (services, not notifiers — per riverpod.md Testing rules) ---

class _FakeStorage implements SecureTokenStorage {
  StoredSession? _v;
  @override
  Future<StoredSession?> read() async => _v;
  @override
  Future<void> write(StoredSession s) async => _v = s;
  @override
  Future<void> clear() async => _v = null;
}

class _FakeCraftskyApi implements CraftskyApiClient {
  // Allow current tests to omit onWhoami (the default in [whoami] covers
  // the AuthSession background-validation path); future tests can stub it.
  _FakeCraftskyApi({
    this.onLogin,
    this.onLogout,
    // Reserved for future tests that want a non-default whoami.
    // ignore: unused_element_parameter
    this.onWhoami,
  });
  final Future<LoginResponse> Function(String)? onLogin;
  final Future<void> Function()? onLogout;
  final Future<WhoAmI> Function()? onWhoami;

  @override
  Future<LoginResponse> login({required String handle}) =>
      onLogin?.call(handle) ?? Future.error(UnimplementedError());
  @override
  Future<WhoAmI> whoami() =>
      onWhoami?.call() ??
      // AuthSession.build's _validateInBackground calls this on cold
      // start. Default to a handle/did that matches the stored session
      // used in the sign-out test, so validation is a no-op.
      Future.value(const WhoAmI(did: 'd', handle: 'h'));
  @override
  Future<void> logout() => onLogout?.call() ?? Future.value();
}

class _FakeHandoffApi implements HandoffApiClient {
  _FakeHandoffApi(this.onWhoami);
  final Future<WhoAmI> Function() onWhoami;
  @override
  Future<WhoAmI> whoami() => onWhoami();
}

class _LaunchRecorder {
  final List<Uri> launched = [];
  bool nextResult = true;
  Future<bool> launch(Uri uri) async {
    launched.add(uri);
    return nextResult;
  }
}

ProviderContainer _container({
  _FakeStorage? storage,
  _FakeCraftskyApi? api,
  _FakeHandoffApi? handoff,
  _LaunchRecorder? launch,
}) {
  storage ??= _FakeStorage();
  api ??= _FakeCraftskyApi();
  launch ??= _LaunchRecorder();
  return ProviderContainer.test(
    overrides: [
      secureTokenStorageProvider.overrideWithValue(storage),
      craftskyApiClientProvider.overrideWithValue(api),
      launchAuthUrlProvider.overrideWithValue(launch.launch),
      // Override the handoff family for ANY token value — the test
      // passes a specific token and the override serves that.
      if (handoff != null)
        handoffApiClientProvider.overrideWith(
          (ref, token) => handoff,
        ),
    ],
  );
}

void main() {
  setUpAll(initializeMappers);

  test('signIn trims handle + @ prefix and posts to /login', () async {
    final launch = _LaunchRecorder();
    final api = _FakeCraftskyApi(
      onLogin: (h) async {
        expect(h, 'alice.bsky.social');
        return const LoginResponse(authUrl: 'https://pds.example.com/a?b=1');
      },
    );
    final container = _container(api: api, launch: launch);

    await container
        .read(authControllerProvider.notifier)
        .signIn(handle: '  @alice.bsky.social  ');

    expect(launch.launched, hasLength(1));
    expect(launch.launched.single.toString(), 'https://pds.example.com/a?b=1');
  });

  test('signIn with empty handle surfaces HandleRequired', () async {
    final container = _container();
    await container.read(authControllerProvider.notifier).signIn(handle: '');
    expect(
      container.read(authControllerProvider).error,
      isA<HandleRequired>(),
    );
  });

  test('signIn maps ApiBadRequest(handle_required) → HandleRequired',
      () async {
    final container = _container(
      api: _FakeCraftskyApi(
        onLogin: (_) async => throw const ApiBadRequest('handle_required'),
      ),
    );
    await container
        .read(authControllerProvider.notifier)
        .signIn(handle: 'a.bsky.social');
    expect(
      container.read(authControllerProvider).error,
      isA<HandleRequired>(),
    );
  });

  test('signIn maps ApiNetworkError → ServerUnavailable', () async {
    final container = _container(
      api: _FakeCraftskyApi(
        onLogin: (_) async => throw const ApiNetworkError('offline'),
      ),
    );
    await container
        .read(authControllerProvider.notifier)
        .signIn(handle: 'a.bsky.social');
    expect(
      container.read(authControllerProvider).error,
      isA<ServerUnavailable>(),
    );
  });

  test(
    'signIn surfaces BrowserLaunchFailed when launchUrl returns false',
    () async {
      final launch = _LaunchRecorder()..nextResult = false;
      final container = _container(
        api: _FakeCraftskyApi(
          onLogin: (_) async => const LoginResponse(authUrl: 'https://x'),
        ),
        launch: launch,
      );
      await container
          .read(authControllerProvider.notifier)
          .signIn(handle: 'a.bsky.social');
      expect(
        container.read(authControllerProvider).error,
        isA<BrowserLaunchFailed>(),
      );
      // Pending was started then cleared on browser-launch failure.
      expect(container.read(pendingAuthProvider), isNull);
    },
  );

  test(
    'completeFromDeepLink with no pending surfaces NoPendingSignIn',
    () async {
      final container = _container();
      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('tok');
      expect(
        container.read(authControllerProvider).error,
        isA<NoPendingSignIn>(),
      );
    },
  );

  test('completeFromDeepLink stale pending surfaces SignInTimedOut',
      () async {
    final container = _container();
    container.read(pendingAuthProvider.notifier).debugSet(
          model.PendingAuth(
            handle: 'a.bsky.social',
            startedAt: DateTime.now().subtract(const Duration(minutes: 15)),
          ),
        );

    await container
        .read(authControllerProvider.notifier)
        .completeFromDeepLink('tok');
    expect(
      container.read(authControllerProvider).error,
      isA<SignInTimedOut>(),
    );
    // Pending cleared on timeout.
    expect(container.read(pendingAuthProvider), isNull);
  });

  test(
    'completeFromDeepLink happy path writes storage + flips SignedIn',
    () async {
      final storage = _FakeStorage();
      final handoff = _FakeHandoffApi(
        () async => const WhoAmI(did: 'did:plc:a', handle: 'a.bsky.social'),
      );
      final container = _container(storage: storage, handoff: handoff);

      // Seed AuthSession build so setSignedIn lands on a ready state.
      await container.read(authSessionProvider.future);
      container.read(pendingAuthProvider.notifier).start('a.bsky.social');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('tok');

      final state = container.read(authSessionProvider).value;
      expect(state, isA<SignedIn>());
      expect((state! as SignedIn).did, 'did:plc:a');

      final stored = await storage.read();
      expect(stored?.token, 'tok');
      expect(stored?.did, 'did:plc:a');
      expect(container.read(pendingAuthProvider), isNull);
    },
  );

  test(
    'completeFromDeepLink whoami failure clears pending, leaves storage empty',
    () async {
      final storage = _FakeStorage();
      final handoff = _FakeHandoffApi(
        () async => throw const ApiUnauthorized(),
      );
      final container = _container(storage: storage, handoff: handoff);

      await container.read(authSessionProvider.future);
      container.read(pendingAuthProvider.notifier).start('a.bsky.social');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('tok');

      expect(await storage.read(), isNull);
      expect(container.read(pendingAuthProvider), isNull);
      expect(
        container.read(authControllerProvider).error,
        isA<ApiUnauthorized>(),
      );
    },
  );

  test(
    'signOut clears storage + flips SignedOut even on server failure',
    () async {
      final storage = _FakeStorage();
      await storage.write(
        const StoredSession(token: 't', did: 'd', handle: 'h'),
      );
      final container = _container(
        storage: storage,
        api: _FakeCraftskyApi(
          onLogout: () async => throw const ApiNetworkError('offline'),
        ),
      );

      await container.read(authSessionProvider.future);
      await container.read(authControllerProvider.notifier).signOut();

      expect(await storage.read(), isNull);
      expect(container.read(authSessionProvider).value, isA<SignedOut>());
    },
  );
}
