import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_projects_provider.g.dart';

const userProjectsPageLimit = 10;

@riverpod
class UserProjects extends _$UserProjects {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserProjectsState> build(Did did) async {
    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(items: applyLoadedPostEdits(ref, current.items)),
      );
    });
    ref.watch(activeContentLanguagePolicyProvider);
    final accountOperation = ActiveAccountOperation.capture(ref);
    final repo = ref.watch(postRepositoryProvider);
    final page = await repo.listProjectsByAuthor(
      did,
      limit: userProjectsPageLimit,
    );
    accountOperation.requireCurrent(discarded: const ApiCanceled());
    return UserProjectsState(
      items: applyPostRecordListOverlays(
        ref,
        page.items,
        listKey: ('author-projects', did),
        includes: (post) =>
            post.author.did == did &&
            post.reply == null &&
            post.project != null,
      ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
      cursor: page.cursor,
      pinnedPostUri: page.pinnedPostUri,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<UserProjectsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      late final PostPage page;
      try {
        page = await repo.listProjectsByAuthor(
          did,
          cursor: current.cursor,
          limit: userProjectsPageLimit,
        );
      } on ApiBadRequest catch (error) {
        if (error.code != 'invalid_cursor') rethrow;
        accountOperation.requireCurrent(discarded: const ApiCanceled());
        final restarted = await repo.listProjectsByAuthor(
          did,
          limit: userProjectsPageLimit,
        );
        accountOperation.requireCurrent(discarded: const ApiCanceled());
        return UserProjectsState(
          items: applyPostRecordListOverlays(
            ref,
            restarted.items,
            listKey: ('author-projects', did),
            includes: (post) =>
                post.author.did == did &&
                post.reply == null &&
                post.project != null,
          ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
          cursor: restarted.cursor,
          pinnedPostUri: restarted.pinnedPostUri,
        );
      }
      accountOperation.requireCurrent(discarded: const ApiCanceled());
      observePostListMembership(ref, page.items, ('author-projects', did));
      for (final post in page.items) {
        applyPostRecordOverlay(ref, post);
      }
      return UserProjectsState(
        items: applyPostRecordListOverlays(
          ref,
          [...current.items, ...page.items],
          listKey: ('author-projects', did),
          reconcile: false,
          includes: (post) =>
              post.author.did == did &&
              post.reply == null &&
              post.project != null,
        ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
        cursor: page.cursor,
        pinnedPostUri: current.pinnedPostUri,
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
