import 'dart:typed_data';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  final aliceDid = Did.parse('did:plc:alice');
  final bobDid = Did.parse('did:plc:bob');
  final postRkey = RecordKey.parse('3lf2abc');
  final missingRkey = RecordKey.parse('missing');

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
      'text': text,
      'tags': <String>[],
      'likeCount': 2,
      'repostCount': 1,
      'replyCount': 3,
      'viewerHasLiked': true,
      'viewerHasReposted': false,
      'viewerHasReplied': true,
      'createdAt': '2026-05-04T18:23:45.000Z',
      'indexedAt': '2026-05-04T18:23:47.000Z',
      'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
    };
  }

  group('PostApiClient.uploadImage', () {
    test(
      'POSTs prepared bytes to /v1/blobs/images and parses response',
      () async {
        final dio = buildDio();
        final bytes = Uint8List.fromList([0, 1, 2, 3]);
        DioAdapter(dio: dio).onPost(
          '/v1/blobs/images',
          (server) => server.reply(201, {
            'blob': {
              r'$type': 'blob',
              'ref': {r'$link': 'bafkimage1'},
              'mimeType': 'image/jpeg',
              'size': 253496,
            },
            'cid': 'bafkimage1',
            'mime': 'image/jpeg',
            'size': 253496,
          }),
          data: bytes,
          headers: {'content-type': 'image/jpeg'},
        );

        final uploaded = await PostApiClient(
          dio,
        ).uploadImage(bytes: bytes, mimeType: 'image/jpeg');

        expect(uploaded.cid, 'bafkimage1');
        expect(uploaded.mime, 'image/jpeg');
        expect(uploaded.size, 253496);
        expect(uploaded.blob.type, 'blob');
        expect(uploaded.blob.ref.link, 'bafkimage1');
        expect(uploaded.blob.mimeType, 'image/jpeg');
        expect(uploaded.blob.size, 253496);
      },
    );
  });

  group('PostApiClient.createPost', () {
    test('POSTs /v1/posts with text body and parses response', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'hi')),
        data: {'text': 'hi'},
      );

      final post = await PostApiClient(dio).createPost(text: 'hi');
      expect(post.text, 'hi');
      expect(post.rkey, '3lf2abc');
      expect(post.viewerHasReplied, isTrue);
    });

    test('omits reply for top-level posts', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'top-level')),
        data: {'text': 'top-level'},
      );

      final post = await PostApiClient(dio).createPost(text: 'top-level');
      expect(post.text, 'top-level');
    });

    test('includes facets in create body when provided', () async {
      final dio = buildDio();
      final facets = [
        {
          'index': {'byteStart': 0, 'byteEnd': 6},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'Mending'},
          ],
        },
      ];
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: '#Mending')),
        data: {'text': '#Mending', 'facets': facets},
      );

      final post = await PostApiClient(
        dio,
      ).createPost(text: '#Mending', facets: facets);

      expect(post.text, '#Mending');
    });

    test('sends nested root and parent refs when reply is provided', () async {
      final dio = buildDio();
      final reply = PostReply(
        root: PostRef(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
          cid: 'bafy_root',
        ),
        parent: PostRef(
          uri: 'at://did:plc:bob/social.craftsky.feed.post/parent',
          cid: 'bafy_parent',
        ),
      );
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'reply')),
        data: {
          'text': 'reply',
          'reply': {
            'root': {
              'uri': 'at://did:plc:alice/social.craftsky.feed.post/root',
              'cid': 'bafy_root',
            },
            'parent': {
              'uri': 'at://did:plc:bob/social.craftsky.feed.post/parent',
              'cid': 'bafy_parent',
            },
          },
        },
      );

      final post = await PostApiClient(
        dio,
      ).createPost(text: 'reply', reply: reply);
      expect(post.text, 'reply');
    });

    test(
      'serializes top-level images[] with image, alt, and aspectRatio',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onPost(
          '/v1/posts',
          (server) => server.reply(201, samplePost(text: 'with images')),
          data: {
            'text': 'with images',
            'images': [
              {
                'image': {
                  r'$type': 'blob',
                  'ref': {r'$link': 'bafkimage1'},
                  'mimeType': 'image/jpeg',
                  'size': 253496,
                },
                'alt': 'Blue shawl on a blocking mat',
                'aspectRatio': {'width': 4, 'height': 5},
              },
              {
                'image': {
                  r'$type': 'blob',
                  'ref': {r'$link': 'bafkimage2'},
                  'mimeType': 'image/png',
                  'size': 183122,
                },
                'alt': 'Close-up of the stitch texture',
              },
            ],
          },
        );

        final post = await PostApiClient(dio).createPost(
          text: 'with images',
          images: [
            const CreatePostImage(
              blob: CreatePostBlob(
                ref: CreatePostBlobRef(link: 'bafkimage1'),
                mimeType: 'image/jpeg',
                size: 253496,
              ),
              alt: 'Blue shawl on a blocking mat',
              aspectRatio: CreatePostImageAspectRatio(
                width: 4,
                height: 5,
              ),
            ),
            const CreatePostImage(
              blob: CreatePostBlob(
                ref: CreatePostBlobRef(link: 'bafkimage2'),
                mimeType: 'image/png',
                size: 183122,
              ),
              alt: 'Close-up of the stitch texture',
            ),
          ],
        );

        expect(post.text, 'with images');
      },
    );

    test('omits empty image alt from create payload', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'with image')),
        data: {
          'text': 'with image',
          'images': [
            {
              'image': {
                r'$type': 'blob',
                'ref': {r'$link': 'bafkimage1'},
                'mimeType': 'image/jpeg',
                'size': 253496,
              },
            },
          ],
        },
      );

      final post = await PostApiClient(dio).createPost(
        text: 'with image',
        images: [
          const CreatePostImage(
            blob: CreatePostBlob(
              ref: CreatePostBlobRef(link: 'bafkimage1'),
              mimeType: 'image/jpeg',
              size: 253496,
            ),
          ),
        ],
      );

      expect(post.text, 'with image');
    });

    test('422 validation_failed surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(422, {'error': 'validation_failed'}),
        data: {'text': ''},
      );

      await expectLater(
        () => PostApiClient(dio).createPost(text: ''),
        throwsA(
          isA<ApiBadRequest>().having(
            (e) => e.code,
            'code',
            'validation_failed',
          ),
        ),
      );
    });

    test('preserves provided image order in create payload', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: 'ordered images')),
        data: {
          'text': 'ordered images',
          'images': [
            {
              'image': {
                r'$type': 'blob',
                'ref': {r'$link': 'bafkimageB'},
                'mimeType': 'image/png',
                'size': 444,
              },
              'alt': 'second selected, first in composer now',
            },
            {
              'image': {
                r'$type': 'blob',
                'ref': {r'$link': 'bafkimageA'},
                'mimeType': 'image/jpeg',
                'size': 333,
              },
              'alt': 'first selected, moved second',
              'aspectRatio': {'width': 1, 'height': 1},
            },
          ],
        },
      );

      final post = await PostApiClient(dio).createPost(
        text: 'ordered images',
        images: [
          const CreatePostImage(
            blob: CreatePostBlob(
              ref: CreatePostBlobRef(link: 'bafkimageB'),
              mimeType: 'image/png',
              size: 444,
            ),
            alt: 'second selected, first in composer now',
          ),
          const CreatePostImage(
            blob: CreatePostBlob(
              ref: CreatePostBlobRef(link: 'bafkimageA'),
              mimeType: 'image/jpeg',
              size: 333,
            ),
            alt: 'first selected, moved second',
            aspectRatio: CreatePostImageAspectRatio(width: 1, height: 1),
          ),
        ],
      );

      expect(post.text, 'ordered images');
    });
  });

  group('PostApiClient.getPost', () {
    test('GETs /v1/posts/{did}/{rkey} and parses', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(200, samplePost()),
      );

      final post = await PostApiClient(dio).getPost(aliceDid, postRkey);
      expect(post.rkey, '3lf2abc');
    });

    test('404 surfaces as ApiBadRequest(post_not_found)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/missing',
        (server) => server.reply(404, {'error': 'post_not_found'}),
      );

      await expectLater(
        () => PostApiClient(dio).getPost(aliceDid, missingRkey),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'post_not_found'),
        ),
      );
    });
  });

  group('PostApiClient.deletePost', () {
    test('DELETEs /v1/posts/{did}/{rkey} and returns on 204', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:alice/3lf2abc',
        (server) => server.reply(204, null),
      );

      await PostApiClient(dio).deletePost(aliceDid, postRkey);
    });

    test('403 forbidden surfaces as ApiBadRequest', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/posts/did:plc:bob/3lf2abc',
        (server) => server.reply(403, {'error': 'forbidden'}),
      );

      await expectLater(
        () => PostApiClient(dio).deletePost(bobDid, postRkey),
        throwsA(
          isA<ApiBadRequest>().having((e) => e.code, 'code', 'forbidden'),
        ),
      );
    });
  });

  group('PostApiClient.listPostsByAuthor', () {
    test('GETs /v1/profiles/@{handleOrDid}/posts (no cursor)', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next-cursor',
        }),
      );

      final page = await PostApiClient(
        dio,
      ).listPostsByAuthor('alice.craftsky.social');
      expect(page.items, hasLength(1));
      expect(page.cursor, 'next-cursor');
    });

    test('passes cursor and limit as query params', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/posts',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {'cursor': 'c1', 'limit': '50'},
      );

      final page = await PostApiClient(
        dio,
      ).listPostsByAuthor('alice.craftsky.social', cursor: 'c1', limit: 50);
      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });
  });

  group('PostApiClient.listTimeline', () {
    test('GETs /v1/feed/timeline with no cursor and parses PostPage', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/feed/timeline',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next-cursor',
        }),
      );

      final page = await PostApiClient(dio).listTimeline();

      expect(page.items, hasLength(1));
      expect(page.items.single.text, 'hello');
      expect(page.cursor, 'next-cursor');
    });

    test('passes cursor and limit as query params', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/feed/timeline',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {'cursor': 'opaque:abc', 'limit': '20'},
      );

      final page = await PostApiClient(
        dio,
      ).listTimeline(cursor: 'opaque:abc', limit: 20);

      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });

    test('parses an empty page without a cursor', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/feed/timeline',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
      );

      final page = await PostApiClient(dio).listTimeline();

      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });

    test('maps error envelopes through the shared ApiException path', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/feed/timeline',
        (server) => server.reply(500, {'error': 'timeline_unavailable'}),
      );

      await expectLater(
        () => PostApiClient(dio).listTimeline(),
        throwsA(
          isA<ApiServerError>().having(
            (e) => e.message,
            'message',
            'http_500',
          ),
        ),
      );
    });
  });

  group('PostApiClient.listCommentsByAuthor', () {
    test('GETs /v1/profiles/@{handleOrDid}/comments', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/comments',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next-cursor',
        }),
      );

      final page = await PostApiClient(
        dio,
      ).listCommentsByAuthor('alice.craftsky.social');
      expect(page.items, hasLength(1));
      expect(page.cursor, 'next-cursor');
    });

    test('passes cursor and limit as query params', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/profiles/@alice.craftsky.social/comments',
        (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
        queryParameters: {'cursor': 'c1', 'limit': '50'},
      );

      final page = await PostApiClient(dio).listCommentsByAuthor(
        'alice.craftsky.social',
        cursor: 'c1',
        limit: 50,
      );
      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
    });
  });

  group('PostApiClient.listCommentBranchReplies', () {
    test('GETs /v1/posts/{did}/{rkey}/replies with pagination', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/posts/did:plc:alice/3lf2abc/replies',
        (server) => server.reply(200, {
          'loaded': true,
          'items': [
            {'post': samplePost(text: 'reply'), 'flattened': false},
            {
              'post': samplePost(text: 'nested reply'),
              'flattened': true,
              'replyingTo': {
                'uri': 'at://did:plc:bob/social.craftsky.feed.post/reply',
                'did': 'did:plc:bob',
                'handle': 'bob.craftsky.social',
                'displayName': 'Bob',
              },
            },
          ],
          'cursor': 'next-replies',
        }),
        queryParameters: {'cursor': 'c1', 'limit': '25'},
      );

      final page =
          await PostApiClient(
            dio,
          ).listCommentBranchReplies(
            aliceDid,
            postRkey,
            cursor: 'c1',
            limit: 25,
          );
      expect(page.loaded, isTrue);
      expect(page.items.first.post.text, 'reply');
      expect(page.items.first.flattened, isFalse);
      expect(page.items.last.post.text, 'nested reply');
      expect(page.items.last.flattened, isTrue);
      expect(page.items.last.replyingTo?.handle, 'bob.craftsky.social');
      expect(page.cursor, 'next-replies');
    });
  });

  group('PostApiClient likes and reposts', () {
    final interaction = {
      'uri': 'at://did:plc:viewer/social.craftsky.feed.like/like1',
      'cid': 'bafy_like',
      'rkey': 'like1',
      'subject': {
        'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
        'cid': 'bafy123',
      },
      'createdAt': '2026-05-04T18:25:00.000Z',
    };

    test('POSTs and DELETEs like endpoint', () async {
      final dio = buildDio();
      final adapter = DioAdapter(dio: dio)
        ..onPost(
          '/v1/posts/did:plc:alice/3lf2abc/likes',
          (server) => server.reply(201, interaction),
        )
        ..onDelete(
          '/v1/posts/did:plc:alice/3lf2abc/likes',
          (server) => server.reply(204, null),
        );

      final client = PostApiClient(dio);
      final like = await client.likePost(aliceDid, postRkey);
      await client.unlikePost(aliceDid, postRkey);

      expect(like.rkey, 'like1');
      expect(adapter, isNotNull);
    });

    test('POSTs and DELETEs repost endpoint', () async {
      final dio = buildDio();
      DioAdapter(dio: dio)
        ..onPost(
          '/v1/posts/did:plc:alice/3lf2abc/reposts',
          (server) => server.reply(201, interaction),
        )
        ..onDelete(
          '/v1/posts/did:plc:alice/3lf2abc/reposts',
          (server) => server.reply(204, null),
        );

      final client = PostApiClient(dio);
      final repost = await client.repostPost(aliceDid, postRkey);
      await client.unrepostPost(aliceDid, postRkey);

      expect(repost.subject.cid, 'bafy123');
    });
  });

  group('PostApiClient.reportPost', () {
    test('report wire models use dart mappable serialization', () {
      expect(
        const ReportSubmission(
          reasonType: 'spam',
          details: 'private details',
        ).toMap(),
        {'reasonType': 'spam', 'details': 'private details'},
      );
      expect(
        const ReportSubmission(reasonType: 'other').toMap(),
        {'reasonType': 'other'},
      );

      final result = ReportResultMapper.fromMap({
        'reportId': 'report-post-1',
        'status': 'accepted',
      });
      expect(result.reportId, 'report-post-1');
      expect(result.status, 'accepted');
    });

    test('POSTs report body and parses accepted response', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts/did:plc:alice/3lf2abc/reports',
        (server) => server.reply(201, {
          'reportId': 'report-post-1',
          'status': 'accepted',
        }),
        data: {'reasonType': 'spam', 'details': 'private details'},
      );

      final result = await PostApiClient(dio).reportPost(
        aliceDid,
        postRkey,
        const ReportSubmission(
          reasonType: 'spam',
          details: 'private details',
        ),
      );

      expect(result.reportId, 'report-post-1');
      expect(result.status, 'accepted');
    });

    test('omits details when report details are absent', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onPost(
        '/v1/posts/did:plc:alice/3lf2abc/reports',
        (server) => server.reply(201, {
          'reportId': 'report-post-2',
          'status': 'accepted',
        }),
        data: {'reasonType': 'other'},
      );

      final result = await PostApiClient(dio).reportPost(
        aliceDid,
        postRkey,
        const ReportSubmission(reasonType: 'other'),
      );

      expect(result.status, 'accepted');
    });
  });
}
