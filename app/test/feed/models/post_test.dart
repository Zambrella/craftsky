import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Post', () {
    test('round-trips a fully-populated wire payload', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'Cast on for the Hitchhiker shawl tonight.',
        'facets': [
          {
            'index': {'byteStart': 0, 'byteEnd': 7},
            'features': [
              {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'knitting'},
            ],
          },
        ],
        'tags': ['knitting'],
        'likeCount': 7,
        'repostCount': 2,
        'replyCount': 4,
        'viewerHasLiked': true,
        'viewerHasReposted': false,
        'reply': {
          'root': {'uri': 'at://x/y/1', 'cid': 'bafyR'},
          'parent': {'uri': 'at://x/y/2', 'cid': 'bafyP'},
        },
        'quote': {'uri': 'at://x/y/q', 'cid': 'bafyQ'},
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
          'avatarCid': 'bafyA',
        },
      };

      final post = PostMapper.fromMap(json);

      expect(post.uri, json['uri']);
      expect(post.text, json['text']);
      expect(post.tags, ['knitting']);
      expect(post.reply!.root.uri, 'at://x/y/1');
      expect(post.reply!.parent.cid, 'bafyP');
      expect(post.quote!.uri, 'at://x/y/q');
      expect(post.likeCount, 7);
      expect(post.viewerHasLiked, isTrue);
      expect(post.author.handle, 'alice.craftsky.social');
      expect(post.author.avatarCid, 'bafyA');

      expect(post.toMap(), json);
    });

    test('round-trips a minimal payload (optionals absent)', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'hello',
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

      final post = PostMapper.fromMap(json);

      expect(post.facets, isNull);
      expect(post.reply, isNull);
      expect(post.quote, isNull);
      expect(post.author.displayName, isNull);
      expect(post.author.avatarCid, isNull);
      expect(post.tags, isEmpty);
      expect(post.toMap(), json);
    });
  });
}
