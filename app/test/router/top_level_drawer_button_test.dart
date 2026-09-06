import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/projects/pages/projects_page.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../search/fakes/fake_search_repository.dart';

void main() {
  final pages = <String, Widget>{
    'Feed': const FeedPage(),
    'Projects': const ProjectsPage(),
    'Search': const SearchPage(),
    'Notifications': const NotificationsPage(),
    'Profile': const Scaffold(
      body: CustomScrollView(
        slivers: [
          ProfileSliverAppBar(
            handle: 'maker.test',
            actions: SelfProfileActionSet(
              onEdit: _doNothing,
              onSettings: _doNothing,
            ),
          ),
          SliverFillRemaining(),
        ],
      ),
    ),
  };

  for (final MapEntry(key: name, value: page) in pages.entries) {
    testWidgets('$name top-level app bar opens the shell drawer', (
      tester,
    ) async {
      var openCount = 0;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            if (name == 'Search')
              searchRepositoryProvider.overrideWithValue(
                FakeSearchRepository(),
              ),
            if (name == 'Notifications')
              notificationRepositoryProvider.overrideWithValue(
                const _EmptyNotificationRepository(),
              ),
            if (name == 'Notifications')
              notificationNewnessRepositoryProvider.overrideWithValue(
                const _EmptyNotificationRepository(),
              ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: AppShellDrawerScope(
              openDrawer: () => openCount += 1,
              isDrawerOpen: false,
              child: page,
            ),
          ),
        ),
      );
      await tester.pump();

      if (name == 'Projects') {
        expect(find.byType(CraftIcon), findsNWidgets(5));
      }

      await tester.tap(find.byTooltip('Open navigation menu'));

      expect(openCount, 1);
    });
  }
}

void _doNothing() {}

final class _EmptyNotificationRepository
    implements NotificationRepository, NotificationNewnessRepository {
  const _EmptyNotificationRepository();

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) async =>
      const NotificationPage(items: []);

  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}
