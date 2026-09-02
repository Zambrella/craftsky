import 'package:craftsky_app/shared/api/api_exception.dart';

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

/// The AppView rejected an absent, expired, or already-consumed handoff code or
/// confirmation receipt. The server clock is authoritative for expiry.
final class SignInTimedOut extends AuthError {
  const SignInTimedOut();
}

/// `flutter_secure_storage` read/write failed (Android keystore issues,
/// platform quirks).
final class StorageFailure extends AuthError {
  const StorageFailure(this.cause);

  final Object cause;
}

/// Bounded registration outcomes accepted from trusted callback state or
/// derived from the dedicated registration-start boundary.
enum RegistrationFailure implements AuthError {
  canceled,
  providerUnavailable,
  registrationIncomplete;

  static RegistrationFailure? fromCallback(String? value) => switch (value) {
    'canceled' => canceled,
    'providerUnavailable' => providerUnavailable,
    'registrationIncomplete' => registrationIncomplete,
    _ => null,
  };

  static RegistrationFailure? fromStartError(Object error) => switch (error) {
    ApiBadRequest(code: 'registration_provider_unavailable') =>
      providerUnavailable,
    ApiBadRequest(code: 'registration_incomplete') => registrationIncomplete,
    ApiServerError(
      details: ApiFailureDetails(
        appViewError: 'registration_provider_unavailable',
      ),
    ) =>
      providerUnavailable,
    ApiServerError(
      details: ApiFailureDetails(appViewError: 'registration_incomplete'),
    ) =>
      registrationIncomplete,
    ApiNetworkError() || ApiServerError() => providerUnavailable,
    BrowserLaunchFailed() => registrationIncomplete,
    _ => null,
  };
}
