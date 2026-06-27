import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-009 initializeMappers registers search post page mapper', () {
    initializeMappers();

    final page = SearchPostPageMapper.fromMap({
      'items': <Map<String, dynamic>>[],
    });

    expect(page.items, isEmpty);
  });
}
