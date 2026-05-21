import 'dart:async';

import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'create_post_provider.g.dart';

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// On success, prepends the synthetic post into any live
/// `userPostsProvider` family entries keyed by either the author's
/// handle or DID — sidestepping the AppView's read-after-write window
/// (where a refetch could miss the just-created row until the firehose
/// indexer catches up). `ref.exists` guards against accidentally
/// instantiating a non-live family entry, which would race a fresh
/// `build` against our prepend.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.
@riverpod
class CreatePost extends _$CreatePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> create({
    required String text,
    PostReply? reply,
    List<CreatePostImage>? images,
  }) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      final created = await repo.create(
        text: text,
        reply: reply,
        images: images,
      );
      final post = reply != null && created.reply == null
          ? created.copyWith(reply: reply)
          : created;
      if (!ref.mounted) return null;

      if (reply == null) {
        for (final id in <String>{post.author.handle, post.author.did}) {
          if (ref.exists(userPostsProvider(id))) {
            ref.read(userPostsProvider(id).notifier).prepend(post);
          }
        }
      } else {
        updateLiveUserCommentCaches(ref, post);
      }

      return post;
    });
  }

  /// Resets the notifier to its idle state. Call after consuming a
  /// success/failure transition so a re-entry doesn't see prior result.
  void reset() => state = const AsyncData(null);
}
