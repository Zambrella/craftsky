import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/shared/api/providers/sign_out_on_401_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _SignedInFake extends AuthSession {
  @override
  Future<AuthState> build() async =>
      SignedIn(did: 'did:plc:test', handle: 'h.test', token: 't');
}

class _RecordingStorage implements SecureTokenStorage {
  int clearCalls = 0;
  @override
  Future<StoredSession?> read() async => null;
  @override
  Future<void> write(StoredSession s) async {}
  @override
  Future<void> clear() async => clearCalls++;
}

class _CapturingHandler extends ErrorInterceptorHandler {
  DioException? error;
  @override
  void next(DioException err) => error = err;
}

DioException _exWithStatus(int status) {
  final req = RequestOptions(path: '/v1/whoami');
  return DioException(
    requestOptions: req,
    response: Response(requestOptions: req, statusCode: status),
    type: DioExceptionType.badResponse,
  );
}

/// Builds the production-shape signOut callback from a test container.
void Function() _signOutFrom(ProviderContainer c) => () {
  unawaited(c.read(secureTokenStorageProvider).clear());
  c.read(authSessionProvider.notifier).setSignedOut();
};

void main() {
  test(
    '401 flips authSessionProvider to SignedOut and clears storage',
    () async {
      final storage = _RecordingStorage();
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(_SignedInFake.new),
          secureTokenStorageProvider.overrideWithValue(storage),
        ],
      );
      await container.read(authSessionProvider.future);

      SignOutOn401Interceptor.withSignOut(
        _signOutFrom(container),
      ).onError(_exWithStatus(401), _CapturingHandler());

      // storage.clear is unawaited inside the interceptor — pump the loop.
      for (var i = 0; i < 3; i++) {
        await Future<void>.delayed(Duration.zero);
      }

      expect(container.read(authSessionProvider).value, isA<SignedOut>());
      expect(storage.clearCalls, 1);
    },
  );

  test('non-401 errors pass through unchanged', () async {
    final storage = _RecordingStorage();
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(_SignedInFake.new),
        secureTokenStorageProvider.overrideWithValue(storage),
      ],
    );
    await container.read(authSessionProvider.future);

    final err500 = _exWithStatus(500);
    final handler = _CapturingHandler();

    SignOutOn401Interceptor.withSignOut(
      _signOutFrom(container),
    ).onError(err500, handler);

    expect(handler.error, same(err500));
    expect(storage.clearCalls, 0);
    expect(container.read(authSessionProvider).value, isA<SignedIn>());
  });
}
