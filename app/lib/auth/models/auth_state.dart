import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// High-level auth state exposed by `authSessionProvider`. Distinct
/// from the notifier class name (`AuthSession`) to avoid shadowing
/// inside `AuthSession.build()`.
sealed class AuthState {
  const AuthState();
}

final class SignedOut extends AuthState {
  const SignedOut();
}

final class SignedIn extends AuthState {
  SignedIn({
    required String did,
    required String handle,
    required this.token,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  final String token;

  /// Token is redacted in string form so logs + error screens can't
  /// accidentally leak it via `'$state'` or `toString`.
  @override
  String toString() =>
      'SignedIn(did: $did, handle: $handle, token: <redacted>)';
}
