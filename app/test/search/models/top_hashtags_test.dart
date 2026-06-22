import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-004 decodes top hashtag groups and counts', () {
    final response = TopHashtagsResponseMapper.fromMap({
      'groups': [
        {
          'craftType': ProjectOptionCatalogs.defaultSupportedCraftTokens.first,
          'items': [
            {'tag': 'sockkal', 'count': 12},
          ],
        },
        {
          'craftType': ProjectOptionCatalogs.defaultSupportedCraftTokens[1],
          'items': <Map<String, dynamic>>[],
        },
      ],
    });

    expect(response.groups, hasLength(2));
    expect(
      response.groups.map((group) => group.craftType),
      ProjectOptionCatalogs.defaultSupportedCraftTokens.take(2),
    );
    expect(response.groups.first.items.single.tag, 'sockkal');
    expect(response.groups.first.items.single.count, 12);
    expect(response.groups.last.items, isEmpty);
  });
}
