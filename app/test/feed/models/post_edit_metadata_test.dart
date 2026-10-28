import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/post_fixtures.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'IT-002 authoritative editor seed does not invent a record revision',
    () {
      final post = postFixture();
      final seed = PostEditSeed.fromMap({
        'post': post.toMap(),
        'baseCid': post.cid.value,
        'editable': {'text': post.text, 'sponsored': post.sponsored},
      });
      expect(seed.recordRevision, isNull);
      expect(seed.baseCid, post.cid);
    },
  );

  test('UT-005 post and quoted original retain independent edit metadata', () {
    final post = PostMapper.fromMap({
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/post',
      'cid': 'bafycurrent',
      'rkey': 'post',
      'text': 'My commentary',
      'tags': <String>[],
      'createdAt': '2026-01-01T00:00:00Z',
      'indexedAt': '2026-01-01T00:00:01Z',
      'editedAt': '2026-10-26T12:00:00Z',
      'recordRevision': '3aaaaaaaaaaa3',
      'likeCount': 7,
      'repostCount': 2,
      'replyCount': 1,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasSaved': true,
      'sponsored': false,
      'author': {'did': 'did:plc:alice', 'handle': 'alice.example'},
      'quoteView': {
        'state': 'visible',
        'revealable': false,
        'post': {
          'uri': 'at://did:plc:bob/social.craftsky.feed.post/original',
          'cid': 'bafyoriginal',
          'text': 'Updated original',
          'createdAt': '2025-01-01T00:00:00Z',
          'editedAt': '2026-10-25T12:00:00Z',
          'recordRevision': '3aaaaaaaaaaa2',
          'sponsored': false,
          'author': {'did': 'did:plc:bob', 'handle': 'bob.example'},
        },
      },
    });
    expect(post.editedAt, DateTime.utc(2026, 10, 26, 12));
    expect(post.recordRevision, '3aaaaaaaaaaa3');
    expect(post.quoteView?.post?.editedAt, DateTime.utc(2026, 10, 25, 12));
    expect(post.quoteView?.post?.recordRevision, '3aaaaaaaaaaa2');
    expect(post.likeCount, 7);
  });
}
