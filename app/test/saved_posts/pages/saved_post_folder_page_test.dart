import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/pages/saved_post_folder_page.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/saved_posts/widgets/save_post_dialog.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_row_actions.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('AT-005 folder page pages independently and changes sort', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('new-1')], cursor: 'next-new'),
      )
      ..enqueue(
        SavedPostSort.newest,
        'next-new',
        SavedPostPage(items: [_item('new-2')]),
      )
      ..enqueue(
        SavedPostSort.oldest,
        null,
        SavedPostPage(items: [_item('old-1')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Ideas'), findsOneWidget);
    expect(find.text('new-1'), findsOneWidget);
    await tester.tap(find.widgetWithText(TextButton, 'Load more'));
    await tester.pumpAndSettle();
    expect(find.text('new-2'), findsOneWidget);

    await tester.tap(find.byType(DropdownButton<SavedPostSort>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Oldest').last);
    await tester.pumpAndSettle();
    expect(find.text('old-1'), findsOneWidget);
    expect(repository.calls, [
      (SavedPostSort.newest, null),
      (SavedPostSort.newest, 'next-new'),
      (SavedPostSort.oldest, null),
    ]);
  });

  testWidgets('AT-007 renames and explicitly deletes a folder', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..renamedFolder = _folder('folder-a', 'Renamed ideas')
      ..enqueue(
        SavedPostSort.newest,
        null,
        const SavedPostPage(items: []),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Folder actions'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Rename folder'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), ' Renamed ideas ');
    await tester.tap(find.widgetWithText(FilledButton, 'Rename folder'));
    await tester.pumpAndSettle();
    expect(find.text('Renamed ideas'), findsOneWidget);

    await tester.tap(find.byTooltip('Folder actions'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete folder'));
    await tester.pumpAndSettle();
    expect(find.text('Keep saved posts'), findsOneWidget);
    expect(find.text('Delete saved posts'), findsOneWidget);
    expect(
      tester
          .widget<TextButton>(find.widgetWithText(TextButton, 'Cancel'))
          .autofocus,
      isTrue,
    );
    expect(
      tester
          .getSemantics(
            find.bySemanticsLabel('Delete saved posts'),
          )
          .hint,
      'Destructive action',
    );
    await tester.tap(find.text('Keep saved posts'));
    await tester.pumpAndSettle();
    expect(repository.deleted, [('folder-a', false)]);
  });

  testWidgets('IT-008 reconciles a confirmed move into loaded Unfiled', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final item = _item('move-to-unfiled');
    final repository = _MoveReconciliationRepository(item);

    await tester.pumpWidget(
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
          home: _MoveReconciliationHarness(account: account, item: item),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.byKey(const ValueKey('source-move-to-unfiled')),
      findsOneWidget,
    );
    expect(
      find.byKey(const ValueKey('destination-move-to-unfiled')),
      findsNothing,
    );

    await tester.tap(find.text('Move item'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('No folder'));
    await tester.tap(find.widgetWithText(FilledButton, 'Move'));
    await tester.pumpAndSettle();

    expect(repository.savedFolderIds, [null]);
    expect(find.byKey(const ValueKey('source-move-to-unfiled')), findsNothing);
    expect(
      find.byKey(const ValueKey('destination-move-to-unfiled')),
      findsOneWidget,
    );
    expect(repository.listCalls, hasLength(2));
  });

  testWidgets(
    'IT-008 reconciles a cross-sort move into the loaded destination',
    (tester) async {
      final account = AccountKey('did:plc:alice');
      final item = _item('cross-sort-move');
      final confirmedSavedAt = DateTime.utc(2026, 7, 21, 14);
      final repository = _MoveReconciliationRepository(
        item,
        confirmedSavedAt: confirmedSavedAt,
      );

      await tester.pumpWidget(
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
            home: _MoveReconciliationHarness(
              account: account,
              item: item,
              sourceSort: SavedPostSort.oldest,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Move item'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('No folder'));
      await tester.tap(find.widgetWithText(FilledButton, 'Move'));
      await tester.pumpAndSettle();

      expect(
        find.byKey(const ValueKey('source-cross-sort-move')),
        findsNothing,
      );
      expect(
        find.byKey(const ValueKey('destination-cross-sort-move')),
        findsOneWidget,
      );
      expect(find.text(confirmedSavedAt.toIso8601String()), findsOneWidget);
      expect(repository.listCalls, [
        (SavedPostScopeKind.folder, SavedPostSort.oldest),
        (SavedPostScopeKind.unfiled, SavedPostSort.newest),
      ]);
    },
  );

  testWidgets('UT-010 canceled rename stays silent and editable', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..renameError = const ApiCanceled()
      ..enqueue(
        SavedPostSort.newest,
        null,
        const SavedPostPage(items: []),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Folder actions'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Rename folder'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Renamed ideas');
    await tester.tap(find.widgetWithText(FilledButton, 'Rename folder'));
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    expect(tester.widget<TextField>(find.byType(TextField)).enabled, isTrue);
    expect(
      find.text("That folder couldn't be created. Try again."),
      findsNothing,
    );
  });

  testWidgets('IT-008 shows an empty state for an empty folder', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        const SavedPostPage(items: []),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Nothing saved yet'), findsOneWidget);
    expect(find.text('Ideas'), findsOneWidget);
  });

  testWidgets('IT-008 remove-mode folder deletion sends one scoped command', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('remove-with-folder')]),
      )
      ..enqueue(
        SavedPostSort.newest,
        null,
        const SavedPostPage(items: []),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Folder actions'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete folder'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete saved posts'));
    await tester.pumpAndSettle();

    expect(repository.deleted, [('folder-a', true)]);
    expect(find.text('remove-with-folder'), findsNothing);
  });

  testWidgets('IT-008 retains rows while incremental Retry recovers', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('first')], cursor: 'next'),
      )
      ..enqueue(
        SavedPostSort.newest,
        'next',
        StateError('incremental failure'),
      )
      ..enqueue(
        SavedPostSort.newest,
        'next',
        SavedPostPage(items: [_item('second')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Load more'));
    await tester.pumpAndSettle();

    expect(find.text('first'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

    await tester.tap(find.widgetWithText(TextButton, 'Retry'));
    await tester.pumpAndSettle();

    expect(find.text('first'), findsOneWidget);
    expect(find.text('second'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
    expect(repository.calls, [
      (SavedPostSort.newest, null),
      (SavedPostSort.newest, 'next'),
      (SavedPostSort.newest, 'next'),
    ]);
  });

  testWidgets('IT-008 invalid cursor restarts only the visible folder list', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('stale')], cursor: 'invalid-next'),
      )
      ..enqueue(
        SavedPostSort.newest,
        'invalid-next',
        const ApiBadRequest('invalid_cursor'),
      )
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('restarted')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Load more'));
    await tester.pumpAndSettle();

    expect(find.text('stale'), findsNothing);
    expect(find.text('restarted'), findsOneWidget);
    expect(repository.calls, [
      (SavedPostSort.newest, null),
      (SavedPostSort.newest, 'invalid-next'),
      (SavedPostSort.newest, null),
    ]);
  });

  testWidgets('IT-008 pull-to-refresh replaces stale folder rows', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('stale')], cursor: 'stale-cursor'),
      )
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('fresh')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.fling(
      find.byType(ListView),
      const Offset(0, 320),
      1000,
    );
    await tester.pumpAndSettle();

    expect(find.text('stale'), findsNothing);
    expect(find.text('fresh'), findsOneWidget);
    expect(repository.calls, [
      (SavedPostSort.newest, null),
      (SavedPostSort.newest, null),
    ]);
  });

  testWidgets('IT-008 initial folder failure retries into confirmed content', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        StateError('initial failure'),
      )
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('recovered')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
    expect(find.textContaining('initial failure'), findsNothing);

    await tester.tap(find.widgetWithText(TextButton, 'Retry'));
    await tester.pumpAndSettle();

    expect(find.text('recovered'), findsOneWidget);
    expect(repository.calls, [
      (SavedPostSort.newest, null),
      (SavedPostSort.newest, null),
    ]);
  });

  testWidgets('IT-008 unsaves a visible row without confirmation', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('unsave-row')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Unsave'));
    await tester.pumpAndSettle();

    expect(repository.unsaveCalls, 1);
    expect(find.text('unsave-row'), findsNothing);
    expect(find.byType(AlertDialog), findsNothing);
  });

  testWidgets('IT-008 canceled row unsave rolls back silently', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..unsaveError = const ApiCanceled()
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('canceled-unsave')]),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Unsave'));
    await tester.pumpAndSettle();

    expect(repository.unsaveCalls, 1);
    expect(find.text('canceled-unsave'), findsOneWidget);
    expect(find.byType(SnackBar), findsNothing);
  });

  testWidgets('IT-008 hides Retry for non-retryable incremental failure', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final folder = _folder('folder-a', 'Ideas');
    final repository = _FolderPageRepository()
      ..visibleFolder = folder
      ..enqueue(
        SavedPostSort.newest,
        null,
        SavedPostPage(items: [_item('confirmed')], cursor: 'next'),
      )
      ..enqueue(
        SavedPostSort.newest,
        'next',
        const ApiBadRequest('validation_failed'),
      );
    await tester.pumpWidget(
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
          home: SavedPostFolderScreen(account: account, folder: folder),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Load more'));
    await tester.pumpAndSettle();

    expect(find.text('confirmed'), findsOneWidget);
    expect(find.text("Saved posts couldn't load."), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
  });

  testWidgets('IT-008 failed move keeps source and attempted selection', (
    tester,
  ) async {
    final account = AccountKey('did:plc:alice');
    final item = _item('failed-move');
    final repository = _MoveReconciliationRepository(item)
      ..saveError = StateError('move failed');

    await tester.pumpWidget(
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
          home: _MoveReconciliationHarness(account: account, item: item),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Move item'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('No folder'));
    await tester.tap(find.widgetWithText(FilledButton, 'Move'));
    await tester.pumpAndSettle();

    expect(find.byType(SavePostDialog), findsOneWidget);
    expect(
      tester
          .widget<RadioGroup<String?>>(find.byType(RadioGroup<String?>))
          .groupValue,
      isNull,
    );
    expect(
      find.text("That change couldn't be saved. Try again."),
      findsOneWidget,
    );
    expect(find.byKey(const ValueKey('source-failed-move')), findsOneWidget);
    expect(find.byKey(const ValueKey('destination-failed-move')), findsNothing);
  });
}

class _MoveReconciliationHarness extends ConsumerWidget {
  const _MoveReconciliationHarness({
    required this.account,
    required this.item,
    this.sourceSort = SavedPostSort.newest,
  });

  final AccountKey account;
  final SavedPostItem item;
  final SavedPostSort sourceSort;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final sourceKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.folder('folder-a'),
      sort: sourceSort,
    );
    final destinationKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.newest,
    );
    final source = ref.watch(savedPostsProvider(sourceKey)).value;
    final destination = ref.watch(savedPostsProvider(destinationKey)).value;

    return Scaffold(
      body: Column(
        children: [
          for (final sourceItem in source?.items ?? const <SavedPostItem>[])
            Text(
              sourceItem.post.rkey,
              key: ValueKey('source-${sourceItem.post.rkey}'),
            ),
          for (final destinationItem
              in destination?.items ?? const <SavedPostItem>[])
            Column(
              children: [
                Text(
                  destinationItem.post.rkey,
                  key: ValueKey('destination-${destinationItem.post.rkey}'),
                ),
                Text(destinationItem.savedAt.toIso8601String()),
              ],
            ),
          FilledButton(
            onPressed: () => moveSavedPost(
              context,
              ref,
              account: account,
              item: item,
              sourceKey: sourceKey,
            ),
            child: const Text('Move item'),
          ),
        ],
      ),
    );
  }
}

