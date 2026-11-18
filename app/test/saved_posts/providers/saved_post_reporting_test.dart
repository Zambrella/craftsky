import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/account_saved_post_state_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/session_registry_storage.dart';

enum _Action {
  save,
  move,
  unsave,
  postsMore,
  postsRefresh,
  foldersMore,
  foldersRefresh,
  create,
  rename,
  delete,
}

void main() {
  setUpAll(initializeMappers);

  for (final action in _Action.values) {
    test('${action.name} owns its consumed failure once', () async {
      final harness = _Harness();
      await harness.prepare();
      final cause = StateError('PRIVATE_SAVED_POST_PAYLOAD');
      final stack = StackTrace.fromString(
        '#0 savedAction (package:craftsky_app/saved_posts/action.dart:42:7)',
      );
      harness.repository
        ..failure = cause
        ..failureStack = stack;

      await harness.run(action);

      expect(harness.reporter.errors, [same(cause)]);
      expect(harness.reporter.stacks.single.toString(), stack.toString());
      final context = harness.reporter.contexts.single;
      expect(context.feature, 'SavedPosts');
      expect(
        context.workflow,
        isA<PrivateOperationalFailureContext>().having(
          (value) => value.accountDid,
          'initiating account',
          harness.account.did.value,
        ),
      );
    });
  }

  for (final failure in [
    const ApiNetworkError('PRIVATE_NETWORK'),
    const ApiCanceled(),
    const ApiBadRequest('validation_failed'),
  ]) {
    test(
      '${failure.runtimeType} remains local during save and pagination',
      () async {
        final harness = _Harness();
        await harness.prepare();
        harness.repository.failure = failure;
        await harness.run(_Action.save);
        await harness.run(_Action.postsMore);
        await harness.run(_Action.create);
        expect(harness.reporter.errors, isEmpty);
      },
    );
  }

  test('folder name validation remains a correctable local failure', () async {
    final harness = _Harness();
    await harness.prepare();
    harness.repository.failure = const SavedPostFolderNameException(
      SavedPostFolderNameError.empty,
    );
    await harness.run(_Action.create);
    expect(harness.reporter.errors, isEmpty);
  });

  test('retired mutations do not report or relabel late failures', () async {
    final harness = _Harness();
    await harness.prepare();
    final pending = Completer<SavedPostState>();
    harness.repository.pendingSave = pending.future;
    final mutation = harness.run(_Action.save);
    await harness.container.pump();
    harness.container.read(savedPostAccountBoundaryProvider.notifier).advance();
    await harness.container.pump();
    pending.completeError(StateError('PRIVATE_OLD_ACCOUNT'));
    await mutation;
    expect(harness.reporter.errors, isEmpty);
  });

  test(
    'failed repository acquisition keeps its existing provider owner',
    () async {
      final harness = _Harness();
      await harness.container.read(sessionRegistryProvider.future);
      final cause = StateError('PRIVATE_ACQUISITION');
      harness.container.updateOverrides([
        secureSessionRegistryStorageProvider.overrideWithValue(
          InMemorySessionRegistryStorage(SessionRegistry.empty()),
        ),
        errorReporterProvider.overrideWithValue(harness.reporter),
        accountSavedPostRepositoryProvider(harness.account).overrideWith(
          (ref) async => throw cause,
        ),
      ]);
      await harness.run(_Action.save);
      expect(harness.reporter.errors, [same(cause)]);
    },
  );
}

