import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'search_suggestions.mapper.dart';

enum SearchSuggestionType {
  profiles,
  hashtags;

  String get wireValue => name;
}

@MappableClass()
class SearchSuggestions with SearchSuggestionsMappable {
  const SearchSuggestions({required this.profiles, required this.hashtags});

  factory SearchSuggestions.empty() => const SearchSuggestions(
    profiles: SearchSuggestionProfileSection(items: [], hasMore: false),
    hashtags: SearchSuggestionHashtagSection(items: [], hasMore: false),
  );

  final SearchSuggestionProfileSection profiles;
  final SearchSuggestionHashtagSection hashtags;
}

@MappableClass()
class SearchSuggestionProfileSection
    with SearchSuggestionProfileSectionMappable {
  const SearchSuggestionProfileSection({
    required this.items,
    required this.hasMore,
  });

  final List<ProfileSearchResult> items;
  final bool hasMore;
}

@MappableClass()
class SearchSuggestionHashtagSection
    with SearchSuggestionHashtagSectionMappable {
  const SearchSuggestionHashtagSection({
    required this.items,
    required this.hasMore,
  });

  final List<HashtagSearchResult> items;
  final bool hasMore;
}
