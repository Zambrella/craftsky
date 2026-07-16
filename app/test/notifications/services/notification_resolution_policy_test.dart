import 'package:craftsky_app/notifications/models/notification_resolution.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('UT-004 authorized notification resolution policy', () {
    test('uses only active server-supplied post and profile targets', () {
      final post = NotificationResolution.fromMap({
        'id': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
        'type': 'like',
        'state': 'active',
        'target': {
          'kind': 'post',
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/one',
        },
      });
      final profile = NotificationResolution.fromMap({
        'id': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
        'type': 'follow',
        'state': 'active',
        'target': {'kind': 'actorProfile', 'did': 'did:plc:alice'},
      });

      expect(
        NotificationResolutionPolicy.forResolution(post).destination,
        NotificationDestination.post(
          AtUri.parse(
            'at://did:plc:alice/social.craftsky.feed.post/one',
          ),
        ),
      );
      expect(
        NotificationResolutionPolicy.forResolution(profile).destination,
        NotificationDestination.profile(Did.parse('did:plc:alice')),
      );
    });

    test('falls back for notifications, retracted, and malformed targets', () {
      for (final map in <Map<String, Object?>>[
        {
          'id': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
          'type': 'mention',
          'state': 'active',
          'target': {'kind': 'notifications'},
        },
        {
          'id': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
          'type': 'mention',
          'state': 'retracted',
          'target': {'kind': 'actorProfile', 'did': 'did:plc:alice'},
        },
        {
          'id': '018f47a2-4b0e-7f39-a621-9f6f6c75e312',
          'type': 'like',
          'state': 'active',
          'target': {'kind': 'post', 'uri': ''},
        },
      ]) {
        expect(
          NotificationResolutionPolicy.forResolution(
            NotificationResolution.fromMap(map),
          ).destination,
          const NotificationDestination.notifications(),
        );
      }
    });

    test('maps not-found and transport failures without retry state', () {
      final notFound = NotificationResolutionPolicy.forFailure(
        NotificationResolutionFailure.notFound,
      );
      final offline = NotificationResolutionPolicy.forFailure(
        NotificationResolutionFailure.network,
      );
      final timeout = NotificationResolutionPolicy.forFailure(
        NotificationResolutionFailure.timeout,
      );

      expect(
        notFound.destination,
        const NotificationDestination.notifications(),
      );
      expect(notFound.feedback, isNull);
      expect(
        offline.destination,
        const NotificationDestination.notifications(),
      );
      expect(offline.feedback, NotificationOpenFeedback.unableToOpen);
      expect(timeout.feedback, NotificationOpenFeedback.unableToOpen);
      expect(notFound.shouldRetry, isFalse);
      expect(offline.shouldRetry, isFalse);
      expect(timeout.shouldRetry, isFalse);
    });

    test('maps API 404 separately from transport failures', () {
      final notFound = NotificationResolutionPolicy.forException(
        const ApiBadRequest(
          'notification_not_found',
          details: ApiFailureDetails(statusCode: 404),
        ),
      );
      final offline = NotificationResolutionPolicy.forException(
        const ApiNetworkError('offline'),
      );

      expect(notFound.feedback, isNull);
      expect(offline.feedback, NotificationOpenFeedback.unableToOpen);
    });
  });
}
