import 'package:dio/dio.dart';

/// Paths on which the Authorization header should never be attached.
/// X-Craftsky-Device-Id is sent on ALL paths including these — the
/// server treats device-id as install-identity, not user-identity.
const _anonymousPaths = <String>{'/v1/auth/login'};

class SessionAuthInterceptor extends Interceptor {
  /// Account-bound constructor. The bearer is captured once and cannot change
  /// if another account becomes active while a request is in flight.
  SessionAuthInterceptor.fixed({
    required String token,
    required Future<String> Function() readDeviceId,
  }) : this._(() => token, readDeviceId);

  SessionAuthInterceptor.anonymous({
    required Future<String> Function() readDeviceId,
  }) : this._(() => null, readDeviceId);

  SessionAuthInterceptor._(this._readToken, this._readDeviceId);

  final String? Function() _readToken;
  final Future<String> Function() _readDeviceId;

  // Dio's base signature is `void onRequest(...)`. We declare `void`
  // with an `async` body so we can await the device-id future; Dio
  // continues the chain when we call `handler.next(options)`, not
  // when our future resolves. Do NOT change the return type to
  // Future<void> — it's an invalid override of Dio's base.
  @override
  // Required to match Dio's `void onRequest(...)` base signature; we
  // still need an async body to await the device-id future.
  // ignore: avoid_void_async
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final deviceId = await _readDeviceId();
    options.headers['X-Craftsky-Device-Id'] = deviceId;

    if (!_anonymousPaths.contains(options.path)) {
      final token = _readToken();
      if (token != null) {
        options.headers['Authorization'] = 'Bearer $token';
      }
    }
    handler.next(options);
  }
}
