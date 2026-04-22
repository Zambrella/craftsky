import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Paths on which the Authorization header should never be attached.
/// X-Craftsky-Device-Id is sent on ALL paths including these — the
/// server treats device-id as install-identity, not user-identity.
const _anonymousPaths = <String>{'/v1/auth/login'};

class SessionAuthInterceptor extends Interceptor {
  /// Production constructor.
  SessionAuthInterceptor.fromRef(Ref ref)
    : _readAuth = (() => ref.read(authSessionProvider)),
      _readDeviceId = (() => ref.read(deviceIdProvider.future));

  /// Back-compat test constructor — injects a fixed dummy device-id
  /// so existing tests that only care about the Authorization header
  /// keep working without a constructor rewrite.
  SessionAuthInterceptor.withReader(
    AsyncValue<AuthState> Function() readAuth,
  ) : _readAuth = readAuth,
      _readDeviceId = (() async => 'test-device-id');

  /// Full test constructor — both readers explicit.
  SessionAuthInterceptor.withReaders({
    required AsyncValue<AuthState> Function() readAuth,
    required Future<String> Function() readDeviceId,
  }) : _readAuth = readAuth,
       _readDeviceId = readDeviceId;

  final AsyncValue<AuthState> Function() _readAuth;
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
      final auth = _readAuth().value;
      if (auth is SignedIn) {
        options.headers['Authorization'] = 'Bearer ${auth.token}';
      }
    }
    handler.next(options);
  }
}
