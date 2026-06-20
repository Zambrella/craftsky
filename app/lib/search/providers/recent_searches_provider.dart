import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'recent_searches_provider.g.dart';

@riverpod
class RecentSearches extends _$RecentSearches {
  @override
  Future<RecentSearchPage> build() =>
      ref.watch(searchRepositoryProvider).listRecentSearches();

  Future<RecentSearchItem> save(SaveRecentSearchRequest request) async {
    final saved = await ref
        .read(searchRepositoryProvider)
        .saveRecentSearch(request);
    ref.invalidateSelf();
    await future;
    return saved;
  }

  Future<void> delete(String id) async {
    await ref.read(searchRepositoryProvider).deleteRecentSearch(id);
    ref.invalidateSelf();
    await future;
  }
}
