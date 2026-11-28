import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/create_post_external.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/instagram_import_publication.dart';
import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/profile/providers/profile_post_stats_overlay.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'create_post_provider.g.dart';

typedef InstagramPublicationCheckpoint = Future<void> Function(
  InstagramImportPublication? publication,
);

final _diagnosticLog = Logger('CreatePostMutation');

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.
@riverpod
class CreatePost extends _$CreatePost {
  @override
  FutureOr<Post?> build() => null;

  Future<Post?> create({
    required String text,
    required List<String> langs,
    required bool sponsored,
    PostReply? reply,
    PostRef? quote,
    String? quoteImageCid,
    Project? project,
    List<CreatePostImage>? images,
    CreatePostExternal? external,
    CreatePostVideo? video,
    List<Map<String, dynamic>>? facets,
    InstagramPostImportSource? instagramImport,
    ActiveAccountLease? ownership,
    bool allowVideoBlobRecovery = false,
    InstagramPublicationCheckpoint? checkpointInstagramPublication,
  }) async {
    final frozenLangs = List<String>.unmodifiable(langs);
    final frozenImages = images == null
        ? null
        : List<CreatePostImage>.unmodifiable(images);
    final frozenFacets = facets == null
        ? null
        : (jsonDecode(jsonEncode(facets)) as List)
              .map((item) => Map<String, dynamic>.from(item as Map))
              .toList(growable: false);
    final request = PostApiClient.createRequestBody(
      text: text,
      langs: frozenLangs,
      sponsored: sponsored,
      reply: reply,
      quote: quote,
      quoteImageCid: quoteImageCid,
      project: project,
      images: frozenImages,
      external: external,
      video: video,
      facets: frozenFacets,
      instagramImport: instagramImport,
    );
    InstagramImportPublication? frozenImport;
    return await _publish(
      ownership: ownership,
      instagramImport: instagramImport,
      immutableBody: jsonEncode(request),
      newOperationKey: newPdsMutationOperationKey,
      send: (repository, operationKey) {
        assert(
          project == null || reply == null,
          'Project posts cannot be replies',
        );
        assert(quote == null || reply == null, 'Quote posts cannot be replies');
        assert(
          quote == null || project == null,
          'Project posts cannot be quote posts',
        );
        if (frozenImport case final pending?) {
          return repository.publishPreparedInstagramImport(pending);
        }
        return repository.create(
          operationKey: operationKey,
          text: text,
          langs: frozenLangs,
          sponsored: sponsored,
          reply: reply,
          quote: quote,
          quoteImageCid: quoteImageCid,
          project: project,
          images: frozenImages,
          external: external,
          video: video,
          facets: frozenFacets,
          instagramImport: instagramImport,
        );
      },
      beforeSend: checkpointInstagramPublication == null
          ? null
          : (key) async {
              final pending = InstagramImportPublication(
                operationKey: key,
                request: request,
              );
              await checkpointInstagramPublication(pending);
              frozenImport = pending;
            },
      clearCheckpoint: checkpointInstagramPublication == null
          ? null
          : () => checkpointInstagramPublication(null),
      reply: reply,
      quote: quote,
      project: project,
      langs: frozenLangs,
      allowVideoBlobRecovery: allowVideoBlobRecovery,
    );
  }

  Future<Post?> retryInstagramPublication(
    InstagramImportPublication pending, {
    required ActiveAccountLease ownership,
    required Future<void> Function() clearCheckpoint,
  }) => _publish(
    ownership: ownership,
    instagramImport: pending.source,
    immutableBody: pending.encodedRequest,
    newOperationKey: () => pending.operationKey,
    send: (repository, _) => repository.publishPreparedInstagramImport(pending),
    clearCheckpoint: clearCheckpoint,
  );

