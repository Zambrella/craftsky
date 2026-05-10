import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_thread.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> post(String rkey, String text) => {
    'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    'cid': 'bafy_$rkey',
    'rkey': rkey,
    'text': text,
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'createdAt': '2026-05-04T18:23:45.000Z',
    'indexedAt': '2026-05-04T18:23:47.000Z',
    'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
  };

  group('PostThread', () {
    test('parses ancestors in root-to-parent order', () {
      final thread = PostThreadMapper.fromMap({
        'ancestors': [post('root', 'root'), post('parent', 'parent')],
        'post': post('target', 'target'),
        'replies': <Map<String, dynamic>>[],
        'truncated': false,
      });

      expect(thread.ancestors.map((post) => post.rkey), ['root', 'parent']);
      expect(thread.post.rkey, 'target');
      expect(thread.replies, isEmpty);
      expect(thread.truncated, isFalse);
    });
  });
}
