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
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'toggle_repost_post_provider.g.dart';

@riverpod
class ToggleRepostPost extends _$ToggleRepostPost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> toggle({required Post post}) async {
    final ownership = captureActiveAccountOperation(ref);
    final desiredReposted = !post.viewerHasReposted;
    final next = post.copyWith(
      viewerHasReposted: desiredReposted,
      repostCount: post.viewerHasReposted
          ? (post.repostCount > 0 ? post.repostCount - 1 : 0)
          : post.repostCount + 1,
    );
    final endpoint = '/v1/posts/${post.author.did}/${post.rkey}/reposts';
    final immutableBody = desiredReposted ? 'POST\n' : 'DELETE\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = repostPostMutationScope(ref, post);
    final token = controller.beginOrRetry(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;

    state = AsyncData(next);
    try {
      await runPdsMutation<void>(
        ref: ref,
        controller: controller,
        token: token,
        isCurrent: () => isActiveAccountOperationCurrent(ref, ownership),
        send: () async {
          final repo = ref.read(postRepositoryProvider);
          if (desiredReposted) {
            await repo.repost(
              post.author.did,
              post.rkey,
              operationKey: operationKey,
            );
          } else {
            await repo.unrepost(
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
      state = AsyncError<Post?>(error, stackTrace);
      return;
    } on Object catch (error, stackTrace) {
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      controller.markFailed(token);
      state = const AsyncData(null);
      state = AsyncError<Post?>(error, stackTrace);
      return;
    }

    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    final reconciliation = PdsSetReconciliation(active: desiredReposted);
    if (!controller.markAccepted(
      token,
      optimisticValue: desiredReposted,
      agrees: (value) => value is bool && reconciliation.agrees(value),
      refresh: () => _invalidateRepostReads(ref, post),
    )) {
      return;
    }
    state = AsyncData<Post?>(next);
  }

  void reset() => state = const AsyncData(null);
}

void _invalidateRepostReads(Ref ref, Post post) {
  if (!ref.mounted) return;
  ref
    ..invalidate(postProvider(post.author.did, post.rkey))
    ..invalidate(postCommentSectionProvider)
    ..invalidate(postQuotesProvider)
    ..invalidate(timelineProvider)
    ..invalidate(userPostsProvider)
    ..invalidate(userCommentsProvider)
    ..invalidate(userRepostsProvider)
    ..invalidate(userProjectsProvider);
}
