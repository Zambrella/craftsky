import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/save_post_dialog.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('AT-002 pages distinct folders and confirms once', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderPages: {
        null: SavedPostFolderPage(
          items: [
            _folder('folder-1', 'Ideas'),
            _folder('folder-2', 'IDEAS'),
          ],
          cursor: 'opaque/private cursor',
        ),
        'opaque/private cursor': SavedPostFolderPage(
          items: [_folder('folder-3', 'Ideas')],
        ),
      },
    )..saveCompleter = Completer<SavedPostState>();
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    expect(
      tester
          .widget<RadioGroup<String?>>(find.byType(RadioGroup<String?>))
          .groupValue,
      isNull,
    );
    expect(find.text('Ideas'), findsOneWidget);
    expect(find.text('IDEAS'), findsOneWidget);
    expect(repository.saveCalls, 0);

    await tester.tap(find.widgetWithText(TextButton, 'Load more folders'));
    await tester.pumpAndSettle();
    expect(repository.folderCursors, [null, 'opaque/private cursor']);
    expect(find.text('Ideas'), findsNWidgets(2));

    await tester.tap(find.byKey(const ValueKey('saved-folder-folder-3')));
    await tester.tap(find.widgetWithText(FilledButton, 'Save post'));
    await tester.pump();
    await tester.tap(find.byType(FilledButton).last);
    await tester.pump();

    expect(repository.saveCalls, 1);
    expect(repository.lastSavedFolderId, 'folder-3');
    expect(find.byType(SavePostDialog), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    repository.saveCompleter!.complete(
      SavedPostState(
        savedAt: DateTime.utc(2026, 7, 21, 15),
        folderId: 'folder-3',
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(SavePostDialog), findsNothing);
    expect(find.byType(SnackBar), findsNothing);
  });

  testWidgets('AT-002 folder failure leaves No folder savable', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderError: Exception('private folder failure'),
    );
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    expect(find.text("Folders couldn't load."), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    expect(find.text('No folder'), findsOneWidget);
    expect(find.textContaining('private folder failure'), findsNothing);
    expect(find.byType(SearchBar), findsNothing);

    await tester.tap(find.widgetWithText(FilledButton, 'Save post'));
    await tester.pumpAndSettle();
    expect(repository.saveCalls, 1);
    expect(repository.lastSavedFolderId, isNull);
  });

  testWidgets('AT-002 creates a folder independently and keeps it selected', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderPages: {null: const SavedPostFolderPage(items: [])},
      createdFolder: _folder('created-id', 'Fresh ideas'),
    );
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'New folder'));
    await tester.pump();
    await tester.enterText(find.byType(TextField), '  Fresh ideas  ');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    expect(repository.createNames, ['Fresh ideas']);
    expect(find.byType(TextField), findsNothing);
    expect(find.text('Fresh ideas'), findsOneWidget);
    expect(
      tester
          .widget<RadioGroup<String?>>(find.byType(RadioGroup<String?>))
          .groupValue,
      'created-id',
    );

    await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
    await tester.pumpAndSettle();
    expect(repository.saveCalls, 0);
    expect(repository.createNames, hasLength(1));
  });

  testWidgets('AT-002 create failure stays editable with safe error', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderPages: {null: const SavedPostFolderPage(items: [])},
      createError: Exception('folder-name-private-sentinel'),
    );
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'New folder'));
    await tester.pump();
    await tester.enterText(find.byType(TextField), 'Ideas');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    expect(
      find.text("That folder couldn't be created. Try again."),
      findsOneWidget,
    );
    expect(find.textContaining('folder-name-private-sentinel'), findsNothing);
    expect(find.byType(SavePostDialog), findsOneWidget);
  });

  testWidgets('IT-008 retries a failed folder mutation restart from page one', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _RestartRetryRepository();
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'New folder'));
    await tester.pump();
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

  testWidgets('UT-010 canceled folder creation stays silent and editable', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderPages: {null: const SavedPostFolderPage(items: [])},
      createError: const ApiCanceled(),
    );
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'New folder'));
    await tester.pump();
    await tester.enterText(find.byType(TextField), 'Ideas');
    await tester.tap(find.widgetWithText(FilledButton, 'Create folder'));
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    expect(tester.widget<TextField>(find.byType(TextField)).enabled, isTrue);
    expect(
      find.text("That folder couldn't be created. Try again."),
      findsNothing,
    );
    expect(find.byType(SavePostDialog), findsOneWidget);
  });

  testWidgets('UT-010 hides Retry for a non-retryable folder failure', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final repository = _DialogSavedPostRepository(
      folderError: const ApiBadRequest('validation_failed'),
    );
    await _pumpDialog(tester, account, repository);
    await tester.pumpAndSettle();

    expect(find.text("Folders couldn't load."), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
    expect(find.text('No folder'), findsOneWidget);
  });
}

