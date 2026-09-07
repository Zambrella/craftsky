import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('PostPage', () {
    test('round-trips with cursor present', () {
      const cursor =
          'eyJpbmRleGVkQXQiOiIyMDI2LTA1LTA0VDE4OjIzOjQ3WiIsInVyaSI6ImF0Oi8vIn0';
      final json = {
        'items': <Map<String, dynamic>>[],
        'cursor': cursor,
      };

      final page = PostPageMapper.fromMap(json);
      expect(page.items, isEmpty);
      expect(page.cursor, cursor);
      expect(page.toMap(), json);
    });

    test('absent cursor decodes as null and re-encodes without the key', () {
      final json = {'items': <Map<String, dynamic>>[]};

      final page = PostPageMapper.fromMap(json);
      expect(page.cursor, isNull);

      // Re-encoding omits the null cursor entirely (matches AppView's
      // pagination contract: `cursor` is omitted, not `null`, when no
      // more pages exist).
      expect(page.toMap(), {'items': <Map<String, dynamic>>[]});
    });

    test('UT-001 round-trips visible page-level pinnedPostUri', () {
      const pinned =
          'at://did:plc:alice/social.craftsky.feed.post/pinned-standard';
      final json = {
        'items': <Map<String, dynamic>>[],
        'cursor': 'opaque-next',
        'pinnedPostUri': pinned,
      };

      final page = PostPageMapper.fromMap(json);

      expect(page.pinnedPostUri, pinned);
      expect(page.toMap(), json);
    });

    test('UT-001 treats absent or null pin metadata as omitted on encode', () {
      final absent = PostPageMapper.fromMap({
        'items': <Map<String, dynamic>>[],
      });
      final explicitNull = PostPageMapper.fromMap({
        'items': <Map<String, dynamic>>[],
        'pinnedPostUri': null,
      });

      expect(absent.pinnedPostUri, isNull);
      expect(explicitNull.pinnedPostUri, isNull);
      expect(absent.toMap(), {'items': <Map<String, dynamic>>[]});
      expect(explicitNull.toMap(), {'items': <Map<String, dynamic>>[]});
      expect(absent.toMap().containsKey('isPinned'), isFalse);
      expect(explicitNull.toMap().containsKey('isPinned'), isFalse);
    });

    test('UT-007 decodes quote post pages without pin metadata', () {
      Map<String, dynamic> quotePost(String rkey) => {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
        'cid': 'bafy$rkey',
        'rkey': rkey,
        'text': 'A quote post',
        'tags': ['knitting'],
        'createdAt': '2026-09-06T10:00:00.000Z',
        'indexedAt': '2026-09-06T10:00:01.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
        },
        'likeCount': 3,
        'repostCount': 1,
        'quoteCount': 2,
        'replyCount': 4,
        'viewerHasLiked': true,
        'viewerHasReposted': false,
        'viewerHasReplied': true,
        'viewerHasSaved': false,
      };

      final firstPage = PostPageMapper.fromMap({
        'items': [quotePost('quote-first')],
        'cursor': 'opaque-quotes-next',
      });
      final finalPage = PostPageMapper.fromMap({
        'items': [quotePost('quote-final')],
      });

      expect(firstPage.cursor, 'opaque-quotes-next');
      expect(firstPage.pinnedPostUri, isNull);
      final post = firstPage.items.single;
      expect(
        post.uri.toString(),
        'at://did:plc:alice/social.craftsky.feed.post/quote-first',
      );
      expect(post.author.did.toString(), 'did:plc:alice');
      expect(post.author.handle.toString(), 'alice.craftsky.social');
      expect(post.likeCount, 3);
      expect(post.repostCount, 1);
      expect(post.quoteCount, 2);
      expect(post.replyCount, 4);
      expect(post.viewerHasLiked, isTrue);
      expect(post.viewerHasReplied, isTrue);

      expect(finalPage.cursor, isNull);
      expect(finalPage.pinnedPostUri, isNull);
      expect(finalPage.items.single.rkey.toString(), 'quote-final');
      expect(firstPage.toMap().containsKey('pinnedPostUri'), isFalse);
      expect(finalPage.toMap().containsKey('pinnedPostUri'), isFalse);
    });
  });
}
