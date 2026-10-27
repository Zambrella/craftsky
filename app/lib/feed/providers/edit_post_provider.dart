import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
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
import 'package:craftsky_app/saved_posts/providers/saved_posts_provider.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/post_search_provider.dart';
import 'package:craftsky_app/search/providers/project_search_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'edit_post_provider.g.dart';

final _log = Logger('EditPostMutation');

@riverpod
class EditPost extends _$EditPost {
  @override
  FutureOr<PostEditResult?> build() => null;

  Future<PostEditResult?> save({
    required PostEditSeed seed,
    required PostEditChanges changes,
    ActiveAccountLease? ownership,
    bool allowVideoBlobRecovery = false,
  }) async {
    final lease = ownership ?? captureActiveAccountOperation(ref);
    final operation = ActiveAccountOperation.fromLease(ref, lease);
    if (!operation.isCurrent ||
        (lease != null && lease.session.account.did != seed.post.author.did)) {
      return null;
    }
    final repository = ref.read(postRepositoryProvider);
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final endpoint = '/v1/posts/${seed.post.author.did}/${seed.post.rkey}';
    final token = controller.beginOrRetry(
      scope: postRecordMutationScope(ref, seed.post),
      endpoint: endpoint,
      immutableBody: jsonEncode({
        'baseCid': seed.baseCid,
        'changes': changes.toMap(),
      }),
      newOperationKey: newPdsMutationOperationKey,
    );
    final retry = PdsMutationRetry(
      ref: ref,
      controller: controller,
      token: token,
      isCurrent: () => operation.isCurrent,
    );
    state = const AsyncLoading();
    while (true) {
      try {
        final result = await repository.edit(
          seed.post.author.did,
          seed.post.rkey,
          changes: changes,
          expectedCid: seed.baseCid,
          operationKey: token.operationKey,
        );
        if (!operation.isCurrent) return null;
        final post = result.post;
        final accepted = post != null && result.changed
            ? controller.markAccepted(
                token,
                optimisticValue: PostEditOverlay(
                  post.copyWith(
                    editedAt: result.editedAt,
                    editTimeSource: result.editTimeSource,
                    recordRevision:
                        post.recordRevision ?? result.recordRevision,
                  ),
                ),
                agrees: (value) =>
                    value is PdsRecordProjection &&
                    PdsPostEditReconciliation(
                      uri: seed.post.uri,
                      acceptedCid: result.acceptedCid,
                      recordRevision: result.recordRevision,
                      observationRevision: result.observationRevision,
                    ).agrees(value),
                refresh: () => _invalidatePostEditReads(ref, seed.post),
              )
            : controller.markAcceptedWithoutOverlay(token);
        if (!accepted) return null;
        _invalidatePostEditReads(ref, seed.post);
        state = AsyncData(result);
        return result;
      } on PdsMutationAmbiguousException catch (error, stack) {
        final decision = await retry.wait(error);
        if (!operation.isCurrent ||
            decision == PdsMutationRetryDecision.obsolete) {
          return null;
        }
        if (decision == PdsMutationRetryDecision.retry) {
          continue;
        }
        state = AsyncError(
          PdsMutationUnresolvedException(
            diagnosticCause: error,
            diagnosticStack: stack,
          ),
          stack,
        );
        return null;
      } on Object catch (error, stack) {
        final mapped = AppErrorMapper.map(
          error,
          source: 'post_mutation',
          fallbackClassification: 'post.edit.failed',
        );
        final diagnostic = ReportContext(
          feature: 'EditPostMutation',
          operation: 'post.edit',
          classification: mapped.sentryClassification,
          safeDiagnostics: {
            ...mapped.safeDiagnostics,
            'failureStage': 'pds_write',
            'attempt': retry.attempt,
          },
          workflow: PublicRecordContext(
            actorDid: seed.post.author.did.value,
            recordUri: seed.post.uri.value,
          ),
        );
        _log.warning(
          DiagnosticMessage('post edit failed', context: diagnostic),
          error,
          stack,
        );
        if (!operation.isCurrent) {
          if (mapped.reportable) {
            unawaited(
              reporter.captureException(
                error,
                stackTrace: stack,
                context: diagnostic,
              ),
            );
          }
          return null;
        }
        controller.markFailed(token);
        if (allowVideoBlobRecovery &&
            error is ApiException &&
            error.details.appViewError == 'video_blob_missing') {
          Error.throwWithStackTrace(error, stack);
        }
        withFailureDiagnosticContext(
          error,
          diagnostic,
          () => state = AsyncError(error, stack),
        );
        return null;
      }
    }
  }

  void reset() => state = const AsyncData(null);

  /// Acceptance is final. A projection refresh performs only an eligible
  /// AppView read and never repeats or rebuilds the PDS command.
  Future<Post?> refreshAccepted({
    required PostEditSeed seed,
    required PostEditResult accepted,
    ActiveAccountLease? ownership,
  }) async {
    if (state.isLoading || state.value != accepted) return null;
    final lease = ownership ?? captureActiveAccountOperation(ref);
    final operation = ActiveAccountOperation.fromLease(ref, lease)
      ..requireCurrent();
    if (lease != null && lease.session.account.did != seed.post.author.did) {
      return null;
    }
    final repository = ref.read(postRepositoryProvider);
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    late PostEditSeed current;
    try {
      current = await repository.fetchEdit(
        seed.post.author.did,
        seed.post.rkey,
      );
    } on Object catch (error, stack) {
      final mapped = AppErrorMapper.map(
        error,
        source: 'post_mutation',
        fallbackClassification: 'post.edit.refresh.failed',
      );
      final diagnostic = ReportContext(
        feature: 'EditPostMutation',
        operation: 'post.edit.refresh',
        classification: mapped.sentryClassification,
        safeDiagnostics: {
          ...mapped.safeDiagnostics,
          'failureStage': 'accepted_read',
        },
        workflow: PublicRecordContext(
          actorDid: seed.post.author.did.value,
          recordUri: seed.post.uri.value,
        ),
      );
      _log.warning(
        DiagnosticMessage(
          'accepted post edit refresh failed',
          context: diagnostic,
        ),
        error,
        stack,
      );
      if (mapped.reportable) {
        unawaited(
          reporter.captureException(
            error,
            stackTrace: stack,
            context: diagnostic,
          ),
        );
      }
      Error.throwWithStackTrace(error, stack);
    }
    operation.requireCurrent();
    if (state.isLoading || state.value != accepted) return null;
    final post = current.post.copyWith(recordRevision: current.recordRevision);
    final agrees = PdsPostEditReconciliation(
      uri: seed.post.uri,
      acceptedCid: accepted.acceptedCid,
      recordRevision: accepted.recordRevision,
      observationRevision: accepted.observationRevision,
    ).agrees(postRecordProjection(post));
    if (!agrees) return null;
    _invalidatePostEditReads(ref, seed.post);
    return post;
  }
}

void _invalidatePostEditReads(Ref ref, Post post) {
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
    ..invalidate(userProjectsProvider)
    ..invalidate(postSearchProvider)
    ..invalidate(projectSearchProvider)
    ..invalidate(hashtagSearchProvider)
    ..invalidate(savedPostsProvider);
}