Future<void> _pumpDialog(
  WidgetTester tester,
  AccountKey account,
  SavedPostRepository repository,
) => tester
    .pumpWidget(
      ProviderScope(
        overrides: [
          accountSavedPostRepositoryProvider(
            account,
          ).overrideWith((ref) async => repository),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Builder(
              builder: (context) => FilledButton(
                onPressed: () => showDialog<void>(
                  context: context,
                  builder: (_) =>
                      SavePostDialog(account: account, post: _post()),
                ),
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      ),
    )
    .then((_) async {
      await tester.tap(find.text('Open'));
      await tester.pump();
    });

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

Post _post() => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/dialog',
  'cid': 'bafydialog',
  'rkey': 'dialog',
  'text': 'Save this post.',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'quoteCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasReplied': false,
  'viewerHasSaved': false,
  'viewerSavedFolderId': null,
  'createdAt': '2026-07-21T10:00:00.000Z',
  'indexedAt': '2026-07-21T10:00:01.000Z',
  'author': {
    'did': 'did:plc:author',
    'handle': 'author.craftsky.social',
  },
});

final class _DialogSavedPostRepository implements SavedPostRepository {
  _DialogSavedPostRepository({
    this.folderPages = const {},
    this.folderError,
    this.createdFolder,
    this.createError,
  });

  final Map<String?, SavedPostFolderPage> folderPages;
  final Exception? folderError;
  final SavedPostFolder? createdFolder;
  final Exception? createError;
  final List<String?> folderCursors = [];
  final List<String> createNames = [];
  Completer<SavedPostState>? saveCompleter;
  int saveCalls = 0;
  String? lastSavedFolderId;

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) {
    saveCalls++;
    lastSavedFolderId = folderId;
    return saveCompleter?.future ??
        Future.value(
          SavedPostState(
            savedAt: DateTime.utc(2026, 7, 21),
            folderId: folderId,
          ),
        );
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    folderCursors.add(cursor);
    if (folderError case final error?) throw error;
    return folderPages[cursor] ?? const SavedPostFolderPage(items: []);
  }

  @override
  Future<SavedPostFolder> createFolder(String name) async {
    createNames.add(name);
    if (createError case final error?) throw error;
    return createdFolder!;
  }

  @override
  Future<void> unsave(Post post) => throw UnimplementedError();

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) => throw UnimplementedError();

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}

final class _RestartRetryRepository implements SavedPostRepository {
  int folderCalls = 0;

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    folderCalls++;
    if (folderCalls == 2) throw StateError('restart failed');
    return folderCalls == 1
        ? const SavedPostFolderPage(items: [])
        : SavedPostFolderPage(items: [_folder('created', 'Created')]);
  }

  @override
  Future<SavedPostFolder> createFolder(String name) async =>
      _folder('created', 'Created');

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) async =>
      SavedPostState(
        savedAt: DateTime.utc(2026, 7, 21),
        folderId: folderId,
      );

  @override
  Future<void> unsave(Post post) => throw UnimplementedError();

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) => throw UnimplementedError();

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) => throw UnimplementedError();
}
