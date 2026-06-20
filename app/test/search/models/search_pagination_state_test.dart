import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 stores opaque cursors exactly and derives hasMore', () {
    const opaqueCursor = 'opaque:abc/+/=';
    const page = SearchPostResultsState(items: [], cursor: opaqueCursor);

    expect(page.cursor, opaqueCursor);
    expect(page.hasMore, isTrue);
    expect(const SearchPostResultsState(items: []).hasMore, isFalse);
  });
}
