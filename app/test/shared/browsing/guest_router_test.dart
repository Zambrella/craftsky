import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/home/pages/home_page.dart';
import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/browsing/guest_session.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../projects/fakes/fake_project_repository.dart';
import '../../search/fakes/fake_search_repository.dart';
import '../../test_support/post_fixtures.dart';

class _EmptyRegistry extends SessionRegistry {
  @override
  Future<model.SessionRegistry> build() async => model.SessionRegistry.empty();
}

void main() {
  setUpAll(initializeMappers);
  testWidgets('guest post opens the existing report form without signing in', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [sessionRegistryProvider.overrideWith(_EmptyRegistry.new)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: PostCard(
              post: postFixture(rkey: 'guest-report', text: 'Public post'),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Report post'));
    await tester.pumpAndSettle();
    expect(find.byType(ReportSubjectSheet), findsOneWidget);
    expect(find.byType(WelcomePage), findsNothing);
  });
  for (final size in [
    const Size(320, 932),
    const Size(430, 932),
    const Size(820, 1180),
    const Size(1100, 900),
    const Size(1600, 1000),
  ]) {
    testWidgets(
      'AT-001/002 guest home shows craft feeds beneath Explore and keeps banner at $size',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = size;
        addTearDown(tester.view.resetDevicePixelRatio);
        addTearDown(tester.view.resetPhysicalSize);
        final container = ProviderContainer.test(
          overrides: [
            sessionRegistryProvider.overrideWith(_EmptyRegistry.new),
            projectRepositoryProvider.overrideWithValue(
              FakeProjectRepository(
                onListProjects: ({required query, limit, cursor}) async =>
                    const PostPage(items: []),
              ),
            ),
          ],
        );
        final subscription = container.listen(goRouterProvider, (_, _) {});
        addTearDown(subscription.close);
        final router = subscription.read()..go('/feed?source=extended-network');
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
              builder: (context, child) => FormFactorWidget(child: child!),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.byType(HomePage), findsOneWidget);
        expect(find.text('Extended network'), findsNothing);
        expect(find.text('Your craft. Your community.'), findsNothing);
        expect(
          tester.widget<Dialog>(find.byType(Dialog)).backgroundColor,
          Colors.transparent,
        );
        expect(
          find.descendant(
            of: find.byType(WelcomePage),
            matching: find.byType(Scaffold),
          ),
          findsNothing,
        );
        await tester.ensureVisible(find.text('Explore CraftSky'));
        await tester.tap(find.text('Explore CraftSky'));
        await tester.pumpAndSettle();
        expect(find.text('Explore CraftSky'), findsNothing);
        expect(find.text('Your craft. Your community.'), findsOneWidget);
        expect(find.widgetWithText(TextButton, 'Sign in'), findsNothing);
        expect(find.widgetWithText(FilledButton, 'Register'), findsNothing);
        final join = find.widgetWithText(ChunkyButton, 'Join');
        expect(join, findsOneWidget);
        final messageBounds = tester.getRect(
          find.text('Your craft. Your community.'),
        );
        final buttonBounds = tester.getRect(join);
        final barBounds = tester.getRect(
          find.ancestor(of: join, matching: find.byType(Row)).first,
        );
        expect(messageBounds.left, closeTo(barBounds.left, 1));
        expect(buttonBounds.right, closeTo(barBounds.right, 1));
        expect(messageBounds.right, lessThan(buttonBounds.left));
        expect(messageBounds.center.dy, closeTo(buttonBounds.center.dy, 1));
        await tester.tap(join);
        await tester.pumpAndSettle();
        expect(find.text('Explore CraftSky'), findsOneWidget);
        expect(find.text('Your craft. Your community.'), findsNothing);
        await tester.ensureVisible(find.text('Explore CraftSky'));
        await tester.tap(find.text('Explore CraftSky'));
        await tester.pumpAndSettle();
        expect(find.text('All crafts'), findsOneWidget);
        expect(
          find.descendant(
            of: find.ancestor(
              of: find.text('All crafts'),
              matching: find.byType(Tab),
            ),
            matching: find.byIcon(CraftskyIcons.home),
          ),
          findsOneWidget,
        );
        expect(find.byTooltip('Filters'), findsNothing);
        expect(find.text('Following'), findsNothing);
        if (size.width > 900) {
          expect(find.byType(NavigationRail), findsOneWidget);
          expect(
            tester
                .widget<NavigationRail>(find.byType(NavigationRail))
                .destinations,
            hasLength(2),
          );
          expect(find.byType(NavigationBar), findsNothing);
          expect(find.text('Terms'), findsOneWidget);
          expect(find.text('Privacy'), findsOneWidget);
          expect(find.text('Feedback'), findsOneWidget);
        } else {
          expect(
            tester
                .widget<NavigationBar>(find.byType(NavigationBar))
                .destinations,
            hasLength(2),
          );
          await tester.tap(find.byTooltip('Open navigation menu'));
          await tester.pumpAndSettle();
          expect(find.text('Terms'), findsOneWidget);
          expect(find.text('Privacy'), findsOneWidget);
          expect(find.text('Feedback'), findsOneWidget);
          expect(find.widgetWithText(ListTile, 'Messages'), findsNothing);
          await tester.tap(find.widgetWithText(ListTile, 'Home'));
          await tester.pumpAndSettle();
        }

        expect(find.byTooltip('Compose'), findsNothing);
        router.go('/sign-in');
        await tester.pumpAndSettle();
        expect(find.text('Explore CraftSky'), findsOneWidget);
        expect(find.text('Your craft. Your community.'), findsNothing);
        await tester.ensureVisible(find.text('Explore CraftSky'));
        await tester.tap(find.text('Explore CraftSky'));
        await tester.pumpAndSettle();
        expect(find.text('Your craft. Your community.'), findsOneWidget);
        expect(find.text('Explore CraftSky'), findsNothing);
        expect(
          router.routerDelegate.currentConfiguration.uri.queryParameters
              .containsKey('auth'),
          isFalse,
        );
      },
    );
  }
  testWidgets(
    'AT-003 guest blank search and typing have no discovery requests',
    (tester) async {
      final repository = FakeSearchRepository();
      final container = ProviderContainer.test(
        overrides: [
          sessionRegistryProvider.overrideWith(_EmptyRegistry.new),
          searchRepositoryProvider.overrideWithValue(repository),
        ],
      );
      container.read(guestSessionProvider).explore();
      final subscription = container.listen(goRouterProvider, (_, _) {});
      addTearDown(subscription.close);
      final router = subscription.read()..go('/search');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: router,
            builder: (context, child) => FormFactorWidget(child: child!),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(SearchPage), findsOneWidget);
      expect(find.text('Recent searches'), findsNothing);
      expect(find.text('Top hashtags'), findsNothing);
      await tester.enterText(find.byType(TextField).first, 'sewing');
      await tester.pump(const Duration(seconds: 1));
      expect(tester.takeException(), isNull);
      expect(find.text('Submit a search to explore CraftSky'), findsOneWidget);
    },
  );

  testWidgets(
    'AT-007 shared post requests sign in without sending guest mutations',
    (
      tester,
    ) async {
      var mutated = false;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [sessionRegistryProvider.overrideWith(_EmptyRegistry.new)],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: PostCard(
                post: postFixture(
                  rkey: 'guest-post',
                  text: 'public guest post',
                ),
                onLike: () => mutated = true,
                onRepost: () => mutated = true,
                onQuote: (_) => mutated = true,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('public guest post'), findsOneWidget);
      expect(find.byKey(keys.feed.action('Like')), findsOneWidget);
      await tester.tap(find.byKey(keys.feed.action('Like')));
      await tester.pumpAndSettle();
      expect(find.text('Join CraftSky'), findsOneWidget);
      expect(mutated, false);
      await tester.ensureVisible(find.text('Explore CraftSky'));
      await tester.tap(find.text('Explore CraftSky'));
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(CraftskyIconsBold.repost));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Repost'));
      await tester.pumpAndSettle();
      expect(find.text('Join CraftSky'), findsOneWidget);
      expect(mutated, false);
    },
  );
}
