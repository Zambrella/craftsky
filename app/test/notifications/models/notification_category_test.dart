import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('shared category maps known and future wire values', () {
    expect(
      NotificationCategoryMapper.fromValue('quote'),
      NotificationCategory.quote,
    );
    expect(
      NotificationCategoryMapper.fromValue('futureCategory'),
      NotificationCategory.unknown,
    );
    expect(NotificationCategory.quote.toValue(), 'quote');
  });

  test('preference categories include the actorless Instagram category', () {
    expect(NotificationCategory.preferenceValues, hasLength(8));
    expect(
      NotificationCategory.preferenceValues,
      contains(NotificationCategory.instagramMatch),
    );
    expect(
      NotificationCategory.preferenceValues,
      isNot(contains(NotificationCategory.unknown)),
    );
  });
}
