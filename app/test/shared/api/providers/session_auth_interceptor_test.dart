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

  test('anonymous handoff confirmation preserves its pending bearer', () async {
    final options = RequestOptions(
      path: '/v1/auth/handoffs/confirm',
      headers: {'Authorization': 'Bearer pending-handoff-token'},
    );
    SessionAuthInterceptor.anonymous(
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers['Authorization'], 'Bearer pending-handoff-token');
    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
  });

  test('registration stays anonymous with the stable device ID', () async {
    final options = RequestOptions(path: '/v1/auth/registrations');
    SessionAuthInterceptor.fixed(
      token: 'craftsky-session-secret',
      readDeviceId: () async => 'stable-device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers.containsKey('Authorization'), isFalse);
    expect(
      options.headers['X-Craftsky-Device-Id'],
      'stable-device-abc',
    );
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

  test('UT-017 transmits only the Craftsky bearer credential', () async {
    final options = RequestOptions(
      path: '/v1/profile',
      method: 'POST',
      headers: {
        'Authorization': 'DPoP pds-access-token-canary',
        'DPoP': 'dpop-proof-canary',
        'X-PDS-Refresh-Token': 'pds-refresh-token-canary',
        'X-PDS-Endpoint': 'https://obsolete-pds.example',
      },
    );

    SessionAuthInterceptor.fixed(
      token: 'craftsky-session-canary',
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers, {
      'Authorization': 'Bearer craftsky-session-canary',
      'X-Craftsky-Device-Id': 'device-abc',
    });
  });

  test('UT-017 anonymous requests strip attempted credentials', () async {
    final options = RequestOptions(
      path: '/v1/auth/login',
      headers: {
        'Authorization': 'Bearer pds-access-token-canary',
        'dpop': 'dpop-proof-canary',
      },
    );

    SessionAuthInterceptor.anonymous(
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());
    await _pumpEventLoop();

    expect(options.headers, {'X-Craftsky-Device-Id': 'device-abc'});
  });
}

Future<void> _pumpEventLoop() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}
