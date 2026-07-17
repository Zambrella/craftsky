import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
            data: _providerData(type: 'like', subjectUri: subjectUri),
            expected: PostDestination(AtUri.parse(subjectUri)),
          ),
          (
            data: _providerData(type: 'repost', subjectUri: subjectUri),
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
}

Map<String, Object?> _providerData({
  required String type,
  String? actorDid,
  String? subjectUri,
  String? sourceUri,
}) => <String, Object?>{
  'payloadVersion': '1',
  'type': type,
  'accountSubscriptionId': 'subscription_Abc123',
  'actorDid': ?actorDid,
  'subjectUri': ?subjectUri,
  'sourceUri': ?sourceUri,
};
