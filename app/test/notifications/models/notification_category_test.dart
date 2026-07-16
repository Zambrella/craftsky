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

  test('preference categories are the seven configurable values', () {
    expect(NotificationCategory.preferenceValues, hasLength(7));
    expect(
      NotificationCategory.preferenceValues,
      isNot(contains(NotificationCategory.unknown)),
    );
  });
}
