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
    final ownership = captureActiveAccountOperation(ref);
    final desiredLiked = !post.viewerHasLiked;
    final next = post.copyWith(
      viewerHasLiked: desiredLiked,
      likeCount: post.viewerHasLiked
          ? (post.likeCount > 0 ? post.likeCount - 1 : 0)
          : post.likeCount + 1,
    );
    final endpoint = '/v1/posts/${post.author.did}/${post.rkey}/likes';
    final immutableBody = desiredLiked ? 'POST\n' : 'DELETE\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = likePostMutationScope(ref, post);
    final token = controller.beginOrRetry(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;

    state = const AsyncLoading<Post?>();
    while (true) {
      try {
        final repo = ref.read(postRepositoryProvider);
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
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return;
        controller.markAmbiguous(
          token,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) {
          state = AsyncError<Post?>(
            const PdsMutationUnresolvedException(),
            StackTrace.current,
          );
          return;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !controller.canRetry(
              token,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          return;
        }
        continue;
      } on Object catch (error, stackTrace) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) return;
        controller.markFailed(token);
        state = AsyncError<Post?>(error, stackTrace);
        return;
      }

      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      final reconciliation = PdsSetReconciliation(active: desiredLiked);
      if (!controller.markAccepted(
        token,
        optimisticValue: desiredLiked,
        agrees: (value) => value is bool && reconciliation.agrees(value),
        refresh: () => _invalidateLikeReads(ref, post),
      )) {
        return;
      }
      state = AsyncData<Post?>(next);
      return;
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
    ..invalidate(userProjectsProvider);
}
