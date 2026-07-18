import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/author_post_cache.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_projects_provider.g.dart';

const userProjectsPageLimit = 10;

@riverpod
class UserProjects extends _$UserProjects {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserProjectsState> build(String handleOrDid) async {
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listProjectsByAuthor(
      handleOrDid,
      limit: userProjectsPageLimit,
    );
    return UserProjectsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final ownership = captureActiveAccountOperation(ref);

    state = const AsyncLoading<UserProjectsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listProjectsByAuthor(
        handleOrDid,
        cursor: current.cursor,
        limit: userProjectsPageLimit,
      );
      return UserProjectsState(
        items: [...current.items, ...page.items],
        cursor: page.cursor,
      );
    });

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = next;
  }

  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    final items = prependPostIfAbsent(current.items, post);
    if (identical(items, current.items)) return;
    state = AsyncData(current.copyWith(items: items));
  }

  void removeByRkey(String rkey) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(items: removePostByRkey(current.items, rkey)),
    );
  }

  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(items: replacePostByIdentity(current.items, post)),
    );
  }
}

void prependLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in authorPostCacheIds(post)) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).prepend(post);
    }
  }
}

void updateLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in authorPostCacheIds(post)) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).replace(post);
    }
  }
}

void removeFromLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in authorPostCacheIds(post)) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).removeByRkey(post.rkey);
    }
  }
}
