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

  test('preference categories include fixed-scope private categories', () {
    expect(NotificationCategory.preferenceValues, hasLength(9));
    expect(
      NotificationCategory.fromWireValue('instagramMatch'),
      NotificationCategory.instagramMatch,
    );
    expect(
      NotificationCategory.fromWireValue('moderation'),
      NotificationCategory.moderation,
    );
    expect(NotificationCategory.instagramMatch.hasFixedScope, isTrue);
    expect(NotificationCategory.moderation.hasFixedScope, isTrue);
    expect(NotificationCategory.like.hasFixedScope, isFalse);
    expect(
      NotificationCategory.preferenceValues,
      isNot(contains(NotificationCategory.unknown)),
    );
  });
}
