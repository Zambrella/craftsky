import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/pages/notification_settings_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'AT-008 renders social scopes and a push-only Instagram control',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            notificationPreferencesRepositoryProvider.overrideWithValue(
              const _Repository(),
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
      expect(find.byType(Switch), findsNWidgets(8));
      for (final category in NotificationCategory.preferenceValues) {
        expect(
          find.byKey(
            Key('notification-${category.wireValue}-preference-card'),
          ),
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
      expect(find.text('Instagram matches'), findsOneWidget);
      expect(
        find.textContaining('based on your Instagram migration eligibility'),
        findsOneWidget,
      );
      expect(find.byType(DropdownButtonFormField), findsNothing);
      expect(find.byType(SwitchListTile), findsNothing);
      expect(find.text('Everything else'), findsOneWidget);
      expect(find.text('futureCategory'), findsNothing);
      expect(find.text('Master switch'), findsNothing);

      await tester.tap(
        find.byKey(const Key('notification-like-push-switch')),
      );
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
    },
  );
}

final class _Repository implements NotificationPreferencesRepository {
  const _Repository();

  NotificationPreferences get preferences => NotificationPreferences(
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

  @override
  Future<NotificationPreferences> load() async => preferences;

  @override
  Future<NotificationPreferences> patch(
    NotificationPreferencePatch patch,
  ) async => preferences;
}
