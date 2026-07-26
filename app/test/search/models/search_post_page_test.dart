import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> samplePost({String rkey = '3lf2abc'}) => {
    'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    'cid': 'bafy_$rkey',
    'rkey': rkey,
    'text': 'search result $rkey',
    'tags': ['sockkal'],
    'likeCount': 2,
    'repostCount': 1,
    'replyCount': 3,
    'viewerHasLiked': false,
    'viewerHasReposted': true,
    'viewerHasSaved': false,
    'createdAt': '2026-05-04T18:23:45.000Z',
    'indexedAt': '2026-05-04T18:23:47.000Z',
    'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
  };

  test('UT-002 decodes search page items as existing Post objects', () {
    final page = SearchPostPageMapper.fromMap({
      'hashtag': 'sockkal',
      'items': [samplePost(rkey: 'a')],
      'cursor': 'opaque:next',
    });

    expect(page.hashtag, 'sockkal');
    expect(page.cursor, 'opaque:next');
    expect(page.items, hasLength(1));
    expect(page.items.single, isA<Post>());
    expect(page.items.single.uri.toString(), contains('/a'));
  });
}
