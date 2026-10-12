import 'package:craftsky_app/shared/atproto/identifiers.dart';

/// High-level auth state exposed by `authSessionProvider`. Distinct
/// from the notifier class name (`AuthSession`) to avoid shadowing
/// inside `AuthSession.build()`.
sealed class AuthState {
  const new();
}

final class SignedOut extends AuthState {
  const new();
}

final class SignedIn extends AuthState {
  new({
    required String did,
    required String handle,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle);

  final Did did;
  final Handle handle;
  @override
  String toString() => 'SignedIn(<redacted>)';
}
