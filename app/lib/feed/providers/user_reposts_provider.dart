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
    ref.watch(activeContentLanguagePolicyProvider);
    final page = await ref
        .watch(postRepositoryProvider)
        .listRepostsByAuthor(
          did,
          limit: userRepostsPageLimit,
        );
    return _stateFor(page);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final ownership = captureActiveAccountOperation(ref);

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
        return _stateFor(
          await repo.listRepostsByAuthor(did, limit: userRepostsPageLimit),
        );
      }
      return _stateFor(
        PostPage(items: [...current.items, ...page.items], cursor: page.cursor),
      );
    });

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = next;
  }

  UserPostsState _stateFor(PostPage page) {
    final posts = applyPostRecordListOverlays(
      ref,
      page.items,
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
