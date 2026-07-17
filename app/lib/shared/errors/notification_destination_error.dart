import 'package:craftsky_app/shared/api/api_exception.dart';

enum NotificationDestinationErrorKind {
  permanentUnavailable,
  retryable,
  authenticationLost,
}

NotificationDestinationErrorKind classifyNotificationDestinationError(
  Object error,
) => switch (error) {
  ApiUnauthorized() => NotificationDestinationErrorKind.authenticationLost,
  ApiBadRequest(:final code, :final details)
      when details.statusCode == 404 &&
          (code == 'post_not_found' || code == 'profile_not_found') =>
    NotificationDestinationErrorKind.permanentUnavailable,
  _ => NotificationDestinationErrorKind.retryable,
};