final class _MoveReconciliationRepository implements SavedPostRepository {
  _MoveReconciliationRepository(this.item, {DateTime? confirmedSavedAt})
    : confirmedSavedAt = confirmedSavedAt ?? item.savedAt;

  final SavedPostItem item;
  final DateTime confirmedSavedAt;
  final List<(SavedPostScopeKind, SavedPostSort)> listCalls = [];
  final List<String?> savedFolderIds = [];
  Object? saveError;

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    listCalls.add((scope.kind, sort));
    return scope.kind == SavedPostScopeKind.folder
        ? SavedPostPage(items: [item])
        : const SavedPostPage(items: []);
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(items: [_folder('folder-a', 'Ideas')]);

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) async {
    savedFolderIds.add(folderId);
    if (saveError case final error?) _throwTestError(error);
    return SavedPostState(savedAt: confirmedSavedAt, folderId: folderId);
  }

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

final class _FolderPageRepository implements SavedPostRepository {
  final Map<String, List<Object>> responses = {};
  final List<(SavedPostSort, String?)> calls = [];
  final List<(String, bool)> deleted = [];
  SavedPostFolder? visibleFolder;
  SavedPostFolder? renamedFolder;
  Object? renameError;
  Object? unsaveError;
  int unsaveCalls = 0;