final class _Harness {
  new() {
    container = ProviderContainer.test(
      retry: (_, _) => null,
      observers: [ProviderLogger(reporter: reporter)],
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          InMemorySessionRegistryStorage(SessionRegistry.empty()),
        ),
        errorReporterProvider.overrideWithValue(reporter),
        accountSavedPostRepositoryProvider(account).overrideWith(
          (ref) async => repository,
        ),
      ],
    );
  }

  final account = AccountKey('did:plc:alice');
  final reporter = _Reporter();
  final repository = _Repository();
  late final ProviderContainer container;
  final _subscriptions = <ProviderSubscription<Object?>>[];

  SavedPostListKey get listKey => SavedPostListKey(
    account: account,
    scope: const SavedPostScope.all(),
    sort: SavedPostSort.newest,
  );

  Future<void> prepare() async {
    await container.read(sessionRegistryProvider.future);
    _subscriptions
      ..add(container.listen(savedPostsProvider(listKey), (_, _) {}))
      ..add(container.listen(savedPostFoldersProvider(account), (_, _) {}));
    addTearDown(() {
      for (final subscription in _subscriptions) {
        subscription.close();
      }
    });
    await Future.wait([
      container.read(savedPostsProvider(listKey).future),
      container.read(savedPostFoldersProvider(account).future),
    ]);
  }

  Future<void> run(_Action action) async {
    final saved = container.read(
      accountSavedPostStateProvider(account).notifier,
    );
    switch (action) {
      case _Action.save:
        await saved.save(_post, null);
      case _Action.move:
        await saved.move(_item, 'PRIVATE_FOLDER');
      case _Action.unsave:
        await saved.unsave(_post);
      case _Action.postsMore:
        await container.read(savedPostsProvider(listKey).notifier).loadMore();
      case _Action.postsRefresh:
        await container.read(savedPostsProvider(listKey).notifier).refresh();
      case _Action.foldersMore:
        await container
            .read(savedPostFoldersProvider(account).notifier)
            .loadMore();
      case _Action.foldersRefresh:
        await container
            .read(savedPostFoldersProvider(account).notifier)
            .refresh();
      case _Action.create:
        await container
            .read(savedPostFoldersProvider(account).notifier)
            .create('PRIVATE_FOLDER_NAME');
      case _Action.rename:
        await container
            .read(savedPostFoldersProvider(account).notifier)
            .rename('PRIVATE_FOLDER', 'PRIVATE_FOLDER_NAME');
      case _Action.delete:
        await container
            .read(savedPostFoldersProvider(account).notifier)
            .delete('PRIVATE_FOLDER', deleteSaves: false);
    }
  }
}

Post get _post => PostMapper.fromMap({
  'uri': 'at://did:plc:author/social.craftsky.feed.post/first',
  'cid': 'post-cid',
  'rkey': 'first',
  'text': 'PRIVATE_POST',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'quoteCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasReplied': false,
  'viewerHasSaved': true,
  'sponsored': false,
  'createdAt': '2026-07-21T10:00:00.000Z',
  'indexedAt': '2026-07-21T10:00:01.000Z',
  'author': {'did': 'did:plc:author', 'handle': 'author.test'},
});

SavedPostItem get _item => SavedPostItem(
  post: _post,
  savedAt: DateTime.utc(2026),
);

SavedPostFolder get _folder => SavedPostFolder(
  id: 'PRIVATE_FOLDER',
  name: 'PRIVATE_FOLDER_NAME',
  createdAt: DateTime.utc(2026),
  updatedAt: DateTime.utc(2026),
);

final class _Repository implements SavedPostRepository {
  Object? failure;
  StackTrace failureStack = StackTrace.empty;
  Future<SavedPostState>? pendingSave;

  void _check() {
    if (failure case final error?) {
      Error.throwWithStackTrace(error, failureStack);
    }
  }

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) async {
    _check();
    return await pendingSave ?? SavedPostState(savedAt: DateTime.utc(2026));
  }

  @override
  Future<void> unsave(Post post) async => _check();

  @override
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  }) async {
    _check();
    return SavedPostPage(items: [_item], cursor: 'PRIVATE_CURSOR');
  }

  @override
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async {
    _check();
    return SavedPostFolderPage(items: [_folder], cursor: 'PRIVATE_CURSOR');
  }

  @override
  Future<SavedPostFolder> createFolder(String name) async {
    _check();
    return _folder;
  }

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) async {
    _check();
    return _folder;
  }

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) async => _check();
}

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace>[];
  final contexts = <ReportContext>[];

  @override
  bool get enabled => true;

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace!);
    contexts.add(context);
    return null;
  }
}
