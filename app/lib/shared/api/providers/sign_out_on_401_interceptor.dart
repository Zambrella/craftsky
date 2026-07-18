import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:dio/dio.dart';

/// Invalidates only the captured session lease when an authenticated request
/// returns 401. The interceptor never consults mutable active-account state.
class SignOutOn401Interceptor extends Interceptor {
  SignOutOn401Interceptor.withLease({
    required AccountSessionLease lease,
    required FutureOr<void> Function(AccountSessionLease lease) invalidate,
  }) : _signOut = (() => invalidate(lease));

  final FutureOr<void> Function() _signOut;
  Future<void>? _inFlight;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      _inFlight ??= Future.sync(_signOut).whenComplete(() => _inFlight = null);
    }
    handler.next(err);
  }
}
