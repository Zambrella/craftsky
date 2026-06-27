import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'hashtag_search_provider.g.dart';

const searchResultsPageLimit = 25;

@riverpod
class HashtagSearch extends _$HashtagSearch {
  @override
  Future<SearchPostResultsState> build(HashtagSearchQuery query) async {
    final page = await ref
        .watch(searchRepositoryProvider)
        .searchHashtagPosts(
          query.tag,
          sort: query.sort,
          limit: searchResultsPageLimit,
        );
    return SearchPostResultsState(
      items: page.items,
      cursor: page.cursor,
      hashtag: page.hashtag,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    state = const AsyncLoading<SearchPostResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchHashtagPosts(
            query.tag,
            sort: query.sort,
            limit: searchResultsPageLimit,
            cursor: current.cursor,
          );
      return SearchPostResultsState(
        items: appendUniquePosts(current.items, page.items),
        cursor: page.cursor,
        hashtag: page.hashtag ?? current.hashtag,
      );
    });
    if (!ref.mounted) return;
    state = next;
  }
}
