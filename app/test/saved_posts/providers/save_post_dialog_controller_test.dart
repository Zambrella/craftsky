import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/saved_posts/data/saved_post_repository.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/save_post_dialog_controller.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-013 manages selection and independent folder creation', () async {
    final account = AccountKey('did:plc:alice');
    final uri = AtUri.parse(
      'at://did:plc:author/social.craftsky.feed.post/3lsaved',
    );
    final repository = _DialogSavedPostRepository();
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);

    final newSaveKey = SavePostDialogKey(account: account, uri: uri);
    final moveKey = SavePostDialogKey(
      account: account,
      uri: uri,
      initialFolderId: 'folder-current',
    );
    final newSaveProvider = savePostDialogControllerProvider(newSaveKey);
    final moveProvider = savePostDialogControllerProvider(moveKey);
    final subscription = container.listen(
      newSaveProvider,
      (_, _) {},
      fireImmediately: true,
    );
    addTearDown(subscription.close);

    expect(container.read(newSaveProvider).selectedFolderId, isNull);
    expect(
      container.read(moveProvider).selectedFolderId,
      'folder-current',
    );
    expect(container.read(newSaveProvider).canConfirm, isTrue);
    expect(newSaveKey.toString(), isNot(contains(uri.toString())));
    expect(newSaveKey.toString(), isNot(contains('did:plc:alice')));

    container.read(newSaveProvider.notifier).selectFolder('folder-duplicate-a');
    expect(
      container.read(newSaveProvider).selectedFolderId,
      'folder-duplicate-a',
    );
    container.read(newSaveProvider.notifier).selectFolder('folder-duplicate-b');
    expect(
      container.read(newSaveProvider).selectedFolderId,
      'folder-duplicate-b',
    );
    container.read(newSaveProvider.notifier).selectFolder(null);
    expect(container.read(newSaveProvider).selectedFolderId, isNull);

    final notifier = container.read(newSaveProvider.notifier);
    final createCompleter = Completer<SavedPostFolder>();
    repository.createCompleter = createCompleter;
    final create =
        (notifier
              ..beginCreatingFolder()
              ..updateCreateName('  Ideas  '))
            .createFolder();
    expect(container.read(newSaveProvider).isCreatingFolder, isTrue);
    expect(container.read(newSaveProvider).isCreatePending, isTrue);
    await Future<void>.delayed(Duration.zero);
    expect(repository.createdNames, ['Ideas']);

    final created = SavedPostFolder(
      id: 'folder-created',
      name: 'Ideas',
      createdAt: DateTime.utc(2026, 7, 21, 12),
      updatedAt: DateTime.utc(2026, 7, 21, 12),
    );
    createCompleter.complete(created);
    expect(await create, same(created));
    expect(container.read(newSaveProvider).selectedFolderId, 'folder-created');
    expect(container.read(newSaveProvider).isCreatingFolder, isFalse);
    expect(container.read(newSaveProvider).createName, isEmpty);
    expect(container.read(newSaveProvider).createError, isNull);

    notifier
      ..beginCreatingFolder()
      ..updateCreateName('Retry me');
    repository.createCompleter = Completer<SavedPostFolder>();
    final failedCreate = notifier.createFolder();
    await Future<void>.delayed(Duration.zero);
    repository.createCompleter!.completeError(StateError('private raw error'));
    expect(await failedCreate, isNull);

    final failed = container.read(newSaveProvider);
    expect(failed.isCreatingFolder, isTrue);
    expect(failed.isCreatePending, isFalse);
    expect(failed.createName, 'Retry me');
    expect(failed.createError, SavePostDialogError.createFailed);
    expect(failed.toString(), isNot(contains('Retry me')));
    expect(failed.toString(), isNot(contains('private raw error')));

    notifier.cancel();
    expect(container.read(newSaveProvider).isCancelled, isTrue);
    expect(repository.saveCalls, 0);
    expect(repository.createdNames, ['Ideas', 'Retry me']);
  });

  test('IT-008 clears only a confirmed deleted folder selection', () async {
    final account = AccountKey('did:plc:alice');
    final uri = AtUri.parse(
      'at://did:plc:author/social.craftsky.feed.post/3lsaved',
    );
    final repository = _DialogSavedPostRepository()
      ..folders = [_folder('folder-current', 'Current')];
    final container = ProviderContainer.test(
      overrides: [
        accountSavedPostRepositoryProvider(
          account,
        ).overrideWith((ref) async => repository),
      ],
    );
    addTearDown(container.dispose);
    final provider = savePostDialogControllerProvider(
      SavePostDialogKey(
        account: account,
        uri: uri,
        initialFolderId: 'folder-current',
      ),
    );
    final subscription = container.listen(provider, (_, _) {});
    addTearDown(subscription.close);
    await container.read(savedPostFoldersProvider(account).future);

    expect(container.read(provider).selectedFolderId, 'folder-current');

    await container
        .read(savedPostFoldersProvider(account).notifier)
        .delete('folder-current', deleteSaves: false);

    expect(container.read(provider).selectedFolderId, isNull);
    expect(repository.deletedFolderIds, ['folder-current']);
  });
}

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

final class _DialogSavedPostRepository implements SavedPostRepository {
  Completer<SavedPostFolder>? createCompleter;
  final List<String> createdNames = [];
  int saveCalls = 0;
  List<SavedPostFolder> folders = [];
  final List<String> deletedFolderIds = [];

  @override
  Future<SavedPostFolder> createFolder(String name) {
    createdNames.add(name);
    return createCompleter!.future;
  }

  @override
  Future<SavedPostState> save(Post post, {required String? folderId}) {
    saveCalls++;
    throw UnimplementedError();
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
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit}) async =>
      SavedPostFolderPage(items: folders);

  @override
  Future<SavedPostFolder> renameFolder(String folderId, String name) =>
      throw UnimplementedError();

  @override
  Future<void> deleteFolder(
    String folderId, {
    required bool deleteSaves,
  }) async {
    deletedFolderIds.add(folderId);
    folders = folders.where((folder) => folder.id != folderId).toList();
  }
}
