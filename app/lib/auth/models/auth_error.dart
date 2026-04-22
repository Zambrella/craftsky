/// User-actionable auth errors surfaced by `AuthController`. Sealed so
/// call sites can exhaustively switch on them.
sealed class AuthError implements Exception {
  const AuthError();
}

/// User submitted an empty handle.
final class HandleRequired extends AuthError {
  const HandleRequired();
}

/// Server rejected the handle (e.g. malformed). Mapped from any
/// non-specific 4xx from `/v1/auth/login`.
final class InvalidHandle extends AuthError {
  const InvalidHandle();
}

/// AppView is unreachable or returned 5xx, or the device is offline.
final class ServerUnavailable extends AuthError {
  const ServerUnavailable();
}

/// `url_launcher` failed to open the system browser.
final class BrowserLaunchFailed extends AuthError {
  const BrowserLaunchFailed();
}

/// A deep link arrived but no sign-in is in progress.
final class NoPendingSignIn extends AuthError {
  const NoPendingSignIn();
}

/// A deep link arrived more than 10 minutes after the user started
/// the sign-in.
final class SignInTimedOut extends AuthError {
  const SignInTimedOut();
}

/// `flutter_secure_storage` read/write failed (Android keystore issues,
/// platform quirks).
final class StorageFailure extends AuthError {
  const StorageFailure(this.cause);

  final Object cause;
}
