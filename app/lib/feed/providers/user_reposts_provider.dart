import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
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

part 'user_reposts_provider.g.dart';

const userRepostsPageLimit = 10;

/// Cursor-accumulating active repost subjects, keyed by the reposter DID.
@riverpod
class UserReposts extends _$UserReposts {
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
    return _stateFor(page);
  }

  Future<PostPage> _reload(int count) async {
    final operation = ContentReadGuard.capture(ref);
    final repo = ref.read(postRepositoryProvider);
    final page = await reloadLoadedRange<Post, String>(
      minimumItems: count,
      key: (post) => post.uri,
      fetch: (cursor) async {
        final page = await repo.listRepostsByAuthor(
          did,
          cursor: cursor,
          limit: userRepostsPageLimit,
        );
        operation.requireCurrent(discarded: const ApiCanceled());
        return (items: page.items, cursor: page.cursor);
      },
    );
    return PostPage(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ContentReadGuard.capture(ref);

    state = const AsyncLoading<UserPostsState>();
    final next = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      late final PostPage page;
      try {
        page = await repo.listRepostsByAuthor(
          did,
          cursor: current.cursor,
          limit: userRepostsPageLimit,
        );
      } on ApiBadRequest catch (error) {
        if (error.code != 'invalid_cursor') rethrow;
        accountOperation.requireCurrent(discarded: const ApiCanceled());
        final restarted = await _reload(current.items.length);
        accountOperation.requireCurrent(discarded: const ApiCanceled());
        return _stateFor(restarted);
      }
      accountOperation.requireCurrent(discarded: const ApiCanceled());
      for (final post in page.items) {
        applyPostRecordOverlay(ref, post);
      }
      return _stateFor(
        PostPage(items: [...current.items, ...page.items], cursor: page.cursor),
        reconcile: false,
      );
    });

    accountOperation.publish(
      () => state = next.whenData(
        (value) =>
            value.copyWith(items: applyLoadedPostEdits(ref, value.items)),
      ),
    );
  }

  UserPostsState _stateFor(PostPage page, {bool reconcile = true}) {
    final posts = applyPostRecordListOverlays(
      ref,
      page.items,
      reconcile: reconcile,
      includes: (_) => false,
    ).map((post) => applyPostInteractionOverlays(ref, post));
    final isOwnProfile =
        captureActiveAccountOperation(ref)?.session.account.did == did;
    return UserPostsState(
      items: isOwnProfile
          ? posts.where((post) => post.viewerHasReposted).toList()
          : posts.toList(),
      cursor: page.cursor,
    );
  }
}
