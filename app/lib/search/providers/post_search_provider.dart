import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_search_provider.g.dart';

@riverpod
class PostSearch extends _$PostSearch {
  late PostSearchQuery _query;

  @override
  Future<SearchPostResultsState> build(PostSearchQuery query) async {
    _query = query;
    final page = await ref
        .watch(searchRepositoryProvider)
        .searchPosts(
          q: query.q,
          sort: query.sort,
          limit: searchResultsPageLimit,
        );
    return SearchPostResultsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;
    state = const AsyncLoading<SearchPostResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchPosts(
            q: _query.q,
            sort: _query.sort,
            limit: searchResultsPageLimit,
            cursor: current.cursor,
          );
      return SearchPostResultsState(
        items: appendUniquePosts(current.items, page.items),
        cursor: page.cursor,
      );
    });
    if (!ref.mounted) return;
    state = next;
  }
}
