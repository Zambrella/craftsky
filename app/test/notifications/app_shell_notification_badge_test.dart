import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

void main() {
  testWidgets(
    'IT-008 compact shell shows an accessible capped badge',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await _pumpShell(tester, const Size(500, 800));
      expect(find.byType(NavigationBar), findsOneWidget);
      expect(find.byType(NavigationRail), findsNothing);
      expect(find.text('99+'), findsOneWidget);
      expect(
        find.bySemanticsLabel(RegExp('137 new activities')),
        findsOneWidget,
      );
      semantics.dispose();
    },
  );

  testWidgets(
    'IT-008 large shell shows the same accessible capped badge on its rail',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await _pumpShell(tester, const Size(1100, 800));
      expect(find.byType(NavigationBar), findsNothing);
      expect(find.byType(NavigationRail), findsOneWidget);
      expect(find.text('99+'), findsOneWidget);
      expect(
        find.bySemanticsLabel(RegExp('137 new activities')),
        findsOneWidget,
      );
      semantics.dispose();
    },
  );
}

Future<void> _pumpShell(WidgetTester tester, Size size) async {
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  final router = _buildRouter();
  addTearDown(router.dispose);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        notificationNewnessRepositoryProvider.overrideWithValue(
          const _NewnessRepository(137),
        ),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => FormFactorWidget(child: child!),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

GoRouter _buildRouter() => GoRouter(
  initialLocation: '/feed',
  routes: [
    StatefulShellRoute.indexedStack(
      builder: (context, state, navigationShell) =>
          AppShell(navigationShell: navigationShell),
      branches: [
        for (final path in [
          '/feed',
          '/projects',
          '/search',
          '/notifications',
          '/profile',
        ])
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: path,
                builder: (context, state) => Text(path),
              ),
            ],
          ),
      ],
    ),
  ],
);

final class _NewnessRepository implements NotificationNewnessRepository {
  const _NewnessRepository(this.value);

  final int value;

  @override
  Future<int> count() async => value;

  @override
  Future<void> markSeen() async {}
}
