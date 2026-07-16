import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/foreground_notification_handler.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-018 / AT-003 forwards equal events without deduplication',
    () async {
      final banners = <ForegroundNotificationEvent>[];
      var listInvalidations = 0;
      var countRefreshes = 0;
      final handler = ForegroundNotificationHandler(
        showBanner: banners.add,
        invalidateList: () => listInvalidations++,
        refreshCount: () => countRefreshes++,
      );
      final event = ForegroundNotificationEvent(
        title: 'New activity',
        body: 'Someone interacted with your work',
        openEvent: NotificationOpenEvent(
          notificationId: NotificationId.parse(
            '00000000-0000-0000-0000-000000000001',
          ),
          category: NotificationCategory.like,
          accountSubscriptionId: AccountSubscriptionId.parse('binding'),
          source: NotificationOpenSource.foregroundBanner,
        ),
      );

      await handler.handle(event);
      await handler.handle(event);

      expect(banners, [same(event), same(event)]);
      expect(listInvalidations, 2);
      expect(countRefreshes, 2);
    },
  );
}
