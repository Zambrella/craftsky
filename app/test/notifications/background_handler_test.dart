import 'package:craftsky_app/notifications/services/firebase_notification_background_handler.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const data = <String, String>{
    'payloadVersion': '1',
    'type': 'everythingElse',
    'accountSubscriptionId': 'routing-account-one',
    'notificationId': '00000000-0000-4000-8000-000000000030',
    'displayTitle': 'CraftSky',
    'displayBody': 'You have a new notification',
  };

  test(
    'IT-PUSH-015 background entry point initializes before presentation',
    () async {
      final operations = <String>[];

      await firebaseMessagingBackgroundHandler(
        const RemoteMessage(data: data),
        initializeFirebase: () async {
          operations.add('initialize');
        },
        presentData: (data) async => operations.add('present:${data['type']}'),
      );

      expect(operations, ['initialize', 'present:everythingElse']);
    },
  );

  test(
    'IT-PUSH-016 reconstructed background handlers present a duplicate once',
    () async {
      final dedupe = _Dedupe();
      final firstGateway = _Gateway();
      final reconstructedGateway = _Gateway();
      final first = NotificationLocalPresenter(
        gateway: firstGateway,
        dedupe: dedupe,
        eligibility: const _Eligibility(),
      );
      final reconstructed = NotificationLocalPresenter(
        gateway: reconstructedGateway,
        dedupe: dedupe,
        eligibility: const _Eligibility(),
      );

      await presentBackgroundNotificationData(data, first);
      await presentBackgroundNotificationData(data, reconstructed);

      expect(firstGateway.presentations, 1);
      expect(reconstructedGateway.presentations, 0);
    },
  );
}

final class _Eligibility implements NotificationPresentationEligibility {
  const _Eligibility();

  @override
  Future<bool> allows(AccountSubscriptionId accountSubscriptionId) async =>
      true;
}

final class _Dedupe implements NotificationDeliveryDedupeStore {
  final _claims = <String>{};

  @override
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  }) async => _claims.add('$notificationId:${stage.name}');

  @override
  Future<void> clearAccountPartition(String accountPartition) async {}
}

final class _Gateway implements NotificationPresentationGateway {
  int presentations = 0;

  @override
  Future<void> initialize({
    required void Function(String? payload) onOpen,
  }) async {}

  @override
  Future<void> present(NotificationPresentation presentation) async {
    presentations++;
  }

  @override
  Future<String?> takeInitialOpenPayload() async => null;
}
