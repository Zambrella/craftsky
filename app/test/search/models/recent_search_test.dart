import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-005 save payloads serialize supported recent-search types', () {
    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.query,
        displayLabel: 'Alpaca socks',
        payload: QueryRecentSearchPayload(q: 'alpaca socks'),
      ).toMap(),
      {
        'type': 'query',
        'displayLabel': 'Alpaca socks',
        'payload': {'q': 'alpaca socks'},
      },
    );

    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.hashtag,
        displayLabel: '#SockKAL',
        payload: HashtagRecentSearchPayload(tag: 'sockkal'),
      ).toMap(),
      {
        'type': 'hashtag',
        'displayLabel': '#SockKAL',
        'payload': {'tag': 'sockkal'},
      },
    );

    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.profile,
        displayLabel: 'Alice',
        payload: ProfileRecentSearchPayload(
          did: 'did:plc:alice',
          handle: 'alice.craftsky.social',
          displayName: 'Alice',
        ),
      ).toMap(),
      {
        'type': 'profile',
        'displayLabel': 'Alice',
        'payload': {
          'did': 'did:plc:alice',
          'handle': 'alice.craftsky.social',
          'displayName': 'Alice',
        },
      },
    );

    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.post,
        displayLabel: 'Alpaca posts',
        payload: PostRecentSearchPayload(
          q: 'alpaca',
          sort: SearchSort.popular,
        ),
      ).toMap(),
      {
        'type': 'post',
        'displayLabel': 'Alpaca posts',
        'payload': {'q': 'alpaca', 'sort': 'popular'},
      },
    );

    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.project,
        displayLabel: 'Cardigan',
        payload: ProjectRecentSearchPayload(
          q: 'cardigan',
          sort: SearchSort.popular,
          filters: ProjectSearchFilters(craftType: ['knitting']),
        ),
      ).toMap(),
      {
        'type': 'project',
        'displayLabel': 'Cardigan',
        'payload': {
          'q': 'cardigan',
          'sort': 'popular',
          'filters': {
            'craftType': ['knitting'],
          },
        },
      },
    );
  });

  test('UT-005 recent items deserialize typed rerunnable payloads', () {
    final page = RecentSearchPage.fromMap({
      'items': [
        {
          'id': 'recent_0',
          'type': 'query',
          'displayLabel': 'Alpaca socks',
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
          'displayLabel': 'Alice',
          'payload': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
          },
          'updatedAt': '2026-06-20T11:00:00Z',
        },
        {
          'id': 'recent_3',
          'type': 'post',
          'displayLabel': 'Alpaca posts',
          'payload': {'q': 'alpaca', 'sort': 'popular'},
          'updatedAt': '2026-06-20T12:00:00Z',
        },
        {
          'id': 'recent_4',
          'type': 'project',
          'displayLabel': 'Cardigan',
          'payload': {
            'q': 'cardigan',
            'sort': 'chronological',
            'filters': {
              'craftType': ['knitting'],
              'material': ['wool'],
            },
          },
          'updatedAt': '2026-06-20T13:00:00Z',
        },
      ],
    });

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
    expect(
      page.items[4].payload.toMap(),
      {
        'q': 'cardigan',
        'sort': 'chronological',
        'filters': {
          'craftType': ['knitting'],
          'material': ['wool'],
        },
      },
    );
    expect(
      page.items.first.updatedAt.toUtc().toIso8601String(),
      '2026-06-20T09:00:00.000Z',
    );
  });
}
