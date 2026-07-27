import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_keys.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-005 merges opaque pages and reconciles folder mutations', () {
    final account = AccountKey('did:plc:alice');
    final listKey = SavedPostListKey(
      account: account,
      scope: const SavedPostScope.unfiled(),
      sort: SavedPostSort.newest,
    );
    expect(listKey.toString(), isNot(contains('did:plc:alice')));

    final firstSavedPage = SavedPostPage(
      items: [_savedItem('a'), _savedItem('b')],
      cursor: 'opaque saved cursor /?=private',
    );
    final savedState = SavedPostListState.fromPage(firstSavedPage).appendPage(
      SavedPostPage(items: [_savedItem('b'), _savedItem('c')]),
    );
    expect(savedState.items.map((item) => item.post.rkey), ['a', 'b', 'c']);
    expect(savedState.cursor, isNull);
    final savedFailure = savedState.withIncrementalError(
      StateError('private cursor failure'),
    );
    expect(savedFailure.items, savedState.items);
    expect(savedFailure.incrementalError, isNotNull);
    expect(
      isInvalidSavedPostCursorError(const ApiBadRequest('invalid_cursor')),
      isTrue,
    );
    expect(
      isInvalidSavedPostCursorError(const ApiBadRequest('validation_failed')),
      isFalse,
    );

    final firstFolderPage = SavedPostFolderPage(
      items: [_folder('b', 'B'), _folder('d', 'D')],
      cursor: 'opaque folder cursor',
    );
    final folderState = SavedPostFolderListState.fromPage(firstFolderPage)
        .appendPage(
          SavedPostFolderPage(
            items: [_folder('d', 'duplicate'), _folder('f', 'F')],
          ),
        );
    expect(folderState.items.map((folder) => folder.id), ['b', 'd', 'f']);

    final created = _folder('z-created', 'A name sorted off page');
    final awaitingRestart = folderState.afterConfirmedMutation(
      retain: created,
    );
    expect(awaitingRestart.cursor, isNull);
    expect(awaitingRestart.folderById(created.id), same(created));
    expect(awaitingRestart.items.map((folder) => folder.id), ['b', 'd', 'f']);

    final renamed = _folder('d', 'Renamed');
    final restarted = awaitingRestart.restartPage(
      SavedPostFolderPage(
        items: [_folder('a', 'A'), renamed],
        cursor: 'new opaque cursor',
      ),
    );
    expect(restarted.items.map((folder) => folder.id), ['a', 'd']);
    expect(restarted.folderById('d')?.name, 'Renamed');
    expect(restarted.folderById(created.id), same(created));
    expect(restarted.cursor, 'new opaque cursor');

    final deleted = restarted.removeFolder('d');
    expect(deleted.folderById('d'), isNull);
    expect(deleted.items.map((folder) => folder.id), ['a']);
    expect(deleted.folderById(created.id), same(created));
  });
}

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

SavedPostItem _savedItem(String rkey) => SavedPostItemMapper.fromMap({
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
  'folderId': null,
});
