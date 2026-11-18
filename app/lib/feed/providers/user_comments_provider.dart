import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:craftsky_app/shared/browsing/content_read_guard.dart';
import 'package:craftsky_app/shared/widgets/retained_page.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'user_comments_provider.g.dart';

const userCommentsPageLimit = 10;

/// Cursor-accumulating authored comments/replies list, keyed by DID.
@riverpod
class UserComments extends _$UserComments {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<UserPostsState> build(Did did, {BrowsingContext? audience}) async {
    final read = await ContentReadGuard.captureWhenReady(ref);
    read.requireCurrent(discarded: const ApiCanceled());
    if (audience != null && read.context != audience) {
      throw const ContentReadDiscarded();
    }

    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(items: applyLoadedPostEdits(ref, current.items)),
      );
    });
    if (!ref.watch(isGuestBrowsingProvider)) {
      ref.watch(activeContentLanguagePolicyProvider);
    }
    ref.watch(postRepositoryProvider);
    final page = await _reload(state.value?.items.length ?? 0);
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

  Future<PostPage> _reload(int count) async {
    final operation = ContentReadGuard.capture(ref);
    final repo = ref.read(postRepositoryProvider);
    String? pinned;
    final page = await reloadLoadedRange<Post, String>(
      minimumItems: count,
      key: (post) => post.uri,
      fetch: (cursor) async {
        final page = await repo.listCommentsByAuthor(
          did,
          cursor: cursor,
          limit: userCommentsPageLimit,
        );
        operation.requireCurrent(discarded: const ApiCanceled());
        if (cursor == null) pinned = page.pinnedPostUri;
        return (items: page.items, cursor: page.cursor);
      },
    );
    return PostPage(
      items: page.items,
      cursor: page.cursor,
      pinnedPostUri: pinned,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ContentReadGuard.capture(ref);

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
