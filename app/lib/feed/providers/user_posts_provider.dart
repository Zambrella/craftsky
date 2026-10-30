import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_posts_provider.g.dart';

const userPostsPageLimit = 10;

/// Cursor-accumulating list-by-author provider, keyed by DID.
@riverpod
class UserPosts extends _$UserPosts {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserPostsState> build(Did did) async {
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
    final page = await repo.listByAuthor(
      did,
      limit: userPostsPageLimit,
    );
    if (!accountOperation.isCurrent) throw const ApiCanceled();
    return UserPostsState(
      items: applyPostRecordListOverlays(
        ref,
        page.items,
        listKey: ('author-posts', did),
        includes: (post) =>
            post.author.did == did &&
            post.reply == null &&
            post.project == null,
      ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
      cursor: page.cursor,
      pinnedPostUri: page.pinnedPostUri,
    );
  }

  /// Append-next-page. No-op when:
  ///   - data hasn't loaded yet (`state.value == null`),
  ///   - we've reached the end (`!hasMore`),
  ///   - or a `loadMore` is already in flight (`state.isLoading`).
  ///
  /// On success, appends items and advances cursor. Riverpod preserves
  /// previous data across loading/error transitions so retry can use the
  /// same cursor after a load-more failure.
  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<UserPostsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      late final PostPage page;
      try {
        page = await repo.listByAuthor(
          did,
          cursor: current.cursor,
          limit: userPostsPageLimit,
        );
      } on ApiBadRequest catch (error) {
        if (error.code != 'invalid_cursor') rethrow;
        if (!accountOperation.isCurrent) throw const ApiCanceled();
        final restarted = await repo.listByAuthor(
          did,
          limit: userPostsPageLimit,
        );
        if (!accountOperation.isCurrent) throw const ApiCanceled();
        return UserPostsState(
          items: applyPostRecordListOverlays(
            ref,
            restarted.items,
            listKey: ('author-posts', did),
            includes: (post) =>
                post.author.did == did &&
                post.reply == null &&
                post.project == null,
          ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
          cursor: restarted.cursor,
          pinnedPostUri: restarted.pinnedPostUri,
        );
      }
      if (!accountOperation.isCurrent) throw const ApiCanceled();
      observePostListMembership(ref, page.items, ('author-posts', did));
      for (final post in page.items) {
        applyPostRecordOverlay(ref, post);
      }
      return UserPostsState(
        items: applyPostRecordListOverlays(
          ref,
          [...current.items, ...page.items],
          listKey: ('author-posts', did),
          reconcile: false,
          includes: (post) =>
              post.author.did == did &&
              post.reply == null &&
              post.project == null,
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
