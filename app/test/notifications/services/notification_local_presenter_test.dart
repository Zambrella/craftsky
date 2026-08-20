import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_envelope.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-PUSH-017 removed account cannot present or navigate', () async {
    final gateway = _Gateway();
    final dedupe = _Dedupe();
    final presenter = NotificationLocalPresenter(
      gateway: gateway,
      dedupe: dedupe,
      eligibility: const _Eligibility(allowed: false),
    );
    final envelope = _envelope(
      '00000000-0000-4000-8000-000000000009',
    );
    await presenter.initialize();

    await presenter.present(envelope);
    expect(await presenter.claimForegroundEffect(envelope), isFalse);
    gateway.open(envelope.localOpenPayload);
    await Future<void>.delayed(Duration.zero);

    expect(gateway.presentations, isEmpty);
    expect(dedupe.claims, isEmpty);
  });

  test(
    'UT-PUSH-019 account removed during dedupe cannot present',
    () async {
      final gateway = _Gateway();
      final dedupe = _Dedupe();
      final presenter = NotificationLocalPresenter(
        gateway: gateway,
        dedupe: dedupe,
        eligibility: _SequenceEligibility([true, false]),
      );

      await presenter.present(
        _envelope('00000000-0000-4000-8000-000000000019'),
      );

      expect(dedupe.claims, hasLength(1));
      expect(gateway.presentations, isEmpty);
    },
  );

  test(
    'UT-PUSH-009 five distinct events keep full tags and one duplicate entry',
    () async {
      final gateway = _Gateway();
      final presenter = NotificationLocalPresenter(
        gateway: gateway,
        dedupe: _Dedupe(),
        eligibility: const _Eligibility(allowed: true),
      );
      final envelopes = [
        for (var index = 10; index < 15; index++)
          _envelope(
            '00000000-0000-4000-8000-${index.toString().padLeft(12, '0')}',
          ),
      ];

      await Future.wait([
        presenter.present(envelopes.first),
        presenter.present(envelopes.first),
      ]);
      for (final envelope in envelopes.skip(1)) {
        await presenter.present(envelope);
      }

      expect(gateway.presentations, hasLength(5));
      expect(
        gateway.presentations.map((presentation) => presentation.id).toSet(),
        {NotificationLocalPresenter.androidTypeId},
      );
      expect(
        gateway.presentations.map((presentation) => presentation.tag).toSet(),
        envelopes.map((envelope) => envelope.notificationId).toSet(),
      );
      expect(
        gateway.presentations.every(
          (presentation) => presentation.tag.length == 36,
        ),
        isTrue,
      );
    },
  );

  test(
    'UT-PUSH-010 foreground, presentation, and open stages are separate',
    () async {
      final gateway = _Gateway();
      final presenter = NotificationLocalPresenter(
        gateway: gateway,
        dedupe: _Dedupe(),
        eligibility: const _Eligibility(allowed: true),
      );
      final envelope = _envelope('00000000-0000-4000-8000-000000000012');
      await presenter.initialize();

      expect(await presenter.claimForegroundEffect(envelope), isTrue);
      expect(await presenter.claimForegroundEffect(envelope), isFalse);
      await presenter.present(envelope);
      expect(gateway.presentations, hasLength(1));

      final opened = presenter.openedNotifications.first;
      gateway.open(envelope.localOpenPayload);
      expect((await opened).facts, isA<ValidNotificationFacts>());
      gateway.open(envelope.localOpenPayload);
      await Future<void>.delayed(Duration.zero);
      expect(gateway.openCallbacks, 2);
    },
  );

  test(
    'UT-PUSH-011 duplicate local opens route once across reconstruction',
    () async {
      final dedupe = _Dedupe();
      final firstGateway = _Gateway();
      final secondGateway = _Gateway();
      final first = NotificationLocalPresenter(
        gateway: firstGateway,
        dedupe: dedupe,
        eligibility: const _Eligibility(allowed: true),
      );
      final reconstructed = NotificationLocalPresenter(
        gateway: secondGateway,
        dedupe: dedupe,
        eligibility: const _Eligibility(allowed: true),
      );
      final envelope = _envelope('00000000-0000-4000-8000-000000000013');
      await first.initialize();
      await reconstructed.initialize();

      final firstOpen = first.openedNotifications.first;
      firstGateway.open(envelope.localOpenPayload);
      await firstOpen;
      var reconstructedOpened = false;
      final subscription = reconstructed.openedNotifications.listen(
        (_) => reconstructedOpened = true,
      );
      secondGateway.open(envelope.localOpenPayload);
      await Future<void>.delayed(Duration.zero);

      expect(reconstructedOpened, isFalse);
      await subscription.cancel();
    },
  );

  test(
    'UT-PUSH-012 records before OS presentation and exposes crash residual',
    () async {
      final gateway = _Gateway()
        ..failure = StateError('OS presentation failed');
      final presenter = NotificationLocalPresenter(
        gateway: gateway,
        dedupe: _Dedupe(),
        eligibility: const _Eligibility(allowed: true),
      );
      final envelope = _envelope('00000000-0000-4000-8000-000000000014');

      await expectLater(presenter.present(envelope), throwsStateError);
      gateway.failure = null;
      await presenter.present(envelope);

      expect(
        gateway.presentations,
        hasLength(1),
        reason:
            'claim-before-present intentionally prefers duplicate suppression; '
            'a crash/failure can lose this local presentation',
      );
    },
  );
}

NotificationDeliveryEnvelope _envelope(String notificationId) =>
    NotificationDeliveryEnvelope.tryParse({
      'payloadVersion': '1',
      'type': 'like',
      'accountSubscriptionId': 'routing-account-one',
      'notificationId': notificationId,
      'displayTitle': 'Alice',
      'displayBody': 'liked your post',
      'subjectUri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
      'rootUri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
    })!;

final class _Dedupe implements NotificationDeliveryDedupeStore {
  final _claims = <String>{};
  Set<String> get claims => Set.unmodifiable(_claims);

  @override
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  }) async => _claims.add('$notificationId:${stage.name}');

  @override
  Future<void> clearAccountPartition(String accountPartition) async {}
}

final class _Eligibility implements NotificationPresentationEligibility {
  const _Eligibility({required this.allowed});

  final bool allowed;

  @override
  Future<bool> allows(AccountSubscriptionId accountSubscriptionId) async =>
      allowed;
}

final class _SequenceEligibility
    implements NotificationPresentationEligibility {
  _SequenceEligibility(this._answers);

  final List<bool> _answers;
  var _index = 0;

  @override
  Future<bool> allows(AccountSubscriptionId accountSubscriptionId) async {
    final position = _index < _answers.length ? _index : _answers.length - 1;
    final answer = _answers[position];
    _index++;
    return answer;
  }
}

final class _Gateway implements NotificationPresentationGateway {
  final presentations = <NotificationPresentation>[];
  void Function(String? payload)? _onOpen;
  StateError? failure;
  int openCallbacks = 0;

  @override
  Future<void> initialize({
    required void Function(String? payload) onOpen,
  }) async {
    _onOpen = onOpen;
  }

  void open(String? payload) {
    openCallbacks++;
    _onOpen?.call(payload);
  }

  @override
  Future<void> present(NotificationPresentation presentation) async {
    presentations.add(presentation);
    if (failure case final failure?) throw failure;
  }

  @override
  Future<String?> takeInitialOpenPayload() async => null;
}