  Future<Post?> _publish({
    required ActiveAccountLease? ownership,
    required InstagramPostImportSource? instagramImport,
    required String immutableBody,
    required String Function() newOperationKey,
    required Future<Post> Function(PostRepository, String) send,
    Future<void> Function(String operationKey)? beforeSend,
    Future<void> Function()? clearCheckpoint,
    PostReply? reply,
    PostRef? quote,
    Project? project,
    List<String>? langs,
    bool allowVideoBlobRecovery = false,
  }) async {
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    final operationOwnership = ownership ?? captureActiveAccountOperation(ref);
    final accountOperation = ActiveAccountOperation.fromLease(
      ref,
      operationOwnership,
    );
    if (!accountOperation.isCurrent) return null;
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final token = controller.beginOrRetry(
      scope: postCreateOperationScope(
        operationOwnership,
        instagramImport: instagramImport,
      ),
      endpoint: '/v1/posts',
      immutableBody: immutableBody,
      newOperationKey: newOperationKey,
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
    var needsCheckpoint = beforeSend != null;
    while (true) {
      late Post created;
      try {
        if (needsCheckpoint) {
          await beforeSend!(operationKey);
          needsCheckpoint = false;
          if (!accountOperation.isCurrent) return null;
        }
        created = await send(repository, operationKey);
      } on PdsMutationAmbiguousException catch (error, stackTrace) {
        final decision = await retry.wait(error);
        if (!accountOperation.isCurrent) return null;
        switch (decision) {
          case PdsMutationRetryDecision.obsolete:
            return null;
          case PdsMutationRetryDecision.exhausted:
            state = AsyncError<Post?>(
              PdsMutationUnresolvedException(
                diagnosticCause: error,
                diagnosticStack: stackTrace,
              ),
              stackTrace,
            );
            return null;
          case PdsMutationRetryDecision.retry:
            continue;
        }
      } on Object catch (error, stackTrace) {
        final current = accountOperation.isCurrent;
        final mapped = AppErrorMapper.map(
          error,
          source: 'post_mutation',
          fallbackClassification: 'post.create.failed',
        );
        final diagnostic = ReportContext(
          feature: 'CreatePostMutation',
          operation: 'post.create',
          classification: mapped.sentryClassification,
          safeDiagnostics: {
            ...mapped.safeDiagnostics,
            'failureStage': 'pds_write',
            'attempt': retry.attempt,
          },
          workflow: operationOwnership == null
              ? null
              : instagramImport == null
              ? PublicRecordContext(
                  actorDid: operationOwnership.session.account.did.value,
                )
              : PrivateOperationalFailureContext(
                  accountDid: operationOwnership.session.account.did.value,
                ),
        );
        final message = DiagnosticMessage(
          'post mutation failed',
          context: diagnostic,
        );
        if (current) {
          _diagnosticLog.warning(message, error, stackTrace);
        } else {
          _diagnosticLog.severe(message, error, stackTrace);
        }
        if (!current) {
          // No provider state is published for this consumed late failure.
          if (mapped.reportable) {
            unawaited(
              reporter.captureException(
                error,
                stackTrace: stackTrace,
                context: diagnostic,
              ),
            );
          }
          return null;
        }
        controller.markFailed(token);
        if (!needsCheckpoint &&
            clearCheckpoint != null &&
            _canEditAfterRejection(error)) {
          try {
            await clearCheckpoint();
          } on Object catch (checkpointError, checkpointStack) {
            if (accountOperation.isCurrent) {
              state = AsyncError<Post?>(checkpointError, checkpointStack);
            }
            return null;
          }
          if (!accountOperation.isCurrent) return null;
        }
        if (allowVideoBlobRecovery &&
            error is ApiException &&
            error.details.appViewError == 'video_blob_missing') {
          Error.throwWithStackTrace(error, stackTrace);
        }
        withFailureDiagnosticContext(
          error,
          diagnostic,
          () => state = AsyncError<Post?>(error, stackTrace),
        );
        return null;
      }

      var post = created;
      if (reply != null && post.reply == null) {
        post = post.copyWith(reply: reply);
      }
      if (quote != null && post.quote == null) {
        post = post.copyWith(quote: quote);
      }
      if (project != null && post.project == null) {
        post = post.copyWith(project: project);
      }
      if (post.langs.isEmpty && langs != null) {
        post = post.copyWith(langs: langs);
      }
      if (!accountOperation.isCurrent) {
        return null;
      }

      final accepted = reply == null && instagramImport == null
          ? controller.markAccepted(
              token,
              overlayScope: postRecordMutationScope(ref, post),
              optimisticValue: post,
              agrees: (value) =>
                  value is PdsRecordProjection &&
                  PdsAppendReconciliation(
                    selectedUri: post.uri.toString(),
                    acceptedCid: post.cid.toString(),
                    controlledContent: postRecordProjection(post).content,
                  ).agrees(value),
              refresh: () => _invalidatePostMutationReads(ref, post),
            )
          : controller.markAcceptedWithoutOverlay(token);
      if (!accepted) {
        return null;
      }
      if (instagramImport == null) {
        publishProfilePostStats(ref, post);
      } else {
        // AppView indexes historical imports in profile and search order. The
        // ordinary publication overlay would show them as new home feed posts.
        _invalidatePostMutationReads(ref, post);
      }
      state = AsyncData<Post?>(post);
      return post;
    }
  }

  bool _canEditAfterRejection(Object error) => switch (error) {
    ApiBadRequest(
      code: 'instagram_verification_required' || 'idempotency_conflict',
    ) =>
      false,
    ApiBadRequest(details: ApiFailureDetails(statusCode: 403 || 429)) => false,
    ApiBadRequest() => true,
    ApiServerError(
      details: ApiFailureDetails(
        appViewError: 'video_blob_missing' || 'pds_write_rejected',
      ),
    ) =>
      true,
    _ => false,
  };

  /// Resets the notifier to its idle state. Call after consuming a
  /// success/failure transition so a re-entry doesn't see prior result.
  void reset() => state = const AsyncData(null);
}

void _invalidatePostMutationReads(Ref ref, Post post) {
  if (!ref.mounted) return;
  ref
    ..invalidate(
      postProvider(
        post.author.did,
        post.rkey,
        audience: ref.read(browsingContextProvider),
      ),
    )
    ..invalidate(postCommentSectionProvider)
    ..invalidate(postQuotesProvider)
    ..invalidate(timelineProvider)
    ..invalidate(extendedNetworkProvider)
    ..invalidate(userPostsProvider)
    ..invalidate(userCommentsProvider)
    ..invalidate(userProjectsProvider);
}
