import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/models/saved_posts_overview.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-006 projects folders and Unfiled using saved chronology', () {
    final folders = [_folder('b', 'B'), _folder('a', 'A')];
    final newest = SavedPostsOverview.project(
      folders: folders,
      items: [
        _item('created-newer', createdHour: 14, savedHour: 10),
        _item('saved-newer', createdHour: 9, savedHour: 13),
        _item(
          'foldered',
          createdHour: 15,
          savedHour: 15,
          folderId: 'a',
        ),
      ],
      sort: SavedPostSort.newest,
    );

    expect(newest.folders, folders);
    expect(newest.folders.map((folder) => folder.id), ['b', 'a']);
    expect(newest.unfiledItems.map((item) => item.post.rkey), [
      'saved-newer',
      'created-newer',
    ]);
    expect(newest.showUnfiled, isTrue);
    expect(newest.isEmpty, isFalse);

    final oldest = SavedPostsOverview.project(
      folders: folders,
      items: newest.unfiledItems,
      sort: SavedPostSort.oldest,
    );
    expect(oldest.unfiledItems.map((item) => item.post.rkey), [
      'created-newer',
      'saved-newer',
    ]);
    expect(oldest.folders.map((folder) => folder.id), ['b', 'a']);

    final foldersOnly = SavedPostsOverview.project(
      folders: folders,
      items: const [],
      sort: SavedPostSort.newest,
    );
    expect(foldersOnly.showUnfiled, isFalse);
    expect(foldersOnly.isEmpty, isFalse);

    final empty = SavedPostsOverview.project(
      folders: const [],
      items: const [],
      sort: SavedPostSort.newest,
    );
    expect(empty.showUnfiled, isFalse);
    expect(empty.isEmpty, isTrue);
  });
}

SavedPostFolder _folder(String id, String name) => SavedPostFolder(
  id: id,
  name: name,
  createdAt: DateTime.utc(2026, 7, 21),
  updatedAt: DateTime.utc(2026, 7, 21),
);

SavedPostItem _item(
  String rkey, {
  required int createdHour,
  required int savedHour,
  String? folderId,
}) => SavedPostItemMapper.fromMap({
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
    'viewerSavedFolderId': folderId,
    'createdAt': DateTime.utc(2026, 7, 21, createdHour).toIso8601String(),
    'indexedAt': DateTime.utc(2026, 7, 21, createdHour).toIso8601String(),
    'author': {
      'did': 'did:plc:author',
      'handle': 'author.craftsky.social',
    },
  },
  'savedAt': DateTime.utc(2026, 7, 21, savedHour).toIso8601String(),
  'folderId': folderId,
});
