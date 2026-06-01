// Suggestion fixtures are clearer without forcing every constructor const.
// ignore_for_file: prefer_const_constructors

import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MockHashtagSuggestionRepository', () {
    test('UT-013 filters hashtags and exposes 28-day counts', () async {
      final repository = MockHashtagSuggestionRepository(
        hashtags: const [
          HashtagSuggestion(tag: 'SockKAL', postsLast28Days: 128),
          HashtagSuggestion(tag: 'sockmending', postsLast28Days: 12),
          HashtagSuggestion(tag: 'Lace', postsLast28Days: 7),
        ],
      );

      final results = await repository.searchHashtags('sock');

      expect(results.map((hashtag) => hashtag.tag), ['SockKAL', 'sockmending']);
      expect(results.map((hashtag) => hashtag.postsLast28Days), [128, 12]);
    });
  });
}
