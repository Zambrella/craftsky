import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// --- Fakes (services, not notifiers — per riverpod.md Testing rules) ---

class _FakeRegistryStorage implements SessionRegistryStorage {
  _FakeRegistryStorage(this.value);

  SessionRegistry value;
  bool failWrites = false;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {
    if (failWrites) {
      throw const SessionRegistryStorageException('writeFailed');
    }
    value = registry;
  }
}

class _FakeAuthApi implements AuthApiClient {
  // Allow current tests to omit onWhoami (the default in [whoami] covers
  // the AuthSession background-validation path); future tests can stub it.
  _FakeAuthApi({
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
      Future.value(WhoAmI(did: 'did:plc:test', handle: 'h.test'));
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
  _FakeAuthApi? api,
  _FakeHandoffApi? handoff,
  _LaunchRecorder? launch,
  _FakeRegistryStorage? registryStorage,
  Future<void> Function()? invalidateAccountState,
  Future<void> Function()? resetToHome,
}) {
  final resolvedApi = api ?? _FakeAuthApi();
  launch ??= _LaunchRecorder();
  return ProviderContainer.test(
    overrides: [
      if (registryStorage != null)
        secureSessionRegistryStorageProvider.overrideWithValue(registryStorage),
      sessionValidationLauncherProvider.overrideWithValue((_) async {}),
      authApiClientProvider.overrideWithValue(resolvedApi),
      accountAuthApiClientProvider.overrideWith(
        (ref, account) async => resolvedApi,
      ),
      authUrlLauncherProvider.overrideWithValue(launch.launch),
      // Override the handoff family for ANY (token, deviceId) — the
      // test passes a specific token and the override serves that.
      // riverpod_generator 4.x packs multi-arg families into a single
      // positional record ((String, String) here), so the override
      // signature is `(ref, (token, deviceId))`.
      if (handoff != null)
        handoffApiClientProvider.overrideWith(
          (ref, args) => handoff,
        ),
      // Stub deviceIdProvider so completeFromDeepLink doesn't touch
      // the real FlutterSecureStorage (unavailable in unit tests).
      deviceIdProvider.overrideWith((ref) async => 'test-device-id'),
      if (invalidateAccountState != null)
        accountStateInvalidatorProvider.overrideWithValue(
          invalidateAccountState,
        ),
      if (resetToHome != null)
        accountHomeResetProvider.overrideWithValue(resetToHome),
    ],
  );
}

void main() {
  setUpAll(initializeMappers);

  test('signIn trims handle + @ prefix and posts to /login', () async {
    final launch = _LaunchRecorder();
    final api = _FakeAuthApi(
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

  test('signIn maps ApiBadRequest(handle_required) → HandleRequired', () async {
    final container = _container(
      api: _FakeAuthApi(
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
      api: _FakeAuthApi(
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
        api: _FakeAuthApi(
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

  test('completeFromDeepLink stale pending surfaces SignInTimedOut', () async {
    final container = _container();
    container
        .read(pendingAuthProvider.notifier)
        .debugSet(
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
      final registryStorage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        () async => WhoAmI(did: 'did:plc:a', handle: 'a.bsky.social'),
      );
      final container = _container(
        handoff: handoff,
        registryStorage: registryStorage,
      );

      // Seed AuthSession build so setSignedIn lands on a ready state.
      await container.read(authSessionProvider.future);
      container.read(pendingAuthProvider.notifier).start('a.bsky.social');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('tok');

      final state = await container.read(authSessionProvider.future);
      expect(state, isA<SignedIn>());
      expect((state as SignedIn).did, 'did:plc:a');

      final stored = await registryStorage.read();
      expect(stored.sessions['did:plc:a']?.token, 'tok');
      expect(stored.activeDid, 'did:plc:a');
      expect(container.read(pendingAuthProvider), isNull);
    },
  );

  test('Add account completion preserves A and activates new B', () async {
    final boundaryEvents = <String>[];
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final registryStorage = _FakeRegistryStorage(initial);
    final handoff = _FakeHandoffApi(
      () async => WhoAmI(did: 'did:plc:bob', handle: 'bob.test'),
    );
    final container = _container(
      handoff: handoff,
      registryStorage: registryStorage,
      invalidateAccountState: () async => boundaryEvents.add('invalidate'),
    );
    await container.read(sessionRegistryProvider.future);
    container.read(pendingAuthProvider.notifier).start('bob.test');

    await container
        .read(authControllerProvider.notifier)
        .completeFromDeepLink('token-b');

    final registry = container.read(sessionRegistryProvider).requireValue;
    expect(registry.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
    expect(registry.activeDid, 'did:plc:bob');
    expect(registry.sessions['did:plc:bob']?.token, 'token-b');
    expect(boundaryEvents, ['invalidate']);
    expect(container.read(pendingAuthProvider), isNull);
  });

  test('Add account storage failure preserves the entire A snapshot', () async {
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final registryStorage = _FakeRegistryStorage(initial)..failWrites = true;
    final container = _container(
      handoff: _FakeHandoffApi(
        () async => WhoAmI(did: 'did:plc:bob', handle: 'bob.test'),
      ),
      registryStorage: registryStorage,
    );
    await container.read(sessionRegistryProvider.future);
    container.read(pendingAuthProvider.notifier).start('bob.test');

    await container
        .read(authControllerProvider.notifier)
        .completeFromDeepLink('token-b');

    final registry = container.read(sessionRegistryProvider).requireValue;
    expect(registry.toJson(), initial.toJson());
    expect(container.read(authControllerProvider).error, isA<StorageFailure>());
    expect(container.read(pendingAuthProvider), isNull);
  });

  test(
    'completeFromDeepLink whoami failure clears pending, leaves storage empty',
    () async {
      final registryStorage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        () async => throw const ApiUnauthorized(),
      );
      final container = _container(
        registryStorage: registryStorage,
        handoff: handoff,
      );

      await container.read(authSessionProvider.future);
      container.read(pendingAuthProvider.notifier).start('a.bsky.social');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('tok');

      expect(registryStorage.value.sessions, isEmpty);
      expect(container.read(pendingAuthProvider), isNull);
      expect(
        container.read(authControllerProvider).error,
        isA<ApiUnauthorized>(),
      );
    },
  );

  test(
    'confirmed signOut removes only active A and selects retained B',
    () async {
      final events = <String>[];
      final initial = SessionRegistry.empty()
          .upsertAndActivate(
            token: 'token-b',
            did: 'did:plc:bob',
            handle: 'bob.test',
          )
          .upsertAndActivate(
            token: 'token-a',
            did: 'did:plc:alice',
            handle: 'alice.test',
          );
      final registryStorage = _FakeRegistryStorage(initial);
      final container = _container(
        api: _FakeAuthApi(onLogout: () async => events.add('server-logout')),
        registryStorage: registryStorage,
        invalidateAccountState: () async => events.add('invalidate-account'),
        resetToHome: () async => events.add('home'),
      );

      await container.read(authSessionProvider.future);
      final result = await container
          .read(authControllerProvider.notifier)
          .signOut();

      expect(events, ['server-logout', 'invalidate-account', 'home']);
      expect(result?.activeHandle, 'bob.test');
      final registry = container.read(sessionRegistryProvider).requireValue;
      expect(registry.sessions.keys, {'did:plc:bob'});
      expect(registry.activeDid, 'did:plc:bob');
      expect(
        (await container.read(authSessionProvider.future) as SignedIn).did,
        'did:plc:bob',
      );
    },
  );

  test('confirmed signOut of the last account projects SignedOut', () async {
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final container = _container(
      registryStorage: _FakeRegistryStorage(initial),
      api: _FakeAuthApi(),
    );
    await container.read(authSessionProvider.future);

    final result = await container
        .read(authControllerProvider.notifier)
        .signOut();

    expect(result?.activeHandle, isNull);
    expect(await container.read(authSessionProvider.future), isA<SignedOut>());
    expect(
      container.read(sessionRegistryProvider).requireValue.sessions,
      isEmpty,
    );
  });

  test(
    'SIM-UT-002 offline signOut keeps the active account for retry',
    () async {
      final events = <String>[];
      final initial = SessionRegistry.empty().upsertAndActivate(
        token: 't',
        did: 'did:plc:test',
        handle: 'h.test',
      );
      final registryStorage = _FakeRegistryStorage(initial);
      final container = _container(
        registryStorage: registryStorage,
        api: _FakeAuthApi(
          onLogout: () async {
            events.add('server-logout');
            throw const ApiNetworkError('offline');
          },
        ),
        invalidateAccountState: () async => events.add('invalidate-account'),
      );

      await container.read(authSessionProvider.future);
      final result = await container
          .read(authControllerProvider.notifier)
          .signOut();

      final registry = container.read(sessionRegistryProvider).requireValue;
      expect(result, isNull);
      expect(registry.toJson(), initial.toJson());
      expect(registry.activeDid, 'did:plc:test');
      expect(events, ['server-logout']);
      expect(
        container.read(authControllerProvider).error,
        isA<ApiNetworkError>(),
      );
      expect(await container.read(authSessionProvider.future), isA<SignedIn>());
    },
  );
}
