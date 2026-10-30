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

part 'user_reposts_provider.g.dart';

const userRepostsPageLimit = 10;

/// Cursor-accumulating active repost subjects, keyed by the reposter DID.
@riverpod
class UserReposts extends _$UserReposts {
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
    final page = await ref
        .watch(postRepositoryProvider)
        .listRepostsByAuthor(
          did,
          limit: userRepostsPageLimit,
        );
    accountOperation.requireCurrent(discarded: const ApiCanceled());
    return _stateFor(page);
  }

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
        page = await repo.listRepostsByAuthor(
          did,
          cursor: current.cursor,
          limit: userRepostsPageLimit,
        );
      } on ApiBadRequest catch (error) {
        if (error.code != 'invalid_cursor') rethrow;
        accountOperation.requireCurrent(discarded: const ApiCanceled());
        final restarted = await repo.listRepostsByAuthor(
          did,
          limit: userRepostsPageLimit,
        );
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
