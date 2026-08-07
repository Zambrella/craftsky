import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/drafts/pages/drafts_page.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/pages/saved_post_folder_page.dart';
import 'package:craftsky_app/saved_posts/pages/saved_posts_page.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/scheduled_posts/pages/scheduled_posts_page.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  test('UT-011 uses canonical static and redacted saved routes', () {
    expect(RouteLocations.savedPosts, '/profile/saved');
    expect(const SavedPostsRoute().location, '/profile/saved');
    expect(const ScheduledPostsRoute().location, '/profile/scheduled');
    expect(const DraftsRoute().location, '/profile/drafts');

    final folder = SavedPostFolder(
      id: 'private-folder-id',
      name: 'Private folder name',
      createdAt: DateTime.utc(2026, 7, 21),
      updatedAt: DateTime.utc(2026, 7, 21),
    );
    final extra = SavedPostFolderRouteData(folder: folder);
    final route = SavedPostFolderRoute($extra: extra);
    expect(route.location, '/profile/saved/folder');
    expect(route.location, isNot(contains(folder.id)));
    expect(route.location, isNot(contains(folder.name)));
    expect(extra.copyWith(), extra);
    expect(extra.toString(), isNot(contains(folder.id)));
    expect(extra.toString(), isNot(contains(folder.name)));
    expect(route.toString(), isNot(contains(folder.id)));
    expect(route.toString(), isNot(contains(folder.name)));
  });

  testWidgets('IT-007 nests Saved posts and folders directly under Profile', (
    tester,
  ) async {
    final account = AccountKey('did:plc:test');
    final folder = SavedPostFolder(
      id: 'private-folder-id',
      name: 'Ideas',
      createdAt: DateTime.utc(2026, 7, 21),
      updatedAt: DateTime.utc(2026, 7, 21),
    );
    final repository = _RouteRepository(folder);
    final router = GoRouter(
      initialLocation: '/profile/saved',
      routes: [
        GoRoute(
          path: '/profile',
          builder: (_, _) => const Scaffold(body: Text('Profile route')),
          routes: [
            GoRoute(
              path: 'saved',
              name: 'saved-posts',
              builder: (_, _) => const SavedPostsPage(),
              routes: [
                GoRoute(
                  path: 'folder',
                  name: 'saved-post-folder',
                  builder: (_, state) {
                    final data = state.extra! as SavedPostFolderRouteData;
                    return SavedPostFolderScreen(
                      account: account,
                      folder: data.folder,
                    );
                  },
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
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          accountSavedPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(SavedPostsPage), findsOneWidget);
    expect(router.state.matchedLocation, '/profile/saved');

    await tester.tap(find.text('Ideas'));
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostFolderScreen), findsOneWidget);
    expect(router.state.matchedLocation, '/profile/saved/folder');

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostsPage), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Profile route'), findsOneWidget);
  });

  testWidgets(
    'Saved posts opens above Profile with the production router',
    (tester) async {
      final folder = SavedPostFolder(
        id: 'private-folder-id',
        name: 'Ideas',
        createdAt: DateTime.utc(2026, 7, 23),
        updatedAt: DateTime.utc(2026, 7, 23),
      );
      final container = _productionContainer(
        savedRepository: _RouteRepository(folder),
      );
      addTearDown(container.dispose);
      final routerSubscription = container.listen(
        goRouterProvider,
        (_, _) {},
        fireImmediately: true,
      );
      addTearDown(routerSubscription.close);
      final router = container.read(goRouterProvider)
        ..go(const SavedPostsRoute().location);

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

      expect(router.state.matchedLocation, RouteLocations.savedPosts);
      expect(find.byType(SavedPostsPage), findsOneWidget);

      await tester.tap(find.text('Ideas'));
      await tester.pumpAndSettle();

      expect(
        router.state.matchedLocation,
        '/profile/saved/folder',
      );
      expect(find.byType(SavedPostFolderScreen), findsOneWidget);
      expect(find.byType(SavedPostsPage), findsNothing);

      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byType(SavedPostsPage), findsOneWidget);

      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byType(ProfilePage), findsOneWidget);
    },
  );

  for (final routeCase in [
    (
      label: 'Scheduled posts',
      location: const ScheduledPostsRoute().location,
      pageType: ScheduledPostsPage,
    ),
    (
      label: 'Drafts',
      location: const DraftsRoute().location,
      pageType: DraftsPage,
    ),
  ]) {
    testWidgets(
      'CORR-007 ${routeCase.label} returns directly to Profile',
      (tester) async {
        final container = _productionContainer();
        addTearDown(container.dispose);
        final routerSubscription = container.listen(
          goRouterProvider,
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(routerSubscription.close);
        final router = routerSubscription.read();

        await _pumpProductionRouter(tester, container, router);
        expect(router.state.matchedLocation, RouteLocations.home);

        router.go(RouteLocations.profile);
        await tester.pump();
        await tester.pump(const Duration(seconds: 1));
        expect(router.state.matchedLocation, RouteLocations.profile);

        router.push<void>(routeCase.location).ignore();
        await tester.pump();
        await tester.pump(const Duration(seconds: 1));

        expect(router.state.matchedLocation, routeCase.location);
        expect(
          find.byWidgetPredicate(
            (widget) => widget.runtimeType == routeCase.pageType,
          ),
          findsOneWidget,
        );

        expect(find.byType(BackButton), findsOneWidget);
        await tester.tap(find.byType(BackButton));
        await tester.pumpAndSettle();

        expect(router.state.matchedLocation, RouteLocations.profile);
        expect(find.byType(ProfilePage), findsOneWidget);
      },
    );
  }

  testWidgets('CORR-007 former Settings-owned paths remain unknown', (
    tester,
  ) async {
    final container = _productionContainer();
    addTearDown(container.dispose);
    final routerSubscription = container.listen(
      goRouterProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(routerSubscription.close);
    final router = routerSubscription.read();

    await _pumpProductionRouter(tester, container, router);

    for (final oldLocation in [
      '/profile/settings/saved',
      '/profile/settings/saved/folder',
      '/profile/settings/scheduled',
      '/profile/settings/drafts',
    ]) {
      router.go(oldLocation);
      await tester.pumpAndSettle();

      expect(router.routeInformationProvider.value.uri.path, oldLocation);
      expect(find.byType(ErrorScreen), findsOneWidget);
      expect(find.byType(SavedPostsPage), findsNothing);
      expect(find.byType(ScheduledPostsPage), findsNothing);
      expect(find.byType(DraftsPage), findsNothing);
    }
  });
}

ProviderContainer _productionContainer({SavedPostRepository? savedRepository}) {
  final account = AccountKey('did:plc:test');
  return ProviderContainer(
    overrides: [
      authSessionProvider.overrideWith(SignedInAuthSession.new),
      onboardingStatusProvider.overrideWith2(
        (_) => CompletedOnboardingStatus(),
      ),
      profileRepositoryProvider.overrideWithValue(
        FakeProfileRepository(
          onFetch: (_) async => Profile(
            did: account.did,
            handle: 'test.bsky.social',
            crafts: const [],
          ),
        ),
      ),
      postRepositoryProvider.overrideWithValue(
        FakePostRepository(
          onListByAuthor: (_, {cursor, limit}) async =>
              const PostPage(items: []),
        ),
      ),
      if (savedRepository != null)
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => savedRepository),
    ],
    retry: (_, _) => null,
  );
}

Future<void> _pumpProductionRouter(
  WidgetTester tester,
  ProviderContainer container,
  GoRouter router,
) async {
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
  await tester.pump();
  await tester.pump(const Duration(seconds: 1));
}

final class _RouteRepository implements SavedPostRepository {
  const _RouteRepository(this.folder);
  final SavedPostFolder folder;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(items: [folder]);
  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async => const SavedPostPage(items: []);
  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      throw UnimplementedError();
  @override
  Future<void> unsave(Post post) => throw UnimplementedError();
  @override
  Future<SavedPostFolder> createFolder(String name) =>
      throw UnimplementedError();
  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();
  @override
  Future<void> deleteFolder(String folderId, {required bool deleteSaves}) =>
      throw UnimplementedError();
}
