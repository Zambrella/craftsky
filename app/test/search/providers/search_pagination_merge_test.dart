import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, dynamic> _postMap(String rkey) => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
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

Post _post(String rkey) => PostMapper.fromMap(_postMap(rkey));

void main() {
  setUpAll(initializeMappers);

  test(
    'UT-008 appendUniquePosts keeps existing duplicate and appends new rows',
    () {
      final merged = appendUniquePosts([_post('a')], [_post('a'), _post('b')]);

      expect(merged.map((post) => post.rkey.toString()), ['a', 'b']);
    },
  );
}
