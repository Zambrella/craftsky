import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:craftsky_app/projects/providers/project_repository_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'project_feed_provider.g.dart';

const projectFeedPageLimit = 25;

@riverpod
class ProjectFeed extends _$ProjectFeed {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserProjectsState> build(ProjectBrowseQuery query) async {
    final loadedCount = state.value?.items.length ?? 0;
    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(items: applyLoadedPostEdits(ref, current.items)),
      );
    });
    ref.watch(activeContentLanguagePolicyProvider);
    final accountOperation = ActiveAccountOperation.capture(ref);
    final repository = ref.watch(projectRepositoryProvider);
    var page = await repository.listProjects(
      query: query,
      limit: projectFeedPageLimit,
    );
    accountOperation.requireCurrent(discarded: const ApiCanceled());
    var items = _visibleProjects(ref, page.items);
    // Interaction reconciliation must not shrink a long, mounted list back to
    // its first page and clamp the user's scroll position.
    final visitedCursors = <String>{};
    while (items.length < loadedCount &&
        page.cursor != null &&
        visitedCursors.add(page.cursor!)) {
      page = await repository.listProjects(
        query: query,
        limit: projectFeedPageLimit,
        cursor: page.cursor,
      );
      accountOperation.requireCurrent(discarded: const ApiCanceled());
      items = appendUniquePosts(items, _visibleProjects(ref, page.items));
    }
    return UserProjectsState(
      items: items,
      cursor: page.cursor,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<UserProjectsState>();

    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(projectRepositoryProvider)
          .listProjects(
            query: query,
            limit: projectFeedPageLimit,
            cursor: current.cursor,
          );
      accountOperation.requireCurrent(discarded: const ApiCanceled());
      return UserProjectsState(
        items: appendUniquePosts(
          current.items,
          _visibleProjects(ref, page.items),
        ),
        cursor: page.cursor,
      );
    });

    accountOperation.publish(
      () => state = next.whenData(
        (value) =>
            value.copyWith(items: applyLoadedPostEdits(ref, value.items)),
      ),
    );
  }
}

List<Post> _visibleProjects(Ref ref, List<Post> posts) => [
  for (final post in posts)
    if (applyPostRecordOverlay(ref, post) case final visible?)
      applyPostInteractionOverlays(ref, visible),
];
