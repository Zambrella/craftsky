import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_like_post_provider.g.dart';

@riverpod
class ToggleLikePost extends _$ToggleLikePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> toggle({required Post post}) async {
    final next = post.copyWith(
      viewerHasLiked: !post.viewerHasLiked,
      likeCount: post.viewerHasLiked
          ? (post.likeCount > 0 ? post.likeCount - 1 : 0)
          : post.likeCount + 1,
    );

    updateLiveUserPostCaches(ref, next);
    updateLiveUserProjectCaches(ref, next);
    updateLiveUserCommentCaches(ref, next);
    updateLiveTimelineCache(ref, next);
    state = AsyncData(next);

    try {
      final repo = ref.read(postRepositoryProvider);
      if (next.viewerHasLiked) {
        await repo.like(post.author.did, post.rkey);
      } else {
        await repo.unlike(post.author.did, post.rkey);
      }
    } on Object catch (error, stackTrace) {
      if (!ref.mounted) return;
      updateLiveUserPostCaches(ref, post);
      updateLiveUserProjectCaches(ref, post);
      updateLiveUserCommentCaches(ref, post);
      updateLiveTimelineCache(ref, post);
      state = AsyncData(post);
      state = AsyncError<Post?>(error, stackTrace);
    }
  }

  void reset() => state = const AsyncData(null);
}
