import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('UT-009 maps suggestions, hashtag pages, and refined recents', () {
    final suggestions = SearchSuggestionsMapper.fromMap({
      'profiles': {
        'items': [
          {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'displayName': 'Alice',
            'description': 'Knitter',
            'avatar': 'https://example.com/a.jpg',
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
    });

    expect(suggestions.profiles.hasMore, isTrue);
    expect(suggestions.profiles.items.single.crafts, [
      'social.craftsky.feed.defs#knitting',
    ]);
    expect(suggestions.hashtags.items.single.tag, 'sockkal');

    final hashtagPage = HashtagSearchPageMapper.fromMap({
      'items': [
        {'tag': 'sockkal', 'postsLast28Days': 12},
      ],
      'cursor': 'opaque:hashtags',
    });
    expect(hashtagPage.cursor, 'opaque:hashtags');
    expect(hashtagPage.items.single.postsLast28Days, 12);

    final recentPage = RecentSearchPage.fromMap({
      'items': [
        {
          'id': 'recent_query',
          'type': 'query',
          'displayLabel': 'alpaca socks',
          'payload': {'q': 'alpaca socks'},
          'updatedAt': '2026-06-20T10:00:00Z',
        },
        {
          'id': 'recent_hash',
          'type': 'hashtag',
          'displayLabel': '#SockKAL',
          'payload': {'tag': 'sockkal'},
          'updatedAt': '2026-06-20T11:00:00Z',
        },
        {
          'id': 'recent_profile',
          'type': 'profile',
          'displayLabel': 'Alice',
          'payload': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'displayName': 'Alice',
            'avatar': 'https://example.com/a.jpg',
          },
          'updatedAt': '2026-06-20T12:00:00Z',
        },
      ],
    });

    expect(recentPage.items[0].payload, isA<QueryRecentSearchPayload>());
    expect(recentPage.items[1].payload.toMap(), {'tag': 'sockkal'});
    expect(recentPage.items[2].payload.toMap(), {
      'did': 'did:plc:alice',
      'handle': 'alice.craftsky.social',
      'displayName': 'Alice',
      'avatar': 'https://example.com/a.jpg',
    });
    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.query,
        displayLabel: 'alpaca socks',
        payload: QueryRecentSearchPayload(q: 'alpaca socks'),
      ).toMap(),
      {
        'type': 'query',
        'displayLabel': 'alpaca socks',
        'payload': {'q': 'alpaca socks'},
      },
    );
  });
}
