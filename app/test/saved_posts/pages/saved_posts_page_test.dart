import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/pages/saved_posts_page.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('AT-004 renders folders before the Unfiled collection', (
    tester,
  ) async {
    final repository = _OverviewRepository(
      folders: SavedPostFolderPage(
        items: [_folder('folder-a', 'Ideas'), _folder('folder-b', 'Later')],
      ),
      unfiled: SavedPostPage(items: [_item('saved-row')]),
    );
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    expect(find.text('Saved posts'), findsOneWidget);
    expect(find.text('Folders'), findsOneWidget);
    expect(find.text('Ideas'), findsOneWidget);
    expect(find.text('Later'), findsOneWidget);
    expect(find.text('Unfiled'), findsOneWidget);
    expect(find.text('saved-row'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Later')).dy,
      lessThan(tester.getTopLeft(find.text('Unfiled')).dy),
    );
    expect(find.textContaining(RegExp(r'\d+ posts')), findsNothing);
  });

  testWidgets('AT-004 shows full empty state only when both sections empty', (
    tester,
  ) async {
    await _pump(
      tester,
      _OverviewRepository(
        folders: const SavedPostFolderPage(items: []),
        unfiled: const SavedPostPage(items: []),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Nothing saved yet'), findsOneWidget);
    expect(find.text('Unfiled'), findsNothing);
  });

  testWidgets('IT-008 refreshes a fully empty overview into hierarchy', (
    tester,
  ) async {
    final repository = _OverviewRefreshRepository();
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    expect(find.text('Nothing saved yet'), findsOneWidget);
    expect(find.byType(RefreshIndicator), findsOneWidget);

    await tester.fling(
      find.byType(CustomScrollView),
      const Offset(0, 320),
      1000,
    );
    await tester.pumpAndSettle();

    expect(find.text('Nothing saved yet'), findsNothing);
    expect(find.text('Ideas'), findsOneWidget);
    expect(find.text('refreshed-unfiled'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Ideas')).dy,
      lessThan(tester.getTopLeft(find.text('Unfiled')).dy),
    );
    expect(repository.folderCalls, 2);
    expect(repository.unfiledCalls, 2);
  });

  testWidgets('AT-007 creates a folder from the overview action', (
    tester,
  ) async {
    final repository = _OverviewRepository(
      folders: const SavedPostFolderPage(items: []),
      unfiled: const SavedPostPage(items: []),
    )..createdFolder = _folder('created-id', 'Fresh ideas');
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('New folder'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '  Fresh ideas  ');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    expect(repository.createNames, ['Fresh ideas']);
    expect(find.text('Fresh ideas'), findsOneWidget);
  });

  testWidgets('IT-008 retains overview scroll across folder mutation', (
    tester,
  ) async {
    final repository = _OverviewRepository(
      folders: SavedPostFolderPage(
        items: [
          for (var i = 0; i < 18; i++) _folder('folder-$i', 'Folder $i'),
        ],
      ),
      unfiled: const SavedPostPage(items: []),
    )..createdFolder = _folder('created-id', 'Created while scrolled');
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    await tester.drag(find.byType(CustomScrollView), const Offset(0, -320));
    await tester.pumpAndSettle();
    final before = tester
        .state<ScrollableState>(find.byType(Scrollable))
        .position
        .pixels;
    expect(before, greaterThan(0));

    await tester.tap(find.byTooltip('New folder'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Created while scrolled');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    final after = tester
        .state<ScrollableState>(find.byType(Scrollable))
        .position
        .pixels;
    expect(after, before);
    expect(tester.takeException(), isNull);
  });

  testWidgets('IT-008 overview retries a failed folder restart', (
    tester,
  ) async {
    final repository = _OverviewRestartRetryRepository();
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('New folder'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Created');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    expect(repository.folderCalls, 2);
    expect(find.text('Created'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

    await tester.tap(find.widgetWithText(TextButton, 'Retry'));
    await tester.pumpAndSettle();

    expect(repository.folderCalls, 3);
    expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
    expect(find.text('Created'), findsOneWidget);
  });

  testWidgets('IT-008 overview sort changes only the Unfiled resource', (
    tester,
  ) async {
    final repository = _OverviewSortRepository();
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    expect(
      tester.getTopLeft(find.text('newer')).dy,
      lessThan(tester.getTopLeft(find.text('older')).dy),
    );
    final folderTop = tester.getTopLeft(find.text('Ideas')).dy;

    await tester.tap(find.byType(DropdownButton<SavedPostSort>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Oldest').last);
    await tester.pumpAndSettle();

    expect(
      tester.getTopLeft(find.text('older')).dy,
      lessThan(tester.getTopLeft(find.text('newer')).dy),
    );
    expect(tester.getTopLeft(find.text('Ideas')).dy, folderTop);
    expect(repository.sortCalls, [SavedPostSort.newest, SavedPostSort.oldest]);
  });

  testWidgets('IT-008 Unfiled hides Retry for non-retryable failure', (
    tester,
  ) async {
    final repository = _OverviewIncrementalErrorRepository();
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Load more'));
    await tester.pumpAndSettle();

    expect(find.text('confirmed'), findsOneWidget);
    expect(find.text("Saved posts couldn't load."), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
  });

  testWidgets('IT-008 later folder pages remain above Unfiled', (
    tester,
  ) async {
    final repository = _OverviewPagedFoldersRepository();
    await _pump(tester, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Load more folders'));
    await tester.pumpAndSettle();

    expect(find.text('Ideas'), findsOneWidget);
    expect(find.text('Later'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Later')).dy,
      lessThan(tester.getTopLeft(find.text('Unfiled')).dy),
    );
    expect(repository.folderCursors, [null, 'next-folders']);
  });
}

Future<void> _pump(
  WidgetTester tester,
  SavedPostRepository repository,
) => tester.pumpWidget(
  ProviderScope(
    overrides: [
      authSessionProvider.overrideWith(SignedInAuthSession.new),
      accountSavedPostRepositoryProvider(
        AccountKey('did:plc:test'),
      ).overrideWith((ref) async => repository),
    ],
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: const SavedPostsPage(),
    ),
  ),
);

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

SavedPostItem _item(String rkey) => SavedPostItemMapper.fromMap({
  'post': {
    'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
    'cid': 'bafy$rkey',
    'rkey': rkey,
    'text': rkey,
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'quoteCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasReplied': false,
    'viewerHasSaved': true,
    'viewerSavedFolderId': null,
    'createdAt': '2026-07-21T10:00:00.000Z',
    'indexedAt': '2026-07-21T10:00:01.000Z',
    'author': {
      'did': 'did:plc:author',
      'handle': 'author.craftsky.social',
    },
  },
  'savedAt': '2026-07-21T12:00:00.000Z',
});

SavedPostItem _itemAt(String rkey, DateTime savedAt) =>
    SavedPostItemMapper.fromMap({
      'post': {
        'uri': 'at://did:plc:author/social.craftsky.feed.post/$rkey',
        'cid': 'bafy$rkey',
        'rkey': rkey,
        'text': rkey,
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 0,
        'quoteCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
        'viewerHasSaved': true,
        'viewerSavedFolderId': null,
        'createdAt': '2026-07-21T10:00:00.000Z',
        'indexedAt': '2026-07-21T10:00:01.000Z',
        'author': {
          'did': 'did:plc:author',
          'handle': 'author.craftsky.social',
        },
      },
      'savedAt': savedAt.toIso8601String(),
    });

final class _OverviewRepository implements SavedPostRepository {
  _OverviewRepository({required this.folders, required this.unfiled});

  final SavedPostFolderPage folders;
  final SavedPostPage unfiled;
  final List<String> createNames = [];
  SavedPostFolder? createdFolder;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      folders;

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async => unfiled;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      throw UnimplementedError();

  @override
  Future<void> unsave(Post post) => throw UnimplementedError();

  @override
  Future<SavedPostFolder> createFolder(String name) async {
    createNames.add(name);
    return createdFolder!;
  }

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _OverviewRefreshRepository implements SavedPostRepository {
  int folderCalls = 0;
  int unfiledCalls = 0;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    folderCalls++;
    return folderCalls == 1
        ? const SavedPostFolderPage(items: [])
        : SavedPostFolderPage(items: [_folder('folder-a', 'Ideas')]);
  }

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    unfiledCalls++;
    return unfiledCalls == 1
        ? const SavedPostPage(items: [])
        : SavedPostPage(items: [_item('refreshed-unfiled')]);
  }

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
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _OverviewRestartRetryRepository implements SavedPostRepository {
  int folderCalls = 0;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    folderCalls++;
    if (folderCalls == 2) throw StateError('restart failed');
    return folderCalls == 1
        ? SavedPostFolderPage(items: [_folder('folder-a', 'Ideas')])
        : SavedPostFolderPage(
            items: [
              _folder('created', 'Created'),
              _folder('folder-a', 'Ideas'),
            ],
          );
  }

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async => SavedPostPage(items: [_item('unfiled')]);

  @override
  Future<SavedPostFolder> createFolder(String name) async =>
      _folder('created', 'Created');

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      throw UnimplementedError();

  @override
  Future<void> unsave(Post post) => throw UnimplementedError();

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _OverviewSortRepository implements SavedPostRepository {
  final List<SavedPostSort> sortCalls = [];

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(items: [_folder('folder-a', 'Ideas')]);

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    sortCalls.add(sort);
    final older = _itemAt('older', DateTime.utc(2026, 7, 20));
    final newer = _itemAt('newer', DateTime.utc(2026, 7, 21));
    return SavedPostPage(
      items: sort == SavedPostSort.newest ? [newer, older] : [older, newer],
    );
  }

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
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _OverviewIncrementalErrorRepository implements SavedPostRepository {
  int listCalls = 0;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      const SavedPostFolderPage(items: []);

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    listCalls++;
    if (cursor != null) throw const ApiBadRequest('validation_failed');
    return SavedPostPage(items: [_item('confirmed')], cursor: 'next');
  }

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
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _OverviewPagedFoldersRepository implements SavedPostRepository {
  final List<String?> folderCursors = [];

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    folderCursors.add(cursor);
    return cursor == null
        ? SavedPostFolderPage(
            items: [_folder('folder-a', 'Ideas')],
            cursor: 'next-folders',
          )
        : SavedPostFolderPage(items: [_folder('folder-b', 'Later')]);
  }

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async => SavedPostPage(items: [_item('unfiled')]);

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
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}
