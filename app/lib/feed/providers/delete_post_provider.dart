import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
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

part 'delete_post_provider.g.dart';

/// Standalone delete-a-post mutation notifier. The caller supplies the [Post]
/// so the command can retain its exact URI and expected CID.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
@riverpod
class DeletePost extends _$DeletePost {
  @override
  FutureOr<Post?> build() => null;

  Future<void> delete({required Post post}) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    final endpoint = '/v1/posts/${post.author.did}/${post.rkey}';
    final immutableBody = 'DELETE\nIf-Match:${post.cid}\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = postRecordMutationScope(ref, post);
    final token = controller.beginOrRetry(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final retry = PdsMutationRetry(
      ref: ref,
      controller: controller,
      token: token,
      isCurrent: () => accountOperation.isCurrent,
      requireCurrentToken: false,
    );

    state = const AsyncLoading<Post?>();
    late final repository = ref.read(postRepositoryProvider);
    while (true) {
      try {
        await repository.delete(
          post.author.did,
          post.rkey,
          operationKey: operationKey,
          expectedCid: post.cid.toString(),
        );
      } on PdsMutationAmbiguousException catch (error) {
        final decision = await retry.wait(error);
        if (!accountOperation.isCurrent) return;
        switch (decision) {
          case PdsMutationRetryDecision.obsolete:
            return;
          case PdsMutationRetryDecision.exhausted:
            state = AsyncError<Post?>(
              const PdsMutationUnresolvedException(),
              StackTrace.current,
            );
            return;
          case PdsMutationRetryDecision.retry:
            continue;
        }
      } on Object catch (error, stackTrace) {
        if (!accountOperation.isCurrent) return;
        controller.markFailed(token);
        state = AsyncError<Post?>(error, stackTrace);
        return;
      }
      if (!accountOperation.isCurrent) return;

      final reconciliation = PdsAddressedDeleteReconciliation(
        uri: post.uri.toString(),
      );
      if (!controller.markAccepted(
        token,
        optimisticValue: null,
        agrees: (value) => reconciliation.agrees(
          value is PdsRecordProjection ? value : null,
        ),
        refresh: () => _invalidateDeletePostReads(ref, post),
      )) {
        return;
      }

      state = AsyncData<Post?>(post);
      return;
    }
  }

  void reset() => state = const AsyncData(null);
}

void _invalidateDeletePostReads(Ref ref, Post post) {
  if (!ref.mounted) return;
  ref
    ..invalidate(postProvider(post.author.did, post.rkey))
    ..invalidate(postCommentSectionProvider)
    ..invalidate(postQuotesProvider)
    ..invalidate(timelineProvider)
    ..invalidate(userPostsProvider)
    ..invalidate(userCommentsProvider)
    ..invalidate(userRepostsProvider)
    ..invalidate(projectFeedProvider)
    ..invalidate(userProjectsProvider);
}
