import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

PdsMutationScope likePostMutationScope(Ref ref, Post post) {
  final lease =
      captureActiveAccountOperation(ref)?.session ??
      AccountSessionLease(
        account: AccountKey(post.author.did.toString()),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'like:${post.uri}');
}

PdsMutationScope repostPostMutationScope(Ref ref, Post post) {
  final lease =
      captureActiveAccountOperation(ref)?.session ??
      AccountSessionLease(
        account: AccountKey(post.author.did.toString()),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'repost:${post.uri}');
}

Post applyLikePostOverlay(Ref ref, Post post) {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = likePostMutationScope(ref, post);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return post;
  if (controller.reconcile(scope, post.viewerHasLiked)) return post;
  final desired = overlay.optimisticValue;
  if (desired is! bool || desired == post.viewerHasLiked) return post;
  return post.copyWith(
    viewerHasLiked: desired,
    likeCount: desired
        ? post.likeCount + 1
        : (post.likeCount > 0 ? post.likeCount - 1 : 0),
  );
}

Post applyPostInteractionOverlays(Ref ref, Post post) {
  final likedPost = applyLikePostOverlay(ref, post);
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = repostPostMutationScope(ref, likedPost);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return likedPost;
  if (controller.reconcile(scope, likedPost.viewerHasReposted)) {
    return likedPost;
  }
  final desired = overlay.optimisticValue;
  if (desired is! bool || desired == likedPost.viewerHasReposted) {
    return likedPost;
  }
  return likedPost.copyWith(
    viewerHasReposted: desired,
    repostCount: desired
        ? likedPost.repostCount + 1
        : (likedPost.repostCount > 0 ? likedPost.repostCount - 1 : 0),
  );
}

PostCommentSection applyPostCommentSectionInteractionOverlays(
  Ref ref,
  PostCommentSection section,
) {
  return section.copyWith(
    post: applyPostInteractionOverlays(ref, section.post),
    comments: section.comments.copyWith(
      items: [
        for (final item in section.comments.items)
          item.copyWith(
            post: applyPostInteractionOverlays(ref, item.post),
            replies: item.replies.copyWith(
              items: [
                for (final reply in item.replies.items)
                  reply.copyWith(
                    post: applyPostInteractionOverlays(ref, reply.post),
                  ),
              ],
            ),
          ),
      ],
    ),
  );
}
