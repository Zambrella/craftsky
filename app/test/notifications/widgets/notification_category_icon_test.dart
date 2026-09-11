import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_icon.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('moderation uses the privacy icon', () {
    expect(
      notificationCategoryIcon(NotificationCategory.moderation),
      CraftskyIcons.privacy,
    );
  });
}
