import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_comments_provider.g.dart';

const userCommentsPageLimit = 10;

/// Cursor-accumulating authored comments/replies list, keyed by DID.
@riverpod
class UserComments extends _$UserComments {
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
    final page = await repo.listCommentsByAuthor(
      did,
      limit: userCommentsPageLimit,
    );
    accountOperation.requireCurrent(discarded: const ApiCanceled());
    return UserPostsState(
      items: applyPostRecordListOverlays(
        ref,
        page.items,
        listKey: ('author-comments', did),
        includes: (post) => post.author.did == did && post.reply != null,
      ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
      cursor: page.cursor,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ActiveAccountOperation.capture(ref);

    state = const AsyncLoading<UserPostsState>();

    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final page = await repo.listCommentsByAuthor(
        did,
        cursor: current.cursor,
        limit: userCommentsPageLimit,
      );
      accountOperation.requireCurrent(discarded: const ApiCanceled());
      observePostListMembership(ref, page.items, ('author-comments', did));
      for (final post in page.items) {
        applyPostRecordOverlay(ref, post);
      }
      return UserPostsState(
        items: applyPostRecordListOverlays(
          ref,
          [...current.items, ...page.items],
          listKey: ('author-comments', did),
          reconcile: false,
          includes: (post) => post.author.did == did && post.reply != null,
        ).map((post) => applyPostInteractionOverlays(ref, post)).toList(),
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
