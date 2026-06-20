import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-001 exposes only supported AppView result sort wire values', () {
    expect(SearchSort.values, [SearchSort.chronological, SearchSort.popular]);
    expect(SearchSort.chronological.wireValue, 'chronological');
    expect(SearchSort.popular.wireValue, 'popular');
  });

  test('UT-001 profile search query has no sort field', () {
    const query = ProfileSearchQuery(q: 'alice');

    expect(query.q, 'alice');
    expect(query.toString(), isNot(contains('sort')));
  });
}