  void enqueue(SavedPostSort sort, String? cursor, Object response) => responses
      .putIfAbsent('${sort.name}:${cursor ?? 'first'}', () => [])
      .add(response);

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    calls.add((sort, cursor));
    final response = responses['${sort.name}:${cursor ?? 'first'}']!.removeAt(
      0,
    );
    if (response is SavedPostPage) return response;
    _throwTestError(response);
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(
        items: [?visibleFolder],
      );
  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) =>
      throw UnimplementedError();
  @override
  Future<void> unsave(Post post) async {
    unsaveCalls++;
    if (unsaveError case final error?) _throwTestError(error);
  }

  @override
  Future<SavedPostFolder> createFolder(String name) =>
      throw UnimplementedError();
  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) async {
    if (renameError case final error?) _throwTestError(error);
    visibleFolder = renamedFolder;
    return renamedFolder!;
  }

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) async {
    deleted.add((folderId, deleteSaves));
    visibleFolder = null;
  }
}

Never _throwTestError(Object error) {
  if (error is Exception) throw error;
  if (error is Error) throw error;
  throw StateError(error.toString());
}

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
    'viewerSavedFolderId': 'folder-a',
    'createdAt': '2026-07-21T10:00:00.000Z',
    'indexedAt': '2026-07-21T10:00:01.000Z',
    'author': {'did': 'did:plc:author', 'handle': 'author.craftsky.social'},
  },
  'savedAt': '2026-07-21T12:00:00.000Z',
  'folderId': 'folder-a',
});
