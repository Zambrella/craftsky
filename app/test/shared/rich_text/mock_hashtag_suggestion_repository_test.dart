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
          HashtagSuggestion(tag: 'SockKAL', postCount: 128),
          HashtagSuggestion(tag: 'sockmending', postCount: 12),
          HashtagSuggestion(tag: 'Lace', postCount: 7),
        ],
      );

      final results = await repository.searchHashtags('sock');

      expect(results.map((hashtag) => hashtag.tag), ['SockKAL', 'sockmending']);
      expect(results.map((hashtag) => hashtag.postCount), [128, 12]);
    });
  });
}
