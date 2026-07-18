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
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  @override
  String toString() => 'SignedIn(<redacted>)';
}
