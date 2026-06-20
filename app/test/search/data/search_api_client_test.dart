import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

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
    'viewerHasReplied': false,
    'createdAt': '2026-05-04T18:23:45.000Z',
    'indexedAt': '2026-05-04T18:23:47.000Z',
    'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
  };

  group('SearchApiClient.searchHashtagPosts', () {
    test(
      'IT-002 sends path/query and decodes Post items with cursor',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/search/hashtags/SockKAL/posts',
          (server) => server.reply(200, {
            'hashtag': 'sockkal',
            'items': [samplePost(rkey: 'a'), samplePost(rkey: 'b')],
            'cursor': 'opaque:next',
          }),
          queryParameters: {
            'sort': 'popular',
            'limit': '20',
            'cursor': 'opaque:start',
          },
        );

        final page = await SearchApiClient(dio).searchHashtagPosts(
          'SockKAL',
          sort: SearchSort.popular,
          limit: 20,
          cursor: 'opaque:start',
        );

        expect(page.hashtag, 'sockkal');
        expect(page.cursor, 'opaque:next');
        expect(page.items, everyElement(isA<Post>()));
        expect(page.items.map((post) => post.rkey), ['a', 'b']);
      },
    );

    test('IT-002 encodes hashtag values as one safe path segment', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/hashtags/fiber%20art/posts',
        (server) => server.reply(400, {
          'error': 'validation_failed',
          'message': 'invalid hashtag',
          'requestId': 'req_safe_path',
        }),
      );

      await expectLater(
        () => SearchApiClient(dio).searchHashtagPosts('fiber art'),
        throwsA(isA<ApiBadRequest>()),
      );
    });

    test('IT-002 invalid cursor errors surface as ApiException', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/hashtags/SockKAL/posts',
        (server) => server.reply(400, {
          'error': 'invalid_cursor',
          'message': 'invalid cursor',
          'requestId': 'req_cursor',
        }),
        queryParameters: {'cursor': 'not-a-real-cursor'},
      );

      await expectLater(
        () => SearchApiClient(
          dio,
        ).searchHashtagPosts('SockKAL', cursor: 'not-a-real-cursor'),
        throwsA(
          isA<ApiBadRequest>().having(
            (error) => error.code,
            'code',
            'invalid_cursor',
          ),
        ),
      );
    });
  });

  group('SearchApiClient.searchProfiles', () {
    test(
      'IT-003 sends q/limit/cursor and decodes follow state without sort',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/search/profiles',
          (server) => server.reply(200, {
            'items': [
              {
                'did': 'did:plc:alice',
                'handle': 'alice.craftsky.social',
                'displayName': 'Alice',
                'description': 'knitter',
                'avatar': 'https://example.com/a.jpg',
                'isCraftskyProfile': true,
                'viewerIsFollowing': true,
              },
            ],
            'cursor': 'opaque:profiles',
          }),
          queryParameters: {
            'q': 'ali',
            'limit': '25',
            'cursor': 'opaque:start',
          },
        );

        final page = await SearchApiClient(
          dio,
        ).searchProfiles(q: 'ali', limit: 25, cursor: 'opaque:start');

        expect(page.cursor, 'opaque:profiles');
        expect(page.items.single.viewerIsFollowing, isTrue);
        expect(
          page.items.single.summary.handle.toString(),
          'alice.craftsky.social',
        );
      },
    );
  });

  group('SearchApiClient post/project search', () {
    test(
      'IT-004 sends post search q/sort/limit/cursor and decodes Post page',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/search/posts',
          (server) => server.reply(200, {
            'items': [samplePost(rkey: 'post')],
            'cursor': 'opaque:posts',
          }),
          queryParameters: {
            'q': 'alpaca',
            'sort': 'chronological',
            'limit': '10',
            'cursor': 'opaque:start',
          },
        );

        final page = await SearchApiClient(dio).searchPosts(
          q: 'alpaca',
          sort: SearchSort.chronological,
          limit: 10,
          cursor: 'opaque:start',
        );

        expect(page.cursor, 'opaque:posts');
        expect(page.items.single, isA<Post>());
      },
    );

    test(
      'IT-005 sends project search repeated filters and browse-all',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio)
          ..onGet(
            '/v1/search/projects',
            (server) => server.reply(200, {
              'items': [samplePost(rkey: 'project')],
            }),
            queryParameters: {
              'q': 'cardigan',
              'sort': 'popular',
              'limit': '25',
              'cursor': 'opaque:projects',
              'craftType': ['knitting', 'crochet'],
              'material': ['wool', 'cotton'],
              'designTag': ['cables'],
              'projectTag': ['gift'],
            },
          )
          ..onGet(
            '/v1/search/projects',
            (server) => server.reply(200, {'items': <Map<String, dynamic>>[]}),
            queryParameters: {'sort': 'chronological'},
          );

        const filters = ProjectSearchFilters(
          craftType: ['knitting', 'crochet'],
          material: ['wool', 'cotton'],
          designTag: ['cables'],
          projectTag: ['gift'],
        );
        final page = await SearchApiClient(dio).searchProjects(
          q: 'cardigan',
          sort: SearchSort.popular,
          filters: filters,
          limit: 25,
          cursor: 'opaque:projects',
        );
        final browse = await SearchApiClient(
          dio,
        ).searchProjects(sort: SearchSort.chronological);

        expect(page.items.single.rkey.toString(), 'project');
        expect(browse.items, isEmpty);
      },
    );
  });

  group('SearchApiClient top hashtags and recents', () {
    test('IT-006 requests repeated craftTypes and decodes groups', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/hashtags/top',
        (server) => server.reply(200, {
          'groups': [
            {
              'craftType': 'knitting',
              'items': [
                {'tag': 'sockkal', 'count': 12},
              ],
            },
            {'craftType': 'crochet', 'items': <Map<String, dynamic>>[]},
          ],
        }),
        queryParameters: {
          'craftTypes': ['knitting', 'crochet'],
          'limit': '10',
        },
      );

      final response = await SearchApiClient(
        dio,
      ).topHashtags(craftTypes: ['knitting', 'crochet'], limit: 10);

      expect(response.groups.map((group) => group.craftType), [
        'knitting',
        'crochet',
      ]);
      expect(response.groups.first.items.single.count, 12);
    });

    test('IT-007 lists recent searches with typed payloads', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/recent',
        (server) => server.reply(200, {
          'items': [
            {
              'id': 'recent_1',
              'type': 'hashtag',
              'displayLabel': '#SockKAL',
              'payload': {'tag': 'sockkal', 'sort': 'popular'},
              'updatedAt': '2026-06-20T10:00:00Z',
            },
            {
              'id': 'recent_2',
              'type': 'profile',
              'displayLabel': 'alice',
              'payload': {'q': 'alice'},
              'updatedAt': '2026-06-20T11:00:00Z',
            },
          ],
        }),
      );

      final page = await SearchApiClient(dio).listRecentSearches();

      expect(page.items.map((item) => item.id), ['recent_1', 'recent_2']);
      expect(page.items.first.payload, isA<HashtagRecentSearchPayload>());
      expect(page.items.last.payload, isA<ProfileRecentSearchPayload>());
    });

    test('IT-008 saves recent search payloads', () async {
      final dio = buildDio();
      const request = SaveRecentSearchRequest(
        type: RecentSearchType.project,
        displayLabel: 'Cardigan projects',
        payload: ProjectRecentSearchPayload(
          q: 'cardigan',
          sort: SearchSort.popular,
          filters: ProjectSearchFilters(craftType: ['knitting']),
        ),
      );
      DioAdapter(dio: dio).onPost(
        '/v1/search/recent',
        (server) => server.reply(201, {
          'id': 'recent_project',
          'type': 'project',
          'displayLabel': 'Cardigan projects',
          'payload': {
            'q': 'cardigan',
            'sort': 'popular',
            'filters': {
              'craftType': ['knitting'],
            },
          },
          'updatedAt': '2026-06-20T12:00:00Z',
        }),
        data: {
          'type': 'project',
          'displayLabel': 'Cardigan projects',
          'payload': {
            'q': 'cardigan',
            'sort': 'popular',
            'filters': {
              'craftType': ['knitting'],
            },
          },
        },
      );

      final saved = await SearchApiClient(dio).saveRecentSearch(request);

      expect(saved.id, 'recent_project');
      expect(saved.payload, isA<ProjectRecentSearchPayload>());
    });

    test('IT-009 deletes recent search by opaque id and accepts 204', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onDelete(
        '/v1/search/recent/recent_123',
        (server) => server.reply(204, null),
      );

      await SearchApiClient(dio).deleteRecentSearch('recent_123');
    });
  });
}
