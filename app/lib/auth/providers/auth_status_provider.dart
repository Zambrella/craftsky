import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_status_provider.g.dart';

/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).
@riverpod
class AuthStatus extends _$AuthStatus {
  // Flip the first operand to `true` locally to skip past the auth/onboarding
  // flow during manual dev runs. kReleaseMode always defaults to `false`.
  @override
  bool build() =>
      // Intentional same-literal ternary — lets the dev flip the first
      // operand without touching the second. Disabled lint would hide the
      // toggle surface.
      // ignore: avoid_bool_literals_in_conditional_expressions
      kDebugMode ? false : false;

  void signIn() => state = true;
  void signOut() => state = false;
}
