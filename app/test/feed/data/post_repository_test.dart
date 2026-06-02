import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
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
  }

  group('ApiPostRepository.create', () {
    test('IT-002 forwards facets to the API client', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
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

      final post = await ApiPostRepository(
        PostApiClient(dio),
      ).create(text: '#Mending', facets: facets);

      expect(post.text, '#Mending');
    });
  });

  group('PostRepository.listTimeline', () {
    test('fake exposes timeline method without handle or DID input', () async {
      String? seenCursor;
      int? seenLimit;
      final repo = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          seenCursor = cursor;
          seenLimit = limit;
          return const PostPage(items: [], cursor: 'next');
        },
      );

      final asInterface = repo as PostRepository;
      final page = await asInterface.listTimeline(cursor: 'c1', limit: 20);

      expect(seenCursor, 'c1');
      expect(seenLimit, 20);
      expect(page.cursor, 'next');
    });
  });
}
