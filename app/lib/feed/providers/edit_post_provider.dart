import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/composer/post_edit_presentation.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/post_edit_refresh.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
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
    final submittedPost = postEditPresentation(seed, changes);
    final refresh = ref.read(postEditRefreshProvider);
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
        final post = (result.post ?? submittedPost).copyWith(
          cid: result.acceptedCid,
          editedAt: result.editedAt,
          recordRevision: result.post?.recordRevision ?? result.recordRevision,
        );
        final previous = controller.overlayFor(token.scope);
        final publicationLists = switch (previous?.optimisticValue) {
          Post() => {...previous!.publicationLists},
          PostEditOverlay(:final publicationLists) => {...publicationLists},
          _ => <Object>{},
        };
        final overlay = PostEditOverlay(
          post,
          revisionBound: result.recordRevision ?? result.observationRevision,
          publicationLists: publicationLists,
        );
        final accepted = result.changed
            ? controller.markAccepted(
                token,
                optimisticValue: overlay,
                agrees: overlay.agreesWith,
                refresh: () => refresh(seed.post, lease),
                retainVersion: true,
              )
            : controller.markAcceptedWithoutOverlay(token);
        if (!accepted) return null;
        notifyPostRecordOverlays(ref);

        final presented = PostEditResult(
          acceptedCid: result.acceptedCid,
          changed: result.changed,
          post: post,
          editedAt: result.editedAt,
          recordRevision: result.recordRevision,
          observationCid: result.observationCid,
          observationRevision: result.observationRevision,
        );
        state = AsyncData(presented);
        return presented;
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
}
