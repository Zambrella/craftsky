import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/pages/saved_post_folder_page.dart';
import 'package:craftsky_app/saved_posts/pages/saved_posts_page.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/auth_session_fakes.dart';

void main() {
  test('UT-011 uses canonical static and redacted saved routes', () {
    expect(RouteLocations.savedPosts, '/profile/settings/saved');
    expect(const SavedPostsRoute().location, '/profile/settings/saved');

    final folder = SavedPostFolder(
      id: 'private-folder-id',
      name: 'Private folder name',
      createdAt: DateTime.utc(2026, 7, 21),
      updatedAt: DateTime.utc(2026, 7, 21),
    );
    final extra = SavedPostFolderRouteData(folder: folder);
    final route = SavedPostFolderRoute($extra: extra);
    expect(route.location, '/profile/settings/saved/folder');
    expect(route.location, isNot(contains(folder.id)));
    expect(route.location, isNot(contains(folder.name)));
    expect(extra.toString(), isNot(contains(folder.id)));
    expect(extra.toString(), isNot(contains(folder.name)));
    expect(route.toString(), isNot(contains(folder.id)));
    expect(route.toString(), isNot(contains(folder.name)));
  });

  testWidgets('IT-007 pushes Settings to overview to folder and back', (
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
      initialLocation: '/profile/settings',
      routes: [
        GoRoute(
          path: '/profile/settings',
          builder: (_, _) => const SettingsPage(),
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

    await tester.tap(find.text('Saved posts'));
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostsPage), findsOneWidget);
    expect(router.state.matchedLocation, '/profile/settings/saved');

    await tester.tap(find.text('Ideas'));
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostFolderScreen), findsOneWidget);
    expect(router.state.matchedLocation, '/profile/settings/saved/folder');

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byType(SavedPostsPage), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byType(SettingsPage), findsOneWidget);
  });
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
