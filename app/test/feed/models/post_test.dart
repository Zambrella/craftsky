import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/projects/models/project.dart';
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
        'quoteCount': 0,
        'replyCount': 4,
        'viewerHasLiked': true,
        'viewerHasReposted': false,
        'viewerHasReplied': true,
        'reply': {
          'root': {'uri': 'at://x/y/1', 'cid': 'bafyR'},
          'parent': {'uri': 'at://x/y/2', 'cid': 'bafyP'},
        },
        'quote': {'uri': 'at://x/y/q', 'cid': 'bafyQ'},
        'images': [
          {
            'cid': 'bafkimage1',
            'mime': 'image/jpeg',
            'size': 253496,
            'alt': 'Blue shawl on blocking mats',
            'aspectRatio': {'width': 4, 'height': 5},
            'thumb': 'https://cdn.example.com/thumb/1.jpg',
            'fullsize': 'https://cdn.example.com/full/1.jpg',
          },
          {
            'cid': 'bafkimage2',
            'mime': 'image/png',
            'size': 183122,
            'alt': 'Close-up stitch detail',
          },
        ],
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
          'avatarCid': 'bafyA',
        },
        'moderation': {'warningKind': 'post'},
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
      expect(post.viewerHasReplied, isTrue);
      expect(post.author.handle, 'alice.craftsky.social');
      expect(post.author.avatarCid, 'bafyA');
      expect(post.moderation?.warningKind, 'post');
      expect(post.images, hasLength(2));
      expect(post.images!.first.cid, 'bafkimage1');
      expect(post.images!.first.mime, 'image/jpeg');
      expect(post.images!.first.size, 253496);
      expect(post.images!.first.alt, 'Blue shawl on blocking mats');
      expect(post.images!.first.aspectRatio?.width, 4);
      expect(post.images!.first.aspectRatio?.height, 5);
      expect(post.images!.first.thumb, 'https://cdn.example.com/thumb/1.jpg');
      expect(post.images!.first.fullsize, 'https://cdn.example.com/full/1.jpg');
      expect(post.images!.last.aspectRatio, isNull);
      expect(post.images!.last.thumb, isNull);
      expect(post.images!.last.fullsize, isNull);

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
        'quoteCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
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
      expect(post.images, isNull);
      expect(post.tags, isEmpty);
      expect(post.toMap(), json);
    });

    test('UT-006 decodes quote count and compact quote preview', () {
      final json = {
        'uri': 'at://did:plc:bob/social.craftsky.feed.post/quote',
        'cid': 'bafyquote',
        'rkey': 'quote',
        'text': 'This is useful context.',
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 2,
        'quoteCount': 3,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
        'quote': {
          'uri': 'at://did:plc:carol/social.craftsky.feed.post/root',
          'cid': 'bafyroot',
        },
        'quoteView': {
          'state': 'visible',
          'post': {
            'uri': 'at://did:plc:carol/social.craftsky.feed.post/root',
            'cid': 'bafyroot',
            'text': 'Original post',
            'createdAt': '2026-05-04T18:20:00.000Z',
            'author': {
              'did': 'did:plc:carol',
              'handle': 'carol.craftsky.social',
              'displayName': 'Carol',
            },
            'images': [
              {
                'cid': 'bafkpreview',
                'mime': 'image/jpeg',
                'size': 42,
                'alt': 'Preview image',
              },
            ],
          },
        },
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {'did': 'did:plc:bob', 'handle': 'bob.craftsky.social'},
      };

      final post = PostMapper.fromMap(json);

      expect(post.quoteCount, 3);
      expect(post.quoteView?.state, 'visible');
      expect(
        post.quoteView?.post?.uri,
        'at://did:plc:carol/social.craftsky.feed.post/root',
      );
      expect(post.quoteView?.post?.author.handle, 'carol.craftsky.social');
      expect(post.quoteView?.post?.images, hasLength(1));
      expect(post.toMap(), json);
    });

    test('UT-006 defaults absent quote count and quote view', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'hello',
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 0,
        'quoteCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
      };

      final post = PostMapper.fromMap(json);

      expect(post.quoteCount, 0);
      expect(post.quoteView, isNull);
    });

    test('UT-008 exposes optional project metadata for project posts', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2project',
        'cid': 'bafyproject',
        'rkey': '3lf2project',
        'text': 'Finished my shawl.',
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 0,
        'quoteCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
        'project': {
          'common': {
            'craftType': 'social.craftsky.feed.defs#knitting',
            'title': 'Hitchhiker Shawl',
          },
          'details': {
            r'$type': knittingProjectDetailsType,
            'needleSizeMm': '4.0mm',
          },
        },
      };

      final post = PostMapper.fromMap(json);

      expect(post.project, isA<Project>());
      expect(post.project?.common.title, 'Hitchhiker Shawl');
      expect(post.project?.details, isA<KnittingProjectDetails>());
      expect(post.toMap(), json);
    });

    test('UT-009 general posts omit project when absent', () {
      final json = {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
        'rkey': '3lf2abc',
        'text': 'hello',
        'tags': <String>[],
        'likeCount': 0,
        'repostCount': 0,
        'quoteCount': 0,
        'replyCount': 0,
        'viewerHasLiked': false,
        'viewerHasReposted': false,
        'viewerHasReplied': false,
        'createdAt': '2026-05-04T18:23:45.000Z',
        'indexedAt': '2026-05-04T18:23:47.000Z',
        'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
      };

      final post = PostMapper.fromMap(json);

      expect(post.project, isNull);
      expect(post.toMap(), isNot(contains('project')));
      expect(post.toMap(), json);
    });
  });
}
