import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('BUG-002 like facts route a comment through its root thread', () {
    const subjectUri =
        'at://did:plc:commenter/social.craftsky.feed.post/comment';
    const rootUri = 'at://did:plc:author/social.craftsky.feed.post/root';

    final attempt = NotificationOpenAttempt.fromProviderData(
      _providerData(
        type: 'like',
        subjectUri: subjectUri,
        rootUri: rootUri,
      ),
    );
    final outcome = NotificationDestinationInference.forFacts(attempt.facts);

    expect(
      outcome.destination,
      PostDestination(
        AtUri.parse(rootUri),
        focusUri: AtUri.parse(subjectUri),
      ),
    );
  });

  test('UT-004 infers every destination from canonical category facts', () {
    const actorDid = 'did:plc:alice';
    const subjectUri = 'at://did:plc:subject/social.craftsky.feed.post/subject';
    const sourceUri = 'at://did:plc:source/social.craftsky.feed.post/source';
    final cases =
        <({Map<String, Object?> data, NotificationDestination expected})>[
          (
            data: _providerData(type: 'follow', actorDid: actorDid),
            expected: ProfileDestination(Did.parse(actorDid)),
          ),
          (
            data: _providerData(
              type: 'like',
              subjectUri: subjectUri,
              rootUri: subjectUri,
            ),
            expected: PostDestination(AtUri.parse(subjectUri)),
          ),
          (
            data: _providerData(
              type: 'repost',
              subjectUri: subjectUri,
              rootUri: subjectUri,
            ),
            expected: PostDestination(AtUri.parse(subjectUri)),
          ),
          (
            data: _providerData(type: 'mention', sourceUri: sourceUri),
            expected: PostDestination(AtUri.parse(sourceUri)),
          ),
          (
            data: _providerData(type: 'quote', sourceUri: sourceUri),
            expected: PostDestination(AtUri.parse(sourceUri)),
          ),
          (
            data: _providerData(
              type: 'reply',
              subjectUri: subjectUri,
              sourceUri: sourceUri,
            ),
            expected: PostDestination(
              AtUri.parse(subjectUri),
              focusUri: AtUri.parse(sourceUri),
            ),
          ),
          (
            data: _providerData(type: 'everythingElse'),
            expected: const NotificationsDestination(),
          ),
        ];

    for (final testCase in cases) {
      final attempt = NotificationOpenAttempt.fromProviderData(testCase.data);
      final outcome = NotificationDestinationInference.forFacts(attempt.facts);

      expect(outcome.destination, testCase.expected);
      expect(outcome.feedback, isNull);
    }
  });

  test('UT-005 and AT-003 distinguish invalid from unknown facts', () {
    final invalid = <Map<String, Object?>>[
      <String, Object?>{
        'type': 'like',
        'accountSubscriptionId': 'subscription_Abc123',
        'subjectUri': 'at://did:plc:subject/social.craftsky.feed.post/subject',
      },
      <String, Object?>{
        ..._providerData(type: 'everythingElse'),
        'payloadVersion': '2',
      },
      _providerData(type: 'not-valid!'),
      _providerData(type: 'reply'),
    ];

    for (final data in invalid) {
      final attempt = NotificationOpenAttempt.fromProviderData(data);
      final outcome = NotificationDestinationInference.forFacts(attempt.facts);

      expect(outcome.destination, const NotificationsDestination());
      expect(outcome.feedback, NotificationOpenFeedback.unableToOpen);
    }

    final future = NotificationOpenAttempt.fromProviderData({
      ..._providerData(type: 'projectInvite2'),
      'route': '/must/not/be/used',
      'subjectUri': 'https://must.not.be/used.example',
    });
    final futureOutcome = NotificationDestinationInference.forFacts(
      future.facts,
    );

    expect(future.facts, isA<UnknownNotificationFacts>());
    expect(futureOutcome.destination, const NotificationsDestination());
    expect(futureOutcome.feedback, isNull);
  });

  test('UT-012 infers actorless Instagram migration push destinations', () {
    final attempt = NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'instagramMatch',
      'accountSubscriptionId': 'subscription_Abc123',
      'notificationId': '00000000-0000-0000-0000-000000000321',
      'count': '3',
      'countCapped': 'false',
      'destination': 'instagramMigration',
    });

    expect(attempt.facts, isA<ValidNotificationFacts>());
    final facts = attempt.facts as ValidNotificationFacts;
    expect(facts.category, NotificationCategory.instagramMatch);
    expect(facts.actorDid, isNull);
    expect(facts.subjectUri, isNull);
    expect(
      NotificationDestinationInference.forFacts(facts).destination,
      const InstagramMigrationDestination(),
    );

    for (final invalid in [
      {..._providerData(type: 'instagramMatch'), 'count': '0', 'countCapped': 'false', 'destination': 'instagramMigration'},
      {..._providerData(type: 'instagramMatch'), 'count': '3', 'countCapped': 'not-bool', 'destination': 'instagramMigration'},
      {..._providerData(type: 'instagramMatch'), 'count': '3', 'countCapped': 'false', 'destination': 'profile'},
    ]) {
      expect(
        NotificationOpenAttempt.fromProviderData(invalid).facts,
        isA<InvalidNotificationFacts>(),
      );
    }
  });
}

Map<String, Object?> _providerData({
  required String type,
  String? actorDid,
  String? subjectUri,
  String? rootUri,
  String? sourceUri,
}) => <String, Object?>{
  'payloadVersion': '1',
  'type': type,
  'accountSubscriptionId': 'subscription_Abc123',
  'actorDid': ?actorDid,
  'subjectUri': ?subjectUri,
  'rootUri': ?rootUri,
  'sourceUri': ?sourceUri,
};
