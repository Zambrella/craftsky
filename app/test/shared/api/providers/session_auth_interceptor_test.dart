import 'package:craftsky_app/shared/api/providers/session_auth_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _CapturingHandler extends RequestInterceptorHandler {
  bool continued = false;

  @override
  void next(RequestOptions options) => continued = true;
}

void main() {
  test('fixed account requests attach captured bearer and device ID', () async {
    final options = RequestOptions(path: '/v1/whoami');
    SessionAuthInterceptor.fixed(
      token: 'token-a',
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers['Authorization'], 'Bearer token-a');
    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
  });

  test('anonymous requests never attach a bearer', () async {
    final options = RequestOptions(path: '/v1/auth/login');
    SessionAuthInterceptor.anonymous(
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers.containsKey('Authorization'), isFalse);
    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
  });

  test('keeps using the account token captured at construction', () async {
    final interceptor = SessionAuthInterceptor.fixed(
      token: 'token-a',
      readDeviceId: () async => 'device-abc',
    );
    final first = RequestOptions(path: '/v1/whoami');
    final second = RequestOptions(path: '/v1/feed');

    interceptor
      ..onRequest(first, _CapturingHandler())
      ..onRequest(second, _CapturingHandler());
    await _pumpEventLoop();

    expect(first.headers['Authorization'], 'Bearer token-a');
    expect(second.headers['Authorization'], 'Bearer token-a');
  });
}

Future<void> _pumpEventLoop() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}
