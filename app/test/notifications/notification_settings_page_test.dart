import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/pages/notification_settings_page.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'AT-011 renders fixed-scope preferences and reloads moderation toggle',
    (tester) async {
      final repository = _Repository();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationPreferencesRepositoryProvider.overrideWithValue(
              repository,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const NotificationSettingsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Notification settings'), findsOneWidget);
      expect(find.textContaining('all devices'), findsOneWidget);
      expect(find.byType(Switch), findsNWidgets(9));
      for (final category in NotificationCategory.preferenceValues) {
        expect(
          find.byKey(Key('notification-${category.wireValue}-preference-card')),
          findsOneWidget,
        );
      }
      expect(
        find.byWidgetPredicate(
          (widget) =>
              widget is CraftskySingleSelectInput<NotificationPreferenceScope>,
        ),
        findsNWidgets(7),
      );
      expect(
        find.byKey(const Key('notification-instagramMatch-scope')),
        findsNothing,
      );
      expect(
        find.byKey(const Key('notification-moderation-scope')),
        findsNothing,
      );
      expect(find.text('Instagram matches'), findsOneWidget);
      expect(
        find.textContaining('never name the matched account'),
        findsOneWidget,
      );
      expect(find.text('Moderation'), findsOneWidget);
      expect(
        find.textContaining('does not change your standing'),
        findsOneWidget,
      );
      expect(find.byType(DropdownButtonFormField), findsNothing);
      expect(find.byType(SwitchListTile), findsNothing);
      expect(find.text('Everything else'), findsOneWidget);
      expect(find.text('futureCategory'), findsNothing);
      expect(find.text('Master switch'), findsNothing);

      final moderationSwitch = find.byKey(
        const Key('notification-moderation-push-switch'),
      );
      await tester.ensureVisible(moderationSwitch);
      await tester.tap(moderationSwitch);
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      expect(repository.patches.single.toMap(), {
        'preferences': {
          'moderation': {'pushEnabled': false},
        },
      });
      expect(
        repository.current.known[NotificationCategory.like]!.pushEnabled,
        isTrue,
      );

      final context = tester.element(find.byType(NotificationSettingsPage));
      ProviderScope.containerOf(
        context,
      ).invalidate(notificationPreferencesProvider);
      await tester.pumpAndSettle();

      expect(repository.loadCount, 2);
      expect(tester.widget<Switch>(moderationSwitch).value, isFalse);
      expect(
        tester
            .widget<Switch>(
              find.byKey(const Key('notification-like-push-switch')),
            )
            .value,
        isTrue,
      );
    },
  );
}

final class _Repository implements NotificationPreferencesRepository {
  NotificationPreferences current = NotificationPreferences(
    known: {
      for (final category in NotificationCategory.preferenceValues)
        category: const NotificationPreference(
          scope: NotificationPreferenceScope.everyone,
          pushEnabled: true,
        ),
    },
    unknown: const {
      'futureCategory': {'scope': 'everyone', 'pushEnabled': true},
    },
  );
  final patches = <NotificationPreferencePatch>[];
  int loadCount = 0;

  @override
  Future<NotificationPreferences> load() async {
    loadCount += 1;
    return current;
  }

  @override
  Future<NotificationPreferences> patch(
    NotificationPreferencePatch patch,
  ) async {
    patches.add(patch);
    final previous = current.known[patch.category]!;
    return current = current.replace(patch.category, switch (patch.field) {
      NotificationPreferenceField.scope => previous.copyWith(
        scope: patch.scopeValue,
      ),
      NotificationPreferenceField.pushEnabled => previous.copyWith(
        pushEnabled: patch.pushEnabledValue,
      ),
    });
  }
}
