import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pending_auth_provider.g.dart';

/// Tracks the in-flight sign-in attempt. Lets
/// `AuthController.completeFromDeepLink` reject deep links that
/// arrive without a prior `signIn()` or later than the 10-minute
/// staleness window.
///
/// The notifier class is named `PendingAuth` — same identifier as
/// the data class it holds, imported under the `model` prefix to
/// dodge the collision inside this file. The generated provider is
/// `pendingAuthProvider`.
@Riverpod(keepAlive: true)
class PendingAuth extends _$PendingAuth {
  @override
  model.PendingAuth? build() => null;

  void start(String handle) => state = model.PendingAuth(
        handle: handle,
        startedAt: DateTime.now(),
      );

  void clear() => state = null;

  /// Direct state setter — used by tests that need to age the
  /// `startedAt` without real clock manipulation (see
  /// `auth_controller_test.dart` for the stale-pending scenario).
  @visibleForTesting
  void debugSet(model.PendingAuth value) => state = value;
}
