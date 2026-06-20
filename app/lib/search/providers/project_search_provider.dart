import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_search_provider.g.dart';

@riverpod
class ProjectSearch extends _$ProjectSearch {
  late ProjectSearchQuery _query;

  @override
  Future<SearchPostResultsState> build(ProjectSearchQuery query) async {
    _query = query;
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
    final current = state.value;
    if (current == null || !current.hasMore || state.isLoading) return;
    state = const AsyncLoading<SearchPostResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchProjects(
            q: _query.q,
            sort: _query.sort,
            filters: _query.filters,
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
