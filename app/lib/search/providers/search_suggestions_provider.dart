import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'search_suggestions_provider.g.dart';

@riverpod
Future<SearchSuggestions> searchSuggestions(
  Ref ref,
  SearchSuggestionQuery query,
) {
  final q = query.q.trim();
  if (q.isEmpty) {
    return Future.value(SearchSuggestions.empty());
  }
  return ref
      .watch(searchRepositoryProvider)
      .searchSuggestions(
        q: q,
        types: query.types.isEmpty ? null : query.types,
        profileLimit: query.profileLimit,
        hashtagLimit: query.hashtagLimit,
      );
}
