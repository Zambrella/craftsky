import 'dart:async';

import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:craftsky_app/auth/models/pending_handoff.dart';
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
  int? failOnWrite;
  int writeCount = 0;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async {
    writeCount++;
    if (failWrites || writeCount == failOnWrite) {
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
  _FakeHandoffApi({this.onExchange, this.onConfirm});

  final Future<PendingHandoff> Function(String code)? onExchange;
  final Future<void> Function(String token, String receiptId)? onConfirm;
  final List<String> exchangedCodes = [];
  final List<({String token, String receiptId})> confirmations = [];

  @override
  Future<PendingHandoff> exchange({required String code}) {
    exchangedCodes.add(code);
    return onExchange?.call(code) ?? Future.error(UnimplementedError());
  }

  @override
  Future<void> confirm({required String token, required String receiptId}) {
    confirmations.add((token: token, receiptId: receiptId));
    return onConfirm?.call(token, receiptId) ?? Future.value();
  }
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
  Future<void> Function()? clearSessionPrivateState,
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
      if (handoff != null) handoffApiClientProvider.overrideWithValue(handoff),
      // Stub deviceIdProvider so completeFromDeepLink doesn't touch
      // the real FlutterSecureStorage (unavailable in unit tests).
      deviceIdProvider.overrideWith((ref) async => 'test-device-id'),
      if (invalidateAccountState != null)
        accountStateInvalidatorProvider.overrideWithValue(
          invalidateAccountState,
        ),
      if (clearSessionPrivateState != null)
        accountSessionPrivateStateCleanerProvider.overrideWithValue(
          (_) => clearSessionPrivateState(),
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

  test('process restart can exchange a device-bound code', () async {
    final storage = _FakeRegistryStorage(SessionRegistry.empty());
    final handoff = _FakeHandoffApi(
      onExchange: (_) async => PendingHandoff(
        token: 'restarted-browser-token',
        did: 'did:plc:browser-restart',
        handle: 'browser-restart.test',
        receiptId: '00000000-0000-4000-8000-000000000819',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      ),
    );
    final container = _container(handoff: handoff, registryStorage: storage);

    await container
        .read(authControllerProvider.notifier)
        .completeFromDeepLink('restart-browser-code');

    expect(handoff.exchangedCodes, ['restart-browser-code']);
    expect(storage.value.activeDid, 'did:plc:browser-restart');
  });

  test('client clock does not reject a server-valid handoff', () async {
    final storage = _FakeRegistryStorage(SessionRegistry.empty());
    final handoff = _FakeHandoffApi(
      onExchange: (_) async => PendingHandoff(
        token: 'server-valid-token',
        did: 'did:plc:valid',
        handle: 'valid.test',
        receiptId: '00000000-0000-4000-8000-000000000818',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      ),
    );
    final container = _container(handoff: handoff, registryStorage: storage);
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
        .completeFromDeepLink('server-valid-code');

    expect(handoff.exchangedCodes, ['server-valid-code']);
    expect(storage.value.activeDid, 'did:plc:valid');
  });

  test(
    'completeFromDeepLink happy path writes storage + flips SignedIn',
    () async {
      final registryStorage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        onExchange: (_) async => PendingHandoff(
          token: 'pending-bearer',
          did: 'did:plc:a',
          handle: 'a.bsky.social',
          receiptId: '00000000-0000-4000-8000-000000000811',
          confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
        ),
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
          .completeFromDeepLink('browser-code');

      final state = await container.read(authSessionProvider.future);
      expect(state, isA<SignedIn>());
      expect((state as SignedIn).did, 'did:plc:a');

      final stored = await registryStorage.read();
      expect(stored.sessions['did:plc:a']?.token, 'pending-bearer');
      expect(stored.activeDid, 'did:plc:a');
      expect(stored.pendingHandoff, isNull);
      expect(handoff.exchangedCodes, ['browser-code']);
      expect(handoff.confirmations, [
        (
          token: 'pending-bearer',
          receiptId: '00000000-0000-4000-8000-000000000811',
        ),
      ]);
      expect(container.read(pendingAuthProvider), isNull);
    },
  );

  test(
    'lost exchange response retries the same code without partial state',
    () async {
      var attempts = 0;
      final storage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        onExchange: (_) async {
          attempts++;
          if (attempts == 1) throw const ApiNetworkError('response_lost');
          return PendingHandoff(
            token: 'same-pending-bearer',
            did: 'did:plc:retry',
            handle: 'retry.test',
            receiptId: '00000000-0000-4000-8000-000000000814',
            confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
          );
        },
      );
      final container = _container(handoff: handoff, registryStorage: storage);
      await container.read(sessionRegistryProvider.future);
      container.read(pendingAuthProvider.notifier).start('retry.test');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('same-browser-code');

      expect(
        container.read(authControllerProvider).error,
        isA<ServerUnavailable>(),
      );
      expect(container.read(pendingAuthProvider), isNotNull);
      expect(storage.value.pendingHandoff, isNull);

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('same-browser-code');

      expect(handoff.exchangedCodes, [
        'same-browser-code',
        'same-browser-code',
      ]);
      expect(handoff.confirmations, hasLength(1));
      expect(storage.value.activeDid, 'did:plc:retry');
      expect(storage.value.pendingHandoff, isNull);
    },
  );

  test(
    'lost confirmation response retries the durable receipt idempotently',
    () async {
      var confirmations = 0;
      final storage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        onExchange: (_) async => PendingHandoff(
          token: 'pending-confirm-token',
          did: 'did:plc:confirm',
          handle: 'confirm.test',
          receiptId: '00000000-0000-4000-8000-000000000815',
          confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
        ),
        onConfirm: (_, _) async {
          confirmations++;
          if (confirmations == 1) {
            throw const ApiNetworkError('confirmation_response_lost');
          }
        },
      );
      final container = _container(handoff: handoff, registryStorage: storage);
      await container.read(sessionRegistryProvider.future);
      container.read(pendingAuthProvider.notifier).start('confirm.test');

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('one-browser-code');

      expect(
        container.read(authControllerProvider).error,
        isA<ServerUnavailable>(),
      );
      expect(storage.value.sessions, isEmpty);
      expect(storage.value.pendingHandoff?.token, 'pending-confirm-token');
      expect(container.read(pendingAuthProvider), isNull);

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('one-browser-code');

      expect(handoff.exchangedCodes, ['one-browser-code']);
      expect(handoff.confirmations, hasLength(2));
      expect(storage.value.pendingHandoff, isNull);
      expect(storage.value.activeDid, 'did:plc:confirm');
    },
  );

  test(
    'confirmed receipt remains retryable when final local storage fails',
    () async {
      final storage = _FakeRegistryStorage(SessionRegistry.empty())
        ..failOnWrite = 2;
      final handoff = _FakeHandoffApi(
        onExchange: (_) async => PendingHandoff(
          token: 'durable-pending-token',
          did: 'did:plc:storage-retry',
          handle: 'storage-retry.test',
          receiptId: '00000000-0000-4000-8000-000000000822',
          confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
        ),
      );
      final container = _container(handoff: handoff, registryStorage: storage);

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('storage-retry-code');

      expect(
        container.read(authControllerProvider).error,
        isA<StorageFailure>(),
      );
      expect(storage.value.sessions, isEmpty);
      expect(storage.value.pendingHandoff?.token, 'durable-pending-token');
      expect(handoff.confirmations, hasLength(1));

      await container
          .read(authControllerProvider.notifier)
          .completeFromDeepLink('storage-retry-code');

      expect(handoff.exchangedCodes, ['storage-retry-code']);
      expect(handoff.confirmations, hasLength(2));
      expect(storage.value.pendingHandoff, isNull);
      expect(storage.value.activeDid, 'did:plc:storage-retry');
    },
  );

  test('restart resumes a stored receipt without the browser code', () async {
    final pending = PendingHandoff(
      token: 'restart-pending-token',
      did: 'did:plc:restart',
      handle: 'restart.test',
      receiptId: '00000000-0000-4000-8000-000000000816',
      confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
    );
    final storage = _FakeRegistryStorage(
      SessionRegistry.empty().stageHandoff(pending),
    );
    final handoff = _FakeHandoffApi();
    final container = _container(handoff: handoff, registryStorage: storage);

    await container
        .read(authControllerProvider.notifier)
        .resumePendingHandoff();

    expect(handoff.exchangedCodes, isEmpty);
    expect(handoff.confirmations, [
      (token: pending.token, receiptId: pending.receiptId),
    ]);
    expect(storage.value.pendingHandoff, isNull);
    expect(storage.value.activeDid, 'did:plc:restart');
  });

  test(
    'server-invalid stored receipt is discarded without activation',
    () async {
      final pending = PendingHandoff(
        token: 'expired-pending-token',
        did: 'did:plc:expired',
        handle: 'expired.test',
        receiptId: '00000000-0000-4000-8000-000000000823',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      );
      final storage = _FakeRegistryStorage(
        SessionRegistry.empty().stageHandoff(pending),
      );
      final handoff = _FakeHandoffApi(
        onConfirm: (_, _) async => throw const ApiBadRequest('invalid_handoff'),
      );
      final container = _container(handoff: handoff, registryStorage: storage);

      await container
          .read(authControllerProvider.notifier)
          .resumePendingHandoff();

      expect(storage.value.pendingHandoff, isNull);
      expect(storage.value.sessions, isEmpty);
      expect(storage.value.activeDid, isNull);
      expect(
        container.read(authControllerProvider).error,
        isA<SignInTimedOut>(),
      );
    },
  );

  test('concurrent recovery shares one confirmation operation', () async {
    final pending = PendingHandoff(
      token: 'concurrent-pending-token',
      did: 'did:plc:concurrent',
      handle: 'concurrent.test',
      receiptId: '00000000-0000-4000-8000-000000000820',
      confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
    );
    final storage = _FakeRegistryStorage(
      SessionRegistry.empty().stageHandoff(pending),
    );
    final confirmationStarted = Completer<void>();
    final releaseConfirmation = Completer<void>();
    final handoff = _FakeHandoffApi(
      onConfirm: (_, _) async {
        if (!confirmationStarted.isCompleted) confirmationStarted.complete();
        await releaseConfirmation.future;
      },
    );
    final container = _container(handoff: handoff, registryStorage: storage);
    final controller = container.read(authControllerProvider.notifier);

    final coldStartRecovery = controller.resumePendingHandoff();
    await confirmationStarted.future;
    final deepLinkRecovery = controller.completeFromDeepLink('unused-code');
    await Future<void>.delayed(Duration.zero);

    expect(handoff.confirmations, hasLength(1));

    releaseConfirmation.complete();
    await Future.wait([coldStartRecovery, deepLinkRecovery]);
    expect(storage.value.pendingHandoff, isNull);
    expect(storage.value.activeDid, 'did:plc:concurrent');
    expect(container.read(authControllerProvider).hasError, isFalse);
  });

  test('Add account completion preserves A and activates new B', () async {
    final boundaryEvents = <String>[];
    final initial = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final registryStorage = _FakeRegistryStorage(initial);
    final handoff = _FakeHandoffApi(
      onExchange: (_) async => PendingHandoff(
        token: 'token-b',
        did: 'did:plc:bob',
        handle: 'bob.test',
        receiptId: '00000000-0000-4000-8000-000000000812',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      ),
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
    final handoff = _FakeHandoffApi(
      onExchange: (_) async => PendingHandoff(
        token: 'token-b',
        did: 'did:plc:bob',
        handle: 'bob.test',
        receiptId: '00000000-0000-4000-8000-000000000813',
        confirmBy: DateTime.utc(2026, 8, 14, 12, 5),
      ),
    );
    final container = _container(
      handoff: handoff,
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
    expect(container.read(pendingAuthProvider), isNotNull);
    expect(handoff.confirmations, isEmpty);
  });

  test(
    'invalid handoff clears pending and leaves storage empty',
    () async {
      final registryStorage = _FakeRegistryStorage(SessionRegistry.empty());
      final handoff = _FakeHandoffApi(
        onExchange: (_) async => throw const ApiBadRequest('invalid_handoff'),
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
        isA<SignInTimedOut>(),
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
        clearSessionPrivateState: () async => events.add('clear-private'),
        resetToHome: () async => events.add('home'),
      );

      await container.read(authSessionProvider.future);
      final result = await container
          .read(authControllerProvider.notifier)
          .signOut();

      expect(events, [
        'server-logout',
        'invalidate-account',
        'clear-private',
        'home',
      ]);
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
