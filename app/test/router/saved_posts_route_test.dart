import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/drafts/pages/drafts_page.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
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
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  new(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

late SharedPreferences _preferences;

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
    _preferences = await SharedPreferences.getInstance();
  });
  test('UT-011 uses canonical static and redacted saved routes', () {
    expect(RouteLocations.savedPosts, '/saved');
    expect(const SavedPostsRoute().location, '/saved');
    expect(const ScheduledPostsRoute().location, '/scheduled');
    expect(const DraftsRoute().location, '/drafts');

    final folder = SavedPostFolder(
      id: 'private-folder-id',
      name: 'Private folder name',
      createdAt: DateTime.utc(2026, 7, 21),
      updatedAt: DateTime.utc(2026, 7, 21),
    );
    final extra = SavedPostFolderRouteData(folder: folder);
    final route = SavedPostFolderRoute($extra: extra);
    expect(route.location, '/saved/folder');
    expect(route.location, isNot(contains(folder.id)));
    expect(route.location, isNot(contains(folder.name)));
    expect(extra.copyWith(), extra);
    expect(extra.toString(), isNot(contains(folder.id)));
    expect(extra.toString(), isNot(contains(folder.name)));
    expect(route.toString(), isNot(contains(folder.id)));
    expect(route.toString(), isNot(contains(folder.name)));
  });

  testWidgets('IT-007 nests folders under Saved posts', (
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
      initialLocation: '/saved',
      routes: [
        GoRoute(
          path: '/feed',
          builder: (_, _) => const Scaffold(body: Text('Feed route')),
        ),
        GoRoute(
          path: '/saved',
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
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'test-token',
                did: account.did.value,
                handle: 'test.bsky.social',
              ),
            ),
          ),
          subscriptionAccessProvider.overrideWith(
            (ref, lease) async => SubscriptionAccess(
              did: lease.account.did,
              effectiveTier: SubscriptionTier.plus,
              givesAccess: true,
              assignedTier: SubscriptionTier.plus,
            ),
          ),
          accountSavedPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(SavedPostsPage), findsOneWidget);
    expect(router.state.matchedLocation, '/saved');

    await tester.tap(find.text('Ideas'));
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostFolderScreen), findsOneWidget);
    expect(router.state.matchedLocation, '/saved/folder');

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostsPage), findsOneWidget);
    expect(find.byType(BackButton), findsNothing);
  });

  testWidgets(
    'Saved posts is a peer destination with a folder back stack',
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
            localizationsDelegates: appLocalizationsDelegates,
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
        '/saved/folder',
      );
      expect(find.byType(SavedPostFolderScreen), findsOneWidget);
      expect(find.byType(SavedPostsPage), findsNothing);

      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byType(SavedPostsPage), findsOneWidget);

      expect(find.byType(BackButton), findsNothing);
      expect(router.canPop(), isFalse);
    },
  );

  for (final routePresentation in [
    (size: const Size(1200, 800), showsRail: true),
    (size: const Size(500, 800), showsRail: false),
  ]) {
    final presentationName = routePresentation.showsRail
        ? 'keeps the large rail'
        : 'stays full-screen on compact';
    testWidgets(
      'Saved posts $presentationName',
      (tester) async {
        final container = _productionContainer(
          savedRepository: const _RouteRepository(null),
        );
        addTearDown(container.dispose);
        final routerSubscription = container.listen(
          goRouterProvider,
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(routerSubscription.close);
        final router = routerSubscription.read()
          ..go(const SavedPostsRoute().location);

        await _pumpProductionRouter(
          tester,
          container,
          router,
          size: routePresentation.size,
        );

        expect(find.byType(SavedPostsPage), findsOneWidget);
        expect(
          find.byType(NavigationRail),
          routePresentation.showsRail ? findsOneWidget : findsNothing,
        );
        if (routePresentation.showsRail) {
          expect(
            tester
                .widget<NavigationRail>(find.byType(NavigationRail))
                .selectedIndex,
            5,
          );
        }
        expect(find.byType(NavigationBar), findsNothing);
      },
    );
  }

  testWidgets('pushing Saved posts does not cover the large rail', (
    tester,
  ) async {
    final container = _productionContainer(
      savedRepository: const _RouteRepository(null),
    );
    addTearDown(container.dispose);
    final routerSubscription = container.listen(
      goRouterProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(routerSubscription.close);
    final router = routerSubscription.read()..go(RouteLocations.feed);

    await _pumpProductionRouter(
      tester,
      container,
      router,
      size: const Size(1200, 800),
    );

    router.push<void>(const SavedPostsRoute().location).ignore();
    await tester.pumpAndSettle();

    expect(find.byType(SavedPostsPage), findsOneWidget);
    expect(find.byType(NavigationRail), findsOneWidget);
  });

  for (final routeCase in [
    (
      label: 'Settings',
      location: const SettingsRoute().location,
      pageType: SettingsPage,
    ),
    (
      label: 'Saved posts',
      location: const SavedPostsRoute().location,
      pageType: SavedPostsPage,
    ),
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
    for (final large in [false, true]) {
      testWidgets(
        '${routeCase.label} switches through the ${large ? 'rail' : 'drawer'}',
        (tester) async {
          final container = _productionContainer(
            savedRepository: const _RouteRepository(null),
          );
          addTearDown(container.dispose);
          final routerSubscription = container.listen(
            goRouterProvider,
            (_, _) {},
            fireImmediately: true,
          );
          addTearDown(routerSubscription.close);
          final router = routerSubscription.read();

          await _pumpProductionRouter(
            tester,
            container,
            router,
            size: Size(large ? 1200 : 500, 800),
          );
          expect(router.state.matchedLocation, RouteLocations.home);

          router.go(routeCase.location);
          await tester.pump();
          await tester.pump(const Duration(seconds: 1));

          expect(router.state.matchedLocation, routeCase.location);
          expect(
            find.byWidgetPredicate(
              (widget) => widget.runtimeType == routeCase.pageType,
            ),
            findsOneWidget,
          );

          expect(find.byType(BackButton), findsNothing);
          expect(router.canPop(), isFalse);
          if (large) {
            await tester.tap(
              find.descendant(
                of: find.byType(NavigationRail),
                matching: find.text('Home'),
              ),
            );
          } else {
            await tester.tap(find.byTooltip('Open navigation menu'));
            // Drafts and Scheduled can still be showing animated skeletons.
            await tester.pump();
            await tester.pump(const Duration(seconds: 1));
            final drawer = find.byType(Drawer);
            final tile = find.widgetWithText(
              ListTile,
              routeCase.label == 'Saved posts'
                  ? 'Saved'
                  : routeCase.label == 'Scheduled posts'
                  ? 'Scheduled'
                  : routeCase.label,
            );
            expect(
              tester
                  .widget<ListTile>(find.descendant(of: drawer, matching: tile))
                  .selected,
              isTrue,
            );
            await tester.tap(find.widgetWithText(ListTile, 'Home'));
          }
          await tester.pumpAndSettle();

          expect(router.state.matchedLocation, RouteLocations.feed);
          expect(find.byType(FeedPage), findsOneWidget);
        },
      );
    }
  }

  for (final large in [false, true]) {
    testWidgets(
      'section state survives ${large ? 'rail' : 'drawer'} switching',
      (tester) async {
        final folder = SavedPostFolder(
          id: 'folder',
          name: 'Ideas',
          createdAt: DateTime.utc(2026),
          updatedAt: DateTime.utc(2026),
        );
        final container = _productionContainer(
          savedRepository: _RouteRepository(folder),
        );
        addTearDown(container.dispose);
        final subscription = container.listen(
          goRouterProvider,
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(subscription.close);
        final router = subscription.read()..go(const SettingsRoute().location);
        await _pumpProductionRouter(
          tester,
          container,
          router,
          size: Size(large ? 1200 : 500, 800),
        );

        Future<void> select(String label) async {
          if (large) {
            await tester.tap(
              find.descendant(
                of: find.byType(NavigationRail),
                matching: find.text(label),
              ),
            );
          } else {
            await tester.dragFrom(const Offset(0, 300), const Offset(350, 0));
            await tester.pumpAndSettle();
            await tester.tap(find.widgetWithText(ListTile, label));
          }
          await tester.pumpAndSettle();
        }

        final settingsList = find.descendant(
          of: find.byType(SettingsPage),
          matching: find.byType(Scrollable),
        );
        await tester.drag(settingsList, const Offset(0, -400));
        await tester.pumpAndSettle();
        final position = tester.state<ScrollableState>(settingsList).position;
        final offset = position.pixels;
        expect(offset, greaterThan(0));
        await select('Saved');
        await select('Settings');
        expect(
          tester.state<ScrollableState>(settingsList).position,
          same(position),
        );
        expect(position.pixels, offset);

        // On compact child pages the edge gesture belongs to Back. The rail
        // remains available for switching sections while a child is open.
        if (!large) return;

        // Nested Settings and Saved pages retain their own branch stacks.
        router.go(const AppearanceRoute().location);
        await tester.pumpAndSettle();
        await select('Saved');
        await tester.tap(find.text('Ideas'));
        await tester.pumpAndSettle();
        expect(router.state.matchedLocation, '/saved/folder');
        await select('Settings');
        expect(router.state.matchedLocation, const AppearanceRoute().location);
        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(router.state.matchedLocation, const SettingsRoute().location);
        await select('Saved');
        expect(router.state.matchedLocation, '/saved/folder');
        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(router.state.matchedLocation, const SavedPostsRoute().location);
        expect(find.byType(BackButton), findsNothing);
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
      '/settings/saved',
      '/settings/saved/folder',
      '/settings/scheduled',
      '/settings/drafts',
      '/profile/saved',
      '/profile/saved/folder',
      '/profile/scheduled',
      '/profile/drafts',
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
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'token',
    did: account.did.value,
    handle: 'test.bsky.social',
  );
  return ProviderContainer(
    overrides: [
      sharedPreferencesProvider.overrideWithValue(_preferences),
      subscriptionAccessProvider.overrideWith(
        (ref, lease) async => SubscriptionAccess(
          did: lease.account.did,
          effectiveTier: SubscriptionTier.plus,
          givesAccess: true,
          assignedTier: SubscriptionTier.plus,
        ),
      ),
      authSessionProvider.overrideWith(SignedInAuthSession.new),
      secureSessionRegistryStorageProvider.overrideWithValue(
        _RegistryStorage(registry),
      ),
      activeAccountInitializationProvider.overrideWith(
        (_) => completedActiveAccountInitialization(),
      ),
      profileRepositoryProvider.overrideWithValue(
        FakeProfileRepository(
          onFetch: (_) async => Profile(
            did: account.did,
            handle: 'test.bsky.social',
            crafts: const [],
          ),
          onFetchMe: () async => Profile(
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
  GoRouter router, {
  Size? size,
}) async {
  if (size case final value?) {
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = value;
  }
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
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
  const new(this.folder);
  final SavedPostFolder? folder;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(items: [?folder]);
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
