import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'delete_post_provider.g.dart';

/// Standalone delete-a-post mutation notifier. Takes the full [Post]
/// because the cache update needs `did`, `handle`, and `rkey` to splice
/// the post out of any live family entries (lists may be keyed by
/// either form). The caller — UI deleting a post it's already
/// rendering — has the [Post] in hand.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
///
/// On success, removes the post from any live `userPostsProvider`
/// family entries keyed by either the author's handle or DID,
/// sidestepping the AppView's read-after-delete window (where a
/// refetch could still include the just-deleted row until the firehose
/// tombstone arrives).
@riverpod
class DeletePost extends _$DeletePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> delete({required Post post}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      await repo.delete(post.author.did, post.rkey);
      if (!ref.mounted) return null;

      for (final id in <String>{post.author.did, post.author.handle}) {
        final entry = userPostsProvider(id);
        if (ref.exists(entry)) {
          ref.read(entry.notifier).removeByRkey(post.rkey);
        }
      }

      return post;
    });
  }

  void reset() => state = const AsyncData(null);
}
