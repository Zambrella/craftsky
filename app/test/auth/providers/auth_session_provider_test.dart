import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
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

class _FakeApi implements AuthApiClient {
  _FakeApi({required this.onWhoami});
  final Future<WhoAmI> Function() onWhoami;
  @override
  Future<WhoAmI> whoami() => onWhoami();
  @override
  Future<LoginResponse> login({required String handle}) =>
      Future<LoginResponse>.error(UnimplementedError('login not used here'));
  @override
  Future<void> logout() =>
      Future<void>.error(UnimplementedError('logout not used here'));
}

ProviderContainer _container({
  required SecureTokenStorage storage,
  AuthApiClient? api,
}) => ProviderContainer.test(
  overrides: [
    secureTokenStorageProvider.overrideWithValue(storage),
    if (api != null) authApiClientProvider.overrideWithValue(api),
  ],
);

/// `_validateInBackground` is `unawaited` inside `AuthSession.build`;
/// the chain performs up to three awaits (whoami → storage.{clear|write}
/// → state =), each yielding one microtask. Flush the event loop a few
/// times to settle.
Future<void> _flushBackgroundValidation() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}

void main() {
  setUpAll(initializeMappers);

  test('resolves to SignedOut when storage is empty', () async {
    final container = _container(storage: _FakeStorage());
    final state = await container.read(authSessionProvider.future);
    expect(state, isA<SignedOut>());
  });

  test('resolves to SignedIn when storage has a session', () async {
    final container = _container(
      storage: _FakeStorage(
        initial: const StoredSession(token: 't', did: 'd', handle: 'h'),
      ),
      api: _FakeApi(
        onWhoami: () async => WhoAmI(did: 'did:plc:test', handle: 'h.test'),
      ),
    );
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
        onWhoami: () async => WhoAmI(did: 'did:plc:new', handle: 'h.test'),
      ),
    );
    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    expect(container.read(authSessionProvider).value, isA<SignedOut>());
    expect(await storage.read(), isNull);
  });

  test('whoami handle drift updates cache + state, keeps SignedIn', () async {
    final storage = _FakeStorage(
      initial: const StoredSession(
        token: 't',
        did: 'did:plc:test',
        handle: 'old.bsky.social',
      ),
    );
    final container = _container(
      storage: storage,
      api: _FakeApi(
        onWhoami: () async => WhoAmI(
          did: 'did:plc:test',
          handle: 'new.bsky.social',
        ),
      ),
    );
    await container.read(authSessionProvider.future);
    await _flushBackgroundValidation();

    final signed = container.read(authSessionProvider).value! as SignedIn;
    expect(signed.handle, 'new.bsky.social');
    expect((await storage.read())!.handle, 'new.bsky.social');
  });

  test(
    'whoami network error keeps cached SignedIn (offline tolerance)',
    () async {
      final storage = _FakeStorage(
        initial: const StoredSession(token: 't', did: 'd', handle: 'h'),
      );
      final container = _container(
        storage: storage,
        api: _FakeApi(
          onWhoami: () async => throw const ApiNetworkError('offline'),
        ),
      );
      await container.read(authSessionProvider.future);
      await _flushBackgroundValidation();

      expect(container.read(authSessionProvider).value, isA<SignedIn>());
      expect(await storage.read(), isNotNull);
    },
  );

  test('setSignedIn/setSignedOut set state imperatively', () async {
    final container = _container(storage: _FakeStorage());
    await container.read(authSessionProvider.future);
    container
        .read(authSessionProvider.notifier)
        .setSignedIn(
          const SignedIn(did: 'd', handle: 'h', token: 't'),
        );
    expect(container.read(authSessionProvider).value, isA<SignedIn>());

    container.read(authSessionProvider.notifier).setSignedOut();
    expect(container.read(authSessionProvider).value, isA<SignedOut>());
  });
}
