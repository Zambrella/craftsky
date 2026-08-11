import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:dio/dio.dart';

/// Invalidates only the captured session lease when an authenticated request
/// returns 401. The interceptor never consults mutable active-account state.
class SignOutOn401Interceptor extends Interceptor {
  factory SignOutOn401Interceptor.withLease({
    required AccountSessionLease lease,
    required FutureOr<void> Function(AccountSessionLease lease) invalidate,
    FutureOr<void> Function()? recoverPendingDeletion,
  }) => SignOutOn401Interceptor._(
    () => invalidate(lease),
    recoverPendingDeletion,
  );

  SignOutOn401Interceptor._(this._signOut, this._recoverPendingDeletion);

  final FutureOr<void> Function() _signOut;
  final FutureOr<void> Function()? _recoverPendingDeletion;
  Future<void>? _inFlight;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      final data = err.response?.data;
      final pendingDeletion =
          data is Map && data['error'] == 'account_deletion_pending';
      final action = pendingDeletion && _recoverPendingDeletion != null
          ? _recoverPendingDeletion
          : _signOut;
      _inFlight ??= Future.sync(action).whenComplete(() => _inFlight = null);
    }
    handler.next(err);
  }
}
