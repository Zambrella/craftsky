import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Post', () {
    test('UT-001 saved viewer state', () {
      final base = <String, dynamic>{
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
        'createdAt': '2026-07-21T10:00:00.000Z',
        'indexedAt': '2026-07-21T10:00:01.000Z',
        'author': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
        },
      };

      final foldered = PostMapper.fromMap({
        ...base,
        'viewerHasSaved': true,
        'viewerSavedFolderId': '018f-folder-opaque',
      });
      final unfiled = PostMapper.fromMap({
        ...base,
        'viewerHasSaved': true,
        'viewerSavedFolderId': null,
      });
      final unsaved = PostMapper.fromMap({
        ...base,
        'viewerHasSaved': false,
        'viewerSavedFolderId': null,
      });
      final protected = PostMapper.fromMap({
        'uri': 'at://did:plc:bob/social.craftsky.feed.post/protected',
        'availability': 'blocked',
        'relationship': {'state': 'blocked', 'revealable': false},
      });

      expect(foldered.viewerHasSaved, isTrue);
      expect(foldered.viewerSavedFolderId, '018f-folder-opaque');
      expect(unfiled.viewerHasSaved, isTrue);
      expect(unfiled.viewerSavedFolderId, isNull);
      expect(unsaved.viewerHasSaved, isFalse);
      expect(unsaved.viewerSavedFolderId, isNull);
      expect(protected.viewerHasSaved, isFalse);
      expect(protected.viewerSavedFolderId, isNull);

      final preserved = foldered.copyWith();
      expect(preserved.viewerHasSaved, isTrue);
      expect(preserved.viewerSavedFolderId, '018f-folder-opaque');

      final cleared = foldered.copyWith(
        viewerHasSaved: false,
        viewerSavedFolderId: null,
      );
      expect(cleared.viewerHasSaved, isFalse);
      expect(cleared.viewerSavedFolderId, isNull);

      expect(
        () => PostMapper.fromMap({...base, 'viewerHasSaved': <String>[]}),
        throwsA(anything),
      );
      expect(
        () => PostMapper.fromMap({
          ...base,
          'viewerHasSaved': true,
          'viewerSavedFolderId': <String, dynamic>{},
        }),
        throwsA(anything),
      );
    });

    test('UT-005 decodes content-free muted and blocked placeholders', () {
      final muted = PostMapper.fromMap({
        'uri': 'at://did:plc:bob/social.craftsky.feed.post/muted',
        'availability': 'muted',
        'relationship': {'state': 'muted', 'revealable': true},
      });
      final blocked = PostMapper.fromMap({
        'availability': 'blocked',
        'relationship': {'state': 'blocked', 'revealable': false},
      });

      expect(muted.isProtected, isTrue);
      expect(muted.relationship?.revealable, isTrue);
      expect(muted.text, isEmpty);
      expect(blocked.isProtected, isTrue);
      expect(blocked.availability, 'blocked');
      expect(blocked.relationship?.revealable, isFalse);
      expect(blocked.text, isEmpty);
    });

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
        'viewerHasSaved': false,
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
        'viewerHasSaved': false,
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

    test('AT-003 decodes post-author viewer relationship state', () {
      final author = PostAuthorMapper.fromMap({
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
        'muted': true,
        'blocking': false,
        'blockedBy': false,
      });

      expect(author.muted, isTrue);
      expect(author.blocking, isFalse);
      expect(author.blockedBy, isFalse);
      expect(author.hasViewerState, isTrue);
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
        'viewerHasSaved': false,
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
        'viewerHasSaved': false,
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
        'viewerHasSaved': false,
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
        'viewerHasSaved': false,
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
