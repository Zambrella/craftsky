import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Paths on which the Authorization header should never be attached.
const _anonymousPaths = <String>{'/v1/auth/login'};

class SessionAuthInterceptor extends Interceptor {
  /// Production constructor: `SessionAuthInterceptor.fromRef(ref)`.
  SessionAuthInterceptor.fromRef(Ref ref)
    : _readAuth = (() => ref.read(authSessionProvider));

  /// Test constructor: accepts any callable that returns the current
  /// `AsyncValue<AuthState>`. Production wiring uses `fromRef`; tests
  /// can inject a fake reader driven by a `ProviderContainer.test()`.
  SessionAuthInterceptor.withReader(this._readAuth);

  final AsyncValue<AuthState> Function() _readAuth;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (_anonymousPaths.contains(options.path)) {
      handler.next(options);
      return;
    }
    final auth = _readAuth().value;
    if (auth is SignedIn) {
      options.headers['Authorization'] = 'Bearer ${auth.token}';
    }
    handler.next(options);
  }
}
