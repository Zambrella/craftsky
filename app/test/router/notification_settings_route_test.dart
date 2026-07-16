import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/pages/notification_settings_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('IT-009 typed settings route pushes full-screen and pops back', (
    tester,
  ) async {
    expect(
      const NotificationSettingsRoute().location,
      '/notifications/settings',
    );
    final notifications = _NotificationsRepository();
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        onboardingStatusProvider.overrideWith2(
          (_) => CompletedOnboardingStatus(),
        ),
        notificationRepositoryProvider.overrideWithValue(notifications),
        notificationNewnessRepositoryProvider.overrideWithValue(
          const _NewnessRepository(),
        ),
        notificationPreferencesRepositoryProvider.overrideWithValue(
          const _PreferencesRepository(),
        ),
      ],
    );
    final routerSubscription = container.listen(
      goRouterProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(routerSubscription.close);
    final router = container.read(goRouterProvider)..go('/notifications');

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MessengerScope(
            messenger: RecordingMessenger(),
            child: FormFactorWidget(child: child!),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(NotificationsPage), findsOneWidget);
    expect(notifications.listCalls, 1);

    expect(find.byTooltip('Notification settings'), findsOneWidget);
    final settingsButton = find.widgetWithIcon(
      IconButton,
      Icons.settings_outlined,
    );
    expect(GoRouter.of(tester.element(settingsButton)), same(router));
    tester.widget<IconButton>(settingsButton).onPressed!();
    await tester.pumpAndSettle();

    expect(find.byType(NotificationSettingsPage), findsOneWidget);
    expect(
      GoRouterState.of(
        tester.element(find.byType(NotificationSettingsPage)),
      ).matchedLocation,
      '/notifications/settings',
    );
    expect(find.byType(NavigationBar), findsNothing);

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(find.byType(NotificationsPage), findsOneWidget);
    expect(
      GoRouterState.of(
        tester.element(find.byType(NotificationsPage)),
      ).matchedLocation,
      '/notifications',
    );
    expect(notifications.listCalls, 1);
  });
}

final class _NotificationsRepository implements NotificationRepository {
  int listCalls = 0;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) async {
    listCalls++;
    return const NotificationPage(items: []);
  }
}

final class _NewnessRepository implements NotificationNewnessRepository {
  const _NewnessRepository();

  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}

final class _PreferencesRepository
    implements NotificationPreferencesRepository {
  const _PreferencesRepository();

  NotificationPreferences get preferences => NotificationPreferences(
    known: {
      for (final category in NotificationCategory.preferenceValues)
        category: const NotificationPreference(
          scope: NotificationPreferenceScope.everyone,
          pushEnabled: true,
        ),
    },
    unknown: const {},
  );

  @override
  Future<NotificationPreferences> load() async => preferences;

  @override
  Future<NotificationPreferences> patch(
    NotificationPreferencePatch patch,
  ) async => preferences;
}
