import 'package:craftsky_app/auth/models/pending_auth.dart' as model;
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'pending_auth_provider.g.dart';

/// Tracks the in-flight sign-in attempt for UI state only. The server-bound
/// code and stable device ID remain authoritative if the process restarts while
/// the system browser is open.
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
  /// Kept as a method (not a setter) because the `@visibleForTesting`
  /// intent is easier to see on call sites like `debugSet(...)`.
  @visibleForTesting
  // Methods make the test-only intent explicit at call sites.
  // ignore: use_setters_to_change_properties
  void debugSet(model.PendingAuth value) => state = value;
}
