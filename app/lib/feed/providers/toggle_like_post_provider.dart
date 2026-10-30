import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/like_post_overlay.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/feed/providers/user_reposts_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_like_post_provider.g.dart';

@riverpod
class ToggleLikePost extends _$ToggleLikePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> toggle({required Post post}) async {
    // A submitted mutation owns its completion even if its UI listener leaves.
    // Explicit invalidation/account changes still end its authority.
    final lifetime = ref.keepAlive();
    try {
      final operation = ActiveAccountOperation.capture(ref);
      final controller = ref.read(pdsRecordOperationControllerProvider);
      final scope = likePostMutationScope(ref, post);
      final pending = controller.operationFor(scope);
      final pendingValue = pending?.optimisticValue;
      final current = applyPostInteractionOverlays(ref, post, reconcile: false);
      final desiredLiked =
          pending?.status == PdsMutationStatus.ambiguous && pendingValue is bool
          ? pendingValue
          : !current.viewerHasLiked;
      final next = current.copyWith(
        viewerHasLiked: desiredLiked,
        likeCount: current.viewerHasLiked == desiredLiked
            ? current.likeCount
            : desiredLiked
            ? current.likeCount + 1
            : (current.likeCount > 0 ? current.likeCount - 1 : 0),
      );
      final endpoint = '/v1/posts/${post.author.did}/${post.rkey}/likes';
      final immutableBody = desiredLiked ? 'POST\n' : 'DELETE\n';
      final token = controller.beginOrRetry(
        scope: scope,
        endpoint: endpoint,
        immutableBody: immutableBody,
        newOperationKey: newPdsMutationOperationKey,
        optimisticValue: desiredLiked,
      );
      final operationKey = token.operationKey;
      late final repo = ref.read(postRepositoryProvider);

      state = AsyncData(next);
      try {
        await runPdsMutation<void>(
          ref: ref,
          controller: controller,
          token: token,
          isCurrent: () => operation.isCurrent,
          send: () async {
            if (desiredLiked) {
              await repo.like(
                post.author.did,
                post.rkey,
                operationKey: operationKey,
              );
            } else {
              await repo.unlike(
                post.author.did,
                post.rkey,
                operationKey: operationKey,
              );
            }
          },
        );
      } on PdsMutationObsoleteException {
        return;
      } on PdsMutationUnresolvedException catch (error, stackTrace) {
        operation.publish(() => state = AsyncError<Post?>(error, stackTrace));
        return;
      } on Object catch (error, stackTrace) {
        if (!operation.isCurrent) return;
        controller.markFailed(token);
        state = const AsyncData(null);
        state = AsyncError<Post?>(error, stackTrace);
        return;
      }

      if (!operation.isCurrent) return;
      final reconciliation = PdsSetReconciliation(active: desiredLiked);
      if (!controller.markAccepted(
        token,
        optimisticValue: desiredLiked,
        retainUntilExpiry: true,
        agrees: (value) => value is bool && reconciliation.agrees(value),
        refresh: () => _invalidateLikeReads(ref, post),
      )) {
        return;
      }
      state = AsyncData<Post?>(next);
    } finally {
      lifetime.close();
    }
  }

  void reset() => state = const AsyncData(null);
}

void _invalidateLikeReads(Ref ref, Post post) {
  if (!ref.mounted) return;
  ref
    ..invalidate(postProvider(post.author.did, post.rkey))
    ..invalidate(postCommentSectionProvider)
    ..invalidate(postQuotesProvider)
    ..invalidate(timelineProvider)
    ..invalidate(userPostsProvider)
    ..invalidate(userCommentsProvider)
    ..invalidate(userRepostsProvider)
    ..invalidate(userProjectsProvider)
    ..invalidate(projectFeedProvider);
}
