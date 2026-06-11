import 'package:craftsky_app/feed/models/post.dart';
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

    if (!ref.mounted) return;
    state = next;
  }

  void prepend(Post post) {
    final current = state.value;
    if (current == null) return;
    if (current.items.any((item) => item.uri == post.uri)) return;
    state = AsyncData(current.copyWith(items: [post, ...current.items]));
  }

  void removeByRkey(String rkey) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: current.items.where((post) => post.rkey != rkey).toList(),
      ),
    );
  }

  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        items: [
          for (final item in current.items)
            if (item.uri == post.uri || item.rkey == post.rkey) post else item,
        ],
      ),
    );
  }
}

void prependLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in <String>{post.author.did, post.author.handle}) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).prepend(post);
    }
  }
}

void updateLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in <String>{post.author.did, post.author.handle}) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).replace(post);
    }
  }
}

void removeFromLiveUserProjectCaches(Ref ref, Post post) {
  if (post.project == null) return;
  for (final id in <String>{post.author.did, post.author.handle}) {
    if (ref.exists(userProjectsProvider(id))) {
      ref.read(userProjectsProvider(id).notifier).removeByRkey(post.rkey);
    }
  }
}
