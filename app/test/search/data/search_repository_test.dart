import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/search/data/api_search_repository.dart';
import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() =>
      Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());

  Map<String, dynamic> samplePost() => {
    'uri': 'at://did:plc:alice/social.craftsky.feed.post/a',
    'cid': 'bafy_a',
    'rkey': 'a',
    'text': 'delegated',
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

  test(
    'IT-011 ApiSearchRepository delegates arguments to SearchApiClient',
    () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/posts',
        (server) => server.reply(200, {
          'items': [samplePost()],
          'cursor': 'next',
        }),
        queryParameters: {
          'q': 'alpaca',
          'sort': 'popular',
          'limit': '7',
          'cursor': 'opaque:start',
        },
      );

      final repo = ApiSearchRepository(SearchApiClient(dio));
      final page = await repo.searchPosts(
        q: 'alpaca',
        sort: SearchSort.popular,
        limit: 7,
        cursor: 'opaque:start',
      );

      expect(page.cursor, 'next');
      expect(page.items.single.text, 'delegated');
    },
  );
}
