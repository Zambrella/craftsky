import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_status_provider.g.dart';

/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).
@riverpod
class AuthStatus extends _$AuthStatus {
  @override
  bool build() => false;

  void signIn() => state = true;
  void signOut() => state = false;
}
