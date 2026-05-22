import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// Fake AuthSession that returns a configured state immediately.
// Subclass (not implements) per the riverpod.md testing rule.
class _SignedInFake extends AuthSession {
  _SignedInFake(this.token);
  final String token;
  @override
  Future<AuthState> build() async =>
      SignedIn(did: 'd', handle: 'h', token: token);
}

class _SignedOutFake extends AuthSession {
  @override
  Future<AuthState> build() async => const SignedOut();
}

class _CapturingHandler extends RequestInterceptorHandler {
  bool continued = false;
  @override
  void next(RequestOptions options) => continued = true;
}

void main() {
  Future<void> seed(ProviderContainer c) async =>
      c.read(authSessionProvider.future);

  test('attaches Bearer from SignedIn state', () async {
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(() => _SignedInFake('tok-abc')),
      ],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/whoami');
    SessionAuthInterceptor.withReader(
      () => container.read(authSessionProvider),
    ).onRequest(options, _CapturingHandler());

    await _pumpEventLoop();

    expect(options.headers['Authorization'], 'Bearer tok-abc');
  });

  test('omits Authorization when SignedOut', () async {
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(_SignedOutFake.new),
      ],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/whoami');
    SessionAuthInterceptor.withReader(
      () => container.read(authSessionProvider),
    ).onRequest(options, _CapturingHandler());

    await _pumpEventLoop();

    expect(options.headers.containsKey('Authorization'), isFalse);
  });

  test('skips Authorization for /v1/auth/login even when SignedIn', () async {
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(() => _SignedInFake('tok-abc')),
      ],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/auth/login');
    SessionAuthInterceptor.withReader(
      () => container.read(authSessionProvider),
    ).onRequest(options, _CapturingHandler());

    await _pumpEventLoop();

    expect(options.headers.containsKey('Authorization'), isFalse);
    expect(
      options.headers['X-Craftsky-Device-Id'],
      'test-device-id',
    ); // new — device-id attaches even on anonymous paths
  });

  test(
    'omits header when AuthSession is still loading (no value yet)',
    () async {
      final container = ProviderContainer.test();
      final options = RequestOptions(path: '/v1/whoami');
      SessionAuthInterceptor.withReader(
        () => container.read(authSessionProvider),
      ).onRequest(options, _CapturingHandler());

      await _pumpEventLoop();

      expect(options.headers.containsKey('Authorization'), isFalse);
    },
  );

  // --- Device-ID header attachment ---

  test('attaches X-Craftsky-Device-Id on anonymous requests', () async {
    final container = ProviderContainer.test(
      overrides: [authSessionProvider.overrideWith(_SignedOutFake.new)],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/auth/login');
    SessionAuthInterceptor.withReaders(
      readAuth: () => container.read(authSessionProvider),
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());

    // onRequest is `void`-returning but internally awaits; pump until
    // microtasks settle so the header-mutation assertions are sound.
    await _pumpEventLoop();

    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
    expect(options.headers.containsKey('Authorization'), isFalse);
  });

  test(
    'attaches BOTH Authorization and X-Craftsky-Device-Id on authed requests',
    () async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(() => _SignedInFake('tok')),
        ],
      );
      await seed(container);

      final options = RequestOptions(path: '/v1/whoami');
      SessionAuthInterceptor.withReaders(
        readAuth: () => container.read(authSessionProvider),
        readDeviceId: () async => 'device-abc',
      ).onRequest(options, _CapturingHandler());

      await _pumpEventLoop();

      expect(options.headers['Authorization'], 'Bearer tok');
      expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
    },
  );
}

/// Flushes microtasks so async gaps inside a `void`-returning async
/// interceptor settle before assertions run.
Future<void> _pumpEventLoop() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}
