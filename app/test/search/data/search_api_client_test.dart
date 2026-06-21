import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
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

  group('SearchApiClient suggestions and hashtags tab', () {
    test('IT-012 fetches unified suggestions with bounded sections', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/suggestions',
        (server) => server.reply(200, {
          'profiles': {
            'items': [
              {
                'did': 'did:plc:alice',
                'handle': 'alice.craftsky.social',
                'isCraftskyProfile': true,
                'viewerIsFollowing': true,
                'crafts': ['social.craftsky.feed.defs#knitting'],
              },
            ],
            'hasMore': true,
          },
          'hashtags': {
            'items': [
              {'tag': 'sockkal', 'postsLast28Days': 12},
            ],
            'hasMore': false,
          },
        }),
        queryParameters: {
          'q': 'sock',
          'types': 'profiles,hashtags',
          'profileLimit': '2',
          'hashtagLimit': '2',
        },
      );

      final suggestions = await SearchApiClient(dio).searchSuggestions(
        q: 'sock',
        types: const [
          SearchSuggestionType.profiles,
          SearchSuggestionType.hashtags,
        ],
        profileLimit: 2,
        hashtagLimit: 2,
      );

      expect(suggestions.profiles.hasMore, isTrue);
      expect(suggestions.profiles.items.single.crafts, [
        'social.craftsky.feed.defs#knitting',
      ]);
      expect(suggestions.hashtags.items.single.tag, 'sockkal');
    });

    test('IT-012 fetches committed hashtag-query results', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/hashtags',
        (server) => server.reply(200, {
          'items': [
            {'tag': 'sock', 'postsLast28Days': 4},
          ],
          'cursor': 'opaque:hashtags',
        }),
        queryParameters: {
          'q': 'sock',
          'limit': '10',
          'cursor': 'opaque:start',
        },
      );

      final page = await SearchApiClient(
        dio,
      ).searchHashtags(q: 'sock', limit: 10, cursor: 'opaque:start');

      expect(page.cursor, 'opaque:hashtags');
      expect(page.items.single.postsLast28Days, 4);
    });
  });

  group('SearchApiClient post/project search', () {
    test(
      'IT-004 sends text-only post search q/limit/cursor and decodes Post page',
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
            'limit': '10',
            'cursor': 'opaque:start',
          },
        );

        final page = await SearchApiClient(dio).searchPosts(
          q: 'alpaca',
          limit: 10,
          cursor: 'opaque:start',
        );

        expect(page.cursor, 'opaque:posts');
        expect(page.items.single, isA<Post>());
      },
    );

    test(
      'IT-005 sends text-only project search without browse filters or sort',
      () async {
        final dio = buildDio();
        DioAdapter(dio: dio).onGet(
          '/v1/search/projects',
          (server) => server.reply(200, {
            'items': [samplePost(rkey: 'project')],
          }),
          queryParameters: {
            'q': 'cardigan',
            'limit': '25',
            'cursor': 'opaque:projects',
          },
        );
        final page = await SearchApiClient(dio).searchProjects(
          q: 'cardigan',
          limit: 25,
          cursor: 'opaque:projects',
        );

        expect(page.items.single.rkey.toString(), 'project');
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

    test('IT-007 lists recent searches with all typed payloads', () async {
      final dio = buildDio();
      DioAdapter(dio: dio).onGet(
        '/v1/search/recent',
        (server) => server.reply(200, {
          'items': [
            {
              'id': 'recent_0',
              'type': 'query',
              'displayLabel': 'alpaca socks',
              'payload': {'q': 'alpaca socks'},
              'updatedAt': '2026-06-20T09:00:00Z',
            },
            {
              'id': 'recent_1',
              'type': 'hashtag',
              'displayLabel': '#SockKAL',
              'payload': {'tag': 'sockkal'},
              'updatedAt': '2026-06-20T10:00:00Z',
            },
            {
              'id': 'recent_2',
              'type': 'profile',
              'displayLabel': 'alice',
              'payload': {
                'did': 'did:plc:alice',
                'handle': 'alice.craftsky.social',
              },
              'updatedAt': '2026-06-20T11:00:00Z',
            },
            {
              'id': 'recent_3',
              'type': 'post',
              'displayLabel': 'alpaca posts',
              'payload': {'q': 'alpaca', 'sort': 'chronological'},
              'updatedAt': '2026-06-20T12:00:00Z',
            },
            {
              'id': 'recent_4',
              'type': 'project',
              'displayLabel': 'cardigan projects',
              'payload': {
                'q': 'cardigan',
                'sort': 'popular',
                'filters': {
                  'craftType': ['knitting'],
                  'projectTag': ['gift'],
                },
              },
              'updatedAt': '2026-06-20T13:00:00Z',
            },
          ],
        }),
      );

      final page = await SearchApiClient(dio).listRecentSearches();

      expect(page.items.map((item) => item.id), [
        'recent_0',
        'recent_1',
        'recent_2',
        'recent_3',
        'recent_4',
      ]);
      expect(page.items[0].payload, isA<QueryRecentSearchPayload>());
      expect(page.items[1].payload, isA<HashtagRecentSearchPayload>());
      expect(page.items[2].payload, isA<ProfileRecentSearchPayload>());
      expect(page.items[3].payload, isA<PostRecentSearchPayload>());
      expect(page.items[4].payload, isA<ProjectRecentSearchPayload>());
      expect(
        page.items[1].payload.toMap(),
        {'tag': 'sockkal'},
      );
      expect(page.items[4].payload.toMap(), {
        'q': 'cardigan',
        'sort': 'popular',
        'filters': {
          'craftType': ['knitting'],
          'projectTag': ['gift'],
        },
      });
    });

    test('IT-008 saves future Search recent payloads', () async {
      final dio = buildDio();
      DioAdapter(dio: dio)
        ..onPost(
          '/v1/search/recent',
          (server) => server.reply(201, {
            'id': 'recent_query',
            'type': 'query',
            'displayLabel': 'Alpaca socks',
            'payload': {'q': 'alpaca socks'},
            'updatedAt': '2026-06-20T11:59:00Z',
          }),
          data: {
            'type': 'query',
            'displayLabel': 'Alpaca socks',
            'payload': {'q': 'alpaca socks'},
          },
        )
        ..onPost(
          '/v1/search/recent',
          (server) => server.reply(201, {
            'id': 'recent_hashtag',
            'type': 'hashtag',
            'displayLabel': '#SockKAL',
            'payload': {'tag': 'sockkal'},
            'updatedAt': '2026-06-20T12:00:00Z',
          }),
          data: {
            'type': 'hashtag',
            'displayLabel': '#SockKAL',
            'payload': {'tag': 'sockkal'},
          },
        )
        ..onPost(
          '/v1/search/recent',
          (server) => server.reply(201, {
            'id': 'recent_profile',
            'type': 'profile',
            'displayLabel': 'Alice',
            'payload': {
              'did': 'did:plc:alice',
              'handle': 'alice.craftsky.social',
              'displayName': 'Alice',
            },
            'updatedAt': '2026-06-20T12:01:00Z',
          }),
          data: {
            'type': 'profile',
            'displayLabel': 'Alice',
            'payload': {
              'did': 'did:plc:alice',
              'handle': 'alice.craftsky.social',
              'displayName': 'Alice',
            },
          },
        );

      final client = SearchApiClient(dio);
      final saved = [
        await client.saveRecentSearch(
          const SaveRecentSearchRequest(
            type: RecentSearchType.query,
            displayLabel: 'Alpaca socks',
            payload: QueryRecentSearchPayload(q: 'alpaca socks'),
          ),
        ),
        await client.saveRecentSearch(
          const SaveRecentSearchRequest(
            type: RecentSearchType.hashtag,
            displayLabel: '#SockKAL',
            payload: HashtagRecentSearchPayload(tag: 'sockkal'),
          ),
        ),
        await client.saveRecentSearch(
          const SaveRecentSearchRequest(
            type: RecentSearchType.profile,
            displayLabel: 'Alice',
            payload: ProfileRecentSearchPayload(
              did: 'did:plc:alice',
              handle: 'alice.craftsky.social',
              displayName: 'Alice',
            ),
          ),
        ),
      ];

      expect(saved.map((item) => item.id), [
        'recent_query',
        'recent_hashtag',
        'recent_profile',
      ]);
      expect(saved[0].payload, isA<QueryRecentSearchPayload>());
      expect(saved[1].payload, isA<HashtagRecentSearchPayload>());
      expect(saved[2].payload, isA<ProfileRecentSearchPayload>());
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
