import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 AT-005 IT-009 builds reply subject route with source focus', () {
    const subjectUri = 'at://did:plc:subject/social.craftsky.feed.post/subject';
    const sourceUri = 'at://did:plc:source/social.craftsky.feed.post/source';

    final route = postThreadRouteForNotification(
      PostDestination(
        AtUri.parse(subjectUri),
        focusUri: AtUri.parse(sourceUri),
      ),
    )!;

    expect(route.did, 'did:plc:subject');
    expect(route.rkey, 'subject');
    expect(route.focus, sourceUri);
    expect(
      route.location,
      '/posts/did%3Aplc%3Asubject/subject?focus='
      'at%3A%2F%2Fdid%3Aplc%3Asource%2Fsocial.craftsky.feed.post%2Fsource',
    );
  });

  test('AT-004 builds the canonical moderation history route', () {
    final route = accountStandingRouteForNotification(
      ModerationHistoryDestination(
        ModerationCaseReference.parse(
          'mod-550E8400-E29B-41D4-A716-446655440000',
        ),
      ),
    );

    expect(route, isA<AccountStandingRoute>());
    expect(route.location, '/settings/moderation');
  });

  for (final type in ['mention', 'quote']) {
    test('a $type push opens the root with the reply focused', () {
      const root = 'at://did:plc:author/social.craftsky.feed.post/root';
      const source = 'at://did:plc:author/social.craftsky.feed.post/reply';
      final attempt = NotificationOpenAttempt.fromProviderData({
        'payloadVersion': '1',
        'type': type,
        'accountSubscriptionId': 'test-binding',
        'rootUri': root,
        'sourceUri': source,
      });
      final destination =
          NotificationDestinationInference.forFacts(
                attempt.facts,
              ).destination
              as PostDestination;
      final route = postThreadRouteForNotification(destination)!;

      expect(route.did, 'did:plc:author');
      expect(route.rkey, 'root');
      expect(route.focus, source);
    });
  }
}
