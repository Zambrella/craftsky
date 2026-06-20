import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-005 save payloads serialize supported recent-search types', () {
    expect(
      const SaveRecentSearchRequest(
        type: RecentSearchType.profile,
        displayLabel: 'Alice',
        payload: ProfileRecentSearchPayload(q: 'alice'),
      ).toMap(),
      {
        'type': 'profile',
        'displayLabel': 'Alice',
        'payload': {'q': 'alice'},
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
    final item = RecentSearchItem.fromMap({
      'id': 'recent_1',
      'type': 'hashtag',
      'displayLabel': '#SockKAL',
      'payload': {'tag': 'sockkal', 'sort': 'chronological'},
      'updatedAt': '2026-06-20T10:00:00Z',
    });

    expect(item.id, 'recent_1');
    expect(item.payload, isA<HashtagRecentSearchPayload>());
    expect(
      item.updatedAt.toUtc().toIso8601String(),
      '2026-06-20T10:00:00.000Z',
    );
  });
}
