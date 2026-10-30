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

Post applyPostInteractionOverlays(
  Ref ref,
  Post post, {
  bool reconcile = true,
}) => applyPostInteractionOverlaysForLease(
  ref.read(pdsRecordOperationControllerProvider),
  captureActiveAccountOperation(ref)?.session,
  post,
  reconcile: reconcile,
);

Post applyPostInteractionOverlaysForLease(
  PdsRecordOperationController controller,
  AccountSessionLease? lease,
  Post post, {
  bool reconcile = true,
}) {
  var result = post;
  final likeScope = likePostMutationScopeForLease(lease, post);
  final likeOverlay = controller.overlayFor(likeScope);
  if (likeOverlay != null &&
      !(reconcile && controller.reconcile(likeScope, post.viewerHasLiked))) {
    final desired = likeOverlay.optimisticValue;
    if (desired is bool && desired != post.viewerHasLiked) {
      result = result.copyWith(
        viewerHasLiked: desired,
        likeCount: desired
            ? post.likeCount + 1
            : (post.likeCount > 0 ? post.likeCount - 1 : 0),
      );
    }
  }
  final repostScope = repostPostMutationScopeForLease(lease, post);
  final repostOverlay = controller.overlayFor(repostScope);
  if (repostOverlay != null &&
      !(reconcile &&
          controller.reconcile(repostScope, post.viewerHasReposted))) {
    final desired = repostOverlay.optimisticValue;
    if (desired is bool && desired != post.viewerHasReposted) {
      result = result.copyWith(
        viewerHasReposted: desired,
        repostCount: desired
            ? post.repostCount + 1
            : (post.repostCount > 0 ? post.repostCount - 1 : 0),
      );
    }
  }
  return result;
}

PostCommentSection applyPostCommentSectionInteractionOverlays(
  Ref ref,
  PostCommentSection section, {
  bool reconcile = true,
}) {
  var visible = section;
  for (final item in section.comments.items) {
    if (applyPostRecordOverlay(ref, item.post, reconcile: reconcile) == null) {
      visible = visible.removeDeletedResponse(item.post.uri);
      continue;
    }
    for (final reply in item.replies.items) {
      if (applyPostRecordOverlay(ref, reply.post, reconcile: reconcile) ==
          null) {
        visible = visible.removeDeletedResponse(reply.post.uri);
      }
    }
  }
  return visible.copyWith(
    post: applyPostInteractionOverlays(
      ref,
      applyPostRecordOverlay(ref, visible.post, reconcile: reconcile) ??
          visible.post,
    ),
    comments: visible.comments.copyWith(
      items: [
        for (final item in visible.comments.items)
          item.copyWith(
            post: applyPostInteractionOverlays(
              ref,
              applyPostRecordOverlay(ref, item.post, reconcile: reconcile) ??
                  item.post,
            ),
            replies: item.replies.copyWith(
              items: [
                for (final reply in item.replies.items)
                  reply.copyWith(
                    post: applyPostInteractionOverlays(
                      ref,
                      applyPostRecordOverlay(
                            ref,
                            reply.post,
                            reconcile: reconcile,
                          ) ??
                          reply.post,
                    ),
                  ),
              ],
            ),
          ),
      ],
    ),
  );
}
