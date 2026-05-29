import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> actor([String did = 'did:plc:alice']) => {
    'did': did,
    'handle': 'alice.craftsky.social',
    'displayName': 'Alice',
    'avatarCid': 'bafyavatar',
  };

  Map<String, dynamic> post() => {
    'uri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
    'cid': 'bafypost',
    'rkey': 'root',
    'text': 'viewer post',
    'tags': <String>[],
    'likeCount': 1,
    'repostCount': 2,
    'replyCount': 3,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasReplied': false,
    'createdAt': '2026-05-28T12:00:00Z',
    'indexedAt': '2026-05-28T12:00:01Z',
    'author': {'did': 'did:plc:viewer', 'handle': 'viewer.craftsky.social'},
  };

  Map<String, dynamic> base(String type, String rkey) => {
    'uri': 'at://did:plc:alice/social.craftsky.feed.$type/$rkey',
    'cid': 'bafy$rkey',
    'rkey': rkey,
    'type': type,
    'actor': actor(),
    'createdAt': '2026-05-28T13:00:00Z',
    'indexedAt': '2026-05-28T13:00:01Z',
  };

  test('decodes mixed notification item types and opaque cursor', () {
    final page = NotificationPage.fromMap({
      'cursor': 'opaque-next',
      'items': [
        base('follow', 'follow1'),
        {...base('like', 'like1'), 'subjectPost': post()},
        {...base('repost', 'repost1'), 'subjectPost': post()},
        {
          ...base('reply', 'reply1'),
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/reply1',
          'subjectPost': post(),
          'reply': {
            'uri': 'at://did:plc:alice/social.craftsky.feed.post/reply1',
            'cid': 'bafyreply1',
            'rkey': 'reply1',
          },
        },
      ],
    });

    expect(page.cursor, 'opaque-next');
    expect(page.items, hasLength(4));
    expect(page.items[0], isA<FollowNotification>());
    expect(page.items[1], isA<LikeNotification>());
    expect(page.items[2], isA<RepostNotification>());
    expect(page.items[3], isA<ReplyNotification>());
    expect(page.items[0].actor.displayLabel, 'Alice');
    expect((page.items[1] as LikeNotification).subjectPost.text, 'viewer post');
    expect(
      (page.items[3] as ReplyNotification).reply!.rkey.toString(),
      'reply1',
    );
  });
}
