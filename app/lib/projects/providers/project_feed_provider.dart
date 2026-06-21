import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_feed_provider.g.dart';

const projectFeedPageLimit = 25;

@riverpod
class ProjectFeed extends _$ProjectFeed {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserProjectsState> build(ProjectBrowseQuery query) async {
    final page = await ref
        .watch(projectRepositoryProvider)
        .listProjects(
          query: query,
          limit: projectFeedPageLimit,
        );
    return UserProjectsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;

    state = const AsyncLoading<UserProjectsState>();

    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(projectRepositoryProvider)
          .listProjects(
            query: query,
            limit: projectFeedPageLimit,
            cursor: current.cursor,
          );
      return UserProjectsState(
        items: appendUniquePosts(current.items, page.items),
        cursor: page.cursor,
      );
    });

    if (!ref.mounted) return;
    state = next;
  }
}
