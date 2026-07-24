import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-002 decodes typed saved wire models', () {
    final post = <String, dynamic>{
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lsaved',
      'cid': 'bafysaved',
      'rkey': '3lsaved',
      'text': 'A post worth returning to.',
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'quoteCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasReplied': false,
      'viewerHasSaved': true,
      'viewerSavedFolderId': '018f-folder-opaque',
      'createdAt': '2026-07-21T10:00:00.000Z',
      'indexedAt': '2026-07-21T10:00:01.000Z',
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
      },
    };
    final stateJson = <String, dynamic>{
      'savedAt': '2026-07-21T11:00:00.000Z',
      'folderId': '018f-folder-opaque',
    };
    final folderJson = <String, dynamic>{
      'id': '018f-folder-opaque',
      'name': 'Ideas',
      'createdAt': '2026-07-21T09:00:00.000Z',
      'updatedAt': '2026-07-21T09:30:00.000Z',
    };

    final state = SavedPostStateMapper.fromMap(stateJson);
    final unfiled = SavedPostStateMapper.fromMap({
      ...stateJson,
      'folderId': null,
    });
    final item = SavedPostItemMapper.fromMap({
      'post': post,
      ...stateJson,
    });
    final page = SavedPostPageMapper.fromMap({
      'items': [
        {'post': post, ...stateJson},
      ],
      'cursor': 'opaque-post-cursor',
    });
    final folder = SavedPostFolderMapper.fromMap(folderJson);
    final folderPage = SavedPostFolderPageMapper.fromMap({
      'items': [folderJson],
      'cursor': 'opaque-folder-cursor',
    });

    expect(state.savedAt, DateTime.parse(stateJson['savedAt']! as String));
    expect(state.folderId, '018f-folder-opaque');
    expect(unfiled.folderId, isNull);
    expect(item.post.uri.toString(), post['uri']);
    expect(item.savedAt, state.savedAt);
    expect(item.folderId, state.folderId);
    expect(page.items.single.post.uri, item.post.uri);
    expect(page.cursor, 'opaque-post-cursor');
    expect(folder.id, '018f-folder-opaque');
    expect(folder.name, 'Ideas');
    expect(
      folder.createdAt,
      DateTime.parse(folderJson['createdAt']! as String),
    );
    expect(
      folder.updatedAt,
      DateTime.parse(folderJson['updatedAt']! as String),
    );
    expect(folderPage.items.single.id, folder.id);
    expect(folderPage.cursor, 'opaque-folder-cursor');

    expect(state.toMap(), stateJson);
    expect(item.toMap(), {'post': post, ...stateJson});
    expect(page.toMap(), {
      'items': [
        {'post': post, ...stateJson},
      ],
      'cursor': 'opaque-post-cursor',
    });
    expect(folder.toMap(), folderJson);
    expect(folderPage.toMap(), {
      'items': [folderJson],
      'cursor': 'opaque-folder-cursor',
    });

    expect(state.copyWith(folderId: null).folderId, isNull);
    expect(page.copyWith(cursor: null).cursor, isNull);
    expect(folderPage.copyWith(cursor: null).cursor, isNull);
  });

  test('UT-010 private saved DTO diagnostics redact every sentinel', () {
    const sentinels = [
      'did:plc:private-owner',
      'at://did:plc:private-owner/social.craftsky.feed.post/private-rkey',
      'private post content',
      'private-folder-id',
      'Private Folder Name',
      'private-cursor-token',
    ];
    final post = {
      'uri': sentinels[1],
      'cid': 'bafyprivate',
      'rkey': 'private-rkey',
      'text': sentinels[2],
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'quoteCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasReplied': false,
      'viewerHasSaved': true,
      'viewerSavedFolderId': sentinels[3],
      'createdAt': '2026-07-21T10:00:00.000Z',
      'indexedAt': '2026-07-21T10:00:01.000Z',
      'author': {'did': sentinels[0], 'handle': 'private.example'},
    };
    final state = SavedPostState(
      savedAt: DateTime.utc(2026, 7, 21, 11),
      folderId: sentinels[3],
    );
    final item = SavedPostItemMapper.fromMap({
      'post': post,
      'savedAt': '2026-07-21T11:00:00.000Z',
      'folderId': sentinels[3],
    });
    final folder = SavedPostFolder(
      id: sentinels[3],
      name: sentinels[4],
      createdAt: DateTime.utc(2026, 7, 21),
      updatedAt: DateTime.utc(2026, 7, 21),
    );
    final values = <Object>[
      state,
      item,
      SavedPostPage(items: [item], cursor: sentinels[5]),
      folder,
      SavedPostFolderPage(items: [folder], cursor: sentinels[5]),
    ];

    for (final value in values) {
      final diagnostic = value.toString();
      for (final sentinel in sentinels) {
        expect(
          diagnostic,
          isNot(contains(sentinel)),
          reason: '${value.runtimeType} exposed a private sentinel',
        );
      }
    }
  });
}
