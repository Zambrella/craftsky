import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/notification_destination_error.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-009 classifies destination errors without retry scheduling', () {
    const notFoundDetails = ApiFailureDetails(statusCode: 404);
    const identityDetails = ApiFailureDetails(
      statusCode: 502,
      appViewError: 'identity_unavailable',
    );
    final cases = <(Object, NotificationDestinationErrorKind)>[
      (
        const ApiBadRequest(
          'post_not_found',
          details: notFoundDetails,
        ),
        NotificationDestinationErrorKind.permanentUnavailable,
      ),
      (
        const ApiBadRequest(
          'profile_not_found',
          details: notFoundDetails,
        ),
        NotificationDestinationErrorKind.permanentUnavailable,
      ),
      (
        const ApiNetworkError('offline'),
        NotificationDestinationErrorKind.retryable,
      ),
      (
        const ApiServerError('http_500'),
        NotificationDestinationErrorKind.retryable,
      ),
      (
        const ApiServerError(
          'identity unavailable',
          details: identityDetails,
        ),
        NotificationDestinationErrorKind.retryable,
      ),
      (
        const ApiUnauthorized(),
        NotificationDestinationErrorKind.authenticationLost,
      ),
      (StateError('unexpected'), NotificationDestinationErrorKind.retryable),
    ];

    for (final (error, expected) in cases) {
      expect(classifyNotificationDestinationError(error), expected);
    }
  });
}
