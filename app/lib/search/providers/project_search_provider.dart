import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_search_provider.g.dart';

@riverpod
class ProjectSearch extends _$ProjectSearch {
  @override
  Future<SearchPostResultsState> build(ProjectSearchQuery query) async {
    final page = await ref
        .watch(searchRepositoryProvider)
        .searchProjects(
          q: query.q,
          sort: query.sort,
          filters: query.filters,
          limit: searchResultsPageLimit,
        );
    return SearchPostResultsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    state = const AsyncLoading<SearchPostResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchProjects(
            q: query.q,
            sort: query.sort,
            filters: query.filters,
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
