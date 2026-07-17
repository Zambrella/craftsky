import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const notificationId = '018f47a2-4b0e-7f39-a621-9f6f6c75e312';
  const routingId = 'subscription_Abc123';

  group('UT-001 structured provider envelope', () {
    test('preserves binding validity independently from fact validity', () {
      final legacy = NotificationOpenAttempt.fromProviderData({
        'notificationId': notificationId,
        'type': 'like',
        'accountSubscriptionId': routingId,
      });
      final invalidBinding = NotificationOpenAttempt.fromProviderData({
        'payloadVersion': '1',
        'type': 'everythingElse',
        'accountSubscriptionId': 'contains spaces',
      });

      expect(
        legacy.accountSubscriptionId,
        AccountSubscriptionId.parse(routingId),
      );
      expect(legacy.facts, isA<InvalidNotificationFacts>());
      expect(invalidBinding.accountSubscriptionId, isNull);
      expect(invalidBinding.facts, isA<ValidNotificationFacts>());
    });

    test('ignores notificationId and keeps diagnostics redacted', () {
      final attempt = NotificationOpenAttempt.fromProviderData(
        {
          'payloadVersion': '1',
          'type': 'everythingElse',
          'accountSubscriptionId': routingId,
          'notificationId': notificationId,
        },
        source: NotificationOpenSource.initialOpen,
      );

      expect(attempt.source, NotificationOpenSource.initialOpen);
      expect(attempt.facts, isA<ValidNotificationFacts>());
      expect(attempt.toString(), isNot(contains(notificationId)));
      expect(attempt.toString(), isNot(contains(routingId)));
      expect(attempt.toString(), isNot(contains('notificationId')));
    });

    test('retains a valid binding when the type is malformed', () {
      final attempt = NotificationOpenAttempt.fromProviderData({
        'payloadVersion': '1',
        'type': 'not-valid!',
        'accountSubscriptionId': routingId,
      });

      expect(
        attempt.accountSubscriptionId,
        AccountSubscriptionId.parse(routingId),
      );
      expect(attempt.facts, isA<InvalidNotificationFacts>());
    });
  });

  test('UT-002 enforces required facts and ignores every extra', () {
    const actorDid = 'did:plc:alice';
    const subjectUri = 'at://did:plc:subject/social.craftsky.feed.post/subject';
    const rootUri = 'at://did:plc:root/social.craftsky.feed.post/root';
    const sourceUri = 'at://did:plc:source/social.craftsky.feed.post/source';
    final cases = <({String type, List<String> required})>[
      (type: 'follow', required: ['actorDid']),
      (type: 'like', required: ['subjectUri', 'rootUri']),
      (type: 'repost', required: ['subjectUri', 'rootUri']),
      (type: 'mention', required: ['sourceUri']),
      (type: 'quote', required: ['sourceUri']),
      (type: 'reply', required: ['subjectUri', 'sourceUri']),
      (type: 'everythingElse', required: []),
    ];

    for (final testCase in cases) {
      final data = <String, Object?>{
        'payloadVersion': '1',
        'type': testCase.type,
        'accountSubscriptionId': routingId,
        'actorDid': actorDid,
        'subjectUri': subjectUri,
        'rootUri': rootUri,
        'sourceUri': sourceUri,
        'route': '/must/not/be/used',
        'url': 'https://must.not.be/used.example',
      };
      final attempt = NotificationOpenAttempt.fromProviderData(data);
      final facts = attempt.facts as ValidNotificationFacts;

      expect(facts.category.wireValue, testCase.type);
      expect(
        facts.actorDid,
        testCase.required.contains('actorDid') ? Did.parse(actorDid) : null,
      );
      expect(
        facts.subjectUri,
        testCase.required.contains('subjectUri')
            ? AtUri.parse(subjectUri)
            : null,
      );
      expect(
        facts.rootUri,
        testCase.required.contains('rootUri') ? AtUri.parse(rootUri) : null,
      );
      expect(
        facts.sourceUri,
        testCase.required.contains('sourceUri') ? AtUri.parse(sourceUri) : null,
      );
      expect(attempt.toString(), isNot(contains('must.not.be.used')));

      for (final required in testCase.required) {
        final missing = NotificationOpenAttempt.fromProviderData(
          {
            ...data,
          }..remove(required),
        );
        expect(
          missing.facts,
          isA<InvalidNotificationFacts>(),
          reason: '${testCase.type} must require $required',
        );
        expect(
          missing.accountSubscriptionId,
          AccountSubscriptionId.parse(routingId),
        );
      }
    }
  });

  test('UT-003 and REG-009 reject untrusted public identifiers', () {
    const rootUri = 'at://did:plc:root/social.craftsky.feed.post/root';
    final invalid = <Map<String, Object?>>[
      _providerData(type: 'follow', actorDid: 'not-a-did'),
      _providerData(
        type: 'like',
        subjectUri: 'https://craftsky.social/posts/arbitrary',
        rootUri: rootUri,
      ),
      _providerData(
        type: 'like',
        subjectUri: 'at://did:plc:alice/social.craftsky.project/project',
        rootUri: rootUri,
      ),
      _providerData(
        type: 'like',
        subjectUri: 'at://not-a-did/social.craftsky.feed.post/post',
        rootUri: rootUri,
      ),
      _providerData(
        type: 'like',
        subjectUri: 'at://did:plc:alice/social.craftsky.feed.post/..',
        rootUri: rootUri,
      ),
      _providerData(type: 'follow', actorDid: 'did:plc:${'a' * 1017}'),
      _providerData(type: 'follow', actorDid: 'did:plc:unicodé'),
    ];

    for (final data in invalid) {
      final attempt = NotificationOpenAttempt.fromProviderData(data);
      expect(
        attempt.facts,
        isA<InvalidNotificationFacts>(),
        reason: 'must reject ${data['type']}',
      );
      expect(
        attempt.accountSubscriptionId,
        AccountSubscriptionId.parse(routingId),
      );
    }
  });

  test('UT-010 diagnostics expose classes but no routing identifiers', () {
    const actorDid = 'did:plc:privacyactor';
    const subjectUri =
        'at://did:plc:subject/social.craftsky.feed.post/privacy-subject';
    const sourceUri =
        'at://did:plc:source/social.craftsky.feed.post/privacy-focus';
    final attempt = NotificationOpenAttempt.fromProviderData({
      'payloadVersion': '1',
      'type': 'reply',
      'accountSubscriptionId': routingId,
      'actorDid': actorDid,
      'subjectUri': subjectUri,
      'sourceUri': sourceUri,
      'rawPayload': '{"private":"payload"}',
    });

    final diagnostics = '${attempt.facts} $attempt';
    expect(diagnostics, contains('reply'));
    for (final sentinel in [
      routingId,
      actorDid,
      subjectUri,
      sourceUri,
      'private',
      'payload',
    ]) {
      expect(diagnostics, isNot(contains(sentinel)));
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
