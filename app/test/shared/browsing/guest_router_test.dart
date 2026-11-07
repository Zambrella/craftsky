import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/home/pages/home_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/browsing/guest_session.dart';
import 'package:craftsky_app/theme/app_theme.dart';
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
  testWidgets(
    'AT-001/002 guest home shows craft feeds beneath Explore and keeps banner',
    (tester) async {
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
      await tester.ensureVisible(find.text('Explore CraftSky'));
      await tester.tap(find.text('Explore CraftSky'));
      await tester.pumpAndSettle();
      expect(find.text('Explore CraftSky'), findsNothing);
      expect(find.text('Sign in / Register'), findsOneWidget);
      expect(find.text('All crafts'), findsOneWidget);
      expect(find.text('Following'), findsNothing);
      expect(find.byTooltip('Compose'), findsNothing);
      router.go('/sign-in');
      await tester.pumpAndSettle();
      expect(find.text('Explore CraftSky'), findsOneWidget);
      await tester.ensureVisible(find.text('Explore CraftSky'));
      await tester.tap(find.text('Explore CraftSky'));
      await tester.pumpAndSettle();
      expect(find.text('Sign in / Register'), findsOneWidget);
      expect(find.text('Explore CraftSky'), findsNothing);
      expect(
        router.routerDelegate.currentConfiguration.uri.queryParameters
            .containsKey('auth'),
        isFalse,
      );
    },
  );
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

  testWidgets('AT-007 shared post hides guest mutations and chat sharing', (
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
              post: postFixture(rkey: 'guest-post', text: 'public guest post'),
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
    expect(find.text('0 likes'), findsOneWidget);
    expect(find.byTooltip('Like'), findsNothing);
    expect(find.byTooltip('Repost'), findsNothing);
    expect(mutated, false);
  });
}
