import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-004 decodes top hashtag groups and counts', () {
    final response = TopHashtagsResponseMapper.fromMap({
      'groups': [
        {
          'craftType': 'knitting',
          'items': [
            {'tag': 'sockkal', 'count': 12},
          ],
        },
        {'craftType': 'crochet', 'items': <Map<String, dynamic>>[]},
      ],
    });

    expect(response.groups, hasLength(2));
    expect(response.groups.first.items.single.tag, 'sockkal');
    expect(response.groups.first.items.single.count, 12);
    expect(response.groups.last.items, isEmpty);
  });
}
