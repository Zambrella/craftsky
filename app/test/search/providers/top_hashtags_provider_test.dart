import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/search/providers/top_hashtags_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test('IT-014 top hashtags provider fetches grouped metadata', () async {
    List<String>? seenCraftTypes;
    int? seenLimit;
    final fake = FakeSearchRepository(
      onTopHashtags: ({craftTypes, limit}) async {
        seenCraftTypes = craftTypes;
        seenLimit = limit;
        return const TopHashtagsResponse(
          groups: [
            TopHashtagGroup(
              craftType: 'knitting',
              items: [TopHashtagItem(tag: 'sockkal', count: 12)],
            ),
          ],
        );
      },
    );
    final container = ProviderContainer.test(
      overrides: [searchRepositoryProvider.overrideWithValue(fake)],
    );

    final response = await container.read(
      topHashtagsProvider(
        const TopHashtagsQuery(craftTypes: ['knitting'], limit: 10),
      ).future,
    );

    expect(seenCraftTypes, ['knitting']);
    expect(seenLimit, 10);
    expect(response.groups.single.items.single.tag, 'sockkal');
  });
}
