import 'dart:ui' show Tristate;

import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'compact shell drawer exposes navigation, legal links, and inert feedback',
    (tester) async {
      final router = await _pumpShell(tester, const Size(500, 800));

      await tester.dragFrom(
        const Offset(0, 400),
        const Offset(320, 0),
      );
      await tester.pumpAndSettle();

      final drawer = find.byType(Drawer);
      expect(drawer, findsOneWidget);
      for (final label in [
        'Feed',
        'Projects',
        'Search',
        'Notifications',
        'Profile',
        'Saved',
        'Scheduled',
        'Drafts',
        'Settings',
        'Terms',
        'Privacy',
        'Feedback',
      ]) {
        expect(
          find.descendant(of: drawer, matching: find.text(label)),
          findsOneWidget,
        );
      }
      expect(find.textContaining('@'), findsNothing);

      await tester.tap(
        find.descendant(of: drawer, matching: find.text('Feedback')),
      );
      await tester.pumpAndSettle();

      expect(router.state.matchedLocation, '/feed');
      expect(find.byType(Drawer), findsOneWidget);
    },
  );

  testWidgets('shell drawer button opens the owning compact drawer', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(500, 800));

    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsOneWidget);
    expect(
      find.descendant(of: find.byType(Drawer), matching: find.text('Feed')),
      findsOneWidget,
    );
  });

  testWidgets('UIP-004 compact drawer opens from the wider swipe region', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(500, 800));

    await tester.dragFrom(
      const Offset(80, 400),
      const Offset(320, 0),
    );
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsOneWidget);
  });

  testWidgets('UIP-001 drawer is outlined, divider-free, and icon-only', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    final drawerFinder = find.byType(Drawer);
    final drawer = tester.widget<Drawer>(drawerFinder);
    final shape = drawer.shape! as RoundedRectangleBorder;
    expect(shape.side.width, 1.5);
    expect(
      shape.side.color,
      Theme.of(tester.element(drawerFinder)).colorScheme.onSurface,
    );
    expect(
      shape.borderRadius.resolve(TextDirection.ltr).topRight.x,
      greaterThan(0),
    );
    expect(
      find.descendant(
        of: drawerFinder,
        matching: find.byType(CraftskyDivider),
      ),
      findsNothing,
    );

    final profileTile = find.widgetWithText(ListTile, 'Profile');
    expect(
      find.descendant(
        of: profileTile,
        matching: find.byType(AccountAvatar),
      ),
      findsNothing,
    );
    expect(
      find.descendant(
        of: profileTile,
        matching: find.byIcon(Icons.person_outline),
      ),
      findsOneWidget,
    );
  });

  testWidgets('UIP-002 rail is divider-free and uses a profile icon', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(1200, 800));

    expect(find.byType(NavigationRail), findsOneWidget);
    expect(find.byType(CraftskyDivider), findsNothing);
    expect(find.byType(AccountAvatar), findsNothing);
    expect(find.byIcon(Icons.person_outline), findsOneWidget);
  });

  testWidgets('UIP-009 rail selection uses primary theme colors', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(1200, 800));

    final railFinder = find.byType(NavigationRail);
    final theme = Theme.of(tester.element(railFinder));
    final railTheme = theme.navigationRailTheme;

    expect(railTheme.indicatorColor, theme.colorScheme.primary);
    expect(railTheme.selectedIconTheme?.color, theme.colorScheme.onPrimary);
    expect(
      railTheme.selectedLabelTextStyle?.color,
      theme.colorScheme.primary,
    );
  });

  testWidgets('UIP-012 rail has only a content-edge border', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(1200, 800));

    final railFinder = find.byType(NavigationRail);
    final outlinedSurface = find.ancestor(
      of: railFinder,
      matching: find.byWidgetPredicate(
        (widget) => widget is Material && widget.shape != null,
      ),
    );

    expect(outlinedSurface, findsOneWidget);
    final material = tester.widget<Material>(outlinedSurface);
    final shape = material.shape! as BorderDirectional;
    expect(shape.start, BorderSide.none);
    expect(shape.top, BorderSide.none);
    expect(shape.bottom, BorderSide.none);
    expect(shape.end.width, 1.5);
    expect(
      shape.end.color,
      Theme.of(tester.element(railFinder)).colorScheme.onSurface,
    );
    expect(material.clipBehavior, Clip.antiAlias);
  });

  testWidgets('CORR-004 hamburger exposes state and returns keyboard focus', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await _pumpShell(tester, const Size(500, 800));
    final menuButton = find.byTooltip('Open navigation menu');
    final menuSemantics = find.bySemanticsLabel('Open navigation menu');

    expect(
      tester.getSemantics(menuSemantics).flagsCollection.isExpanded,
      Tristate.isFalse,
    );

    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsOneWidget);
    expect(
      tester.getSemantics(menuSemantics).flagsCollection.isExpanded,
      Tristate.isTrue,
    );

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsNothing);
    expect(
      tester.getSemantics(menuSemantics).flagsCollection.isExpanded,
      Tristate.isFalse,
    );
    expect(
      tester
          .widgetList<Focus>(
            find.descendant(of: menuButton, matching: find.byType(Focus)),
          )
          .any((focus) => focus.focusNode?.hasFocus ?? false),
      isTrue,
    );
    semantics.dispose();
  });

  testWidgets('compact drawer opens secondary route and dismisses', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Saved'));
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/profile/saved');
    expect(find.byType(Drawer), findsNothing);
  });

  testWidgets('CORR-007 Back from personal content returns to Profile', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ListTile, 'Scheduled'));
    await tester.pumpAndSettle();
    expect(router.state.matchedLocation, '/profile/scheduled');

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/profile');
  });

  for (final (label, location) in const [
    ('Feed', '/feed'),
    ('Projects', '/projects'),
    ('Search', '/search'),
    ('Notifications', '/notifications'),
    ('Profile', '/profile'),
    ('Saved', '/profile/saved'),
    ('Scheduled', '/profile/scheduled'),
    ('Drafts', '/profile/drafts'),
    ('Settings', '/profile/settings'),
  ]) {
    testWidgets('CORR-005 compact drawer selects $label once', (tester) async {
      var navigations = 0;
      final router = await _pumpShell(
        tester,
        const Size(500, 800),
        onRedirect: (redirectedLocation) {
          if (redirectedLocation == location) navigations += 1;
        },
      );
      navigations = 0;
      await tester.tap(find.byTooltip('Open navigation menu'));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(ListTile, label));
      await tester.pumpAndSettle();

      expect(router.state.matchedLocation, location);
      expect(navigations, 1);
      expect(find.byType(Drawer), findsNothing);
    });
  }

  for (final (label, location) in const [
    ('Feed', '/feed'),
    ('Projects', '/projects'),
    ('Search', '/search'),
    ('Notifications', '/notifications'),
    ('Profile', '/profile'),
    ('Saved', '/profile/saved'),
    ('Scheduled', '/profile/scheduled'),
    ('Drafts', '/profile/drafts'),
    ('Settings', '/profile/settings'),
  ]) {
    testWidgets('CORR-005 large rail selects $label once', (tester) async {
      var navigations = 0;
      final router = await _pumpShell(
        tester,
        const Size(1200, 800),
        onRedirect: (redirectedLocation) {
          if (redirectedLocation == location) navigations += 1;
        },
      );
      navigations = 0;

      await tester.ensureVisible(find.text(label));
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();

      expect(router.state.matchedLocation, location);
      expect(navigations, 1);
    });
  }

  testWidgets('CORR-005 compact navigation surfaces retain active state', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(500, 800));

    final bottomBar = tester.widget<NavigationBar>(find.byType(NavigationBar));
    expect(bottomBar.destinations, hasLength(5));
    expect(bottomBar.selectedIndex, 0);

    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    expect(
      tester.widget<ListTile>(find.widgetWithText(ListTile, 'Feed')).selected,
      isTrue,
    );
    expect(
      tester
          .widgetList<ListTile>(
            find.descendant(
              of: find.byType(Drawer),
              matching: find.byType(ListTile),
            ),
          )
          .where((tile) => tile.selected),
      hasLength(1),
    );
  });

  testWidgets('CORR-005 scrim and system Back dismiss without navigation', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(500, 800));

    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();
    await tester.tapAt(const Offset(480, 400));
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsNothing);
    expect(router.state.matchedLocation, '/feed');

    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byType(Drawer), findsNothing);
    expect(router.state.matchedLocation, '/feed');
  });

  testWidgets('CORR-003 rapid drawer selection dispatches once', (
    tester,
  ) async {
    var savedNavigations = 0;
    final router = await _pumpShell(
      tester,
      const Size(500, 800),
      onRedirect: (location) {
        if (location == '/profile/saved') savedNavigations += 1;
      },
    );
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    final savedTile = tester.widget<ListTile>(
      find.widgetWithText(ListTile, 'Saved'),
    );
    savedTile.onTap!.call();
    savedTile.onTap!.call();
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/profile/saved');
    expect(savedNavigations, 1);
    expect(find.byType(Drawer), findsNothing);
  });

  testWidgets('CORR-003 rapid core selection dispatches once', (tester) async {
    var projectNavigations = 0;
    final router = await _pumpShell(
      tester,
      const Size(500, 800),
      onRedirect: (location) {
        if (location == '/projects') projectNavigations += 1;
      },
    );
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    final projectsTile = tester.widget<ListTile>(
      find.widgetWithText(ListTile, 'Projects'),
    );
    projectsTile.onTap!.call();
    projectsTile.onTap!.call();
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/projects');
    expect(projectNavigations, 1);
    expect(find.byType(Drawer), findsNothing);
  });

  testWidgets('legal links use the exact CraftSky URLs', (tester) async {
    final opened = <Uri>[];
    await _pumpShell(
      tester,
      const Size(500, 800),
      linkLauncher: (uri) async {
        opened.add(uri);
        return true;
      },
    );

    for (final label in ['Terms', 'Privacy']) {
      await tester.tap(find.byTooltip('Open navigation menu'));
      await tester.pumpAndSettle();
      await tester.tap(find.text(label));
      await tester.pumpAndSettle();
    }

    expect(opened, [
      Uri.parse('https://craftsky.social/terms'),
      Uri.parse('https://craftsky.social/privacy'),
    ]);
    expect(find.byType(Drawer), findsNothing);
  });

  testWidgets('UIP-005 legal link tap targets are compact', (tester) async {
    await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    for (final label in ['Terms', 'Privacy']) {
      expect(
        tester.getSize(find.widgetWithText(ListTile, label)).height,
        40,
      );
    }

    await _pumpShell(tester, const Size(1200, 800));

    for (final label in ['Terms', 'Privacy']) {
      expect(
        tester.getSize(find.widgetWithText(TextButton, label)).height,
        40,
      );
    }
  });

  testWidgets('UIP-007 menu uses compact Saved and Scheduled labels', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    final drawer = find.byType(Drawer);
    expect(find.descendant(of: drawer, matching: find.text('Saved')), findsOne);
    expect(
      find.descendant(of: drawer, matching: find.text('Scheduled')),
      findsOne,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('Saved posts')),
      findsNothing,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('Scheduled posts')),
      findsNothing,
    );

    await _pumpShell(tester, const Size(1200, 800));

    final rail = find.byType(NavigationRail);
    expect(find.descendant(of: rail, matching: find.text('Saved')), findsOne);
    expect(
      find.descendant(of: rail, matching: find.text('Scheduled')),
      findsOne,
    );
  });

  testWidgets('UIP-008 build version appears below Feedback', (tester) async {
    await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();

    final drawer = find.byType(Drawer);
    final drawerVersion = find.descendant(
      of: drawer,
      matching: find.text('1.0.0 (1)'),
    );
    expect(drawerVersion, findsOneWidget);
    expect(
      tester.getTopLeft(drawerVersion).dy,
      greaterThan(
        tester
            .getBottomLeft(
              find.descendant(
                of: drawer,
                matching: find.widgetWithText(OutlinedButton, 'Feedback'),
              ),
            )
            .dy,
      ),
    );

    await _pumpShell(tester, const Size(1200, 800));

    final rail = find.byType(NavigationRail);
    final railVersion = find.descendant(
      of: rail,
      matching: find.text('1.0.0 (1)'),
    );
    expect(railVersion, findsOneWidget);
    expect(
      tester.getTopLeft(railVersion).dy,
      greaterThan(
        tester
            .getBottomLeft(
              find.descendant(
                of: rail,
                matching: find.widgetWithText(OutlinedButton, 'Feedback'),
              ),
            )
            .dy,
      ),
    );
  });

  testWidgets('large shell rail exposes primary and utility destinations', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(1200, 500));

    expect(find.byType(NavigationRail), findsOneWidget);
    for (final label in [
      'Feed',
      'Projects',
      'Search',
      'Notifications',
      'Profile',
      'Saved',
      'Scheduled',
      'Drafts',
      'Settings',
      'Terms',
      'Privacy',
      'Feedback',
    ]) {
      expect(find.text(label), findsOneWidget);
    }
  });

  testWidgets('CORR-005 large rail is selected and Feedback remains inert', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(1200, 800));
    final rail = tester.widget<NavigationRail>(find.byType(NavigationRail));

    expect(rail.destinations, hasLength(9));
    expect(rail.selectedIndex, 0);
    expect(find.byType(NavigationBar), findsNothing);
    expect(find.byTooltip('Open navigation menu'), findsNothing);

    await tester.ensureVisible(find.text('Feedback'));
    await tester.tap(find.text('Feedback'));
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/feed');
    expect(find.byType(NavigationRail), findsOneWidget);
  });

  testWidgets('UIP-020 large pages share a centered maximum content width', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(1600, 800));
    final content = find.byKey(const Key('large-shell-content'));

    expect(content, findsOneWidget);
    expect(tester.getRect(content).width, 800);

    await tester.tap(find.text('Settings'));
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/profile/settings');
    expect(tester.getRect(content).width, 800);
  });

  testWidgets('UIP-022 desktop adds a fixed themed right-hand section', (
    tester,
  ) async {
    await _pumpShell(tester, const Size(1600, 800));

    final content = find.byKey(const Key('large-shell-content'));
    final divider = find.byKey(const Key('desktop-sidebar-divider'));
    final section = find.byKey(const Key('desktop-sidebar-placeholder'));

    expect(divider, findsOneWidget);
    expect(section, findsOneWidget);
    expect(tester.getSize(section), const Size.square(300));
    expect(
      tester.getRect(section).left,
      greaterThan(tester.getRect(content).right),
    );
    expect(tester.getSize(divider), const Size(1.5, 800));
    expect(
      tester.getRect(divider).left,
      greaterThanOrEqualTo(tester.getRect(content).right),
    );
    expect(
      tester.getRect(divider).right,
      lessThan(tester.getRect(section).left),
    );
    expect(
      find.descendant(of: section, matching: find.byType(CraftskyCard)),
      findsOneWidget,
    );

    await _pumpShell(tester, const Size(1200, 800));

    expect(divider, findsNothing);
    expect(section, findsNothing);
  });

  testWidgets('CORR-006 resizing an open drawer leaves one clean rail', (
    tester,
  ) async {
    final router = await _pumpShell(tester, const Size(500, 800));
    await tester.tap(find.byTooltip('Open navigation menu'));
    await tester.pumpAndSettle();
    expect(find.byType(Drawer), findsOneWidget);

    tester.view.physicalSize = const Size(1200, 800);
    await tester.pumpAndSettle();

    expect(router.state.matchedLocation, '/feed');
    expect(find.byType(Drawer), findsNothing);
    expect(find.byType(NavigationRail), findsOneWidget);
    expect(find.byType(NavigationBar), findsNothing);

    await tester.tap(find.text('Projects'));
    await tester.pumpAndSettle();
    expect(router.state.matchedLocation, '/projects');
  });

  testWidgets('CORR-006 RTL drawer opens from the logical start edge', (
    tester,
  ) async {
    await _pumpShell(
      tester,
      const Size(500, 800),
      textDirection: TextDirection.rtl,
    );

    await tester.dragFrom(
      const Offset(499, 400),
      const Offset(-320, 0),
    );
    await tester.pumpAndSettle();

    final drawer = find.byType(Drawer);
    expect(drawer, findsOneWidget);
    expect(tester.getTopRight(drawer).dx, closeTo(500, 1));
  });

  testWidgets(
    'CORR-006 drawer remains reachable at large text and low height',
    (
      tester,
    ) async {
      await _pumpShell(
        tester,
        const Size(500, 350),
        textScaler: const TextScaler.linear(2),
      );
      await tester.tap(find.byTooltip('Open navigation menu'));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      await tester.scrollUntilVisible(
        find.text('Settings'),
        100,
        scrollable: find.descendant(
          of: find.byType(Drawer),
          matching: find.byType(Scrollable),
        ),
      );
      expect(find.text('Settings'), findsOneWidget);
      expect(find.text('Terms'), findsOneWidget);
      expect(find.text('Privacy'), findsOneWidget);
      expect(find.text('Feedback'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('CORR-006 rail remains scrollable at large text and low height', (
    tester,
  ) async {
    await _pumpShell(
      tester,
      const Size(1200, 500),
      textScaler: const TextScaler.linear(2),
    );

    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.text('Feedback'));
    await tester.pumpAndSettle();
    expect(find.text('Feedback'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  for (final throws in [false, true]) {
    testWidgets(
      'CORR-006 legal launch ${throws ? 'throw' : 'false'} reports safely',
      (tester) async {
        final messenger = RecordingMessenger();
        final router = await _pumpShell(
          tester,
          const Size(500, 800),
          messenger: messenger,
          linkLauncher: (_) async {
            if (throws) throw StateError('platform detail');
            return false;
          },
        );
        await tester.tap(find.byTooltip('Open navigation menu'));
        await tester.pumpAndSettle();

        await tester.tap(find.widgetWithText(ListTile, 'Terms'));
        await tester.pumpAndSettle();

        expect(router.state.matchedLocation, '/feed');
        expect(find.byType(Drawer), findsNothing);
        expect(messenger.calls, [
          ('error', "Couldn't open that link.", null),
        ]);
      },
    );
  }
}

Future<GoRouter> _pumpShell(
  WidgetTester tester,
  Size size, {
  ExternalLinkLauncher? linkLauncher,
  ValueChanged<String>? onRedirect,
  TextDirection? textDirection,
  TextScaler? textScaler,
  RecordingMessenger? messenger,
}) async {
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  final router = GoRouter(
    initialLocation: '/feed',
    redirect: (context, state) {
      onRedirect?.call(state.matchedLocation);
      return null;
    },
    routes: [
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) => AppShell(
          navigationShell: navigationShell,
          linkLauncher: linkLauncher ?? (_) async => true,
          buildVersionLabel: '1.0.0 (1)',
        ),
        branches: [
          for (final path in [
            '/feed',
            '/projects',
            '/search',
            '/notifications',
          ])
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: path,
                  builder: (context, state) => Scaffold(
                    appBar: AppBar(leading: const AppShellDrawerButton()),
                    body: Text(path),
                  ),
                ),
              ],
            ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/profile',
                builder: (context, state) => Scaffold(
                  appBar: AppBar(leading: const AppShellDrawerButton()),
                  body: const Text('/profile'),
                ),
                routes: [
                  for (final path in [
                    'saved',
                    'scheduled',
                    'drafts',
                    'settings',
                  ])
                    GoRoute(
                      path: path,
                      builder: (context, state) =>
                          Scaffold(body: Text('/profile/$path')),
                    ),
                ],
              ),
            ],
          ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        notificationNewnessRepositoryProvider.overrideWithValue(
          const _ZeroNewnessRepository(),
        ),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) {
          Widget result = FormFactorWidget(child: child!);
          if (textDirection != null) {
            result = Directionality(
              textDirection: textDirection,
              child: result,
            );
          }
          if (textScaler != null) {
            result = MediaQuery(
              data: MediaQuery.of(context).copyWith(textScaler: textScaler),
              child: result,
            );
          }
          return MessengerScope(
            messenger: messenger ?? RecordingMessenger(),
            child: result,
          );
        },
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

final class _ZeroNewnessRepository implements NotificationNewnessRepository {
  const _ZeroNewnessRepository();

  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}
