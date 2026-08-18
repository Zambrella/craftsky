import 'package:craftsky_app/notifications/services/firebase_notification_service.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:craftsky_app/notifications/services/notification_local_presenter.dart';
import 'package:craftsky_app/notifications/services/notification_presentation_eligibility.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-PUSH-013 Android data-only foreground duplicates emit one effect',
    () async {
      final presenter = NotificationLocalPresenter(
        gateway: _NoopGateway(),
        dedupe: _Dedupe(),
        eligibility: const _Eligibility(),
      );
      const message = RemoteMessage(
        data: {
          'payloadVersion': '1',
          'type': 'like',
          'accountSubscriptionId': 'routing-account-one',
          'notificationId': '00000000-0000-4000-8000-000000000020',
          'displayTitle': 'Alice',
          'displayBody': 'liked your post',
          'subjectUri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
          'rootUri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
        },
      );

      final first =
          await FirebaseNotificationService.foregroundEventFromMessage(
            message,
            presenter,
          );
      final duplicate =
          await FirebaseNotificationService.foregroundEventFromMessage(
            message,
            presenter,
          );

      expect(first?.title, 'Alice');
      expect(first?.body, 'liked your post');
      expect(duplicate, isNull);
    },
  );

  test(
    'UT-PUSH-014 APNs alert copy enters the same validated envelope',
    () async {
      final presenter = NotificationLocalPresenter(
        gateway: _NoopGateway(),
        dedupe: _Dedupe(),
        eligibility: const _Eligibility(),
      );
      const message = RemoteMessage(
        data: {
          'payloadVersion': '1',
          'type': 'everythingElse',
          'accountSubscriptionId': 'routing-account-one',
          'notificationId': '00000000-0000-4000-8000-000000000021',
        },
        notification: RemoteNotification(
          title: 'CraftSky',
          body: 'You have a new notification',
        ),
      );

      final event =
          await FirebaseNotificationService.foregroundEventFromMessage(
            message,
            presenter,
          );

      expect(event?.title, 'CraftSky');
      expect(event?.body, 'You have a new notification');
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
  final claims = <String>{};

  @override
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  }) async => claims.add('$notificationId:${stage.name}');

  @override
  Future<void> clearAccountPartition(String accountPartition) async {}
}

final class _NoopGateway implements NotificationPresentationGateway {
  @override
  Future<void> initialize({
    required void Function(String? payload) onOpen,
  }) async {}

  @override
  Future<void> present(NotificationPresentation presentation) async {}

  @override
  Future<String?> takeInitialOpenPayload() async => null;
}
