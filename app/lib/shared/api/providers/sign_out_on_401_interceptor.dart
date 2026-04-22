import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Installed on the session Dio only. On 401 from any authenticated
/// call, clears secure storage and flips `authSessionProvider` to
/// `SignedOut`. The router's refresh listenable then boots the user
/// to `/welcome`. Feature code never sees 401 recovery plumbing.
class SignOutOn401Interceptor extends Interceptor {
  SignOutOn401Interceptor.fromRef(Ref ref)
    : _signOut = (() {
        unawaited(ref.read(secureTokenStorageProvider).clear());
        ref.read(authSessionProvider.notifier).setSignedOut();
      });

  /// Test constructor: accepts a `signOut` callable the test drives
  /// (or closes over its own `ProviderContainer.test()`).
  SignOutOn401Interceptor.withSignOut(this._signOut);

  final void Function() _signOut;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (err.response?.statusCode == 401) {
      _signOut();
    }
    handler.next(err);
  }
}
