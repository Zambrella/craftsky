import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-011 preserves unknown server entries outside known UI values', () {
    final preferences = NotificationPreferences.fromMap({
      'preferences': {
        'like': {'scope': 'everyone', 'pushEnabled': true},
        'futureCategory': {'scope': 'futureScope', 'pushEnabled': false},
      },
    });

    expect(
      preferences.known[NotificationCategory.like]?.pushEnabled,
      isTrue,
    );
    expect(preferences.unknown, contains('futureCategory'));
    expect(
      preferences.known,
      isNot(contains(NotificationCategory.everythingElse)),
    );
  });

  test('UT-009 builds exactly one-category one-field patches', () {
    expect(
      NotificationPreferencePatch.pushEnabled(
        NotificationCategory.like,
        value: false,
      ).toMap(),
      {
        'preferences': {
          'like': {'pushEnabled': false},
        },
      },
    );
    expect(
      NotificationPreferencePatch.scope(
        NotificationCategory.quote,
        value: NotificationPreferenceScope.peopleIFollow,
      ).toMap(),
      {
        'preferences': {
          'quote': {'scope': 'peopleIFollow'},
        },
      },
    );
  });

  test('preference values use generated mapping and copyWith', () {
    final preference = NotificationPreferenceMapper.fromMap({
      'scope': 'peopleIFollow',
      'pushEnabled': true,
    });

    expect(preference.scope, NotificationPreferenceScope.peopleIFollow);
    expect(preference.copyWith(pushEnabled: false).pushEnabled, isFalse);
    expect(preference.toMap(), {
      'scope': 'peopleIFollow',
      'pushEnabled': true,
    });
  });
}
