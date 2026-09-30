import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

PdsMutationScope likePostMutationScope(Ref ref, Post post) {
  return likePostMutationScopeForLease(
    captureActiveAccountOperation(ref)?.session,
    post,
  );
}

PdsMutationScope likePostMutationScopeForLease(
  AccountSessionLease? activeLease,
  Post post,
) {
  final lease =
      activeLease ??
      AccountSessionLease(
        account: AccountKey(post.author.did.toString()),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'like:${post.uri}');
}

PdsMutationScope repostPostMutationScope(Ref ref, Post post) {
  return repostPostMutationScopeForLease(
    captureActiveAccountOperation(ref)?.session,
    post,
  );
}

PdsMutationScope repostPostMutationScopeForLease(
  AccountSessionLease? activeLease,
  Post post,
) {
  final lease =
      activeLease ??
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
  var visible = section;
  for (final item in section.comments.items) {
    if (applyPostRecordOverlay(ref, item.post) == null) {
      visible = visible.removeDeletedResponse(item.post.uri);
      continue;
    }
    for (final reply in item.replies.items) {
      if (applyPostRecordOverlay(ref, reply.post) == null) {
        visible = visible.removeDeletedResponse(reply.post.uri);
      }
    }
  }
  return visible.copyWith(
    post: applyPostInteractionOverlays(ref, visible.post),
    comments: visible.comments.copyWith(
      items: [
        for (final item in visible.comments.items)
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
