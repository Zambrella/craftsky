import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/notifications/providers/notification_lifecycle_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Installed on the session Dio only. On 401 from any authenticated
/// call, clears secure storage and flips `authSessionProvider` to
/// `SignedOut`. The router's refresh listenable then boots the user
/// to `/welcome`. Feature code never sees 401 recovery plumbing.
class SignOutOn401Interceptor extends Interceptor {
  SignOutOn401Interceptor.fromRef(Ref ref)
    : _signOut = (() async {
        final auth = ref.read(authSessionProvider).value;
        if (auth case SignedIn(:final did)) {
          await ref
              .read(notificationSignOutCleanupProvider)
              .run(did: did.toString(), confirmedLogout: false);
        }
        await ref.read(secureTokenStorageProvider).clear();
        ref.read(authSessionProvider.notifier).setSignedOut();
      });

  /// Test constructor: accepts a `signOut` callable the test drives
  /// (or closes over its own `ProviderContainer.test()`).
  SignOutOn401Interceptor.withSignOut(FutureOr<void> Function() signOut)
    : _signOut = signOut;

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
