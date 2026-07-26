import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/saved_posts/navigation/saved_post_destination.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-007 infers exact top-level, comment, and reply destinations', () {
    final rootUri = AtUri.parse(
      'at://did:plc:author/social.craftsky.feed.post/root',
    );
    final commentUri = AtUri.parse(
      'at://did:plc:commenter/social.craftsky.feed.post/comment',
    );
    final nestedUri = AtUri.parse(
      'at://did:plc:replier/social.craftsky.feed.post/nested',
    );

    expect(
      SavedPostDestination.forItem(_item('root')),
      SavedPostDestination(threadUri: rootUri),
    );
    expect(
      SavedPostDestination.forItem(
        _item('comment', rootUri: rootUri, parentUri: rootUri),
      ),
      SavedPostDestination(threadUri: rootUri, focusUri: commentUri),
    );
    expect(
      SavedPostDestination.forItem(
        _item('nested', rootUri: rootUri, parentUri: commentUri),
      ),
      SavedPostDestination(threadUri: rootUri, focusUri: nestedUri),
    );

    final focused = SavedPostDestination(
      threadUri: rootUri,
      focusUri: nestedUri,
    );
    expect(
      focused.copyWith(focusUri: null),
      SavedPostDestination(threadUri: rootUri),
    );
    expect(focused.toString(), isNot(contains(nestedUri.toString())));
  });
}

SavedPostItem _item(
  String rkey, {
  AtUri? rootUri,
  AtUri? parentUri,
}) {
  final uri =
      'at://did:plc:${rkey == 'root'
          ? 'author'
          : rkey == 'comment'
          ? 'commenter'
          : 'replier'}/social.craftsky.feed.post/$rkey';
  return SavedPostItemMapper.fromMap({
    'post': {
      'uri': uri,
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
      if (rootUri != null)
        'reply': {
          'root': {'uri': rootUri.toString(), 'cid': 'bafyroot'},
          'parent': {
            'uri': parentUri.toString(),
            'cid': 'bafyparent',
          },
        },
    },
    'savedAt': '2026-07-21T12:00:00.000Z',
  });
}
